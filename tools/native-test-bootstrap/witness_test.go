package bootstrap

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func testLimits() Limits {
	return Limits{MaxChildren: 4, MaxDescriptors: 12, MaxScratchBytes: 1 << 20, MaxScratchEntries: 100}
}

// fixture resolves a real executable for bounded subprocess fixtures.
func fixture(t *testing.T, name string) string {
	t.Helper()
	p, err := exec.LookPath(name)
	if err != nil {
		t.Skipf("fixture %q unavailable: %v", name, err)
	}
	return p
}

func TestFiniteAuthority(t *testing.T) {
	tw := t.TempDir()
	w, err := New(filepath.Join(tw, "witness"), Limits{MaxChildren: 1, MaxDescriptors: DescriptorsPerChild, MaxScratchBytes: 100, MaxScratchEntries: 10})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	sleep := fixture(t, "sleep")

	// Unresolvable executables and unknown scratch never register.
	if err := w.Register(ChildSpec{ID: "bad", Path: "definitely-not-a-real-binary-xyz"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("bad executable: got %v, want ErrInvalid", err)
	}
	if err := w.Register(ChildSpec{ID: "bad2", Path: sleep, ScratchID: "nope"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown scratch: got %v, want ErrNotFound", err)
	}
	// Identical re-registration rejoins; conflicting spec is rejected.
	spec := ChildSpec{ID: "w1", Path: sleep, Args: []string{"30"}}
	if err := w.Register(spec); err != nil {
		t.Fatal(err)
	}
	if err := w.Register(spec); err != nil {
		t.Fatalf("identical re-register: %v", err)
	}
	changed := spec
	changed.Args = []string{"31"}
	if err := w.Register(changed); !errors.Is(err, ErrConflict) {
		t.Fatalf("conflicting register: got %v, want ErrConflict", err)
	}
	// Finite child authority: the second child is refused before any effect.
	if err := w.Register(ChildSpec{ID: "w2", Path: sleep, Args: []string{"30"}}); !errors.Is(err, ErrExhausted) {
		t.Fatalf("second child: got %v, want ErrExhausted", err)
	}
	// Scratch budgets are finite too.
	if err := w.RegisterScratch(ScratchSpec{ID: "s1", Path: filepath.Join(tw, "s1"), MaxBytes: 60, MaxEntries: 6}); err != nil {
		t.Fatal(err)
	}
	if err := w.RegisterScratch(ScratchSpec{ID: "s2", Path: filepath.Join(tw, "s2"), MaxBytes: 60, MaxEntries: 6}); !errors.Is(err, ErrExhausted) {
		t.Fatalf("scratch over budget: got %v, want ErrExhausted", err)
	}
	// A lost witness grants no new admission.
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := w.Admit(ChildSpec{ID: "w3", Path: sleep}); !errors.Is(err, ErrWitnessLost) {
		t.Fatalf("admit after loss: got %v, want ErrWitnessLost", err)
	}
}

func TestExitFactNonzero(t *testing.T) {
	w, err := New(filepath.Join(t.TempDir(), "witness"), testLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	sh := fixture(t, "sh")
	if err := w.Register(ChildSpec{ID: "failer", Path: sh, Args: []string{"-c", "exit 3"}}); err != nil {
		t.Fatal(err)
	}
	if err := w.Start("failer"); err != nil {
		t.Fatal(err)
	}
	fact, err := w.Wait("failer", 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if !fact.Exited || !fact.Reaped || fact.Code != 3 || fact.Signal != "" {
		t.Fatalf("exit fact: %+v, want code 3 exited+reaped", fact)
	}
	// A fully released registration receipts clean.
	if err := w.CheckRelease(ReleaseClaim{Children: []string{"failer"}}); err != nil {
		t.Fatalf("CheckRelease clean claim: %v", err)
	}
}

// TestWorkerDeathLeavesTAlive kills a fixture child from outside T (SIGKILL,
// as an N/worker death would land) and requires T to stay functional.
func TestWorkerDeathLeavesTAlive(t *testing.T) {
	w, err := New(filepath.Join(t.TempDir(), "witness"), testLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	sleep := fixture(t, "sleep")
	if err := w.Register(ChildSpec{ID: "worker", Path: sleep, Args: []string{"30"}}); err != nil {
		t.Fatal(err)
	}
	if err := w.Start("worker"); err != nil {
		t.Fatal(err)
	}
	pid := w.Snapshot().Children["worker"].PID
	if pid <= 0 {
		t.Fatal("missing start identity")
	}
	// External death, not via T's Stop.
	proc, err := os.FindProcess(pid)
	if err != nil {
		t.Fatal(err)
	}
	if err := proc.Kill(); err != nil {
		t.Fatal(err)
	}
	fact, err := w.Wait("worker", 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if !fact.Exited || !fact.Reaped || fact.Signal == "" {
		t.Fatalf("death fact: %+v, want signaled+reaped", fact)
	}
	// T is alive: new admission, start, stop and a clean receipt all work.
	if err := w.Admit(ChildSpec{ID: "next", Path: sleep, Args: []string{"30"}}); err != nil {
		t.Fatalf("admit after worker death: %v", err)
	}
	if err := w.Start("next"); err != nil {
		t.Fatal(err)
	}
	if err := w.Stop("next", 100*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if err := w.CheckRelease(ReleaseClaim{Children: []string{"worker", "next"}}); err != nil {
		t.Fatalf("CheckRelease after worker death: %v", err)
	}
}

func TestStopEscalation(t *testing.T) {
	w, err := New(filepath.Join(t.TempDir(), "witness"), testLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	sleep := fixture(t, "sleep")
	if err := w.Register(ChildSpec{ID: "sleeper", Path: sleep, Args: []string{"30"}}); err != nil {
		t.Fatal(err)
	}
	if err := w.Start("sleeper"); err != nil {
		t.Fatal(err)
	}
	if err := w.Stop("sleeper", 100*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	fact, err := w.Wait("sleeper", 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if !fact.Exited || !fact.Reaped || len(fact.Forced) == 0 {
		t.Fatalf("stop fact: %+v, want exited+reaped with forced actions", fact)
	}
	// Repeated Stop joins the same disposal.
	if err := w.Stop("sleeper", 100*time.Millisecond); err != nil {
		t.Fatalf("repeated Stop: %v", err)
	}
}

func TestWaitDeadlineKeepsOwnership(t *testing.T) {
	w, err := New(filepath.Join(t.TempDir(), "witness"), testLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	sleep := fixture(t, "sleep")
	if err := w.Register(ChildSpec{ID: "slow", Path: sleep, Args: []string{"30"}}); err != nil {
		t.Fatal(err)
	}
	if err := w.Start("slow"); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Wait("slow", 50*time.Millisecond); !errors.Is(err, ErrTimeout) {
		t.Fatalf("short wait: got %v, want ErrTimeout", err)
	}
	// A running child cannot be receipted as released.
	if err := w.CheckRelease(ReleaseClaim{Children: []string{"slow"}}); err == nil {
		t.Fatal("CheckRelease receipted a running child")
	} else {
		var mm *MismatchError
		if !errors.As(err, &mm) {
			t.Fatalf("CheckRelease running child: got %T, want *MismatchError", err)
		}
	}
	if err := w.Stop("slow", 100*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if err := w.CheckRelease(ReleaseClaim{Children: []string{"slow"}}); err != nil {
		t.Fatalf("CheckRelease after stop: %v", err)
	}
}

func TestReleaseWitnessScratch(t *testing.T) {
	root := t.TempDir()
	w, err := New(filepath.Join(root, "witness"), testLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	sh := fixture(t, "sh")
	scratchDir := filepath.Join(root, "case-scratch")
	if err := os.MkdirAll(scratchDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := w.RegisterScratch(ScratchSpec{ID: "case", Path: scratchDir, MaxBytes: 1024, MaxEntries: 10}); err != nil {
		t.Fatal(err)
	}
	if err := w.Register(ChildSpec{ID: "job", Path: sh, Args: []string{"-c", "exit 0"}}); err != nil {
		t.Fatal(err)
	}
	if err := w.Start("job"); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Wait("job", 5*time.Second); err != nil {
		t.Fatal(err)
	}
	// Claimed removal while the directory still exists is not a release.
	err = w.CheckRelease(ReleaseClaim{Children: []string{"job"}, Scratch: []string{"case"}})
	var mm *MismatchError
	if !errors.As(err, &mm) {
		t.Fatalf("present scratch: got %v, want *MismatchError", err)
	}
	// Unknown and unclaimed authority also refuse the green.
	if err := w.CheckRelease(ReleaseClaim{Children: []string{"ghost"}}); !errors.As(err, &mm) {
		t.Fatalf("unknown child: got %v, want *MismatchError", err)
	}
	if err := w.CheckRelease(ReleaseClaim{Children: []string{"job"}}); !errors.As(err, &mm) {
		t.Fatalf("unclaimed scratch: got %v, want *MismatchError", err)
	}
	// After the disposer actually removes the directory, the claim verifies.
	if err := os.RemoveAll(scratchDir); err != nil {
		t.Fatal(err)
	}
	if err := w.CheckRelease(ReleaseClaim{Children: []string{"job"}, Scratch: []string{"case"}}); err != nil {
		t.Fatalf("CheckRelease after disposal: %v", err)
	}
}

// TestWitnessLossCannotReceipt closes T and requires every admission and
// receipt path to fail, even for a fully clean claim.
func TestWitnessLossCannotReceipt(t *testing.T) {
	root := t.TempDir()
	wdir := filepath.Join(root, "witness")
	w, err := New(wdir, testLimits())
	if err != nil {
		t.Fatal(err)
	}
	sh := fixture(t, "sh")
	scratchDir := filepath.Join(root, "case-scratch")
	if err := os.MkdirAll(scratchDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := w.RegisterScratch(ScratchSpec{ID: "case", Path: scratchDir, MaxBytes: 1024, MaxEntries: 10}); err != nil {
		t.Fatal(err)
	}
	if err := w.Register(ChildSpec{ID: "job", Path: sh, Args: []string{"-c", "exit 0"}}); err != nil {
		t.Fatal(err)
	}
	if err := w.Start("job"); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Wait("job", 5*time.Second); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(scratchDir); err != nil {
		t.Fatal(err)
	}
	clean := ReleaseClaim{Children: []string{"job"}, Scratch: []string{"case"}}
	if err := w.CheckRelease(clean); err != nil {
		t.Fatalf("pre-loss CheckRelease: %v", err)
	}
	// Lose T.
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := w.CheckRelease(clean); !errors.Is(err, ErrWitnessLost) {
		t.Fatalf("CheckRelease after loss: got %v, want ErrWitnessLost", err)
	}
	if err := w.Admit(ChildSpec{ID: "late", Path: sh}); !errors.Is(err, ErrWitnessLost) {
		t.Fatalf("Admit after loss: got %v, want ErrWitnessLost", err)
	}
	if err := w.Register(ChildSpec{ID: "late", Path: sh}); !errors.Is(err, ErrWitnessLost) {
		t.Fatalf("Register after loss: got %v, want ErrWitnessLost", err)
	}
	if err := w.Start("job"); !errors.Is(err, ErrWitnessLost) {
		t.Fatalf("Start after loss: got %v, want ErrWitnessLost", err)
	}
	if err := w.Stop("job", time.Second); !errors.Is(err, ErrWitnessLost) {
		t.Fatalf("Stop after loss: got %v, want ErrWitnessLost", err)
	}
	if _, err := w.Wait("job", time.Second); !errors.Is(err, ErrWitnessLost) {
		t.Fatalf("Wait after loss: got %v, want ErrWitnessLost", err)
	}
	// The retained record stays inspectable but can never receipt.
	snap, err := Inspect(wdir)
	if err != nil {
		t.Fatal(err)
	}
	if !snap.WitnessLost {
		t.Fatal("inspected record must report the witness lost")
	}
	if !snap.Children["job"].Fact.Exited || !snap.Children["job"].Fact.Reaped {
		t.Fatalf("inspected exit fact lost: %+v", snap.Children["job"])
	}
	// A new witness refuses to resume the old run in place.
	if _, err := New(wdir, testLimits()); !errors.Is(err, ErrPriorState) {
		t.Fatalf("re-New over prior state: got %v, want ErrPriorState", err)
	}
}

// TestWitnessLossWithRunningChild loses T while a short fixture is still
// running: admission fails immediately and the record stays inspectable.
func TestWitnessLossWithRunningChild(t *testing.T) {
	wdir := filepath.Join(t.TempDir(), "witness")
	w, err := New(wdir, testLimits())
	if err != nil {
		t.Fatal(err)
	}
	sleep := fixture(t, "sleep")
	if err := w.Register(ChildSpec{ID: "short", Path: sleep, Args: []string{"2"}}); err != nil {
		t.Fatal(err)
	}
	if err := w.Start("short"); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := w.Admit(ChildSpec{ID: "late", Path: sleep}); !errors.Is(err, ErrWitnessLost) {
		t.Fatalf("Admit after loss: got %v, want ErrWitnessLost", err)
	}
	snap, err := Inspect(wdir)
	if err != nil {
		t.Fatal(err)
	}
	if !snap.Children["short"].Started {
		t.Fatalf("inspected start fact lost: %+v", snap.Children["short"])
	}
	// The orphaned fixture exits on its own within 2s; the in-flight
	// observer records the fact to memory without a live witness. Poll the
	// read-only snapshot briefly. (The retained file keeps only pre-loss
	// lines; post-loss facts grant no authority.)
	deadline := time.Now().Add(5 * time.Second)
	for {
		if w.Snapshot().Children["short"].Fact.Exited {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("orphaned fixture exit never recorded")
		}
		time.Sleep(50 * time.Millisecond)
	}
}
