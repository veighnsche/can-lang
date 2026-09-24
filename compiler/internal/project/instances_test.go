package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func lineageFixture(t *testing.T, root string) {
	t.Helper()
	writeFixture(t, root, "can.project.json", `{"source_root":"src","project":"shop_root","dependencies":{"left":"libs/left","right":"libs/right"},"error_registry":"can.errors.json"}`)
	writeFixture(t, root, "can.errors.json", `{"active":[],"retired":[]}`)
	writeFixture(t, root, "src/main.can", "package app\n    provides []\n    uses [left::model as first, right::model as second]\nrecord holder\n    first::item one\n    second::item two\n")
	for _, lib := range []string{"left", "right"} {
		writeFixture(t, root, "libs/"+lib+"/can.project.json", `{"source_root":"src","project":"shop_`+lib+`","error_registry":"can.errors.json"}`)
	}
	writeFixture(t, root, "libs/left/can.errors.json", `{"active":[{"id":1000000,"kind":"model::failed"}],"retired":[]}`)
	writeFixture(t, root, "libs/right/can.errors.json", `{"active":[{"id":1000001,"kind":"model::failed"}],"retired":[]}`)
	writeFixture(t, root, "libs/left/src/model.can", "package model\n    provides [item, failed]\n    uses []\nrecord item\n    int value\nerror 1000000 failed(str reason)\n")
	writeFixture(t, root, "libs/right/src/model.can", "package model\n    provides [item, failed]\n    uses []\nrecord item\n    str label\nerror 1000001 failed(str reason)\n")
	writeFixtureLock(t, root, map[string]string{"left": "libs/left", "right": "libs/right"})
}

func TestLineageInstancesComposeAcrossSamePackageNames(t *testing.T) {
	root := t.TempDir()
	lineageFixture(t, root)
	graph, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(graph.Projects) != 3 {
		t.Fatalf("projects: %v", graph.Projects)
	}
	left := graph.Packages["can.project.lineage/shop_left/model"]
	right := graph.Packages["can.project.lineage/shop_right/model"]
	if left == nil || right == nil || left == right {
		t.Fatal("same-name model packages did not compose as distinct instances")
	}
	if left.Sources[0].OutputPath == right.Sources[0].OutputPath {
		t.Fatal("instance outputs collide")
	}
	if graph.Root.Lineage != "shop_root" || graph.Root.ID != "can.project.root" {
		t.Fatalf("root identity: %+v", graph.Root)
	}
}

func TestLineageIdentitySurvivesRelocationAndGrowth(t *testing.T) {
	root := t.TempDir()
	lineageFixture(t, root)
	graph, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	identities := map[string]string{}
	for _, p := range graph.Projects {
		for _, s := range p.Sources {
			if strings.Contains(s.ID, root) {
				t.Fatalf("identity leaks machine path: %s", s.ID)
			}
			identities[s.ID] = s.OutputPath
		}
	}
	moved := t.TempDir()
	lineageFixture(t, moved)
	again, err := Load(moved)
	if err != nil {
		t.Fatal(err)
	}
	relocated := map[string]string{}
	for _, p := range again.Projects {
		for _, s := range p.Sources {
			relocated[s.ID] = s.OutputPath
		}
	}
	if !reflect.DeepEqual(identities, relocated) {
		t.Fatal("relocation changed instance identities or outputs")
	}
	// Unrelated graph growth plus a new package keep every existing identity.
	writeFixture(t, moved, "libs/spare/can.project.json", `{"source_root":"src","project":"shop_spare","error_registry":"can.errors.json"}`)
	writeFixture(t, moved, "libs/spare/can.errors.json", `{"active":[],"retired":[]}`)
	writeFixture(t, moved, "libs/spare/src/extra.can", sourceText("extra", ""))
	writeFixture(t, moved, "libs/left/src/second.can", "package model\n    provides [item, failed, bonus]\n    uses []\nrecord bonus\n")
	writeFixture(t, moved, "can.project.json", `{"source_root":"src","project":"shop_root","dependencies":{"left":"libs/left","right":"libs/right","spare":"libs/spare"},"error_registry":"can.errors.json"}`)
	writeFixtureLock(t, moved, map[string]string{"left": "libs/left", "right": "libs/right", "spare": "libs/spare"})
	grown, err := Load(moved)
	if err != nil {
		t.Fatal(err)
	}
	kept := map[string]string{}
	for _, p := range grown.Projects {
		for _, s := range p.Sources {
			kept[s.ID] = s.OutputPath
		}
	}
	for id, path := range identities {
		if kept[id] != path {
			t.Fatalf("growth changed %s", id)
		}
	}
}

