package driver

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/browser"
	"github.com/veighnsche/can-lang/compiler/internal/project"
)

var pairingScript = []byte("console.log(\"paired\");\n//# sourceMappingURL=browser.js.map\n")
var pairingMap = []byte(`{"version":3,"file":"browser.js","sources":["browser.ts"],"sourcesContent":["export const browser = 1;\n"],"names":[],"mappings":"AAAA"}`)
var pairingTable = []byte(`{"schemaVersion":1,"kind":"can.diagnostic-table","index":{},"maps":{}}`)

func pairingBundle() map[string][]byte {
	return map[string][]byte{
		"browser/browser.js":     append([]byte(nil), pairingScript...),
		"browser/browser.js.map": append([]byte(nil), pairingMap...),
		"diagnostics/table.json": append([]byte(nil), pairingTable...),
	}
}

func pairingInputs() BuildInputs {
	return BuildInputs{
		Source: strings.Repeat("a", 64), Dependencies: strings.Repeat("b", 64),
		Catalogue: strings.Repeat("c", 64), Compiler: strings.Repeat("d", 64),
		Runtime: strings.Repeat("e", 64), Options: strings.Repeat("f", 64),
	}
}

// writeBrowserGeneration mints a self-consistent fake browser generation:
// manifest bytes from the real manifest writer plus a content-bound
// generation manifest over the exact files on disk.
func writeBrowserGeneration(t *testing.T, dir string, files map[string][]byte, lock map[string]project.LockEntry, lockDigest string) string {
	t.Helper()
	graph := &project.Graph{LockSHA256: lockDigest, Lock: project.Lock{Projects: lock}}
	inputs := pairingInputs()
	toolchain := browserToolchain{Target: "test-target", Version: "1.0", Revision: "r", SHA256: strings.Repeat("9", 64), Compiler: inputs.Compiler, Runtime: inputs.Runtime}
	manifest, err := browserManifestBytes(graph, inputs, toolchain, browserBundleOutputs{entry: "browser/browser.js", files: files})
	if err != nil {
		t.Fatal(err)
	}
	onDisk := map[string][]byte{"browser.ts": []byte("export const browser = 1;\n"), browserBundleManifest: manifest}
	for name, data := range files {
		onDisk[name] = data
	}
	for name, data := range onDisk {
		full := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	tree := OutputManifest{SchemaVersion: 1, Kind: "can.output-generation", Inputs: inputs, Entry: browser.BrowserEntry, Files: map[string]string{}, Imports: map[string][]string{"browser.ts": {}}, NativeImports: map[string][]string{}}
	for name, data := range onDisk {
		sum := sha256.Sum256(data)
		tree.Files[name] = hex.EncodeToString(sum[:])
	}
	id, err := manifestID(tree)
	if err != nil {
		t.Fatal(err)
	}
	tree.BuildID = id
	encoded, err := json.MarshalIndent(tree, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), append(encoded, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(dir, "browser", "manifest.json")
}

func pairingServerGraph(lock map[string]project.LockEntry) *project.Graph {
	return &project.Graph{Lock: project.Lock{Projects: lock}}
}

func TestVerifyBrowserManifest(t *testing.T) {
	dir := t.TempDir()
	manifestPath := writeBrowserGeneration(t, dir, pairingBundle(), nil, "")
	pairing, err := verifyBrowserManifest(manifestPath, pairingServerGraph(nil))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(pairing.entry, "/__can/assets/") || !strings.HasSuffix(pairing.entry, ".js") {
		t.Fatalf("entry route = %q", pairing.entry)
	}
	if !strings.HasSuffix(pairing.table, ".json") {
		t.Fatalf("table route = %q", pairing.table)
	}
	if len(pairing.files) != 3 || len(pairing.assets) != 3 {
		t.Fatalf("paired files = %d", len(pairing.files))
	}
	for i, file := range pairing.files {
		asset := pairing.assets[i]
		if file.Route != asset.URL || file.Digest != asset.Digest || file.File != asset.Artifact {
			t.Fatal("paired file and synthetic asset diverge")
		}
		raw, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(pairing.files[i].Logical)))
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(raw)
		if hex.EncodeToString(sum[:]) != file.Digest {
			t.Fatal("paired digest does not match generation bytes")
		}
	}
	report := pairing.pairedReport()
	if report == nil || report.BrowserBuildID != pairing.buildID || report.Entry != pairing.entry || len(report.Files) != 3 {
		t.Fatalf("report = %+v", report)
	}
	if err := pairing.reread(); err != nil {
		t.Fatal(err)
	}
	var record struct {
		SchemaVersion int               `json:"schemaVersion"`
		Kind          string            `json:"kind"`
		Entry         string            `json:"entry"`
		Files         []pairedAssetFile `json:"files"`
	}
	if err := json.Unmarshal(pairing.pairingJSON, &record); err != nil || record.Kind != "can.browser-pairing" || record.Entry != pairing.entry || len(record.Files) != 3 {
		t.Fatalf("pairing record = %s %v", pairing.pairingJSON, err)
	}
}

