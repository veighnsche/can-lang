// Run-scoped immutable build reuse for the native-test driver boundary.
//
// P27 splits reuse in two: Can (tests/native-can/src/builds/) owns complete
// cache keys and reuse policy, and this file owns the native mechanics —
// atomic publication, producer reservations, waiter cancellation, owner-bound
// leases, lease accounting and release facts. The split is strict:
//
//   - Keys are opaque to Go. Can renders the canonical key from the complete
//     source/dependencies/compiler/catalogue/runtime/options/profile input
//     set; Go never parses a key, only rejects empty or overlong ones. Key
//     completeness, input drift and reuse-vs-rebuild policy live in Can.
//   - Storage mechanics live here. One producer per key is enforced by an
//     in-process reservation table; publication is OutputStore.Stage (atomic
//     rename after full manifest validation); reads revalidate through
//     AcquireTestGeneration, so a corrupt bundle fails instead of serving.
//
// Enforcement reuses the P12 admission primitive instead of reinventing it:
// every allocation effect (producer fill work, publication, lease acquisition)
// first calls ReuseAllocator.Allocate. In production the allocator is the
// run's *admission.Grant (build-producer lane, finite Capability ceilings,
// absolute deadline, 10 GiB floor recheck); the narrow interface keeps the
// documented no-code-coupling integration while the checkpoint stays the
// real gate. Lease accounting counts outstanding leases for release facts;
// policing live consumption against the declared ceilings is P13 strict
// host enforcement, which this file does not claim.
//
// Within-run scope only: the key→generation table is memory-only. No key
// index is ever written to disk, so a second ReuseCache over the same store
// starts empty and cross-run reuse is impossible by construction. Staged
// generations left behind are ordinary content-addressed store generations,
// reclaimable by Prune; without the key map they are unreachable by key.
package driver

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// MaxReuseKeyLen caps one opaque reuse key. Rendered Can keys are an order
// of magnitude smaller; anything longer is rejected before any effect.
const MaxReuseKeyLen = 4096

// Reuse errors. Callers distinguish them with errors.Is.
var (
	// ErrReuseInvalid reports an empty or overlong key, or an invalid
	// cache/owner identity. It refuses before any allocator checkpoint.
	ErrReuseInvalid = errors.New("driver: invalid reuse request")
	// ErrReuseMiss reports a Lookup for a key with no published generation
	// and no in-flight fill. Lookup never waits; call Fill to produce.
	ErrReuseMiss = errors.New("driver: reuse key not published")
	// ErrReuseLeased reports an Evict refused while leases are outstanding.
	ErrReuseLeased = errors.New("driver: reuse generation has outstanding leases")
	// ErrReuseFilling reports an Evict refused while a producer fill is in
	// flight for the key.
	ErrReuseFilling = errors.New("driver: reuse fill in flight")
	// ErrReuseClosed reports any operation on a closed cache, and wakes
	// waiters whose fill finished after Close.
	ErrReuseClosed = errors.New("driver: reuse cache closed")
)

// ReuseAllocator is the per-allocation enforcement checkpoint: the P12
// admission Grant's Allocate method. Every fill, publication and lease
// acquisition calls it first; a refusal (deadline, disk floor, released
// grant) fails the operation before any effect.
type ReuseAllocator interface {
	Allocate() error
}

// reuseFill is one in-flight producer reservation: exactly one producer per
// key, with any number of waiters blocked on done. done closes exactly once
// when the fill settles; result carries the outcome for every waiter.
type reuseFill struct {
	done   chan struct{}
	result *reuseOutcome
}

// reuseOutcome is the settled result of one fill: either a published build
// ID or the failure that left nothing published.
type reuseOutcome struct {
	buildID string
	err     error
}

// reusePublished is one published key→generation mapping with its live
// lease count. The mapping is immutable: once published it never changes
// within the run; only Evict (unleased, same run) removes it.
type reusePublished struct {
	buildID string
	leases  int
}

// ReuseCache is one run's immutable build-reuse table over one OutputStore.
// The zero ReuseCache is unusable; construct with NewReuseCache. A
// ReuseCache is safe for concurrent use.
type ReuseCache struct {
	store *OutputStore
	runID string
	alloc ReuseAllocator
	owner TestOwner

	mu        sync.Mutex
	published map[string]*reusePublished
	filling   map[string]*reuseFill
	closed    bool

	fillsFailed     int
	leasesAcquired  int
	leasesReleased  int
	waitersJoined   int
	waitersDetached int

	lastRelease *ReuseRelease
}

