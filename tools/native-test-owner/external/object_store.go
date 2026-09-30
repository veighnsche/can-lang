// Owned object-store observer: P29-side prefix/session discipline.
//
// This file implements task K27's Go mirror: an owned object-store
// observer over in-memory doubles only. No bucket is contacted, no SDK is
// loaded, and no network I/O occurs. The observer plugs into this
// package's vocabulary only: prefix grants (KindPrefix), journaled
// admission, opaque handles, digest-only facts, and explicit release.
//
// Independence scope from the Can adapter under test (normative, mirrored
// in tools/runtime/test-services/object-store/service.ts): the observer
// shares no adapter code, decoded values, or client state; it proves
// prefix receipt, pinned sessions, exact bytes, paginated listing
// completeness, write settlement, and layered error provenance from its
// own seeded doubles alone. Adapter claims are compared against these
// facts; no fact is ever derived from an adapter claim.
//
// Split with the TS service: the TS service owns byte-exact put/get
// read-out, pagination mechanics, write settlement, and the bounded
// self-check. This file owns P29-journaled prefix admission, the pinned
// identity-bound sessions, the single-use generation-bound continuations,
// and the cleanup seal. Both enforce the same contract: owned-prefix
// admission (the caller prefix is not authority), opaque tokens and
// continuations, digest-only facts, and every failure naming its layer
// (engine/driver/store).
//
// A session is an observer-local opaque pin, not a P29 leg: the registry
// mints no session grants, so the observer mints its own unforgeable
// tokens and verifies them by table lookup. The prefix admission itself is
// journaled: ObservePrefix binds only to this registry's own live prefix
// grant, re-verified on every receipt. Sessions, pending writes, live
// objects, and the terminal scan are all cleanup dependents of the seal:
// SealCleanup succeeds only when every dependent has drained.

package external

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/veighnsche/can-lang/tools/native-test-owner/journal"
)

// StoreLayer names the failing layer of one observer error. Every
// observer failure names its layer; nothing unclassified is returned.
type StoreLayer string

// Observer layers. Engine owns prefix admission, capacity, ownership,
// lifecycle, and the cleanup seal; driver owns session pinning, tokens,
// and continuation discipline; store owns keys, bytes, page shape, and
// write settlement.
const (
	StoreLayerEngine StoreLayer = "engine"
	StoreLayerDriver StoreLayer = "driver"
	StoreLayerStore  StoreLayer = "store"
)

// Observer codes. Each code maps to exactly one layer, mirroring the TS
// service's closed code vocabulary.
const (
	StoreCodeUnknownPrefix = "unknown-prefix"      // engine
	StoreCodeWrongOwner    = "wrong-owner"         // engine
	StoreCodeCapacity      = "capacity-exhausted"  // engine
	StoreCodeClosedHandle  = "closed-handle"       // engine
	StoreCodeKeysRemain    = "keys-remain"         // engine
	StoreCodePending       = "pending-writes"      // engine
	StoreCodeScanOpen      = "scan-incomplete"     // engine
	StoreCodeSessionLive   = "session-live"        // engine
	StoreCodeUnknownSess   = "unknown-session"     // driver
	StoreCodeForgedToken   = "forged-token"        // driver
	StoreCodeForgedCont    = "forged-continuation" // driver
	StoreCodeStaleCont     = "stale-continuation"  // driver
	StoreCodeUnknownKey    = "unknown-key"         // store
	StoreCodeKeyEscapes    = "key-escapes-prefix"  // store
	StoreCodeBadKey        = "malformed-key"       // store
	StoreCodeWriteUnknown  = "write-unknown"       // store
	StoreCodeWriteBusy     = "write-busy"          // store
	StoreCodeWriteSettled  = "write-settled"       // store
	StoreCodeBadPage       = "bad-page"            // store
)

