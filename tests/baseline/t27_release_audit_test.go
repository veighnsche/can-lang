// T27 final release audit checks.
//
// The audit (docs/syntax-taste/t27-release-audit-2026-09-24.md)
// records every DI disposition, maps accepted items to tests/docs,
// proves no deferred mechanism slipped in, links the Gate 1-5
// evidence, and scopes the recommendation to the platforms and
// browsers that passed. These tests pin each of those claims to
// the files that prove them.
package baseline

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const t27Audit = "docs/syntax-taste/t27-release-audit-2026-09-24.md"

func readRepoFile(t *testing.T, root, name string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(raw)
}

// Every DI inventory row must carry an explicit disposition in the
// audit. DI-05 splits into 05a/05b, DI-09 into 09/09b, DI-14 into
// 14/14a; the four identifiers with no committed description must
// still appear as unattested, never silently dropped.
func TestT27DIInventoryComplete(t *testing.T) {
	root := repoRoot(t)
	text := readRepoFile(t, root, t27Audit)
	ids := []string{
		"DI-01", "DI-02", "DI-03", "DI-04", "DI-05a", "DI-05b",
		"DI-06", "DI-07", "DI-08", "DI-09", "DI-09b", "DI-10",
		"DI-11", "DI-12", "DI-13", "DI-14", "DI-14a", "DI-15",
		"DI-16", "DI-17", "DI-18", "DI-19", "DI-20", "DI-21",
		"DI-22", "DI-23",
	}
	dispositions := []string{"accepted", "retained", "deferred", "unselected", "unattested"}
	for _, id := range ids {
		mentioned, disposed := false, false
		for line := range strings.Lines(text) {
			i := strings.Index(line, id)
			if i < 0 {
				continue
			}
			// Do not let DI-09 match the DI-09b row or DI-14 the DI-14a row.
			if end := i + len(id); end < len(line) {
				if c := line[end]; c >= 'a' && c <= 'z' || c >= '0' && c <= '9' {
					continue
				}
			}
			mentioned = true
			lower := strings.ToLower(line)
			for _, d := range dispositions {
				if strings.Contains(lower, d) {
					disposed = true
					break
				}
			}
			if disposed {
				break
			}
		}
		switch {
		case !mentioned:
			t.Errorf("audit %s omits DI inventory row %s", t27Audit, id)
		case !disposed:
			t.Errorf("audit mentions %s but no line carries a disposition", id)
		}
	}
}

// Every accepted or retained item must map to a proving test and a
// proving doc or implementation anchor, and each anchor must still
// contain its mechanism marker.
func TestT27AcceptedItemsMapToTestsAndDocs(t *testing.T) {
	root := repoRoot(t)
	audit := readRepoFile(t, root, t27Audit)
	if !strings.Contains(audit, "TestPatternBindCapturesAtEveryDepth") {
		t.Errorf("audit does not map accepted items to tests")
	}
	anchors := map[string][]string{
		"compiler/internal/syntax/ast.go": {
			"type BindPattern struct",
			"Dependency *Token",
			"Scenario *Token",
		},
		"compiler/internal/check/pattern_bind_test.go": {
			"func TestPatternBindCapturesAtEveryDepth",
			"func TestPatternBindDuplicatesAndAlternatives",
		},
		"compiler/internal/syntax/pattern_bind_test.go": {
			"func TestBindPatternNodes",
		},
		"compiler/internal/check/owner_record_test.go": {
			"func TestOwnerRecordBoundaryPositives",
			"owner record email",
		},
		"compiler/internal/check/exported_generics_test.go": {
			"func TestExportedGenericStructuralComposites",
		},
		"compiler/internal/check/exported_generics.go": {
			"package check",
		},
		"compiler/internal/resolve/instances_test.go": {
			"func TestQualifiedImportsRejectUndeclaredAndPrivate",
		},
		"compiler/internal/project/error_identity_test.go": {
			"func TestFormerNumericCollisionComposesWithQualifiedIdentity",
		},
		"compiler/internal/project/lock.go": {
			`ReportIdentityVersion = "can.error.v2"`,
		},
		"compiler/internal/check/scenario_test.go": {
			"func TestScenarioLinkResolves",
		},
		"compiler/internal/types/types_test.go": {
			"func TestGenericVariantLeafSetCompatibility",
		},
		"compiler/internal/types/compatibility.go": {
			"extensional named leaf sets",
		},
		"docs/syntax-taste/decisions.md": {
			"(DI-06)",
		},
		"docs/syntax-taste/t26-product-guide-2026-09-24.md": {
			"owner-only",
			"action-routes",
			"lines_order",
		},
		"docs/implementation/shutdown.md": {
			"exactly once",
		},
	}
	for file, phrases := range anchors {
		raw := readRepoFile(t, root, file)
		for _, phrase := range phrases {
			if !strings.Contains(raw, phrase) {
				t.Errorf("anchor %s no longer contains %q", file, phrase)
			}
		}
	}
	for _, suite := range []string{
		"runtime/test/action-routes.test.ts",
		"runtime/test/form-rows.test.ts",
		"runtime/test/action-json.test.ts",
		"runtime/test/shutdown.test.ts",
	} {
		info, err := os.Stat(filepath.Join(root, filepath.FromSlash(suite)))
		if err != nil || info.Size() == 0 {
			t.Errorf("runtime suite %s missing or empty: %v", suite, err)
		}
	}
}

