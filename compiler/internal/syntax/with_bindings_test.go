package syntax

import (
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/source"
)

func withExpression(t *testing.T, text string) *ReferenceExpr {
	t.Helper()
	file, err := source.New("with.can", text)
	if err != nil {
		t.Fatal(err)
	}
	node, diagnostics := ParseExpression(file)
	if len(diagnostics) != 0 {
		t.Fatalf("parse: %s", diagnostics[0].Format(file))
	}
	reference, ok := node.(*ReferenceExpr)
	if !ok {
		t.Fatalf("with parsed as %T", node)
	}
	return reference
}

// Q3: `callable <name> with <param> = <expr>, ...` pins near-inputs in
// listed order; the formatter preserves that order at the fixpoint.
func TestWithBindingsParseAndFormat(t *testing.T) {
	reference := withExpression(t, "callable combine with suffix = 5, prefix = 3")
	if len(reference.Bindings) != 2 || reference.Bindings[0].Name.Text != "suffix" || reference.Bindings[1].Name.Text != "prefix" {
		t.Fatalf("binding order lost: %+v", reference.Bindings)
	}
	formatted := FormatExpression(reference)
	if formatted != "callable combine with suffix = 5, prefix = 3" {
		t.Fatalf("bindings not preserved: %q", formatted)
	}
	again := withExpression(t, formatted)
	if FormatExpression(again) != formatted {
		t.Fatal("bindings not at fixpoint")
	}
}

func TestWithBindingsSingle(t *testing.T) {
	reference := withExpression(t, "callable combine with prefix = 3")
	if len(reference.Bindings) != 1 || reference.Bindings[0].Name.Text != "prefix" {
		t.Fatalf("single binding lost: %+v", reference.Bindings)
	}
	if formatted := FormatExpression(reference); formatted != "callable combine with prefix = 3" {
		t.Fatalf("single binding moved: %q", formatted)
	}
}

func TestWithBindingsMethodCallee(t *testing.T) {
	reference := withExpression(t, "callable box.read with prefix = 3")
	if len(reference.Bindings) != 1 {
		t.Fatalf("method binding lost: %+v", reference.Bindings)
	}
	if formatted := FormatExpression(reference); formatted != "callable box.read with prefix = 3" {
		t.Fatalf("method binding moved: %q", formatted)
	}
}

func TestWithBindingsNegatives(t *testing.T) {
	for _, text := range []string{
		"callable combine with prefix = 3,",
		"callable combine with prefix = ,",
		"callable combine with prefix",
		"callable combine with 3 = 3",
	} {
		file, err := source.New("with.can", text)
		if err != nil {
			t.Fatal(err)
		}
		if node, diagnostics := ParseExpression(file); len(diagnostics) == 0 {
			t.Fatalf("invalid bindings parsed: %T", node)
		}
	}
}

// A pinned callable stays a valid element of an outer comma list: a
// comma continues the bindings only when `Name =` follows it.
func TestWithBindingsInsideCallArguments(t *testing.T) {
	file, err := source.New("with.can", "call consume(callable combine with prefix = 3, value)")
	if err != nil {
		t.Fatal(err)
	}
	node, diagnostics := ParseExpression(file)
	if len(diagnostics) != 0 {
		t.Fatalf("parse: %s", diagnostics[0].Format(file))
	}
	call, ok := node.(*CallExpr)
	if !ok || len(call.Invocation.Arguments) != 2 {
		t.Fatalf("pinned callable broke the argument list: %T", node)
	}
	reference, ok := call.Invocation.Arguments[0].Value.(*ReferenceExpr)
	if !ok || len(reference.Bindings) != 1 || reference.Bindings[0].Name.Text != "prefix" {
		t.Fatalf("first argument lost its binding: %T", call.Invocation.Arguments[0].Value)
	}
	if formatted := FormatExpression(node); formatted != "call consume(callable combine with prefix = 3, value)" {
		t.Fatalf("argument list moved: %q", formatted)
	}
}

// A parenthesized tail stays record-update syntax: only `with Name =`
// takes the binding path.
func TestWithBindingsParenthesizedStaysUpdate(t *testing.T) {
	file, err := source.New("with.can", "callable combine with (prefix = 3, suffix = 5)")
	if err != nil {
		t.Fatal(err)
	}
	node, diagnostics := ParseExpression(file)
	if len(diagnostics) != 0 {
		t.Fatalf("parse: %s", diagnostics[0].Format(file))
	}
	if _, ok := node.(*UpdateExpr); !ok {
		t.Fatalf("parenthesized tail parsed as %T", node)
	}
}
