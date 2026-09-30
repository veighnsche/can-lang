package dispatch

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// Transitional bounds. Every field is finite; nothing here means unlimited.
const (
	// MaxGrants caps live entries in one grant table.
	MaxGrants = 4096
	// MaxGrantTTLMs caps one grant lifetime at 24 hours.
	MaxGrantTTLMs = 24 * 60 * 60 * 1000
	// grantTokenPrefix tags minted tokens; the token fits the owner_grant
	// envelope field (1..256 chars).
	grantTokenPrefix = "ng1-"
	grantTokenBytes  = 16
)

// Scope names the authority a grant carries.
type Scope string

const (
	ScopeRun  Scope = "run"
	ScopeCase Scope = "case"
)

// Grant is an unforgeable capability: a random token bound to one run (run
// scope) or one case of one run (case scope) with a fixed expiry. Grants
// are verified by table lookup; printed IDs alone are never authority.
type Grant struct {
	Token         string
	Scope         Scope
	RunID         string
	CaseID        string // "" for run grants
	ExpiresWallMs int64
	Revoked       bool
}

func (g Grant) copy() Grant { return g }

// GrantTable issues and verifies run/case grants. Revocation is per-token:
// derived case grants survive parent revocation unless revoked themselves,
// but a case grant never outlives its parent's expiry. A GrantTable is safe
// for concurrent use.
type GrantTable struct {
	mu     sync.Mutex
	grants map[string]*Grant
	clock  func() int64 // wall ms; overridden in tests
}

// NewGrantTable returns a table using the wall clock.
func NewGrantTable() *GrantTable {
	return NewGrantTableWithClock(func() int64 { return time.Now().UnixMilli() })
}

// NewGrantTableWithClock returns a table using clock for expiry checks.
func NewGrantTableWithClock(clock func() int64) *GrantTable {
	if clock == nil {
		clock = func() int64 { return time.Now().UnixMilli() }
	}
	return &GrantTable{grants: make(map[string]*Grant), clock: clock}
}

func mintToken() (string, error) {
	var b [grantTokenBytes]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("dispatch: random grant token: %w", err)
	}
	return grantTokenPrefix + hex.EncodeToString(b[:]), nil
}

func mintUniqueLocked(grants map[string]*Grant) (string, error) {
	for i := 0; i < 3; i++ {
		tok, err := mintToken()
		if err != nil {
			return "", err
		}
		if _, dup := grants[tok]; !dup {
			return tok, nil
		}
	}
	return "", fmt.Errorf("%w: grant token collision", ErrInvalid)
}

func validTTL(ttl int64) bool { return ttl > 0 && ttl <= MaxGrantTTLMs }

// pruneExpiredLocked drops expired unrevoked entries to reclaim bounded
// capacity. Revoked entries are retained as explicit audit facts.
func (t *GrantTable) pruneExpiredLocked(now int64) {
	for tok, g := range t.grants {
		if !g.Revoked && now >= g.ExpiresWallMs {
			delete(t.grants, tok)
		}
	}
}

// IssueRunGrant authorizes runID for ttlMs milliseconds.
func (t *GrantTable) IssueRunGrant(runID string, ttlMs int64) (Grant, error) {
	if !validID(runID) {
		return Grant{}, fmt.Errorf("%w: malformed run ID", ErrInvalid)
	}
	if !validTTL(ttlMs) {
		return Grant{}, fmt.Errorf("%w: grant TTL out of range", ErrInvalid)
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	now := t.clock()
	t.pruneExpiredLocked(now)
	if len(t.grants) >= MaxGrants {
		return Grant{}, fmt.Errorf("%w: grant table full", ErrCapacity)
	}
	tok, err := mintUniqueLocked(t.grants)
	if err != nil {
		return Grant{}, err
	}
	g := &Grant{Token: tok, Scope: ScopeRun, RunID: runID, ExpiresWallMs: now + ttlMs}
	t.grants[tok] = g
	return g.copy(), nil
}

// IssueCaseGrant derives a case grant from a live run grant. The case grant
// binds the same run and never outlives the parent run grant.
func (t *GrantTable) IssueCaseGrant(runToken, caseID string, ttlMs int64) (Grant, error) {
	if !validID(caseID) {
		return Grant{}, fmt.Errorf("%w: malformed case ID", ErrInvalid)
	}
	if !validTTL(ttlMs) {
		return Grant{}, fmt.Errorf("%w: grant TTL out of range", ErrInvalid)
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	now := t.clock()
	parent, ok := t.grants[runToken]
	if !ok {
		return Grant{}, fmt.Errorf("%w: unknown run grant", ErrNotFound)
	}
	if parent.Revoked {
		return Grant{}, fmt.Errorf("%w: run grant revoked", ErrRevoked)
	}
	if now >= parent.ExpiresWallMs {
		return Grant{}, fmt.Errorf("%w: run grant expired", ErrExpired)
	}
	if parent.Scope != ScopeRun {
		return Grant{}, fmt.Errorf("%w: case grants derive from run grants", ErrInvalid)
	}
	t.pruneExpiredLocked(now)
	if len(t.grants) >= MaxGrants {
		return Grant{}, fmt.Errorf("%w: grant table full", ErrCapacity)
	}
	tok, err := mintUniqueLocked(t.grants)
	if err != nil {
		return Grant{}, err
	}
	exp := now + ttlMs
	if exp > parent.ExpiresWallMs {
		exp = parent.ExpiresWallMs
	}
	g := &Grant{Token: tok, Scope: ScopeCase, RunID: parent.RunID, CaseID: caseID, ExpiresWallMs: exp}
	t.grants[tok] = g
	return g.copy(), nil
}

// Verify checks that token is live and bound to the wanted scope and IDs.
func (t *GrantTable) Verify(token string, wantScope Scope, runID, caseID string) error {
	if token == "" {
		return fmt.Errorf("%w: empty grant token", ErrInvalid)
	}
	if wantScope != ScopeRun && wantScope != ScopeCase {
		return fmt.Errorf("%w: unknown scope %q", ErrInvalid, wantScope)
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	g, ok := t.grants[token]
	if !ok {
		return fmt.Errorf("%w: unknown grant", ErrNotFound)
	}
	if g.Revoked {
		return fmt.Errorf("%w: grant revoked", ErrRevoked)
	}
	if t.clock() >= g.ExpiresWallMs {
		return fmt.Errorf("%w: grant expired", ErrExpired)
	}
	if g.Scope != wantScope {
		return fmt.Errorf("%w: grant scope %q is not %q", ErrInvalid, g.Scope, wantScope)
	}
	if g.RunID != runID {
		return fmt.Errorf("%w: grant bound to another run", ErrInvalid)
	}
	if wantScope == ScopeCase && g.CaseID != caseID {
		return fmt.Errorf("%w: grant bound to another case", ErrInvalid)
	}
	return nil
}

// Revoke marks a grant revoked; it is idempotent. Unknown tokens fail with
// ErrNotFound.
func (t *GrantTable) Revoke(token string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	g, ok := t.grants[token]
	if !ok {
		return fmt.Errorf("%w: unknown grant", ErrNotFound)
	}
	g.Revoked = true
	return nil
}

// Lookup returns a copy of one grant. It reports false for unknown tokens.
func (t *GrantTable) Lookup(token string) (Grant, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	g, ok := t.grants[token]
	if !ok {
		return Grant{}, false
	}
	return g.copy(), true
}
