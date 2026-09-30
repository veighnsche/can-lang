package recovery

import (
	"errors"
	"os"
	"testing"

	"github.com/veighnsche/can-lang/tools/native-test-owner/journal"
)

func TestVerifyProofTable(t *testing.T) {
	o := procOwner(t, journal.OpenMemory(), "p11-proof-proc")
	foreign := spawnSleeper(t, o, "proof-foreign", 43)
	defer func() {
		_ = o.Abort(foreign)
		assertNoStraySleepers(t, "sleep 43")
	}()
	dead := deadPID(t, o, "proof")
	self := testSelf()

	complete := LivenessProof{SchemaVersion: "1", OperationID: "proof1", PID: self.PID, StartToken: self.StartToken}
	neverAlive := func(int) bool { return false }
	alwaysAlive := func(int) bool { return true }

	cases := []struct {
		name    string
		proof   LivenessProof
		opID    string
		want    journal.Owner
		alive   func(int) bool
		wantErr error
	}{
		{"complete", complete, "proof1", self, nil, nil},
		{"complete-explicit-alive", complete, "proof1", self, alwaysAlive, nil},
		{"unknown-schema", LivenessProof{SchemaVersion: "99", OperationID: "proof1", PID: self.PID, StartToken: self.StartToken}, "proof1", self, nil, ErrUnknownSchema},
		{"empty-schema", LivenessProof{OperationID: "proof1", PID: self.PID, StartToken: self.StartToken}, "proof1", self, nil, ErrUnknownSchema},
		{"operation-mismatch", LivenessProof{SchemaVersion: "1", OperationID: "other", PID: self.PID, StartToken: self.StartToken}, "proof1", self, nil, ErrIncompleteProof},
		{"zero-pid", LivenessProof{SchemaVersion: "1", OperationID: "proof1", StartToken: self.StartToken}, "proof1", self, nil, ErrIncompleteProof},
		{"empty-token", LivenessProof{SchemaVersion: "1", OperationID: "proof1", PID: self.PID}, "proof1", self, nil, ErrIncompleteProof},
		{"foreign-live", LivenessProof{SchemaVersion: "1", OperationID: "proof1", PID: foreign.PID, StartToken: "foreign-token"}, "proof1", self, nil, ErrForeignProcess},
		{"foreign-dead", LivenessProof{SchemaVersion: "1", OperationID: "proof1", PID: dead, StartToken: "dead-token"}, "proof1", self, nil, ErrIncompleteProof},
		{"owner-dead", complete, "proof1", self, neverAlive, ErrIncompleteProof},
		{"malformed-op", complete, "../escape", self, nil, ErrInvalid},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := VerifyProof(c.proof, c.opID, c.want, c.alive)
			if c.wantErr == nil {
				if err != nil {
					t.Fatalf("VerifyProof: %v", err)
				}
				return
			}
			if !errors.Is(err, c.wantErr) {
				t.Fatalf("VerifyProof: got %v, want %v", err, c.wantErr)
			}
		})
	}
	if !ProbeAlive(foreign.PID) {
		t.Fatalf("foreign sleeper died during proof checks (pid %d)", foreign.PID)
	}
	if ProbeAlive(-1) || ProbeAlive(0) {
		t.Fatalf("ProbeAlive reports invalid PIDs live")
	}
	if !ProbeAlive(os.Getpid()) {
		t.Fatalf("ProbeAlive reports our own PID dead")
	}
}

func TestReasonMapping(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{ErrUnknownSchema, ReasonUnknownSchema},
		{ErrForeignProcess, ReasonForeignLive},
		{ErrOwnerMismatch, ReasonOwnerMismatch},
		{ErrReplacedPath, ReasonReplacedPath},
		{ErrIncompleteProof, ReasonIncompleteProof},
		{ErrInvalid, ReasonDenied},
		{ErrDenied, ReasonDenied},
		{errors.New("boom"), ReasonDenied},
	}
	for _, c := range cases {
		if got := Reason(c.err); got != c.want {
			t.Errorf("Reason(%v) = %q, want %q", c.err, got, c.want)
		}
	}
}
