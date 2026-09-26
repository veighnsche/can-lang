package browser

import (
	"bytes"
	"fmt"
	"path"
	"sort"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
)

// SourceIndexPath is the generation-relative sealed source index binding
// generated modules to checked Can spans.
const SourceIndexPath = "diagnostics/source-index.json"

// browserOverlay is the sealed-profile substitution table, stated in
// runtime-relative module paths: at browser bundle time every edge to a
// canonical module resolves to its alternate instead. The pre-bundle audit
// applies the same rule while walking the reachable graph, so the bodies
// inspected are the bodies that ship. A canonical module whose alternate
// is absent from the generation is inspected as-is and fails closed on its
// own host edges. UP07 sealed the owner-context core; UP15 extends the
// table to the remaining profile-divergent modules (coordinator-flagged):
// stub alternates for server-only capabilities plus native clock and log
// alternates. The canonical HTML escaper is portable instead of overlaid.
var browserOverlay = map[string]string{
	"reflect.ts":           "browser/reflect.ts",
	"domain.ts":            "browser/domain.ts",
	"diagnostics.ts":       "browser/diagnostics.ts",
	"owner.ts":             "browser/owner.ts",
	"callable.ts":          "browser/callable.ts",
	"coordination.ts":      "browser/coordination.ts",
	"entry.ts":             "browser/entry.ts",
	"codec/formats.ts":     "browser/formats.ts",
	"platform/cookies.ts":  "browser/cookies.ts",
	"platform/csrf.ts":     "browser/csrf.ts",
	"platform/clock.ts":    "browser/clock.ts",
	"platform/log.ts":      "browser/log.ts",
	"platform/markdown.ts": "browser/markdown.ts",
	"platform/assets.ts":   "browser/assets.ts",
}

// OverlayShipping maps a generation artifact path to the module that ships
// in the browser profile: the sealed alternate when the runtime-relative
// path names one, else the path itself. The bundler applies this exact
// substitution so shipped bytes are the inspected bytes; callers confirm
// the alternate exists in the generation exactly as the audit does.
func OverlayShipping(artifactPath string, runtime bool) string {
	rel, ok := runtimeRelativePath(artifactPath, runtime)
	if !ok {
		return artifactPath
	}
	alternate, ok := browserOverlay[rel]
	if !ok {
		return artifactPath
	}
	return artifactPath[:len(artifactPath)-len(rel)] + alternate
}

// forbiddenRuntimeDirs denies whole server-capability runtime domains at
// segment boundaries: SQL pools and transactions, process control,
// filesystem access, server crypto, and server AI oracles.
var forbiddenRuntimeDirs = []struct{ dir, reason string }{
	{"platform/sql/", "SQL pools and transactions are server-only"},
	{"platform/process/", "process control is server-only"},
	{"platform/files/", "filesystem access is server-only"},
	{"platform/crypto/", "server crypto is unavailable to the browser profile"},
	{"ai/", "server AI oracles are unavailable to the browser profile"},
}

// forbiddenRuntimeFiles denies server-only runtime modules by
// runtime-relative path: host environment and secrets, credentialed object
// storage, the HTTP server lifecycle, host stdio, the server CLI, server
// websocket sessions, and the Bun entry module.
var forbiddenRuntimeFiles = map[string]string{
	"environment.ts":        "host environment secrets never enter the browser bundle",
	"platform/env.ts":       "environment secrets never enter the browser bundle",
	"platform/s3.ts":        "credentialed object storage is server-only",
	"platform/server.ts":    "the HTTP server lifecycle is server-only",
	"platform/io.ts":        "host stdio is server-only",
	"platform/cli.ts":       "the server CLI is server-only",
	"platform/websocket.ts": "websocket sessions are server-only",
	"entry.ts":              "the Bun entry module must be absent from a browser generation",
}

