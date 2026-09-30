// Fresh-pool transaction identity: P29-side settlement/ack discipline.
//
// This file implements task K24's Go mirror: transaction identity facts over
// in-memory doubles only. No real database is contacted, no driver is loaded,
// and no SQL is executed. A TxnObserver watches one K22 Observer: each actor
// entry pins a P29 credential leg under the observed namespace grant, so
// entry is journaled, handles stay opaque, facts carry digests only, and
// release drains through the registry's own Release.
//
// Facts-only boundary (normative, mirrored in
// tools/runtime/test-services/db-observer/transactions.ts): this file records
// callback entry, actor identity, LAST_INSERT_ID, settlement, and release
// facts only. It never decides rollback expectations: there is no expected
// outcome argument and no verdict return. QD2 owns the D2 verdict.
//
// Split with the TS core: the TS core owns typed exact id cells, comparison
// detail, and the bounded self-check. This file owns P29-journaled actor
// admission, the pinned identity-bound handle per actor, digest-only
// settlement observations, and release acknowledgment. Both enforce the same
// contract: one entry per actor, token-bound identity, settlement facts
// without expectations, release only after settlement, and every failure
// naming its layer (engine/driver/sql).
//
// A swapped actor presents a foreign token and fails token verification; a
// wrong-handle LAST_INSERT_ID claim carries a foreign handle digest and
// mismatches on identity even when the id digest is coincidentally correct.
//
// (No Example* functions live here: Go only runs them from _test.go files,
// so the integrator places executed examples there.)

package external

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/veighnsche/can-lang/tools/native-test-owner/journal"
)

// Transaction bounds and vocabularies. One bounded fresh-pool control covers
// exactly eight actors with no warmup.
const (
	// TxnMaxActors caps observed actors per transaction observer.
	TxnMaxActors = 8
)

// TxnEngines is the closed engine vocabulary for settlement observations.
var TxnEngines = []string{"postgres", "sqlite", "mysql"}

// TxnOutcomes is the closed settlement-outcome vocabulary. Outcomes are
// recorded facts, never expectations: "unknown" is an explicit
// unsettled-visible outcome, not a gap.
var TxnOutcomes = []string{"committed", "rolled-back", "unknown"}

func validTxnEngine(s string) bool {
	for _, e := range TxnEngines {
		if s == e {
			return true
		}
	}
	return false
}

func validTxnOutcome(s string) bool {
	for _, o := range TxnOutcomes {
		if s == o {
			return true
		}
	}
	return false
}

