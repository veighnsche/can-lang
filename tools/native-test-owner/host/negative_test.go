package host

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"
)

// generousFakeFacts isolates negative controls to behavior: fit always
// passes, so failures prove enforcement flaws, never arithmetic.
func generousFakeFacts() HostFacts {
	return HostFacts{
		GOOS:            runtime.GOOS,
		NProcSoft:       1 << 40,
		NProcHard:       1 << 40,
		AddrSpaceCapped: true,
		FileSizeCapped:  true,
		MemBytes:        1 << 60,
		TmpFreeBytes:    1 << 60,
	}
}

// fakeScopeBase is the shared scaffold for must-fail scopes: real tmp
// roots, real disposal, but admit-everything charges and no rescan.
type fakeScopeBase struct {
	mu       sync.Mutex
	env      Envelope
	tmpRoot  string
	nextID   uint64
	procs    map[uint64]ProcToken
	procsN   int
	detachN  int
	mem      int64
	tmp      int64
	disposed bool
}

func openFakeBase(env Envelope, tmpParent string) (*fakeScopeBase, error) {
	if err := env.valid(); err != nil {
		return nil, err
	}
	if err := checkTmpParent(tmpParent); err != nil {
		return nil, err
	}
	root, err := os.MkdirTemp(tmpParent, "fake-*")
	if err != nil {
		return nil, err
	}
	return &fakeScopeBase{env: env, tmpRoot: root, procs: make(map[uint64]ProcToken)}, nil
}

func (b *fakeScopeBase) dispose() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.disposed {
		return nil
	}
	entries, err := os.ReadDir(b.tmpRoot)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if err := os.RemoveAll(b.tmpRoot + "/" + e.Name()); err != nil {
			return err
		}
	}
	if err := os.Remove(b.tmpRoot); err != nil {
		return err
	}
	b.disposed = true
	return nil
}

// pollKillAdapter admits every charge and only asynchronously "kills"
// the excess: the over-limit effect exists before enforcement reacts.
type pollKillAdapter struct{}

func (pollKillAdapter) Name() string { return "poll-kill-test" }

func (pollKillAdapter) Probe(_ string) Capabilities {
	return Capabilities{Available: true, Mechanisms: []Mechanism{MechPollKill}, Facts: generousFakeFacts()}
}

func (pollKillAdapter) OpenScope(env Envelope, tmpParent string) (Scope, error) {
	b, err := openFakeBase(env, tmpParent)
	if err != nil {
		return nil, err
	}
	return &pollKillScope{base: b}, nil
}

type pollKillScope struct {
	base   *fakeScopeBase
	killed int
}

func (s *pollKillScope) Envelope() Envelope { return s.base.env }
func (s *pollKillScope) TmpRoot() string    { return s.base.tmpRoot }

func (s *pollKillScope) Usage() Usage {
	s.base.mu.Lock()
	defer s.base.mu.Unlock()
	return Usage{Procs: s.base.procsN, DetachedProcs: s.base.detachN, MemBytes: s.base.mem, TmpBytes: s.base.tmp}
}

// Overshoot is honestly unbounded: kill latency after observation has
// no finite bound.
func (s *pollKillScope) Overshoot() Overshoot { return Overshoot{Unbounded: true} }

func (s *pollKillScope) ChargeProc(detached bool, memBytes int64) (ProcToken, error) {
	s.base.mu.Lock()
	over := s.base.procsN+1 > s.base.env.CeilingProcs()
	s.base.nextID++
	t := ProcToken{id: s.base.nextID, detached: detached, mem: memBytes}
	s.base.procs[t.id] = t
	s.base.procsN++
	if detached {
		s.base.detachN++
	}
	s.base.mem += memBytes
	s.base.mu.Unlock()
	if over {
		// The poll-and-kill reaction: async, after the effect.
		time.AfterFunc(5*time.Millisecond, func() {
			s.base.mu.Lock()
			s.killed++
			s.base.mu.Unlock()
		})
	}
	return t, nil
}

func (s *pollKillScope) ReleaseProc(t ProcToken) error {
	s.base.mu.Lock()
	defer s.base.mu.Unlock()
	held, ok := s.base.procs[t.id]
	if !ok || t.id == 0 {
		return fmt.Errorf("%w: unknown token", ErrInvalid)
	}
	delete(s.base.procs, t.id)
	s.base.procsN--
	if held.detached {
		s.base.detachN--
	}
	s.base.mem -= held.mem
	return nil
}

func (s *pollKillScope) ChargeTmp(n int64) error {
	s.base.mu.Lock()
	defer s.base.mu.Unlock()
	s.base.tmp += n
	return nil
}

func (s *pollKillScope) ReleaseTmp(n int64) error {
	s.base.mu.Lock()
	defer s.base.mu.Unlock()
	s.base.tmp -= n
	return nil
}

// ReconcileTmp never rescans: uncharged bytes escape, like a sampler
// blind between ticks.
func (s *pollKillScope) ReconcileTmp() (TmpFacts, error) {
	s.base.mu.Lock()
	defer s.base.mu.Unlock()
	return TmpFacts{Charged: s.base.tmp, Observed: s.base.tmp}, nil
}

