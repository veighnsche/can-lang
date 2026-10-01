package admission

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Lanes are the three host-wide capacity slots. Each lane holds at most
// one reservation across the whole host, whatever repository root asked.
type Lane string

const (
	LaneLive    Lane = "live-case"
	LaneVerify  Lane = "offline-verify"
	LaneProduce Lane = "build-producer"
)

// laneOrder is the global acquisition order. Admit takes demanded lanes
// in this order and releases partial takes in reverse; the order is
// fixed so two atomic demands can never deadlock against each other.
var laneOrder = []Lane{LaneLive, LaneVerify, LaneProduce}

// MaxBudget caps one admission's budget. Every field is finite; nothing
// here means unlimited.
const MaxBudget = 30 * time.Minute

// MinFreeDiskBytes is the lifecycle available-disk admission floor: 10 GiB.
// Admit refuses when the gate filesystem holds less, and every Allocate
// rechecks it. The floor is a guard, not a reservation: it never holds
// bytes against other programs, it only refuses to start (or continue)
// when the host is already too full.
const MinFreeDiskBytes = 10 << 30

func validLane(l Lane) bool {
	return l == LaneLive || l == LaneVerify || l == LaneProduce
}

// Demand is the declared peak: the set of lanes one run needs at once.
// The zero Demand asks for nothing and is rejected as invalid.
type Demand struct {
	Live    bool
	Verify  bool
	Produce bool
}

// lanes returns the demanded lanes in global acquisition order.
func (d Demand) lanes() []Lane {
	var out []Lane
	for _, l := range laneOrder {
		switch l {
		case LaneLive:
			if d.Live {
				out = append(out, l)
			}
		case LaneVerify:
			if d.Verify {
				out = append(out, l)
			}
		case LaneProduce:
			if d.Produce {
				out = append(out, l)
			}
		}
	}
	return out
}

// String renders the declared peak in lane order for receipts.
func (d Demand) String() string {
	lanes := d.lanes()
	parts := make([]string, len(lanes))
	for i, l := range lanes {
		parts[i] = string(l)
	}
	return strings.Join(parts, "+")
}

// Capability declares the run's finite capability ceilings: open handles,
// in-flight pending operations and scratch bytes. Every field must be
// positive; there is no unlimited spelling — zero omits a limit and
// negative claims an unbounded one, and both are rejected. Admission
// records the declared ceilings on the Grant; policing live consumption
// against them is strict host enforcement (P13), which this gate does not
// claim.
type Capability struct {
	MaxHandles int
	MaxPending int
	MaxBytes   int64
}

func (c Capability) valid() bool {
	return c.MaxHandles > 0 && c.MaxPending > 0 && c.MaxBytes > 0
}

// FreeDiskFunc reports available bytes for unprivileged callers on the
// filesystem holding path. Production uses statfsFreeDisk; tests inject a
// fake.
type FreeDiskFunc func(path string) (uint64, error)

// statfsFreeDisk is the production free-disk probe: Bavail * Bsize via
// Statfs, saturated at ^uint64(0). A Statfs failure is reported and the
// caller refuses: admission fails closed.
func statfsFreeDisk(path string) (uint64, error) {
	var s syscall.Statfs_t
	if err := syscall.Statfs(path, &s); err != nil {
		return 0, err
	}
	if s.Bsize <= 0 || s.Bavail <= 0 {
		return 0, nil
	}
	const maxUint64 = ^uint64(0)
	bs := uint64(s.Bsize)
	ba := uint64(s.Bavail)
	if ba > maxUint64/bs {
		return maxUint64, nil
	}
	return ba * bs, nil
}

// Request is one admission demand with its charged costs.
type Request struct {
	// Demand is the declared peak: every lane the run needs at once.
	Demand Demand
	// Capability declares the run's finite ceilings. Every field must be
	// positive: a request with an omitted or unbounded limit never
	// starts.
	Capability Capability
	// Budget is the total absolute budget measured from admission.
	Budget time.Duration
	// Body is the declared body cost.
	Body time.Duration
	// CleanupReserve is the cleanup cost charged against every
	// admission. It must be positive: a run with no cleanup budget
	// never starts.
	CleanupReserve time.Duration
}

