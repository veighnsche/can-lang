package cleanup

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/veighnsche/can-lang/tools/native-test-owner/journal"
	"github.com/veighnsche/can-lang/tools/native-test-owner/recovery"
)

const testDigest = "sha256:" + "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func testSelf() journal.Owner {
	return journal.Owner{PID: os.Getpid(), StartToken: "p11-test-owner"}
}

func testStranger() journal.Owner {
	return journal.Owner{PID: os.Getpid(), StartToken: "p11-stranger"}
}

func reserve(t *testing.T, j *journal.Journal, opID string, owner journal.Owner, path journal.PathIdentity) {
	t.Helper()
	if _, err := j.Reserve(opID, testDigest, owner, path, "", 0); err != nil {
		t.Fatalf("Reserve %q: %v", opID, err)
	}
}

func writeTempFile(t *testing.T, name, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return p
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile %q: %v", path, err)
	}
	return string(raw)
}

// spyRemover records deletion attempts. Refusals must never call it.
type spyRemover struct {
	mu    sync.Mutex
	calls []string
	err   error
}

func (s *spyRemover) remove(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, path)
	return s.err
}

func (s *spyRemover) called() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.calls) > 0
}

func journalState(t *testing.T, j *journal.Journal, opID string) journal.State {
	t.Helper()
	r, ok := j.Lookup(opID)
	if !ok {
		t.Fatalf("Lookup %q: unknown operation", opID)
	}
	return r.State
}

