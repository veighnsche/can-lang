package driver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/browser"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/project"
	"github.com/veighnsche/can-lang/distribution"
)

// Browser bundle output layout (UP15). The bundle entry and its chunks live
// beside the compiler-produced asset manifest under browser/; the immutable
// diagnostic table and the I-6 browser manifest are data artifacts bound by
// the generation manifest like every other published byte.
const (
	browserBundleDirectory = "browser"
	browserBundleTable     = "diagnostics/table.json"
	browserBundleManifest  = "browser/manifest.json"
	browserBundleTool      = "tools/runtime/browser-bundle.ts"
)

// browserBundleOutputs is one normalized, audited browser bundle: logical
// generation path to published bytes for the final digest JavaScript, one
// source map per script, and the immutable diagnostic table.
type browserBundleOutputs struct {
	entry string
	files map[string][]byte
}

// browserTablePlaceholder is the exact UP11 placeholder table line the
// compiler-owned browser entry carries. UP15 replaces it with the checked
// sealed table (one line for one line, so sealed mappings keep their
// coordinates) and publishes the same bytes as a verified asset. The match
// is exact and the replacement is mandatory: an entry without the known
// placeholder fails instead of shipping an unknown table state.
const browserTablePlaceholder = `const $canTable = Object.freeze({index: Object.freeze({schemaVersion: 1, kind: "can.source-index", sources: Object.freeze([]), modules: Object.freeze([])}), maps: Object.freeze({})});`

// sealBrowserEntryTable swaps the placeholder table line in the browser
// root for the frozen checked table. It runs before the asset manifest and
// the pre-bundle audit so both bind the exact shipped bytes.
func sealBrowserEntryTable(artifacts []ir.Artifact, table []byte) ([]ir.Artifact, error) {
	if len(table) == 0 {
		return nil, fmt.Errorf("browser entry sealing requires the checked table")
	}
	sealed := append([]ir.Artifact{}, artifacts...)
	found := false
	for i := range sealed {
		if sealed[i].Path != browser.BrowserEntry {
			continue
		}
		lines := strings.Split(string(sealed[i].Bytes), "\n")
		for j, line := range lines {
			if line != browserTablePlaceholder {
				continue
			}
			if found {
				return nil, fmt.Errorf("browser entry carries the table placeholder twice")
			}
			found = true
			lines[j] = "const $canTable = Object.freeze(" + string(table) + ");"
		}
		if !found {
			return nil, fmt.Errorf("browser entry lacks the known table placeholder")
		}
		sealed[i].Bytes = []byte(strings.Join(lines, "\n"))
	}
	if !found {
		return nil, fmt.Errorf("browser sealing lacks the %s root", browser.BrowserEntry)
	}
	return sealed, nil
}

// buildBrowserBundle bundles the audited pre-bundle tree with the native
// pinned bundler and returns the publishable bundle artifacts: final digest
// JavaScript plus maps, the sealed diagnostic table, and the I-6 browser
// manifest. It runs after the pre-bundle graph audit and before output
// preparation, so any bundle, audit, or shaping failure prevents staging
// and publication. The same inputs always yield identical bytes.
func (r *Runtime) buildBrowserBundle(ctx context.Context, graph *project.Graph, inputs BuildInputs, toolchain browserToolchain, artifacts []ir.Artifact, table []byte) ([]ir.Artifact, error) {
	if len(table) == 0 {
		return nil, fmt.Errorf("browser bundling requires the sealed diagnostic table")
	}
	raw, staged, sourceDir, outDir, cleanup, err := r.invokeBrowserBundler(ctx, toolchain, artifacts)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	bundle, err := normalizeBrowserBundle(sourceDir, outDir, staged, raw)
	if err != nil {
		return nil, err
	}
	bundle.files[browserBundleTable] = append([]byte(nil), table...)
	if err := auditBrowserBundle(bundle); err != nil {
		return nil, err
	}
	manifest, err := browserManifestBytes(graph, inputs, toolchain, bundle)
	if err != nil {
		return nil, err
	}
	ordered := append([]string{}, sortedOutputKeys(bundle.files)...)
	assembled := make([]ir.Artifact, 0, len(ordered)+1)
	for _, name := range ordered {
		assembled = append(assembled, ir.Artifact{Path: name, Bytes: bundle.files[name]})
	}
	return append(assembled, ir.Artifact{Path: browserBundleManifest, Bytes: manifest}), nil
}

