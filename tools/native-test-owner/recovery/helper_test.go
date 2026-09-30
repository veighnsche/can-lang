package recovery

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/tools/native-test-owner/journal"
	"github.com/veighnsche/can-lang/tools/native-test-owner/process"
)

const testDigest = "sha256:" + "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func testSelf() journal.Owner {
	return journal.Owner{PID: os.Getpid(), StartToken: "p11-test-owner"}
}

func testStranger() journal.Owner {
	return journal.Owner{PID: os.Getpid(), StartToken: "p11-stranger"}
}

func reserve(t *testing.T, j *journal.Journal, opID string, owner journal.Owner, path journal.PathIdentity) {
	t.Helper()
	if _, err := j.Reserve(opID, testDigest, owner, path, "", 0); err != nil {
		t.Fatalf("Reserve %q: %v", opID, err)
	}
}

// procOwner returns a process owner that aborts every retained child on
// cleanup and fails the test when a stray remains.
func procOwner(t *testing.T, j *journal.Journal, token string) *process.Owner {
	t.Helper()
	o, err := process.New(j, os.Getpid(), token)
	if err != nil {
		t.Fatalf("process.New: %v", err)
	}
	t.Cleanup(func() {
		for _, e := range o.Tree() {
			_ = o.Abort(e.Child)
		}
		if n := o.Live(); n != 0 {
			t.Errorf("Live() = %d, want 0 (stray processes)", n)
		}
	})
	return o
}

// spawnSleeper starts one exec-form sleeper: the child is sleep itself,
// with no subshell left behind to stray.
func spawnSleeper(t *testing.T, o *process.Owner, opID string, secs int) process.Identity {
	t.Helper()
	id, err := o.Spawn(opID, process.Spec{
		Executable: "/bin/sh",
		Args:       []string{"-c", fmt.Sprintf("exec /bin/sleep %d", secs), "sh"},
		Env:        map[string]string{"P11_SLEEPER": opID},
	}, nil)
	if err != nil {
		t.Fatalf("Spawn %q: %v", opID, err)
	}
	return id
}

// deadPID returns a PID that is provably not live right now.
func deadPID(t *testing.T, o *process.Owner, base string) int {
	t.Helper()
	for i := 0; i < 8; i++ {
		opID := fmt.Sprintf("%s-dead-%d", base, i)
		id, err := o.Spawn(opID, process.Spec{
			Executable: "/bin/sh",
			Args:       []string{"-c", "exit 0", "sh"},
			Env:        map[string]string{},
		}, nil)
		if err != nil {
			t.Fatalf("Spawn %q: %v", opID, err)
		}
		if err := o.Abort(id); err != nil {
			t.Fatalf("Abort %q: %v", opID, err)
		}
		if !ProbeAlive(id.PID) {
			return id.PID
		}
	}
	t.Fatalf("cannot mint a dead PID")
	return -1
}

func writeTempFile(t *testing.T, name, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return p
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile %q: %v", path, err)
	}
	return string(raw)
}

// pendingStates snapshots opID -> state for no-mutation assertions.
func pendingStates(j *journal.Journal) map[string]journal.State {
	out := map[string]journal.State{}
	for _, r := range j.Pending() {
		out[r.OperationID] = r.State
	}
	return out
}

func sameStates(a, b map[string]journal.State) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

// assertNoStraySleepers sweeps ps for our unique sleeper markers and fails
// when any test sleeper outlived the run.
func assertNoStraySleepers(t *testing.T, markers ...string) {
	t.Helper()
	out, err := exec.Command("ps", "-ax", "-o", "pid=,args=").Output()
	if err != nil {
		t.Fatalf("ps sweep: %v", err)
	}
	var strays []string
	for _, line := range strings.Split(string(out), "\n") {
		for _, m := range markers {
			if strings.Contains(line, m) {
				strays = append(strays, strings.TrimSpace(line))
			}
		}
	}
	if len(strays) > 0 {
		t.Fatalf("stray sleepers after run:\n%s", strings.Join(strays, "\n"))
	}
}
