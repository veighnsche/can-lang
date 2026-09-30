package external

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/tools/native-test-owner/journal"
)

const testTTL = int64(60 * 60 * 1000)

func testConfig() Config {
	return Config{
		Listeners:    []string{"loopback:http", "loopback:grpc"},
		Destinations: []string{"svc-a.internal", "svc-b.internal"},
		Namespaces:   []string{"tenant-alpha", "tenant-beta"},
		Prefixes:     []string{"backups/daily", "artifacts"},
	}
}

func newRegistry(t *testing.T, j *journal.Journal, pid int, token string) *Registry {
	t.Helper()
	if j == nil {
		j = journal.OpenMemory()
	}
	r, err := New(j, pid, token, testConfig())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return r
}

func selfOf(r *Registry) journal.Owner { return r.Owner() }

func pendingOps(j *journal.Journal) map[string]bool {
	out := map[string]bool{}
	for _, p := range j.Pending() {
		out[p.OperationID] = true
	}
	return out
}

func TestAdmitPositive(t *testing.T) {
	j := journal.OpenMemory()
	r := newRegistry(t, j, 4242, "start-token-1")
	self := selfOf(r)

	g1, err := r.AdmitListener("op-listener", "loopback:http", testTTL)
	if err != nil {
		t.Fatalf("AdmitListener: %v", err)
	}
	g2, err := r.AdmitConnection("op-conn", "svc-a.internal", testTTL)
	if err != nil {
		t.Fatalf("AdmitConnection: %v", err)
	}
	g3, err := r.OpenContext("op-ctx", g2.Token, "session-7", testTTL)
	if err != nil {
		t.Fatalf("OpenContext: %v", err)
	}
	g4, err := r.GrantNamespace("op-ns", "tenant-alpha", testTTL)
	if err != nil {
		t.Fatalf("GrantNamespace: %v", err)
	}
	g5, err := r.IssueCredential("op-cred", g4.Token, testTTL)
	if err != nil {
		t.Fatalf("IssueCredential: %v", err)
	}
	g6, err := r.GrantPrefix("op-prefix", "backups/daily/2026-09-30", testTTL)
	if err != nil {
		t.Fatalf("GrantPrefix: %v", err)
	}
	for i, g := range []Grant{g1, g2, g3, g4, g5, g6} {
		if g.Token == "" || g.Revoked {
			t.Fatalf("grant %d not live: %+v", i, g)
		}
		if g.Owner != self {
			t.Fatalf("grant %d bound to wrong owner: %+v", i, g)
		}
	}
	// Journal intent precedes every effect: all six are live.
	for _, op := range []string{"op-listener", "op-conn", "op-ctx", "op-ns", "op-cred", "op-prefix"} {
		res, ok := j.Lookup(op)
		if !ok {
			t.Fatalf("journal misses %q", op)
		}
		if res.State != journal.StateLive {
			t.Fatalf("%q state = %s, want live", op, res.State)
		}
	}
	// Six retained unsettled charges.
	if got := r.Unsettled(); len(got) != 6 {
		t.Fatalf("unsettled charges = %d, want 6", len(got))
	}
	// Release drains in dependency order.
	for _, op := range r.CleanupOrder() {
		tok := ""
		switch op {
		case "op-listener":
			tok = g1.Token
		case "op-conn":
			tok = g2.Token
		case "op-ctx":
			tok = g3.Token
		case "op-ns":
			tok = g4.Token
		case "op-cred":
			tok = g5.Token
		case "op-prefix":
			tok = g6.Token
		}
		if err := r.Release(op, tok, self); err != nil {
			t.Fatalf("Release %q: %v", op, err)
		}
	}
	if got := r.Unsettled(); len(got) != 0 {
		t.Fatalf("unsettled after release = %d, want 0", len(got))
	}
}