// forbiddenMappingOperations are server-request source-map operations that
// must never appear in browser output.
var forbiddenMappingOperations = map[string]bool{
	"fetch_request": true, "llm_request": true, "judge_request": true, "question_preparation": true,
}

// secretCanaries are exact sentinel strings that must never appear in any
// audited byte, in code or in data: a leaked secret inside a string
// literal is still a leak. The first two are documented build sentinels
// the verified pipeline plants in server-only material; the rest are
// well-known private-key markers. Unlike host operations, canary matching
// is deliberately content-based rather than lexical.
var secretCanaries = []string{
	"__CAN_SECRET_CANARY__",
	"CANARY-SECRET-DO-NOT-SHIP",
	"-----BEGIN PRIVATE KEY-----",
	"-----BEGIN RSA PRIVATE KEY-----",
	"-----BEGIN DSA PRIVATE KEY-----",
	"-----BEGIN EC PRIVATE KEY-----",
	"-----BEGIN OPENSSH PRIVATE KEY-----",
	"-----BEGIN ENCRYPTED PRIVATE KEY-----",
}

// AuditArtifacts verifies the emitted pre-bundle browser graph. It replaces
// the retired substring scan with structural resolution:
//
//   - the single browser root replaces the Bun entry;
//   - every static edge lexed from a module body must be declared in that
//     module's import inventory, and every declared edge must appear in the
//     body: unknown or unaccounted edges fail;
//   - the reachable graph is walked from the browser root through
//     overlay-resolved value edges; type-only edges are verified but not
//     traversed, since the transpiler erases them deterministically;
//   - every generated module and every reachable runtime module is scanned
//     for actual host operations (Bun/process/require/eval/Function,
//     dynamic import); quoted data never matches;
//   - reachable native (node:/bun:) edges, server-capability runtime
//     modules, server-request mappings, secret canaries and unaccounted
//     artifacts fail;
//   - published asset files are canary-scanned, and asset scripts must be
//     self-contained and host free;
//   - when the generation carries sealed source maps, every generated
//     module must bind a complete, consistent map, trailer and index
//     entry, and the asset manifest must bind the exact generated bytes.
//
// Diagnostics cite the offending module with line and column, the
// originating Can span and operation when mappings cover the position,
// and the module edge chain from the browser root.
func AuditArtifacts(artifacts []ir.Artifact) error {
	audit, err := indexArtifacts(artifacts)
	if err != nil {
		return err
	}
	if _, ok := audit.byPath[BrowserEntry]; !ok {
		return fmt.Errorf("browser audit: missing %s root", BrowserEntry)
	}
	if _, forbidden := audit.byPath[BunEntry]; forbidden {
		return fmt.Errorf("browser audit: bun entry %s must be absent from a browser generation", BunEntry)
	}
	asset, ok := audit.byPath[AssetPath]
	if !ok {
		return fmt.Errorf("browser audit: missing %s", AssetPath)
	}
	for _, artifact := range artifacts {
		if err := audit.classify(artifact); err != nil {
			return err
		}
	}
	chain := []string{BrowserEntry}
	if err := audit.visit(BrowserEntry, chain); err != nil {
		return err
	}
	for _, name := range audit.generated {
		if audit.visited[name] {
			continue
		}
		unreachable := []string{"<emitted, unreachable from main>", name}
		if err := audit.inspect(name, unreachable); err != nil {
			return err
		}
	}
	for _, artifact := range artifacts {
		if !strings.HasPrefix(artifact.Path, "assets/") {
			continue
		}
		if err := audit.inspectAsset(artifact); err != nil {
			return err
		}
	}
	if err := audit.sealedMaps(); err != nil {
		return err
	}
	if err := scanCanaries(AssetPath, asset.Bytes, nil); err != nil {
		return err
	}
	return audit.bindAsset(asset)
}

type graphAudit struct {
	byPath    map[string]ir.Artifact
	generated []string
	maps      map[string]ir.Artifact
	indexed   map[string]bool
	visited   map[string]bool
	scans     map[string]ModuleScan
}