func (s *pollKillScope) Dispose() error { return s.base.dispose() }

// samplingAdapter admits everything and reports peaks later: overshoot
// between samples is unbounded by construction.
type samplingAdapter struct{}

func (samplingAdapter) Name() string { return "sampling-test" }

func (samplingAdapter) Probe(_ string) Capabilities {
	return Capabilities{Available: true, Mechanisms: []Mechanism{MechSampledPeak}, Facts: generousFakeFacts()}
}

func (samplingAdapter) OpenScope(env Envelope, tmpParent string) (Scope, error) {
	b, err := openFakeBase(env, tmpParent)
	if err != nil {
		return nil, err
	}
	return &samplingScope{base: b}, nil
}

type samplingScope struct {
	base *fakeScopeBase
	peak int64
}

func (s *samplingScope) Envelope() Envelope { return s.base.env }
func (s *samplingScope) TmpRoot() string    { return s.base.tmpRoot }

func (s *samplingScope) Usage() Usage {
	s.base.mu.Lock()
	defer s.base.mu.Unlock()
	return Usage{Procs: s.base.procsN, DetachedProcs: s.base.detachN, MemBytes: s.base.mem, TmpBytes: s.base.tmp}
}

func (s *samplingScope) Overshoot() Overshoot { return Overshoot{Unbounded: true} }

func (s *samplingScope) ChargeProc(detached bool, memBytes int64) (ProcToken, error) {
	s.base.mu.Lock()
	defer s.base.mu.Unlock()
	s.base.nextID++
	t := ProcToken{id: s.base.nextID, detached: detached, mem: memBytes}
	s.base.procs[t.id] = t
	s.base.procsN++
	if detached {
		s.base.detachN++
	}
	s.base.mem += memBytes
	if s.base.mem > s.peak {
		s.peak = s.base.mem // the "sampled peak": observed, never enforced
	}
	return t, nil
}

func (s *samplingScope) ReleaseProc(t ProcToken) error {
	s.base.mu.Lock()
	defer s.base.mu.Unlock()
	held, ok := s.base.procs[t.id]
	if !ok || t.id == 0 {
		return fmt.Errorf("%w: unknown token", ErrInvalid)
	}
	delete(s.base.procs, t.id)
	s.base.procsN--
	if held.detached {
		s.base.detachN--
	}
	s.base.mem -= held.mem
	return nil
}

func (s *samplingScope) ChargeTmp(n int64) error {
	s.base.mu.Lock()
	defer s.base.mu.Unlock()
	s.base.tmp += n
	return nil
}

func (s *samplingScope) ReleaseTmp(n int64) error {
	s.base.mu.Lock()
	defer s.base.mu.Unlock()
	s.base.tmp -= n
	return nil
}

func (s *samplingScope) ReconcileTmp() (TmpFacts, error) {
	s.base.mu.Lock()
	defer s.base.mu.Unlock()
	return TmpFacts{Charged: s.base.tmp, Observed: s.base.tmp}, nil
}

func (s *samplingScope) Dispose() error { return s.base.dispose() }

// lyingAdapter claims refuse-before-effect and finite bounds but
// behaves like admit-everything: it proves behavior decides, labels do
// not. It also omits the resource-limit claim, so kernel checks fail.
type lyingAdapter struct{}

func (lyingAdapter) Name() string { return "lying-test" }

func (lyingAdapter) Probe(_ string) Capabilities {
	return Capabilities{Available: true, Mechanisms: []Mechanism{MechRefuseBeforeEffect}, Facts: generousFakeFacts()}
}

func (lyingAdapter) OpenScope(env Envelope, tmpParent string) (Scope, error) {
	b, err := openFakeBase(env, tmpParent)
	if err != nil {
		return nil, err
	}
	return &lyingScope{base: b}, nil
}

type lyingScope struct{ base *fakeScopeBase }

func (s *lyingScope) Envelope() Envelope { return s.base.env }
func (s *lyingScope) TmpRoot() string    { return s.base.tmpRoot }

func (s *lyingScope) Usage() Usage {
	s.base.mu.Lock()
	defer s.base.mu.Unlock()
	return Usage{Procs: s.base.procsN, DetachedProcs: s.base.detachN, MemBytes: s.base.mem, TmpBytes: s.base.tmp}
}

// Overshoot lies too: finite zeros while charges exceed the ceiling.
func (s *lyingScope) Overshoot() Overshoot { return Overshoot{TmpBypassBound: 1 << 30} }

func (s *lyingScope) ChargeProc(detached bool, memBytes int64) (ProcToken, error) {
	s.base.mu.Lock()
	defer s.base.mu.Unlock()
	s.base.nextID++
	t := ProcToken{id: s.base.nextID, detached: detached, mem: memBytes}
	s.base.procs[t.id] = t
	s.base.procsN++
	if detached {
		s.base.detachN++
	}
	s.base.mem += memBytes
	return t, nil
}

