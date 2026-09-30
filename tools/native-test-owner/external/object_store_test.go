// Executed K27 store-observer tests. The worker verified this behavior with
// a scratch test and deleted it; the integrator keeps the maintained
// coverage here: admission, sessions, writes, escapes, pagination, the
// four seal negatives with a full seal, and close discipline.

package external

import (
	"errors"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/tools/native-test-owner/journal"
)

func storeObserver(t *testing.T, prefix string) (*Registry, *StoreObserver, journal.Owner) {
	t.Helper()
	j := journal.OpenMemory()
	r := newRegistry(t, j, 4242, "start-token-1")
	g, err := r.GrantPrefix("op-prefix", prefix, testTTL)
	if err != nil {
		t.Fatalf("GrantPrefix: %v", err)
	}
	o, err := ObservePrefix(r, g.Token)
	if err != nil {
		t.Fatalf("ObservePrefix: %v", err)
	}
	return r, o, r.Owner()
}

func requireStoreErr(t *testing.T, err error, layer StoreLayer, code string) {
	t.Helper()
	if err == nil {
		t.Fatalf("want [%s:%s], got nil", layer, code)
	}
	var serr *StoreError
	if !errors.As(err, &serr) {
		t.Fatalf("want *StoreError, got %T (%v)", err, err)
	}
	if serr.Layer != layer || serr.Code != code {
		t.Fatalf("want [%s:%s], got [%s:%s] (%v)", layer, code, serr.Layer, serr.Code, err)
	}
	if !strings.HasPrefix(err.Error(), "["+string(layer)+":"+code+"]") {
		t.Fatalf("error lacks layer prefix: %v", err)
	}
}

func TestObservePrefixAdmission(t *testing.T) {
	j := journal.OpenMemory()
	r := newRegistry(t, j, 4242, "start-token-1")
	if _, err := ObservePrefix(nil, "x"); err == nil {
		t.Fatal("nil registry binds")
	} else {
		requireStoreErr(t, err, StoreLayerEngine, StoreCodeUnknownPrefix)
	}
	if _, err := ObservePrefix(r, "forged-token"); err == nil {
		t.Fatal("forged token binds")
	} else {
		requireStoreErr(t, err, StoreLayerEngine, StoreCodeUnknownPrefix)
	}
	ns, err := r.GrantNamespace("op-ns", "tenant-alpha", testTTL)
	if err != nil {
		t.Fatalf("GrantNamespace: %v", err)
	}
	if _, err := ObservePrefix(r, ns.Token); err == nil {
		t.Fatal("namespace grant binds as prefix")
	} else {
		requireStoreErr(t, err, StoreLayerEngine, StoreCodeUnknownPrefix)
	}
	// The observed prefix is the grant's own target; the caller supplies
	// no prefix string at all.
	g, err := r.GrantPrefix("op-p", "backups/daily/2026-09-30", testTTL)
	if err != nil {
		t.Fatalf("GrantPrefix: %v", err)
	}
	o, err := ObservePrefix(r, g.Token)
	if err != nil {
		t.Fatalf("ObservePrefix: %v", err)
	}
	rc, err := o.Receipt()
	if err != nil {
		t.Fatalf("Receipt: %v", err)
	}
	if rc.Prefix != "backups/daily/2026-09-30" {
		t.Fatalf("receipt prefix = %q", rc.Prefix)
	}
	if strings.Contains(rc.HandleDigest, g.Token) || rc.HandleDigest == "" {
		t.Fatalf("receipt leaks or lacks handle digest: %+v", rc)
	}
}

