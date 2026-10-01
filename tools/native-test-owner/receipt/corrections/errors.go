package corrections

import "errors"

// Errors. Callers distinguish them with errors.Is.
var (
	ErrInvalid       = errors.New("corrections: invalid entry or file")
	ErrBrokenChain   = errors.New("corrections: chain breakage")
	ErrUnknownSchema = errors.New("corrections: unknown schema version")
)