func TestForeignTargetsRejected(t *testing.T) {
	j := journal.OpenMemory()
	r := newRegistry(t, j, 4242, "start-token-1")

	cases := []struct {
		name string
		call func() error
	}{
		{"listener", func() error {
			_, err := r.AdmitListener("op-f1", "public:internet", testTTL)
			return err
		}},
		{"server", func() error {
			_, err := r.AdmitConnection("op-f2", "evil.example.com", testTTL)
			return err
		}},
		{"namespace", func() error {
			_, err := r.GrantNamespace("op-f3", "tenant-mallory", testTTL)
			return err
		}},
		{"prefix", func() error {
			_, err := r.GrantPrefix("op-f4", "secrets/prod", testTTL)
			return err
		}},
		{"prefix-sibling", func() error {
			// "artifacts-evil" shares a string prefix with "artifacts"
			// but is not under it.
			_, err := r.GrantPrefix("op-f5", "artifacts-evil/x", testTTL)
			return err
		}},
		{"dispatch-server", func() error {
			_, err := r.Dispatch("op-f6", KindConnection, "evil.example.com", func() error { return nil })
			return err
		}},
	}
	for _, c := range cases {
		if err := c.call(); !errors.Is(err, ErrForeign) {
			t.Fatalf("%s: err = %v, want ErrForeign", c.name, err)
		}
	}
	// Rejection precedes journaling: no intent was recorded.
	if pend := pendingOps(j); len(pend) != 0 {
		t.Fatalf("journal pending after foreign rejects = %v, want empty", pend)
	}
	if got := r.Unsettled(); len(got) != 0 {
		t.Fatalf("charges after foreign rejects = %d, want 0", len(got))
	}
	if got := r.Legs(); len(got) != 0 {
		t.Fatalf("legs after foreign rejects = %d, want 0", len(got))
	}
}

func TestCallerInventedAuthorityRejected(t *testing.T) {
	j := journal.OpenMemory()
	r := newRegistry(t, j, 4242, "start-token-1")
	conn, err := r.AdmitConnection("op-conn", "svc-a.internal", testTTL)
	if err != nil {
		t.Fatalf("AdmitConnection: %v", err)
	}
	ns, err := r.GrantNamespace("op-ns", "tenant-alpha", testTTL)
	if err != nil {
		t.Fatalf("GrantNamespace: %v", err)
	}

	// Self-minted tokens are unknown.
	if _, err := r.OpenContext("op-x1", "ex1-deadbeefdeadbeefdeadbeefdeadbeef", "s", testTTL); !errors.Is(err, ErrNotFound) {
		t.Fatalf("invented context parent: err = %v, want ErrNotFound", err)
	}
	if _, err := r.IssueCredential("op-x2", "ex1-deadbeefdeadbeefdeadbeefdeadbeef", testTTL); !errors.Is(err, ErrNotFound) {
		t.Fatalf("invented credential parent: err = %v, want ErrNotFound", err)
	}
	// A live grant for the wrong leg is not authority either.
	if _, err := r.OpenContext("op-x3", ns.Token, "s", testTTL); !errors.Is(err, ErrDenied) {
		t.Fatalf("namespace grant as context parent: err = %v, want ErrDenied", err)
	}
	if _, err := r.IssueCredential("op-x4", conn.Token, testTTL); !errors.Is(err, ErrDenied) {
		t.Fatalf("connection grant as credential parent: err = %v, want ErrDenied", err)
	}
	// Nothing was admitted behind the rejections.
	if pend := pendingOps(j); len(pend) != 2 {
		t.Fatalf("journal pending = %v, want only the two admissions", pend)
	}
	if got := r.Unsettled(); len(got) != 2 {
		t.Fatalf("charges = %d, want 2", len(got))
	}
}

