package external

import (
	"fmt"

	"github.com/veighnsche/can-lang/tools/native-test-owner/journal"
)

// dispatchRec is one at-most-once dispatch journal entry: the effect runs
// at most once per operation ID, and an uncertain entry is never
// redispatched.
type dispatchRec struct {
	opID      string
	digest    string
	runs      int
	uncertain bool
}

// dispatchKinds are the legs dispatchable without a parent grant.
// Contexts and credentials need OpenContext/IssueCredential instead.
func dispatchKinds(kind Kind) bool {
	switch kind {
	case KindListener, KindConnection, KindNamespace, KindPrefix:
		return true
	}
	return false
}

// Dispatch journals intent and runs one service effect at most once.
// Repeating the same operation ID with the same leg joins the recorded
// fact without running the effect again; the same ID with a different leg
// is rejected with ErrChangedInput, and any repeat of an uncertain
// dispatch is rejected with ErrIndeterminate without redispatching. A
// failed effect is recorded uncertain — it may have partially run, so
// repeats report indeterminate rather than joining a failure as success,
// while the first caller keeps the real error. A durable intent without
// a local terminal fact likewise reports indeterminate. Foreign targets
// are rejected before anything is journaled. It reports joined=true when
// no new effect ran. The effect runs under the registry lock and must
// not call back into the Registry.
func (r *Registry) Dispatch(opID string, kind Kind, target string, effect func() error) (bool, error) {
	if !validID(opID) {
		return false, fmt.Errorf("%w: malformed operation ID", ErrInvalid)
	}
	if target == "" || len(target) > MaxTargetLen {
		return false, fmt.Errorf("%w: malformed target", ErrInvalid)
	}
	if !dispatchKinds(kind) {
		return false, fmt.Errorf("%w: kind %q needs its parent-grant admission", ErrInvalid, kind)
	}
	if effect == nil {
		return false, fmt.Errorf("%w: nil effect", ErrInvalid)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.stopped {
		return false, fmt.Errorf("%w: registry quiesced", ErrQuiesced)
	}
	if !r.memberLocked(kind, target) {
		return false, fmt.Errorf("%w: %s %q is not a declared service target", ErrForeign, kind, target)
	}
	digest := admitDigest(kind, target, "")
	if prev, known := r.dispatches[opID]; known {
		if prev.digest != digest {
			return false, fmt.Errorf("%w: operation %q", ErrChangedInput, opID)
		}
		if prev.uncertain {
			return false, fmt.Errorf("%w: operation %q", ErrIndeterminate, opID)
		}
		return true, nil
	}
	// Intent before effect: consult the durable journal before running
	// anything. A reservation this registry cannot join is never
	// re-executed.
	if prev, ok := r.j.Lookup(opID); ok {
		if prev.ArgumentsDigest != digest {
			return false, fmt.Errorf("%w: operation %q", ErrChangedInput, opID)
		}
		r.dispatches[opID] = &dispatchRec{opID: opID, digest: digest, uncertain: true}
		return false, fmt.Errorf("%w: operation %q", ErrIndeterminate, opID)
	}
	if _, err := r.j.Reserve(opID, digest, r.owner, journal.PathIdentity{}, "", 0); err != nil {
		return false, fmt.Errorf("external: journal reserve: %w", err)
	}
	if err := effect(); err != nil {
		_, _ = r.j.Advance(opID, journal.StateFailed, "dispatch effect failed")
		r.dispatches[opID] = &dispatchRec{opID: opID, digest: digest, runs: 1, uncertain: true}
		return false, err
	}
	for _, st := range []journal.State{journal.StateOpening, journal.StateLive} {
		if _, err := r.j.Advance(opID, st); err != nil {
			return false, fmt.Errorf("external: journal advance: %w", err)
		}
	}
	r.dispatches[opID] = &dispatchRec{opID: opID, digest: digest, runs: 1}
	return false, nil
}

// ReportUncertain reports a lost acknowledgment: later joins of the
// operation report indeterminate without redispatching.
func (r *Registry) ReportUncertain(opID string) error {
	if !validID(opID) {
		return fmt.Errorf("%w: malformed operation ID", ErrInvalid)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.dispatches[opID]
	if !ok {
		return fmt.Errorf("%w: operation %q", ErrNotFound, opID)
	}
	rec.uncertain = true
	return nil
}

// EffectRuns reports how many times a dispatch effect ran here.
func (r *Registry) EffectRuns(opID string) (int, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.dispatches[opID]
	if !ok {
		return 0, false
	}
	return rec.runs, true
}
