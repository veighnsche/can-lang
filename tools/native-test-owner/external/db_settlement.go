// SQL deadline separate from server settlement: P29-side discipline.
//
// This file implements task K26's Go mirror: deadline/settlement facts over
// in-memory doubles only. No real database is contacted, no driver is loaded,
// and no SQL is executed. A SettlementObserver watches one K22 Observer: each
// work dispatch pins a P29 credential leg under the observed namespace grant,
// so dispatch is journaled, handles stay opaque, facts carry digests only,
// and release drains through the registry's own Release.
//
// Driver/server boundary (normative, mirrored in
// tools/runtime/test-services/db-observer/deadline.ts): driver settlement is
// the driver's local view only and never implies server settlement; server
// effect is proven by server acknowledgment alone. Release requires driver
// settlement, a KNOWN server effect (applied or absent), and a fence: an
// unknown server effect blocks cleanup. QD4 owns the D4 verdict.
//
// Cancellation boundary (normative): a cancel request never claims server
// cancellation unless an engine-specific capable grant backs it. The adapter
// implements a server-cancel channel for postgres only; no generic P29 grant
// — namespace, credential, or otherwise — is ever treated as
// server-cancellation authority, and cancellation without a capable grant
// refuses. A recorded cancel proves nothing about the server.
//
// Split with the TS core: the TS core owns the bounded self-check and the
// full fact detail. This file owns P29-journaled dispatch, the pinned
// identity-bound handle per work, digest-only deadline/settlement/ack facts,
// the engine-specific quiescence/fence adapter with retained namespace
// authority, capable-grant cancellation, and acknowledged release. Both
// enforce the same contract: at most one dispatch, visible deadline before
// driver settlement, server acknowledgment before cleanup, fence before
// release, and every failure naming its layer (engine/driver/sql).
//
// (No Example* functions live here: Go only runs them from _test.go files,
// so the integrator places executed examples there.)

package external

import (
	"fmt"
	"sort"
	"sync"

	"github.com/veighnsche/can-lang/tools/native-test-owner/journal"
)

// Settlement bounds and vocabularies. One bounded control covers at most
// eight dispatched works and eight cancel grants.
const (
	// SettlementMaxWorks caps dispatched works per settlement observer.
	SettlementMaxWorks = 8
	// SettlementMaxCancelGrants caps live cancel grants per observer.
	SettlementMaxCancelGrants = 8
)

// SettlementEngines is the closed engine vocabulary for dispatch.
var SettlementEngines = []string{"postgres", "sqlite", "mysql"}

// SettlementDriverOutcomes is the closed driver-settlement vocabulary. Driver
// settlement is the driver's local view only: it never implies the server.
var SettlementDriverOutcomes = []string{"completed", "timed-out"}

// SettlementServerEffects is the closed server-effect vocabulary. "unknown"
// is an explicit recorded effect, not a gap — and it blocks cleanup.
var SettlementServerEffects = []string{"applied", "absent", "unknown"}

func validSettlementEngine(s string) bool {
	for _, e := range SettlementEngines {
		if s == e {
			return true
		}
	}
	return false
}

// settlementCancelCapable reports whether the engine adapter implements a
// server-cancel channel. Only postgres can back a cancellation claim:
// sqlite is in-process (there is no server to cancel) and this adapter
// implements no mysql cancel channel. Capability is a property of the
// engine adapter, never of a generic grant.
func settlementCancelCapable(engine string) bool { return engine == "postgres" }

func validDriverOutcome(s string) bool {
	for _, o := range SettlementDriverOutcomes {
		if s == o {
			return true
		}
	}
	return false
}

func validServerEffect(s string) bool {
	for _, e := range SettlementServerEffects {
		if s == e {
			return true
		}
	}
	return false
}

// SettlementDispatch is the observable record of one dispatch: the
// operation, the pinned work label, the engine, the statement digest only,
// and the handle digest only.
type SettlementDispatch struct {
	OperationID     string
	Work            string
	Engine          string
	StatementDigest string
	TokenDigest     string
}

// DeadlineFacts is the observable visible-deadline record for one work.
type DeadlineFacts struct {
	OperationID string
	Work        string
	Engine      string
	DeadlineMs  int64
	Digest      string
}

// DriverSettlement is one driver-local settlement observation. ServerEffect
// is always "unresolved": the driver view never speaks for the server.
type DriverSettlement struct {
	OperationID  string
	Work         string
	Engine       string
	Outcome      string
	ServerEffect string
	Digest       string
}

