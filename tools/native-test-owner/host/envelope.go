package host

import "fmt"

// Size units. Envelopes name GiB-scale budgets; the battery proves the
// accounting with small numbers and never allocates at envelope scale.
const (
	MiB = 1 << 20
	GiB = 1 << 30
)

// Reserve is the disposal capacity held back from the admission ceiling.
// A scope charged to its ceiling must still dispose: the reserve covers
// an external disposer process, its RSS headroom (bounded walker plus
// receipt writer), and receipt/journal spill on tmp.
type Reserve struct {
	Procs    int   `json:"procs"`
	MemBytes int64 `json:"memBytes"`
	TmpBytes int64 `json:"tmpBytes"`
}

// StandardDisposalReserve is the reserve both envelopes carry: one
// disposer proc slot, 64 MiB RSS headroom, 1 MiB receipt spill. These
// are judgments, not measurements: disposal walks a bounded tree and
// writes small receipts, so one slot plus tens of MiB is ample while
// staying negligible against GiB envelopes.
func StandardDisposalReserve() Reserve {
	return Reserve{Procs: 1, MemBytes: 64 * MiB, TmpBytes: 1 * MiB}
}

// Envelope is one finite resource budget with its disposal reserve.
// Charges are admitted only up to the ceiling (max minus reserve).
type Envelope struct {
	Name            string  `json:"name"`
	MaxProcs        int     `json:"maxProcs"`
	MaxMemBytes     int64   `json:"maxMemBytes"`
	MaxTmpBytes     int64   `json:"maxTmpBytes"`
	DisposalReserve Reserve `json:"reserve"`
}

// Quick is the quick envelope: 64 processes, 4 GiB, 512 MiB tmp.
func Quick() Envelope {
	return Envelope{
		Name: "quick", MaxProcs: 64, MaxMemBytes: 4 * GiB, MaxTmpBytes: 512 * MiB,
		DisposalReserve: StandardDisposalReserve(),
	}
}

// BoundedJob is the bounded-job envelope: 64 processes, 6 GiB, 2 GiB tmp.
func BoundedJob() Envelope {
	return Envelope{
		Name: "bounded-job", MaxProcs: 64, MaxMemBytes: 6 * GiB, MaxTmpBytes: 2 * GiB,
		DisposalReserve: StandardDisposalReserve(),
	}
}

// CeilingProcs is the admission ceiling: max minus reserve.
func (e Envelope) CeilingProcs() int { return e.MaxProcs - e.DisposalReserve.Procs }

// CeilingMem is the admission ceiling: max minus reserve.
func (e Envelope) CeilingMem() int64 { return e.MaxMemBytes - e.DisposalReserve.MemBytes }

// CeilingTmp is the admission ceiling: max minus reserve.
func (e Envelope) CeilingTmp() int64 { return e.MaxTmpBytes - e.DisposalReserve.TmpBytes }

// valid rejects empty names, non-positive maxima, non-positive reserves,
// and reserves that leave no admission room.
func (e Envelope) valid() error {
	if e.Name == "" {
		return fmt.Errorf("%w: envelope needs a name", ErrInvalid)
	}
	if e.MaxProcs <= 0 || e.MaxMemBytes <= 0 || e.MaxTmpBytes <= 0 {
		return fmt.Errorf("%w: envelope %q maxima must all be positive", ErrInvalid, e.Name)
	}
	r := e.DisposalReserve
	if r.Procs <= 0 || r.MemBytes <= 0 || r.TmpBytes <= 0 {
		return fmt.Errorf("%w: envelope %q needs a positive disposal reserve", ErrInvalid, e.Name)
	}
	if e.CeilingProcs() <= 0 || e.CeilingMem() <= 0 || e.CeilingTmp() <= 0 {
		return fmt.Errorf("%w: envelope %q reserve leaves no admission room", ErrInvalid, e.Name)
	}
	return nil
}

// controlEnvelope is the battery's small-numbers envelope. Refusal,
// detached-charging, bypass and reserve logic are proven here (ceilings
// 1 proc, 96 mem units, 96 tmp units); the real envelopes are proven by
// charged-scope arithmetic against probed host facts, never by forking
// 64 processes or allocating GiBs.
func controlEnvelope() Envelope {
	return Envelope{
		Name: "control", MaxProcs: 2, MaxMemBytes: 128, MaxTmpBytes: 128,
		DisposalReserve: Reserve{Procs: 1, MemBytes: 32, TmpBytes: 32},
	}
}

// memControlEnvelope is the memory-refusal envelope: proc headroom with
// the same 96-unit mem ceiling, so the over-mem charge is refused by
// the memory ledger and nothing else.
func memControlEnvelope() Envelope {
	return Envelope{
		Name: "control-mem", MaxProcs: 4, MaxMemBytes: 128, MaxTmpBytes: 128,
		DisposalReserve: Reserve{Procs: 1, MemBytes: 32, TmpBytes: 32},
	}
}
