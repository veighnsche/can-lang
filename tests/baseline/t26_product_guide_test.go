// T26 product-guide and agent-comparison consistency checks.
//
// The guide (docs/syntax-taste/t26-product-guide-2026-09-24.md) is the
// authoritative T26 documentation update; the report
// (tests/baseline/reports/t26-agent-comparison-2026-09-24.json) is the
// held-out comparison replay. These tests pin the required coverage
// and keep every grounded claim anchored to the file that proves it.
package baseline

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const t26Guide = "docs/syntax-taste/t26-product-guide-2026-09-24.md"

// Every required T26 contract statement must appear in the guide, and
// each one must stay grounded in the implementation file that proves
// it. A phrase that drifts from its anchor is a defect in the guide,
// not in the anchor.
//
// Post-upgrade note: the selected behavior intentionally renames and
// relocates the invoice actions into the shared handler-free contract
// (load_invoice_grid, save_invoice_grid, save_invoice_html) and scopes
// the replay key by tenant. The T26 guide stays the historical record;
// the anchors below track the selected contracts, not the old spellings.
func TestT26ProductGuideCoversContracts(t *testing.T) {
	root := repoRoot(t)
	guide, err := os.ReadFile(filepath.Join(root, t26Guide))
	if err != nil {
		t.Fatal(err)
	}
	text := string(guide)
	required := []string{
		// Supported platforms.
		"bun-1.4.2-darwin-arm64-v1",
		"bun-1.4.2-linux-amd64-v1",
		"Debian 13",
		"glibc 2.41",
		"chromium 140",
		"webkit 26",
		"firefox",
		"PostgreSQL 17.11",
		// Action recipes, both modes.
		"save_invoice",
		"save_invoice_form",
		"load_invoice",
		"8192",
		"64-row",
		"2048",
		"lines_order",
		"serve_form_action",
		"422",
		"allow",
		// SQL guarantees.
		"libpg_query",
		"row_limit_parameter",
		"commit_unknown",
		"sql.unsafe",
		// Race/shutdown semantics.
		"exactly once",
		"resource_state",
		// Form/JSON/owner boundaries and replay.
		"owner record",
		"owner-only",
		"invoice_replay",
		"operation_id",
		// Initial offline limit.
		"no reload durability",
		"no automatic queue",
		// Stream correction, deferrals, comparisons.
		"close_reader",
		"primary error",
		"DI-17",
		"no significance is claimed",
	}
	for _, phrase := range required {
		if !strings.Contains(text, phrase) {
			t.Errorf("guide %s omits required phrase %q", t26Guide, phrase)
		}
	}

	// Anchors: the guide must not outlive the implementation it cites.
	anchors := map[string][]string{
		"examples/stream/src/main.can": {
			"close_reader",
			"never\n/// replaces the primary error",
		},
		"examples/invoice/vendor/billing/src/invoice_contract/invoice_contract.can": {
			"action load_invoice_grid\n",
			"action save_invoice_grid\n",
			"action save_invoice_html\n",
		},
		"runtime/platform/form.ts": {
			"maxFormRows = 64",
			"maxFormRowBytes = 2048",
		},
		"runtime/platform/action-json.ts": {
			"ACTION_JSON_BODY_LIMIT = 8192",
		},
		"examples/invoice/schema.sql": {
			"invoice_replay",
			"PRIMARY KEY (actor, tenant, invoice_id, operation_id)",
		},
		"docs/implementation/shutdown.md": {
			"exactly once",
		},
		"compiler/internal/catalogue/catalogue.json": {
			"can.std.http@1::serve_form_action",
		},
		"tests/baseline/candidates/T26-native-ai/README.md": {
			"deferred",
		},
	}
	for file, phrases := range anchors {
		raw, err := os.ReadFile(filepath.Join(root, file))
		if err != nil {
			t.Errorf("anchor %s unreadable: %v", file, err)
			continue
		}
		for _, phrase := range phrases {
			if !strings.Contains(string(raw), phrase) {
				t.Errorf("anchor %s no longer contains %q", file, phrase)
			}
		}
	}
}

