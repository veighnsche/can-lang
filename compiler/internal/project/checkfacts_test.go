package project

import (
	"context"
	"errors"
	"os"
	"syscall"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/source"
)

func TestClassifyLoadError(t *testing.T) {
	notExist := &os.PathError{Op: "open", Path: "dep", Err: syscall.ENOENT}
	denied := &os.PathError{Op: "open", Path: "src", Err: syscall.EACCES}
	locatedContent := &source.LocatedError{Code: "CAN-PROJECT-CONFIG", File: "can.project.json", Err: errors.New("bad field")}
	locatedMissing := &source.LocatedError{Code: "CAN-PROJECT-CONFIG", File: "can.project.json", Err: notExist}
	cases := []struct {
		name    string
		err     error
		cause   LoadCause
		verdict LoadVerdict
	}{
		{"nil", nil, CauseNone, VerdictOK},
		{"source", &SourceError{Path: "src/a.can", Message: "bad"}, CauseContent, VerdictReject},
		{"located-content", locatedContent, CauseContent, VerdictReject},
		{"located-hides-missing", locatedMissing, CauseMissingInput, VerdictFail},
		{"graph", &GraphError{Kind: KindCycle, Msg: "cycle"}, CauseContent, VerdictReject},
		{"graph-hides-io", &GraphError{Kind: KindRegistryMismatch, Msg: "x", Cause: denied}, CauseDeniedRead, VerdictFail},
		{"missing", notExist, CauseMissingInput, VerdictFail},
		{"denied", denied, CauseDeniedRead, VerdictFail},
		{"dep-missing", &DependencyError{Edge: "dep", Dir: "dep", Err: notExist}, CauseDependencyUnavailable, VerdictFail},
		{"dep-located-missing", &DependencyError{Edge: "dep", Dir: "dep", Err: locatedMissing}, CauseDependencyUnavailable, VerdictFail},
		{"dep-content", &DependencyError{Edge: "dep", Dir: "dep", Err: locatedContent}, CauseContent, VerdictReject},
		{"unknown", errors.New("boom"), CauseUnknown, VerdictFail},
		{"join-infra-wins", errors.Join(&SourceError{Path: "a", Message: "x"}, notExist), CauseMissingInput, VerdictFail},
		{"cancelled", context.Canceled, CauseCancelled, VerdictFail},
	}
	for _, c := range cases {
		cause, verdict := ClassifyLoadError(c.err)
		if cause != c.cause || verdict != c.verdict {
			t.Errorf("%s: got (%d,%d), want (%d,%d)", c.name, cause, verdict, c.cause, c.verdict)
		}
	}
}

func TestTypedErrorsPreserveMessages(t *testing.T) {
	inner := errors.New("lstat dep: no such file or directory")
	dep := &DependencyError{Edge: "dep", Dir: "dep", Err: inner}
	if dep.Error() != inner.Error() {
		t.Fatalf("dependency message changed: %q", dep.Error())
	}
	graph := &GraphError{Kind: KindCycle, Msg: "dependency manifest cycle at /root"}
	if graph.Error() != "dependency manifest cycle at /root" {
		t.Fatalf("graph message changed: %q", graph.Error())
	}
}

func TestReadFactsSortAndMarkAbsent(t *testing.T) {
	g := &Graph{Inputs: map[string][]byte{"/b": nil, "/a": []byte("xy")}}
	facts := g.ReadFacts()
	if len(facts) != 2 || facts[0].Path != "/a" || !facts[1].Absent {
		t.Fatalf("facts = %+v", facts)
	}
	if facts[0].Length != 2 || facts[0].SHA256 == "" {
		t.Fatalf("read fact = %+v", facts[0])
	}
	if ReadFacts := (&Graph{}).ReadFacts(); len(ReadFacts) != 0 {
		t.Fatalf("empty graph facts = %+v", ReadFacts)
	}
	var nilGraph *Graph
	if nilGraph.ReadFacts() != nil || nilGraph.SortedEnumerated() != nil {
		t.Fatal("nil graph must yield nil facts")
	}
}