// NewReuseCache binds one run's reuse table to a store, a run identity, the
// run's admission allocator and the lease owner identity. It rejects a nil
// store, an empty run ID, a nil allocator and a non-positive PID or an
// empty/overlong start token before any effect.
func NewReuseCache(store *OutputStore, runID string, alloc ReuseAllocator, owner TestOwner) (*ReuseCache, error) {
	if store == nil {
		return nil, fmt.Errorf("%w: reuse cache needs a store", ErrReuseInvalid)
	}
	if runID == "" {
		return nil, fmt.Errorf("%w: reuse cache needs a run ID", ErrReuseInvalid)
	}
	if alloc == nil {
		return nil, fmt.Errorf("%w: reuse cache needs an admission allocator", ErrReuseInvalid)
	}
	if owner.PID <= 0 {
		return nil, fmt.Errorf("%w: reuse owner needs a positive PID", ErrReuseInvalid)
	}
	if owner.StartToken == "" || len(owner.StartToken) > MaxTestOwnerTokenLen {
		return nil, fmt.Errorf("%w: reuse owner needs a start token of 1..%d bytes", ErrReuseInvalid, MaxTestOwnerTokenLen)
	}
	return &ReuseCache{
		store: store, runID: runID, alloc: alloc, owner: owner,
		published: make(map[string]*reusePublished),
		filling:   make(map[string]*reuseFill),
	}, nil
}

// RunID reports the bound run identity.
func (c *ReuseCache) RunID() string { return c.runID }

// validKey rejects empty or overlong keys before any effect or checkpoint.
func validReuseKey(key string) error {
	if key == "" || len(key) > MaxReuseKeyLen {
		return fmt.Errorf("%w: key must be 1..%d bytes", ErrReuseInvalid, MaxReuseKeyLen)
	}
	return nil
}

// Lookup leases the published generation for key. A miss (nothing published
// and nothing in flight) reports ErrReuseMiss without waiting; a corrupt
// staged bundle reports its validation failure instead of serving bytes.
// Lease acquisition is an allocation effect: it checkpoints the admission
// allocator first, so a refused allocator fails before any lease exists.
func (c *ReuseCache) Lookup(key string) (*ReuseLease, error) {
	if err := validReuseKey(key); err != nil {
		return nil, err
	}
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil, ErrReuseClosed
	}
	pub, ok := c.published[key]
	c.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrReuseMiss, key)
	}
	if err := c.alloc.Allocate(); err != nil {
		return nil, err
	}
	lease, err := AcquireTestGeneration(c.store, pub.buildID, c.owner)
	if err != nil {
		// An evict that landed between the mapping check and the
		// acquire reads as a miss, not a generation failure: the key
		// is unmapped now, and the caller retries into a producer. A
		// failure under an unchanged mapping is genuine corruption.
		c.mu.Lock()
		current, still := c.published[key]
		c.mu.Unlock()
		if !still || current.buildID != pub.buildID {
			return nil, fmt.Errorf("%w: %q", ErrReuseMiss, key)
		}
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		lease.Close()
		return nil, ErrReuseClosed
	}
	current, still := c.published[key]
	if !still || current.buildID != pub.buildID {
		// Evicted between the checkpoint and the acquire: never hand
		// out a lease the table no longer accounts for.
		lease.Close()
		return nil, fmt.Errorf("%w: %q", ErrReuseMiss, key)
	}
	current.leases++
	c.leasesAcquired++
	return &ReuseLease{cache: c, key: key, inner: lease}, nil
}

// Fill returns a lease on the published generation for key, producing it
// when absent. Exactly one caller per key becomes the producer and runs
// produce; concurrent callers for the same key wait for its outcome, and
// callers for an already-published key join without running produce. A
// waiter whose context cancels detaches with its context error while the
// producer continues unaffected. A failed produce (error, nil or
// unvalidated output) publishes nothing and wakes every waiter with the
// failure, so the next Fill retries with a fresh producer. produce runs
// without any cache lock held.
func (c *ReuseCache) Fill(ctx context.Context, key string, produce func(context.Context) (*PreparedOutput, error)) (*ReuseLease, error) {
	if err := validReuseKey(key); err != nil {
		return nil, err
	}
	if produce == nil {
		return nil, fmt.Errorf("%w: fill needs a producer", ErrReuseInvalid)
	}
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil, ErrReuseClosed
	}
	if _, ok := c.published[key]; ok {
		c.mu.Unlock()
		return c.Lookup(key)
	}
	if fill, ok := c.filling[key]; ok {
		c.waitersJoined++
		c.mu.Unlock()
		return c.waitFill(ctx, key, fill)
	}
	fill := &reuseFill{done: make(chan struct{})}
	c.filling[key] = fill
	c.mu.Unlock()
	return c.produce(ctx, key, fill, produce)
}