// ServerAck is one reconciled server-effect observation.
type ServerAck struct {
	OperationID string
	Work        string
	Engine      string
	Effect      string
	Digest      string
}

// CancelGrantFacts names the engine behind one opaque cancel grant. The raw
// grant is returned alongside exactly once; facts carry its digest only.
type CancelGrantFacts struct {
	Engine      string
	GrantDigest string
}

// CancelFacts records one backed cancellation request. It proves nothing
// about the server and never unblocks cleanup.
type CancelFacts struct {
	OperationID string
	Work        string
	Engine      string
	Proves      string
	Digest      string
}

// SettlementFence is one fence acknowledgment over a work. Settles is always
// "nothing": the fence surveys and never settles or resolves a release.
type SettlementFence struct {
	OperationID string
	Work        string
	Engine      string
	Settles     string
	AckDigest   string
}

// SettlementLease is the retained namespace lease for one work. The lease
// stays retained until explicit release.
type SettlementLease struct {
	OperationID string
	Work        string
	Engine      string
	Namespace   string
	TokenDigest string
	Retained    bool
}

// SettlementReleaseAck acknowledges one work release after reconciliation.
type SettlementReleaseAck struct {
	OperationID string
	Work        string
	Engine      string
	Released    bool
	AckDigest   string
}

type settlementRecord struct {
	opID        string
	work        string
	engine      string
	namespace   string
	tokenDigest string
	stmtDigest  string
	deadlineMs  int64
	hasDeadline bool
	outcome     string
	hasDriver   bool
	effect      string
	hasServer   bool
	hasCancel   bool
	fenced      bool
	released    bool
}

type cancelGrantRec struct {
	engine string
	digest string
}

// SettlementObserver watches one Observer's deadline/settlement path: one
// pinned P29 handle per dispatched work, digest-only deadline, settlement,
// acknowledgment, fence, and lease facts, capable-grant cancellation, and
// acknowledged release. A SettlementObserver is safe for concurrent use;
// the Observer it watches must outlive it.
type SettlementObserver struct {
	mu       sync.Mutex
	o        *Observer
	works    map[string]*settlementRecord
	grants   map[string]*cancelGrantRec
	quiesced map[string]bool
	closed   bool
}

// ObserveSettlement binds a settlement observer to one live Observer.
func ObserveSettlement(o *Observer) (*SettlementObserver, error) {
	if o == nil {
		return nil, observerErr(ObserverLayerEngine, ObserverCodeUnknownNamespace, fmt.Errorf("%w: nil observer", ErrInvalid))
	}
	return &SettlementObserver{
		o:        o,
		works:    make(map[string]*settlementRecord),
		grants:   make(map[string]*cancelGrantRec),
		quiesced: make(map[string]bool),
	}, nil
}

