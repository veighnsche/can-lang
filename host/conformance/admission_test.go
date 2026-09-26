// Package conformance is the Lane D (host integration) D02 conformance
// suite: no-self-admission legs, review-manifest closure, and located
// rejection of the unadmitted host capabilities, all runnable without
// a browser. The bun legs beside this file re-run the D01 T2/T3 legs
// against the delivered modules; the live-browser legs live under
// live/ and stay debt until C01 unblocks.
package conformance

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// hypotheticalPackages are the catalogue packages the T1 sketches would
// introduce. None may exist: the closed catalogue admits no host
// capability without the E-owned merge plus distribution review. This
// leg reads the E-owned merge source directly (no internal imports);
// E's own catalogue tests pin generated output to this same file.
var hypotheticalPackages = []string{
	"can.std.storage@",
	"can.std.clipboard@",
	"can.std.chart@",
	// D03: the vendor-specific catalogue identities W2 would have
	// introduced on a vendor-patch path. None may exist either: W2
	// runs behind the assigned T3 boundary with no catalogue change.
	"can.std.chart_vendor_b@",
	"can.std.vendor_b@",
}

func TestDeliveredCapabilitiesUnadmitted(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "compiler/internal/catalogue/catalogue.json"))
	if err != nil {
		t.Fatalf("read catalogue: %v", err)
	}
	var inventory struct {
		Operations []struct {
			Identity string `json:"identity"`
		} `json:"operations"`
	}
	if err := json.Unmarshal(raw, &inventory); err != nil {
		t.Fatalf("parse catalogue: %v", err)
	}
	if len(inventory.Operations) == 0 {
		t.Fatal("catalogue has no operations; leg is vacuous")
	}
	for _, operation := range inventory.Operations {
		for _, prefix := range hypotheticalPackages {
			if strings.HasPrefix(operation.Identity, prefix) {
				t.Fatalf("delivered capability %q admitted; self-admission path exists", operation.Identity)
			}
		}
	}
	generated, err := os.ReadFile(filepath.Join(repoRoot(t), "runtime/catalogue.ts"))
	if err != nil {
		t.Fatalf("read generated catalogue: %v", err)
	}
	for _, prefix := range hypotheticalPackages {
		if strings.Contains(string(generated), prefix) {
			t.Fatalf("generated catalogue contains %q; self-admission path exists", prefix)
		}
	}
}

// deliverableMarkers are distinctive D02 strings: a hit in any owned
// tree means a deliverable leaked into a build, a catalogue, or a
// checked-in example outside distribution review.
var deliverableMarkers = []string{
	"host/adapters/",
	"host/companions/",
	"d02.chart/1",
	"D02_CHART_COMPANION_TOKEN",
	"CHART_SELECT_ROUNDTRIP_BUDGET_MS",
	// D03: Vendor B markers. A hit in any owned tree means the second
	// vendor leaked a vendor-specific patch into a build, a catalogue,
	// or a checked-in example outside distribution review.
	"chart-vendor-b",
	"companion-b.example",
	"D03_CHART_VENDOR_B_TOKEN",
	"2026-09-26.d03-chart-vendor-b",
}

// TestDeliverablesUnreferenced walks every tree another lane owns and
// fails if a D02 deliverable marker leaks into it. Deliverables may
// import from owned trees (F01 C-G policy); the reverse is
// self-admission.
func TestDeliverablesUnreferenced(t *testing.T) {
	root := repoRoot(t)
	owned := []string{
		"runtime",
		"tools/runtime",
		"compiler",
		"examples",
		"distribution",
		"internal",
	}
	var violations []string
	for _, tree := range owned {
		base := filepath.Join(root, tree)
		if err := filepath.WalkDir(base, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				name := entry.Name()
				if name == "vendor" || name == "node_modules" || name == ".git" {
					return filepath.SkipDir
				}
				return nil
			}
			if !(strings.HasSuffix(path, ".ts") || strings.HasSuffix(path, ".go") || strings.HasSuffix(path, ".json")) {
				return nil
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			for _, marker := range deliverableMarkers {
				if strings.Contains(string(raw), marker) {
					violations = append(violations, path+" contains "+marker)
				}
			}
			return nil
		}); err != nil {
			t.Fatalf("walk %s: %v", base, err)
		}
	}
	if len(violations) > 0 {
		t.Fatalf("deliverable referenced from owned trees: %v", violations)
	}
}

type reviewManifest struct {
	Status       string `json:"status"`
	Deliverables []struct {
		Path   string `json:"path"`
		Tier   string `json:"tier"`
		Class  string `json:"class"`
		Status string `json:"status"`
	} `json:"deliverables"`
}

