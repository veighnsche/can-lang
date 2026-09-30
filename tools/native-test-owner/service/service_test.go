package service

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"sync/atomic"
	"testing"

	"github.com/veighnsche/can-lang/tools/native-test-owner/journal"
	"github.com/veighnsche/can-lang/tools/native-test-owner/protocol/codec"
	"github.com/veighnsche/can-lang/tools/native-test-owner/protocol/dispatch"
)

func td(n int) string { return fmt.Sprintf("sha256:%064x", n) }

func newTestService(t *testing.T, cfg Config) (*Service, *journal.Journal) {
	t.Helper()
	j := journal.OpenMemory()
	t.Cleanup(func() { _ = j.Close() })
	s, err := New(j, os.Getpid(), "test-start-token", cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return s, j
}

// echoHandler returns completed/ok echoing the digest and counts calls.
func echoHandler(calls *atomic.Int32) Handler {
	return func(digest string) (dispatch.Result, error) {
		calls.Add(1)
		return dispatch.Result{Outcome: codec.OutcomeCompleted, Kind: codec.KindOK, Facts: map[string]any{"digest": digest}}, nil
	}
}

func mustRunGrant(t *testing.T, s *Service, runID string) (dispatch.Grant, dispatch.Handle) {
	t.Helper()
	g, h, err := s.IssueRunGrant(runID, 60_000)
	if err != nil {
		t.Fatalf("IssueRunGrant: %v", err)
	}
	return g, h
}

func TestServiceConfigValidation(t *testing.T) {
	j := journal.OpenMemory()
	defer j.Close()
	cases := []struct {
		name  string
		j     *journal.Journal
		pid   int
		token string
		cfg   Config
	}{
		{"nil journal", nil, 1, "tok", Config{}},
		{"zero pid", j, 0, "tok", Config{}},
		{"empty token", j, 1, "", Config{}},
		{"bad queue bound", j, 1, "tok", Config{MaxQueue: dispatch.MaxQueueCap + 1}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := New(c.j, c.pid, c.token, c.cfg); err == nil {
				t.Fatal("New accepted an invalid config")
			}
		})
	}
	s, _ := newTestService(t, Config{})
	if err := s.RegisterOperation("", echoHandler(&atomic.Int32{})); !errors.Is(err, ErrInvalid) {
		t.Fatalf("empty name err = %v, want ErrInvalid", err)
	}
	if err := s.RegisterOperation("echo", nil); !errors.Is(err, ErrInvalid) {
		t.Fatalf("nil handler err = %v, want ErrInvalid", err)
	}
	if err := s.RegisterOperation("echo", echoHandler(&atomic.Int32{})); err != nil {
		t.Fatalf("RegisterOperation: %v", err)
	}
	if err := s.RegisterOperation("echo", echoHandler(&atomic.Int32{})); !errors.Is(err, ErrInvalid) {
		t.Fatalf("duplicate err = %v, want ErrInvalid", err)
	}
}

