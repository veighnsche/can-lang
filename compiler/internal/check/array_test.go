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
