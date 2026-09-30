package workspace

import (
	"fmt"
	"os"
	"sort"

	journal "github.com/veighnsche/can-lang/tools/native-test-owner/journal"
)

// SealScope declares what a seal covers. A full-tree seal covers the whole
// workspace including additions and removals; a partial seal covers exactly
// the listed paths (directories cover their children) and cannot claim the
// complete namespace.
type SealScope struct {
	Full  bool     `json:"full"`
	Paths []string `json:"paths,omitempty"`
}

// SealEntry is one inventoried node with content/metadata identity.
type SealEntry struct {
	Path      string    `json:"path"`
	Kind      Kind      `json:"kind"`
	Size      int64     `json:"size,omitempty"`
	Digest    string    `json:"digest,omitempty"`
	Target    string    `json:"target,omitempty"`
	Stability Stability `json:"stability"`
}

// SealReceipt is the terminal fact of one seal: a sorted inventory plus the
// protection strength actually achieved.
type SealReceipt struct {
	Scope     SealScope   `json:"scope"`
	Revision  uint64      `json:"revision"`
	Stability Stability   `json:"stability"`
	Entries   []SealEntry `json:"entries"`
	Absent    []string    `json:"absent,omitempty"`
	Joined    bool        `json:"joined,omitempty"`
}

// Seal stops API mutation within the scope and returns a sorted inventory
// with content identity and protection strength. Full-tree seals cover
// additions and removals; partial seals list relevant absence claims for
// selected paths that do not exist. Strength is scoped_revision only when
// every inventoried node matches capability-tracked state; any untracked or
// out-of-band-changed node degrades the whole receipt.
func (o *Owner) Seal(opID string, h Handle, scope SealScope) (*SealReceipt, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	s, err := o.lookupLocked(h, true)
	if err != nil {
		return nil, err
	}
	var normPaths []string
	if scope.Full {
		if len(scope.Paths) != 0 {
			return nil, fmt.Errorf("%w: full seal takes no paths", ErrInvalid)
		}
	} else {
		if len(scope.Paths) == 0 {
			return nil, fmt.Errorf("%w: partial seal needs paths", ErrInvalid)
		}
		seen := map[string]bool{}
		for _, p := range scope.Paths {
			parts, verr := splitRel(p)
			if verr != nil {
				return nil, verr
			}
			n := joinRel(parts)
			if n == "" {
				return nil, fmt.Errorf("%w: partial seal cannot name the root; use full", ErrInvalid)
			}
			if !seen[n] {
				seen[n] = true
				normPaths = append(normPaths, n)
			}
		}
		sort.Strings(normPaths)
	}
	normScope := SealScope{Full: scope.Full, Paths: normPaths}
	args := struct {
		Space string    `json:"space"`
		Scope SealScope `json:"scope"`
	}{Space: h.SpaceID, Scope: normScope}
	joined, err := o.journalIntent(opID, args, journal.PathIdentity{Path: s.root}, "", 0)
	if err != nil {
		return nil, err
	}
	if joined {
		v, rerr := o.rejoined(opID)
		if v == nil {
			return nil, rerr
		}
		res := *(v.(*SealReceipt))
		res.Joined = true
		return &res, rerr
	}
	// An existing full seal (or an overlapping partial seal) joins: sealing
	// never reopens mutation, and a second seal cannot strengthen an older
	// receipt's stability claim.
	if s.seal != nil && (s.sealedFull || coversLocked(s, normScope)) {
		res := *s.seal
		res.Joined = true
		o.remember(opID, &res, nil)
		o.closeOp(opID)
		out := res
		return &out, nil
	}

	receipt := &SealReceipt{Scope: normScope, Revision: s.rev, Stability: StabilityScopedRevision}
	if scope.Full {
		entries, werr := o.walkInventoryLocked(s)
		if werr != nil {
			return nil, o.reject(opID, werr, "inventory")
		}
		receipt.Entries = entries
	} else {
		for _, p := range normPaths {
			fi, lerr := s.r.Lstat(p)
			if lerr != nil {
				if os.IsNotExist(lerr) {
					receipt.Absent = append(receipt.Absent, p)
					continue
				}
				return nil, o.reject(opID, fmt.Errorf("workspace: seal lstat %q: %w", p, lerr), "inventory")
			}
			if fi.Mode()&os.ModeSymlink != 0 {
				entry, rerr := o.sealOneLocked(s, p, fi)
				if rerr != nil {
					return nil, o.reject(opID, rerr, "inventory")
				}
				receipt.Entries = append(receipt.Entries, entry)
				continue
			}
			if fi.IsDir() {
				sub, werr := o.walkSubtreeLocked(s, p)
				if werr != nil {
					return nil, o.reject(opID, werr, "inventory")
				}
				receipt.Entries = append(receipt.Entries, sub...)
				continue
			}
			entry, rerr := o.sealOneLocked(s, p, fi)
			if rerr != nil {
				return nil, o.reject(opID, rerr, "inventory")
			}
			receipt.Entries = append(receipt.Entries, entry)
		}
		sort.Slice(receipt.Entries, func(i, j int) bool { return receipt.Entries[i].Path < receipt.Entries[j].Path })
	}
	for _, e := range receipt.Entries {
		switch e.Stability {
		case StabilityUnknown:
			receipt.Stability = StabilityUnknown
		case StabilityChangedObserved:
			if receipt.Stability == StabilityScopedRevision {
				receipt.Stability = StabilityChangedObserved
			}
		}
		if receipt.Stability == StabilityUnknown {
			break
		}
	}
	if scope.Full {
		s.sealedFull = true
	} else {
		for _, p := range normPaths {
			s.sealedPaths[p] = true
		}
	}
	s.sealRev = s.rev
	s.seal = receipt
	o.closeOp(opID)
	o.remember(opID, receipt, nil)
	out := *receipt
	return &out, nil
}

