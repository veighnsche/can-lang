// Executed K26 settlement-observer tests. The worker verified this behavior
// with a throwaway probe and removed it; the integrator keeps the maintained
// coverage here: journaled dispatch, deadline/driver/server ordering,
// unknown-blocks-cleanup, capable-grant cancellation, quiescence/fence,
// retained lease, acknowledged release, and the concurrent-dispatch pin.

package external

import (
	"fmt"
	"sync"
	"testing"

	"github.com/veighnsche/can-lang/tools/native-test-owner/journal"
)

func settlementObserver(t *testing.T) (*Observer, *SettlementObserver, journal.Owner) {
	t.Helper()
	j := journal.OpenMemory()
	r := newRegistry(t, j, 4242, "start-token-1")
	g, err := r.GrantNamespace("op-ns", "tenant-alpha", testTTL)
	if err != nil {
		t.Fatalf("GrantNamespace: %v", err)
	}
	o, err := ObserveNamespace(r, g.Token)
	if err != nil {
		t.Fatalf("ObserveNamespace: %v", err)
	}
	s, err := ObserveSettlement(o)
	if err != nil {
		t.Fatalf("ObserveSettlement: %v", err)
	}
	return o, s, r.Owner()
}

func TestObserveSettlementNil(t *testing.T) {
	if _, err := ObserveSettlement(nil); err == nil {
		t.Fatal("nil observer binds")
	} else {
		requireObserverErr(t, err, ObserverLayerEngine, ObserverCodeUnknownNamespace)
	}
}

func TestSettlementDispatchAtMostOnce(t *testing.T) {
	_, s, _ := settlementObserver(t)
	d, tok, err := s.Dispatch("op-w1", "w1", "SELECT 1", "postgres", testTTL)
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if d.TokenDigest == "" || d.StatementDigest == "" {
		t.Fatalf("dispatch lacks digests: %+v", d)
	}
	if _, _, err := s.Dispatch("op-w1b", "w1", "SELECT 2", "postgres", testTTL); err == nil {
		t.Fatal("work re-dispatch succeeds")
	} else {
		requireObserverErr(t, err, ObserverLayerDriver, ObserverCodeConnBusy)
	}
	if _, _, err := s.Dispatch("op-w1", "w1b", "SELECT 2", "postgres", testTTL); err == nil {
		t.Fatal("opID re-dispatch succeeds")
	} else {
		requireObserverErr(t, err, ObserverLayerDriver, ObserverCodeUnknownConn)
	}
	if _, _, err := s.Dispatch("op-bad!", "w1c", "SELECT 1", "postgres", testTTL); err == nil {
		t.Fatal("malformed opID dispatches")
	}
	if _, _, err := s.Dispatch("op-w1c", "w1c", "", "postgres", testTTL); err == nil {
		t.Fatal("empty statement dispatches")
	}
	if _, _, err := s.Dispatch("op-w1c", "w1c", "SELECT 1", "oracle", testTTL); err == nil {
		t.Fatal("unknown engine dispatches")
	}
	_ = tok
}

func TestSettlementConcurrentSameWorkDispatchesOnce(t *testing.T) {
	_, s, _ := settlementObserver(t)
	const racers = 8
	start := make(chan struct{})
	var wg sync.WaitGroup
	var mu sync.Mutex
	succeeded := 0
	for i := 0; i < racers; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			<-start
			if _, _, err := s.Dispatch(fmt.Sprintf("op-r%d", n), "same-work", "SELECT 1", "postgres", testTTL); err == nil {
				mu.Lock()
				succeeded++
				mu.Unlock()
			}
		}(i)
	}
	close(start)
	wg.Wait()
	if succeeded != 1 {
		t.Fatalf("same-work dispatches succeeded = %d, want exactly 1", succeeded)
	}
}