type t26Report struct {
	Task         string `json:"task"`
	Significance string `json:"significance"`
	AgentTrials  struct {
		Status   string `json:"status"`
		Attempts int    `json:"attempts"`
	} `json:"agentTrials"`
	Cases []struct {
		ID          string `json:"id"`
		AgentTrials string `json:"agentTrials"`
		Disposition string `json:"disposition"`
		Tokens      *int   `json:"tokens"`
	} `json:"cases"`
	Deferred []string `json:"deferred"`
}

// The comparison report must cover every registered case, run no
// phantom trials, claim no significance, and name its deferrals.
func TestT26AgentComparisonReportConsistent(t *testing.T) {
	root := repoRoot(t)
	var reg registryFile
	loadJSON(t, filepath.Join(root, "tests", "baseline", "registry.json"), &reg)
	if reg.Task != "T01" {
		t.Fatalf("registry task = %q, want T01 (T26 must not revise it)", reg.Task)
	}
	var rep t26Report
	loadJSON(t, filepath.Join(root, "tests", "baseline", "reports", "t26-agent-comparison-2026-09-24.json"), &rep)
	if rep.Task != "T26" {
		t.Fatalf("report task = %q, want T26", rep.Task)
	}
	if rep.AgentTrials.Status != "not-run" || rep.AgentTrials.Attempts != 0 {
		t.Fatalf("report trials = %+v, want not-run with 0 attempts", rep.AgentTrials)
	}
	if !strings.Contains(strings.ToLower(rep.Significance), "none claimed") {
		t.Fatalf("report must state that no significance is claimed: %q", rep.Significance)
	}
	got := map[string]bool{}
	for _, c := range rep.Cases {
		got[c.ID] = true
		if c.AgentTrials != "not-run" {
			t.Errorf("case %s trials = %q, want not-run", c.ID, c.AgentTrials)
		}
		if c.Tokens != nil {
			t.Errorf("case %s reports tokens %d without trials", c.ID, *c.Tokens)
		}
		if c.Disposition == "" {
			t.Errorf("case %s has no disposition", c.ID)
		}
	}
	for _, c := range reg.Cases {
		if !got[c.ID] {
			t.Errorf("report omits registered case %s", c.ID)
		}
	}
	if len(rep.Cases) != len(reg.Cases) {
		t.Errorf("report covers %d cases, registry has %d", len(rep.Cases), len(reg.Cases))
	}
	if len(rep.Deferred) == 0 {
		t.Error("report names no deferred candidates")
	}
	// The T26 candidate slot exists and records the deferral.
	slot := filepath.Join(root, "tests", "baseline", "candidates", "T26-native-ai")
	for _, name := range []string{"README.md", "candidate.json"} {
		raw, err := os.ReadFile(filepath.Join(slot, name))
		if err != nil {
			t.Errorf("candidate slot missing %s: %v", name, err)
			continue
		}
		if !strings.Contains(strings.ToLower(string(raw)), "deferr") {
			t.Errorf("candidate slot %s does not record the deferral", name)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "tests", "baseline", "reports", "t26-agent-comparison-2026-09-24.md")); err != nil {
		t.Errorf("prose comparison report missing: %v", err)
	}
}

// Generated reference artifacts must be fresh: the catalogue source
// of truth regenerates byte-identical outputs.
func TestT26GeneratedArtifactsFresh(t *testing.T) {
	root := repoRoot(t)
	cmd := exec.Command("go", "run", "./compiler/internal/catalogue/cmd/cataloguegen", "--check")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("cataloguegen --check: %v\n%s", err, out)
	}
	info, err := os.Stat(filepath.Join(root, "runtime", "catalogue.ts"))
	if err != nil || info.Size() == 0 {
		t.Fatalf("runtime/catalogue.ts missing or empty: %v", err)
	}
}
