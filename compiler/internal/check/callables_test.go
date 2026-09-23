package check

import (
	"fmt"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
	"os"
	"strings"
	"testing"
)

func callableFixture(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("../../testdata/current/callables/captures.can")
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
func TestCheckedCallableCapturesAndDynamicCallees(t *testing.T) {
	p, err := programFixture(t, map[string]string{"src/main.can": callableFixture(t)})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, fn := range p.Functions {
		if fn.Symbol.Name == "compute" {
			closure := fn.Region.Body.Steps[0].Value
			if closure.Kind != ir.CallableValue || len(closure.Inputs) != 2 || len(closure.Callable.Positions) != 2 || closure.Callable.Positions[0] != 0 || closure.Callable.Positions[1] != 2 || len(closure.Type.Inputs()) != 1 {
				t.Fatalf("invalid capture plan: %+v", closure)
			}
			found = true
		}
	}
	if !found {
		t.Fatal("missing concrete callable region")
	}
}
func TestCallableReferenceRefusals(t *testing.T) {
	original := callableFixture(t)
	for _, tc := range []struct{ name, old, replacement string }{
		{"missing capture", "int prefix\n        int suffix\n        int value", "int absent\n        int suffix\n        int value"},
		{"wrong capture type", "int prefix\n        int suffix\n        int value", "str prefix\n        int suffix\n        int value"},
		{"wrong input", "callable int (int) emits [] action = callable combine", "callable int (str) emits [] action = callable combine"},
		{"wrong result", "callable int (int) emits [] action = callable combine", "callable str (int) emits [] action = callable combine"},
		{"static method reference", "callable (call make_box()).read", "callable read"},
		{"direct near omitted", "call compute(3, 5, 4)", "call combine(4)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := strings.Replace(original, tc.old, tc.replacement, 1)
			if source == original {
				t.Fatal("invalid refusal fixture")
			}
			if _, err := programFixture(t, map[string]string{"src/main.can": source}); err == nil {
				t.Fatal("invalid reference accepted")
			}
		})
	}
}

func TestNativeDeclarationReferenceEligibility(t *testing.T) {
	p, err := programFixture(t, map[string]string{"src/main.can": callableFixture(t)})
	if err != nil {
		t.Fatal(err)
	}
	fn := p.Functions[0].Region
	var inputs []*types.Type
	for _, input := range fn.Inputs {
		inputs = append(inputs, input.Type)
	}
	contract, err := types.CallableOfChecked(fn.Result, inputs, fn.Errors)
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []resolve.Kind{resolve.Function, resolve.Fetch, resolve.Question, resolve.Judge, resolve.LLM} {
		expected := kind == resolve.Function || kind == resolve.Fetch
		symbol := &resolve.Symbol{Kind: kind}
		if symbol.Eligible(resolve.ReferenceUse) != expected {
			t.Fatalf("incorrect reference namespace for %s", kind)
		}
		declaration := CallableDeclaration{Kind: kind, Contract: contract, Names: []string{"value"}, Near: []bool{false}}
		if (declaration.validate() == nil) != expected {
			t.Fatalf("incorrect callable gate for %s", kind)
		}
		src, _ := source.New("reference.can", programHeader+"fn callable int (int) emits [] make\n    emits []\n    asserts\n        sample: => ok callable chosen\n    ok callable chosen\n")
		parsed := syntax.Parse(src)
		if !parsed.OK() {
			t.Fatal(parsed.Diagnostics)
		}
		body := parsed.File.Declarations[0].(*syntax.FunctionDecl).Body
		scope := resolve.NewScope(nil)
		symbol.Name = "chosen"
		symbol.ID = "native/chosen"
		if err := scope.Define(symbol); err != nil {
			t.Fatal(err)
		}
		bound, err := p.Registry.Bound(nil)
		if err != nil {
			t.Fatal(err)
		}
		context := CompletionContext{Identity: "test/reference", Kind: ir.FunctionRegion, File: src, Scope: scope, Result: contract, Registry: p.Registry, Errors: bound, Callables: map[string]CallableDeclaration{symbol.ID: declaration}, Expressions: &Expressions{Reference: func(name syntax.QualifiedName) (ValueBinding, error) {
			found, err := scope.Lookup(name.Name, resolve.ReferenceUse)
			if err != nil {
				return ValueBinding{}, err
			}
			return ValueBinding{Identity: found.ID, Type: contract}, nil
		}}, Type: func(syntax.TypeNode, bool) (*types.Type, error) { return nil, fmt.Errorf("unexpected annotation") }}
		region, err := CheckRegion(context, body)
		if (err == nil) != expected {
			t.Fatalf("parsed %s reference eligibility: %v", kind, err)
		}
		if expected && (region.Body.Terminal.Value.Kind != ir.CallableValue || region.Body.Terminal.Value.Callable.Target != symbol.ID) {
			t.Fatal("reference did not retain native target")
		}

	}
}
func TestCallableErrorBoundsAndExactCaptureTypes(t *testing.T) {
	source := programHeader + `fn int safe
    emits []
    given
        int value
    asserts
        sample: 1 => ok 1
    ok value
fn int fallible
    emits [codec::invalid_data]
    given
        int value
    asserts
        sample: 1 => ok 1
    ok value
` + programMain + `    callable int (int) emits [codec::invalid_data] action = callable safe
    ok
`
	source = strings.Replace(source, "uses []", "uses [codec]", 1)
	if _, err := programFixture(t, map[string]string{"src/main.can": source}); err != nil {
		t.Fatal(err)
	}
	bad := strings.Replace(source, "callable int (int) emits [codec::invalid_data] action = callable safe", "callable int (int) emits [] action = callable fallible", 1)
	if _, err := programFixture(t, map[string]string{"src/main.can": bad}); err == nil || !strings.Contains(err.Error(), "expected type") {
		t.Fatalf("wider error bound admitted: %v", err)
	}
}