func txnDigest(parts ...string) string {
	h := sha256.New()
	for i, p := range parts {
		if i > 0 {
			h.Write([]byte("|"))
		}
		h.Write([]byte(p))
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

// TxnEntry is the observable record of one callback entry: the operation,
// the pinned actor name, the handle digest only, and the entry sequence.
type TxnEntry struct {
	OperationID string
	Actor       string
	TokenDigest string
	EntrySeq    int
}

// TxnActorFacts is the observable identity record of one entered actor.
type TxnActorFacts struct {
	OperationID string
	Actor       string
	TokenDigest string
	EntrySeq    int
	Settled     bool
	Released    bool
}

// TxnInsertFacts binds one LAST_INSERT_ID observation to its handle. Both
// digests cross; the raw token and the raw id never do.
type TxnInsertFacts struct {
	OperationID string
	Actor       string
	TokenDigest string
	IDDigest    string
}

// TxnSettlement is one engine-specific settlement observation: the recorded
// outcome plus the engine tag it was observed on, under one digest.
type TxnSettlement struct {
	OperationID string
	Actor       string
	Outcome     string
	Engine      string
	Digest      string
}

// TxnReleaseAck acknowledges one handle release after settlement.
type TxnReleaseAck struct {
	OperationID string
	Actor       string
	Released    bool
	AckDigest   string
}

// TxnComparison reports an identity comparison: -1 names an identity
// (actor/handle) mismatch, 0 names a value mismatch. Identity is checked
// independently of value.
type TxnComparison struct {
	Match      bool
	Mismatches []int
}

type txnRecord struct {
	opID        string
	actor       string
	tokenDigest string
	entrySeq    int
	idDigest    string
	hasID       bool
	outcome     string
	engine      string
	settled     bool
	released    bool
}

// TxnObserver watches one Observer's transactions: one pinned P29 handle per
// actor, digest-only identity and settlement facts, and acknowledged
// release. A TxnObserver is safe for concurrent use; the Observer it watches
// must outlive it.
type TxnObserver struct {
	mu     sync.Mutex
	o      *Observer
	actors map[string]*txnRecord
	seq    int
	closed bool
}

// ObserveTransactions binds a transaction observer to one live Observer.
func ObserveTransactions(o *Observer) (*TxnObserver, error) {
	if o == nil {
		return nil, observerErr(ObserverLayerEngine, ObserverCodeUnknownNamespace, fmt.Errorf("%w: nil observer", ErrInvalid))
	}
	return &TxnObserver{o: o, actors: make(map[string]*txnRecord)}, nil
}

// EnterCallback observes one actor entering its transaction callback on the
// fresh pool. Entry pins one P29 credential leg named for the actor under
// the observed namespace grant: journaled, opaque, and identity-bound. The
// returned fact carries the token digest only; the raw token is returned
// alongside exactly once for the caller's later calls. No warmup: the first
// entry lands directly in the facts table.
func (t *TxnObserver) EnterCallback(opID, actor string, ttlMs int64) (TxnEntry, string, error) {
	if !validID(opID) {
		return TxnEntry{}, "", observerErr(ObserverLayerDriver, ObserverCodeUnknownConn, fmt.Errorf("%w: malformed operation ID", ErrInvalid))
	}
	if !validObserverName(actor) {
		return TxnEntry{}, "", observerErr(ObserverLayerSQL, ObserverCodeBadStatement, fmt.Errorf("%w: malformed actor name", ErrInvalid))
	}
	// Held across PinConnection: the actor/opID checks and the record form
	// one atomic entry, so concurrent same-actor entries cannot both pin.
	// t.mu is outermost everywhere (the Observer and registry never call
	// back here), so no lock cycle exists.
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return TxnEntry{}, "", observerErr(ObserverLayerEngine, ObserverCodeClosedHandle, fmt.Errorf("%w: transaction observer closed", ErrInvalid))
	}
	if len(t.actors) >= TxnMaxActors {
		return TxnEntry{}, "", observerErr(ObserverLayerEngine, ObserverCodeCapacity, fmt.Errorf("%w: actor table full", ErrCapacity))
	}
	if _, dup := t.actors[opID]; dup {
		return TxnEntry{}, "", observerErr(ObserverLayerDriver, ObserverCodeUnknownConn, fmt.Errorf("%w: operation %q already entered", ErrInvalid, opID))
	}
	for _, rec := range t.actors {
		if rec.actor == actor {
			return TxnEntry{}, "", observerErr(ObserverLayerDriver, ObserverCodeConnBusy, fmt.Errorf("%w: actor %q already entered", ErrDenied, actor))
		}
	}
	// The registry mints the opaque handle and journals the credential leg.
	_, token, err := t.o.PinConnection(opID, actor, ttlMs)
	if err != nil {
		return TxnEntry{}, "", err
	}
	rec := &txnRecord{opID: opID, actor: actor, tokenDigest: handleDigest(token), entrySeq: t.seq}
	t.seq++
	t.actors[opID] = rec
	return TxnEntry{OperationID: opID, Actor: actor, TokenDigest: rec.tokenDigest, EntrySeq: rec.entrySeq}, token, nil
}

// lookupLocked verifies one entered actor and its token. Unknown actors and
// invented or swapped tokens are never authority.
func (t *TxnObserver) lookupLocked(opID, token string) (*txnRecord, error) {
	rec, ok := t.actors[opID]
	if !ok || rec.released {
		return nil, observerErr(ObserverLayerDriver, ObserverCodeUnknownConn, fmt.Errorf("%w: operation %q", ErrNotFound, opID))
	}
	if token == "" || handleDigest(token) != rec.tokenDigest {
		return nil, observerErr(ObserverLayerDriver, ObserverCodeForgedToken, fmt.Errorf("%w: actor token is not the pinned token", ErrDenied))
	}
	return rec, nil
}

// ActorFacts reports the observable identity record of one entered actor.
func (t *TxnObserver) ActorFacts(opID string) (TxnActorFacts, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	rec, ok := t.actors[opID]
	if !ok {
		return TxnActorFacts{}, observerErr(ObserverLayerDriver, ObserverCodeUnknownConn, fmt.Errorf("%w: operation %q", ErrNotFound, opID))
	}
	return TxnActorFacts{
		OperationID: rec.opID, Actor: rec.actor, TokenDigest: rec.tokenDigest,
		EntrySeq: rec.entrySeq, Settled: rec.settled, Released: rec.released,
	}, nil
}

// RecordInsert records the LAST_INSERT_ID visible on the actor's own pinned
// handle. Only the id digest is stored, bound to the handle digest: a
// wrong-handle read is detectable by identity even when the id matches.
func (t *TxnObserver) RecordInsert(opID, token, id string) (TxnInsertFacts, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return TxnInsertFacts{}, observerErr(ObserverLayerEngine, ObserverCodeClosedHandle, fmt.Errorf("%w: transaction observer closed", ErrInvalid))
	}
	rec, err := t.lookupLocked(opID, token)
	if err != nil {
		return TxnInsertFacts{}, err
	}
	if rec.settled {
		return TxnInsertFacts{}, observerErr(ObserverLayerDriver, ObserverCodeNoConversation, fmt.Errorf("%w: transaction already settled", ErrInvalid))
	}
	// The digest covers the id value only: the same id on two handles
	// yields the same digest, so handle binding is proven by the
	// TokenDigest field alone and a wrong-handle claim mismatches on
	// identity even when the id is coincidentally correct.
	rec.idDigest = txnDigest("txn-id", id)
	rec.hasID = true
	return TxnInsertFacts{
		OperationID: rec.opID, Actor: rec.actor,
		TokenDigest: rec.tokenDigest, IDDigest: rec.idDigest,
	}, nil
}

