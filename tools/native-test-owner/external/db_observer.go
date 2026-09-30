// Independent raw DB observer: P29-side namespace/connection discipline.
//
// This file implements task K22's Go mirror: an independent raw database
// observer over in-memory doubles only. No real database is contacted, no
// driver is loaded, and no SQL is executed. The observer plugs into this
// package's vocabulary only: namespace grants (KindNamespace), opaque
// credential handles (KindCredential) as pinned raw connections, journaled
// admission, digest-only facts, and explicit release.
//
// Independence scope from the Can adapter under test (normative, mirrored
// in tools/runtime/test-services/db-observer/core.ts): the observer shares
// no adapter code, decoded values, or connection state; it proves namespace
// receipt, pinned connection, exact cells, row order/count, and layered
// error provenance from its own seeded doubles alone. Adapter claims are
// compared against these facts; no fact is ever derived from an adapter
// claim.
//
// Split with the TS core: the TS core owns typed exact cells (number lexeme,
// text code units, bytes, null-vs-empty), row order, and read-out facts plus
// the bounded self-check. This file owns P29-journaled namespace admission,
// the pinned identity-bound connection, the single-conversation discipline,
// and namespace receipts. Both enforce the same contract: owned namespace,
// pinned connection, one conversation per connection, and every failure
// naming its layer (engine/driver/sql).
//
// A pinned raw connection is a P29 credential leg issued under the observed
// namespace grant: the handle is the opaque credential token, facts carry
// only its digest, and the connection is a cleanup dependent of the
// namespace, so release drains dependents first via CleanupOrder.

package external

import (
	"fmt"
	"sync"

	"github.com/veighnsche/can-lang/tools/native-test-owner/journal"
)

// ObserverLayer names the failing layer of one observer error. Every
// observer failure names its layer; nothing unclassified is returned.
type ObserverLayer string

// Observer layers. Engine owns namespace admission, capacity, ownership and
// lifecycle; driver owns connection pinning, tokens and conversation
// discipline; SQL owns table names and read shape.
const (
	ObserverLayerEngine ObserverLayer = "engine"
	ObserverLayerDriver ObserverLayer = "driver"
	ObserverLayerSQL    ObserverLayer = "sql"
)

// Observer codes. Each code maps to exactly one layer, mirroring the TS
// core's closed code vocabulary.
const (
	ObserverCodeUnknownNamespace = "unknown-namespace"   // engine
	ObserverCodeWrongOwner       = "wrong-owner"         // engine
	ObserverCodeCapacity         = "capacity-exhausted"  // engine
	ObserverCodeClosedHandle     = "closed-handle"       // engine
	ObserverCodeUnknownConn      = "unknown-connection"  // driver
	ObserverCodeForgedToken      = "forged-token"        // driver
	ObserverCodeConnBusy         = "connection-busy"     // driver
	ObserverCodeNoConversation   = "no-conversation"     // driver
	ObserverCodeUnknownTable     = "unknown-table"       // sql
	ObserverCodeBadStatement     = "malformed-statement" // sql
)

// Observer bounds. Every observer table is finite; nothing here means
// unlimited.
const (
	// MaxObserverConnections caps pinned raw connections per observer.
	MaxObserverConnections = 16
	// MaxObserverNameLen caps one observer connection or table name.
	MaxObserverNameLen = 128
)

// ObserverError is one layered observer failure. Layer always names
// engine, driver, or SQL; Code is the closed failure code; Err carries the
// wrapped registry or journal cause for errors.Is matching.
type ObserverError struct {
	Layer ObserverLayer
	Code  string
	Err   error
}

func (e *ObserverError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s:%s] %v", e.Layer, e.Code, e.Err)
	}
	return fmt.Sprintf("[%s:%s] observer refused", e.Layer, e.Code)
}

// Unwrap exposes the wrapped registry cause.
func (e *ObserverError) Unwrap() error { return e.Err }

func observerErr(layer ObserverLayer, code string, err error) *ObserverError {
	return &ObserverError{Layer: layer, Code: code, Err: err}
}

// ObserverConnection is the observable record of one pinned raw connection.
// The token itself is never part of any fact; TokenDigest carries the
// digest only, matching P29 credential facts.
type ObserverConnection struct {
	OperationID      string
	Name             string
	Namespace        string
	TokenDigest      string
	ConversationOpen bool
	Released         bool
}

// ObserverReceipt is the namespace receipt: the observed namespace, its
// owner, the digest of the namespace handle, and the pinned count.
type ObserverReceipt struct {
	Namespace    string
	Owner        journal.Owner
	HandleDigest string
	Pinned       int
}

type observerPin struct {
	opID        string
	name        string
	token       string
	tokenDigest string
	active      bool   // a read conversation is open
	table       string // table of the open conversation
	released    bool
}

// Observer watches one owned P29 database namespace through its live
// namespace grant. An Observer is safe for concurrent use; the registry it
// observes must outlive it.
type Observer struct {
	mu      sync.Mutex
	r       *Registry
	nsToken string
	nsOp    string
	ns      string
	owner   journal.Owner
	pins    map[string]*observerPin
	closed  bool
}

