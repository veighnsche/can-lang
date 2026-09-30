package recovery

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/veighnsche/can-lang/tools/native-test-owner/journal"
)

// SchemaVersion is the only recovery layout this scanner understands. A
// journal directory carrying any other marker is an old (or newer) run
// layout: every pending operation is refused with ReasonUnknownSchema and
// nothing is signaled, deleted or passed.
const SchemaVersion = "1"

// schemaMarkerName is the optional version file read by ScanDir. The
// journal itself ignores it; its absence selects the current schema.
const schemaMarkerName = "recovery-schema"

// maxSchemaMarkerBytes caps the marker read; anything larger is not a
// schema this scanner knows.
const maxSchemaMarkerBytes = 64

// Decision is one scanned operation: either recoverable here (Allow) or
// refused with a named reason. A refusal authorizes no signal, no deletion
// and no old-run pass.
type Decision struct {
	OperationID string
	Allow       bool
	Reason      string // "" when Allow; otherwise a Reason* constant
	Owner       journal.Owner
	State       journal.State
}

// Report is one recovery scan: the observed schema plus one decision per
// pending journaled operation in reservation order.
type Report struct {
	SchemaVersion string
	Decisions     []Decision
}

// Allowed reports whether the scan allowed opID. Unknown IDs report false:
// an absent decision is never authority.
func (r *Report) Allowed(opID string) bool {
	if r == nil {
		return false
	}
	for _, d := range r.Decisions {
		if d.OperationID == opID {
			return d.Allow
		}
	}
	return false
}

// ReasonFor reports the named refusal reason for opID, or "" when the
// operation was allowed. Unknown IDs report ReasonDenied.
func (r *Report) ReasonFor(opID string) string {
	if r == nil {
		return ReasonDenied
	}
	for _, d := range r.Decisions {
		if d.OperationID == opID {
			return d.Reason
		}
	}
	return ReasonDenied
}

// ScanConfig tunes a scan. The zero value probes real process liveness,
// requires no proofs and assumes the current schema.
type ScanConfig struct {
	// Alive overrides the liveness probe; nil selects ProbeAlive.
	Alive func(pid int) bool
	// Proofs, when non-nil, requires a complete liveness proof per
	// operation; a missing or incomplete proof refuses with a named
	// reason. Nil disables the proof gate.
	Proofs map[string]LivenessProof
	// ObservedSchema overrides the on-disk marker: "" selects the
	// current schema, anything else must equal SchemaVersion or every
	// decision refuses with ReasonUnknownSchema.
	ObservedSchema string
}

// Scan replays journal intent without mutating it: no advances, no
// signals, no deletions. Each pending operation is classified in order:
// unknown schema refuses everything; a foreign owner refuses (with a
// distinct reason when its PID is still live, so callers never signal
// it); a replaced path refuses; a required-but-incomplete liveness proof
// refuses; anything left is recoverable here. Identity is checked before
// path: an intact target under a dead or foreign owner is still refused.
func Scan(j *journal.Journal, self journal.Owner, cfg ScanConfig) *Report {
	alive := cfg.Alive
	if alive == nil {
		alive = ProbeAlive
	}
	schema := cfg.ObservedSchema
	if schema == "" {
		schema = SchemaVersion
	}
	rep := &Report{SchemaVersion: schema}
	unknownSchema := schema != SchemaVersion
	if j == nil {
		return rep
	}
	for _, r := range j.Pending() {
		d := Decision{OperationID: r.OperationID, Owner: r.Owner, State: r.State}
		switch {
		case unknownSchema:
			d.Reason = ReasonUnknownSchema
		case r.Owner.PID != self.PID || r.Owner.StartToken != self.StartToken:
			// Worker/controller/owner death with an intact target
			// lands here: the target may be untouched, but a
			// stranger's scan is never deletion or signal
			// authority. A live foreign PID gets its own reason.
			if alive(r.Owner.PID) {
				d.Reason = ReasonForeignLive
			} else {
				d.Reason = ReasonOwnerMismatch
			}
		case pathReplaced(r.Path):
			d.Reason = ReasonReplacedPath
		case cfg.Proofs != nil:
			p, ok := cfg.Proofs[r.OperationID]
			if !ok {
				d.Reason = ReasonIncompleteProof
				break
			}
			if err := VerifyProof(p, r.OperationID, self, alive); err != nil {
				d.Reason = Reason(err)
			} else {
				d.Allow = true
			}
		default:
			d.Allow = true
		}
		rep.Decisions = append(rep.Decisions, d)
	}
	return rep
}

// pathReplaced reports whether a reserved path can no longer be proven to
// be the same object: a vanished target, an unreadable target, or a
// device/inode mismatch all refuse deletion authority. Path-only
// reservations (HasFileID false) cannot prove replacement and pass here;
// cleanup still re-verifies authority before touching anything.
func pathReplaced(p journal.PathIdentity) bool {
	if p.Path == "" || !p.HasFileID {
		return false
	}
	now, err := journal.StatPath(p.Path)
	if err != nil || !now.HasFileID {
		return true
	}
	return now.Dev != p.Dev || now.Ino != p.Ino
}

// ScanDir opens the journal in dir, reads the optional recovery-schema
// marker, and scans as self. Opening replays the log; a torn final line
// may be truncated by the journal itself, which is the only mutation
// ScanDir performs: it never advances journal state, signals a process,
// deletes a path, or reports an old-run pass. An unknown or unreadable
// marker refuses every pending operation with ReasonUnknownSchema.
func ScanDir(dir string, self journal.Owner, cfg ScanConfig) (*Report, error) {
	if dir == "" {
		return nil, fmt.Errorf("%w: empty journal directory", ErrInvalid)
	}
	j, err := journal.OpenDir(dir)
	if err != nil {
		return nil, fmt.Errorf("recovery: open journal: %w", err)
	}
	defer j.Close()
	if cfg.ObservedSchema == "" {
		raw, rerr := os.ReadFile(filepath.Join(dir, schemaMarkerName))
		switch {
		case rerr == nil:
			cfg.ObservedSchema = strings.TrimSpace(string(raw))
			if len(raw) > maxSchemaMarkerBytes || cfg.ObservedSchema == "" {
				cfg.ObservedSchema = "unparseable-schema-marker"
			}
		case os.IsNotExist(rerr):
			cfg.ObservedSchema = SchemaVersion
		default:
			// An unreadable marker fails closed: the layout is not
			// provably current.
			cfg.ObservedSchema = "unreadable-schema-marker"
		}
	}
	return Scan(j, self, cfg), nil
}
