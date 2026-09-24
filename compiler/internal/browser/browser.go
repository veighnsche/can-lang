// Package browser implements the `canlc build --target browser` compile-time
// boundary: a distinct main-thread root/profile, a transitive capability
// closure over the checked program, a compiler-produced content-addressed
// same-origin asset, and an emitted-graph audit. It never executes programs
// and never reads authored JavaScript or TypeScript: every input is checked
// Can IR or compiler-emitted artifacts.
package browser

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/source"
)

// Target selects the compiler build profile. Only the two named profiles
// exist: there is no worker profile and no arbitrary authored-JS profile.
type Target string

const (
	// TargetBun is the default server profile executed by Bun.
	TargetBun Target = "bun"
	// TargetBrowser is the main-thread browser profile.
	TargetBrowser Target = "browser"
)

// BrowserEntry is the generation-relative browser root module. A browser
// generation never contains the Bun entry module.
const BrowserEntry = "browser.ts"

// BunEntry names the Bun root module, which must be absent from browser output.
const BunEntry = "entry.ts"

// AssetPath is the generation-relative browser asset manifest.
const AssetPath = "browser/asset.json"

// Profile marks the browser main-thread root.
const Profile = "browser-main"

// ParseTarget resolves a `--target` flag value. Unknown values fail closed
// with the supported profiles; worker execution is explicitly rejected.
func ParseTarget(value string) (Target, error) {
	switch Target(value) {
	case TargetBun, TargetBrowser:
		return Target(value), nil
	default:
		return "", fmt.Errorf("unknown --target %q: expected \"bun\" or \"browser\" (no worker profile)", value)
	}
}

// forbiddenPrefixes denies whole server-capability domains: SQL pools and
// transactions, process control, filesystem access, environment secrets,
// server crypto and password hashing, credentialed object storage, stdio,
// and the stream/websocket handles that back them.
var forbiddenPrefixes = []struct{ prefix, reason string }{
	{"can.std.sql@1::", "SQL pools and transactions are server-only"},
	{"can.std.process@1::", "process control is server-only"},
	{"can.std.files@1::", "filesystem access is server-only"},
	{"can.std.env@1::", "environment secrets never enter the browser bundle"},
	{"can.std.crypto@1::", "server crypto is unavailable to the browser profile"},
	{"can.std.password@1::", "password hashing is server-only"},
	{"can.std.s3@1::", "credentialed object storage is server-only"},
	{"can.std.io@1::", "host stdio is server-only"},
	{"can.std.stream@1::", "stream handles are server-only"},
	{"can.std.ws@1::", "websocket sessions are server-only"},
}

// forbiddenExact denies server lifecycle operations. Pure HTTP request,
// response and router constructors stay available.
var forbiddenExact = map[string]string{
	"can.std.http@1::server_start":       "the HTTP server lifecycle is server-only",
	"can.std.http@1::server_start_tls":   "the HTTP server lifecycle is server-only",
	"can.std.http@1::server_stop":        "the HTTP server lifecycle is server-only",
	"can.std.http@1::server_wait":        "the HTTP server lifecycle is server-only",
	"can.std.http@1::make_tls_config":    "TLS configuration is server-only",
	"can.std.http@1::make_server_config": "the HTTP server lifecycle is server-only",
}

// ForbiddenReason reports whether an invocation or callable identity names a
// server-only operation. Specialization keys (`<op>/instance/<hex>`) match
// through their declaration prefix, so generic instantiation paths are
// covered exactly like direct calls.
func ForbiddenReason(identity string) (string, bool) {
	declaration := identity
	if index := strings.Index(declaration, "/instance/"); index >= 0 {
		declaration = declaration[:index]
	}
	if reason, denied := forbiddenExact[declaration]; denied {
		return reason, true
	}
	for _, denied := range forbiddenPrefixes {
		if strings.HasPrefix(declaration, denied.prefix) {
			return denied.reason, true
		}
	}
	return "", false
}

