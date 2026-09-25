package check

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/project"
)

func browserTargetFixture(t *testing.T, files map[string]string, target Target) (*Program, error) {
	t.Helper()
	root := t.TempDir()
	files["can.project.json"] = `{"source_root":"src","error_registry":"can.errors.json"}`
	files["can.errors.json"] = `{"active":[],"retired":[]}`
	for name, text := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	graph, err := project.Load(root)
	if err != nil {
		return nil, err
	}
	if target == TargetBrowser {
		return CheckBrowserProgram(graph)
	}
	return CheckProgram(graph)
}

const browserTargetHeader = "package app\n    provides []\n    uses []\n"

const browserZeroMain = browserTargetHeader + `fn void main
    emits []
    asserts
        empty: => ok
    ok
`

const bunArgsMain = browserTargetHeader + `fn void main
    emits []
    given
        str[] arguments
    asserts
        empty: [] => ok
    ok
`

func TestBrowserTargetAcceptsZeroArgMain(t *testing.T) {
	program, err := browserTargetFixture(t, map[string]string{"src/main.can": browserZeroMain}, TargetBrowser)
	if err != nil {
		t.Fatalf("browser zero-arg main rejected: %v", err)
	}
	if program.Target != TargetBrowser {
		t.Fatalf("browser program target = %q, want browser", string(program.Target))
	}
	if program.Entry == nil || program.Entry.Region == nil {
		t.Fatal("browser program lacks a checked entry region")
	}
	if len(program.Entry.Region.Inputs) != 0 {
		t.Fatalf("browser entry has %d inputs, want 0", len(program.Entry.Region.Inputs))
	}
}

func TestBrowserTargetRejectsBunMain(t *testing.T) {
	_, err := browserTargetFixture(t, map[string]string{"src/main.can": bunArgsMain}, TargetBrowser)
	if err == nil {
		t.Fatal("browser target admitted main(str[] args)")
	}
	if !strings.Contains(err.Error(), "browser entry must be") {
		t.Fatalf("browser diagnostic omits browser expectation: %v", err)
	}
}

func TestBunTargetRejectsZeroArgMain(t *testing.T) {
	_, err := browserTargetFixture(t, map[string]string{"src/main.can": browserZeroMain}, TargetBun)
	if err == nil {
		t.Fatal("bun target admitted zero-argument main")
	}
	if !strings.Contains(err.Error(), "void main(str[] args)") {
		t.Fatalf("bun diagnostic omits bun expectation: %v", err)
	}
}

func TestBrowserTargetRejectsEmittingMain(t *testing.T) {
	source := browserTargetHeader + `fn void main
    emits [checks::failed]
    asserts
        empty: => ok
    ok
`
	// checks is not imported; use a catalogue error via uses.
	source = "package app\n    provides []\n    uses [checks]\n" + `fn void main
    emits [checks::failed]
    asserts
        empty: => ok
    ok
`
	_, err := browserTargetFixture(t, map[string]string{"src/main.can": source}, TargetBrowser)
	if err == nil {
		t.Fatal("browser target admitted emitting main")
	}
	if !strings.Contains(err.Error(), "browser entry must be") {
		t.Fatalf("browser emitting-main diagnostic omits expectation: %v", err)
	}
}

func TestBrowserTargetRequiresEntry(t *testing.T) {
	_, err := browserTargetFixture(t, map[string]string{"src/main.can": browserTargetHeader}, TargetBrowser)
	if err == nil {
		t.Fatal("browser target admitted missing main")
	}
	if !strings.Contains(err.Error(), "target browser") {
		t.Fatalf("browser missing-entry diagnostic omits target: %v", err)
	}
}

func TestCheckProgramForTargetRejectsUnknown(t *testing.T) {
	root := t.TempDir()
	for name, text := range map[string]string{
		"can.project.json": `{"source_root":"src","error_registry":"can.errors.json"}`,
		"can.errors.json":  `{"active":[],"retired":[]}`,
		"src/main.can":     browserZeroMain,
	} {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	graph, err := project.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CheckProgramForTarget(graph, Target("worker")); err == nil {
		t.Fatal("unknown check target admitted")
	}
}
