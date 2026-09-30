// Independent N remote-context cleanup authority: K16's Go mirror.
//
// This file implements the ACTUAL outside-service authority under
// independent N ownership, over in-memory doubles only. No browser is
// contacted, no process is spawned or sampled, no environment is read,
// and no network I/O is performed. The authority admits declared shared
// servers, declares owned connections and contexts separately from the
// server, notes worker death as a fact, requests closes, tracks
// confirmations, and seals closes with an opaque minted witness token
// verified by lookup. Worker death stops opens while close requests,
// confirmation tracking, and the confirmed seal proceed. The shared
// server is preserved: a case close-server attempt refuses before any
// effect and the server stays provably live and untouched. A lost close
// confirmation remains unresolved: the context stays explicitly
// pending/unknown and is never treated as closed.
//
// Independence scope from the launch/driver service under test
// (normative, mirrored in
// tools/runtime/test-services/browser-driver/remote.ts): the authority
// shares no driver code, decoded values, receipts, flags, environment,
// network state, or process state; it proves shared-server admission and
// preservation, connection/context ownership separate from the server,
// worker-death noting, close request/confirmation separation, externally
// confirmed seals, and layered error provenance from its own seeded
// doubles alone. Driver claims are compared against these facts; no
// cleanup fact is ever derived from a driver claim. Consequently a
// worker receipt can never prove cleanup: only an independent close
// confirmation over the exact context proves it.
//
// Split with the TS service: the TS service owns the typed remote gate,
// connection/context ownership, and the close-verifier seam plus the
// bounded self-check. This file owns the same contract as the
// outside-service authority: shared-server admission, worker-death
// noting, close minting, and confirmation. Both enforce declared
// servers, opaque tokens verified by lookup, digest-only facts, and
// layered errors. Neither contacts a browser.
//
// Pure in-memory mechanics: no I/O, timers, transports, services,
// browsers, processes, networks, or live runtimes. Local controls only.
// Live host-dependent controls wait for the corresponding qualified
// profile and Q task.

package external

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"sync"

	"github.com/veighnsche/can-lang/tools/native-test-owner/journal"
)

// BrowserRemoteLayer names the failing layer of one authority error.
// Every authority failure names its layer; nothing unclassified is
// returned.
type BrowserRemoteLayer string

// Authority layers. Service owns declared servers, ownership, and
// capacity; remote owns connection/context binding, tokens, lifecycle,
// worker death, and the server-close refusal; cleanup owns close
// admission, close tokens, confirmation, and the close seal.
const (
	BrowserRemoteLayerService BrowserRemoteLayer = "service"
	BrowserRemoteLayerRemote  BrowserRemoteLayer = "remote"
	BrowserRemoteLayerCleanup BrowserRemoteLayer = "cleanup"
)

// Closed failure-code vocabulary. Each code maps to exactly one layer,
// mirroring the TS service's vocabulary for this facet.
const (
	BrowserRemoteCodeUnknownServer        = "unknown-server"         // service
	BrowserRemoteCodeWrongOwner           = "wrong-owner"            // service
	BrowserRemoteCodeCapacity             = "capacity-exhausted"     // service
	BrowserRemoteCodeForgedToken          = "forged-token"           // remote
	BrowserRemoteCodeUnknownConnection    = "unknown-connection"     // remote
	BrowserRemoteCodeUnknownContext       = "unknown-context"        // remote
	BrowserRemoteCodeContextClosed        = "context-closed"         // remote
	BrowserRemoteCodeWorkerDead           = "worker-dead"            // remote
	BrowserRemoteCodeForbiddenServerClose = "forbidden-server-close" // remote
	BrowserRemoteCodeForbiddenClose       = "forbidden-close"        // cleanup
	BrowserRemoteCodeForgedClose          = "forged-close"           // cleanup
	BrowserRemoteCodeCloseUnconfirmed     = "close-unconfirmed"      // cleanup
	BrowserRemoteCodeOrphanOpen           = "orphan-open"            // cleanup
	BrowserRemoteCodeAlreadyClosed        = "already-closed"         // cleanup
)

// BrowserRemoteCodes lists the closed vocabulary in layer order.
var BrowserRemoteCodes = []string{
	BrowserRemoteCodeUnknownServer, BrowserRemoteCodeWrongOwner,
	BrowserRemoteCodeCapacity, BrowserRemoteCodeForgedToken,
	BrowserRemoteCodeUnknownConnection, BrowserRemoteCodeUnknownContext,
	BrowserRemoteCodeContextClosed, BrowserRemoteCodeWorkerDead,
	BrowserRemoteCodeForbiddenServerClose, BrowserRemoteCodeForbiddenClose,
	BrowserRemoteCodeForgedClose, BrowserRemoteCodeCloseUnconfirmed,
	BrowserRemoteCodeOrphanOpen, BrowserRemoteCodeAlreadyClosed,
}