// Observer bounds. Every observer table is finite; nothing here means
// unlimited.
const (
	// MaxStoreSessions caps pinned sessions per observer.
	MaxStoreSessions = 16
	// MaxStoreObjects caps settled objects per observer.
	MaxStoreObjects = 128
	// MaxStoreObjectBytes caps one object payload.
	MaxStoreObjectBytes = 64 << 10
	// MaxStorePageSize caps one listing page.
	MaxStorePageSize = 32
	// MaxStorePending caps accepted unsettled writes per observer.
	MaxStorePending = 32
	// MaxStoreKeyLen caps one object key.
	MaxStoreKeyLen = 512
	// MaxStoreNameLen caps one observer session name.
	MaxStoreNameLen = 128
)

// StoreError is one layered observer failure. Layer always names engine,
// driver, or store; Code is the closed failure code; Err carries the
// wrapped registry cause for errors.Is matching.
type StoreError struct {
	Layer StoreLayer
	Code  string
	Err   error
}

func (e *StoreError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s:%s] %v", e.Layer, e.Code, e.Err)
	}
	return fmt.Sprintf("[%s:%s] observer refused", e.Layer, e.Code)
}

// Unwrap exposes the wrapped registry cause.
func (e *StoreError) Unwrap() error { return e.Err }

func storeErr(layer StoreLayer, code string, err error) *StoreError {
	return &StoreError{Layer: layer, Code: code, Err: err}
}

// StoreSession is the observable record of one pinned session. The token
// itself is never part of any fact; TokenDigest carries the digest only.
type StoreSession struct {
	OperationID string
	Name        string
	Prefix      string
	TokenDigest string
	Released    bool
}

// StoreReceipt is the prefix receipt: the observed prefix, its owner, the
// digest of the prefix handle, and the live session/object/pending
// counts.
type StoreReceipt struct {
	Prefix       string
	Owner        journal.Owner
	HandleDigest string
	Sessions     int
	Objects      int
	Pending      int
}

// StoreObjectFacts names one settled object without exposing its bytes:
// key, size, and digest only.
type StoreObjectFacts struct {
	Key    string
	Size   int
	Digest string
}

// StoreWriteFacts names one accepted write: its id, key, payload digest,
// and settlement state.
type StoreWriteFacts struct {
	WriteID     string
	Key         string
	BytesDigest string
	Settled     bool
}

// StorePageFacts is one listing page. Complete is explicit: only a page
// with Complete set observed the terminal position. A page with Complete
// unset proves nothing about keys beyond its edge, and no page alone
// proves cleanup.
type StorePageFacts struct {
	Prefix     string
	Keys       []StoreObjectFacts
	Count      int
	Complete   bool
	Generation int64
}

// StoreCleanupReceipt seals the prefix as cleaned up: zero live sessions,
// zero pending writes, zero live objects, and a terminal empty listing at
// the sealed generation.
type StoreCleanupReceipt struct {
	Prefix       string
	Owner        journal.Owner
	HandleDigest string
	Generation   int64
}

type storePin struct {
	opID        string
	name        string
	token       string
	tokenDigest string
	released    bool
}

type storeWrite struct {
	id      string
	key     string
	data    []byte
	settled bool
}

type storeContinuation struct {
	token      string
	after      string
	generation int64
	consumed   bool
}

// StoreObserver watches one owned P29 object prefix through its live
// prefix grant. A StoreObserver is safe for concurrent use; the registry
// it observes must outlive it.
type StoreObserver struct {
	mu       sync.Mutex
	r        *Registry
	prefixOp string
	prefix   string
	token    string
	owner    journal.Owner

	sessions map[string]*storePin
	objects  map[string][]byte
	pending  map[string]*storeWrite
	conts    map[string]*storeContinuation

	writeSeq int64
	// generation moves on every accepted write, settlement, and delete. A
	// terminal scan records the generation it observed; the seal requires
	// a terminal empty scan at the current generation.
	generation int64
	sealedGen  int64
	sealedNew  bool
	sealedHeld bool

	closed bool
}

