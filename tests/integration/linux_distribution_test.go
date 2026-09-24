package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/distribution"
)

// TestLinuxInstalledArtifactSmoke builds, releases, installs, and smokes
// the Debian 13 amd64 distribution on its own target. It runs no
// macOS-only tooling and needs no network: full network denial comes from
// the documented invocation (docker run --network none), which qualify.py
// verifies against live interfaces and fails closed without.
func TestLinuxInstalledArtifactSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux amd64 installed-artifact smoke only")
	}
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE to the pinned bun-linux-x64.zip")
	}
	sourceRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Minute)
	defer cancel()

	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "linux-smoke")
	if err != nil {
		t.Fatal(err)
	}
	artifacts, err := distribution.Release(ctx, bundle, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	installRoot := t.TempDir()
	installed, err := distribution.Install(ctx, artifacts.Archive, artifacts.SHA256, installRoot)
	if err != nil {
		t.Fatal(err)
	}
	selection, err := distribution.Selection(installRoot)
	if err != nil || selection != installed {
		t.Fatalf("wrong install selection %q: %v", selection, err)
	}
	canlc := filepath.Join(installRoot, "current", "bin", "canlc")
	sidecar := filepath.Join(installRoot, "current", "runtime", "bun")

	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	run := func(name string, args ...string) (int, string, string) {
		t.Helper()
		cmd := exec.CommandContext(ctx, canlc, append([]string{name}, args...)...)
		cmd.Dir = home
		cmd.Env = []string{"PATH=/nonexistent", "HOME=" + home}
		var out, diag bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &diag
		if err := cmd.Run(); err != nil {
			var status *exec.ExitError
			if !errors.As(err, &status) {
				t.Fatal(err)
			}
			return status.ExitCode(), out.String(), diag.String()
		}
		return 0, out.String(), diag.String()
	}

	status, out, diag := run("runtime-check")
	if status != 0 || diag != "" {
		t.Fatalf("runtime-check: %d %s %s", status, out, diag)
	}
	var identity struct {
		Bun          string `json:"bun"`
		Revision     string `json:"revision"`
		Platform     string `json:"platform"`
		Architecture string `json:"architecture"`
		Executable   string `json:"executable"`
	}
	if err := json.Unmarshal([]byte(out), &identity); err != nil {
		t.Fatal(err)
	}
	if identity.Bun != "1.4.2" || identity.Platform != "linux" || identity.Architecture != "x64" {
		t.Fatalf("wrong installed runtime identity: %+v", identity)
	}
	if identity.Revision != "744846f844374847c902b5e7fd59b4342a51ef99" {
		t.Fatalf("wrong installed runtime revision: %+v", identity)
	}
	if !strings.HasPrefix(identity.Executable, installed) {
		t.Fatalf("runtime-check ran outside the install root: %+v", identity)
	}

	if python, err := exec.LookPath("python3"); err != nil {
		t.Log("python3 unavailable; skipping native qualification leg")
	} else {
		reportPath := filepath.Join(t.TempDir(), "native-capabilities-linux.json")
		qualify := exec.CommandContext(ctx, python, filepath.Join(sourceRoot, "distribution", "qualify.py"),
			"--archive", archive, "--report", reportPath,
			"--target-file", filepath.Join(sourceRoot, "distribution", "target-linux-amd64.json"),
			"--network-isolation", "documented container invocation (docker run --network none)")
		qualify.Dir = home
		qualify.Env = []string{"PATH=/usr/bin:/bin", "HOME=" + home}
		if output, err := qualify.CombinedOutput(); err != nil {
			t.Fatalf("native qualification: %v\n%s", err, output)
		}
		raw, err := os.ReadFile(reportPath)
		if err != nil {
			t.Fatal(err)
		}
		var report struct {
			Passed       bool            `json:"passed"`
			TargetID     string          `json:"targetId"`
			Capabilities map[string]bool `json:"capabilities"`
			Behaviors    map[string]bool `json:"behaviors"`
			Execution    map[string]any  `json:"execution"`
			Failures     []string        `json:"failures"`
		}
		if err := json.Unmarshal(raw, &report); err != nil {
			t.Fatal(err)
		}
		if !report.Passed || report.TargetID != "bun-1.4.2-linux-amd64-v1" {
			t.Fatalf("native qualification failed: %v", report.Failures)
		}
		for name, ok := range report.Capabilities {
			if !ok {
				t.Fatalf("missing native API on installed target: %s", name)
			}
		}
		for name, ok := range report.Behaviors {
			if !ok {
				t.Fatalf("failing behavior probe on installed target: %s", name)
			}
		}
		if report.Execution["negativeTestsPassed"] != true || report.Execution["glibc"] == nil {
			t.Fatalf("native qualification lacks rejection or glibc evidence: %v", report.Execution)
		}
		t.Logf("native ok: %d APIs, %d probes", len(report.Capabilities), len(report.Behaviors))
	}

	stage := func(name string) string {
		t.Helper()
		root, err := filepath.EvalSymlinks(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		entries, err := os.ReadDir(filepath.Join(sourceRoot, "examples", name))
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				data, err := os.ReadFile(filepath.Join(sourceRoot, "examples", name, entry.Name()))
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, entry.Name()), data, 0600); err != nil {
					t.Fatal(err)
				}
				continue
			}
			if err := os.MkdirAll(filepath.Join(root, entry.Name()), 0700); err != nil {
				t.Fatal(err)
			}
			files, err := os.ReadDir(filepath.Join(sourceRoot, "examples", name, entry.Name()))
			if err != nil {
				t.Fatal(err)
			}
			for _, file := range files {
				data, err := os.ReadFile(filepath.Join(sourceRoot, "examples", name, entry.Name(), file.Name()))
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, entry.Name(), file.Name()), data, 0600); err != nil {
					t.Fatal(err)
				}
			}
		}
		return root
	}
	assertExample := func(name, root string) {
		t.Helper()
		status, out, diag := run("assert", root)
		if status != 0 || diag != "" {
			t.Fatalf("%s assert: %d %s %s", name, status, out, diag)
		}
		var report struct {
			Passed     bool  `json:"passed"`
			Assertions []any `json:"assertions"`
		}
		if err := json.Unmarshal([]byte(out), &report); err != nil || !report.Passed || len(report.Assertions) == 0 {
			t.Fatalf("%s invalid assert report %v %s", name, err, out)
		}
	}
	runExample := func(name, root string, want string, args ...string) {
		t.Helper()
		argv := append([]string{root, "--"}, args...)
		status, out, diag := run("run", argv...)
		if status != 0 || out != want || diag != "" {
			t.Fatalf("%s run: %d stdout=%q stderr=%s", name, status, out, diag)
		}
	}

	process := stage("process")
	assertExample("process", process)
	runExample("process", process, "ok", "ada")

	files := stage("files")
	assertExample("files", files)
	listDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(listDir, "a.txt"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	runExample("files", files, "file a.txt\n", listDir)

	crypto := stage("crypto")
	assertExample("crypto", crypto)
	runExample("crypto", crypto, "match", "correct horse")
	runExample("crypto", crypto, "mismatch", "wrong")

	sqlite := stage("sqlite")
	assertExample("sqlite", sqlite)
	db := filepath.Join(t.TempDir(), "notes.sqlite")
	setup := exec.CommandContext(ctx, sidecar,
		filepath.Join(sourceRoot, "tests/integration/testdata/sql/sqlite-driver.ts"),
		"setup", db, filepath.Join(sourceRoot, "tests/integration/testdata/sql/sqlite-seed.sql"))
	setup.Dir = home
	setup.Env = []string{"PATH=/nonexistent", "HOME=" + home}
	if output, err := setup.CombinedOutput(); err != nil {
		t.Fatalf("sqlite setup: %v %s", err, output)
	}
	seed := exec.CommandContext(ctx, sidecar,
		filepath.Join(sourceRoot, "distribution/linux/smoke/sqlite-seed.ts"), db)
	seed.Dir = home
	seed.Env = []string{"PATH=/nonexistent", "HOME=" + home}
	if output, err := seed.CombinedOutput(); err != nil {
		t.Fatalf("sqlite seed: %v %s", err, output)
	}
	runExample("sqlite", sqlite, "7: remember", db)

	first, _ := applicationBuildOn(t, ctx, canlc, home, sqlite)
	second, _ := applicationBuildOn(t, ctx, canlc, home, sqlite)
	if first != second {
		t.Fatalf("rebuild drifted: %s vs %s", first, second)
	}

	t.Logf("linux installed-artifact smoke passed on %s", installed)
}

