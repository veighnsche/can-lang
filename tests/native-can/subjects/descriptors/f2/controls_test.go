package f2

import (
	"bytes"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/tests/native-can/subjects/descriptors/f1"
	"github.com/veighnsche/can-lang/tools/native-test-owner/journal"
	"github.com/veighnsche/can-lang/tools/native-test-owner/process"
)

const (
	testShell = "/bin/sh"
	testCat   = "/bin/cat"
	testSleep = "/bin/sleep"
)

// requireBin skips when a bounded local child binary is unavailable.
func requireBin(t *testing.T, paths ...string) {
	t.Helper()
	for _, p := range paths {
		if fi, err := os.Stat(p); err != nil || fi.IsDir() {
			t.Skipf("bounded child binary %s unavailable: %v", p, err)
		}
	}
}

func testOwner(t *testing.T) *process.Owner {
	t.Helper()
	o, err := process.New(journal.OpenMemory(), os.Getpid(), "f2-test-owner")
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

// testGeneration builds one generation directory holding its lease
// file, with the launcher's share acquired. The TempDir vanishes with
// the test; the lease closes on cleanup.
func testGeneration(t *testing.T, name string) (dir, path string, l *process.Lease) {
	t.Helper()
	dir = t.TempDir()
	path = LeasePath(dir)
	var err error
	l, err = process.AcquireLease(path)
	if err != nil {
		t.Fatalf("AcquireLease: %v", err)
	}
	t.Cleanup(func() { _ = l.Close() })
	return dir, path, l
}

func f2Env() map[string]string {
	return map[string]string{"F2_K": "v"}
}

func shArgs(script string, extras ...string) []string {
	return append([]string{"-c", script, "sh"}, extras...)
}

func mustAbort(o *process.Owner, id process.Identity) { _ = o.Abort(id) }

// Readiness is gated on the owner's facts: a holding child is ready
// once its ack is accepted while the status channel stays open;
// inheritance alone never implies readiness.
func TestReadinessGate(t *testing.T) {
	requireBin(t, testShell, testCat, testSleep)
	t.Run("ready-hold", func(t *testing.T) {
		o := testOwner(t)
		_, _, l := testGeneration(t, "ready")
		opID := "f2ready1"
		env := f2Env()
		frame := f1.MustAckFrameFor(opID, env)
		script := ReadyHoldScript(testCat, testSleep, frame, 30)
		id, err := o.Spawn(opID, process.Spec{Executable: testShell, Args: shArgs(script), Env: env, WithLease: true}, l)
		if err != nil {
			t.Fatalf("Spawn: %v", err)
		}
		defer mustAbort(o, id)
		// The reused ack frame is exactly what the owner expects.
		if want, err := o.ExpectedAck(id); err != nil || !bytes.Equal(want, frame) {
			t.Fatalf("ExpectedAck = %x err=%v, want reused frame", want, err)
		}
		// Ready means accepted-but-open: the ack landed, EOF is
		// withheld by the live holder, the child is not reaped. The
		// offer joins at Release, so readiness asserts no writer
		// facts mid-flight.
		if err := o.CollectStatus(id, 5*time.Second, 300*time.Millisecond); err == nil {
			t.Fatalf("CollectStatus: want EOF timeout, got nil")
		}
		f, _ := o.Facts(id)
		if !f.Accepted || f.EOF || f.Reaped || f.ChildExit != nil {
			t.Fatalf("ready: facts=%+v, want accepted, open, unreaped", f)
		}
	})
	t.Run("never-ready", func(t *testing.T) {
		o := testOwner(t)
		_, _, l := testGeneration(t, "neverready")
		env := f2Env()
		id, err := o.Spawn("f2neverready1", process.Spec{Executable: testShell, Args: shArgs(`exit 0`), Env: env, WithLease: true}, l)
		if err != nil {
			t.Fatalf("Spawn: %v", err)
		}
		defer mustAbort(o, id)
		// The lease was inherited, yet no ack can ever follow.
		fdMap, err := o.FDMap(id)
		if err != nil {
			t.Fatalf("FDMap: %v", err)
		}
		if fdMap[process.FDLease] != "lease" {
			t.Fatalf("FDMap = %v, want fd 4 recorded as the lease", fdMap)
		}
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
		if rep.Clean || rep.Facts.Accepted {
			t.Fatalf("Clean=%v facts=%+v, want readiness red", rep.Clean, rep.Facts)
		}
		f := rep.Facts
		if !f.EOF || !f.WriterDone {
			t.Fatalf("facts=%+v, want EOF observed and writer joined", f)
		}
		if f.OfferedBytes < 0 || f.OfferedBytes > len(f1.MustRenderSnapshot(env)) {
			t.Fatalf("facts=%+v, want bounded offered bytes", f)
		}
		if f.Offered && f.WriterErr != "" {
			t.Fatalf("facts=%+v, want no writer error on a full offer", f)
		}
	})
	t.Run("inheritance-precedes-readiness", func(t *testing.T) {
		o := testOwner(t)
		_, path, l := testGeneration(t, "precedes")
		opID := "f2precedes1"
		env := f2Env()
		script := ReadyHoldScript(testCat, testSleep, f1.MustAckFrameFor(opID, env), 30)
		id, err := o.Spawn(opID, process.Spec{Executable: testShell, Args: shArgs(script), Env: env, WithLease: true}, l)
		if err != nil {
			t.Fatalf("Spawn: %v", err)
		}
		defer mustAbort(o, id)
		// Fork establishes the inheritance; no readiness needed.
		fdMap, err := o.FDMap(id)
		if err != nil {
			t.Fatalf("FDMap: %v", err)
		}
		if fdMap[process.FDLease] != "lease" {
			t.Fatalf("FDMap = %v, want fd 4 inherited at fork", fdMap)
		}
		if err := o.CloseLeaseEarly(id); err != nil {
			t.Fatalf("CloseLeaseEarly: %v", err)
		}
		if w := WitnessLease(path); !w.Held || w.Err != nil {
			t.Fatalf("witness before readiness: %+v, want held", w)
		}
		// Readiness still arrives afterwards on its own facts.
		if err := o.CollectStatus(id, 5*time.Second, 300*time.Millisecond); err == nil {
			t.Fatalf("CollectStatus: want EOF timeout, got nil")
		}
		f, _ := o.Facts(id)
		if !f.Accepted || f.EOF {
			t.Fatalf("ready: facts=%+v, want accepted-but-open", f)
		}
	})
}

// The core protection fact, staged: the inherited kernel lease keeps
// the generation held across the launcher's exit, the owner's staged
// release observes held-then-released, and holder death releases. The
// same witness reports both states, so neither reading is vacuous.
func TestLeaseHeldAcrossLauncherExit(t *testing.T) {
	requireBin(t, testShell, testCat, testSleep)
	o := testOwner(t)
	_, path, l := testGeneration(t, "launcher")
	opID := "f2launcher1"
	env := f2Env()
	script := ReadyHoldScript(testCat, testSleep, f1.MustAckFrameFor(opID, env), 30)
	id, err := o.Spawn(opID, process.Spec{Executable: testShell, Args: shArgs(script), Env: env, WithLease: true}, l)
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	defer mustAbort(o, id)
	// Readiness gate: accepted-but-open.
	if err := o.CollectStatus(id, 5*time.Second, 300*time.Millisecond); err == nil {
		t.Fatalf("CollectStatus: want EOF timeout, got nil")
	}
	// First clean attempt while the holder lives: protection is
	// observed held, release is refused on the withheld EOF.
	rep, err := o.Release(id)
	if err != nil {
		t.Fatalf("Release#1: %v", err)
	}
	if rep.Clean || rep.Reason != "status EOF not observed" {
		t.Fatalf("Release#1: Clean=%v reason=%q, want EOF refusal", rep.Clean, rep.Reason)
	}
	if !rep.Lease.Inherited || !rep.Lease.Held || rep.Lease.Released {
		t.Fatalf("Release#1: lease=%+v, want inherited+held, not released", rep.Lease)
	}
	if !rep.Facts.Offered || rep.Facts.OfferedBytes != len(f1.MustRenderSnapshot(env)) {
		t.Fatalf("Release#1: facts=%+v, want the full snapshot offered", rep.Facts)
	}
	// Launcher exit: the owner's share closes; the child still holds
	// fd 4, so the independent witness stays held.
	if err := o.CloseLeaseEarly(id); err != nil {
		t.Fatalf("CloseLeaseEarly: %v", err)
	}
	if w := WitnessLease(path); !w.Held || w.Err != nil {
		t.Fatalf("witness after launcher exit: %+v, want held", w)
	}
	f, _ := o.Facts(id)
	if f.Reaped || f.ChildExit != nil {
		t.Fatalf("after launcher exit: facts=%+v, want holder unreaped", f)
	}
	// Holder death releases: kill, reap, re-collect the EOF the pipe
	// close delivers.
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
	if w := WitnessLease(path); w.Held || w.Err != nil {
		t.Fatalf("witness after holder death: %+v, want released", w)
	}
	if err := o.CollectStatus(id, time.Second, 5*time.Second); err != nil {
		t.Fatalf("CollectStatus after kill: %v", err)
	}
	rep, err = o.Release(id)
	if err != nil {
		t.Fatalf("Release#2: %v", err)
	}
	if rep.Clean || rep.Reason != "child did not exit 0" {
		t.Fatalf("Release#2: Clean=%v reason=%q, want child-exit refusal", rep.Clean, rep.Reason)
	}
	if !rep.Lease.Inherited || !rep.Lease.Held || !rep.Lease.Released || rep.Orphan {
		t.Fatalf("Release#2: lease=%+v orphan=%v, want held-then-released, no orphan", rep.Lease, rep.Orphan)
	}
}

// Without fd 4 there is no protection: closing the launcher's share
// loses the generation while the owner still retains a live child,
// and a fully well-behaved child that never inherited still refuses
// release on the missing lease.
func TestMissingLeaseLosesProtection(t *testing.T) {
	requireBin(t, testShell, testCat, testSleep)
	o := testOwner(t)
	_, path, l := testGeneration(t, "missing")
	env := f2Env()
	// The witness is live: the launcher's own share holds first.
	if w := WitnessLease(path); !w.Held || w.Err != nil {
		t.Fatalf("witness before close: %+v, want held", w)
	}
	// A live child elsewhere, holding no fd 4.
	sleeper, err := o.Spawn("f2missing-sleeper", process.Spec{Executable: testShell, Args: shArgs(`exec ` + testSleep + ` 30`), Env: env}, nil)
	if err != nil {
		t.Fatalf("Spawn sleeper: %v", err)
	}
	defer mustAbort(o, sleeper)
	if n := o.Live(); n != 1 {
		t.Fatalf("Live() = %d, want 1 retained child", n)
	}
	fdMap, err := o.FDMap(sleeper)
	if err != nil {
		t.Fatalf("FDMap: %v", err)
	}
	if len(fdMap) != 4 {
		t.Fatalf("FDMap = %v, want exactly 0-3 (no lease)", fdMap)
	}
	// The launcher's share closes with no inheritor: protection is
	// lost although the child is alive (its later signaled exit
	// proves it never exited before the kill).
	if err := l.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if w := WitnessLease(path); w.Held || w.Err != nil {
		t.Fatalf("witness with live no-lease child: %+v, want lost", w)
	}
	if err := o.Kill(sleeper); err != nil {
		t.Fatalf("Kill: %v", err)
	}
	exit, err := o.Wait(sleeper, 10*time.Second)
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if !exit.Signaled {
		t.Fatalf("exit=%+v, want signaled (alive past the close)", exit)
	}
	// A drain+ack+exit-0 child without fd 4: every fact green except
	// the lease, and release refuses on exactly that.
	opID := "f2missing-clean"
	script := ReadyExitScript(testCat, f1.MustAckFrameFor(opID, env), 0)
	id, err := o.Spawn(opID, process.Spec{Executable: testShell, Args: shArgs(script), Env: env}, nil)
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	defer mustAbort(o, id)
	if err := o.CollectStatus(id, 5*time.Second, 5*time.Second); err != nil {
		t.Fatalf("CollectStatus: %v", err)
	}
	if exit, err := o.Wait(id, 10*time.Second); err != nil || exit.Code != 0 {
		t.Fatalf("Wait: exit=%+v err=%v, want code 0", exit, err)
	}
	rep, err := o.Release(id)
	if err != nil {
		t.Fatalf("Release: %v", err)
	}
	if rep.Clean || rep.Reason != "lease never inherited" {
		t.Fatalf("Clean=%v reason=%q, want inheritance refusal", rep.Clean, rep.Reason)
	}
	if rep.Lease.Inherited || rep.Lease.Held || rep.Lease.Released {
		t.Fatalf("lease=%+v, want no lease facts at all", rep.Lease)
	}
	f := rep.Facts
	if !f.Accepted || f.Malformed || !f.EOF || f.ChildExit == nil || f.ChildExit.Code != 0 {
		t.Fatalf("facts=%+v, want every non-lease fact green", f)
	}
}

// Early closes lose protection: a share closed with no inheritor (or
// before spawn) never protects, while an early close beside a live
// inheritor disturbs nothing and refuses a second close.
func TestEarlyCloseLosesProtection(t *testing.T) {
	requireBin(t, testShell, testCat, testSleep)
	t.Run("share-only", func(t *testing.T) {
		o := testOwner(t)
		_, path, l := testGeneration(t, "shareonly")
		// The launcher's own share alone holds the lock ...
		if w := WitnessLease(path); !w.Held || w.Err != nil {
			t.Fatalf("witness before close: %+v, want held", w)
		}
		// ... and closing it with no child loses protection at once.
		if err := l.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
		if w := WitnessLease(path); w.Held || w.Err != nil {
			t.Fatalf("witness after early close: %+v, want lost", w)
		}
		if n := o.Live(); n != 0 {
			t.Fatalf("Live() = %d, want 0", n)
		}
	})
	t.Run("closed-before-spawn", func(t *testing.T) {
		o := testOwner(t)
		_, path, l := testGeneration(t, "closedspawn")
		if err := l.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
		// A closed share still spawns, but no descriptor survives to
		// inherit: the child observes fd 4 closed although the owner
		// recorded a lease, and the witness stays lost throughout.
		probe, err := o.Spawn("f2closedspawn-probe", process.Spec{Executable: testShell, Args: shArgs(`if ` + testCat + ` <&4 >/dev/null 2>&1; then exit 7; else exit 0; fi`), Env: f2Env(), WithLease: true}, l)
		if err != nil {
			t.Fatalf("Spawn probe: %v", err)
		}
		defer mustAbort(o, probe)
		if exit, err := o.Wait(probe, 10*time.Second); err != nil || exit.Code != 0 {
			t.Fatalf("probe: exit=%+v err=%v, want code 0 (fd 4 closed child-side)", exit, err)
		}
		fdMap, err := o.FDMap(probe)
		if err != nil {
			t.Fatalf("FDMap: %v", err)
		}
		if fdMap[process.FDLease] != "lease" {
			t.Fatalf("FDMap = %v, want fd 4 recorded as the lease", fdMap)
		}
		// A fully well-behaved child on the closed share still
		// refuses release: protection was never observed.
		opID := "f2closedspawn1"
		env := f2Env()
		script := ReadyExitScript(testCat, f1.MustAckFrameFor(opID, env), 0)
		id, err := o.Spawn(opID, process.Spec{Executable: testShell, Args: shArgs(script), Env: env, WithLease: true}, l)
		if err != nil {
			t.Fatalf("Spawn: %v", err)
		}
		defer mustAbort(o, id)
		if err := o.CollectStatus(id, 5*time.Second, 5*time.Second); err != nil {
			t.Fatalf("CollectStatus: %v", err)
		}
		if exit, err := o.Wait(id, 10*time.Second); err != nil || exit.Code != 0 {
			t.Fatalf("Wait: exit=%+v err=%v, want code 0", exit, err)
		}
		rep, err := o.Release(id)
		if err != nil {
			t.Fatalf("Release: %v", err)
		}
		if rep.Clean || rep.Reason != "lease protection never observed" {
			t.Fatalf("Clean=%v reason=%q, want protection refusal", rep.Clean, rep.Reason)
		}
		if !rep.Lease.Inherited || rep.Lease.Held || rep.Lease.Released {
			t.Fatalf("lease=%+v, want inherited-but-never-held", rep.Lease)
		}
		if w := WitnessLease(path); w.Held || w.Err != nil {
			t.Fatalf("witness: %+v, want lost", w)
		}
	})
	t.Run("double-early-close", func(t *testing.T) {
		o := testOwner(t)
		_, path, l := testGeneration(t, "doubleclose")
		opID := "f2doubleclose1"
		env := f2Env()
		script := ReadyHoldScript(testCat, testSleep, f1.MustAckFrameFor(opID, env), 30)
		id, err := o.Spawn(opID, process.Spec{Executable: testShell, Args: shArgs(script), Env: env, WithLease: true}, l)
		if err != nil {
			t.Fatalf("Spawn: %v", err)
		}
		defer mustAbort(o, id)
		if err := o.CloseLeaseEarly(id); err != nil {
			t.Fatalf("CloseLeaseEarly: %v", err)
		}
		if err := o.CloseLeaseEarly(id); !errors.Is(err, process.ErrState) {
			t.Fatalf("second CloseLeaseEarly: err=%v, want ErrState", err)
		}
		// The live inheritor is undisturbed by the early close.
		if w := WitnessLease(path); !w.Held || w.Err != nil {
			t.Fatalf("witness after early close: %+v, want held", w)
		}
		if err := o.Kill(id); err != nil {
			t.Fatalf("Kill: %v", err)
		}
		if _, err := o.Wait(id, 10*time.Second); err != nil {
			t.Fatalf("Wait: %v", err)
		}
		if w := WitnessLease(path); w.Held || w.Err != nil {
			t.Fatalf("witness after holder death: %+v, want released", w)
		}
	})
}

// The clean end-to-end: a ready child that releases (drains, acks,
// exits 0) lets the staged release observe inherited, held and
// released together. A nonzero exit refuses on its own cause while
// the lease facts still record held-then-released beside it.
func TestCleanReleaseAfterChildRelease(t *testing.T) {
	requireBin(t, testShell, testCat)
	t.Run("clean", func(t *testing.T) {
		o := testOwner(t)
		_, path, l := testGeneration(t, "clean")
		opID := "f2clean1"
		env := f2Env()
		script := ReadyExitScript(testCat, f1.MustAckFrameFor(opID, env), 0)
		id, err := o.Spawn(opID, process.Spec{Executable: testShell, Args: shArgs(script), Env: env, WithLease: true}, l)
		if err != nil {
			t.Fatalf("Spawn: %v", err)
		}
		defer mustAbort(o, id)
		if err := o.CollectStatus(id, 5*time.Second, 5*time.Second); err != nil {
			t.Fatalf("CollectStatus: %v", err)
		}
		f, _ := o.Facts(id)
		if !f.Accepted || !f.EOF || f.Reaped {
			t.Fatalf("after status: facts=%+v, want accepted+EOF, unreaped", f)
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
		if !rep.Lease.Inherited || !rep.Lease.Held || !rep.Lease.Released || rep.Orphan {
			t.Fatalf("lease=%+v orphan=%v, want inherited+held+released, no orphan", rep.Lease, rep.Orphan)
		}
		// A clean release forgets the child: the same operation is
		// refused afterwards, so the first decision was live, not
		// vacuous.
		if _, err := o.Release(id); !errors.Is(err, process.ErrNotFound) {
			t.Fatalf("second Release: err=%v, want ErrNotFound", err)
		}
		// Prune after release removes the lease file; the path
		// witnesses missing afterwards.
		out := PruneLeaseFile(path)
		if !out.Removed || out.RemoveErr != nil {
			t.Fatalf("Prune = %+v, want removed", out)
		}
		if out.After.Held || out.After.Err == nil {
			t.Fatalf("Prune.After = %+v, want missing-path witness", out.After)
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("Stat after prune: err=%v, want path gone", err)
		}
	})
	t.Run("exit3", func(t *testing.T) {
		o := testOwner(t)
		_, path, l := testGeneration(t, "exit3")
		opID := "f2exit31"
		env := f2Env()
		script := ReadyExitScript(testCat, f1.MustAckFrameFor(opID, env), 3)
		id, err := o.Spawn(opID, process.Spec{Executable: testShell, Args: shArgs(script), Env: env, WithLease: true}, l)
		if err != nil {
			t.Fatalf("Spawn: %v", err)
		}
		defer mustAbort(o, id)
		if err := o.CollectStatus(id, 5*time.Second, 5*time.Second); err != nil {
			t.Fatalf("CollectStatus: %v", err)
		}
		exit, err := o.Wait(id, 10*time.Second)
		if err != nil {
			t.Fatalf("Wait: %v", err)
		}
		if exit.Code != 3 || exit.Signaled {
			t.Fatalf("exit = %+v, want code 3", exit)
		}
		rep, err := o.Release(id)
		if err != nil {
			t.Fatalf("Release: %v", err)
		}
		if rep.Clean || rep.Reason != "child did not exit 0" {
			t.Fatalf("Clean=%v reason=%q, want child-exit refusal", rep.Clean, rep.Reason)
		}
		// The exit refuses, yet the lease still records its own
		// held-then-released history beside the red exit.
		if !rep.Lease.Inherited || !rep.Lease.Held || !rep.Lease.Released || rep.Orphan {
			t.Fatalf("lease=%+v orphan=%v, want inherited+held+released, no orphan", rep.Lease, rep.Orphan)
		}
		out := PruneLeaseFile(path)
		if !out.Removed || out.RemoveErr != nil {
			t.Fatalf("Prune = %+v, want removed", out)
		}
	})
}

// Prune observations are gated on the witness: pruning after release
// clears the path, while pruning under a live holder unlinks anyway
// (locks never block unlink) and destroys path-bound protection — a
// fresh lock becomes obtainable although the holder still lives.
func TestPruneObservations(t *testing.T) {
	requireBin(t, testShell, testCat, testSleep)
	t.Run("after-release", func(t *testing.T) {
		o := testOwner(t)
		_, path, l := testGeneration(t, "pruneclean")
		opID := "f2pruneclean1"
		env := f2Env()
		script := ReadyExitScript(testCat, f1.MustAckFrameFor(opID, env), 0)
		id, err := o.Spawn(opID, process.Spec{Executable: testShell, Args: shArgs(script), Env: env, WithLease: true}, l)
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
		if !rep.Clean || !rep.Lease.Released {
			t.Fatalf("Clean=%v lease=%+v, want clean release first", rep.Clean, rep.Lease)
		}
		out := PruneLeaseFile(path)
		if !out.Removed || out.RemoveErr != nil {
			t.Fatalf("Prune = %+v, want removed", out)
		}
		if out.After.Held || out.After.Err == nil {
			t.Fatalf("Prune.After = %+v, want missing-path witness", out.After)
		}
		// The pruned path accepts a fresh generation lease.
		fresh, err := process.AcquireLease(path)
		if err != nil {
			t.Fatalf("re-acquire after prune: %v", err)
		}
		if w := WitnessLease(path); !w.Held || w.Err != nil {
			_ = fresh.Close()
			t.Fatalf("witness on fresh lease: %+v, want held", w)
		}
		if err := fresh.Close(); err != nil {
			t.Fatalf("Close fresh: %v", err)
		}
		if err := os.Remove(path); err != nil {
			t.Fatalf("Remove fresh: %v", err)
		}
	})
	t.Run("while-held", func(t *testing.T) {
		o := testOwner(t)
		_, path, l := testGeneration(t, "pruneheld")
		opID := "f2pruneheld1"
		env := f2Env()
		script := ReadyHoldScript(testCat, testSleep, f1.MustAckFrameFor(opID, env), 30)
		id, err := o.Spawn(opID, process.Spec{Executable: testShell, Args: shArgs(script), Env: env, WithLease: true}, l)
		if err != nil {
			t.Fatalf("Spawn: %v", err)
		}
		defer mustAbort(o, id)
		// Readiness gate first: the holder is up and responsive.
		if err := o.CollectStatus(id, 5*time.Second, 300*time.Millisecond); err == nil {
			t.Fatalf("CollectStatus: want EOF timeout, got nil")
		}
		if w := WitnessLease(path); !w.Held || w.Err != nil {
			t.Fatalf("witness before prune: %+v, want held", w)
		}
		// Pruning under a live holder still unlinks: flock never
		// blocks unlink, so Removed is true and the path witness
		// goes missing while the holder lives on.
		out := PruneLeaseFile(path)
		if !out.Removed || out.RemoveErr != nil {
			t.Fatalf("Prune = %+v, want removed despite the hold", out)
		}
		if out.After.Held || out.After.Err == nil {
			t.Fatalf("Prune.After = %+v, want missing-path witness", out.After)
		}
		if n := o.Live(); n != 1 {
			t.Fatalf("Live() = %d, want the holder retained", n)
		}
		// Path-bound protection is destroyed: a fresh lock at the
		// same path is obtainable although the holder still lives
		// (its lock stays on the orphaned inode).
		stolen, err := process.AcquireLease(path)
		if err != nil {
			t.Fatalf("re-acquire under live holder: %v (want success: protection lost)", err)
		}
		if err := stolen.Close(); err != nil {
			t.Fatalf("Close stolen: %v", err)
		}
		if err := os.Remove(path); err != nil {
			t.Fatalf("Remove stolen: %v", err)
		}
		// The holder was alive throughout: only the kill reaps it.
		if err := o.Kill(id); err != nil {
			t.Fatalf("Kill: %v", err)
		}
		exit, err := o.Wait(id, 10*time.Second)
		if err != nil {
			t.Fatalf("Wait: %v", err)
		}
		if !exit.Signaled {
			t.Fatalf("exit=%+v, want signaled (alive past the prune)", exit)
		}
	})
	t.Run("held-no-child", func(t *testing.T) {
		testOwner(t)
		_, path, _ := testGeneration(t, "prunehold")
		// A launcher-held path with no child at all: held before,
		// unlinked by the prune, missing after.
		if w := WitnessLease(path); !w.Held || w.Err != nil {
			t.Fatalf("witness before prune: %+v, want held", w)
		}
		out := PruneLeaseFile(path)
		if !out.Removed || out.RemoveErr != nil {
			t.Fatalf("Prune = %+v, want removed", out)
		}
		if out.After.Held || out.After.Err == nil {
			t.Fatalf("Prune.After = %+v, want missing-path witness", out.After)
		}
	})
}

// Lease error facts are observable failures, proving the controls
// above are non-vacuous: invalid requests refuse, a missing share
// refuses the early close, and a foreign start token is never
// authority even for a live PID.
func TestLeaseErrorFacts(t *testing.T) {
	requireBin(t, testShell, testCat, testSleep)
	t.Run("acquire-and-spawn-validation", func(t *testing.T) {
		o := testOwner(t)
		if _, err := process.AcquireLease(""); !errors.Is(err, process.ErrInvalid) {
			t.Fatalf("AcquireLease(\"\"): err=%v, want ErrInvalid", err)
		}
		_, _, l := testGeneration(t, "errvalid")
		if _, err := o.Spawn("f2errvalid1", process.Spec{Executable: testShell, Args: shArgs(`exit 0`), Env: f2Env(), WithLease: true}, nil); !errors.Is(err, process.ErrInvalid) {
			t.Fatalf("Spawn WithLease without lease: err=%v, want ErrInvalid", err)
		}
		if _, err := o.Spawn("f2errvalid2", process.Spec{Executable: testShell, Args: shArgs(`exit 0`), Env: f2Env()}, l); !errors.Is(err, process.ErrInvalid) {
			t.Fatalf("Spawn lease without WithLease: err=%v, want ErrInvalid", err)
		}
		if n := o.Live(); n != 0 {
			t.Fatalf("Live() = %d, want 0 (refused spawns retain nothing)", n)
		}
	})
	t.Run("early-close-without-lease", func(t *testing.T) {
		o := testOwner(t)
		id, err := o.Spawn("f2errclose1", process.Spec{Executable: testShell, Args: shArgs(`exit 0`), Env: f2Env()}, nil)
		if err != nil {
			t.Fatalf("Spawn: %v", err)
		}
		defer mustAbort(o, id)
		if err := o.CloseLeaseEarly(id); !errors.Is(err, process.ErrState) {
			t.Fatalf("CloseLeaseEarly without lease: err=%v, want ErrState", err)
		}
		if _, err := o.Wait(id, 10*time.Second); err != nil {
			t.Fatalf("Wait: %v", err)
		}
	})
	t.Run("foreign-token", func(t *testing.T) {
		o := testOwner(t)
		_, _, l := testGeneration(t, "errtoken")
		opID := "f2errtoken1"
		env := f2Env()
		script := ReadyHoldScript(testCat, testSleep, f1.MustAckFrameFor(opID, env), 30)
		id, err := o.Spawn(opID, process.Spec{Executable: testShell, Args: shArgs(script), Env: env, WithLease: true}, l)
		if err != nil {
			t.Fatalf("Spawn: %v", err)
		}
		defer mustAbort(o, id)
		foreign := process.Identity{PID: id.PID, StartToken: "foreign-token"}
		if _, err := o.Release(foreign); !errors.Is(err, process.ErrWrongOwner) {
			t.Fatalf("Release with foreign token: err=%v, want ErrWrongOwner", err)
		}
		if _, err := o.Facts(foreign); !errors.Is(err, process.ErrWrongOwner) {
			t.Fatalf("Facts with foreign token: err=%v, want ErrWrongOwner", err)
		}
		// The refused operations disturbed nothing: the real child
		// is still killable and reapable through its own identity.
		if err := o.Kill(id); err != nil {
			t.Fatalf("Kill: %v", err)
		}
		if _, err := o.Wait(id, 10*time.Second); err != nil {
			t.Fatalf("Wait: %v", err)
		}
	})
}