func validStoreName(s string) bool {
	if len(s) == 0 || len(s) > MaxStoreNameLen {
		return false
	}
	return validID(s)
}

// scopeStoreKey validates one caller key against the owned prefix. Shape
// is checked before scope: a malformed key rejects even when it would
// also escape. Only keys strictly under the grant edge admit; the shared
// bucket outside the edge is unreachable.
func scopeStoreKey(prefix, key string) error {
	if len(key) == 0 || len(key) > MaxStoreKeyLen {
		return storeErr(StoreLayerStore, StoreCodeBadKey, fmt.Errorf("%w: malformed object key", ErrInvalid))
	}
	if strings.HasPrefix(key, "/") || strings.HasSuffix(key, "/") || strings.Contains(key, "//") {
		return storeErr(StoreLayerStore, StoreCodeBadKey, fmt.Errorf("%w: key is not a clean relative path", ErrInvalid))
	}
	for _, seg := range strings.Split(key, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return storeErr(StoreLayerStore, StoreCodeBadKey, fmt.Errorf("%w: key carries an empty or dot segment", ErrInvalid))
		}
	}
	if key == prefix {
		return storeErr(StoreLayerStore, StoreCodeBadKey, fmt.Errorf("%w: key must name an object under the prefix", ErrInvalid))
	}
	if !strings.HasPrefix(key, prefix+"/") {
		return storeErr(StoreLayerStore, StoreCodeKeyEscapes, fmt.Errorf("%w: key is outside the owned prefix", ErrDenied))
	}
	return nil
}

// ObservePrefix binds an observer to one live prefix grant. The token must
// be this registry's own live grant for a prefix leg: invented tokens,
// expired or revoked grants, and grants for other legs are rejected. The
// caller-supplied prefix string is never consulted: the observed prefix is
// the grant's own target. The observer records no raw secret: the receipt
// carries the prefix handle digest only.
func ObservePrefix(r *Registry, prefixToken string) (*StoreObserver, error) {
	if r == nil {
		return nil, storeErr(StoreLayerEngine, StoreCodeUnknownPrefix, fmt.Errorf("%w: nil registry", ErrInvalid))
	}
	r.mu.Lock()
	g, err := r.lookupGrantLocked(prefixToken)
	var cp Grant
	if err == nil {
		if g.Kind != KindPrefix {
			err = fmt.Errorf("%w: observer needs a prefix grant", ErrDenied)
		} else {
			cp = g.copy()
		}
	}
	r.mu.Unlock()
	if err != nil {
		return nil, storeErr(StoreLayerEngine, StoreCodeUnknownPrefix, err)
	}
	return &StoreObserver{
		r:         r,
		prefixOp:  cp.OperationID,
		prefix:    cp.Target,
		token:     cp.Token,
		owner:     cp.Owner,
		sessions:  make(map[string]*storePin),
		objects:   make(map[string][]byte),
		pending:   make(map[string]*storeWrite),
		conts:     make(map[string]*storeContinuation),
		sealedGen: -1,
	}, nil
}

// liveLocked re-verifies the prefix grant under the registry lock.
func (o *StoreObserver) liveLocked() error {
	g, err := o.r.lookupGrantLocked(o.token)
	if err != nil {
		return err
	}
	if g.Kind != KindPrefix || g.OperationID != o.prefixOp {
		return fmt.Errorf("%w: prefix grant changed legs", ErrDenied)
	}
	return nil
}

