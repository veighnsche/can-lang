package workspace

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	journal "github.com/veighnsche/can-lang/tools/native-test-owner/journal"
)

// Create/replace rules for Materialize and Write.
const (
	// ModeCreate fails when the destination already exists.
	ModeCreate = "create"
	// ModeReplace overwrites an existing regular file or creates it.
	ModeReplace = "replace"
	// ModeCompareRevision overwrites only when the workspace revision
	// matches the expected value (Write only).
	ModeCompareRevision = "compare"
)

// MaterialEntry is one selected byte entry to import. Imports carry bytes
// only: link entries are rejected, and any link met during creation is an
// error, never traversed.
type MaterialEntry struct {
	Path string `json:"path"`
	Data []byte `json:"data"`
}

// EntryOutcome is the per-entry result of a multi-file publication. Partial
// publication is explicit: entries before the failure are retained and
// charged, entries after it are not attempted.
type EntryOutcome struct {
	Path   string `json:"path"`
	Digest string `json:"digest,omitempty"`
	Error  string `json:"error,omitempty"`
}

// MaterializeResult is the terminal fact of one materialize operation.
type MaterializeResult struct {
	Revision uint64         `json:"revision"`
	Outcomes []EntryOutcome `json:"outcomes"`
	Complete bool           `json:"complete"`
	Joined   bool           `json:"joined,omitempty"`
}