func validObserverName(s string) bool {
	if len(s) == 0 || len(s) > MaxObserverNameLen {
		return false
	}
	return validID(s)
}

// ObserveNamespace binds an observer to one live namespace grant. The token
// must be this registry's own live grant for a namespace leg: invented
// tokens, expired or revoked grants, and grants for other legs are
// rejected. The observer records no raw secret: the receipt carries the
// namespace handle digest only.
func ObserveNamespace(r *Registry, nsToken string) (*Observer, error) {
	if r == nil {
		return nil, observerErr(ObserverLayerEngine, ObserverCodeUnknownNamespace, fmt.Errorf("%w: nil registry", ErrInvalid))
	}
	r.mu.Lock()
	g, err := r.lookupGrantLocked(nsToken)
	var cp Grant
	if err == nil {
		if g.Kind != KindNamespace {
			err = fmt.Errorf("%w: observer needs a namespace grant", ErrDenied)
		} else {
			cp = g.copy()
		}
	}
	r.mu.Unlock()
	if err != nil {
		return nil, observerErr(ObserverLayerEngine, ObserverCodeUnknownNamespace, err)
	}
	return &Observer{
		r:       r,
		nsToken: cp.Token,
		nsOp:    cp.OperationID,
		ns:      cp.Target,
		owner:   cp.Owner,
		pins:    make(map[string]*observerPin),
	}, nil
}

// liveLocked re-verifies the namespace grant under the registry lock.
func (o *Observer) liveLocked() error {
	g, err := o.r.lookupGrantLocked(o.nsToken)
	if err != nil {
		return err
	}
	if g.Kind != KindNamespace || g.OperationID != o.nsOp {
		return fmt.Errorf("%w: namespace grant changed legs", ErrDenied)
	}
	return nil
}

// Receipt reports the namespace receipt: namespace, owner, handle digest,
// and pinned count. The raw token never crosses.
func (o *Observer) Receipt() (ObserverReceipt, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return ObserverReceipt{}, observerErr(ObserverLayerEngine, ObserverCodeClosedHandle, fmt.Errorf("%w: observer closed", ErrInvalid))
	}
	o.r.mu.Lock()
	defer o.r.mu.Unlock()
	if err := o.liveLocked(); err != nil {
		return ObserverReceipt{}, observerErr(ObserverLayerEngine, ObserverCodeUnknownNamespace, err)
	}
	pinned := 0
	for _, p := range o.pins {
		if !p.released {
			pinned++
		}
	}
	return ObserverReceipt{
		Namespace:    o.ns,
		Owner:        o.owner,
		HandleDigest: handleDigest(o.nsToken),
		Pinned:       pinned,
	}, nil
}

// PinConnection pins one raw connection to the observed namespace. The pin
// is a P29 credential leg issued under the namespace grant: journaled,
// opaque, identity-bound, and a cleanup dependent of the namespace. The
// returned fact carries the token digest only; the raw token is returned
// alongside exactly once for the caller's later reads.
func (o *Observer) PinConnection(opID, name string, ttlMs int64) (ObserverConnection, string, error) {
	if !validID(opID) {
		return ObserverConnection{}, "", observerErr(ObserverLayerDriver, ObserverCodeUnknownConn, fmt.Errorf("%w: malformed operation ID", ErrInvalid))
	}
	if !validObserverName(name) {
		return ObserverConnection{}, "", observerErr(ObserverLayerSQL, ObserverCodeBadStatement, fmt.Errorf("%w: malformed connection name", ErrInvalid))
	}
	o.mu.Lock()
	if o.closed {
		o.mu.Unlock()
		return ObserverConnection{}, "", observerErr(ObserverLayerEngine, ObserverCodeClosedHandle, fmt.Errorf("%w: observer closed", ErrInvalid))
	}
	if len(o.pins) >= MaxObserverConnections {
		o.mu.Unlock()
		return ObserverConnection{}, "", observerErr(ObserverLayerEngine, ObserverCodeCapacity, fmt.Errorf("%w: observer connection table full", ErrCapacity))
	}
	if _, dup := o.pins[opID]; dup {
		o.mu.Unlock()
		return ObserverConnection{}, "", observerErr(ObserverLayerDriver, ObserverCodeUnknownConn, fmt.Errorf("%w: operation %q already pinned", ErrInvalid, opID))
	}
	o.mu.Unlock()
	// Issue outside the observer lock: the registry mints the opaque
	// handle and journals the credential leg.
	g, err := o.r.IssueCredential(opID, o.nsToken, ttlMs)
	if err != nil {
		return ObserverConnection{}, "", observerErr(ObserverLayerEngine, ObserverCodeUnknownNamespace, err)
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		// The leg stays journaled and releasable; the observer just
		// stops tracking it. Release it now so no charge is stranded
		// behind a closed observer.
		o.mu.Unlock()
		_ = o.r.Release(opID, g.Token, o.owner)
		o.mu.Lock()
		return ObserverConnection{}, "", observerErr(ObserverLayerEngine, ObserverCodeClosedHandle, fmt.Errorf("%w: observer closed", ErrInvalid))
	}
	o.pins[opID] = &observerPin{opID: opID, name: name, token: g.Token, tokenDigest: handleDigest(g.Token)}
	return ObserverConnection{
		OperationID: opID, Name: name, Namespace: o.ns,
		TokenDigest: handleDigest(g.Token),
	}, g.Token, nil
}

