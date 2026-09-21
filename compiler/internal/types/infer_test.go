package types

import (
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/resolve"
)

func inferenceFixture(t *testing.T) (*resolve.File, map[string]*Type) {
	t.Helper()
	b, file := buildSource(t, sourceHeader+`record box<item>
    item value
record other<item>
    item value
record node
    node[] children
variant failures
    node
    standard_failure
`)
	values := map[string]*Type{}
	for _, name := range []string{"int", "float", "str", "void", "box<int>", "box<float>", "other<int>", "box<node>", "node", "node[]", "all_failed<failures>", "failures", "callable int (int[]) emits []", "callable str (int[]) emits []"} {
		typ, err := b.Resolve(file, annotation(t, name), nil, true)
		if err != nil {
			t.Fatal(err)
		}
		values[name] = typ
	}
	if _, err := b.Finish(); err != nil {
		t.Fatal(err)
	}
	return file, values
}

func TestFiniteInferenceEqualities(t *testing.T) {
	file, values := inferenceFixture(t)
	for _, tc := range []struct{ pattern, actual, want string }{
		{"item", "int", "int"}, {"box<item>", "box<int>", "int"}, {"item[]", "node[]", "node"},
		{"box<item>", "box<node>", "node"}, {"callable item (item[]) emits []", "callable int (int[]) emits []", "int"},
		{"all_failed<item>", "all_failed<failures>", "failures"},
	} {
		t.Run(tc.pattern+"/"+tc.actual, func(t *testing.T) {
			pattern, err := Pattern(file, annotation(t, tc.pattern), []string{"item"})
			if err != nil {
				t.Fatal(err)
			}
			solver, _ := NewInference([]string{"item"})
			if err := solver.Constrain(pattern, values[tc.actual]); err != nil {
				t.Fatal(err)
			}
			args, err := solver.Arguments()
			if err != nil || len(args) != 1 || !Equal(args[0], values[tc.want]) {
				t.Fatalf("inferred %v: %v", args, err)
			}
		})
	}
}

func TestInferenceRejectsConflictAmbiguityAndWidening(t *testing.T) {
	file, values := inferenceFixture(t)
	item, _ := Pattern(file, annotation(t, "item"), []string{"item"})
	solver, _ := NewInference([]string{"item"})
	if _, err := solver.Arguments(); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("unconstrained parameter: %v", err)
	}
	if err := solver.Constrain(item, values["int"]); err != nil {
		t.Fatal(err)
	}
	if err := solver.Constrain(item, values["float"]); err == nil {
		t.Fatal("numeric widening entered inference")
	}
	args, err := solver.Arguments()
	if err != nil || !Equal(args[0], values["int"]) {
		t.Fatal("conflict overwrote prior constraint")
	}
	for _, tc := range []struct{ pattern, actual string }{
		{"box<item>", "other<int>"}, {"box<item>", "int"}, {"item[]", "box<int>"},
		{"callable item (item[]) emits []", "callable str (int[]) emits []"}, {"item", "void"},
	} {
		pattern, err := Pattern(file, annotation(t, tc.pattern), []string{"item"})
		if err != nil {
			t.Fatal(err)
		}
		solver, _ := NewInference([]string{"item"})
		if err := solver.Constrain(pattern, values[tc.actual]); err == nil {
			t.Fatalf("accepted %s from %s", tc.pattern, tc.actual)
		}
		if _, err := solver.Arguments(); err == nil {
			t.Fatal("failed structural constraint retained a partial binding")
		}
	}
}

func TestInferenceExpectedAndFixedConstraintsAgree(t *testing.T) {
	file, values := inferenceFixture(t)
	parameter, _ := Pattern(file, annotation(t, "item"), []string{"item"})
	result, _ := Pattern(file, annotation(t, "box<item>"), []string{"item"})
	for _, reverse := range []bool{false, true} {
		solver, _ := NewInference([]string{"item"})
		patterns := []*InferencePattern{parameter, result}
		actuals := []*Type{values["int"], values["box<int>"]}
		if reverse {
			patterns[0], patterns[1] = patterns[1], patterns[0]
			actuals[0], actuals[1] = actuals[1], actuals[0]
		}
		for i, pattern := range patterns {
			if err := solver.Constrain(pattern, actuals[i]); err != nil {
				t.Fatal(err)
			}
		}
		if args, err := solver.Arguments(); err != nil || !Equal(args[0], values["int"]) {
			t.Fatal("constraint order changed solution")
		}
		if err := solver.Constrain(result, values["box<float>"]); err == nil {
			t.Fatal("expected result silently changed fixed input solution")
		}
	}
}

func TestInferenceRequiresSealedEvidence(t *testing.T) {
	b, file := buildSource(t, sourceHeader)
	actual, err := b.Resolve(file, annotation(t, "int"), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	pattern, err := Pattern(file, annotation(t, "item"), []string{"item"})
	if err != nil {
		t.Fatal(err)
	}
	solver, _ := NewInference([]string{"item"})
	if err := solver.Constrain(pattern, actual); err == nil {
		t.Fatal("unsealed evidence entered inference")
	}
	if _, err := b.Finish(); err != nil {
		t.Fatal(err)
	}
	if err := solver.Constrain(pattern, actual); err != nil {
		t.Fatal(err)
	}
}
