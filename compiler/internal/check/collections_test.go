package check

import (
	"os"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/catalogue"
)

func TestCollectionsCatalogue(t *testing.T) {
	source, err := os.ReadFile("../../../std/map/current/src/main.can")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = programFixture(t, map[string]string{"src/main.can": string(source)}); err != nil {
		t.Fatal(err)
	}
}

// A05: factory selection covers the bulk builders, which take no
// collection input: the result type selects the factory. The synthetic
// operations mirror the catalogue patch handed to lane E.
func TestCollectionKindSelectsBulkBuilders(t *testing.T) {
	buildMap := &catalogue.Operation{
		Name:     "collections::build_map",
		Identity: "can.std.collections@1::build_map",
		Inputs:   []catalogue.Field{{Name: "entries", Type: "collections::entry<K,V>[]"}},
		Result:   "collections::map<K,V>",
	}
	buildSet := &catalogue.Operation{
		Name:     "collections::build_set",
		Identity: "can.std.collections@1::build_set",
		Inputs:   []catalogue.Field{{Name: "values", Type: "K[]"}},
		Result:   "collections::set<K>",
	}
	if kind := collectionKind(buildMap); kind != "collections::map<K,V>" {
		t.Fatalf("build_map selected %s", kind)
	}
	if kind := collectionKind(buildSet); kind != "collections::set<K>" {
		t.Fatalf("build_set selected %s", kind)
	}
	for name, want := range map[string]string{
		"can.std.collections@1::empty_map": "collections::map<K,V>",
		"can.std.collections@1::empty_set": "collections::set<K>",
		"can.std.collections@1::insert":    "collections::map<K,V>",
		"can.std.collections@1::entries":   "collections::map<K,V>",
		"can.std.collections@1::add":       "collections::set<K>",
		"can.std.collections@1::union":     "collections::set<K>",
	} {
		op := collectionOperation(name)
		if op == nil {
			t.Fatalf("missing catalogue operation %s", name)
		}
		if kind := collectionKind(op); kind != want {
			t.Fatalf("%s selected %s, want %s", name, kind, want)
		}
	}
}

func TestCollectionsRejectInvalidContracts(t *testing.T) {
	source, err := os.ReadFile("../../../std/map/current/src/main.can")
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range [][2]string{
		{"collections::empty_set<int>()", "collections::empty_set<float>()"},
		{"collections::empty_map<int,str>()", "collections::empty_map<collections::entry<int,str>,str>()"},
		{"collections::empty_set<int>()", "collections::set<int>()"},
		{"collections::contains(first, 1)", "collections::contains(first, 1.0)"},
		{"callable collections::add\n", "callable collections::contains\n"},
		{"emits [collections::key_absent]\n    asserts", "emits []\n    asserts"},
		{"collections::add(empty, false)", "collections::add(empty, 1)"},
	} {
		t.Run(change[1], func(t *testing.T) {
			text := strings.Replace(string(source), change[0], change[1], 1)
			if text == string(source) {
				t.Fatal("mutation missed source")
			}
			if _, err := programFixture(t, map[string]string{"src/main.can": text}); err == nil {
				t.Fatal("invalid collection contract admitted")
			}
		})
	}
}