// Materialize imports only the selected entries into the workspace,
// enforcing budgets before retention and verifying written bytes. A mid-way
// failure keeps the landed entries owned and charged and reports per-entry
// outcomes; it never rolls back silently and never claims completeness.
func (o *Owner) Materialize(opID string, h Handle, entries []MaterialEntry, mode string) (*MaterializeResult, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	s, err := o.lookupLocked(h, true)
	if err != nil {
		return nil, err
	}
	if mode != ModeCreate && mode != ModeReplace {
		return nil, fmt.Errorf("%w: unknown mode %q", ErrInvalid, mode)
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("%w: no entries selected", ErrInvalid)
	}
	type argEntry struct {
		Path   string `json:"path"`
		Digest string `json:"digest"`
	}
	argList := make([]argEntry, len(entries))
	partsList := make([][]string, len(entries))
	for i, e := range entries {
		p, verr := splitRel(e.Path)
		if verr != nil {
			return nil, verr
		}
		if len(p) == 0 {
			return nil, fmt.Errorf("%w: cannot materialize the root", ErrInvalid)
		}
		if sealedLocked(s, joinRel(p)) {
			return nil, fmt.Errorf("%w: %q", ErrSealed, e.Path)
		}
		partsList[i] = p
		argList[i] = argEntry{Path: joinRel(p), Digest: digestBytes(e.Data)}
	}
	args := struct {
		Space   string     `json:"space"`
		Mode    string     `json:"mode"`
		Entries []argEntry `json:"entries"`
	}{Space: h.SpaceID, Mode: mode, Entries: argList}
	joined, err := o.journalIntent(opID, args, journal.PathIdentity{Path: s.root}, "", 0)
	if err != nil {
		return nil, err
	}
	if joined {
		v, rerr := o.rejoined(opID)
		if v == nil {
			return nil, rerr
		}
		res := *(v.(*MaterializeResult))
		res.Joined = true
		return &res, rerr
	}

	// Enforce budgets before retention: prospective new bytes, entries
	// (including parent directories to create) and depth.
	var newBytes int64
	var newEntries int
	maxDepth := s.maxDepth
	parentsNeeded := map[string]bool{}
	for i, e := range entries {
		depth := len(partsList[i])
		if depth > s.budgets.MaxDepth {
			return nil, o.reject(opID, fmt.Errorf("%w: %q exceeds depth %d", ErrBudget, e.Path, s.budgets.MaxDepth), "depth: "+e.Path)
		}
		if depth > maxDepth {
			maxDepth = depth
		}
		for k := 1; k < len(partsList[i]); k++ {
			parentsNeeded[joinRel(partsList[i][:k])] = true
		}
		rel := joinRel(partsList[i])
		fi, lerr := s.r.Lstat(rel)
		if lerr != nil && !os.IsNotExist(lerr) {
			return nil, o.reject(opID, fmt.Errorf("workspace: lstat %q: %w", rel, lerr), "lstat: "+rel)
		}
		switch {
		case os.IsNotExist(lerr):
			newEntries++
			newBytes += int64(len(e.Data))
		case fi.Mode()&os.ModeSymlink != 0:
			return nil, o.reject(opID, fmt.Errorf("%w: %q is a symlink; imports never traverse links", ErrWrongKind, rel), "link: "+rel)
		case fi.IsDir():
			return nil, o.reject(opID, fmt.Errorf("%w: %q is a directory", ErrWrongKind, rel), "dir: "+rel)
		default:
			if mode == ModeCreate {
				return nil, o.reject(opID, fmt.Errorf("%w: %q already exists", ErrCollision, rel), "exists: "+rel)
			}
			newBytes += int64(len(e.Data)) - fi.Size()
		}
	}
	for p := range parentsNeeded {
		if _, lerr := s.r.Lstat(p); os.IsNotExist(lerr) {
			newEntries++
		}
	}
	if s.chargedBytes+newBytes > s.budgets.MaxBytes {
		return nil, o.reject(opID, fmt.Errorf("%w: materialize needs %d bytes, %d charged of %d", ErrBudget, newBytes, s.chargedBytes, s.budgets.MaxBytes), "bytes")
	}
	if s.chargedEntries+newEntries > s.budgets.MaxEntries {
		return nil, o.reject(opID, fmt.Errorf("%w: materialize needs %d entries, %d charged of %d", ErrBudget, newEntries, s.chargedEntries, s.budgets.MaxEntries), "entries")
	}

	// Apply in order. A failure keeps landed entries and charges.
	res := &MaterializeResult{Outcomes: make([]EntryOutcome, 0, len(entries))}
	var partials []string
	for i, e := range entries {
		rel := joinRel(partsList[i])
		if err := o.mkdirParentsLocked(s, partsList[i]); err != nil {
			out := EntryOutcome{Path: rel, Error: err.Error()}
			res.Outcomes = append(res.Outcomes, out)
			partials = append(partials, "mkdir: "+rel)
			o.failOp(opID, append(partials, landedLocked(s, res))...)
			o.remember(opID, res, err)
			return res, err
		}
		werr := o.writeBytesLocked(s, rel, e.Data, mode)
		out := EntryOutcome{Path: rel}
		if werr != nil {
			out.Error = werr.Error()
			res.Outcomes = append(res.Outcomes, out)
			partials = append(partials, "write: "+rel)
			o.failOp(opID, append(partials, landedLocked(s, res))...)
			o.remember(opID, res, werr)
			return res, werr
		}
		// Verify bytes before accounting them.
		back, rerr := s.r.ReadFile(rel)
		if rerr != nil {
			out.Error = rerr.Error()
			res.Outcomes = append(res.Outcomes, out)
			o.failOp(opID, append(partials, "verify: "+rel)...)
			verr := fmt.Errorf("workspace: verify %q: %w", rel, rerr)
			o.remember(opID, res, verr)
			return res, verr
		}
		got := digestBytes(back)
		want := digestBytes(e.Data)
		if got != want {
			out.Error = "verify digest mismatch"
			res.Outcomes = append(res.Outcomes, out)
			o.failOp(opID, append(partials, "verify: "+rel)...)
			merr := fmt.Errorf("%w: %q failed verification", ErrChanged, rel)
			o.remember(opID, res, merr)
			return res, merr
		}
		out.Digest = got
		res.Outcomes = append(res.Outcomes, out)
		s.tracked[rel] = trackedEntry{kind: KindFile, size: int64(len(back)), digest: got}
	}
	s.rev++
	s.maxDepth = maxDepth
	res.Revision = s.rev
	res.Complete = true
	o.closeOp(opID)
	o.remember(opID, res, nil)
	out := *res
	return &out, nil
}

// landedLocked summarizes retained entries for partial-failure partials.
func landedLocked(s *space, res *MaterializeResult) string {
	n := 0
	for _, oc := range res.Outcomes {
		if oc.Error == "" {
			n++
		}
	}
	return fmt.Sprintf("landed %d entries, still charged", n)
}

