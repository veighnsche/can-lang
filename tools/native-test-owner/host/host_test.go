package host

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/tools/native-test-owner/admission"
)

func requireDarwin(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "darwin" {
		t.Skip("darwin-strict qualification is darwin-only")
	}
}

func logChecks(t *testing.T, q *Qualification) {
	t.Helper()
	for _, c := range q.Checks {
		t.Logf("check %-28s pass=%-5v %s", c.Name, c.Pass, c.Detail)
	}
}

// TestQualifyQuickEnvelope runs the full battery on the quick envelope:
// 64 procs, 4 GiB, 512 MiB tmp admitted only where the mechanism,
// bounds, reserve and host fit are all demonstrated.
func TestQualifyQuickEnvelope(t *testing.T) {
	requireDarwin(t)
	q, err := Qualify(Darwin(), Quick(), QualifyOpts{TmpParent: t.TempDir()})
	if err != nil {
		t.Fatalf("battery did not run: %v", err)
	}
	logChecks(t, q)
	if !q.Valid() {
		t.Fatalf("quick did not qualify: %v", q.Failing())
	}
}

// TestQualifyBoundedJobEnvelope runs the full battery on the
// bounded-job envelope: 64 procs, 6 GiB, 2 GiB tmp.
func TestQualifyBoundedJobEnvelope(t *testing.T) {
	requireDarwin(t)
	q, err := Qualify(Darwin(), BoundedJob(), QualifyOpts{TmpParent: t.TempDir()})
	if err != nil {
		t.Fatalf("battery did not run: %v", err)
	}
	logChecks(t, q)
	if !q.Valid() {
		t.Fatalf("bounded-job did not qualify: %v", q.Failing())
	}
}

// TestRealDetachedChildCharged holds a real Setsid child under a
// detached token: the child is provably detached (own session group)
// and provably charged (1/1) while alive; kill plus release zeroes it.
func TestRealDetachedChildCharged(t *testing.T) {
	requireDarwin(t)
	sleep, err := exec.LookPath("sleep")
	if err != nil {
		t.Skip("sleep unavailable")
	}
	env := Envelope{Name: "detach-live", MaxProcs: 4, MaxMemBytes: 1 << 20, MaxTmpBytes: 1 << 20,
		DisposalReserve: Reserve{Procs: 1, MemBytes: 1 << 10, TmpBytes: 1 << 10}}
	s, err := Darwin().OpenScope(env, t.TempDir())
	if err != nil {
		t.Fatalf("open scope: %v", err)
	}
	defer s.Dispose()
	tok, err := s.ChargeProc(true, 4096)
	if err != nil {
		t.Fatalf("charge detached: %v", err)
	}
	if !tok.Detached() {
		t.Fatal("token lost the detached flag")
	}
	cmd := exec.Command(sleep, "30")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		t.Fatalf("start detached child: %v", err)
	}
	reaped := false
	defer func() {
		_ = cmd.Process.Kill()
		if !reaped {
			_ = cmd.Wait()
		}
	}()
	// Detached means its own session group: pgid equals its pid.
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		t.Fatalf("getpgid: %v", err)
	}
	if pgid != cmd.Process.Pid {
		t.Fatalf("child not detached: pgid %d != pid %d", pgid, cmd.Process.Pid)
	}
	if u := s.Usage(); u.Procs != 1 || u.DetachedProcs != 1 || u.MemBytes != 4096 {
		t.Fatalf("live detached child charged as %+v, want 1/1/4096", u)
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf("kill: %v", err)
	}
	if err := cmd.Wait(); err == nil {
		t.Fatal("killed child exited clean")
	}
	reaped = true
	if err := s.ReleaseProc(tok); err != nil {
		t.Fatalf("release: %v", err)
	}
	if u := s.Usage(); u != (Usage{}) {
		t.Fatalf("usage after release is %+v, want zero", u)
	}
}

// TestOverflowLatchPersists proves overflow is fail-closed until
// disposal: releases still work (books stay truthful) but no new
// charge is admitted.
func TestOverflowLatchPersists(t *testing.T) {
	requireDarwin(t)
	s, err := Darwin().OpenScope(controlEnvelope(), t.TempDir())
	if err != nil {
		t.Fatalf("open scope: %v", err)
	}
	defer s.Dispose()
	planted := filepath.Join(s.TmpRoot(), "flood.bin")
	if err := os.WriteFile(planted, make([]byte, 200), 0o600); err != nil {
		t.Fatalf("plant: %v", err)
	}
	if _, err := s.ReconcileTmp(); !errors.Is(err, ErrOverflow) {
		t.Fatalf("reconcile returned %v, want ErrOverflow", err)
	}
	// Releases keep working in overflow; charges do not recover.
	if err := os.Remove(planted); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if err := s.ReleaseTmp(200); err != nil {
		t.Fatalf("release in overflow: %v", err)
	}
	if err := s.ChargeTmp(1); !errors.Is(err, ErrOverflow) {
		t.Fatalf("charge after release returned %v, want latched ErrOverflow", err)
	}
	if err := s.Dispose(); err != nil {
		t.Fatalf("dispose: %v", err)
	}
}