// browserToolchain pins the exact bundler: the distribution target identity
// plus the Bun version, revision, and sidecar digest. The tool refuses any
// other runtime before invoking Bun.build.
type browserToolchain struct {
	Target   string
	Version  string
	Revision string
	SHA256   string
	Compiler string
	Runtime  string
}

func browserBundlerToolchain(compiler, runtime string) browserToolchain {
	target := distribution.PinnedTarget()
	return browserToolchain{Target: target.TargetID, Version: target.Runtime.Version, Revision: target.Runtime.Revision, SHA256: target.Runtime.SHA256, Compiler: compiler, Runtime: runtime}
}

// browserTreeFile is one staged bundler input: a generation-logical path
// with the exact audited bytes.
type browserTreeFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type browserBundleRequest struct {
	SchemaVersion int               `json:"schemaVersion"`
	Kind          string            `json:"kind"`
	SourceDir     string            `json:"sourceDir"`
	Entry         string            `json:"entry"`
	OutDir        string            `json:"outDir"`
	Files         []browserTreeFile `json:"files"`
	Expected      struct {
		Version  string `json:"version"`
		Revision string `json:"revision"`
	} `json:"expected"`
}

type browserBundleReport struct {
	SchemaVersion int    `json:"schemaVersion"`
	Kind          string `json:"kind"`
	RequestSHA256 string `json:"requestSHA256"`
	Bun           struct {
		Version  string `json:"version"`
		Revision string `json:"revision"`
	} `json:"bun"`
	Files []struct {
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
		Bytes  int    `json:"bytes"`
	} `json:"files"`
}