// Dispatch dispatches one SQL work item under the observed namespace grant.
// At most one dispatch runs per operation ID and per work label; dispatch
// to a quiesced engine refuses. The pin is a P29 credential leg: journaled,
// opaque, and identity-bound. The returned fact carries the statement and
// token digests only; the raw statement is never executed and the raw token
// is returned alongside exactly once for the caller's later calls.
func (s *SettlementObserver) Dispatch(opID, work, statement, engine string, ttlMs int64) (SettlementDispatch, string, error) {
	if !validID(opID) {
		return SettlementDispatch{}, "", observerErr(ObserverLayerDriver, ObserverCodeUnknownConn, fmt.Errorf("%w: malformed operation ID", ErrInvalid))
	}
	if !validObserverName(work) {
		return SettlementDispatch{}, "", observerErr(ObserverLayerSQL, ObserverCodeBadStatement, fmt.Errorf("%w: malformed work label", ErrInvalid))
	}
	if statement == "" || len(statement) > MaxObserverNameLen*32 {
		return SettlementDispatch{}, "", observerErr(ObserverLayerSQL, ObserverCodeBadStatement, fmt.Errorf("%w: malformed statement", ErrInvalid))
	}
	if !validSettlementEngine(engine) {
		return SettlementDispatch{}, "", observerErr(ObserverLayerSQL, ObserverCodeBadStatement, fmt.Errorf("%w: unknown engine", ErrInvalid))
	}
	// Held across PinConnection: the work/opID checks and the record form
	// one atomic dispatch, so concurrent same-work dispatches cannot both
	// pin. s.mu is outermost everywhere (the Observer and registry never
	// call back here), so no lock cycle exists.
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return SettlementDispatch{}, "", observerErr(ObserverLayerEngine, ObserverCodeClosedHandle, fmt.Errorf("%w: settlement observer closed", ErrInvalid))
	}
	if s.quiesced[engine] {
		return SettlementDispatch{}, "", observerErr(ObserverLayerEngine, ObserverCodeClosedHandle, fmt.Errorf("%w: engine %q quiesced", ErrQuiesced, engine))
	}
	if len(s.works) >= SettlementMaxWorks {
		return SettlementDispatch{}, "", observerErr(ObserverLayerEngine, ObserverCodeCapacity, fmt.Errorf("%w: work table full", ErrCapacity))
	}
	if _, dup := s.works[opID]; dup {
		return SettlementDispatch{}, "", observerErr(ObserverLayerDriver, ObserverCodeUnknownConn, fmt.Errorf("%w: operation %q already dispatched", ErrInvalid, opID))
	}
	for _, rec := range s.works {
		if rec.work == work {
			return SettlementDispatch{}, "", observerErr(ObserverLayerDriver, ObserverCodeConnBusy, fmt.Errorf("%w: work %q already dispatched", ErrDenied, work))
		}
	}
	// The registry mints the opaque handle and journals the credential leg.
	conn, token, err := s.o.PinConnection(opID, work, ttlMs)
	if err != nil {
		return SettlementDispatch{}, "", err
	}
	rec := &settlementRecord{
		opID: opID, work: work, engine: engine, namespace: conn.Namespace,
		tokenDigest: handleDigest(token), stmtDigest: txnDigest("set-stmt", statement),
	}
	s.works[opID] = rec
	return SettlementDispatch{
		OperationID: opID, Work: work, Engine: engine,
		StatementDigest: rec.stmtDigest, TokenDigest: rec.tokenDigest,
	}, token, nil
}

// lookupLocked verifies one dispatched work and its token. Unknown works
// and invented or swapped tokens are never authority.
func (s *SettlementObserver) lookupLocked(opID, token string) (*settlementRecord, error) {
	rec, ok := s.works[opID]
	if !ok || rec.released {
		return nil, observerErr(ObserverLayerDriver, ObserverCodeUnknownConn, fmt.Errorf("%w: operation %q", ErrNotFound, opID))
	}
	if token == "" || handleDigest(token) != rec.tokenDigest {
		return nil, observerErr(ObserverLayerDriver, ObserverCodeForgedToken, fmt.Errorf("%w: work token is not the pinned token", ErrDenied))
	}
	return rec, nil
}

// DispatchFacts re-reads the dispatch record for one work.
func (s *SettlementObserver) DispatchFacts(opID string) (SettlementDispatch, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.works[opID]
	if !ok {
		return SettlementDispatch{}, observerErr(ObserverLayerDriver, ObserverCodeUnknownConn, fmt.Errorf("%w: operation %q", ErrNotFound, opID))
	}
	return SettlementDispatch{
		OperationID: rec.opID, Work: rec.work, Engine: rec.engine,
		StatementDigest: rec.stmtDigest, TokenDigest: rec.tokenDigest,
	}, nil
}

// ObserveDeadline records the visible deadline for one dispatched work: the
// driver-visible tick, supplied by the caller (no timers run here). Single
// record: observing twice refuses. Driver settlement requires the deadline
// first, so the deadline stays its own fact.
func (s *SettlementObserver) ObserveDeadline(opID, token string, deadlineMs int64) (DeadlineFacts, error) {
	if deadlineMs < 0 {
		return DeadlineFacts{}, observerErr(ObserverLayerSQL, ObserverCodeBadStatement, fmt.Errorf("%w: deadline must be a non-negative tick", ErrInvalid))
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return DeadlineFacts{}, observerErr(ObserverLayerEngine, ObserverCodeClosedHandle, fmt.Errorf("%w: settlement observer closed", ErrInvalid))
	}
	rec, err := s.lookupLocked(opID, token)
	if err != nil {
		return DeadlineFacts{}, err
	}
	if rec.hasDeadline {
		return DeadlineFacts{}, observerErr(ObserverLayerDriver, ObserverCodeConnBusy, fmt.Errorf("%w: deadline already observed", ErrDenied))
	}
	rec.deadlineMs = deadlineMs
	rec.hasDeadline = true
	return DeadlineFacts{
		OperationID: rec.opID, Work: rec.work, Engine: rec.engine, DeadlineMs: deadlineMs,
		Digest: txnDigest("set-deadline", rec.work, rec.opID, rec.engine, fmt.Sprint(deadlineMs)),
	}, nil
}

