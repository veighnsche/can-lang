package driver

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

// BuildInputs is semantic input identity. Absolute workspace paths, timestamps,
// random staging names and process IDs never contribute to the generation ID.
type BuildInputs struct {
	Source       string `json:"source"`
	Dependencies string `json:"dependencies"`
	Catalogue    string `json:"catalogue"`
	Compiler     string `json:"compiler"`
	Runtime      string `json:"runtime"`
	Options      string `json:"options"`
}

type OutputArtifact struct {
	Path  string
	Bytes []byte
	// Imports are complete structured emission edges, including import type.
	// Only verified distribution runtime artifacts may carry NativeImports.
	Imports       []string
	NativeImports []string
	Runtime       bool
}

type OutputManifest struct {
	SchemaVersion int                 `json:"schemaVersion"`
	Kind          string              `json:"kind"`
	BuildID       string              `json:"buildID"`
	Inputs        BuildInputs         `json:"inputs"`
	Entry         string              `json:"entry"`
	Files         map[string]string   `json:"files"`
	Imports       map[string][]string `json:"imports"`
	NativeImports map[string][]string `json:"nativeImports"`
}

type PreparedOutput struct {
	validated     bool
	manifest      OutputManifest
	files         map[string][]byte
	manifestBytes []byte
}

func (p *PreparedOutput) BuildID() string      { return p.manifest.BuildID }
func (p *PreparedOutput) ManifestJSON() []byte { return append([]byte(nil), p.manifestBytes...) }

var digestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
var outputComponent = regexp.MustCompile(`^[A-Za-z0-9_$.-]+$`)
var deviceName = regexp.MustCompile(`(?i)^(con|prn|aux|nul|com[1-9]|lpt[1-9])(?:\.|$)`)