// TestChargeProcReconcilesFirst proves proc charges never issue a
// verdict on unreconciled state: a planted bypass flood the caller
// never reconciled still refuses the proc charge via the fresh rescan.
func TestChargeProcReconcilesFirst(t *testing.T) {
	requireDarwin(t)
	s, err := Darwin().OpenScope(controlEnvelope(), t.TempDir())
	if err != nil {
		t.Fatalf("open scope: %v", err)
	}
	defer s.Dispose()
	if err := os.WriteFile(filepath.Join(s.TmpRoot(), "flood.bin"), make([]byte, 200), 0o600); err != nil {
		t.Fatalf("plant: %v", err)
	}
	// No explicit ReconcileTmp: the proc charge must still refuse.
	if _, err := s.ChargeProc(false, 0); !errors.Is(err, ErrOverflow) {
		t.Fatalf("proc charge on unreconciled flood returned %v, want ErrOverflow", err)
	}
	if u := s.Usage(); u.Procs != 0 {
		t.Fatalf("refused proc charge had effect: %+v", u)
	}
}

// TestAbsurdChargesFailClosed proves ceiling comparisons cannot wrap
// around: near-max charges refuse even against a partially filled
// ledger instead of overflowing into admission.
func TestAbsurdChargesFailClosed(t *testing.T) {
	requireDarwin(t)
	const huge = int64(^uint64(0) >> 1)
	// The mem envelope leaves proc headroom so the absurd mem charge
	// exercises the memory comparison, not the proc ceiling.
	s, err := Darwin().OpenScope(memControlEnvelope(), t.TempDir())
	if err != nil {
		t.Fatalf("open scope: %v", err)
	}
	defer s.Dispose()
	if _, err := s.ChargeProc(false, 96); err != nil {
		t.Fatalf("fill mem ceiling: %v", err)
	}
	if _, err := s.ChargeProc(false, huge); !errors.Is(err, ErrOverEnvelope) {
		t.Fatalf("absurd mem charge returned %v, want ErrOverEnvelope", err)
	}
	if err := s.ChargeTmp(96); err != nil {
		t.Fatalf("fill tmp ceiling: %v", err)
	}
	if err := s.ChargeTmp(huge); !errors.Is(err, ErrOverEnvelope) {
		t.Fatalf("absurd tmp charge returned %v, want ErrOverEnvelope", err)
	}
	if u := s.Usage(); u.MemBytes != 96 || u.TmpBytes != 96 || u.Procs != 1 {
		t.Fatalf("refused absurd charges had effect: %+v", u)
	}
}

// TestTokenDiscipline proves releases are exact: unknown, zero and
// double-released tokens fail, and tmp releases cannot exceed charges.
func TestTokenDiscipline(t *testing.T) {
	requireDarwin(t)
	s, err := Darwin().OpenScope(controlEnvelope(), t.TempDir())
	if err != nil {
		t.Fatalf("open scope: %v", err)
	}
	defer s.Dispose()
	tok, err := s.ChargeProc(false, 8)
	if err != nil {
		t.Fatalf("charge: %v", err)
	}
	if err := s.ReleaseProc(ProcToken{}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("zero token release returned %v, want ErrInvalid", err)
	}
	if err := s.ReleaseProc(tok); err != nil {
		t.Fatalf("release: %v", err)
	}
	if err := s.ReleaseProc(tok); !errors.Is(err, ErrInvalid) {
		t.Fatalf("double release returned %v, want ErrInvalid", err)
	}
	if u := s.Usage(); u.MemBytes != 0 {
		t.Fatalf("double release corrupted books: %+v", u)
	}
	if err := s.ChargeTmp(10); err != nil {
		t.Fatalf("charge tmp: %v", err)
	}
	if err := s.ReleaseTmp(11); !errors.Is(err, ErrInvalid) {
		t.Fatalf("over-release returned %v, want ErrInvalid", err)
	}
	if err := s.ChargeTmp(0); !errors.Is(err, ErrInvalid) {
		t.Fatalf("zero charge returned %v, want ErrInvalid", err)
	}
}

