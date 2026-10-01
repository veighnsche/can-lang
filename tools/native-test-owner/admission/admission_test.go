package admission

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeClock is a thread-safe manual clock for deterministic deadline
// controls: no test waits on wall time to reach a verdict.
type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{t: time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)}
}

func (c *fakeClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

func finiteCapability() Capability {
	return Capability{MaxHandles: 64, MaxPending: 16, MaxBytes: 1 << 30}
}

func liveReq() Request {
	return Request{Demand: Demand{Live: true}, Capability: finiteCapability(), Budget: time.Minute, Body: time.Second, CleanupReserve: time.Second}
}

func produceReq() Request {
	return Request{Demand: Demand{Produce: true}, Capability: finiteCapability(), Budget: time.Minute, Body: time.Second, CleanupReserve: time.Second}
}

// fakeDisk is a thread-safe manual free-disk probe for deterministic
// floor controls: tests set available bytes (or a probe error) without
// touching a real filesystem. Every probed path is recorded.
type fakeDisk struct {
	mu    sync.Mutex
	avail uint64
	err   error
	paths []string
}

func newFakeDisk(avail uint64) *fakeDisk {
	return &fakeDisk{avail: avail}
}

func (f *fakeDisk) probe(path string) (uint64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.paths = append(f.paths, path)
	return f.avail, f.err
}

func (f *fakeDisk) set(avail uint64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.avail, f.err = avail, nil
}

func (f *fakeDisk) setErr(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.err = err
}

func (f *fakeDisk) probedPaths() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.paths...)
}

func openHost(t *testing.T, dir string) *Host {
	t.Helper()
	h, err := OpenHostWithClockAndFreeDisk(dir, newFakeClock().now, newFakeDisk(MinFreeDiskBytes*2).probe)
	if err != nil {
		t.Fatalf("OpenHost: %v", err)
	}
	return h
}

func mustProbe(t *testing.T, dir string, lane Lane) bool {
	t.Helper()
	held, err := ProbeLane(dir, lane)
	if err != nil {
		t.Fatalf("ProbeLane(%s): %v", lane, err)
	}
	return held
}

func TestAdmitReleaseRoundTrip(t *testing.T) {
	dir := t.TempDir()
	root := t.TempDir()
	clk := newFakeClock()
	h, err := OpenHostWithClockAndFreeDisk(dir, clk.now, newFakeDisk(MinFreeDiskBytes*2).probe)
	if err != nil {
		t.Fatalf("OpenHost: %v", err)
	}
	g, err := h.Admit(root, liveReq())
	if err != nil {
		t.Fatalf("Admit: %v", err)
	}
	if g.Root() != root {
		t.Errorf("Root() = %q, want %q", g.Root(), root)
	}
	if got := g.Lanes(); len(got) != 1 || got[0] != LaneLive {
		t.Errorf("Lanes() = %v, want [live-case]", got)
	}
	if !g.Start().Equal(clk.now()) {
		t.Errorf("Start() = %v, want %v", g.Start(), clk.now())
	}
	if want := g.Start().Add(time.Minute); !g.Deadline().Equal(want) {
		t.Errorf("Deadline() = %v, want %v", g.Deadline(), want)
	}
	if g.Remaining() != time.Minute {
		t.Errorf("Remaining() = %v, want %v", g.Remaining(), time.Minute)
	}
	if !mustProbe(t, dir, LaneLive) {
		t.Errorf("lane witness reports free while held")
	}
	if err := g.Complete(); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if mustProbe(t, dir, LaneLive) {
		t.Errorf("lane witness reports held after Complete")
	}
	// The slot is free again: a second admission succeeds.
	g2, err := h.Admit(root, liveReq())
	if err != nil {
		t.Fatalf("second Admit: %v", err)
	}
	g2.Release()
}