func TestForgedReleaseRejected(t *testing.T) {
	j := journal.OpenMemory()
	r := newRegistry(t, j, 4242, "start-token-1")
	self := selfOf(r)
	a, err := r.AdmitConnection("op-a", "svc-a.internal", testTTL)
	if err != nil {
		t.Fatalf("AdmitConnection: %v", err)
	}
	b, err := r.AdmitConnection("op-b", "svc-b.internal", testTTL)
	if err != nil {
		t.Fatalf("AdmitConnection: %v", err)
	}

	// Invented token: forged.
	if err := r.Release("op-a", "ex1-forgedforgedforgedforgedforged00", self); !errors.Is(err, ErrNotFound) {
		t.Fatalf("invented release token: err = %v, want ErrNotFound", err)
	}
	// Live token bound to another operation: forged.
	if err := r.Release("op-a", b.Token, self); !errors.Is(err, ErrDenied) {
		t.Fatalf("cross-operation release: err = %v, want ErrDenied", err)
	}
	// Foreign identity with the right token: forged.
	alien := journal.Owner{PID: 9999, StartToken: "alien"}
	if err := r.Release("op-a", a.Token, alien); !errors.Is(err, ErrWrongOwner) {
		t.Fatalf("foreign-identity release: err = %v, want ErrWrongOwner", err)
	}
	// Unknown operation: nothing to release.
	if err := r.Release("op-ghost", a.Token, self); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ghost release: err = %v, want ErrNotFound", err)
	}
	// Every forgery left the resource live and its charge unsettled.
	f, err := r.Facts("op-a")
	if err != nil {
		t.Fatalf("Facts: %v", err)
	}
	if f.Released {
		t.Fatal("forged releases marked op-a released")
	}
	if got := r.Unsettled(); len(got) != 2 {
		t.Fatalf("unsettled = %d, want 2", len(got))
	}
	if res, _ := j.Lookup("op-a"); res.State != journal.StateLive {
		t.Fatalf("op-a journal state = %s, want live", res.State)
	}
}

func TestDuplicateUncertainDispatchRejected(t *testing.T) {
	j := journal.OpenMemory()
	r := newRegistry(t, j, 4242, "start-token-1")
	runs := 0
	effect := func() error { runs++; return nil }

	joined, err := r.Dispatch("op-d", KindConnection, "svc-a.internal", effect)
	if err != nil || joined {
		t.Fatalf("first dispatch: joined=%v err=%v, want fresh run", joined, err)
	}
	// Same leg rejoins without a new effect.
	joined, err = r.Dispatch("op-d", KindConnection, "svc-a.internal", effect)
	if err != nil || !joined {
		t.Fatalf("repeat dispatch: joined=%v err=%v, want join", joined, err)
	}
	if runs != 1 {
		t.Fatalf("effect runs = %d, want 1", runs)
	}
	// Same ID, different leg: changed input, never joined.
	if _, err := r.Dispatch("op-d", KindConnection, "svc-b.internal", effect); !errors.Is(err, ErrChangedInput) {
		t.Fatalf("changed leg: err = %v, want ErrChangedInput", err)
	}
	// Lost acknowledgment: the dispatch is uncertain from here on.
	if err := r.ReportUncertain("op-d"); err != nil {
		t.Fatalf("ReportUncertain: %v", err)
	}
	if _, err := r.Dispatch("op-d", KindConnection, "svc-a.internal", effect); !errors.Is(err, ErrIndeterminate) {
		t.Fatalf("uncertain redispatch: err = %v, want ErrIndeterminate", err)
	}
	if runs != 1 {
		t.Fatalf("effect runs after uncertain redispatch = %d, want 1", runs)
	}
	if n, ok := r.EffectRuns("op-d"); !ok || n != 1 {
		t.Fatalf("EffectRuns = (%d,%v), want (1,true)", n, ok)
	}
	// Uncertain report for an unknown operation invents nothing.
	if err := r.ReportUncertain("op-ghost"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ghost uncertain: err = %v, want ErrNotFound", err)
	}
}

