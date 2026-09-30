package workspace

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	journal "github.com/veighnsche/can-lang/tools/native-test-owner/journal"
)

// Mechanical errors. Callers distinguish them with errors.Is; operation
// results additionally carry the observed facts (remaining paths, partial
// outcomes, stability) that stay charged or unresolved.
var (
	ErrInvalid       = errors.New("workspace: invalid request")
	ErrEscape        = errors.New("workspace: path escapes workspace or traverses a link")
	ErrNotFound      = errors.New("workspace: not found")
	ErrWrongKind     = errors.New("workspace: wrong node kind")
	ErrCollision     = errors.New("workspace: create/replace rule violated")
	ErrBudget        = errors.New("workspace: budget exceeded")
	ErrSealed        = errors.New("workspace: sealed scope is read-only")
	ErrReplaced      = errors.New("workspace: path replaced by a different object")
	ErrActiveLease   = errors.New("workspace: active lease blocks disposal")
	ErrForeignLease  = errors.New("workspace: foreign lease blocks disposal")
	ErrUnresolved    = errors.New("workspace: unresolved cleanup")
	ErrConflict      = errors.New("workspace: operation ID replayed with different arguments")
	ErrIndeterminate = errors.New("workspace: effect may have occurred but outcome is unknown")
	ErrChanged       = errors.New("workspace: revision or content changed")
	ErrStaleHandle   = errors.New("workspace: stale handle generation")
	ErrClosedHandle  = errors.New("workspace: workspace is closed")
	ErrClosing       = errors.New("workspace: workspace is closing")
	ErrStrength      = errors.New("workspace: required stability strength unavailable")
	ErrStaleCursor   = errors.New("workspace: pagination invalidated by mutation")
	ErrOwner         = errors.New("workspace: wrong owner")
)

// Transitional bounds. Every field is finite; nothing here means unlimited.
const (
	// MaxListPage caps one list page.
	MaxListPage = 1024
	// MaxReadBytes caps one read call.
	MaxReadBytes = 8 << 20
	// MaxPartialLabels caps partial-acquisition labels recorded per op.
	MaxPartialLabels = 32
	// MaxNameLen caps one generated root name retry loop.
	maxRootRetries = 8
	// stagingDir is the internal atomic-publish staging directory.
	stagingDir = ".stage"
)

// Stability strengths for read/list/seal receipts.
type Stability string

const (
	// StabilityScopedRevision means capability-mediated mutations are
	// accounted for and the observed bytes match the tracked revision.
	StabilityScopedRevision Stability = "scoped_revision"
	// StabilityQualifiedSnapshot names a host snapshot guarantee. This
	// package never issues it; requiring it is rejected with ErrStrength.
	StabilityQualifiedSnapshot Stability = "qualified_snapshot"
	// StabilityChangedObserved means tracked content changed out of band.
	StabilityChangedObserved Stability = "changed_observed"
	// StabilityUnknown means the content was never capability-tracked (or
	// can no longer be proven), so no complete/stable claim is possible.
	StabilityUnknown Stability = "unknown"
)

// Kind names a filesystem node kind.
type Kind string

const (
	KindFile Kind = "file"
	KindDir  Kind = "dir"
	KindLink Kind = "symlink"
)

// Budgets are the finite workspace ceilings. Every field must be positive;
// zero or negative budgets are rejected.
type Budgets struct {
	MaxBytes   int64 `json:"maxBytes"`
	MaxEntries int   `json:"maxEntries"`
	MaxDepth   int   `json:"maxDepth"`
}

func (b Budgets) valid() bool {
	return b.MaxBytes > 0 && b.MaxEntries > 0 && b.MaxDepth > 0
}

// Handle binds owner, workspace and generation. Printed IDs and paths alone
// are never authority; every use re-checks registration, generation, owner
// and root identity.
type Handle struct {
	SpaceID    string `json:"spaceId"`
	Generation uint64 `json:"generation"`
}