func TestCompleteTwiceAndReleaseIdempotent(t *testing.T) {
	h := openHost(t, t.TempDir())
	g, err := h.Admit(t.TempDir(), liveReq())
	if err != nil {
		t.Fatalf("Admit: %v", err)
	}
	if err := g.Complete(); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if err := g.Complete(); !errors.Is(err, ErrReleased) {
		t.Errorf("second Complete: got %v, want ErrReleased", err)
	}
	// Release is idempotent and safe after Complete.
	g.Release()
	g.Release()
	if mustProbe(t, h.Dir(), LaneLive) {
		t.Errorf("lane held after release")
	}
}

// TestTwoRootsContendOneLiveSlot proves the gate is host-wide: two
// repository roots over one host lock directory share one live-case
// slot, and the loser sees a named lane-busy refusal.
func TestTwoRootsContendOneLiveSlot(t *testing.T) {
	hostDir := t.TempDir()
	rootA := t.TempDir()
	rootB := t.TempDir()
	hostA := openHost(t, hostDir)
	hostB := openHost(t, hostDir)

	grantA, err := hostA.Admit(rootA, liveReq())
	if err != nil {
		t.Fatalf("root A Admit: %v", err)
	}
	defer grantA.Release()

	if _, err := hostB.Admit(rootB, liveReq()); !errors.Is(err, ErrLaneBusy) {
		t.Fatalf("root B Admit: got %v, want ErrLaneBusy", err)
	} else if Reason(err) != ReasonLaneBusy {
		t.Fatalf("root B Reason: got %q, want %q", Reason(err), ReasonLaneBusy)
	}
	if !mustProbe(t, hostDir, LaneLive) {
		t.Fatalf("lane witness reports free while root A holds it")
	}

	grantA.Release()
	grantB, err := hostB.Admit(rootB, liveReq())
	if err != nil {
		t.Fatalf("root B Admit after release: %v", err)
	}
	defer grantB.Release()
	if _, err := hostA.Admit(rootA, liveReq()); !errors.Is(err, ErrLaneBusy) {
		t.Fatalf("root A re-Admit: got %v, want ErrLaneBusy", err)
	}
}

// TestHelperLaneHolder is the cross-process holder: re-executed as a
// child test binary, it admits one lane, drops the ready file, holds
// until stdin closes, then releases and exits.
func TestHelperLaneHolder(t *testing.T) {
	dir := os.Getenv("ADMISSION_HELPER_DIR")
	if dir == "" {
		return
	}
	lane := Lane(os.Getenv("ADMISSION_HELPER_LANE"))
	ready := os.Getenv("ADMISSION_HELPER_READY")
	h, err := OpenHost(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "helper: open host: %v\n", err)
		os.Exit(2)
	}
	demand := Demand{}
	switch lane {
	case LaneLive:
		demand.Live = true
	case LaneVerify:
		demand.Verify = true
	case LaneProduce:
		demand.Produce = true
	default:
		fmt.Fprintf(os.Stderr, "helper: unknown lane %q\n", lane)
		os.Exit(2)
	}
	g, err := h.Admit(filepath.Join(dir, "root-child"), Request{
		Demand: demand, Capability: finiteCapability(), Budget: 2 * time.Minute, Body: time.Second, CleanupReserve: time.Second,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "helper: admit: %v\n", err)
		os.Exit(2)
	}
	if err := os.WriteFile(ready, []byte("ready"), 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "helper: ready file: %v\n", err)
		g.Release()
		os.Exit(2)
	}
	// Hold until the parent closes stdin, then release and exit 0.
	_, _ = io.Copy(io.Discard, os.Stdin)
	g.Release()
}

