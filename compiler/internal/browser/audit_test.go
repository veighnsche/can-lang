package browser

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
)

type graphModule struct {
	path     string
	body     string
	imports  []string
	native   []string
	runtime  bool
	mappings []ir.Mapping
}

// sealGraph builds an audited artifact set: the given modules plus a bound
// asset manifest. Bodies are used verbatim so inventories can agree or
// disagree with them by construction.
func sealGraph(t *testing.T, modules []graphModule, extra ...ir.Artifact) []ir.Artifact {
	t.Helper()
	var artifacts []ir.Artifact
	files := map[string]string{}
	for _, module := range modules {
		artifacts = append(artifacts, ir.Artifact{
			Path: module.path, Bytes: []byte(module.body),
			Imports: module.imports, NativeImports: module.native,
			Runtime: module.runtime, Mappings: module.mappings,
		})
		if !module.runtime && strings.HasSuffix(module.path, ".ts") {
			sum := sha256.Sum256([]byte(module.body))
			files[module.path] = hex.EncodeToString(sum[:])
		}
	}
	artifacts = append(artifacts, extra...)
	asset, err := AssetBytes(files)
	if err != nil {
		t.Fatal(err)
	}
	return append(artifacts, ir.Artifact{Path: AssetPath, Bytes: asset})
}

func auditFails(t *testing.T, artifacts []ir.Artifact, wants ...string) string {
	t.Helper()
	err := AuditArtifacts(artifacts)
	if err == nil {
		t.Fatal("violation admitted")
	}
	for _, want := range wants {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("diagnostic %q loses %q", err.Error(), want)
		}
	}
	return err.Error()
}

func TestAuditAdmitsQuotedHostData(t *testing.T) {
	runtime := "runtime/r-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	body := "import \"./program/state.ts\";\n" +
		"export const a = \"Bun.\";\n" +
		"export const b = 'require(';\n" +
		"export const c = `node: introduction`;\n" +
		"export const d = \"Bun.stdin\";\n" +
		"// Bun.write(x) process.env.HOME require(\"node:fs\")\n" +
		"/* import(\"./evil.ts\") eval(x) */\n" +
		"export const re = /Bun\\.write|require\\(/;\n" +
		"export async function $canBrowserMain(): Promise<void> {}\n"
	artifacts := sealGraph(t, []graphModule{
		{path: BrowserEntry, body: body, imports: []string{"./program/state.ts"}},
		{path: "program/state.ts", body: "import \"../" + runtime + "/domain.ts\";\nexport function $canInitialize(): void {}\n", imports: []string{"../" + runtime + "/domain.ts"}},
		{path: runtime + "/domain.ts", body: "export const note = \"node: introduction\";\nexport function x(): void {}\n", runtime: true},
	})
	if err := AuditArtifacts(artifacts); err != nil {
		t.Fatalf("quoted host data rejected: %v", err)
	}
}

func TestAuditRejectsHostOperations(t *testing.T) {
	cases := map[string]struct {
		body string
		want string
	}{
		"bun member":     {"export const x = Bun.write(out, line);\n", "Bun.write"},
		"bun bare":       {"export const host = Bun;\n", "Bun"},
		"process member": {"export const home = process.env.HOME;\n", "process.env"},
		"require call":   {"export const fs = require(\"node:fs\");\n", "require"},
		"dynamic import": {"export const lazy = await import(\"./lazy.ts\");\n", "import()"},
		"eval call":      {"export const v = eval(\"1+1\");\n", "eval"},
		"function ctor":  {"export const f = new Function(\"return 1\");\n", "Function"},
		"global host":    {"export const w = globalThis.Bun;\n", "globalThis.Bun"},
	}
	for name, fixture := range cases {
		t.Run(name, func(t *testing.T) {
			artifacts := sealGraph(t, []graphModule{
				{path: BrowserEntry, body: "import \"./program/state.ts\";\n" + fixture.body, imports: []string{"./program/state.ts"}},
				{path: "program/state.ts", body: "export function $canInitialize(): void {}\n"},
			})
			auditFails(t, artifacts, "browser.ts:2:", "host operation "+fixture.want, "reachable via browser.ts")
		})
	}
}

func TestAuditRejectsNativeEdges(t *testing.T) {
	artifacts := sealGraph(t, []graphModule{
		{path: BrowserEntry, body: "import { types } from \"node:util\";\nexport const x = 1;\n", imports: []string{"node:util"}},
	})
	auditFails(t, artifacts, "native edge", "node:util")
}

