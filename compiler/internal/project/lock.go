package project

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
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
	var problems []error
	add := func(err error, path ...string) { problems = append(problems, jsonAt(err, path...)) }
	active, retired := map[string]bool{}, map[string]bool{}
	for _, entry := range []struct {
		name   string
		values []string
		set    map[string]bool
	}{{"active", r.Active, active}, {"retired", r.Retired, retired}} {
		for i, name := range entry.values {
			index := strconv.Itoa(i)
			if !qualifiedKind(name) {
				add(fmt.Errorf("registry kind %q must be a qualified declaration name", name), entry.name, index)
				continue
			}
			if i > 0 && qualifiedKind(entry.values[i-1]) && entry.values[i-1] >= name {
				add(fmt.Errorf("%s registry kinds must be sorted and unique", entry.name), entry.name, index)
			}
			entry.set[name] = true
			if entry.name == "retired" && active[name] {
				add(fmt.Errorf("retired error %q is still active", name), entry.name, index)
			}
		}
	}
	chained := map[string]string{}
	for _, current := range sortedKeys(r.Predecessors) {
		chain := r.Predecessors[current]
		if !active[current] {
			problems = append(problems, &jsonPathError{path: []string{"predecessors", current}, key: true, err: fmt.Errorf("predecessor chain for %q lacks an active declaration", current)})
			continue
		}
		if len(chain) == 0 {
			add(fmt.Errorf("predecessor chain for %q is empty", current), "predecessors", current)
			continue
		}
		seen := map[string]bool{}
		for i, prior := range chain {
			at := []string{"predecessors", current, strconv.Itoa(i)}
			if !qualifiedKind(prior) {
				add(fmt.Errorf("predecessor %q must be a qualified declaration name", prior), at...)
				continue
			}
			if seen[prior] {
				add(fmt.Errorf("predecessor %q repeats within one chain", prior), at...)
			}
			seen[prior] = true
			if !retired[prior] {
				add(fmt.Errorf("predecessor %q is not retired", prior), at...)
				continue
			}
			if owner, exists := chained[prior]; exists && owner != current {
				add(fmt.Errorf("predecessor %q chains to both %s and %s", prior, owner, current), at...)
			} else {
				chained[prior] = current
			}
		}
	}
	return errors.Join(problems...)
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
	fields, shapeErr := object(data, []string{"active", "retired"}, []string{"predecessors"})
	if fields == nil {
		return r, shapeErr
	}
	problems := []error{shapeErr}
	invalid := map[string]bool{}
	stringsAt := func(raw json.RawMessage, path ...string) []string {
		values, err := array(raw)
		if err != nil {
			problems = append(problems, jsonFieldError(data, err, path...))
			invalid[jsonPathKey(path)] = true
			return nil
		}
		names := make([]string, len(values))
		for i, value := range values {
			name, err := text(value)
			if err != nil {
				at := append(append([]string{}, path...), strconv.Itoa(i))
				problems = append(problems, jsonFieldError(data, err, at...))
				invalid[jsonPathKey(at)] = true
				continue
			}
			names[i] = name
		}
		return names
	}
	if raw, ok := fields["active"]; ok {
		r.Active = stringsAt(raw, "active")
	}
	if raw, ok := fields["retired"]; ok {
		r.Retired = stringsAt(raw, "retired")
	}
	if raw, ok := fields["predecessors"]; ok {
		entries, err := dictionary(raw)
		if err != nil {
			problems = append(problems, jsonFieldError(data, err, "predecessors"))
		} else {
			r.Predecessors = map[string][]string{}
			for _, current := range sortedKeys(entries) {
				r.Predecessors[current] = stringsAt(entries[current], "predecessors", current)
			}
		}
	}
	var addValidated func(error)
	addValidated = func(err error) {
		if err == nil {
			return
		}
		if many, ok := err.(interface{ Unwrap() []error }); ok {
			for _, child := range many.Unwrap() {
				addValidated(child)
			}
			return
		}
		if located, ok := err.(*jsonPathError); ok {
			for i := len(located.path); i > 0; i-- {
				if invalid[jsonPathKey(located.path[:i])] {
					return
				}
			}
			// Allocation relationships require both correctly decoded lists.
			if len(located.path) > 0 && located.path[0] == "predecessors" && (invalid[jsonPathKey([]string{"active"})] || invalid[jsonPathKey([]string{"retired"})]) {
				return
			}
		}
		problems = append(problems, bindJSONPaths(data, err))
	}
	addValidated(r.Validate())
	return r, errors.Join(problems...)
}