// TestTwoRootsCrossProcessGate proves kernel arbitration: a separate
// process holding the live-case slot blocks this process with
// ErrLaneBusy, and release in the child frees the slot here. The
// process-local registry cannot help across processes, so only flock
// explains the refusal.
func TestTwoRootsCrossProcessGate(t *testing.T) {
	hostDir := t.TempDir()
	root := t.TempDir()
	ready := filepath.Join(hostDir, "child.ready")

	cmd := exec.Command(os.Args[0], "-test.run", "^TestHelperLaneHolder$", "-test.count=1")
	cmd.Env = append(os.Environ(),
		"ADMISSION_HELPER_DIR="+hostDir,
		"ADMISSION_HELPER_LANE="+string(LaneLive),
		"ADMISSION_HELPER_READY="+ready,
	)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("StdinPipe: %v", err)
	}
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Start(); err != nil {
		t.Fatalf("start holder: %v", err)
	}
	waited := make(chan error, 1)
	go func() { waited <- cmd.Wait() }()
	// reaped is set by the main path once it consumes waited; the
	// cleanup kills and reaps only when the main path did not. It is
	// written before the test function returns and read in cleanup, so
	// no synchronization is needed beyond that ordering.
	reaped := false
	t.Cleanup(func() {
		if reaped {
			return
		}
		_ = cmd.Process.Kill()
		<-waited
	})

	// Bounded wait for the child to hold the slot (ready file drop).
	deadline := time.Now().Add(15 * time.Second)
	for {
		if _, serr := os.Stat(ready); serr == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("holder never became ready: %s", output.String())
		}
		time.Sleep(5 * time.Millisecond)
	}

	h := openHost(t, hostDir)
	if _, err := h.Admit(root, liveReq()); !errors.Is(err, ErrLaneBusy) {
		t.Fatalf("Admit against child-held slot: got %v, want ErrLaneBusy", err)
	} else if Reason(err) != ReasonLaneBusy {
		t.Fatalf("Reason: got %q, want %q", Reason(err), ReasonLaneBusy)
	}
	if held, err := ProbeLane(hostDir, LaneLive); err != nil || !held {
		t.Fatalf("ProbeLane = (%v, %v), want (true, nil)", held, err)
	}

	// Closing stdin releases the child; its exit frees the slot here.
	_ = stdin.Close()
	select {
	case werr := <-waited:
		reaped = true
		if werr != nil {
			t.Fatalf("holder exit: %v: %s", werr, output.String())
		}
	case <-time.After(15 * time.Second):
		t.Fatalf("holder did not exit: %s", output.String())
	}
	g, err := h.Admit(root, liveReq())
	if err != nil {
		t.Fatalf("Admit after child release: %v", err)
	}
	g.Release()
}

// TestNestedProducerDemandFailsFast proves a nested demand from inside
// a held reservation fails fast with a named reason instead of
// deadlocking: the inner Admit returns (proved by the bounded select),
// the outer reservation is unaffected, and the lane stays admittable
// after release.
func TestNestedProducerDemandFailsFast(t *testing.T) {
	hostDir := t.TempDir()
	root := t.TempDir()
	h := openHost(t, hostDir)

	outer, err := h.Admit(root, liveReq())
	if err != nil {
		t.Fatalf("outer Admit: %v", err)
	}
	defer outer.Release()

	type outcome struct {
		grant *Grant
		err   error
	}
	done := make(chan outcome, 1)
	go func() {
		g, err := h.Admit(root, produceReq())
		done <- outcome{g, err}
	}()
	select {
	case r := <-done:
		if r.err == nil {
			r.grant.Release()
			t.Fatalf("nested producer demand admitted while live held")
		}
		if !errors.Is(r.err, ErrNestedDemand) {
			t.Fatalf("nested demand: got %v, want ErrNestedDemand", r.err)
		}
		if Reason(r.err) != ReasonNestedDemand {
			t.Fatalf("nested demand Reason: got %q, want %q", Reason(r.err), ReasonNestedDemand)
		}
	case <-time.After(10 * time.Second):
		t.Fatalf("nested producer demand blocked: deadlock")
	}

	// The outer reservation still holds its lane.
	if !mustProbe(t, hostDir, LaneLive) {
		t.Fatalf("outer lane lost after nested refusal")
	}
	if mustProbe(t, hostDir, LaneProduce) {
		t.Fatalf("refused nested demand left the producer lane held")
	}

	outer.Release()
	inner, err := h.Admit(root, produceReq())
	if err != nil {
		t.Fatalf("Admit after outer release: %v", err)
	}
	inner.Release()
}