func TestAuditInspectsReachableRuntimeBodies(t *testing.T) {
	runtime := "runtime/r-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	reachable := sealGraph(t, []graphModule{
		{path: BrowserEntry, body: "import \"./" + runtime + "/helper.ts\";\nexport async function $canBrowserMain(): Promise<void> {}\n", imports: []string{"./" + runtime + "/helper.ts"}},
		{path: runtime + "/helper.ts", body: "export const x = Bun.write(out, line);\n", runtime: true},
	})
	auditFails(t, reachable, runtime+"/helper.ts", "Bun.write", "reachable via browser.ts -> "+runtime+"/helper.ts")
	// Unreachable inventory modules are not inspected: the shared private
	// runtime ships in every generation, reachable or not.
	unreachable := sealGraph(t, []graphModule{
		{path: BrowserEntry, body: "export async function $canBrowserMain(): Promise<void> {}\n"},
		{path: runtime + "/helper.ts", body: "export const x = Bun.write(out, line);\n", runtime: true},
	})
	if err := AuditArtifacts(unreachable); err != nil {
		t.Fatalf("unreachable inventory rejected: %v", err)
	}
}

func TestForbiddenModuleMatching(t *testing.T) {
	runtime := func(path string) ir.Artifact { return ir.Artifact{Path: path, Runtime: true} }
	generated := func(path string) ir.Artifact { return ir.Artifact{Path: path} }
	for path, denied := range map[string]bool{
		"runtime/r-0/platform/sql/pool.ts":          true,
		"runtime/r-0/platform/process/spawn.ts":     true,
		"runtime/r-0/platform/files/read.ts":        true,
		"runtime/r-0/platform/crypto/keys.ts":       true,
		"runtime/r-0/ai/typesafe.ts":                true,
		"runtime/r-0/platform/env.ts":               true,
		"runtime/r-0/environment.ts":                true,
		"runtime/r-0/platform/s3.ts":                true,
		"runtime/r-0/platform/server.ts":            true,
		"runtime/r-0/platform/io.ts":                true,
		"runtime/r-0/platform/cli.ts":               true,
		"runtime/r-0/platform/websocket.ts":         true,
		"runtime/r-0/entry.ts":                      true,
		"runtime/r-0/browser/entry.ts":              false,
		"runtime/r-0/domain.ts":                     false,
		"runtime/r-0/platform/router.ts":            false,
		"runtime/r-0/platform/log.ts":               false,
		"runtime/r-0/transport/stream/lifecycle.ts": false,
	} {
		if _, forbidden := forbiddenModule(runtime(path)); forbidden != denied {
			t.Fatalf("forbiddenModule(%s) = %v, want %v", path, forbidden, denied)
		}
	}
	// Bare inventory names match runtime paths only; directory-qualified
	// server domains match anywhere.
	if _, forbidden := forbiddenModule(generated("program/entry.ts")); forbidden {
		t.Fatal("generated program/entry.ts forbidden")
	}
	if _, forbidden := forbiddenModule(generated("packages/p-0/platform/sql/x.ts")); !forbidden {
		t.Fatal("generated server-domain path admitted")
	}
	if _, forbidden := forbiddenModule(generated("packages/p-0/s-0.ts")); forbidden {
		t.Fatal("generated module forbidden")
	}
}

func TestRuntimeRelativeShapes(t *testing.T) {
	for path, want := range map[string]string{
		"runtime/r-abc/domain.ts":          "domain.ts",
		"runtime/domain.ts":                "domain.ts",
		"gen/runtime/r-abc/a/b.ts":         "a/b.ts",
		"runtime/r-abc/browser/reflect.ts": "browser/reflect.ts",
	} {
		rel, ok := runtimeRelative(ir.Artifact{Path: path, Runtime: true})
		if !ok || rel != want {
			t.Fatalf("runtimeRelative(%s) = %q, %v, want %q", path, rel, ok, want)
		}
	}
	if _, ok := runtimeRelative(ir.Artifact{Path: "program/state.ts"}); ok {
		t.Fatal("generated path resolves as runtime")
	}
	if _, ok := runtimeRelative(ir.Artifact{Path: "runtime/r-abc/domain.ts"}); ok {
		t.Fatal("unflagged runtime path resolves")
	}
}

