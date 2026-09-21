package distribution

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallRoundTrip(t *testing.T) {
	ctx := context.Background()
	runtimeBytes := archiveRuntime(t)
	bundle := syntheticBundle(t, t.TempDir(), "inst-1", runtimeBytes)
	out := t.TempDir()
	artifacts, err := Release(ctx, bundle, out)
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(t.TempDir(), "fresh-root")
	installed, err := Install(ctx, artifacts.Archive, artifacts.SHA256, root)
	if err != nil {
		t.Fatal(err)
	}
	name := "can-inst-1-" + PinnedTarget().TargetID
	if installed != filepath.Join(root, "versions", name) {
		t.Fatalf("wrong install path: %s", installed)
	}
	selected, err := Selection(root)
	if err != nil || selected != installed {
		t.Fatalf("selection %q points away from install: %v", selected, err)
	}
	if manifest, err := VerifyBundle(installed); err != nil || manifest.Version != "inst-1" {
		t.Fatalf("installed tree does not verify: %v", err)
	}
	if _, err := Install(ctx, artifacts.Archive, artifacts.SHA256, root); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("reinstalled an existing version: %v", err)
	}
	if entries, err := os.ReadDir(root); err != nil || len(entries) != 3 {
		t.Fatalf("root holds unexpected entries: %v %v", entries, err)
	}
}

func TestInstallRefusals(t *testing.T) {
	ctx := context.Background()
	runtimeBytes := archiveRuntime(t)
	bundle := syntheticBundle(t, t.TempDir(), "inst-bad", runtimeBytes)
	out := t.TempDir()
	artifacts, err := Release(ctx, bundle, out)
	if err != nil {
		t.Fatal(err)
	}
	// A tampered archive fails the detached record before anything stages.
	tampered := filepath.Join(out, "tampered.zip")
	data, err := os.ReadFile(artifacts.Archive)
	if err != nil {
		t.Fatal(err)
	}
	data[len(data)/2] ^= 0xff
	if err := os.WriteFile(tampered, data, 0644); err != nil {
		t.Fatal(err)
	}
	fakeSHA := filepath.Join(out, "tampered.zip.sha256")
	if err := os.WriteFile(fakeSHA, []byte(Hash(data)+"  tampered.zip\n"), 0644); err != nil {
		t.Fatal(err)
	}
	wrongSHA := filepath.Join(out, "wrong.zip.sha256")
	original, err := ParseSHA256Record(artifacts.SHA256, filepath.Base(artifacts.Archive))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(wrongSHA, []byte(original+"  tampered.zip\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(ctx, tampered, wrongSHA, t.TempDir()); err == nil || !strings.Contains(err.Error(), "SHA-256 mismatch") {
		t.Fatalf("installed despite record mismatch: %v", err)
	}
	// The same bytes with a matching forged record still fail: the staged
	// tree no longer matches its manifest.
	if _, err := Install(ctx, tampered, fakeSHA, t.TempDir()); err == nil {
		t.Fatal("installed a tampered tree with a forged record")
	}
	if _, err := Install(ctx, artifacts.Archive, filepath.Join(out, "absent.sha256"), t.TempDir()); err == nil {
		t.Fatal("installed without a sha record")
	}
	linkRoot := filepath.Join(t.TempDir(), "link-root")
	if err := os.Symlink(t.TempDir(), linkRoot); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(ctx, artifacts.Archive, artifacts.SHA256, linkRoot); err == nil || !strings.Contains(err.Error(), "must not be a symlink") {
		t.Fatalf("installed into a symlinked root: %v", err)
	}
	fileRoot := filepath.Join(t.TempDir(), "file-root")
	if err := os.WriteFile(fileRoot, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(ctx, artifacts.Archive, artifacts.SHA256, fileRoot); err == nil || !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("installed into a file root: %v", err)
	}
	// Failed installs leave no staging residue and select nothing.
	root := filepath.Join(t.TempDir(), "failed-root")
	if _, err := Install(ctx, tampered, fakeSHA, root); err == nil {
		t.Fatal("tampered install unexpectedly succeeded")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".can-stage-") {
			t.Fatalf("failed install left staging: %s", entry.Name())
		}
	}
	if selected, err := Selection(root); err != nil || selected != "" {
		t.Fatalf("failed install selected %q: %v", selected, err)
	}
}

func TestExtractReleaseRefusals(t *testing.T) {
	craft := func(t *testing.T, entries map[string]os.FileMode, links map[string]string) string {
		t.Helper()
		path := filepath.Join(t.TempDir(), "craft.zip")
		file, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		writer := zip.NewWriter(file)
		for name, mode := range entries {
			header := &zip.FileHeader{Name: name, Method: zip.Store}
			header.SetMode(mode)
			entry, err := writer.CreateHeader(header)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := entry.Write([]byte("x")); err != nil {
				t.Fatal(err)
			}
		}
		for name, target := range links {
			header := &zip.FileHeader{Name: name, Method: zip.Store}
			header.SetMode(os.ModeSymlink | 0777)
			entry, err := writer.CreateHeader(header)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := entry.Write([]byte(target)); err != nil {
				t.Fatal(err)
			}
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
		return path
	}
	cases := []struct {
		name    string
		entries map[string]os.FileMode
		links   map[string]string
		want    string
	}{
		{"escape", map[string]os.FileMode{"v/../../escape": 0644}, nil, "invalid archive path"},
		{"absolute", map[string]os.FileMode{"/tmp/x": 0644}, nil, "invalid archive path"},
		{"dot top", map[string]os.FileMode{".hidden/file": 0644}, nil, "invalid version directory"},
		{"two tops", map[string]os.FileMode{"v1/a": 0644, "v2/a": 0644}, nil, "more than one version"},
		{"case aliasing", map[string]os.FileMode{"v/Runtime/Bun": 0755, "v/runtime/bun": 0755}, nil, "case-aliasing"},
		{"symlink", map[string]os.FileMode{"v/file": 0644}, map[string]string{"v/link": "file"}, "non-regular archive entry"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			archive := craft(t, tc.entries, tc.links)
			if _, err := extractRelease(archive, t.TempDir()); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q, got %v", tc.want, err)
			}
		})
	}
	if _, err := extractRelease(filepath.Join(t.TempDir(), "absent.zip"), t.TempDir()); err == nil {
		t.Fatal("extracted an absent archive")
	}
}

func TestPruneStaging(t *testing.T) {
	root := t.TempDir()
	foreign := filepath.Join(root, "foreign.txt")
	if err := os.WriteFile(foreign, []byte("operator data"), 0600); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(root, ".can-stage-interrupted", "versions", "x")
	if err := os.MkdirAll(stale, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".can-stage-interrupted", "partial"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".can-current-temp"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := PruneStaging(root); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(foreign); err != nil {
		t.Fatal("prune touched a foreign file")
	}
	for _, stale := range []string{".can-stage-interrupted", ".can-current-temp"} {
		if _, err := os.Lstat(filepath.Join(root, stale)); !os.IsNotExist(err) {
			t.Fatalf("prune left %s", stale)
		}
	}
	selected, err := Selection(root)
	if err != nil || selected != "" {
		t.Fatalf("empty root selects %q: %v", selected, err)
	}
}
