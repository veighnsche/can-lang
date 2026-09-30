package journal

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"syscall"
)

// Resource lifecycle states. Normal flow is reserved -> opening -> live ->
// closing -> closed; failed is a terminal side state reachable from any
// non-terminal state.
type State string

const (
	StateReserved State = "reserved"
	StateOpening  State = "opening"
	StateLive     State = "live"
	StateClosing  State = "closing"
	StateClosed   State = "closed"
	StateFailed   State = "failed"
)

// Transitional bounds. Every field is finite; nothing here means unlimited.
const (
	// MaxOperations caps reservations held by one journal handle.
	MaxOperations = 65536
	// MaxPartialsPerOperation caps partial-acquisition entries per operation.
	MaxPartialsPerOperation = 64
	// MaxPartialLen caps one partial-acquisition label.
	MaxPartialLen = 256
	// MaxStartTokenLen caps the caller-supplied spawn-start token.
	MaxStartTokenLen = 256
	// MaxLeaseIDLen caps the lease identifier.
	MaxLeaseIDLen = 256
	// MaxLogBytes caps the durable log replayed at open.
	MaxLogBytes = 8 << 20
	// logFileName is the append-only record file inside the journal directory.
	logFileName = "journal.jsonl"
)

var (
	ErrInvalid    = errors.New("journal: invalid request")
	ErrConflict   = errors.New("journal: conflicting replay")
	ErrTransition = errors.New("journal: illegal state transition")
	ErrNotFound   = errors.New("journal: unknown operation")
	ErrWrongOwner = errors.New("journal: wrong owner")
	ErrWrongPath  = errors.New("journal: wrong or replaced path")
	ErrCorrupt    = errors.New("journal: corrupt log")
	ErrClosed     = errors.New("journal: closed")
	errSyncFault  = errors.New("journal: injected sync fault")
)

// Owner identifies the reserving owner: a PID plus a caller-supplied
// spawn-start token. The token disambiguates PID reuse; a bare PID is never
// authority by itself.
type Owner struct {
	PID        int    `json:"pid"`
	StartToken string `json:"startToken"`
}

// PathIdentity binds an absolute path to the file object observed there.
// HasFileID is false when the marker did not exist at reservation time
// (intent is registered before the effect) or when the platform exposes no
// stable file identity.
type PathIdentity struct {
	Path      string `json:"path"`
	Dev       uint64 `json:"dev,omitempty"`
	Ino       uint64 `json:"ino,omitempty"`
	HasFileID bool   `json:"hasFileID"`
}

// Reservation is one journaled ownership record.
type Reservation struct {
	OperationID        string       `json:"operationId"`
	ArgumentsDigest    string       `json:"argumentsDigest"`
	Owner              Owner        `json:"owner"`
	Path               PathIdentity `json:"path"`
	LeaseID            string       `json:"leaseId,omitempty"`
	LeaseExpiresWallMs int64        `json:"leaseExpiresWallMs,omitempty"`
	State              State        `json:"state"`
	Partial            []string     `json:"partial,omitempty"`
	Seq                uint64       `json:"seq"`
}

func (r *Reservation) copy() *Reservation {
	out := *r
	out.Partial = append([]string(nil), r.Partial...)
	return &out
}

func terminal(s State) bool { return s == StateClosed || s == StateFailed }

func knownState(s State) bool {
	switch s {
	case StateReserved, StateOpening, StateLive, StateClosing, StateClosed, StateFailed:
		return true
	}
	return false
}

// legalAdvance reports whether from -> to follows the monotonic lifecycle.
func legalAdvance(from, to State) bool {
	if terminal(from) {
		return false
	}
	if to == StateFailed {
		return true
	}
	switch from {
	case StateReserved:
		return to == StateOpening
	case StateOpening:
		return to == StateLive
	case StateLive:
		return to == StateClosing
	case StateClosing:
		return to == StateClosed
	}
	return false
}

// validID matches the shared wire ID shape: ^[A-Za-z0-9][A-Za-z0-9_.:-]{0,127}$.
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

