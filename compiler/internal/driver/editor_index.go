package driver

import (
	"sort"

	"github.com/veighnsche/can-lang/compiler/internal/source"
)

// Frozen occurrence-index contracts (F03). E30 builds this index from canonical
// compiler facts; every later editor feature queries it instead of rescanning
// sources or rerunning checks. All spans are original-byte offsets.

// OccurrenceKind names the declared entity behind an occurrence.
type OccurrenceKind string

const (
	OccurrenceFunction   OccurrenceKind = "function"
	OccurrenceValue      OccurrenceKind = "value"
	OccurrenceRecord     OccurrenceKind = "record"
	OccurrenceVariant    OccurrenceKind = "variant"
	OccurrenceError      OccurrenceKind = "error"
	OccurrenceField      OccurrenceKind = "field"
	OccurrenceInput      OccurrenceKind = "input"
	OccurrenceLocal      OccurrenceKind = "local"
	OccurrenceCapture    OccurrenceKind = "capture"
	OccurrencePin        OccurrenceKind = "pin"
	OccurrenceImport     OccurrenceKind = "import"
	OccurrenceProvides   OccurrenceKind = "provides"
	OccurrenceTypeParam  OccurrenceKind = "type-parameter"
	OccurrencePattern    OccurrenceKind = "pattern"
	OccurrenceNative     OccurrenceKind = "native"
	OccurrenceGenerated  OccurrenceKind = "generated"
	OccurrenceUnresolved OccurrenceKind = "unresolved"
)

// Visibility names the reference scope of a binding.
type Visibility string

const (
	VisibilityLocal     Visibility = "local"
	VisibilityPackage   Visibility = "package"
	VisibilityExported  Visibility = "exported"
	VisibilityImport    Visibility = "import"
	VisibilityGenerated Visibility = "generated"
)

// Occurrence is one indexed binding site: either a declaration or one use.
// ID is the stable binding identity shared by a declaration and all its uses;
// shadowing produces distinct IDs. Declaration spans locate the binding site;
// NameSpan is the exact token to jump to or rename. Proven-only rule: Type,
// Signature, and Arguments stay empty unless the checker proved them.
type Occurrence struct {
	ID          string
	Kind        OccurrenceKind
	Visibility  Visibility
	File        string
	DeclSpan    source.Span
	NameSpan    source.Span
	Scope       []string
	Type        string
	Signature   string
	Arguments   []source.Span
	Declaration bool
	Valid       bool
}

// OccurrenceIndex is the sealed, version-bound semantic index for one analysis.
// Occurrences are ordered by file and byte offsets; lookups never guess.
type OccurrenceIndex struct {
	occurrences []Occurrence
	byID        map[string][]int
}

// Occurrences returns all indexed occurrences in canonical order.
func (x *OccurrenceIndex) Occurrences() []Occurrence {
	if x == nil {
		return nil
	}
	out := make([]Occurrence, len(x.occurrences))
	for i, occurrence := range x.occurrences {
		out[i] = cloneOccurrence(occurrence)
	}
	return out
}

// Binding returns every valid occurrence sharing one binding identity: the
// declaration (when indexed) plus all uses. Unknown IDs return nothing.
func (x *OccurrenceIndex) Binding(id string) []Occurrence {
	if x == nil {
		return nil
	}
	var out []Occurrence
	for _, position := range x.byID[id] {
		if occurrence := x.occurrences[position]; occurrence.Valid {
			out = append(out, cloneOccurrence(occurrence))
		}
	}
	return out
}

// At returns the innermost valid occurrence whose name span contains the byte
// offset in the named file. Blocked or unresolved tokens return false.
func (x *OccurrenceIndex) At(file string, offset int) (Occurrence, bool) {
	if x == nil {
		return Occurrence{}, false
	}
	best := -1
	for i, occurrence := range x.occurrences {
		if !occurrence.Valid || occurrence.File != file {
			continue
		}
		if occurrence.NameSpan.Start <= offset && offset < occurrence.NameSpan.End {
			if best < 0 || occurrence.NameSpan.Start >= x.occurrences[best].NameSpan.Start &&
				occurrence.NameSpan.End <= x.occurrences[best].NameSpan.End {
				best = i
			}
		}
	}
	if best < 0 {
		return Occurrence{}, false
	}
	return cloneOccurrence(x.occurrences[best]), true
}

// IndexBuilder accumulates occurrences with private mutable state; Seal freezes
// the canonical order and identity map.
type IndexBuilder struct {
	occurrences []Occurrence
}

// NewIndexBuilder starts one empty occurrence collection.
func NewIndexBuilder() *IndexBuilder { return &IndexBuilder{} }

// Add records one occurrence. Zero ID or invalid name spans are rejected as
// unqueryable; callers must supply binding identity and real token spans.
func (b *IndexBuilder) Add(occurrence Occurrence) {
	if occurrence.ID == "" || occurrence.NameSpan.End <= occurrence.NameSpan.Start {
		return
	}
	b.occurrences = append(b.occurrences, cloneOccurrence(occurrence))
}

func cloneOccurrence(occurrence Occurrence) Occurrence {
	occurrence.Scope = append([]string(nil), occurrence.Scope...)
	occurrence.Arguments = append([]source.Span(nil), occurrence.Arguments...)
	return occurrence
}

// Seal orders occurrences by file and byte offsets and freezes the index.
func (b *IndexBuilder) Seal() *OccurrenceIndex {
	ordered := make([]Occurrence, len(b.occurrences))
	for i, occurrence := range b.occurrences {
		ordered[i] = cloneOccurrence(occurrence)
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].File != ordered[j].File {
			return ordered[i].File < ordered[j].File
		}
		if ordered[i].NameSpan.Start != ordered[j].NameSpan.Start {
			return ordered[i].NameSpan.Start < ordered[j].NameSpan.Start
		}
		return ordered[i].NameSpan.End < ordered[j].NameSpan.End
	})
	byID := map[string][]int{}
	for i, occurrence := range ordered {
		byID[occurrence.ID] = append(byID[occurrence.ID], i)
	}
	return &OccurrenceIndex{occurrences: ordered, byID: byID}
}