// TestServiceDispatchSequence covers join, reject and indeterminate through
// the full authorize-journal-execute path with at-most-once effects.
func TestServiceDispatchSequence(t *testing.T) {
	s, _ := newTestService(t, Config{})
	var calls atomic.Int32
	if err := s.RegisterOperation("echo", echoHandler(&calls)); err != nil {
		t.Fatal(err)
	}
	if err := s.RegisterOperation("fail", func(string) (dispatch.Result, error) {
		calls.Add(1)
		return dispatch.Result{}, errors.New("boom")
	}); err != nil {
		t.Fatal(err)
	}
	g, h := mustRunGrant(t, s, "run-1")
	req := func(opID, op, digest string) Request {
		return Request{GrantToken: g.Token, Handle: h, RunID: "run-1", OperationID: opID, Operation: op, ArgumentsDigest: digest}
	}

	rep, err := s.Dispatch(req("op-1", "echo", td(1)))
	if err != nil || rep.Joined {
		t.Fatalf("fresh = %+v,%v, want unjoined success", rep, err)
	}
	if rep.Result.Outcome != codec.OutcomeCompleted || rep.Result.Kind != codec.KindOK {
		t.Fatalf("result = %+v", rep.Result)
	}
	if rep.Handle.Kind != dispatch.HandleOperation || rep.Handle.ID != "op-1" {
		t.Fatalf("handle = %+v", rep.Handle)
	}
	if calls.Load() != 1 {
		t.Fatalf("calls = %d, want 1", calls.Load())
	}

	joined, err := s.Dispatch(req("op-1", "echo", td(1)))
	if err != nil || !joined.Joined {
		t.Fatalf("join = %+v,%v, want joined success", joined, err)
	}
	if joined.Handle != rep.Handle || joined.Result.Facts["digest"] != td(1) {
		t.Fatalf("joined = %+v, want identical outcome and handle", joined)
	}
	if calls.Load() != 1 {
		t.Fatalf("calls after join = %d, want 1", calls.Load())
	}

	rejects := []struct {
		name string
		req  Request
	}{
		{"changed digest", req("op-1", "echo", td(2))},
		{"changed operation", req("op-1", "fail", td(1))},
	}
	for _, c := range rejects {
		t.Run(c.name, func(t *testing.T) {
			if _, err := s.Dispatch(c.req); !errors.Is(err, dispatch.ErrChangedInput) {
				t.Fatalf("err = %v, want ErrChangedInput", err)
			}
		})
	}
	if calls.Load() != 1 {
		t.Fatalf("calls after rejects = %d, want 1", calls.Load())
	}

	// Unknown operations consume their ID as a terminal rejection.
	unk, err := s.Dispatch(req("op-unknown", "missing", td(3)))
	if err != nil || unk.Joined {
		t.Fatalf("unknown = %+v,%v", unk, err)
	}
	if unk.Result.Outcome != codec.OutcomeRejected || unk.Result.Kind != "unsupported-capability" {
		t.Fatalf("unknown result = %+v", unk.Result)
	}
	unk2, err := s.Dispatch(req("op-unknown", "missing", td(3)))
	if err != nil || !unk2.Joined || unk2.Result.Kind != "unsupported-capability" {
		t.Fatalf("unknown join = %+v,%v", unk2, err)
	}

	// Effect errors become terminal failures and join.
	bad, err := s.Dispatch(req("op-fail", "fail", td(4)))
	if err != nil || bad.Result.Outcome != codec.OutcomeFailed {
		t.Fatalf("failed = %+v,%v", bad, err)
	}
	if _, err := s.Dispatch(req("op-fail", "fail", td(4))); err != nil {
		t.Fatalf("failed join: %v", err)
	}
	if calls.Load() != 2 {
		t.Fatalf("calls = %d, want 2", calls.Load())
	}

	// Lost acknowledgment reports indeterminate without redispatch.
	if err := s.ReportAckLost("op-1"); err != nil {
		t.Fatalf("ReportAckLost: %v", err)
	}
	if err := s.ReportAckLost("op-1"); err != nil {
		t.Fatalf("double ReportAckLost: %v", err)
	}
	ind, err := s.Dispatch(req("op-1", "echo", td(1)))
	if !errors.Is(err, dispatch.ErrIndeterminate) || !ind.Joined {
		t.Fatalf("indeterminate join = %+v,%v", ind, err)
	}
	if ind.Result.Outcome != codec.OutcomeIndeterminate || ind.Result.Kind != "transport-failure" {
		t.Fatalf("indeterminate result = %+v", ind.Result)
	}
	if calls.Load() != 2 {
		t.Fatalf("calls after indeterminate = %d, want 2", calls.Load())
	}
	if err := s.ReportAckLost("op-missing"); !errors.Is(err, dispatch.ErrNotFound) {
		t.Fatalf("ack lost unknown err = %v, want ErrNotFound", err)
	}
	if err := s.ReportAckLost("bad id!"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("ack lost invalid err = %v, want ErrInvalid", err)
	}

	// Status follows the typed operation handle.
	got, err := s.Status(rep.Handle)
	if err != nil || got.Outcome != codec.OutcomeIndeterminate {
		t.Fatalf("Status = %+v,%v", got, err)
	}
	if _, err := s.Status(h); !errors.Is(err, dispatch.ErrWrongKind) {
		t.Fatalf("Status with run handle err = %v, want ErrWrongKind", err)
	}
	ghost := dispatch.Handle{Kind: dispatch.HandleOperation, ID: "op-ghost", Generation: 1}
	if _, err := s.Status(ghost); !errors.Is(err, dispatch.ErrNotFound) {
		t.Fatalf("Status ghost err = %v, want ErrNotFound", err)
	}
}