// TestReviewManifestCoversDeliverables pins the manifest closed:
// status stays PENDING-REVIEW (Lane D never grants itself review),
// every deliverable source is listed exactly once as pending-review,
// and no listing dangles.
func TestReviewManifestCoversDeliverables(t *testing.T) {
	dir := hostDir(t)
	raw, err := os.ReadFile(filepath.Join(dir, "REVIEW-MANIFEST.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifest reviewManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if manifest.Status != "PENDING-REVIEW" {
		t.Fatalf("manifest status = %q, want PENDING-REVIEW", manifest.Status)
	}
	listed := make(map[string]string, len(manifest.Deliverables))
	for _, entry := range manifest.Deliverables {
		if _, dup := listed[entry.Path]; dup {
			t.Fatalf("manifest lists %q twice", entry.Path)
		}
		listed[entry.Path] = entry.Status
		if entry.Status != "pending-review" {
			t.Fatalf("manifest entry %q status = %q, want pending-review", entry.Path, entry.Status)
		}
		if entry.Tier == "" || entry.Class == "" {
			t.Fatalf("manifest entry %q lacks tier/class attribution", entry.Path)
		}
		if _, err := os.Stat(filepath.Join(dir, entry.Path)); err != nil {
			t.Fatalf("manifest entry %q dangles: %v", entry.Path, err)
		}
	}
	for _, tree := range []string{"adapters", "companions"} {
		base := filepath.Join(dir, tree)
		entries, err := os.ReadDir(base)
		if err != nil {
			t.Fatalf("read %s: %v", base, err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".ts") || strings.HasSuffix(entry.Name(), ".test.ts") {
				continue
			}
			rel := filepath.Join(tree, entry.Name())
			status, ok := listed[rel]
			if !ok {
				t.Fatalf("deliverable %q missing from manifest", rel)
			}
			if status != "pending-review" {
				t.Fatalf("deliverable %q status = %q, want pending-review", rel, status)
			}
		}
	}
}

// TestDeliverablesAvoidHostNamespaces pins the adapter audit shape:
// delivered modules touch the host only through their injected host
// parameters, never through ambient namespaces. Credential
// environment names travel as string literals (visible to this leg's
// marker list), never via process reads.
func TestDeliverablesAvoidHostNamespaces(t *testing.T) {
	dir := hostDir(t)
	raw, err := os.ReadFile(filepath.Join(dir, "REVIEW-MANIFEST.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifest reviewManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	for _, entry := range manifest.Deliverables {
		body, err := os.ReadFile(filepath.Join(dir, entry.Path))
		if err != nil {
			t.Fatalf("read %s: %v", entry.Path, err)
		}
		for _, ambient := range []string{"Bun.", "process.env", "Deno.", "require("} {
			if strings.Contains(string(body), ambient) {
				t.Fatalf("deliverable %q reaches ambient %q; hosts stay injected", entry.Path, ambient)
			}
		}
	}
}

var canlcOnce sync.Once
var canlcPath string
var canlcErr error

// canlcBinary builds the real compiler once per test run into a temp
// dir. The located-rejection legs drive it black-box
// (`inspect-project` needs no development bundle), so D02 pins the
// compiler's verdict without importing lane-owned internals.
func canlcBinary(t *testing.T) string {
	t.Helper()
	canlcOnce.Do(func() {
		dir, err := os.MkdirTemp("", "d02-canlc")
		if err != nil {
			canlcErr = err
			return
		}
		out := filepath.Join(dir, "canlc")
		cmd := exec.Command("go", "build", "-o", out, "./compiler")
		cmd.Dir = repoRootForBuild()
		if out, err := cmd.CombinedOutput(); err != nil {
			canlcErr = &buildError{err: err, output: string(out)}
			return
		}
		canlcPath = out
	})
	if canlcErr != nil {
		t.Fatalf("build canlc: %v", canlcErr)
	}
	return canlcPath
}

type buildError struct {
	err    error
	output string
}

func (e *buildError) Error() string { return e.err.Error() + ": " + e.output }

// TestUnadmittedCapabilitiesRejectWithLocation proves each host
// capability is genuinely unadmitted: a Can source naming it fails
// the real compiler with file-attributed location evidence naming
// the capability. No silent admission, no assumed rejection.
func TestUnadmittedCapabilitiesRejectWithLocation(t *testing.T) {
	binary := canlcBinary(t)
	for _, capability := range []string{"storage", "clipboard", "chart"} {
		t.Run(capability, func(t *testing.T) {
			root := t.TempDir()
			files := map[string]string{
				"can.project.json": `{"source_root":"src","error_registry":"can.errors.json"}`,
				"can.errors.json":  `{"active":[],"retired":[]}`,
				"src/main.can": "package app\n" +
					"    provides []\n" +
					"    uses [" + capability + "]\n" +
					"fn void main\n" +
					"    emits []\n" +
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
			cmd := exec.Command(binary, "inspect-project", root)
			combined, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatalf("capability %q admitted; inspect-project passed:\n%s", capability, combined)
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

func hostDir(t *testing.T) string {
	t.Helper()
	// This file lives in host/conformance/.
	abs, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	return filepath.Dir(abs)
}

func repoRoot(t *testing.T) string {
	t.Helper()
	return filepath.Dir(hostDir(t))
}

// repoRootForBuild resolves the repo root without test helpers for
// use inside sync.Once (t.TempDir and friends are test-scoped).
func repoRootForBuild() string {
	abs, err := filepath.Abs(".")
	if err != nil {
		return "."
	}
	return filepath.Dir(filepath.Dir(abs))
}
