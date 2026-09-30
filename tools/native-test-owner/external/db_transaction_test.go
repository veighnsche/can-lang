// Executed K24 transaction-observer tests. The worker verified this behavior
// with a throwaway probe and removed it; the integrator keeps the maintained
// coverage here: journaled entry, token-bound identity, wrong-handle and
// swapped-actor comparison, facts-only settlement, acknowledged release,
// close discipline, and the concurrent-entry pin.

package external

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/veighnsche/can-lang/tools/native-test-owner/journal"
)

func txnObserver(t *testing.T) (*Registry, *Observer, *TxnObserver, journal.Owner) {
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
	tx, err := ObserveTransactions(o)
	if err != nil {
		t.Fatalf("ObserveTransactions: %v", err)
	}
	return r, o, tx, r.Owner()
}

func requireObserverErr(t *testing.T, err error, layer ObserverLayer, code string) {
	t.Helper()
	if err == nil {
		t.Fatalf("want [%s:%s], got nil", layer, code)
	}
	var oerr *ObserverError
	if !errors.As(err, &oerr) {
		t.Fatalf("want *ObserverError, got %T (%v)", err, err)
	}
	if oerr.Layer != layer || oerr.Code != code {
		t.Fatalf("want [%s:%s], got [%s:%s] (%v)", layer, code, oerr.Layer, oerr.Code, err)
	}
	if !strings.HasPrefix(err.Error(), "["+string(layer)+":"+code+"]") {
		t.Fatalf("error lacks layer prefix: %v", err)
	}
}

func TestObserveTransactionsNil(t *testing.T) {
	if _, err := ObserveTransactions(nil); err == nil {
		t.Fatal("nil observer binds")
	} else {
		requireObserverErr(t, err, ObserverLayerEngine, ObserverCodeUnknownNamespace)
	}
}

func TestTxnCallbackEntry(t *testing.T) {
	_, _, tx, _ := txnObserver(t)
	e1, tok1, err := tx.EnterCallback("op-alpha", "alpha", testTTL)
	if err != nil {
		t.Fatalf("EnterCallback: %v", err)
	}
	if e1.EntrySeq != 0 || e1.Actor != "alpha" {
		t.Fatalf("entry = %+v", e1)
	}
	if e1.TokenDigest == "" || strings.Contains(e1.TokenDigest, tok1) {
		t.Fatalf("entry leaks or lacks digest: %+v", e1)
	}
	e2, _, err := tx.EnterCallback("op-beta", "beta", testTTL)
	if err != nil {
		t.Fatalf("EnterCallback beta: %v", err)
	}
	if e2.EntrySeq != 1 {
		t.Fatalf("beta seq = %d", e2.EntrySeq)
	}
	// Same actor twice refuses even under a fresh operation.
	if _, _, err := tx.EnterCallback("op-alpha-2", "alpha", testTTL); err == nil {
		t.Fatal("actor re-entry succeeds")
	} else {
		requireObserverErr(t, err, ObserverLayerDriver, ObserverCodeConnBusy)
	}
	// Same operation twice refuses even for a fresh actor.
	if _, _, err := tx.EnterCallback("op-alpha", "gamma", testTTL); err == nil {
		t.Fatal("operation re-entry succeeds")
	} else {
		requireObserverErr(t, err, ObserverLayerDriver, ObserverCodeUnknownConn)
	}
	if _, _, err := tx.EnterCallback("op-bad actor!", "bad", testTTL); err == nil {
		t.Fatal("malformed opID enters")
	}
	if _, _, err := tx.EnterCallback("op-ok", "bad actor!", testTTL); err == nil {
		t.Fatal("malformed actor enters")
	} else {
		requireObserverErr(t, err, ObserverLayerSQL, ObserverCodeBadStatement)
	}
	// Swapped actor: beta's token presented for alpha's handle refuses.
	if _, err := tx.RecordInsert("op-alpha", "bogus", "42"); err == nil {
		t.Fatal("forged token records")
	} else {
		requireObserverErr(t, err, ObserverLayerDriver, ObserverCodeForgedToken)
	}
}