// waitFill blocks until the in-flight producer for key settles, the waiter
// context cancels, or the cache closes. Cancellation detaches only this
// waiter: the producer and every other waiter are unaffected. A waiter
// that already cancelled never joins, even when the fill settled first;
// only a cancel landing in the same instant as the settle may still join,
// which is inherent to the race.
func (c *ReuseCache) waitFill(ctx context.Context, key string, fill *reuseFill) (*ReuseLease, error) {
	detach := func() (*ReuseLease, error) {
		c.mu.Lock()
		c.waitersDetached++
		c.mu.Unlock()
		return nil, fmt.Errorf("driver: reuse waiter for %q detached: %w", key, ctx.Err())
	}
	select {
	case <-ctx.Done():
		return detach()
	default:
	}
	select {
	case <-ctx.Done():
		return detach()
	case <-fill.done:
	}
	if fill.result.err != nil {
		return nil, fill.result.err
	}
	return c.Lookup(key)
}

// produce runs the single producer for key: checkpoint, build, checkpoint,
// stage, publish. Every failure path leaves nothing published. produce runs
// without the cache lock; only the short reservation checks and the final
// publication take it.
func (c *ReuseCache) produce(ctx context.Context, key string, fill *reuseFill, build func(context.Context) (*PreparedOutput, error)) (*ReuseLease, error) {
	// settle records the outcome, wakes every waiter and, on success,
	// leases the published generation to the producer. A fill that
	// finished after Close keeps its staged build ID on the outcome so
	// the orphan staged generation can be discarded best-effort below;
	// staged leftovers are ordinary content-addressed generations and
	// Prune reclaims anything the discard misses.
	settle := func(buildID string, err error) (*ReuseLease, error) {
		c.mu.Lock()
		delete(c.filling, key)
		outcome := &reuseOutcome{buildID: buildID, err: err}
		switch {
		case err != nil:
			c.fillsFailed++
		case c.closed:
			outcome.err = ErrReuseClosed
		default:
			c.published[key] = &reusePublished{buildID: buildID}
		}
		fill.result = outcome
		close(fill.done)
		c.mu.Unlock()
		if outcome.err != nil {
			if outcome.err == ErrReuseClosed && buildID != "" {
				_ = c.store.DiscardGeneration(buildID)
			}
			return nil, outcome.err
		}
		return c.Lookup(key)
	}
	if err := c.alloc.Allocate(); err != nil {
		return settle("", err)
	}
	prepared, err := build(ctx)
	if err != nil {
		return settle("", fmt.Errorf("driver: reuse fill for %q failed: %w", key, err))
	}
	if prepared == nil {
		return settle("", fmt.Errorf("driver: reuse fill for %q produced no output", key))
	}
	if err := c.alloc.Allocate(); err != nil {
		return settle("", err)
	}
	id, _, err := c.store.Stage(prepared)
	if err != nil {
		return settle("", fmt.Errorf("driver: reuse fill for %q did not publish: %w", key, err))
	}
	return settle(id, nil)
}

// Evict discards the published generation for key and drops its mapping. It
// refuses while a fill is in flight (ErrReuseFilling), while leases are
// outstanding (ErrReuseLeased), and for unknown keys (ErrReuseMiss). The
// production current selection can never be evicted: DiscardGeneration
// refuses it. Evict is not an allocation effect, so it takes no allocator
// checkpoint; it only releases.
func (c *ReuseCache) Evict(key string) error {
	if err := validReuseKey(key); err != nil {
		return err
	}
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return ErrReuseClosed
	}
	if _, ok := c.filling[key]; ok {
		c.mu.Unlock()
		return fmt.Errorf("%w: %q", ErrReuseFilling, key)
	}
	pub, ok := c.published[key]
	if !ok {
		c.mu.Unlock()
		return fmt.Errorf("%w: %q", ErrReuseMiss, key)
	}
	if pub.leases > 0 {
		c.mu.Unlock()
		return fmt.Errorf("%w: %q holds %d", ErrReuseLeased, key, pub.leases)
	}
	buildID := pub.buildID
	// Generations are content-addressed, so distinct keys can share one
	// build ID (same output bytes from different inputs). Evicting one
	// sharer drops only its mapping; the last sharer discards the
	// generation. Both checks run under this lock, so concurrent evicts
	// of sharers serialize into drops-then-one-discard, never a
	// double-discard or a dangling mapping.
	for k, p := range c.published {
		if k != key && p.buildID == buildID {
			delete(c.published, key)
			c.mu.Unlock()
			return nil
		}
	}
	c.mu.Unlock()
	// Discard outside the lock: it revalidates and unlinks on disk. A
	// Lookup can count a lease between the pre-check above and the
	// relock below; the mapping is still dropped once the generation
	// is gone, and those leases drain as orphans (releaseLease already
	// tolerates unmapped keys). Keeping a mapping to a discarded
	// generation would wedge the key: every Lookup would fail yet Fill
	// would never retry produce.
	if err := c.store.DiscardGeneration(buildID); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if current, still := c.published[key]; still && current.buildID == buildID {
		delete(c.published, key)
	}
	return nil
}

