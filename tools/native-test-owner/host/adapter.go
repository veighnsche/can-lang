package host

import "errors"

// Errors. Callers distinguish them with errors.Is.
var (
	ErrInvalid      = errors.New("host: invalid request")
	ErrUnavailable  = errors.New("host: enforcement unavailable")
	ErrNotQualified = errors.New("host: no valid qualification")
	ErrOverEnvelope = errors.New("host: charge refused: over envelope")
	ErrOverflow     = errors.New("host: uncharged bytes exceed envelope")
	ErrReleased     = errors.New("host: scope already disposed")
	ErrNoFit        = errors.New("host: envelope does not fit host")
	ErrMechanism    = errors.New("host: non-qualifying enforcement mechanism")
)

// Named refusal reasons. Every host refusal carries one; callers match
// on the string in receipts. Reason maps refusal errors to reasons.
const (
	ReasonUnavailable  = "unavailable"
	ReasonNotQualified = "not-qualified"
	ReasonOverEnvelope = "over-envelope"
	ReasonOverflow     = "bypass-overflow"
	ReasonNoFit        = "envelope-no-fit"
	ReasonDenied       = "denied"
)

// Reason maps a host error to its named reason. Unknown errors map to
// ReasonDenied: nothing unclassified ever reads as admitted.
func Reason(err error) string {
	switch {
	case errors.Is(err, ErrUnavailable):
		return ReasonUnavailable
	case errors.Is(err, ErrNotQualified):
		return ReasonNotQualified
	case errors.Is(err, ErrOverEnvelope):
		return ReasonOverEnvelope
	case errors.Is(err, ErrOverflow):
		return ReasonOverflow
	case errors.Is(err, ErrNoFit):
		return ReasonNoFit
	default:
		return ReasonDenied
	}
}

// Mechanism names the enforcement mechanism class an adapter claims.
// The claim is checked, but behavior decides: the battery fails adapters
// that admit over the envelope whatever they declare (see the lying
// negative control).
type Mechanism string

const (
	// MechRefuseBeforeEffect is synchronous pre-effect refusal: the
	// over-limit operation fails before any effect exists. Required.
	MechRefuseBeforeEffect Mechanism = "refuse-before-effect"
	// MechResourceLimit is a kernel resource-limit backstop whose
	// refusal is demonstrated live (rlimit fork/write refusal).
	// Required alongside refuse-before-effect.
	MechResourceLimit Mechanism = "resource-limit"
	// MechPollKill admits then asynchronously kills. Never qualifies.
	MechPollKill Mechanism = "poll-and-kill"
	// MechSampledPeak admits then observes peaks. Never qualifies.
	MechSampledPeak Mechanism = "sampled-peak"
)

// HostFacts are probed kernel/host facts. Zero values mean unknown, and
// unknown fails closed: an envelope is admitted only where every fit
// comparison is demonstrated against a known fact.
type HostFacts struct {
	GOOS            string `json:"goos"`
	NProcSoft       uint64 `json:"nprocSoft"`
	NProcHard       uint64 `json:"nprocHard"`
	AddrSpaceCapped bool   `json:"addrSpaceCapped"`
	FileSizeCapped  bool   `json:"fileSizeCapped"`
	MemBytes        uint64 `json:"memBytes"`
	TmpFreeBytes    uint64 `json:"tmpFreeBytes"`
}

// Capabilities are one adapter's probe outcome.
type Capabilities struct {
	Available  bool
	Mechanisms []Mechanism
	Facts      HostFacts
	Detail     string
}

// Usage is the charged scope: exact ledger totals. Detached spawns
// charge identically to attached ones; bypass tmp bytes are absorbed
// at every decision-point reconcile.
type Usage struct {
	Procs         int
	DetachedProcs int
	MemBytes      int64
	TmpBytes      int64
}

// Overshoot states the finite bounds. Strict scopes refuse at the
// ceiling with zero mediated overshoot; the only nonzero bound is the
// physical ceiling on uncharged bypass bytes between decision-point
// reconciles. Unbounded (sampling) never qualifies.
type Overshoot struct {
	MaxProcsOver       int64 `json:"maxProcsOver"`
	MaxMemOver         int64 `json:"maxMemOver"`
	MaxTmpMediatedOver int64 `json:"maxTmpMediatedOver"`
	TmpBypassBound     int64 `json:"tmpBypassBound"`
	Unbounded          bool  `json:"unbounded"`
}

// Finite reports whether every bound is stated and finite.
func (o Overshoot) Finite() bool {
	return !o.Unbounded &&
		o.MaxProcsOver >= 0 && o.MaxMemOver >= 0 &&
		o.MaxTmpMediatedOver >= 0 && o.TmpBypassBound > 0
}

// TmpFacts are the reconcile outcome: charged ledger versus rescan
// observation. BypassCharged is the delta absorbed from writes that
// bypassed the helpers.
type TmpFacts struct {
	Charged       int64
	Observed      int64
	BypassCharged int64
	Files         int
}

// ProcToken authorizes one spawn. It is minted by ChargeProc before fork
// and must be presented to ReleaseProc; tokens never transfer between
// scopes. The zero token is never valid.
type ProcToken struct {
	id       uint64
	detached bool
	mem      int64
}

// Detached reports whether the authorized spawn is detached (Setsid).
// Detached spawns charge identically; the flag exists for evidence.
func (t ProcToken) Detached() bool { return t.detached }

// MemBytes reports the declared memory charge the token holds.
func (t ProcToken) MemBytes() int64 { return t.mem }

// Adapter probes host capabilities and opens charged scopes. Probe
// takes the tmp parent so free-disk facts are measured where tmp will
// live; OpenScope takes the envelope plus the tmp parent for the same
// reason. Both parents must be absolute existing directories.
type Adapter interface {
	Name() string
	Probe(tmpParent string) Capabilities
	OpenScope(env Envelope, tmpParent string) (Scope, error)
}

// Scope is one open charged scope. Charge calls authorize effects
// BEFORE they happen and refuse synchronously over the ceiling; every
// charge path reconciles tmp first, so no admission verdict is ever
// issued on unreconciled state. A Scope is safe for concurrent use.
type Scope interface {
	Envelope() Envelope
	Usage() Usage
	Overshoot() Overshoot
	// TmpRoot is the owned tmp root: helper-mediated writes land here
	// after ChargeTmp, and bypass writes here are caught by reconcile.
	TmpRoot() string
	// ChargeProc authorizes one spawn (detached or not) plus its
	// declared memory peak. Over the ceiling it fails with
	// ErrOverEnvelope and no token exists: the caller must not fork.
	ChargeProc(detached bool, memBytes int64) (ProcToken, error)
	ReleaseProc(t ProcToken) error
	// ChargeTmp authorizes n bytes before the write. Over the ceiling
	// it fails with ErrOverEnvelope and the caller must not write.
	ChargeTmp(n int64) error
	ReleaseTmp(n int64) error
	// ReconcileTmp rescans the tmp root, charges bypass bytes, and
	// fails closed with ErrOverflow when uncharged bytes push the
	// scope past its max. Past overflow, charges refuse until Dispose.
	ReconcileTmp() (TmpFacts, error)
	// Dispose removes all tmp contents and releases every charge. It
	// succeeds at full charge via the disposal reserve and is
	// idempotent; a failed removal keeps the scope undisposed.
	Dispose() error
}
