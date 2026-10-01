package corrections

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// SchemaVersion is the only record layout this package reads. Older or
// newer layouts fail closed: an unreadable old chain is never resolved.
const SchemaVersion = "1"

// GenesisPrev is the PrevDigest of the seq-0 record and the Head of an
// empty registry. The Can authority pins the same word structurally.
const GenesisPrev = "genesis"

// Transitional bounds. Every field is finite; nothing here means unlimited.
const (
	// MaxRegistryBytes caps a registry file read. Registries are
	// compact by construction; anything larger is not a registry this
	// package trusts.
	MaxRegistryBytes = 64 << 10
	// MaxNoteLen caps a correction note recorded in a record.
	MaxNoteLen = 256
	// MaxCaseIDLen caps a correction case identity.
	MaxCaseIDLen = 128
)

// Outcome is the recorded standing of one correction: what the corrected
// run showed. Resolve preserves history; it never upgrades an old
// failure or incomplete to a pass.
type Outcome string

// Recorded standings.
const (
	OutcomePass       Outcome = "pass"
	OutcomeIncomplete Outcome = "incomplete"
	OutcomeFailed     Outcome = "failed"
)

func (o Outcome) valid() bool {
	switch o {
	case OutcomePass, OutcomeIncomplete, OutcomeFailed:
		return true
	}
	return false
}

// Record is one correction in the chain: its position, its link to the
// previous digest, its own digest, the corrected case, the recorded
// standing, and a human note. Digest commits to every other field.
type Record struct {
	SchemaVersion string  `json:"schemaVersion"`
	Seq           int     `json:"seq"`
	PrevDigest    string  `json:"prevDigest"`
	Digest        string  `json:"digest"`
	CaseID        string  `json:"caseId"`
	Outcome       Outcome `json:"outcome"`
	Note          string  `json:"note"`
}

// Entry is the caller-supplied body of one correction. Seq, PrevDigest
// and Digest are assigned by Append, never by the caller.
type Entry struct {
	CaseID  string
	Outcome Outcome
	Note    string
}

// Registry is a verified in-memory correction chain: records in seq
// order with intact links. The zero value is a healthy empty chain.
type Registry struct {
	Records []Record
}

// Resolution is the conservative current-chain standing: the sticky
// outcome over the full history, the head digest it was resolved at,
// and the record count behind it.
type Resolution struct {
	Standing Outcome
	Head     string
	Count    int
}

// digestBody is the canonical digest input: every committed field.
// JSON field order is fixed by the struct, so the encoding — and the
// digest — is deterministic for equal inputs.
type digestBody struct {
	Schema string  `json:"schemaVersion"`
	Seq    int     `json:"seq"`
	Prev   string  `json:"prevDigest"`
	Case   string  `json:"caseId"`
	Out    Outcome `json:"outcome"`
	Note   string  `json:"note"`
}