// Deadline re-reads the visible-deadline facts for one work. Missing
// evidence stays missing: works with no observed deadline throw.
func (s *SettlementObserver) Deadline(opID string) (DeadlineFacts, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.works[opID]
	if !ok {
		return DeadlineFacts{}, observerErr(ObserverLayerDriver, ObserverCodeUnknownConn, fmt.Errorf("%w: operation %q", ErrNotFound, opID))
	}
	if !rec.hasDeadline {
		return DeadlineFacts{}, observerErr(ObserverLayerDriver, ObserverCodeNoConversation, fmt.Errorf("%w: no deadline observed", ErrInvalid))
	}
	return DeadlineFacts{
		OperationID: rec.opID, Work: rec.work, Engine: rec.engine, DeadlineMs: rec.deadlineMs,
		Digest: txnDigest("set-deadline", rec.work, rec.opID, rec.engine, fmt.Sprint(rec.deadlineMs)),
	}, nil
}

// RecordDriverSettlement records the driver's local settlement: what the
// driver saw when the visible deadline passed. Facts only, driver-local:
// the fact carries ServerEffect "unresolved", so it can never be read as
// server settlement. Single record.
func (s *SettlementObserver) RecordDriverSettlement(opID, token, outcome string) (DriverSettlement, error) {
	if !validDriverOutcome(outcome) {
		return DriverSettlement{}, observerErr(ObserverLayerSQL, ObserverCodeBadStatement, fmt.Errorf("%w: unknown driver outcome", ErrInvalid))
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return DriverSettlement{}, observerErr(ObserverLayerEngine, ObserverCodeClosedHandle, fmt.Errorf("%w: settlement observer closed", ErrInvalid))
	}
	rec, err := s.lookupLocked(opID, token)
	if err != nil {
		return DriverSettlement{}, err
	}
	if !rec.hasDeadline {
		return DriverSettlement{}, observerErr(ObserverLayerDriver, ObserverCodeNoConversation, fmt.Errorf("%w: observe the visible deadline first", ErrInvalid))
	}
	if rec.hasDriver {
		return DriverSettlement{}, observerErr(ObserverLayerDriver, ObserverCodeConnBusy, fmt.Errorf("%w: driver already settled", ErrDenied))
	}
	rec.outcome = outcome
	rec.hasDriver = true
	return DriverSettlement{
		OperationID: rec.opID, Work: rec.work, Engine: rec.engine,
		Outcome: outcome, ServerEffect: "unresolved",
		Digest: txnDigest("set-driver", rec.work, rec.opID, rec.engine, outcome),
	}, nil
}

// DriverSettlement re-reads the driver-local settlement for one work.
func (s *SettlementObserver) DriverSettlement(opID string) (DriverSettlement, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.works[opID]
	if !ok {
		return DriverSettlement{}, observerErr(ObserverLayerDriver, ObserverCodeUnknownConn, fmt.Errorf("%w: operation %q", ErrNotFound, opID))
	}
	if !rec.hasDriver {
		return DriverSettlement{}, observerErr(ObserverLayerDriver, ObserverCodeNoConversation, fmt.Errorf("%w: no driver settlement recorded", ErrInvalid))
	}
	return DriverSettlement{
		OperationID: rec.opID, Work: rec.work, Engine: rec.engine,
		Outcome: rec.outcome, ServerEffect: "unresolved",
		Digest: txnDigest("set-driver", rec.work, rec.opID, rec.engine, rec.outcome),
	}, nil
}