// mkdirParentsLocked creates missing parent directories, refusing to
// traverse symlinks. It charges each created directory.
func (o *Owner) mkdirParentsLocked(s *space, parts []string) error {
	for k := 1; k < len(parts); k++ {
		p := joinRel(parts[:k])
		fi, err := s.r.Lstat(p)
		if err == nil {
			if fi.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("%w: %q traverses a symlink", ErrEscape, joinRel(parts))
			}
			if !fi.IsDir() {
				return fmt.Errorf("%w: %q is not a directory", ErrWrongKind, p)
			}
			continue
		}
		if !os.IsNotExist(err) {
			return fmt.Errorf("workspace: lstat %q: %w", p, err)
		}
		if err := s.r.Mkdir(p, 0o700); err != nil {
			if errors.Is(err, os.ErrExist) {
				continue
			}
			return fmt.Errorf("workspace: mkdir %q: %w", p, err)
		}
		s.chargedEntries++
		if _, ok := s.tracked[p]; !ok {
			s.tracked[p] = trackedEntry{kind: KindDir}
		}
	}
	return nil
}

// writeBytesLocked writes one file and adjusts byte/entry charges.
// Callers enforce budgets and seals first.
func (o *Owner) writeBytesLocked(s *space, rel string, data []byte, mode string) error {
	existed := false
	if fi, err := s.r.Lstat(rel); err == nil {
		existed = true
		if fi.Mode()&os.ModeSymlink != 0 || fi.IsDir() {
			return fmt.Errorf("%w: %q", ErrWrongKind, rel)
		}
		s.chargedBytes -= fi.Size()
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("workspace: lstat %q: %w", rel, err)
	}
	flag := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	if mode == ModeCreate {
		flag = os.O_WRONLY | os.O_CREATE | os.O_EXCL
	}
	f, err := s.r.OpenFile(rel, flag, 0o600)
	if err != nil {
		o.restoreBytesLocked(s, rel, existed)
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("%w: %q already exists", ErrCollision, rel)
		}
		return fmt.Errorf("workspace: open %q: %w", rel, err)
	}
	werr := func() error {
		if len(data) > 0 {
			if _, err := f.Write(data); err != nil {
				return err
			}
		}
		if o.failSync != nil {
			return o.failSync
		}
		return f.Sync()
	}()
	cerr := f.Close()
	if werr != nil {
		o.restoreBytesLocked(s, rel, existed)
		return fmt.Errorf("workspace: write %q: %w", rel, werr)
	}
	if cerr != nil {
		o.restoreBytesLocked(s, rel, existed)
		return fmt.Errorf("workspace: close %q: %w", rel, cerr)
	}
	if !existed {
		s.chargedEntries++
	}
	s.chargedBytes += int64(len(data))
	return nil
}

// restoreBytesLocked re-charges the visible bytes after a failed write so a
// truncated or partial file stays charged instead of leaking its bytes.
func (o *Owner) restoreBytesLocked(s *space, rel string, existed bool) {
	if !existed {
		return
	}
	if fi, serr := s.r.Lstat(rel); serr == nil {
		s.chargedBytes += fi.Size()
	}
}

// ReadResult is the observed fact of one read.
type ReadResult struct {
	Data      []byte    `json:"-"`
	Kind      Kind      `json:"kind"`
	Size      int64     `json:"size"`
	Digest    string    `json:"digest,omitempty"`
	Revision  uint64    `json:"revision"`
	Stability Stability `json:"stability"`
	Truncated bool      `json:"truncated"`
	Offset    int64     `json:"offset"`
}