// Lease is a read/retention grant over a workspace. A lease whose owner
// differs from the workspace owner is foreign; a lease that is neither
// released nor expired is active. Active or foreign leases block disposal
// and are reported instead of being cleaned.
type Lease struct {
	ID            string `json:"id"`
	PID           int    `json:"pid"`
	StartToken    string `json:"startToken"`
	ExpiresWallMs int64  `json:"expiresWallMs"`
	Released      bool   `json:"released"`
}

func (l Lease) active(nowMs int64) bool {
	return !l.Released && l.ExpiresWallMs > nowMs
}

// trackedEntry is the last capability-mediated state of one path.
type trackedEntry struct {
	kind   Kind
	size   int64
	digest string // sha256:... for files, "" otherwise
	target string // literal symlink target for links
}

type spaceState string

const (
	spaceLive    spaceState = "live"
	spaceClosing spaceState = "closing"
	spaceClosed  spaceState = "closed"
	spaceFailed  spaceState = "failed"
)

// space is one owned allocation.
type space struct {
	opID      string
	root      string // absolute path generated at reservation
	rootDev   uint64
	rootIno   uint64
	hasFileID bool
	budgets   Budgets
	purpose   string

	ownerPID   int
	ownerToken string

	r       *os.Root
	state   spaceState
	gen     uint64
	rev     uint64 // capability-mediated mutation counter
	partial bool   // partial allocation: failed setup is still owned

	chargedBytes   int64
	chargedEntries int
	maxDepth       int

	tracked map[string]trackedEntry

	sealedFull  bool
	sealedPaths map[string]bool // partial-seal roots (dirs cover children)
	sealRev     uint64
	seal        *SealReceipt

	leases map[string]*Lease

	dispose *DisposeReceipt
}

// Owner issues and owns workspaces. It journals intent before effects and
// retains every partial or unresolved allocation until reconciled.
type Owner struct {
	mu       sync.Mutex
	j        *journal.Journal
	pid      int
	token    string
	spaces   map[string]*space
	opResult map[string]outcome // completed repeatable op outcomes by opID
	clock    func() int64       // wall ms; overridden in tests
	// failSync injects a sync failure after bytes are written for bounded
	// fault tests. It is nil outside tests; production callers never set it.
	failSync error
}

// New returns an Owner bound to a journal and an owner identity. The
// spawn-start token disambiguates PID reuse; a bare PID is never authority.
func New(j *journal.Journal, pid int, startToken string) *Owner {
	return &Owner{
		j:        j,
		pid:      pid,
		token:    startToken,
		spaces:   make(map[string]*space),
		opResult: make(map[string]outcome),
		clock:    func() int64 { return time.Now().UnixMilli() },
	}
}

func (o *Owner) owner() journal.Owner {
	return journal.Owner{PID: o.pid, StartToken: o.token}
}

