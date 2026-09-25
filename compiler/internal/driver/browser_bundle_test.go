package driver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/browser"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/project"
)

func TestNormalizeBundleScriptBindsTrailer(t *testing.T) {
	out, err := normalizeBundleScript("browser.js", []byte("export const x = 1;\n//# debugId=ABCDEF0123456789\n"))
	if err != nil {
		t.Fatal(err)
	}
	if want := "export const x = 1;\n//# sourceMappingURL=browser.js.map\n"; string(out) != want {
		t.Fatalf("script = %q", out)
	}
	again, err := normalizeBundleScript("browser.js", out)
	if err != nil {
		t.Fatal(err)
	}
	if string(again) != string(out) {
		t.Fatal("trailer binding is not idempotent")
	}
}

func TestNormalizeBundleScriptRejects(t *testing.T) {
	for name, data := range map[string][]byte{
		"empty":         {},
		"only trailers": []byte("//# debugId=ABC\n//# sourceMappingURL=x.js.map\n"),
	} {
		if _, err := normalizeBundleScript("browser.js", data); err == nil {
			t.Fatalf("%s admitted", name)
		}
	}
}

func writeBundleTree(t *testing.T, files map[string]string) (source, out string) {
	t.Helper()
	root := t.TempDir()
	source = filepath.Join(root, "stage")
	out = filepath.Join(source, "can-bundle-out")
	for name, body := range files {
		full := filepath.Join(source, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(out, 0700); err != nil {
		t.Fatal(err)
	}
	return source, out
}

func TestNormalizeBundleMapRewritesSources(t *testing.T) {
	source, out := writeBundleTree(t, map[string]string{
		"browser.ts":                 "export const x = 1;\n",
		"runtime/r-1/browser/log.ts": "export const y = 2;\n",
	})
	staged := map[string][]byte{"browser.ts": {}, "runtime/r-1/browser/log.ts": {}}
	raw := []byte(`{"version":3,"sources":["../browser.ts","../runtime/r-1/browser/log.ts"],"sourcesContent":["a","b"],"mappings":"AAAA;AACA","debugId":"ABC","names":[]}`)
	normalized, err := normalizeBundleMap(source, out, staged, "browser.js", raw)
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Version        int      `json:"version"`
		File           string   `json:"file"`
		Sources        []string `json:"sources"`
		SourcesContent []string `json:"sourcesContent"`
		Names          []string `json:"names"`
		Mappings       string   `json:"mappings"`
	}
	if err := json.Unmarshal(normalized, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Version != 3 || decoded.File != "browser.js" || decoded.Mappings != "AAAA;AACA" {
		t.Fatalf("map = %s", normalized)
	}
	if len(decoded.Sources) != 2 || decoded.Sources[0] != "browser.ts" || decoded.Sources[1] != "runtime/r-1/browser/log.ts" {
		t.Fatalf("sources = %q", decoded.Sources)
	}
	if strings.Contains(string(normalized), "debugId") {
		t.Fatal("debugId survives normalization")
	}
	again, err := normalizeBundleMap(source, out, staged, "browser.js", raw)
	if err != nil || string(again) != string(normalized) {
		t.Fatal("map normalization is not deterministic")
	}
}

func TestNormalizeBundleMapRejects(t *testing.T) {
	source, out := writeBundleTree(t, map[string]string{"browser.ts": "export const x = 1;\n"})
	staged := map[string][]byte{"browser.ts": {}}
	cases := map[string]string{
		"remote source":    `{"version":3,"sources":["https://x/y.ts"],"sourcesContent":["a"],"mappings":"AAAA","names":[]}`,
		"escaping source":  `{"version":3,"sources":["../../outside.ts"],"sourcesContent":["a"],"mappings":"AAAA","names":[]}`,
		"unstaged source":  `{"version":3,"sources":["../missing.ts"],"sourcesContent":["a"],"mappings":"AAAA","names":[]}`,
		"shim source":      `{"version":3,"sources":["node:util"],"sourcesContent":["a"],"mappings":"AAAA","names":[]}`,
		"unknown field":    `{"version":3,"sources":["../browser.ts"],"sourcesContent":["a"],"mappings":"AAAA","names":[],"sourceRoot":"/"}`,
		"content mismatch": `{"version":3,"sources":["../browser.ts"],"sourcesContent":[],"mappings":"AAAA","names":[]}`,
		"no mappings":      `{"version":3,"sources":["../browser.ts"],"sourcesContent":["a"],"mappings":"","names":[]}`,
		"trailing data":    `{"version":3,"sources":["../browser.ts"],"sourcesContent":["a"],"mappings":"AAAA","names":[]} {}`,
		"trailing token":   `{"version":3,"sources":["../browser.ts"],"sourcesContent":["a"],"mappings":"AAAA","names":[]} x`,
		"not json":         `not json`,
	}
	for name, raw := range cases {
		if _, err := normalizeBundleMap(source, out, staged, "browser.js", []byte(raw)); err == nil {
			t.Fatalf("%s admitted", name)
		}
	}
}

func TestResolveBundleSource(t *testing.T) {
	source, out := writeBundleTree(t, map[string]string{"runtime/r-1/a.ts": "x\n"})
	if got, err := resolveBundleSource(source, out, "../runtime/r-1/a.ts"); err != nil || got != "runtime/r-1/a.ts" {
		t.Fatalf("relative = %q, %v", got, err)
	}
	absolute := filepath.Join(source, "runtime", "r-1", "a.ts")
	if got, err := resolveBundleSource(source, out, absolute); err != nil || got != "runtime/r-1/a.ts" {
		t.Fatalf("absolute = %q, %v", got, err)
	}
	for name, spec := range map[string]string{
		"remote":   "https://example.com/a.ts",
		"protocol": "//example.com/a.ts",
		"data":     "data:text/plain,x",
		"bare":     "node:util",
		"escape":   "../../a.ts",
		"empty":    "",
	} {
		if _, err := resolveBundleSource(source, out, spec); err == nil {
			t.Fatalf("%s admitted", name)
		}
	}
}

func TestRelativeModuleSpecifier(t *testing.T) {
	for _, tc := range []struct{ from, target, want string }{
		{".", "runtime/r-1/browser/log.ts", "./runtime/r-1/browser/log.ts"},
		{"program", "runtime/r-1/browser/log.ts", "../runtime/r-1/browser/log.ts"},
		{"runtime/r-1/codec", "runtime/r-1/browser/formats.ts", "../browser/formats.ts"},
		{"runtime/r-1/browser", "runtime/r-1/browser/log.ts", "./log.ts"},
	} {
		got, err := relativeModuleSpecifier(tc.from, tc.target)
		if err != nil || got != tc.want {
			t.Fatalf("specifier(%q, %q) = %q, %v", tc.from, tc.target, got, err)
		}
	}
	if _, err := relativeModuleSpecifier("program", "data.json"); err == nil {
		t.Fatal("non-module redirect admitted")
	}
}

func TestApplyBrowserOverlay(t *testing.T) {
	runtime := "runtime/r-1"
	modules := map[string][]byte{
		"program/state.ts":           []byte("import { createLog } from \"../runtime/r-1/platform/log.ts\";\nimport type { X } from '../runtime/r-1/platform/log.ts';\nimport { ok } from \"../runtime/r-1/completion.ts\";\n"),
		runtime + "/platform/log.ts": []byte("export function createLog(): void {}\n"),
		runtime + "/browser/log.ts":  []byte("export function createLog(): void {}\n"),
		runtime + "/completion.ts":   []byte("export const ok = 1;\n"),
	}
	inventories := map[string][]string{
		"program/state.ts":           {"../runtime/r-1/platform/log.ts", "../runtime/r-1/completion.ts"},
		runtime + "/platform/log.ts": {},
		runtime + "/browser/log.ts":  {},
		runtime + "/completion.ts":   {},
	}
	flags := map[string]bool{
		"program/state.ts":           false,
		runtime + "/platform/log.ts": true,
		runtime + "/browser/log.ts":  true,
		runtime + "/completion.ts":   true,
	}
	rewritten, err := applyBrowserOverlay(modules, inventories, flags)
	if err != nil {
		t.Fatal(err)
	}
	got := string(rewritten["program/state.ts"])
	if strings.Contains(got, "platform/log.ts") {
		t.Fatalf("canonical edge survives: %q", got)
	}
	if count := strings.Count(got, "../runtime/r-1/browser/log.ts"); count != 2 {
		t.Fatalf("expected both edges redirected, got %q", got)
	}
	if !strings.Contains(got, "../runtime/r-1/completion.ts") {
		t.Fatalf("unrelated edge disturbed: %q", got)
	}
	if strings.Contains(got, `"`) == false || strings.Contains(got, `'`) == false {
		t.Fatalf("quote styles disturbed: %q", got)
	}
}

func TestApplyBrowserOverlaySkipsSelfAndMissing(t *testing.T) {
	runtime := "runtime/r-1"
	modules := map[string][]byte{
		// A type-only self-hop through the canonical asset types must not
		// rewrite the stub into importing itself.
		runtime + "/browser/assets.ts":  []byte("import type { AssetServer } from \"../platform/assets.ts\";\nexport function createAssets(): AssetServer { throw new Error(); }\n"),
		runtime + "/platform/assets.ts": []byte("export type AssetServer = {};\n"),
		// An overlaid edge whose alternate is absent stays untouched; the
		// bundled canonical fails the post-bundle audit instead.
		"program/state.ts":           []byte("import { x } from \"../runtime/r-1/platform/log.ts\";\n"),
		runtime + "/platform/log.ts": []byte("export const x = 1;\n"),
	}
	inventories := map[string][]string{
		runtime + "/browser/assets.ts":  {"../platform/assets.ts"},
		runtime + "/platform/assets.ts": {},
		"program/state.ts":              {"../runtime/r-1/platform/log.ts"},
		runtime + "/platform/log.ts":    {},
	}
	flags := map[string]bool{
		runtime + "/browser/assets.ts":  true,
		runtime + "/platform/assets.ts": true,
		"program/state.ts":              false,
		runtime + "/platform/log.ts":    true,
	}
	rewritten, err := applyBrowserOverlay(modules, inventories, flags)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(rewritten[runtime+"/browser/assets.ts"]); !strings.Contains(got, "../platform/assets.ts") {
		t.Fatalf("self edge rewritten: %q", got)
	}
	if got := string(rewritten["program/state.ts"]); !strings.Contains(got, "../runtime/r-1/platform/log.ts") {
		t.Fatalf("missing alternate edge rewritten: %q", got)
	}
}

func TestApplyBrowserOverlayFailsClosed(t *testing.T) {
	modules := map[string][]byte{
		"program/state.ts":            []byte("export const x = 1;\n"),
		"runtime/r-1/platform/log.ts": []byte("export const x = 1;\n"),
		"runtime/r-1/browser/log.ts":  []byte("export const x = 1;\n"),
	}
	inventories := map[string][]string{"program/state.ts": {"../runtime/r-1/platform/log.ts"}}
	flags := map[string]bool{"program/state.ts": false, "runtime/r-1/platform/log.ts": true, "runtime/r-1/browser/log.ts": true}
	if _, err := applyBrowserOverlay(modules, inventories, flags); err == nil {
		t.Fatal("declared but unlexed redirect admitted")
	}
}

func tableFixture(t *testing.T) []ir.Artifact {
	t.Helper()
	index := []byte(`{"schemaVersion":1,"kind":"can.source-index","sources":[{"id":"s1","path":"app/main.can","spans":{"call:1:2":{"start":1,"end":2,"line":1,"column":2,"endLine":1,"endColumn":3,"operation":"call"}}}],"modules":[{"path":"browser.ts","segments":[{"line":1,"column":0,"source":"s1","name":"call:1:2"}]}]}`)
	return []ir.Artifact{
		{Path: "browser.ts", Bytes: []byte("export const x = 1;\n//# sourceMappingURL=browser.ts.map\n")},
		{Path: "browser.ts.map", Bytes: []byte(`{"version":3,"file":"browser.ts","sources":["s1"],"sourcesContent":["x"],"names":[],"mappings":"AAAA"}`)},
		{Path: "diagnostics/source-index.json", Bytes: index},
	}
}

func TestSealBrowserDiagnosticTable(t *testing.T) {
	table, err := sealBrowserDiagnosticTable(tableFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		SchemaVersion int                        `json:"schemaVersion"`
		Kind          string                     `json:"kind"`
		Index         json.RawMessage            `json:"index"`
		Maps          map[string]json.RawMessage `json:"maps"`
	}
	if err := json.Unmarshal(table, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.SchemaVersion != 1 || decoded.Kind != "can.diagnostic-table" {
		t.Fatalf("table = %s", table)
	}
	if len(decoded.Maps) != 1 || len(decoded.Maps["browser.ts"]) == 0 {
		t.Fatalf("maps = %s", table)
	}
	again, err := sealBrowserDiagnosticTable(tableFixture(t))
	if err != nil || string(again) != string(table) {
		t.Fatal("table sealing is not deterministic")
	}
}

func TestSealBrowserDiagnosticTableRejects(t *testing.T) {
	full := tableFixture(t)
	drop := func(path string) []ir.Artifact {
		var kept []ir.Artifact
		for _, artifact := range full {
			if artifact.Path != path {
				kept = append(kept, artifact)
			}
		}
		return kept
	}
	if _, err := sealBrowserDiagnosticTable(drop("diagnostics/source-index.json")); err == nil {
		t.Fatal("missing index admitted")
	}
	if _, err := sealBrowserDiagnosticTable(drop("browser.ts.map")); err == nil {
		t.Fatal("missing module map admitted")
	}
	broken := append([]ir.Artifact{}, full...)
	broken[2].Bytes = []byte(`{"schemaVersion":1,"kind":"can.source-index","sources":[],"modules":[]}`)
	if _, err := sealBrowserDiagnosticTable(broken); err == nil {
		t.Fatal("empty module index admitted")
	}
}

func TestSealBrowserEntryTable(t *testing.T) {
	entry := "export const BROWSER_PROFILE = \"browser-main\";\n" + browserTablePlaceholder + "\nvoid $canBrowserMain();\n"
	artifacts := []ir.Artifact{{Path: browser.BrowserEntry, Bytes: []byte(entry)}}
	table := []byte(`{"schemaVersion":1,"kind":"can.diagnostic-table","index":{},"maps":{}}`)
	sealed, err := sealBrowserEntryTable(artifacts, table)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(sealed[0].Bytes), "\n")
	if len(lines) != 4 {
		t.Fatalf("line count changed: %d", len(lines))
	}
	if !strings.Contains(lines[1], `"can.diagnostic-table"`) || strings.Contains(string(sealed[0].Bytes), "Object.freeze([])") {
		t.Fatal("placeholder survives sealing")
	}
	if _, err := sealBrowserEntryTable(artifacts, nil); err == nil {
		t.Fatal("empty table admitted")
	}
	if _, err := sealBrowserEntryTable([]ir.Artifact{{Path: browser.BrowserEntry, Bytes: []byte("no placeholder\n")}}, table); err == nil {
		t.Fatal("missing placeholder admitted")
	}
	doubled := []ir.Artifact{{Path: browser.BrowserEntry, Bytes: []byte(browserTablePlaceholder + "\n" + browserTablePlaceholder + "\n")}}
	if _, err := sealBrowserEntryTable(doubled, table); err == nil {
		t.Fatal("doubled placeholder admitted")
	}
}

func TestAuditBrowserBundleWiring(t *testing.T) {
	clean := browserBundleOutputs{entry: "browser/browser.js", files: map[string][]byte{
		"browser/browser.js":     []byte("\"use strict\";\nexport const answer = 41 + 1;\n//# sourceMappingURL=browser.js.map\n"),
		"browser/browser.js.map": []byte(`{"version":3,"file":"browser.js","sources":["../src/main.can"],"names":[],"mappings":"AAAA"}`),
		"diagnostics/table.json": []byte(`{"schemaVersion":1,"kind":"can.diagnostic-table","index":{},"maps":{}}`),
	}}
	if err := auditBrowserBundle(clean); err != nil {
		t.Fatalf("clean bundle rejected: %v", err)
	}
	hostile := browserBundleOutputs{entry: clean.entry, files: map[string][]byte{}}
	for name, body := range clean.files {
		hostile.files[name] = body
	}
	hostile.files["browser/browser.js"] = []byte("export const home = process.env.HOME;\n//# sourceMappingURL=browser.js.map\n")
	if err := auditBrowserBundle(hostile); err == nil || !strings.Contains(err.Error(), "process.env") {
		t.Fatalf("host operation admitted: %v", err)
	}
	extra := browserBundleOutputs{entry: clean.entry, files: map[string][]byte{}}
	for name, body := range clean.files {
		extra.files[name] = body
	}
	extra.files["browser/extra.js"] = []byte("export const x = 1;\n//# sourceMappingURL=extra.js.map\n")
	if err := auditBrowserBundle(extra); err == nil {
		t.Fatal("unaccounted script admitted")
	}
	leaky := browserBundleOutputs{entry: clean.entry, files: map[string][]byte{}}
	for name, body := range clean.files {
		leaky.files[name] = body
	}
	leaky.files["diagnostics/table.json"] = []byte(`{"records":[{"note":"__CAN_SECRET_CANARY__"}]}`)
	if err := auditBrowserBundle(leaky); err == nil {
		t.Fatal("canary leak admitted")
	}
}

func TestBrowserManifestBytes(t *testing.T) {
	bundle := browserBundleOutputs{entry: "browser/browser.js", files: map[string][]byte{
		"browser/browser.js":     []byte("a\n"),
		"browser/browser.js.map": []byte("b\n"),
		"diagnostics/table.json": []byte("c\n"),
	}}
	graph := &project.Graph{LockSHA256: strings.Repeat("1", 64), Lock: project.Lock{Projects: map[string]project.LockEntry{
		"can.project.dependency/vendor": {Lineage: "vendor", ManifestSHA256: strings.Repeat("2", 64), SourceSHA256: strings.Repeat("3", 64), FixturesSHA256: strings.Repeat("4", 64)},
	}}}
	inputs := BuildInputs{Source: strings.Repeat("a", 64), Dependencies: strings.Repeat("b", 64), Catalogue: strings.Repeat("c", 64), Compiler: strings.Repeat("d", 64), Runtime: strings.Repeat("e", 64), Options: strings.Repeat("f", 64)}
	toolchain := browserToolchain{Target: "bun-1.4.2-darwin-arm64-v1", Version: "1.4.2", Revision: "r", SHA256: strings.Repeat("9", 64), Compiler: inputs.Compiler, Runtime: inputs.Runtime}
	encoded, err := browserManifestBytes(graph, inputs, toolchain, bundle)
	if err != nil {
		t.Fatal(err)
	}
	var decoded browserManifest
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.SchemaVersion != 1 || decoded.Kind != "can.browser-manifest" || decoded.Entry != bundle.entry || decoded.Table != browserBundleTable {
		t.Fatalf("manifest = %s", encoded)
	}
	if len(decoded.BrowserBuildID) != 64 || len(decoded.Files) != 3 {
		t.Fatalf("manifest = %s", encoded)
	}
	for i, file := range decoded.Files {
		sum := sha256.Sum256(bundle.files[file.Path])
		if file.SHA256 != hex.EncodeToString(sum[:]) {
			t.Fatalf("digest mismatch for %s", file.Path)
		}
		if i > 0 && decoded.Files[i-1].Path >= file.Path {
			t.Fatal("files not sorted")
		}
	}
	routes := map[string]string{}
	for _, file := range decoded.Files {
		routes[file.Path] = file.Route
	}
	for name, suffix := range map[string]string{
		"browser/browser.js":     ".js",
		"browser/browser.js.map": ".js.map",
		"diagnostics/table.json": ".json",
	} {
		sum := sha256.Sum256(bundle.files[name])
		if want := "/__can/assets/" + hex.EncodeToString(sum[:]) + suffix; routes[name] != want {
			t.Fatalf("route(%s) = %q, want %q", name, routes[name], want)
		}
	}
	if decoded.Toolchain.Target != toolchain.Target || decoded.Toolchain.Version != "1.4.2" || decoded.Inputs.Source != inputs.Source || decoded.Lock != graph.LockSHA256 {
		t.Fatalf("manifest = %s", encoded)
	}
	if len(decoded.LockedInstances) != 1 || decoded.LockedInstances[0].Instance != "can.project.dependency/vendor" {
		t.Fatalf("manifest = %s", encoded)
	}
	again, err := browserManifestBytes(graph, inputs, toolchain, bundle)
	if err != nil || string(again) != string(encoded) {
		t.Fatal("manifest is not deterministic")
	}
	edited := browserBundleOutputs{entry: bundle.entry, files: map[string][]byte{}}
	for name, body := range bundle.files {
		edited.files[name] = body
	}
	edited.files["browser/browser.js"] = []byte("tampered\n")
	retampered, err := browserManifestBytes(graph, inputs, toolchain, edited)
	if err != nil {
		t.Fatal(err)
	}
	if string(retampered) == string(encoded) {
		t.Fatal("manifest ignores tampering")
	}
}

func TestBrowserAssetRoute(t *testing.T) {
	digest := strings.Repeat("a", 64)
	if got := browserAssetRoute("browser/browser.js", digest); got != "/__can/assets/"+digest+".js" {
		t.Fatalf("route = %q", got)
	}
	if got := browserAssetRoute("browser/browser.js.map", digest); got != "/__can/assets/"+digest+".js.map" {
		t.Fatalf("route = %q", got)
	}
	if got := browserAssetRoute("diagnostics/table.json", digest); got != "/__can/assets/"+digest+".json" {
		t.Fatalf("route = %q", got)
	}
}

func TestBuildBrowserBundleRequiresDistribution(t *testing.T) {
	runtime := &Runtime{}
	artifacts := []ir.Artifact{{Path: browser.BrowserEntry, Bytes: []byte("x\n")}}
	if _, err := runtime.buildBrowserBundle(context.Background(), &project.Graph{}, BuildInputs{}, browserToolchain{}, artifacts, []byte("{}")); err == nil {
		t.Fatal("unbundled runtime admitted")
	}
	if _, err := runtime.buildBrowserBundle(context.Background(), &project.Graph{}, BuildInputs{}, browserToolchain{}, artifacts, nil); err == nil {
		t.Fatal("missing table admitted")
	}
}
