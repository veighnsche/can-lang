package process

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/tools/native-test-owner/journal"
)

// requireDarwin skips tests whose well-behaved child hashes fd 3 with
// /usr/bin/shasum. The P10 control record is OS-specific (darwin); the
// package itself builds anywhere flock exists.
func requireDarwin(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "darwin" {
		t.Skip("darwin-only shasum control (P10 OS-specific record)")
	}
}

func testSig() os.Signal { return syscall.SIGTERM }

func isWrongOwner(err error) bool { return errors.Is(err, ErrWrongOwner) }

const testShell = "/bin/sh"

func testOwner(t *testing.T) *Owner {
	t.Helper()
	o, err := New(journal.OpenMemory(), os.Getpid(), "test-owner-token")
	if err != nil {
		t.Fatalf("New: %v", err)
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

func testLease(t *testing.T, name string) *Lease {
	t.Helper()
	l, err := AcquireLease(filepath.Join(t.TempDir(), "lease-"+name))
	if err != nil {
		t.Fatalf("AcquireLease: %v", err)
	}
	t.Cleanup(func() { _ = l.Close() })
	return l
}

// ackFrameFor precomputes the exact status bytes for an operation whose
// environment snapshot is known before spawn. Scripts embedding it
// stand in for children that read fd 3 and bound their ack to the
// offered bytes.
func ackFrameFor(t *testing.T, opID string, env map[string]string) []byte {
	t.Helper()
	snap, err := snapshotEnv(env)
	if err != nil {
		t.Fatalf("snapshotEnv: %v", err)
	}
	frame, err := encodeStatusFrame(statusPayload(opID, envDigest(snap)))
	if err != nil {
		t.Fatalf("encodeStatusFrame: %v", err)
	}
	return frame
}

// wellBehavedScript builds a child that genuinely reads fd 3, hashes
// the bytes with shasum, and emits a content-bound ack frame. The
// header length is fixed because the digest is always 64 hex chars.
func wellBehavedScript(opID string, env map[string]string) string {
	snap, _ := snapshotEnv(env)
	_ = snap
	payloadLen := 4 + len(opID) + 1 + 64
	var hdr strings.Builder
	for _, b := range []byte{0x4E, 0x54, 0x01, 0x03, 0, 0, 0, 0,
		byte(payloadLen >> 24), byte(payloadLen >> 16), byte(payloadLen >> 8), byte(payloadLen)} {
		fmt.Fprintf(&hdr, "\\%03o", b)
	}
	return fmt.Sprintf(`d=$(/bin/cat <&3 | /usr/bin/shasum -a 256 | /usr/bin/cut -d' ' -f1); printf '%sACK:%s:%%s' "$d"`, hdr.String(), opID)
}

func shArgs(script string, extras ...string) []string {
	return append([]string{"-c", script, "sh"}, extras...)
}

func testEnv() map[string]string {
	return map[string]string{"ALPHA": "1", "BETA": "two"}
}

func mustAbort(o *Owner, id Identity) {
	_ = o.Abort(id)
}

func TestHappyCleanRelease(t *testing.T) {
	requireDarwin(t)
	o := testOwner(t)
	l := testLease(t, "happy")
	opID := "happy1"
	env := testEnv()
	id, err := o.Spawn(opID, Spec{Executable: testShell, Args: shArgs(wellBehavedScript(opID, env)), Env: env, WithLease: true}, l)
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	if err := o.CollectStatus(id, 5*time.Second, 5*time.Second); err != nil {
		t.Fatalf("CollectStatus: %v", err)
	}
	// The lease is independent of the channel lifecycle: EOF on the
	// status channel and a finished env writer do not release fd 4.
	if held, err := ProbeLease(l.Path()); err != nil || !held {
		t.Fatalf("ProbeLease before release: held=%v err=%v, want held=true", held, err)
	}
	exit, err := o.Wait(id, 10*time.Second)
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if exit.Code != 0 || exit.Signaled {
		t.Fatalf("exit = %+v, want code 0", exit)
	}
	rep, err := o.Release(id)
	if err != nil {
		t.Fatalf("Release: %v", err)
	}
	if !rep.Clean {
		t.Fatalf("Clean=false reason=%q facts=%+v lease=%+v", rep.Reason, rep.Facts, rep.Lease)
	}
	f := rep.Facts
	if !f.Offered || !f.WriterDone || f.WriterErr != "" || !f.Accepted || f.Malformed || !f.EOF || !f.Reaped {
		t.Fatalf("facts = %+v, want all positive", f)
	}
	if !rep.Lease.Inherited || !rep.Lease.Held || !rep.Lease.Released || rep.Orphan {
		t.Fatalf("lease = %+v, want inherited+held+released, no orphan", rep.Lease)
	}
	// Independent witness: a fresh probe after release observes loss of
	// the lock, and the child handle is gone.
	if held, err := ProbeLease(l.Path()); err != nil || held {
		t.Fatalf("ProbeLease after release: held=%v err=%v, want false", held, err)
	}
	if _, err := o.Facts(id); err == nil {
		t.Fatalf("Facts after clean release: want error, got nil")
	}
}

func TestExactExecutableEnvironmentFDs(t *testing.T) {
	o := testOwner(t)
	l := testLease(t, "exact")
	dir := t.TempDir()
	envOut := filepath.Join(dir, "env.out")
	fdsOut := filepath.Join(dir, "fds.out")
	opID := "exact1"
	env := map[string]string{"ZED": "last", "ALPHA": "first", "EMPTY": ""}
	script := `/bin/cat <&3 > "$1"; /bin/ls /dev/fd > "$2"`
	id, err := o.Spawn(opID, Spec{Executable: testShell, Args: shArgs(script, envOut, fdsOut), Env: env, WithLease: true}, l)
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	defer mustAbort(o, id)
	if _, err := o.Wait(id, 10*time.Second); err != nil {
		t.Fatalf("Wait: %v", err)
	}
	wantExe, _ := filepath.EvalSymlinks(testShell)
	if got, _ := o.Executable(id); got != wantExe {
		t.Fatalf("Executable = %q, want %q", got, wantExe)
	}
	// Exact environment bytes: sorted-key JSON, no trailing newline.
	wantSnap := []byte(`{"ALPHA":"first","EMPTY":"","ZED":"last"}`)
	if got, _ := o.Snapshot(id); string(got) != string(wantSnap) {
		t.Fatalf("Snapshot = %q, want %q", got, wantSnap)
	}
	if got, err := os.ReadFile(envOut); err != nil || string(got) != string(wantSnap) {
		t.Fatalf("child observed %q err=%v, want %q", got, err, wantSnap)
	}
	fdMap, err := o.FDMap(id)
	if err != nil {
		t.Fatalf("FDMap: %v", err)
	}
	if fdMap[FDEnv] != "env" || fdMap[FDLease] != "lease" {
		t.Fatalf("FDMap = %v, want fd 3=env fd 4=lease distinct", fdMap)
	}
	raw, err := os.ReadFile(fdsOut)
	if err != nil {
		t.Fatalf("read fds: %v", err)
	}
	seen := map[string]bool{}
	for _, line := range strings.Split(string(raw), "\n") {
		seen[strings.TrimSpace(line)] = true
	}
	for _, want := range []string{"0", "1", "2", "3", "4"} {
		if !seen[want] {
			t.Fatalf("/dev/fd listing = %q, want entry %s", raw, want)
		}
	}
	// This child never acked, so release cannot be clean even though
	// every byte was exact.
	_ = o.CollectStatus(id, time.Second, time.Second)
	rep, err := o.Release(id)
	if err != nil {
		t.Fatalf("Release: %v", err)
	}
	if rep.Clean {
		t.Fatalf("Clean=true without ack, facts=%+v", rep.Facts)
	}
}

// TestControlsCannotReleaseClean runs every acceptance control: none of
// these scenarios may produce a clean release, and each must fail on
// its own operative cause.
func TestControlsCannotReleaseClean(t *testing.T) {
	t.Run("fd-3-non-reader", func(t *testing.T) {
		o := testOwner(t)
		l := testLease(t, "nonreader")
		// Exits at once: reads neither fd 3 nor knows the ack.
		id, err := o.Spawn("nonreader1", Spec{Executable: testShell, Args: shArgs(`exit 0`), Env: testEnv(), WithLease: true}, l)
		if err != nil {
			t.Fatalf("Spawn: %v", err)
		}
		defer mustAbort(o, id)
		if err := o.CollectStatus(id, 5*time.Second, 5*time.Second); err == nil {
			t.Fatalf("CollectStatus: want error, got nil")
		}
		if _, err := o.Wait(id, 10*time.Second); err != nil {
			t.Fatalf("Wait: %v", err)
		}
		rep, err := o.Release(id)
		if err != nil {
			t.Fatalf("Release: %v", err)
		}
		if rep.Clean {
			t.Fatalf("Clean=true for fd-3 non-reader")
		}
		// Two independent failures race here by design: the writer may
		// or may not deliver before the instant-exiting child closes
		// fd 3, so either refusal cause is correct. The ack is missing
		// in both cases.
		if rep.Reason != "status ack not accepted" && rep.Reason != "environment not offered" {
			t.Fatalf("Reason=%q, want ack or offer refusal", rep.Reason)
		}
		if rep.Facts.Accepted || !rep.Facts.EOF {
			t.Fatalf("facts=%+v, want accepted=false EOF=true", rep.Facts)
		}
		if !rep.Facts.WriterDone {
			t.Fatalf("facts=%+v, want writer-exit recorded", rep.Facts)
		}
		// The lease facts stay healthy and distinct: held, then
		// released. Only the missing acceptance blocks release.
		if !rep.Lease.Held || !rep.Lease.Released {
			t.Fatalf("lease=%+v, want held+released", rep.Lease)
		}
	})

	t.Run("stale-ack", func(t *testing.T) {
		o := testOwner(t)
		l := testLease(t, "staleack")
		opID := "staleack1"
		// A non-reader guessing a foreign ack: bound to another op.
		foreign := ackFrameFor(t, "someone-else", testEnv())
		script := fmt.Sprintf(`printf '%%b' '%s'`, OctalEscape(foreign))
		id, err := o.Spawn(opID, Spec{Executable: testShell, Args: shArgs(script), Env: testEnv(), WithLease: true}, l)
		if err != nil {
			t.Fatalf("Spawn: %v", err)
		}
		defer mustAbort(o, id)
		if err := o.CollectStatus(id, 5*time.Second, 5*time.Second); err == nil {
			t.Fatalf("CollectStatus: want error, got nil")
		}
		if _, err := o.Wait(id, 10*time.Second); err != nil {
			t.Fatalf("Wait: %v", err)
		}
		rep, err := o.Release(id)
		if err != nil {
			t.Fatalf("Release: %v", err)
		}
		if rep.Clean || !rep.Facts.Malformed || rep.Facts.Accepted {
			t.Fatalf("Clean=%v facts=%+v, want refused+malformed", rep.Clean, rep.Facts)
		}
	})

	t.Run("delayed-eof", func(t *testing.T) {
		o := testOwner(t)
		l := testLease(t, "delayedeof")
		opID := "delayedeof1"
		// Acks correctly, then holds the status channel open.
		script := fmt.Sprintf(`printf '%%b' '%s'; exec /bin/sleep 30`, OctalEscape(ackFrameFor(t, opID, testEnv())))
		id, err := o.Spawn(opID, Spec{Executable: testShell, Args: shArgs(script), Env: testEnv(), WithLease: true}, l)
		if err != nil {
			t.Fatalf("Spawn: %v", err)
		}
		defer mustAbort(o, id)
		if err := o.CollectStatus(id, 5*time.Second, 300*time.Millisecond); err == nil {
			t.Fatalf("CollectStatus: want EOF timeout, got nil")
		}
		f, _ := o.Facts(id)
		if !f.Accepted || f.EOF {
			t.Fatalf("facts=%+v, want accepted=true EOF=false", f)
		}
		rep, err := o.Release(id)
		if err != nil {
			t.Fatalf("Release: %v", err)
		}
		if rep.Clean {
			t.Fatalf("Clean=true with delayed EOF")
		}
		if rep.Reason != "status EOF not observed" {
			t.Fatalf("Reason=%q, want EOF refusal", rep.Reason)
		}
		// Transient: the lease share stays open and the journal
		// stays live while the child runs.
		if held, _ := ProbeLease(l.Path()); !held {
			t.Fatalf("lease lost while child runs")
		}
		// Cleanup: a killed child can never become clean either.
		if err := o.Kill(id); err != nil {
			t.Fatalf("Kill: %v", err)
		}
		exit, err := o.Wait(id, 10*time.Second)
		if err != nil {
			t.Fatalf("Wait: %v", err)
		}
		if !exit.Signaled {
			t.Fatalf("exit=%+v, want signaled", exit)
		}
	})

	t.Run("malformed-frame", func(t *testing.T) {
		o := testOwner(t)
		l := testLease(t, "malformed")
		id, err := o.Spawn("malformed1", Spec{Executable: testShell, Args: shArgs(`printf 'garbage-not-a-frame'; exit 0`), Env: testEnv(), WithLease: true}, l)
		if err != nil {
			t.Fatalf("Spawn: %v", err)
		}
		defer mustAbort(o, id)
		if err := o.CollectStatus(id, 5*time.Second, 5*time.Second); err == nil {
			t.Fatalf("CollectStatus: want error, got nil")
		}
		if _, err := o.Wait(id, 10*time.Second); err != nil {
			t.Fatalf("Wait: %v", err)
		}
		rep, err := o.Release(id)
		if err != nil {
			t.Fatalf("Release: %v", err)
		}
		if rep.Clean || !rep.Facts.Malformed || rep.Facts.Accepted {
			t.Fatalf("Clean=%v facts=%+v, want refused+malformed", rep.Clean, rep.Facts)
		}
	})

	t.Run("malformed-trailer", func(t *testing.T) {
		o := testOwner(t)
		l := testLease(t, "trailer")
		opID := "trailer1"
		// Acks, then emits a second frame where EOF belongs.
		esc := OctalEscape(ackFrameFor(t, opID, testEnv()))
		script := fmt.Sprintf(`printf '%%b' '%s'; printf '%%b' '%s'`, esc, esc)
		id, err := o.Spawn(opID, Spec{Executable: testShell, Args: shArgs(script), Env: testEnv(), WithLease: true}, l)
		if err != nil {
			t.Fatalf("Spawn: %v", err)
		}
		defer mustAbort(o, id)
		if err := o.CollectStatus(id, 5*time.Second, 5*time.Second); err == nil {
			t.Fatalf("CollectStatus: want error, got nil")
		}
		if _, err := o.Wait(id, 10*time.Second); err != nil {
			t.Fatalf("Wait: %v", err)
		}
		rep, err := o.Release(id)
		if err != nil {
			t.Fatalf("Release: %v", err)
		}
		if rep.Clean || !rep.Facts.Malformed {
			t.Fatalf("Clean=%v facts=%+v, want refused+malformed", rep.Clean, rep.Facts)
		}
	})

	t.Run("orphan-detached-child", func(t *testing.T) {
		o := testOwner(t)
		l := testLease(t, "orphan")
		dir := t.TempDir()
		pidFile := filepath.Join(dir, "grandchild.pid")
		opID := "orphan1"
		// Well-behaved except for orphaning a grandchild that
		// inherits fd 4 (stdio redirected away so EOF is clean).
		// Draining fd 3 first makes the offer deterministic.
		script := fmt.Sprintf(`/bin/cat <&3 >/dev/null; printf '%%b' '%s'; /bin/sleep 30 </dev/null >/dev/null 2>&1 & echo $! > "$1"`,
			OctalEscape(ackFrameFor(t, opID, testEnv())))
		id, err := o.Spawn(opID, Spec{Executable: testShell, Args: shArgs(script, pidFile), Env: testEnv(), WithLease: true, Detached: true}, l)
		if err != nil {
			t.Fatalf("Spawn: %v", err)
		}
		defer mustAbort(o, id)
		if err := o.CollectStatus(id, 5*time.Second, 5*time.Second); err != nil {
			t.Fatalf("CollectStatus: %v", err)
		}
		if _, err := o.Wait(id, 10*time.Second); err != nil {
			t.Fatalf("Wait: %v", err)
		}
		rep, err := o.Release(id)
		if err != nil {
			t.Fatalf("Release: %v", err)
		}
		if rep.Clean || !rep.Orphan {
			t.Fatalf("Clean=%v Orphan=%v rep=%+v, want refused+orphan", rep.Clean, rep.Orphan, rep)
		}
		if rep.Reason != "orphaned descendant still holds the lease" {
			t.Fatalf("Reason=%q", rep.Reason)
		}
		if !rep.Facts.Accepted || !rep.Facts.EOF {
			t.Fatalf("facts=%+v, want accepted+EOF (orphan is the sole cause)", rep.Facts)
		}
		// Independent witness confirms the hold from a fresh open.
		if held, err := ProbeLease(l.Path()); err != nil || !held {
			t.Fatalf("ProbeLease: held=%v err=%v, want held", held, err)
		}
		// Reap the stray grandchild, then the witness observes the
		// release and a fresh Release decides clean.
		raw, err := os.ReadFile(pidFile)
		if err != nil {
			t.Fatalf("read grandchild pid: %v", err)
		}
		var gpid int
		if _, err := fmt.Sscanf(strings.TrimSpace(string(raw)), "%d", &gpid); err != nil || gpid <= 0 {
			t.Fatalf("parse grandchild pid %q: %v", raw, err)
		}
		if gp, err := os.FindProcess(gpid); err != nil {
			t.Fatalf("FindProcess: %v", err)
		} else if err := gp.Kill(); err != nil {
			t.Fatalf("kill grandchild: %v", err)
		}
		deadline := time.Now().Add(10 * time.Second)
		for {
			held, err := ProbeLease(l.Path())
			if err != nil {
				t.Fatalf("ProbeLease: %v", err)
			}
			if !held {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("lease still held 10s after grandchild kill")
			}
			time.Sleep(50 * time.Millisecond)
		}
		rep2, err := o.Release(id)
		if err != nil {
			t.Fatalf("Release#2: %v", err)
		}
		if !rep2.Clean {
			t.Fatalf("Clean#2=false reason=%q rep=%+v", rep2.Reason, rep2)
		}
	})

	t.Run("pid-reuse", func(t *testing.T) {
		o := testOwner(t)
		l := testLease(t, "pidreuse")
		idA, err := o.Spawn("pidreuseA", Spec{Executable: testShell, Args: shArgs(`exit 0`), Env: testEnv(), WithLease: true}, l)
		if err != nil {
			t.Fatalf("Spawn A: %v", err)
		}
		if _, err := o.Wait(idA, 10*time.Second); err != nil {
			t.Fatalf("Wait A: %v", err)
		}
		l2 := testLease(t, "pidreuse2")
		idB, err := o.Spawn("pidreuseB", Spec{Executable: testShell, Args: shArgs(`exec /bin/sleep 30`), Env: testEnv(), WithLease: true}, l2)
		if err != nil {
			t.Fatalf("Spawn B: %v", err)
		}
		defer mustAbort(o, idB)
		defer mustAbort(o, idA)
		if idA.StartToken == idB.StartToken {
			t.Fatalf("start tokens collide")
		}
		// A live PID with a stale token is never authority.
		forged := Identity{PID: idB.PID, StartToken: idA.StartToken}
		for name, op := range map[string]func() error{
			"Facts":   func() error { _, err := o.Facts(forged); return err },
			"Wait":    func() error { _, err := o.Wait(forged, time.Second); return err },
			"Signal":  func() error { return o.Signal(forged, testSig()) },
			"Release": func() error { _, err := o.Release(forged); return err },
		} {
			if err := op(); err == nil {
				t.Fatalf("%s(forged identity): want error, got nil", name)
			}
		}
		if _, err := o.Release(forged); !isWrongOwner(err) {
			t.Fatalf("Release(forged) err=%v, want ErrWrongOwner", err)
		}
		// Aborting A leaves B untouched: distinct handles.
		if err := o.Abort(idA); err != nil {
			t.Fatalf("Abort A: %v", err)
		}
		if _, err := o.Facts(idA); err == nil {
			t.Fatalf("Facts(A) after abort: want error, got nil")
		}
		if _, err := o.Facts(idB); err != nil {
			t.Fatalf("Facts(B): %v", err)
		}
		rep, err := o.Release(idA)
		if err == nil || rep.Clean {
			t.Fatalf("Release(A) after abort: err=%v clean=%v, want refused", err, rep.Clean)
		}
	})

	t.Run("missing-lease", func(t *testing.T) {
		o := testOwner(t)
		opID := "missinglease1"
		// Well-behaved on every channel fact, but fd 4 was never
		// inherited: protection was never established. Draining fd 3
		// first makes the offer deterministic: the writer must have
		// finished before the drain observes EOF.
		script := fmt.Sprintf(`/bin/cat <&3 >/dev/null; printf '%%b' '%s'`, OctalEscape(ackFrameFor(t, opID, testEnv())))
		id, err := o.Spawn(opID, Spec{Executable: testShell, Args: shArgs(script), Env: testEnv()}, nil)
		if err != nil {
			t.Fatalf("Spawn: %v", err)
		}
		defer mustAbort(o, id)
		if err := o.CollectStatus(id, 5*time.Second, 5*time.Second); err != nil {
			t.Fatalf("CollectStatus: %v", err)
		}
		if _, err := o.Wait(id, 10*time.Second); err != nil {
			t.Fatalf("Wait: %v", err)
		}
		if m, _ := o.FDMap(id); len(m) != 4 {
			t.Fatalf("FDMap=%v, want no fd 4", m)
		}
		rep, err := o.Release(id)
		if err != nil {
			t.Fatalf("Release: %v", err)
		}
		if rep.Clean || rep.Reason != "lease never inherited" {
			t.Fatalf("Clean=%v Reason=%q, want lease refusal", rep.Clean, rep.Reason)
		}
	})

	t.Run("early-lease-close", func(t *testing.T) {
		o := testOwner(t)
		l := testLease(t, "earlyclose")
		opID := "earlyclose1"
		script := fmt.Sprintf(`printf '%%b' '%s'; exec /bin/sleep 30`, OctalEscape(ackFrameFor(t, opID, testEnv())))
		id, err := o.Spawn(opID, Spec{Executable: testShell, Args: shArgs(script), Env: testEnv(), WithLease: true}, l)
		if err != nil {
			t.Fatalf("Spawn: %v", err)
		}
		defer mustAbort(o, id)
		// Launcher exit: the owner's share closes while the child
		// still runs. The child's inherited share keeps the lock.
		if err := o.CloseLeaseEarly(id); err != nil {
			t.Fatalf("CloseLeaseEarly: %v", err)
		}
		if held, err := ProbeLease(l.Path()); err != nil || !held {
			t.Fatalf("ProbeLease after launcher exit: held=%v err=%v, want held", held, err)
		}
		if err := o.Kill(id); err != nil {
			t.Fatalf("Kill: %v", err)
		}
		if _, err := o.Wait(id, 10*time.Second); err != nil {
			t.Fatalf("Wait: %v", err)
		}
		_ = o.CollectStatus(id, time.Second, time.Second)
		rep, err := o.Release(id)
		if err != nil {
			t.Fatalf("Release: %v", err)
		}
		if rep.Clean {
			t.Fatalf("Clean=true after early close without witness")
		}
	})
}

// TestKernelLeaseAcrossLauncherExit is the bounded kernel-level
// control: the flock survives the launcher closing its share while a
// descendant holds fd 4, and the independent witness observes both the
// protection and the later release.
func TestKernelLeaseAcrossLauncherExit(t *testing.T) {
	o := testOwner(t)
	l := testLease(t, "kernel")
	id, err := o.Spawn("kernel1", Spec{Executable: testShell, Args: shArgs(`exec /bin/sleep 30`), Env: testEnv(), WithLease: true}, l)
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	defer mustAbort(o, id)
	// Launcher exit: drop the owner's share; the child still holds fd 4.
	if err := o.CloseLeaseEarly(id); err != nil {
		t.Fatalf("CloseLeaseEarly: %v", err)
	}
	if held, err := ProbeLease(l.Path()); err != nil || !held {
		t.Fatalf("witness during hold: held=%v err=%v, want held", held, err)
	}
	if err := o.Kill(id); err != nil {
		t.Fatalf("Kill: %v", err)
	}
	if _, err := o.Wait(id, 10*time.Second); err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if held, err := ProbeLease(l.Path()); err != nil || held {
		t.Fatalf("witness after holder death: held=%v err=%v, want released", held, err)
	}
	// Loss after a missing descriptor: a child spawned without fd 4
	// never establishes protection.
	id2, err := o.Spawn("kernel2", Spec{Executable: testShell, Args: shArgs(`exec /bin/sleep 5`), Env: testEnv()}, nil)
	if err != nil {
		t.Fatalf("Spawn without lease: %v", err)
	}
	defer mustAbort(o, id2)
	l3 := testLease(t, "kernel3")
	// The owner's own share is the only hold; closing it loses
	// protection immediately even with a live child elsewhere.
	if err := l3.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if held, err := ProbeLease(l3.Path()); err != nil || held {
		t.Fatalf("witness after early close: held=%v err=%v, want lost", held, err)
	}
	if err := o.Kill(id2); err != nil {
		t.Fatalf("Kill: %v", err)
	}
	if _, err := o.Wait(id2, 10*time.Second); err != nil {
		t.Fatalf("Wait: %v", err)
	}
}

func TestSignalWaitReap(t *testing.T) {
	o := testOwner(t)
	l := testLease(t, "signal")
	id, err := o.Spawn("signal1", Spec{Executable: testShell, Args: shArgs(`exec /bin/sleep 30`), Env: testEnv(), WithLease: true}, l)
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	defer mustAbort(o, id)
	if _, err := o.Wait(id, 200*time.Millisecond); !errors.Is(err, ErrTimeout) {
		t.Fatalf("Wait on running child: err=%v, want ErrTimeout", err)
	}
	if err := o.Signal(id, syscall.SIGTERM); err != nil {
		t.Fatalf("Signal: %v", err)
	}
	exit, err := o.Wait(id, 10*time.Second)
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if !exit.Signaled || exit.Signal != "terminated" {
		t.Fatalf("exit=%+v, want signaled/terminated", exit)
	}
	// Re-wait joins the recorded outcome; signalling a reaped child fails.
	if exit2, err := o.Wait(id, time.Second); err != nil || exit2 != exit {
		t.Fatalf("Wait#2 = %+v err=%v, want %+v", exit2, err, exit)
	}
	if err := o.Signal(id, syscall.SIGTERM); !errors.Is(err, ErrState) {
		t.Fatalf("Signal after reap: err=%v, want ErrState", err)
	}
	rep, err := o.Release(id)
	if err != nil {
		t.Fatalf("Release: %v", err)
	}
	if rep.Clean {
		t.Fatalf("Clean=true for signaled child")
	}
}

func TestTreeEdges(t *testing.T) {
	o := testOwner(t)
	l1 := testLease(t, "tree1")
	l2 := testLease(t, "tree2")
	idA, err := o.Spawn("treeA", Spec{Executable: testShell, Args: shArgs(`exit 0`), Env: testEnv(), WithLease: true}, l1)
	if err != nil {
		t.Fatalf("Spawn A: %v", err)
	}
	defer mustAbort(o, idA)
	idB, err := o.Spawn("treeB", Spec{Executable: testShell, Args: shArgs(`exec /bin/sleep 30`), Env: testEnv(), WithLease: true}, l2)
	if err != nil {
		t.Fatalf("Spawn B: %v", err)
	}
	defer mustAbort(o, idB)
	if _, err := o.Wait(idA, 10*time.Second); err != nil {
		t.Fatalf("Wait A: %v", err)
	}
	edges := o.Tree()
	if len(edges) != 2 {
		t.Fatalf("Tree() has %d edges, want 2", len(edges))
	}
	byPID := map[int]Edge{}
	for _, e := range edges {
		if e.ParentPID != os.Getpid() {
			t.Fatalf("edge %+v: parent is not the test process", e)
		}
		byPID[e.Child.PID] = e
	}
	if byPID[idA.PID].Running {
		t.Fatalf("edge A: want reaped (not running)")
	}
	if !byPID[idB.PID].Running {
		t.Fatalf("edge B: want running")
	}
	if byPID[idB.PID].Child.StartToken != idB.StartToken {
		t.Fatalf("edge B token mismatch")
	}
	if err := o.Kill(idB); err != nil {
		t.Fatalf("Kill B: %v", err)
	}
	if _, err := o.Wait(idB, 10*time.Second); err != nil {
		t.Fatalf("Wait B: %v", err)
	}
}

func TestSpawnValidation(t *testing.T) {
	o := testOwner(t)
	l := testLease(t, "valid")
	if _, err := New(nil, os.Getpid(), "tok"); err == nil {
		t.Fatalf("New(nil journal): want error")
	}
	if _, err := New(journal.OpenMemory(), 0, "tok"); err == nil {
		t.Fatalf("New(pid 0): want error")
	}
	cases := []struct {
		name  string
		opID  string
		spec  Spec
		lease *Lease
	}{
		{"bad-opid", "no spaces", Spec{Executable: testShell, Env: testEnv()}, nil},
		{"relative-exe", "valid1", Spec{Executable: "bin/sh", Env: testEnv()}, nil},
		{"missing-exe", "valid2", Spec{Executable: "/nonexistent/child", Env: testEnv()}, nil},
		{"lease-without-flag", "valid3", Spec{Executable: testShell, Env: testEnv()}, l},
		{"flag-without-lease", "valid4", Spec{Executable: testShell, Env: testEnv(), WithLease: true}, nil},
		{"bad-env-key", "valid5", Spec{Executable: testShell, Env: map[string]string{"A=B": "x"}}, nil},
		{"oversize-env", "valid6", Spec{Executable: testShell, Env: map[string]string{"BIG": strings.Repeat("x", MaxEnvBytes)}}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := o.Spawn(c.opID, c.spec, c.lease); err == nil {
				t.Fatalf("Spawn(%s): want error, got nil", c.name)
			}
		})
	}
	// Replaying an operation ID with different arguments conflicts.
	id, err := o.Spawn("replay1", Spec{Executable: testShell, Args: shArgs(`exit 0`), Env: testEnv()}, nil)
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	defer mustAbort(o, id)
	if _, err := o.Wait(id, 10*time.Second); err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if _, err := o.Spawn("replay1", Spec{Executable: testShell, Args: shArgs(`exit 1`), Env: testEnv()}, nil); err == nil {
		t.Fatalf("Spawn replay with different args: want conflict, got nil")
	}
	// Unknown identities fail on every operation.
	ghost := Identity{PID: 1 << 30, StartToken: "ghost"}
	if _, err := o.Facts(ghost); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Facts(ghost): err=%v, want ErrNotFound", err)
	}
	if _, err := o.Release(ghost); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Release(ghost): err=%v, want ErrNotFound", err)
	}
	if _, err := ProbeLease(filepath.Join(t.TempDir(), "absent")); err == nil {
		t.Fatalf("ProbeLease(absent): want error, got nil")
	}
	if _, err := AcquireLease(""); err == nil {
		t.Fatalf("AcquireLease(empty): want error, got nil")
	}
}

