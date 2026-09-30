package workspace

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"time"

	journal "github.com/veighnsche/can-lang/tools/native-test-owner/journal"
)

// GrantLease registers a read/retention grant over the workspace. The grant
// is journaled before it takes effect. A lease owned by anyone other than
// the workspace owner is foreign; foreign leases are recorded (so recovery
// can report them) but never grant disposal authority.
func (o *Owner) GrantLease(opID string, h Handle, id string, pid int, startToken string, expiresWallMs int64) (*Lease, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	s, err := o.lookupLocked(h, false)
	if err != nil {
		return nil, err
	}
	if id == "" || len(id) > journal.MaxLeaseIDLen {
		return nil, fmt.Errorf("%w: malformed lease ID", ErrInvalid)
	}
	if pid <= 0 || startToken == "" || len(startToken) > journal.MaxStartTokenLen {
		return nil, fmt.Errorf("%w: lease needs positive PID and spawn-start token", ErrInvalid)
	}
	args := struct {
		Space   string `json:"space"`
		Lease   string `json:"lease"`
		PID     int    `json:"pid"`
		Token   string `json:"token"`
		Expires int64  `json:"expires"`
	}{Space: h.SpaceID, Lease: id, PID: pid, Token: startToken, Expires: expiresWallMs}
	joined, err := o.journalIntent(opID, args, journal.PathIdentity{Path: s.root}, id, expiresWallMs)
	if err != nil {
		return nil, err
	}
	if joined {
		v, rerr := o.rejoined(opID)
		if v == nil {
			return nil, rerr
		}
		out := *(v.(*Lease))
		return &out, rerr
	}
	if prev, dup := s.leases[id]; dup && !prev.Released {
		return nil, o.reject(opID, fmt.Errorf("%w: lease %q already held", ErrCollision, id), "duplicate lease")
	}
	l := &Lease{ID: id, PID: pid, StartToken: startToken, ExpiresWallMs: expiresWallMs}
	s.leases[id] = l
	o.closeOp(opID)
	o.remember(opID, l, nil)
	out := *l
	return &out, nil
}

// ReleaseLease releases a lease previously granted to the calling owner.
// Only the lease holder (matching PID and spawn-start token) or the
// workspace owner releasing its own lease can release; anyone else is
// rejected without touching the lease.
func (o *Owner) ReleaseLease(opID string, h Handle, id string, pid int, startToken string) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	s, err := o.lookupLocked(h, false)
	if err != nil {
		return err
	}
	args := struct {
		Space string `json:"space"`
		Lease string `json:"lease"`
		PID   int    `json:"pid"`
		Token string `json:"token"`
	}{Space: h.SpaceID, Lease: id, PID: pid, Token: startToken}
	joined, jerr := o.journalIntent(opID, args, journal.PathIdentity{Path: s.root}, id, 0)
	if jerr != nil {
		return jerr
	}
	if joined {
		_, rerr := o.rejoined(opID)
		return rerr
	}
	l, ok := s.leases[id]
	if !ok || l.Released {
		return o.reject(opID, fmt.Errorf("%w: lease %q", ErrNotFound, id), "unknown lease")
	}
	if pid != l.PID || startToken != l.StartToken {
		// The workspace owner may release its own leases even when
		// calling under its current identity; anything else is foreign.
		if !(pid == o.pid && startToken == o.token && l.PID == s.ownerPID && l.StartToken == s.ownerToken) {
			return o.reject(opID, fmt.Errorf("%w: lease %q held by another owner", ErrOwner, id), "wrong lease holder")
		}
	}
	l.Released = true
	o.closeOp(opID)
	o.remember(opID, true, nil)
	return nil
}

// Leases lists the current leases over a workspace.
func (o *Owner) Leases(h Handle) ([]Lease, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	s, err := o.lookupLocked(h, false)
	if err != nil {
		return nil, err
	}
	var out []Lease
	for _, id := range sortedKeys(s.leases) {
		out = append(out, *s.leases[id])
	}
	return out, nil
}

// DisposeReceipt is the terminal fact of one disposal. A complete receipt
// lists everything released; an incomplete receipt lists remaining paths and
// errors, and the workspace stays owned and charged.
type DisposeReceipt struct {
	Released  []string `json:"released,omitempty"`
	Remaining []string `json:"remaining,omitempty"`
	Errors    []string `json:"errors,omitempty"`
	Complete  bool     `json:"complete"`
	Joined    bool     `json:"joined,omitempty"`
}

