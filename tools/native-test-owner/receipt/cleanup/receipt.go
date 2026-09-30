package cleanup

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// SchemaVersion is the only receipt layout this package reads. Older or
// newer layouts fail closed: an unreadable old receipt is never a pass.
const SchemaVersion = "1"

// Transitional bounds. Every field is finite; nothing here means unlimited.
const (
	// MaxReceiptBytes caps a receipt file read. Receipts are compact by
	// construction; anything larger is not a receipt this package trusts.
	MaxReceiptBytes = 64 << 10
	// MaxReasonLen caps a refusal reason recorded in a receipt.
	MaxReasonLen = 256
	// CompactReceiptBytes is the envelope a typical completion receipt
	// must fit: compact evidence, not an execution workspace.
	CompactReceiptBytes = 1024
)

// Outcome is the recorded run outcome. Cleanup preserves it; it never
// rewrites history in either direction.
type Outcome string

// Recorded outcomes.
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

// Receipt is the compact terminal fact of one cleanup: which operation,
// the preserved prior outcome, a named reason, and whether the target is
// gone. Outcome is the old outcome, never an upgrade: a later safe
// cleanup of an incomplete run reports Complete with Outcome incomplete.
type Receipt struct {
	SchemaVersion string  `json:"schemaVersion"`
	OperationID   string  `json:"operationId"`
	Outcome       Outcome `json:"outcome"`
	Reason        string  `json:"reason"`
	Removed       bool    `json:"removed,omitempty"`
	Complete      bool    `json:"complete"`
}

// Marshal encodes one receipt. The encoding is compact by construction.
func (r *Receipt) Marshal() ([]byte, error) {
	if r == nil {
		return nil, fmt.Errorf("%w: nil receipt", ErrInvalid)
	}
	raw, err := json.Marshal(r)
	if err != nil {
		return nil, fmt.Errorf("cleanup: encode receipt: %w", err)
	}
	return raw, nil
}

// WriteFile stores one receipt, stamping the current schema version.
func WriteFile(path string, r *Receipt) error {
	if path == "" {
		return fmt.Errorf("%w: empty receipt path", ErrInvalid)
	}
	if r == nil {
		return fmt.Errorf("%w: nil receipt", ErrInvalid)
	}
	cp := *r
	cp.SchemaVersion = SchemaVersion
	raw, err := cp.Marshal()
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return fmt.Errorf("cleanup: write receipt: %w", err)
	}
	return nil
}

// LoadFile reads one receipt. Unknown schemas, unknown outcomes and
// oversized or malformed files fail closed: none of them is ever a pass.
// The size bound is enforced during the read, so a hostile oversized
// file cannot exhaust memory before it is rejected.
func LoadFile(path string) (*Receipt, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cleanup: read receipt: %w", err)
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, MaxReceiptBytes+1))
	if err != nil {
		return nil, fmt.Errorf("cleanup: read receipt: %w", err)
	}
	if len(raw) > MaxReceiptBytes {
		return nil, fmt.Errorf("%w: receipt exceeds bound", ErrInvalid)
	}
	var r Receipt
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("%w: malformed receipt: %v", ErrInvalid, err)
	}
	if r.SchemaVersion != SchemaVersion {
		return nil, fmt.Errorf("%w: receipt for %q names %q", ErrUnknownSchema, r.OperationID, r.SchemaVersion)
	}
	if !r.Outcome.valid() {
		return nil, fmt.Errorf("%w: receipt for %q names unknown outcome %q", ErrInvalid, r.OperationID, r.Outcome)
	}
	return &r, nil
}

// PriorFromFile loads an old receipt and reports its outcome for
// chaining into a later cleanup. Any load failure — including an unknown
// schema — fails closed with OutcomeIncomplete: the caller must treat the
// old run as incomplete, never as a pass. Callers must check the error;
// the outcome alone authorizes nothing.
func PriorFromFile(path string) (Outcome, error) {
	r, err := LoadFile(path)
	if err != nil {
		return OutcomeIncomplete, err
	}
	return r.Outcome, nil
}