// TestDisposeIdempotent proves post-disposal charges refuse while
// Dispose itself joins.
func TestDisposeIdempotent(t *testing.T) {
	requireDarwin(t)
	s, err := Darwin().OpenScope(controlEnvelope(), t.TempDir())
	if err != nil {
		t.Fatalf("open scope: %v", err)
	}
	if err := s.Dispose(); err != nil {
		t.Fatalf("dispose: %v", err)
	}
	if err := s.Dispose(); err != nil {
		t.Fatalf("second dispose returned %v, want nil", err)
	}
	if _, err := s.ChargeProc(false, 0); !errors.Is(err, ErrReleased) {
		t.Fatalf("charge after dispose returned %v, want ErrReleased", err)
	}
}

// TestEnvelopeValidation proves bad envelopes and tmp parents never
// open a scope.
func TestEnvelopeValidation(t *testing.T) {
	bad := []Envelope{
		{Name: "", MaxProcs: 1, MaxMemBytes: 1, MaxTmpBytes: 1, DisposalReserve: Reserve{1, 1, 1}},
		{Name: "zero", MaxProcs: 0, MaxMemBytes: 1, MaxTmpBytes: 1, DisposalReserve: Reserve{1, 1, 1}},
		{Name: "nores", MaxProcs: 2, MaxMemBytes: 128, MaxTmpBytes: 128},
		{Name: "full", MaxProcs: 1, MaxMemBytes: 1, MaxTmpBytes: 1, DisposalReserve: Reserve{1, 1, 1}},
	}
	for _, env := range bad {
		if _, err := Darwin().OpenScope(env, t.TempDir()); !errors.Is(err, ErrInvalid) {
			t.Fatalf("envelope %+v opened: %v, want ErrInvalid", env, err)
		}
		if _, err := Qualify(Darwin(), env, QualifyOpts{TmpParent: t.TempDir()}); !errors.Is(err, ErrInvalid) {
			t.Fatalf("envelope %+v qualified: %v, want ErrInvalid", env, err)
		}
	}
	if _, err := Darwin().OpenScope(controlEnvelope(), "relative/path"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("relative tmp parent opened: %v", err)
	}
	if _, err := Qualify(Darwin(), Quick(), QualifyOpts{}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("empty tmp parent qualified: %v", err)
	}
	if _, err := Qualify(nil, Quick(), QualifyOpts{TmpParent: t.TempDir()}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("nil adapter qualified: %v", err)
	}
}

// TestAdmitBindsP12Gate proves admission holds the P12 lane across the
// bound scope: the lane reads held while bound, contenders refuse, and
// Complete releases with disposal done first.
func TestAdmitBindsP12Gate(t *testing.T) {
	requireDarwin(t)
	q, err := Qualify(Darwin(), Quick(), QualifyOpts{TmpParent: t.TempDir()})
	if err != nil || !q.Valid() {
		t.Fatalf("quick did not qualify: %v %v", err, q.Failing())
	}
	lockDir := t.TempDir()
	h, err := admission.OpenHost(lockDir)
	if err != nil {
		t.Fatalf("open host gate: %v", err)
	}
	root := t.TempDir()
	req := admission.Request{Demand: admission.Demand{Live: true},
		Capability: admission.Capability{MaxHandles: 64, MaxPending: 16, MaxBytes: 1 << 30},
		Budget:     30 * time.Second, Body: time.Second, CleanupReserve: time.Second}
	b, err := Admit(h, root, req, q, t.TempDir())
	if err != nil {
		t.Fatalf("admit: %v", err)
	}
	held, err := admission.ProbeLane(lockDir, admission.LaneLive)
	if err != nil || !held {
		t.Fatalf("lane reads held=%v err=%v while bound", held, err)
	}
	// A contender on the same gate refuses; the failure holds nothing new.
	if _, err := Admit(h, root, req, q, t.TempDir()); err == nil {
		t.Fatal("contending admit succeeded while lane held")
	}
	if _, err := b.Scope().ChargeProc(false, 1<<20); err != nil {
		t.Fatalf("charge in binding: %v", err)
	}
	if err := b.Complete(); err != nil {
		t.Fatalf("complete: %v", err)
	}
	if held, _ := admission.ProbeLane(lockDir, admission.LaneLive); held {
		t.Fatal("lane still held after Complete")
	}
	// Abort path: Release frees the lane too.
	b2, err := Admit(h, root, req, q, t.TempDir())
	if err != nil {
		t.Fatalf("second admit: %v", err)
	}
	b2.Release()
	if held, _ := admission.ProbeLane(lockDir, admission.LaneLive); held {
		t.Fatal("lane still held after Release")
	}
}

// flipAdapter delegates scope mechanics to darwin-strict but serves
// mutable probe facts, proving Admit gates on fresh facts.
type flipAdapter struct {
	mu    sync.Mutex
	avail bool
	facts HostFacts
}

func (f *flipAdapter) Name() string { return "flip-test" }