// Receipt reports the prefix receipt: prefix, owner, handle digest, and
// live counts. The raw token never crosses.
func (o *StoreObserver) Receipt() (StoreReceipt, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return StoreReceipt{}, storeErr(StoreLayerEngine, StoreCodeClosedHandle, fmt.Errorf("%w: observer closed", ErrInvalid))
	}
	o.r.mu.Lock()
	defer o.r.mu.Unlock()
	if err := o.liveLocked(); err != nil {
		return StoreReceipt{}, storeErr(StoreLayerEngine, StoreCodeUnknownPrefix, err)
	}
	sessions := 0
	for _, p := range o.sessions {
		if !p.released {
			sessions++
		}
	}
	pending := 0
	for _, w := range o.pending {
		if !w.settled {
			pending++
		}
	}
	return StoreReceipt{
		Prefix:       o.prefix,
		Owner:        o.owner,
		HandleDigest: handleDigest(o.token),
		Sessions:     sessions,
		Objects:      len(o.objects),
		Pending:      pending,
	}, nil
}

// OpenSession pins one session to the observed prefix. The token is minted
// here and verified by table lookup on every use, so invented tokens are
// never authority. The returned fact carries the token digest only; the
// raw token is returned alongside exactly once for the caller's later
// calls.
func (o *StoreObserver) OpenSession(opID, name string) (StoreSession, string, error) {
	if !validID(opID) {
		return StoreSession{}, "", storeErr(StoreLayerDriver, StoreCodeUnknownSess, fmt.Errorf("%w: malformed operation ID", ErrInvalid))
	}
	if !validStoreName(name) {
		return StoreSession{}, "", storeErr(StoreLayerStore, StoreCodeBadPage, fmt.Errorf("%w: malformed session name", ErrInvalid))
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return StoreSession{}, "", storeErr(StoreLayerEngine, StoreCodeClosedHandle, fmt.Errorf("%w: observer closed", ErrInvalid))
	}
	live := 0
	for _, p := range o.sessions {
		if !p.released {
			live++
		}
	}
	if live >= MaxStoreSessions {
		return StoreSession{}, "", storeErr(StoreLayerEngine, StoreCodeCapacity, fmt.Errorf("%w: observer session table full", ErrCapacity))
	}
	if _, dup := o.sessions[opID]; dup {
		return StoreSession{}, "", storeErr(StoreLayerDriver, StoreCodeUnknownSess, fmt.Errorf("%w: operation %q already pinned", ErrInvalid, opID))
	}
	token, err := mintToken()
	if err != nil {
		return StoreSession{}, "", storeErr(StoreLayerEngine, StoreCodeCapacity, err)
	}
	o.sessions[opID] = &storePin{opID: opID, name: name, token: token, tokenDigest: handleDigest(token)}
	return StoreSession{
		OperationID: opID, Name: name, Prefix: o.prefix,
		TokenDigest: handleDigest(token),
	}, token, nil
}

// lookupPinLocked verifies one pin and its token. Unknown pins and
// invented tokens are never authority.
func (o *StoreObserver) lookupPinLocked(opID, token string) (*storePin, error) {
	p, ok := o.sessions[opID]
	if !ok || p.released {
		return nil, storeErr(StoreLayerDriver, StoreCodeUnknownSess, fmt.Errorf("%w: operation %q", ErrNotFound, opID))
	}
	if token == "" || token != p.token {
		return nil, storeErr(StoreLayerDriver, StoreCodeForgedToken, fmt.Errorf("%w: session token is not the pinned token", ErrDenied))
	}
	return p, nil
}

// Session reports the observable record of one pinned session.
func (o *StoreObserver) Session(opID string) (StoreSession, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	p, ok := o.sessions[opID]
	if !ok {
		return StoreSession{}, storeErr(StoreLayerDriver, StoreCodeUnknownSess, fmt.Errorf("%w: operation %q", ErrNotFound, opID))
	}
	return StoreSession{
		OperationID: p.opID, Name: p.name, Prefix: o.prefix,
		TokenDigest: p.tokenDigest,
		Released:    p.released,
	}, nil
}

