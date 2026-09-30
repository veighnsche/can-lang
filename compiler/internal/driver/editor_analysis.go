package driver

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/project"
	compileresolve "github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/source"
)

// Frozen recoverable-analysis contracts (F03). Dependent lanes (P/R/E/S) build
// on these exact types; the coordinator owns changes to them until H75.
//
// Ownership rules:
//   - Analysis is immutable after Seal. Builders hold the only mutable state
//     and must never publish it before Seal.
//   - Graph/World pointers inside Analysis are partial results for read-only
//     queries; mutation after Seal is forbidden.
//   - Issues are in original-byte coordinates, deterministically ordered and
//     deduplicated by Seal. UTF-16 conversion happens once at the editor
//     boundary (see diagnostics.go bridges).

// DocInput is one text input to analysis identity: an open document version
// plus the exact text behind it. Unsaved overlays and disk files share this
// shape; the fingerprint hashes the text so version reuse cannot collide.
type DocInput struct {
	Path    string
	Version int64
	Text    string
}

// DiskInput is one non-document analysis input: manifest, lock, registry,
// asset, or dependency source identity. Value is an opaque content identity
// (content hash or canonical path+digest); empty means absent.
type DiskInput struct {
	Label string
	Value string
}

// Fingerprint is the complete analysis identity: every relevant document
// version+text plus every relevant disk input. A scalar root version is
// insufficient. Fingerprints compare with Equal; maps stay private so sealed
// values cannot be mutated.
type Fingerprint struct {
	docs []DocInput
	disk []DiskInput
}

// NewFingerprint builds the canonical identity from document and disk inputs.
// Inputs are copied and canonically ordered; callers retain their slices.
func NewFingerprint(docs []DocInput, disk []DiskInput) Fingerprint {
	orderedDocs := append([]DocInput(nil), docs...)
	sort.Slice(orderedDocs, func(i, j int) bool { return orderedDocs[i].Path < orderedDocs[j].Path })
	orderedDisk := append([]DiskInput(nil), disk...)
	sort.Slice(orderedDisk, func(i, j int) bool { return orderedDisk[i].Label < orderedDisk[j].Label })
	return Fingerprint{docs: orderedDocs, disk: orderedDisk}
}

// Equal reports whether two fingerprints name identical inputs, comparing
// exact document text (not versions alone) and every disk identity.
func (f Fingerprint) Equal(other Fingerprint) bool {
	if len(f.docs) != len(other.docs) || len(f.disk) != len(other.disk) {
		return false
	}
	for i := range f.docs {
		if f.docs[i] != other.docs[i] {
			return false
		}
	}
	for i := range f.disk {
		if f.disk[i] != other.disk[i] {
			return false
		}
	}
	return true
}

// Digest is a compact content hash of the full fingerprint for cache keys and
// logs. Equality stays authoritative; digests never replace Equal.
func (f Fingerprint) Digest() string {
	h := sha256.New()
	for _, d := range f.docs {
		h.Write([]byte(d.Path))
		h.Write([]byte{0})
		text := sha256.Sum256([]byte(d.Text))
		h.Write(text[:])
		var version [8]byte
		for i := 0; i < 8; i++ {
			version[i] = byte(d.Version >> (8 * i))
		}
		h.Write(version[:])
	}
	for _, d := range f.disk {
		h.Write([]byte(d.Label))
		h.Write([]byte{0})
		h.Write([]byte(d.Value))
		h.Write([]byte{0})
	}
	sum := h.Sum(nil)
	return hex.EncodeToString(sum)
}

// Docs returns a copy of the ordered document inputs.
func (f Fingerprint) Docs() []DocInput { return append([]DocInput(nil), f.docs...) }

// Disk returns a copy of the ordered disk inputs.
func (f Fingerprint) Disk() []DiskInput { return append([]DiskInput(nil), f.disk...) }

// UnitID names one validity-tracked unit: a source path, a declaration
// identity, or a synthetic phase key such as "project:manifest".
type UnitID string

// UnitReport is the per-unit validity outcome. Invalid units have their own
// issues in Analysis.Issues; blocked units name the missing prerequisites and
// must not contribute cascade diagnostics.
type UnitReport struct {
	Status    source.UnitStatus
	BlockedBy []string
}

// Analysis is one sealed recoverable analysis: immutable inputs, complete
// fingerprint, partial graph/world for read-only queries, ordered issues,
// per-unit validity, and the shared semantic index. The zero value is invalid;
// only Builder.Seal produces usable values.
type Analysis struct {
	Program     *check.Program
	Root        string
	Fingerprint Fingerprint
	Sources     map[string]DocInput
	Graph       *project.Graph
	World       *compileresolve.World
	Issues      []source.Issue
	Units       map[UnitID]UnitReport
	Index       *OccurrenceIndex
	sealed      bool
}

// Sealed reports whether this Analysis came from Builder.Seal.
func (a *Analysis) Sealed() bool { return a != nil && a.sealed }

// Errors reports whether any issue has error severity.
func (a *Analysis) Errors() bool {
	if a == nil {
		return true
	}
	for _, issue := range a.Issues {
		if issue.Severity == source.SeverityError {
			return true
		}
	}
	return false
}