func TestGraphInternsIdenticalPaths(t *testing.T) {
	root := t.TempDir()
	projectFixture(t, root)
	writeFixture(t, root, "can.project.json", `{"source_root":"src","dependencies":{"one":"vendor","two":"vendor"},"error_registry":"can.errors.json"}`)
	writeFixtureLock(t, root, map[string]string{"one": "vendor", "two": "vendor"})
	graph, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(graph.Projects) != 2 || graph.Root.Dependencies["one"] != graph.Root.Dependencies["two"] {
		t.Fatal("identical paths did not intern to one instance")
	}
	if len(graph.Lock.Projects) != 1 {
		t.Fatalf("lock pins %d instances", len(graph.Lock.Projects))
	}

	// Edge paths stay confined to the owning manifest directory, so one
	// real directory is reachable from one parent only; nested duplicate
	// edges must still intern under a lineage identity.
	nested := t.TempDir()
	writeFixture(t, nested, "can.project.json", `{"source_root":"src","dependencies":{"alpha":"deps/alpha"},"error_registry":"can.errors.json"}`)
	writeFixture(t, nested, "can.errors.json", `{"active":[],"retired":[]}`)
	writeFixture(t, nested, "src/main.can", sourceText("app", ""))
	writeFixture(t, nested, "deps/alpha/can.project.json", `{"source_root":"src","dependencies":{"one":"shared","two":"shared"},"error_registry":"can.errors.json"}`)
	writeFixture(t, nested, "deps/alpha/can.errors.json", `{"active":[],"retired":[]}`)
	writeFixture(t, nested, "deps/alpha/src/main.can", sourceText("alpha_pkg", ""))
	writeFixture(t, nested, "deps/alpha/shared/can.project.json", `{"source_root":"src","project":"shared_lib","error_registry":"can.errors.json"}`)
	writeFixture(t, nested, "deps/alpha/shared/can.errors.json", `{"active":[],"retired":[]}`)
	writeFixture(t, nested, "deps/alpha/shared/src/main.can", sourceText("shared_pkg", ""))
	writeFixtureLock(t, nested, map[string]string{"alpha": "deps/alpha"})
	graph, err = Load(nested)
	if err != nil {
		t.Fatal(err)
	}
	alpha := graph.Root.Dependencies["alpha"]
	if alpha.Dependencies["one"] != alpha.Dependencies["two"] || alpha.Dependencies["one"].ID != "can.project.lineage/shared_lib" || len(graph.Projects) != 3 {
		t.Fatal("nested duplicate edges did not intern to one lineage instance")
	}
}

func TestGraphRejectsDivergentSameLineage(t *testing.T) {
	siblings := t.TempDir()
	projectFixture(t, siblings)
	writeFixture(t, siblings, "can.project.json", `{"source_root":"src","dependencies":{"one":"first","two":"second"},"error_registry":"can.errors.json"}`)
	for _, dir := range []string{"first", "second"} {
		writeFixture(t, siblings, dir+"/can.project.json", `{"source_root":"src","project":"twin_lib","error_registry":"can.errors.json"}`)
		writeFixture(t, siblings, dir+"/can.errors.json", `{"active":[],"retired":[]}`)
		writeFixture(t, siblings, dir+"/src/main.can", sourceText("twin", ""))
	}
	if _, err := Load(siblings); err == nil || !strings.Contains(err.Error(), "divergent instances") {
		t.Fatalf("divergent same-lineage siblings admitted: %v", err)
	}

	nested := t.TempDir()
	projectFixture(t, nested)
	writeFixture(t, nested, "vendor/can.project.json", `{"source_root":"src","project":"inner_lib","dependencies":{"twin":"twin"},"error_registry":"can.errors.json"}`)
	writeFixture(t, nested, "vendor/twin/can.project.json", `{"source_root":"src","project":"inner_lib","error_registry":"can.errors.json"}`)
	writeFixture(t, nested, "vendor/twin/can.errors.json", `{"active":[],"retired":[]}`)
	writeFixture(t, nested, "vendor/twin/src/main.can", sourceText("twin", ""))
	if _, err := Load(nested); err == nil || !strings.Contains(err.Error(), "divergent instances") {
		t.Fatalf("nested divergent lineage admitted: %v", err)
	}

	claimed := t.TempDir()
	projectFixture(t, claimed)
	writeFixture(t, claimed, "can.project.json", `{"source_root":"src","project":"shop_root","dependencies":{"vendor":"vendor"},"error_registry":"can.errors.json"}`)
	writeFixture(t, claimed, "vendor/can.project.json", `{"source_root":"src","project":"shop_root","error_registry":"can.errors.json"}`)
	if _, err := Load(claimed); err == nil || !strings.Contains(err.Error(), "divergent instances") {
		t.Fatalf("root/dependency lineage collision admitted: %v", err)
	}
}