// CloseSession retires one pinned session under its own live token. A
// foreign identity is rejected and changes nothing.
func (o *StoreObserver) CloseSession(opID, token string, self journal.Owner) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return storeErr(StoreLayerEngine, StoreCodeClosedHandle, fmt.Errorf("%w: observer closed", ErrInvalid))
	}
	p, err := o.lookupPinLocked(opID, token)
	if err != nil {
		return err
	}
	if self.PID != o.owner.PID || self.StartToken != o.owner.StartToken {
		return storeErr(StoreLayerEngine, StoreCodeWrongOwner, fmt.Errorf("%w: close presents a foreign identity", ErrWrongOwner))
	}
	p.released = true
	return nil
}

// Put accepts one write. Accepted is not settled: the payload is staged
// under a write id and becomes an object only through SettleWrite.
// Staging copies the payload, so later caller mutation cannot corrupt the
// staged bytes.
func (o *StoreObserver) Put(opID, token, key string, data []byte) (StoreWriteFacts, error) {
	if err := scopeStoreKey(o.prefix, key); err != nil {
		return StoreWriteFacts{}, err
	}
	if len(data) > MaxStoreObjectBytes {
		return StoreWriteFacts{}, storeErr(StoreLayerEngine, StoreCodeCapacity, fmt.Errorf("%w: object exceeds the byte cap", ErrCapacity))
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return StoreWriteFacts{}, storeErr(StoreLayerEngine, StoreCodeClosedHandle, fmt.Errorf("%w: observer closed", ErrInvalid))
	}
	if _, err := o.lookupPinLocked(opID, token); err != nil {
		return StoreWriteFacts{}, err
	}
	live := 0
	for _, w := range o.pending {
		if !w.settled {
			live++
			if w.key == key {
				return StoreWriteFacts{}, storeErr(StoreLayerStore, StoreCodeWriteBusy, fmt.Errorf("%w: a write is already accepted for key", ErrDenied))
			}
		}
	}
	if live >= MaxStorePending {
		return StoreWriteFacts{}, storeErr(StoreLayerEngine, StoreCodeCapacity, fmt.Errorf("%w: pending-write table full", ErrCapacity))
	}
	if _, exists := o.objects[key]; !exists && len(o.objects) >= MaxStoreObjects {
		return StoreWriteFacts{}, storeErr(StoreLayerEngine, StoreCodeCapacity, fmt.Errorf("%w: object table full", ErrCapacity))
	}
	o.writeSeq++
	id := fmt.Sprintf("w%d", o.writeSeq)
	staged := make([]byte, len(data))
	copy(staged, data)
	o.pending[id] = &storeWrite{id: id, key: key, data: staged}
	o.generation++
	return StoreWriteFacts{WriteID: id, Key: key, BytesDigest: handleDigest(string(staged))}, nil
}

// SettleWrite settles one accepted write into a settled object.
func (o *StoreObserver) SettleWrite(opID, token, writeID string) (StoreObjectFacts, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return StoreObjectFacts{}, storeErr(StoreLayerEngine, StoreCodeClosedHandle, fmt.Errorf("%w: observer closed", ErrInvalid))
	}
	if _, err := o.lookupPinLocked(opID, token); err != nil {
		return StoreObjectFacts{}, err
	}
	w, ok := o.pending[writeID]
	if !ok {
		return StoreObjectFacts{}, storeErr(StoreLayerStore, StoreCodeWriteUnknown, fmt.Errorf("%w: no accepted write", ErrNotFound))
	}
	if w.settled {
		return StoreObjectFacts{}, storeErr(StoreLayerStore, StoreCodeWriteSettled, fmt.Errorf("%w: write already settled", ErrInvalid))
	}
	w.settled = true
	stored := make([]byte, len(w.data))
	copy(stored, w.data)
	o.objects[w.key] = stored
	o.generation++
	return StoreObjectFacts{Key: w.key, Size: len(stored), Digest: handleDigest(string(stored))}, nil
}