// invokeBrowserBundler stages exactly the .ts modules of the audited
// generation (generated plus private runtime; maps, index, and data assets
// are not bundler inputs), runs the maintained native bundler tool, and
// returns the raw emitted bytes keyed by outDir-relative path. Callers own
// normalization and audit; nothing here is publishable yet.
func (r *Runtime) invokeBrowserBundler(ctx context.Context, toolchain browserToolchain, artifacts []ir.Artifact) (map[string][]byte, map[string][]byte, string, string, func(), error) {
	failed := func() {}
	modules := map[string][]byte{}
	inventories := map[string][]string{}
	runtimeFlags := map[string]bool{}
	for _, artifact := range artifacts {
		if !strings.HasSuffix(artifact.Path, ".ts") {
			continue
		}
		if _, dup := modules[artifact.Path]; dup {
			return nil, nil, "", "", failed, fmt.Errorf("browser bundler input %s is duplicated", artifact.Path)
		}
		modules[artifact.Path] = artifact.Bytes
		inventories[artifact.Path] = artifact.Imports
		runtimeFlags[artifact.Path] = artifact.Runtime
	}
	if modules[browser.BrowserEntry] == nil {
		return nil, nil, "", "", failed, fmt.Errorf("browser bundler input lacks the %s root", browser.BrowserEntry)
	}
	modules, err := applyBrowserOverlay(modules, inventories, runtimeFlags)
	if err != nil {
		return nil, nil, "", "", failed, err
	}
	sourceDir, err := os.MkdirTemp("", "can-browser-stage-")
	if err != nil {
		return nil, nil, "", "", failed, err
	}
	cleanup := func() { os.RemoveAll(sourceDir) }
	// The bundler records module paths relative to the output directory in
	// banner comments, so the output lives inside the staged tree under a
	// fixed name: every recorded path is then identical across builds.
	outDir := filepath.Join(sourceDir, "can-bundle-out")
	if err := os.MkdirAll(outDir, 0700); err != nil {
		cleanup()
		return nil, nil, "", "", failed, err
	}
	// The bundler resolves symlinked temporary parents (notably /var and
	// /tmp on macOS) when recording map sources. Canonicalize the root
	// first so emitted sources resolve back into the staged tree.
	if sourceDir, err = filepath.EvalSymlinks(sourceDir); err != nil {
		cleanup()
		return nil, nil, "", "", failed, err
	}
	outDir = filepath.Join(sourceDir, "can-bundle-out")
	request := browserBundleRequest{SchemaVersion: 1, Kind: "can.browser-bundle-request", SourceDir: sourceDir, Entry: browser.BrowserEntry, OutDir: outDir}
	request.Expected.Version = toolchain.Version
	request.Expected.Revision = toolchain.Revision
	for _, name := range sortedOutputKeys(modules) {
		full := filepath.Join(sourceDir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0700); err != nil {
			cleanup()
			return nil, nil, "", "", failed, err
		}
		if err := os.WriteFile(full, modules[name], 0600); err != nil {
			cleanup()
			return nil, nil, "", "", failed, err
		}
		request.Files = append(request.Files, browserTreeFile{name, hashBytes(modules[name])})
	}
	encoded, err := json.Marshal(request)
	if err != nil {
		cleanup()
		return nil, nil, "", "", failed, err
	}
	var stdout, stderr bytes.Buffer
	if err := r.RunTool(ctx, browserBundleTool, nil, nil, bytes.NewReader(encoded), &stdout, &stderr); err != nil {
		cleanup()
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			return nil, nil, "", "", failed, fmt.Errorf("browser bundling failed: %w", err)
		}
		return nil, nil, "", "", failed, fmt.Errorf("browser bundling failed: %w: %s", err, truncateBundleLog(detail))
	}
	var report browserBundleReport
	if err := decodeOutput(stdout.Bytes(), &report); err != nil || report.SchemaVersion != 1 || report.Kind != "can.browser-bundle-report" || report.RequestSHA256 != hashBytes(encoded) {
		cleanup()
		return nil, nil, "", "", failed, fmt.Errorf("invalid browser bundle report")
	}
	if report.Bun.Version != toolchain.Version || report.Bun.Revision != toolchain.Revision {
		cleanup()
		return nil, nil, "", "", failed, fmt.Errorf("browser bundler toolchain mismatch")
	}
	if len(report.Files) == 0 {
		cleanup()
		return nil, nil, "", "", failed, fmt.Errorf("browser bundler emitted no files")
	}
	raw := map[string][]byte{}
	for _, file := range report.Files {
		if file.Path == "" || filepath.IsAbs(file.Path) || filepath.ToSlash(filepath.Clean(file.Path)) != file.Path || strings.HasPrefix(file.Path, "../") || !digestPattern.MatchString(file.SHA256) || file.Bytes < 0 {
			cleanup()
			return nil, nil, "", "", failed, fmt.Errorf("browser bundler reported an invalid path %q", file.Path)
		}
		if _, dup := raw[file.Path]; dup {
			cleanup()
			return nil, nil, "", "", failed, fmt.Errorf("browser bundler reported %s twice", file.Path)
		}
		data, err := os.ReadFile(filepath.Join(outDir, filepath.FromSlash(file.Path)))
		if err != nil {
			cleanup()
			return nil, nil, "", "", failed, fmt.Errorf("browser bundler output %s is unreadable: %w", file.Path, err)
		}
		if len(data) != file.Bytes || hashBytes(data) != file.SHA256 {
			cleanup()
			return nil, nil, "", "", failed, fmt.Errorf("browser bundler output %s changed after emission", file.Path)
		}
		raw[file.Path] = data
	}
	return raw, modules, sourceDir, outDir, cleanup, nil
}

