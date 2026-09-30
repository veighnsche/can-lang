// W2 browser-build end-to-end negative (H11-F6 closure): an unadmitted
// capability name fails the full `canlc build --target browser`
// pipeline with file-attributed location evidence — not just the
// check phase covered by TestVendorCapabilitiesRejectWithLocation.
package conformance

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/distribution"
)

// TestBrowserBuildRejectsUnadmittedCapabilities builds a development
// distribution and proves every vendor-flavored capability name fails
// the real browser build with a located unknown-package diagnostic.
// It needs CAN_BUN_ARCHIVE (pinned local zip) and skips honestly
// without it; a skip is not a pass.
func TestBrowserBuildRejectsUnadmittedCapabilities(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for the browser-build e2e negative leg")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, repoRoot(t), t.TempDir(), archive, "w2-e2e")
	if err != nil {
		t.Fatalf("build dev distribution: %v", err)
	}
	canlc := filepath.Join(bundle, "bin", "canlc")
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	for _, capability := range vendorCapabilities {
		t.Run(capability, func(t *testing.T) {
			raw := t.TempDir()
			root, err := filepath.EvalSymlinks(raw)
			if err != nil {
				t.Fatal(err)
			}
			files := map[string]string{
				"can.project.json": `{"source_root":"src","error_registry":"can.errors.json"}`,
				"can.errors.json":  `{"active":[],"retired":[]}`,
				"src/main.can": "package app\n" +
					"    provides []\n" +
					"    uses [" + capability + "]\n" +
					"fn void main\n" +
					"    emits {}\n" +
					"    given\n" +
					"        str[] arguments\n" +
					"    asserts\n" +
					"        empty: [] => ok\n" +
					"    ok\n",
			}
			for name, text := range files {
				p := filepath.Join(root, name)
				if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(p, []byte(text), 0600); err != nil {
					t.Fatal(err)
				}
			}
			cmd := exec.Command(canlc, "build", "--target", "browser", root)
			combined, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatalf("capability %q admitted; browser build passed:\n%s", capability, combined)
			}
			output := string(combined)
			if !strings.Contains(output, "src/main.can") {
				t.Fatalf("capability %q rejection lacks file location:\n%s", capability, output)
			}
			if !strings.Contains(output, `unknown package "`+capability+`"`) {
				t.Fatalf("capability %q rejection names no capability:\n%s", capability, output)
			}
		})
	}
}
