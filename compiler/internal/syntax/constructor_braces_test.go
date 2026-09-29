package syntax

import (
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/source"
)

// Terminal brace constructors complete the region with a FailureBody;
// success-data construction and bare match forwarding stay distinct.
func TestTerminalBraceConstructorsCompleteRegions(t *testing.T) {
	file := parseFile(t, testHeader+`fn int load
    emits {missing}
    asserts
        sample: => ok 1
    missing{"x"}
`)
	fn := file.Declarations[0].(*FunctionDecl)
	failure, ok := fn.Body.Terminal.(*FailureBody)
	if !ok {
		t.Fatalf("terminal is %T", fn.Body.Terminal)
	}
	if failure.Error == nil || !failure.Error.Braces || failure.Error.Name.Name != "missing" {
		t.Fatalf("terminal braces lost: %+v", failure.Error)
	}
	if len(failure.Error.Arguments) != 1 {
		t.Fatalf("terminal arity: %+v", failure.Error)
	}
}

func TestTerminalBraceConstructorsCoverEmptyAndGeneric(t *testing.T) {
	file := parseFile(t, testHeader+`fn void quiet
    emits {}
    asserts
        sample: => ok
    ok
fn int aggregate
    emits {all_failed<a_failure>}
    asserts
        sample: => ok 1
    all_failed<a_failure>{[codec::invalid_data{"a", "type"}]}
`)
	quiet := file.Declarations[0].(*FunctionDecl)
	if _, ok := quiet.Body.Terminal.(*SuccessBody); !ok {
		t.Fatalf("quiet terminal is %T", quiet.Body.Terminal)
	}
	aggregate := file.Declarations[1].(*FunctionDecl)
	failure, ok := aggregate.Body.Terminal.(*FailureBody)
	if !ok {
		t.Fatalf("aggregate terminal is %T", aggregate.Body.Terminal)
	}
	if !failure.Error.Braces || FormatType(failure.Error.Types[0]) != "a_failure" {
		t.Fatalf("generic terminal: %+v", failure.Error)
	}
	payload := failure.Error.Arguments[0].Value.(*ArrayExpr)
	inner := payload.Elements[0].Value.(*ConstructorExpr)
	if !inner.Braces || inner.Name.Name != "invalid_data" {
		t.Fatalf("nested terminal payload: %+v", inner)
	}
}

func TestBraceValueInsideSuccessDataDoesNotComplete(t *testing.T) {
	file := parseFile(t, testHeader+`fn wrapper fetch
    emits {}
    asserts
        sample: => ok wrapper(missing{"x"})
    ok wrapper(missing{"x"})
`)
	fn := file.Declarations[0].(*FunctionDecl)
	success, ok := fn.Body.Terminal.(*SuccessBody)
	if !ok {
		t.Fatalf("terminal is %T", fn.Body.Terminal)
	}
	outer, ok := success.Value.(*ConstructorExpr)
	if !ok || outer.Braces {
		t.Fatalf("outer value: %+v", success.Value)
	}
	inner, ok := outer.Arguments[0].Value.(*ConstructorExpr)
	if !ok || !inner.Braces || inner.Name.Name != "missing" {
		t.Fatalf("stored error value: %+v", outer.Arguments[0].Value)
	}
}

func TestBareErrorForwardingStaysDistinctFromBraces(t *testing.T) {
	file := parseFile(t, testHeader+`fn int guarded
    emits {missing}
    asserts
        sample: => ok 1
    match call lookup("x")
        missing
        ok int value => ok value
`)
	fn := file.Declarations[0].(*FunctionDecl)
	match := fn.Body.Terminal.(*MatchBody)
	if len(match.Match.Arms) != 2 || !match.Match.Arms[0].Forward {
		t.Fatalf("forwarding lost: %+v", match.Match.Arms)
	}
	if match.Match.Arms[0].Body != nil {
		t.Fatalf("forwarding arm gained a body: %+v", match.Match.Arms[0])
	}
}

func TestTerminalBraceSpansMatchSource(t *testing.T) {
	text := testHeader + "fn int load\n    emits {missing}\n    asserts\n        sample: => ok 1\n    missing{\"x\"}\n"
	src, err := source.New("span.can", text)
	if err != nil {
		t.Fatal(err)
	}
	result := Parse(src)
	if !result.OK() {
		t.Fatal(result.Diagnostics[0].Format(src))
	}
	fn := result.File.Declarations[0].(*FunctionDecl)
	failure := fn.Body.Terminal.(*FailureBody)
	span := failure.Error.ExprSpan()
	if got := text[span.Start:span.End]; got != `missing{"x"}` {
		t.Fatalf("terminal span %q", got)
	}
	if err := src.Validate(span); err != nil {
		t.Fatal(err)
	}
}
