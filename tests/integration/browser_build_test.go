package integration

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/distribution"
)

const browserPureMain = `package app
    provides []
    uses [text, codec, bytes, strings]
record point
    int x
    int y
fn point load
    emits [codec::invalid_data]
    given
        str text
    asserts
        sample: "{\"x\":1,\"y\":2}" => ok point(1, 2)
    match chain
        call bytes::from_utf8(text) as bytes::buffer raw
        call codec::decode_json<point>(raw) as point found
        codec::invalid_data
        ok => ok found
fn void main
    emits []
    given
        str[] arguments
    asserts
        empty: [] => ok
    match call load(call strings::join(arguments, ","))
        codec::invalid_data => ok
        ok point found => ok
`

const browserHelperPackage = `package strings
    provides [join]
    uses [text]
fn str join
    emits []
    given
        str[] parts
        str separator
    asserts
        sample: [], "," => ok ""
        pair: ["a", "b"], "," => ok "a,b"
    ok call text::join(parts, separator)
`

func TestBrowserBuildTarget(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged browser execution")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "browser-integration")
	if err != nil {
		t.Fatal(err)
	}
	launcher := filepath.Join(bundle, "bin/canlc")
	outside := t.TempDir()
	link := filepath.Join(outside, "canlc")
	if err = os.Symlink(launcher, link); err != nil {
		t.Fatal(err)
	}
	newProject := func(t *testing.T) (root string, write func(name, text string)) {
		t.Helper()
		root, err = filepath.EvalSymlinks(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		write = func(name, text string) {
			t.Helper()
			p := filepath.Join(root, name)
			if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, []byte(text), 0600); err != nil {
				t.Fatal(err)
			}
		}
		write("can.project.json", `{"source_root":"src","error_registry":"can.errors.json"}`)
		write("can.errors.json", `{"active":[],"retired":[]}`)
		return root, write
	}
	run := func(args ...string) (int, string, string) {
		t.Helper()
		command := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", append([]string{"-p", "(version 1)(allow default)(deny network*)", link}, args...)...)
		command.Dir = outside
		command.Env = []string{"PATH=/nonexistent", "HOME=" + outside}
		var stdout, stderr bytes.Buffer
		command.Stdout = &stdout
		command.Stderr = &stderr
		err := command.Run()
		if err == nil {
			return 0, stdout.String(), stderr.String()
		}
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return exit.ExitCode(), stdout.String(), stderr.String()
		}
		t.Fatal(err)
		return 0, "", ""
	}

	t.Run("pure multi-package project ships the browser root", func(t *testing.T) {
		root, write := newProject(t)
		write("src/app/main.can", browserPureMain)
		write("src/strings/join.can", browserHelperPackage)
		code, out, diag := run("build", "--target", "browser", root)
		if code != 0 {
			t.Fatalf("browser build: %d %s %s", code, out, diag)
		}
		var report struct {
			Target    string `json:"target"`
			BuildID   string `json:"buildID"`
			Directory string `json:"directory"`
			Entry     string `json:"entry"`
			Asset     string `json:"asset"`
		}
		if err := json.Unmarshal([]byte(out), &report); err != nil {
			t.Fatal(err)
		}
		if report.Target != "browser" || report.Entry != "browser.ts" || report.Asset != "browser/asset.json" {
			t.Fatalf("report = %+v", report)
		}
		if _, err := os.Stat(filepath.Join(report.Directory, "entry.ts")); !os.IsNotExist(err) {
			t.Fatal("browser generation carries the bun entry")
		}
		entry, err := os.ReadFile(filepath.Join(report.Directory, "browser.ts"))
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{`BROWSER_PROFILE = "browser-main"`, "$canBrowserMain", "$canInitialize()"} {
			if !strings.Contains(string(entry), want) {
				t.Fatalf("browser root lacks %q", want)
			}
		}
		for _, token := range []string{"process.", "Bun.", "require(", "node:"} {
			if strings.Contains(string(entry), token) {
				t.Fatalf("browser root contains host token %q", token)
			}
		}
		assetRaw, err := os.ReadFile(filepath.Join(report.Directory, "browser/asset.json"))
		if err != nil {
			t.Fatal(err)
		}
		var manifest struct {
			SchemaVersion int               `json:"schemaVersion"`
			Kind          string            `json:"kind"`
			Profile       string            `json:"profile"`
			Entry         string            `json:"entry"`
			ContentSHA256 string            `json:"contentSHA256"`
			Files         map[string]string `json:"files"`
		}
		if err := json.Unmarshal(assetRaw, &manifest); err != nil {
			t.Fatalf("asset invalid: %v", err)
		}
		if manifest.SchemaVersion != 1 || manifest.Kind != "can.browser-asset" || manifest.Profile != "browser-main" || manifest.Entry != "browser.ts" {
			t.Fatalf("asset identity = %+v", manifest)
		}
		names := make([]string, 0, len(manifest.Files))
		for name := range manifest.Files {
			names = append(names, name)
		}
		sort.Strings(names)
		hash := sha256.New()
		for _, name := range names {
			hash.Write([]byte(name))
			hash.Write([]byte{0})
			hash.Write([]byte(manifest.Files[name]))
			hash.Write([]byte{0})
		}
		if hex.EncodeToString(hash.Sum(nil)) != manifest.ContentSHA256 {
			t.Fatal("asset content digest mismatch")
		}
		for name, digest := range manifest.Files {
			raw, err := os.ReadFile(filepath.Join(report.Directory, filepath.FromSlash(name)))
			if err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(raw)
			if hex.EncodeToString(sum[:]) != digest {
				t.Fatalf("asset digest mismatch for %s", name)
			}
			if strings.Contains(string(raw), "process.exitCode") {
				t.Fatalf("%s carries the bun supervisor", name)
			}
		}
		manifestRaw, err := os.ReadFile(filepath.Join(report.Directory, "manifest.json"))
		if err != nil {
			t.Fatal(err)
		}
		var generation struct {
			Entry   string              `json:"entry"`
			Imports map[string][]string `json:"imports"`
		}
		if err := json.Unmarshal(manifestRaw, &generation); err != nil {
			t.Fatal(err)
		}
		if generation.Entry != "browser.ts" {
			t.Fatalf("generation entry = %q", generation.Entry)
		}
		for from, edges := range generation.Imports {
			for _, edge := range edges {
				if strings.Contains(edge, "/platform/sql/") || strings.Contains(edge, "/platform/files/") ||
					strings.Contains(edge, "/platform/process/") || strings.Contains(edge, "/platform/env") ||
					strings.Contains(edge, "/platform/crypto/") || strings.Contains(edge, "/platform/server") {
					t.Fatalf("emitted edge %s -> %s reaches a server module", from, edge)
				}
			}
		}
	})

	t.Run("same project still builds for bun", func(t *testing.T) {
		root, write := newProject(t)
		write("src/app/main.can", browserPureMain)
		write("src/strings/join.can", browserHelperPackage)
		code, out, diag := run("build", "--target", "bun", root)
		if code != 0 {
			t.Fatalf("bun build: %d %s %s", code, out, diag)
		}
		var report struct {
			Target string `json:"target"`
			Entry  string `json:"entry"`
		}
		if err := json.Unmarshal([]byte(out), &report); err != nil {
			t.Fatal(err)
		}
		if report.Target != "bun" || report.Entry != "entry.ts" {
			t.Fatalf("report = %+v", report)
		}
	})

	t.Run("combined flags", func(t *testing.T) {
		root, write := newProject(t)
		write("src/app/main.can", browserPureMain)
		write("src/strings/join.can", browserHelperPackage)
		code, out, diag := run("build", "--assert-timeout-ms", "60000", "--target", "browser", root)
		if code != 0 {
			t.Fatalf("combined flags: %d %s %s", code, out, diag)
		}
	})

	t.Run("worker target rejected", func(t *testing.T) {
		root, write := newProject(t)
		write("src/app/main.can", browserPureMain)
		for _, args := range [][]string{{"build", "--target", "worker", root}, {"build", "--target", "browser-worker", root}} {
			code, _, diag := run(args...)
			if code != 2 || !strings.Contains(diag, "usage:") || !strings.Contains(diag, "no worker profile") {
				t.Fatalf("worker target: %v %d %q", args, code, diag)
			}
		}
	})

	t.Run("server capability diagnosed", func(t *testing.T) {
		root, write := newProject(t)
		write("src/main.can", `package app
    provides []
    uses [env, http]
fn void main
    emits [env::invalid_name, http::credentials_missing]
    given
        str[] arguments
    asserts
        empty: [] => ok
    match call env::required("HOME")
        when
            empty: "HOME" => ok "fixture"
        env::invalid_name
        http::credentials_missing
        ok str value => ok
`)
		code, out, diag := run("build", "--target", "browser", root)
		if code != 1 || out != "" || !strings.Contains(diag, "can.std.env@1::required") {
			t.Fatalf("capability gate: %d %q %q", code, out, diag)
		}
		code, _, diag = run("build", root)
		if code != 0 {
			t.Fatalf("bun build of the same project: %d %s", code, diag)
		}
	})

	t.Run("empty app builds a verified bundle", func(t *testing.T) {
		root, write := newProject(t)
		write("src/main.can", `package app
    provides []
    uses []
fn void main
    emits []
    given
        str[] arguments
    asserts
        empty: [] => ok
    ok
`)
		code, out, diag := run("build", "--target", "browser", root)
		if code != 0 {
			t.Fatalf("empty browser build: %d %s %s", code, out, diag)
		}
		var report struct {
			BuildID   string `json:"buildID"`
			Directory string `json:"directory"`
		}
		if err := json.Unmarshal([]byte(out), &report); err != nil {
			t.Fatal(err)
		}
		assertVerifiedBundle(t, report.Directory)
	})

	t.Run("bundle publishes verified outputs", func(t *testing.T) {
		root, write := newProject(t)
		write("src/app/main.can", browserPureMain)
		write("src/strings/join.can", browserHelperPackage)
		code, out, diag := run("build", "--target", "browser", root)
		if code != 0 {
			t.Fatalf("browser build: %d %s %s", code, out, diag)
		}
		var report struct {
			BuildID   string `json:"buildID"`
			Directory string `json:"directory"`
		}
		if err := json.Unmarshal([]byte(out), &report); err != nil {
			t.Fatal(err)
		}
		assertVerifiedBundle(t, report.Directory)
		// The entry ships the sealed table, never the empty placeholder.
		entry, err := os.ReadFile(filepath.Join(report.Directory, "browser.ts"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(entry), `"can.diagnostic-table"`) || strings.Contains(string(entry), "sources: Object.freeze([])") {
			t.Fatal("browser entry does not seal the checked table")
		}
		// No assertion roots or test-only edges ship in the browser tree.
		if _, err := os.Stat(filepath.Join(report.Directory, "assertions")); !os.IsNotExist(err) {
			t.Fatal("browser generation ships assertion roots")
		}
		manifestRaw, err := os.ReadFile(filepath.Join(report.Directory, "manifest.json"))
		if err != nil {
			t.Fatal(err)
		}
		var generation struct {
			Imports map[string][]string `json:"imports"`
		}
		if err := json.Unmarshal(manifestRaw, &generation); err != nil {
			t.Fatal(err)
		}
		for from, edges := range generation.Imports {
			if strings.HasPrefix(from, "runtime/") {
				continue
			}
			for _, edge := range edges {
				if strings.Contains(edge, "/assert/") {
					t.Fatalf("generated edge %s -> %s reaches test code", from, edge)
				}
			}
		}
	})

	t.Run("repeated clean builds are identical", func(t *testing.T) {
		build := func(t *testing.T) (string, string) {
			t.Helper()
			root, write := newProject(t)
			write("src/app/main.can", browserPureMain)
			write("src/strings/join.can", browserHelperPackage)
			code, out, diag := run("build", "--target", "browser", root)
			if code != 0 {
				t.Fatalf("browser build: %d %s %s", code, out, diag)
			}
			var report struct {
				BuildID   string `json:"buildID"`
				Directory string `json:"directory"`
			}
			if err := json.Unmarshal([]byte(out), &report); err != nil {
				t.Fatal(err)
			}
			return report.BuildID, report.Directory
		}
		firstID, firstDir := build(t)
		secondID, secondDir := build(t)
		if firstID != secondID {
			t.Fatalf("clean builds diverge: %s vs %s", firstID, secondID)
		}
		for _, name := range []string{"browser/browser.js", "browser/browser.js.map", "diagnostics/table.json", "browser/manifest.json"} {
			first, err := os.ReadFile(filepath.Join(firstDir, filepath.FromSlash(name)))
			if err != nil {
				t.Fatal(err)
			}
			second, err := os.ReadFile(filepath.Join(secondDir, filepath.FromSlash(name)))
			if err != nil {
				t.Fatal(err)
			}
			if string(first) != string(second) {
				t.Fatalf("clean builds diverge in %s", name)
			}
		}
	})

	t.Run("failing assertions prevent browser publication", func(t *testing.T) {
		root, write := newProject(t)
		write("src/app/main.can", strings.Replace(browserPureMain, "ok point(1, 2)", "ok point(9, 9)", 1))
		write("src/strings/join.can", browserHelperPackage)
		code, _, diag := run("build", "--target", "browser", root)
		if code == 0 || !strings.Contains(diag, "build verification failed") {
			t.Fatalf("wrong assertion published: %d %s", code, diag)
		}
		if _, err := os.Stat(filepath.Join(root, "dist/current.json")); !os.IsNotExist(err) {
			t.Fatal("failed browser build selected production current")
		}
	})

	t.Run("tampered bundle prevents publication", func(t *testing.T) {
		root, write := newProject(t)
		write("src/app/main.can", browserPureMain)
		write("src/strings/join.can", browserHelperPackage)
		code, out, diag := run("build", "--target", "browser", root)
		if code != 0 {
			t.Fatalf("browser build: %d %s %s", code, out, diag)
		}
		var report struct {
			Directory string `json:"directory"`
		}
		if err := json.Unmarshal([]byte(out), &report); err != nil {
			t.Fatal(err)
		}
		before, err := os.ReadFile(filepath.Join(root, "dist/current.json"))
		if err != nil {
			t.Fatal(err)
		}
		entry := filepath.Join(report.Directory, "browser/browser.js")
		raw, err := os.ReadFile(entry)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(entry, append(raw, []byte("//tamper\n")...), 0600); err != nil {
			t.Fatal(err)
		}
		code, _, diag = run("build", "--target", "browser", root)
		if code == 0 {
			t.Fatalf("tampered bundle published: %s", diag)
		}
		after, err := os.ReadFile(filepath.Join(root, "dist/current.json"))
		if err != nil {
			t.Fatal(err)
		}
		if string(after) != string(before) {
			t.Fatal("tampered rebuild moved production current")
		}
	})

	t.Run("missing map prevents publication", func(t *testing.T) {
		root, write := newProject(t)
		write("src/app/main.can", browserPureMain)
		write("src/strings/join.can", browserHelperPackage)
		code, out, diag := run("build", "--target", "browser", root)
		if code != 0 {
			t.Fatalf("browser build: %d %s %s", code, out, diag)
		}
		var report struct {
			Directory string `json:"directory"`
		}
		if err := json.Unmarshal([]byte(out), &report); err != nil {
			t.Fatal(err)
		}
		before, err := os.ReadFile(filepath.Join(root, "dist/current.json"))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(filepath.Join(report.Directory, "browser/browser.js.map")); err != nil {
			t.Fatal(err)
		}
		code, _, diag = run("build", "--target", "browser", root)
		if code == 0 {
			t.Fatalf("map-less bundle published: %s", diag)
		}
		after, err := os.ReadFile(filepath.Join(root, "dist/current.json"))
		if err != nil {
			t.Fatal(err)
		}
		if string(after) != string(before) {
			t.Fatal("map-less rebuild moved production current")
		}
	})

	t.Run("profile-unavailable operations diagnose", func(t *testing.T) {
		root, write := newProject(t)
		write("src/main.can", `package app
    provides []
    uses [cookie, option]
fn option::value<str> session_id
    emits []
    given
        str header
    asserts
        present: "theme=dark; session=A" => ok option::some("A")
    match call cookie::parse(header)
        ok cookie::collection found => match call cookie::get(found, "session")
            ok option::value<str> id => ok id
fn void main
    emits []
    given
        str[] arguments
    asserts
        empty: [] => ok
    match call session_id("session=A")
        ok option::value<str> id => ok
`)
		code, out, diag := run("build", "--target", "browser", root)
		if code != 1 || out != "" || !strings.Contains(diag, "can.std.cookie@1::parse") {
			t.Fatalf("cookie gate: %d %q %q", code, out, diag)
		}
		if _, err := os.Stat(filepath.Join(root, "dist/current.json")); !os.IsNotExist(err) {
			t.Fatal("gated browser build selected production current")
		}
		code, _, diag = run("build", root)
		if code != 0 {
			t.Fatalf("bun build of the same project: %d %s", code, diag)
		}
	})
}