func indexArtifacts(artifacts []ir.Artifact) (*graphAudit, error) {
	audit := &graphAudit{
		byPath:  map[string]ir.Artifact{},
		maps:    map[string]ir.Artifact{},
		indexed: map[string]bool{},
		visited: map[string]bool{},
		scans:   map[string]ModuleScan{},
	}
	for _, artifact := range artifacts {
		if artifact.Path == "" {
			return nil, fmt.Errorf("browser audit: artifact carries no path")
		}
		if _, dup := audit.byPath[artifact.Path]; dup {
			return nil, fmt.Errorf("browser audit: duplicate artifact %s", artifact.Path)
		}
		audit.byPath[artifact.Path] = artifact
		if !artifact.Runtime && strings.HasSuffix(artifact.Path, ".ts") {
			audit.generated = append(audit.generated, artifact.Path)
		}
		if strings.HasSuffix(artifact.Path, ".ts.map") {
			audit.maps[artifact.Path] = artifact
		}
	}
	sort.Strings(audit.generated)
	return audit, nil
}

func (audit *graphAudit) classify(artifact ir.Artifact) error {
	switch {
	case strings.HasSuffix(artifact.Path, ".ts"),
		strings.HasSuffix(artifact.Path, ".ts.map"),
		artifact.Path == AssetPath,
		artifact.Path == SourceIndexPath,
		strings.HasPrefix(artifact.Path, "assets/"):
		return nil
	default:
		return fmt.Errorf("browser audit: %s is an unaccounted pre-bundle artifact", artifact.Path)
	}
}

// inspectAsset audits one published asset file: the pinned vendor script
// and checked project assets (executable project formats are rejected at
// load, so project scripts never occur here). Scripts must be
// self-contained — non-module artifacts cannot declare imports — and host
// free; every asset byte is canary-scanned.
func (audit *graphAudit) inspectAsset(artifact ir.Artifact) error {
	if len(artifact.Imports) != 0 || len(artifact.NativeImports) != 0 {
		return fmt.Errorf("browser audit: asset %s carries module imports", artifact.Path)
	}
	if strings.HasSuffix(artifact.Path, ".js") {
		scan, err := ScanModule(artifact.Bytes)
		if err != nil {
			return fmt.Errorf("browser audit: asset %s cannot be inspected structurally: %w", artifact.Path, err)
		}
		if len(scan.Edges) != 0 {
			return fmt.Errorf("browser audit: asset %s:%s carries unaccounted edge %q: published scripts are self-contained", artifact.Path, scan.Edges[0].Pos, scan.Edges[0].Specifier)
		}
		for _, finding := range scan.Findings {
			return fmt.Errorf("browser audit: asset %s:%s forbids host operation %s", artifact.Path, finding.Pos, finding.Operation)
		}
	}
	return scanCanaries(artifact.Path, artifact.Bytes, nil)
}

// runtimeRelative resolves a generation runtime path to its runtime-relative
// module path: runtime/r-<id>/<rel> and the emit-test shape runtime/<rel>
// both yield <rel>. Non-runtime paths yield ok=false.
func runtimeRelative(artifact ir.Artifact) (rel string, ok bool) {
	return runtimeRelativePath(artifact.Path, artifact.Runtime)
}

func runtimeRelativePath(name string, runtime bool) (rel string, ok bool) {
	if !runtime {
		return "", false
	}
	start := -1
	for index := 0; index+len("runtime/") <= len(name); index++ {
		if (index == 0 || name[index-1] == '/') && strings.HasPrefix(name[index:], "runtime/") {
			start = index + len("runtime/")
			break
		}
	}
	if start < 0 || start >= len(name) {
		return "", false
	}
	rest := name[start:]
	if cut := strings.IndexByte(rest, '/'); cut >= 0 {
		return rest[cut+1:], true
	}
	return rest, true
}

