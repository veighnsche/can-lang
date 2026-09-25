package browser

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"
)

// BundleFile is one published post-bundle byte sequence: final JavaScript,
// a source map, or the immutable diagnostic table.
type BundleFile struct {
	Path  string
	Bytes []byte
}

// Bundle is the post-bundle published set for one browser build: the final
// digest JavaScript (entry plus any code-split chunks), one source map per
// script, and the immutable diagnostic table. Digests is the verified-build
// manifest projection (UP15 I-6): logical path to lowercase SHA-256 hex for
// every published byte. Secrets carries build-known secret material, such
// as connection endpoints and bearer names, which must not appear anywhere
// in the output. Paths are manifest-logical (e.g. "app.js",
// "app.js.map", "diagnostics/table.json").
type Bundle struct {
	Entry   string
	Table   string
	Files   []BundleFile
	Digests map[string]string
	Secrets []string
}

// forbiddenTableKeys implements the I-5 diagnostic exclusion list
// structurally: records may carry category, phase and sealed Can
// file/line/column, never native causes, raw input, stacks or secrets.
var forbiddenTableKeys = map[string]bool{
	"stack": true, "stackTrace": true, "stacktrace": true,
	"cause": true, "nativeCause": true, "nativeStack": true,
	"rawInput": true, "raw": true,
	"secret": true, "secrets": true, "secretBytes": true,
}

// AuditBundle verifies the published post-bundle browser output: every
// file is digest-bound to the manifest with nothing missing or extra,
// scripts carry no unaccounted edges or host operations, every script
// binds its source map, maps decode and agree with their scripts, the
// diagnostic table parses without excluded data, and no secret canary or
// build-known secret appears in any byte. Diagnostics cite the offending
// file with line and column plus the map-resolved original position.
func AuditBundle(bundle Bundle) error {
	if bundle.Entry == "" || !strings.HasSuffix(bundle.Entry, ".js") {
		return fmt.Errorf("browser bundle audit: entry must be a published .js file")
	}
	if bundle.Table == "" || !strings.HasSuffix(bundle.Table, ".json") {
		return fmt.Errorf("browser bundle audit: table must be a published .json diagnostic table")
	}
	if len(bundle.Files) == 0 {
		return fmt.Errorf("browser bundle audit: bundle publishes no files")
	}
	byPath := map[string]BundleFile{}
	for _, file := range bundle.Files {
		if err := checkBundlePath(file.Path); err != nil {
			return fmt.Errorf("browser bundle audit: %w", err)
		}
		if _, dup := byPath[file.Path]; dup {
			return fmt.Errorf("browser bundle audit: duplicate published file %s", file.Path)
		}
		byPath[file.Path] = file
	}
	if err := checkBundleDigests(byPath, bundle.Digests); err != nil {
		return err
	}
	if _, ok := byPath[bundle.Entry]; !ok {
		return fmt.Errorf("browser bundle audit: entry %s is not published", bundle.Entry)
	}
	table, ok := byPath[bundle.Table]
	if !ok {
		return fmt.Errorf("browser bundle audit: diagnostic table %s is not published", bundle.Table)
	}
	scripts, maps, err := classifyBundle(byPath, bundle.Table)
	if err != nil {
		return err
	}
	decoded := map[string]bundleMap{}
	for _, name := range maps {
		parsed, segments, err := parseBundleMap(byPath[name])
		if err != nil {
			return err
		}
		decoded[name] = bundleMap{parsed: parsed, segments: segments}
	}
	for _, name := range scripts {
		script := byPath[name]
		mapName := name + ".map"
		paired, ok := decoded[mapName]
		if !ok {
			return fmt.Errorf("browser bundle audit: published script %s lacks its source map %s", name, mapName)
		}
		if err := auditBundleScript(script, paired, byPath, bundle.Secrets); err != nil {
			return err
		}
	}
	if err := auditBundleTable(table, bundle.Secrets); err != nil {
		return err
	}
	for _, name := range maps {
		if err := scanBundleSecrets(byPath[name], bundle.Secrets); err != nil {
			return err
		}
	}
	return nil
}

