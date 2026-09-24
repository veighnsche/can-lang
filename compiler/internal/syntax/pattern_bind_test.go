package syntax

import (
	"strings"
	"testing"
)

// T02 (DI-01): `bind name` explicitly captures the matched value at every
// pattern depth. Bare identifiers are nominal leaf tests, never captures.
func TestBindPatternNodes(t *testing.T) {
	program := testHeader + `fn int inspect
    emits []
    given
        int[] values
    asserts
        empty: [] => ok 0
    match values
        [] => ok 0
        [bind head, ...tail] => ok head
        bind all => ok all.length
`
	file := parseFile(t, program)
	match := file.Declarations[0].(*FunctionDecl).Body.Terminal.(*MatchBody).Match
	array, ok := match.Arms[1].Patterns[0].(*ArrayPattern)
	if !ok {
		t.Fatalf("array arm is %T", match.Arms[1].Patterns[0])
	}
	head, ok := array.Elements[0].(*BindPattern)
	if !ok || head.Name.Text != "head" {
		t.Fatalf("array element is %+v", array.Elements[0])
	}
	if array.Rest == nil || array.Rest.Text != "tail" {
		t.Fatalf("rest lost: %+v", array.Rest)
	}
	all, ok := match.Arms[2].Patterns[0].(*BindPattern)
	if !ok || all.Name.Text != "all" {
		t.Fatalf("top-level binder is %+v", match.Arms[2].Patterns[0])
	}
	rendered := Format(file)
	if !strings.Contains(rendered, "[bind head, ...tail]") || !strings.Contains(rendered, "bind all =>") {
		t.Fatalf("bind patterns did not render:\n%s", rendered)
	}
}

func TestBindPatternNestedAlternatives(t *testing.T) {
	program := testHeader + `fn int inspect
    emits []
    given
        shape value
    asserts
        round: circle(3) => ok 3
    match value
        circle(bind radius) | rectangle(bind radius, _) => ok radius
`
	file := parseFile(t, program)
	alternative, ok := file.Declarations[0].(*FunctionDecl).Body.Terminal.(*MatchBody).Match.Arms[0].Patterns[0].(*AlternativePattern)
	if !ok || len(alternative.Alternatives) != 2 {
		t.Fatal("alternative arm lost")
	}
	first, ok := alternative.Alternatives[0].(*ConstructorPattern)
	if !ok {
		t.Fatalf("first branch is %T", alternative.Alternatives[0])
	}
	radius, ok := first.Fields[0].(*BindPattern)
	if !ok || radius.Name.Text != "radius" {
		t.Fatalf("nested binder is %+v", first.Fields[0])
	}
	if !strings.Contains(Format(file), "circle(bind radius) | rectangle(bind radius, _)") {
		t.Fatalf("nested binders did not render:\n%s", Format(file))
	}
}

func TestBindFallsBackToNominal(t *testing.T) {
	// `bind` followed by anything but a name keeps its nominal reading, so
	// a leaf literally named bind is still addressable.
	program := testHeader + `fn int inspect
    emits []
    given
        shape value
    asserts
        round: circle(3) => ok 1
    match value
        bind => ok 1
        bind(_) => ok 2
        bind bind => ok 3
`
	file := parseFile(t, program)
	arms := file.Declarations[0].(*FunctionDecl).Body.Terminal.(*MatchBody).Match.Arms
	if name, ok := arms[0].Patterns[0].(*NamePattern); !ok || name.Name.Name != "bind" {
		t.Fatalf("bare bind is %+v", arms[0].Patterns[0])
	}
	if constructor, ok := arms[1].Patterns[0].(*ConstructorPattern); !ok || constructor.Name.Name != "bind" {
		t.Fatalf("bind(_) is %+v", arms[1].Patterns[0])
	}
	if binder, ok := arms[2].Patterns[0].(*BindPattern); !ok || binder.Name.Text != "bind" {
		t.Fatalf("bind bind is %+v", arms[2].Patterns[0])
	}
}
