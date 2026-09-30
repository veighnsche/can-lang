package recovery

import "errors"

// Errors. Callers distinguish them with errors.Is.
var (
	ErrInvalid         = errors.New("recovery: invalid request")
	ErrAdmissionClosed = errors.New("recovery: admission stopped")
	ErrDenied          = errors.New("recovery: refused")
	ErrUnknownSchema   = errors.New("recovery: unknown schema version")
	ErrOwnerMismatch   = errors.New("recovery: owner mismatch")
	ErrForeignProcess  = errors.New("recovery: foreign process live")
	ErrReplacedPath    = errors.New("recovery: path replaced")
	ErrIncompleteProof = errors.New("recovery: incomplete liveness proof")
)

// Named refusal and drain reasons. Every scan refusal and every drain
// outcome carries one of these; callers match on the string in receipts.
const (
	ReasonOwnerMismatch   = "owner-mismatch"
	ReasonForeignLive     = "foreign-process-live"
	ReasonReplacedPath    = "replaced-path"
	ReasonUnknownSchema   = "unknown-schema-version"
	ReasonIncompleteProof = "incomplete-liveness-proof"
	ReasonDenied          = "denied"
	ReasonDrained         = "drained"
	ReasonDrainEscalated  = "drain-escalated"
	ReasonDrainFailed     = "drain-failed"
	ReasonDrainCap        = "drain-cap-exceeded"
)

// Reason maps a refusal error to its named reason. Unknown errors map to
// ReasonDenied: nothing unclassified ever reads as allowed.
func Reason(err error) string {
	switch {
	case errors.Is(err, ErrUnknownSchema):
		return ReasonUnknownSchema
	case errors.Is(err, ErrForeignProcess):
		return ReasonForeignLive
	case errors.Is(err, ErrOwnerMismatch):
		return ReasonOwnerMismatch
	case errors.Is(err, ErrReplacedPath):
		return ReasonReplacedPath
	case errors.Is(err, ErrIncompleteProof):
		return ReasonIncompleteProof
	default:
		return ReasonDenied
	}
}