// validDigest matches the shared digest shape: sha256:<64 lowercase hex>.
func validDigest(s string) bool {
	if !strings.HasPrefix(s, "sha256:") || len(s) != 7+64 {
		return false
	}
	for i := 7; i < len(s); i++ {
		c := s[i]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

// record is one JSONL log line.
type record struct {
	Seq                uint64        `json:"seq"`
	Op                 string        `json:"op"` // "reserve" | "advance"
	OperationID        string        `json:"operationId"`
	ArgumentsDigest    string        `json:"argumentsDigest,omitempty"`
	Owner              *Owner        `json:"owner,omitempty"`
	Path               *PathIdentity `json:"path,omitempty"`
	LeaseID            string        `json:"leaseId,omitempty"`
	LeaseExpiresWallMs int64         `json:"leaseExpiresWallMs,omitempty"`
	State              State         `json:"state,omitempty"`
	Partial            []string      `json:"partial,omitempty"`
}

// Journal is a durable outside-workspace reservation/ownership ledger.
// Intent is appended and fsynced before Reserve/Advance return, so a crash
// between reservation and the first effect stays discoverable on reopen.
//
// The zero handle is unusable; construct with OpenDir or OpenMemory.
// A Journal is safe for concurrent use.
type Journal struct {
	mu       sync.Mutex
	ops      map[string]*Reservation
	seq      uint64
	file     *os.File // nil for memory-only journals
	closed   bool
	failSync bool // fault point: fail the next durable append (tests only)
}

// OpenMemory returns a non-durable journal with the same state machine.
// It offers no crash discoverability.
func OpenMemory() *Journal {
	return &Journal{ops: make(map[string]*Reservation)}
}

// OpenDir opens (or creates) a durable journal in dir, which must live
// outside every managed workspace. The log is replayed; Pending then reports
// every operation left non-terminal by a crash.
func OpenDir(dir string) (*Journal, error) {
	if dir == "" {
		return nil, fmt.Errorf("%w: empty journal directory", ErrInvalid)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("journal: create directory: %w", err)
	}
	p := filepath.Join(dir, logFileName)
	f, err := os.OpenFile(p, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("journal: open log: %w", err)
	}
	j := &Journal{ops: make(map[string]*Reservation), file: f}
	if err := j.replay(); err != nil {
		f.Close()
		return nil, err
	}
	return j, nil
}

// replay loads the log. A torn final line (crash mid-write, no trailing
// newline) is truncated; any other malformed record fails closed.
func (j *Journal) replay() error {
	fi, err := j.file.Stat()
	if err != nil {
		return fmt.Errorf("journal: stat log: %w", err)
	}
	if fi.Size() > MaxLogBytes {
		return fmt.Errorf("%w: log exceeds transitional bound", ErrCorrupt)
	}
	if _, err := j.file.Seek(0, 0); err != nil {
		return fmt.Errorf("journal: seek log: %w", err)
	}
	// The log is small and bounded; read it fully.
	buf := make([]byte, fi.Size())
	n := 0
	for n < len(buf) {
		m, rerr := j.file.Read(buf[n:])
		n += m
		if rerr != nil {
			return fmt.Errorf("journal: read log: %w", rerr)
		}
	}
	buf = buf[:n]

	endsNL := len(buf) > 0 && buf[len(buf)-1] == '\n'
	start := 0
	for start < len(buf) {
		end := start
		for end < len(buf) && buf[end] != '\n' {
			end++
		}
		line := buf[start:end]
		isLast := end == len(buf)
		if len(line) == 0 {
			start = end + 1
			continue
		}
		var rec record
		if err := json.Unmarshal(line, &rec); err != nil {
			if isLast && !endsNL {
				// Torn tail from a crash mid-write: truncate and continue.
				if terr := j.file.Truncate(int64(start)); terr != nil {
					return fmt.Errorf("journal: truncate torn tail: %w", terr)
				}
				break
			}
			return fmt.Errorf("%w: line at offset %d: %v", ErrCorrupt, start, err)
		}
		if err := j.apply(rec); err != nil {
			return err
		}
		start = end + 1
	}
	return nil
}

// apply folds one replayed record into memory. Caller holds no lock (open
// path only); every anomaly fails closed.
func (j *Journal) apply(rec record) error {
	if !validID(rec.OperationID) || rec.Seq == 0 {
		return fmt.Errorf("%w: malformed record identity", ErrCorrupt)
	}
	switch rec.Op {
	case "reserve":
		if !validDigest(rec.ArgumentsDigest) || rec.Owner == nil || rec.Path == nil {
			return fmt.Errorf("%w: malformed reserve record", ErrCorrupt)
		}
		if prev, dup := j.ops[rec.OperationID]; dup {
			if prev.ArgumentsDigest != rec.ArgumentsDigest {
				return fmt.Errorf("%w: log replays conflicting digests", ErrCorrupt)
			}
			if rec.Seq > j.seq {
				j.seq = rec.Seq
			}
			return nil
		}
		j.ops[rec.OperationID] = &Reservation{
			OperationID:        rec.OperationID,
			ArgumentsDigest:    rec.ArgumentsDigest,
			Owner:              *rec.Owner,
			Path:               *rec.Path,
			LeaseID:            rec.LeaseID,
			LeaseExpiresWallMs: rec.LeaseExpiresWallMs,
			State:              StateReserved,
			Seq:                rec.Seq,
		}
	case "advance":
		cur, ok := j.ops[rec.OperationID]
		if !ok {
			return fmt.Errorf("%w: advance of unknown operation", ErrCorrupt)
		}
		if !knownState(rec.State) || !legalAdvance(cur.State, rec.State) {
			return fmt.Errorf("%w: illegal replayed transition %s->%s", ErrCorrupt, cur.State, rec.State)
		}
		cur.State = rec.State
		cur.Partial = append(cur.Partial, rec.Partial...)
	default:
		return fmt.Errorf("%w: unknown record op %q", ErrCorrupt, rec.Op)
	}
	if rec.Seq > j.seq {
		j.seq = rec.Seq
	}
	return nil
}

// appendLocked marshals, appends and fsyncs one record. The fault point fires
// before any backend write so tests can observe a failed reservation.
func (j *Journal) appendLocked(rec record) error {
	if j.failSync {
		j.failSync = false
		return errSyncFault
	}
	if j.file == nil {
		return nil
	}
	line, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("journal: encode record: %w", err)
	}
	line = append(line, '\n')
	if _, err := j.file.Write(line); err != nil {
		return fmt.Errorf("journal: append record: %w", err)
	}
	if err := j.file.Sync(); err != nil {
		return fmt.Errorf("journal: sync record: %w", err)
	}
	return nil
}

// Reserve registers intent before the first externally visible effect.
// Repeating the same operation ID with the same argument digest rejoins the
// existing reservation without a new effect; the same ID with a different
// digest is rejected with ErrConflict.
func (j *Journal) Reserve(opID, digest string, owner Owner, path PathIdentity, leaseID string, leaseExpiresWallMs int64) (*Reservation, error) {
	if !validID(opID) {
		return nil, fmt.Errorf("%w: malformed operation ID", ErrInvalid)
	}
	if !validDigest(digest) {
		return nil, fmt.Errorf("%w: malformed argument digest", ErrInvalid)
	}
	if owner.PID <= 0 || owner.StartToken == "" || len(owner.StartToken) > MaxStartTokenLen {
		return nil, fmt.Errorf("%w: owner needs positive PID and spawn-start token", ErrInvalid)
	}
	if path.Path != "" && !filepath.IsAbs(path.Path) {
		return nil, fmt.Errorf("%w: path identity must be absolute", ErrInvalid)
	}
	if len(leaseID) > MaxLeaseIDLen {
		return nil, fmt.Errorf("%w: lease ID too long", ErrInvalid)
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.closed {
		return nil, ErrClosed
	}
	if prev, dup := j.ops[opID]; dup {
		if prev.ArgumentsDigest != digest {
			return nil, fmt.Errorf("%w: operation %q replayed with different arguments", ErrConflict, opID)
		}
		return prev.copy(), nil
	}
	if len(j.ops) >= MaxOperations {
		return nil, fmt.Errorf("%w: journal full", ErrInvalid)
	}
	rec := record{
		Seq: j.seq + 1, Op: "reserve",
		OperationID: opID, ArgumentsDigest: digest,
		Owner:   &Owner{PID: owner.PID, StartToken: owner.StartToken},
		Path:    &PathIdentity{Path: path.Path, Dev: path.Dev, Ino: path.Ino, HasFileID: path.HasFileID},
		LeaseID: leaseID, LeaseExpiresWallMs: leaseExpiresWallMs,
	}
	if err := j.appendLocked(rec); err != nil {
		return nil, err
	}
	j.seq = rec.Seq
	r := &Reservation{
		OperationID: opID, ArgumentsDigest: digest,
		Owner: *rec.Owner, Path: *rec.Path,
		LeaseID: leaseID, LeaseExpiresWallMs: leaseExpiresWallMs,
		State: StateReserved, Seq: rec.Seq,
	}
	j.ops[opID] = r
	return r.copy(), nil
}

// Advance moves an operation forward along the monotonic lifecycle and
// records newly acquired partial effects. Repeating the current state is an
// idempotent no-op; backward moves fail with ErrTransition.
func (j *Journal) Advance(opID string, to State, partials ...string) (*Reservation, error) {
	if !knownState(to) {
		return nil, fmt.Errorf("%w: unknown state %q", ErrInvalid, to)
	}
	for _, p := range partials {
		if len(p) == 0 || len(p) > MaxPartialLen {
			return nil, fmt.Errorf("%w: malformed partial label", ErrInvalid)
		}
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.closed {
		return nil, ErrClosed
	}
	cur, ok := j.ops[opID]
	if !ok {
		return nil, fmt.Errorf("%w: operation %q", ErrNotFound, opID)
	}
	if cur.State == to {
		return cur.copy(), nil
	}
	if !legalAdvance(cur.State, to) {
		return nil, fmt.Errorf("%w: %s->%s", ErrTransition, cur.State, to)
	}
	if len(cur.Partial)+len(partials) > MaxPartialsPerOperation {
		return nil, fmt.Errorf("%w: too many partial acquisitions", ErrInvalid)
	}
	rec := record{Seq: j.seq + 1, Op: "advance", OperationID: opID, State: to, Partial: partials}
	if err := j.appendLocked(rec); err != nil {
		return nil, err
	}
	j.seq = rec.Seq
	cur.State = to
	cur.Partial = append(cur.Partial, partials...)
	return cur.copy(), nil
}

// Lookup returns a copy of one reservation. It reports false for unknown IDs.
func (j *Journal) Lookup(opID string) (*Reservation, bool) {
	j.mu.Lock()
	defer j.mu.Unlock()
	cur, ok := j.ops[opID]
	if !ok {
		return nil, false
	}
	return cur.copy(), true
}

// Pending reports every non-terminal operation in reservation order, so a
// crash between reservation and marker creation stays discoverable.
func (j *Journal) Pending() []Reservation {
	j.mu.Lock()
	defer j.mu.Unlock()
	var out []Reservation
	for _, r := range j.ops {
		if !terminal(r.State) {
			out = append(out, *r.copy())
		}
	}
	sort.Slice(out, func(i, k int) bool { return out[i].Seq < out[k].Seq })
	return out
}

// VerifyAuthority refuses callers whose owner or path identity does not match
// the reservation: wrong PID, wrong spawn-start token, wrong path, a path
// replaced by a different object, or an identity that can no longer be
// proven all fail. Printed IDs and PIDs alone are never authority.
func (j *Journal) VerifyAuthority(opID string, owner Owner, path string) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.closed {
		return ErrClosed
	}
	cur, ok := j.ops[opID]
	if !ok {
		return fmt.Errorf("%w: operation %q", ErrNotFound, opID)
	}
	if owner.PID != cur.Owner.PID || owner.StartToken != cur.Owner.StartToken {
		return fmt.Errorf("%w: owner PID/start-token mismatch", ErrWrongOwner)
	}
	want := cur.Path.Path
	if want == "" {
		if path != "" {
			return fmt.Errorf("%w: operation has no path", ErrWrongPath)
		}
		return nil
	}
	if path != want {
		return fmt.Errorf("%w: path mismatch", ErrWrongPath)
	}
	if !cur.Path.HasFileID {
		return nil
	}
	fi, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("%w: cannot stat reserved path: %v", ErrWrongPath, err)
	}
	dev, ino, ok := fileID(fi)
	if !ok || dev != cur.Path.Dev || ino != cur.Path.Ino {
		return fmt.Errorf("%w: path replaced or identity unavailable", ErrWrongPath)
	}
	return nil
}

// StatPath captures the path identity to register before the effect. A path
// whose marker does not exist yet records path-only identity (HasFileID
// false); once the marker exists, device/inode bind the exact object.
func StatPath(path string) (PathIdentity, error) {
	if path == "" || !filepath.IsAbs(path) {
		return PathIdentity{}, fmt.Errorf("%w: path identity must be absolute", ErrInvalid)
	}
	fi, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return PathIdentity{Path: path}, nil
		}
		return PathIdentity{}, fmt.Errorf("journal: stat path: %w", err)
	}
	dev, ino, ok := fileID(fi)
	if !ok {
		return PathIdentity{Path: path}, nil
	}
	return PathIdentity{Path: path, Dev: dev, Ino: ino, HasFileID: true}, nil
}

// fileID extracts device/inode where the platform exposes them.
func fileID(fi os.FileInfo) (uint64, uint64, bool) {
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok || st == nil {
		return 0, 0, false
	}
	v := reflect.ValueOf(st).Elem()
	dev, ok := statUint(v.FieldByName("Dev"))
	if !ok {
		return 0, 0, false
	}
	ino, ok := statUint(v.FieldByName("Ino"))
	if !ok {
		return 0, 0, false
	}
	return dev, ino, true
}

func statUint(v reflect.Value) (uint64, bool) {
	if !v.IsValid() {
		return 0, false
	}
	switch v.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint(), true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if v.Int() < 0 {
			return 0, false
		}
		return uint64(v.Int()), true
	}
	return 0, false
}

// Close syncs and releases the journal. Mutations and authority checks fail
// after Close; Lookup and Pending remain readable.
func (j *Journal) Close() error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.closed {
		return nil
	}
	j.closed = true
	if j.file == nil {
		return nil
	}
	serr := j.file.Sync()
	cerr := j.file.Close()
	j.file = nil
	if serr != nil {
		return fmt.Errorf("journal: sync on close: %w", serr)
	}
	if cerr != nil {
		return fmt.Errorf("journal: close: %w", cerr)
	}
	return nil
}