// BrowserRemoteLayerOfCode maps one closed code to its single layer.
func BrowserRemoteLayerOfCode(code string) (BrowserRemoteLayer, bool) {
	switch code {
	case BrowserRemoteCodeUnknownServer, BrowserRemoteCodeWrongOwner,
		BrowserRemoteCodeCapacity:
		return BrowserRemoteLayerService, true
	case BrowserRemoteCodeForgedToken, BrowserRemoteCodeUnknownConnection,
		BrowserRemoteCodeUnknownContext, BrowserRemoteCodeContextClosed,
		BrowserRemoteCodeWorkerDead, BrowserRemoteCodeForbiddenServerClose:
		return BrowserRemoteLayerRemote, true
	case BrowserRemoteCodeForbiddenClose, BrowserRemoteCodeForgedClose,
		BrowserRemoteCodeCloseUnconfirmed, BrowserRemoteCodeOrphanOpen,
		BrowserRemoteCodeAlreadyClosed:
		return BrowserRemoteLayerCleanup, true
	default:
		return "", false
	}
}

// Authority bounds. Every authority table is finite; nothing here means
// unlimited.
const (
	// MaxRemoteServers caps declared shared servers per authority.
	MaxRemoteServers = 16
	// MaxRemoteConnections caps declared connections per authority.
	MaxRemoteConnections = 64
	// MaxRemoteContexts caps contexts known per connection.
	MaxRemoteContexts = 64
)

// BrowserRemoteError is one layered authority failure. Layer always names
// service, remote, or cleanup; Code is the closed failure code; Err
// carries the wrapped package cause for errors.Is matching.
type BrowserRemoteError struct {
	Layer BrowserRemoteLayer
	Code  string
	Err   error
}

func (e *BrowserRemoteError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s:%s] %v", e.Layer, e.Code, e.Err)
	}
	return fmt.Sprintf("[%s:%s] authority refused", e.Layer, e.Code)
}

// Unwrap exposes the wrapped package cause.
func (e *BrowserRemoteError) Unwrap() error { return e.Err }

func remoteErr(layer BrowserRemoteLayer, code string, err error) *BrowserRemoteError {
	return &BrowserRemoteError{Layer: layer, Code: code, Err: err}
}

func checkRemoteOwner(self journal.Owner) error {
	if self.PID <= 0 || !validID(self.StartToken) {
		return remoteErr(BrowserRemoteLayerService, BrowserRemoteCodeWrongOwner, fmt.Errorf("%w: owner needs a positive PID and a valid spawn-start token", ErrInvalid))
	}
	return nil
}

func checkRemoteServer(s string) error {
	if !validID(s) {
		return remoteErr(BrowserRemoteLayerService, BrowserRemoteCodeUnknownServer, fmt.Errorf("%w: malformed server name", ErrInvalid))
	}
	return nil
}

func checkRemoteConnectionID(s string) error {
	if !validID(s) {
		return remoteErr(BrowserRemoteLayerRemote, BrowserRemoteCodeUnknownConnection, fmt.Errorf("%w: malformed connection ID", ErrInvalid))
	}
	return nil
}

func checkRemoteContextID(s string) error {
	if !validID(s) {
		return remoteErr(BrowserRemoteLayerRemote, BrowserRemoteCodeUnknownContext, fmt.Errorf("%w: malformed context ID", ErrInvalid))
	}
	return nil
}

// ---------------------------------------------------------------------------
// Facts (digest-only; returned by value)
// ---------------------------------------------------------------------------

// RemoteCloseState is the close lifecycle of one owned context: open,
// close-pending, closed, or unknown. Unknown means the confirmation was
// lost; the close is unresolved and the context is never closed.
type RemoteCloseState string

// Close lifecycle states.
const (
	RemoteCloseOpen    RemoteCloseState = "open"
	RemoteClosePending RemoteCloseState = "close-pending"
	RemoteCloseClosed  RemoteCloseState = "closed"
	RemoteCloseUnknown RemoteCloseState = "unknown"
)

// RemoteServerFacts names one declared shared server: always live, with
// the sorted owned connection IDs. Raw tokens never cross.
type RemoteServerFacts struct {
	Server      string
	Owner       journal.Owner
	Live        bool
	Connections []string
	Digest      string
}

// RemoteConnectionFacts names one declared connection: its server, the
// worker-death fact, the sorted context IDs, and the token digest. The
// raw token never crosses.
type RemoteConnectionFacts struct {
	ConnectionID string
	Server       string
	WorkerDead   bool
	Contexts     []string
	TokenDigest  string
}

// RemoteContextFacts names one owned context: its connection and server,
// the close state, the closed bit, the worker-death fact, and the token
// digest. The raw token never crosses.
type RemoteContextFacts struct {
	ContextID    string
	ConnectionID string
	Server       string
	CloseState   RemoteCloseState
	Closed       bool
	WorkerDead   bool
	TokenDigest  string
}

