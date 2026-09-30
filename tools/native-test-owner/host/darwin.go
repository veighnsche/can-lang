package host

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Resource selectors from <sys/resource.h>. The stdlib syscall package
// exposes Getrlimit but not the RLIMIT_* selectors on darwin (they
// live in x/sys, which this module does not take), so the three the
// probe needs are spelled out. A wrong selector fails the getrlimit
// call, so the probe test proves the numbers, not the comment.
const (
	rlimitFSIZE = 1 // RLIMIT_FSIZE: file size
	rlimitAS    = 5 // RLIMIT_AS: address space
	rlimitNPROC = 7 // RLIMIT_NPROC: number of processes
)

// rlimInfinity is RLIM_INFINITY on 64-bit darwin (comparisons only;
// the package never sets a limit).
const rlimInfinity = ^uint64(0) >> 1

// darwinAdapter is the selected strict adapter. It qualifies on darwin
// only; anywhere else it probes unavailable so no envelope is admitted
// on an unqualified host.
type darwinAdapter struct{}

// Darwin returns the selected strict host adapter.
func Darwin() Adapter { return darwinAdapter{} }

// Name reports the adapter identity for receipts.
func (darwinAdapter) Name() string { return "darwin-strict" }

// Probe reports capabilities with live host facts. Facts are best
// effort; unknown (zero) fails closed at fit time.
func (darwinAdapter) Probe(tmpParent string) Capabilities {
	if runtime.GOOS != "darwin" {
		return Capabilities{Available: false, Detail: "darwin-strict selected on " + runtime.GOOS}
	}
	return Capabilities{
		Available:  true,
		Mechanisms: []Mechanism{MechRefuseBeforeEffect, MechResourceLimit},
		Facts:      probeHostFacts(tmpParent),
	}
}

// probeHostFacts reads kernel facts: rlimits via getrlimit, RAM via
// sysctl hw.memsize, tmp free bytes via statfs on tmpParent.
func probeHostFacts(tmpParent string) HostFacts {
	f := HostFacts{GOOS: runtime.GOOS}
	var lim syscall.Rlimit
	if err := syscall.Getrlimit(rlimitNPROC, &lim); err == nil {
		f.NProcSoft, f.NProcHard = lim.Cur, lim.Max
	}
	if err := syscall.Getrlimit(rlimitAS, &lim); err == nil {
		f.AddrSpaceCapped = lim.Cur != rlimInfinity
	}
	if err := syscall.Getrlimit(rlimitFSIZE, &lim); err == nil {
		f.FileSizeCapped = lim.Cur != rlimInfinity
	}
	if mem, err := probeMemBytes(); err == nil {
		f.MemBytes = mem
	}
	if tmpParent != "" {
		if free, err := probeTmpFree(tmpParent); err == nil {
			f.TmpFreeBytes = free
		}
	}
	return f
}

// probeMemBytes reads hw.memsize via a bounded sysctl call.
func probeMemBytes() (uint64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := execCommand(ctx, "sysctl", "-n", "hw.memsize")
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(strings.TrimSpace(string(out)), 10, 64)
}

// probeTmpFree reports free bytes on the filesystem holding path.
func probeTmpFree(path string) (uint64, error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0, err
	}
	if st.Bavail < 0 || st.Bsize < 0 {
		return 0, fmt.Errorf("host: negative statfs result")
	}
	return uint64(st.Bavail) * uint64(st.Bsize), nil
}

// OpenScope validates the envelope and tmp parent, probes the bypass
// bound, and opens an empty charged scope with its own tmp root.
func (darwinAdapter) OpenScope(env Envelope, tmpParent string) (Scope, error) {
	if runtime.GOOS != "darwin" {
		return nil, fmt.Errorf("%w: darwin-strict selected on %s", ErrUnavailable, runtime.GOOS)
	}
	if err := env.valid(); err != nil {
		return nil, err
	}
	if err := checkTmpParent(tmpParent); err != nil {
		return nil, err
	}
	bound, err := probeTmpFree(tmpParent)
	if err != nil || bound == 0 {
		return nil, fmt.Errorf("%w: cannot state tmp bypass bound", ErrInvalid)
	}
	root, err := os.MkdirTemp(tmpParent, "scope-*")
	if err != nil {
		return nil, fmt.Errorf("host: create scope tmp root: %w", err)
	}
	if err := os.Chmod(root, 0o700); err != nil {
		os.RemoveAll(root)
		return nil, fmt.Errorf("host: chmod scope tmp root: %w", err)
	}
	return &darwinScope{
		env: env, tmpRoot: root,
		procs: make(map[uint64]ProcToken),
		// The bypass bound is the free disk probed at open: uncharged
		// bytes physically cannot exceed it, and every decision point
		// reconciles before verdict. int64 is ample (free disk on a
		// test host is far below 8 EiB); saturate defensively.
		bypassBound: saturateInt64(bound),
	}, nil
}

