package integration

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	up23VerdictPortChromium = 18649
	up23VerdictPortWebkit   = 18650
)

// up23CheckToLeg maps the live invoice.mjs harness checks onto the
// machine-readable UP23 verdict legs consumed by
// TestInvoiceBrowserGuardDOM. Every leg must pass on both named engines
// for the verdict to be written.
var up23CheckToLeg = map[string]string{
	"save-200-swap":         "swap-200",
	"denied-403-swap":       "swap-403",
	"stale-409-swap":        "swap-409",
	"validation-422-swap":   "swap-422",
	"busy-503-swap":         "swap-503",
	"target-absence-stable": "missing-target-before",
	"missing-target-during": "missing-target-during",
	"oob-rejected":          "oob-rejected",
	"partial-rejected":      "partial-rejected",
	"control-header":        "control-headers-rejected",
	"remount-stable":        "remount-stable",
}

// TestUP23WriteVerdict is the UP23 browser-DOM-guard handoff step. It
// stages the paired invoice server (grid browser build served under the
// verified server report), runs the committed invoice.mjs guard legs
// live in both named engines, and writes the machine-readable verdict
// file that TestInvoiceBrowserGuardDOM consumes via CAN_UP23_RESULTS.
//
// Run it standalone before the full suite:
//
//	CAN_UP23_OUT=/tmp/up23-verdict.json go test ./tests/integration -run TestUP23WriteVerdict -count=1
//
// then run the suite with CAN_UP23_RESULTS=/tmp/up23-verdict.json. The
// verdict binds the exact candidate commit; a stale file from another
// commit must not be reused. Without CAN_UP23_OUT the test skips so the
// full suite never pays for a second browser pass.
func TestUP23WriteVerdict(t *testing.T) {
	out := os.Getenv("CAN_UP23_OUT")
	if out == "" {
		t.Skip("set CAN_UP23_OUT to write the UP23 verdict file")
	}
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged UP23 execution")
	}
	nodePath, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is required for the browser harness")
	}
	sourceRoot := mustSourceRoot(t)
	browserDir := filepath.Join(sourceRoot, "tests/integration/browser")
	if _, err := os.Stat(filepath.Join(browserDir, "node_modules/playwright/package.json")); err != nil {
		t.Skip("run bun ci in tests/integration/browser for the pinned harness")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Minute)
	defer cancel()

	// Both named engines are required: probe before building so a
	// missing engine fails fast instead of after a long build.
	versions := map[string]string{}
	for _, engine := range []string{"chromium", "webkit"} {
		versions[engine] = gate5BrowserProbe(t, ctx, nodePath, browserDir, engine)
		t.Logf("up23 required browser %s at %s", engine, versions[engine])
	}

	toolchain, canlc, _, _ := gate5Toolchain(t, ctx, sourceRoot, archive)
	serverRoot, serverHome := stageApplication(t, ctx, toolchain, sourceRoot, "invoice")
	gridRoot, gridHome := stageProject(t, sourceRoot, "examples/invoice-grid")
	browser := gate5Build(t, ctx, canlc, gridHome, gridRoot, "--target", "browser")
	pairing := gate5PairBuild(t, ctx, canlc, serverHome, serverRoot, filepath.Join(browser.Directory, "browser", "manifest.json"))
	driver := filepath.Join(sourceRoot, "tests/integration/testdata/invoice/driver.ts")
	t.Logf("up23 paired: browser %s server %s entry %s", browser.BuildID[:12], pairing.BuildID[:12], pairing.Entry)

	legs := map[string]struct {
		Verdict  string `json:"verdict"`
		Evidence string `json:"evidence"`
	}{}
	evidence := map[string][]string{}
	ports := map[string]int{"chromium": up23VerdictPortChromium, "webkit": up23VerdictPortWebkit}
	for _, engine := range []string{"chromium", "webkit"} {
		home := t.TempDir()
		db := gate5SeedDB(t, ctx, toolchain, home, driver, serverRoot)
		base, stop := serveInvoice(t, ctx, toolchain, home, filepath.Join(pairing.Directory, "entry.ts"), db, ports[engine], "")
		outdir := t.TempDir()
		gate5RunHarness(t, ctx, nodePath, browserDir, "up23", engine,
			"invoice.mjs", base, outdir, db, engine, pairing.Entry)
		stop()
		raw, err := os.ReadFile(filepath.Join(outdir, "report.json"))
		if err != nil {
			t.Fatal(err)
		}
		report := gate5ReadReport(t, "up23", engine, raw, 20, false)
		gate5InvoiceOccurrences(t, engine, report)
		for _, check := range report.Checks {
			leg, ok := up23CheckToLeg[check.Name]
			if !ok {
				continue
			}
			evidence[leg] = append(evidence[leg], engine+" "+report.Version+" ("+check.Detail+")")
		}
		store := inspectInvoice(t, ctx, toolchain, t.TempDir(), driver, db)
		row := requireInvoice(t, store, "7", "5", gate5SeedSeats, "Guard Final")
		gate5RequireReplay(t, "up23", engine, store, "2", "3", "4", "5")
		_ = row
		t.Logf("up23 %s %s: 20 checks, %d guard occurrences, rev 5 committed with 4 replay rows",
			engine, report.Version, len(report.Occurrences))
	}
	for leg, proofs := range evidence {
		if len(proofs) != 2 {
			t.Fatalf("up23 leg %s ran on %d engines, want both", leg, len(proofs))
		}
		legs[leg] = struct {
			Verdict  string `json:"verdict"`
			Evidence string `json:"evidence"`
		}{Verdict: "pass", Evidence: strings.Join(proofs, "; ")}
	}
	if len(legs) != len(up23CheckToLeg) {
		t.Fatalf("up23 mapped %d legs, want %d", len(legs), len(up23CheckToLeg))
	}
	commitOut, err := exec.CommandContext(ctx, "git", "-C", sourceRoot, "rev-parse", "HEAD").CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	verdict := map[string]any{
		"engine": "chromium " + versions["chromium"] + " + webkit " + versions["webkit"],
		"commit": strings.TrimSpace(string(commitOut)),
		"legs":   legs,
	}
	encoded, err := json.MarshalIndent(verdict, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, append(encoded, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("up23 verdict: %d legs green on both engines at commit %s", len(legs), strings.TrimSpace(string(commitOut))[:12])
}
