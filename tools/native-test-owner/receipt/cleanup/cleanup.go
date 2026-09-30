package cleanup

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/veighnsche/can-lang/tools/native-test-owner/journal"
)

// Request asks for one owner-only cleanup. Self is the caller's
// spawn-start identity; Allow with DenyReason carries the recovery scan
// decision for OperationID; Prior is the old recorded outcome, preserved
// verbatim in the receipt.
type Request struct {
	OperationID string
	Path        string // target to delete; "" deletes nothing
	Self        journal.Owner
	Allow       bool    // recovery scan decision for OperationID
	DenyReason  string  // named reason when !Allow
	Prior       Outcome // old recorded outcome; "" selects incomplete (fail closed)
}

// Perform verifies owner-only authority and the scan decision, then
// deletes at most the reserved target. Refusals — wrong identity,
// replaced path, unknown operation, a negative scan decision, or a failed
// removal — delete nothing and preserve the prior outcome; they never
// report a pass, and (except for a failed removal, which is terminal)
// leave the journal untouched so a later safe cleanup can still proceed.
// Success walks the journal toward its terminal state and reports
// Complete with the prior outcome preserved: a later safe cleanup of an
// incomplete run stays incomplete. remove deletes the reserved target; it
// must be non-nil when Path is non-empty, and is never called on a
// refusal. Perform always returns a receipt, even on refusal.
func Perform(j *journal.Journal, req Request, remove func(path string) error) (*Receipt, error) {
	prior := req.Prior
	if prior == "" {
		prior = OutcomeIncomplete
	}
	rec := func(reason string) *Receipt {
		return &Receipt{SchemaVersion: SchemaVersion, OperationID: req.OperationID, Outcome: prior, Reason: reason}
	}
	if j == nil {
		return rec(ReasonDenied), fmt.Errorf("%w: nil journal", ErrInvalid)
	}
	if !validID(req.OperationID) {
		return rec(ReasonDenied), fmt.Errorf("%w: malformed operation ID", ErrInvalid)
	}
	if !prior.valid() {
		return rec(ReasonDenied), fmt.Errorf("%w: unknown prior outcome %q", ErrInvalid, req.Prior)
	}
	if req.Self.PID <= 0 || req.Self.StartToken == "" || len(req.Self.StartToken) > journal.MaxStartTokenLen {
		return rec(ReasonDenied), fmt.Errorf("%w: caller needs positive PID and spawn-start token", ErrInvalid)
	}
	if req.Path != "" && !filepath.IsAbs(req.Path) {
		return rec(ReasonDenied), fmt.Errorf("%w: target must be absolute", ErrInvalid)
	}
	// Owner-only rule: the spawn-start identity must match the
	// reservation, and the path must still be the reserved object.
	if err := j.VerifyAuthority(req.OperationID, req.Self, req.Path); err != nil {
		switch {
		case errors.Is(err, journal.ErrWrongOwner):
			return rec(ReasonOwnerMismatch), fmt.Errorf("%w: %w", ErrWrongOwner, err)
		case errors.Is(err, journal.ErrWrongPath):
			return rec(ReasonReplacedPath), fmt.Errorf("%w: %w", ErrWrongPath, err)
		case errors.Is(err, journal.ErrNotFound):
			return rec(ReasonUnknownOp), fmt.Errorf("%w: operation %q", ErrNotFound, req.OperationID)
		default:
			return rec(ReasonDenied), fmt.Errorf("%w: %w", ErrDenied, err)
		}
	}
	if !req.Allow {
		reason := req.DenyReason
		if reason == "" {
			reason = ReasonDenied
		}
		if len(reason) > MaxReasonLen {
			reason = reason[:MaxReasonLen]
		}
		return rec(reason), fmt.Errorf("%w: operation %q: %s", ErrDenied, req.OperationID, reason)
	}
	if req.Path != "" {
		if remove == nil {
			return rec(ReasonDenied), fmt.Errorf("%w: removal needed but no remover given", ErrInvalid)
		}
		if err := remove(req.Path); err != nil {
			_, _ = j.Advance(req.OperationID, journal.StateFailed, truncatePartial(ReasonRemoveFailed))
			return rec(ReasonRemoveFailed), fmt.Errorf("%w: %w", ErrRemoveFailed, err)
		}
	}
	closeJournal(j, req.OperationID)
	r := rec(ReasonCleanupComplete)
	r.Removed = req.Path != ""
	r.Complete = true
	return r, nil
}

// closeJournal walks one operation from its current state toward closed.
// Already-terminal operations keep their terminal state; errors stop the
// walk without hiding the cleanup outcome.
func closeJournal(j *journal.Journal, opID string) {
	r, ok := j.Lookup(opID)
	if !ok {
		return
	}
	chain := []journal.State{journal.StateOpening, journal.StateLive, journal.StateClosing, journal.StateClosed}
	start := 0
	switch r.State {
	case journal.StateOpening:
		start = 1
	case journal.StateLive:
		start = 2
	case journal.StateClosing:
		start = 3
	case journal.StateClosed, journal.StateFailed:
		return
	}
	for _, st := range chain[start:] {
		if _, err := j.Advance(opID, st); err != nil {
			return
		}
	}
}

func truncatePartial(s string) string {
	if len(s) > journal.MaxPartialLen {
		return s[:journal.MaxPartialLen]
	}
	if s == "" {
		return "cleanup refused"
	}
	return s
}

// validID matches the shared wire ID shape: ^[A-Za-z0-9][A-Za-z0-9_.:-]{0,127}$.
func validID(s string) bool {
	if len(s) == 0 || len(s) > 128 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		ok := c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' ||
			c == '_' || c == '.' || c == ':' || c == '-'
		if !ok {
			return false
		}
		if i == 0 && !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}