// forbiddenModule reports whether a resolved module path names a
// server-capability runtime module. Directory-qualified domains match at
// segment boundaries on any path; bare module names match runtime paths
// only, so generated modules never collide with the inventory.
func forbiddenModule(artifact ir.Artifact) (string, bool) {
	for _, denied := range forbiddenRuntimeDirs {
		if hasSegmentPath(artifact.Path, denied.dir) {
			return denied.reason, true
		}
	}
	rel, ok := runtimeRelative(artifact)
	if !ok {
		return "", false
	}
	if reason, denied := forbiddenRuntimeFiles[rel]; denied {
		return reason, true
	}
	return "", false
}

func hasSegmentPath(path, dir string) bool {
	if path == strings.TrimSuffix(dir, "/") {
		return true
	}
	if strings.HasPrefix(path, dir) {
		return true
	}
	return strings.Contains(path, "/"+dir)
}

// overlay resolves a runtime module through the sealed-profile substitution
// table. It returns the shipping path and whether a substitution applied.
func (audit *graphAudit) overlay(artifact ir.Artifact) (ir.Artifact, bool) {
	redirect := OverlayShipping(artifact.Path, artifact.Runtime)
	if redirect == artifact.Path {
		return artifact, false
	}
	if target, ok := audit.byPath[redirect]; ok {
		return target, true
	}
	return artifact, false
}

func (audit *graphAudit) visit(name string, chain []string) error {
	if audit.visited[name] {
		return nil
	}
	audit.visited[name] = true
	if reason, denied := forbiddenModule(audit.byPath[name]); denied {
		return fmt.Errorf("browser audit: %s is a server-capability runtime module (%s; reachable via %s)", name, reason, strings.Join(chain, " -> "))
	}
	if err := audit.inspect(name, chain); err != nil {
		return err
	}
	scan := audit.scans[name]
	typeOnly := map[string]bool{}
	for _, edge := range scan.Edges {
		if edge.TypeOnly {
			typeOnly[edge.Specifier] = true
		}
	}
	neighbors := append([]string{}, audit.byPath[name].Imports...)
	sort.Strings(neighbors)
	for _, spec := range neighbors {
		if typeOnly[spec] {
			continue
		}
		shipping, canonical, err := audit.resolve(name, spec)
		if err != nil {
			return err
		}
		next := append(append([]string(nil), chain...), canonical.Path)
		if shipping.Path != canonical.Path {
			next = append(next, shipping.Path+" [profile overlay]")
		}
		if err := audit.visit(shipping.Path, next); err != nil {
			return err
		}
	}
	return nil
}

// resolve maps one declared edge to its shipping artifact, applying the
// profile overlay and rejecting unaccounted and server-only targets. It
// returns both the edge target and the shipping module so diagnostics can
// show the overlay hop.
func (audit *graphAudit) resolve(from, spec string) (shipping, canonical ir.Artifact, err error) {
	if !strings.HasPrefix(spec, "./") && !strings.HasPrefix(spec, "../") {
		return ir.Artifact{}, ir.Artifact{}, fmt.Errorf("browser audit: %s has a non-relative edge %q", from, spec)
	}
	resolved := path.Join(path.Dir(from), spec)
	target, ok := audit.byPath[resolved]
	if !ok {
		return ir.Artifact{}, ir.Artifact{}, fmt.Errorf("browser audit: %s has an unaccounted edge %q: no module %s in the generation", from, spec, resolved)
	}
	if !strings.HasSuffix(target.Path, ".ts") {
		return ir.Artifact{}, ir.Artifact{}, fmt.Errorf("browser audit: %s edge %q resolves to non-module %s", from, spec, target.Path)
	}
	shipping, _ = audit.overlay(target)
	if reason, denied := forbiddenModule(shipping); denied {
		return ir.Artifact{}, ir.Artifact{}, fmt.Errorf("browser audit: %s reaches server-capability runtime module %s (%s)", from, shipping.Path, reason)
	}
	return shipping, target, nil
}

