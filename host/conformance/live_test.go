package conformance

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// liveDebtCommand is the exact command that runs the live legs once
// C01 provisions the pinned browsers. It is the skip message below
// and the debt ledger in live/README.md — one string, three places.
const liveDebtCommand = "CAN_D02_LIVE=1 go test ./host/conformance/ -run TestLiveBrowserLegs -count=1 -v"

// TestLiveBrowserLegs is the CI-shaped gate for the pinned
// live-browser legs (real localStorage/navigator.clipboard on
// Chromium/WebKit/Firefox). While C01 is blocked-open it skips with
// the precise unblock command; it never claims an unrun leg as
// passing. With CAN_D02_LIVE=1 it pre-bundles the delivered adapters,
// resolves the pinned Playwright harness, runs the runner once per
// browser, and fails on any failed check.
func TestLiveBrowserLegs(t *testing.T) {
	if os.Getenv("CAN_D02_LIVE") != "1" {
		t.Skipf("live legs debt (C01 blocked-open); unblock with: %s", liveDebtCommand)
	}
	root := repoRoot(t)
	playwright := filepath.Join(root, "tests/integration/browser/node_modules/playwright")
	if _, err := os.Stat(playwright); err != nil {
		t.Fatalf("pinned playwright missing at %s; provision per live/README.md, then: %s", playwright, liveDebtCommand)
	}
	bun, err := exec.LookPath("bun")
	if err != nil {
		t.Fatal("bun not on PATH; live legs need bun build for the adapter bundles")
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal("node not on PATH; live legs run the Playwright runner under node")
	}
	bundles := t.TempDir()
	for _, entry := range [][2]string{
		{"host/adapters/storage.ts", "storage.bundle.js"},
		{"host/adapters/clipboard.ts", "clipboard.bundle.js"},
	} {
		cmd := exec.Command(bun, "build", entry[0], "--outfile", filepath.Join(bundles, entry[1]), "--format=esm")
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("bundle %s: %v\n%s", entry[0], err, out)
		}
	}
	for _, browser := range []string{"chromium", "webkit", "firefox"} {
		t.Run(browser, func(t *testing.T) {
			outdir := t.TempDir()
			cmd := exec.Command(node, filepath.Join(root, "host/conformance/live/storage-clipboard.mjs"), browser, bundles, outdir)
			cmd.Dir = root
			cmd.Env = append(os.Environ(), "NODE_PATH="+filepath.Join(root, "tests/integration/browser/node_modules"))
			out, err := cmd.CombinedOutput()
			t.Logf("runner output:\n%s", out)
			if err != nil {
				t.Fatalf("live legs failed on %s: %v", browser, err)
			}
			report, err := os.ReadFile(filepath.Join(outdir, "report.json"))
			if err != nil {
				t.Fatalf("live legs on %s wrote no report: %v", browser, err)
			}
			var decoded struct {
				Browser string `json:"browser"`
				Checks  []struct {
					Name   string `json:"name"`
					Passed bool   `json:"passed"`
					Detail string `json:"detail"`
				} `json:"checks"`
			}
			if err := json.Unmarshal(report, &decoded); err != nil {
				t.Fatalf("parse %s report: %v", browser, err)
			}
			if decoded.Browser != browser || len(decoded.Checks) == 0 {
				t.Fatalf("live report on %s is empty or misattributed: %s", browser, report)
			}
			for _, check := range decoded.Checks {
				if !check.Passed {
					t.Errorf("live check %q failed on %s: %s", check.Name, browser, check.Detail)
				}
			}
		})
	}
}