// Dispose stops admission, retires expired or released dependent leases,
// removes identity-verified owned entries and reports remaining paths and
// errors. Repeats join a terminal receipt. Active or foreign leases, a
// replaced root, and failed deletions keep the workspace charged and
// unresolved; they are never silently cleaned.
func (o *Owner) Dispose(opID string, h Handle) (*DisposeReceipt, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	s, ok := o.spaces[h.SpaceID]
	if !ok {
		return nil, fmt.Errorf("%w: unknown workspace", ErrNotFound)
	}
	if h.Generation != s.gen {
		return nil, fmt.Errorf("%w: want %d", ErrStaleHandle, s.gen)
	}
	if o.pid != s.ownerPID || o.token != s.ownerToken {
		return nil, fmt.Errorf("%w: workspace owned by another owner", ErrOwner)
	}
	args := struct {
		Space string `json:"space"`
		Op    string `json:"op"`
	}{Space: h.SpaceID, Op: "dispose"}
	joined, err := o.journalIntent(opID, args, journal.PathIdentity{Path: s.root}, "", 0)
	if err != nil {
		return nil, err
	}
	if joined {
		v, rerr := o.rejoined(opID)
		if v == nil {
			return nil, rerr
		}
		res := *(v.(*DisposeReceipt))
		res.Joined = true
		return &res, rerr
	}
	// A terminal receipt joins across operation IDs: repeats never
	// re-delete, and a later safe cleanup preserves the earlier outcome
	// shape. An incomplete receipt retries instead of joining.
	if s.dispose != nil && s.dispose.Complete {
		res := *s.dispose
		res.Joined = true
		o.remember(opID, &res, nil)
		o.closeOp(opID)
		out := res
		return &out, nil
	}
	receipt := &DisposeReceipt{}
	nowMs := o.clock()

	// Stop admission for the disposal interval.
	prevState := s.state
	s.state = spaceClosing

	// Retire only dependent leases that are already expired or released.
	// Active or foreign leases block disposal and stay reported.
	for _, id := range sortedKeys(s.leases) {
		l := s.leases[id]
		if l.Released || !l.active(nowMs) {
			delete(s.leases, id)
			continue
		}
		if l.PID != s.ownerPID || l.StartToken != s.ownerToken {
			receipt.Remaining = append(receipt.Remaining, s.root)
			receipt.Errors = append(receipt.Errors, "foreign lease "+id)
			s.state = prevState
			o.failOp(opID, "foreign lease "+id)
			ferr := fmt.Errorf("%w: %q", ErrForeignLease, id)
			o.remember(opID, receipt, ferr)
			out := *receipt
			return &out, ferr
		}
		receipt.Remaining = append(receipt.Remaining, s.root)
		receipt.Errors = append(receipt.Errors, "active lease "+id)
		s.state = prevState
		o.failOp(opID, "active lease "+id)
		aerr := fmt.Errorf("%w: %q", ErrActiveLease, id)
		o.remember(opID, receipt, aerr)
		out := *receipt
		return &out, aerr
	}

	// Verify the root is still the exact reserved object before touching
	// anything. A replaced root is never deletion authority.
	if err := o.checkRootLocked(s); err != nil {
		receipt.Remaining = append(receipt.Remaining, s.root)
		receipt.Errors = append(receipt.Errors, err.Error())
		s.state = prevState
		o.failOp(opID, "replaced root")
		o.remember(opID, receipt, err)
		out := *receipt
		return &out, err
	}

	released, remaining, errs := o.removeTreeLocked(s)
	receipt.Released = released
	receipt.Remaining = remaining
	receipt.Errors = errs
	if len(remaining) > 0 || len(errs) > 0 {
		receipt.Complete = false
		s.state = prevState // still owned and charged
		partials := append([]string{}, remaining...)
		partials = append(partials, errs...)
		o.failOp(opID, partials...)
		uerr := fmt.Errorf("%w: %d paths remain", ErrUnresolved, len(remaining))
		o.remember(opID, receipt, uerr)
		out := *receipt
		return &out, uerr
	}
	// All entries are gone; remove the root itself after one final identity
	// check, then confirm it is really gone.
	if err := o.checkRootLocked(s); err != nil {
		receipt.Remaining = append(receipt.Remaining, s.root)
		receipt.Errors = append(receipt.Errors, err.Error())
		s.state = prevState
		o.failOp(opID, "replaced root")
		o.remember(opID, receipt, err)
		out := *receipt
		return &out, err
	}
	if rerr := os.Remove(s.root); rerr != nil {
		receipt.Remaining = append(receipt.Remaining, s.root)
		receipt.Errors = append(receipt.Errors, rerr.Error())
		s.state = prevState
		o.failOp(opID, "remove root")
		derr := fmt.Errorf("%w: remove root: %v", ErrUnresolved, rerr)
		o.remember(opID, receipt, derr)
		out := *receipt
		return &out, derr
	}
	if _, serr := os.Lstat(s.root); serr == nil {
		receipt.Remaining = append(receipt.Remaining, s.root)
		receipt.Errors = append(receipt.Errors, "root still present after removal")
		s.state = prevState
		o.failOp(opID, "root persists")
		perr := fmt.Errorf("%w: root persists", ErrUnresolved)
		o.remember(opID, receipt, perr)
		out := *receipt
		return &out, perr
	} else if !os.IsNotExist(serr) {
		receipt.Remaining = append(receipt.Remaining, s.root)
		receipt.Errors = append(receipt.Errors, serr.Error())
		s.state = prevState
		o.failOp(opID, "stat root")
		serr2 := fmt.Errorf("%w: confirm root removal: %v", ErrUnresolved, serr)
		o.remember(opID, receipt, serr2)
		out := *receipt
		return &out, serr2
	}
	receipt.Complete = true
	s.state = spaceClosed
	s.chargedBytes = 0
	s.chargedEntries = 0
	s.r.Close()
	s.dispose = receipt
	o.closeOp(opID)
	o.remember(opID, receipt, nil)
	for _, st := range []journal.State{journal.StateClosing, journal.StateClosed} {
		_, _ = o.j.Advance(s.opID, st)
	}
	out := *receipt
	return &out, nil
}

