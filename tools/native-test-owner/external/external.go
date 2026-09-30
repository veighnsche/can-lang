package external

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/veighnsche/can-lang/tools/native-test-owner/journal"
)

// Kind names one generic external-service resource leg. Every leg is
// recorded unqualified: engine-specific release implementations belong to
// later tasks.
type Kind string

const (
	KindListener   Kind = "listener"
	KindConnection Kind = "connection"
	KindContext    Kind = "context"
	KindNamespace  Kind = "db-namespace"
	KindCredential Kind = "db-credential"
	KindPrefix     Kind = "object-prefix"
)

// QualificationUnqualified is the only qualification this package records.
// Generic grants alone never prove cleanup.
const QualificationUnqualified = "unqualified"

// Transitional bounds. Every field is finite; nothing here means unlimited.
const (
	// MaxDeclared caps one finite declared set (listeners, destinations,
	// namespaces, prefixes).
	MaxDeclared = 64
	// MaxResources caps journaled resources of one registry.
	MaxResources = 1024
	// MaxGrants caps entries in one grant table. Revoked entries are
	// retained as explicit audit facts and count toward the cap; only
	// expired unrevoked entries are pruned.
	MaxGrants = 4096
	// MaxTargetLen caps one target name.
	MaxTargetLen = 256
	// MaxGrantTTLMs caps one grant lifetime at 24 hours.
	MaxGrantTTLMs = 24 * 60 * 60 * 1000

	grantTokenPrefix = "ex1-"
	grantTokenBytes  = 16
)

// Config declares the finite service sets this registry may admit. Every
// set is fixed at construction; anything outside them is foreign. Clock
// supplies wall ms for grant expiry; nil selects the wall clock.
type Config struct {
	Listeners    []string
	Destinations []string
	Namespaces   []string
	Prefixes     []string
	Clock        func() int64
}

// Grant is an unforgeable capability: a random token bound to one owner,
// one operation and one resource leg with a fixed expiry. Grants are
// verified by table lookup; printed IDs alone are never authority.
type Grant struct {
	Token         string
	Kind          Kind
	Target        string
	OperationID   string
	Owner         journal.Owner
	ExpiresWallMs int64
	Revoked       bool
}

func (g Grant) copy() Grant { return g }

// Charge is one retained resource charge. Unsettled charges are never
// silently dropped: only an explicit Release settles them.
type Charge struct {
	OperationID string
	Kind        Kind
	Target      string
	Settled     bool
}

func (c Charge) copy() Charge { return c }

// Fact is the observable record of one resource leg. Qualification is
// always unqualified and CleanupProven is always false: generic ownership
// never claims an engine-specific release it cannot perform. Credential
// facts carry only a digest of the opaque handle, never a raw secret.
type Fact struct {
	OperationID   string
	Kind          Kind
	Target        string
	Owner         journal.Owner
	Qualification string
	CleanupProven bool
	Released      bool
	HandleDigest  string // "sha256:<hex>" over the opaque grant token
}

// Leg is one recorded service leg for evidence.
type Leg struct {
	OperationID   string
	Kind          Kind
	Target        string
	Qualification string
}

// resource is the live in-memory double behind one journaled admission.
// Doubles are local fakes only: no sockets are bound, no database or
// cloud account is contacted.
type resource struct {
	opID     string
	kind     Kind
	target   string
	parent   string // parent opID for context/credential legs; "" otherwise
	owner    journal.Owner
	grant    string // live grant token
	released bool
}

func validID(s string) bool {
	if len(s) == 0 || len(s) > 128 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		ok := c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' ||
			c == '_' || c == '.' || c == ':' || c == '-'
		if !ok {
			return false
		}
		if i == 0 && !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}

func validTTL(ttl int64) bool { return ttl > 0 && ttl <= MaxGrantTTLMs }

func mintToken() (string, error) {
	var b [grantTokenBytes]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("external: random grant token: %w", err)
	}
	return grantTokenPrefix + hex.EncodeToString(b[:]), nil
}

// admitDigest binds an admission to its exact leg: kind, target and parent.
func admitDigest(kind Kind, target, parent string) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%s", kind, target, parent)))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func handleDigest(token string) string {
	sum := sha256.Sum256([]byte(token))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// Registry owns generic external-service resources: journaled admission,