// inspect scans one module body structurally: the lexed edge inventory
// must equal the declared inventory exactly, edges must be well-formed,
// native edges and host operations fail with origin evidence, mappings
// must carry no server-request operation, and no secret canary may appear.
func (audit *graphAudit) inspect(name string, chain []string) error {
	artifact := audit.byPath[name]
	scan, err := ScanModule(artifact.Bytes)
	if err != nil {
		return fmt.Errorf("browser audit: %s cannot be inspected structurally: %w (reachable via %s)", name, err, strings.Join(chain, " -> "))
	}
	audit.scans[name] = scan
	if err := audit.checkInventory(artifact, scan, chain); err != nil {
		return err
	}
	for _, edge := range scan.Edges {
		if strings.HasPrefix(edge.Specifier, "./") || strings.HasPrefix(edge.Specifier, "../") {
			if err := audit.checkEdgeTarget(artifact.Path, edge, chain); err != nil {
				return err
			}
			continue
		}
		via := strings.Join(chain, " -> ")
		if edge.Specifier == "" {
			return fmt.Errorf("browser audit: %s:%s has an empty module edge (reachable via %s)", name, edge.Pos, via)
		}
		if strings.HasPrefix(edge.Specifier, "node:") || strings.HasPrefix(edge.Specifier, "bun:") {
			return fmt.Errorf("browser audit: %s:%s carries native edge %q, unavailable to --target browser (reachable via %s)", name, edge.Pos, edge.Specifier, via)
		}
		return fmt.Errorf("browser audit: %s:%s has a non-relative edge %q (reachable via %s)", name, edge.Pos, edge.Specifier, via)
	}
	for _, finding := range scan.Findings {
		via := strings.Join(chain, " -> ")
		origin := mappingOrigin(artifact.Mappings, finding.Pos)
		return fmt.Errorf("browser audit: %s:%s forbids host operation %s%s (reachable via %s)", name, finding.Pos, finding.Operation, origin, via)
	}
	for _, mapping := range artifact.Mappings {
		if forbiddenMappingOperations[mapping.Operation] {
			return fmt.Errorf("browser audit: %s:%d:%d carries server-request mapping %q", name, mapping.Line, mapping.Column, mapping.Operation)
		}
	}
	return scanCanaries(name, artifact.Bytes, chain)
}

// checkInventory requires the lexed static edge set to equal the declared
// import inventory exactly in both directions: neither side may carry an
// edge the other omits. Only verified distribution runtime artifacts may
// declare native imports.
func (audit *graphAudit) checkInventory(artifact ir.Artifact, scan ModuleScan, chain []string) error {
	via := strings.Join(chain, " -> ")
	lexedRelative, lexedNative := map[string]Position{}, map[string]Position{}
	for _, edge := range scan.Edges {
		if strings.HasPrefix(edge.Specifier, "./") || strings.HasPrefix(edge.Specifier, "../") {
			lexedRelative[edge.Specifier] = edge.Pos
		} else {
			lexedNative[edge.Specifier] = edge.Pos
		}
	}
	declaredRelative, declaredNative := map[string]bool{}, map[string]bool{}
	for _, spec := range artifact.Imports {
		declaredRelative[spec] = true
	}
	for _, spec := range artifact.NativeImports {
		declaredNative[spec] = true
	}
	if len(artifact.NativeImports) != 0 && !artifact.Runtime {
		return fmt.Errorf("browser audit: %s carries native imports but is not a verified runtime module (reachable via %s)", artifact.Path, via)
	}
	for spec, pos := range lexedRelative {
		if !declaredRelative[spec] {
			return fmt.Errorf("browser audit: %s:%s has an unaccounted edge %q: lexed from the body but absent from the import inventory (reachable via %s)", artifact.Path, pos, spec, via)
		}
	}
	for spec, pos := range lexedNative {
		if !declaredNative[spec] {
			return fmt.Errorf("browser audit: %s:%s has an unaccounted native edge %q (reachable via %s)", artifact.Path, pos, spec, via)
		}
	}
	for spec := range declaredRelative {
		if _, ok := lexedRelative[spec]; !ok {
			return fmt.Errorf("browser audit: %s declares edge %q with no matching import in the body (reachable via %s)", artifact.Path, spec, via)
		}
	}
	for spec := range declaredNative {
		if _, ok := lexedNative[spec]; !ok {
			return fmt.Errorf("browser audit: %s declares native edge %q with no matching import in the body (reachable via %s)", artifact.Path, spec, via)
		}
	}
	return nil
}