func TestJournalLifecycle(t *testing.T) {
	requireDarwin(t)
	j := journal.OpenMemory()
	o, err := New(j, os.Getpid(), "journal-token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		for _, e := range o.Tree() {
			_ = o.Abort(e.Child)
		}
	})
	l := testLease(t, "journal")
	// Clean release walks reserved -> ... -> closed.
	opClean := "journalclean"
	id, err := o.Spawn(opClean, Spec{Executable: testShell, Args: shArgs(wellBehavedScript(opClean, testEnv())), Env: testEnv(), WithLease: true}, l)
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	if err := o.CollectStatus(id, 5*time.Second, 5*time.Second); err != nil {
		t.Fatalf("CollectStatus: %v", err)
	}
	if _, err := o.Wait(id, 10*time.Second); err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if rep, err := o.Release(id); err != nil || !rep.Clean {
		t.Fatalf("Release: rep=%+v err=%v, want clean", rep, err)
	}
	if r, ok := j.Lookup(opClean); !ok || r.State != journal.StateClosed {
		t.Fatalf("journal state = %+v ok=%v, want closed", r, ok)
	}
	// Terminal refusal records failed.
	id2, err := o.Spawn("journalfail", Spec{Executable: testShell, Args: shArgs(`printf 'junk'`), Env: testEnv(), WithLease: true}, testLease(t, "journal2"))
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	defer mustAbort(o, id2)
	_ = o.CollectStatus(id2, 5*time.Second, 5*time.Second)
	if _, err := o.Wait(id2, 10*time.Second); err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if rep, err := o.Release(id2); err != nil || rep.Clean {
		t.Fatalf("Release: rep=%+v err=%v, want refused", rep, err)
	}
	if r, ok := j.Lookup("journalfail"); !ok || r.State != journal.StateFailed {
		t.Fatalf("journal state = %+v ok=%v, want failed", r, ok)
	}
	if len(j.Pending()) != 0 {
		t.Fatalf("Pending() has %d entries, want 0", len(j.Pending()))
	}
}