// TestDeadlineAtSuccessIsFailure proves the deadline is absolute and
// monotonic: completing exactly at the deadline fails even though the
// body succeeded, completing past it fails, and completing just before
// it succeeds. The fake clock makes every boundary exact.
func TestDeadlineAtSuccessIsFailure(t *testing.T) {
	hostDir := t.TempDir()
	root := t.TempDir()

	admit := func(clk *fakeClock) *Grant {
		t.Helper()
		h, err := OpenHostWithClockAndFreeDisk(hostDir, clk.now, newFakeDisk(MinFreeDiskBytes*2).probe)
		if err != nil {
			t.Fatalf("OpenHost: %v", err)
		}
		g, err := h.Admit(root, liveReq())
		if err != nil {
			t.Fatalf("Admit: %v", err)
		}
		return g
	}

	// Exactly at the deadline: failure.
	clk := newFakeClock()
	g := admit(clk)
	clk.advance(time.Minute)
	if err := g.Complete(); !errors.Is(err, ErrDeadlineExceeded) {
		t.Errorf("Complete at deadline: got %v, want ErrDeadlineExceeded", err)
	} else if Reason(err) != ReasonDeadlineExceeded {
		t.Errorf("Reason: got %q, want %q", Reason(err), ReasonDeadlineExceeded)
	}

	// Past the deadline: failure.
	clk = newFakeClock()
	g = admit(clk)
	clk.advance(time.Minute + time.Second)
	if err := g.Complete(); !errors.Is(err, ErrDeadlineExceeded) {
		t.Errorf("Complete past deadline: got %v, want ErrDeadlineExceeded", err)
	}

	// Just before the deadline: success.
	clk = newFakeClock()
	g = admit(clk)
	clk.advance(time.Minute - time.Nanosecond)
	if err := g.Complete(); err != nil {
		t.Errorf("Complete before deadline: %v", err)
	}

	if mustProbe(t, hostDir, LaneLive) {
		t.Errorf("lane held after deadline controls")
	}
}

// TestBodyPlusCleanupFit proves no case starts when body plus cleanup
// cannot fit: over budget refuses, exactly equal refuses (completion
// must precede the deadline), and strictly inside admits. Every
// refusal holds nothing.
func TestBodyPlusCleanupFit(t *testing.T) {
	hostDir := t.TempDir()
	root := t.TempDir()
	h := openHost(t, hostDir)

	cases := []struct {
		name    string
		body    time.Duration
		cleanup time.Duration
		budget  time.Duration
		wantErr error
	}{
		{"over-budget", 50 * time.Second, 50 * time.Second, time.Minute, ErrNoFit},
		{"exactly-equal", 30 * time.Second, 30 * time.Second, time.Minute, ErrNoFit},
		{"cleanup-alone-fills-budget", 0, time.Minute, time.Minute, ErrNoFit},
		{"strictly-inside", 30 * time.Second, 29 * time.Second, time.Minute, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := Request{
				Demand:         Demand{Live: true},
				Capability:     finiteCapability(),
				Budget:         tc.budget,
				Body:           tc.body,
				CleanupReserve: tc.cleanup,
			}
			g, err := h.Admit(root, req)
			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("Admit: %v", err)
				}
				g.Release()
				return
			}
			if !errors.Is(err, tc.wantErr) {
				if err == nil {
					g.Release()
				}
				t.Fatalf("Admit: got %v, want %v", err, tc.wantErr)
			}
			if Reason(err) != ReasonNoFit {
				t.Fatalf("Reason: got %q, want %q", Reason(err), ReasonNoFit)
			}
			// The refusal holds nothing: the lane is free and a
			// fitting case admits immediately.
			if mustProbe(t, hostDir, LaneLive) {
				t.Fatalf("refused admission left the lane held")
			}
			fit, ferr := h.Admit(root, liveReq())
			if ferr != nil {
				t.Fatalf("Admit after refusal: %v", ferr)
			}
			fit.Release()
		})
	}
}

