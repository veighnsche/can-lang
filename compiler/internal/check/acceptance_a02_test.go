package check

import (
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/source"
)

const withCombineDecl = `fn int combine
    emits []
    given
        near int prefix
        int value
        near int suffix
    asserts
        sample: 3, 4, 5 => ok 12
    ok prefix + value + suffix
`

// AU-Q3-core: explicit near bindings check, capture the pinned values and
// leave unlisted near-inputs on name lookup; direct calls stay positional.
func TestAUQ3CoreExplicitBindings(t *testing.T) {
	text := programHeader + withCombineDecl + `fn int compute
    emits []
    given
        int seed
    asserts
        first: 4 => ok 12
    callable int (int) emits [] action = callable combine with suffix = 5, prefix = 3
    ok call action(seed)
fn int direct
    emits []
    asserts
        sample: => ok 12
    ok call combine(3, 4, 5)
` + programMain + "    ok\n"
	program, err := programFixture(t, map[string]string{"src/main.can": text})
	if err != nil {
		t.Fatalf("explicit bindings rejected: %v", err)
	}
	found := false
	for _, fn := range program.Functions {
		if fn.Symbol.Name != "compute" {
			continue
		}
		closure := fn.Region.Body.Steps[0].Value
		if closure.Kind != ir.CallableValue {
			t.Fatalf("binding did not produce a callable: %+v", closure)
		}
		// Listed out of declaration order; captures still land on the
		// declared slots with exactly-once pinned values.
		if len(closure.Callable.Positions) != 2 || closure.Callable.Positions[0] != 0 || closure.Callable.Positions[1] != 2 {
			t.Fatalf("invalid capture plan: %+v", closure.Callable)
		}
		if len(closure.Inputs) != 2 || closure.Inputs[0].Kind != ir.Literal || closure.Inputs[0].Text != "3" || closure.Inputs[1].Kind != ir.Literal || closure.Inputs[1].Text != "5" {
			t.Fatalf("pinned values not captured: %+v", closure.Inputs)
		}
		found = true
	}
	if !found {
		t.Fatal("missing concrete compute region")
	}
}

func TestAUQ3CorePartialFallback(t *testing.T) {
	text := programHeader + withCombineDecl + `fn int partial
    emits []
    given
        int suffix
        int value
    asserts
        sample: 5, 4 => ok 12
    callable int (int) emits [] action = callable combine with prefix = 3
    ok call action(value)
` + programMain + "    ok\n"
	program, err := programFixture(t, map[string]string{"src/main.can": text})
	if err != nil {
		t.Fatalf("partial bindings rejected: %v", err)
	}
	found := false
	for _, fn := range program.Functions {
		if fn.Symbol.Name != "partial" {
			continue
		}
		closure := fn.Region.Body.Steps[0].Value
		if closure.Kind != ir.CallableValue || len(closure.Inputs) != 2 {
			t.Fatalf("fallback capture plan lost: %+v", closure)
		}
		if closure.Inputs[0].Kind != ir.Literal || closure.Inputs[0].Text != "3" {
			t.Fatalf("pinned prefix lost: %+v", closure.Inputs)
		}
		if closure.Inputs[1].Kind != ir.Binding || closure.Inputs[1].Text == "" {
			t.Fatalf("unlisted suffix left lookup: %+v", closure.Inputs)
		}
		found = true
	}
	if !found {
		t.Fatal("missing concrete partial region")
	}
}

// AU-Q3-core negatives: unknown, duplicate, non-near and mistyped names
// fail with located CAN-CHECK-CAPTURE.
func TestAUQ3CoreBindingRefusals(t *testing.T) {
	for _, tc := range []struct{ name, binding, message string }{
		{"unknown", "callable combine with absent = 3", "unknown with binding absent"},
		{"duplicate", "callable combine with prefix = 3, prefix = 4", "duplicate with binding prefix"},
		{"non-near", "callable combine with value = 3", "is not a near input"},
		{"mistyped", `callable combine with prefix = "s"`, "does not fit expected type"},
		{"array", "callable ([1, 2, 3]).map with policy = 1", "declares no near inputs"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			text := programHeader + withCombineDecl + `fn int broken
    emits []
    given
        int seed
    asserts
        first: 4 => ok 12
    callable int (int) emits [] action = ` + tc.binding + `
    ok call action(seed)
` + programMain + "    ok\n"
			_, err := programFixture(t, map[string]string{"src/main.can": text})
			if err == nil {
				t.Fatalf("invalid binding accepted: %s", tc.binding)
			}
			located, ok := source.AsLocated(err)
			if !ok || located.Code != "CAN-CHECK-CAPTURE" {
				t.Fatalf("refusal lost its CAN-CHECK-CAPTURE location: %v", err)
			}
			if !strings.Contains(err.Error(), tc.message) {
				t.Fatalf("wrong refusal: %v", err)
			}
		})
	}
}