// TestConcurrentReleaseAbortClose races the three lease-closing paths.
// Release, Abort and CloseLeaseEarly may all close the owner's share
// at once; the lease serializes them. Run with -race: without the
// lease mutex this test reports a data race on the share.
func TestConcurrentReleaseAbortClose(t *testing.T) {
	for i := 0; i < 8; i++ {
		o := testOwner(t)
		l := testLease(t, "concurrent")
		opID := fmt.Sprintf("concurrent%d", i)
		id, err := o.Spawn(opID, Spec{Executable: testShell, Args: shArgs(`exec /bin/sleep 30`), Env: testEnv(), WithLease: true}, l)
		if err != nil {
			t.Fatalf("Spawn: %v", err)
		}
		if err := o.Kill(id); err != nil {
			t.Fatalf("Kill: %v", err)
		}
		if _, err := o.Wait(id, 10*time.Second); err != nil {
			t.Fatalf("Wait: %v", err)
		}
		var wg sync.WaitGroup
		for k := 0; k < 3; k++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _ = o.Release(id)
			}()
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = o.CloseLeaseEarly(id)
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = o.Abort(id)
		}()
		wg.Wait()
	}
}

// TestConcurrentLeaseCloseHammer documents the Lease contract directly:
// Close is safe for concurrent use. Without the lease mutex this test
// reports a data race on the share under -race.
func TestConcurrentLeaseCloseHammer(t *testing.T) {
	l := testLease(t, "hammer")
	var wg sync.WaitGroup
	for k := 0; k < 16; k++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				_ = l.Close()
				_ = l.File()
			}
		}()
	}
	wg.Wait()
}

func TestSnapshotOrdering(t *testing.T) {
	got, err := snapshotEnv(map[string]string{"B": "2", "A": "1"})
	if err != nil {
		t.Fatalf("snapshotEnv: %v", err)
	}
	if string(got) != `{"A":"1","B":"2"}` {
		t.Fatalf("snapshot = %s", got)
	}
	if OctalEscape([]byte{'A', 0, '%'}) != `\101\000\045` {
		t.Fatalf("OctalEscape = %q", OctalEscape([]byte{'A', 0, '%'}))
	}
}
