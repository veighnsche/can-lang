package dispatch

import "errors"

// Mechanical errors. Callers distinguish them with errors.Is; service-level
// replies additionally map them onto the operation result outcome/kind
// vocabulary.
var (
	ErrInvalid       = errors.New("dispatch: invalid request")
	ErrChangedInput  = errors.New("dispatch: operation replayed with different arguments")
	ErrIndeterminate = errors.New("dispatch: effect may have occurred but outcome is unknown")
	ErrNotFound      = errors.New("dispatch: unknown operation, grant or handle")
	ErrExpired       = errors.New("dispatch: grant expired")
	ErrRevoked       = errors.New("dispatch: grant revoked")
	ErrStaleHandle   = errors.New("dispatch: stale handle generation")
	ErrClosedHandle  = errors.New("dispatch: handle is closed")
	ErrWrongKind     = errors.New("dispatch: wrong handle kind")
	ErrQueueFull     = errors.New("dispatch: channel queue full")
	ErrEmpty         = errors.New("dispatch: channel empty")
	ErrClosed        = errors.New("dispatch: channel closed")
	ErrCapacity      = errors.New("dispatch: table capacity exhausted")
)