func ParseLock(data []byte) (Lock, error) {
	lock := Lock{Edges: map[string]LockEdge{}, Projects: map[string]LockEntry{}}
	if err := validateJSON(data); err != nil {
		return lock, err
	}
	fields, shapeErr := object(data, []string{"edges", "projects"}, nil)
	if fields == nil {
		return lock, shapeErr
	}
	problems := []error{shapeErr}
	if raw, ok := fields["edges"]; ok {
		edges, err := parseLockEdges(raw)
		lock.Edges = edges
		problems = append(problems, jsonChildError(data, err, "edges"))
	}
	if raw, ok := fields["projects"]; ok {
		entries, err := dictionary(raw)
		if err != nil {
			problems = append(problems, jsonFieldError(data, err, "projects"))
		} else {
			for _, id := range sortedKeys(entries) {
				if !ValidNodeID(id) || id == "can.project.root" {
					problems = append(problems, jsonKeyError(data, fmt.Errorf("invalid lock project identity %q", id), "projects", id))
					continue
				}
				entry, err := parseLockEntry(entries[id], id)
				if err != nil {
					problems = append(problems, jsonChildError(data, err, "projects", id))
					continue
				}
				lock.Projects[id] = entry
			}
		}
	}
	return lock, errors.Join(problems...)
}

func parseLockEntry(raw json.RawMessage, id string) (LockEntry, error) {
	entry := LockEntry{Edges: map[string]LockEdge{}}
	fields, shapeErr := object(raw, []string{"lineage", "manifest_sha256", "source_sha256", "fixtures_sha256", "error_registry", "edges"}, nil)
	if fields == nil {
		return entry, shapeErr
	}
	problems := []error{shapeErr}
	destinations := map[string]*string{"lineage": &entry.Lineage, "manifest_sha256": &entry.ManifestSHA256, "source_sha256": &entry.SourceSHA256, "fixtures_sha256": &entry.FixturesSHA256}
	for _, key := range sortedKeys(destinations) {
		value, exists := fields[key]
		if !exists {
			continue
		}
		text, err := text(value)
		if err == nil {
			if key == "lineage" {
				if text != "" && !Identifier(text) {
					err = fmt.Errorf("lock lineage %q must be a lowercase identifier", text)
				} else if NodeLineage(id) != text {
					err = fmt.Errorf("lock project %q disagrees with its lineage pin", id)
				}
			} else {
				if len(text) != 64 || strings.ToLower(text) != text {
					err = fmt.Errorf("lock digest must be 64 lowercase hexadecimal characters")
				} else if _, decodeErr := hex.DecodeString(text); decodeErr != nil {
					err = fmt.Errorf("invalid lock digest")
				}
			}
		}
		if err != nil {
			problems = append(problems, jsonFieldError(raw, err, key))
			continue
		}
		*destinations[key] = text
	}
	if value, exists := fields["error_registry"]; exists {
		registry, err := ParseRegistry(value)
		entry.ErrorRegistry = registry
		problems = append(problems, jsonChildError(raw, err, "error_registry"))
	}
	if value, exists := fields["edges"]; exists {
		edges, err := parseLockEdges(value)
		entry.Edges = edges
		problems = append(problems, jsonChildError(raw, err, "edges"))
	}
	return entry, errors.Join(problems...)
}

func parseLockEdges(raw json.RawMessage) (map[string]LockEdge, error) {
	edges := map[string]LockEdge{}
	entries, err := dictionary(raw)
	if err != nil {
		return edges, jsonFieldError(raw, err)
	}
	var problems []error
	for _, name := range sortedKeys(entries) {
		if !Identifier(name) {
			problems = append(problems, jsonKeyError(raw, fmt.Errorf("invalid lock edge name %q", name), name))
			continue
		}
		fields, shapeErr := object(entries[name], []string{"target", "path"}, nil)
		entryProblems := []error{shapeErr}
		edge := LockEdge{}
		if fields != nil {
			if value, ok := fields["target"]; ok {
				target, err := text(value)
				if err == nil && (!ValidNodeID(target) || target == "can.project.root") {
					err = fmt.Errorf("invalid lock edge target %q", target)
				}
				if err != nil {
					entryProblems = append(entryProblems, jsonFieldError(entries[name], err, "target"))
				} else {
					edge.Target = target
				}
			}
			if value, ok := fields["path"]; ok {
				path, err := text(value)
				if err == nil {
					err = NormalizePath(path)
				}
				if err != nil {
					entryProblems = append(entryProblems, jsonFieldError(entries[name], err, "path"))
				} else {
					edge.Path = path
				}
			}
		}
		if err := errors.Join(entryProblems...); err != nil {
			problems = append(problems, jsonChildError(raw, err, name))
			continue
		}
		edges[name] = edge
	}
	return edges, errors.Join(problems...)
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
