package driver

import (
	"bytes"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/source"
)

// AU-Q2-core CLI leg: advisory findings print to the diagnostic stream
// without failing the pipeline; exit codes stay 0 for warnings-only runs
// because callers continue after reporting.
func TestReportWarnings(t *testing.T) {
	var stderr bytes.Buffer
	reportWarnings(&stderr, []check.Warning{{Code: "CAN-CHECK-UNNECESSARY-LOCAL", File: "src/main.can", Line: 12, Column: 5, Span: source.Span{Start: 184, End: 212}, Message: "accidental alias total — consider inlining"}})
	out := stderr.String()
	if !strings.Contains(out, "src/main.can:12:5: CAN-CHECK-UNNECESSARY-LOCAL: accidental alias total — consider inlining") {
		t.Fatalf("warning not surfaced on CLI stream: %q", out)
	}
	var empty bytes.Buffer
	reportWarnings(&empty, nil)
	if empty.Len() != 0 {
		t.Fatalf("empty warnings printed %q", empty.String())
	}
}
