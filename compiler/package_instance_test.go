package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/project"
)

func writeInstanceProject(t *testing.T, root, uses, body string) {
	t.Helper()
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
	write("can.project.json", `{"source_root":"src","project":"shop_root","dependencies":{"left":"libs/left","right":"libs/right"},"error_registry":"can.errors.json"}`)
	write("can.errors.json", `{"active":[],"retired":[]}`)
	write("src/main.can", "package app\n    provides []\n    uses ["+uses+"]\n"+body)
	write("libs/left/can.project.json", `{"source_root":"src","project":"shop_left","error_registry":"can.errors.json"}`)
	write("libs/left/can.errors.json", `{"active":[{"id":1000000,"kind":"model::failed"}],"retired":[]}`)
	write("libs/left/src/model.can", "package model\n    provides [item, failed]\n    uses []\nrecord item\n    int value\nerror 1000000 failed(str reason)\n")
	write("libs/right/can.project.json", `{"source_root":"src","project":"shop_right","error_registry":"can.errors.json"}`)
	write("libs/right/can.errors.json", `{"active":[{"id":1000001,"kind":"model::failed"}],"retired":[]}`)
	write("libs/right/src/model.can", "package model\n    provides [item, failed]\n    uses []\nrecord item\n    str label\nerror 1000001 failed(str reason)\n")
	edges := map[string]any{}
	entries := map[string]any{}
	for _, lib := range []string{"left", "right"} {
		dir := "libs/" + lib
		id := "can.project.lineage/shop_" + lib
		manifest, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(dir), "can.project.json"))
		if err != nil {
			t.Fatal(err)
		}
		source, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(dir), "src", "model.can"))
		if err != nil {
			t.Fatal(err)
		}
		registryData, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(dir), "can.errors.json"))
		if err != nil {
			t.Fatal(err)
		}
		registry, err := project.ParseRegistry(registryData)
		if err != nil {
			t.Fatal(err)
		}
		digest, err := project.SourceDigest([]project.SourceBytes{{Path: "model.can", Bytes: source}})
		if err != nil {
			t.Fatal(err)
		}
		fixtures, err := project.FixtureDigest(nil)
		if err != nil {
			t.Fatal(err)
		}
		edges[lib] = map[string]any{"target": id, "path": dir}
		entries[id] = map[string]any{"lineage": "shop_" + lib, "manifest_sha256": project.Digest(manifest), "source_sha256": digest, "fixtures_sha256": fixtures, "error_registry": registry, "edges": map[string]any{}}
	}
	data, err := json.Marshal(map[string]any{"edges": edges, "projects": entries})
	if err != nil {
		t.Fatal(err)
	}
	write("can.lock.json", string(data))
}

func inspectInstanceProject(t *testing.T, root string) projectReport {
	t.Helper()
	var out, diagnostics bytes.Buffer
	if code := runInspectProject(&out, &diagnostics, []string{root}); code != 0 {
		t.Fatalf("%d: %s", code, &diagnostics)
	}
	if strings.Contains(out.String(), root) {
		t.Fatal("report leaks machine path into identity")
	}
	var report projectReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	return report
}

func TestInspectProjectComposesTwinModelInstances(t *testing.T) {
	body := "record holder\n    first::item one\n    second::item two\n"
	var previous []byte
	for i := 0; i < 2; i++ {
		root := t.TempDir()
		writeInstanceProject(t, root, "left::model as first, right::model as second", body)
		var out, diagnostics bytes.Buffer
		if code := runInspectProject(&out, &diagnostics, []string{root}); code != 0 {
			t.Fatalf("%d: %s", code, &diagnostics)
		}
		if strings.Contains(out.String(), root) {
			t.Fatal("report leaks machine path into identity")
		}
		var report projectReport
		if err := json.Unmarshal(out.Bytes(), &report); err != nil {
			t.Fatal(err)
		}
		if report.SchemaVersion != 2 || report.Kind != "can.package-resolution" || len(report.Projects) != 3 || len(report.Packages) != 3 {
			t.Fatalf("%+v", report)
		}
		packages := map[string]projectPackageReport{}
		for _, pkg := range report.Packages {
			packages[pkg.ID] = pkg
		}
		left, right := packages["can.project.lineage/shop_left/model"], packages["can.project.lineage/shop_right/model"]
		if left.Name != "model" || right.Name != "model" || left.OutputDirectory == right.OutputDirectory {
			t.Fatal("twin model instances did not compose with distinct identities")
		}
		symbols := map[string]string{}
		for _, pkg := range report.Packages {
			for _, symbol := range pkg.Symbols {
				symbols[symbol.ID] = string(symbol.Kind)
			}
		}
		if symbols["can.project.lineage/shop_left/model::failed"] != "error" || symbols["can.project.lineage/shop_right/model::failed"] != "error" {
			t.Fatal("overlapping error kinds did not compose with distinct identities")
		}
		app := packages["can.project.root/app"]
		if app.Sources[0].Imports["first"] != "can.project.lineage/shop_left/model" || app.Sources[0].Imports["second"] != "can.project.lineage/shop_right/model" {
			t.Fatalf("file imports: %+v", app.Sources[0].Imports)
		}
		if i == 0 {
			previous = append([]byte(nil), out.Bytes()...)
		} else if !bytes.Equal(previous, out.Bytes()) {
			t.Fatal("relocation changed the instance report")
		}
	}
}

func TestInspectProjectAliasEditPreservesIdentities(t *testing.T) {
	firstBody := "record holder\n    first::item one\n    second::item two\n"
	secondBody := "record holder\n    alpha::item one\n    beta::item two\n"
	root := t.TempDir()
	writeInstanceProject(t, root, "left::model as first, right::model as second", firstBody)
	before := inspectInstanceProject(t, root)
	writeInstanceProject(t, root, "left::model as alpha, right::model as beta", secondBody)
	after := inspectInstanceProject(t, root)
	identities := func(report projectReport) map[string]string {
		out := map[string]string{}
		for _, pkg := range report.Packages {
			out[pkg.ID] = pkg.OutputDirectory
			for _, symbol := range pkg.Symbols {
				out[symbol.ID] = string(symbol.Kind)
			}
			for _, source := range pkg.Sources {
				out[source.ID] = source.OutputPath
			}
		}
		return out
	}
	earlier, later := identities(before), identities(after)
	if len(earlier) != len(later) {
		t.Fatalf("alias edit changed the identity set: %d vs %d", len(earlier), len(later))
	}
	for id, value := range earlier {
		if later[id] != value {
			t.Fatalf("alias edit moved %s", id)
		}
	}
}
