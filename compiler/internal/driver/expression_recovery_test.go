package driver

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/source"
)

// Each pair belongs to one expression. A trustworthy sibling is checkable
// without inventing the failed sibling's result type or executable IR.
func TestRecoveryIndependentExpressionChildren(t *testing.T) {
	cases := []struct{ name, result, expectation, expression string }{
		{"binary", "int", "1", "call missing_left() + call missing_right()"},
		{"comparison", "bool", "true", "call missing_left() < call missing_right()"},
		{"array", "int[]", "[1]", "[call missing_left(), call missing_right()]"},
		{"constructor", "pair", "pair(1, 2)", "pair(call missing_left(), call missing_right())"},
		{"update", "pair", "pair(1, 2)", "config with (left = call missing_left(), right = call missing_right())"},
		{"slice", "int[]", "[1]", "[1, 2][call missing_left():call missing_right()]"},
		{"index", "int", "1", "missing_left[call missing_right()]"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			text := fmt.Sprintf(`package app
    provides []
    uses []
record pair
    int left
    int right
pair config = pair(1, 2)
fn %s broken
    emits {}
    asserts
        sample: => ok %s
    ok %s
`, tc.result, tc.expectation, tc.expression)
			root := writeBridgeProject(t, map[string]string{"src/main.can": text})
			path := filepath.Join(root, "src/main.can")
			snapshot, err := CheckSnapshot(root, path, nil)
			if err != nil {
				t.Fatal(err)
			}
			file, _ := source.New(path, text)
			found := map[string]int{}
			for _, diagnostic := range snapshot.Diagnostics {
				if diagnostic.Severity != "error" {
					continue
				}
				start, e1 := file.Offset(source.UTF16Position{Line: diagnostic.Line, Character: diagnostic.Start})
				end, e2 := file.Offset(source.UTF16Position{Line: diagnostic.EndLine, Character: diagnostic.End})
				if e1 != nil || e2 != nil {
					t.Fatalf("invalid diagnostic coordinates: %+v", diagnostic)
				}
				token := text[start:end]
				found[token]++
				if token != "missing_left" && token != "missing_right" {
					t.Errorf("cascade or widened range %q: %+v", token, diagnostic)
				}
			}
			if len(found) != 2 || found["missing_left"] != 1 || found["missing_right"] != 1 {
				t.Fatalf("independent child findings missing: %v; diagnostics %+v", found, snapshot.Diagnostics)
			}
			for _, fn := range snapshot.Program.Functions {
				if fn.Symbol.Name == "broken" && fn.Region != nil {
					t.Fatal("invalid expression published executable region")
				}
			}
			if program, err := check.CheckAssertionProgram(snapshot.Graph); err == nil || program != nil {
				t.Fatal("strict compiler accepted invalid expression")
			}
		})
	}
}