// Digest computes the chain digest for one record body: "sha256:<hex>"
// over the canonical JSON encoding of the committed fields.
func Digest(prev string, seq int, caseID string, outcome Outcome, note string) string {
	raw, _ := json.Marshal(digestBody{Schema: SchemaVersion, Seq: seq, Prev: prev, Case: caseID, Out: outcome, Note: note})
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// digestOf recomputes the digest a record must carry.
func digestOf(r Record) string {
	return Digest(r.PrevDigest, r.Seq, r.CaseID, r.Outcome, r.Note)
}

// Head reports the current chain head: the last record's digest, or the
// genesis word when no correction is recorded yet.
func (r *Registry) Head() string {
	if r == nil || len(r.Records) == 0 {
		return GenesisPrev
	}
	return r.Records[len(r.Records)-1].Digest
}

// Verify checks one record vector as a chain: seq numbers run with the
// index, every schema is current, every identity and outcome is known,
// every prev links to its predecessor (genesis for seq 0), and every
// digest recomputes. The first breakage refuses the whole chain.
func Verify(records []Record) error {
	for i, r := range records {
		if r.SchemaVersion != SchemaVersion {
			return fmt.Errorf("%w: record %d names %q", ErrUnknownSchema, i, r.SchemaVersion)
		}
		if r.Seq != i {
			return fmt.Errorf("%w: record %d carries seq %d", ErrBrokenChain, i, r.Seq)
		}
		if !validID(r.CaseID) {
			return fmt.Errorf("%w: record %d carries malformed case ID", ErrInvalid, i)
		}
		if !r.Outcome.valid() {
			return fmt.Errorf("%w: record %d names unknown outcome %q", ErrInvalid, i, r.Outcome)
		}
		if len(r.Note) > MaxNoteLen {
			return fmt.Errorf("%w: record %d note exceeds bound", ErrInvalid, i)
		}
		wantPrev := GenesisPrev
		if i > 0 {
			wantPrev = records[i-1].Digest
		}
		if r.PrevDigest != wantPrev {
			return fmt.Errorf("%w: record %d prev does not link", ErrBrokenChain, i)
		}
		if r.Digest == "" || r.Digest != digestOf(r) {
			return fmt.Errorf("%w: record %d digest does not recompute", ErrBrokenChain, i)
		}
	}
	return nil
}

// Resolve derives the conservative current standing over the full
// history. The chain is re-verified first: any breakage refuses, and no
// standing is reported. Otherwise the worst recorded standing sticks — a
// fresh pass appended after a failure still resolves to that failure,
// and an incomplete anywhere (with no failure) resolves to incomplete.
// Only an all-pass (or empty) chain resolves to pass.
func (r *Registry) Resolve() (Resolution, error) {
	var records []Record
	if r != nil {
		records = r.Records
	}
	if err := Verify(records); err != nil {
		return Resolution{}, err
	}
	head := GenesisPrev
	standing := OutcomePass
	if len(records) > 0 {
		head = records[len(records)-1].Digest
	}
	for _, rec := range records {
		switch rec.Outcome {
		case OutcomeFailed:
			standing = OutcomeFailed
		case OutcomeIncomplete:
			if standing != OutcomeFailed {
				standing = OutcomeIncomplete
			}
		}
		if standing == OutcomeFailed {
			break
		}
	}
	return Resolution{Standing: standing, Head: head, Count: len(records)}, nil
}

// Load reads and verifies one registry file. A missing file is a healthy
// empty chain; every other failure — oversize, malformed lines, unknown
// schemas, broken links — refuses. The size bound is enforced during the
// read, so a hostile oversized file cannot exhaust memory before it is
// rejected.
func Load(path string) (*Registry, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Registry{}, nil
		}
		return nil, fmt.Errorf("corrections: read registry: %w", err)
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, MaxRegistryBytes+1))
	if err != nil {
		return nil, fmt.Errorf("corrections: read registry: %w", err)
	}
	if len(raw) > MaxRegistryBytes {
		return nil, fmt.Errorf("%w: registry exceeds bound", ErrInvalid)
	}
	if len(raw) == 0 {
		return &Registry{}, nil
	}
	var records []Record
	line := 0
	rest := raw
	for len(rest) > 0 {
		line++
		end := 0
		for end < len(rest) && rest[end] != '\n' {
			end++
		}
		body := rest[:end]
		if end < len(rest) {
			end++ // consume the newline
		}
		rest = rest[end:]
		if len(body) == 0 {
			continue
		}
		var r Record
		if err := json.Unmarshal(body, &r); err != nil {
			return nil, fmt.Errorf("%w: malformed record on line %d: %v", ErrInvalid, line, err)
		}
		records = append(records, r)
	}
	if err := Verify(records); err != nil {
		return nil, err
	}
	return &Registry{Records: records}, nil
}

// Append verifies the existing chain and appends one correction record
// with the next seq, the current head as prev, and a fresh digest. A
// refused append — an invalid entry or a broken existing chain — writes
// nothing and preserves the prior bytes. Append never rewrites history:
// the old records keep their seqs and digests verbatim.
func Append(path string, e Entry) (Record, error) {
	var zero Record
	if path == "" {
		return zero, fmt.Errorf("%w: empty registry path", ErrInvalid)
	}
	if !validID(e.CaseID) {
		return zero, fmt.Errorf("%w: malformed case ID", ErrInvalid)
	}
	if !e.Outcome.valid() {
		return zero, fmt.Errorf("%w: unknown outcome %q", ErrInvalid, e.Outcome)
	}
	if len(e.Note) > MaxNoteLen {
		return zero, fmt.Errorf("%w: note exceeds bound", ErrInvalid)
	}
	reg, err := Load(path)
	if err != nil {
		return zero, err
	}
	rec := Record{
		SchemaVersion: SchemaVersion,
		Seq:           len(reg.Records),
		PrevDigest:    reg.Head(),
		CaseID:        e.CaseID,
		Outcome:       e.Outcome,
		Note:          e.Note,
	}
	rec.Digest = digestOf(rec)
	raw, err := json.Marshal(rec)
	if err != nil {
		return zero, fmt.Errorf("corrections: encode record: %w", err)
	}
	raw = append(raw, '\n')
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return zero, fmt.Errorf("corrections: append record: %w", err)
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	if _, err := w.Write(raw); err != nil {
		return zero, fmt.Errorf("corrections: append record: %w", err)
	}
	if err := w.Flush(); err != nil {
		return zero, fmt.Errorf("corrections: append record: %w", err)
	}
	return rec, nil
}

// validID matches the case-ID wire shape:
// ^[A-Za-z0-9][A-Za-z0-9_.:/-]{0,127}$. Case identities carry slashes
// ("process/nonzero-result"), which the operation-ID shape forbids, so
// the registry pins its own shape here instead of reusing it.
func validID(s string) bool {
	if len(s) == 0 || len(s) > MaxCaseIDLen {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		ok := c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' ||
			c == '_' || c == '.' || c == ':' || c == '-' || c == '/'
		if !ok {
			return false
		}
		if i == 0 && !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}