// validate checks the request and returns the demanded lanes in global
// acquisition order. Fit is strict — body plus cleanup must land
// strictly inside the budget — because completion must precede the
// deadline: reaching the deadline is failure, so a case whose costs
// merely equal the budget could never succeed.
func (r Request) validate() ([]Lane, error) {
	lanes := r.Demand.lanes()
	if len(lanes) == 0 {
		return nil, fmt.Errorf("%w: demand names no lane", ErrInvalid)
	}
	if !r.Capability.valid() {
		return nil, fmt.Errorf("%w: capability limits must be finite: every field positive", ErrInvalid)
	}
	if r.Budget <= 0 || r.Budget > MaxBudget {
		return nil, fmt.Errorf("%w: budget must be within (0, %v]", ErrInvalid, MaxBudget)
	}
	if r.CleanupReserve <= 0 {
		return nil, fmt.Errorf("%w: cleanup reserve must be positive", ErrInvalid)
	}
	if r.Body < 0 {
		return nil, fmt.Errorf("%w: body cost must not be negative", ErrInvalid)
	}
	// Overflow-free strict fit: cleanup >= budget, or body >= the
	// remainder, means the case can never complete before the deadline.
	if r.CleanupReserve >= r.Budget || r.Body >= r.Budget-r.CleanupReserve {
		return nil, fmt.Errorf("%w: body %v plus cleanup %v cannot fit budget %v",
			ErrNoFit, r.Body, r.CleanupReserve, r.Budget)
	}
	return lanes, nil
}

// registry shadows the kernel locks inside this process, keyed by
// absolute lock path. Same-process contenders (two Host handles over one
// directory) arbitrate here; cross-process contenders arbitrate in the
// kernel. The registry is always taken before any Host mutex and never
// held across a blocking call.
var (
	registryMu sync.Mutex
	registry   = make(map[string]*Host)
)

// Host is one handle on the host-wide gate. The zero Host is unusable;
// construct with OpenHost, OpenHostWithClock or
// OpenHostWithClockAndFreeDisk. A Host is safe for concurrent use.
type Host struct {
	dir      string
	now      func() time.Time
	freeDisk FreeDiskFunc

	mu   sync.Mutex
	held map[Lane]*os.File
}

// OpenHost opens the gate on the host lock directory dir, creating it.
// In production dir is a well-known host path shared by every
// repository root; tests pass an isolated t.TempDir. The clock is
// time.Now and the free-disk probe is the production Statfs probe.
func OpenHost(dir string) (*Host, error) {
	return OpenHostWithClock(dir, time.Now)
}

// OpenHostWithClock opens the gate with an explicit clock. A nil clock
// selects time.Now. Tests inject a fake clock for deterministic
// deadline controls. The free-disk probe is the production Statfs probe;
// use OpenHostWithClockAndFreeDisk to inject a fake probe.
func OpenHostWithClock(dir string, now func() time.Time) (*Host, error) {
	return OpenHostWithClockAndFreeDisk(dir, now, nil)
}

// OpenHostWithClockAndFreeDisk opens the gate with an explicit clock and
// an explicit free-disk probe. A nil clock selects time.Now; a nil probe
// selects the production Statfs probe, so the floor is never silently
// skipped. Tests inject a fake clock for deterministic deadline controls
// and a fake probe for deterministic floor controls.
func OpenHostWithClockAndFreeDisk(dir string, now func() time.Time, freeDisk FreeDiskFunc) (*Host, error) {
	if dir == "" {
		return nil, fmt.Errorf("%w: empty host lock directory", ErrInvalid)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("admission: create host lock directory: %w", err)
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("admission: resolve host lock directory: %w", err)
	}
	if now == nil {
		now = time.Now
	}
	if freeDisk == nil {
		freeDisk = statfsFreeDisk
	}
	return &Host{dir: abs, now: now, freeDisk: freeDisk, held: make(map[Lane]*os.File)}, nil
}

// Dir reports the host lock directory.
func (h *Host) Dir() string { return h.dir }

func (h *Host) lanePath(l Lane) string {
	return filepath.Join(h.dir, string(l)+".lock")
}

// checkFloor enforces the admission floor on the gate filesystem: below
// MinFreeDiskBytes refuses with ErrBelowFloor, and a probe failure
// refuses the same way — admission fails closed. Callers invoke it with
// no admission lock held, so the probe (a Statfs syscall in production,
// a test fake under test control) can never join a lock cycle.
func (h *Host) checkFloor() error {
	avail, err := h.freeDisk(h.dir)
	if err != nil {
		return fmt.Errorf("%w: probe %s: %v", ErrBelowFloor, h.dir, err)
	}
	if avail < MinFreeDiskBytes {
		return fmt.Errorf("%w: %d available on %s, floor %d", ErrBelowFloor, avail, h.dir, MinFreeDiskBytes)
	}
	return nil
}

