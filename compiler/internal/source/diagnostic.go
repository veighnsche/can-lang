package source

import "errors"

// SpanUnavailable marks a diagnostic whose structured span could not be
// converted to editor coordinates: the attributed file has no loaded bytes
// or the offsets are invalid for the loaded bytes. It keeps an unusable
// location explicit instead of silently anchoring to the file's first line.
const SpanUnavailable = "CAN-SPAN-UNAVAILABLE"

// Severities shared by check, driver and editor bridges. Notes are
// informational and must never be reclassified as warnings.
const (
	SeverityError   = "error"
	SeverityWarning = "warning"
	SeverityNote    = "note"
	SeverityHint    = "hint"
)

// UnitStatus is the per-unit validity carried by recoverable analysis: a unit
// is valid, invalid (its own diagnostics explain why), or blocked (a named
// prerequisite is unavailable, so no cascade diagnostics may be emitted).
type UnitStatus int

const (
	UnitValid UnitStatus = iota
	UnitInvalid
	UnitBlocked
)

// Issue is one positioned finding in original-byte coordinates. Bridges convert
// Span to zero-based UTF-16 exactly once at the editor boundary, preserving
// multiline end lines and zero-width insertion spans (including EOF).
type Issue struct {
	File     string
	Span     Span
	Code     string
	Severity string
	Message  string
	Related  []RelatedSpan
	Fixes    []Fix
}

// RelatedSpan is one secondary source location attached to a located
// failure: the canonical file path, byte offsets into that file, and a
// short note naming the relationship to the primary span.
type RelatedSpan struct {
	File string
	Span Span
	Note string
}

// LocatedError carries structured primary and related source spans for a
// resolver or checker failure. It preserves the wrapped CLI message
// byte-for-byte; bridges convert the spans without parsing prose.
type LocatedError struct {
	Code    string
	File    string
	Span    Span
	Related []RelatedSpan
	Fixes   []Fix
	Err     error
}

// Fix is one compiler-proposed source repair: a title plus a single
// replaced byte range of the diagnosed file. Compiler-generated fixes are
// insert-only (Start == End) so validation can prove nothing was deleted,
// suppressed or flattened; drivers must recheck every fix against an
// isolated overlay snapshot and present only fixes that typecheck with
// preserved contracts.
type Fix struct {
	Title   string
	File    string
	Start   int
	End     int
	Text    string
	Version int64
}

func (e *LocatedError) Error() string { return e.Err.Error() }
func (e *LocatedError) Unwrap() error { return e.Err }

// Locate attaches a primary file span to err. When the chain already
// carries a location, the innermost span wins and err is returned
// unchanged, so nested checking keeps the deepest expression position.
func Locate(file string, span Span, err error) error {
	return LocateCode(file, span, "", err)
}

// LocateCode attaches a primary file span plus a stable diagnostic code
// to err. The innermost location still wins; a second code never replaces
// the first, so nested checking keeps the deepest classification.
func LocateCode(file string, span Span, code string, err error) error {
	if err == nil || file == "" {
		return err
	}
	if _, ok := err.(*LocatedError); ok {
		return err
	}
	if origin, ok := err.(interface {
		SourceSpan() Span
		Unwrap() error
	}); ok {
		return LocateCode(file, origin.SourceSpan(), code, origin.Unwrap())
	}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		children := joined.Unwrap()
		located := make([]error, len(children))
		for i, child := range children {
			located[i] = LocateCode(file, span, code, child)
		}
		return errors.Join(located...)
	}
	if wrapped, ok := err.(interface{ Unwrap() error }); ok {
		return &locatedContext{message: err.Error(), cause: LocateCode(file, span, code, wrapped.Unwrap())}
	}
	if IsBlocked(err) {
		return err
	}
	return &LocatedError{File: file, Span: span, Code: code, Err: err}
}

// Suggest attaches one compiler-proposed repair to the located failure in
// err's chain. Without a located failure it returns err unchanged.
func Suggest(fix Fix, err error) error {
	if err == nil || fix.File == "" {
		return err
	}
	located, ok := AsLocated(err)
	if !ok {
		return err
	}
	located.Fixes = append(located.Fixes, fix)
	return err
}

// Relate appends a secondary span to the located failure in err's chain.
// Without a located failure it returns err unchanged.
func Relate(file string, span Span, note string, err error) error {
	if err == nil || file == "" {
		return err
	}
	located, ok := AsLocated(err)
	if !ok {
		return err
	}
	located.Related = append(located.Related, RelatedSpan{File: file, Span: span, Note: note})
	return err
}

// AsLocated returns the innermost located failure in err's chain.
func AsLocated(err error) (*LocatedError, bool) {
	var located *LocatedError
	ok := errors.As(err, &located)
	return located, ok
}

// BlockedError records a missing semantic prerequisite without manufacturing a
// second authored error at each use of that prerequisite.
type BlockedError struct{ Dependency string }

func (e *BlockedError) Error() string { return "blocked by invalid prerequisite " + e.Dependency }

// IsBlocked is true only when every leaf is an unavailable prerequisite.
// A joined direct error and blocked sibling must retain the direct finding.
func IsBlocked(err error) bool {
	if err == nil {
		return false
	}
	if _, ok := err.(*BlockedError); ok {
		return true
	}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		children := joined.Unwrap()
		if len(children) == 0 {
			return false
		}
		for _, child := range children {
			if !IsBlocked(child) {
				return false
			}
		}
		return true
	}
	if wrapped, ok := err.(interface{ Unwrap() error }); ok {
		return IsBlocked(wrapped.Unwrap())
	}
	return false
}

type locatedContext struct {
	message string
	cause   error
}

func (e *locatedContext) Error() string { return e.message }
func (e *locatedContext) Unwrap() error { return e.cause }

// SpanError retains an exact authored token before its owning source file is known.
type SpanError struct {
	Span Span
	Err  error
}

func (e *SpanError) Error() string    { return e.Err.Error() }
func (e *SpanError) Unwrap() error    { return e.Err }
func (e *SpanError) SourceSpan() Span { return e.Span }