func (f *flipAdapter) Probe(_ string) Capabilities {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.avail {
		return Capabilities{Available: false, Detail: "flipped unavailable"}
	}
	return Capabilities{Available: true,
		Mechanisms: []Mechanism{MechRefuseBeforeEffect, MechResourceLimit}, Facts: f.facts}
}

func (f *flipAdapter) OpenScope(env Envelope, tmpParent string) (Scope, error) {
	return Darwin().OpenScope(env, tmpParent)
}

// TestAdmitReprobesFreshFacts proves admission refuses when the host
// drifts after qualification: flipped-unavailable blocks with
// ErrUnavailable, shrunken facts block with ErrNoFit, and neither
// holds a lane.
func TestAdmitReprobesFreshFacts(t *testing.T) {
	requireDarwin(t)
	facts := Darwin().Probe(t.TempDir()).Facts
	if facts.NProcSoft == 0 || facts.MemBytes == 0 || facts.TmpFreeBytes == 0 {
		t.Skip("host facts incomplete; nothing to flip from")
	}
	f := &flipAdapter{avail: true, facts: facts}
	q, err := Qualify(f, Quick(), QualifyOpts{TmpParent: t.TempDir()})
	if err != nil || !q.Valid() {
		t.Fatalf("flip adapter did not qualify while available: %v %v", err, q.Failing())
	}
	lockDir := t.TempDir()
	h, err := admission.OpenHost(lockDir)
	if err != nil {
		t.Fatalf("open host gate: %v", err)
	}
	root := t.TempDir()
	req := admission.Request{Demand: admission.Demand{Live: true},
		Capability: admission.Capability{MaxHandles: 64, MaxPending: 16, MaxBytes: 1 << 30},
		Budget:     30 * time.Second, Body: time.Second, CleanupReserve: time.Second}

	f.mu.Lock()
	f.avail = false
	f.mu.Unlock()
	if _, err := Admit(h, root, req, q, t.TempDir()); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("admit on drifted-unavailable returned %v, want ErrUnavailable", err)
	}
	if held, _ := admission.ProbeLane(lockDir, admission.LaneLive); held {
		t.Fatal("refused admit holds a lane")
	}

	f.mu.Lock()
	f.avail = true
	f.facts = facts
	f.facts.NProcSoft = 1 // the host can no longer fit 64 procs
	f.mu.Unlock()
	if _, err := Admit(h, root, req, q, t.TempDir()); !errors.Is(err, ErrNoFit) {
		t.Fatalf("admit on shrunken host returned %v, want ErrNoFit", err)
	}
	if held, _ := admission.ProbeLane(lockDir, admission.LaneLive); held {
		t.Fatal("refused admit holds a lane")
	}

	if _, err := Admit(h, root, req, nil, t.TempDir()); !errors.Is(err, ErrNotQualified) {
		t.Fatalf("admit without qualification returned %v, want ErrNotQualified", err)
	}
}

// TestConcurrentScopeHammer proves the scope is safe for concurrent use
// (race detector) and the books balance exactly under contention.
func TestConcurrentScopeHammer(t *testing.T) {
	requireDarwin(t)
	env := Envelope{Name: "hammer", MaxProcs: 33, MaxMemBytes: 1 << 20, MaxTmpBytes: 1 << 20,
		DisposalReserve: Reserve{Procs: 1, MemBytes: 1, TmpBytes: 1}}
	s, err := Darwin().OpenScope(env, t.TempDir())
	if err != nil {
		t.Fatalf("open scope: %v", err)
	}
	defer s.Dispose()
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				tok, cerr := s.ChargeProc(i%2 == 0, 8)
				if cerr != nil {
					t.Errorf("charge proc: %v", cerr)
					return
				}
				if cerr := s.ChargeTmp(8); cerr != nil {
					t.Errorf("charge tmp: %v", cerr)
					return
				}
				_ = s.Usage()
				if cerr := s.ReleaseTmp(8); cerr != nil {
					t.Errorf("release tmp: %v", cerr)
					return
				}
				if cerr := s.ReleaseProc(tok); cerr != nil {
					t.Errorf("release proc: %v", cerr)
					return
				}
			}
		}()
	}
	wg.Wait()
	if u := s.Usage(); u != (Usage{}) {
		t.Fatalf("usage after hammer is %+v, want zero", u)
	}
}

// TestReasonMapping pins the refusal vocabulary.
func TestReasonMapping(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{ErrUnavailable, ReasonUnavailable},
		{ErrNotQualified, ReasonNotQualified},
		{ErrOverEnvelope, ReasonOverEnvelope},
		{ErrOverflow, ReasonOverflow},
		{ErrNoFit, ReasonNoFit},
		{errors.New("boom"), ReasonDenied},
	}
	for _, c := range cases {
		if got := Reason(c.err); got != c.want {
			t.Fatalf("Reason(%v) = %q, want %q", c.err, got, c.want)
		}
	}
}
