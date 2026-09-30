package external

import "errors"

// Errors. Callers distinguish them with errors.Is.
var (
	ErrInvalid       = errors.New("external: invalid request")
	ErrForeign       = errors.New("external: foreign service target")
	ErrNotFound      = errors.New("external: unknown operation, grant or resource")
	ErrWrongOwner    = errors.New("external: wrong spawn-start identity")
	ErrDenied        = errors.New("external: refused")
	ErrChangedInput  = errors.New("external: operation replayed with different arguments")
	ErrIndeterminate = errors.New("external: effect may have occurred but outcome is unknown")
	ErrDependency    = errors.New("external: live dependent resource blocks release")
	ErrQuiesced      = errors.New("external: admission stopped")
	ErrCapacity      = errors.New("external: table capacity exhausted")
	ErrExpired       = errors.New("external: grant expired")
	ErrRevoked       = errors.New("external: grant revoked")
)

// Named recovery and release reasons. Recovery outcomes carry Touched,
// Denied, Foreign, NoEvidence or a caller scan reason; Released,
// ForgedRelease, Unresolved and QualUnqualified are reserved for
// receipt-matching callers. Callers match on the string in receipts.
const (
	ReasonTouched         = "touched"
	ReasonDenied          = "denied"
	ReasonForeign         = "foreign-target"
	ReasonNoEvidence      = "no-evidence"
	ReasonUnresolved      = "unresolved-release-missing"
	ReasonReleased        = "released"
	ReasonForgedRelease   = "forged-release"
	ReasonDependency      = "dependent-live"
	ReasonQualUnqualified = "unqualified"
)

// Reason maps a refusal error to its named reason. Unknown errors map to
// ReasonDenied: nothing unclassified ever reads as allowed.
func Reason(err error) string {
	switch {
	case errors.Is(err, ErrForeign):
		return ReasonForeign
	case errors.Is(err, ErrDependency):
		return ReasonDependency
	default:
		return ReasonDenied
	}
}