func digestOf(v any) (string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("%w: encode arguments: %v", ErrInvalid, err)
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func digestBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// journalIntent registers intent before the first effect. When opID was
// already journaled with the same digest it returns joined=true and the
// caller must rejoin (or report indeterminate) instead of re-dispatching.
// A different digest under the same ID is a protocol conflict.
func (o *Owner) journalIntent(opID string, args any, path journal.PathIdentity, leaseID string, leaseExp int64) (joined bool, err error) {
	d, err := digestOf(args)
	if err != nil {
		return false, err
	}
	if prev, ok := o.j.Lookup(opID); ok {
		if prev.ArgumentsDigest != d {
			return false, fmt.Errorf("%w: operation %q", ErrConflict, opID)
		}
		return true, nil
	}
	if _, err := o.j.Reserve(opID, d, o.owner(), path, leaseID, leaseExp); err != nil {
		if errors.Is(err, journal.ErrConflict) {
			return false, fmt.Errorf("%w: operation %q", ErrConflict, opID)
		}
		return false, fmt.Errorf("workspace: journal reserve: %w", err)
	}
	return false, nil
}

// outcome is the terminal fact of one operation: its result value (nil for
// error-only operations) plus the error the first execution returned, so a
// rejoin replays the identical outcome instead of redispatching.
type outcome struct {
	v   any
	err error
}

// rejoined returns the cached outcome for a joined opID, or ErrIndeterminate
// when the effect may have occurred but this owner has no terminal fact
// (for example after a restart): never re-dispatch, reconcile instead.
func (o *Owner) rejoined(opID string) (any, error) {
	if oc, ok := o.opResult[opID]; ok {
		return oc.v, oc.err
	}
	return nil, fmt.Errorf("%w: operation %q has no terminal fact here", ErrIndeterminate, opID)
}

func (o *Owner) remember(opID string, v any, err error) {
	o.opResult[opID] = outcome{v: v, err: err}
}

// failOp records a failed operation in the journal; journal errors are
// appended to partials rather than hiding the original failure.
func (o *Owner) failOp(opID string, partials ...string) {
	if len(partials) > MaxPartialLabels {
		partials = partials[:MaxPartialLabels]
	}
	for i, p := range partials {
		if len(p) > journal.MaxPartialLen {
			partials[i] = p[:journal.MaxPartialLen]
		}
	}
	_, _ = o.j.Advance(opID, journal.StateFailed, partials...)
}

// reject records a terminal pre-effect rejection (no effect occurred, but
// the operation ID is consumed) and returns the error for the caller.
// Repeats rejoin the same rejection instead of redispatching.
func (o *Owner) reject(opID string, err error, partials ...string) error {
	o.failOp(opID, partials...)
	o.remember(opID, nil, err)
	return err
}

// closeOp walks a successful operation to its terminal journal state.
func (o *Owner) closeOp(opID string) {
	for _, s := range []journal.State{journal.StateOpening, journal.StateLive, journal.StateClosing, journal.StateClosed} {
		if _, err := o.j.Advance(opID, s); err != nil {
			return
		}
	}
}

// splitRel validates a workspace-relative path and splits it into
// components. It rejects NUL, absolute paths, "." and ".." elements and
// empty segments. The root itself ("", ".") yields no components.
func splitRel(rel string) ([]string, error) {
	if strings.ContainsRune(rel, 0) {
		return nil, fmt.Errorf("%w: NUL in path", ErrInvalid)
	}
	if rel == "" || rel == "." {
		return nil, nil
	}
	if strings.HasPrefix(rel, "/") {
		return nil, fmt.Errorf("%w: absolute path %q", ErrEscape, rel)
	}
	parts := strings.Split(rel, "/")
	for _, p := range parts {
		if p == "" || p == "." || p == ".." {
			return nil, fmt.Errorf("%w: illegal segment in %q", ErrEscape, rel)
		}
	}
	return parts, nil
}

func joinRel(parts []string) string {
	return strings.Join(parts, "/")
}

// checkRootLocked verifies the workspace root is still the exact object
// reserved: a symlink or a device/inode mismatch is a replacement, never
// authority to operate through.
func (o *Owner) checkRootLocked(s *space) error {
	fi, err := os.Lstat(s.root)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: root %q vanished", ErrReplaced, s.root)
		}
		return fmt.Errorf("workspace: lstat root: %w", err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%w: root %q is now a symlink", ErrReplaced, s.root)
	}
	id, err := journal.StatPath(s.root)
	if err != nil {
		return fmt.Errorf("workspace: stat root identity: %w", err)
	}
	if s.hasFileID {
		if !id.HasFileID || id.Dev != s.rootDev || id.Ino != s.rootIno {
			return fmt.Errorf("%w: root %q", ErrReplaced, s.root)
		}
	}
	return nil
}

// lookupLocked resolves a handle to its live space, checking registration,
// generation, owner and root identity. Disposal uses its own path so that
// repeats can join a terminal receipt.
func (o *Owner) lookupLocked(h Handle, wantOwner bool) (*space, error) {
	s, ok := o.spaces[h.SpaceID]
	if !ok {
		return nil, fmt.Errorf("%w: unknown workspace", ErrNotFound)
	}
	if h.Generation != s.gen {
		return nil, fmt.Errorf("%w: want %d", ErrStaleHandle, s.gen)
	}
	if wantOwner && (o.pid != s.ownerPID || o.token != s.ownerToken) {
		return nil, fmt.Errorf("%w: workspace owned by another owner", ErrOwner)
	}
	if s.state == spaceClosed {
		return nil, ErrClosedHandle
	}
	if s.state == spaceClosing {
		return nil, ErrClosing
	}
	if err := o.checkRootLocked(s); err != nil {
		return nil, err
	}
	return s, nil
}

