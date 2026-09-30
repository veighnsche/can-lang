package recovery

import "sync"

// Gate is a stop-admission latch. Cooperating work calls Admit before
// starting; the first Stop closes the gate and every later Admit fails
// with ErrAdmissionClosed. Drain stops its gate before reaping children,
// so cooperating admitters start no new work while owned resources
// drain; a non-cooperating spawner racing the drain leaves Live non-zero
// and the drain reports incomplete rather than silent success. A Gate is
// safe for concurrent use.
type Gate struct {
	mu      sync.Mutex
	stopped bool
}

// Admit reports whether new work may start.
func (g *Gate) Admit() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.stopped {
		return ErrAdmissionClosed
	}
	return nil
}

// Stop closes the gate. It is idempotent.
func (g *Gate) Stop() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.stopped = true
}

// Closed reports whether the gate is stopped.
func (g *Gate) Closed() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.stopped
}
