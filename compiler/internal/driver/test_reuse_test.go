package driver

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/tools/native-test-owner/admission"
)

// fakeReuseAlloc is a scriptable ReuseAllocator: it counts checkpoints and
// fails with err when set.
type fakeReuseAlloc struct {
	mu    sync.Mutex
	calls int
	err   error
}

func (f *fakeReuseAlloc) Allocate() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	return f.err
}

func (f *fakeReuseAlloc) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func reuseOwner() TestOwner {
	return TestOwner{PID: os.Getpid(), StartToken: "reuse-test-owner"}
}

func reuseSetup(t *testing.T, alloc ReuseAllocator) (string, *OutputStore, *ReuseCache) {
	t.Helper()
	root := outputProject(t)
	store := outputBegin(t, root)
	cache, err := NewReuseCache(store, "run-1", alloc, reuseOwner())
	if err != nil {
		t.Fatal(err)
	}
	return root, store, cache
}

func reuseBuildsDir(t *testing.T, root string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, "dist", "builds"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func TestNewReuseCacheValidation(t *testing.T) {
	root := outputProject(t)
	store := outputBegin(t, root)
	alloc := &fakeReuseAlloc{}
	if _, err := NewReuseCache(nil, "run-1", alloc, reuseOwner()); !errors.Is(err, ErrReuseInvalid) {
		t.Fatalf("nil store: %v", err)
	}
	if _, err := NewReuseCache(store, "", alloc, reuseOwner()); !errors.Is(err, ErrReuseInvalid) {
		t.Fatalf("empty run: %v", err)
	}
	if _, err := NewReuseCache(store, "run-1", nil, reuseOwner()); !errors.Is(err, ErrReuseInvalid) {
		t.Fatalf("nil allocator: %v", err)
	}
	for name, owner := range map[string]TestOwner{
		"zero pid":    {PID: 0, StartToken: "tok"},
		"empty token": {PID: os.Getpid()},
		"long token":  {PID: os.Getpid(), StartToken: strings.Repeat("t", MaxTestOwnerTokenLen+1)},
	} {
		if _, err := NewReuseCache(store, "run-1", alloc, owner); !errors.Is(err, ErrReuseInvalid) {
			t.Fatalf("%s: %v", name, err)
		}
	}
	cache, err := NewReuseCache(store, "run-1", alloc, reuseOwner())
	if err != nil {
		t.Fatal(err)
	}
	if cache.RunID() != "run-1" {
		t.Fatalf("RunID = %q", cache.RunID())
	}
}

func TestReuseFillLookupHit(t *testing.T) {
	_, store, cache := reuseSetup(t, &fakeReuseAlloc{})
	var produced atomic.Int32
	build := func(context.Context) (*PreparedOutput, error) {
		produced.Add(1)
		return outputPrepared(t, store, "export const value=1n;"), nil
	}
	first, err := cache.Fill(context.Background(), "key-a", build)
	if err != nil {
		t.Fatal(err)
	}
	second, err := cache.Fill(context.Background(), "key-a", build)
	if err != nil {
		t.Fatal(err)
	}
	if produced.Load() != 1 {
		t.Fatalf("producer ran %d times, want 1", produced.Load())
	}
	if first.BuildID() != second.BuildID() {
		t.Fatal("joined fill disagrees on build ID")
	}
	third, err := cache.Lookup("key-a")
	if err != nil {
		t.Fatal(err)
	}
	if third.BuildID() != first.BuildID() {
		t.Fatal("lookup disagrees on build ID")
	}
	if err := third.VerifyOwner(os.Getpid(), "reuse-test-owner"); err != nil {
		t.Fatalf("owner rejected: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	if err := second.Close(); err != nil {
		t.Fatal(err)
	}
	if err := third.Close(); err != nil {
		t.Fatal(err)
	}
	facts, err := cache.Close()
	if err != nil {
		t.Fatal(err)
	}
	if !facts.Clean || facts.RunID != "run-1" || len(facts.Published) != 1 || facts.Published[0] != "key-a" {
		t.Fatalf("facts = %+v", facts)
	}
	if facts.FillsFailed != 0 || facts.LeasesAcquired != 3 || facts.LeasesReleased != 3 || facts.Outstanding != 0 {
		t.Fatalf("accounting = %+v", facts)
	}
	if _, err := cache.Close(); !errors.Is(err, ErrReuseClosed) {
		t.Fatalf("second close: %v", err)
	}
	if _, err := cache.Lookup("key-a"); !errors.Is(err, ErrReuseClosed) {
		t.Fatalf("lookup after close: %v", err)
	}
}

func TestReuseLookupMiss(t *testing.T) {
	_, _, cache := reuseSetup(t, &fakeReuseAlloc{})
	if _, err := cache.Lookup("absent"); !errors.Is(err, ErrReuseMiss) {
		t.Fatalf("miss: %v", err)
	}
	if _, err := cache.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestReuseInvalidKeysRefuseBeforeCheckpoint(t *testing.T) {
	alloc := &fakeReuseAlloc{}
	_, _, cache := reuseSetup(t, alloc)
	long := strings.Repeat("k", MaxReuseKeyLen+1)
	build := func(context.Context) (*PreparedOutput, error) {
		t.Error("producer ran for invalid key")
		return nil, nil
	}
	for _, key := range []string{"", long} {
		if _, err := cache.Lookup(key); !errors.Is(err, ErrReuseInvalid) {
			t.Fatalf("lookup %q: %v", key, err)
		}
		if _, err := cache.Fill(context.Background(), key, build); !errors.Is(err, ErrReuseInvalid) {
			t.Fatalf("fill %q: %v", key, err)
		}
		if err := cache.Evict(key); !errors.Is(err, ErrReuseInvalid) {
			t.Fatalf("evict %q: %v", key, err)
		}
	}
	if _, err := cache.Fill(context.Background(), "ok", nil); !errors.Is(err, ErrReuseInvalid) {
		t.Fatalf("nil producer: %v", err)
	}
	if alloc.count() != 0 {
		t.Fatalf("invalid requests took %d allocator checkpoints", alloc.count())
	}
	if _, err := cache.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestReuseFailedFillPublishesNothing(t *testing.T) {
	root, store, cache := reuseSetup(t, &fakeReuseAlloc{})
	before := reuseBuildsDir(t, root)
	boom := errors.New("boom")
	if _, err := cache.Fill(context.Background(), "key-f", func(context.Context) (*PreparedOutput, error) {
		return nil, boom
	}); !errors.Is(err, boom) {
		t.Fatalf("failed fill: %v", err)
	}
	if _, err := cache.Lookup("key-f"); !errors.Is(err, ErrReuseMiss) {
		t.Fatalf("failed fill published: %v", err)
	}
	if _, err := cache.Fill(context.Background(), "key-f", func(context.Context) (*PreparedOutput, error) {
		return nil, nil
	}); err == nil || !strings.Contains(err.Error(), "produced no output") {
		t.Fatalf("nil fill: %v", err)
	}
	// A retry with real output publishes exactly one generation.
	lease, err := cache.Fill(context.Background(), "key-f", func(context.Context) (*PreparedOutput, error) {
		return outputPrepared(t, store, "export const value=1n;"), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	after := reuseBuildsDir(t, root)
	if len(after)-len(before) != 1 {
		t.Fatalf("builds dir grew by %d, want 1", len(after)-len(before))
	}
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}
	facts, err := cache.Close()
	if err != nil {
		t.Fatal(err)
	}
	if facts.FillsFailed != 2 {
		t.Fatalf("fillsFailed = %d, want 2", facts.FillsFailed)
	}
}

func TestReuseUnvalidatedFillPublishesNothing(t *testing.T) {
	root, store, cache := reuseSetup(t, &fakeReuseAlloc{})
	before := reuseBuildsDir(t, root)
	prepared := outputPrepared(t, store, "export const value=1n;")
	prepared.validated = false
	if _, err := cache.Fill(context.Background(), "key-u", func(context.Context) (*PreparedOutput, error) {
		return prepared, nil
	}); err == nil || !strings.Contains(err.Error(), "did not publish") {
		t.Fatalf("unvalidated fill: %v", err)
	}
	if _, err := cache.Lookup("key-u"); !errors.Is(err, ErrReuseMiss) {
		t.Fatalf("unvalidated fill published: %v", err)
	}
	if after := reuseBuildsDir(t, root); len(after) != len(before) {
		t.Fatal("unvalidated fill left staged residue")
	}
	if _, err := cache.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestReuseSingleProducerOverlap(t *testing.T) {
	_, store, cache := reuseSetup(t, &fakeReuseAlloc{})
	var produced atomic.Int32
	start := make(chan struct{})
	var started sync.WaitGroup
	started.Add(8)
	build := func(context.Context) (*PreparedOutput, error) {
		produced.Add(1)
		return outputPrepared(t, store, "export const value=1n;"), nil
	}
	results := make([]*ReuseLease, 8)
	errs := make([]error, 8)
	var wg sync.WaitGroup
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			started.Done()
			<-start
			results[i], errs[i] = cache.Fill(context.Background(), "key Hot", build)
		}(i)
	}
	started.Wait()
	close(start)
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("filler %d: %v", i, err)
		}
		defer results[i].Close()
		if results[i].BuildID() != results[0].BuildID() {
			t.Fatal("filler disagrees on build ID")
		}
	}
	if produced.Load() != 1 {
		t.Fatalf("producer ran %d times, want 1", produced.Load())
	}
	for _, l := range results {
		if err := l.Close(); err != nil {
			t.Fatal(err)
		}
	}
	facts, err := cache.Close()
	if err != nil {
		t.Fatal(err)
	}
	if facts.WaitersJoined != 7 {
		t.Fatalf("waitersJoined = %d, want 7", facts.WaitersJoined)
	}
}

func TestReuseDistinctKeysProceedIndependently(t *testing.T) {
	_, store, cache := reuseSetup(t, &fakeReuseAlloc{})
	aEntered := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	buildA := func(context.Context) (*PreparedOutput, error) {
		once.Do(func() { close(aEntered) })
		<-release
		return outputPrepared(t, store, "export const value=1n;"), nil
	}
	buildB := func(context.Context) (*PreparedOutput, error) {
		return outputPrepared(t, store, "export const value=2n;"), nil
	}
	type outcome struct {
		lease *ReuseLease
		err   error
	}
	aOut := make(chan outcome, 1)
	bOut := make(chan outcome, 1)
	go func() {
		lease, err := cache.Fill(context.Background(), "key-a", buildA)
		aOut <- outcome{lease, err}
	}()
	<-aEntered
	go func() {
		lease, err := cache.Fill(context.Background(), "key-b", buildB)
		bOut <- outcome{lease, err}
	}()
	b := <-bOut
	if b.err != nil {
		t.Fatalf("key-b blocked behind key-a: %v", b.err)
	}
	close(release)
	a := <-aOut
	if a.err != nil {
		t.Fatal(a.err)
	}
	if a.lease.BuildID() == b.lease.BuildID() {
		t.Fatal("distinct inputs share a build ID")
	}
	a.lease.Close()
	b.lease.Close()
	if _, err := cache.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestReuseWaiterCancellationDetachesOnlyTheWaiter(t *testing.T) {
	_, store, cache := reuseSetup(t, &fakeReuseAlloc{})
	entered := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	build := func(context.Context) (*PreparedOutput, error) {
		once.Do(func() { close(entered) })
		<-release
		return outputPrepared(t, store, "export const value=1n;"), nil
	}
	type outcome struct {
		lease *ReuseLease
		err   error
	}
	prodOut := make(chan outcome, 1)
	go func() {
		lease, err := cache.Fill(context.Background(), "key-w", build)
		prodOut <- outcome{lease, err}
	}()
	<-entered
	ctx, cancel := context.WithCancel(context.Background())
	waitOut := make(chan outcome, 1)
	go func() {
		lease, err := cache.Fill(ctx, "key-w", build)
		waitOut <- outcome{lease, err}
	}()
	// Give the waiter a bounded moment to join, then cancel it while the
	// producer is still blocked.
	deadline := time.Now().Add(5 * time.Second)
	for {
		cache.mu.Lock()
		joined := cache.waitersJoined
		cache.mu.Unlock()
		if joined == 1 || time.Now().After(deadline) {
			break
		}
		time.Sleep(time.Millisecond)
	}
	cancel()
	w := <-waitOut
	if w.lease != nil || !errors.Is(w.err, context.Canceled) {
		t.Fatalf("cancelled waiter: lease=%v err=%v", w.lease, w.err)
	}
	close(release)
	p := <-prodOut
	if p.err != nil {
		t.Fatalf("producer failed after waiter cancel: %v", p.err)
	}
	later, err := cache.Lookup("key-w")
	if err != nil {
		t.Fatalf("lookup after cancel: %v", err)
	}
	if later.BuildID() != p.lease.BuildID() {
		t.Fatal("post-cancel lookup disagrees on build ID")
	}
	p.lease.Close()
	later.Close()
	facts, err := cache.Close()
	if err != nil {
		t.Fatal(err)
	}
	if facts.WaitersJoined != 1 || facts.WaitersDetached != 1 {
		t.Fatalf("waiter accounting = %+v", facts)
	}
}

func TestReuseAllocatorRefusalBlocksEffects(t *testing.T) {
	alloc := &fakeReuseAlloc{}
	_, store, cache := reuseSetup(t, alloc)
	lease, err := cache.Fill(context.Background(), "key-g", func(context.Context) (*PreparedOutput, error) {
		return outputPrepared(t, store, "export const value=1n;"), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}
	// Refuse everything from here on: fills must not run their producer
	// and lookups must not acquire leases.
	alloc.mu.Lock()
	alloc.err = errors.New("admission refused")
	alloc.mu.Unlock()
	ran := false
	if _, err := cache.Fill(context.Background(), "key-h", func(context.Context) (*PreparedOutput, error) {
		ran = true
		return outputPrepared(t, store, "export const value=2n;"), nil
	}); err == nil || !strings.Contains(err.Error(), "admission refused") {
		t.Fatalf("refused fill: %v", err)
	}
	if ran {
		t.Fatal("producer ran despite allocator refusal")
	}
	if _, err := cache.Lookup("key-g"); err == nil || !strings.Contains(err.Error(), "admission refused") {
		t.Fatalf("refused lookup: %v", err)
	}
	facts, err := cache.Close()
	if err != nil {
		t.Fatal(err)
	}
	if facts.LeasesAcquired != 1 || facts.FillsFailed != 1 {
		t.Fatalf("accounting = %+v", facts)
	}
}

// fakeReuseClock and fakeReuseDisk drive the real P12 gate deterministically.
type fakeReuseClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fakeReuseClock) at() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeReuseClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

type fakeReuseDisk struct {
	mu    sync.Mutex
	avail uint64
}

func (d *fakeReuseDisk) probe(string) (uint64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.avail, nil
}

func (d *fakeReuseDisk) set(avail uint64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.avail = avail
}

func TestReuseAdmissionGrantIsTheEnforcementPrimitive(t *testing.T) {
	// The allocator behind a production ReuseCache is the run's real P12
	// Grant: build-producer lane, finite Capability ceilings, absolute
	// deadline and floor recheck on every Allocate.
	clock := &fakeReuseClock{now: time.Now()}
	disk := &fakeReuseDisk{avail: 20 << 30}
	host, err := admission.OpenHostWithClockAndFreeDisk(t.TempDir(), clock.at, disk.probe)
	if err != nil {
		t.Fatal(err)
	}
	grant, err := host.Admit("/repo", admission.Request{
		Demand:         admission.Demand{Produce: true},
		Capability:     admission.Capability{MaxHandles: 64, MaxPending: 16, MaxBytes: 1 << 30},
		Budget:         time.Minute,
		Body:           time.Second,
		CleanupReserve: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer grant.Release()
	if got := grant.Capability(); got.MaxHandles != 64 || got.MaxPending != 16 || got.MaxBytes != 1<<30 {
		t.Fatalf("capability ceilings = %+v", got)
	}
	_, store, cache := reuseSetup(t, grant)
	lease, err := cache.Fill(context.Background(), "key-p12", func(context.Context) (*PreparedOutput, error) {
		return outputPrepared(t, store, "export const value=1n;"), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}
	// Past the absolute deadline the gate refuses and the fill fails
	// before running its producer.
	clock.advance(2 * time.Minute)
	ran := false
	if _, err := cache.Fill(context.Background(), "key-late", func(context.Context) (*PreparedOutput, error) {
		ran = true
		return outputPrepared(t, store, "export const value=2n;"), nil
	}); !errors.Is(err, admission.ErrDeadlineExceeded) {
		t.Fatalf("past-deadline fill: %v", err)
	}
	if ran {
		t.Fatal("producer ran past the deadline")
	}
	// Below the disk floor the gate refuses even with time remaining.
	clock.advance(-90 * time.Second)
	disk.set((10 << 30) - 1)
	if _, err := cache.Lookup("key-p12"); !errors.Is(err, admission.ErrBelowFloor) {
		t.Fatalf("below-floor lookup: %v", err)
	}
	disk.set(20 << 30)
	recovered, err := cache.Lookup("key-p12")
	if err != nil {
		t.Fatalf("recovered lookup: %v", err)
	}
	// Leases acquired above: producer fill (released) + recovered lookup
	// (outstanding).
	facts, err := cache.Close()
	if err == nil {
		t.Fatal("close with outstanding leases succeeded")
	}
	if facts.Outstanding != 1 || facts.Clean {
		t.Fatalf("facts = %+v", facts)
	}
	if err := recovered.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestReuseLeasedEvictionRefused(t *testing.T) {
	root, store, cache := reuseSetup(t, &fakeReuseAlloc{})
	lease, err := cache.Fill(context.Background(), "key-e", func(context.Context) (*PreparedOutput, error) {
		return outputPrepared(t, store, "export const value=1n;"), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := cache.Evict("key-e"); !errors.Is(err, ErrReuseLeased) {
		t.Fatalf("leased evict: %v", err)
	}
	// A second lease keeps the refusal standing after the first release.
	other, err := cache.Lookup("key-e")
	if err != nil {
		t.Fatal(err)
	}
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}
	if err := cache.Evict("key-e"); !errors.Is(err, ErrReuseLeased) {
		t.Fatalf("leased evict after one release: %v", err)
	}
	held, err := other.ProbeHeld()
	if err != nil || !held {
		t.Fatalf("probe held=%v err=%v", held, err)
	}
	if err := other.Close(); err != nil {
		t.Fatal(err)
	}
	// Idempotent close records exactly one release per lease.
	if err := other.Close(); err != nil {
		t.Fatal(err)
	}
	if err := cache.Evict("key-e"); err != nil {
		t.Fatalf("unleased evict: %v", err)
	}
	if _, err := cache.Lookup("key-e"); !errors.Is(err, ErrReuseMiss) {
		t.Fatalf("lookup after evict: %v", err)
	}
	if err := cache.Evict("key-e"); !errors.Is(err, ErrReuseMiss) {
		t.Fatalf("second evict: %v", err)
	}
	if entries := reuseBuildsDir(t, root); len(entries) != 0 {
		t.Fatalf("evicted generation left %v", entries)
	}
	facts, err := cache.Close()
	if err != nil {
		t.Fatal(err)
	}
	if facts.LeasesAcquired != 2 || facts.LeasesReleased != 2 {
		t.Fatalf("accounting = %+v", facts)
	}
}

func TestReuseEvictDuringFillRefused(t *testing.T) {
	_, store, cache := reuseSetup(t, &fakeReuseAlloc{})
	entered := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	type outcome struct {
		lease *ReuseLease
		err   error
	}
	out := make(chan outcome, 1)
	go func() {
		lease, err := cache.Fill(context.Background(), "key-i", func(context.Context) (*PreparedOutput, error) {
			once.Do(func() { close(entered) })
			<-release
			return outputPrepared(t, store, "export const value=1n;"), nil
		})
		out <- outcome{lease, err}
	}()
	<-entered
	if err := cache.Evict("key-i"); !errors.Is(err, ErrReuseFilling) {
		t.Fatalf("evict during fill: %v", err)
	}
	close(release)
	r := <-out
	if r.err != nil {
		t.Fatal(r.err)
	}
	r.lease.Close()
	if _, err := cache.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestReuseEvictRefusesProductionCurrent(t *testing.T) {
	_, store, cache := reuseSetup(t, &fakeReuseAlloc{})
	prepared := outputPrepared(t, store, "export const value=1n;")
	if _, err := store.Publish(prepared); err != nil {
		t.Fatal(err)
	}
	// The same bytes stage idempotently onto the current generation; the
	// reuse mapping is servable but never evictable.
	lease, err := cache.Fill(context.Background(), "key-c", func(context.Context) (*PreparedOutput, error) {
		return outputPrepared(t, store, "export const value=1n;"), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}
	if err := cache.Evict("key-c"); err == nil || !strings.Contains(err.Error(), "production current") {
		t.Fatalf("evict current: %v", err)
	}
	served, err := cache.Lookup("key-c")
	if err != nil {
		t.Fatalf("current still servable: %v", err)
	}
	if err := served.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := cache.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestReuseCorruptBundleNeverServed(t *testing.T) {
	t.Run("modified file", func(t *testing.T) {
		_, store, cache := reuseSetup(t, &fakeReuseAlloc{})
		lease, err := cache.Fill(context.Background(), "key-x", func(context.Context) (*PreparedOutput, error) {
			return outputPrepared(t, store, "export const value=1n;"), nil
		})
		if err != nil {
			t.Fatal(err)
		}
		dir := lease.Lease().Directory
		if err := lease.Close(); err != nil {
			t.Fatal(err)
		}
		victim := filepath.Join(dir, "packages", "p-a", "a.ts")
		original, err := os.ReadFile(victim)
		if err != nil {
			t.Fatal(err)
		}
		// Staged content files are read-only; force writability to
		// simulate a corrupting writer.
		if err := os.Chmod(victim, 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(victim, []byte("export const value=MoRd;"), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := cache.Lookup("key-x"); err == nil {
			t.Fatal("corrupt bundle served")
		} else if !strings.Contains(err.Error(), "modified output file") {
			t.Fatalf("corrupt lookup: %v", err)
		}
		if err := os.WriteFile(victim, original, 0600); err != nil {
			t.Fatal(err)
		}
		// Restored bytes serve again: the refusal was the corruption,
		// not the mapping.
		restored, err := cache.Lookup("key-x")
		if err != nil {
			t.Fatalf("restored lookup: %v", err)
		}
		restored.Close()
		if _, err := cache.Close(); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("tampered manifest", func(t *testing.T) {
		_, store, cache := reuseSetup(t, &fakeReuseAlloc{})
		lease, err := cache.Fill(context.Background(), "key-m", func(context.Context) (*PreparedOutput, error) {
			return outputPrepared(t, store, "export const value=1n;"), nil
		})
		if err != nil {
			t.Fatal(err)
		}
		manifest := filepath.Join(lease.Lease().Directory, "manifest.json")
		if err := lease.Close(); err != nil {
			t.Fatal(err)
		}
		original, err := os.ReadFile(manifest)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(manifest, []byte(`{"bogus":true}`), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := cache.Lookup("key-m"); err == nil {
			t.Fatal("tampered manifest served")
		}
		if err := os.WriteFile(manifest, original, 0600); err != nil {
			t.Fatal(err)
		}
		restored, err := cache.Lookup("key-m")
		if err != nil {
			t.Fatalf("restored lookup: %v", err)
		}
		restored.Close()
		if _, err := cache.Close(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestReuseTableIsMemoryOnly(t *testing.T) {
	// A second cache over the same store and run ID starts empty: no key
	// index is persisted, so reuse cannot leak across runs.
	_, store, first := reuseSetup(t, &fakeReuseAlloc{})
	lease, err := first.Fill(context.Background(), "key-s", func(context.Context) (*PreparedOutput, error) {
		return outputPrepared(t, store, "export const value=1n;"), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}
	second, err := NewReuseCache(store, "run-1", &fakeReuseAlloc{}, reuseOwner())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := second.Lookup("key-s"); !errors.Is(err, ErrReuseMiss) {
		t.Fatalf("second cache hit persisted state: %v", err)
	}
	if _, err := first.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := second.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestReuseCloseDuringFill(t *testing.T) {
	root, store, cache := reuseSetup(t, &fakeReuseAlloc{})
	entered := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	type outcome struct {
		lease *ReuseLease
		err   error
	}
	out := make(chan outcome, 1)
	go func() {
		lease, err := cache.Fill(context.Background(), "key-z", func(context.Context) (*PreparedOutput, error) {
			once.Do(func() { close(entered) })
			<-release
			return outputPrepared(t, store, "export const value=1n;"), nil
		})
		out <- outcome{lease, err}
	}()
	<-entered
	facts, err := cache.Close()
	if err == nil {
		t.Fatal("close during fill succeeded")
	}
	if facts.Clean || len(facts.InFlight) != 1 || facts.InFlight[0] != "key-z" {
		t.Fatalf("facts = %+v", facts)
	}
	close(release)
	r := <-out
	if r.lease != nil || !errors.Is(r.err, ErrReuseClosed) {
		t.Fatalf("producer after close: lease=%v err=%v", r.lease, r.err)
	}
	if entries := reuseBuildsDir(t, root); len(entries) != 0 {
		t.Fatalf("closed fill left staged residue: %v", entries)
	}
}

func TestReuseCloseWithOutstandingLease(t *testing.T) {
	_, store, cache := reuseSetup(t, &fakeReuseAlloc{})
	lease, err := cache.Fill(context.Background(), "key-o", func(context.Context) (*PreparedOutput, error) {
		return outputPrepared(t, store, "export const value=1n;"), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	facts, err := cache.Close()
	if err == nil {
		t.Fatal("close with outstanding lease succeeded")
	}
	if facts.Clean || facts.Outstanding != 1 {
		t.Fatalf("facts = %+v", facts)
	}
	// The lease still releases cleanly after close; accounting records it
	// but the run already ended unresolved.
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}
	again, err := cache.Close()
	if !errors.Is(err, ErrReuseClosed) {
		t.Fatalf("second close: %v", err)
	}
	if again != facts {
		t.Fatal("second close returned different facts")
	}
}

// TestReuseEvictLookupRaceNeverWedges hammers concurrent Lookup and Evict
// across a few keys. A Lookup that counts a lease between Evict's
// unleased pre-check and its post-discard relock must not leave a mapping
// to a discarded generation: every Evict reports only nil, miss, leased,
// or filling, and after quiescence every key Fills cleanly (a wedged
// mapping would fail Lookup with a generation error instead of serving).
func TestReuseEvictLookupRaceNeverWedges(t *testing.T) {
	_, store, cache := reuseSetup(t, &fakeReuseAlloc{})
	// Deliberately identical bytes for every key: generations are
	// content-addressed, so all four keys share one build ID and every
	// evict races both acquisition and aliasing.
	build := func(context.Context) (*PreparedOutput, error) {
		return outputPrepared(t, store, "export const value=1n;"), nil
	}
	keys := []string{"key-r0", "key-r1", "key-r2", "key-r3"}
	for _, key := range keys {
		lease, err := cache.Fill(context.Background(), key, build)
		if err != nil {
			t.Fatal(err)
		}
		if err := lease.Close(); err != nil {
			t.Fatal(err)
		}
	}
	var wg sync.WaitGroup
	for w := 0; w < 8; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < 25; i++ {
				key := keys[(w+i)%len(keys)]
				if (w+i)%2 == 0 {
					lease, err := cache.Lookup(key)
					if err != nil {
						if errors.Is(err, ErrReuseMiss) {
							if lease, ferr := cache.Fill(context.Background(), key, build); ferr == nil {
								_ = lease.Close()
							} else if !errors.Is(ferr, ErrReuseMiss) {
								t.Errorf("fill %q: %v", key, ferr)
								return
							}
						} else {
							t.Errorf("lookup %q: %v", key, err)
							return
						}
						continue
					}
					_ = lease.Close()
					continue
				}
				err := cache.Evict(key)
				switch {
				case err == nil,
					errors.Is(err, ErrReuseMiss),
					errors.Is(err, ErrReuseLeased),
					errors.Is(err, ErrReuseFilling):
				default:
					t.Errorf("evict %q: %v", key, err)
					return
				}
			}
		}(w)
	}
	wg.Wait()
	// Post-quiescence wedge detector: with no lease outstanding anywhere,
	// every key either serves or produces. A mapping to a discarded
	// generation would fail here with a generation error.
	for _, key := range keys {
		lease, err := cache.Fill(context.Background(), key, build)
		if err != nil {
			t.Fatalf("post-race fill %q: %v", key, err)
		}
		_ = lease.Close()
	}
	facts, err := cache.Close()
	if err != nil {
		t.Fatalf("close: %v (%+v)", err, facts)
	}
	if !facts.Clean || facts.Outstanding != 0 {
		t.Fatalf("facts = %+v", facts)
	}
}
