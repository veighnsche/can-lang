package recovery

import (
	"fmt"
	"syscall"

	"github.com/veighnsche/can-lang/tools/native-test-owner/journal"
)

// LivenessProof is a versioned claim that one operation's owner process is
// live right now. Every field is required: a proof with an unknown schema,
// a naming mismatch, or an unverifiable process authorizes nothing.
type LivenessProof struct {
	SchemaVersion string `json:"schemaVersion"`
	OperationID   string `json:"operationId"`
	PID           int    `json:"pid"`
	StartToken    string `json:"startToken"`
}

// ProbeAlive reports whether pid names a live process without signaling
// it. A permission error still proves existence; only absence (or an
// invalid PID) reports false.
func ProbeAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}

// VerifyProof checks one liveness proof against the expected operation and
// owner identity. Unknown schemas fail with ErrUnknownSchema; naming gaps,
// dead processes and unverifiable claims fail with ErrIncompleteProof; a
// live process under a foreign identity fails with ErrForeignProcess. A
// nil alive probe selects the signal-0 probe.
func VerifyProof(p LivenessProof, opID string, want journal.Owner, alive func(int) bool) error {
	if !validID(opID) {
		return fmt.Errorf("%w: malformed operation ID", ErrInvalid)
	}
	if p.SchemaVersion != SchemaVersion {
		return fmt.Errorf("%w: proof for %q names %q", ErrUnknownSchema, opID, p.SchemaVersion)
	}
	if p.OperationID != opID || p.PID <= 0 || p.StartToken == "" || len(p.StartToken) > journal.MaxStartTokenLen {
		return fmt.Errorf("%w: proof for %q names another operation or process", ErrIncompleteProof, opID)
	}
	if alive == nil {
		alive = ProbeAlive
	}
	if p.PID != want.PID || p.StartToken != want.StartToken {
		if alive(p.PID) {
			return fmt.Errorf("%w: pid %d presents a foreign start token", ErrForeignProcess, p.PID)
		}
		return fmt.Errorf("%w: proof for %q names an unverifiable process", ErrIncompleteProof, opID)
	}
	if !alive(p.PID) {
		return fmt.Errorf("%w: owner process %d is not live", ErrIncompleteProof, p.PID)
	}
	return nil
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