func TestFailedDispatchRepeatsIndeterminate(t *testing.T) {
	j := journal.OpenMemory()
	r := newRegistry(t, j, 4242, "start-token-1")
	runs := 0
	boom := errors.New("boom: effect failed")
	effect := func() error { runs++; return boom }

	// The first caller keeps the real error.
	if _, err := r.Dispatch("op-fail", KindConnection, "svc-a.internal", effect); !errors.Is(err, boom) {
		t.Fatalf("first dispatch: err = %v, want the effect error", err)
	}
	// A failure may have partially run: repeats report indeterminate
	// rather than joining a failure as success, and never redispatch.
	if _, err := r.Dispatch("op-fail", KindConnection, "svc-a.internal", effect); !errors.Is(err, ErrIndeterminate) {
		t.Fatalf("repeat after failure: err = %v, want ErrIndeterminate", err)
	}
	if runs != 1 {
		t.Fatalf("effect runs = %d, want 1", runs)
	}
	if res, _ := j.Lookup("op-fail"); res.State != journal.StateFailed {
		t.Fatalf("journal state = %s, want failed", res.State)
	}
}

func TestCrashBeforeJournalLeavesNoAuthority(t *testing.T) {
	dir := t.TempDir()
	j, err := journal.OpenDir(dir)
	if err != nil {
		t.Fatalf("OpenDir: %v", err)
	}
	r, err := New(j, 4242, "start-token-1", testConfig())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// Simulate the crash window: the journal is gone before intent lands.
	if err := j.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := r.AdmitConnection("op-crash", "svc-a.internal", testTTL); err == nil {
		t.Fatal("admit against a dead journal succeeded, want failure")
	}
	if _, err := r.Dispatch("op-crash-d", KindNamespace, "tenant-alpha", func() error { return nil }); err == nil {
		t.Fatal("dispatch against a dead journal succeeded, want failure")
	}
	// No grant, no resource, no charge, no fact survived the failure.
	if got := r.Unsettled(); len(got) != 0 {
		t.Fatalf("charges after crash = %d, want 0", len(got))
	}
	if got := r.Legs(); len(got) != 0 {
		t.Fatalf("legs after crash = %d, want 0", len(got))
	}
	if _, err := r.Facts("op-crash"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Facts(op-crash): err = %v, want ErrNotFound", err)
	}
	// Recovery finds nothing to touch.
	rep := r.Recover(RecoverRequest{Self: r.Owner(), Allow: map[string]bool{"op-crash": true}}, func(string) {
		t.Error("touch called with no evidenced resources")
	})
	if len(rep.Outcomes) != 0 || rep.Touched != 0 {
		t.Fatalf("recovery after crash = %+v, want empty", rep)
	}
}