// TestServiceAuthorization proves every rejection path leaves no intent and
// runs no effect.
func TestServiceAuthorization(t *testing.T) {
	now := int64(1_700_000_000_000)
	s, j := newTestService(t, Config{Clock: func() int64 { return now }})
	var calls atomic.Int32
	if err := s.RegisterOperation("echo", echoHandler(&calls)); err != nil {
		t.Fatal(err)
	}
	g, h := mustRunGrant(t, s, "run-1")
	cg, ch, err := s.IssueCaseGrant(g.Token, "case-1", 60_000)
	if err != nil {
		t.Fatalf("IssueCaseGrant: %v", err)
	}
	staleRun, staleH, err := s.IssueRunGrant("run-stale", 60_000)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.CloseHandle(staleH); err != nil {
		t.Fatal(err)
	}
	reopened, err := s.handles.Open(dispatch.HandleRun, "run-stale")
	if err != nil {
		t.Fatal(err)
	}
	_ = reopened
	closedG, closedH, err := s.IssueRunGrant("run-closed", 60_000)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.CloseHandle(closedH); err != nil {
		t.Fatal(err)
	}
	revokedG, revokedH, err := s.IssueRunGrant("run-revoked", 60_000)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.RevokeGrant(revokedG.Token); err != nil {
		t.Fatal(err)
	}
	if err := s.RevokeGrant("ng1-missing"); !errors.Is(err, dispatch.ErrNotFound) {
		t.Fatalf("revoke unknown err = %v, want ErrNotFound", err)
	}
	opH := dispatch.Handle{Kind: dispatch.HandleOperation, ID: "op-1", Generation: 1}

	cases := []struct {
		name string
		req  Request
		want error
	}{
		{"unknown grant", Request{GrantToken: "ng1-missing", Handle: h, RunID: "run-1", OperationID: "op-auth-1", Operation: "echo", ArgumentsDigest: td(1)}, dispatch.ErrNotFound},
		{"revoked grant", Request{GrantToken: revokedG.Token, Handle: revokedH, RunID: "run-revoked", OperationID: "op-auth-2", Operation: "echo", ArgumentsDigest: td(2)}, dispatch.ErrRevoked},
		{"operation handle as auth", Request{GrantToken: g.Token, Handle: opH, RunID: "run-1", OperationID: "op-auth-3", Operation: "echo", ArgumentsDigest: td(3)}, dispatch.ErrWrongKind},
		{"stale handle", Request{GrantToken: staleRun.Token, Handle: staleH, RunID: "run-stale", OperationID: "op-auth-4", Operation: "echo", ArgumentsDigest: td(4)}, dispatch.ErrStaleHandle},
		{"closed handle", Request{GrantToken: closedG.Token, Handle: closedH, RunID: "run-closed", OperationID: "op-auth-5", Operation: "echo", ArgumentsDigest: td(5)}, dispatch.ErrClosedHandle},
		{"run grant with case handle", Request{GrantToken: g.Token, Handle: ch, RunID: "run-1", OperationID: "op-auth-6", Operation: "echo", ArgumentsDigest: td(6)}, dispatch.ErrInvalid},
		{"case grant with run handle", Request{GrantToken: cg.Token, Handle: h, RunID: "run-1", OperationID: "op-auth-7", Operation: "echo", ArgumentsDigest: td(7)}, dispatch.ErrInvalid},
		{"grant bound to another run", Request{GrantToken: g.Token, Handle: h, RunID: "run-2", OperationID: "op-auth-8", Operation: "echo", ArgumentsDigest: td(8)}, dispatch.ErrInvalid},
		{"bad run id", Request{GrantToken: g.Token, Handle: h, RunID: "bad id!", OperationID: "op-auth-9", Operation: "echo", ArgumentsDigest: td(9)}, ErrInvalid},
		{"bad op id", Request{GrantToken: g.Token, Handle: h, RunID: "run-1", OperationID: "", Operation: "echo", ArgumentsDigest: td(10)}, ErrInvalid},
		{"bad digest", Request{GrantToken: g.Token, Handle: h, RunID: "run-1", OperationID: "op-auth-11", Operation: "echo", ArgumentsDigest: "nope"}, ErrInvalid},
		{"empty operation", Request{GrantToken: g.Token, Handle: h, RunID: "run-1", OperationID: "op-auth-12", Operation: "", ArgumentsDigest: td(12)}, ErrInvalid},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := s.Dispatch(c.req); !errors.Is(err, c.want) {
				t.Fatalf("err = %v, want %v", err, c.want)
			}
			if _, ok := j.Lookup(c.req.OperationID); ok {
				t.Fatal("rejected request journaled intent")
			}
			if _, ok := s.EffectRuns(c.req.OperationID); ok {
				t.Fatal("rejected request recorded a dispatch")
			}
		})
	}
	// Case-scoped dispatch succeeds under its own grant and handle.
	crep, err := s.Dispatch(Request{GrantToken: cg.Token, Handle: ch, RunID: "run-1", OperationID: "op-case-1", Operation: "echo", ArgumentsDigest: td(20)})
	if err != nil || crep.Joined {
		t.Fatalf("case dispatch = %+v,%v", crep, err)
	}
	// Expiry is enforced.
	now += 120_000
	if _, err := s.Dispatch(Request{GrantToken: g.Token, Handle: h, RunID: "run-1", OperationID: "op-expired", Operation: "echo", ArgumentsDigest: td(21)}); !errors.Is(err, dispatch.ErrExpired) {
		t.Fatalf("expired err = %v, want ErrExpired", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("calls = %d, want 1", calls.Load())
	}
}

