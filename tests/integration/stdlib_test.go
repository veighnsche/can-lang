// Fresh staged builds for every maintained example project. Discovery
// walks std/ and examples/ for manifests, so a new project is covered
// automatically and a removed one fails the expected set below; no
// maintained example can be silently omitted from compilation. Each
// project asserts its real ordinary Can computation, builds twice
// with identical IDs, emits only into owned dist, and — for the pure
// std/current projects — runs to a clean exit.
package integration

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/distribution"
)

var expectedMaintained = []string{
	"examples/account-search",
	"examples/dashboard",
	"examples/files",
	"examples/form-validation",
	"examples/process",
	"examples/native-ai",
	"std/map/current",
	"std/ratio/current",
	"std/scalars/current",
	"std/text/current",
}

func discoverMaintained(t *testing.T, sourceRoot string) []string {
	t.Helper()
	var found []string
	for _, tree := range []string{"std", "examples"} {
		_ = filepath.WalkDir(filepath.Join(sourceRoot, tree), func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() || entry.Name() != "can.project.json" {
				return nil
			}
			rel, err := filepath.Rel(sourceRoot, filepath.Dir(path))
			if err != nil {
				t.Fatal(err)
			}
			found = append(found, rel)
			return nil
		})
	}
	sort.Strings(found)
	seen := map[string]bool{}
	for _, rel := range found {
		seen[rel] = true
	}
	for _, rel := range expectedMaintained {
		if !seen[rel] {
			t.Fatalf("maintained project %s not discovered", rel)
		}
	}
	if len(found) == 0 {
		t.Fatal("no maintained projects discovered")
	}
	return found
}

func stageProject(t *testing.T, sourceRoot, rel string) (root, home string) {
	t.Helper()
	var err error
	root, err = filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(filepath.Join(sourceRoot, rel))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() == "dist" {
			continue
		}
		src := filepath.Join(sourceRoot, rel, entry.Name())
		dst := filepath.Join(root, entry.Name())
		if entry.IsDir() {
			copyDir(t, src, dst)
			continue
		}
		data, err := os.ReadFile(src)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dst, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	home, err = filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return root, home
}

func copyFreshEmit(t *testing.T, dir, dst string) {
	t.Helper()
	if err := os.MkdirAll(dst, 0700); err != nil {
		t.Fatal(err)
	}
	copyDir(t, dir, dst)
}

func assertNoStrayEmit(t *testing.T, root, dir string) {
	t.Helper()
	owned := filepath.Join(root, "dist")
	rel, err := filepath.Rel(owned, dir)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		t.Fatalf("build directory %s escapes owned dist %s", dir, owned)
	}
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".ts") {
			return nil
		}
		if path == owned || strings.HasPrefix(path, owned+string(filepath.Separator)) {
			return nil
		}
		t.Fatalf("stray emit beside sources: %s", path)
		return nil
	})
}

func TestStdlibMaintained(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged example execution")
	}
	sourceRoot, _ := filepath.Abs("../..")
	projects := discoverMaintained(t, sourceRoot)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "stdlib-maintained")
	if err != nil {
		t.Fatal(err)
	}
	for _, rel := range projects {
		root, home := stageProject(t, sourceRoot, rel)
		status, out, diag := canlcOffline(t, ctx, bundle, home, "assert", root)
		if status != 0 || diag != "" {
			t.Fatalf("%s assert: %d %s %s", rel, status, out, diag)
		}
		var report struct {
			Passed     bool `json:"passed"`
			Assertions []struct {
				Evidence []string `json:"evidence"`
			} `json:"assertions"`
		}
		if err := json.Unmarshal([]byte(out), &report); err != nil || !report.Passed || len(report.Assertions) == 0 {
			t.Fatalf("%s invalid assert report %v %s", rel, err, out)
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
			t.Fatalf("%s asserts nothing real", rel)
		}
		firstID, firstDir := applicationBuild(t, ctx, bundle, home, root)
		secondID, _ := applicationBuild(t, ctx, bundle, home, root)
		if firstID != secondID {
			t.Fatalf("%s rebuild drifted: %s vs %s", rel, firstID, secondID)
		}
		assertNoStrayEmit(t, root, firstDir)
		if fresh := os.Getenv("CAN_FRESH_EMIT_DIR"); fresh != "" {
			copyFreshEmit(t, firstDir, filepath.Join(fresh, strings.ReplaceAll(rel, "/", "-")))
		}
		ran := "server/cli run covered by applications suite"
		if strings.HasPrefix(rel, "std/") {
			status, out, diag := canlcOffline(t, ctx, bundle, home, "run", root)
			if status != 0 || diag != "" {
				t.Fatalf("%s run: %d %s %s", rel, status, out, diag)
			}
			ran = "ran clean"
		}
		t.Logf("%s: %d assertions (%d real-can), build %s, %s", rel, len(report.Assertions), real, firstID[:12], ran)
	}
}