// Read observes bytes without following symlinks. Absent paths, escapes,
// wrong kinds, detected changes, truncation and I/O failures are
// distinguished in the result and error. A required qualified_snapshot
// strength is always rejected: this owner cannot provide it.
func (o *Owner) Read(h Handle, rel string, offset int64, limit int, wantRev *uint64, wantStability Stability) (*ReadResult, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	s, err := o.lookupLocked(h, false)
	if err != nil {
		return nil, err
	}
	if wantStability == StabilityQualifiedSnapshot {
		return nil, fmt.Errorf("%w: qualified_snapshot is not provided", ErrStrength)
	}
	parts, err := splitRel(rel)
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("%w: cannot read the root as a file", ErrWrongKind)
	}
	if _, err := checkParentsLocked(s, parts); err != nil {
		return nil, err
	}
	if offset < 0 || limit < 0 || limit > MaxReadBytes {
		return nil, fmt.Errorf("%w: offset/limit out of range", ErrInvalid)
	}
	name := joinRel(parts)
	fi, err := s.r.Lstat(name)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %q", ErrNotFound, rel)
		}
		return nil, fmt.Errorf("workspace: lstat %q: %w", rel, err)
	}
	kind := kindOf(fi)
	if kind != KindFile {
		res := &ReadResult{Kind: kind, Size: fi.Size(), Revision: s.rev, Stability: o.stabilityLocked(s, name, fi)}
		return res, fmt.Errorf("%w: %q is %s", ErrWrongKind, rel, kind)
	}
	f, err := s.r.OpenFile(name, os.O_RDONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("workspace: open %q: %w", rel, err)
	}
	size := fi.Size()
	data := make([]byte, 0)
	truncated := false
	if offset < size {
		if limit == 0 {
			truncated = size > offset
		} else {
			sect := io.NewSectionReader(f, offset, size-offset)
			buf := make([]byte, limit)
			n, rerr := io.ReadFull(sect, buf)
			data = buf[:n]
			if rerr == nil {
				// Limit exhausted while bytes remain.
				truncated = true
			} else if rerr != io.ErrUnexpectedEOF && rerr != io.EOF {
				f.Close()
				return nil, fmt.Errorf("workspace: read %q: %w", rel, rerr)
			}
		}
	} else if offset > size {
		f.Close()
		return nil, fmt.Errorf("%w: offset %d beyond size %d", ErrInvalid, offset, size)
	}
	// Content identity covers the whole file, not just the returned slice.
	if _, err := f.Seek(0, 0); err != nil {
		f.Close()
		return nil, fmt.Errorf("workspace: seek %q: %w", rel, err)
	}
	sum := sha256.New()
	if _, err := io.Copy(sum, f); err != nil {
		f.Close()
		return nil, fmt.Errorf("workspace: digest %q: %w", rel, err)
	}
	f.Close()
	digest := "sha256:" + hex.EncodeToString(sum.Sum(nil))
	stability := StabilityUnknown
	if t, ok := s.tracked[name]; ok && t.kind == KindFile {
		if t.digest == digest && t.size == size {
			stability = StabilityScopedRevision
		} else {
			stability = StabilityChangedObserved
		}
	}
	res := &ReadResult{Data: data, Kind: KindFile, Size: size, Digest: digest, Revision: s.rev, Stability: stability, Truncated: truncated, Offset: offset}
	if wantRev != nil && *wantRev != s.rev {
		return res, fmt.Errorf("%w: want revision %d, have %d", ErrChanged, *wantRev, s.rev)
	}
	if wantStability == StabilityScopedRevision && stability != StabilityScopedRevision {
		return res, fmt.Errorf("%w: have %s", ErrChanged, stability)
	}
	return res, nil
}

func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// stabilityLocked classifies a Lstat'd node against tracked state without
// following symlinks or reading contents (size/identity based).
func (o *Owner) stabilityLocked(s *space, name string, fi os.FileInfo) Stability {
	t, ok := s.tracked[name]
	if !ok || t.kind != kindOf(fi) {
		return StabilityUnknown
	}
	if t.kind == KindFile && t.size != fi.Size() {
		return StabilityChangedObserved
	}
	if t.kind == KindLink {
		if cur, err := s.r.Readlink(name); err != nil || cur != t.target {
			return StabilityChangedObserved
		}
	}
	return StabilityScopedRevision
}

// NodeInfo is the no-follow identity and metadata of one node.
type NodeInfo struct {
	Kind      Kind      `json:"kind"`
	Size      int64     `json:"size"`
	Target    string    `json:"target,omitempty"`
	Revision  uint64    `json:"revision"`
	Stability Stability `json:"stability"`
}

// Stat inspects one node without following symlinks.
func (o *Owner) Stat(h Handle, rel string) (*NodeInfo, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	s, err := o.lookupLocked(h, false)
	if err != nil {
		return nil, err
	}
	parts, err := splitRel(rel)
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return &NodeInfo{Kind: KindDir, Revision: s.rev, Stability: StabilityScopedRevision}, nil
	}
	if _, err := checkParentsLocked(s, parts); err != nil {
		return nil, err
	}
	name := joinRel(parts)
	fi, err := s.r.Lstat(name)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %q", ErrNotFound, rel)
		}
		return nil, fmt.Errorf("workspace: lstat %q: %w", rel, err)
	}
	info := &NodeInfo{Kind: kindOf(fi), Size: fi.Size(), Revision: s.rev}
	if info.Kind == KindLink {
		t, rerr := s.r.Readlink(name)
		if rerr != nil {
			return nil, fmt.Errorf("workspace: readlink %q: %w", rel, rerr)
		}
		info.Target = t
	}
	info.Stability = o.stabilityLocked(s, name, fi)
	return info, nil
}