// LastInsertID reports the LAST_INSERT_ID facts bound to one actor's handle.
// Missing evidence stays missing: actors with no recorded id throw.
func (t *TxnObserver) LastInsertID(opID string) (TxnInsertFacts, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	rec, ok := t.actors[opID]
	if !ok {
		return TxnInsertFacts{}, observerErr(ObserverLayerDriver, ObserverCodeUnknownConn, fmt.Errorf("%w: operation %q", ErrNotFound, opID))
	}
	if !rec.hasID {
		return TxnInsertFacts{}, observerErr(ObserverLayerDriver, ObserverCodeNoConversation, fmt.Errorf("%w: no LAST_INSERT_ID recorded", ErrInvalid))
	}
	return TxnInsertFacts{
		OperationID: rec.opID, Actor: rec.actor,
		TokenDigest: rec.tokenDigest, IDDigest: rec.idDigest,
	}, nil
}

// CompareLastInsertID compares one claimed LAST_INSERT_ID against the
// observer's bound facts. Identity (actor/handle digest) is compared
// independently of the id digest: a claim from the wrong actor or the wrong
// handle mismatches with -1 even when the id digest is coincidentally
// correct, and a wrong id mismatches with 0.
func CompareLastInsertID(stored TxnInsertFacts, claimedActor, claimedHandleDigest, claimedIDDigest string) TxnComparison {
	var mismatches []int
	if claimedActor != stored.Actor || claimedHandleDigest != stored.TokenDigest {
		mismatches = append(mismatches, -1)
	}
	if claimedIDDigest != stored.IDDigest {
		mismatches = append(mismatches, 0)
	}
	return TxnComparison{Match: len(mismatches) == 0, Mismatches: mismatches}
}

// RecordSettlement records the settled outcome of one actor's transaction as
// observed on the given engine. Facts only: recording a settlement states
// what the engine showed, never what the transaction should have done.
func (t *TxnObserver) RecordSettlement(opID, token, outcome, engine string) (TxnSettlement, error) {
	if !validTxnOutcome(outcome) {
		return TxnSettlement{}, observerErr(ObserverLayerSQL, ObserverCodeBadStatement, fmt.Errorf("%w: unknown settlement outcome", ErrInvalid))
	}
	if !validTxnEngine(engine) {
		return TxnSettlement{}, observerErr(ObserverLayerSQL, ObserverCodeBadStatement, fmt.Errorf("%w: unknown engine", ErrInvalid))
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return TxnSettlement{}, observerErr(ObserverLayerEngine, ObserverCodeClosedHandle, fmt.Errorf("%w: transaction observer closed", ErrInvalid))
	}
	rec, err := t.lookupLocked(opID, token)
	if err != nil {
		return TxnSettlement{}, err
	}
	if rec.settled {
		return TxnSettlement{}, observerErr(ObserverLayerDriver, ObserverCodeConnBusy, fmt.Errorf("%w: transaction already settled", ErrDenied))
	}
	rec.outcome = outcome
	rec.engine = engine
	rec.settled = true
	return TxnSettlement{
		OperationID: rec.opID, Actor: rec.actor, Outcome: outcome, Engine: engine,
		Digest: txnDigest("txn-settlement", rec.actor, rec.opID, outcome, engine),
	}, nil
}