func TestObservePrefixRevokedGrantStopsReceipts(t *testing.T) {
	now := int64(1_000_000)
	cfg := testConfig()
	cfg.Clock = func() int64 { return now }
	j := journal.OpenMemory()
	r, err := New(j, 4242, "start-token-1", cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	g, err := r.GrantPrefix("op-p", "backups/daily", 1000)
	if err != nil {
		t.Fatalf("GrantPrefix: %v", err)
	}
	o, err := ObservePrefix(r, g.Token)
	if err != nil {
		t.Fatalf("ObservePrefix: %v", err)
	}
	now += 1001
	if _, err := o.Receipt(); err == nil {
		t.Fatal("receipt under expired grant succeeds")
	} else {
		requireStoreErr(t, err, StoreLayerEngine, StoreCodeUnknownPrefix)
	}
}

func TestStoreSessionDiscipline(t *testing.T) {
	_, o, self := storeObserver(t, "backups/daily")
	sess, tok, err := o.OpenSession("op-s", "sess-1")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	if sess.TokenDigest == "" || strings.Contains(sess.TokenDigest, tok) {
		t.Fatalf("session fact leaks or lacks digest: %+v", sess)
	}
	if _, _, err := o.OpenSession("op-s", "sess-1"); err == nil {
		t.Fatal("duplicate opID pins twice")
	}
	if err := o.CloseSession("op-s", "forged", self); err == nil {
		t.Fatal("forged token closes")
	} else {
		requireStoreErr(t, err, StoreLayerDriver, StoreCodeForgedToken)
	}
	foreign := journal.Owner{PID: 9999, StartToken: "other"}
	if err := o.CloseSession("op-s", tok, foreign); err == nil {
		t.Fatal("foreign identity closes")
	} else {
		requireStoreErr(t, err, StoreLayerEngine, StoreCodeWrongOwner)
	}
	if err := o.CloseSession("op-s", tok, self); err != nil {
		t.Fatalf("CloseSession: %v", err)
	}
	if _, err := o.Put("op-s", tok, "backups/daily/k", []byte{1}); err == nil {
		t.Fatal("released session still writes")
	} else {
		requireStoreErr(t, err, StoreLayerDriver, StoreCodeUnknownSess)
	}
}

func TestStoreSessionTableCountsLiveOnly(t *testing.T) {
	_, o, self := storeObserver(t, "backups/daily")
	toks := map[string]string{}
	for i := 0; i < MaxStoreSessions; i++ {
		op := strings.Repeat("s", 3) + string(rune('a'+i))
		_, tok, err := o.OpenSession(op, "n"+string(rune('a'+i)))
		if err != nil {
			t.Fatalf("OpenSession %d: %v", i, err)
		}
		toks[op] = tok
	}
	if _, _, err := o.OpenSession("op-full", "extra"); err == nil {
		t.Fatal("session table overfills")
	} else {
		requireStoreErr(t, err, StoreLayerEngine, StoreCodeCapacity)
	}
	// Released pins free table room: the cap counts live sessions only.
	for op, tok := range toks {
		if err := o.CloseSession(op, tok, self); err != nil {
			t.Fatalf("CloseSession %s: %v", op, err)
		}
	}
	if _, _, err := o.OpenSession("op-again", "fresh"); err != nil {
		t.Fatalf("OpenSession after drain: %v", err)
	}
}

func TestStoreWriteSettlement(t *testing.T) {
	_, o, _ := storeObserver(t, "backups/daily")
	_, tok, err := o.OpenSession("op-s", "sess-1")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	payload := []byte{1, 2, 3}
	wf, err := o.Put("op-s", tok, "backups/daily/k1", payload)
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	payload[0] = 99 // staging copies; caller mutation cannot corrupt
	if _, err := o.Get("op-s", tok, "backups/daily/k1"); err == nil {
		t.Fatal("unsettled write is visible")
	} else {
		requireStoreErr(t, err, StoreLayerStore, StoreCodeUnknownKey)
	}
	if _, err := o.Put("op-s", tok, "backups/daily/k1", []byte{4}); err == nil {
		t.Fatal("second unsettled write for key accepted")
	} else {
		requireStoreErr(t, err, StoreLayerStore, StoreCodeWriteBusy)
	}
	got, err := o.SettleWrite("op-s", tok, wf.WriteID)
	if err != nil {
		t.Fatalf("SettleWrite: %v", err)
	}
	if got.Size != 3 {
		t.Fatalf("settled size = %d", got.Size)
	}
	if _, err := o.SettleWrite("op-s", tok, wf.WriteID); err == nil {
		t.Fatal("double settle succeeds")
	} else {
		requireStoreErr(t, err, StoreLayerStore, StoreCodeWriteSettled)
	}
	if _, err := o.SettleWrite("op-s", tok, "w999"); err == nil {
		t.Fatal("unknown write settles")
	} else {
		requireStoreErr(t, err, StoreLayerStore, StoreCodeWriteUnknown)
	}
}

func TestStoreKeysStayInsideGrant(t *testing.T) {
	_, o, _ := storeObserver(t, "backups/daily/2026-09-30")
	_, tok, err := o.OpenSession("op-s", "sess-1")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	for _, key := range []string{
		"backups/daily",             // grant parent
		"backups/daily/2026-09-29",  // sibling date
		"backups/daily/2026-09-30x", // shared stem, not under edge
		"artifacts/k",               // another declared prefix
		"backups/daily/2026-09-30",  // the edge itself
	} {
		if _, err := o.Put("op-s", tok, key, []byte{1}); err == nil {
			t.Fatalf("key %q admitted", key)
		} else if se := err.(*StoreError); se.Code != StoreCodeKeyEscapes && se.Code != StoreCodeBadKey {
			t.Fatalf("key %q: code = %s", key, se.Code)
		}
	}
	for _, key := range []string{"", "/abs", "a//b", "a/./b", "a/../b", "x/"} {
		if _, err := o.Put("op-s", tok, key, []byte{1}); err == nil {
			t.Fatalf("malformed key %q admitted", key)
		} else {
			requireStoreErr(t, err, StoreLayerStore, StoreCodeBadKey)
		}
	}
	rc, err := o.Receipt()
	if err != nil {
		t.Fatalf("Receipt: %v", err)
	}
	if rc.Objects != 0 || rc.Pending != 0 {
		t.Fatalf("rejected keys had effect: %+v", rc)
	}
}

func TestStorePaginationWalk(t *testing.T) {
	_, o, _ := storeObserver(t, "backups/daily")
	_, tok, err := o.OpenSession("op-s", "sess-1")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	for _, k := range []string{"backups/daily/a", "backups/daily/b", "backups/daily/c"} {
		wf, err := o.Put("op-s", tok, k, []byte{1})
		if err != nil {
			t.Fatalf("Put %s: %v", k, err)
		}
		if _, err := o.SettleWrite("op-s", tok, wf.WriteID); err != nil {
			t.Fatalf("SettleWrite %s: %v", k, err)
		}
	}
	p1, cont, err := o.List("op-s", tok, 2, "")
	if err != nil {
		t.Fatalf("List p1: %v", err)
	}
	if p1.Complete || p1.Count != 2 || cont == "" {
		t.Fatalf("p1 = %+v cont=%q", p1, cont)
	}
	if _, _, err := o.List("op-s", tok, 2, "forged"); err == nil {
		t.Fatal("forged continuation lists")
	} else {
		requireStoreErr(t, err, StoreLayerDriver, StoreCodeForgedCont)
	}
	p2, cont2, err := o.List("op-s", tok, 2, cont)
	if err != nil {
		t.Fatalf("List p2: %v", err)
	}
	if !p2.Complete || p2.Count != 1 || cont2 != "" {
		t.Fatalf("p2 = %+v cont=%q", p2, cont2)
	}
	if _, _, err := o.List("op-s", tok, 2, cont); err == nil {
		t.Fatal("consumed continuation replays")
	} else {
		requireStoreErr(t, err, StoreLayerDriver, StoreCodeForgedCont)
	}
	// A mutation stales an open continuation rather than shifting the walk.
	p3, cont3, err := o.List("op-s", tok, 1, "")
	if err != nil || p3.Complete {
		t.Fatalf("List p3: %+v %v", p3, err)
	}
	wf, err := o.Put("op-s", tok, "backups/daily/d", []byte{1})
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if _, err := o.SettleWrite("op-s", tok, wf.WriteID); err != nil {
		t.Fatalf("SettleWrite: %v", err)
	}
	if _, _, err := o.List("op-s", tok, 1, cont3); err == nil {
		t.Fatal("stale continuation lists")
	} else {
		requireStoreErr(t, err, StoreLayerDriver, StoreCodeStaleCont)
	}
}

func TestStoreSealRequiresAllFour(t *testing.T) {
	_, o, self := storeObserver(t, "backups/daily")
	_, tok, err := o.OpenSession("op-s", "sess-1")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	// 1. live session blocks first.
	if _, err := o.SealCleanup(self); err == nil {
		t.Fatal("seal with live session")
	} else {
		requireStoreErr(t, err, StoreLayerEngine, StoreCodeSessionLive)
	}
	if err := o.CloseSession("op-s", tok, self); err != nil {
		t.Fatalf("CloseSession: %v", err)
	}
	// Reopen for the remaining stages.
	_, tok2, err := o.OpenSession("op-s2", "sess-2")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	// 2. pending write blocks.
	wf, err := o.Put("op-s2", tok2, "backups/daily/k", []byte{7})
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if err := o.CloseSession("op-s2", tok2, self); err != nil {
		t.Fatalf("CloseSession: %v", err)
	}
	if _, err := o.SealCleanup(self); err == nil {
		t.Fatal("seal with pending write")
	} else {
		requireStoreErr(t, err, StoreLayerEngine, StoreCodePending)
	}
	_, tok3, err := o.OpenSession("op-s3", "sess-3")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	if _, err := o.SettleWrite("op-s3", tok3, wf.WriteID); err != nil {
		t.Fatalf("SettleWrite: %v", err)
	}
	if err := o.CloseSession("op-s3", tok3, self); err != nil {
		t.Fatalf("CloseSession: %v", err)
	}
	// 3. live objects block.
	if _, err := o.SealCleanup(self); err == nil {
		t.Fatal("seal with live objects")
	} else {
		requireStoreErr(t, err, StoreLayerEngine, StoreCodeKeysRemain)
	}
	_, tok4, err := o.OpenSession("op-s4", "sess-4")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	if err := o.Delete("op-s4", tok4, "backups/daily/k"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := o.CloseSession("op-s4", tok4, self); err != nil {
		t.Fatalf("CloseSession: %v", err)
	}
	// 4. no terminal scan at the current generation blocks.
	if _, err := o.SealCleanup(self); err == nil {
		t.Fatal("seal without terminal scan")
	} else {
		requireStoreErr(t, err, StoreLayerEngine, StoreCodeScanOpen)
	}
	// Terminal empty scan, drain the session, seal.
	_, tok5, err := o.OpenSession("op-s5", "sess-5")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	page, next, err := o.List("op-s5", tok5, MaxStorePageSize, "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if !page.Complete || page.Count != 0 || next != "" {
		t.Fatalf("terminal scan = %+v next=%q", page, next)
	}
	if err := o.CloseSession("op-s5", tok5, self); err != nil {
		t.Fatalf("CloseSession: %v", err)
	}
	rc, err := o.SealCleanup(self)
	if err != nil {
		t.Fatalf("SealCleanup: %v", err)
	}
	if rc.Prefix != "backups/daily" || rc.HandleDigest == "" {
		t.Fatalf("seal receipt = %+v", rc)
	}
	// A foreign identity cannot seal.
	if _, err := o.SealCleanup(journal.Owner{PID: 1, StartToken: "x"}); err == nil {
		t.Fatal("foreign seal succeeds")
	} else {
		requireStoreErr(t, err, StoreLayerEngine, StoreCodeWrongOwner)
	}
}

func TestStoreCloseDiscipline(t *testing.T) {
	_, o, self := storeObserver(t, "backups/daily")
	_, tok, err := o.OpenSession("op-s", "sess-1")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	if err := o.Close(); err == nil {
		t.Fatal("close with live session")
	} else {
		requireStoreErr(t, err, StoreLayerEngine, StoreCodeClosedHandle)
	}
	if err := o.CloseSession("op-s", tok, self); err != nil {
		t.Fatalf("CloseSession: %v", err)
	}
	if err := o.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := o.Close(); err == nil {
		t.Fatal("double close succeeds")
	} else {
		requireStoreErr(t, err, StoreLayerEngine, StoreCodeClosedHandle)
	}
	if _, err := o.Receipt(); err == nil {
		t.Fatal("receipt after close succeeds")
	}
}