func TestVerifyBrowserManifestSharedLock(t *testing.T) {
	shared := map[string]project.LockEntry{
		"can.project.lineage/contract": {Lineage: "contract", ManifestSHA256: strings.Repeat("2", 64), SourceSHA256: strings.Repeat("3", 64), FixturesSHA256: strings.Repeat("4", 64)},
	}
	dir := t.TempDir()
	manifestPath := writeBrowserGeneration(t, dir, pairingBundle(), shared, strings.Repeat("1", 64))
	if _, err := verifyBrowserManifest(manifestPath, pairingServerGraph(shared)); err != nil {
		t.Fatalf("matching shared snapshot rejected: %v", err)
	}
	drifted := map[string]project.LockEntry{
		"can.project.lineage/contract": {Lineage: "contract", ManifestSHA256: strings.Repeat("2", 64), SourceSHA256: strings.Repeat("9", 64), FixturesSHA256: strings.Repeat("4", 64)},
	}
	if _, err := verifyBrowserManifest(manifestPath, pairingServerGraph(drifted)); err == nil || !strings.Contains(err.Error(), "can.project.lineage/contract") {
		t.Fatalf("drifted shared instance admitted: %v", err)
	}
	// An empty intersection passes: the dependency-free fixture pairs while
	// the grid migrates onto the shared contract.
	if _, err := verifyBrowserManifest(manifestPath, pairingServerGraph(nil)); err != nil {
		t.Fatalf("empty intersection rejected: %v", err)
	}
}