// Settlement reports the engine-specific settlement observation for one
// actor. Unsettled actors throw rather than yielding a default.
func (t *TxnObserver) Settlement(opID string) (TxnSettlement, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	rec, ok := t.actors[opID]
	if !ok {
		return TxnSettlement{}, observerErr(ObserverLayerDriver, ObserverCodeUnknownConn, fmt.Errorf("%w: operation %q", ErrNotFound, opID))
	}
	if !rec.settled {
		return TxnSettlement{}, observerErr(ObserverLayerDriver, ObserverCodeNoConversation, fmt.Errorf("%w: no settlement recorded", ErrInvalid))
	}
	return TxnSettlement{
		OperationID: rec.opID, Actor: rec.actor, Outcome: rec.outcome, Engine: rec.engine,
		Digest: txnDigest("txn-settlement", rec.actor, rec.opID, rec.outcome, rec.engine),
	}, nil
}

// Release ends the actor's pinned handle after settlement and acknowledges
// the release. Release before settlement is refused so unsettled work is
// never silently dropped; the registry walks the journal to closed, revokes
// the grant, and settles the retained charge.
func (t *TxnObserver) Release(opID, token string, self journal.Owner) (TxnReleaseAck, error) {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return TxnReleaseAck{}, observerErr(ObserverLayerEngine, ObserverCodeClosedHandle, fmt.Errorf("%w: transaction observer closed", ErrInvalid))
	}
	rec, err := t.lookupLocked(opID, token)
	if err != nil {
		t.mu.Unlock()
		return TxnReleaseAck{}, err
	}
	if !rec.settled {
		t.mu.Unlock()
		return TxnReleaseAck{}, observerErr(ObserverLayerDriver, ObserverCodeConnBusy, fmt.Errorf("%w: settle before release", ErrDenied))
	}
	t.mu.Unlock()
	if err := t.o.ReleaseConn(opID, token, self); err != nil {
		return TxnReleaseAck{}, err
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	rec.released = true
	return TxnReleaseAck{
		OperationID: rec.opID, Actor: rec.actor, Released: true,
		AckDigest: txnDigest("txn-release", rec.actor, rec.opID),
	}, nil
}

// ReleaseAck re-reads the release acknowledgment for one released actor.
func (t *TxnObserver) ReleaseAck(opID string) (TxnReleaseAck, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	rec, ok := t.actors[opID]
	if !ok {
		return TxnReleaseAck{}, observerErr(ObserverLayerDriver, ObserverCodeUnknownConn, fmt.Errorf("%w: operation %q", ErrNotFound, opID))
	}
	if !rec.released {
		return TxnReleaseAck{}, observerErr(ObserverLayerDriver, ObserverCodeNoConversation, fmt.Errorf("%w: no release recorded", ErrInvalid))
	}
	return TxnReleaseAck{
		OperationID: rec.opID, Actor: rec.actor, Released: true,
		AckDigest: txnDigest("txn-release", rec.actor, rec.opID),
	}, nil
}

// Close retires the transaction observer. Actors still held must be released
// first via Release so no charge is stranded; Close itself settles nothing.
func (t *TxnObserver) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return observerErr(ObserverLayerEngine, ObserverCodeClosedHandle, fmt.Errorf("%w: transaction observer already closed", ErrInvalid))
	}
	for _, rec := range t.actors {
		if !rec.released {
			return observerErr(ObserverLayerEngine, ObserverCodeClosedHandle, fmt.Errorf("%w: actor %q still entered", ErrDependency, rec.opID))
		}
	}
	t.closed = true
	return nil
}
