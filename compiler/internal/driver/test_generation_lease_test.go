package driver

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testGenerationOwner() TestOwner {
	return TestOwner{PID: os.Getpid(), StartToken: "test-owner-token"}
}

func TestAcquireTestGenerationValidation(t *testing.T) {
	root := outputProject(t)
	s := outputBegin(t, root)
	id, _, err := s.Stage(outputPrepared(t, s, "export const value=1n;"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = AcquireTestGeneration(nil, id, testGenerationOwner()); err == nil {
		t.Fatal("nil store acquired")
	}
	for name, owner := range map[string]TestOwner{
		"zero pid":     {PID: 0, StartToken: "tok"},
		"negative pid": {PID: -1, StartToken: "tok"},
		"empty token":  {PID: os.Getpid()},
		"long token":   {PID: os.Getpid(), StartToken: strings.Repeat("t", MaxTestOwnerTokenLen+1)},
	} {
		if _, err = AcquireTestGeneration(s, id, owner); err == nil {
			t.Fatalf("%s acquired", name)
		}
	}
	if _, err = AcquireTestGeneration(s, strings.Repeat("0", 64), testGenerationOwner()); err == nil {
		t.Fatal("missing generation acquired")
	}
	if _, err = AcquireTestGeneration(s, "../current", testGenerationOwner()); err == nil {
		t.Fatal("generation escape acquired")
	}
	before := time.Now()
	lease, err := AcquireTestGeneration(s, id, testGenerationOwner())
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	if lease.BuildID() != id {
		t.Fatalf("BuildID = %q, want %q", lease.BuildID(), id)
	}
	if lease.Owner() != testGenerationOwner() {
		t.Fatalf("Owner = %+v", lease.Owner())
	}
	if lease.AcquiredAt().Before(before) || lease.AcquiredAt().After(time.Now()) {
		t.Fatalf("AcquiredAt = %v", lease.AcquiredAt())
	}
	if lease.Lease() == nil || lease.Lease().Manifest.BuildID != id {
		t.Fatal("underlying lease missing or mismatched")
	}
}

func TestVerifyOwnerExactMatch(t *testing.T) {
	root := outputProject(t)
	s := outputBegin(t, root)
	id, _, err := s.Stage(outputPrepared(t, s, "export const value=1n;"))
	if err != nil {
		t.Fatal(err)
	}
	lease, err := AcquireTestGeneration(s, id, testGenerationOwner())
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	if err = lease.VerifyOwner(os.Getpid(), "test-owner-token"); err != nil {
		t.Fatalf("exact owner rejected: %v", err)
	}
	// A live PID with a stale or foreign token is never authority.
	for name, got := range map[string]TestOwner{
		"foreign pid":  {PID: os.Getpid() + 1, StartToken: "test-owner-token"},
		"stale token":  {PID: os.Getpid(), StartToken: "stale-token"},
		"foreign both": {PID: 1, StartToken: "x"},
		"empty token":  {PID: os.Getpid()},
		"zero pid":     {StartToken: "test-owner-token"},
		"empty owner":  {},
	} {
		err = lease.VerifyOwner(got.PID, got.StartToken)
		if !errors.Is(err, ErrTestWrongOwner) {
			t.Fatalf("%s: err = %v, want ErrTestWrongOwner", name, err)
		}
		var typed *TestOwnerError
		if !errors.As(err, &typed) || typed.BuildID != id {
			t.Fatalf("%s: err = %#v, want *TestOwnerError for %s", name, err, id)
		}
	}
}

func TestTestGenerationCloseIdempotent(t *testing.T) {
	root := outputProject(t)
	s := outputBegin(t, root)
	id, _, err := s.Stage(outputPrepared(t, s, "export const value=1n;"))
	if err != nil {
		t.Fatal(err)
	}
	lease, err := AcquireTestGeneration(s, id, testGenerationOwner())
	if err != nil {
		t.Fatal(err)
	}
	if err = lease.Close(); err != nil {
		t.Fatal(err)
	}
	if err = lease.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func TestProbeHeldMissingManifest(t *testing.T) {
	root := outputProject(t)
	s := outputBegin(t, root)
	id, directory, err := s.Stage(outputPrepared(t, s, "export const value=1n;"))
	if err != nil {
		t.Fatal(err)
	}
	lease, err := AcquireTestGeneration(s, id, testGenerationOwner())
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	if held, err := lease.ProbeHeld(); err != nil || !held {
		t.Fatalf("ProbeHeld while held: held=%v err=%v", held, err)
	}
	if err = os.Remove(filepath.Join(directory, "manifest.json")); err != nil {
		t.Fatal(err)
	}
	if _, err = lease.ProbeHeld(); err == nil {
		t.Fatal("ProbeHeld on removed manifest: want error")
	}
}

// TestTestGenerationLeaseKernelLifetime is the driver-boundary kernel
// control for P10: a child inheriting the staged manifest descriptor
// at fd 4 keeps generation protection across the launcher closing its
// share, and the independent witness observes held-then-released. A
// child spawned without fd 4 never establishes protection.
func TestTestGenerationLeaseKernelLifetime(t *testing.T) {
	root := outputProject(t)
	s := outputBegin(t, root)
	id, directory, err := s.Stage(outputPrepared(t, s, "export const value=1n;"))
	if err != nil {
		t.Fatal(err)
	}
	manifestBytes, err := os.ReadFile(filepath.Join(directory, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	lease, err := AcquireTestGeneration(s, id, testGenerationOwner())
	if err != nil {
		t.Fatal(err)
	}
	// The launcher's own share holds the lock from acquire time.
	if held, err := lease.ProbeHeld(); err != nil || !held {
		t.Fatalf("ProbeHeld after acquire: held=%v err=%v", held, err)
	}
	null, err := os.OpenFile(os.DevNull, os.O_RDONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer null.Close()
	// The child observes fd 4 verbatim, then sleeps: its fd 4 must be
	// the leased manifest, mirroring the prepareEntry fd layout.
	fdOut := filepath.Join(t.TempDir(), "fd4.out")
	cmd := exec.Command("/bin/sh", "-c", `/bin/cat /dev/fd/4 > "$1"; exec /bin/sleep 30`, "sh", fdOut)
	cmd.ExtraFiles = []*os.File{null, lease.File()}
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	holder := cmd.Process
	waited := make(chan error, 1)
	go func() { waited <- cmd.Wait() }()
	// Best-effort strays cleanup: the explicit reap below consumes the
	// outcome, so the deferred join must never block.
	defer func() {
		_ = holder.Kill()
		select {
		case <-waited:
		default:
		}
	}()
	deadline := time.Now().Add(10 * time.Second)
	for {
		if raw, rerr := os.ReadFile(fdOut); rerr == nil && len(raw) > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("child never observed fd 4")
		}
		time.Sleep(20 * time.Millisecond)
	}
	// Launcher exit: drop the owner's share while the child runs. The
	// inherited fd 4 keeps the kernel lock held.
	if err = lease.Close(); err != nil {
		t.Fatal(err)
	}
	if held, err := lease.ProbeHeld(); err != nil || !held {
		t.Fatalf("witness during hold: held=%v err=%v", held, err)
	}
	if err = holder.Kill(); err != nil {
		t.Fatal(err)
	}
	<-waited
	releaseDeadline := time.Now().Add(10 * time.Second)
	for {
		held, err := lease.ProbeHeld()
		if err != nil {
			t.Fatal(err)
		}
		if !held {
			break
		}
		if time.Now().After(releaseDeadline) {
			t.Fatal("witness still held 10s after holder death")
		}
		time.Sleep(20 * time.Millisecond)
	}
	// fd 4 carried the leased manifest bytes exactly.
	if raw, err := os.ReadFile(fdOut); err != nil || !bytes.Equal(raw, manifestBytes) {
		t.Fatalf("fd 4 observed %d bytes, manifest holds %d (err=%v)", len(raw), len(manifestBytes), err)
	}
	// Loss after a missing descriptor: a child without fd 4 never
	// establishes protection, even while it still runs.
	lease2, err := AcquireTestGeneration(s, id, TestOwner{PID: os.Getpid(), StartToken: "second-owner"})
	if err != nil {
		t.Fatal(err)
	}
	cmd2 := exec.Command("/bin/sh", "-c", `exec /bin/sleep 30`)
	cmd2.ExtraFiles = []*os.File{null}
	if err = cmd2.Start(); err != nil {
		t.Fatal(err)
	}
	holder2 := cmd2.Process
	waited2 := make(chan error, 1)
	go func() { waited2 <- cmd2.Wait() }()
	defer func() {
		_ = holder2.Kill()
		select {
		case <-waited2:
		default:
		}
	}()
	if err = lease2.Close(); err != nil {
		t.Fatal(err)
	}
	if held, err := lease2.ProbeHeld(); err != nil || held {
		t.Fatalf("witness after missing-fd close: held=%v err=%v", held, err)
	}
	if err = holder2.Kill(); err != nil {
		t.Fatal(err)
	}
	<-waited2
}

// TestPrepareEntryFDLayout pins the driver process boundary the P10
// model rests on: ExtraFiles[0] is the fd 3 environment pipe and
// leases follow at fd 4+, in order. No behavior change; regression
// coverage only.
func TestPrepareEntryFDLayout(t *testing.T) {
	work := t.TempDir()
	r := &Runtime{Root: work, Executable: "/bin/true"}
	leaseA, err := os.CreateTemp("", "lease-a-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(leaseA.Name())
	defer leaseA.Close()
	leaseB, err := os.CreateTemp("", "lease-b-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(leaseB.Name())
	defer leaseB.Close()
	launch, err := r.prepareEntry(context.Background(), "/bin/true", []string{"root=0"},
		[]string{"ALPHA=1", "BETA=two"}, nil, &bytes.Buffer{}, &bytes.Buffer{}, []*os.File{leaseA, leaseB})
	if err != nil {
		t.Fatal(err)
	}
	defer launch.cleanup()
	extra := launch.cmd.ExtraFiles
	if len(extra) != 3 {
		t.Fatalf("ExtraFiles holds %d entries, want 3 (env + 2 leases)", len(extra))
	}
	// fd 3 is the environment read pipe; draining it yields the exact
	// sorted-key snapshot the owner offered.
	if extra[0] != launch.read {
		t.Fatal("ExtraFiles[0] is not the environment pipe: fd 3 moved")
	}
	if _, err = launch.write.Write(launch.input); err != nil {
		t.Fatal(err)
	}
	if err = launch.write.Close(); err != nil {
		t.Fatal(err)
	}
	raw, err := io.ReadAll(extra[0])
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"ALPHA":"1","BETA":"two"}` {
		t.Fatalf("env snapshot = %q", raw)
	}
	if extra[1] != leaseA || extra[2] != leaseB {
		t.Fatal("leases out of order: fd 4 and fd 5 must carry the generation leases in order")
	}
	// The startup environment is the fixed allowlist, never inherited.
	wantEnv := map[string]bool{}
	for _, item := range launch.cmd.Env {
		key := strings.SplitN(item, "=", 2)[0]
		wantEnv[key] = true
	}
	for _, key := range []string{"HOME", "XDG_CONFIG_HOME", "TMPDIR", "PATH"} {
		if !wantEnv[key] {
			t.Fatalf("cmd.Env lacks %s: %q", key, launch.cmd.Env)
		}
	}
	if len(launch.cmd.Env) != 4 {
		t.Fatalf("cmd.Env = %q, want exactly the 4-entry allowlist", launch.cmd.Env)
	}
	if launch.cmd.Dir == "" {
		t.Fatal("child workdir unset")
	}
	workdir := launch.work
	launch.cleanup()
	if _, err = os.Stat(workdir); !os.IsNotExist(err) {
		t.Fatalf("workdir survives cleanup: %v", err)
	}
}