// TestAtomicDeclaredPeakReservation proves a multi-lane demand is
// all-or-nothing: when one lane is busy the demand fails and the other
// lane stays free for an immediate solo admission.
func TestAtomicDeclaredPeakReservation(t *testing.T) {
	hostDir := t.TempDir()
	root := t.TempDir()
	hostA := openHost(t, hostDir)
	hostB := openHost(t, hostDir)

	prod, err := hostB.Admit(root, produceReq())
	if err != nil {
		t.Fatalf("producer Admit: %v", err)
	}
	defer prod.Release()

	peak := Request{
		Demand:         Demand{Live: true, Produce: true},
		Capability:     finiteCapability(),
		Budget:         time.Minute,
		Body:           time.Second,
		CleanupReserve: time.Second,
	}
	if _, err := hostA.Admit(root, peak); !errors.Is(err, ErrLaneBusy) {
		t.Fatalf("peak Admit: got %v, want ErrLaneBusy", err)
	} else if !strings.Contains(err.Error(), string(LaneProduce)) {
		t.Fatalf("peak Admit error %q does not name the busy lane", err)
	}
	// Atomicity: the live lane was not partially taken.
	if mustProbe(t, hostDir, LaneLive) {
		t.Fatalf("failed peak left the live lane held")
	}
	solo, err := hostA.Admit(root, liveReq())
	if err != nil {
		t.Fatalf("solo live Admit after failed peak: %v", err)
	}
	defer solo.Release()
}

// TestConcurrentContendersMutualExclusion runs two rounds of eight
// contenders (two Host handles, one slot) behind a start barrier. Each
// round admits exactly one winner; every goroutine joins via WaitGroup.
func TestConcurrentContendersMutualExclusion(t *testing.T) {
	hostDir := t.TempDir()
	clk := newFakeClock()
	disk := newFakeDisk(MinFreeDiskBytes * 2)
	hostA, err := OpenHostWithClockAndFreeDisk(hostDir, clk.now, disk.probe)
	if err != nil {
		t.Fatalf("OpenHost A: %v", err)
	}
	hostB, err := OpenHostWithClockAndFreeDisk(hostDir, clk.now, disk.probe)
	if err != nil {
		t.Fatalf("OpenHost B: %v", err)
	}
	roots := []string{t.TempDir(), t.TempDir()}

	for round := 0; round < 2; round++ {
		const contenders = 8
		start := make(chan struct{})
		// release closes once every contender has attempted, so the
		// winner holds the slot across all attempts and late
		// contenders still observe contention.
		release := make(chan struct{})
		var wg sync.WaitGroup
		var mu sync.Mutex
		wins, inside, maxInside, attempted := 0, 0, 0, 0
		for i := 0; i < contenders; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				<-start
				h := hostA
				if i%2 == 1 {
					h = hostB
				}
				g, err := h.Admit(roots[i%2], liveReq())
				mu.Lock()
				attempted++
				if attempted == contenders {
					close(release)
				}
				mu.Unlock()
				if err != nil {
					// Cross-handle losers see lane-busy; a
					// same-handle racer behind an in-flight
					// demand sees nested-demand. Both are
					// named contention refusals.
					if !errors.Is(err, ErrLaneBusy) && !errors.Is(err, ErrNestedDemand) {
						t.Errorf("round %d contender %d: got %v, want contention refusal or success", round, i, err)
					}
					return
				}
				mu.Lock()
				wins++
				inside++
				if inside > maxInside {
					maxInside = inside
				}
				mu.Unlock()
				<-release
				mu.Lock()
				inside--
				mu.Unlock()
				if cerr := g.Complete(); cerr != nil {
					t.Errorf("round %d contender %d Complete: %v", round, i, cerr)
				}
			}(i)
		}
		close(start)
		wg.Wait()
		if wins != 1 {
			t.Fatalf("round %d: %d winners, want exactly 1", round, wins)
		}
		if maxInside != 1 {
			t.Fatalf("round %d: max concurrent holders %d, want 1", round, maxInside)
		}
	}
	if mustProbe(t, hostDir, LaneLive) {
		t.Fatalf("lane held after contention rounds")
	}
}