func TestRecoveryTouchesOnlyEvidencedOwned(t *testing.T) {
	j := journal.OpenMemory()
	a := newRegistry(t, j, 4242, "start-token-1")
	b, err := New(j, 7777, "start-token-2", testConfig())
	if err != nil {
		t.Fatalf("New(b): %v", err)
	}
	selfA := a.Owner()
	if _, err := a.AdmitConnection("op-own", "svc-a.internal", testTTL); err != nil {
		t.Fatalf("AdmitConnection(own): %v", err)
	}
	if _, err := a.AdmitConnection("op-refused", "svc-b.internal", testTTL); err != nil {
		t.Fatalf("AdmitConnection(refused): %v", err)
	}
	if _, err := b.AdmitConnection("op-alien", "svc-a.internal", testTTL); err != nil {
		t.Fatalf("AdmitConnection(alien): %v", err)
	}
	// Crash between reserve and effect: journal intent with no local
	// resource behind it.
	digest := "sha256:" + strings.Repeat("0", 64)
	if _, err := j.Reserve("op-journal-only", digest, selfA, journal.PathIdentity{}, "", 0); err != nil {
		t.Fatalf("Reserve journal-only: %v", err)
	}

	var touched []string
	rep := a.Recover(RecoverRequest{
		Self:   selfA,
		Allow:  map[string]bool{"op-own": true, "op-alien": true, "op-journal-only": true},
		Reason: map[string]string{"op-refused": "owner-mismatch"},
	}, func(op string) { touched = append(touched, op) })

	if rep.Touched != 1 || len(touched) != 1 || touched[0] != "op-own" {
		t.Fatalf("touched = %v (n=%d), want [op-own]", touched, rep.Touched)
	}
	byOp := map[string]RecoverOutcome{}
	for _, o := range rep.Outcomes {
		byOp[o.OperationID] = o
	}
	// The refused operation is classified, never touched.
	ro, ok := byOp["op-refused"]
	if !ok || ro.Touched || ro.Reason != "owner-mismatch" {
		t.Fatalf("refused outcome = %+v, want untouched with scan reason", ro)
	}
	// Foreign and journal-only intents are outside this registry: they
	// appear in no outcome and are never touched.
	if _, ok := byOp["op-alien"]; ok {
		t.Fatal("foreign resource appears in owned recovery outcomes")
	}
	if _, ok := byOp["op-journal-only"]; ok {
		t.Fatal("journal-only intent appears in owned recovery outcomes")
	}
	// From the alien registry's view the mirror holds: its scan touches
	// only its own resource.
	var touchedB []string
	repB := b.Recover(RecoverRequest{Self: b.Owner(), Allow: map[string]bool{"op-alien": true}}, func(op string) {
		touchedB = append(touchedB, op)
	})
	if repB.Touched != 1 || len(touchedB) != 1 || touchedB[0] != "op-alien" {
		t.Fatalf("alien touched = %v, want [op-alien]", touchedB)
	}
}

func TestMissingReleaseStaysUnresolved(t *testing.T) {
	j := journal.OpenMemory()
	r := newRegistry(t, j, 4242, "start-token-1")
	self := selfOf(r)
	g, err := r.AdmitConnection("op-lingering", "svc-a.internal", testTTL)
	if err != nil {
		t.Fatalf("AdmitConnection: %v", err)
	}

	// Full recovery cycle: quiesce, fence, recover-all.
	if live := r.Quiesce(); len(live) != 1 || live[0] != "op-lingering" {
		t.Fatalf("quiesced live = %v, want [op-lingering]", live)
	}
	ack, err := r.Fence()
	if err != nil {
		t.Fatalf("Fence: %v", err)
	}
	if len(ack.Live) != 1 || ack.Live[0] != "op-lingering" {
		t.Fatalf("fence ack live = %v, want [op-lingering]", ack.Live)
	}
	rep := r.Recover(RecoverRequest{Self: self, Allow: map[string]bool{"op-lingering": true}}, func(string) {})
	if rep.Touched != 1 {
		t.Fatalf("recovery touched = %d, want 1", rep.Touched)
	}
	if !rep.Outcomes[0].ReleaseMissing {
		t.Fatal("recovered outcome hides the missing release")
	}
	// Nothing auto-resolved: still live, still charged, still unreleased.
	if live := r.Live(); len(live) != 1 {
		t.Fatalf("live after recovery = %v, want [op-lingering]", live)
	}
	if got := r.Unsettled(); len(got) != 1 || got[0].Settled {
		t.Fatalf("charges after recovery = %+v, want one unsettled", got)
	}
	f, err := r.Facts("op-lingering")
	if err != nil {
		t.Fatalf("Facts: %v", err)
	}
	if f.Released {
		t.Fatal("recovery auto-resolved the missing release")
	}
	if res, _ := j.Lookup("op-lingering"); res.State != journal.StateLive {
		t.Fatalf("journal state = %s, want live", res.State)
	}
	// Only an explicit Release settles it.
	if err := r.Release("op-lingering", g.Token, self); err != nil {
		t.Fatalf("Release: %v", err)
	}
	if got := r.Unsettled(); len(got) != 0 {
		t.Fatalf("charges after release = %d, want 0", len(got))
	}
}

