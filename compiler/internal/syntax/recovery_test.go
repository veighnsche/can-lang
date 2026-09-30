package syntax

import (
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"strings"
	"testing"
)

func TestRecoveryRetainsPartialFunctionAndCallContext(t *testing.T) {
	text := `package app
    provides []
    uses []

fn int sample
    emits {}
    asserts
        row: => ok 1
    int earlier = 1
    ok call combine(earlier, call nested(2),
`
	file, err := source.New("partial.can", text)
	if err != nil {
		t.Fatal(err)
	}
	parsed := Parse(file)
	if parsed.OK() || len(parsed.File.Declarations) != 1 {
		t.Fatalf("lost partial declaration: %+v", parsed)
	}
	fn := parsed.File.Declarations[0].(*FunctionDecl)
	if fn.Name.Text != "sample" || len(fn.Body.Steps) != 1 || len(fn.Body.Invalid) == 0 {
		t.Fatalf("bad partial body: %+v", fn)
	}
	if len(parsed.File.Incomplete) != 1 {
		t.Fatalf("contexts: %+v", parsed.File.Incomplete)
	}
	context := parsed.File.Incomplete[0]
	if context.Kind != "call" || len(context.Arguments) != 2 || len(context.ArgumentStarts) != 3 || context.Span.Start != strings.Index(text, "combine(") {
		t.Fatalf("bad canonical argument context: %+v", context)
	}
}

func TestRecoveryLexicalAndDeclarationFindings(t *testing.T) {
	file, _ := source.New("bad.can", "package app\n    provides []\n    uses []\n\nfn int one(\n\nfn int two(\n\nrecord good\n")
	parsed := Parse(file)
	if len(parsed.Diagnostics) < 2 || len(parsed.File.Declarations) != 1 || len(parsed.File.InvalidNames) != 2 {
		t.Fatalf("recovery lost units: %+v / %+v", parsed, parsed.File)
	}
}

func TestRecoveryDeclarationInteriorBoundaries(t *testing.T) {
	file, _ := source.New("interior.can", `package app
    provides []
    uses []
record damaged
    int
    str retained
fn int broken_signature
    emits {}
    given
        int
        str retained
    asserts
        good: "x" => ok 1
    ok 1
fn int independent
    emits {}
    asserts
        broken: => ok )
        good: => ok 1
    ok 1
`)
	result := Parse(file)
	if result.OK() || len(result.File.Declarations) != 3 {
		t.Fatalf("lost independent declarations: %+v", result)
	}
	record := result.File.Declarations[0].(*RecordDecl)
	if len(record.Fields) != 1 || record.Fields[0].Name.Text != "retained" || len(result.File.InvalidDeclarations[record]) != 1 {
		t.Fatalf("field recovery: %+v", record)
	}
	broken := result.File.Declarations[1].(*FunctionDecl)
	if len(result.File.InvalidDeclarations[broken]) != 1 || broken.Body.Terminal == nil {
		t.Fatalf("parameter recovery: %+v", broken)
	}
	fn := result.File.Declarations[2].(*FunctionDecl)
	if len(fn.InvalidAssertions) == 0 || fn.Body.Terminal == nil {
		t.Fatalf("assertion recovery: %+v", fn)
	}
}