// RecordServerAck records the server acknowledgment: the reconciled server
// effect. The acknowledgment requires driver settlement first and is
// single-record. "unknown" is an explicit recorded effect, not a gap — and
// it blocks cleanup.
func (s *SettlementObserver) RecordServerAck(opID, token, effect string) (ServerAck, error) {
	if !validServerEffect(effect) {
		return ServerAck{}, observerErr(ObserverLayerSQL, ObserverCodeBadStatement, fmt.Errorf("%w: unknown server effect", ErrInvalid))
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ServerAck{}, observerErr(ObserverLayerEngine, ObserverCodeClosedHandle, fmt.Errorf("%w: settlement observer closed", ErrInvalid))
	}
	rec, err := s.lookupLocked(opID, token)
	if err != nil {
		return ServerAck{}, err
	}
	if !rec.hasDriver {
		return ServerAck{}, observerErr(ObserverLayerDriver, ObserverCodeNoConversation, fmt.Errorf("%w: record driver settlement first", ErrInvalid))
	}
	if rec.hasServer {
		return ServerAck{}, observerErr(ObserverLayerDriver, ObserverCodeConnBusy, fmt.Errorf("%w: server already acknowledged", ErrDenied))
	}
	rec.effect = effect
	rec.hasServer = true
	return ServerAck{
		OperationID: rec.opID, Work: rec.work, Engine: rec.engine, Effect: effect,
		Digest: txnDigest("set-server", rec.work, rec.opID, rec.engine, effect),
	}, nil
}

// ServerAck re-reads the server acknowledgment for one work.
func (s *SettlementObserver) ServerAck(opID string) (ServerAck, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.works[opID]
	if !ok {
		return ServerAck{}, observerErr(ObserverLayerDriver, ObserverCodeUnknownConn, fmt.Errorf("%w: operation %q", ErrNotFound, opID))
	}
	if !rec.hasServer {
		return ServerAck{}, observerErr(ObserverLayerDriver, ObserverCodeNoConversation, fmt.Errorf("%w: no server acknowledgment recorded", ErrInvalid))
	}
	return ServerAck{
		OperationID: rec.opID, Work: rec.work, Engine: rec.engine, Effect: rec.effect,
		Digest: txnDigest("set-server", rec.work, rec.opID, rec.engine, rec.effect),
	}, nil
}

// IssueCancelGrant mints one engine-scoped cancel grant. Only
// cancel-capable engines can back a cancellation claim: issuing a grant
// for any other engine refuses rather than emitting an unsupported claim.
// The grant is opaque; facts carry its digest only.
func (s *SettlementObserver) IssueCancelGrant(engine string) (CancelGrantFacts, string, error) {
	if !validSettlementEngine(engine) {
		return CancelGrantFacts{}, "", observerErr(ObserverLayerSQL, ObserverCodeBadStatement, fmt.Errorf("%w: unknown engine", ErrInvalid))
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return CancelGrantFacts{}, "", observerErr(ObserverLayerEngine, ObserverCodeClosedHandle, fmt.Errorf("%w: settlement observer closed", ErrInvalid))
	}
	if !settlementCancelCapable(engine) {
		return CancelGrantFacts{}, "", observerErr(ObserverLayerSQL, ObserverCodeBadStatement, fmt.Errorf("%w: engine %q has no server-cancel channel", ErrDenied, engine))
	}
	if len(s.grants) >= SettlementMaxCancelGrants {
		return CancelGrantFacts{}, "", observerErr(ObserverLayerEngine, ObserverCodeCapacity, fmt.Errorf("%w: cancel-grant table full", ErrCapacity))
	}
	grant, err := mintToken()
	if err != nil {
		return CancelGrantFacts{}, "", observerErr(ObserverLayerEngine, ObserverCodeCapacity, err)
	}
	if _, dup := s.grants[grant]; dup {
		return CancelGrantFacts{}, "", observerErr(ObserverLayerEngine, ObserverCodeCapacity, fmt.Errorf("%w: cancel grant collision", ErrInvalid))
	}
	s.grants[grant] = &cancelGrantRec{engine: engine, digest: handleDigest(grant)}
	return CancelGrantFacts{Engine: engine, GrantDigest: handleDigest(grant)}, grant, nil
}