// RemoteCloseRequestFacts names one pending close request. A request is
// never itself a close.
type RemoteCloseRequestFacts struct {
	ContextID    string
	ConnectionID string
	CloseState   RemoteCloseState
	Closed       bool
	TokenDigest  string
}

// RemoteWitnessFacts names one confirmed close: the context, closure,
// and the witness token digest.
type RemoteWitnessFacts struct {
	ContextID    string
	ConnectionID string
	Closed       bool
	TokenDigest  string
}

// RemoteCloseReceipt seals one context through independent confirmation.
type RemoteCloseReceipt struct {
	ContextID    string
	ConnectionID string
	Owner        journal.Owner
	Server       string
	WorkerDead   bool
	Digest       string
}

// RemoteCleanupClaim is one cleanup proof claim. Only Kind
// "external-close" with a verifying receipt judges; every other kind
// refuses with forbidden-close before any verdict is read.
type RemoteCleanupClaim struct {
	Kind    string
	Receipt RemoteCloseReceipt
	Detail  string
}

// RemoteCleanupVerdict judges one verified independent close.
type RemoteCleanupVerdict struct {
	ContextID    string
	ConnectionID string
	Witnessed    bool
	Digest       string
}

// ForbiddenCloseKinds are the inadmissible cleanup proof kinds.
var ForbiddenCloseKinds = []string{"worker-receipt", "server-log", "exit-code"}

