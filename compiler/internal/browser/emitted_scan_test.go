package browser_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/browser"
	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/emit"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/project"
)

// TestEmittedBrowserBodiesScanClean proves compiler-emitted browser modules
// satisfy the structural audit surface: every body lexes, carries no host
// operation, and declares exactly the edges its import statements name.
// Runtime dependencies are auto-stubbed: any missing module the emitter
// names is added until emission succeeds, so the test tracks emission
// without pinning its import list.
func TestEmittedBrowserBodiesScanClean(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"can.project.json": `{"source_root":"src","error_registry":"can.errors.json"}`,
		"can.errors.json":  `{"active":[],"retired":[]}`,
		"src/main.can": `package app
    provides []
    uses [text]
fn str greet
    emits []
    given
        str name
    asserts
        sample: "a" => ok "a"
    ok name
fn void main
    emits []
    given
        str[] arguments
    asserts
        empty: [] => ok
    match call greet("a")
        ok str text => ok
`,
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
	graph, err := project.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	program, err := check.CheckProgram(graph)
	if err != nil {
		t.Fatal(err)
	}
	var dependencies []ir.Artifact
	var artifacts []ir.Artifact
	for range 100 {
		artifacts, err = emit.BrowserModules(program, "runtime", dependencies)
		if err == nil {
			break
		}
		missing, ok := strings.CutPrefix(err.Error(), "missing emitted module ")
		if !ok {
			t.Fatalf("browser emission failed: %v", err)
		}
		dependencies = append(dependencies, ir.Artifact{
			Path: missing, Bytes: []byte("export const $canStub = 1;\n"), Runtime: true,
		})
	}
	if err != nil {
		t.Fatalf("browser emission failed after stubbing: %v", err)
	}
	generated := 0
	for _, artifact := range artifacts {
		if artifact.Runtime {
			continue
		}
		generated++
		scan, err := browser.ScanModule(artifact.Bytes)
		if err != nil {
			t.Fatalf("%s does not lex: %v", artifact.Path, err)
		}
		if len(scan.Findings) != 0 {
			t.Fatalf("%s findings: %+v", artifact.Path, scan.Findings)
		}
		lexed := map[string]bool{}
		for _, edge := range scan.Edges {
			lexed[edge.Specifier] = true
		}
		if len(lexed) != len(artifact.Imports) {
			t.Fatalf("%s lexed %v against declared %v", artifact.Path, lexed, artifact.Imports)
		}
		for _, spec := range artifact.Imports {
			if !lexed[spec] {
				t.Fatalf("%s declares edge %q with no matching import", artifact.Path, spec)
			}
		}
	}
	if generated == 0 {
		t.Fatal("browser emission produced no generated modules")
	}
	t.Logf("scanned %d generated modules against %d stubbed runtime modules", generated, len(dependencies))
}
