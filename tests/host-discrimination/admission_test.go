// Package hostdiscrimination is the Lane D (host integration) X-R01-1
// discrimination harness: catalogue-vs-adapter-vs-companion tier
// assignment for two missing operations and one required widget class.
// The comparison legs are bun tests over isolated prototypes; this file
// owns the no-self-admission legs: no prototype capability is callable
// from Can, no prototype is referenced from any owned tree, and every
// prototype file is listed as unreviewed in REVIEW-MANIFEST.json.
package hostdiscrimination

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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
}

func TestPrototypeCapabilitiesUnadmitted(t *testing.T) {
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
				t.Fatalf("prototype capability %q admitted; self-admission path exists", operation.Identity)
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

// TestPrototypesUnreferenced walks every tree another lane owns and fails
// if a D01 prototype module path leaks into it. Prototypes may import
// from owned trees (F01 C-G policy); the reverse is self-admission.
func TestPrototypesUnreferenced(t *testing.T) {
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
		entries, err := os.ReadDir(base)
		if err != nil {
			t.Fatalf("read %s: %v", base, err)
		}
		_ = entries
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
			if strings.Contains(string(raw), "host-discrimination") {
				violations = append(violations, path)
			}
			return nil
		}); err != nil {
			t.Fatalf("walk %s: %v", base, err)
		}
	}
	if len(violations) > 0 {
		t.Fatalf("prototype referenced from owned trees: %v", violations)
	}
}

type reviewManifest struct {
	Status     string `json:"status"`
	Prototypes []struct {
		Path   string `json:"path"`
		Status string `json:"status"`
	} `json:"prototypes"`
}

// TestReviewManifestCoversPrototypes pins the manifest closed: status
// stays UNREVIEWED-PROTOTYPE, every prototype source is listed exactly
// once as unreviewed, and no listing dangles.
func TestReviewManifestCoversPrototypes(t *testing.T) {
	dir := testDir(t)
	raw, err := os.ReadFile(filepath.Join(dir, "REVIEW-MANIFEST.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifest reviewManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if manifest.Status != "UNREVIEWED-PROTOTYPE" {
		t.Fatalf("manifest status = %q, want UNREVIEWED-PROTOTYPE", manifest.Status)
	}
	listed := make(map[string]string, len(manifest.Prototypes))
	for _, entry := range manifest.Prototypes {
		if _, dup := listed[entry.Path]; dup {
			t.Fatalf("manifest lists %q twice", entry.Path)
		}
		listed[entry.Path] = entry.Status
		if entry.Status != "unreviewed" {
			t.Fatalf("manifest entry %q status = %q, want unreviewed", entry.Path, entry.Status)
		}
		if _, err := os.Stat(filepath.Join(dir, entry.Path)); err != nil {
			t.Fatalf("manifest entry %q dangles: %v", entry.Path, err)
		}
	}
	for _, tree := range []string{"adapters", "companions"} {
		base := filepath.Join(dir, tree)
		entries, err := os.ReadDir(base)
		if os.IsNotExist(err) {
			continue
		}
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
				t.Fatalf("prototype %q missing from manifest", rel)
			}
			if status != "unreviewed" {
				t.Fatalf("prototype %q status = %q, want unreviewed", rel, status)
			}
		}
	}
}

func testDir(t *testing.T) string {
	t.Helper()
	// This file lives in tests/host-discrimination/.
	return "."
}

func repoRoot(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs(testDir(t))
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	return filepath.Dir(filepath.Dir(abs))
}
