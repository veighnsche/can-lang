package project

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Registry records one owner's qualified error names. Active kinds must
// exactly match that owner's source declarations; retired names are
// permanently withdrawn and never reused. Predecessors supplies rename
// lineage: each active kind maps to its chain of former qualified names,
// immediate predecessor first, so archived reports keep attributing to
// the current declaration.
type Registry struct {
	Active       []string            `json:"active"`
	Retired      []string            `json:"retired"`
	Predecessors map[string][]string `json:"predecessors,omitempty"`
}

// ReportIdentityVersion tags qualified error identities in terminal
// reports: `can.error.v2:<declaration identity>`.
const ReportIdentityVersion = "can.error.v2"

// Validate enforces the registry contract on parsed or mutated state:
// sorted unique active/retired qualified names, disjoint sets, and
// unambiguous predecessor chains over retired names.
func (r Registry) Validate() error {
	if !sortedUnique(r.Active) {
		return fmt.Errorf("active registry kinds must be sorted and unique")
	}
	if !sortedUnique(r.Retired) {
		return fmt.Errorf("retired registry names must be sorted and unique")
	}
	for _, kind := range r.Active {
		if !qualifiedKind(kind) {
			return fmt.Errorf("registry kind %q must be a qualified declaration name", kind)
		}
	}
	active := map[string]bool{}
	for _, kind := range r.Active {
		active[kind] = true
	}
	retired := map[string]bool{}
	for _, name := range r.Retired {
		if !qualifiedKind(name) {
			return fmt.Errorf("retired name %q must be a qualified declaration name", name)
		}
		if active[name] {
			return fmt.Errorf("retired error %q is still active", name)
		}
		retired[name] = true
	}
	chained := map[string]string{}
	for _, current := range sortedKeys(r.Predecessors) {
		chain := r.Predecessors[current]
		if !active[current] {
			return fmt.Errorf("predecessor chain for %q lacks an active declaration", current)
		}
		if len(chain) == 0 {
			return fmt.Errorf("predecessor chain for %q is empty", current)
		}
		seen := map[string]bool{}
		for _, prior := range chain {
			if !qualifiedKind(prior) {
				return fmt.Errorf("predecessor %q must be a qualified declaration name", prior)
			}
			if seen[prior] {
				return fmt.Errorf("predecessor %q repeats within one chain", prior)
			}
			seen[prior] = true
			if !retired[prior] {
				return fmt.Errorf("predecessor %q is not retired", prior)
			}
			if owner, exists := chained[prior]; exists {
				return fmt.Errorf("predecessor %q chains to both %s and %s", prior, owner, current)
			}
			chained[prior] = current
		}
	}
	return nil
}

func sortedUnique(names []string) bool {
	for i, name := range names {
		if i > 0 && names[i-1] >= name {
			return false
		}
	}
	return true
}

func qualifiedKind(kind string) bool {
	parts := strings.Split(kind, "::")
	return len(parts) == 2 && Identifier(parts[0]) && Identifier(parts[1])
}

// ResolveKind maps a qualified error name to its current active kind,
// following one supplied predecessor chain. Active kinds resolve to
// themselves; retired names without a chain do not resolve.
func (r Registry) ResolveKind(kind string) (string, bool) {
	for _, active := range r.Active {
		if active == kind {
			return kind, true
		}
	}
	for _, current := range sortedKeys(r.Predecessors) {
		for _, prior := range r.Predecessors[current] {
			if prior == kind {
				return current, true
			}
		}
	}
	return "", false
}

// LockEdge pins one direct dependency edge: the local edge name maps to a
// canonical node identity reached through a parent-relative path.
type LockEdge struct {
	Target, Path string
}

// LockEntry pins one non-root instance: its lineage, content digests,
// registry snapshot, and direct edges.
type LockEntry struct {
	Lineage                                      string
	ManifestSHA256, SourceSHA256, FixturesSHA256 string
	ErrorRegistry                                Registry
	Edges                                        map[string]LockEdge
}

// Lock pins the root's direct edges plus every non-root instance by
// canonical node identity. Paths stay parent-relative so checkout
// relocation never invalidates a lock.
type Lock struct {
	Edges    map[string]LockEdge
	Projects map[string]LockEntry
}

func ParseRegistry(data []byte) (Registry, error) {
	r := Registry{Active: []string{}, Retired: []string{}}
	if err := validateJSON(data); err != nil {
		return r, err
	}
	fields, err := object(data, []string{"active", "retired"}, []string{"predecessors"})
	if err != nil {
		return r, err
	}
	active, err := array(fields["active"])
	if err != nil {
		return r, err
	}
	for _, raw := range active {
		kind, err := text(raw)
		if err != nil {
			return r, err
		}
		r.Active = append(r.Active, kind)
	}
	retired, err := array(fields["retired"])
	if err != nil {
		return r, err
	}
	for _, raw := range retired {
		name, err := text(raw)
		if err != nil {
			return r, err
		}
		r.Retired = append(r.Retired, name)
	}
	if raw, exists := fields["predecessors"]; exists {
		entries, err := dictionary(raw)
		if err != nil {
			return r, err
		}
		r.Predecessors = map[string][]string{}
		for _, current := range sortedKeys(entries) {
			chain, err := array(entries[current])
			if err != nil {
				return r, err
			}
			names := []string{}
			for _, raw := range chain {
				name, err := text(raw)
				if err != nil {
					return r, err
				}
				names = append(names, name)
			}
			r.Predecessors[current] = names
		}
	}
	if err := r.Validate(); err != nil {
		return r, err
	}
	return r, nil
}