// PendingWrites lists every accepted unsettled write: ids, keys, and
// payload digests only.
func (o *StoreObserver) PendingWrites(opID, token string) ([]StoreWriteFacts, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return nil, storeErr(StoreLayerEngine, StoreCodeClosedHandle, fmt.Errorf("%w: observer closed", ErrInvalid))
	}
	if _, err := o.lookupPinLocked(opID, token); err != nil {
		return nil, err
	}
	var out []StoreWriteFacts
	for _, w := range o.pending {
		if !w.settled {
			out = append(out, StoreWriteFacts{WriteID: w.id, Key: w.key, BytesDigest: handleDigest(string(w.data))})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].WriteID < out[j].WriteID })
	return out, nil
}

// Get names one settled object without exposing its bytes. Accepted but
// unsettled writes are not visible here; they are observable only through
// PendingWrites.
func (o *StoreObserver) Get(opID, token, key string) (StoreObjectFacts, error) {
	if err := scopeStoreKey(o.prefix, key); err != nil {
		return StoreObjectFacts{}, err
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return StoreObjectFacts{}, storeErr(StoreLayerEngine, StoreCodeClosedHandle, fmt.Errorf("%w: observer closed", ErrInvalid))
	}
	if _, err := o.lookupPinLocked(opID, token); err != nil {
		return StoreObjectFacts{}, err
	}
	data, ok := o.objects[key]
	if !ok {
		return StoreObjectFacts{}, storeErr(StoreLayerStore, StoreCodeUnknownKey, fmt.Errorf("%w: no settled object", ErrNotFound))
	}
	return StoreObjectFacts{Key: key, Size: len(data), Digest: handleDigest(string(data))}, nil
}

// Delete removes one settled object. Keys outside the owned prefix reject
// without effect: the shared bucket is untouched.
func (o *StoreObserver) Delete(opID, token, key string) error {
	if err := scopeStoreKey(o.prefix, key); err != nil {
		return err
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return storeErr(StoreLayerEngine, StoreCodeClosedHandle, fmt.Errorf("%w: observer closed", ErrInvalid))
	}
	if _, err := o.lookupPinLocked(opID, token); err != nil {
		return err
	}
	if _, ok := o.objects[key]; !ok {
		return storeErr(StoreLayerStore, StoreCodeUnknownKey, fmt.Errorf("%w: no settled object", ErrNotFound))
	}
	delete(o.objects, key)
	o.generation++
	return nil
}