func TestCleanupDependencyOrdering(t *testing.T) {
	j := journal.OpenMemory()
	r := newRegistry(t, j, 4242, "start-token-1")
	self := selfOf(r)
	conn, err := r.AdmitConnection("op-conn", "svc-a.internal", testTTL)
	if err != nil {
		t.Fatalf("AdmitConnection: %v", err)
	}
	ctx, err := r.OpenContext("op-ctx", conn.Token, "s", testTTL)
	if err != nil {
		t.Fatalf("OpenContext: %v", err)
	}
	ns, err := r.GrantNamespace("op-ns", "tenant-alpha", testTTL)
	if err != nil {
		t.Fatalf("GrantNamespace: %v", err)
	}
	cred, err := r.IssueCredential("op-cred", ns.Token, testTTL)
	if err != nil {
		t.Fatalf("IssueCredential: %v", err)
	}

	// Parents cannot release while a dependent is live.
	if err := r.Release("op-conn", conn.Token, self); !errors.Is(err, ErrDependency) {
		t.Fatalf("release connection with live context: err = %v, want ErrDependency", err)
	}
	if err := r.Release("op-ns", ns.Token, self); !errors.Is(err, ErrDependency) {
		t.Fatalf("release namespace with live credential: err = %v, want ErrDependency", err)
	}
	// Cleanup order drains dependents first.
	order := r.CleanupOrder()
	pos := map[string]int{}
	for i, op := range order {
		pos[op] = i
	}
	if pos["op-ctx"] > pos["op-conn"] {
		t.Fatalf("cleanup order = %v, want context before connection", order)
	}
	if pos["op-cred"] > pos["op-ns"] {
		t.Fatalf("cleanup order = %v, want credential before namespace", order)
	}
	// Releasing in that order succeeds.
	toks := map[string]string{"op-conn": conn.Token, "op-ctx": ctx.Token, "op-ns": ns.Token, "op-cred": cred.Token}
	for _, op := range order {
		if err := r.Release(op, toks[op], self); err != nil {
			t.Fatalf("Release %q in cleanup order: %v", op, err)
		}
	}
	if got := r.Unsettled(); len(got) != 0 {
		t.Fatalf("charges after ordered release = %d, want 0", len(got))
	}
}

func TestQuiescenceAndFence(t *testing.T) {
	j := journal.OpenMemory()
	r := newRegistry(t, j, 4242, "start-token-1")
	self := selfOf(r)
	g, err := r.AdmitListener("op-l", "loopback:http", testTTL)
	if err != nil {
		t.Fatalf("AdmitListener: %v", err)
	}

	// Fencing before quiescence is rejected.
	if _, err := r.Fence(); !errors.Is(err, ErrInvalid) {
		t.Fatalf("early fence: err = %v, want ErrInvalid", err)
	}
	if _, ok := r.FenceAck(); ok {
		t.Fatal("fence ack present before fencing")
	}
	r.Quiesce()
	// Admission and dispatch stop; release still drains.
	if _, err := r.AdmitListener("op-l2", "loopback:grpc", testTTL); !errors.Is(err, ErrQuiesced) {
		t.Fatalf("admit after quiesce: err = %v, want ErrQuiesced", err)
	}
	if _, err := r.Dispatch("op-d", KindNamespace, "tenant-alpha", func() error { return nil }); !errors.Is(err, ErrQuiesced) {
		t.Fatalf("dispatch after quiesce: err = %v, want ErrQuiesced", err)
	}
	ack, err := r.Fence()
	if err != nil {
		t.Fatalf("Fence: %v", err)
	}
	if ack.Owner != self || len(ack.Live) != 1 || ack.Live[0] != "op-l" {
		t.Fatalf("fence ack = %+v, want owner with [op-l]", ack)
	}
	again, err := r.Fence()
	if err != nil {
		t.Fatalf("second Fence: %v", err)
	}
	if again.TimeMs != ack.TimeMs || len(again.Live) != 1 {
		t.Fatal("fence is not single-shot")
	}
	if err := r.Release("op-l", g.Token, self); err != nil {
		t.Fatalf("Release after fence: %v", err)
	}
}