func TestPerformRefusalsTable(t *testing.T) {
	self := testSelf()
	stranger := testStranger()

	t.Run("wrong-owner", func(t *testing.T) {
		target := writeTempFile(t, "target.txt", "owned-bytes")
		id, err := journal.StatPath(target)
		if err != nil {
			t.Fatalf("StatPath: %v", err)
		}
		j := journal.OpenMemory()
		defer j.Close()
		reserve(t, j, "refuse1", self, id)
		spy := &spyRemover{}
		rec, err := Perform(j, Request{OperationID: "refuse1", Path: target, Self: stranger, Allow: true, Prior: OutcomeIncomplete}, spy.remove)
		if !errors.Is(err, ErrWrongOwner) {
			t.Fatalf("Perform: got %v, want ErrWrongOwner", err)
		}
		if !errors.Is(err, journal.ErrWrongOwner) {
			t.Fatalf("Perform: journal cause lost: %v", err)
		}
		if rec == nil || rec.Outcome != OutcomeIncomplete || rec.Reason != ReasonOwnerMismatch || rec.Complete || rec.Removed {
			t.Fatalf("receipt = %+v, want preserved incomplete owner-mismatch refusal", rec)
		}
		if spy.called() {
			t.Fatalf("refusal called the remover")
		}
		if got := readFile(t, target); got != "owned-bytes" {
			t.Fatalf("refusal touched the target: %q", got)
		}
		if st := journalState(t, j, "refuse1"); st != journal.StateReserved {
			t.Fatalf("refusal mutated the journal to %s", st)
		}
	})

	t.Run("replaced-path", func(t *testing.T) {
		target := writeTempFile(t, "target.txt", "owned-bytes")
		id, err := journal.StatPath(target)
		if err != nil {
			t.Fatalf("StatPath: %v", err)
		}
		bogus := id
		bogus.Ino++
		j := journal.OpenMemory()
		defer j.Close()
		reserve(t, j, "refuse2", self, bogus)
		spy := &spyRemover{}
		rec, err := Perform(j, Request{OperationID: "refuse2", Path: target, Self: self, Allow: true, Prior: OutcomeIncomplete}, spy.remove)
		if !errors.Is(err, ErrWrongPath) {
			t.Fatalf("Perform: got %v, want ErrWrongPath", err)
		}
		if rec == nil || rec.Outcome != OutcomeIncomplete || rec.Reason != ReasonReplacedPath || rec.Complete {
			t.Fatalf("receipt = %+v, want preserved incomplete replaced-path refusal", rec)
		}
		if spy.called() {
			t.Fatalf("refusal called the remover")
		}
		if got := readFile(t, target); got != "owned-bytes" {
			t.Fatalf("refusal deleted the replacement: target gone or changed")
		}
		if st := journalState(t, j, "refuse2"); st != journal.StateReserved {
			t.Fatalf("refusal mutated the journal to %s", st)
		}
	})

	t.Run("unknown-operation", func(t *testing.T) {
		j := journal.OpenMemory()
		defer j.Close()
		spy := &spyRemover{}
		rec, err := Perform(j, Request{OperationID: "ghost1", Self: self, Allow: true, Prior: OutcomeIncomplete}, spy.remove)
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("Perform: got %v, want ErrNotFound", err)
		}
		if rec == nil || rec.Reason != ReasonUnknownOp || rec.Complete {
			t.Fatalf("receipt = %+v, want unknown-operation refusal", rec)
		}
		if spy.called() {
			t.Fatalf("refusal called the remover")
		}
	})

	t.Run("scan-denied", func(t *testing.T) {
		j := journal.OpenMemory()
		defer j.Close()
		reserve(t, j, "refuse4", self, journal.PathIdentity{})
		spy := &spyRemover{}
		rec, err := Perform(j, Request{OperationID: "refuse4", Self: self, Allow: false, DenyReason: recovery.ReasonForeignLive, Prior: OutcomeIncomplete}, spy.remove)
		if !errors.Is(err, ErrDenied) {
			t.Fatalf("Perform: got %v, want ErrDenied", err)
		}
		if rec == nil || rec.Outcome != OutcomeIncomplete || rec.Reason != recovery.ReasonForeignLive || rec.Complete {
			t.Fatalf("receipt = %+v, want preserved incomplete scan denial", rec)
		}
		if spy.called() {
			t.Fatalf("denial called the remover")
		}
		if st := journalState(t, j, "refuse4"); st != journal.StateReserved {
			t.Fatalf("denial mutated the journal to %s", st)
		}
	})

	t.Run("scan-denied-empty-reason", func(t *testing.T) {
		j := journal.OpenMemory()
		defer j.Close()
		reserve(t, j, "refuse5", self, journal.PathIdentity{})
		spy := &spyRemover{}
		rec, err := Perform(j, Request{OperationID: "refuse5", Self: self, Allow: false, Prior: OutcomeIncomplete}, spy.remove)
		if !errors.Is(err, ErrDenied) {
			t.Fatalf("Perform: got %v, want ErrDenied", err)
		}
		if rec == nil || rec.Reason != ReasonDenied || rec.Complete {
			t.Fatalf("receipt = %+v, want denied refusal", rec)
		}
	})

	t.Run("scan-denied-long-reason-truncated", func(t *testing.T) {
		j := journal.OpenMemory()
		defer j.Close()
		reserve(t, j, "refuse5b", self, journal.PathIdentity{})
		spy := &spyRemover{}
		long := strings.Repeat("r", MaxReasonLen+100)
		rec, err := Perform(j, Request{OperationID: "refuse5b", Self: self, Allow: false, DenyReason: long, Prior: OutcomeIncomplete}, spy.remove)
		if !errors.Is(err, ErrDenied) {
			t.Fatalf("Perform: got %v, want ErrDenied", err)
		}
		if rec == nil || rec.Reason != long[:MaxReasonLen] || rec.Complete {
			t.Fatalf("receipt reason len = %d, want %d", len(rec.Reason), MaxReasonLen)
		}
		if spy.called() {
			t.Fatalf("denial called the remover")
		}
	})

	t.Run("path-without-reservation", func(t *testing.T) {
		j := journal.OpenMemory()
		defer j.Close()
		reserve(t, j, "refuse5c", self, journal.PathIdentity{})
		spy := &spyRemover{}
		target := writeTempFile(t, "target.txt", "not-reserved")
		rec, err := Perform(j, Request{OperationID: "refuse5c", Path: target, Self: self, Allow: true, Prior: OutcomeIncomplete}, spy.remove)
		if !errors.Is(err, ErrWrongPath) {
			t.Fatalf("Perform: got %v, want ErrWrongPath", err)
		}
		if rec == nil || rec.Reason != ReasonReplacedPath || rec.Complete {
			t.Fatalf("receipt = %+v, want replaced-path refusal", rec)
		}
		if spy.called() {
			t.Fatalf("refusal called the remover")
		}
		if got := readFile(t, target); got != "not-reserved" {
			t.Fatalf("refusal touched the target: %q", got)
		}
	})

	t.Run("remove-fails", func(t *testing.T) {
		target := writeTempFile(t, "target.txt", "owned-bytes")
		id, err := journal.StatPath(target)
		if err != nil {
			t.Fatalf("StatPath: %v", err)
		}
		j := journal.OpenMemory()
		defer j.Close()
		reserve(t, j, "refuse6", self, id)
		spy := &spyRemover{err: errors.New("EIO: disk gone")}
		rec, err := Perform(j, Request{OperationID: "refuse6", Path: target, Self: self, Allow: true, Prior: OutcomeIncomplete}, spy.remove)
		if !errors.Is(err, ErrRemoveFailed) {
			t.Fatalf("Perform: got %v, want ErrRemoveFailed", err)
		}
		if rec == nil || rec.Outcome != OutcomeIncomplete || rec.Reason != ReasonRemoveFailed || rec.Complete || rec.Removed {
			t.Fatalf("receipt = %+v, want preserved incomplete remove-failed refusal", rec)
		}
		if got := readFile(t, target); got != "owned-bytes" {
			t.Fatalf("failed removal lost the target: %q", got)
		}
		if st := journalState(t, j, "refuse6"); st != journal.StateFailed {
			t.Fatalf("failed removal left journal state %s, want failed", st)
		}
	})

	t.Run("invalid-requests", func(t *testing.T) {
		j := journal.OpenMemory()
		defer j.Close()
		reserve(t, j, "refuse7", self, journal.PathIdentity{})
		reserve(t, j, "refuse7b", self, journal.PathIdentity{Path: "/absent"})
		spy := &spyRemover{}
		cases := []struct {
			name      string
			j         *journal.Journal
			req       Request
			nilRemove bool
		}{
			{"malformed-op", j, Request{OperationID: "../x", Self: self, Allow: true}, false},
			{"unknown-prior", j, Request{OperationID: "refuse7", Self: self, Allow: true, Prior: "bogus"}, false},
			{"nil-journal", nil, Request{OperationID: "refuse7", Self: self, Allow: true}, false},
			{"relative-path", j, Request{OperationID: "refuse7", Path: "rel/path", Self: self, Allow: true}, false},
			{"nil-remover-with-path", j, Request{OperationID: "refuse7b", Path: "/absent", Self: self, Allow: true}, true},
			{"zero-pid", j, Request{OperationID: "refuse7", Self: journal.Owner{}}, false},
			{"empty-token", j, Request{OperationID: "refuse7", Self: journal.Owner{PID: 1}}, false},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				var remove func(string) error
				if !c.nilRemove {
					remove = spy.remove
				}
				rec, err := Perform(c.j, c.req, remove)
				if !errors.Is(err, ErrInvalid) {
					t.Fatalf("Perform: got %v, want ErrInvalid", err)
				}
				if rec == nil || rec.Complete {
					t.Fatalf("receipt = %+v, want non-nil incomplete refusal", rec)
				}
			})
		}
		if spy.called() {
			t.Fatalf("invalid request called the remover")
		}
	})
}