// applyBrowserOverlay redirects every staged edge that resolves to an
// overlaid canonical runtime module to its sealed alternate, using the
// exact substitution the pre-bundle audit models (browser.OverlayShipping).
// The bundler therefore loads the inspected bodies with no resolve hooks:
// the staged bytes are the shipped sources. An overlaid edge whose
// alternate is absent from the generation is left untouched, exactly as
// the audit inspects it as-is; the bundled canonical then fails the
// post-bundle audit on its own host edges.
func applyBrowserOverlay(modules map[string][]byte, inventories map[string][]string, runtimeFlags map[string]bool) (map[string][]byte, error) {
	rewritten := map[string][]byte{}
	for _, name := range sortedOutputKeys(modules) {
		redirects := map[string]string{}
		for _, spec := range inventories[name] {
			if !strings.HasPrefix(spec, "./") && !strings.HasPrefix(spec, "../") {
				continue
			}
			resolved := path.Join(path.Dir(name), spec)
			target, ok := modules[resolved]
			if !ok || target == nil {
				continue
			}
			shipping := browser.OverlayShipping(resolved, runtimeFlags[resolved])
			if shipping == resolved || shipping == name || modules[shipping] == nil {
				continue
			}
			fresh, err := relativeModuleSpecifier(path.Dir(name), shipping)
			if err != nil {
				return nil, err
			}
			redirects[spec] = fresh
		}
		if len(redirects) == 0 {
			rewritten[name] = modules[name]
			continue
		}
		scan, err := browser.ScanModule(modules[name])
		if err != nil {
			return nil, fmt.Errorf("browser overlay cannot inspect %s structurally: %w", name, err)
		}
		rewritten[name], err = rewriteModuleSpecifiers(name, modules[name], scan, redirects)
		if err != nil {
			return nil, err
		}
	}
	return rewritten, nil
}

// rewriteModuleSpecifiers splices replacement specifiers into the lexed
// static edge positions, from the end of the module backward so offsets
// stay valid. Every redirected specifier must appear lexed at least once;
// anything else fails closed rather than shipping a half-substituted tree.
func rewriteModuleSpecifiers(name string, data []byte, scan browser.ModuleScan, redirects map[string]string) ([]byte, error) {
	type splice struct {
		offset  int
		length  int
		current string
	}
	var splices []splice
	seen := map[string]bool{}
	for _, edge := range scan.Edges {
		fresh, ok := redirects[edge.Specifier]
		if !ok {
			continue
		}
		seen[edge.Specifier] = true
		start := edge.Pos.Offset + 1
		if start < 1 || start+len(edge.Specifier) > len(data) || string(data[start:start+len(edge.Specifier)]) != edge.Specifier {
			return nil, fmt.Errorf("browser overlay cannot locate edge %q in %s", edge.Specifier, name)
		}
		quote := data[edge.Pos.Offset]
		if quote != '"' && quote != '\'' {
			return nil, fmt.Errorf("browser overlay cannot locate edge %q in %s", edge.Specifier, name)
		}
		splices = append(splices, splice{start, len(edge.Specifier), fresh})
	}
	for spec := range redirects {
		if !seen[spec] {
			return nil, fmt.Errorf("browser overlay found no lexed edge %q in %s", spec, name)
		}
	}
	out := append([]byte(nil), data...)
	for i := len(splices) - 1; i >= 0; i-- {
		cut := splices[i]
		next := make([]byte, 0, len(out)-cut.length+len(cut.current))
		next = append(next, out[:cut.offset]...)
		next = append(next, cut.current...)
		out = append(next, out[cut.offset+cut.length:]...)
	}
	return out, nil
}

