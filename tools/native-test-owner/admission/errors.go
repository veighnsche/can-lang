package admission

import "errors"

// Errors. Callers distinguish them with errors.Is.
var (
	ErrInvalid          = errors.New("admission: invalid request")
	ErrLaneBusy         = errors.New("admission: lane busy")
	ErrNestedDemand     = errors.New("admission: nested demand")
	ErrNoFit            = errors.New("admission: body plus cleanup cannot fit deadline")
	ErrDeadlineExceeded = errors.New("admission: deadline exceeded")
	ErrReleased         = errors.New("admission: already released")
	ErrBelowFloor       = errors.New("admission: available disk below floor")
)

// Named refusal reasons. Every admission refusal and every failed
// completion carries one of these; callers match on the string in
// receipts. Reason maps refusal errors to reasons.
const (
	ReasonLaneBusy         = "lane-busy"
	ReasonNestedDemand     = "nested-demand"
	ReasonNoFit            = "body-cleanup-no-fit"
	ReasonDeadlineExceeded = "deadline-exceeded"
	ReasonBelowFloor       = "below-disk-floor"
	ReasonDenied           = "denied"
)

// Reason maps an admission error to its named reason. Unknown errors map
// to ReasonDenied: nothing unclassified ever reads as admitted.
func Reason(err error) string {
	switch {
	case errors.Is(err, ErrLaneBusy):
		return ReasonLaneBusy
	case errors.Is(err, ErrNestedDemand):
		return ReasonNestedDemand
	case errors.Is(err, ErrNoFit):
		return ReasonNoFit
	case errors.Is(err, ErrDeadlineExceeded):
		return ReasonDeadlineExceeded
	case errors.Is(err, ErrBelowFloor):
		return ReasonBelowFloor
	default:
		return ReasonDenied
	}
}