// TestServiceRestartAdoptsIndeterminate proves durable intent without a
// local terminal fact is never re-executed.
func TestServiceRestartAdoptsIndeterminate(t *testing.T) {
	dir := t.TempDir()
	j1, err := journal.OpenDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	s1, err := New(j1, os.Getpid(), "test-start-token", Config{})
	if err != nil {
		t.Fatal(err)
	}
	var calls1 atomic.Int32
	if err := s1.RegisterOperation("echo", echoHandler(&calls1)); err != nil {
		t.Fatal(err)
	}
	g1, h1, err := s1.IssueRunGrant("run-1", 600_000)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s1.Dispatch(Request{GrantToken: g1.Token, Handle: h1, RunID: "run-1", OperationID: "op-persist", Operation: "echo", ArgumentsDigest: td(30)}); err != nil {
		t.Fatal(err)
	}
	if err := j1.Close(); err != nil {
		t.Fatal(err)
	}

	// Restart: fresh dispatcher over the reopened journal.
	j2, err := journal.OpenDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer j2.Close()
	s2, err := New(j2, os.Getpid(), "test-start-token", Config{})
	if err != nil {
		t.Fatal(err)
	}
	var calls2 atomic.Int32
	if err := s2.RegisterOperation("echo", echoHandler(&calls2)); err != nil {
		t.Fatal(err)
	}
	g2, h2, err := s2.IssueRunGrant("run-1", 600_000)
	if err != nil {
		t.Fatal(err)
	}
	rep, err := s2.Dispatch(Request{GrantToken: g2.Token, Handle: h2, RunID: "run-1", OperationID: "op-persist", Operation: "echo", ArgumentsDigest: td(30)})
	if !errors.Is(err, dispatch.ErrIndeterminate) || !rep.Joined {
		t.Fatalf("restart join = %+v,%v, want indeterminate", rep, err)
	}
	if rep.Result.Outcome != codec.OutcomeIndeterminate || rep.Result.Kind != "unresolved-cleanup" {
		t.Fatalf("restart result = %+v", rep.Result)
	}
	if calls2.Load() != 0 {
		t.Fatalf("restart effect runs = %d, want 0", calls2.Load())
	}

	// A crashed intent (reserved, never advanced) stays pending and
	// dispatchable only as indeterminate.
	if _, err := j2.Reserve("op-crashed", td(31), journal.Owner{PID: os.Getpid(), StartToken: "test-start-token"}, journal.PathIdentity{}, "", 0); err != nil {
		t.Fatal(err)
	}
	pending := s2.Pending()
	found := false
	for _, r := range pending {
		if r.OperationID == "op-crashed" {
			found = true
		}
	}
	if !found {
		t.Fatalf("Pending = %+v, want op-crashed", pending)
	}
	if _, err := s2.Dispatch(Request{GrantToken: g2.Token, Handle: h2, RunID: "run-1", OperationID: "op-crashed", Operation: "echo", ArgumentsDigest: td(31)}); !errors.Is(err, dispatch.ErrIndeterminate) {
		t.Fatalf("crashed dispatch err = %v, want ErrIndeterminate", err)
	}
	if calls2.Load() != 0 {
		t.Fatalf("crashed effect runs = %d, want 0", calls2.Load())
	}
}