// DigestRemoteClose recomputes the close digest from carried fields, so
// AssertRemoteCleanup re-verifies a presented receipt instead of trusting
// its digest string.
func DigestRemoteClose(contextID, connectionID string, owner journal.Owner, server string, workerDead bool) string {
	h := sha256.New()
	h.Write([]byte(contextID))
	h.Write([]byte{0})
	h.Write([]byte(connectionID))
	h.Write([]byte{0})
	h.Write([]byte(fmt.Sprintf("%d\x00%s", owner.PID, owner.StartToken)))
	h.Write([]byte{0})
	h.Write([]byte(server))
	h.Write([]byte{0})
	if workerDead {
		h.Write([]byte("dead"))
	} else {
		h.Write([]byte("live"))
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

func digestRemoteText(text string) string {
	sum := sha256.Sum256([]byte(text))
	return "sha256:" + hex.EncodeToString(sum[:])
}

type remoteContext struct {
	contextID    string
	connectionID string
	owner        journal.Owner
	server       string
	token        string
	closeState   RemoteCloseState
	closeToken   string // minted close token; "" until requested
	witness      string // minted witness token; "" until sealed
	receipt      RemoteCloseReceipt
}

type remoteConnection struct {
	connectionID string
	owner        journal.Owner
	server       string
	token        string
	workerDead   bool
	contexts     map[string]*remoteContext
}

// RemoteServerGrant declares one shared server for one keeper owner.
// Cases connect to declared servers; they never close them.
type RemoteServerGrant struct {
	Owner  journal.Owner
	Server string
}

// RemoteAuthority is the independent N remote-context cleanup authority.
// It never reads the driver or the network: admission, worker-death
// noting, close minting, and confirmation come from its own table alone.
// A RemoteAuthority is safe for concurrent use.
type RemoteAuthority struct {
	mu          sync.Mutex
	servers     map[string]RemoteServerGrant
	connections map[string]*remoteConnection
}

// NewRemoteAuthority binds an authority to its finite declared shared
// servers. Anything outside them is foreign.
func NewRemoteAuthority(servers []RemoteServerGrant) (*RemoteAuthority, error) {
	if len(servers) == 0 {
		return nil, remoteErr(BrowserRemoteLayerService, BrowserRemoteCodeUnknownServer, fmt.Errorf("%w: declare at least one shared server", ErrInvalid))
	}
	if len(servers) > MaxRemoteServers {
		return nil, remoteErr(BrowserRemoteLayerService, BrowserRemoteCodeCapacity, fmt.Errorf("%w: declared servers exceed the cap", ErrCapacity))
	}
	stable := make(map[string]RemoteServerGrant, len(servers))
	for _, g := range servers {
		if err := checkRemoteOwner(g.Owner); err != nil {
			return nil, err
		}
		if err := checkRemoteServer(g.Server); err != nil {
			return nil, err
		}
		if _, dup := stable[g.Server]; dup {
			return nil, remoteErr(BrowserRemoteLayerService, BrowserRemoteCodeUnknownServer, fmt.Errorf("%w: duplicate shared server", ErrInvalid))
		}
		stable[g.Server] = g
	}
	return &RemoteAuthority{
		servers:     stable,
		connections: make(map[string]*remoteConnection),
	}, nil
}

func sameRemoteOwner(a, b journal.Owner) bool {
	return a.PID == b.PID && a.StartToken == b.StartToken
}

// ServerFacts reads one declared shared server. Servers stay live
// forever: nothing in this authority closes, kills, or otherwise
// touches them.
func (a *RemoteAuthority) ServerFacts(server string) (RemoteServerFacts, error) {
	if err := checkRemoteServer(server); err != nil {
		return RemoteServerFacts{}, err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	grant, ok := a.servers[server]
	if !ok {
		return RemoteServerFacts{}, remoteErr(BrowserRemoteLayerService, BrowserRemoteCodeUnknownServer, fmt.Errorf("%w: server %q", ErrNotFound, server))
	}
	var conns []string
	for id, rec := range a.connections {
		if rec.server == server {
			conns = append(conns, id)
		}
	}
	sort.Strings(conns)
	return RemoteServerFacts{
		Server: grant.Server, Owner: grant.Owner, Live: true,
		Connections: conns,
		Digest:      digestRemoteText("server:" + server + ":" + fmt.Sprintf("%v", conns)),
	}, nil
}

// CloseServer is the case close-server attempt. It always refuses with
// forbidden-server-close before any effect: the refusal precedes every
// mutation, and the server stays provably live and untouched.
func (a *RemoteAuthority) CloseServer(self journal.Owner, server string) error {
	if err := checkRemoteOwner(self); err != nil {
		return err
	}
	if err := checkRemoteServer(server); err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, ok := a.servers[server]; !ok {
		return remoteErr(BrowserRemoteLayerService, BrowserRemoteCodeUnknownServer, fmt.Errorf("%w: server %q", ErrNotFound, server))
	}
	return remoteErr(BrowserRemoteLayerRemote, BrowserRemoteCodeForbiddenServerClose, fmt.Errorf("%w: cases never close the shared server", ErrDenied))
}

// DeclareConnection declares one owned connection to a declared shared
// server. Admission-first: the server is checked before anything else.
// The connection token is minted here and verified by table lookup on
// every later call; the returned facts carry its digest only.
// Re-declaring the identical connection joins; re-declaring the ID on
// another server is refused.
func (a *RemoteAuthority) DeclareConnection(self journal.Owner, server, connectionID string) (RemoteConnectionFacts, string, error) {
	if err := checkRemoteOwner(self); err != nil {
		return RemoteConnectionFacts{}, "", err
	}
	if err := checkRemoteServer(server); err != nil {
		return RemoteConnectionFacts{}, "", err
	}
	if err := checkRemoteConnectionID(connectionID); err != nil {
		return RemoteConnectionFacts{}, "", err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, ok := a.servers[server]; !ok {
		return RemoteConnectionFacts{}, "", remoteErr(BrowserRemoteLayerService, BrowserRemoteCodeUnknownServer, fmt.Errorf("%w: server %q is not a declared shared server", ErrDenied, server))
	}
	if prior, dup := a.connections[connectionID]; dup {
		if !sameRemoteOwner(prior.owner, self) {
			return RemoteConnectionFacts{}, "", remoteErr(BrowserRemoteLayerService, BrowserRemoteCodeWrongOwner, fmt.Errorf("%w: connection is owned by another identity", ErrWrongOwner))
		}
		if prior.server != server {
			return RemoteConnectionFacts{}, "", remoteErr(BrowserRemoteLayerRemote, BrowserRemoteCodeUnknownConnection, fmt.Errorf("%w: connection ID is already declared", ErrInvalid))
		}
		return a.connectionFactsLocked(prior), prior.token, nil
	}
	if len(a.connections) >= MaxRemoteConnections {
		return RemoteConnectionFacts{}, "", remoteErr(BrowserRemoteLayerService, BrowserRemoteCodeCapacity, fmt.Errorf("%w: connection table full", ErrCapacity))
	}
	tok, err := mintToken()
	if err != nil {
		return RemoteConnectionFacts{}, "", remoteErr(BrowserRemoteLayerService, BrowserRemoteCodeCapacity, err)
	}
	rec := &remoteConnection{
		connectionID: connectionID, owner: self, server: server,
		token: tok, contexts: make(map[string]*remoteContext),
	}
	a.connections[connectionID] = rec
	return a.connectionFactsLocked(rec), tok, nil
}

func (a *RemoteAuthority) connectionFactsLocked(rec *remoteConnection) RemoteConnectionFacts {
	var ctxs []string
	for id := range rec.contexts {
		ctxs = append(ctxs, id)
	}
	sort.Strings(ctxs)
	return RemoteConnectionFacts{
		ConnectionID: rec.connectionID, Server: rec.server,
		WorkerDead: rec.workerDead, Contexts: ctxs, TokenDigest: handleDigest(rec.token),
	}
}

func (a *RemoteAuthority) contextFactsLocked(rec *remoteConnection, ctx *remoteContext) RemoteContextFacts {
	return RemoteContextFacts{
		ContextID: ctx.contextID, ConnectionID: rec.connectionID, Server: rec.server,
		CloseState: ctx.closeState, Closed: ctx.closeState == RemoteCloseClosed,
		WorkerDead: rec.workerDead, TokenDigest: handleDigest(ctx.token),
	}
}

// liveConnectionLocked verifies one connection and its token by table
// lookup. Unknown connections reject before the token is even read;
// invented tokens are never authority. Callers hold the lock.
func (a *RemoteAuthority) liveConnectionLocked(self journal.Owner, connectionID, token string) (*remoteConnection, error) {
	rec, ok := a.connections[connectionID]
	if !ok {
		return nil, remoteErr(BrowserRemoteLayerRemote, BrowserRemoteCodeUnknownConnection, fmt.Errorf("%w: connection %q", ErrNotFound, connectionID))
	}
	if !sameRemoteOwner(rec.owner, self) {
		return nil, remoteErr(BrowserRemoteLayerService, BrowserRemoteCodeWrongOwner, fmt.Errorf("%w: connection is owned by another identity", ErrWrongOwner))
	}
	if token == "" || token != rec.token {
		return nil, remoteErr(BrowserRemoteLayerRemote, BrowserRemoteCodeForgedToken, fmt.Errorf("%w: connection token is not the declared token", ErrDenied))
	}
	return rec, nil
}

// liveContextLocked verifies one context and its token by table lookup.
// Unknown contexts reject before the token is even read; invented
// tokens are never authority. Callers hold the lock.
func (a *RemoteAuthority) liveContextLocked(self journal.Owner, connectionID, contextID, token string) (*remoteConnection, *remoteContext, error) {
	rec, ok := a.connections[connectionID]
	if !ok {
		return nil, nil, remoteErr(BrowserRemoteLayerRemote, BrowserRemoteCodeUnknownConnection, fmt.Errorf("%w: connection %q", ErrNotFound, connectionID))
	}
	if !sameRemoteOwner(rec.owner, self) {
		return nil, nil, remoteErr(BrowserRemoteLayerService, BrowserRemoteCodeWrongOwner, fmt.Errorf("%w: connection is owned by another identity", ErrWrongOwner))
	}
	ctx, ok := rec.contexts[contextID]
	if !ok {
		return nil, nil, remoteErr(BrowserRemoteLayerRemote, BrowserRemoteCodeUnknownContext, fmt.Errorf("%w: context %q", ErrNotFound, contextID))
	}
	if token == "" || token != ctx.token {
		return nil, nil, remoteErr(BrowserRemoteLayerRemote, BrowserRemoteCodeForgedToken, fmt.Errorf("%w: context token is not the declared token", ErrDenied))
	}
	return rec, ctx, nil
}

// OpenContext opens one owned context under a live connection. A dead
// worker opens nothing further; cleanup of already-open contexts
// proceeds independently. Reopening a live context joins; a closed ID
// never reopens.
func (a *RemoteAuthority) OpenContext(self journal.Owner, connectionID, connToken, contextID string) (RemoteContextFacts, string, error) {
	if err := checkRemoteOwner(self); err != nil {
		return RemoteContextFacts{}, "", err
	}
	if err := checkRemoteConnectionID(connectionID); err != nil {
		return RemoteContextFacts{}, "", err
	}
	if err := checkRemoteContextID(contextID); err != nil {
		return RemoteContextFacts{}, "", err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	rec, err := a.liveConnectionLocked(self, connectionID, connToken)
	if err != nil {
		return RemoteContextFacts{}, "", err
	}
	if rec.workerDead {
		return RemoteContextFacts{}, "", remoteErr(BrowserRemoteLayerRemote, BrowserRemoteCodeWorkerDead, fmt.Errorf("%w: worker is dead for connection", ErrDenied))
	}
	if prior, dup := rec.contexts[contextID]; dup {
		if prior.closeState == RemoteCloseClosed {
			return RemoteContextFacts{}, "", remoteErr(BrowserRemoteLayerRemote, BrowserRemoteCodeContextClosed, fmt.Errorf("%w: context ID is single-use and already closed", ErrInvalid))
		}
		return a.contextFactsLocked(rec, prior), prior.token, nil
	}
	if len(rec.contexts) >= MaxRemoteContexts {
		return RemoteContextFacts{}, "", remoteErr(BrowserRemoteLayerService, BrowserRemoteCodeCapacity, fmt.Errorf("%w: context table full for connection", ErrCapacity))
	}
	tok, err := mintToken()
	if err != nil {
		return RemoteContextFacts{}, "", remoteErr(BrowserRemoteLayerService, BrowserRemoteCodeCapacity, err)
	}
	ctx := &remoteContext{
		contextID: contextID, connectionID: connectionID,
		owner: self, server: rec.server, token: tok, closeState: RemoteCloseOpen,
	}
	rec.contexts[contextID] = ctx
	return a.contextFactsLocked(rec, ctx), tok, nil
}

// NoteWorkerDeath notes the worker's death. Opens stop; close requests,
// confirmation tracking, and the confirmed seal proceed.
func (a *RemoteAuthority) NoteWorkerDeath(self journal.Owner, connectionID, connToken string) error {
	if err := checkRemoteOwner(self); err != nil {
		return err
	}
	if err := checkRemoteConnectionID(connectionID); err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	rec, err := a.liveConnectionLocked(self, connectionID, connToken)
	if err != nil {
		return err
	}
	rec.workerDead = true
	return nil
}

// ConnectionFacts reads the externally known connection facts. Facts
// stay readable after death, after a lost confirmation, and after the
// seal: the connection is evidence.
func (a *RemoteAuthority) ConnectionFacts(self journal.Owner, connectionID string) (RemoteConnectionFacts, error) {
	if err := checkRemoteOwner(self); err != nil {
		return RemoteConnectionFacts{}, err
	}
	if err := checkRemoteConnectionID(connectionID); err != nil {
		return RemoteConnectionFacts{}, err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	rec, ok := a.connections[connectionID]
	if !ok {
		return RemoteConnectionFacts{}, remoteErr(BrowserRemoteLayerRemote, BrowserRemoteCodeUnknownConnection, fmt.Errorf("%w: connection %q", ErrNotFound, connectionID))
	}
	if !sameRemoteOwner(rec.owner, self) {
		return RemoteConnectionFacts{}, remoteErr(BrowserRemoteLayerService, BrowserRemoteCodeWrongOwner, fmt.Errorf("%w: connection is owned by another identity", ErrWrongOwner))
	}
	return a.connectionFactsLocked(rec), nil
}

// ContextFacts reads the externally known context facts, including the
// explicit close state. A lost confirmation reads unknown, never closed.
func (a *RemoteAuthority) ContextFacts(self journal.Owner, connectionID, contextID string) (RemoteContextFacts, error) {
	if err := checkRemoteOwner(self); err != nil {
		return RemoteContextFacts{}, err
	}
	if err := checkRemoteConnectionID(connectionID); err != nil {
		return RemoteContextFacts{}, err
	}
	if err := checkRemoteContextID(contextID); err != nil {
		return RemoteContextFacts{}, err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	rec, ok := a.connections[connectionID]
	if !ok {
		return RemoteContextFacts{}, remoteErr(BrowserRemoteLayerRemote, BrowserRemoteCodeUnknownConnection, fmt.Errorf("%w: connection %q", ErrNotFound, connectionID))
	}
	if !sameRemoteOwner(rec.owner, self) {
		return RemoteContextFacts{}, remoteErr(BrowserRemoteLayerService, BrowserRemoteCodeWrongOwner, fmt.Errorf("%w: connection is owned by another identity", ErrWrongOwner))
	}
	ctx, ok := rec.contexts[contextID]
	if !ok {
		return RemoteContextFacts{}, remoteErr(BrowserRemoteLayerRemote, BrowserRemoteCodeUnknownContext, fmt.Errorf("%w: context %q", ErrNotFound, contextID))
	}
	return a.contextFactsLocked(rec, ctx), nil
}

// RequestClose requests the close of one open context. The request moves
// the context to close-pending and mints the close token; only an
// independent confirmation seals it. A request is never itself a close.
// The close token is minted here and verified by table lookup in
// ConfirmClose; the returned facts carry its digest only and the raw
// token crosses alongside exactly once.
func (a *RemoteAuthority) RequestClose(self journal.Owner, connectionID, contextID, ctxToken string) (RemoteCloseRequestFacts, string, error) {
	if err := checkRemoteOwner(self); err != nil {
		return RemoteCloseRequestFacts{}, "", err
	}
	if err := checkRemoteConnectionID(connectionID); err != nil {
		return RemoteCloseRequestFacts{}, "", err
	}
	if err := checkRemoteContextID(contextID); err != nil {
		return RemoteCloseRequestFacts{}, "", err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	_, ctx, err := a.liveContextLocked(self, connectionID, contextID, ctxToken)
	if err != nil {
		return RemoteCloseRequestFacts{}, "", err
	}
	if ctx.closeState == RemoteCloseClosed {
		return RemoteCloseRequestFacts{}, "", remoteErr(BrowserRemoteLayerCleanup, BrowserRemoteCodeAlreadyClosed, fmt.Errorf("%w: context is already closed", ErrInvalid))
	}
	if ctx.closeState == RemoteCloseUnknown {
		return RemoteCloseRequestFacts{}, "", remoteErr(BrowserRemoteLayerCleanup, BrowserRemoteCodeCloseUnconfirmed, fmt.Errorf("%w: close confirmation is lost for context", ErrDenied))
	}
	if ctx.closeState == RemoteClosePending {
		return RemoteCloseRequestFacts{
			ContextID: ctx.contextID, ConnectionID: ctx.connectionID,
			CloseState: RemoteClosePending, Closed: false, TokenDigest: handleDigest(ctx.token),
		}, ctx.closeToken, nil
	}
	ctok, err := mintToken()
	if err != nil {
		return RemoteCloseRequestFacts{}, "", remoteErr(BrowserRemoteLayerService, BrowserRemoteCodeCapacity, err)
	}
	ctx.closeToken = ctok
	ctx.closeState = RemoteClosePending
	return RemoteCloseRequestFacts{
		ContextID: ctx.contextID, ConnectionID: ctx.connectionID,
		CloseState: RemoteClosePending, Closed: false, TokenDigest: handleDigest(ctx.token),
	}, ctok, nil
}

// LoseConfirmation loses the close confirmation permanently: the close
// stays explicitly unresolved. Later confirmations and receipts refuse
// with close-unconfirmed, and the context is never treated as closed.
func (a *RemoteAuthority) LoseConfirmation(self journal.Owner, connectionID, contextID, ctxToken string) error {
	if err := checkRemoteOwner(self); err != nil {
		return err
	}
	if err := checkRemoteConnectionID(connectionID); err != nil {
		return err
	}
	if err := checkRemoteContextID(contextID); err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	_, ctx, err := a.liveContextLocked(self, connectionID, contextID, ctxToken)
	if err != nil {
		return err
	}
	if ctx.closeState == RemoteCloseClosed {
		return remoteErr(BrowserRemoteLayerCleanup, BrowserRemoteCodeAlreadyClosed, fmt.Errorf("%w: context is already closed", ErrInvalid))
	}
	if ctx.closeState != RemoteClosePending {
		return remoteErr(BrowserRemoteLayerCleanup, BrowserRemoteCodeCloseUnconfirmed, fmt.Errorf("%w: no close confirmation is awaited for context", ErrDenied))
	}
	ctx.closeState = RemoteCloseUnknown
	ctx.closeToken = ""
	return nil
}

// ConfirmClose seals the close under independent ownership. The close
// token is verified by table lookup; the seal needs a pending request
// and a lost confirmation never seals. The witness token is minted here
// and verified by table lookup in VerifyWitness; the returned facts
// carry its digest only and the raw token crosses alongside exactly
// once.
func (a *RemoteAuthority) ConfirmClose(self journal.Owner, connectionID, contextID, ctxToken, closeToken string) (RemoteWitnessFacts, string, error) {
	if err := checkRemoteOwner(self); err != nil {
		return RemoteWitnessFacts{}, "", err
	}
	if err := checkRemoteConnectionID(connectionID); err != nil {
		return RemoteWitnessFacts{}, "", err
	}
	if err := checkRemoteContextID(contextID); err != nil {
		return RemoteWitnessFacts{}, "", err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	rec, ctx, err := a.liveContextLocked(self, connectionID, contextID, ctxToken)
	if err != nil {
		return RemoteWitnessFacts{}, "", err
	}
	if ctx.closeState == RemoteCloseClosed {
		return RemoteWitnessFacts{}, "", remoteErr(BrowserRemoteLayerCleanup, BrowserRemoteCodeAlreadyClosed, fmt.Errorf("%w: context is already closed", ErrInvalid))
	}
	if ctx.closeState == RemoteCloseUnknown {
		return RemoteWitnessFacts{}, "", remoteErr(BrowserRemoteLayerCleanup, BrowserRemoteCodeCloseUnconfirmed, fmt.Errorf("%w: close confirmation is lost for context", ErrDenied))
	}
	if ctx.closeState != RemoteClosePending {
		return RemoteWitnessFacts{}, "", remoteErr(BrowserRemoteLayerCleanup, BrowserRemoteCodeCloseUnconfirmed, fmt.Errorf("%w: no close was requested for context", ErrDenied))
	}
	if closeToken == "" || closeToken != ctx.closeToken {
		return RemoteWitnessFacts{}, "", remoteErr(BrowserRemoteLayerCleanup, BrowserRemoteCodeForgedClose, fmt.Errorf("%w: close token is not the requested token", ErrDenied))
	}
	wtok, err := mintToken()
	if err != nil {
		return RemoteWitnessFacts{}, "", remoteErr(BrowserRemoteLayerService, BrowserRemoteCodeCapacity, err)
	}
	ctx.witness = wtok
	ctx.closeState = RemoteCloseClosed
	ctx.receipt = RemoteCloseReceipt{
		ContextID: ctx.contextID, ConnectionID: rec.connectionID, Owner: ctx.owner,
		Server: ctx.server, WorkerDead: rec.workerDead,
		Digest: DigestRemoteClose(ctx.contextID, rec.connectionID, ctx.owner, ctx.server, rec.workerDead),
	}
	return RemoteWitnessFacts{
		ContextID: ctx.contextID, ConnectionID: rec.connectionID,
		Closed: true, TokenDigest: handleDigest(wtok),
	}, wtok, nil
}

// VerifyWitness verifies one witness token by table lookup. Unknown
// tokens and cross-context tokens are never authority.
func (a *RemoteAuthority) VerifyWitness(connectionID, contextID, token string) (RemoteWitnessFacts, error) {
	if err := checkRemoteConnectionID(connectionID); err != nil {
		return RemoteWitnessFacts{}, err
	}
	if err := checkRemoteContextID(contextID); err != nil {
		return RemoteWitnessFacts{}, err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, rec := range a.connections {
		for id, ctx := range rec.contexts {
			if ctx.witness != "" && ctx.witness == token {
				if id != contextID || rec.connectionID != connectionID {
					return RemoteWitnessFacts{}, remoteErr(BrowserRemoteLayerCleanup, BrowserRemoteCodeForgedClose, fmt.Errorf("%w: witness is for another context", ErrDenied))
				}
				return RemoteWitnessFacts{
					ContextID: id, ConnectionID: rec.connectionID,
					Closed: true, TokenDigest: handleDigest(token),
				}, nil
			}
		}
	}
	return RemoteWitnessFacts{}, remoteErr(BrowserRemoteLayerCleanup, BrowserRemoteCodeForgedClose, fmt.Errorf("%w: witness token is unknown", ErrNotFound))
}

// CloseReceipt reads the sealed receipt. Before the seal the close is
// still open; after a lost confirmation it is unresolved.
func (a *RemoteAuthority) CloseReceipt(self journal.Owner, connectionID, contextID string) (RemoteCloseReceipt, error) {
	if err := checkRemoteOwner(self); err != nil {
		return RemoteCloseReceipt{}, err
	}
	if err := checkRemoteConnectionID(connectionID); err != nil {
		return RemoteCloseReceipt{}, err
	}
	if err := checkRemoteContextID(contextID); err != nil {
		return RemoteCloseReceipt{}, err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	rec, ok := a.connections[connectionID]
	if !ok {
		return RemoteCloseReceipt{}, remoteErr(BrowserRemoteLayerRemote, BrowserRemoteCodeUnknownConnection, fmt.Errorf("%w: connection %q", ErrNotFound, connectionID))
	}
	if !sameRemoteOwner(rec.owner, self) {
		return RemoteCloseReceipt{}, remoteErr(BrowserRemoteLayerService, BrowserRemoteCodeWrongOwner, fmt.Errorf("%w: connection is owned by another identity", ErrWrongOwner))
	}
	ctx, ok := rec.contexts[contextID]
	if !ok {
		return RemoteCloseReceipt{}, remoteErr(BrowserRemoteLayerRemote, BrowserRemoteCodeUnknownContext, fmt.Errorf("%w: context %q", ErrNotFound, contextID))
	}
	if ctx.closeState == RemoteCloseUnknown {
		return RemoteCloseReceipt{}, remoteErr(BrowserRemoteLayerCleanup, BrowserRemoteCodeCloseUnconfirmed, fmt.Errorf("%w: close confirmation is lost for context", ErrDenied))
	}
	if ctx.closeState != RemoteCloseClosed {
		return RemoteCloseReceipt{}, remoteErr(BrowserRemoteLayerCleanup, BrowserRemoteCodeOrphanOpen, fmt.Errorf("%w: context close is still open", ErrDenied))
	}
	return ctx.receipt, nil
}

// AssertRemoteCleanup judges one cleanup proof claim. Only a verified
// external-close receipt proves cleanup: worker receipts, server logs,
// exit codes, and any other non-authority claim refuse with
// forbidden-close before any verdict is read, and a forged receipt never
// verifies.
func AssertRemoteCleanup(claim RemoteCleanupClaim) (RemoteCleanupVerdict, error) {
	if claim.Kind != "external-close" {
		return RemoteCleanupVerdict{}, remoteErr(BrowserRemoteLayerCleanup, BrowserRemoteCodeForbiddenClose, fmt.Errorf("%w: cleanup proof kind is inadmissible: %q", ErrDenied, claim.Kind))
	}
	r := claim.Receipt
	if !validID(r.ContextID) || !validID(r.ConnectionID) || !validID(r.Server) {
		return RemoteCleanupVerdict{}, remoteErr(BrowserRemoteLayerCleanup, BrowserRemoteCodeForbiddenClose, fmt.Errorf("%w: independent close carries a malformed receipt", ErrInvalid))
	}
	if r.Owner.PID <= 0 || !validID(r.Owner.StartToken) {
		return RemoteCleanupVerdict{}, remoteErr(BrowserRemoteLayerCleanup, BrowserRemoteCodeForbiddenClose, fmt.Errorf("%w: independent close carries a malformed receipt", ErrInvalid))
	}
	recomputed := DigestRemoteClose(r.ContextID, r.ConnectionID, r.Owner, r.Server, r.WorkerDead)
	if recomputed != r.Digest {
		return RemoteCleanupVerdict{}, remoteErr(BrowserRemoteLayerCleanup, BrowserRemoteCodeForbiddenClose, fmt.Errorf("%w: close receipt digest does not verify", ErrDenied))
	}
	return RemoteCleanupVerdict{
		ContextID: r.ContextID, ConnectionID: r.ConnectionID, Witnessed: true,
		Digest: digestRemoteText("verdict:" + r.ContextID + ":" + r.ConnectionID),
	}, nil
}
