package recovery

import (
	"fmt"
	"sort"
	"syscall"
	"time"

	"github.com/veighnsche/can-lang/tools/native-test-owner/process"
)

// Transitional bounds. Every field is finite; nothing here means unlimited.
const (
	// DefaultDrainGrace bounds one child's wait after SIGTERM when the
	// caller passes a non-positive grace.
	DefaultDrainGrace = 3 * time.Second
	// DefaultDrainCap bounds drained children when the caller passes a
	// non-positive cap.
	DefaultDrainCap = process.MaxProcs
)

// DrainConfig tunes a drain. The zero value selects the defaults.
type DrainConfig struct {
	// Grace bounds one child's wait after SIGTERM; <=0 selects
	// DefaultDrainGrace.
	Grace time.Duration
	// MaxChildren caps drained children; <=0 selects DefaultDrainCap.
	// Children past the cap are left untouched and the report refuses
	// with ReasonDrainCap.
	MaxChildren int
}

// DrainOutcome is one child's separate drain facts. Signaled and Reaped
// are independent: a child may already have exited (reaped without a
// signal) or may need escalation (signaled, then reaped by abort).
type DrainOutcome struct {
	PID      int
	Signaled bool
	Reaped   bool
	Reason   string // ReasonDrained | ReasonDrainEscalated | ReasonDrainFailed
}

// DrainReport is the terminal fact of one drain. Complete is true only
// when every owned child was reaped and none remain.
type DrainReport struct {
	Outcomes []DrainOutcome
	Complete bool
	Reason   string // "" when Complete; otherwise a Reason* constant
}

// Drain stops admission, then reaps every live owned child within bounds:
// SIGTERM, a bounded grace wait, then abort (SIGKILL and reap). Only
// children of o are ever signaled — process.Owner verifies the
// spawn-start token first, so a reused PID can never redirect a signal —
// and foreign PIDs are unreachable here by construction. Children are
// drained in PID order. The gate stops cooperating admitters only; a
// racing spawner leaves Live non-zero and the report refuses instead of
// claiming a complete drain. Drain returns an error only for misuse; an
// incomplete drain is a Report with Complete false and a named reason,
// never a silent partial success.
func Drain(g *Gate, o *process.Owner, cfg DrainConfig) (DrainReport, error) {
	if g == nil || o == nil {
		return DrainReport{}, fmt.Errorf("%w: drain needs a gate and an owner", ErrInvalid)
	}
	g.Stop()
	grace := cfg.Grace
	if grace <= 0 {
		grace = DefaultDrainGrace
	}
	max := cfg.MaxChildren
	if max <= 0 {
		max = DefaultDrainCap
	}
	edges := o.Tree()
	sort.Slice(edges, func(i, j int) bool { return edges[i].Child.PID < edges[j].Child.PID })
	rep := DrainReport{}
	if len(edges) > max {
		rep.Reason = ReasonDrainCap
		edges = edges[:max]
	}
	for _, e := range edges {
		out := DrainOutcome{PID: e.Child.PID}
		if serr := o.Signal(e.Child, syscall.SIGTERM); serr == nil {
			out.Signaled = true
		}
		if _, werr := o.Wait(e.Child, grace); werr == nil {
			out.Reaped = true
			out.Reason = ReasonDrained
			// A waited child is still retained until abort; forget
			// it so the owner holds no strays.
			if aerr := o.Abort(e.Child); aerr != nil {
				out.Reason = ReasonDrainFailed
			}
			rep.Outcomes = append(rep.Outcomes, out)
			continue
		}
		// Still running past grace: escalate to abort.
		if aerr := o.Abort(e.Child); aerr == nil {
			out.Reaped = true
			out.Reason = ReasonDrainEscalated
		} else {
			out.Reason = ReasonDrainFailed
		}
		rep.Outcomes = append(rep.Outcomes, out)
	}
	rep.Complete = rep.Reason == "" && o.Live() == 0 && allReaped(rep.Outcomes)
	if !rep.Complete && rep.Reason == "" {
		rep.Reason = ReasonDrainFailed
	}
	return rep, nil
}

func allReaped(outcomes []DrainOutcome) bool {
	for _, out := range outcomes {
		if !out.Reaped || out.Reason == ReasonDrainFailed {
			return false
		}
	}
	return true
}
