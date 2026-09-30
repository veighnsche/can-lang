package recovery

import (
	"errors"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/tools/native-test-owner/journal"
	"github.com/veighnsche/can-lang/tools/native-test-owner/process"
)

// awaitExec polls ps until pid's command line is the exec'd sleeper,
// proving the shell already ran everything before the exec (including any
// trap arming). A prefix match is required: the pre-exec shell command
// line contains the -c script text, which merely mentions sleep. The wait
// is bounded so a stuck spawn fails loudly instead of hanging the suite.
func awaitExec(t *testing.T, pid int, wantPrefix string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "args=").Output()
		if err == nil && strings.HasPrefix(strings.TrimSpace(string(out)), wantPrefix) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("pid %d never exec'd %q", pid, wantPrefix)
}

func TestDrainReapsOwnedSleepers(t *testing.T) {
	o := procOwner(t, journal.OpenMemory(), "p11-drain-proc")
	a := spawnSleeper(t, o, "drain-a", 41)
	b := spawnSleeper(t, o, "drain-b", 41)
	var g Gate
	if err := g.Admit(); err != nil {
		t.Fatalf("Admit before Drain: %v", err)
	}

	rep, err := Drain(&g, o, DrainConfig{Grace: 5 * time.Second})
	if err != nil {
		t.Fatalf("Drain: %v", err)
	}
	if !rep.Complete || rep.Reason != "" {
		t.Fatalf("report = (complete=%v reason=%q), want (complete=true reason=%q)",
			rep.Complete, rep.Reason, "")
	}
	if len(rep.Outcomes) != 2 {
		t.Fatalf("outcomes = %d, want 2", len(rep.Outcomes))
	}
	seen := map[int]DrainOutcome{}
	for _, out := range rep.Outcomes {
		seen[out.PID] = out
	}
	for _, id := range []process.Identity{a, b} {
		out, ok := seen[id.PID]
		if !ok {
			t.Fatalf("no outcome for pid %d", id.PID)
		}
		if !out.Signaled || !out.Reaped || out.Reason != ReasonDrained {
			t.Fatalf("outcome for pid %d = %+v, want signaled+reaped drained", id.PID, out)
		}
		if ProbeAlive(id.PID) {
			t.Fatalf("pid %d still live after drain", id.PID)
		}
	}
	if n := o.Live(); n != 0 {
		t.Fatalf("Live() = %d, want 0", n)
	}
	if !g.Closed() {
		t.Fatalf("gate open after Drain")
	}
	if err := g.Admit(); !errors.Is(err, ErrAdmissionClosed) {
		t.Fatalf("Admit after Drain: got %v, want ErrAdmissionClosed", err)
	}
	assertNoStraySleepers(t, "sleep 41")
}

func TestDrainBoundedCap(t *testing.T) {
	o := procOwner(t, journal.OpenMemory(), "p11-cap-proc")
	spawnSleeper(t, o, "cap-a", 42)
	spawnSleeper(t, o, "cap-b", 42)
	var g Gate

	rep, err := Drain(&g, o, DrainConfig{Grace: 5 * time.Second, MaxChildren: 1})
	if err != nil {
		t.Fatalf("Drain: %v", err)
	}
	if rep.Complete || rep.Reason != ReasonDrainCap {
		t.Fatalf("report = (complete=%v reason=%q), want (complete=false reason=%q)",
			rep.Complete, rep.Reason, ReasonDrainCap)
	}
	if len(rep.Outcomes) != 1 || !rep.Outcomes[0].Reaped {
		t.Fatalf("outcomes = %+v, want one reaped child", rep.Outcomes)
	}
	if n := o.Live(); n != 1 {
		t.Fatalf("Live() = %d, want 1 (capped remainder untouched)", n)
	}
	// The test owns the capped remainder: abort it explicitly so no
	// stray survives the run.
	for _, e := range o.Tree() {
		if err := o.Abort(e.Child); err != nil {
			t.Fatalf("Abort remainder: %v", err)
		}
	}
	if n := o.Live(); n != 0 {
		t.Fatalf("Live() = %d, want 0", n)
	}
	assertNoStraySleepers(t, "sleep 42")
}

func TestDrainEscalatesIgnoredTerm(t *testing.T) {
	o := procOwner(t, journal.OpenMemory(), "p11-escalate-proc")
	id, err := o.Spawn("escalate-a", process.Spec{
		Executable: "/bin/sh",
		// An ignored disposition survives exec, so sleep inherits
		// SIG_IGN for TERM and outlives the grace wait.
		Args: []string{"-c", "trap '' TERM; exec /bin/sleep 44", "sh"},
		Env:  map[string]string{"P11_SLEEPER": "escalate-a"},
	}, nil)
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	// Wait for the exec: draining before the shell arms its trap would
	// kill a child that must survive SIGTERM.
	awaitExec(t, id.PID, "/bin/sleep 44")
	var g Gate

	rep, err := Drain(&g, o, DrainConfig{Grace: time.Second})
	if err != nil {
		t.Fatalf("Drain: %v", err)
	}
	if !rep.Complete || rep.Reason != "" {
		t.Fatalf("report = (complete=%v reason=%q), want complete", rep.Complete, rep.Reason)
	}
	if len(rep.Outcomes) != 1 {
		t.Fatalf("outcomes = %d, want 1", len(rep.Outcomes))
	}
	out := rep.Outcomes[0]
	if out.PID != id.PID || !out.Signaled || !out.Reaped || out.Reason != ReasonDrainEscalated {
		t.Fatalf("outcome = %+v, want signaled+reaped escalated", out)
	}
	if n := o.Live(); n != 0 {
		t.Fatalf("Live() = %d, want 0", n)
	}
	assertNoStraySleepers(t, "sleep 44")
}

func TestDrainEmpty(t *testing.T) {
	o := procOwner(t, journal.OpenMemory(), "p11-empty-proc")
	var g Gate
	rep, err := Drain(&g, o, DrainConfig{})
	if err != nil {
		t.Fatalf("Drain: %v", err)
	}
	if !rep.Complete || rep.Reason != "" || len(rep.Outcomes) != 0 {
		t.Fatalf("report = %+v, want complete and empty", rep)
	}
	if !g.Closed() {
		t.Fatalf("gate open after Drain")
	}
}

func TestDrainInvalid(t *testing.T) {
	o := procOwner(t, journal.OpenMemory(), "p11-invalid-proc")
	var g Gate
	if _, err := Drain(nil, o, DrainConfig{}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("nil gate: got %v, want ErrInvalid", err)
	}
	if _, err := Drain(&g, nil, DrainConfig{}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("nil owner: got %v, want ErrInvalid", err)
	}
}