// TestLaterSafeCleanupPreservesIncomplete is the headline acceptance
// clause: refusals delete nothing, and a later safe cleanup of the same
// operation deletes the target yet preserves the old incomplete outcome
// instead of rewriting history to pass.
func TestLaterSafeCleanupPreservesIncomplete(t *testing.T) {
	self := testSelf()
	target := writeTempFile(t, "target.txt", "old-run-bytes")
	id, err := journal.StatPath(target)
	if err != nil {
		t.Fatalf("StatPath: %v", err)
	}
	j := journal.OpenMemory()
	defer j.Close()
	reserve(t, j, "later1", self, id)
	spy := &spyRemover{}

	// A stranger's cleanup is refused before any deletion.
	rec, err := Perform(j, Request{OperationID: "later1", Path: target, Self: testStranger(), Allow: false, DenyReason: ReasonOwnerMismatch, Prior: OutcomeIncomplete}, spy.remove)
	if !errors.Is(err, ErrWrongOwner) {
		t.Fatalf("stranger Perform: got %v, want ErrWrongOwner", err)
	}
	if rec == nil || rec.Outcome != OutcomeIncomplete || rec.Complete {
		t.Fatalf("stranger receipt = %+v, want incomplete refusal", rec)
	}
	// A scan denial against the true owner is refused too.
	rec, err = Perform(j, Request{OperationID: "later1", Path: target, Self: self, Allow: false, DenyReason: recovery.ReasonForeignLive, Prior: OutcomeIncomplete}, spy.remove)
	if !errors.Is(err, ErrDenied) {
		t.Fatalf("denied Perform: got %v, want ErrDenied", err)
	}
	if spy.called() {
		t.Fatalf("refusals called the remover")
	}
	if got := readFile(t, target); got != "old-run-bytes" {
		t.Fatalf("refusals touched the target: %q", got)
	}
	if st := journalState(t, j, "later1"); st != journal.StateReserved {
		t.Fatalf("refusals mutated the journal to %s", st)
	}

	// The later safe cleanup deletes the target but preserves the old
	// incomplete outcome.
	rec, err = Perform(j, Request{OperationID: "later1", Path: target, Self: self, Allow: true, Prior: OutcomeIncomplete}, os.Remove)
	if err != nil {
		t.Fatalf("safe Perform: %v", err)
	}
	if rec == nil || !rec.Complete || !rec.Removed || rec.Reason != ReasonCleanupComplete {
		t.Fatalf("safe receipt = %+v, want complete cleanup", rec)
	}
	if rec.Outcome != OutcomeIncomplete {
		t.Fatalf("safe receipt outcome = %q, want incomplete (history rewritten)", rec.Outcome)
	}
	if rec.Outcome == OutcomePass {
		t.Fatalf("safe cleanup reported a pass for an incomplete run")
	}
	if _, serr := os.Lstat(target); !os.IsNotExist(serr) {
		t.Fatalf("target still present after safe cleanup: %v", serr)
	}
	if st := journalState(t, j, "later1"); st != journal.StateClosed {
		t.Fatalf("safe cleanup left journal state %s, want closed", st)
	}

	// The preserved outcome survives a receipt round trip.
	receiptPath := filepath.Join(t.TempDir(), "later1.json")
	if err := WriteFile(receiptPath, rec); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	loaded, err := LoadFile(receiptPath)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if loaded.Outcome != OutcomeIncomplete || !loaded.Complete {
		t.Fatalf("loaded receipt = %+v, want complete incomplete", loaded)
	}
}