func TestVerifyBrowserManifestTamper(t *testing.T) {
	write := func(t *testing.T) (string, string) {
		t.Helper()
		dir := t.TempDir()
		return dir, writeBrowserGeneration(t, dir, pairingBundle(), nil, "")
	}
	t.Run("script bytes", func(t *testing.T) {
		dir, manifestPath := write(t)
		if err := os.WriteFile(filepath.Join(dir, "browser", "browser.js"), []byte("tampered\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := verifyBrowserManifest(manifestPath, pairingServerGraph(nil)); err == nil {
			t.Fatal("tampered script admitted")
		}
	})
	t.Run("generation source", func(t *testing.T) {
		dir, manifestPath := write(t)
		if err := os.WriteFile(filepath.Join(dir, "browser.ts"), []byte("tampered\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := verifyBrowserManifest(manifestPath, pairingServerGraph(nil)); err == nil {
			t.Fatal("tampered generation admitted")
		}
	})
	t.Run("manifest identity", func(t *testing.T) {
		_, manifestPath := write(t)
		raw, err := os.ReadFile(manifestPath)
		if err != nil {
			t.Fatal(err)
		}
		raw = append([]byte(nil), raw...)
		raw[len(raw)-2] ^= 0x01
		if err := os.WriteFile(manifestPath, raw, 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := verifyBrowserManifest(manifestPath, pairingServerGraph(nil)); err == nil {
			t.Fatal("tampered manifest admitted")
		}
	})
	t.Run("canary with consistent hashes", func(t *testing.T) {
		files := pairingBundle()
		files["browser/browser.js"] = []byte("const leak = \"__CAN_SECRET_CANARY__\";\n//# sourceMappingURL=browser.js.map\n")
		dir := t.TempDir()
		manifestPath := writeBrowserGeneration(t, dir, files, nil, "")
		if _, err := verifyBrowserManifest(manifestPath, pairingServerGraph(nil)); err == nil {
			t.Fatal("canary bytes admitted")
		}
	})
	t.Run("host operation with consistent hashes", func(t *testing.T) {
		files := pairingBundle()
		files["browser/browser.js"] = []byte("Bun.serve({});\n//# sourceMappingURL=browser.js.map\n")
		dir := t.TempDir()
		manifestPath := writeBrowserGeneration(t, dir, files, nil, "")
		if _, err := verifyBrowserManifest(manifestPath, pairingServerGraph(nil)); err == nil {
			t.Fatal("host operation admitted")
		}
	})
}

func TestVerifyBrowserManifestReferences(t *testing.T) {
	t.Run("missing map", func(t *testing.T) {
		files := pairingBundle()
		delete(files, "browser/browser.js.map")
		dir := t.TempDir()
		manifestPath := writeBrowserGeneration(t, dir, files, nil, "")
		if _, err := verifyBrowserManifest(manifestPath, pairingServerGraph(nil)); err == nil || !strings.Contains(err.Error(), "source map") {
			t.Fatalf("missing map admitted: %v", err)
		}
	})
	t.Run("missing table", func(t *testing.T) {
		files := pairingBundle()
		delete(files, "diagnostics/table.json")
		dir := t.TempDir()
		manifestPath := writeBrowserGeneration(t, dir, files, nil, "")
		if _, err := verifyBrowserManifest(manifestPath, pairingServerGraph(nil)); err == nil || !strings.Contains(err.Error(), "table") {
			t.Fatalf("missing table admitted: %v", err)
		}
	})
	t.Run("unbound manifest", func(t *testing.T) {
		dir := t.TempDir()
		manifestPath := writeBrowserGeneration(t, dir, pairingBundle(), nil, "")
		second := t.TempDir()
		other := writeBrowserGeneration(t, second, pairingBundle(), nil, strings.Repeat("7", 64))
		raw, err := os.ReadFile(other)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(manifestPath, raw, 0600); err != nil {
			t.Fatal(err)
		}
		_ = dir
		if _, err := verifyBrowserManifest(manifestPath, pairingServerGraph(nil)); err == nil {
			t.Fatal("unbound manifest admitted")
		}
	})
	t.Run("wrong path shape", func(t *testing.T) {
		dir := t.TempDir()
		writeBrowserGeneration(t, dir, pairingBundle(), nil, "")
		for _, bad := range []string{filepath.Join(dir, "manifest.json"), filepath.Join(dir, "browser", "browser.js"), filepath.Join(dir, "elsewhere.json")} {
			if _, err := verifyBrowserManifest(bad, pairingServerGraph(nil)); err == nil {
				t.Fatalf("path %s admitted", bad)
			}
		}
	})
}

func TestBrowserPairingReread(t *testing.T) {
	dir := t.TempDir()
	manifestPath := writeBrowserGeneration(t, dir, pairingBundle(), nil, "")
	pairing, err := verifyBrowserManifest(manifestPath, pairingServerGraph(nil))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "browser", "browser.js"), pairingScript, 0600); err != nil {
		t.Fatal(err)
	}
	if err := pairing.reread(); err != nil {
		t.Fatalf("identical reread rejected: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "diagnostics", "table.json"), []byte("{}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := pairing.reread(); err == nil || !strings.Contains(err.Error(), "changed during the server build") {
		t.Fatalf("changed inputs admitted: %v", err)
	}
}

func TestBindSharedLock(t *testing.T) {
	entry := func(digest string) browserLockedInstance {
		return browserLockedInstance{Instance: "can.project.lineage/contract", Lineage: "contract", ManifestSHA256: digest, SourceSHA256: digest, FixturesSHA256: digest}
	}
	server := func(digest string) *project.Graph {
		return pairingServerGraph(map[string]project.LockEntry{
			"can.project.lineage/contract": {Lineage: "contract", ManifestSHA256: digest, SourceSHA256: digest, FixturesSHA256: digest},
		})
	}
	digest := strings.Repeat("2", 64)
	if err := bindSharedLock([]browserLockedInstance{entry(digest)}, server(digest)); err != nil {
		t.Fatal(err)
	}
	if err := bindSharedLock([]browserLockedInstance{entry(digest)}, server(strings.Repeat("3", 64))); err == nil {
		t.Fatal("drifted digests admitted")
	}
	if err := bindSharedLock(nil, server(digest)); err != nil {
		t.Fatal(err)
	}
	if err := bindSharedLock([]browserLockedInstance{entry(digest)}, pairingServerGraph(nil)); err != nil {
		t.Fatal(err)
	}
	if err := bindSharedLock([]browserLockedInstance{entry(digest), entry(digest)}, server(digest)); err == nil {
		t.Fatal("duplicate instance admitted")
	}
}

func TestPairedRouteExtension(t *testing.T) {
	digest := strings.Repeat("ab", 32)
	for route, ext := range map[string]string{
		"/__can/assets/" + digest + ".js":     ".js",
		"/__can/assets/" + digest + ".js.map": ".js.map",
		"/__can/assets/" + digest + ".json":   ".json",
	} {
		got, gotExt, err := pairedRouteExtension(route)
		if err != nil || got != digest || gotExt != ext {
			t.Fatalf("route %s -> %q %q %v", route, got, gotExt, err)
		}
	}
	for _, bad := range []string{
		"/__can/assets/" + digest + ".css",
		"/__can/assets/short.js",
		"/__can/project/" + digest + "/browser.js",
		"/__can/assets/" + digest + ".JS",
		"/__can/assets/" + digest,
	} {
		if _, _, err := pairedRouteExtension(bad); err == nil {
			t.Fatalf("route %s admitted", bad)
		}
	}
}
