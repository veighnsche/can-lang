package emit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/project"
)

func TestAssetPageEmitsDigestManifest(t *testing.T) {
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
	program, err := check.CheckProgram(graph)
	if err != nil {
		t.Fatal(err)
	}
	artifacts, err := ProgramModules(program, "runtime", httpDependencies(t))
	if err != nil {
		t.Fatal(err)
	}
	var state string
	var css, script bool
	for _, artifact := range artifacts {
		text := string(artifact.Bytes)
		if artifact.Path == "program/state.ts" {
			state = text
		}
		if strings.HasPrefix(artifact.Path, "assets/") && strings.HasSuffix(artifact.Path, "/site.css") && text == "body{color:black}\n" {
			parts := strings.Split(artifact.Path, "/")
			css = len(parts) == 3 && parts[1] != ""
		}
		if strings.HasSuffix(artifact.Path, "/htmx-4.0.0.min.js") {
			script = strings.Contains(artifact.Path, "assets/")
		}
		if strings.Contains(text, `declareAsset("/__can/project/`) && !strings.Contains(text, "asset::url") {
			// Generated calls carry the resolved URL, not a runtime name lookup.
		}
	}
	if !strings.Contains(state, "/__can/assets/htmx-4.0.0.min.js") || !strings.Contains(state, "sha384-BvJpBiO8Kh31EqtJe5DRIeWrHWnCGkwytKs9NKFi86Hhw96dEqdEMzZDeK9iEGTc") || !strings.Contains(state, "$canAssets") || !css || !script {
		t.Fatalf("missing asset emission css=%v script=%v", css, script)
	}
	if !strings.Contains(state, "declareAsset") && !containsAny(artifacts, "declareAsset") {
		t.Fatal("asset::url was not lowered to the private URL constructor")
	}
	if containsAny(artifacts, "$canCallableInstance()") {
		t.Fatal("static asset call emitted an empty callable-instance lookup")
	}
}

func containsAny(artifacts []ir.Artifact, needle string) bool {
	for _, artifact := range artifacts {
		if strings.Contains(string(artifact.Bytes), needle) {
			return true
		}
	}
	return false
}