type bundleMap struct {
	parsed   decodedSourceMap
	segments [][]mapSegment
}

func checkBundlePath(name string) error {
	if name == "" || strings.HasPrefix(name, "/") || strings.Contains(name, "\\") || path.Clean(name) != name {
		return fmt.Errorf("published path %q is not normalized and relative", name)
	}
	for _, part := range strings.Split(name, "/") {
		if part == "" || part == "." || part == ".." {
			return fmt.Errorf("published path %q escapes the bundle", name)
		}
	}
	return nil
}

func checkBundleDigests(byPath map[string]BundleFile, digests map[string]string) error {
	for name, file := range byPath {
		want, ok := digests[name]
		if !ok {
			return fmt.Errorf("browser bundle audit: published file %s has no manifest digest", name)
		}
		if len(want) != 64 {
			return fmt.Errorf("browser bundle audit: manifest digest for %s is not SHA-256 hex", name)
		}
		sum := sha256.Sum256(file.Bytes)
		if hex.EncodeToString(sum[:]) != want {
			return fmt.Errorf("browser bundle audit: manifest digest mismatch for %s", name)
		}
	}
	for name := range digests {
		if _, ok := byPath[name]; !ok {
			return fmt.Errorf("browser bundle audit: manifest lists unpublished file %s", name)
		}
	}
	return nil
}

func classifyBundle(byPath map[string]BundleFile, table string) (scripts, maps []string, err error) {
	for name := range byPath {
		switch {
		case name == table:
			continue
		case strings.HasSuffix(name, ".js"):
			scripts = append(scripts, name)
		case strings.HasSuffix(name, ".js.map"):
			module := strings.TrimSuffix(name, ".map")
			if _, ok := byPath[module]; !ok {
				return nil, nil, fmt.Errorf("browser bundle audit: orphan source map %s with no published script", name)
			}
			maps = append(maps, name)
		default:
			return nil, nil, fmt.Errorf("browser bundle audit: %s is an unaccounted published file", name)
		}
	}
	sort.Strings(scripts)
	sort.Strings(maps)
	return scripts, maps, nil
}

func parseBundleMap(file BundleFile) (decodedSourceMap, [][]mapSegment, error) {
	parsed, err := parseSourceMap(file.Bytes)
	if err != nil {
		return decodedSourceMap{}, nil, fmt.Errorf("browser bundle audit: %s: %w", file.Path, err)
	}
	script := strings.TrimSuffix(file.Path, ".map")
	if parsed.File != path.Base(script) {
		return decodedSourceMap{}, nil, fmt.Errorf("browser bundle audit: %s names file %q, want %q", file.Path, parsed.File, path.Base(script))
	}
	if len(parsed.Sources) == 0 {
		return decodedSourceMap{}, nil, fmt.Errorf("browser bundle audit: %s carries no sources", file.Path)
	}
	for _, source := range parsed.Sources {
		if strings.Contains(source, "://") || strings.HasPrefix(source, "//") || strings.HasPrefix(source, "data:") {
			return decodedSourceMap{}, nil, fmt.Errorf("browser bundle audit: %s names remote source %q", file.Path, source)
		}
	}
	segments, err := decodeMappings(parsed.Mappings)
	if err != nil {
		return decodedSourceMap{}, nil, fmt.Errorf("browser bundle audit: %s: %w", file.Path, err)
	}
	for _, line := range segments {
		for _, segment := range line {
			if segment.Source >= len(parsed.Sources) {
				return decodedSourceMap{}, nil, fmt.Errorf("browser bundle audit: %s segment cites source %d of %d", file.Path, segment.Source, len(parsed.Sources))
			}
			if segment.Name >= len(parsed.Names) {
				return decodedSourceMap{}, nil, fmt.Errorf("browser bundle audit: %s segment cites name %d of %d", file.Path, segment.Name, len(parsed.Names))
			}
		}
	}
	return parsed, segments, nil
}

