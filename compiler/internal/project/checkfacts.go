package project

import (
	"context"
	"errors"
	"os"
	"sort"

	"github.com/veighnsche/can-lang/compiler/internal/source"
)

// GraphErrorKind names a project-shape problem detected by the loader. All
// GraphError leaves are content problems (malformed project/manifest/source
// arrangement), never infrastructure failures.
type GraphErrorKind string

const (
	KindCycle            GraphErrorKind = "dependency-cycle"
	KindTooDeep          GraphErrorKind = "graph-too-deep"
	KindLineage          GraphErrorKind = "lineage-divergent"
	KindPackageConflict  GraphErrorKind = "package-conflict"
	KindOutputCollision  GraphErrorKind = "output-collision"
	KindEscape           GraphErrorKind = "path-escape"
	KindBadPath          GraphErrorKind = "bad-path"
	KindNotContainer     GraphErrorKind = "not-container"
	KindRegistryMismatch GraphErrorKind = "registry-mismatch"
	KindUTF8             GraphErrorKind = "non-utf8-path"
	KindLockMismatch     GraphErrorKind = "lock-mismatch"
)

// GraphError types a project-shape failure while preserving the exact CLI
// message byte-for-byte: Error returns Msg unchanged, so existing output and
// goldens are unaffected. Path names the offending manifest or source path
// when known.
type GraphError struct {
	Kind  GraphErrorKind
	Path  string
	Msg   string
	Cause error
}

func (e *GraphError) Error() string { return e.Msg }
func (e *GraphError) Unwrap() error { return e.Cause }

// DependencyError attributes a dependency-subtree load failure to its edge
// without changing the message: Error returns the inner message exactly.
type DependencyError struct {
	Edge string
	Dir  string
	Err  error
}

func (e *DependencyError) Error() string { return e.Err.Error() }
func (e *DependencyError) Unwrap() error { return e.Err }

// contentOrRaw wraps err as a content GraphError unless its chain carries an
// OS cause, which stays raw so infrastructure failures are never reported as
// source content. Msg preserves err text exactly.
func contentOrRaw(kind GraphErrorKind, path string, err error) error {
	if err == nil {
		return nil
	}
	var pathErr *os.PathError
	var syscallErr *os.SyscallError
	if errors.Is(err, os.ErrNotExist) || errors.Is(err, os.ErrPermission) ||
		errors.As(err, &pathErr) || errors.As(err, &syscallErr) {
		return err
	}
	return &GraphError{Kind: kind, Path: path, Msg: err.Error(), Cause: err}
}

// LoadCause classifies one load failure leaf for check reporting.
type LoadCause int

const (
	CauseNone LoadCause = iota
	CauseContent
	CauseMissingInput
	CauseDeniedRead
	CauseDependencyUnavailable
	CauseIOError
	CauseCancelled
	CauseUnknown
)

// LoadVerdict is the check-level meaning of a load error tree.
type LoadVerdict int

const (
	VerdictOK LoadVerdict = iota
	VerdictReject
	VerdictFail
)

// ClassifyLoadError walks an error tree from Load and reports the check
// verdict plus the first infrastructure cause. Content leaves (source,
// manifest/registry/dependency content, project-shape errors) yield
// VerdictReject; OS, cancellation and unknown leaves yield VerdictFail.
// A dependency edge carrying a missing/unreadable input yields
// CauseDependencyUnavailable. Classification uses types only, never message
// text.
func ClassifyLoadError(err error) (LoadCause, LoadVerdict) {
	if err == nil {
		return CauseNone, VerdictOK
	}
	state := &classifyState{}
	walkLoadError(err, false, state)
	if state.infra != CauseNone {
		return state.infra, VerdictFail
	}
	if state.content {
		return CauseContent, VerdictReject
	}
	return CauseNone, VerdictOK
}

type classifyState struct {
	infra      LoadCause
	knownInfra bool
	content    bool
}

func (s *classifyState) setInfra(cause LoadCause) {
	if s.infra == CauseNone {
		s.infra = cause
	}
	if cause != CauseUnknown {
		s.knownInfra = true
	}
}