// RequestCancel requests cancellation of one dispatched work under a
// capable grant. The grant must be this observer's own live grant for the
// work's engine: unknown, invented, wrong-engine, or generic P29 grants
// refuse with forged-token — a generic handle is never cancellation
// authority. A recorded cancel proves nothing about the server, never
// changes the server effect, and never unblocks cleanup; once the server
// has acknowledged, a cancel is stale and refuses.
func (s *SettlementObserver) RequestCancel(opID, token, grant string) (CancelFacts, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return CancelFacts{}, observerErr(ObserverLayerEngine, ObserverCodeClosedHandle, fmt.Errorf("%w: settlement observer closed", ErrInvalid))
	}
	rec, err := s.lookupLocked(opID, token)
	if err != nil {
		return CancelFacts{}, err
	}
	held, ok := s.grants[grant]
	if !ok {
		return CancelFacts{}, observerErr(ObserverLayerDriver, ObserverCodeForgedToken, fmt.Errorf("%w: cancel grant is unknown", ErrDenied))
	}
	if held.engine != rec.engine {
		return CancelFacts{}, observerErr(ObserverLayerDriver, ObserverCodeForgedToken, fmt.Errorf("%w: cancel grant is for another engine", ErrDenied))
	}
	if rec.hasServer {
		return CancelFacts{}, observerErr(ObserverLayerDriver, ObserverCodeConnBusy, fmt.Errorf("%w: server already acknowledged; cancel is stale", ErrDenied))
	}
	if rec.hasCancel {
		return CancelFacts{}, observerErr(ObserverLayerDriver, ObserverCodeConnBusy, fmt.Errorf("%w: cancel already requested", ErrDenied))
	}
	rec.hasCancel = true
	return CancelFacts{
		OperationID: rec.opID, Work: rec.work, Engine: rec.engine,
		Proves: "nothing-about-server",
		Digest: txnDigest("set-cancel", rec.work, rec.opID, rec.engine),
	}, nil
}

// Cancel re-reads the cancellation facts for one work.
func (s *SettlementObserver) Cancel(opID string) (CancelFacts, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.works[opID]
	if !ok {
		return CancelFacts{}, observerErr(ObserverLayerDriver, ObserverCodeUnknownConn, fmt.Errorf("%w: operation %q", ErrNotFound, opID))
	}
	if !rec.hasCancel {
		return CancelFacts{}, observerErr(ObserverLayerDriver, ObserverCodeNoConversation, fmt.Errorf("%w: no cancel requested", ErrInvalid))
	}
	return CancelFacts{
		OperationID: rec.opID, Work: rec.work, Engine: rec.engine,
		Proves: "nothing-about-server",
		Digest: txnDigest("set-cancel", rec.work, rec.opID, rec.engine),
	}, nil
}

// QuiesceEngine quiesces one engine: new dispatches there refuse, and the
// call reports that engine's live work operations in sorted order.
// Idempotent: repeats return the same live set without further effect.
func (s *SettlementObserver) QuiesceEngine(engine string) ([]string, error) {
	if !validSettlementEngine(engine) {
		return nil, observerErr(ObserverLayerSQL, ObserverCodeBadStatement, fmt.Errorf("%w: unknown engine", ErrInvalid))
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.quiesced[engine] = true
	var live []string
	for id, rec := range s.works {
		if !rec.released && rec.engine == engine {
			live = append(live, id)
		}
	}
	sort.Strings(live)
	return live, nil
}

// Fence fences one work under its quiesced engine. Fencing requires
// quiescence first; repeats join the retained acknowledgment. The fence
// surveys — it is recorded even with an unknown server effect — and
// settles nothing: cleanup still requires a known server effect plus
// release.
func (s *SettlementObserver) Fence(opID, token string) (SettlementFence, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return SettlementFence{}, observerErr(ObserverLayerEngine, ObserverCodeClosedHandle, fmt.Errorf("%w: settlement observer closed", ErrInvalid))
	}
	rec, err := s.lookupLocked(opID, token)
	if err != nil {
		return SettlementFence{}, err
	}
	if !s.quiesced[rec.engine] {
		return SettlementFence{}, observerErr(ObserverLayerDriver, ObserverCodeNoConversation, fmt.Errorf("%w: quiesce the engine before fencing", ErrInvalid))
	}
	rec.fenced = true
	return SettlementFence{
		OperationID: rec.opID, Work: rec.work, Engine: rec.engine,
		Settles:   "nothing",
		AckDigest: txnDigest("set-fence", rec.work, rec.opID, rec.engine),
	}, nil
}

// FenceFacts re-reads the fence acknowledgment for one work.
func (s *SettlementObserver) FenceFacts(opID string) (SettlementFence, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.works[opID]
	if !ok {
		return SettlementFence{}, observerErr(ObserverLayerDriver, ObserverCodeUnknownConn, fmt.Errorf("%w: operation %q", ErrNotFound, opID))
	}
	if !rec.fenced {
		return SettlementFence{}, observerErr(ObserverLayerDriver, ObserverCodeNoConversation, fmt.Errorf("%w: no fence recorded", ErrInvalid))
	}
	return SettlementFence{
		OperationID: rec.opID, Work: rec.work, Engine: rec.engine,
		Settles:   "nothing",
		AckDigest: txnDigest("set-fence", rec.work, rec.opID, rec.engine),
	}, nil
}