func TestTxnActorTableCapped(t *testing.T) {
	_, _, tx, _ := txnObserver(t)
	for i := 0; i < TxnMaxActors; i++ {
		op := fmt.Sprintf("op-%d", i)
		actor := fmt.Sprintf("actor-%d", i)
		if _, _, err := tx.EnterCallback(op, actor, testTTL); err != nil {
			t.Fatalf("EnterCallback %d: %v", i, err)
		}
	}
	if _, _, err := tx.EnterCallback("op-over", "extra", testTTL); err == nil {
		t.Fatal("actor table overfills")
	} else {
		requireObserverErr(t, err, ObserverLayerEngine, ObserverCodeCapacity)
	}
}

func TestTxnConcurrentSameActorEntersOnce(t *testing.T) {
	_, _, tx, _ := txnObserver(t)
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
			if _, _, err := tx.EnterCallback(fmt.Sprintf("op-r%d", n), "same-actor", testTTL); err == nil {
				mu.Lock()
				succeeded++
				mu.Unlock()
			}
		}(i)
	}
	close(start)
	wg.Wait()
	if succeeded != 1 {
		t.Fatalf("same-actor entries succeeded = %d, want exactly 1", succeeded)
	}
}

func TestTxnInsertCompare(t *testing.T) {
	_, _, tx, _ := txnObserver(t)
	_, tokA, err := tx.EnterCallback("op-alpha", "alpha", testTTL)
	if err != nil {
		t.Fatalf("EnterCallback: %v", err)
	}
	_, tokB, err := tx.EnterCallback("op-beta", "beta", testTTL)
	if err != nil {
		t.Fatalf("EnterCallback: %v", err)
	}
	if _, err := tx.LastInsertID("op-alpha"); err == nil {
		t.Fatal("missing id reads")
	} else {
		requireObserverErr(t, err, ObserverLayerDriver, ObserverCodeNoConversation)
	}
	stored, err := tx.RecordInsert("op-alpha", tokA, "42")
	if err != nil {
		t.Fatalf("RecordInsert: %v", err)
	}
	other, err := tx.RecordInsert("op-beta", tokB, "42")
	if err != nil {
		t.Fatalf("RecordInsert beta: %v", err)
	}
	// The id digest covers the value only: identical values agree even
	// across handles, so handle binding rests on TokenDigest alone.
	if stored.IDDigest != other.IDDigest {
		t.Fatal("same-value id digests diverge")
	}
	if stored.TokenDigest == other.TokenDigest {
		t.Fatal("distinct handles share a digest")
	}
	// Swapped actor with the coincidentally correct value: identity only.
	cmp := CompareLastInsertID(stored, "beta", other.TokenDigest, other.IDDigest)
	if cmp.Match || len(cmp.Mismatches) != 1 || cmp.Mismatches[0] != -1 {
		t.Fatalf("swapped actor = %+v", cmp)
	}
	// Wrong handle digest, right value: identity only.
	cmp = CompareLastInsertID(stored, "alpha", other.TokenDigest, stored.IDDigest)
	if cmp.Match || len(cmp.Mismatches) != 1 || cmp.Mismatches[0] != -1 {
		t.Fatalf("wrong handle = %+v", cmp)
	}
	// Right handle, wrong value: value only.
	cmp = CompareLastInsertID(stored, "alpha", stored.TokenDigest, other.IDDigest+"x")
	if cmp.Match || len(cmp.Mismatches) != 1 || cmp.Mismatches[0] != 0 {
		t.Fatalf("wrong value = %+v", cmp)
	}
	// Both: both markers, identity first.
	cmp = CompareLastInsertID(stored, "beta", other.TokenDigest, "sha256:dead")
	if cmp.Match || len(cmp.Mismatches) != 2 || cmp.Mismatches[0] != -1 || cmp.Mismatches[1] != 0 {
		t.Fatalf("both wrong = %+v", cmp)
	}
	// Exact claim matches.
	cmp = CompareLastInsertID(stored, "alpha", stored.TokenDigest, stored.IDDigest)
	if !cmp.Match {
		t.Fatalf("exact claim = %+v", cmp)
	}
}

