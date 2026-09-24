// T24 staged invoice grid coverage: bundled assert plus deterministic
// browser-target builds of the Can-authored editable grid. Check-level
// coverage (browser closure, cross-target contract, emission) lives in
// compiler/internal/driver.
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/distribution"
)

func canlcBrowser(t *testing.T, ctx context.Context, bundle, home string, args ...string) (int, string, string) {
	t.Helper()
	argv := append([]string{"-p", "(version 1)(allow default)(deny network*)", filepath.Join(bundle, "bin/canlc")}, args...)
	cmd := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", argv...)
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

func TestInvoiceGridStagedBrowserBuild(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged grid execution")
	}
	sourceRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "invoice-grid")
	if err != nil {
		t.Fatal(err)
	}
	root, home := stageProject(t, sourceRoot, "examples/invoice-grid")
	status, out, diag := canlcOffline(t, ctx, bundle, home, "assert", root)
	if status != 0 || diag != "" {
		t.Fatalf("grid assert: %d %s %s", status, out, diag)
	}
	var report struct {
		Passed     bool `json:"passed"`
		Assertions []struct {
			Evidence []string `json:"evidence"`
		} `json:"assertions"`
	}
	if err := json.Unmarshal([]byte(out), &report); err != nil || !report.Passed || len(report.Assertions) == 0 {
		t.Fatalf("invalid grid assert report %v %s", err, out)
	}
	real := 0
	for _, assertion := range report.Assertions {
		for _, evidence := range assertion.Evidence {
			if evidence == "real-can" {
				real++
				break
			}
		}
	}
	if real == 0 {
		t.Fatal("grid asserts nothing real")
	}
	build := func() (string, string) {
		t.Helper()
		status, out, diag := canlcBrowser(t, ctx, bundle, home, "build", "--target", "browser", root)
		if status != 0 || diag != "" {
			t.Fatalf("grid browser build: %d %s %s", status, out, diag)
		}
		var manifest struct {
			BuildID   string `json:"buildID"`
			Directory string `json:"directory"`
			Entry     string `json:"entry"`
			Asset     string `json:"asset"`
		}
		if err := json.Unmarshal([]byte(out), &manifest); err != nil || manifest.BuildID == "" || manifest.Directory == "" {
			t.Fatalf("invalid grid build manifest %v %s", err, out)
		}
		return manifest.BuildID, manifest.Directory
	}
	firstID, firstDir := build()
	secondID, _ := build()
	if firstID != secondID {
		t.Fatalf("grid browser rebuild drifted: %s vs %s", firstID, secondID)
	}
	asset, err := os.ReadFile(filepath.Join(firstDir, "browser/asset.json"))
	if err != nil {
		t.Fatal(err)
	}
	var descriptor struct {
		Profile string            `json:"profile"`
		Entry   string            `json:"entry"`
		Files   map[string]string `json:"files"`
	}
	if err := json.Unmarshal(asset, &descriptor); err != nil {
		t.Fatal(err)
	}
	if descriptor.Profile != "browser-main" || descriptor.Entry != "browser.ts" || len(descriptor.Files) == 0 {
		t.Fatalf("invalid grid browser asset %s", asset)
	}
	assertNoStrayEmit(t, root, firstDir)
	t.Logf("grid: %d assertions (%d real-can), browser build %s", len(report.Assertions), real, firstID[:12])
}
