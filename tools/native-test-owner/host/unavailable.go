package host

import "fmt"

// unavailableAdapter is the explicit no-mechanism host. It probes
// unavailable and refuses every scope: an unqualified host never
// admits an envelope.
type unavailableAdapter struct {
	reason string
}

// Unavailable returns the blocking adapter with its evidence reason.
func Unavailable(reason string) Adapter {
	if reason == "" {
		reason = "no enforcement mechanism on this host"
	}
	return unavailableAdapter{reason: reason}
}

// Name reports the adapter identity for receipts.
func (unavailableAdapter) Name() string { return "unavailable" }

// Probe always reports unavailable.
func (u unavailableAdapter) Probe(_ string) Capabilities {
	return Capabilities{Available: false, Detail: u.reason}
}

// OpenScope always refuses with ErrUnavailable.
func (u unavailableAdapter) OpenScope(_ Envelope, _ string) (Scope, error) {
	return nil, fmt.Errorf("%w: %s", ErrUnavailable, u.reason)
}