// TestLinuxPostgresRoundtrip runs the Bun.SQL create/insert/select/drop
// roundtrip through an installed Linux sidecar against a disposable
// PostgreSQL 17. It is separate from the offline smoke because it needs
// database networking, which the smoke denies by construction.
func TestLinuxPostgresRoundtrip(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux amd64 postgres roundtrip only")
	}
	installRoot := os.Getenv("CAN_LINUX_INSTALL_ROOT")
	database := os.Getenv("DATABASE_URL")
	if installRoot == "" || database == "" {
		t.Skip("set CAN_LINUX_INSTALL_ROOT and DATABASE_URL for the postgres roundtrip")
	}
	sourceRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	sidecar := filepath.Join(installRoot, "current", "runtime", "bun")
	if info, err := os.Stat(sidecar); err != nil || info.IsDir() {
		t.Fatalf("installed sidecar missing at %s", sidecar)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	roundtrip := exec.CommandContext(ctx, sidecar,
		filepath.Join(sourceRoot, "distribution/linux/smoke/pg-driver.ts"))
	roundtrip.Dir = t.TempDir()
	roundtrip.Env = []string{"PATH=/nonexistent", "HOME=" + roundtrip.Dir, "DATABASE_URL=" + database}
	output, err := roundtrip.CombinedOutput()
	if err != nil {
		t.Fatalf("postgres roundtrip: %v %s", err, output)
	}
	var round struct {
		ServerVersion string `json:"serverVersion"`
		ID            string `json:"id"`
		Body          string `json:"body"`
	}
	if err := json.Unmarshal(output, &round); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(round.ServerVersion, "17.") || round.ID != "7" || round.Body != "remember" {
		t.Fatalf("wrong postgres roundtrip: %s", output)
	}
	t.Logf("postgres ok: server %s", round.ServerVersion)
}

// applicationBuildOn builds a staged project through an installed launcher
// without macOS-only wrappers and reports its build ID and directory.
func applicationBuildOn(t *testing.T, ctx context.Context, canlc, home, root string) (string, string) {
	t.Helper()
	cmd := exec.CommandContext(ctx, canlc, "build", root)
	cmd.Dir = home
	cmd.Env = []string{"PATH=/nonexistent", "HOME=" + home}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build: %v %s", err, out)
	}
	var report struct {
		BuildID   string `json:"buildID"`
		Directory string `json:"directory"`
	}
	if err := json.Unmarshal(out, &report); err != nil || report.BuildID == "" || report.Directory == "" {
		t.Fatalf("invalid build report %v %s", err, out)
	}
	return report.BuildID, report.Directory
}
