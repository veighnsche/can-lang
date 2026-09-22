package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
)

func writeTestProject(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	write := func(name, text string) {
		t.Helper()
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("can.project.json", `{"source_root":"src","error_registry":"can.errors.json"}`)
	write("can.errors.json", `{"active":[],"retired":[]}`)
	for name, text := range files {
		write(name, text)
	}
	return root
}

const overlayMain = "package app\n    provides [main]\n    uses []\n\nfn int main\n    emits []\n    given\n        int seed\n    asserts\n        sample: 1 => ok 1\n    ok seed\n"

func TestOverlaySubstitutesBeforeParse(t *testing.T) {
	root := writeTestProject(t, map[string]string{"src/main.can": overlayMain})
	real, err := filepath.EvalSymlinks(filepath.Join(root, "src/main.can"))
	if err != nil {
		t.Fatal(err)
	}
	overlay := NewOverlay()
	broken := "package app\n    provides [main]\n    uses []\n\nfn int main(\n    emits []\n    ok 1\n"
	if err := overlay.Set(real, 7, broken); err != nil {
		t.Fatal(err)
	}
	if entry, ok := overlay.Get(real); !ok || entry.Version != 7 || entry.Text != broken {
		t.Fatal("overlay did not retain the versioned buffer")
	}
	_, err = LoadWithOverlay(root, overlay)
	sourceErr, ok := err.(*SourceError)
	if !ok {
		t.Fatalf("expected a source error, got %T %v", err, err)
	}
	if sourceErr.Path != real || sourceErr.File == nil || len(sourceErr.Diagnostics) == 0 {
		t.Fatalf("source error lost structure: %+v", sourceErr)
	}
	if sourceErr.Diagnostics[0].Code == "" || sourceErr.Diagnostics[0].Span.Start < 0 {
		t.Fatalf("source error lost code/span: %+v", sourceErr.Diagnostics[0])
	}
	// The same bytes saved to disk fail the CLI load identically.
	if err := os.WriteFile(real, []byte(broken), 0644); err != nil {
		t.Fatal(err)
	}
	_, diskErr := Load(root)
	if diskErr == nil || diskErr.Error() != sourceErr.Error() {
		t.Fatalf("overlay diagnosis %q differs from disk diagnosis %q", sourceErr.Error(), diskErr)
	}
	// Clearing the overlay restores the on-disk diagnosis (still broken
	// here, so restore the good file first).
	if err := os.WriteFile(real, []byte(overlayMain), 0644); err != nil {
		t.Fatal(err)
	}
	if err := overlay.Clear(real); err != nil {
		t.Fatal(err)
	}
	graph, err := LoadWithOverlay(root, overlay)
	if err != nil {
		t.Fatal(err)
	}
	if len(graph.Root.Sources) != 1 || string(graph.Root.Sources[0].Bytes) != overlayMain {
		t.Fatal("cleared overlay did not restore disk bytes")
	}
}

func TestOverlayIgnoresUnwalkedPaths(t *testing.T) {
	root := writeTestProject(t, map[string]string{"src/main.can": overlayMain})
	overlay := NewOverlay()
	ghost := filepath.Join(root, "src", "ghost.can")
	if err := overlay.Set(ghost, 1, "package app\n"); err != nil {
		t.Fatal(err)
	}
	graph, err := LoadWithOverlay(root, overlay)
	if err != nil {
		t.Fatal(err)
	}
	if len(graph.Root.Sources) != 1 {
		t.Fatalf("overlay invented a source: %d", len(graph.Root.Sources))
	}
}

func TestOverlayRefusals(t *testing.T) {
	overlay := NewOverlay()
	if err := overlay.Set("relative/path.can", 1, "x"); err == nil {
		t.Fatal("accepted a relative path")
	}
	abs := filepath.Join(t.TempDir(), "notes.txt")
	if err := overlay.Set(abs, 1, "x"); err == nil || !strings.Contains(err.Error(), "Can sources") {
		t.Fatalf("accepted a non-source path: %v", err)
	}
	if err := overlay.Clear("relative/path.can"); err == nil {
		t.Fatal("cleared a relative path")
	}
	if _, ok := overlay.Get("relative/path.can"); ok {
		t.Fatal("found a relative path")
	}
}

func TestSourceErrorPreservesCLIMessage(t *testing.T) {
	real := filepath.Join(writeTestProject(t, map[string]string{"src/main.can": "package app\n\nfn broken( -> int\n    ok 1\n"}), "src", "main.can")
	_, err := Load(filepath.Dir(filepath.Dir(real)))
	sourceErr, ok := err.(*SourceError)
	if !ok {
		t.Fatalf("plain load lost the typed error: %T %v", err, err)
	}
	data, err := os.ReadFile(real)
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := filepath.EvalSymlinks(real)
	if err != nil {
		t.Fatal(err)
	}
	file, err := source.New(resolved, string(data))
	if err != nil {
		t.Fatal(err)
	}
	parsed := syntax.Parse(file)
	if parsed.OK() || parsed.Diagnostics[0].Format(file) != sourceErr.Message {
		t.Fatal("typed error message differs from the historical format")
	}
}
