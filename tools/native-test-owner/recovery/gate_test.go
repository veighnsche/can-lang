package recovery

import (
	"errors"
	"sync"
	"testing"
)

func TestGateStopAdmission(t *testing.T) {
	var g Gate
	if g.Closed() {
		t.Fatalf("new gate reports closed")
	}
	if err := g.Admit(); err != nil {
		t.Fatalf("Admit before Stop: %v", err)
	}
	g.Stop()
	if !g.Closed() {
		t.Fatalf("gate open after Stop")
	}
	if err := g.Admit(); !errors.Is(err, ErrAdmissionClosed) {
		t.Fatalf("Admit after Stop: got %v, want ErrAdmissionClosed", err)
	}
	// Stop is idempotent.
	g.Stop()
	if err := g.Admit(); !errors.Is(err, ErrAdmissionClosed) {
		t.Fatalf("Admit after second Stop: got %v, want ErrAdmissionClosed", err)
	}
}

func TestGateConcurrent(t *testing.T) {
	var g Gate
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for k := 0; k < 50; k++ {
				_ = g.Admit()
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		g.Stop()
	}()
	wg.Wait()
	if !g.Closed() {
		t.Fatalf("gate open after concurrent Stop")
	}
	if err := g.Admit(); !errors.Is(err, ErrAdmissionClosed) {
		t.Fatalf("Admit after concurrent Stop: got %v, want ErrAdmissionClosed", err)
	}
}