// ListCursor pages a revision-bound directory listing.
type ListCursor struct {
	Revision uint64 `json:"revision"`
	Offset   int    `json:"offset"`
}

// ListEntry is one directory member observed without following symlinks.
type ListEntry struct {
	Name      string    `json:"name"`
	Kind      Kind      `json:"kind"`
	Size      int64     `json:"size"`
	Stability Stability `json:"stability"`
}

// ListPage is one bounded page. Capability-mediated mutation invalidates
// cursors; unknown external mutation cannot produce a claimed complete
// inventory, so Complete is true only under scoped_revision stability.
type ListPage struct {
	Entries    []ListEntry `json:"entries"`
	Next       ListCursor  `json:"next"`
	Revision   uint64      `json:"revision"`
	Stability  Stability   `json:"stability"`
	Complete   bool        `json:"complete"`
	TotalCount int         `json:"totalCount"`
}

// List returns a bounded revision-bound page of one directory. Symlinks are
// reported as links, never traversed.
func (o *Owner) List(h Handle, rel string, cur ListCursor, limit int) (*ListPage, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	s, err := o.lookupLocked(h, false)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > MaxListPage {
		return nil, fmt.Errorf("%w: page limit out of range", ErrInvalid)
	}
	if cur.Offset < 0 {
		return nil, fmt.Errorf("%w: negative page offset", ErrInvalid)
	}
	parts, err := splitRel(rel)
	if err != nil {
		return nil, err
	}
	if len(parts) > 0 {
		if _, err := checkParentsLocked(s, parts); err != nil {
			return nil, err
		}
	}
	if cur.Revision != s.rev {
		return nil, fmt.Errorf("%w: cursor revision %d, workspace %d", ErrStaleCursor, cur.Revision, s.rev)
	}
	name := joinRel(parts)
	if name == "" {
		name = "."
	}
	fi, err := s.r.Lstat(name)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %q", ErrNotFound, rel)
		}
		return nil, fmt.Errorf("workspace: lstat %q: %w", rel, err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("%w: %q is a symlink", ErrWrongKind, rel)
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("%w: %q is not a directory", ErrWrongKind, rel)
	}
	f, err := s.r.Open(name)
	if err != nil {
		return nil, fmt.Errorf("workspace: open dir %q: %w", rel, err)
	}
	dirents, rerr := f.ReadDir(-1)
	cerr := f.Close()
	if rerr != nil {
		return nil, fmt.Errorf("workspace: read dir %q: %w", rel, rerr)
	}
	if cerr != nil {
		return nil, fmt.Errorf("workspace: close dir %q: %w", rel, cerr)
	}
	sort.Slice(dirents, func(i, j int) bool { return dirents[i].Name() < dirents[j].Name() })
	total := len(dirents)
	if cur.Offset > total {
		return nil, fmt.Errorf("%w: offset beyond %d entries", ErrInvalid, total)
	}
	end := cur.Offset + limit
	if end > total {
		end = total
	}
	base := joinRel(parts)
	page := &ListPage{Revision: s.rev, TotalCount: total, Stability: StabilityScopedRevision}
	for _, de := range dirents[cur.Offset:end] {
		le := ListEntry{Name: de.Name()}
		typ := de.Type()
		switch {
		case typ&os.ModeSymlink != 0:
			le.Kind = KindLink
		case de.IsDir():
			le.Kind = KindDir
		default:
			le.Kind = KindFile
		}
		if le.Kind != KindLink {
			if info, ierr := de.Info(); ierr == nil {
				le.Size = info.Size()
			}
		}
		child := de.Name()
		if base != "" {
			child = base + "/" + de.Name()
		}
		le.Stability = StabilityUnknown
		if t, ok := s.tracked[child]; ok && t.kind == le.Kind {
			if le.Kind != KindFile || t.size == le.Size {
				le.Stability = StabilityScopedRevision
			} else {
				le.Stability = StabilityChangedObserved
			}
		} else if ok {
			le.Stability = StabilityChangedObserved
		}
		page.Entries = append(page.Entries, le)
	}
	for _, le := range page.Entries {
		switch le.Stability {
		case StabilityUnknown:
			page.Stability = StabilityUnknown
		case StabilityChangedObserved:
			if page.Stability == StabilityScopedRevision {
				page.Stability = StabilityChangedObserved
			}
		}
		if page.Stability == StabilityUnknown {
			break
		}
	}
	page.Next = ListCursor{Revision: s.rev, Offset: end}
	page.Complete = end == total && page.Stability == StabilityScopedRevision
	return page, nil
}