func TestAuditRejectsTransitiveForbiddenModule(t *testing.T) {
	runtime := "runtime/r-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	artifacts := sealGraph(t, []graphModule{
		{path: BrowserEntry, body: "import \"./program/state.ts\";\nexport async function $canBrowserMain(): Promise<void> {}\n", imports: []string{"./program/state.ts"}},
		{path: "program/state.ts", body: "import \"../" + runtime + "/mid.ts\";\nexport function $canInitialize(): void {}\n", imports: []string{"../" + runtime + "/mid.ts"}},
		{path: runtime + "/mid.ts", body: "import \"./platform/sql/pool.ts\";\nexport function mid(): void {}\n", imports: []string{"./platform/sql/pool.ts"}, runtime: true},
		{path: runtime + "/platform/sql/pool.ts", body: "export function pool(): void {}\n", runtime: true},
	})
	// The bundler might tree-shake the middle module; the audit must not
	// rely on that. The forbidden module fails while still reachable.
	auditFails(t, artifacts, "server-capability runtime module "+runtime+"/platform/sql/pool.ts", "SQL pools and transactions are server-only")
}

func TestAuditAppliesProfileOverlay(t *testing.T) {
	runtime := "runtime/r-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	entry := graphModule{path: BrowserEntry, body: "import \"./" + runtime + "/domain.ts\";\nexport async function $canBrowserMain(): Promise<void> {}\n", imports: []string{"./" + runtime + "/domain.ts"}}
	canonical := graphModule{
		path:   runtime + "/domain.ts",
		body:   "import { createHash } from \"node:crypto\";\nexport function legacy(): void {}\n",
		native: []string{"node:crypto"}, runtime: true,
	}
	alternate := graphModule{
		path:    runtime + "/browser/domain.ts",
		body:    "export function createDomainRuntime(): unknown { return null; }\n",
		runtime: true,
	}
	redirected := sealGraph(t, []graphModule{entry, canonical, alternate})
	if err := AuditArtifacts(redirected); err != nil {
		t.Fatalf("overlaid canonical rejected: %v", err)
	}
	// Without the alternate the canonical body ships as-is and fails
	// closed on its own native edge.
	missing := sealGraph(t, []graphModule{entry, canonical})
	auditFails(t, missing, "native edge", "node:crypto")
}

func TestOverlayShippingCoversSealedProfile(t *testing.T) {
	runtime := "runtime/r-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	for canonical, alternate := range map[string]string{
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
	} {
		if got := OverlayShipping(runtime+"/"+canonical, true); got != runtime+"/"+alternate {
			t.Fatalf("OverlayShipping(%s) = %s, want %s", canonical, got, alternate)
		}
	}
	for _, untouched := range []struct {
		path    string
		runtime bool
	}{
		{"program/state.ts", false},
		{BrowserEntry, false},
		{runtime + "/completion.ts", true},
		{runtime + "/platform/html.ts", true},
		{runtime + "/assert/lineage.ts", true},
		{"runtime/r-other/domain.ts", false},
		{"", true},
	} {
		if got := OverlayShipping(untouched.path, untouched.runtime); got != untouched.path {
			t.Fatalf("OverlayShipping(%q) = %q, want unchanged", untouched.path, got)
		}
	}
}

func TestAuditRejectsUnknownEdges(t *testing.T) {
	t.Run("lexed but undeclared", func(t *testing.T) {
		artifacts := sealGraph(t, []graphModule{
			{path: BrowserEntry, body: "import \"./program/state.ts\";\nimport \"./sneaky.ts\";\nexport async function $canBrowserMain(): Promise<void> {}\n", imports: []string{"./program/state.ts"}},
			{path: "program/state.ts", body: "export function $canInitialize(): void {}\n"},
			{path: "packages/sneaky.ts", body: "export const x = 1;\n"},
		})
		auditFails(t, artifacts, "unaccounted edge \"./sneaky.ts\"")
	})
	t.Run("declared but absent", func(t *testing.T) {
		artifacts := sealGraph(t, []graphModule{
			{path: BrowserEntry, body: "export async function $canBrowserMain(): Promise<void> {}\n", imports: []string{"./program/state.ts"}},
			{path: "program/state.ts", body: "export function $canInitialize(): void {}\n"},
		})
		auditFails(t, artifacts, "declares edge \"./program/state.ts\" with no matching import")
	})
	t.Run("missing target", func(t *testing.T) {
		artifacts := sealGraph(t, []graphModule{
			{path: BrowserEntry, body: "import \"./gone.ts\";\nexport async function $canBrowserMain(): Promise<void> {}\n", imports: []string{"./gone.ts"}},
		})
		auditFails(t, artifacts, "unaccounted edge \"./gone.ts\"", "no module gone.ts")
	})
	t.Run("non module edge", func(t *testing.T) {
		artifacts := sealGraph(t, []graphModule{
			{path: BrowserEntry, body: "import \"./data.json\";\nexport async function $canBrowserMain(): Promise<void> {}\n", imports: []string{"./data.json"}},
		})
		auditFails(t, artifacts, "non-module edge")
	})
}

