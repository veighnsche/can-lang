// Owner-bound test-generation leases for the native-test driver boundary.
//
// P10 owns processes and descriptors: the kernel fd 4 generation
// read-lease handoff already works through prepareEntry (ExtraFiles[0]
// is the fd 3 environment pipe, leases follow at fd 4+) and
// AcquireGeneration (a shared flock on builds/<id>/manifest.json that
// children inherit, so protection survives the launcher's exit while
// any descendant holds the descriptor). This file adds the two pieces
// that handoff lacked:
//
//   - Owner identity: an acquire records the launcher PID plus a start
//     token, and every use re-verifies both. A bare build ID is never
//     authority, so PID reuse or a cross-launcher handle fails closed
//     instead of operating on a stranger's generation.
//   - Release witness: ProbeHeld re-opens the generation manifest
//     afresh and attempts an exclusive lock, with no shared fd state.
//     Any process can run it.
//
// Shared-generation caveat: staged generations are shared across
// processes, and shared read locks compose. A held probe after this
// owner closes its share therefore does NOT prove an orphaned
// descendant: a concurrent holder (another canlc, the R seed, a
// candidate) keeps the lock held too. Orphan detection needs a
// private lease (as in tools/native-test-owner/process) or the
// per-holder reuse accounting of P27; until then ProbeHeld is an
// observation, never an isolation failure. Likewise there is no
// ack/EOF release pair here: driver workers report on stdout, not
// through a status channel, so acceptance/EOF separation lives in the
// owner package, not at this boundary. The supervise path is
// unchanged by this file.
package driver

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"
)

// MaxTestOwnerTokenLen caps a spawn-start token.
const MaxTestOwnerTokenLen = 128

// ErrTestWrongOwner reports a lease use under a foreign spawn-start
// identity. Callers distinguish it with errors.Is.
var ErrTestWrongOwner = errors.New("driver: wrong test-generation owner")

// TestOwner binds one lease handle to the launcher that acquired it.
type TestOwner struct {
	PID        int
	StartToken string
}

// TestOwnerError is a typed wrong-owner failure naming the expected
// and presented identities without equating either to authority.
type TestOwnerError struct {
	BuildID string
	Want    TestOwner
	Got     TestOwner
}

func (e *TestOwnerError) Error() string {
	return fmt.Sprintf("driver: test generation %s owned by pid %d, not pid %d", e.BuildID, e.Want.PID, e.Got.PID)
}

// Is reports ErrTestWrongOwner for errors.Is.
func (e *TestOwnerError) Is(target error) bool { return target == ErrTestWrongOwner }

// TestGenerationLease is one owner-bound shared read lease on a staged
// generation manifest. The embedded kernel lock is inherited by
// generated children through fd 4; closing every copy releases it.
// A TestGenerationLease is safe for concurrent use.
type TestGenerationLease struct {
	store      *OutputStore
	buildID    string
	owner      TestOwner
	acquiredAt time.Time
	lease      *OutputLease

	mu     sync.Mutex
	closed bool
}

// AcquireTestGeneration leases one staged generation for owner without
// consulting production current. It rejects a nil store, an empty or
// overlong start token, and a non-positive PID before touching the
// store, then records the owner and wall time alongside the kernel
// lease.
func AcquireTestGeneration(store *OutputStore, buildID string, owner TestOwner) (*TestGenerationLease, error) {
	if store == nil {
		return nil, fmt.Errorf("driver: test generation lease needs a store")
	}
	if owner.PID <= 0 {
		return nil, fmt.Errorf("driver: test generation owner needs a positive PID")
	}
	if owner.StartToken == "" || len(owner.StartToken) > MaxTestOwnerTokenLen {
		return nil, fmt.Errorf("driver: test generation owner needs a start token of 1..%d bytes", MaxTestOwnerTokenLen)
	}
	lease, err := store.AcquireGeneration(buildID)
	if err != nil {
		return nil, err
	}
	return &TestGenerationLease{store: store, buildID: buildID, owner: owner, acquiredAt: time.Now(), lease: lease}, nil
}

// BuildID reports the leased staged generation.
func (l *TestGenerationLease) BuildID() string { return l.buildID }

// Owner reports the bound launcher identity.
func (l *TestGenerationLease) Owner() TestOwner { return l.owner }

// AcquiredAt reports the wall time of the acquire for P27 accounting.
func (l *TestGenerationLease) AcquiredAt() time.Time { return l.acquiredAt }

// Lease exposes the underlying generation lease for the existing run
// paths (RunSupervised, RunOutput). Callers must VerifyOwner before
// using it: the build ID alone is not authority.
func (l *TestGenerationLease) Lease() *OutputLease { return l.lease }

// VerifyOwner requires an exact spawn-start identity match. A live PID
// with a stale or foreign token fails with ErrTestWrongOwner even
// when the generation itself is intact.
func (l *TestGenerationLease) VerifyOwner(pid int, token string) error {
	if pid == l.owner.PID && token == l.owner.StartToken {
		return nil
	}
	return &TestOwnerError{BuildID: l.buildID, Want: l.owner, Got: TestOwner{PID: pid, StartToken: token}}
}

// ProbeHeld is the independent release witness: it opens the leased
// generation manifest afresh and attempts a non-blocking exclusive
// lock, releasing the probe before returning. It reports held=true
// while any process holds a shared or exclusive lock. See the
// shared-generation caveat above before treating held-after-close as
// an orphan signal.
func (l *TestGenerationLease) ProbeHeld() (bool, error) {
	file, err := outputOpen(l.store.dist, "builds/"+l.buildID+"/manifest.json", os.O_RDONLY, 0)
	if err != nil {
		return false, err
	}
	defer file.Close()
	if err := outputLock(file, true); err != nil {
		if outputLockBusy(err) {
			return true, nil
		}
		return false, err
	}
	return false, nil
}

// File exposes the locked manifest descriptor for inheritance across
// spawn as fd 4. The caller must VerifyOwner first and must retain
// the lease until the inheriting child exits.
func (l *TestGenerationLease) File() *os.File { return l.lease.File() }

// Close releases this owner's share of the kernel lock. Protection
// continues while any inheriting descendant (or concurrent holder)
// still holds its copy. Close is idempotent.
func (l *TestGenerationLease) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return nil
	}
	l.closed = true
	return l.lease.Close()
}