// WriteResult is the terminal fact of one atomic single-file publish.
type WriteResult struct {
	PriorRevision uint64 `json:"priorRevision"`
	Revision      uint64 `json:"revision"`
	Digest        string `json:"digest"`
	Joined        bool   `json:"joined,omitempty"`
}

// Write stages and publishes one file. New files are created exclusively;
// replacements are staged (staging bytes are charged too) and renamed
// atomically. There is no implied multi-file transaction. Failures report
// the visible state and leave prior content and charges intact.
func (o *Owner) Write(opID string, h Handle, rel string, data []byte, mode string, expectRev uint64) (*WriteResult, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	s, err := o.lookupLocked(h, true)
	if err != nil {
		return nil, err
	}
	if mode != ModeCreate && mode != ModeReplace && mode != ModeCompareRevision {
		return nil, fmt.Errorf("%w: unknown mode %q", ErrInvalid, mode)
	}
	parts, err := splitRel(rel)
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("%w: cannot write the root", ErrInvalid)
	}
	name := joinRel(parts)
	if sealedLocked(s, name) {
		return nil, fmt.Errorf("%w: %q", ErrSealed, rel)
	}
	if len(parts) > s.budgets.MaxDepth {
		return nil, fmt.Errorf("%w: %q exceeds depth %d", ErrBudget, rel, s.budgets.MaxDepth)
	}
	if _, err := checkParentsLocked(s, parts); err != nil {
		// Parents may be missing; creation happens after intent below,
		// but a symlink in the chain already rejects here. Missing
		// parents are only an error for the Lstat check when they are
		// neither creatable... re-check after intent; for now only fail
		// on escape/kind errors.
		if !errors.Is(err, ErrNotFound) {
			return nil, err
		}
	}
	args := struct {
		Space  string `json:"space"`
		Path   string `json:"path"`
		Digest string `json:"digest"`
		Mode   string `json:"mode"`
		Expect uint64 `json:"expect"`
	}{Space: h.SpaceID, Path: name, Digest: digestBytes(data), Mode: mode, Expect: expectRev}
	joined, err := o.journalIntent(opID, args, journal.PathIdentity{Path: s.root}, "", 0)
	if err != nil {
		return nil, err
	}
	if joined {
		v, rerr := o.rejoined(opID)
		if v == nil {
			return nil, rerr
		}
		res := *(v.(*WriteResult))
		res.Joined = true
		return &res, rerr
	}
	if mode == ModeCompareRevision && s.rev != expectRev {
		return nil, o.reject(opID, fmt.Errorf("%w: want revision %d, have %d", ErrChanged, expectRev, s.rev), "compare revision")
	}
	priorRev := s.rev
	fi, lerr := s.r.Lstat(name)
	exists := lerr == nil
	if lerr != nil && !os.IsNotExist(lerr) {
		return nil, o.reject(opID, fmt.Errorf("workspace: lstat %q: %w", rel, lerr), "lstat")
	}
	if exists {
		if fi.Mode()&os.ModeSymlink != 0 || fi.IsDir() {
			return nil, o.reject(opID, fmt.Errorf("%w: %q", ErrWrongKind, rel), "kind")
		}
		if mode == ModeCreate {
			return nil, o.reject(opID, fmt.Errorf("%w: %q already exists", ErrCollision, rel), "exists")
		}
	} else if mode == ModeCompareRevision {
		// Compare-revision on an absent file: revision matched but there
		// is nothing to replace.
		return nil, o.reject(opID, fmt.Errorf("%w: %q", ErrNotFound, rel), "absent")
	}
	// Budget check before retention. Staging is charged too: a replacement
	// transiently holds old bytes plus the staged copy.
	extraEntries := 0
	if !exists {
		extraEntries = 1
		for k := 1; k < len(parts); k++ {
			if _, perr := s.r.Lstat(joinRel(parts[:k])); os.IsNotExist(perr) {
				extraEntries++
			}
		}
	}
	var finalBytes int64
	oldSize := int64(0)
	if exists {
		oldSize = fi.Size()
		finalBytes = s.chargedBytes - oldSize + int64(len(data))
		if s.chargedBytes+int64(len(data)) > s.budgets.MaxBytes {
			return nil, o.reject(opID, fmt.Errorf("%w: staging %d bytes exceeds %d charged of %d", ErrBudget, len(data), s.chargedBytes, s.budgets.MaxBytes), "staging bytes")
		}
	} else {
		finalBytes = s.chargedBytes + int64(len(data))
	}
	if finalBytes > s.budgets.MaxBytes {
		return nil, o.reject(opID, fmt.Errorf("%w: write needs %d bytes of %d", ErrBudget, finalBytes, s.budgets.MaxBytes), "bytes")
	}
	if s.chargedEntries+extraEntries > s.budgets.MaxEntries {
		return nil, o.reject(opID, fmt.Errorf("%w: write needs %d entries of %d", ErrBudget, s.chargedEntries+extraEntries, s.budgets.MaxEntries), "entries")
	}

	if err := o.mkdirParentsLocked(s, parts); err != nil {
		return nil, o.reject(opID, err, "mkdir parents")
	}
	if !exists {
		if werr := o.writeBytesLocked(s, name, data, ModeCreate); werr != nil {
			return nil, o.reject(opID, werr, "create")
		}
	} else {
		// Stage under an op-bound name, then publish with one rename.
		// The opID is journal-validated ([A-Za-z0-9_.:-], max 128) so it
		// cannot escape the staging directory.
		stage := stagingDir + "/w-" + opID
		_ = s.r.Remove(stage)
		sf, serr := s.r.OpenFile(stage, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if serr != nil {
			return nil, o.reject(opID, fmt.Errorf("workspace: stage %q: %w", rel, serr), "stage")
		}
		_, werr := sf.Write(data)
		syncErr := sf.Sync()
		cerr := sf.Close()
		if werr != nil || syncErr != nil {
			_ = s.r.Remove(stage)
			if werr != nil {
				return nil, o.reject(opID, fmt.Errorf("workspace: stage %q: %w", rel, werr), "stage write")
			}
			return nil, o.reject(opID, fmt.Errorf("workspace: stage sync %q: %w", rel, syncErr), "stage write")
		}
		if cerr != nil {
			_ = s.r.Remove(stage)
			return nil, o.reject(opID, fmt.Errorf("workspace: stage close %q: %w", rel, cerr), "stage close")
		}
		if rerr := s.r.Rename(stage, name); rerr != nil {
			_ = s.r.Remove(stage)
			return nil, o.reject(opID, fmt.Errorf("workspace: publish %q: %w", rel, rerr), "publish")
		}
		s.chargedBytes = finalBytes
	}
	s.tracked[name] = trackedEntry{kind: KindFile, size: int64(len(data)), digest: digestBytes(data)}
	s.rev++
	if len(parts) > s.maxDepth {
		s.maxDepth = len(parts)
	}
	res := &WriteResult{PriorRevision: priorRev, Revision: s.rev, Digest: digestBytes(data)}
	o.closeOp(opID)
	o.remember(opID, res, nil)
	out := *res
	return &out, nil
}