// relativeModuleSpecifier renders target as a canonical relative import
// from the importing directory: dot-prefixed, slash-separated, with no
// redundant segments.
func relativeModuleSpecifier(fromDir, target string) (string, error) {
	if fromDir == "" || target == "" || !strings.HasSuffix(target, ".ts") {
		return "", fmt.Errorf("browser overlay cannot redirect to %q", target)
	}
	from := strings.Split(path.Clean(fromDir), "/")
	if path.Clean(fromDir) == "." {
		from = []string{}
	}
	to := strings.Split(path.Clean(target), "/")
	shared := 0
	for shared < len(from) && shared < len(to) && from[shared] == to[shared] {
		shared++
	}
	var parts []string
	for range from[shared:] {
		parts = append(parts, "..")
	}
	parts = append(parts, to[shared:]...)
	spec := path.Join(parts...)
	if spec == "" || spec == "." {
		return "", fmt.Errorf("browser overlay cannot redirect to %q", target)
	}
	if !strings.HasPrefix(spec, ".") {
		spec = "./" + spec
	}
	if !strings.HasPrefix(spec, "./") && !strings.HasPrefix(spec, "../") {
		return "", fmt.Errorf("browser overlay cannot redirect to %q", target)
	}
	return spec, nil
}

func truncateBundleLog(detail string) string {
	if len(detail) > 2048 {
		return detail[:2048] + "\n[truncated]"
	}
	return detail
}

// normalizeBrowserBundle shapes raw bundler output into publishable bytes:
// every script loses Bun's session debugId and gains its canonical map
// trailer, every map is rewritten with its published file name and staged
// logical sources in canonical JSON, and outputs are assigned their logical
// generation paths. Anything the bundler invented outside the staged
// inventory (shims, remotes, orphans) fails closed here.
func normalizeBrowserBundle(sourceDir, outDir string, staged map[string][]byte, raw map[string][]byte) (browserBundleOutputs, error) {
	entryBase := strings.TrimSuffix(path.Base(browser.BrowserEntry), ".ts")
	entryName := entryBase + ".js"
	if _, ok := raw[entryName]; !ok {
		return browserBundleOutputs{}, fmt.Errorf("browser bundler omitted the %s entry script", entryName)
	}
	scripts := map[string][]byte{}
	maps := map[string][]byte{}
	for name, data := range raw {
		switch {
		case strings.HasSuffix(name, ".js.map"):
			maps[name] = data
		case strings.HasSuffix(name, ".js"):
			scripts[name] = data
		default:
			return browserBundleOutputs{}, fmt.Errorf("browser bundler emitted unaccounted file %s", name)
		}
	}
	bundle := browserBundleOutputs{entry: browserBundleDirectory + "/" + entryName, files: map[string][]byte{}}
	for name, data := range scripts {
		mapName := name + ".map"
		paired, ok := maps[mapName]
		if !ok {
			return browserBundleOutputs{}, fmt.Errorf("bundled script %s lacks its source map", name)
		}
		script, err := normalizeBundleScript(name, data)
		if err != nil {
			return browserBundleOutputs{}, err
		}
		normalized, err := normalizeBundleMap(sourceDir, outDir, staged, name, paired)
		if err != nil {
			return browserBundleOutputs{}, err
		}
		bundle.files[browserBundleDirectory+"/"+name] = script
		bundle.files[browserBundleDirectory+"/"+mapName] = normalized
	}
	for name := range maps {
		if _, ok := scripts[strings.TrimSuffix(name, ".map")]; !ok {
			return browserBundleOutputs{}, fmt.Errorf("bundled source map %s has no published script", name)
		}
	}
	return bundle, nil
}

// normalizeBundleScript strips Bun's per-session debugId correlation line
// (which varies across identical builds) and binds the script to its
// published map with the one canonical trailer the post-bundle audit
// requires.
func normalizeBundleScript(name string, data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("bundled script %s is empty", name)
	}
	text := strings.TrimSuffix(string(data), "\n")
	lines := strings.Split(text, "\n")
	for len(lines) != 0 {
		last := lines[len(lines)-1]
		switch {
		case strings.HasPrefix(last, "//# debugId="):
			lines = lines[:len(lines)-1]
		case strings.HasPrefix(last, "//# sourceMappingURL="):
			lines = lines[:len(lines)-1]
		default:
			goto stripped
		}
	}