func TestAuditRejectsUnreachableGeneratedHostOp(t *testing.T) {
	artifacts := sealGraph(t, []graphModule{
		{path: BrowserEntry, body: "export async function $canBrowserMain(): Promise<void> {}\n"},
		{path: "packages/p-0/s-0.ts", body: "export const x = process.env.HOME;\n"},
	})
	auditFails(t, artifacts, "packages/p-0/s-0.ts", "process.env", "unreachable from main")
}

func TestAuditRejectsSecretCanaries(t *testing.T) {
	runtime := "runtime/r-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	t.Run("quoted canary in generated code", func(t *testing.T) {
		artifacts := sealGraph(t, []graphModule{
			{path: BrowserEntry, body: "export const leak = \"__CAN_SECRET_CANARY__\";\n"},
		})
		auditFails(t, artifacts, "secret canary", "__CAN_SECRET_CANARY__", "browser.ts:1:")
	})
	t.Run("canary in reachable runtime", func(t *testing.T) {
		artifacts := sealGraph(t, []graphModule{
			{path: BrowserEntry, body: "import \"./" + runtime + "/helper.ts\";\nexport async function $canBrowserMain(): Promise<void> {}\n", imports: []string{"./" + runtime + "/helper.ts"}},
			{path: runtime + "/helper.ts", body: "export const key = \"-----BEGIN PRIVATE KEY-----\\nMII...\";\n", runtime: true},
		})
		auditFails(t, artifacts, "secret canary", "BEGIN PRIVATE KEY")
	})
	t.Run("canary in comment", func(t *testing.T) {
		artifacts := sealGraph(t, []graphModule{
			{path: BrowserEntry, body: "// CANARY-SECRET-DO-NOT-SHIP\nexport const x = 1;\n"},
		})
		auditFails(t, artifacts, "secret canary", "CANARY-SECRET-DO-NOT-SHIP")
	})
}

func TestAuditVerifiesTypeOnlyEdgesWithoutTraversing(t *testing.T) {
	runtime := "runtime/r-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	t.Run("type-only target not inspected", func(t *testing.T) {
		artifacts := sealGraph(t, []graphModule{
			{path: BrowserEntry, body: "import type { T } from \"./" + runtime + "/types.ts\";\nexport async function $canBrowserMain(): Promise<void> {}\n", imports: []string{"./" + runtime + "/types.ts"}},
			{path: runtime + "/types.ts", body: "export const x = Bun.write(out, line);\n", runtime: true},
		})
		// The edge is verified (declared, resolving) but erased before
		// bundling, so its target body cannot execute and is not walked.
		if err := AuditArtifacts(artifacts); err != nil {
			t.Fatalf("type-only edge rejected: %v", err)
		}
	})
	t.Run("undeclared type-only edge fails", func(t *testing.T) {
		artifacts := sealGraph(t, []graphModule{
			{path: BrowserEntry, body: "import type { T } from \"./program/state.ts\";\nexport async function $canBrowserMain(): Promise<void> {}\n"},
			{path: "program/state.ts", body: "export type T = number;\n"},
		})
		auditFails(t, artifacts, "unaccounted edge")
	})
}

