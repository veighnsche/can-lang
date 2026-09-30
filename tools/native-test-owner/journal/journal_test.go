package journal

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testDigest = "sha256:" + "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func otherDigest() string {
	return "sha256:" + "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"
}

func testOwner() Owner { return Owner{PID: os.Getpid(), StartToken: "start-1"} }

func TestReserveRejoinAndConflict(t *testing.T) {
	j := OpenMemory()
	defer j.Close()
	r1, err := j.Reserve("op-1", testDigest, testOwner(), PathIdentity{}, "lease-1", 0)
	if err != nil {
		t.Fatalf("Reserve: %v", err)
	}
	if r1.State != StateReserved || r1.Seq == 0 {
		t.Fatalf("unexpected reservation: %+v", r1)
	}
	// Same ID, same digest rejoins without a new effect.
	r2, err := j.Reserve("op-1", testDigest, testOwner(), PathIdentity{}, "lease-1", 0)
	if err != nil {
		t.Fatalf("rejoin Reserve: %v", err)
	}
	if r2.Seq != r1.Seq {
		t.Fatalf("rejoin created a new effect: seq %d -> %d", r1.Seq, r2.Seq)
	}
	// Same ID, changed arguments is a protocol error.
	if _, err := j.Reserve("op-1", otherDigest(), testOwner(), PathIdentity{}, "", 0); !errors.Is(err, ErrConflict) {
		t.Fatalf("conflicting replay: got %v, want ErrConflict", err)
	}
}

func TestAdvanceMonotonic(t *testing.T) {
	j := OpenMemory()
	defer j.Close()
	if _, err := j.Reserve("op-1", testDigest, testOwner(), PathIdentity{}, "", 0); err != nil {
		t.Fatal(err)
	}
	chain := []State{StateOpening, StateLive, StateClosing, StateClosed}
	for i, s := range chain {
		r, err := j.Advance("op-1", s, "partial-"+string(rune('a'+i)))
		if err != nil {
			t.Fatalf("Advance to %s: %v", s, err)
		}
		if r.State != s {
			t.Fatalf("state = %s, want %s", r.State, s)
		}
	}
	// Idempotent repeat of the terminal state.
	if _, err := j.Advance("op-1", StateClosed); err != nil {
		t.Fatalf("idempotent Advance: %v", err)
	}
	// Backward move rejected.
	if _, err := j.Advance("op-1", StateLive); !errors.Is(err, ErrTransition) {
		t.Fatalf("backward Advance: got %v, want ErrTransition", err)
	}
	// Unknown operation rejected.
	if _, err := j.Advance("op-missing", StateOpening); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown Advance: got %v, want ErrNotFound", err)
	}
	// Failed is terminal from any non-terminal state.
	if _, err := j.Reserve("op-2", testDigest, testOwner(), PathIdentity{}, "", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := j.Advance("op-2", StateFailed, "spawn-errno"); err != nil {
		t.Fatal(err)
	}
	if _, err := j.Advance("op-2", StateClosed); !errors.Is(err, ErrTransition) {
		t.Fatalf("advance after failed: got %v, want ErrTransition", err)
	}
	r, _ := j.Lookup("op-2")
	if len(r.Partial) != 1 || r.Partial[0] != "spawn-errno" {
		t.Fatalf("partial acquisitions lost: %+v", r)
	}
}

func TestInvalidRequests(t *testing.T) {
	j := OpenMemory()
	defer j.Close()
	bad := []struct {
		name   string
		opID   string
		digest string
		owner  Owner
		path   PathIdentity
	}{
		{"empty op id", "", testDigest, testOwner(), PathIdentity{}},
		{"bad op id", "../escape", testDigest, testOwner(), PathIdentity{}},
		{"bad digest", "op-1", "not-a-digest", testOwner(), PathIdentity{}},
		{"short digest", "op-1", "sha256:abc", testOwner(), PathIdentity{}},
		{"zero pid", "op-1", testDigest, Owner{}, PathIdentity{}},
		{"missing start token", "op-1", testDigest, Owner{PID: 1}, PathIdentity{}},
		{"relative path", "op-1", testDigest, testOwner(), PathIdentity{Path: "rel/path"}},
	}
	for _, c := range bad {
		if _, err := j.Reserve(c.opID, c.digest, c.owner, c.path, "", 0); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: got %v, want ErrInvalid", c.name, err)
		}
	}
	if _, err := j.Advance("op-1", State("bogus")); !errors.Is(err, ErrInvalid) {
		t.Errorf("bogus state: got %v, want ErrInvalid", err)
	}
}