// Unit returns the validity report for one unit, or blocked-by-analysis when
// the unit was never recorded (unknown units never read as valid).
func (a *Analysis) Unit(id UnitID) UnitReport {
	if a != nil {
		if report, ok := a.Units[id]; ok {
			report.BlockedBy = append([]string(nil), report.BlockedBy...)
			return report
		}
	}
	return UnitReport{Status: source.UnitBlocked, BlockedBy: []string{"analysis:unknown-unit"}}
}

// Builder accumulates one analysis with private mutable state. It is not safe
// for concurrent use; the server lane serializes all builders on one worker.
type Builder struct {
	root    string
	sources map[string]DocInput
	disk    []DiskInput
	graph   *project.Graph
	world   *compileresolve.World
	issues  []source.Issue
	units   map[UnitID]UnitReport
	index   *IndexBuilder
	sealed  bool
}

// NewBuilder starts one analysis for a project root.
func NewBuilder(root string) *Builder {
	return &Builder{root: root, sources: map[string]DocInput{}, units: map[UnitID]UnitReport{}}
}

// AddSource records one document input. Later calls for the same path replace
// earlier ones; Seal copies the map.
func (b *Builder) AddSource(input DocInput) {
	b.sources[input.Path] = input
}

// AddDisk records one disk input.
func (b *Builder) AddDisk(input DiskInput) { b.disk = append(b.disk, input) }

// SetGraph records the partial graph for read-only queries.
func (b *Builder) SetGraph(graph *project.Graph) { b.graph = graph }

// SetWorld records the partial world for read-only queries.
func (b *Builder) SetWorld(world *compileresolve.World) { b.world = world }

// AddIssue records one positioned finding in original-byte coordinates.
func (b *Builder) AddIssue(issue source.Issue) { b.issues = append(b.issues, issue) }

// SetUnit records one unit validity outcome.
func (b *Builder) SetUnit(id UnitID, report UnitReport) {
	report.BlockedBy = append([]string(nil), report.BlockedBy...)
	b.units[id] = report
}

// Index exposes the private occurrence-index builder. Callers add occurrences
// during analysis; Seal freezes the result.
func (b *Builder) Index() *IndexBuilder {
	if b.index == nil {
		b.index = NewIndexBuilder()
	}
	return b.index
}

// Seal deterministically orders and deduplicates issues, freezes the
// fingerprint, index, and all maps, and returns the immutable Analysis.
// Further Builder calls do not affect sealed values.
func (b *Builder) Seal() *Analysis {
	docs := make([]DocInput, 0, len(b.sources))
	for _, input := range b.sources {
		docs = append(docs, input)
	}
	sources := make(map[string]DocInput, len(b.sources))
	for path, input := range b.sources {
		sources[path] = input
	}
	units := make(map[UnitID]UnitReport, len(b.units))
	for id, report := range b.units {
		report.BlockedBy = append([]string(nil), report.BlockedBy...)
		units[id] = report
	}
	var index *OccurrenceIndex
	if b.index != nil {
		index = b.index.Seal()
	}
	analysis := &Analysis{
		Root:        b.root,
		Fingerprint: NewFingerprint(docs, b.disk),
		Sources:     sources,
		Graph:       b.graph,
		World:       b.world,
		Issues:      OrderIssues(b.issues),
		Units:       units,
		Index:       index,
		sealed:      true,
	}
	b.sealed = true
	return analysis
}

// Sealed reports whether Seal has been called (further mutation still only
// affects future Seal results, never past ones).
func (b *Builder) Sealed() bool { return b.sealed }

// OrderIssues returns the canonical issue order with exact duplicates removed:
// file, byte start/end, code, severity, message, then related/fix contents.
func OrderIssues(issues []source.Issue) []source.Issue {
	ordered := make([]source.Issue, len(issues))
	key := func(issue source.Issue) string { data, _ := json.Marshal(issue); return string(data) }
	for i, issue := range issues {
		issue.Related = append([]source.RelatedSpan(nil), issue.Related...)
		issue.Fixes = append([]source.Fix(nil), issue.Fixes...)
		ordered[i] = issue
	}
	sort.Slice(ordered, func(i, j int) bool {
		a, b := ordered[i], ordered[j]
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Span.Start != b.Span.Start {
			return a.Span.Start < b.Span.Start
		}
		if a.Span.End != b.Span.End {
			return a.Span.End < b.Span.End
		}
		return key(a) < key(b)
	})
	out := make([]source.Issue, 0, len(ordered))
	for _, issue := range ordered {
		if len(out) == 0 || !issueEqual(issue, out[len(out)-1]) {
			out = append(out, issue)
		}
	}
	return out
}

func issueEqual(a, b source.Issue) bool {
	if a.File != b.File || a.Span != b.Span || a.Code != b.Code || a.Severity != b.Severity || a.Message != b.Message {
		return false
	}
	if len(a.Related) != len(b.Related) || len(a.Fixes) != len(b.Fixes) {
		return false
	}
	for i := range a.Related {
		if a.Related[i] != b.Related[i] {
			return false
		}
	}
	for i := range a.Fixes {
		if a.Fixes[i] != b.Fixes[i] {
			return false
		}
	}
	return true
}