// coversLocked reports whether an existing partial seal already covers every
// path of the requested scope.
func coversLocked(s *space, want SealScope) bool {
	if want.Full {
		return false
	}
	for _, p := range want.Paths {
		if !sealedLocked(s, p) {
			return false
		}
	}
	return true
}

// sealOneLocked builds the inventory entry for one Lstat'd path without
// following symlinks. Files are content-hashed; links record their literal
// target.
func (o *Owner) sealOneLocked(s *space, rel string, fi os.FileInfo) (SealEntry, error) {
	e := SealEntry{Path: rel, Kind: kindOf(fi), Size: fi.Size()}
	switch e.Kind {
	case KindFile:
		back, err := s.r.ReadFile(rel)
		if err != nil {
			return e, fmt.Errorf("workspace: seal read %q: %w", rel, err)
		}
		e.Digest = digestBytes(back)
		e.Size = int64(len(back))
	case KindLink:
		t, err := s.r.Readlink(rel)
		if err != nil {
			return e, fmt.Errorf("workspace: seal readlink %q: %w", rel, err)
		}
		e.Target = t
		e.Size = 0
	}
	e.Stability = StabilityUnknown
	if t, ok := s.tracked[rel]; ok && t.kind == e.Kind {
		switch e.Kind {
		case KindFile:
			if t.digest == e.Digest && t.size == e.Size {
				e.Stability = StabilityScopedRevision
			} else {
				e.Stability = StabilityChangedObserved
			}
		case KindLink:
			if t.target == e.Target {
				e.Stability = StabilityScopedRevision
			} else {
				e.Stability = StabilityChangedObserved
			}
		case KindDir:
			e.Stability = StabilityScopedRevision
		}
	} else if ok {
		e.Stability = StabilityChangedObserved
	}
	return e, nil
}

// walkInventoryLocked inventories the whole tree without following
// symlinks. Internal staging is included: it is owned bytes, not hidden.
func (o *Owner) walkInventoryLocked(s *space) ([]SealEntry, error) {
	return o.walkSubtreeLocked(s, ".")
}

// walkSubtreeLocked inventories rel ("." for the root) depth-first with an
// explicit bounded stack. Directory entries are Lstat-observed; symlinks
// are recorded, never descended.
func (o *Owner) walkSubtreeLocked(s *space, rel string) ([]SealEntry, error) {
	var out []SealEntry
	type frame struct {
		rel   string
		depth int
	}
	stack := []frame{{rel: rel, depth: depthOf(rel)}}
	seen := 0
	for len(stack) > 0 {
		fr := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		name := fr.rel
		if name == "" {
			name = "."
		}
		fi, err := s.r.Lstat(name)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("workspace: inventory lstat %q: %w", name, err)
		}
		if name != "." {
			entry, serr := o.sealOneLocked(s, name, fi)
			if serr != nil {
				return nil, serr
			}
			out = append(out, entry)
			seen++
			if seen > s.budgets.MaxEntries+16 {
				return nil, fmt.Errorf("%w: inventory exceeds entry budget", ErrBudget)
			}
		}
		if fi.Mode()&os.ModeSymlink != 0 || !fi.IsDir() {
			continue
		}
		if fr.depth >= s.budgets.MaxDepth+1 {
			return nil, fmt.Errorf("%w: inventory exceeds depth %d", ErrBudget, s.budgets.MaxDepth)
		}
		f, oerr := s.r.Open(name)
		if oerr != nil {
			return nil, fmt.Errorf("workspace: inventory open %q: %w", name, oerr)
		}
		dirents, rerr := f.ReadDir(-1)
		cerr := f.Close()
		if rerr != nil {
			return nil, fmt.Errorf("workspace: inventory read %q: %w", name, rerr)
		}
		if cerr != nil {
			return nil, fmt.Errorf("workspace: inventory close %q: %w", name, cerr)
		}
		// Push in reverse so the walk visits sorted order; the receipt is
		// re-sorted anyway.
		for i := len(dirents) - 1; i >= 0; i-- {
			child := dirents[i].Name()
			if name != "." {
				child = name + "/" + child
			}
			stack = append(stack, frame{rel: child, depth: fr.depth + 1})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

func depthOf(rel string) int {
	if rel == "" || rel == "." {
		return 0
	}
	n := 1
	for i := 0; i < len(rel); i++ {
		if rel[i] == '/' {
			n++
		}
	}
	return n
}