func rewriteLock(t *testing.T, root string, edit func(lock map[string]any)) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, "can.lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	var lock map[string]any
	if err := json.Unmarshal(data, &lock); err != nil {
		t.Fatal(err)
	}
	edit(lock)
	data, err = json.Marshal(lock)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, "can.lock.json", string(data))
}

func TestGraphLockPinsContentAndDirectEdges(t *testing.T) {
	setup := func(t *testing.T) string {
		root := t.TempDir()
		lineageFixture(t, root)
		return root
	}
	t.Run("retarget", func(t *testing.T) {
		root := setup(t)
		rewriteLock(t, root, func(lock map[string]any) {
			edges := lock["edges"].(map[string]any)
			left := edges["left"].(map[string]any)
			left["target"] = "can.project.lineage/shop_right"
		})
		if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "target mismatch") {
			t.Fatalf("retargeted edge admitted: %v", err)
		}
	})
	t.Run("repath", func(t *testing.T) {
		root := setup(t)
		rewriteLock(t, root, func(lock map[string]any) {
			edges := lock["edges"].(map[string]any)
			left := edges["left"].(map[string]any)
			left["path"] = "libs/right"
		})
		if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "path mismatch") {
			t.Fatalf("repathed edge admitted: %v", err)
		}
	})
	t.Run("edge", func(t *testing.T) {
		root := setup(t)
		rewriteLock(t, root, func(lock map[string]any) {
			edges := lock["edges"].(map[string]any)
			delete(edges, "right")
		})
		if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "edges do not match") {
			t.Fatalf("dropped edge admitted: %v", err)
		}
	})
	t.Run("entry", func(t *testing.T) {
		root := setup(t)
		rewriteLock(t, root, func(lock map[string]any) {
			projects := lock["projects"].(map[string]any)
			delete(projects, "can.project.lineage/shop_right")
		})
		if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "missing dependency lock entry") {
			t.Fatalf("dropped entry admitted: %v", err)
		}
	})
	t.Run("lineage", func(t *testing.T) {
		root := setup(t)
		// Rewriting the manifest lineage without relocking breaks the
		// pinned node identity, not just a digest.
		writeFixture(t, root, "libs/left/can.project.json", `{"source_root":"src","project":"shop_renamed","error_registry":"can.errors.json"}`)
		if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "target mismatch") {
			t.Fatalf("renamed lineage admitted: %v", err)
		}
	})
	t.Run("transitive", func(t *testing.T) {
		root := setup(t)
		writeFixture(t, root, "libs/left/can.project.json", `{"source_root":"src","project":"shop_left","dependencies":{"nested":"nested"},"error_registry":"can.errors.json"}`)
		writeFixture(t, root, "libs/left/nested/can.project.json", `{"source_root":"src","error_registry":"can.errors.json"}`)
		writeFixture(t, root, "libs/left/nested/can.errors.json", `{"active":[],"retired":[]}`)
		writeFixture(t, root, "libs/left/nested/src/main.can", sourceText("nested", ""))
		if _, err := Load(root); err == nil {
			t.Fatal("unpinned transitive growth admitted")
		}
		writeFixtureLock(t, root, map[string]string{"left": "libs/left", "right": "libs/right"})
		graph, err := Load(root)
		if err != nil {
			t.Fatal(err)
		}
		if graph.Root.Dependencies["left"].Dependencies["nested"].ID != "can.project.dependency/left/nested" {
			t.Fatal("transitive legacy identity wrong")
		}
	})
}