func TestAuditReportsMappingOrigin(t *testing.T) {
	body := "import \"./program/state.ts\";\nexport const x = 1;\nexport const y = Bun.write(out, line);\n"
	artifacts := sealGraph(t, []graphModule{
		{
			path: BrowserEntry, body: body, imports: []string{"./program/state.ts"},
			mappings: []ir.Mapping{
				{Line: 2, Column: 0, Source: "can.project.root/app/main.can", Start: 10, End: 20, Operation: "call"},
			},
		},
		{path: "program/state.ts", body: "export function $canInitialize(): void {}\n"},
	})
	auditFails(t, artifacts,
		"browser.ts:3:17", "Bun.write",
		"originating can can.project.root/app/main.can [10:20] operation call",
		"reachable via browser.ts")
}

func TestAuditAcceptsPinnedVendorScript(t *testing.T) {
	script, err := os.ReadFile(filepath.Join("..", "..", "..", "distribution", "assets", "htmx-4.0.0.min.js"))
	if err != nil {
		t.Fatal(err)
	}
	scan, err := ScanModule(script)
	if err != nil {
		t.Fatalf("pinned script does not lex: %v", err)
	}
	if len(scan.Edges) != 0 || len(scan.Findings) != 0 {
		t.Fatalf("pinned script surface: %+v %+v", scan.Edges, scan.Findings)
	}
	artifacts := sealGraph(t, []graphModule{
		{path: BrowserEntry, body: "export async function $canBrowserMain(): Promise<void> {}\n"},
	}, ir.Artifact{Path: "assets/e484d9171a9db30a39c8f16e3d709d4137f3211c659f8e6125816635033d593f/htmx-4.0.0.min.js", Bytes: script})
	if err := AuditArtifacts(artifacts); err != nil {
		t.Fatalf("pinned vendor script rejected: %v", err)
	}
}

func TestAuditRejectsAssetViolations(t *testing.T) {
	seal := func(t *testing.T, artifact ir.Artifact) []ir.Artifact {
		t.Helper()
		return sealGraph(t, []graphModule{
			{path: BrowserEntry, body: "export async function $canBrowserMain(): Promise<void> {}\n"},
		}, artifact)
	}
	t.Run("script host operation", func(t *testing.T) {
		auditFails(t, seal(t, ir.Artifact{Path: "assets/abc/script.js", Bytes: []byte("export const x = Bun.write(o, l);\n")}),
			"asset assets/abc/script.js", "Bun.write")
	})
	t.Run("script edge", func(t *testing.T) {
		auditFails(t, seal(t, ir.Artifact{Path: "assets/abc/script.js", Bytes: []byte("import \"./other.js\";\n")}),
			"unaccounted edge", "self-contained")
	})
	t.Run("script canary", func(t *testing.T) {
		auditFails(t, seal(t, ir.Artifact{Path: "assets/abc/script.js", Bytes: []byte("export const k = \"__CAN_SECRET_CANARY__\";\n")}),
			"secret canary")
	})
	t.Run("text canary", func(t *testing.T) {
		auditFails(t, seal(t, ir.Artifact{Path: "assets/abc/style.css", Bytes: []byte("/* CANARY-SECRET-DO-NOT-SHIP */\n")}),
			"secret canary")
	})
	t.Run("declared imports", func(t *testing.T) {
		auditFails(t, seal(t, ir.Artifact{Path: "assets/abc/script.js", Bytes: []byte("export const x = 1;\n"), Imports: []string{"./other.js"}}),
			"carries module imports")
	})
	t.Run("clean assets pass", func(t *testing.T) {
		artifacts := sealGraph(t, []graphModule{
			{path: BrowserEntry, body: "export async function $canBrowserMain(): Promise<void> {}\n"},
		},
			ir.Artifact{Path: "assets/abc/script.js", Bytes: []byte("export const x = 1;\n")},
			ir.Artifact{Path: "assets/abc/style.css", Bytes: []byte(".x{color:red}\n")},
			ir.Artifact{Path: "assets/abc/icon.png", Bytes: []byte{0x89, 'P', 'N', 'G', 0, 1}},
		)
		if err := AuditArtifacts(artifacts); err != nil {
			t.Fatalf("clean assets rejected: %v", err)
		}
	})
}

func TestAuditRejectsUnaccountedArtifact(t *testing.T) {
	artifacts := sealGraph(t, []graphModule{
		{path: BrowserEntry, body: "export async function $canBrowserMain(): Promise<void> {}\n"},
	}, ir.Artifact{Path: "notes.txt", Bytes: []byte("stray\n")})
	auditFails(t, artifacts, "unaccounted pre-bundle artifact", "notes.txt")
}

