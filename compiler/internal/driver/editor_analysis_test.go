package driver

import (
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/source"
)

func TestFingerprintRequiresCompleteInputs(t *testing.T) {
	base := NewFingerprint(
		[]DocInput{{Path: "a.can", Version: 3, Text: "ok"}},
		[]DiskInput{{Label: "manifest", Value: "m1"}, {Label: "registry", Value: "r1"}},
	)
	same := NewFingerprint(
		[]DocInput{{Path: "a.can", Version: 3, Text: "ok"}},
		[]DiskInput{{Label: "registry", Value: "r1"}, {Label: "manifest", Value: "m1"}},
	)
	if !base.Equal(same) {
		t.Fatal("input order changed identity")
	}
	// Same version with different text is a different analysis.
	edited := NewFingerprint(
		[]DocInput{{Path: "a.can", Version: 3, Text: "ok "}},
		[]DiskInput{{Label: "manifest", Value: "m1"}, {Label: "registry", Value: "r1"}},
	)
	if base.Equal(edited) {
		t.Fatal("text change reused identity")
	}
	// Any disk mutation invalidates.
	manifest := NewFingerprint(
		[]DocInput{{Path: "a.can", Version: 3, Text: "ok"}},
		[]DiskInput{{Label: "manifest", Value: "m2"}, {Label: "registry", Value: "r1"}},
	)
	if base.Equal(manifest) {
		t.Fatal("manifest change reused identity")
	}
	sibling := NewFingerprint(
		[]DocInput{{Path: "a.can", Version: 3, Text: "ok"}, {Path: "b.can", Version: 1, Text: "ok"}},
		[]DiskInput{{Label: "manifest", Value: "m1"}, {Label: "registry", Value: "r1"}},
	)
	if base.Equal(sibling) {
		t.Fatal("sibling addition reused identity")
	}
	if base.Digest() == "" || base.Digest() != same.Digest() || base.Digest() == edited.Digest() {
		t.Fatal("digest mismatch")
	}
}

func TestOrderIssuesDeterministicAndDeduped(t *testing.T) {
	issues := []source.Issue{
		{File: "b.can", Span: source.Span{Start: 5, End: 6}, Code: "E2", Severity: source.SeverityError, Message: "second file"},
		{File: "a.can", Span: source.Span{Start: 9, End: 10}, Code: "E1", Severity: source.SeverityWarning, Message: "later span"},
		{File: "a.can", Span: source.Span{Start: 1, End: 2}, Code: "E1", Severity: source.SeverityError, Message: "earlier span"},
		{File: "a.can", Span: source.Span{Start: 1, End: 2}, Code: "E1", Severity: source.SeverityError, Message: "earlier span"},
		{File: "a.can", Span: source.Span{Start: 1, End: 2}, Code: "E0", Severity: source.SeverityError, Message: "earlier span"},
	}
	ordered := OrderIssues(issues)
	if len(ordered) != 4 {
		t.Fatalf("dedup failed: %+v", ordered)
	}
	want := []source.Issue{issues[4], issues[2], issues[1], issues[0]}
	for i := range want {
		if ordered[i].File != want[i].File || ordered[i].Span != want[i].Span || ordered[i].Code != want[i].Code {
			t.Fatalf("order[%d] = %+v, want %+v", i, ordered[i], want[i])
		}
	}
	// Input slice order is preserved for the caller (OrderIssues copies).
	if issues[0].File != "b.can" {
		t.Fatal("input reordered")
	}
}