func outputPath(name string) error {
	if name == "" || path.IsAbs(name) || path.Clean(name) != name || strings.Contains(name, "\\") {
		return fmt.Errorf("output path must be normalized and relative: %q", name)
	}
	for _, part := range strings.Split(name, "/") {
		if part == "." || part == ".." || !outputComponent.MatchString(part) || strings.HasSuffix(part, ".") || deviceName.MatchString(part) {
			return fmt.Errorf("unsafe output component: %q", part)
		}
	}
	if strings.EqualFold(strings.Split(name, "/")[0], "manifest.json") || strings.HasPrefix(name, ".") {
		return fmt.Errorf("reserved output path: %s", name)
	}
	return nil
}
func hashBytes(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func manifestID(manifest OutputManifest) (string, error) {
	manifest.BuildID = ""
	data, err := json.Marshal(manifest)
	if err != nil {
		return "", err
	}
	return hashBytes(append([]byte("can-output-generation-v1\x00"), data...)), nil
}
func sortedOutputKeys[V any](values map[string]V) []string {
	out := make([]string, 0, len(values))
	for key := range values {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

// PrepareOutput validates the complete artifact inventory before any output tree
// is claimed. Callers supply compiler-generated modules and hash-verified private
// runtime bytes; authored TypeScript and ambient package discovery are not inputs.
func PrepareOutput(inputs BuildInputs, entry string, artifacts []OutputArtifact) (*PreparedOutput, error) {
	for _, id := range []string{inputs.Source, inputs.Dependencies, inputs.Catalogue, inputs.Compiler, inputs.Runtime, inputs.Options} {
		if !digestPattern.MatchString(id) {
			return nil, fmt.Errorf("every semantic build input requires a SHA-256 identity")
		}
	}
	m := OutputManifest{SchemaVersion: 1, Kind: "can.output-generation", Inputs: inputs, Entry: entry, Files: map[string]string{}, Imports: map[string][]string{}, NativeImports: map[string][]string{}}
	files := map[string][]byte{}
	claims := map[string]string{}
	for _, a := range artifacts {
		if err := outputPath(a.Path); err != nil {
			return nil, err
		}
		// Claim directories too: case aliases and file/directory conflicts must fail
		// even when the development filesystem happens to be case sensitive.
		parts := strings.Split(a.Path, "/")
		for i := range parts {
			spelling := strings.Join(parts[:i+1], "/")
			key := strings.ToLower(spelling)
			kind := "dir:"
			if i == len(parts)-1 {
				kind = "file:"
			}
			expected := kind + spelling
			if prior, ok := claims[key]; ok && (prior != expected || kind == "file:") {
				return nil, fmt.Errorf("output path collision: %s", a.Path)
			}
			claims[key] = expected
		}
		if !strings.HasSuffix(a.Path, ".ts") && (len(a.Imports) != 0 || len(a.NativeImports) != 0) {
			return nil, fmt.Errorf("non-module artifact has imports")
		}
		if len(a.NativeImports) != 0 && (!a.Runtime || !strings.HasPrefix(a.Path, "runtime/")) {
			return nil, fmt.Errorf("native imports belong only to verified private runtime modules")
		}
		m.Files[a.Path] = hashBytes(a.Bytes)
		files[a.Path] = append([]byte(nil), a.Bytes...)
		if strings.HasSuffix(a.Path, ".ts") {
			m.Imports[a.Path] = append([]string{}, a.Imports...)
			sort.Strings(m.Imports[a.Path])
			if len(a.NativeImports) != 0 {
				m.NativeImports[a.Path] = append([]string{}, a.NativeImports...)
				sort.Strings(m.NativeImports[a.Path])
			}
		}
	}
	if err := validateArtifactPayloads(files); err != nil {
		return nil, err
	}
	if err := validateOutputManifest(m, false); err != nil {
		return nil, err
	}
	var err error
	m.BuildID, err = manifestID(m)
	if err != nil {
		return nil, err
	}
	encoded, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, err
	}
	encoded = append(encoded, '\n')
	return &PreparedOutput{manifest: m, files: files, manifestBytes: encoded}, nil
}

func validateOutputManifest(m OutputManifest, checkID bool) error {
	if m.SchemaVersion != 1 || m.Kind != "can.output-generation" {
		return fmt.Errorf("unsupported generation manifest")
	}
	for _, digest := range []string{m.Inputs.Source, m.Inputs.Dependencies, m.Inputs.Catalogue, m.Inputs.Compiler, m.Inputs.Runtime, m.Inputs.Options} {
		if !digestPattern.MatchString(digest) {
			return fmt.Errorf("invalid semantic build input digest")
		}
	}
	claims := map[string]string{}
	for name := range m.Files {
		parts := strings.Split(name, "/")
		for i := range parts {
			spelling := strings.Join(parts[:i+1], "/")
			kind := "dir:"
			if i == len(parts)-1 {
				kind = "file:"
			}
			key := strings.ToLower(spelling)
			if prior, ok := claims[key]; ok && prior != kind+spelling {
				return fmt.Errorf("generation path collision")
			}
			claims[key] = kind + spelling
		}
	}
	if m.Files[m.Entry] == "" || !strings.HasSuffix(m.Entry, ".ts") {
		return fmt.Errorf("missing generated entry module")
	}
	for name, digest := range m.Files {
		if err := outputPath(name); err != nil {
			return err
		}
		if !digestPattern.MatchString(digest) {
			return fmt.Errorf("invalid artifact digest")
		}
	}
	for from, imports := range m.Imports {
		if m.Files[from] == "" || !strings.HasSuffix(from, ".ts") {
			return fmt.Errorf("import owner is not a module")
		}
		seen := map[string]bool{}
		for _, target := range imports {
			if (!strings.HasPrefix(target, "./") && !strings.HasPrefix(target, "../")) || !strings.HasSuffix(target, ".ts") || strings.ContainsAny(target, "\\?#") {
				return fmt.Errorf("module import must name an exact relative .ts file: %q", target)
			}
			clean := path.Clean(target)
			if !strings.HasPrefix(clean, "../") {
				clean = "./" + clean
			}
			if clean != target {
				return fmt.Errorf("noncanonical module import")
			}
			resolved := path.Join(path.Dir(from), target)
			if err := outputPath(resolved); err != nil {
				return err
			}
			if m.Files[resolved] == "" {
				return fmt.Errorf("missing imported module %s from %s", resolved, from)
			}
			if seen[target] {
				return fmt.Errorf("duplicate module edge")
			}
			seen[target] = true
		}
	}
	for name := range m.Files {
		if strings.HasSuffix(name, ".ts") {
			if _, ok := m.Imports[name]; !ok {
				return fmt.Errorf("module lacks its complete import inventory")
			}
		}
	}
	for from, imports := range m.NativeImports {
		if m.Files[from] == "" || !strings.HasPrefix(from, "runtime/") {
			return fmt.Errorf("invalid native import owner")
		}
		for _, target := range imports {
			switch target {
			case "node:assert", "node:buffer", "node:crypto", "node:fs", "node:path", "node:util":
			default:
				return fmt.Errorf("unadmitted native runtime import %q", target)
			}
		}
	}
	if checkID {
		id, err := manifestID(m)
		if err != nil || id != m.BuildID {
			return fmt.Errorf("generation content identity mismatch")
		}
	}
	return nil
}

func validateArtifactPayloads(files map[string][]byte) error {
	for name, data := range files {
		if (strings.HasSuffix(name, ".ts") || strings.HasSuffix(name, ".map")) && !utf8.Valid(data) {
			return fmt.Errorf("generated text is not UTF-8")
		}
		if strings.HasPrefix(name, "assets/") {
			parts := strings.Split(name, "/")
			if len(parts) != 3 || parts[1] != hashBytes(data) {
				return fmt.Errorf("asset path does not bind its content digest")
			}
		}
		if strings.HasSuffix(name, ".ts.map") {
			if _, ok := files[strings.TrimSuffix(name, ".map")]; !ok {
				return fmt.Errorf("source map lacks generated module")
			}
			var sourceMap struct {
				Version        int      `json:"version"`
				File           string   `json:"file"`
				Sources        []string `json:"sources"`
				SourcesContent []string `json:"sourcesContent"`
				Names          []string `json:"names"`
				Mappings       string   `json:"mappings"`
				SourceRoot     string   `json:"sourceRoot,omitempty"`
			}
			if err := decodeOutput(data, &sourceMap); err != nil || sourceMap.Version != 3 || len(sourceMap.Sources) != len(sourceMap.SourcesContent) {
				return fmt.Errorf("invalid generated source map")
			}
			expected := path.Base(strings.TrimSuffix(name, ".map"))
			if sourceMap.File != expected {
				return fmt.Errorf("source map names a different generated module")
			}
		}
	}
	return nil
}