// finite target sets, unforgeable grants, retained charges and recovery
// acknowledgments. The journal is owned by the caller and must outlive the
// registry. A Registry is safe for concurrent use.
type Registry struct {
	mu     sync.Mutex
	j      *journal.Journal
	owner  journal.Owner
	clock  func() int64
	finite map[Kind]map[string]bool
	prefix []string

	grants    map[string]*Grant
	resources map[string]*resource
	charges   map[string]*Charge
	children  map[string][]string

	dispatches map[string]*dispatchRec

	stopped  bool
	fenced   bool
	fenceAck *FenceAck
}

func checkDeclared(name string, vals []string) (map[string]bool, error) {
	if len(vals) > MaxDeclared {
		return nil, fmt.Errorf("%w: too many declared %s", ErrInvalid, name)
	}
	out := make(map[string]bool, len(vals))
	for _, v := range vals {
		if v == "" || len(v) > MaxTargetLen {
			return nil, fmt.Errorf("%w: malformed declared %s", ErrInvalid, name)
		}
		if out[v] {
			return nil, fmt.Errorf("%w: duplicate declared %s %q", ErrInvalid, name, v)
		}
		out[v] = true
	}
	return out, nil
}

// New returns a Registry bound to a journal and an owner identity. The
// spawn-start token disambiguates PID reuse; a bare PID is never authority.
func New(j *journal.Journal, pid int, startToken string, cfg Config) (*Registry, error) {
	if j == nil {
		return nil, fmt.Errorf("%w: nil journal", ErrInvalid)
	}
	if pid <= 0 || startToken == "" || len(startToken) > journal.MaxStartTokenLen {
		return nil, fmt.Errorf("%w: owner needs positive PID and spawn-start token", ErrInvalid)
	}
	listeners, err := checkDeclared("listener", cfg.Listeners)
	if err != nil {
		return nil, err
	}
	destinations, err := checkDeclared("destination", cfg.Destinations)
	if err != nil {
		return nil, err
	}
	namespaces, err := checkDeclared("namespace", cfg.Namespaces)
	if err != nil {
		return nil, err
	}
	prefixSet, err := checkDeclared("prefix", cfg.Prefixes)
	if err != nil {
		return nil, err
	}
	prefixes := make([]string, 0, len(prefixSet))
	for p := range prefixSet {
		prefixes = append(prefixes, p)
	}
	sort.Strings(prefixes)
	clock := cfg.Clock
	if clock == nil {
		clock = func() int64 { return time.Now().UnixMilli() }
	}
	return &Registry{
		j:     j,
		owner: journal.Owner{PID: pid, StartToken: startToken},
		clock: clock,
		finite: map[Kind]map[string]bool{
			KindListener:   listeners,
			KindConnection: destinations,
			KindNamespace:  namespaces,
		},
		prefix:     prefixes,
		grants:     make(map[string]*Grant),
		resources:  make(map[string]*resource),
		charges:    make(map[string]*Charge),
		children:   make(map[string][]string),
		dispatches: make(map[string]*dispatchRec),
	}, nil
}

// Owner reports the registry's spawn-start identity.
func (r *Registry) Owner() journal.Owner { return r.owner }

// memberLocked reports whether target is a declared member for kind.
// Prefix legs match when target equals or extends a declared prefix.
func (r *Registry) memberLocked(kind Kind, target string) bool {
	if kind == KindPrefix {
		for _, p := range r.prefix {
			if target == p || strings.HasPrefix(target, p+"/") {
				return true
			}
		}
		return false
	}
	return r.finite[kind][target]
}

// mintGrantLocked mints one unforgeable grant. Expired unrevoked entries
// are pruned to reclaim bounded capacity; revoked entries are retained as
// explicit audit facts.
func (r *Registry) mintGrantLocked(kind Kind, target, opID string, ttlMs int64) (Grant, error) {
	now := r.clock()
	for tok, g := range r.grants {
		if !g.Revoked && now >= g.ExpiresWallMs {
			delete(r.grants, tok)
		}
	}
	if len(r.grants) >= MaxGrants {
		return Grant{}, fmt.Errorf("%w: grant table full", ErrCapacity)
	}
	var tok string
	for i := 0; i < 3; i++ {
		t, err := mintToken()
		if err != nil {
			return Grant{}, err
		}
		if _, dup := r.grants[t]; !dup {
			tok = t
			break
		}
	}
	if tok == "" {
		return Grant{}, fmt.Errorf("%w: grant token collision", ErrInvalid)
	}
	g := &Grant{
		Token: tok, Kind: kind, Target: target, OperationID: opID,
		Owner: r.owner, ExpiresWallMs: now + ttlMs,
	}
	r.grants[tok] = g
	return g.copy(), nil
}