func TestProbeLaneWitness(t *testing.T) {
	dir := t.TempDir()
	// A lane never admitted proves free without creating the file.
	if mustProbe(t, dir, LaneVerify) {
		t.Fatalf("never-admitted lane reports held")
	}
	if _, err := os.Stat(filepath.Join(dir, "offline-verify.lock")); !os.IsNotExist(err) {
		t.Fatalf("probe created the lane file")
	}
	h := openHost(t, dir)
	g, err := h.Admit(t.TempDir(), Request{
		Demand: Demand{Verify: true}, Capability: finiteCapability(), Budget: time.Minute, Body: time.Second, CleanupReserve: time.Second,
	})
	if err != nil {
		t.Fatalf("Admit: %v", err)
	}
	if !mustProbe(t, dir, LaneVerify) {
		t.Fatalf("held lane reports free")
	}
	// Other lanes are independent.
	if mustProbe(t, dir, LaneLive) || mustProbe(t, dir, LaneProduce) {
		t.Fatalf("idle lane reports held")
	}
	g.Release()
	if mustProbe(t, dir, LaneVerify) {
		t.Fatalf("released lane reports held")
	}
	if _, err := ProbeLane(dir, Lane("bogus")); !errors.Is(err, ErrInvalid) {
		t.Fatalf("ProbeLane(bogus): got %v, want ErrInvalid", err)
	}
}

func TestValidation(t *testing.T) {
	h := openHost(t, t.TempDir())
	root := t.TempDir()
	bad := []struct {
		name string
		root string
		req  Request
	}{
		{"empty-root", "", liveReq()},
		{"relative-root", "rel/root", liveReq()},
		{"empty-demand", root, Request{Capability: finiteCapability(), Budget: time.Minute, Body: time.Second, CleanupReserve: time.Second}},
		{"zero-budget", root, Request{Demand: Demand{Live: true}, Capability: finiteCapability(), Body: time.Second, CleanupReserve: time.Second}},
		{"over-max-budget", root, Request{Demand: Demand{Live: true}, Capability: finiteCapability(), Budget: MaxBudget + time.Second, Body: time.Second, CleanupReserve: time.Second}},
		{"zero-cleanup", root, Request{Demand: Demand{Live: true}, Capability: finiteCapability(), Budget: time.Minute, Body: time.Second}},
		{"negative-body", root, Request{Demand: Demand{Live: true}, Capability: finiteCapability(), Budget: time.Minute, Body: -time.Second, CleanupReserve: time.Second}},
	}
	for _, tc := range bad {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := h.Admit(tc.root, tc.req); !errors.Is(err, ErrInvalid) {
				t.Fatalf("Admit: got %v, want ErrInvalid", err)
			}
		})
	}
	if _, err := OpenHost(""); !errors.Is(err, ErrInvalid) {
		t.Fatalf("OpenHost(\"\"): got %v, want ErrInvalid", err)
	}
	if _, err := ProbeLane("", LaneLive); !errors.Is(err, ErrInvalid) {
		t.Fatalf("ProbeLane(\"\"): got %v, want ErrInvalid", err)
	}
}

func TestReasonMapping(t *testing.T) {
	for err, want := range map[error]string{
		ErrLaneBusy:         ReasonLaneBusy,
		ErrNestedDemand:     ReasonNestedDemand,
		ErrNoFit:            ReasonNoFit,
		ErrDeadlineExceeded: ReasonDeadlineExceeded,
		ErrBelowFloor:       ReasonBelowFloor,
		ErrInvalid:          ReasonDenied,
		ErrReleased:         ReasonDenied,
		errors.New("other"): ReasonDenied,
	} {
		if got := Reason(err); got != want {
			t.Errorf("Reason(%v) = %q, want %q", err, got, want)
		}
	}
}

func TestDemandString(t *testing.T) {
	d := Demand{Produce: true, Live: true}
	if got, want := d.String(), "live-case+build-producer"; got != want {
		t.Errorf("Demand.String() = %q, want %q", got, want)
	}
	if got := (Demand{}).String(); got != "" {
		t.Errorf("empty Demand.String() = %q, want \"\"", got)
	}
}

