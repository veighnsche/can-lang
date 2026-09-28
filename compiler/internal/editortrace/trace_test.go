package editortrace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type traceRow struct {
	Sequence int64           `json:"sequence"`
	Method   string          `json:"method"`
	ID       json.RawMessage `json:"id"`
	Version  int64           `json:"version"`
	Stage    string          `json:"stage"`
	Duration int64           `json:"duration_ns"`
}

func readRows(t *testing.T, path string) []traceRow {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var rows []traceRow
	for _, line := range strings.Split(string(data), "\n") {
		if line == "" {
			continue
		}
		var row traceRow
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			t.Fatalf("invalid trace row %q: %v", line, err)
		}
		rows = append(rows, row)
	}
	return rows
}

// Tests in this package share the process-global recorder and must stay
// sequential: each Open is closed before the test returns.

// Disabled tracing records nothing and creates no sink.
func TestDisabledIsNoOutput(t *testing.T) {
	if Enabled() {
		t.Fatal("tracer unexpectedly active")
	}
	endRequest := Request("initialize", nil, 0)
	endStage := Stage("snapshot")
	endStage()
	endRequest()
	endRequest()
	if Enabled() {
		t.Fatal("tracer activated without Open")
	}
	SetRequestVersion(3)
	if Enabled() {
		t.Fatal("SetRequestVersion activated tracer without Open")
	}
}

func TestOpenIsExclusive(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "a.trace.jsonl")
	closeTrace, err := Open(first)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if Enabled() {
			_ = closeTrace()
		}
	})
	if !Enabled() {
		t.Fatal("tracer not active after Open")
	}
	if _, err := Open(first); err == nil {
		t.Fatal("second Open of same sink succeeded")
	}
	if _, err := Open(filepath.Join(dir, "b.trace.jsonl")); err == nil {
		t.Fatal("concurrent Open of second sink succeeded")
	}
	if err := closeTrace(); err != nil {
		t.Fatal(err)
	}
}

func TestRequestStageMetadata(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "trace.jsonl")
	closeTrace, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if Enabled() {
			_ = closeTrace()
		}
	})
	id := json.RawMessage(`42`)
	endRequest := Request("textDocument/completion", &id, 7)
	endChild := Stage("completion-context")
	endChild()
	endRequest()
	if err := closeTrace(); err != nil {
		t.Fatal(err)
	}
	rows := readRows(t, path)
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want child + request", len(rows))
	}
	child, request := rows[0], rows[1]
	if child.Stage != "completion-context" || request.Stage != "request" {
		t.Fatalf("unexpected stage order %+v", rows)
	}
	for _, row := range rows {
		if row.Sequence != 1 || row.Method != "textDocument/completion" || string(row.ID) != "42" || row.Version != 7 {
			t.Fatalf("row lost request attribution: %+v", row)
		}
		if row.Duration < 0 {
			t.Fatalf("negative duration: %+v", row)
		}
	}
	// Rows carry exactly the documented key set: no source, payloads or paths.
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		var keys map[string]json.RawMessage
		if err := json.Unmarshal([]byte(line), &keys); err != nil {
			t.Fatal(err)
		}
		for key := range keys {
			switch key {
			case "sequence", "method", "id", "version", "stage", "duration_ns":
			default:
				t.Fatalf("trace row carries unexpected key %q", key)
			}
		}
	}
}

func TestSetRequestVersionAfterStart(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "trace.jsonl")
	closeTrace, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if Enabled() {
			_ = closeTrace()
		}
	})
	endRequest := Request("textDocument/didChange", nil, 0)
	SetRequestVersion(9)
	endRequest()
	if err := closeTrace(); err != nil {
		t.Fatal(err)
	}
	rows := readRows(t, path)
	if len(rows) != 1 || rows[0].Version != 9 {
		t.Fatalf("version attribution lost: %+v", rows)
	}
	if rows[0].ID != nil {
		t.Fatalf("absent request ID should omit the key: %+v", rows[0])
	}
}

func TestInclusiveNesting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "trace.jsonl")
	closeTrace, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if Enabled() {
			_ = closeTrace()
		}
	})
	endOuter := Stage("snapshot")
	endInner := Stage("check")
	endInner()
	endOuter()
	if err := closeTrace(); err != nil {
		t.Fatal(err)
	}
	rows := readRows(t, path)
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want inner + outer", len(rows))
	}
	inner, outer := rows[0], rows[1]
	if inner.Stage != "check" || outer.Stage != "snapshot" {
		t.Fatalf("unexpected stage order %+v", rows)
	}
	if outer.Duration < inner.Duration {
		t.Fatalf("outer %dns excludes inner %dns on a monotonic clock", outer.Duration, inner.Duration)
	}
}

func TestCapFailureInvalidatesTrace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "trace.jsonl")
	closeTrace, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if Enabled() {
			_ = closeTrace()
		}
	})
	r := active.Load()
	if r == nil {
		t.Fatal("tracer not active after Open")
	}
	// Force the byte cap deterministically instead of writing 32 MiB.
	r.mu.Lock()
	r.bytes = Limit - 10
	r.mu.Unlock()
	end := Stage("snapshot")
	end()
	if err := closeTrace(); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("cap breach reported %v, want exceeds-cap error", err)
	}
	// Once failed, later stages stay silent rather than appending partial rows.
	r.mu.Lock()
	failed := r.err != nil
	r.mu.Unlock()
	if !failed {
		t.Fatal("recorder error not retained after cap breach")
	}
}

func TestWriteFailureInvalidatesTrace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "trace.jsonl")
	closeTrace, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if Enabled() {
			_ = closeTrace()
		}
	})
	r := active.Load()
	if r == nil {
		t.Fatal("tracer not active after Open")
	}
	if err := r.file.Close(); err != nil {
		t.Fatal(err)
	}
	end := Stage("snapshot")
	end()
	if err := closeTrace(); err == nil {
		t.Fatal("write to closed sink reported success")
	}
}

func TestCloseRetiresRecorder(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "trace.jsonl")
	closeTrace, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if Enabled() {
			_ = closeTrace()
		}
	})
	end := Stage("snapshot")
	end()
	if err := closeTrace(); err != nil {
		t.Fatal(err)
	}
	if Enabled() {
		t.Fatal("tracer still active after close")
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	endStale := Stage("snapshot")
	endStale()
	endRequest := Request("shutdown", nil, 0)
	endRequest()
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("stages after close appended rows")
	}
	// Retirement frees the global slot; the used path stays exclusive.
	reopen, err := Open(filepath.Join(dir, "next.trace.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if Enabled() {
			_ = reopen()
		}
	})
	if err := reopen(); err != nil {
		t.Fatal(err)
	}
}