// unmark releases registry and host bookkeeping for lanes, keeping only
// entries still owned by h. Callers close the kernel lock files first so
// no window opens where the registry reads free while flock is held.
func (h *Host) unmark(lanes []Lane) {
	registryMu.Lock()
	h.mu.Lock()
	for _, l := range lanes {
		if p := h.lanePath(l); registry[p] == h {
			delete(registry, p)
		}
		delete(h.held, l)
	}
	h.mu.Unlock()
	registryMu.Unlock()
}

// Admit atomically reserves the declared peak for one repository root.
// root is recorded on the Grant for evidence; arbitration is host-wide,
// so the root never widens capacity. Either every demanded lane is held
// or none is: a conflict releases partial takes in reverse order and
// fails with ErrLaneBusy naming the lane, and a demand issued while
// this Host already holds a reservation fails fast with ErrNestedDemand
// before touching the kernel. A case whose body plus cleanup cannot fit
// strictly inside its budget never starts (ErrNoFit) and holds nothing.
// Before any lane is touched, Admit enforces the admission floor:
// below-floor disk (or a failed probe) refuses with ErrBelowFloor and
// holds nothing, and a request with an omitted or unbounded capability
// limit is rejected as invalid during validation.
func (h *Host) Admit(root string, req Request) (*Grant, error) {
	if root == "" || !filepath.IsAbs(root) {
		return nil, fmt.Errorf("%w: repository root must be an absolute path", ErrInvalid)
	}
	lanes, err := req.validate()
	if err != nil {
		return nil, err
	}
	start := h.now()
	deadline := start.Add(req.Budget)

	// Admission floor before any effect: no registry mark, no kernel
	// lock and no held lane exists yet, so a refusal here cannot strand
	// state. Production places the host lock directory on the same
	// filesystem as run scratch so the gate sees allocation pressure.
	if err := h.checkFloor(); err != nil {
		return nil, err
	}

	// Registry first, host second, kernel last; nothing here blocks.
	registryMu.Lock()
	h.mu.Lock()
	if len(h.held) > 0 {
		h.mu.Unlock()
		registryMu.Unlock()
		return nil, fmt.Errorf("%w: host already holds a reservation", ErrNestedDemand)
	}
	for _, l := range lanes {
		if holder, taken := registry[h.lanePath(l)]; taken {
			h.mu.Unlock()
			registryMu.Unlock()
			if holder == h {
				return nil, fmt.Errorf("%w: lane %s already demanded", ErrNestedDemand, l)
			}
			return nil, fmt.Errorf("%w: lane %s", ErrLaneBusy, l)
		}
	}
	for _, l := range lanes {
		registry[h.lanePath(l)] = h
	}
	h.mu.Unlock()
	registryMu.Unlock()

	files := make(map[Lane]*os.File, len(lanes))
	acquired := make([]Lane, 0, len(lanes))
	fail := func(format string, args ...any) (*Grant, error) {
		for i := len(acquired) - 1; i >= 0; i-- {
			files[acquired[i]].Close()
		}
		h.unmark(lanes)
		return nil, fmt.Errorf(format, args...)
	}
	for _, l := range lanes {
		f, oerr := os.OpenFile(h.lanePath(l), os.O_RDWR|os.O_CREATE, 0o600)
		if oerr != nil {
			return fail("admission: open lane %s: %w", l, oerr)
		}
		if lerr := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); lerr != nil {
			f.Close()
			if errors.Is(lerr, syscall.EWOULDBLOCK) {
				return fail("%w: lane %s", ErrLaneBusy, l)
			}
			return fail("admission: lock lane %s: %w", l, lerr)
		}
		files[l] = f
		acquired = append(acquired, l)
	}

	h.mu.Lock()
	for l, f := range files {
		h.held[l] = f
	}
	h.mu.Unlock()
	return &Grant{
		h: h, root: root, demand: req.Demand, lanes: lanes, files: files,
		start: start, deadline: deadline,
		budget: req.Budget, body: req.Body, cleanup: req.CleanupReserve,
		caps: req.Capability,
	}, nil
}