func ParseLock(data []byte) (Lock, error) {
	lock := Lock{Edges: map[string]LockEdge{}, Projects: map[string]LockEntry{}}
	if err := validateJSON(data); err != nil {
		return lock, err
	}
	fields, err := object(data, []string{"edges", "projects"}, nil)
	if err != nil {
		return lock, err
	}
	edges, err := parseLockEdges(fields["edges"])
	if err != nil {
		return lock, err
	}
	lock.Edges = edges
	entries, err := dictionary(fields["projects"])
	if err != nil {
		return lock, err
	}
	for _, id := range sortedKeys(entries) {
		if !ValidNodeID(id) || id == "can.project.root" {
			return lock, fmt.Errorf("invalid lock project identity %q", id)
		}
		fields, err := object(entries[id], []string{"lineage", "manifest_sha256", "source_sha256", "fixtures_sha256", "error_registry", "edges"}, nil)
		if err != nil {
			return lock, err
		}
		entry := LockEntry{Edges: map[string]LockEdge{}}
		destinations := map[string]*string{"lineage": &entry.Lineage, "manifest_sha256": &entry.ManifestSHA256, "source_sha256": &entry.SourceSHA256, "fixtures_sha256": &entry.FixturesSHA256}
		for _, key := range sortedKeys(destinations) {
			dest := destinations[key]
			value, err := text(fields[key])
			if err != nil {
				return lock, err
			}
			*dest = value
		}
		if entry.Lineage != "" && !Identifier(entry.Lineage) {
			return lock, fmt.Errorf("lock lineage %q must be a lowercase identifier", entry.Lineage)
		}
		if NodeLineage(id) != entry.Lineage {
			return lock, fmt.Errorf("lock project %q disagrees with its lineage pin", id)
		}
		for _, digest := range []string{entry.ManifestSHA256, entry.SourceSHA256, entry.FixturesSHA256} {
			if len(digest) != 64 || strings.ToLower(digest) != digest {
				return lock, fmt.Errorf("lock digest must be 64 lowercase hexadecimal characters")
			}
			if _, err := hex.DecodeString(digest); err != nil {
				return lock, fmt.Errorf("invalid lock digest")
			}
		}
		entry.ErrorRegistry, err = ParseRegistry(fields["error_registry"])
		if err != nil {
			return lock, err
		}
		entry.Edges, err = parseLockEdges(fields["edges"])
		if err != nil {
			return lock, err
		}
		lock.Projects[id] = entry
	}
	return lock, nil
}

func parseLockEdges(raw json.RawMessage) (map[string]LockEdge, error) {
	edges := map[string]LockEdge{}
	entries, err := dictionary(raw)
	if err != nil {
		return edges, err
	}
	for _, name := range sortedKeys(entries) {
		if !Identifier(name) {
			return edges, fmt.Errorf("invalid lock edge name %q", name)
		}
		fields, err := object(entries[name], []string{"target", "path"}, nil)
		if err != nil {
			return edges, err
		}
		target, err := text(fields["target"])
		if err != nil {
			return edges, err
		}
		if !ValidNodeID(target) || target == "can.project.root" {
			return edges, fmt.Errorf("invalid lock edge target %q", target)
		}
		edgePath, err := text(fields["path"])
		if err != nil {
			return edges, err
		}
		if err := NormalizePath(edgePath); err != nil {
			return edges, err
		}
		edges[name] = LockEdge{Target: target, Path: edgePath}
	}
	return edges, nil
}

func Digest(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }

// ValidNodeID reports whether id is a canonical instance identity: the
// root, a declared lineage, or a legacy edge path from the root.
func ValidNodeID(id string) bool {
	if id == "can.project.root" {
		return true
	}
	if lineage, ok := strings.CutPrefix(id, "can.project.lineage/"); ok {
		return Identifier(lineage)
	}
	path, ok := strings.CutPrefix(id, "can.project.dependency/")
	if !ok {
		return false
	}
	for _, edge := range strings.Split(path, "/") {
		if !Identifier(edge) {
			return false
		}
	}
	return true
}

// NodeLineage returns the declared lineage carried by a lineage identity,
// or "" for the root and legacy edge-path identities.
func NodeLineage(id string) string {
	lineage, ok := strings.CutPrefix(id, "can.project.lineage/")
	if !ok || !Identifier(lineage) {
		return ""
	}
	return lineage
}

type SourceBytes struct {
	Path  string
	Bytes []byte
}

// SourceDigest implements P2's byte-exact, length-prefixed source tree format.
// Paths are normalized relative to source_root, not machine absolute paths.
func SourceDigest(files []SourceBytes) (string, error) {
	ordered := append([]SourceBytes(nil), files...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Path < ordered[j].Path })
	hash := sha256.New()
	hash.Write([]byte("can-source-tree-v1\x00"))
	var length [8]byte
	previous := ""
	for _, file := range ordered {
		if err := NormalizePath(file.Path); err != nil {
			return "", err
		}
		if !strings.HasSuffix(file.Path, ".can") || file.Path == previous {
			return "", fmt.Errorf("source digest requires unique .can paths")
		}
		previous = file.Path
		binary.BigEndian.PutUint64(length[:], uint64(len([]byte(file.Path))))
		hash.Write(length[:])
		hash.Write([]byte(file.Path))
		binary.BigEndian.PutUint64(length[:], uint64(len(file.Bytes)))
		hash.Write(length[:])
		hash.Write(file.Bytes)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
