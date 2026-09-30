package acceptance

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/veighnsche/can-lang/tools/native-test-owner/admission"
	"github.com/veighnsche/can-lang/tools/native-test-owner/journal"
	"github.com/veighnsche/can-lang/tools/native-test-owner/process"
	"github.com/veighnsche/can-lang/tools/native-test-owner/recovery"
)

// ErrWitnessLost reports T's own loss: a dead witness accepts nothing
// and leaves no green receipt.
var ErrWitnessLost = errors.New("acceptance: witness lost")

// Witness is T, the independent witness: its own journal, process
// owner, generation lease, admission gate and recovery drain. Its
// process/resource graph is separate from N's by construction — N is a
// spawned child observed from the outside, never the observing process.
// A Witness is safe for concurrent use.
type Witness struct {
	id        journal.Owner
	journal   *journal.Journal
	owner     *process.Owner
	lease     *process.Lease
	admission *admission.Host
	gate      *recovery.Gate
	dir       string

	mu   sync.Mutex
	dead bool
}

// NewWitness builds T over dir, which holds its journal, lease and lane
// locks. The witness identity is the calling process plus a fresh
// spawn-start token.
func NewWitness(dir string) (*Witness, error) {
	if dir == "" {
		return nil, fmt.Errorf("%w: empty witness directory", ErrWitnessLost)
	}
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return nil, fmt.Errorf("acceptance: mint witness token: %w", err)
	}
	id := journal.Owner{PID: os.Getpid(), StartToken: hex.EncodeToString(b[:])}
	jdir := filepath.Join(dir, "journal")
	j, err := journal.OpenDir(jdir)
	if err != nil {
		return nil, fmt.Errorf("acceptance: witness journal: %w", err)
	}
	o, err := process.New(j, id.PID, id.StartToken)
	if err != nil {
		_ = j.Close()
		return nil, fmt.Errorf("acceptance: witness owner: %w", err)
	}
	l, err := process.AcquireLease(filepath.Join(dir, "witness.lock"))
	if err != nil {
		_ = j.Close()
		return nil, fmt.Errorf("acceptance: witness lease: %w", err)
	}
	h, err := admission.OpenHost(filepath.Join(dir, "lanes"))
	if err != nil {
		_ = l.Close()
		_ = j.Close()
		return nil, fmt.Errorf("acceptance: witness admission: %w", err)
	}
	return &Witness{
		id: id, journal: j, owner: o, lease: l,
		admission: h, gate: &recovery.Gate{}, dir: dir,
	}, nil
}

// ID reports T's spawn-start identity.
func (w *Witness) ID() journal.Owner { return w.id }

// Alive reports whether T still holds its journal and lease. A dead
// witness accepts nothing.
func (w *Witness) Alive() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return !w.dead
}

// Kill simulates T's own loss: the lease share and journal close
// without reaping children or writing receipts. It is idempotent.
func (w *Witness) Kill() {
	w.mu.Lock()
	if w.dead {
		w.mu.Unlock()
		return
	}
	w.dead = true
	w.mu.Unlock()
	_ = w.lease.Close()
	_ = w.journal.Close()
}

// Close is the graceful teardown: owned children are aborted, then the
// lease and journal close. It is idempotent and safe after Kill.
func (w *Witness) Close() {
	w.mu.Lock()
	if w.dead {
		w.mu.Unlock()
		return
	}
	w.dead = true
	w.mu.Unlock()
	for _, e := range w.owner.Tree() {
		_ = w.owner.Abort(e.Child)
	}
	_ = w.lease.Close()
	_ = w.journal.Close()
}

// Live reports the number of retained N children.
func (w *Witness) Live() int { return w.owner.Live() }
