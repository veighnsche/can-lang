package check

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/project"
)

func TestAssetURLResolvesCallerManifestOnly(t *testing.T) {
	root := t.TempDir()
	write := func(name, text string) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("assets/site.css", "body{color:black}\n")
	write("vendor/assets/mark.css", "h1{}\n")
	write("can.errors.json", `{"active":[],"retired":[]}`)
	write("vendor/can.errors.json", `{"active":[],"retired":[]}`)
	write("can.project.json", `{"source_root":"src","error_registry":"can.errors.json","dependencies":{"vendor":"vendor"},"assets":{"site_css":"assets/site.css"}}`)
	write("vendor/can.project.json", `{"source_root":"src","error_registry":"can.errors.json","assets":{"mark":"assets/mark.css","site_css":"assets/mark.css"}}`)
	vendorSource := "package vendor_app\n    provides [foreign]\n    uses [asset, html]\nfn html::url foreign\n    emits [html::invalid_url]\n    asserts\n        owned: => ok\n    match call asset::url(\"mark\")\n        html::invalid_url\n        ok html::url address => ok address\n"
	write("vendor/src/lib.can", vendorSource)
	write("src/main.can", `package app
    provides []
    uses [asset, html]
fn html::url known
    emits [html::invalid_url]
    asserts
        found: => ok
    match call asset::url("site_css")
        html::invalid_url
        ok html::url address => ok address
fn html::url unknown
    emits [html::invalid_url]
    asserts
        missing: => html::invalid_url("missing")
    match call asset::url("absent")
        html::invalid_url
        ok html::url address => ok address
fn html::url foreign
    emits [html::invalid_url]
    asserts
        unowned: => html::invalid_url("unowned")
    match call asset::url("mark")
        html::invalid_url
        ok html::url address => ok address
fn void main
    emits []
    given
        str[] args
    asserts
        empty: [] => ok
    ok
`)
	vendorManifest, err := os.ReadFile(filepath.Join(root, "vendor/can.project.json"))
	if err != nil {
		t.Fatal(err)
	}
	registry, err := project.ParseRegistry([]byte(`{"active":[],"retired":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	digest, err := project.SourceDigest([]project.SourceBytes{{Path: "lib.can", Bytes: []byte(vendorSource)}})
	if err != nil {
		t.Fatal(err)
	}
	fixtureDigest, err := project.FixtureDigest(nil)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := json.Marshal(map[string]any{"dependencies": map[string]any{"vendor": map[string]any{"path": "vendor", "manifest_sha256": project.Digest(vendorManifest), "source_sha256": digest, "fixtures_sha256": fixtureDigest, "error_registry": registry}}})
	if err != nil {
		t.Fatal(err)
	}
	write("can.lock.json", string(lock))
	graph, err := project.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	program, err := CheckProgram(graph)
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]ir.AssetResolution{}
	var walk func(*ir.Invocation)
	walk = func(call *ir.Invocation) {
		if call == nil {
			return
		}
		for _, step := range call.Steps {
			if step.Asset != nil {
				found[step.Asset.URL+"|"+step.Asset.Reason] = *step.Asset
			}
		}
	}
	for _, fn := range program.Functions {
		if fn.Region == nil {
			continue
		}
		var block func(*ir.Block)
		var comp func(*ir.Completion)
		comp = func(c *ir.Completion) {
			if c == nil {
				return
			}
			walk(c.Call)
			if c.Match != nil {
				walk(c.Match.Call)
			}
		}
		block = func(b *ir.Block) {
			if b == nil {
				return
			}
			for i := range b.Steps {
				walk(b.Steps[i].Call)
			}
			comp(b.Terminal)
		}
		block(fn.Region.Body)
	}
	var owned, missing, unowned bool
	for _, resolution := range found {
		switch resolution.Reason {
		case "":
			if strings.Contains(resolution.URL, "/site.css") {
				owned = true
			}
		case "missing":
			missing = true
		case "unowned":
			unowned = true
		}
	}
	if !owned || !missing || !unowned {
		names := []string{}
		for _, fn := range program.Functions {
			label := fn.Identity()
			if fn.Region != nil && fn.Region.Body != nil && fn.Region.Body.Terminal != nil {
				label += "/" + string(fn.Region.Body.Terminal.Kind)
			}
			names = append(names, label)
		}
		t.Fatalf("resolutions %#v functions %v", found, names)
	}
	bad := `package app
    provides []
    uses [asset, html]
fn html::url dynamic
    emits [html::invalid_url]
    given
        str name
    asserts
        sample: "site_css" => ok
    match call asset::url(name)
        html::invalid_url
        ok html::url address => ok address
fn void main
    emits []
    given
        str[] args
    asserts
        empty: [] => ok
    ok
`
	write("src/main.can", bad)
	graph, err = project.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = CheckProgram(graph); err == nil {
		t.Fatal("dynamic asset name admitted")
	}
}

func TestAssetPageProgram(t *testing.T) {
	root := t.TempDir()
	source, err := os.ReadFile(filepath.Join("..", "..", "testdata", "current", "assets", "page.can"))
	if err != nil {
		t.Fatal(err)
	}
	write := func(name, text string) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("can.project.json", `{"source_root":"src","error_registry":"can.errors.json","assets":{"site_css":"assets/site.css"}}`)
	write("can.errors.json", `{"active":[],"retired":[]}`)
	write("assets/site.css", "body{color:black}\n")
	write("src/main.can", string(source))
	graph, err := project.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = CheckProgram(graph); err != nil {
		t.Fatal(err)
	}
}