func TestSettlementDeadlineDriverServerOrder(t *testing.T) {
	_, s, _ := settlementObserver(t)
	_, tok, err := s.Dispatch("op-w1", "w1", "SELECT 1", "postgres", testTTL)
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if _, err := s.RecordDriverSettlement("op-w1", tok, "completed"); err == nil {
		t.Fatal("driver settles before deadline")
	} else {
		requireObserverErr(t, err, ObserverLayerDriver, ObserverCodeNoConversation)
	}
	if _, err := s.ObserveDeadline("op-w1", tok, -1); err == nil {
		t.Fatal("negative deadline observes")
	}
	dl, err := s.ObserveDeadline("op-w1", tok, 100)
	if err != nil {
		t.Fatalf("ObserveDeadline: %v", err)
	}
	if dl.DeadlineMs != 100 {
		t.Fatalf("deadline = %+v", dl)
	}
	if _, err := s.ObserveDeadline("op-w1", tok, 200); err == nil {
		t.Fatal("deadline re-observes")
	} else {
		requireObserverErr(t, err, ObserverLayerDriver, ObserverCodeConnBusy)
	}
	if _, err := s.RecordServerAck("op-w1", tok, "applied"); err == nil {
		t.Fatal("server acks before driver")
	} else {
		requireObserverErr(t, err, ObserverLayerDriver, ObserverCodeNoConversation)
	}
	drv, err := s.RecordDriverSettlement("op-w1", tok, "timed-out")
	if err != nil {
		t.Fatalf("RecordDriverSettlement: %v", err)
	}
	if drv.Outcome != "timed-out" || drv.ServerEffect != "unresolved" {
		t.Fatalf("driver fact = %+v", drv)
	}
	if _, err := s.RecordDriverSettlement("op-w1", tok, "completed"); err == nil {
		t.Fatal("driver re-settles")
	}
	ack, err := s.RecordServerAck("op-w1", tok, "applied")
	if err != nil {
		t.Fatalf("RecordServerAck: %v", err)
	}
	if ack.Effect != "applied" {
		t.Fatalf("ack = %+v", ack)
	}
	if _, err := s.RecordServerAck("op-w1", tok, "absent"); err == nil {
		t.Fatal("server re-acks")
	} else {
		requireObserverErr(t, err, ObserverLayerDriver, ObserverCodeConnBusy)
	}
}

func TestSettlementUnknownBlocksCleanup(t *testing.T) {
	_, s, self := settlementObserver(t)
	_, tok, err := s.Dispatch("op-w1", "w1", "SELECT 1", "postgres", testTTL)
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if _, err := s.ObserveDeadline("op-w1", tok, 10); err != nil {
		t.Fatalf("ObserveDeadline: %v", err)
	}
	if _, err := s.RecordDriverSettlement("op-w1", tok, "timed-out"); err != nil {
		t.Fatalf("RecordDriverSettlement: %v", err)
	}
	if _, err := s.RecordServerAck("op-w1", tok, "unknown"); err != nil {
		t.Fatalf("RecordServerAck: %v", err)
	}
	// The fence surveys even with an unknown effect — and still the
	// release refuses.
	if _, err := s.QuiesceEngine("postgres"); err != nil {
		t.Fatalf("QuiesceEngine: %v", err)
	}
	fence, err := s.Fence("op-w1", tok)
	if err != nil {
		t.Fatalf("Fence: %v", err)
	}
	if fence.Settles != "nothing" {
		t.Fatalf("fence settles: %+v", fence)
	}
	if _, err := s.Release("op-w1", tok, self); err == nil {
		t.Fatal("release with unknown server effect succeeds")
	} else {
		requireObserverErr(t, err, ObserverLayerDriver, ObserverCodeConnBusy)
	}
}

