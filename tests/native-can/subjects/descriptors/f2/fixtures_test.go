package f2

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/tools/native-test-owner/process"
)

// The generation lease path is fixed: one name per generation.
func TestLeasePathFixed(t *testing.T) {
	if got := LeasePath("/tmp/gen"); got != filepath.Join("/tmp/gen", "generation.lease") {
		t.Fatalf("LeasePath = %q, want the fixed lease name", got)
	}
	if LeaseFileName == "" {
		t.Fatal("LeaseFileName is empty")
	}
}

// Readiness scripts render deterministically: fd-3 drain, the exact
// octal-escaped ack bytes, then the hold or exit tail.
func TestReadyScriptsRender(t *testing.T) {
	frame := []byte{0x4e, 0x54, 0x01, 0x03, 'A', 'C', 'K', 0x00, '%', 'x'}
	octal := process.OctalEscape(frame)
	hold := ReadyHoldScript("/bin/cat", "/bin/sleep", frame, 30)
	for _, want := range []string{"/bin/cat <&3 >/dev/null", "printf '%b' '" + octal + "'", "exec /bin/sleep 30"} {
		if !strings.Contains(hold, want) {
			t.Fatalf("hold script %q lacks %q", hold, want)
		}
	}
	exit := ReadyExitScript("/bin/cat", frame, 3)
	for _, want := range []string{"/bin/cat <&3 >/dev/null", "printf '%b' '" + octal + "'", "exit 3"} {
		if !strings.Contains(exit, want) {
			t.Fatalf("exit script %q lacks %q", exit, want)
		}
	}
	if ReadyHoldScript("/bin/cat", "/bin/sleep", frame, 30) != hold {
		t.Fatal("hold script not deterministic")
	}
	if ReadyExitScript("/bin/cat", frame, 3) != exit {
		t.Fatal("exit script not deterministic")
	}
	if strings.Contains(hold, "%!") || strings.Contains(exit, "%!") {
		t.Fatalf("script renders a format error: hold=%q exit=%q", hold, exit)
	}
}

// The witness records a missing path as an errored, unheld fact.
func TestWitnessMissingPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "absent.lease")
	w := WitnessLease(path)
	if w.Path != path {
		t.Fatalf("Witness.Path = %q, want %q", w.Path, path)
	}
	if w.Held || w.Err == nil {
		t.Fatalf("Witness = %+v, want held=false with probe error", w)
	}
}

// Pruning a missing path removes nothing and witnesses nothing.
func TestPruneMissingPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "absent.lease")
	out := PruneLeaseFile(path)
	if out.Path != path {
		t.Fatalf("Prune.Path = %q, want %q", out.Path, path)
	}
	if out.Removed || out.RemoveErr == nil {
		t.Fatalf("Prune = %+v, want removed=false with unlink error", out)
	}
	if out.After.Held || out.After.Err == nil {
		t.Fatalf("Prune.After = %+v, want held=false with probe error", out.After)
	}
}

// Pruning an unlocked file removes it and leaves a reusable path.
func TestPruneUnlockedFile(t *testing.T) {
	path := LeasePath(t.TempDir())
	if err := os.WriteFile(path, []byte("gen"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	out := PruneLeaseFile(path)
	if !out.Removed || out.RemoveErr != nil {
		t.Fatalf("Prune = %+v, want removed=true without error", out)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("Stat after prune: err=%v, want path gone", err)
	}
	if out.After.Held || out.After.Err == nil {
		t.Fatalf("Prune.After = %+v, want held=false with probe error", out.After)
	}
	// The pruned path accepts a fresh generation file.
	if err := os.WriteFile(path, []byte("next"), 0o600); err != nil {
		t.Fatalf("re-create after prune: %v", err)
	}
}