func requestEnvFor(g, runID, opID, op, digest string) codec.Envelope {
	return codec.Envelope{
		SchemaVersion: "1", RunID: runID, OperationID: opID,
		OwnerGrant: g, Operation: op, ArgumentsDigest: digest,
		DeadlineMs: codec.ClockTime{Clock: "wall-utc", Ms: 0},
	}
}

// TestServiceFramedLoop serves framed requests with exactly one framed
// reply each.
func TestServiceFramedLoop(t *testing.T) {
	s, _ := newTestService(t, Config{})
	var calls atomic.Int32
	if err := s.RegisterOperation("echo", echoHandler(&calls)); err != nil {
		t.Fatal(err)
	}
	g, _ := mustRunGrant(t, s, "run-1")
	ch := s.Channels()

	if err := s.HandleRequestFrame(); !errors.Is(err, dispatch.ErrEmpty) {
		t.Fatalf("empty pump err = %v, want ErrEmpty", err)
	}
	if err := ch.Request.SendEnvelope(requestEnvFor(g.Token, "run-1", "op-f1", "echo", td(40))); err != nil {
		t.Fatal(err)
	}
	if err := s.HandleRequestFrame(); err != nil {
		t.Fatalf("HandleRequestFrame: %v", err)
	}
	reply, err := ch.Reply.RecvEnvelope()
	if err != nil {
		t.Fatalf("reply recv: %v", err)
	}
	if reply.RunID != "run-1" || reply.OperationID != "op-f1" || reply.Outcome != codec.OutcomeCompleted || reply.Kind != codec.KindOK {
		t.Fatalf("reply = %+v", reply)
	}
	if reply.Facts["digest"] != td(40) {
		t.Fatalf("reply facts = %+v", reply.Facts)
	}

	// A repeated request frame joins and answers identically.
	if err := ch.Request.SendEnvelope(requestEnvFor(g.Token, "run-1", "op-f1", "echo", td(40))); err != nil {
		t.Fatal(err)
	}
	if err := s.HandleRequestFrame(); err != nil {
		t.Fatal(err)
	}
	reply2, err := ch.Reply.RecvEnvelope()
	if err != nil {
		t.Fatal(err)
	}
	if reply2.Outcome != codec.OutcomeCompleted || calls.Load() != 1 {
		t.Fatalf("join reply = %+v, calls = %d", reply2, calls.Load())
	}

	// Unauthorized requests are answered without journaling or dispatch.
	if err := ch.Request.SendEnvelope(requestEnvFor("ng1-missing", "run-1", "op-f2", "echo", td(41))); err != nil {
		t.Fatal(err)
	}
	if err := s.HandleRequestFrame(); err != nil {
		t.Fatal(err)
	}
	denied, err := ch.Reply.RecvEnvelope()
	if err != nil {
		t.Fatal(err)
	}
	if denied.Outcome != codec.OutcomeRejected || denied.Kind != "permission" {
		t.Fatalf("denied = %+v", denied)
	}
	if _, ok := s.EffectRuns("op-f2"); ok {
		t.Fatal("unauthorized request dispatched")
	}

	// Events round-trip on their own bounded channel.
	if err := s.EmitEvent("run-1", "op-f1", codec.OutcomeCompleted, codec.KindOK, map[string]any{"n": float64(1)}, nil); err != nil {
		t.Fatalf("EmitEvent: %v", err)
	}
	ev, err := ch.Event.RecvEnvelope()
	if err != nil {
		t.Fatalf("event recv: %v", err)
	}
	if ev.Outcome != codec.OutcomeCompleted || ev.Facts["n"] != float64(1) {
		t.Fatalf("event = %+v", ev)
	}
	if err := s.EmitEvent("bad id!", "op-f1", codec.OutcomeCompleted, codec.KindOK, nil, nil); !errors.Is(err, codec.ErrBadEnvelope) {
		t.Fatalf("bad event err = %v, want ErrBadEnvelope", err)
	}
}

