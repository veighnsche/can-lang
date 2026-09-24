package resolve

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/project"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
)

// pinInstanceLock pins every non-root instance keyed by directory. IDs are
// supplied by the caller: lineage identities for declared lineages, first-
// discovery edge paths otherwise. Sources are single src/main.can files
// unless the fixture overrides them through pinInstanceLockSources.
func pinInstanceLock(t *testing.T, files map[string]string, ids map[string]string) {
	t.Helper()
	sources := map[string]string{}
	for dir := range ids {
		sources[dir] = dir + "/src/main.can"
	}
	pinInstanceLockSources(t, files, ids, map[string]string{}, sources)
}

func pinInstanceLockSources(t *testing.T, files map[string]string, ids, lineages, sources map[string]string) {
	t.Helper()
	nodeEdges := func(dir string) map[string]any {
		key := "can.project.json"
		if dir != "" {
			key = dir + "/can.project.json"
		}
		var shape struct {
			Dependencies map[string]string `json:"dependencies"`
		}
		if err := json.Unmarshal([]byte(files[key]), &shape); err != nil {
			t.Fatal(err)
		}
		out := map[string]any{}
		for name, rel := range shape.Dependencies {
			child := rel
			if dir != "" {
				child = dir + "/" + rel
			}
			out[name] = map[string]any{"target": ids[child], "path": rel}
		}
		return out
	}
	entries := map[string]any{}
	for dir, id := range ids {
		manifest := files[dir+"/can.project.json"]
		data := files[sources[dir]]
		var logical string
		if _, logical, _ = strings.Cut(sources[dir], dir+"/src/"); logical == "" {
			t.Fatalf("source %q is outside the %s source root", sources[dir], dir)
		}
		digest, err := project.SourceDigest([]project.SourceBytes{{Path: logical, Bytes: []byte(data)}})
		if err != nil {
			t.Fatal(err)
		}
		fixtures, err := project.FixtureDigest(nil)
		if err != nil {
			t.Fatal(err)
		}
		registry := map[string]any{"active": []any{}, "retired": []any{}}
		var decoded struct {
			Active  []any `json:"active"`
			Retired []any `json:"retired"`
		}
		if err := json.Unmarshal([]byte(files[dir+"/can.errors.json"]), &decoded); err != nil {
			t.Fatal(err)
		}
		registry["active"] = decoded.Active
		if decoded.Active == nil {
			registry["active"] = []any{}
		}
		registry["retired"] = decoded.Retired
		if decoded.Retired == nil {
			registry["retired"] = []any{}
		}
		entries[id] = map[string]any{
			"lineage": lineages[dir], "manifest_sha256": project.Digest([]byte(manifest)),
			"source_sha256": digest, "fixtures_sha256": fixtures,
			"error_registry": registry, "edges": nodeEdges(dir),
		}
	}
	lock, err := json.Marshal(map[string]any{"edges": nodeEdges(""), "projects": entries})
	if err != nil {
		t.Fatal(err)
	}
	files["can.lock.json"] = string(lock)
}

func twinModelFiles(uses, body string) map[string]string {
	return map[string]string{
		"can.project.json":            `{"source_root":"src","project":"shop_root","dependencies":{"left":"libs/left","right":"libs/right"},"error_registry":"can.errors.json"}`,
		"can.errors.json":             `{"active":[],"retired":[]}`,
		"src/main.can":                header("app", "", uses) + body,
		"libs/left/can.project.json":  `{"source_root":"src","project":"shop_left","error_registry":"can.errors.json"}`,
		"libs/left/can.errors.json":   `{"active":["model::failed"],"retired":[]}`,
		"libs/left/src/model.can":     "package model\n    provides [item, failed]\n    uses []\nrecord item\n    int value\nerror failed(str reason)\nrecord secret\n    int value\n",
		"libs/right/can.project.json": `{"source_root":"src","project":"shop_right","error_registry":"can.errors.json"}`,
		"libs/right/can.errors.json":  `{"active":["model::failed"],"retired":[]}`,
		"libs/right/src/model.can":    "package model\n    provides [item, failed]\n    uses []\nrecord item\n    str label\nerror failed(str reason)\n",
	}
}

func twinModelIDs() map[string]string {
	return map[string]string{"libs/left": "can.project.lineage/shop_left", "libs/right": "can.project.lineage/shop_right"}
}