// No deferred mechanism may have slipped into the implementation.
// Each leg below is a structural check with a pinned rejection or
// absence anchor, not a documentation claim.
func TestT27NoDeferredMechanismPresent(t *testing.T) {
	root := repoRoot(t)

	// Numbered error declarations: none admitted outside the two
	// pinned rejection tests. No leading word boundary: one
	// rejection fixture embeds the spelling after a literal \n escape.
	numbered := regexp.MustCompile(`error\s+[0-9]`)
	allowedNumbered := map[string]bool{
		"compiler/internal/syntax/declarations_test.go": true,
		"compiler/internal/check/checks_test.go":        true,
	}
	var numberedHits []string
	for _, dir := range []string{"compiler", "examples", "std", "tests"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			if !strings.HasSuffix(path, ".can") && !strings.HasSuffix(path, ".go") {
				return nil
			}
			if strings.HasSuffix(path, "_test.go") && !allowedNumbered[relSlash(root, path)] {
				return nil
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if numbered.Match(raw) {
				numberedHits = append(numberedHits, relSlash(root, path))
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", dir, err)
		}
	}
	for _, hit := range numberedHits {
		if !allowedNumbered[hit] {
			t.Errorf("numbered error declaration admitted outside rejection tests: %s", hit)
		}
	}
	if len(numberedHits) != len(allowedNumbered) {
		t.Errorf("numbered-error scan hits = %v, want exactly the %d rejection tests", numberedHits, len(allowedNumbered))
	}

	// DI-14a: the SQL checker pins the RETURNING rejection.
	if raw := readRepoFile(t, root, "compiler/internal/sql/cardinality.go"); !strings.Contains(raw, "RETURNING is not admitted") {
		t.Error("DI-14a rejection anchor lost: cardinality.go no longer reports that RETURNING is not admitted")
	}

	// DI-18 additions: no finally keyword in non-test syntax sources.
	var finallyHits []string
	syntaxFiles, err := filepath.Glob(filepath.Join(root, "compiler", "internal", "syntax", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range syntaxFiles {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if regexp.MustCompile(`\bfinally\b`).Match(raw) {
			finallyHits = append(finallyHits, relSlash(root, path))
		}
	}
	if len(finallyHits) > 0 {
		t.Errorf("finally spelling reached the syntax package: %v", finallyHits)
	}

	// DI-10: no declarative-markup AST node.
	if raw := readRepoFile(t, root, "compiler/internal/syntax/ast.go"); strings.Contains(raw, "Markup") {
		t.Error("Markup node reached the syntax AST")
	}

	// DI-23: the parser rejection suite pins the triple-quote rejection.
	if raw := readRepoFile(t, root, "compiler/internal/syntax/declarations_test.go"); !strings.Contains(raw, "TestFileParserRejectsObsoleteAndMalformedGrammar") || !strings.Contains(raw, `\"\"\"a\nb\"\"\"`) {
		t.Error("DI-23 anchor lost: triple-quote rejection no longer pinned")
	}

	// DI-10/DI-20/DI-21: no bulk, billing, calendar or markup
	// surface in catalogue names or identities.
	var catalogue struct {
		Packages   []namedEntry `json:"packages"`
		Types      []namedEntry `json:"types"`
		Errors     []namedEntry `json:"errors"`
		Operations []namedEntry `json:"operations"`
	}
	loadJSON(t, filepath.Join(root, "compiler", "internal", "catalogue", "catalogue.json"), &catalogue)
	deferredName := regexp.MustCompile(`(?i)bulk|billing|calendar|markup`)
	for _, group := range []struct {
		kind    string
		entries []namedEntry
	}{
		{"packages", catalogue.Packages},
		{"types", catalogue.Types},
		{"errors", catalogue.Errors},
		{"operations", catalogue.Operations},
	} {
		for _, e := range group.entries {
			if deferredName.MatchString(e.Name) || deferredName.MatchString(e.Identity) {
				t.Errorf("deferred surface in catalogue %s: name=%q identity=%q", group.kind, e.Name, e.Identity)
			}
		}
	}
	for _, pkg := range []string{"billing", "calendar"} {
		if _, err := os.Stat(filepath.Join(root, "std", pkg)); !os.IsNotExist(err) {
			t.Errorf("std/%s exists; DI-21 is deferred to a later library", pkg)
		}
	}

	// DI-17: the native-AI candidate slot still records the deferral.
	for _, name := range []string{"README.md", "candidate.json"} {
		raw := readRepoFile(t, root, "tests/baseline/candidates/T26-native-ai/"+name)
		if !strings.Contains(strings.ToLower(raw), "deferr") {
			t.Errorf("DI-17 candidate slot %s no longer records the deferral", name)
		}
	}
}

type namedEntry struct {
	Name     string `json:"name"`
	Identity string `json:"identity"`
}

func relSlash(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}

// Every gate must link live evidence: the test files exist and the
// audit names each one.
func TestT27GateEvidenceLinked(t *testing.T) {
	root := repoRoot(t)
	audit := readRepoFile(t, root, t27Audit)
	evidence := []string{
		"compiler/internal/check/core_integration_test.go",
		"compiler/internal/emit/core_integration_test.go",
		"compiler/internal/syntax/core_integration_test.go",
		"tests/integration/gate3_matrix_test.go",
		"tests/integration/gate4_fault_test.go",
		"tests/integration/gate5_frontend_test.go",
		"tests/integration/browser/grid.mjs",
		"tests/integration/browser/build-grid.mjs",
		"tests/integration/browser/sha256-shim.mjs",
		"tests/baseline/baseline.json",
		"tests/baseline/registry.json",
		"docs/syntax-taste/jev-consultations/findings.md",
	}
	for _, file := range evidence {
		info, err := os.Stat(filepath.Join(root, filepath.FromSlash(file)))
		if err != nil || info.Size() == 0 {
			t.Errorf("gate evidence %s missing or empty: %v", file, err)
			continue
		}
		if !strings.Contains(audit, file) {
			t.Errorf("audit does not link gate evidence %s", file)
		}
	}
}

// Generated artifacts must be fresh: the catalogue source of truth
// regenerates byte-identical outputs.
func TestT27GeneratedArtifactsFresh(t *testing.T) {
	root := repoRoot(t)
	cmd := exec.Command("go", "run", "./compiler/internal/catalogue/cmd/cataloguegen", "--check")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("cataloguegen --check: %v\n%s", err, out)
	}
	for _, gen := range []string{
		"compiler/internal/catalogue/generated.go",
		"runtime/catalogue.ts",
	} {
		raw := readRepoFile(t, root, gen)
		if !strings.Contains(raw, "DO NOT EDIT") {
			t.Errorf("generated file %s lost its generated marker", gen)
		}
	}
}

// The recommendation must name only platforms and browsers that
// passed, with the Gate 4 Linux caveat and the firefox exclusion
// stated, never implied.
func TestT27RecommendationScoped(t *testing.T) {
	root := repoRoot(t)
	audit := readRepoFile(t, root, t27Audit)
	lower := strings.ToLower(audit)
	for _, phrase := range []string{
		"bun-1.4.2-darwin-arm64-v1",
		"chromium 140",
		"webkit 26",
		"firefox is not qualified",
		"native (non-emulated) linux/amd64",
		"no reload durability",
		"no automatic queue",
		"postgresql 17.11",
	} {
		if !strings.Contains(lower, phrase) {
			t.Errorf("audit recommendation omits required scope phrase %q", phrase)
		}
	}
	if strings.Contains(lower, "firefox — qualified") ||
		strings.Contains(lower, "firefox qualified") {
		t.Error("audit must not qualify firefox")
	}
}