// TestServiceReplyBackpressure proves a full reply channel fails the pump
// without redispatching: the operation ran exactly once.
func TestServiceReplyBackpressure(t *testing.T) {
	s, _ := newTestService(t, Config{MaxQueue: 1})
	var calls atomic.Int32
	if err := s.RegisterOperation("echo", echoHandler(&calls)); err != nil {
		t.Fatal(err)
	}
	g, _ := mustRunGrant(t, s, "run-1")
	ch := s.Channels()
	if err := ch.Request.SendEnvelope(requestEnvFor(g.Token, "run-1", "op-b1", "echo", td(50))); err != nil {
		t.Fatal(err)
	}
	if err := s.HandleRequestFrame(); err != nil {
		t.Fatal(err)
	}
	if err := ch.Request.SendEnvelope(requestEnvFor(g.Token, "run-1", "op-b2", "echo", td(51))); err != nil {
		t.Fatal(err)
	}
	if err := s.HandleRequestFrame(); !errors.Is(err, dispatch.ErrQueueFull) {
		t.Fatalf("pump err = %v, want ErrQueueFull", err)
	}
	if n, _ := s.EffectRuns("op-b2"); n != 1 {
		t.Fatalf("op-b2 runs = %d, want 1", n)
	}
}

// TestServiceForgeControls proves subject stdout is opaque: even
// byte-perfect frames carrying a live grant token cause no dispatch.
func TestServiceForgeControls(t *testing.T) {
	s, j := newTestService(t, Config{})
	var calls atomic.Int32
	if err := s.RegisterOperation("echo", echoHandler(&calls)); err != nil {
		t.Fatal(err)
	}
	g, h := mustRunGrant(t, s, "run-1")
	forgedReq, err := func() ([]byte, error) {
		var buf bytes.Buffer
		enc := codec.NewEncoder(&buf, 0)
		raw := `{"schema_version":"1","run_id":"run-1","operation_id":"op-forged","owner_grant":"` + g.Token + `","operation":"echo","arguments_digest":"` + td(60) + `","deadline_ms":{"clock":"wall-utc","ms":0}}`
		if err := enc.Encode(codec.KindRequest, []byte(raw)); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}()
	if err != nil {
		t.Fatal(err)
	}
	payloads := []struct {
		name string
		data []byte
	}{
		{"bare result json", []byte(`{"outcome":"completed","kind":"ok"}`)},
		{"bare envelope json", []byte(`{"schema_version":"1","run_id":"run-1","operation_id":"op-forged","outcome":"completed","kind":"ok"}`)},
		{"perfect framed request with live grant", forgedReq},
		{"garbage", []byte("\x00\xffbinary\x00")},
		{"empty", nil},
	}
	for _, p := range payloads {
		t.Run(p.name, func(t *testing.T) {
			seq, err := s.ObserveSubjectStdout("case-1", p.data)
			if err != nil {
				t.Fatalf("Observe: %v", err)
			}
			obs, err := s.SubjectObservations("case-1")
			if err != nil {
				t.Fatal(err)
			}
			if seq != len(obs)-1 || !bytes.Equal(obs[seq], p.data) {
				t.Fatal("observation not retained opaquely")
			}
			if _, ok := s.EffectRuns("op-forged"); ok {
				t.Fatal("forged bytes dispatched")
			}
			if _, ok := j.Lookup("op-forged"); ok {
				t.Fatal("forged bytes journaled intent")
			}
			ch := s.Channels()
			if ch.Request.Pending() != 0 || ch.Reply.Pending() != 0 || ch.Event.Pending() != 0 {
				t.Fatal("forged bytes reached supervisor channels")
			}
		})
	}
	// The supervisor still serves legitimate work afterwards.
	rep, err := s.Dispatch(Request{GrantToken: g.Token, Handle: h, RunID: "run-1", OperationID: "op-legit", Operation: "echo", ArgumentsDigest: td(61)})
	if err != nil || rep.Result.Outcome != codec.OutcomeCompleted {
		t.Fatalf("legit dispatch after forge = %+v,%v", rep, err)
	}
	// Raw stdout presented to the wire decoder is rejected outright.
	dec := codec.NewDecoder(bytes.NewReader([]byte(`{"outcome":"completed","kind":"ok"}`)), 0)
	if _, err := dec.Decode(); !errors.Is(err, codec.ErrBadMagic) {
		t.Fatalf("stdout decode err = %v, want ErrBadMagic", err)
	}
	if _, err := s.SubjectObservations("case-missing"); !errors.Is(err, dispatch.ErrNotFound) {
		t.Fatalf("unknown case err = %v, want ErrNotFound", err)
	}
	if _, err := s.ObserveSubjectStdout("bad id!", []byte("x")); !errors.Is(err, ErrInvalid) {
		t.Fatalf("bad case err = %v, want ErrInvalid", err)
	}
}