func (s *lyingScope) ReleaseProc(t ProcToken) error {
	s.base.mu.Lock()
	defer s.base.mu.Unlock()
	held, ok := s.base.procs[t.id]
	if !ok || t.id == 0 {
		return fmt.Errorf("%w: unknown token", ErrInvalid)
	}
	delete(s.base.procs, t.id)
	s.base.procsN--
	if held.detached {
		s.base.detachN--
	}
	s.base.mem -= held.mem
	return nil
}

func (s *lyingScope) ChargeTmp(n int64) error {
	s.base.mu.Lock()
	defer s.base.mu.Unlock()
	s.base.tmp += n
	return nil
}

func (s *lyingScope) ReleaseTmp(n int64) error {
	s.base.mu.Lock()
	defer s.base.mu.Unlock()
	s.base.tmp -= n
	return nil
}

func (s *lyingScope) ReconcileTmp() (TmpFacts, error) {
	s.base.mu.Lock()
	defer s.base.mu.Unlock()
	return TmpFacts{Charged: s.base.tmp, Observed: s.base.tmp}, nil
}

func (s *lyingScope) Dispose() error { return s.base.dispose() }

func mustFail(t *testing.T, q *Qualification, names ...string) {
	t.Helper()
	if q.Valid() {
		t.Fatalf("adapter %s qualified, want failure", q.AdapterName)
	}
	for _, n := range names {
		c, ok := q.Check(n)
		if !ok {
			t.Fatalf("check %s missing from record", n)
		}
		if c.Pass {
			t.Fatalf("check %s passed, want failure (%s)", n, c.Detail)
		}
	}
	t.Logf("%s failing: %v", q.AdapterName, q.Failing())
}

// TestPollKillCannotQualify proves a poll-and-kill adapter fails: it
// claims a non-qualifying mechanism, admits over every ceiling
// (reaction comes after the effect), states unbounded overshoot,
// misses bypass bytes, and claims no kernel backstop.
func TestPollKillCannotQualify(t *testing.T) {
	q, err := Qualify(pollKillAdapter{}, Quick(), QualifyOpts{TmpParent: t.TempDir()})
	if err != nil {
		t.Fatalf("battery did not run: %v", err)
	}
	mustFail(t, q, CheckMechanism, CheckRefuseProc, CheckRefuseMem, CheckRefuseTmp,
		CheckOvershootFinite, CheckDetachedCharged, CheckBypassCaught, CheckOverflowClosed,
		CheckKernelNproc, CheckKernelFsize)
}

// TestSamplingCannotQualify proves a sampled-peak adapter fails: same
// behavioral shape (admit-then-observe, unbounded overshoot, blind
// reconcile) under a different claim.
func TestSamplingCannotQualify(t *testing.T) {
	q, err := Qualify(samplingAdapter{}, BoundedJob(), QualifyOpts{TmpParent: t.TempDir()})
	if err != nil {
		t.Fatalf("battery did not run: %v", err)
	}
	mustFail(t, q, CheckMechanism, CheckRefuseProc, CheckRefuseMem, CheckRefuseTmp,
		CheckOvershootFinite, CheckBypassCaught, CheckOverflowClosed)
}

// TestLyingAdapterFailsBehaviorally proves labels do not qualify: the
// liar passes mechanism-declared and overshoot-finite on paper, yet
// fails every behavioral check that exercises the boundary.
func TestLyingAdapterFailsBehaviorally(t *testing.T) {
	q, err := Qualify(lyingAdapter{}, Quick(), QualifyOpts{TmpParent: t.TempDir()})
	if err != nil {
		t.Fatalf("battery did not run: %v", err)
	}
	if q.Valid() {
		t.Fatal("lying adapter qualified, want failure")
	}
	if c, _ := q.Check(CheckMechanism); !c.Pass {
		t.Fatalf("mechanism-declared failed (%s): the liar must pass labels and fail behavior", c.Detail)
	}
	mustFail(t, q, CheckRefuseProc, CheckRefuseMem, CheckRefuseTmp,
		CheckDetachedCharged, CheckBypassCaught, CheckOverflowClosed)
}

// TestUnavailableHostBlocked proves the unavailable host refuses both
// envelopes: no qualification, no scope, and Admit holds no lane.
func TestUnavailableHostBlocked(t *testing.T) {
	for _, env := range []Envelope{Quick(), BoundedJob()} {
		q, err := Qualify(Unavailable("test: no mechanism"), env, QualifyOpts{TmpParent: t.TempDir()})
		if err != nil {
			t.Fatalf("%s: battery did not run: %v", env.Name, err)
		}
		if q.Valid() {
			t.Fatalf("%s: unavailable host qualified", env.Name)
		}
		if c, _ := q.Check("open-refused"); !c.Pass {
			t.Fatalf("%s: OpenScope was not refused with ErrUnavailable", env.Name)
		}
		if _, err := Unavailable("x").OpenScope(env, t.TempDir()); !errors.Is(err, ErrUnavailable) {
			t.Fatalf("%s: OpenScope returned %v, want ErrUnavailable", env.Name, err)
		}
	}
}