// ReuseRelease is the run's reuse accounting at Close: every published key
// still mapped, every fill failure, every lease acquisition and release,
// and every waiter join and detachment. A clean run closes with no
// outstanding leases and no in-flight fills; anything else stays visible
// here instead of reading as released.
type ReuseRelease struct {
	RunID           string
	Published       []string
	FillsFailed     int
	LeasesAcquired  int
	LeasesReleased  int
	Outstanding     int
	WaitersJoined   int
	WaitersDetached int
	InFlight        []string
	// Clean reports no outstanding leases and no in-flight fills.
	Clean bool
}

// Close ends the run's reuse table and returns its release facts. New
// operations refuse with ErrReuseClosed; in-flight producers settle
// normally and their waiters wake with ErrReuseClosed. Close never waits
// for producers or leases: outstanding leases and in-flight fills are
// recorded on the facts, and Close reports an error while any remain.
// Repeated Close returns the recorded facts with ErrReuseClosed.
func (c *ReuseCache) Close() (*ReuseRelease, error) {
	c.mu.Lock()
	if c.closed {
		facts := c.lastRelease
		c.mu.Unlock()
		return facts, ErrReuseClosed
	}
	c.closed = true
	facts := &ReuseRelease{RunID: c.runID}
	for key := range c.published {
		facts.Published = append(facts.Published, key)
	}
	// Outstanding counts every acquired-but-unreleased lease, including
	// orphans whose mapping an evict-during-acquire already dropped. The
	// two counters are exact: acquired increments once per handed-out
	// lease, released once per idempotent lease Close.
	facts.Outstanding = c.leasesAcquired - c.leasesReleased
	for key := range c.filling {
		facts.InFlight = append(facts.InFlight, key)
	}
	facts.FillsFailed = c.fillsFailed
	facts.LeasesAcquired = c.leasesAcquired
	facts.LeasesReleased = c.leasesReleased
	facts.WaitersJoined = c.waitersJoined
	facts.WaitersDetached = c.waitersDetached
	facts.Clean = facts.Outstanding == 0 && len(facts.InFlight) == 0
	c.lastRelease = facts
	c.mu.Unlock()
	if !facts.Clean {
		return facts, fmt.Errorf("driver: reuse run %q closed with %d outstanding leases and %d in-flight fills",
			c.runID, facts.Outstanding, len(facts.InFlight))
	}
	return facts, nil
}

// releaseLease records one lease release for key. It runs under the cache
// lock from ReuseLease.Close exactly once per lease.
func (c *ReuseCache) releaseLease(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if pub, ok := c.published[key]; ok && pub.leases > 0 {
		pub.leases--
	}
	c.leasesReleased++
}

// ReuseLease is one accounted, owner-bound lease on a published reuse
// generation. It delegates the kernel lease to TestGenerationLease
// (P10 mechanics: shared flock inherited across spawn, owner identity,
// ProbeHeld release witness) and records exactly one release against the
// run's accounting on Close. A ReuseLease is safe for concurrent use.
type ReuseLease struct {
	cache *ReuseCache
	key   string
	inner *TestGenerationLease

	mu     sync.Mutex
	closed bool
}

// Key reports the reuse key this lease was acquired under.
func (l *ReuseLease) Key() string { return l.key }

// BuildID reports the leased staged generation.
func (l *ReuseLease) BuildID() string { return l.inner.BuildID() }

// Owner reports the bound launcher identity.
func (l *ReuseLease) Owner() TestOwner { return l.inner.Owner() }

// AcquiredAt reports the wall time of the acquire.
func (l *ReuseLease) AcquiredAt() time.Time { return l.inner.AcquiredAt() }

// Lease exposes the underlying generation lease for the existing run paths.
// Callers must VerifyOwner before using it: the build ID alone is not
// authority.
func (l *ReuseLease) Lease() *OutputLease { return l.inner.Lease() }

// VerifyOwner requires an exact spawn-start identity match before use.
func (l *ReuseLease) VerifyOwner(pid int, token string) error {
	return l.inner.VerifyOwner(pid, token)
}

// ProbeHeld is the independent release witness: it re-opens the generation
// manifest afresh and attempts an exclusive lock. See TestGenerationLease
// for the shared-generation caveat.
func (l *ReuseLease) ProbeHeld() (bool, error) { return l.inner.ProbeHeld() }

// Close releases this lease's share of the kernel lock and records one
// release against the run's accounting. Close is idempotent.
func (l *ReuseLease) Close() error {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return nil
	}
	l.closed = true
	l.mu.Unlock()
	err := l.inner.Close()
	l.cache.releaseLease(l.key)
	return err
}