// List returns one page of settled keys in lexicographic order. The first
// page takes an empty continuation; later pages take the exact token of
// the previous page. A continuation records the generation it was minted
// at; any mutation since rejects as stale rather than silently skipping
// or repeating keys. A page that reaches the end is terminal and records
// the scan; every other page is explicitly incomplete and returns the next
// continuation alongside the facts.
func (o *StoreObserver) List(opID, token string, limit int, continuation string) (StorePageFacts, string, error) {
	if limit <= 0 || limit > MaxStorePageSize {
		return StorePageFacts{}, "", storeErr(StoreLayerStore, StoreCodeBadPage, fmt.Errorf("%w: page limit out of range", ErrInvalid))
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return StorePageFacts{}, "", storeErr(StoreLayerEngine, StoreCodeClosedHandle, fmt.Errorf("%w: observer closed", ErrInvalid))
	}
	if _, err := o.lookupPinLocked(opID, token); err != nil {
		return StorePageFacts{}, "", err
	}
	after := ""
	if continuation != "" {
		edge, ok := o.conts[continuation]
		if !ok || edge.consumed {
			return StorePageFacts{}, "", storeErr(StoreLayerDriver, StoreCodeForgedCont, fmt.Errorf("%w: continuation is not a live listing edge", ErrDenied))
		}
		if edge.generation != o.generation {
			return StorePageFacts{}, "", storeErr(StoreLayerDriver, StoreCodeStaleCont, fmt.Errorf("%w: the listing moved since this continuation was minted", ErrDenied))
		}
		edge.consumed = true
		after = edge.after
	}
	var names []string
	for name := range o.objects {
		if name > after {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	end := limit
	if end > len(names) {
		end = len(names)
	}
	page := names[:end]
	terminal := len(page) == len(names)
	next := ""
	if !terminal {
		tok, err := mintToken()
		if err != nil {
			return StorePageFacts{}, "", storeErr(StoreLayerEngine, StoreCodeCapacity, err)
		}
		next = tok
		o.conts[tok] = &storeContinuation{token: tok, after: page[len(page)-1], generation: o.generation}
	} else {
		o.sealedGen = o.generation
		o.sealedHeld = len(names) == 0
		o.sealedNew = true
	}
	keys := make([]StoreObjectFacts, 0, len(page))
	for _, name := range page {
		data := o.objects[name]
		keys = append(keys, StoreObjectFacts{Key: name, Size: len(data), Digest: handleDigest(string(data))})
	}
	return StorePageFacts{
		Prefix:     o.prefix,
		Keys:       keys,
		Count:      len(keys),
		Complete:   terminal,
		Generation: o.generation,
	}, next, nil
}

// SealCleanup seals the prefix as cleaned up. The seal requires all four,
// jointly: zero live sessions, zero unsettled writes, zero live objects,
// and a terminal empty listing observed at the current generation. An
// incomplete page, a stale terminal scan, or an eventually empty list
// observed while a write was still pending each reject with the reason
// named.
func (o *StoreObserver) SealCleanup(self journal.Owner) (StoreCleanupReceipt, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return StoreCleanupReceipt{}, storeErr(StoreLayerEngine, StoreCodeClosedHandle, fmt.Errorf("%w: observer closed", ErrInvalid))
	}
	if self.PID != o.owner.PID || self.StartToken != o.owner.StartToken {
		return StoreCleanupReceipt{}, storeErr(StoreLayerEngine, StoreCodeWrongOwner, fmt.Errorf("%w: seal presents a foreign identity", ErrWrongOwner))
	}
	for _, p := range o.sessions {
		if !p.released {
			return StoreCleanupReceipt{}, storeErr(StoreLayerEngine, StoreCodeSessionLive, fmt.Errorf("%w: session %q still pinned", ErrDependency, p.opID))
		}
	}
	for _, w := range o.pending {
		if !w.settled {
			return StoreCleanupReceipt{}, storeErr(StoreLayerEngine, StoreCodePending, fmt.Errorf("%w: write accepted but never settled", ErrDependency))
		}
	}
	if len(o.objects) > 0 {
		return StoreCleanupReceipt{}, storeErr(StoreLayerEngine, StoreCodeKeysRemain, fmt.Errorf("%w: objects still live", ErrDependency))
	}
	if !o.sealedNew || o.sealedGen != o.generation || !o.sealedHeld {
		return StoreCleanupReceipt{}, storeErr(StoreLayerEngine, StoreCodeScanOpen, fmt.Errorf("%w: no terminal empty listing at the current generation", ErrDenied))
	}
	return StoreCleanupReceipt{
		Prefix:       o.prefix,
		Owner:        o.owner,
		HandleDigest: handleDigest(o.token),
		Generation:   o.generation,
	}, nil
}

// Close retires the observer. Sessions still held must be closed first
// via CloseSession so no pin is stranded; Close itself settles nothing.
func (o *StoreObserver) Close() error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return storeErr(StoreLayerEngine, StoreCodeClosedHandle, fmt.Errorf("%w: observer already closed", ErrInvalid))
	}
	for _, p := range o.sessions {
		if !p.released {
			return storeErr(StoreLayerEngine, StoreCodeClosedHandle, fmt.Errorf("%w: session %q still pinned", ErrDependency, p.opID))
		}
	}
	o.closed = true
	return nil
}