func TestBuilderSealIsImmutableAndBlockedExplicit(t *testing.T) {
	builder := NewBuilder("/root")
	builder.AddSource(DocInput{Path: "a.can", Version: 1, Text: "ok"})
	builder.AddDisk(DiskInput{Label: "manifest", Value: "m1"})
	builder.AddIssue(source.Issue{File: "a.can", Span: source.Span{Start: 0, End: 1}, Code: "E", Severity: source.SeverityError, Message: "bad"})
	builder.SetUnit("a.can", UnitReport{Status: source.UnitInvalid})
	builder.SetUnit("dependent", UnitReport{Status: source.UnitBlocked, BlockedBy: []string{"a.can"}})
	builder.Index().Add(Occurrence{ID: "pkg::name", Kind: OccurrenceFunction, Visibility: VisibilityPackage, File: "a.can", DeclSpan: source.Span{Start: 0, End: 8}, NameSpan: source.Span{Start: 3, End: 7}, Valid: true})
	first := builder.Seal()
	if !first.Sealed() || !first.Errors() {
		t.Fatal("sealed analysis misreports")
	}
	if got := first.Unit("dependent"); got.Status != source.UnitBlocked || len(got.BlockedBy) != 1 {
		t.Fatalf("blocked unit = %+v", got)
	}
	if got := first.Unit("never-recorded"); got.Status != source.UnitBlocked {
		t.Fatalf("unknown unit reads valid: %+v", got)
	}
	// Mutating the builder after Seal must not affect the sealed value.
	builder.AddSource(DocInput{Path: "b.can", Version: 1, Text: "new"})
	builder.AddIssue(source.Issue{File: "b.can", Span: source.Span{Start: 0, End: 1}, Code: "E", Severity: source.SeverityError, Message: "later"})
	builder.SetUnit("a.can", UnitReport{Status: source.UnitValid})
	builder.Index().Add(Occurrence{ID: "pkg::other", Kind: OccurrenceValue, Visibility: VisibilityPackage, File: "b.can", DeclSpan: source.Span{Start: 0, End: 5}, NameSpan: source.Span{Start: 0, End: 5}, Valid: true})
	if len(first.Issues) != 1 || len(first.Sources) != 1 {
		t.Fatalf("sealed analysis mutated: %+v", first)
	}
	if got := first.Unit("a.can"); got.Status != source.UnitInvalid {
		t.Fatalf("sealed unit mutated: %+v", got)
	}
	if len(first.Index.Occurrences()) != 1 {
		t.Fatalf("sealed index mutated: %+v", first.Index.Occurrences())
	}
	second := builder.Seal()
	if len(second.Issues) != 2 || !first.Fingerprint.Equal(first.Fingerprint) || first.Fingerprint.Equal(second.Fingerprint) {
		t.Fatal("second seal identity wrong")
	}
}

func TestOccurrenceIndexQueriesNeverGuess(t *testing.T) {
	builder := NewIndexBuilder()
	builder.Add(Occurrence{ID: "", Kind: OccurrenceLocal, File: "a.can", NameSpan: source.Span{Start: 1, End: 2}, Valid: true})
	builder.Add(Occurrence{ID: "x", Kind: OccurrenceLocal, File: "a.can", NameSpan: source.Span{Start: 2, End: 2}, Valid: true})
	builder.Add(Occurrence{ID: "pkg::shadow", Kind: OccurrenceLocal, Visibility: VisibilityLocal, File: "a.can", DeclSpan: source.Span{Start: 0, End: 9}, NameSpan: source.Span{Start: 4, End: 9}, Scope: []string{"inner"}, Valid: true})
	builder.Add(Occurrence{ID: "pkg::shadow", Kind: OccurrenceLocal, Visibility: VisibilityLocal, File: "a.can", DeclSpan: source.Span{Start: 0, End: 9}, NameSpan: source.Span{Start: 20, End: 25}, Scope: []string{"inner"}, Valid: true})
	builder.Add(Occurrence{ID: "pkg::blocked", Kind: OccurrenceLocal, File: "a.can", NameSpan: source.Span{Start: 30, End: 36}, Valid: false})
	index := builder.Seal()
	if len(index.Occurrences()) != 3 {
		t.Fatalf("rejected occurrences indexed: %+v", index.Occurrences())
	}
	binding := index.Binding("pkg::shadow")
	if len(binding) != 2 || !binding[0].Declaration && binding[0].NameSpan.Start != 4 {
		t.Fatalf("binding = %+v", binding)
	}
	if len(index.Binding("pkg::blocked")) != 0 || len(index.Binding("missing")) != 0 {
		t.Fatal("invalid/unknown binding returned")
	}
	found, ok := index.At("a.can", 22)
	if !ok || found.ID != "pkg::shadow" {
		t.Fatalf("at = %+v, %v", found, ok)
	}
	if _, ok := index.At("a.can", 32); ok {
		t.Fatal("blocked occurrence queryable")
	}
	if _, ok := index.At("b.can", 22); ok {
		t.Fatal("wrong-file occurrence queryable")
	}
}