// Grant is one held declared-peak reservation. Every Grant must end in
// exactly one verdict — Complete — or in Release on abort paths; a
// Grant is safe for concurrent use but its verdict is single-shot.
type Grant struct {
	h      *Host
	root   string
	demand Demand
	lanes  []Lane
	files  map[Lane]*os.File

	start    time.Time
	deadline time.Time
	budget   time.Duration
	body     time.Duration
	cleanup  time.Duration
	caps     Capability

	mu   sync.Mutex
	done bool
}

// Root reports the admitted repository root.
func (g *Grant) Root() string { return g.root }

// Demand reports the declared peak.
func (g *Grant) Demand() Demand { return g.demand }

// Lanes reports the held lanes in acquisition order.
func (g *Grant) Lanes() []Lane { return append([]Lane(nil), g.lanes...) }

// Start reports the admission instant.
func (g *Grant) Start() time.Time { return g.start }

// Deadline reports the absolute deadline: start plus budget.
func (g *Grant) Deadline() time.Time { return g.deadline }

// Budget reports the admitted budget.
func (g *Grant) Budget() time.Duration { return g.budget }

// Body reports the declared body cost.
func (g *Grant) Body() time.Duration { return g.body }

// CleanupReserve reports the charged cleanup reserve.
func (g *Grant) CleanupReserve() time.Duration { return g.cleanup }

// Capability reports the declared finite capability ceilings.
func (g *Grant) Capability() Capability { return g.caps }

// Remaining reports deadline minus the Host clock now. It may be
// negative past the deadline.
func (g *Grant) Remaining() time.Duration {
	return g.deadline.Sub(g.h.now())
}

// Allocate is the per-allocation checkpoint: call it before each allocation
// effect under the Grant (workspace materialize, process spawn). It rechecks
// the admission floor, so disk that dropped below the floor after Admit
// refuses the next allocation with ErrBelowFloor. The deadline stays
// absolute: an allocation at or past the deadline reports
// ErrDeadlineExceeded, and an allocation on a finished Grant reports
// ErrReleased. Allocate is not a verdict: it never releases the Grant,
// and a refused allocation leaves the Grant held for abort handling.
func (g *Grant) Allocate() error {
	g.mu.Lock()
	done := g.done
	g.mu.Unlock()
	if done {
		return ErrReleased
	}
	if now := g.h.now(); !now.Before(g.deadline) {
		return fmt.Errorf("%w: allocation at %v, deadline %v", ErrDeadlineExceeded, now, g.deadline)
	}
	return g.h.checkFloor()
}

// finish releases the kernel locks and bookkeeping. The caller records
// the verdict.
func (g *Grant) finish() {
	for _, l := range g.lanes {
		g.files[l].Close()
	}
	g.h.unmark(g.lanes)
}

// Complete records the success verdict and releases the reservation.
// The deadline is absolute and monotonic: reaching it is failure even
// if the body just succeeded, so Complete at or past the deadline
// reports ErrDeadlineExceeded. Complete is single-shot; a second call
// reports ErrReleased.
func (g *Grant) Complete() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.done {
		return ErrReleased
	}
	g.done = true
	now := g.h.now()
	g.finish()
	if !now.Before(g.deadline) {
		return fmt.Errorf("%w: completed at %v, deadline %v", ErrDeadlineExceeded, now, g.deadline)
	}
	return nil
}

// Release abandons the reservation without a success verdict, for abort
// paths. It is idempotent.
func (g *Grant) Release() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.done {
		return
	}
	g.done = true
	g.finish()
}

// ProbeLane is the independent per-lane witness: it consults the
// process registry and then attempts a non-blocking exclusive lock on a
// fresh open of the lane file (no shared state with any holder),
// atomically releasing its probe lock before returning held=false. A
// lane file that was never created proves no holder. Any process can
// run it.
func ProbeLane(dir string, lane Lane) (held bool, err error) {
	if !validLane(lane) {
		return false, fmt.Errorf("%w: unknown lane %q", ErrInvalid, lane)
	}
	if dir == "" {
		return false, fmt.Errorf("%w: empty host lock directory", ErrInvalid)
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return false, fmt.Errorf("admission: resolve host lock directory: %w", err)
	}
	path := filepath.Join(abs, string(lane)+".lock")
	registryMu.Lock()
	_, taken := registry[path]
	registryMu.Unlock()
	if taken {
		return true, nil
	}
	f, err := os.OpenFile(path, os.O_RDWR, 0o600)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("admission: probe lane: %w", err)
	}
	defer f.Close()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return true, nil
		}
		return false, fmt.Errorf("admission: probe lane: %w", err)
	}
	return false, nil
}