func (audit *graphAudit) checkEdgeTarget(from string, edge Edge, chain []string) error {
	via := strings.Join(chain, " -> ")
	if !strings.HasSuffix(edge.Specifier, ".ts") || strings.ContainsAny(edge.Specifier, "\\?#") {
		return fmt.Errorf("browser audit: %s:%s has a non-module edge %q (reachable via %s)", from, edge.Pos, edge.Specifier, via)
	}
	resolved := path.Join(path.Dir(from), edge.Specifier)
	target, ok := audit.byPath[resolved]
	if !ok {
		return fmt.Errorf("browser audit: %s:%s has an unaccounted edge %q: no module %s in the generation (reachable via %s)", from, edge.Pos, edge.Specifier, resolved, via)
	}
	if !strings.HasSuffix(target.Path, ".ts") {
		return fmt.Errorf("browser audit: %s:%s edge %q resolves to non-module %s (reachable via %s)", from, edge.Pos, edge.Specifier, target.Path, via)
	}
	return nil
}

// mappingOrigin resolves a generated coordinate to its originating Can
// span through the module mappings, or "" when no mapping covers it.
func mappingOrigin(mappings []ir.Mapping, pos Position) string {
	best := -1
	for index, mapping := range mappings {
		if mapping.Line < pos.Line || (mapping.Line == pos.Line && mapping.Column <= pos.Column) {
			if best < 0 || mapping.Line > mappings[best].Line ||
				(mapping.Line == mappings[best].Line && mapping.Column > mappings[best].Column) {
				best = index
			}
		}
	}
	if best < 0 {
		return ""
	}
	mapping := mappings[best]
	return fmt.Sprintf("; originating can %s [%d:%d] operation %s", mapping.Source, mapping.Start, mapping.End, mapping.Operation)
}

func scanCanaries(name string, data []byte, chain []string) error {
	return scanCanaryList(name, data, chain, secretCanaries)
}

func scanCanaryList(name string, data []byte, chain []string, canaries []string) error {
	for _, canary := range canaries {
		if canary == "" {
			continue
		}
		if offset := bytes.Index(data, []byte(canary)); offset >= 0 {
			pos := offsetPosition(data, offset)
			via := ""
			if len(chain) != 0 {
				via = fmt.Sprintf(" (reachable via %s)", strings.Join(chain, " -> "))
			}
			return fmt.Errorf("browser audit: %s:%s leaks secret canary %q%s", name, pos, canary, via)
		}
	}
	return nil
}

func offsetPosition(data []byte, offset int) Position {
	line, column := 1, 0
	for index := 0; index < offset && index < len(data); {
		char := data[index]
		if char == '\n' {
			line++
			column = 0
			index++
			continue
		}
		if char == '\r' {
			if index+1 < len(data) && data[index+1] == '\n' {
				index++
			}
			line++
			column = 0
			index++
			continue
		}
		runeValue, width := utf8.DecodeRune(data[index:])
		column += utf16.RuneLen(runeValue)
		index += width
	}
	return Position{Offset: offset, Line: line, Column: column}
}