stripped:
	if len(lines) == 0 {
		return nil, fmt.Errorf("bundled script %s holds only bundler trailers", name)
	}
	lines = append(lines, "//# sourceMappingURL="+path.Base(name)+".map")
	return []byte(strings.Join(lines, "\n") + "\n"), nil
}

// bundleMapJSON is the exact published map surface: version 3 with a bound
// file name, staged-logical sources, and verbatim content, names, and
// mappings. debugId is accepted from the bundler and dropped; anything else
// unknown fails closed so no nondeterministic surface ships silently.
type bundleMapJSON struct {
	Version        int      `json:"version"`
	File           string   `json:"file,omitempty"`
	Sources        []string `json:"sources"`
	SourcesContent []string `json:"sourcesContent"`
	Names          []string `json:"names"`
	Mappings       string   `json:"mappings"`
	DebugID        string   `json:"debugId,omitempty"`
}

func normalizeBundleMap(sourceDir, outDir string, staged map[string][]byte, script string, data []byte) ([]byte, error) {
	var decoded bundleMapJSON
	parser := json.NewDecoder(bytes.NewReader(data))
	parser.DisallowUnknownFields()
	if err := parser.Decode(&decoded); err != nil {
		return nil, fmt.Errorf("bundled map %s.map is not a known source map: %w", script, err)
	}
	if parser.More() {
		return nil, fmt.Errorf("bundled map %s.map carries trailing data", script)
	}
	if decoded.Version != 3 || len(decoded.Sources) == 0 || len(decoded.Sources) != len(decoded.SourcesContent) || decoded.Mappings == "" {
		return nil, fmt.Errorf("bundled map %s.map is incomplete", script)
	}
	logical := make([]string, 0, len(decoded.Sources))
	for _, source := range decoded.Sources {
		resolved, err := resolveBundleSource(sourceDir, outDir, source)
		if err != nil {
			return nil, fmt.Errorf("bundled map %s.map names %q: %w", script, source, err)
		}
		if staged[resolved] == nil {
			return nil, fmt.Errorf("bundled map %s.map names unstaged source %q", script, resolved)
		}
		logical = append(logical, resolved)
	}
	normalized := struct {
		Version        int      `json:"version"`
		File           string   `json:"file"`
		Sources        []string `json:"sources"`
		SourcesContent []string `json:"sourcesContent"`
		Names          []string `json:"names"`
		Mappings       string   `json:"mappings"`
	}{3, path.Base(script), logical, decoded.SourcesContent, nonNilStrings(decoded.Names), decoded.Mappings}
	return json.Marshal(normalized)
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

// resolveBundleSource maps one bundler-emitted source entry back to its
// staged logical path. Absolute entries and outDir-relative entries that
// land inside the staged tree resolve; remote URLs, data URLs, bare
// specifiers (bundler-injected shims such as node:util), and anything
// escaping the tree fail closed.
func resolveBundleSource(sourceDir, outDir, source string) (string, error) {
	if source == "" || strings.Contains(source, "://") || strings.HasPrefix(source, "//") || strings.HasPrefix(source, "data:") {
		return "", fmt.Errorf("remote map source is not a staged module")
	}
	candidate := source
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(outDir, filepath.FromSlash(candidate))
	}
	absolute, err := filepath.Abs(candidate)
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(absolute); err == nil {
		absolute = resolved
	}
	root, err := filepath.Abs(sourceDir)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, absolute)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("map source escapes the staged tree")
	}
	info, err := os.Lstat(absolute)
	if err != nil || !info.Mode().IsRegular() {
		return "", fmt.Errorf("map source is not a staged file")
	}
	return filepath.ToSlash(rel), nil
}