// TestCloseWalkFromLive pins the journal walk from a non-reserved state:
// a safe cleanup of a live operation still lands closed with the prior
// outcome preserved.
func TestCloseWalkFromLive(t *testing.T) {
	self := testSelf()
	target := writeTempFile(t, "target.txt", "live-bytes")
	id, err := journal.StatPath(target)
	if err != nil {
		t.Fatalf("StatPath: %v", err)
	}
	j := journal.OpenMemory()
	defer j.Close()
	reserve(t, j, "livewalk1", self, id)
	if _, err := j.Advance("livewalk1", journal.StateOpening); err != nil {
		t.Fatalf("Advance opening: %v", err)
	}
	if _, err := j.Advance("livewalk1", journal.StateLive); err != nil {
		t.Fatalf("Advance live: %v", err)
	}
	rec, err := Perform(j, Request{OperationID: "livewalk1", Path: target, Self: self, Allow: true, Prior: OutcomeFailed}, os.Remove)
	if err != nil {
		t.Fatalf("Perform: %v", err)
	}
	if rec == nil || !rec.Complete || !rec.Removed || rec.Outcome != OutcomeFailed {
		t.Fatalf("receipt = %+v, want complete failed cleanup", rec)
	}
	if st := journalState(t, j, "livewalk1"); st != journal.StateClosed {
		t.Fatalf("cleanup left journal state %s, want closed", st)
	}
}