// Lease reports the retained namespace lease for one work. The namespace
// stays retained until explicit release; after release the lease shows
// released.
func (s *SettlementObserver) Lease(opID string) (SettlementLease, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.works[opID]
	if !ok {
		return SettlementLease{}, observerErr(ObserverLayerDriver, ObserverCodeUnknownConn, fmt.Errorf("%w: operation %q", ErrNotFound, opID))
	}
	return SettlementLease{
		OperationID: rec.opID, Work: rec.work, Engine: rec.engine,
		Namespace: rec.namespace, TokenDigest: rec.tokenDigest, Retained: !rec.released,
	}, nil
}

// Release releases one work after full reconciliation. Release requires
// driver settlement, a KNOWN server effect, and a fence: an unknown server
// effect blocks cleanup, as does anything missing on the path. The registry
// walks the journal to closed, revokes the grant, and settles the retained
// charge.
func (s *SettlementObserver) Release(opID, token string, self journal.Owner) (SettlementReleaseAck, error) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return SettlementReleaseAck{}, observerErr(ObserverLayerEngine, ObserverCodeClosedHandle, fmt.Errorf("%w: settlement observer closed", ErrInvalid))
	}
	rec, err := s.lookupLocked(opID, token)
	if err != nil {
		s.mu.Unlock()
		return SettlementReleaseAck{}, err
	}
	if !rec.hasDriver {
		s.mu.Unlock()
		return SettlementReleaseAck{}, observerErr(ObserverLayerDriver, ObserverCodeConnBusy, fmt.Errorf("%w: record driver settlement before release", ErrDenied))
	}
	if !rec.hasServer {
		s.mu.Unlock()
		return SettlementReleaseAck{}, observerErr(ObserverLayerDriver, ObserverCodeNoConversation, fmt.Errorf("%w: no server acknowledgment recorded", ErrInvalid))
	}
	if rec.effect == "unknown" {
		s.mu.Unlock()
		return SettlementReleaseAck{}, observerErr(ObserverLayerDriver, ObserverCodeConnBusy, fmt.Errorf("%w: server effect unknown blocks cleanup", ErrDenied))
	}
	if !rec.fenced {
		s.mu.Unlock()
		return SettlementReleaseAck{}, observerErr(ObserverLayerDriver, ObserverCodeNoConversation, fmt.Errorf("%w: fence before release", ErrInvalid))
	}
	s.mu.Unlock()
	if err := s.o.ReleaseConn(opID, token, self); err != nil {
		return SettlementReleaseAck{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rec.released = true
	return SettlementReleaseAck{
		OperationID: rec.opID, Work: rec.work, Engine: rec.engine, Released: true,
		AckDigest: txnDigest("set-release", rec.work, rec.opID, rec.engine),
	}, nil
}

// ReleaseAck re-reads the release acknowledgment for one released work.
func (s *SettlementObserver) ReleaseAck(opID string) (SettlementReleaseAck, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.works[opID]
	if !ok {
		return SettlementReleaseAck{}, observerErr(ObserverLayerDriver, ObserverCodeUnknownConn, fmt.Errorf("%w: operation %q", ErrNotFound, opID))
	}
	if !rec.released {
		return SettlementReleaseAck{}, observerErr(ObserverLayerDriver, ObserverCodeNoConversation, fmt.Errorf("%w: no release recorded", ErrInvalid))
	}
	return SettlementReleaseAck{
		OperationID: rec.opID, Work: rec.work, Engine: rec.engine, Released: true,
		AckDigest: txnDigest("set-release", rec.work, rec.opID, rec.engine),
	}, nil
}

// Close retires the settlement observer. Works still held must be released
// first via Release so no charge is stranded; Close itself settles nothing.
func (s *SettlementObserver) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return observerErr(ObserverLayerEngine, ObserverCodeClosedHandle, fmt.Errorf("%w: settlement observer already closed", ErrInvalid))
	}
	for _, rec := range s.works {
		if !rec.released {
			return observerErr(ObserverLayerEngine, ObserverCodeClosedHandle, fmt.Errorf("%w: work %q still dispatched", ErrDependency, rec.opID))
		}
	}
	s.closed = true
	return nil
}