// sealBrowserDiagnosticTable publishes the immutable diagnostic table the
// sealed browser reporter consumes: the build-sealed source index plus the
// per-module maps that resolve checked Can origins at fault time, with no
// filesystem or fetch. The table carries locations only; native causes,
// raw input, stacks, and secrets never enter it.
func sealBrowserDiagnosticTable(artifacts []ir.Artifact) ([]byte, error) {
	var index json.RawMessage
	maps := map[string]json.RawMessage{}
	for _, artifact := range artifacts {
		switch {
		case artifact.Path == "diagnostics/source-index.json":
			index = append([]byte(nil), artifact.Bytes...)
		case strings.HasSuffix(artifact.Path, ".ts.map"):
			maps[strings.TrimSuffix(artifact.Path, ".map")] = append([]byte(nil), artifact.Bytes...)
		}
	}
	if len(index) == 0 {
		return nil, fmt.Errorf("browser diagnostic table requires the sealed source index")
	}
	var decoded struct {
		SchemaVersion int    `json:"schemaVersion"`
		Kind          string `json:"kind"`
		Sources       []struct {
			ID    string                    `json:"id"`
			Path  string                    `json:"path"`
			Spans map[string]diagnosticSpan `json:"spans"`
		} `json:"sources"`
		Modules []struct {
			Path     string              `json:"path"`
			Segments []diagnosticSegment `json:"segments"`
		} `json:"modules"`
	}
	parser := json.NewDecoder(bytes.NewReader(index))
	parser.DisallowUnknownFields()
	if err := parser.Decode(&decoded); err != nil || decoded.SchemaVersion != 1 || decoded.Kind != "can.source-index" {
		return nil, fmt.Errorf("browser diagnostic table index is invalid")
	}
	if len(decoded.Modules) == 0 {
		return nil, fmt.Errorf("browser diagnostic table index binds no modules")
	}
	bound := map[string]json.RawMessage{}
	for _, module := range decoded.Modules {
		data, ok := maps[module.Path]
		if !ok || len(data) == 0 {
			return nil, fmt.Errorf("browser diagnostic table lacks the %s map", module.Path)
		}
		var probe map[string]any
		if err := json.Unmarshal(data, &probe); err != nil {
			return nil, fmt.Errorf("browser diagnostic table map %s is invalid", module.Path)
		}
		bound[module.Path] = data
	}
	table := struct {
		SchemaVersion int                        `json:"schemaVersion"`
		Kind          string                     `json:"kind"`
		Index         json.RawMessage            `json:"index"`
		Maps          map[string]json.RawMessage `json:"maps"`
	}{1, "can.diagnostic-table", index, bound}
	return json.Marshal(table)
}

// auditBrowserBundle runs the UP10 post-bundle audit over the normalized
// bundle: digest-bound manifest projection, accounted chunk edges, bound
// source maps, a clean diagnostic table, and no secret canaries. Browser
// builds carry no build-known secrets: connections and endpoints fail the
// capability gate long before bundling.
func auditBrowserBundle(bundle browserBundleOutputs) error {
	assembled := browser.Bundle{Entry: bundle.entry, Table: browserBundleTable, Digests: map[string]string{}}
	for _, name := range sortedOutputKeys(bundle.files) {
		assembled.Files = append(assembled.Files, browser.BundleFile{Path: name, Bytes: bundle.files[name]})
		assembled.Digests[name] = hashBytes(bundle.files[name])
	}
	return browser.AuditBundle(assembled)
}

// browserManifestFile binds one published byte: its logical generation
// path, content digest, and served route.
type browserManifestFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Route  string `json:"route"`
}