// TestBelowFloorRefusesAdmitBeforeEffect proves the 10 GiB admission
// floor: below-floor disk refuses Admit with a named reason and holds no
// lane (every lane probes free, and a later above-floor Admit succeeds on
// the same Host). Exactly at the floor admits: "below" is strict.
func TestBelowFloorRefusesAdmitBeforeEffect(t *testing.T) {
	hostDir := t.TempDir()
	root := t.TempDir()
	disk := newFakeDisk(MinFreeDiskBytes - 1)
	h, err := OpenHostWithClockAndFreeDisk(hostDir, newFakeClock().now, disk.probe)
	if err != nil {
		t.Fatalf("OpenHost: %v", err)
	}

	if _, err := h.Admit(root, liveReq()); !errors.Is(err, ErrBelowFloor) {
		t.Fatalf("below-floor Admit: got %v, want ErrBelowFloor", err)
	} else if Reason(err) != ReasonBelowFloor {
		t.Fatalf("below-floor Reason: got %q, want %q", Reason(err), ReasonBelowFloor)
	}
	for _, l := range []Lane{LaneLive, LaneVerify, LaneProduce} {
		if mustProbe(t, hostDir, l) {
			t.Fatalf("refused admission left lane %s held", l)
		}
	}
	// The probe targeted the gate filesystem.
	for _, p := range disk.probedPaths() {
		if p != h.Dir() {
			t.Fatalf("probe path = %q, want gate dir %q", p, h.Dir())
		}
	}

	// A failed probe refuses the same way: admission fails closed.
	disk.setErr(errors.New("statfs: I/O error"))
	if _, err := h.Admit(root, liveReq()); !errors.Is(err, ErrBelowFloor) {
		t.Fatalf("probe-error Admit: got %v, want ErrBelowFloor", err)
	}
	if mustProbe(t, hostDir, LaneLive) {
		t.Fatalf("probe-error refusal left the lane held")
	}

	// Exactly at the floor admits.
	disk.set(MinFreeDiskBytes)
	g, err := h.Admit(root, liveReq())
	if err != nil {
		t.Fatalf("at-floor Admit: %v", err)
	}
	g.Release()
}

// TestFloorRecheckBeforeEachAllocation proves every allocation rechecks
// the floor: disk that drops after Admit refuses the next allocation,
// and restored disk admits again. The deadline stays absolute for
// allocations, and a refused allocation leaves the Grant held for abort
// handling (the lane is still held; Release frees it).
func TestFloorRecheckBeforeEachAllocation(t *testing.T) {
	hostDir := t.TempDir()
	root := t.TempDir()
	clk := newFakeClock()
	disk := newFakeDisk(MinFreeDiskBytes * 2)
	h, err := OpenHostWithClockAndFreeDisk(hostDir, clk.now, disk.probe)
	if err != nil {
		t.Fatalf("OpenHost: %v", err)
	}
	g, err := h.Admit(root, liveReq())
	if err != nil {
		t.Fatalf("Admit: %v", err)
	}
	defer g.Release()

	if err := g.Allocate(); err != nil {
		t.Fatalf("above-floor Allocate: %v", err)
	}
	disk.set(MinFreeDiskBytes - 1)
	if err := g.Allocate(); !errors.Is(err, ErrBelowFloor) {
		t.Fatalf("below-floor Allocate: got %v, want ErrBelowFloor", err)
	} else if Reason(err) != ReasonBelowFloor {
		t.Fatalf("below-floor Allocate Reason: got %q, want %q", Reason(err), ReasonBelowFloor)
	}
	// The refused allocation is not a verdict: the Grant still holds.
	if !mustProbe(t, hostDir, LaneLive) {
		t.Fatalf("lane lost after refused allocation")
	}
	disk.set(MinFreeDiskBytes * 2)
	if err := g.Allocate(); err != nil {
		t.Fatalf("restored-floor Allocate: %v", err)
	}

	// Allocations obey the absolute deadline too.
	clk.advance(time.Minute)
	if err := g.Allocate(); !errors.Is(err, ErrDeadlineExceeded) {
		t.Fatalf("past-deadline Allocate: got %v, want ErrDeadlineExceeded", err)
	}
}

