package external

import (
	"fmt"
	"sort"

	"github.com/veighnsche/can-lang/tools/native-test-owner/journal"
)

// FenceAck is the retained fence acknowledgment: the owner, the fencing
// instant and the owned live set fenced. It proves recovery surveyed the
// owned resources; it settles no charge and resolves no release.
type FenceAck struct {
	Owner  journal.Owner
	TimeMs int64
	Live   []string
}

func (a FenceAck) copy() FenceAck {
	out := a
	out.Live = append([]string(nil), a.Live...)
	return out
}

// Quiesce stops admission and reports the owned live set. It is
// idempotent. Dispatch and every admission fail with ErrQuiesced once
// quiesced; releases still proceed so dependents can drain.
func (r *Registry) Quiesce() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stopped = true
	return r.liveLocked()
}

// Fence records the fence acknowledgment over the owned live set. Fencing
// requires quiescence first; it is single-shot and returns the retained
// acknowledgment on repeats.
func (r *Registry) Fence() (FenceAck, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.stopped {
		return FenceAck{}, fmt.Errorf("%w: fence needs quiescence first", ErrInvalid)
	}
	if r.fenced {
		return r.fenceAck.copy(), nil
	}
	r.fenced = true
	r.fenceAck = &FenceAck{Owner: r.owner, TimeMs: r.clock(), Live: r.liveLocked()}
	return r.fenceAck.copy(), nil
}

// FenceAck reports the retained fence acknowledgment, if any.
func (r *Registry) FenceAck() (FenceAck, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.fenced {
		return FenceAck{}, false
	}
	return r.fenceAck.copy(), true
}

// RecoverRequest carries the caller's spawn-start identity and the
// recovery scan decisions per operation ID. Allow marks an operation the
// scan evidenced as recoverable here; Reason names the refusal otherwise.
type RecoverRequest struct {
	Self   journal.Owner
	Allow  map[string]bool
	Reason map[string]string
}

// RecoverOutcome is one resource's separate recovery facts. Touched is
// true only for evidenced owned resources the scan allowed; every other
// resource is untouched with a named reason. ReleaseMissing reports that
// no Release was recorded; recovery never auto-resolves it.
type RecoverOutcome struct {
	OperationID    string
	Touched        bool
	Reason         string
	ReleaseMissing bool
}

// RecoverReport is the terminal fact of one recovery pass.
type RecoverReport struct {
	Outcomes []RecoverOutcome
	Touched  int
}

// Recover touches only evidenced owned resources the scan allowed: the
// resource must be recorded here, its owner must match Self, its journal
// intent must still be present, and the scan must allow it. Journal-only
// intents (crash between reserve and effect), foreign resources, and
// refused operations are never touched. Recovery settles no charge and
// releases nothing: every outcome keeps ReleaseMissing true, so a later
// explicit Release can still proceed. Touch may be nil for a
// classification-only pass; a non-nil touch runs under the registry lock
// and must not call back into the Registry.
func (r *Registry) Recover(req RecoverRequest, touch func(opID string)) RecoverReport {
	r.mu.Lock()
	defer r.mu.Unlock()
	ids := make([]string, 0, len(r.resources))
	for id, res := range r.resources {
		if !res.released {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	rep := RecoverReport{}
	for _, id := range ids {
		out := RecoverOutcome{OperationID: id, ReleaseMissing: true}
		res := r.resources[id]
		switch {
		case res.owner.PID != req.Self.PID || res.owner.StartToken != req.Self.StartToken:
			out.Reason = ReasonForeign
		default:
			if _, ok := r.j.Lookup(id); !ok {
				out.Reason = ReasonNoEvidence
				break
			}
			if !req.Allow[id] {
				out.Reason = req.Reason[id]
				if out.Reason == "" {
					out.Reason = ReasonDenied
				}
				break
			}
			out.Touched = true
			out.Reason = ReasonTouched
			rep.Touched++
			if touch != nil {
				touch(id)
			}
		}
		// A touched resource still lacks its release: recovery never
		// resolves that on its own.
		rep.Outcomes = append(rep.Outcomes, out)
	}
	return rep
}

// CleanupOrder returns every recorded operation in cleanup dependency
// order — dependents before the resources they were admitted under — so
// releases can drain without tripping ErrDependency. Order among
// unrelated operations is sorted for determinism.
func (r *Registry) CleanupOrder() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	ids := make([]string, 0, len(r.resources))
	for id := range r.resources {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	const (
		white = 0 // unvisited
		gray  = 1 // on the stack
		black = 2 // emitted
	)
	color := make(map[string]int, len(ids))
	var order []string
	var visit func(id string)
	visit = func(id string) {
		if color[id] != white {
			return
		}
		color[id] = gray
		kids := append([]string(nil), r.children[id]...)
		sort.Strings(kids)
		for _, k := range kids {
			if _, known := r.resources[k]; known {
				visit(k)
			}
		}
		color[id] = black
		order = append(order, id)
	}
	for _, id := range ids {
		visit(id)
	}
	return order
}