func TestAuditRejectsDuplicateArtifact(t *testing.T) {
	artifacts := sealGraph(t, []graphModule{
		{path: BrowserEntry, body: "export async function $canBrowserMain(): Promise<void> {}\n"},
	})
	duplicated := append(append([]ir.Artifact(nil), artifacts...), artifacts[0])
	auditFails(t, duplicated, "duplicate artifact")
}

// sealedProfileFiles pins the UP07 inventory consumed by the pre-bundle
// audit: the seven browser alternates plus their resolved closure.
var sealedProfileFiles = []string{
	"runtime/browser/callable.ts",
	"runtime/browser/coordination.ts",
	"runtime/browser/diagnostics.ts",
	"runtime/browser/domain.ts",
	"runtime/browser/entry.ts",
	"runtime/browser/owner.ts",
	"runtime/browser/reflect.ts",
	"runtime/catalogue.ts",
	"runtime/completion.ts",
	"runtime/data.ts",
	"runtime/diagnostics-core.ts",
	"runtime/domain-core.ts",
	"runtime/failure.ts",
	"runtime/owner-core.ts",
	"runtime/primitive.ts",
	"runtime/vendor/trace-mapping.ts",
}

func runtimeInventory(t *testing.T) map[string][]string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "runtime", "modules.json"))
	if err != nil {
		t.Fatal(err)
	}
	var inventory struct {
		Modules map[string][]string `json:"modules"`
	}
	if err := json.Unmarshal(raw, &inventory); err != nil {
		t.Fatal(err)
	}
	return inventory.Modules
}

func TestAuditAcceptsSealedProfileFromDisk(t *testing.T) {
	inventory := runtimeInventory(t)
	runtime := "runtime/r-sealed0123456789abcdef0123456789abcdef0123456789abcdef"
	var modules []graphModule
	rels := map[string]bool{}
	for _, file := range sealedProfileFiles {
		rel := strings.TrimPrefix(file, "runtime/")
		rels[rel] = true
		body, err := os.ReadFile(filepath.Join("..", "..", "..", file))
		if err != nil {
			t.Fatal(err)
		}
		var imports, native []string
		for _, spec := range inventory[rel] {
			if strings.HasPrefix(spec, "node:") || strings.HasPrefix(spec, "bun:") {
				native = append(native, spec)
				continue
			}
			imports = append(imports, spec)
		}
		modules = append(modules, graphModule{path: runtime + "/" + rel, body: string(body), imports: imports, native: native, runtime: true})
	}
	// The audit walks value edges with the profile overlay applied, so
	// canonical bodies and their native edges are never inspected: only
	// the alternates ship. The canonicals still ride along as edge
	// targets, exactly as in a full generation. Assert the inventory
	// shape the walk relies on.
	canonicals := []string{"reflect.ts", "domain.ts", "diagnostics.ts", "owner.ts", "callable.ts", "coordination.ts", "entry.ts"}
	for _, rel := range canonicals {
		if _, ok := rels["browser/"+rel]; !ok {
			t.Fatalf("sealed profile omits browser/%s", rel)
		}
		body, err := os.ReadFile(filepath.Join("..", "..", "..", "runtime", rel))
		if err != nil {
			t.Fatal(err)
		}
		var imports, native []string
		for _, spec := range inventory[rel] {
			if strings.HasPrefix(spec, "node:") || strings.HasPrefix(spec, "bun:") {
				native = append(native, spec)
				continue
			}
			imports = append(imports, spec)
		}
		modules = append(modules, graphModule{path: runtime + "/" + rel, body: string(body), imports: imports, native: native, runtime: true})
	}
	// completion.ts re-exports a type from assert/context.ts. Type-only
	// edges resolve but are never traversed, so the assertion module
	// rides along as an edge target only; its body stays uninspected.
	assertBody, err := os.ReadFile(filepath.Join("..", "..", "..", "runtime", "assert", "context.ts"))
	if err != nil {
		t.Fatal(err)
	}
	var assertImports, assertNative []string
	for _, spec := range inventory["assert/context.ts"] {
		if strings.HasPrefix(spec, "node:") || strings.HasPrefix(spec, "bun:") {
			assertNative = append(assertNative, spec)
			continue
		}
		assertImports = append(assertImports, spec)
	}
	modules = append(modules, graphModule{path: runtime + "/assert/context.ts", body: string(assertBody), imports: assertImports, native: assertNative, runtime: true})
	roots := []string{"browser/entry.ts", "browser/owner.ts", "browser/callable.ts", "browser/coordination.ts", "browser/reflect.ts"}
	var entryBody strings.Builder
	var entryImports []string
	for _, root := range roots {
		spec := "./" + runtime + "/" + root
		entryImports = append(entryImports, spec)
		entryBody.WriteString("import " + quoteJSON(spec) + ";\n")
	}
	entryBody.WriteString("export async function $canBrowserMain(): Promise<void> {}\n")
	modules = append([]graphModule{{path: BrowserEntry, body: entryBody.String(), imports: entryImports}}, modules...)
	if err := AuditArtifacts(sealGraph(t, modules)); err != nil {
		t.Fatalf("sealed profile rejected: %v", err)
	}
}