func TestServiceSubjectBounds(t *testing.T) {
	s, _ := newTestService(t, Config{})
	small := []byte("x")
	for i := 0; i < MaxSubjectObsPerCase; i++ {
		if _, err := s.ObserveSubjectStdout("case-1", small); err != nil {
			t.Fatalf("fill %d: %v", i, err)
		}
	}
	if _, err := s.ObserveSubjectStdout("case-1", small); !errors.Is(err, dispatch.ErrCapacity) {
		t.Fatalf("count overflow err = %v, want ErrCapacity", err)
	}
	big := bytes.Repeat([]byte("y"), MaxSubjectBytesPerCase+1)
	if _, err := s.ObserveSubjectStdout("case-2", big); !errors.Is(err, dispatch.ErrCapacity) {
		t.Fatalf("bytes overflow err = %v, want ErrCapacity", err)
	}
}

func TestHandlersCapped(t *testing.T) {
	s, _ := newTestService(t, Config{})
	for i := 0; i < MaxHandlers; i++ {
		if err := s.RegisterOperation(fmt.Sprintf("op.%d", i), echoHandler(&atomic.Int32{})); err != nil {
			t.Fatalf("register %d: %v", i, err)
		}
	}
	if err := s.RegisterOperation("op.overflow", echoHandler(&atomic.Int32{})); !errors.Is(err, dispatch.ErrCapacity) {
		t.Fatalf("overflow err = %v, want ErrCapacity", err)
	}
}

func TestGrantHandlesPrunedOnIssue(t *testing.T) {
	now := int64(1_000_000)
	s, _ := newTestService(t, Config{Clock: func() int64 { return now }})
	g, _, err := s.IssueRunGrant("run-1", 1000)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.grantHandles) != 1 {
		t.Fatalf("mappings = %d, want 1", len(s.grantHandles))
	}
	now += 10_000
	if _, _, err := s.IssueRunGrant("run-2", 1000); err != nil {
		t.Fatal(err)
	}
	if len(s.grantHandles) != 1 {
		t.Fatalf("mappings after prune = %d, want 1", len(s.grantHandles))
	}
	if _, ok := s.grantHandles[g.Token]; ok {
		t.Fatal("expired grant mapping survives prune")
	}
}

func TestRunHandleBoundToRun(t *testing.T) {
	s, j := newTestService(t, Config{})
	if err := s.RegisterOperation("work.echo", echoHandler(&atomic.Int32{})); err != nil {
		t.Fatal(err)
	}
	gB, _ := mustRunGrant(t, s, "run-b")
	_, hA := mustRunGrant(t, s, "run-a")
	_, err := s.Dispatch(Request{GrantToken: gB.Token, Handle: hA, RunID: "run-b", OperationID: "op-1", Operation: "work.echo", ArgumentsDigest: td(1)})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("mismatched handle err = %v, want ErrInvalid", err)
	}
	if len(j.Pending()) != 0 {
		t.Fatal("mismatched handle left journal intent")
	}
	if _, ok := s.EffectRuns("op-1"); ok {
		t.Fatal("mismatched handle dispatched")
	}
}