func TestLegsUnqualifiedAndCleanupNeverProven(t *testing.T) {
	j := journal.OpenMemory()
	r := newRegistry(t, j, 4242, "start-token-1")
	self := selfOf(r)
	conn, _ := r.AdmitConnection("op-conn", "svc-a.internal", testTTL)
	ctx, _ := r.OpenContext("op-ctx", conn.Token, "s", testTTL)
	ns, _ := r.GrantNamespace("op-ns", "tenant-alpha", testTTL)
	cred, _ := r.IssueCredential("op-cred", ns.Token, testTTL)
	lis, _ := r.AdmitListener("op-lis", "loopback:http", testTTL)
	pfx, _ := r.GrantPrefix("op-pfx", "artifacts/build-9", testTTL)

	legs := r.Legs()
	if len(legs) != 6 {
		t.Fatalf("legs = %d, want 6", len(legs))
	}
	seen := map[Kind]bool{}
	for _, l := range legs {
		if l.Qualification != QualificationUnqualified {
			t.Fatalf("leg %+v qualification = %q, want unqualified", l, l.Qualification)
		}
		seen[l.Kind] = true
	}
	for _, k := range []Kind{KindListener, KindConnection, KindContext, KindNamespace, KindCredential, KindPrefix} {
		if !seen[k] {
			t.Fatalf("no %q leg recorded", k)
		}
	}
	// Facts agree, and cleanup stays unproven even after release.
	for _, op := range []string{"op-ctx", "op-conn", "op-cred", "op-ns", "op-lis", "op-pfx"} {
		f, err := r.Facts(op)
		if err != nil {
			t.Fatalf("Facts(%q): %v", op, err)
		}
		if f.Qualification != QualificationUnqualified || f.CleanupProven {
			t.Fatalf("Facts(%q) = %+v, want unqualified and unproven", op, f)
		}
	}
	// Release in dependency order; facts must still disclaim cleanup.
	for _, op := range r.CleanupOrder() {
		var tok string
		switch op {
		case "op-conn":
			tok = conn.Token
		case "op-ctx":
			tok = ctx.Token
		case "op-ns":
			tok = ns.Token
		case "op-cred":
			tok = cred.Token
		case "op-lis":
			tok = lis.Token
		case "op-pfx":
			tok = pfx.Token
		}
		if err := r.Release(op, tok, self); err != nil {
			t.Fatalf("Release %q: %v", op, err)
		}
		f, err := r.Facts(op)
		if err != nil {
			t.Fatalf("Facts(%q): %v", op, err)
		}
		if f.CleanupProven || f.Qualification != QualificationUnqualified {
			t.Fatalf("post-release Facts(%q) = %+v, want unqualified and unproven", op, f)
		}
	}
}

func TestCredentialFactsExposeNoSecret(t *testing.T) {
	j := journal.OpenMemory()
	r := newRegistry(t, j, 4242, "start-token-1")
	ns, err := r.GrantNamespace("op-ns", "tenant-alpha", testTTL)
	if err != nil {
		t.Fatalf("GrantNamespace: %v", err)
	}
	cred, err := r.IssueCredential("op-cred", ns.Token, testTTL)
	if err != nil {
		t.Fatalf("IssueCredential: %v", err)
	}
	f, err := r.Facts("op-cred")
	if err != nil {
		t.Fatalf("Facts: %v", err)
	}
	rendered := fmt.Sprintf("%+v", f)
	if strings.Contains(rendered, cred.Token) {
		t.Fatal("credential fact renders the opaque handle")
	}
	if f.HandleDigest == "" || f.HandleDigest == cred.Token {
		t.Fatalf("handle digest = %q, want an opaque digest", f.HandleDigest)
	}
	if !strings.HasPrefix(f.HandleDigest, "sha256:") {
		t.Fatalf("handle digest = %q, want sha256:<hex>", f.HandleDigest)
	}
}