func saturateInt64(u uint64) int64 {
	const maxInt64 = int64(^uint64(0) >> 1)
	if u > uint64(maxInt64) {
		return maxInt64
	}
	return int64(u)
}

func checkTmpParent(dir string) error {
	if dir == "" || !filepath.IsAbs(dir) {
		return fmt.Errorf("%w: tmp parent must be an absolute path", ErrInvalid)
	}
	fi, err := os.Stat(dir)
	if err != nil || !fi.IsDir() {
		return fmt.Errorf("%w: tmp parent unavailable", ErrInvalid)
	}
	return nil
}

// darwinScope is the synchronous pre-effect ledger. The mutex guards
// every charge, release, reconcile and dispose; charges refuse at the
// ceiling with zero mediated overshoot.
type darwinScope struct {
	mu          sync.Mutex
	env         Envelope
	tmpRoot     string
	bypassBound int64
	nextID      uint64
	procs       map[uint64]ProcToken
	procsN      int
	detachedN   int
	memCharged  int64
	tmpCharged  int64
	overflow    bool
	disposed    bool
}

// Envelope reports the scope's envelope.
func (s *darwinScope) Envelope() Envelope { return s.env }

// TmpRoot reports the owned tmp root.
func (s *darwinScope) TmpRoot() string { return s.tmpRoot }

// Usage reports exact charged totals.
func (s *darwinScope) Usage() Usage {
	s.mu.Lock()
	defer s.mu.Unlock()
	return Usage{Procs: s.procsN, DetachedProcs: s.detachedN, MemBytes: s.memCharged, TmpBytes: s.tmpCharged}
}

// Overshoot states the finite bounds: zero mediated overshoot on every
// dimension, bypass bounded by probed free disk.
func (s *darwinScope) Overshoot() Overshoot {
	return Overshoot{
		MaxProcsOver: 0, MaxMemOver: 0, MaxTmpMediatedOver: 0,
		TmpBypassBound: s.bypassBound,
	}
}

// ChargeProc authorizes one spawn plus its declared memory peak before
// fork. It reconciles tmp first like every charge path, so no verdict
// is issued on unreconciled state. Detached spawns charge identically:
// the gate sits at the spawn call site, so Setsid descendants cannot
// escape it. Ceiling comparisons subtract instead of adding so absurd
// charges fail closed instead of wrapping around.
func (s *darwinScope) ChargeProc(detached bool, memBytes int64) (ProcToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.disposed {
		return ProcToken{}, ErrReleased
	}
	if _, err := s.reconcileLocked(); err != nil {
		return ProcToken{}, err
	}
	if s.overflow {
		return ProcToken{}, fmt.Errorf("%w: proc charge refused", ErrOverflow)
	}
	if memBytes < 0 {
		return ProcToken{}, fmt.Errorf("%w: negative memory charge", ErrInvalid)
	}
	if s.procsN >= s.env.CeilingProcs() || memBytes > s.env.CeilingMem()-s.memCharged {
		return ProcToken{}, fmt.Errorf("%w: proc (detached=%v, mem=%d)", ErrOverEnvelope, detached, memBytes)
	}
	s.nextID++
	t := ProcToken{id: s.nextID, detached: detached, mem: memBytes}
	s.procs[t.id] = t
	s.procsN++
	if detached {
		s.detachedN++
	}
	s.memCharged += memBytes
	return t, nil
}

// ReleaseProc releases one token. Unknown or already-released tokens
// fail: a double release must never subtract twice.
func (s *darwinScope) ReleaseProc(t ProcToken) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.disposed {
		return ErrReleased
	}
	held, ok := s.procs[t.id]
	if !ok || t.id == 0 {
		return fmt.Errorf("%w: unknown or released proc token", ErrInvalid)
	}
	delete(s.procs, t.id)
	s.procsN--
	if held.detached {
		s.detachedN--
	}
	s.memCharged -= held.mem
	return nil
}