// TestAllocateOnFinishedGrant proves Allocate on a finished Grant reports
// released instead of probing: completion first, then the checkpoint.
func TestAllocateOnFinishedGrant(t *testing.T) {
	h := openHost(t, t.TempDir())
	g, err := h.Admit(t.TempDir(), liveReq())
	if err != nil {
		t.Fatalf("Admit: %v", err)
	}
	if err := g.Complete(); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if err := g.Allocate(); !errors.Is(err, ErrReleased) {
		t.Fatalf("Allocate after Complete: got %v, want ErrReleased", err)
	}
}

// TestCapabilityLimitsMustBeFinite proves omitted and unbounded capability
// limits refuse before effect: every refusal holds no lane, and only the
// all-positive declaration admits.
func TestCapabilityLimitsMustBeFinite(t *testing.T) {
	hostDir := t.TempDir()
	root := t.TempDir()
	h := openHost(t, hostDir)

	cases := []struct {
		name string
		caps Capability
	}{
		{"omitted", Capability{}},
		{"zero-handles", Capability{MaxHandles: 0, MaxPending: 16, MaxBytes: 1 << 30}},
		{"zero-pending", Capability{MaxHandles: 64, MaxPending: 0, MaxBytes: 1 << 30}},
		{"zero-bytes", Capability{MaxHandles: 64, MaxPending: 16, MaxBytes: 0}},
		{"unbounded-handles", Capability{MaxHandles: -1, MaxPending: 16, MaxBytes: 1 << 30}},
		{"unbounded-pending", Capability{MaxHandles: 64, MaxPending: -1, MaxBytes: 1 << 30}},
		{"unbounded-bytes", Capability{MaxHandles: 64, MaxPending: 16, MaxBytes: -1}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := liveReq()
			req.Capability = tc.caps
			if _, err := h.Admit(root, req); !errors.Is(err, ErrInvalid) {
				t.Fatalf("Admit: got %v, want ErrInvalid", err)
			}
			if mustProbe(t, hostDir, LaneLive) {
				t.Fatalf("refused admission left the lane held")
			}
		})
	}

	g, err := h.Admit(root, liveReq())
	if err != nil {
		t.Fatalf("finite-limit Admit: %v", err)
	}
	if got := g.Capability(); got != finiteCapability() {
		t.Errorf("Capability() = %+v, want %+v", got, finiteCapability())
	}
	g.Release()
}

// TestAdmitAllocateCompletePositive is the end-to-end admission control:
// finite limits plus above-floor disk admit, allocate twice and complete
// before the deadline, freeing the lane.
func TestAdmitAllocateCompletePositive(t *testing.T) {
	hostDir := t.TempDir()
	root := t.TempDir()
	clk := newFakeClock()
	disk := newFakeDisk(MinFreeDiskBytes * 2)
	h, err := OpenHostWithClockAndFreeDisk(hostDir, clk.now, disk.probe)
	if err != nil {
		t.Fatalf("OpenHost: %v", err)
	}
	g, err := h.Admit(root, liveReq())
	if err != nil {
		t.Fatalf("Admit: %v", err)
	}
	if err := g.Allocate(); err != nil {
		g.Release()
		t.Fatalf("first Allocate: %v", err)
	}
	clk.advance(time.Second)
	if err := g.Allocate(); err != nil {
		g.Release()
		t.Fatalf("second Allocate: %v", err)
	}
	if err := g.Complete(); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if mustProbe(t, hostDir, LaneLive) {
		t.Fatalf("lane held after Complete")
	}
}

// TestProductionFreeDiskProbe exercises the real Statfs probe against an
// isolated directory. It asserts only that the probe runs without error;
// the value depends on the machine and carries no admission verdict here.
func TestProductionFreeDiskProbe(t *testing.T) {
	dir := t.TempDir()
	avail, err := statfsFreeDisk(dir)
	if err != nil {
		t.Fatalf("statfsFreeDisk(%s): %v", dir, err)
	}
	t.Logf("statfsFreeDisk(%s) = %d bytes", dir, avail)
	if _, err := statfsFreeDisk(filepath.Join(dir, "missing")); err == nil {
		t.Fatalf("statfsFreeDisk(missing path): got nil error, want a Statfs failure")
	}
}
