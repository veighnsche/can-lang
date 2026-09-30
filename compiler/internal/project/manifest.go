package project

import (
	"errors"
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
	// Invalid components retain explicit blocking evidence for recovered loads.
	Invalid             map[string]error
	InvalidDependencies map[string]error
	InvalidSQL          map[string]error
	SourceRoot          string
	Project             string // optional lineage ID; empty means a legacy edge-path identity
	Dependencies        map[string]string
	Assets              map[string]string
	SQL                 map[string]SQLDescriptor
	ErrorRegistry       string
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
		return &GraphError{Kind: KindBadPath, Msg: fmt.Sprintf("path %q must be a normalized relative UTF-8 path", name)}
	}
	for _, part := range strings.Split(name, "/") {
		if part == ".." {
			return &GraphError{Kind: KindBadPath, Msg: fmt.Sprintf("parent traversal is forbidden in path %q", name)}
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
		return "", &GraphError{Kind: KindEscape, Msg: fmt.Sprintf("path %q escapes its manifest directory after symlink resolution", name)}
	}
	info, err := os.Stat(real)
	if err != nil {
		return "", err
	}
	if directory {
		if !info.IsDir() {
			return "", &GraphError{Kind: KindNotContainer, Msg: fmt.Sprintf("path %q is not a directory", name)}
		}
	} else if !info.Mode().IsRegular() {
		return "", &GraphError{Kind: KindNotContainer, Msg: fmt.Sprintf("path %q is not a regular file", name)}
	}
	return real, nil
}
func Contains(root, name string) bool {
	rel, err := filepath.Rel(root, name)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

func ParseManifest(data []byte) (Manifest, error) {
	m := Manifest{Invalid: map[string]error{}, InvalidDependencies: map[string]error{}, InvalidSQL: map[string]error{}, Dependencies: map[string]string{}, Assets: map[string]string{}, SQL: map[string]SQLDescriptor{}}
	if err := validateJSON(data); err != nil {
		m.Invalid["syntax"] = err
		return m, err
	}
	fields, shapeErr := object(data, []string{"source_root", "error_registry"}, []string{"project", "dependencies", "assets", "sql"})
	if fields == nil {
		m.Invalid["shape"] = shapeErr
		return m, shapeErr
	}
	problems := []error{shapeErr}
	if shapeErr != nil {
		m.Invalid["shape"] = shapeErr
	}
	for _, key := range []string{"source_root", "error_registry"} {
		if _, ok := fields[key]; !ok {
			m.Invalid[key] = fmt.Errorf("missing field %q", key)
		}
	}
	add := func(key string, issue error) {
		problems = append(problems, issue)
		m.Invalid[key] = errors.Join(m.Invalid[key], issue)
	}
	if raw, exists := fields["project"]; exists {
		lineage, err := text(raw)
		if err == nil && !Identifier(lineage) {
			err = fmt.Errorf("project lineage %q must be a lowercase identifier", lineage)
		}
		if err != nil {
			add("project", jsonFieldError(data, err, "project"))
		} else {
			m.Project = lineage
		}
	}
	destinations := map[string]*string{"source_root": &m.SourceRoot, "error_registry": &m.ErrorRegistry}
	for _, key := range sortedKeys(destinations) {
		raw, exists := fields[key]
		if !exists {
			continue
		}
		value, err := text(raw)
		if err == nil {
			err = NormalizePath(value)
		}
		if err != nil {
			add(key, jsonFieldError(data, fmt.Errorf("%s: %w", key, err), key))
			continue
		}
		*destinations[key] = value
	}
	for _, key := range []string{"dependencies", "assets"} {
		raw, exists := fields[key]
		if !exists {
			continue
		}
		values, err := dictionary(raw)
		if err != nil {
			add(key, jsonFieldError(data, err, key))
			continue
		}
		for _, name := range sortedKeys(values) {
			if name == "" || strings.ContainsRune(name, 0) || (key == "dependencies" && !Identifier(name)) {
				issue := jsonKeyError(data, fmt.Errorf("invalid %s name %q", key, name), key, name)
				add(key, issue)
				if key == "dependencies" {
					m.InvalidDependencies[name] = issue
				}
				continue
			}
			value, err := text(values[name])
			if err == nil {
				err = NormalizePath(value)
			}
			if err != nil {
				issue := jsonFieldError(data, err, key, name)
				add(key, issue)
				if key == "dependencies" {
					m.InvalidDependencies[name] = issue
				}
				continue
			}
			if key == "dependencies" {
				m.Dependencies[name] = value
			} else {
				m.Assets[name] = value
			}
		}
	}
	if raw, exists := fields["sql"]; exists {
		decoded, invalid, err := decodeSQLMap(raw)
		m.SQL = decoded
		for name, issue := range invalid {
			m.InvalidSQL[name] = jsonChildError(data, issue, "sql")
		}
		if err != nil {
			add("sql", jsonChildError(data, err, "sql"))
		}
	}
	return m, errors.Join(problems...)
}
