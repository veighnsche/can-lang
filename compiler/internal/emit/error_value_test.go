package emit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/project"
)

const errorValueSource = `package app
    provides []
    uses []
error missing{str key}
record wrapper
    missing cause
fn wrapper store
    emits {}
    asserts
        sample: => ok wrapper(missing{"x"})
    ok wrapper(missing{"x"})
fn int load
    emits {missing}
    asserts
        sample: => missing{"x"}
    missing{"x"}
fn int guarded
    emits {missing}
    asserts
        sample: => ok 1
    match call load()
        missing
        ok int value => ok value
fn wrapper rebuilt
    emits {missing}
    asserts
        sample: => ok wrapper(missing{"k"})
    match call load()
        missing as failure => do
            missing fresh = missing{failure.key}
            ok wrapper(fresh)
        ok int value => ok wrapper(missing{"k"})
fn int quiet
    emits {}
    asserts
        sample: => ok 1
    match call load()
        missing => ok 0
        [_] as standard_failure failure => ok failure.occurrence_id
        ok int value => ok value
fn void main
    emits {}
    given
        str[] arguments
    asserts
        empty: [] => ok
    ok
`

func errorValueProgram(t *testing.T) *check.Program {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"can.project.json": `{"source_root":"src","error_registry":"can.errors.json"}`,
		"can.errors.json":  `{"active":["app::missing"],"retired":[]}`,
		"src/main.can":     errorValueSource,
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
	return program
}

// Terminal brace construction emits a domain failure with the checked
// payload identity and the terminal source span; every other position
// lowers constructors to frozen $canRecord data ($canSuccess, never
// $canFailure), and bare forwarding returns the prior completion
// unchanged without creating a new occurrence.
func TestBraceValueFailureLowering(t *testing.T) {
	program := errorValueProgram(t)
	assembly, err := assembleProgramBindings(program)
	if err != nil {
		t.Fatal(err)
	}
	bodies := map[string]string{}
	regions := map[string]*ir.Region{}
	for _, fn := range program.Functions {
		regions[fn.Symbol.Name] = fn.Region
		body, err := (&RegionEmitter{
			Bindings:      assembly.bindings,
			Functions:     assembly.functions,
			DomainRuntime: "$canDomain",
			SourceID:      fn.Symbol.Source.ID,
		}).Function("$"+fn.Symbol.Name, fn.Region)
		if err != nil {
			t.Fatal(err)
		}
		bodies[fn.Symbol.Name] = body
	}

	terminal := regions["load"].Body.Terminal
	if terminal.Kind != ir.DomainCompletion {
		t.Fatalf("load terminal is %s", terminal.Kind)
	}
	load := bodies["load"]
	// The emitted payload identity is the checked error type identity.
	wantCreate := `$canDomain.create("` + terminal.Value.Type.Identity() + `"`
	if !strings.Contains(load, "return $canFailure("+wantCreate) {
		t.Fatalf("terminal misses checked payload creation:\n%s", load)
	}
	// The origin span is the terminal constructor source.
	origin := regions["load"].Body.Terminal.Span
	if got := errorValueSource[origin.Start:origin.End]; got != `missing{"x"}` {
		t.Fatalf("terminal origin span holds %q", got)
	}
	_, mappings, err := extractMappings(load)
	if err != nil {
		t.Fatal(err)
	}
	domain := false
	for _, mapping := range mappings {
		if mapping.Operation != "domain" {
			continue
		}
		if got := errorValueSource[mapping.Start:mapping.End]; got != `missing{"x"}` {
			t.Fatalf("domain mark spans %q", got)
		}
		domain = true
	}
	if !domain {
		t.Fatalf("terminal lost its domain source mark:\n%s", load)
	}

	// Stored, reconstructed, and standard-handled positions stay data.
	for _, name := range []string{"store", "rebuilt", "quiet"} {
		body := bodies[name]
		if !strings.Contains(body, "$canSuccess(") {
			t.Fatalf("%s lost its success completion:\n%s", name, body)
		}
		if strings.Contains(body, "$canFailure(") {
			t.Fatalf("%s emits a failure from data:\n%s", name, body)
		}
	}
	for _, name := range []string{"store", "rebuilt"} {
		if !strings.Contains(bodies[name], "$canRecord(") {
			t.Fatalf("%s lost its frozen record data:\n%s", name, bodies[name])
		}
	}

	// Forwarding returns the matched completion as-is: no new creation,
	// so the prior occurrence (including its occurrence id) survives.
	guarded := bodies["guarded"]
	if !strings.Contains(guarded, "as $canCompletion<") {
		t.Fatalf("forwarding lost its passthrough return:\n%s", guarded)
	}
	if got := strings.Count(guarded, ".create("); got != 0 {
		t.Fatalf("forwarding created %d new occurrences:\n%s", got, guarded)
	}
}