func TestGrantExpiryAndRevocation(t *testing.T) {
	now := time.Now().UnixMilli()
	cfg := testConfig()
	cfg.Clock = func() int64 { return now }
	j := journal.OpenMemory()
	r, err := New(j, 4242, "start-token-1", cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	self := r.Owner()
	g, err := r.AdmitConnection("op-e", "svc-a.internal", 1000)
	if err != nil {
		t.Fatalf("AdmitConnection: %v", err)
	}
	// Expired grants authorize nothing, not even a release.
	now += 1001
	if _, err := r.OpenContext("op-ec", g.Token, "s", testTTL); !errors.Is(err, ErrExpired) {
		t.Fatalf("context under expired grant: err = %v, want ErrExpired", err)
	}
	if err := r.Release("op-e", g.Token, self); !errors.Is(err, ErrExpired) {
		t.Fatalf("release with expired grant: err = %v, want ErrExpired", err)
	}
	if got := r.Unsettled(); len(got) != 1 {
		t.Fatalf("charges after expired release = %d, want 1 retained", len(got))
	}
	// Revocation: a release revokes its grant, and the revoked token
	// authorizes nothing afterwards — not even as a parent grant.
	g2, err := r.AdmitConnection("op-r", "svc-b.internal", testTTL)
	if err != nil {
		t.Fatalf("AdmitConnection: %v", err)
	}
	if err := r.Release("op-r", g2.Token, self); err != nil {
		t.Fatalf("Release: %v", err)
	}
	if _, err := r.OpenContext("op-rc", g2.Token, "s", testTTL); !errors.Is(err, ErrRevoked) {
		t.Fatalf("context under revoked grant: err = %v, want ErrRevoked", err)
	}
	// Double release is a caller error, not a silent join.
	if err := r.Release("op-r", g2.Token, self); !errors.Is(err, ErrInvalid) {
		t.Fatalf("second release: err = %v, want ErrInvalid", err)
	}
}

func TestConcurrentAdmissions(t *testing.T) {
	j := journal.OpenMemory()
	r := newRegistry(t, j, 4242, "start-token-1")
	var wg sync.WaitGroup
	errs := make(chan error, 32)
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			op := fmt.Sprintf("op-par-%02d", i)
			if _, err := r.GrantPrefix(op, fmt.Sprintf("artifacts/shard-%02d", i), testTTL); err != nil {
				errs <- err
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("parallel admit: %v", err)
	}
	if got := r.Unsettled(); len(got) != 16 {
		t.Fatalf("charges = %d, want 16", len(got))
	}
}
func TestConstructorBounds(t *testing.T) {
	j := journal.OpenMemory()
	if _, err := New(nil, 1, "t", testConfig()); !errors.Is(err, ErrInvalid) {
		t.Fatalf("nil journal: err = %v, want ErrInvalid", err)
	}
	if _, err := New(j, 0, "t", testConfig()); !errors.Is(err, ErrInvalid) {
		t.Fatalf("zero pid: err = %v, want ErrInvalid", err)
	}
	big := testConfig()
	for i := 0; i <= MaxDeclared; i++ {
		big.Namespaces = append(big.Namespaces, fmt.Sprintf("ns-%d", i))
	}
	if _, err := New(j, 1, "t", big); !errors.Is(err, ErrInvalid) {
		t.Fatalf("oversize declared set: err = %v, want ErrInvalid", err)
	}
	dup := testConfig()
	dup.Listeners = []string{"a", "a"}
	if _, err := New(j, 1, "t", dup); !errors.Is(err, ErrInvalid) {
		t.Fatalf("duplicate declared entry: err = %v, want ErrInvalid", err)
	}
}