// removeTreeLocked removes every owned entry under the root without
// following symlinks: files and links first, directories deepest-first.
// It returns sorted released and remaining relative paths plus errors.
func (o *Owner) removeTreeLocked(s *space) (released, remaining, errs []string) {
	var files, dirs []string
	type frame struct {
		rel string
	}
	stack := []frame{{rel: "."}}
	for len(stack) > 0 {
		fr := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		f, oerr := s.r.Open(fr.rel)
		if oerr != nil {
			remaining = append(remaining, displayRel(fr.rel))
			errs = append(errs, fr.rel+": "+oerr.Error())
			continue
		}
		dirents, rerr := f.ReadDir(-1)
		cerr := f.Close()
		if rerr != nil {
			remaining = append(remaining, displayRel(fr.rel))
			errs = append(errs, fr.rel+": "+rerr.Error())
			continue
		}
		if cerr != nil {
			remaining = append(remaining, displayRel(fr.rel))
			errs = append(errs, fr.rel+": "+cerr.Error())
			continue
		}
		for _, de := range dirents {
			child := de.Name()
			if fr.rel != "." {
				child = fr.rel + "/" + child
			}
			typ := de.Type()
			switch {
			case typ&os.ModeSymlink != 0:
				files = append(files, child)
			case de.IsDir():
				dirs = append(dirs, child)
				stack = append(stack, frame{rel: child})
			default:
				files = append(files, child)
			}
		}
	}
	for _, p := range files {
		if rerr := s.r.Remove(p); rerr != nil {
			remaining = append(remaining, p)
			errs = append(errs, p+": "+rerr.Error())
			continue
		}
		released = append(released, p)
		delete(s.tracked, p)
	}
	sort.Slice(dirs, func(i, j int) bool { return depthOf(dirs[i]) > depthOf(dirs[j]) })
	for _, p := range dirs {
		if rerr := s.r.Remove(p); rerr != nil {
			remaining = append(remaining, p)
			errs = append(errs, p+": "+rerr.Error())
			continue
		}
		released = append(released, p)
		delete(s.tracked, p)
	}
	sort.Strings(released)
	sort.Strings(remaining)
	sort.Strings(errs)
	return released, remaining, errs
}

func displayRel(rel string) string {
	if rel == "." {
		return "(root)"
	}
	return rel
}

// RecoverOutcome is one reconciled registered allocation.
type RecoverOutcome struct {
	SpaceID   string   `json:"spaceId"`
	Action    string   `json:"action"` // disposed | active | foreign | unresolved
	Detail    string   `json:"detail"`
	Remaining []string `json:"remaining,omitempty"`
}

// RecoverReport reconciles every registered allocation.
type RecoverReport struct {
	Outcomes []RecoverOutcome `json:"outcomes"`
}

