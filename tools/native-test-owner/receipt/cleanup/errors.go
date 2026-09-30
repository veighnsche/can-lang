package cleanup

import "errors"

// Errors. Callers distinguish them with errors.Is.
var (
	ErrInvalid       = errors.New("cleanup: invalid request")
	ErrDenied        = errors.New("cleanup: refused")
	ErrWrongOwner    = errors.New("cleanup: wrong spawn-start identity")
	ErrWrongPath     = errors.New("cleanup: wrong or replaced path")
	ErrNotFound      = errors.New("cleanup: unknown operation")
	ErrUnknownSchema = errors.New("cleanup: unknown schema version")
	ErrRemoveFailed  = errors.New("cleanup: removal failed")
)

// Named refusal and completion reasons. Every receipt carries one of
// these; scan denials additionally carry the scanner's own named reason
// through unchanged.
const (
	ReasonOwnerMismatch   = "owner-mismatch"
	ReasonReplacedPath    = "replaced-path"
	ReasonUnknownSchema   = "unknown-schema-version"
	ReasonUnknownOp       = "unknown-operation"
	ReasonRemoveFailed    = "remove-failed"
	ReasonDenied          = "denied"
	ReasonCleanupComplete = "cleanup-complete"
)