func walkLoadError(err error, inDependency bool, s *classifyState) {
	if err == nil {
		return
	}
	if many, ok := err.(interface{ Unwrap() []error }); ok {
		for _, child := range many.Unwrap() {
			walkLoadError(child, inDependency, s)
		}
		return
	}
	if dep, ok := err.(*DependencyError); ok {
		walkLoadError(dep.Err, true, s)
		return
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		s.setInfra(CauseCancelled)
		return
	}
	var src *SourceError
	if errors.As(err, &src) {
		s.content = true
		return
	}
	var located *source.LocatedError
	if errors.As(err, &located) {
		// A located error attributes content, but its cause may still be
		// infrastructure (for example a manifest field naming a missing
		// directory): infrastructure inside wins, otherwise content.
		if located.Unwrap() != nil {
			inner := &classifyState{}
			walkLoadError(located.Unwrap(), inDependency, inner)
			if inner.knownInfra {
				s.setInfra(inner.infra)
				return
			}
		}
		s.content = true
		return
	}
	var graph *GraphError
	if errors.As(err, &graph) {
		if graph.Cause != nil {
			inner := &classifyState{}
			walkLoadError(graph.Cause, inDependency, inner)
			if inner.knownInfra {
				s.setInfra(inner.infra)
				return
			}
		}
		s.content = true
		return
	}
	if errors.Is(err, os.ErrNotExist) {
		if inDependency {
			s.setInfra(CauseDependencyUnavailable)
		} else {
			s.setInfra(CauseMissingInput)
		}
		return
	}
	if errors.Is(err, os.ErrPermission) {
		if inDependency {
			s.setInfra(CauseDependencyUnavailable)
		} else {
			s.setInfra(CauseDeniedRead)
		}
		return
	}
	var pathErr *os.PathError
	var syscallErr *os.SyscallError
	if errors.As(err, &pathErr) || errors.As(err, &syscallErr) {
		if inDependency {
			s.setInfra(CauseDependencyUnavailable)
		} else {
			s.setInfra(CauseIOError)
		}
		return
	}
	if wrapped, ok := err.(interface{ Unwrap() error }); ok {
		if inner := wrapped.Unwrap(); inner != nil {
			walkLoadError(inner, inDependency, s)
			return
		}
	}
	// Unknown leaves stay unknown even under a dependency edge: only known
	// OS causes upgrade to dependency-unavailable, and content attribution
	// (LocatedError/GraphError) otherwise stands.
	s.setInfra(CauseUnknown)
}

// ReadFact is one actual input read (or failed lookup) from the load: the
// canonical path, byte length and content digest when bytes were read, or
// Absent when the lookup found nothing.
type ReadFact struct {
	Path   string
	Length int
	SHA256 string
	Absent bool
}

// ReadFacts returns the actual-read set sorted by path. Inputs entries with
// nil bytes are absent/failed lookups, never empty successful reads.
func (g *Graph) ReadFacts() []ReadFact {
	if g == nil {
		return nil
	}
	facts := make([]ReadFact, 0, len(g.Inputs))
	for path, data := range g.Inputs {
		if data == nil {
			facts = append(facts, ReadFact{Path: path, Absent: true})
			continue
		}
		facts = append(facts, ReadFact{Path: path, Length: len(data), SHA256: Digest(data)})
	}
	sort.Slice(facts, func(i, k int) bool { return facts[i].Path < facts[k].Path })
	return facts
}

// EnumeratedDir records one source-directory enumeration: the canonical
// directory and the entry count observed there.
type EnumeratedDir struct {
	Dir     string
	Entries int
}

// SortedEnumerated returns a sorted copy of the recorded enumerations.
func (g *Graph) SortedEnumerated() []EnumeratedDir {
	if g == nil {
		return nil
	}
	out := append([]EnumeratedDir(nil), g.Enumerated...)
	sort.Slice(out, func(i, k int) bool { return out[i].Dir < out[k].Dir })
	return out
}

// ErrorLeaves flattens a load error tree to its leaves for diagnostic
// attribution. Join nodes are expanded; single-wrap chains are kept whole
// so attribution sees the outermost typed layer first.
func ErrorLeaves(err error) []error {
	if err == nil {
		return nil
	}
	if many, ok := err.(interface{ Unwrap() []error }); ok {
		var out []error
		for _, child := range many.Unwrap() {
			out = append(out, ErrorLeaves(child)...)
		}
		return out
	}
	return []error{err}
}
