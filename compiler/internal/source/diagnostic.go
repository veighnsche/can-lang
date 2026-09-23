package source

// SpanUnavailable marks a diagnostic whose structured span could not be
// converted to editor coordinates: the attributed file has no loaded bytes
// or the offsets are invalid for the loaded bytes. It keeps an unusable
// location explicit instead of silently anchoring to the file's first line.
const SpanUnavailable = "CAN-SPAN-UNAVAILABLE"

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
	if _, ok := AsLocated(err); ok {
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
	for err != nil {
		if located, ok := err.(*LocatedError); ok {
			return located, true
		}
		unwrapper, ok := err.(interface{ Unwrap() error })
		if !ok {
			return nil, false
		}
		err = unwrapper.Unwrap()
	}
	return nil, false
}