func twinModelLineages() map[string]string {
	return map[string]string{"libs/left": "shop_left", "libs/right": "shop_right"}
}

func twinModelSources() map[string]string {
	return map[string]string{"libs/left": "libs/left/src/model.can", "libs/right": "libs/right/src/model.can"}
}

func TestQualifiedImportsComposeTwinModels(t *testing.T) {
	files := twinModelFiles("left::model as first, right::model as second",
		"record holder\n    first::item one\n    second::item two\nfn void run\n    emits [first::failed, second::failed]\n    asserts\n        sample: => ok\n    ok\n")
	pinInstanceLockSources(t, files, twinModelIDs(), twinModelLineages(), twinModelSources())
	w, err := buildFiles(t, files)
	if err != nil {
		t.Fatal(err)
	}
	first := w.Packages["can.project.lineage/shop_left/model"]
	second := w.Packages["can.project.lineage/shop_right/model"]
	if first == nil || second == nil || first == second {
		t.Fatal("twin model instances did not compose")
	}
	if first.Scope.Symbols["item"].ID != "can.project.lineage/shop_left/model::item" {
		t.Fatalf("left item: %s", first.Scope.Symbols["item"].ID)
	}
	if second.Scope.Symbols["failed"].ID != "can.project.lineage/shop_right/model::failed" {
		t.Fatalf("right failed: %s", second.Scope.Symbols["failed"].ID)
	}
	file := fileNamed(w, "app", "main.can")
	if file.Imports["first"] != first || file.Imports["second"] != second {
		t.Fatal("file aliases do not bind the twin instances")
	}
	got, err := file.Lookup(nil, syntax.QualifiedName{Package: "second", Name: "failed"}, ErrorUse)
	if err != nil || got != second.Scope.Symbols["failed"] {
		t.Fatalf("overlapping error lookup: %v", err)
	}
	// Renaming the local aliases keeps every canonical symbol identity.
	renamed := twinModelFiles("left::model as alpha, right::model as beta",
		"record holder\n    alpha::item one\n    beta::item two\nfn void run\n    emits [alpha::failed, beta::failed]\n    asserts\n        sample: => ok\n    ok\n")
	pinInstanceLockSources(t, renamed, twinModelIDs(), twinModelLineages(), twinModelSources())
	again, err := buildFiles(t, renamed)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"can.project.lineage/shop_left/model", "can.project.lineage/shop_right/model"} {
		before, after := w.Packages[id].Scope.Symbols, again.Packages[id].Scope.Symbols
		if len(before) != len(after) {
			t.Fatal("alias edit changed the symbol set")
		}
		for name, symbol := range before {
			if after[name] == nil || after[name].ID != symbol.ID {
				t.Fatalf("alias edit moved %s", symbol.ID)
			}
		}
	}
}

func TestQualifiedImportsRejectUndeclaredAndPrivate(t *testing.T) {
	base := "record holder\n    first::item one\n"
	cases := []struct {
		name, uses, body, message, span string
	}{
		{"unqualified", "model", base, `import it as left::"model"`, "model"},
		{"unknown-dependency", "middle::model as first", base, `unknown dependency "middle"`, "middle"},
		{"missing-package", "left::absent as first", base, `dependency "left" has no package "absent"`, "absent"},
		{"private", "left::model as first, right::model as second", "record holder\n    first::secret value\n", "first::secret is private", ""},
		{"ambiguous-default", "left::model, right::model", base, `import alias "model" collides`, "model"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files := twinModelFiles(tc.uses, tc.body)
			pinInstanceLockSources(t, files, twinModelIDs(), twinModelLineages(), twinModelSources())
			root := writeFiles(t, files)
			graph, err := project.Load(root)
			if err != nil {
				t.Fatal(err)
			}
			_, err = Build(graph)
			if err == nil || !strings.Contains(err.Error(), tc.message) {
				t.Fatalf("admitted or misdiagnosed: %v", err)
			}
			if tc.span == "" {
				return
			}
			located, ok := source.AsLocated(err)
			if !ok {
				t.Fatalf("unlocated diagnostic: %v", err)
			}
			data, err := os.ReadFile(filepath.Join(root, "src/main.can"))
			if err != nil {
				t.Fatal(err)
			}
			if got := string(data[located.Span.Start:located.Span.End]); got != tc.span {
				t.Fatalf("diagnostic covers %q, expected %q", got, tc.span)
			}
		})
	}
}