// Capture requirements name the capture, the declaring callable and the
// exact expected type at the reference site. No fix guesses a value or a
// different capture.
func TestCaptureObligation(t *testing.T) {
	original := callableFixture(t)
	mistyped := original
	for _, edit := range [][2]string{
		{"int prefix\n        int suffix\n        int value", "str prefix\n        int suffix\n        int value"},
		{`first: 3, 5, 4 => ok 12`, `first: "p", 5, 4 => ok 12`},
		{`second: 7, 11, 4 => ok 22`, `second: "q", 11, 4 => ok 22`},
		{`call compute(3, 5, 4)`, `call compute("p", 5, 4)`},
		{`call compute(7, 11, 4)`, `call compute("q", 11, 4)`},
	} {
		next := strings.Replace(mistyped, edit[0], edit[1], 1)
		if next == mistyped {
			t.Fatalf("mutation missed: %q", edit[0])
		}
		mistyped = next
	}
	_, err := programFixture(t, map[string]string{"src/main.can": mistyped})
	if err == nil {
		t.Fatal("mistyped capture admitted")
	}
	for _, want := range []string{"requires exact declared type", "prefix"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("capture diagnostic omits %q: %v", want, err)
		}
	}
	located, ok := source.AsLocated(err)
	if !ok || located.Code != "CAN-CHECK-CAPTURE" {
		t.Fatalf("capture failure lost code or span: %v", err)
	}
	if len(located.Fixes) != 0 {
		t.Fatalf("capture failure proposed fixes: %+v", located.Fixes)
	}

	missing := strings.Replace(original, "int prefix\n        int suffix\n        int value", "int absent\n        int suffix\n        int value", 1)
	if missing == original {
		t.Fatal("invalid refusal fixture")
	}
	_, err = programFixture(t, map[string]string{"src/main.can": missing})
	if err == nil {
		t.Fatal("missing capture admitted")
	}
	located, ok = source.AsLocated(err)
	if !ok || located.Code != "CAN-CHECK-CAPTURE" || !strings.Contains(err.Error(), "near capture") {
		t.Fatalf("missing capture misdiagnosed: %v", err)
	}
}