// admit journals intent before creating any effect, then records the
// resource, its grant and its retained charge. Every admission failure
// leaves no grant, no resource, no charge and no new journal intent behind
// it beyond what the journal itself reports.
func (r *Registry) admit(opID string, kind Kind, target, parent string, ttlMs int64) (Grant, error) {
	if !validID(opID) {
		return Grant{}, fmt.Errorf("%w: malformed operation ID", ErrInvalid)
	}
	if target == "" || len(target) > MaxTargetLen {
		return Grant{}, fmt.Errorf("%w: malformed target", ErrInvalid)
	}
	if !validTTL(ttlMs) {
		return Grant{}, fmt.Errorf("%w: grant TTL out of range", ErrInvalid)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.stopped {
		return Grant{}, fmt.Errorf("%w: registry quiesced", ErrQuiesced)
	}
	if !r.memberLocked(kind, target) {
		return Grant{}, fmt.Errorf("%w: %s %q is not a declared service target", ErrForeign, kind, target)
	}
	if len(r.resources) >= MaxResources {
		return Grant{}, fmt.Errorf("%w: resource table full", ErrCapacity)
	}
	if _, dup := r.resources[opID]; dup {
		return Grant{}, fmt.Errorf("%w: operation %q already admitted", ErrInvalid, opID)
	}
	// Intent before effect: the journal reservation precedes the double.
	digest := admitDigest(kind, target, parent)
	if _, err := r.j.Reserve(opID, digest, r.owner, journal.PathIdentity{}, "", 0); err != nil {
		return Grant{}, fmt.Errorf("external: journal reserve: %w", err)
	}
	for _, st := range []journal.State{journal.StateOpening, journal.StateLive} {
		if _, err := r.j.Advance(opID, st); err != nil {
			return Grant{}, fmt.Errorf("external: journal advance: %w", err)
		}
	}
	g, err := r.mintGrantLocked(kind, target, opID, ttlMs)
	if err != nil {
		_, _ = r.j.Advance(opID, journal.StateFailed, "grant unavailable")
		return Grant{}, err
	}
	r.resources[opID] = &resource{opID: opID, kind: kind, target: target, parent: parent, owner: r.owner, grant: g.Token}
	r.charges[opID] = &Charge{OperationID: opID, Kind: kind, Target: target}
	if parent != "" {
		r.children[parent] = append(r.children[parent], opID)
	}
	return g, nil
}

// AdmitListener admits one finite declared listener.
func (r *Registry) AdmitListener(opID, name string, ttlMs int64) (Grant, error) {
	return r.admit(opID, KindListener, name, "", ttlMs)
}

// AdmitConnection admits one remote connection to a finite declared
// destination. The connection binds to this registry's spawn-start
// identity; the double performs no network I/O.
func (r *Registry) AdmitConnection(opID, server string, ttlMs int64) (Grant, error) {
	return r.admit(opID, KindConnection, server, "", ttlMs)
}

// GrantNamespace admits one finite declared database namespace.
func (r *Registry) GrantNamespace(opID, namespace string, ttlMs int64) (Grant, error) {
	return r.admit(opID, KindNamespace, namespace, "", ttlMs)
}

// GrantPrefix admits one object-store prefix at or under a declared prefix.
func (r *Registry) GrantPrefix(opID, prefix string, ttlMs int64) (Grant, error) {
	return r.admit(opID, KindPrefix, prefix, "", ttlMs)
}

// lookupGrantLocked verifies one grant by table lookup. Unknown tokens —
// caller-invented authority — fail with ErrNotFound.
func (r *Registry) lookupGrantLocked(token string) (*Grant, error) {
	if token == "" {
		return nil, fmt.Errorf("%w: empty grant token", ErrInvalid)
	}
	g, ok := r.grants[token]
	if !ok {
		return nil, fmt.Errorf("%w: unknown grant", ErrNotFound)
	}
	if g.Revoked {
		return nil, fmt.Errorf("%w: grant revoked", ErrRevoked)
	}
	if r.clock() >= g.ExpiresWallMs {
		return nil, fmt.Errorf("%w: grant expired", ErrExpired)
	}
	if g.Owner.PID != r.owner.PID || g.Owner.StartToken != r.owner.StartToken {
		return nil, fmt.Errorf("%w: grant bound to another owner", ErrWrongOwner)
	}
	return g, nil
}

// OpenContext opens one remote context under a live connection grant. The
// connection grant must be this registry's own live grant for a connection
// leg: invented tokens and grants for other legs are rejected, and the
// context becomes a cleanup dependent of the connection.
func (r *Registry) OpenContext(opID, connGrant, name string, ttlMs int64) (Grant, error) {
	if !validID(opID) {
		return Grant{}, fmt.Errorf("%w: malformed operation ID", ErrInvalid)
	}
	if name == "" || len(name) > MaxTargetLen {
		return Grant{}, fmt.Errorf("%w: malformed context name", ErrInvalid)
	}
	if !validTTL(ttlMs) {
		return Grant{}, fmt.Errorf("%w: grant TTL out of range", ErrInvalid)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.stopped {
		return Grant{}, fmt.Errorf("%w: registry quiesced", ErrQuiesced)
	}
	parent, err := r.lookupGrantLocked(connGrant)
	if err != nil {
		return Grant{}, err
	}
	if parent.Kind != KindConnection {
		return Grant{}, fmt.Errorf("%w: context needs a connection grant", ErrDenied)
	}
	if len(r.resources) >= MaxResources {
		return Grant{}, fmt.Errorf("%w: resource table full", ErrCapacity)
	}
	if _, dup := r.resources[opID]; dup {
		return Grant{}, fmt.Errorf("%w: operation %q already admitted", ErrInvalid, opID)
	}
	target := parent.Target + "/" + name
	digest := admitDigest(KindContext, target, parent.OperationID)
	if _, err := r.j.Reserve(opID, digest, r.owner, journal.PathIdentity{}, "", 0); err != nil {
		return Grant{}, fmt.Errorf("external: journal reserve: %w", err)
	}
	for _, st := range []journal.State{journal.StateOpening, journal.StateLive} {
		if _, err := r.j.Advance(opID, st); err != nil {
			return Grant{}, fmt.Errorf("external: journal advance: %w", err)
		}
	}
	g, err := r.mintGrantLocked(KindContext, target, opID, ttlMs)
	if err != nil {
		_, _ = r.j.Advance(opID, journal.StateFailed, "grant unavailable")
		return Grant{}, err
	}
	r.resources[opID] = &resource{opID: opID, kind: KindContext, target: target, parent: parent.OperationID, owner: r.owner, grant: g.Token}
	r.charges[opID] = &Charge{OperationID: opID, Kind: KindContext, Target: target}
	r.children[parent.OperationID] = append(r.children[parent.OperationID], opID)
	return g, nil
}

// IssueCredential issues one opaque credential handle under a live
// namespace grant. The handle is the grant token itself; no raw secret is
// minted, stored or reported — facts carry only a digest of the token.
// The credential becomes a cleanup dependent of the namespace.
func (r *Registry) IssueCredential(opID, nsGrant string, ttlMs int64) (Grant, error) {
	if !validID(opID) {
		return Grant{}, fmt.Errorf("%w: malformed operation ID", ErrInvalid)
	}
	if !validTTL(ttlMs) {
		return Grant{}, fmt.Errorf("%w: grant TTL out of range", ErrInvalid)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.stopped {
		return Grant{}, fmt.Errorf("%w: registry quiesced", ErrQuiesced)
	}
	parent, err := r.lookupGrantLocked(nsGrant)
	if err != nil {
		return Grant{}, err
	}
	if parent.Kind != KindNamespace {
		return Grant{}, fmt.Errorf("%w: credential needs a namespace grant", ErrDenied)
	}
	if len(r.resources) >= MaxResources {
		return Grant{}, fmt.Errorf("%w: resource table full", ErrCapacity)
	}
	if _, dup := r.resources[opID]; dup {
		return Grant{}, fmt.Errorf("%w: operation %q already admitted", ErrInvalid, opID)
	}
	target := parent.Target + "/credential"
	digest := admitDigest(KindCredential, target, parent.OperationID)
	if _, err := r.j.Reserve(opID, digest, r.owner, journal.PathIdentity{}, "", 0); err != nil {
		return Grant{}, fmt.Errorf("external: journal reserve: %w", err)
	}
	for _, st := range []journal.State{journal.StateOpening, journal.StateLive} {
		if _, err := r.j.Advance(opID, st); err != nil {
			return Grant{}, fmt.Errorf("external: journal advance: %w", err)
		}
	}
	g, err := r.mintGrantLocked(KindCredential, target, opID, ttlMs)
	if err != nil {
		_, _ = r.j.Advance(opID, journal.StateFailed, "grant unavailable")
		return Grant{}, err
	}
	r.resources[opID] = &resource{opID: opID, kind: KindCredential, target: target, parent: parent.OperationID, owner: r.owner, grant: g.Token}
	r.charges[opID] = &Charge{OperationID: opID, Kind: KindCredential, Target: target}
	r.children[parent.OperationID] = append(r.children[parent.OperationID], opID)
	return g, nil
}

// Release releases one owned resource under its own live grant. Forged
// releases — invented tokens, tokens bound to another operation, or a
// foreign identity — are rejected and change nothing; a release with a
// live dependent still held is rejected with ErrDependency. Success walks
// the journal to closed, revokes the grant and settles the retained
// charge. CleanupProven stays false: the generic release cannot stand in
// for an engine-specific one.
func (r *Registry) Release(opID, token string, self journal.Owner) error {
	if !validID(opID) {
		return fmt.Errorf("%w: malformed operation ID", ErrInvalid)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	res, ok := r.resources[opID]
	if !ok {
		return fmt.Errorf("%w: operation %q", ErrNotFound, opID)
	}
	if res.released {
		return fmt.Errorf("%w: operation %q already released", ErrInvalid, opID)
	}
	if self.PID != res.owner.PID || self.StartToken != res.owner.StartToken {
		return fmt.Errorf("%w: release presents a foreign identity", ErrWrongOwner)
	}
	g, err := r.lookupGrantLocked(token)
	if err != nil {
		return err
	}
	if g.OperationID != opID {
		return fmt.Errorf("%w: grant bound to another operation", ErrDenied)
	}
	for _, child := range r.children[opID] {
		if c, known := r.resources[child]; known && !c.released {
			return fmt.Errorf("%w: dependent %q still live", ErrDependency, child)
		}
	}
	if err := r.j.VerifyAuthority(opID, self, ""); err != nil {
		return fmt.Errorf("external: release authority: %w", err)
	}
	for _, st := range []journal.State{journal.StateClosing, journal.StateClosed} {
		if _, err := r.j.Advance(opID, st); err != nil {
			return fmt.Errorf("external: journal advance: %w", err)
		}
	}
	g.Revoked = true
	res.released = true
	if ch, known := r.charges[opID]; known {
		ch.Settled = true
	}
	return nil
}

// Facts reports the observable record of one resource leg. Credential
// facts expose only the handle digest; the opaque token itself is never
// part of any fact.
func (r *Registry) Facts(opID string) (Fact, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	res, ok := r.resources[opID]
	if !ok {
		return Fact{}, fmt.Errorf("%w: operation %q", ErrNotFound, opID)
	}
	return Fact{
		OperationID:   res.opID,
		Kind:          res.kind,
		Target:        res.target,
		Owner:         res.owner,
		Qualification: QualificationUnqualified,
		CleanupProven: false,
		Released:      res.released,
		HandleDigest:  handleDigest(res.grant),
	}, nil
}

// Legs lists every recorded service leg in operation order. Every leg is
// explicitly unqualified; the list is the evidence that generic grants
// alone never claim cleanup proven.
func (r *Registry) Legs() []Leg {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Leg, 0, len(r.resources))
	for _, res := range r.resources {
		out = append(out, Leg{
			OperationID:   res.opID,
			Kind:          res.kind,
			Target:        res.target,
			Qualification: QualificationUnqualified,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].OperationID < out[j].OperationID })
	return out
}

// Unsettled reports every retained unsettled charge in operation order.
// Charges are retained until an explicit Release settles them; nothing
// else — quiescence, fencing, recovery — settles or drops one.
func (r *Registry) Unsettled() []Charge {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []Charge
	for _, ch := range r.charges {
		if !ch.Settled {
			out = append(out, ch.copy())
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].OperationID < out[j].OperationID })
	return out
}

// Live reports every unreleased operation ID in sorted order.
func (r *Registry) Live() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.liveLocked()
}

func (r *Registry) liveLocked() []string {
	var out []string
	for id, res := range r.resources {
		if !res.released {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}