// NodeSpec describes exactly one fixture node operation.
type NodeSpec struct {
	Kind Kind `json:"kind"` // KindDir or KindLink
	// Target is the literal symlink target for KindLink. Absolute targets
	// are rejected: the workspace must not gain writable links to shared
	// sources. Relative targets are stored literally, never resolved.
	Target string `json:"target,omitempty"`
	// Perm optionally tightens the directory mode; zero keeps 0700. It can
	// never widen access beyond 0700.
	Perm os.FileMode `json:"perm,omitempty"`
}

// MakeNodeResult is the terminal fact of one fixture node operation.
type MakeNodeResult struct {
	Kind     Kind   `json:"kind"`
	Target   string `json:"target,omitempty"`
	Revision uint64 `json:"revision"`
	Joined   bool   `json:"joined,omitempty"`
}

// MakeNode applies exactly one fixture operation: create a directory or a
// symlink with a literal relative target. Traversal through symlinks stays
// forbidden, and fixture permissions cannot affect other workspaces.
func (o *Owner) MakeNode(opID string, h Handle, rel string, spec NodeSpec) (*MakeNodeResult, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	s, err := o.lookupLocked(h, true)
	if err != nil {
		return nil, err
	}
	if spec.Kind != KindDir && spec.Kind != KindLink {
		return nil, fmt.Errorf("%w: node kind must be dir or symlink", ErrInvalid)
	}
	parts, err := splitRel(rel)
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("%w: cannot remake the root", ErrInvalid)
	}
	name := joinRel(parts)
	if sealedLocked(s, name) {
		return nil, fmt.Errorf("%w: %q", ErrSealed, rel)
	}
	if len(parts) > s.budgets.MaxDepth {
		return nil, fmt.Errorf("%w: %q exceeds depth %d", ErrBudget, rel, s.budgets.MaxDepth)
	}
	if spec.Kind == KindLink {
		if spec.Target == "" || strings.ContainsRune(spec.Target, 0) {
			return nil, fmt.Errorf("%w: empty or NUL symlink target", ErrInvalid)
		}
		if strings.HasPrefix(spec.Target, "/") {
			return nil, fmt.Errorf("%w: absolute symlink target %q", ErrEscape, spec.Target)
		}
	}
	if _, err := checkParentsLocked(s, parts); err != nil {
		return nil, err
	}
	args := struct {
		Space string   `json:"space"`
		Path  string   `json:"path"`
		Spec  NodeSpec `json:"spec"`
	}{Space: h.SpaceID, Path: name, Spec: spec}
	joined, err := o.journalIntent(opID, args, journal.PathIdentity{Path: s.root}, "", 0)
	if err != nil {
		return nil, err
	}
	if joined {
		v, rerr := o.rejoined(opID)
		if v == nil {
			return nil, rerr
		}
		res := *(v.(*MakeNodeResult))
		res.Joined = true
		return &res, rerr
	}
	if _, lerr := s.r.Lstat(name); lerr == nil {
		return nil, o.reject(opID, fmt.Errorf("%w: %q already exists", ErrCollision, rel), "exists")
	} else if !os.IsNotExist(lerr) {
		return nil, o.reject(opID, fmt.Errorf("workspace: lstat %q: %w", rel, lerr), "lstat")
	}
	if s.chargedEntries+1 > s.budgets.MaxEntries {
		return nil, o.reject(opID, fmt.Errorf("%w: no entry budget for %q", ErrBudget, rel), "entries")
	}
	res := &MakeNodeResult{Kind: spec.Kind, Revision: s.rev + 1}
	if spec.Kind == KindDir {
		perm := os.FileMode(0o700)
		if spec.Perm != 0 {
			// Only tighten; never widen beyond owner-only.
			perm = 0o700 & spec.Perm.Perm()
			perm |= os.FileMode(0o100) // keep traversable for the owner
			if perm&0o700 != perm {
				perm = 0o700
			}
		}
		if merr := s.r.Mkdir(name, perm); merr != nil {
			return nil, o.reject(opID, fmt.Errorf("workspace: mkdir %q: %w", rel, merr), "mkdir")
		}
		s.tracked[name] = trackedEntry{kind: KindDir}
	} else {
		if serr := s.r.Symlink(spec.Target, name); serr != nil {
			return nil, o.reject(opID, fmt.Errorf("workspace: symlink %q: %w", rel, serr), "symlink")
		}
		// Verify the link reads back literally without resolving it.
		back, rerr := s.r.Readlink(name)
		if rerr != nil || back != spec.Target {
			return nil, o.reject(opID, fmt.Errorf("%w: %q link target mismatch", ErrChanged, rel), "verify link")
		}
		res.Target = spec.Target
		s.tracked[name] = trackedEntry{kind: KindLink, target: spec.Target}
	}
	s.chargedEntries++
	s.rev++
	res.Revision = s.rev
	if len(parts) > s.maxDepth {
		s.maxDepth = len(parts)
	}
	o.closeOp(opID)
	o.remember(opID, res, nil)
	out := *res
	return &out, nil
}