// TestCrashBetweenReservationAndMarker reserves durably, then drops the handle
// without closing (simulated crash). Reopening must rediscover the pending
// reservation and allow it to advance.
func TestCrashBetweenReservationAndMarker(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "scratch", "marker")
	j1, err := OpenDir(filepath.Join(dir, "ledger"))
	if err != nil {
		t.Fatal(err)
	}
	pi, err := StatPath(marker) // marker does not exist yet: path-only identity
	if err != nil {
		t.Fatal(err)
	}
	if pi.HasFileID {
		t.Fatal("nonexistent marker must not carry file identity")
	}
	if _, err := j1.Reserve("op-1", testDigest, testOwner(), pi, "lease-1", 0); err != nil {
		t.Fatal(err)
	}
	// Crash: abandon j1 without Close. (Defer close for FD hygiene only.)
	defer j1.Close()

	j2, err := OpenDir(filepath.Join(dir, "ledger"))
	if err != nil {
		t.Fatal(err)
	}
	defer j2.Close()
	pending := j2.Pending()
	if len(pending) != 1 || pending[0].OperationID != "op-1" || pending[0].State != StateReserved {
		t.Fatalf("pending after crash: %+v", pending)
	}
	// The marker effect lands after recovery; authority still verifies.
	if err := os.MkdirAll(filepath.Dir(marker), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(marker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := j2.VerifyAuthority("op-1", testOwner(), marker); err != nil {
		t.Fatalf("VerifyAuthority after marker creation: %v", err)
	}
	if _, err := j2.Advance("op-1", StateOpening, "marker-created"); err != nil {
		t.Fatalf("Advance after recovery: %v", err)
	}
}

// TestTornTailRecovery appends a partial line (crash mid-write) and requires
// the journal to truncate it while keeping earlier records.
func TestTornTailRecovery(t *testing.T) {
	dir := t.TempDir()
	j1, err := OpenDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := j1.Reserve("op-1", testDigest, testOwner(), PathIdentity{}, "", 0); err != nil {
		t.Fatal(err)
	}
	if err := j1.Close(); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(filepath.Join(dir, logFileName), os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(`{"seq":999,"op":"reser`); err != nil {
		t.Fatal(err)
	}
	f.Close()

	j2, err := OpenDir(dir)
	if err != nil {
		t.Fatalf("reopen with torn tail: %v", err)
	}
	defer j2.Close()
	if _, ok := j2.Lookup("op-1"); !ok {
		t.Fatal("earlier reservation lost after torn-tail truncation")
	}
	// Journal remains usable: the truncated sequence space is reused cleanly.
	if _, err := j2.Advance("op-1", StateOpening); err != nil {
		t.Fatalf("Advance after torn-tail recovery: %v", err)
	}
}

// TestCorruptMiddleLine requires a damaged committed record to fail closed.
func TestCorruptMiddleLine(t *testing.T) {
	dir := t.TempDir()
	j1, err := OpenDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := j1.Reserve("op-1", testDigest, testOwner(), PathIdentity{}, "", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := j1.Reserve("op-2", testDigest, testOwner(), PathIdentity{}, "", 0); err != nil {
		t.Fatal(err)
	}
	if err := j1.Close(); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, logFileName)
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(raw), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected two log lines, got %d", len(lines))
	}
	lines[0] = `{"seq":1,"op":"bogus"}`
	if err := os.WriteFile(p, []byte(strings.Join(lines, "\n")), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenDir(dir); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("reopen corrupt log: got %v, want ErrCorrupt", err)
	}
}

func TestWrongAuthority(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "marker")
	if err := os.WriteFile(marker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	pi, err := StatPath(marker)
	if err != nil {
		t.Fatal(err)
	}
	if !pi.HasFileID {
		t.Fatal("existing marker must carry file identity")
	}
	j := OpenMemory()
	defer j.Close()
	if _, err := j.Reserve("op-1", testDigest, testOwner(), pi, "", 0); err != nil {
		t.Fatal(err)
	}
	// Correct owner and path verifies.
	if err := j.VerifyAuthority("op-1", testOwner(), marker); err != nil {
		t.Fatalf("correct authority rejected: %v", err)
	}
	// Wrong PID is not authority.
	wrongPID := testOwner()
	wrongPID.PID++
	if err := j.VerifyAuthority("op-1", wrongPID, marker); !errors.Is(err, ErrWrongOwner) {
		t.Errorf("wrong PID: got %v, want ErrWrongOwner", err)
	}
	// Wrong start token is not authority (PID reuse without the token).
	wrongTok := testOwner()
	wrongTok.StartToken = "start-2"
	if err := j.VerifyAuthority("op-1", wrongTok, marker); !errors.Is(err, ErrWrongOwner) {
		t.Errorf("wrong start token: got %v, want ErrWrongOwner", err)
	}
	// Wrong path is not authority.
	if err := j.VerifyAuthority("op-1", testOwner(), marker+"-other"); !errors.Is(err, ErrWrongPath) {
		t.Errorf("wrong path: got %v, want ErrWrongPath", err)
	}
	// A replaced path (same name, new object) is not authority.
	if err := os.Remove(marker); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(marker, []byte("y"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := j.VerifyAuthority("op-1", testOwner(), marker); !errors.Is(err, ErrWrongPath) {
		t.Errorf("replaced path: got %v, want ErrWrongPath", err)
	}
	// A deleted path is not authority either.
	if err := os.Remove(marker); err != nil {
		t.Fatal(err)
	}
	if err := j.VerifyAuthority("op-1", testOwner(), marker); !errors.Is(err, ErrWrongPath) {
		t.Errorf("deleted path: got %v, want ErrWrongPath", err)
	}
}

// TestSyncFault injects a durability failure and requires the reservation to
// report the error without recording an effect.
func TestSyncFault(t *testing.T) {
	j := OpenMemory()
	defer j.Close()
	j.failSync = true
	if _, err := j.Reserve("op-1", testDigest, testOwner(), PathIdentity{}, "", 0); !errors.Is(err, errSyncFault) {
		t.Fatalf("faulted Reserve: got %v, want sync fault", err)
	}
	if _, ok := j.Lookup("op-1"); ok {
		t.Fatal("faulted reservation must not record an effect")
	}
	// The fault fires once; the next reservation succeeds.
	if _, err := j.Reserve("op-1", testDigest, testOwner(), PathIdentity{}, "", 0); err != nil {
		t.Fatalf("Reserve after fault: %v", err)
	}
}

func TestPendingExcludesTerminal(t *testing.T) {
	dir := t.TempDir()
	j, err := OpenDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer j.Close()
	for _, id := range []string{"op-1", "op-2", "op-3"} {
		if _, err := j.Reserve(id, testDigest, testOwner(), PathIdentity{}, "", 0); err != nil {
			t.Fatal(err)
		}
	}
	for _, s := range []State{StateOpening, StateLive, StateClosing, StateClosed} {
		if _, err := j.Advance("op-1", s); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := j.Advance("op-2", StateFailed); err != nil {
		t.Fatal(err)
	}
	pending := j.Pending()
	if len(pending) != 1 || pending[0].OperationID != "op-3" {
		t.Fatalf("Pending: %+v, want only op-3", pending)
	}
}

func TestClosedJournal(t *testing.T) {
	j := OpenMemory()
	if _, err := j.Reserve("op-1", testDigest, testOwner(), PathIdentity{}, "", 0); err != nil {
		t.Fatal(err)
	}
	if err := j.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := j.Reserve("op-2", testDigest, testOwner(), PathIdentity{}, "", 0); !errors.Is(err, ErrClosed) {
		t.Errorf("Reserve after Close: got %v, want ErrClosed", err)
	}
	if _, err := j.Advance("op-1", StateOpening); !errors.Is(err, ErrClosed) {
		t.Errorf("Advance after Close: got %v, want ErrClosed", err)
	}
	if err := j.VerifyAuthority("op-1", testOwner(), ""); !errors.Is(err, ErrClosed) {
		t.Errorf("VerifyAuthority after Close: got %v, want ErrClosed", err)
	}
	// Reads stay available for inspection.
	if _, ok := j.Lookup("op-1"); !ok {
		t.Error("Lookup after Close should still report")
	}
	if len(j.Pending()) != 1 {
		t.Error("Pending after Close should still report")
	}
}