func TestTxnSettlementFactsOnly(t *testing.T) {
	_, _, tx, _ := txnObserver(t)
	_, tok, err := tx.EnterCallback("op-alpha", "alpha", testTTL)
	if err != nil {
		t.Fatalf("EnterCallback: %v", err)
	}
	if _, err := tx.Settlement("op-alpha"); err == nil {
		t.Fatal("missing settlement reads")
	} else {
		requireObserverErr(t, err, ObserverLayerDriver, ObserverCodeNoConversation)
	}
	if _, err := tx.RecordSettlement("op-alpha", tok, "maybe", "postgres"); err == nil {
		t.Fatal("unknown outcome records")
	}
	if _, err := tx.RecordSettlement("op-alpha", tok, "committed", "oracle"); err == nil {
		t.Fatal("unknown engine records")
	}
	s1, err := tx.RecordSettlement("op-alpha", tok, "rolled-back", "postgres")
	if err != nil {
		t.Fatalf("RecordSettlement: %v", err)
	}
	if _, err := tx.RecordSettlement("op-alpha", tok, "committed", "postgres"); err == nil {
		t.Fatal("double settle records")
	} else {
		requireObserverErr(t, err, ObserverLayerDriver, ObserverCodeConnBusy)
	}
	if _, err := tx.RecordInsert("op-alpha", tok, "7"); err == nil {
		t.Fatal("insert after settle records")
	} else {
		requireObserverErr(t, err, ObserverLayerDriver, ObserverCodeNoConversation)
	}
	back, err := tx.Settlement("op-alpha")
	if err != nil {
		t.Fatalf("Settlement: %v", err)
	}
	if back.Digest != s1.Digest || back.Outcome != "rolled-back" || back.Engine != "postgres" {
		t.Fatalf("settlement = %+v", back)
	}
}

func TestTxnReleaseAfterSettlement(t *testing.T) {
	_, o, tx, self := txnObserver(t)
	_, tok, err := tx.EnterCallback("op-alpha", "alpha", testTTL)
	if err != nil {
		t.Fatalf("EnterCallback: %v", err)
	}
	if _, err := tx.Release("op-alpha", tok, self); err == nil {
		t.Fatal("release before settle succeeds")
	} else {
		requireObserverErr(t, err, ObserverLayerDriver, ObserverCodeConnBusy)
	}
	if _, err := tx.RecordSettlement("op-alpha", tok, "committed", "sqlite"); err != nil {
		t.Fatalf("RecordSettlement: %v", err)
	}
	ack, err := tx.Release("op-alpha", tok, self)
	if err != nil {
		t.Fatalf("Release: %v", err)
	}
	if !ack.Released || ack.AckDigest == "" {
		t.Fatalf("ack = %+v", ack)
	}
	again, err := tx.ReleaseAck("op-alpha")
	if err != nil {
		t.Fatalf("ReleaseAck: %v", err)
	}
	if again.AckDigest != ack.AckDigest {
		t.Fatal("ack not stable across re-reads")
	}
	if _, err := tx.RecordSettlement("op-alpha", tok, "committed", "sqlite"); err == nil {
		t.Fatal("settle after release records")
	}
	// Full drain: transaction observer, then the watched namespace pins.
	if err := tx.Close(); err != nil {
		t.Fatalf("tx Close: %v", err)
	}
	if err := o.Close(); err != nil {
		t.Fatalf("observer Close: %v", err)
	}
}

func TestTxnCloseDiscipline(t *testing.T) {
	_, _, tx, _ := txnObserver(t)
	if _, _, err := tx.EnterCallback("op-alpha", "alpha", testTTL); err != nil {
		t.Fatalf("EnterCallback: %v", err)
	}
	if err := tx.Close(); err == nil {
		t.Fatal("close with entered actor succeeds")
	} else {
		requireObserverErr(t, err, ObserverLayerEngine, ObserverCodeClosedHandle)
	}
}