// Recover reconciles registered allocations against live identity and
// leases. It only ever touches registered roots: active work is left
// untouched, foreign or mismatched roots are reported, and abandoned owned
// entries are disposed or reported unresolved. Recovery never invents
// deletion authority over arbitrary caller paths.
func (o *Owner) Recover() (*RecoverReport, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	nowMs := o.clock()
	report := &RecoverReport{}
	for _, id := range sortedKeys(o.spaces) {
		s := o.spaces[id]
		if s.state == spaceClosed {
			continue
		}
		oc := RecoverOutcome{SpaceID: id}
		// Active leases (by wall clock) mean live work: untouched.
		active := false
		for _, lid := range sortedKeys(s.leases) {
			l := s.leases[lid]
			if l.active(nowMs) {
				if l.PID != s.ownerPID || l.StartToken != s.ownerToken {
					oc.Action = "foreign"
					oc.Detail = "foreign lease " + lid
				} else {
					oc.Action = "active"
					oc.Detail = "active lease " + lid
				}
				active = true
				break
			}
		}
		if active {
			report.Outcomes = append(report.Outcomes, oc)
			continue
		}
		// Identity check before any deletion authority.
		if err := o.checkRootLocked(s); err != nil {
			oc.Action = "foreign"
			oc.Detail = err.Error()
			oc.Remaining = []string{s.root}
			report.Outcomes = append(report.Outcomes, oc)
			continue
		}
		// Retire expired/released leases, then dispose the abandoned
		// allocation under a recovery-bound operation ID.
		for _, lid := range sortedKeys(s.leases) {
			l := s.leases[lid]
			if l.Released || !l.active(nowMs) {
				delete(s.leases, lid)
			}
		}
		recID := "recover-" + id + "-" + time.UnixMilli(nowMs).UTC().Format("20060102150405")
		// The opID must be journal-valid; space IDs already are, and the
		// suffix adds only digits. Truncate defensively.
		if len(recID) > 128 {
			recID = recID[:128]
		}
		if _, err := o.j.Reserve(recID, mustDigest(recoverArgs{Space: id, Op: "recover"}), o.owner(), journal.PathIdentity{Path: s.root}, "", 0); err != nil {
			// A repeated recovery in the same millisecond joins the
			// earlier attempt instead of dispatching twice.
			if errors.Is(err, journal.ErrConflict) {
				oc.Action = "unresolved"
				oc.Detail = "recovery already recorded for " + id
				oc.Remaining = []string{s.root}
				report.Outcomes = append(report.Outcomes, oc)
				continue
			}
			oc.Action = "unresolved"
			oc.Detail = err.Error()
			oc.Remaining = []string{s.root}
			report.Outcomes = append(report.Outcomes, oc)
			continue
		}
		prevState := s.state
		s.state = spaceClosing
		released, remaining, errs := o.removeTreeLocked(s)
		_ = released
		if len(remaining) > 0 || len(errs) > 0 {
			s.state = prevState
			oc.Action = "unresolved"
			oc.Detail = "failed deletion keeps workspace charged"
			oc.Remaining = append([]string{}, remaining...)
			_, _ = o.j.Advance(recID, journal.StateFailed, remaining...)
			report.Outcomes = append(report.Outcomes, oc)
			continue
		}
		if err := o.checkRootLocked(s); err != nil {
			s.state = prevState
			oc.Action = "foreign"
			oc.Detail = err.Error()
			oc.Remaining = []string{s.root}
			_, _ = o.j.Advance(recID, journal.StateFailed, "replaced root")
			report.Outcomes = append(report.Outcomes, oc)
			continue
		}
		if rerr := os.Remove(s.root); rerr != nil {
			s.state = prevState
			oc.Action = "unresolved"
			oc.Detail = rerr.Error()
			oc.Remaining = []string{s.root}
			_, _ = o.j.Advance(recID, journal.StateFailed, "remove root")
			report.Outcomes = append(report.Outcomes, oc)
			continue
		}
		s.state = spaceClosed
		s.chargedBytes = 0
		s.chargedEntries = 0
		s.r.Close()
		for _, st := range []journal.State{journal.StateOpening, journal.StateLive, journal.StateClosing, journal.StateClosed} {
			_, _ = o.j.Advance(recID, st)
		}
		for _, st := range []journal.State{journal.StateClosing, journal.StateClosed} {
			_, _ = o.j.Advance(s.opID, st)
		}
		oc.Action = "disposed"
		oc.Detail = "abandoned owned entries removed"
		report.Outcomes = append(report.Outcomes, oc)
	}
	return report, nil
}

type recoverArgs struct {
	Space string `json:"space"`
	Op    string `json:"op"`
}

func mustDigest(v any) string {
	d, err := digestOf(v)
	if err != nil {
		// digestOf only fails on unmarshalable values; the caller passes
		// plain structs. Failing closed here would invent an opID
		// collision, so panic on this programming error instead.
		panic(err)
	}
	return d
}
