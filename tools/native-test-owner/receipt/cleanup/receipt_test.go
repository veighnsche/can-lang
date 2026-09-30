package cleanup

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReceiptRoundTrip(t *testing.T) {
	want := &Receipt{OperationID: "rcpt1", Outcome: OutcomeIncomplete, Reason: ReasonCleanupComplete, Removed: true, Complete: true}
	path := filepath.Join(t.TempDir(), "rcpt1.json")
	if err := WriteFile(path, want); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if got.SchemaVersion != SchemaVersion || got.OperationID != want.OperationID || got.Outcome != want.Outcome ||
		got.Reason != want.Reason || got.Removed != want.Removed || got.Complete != want.Complete {
		t.Fatalf("round trip = %+v, want %+v", got, want)
	}
	prior, err := PriorFromFile(path)
	if err != nil || prior != OutcomeIncomplete {
		t.Fatalf("PriorFromFile = (%q, %v), want (incomplete, nil)", prior, err)
	}
}

func TestReceiptCompact(t *testing.T) {
	r := &Receipt{SchemaVersion: SchemaVersion, OperationID: "rcpt-compact-1", Outcome: OutcomeIncomplete, Reason: ReasonCleanupComplete, Removed: true, Complete: true}
	raw, err := r.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if len(raw) > CompactReceiptBytes {
		t.Fatalf("receipt is %d bytes, want at most %d: %s", len(raw), CompactReceiptBytes, raw)
	}
	if _, err := (*Receipt)(nil).Marshal(); !errors.Is(err, ErrInvalid) {
		t.Fatalf("nil Marshal: got %v, want ErrInvalid", err)
	}
}

func TestReceiptUnknownSchemaNeverPass(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.json")
	old := `{"schemaVersion":"99","operationId":"old1","outcome":"pass","reason":"legacy","complete":true}`
	if err := os.WriteFile(path, []byte(old), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := LoadFile(path); !errors.Is(err, ErrUnknownSchema) {
		t.Fatalf("LoadFile old schema: got %v, want ErrUnknownSchema", err)
	}
	// Fail closed: an unreadable old receipt chains as incomplete,
	// never as a pass.
	prior, err := PriorFromFile(path)
	if !errors.Is(err, ErrUnknownSchema) {
		t.Fatalf("PriorFromFile old schema: got %v, want ErrUnknownSchema", err)
	}
	if prior != OutcomeIncomplete {
		t.Fatalf("PriorFromFile old schema = %q, want incomplete", prior)
	}
}

func TestReceiptLoadRefusals(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	write := func(body string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}
	write(`{"schemaVersion":"1","operationId":"x","outcome":"bogus","reason":"r","complete":false}`)
	if _, err := LoadFile(path); !errors.Is(err, ErrInvalid) {
		t.Fatalf("unknown outcome: got %v, want ErrInvalid", err)
	}
	write(`{not json`)
	if _, err := LoadFile(path); !errors.Is(err, ErrInvalid) {
		t.Fatalf("malformed JSON: got %v, want ErrInvalid", err)
	}
	write(`{"schemaVersion":"1","operationId":"` + strings.Repeat("x", MaxReceiptBytes) + `"}`)
	if _, err := LoadFile(path); !errors.Is(err, ErrInvalid) {
		t.Fatalf("oversize: got %v, want ErrInvalid", err)
	}
	if _, err := LoadFile(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Fatalf("missing file loaded without error")
	}
	if err := WriteFile("", &Receipt{}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("empty path WriteFile: got %v, want ErrInvalid", err)
	}
	if err := WriteFile(path, nil); !errors.Is(err, ErrInvalid) {
		t.Fatalf("nil receipt WriteFile: got %v, want ErrInvalid", err)
	}
	// WriteFile stamps the current schema even when the caller leaves it
	// blank.
	write(`{}`)
	if err := WriteFile(path, &Receipt{OperationID: "stamp1", Outcome: OutcomePass, Reason: "r"}); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	loaded, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if loaded.SchemaVersion != SchemaVersion {
		t.Fatalf("stamped schema = %q, want %q", loaded.SchemaVersion, SchemaVersion)
	}
}
