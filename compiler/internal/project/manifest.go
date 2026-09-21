package project

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
)

type Manifest struct {
	SourceRoot    string
	Dependencies  map[string]string
	Assets        map[string]string
	SQL           map[string]SQLDescriptor
	ErrorRegistry string
}
type SQLDescriptor struct {
	Dialect, Statement, ParameterType, RowType, Cardinality string
	Parameters                                              []string
	RowLimitParameter                                       uint64
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
	fields, err := object(data, []string{"source_root", "error_registry"}, []string{"dependencies", "assets", "sql"})
	if err != nil {
		return m, err
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
		values, err := dictionary(raw)
		if err != nil {
			return m, fmt.Errorf("sql: %w", err)
		}
		for _, name := range sortedKeys(values) {
			if name == "" || strings.ContainsRune(name, 0) {
				return m, fmt.Errorf("invalid SQL descriptor name")
			}
			descriptor, err := parseSQL(values[name])
			if err != nil {
				return m, fmt.Errorf("sql %q: %w", name, err)
			}
			m.SQL[name] = descriptor
		}
	}
	return m, nil
}

func parseSQL(raw json.RawMessage) (SQLDescriptor, error) {
	d := SQLDescriptor{}
	fields, err := object(raw, []string{"dialect", "statement", "parameters", "parameter_type", "row_type", "cardinality"}, []string{"row_limit_parameter"})
	if err != nil {
		return d, err
	}
	destinations := map[string]*string{"dialect": &d.Dialect, "statement": &d.Statement, "parameter_type": &d.ParameterType, "row_type": &d.RowType, "cardinality": &d.Cardinality}
	for _, key := range sortedKeys(destinations) {
		dest := destinations[key]
		value, err := text(fields[key])
		if err != nil || value == "" {
			return d, fmt.Errorf("%s must be a nonempty string", key)
		}
		*dest = value
	}
	if d.Dialect != "postgresql" {
		return d, fmt.Errorf("unsupported SQL dialect")
	}
	if d.Cardinality != "one" && d.Cardinality != "optional" && d.Cardinality != "many" && d.Cardinality != "execute" {
		return d, fmt.Errorf("invalid SQL cardinality")
	}
	parameters, err := array(fields["parameters"])
	if err != nil {
		return d, err
	}
	seen := map[string]bool{}
	for _, raw := range parameters {
		value, err := text(raw)
		if err != nil || !Identifier(value) || seen[value] {
			return d, fmt.Errorf("invalid or duplicate SQL parameter name")
		}
		seen[value] = true
		d.Parameters = append(d.Parameters, value)
	}
	for _, name := range []string{d.ParameterType, d.RowType} {
		file, err := source.New("<manifest-type>", name)
		if err != nil {
			return d, err
		}
		typ, diagnostics := syntax.ParseType(file)
		if len(diagnostics) > 0 {
			return d, fmt.Errorf("invalid SQL type %q", name)
		}
		nominal, ok := typ.(*syntax.NamedType)
		if !ok || nominal.Name.Package == "" {
			return d, fmt.Errorf("SQL types must be fully qualified nominal types")
		}
	}
	limit, exists := fields["row_limit_parameter"]
	if d.Cardinality == "execute" {
		if exists {
			return d, fmt.Errorf("execute cannot declare row_limit_parameter")
		}
	} else {
		if !exists {
			return d, fmt.Errorf("row-returning SQL requires row_limit_parameter")
		}
		d.RowLimitParameter, err = integer(limit)
		if err != nil || d.RowLimitParameter != uint64(len(d.Parameters))+1 {
			return d, fmt.Errorf("row_limit_parameter must follow the application parameters")
		}
	}
	return d, nil
}