// ChargeTmp authorizes n bytes before the write. It reconciles first:
// no admission verdict is issued on unreconciled state, so bypass bytes
// always count against the ceiling.
func (s *darwinScope) ChargeTmp(n int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.disposed {
		return ErrReleased
	}
	if n <= 0 {
		return fmt.Errorf("%w: tmp charge must be positive", ErrInvalid)
	}
	if _, err := s.reconcileLocked(); err != nil {
		return err
	}
	if s.overflow {
		return fmt.Errorf("%w: tmp charge refused", ErrOverflow)
	}
	if n > s.env.CeilingTmp()-s.tmpCharged {
		return fmt.Errorf("%w: tmp bytes=%d", ErrOverEnvelope, n)
	}
	s.tmpCharged += n
	return nil
}

// ReleaseTmp releases n charged bytes. Releasing more than charged
// fails: the ledger never goes negative.
func (s *darwinScope) ReleaseTmp(n int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.disposed {
		return ErrReleased
	}
	if n <= 0 || n > s.tmpCharged {
		return fmt.Errorf("%w: tmp release of %d against %d charged", ErrInvalid, n, s.tmpCharged)
	}
	s.tmpCharged -= n
	return nil
}

// ReconcileTmp rescans the tmp root and charges bypass bytes. Past the
// max it latches overflow: further charges refuse until Dispose.
func (s *darwinScope) ReconcileTmp() (TmpFacts, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.disposed {
		return TmpFacts{}, ErrReleased
	}
	return s.reconcileLocked()
}

// reconcileLocked walks the tmp root summing regular-file bytes without
// following symlinks, absorbs bypass bytes into the charge, and latches
// overflow past the max. Truthful accounting absorbs first and judges
// after: the charge reflects bytes on disk even in overflow.
func (s *darwinScope) reconcileLocked() (TmpFacts, error) {
	var observed int64
	var files int
	err := filepath.WalkDir(s.tmpRoot, func(_ string, d fs.DirEntry, werr error) error {
		if werr != nil {
			return werr
		}
		if d.IsDir() {
			return nil
		}
		files++
		if !d.Type().IsRegular() {
			return nil
		}
		info, ierr := d.Info()
		if ierr != nil {
			return ierr
		}
		observed += info.Size()
		return nil
	})
	if err != nil {
		return TmpFacts{}, fmt.Errorf("host: rescan tmp root: %w", err)
	}
	facts := TmpFacts{Charged: s.tmpCharged, Observed: observed, Files: files}
	if observed > s.tmpCharged {
		delta := observed - s.tmpCharged
		s.tmpCharged = observed
		facts.Charged = observed
		facts.BypassCharged = delta
	}
	if s.tmpCharged > s.env.MaxTmpBytes {
		s.overflow = true
		return facts, fmt.Errorf("%w: %d tmp bytes over max %d",
			ErrOverflow, s.tmpCharged, s.env.MaxTmpBytes)
	}
	return facts, nil
}

// Dispose removes all tmp contents and releases every charge. Release
// paths charge nothing, so disposal succeeds at full charge; the
// reserve it conceptually spends is the disposer slot held back from
// the ceiling. Dispose is idempotent; a failed removal keeps the scope
// undisposed and charged.
func (s *darwinScope) Dispose() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.disposed {
		return nil
	}
	// Reconcile for truthful books, but disposal is not gated on it:
	// overflow is exactly what disposal clears.
	facts, rerr := s.reconcileLocked()
	if rerr != nil && facts.Observed == 0 && !s.overflow {
		return rerr
	}
	entries, derr := os.ReadDir(s.tmpRoot)
	if derr != nil {
		return fmt.Errorf("host: list tmp root: %w", derr)
	}
	var remaining []string
	for _, e := range entries {
		if rerr := os.RemoveAll(filepath.Join(s.tmpRoot, e.Name())); rerr != nil {
			remaining = append(remaining, e.Name())
		}
	}
	if len(remaining) > 0 {
		return fmt.Errorf("host: %d tmp paths remain", len(remaining))
	}
	if rerr := os.Remove(s.tmpRoot); rerr != nil {
		return fmt.Errorf("host: remove tmp root: %w", rerr)
	}
	s.procs = make(map[uint64]ProcToken)
	s.procsN, s.detachedN = 0, 0
	s.memCharged, s.tmpCharged = 0, 0
	s.overflow = false
	s.disposed = true
	return nil
}
