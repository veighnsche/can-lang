package dispatch

import (
	"errors"
	"fmt"
	"testing"
)

func TestHandleLifecycle(t *testing.T) {
	ht := NewHandleTable()
	kinds := []struct {
		name string
		kind HandleKind
		id   string
	}{
		{"run", HandleRun, "run-1"},
		{"case", HandleCase, "case-1"},
		{"operation", HandleOperation, "op-1"},
	}
	for _, k := range kinds {
		t.Run("open "+k.name, func(t *testing.T) {
			h, err := ht.Open(k.kind, k.id)
			if err != nil {
				t.Fatalf("Open: %v", err)
			}
			if h.Kind != k.kind || h.ID != k.id || h.Generation != 1 {
				t.Fatalf("handle = %+v, want gen 1", h)
			}
			// Opening a live entry is idempotent.
			again, err := ht.Open(k.kind, k.id)
			if err != nil || again != h {
				t.Fatalf("reopen = %+v,%v, want %+v", again, err, h)
			}
			if err := ht.Check(h, k.kind); err != nil {
				t.Fatalf("Check: %v", err)
			}
		})
	}

	checks := []struct {
		name string
		h    Handle
		want HandleKind
		err  error
	}{
		{"wrong kind run-as-case", Handle{Kind: HandleRun, ID: "run-1", Generation: 1}, HandleCase, ErrWrongKind},
		{"wrong kind op-as-run", Handle{Kind: HandleOperation, ID: "op-1", Generation: 1}, HandleRun, ErrWrongKind},
		{"unknown id", Handle{Kind: HandleRun, ID: "run-missing", Generation: 1}, HandleRun, ErrNotFound},
		{"unknown kind", Handle{Kind: HandleKind("bogus"), ID: "run-1", Generation: 1}, HandleRun, ErrWrongKind},
		{"stale generation", Handle{Kind: HandleRun, ID: "run-1", Generation: 7}, HandleRun, ErrStaleHandle},
	}
	for _, c := range checks {
		t.Run(c.name, func(t *testing.T) {
			if err := ht.Check(c.h, c.want); !errors.Is(err, c.err) {
				t.Fatalf("Check err = %v, want %v", err, c.err)
			}
		})
	}

	// Close invalidates; reopen bumps the generation and stales the old.
	h, err := ht.Open(HandleCase, "case-1")
	if err != nil {
		t.Fatal(err)
	}
	if err := ht.Close(h); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := ht.Close(h); err != nil {
		t.Fatalf("double Close: %v", err)
	}
	if err := ht.Check(h, HandleCase); !errors.Is(err, ErrClosedHandle) {
		t.Fatalf("Check after close err = %v, want ErrClosedHandle", err)
	}
	h2, err := ht.Open(HandleCase, "case-1")
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if h2.Generation != 2 {
		t.Fatalf("reopened generation = %d, want 2", h2.Generation)
	}
	if err := ht.Check(h, HandleCase); !errors.Is(err, ErrStaleHandle) {
		t.Fatalf("old generation err = %v, want ErrStaleHandle", err)
	}
	if err := ht.Close(h); !errors.Is(err, ErrStaleHandle) {
		t.Fatalf("stale Close err = %v, want ErrStaleHandle", err)
	}
	if err := ht.Close(Handle{Kind: HandleRun, ID: "run-missing", Generation: 1}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown Close err = %v, want ErrNotFound", err)
	}
}

func TestHandleInvalid(t *testing.T) {
	ht := NewHandleTable()
	if _, err := ht.Open(HandleKind("bogus"), "run-1"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("bad kind err = %v, want ErrInvalid", err)
	}
	if _, err := ht.Open(HandleRun, "bad id!"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("bad id err = %v, want ErrInvalid", err)
	}
}

func TestHandleCapacity(t *testing.T) {
	ht := NewHandleTable()
	for i := 0; i < MaxHandles; i++ {
		if _, err := ht.Open(HandleOperation, fmt.Sprintf("op-%d", i)); err != nil {
			t.Fatalf("fill %d: %v", i, err)
		}
	}
	if _, err := ht.Open(HandleOperation, "op-overflow"); !errors.Is(err, ErrCapacity) {
		t.Fatalf("overflow err = %v, want ErrCapacity", err)
	}
}