func assertVerifiedBundle(t *testing.T, directory string) {
	t.Helper()
	read := func(name string) []byte {
		t.Helper()
		raw, err := os.ReadFile(filepath.Join(directory, filepath.FromSlash(name)))
		if err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
		return raw
	}
	script := read("browser/browser.js")
	scriptMap := read("browser/browser.js.map")
	tableRaw := read("diagnostics/table.json")
	manifestRaw := read("browser/manifest.json")
	if !strings.HasSuffix(string(script), "//# sourceMappingURL=browser.js.map\n") {
		t.Fatal("bundle entry lacks its map trailer")
	}
	var parsedMap struct {
		Version        int      `json:"version"`
		File           string   `json:"file"`
		Sources        []string `json:"sources"`
		SourcesContent []string `json:"sourcesContent"`
		Names          []string `json:"names"`
		Mappings       string   `json:"mappings"`
	}
	decoder := json.NewDecoder(bytes.NewReader(scriptMap))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&parsedMap); err != nil || parsedMap.Version != 3 || parsedMap.File != "browser.js" || len(parsedMap.Sources) == 0 || len(parsedMap.Sources) != len(parsedMap.SourcesContent) || parsedMap.Mappings == "" {
		t.Fatalf("bundle map invalid: %v", err)
	}
	for _, source := range parsedMap.Sources {
		if !strings.HasSuffix(source, ".ts") || strings.Contains(source, ":") {
			t.Fatalf("bundle map names non-logical source %q", source)
		}
	}
	var table struct {
		SchemaVersion int    `json:"schemaVersion"`
		Kind          string `json:"kind"`
		Index         struct {
			Kind    string `json:"kind"`
			Modules []struct {
				Path string `json:"path"`
			} `json:"modules"`
		} `json:"index"`
		Maps map[string]json.RawMessage `json:"maps"`
	}
	if err := json.Unmarshal(tableRaw, &table); err != nil || table.SchemaVersion != 1 || table.Kind != "can.diagnostic-table" || table.Index.Kind != "can.source-index" || len(table.Index.Modules) == 0 {
		t.Fatalf("diagnostic table invalid: %v", err)
	}
	for _, module := range table.Index.Modules {
		published, ok := table.Maps[module.Path]
		if !ok || len(published) == 0 {
			t.Fatalf("table lacks the %s map", module.Path)
		}
		sealed, err := os.ReadFile(filepath.Join(directory, filepath.FromSlash(module.Path+".map")))
		if err != nil {
			t.Fatal(err)
		}
		if string(published) != string(sealed) {
			t.Fatalf("table map for %s differs from the sealed map", module.Path)
		}
	}
	var manifest struct {
		SchemaVersion  int    `json:"schemaVersion"`
		Kind           string `json:"kind"`
		BrowserBuildID string `json:"browserBuildId"`
		Entry          string `json:"entry"`
		Table          string `json:"table"`
		Files          []struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
			Route  string `json:"route"`
		} `json:"files"`
		Toolchain struct {
			Target   string `json:"target"`
			Version  string `json:"version"`
			Revision string `json:"revision"`
			SHA256   string `json:"sha256"`
		} `json:"toolchain"`
		Inputs struct {
			Source, Dependencies, Catalogue, Compiler, Runtime, Options string
		} `json:"inputs"`
		Lock string `json:"lock"`
	}
	if err := json.Unmarshal(manifestRaw, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.SchemaVersion != 1 || manifest.Kind != "can.browser-manifest" || len(manifest.BrowserBuildID) != 64 || manifest.Entry != "browser/browser.js" || manifest.Table != "diagnostics/table.json" {
		t.Fatalf("browser manifest invalid: %s", manifestRaw)
	}
	pinned := distribution.PinnedTarget()
	if manifest.Toolchain.Target != pinned.TargetID || manifest.Toolchain.Version != pinned.Runtime.Version || manifest.Toolchain.Revision != pinned.Runtime.Revision || manifest.Toolchain.SHA256 != pinned.Runtime.SHA256 {
		t.Fatalf("manifest toolchain %+v diverges from the pinned target", manifest.Toolchain)
	}
	for _, digest := range []string{manifest.Inputs.Source, manifest.Inputs.Dependencies, manifest.Inputs.Catalogue, manifest.Inputs.Compiler, manifest.Inputs.Runtime, manifest.Inputs.Options} {
		if len(digest) != 64 {
			t.Fatalf("manifest omits input identity: %s", manifestRaw)
		}
	}
	seen := map[string]bool{}
	scripts := map[string]bool{}
	for _, file := range manifest.Files {
		raw, err := os.ReadFile(filepath.Join(directory, filepath.FromSlash(file.Path)))
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(raw)
		if hex.EncodeToString(sum[:]) != file.SHA256 {
			t.Fatalf("manifest digest mismatch for %s", file.Path)
		}
		if !strings.HasPrefix(file.Route, "/__can/assets/"+file.SHA256) {
			t.Fatalf("manifest route for %s is not content addressed", file.Path)
		}
		seen[file.Path] = true
		if strings.HasSuffix(file.Path, ".js") {
			scripts[file.Path] = true
		}
	}
	for _, required := range []string{"browser/browser.js", "browser/browser.js.map", "diagnostics/table.json"} {
		if !seen[required] {
			t.Fatalf("manifest omits %s", required)
		}
	}
	for name := range scripts {
		if !seen[name+".map"] {
			t.Fatalf("published script %s lacks its manifest map", name)
		}
		// Structural host-operation and edge verification of every
		// published script happens inside the build via the post-bundle
		// audit; a successful build implies it passed, and the driver
		// unit tests prove hostile scripts fail it.
		raw, err := os.ReadFile(filepath.Join(directory, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		base := name[strings.LastIndex(name, "/")+1:]
		if !strings.HasSuffix(string(raw), "//# sourceMappingURL="+base+".map\n") {
			t.Fatalf("published script %s lacks its map trailer", name)
		}
	}
}

type wireVectorResult struct {
	Name   string `json:"name"`
	Pass   bool   `json:"pass"`
	Detail string `json:"detail"`
}

func runWireVectors(t *testing.T, ctx context.Context, bun, script string, args ...string) []wireVectorResult {
	t.Helper()
	command := exec.CommandContext(ctx, bun, append([]string{script}, args...)...)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("%s %v: %v %s", script, args, err, stderr.String())
	}
	return parseWireVectors(t, stdout.Bytes())
}

func parseWireVectors(t *testing.T, raw []byte) []wireVectorResult {
	t.Helper()
	var results []wireVectorResult
	if err := json.Unmarshal(raw, &results); err != nil {
		t.Fatalf("invalid vector JSON: %v %.200q", err, string(raw))
	}
	if len(results) == 0 {
		t.Fatal("empty vector results")
	}
	for _, result := range results {
		if !result.Pass {
			t.Fatalf("vector %s failed: %s", result.Name, result.Detail)
		}
	}
	return results
}

func TestBrowserWireCodecParity(t *testing.T) {
	bunPath, err := exec.LookPath("bun")
	if err != nil {
		t.Skip("bun is required for the wire-codec parity harness")
	}
	nodePath, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is required for the browser harness")
	}
	sourceRoot, _ := filepath.Abs("../..")
	browserDir := filepath.Join(sourceRoot, "tests/integration/browser")
	if _, err := os.Stat(filepath.Join(browserDir, "node_modules/playwright/package.json")); err != nil {
		t.Skip("run bun ci in tests/integration/browser for the pinned harness")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	probeCtx, probeCancel := context.WithTimeout(context.Background(), time.Minute)
	defer probeCancel()
	probe := exec.CommandContext(probeCtx, nodePath, "--input-type=module", "-e", `import("playwright").then(async ({chromium}) => { const browser = await chromium.launch(); console.log(browser.version()); await browser.close(); })`)
	probe.Dir = browserDir
	if result, err := probe.CombinedOutput(); err != nil {
		if strings.Contains(string(result), "Executable doesn't exist") {
			t.Skip("install the pinned playwright chromium for browser evidence")
		}
		t.Fatalf("browser probe: %v %s", err, result)
	}
	vectorsModule := filepath.Join(sourceRoot, "runtime/test/browser-wire-vectors.ts")

	reference := runWireVectors(t, ctx, bunPath, "-e",
		`import { runBrowserWireVectors } from "`+vectorsModule+`"; console.log(JSON.stringify(runBrowserWireVectors()))`)

	outdir := t.TempDir()
	build := exec.CommandContext(ctx, bunPath, filepath.Join(browserDir, "build-vectors.mjs"),
		filepath.Join(browserDir, "codec-vectors-entry.ts"), outdir)
	build.Dir = browserDir
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("vector bundle: %v %s", err, out)
	}
	bundle := filepath.Join(outdir, "codec-vectors-entry.js")

	bundled := runWireVectors(t, ctx, bunPath, "-e",
		`await import("`+bundle+`"); console.log(JSON.stringify(globalThis.__canWireResults))`)
	if len(bundled) != len(reference) {
		t.Fatalf("bundle vectors = %d, reference = %d", len(bundled), len(reference))
	}
	for i := range reference {
		if bundled[i] != reference[i] {
			t.Fatalf("bundle divergence at %s: %+v vs %+v", reference[i].Name, bundled[i], reference[i])
		}
	}

	attempt := func(t *testing.T, name string, required bool) {
		t.Helper()
		command := exec.CommandContext(ctx, nodePath, filepath.Join(browserDir, "codec-parity.mjs"), name, bundle)
		command.Dir = browserDir
		out, err := command.Output()
		if err != nil {
			if exit, ok := err.(*exec.ExitError); ok {
				t.Logf("%s unavailable, skipping: %s", name, strings.TrimSpace(string(exit.Stderr)))
			} else {
				t.Logf("%s unavailable, skipping: %v", name, err)
			}
			if required {
				t.Fatalf("%s is required for the parity matrix", name)
			}
			t.Skipf("%s unavailable: %v", name, err)
			return
		}
		var observed struct {
			Browser   string             `json:"browser"`
			Version   string             `json:"version"`
			UserAgent string             `json:"userAgent"`
			RawJSON   bool               `json:"rawJSON"`
			Results   []wireVectorResult `json:"results"`
		}
		if err := json.Unmarshal(out, &observed); err != nil {
			t.Fatalf("%s: invalid harness JSON: %v %.200q", name, err, string(out))
		}
		t.Logf("%s %s rawJSON=%v %s", observed.Browser, observed.Version, observed.RawJSON, observed.UserAgent)
		if !observed.RawJSON {
			t.Logf("%s lacks JSON.rawJSON, skipping", name)
			if required {
				t.Fatalf("%s is required for the parity matrix", name)
			}
			t.Skipf("%s lacks JSON.rawJSON", name)
			return
		}
		if len(observed.Results) != len(reference) {
			t.Fatalf("%s vectors = %d, reference = %d", name, len(observed.Results), len(reference))
		}
		for i := range reference {
			if !observed.Results[i].Pass {
				t.Fatalf("%s vector %s failed: %s", name, observed.Results[i].Name, observed.Results[i].Detail)
			}
			if observed.Results[i] != reference[i] {
				t.Fatalf("%s divergence at %s: %+v vs %+v", name, reference[i].Name, observed.Results[i], reference[i])
			}
		}
	}

	t.Run("chromium", func(t *testing.T) { attempt(t, "chromium", true) })
	t.Run("webkit", func(t *testing.T) { attempt(t, "webkit", false) })
	t.Run("firefox", func(t *testing.T) { attempt(t, "firefox", false) })
}
