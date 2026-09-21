package check

import (
	"os"
	"strings"
	"testing"
)

func TestArrayCatalogueCalls(t *testing.T) {
	for _, name := range []string{"main", "contracts"} {
		text, err := os.ReadFile("../../testdata/current/arrays/" + name + ".can")
		if err != nil {
			t.Fatal(err)
		}
		program, err := programFixture(t, map[string]string{"src/main.can": string(text)})
		if err != nil {
			t.Fatal(err)
		}
		if len(program.Functions) < 10 {
			t.Fatal("array fixtures did not compile")
		}
	}
}
func TestArrayCatalogueRejectsWrongCallbacks(t *testing.T) {
	source, err := os.ReadFile("../../testdata/current/arrays/main.can")
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range [][3]string{
		{"items.filter(callable selected)", "items.filter(callable twice)", "callback result"},
		{"items.map(callable twice)", "items.map(callable observe)", "map callback must return data"},
		{"items.for_each(callable observe)", "items.for_each(callable twice)", "callback result"},
		{"items.fold(10, callable add)", "items.fold(10, callable selected)", "callback input arity"},
		{"items.map(callable twice)", "items.map(1)", "callback input arity"},
		{".concat([4])", ".concat([true])", "expected type"},
		{"items.sort_by(callable identity)", "items.sort_by(callable observe)", "sort key must be"},
		{"items.map(callable twice)", "items.map(callable selected)", "expression type does not fit"},
	} {
		t.Run(change[1], func(t *testing.T) {
			text := strings.Replace(string(source), change[0], change[1], 1)
			if _, err := programFixture(t, map[string]string{"src/main.can": text}); err == nil || !strings.Contains(err.Error(), change[2]) {
				t.Fatalf("expected %q, got %v", change[2], err)
			}
		})
	}
}

func TestArrayCatalogueRejectsLostErrorBound(t *testing.T) {
	source, err := os.ReadFile("../../testdata/current/arrays/contracts.can")
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range [][2]string{
		{"fn int[] stops\n    emits [codec::invalid_data]", "fn int[] stops\n    emits []"},
		{"callable int[] (callable int (int) emits [codec::invalid_data]) emits [codec::invalid_data] action", "callable int[] (callable int (int) emits [codec::invalid_data]) emits [] action"},
	} {
		text := strings.Replace(string(source), change[0], change[1], 1)
		if text == string(source) {
			t.Fatal("mutation missed source")
		}
		if _, err := programFixture(t, map[string]string{"src/main.can": text}); err == nil {
			t.Fatal("array callback failure bound was discarded")
		}
	}
}

func TestArrayInferenceAndSpreadRejections(t *testing.T) {
	source, err := os.ReadFile("../../testdata/current/arrays/main.can")
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range [][2]string{
		{"items.map(...[callable twice])", "items.map(...[callable twice, callable twice])"},
		{"items.concat(...[[4]])", "items.concat(...items)"},
		{"append(...[items], ...[3])", "append(...[items, 3])"},
		{"[].fold(0, callable generic_add)", "[].fold(false, callable generic_add)"},
		{"[].map(callable identity)", "[].map(callable generic_add)"},
		{"[].map(callable ([1]).concat)", "[].map(callable ([1]).map)"},
	} {
		text := strings.Replace(string(source), change[0], change[1], 1)
		if text == string(source) {
			t.Fatal("mutation missed source")
		}
		if _, err := programFixture(t, map[string]string{"src/main.can": text}); err == nil {
			t.Fatalf("accepted invalid array constraints: %s", change[1])
		}
	}
}

func TestEmptyArrayGenericInferenceDoesNotGuess(t *testing.T) {
	source, err := os.ReadFile("../../testdata/current/arrays/main.can")
	if err != nil {
		t.Fatal(err)
	}
	for name, extra := range map[string]string{
		"near conflict": `
fn item near_identity<item>
    emits []
    given
        near item prefix
        item value
    asserts
        unit: 1, 2 => ok 2
    ok value
fn int[] conflict
    emits []
    given
        str prefix
    asserts
        unit: "x" => ok []
    ok call [].map(callable near_identity)
`,
		"unconstrained input": `
fn output produce<output, input>
    emits []
    given
        near output result
        input value
    asserts
        unit: 1, "x" => ok 1
    ok result
fn int[] ambiguous
    emits []
    given
        int result
    asserts
        unit: 1 => ok []
    ok call [].map(callable produce)
`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := programFixture(t, map[string]string{"src/main.can": string(source) + extra}); err == nil {
				t.Fatal("inconsistent or incomplete constraints admitted")
			}
		})
	}
}
