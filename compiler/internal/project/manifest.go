package project

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/veighnsche/can-lang/compiler/internal/syntax"
)

type Manifest struct {
	SourceRoot    string
	Project       string // optional lineage ID; empty means a legacy edge-path identity
	Dependencies  map[string]string
	Assets        map[string]string
	SQL           map[string]SQLDescriptor
	ErrorRegistry string
}

var identifier = regexp.MustCompile(`^[a-z][a-z0-9]*(?:_[a-z0-9]+)*$`)

func Identifier(name string) bool { return identifier.MatchString(name) && !syntax.IsHardKeyword(name) }
func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func NormalizePath(name string) error {
	if name == "" || !utf8.ValidString(name) || strings.ContainsRune(name, 0) || path.IsAbs(name) || filepath.IsAbs(name) || path.Clean(name) != name {
		return fmt.Errorf("path %q must be a normalized relative UTF-8 path", name)
	}
	for _, part := range strings.Split(name, "/") {
		if part == ".." {
			return fmt.Errorf("parent traversal is forbidden in path %q", name)
		}
	}
	return nil
}

// ConfinedPath resolves every symlink before checking the path against its
// manifest directory. It returns the canonical path used by subsequent reads.
func ConfinedPath(root, name string, directory bool) (string, error) {
	if err := NormalizePath(name); err != nil {
		return "", err
	}
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return "", err
	}
	real, err := filepath.EvalSymlinks(filepath.Join(root, filepath.FromSlash(name)))
	if err != nil {
		return "", err
	}
	real, err = filepath.Abs(real)
	if err != nil {
		return "", err
	}
	if !Contains(root, real) {
		return "", fmt.Errorf("path %q escapes its manifest directory after symlink resolution", name)
	}
	info, err := os.Stat(real)
	if err != nil {
		return "", err
	}
	if directory {
		if !info.IsDir() {
			return "", fmt.Errorf("path %q is not a directory", name)
		}
	} else if !info.Mode().IsRegular() {
		return "", fmt.Errorf("path %q is not a regular file", name)
	}
	return real, nil
}
func Contains(root, name string) bool {
	rel, err := filepath.Rel(root, name)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

func ParseManifest(data []byte) (Manifest, error) {
	m := Manifest{Dependencies: map[string]string{}, Assets: map[string]string{}, SQL: map[string]SQLDescriptor{}}
	if err := validateJSON(data); err != nil {
		return m, err
	}
	fields, err := object(data, []string{"source_root", "error_registry"}, []string{"project", "dependencies", "assets", "sql"})
	if err != nil {
		return m, err
	}
	if raw, exists := fields["project"]; exists {
		lineage, err := text(raw)
		if err != nil {
			return m, fmt.Errorf("project: %w", err)
		}
		if !Identifier(lineage) {
			return m, fmt.Errorf("project lineage %q must be a lowercase identifier", lineage)
		}
		m.Project = lineage
	}
	destinations := map[string]*string{"source_root": &m.SourceRoot, "error_registry": &m.ErrorRegistry}
	for _, key := range sortedKeys(destinations) {
		dest := destinations[key]
		value, err := text(fields[key])
		if err != nil {
			return m, fmt.Errorf("%s: %w", key, err)
		}
		if err := NormalizePath(value); err != nil {
			return m, err
		}
		*dest = value
	}
	for _, key := range []string{"dependencies", "assets"} {
		raw, exists := fields[key]
		if !exists {
			continue
		}
		values, err := dictionary(raw)
		if err != nil {
			return m, fmt.Errorf("%s: %w", key, err)
		}
		for _, name := range sortedKeys(values) {
			if name == "" || strings.ContainsRune(name, 0) || (key == "dependencies" && !Identifier(name)) {
				return m, fmt.Errorf("invalid %s name %q", key, name)
			}
			value, err := text(values[name])
			if err != nil {
				return m, err
			}
			if err := NormalizePath(value); err != nil {
				return m, err
			}
			if key == "dependencies" {
				m.Dependencies[name] = value
			} else {
				m.Assets[name] = value
			}
		}
	}
	if raw, exists := fields["sql"]; exists {
		decoded, err := decodeSQLMap(raw)
		if err != nil {
			return m, err
		}
		m.SQL = decoded
	}
	return m, nil
}