// TestFailedRemovalRetries pins recoverability: after a failed removal
// marks the operation failed, a later safe cleanup with a working
// remover still proceeds (authority re-verified) and preserves the
// prior outcome.
func TestFailedRemovalRetries(t *testing.T) {
	self := testSelf()
	target := writeTempFile(t, "target.txt", "retry-bytes")
	id, err := journal.StatPath(target)
	if err != nil {
		t.Fatalf("StatPath: %v", err)
	}
	j := journal.OpenMemory()
	defer j.Close()
	reserve(t, j, "retry1", self, id)
	spy := &spyRemover{err: errors.New("EIO: disk gone")}
	if _, err := Perform(j, Request{OperationID: "retry1", Path: target, Self: self, Allow: true, Prior: OutcomeIncomplete}, spy.remove); !errors.Is(err, ErrRemoveFailed) {
		t.Fatalf("first Perform: got %v, want ErrRemoveFailed", err)
	}
	if st := journalState(t, j, "retry1"); st != journal.StateFailed {
		t.Fatalf("failed removal left journal state %s, want failed", st)
	}
	rec, err := Perform(j, Request{OperationID: "retry1", Path: target, Self: self, Allow: true, Prior: OutcomeIncomplete}, os.Remove)
	if err != nil {
		t.Fatalf("retry Perform: %v", err)
	}
	if rec == nil || !rec.Complete || !rec.Removed || rec.Outcome != OutcomeIncomplete {
		t.Fatalf("retry receipt = %+v, want complete incomplete cleanup", rec)
	}
	if _, serr := os.Lstat(target); !os.IsNotExist(serr) {
		t.Fatalf("target still present after retry: %v", serr)
	}
	// The terminal failed state is kept: the failed removal happened and
	// history is not rewritten by the later success.
	if st := journalState(t, j, "retry1"); st != journal.StateFailed {
		t.Fatalf("retry left journal state %s, want failed", st)
	}
}

func TestPriorPreserved(t *testing.T) {
	self := testSelf()
	cases := []struct {
		name  string
		prior Outcome
		want  Outcome
	}{
		{"pass-stays-pass", OutcomePass, OutcomePass},
		{"incomplete-stays-incomplete", OutcomeIncomplete, OutcomeIncomplete},
		{"failed-stays-failed", OutcomeFailed, OutcomeFailed},
		{"empty-selects-incomplete", "", OutcomeIncomplete},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			j := journal.OpenMemory()
			defer j.Close()
			reserve(t, j, "prior1", self, journal.PathIdentity{})
			rec, err := Perform(j, Request{OperationID: "prior1", Self: self, Allow: true, Prior: c.prior}, nil)
			if err != nil {
				t.Fatalf("Perform: %v", err)
			}
			if rec.Outcome != c.want || !rec.Complete || rec.Removed {
				t.Fatalf("receipt = %+v, want outcome %q complete without removal", rec, c.want)
			}
		})
	}
}

// TestScanToCleanup wires the recovery decision into cleanup: a stranger's
// scan refuses, the owner's scan allows, and the later safe cleanup
// preserves the old incomplete outcome.
func TestScanToCleanup(t *testing.T) {
	self := testSelf()
	target := writeTempFile(t, "target.txt", "old-run-bytes")
	id, err := journal.StatPath(target)
	if err != nil {
		t.Fatalf("StatPath: %v", err)
	}
	j := journal.OpenMemory()
	defer j.Close()
	reserve(t, j, "e2e1", self, id)
	spy := &spyRemover{}

	rep := recovery.Scan(j, testStranger(), recovery.ScanConfig{})
	if rep.Allowed("e2e1") {
		t.Fatalf("stranger scan allowed e2e1")
	}
	reason := rep.ReasonFor("e2e1")
	if reason == "" {
		t.Fatalf("stranger scan gave no reason")
	}
	if _, err := Perform(j, Request{OperationID: "e2e1", Path: target, Self: testStranger(), Allow: false, DenyReason: reason, Prior: OutcomeIncomplete}, spy.remove); !errors.Is(err, ErrWrongOwner) {
		t.Fatalf("stranger cleanup: got %v, want ErrWrongOwner", err)
	}
	if got := readFile(t, target); got != "old-run-bytes" {
		t.Fatalf("stranger cleanup touched the target: %q", got)
	}

	rep = recovery.Scan(j, self, recovery.ScanConfig{})
	if !rep.Allowed("e2e1") {
		t.Fatalf("owner scan refused e2e1 with %q", rep.ReasonFor("e2e1"))
	}
	rec, err := Perform(j, Request{OperationID: "e2e1", Path: target, Self: self, Allow: true, Prior: OutcomeIncomplete}, os.Remove)
	if err != nil {
		t.Fatalf("safe Perform: %v", err)
	}
	if !rec.Complete || rec.Outcome != OutcomeIncomplete {
		t.Fatalf("receipt = %+v, want complete incomplete", rec)
	}
	if _, serr := os.Lstat(target); !os.IsNotExist(serr) {
		t.Fatalf("target still present after safe cleanup")
	}
	if spy.called() {
		t.Fatalf("refusal called the remover")
	}
}
