package driver

import (
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/source"
)

func TestOccurrenceIndexSealsNestedSlices(t *testing.T) {
	b := NewIndexBuilder()
	input := Occurrence{ID: "local:x", File: "/a.can", NameSpan: source.Span{Start: 2, End: 3}, Scope: []string{"app"}, Arguments: []source.Span{{Start: 5, End: 6}}, Valid: true}
	b.Add(input)
	input.Scope[0] = "corrupt"
	input.Arguments[0].Start = 99
	index := b.Seal()
	all := index.Occurrences()
	if all[0].Scope[0] != "app" || all[0].Arguments[0].Start != 5 {
		t.Fatalf("builder alias: %+v", all[0])
	}
	all[0].Scope[0] = "corrupt"
	all[0].Arguments[0].Start = 99
	byID := index.Binding("local:x")
	if byID[0].Scope[0] != "app" || byID[0].Arguments[0].Start != 5 {
		t.Fatalf("getter alias: %+v", byID[0])
	}
	byID[0].Scope[0] = "corrupt"
	at, ok := index.At("/a.can", 2)
	if !ok || at.Scope[0] != "app" {
		t.Fatalf("At alias: %+v", at)
	}
}

func TestFixDiagnosticMappingKeepsDistinctLocations(t *testing.T) {
	before := "first\nsecond\nthird\n"
	fix := Fix{File: "/a.can", Start: 6, End: 6, NewText: "inserted\n"}
	after := before[:fix.Start] + fix.NewText + before[fix.End:]
	one := Diagnostic{File: fix.File, Severity: "error", Code: "same", Message: "same", Line: 1, EndLine: 1, Start: 0, End: 6}
	two := Diagnostic{File: fix.File, Severity: "error", Code: "same", Message: "same", Line: 2, EndLine: 2, Start: 0, End: 5}
	mappedOne, ok := mapFixDiagnostic(one, before, after, fix)
	if !ok {
		t.Fatal("first mapping failed")
	}
	mappedTwo, ok := mapFixDiagnostic(two, before, after, fix)
	if !ok {
		t.Fatal("second mapping failed")
	}
	if mappedOne.Line != 2 || mappedTwo.Line != 3 || fixIssueKey(mappedOne) == fixIssueKey(mappedTwo) {
		t.Fatalf("distinct diagnostics collapsed: %+v %+v", mappedOne, mappedTwo)
	}
}

func TestFixDiagnosticMappingDoesNotMutateSnapshot(t *testing.T) {
	d := Diagnostic{File: "/other.can", Related: []RelatedDiagnostic{{File: "/a.can", Line: 0, Start: 4, EndLine: 0, End: 7}}}
	mapped, ok := mapFixDiagnostic(d, "foo bar", "longfoo bar", Fix{File: "/a.can", Start: 0, End: 0, NewText: "long"})
	if !ok || mapped.Related[0].Start != 8 || d.Related[0].Start != 4 {
		t.Fatalf("mapping mutated input: mapped=%+v original=%+v", mapped.Related, d.Related)
	}
}