func TestAuditRealLogModuleFails(t *testing.T) {
	inventory := runtimeInventory(t)
	runtime := "runtime/r-full0123456789abcdef0123456789abcdef0123456789abcdef01"
	build := func(t *testing.T, without string) []ir.Artifact {
		t.Helper()
		var names []string
		for name := range inventory {
			if name == without {
				continue
			}
			names = append(names, name)
		}
		sort.Strings(names)
		var modules []graphModule
		for _, name := range names {
			body, err := os.ReadFile(filepath.Join("..", "..", "..", "runtime", filepath.FromSlash(name)))
			if err != nil {
				t.Fatal(err)
			}
			var imports, native []string
			for _, spec := range inventory[name] {
				if strings.HasPrefix(spec, "node:") || strings.HasPrefix(spec, "bun:") {
					native = append(native, spec)
					continue
				}
				imports = append(imports, spec)
			}
			modules = append(modules, graphModule{path: runtime + "/" + name, body: string(body), imports: imports, native: native, runtime: true})
		}
		spec := "./" + runtime + "/platform/log.ts"
		entry := graphModule{path: BrowserEntry, body: "import " + quoteJSON(spec) + ";\nexport async function $canBrowserMain(): Promise<void> {}\n", imports: []string{spec}}
		modules = append([]graphModule{entry}, modules...)
		return sealGraph(t, modules)
	}
	// The sealed overlay ships the native log alternate instead of the
	// server-coupled canonical module.
	if err := AuditArtifacts(build(t, "")); err != nil {
		t.Fatalf("overlaid log module rejected: %v", err)
	}
	// Without its alternate the canonical module is inspected as-is and
	// fails honestly with module, span and chain evidence.
	auditFails(t, build(t, "browser/log.ts"), runtime+"/platform/log.ts", "Bun.write", "reachable via browser.ts")
}

func quoteJSON(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

// TestRuntimeInventoryMatchesBodies pins the input contract the audit
// relies on: the pinned module inventory must equal the lexed static edge
// set of every shipped runtime module in both directions, and every
// shipped module must be inventoried. Test sources are not shipped and
// stay outside the inventory.
func TestRuntimeInventoryMatchesBodies(t *testing.T) {
	inventory := runtimeInventory(t)
	for name, declared := range inventory {
		body, err := os.ReadFile(filepath.Join("..", "..", "..", "runtime", filepath.FromSlash(name)))
		if err != nil {
			t.Fatalf("inventoried module %s has no file: %v", name, err)
		}
		scan, err := ScanModule(body)
		if err != nil {
			t.Fatalf("inventoried module %s does not lex: %v", name, err)
		}
		lexed := map[string]bool{}
		for _, edge := range scan.Edges {
			lexed[edge.Specifier] = true
		}
		for _, spec := range declared {
			if !lexed[spec] {
				t.Fatalf("inventory edge %q of %s has no matching import in the body", spec, name)
			}
		}
		for spec := range lexed {
			found := false
			for _, candidate := range declared {
				if candidate == spec {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("body edge %q of %s is missing from the inventory", spec, name)
			}
		}
	}
	root := filepath.Join("..", "..", "..", "runtime")
	var stray []string
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".ts") {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		if strings.HasSuffix(rel, ".test.ts") || strings.HasPrefix(rel, "test/") {
			return nil
		}
		if _, ok := inventory[rel]; !ok {
			stray = append(stray, rel)
		}
		return nil
	})
	if len(stray) != 0 {
		t.Fatalf("shipped modules without inventory entries: %v", stray)
	}
}