func TestSettlementReleaseNeedsEveryLeg(t *testing.T) {
	_, s, self := settlementObserver(t)
	_, tok, err := s.Dispatch("op-w1", "w1", "SELECT 1", "postgres", testTTL)
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if _, err := s.Release("op-w1", tok, self); err == nil {
		t.Fatal("release without driver succeeds")
	} else {
		requireObserverErr(t, err, ObserverLayerDriver, ObserverCodeConnBusy)
	}
	if _, err := s.ObserveDeadline("op-w1", tok, 10); err != nil {
		t.Fatalf("ObserveDeadline: %v", err)
	}
	if _, err := s.RecordDriverSettlement("op-w1", tok, "completed"); err != nil {
		t.Fatalf("RecordDriverSettlement: %v", err)
	}
	if _, err := s.Release("op-w1", tok, self); err == nil {
		t.Fatal("release without server ack succeeds")
	} else {
		requireObserverErr(t, err, ObserverLayerDriver, ObserverCodeNoConversation)
	}
	if _, err := s.RecordServerAck("op-w1", tok, "absent"); err != nil {
		t.Fatalf("RecordServerAck: %v", err)
	}
	if _, err := s.Release("op-w1", tok, self); err == nil {
		t.Fatal("release without fence succeeds")
	} else {
		requireObserverErr(t, err, ObserverLayerDriver, ObserverCodeNoConversation)
	}
}

func TestSettlementCancelCapableGrantOnly(t *testing.T) {
	_, s, _ := settlementObserver(t)
	_, tok, err := s.Dispatch("op-w1", "w1", "SELECT 1", "postgres", testTTL)
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	for _, engine := range []string{"sqlite", "mysql"} {
		if _, _, err := s.IssueCancelGrant(engine); err == nil {
			t.Fatalf("cancel grant issued for %s", engine)
		} else {
			requireObserverErr(t, err, ObserverLayerSQL, ObserverCodeBadStatement)
		}
	}
	gf, grant, err := s.IssueCancelGrant("postgres")
	if err != nil {
		t.Fatalf("IssueCancelGrant: %v", err)
	}
	if gf.GrantDigest == "" {
		t.Fatalf("grant lacks digest: %+v", gf)
	}
	if _, err := s.RequestCancel("op-w1", tok, "grant-invented"); err == nil {
		t.Fatal("invented grant cancels")
	} else {
		requireObserverErr(t, err, ObserverLayerDriver, ObserverCodeForgedToken)
	}
	cf, err := s.RequestCancel("op-w1", tok, grant)
	if err != nil {
		t.Fatalf("RequestCancel: %v", err)
	}
	if cf.Proves != "nothing-about-server" {
		t.Fatalf("cancel proves: %+v", cf)
	}
	if _, err := s.RequestCancel("op-w1", tok, grant); err == nil {
		t.Fatal("cancel repeats")
	} else {
		requireObserverErr(t, err, ObserverLayerDriver, ObserverCodeConnBusy)
	}
	// Once the server has acknowledged, a cancel is stale and refuses.
	if _, err := s.ObserveDeadline("op-w1", tok, 5); err != nil {
		t.Fatalf("ObserveDeadline: %v", err)
	}
	if _, err := s.RecordDriverSettlement("op-w1", tok, "completed"); err != nil {
		t.Fatalf("RecordDriverSettlement: %v", err)
	}
	if _, err := s.RecordServerAck("op-w1", tok, "applied"); err != nil {
		t.Fatalf("RecordServerAck: %v", err)
	}
	_, tok2, err := s.Dispatch("op-w2", "w2", "SELECT 2", "postgres", testTTL)
	if err != nil {
		t.Fatalf("Dispatch w2: %v", err)
	}
	if _, err := s.ObserveDeadline("op-w2", tok2, 5); err != nil {
		t.Fatalf("ObserveDeadline: %v", err)
	}
	if _, err := s.RecordDriverSettlement("op-w2", tok2, "completed"); err != nil {
		t.Fatalf("RecordDriverSettlement: %v", err)
	}
	if _, err := s.RecordServerAck("op-w2", tok2, "applied"); err != nil {
		t.Fatalf("RecordServerAck: %v", err)
	}
	if _, err := s.RequestCancel("op-w2", tok2, grant); err == nil {
		t.Fatal("stale cancel succeeds")
	} else {
		requireObserverErr(t, err, ObserverLayerDriver, ObserverCodeConnBusy)
	}
}