// checkParentsLocked verifies every intermediate component of parts exists
// as a real directory, refusing to traverse symlinks or non-directories.
// It returns the parent-relative path ("" for the root).
func checkParentsLocked(s *space, parts []string) (string, error) {
	for i := 0; i+1 < len(parts); i++ {
		p := joinRel(parts[:i+1])
		fi, err := s.r.Lstat(p)
		if err != nil {
			if os.IsNotExist(err) {
				return "", fmt.Errorf("%w: %q", ErrNotFound, p)
			}
			return "", fmt.Errorf("workspace: lstat %q: %w", p, err)
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("%w: %q traverses a symlink", ErrEscape, joinRel(parts))
		}
		if !fi.IsDir() {
			return "", fmt.Errorf("%w: %q is not a directory", ErrWrongKind, p)
		}
	}
	if len(parts) <= 1 {
		return "", nil
	}
	return joinRel(parts[:len(parts)-1]), nil
}

// sealedLocked reports whether rel ("" for the root) falls under a seal.
func sealedLocked(s *space, rel string) bool {
	if s.sealedFull {
		return true
	}
	if s.sealedPaths[rel] {
		return true
	}
	for p := range s.sealedPaths {
		if strings.HasPrefix(rel, p+"/") {
			return true
		}
	}
	return false
}

func kindOf(fi os.FileInfo) Kind {
	switch {
	case fi.Mode()&os.ModeSymlink != 0:
		return KindLink
	case fi.IsDir():
		return KindDir
	default:
		return KindFile
	}
}

