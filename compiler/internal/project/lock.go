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

type ErrorAllocation struct {
	ID   uint64 `json:"id"`
	Kind string `json:"kind"`
}
type Registry struct {
	Active  []ErrorAllocation `json:"active"`
	Retired []uint64          `json:"retired"`
}
type LockEntry struct {
	Path, ManifestSHA256, SourceSHA256, FixturesSHA256 string
	ErrorRegistry                                      Registry
}
type Lock struct{ Dependencies map[string]LockEntry }

func ParseRegistry(data []byte) (Registry, error) {
	r := Registry{Active: []ErrorAllocation{}, Retired: []uint64{}}
	if err := validateJSON(data); err != nil {
		return r, err
	}
	fields, err := object(data, []string{"active", "retired"}, nil)
	if err != nil {
		return r, err
	}
	active, err := array(fields["active"])
	if err != nil {
		return r, err
	}
	seenIDs := map[uint64]bool{}
	kinds := map[string]bool{}
	var previous uint64
	for _, raw := range active {
		fields, err := object(raw, []string{"id", "kind"}, nil)
		if err != nil {
			return r, err
		}
		id, err := applicationID(fields["id"])
		if err != nil {
			return r, err
		}
		kind, err := text(fields["kind"])
		if err != nil {
			return r, err
		}
		parts := strings.Split(kind, "::")
		if len(parts) != 2 || !Identifier(parts[0]) || !Identifier(parts[1]) {
			return r, fmt.Errorf("registry kind %q must be a qualified declaration name", kind)
		}
		if id <= previous || seenIDs[id] || kinds[kind] {
			return r, fmt.Errorf("active registry IDs must increase and kinds must be unique")
		}
		previous = id
		seenIDs[id] = true
		kinds[kind] = true
		r.Active = append(r.Active, ErrorAllocation{ID: id, Kind: kind})
	}
	retired, err := array(fields["retired"])
	if err != nil {
		return r, err
	}
	previous = 0
	for _, raw := range retired {
		id, err := applicationID(raw)
		if err != nil {
			return r, err
		}
		if id <= previous || seenIDs[id] {
			return r, fmt.Errorf("retired registry IDs must increase and be disjoint from active IDs")
		}
		previous = id
		seenIDs[id] = true
		r.Retired = append(r.Retired, id)
	}
	return r, nil
}
func applicationID(raw json.RawMessage) (uint64, error) {
	id, err := integer(raw)
	if err != nil || id < 1000000 || id > 2147483647 {
		return 0, fmt.Errorf("application error ID must be an integer token in 1000000..2147483647")
	}
	return id, nil
}

func ParseLock(data []byte) (Lock, error) {
	lock := Lock{Dependencies: map[string]LockEntry{}}
	if err := validateJSON(data); err != nil {
		return lock, err
	}
	fields, err := object(data, []string{"dependencies"}, nil)
	if err != nil {
		return lock, err
	}
	entries, err := dictionary(fields["dependencies"])
	if err != nil {
		return lock, err
	}
	for _, name := range sortedKeys(entries) {
		if !Identifier(name) {
			return lock, fmt.Errorf("invalid dependency lock identity %q", name)
		}
		fields, err := object(entries[name], []string{"path", "manifest_sha256", "source_sha256", "fixtures_sha256", "error_registry"}, nil)
		if err != nil {
			return lock, err
		}
		entry := LockEntry{}
		destinations := map[string]*string{"path": &entry.Path, "manifest_sha256": &entry.ManifestSHA256, "source_sha256": &entry.SourceSHA256, "fixtures_sha256": &entry.FixturesSHA256}
		for _, key := range sortedKeys(destinations) {
			dest := destinations[key]
			value, err := text(fields[key])
			if err != nil {
				return lock, err
			}
			*dest = value
		}
		if err := NormalizePath(entry.Path); err != nil {
			return lock, err
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
		lock.Dependencies[name] = entry
	}
	return lock, nil
}

func Digest(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }

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
