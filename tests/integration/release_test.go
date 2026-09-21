package integration

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/distribution"
)

// TestReleaseInstallUpdate qualifies the offline release path end to end:
// a built bundle releases to one zip plus integrity and inspection records,
// installs into a versioned root, runs offline through the selected link
// with no developer tools on PATH, and updates atomically while the old
// version keeps serving. Every refusal below names its cause; nothing
// downloads, uploads, signs, or bypasses Gatekeeper.
func TestReleaseInstallUpdate(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE to the pinned local archive to run offline release integration")
	}
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		t.Fatal("admitted integration target is macOS arm64")
	}
	source, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	sourceBefore := treeHashes(t, source, []string{"compiler", "distribution", "runtime", "tools"})
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	build := func(version string) string {
		t.Helper()
		bundle, err := distribution.Build(ctx, source, t.TempDir(), archive, version)
		if err != nil {
			t.Fatal(err)
		}
		return bundle
	}
	bundleA := build("rel-a")
	bundleB := build("rel-b")
	target := distribution.PinnedTarget()
	releases := t.TempDir()
	releaseA, err := distribution.Release(ctx, bundleA, releases)
	if err != nil {
		t.Fatal(err)
	}
	base := "can-rel-a-" + target.TargetID
	if filepath.Base(releaseA.Archive) != base+".zip" {
		t.Fatalf("wrong release name: %s", releaseA.Archive)
	}
	digest, err := distribution.ParseSHA256Record(releaseA.SHA256, base+".zip")
	if err != nil {
		t.Fatal(err)
	}
	zipBytes, err := os.ReadFile(releaseA.Archive)
	if err != nil || distribution.Hash(zipBytes) != digest {
		t.Fatal("sha record does not match the release zip")
	}
	transcript, err := os.ReadFile(releaseA.Inspection)
	if err != nil {
		t.Fatal(err)
	}
	var inspection struct {
		Kind          string `json:"kind"`
		TargetID      string `json:"targetId"`
		RuntimeSHA256 string `json:"runtimeSHA256"`
		Display       string `json:"display"`
	}
	if err := json.Unmarshal(transcript, &inspection); err != nil {
		t.Fatal(err)
	}
	if inspection.Kind != "can.release-inspection" || inspection.TargetID != target.TargetID || inspection.RuntimeSHA256 != target.Runtime.SHA256 {
		t.Fatalf("inspection does not bind the pin: %s", transcript)
	}
	if !strings.Contains(inspection.Display, "Identifier=") || !strings.Contains(inspection.Display, "Mach-O") {
		t.Fatalf("inspection missing signature facts: %q", inspection.Display)
	}
	again, err := distribution.Release(ctx, bundleA, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	redigest, err := distribution.ParseSHA256Record(again.SHA256, base+".zip")
	if err != nil || redigest != digest {
		t.Fatal("release bytes not deterministic")
	}
	releaseB, err := distribution.Release(ctx, bundleB, releases)
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(t.TempDir(), "install-root")
	installedA, err := distribution.Install(ctx, releaseA.Archive, releaseA.SHA256, root)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := distribution.Selection(root)
	if err != nil || selected != installedA {
		t.Fatalf("selection %q points away from install: %v", selected, err)
	}
	// The installed tree runs offline through the selected link with no
	// developer tools on PATH, from an unrelated directory.
	parent := t.TempDir()
	home := filepath.Join(parent, "home")
	cwd := filepath.Join(parent, "unrelated")
	for _, p := range []string{home, cwd} {
		if err := os.Mkdir(p, 0700); err != nil {
			t.Fatal(err)
		}
	}
	run := func(args ...string) []byte {
		t.Helper()
		cmd := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", append([]string{"-p", "(version 1)(allow default)(deny network*)", filepath.Join(root, "current/bin/canlc")}, args...)...)
		cmd.Dir = cwd
		cmd.Env = []string{"PATH=/nonexistent", "HOME=" + home}
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("installed run %v: %v\n%s", args, err, out)
		}
		return out
	}
	var check struct {
		Bun        string `json:"bun"`
		Executable string `json:"executable"`
	}
	if err := json.Unmarshal(run("runtime-check"), &check); err != nil {
		t.Fatal(err)
	}
	wantExecutable, _ := filepath.EvalSymlinks(filepath.Join(installedA, "runtime/bun"))
	if check.Bun != "1.4.2" || check.Executable != wantExecutable {
		t.Fatalf("installed runtime resolves wrong: %+v", check)
	}
	refuse := func(name, zipPath, shaPath, want string) {
		t.Helper()
		if _, err := distribution.Install(ctx, zipPath, shaPath, filepath.Join(t.TempDir(), "refuse-root")); err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("%s: expected %q, got %v", name, want, err)
		}
	}
	// A flipped byte fails the detached record first.
	flipped := bytes.Clone(zipBytes)
	flipped[len(flipped)/2] ^= 0xff
	flippedPath := filepath.Join(releases, "flipped.zip")
	if err := os.WriteFile(flippedPath, flipped, 0644); err != nil {
		t.Fatal(err)
	}
	flippedSHA := filepath.Join(releases, "flipped.zip.sha256")
	if err := os.WriteFile(flippedSHA, []byte(distribution.Hash(flipped)+"  flipped.zip\n"), 0644); err != nil {
		t.Fatal(err)
	}
	refuse("record mismatch", flippedPath, releaseA.SHA256, "malformed record")
	mismatchSHA := filepath.Join(releases, "mismatch.zip.sha256")
	if err := os.WriteFile(mismatchSHA, []byte(digest+"  flipped.zip\n"), 0644); err != nil {
		t.Fatal(err)
	}
	refuse("archive mismatch", flippedPath, mismatchSHA, "SHA-256 mismatch")
	refuse("forged record", flippedPath, flippedSHA, "install release")
	craft := func(name string, files map[string][]byte, links map[string]string) (string, string) {
		t.Helper()
		path := filepath.Join(releases, name+".zip")
		file, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		writer := zip.NewWriter(file)
		for entry, data := range files {
			header := &zip.FileHeader{Name: entry, Method: zip.Store}
			header.SetMode(0644)
			out, err := writer.CreateHeader(header)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := out.Write(data); err != nil {
				t.Fatal(err)
			}
		}
		for entry, target := range links {
			header := &zip.FileHeader{Name: entry, Method: zip.Store}
			header.SetMode(os.ModeSymlink | 0777)
			out, err := writer.CreateHeader(header)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := out.Write([]byte(target)); err != nil {
				t.Fatal(err)
			}
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		sha := filepath.Join(releases, name+".zip.sha256")
		if err := os.WriteFile(sha, []byte(distribution.Hash(data)+"  "+name+".zip\n"), 0644); err != nil {
			t.Fatal(err)
		}
		return path, sha
	}
	symlinkZip, symlinkSHA := craft("evil-link", map[string][]byte{"v/manifest.json": []byte("{}")}, map[string]string{"v/link": "/etc/passwd"})
	refuse("symlink entry", symlinkZip, symlinkSHA, "non-regular archive entry")
	escapeZip, escapeSHA := craft("evil-escape", map[string][]byte{"v/../../escape": []byte("x")}, nil)
	refuse("escape entry", escapeZip, escapeSHA, "invalid archive path")
	multiZip, multiSHA := craft("evil-multi", map[string][]byte{"v1/a": []byte("x"), "v2/a": []byte("x")}, nil)
	refuse("two versions", multiZip, multiSHA, "more than one version")
	// A re-zipped tree with an edited manifest fails bundle verification
	// even with a fresh matching record.
	reZip := func(mutate func(entries map[string][]byte)) (string, string) {
		t.Helper()
		reader, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
		if err != nil {
			t.Fatal(err)
		}
		entries := map[string][]byte{}
		for _, file := range reader.File {
			r, err := file.Open()
			if err != nil {
				t.Fatal(err)
			}
			data, err := io.ReadAll(r)
			r.Close()
			if err != nil {
				t.Fatal(err)
			}
			entries[file.Name] = data
		}
		mutate(entries)
		path := filepath.Join(releases, "rezipped.zip")
		file, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		writer := zip.NewWriter(file)
		for entry, data := range entries {
			header := &zip.FileHeader{Name: entry, Method: zip.Store}
			header.SetMode(0644)
			if strings.HasSuffix(entry, "runtime/bun") || strings.HasSuffix(entry, "bin/canlc") {
				header.SetMode(0755)
			}
			out, err := writer.CreateHeader(header)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := out.Write(data); err != nil {
				t.Fatal(err)
			}
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		sha := filepath.Join(releases, "rezipped.zip.sha256")
		if err := os.WriteFile(sha, []byte(distribution.Hash(data)+"  rezipped.zip\n"), 0644); err != nil {
			t.Fatal(err)
		}
		return path, sha
	}
	editedZip, editedSHA := reZip(func(entries map[string][]byte) {
		for name, data := range entries {
			if strings.HasSuffix(name, "manifest.json") {
				entries[name] = bytes.Replace(data, []byte(target.TargetID), []byte("bun-9.9.9-darwin-arm64-v9"), 1)
			}
		}
	})
	refuse("edited manifest", editedZip, editedSHA, "unsupported distribution identity")
	// Update selects the new version while the old one keeps serving: hold
	// its runtime open, then prove the old bytes and the old binary intact.
	held, err := os.Open(filepath.Join(installedA, "runtime/bun"))
	if err != nil {
		t.Fatal(err)
	}
	defer held.Close()
	oldBytes, err := os.ReadFile(filepath.Join(installedA, "runtime/bun"))
	if err != nil {
		t.Fatal(err)
	}
	previous, current, err := distribution.Update(ctx, releaseB.Archive, releaseB.SHA256, root)
	if err != nil {
		t.Fatal(err)
	}
	if previous != installedA || !strings.HasSuffix(current, "can-rel-b-"+target.TargetID) {
		t.Fatalf("update moved wrong: %q -> %q", previous, current)
	}
	if _, err := held.Stat(); err != nil {
		t.Fatal("held runtime descriptor broke across update")
	}
	kept, err := os.ReadFile(filepath.Join(previous, "runtime/bun"))
	if err != nil || distribution.Hash(kept) != distribution.Hash(oldBytes) {
		t.Fatal("old version changed under update")
	}
	oldRun := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-p", "(version 1)(allow default)(deny network*)", filepath.Join(previous, "bin/canlc"), "runtime-check")
	oldRun.Dir = cwd
	oldRun.Env = []string{"PATH=/nonexistent", "HOME=" + home}
	if out, err := oldRun.CombinedOutput(); err != nil {
		t.Fatalf("old binary stopped running: %v\n%s", err, out)
	}
	var now struct {
		Executable string `json:"executable"`
	}
	if err := json.Unmarshal(run("runtime-check"), &now); err != nil {
		t.Fatal(err)
	}
	wantCurrent, _ := filepath.EvalSymlinks(filepath.Join(current, "runtime/bun"))
	if now.Executable != wantCurrent {
		t.Fatalf("selection did not move to the new version: %+v", now)
	}
	// A failed update preserves the previous selection; racing installs of
	// the selected version land exactly once.
	if _, _, err := distribution.Update(ctx, flippedPath, flippedSHA, root); err == nil {
		t.Fatal("tampered update succeeded")
	}
	if selected, err := distribution.Selection(root); err != nil || selected != current {
		t.Fatalf("failed update moved selection to %q: %v", selected, err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, err := distribution.Update(ctx, releaseB.Archive, releaseB.SHA256, root)
			if err != nil && !strings.Contains(err.Error(), "already exists") {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("racing update failed: %v", err)
	}
	if selected, err := distribution.Selection(root); err != nil || selected != current {
		t.Fatalf("racing updates moved selection to %q: %v", selected, err)
	}
	foreign := filepath.Join(root, "operator-note.txt")
	if err := os.WriteFile(foreign, []byte("operator data"), 0600); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(root, ".can-stage-interrupted")
	if err := os.Mkdir(stale, 0755); err != nil {
		t.Fatal(err)
	}
	if err := distribution.PruneStaging(root); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(foreign); err != nil {
		t.Fatal("prune touched a foreign file")
	}
	if _, err := os.Lstat(stale); !os.IsNotExist(err) {
		t.Fatal("prune left owned staging")
	}
	if selected, err := distribution.Selection(root); err != nil || selected != current {
		t.Fatalf("prune moved selection to %q: %v", selected, err)
	}
	if !reflect.DeepEqual(sourceBefore, treeHashes(t, source, []string{"compiler", "distribution", "runtime", "tools"})) {
		t.Fatal("release flow wrote to the source tree")
	}
}

func TestInstallRootRefusals(t *testing.T) {
	if os.Getenv("CAN_BUN_ARCHIVE") == "" {
		t.Skip("set CAN_BUN_ARCHIVE for root refusal checks")
	}
	ctx := context.Background()
	linkRoot := filepath.Join(t.TempDir(), "link-root")
	if err := os.Symlink(t.TempDir(), linkRoot); err != nil {
		t.Fatal(err)
	}
	// The root check fires before any archive is read.
	_, _, err := distribution.Update(ctx, filepath.Join(t.TempDir(), "absent.zip"), filepath.Join(t.TempDir(), "absent.sha256"), linkRoot)
	if err == nil || !strings.Contains(err.Error(), "must not be a symlink") {
		t.Fatalf("wrong root refusal: %v", err)
	}
	if selected, err := distribution.Selection(filepath.Join(t.TempDir(), "empty-root")); err != nil || selected != "" {
		t.Fatalf("empty root selects %q: %v", selected, err)
	}
}