func TestSettlementQuiesceFenceAdapter(t *testing.T) {
	_, s, _ := settlementObserver(t)
	_, tok, err := s.Dispatch("op-w1", "w1", "SELECT 1", "postgres", testTTL)
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if _, err := s.Fence("op-w1", tok); err == nil {
		t.Fatal("fence before quiesce succeeds")
	} else {
		requireObserverErr(t, err, ObserverLayerDriver, ObserverCodeNoConversation)
	}
	live, err := s.QuiesceEngine("postgres")
	if err != nil {
		t.Fatalf("QuiesceEngine: %v", err)
	}
	if len(live) != 1 || live[0] != "op-w1" {
		t.Fatalf("live = %v", live)
	}
	again, err := s.QuiesceEngine("postgres")
	if err != nil {
		t.Fatalf("QuiesceEngine repeat: %v", err)
	}
	if len(again) != 1 {
		t.Fatalf("repeat quiesce live = %v", again)
	}
	if _, _, err := s.Dispatch("op-w2", "w2", "SELECT 2", "postgres", testTTL); err == nil {
		t.Fatal("dispatch after quiesce succeeds")
	}
	if _, err := s.Fence("op-w1", tok); err != nil {
		t.Fatalf("Fence: %v", err)
	}
	if _, err := s.Fence("op-w1", tok); err != nil {
		t.Fatalf("Fence repeat: %v", err)
	}
}

func TestSettlementFullDrainAndClose(t *testing.T) {
	o, s, self := settlementObserver(t)
	_, tok, err := s.Dispatch("op-w1", "w1", "SELECT 1", "postgres", testTTL)
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	lease, err := s.Lease("op-w1")
	if err != nil {
		t.Fatalf("Lease: %v", err)
	}
	if !lease.Retained || lease.Namespace == "" {
		t.Fatalf("lease = %+v", lease)
	}
	if err := s.Close(); err == nil {
		t.Fatal("close with dispatched work succeeds")
	}
	if _, err := s.ObserveDeadline("op-w1", tok, 50); err != nil {
		t.Fatalf("ObserveDeadline: %v", err)
	}
	if _, err := s.RecordDriverSettlement("op-w1", tok, "timed-out"); err != nil {
		t.Fatalf("RecordDriverSettlement: %v", err)
	}
	if _, err := s.RecordServerAck("op-w1", tok, "applied"); err != nil {
		t.Fatalf("RecordServerAck: %v", err)
	}
	if _, err := s.QuiesceEngine("postgres"); err != nil {
		t.Fatalf("QuiesceEngine: %v", err)
	}
	if _, err := s.Fence("op-w1", tok); err != nil {
		t.Fatalf("Fence: %v", err)
	}
	ack, err := s.Release("op-w1", tok, self)
	if err != nil {
		t.Fatalf("Release: %v", err)
	}
	if !ack.Released {
		t.Fatalf("ack = %+v", ack)
	}
	again, err := s.ReleaseAck("op-w1")
	if err != nil {
		t.Fatalf("ReleaseAck: %v", err)
	}
	if again.AckDigest != ack.AckDigest {
		t.Fatal("ack not stable across re-reads")
	}
	lease, err = s.Lease("op-w1")
	if err != nil {
		t.Fatalf("Lease: %v", err)
	}
	if lease.Retained {
		t.Fatal("lease retained after release")
	}
	if err := s.Close(); err != nil {
		t.Fatalf("settlement Close: %v", err)
	}
	if err := o.Close(); err != nil {
		t.Fatalf("observer Close: %v", err)
	}
}
