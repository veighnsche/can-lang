package project

import (
	"strings"
	"testing"
)

// TestFormerNumericCollisionComposesWithQualifiedIdentity covers the DI-05b
// composition case: two libraries exposing the same error kind used to need
// distinct hand-allocated numeric IDs. Identical unnumbered registries now
// compose, and each library's reports keep a distinct qualified identity.
func TestFormerNumericCollisionComposesWithQualifiedIdentity(t *testing.T) {
	root := t.TempDir()
	lineageFixture(t, root)
	graph, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	left, kind, ok := graph.ResolveErrorReport("can.error.v2:can.project.lineage/shop_left/model::failed")
	if !ok || left == nil || left.Lineage != "shop_left" || kind != "model::failed" {
		t.Fatalf("left report misattributed: %+v %q %v", left, kind, ok)
	}
	right, kind, ok := graph.ResolveErrorReport("can.error.v2:can.project.lineage/shop_right/model::failed")
	if !ok || right == nil || right.Lineage != "shop_right" || kind != "model::failed" {
		t.Fatalf("right report misattributed: %+v %q %v", right, kind, ok)
	}
	if left == right {
		t.Fatal("same-kind errors from two libraries share one owner")
	}
}

func TestRegistryResolvesSuppliedPredecessorChain(t *testing.T) {
	registry, err := ParseRegistry([]byte(`{"active":["app::renamed"],"retired":["app::legacy","app::older"],"predecessors":{"app::renamed":["app::legacy","app::older"]}}`))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"app::renamed", "app::legacy", "app::older"} {
		current, ok := registry.ResolveKind(name)
		if !ok || current != "app::renamed" {
			t.Fatalf("chain member %q resolves to %q, %v", name, current, ok)
		}
	}
	if _, ok := registry.ResolveKind("app::unknown"); ok {
		t.Fatal("unknown name resolved")
	}
	// One retired name chaining to two kinds would make archived reports
	// ambiguous.
	ambiguous := `{"active":["app::new","app::renamed"],"retired":["app::legacy"],"predecessors":{"app::new":["app::legacy"],"app::renamed":["app::legacy"]}}`
	if _, err := ParseRegistry([]byte(ambiguous)); err == nil || !strings.Contains(err.Error(), "chains to both") {
		t.Fatalf("ambiguous predecessor chain admitted: %v", err)
	}
}

// TestArchivedTwoBuildReportsAttributeAcrossRename archives one terminal
// report from a pre-rename build and one from a post-rename build, then
// attributes both to the current declaration through the supplied chain.
func TestArchivedTwoBuildReportsAttributeAcrossRename(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "can.project.json", `{"source_root":"src","error_registry":"can.errors.json"}`)
	writeFixture(t, root, "can.errors.json", `{"active":["app::renamed"],"retired":["app::legacy"],"predecessors":{"app::renamed":["app::legacy"]}}`)
	writeFixture(t, root, "src/main.can", "package app\n    provides [renamed]\n    uses []\nerror renamed(str reason)\n")
	graph, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	preRename := "can.error.v2:can.project.root/app::legacy"
	postRename := "can.error.v2:can.project.root/app::renamed"
	var owner *Project
	for _, archived := range []string{preRename, postRename} {
		resolved, kind, ok := graph.ResolveErrorReport(archived)
		if !ok || resolved != graph.Root || kind != "app::renamed" {
			t.Fatalf("archived report %q attributed to %+v %q, %v", archived, resolved, kind, ok)
		}
		owner = resolved
	}
	if owner != graph.Root {
		t.Fatal("rename changed the owning project")
	}
	resolved, kind, ok := graph.ResolveErrorReport("can.error.v2:can.std.codec@1::invalid_data")
	if !ok || resolved != nil || kind != "codec::invalid_data" {
		t.Fatalf("catalogue report misattributed: %+v %q %v", resolved, kind, ok)
	}
	for _, bad := range []string{
		"can.error.v2:can.project.root/app::unknown",
		"can.error.v2:can.project.root/app::renamed<int>",
		"can.error.v1:can.project.root/app::renamed",
		"can.project.root/app::renamed",
		"",
	} {
		if _, _, ok := graph.ResolveErrorReport(bad); ok {
			t.Fatalf("unresolvable report identity admitted: %q", bad)
		}
	}
}