// lookupPinLocked verifies one pin and its token. Unknown pins and invented
// tokens are never authority.
func (o *Observer) lookupPinLocked(opID, token string) (*observerPin, error) {
	p, ok := o.pins[opID]
	if !ok || p.released {
		return nil, observerErr(ObserverLayerDriver, ObserverCodeUnknownConn, fmt.Errorf("%w: operation %q", ErrNotFound, opID))
	}
	if token == "" || token != p.token {
		return nil, observerErr(ObserverLayerDriver, ObserverCodeForgedToken, fmt.Errorf("%w: connection token is not the pinned token", ErrDenied))
	}
	return p, nil
}

// BeginRead opens one read conversation on a pinned connection.
// Single-conversation discipline: a second conversation on the same pin is
// rejected with a driver-layer failure until the first ends. The table name
// is recorded only; no query runs — cell read-out belongs to the TS core.
func (o *Observer) BeginRead(opID, token, table string) error {
	if !validObserverName(table) {
		return observerErr(ObserverLayerSQL, ObserverCodeUnknownTable, fmt.Errorf("%w: malformed table name", ErrInvalid))
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return observerErr(ObserverLayerEngine, ObserverCodeClosedHandle, fmt.Errorf("%w: observer closed", ErrInvalid))
	}
	p, err := o.lookupPinLocked(opID, token)
	if err != nil {
		return err
	}
	if p.active {
		return observerErr(ObserverLayerDriver, ObserverCodeConnBusy, fmt.Errorf("%w: one conversation is already open", ErrDenied))
	}
	p.active = true
	p.table = table
	return nil
}

// EndRead closes the open read conversation on a pinned connection.
func (o *Observer) EndRead(opID, token string) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return observerErr(ObserverLayerEngine, ObserverCodeClosedHandle, fmt.Errorf("%w: observer closed", ErrInvalid))
	}
	p, err := o.lookupPinLocked(opID, token)
	if err != nil {
		return err
	}
	if !p.active {
		return observerErr(ObserverLayerDriver, ObserverCodeNoConversation, fmt.Errorf("%w: no conversation is open", ErrInvalid))
	}
	p.active = false
	p.table = ""
	return nil
}

// Connection reports the observable record of one pinned connection.
func (o *Observer) Connection(opID string) (ObserverConnection, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	p, ok := o.pins[opID]
	if !ok {
		return ObserverConnection{}, observerErr(ObserverLayerDriver, ObserverCodeUnknownConn, fmt.Errorf("%w: operation %q", ErrNotFound, opID))
	}
	return ObserverConnection{
		OperationID: p.opID, Name: p.name, Namespace: o.ns,
		TokenDigest:      p.tokenDigest,
		ConversationOpen: p.active,
		Released:         p.released,
	}, nil
}

// ReleaseConn ends any exclusivity and releases one pinned connection under
// its own live token: the registry walks the journal to closed, revokes the
// grant, and settles the retained charge. A pin with an open conversation
// is rejected so interleaved reads can never straddle a release.
func (o *Observer) ReleaseConn(opID, token string, self journal.Owner) error {
	o.mu.Lock()
	if o.closed {
		o.mu.Unlock()
		return observerErr(ObserverLayerEngine, ObserverCodeClosedHandle, fmt.Errorf("%w: observer closed", ErrInvalid))
	}
	p, err := o.lookupPinLocked(opID, token)
	if err != nil {
		o.mu.Unlock()
		return err
	}
	if p.active {
		o.mu.Unlock()
		return observerErr(ObserverLayerDriver, ObserverCodeConnBusy, fmt.Errorf("%w: a conversation is still open", ErrDenied))
	}
	if self.PID != o.owner.PID || self.StartToken != o.owner.StartToken {
		o.mu.Unlock()
		return observerErr(ObserverLayerEngine, ObserverCodeWrongOwner, fmt.Errorf("%w: release presents a foreign identity", ErrWrongOwner))
	}
	o.mu.Unlock()
	if err := o.r.Release(opID, token, self); err != nil {
		return observerErr(ObserverLayerEngine, ObserverCodeClosedHandle, err)
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	p.released = true
	return nil
}

// Close retires the observer. Pins still held must be released first via
// ReleaseConn so no charge is stranded; Close itself settles nothing.
func (o *Observer) Close() error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return observerErr(ObserverLayerEngine, ObserverCodeClosedHandle, fmt.Errorf("%w: observer already closed", ErrInvalid))
	}
	for _, p := range o.pins {
		if !p.released {
			return observerErr(ObserverLayerEngine, ObserverCodeClosedHandle, fmt.Errorf("%w: connection %q still pinned", ErrDependency, p.opID))
		}
	}
	o.closed = true
	return nil
}