// Reserve creates an owned scratch allocation with unpredictable root
// identity under parentDir. Intent is journaled before the directory is
// created; a failed setup remains owned and discoverable, never silently
// cleaned. Repeating the same opID with identical arguments joins the
// existing reservation.
func (o *Owner) Reserve(opID, parentDir string, b Budgets, purpose, leaseID string, leaseExpiresWallMs int64) (Handle, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if !b.valid() {
		return Handle{}, fmt.Errorf("%w: budgets must all be positive finite", ErrInvalid)
	}
	if parentDir == "" || !filepath.IsAbs(parentDir) {
		return Handle{}, fmt.Errorf("%w: parent directory must be absolute", ErrInvalid)
	}
	if strings.ContainsRune(purpose, 0) {
		return Handle{}, fmt.Errorf("%w: NUL in purpose", ErrInvalid)
	}

	// The idempotency digest covers the caller's stable arguments; the
	// unpredictable root name is bound in the journal path field. A repeat
	// therefore joins instead of conflicting on a fresh random name.
	args := struct {
		Parent  string  `json:"parent"`
		Budgets Budgets `json:"budgets"`
		Purpose string  `json:"purpose"`
	}{Parent: parentDir, Budgets: b, Purpose: purpose}
	d, err := digestOf(args)
	if err != nil {
		return Handle{}, err
	}
	if prev, ok := o.j.Lookup(opID); ok {
		if prev.ArgumentsDigest != d {
			return Handle{}, fmt.Errorf("%w: operation %q", ErrConflict, opID)
		}
		if s, ok := o.spaces[opID]; ok {
			if s.state == spaceFailed || s.partial {
				return Handle{SpaceID: opID, Generation: s.gen}, fmt.Errorf("%w: joined a partial allocation", ErrUnresolved)
			}
			return Handle{SpaceID: opID, Generation: s.gen}, nil
		}
		return Handle{}, fmt.Errorf("%w: reservation %q has no terminal fact here", ErrIndeterminate, opID)
	}
	var root string
	for i := 0; i < maxRootRetries; i++ {
		var rnd [16]byte
		if _, err := rand.Read(rnd[:]); err != nil {
			return Handle{}, fmt.Errorf("workspace: random root: %w", err)
		}
		cand := filepath.Join(parentDir, "ws-"+hex.EncodeToString(rnd[:]))
		if _, err := os.Lstat(cand); os.IsNotExist(err) {
			root = cand
			break
		}
	}
	if root == "" {
		return Handle{}, fmt.Errorf("%w: cannot pick a fresh root", ErrInvalid)
	}
	if _, err := o.j.Reserve(opID, d, o.owner(), journal.PathIdentity{Path: root}, leaseID, leaseExpiresWallMs); err != nil {
		if errors.Is(err, journal.ErrConflict) {
			return Handle{}, fmt.Errorf("%w: operation %q", ErrConflict, opID)
		}
		return Handle{}, fmt.Errorf("workspace: journal reserve: %w", err)
	}

	s := &space{
		opID: opID, root: root, budgets: b, purpose: purpose,
		ownerPID: o.pid, ownerToken: o.token,
		state: spaceLive, gen: 1,
		tracked:     make(map[string]trackedEntry),
		sealedPaths: make(map[string]bool),
		leases:      make(map[string]*Lease),
	}
	if err := os.Mkdir(root, 0o700); err != nil {
		o.spaces[opID] = s
		s.state = spaceFailed
		s.partial = true
		o.failOp(opID, "mkdir root: "+err.Error())
		return Handle{SpaceID: opID, Generation: s.gen}, fmt.Errorf("workspace: create root: %w", err)
	}
	id, err := journal.StatPath(root)
	if err != nil {
		o.spaces[opID] = s
		s.state = spaceFailed
		s.partial = true
		o.failOp(opID, "stat root")
		return Handle{SpaceID: opID, Generation: s.gen}, fmt.Errorf("workspace: bind root identity: %w", err)
	}
	s.rootDev, s.rootIno, s.hasFileID = id.Dev, id.Ino, id.HasFileID
	r, err := os.OpenRoot(root)
	if err != nil {
		o.spaces[opID] = s
		s.state = spaceFailed
		s.partial = true
		o.failOp(opID, "open root")
		return Handle{SpaceID: opID, Generation: s.gen}, fmt.Errorf("workspace: open root: %w", err)
	}
	s.r = r
	s.chargedEntries = 1 // the root itself
	if err := s.r.Mkdir(stagingDir, 0o700); err != nil {
		o.spaces[opID] = s
		s.partial = true
		o.failOp(opID, "mkdir staging")
		return Handle{SpaceID: opID, Generation: s.gen}, fmt.Errorf("workspace: create staging: %w", err)
	}
	s.chargedEntries++
	s.tracked[stagingDir] = trackedEntry{kind: KindDir}
	if leaseID != "" {
		s.leases[leaseID] = &Lease{ID: leaseID, PID: o.pid, StartToken: o.token, ExpiresWallMs: leaseExpiresWallMs}
	}
	o.spaces[opID] = s
	for _, st := range []journal.State{journal.StateOpening, journal.StateLive} {
		if _, err := o.j.Advance(opID, st); err != nil {
			s.state = spaceFailed
			s.partial = true
			return Handle{SpaceID: opID, Generation: s.gen}, fmt.Errorf("workspace: journal advance: %w", err)
		}
	}
	return Handle{SpaceID: opID, Generation: s.gen}, nil
}

// Charges reports the currently charged bytes, entries and depth of a live
// workspace. Unresolved workspaces stay charged until confirmed released.
func (o *Owner) Charges(h Handle) (bytes int64, entries, depth int, err error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	s, err := o.lookupLocked(h, false)
	if err != nil {
		return 0, 0, 0, err
	}
	return s.chargedBytes, s.chargedEntries, s.maxDepth, nil
}

// RootForTesting exposes the absolute root path for bounded test controls
// that simulate out-of-band mutation. Production callers must use handles;
// the path alone is never authority.
func (o *Owner) RootForTesting(h Handle) (string, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	s, ok := o.spaces[h.SpaceID]
	if !ok || h.Generation != s.gen {
		return "", ErrNotFound
	}
	return s.root, nil
}

// sortedKeys returns map keys in sorted order for deterministic receipts.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
