package check

import (
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// Brace constructors build ordinary frozen nominal data; only a terminal
// completion emits a domain failure. Forwarding passes the prior occurrence
// through unchanged, reconstruction builds a new value, and standard
// failures stay outside finite bounds.
func TestBraceValueVersusEmittedFailure(t *testing.T) {
	const main = `package app
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
	registry := `{"active":["app::missing"],"retired":[]}`
	program, err := programFixtureRegistry(t, map[string]string{"src/main.can": main}, registry)
	if err != nil {
		t.Fatalf("value/failure program rejected: %v", err)
	}
	byName := map[string]*ProgramFunction{}
	for _, fn := range program.Functions {
		byName[fn.Symbol.Name] = fn
	}
	region := func(name string) *ir.Region {
		t.Helper()
		fn := byName[name]
		if fn == nil {
			t.Fatalf("missing %s region", name)
		}
		return fn.Region
	}

	// Nested construction stays frozen data: a success completion whose
	// record value embeds the error-typed record input.
	stored := region("store").Body.Terminal
	if stored.Kind != ir.SuccessCompletion {
		t.Fatalf("stored terminal is %s", stored.Kind)
	}
	if stored.Value.Kind != ir.Record || stored.Value.Type.Kind() != types.Record {
		t.Fatalf("stored value is not record data: %+v", stored.Value)
	}
	if len(stored.Value.Inputs) != 1 || stored.Value.Inputs[0].Type.Kind() != types.Error {
		t.Fatalf("stored input is not error data: %+v", stored.Value.Inputs)
	}

	// A terminal constructor emits with the declared bound and keeps its
	// source span.
	terminal := region("load").Body.Terminal
	if terminal.Kind != ir.DomainCompletion {
		t.Fatalf("terminal kind is %s", terminal.Kind)
	}
	if terminal.Value.Type.Kind() != types.Error || types.CanonicalName(terminal.Value.Type) != "app::missing" {
		t.Fatalf("terminal type is %s", types.CanonicalName(terminal.Value.Type))
	}
	if got := main[terminal.Value.Span.Start:terminal.Value.Span.End]; got != `missing{"x"}` {
		t.Fatalf("terminal span holds %q", got)
	}
	bound := false
	for _, wall := range region("load").Errors {
		if types.CanonicalName(wall) == "app::missing" {
			bound = true
		}
	}
	if !bound {
		t.Fatal("load region lost its declared missing bound")
	}
	if len(region("quiet").Errors) != 0 {
		t.Fatalf("empty bound admits errors: %v", region("quiet").Errors)
	}

	// Bare forwarding keeps the prior occurrence: no body, no new value.
	forward := region("guarded").Body.Terminal
	if forward.Kind != ir.MatchCompletion {
		t.Fatalf("guarded terminal is %s", forward.Kind)
	}
	arm := forward.Match.Arms[0]
	if !arm.Forward || arm.Body != nil || arm.Value != nil {
		t.Fatalf("forwarding arm rebuilt the occurrence: %+v", arm)
	}
	if arm.Error == nil || types.CanonicalName(arm.Error) != "app::missing" {
		t.Fatalf("forwarding arm lost its error: %+v", arm.Error)
	}

	// Reconstruction binds a fresh ordinary value before succeeding.
	rebuilt := region("rebuilt").Body.Terminal
	if rebuilt.Kind != ir.MatchCompletion {
		t.Fatalf("rebuilt terminal is %s", rebuilt.Kind)
	}
	var fresh *ir.Expression
	for _, candidate := range rebuilt.Match.Arms[0].Body.Block.Steps {
		if candidate.Local != nil && strings.HasSuffix(candidate.Local.Identity, "/fresh") {
			fresh = candidate.Value
		}
	}
	if fresh == nil || fresh.Kind != ir.Record || fresh.Type.Kind() != types.Error {
		t.Fatalf("reconstruction is not fresh error data: %+v", fresh)
	}
	if rebuilt.Match.Arms[0].Body.Block.Terminal.Kind != ir.SuccessCompletion {
		t.Fatalf("rebuilt arm terminal is %s", rebuilt.Match.Arms[0].Body.Block.Terminal.Kind)
	}

	// The standard arm coexists with an empty finite bound.
	quiet := region("quiet").Body.Terminal
	if quiet.Kind != ir.MatchCompletion {
		t.Fatalf("quiet terminal is %s", quiet.Kind)
	}
	standard := false
	for _, candidate := range quiet.Match.Arms {
		if candidate.Outcome == "standard" {
			standard = true
		}
	}
	if !standard {
		t.Fatalf("quiet lost its standard arm: %+v", quiet.Match.Arms)
	}
}

func TestBraceFailureBoundNegatives(t *testing.T) {
	const main = `package app
    provides []
    uses []
error missing{str key}
fn int load
    emits {}
    asserts
        sample: => ok 1
    missing{"x"}
fn void main
    emits {}
    given
        str[] arguments
    asserts
        empty: [] => ok
    ok
`
	registry := `{"active":["app::missing"],"retired":[]}`
	_, err := programFixtureRegistry(t, map[string]string{"src/main.can": main}, registry)
	if err == nil {
		t.Fatal("domain terminal admitted under an empty bound")
	}
	if !strings.Contains(err.Error(), "undeclared escaping domain error") {
		t.Fatalf("escape diagnostic changed: %v", err)
	}
	located, ok := source.AsLocated(err)
	if !ok || located.Code != "CAN-CHECK-OUTWARD-ERROR" {
		t.Fatalf("escape lost code or span: %v", err)
	}
	if got := main[located.Span.Start:located.Span.End]; got != `missing{"x"}` {
		t.Fatalf("escape span holds %q", got)
	}

	standard := strings.Replace(main, "    emits {}\n    asserts\n        sample: => ok 1\n    missing{\"x\"}", "    emits {standard_failure}\n    asserts\n        sample: => ok 1\n    ok 1", 1)
	if standard == main {
		t.Fatal("standard-bound mutation matched nothing")
	}
	if _, err := programFixtureRegistry(t, map[string]string{"src/main.can": standard}, registry); err == nil {
		t.Fatal("finite bound admitted standard_failure")
	}
}