// CheckProgram enforces the transitive browser capability closure: every
// operation reachable from the entry through authored calls, callable
// references and generic specializations must be browser-available, as must
// every other emitted function, top-level initializer, native declaration
// and checked specialization. Assertion roots are excluded: they execute
// under Bun during the verified build and never ship in the browser asset.
func CheckProgram(program *check.Program) error {
	if program == nil {
		return fmt.Errorf("browser capability closure requires a checked program")
	}
	gate := &closureGate{program: program, paths: map[string]string{}}
	if program.World != nil {
		for src := range program.World.Files {
			gate.paths[src.ID] = src.Path
			gate.paths[src.Name] = src.Path
		}
	}
	gate.functions = map[string]*check.ProgramFunction{}
	for _, fn := range program.Functions {
		gate.functions[fn.Identity()] = fn
	}
	if err := gate.structural(); err != nil {
		return err
	}
	if program.Entry == nil || program.Entry.Region == nil {
		return fmt.Errorf("browser build requires a checked entry")
	}
	visited := map[string]bool{}
	chain := []string{program.Entry.Identity()}
	if err := gate.function(program.Entry, chain, visited); err != nil {
		return err
	}
	for _, fn := range program.Functions {
		if visited[fn.Identity()] {
			continue
		}
		unreachable := []string{"<emitted, unreachable from main>", fn.Identity()}
		if err := gate.region(fn, fn.Region, unreachable, map[string]bool{fn.Identity(): true}); err != nil {
			return err
		}
	}
	for _, init := range program.Initializers {
		file := gate.paths[init.Source]
		chain := []string{"<top-level initializer>", init.Identity}
		walker := &regionWalker{gate: gate, file: file, chain: chain, visited: map[string]bool{}, scopes: map[string]bool{}}
		if init.Value != nil {
			if err := walker.expression(init.Value); err != nil {
				return err
			}
		}
	}
	return nil
}

type closureGate struct {
	program   *check.Program
	functions map[string]*check.ProgramFunction
	paths     map[string]string
}

// structural rejects server-backed declarations and specializations even
// when no reachable call site names them: native declarations and
// connections carry server endpoints and secrets, SQL descriptors bind
// statements, and SQL/transaction/stream specializations are capability
// handles, not pure data.
func (gate *closureGate) structural() error {
	program := gate.program
	if len(program.Natives) != 0 {
		names := make([]string, 0, len(program.Natives))
		for _, native := range program.Natives {
			names = append(names, native.Symbol.ID)
		}
		sort.Strings(names)
		return fmt.Errorf("browser capability closure: native declaration %s needs a server connection or AI oracle and is unavailable to --target browser", names[0])
	}
	if len(program.Connections) != 0 {
		names := make([]string, 0, len(program.Connections))
		for name := range program.Connections {
			names = append(names, name)
		}
		sort.Strings(names)
		return fmt.Errorf("browser capability closure: connection %s carries a server endpoint and is unavailable to --target browser", names[0])
	}
	if len(program.SQL) != 0 {
		return fmt.Errorf("browser capability closure: %d SQL descriptor(s) bind server statements and are unavailable to --target browser", len(program.SQL))
	}
	for _, specializations := range []struct {
		name string
		ids  []string
	}{
		{"SQL", specializationIDs(program.SQLs)},
		{"transaction", specializationIDs(program.Transactions)},
		{"stream", specializationIDs(program.Streams)},
	} {
		if len(specializations.ids) != 0 {
			return fmt.Errorf("browser capability closure: %s specialization %s is unavailable to --target browser", specializations.name, specializations.ids[0])
		}
	}
	return nil
}