func auditBundleScript(script BundleFile, paired bundleMap, byPath map[string]BundleFile, secrets []string) error {
	if len(script.Bytes) == 0 {
		return fmt.Errorf("browser bundle audit: published script %s is empty", script.Path)
	}
	scan, err := ScanModule(script.Bytes)
	if err != nil {
		return fmt.Errorf("browser bundle audit: %s cannot be inspected structurally: %w", script.Path, err)
	}
	lines := splitLines(script.Bytes)
	trailer := "//# sourceMappingURL=" + path.Base(script.Path) + ".map\n"
	if !strings.HasSuffix(string(script.Bytes), trailer) {
		return fmt.Errorf("browser bundle audit: %s lacks its %q trailer", script.Path, strings.TrimSuffix(trailer, "\n"))
	}
	for _, edge := range scan.Edges {
		if !strings.HasPrefix(edge.Specifier, "./") && !strings.HasPrefix(edge.Specifier, "../") {
			return fmt.Errorf("browser bundle audit: %s:%s carries unaccounted edge %q: published scripts import only bundled chunks", script.Path, edge.Pos, edge.Specifier)
		}
		if strings.ContainsAny(edge.Specifier, "\\?#") {
			return fmt.Errorf("browser bundle audit: %s:%s carries unaccounted edge %q", script.Path, edge.Pos, edge.Specifier)
		}
		resolved := path.Join(path.Dir(script.Path), edge.Specifier)
		target, ok := byPath[resolved]
		if !ok || !strings.HasSuffix(target.Path, ".js") {
			return fmt.Errorf("browser bundle audit: %s:%s carries unaccounted edge %q: no published chunk %s", script.Path, edge.Pos, edge.Specifier, resolved)
		}
	}
	for _, finding := range scan.Findings {
		origin := mapOrigin(paired.parsed.Sources, paired.segments, finding.Pos.Line, finding.Pos.Column)
		operation := finding.Operation
		if operation == "import()" || operation == "require" {
			return fmt.Errorf("browser bundle audit: %s:%s carries unaccounted edge %s%s", script.Path, finding.Pos, operation, origin)
		}
		return fmt.Errorf("browser bundle audit: %s:%s forbids host operation %s%s", script.Path, finding.Pos, operation, origin)
	}
	for lineIndex, line := range paired.segments {
		for _, segment := range line {
			if err := checkBounds(script.Path, lines, lineIndex+1, segment.GenColumn); err != nil {
				return fmt.Errorf("browser bundle audit: %s.map: %w", script.Path, err)
			}
		}
	}
	return scanBundleSecrets(script, secrets)
}

func auditBundleTable(table BundleFile, secrets []string) error {
	var decoded any
	if err := json.Unmarshal(table.Bytes, &decoded); err != nil {
		return fmt.Errorf("browser bundle audit: diagnostic table %s is not JSON: %w", table.Path, err)
	}
	if err := checkTableKeys(decoded); err != nil {
		return fmt.Errorf("browser bundle audit: diagnostic table %s %w", table.Path, err)
	}
	return scanBundleSecrets(table, secrets)
}

func checkTableKeys(value any) error {
	switch node := value.(type) {
	case map[string]any:
		for key, child := range node {
			if forbiddenTableKeys[key] {
				return fmt.Errorf("carries excluded diagnostic data %q", key)
			}
			if err := checkTableKeys(child); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range node {
			if err := checkTableKeys(child); err != nil {
				return err
			}
		}
	}
	return nil
}

func scanBundleSecrets(file BundleFile, secrets []string) error {
	if err := scanCanaryList(file.Path, file.Bytes, nil, secretCanaries); err != nil {
		return err
	}
	text := string(file.Bytes)
	for _, secret := range secrets {
		if secret == "" {
			continue
		}
		if offset := strings.Index(text, secret); offset >= 0 {
			// The secret itself is never quoted: diagnostics must not
			// become a second leak channel.
			return fmt.Errorf("browser audit: %s:%s leaks build-known secret material", file.Path, offsetPosition(file.Bytes, offset))
		}
	}
	return nil
}