// browserManifest is the I-6 verified browser manifest: the bundle
// identity, every published byte with its digest and route, the toolchain,
// source, and catalogue inputs, and the locked dependency instances. The
// same inputs yield identical bytes and manifest; UP18 verifies this whole
// record before serving any byte.
type browserManifest struct {
	SchemaVersion  int                   `json:"schemaVersion"`
	Kind           string                `json:"kind"`
	BrowserBuildID string                `json:"browserBuildId"`
	Entry          string                `json:"entry"`
	Table          string                `json:"table"`
	Files          []browserManifestFile `json:"files"`
	Toolchain      struct {
		Target   string `json:"target"`
		Version  string `json:"version"`
		Revision string `json:"revision"`
		SHA256   string `json:"sha256"`
		Compiler string `json:"compiler"`
		Runtime  string `json:"runtime"`
	} `json:"toolchain"`
	Inputs struct {
		Source       string `json:"source"`
		Dependencies string `json:"dependencies"`
		Catalogue    string `json:"catalogue"`
		Compiler     string `json:"compiler"`
		Runtime      string `json:"runtime"`
		Options      string `json:"options"`
	} `json:"inputs"`
	Lock            string `json:"lock"`
	LockedInstances []struct {
		Instance       string `json:"instance"`
		Lineage        string `json:"lineage"`
		ManifestSHA256 string `json:"manifestSHA256"`
		SourceSHA256   string `json:"sourceSHA256"`
		FixturesSHA256 string `json:"fixturesSHA256"`
	} `json:"lockedInstances"`
}

func browserManifestBytes(graph *project.Graph, inputs BuildInputs, toolchain browserToolchain, bundle browserBundleOutputs) ([]byte, error) {
	if graph == nil {
		return nil, fmt.Errorf("browser manifest requires the project graph")
	}
	manifest := browserManifest{SchemaVersion: 1, Kind: "can.browser-manifest", Entry: bundle.entry, Table: browserBundleTable, Files: []browserManifestFile{}, LockedInstances: []struct {
		Instance       string `json:"instance"`
		Lineage        string `json:"lineage"`
		ManifestSHA256 string `json:"manifestSHA256"`
		SourceSHA256   string `json:"sourceSHA256"`
		FixturesSHA256 string `json:"fixturesSHA256"`
	}{}}
	for _, name := range sortedOutputKeys(bundle.files) {
		digest := hashBytes(bundle.files[name])
		manifest.Files = append(manifest.Files, browserManifestFile{name, digest, browserAssetRoute(name, digest)})
	}
	identity, err := json.Marshal(manifest.Files)
	if err != nil {
		return nil, err
	}
	manifest.BrowserBuildID = hashBytes(append([]byte("can-browser-bundle-v1\x00"), identity...))
	manifest.Toolchain.Target = toolchain.Target
	manifest.Toolchain.Version = toolchain.Version
	manifest.Toolchain.Revision = toolchain.Revision
	manifest.Toolchain.SHA256 = toolchain.SHA256
	manifest.Toolchain.Compiler = toolchain.Compiler
	manifest.Toolchain.Runtime = toolchain.Runtime
	manifest.Inputs.Source = inputs.Source
	manifest.Inputs.Dependencies = inputs.Dependencies
	manifest.Inputs.Catalogue = inputs.Catalogue
	manifest.Inputs.Compiler = inputs.Compiler
	manifest.Inputs.Runtime = inputs.Runtime
	manifest.Inputs.Options = inputs.Options
	manifest.Lock = graph.LockSHA256
	for _, instance := range sortedOutputKeys(graph.Lock.Projects) {
		entry := graph.Lock.Projects[instance]
		manifest.LockedInstances = append(manifest.LockedInstances, struct {
			Instance       string `json:"instance"`
			Lineage        string `json:"lineage"`
			ManifestSHA256 string `json:"manifestSHA256"`
			SourceSHA256   string `json:"sourceSHA256"`
			FixturesSHA256 string `json:"fixturesSHA256"`
		}{instance, entry.Lineage, entry.ManifestSHA256, entry.SourceSHA256, entry.FixturesSHA256})
	}
	encoded, err := json.Marshal(manifest)
	if err != nil {
		return nil, err
	}
	return append(encoded, '\n'), nil
}

// browserAssetRoute states the served route for one published byte under
// the shared asset namespace: the content digest plus the published
// extension. UP18 serves exactly these routes after verifying every
// digest; nothing else is addressed.
func browserAssetRoute(logical, digest string) string {
	extension := path.Ext(logical)
	if strings.HasSuffix(logical, ".js.map") {
		extension = ".js.map"
	}
	return "/__can/assets/" + digest + extension
}