func specializationIDs[V any](specializations map[string]V) []string {
	ids := make([]string, 0, len(specializations))
	for id := range specializations {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (gate *closureGate) function(fn *check.ProgramFunction, chain []string, visited map[string]bool) error {
	if visited[fn.Identity()] {
		return nil
	}
	visited[fn.Identity()] = true
	return gate.region(fn, fn.Region, chain, visited)
}

func (gate *closureGate) region(fn *check.ProgramFunction, region *ir.Region, chain []string, visited map[string]bool) error {
	if region == nil {
		return nil
	}
	file := ""
	if fn.Symbol != nil && fn.Symbol.Source != nil {
		file = fn.Symbol.Source.Path
	}
	walker := &regionWalker{gate: gate, file: file, chain: chain, visited: visited, scopes: map[string]bool{}}
	return walker.region(region)
}

type regionWalker struct {
	gate    *closureGate
	file    string
	chain   []string
	visited map[string]bool
	scopes  map[string]bool
}

func (walker *regionWalker) locate(span source.Span, err error) error {
	if walker.file == "" {
		return err
	}
	return source.Locate(walker.file, span, err)
}

func (walker *regionWalker) branch(identity string) []string {
	return append(append([]string(nil), walker.chain...), identity)
}

// operation rejects a callee identity or recurses into an authored
// function, recording the reference chain for the diagnostic.
func (walker *regionWalker) operation(identity string, span source.Span) error {
	if identity == "" {
		return nil
	}
	if fn := walker.gate.functions[identity]; fn != nil {
		if walker.visited[identity] {
			return nil
		}
		walker.visited[identity] = true
		next := &regionWalker{gate: walker.gate, chain: walker.branch(identity), visited: walker.visited, scopes: walker.scopes}
		if fn.Symbol != nil && fn.Symbol.Source != nil {
			next.file = fn.Symbol.Source.Path
		}
		return next.region(fn.Region)
	}
	if reason, denied := ForbiddenReason(identity); denied {
		via := strings.Join(walker.branch(identity), " -> ")
		return walker.locate(span, fmt.Errorf("browser capability closure: %s (%s; reachable via %s)", identity, reason, via))
	}
	return nil
}

func (walker *regionWalker) region(region *ir.Region) error {
	if region == nil {
		return nil
	}
	if region.ID != "" {
		if walker.scopes[region.ID] {
			return nil
		}
		walker.scopes[region.ID] = true
	}
	return walker.block(region.Body)
}

func (walker *regionWalker) block(block *ir.Block) error {
	if block == nil {
		return nil
	}
	for i := range block.Steps {
		if err := walker.statement(&block.Steps[i]); err != nil {
			return err
		}
	}
	return walker.completion(block.Terminal)
}

func (walker *regionWalker) statement(step *ir.Statement) error {
	if step == nil {
		return nil
	}
	if step.Coordination != nil {
		if err := walker.coordination(step.Coordination); err != nil {
			return err
		}
	}
	if step.Value != nil {
		if err := walker.expression(step.Value); err != nil {
			return err
		}
	}
	return walker.invocation(step.Call)
}

func (walker *regionWalker) completion(done *ir.Completion) error {
	if done == nil {
		return nil
	}
	if done.Value != nil {
		if err := walker.expression(done.Value); err != nil {
			return err
		}
	}
	if done.Call != nil {
		if err := walker.invocation(done.Call); err != nil {
			return err
		}
	}
	if done.Block != nil {
		if err := walker.block(done.Block); err != nil {
			return err
		}
	}
	return walker.match(done.Match)
}

func (walker *regionWalker) match(selected *ir.Match) error {
	if selected == nil {
		return nil
	}
	for _, value := range selected.Values {
		if err := walker.expression(value); err != nil {
			return err
		}
	}
	if err := walker.invocation(selected.Call); err != nil {
		return err
	}
	for i := range selected.Arms {
		arm := &selected.Arms[i]
		if arm.Value != nil {
			if err := walker.expression(arm.Value); err != nil {
				return err
			}
		}
		if err := walker.completion(arm.Body); err != nil {
			return err
		}
	}
	return nil
}

func (walker *regionWalker) expression(node *ir.Expression) error {
	if node == nil {
		return nil
	}
	nested := walker
	if node.Source != "" {
		if file, ok := walker.gate.paths[node.Source]; ok {
			copy := *walker
			copy.file = file
			nested = &copy
		}
	}
	if node.Callable != nil && node.Callable.Target != "" {
		if err := nested.operation(node.Callable.Target, node.Span); err != nil {
			return err
		}
	}
	if node.Coordination != nil {
		if err := nested.coordination(node.Coordination); err != nil {
			return err
		}
	}
	if err := nested.invocation(node.Invocation); err != nil {
		return err
	}
	if err := nested.match(node.Match); err != nil {
		return err
	}
	for _, input := range node.Inputs {
		if err := nested.expression(input); err != nil {
			return err
		}
	}
	return nil
}

func (walker *regionWalker) invocation(call *ir.Invocation) error {
	if call == nil {
		return nil
	}
	for i := range call.Steps {
		if err := walker.step(&call.Steps[i]); err != nil {
			return err
		}
	}
	return nil
}

func (walker *regionWalker) step(step *ir.InvocationStep) error {
	if step == nil {
		return nil
	}
	if step.SQL != nil {
		return walker.locate(step.Span, fmt.Errorf("browser capability closure: SQL call site %s.%s is unavailable to --target browser", step.SQL.Owner, step.SQL.Name))
	}
	if err := walker.operation(step.Identity, step.Span); err != nil {
		return err
	}
	if err := walker.expression(step.Callee); err != nil {
		return err
	}
	if err := walker.expression(step.Native); err != nil {
		return err
	}
	for _, argument := range step.Arguments {
		if err := walker.expression(argument); err != nil {
			return err
		}
	}
	for i := range step.Prepare {
		if err := walker.expression(step.Prepare[i].Value); err != nil {
			return err
		}
	}
	return nil
}

func (walker *regionWalker) coordination(coordination *ir.Coordination) error {
	if coordination == nil {
		return nil
	}
	for i := range coordination.Entries {
		entry := &coordination.Entries[i]
		if err := walker.invocation(entry.Call); err != nil {
			return err
		}
		if err := walker.expression(entry.Spread); err != nil {
			return err
		}
		if err := walker.outcome(entry.Handler); err != nil {
			return err
		}
	}
	if err := walker.outcome(coordination.Aggregate); err != nil {
		return err
	}
	return walker.outcome(coordination.Shared)
}

func (walker *regionWalker) outcome(handler *ir.OutcomeHandler) error {
	if handler == nil {
		return nil
	}
	if handler.Region != nil {
		if err := walker.region(handler.Region); err != nil {
			return err
		}
	}
	for i := range handler.Arms {
		arm := &handler.Arms[i]
		if arm.Value != nil {
			if err := walker.expression(arm.Value); err != nil {
				return err
			}
		}
		if err := walker.completion(arm.Body); err != nil {
			return err
		}
	}
	return nil
}

// Asset is the decoded browser asset manifest: the compiler-produced,
// content-addressed same-origin descriptor of one browser generation.
type Asset struct {
	SchemaVersion int               `json:"schemaVersion"`
	Kind          string            `json:"kind"`
	Profile       string            `json:"profile"`
	Entry         string            `json:"entry"`
	ContentSHA256 string            `json:"contentSHA256"`
	Files         map[string]string `json:"files"`
}

// AssetBytes renders the canonical asset manifest binding every
// non-runtime module digest. The digest covers sorted path/digest pairs,
// so identical graphs always produce identical assets.
func AssetBytes(files map[string]string) ([]byte, error) {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	hash := sha256.New()
	for _, name := range names {
		hash.Write([]byte(name))
		hash.Write([]byte{0})
		hash.Write([]byte(files[name]))
		hash.Write([]byte{0})
	}
	asset := Asset{SchemaVersion: 1, Kind: "can.browser-asset", Profile: Profile, Entry: BrowserEntry, ContentSHA256: hex.EncodeToString(hash.Sum(nil)), Files: files}
	encoded, err := json.MarshalIndent(asset, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(encoded, '\n'), nil
}

// ParseAsset decodes and validates a browser asset manifest.
func ParseAsset(raw []byte) (Asset, error) {
	var asset Asset
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&asset); err != nil {
		return Asset{}, fmt.Errorf("invalid browser asset: %w", err)
	}
	if asset.SchemaVersion != 1 || asset.Kind != "can.browser-asset" || asset.Profile != Profile || asset.Entry != BrowserEntry {
		return Asset{}, fmt.Errorf("invalid browser asset identity")
	}
	if asset.ContentSHA256 == "" || len(asset.Files) == 0 {
		return Asset{}, fmt.Errorf("browser asset binds no content")
	}
	expected, err := AssetBytes(asset.Files)
	if err != nil {
		return Asset{}, err
	}
	var canonical Asset
	if err := json.Unmarshal(expected, &canonical); err != nil {
		return Asset{}, err
	}
	if canonical.ContentSHA256 != asset.ContentSHA256 {
		return Asset{}, fmt.Errorf("browser asset content digest mismatch")
	}
	return asset, nil
}

// forbiddenRuntimePaths are server-capability runtime modules no browser
// module edge may reach. The generation-private runtime copy is the shared
// pinned inventory, but the browser reference graph must never touch it.
var forbiddenRuntimePaths = []string{
	"/platform/sql/",
	"/platform/process/",
	"/platform/files/",
	"/platform/env.ts",
	"/platform/crypto/",
	"/platform/s3.ts",
	"/platform/s3/",
	"/platform/server.ts",
	"/platform/io.ts",
	"/platform/websocket.ts",
	"/platform/websocket/",
	"/platform/stream",
	"/environment.ts",
	"/entry.ts",
}

// forbiddenMappingOperations are server-request source-map operations that
// must never appear in browser output.
var forbiddenMappingOperations = map[string]bool{
	"fetch_request": true, "llm_request": true, "judge_request": true, "question_preparation": true,
}

// serverTokens are host-capability markers no compiler-emitted browser
// module may contain. `import.meta.url` stays available for same-origin
// asset resolution.
var serverTokens = []string{"process.", "Bun.", "require(", "node:"}

// AuditArtifacts verifies the emitted browser graph: the single browser
// root replaces the Bun entry, every non-runtime module uses relative
// edges that avoid server-capability runtime modules, no host token or
// server-request mapping survives, and the asset manifest binds the
// emitted bytes. Same-origin holds because every edge and asset path is
// generation-relative: no absolute or remote specifier is admitted.
func AuditArtifacts(artifacts []ir.Artifact) error {
	byPath := map[string]ir.Artifact{}
	for _, artifact := range artifacts {
		byPath[artifact.Path] = artifact
	}
	entry, ok := byPath[BrowserEntry]
	if !ok {
		return fmt.Errorf("browser audit: missing %s root", BrowserEntry)
	}
	if _, forbidden := byPath[BunEntry]; forbidden {
		return fmt.Errorf("browser audit: bun entry %s must be absent from a browser generation", BunEntry)
	}
	_ = entry
	asset, ok := byPath[AssetPath]
	if !ok {
		return fmt.Errorf("browser audit: missing %s", AssetPath)
	}
	files := map[string]string{}
	for _, artifact := range artifacts {
		if artifact.Runtime || !strings.HasSuffix(artifact.Path, ".ts") {
			continue
		}
		if len(artifact.NativeImports) != 0 {
			return fmt.Errorf("browser audit: %s carries native imports", artifact.Path)
		}
		for _, edge := range artifact.Imports {
			if !strings.HasPrefix(edge, "./") && !strings.HasPrefix(edge, "../") {
				return fmt.Errorf("browser audit: %s has a non-relative edge %q", artifact.Path, edge)
			}
			if !strings.HasSuffix(edge, ".ts") || strings.ContainsAny(edge, "\\?#") {
				return fmt.Errorf("browser audit: %s has a non-module edge %q", artifact.Path, edge)
			}
			resolved := path.Join(path.Dir(artifact.Path), edge)
			for _, denied := range forbiddenRuntimePaths {
				if strings.Contains("/"+resolved, denied) {
					return fmt.Errorf("browser audit: %s reaches server-capability runtime module %s", artifact.Path, resolved)
				}
			}
		}
		for _, token := range serverTokens {
			if strings.Contains(string(artifact.Bytes), token) {
				return fmt.Errorf("browser audit: %s contains host token %q", artifact.Path, token)
			}
		}
		for _, mapping := range artifact.Mappings {
			if forbiddenMappingOperations[mapping.Operation] {
				return fmt.Errorf("browser audit: %s carries server-request mapping %q", artifact.Path, mapping.Operation)
			}
		}
		sum := sha256.Sum256(artifact.Bytes)
		files[artifact.Path] = hex.EncodeToString(sum[:])
	}
	manifest, err := ParseAsset(asset.Bytes)
	if err != nil {
		return fmt.Errorf("browser audit: %w", err)
	}
	if len(manifest.Files) != len(files) {
		return fmt.Errorf("browser audit: asset binds %d modules, generation holds %d", len(manifest.Files), len(files))
	}
	for name, digest := range files {
		if manifest.Files[name] != digest {
			return fmt.Errorf("browser audit: asset digest mismatch for %s", name)
		}
	}
	return nil
}
