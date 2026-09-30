package dispatch

import (
	"fmt"
	"sync"
)

// MaxHandles caps live entries in one handle table.
const MaxHandles = 8192

// HandleKind names what a handle refers to. Kinds are checked on every use;
// passing a handle where another kind is required fails with ErrWrongKind.
type HandleKind string

const (
	HandleRun       HandleKind = "run"
	HandleCase      HandleKind = "case"
	HandleOperation HandleKind = "operation"
)

func validHandleKind(k HandleKind) bool {
	return k == HandleRun || k == HandleCase || k == HandleOperation
}

// Handle is a typed reference to a run, case or operation. The generation
// disambiguates reopen after close: a handle for an older generation is
// stale, never authority for the current one.
type Handle struct {
	Kind       HandleKind
	ID         string
	Generation uint64
}

type handleEntry struct {
	generation uint64
	closed     bool
}

// HandleTable issues generation-checked typed handles. Opening a live entry
// is idempotent; opening after close bumps the generation so earlier
// handles go stale. A HandleTable is safe for concurrent use.
type HandleTable struct {
	mu      sync.Mutex
	handles map[string]*handleEntry
}

// NewHandleTable returns an empty table.
func NewHandleTable() *HandleTable {
	return &HandleTable{handles: make(map[string]*handleEntry)}
}

func handleKey(kind HandleKind, id string) string { return string(kind) + "\x00" + id }

// Open returns the live handle for kind+id, creating it at generation 1 or
// reopening a closed entry at the next generation.
func (t *HandleTable) Open(kind HandleKind, id string) (Handle, error) {
	if !validHandleKind(kind) {
		return Handle{}, fmt.Errorf("%w: unknown handle kind %q", ErrInvalid, kind)
	}
	if !validID(id) {
		return Handle{}, fmt.Errorf("%w: malformed handle ID", ErrInvalid)
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	k := handleKey(kind, id)
	if e, ok := t.handles[k]; ok {
		if !e.closed {
			return Handle{Kind: kind, ID: id, Generation: e.generation}, nil
		}
		e.generation++
		e.closed = false
		return Handle{Kind: kind, ID: id, Generation: e.generation}, nil
	}
	if len(t.handles) >= MaxHandles {
		return Handle{}, fmt.Errorf("%w: handle table full", ErrCapacity)
	}
	t.handles[k] = &handleEntry{generation: 1}
	return Handle{Kind: kind, ID: id, Generation: 1}, nil
}

// Check verifies that h names the live entry of the wanted kind.
func (t *HandleTable) Check(h Handle, wantKind HandleKind) error {
	if h.Kind != wantKind {
		return fmt.Errorf("%w: handle kind %q is not %q", ErrWrongKind, h.Kind, wantKind)
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	e, ok := t.handles[handleKey(h.Kind, h.ID)]
	if !ok {
		return fmt.Errorf("%w: unknown handle", ErrNotFound)
	}
	if h.Generation != e.generation {
		return fmt.Errorf("%w: want generation %d", ErrStaleHandle, e.generation)
	}
	if e.closed {
		return fmt.Errorf("%w: handle closed", ErrClosedHandle)
	}
	return nil
}

// Close marks a handle closed; it is idempotent. Closing needs the current
// generation: a stale generation cannot close the live entry.
func (t *HandleTable) Close(h Handle) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	e, ok := t.handles[handleKey(h.Kind, h.ID)]
	if !ok {
		return fmt.Errorf("%w: unknown handle", ErrNotFound)
	}
	if h.Generation != e.generation {
		return fmt.Errorf("%w: want generation %d", ErrStaleHandle, e.generation)
	}
	e.closed = true
	return nil
}
