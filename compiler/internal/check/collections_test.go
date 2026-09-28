package check

import (
	"os"
	"reflect"
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

// A07: the A05 bulk builders resolve from Can source with explicit type
// arguments and specialize to collection contracts carrying the declared
// result and failure types.
func TestBulkBuildersResolveFromSource(t *testing.T) {
	text := strings.Replace(programHeader, "uses []", "uses [collections]", 1) + `fn collections::entry<int,str>[] built
    emits [collections::key_exists]
    asserts
        sample: => ok [collections::entry<int,str>(1, "a")]
    collections::entry<int,str>[] rows = [collections::entry<int,str>(1, "a")]
    match call collections::build_map<int,str>(rows)
        collections::key_exists
        ok collections::map<int,str> made => ok call collections::entries(made)
fn bool grouped
    emits []
    asserts
        sample: => ok true
    match call collections::build_set<int>([3, 2, 3])
        ok collections::set<int> made => ok call collections::contains(made, 2)
` + programMain + "    ok\n"
	program, err := programFixture(t, map[string]string{"src/main.can": text})
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, special := range program.Collections {
		seen[special.Operation] = true
		switch special.Operation {
		case "can.std.collections@1::build_map":
			args := special.Collection.Arguments()
			if special.Collection.Declaration() != "can.std.collections@1::map" || len(args) != 2 || args[0].Declaration() != "int" || args[1].Declaration() != "str" {
				t.Fatalf("build_map collection: %s %+v", special.Collection.Declaration(), args)
			}
			if len(special.Contract.Errors()) != 1 || special.Contract.Errors()[0].Declaration() != "can.std.collections@1::key_exists" {
				t.Fatalf("build_map errors: %+v", special.Contract.Errors())
			}
		case "can.std.collections@1::build_set":
			args := special.Collection.Arguments()
			if special.Collection.Declaration() != "can.std.collections@1::set" || len(args) != 1 || args[0].Declaration() != "int" {
				t.Fatalf("build_set collection: %s %+v", special.Collection.Declaration(), args)
			}
			if len(special.Contract.Errors()) != 0 {
				t.Fatalf("build_set errors: %+v", special.Contract.Errors())
			}
		}
	}
	for _, want := range []string{"can.std.collections@1::build_map", "can.std.collections@1::build_set"} {
		if !seen[want] {
			t.Fatalf("missing specialization %s in %+v", want, program.Collections)
		}
	}
}

// TestCollectionOperationNamespaceInvariant pins the early-rejection guard:
// every I25/A05 operation must live in can.std.collections@1::, the guard
// must return metadata identical to the former full-scan oracle, and all
// other identities must decline without consulting task state.
func TestCollectionOperationNamespaceInvariant(t *testing.T) {
	const namespace = "can.std.collections@1::"
	inventory := catalogue.Builtin().Inventory()
	if len(inventory.Operations) == 0 {
		t.Fatal("embedded inventory has no operations")
	}
	admitted := 0
	for _, want := range inventory.Operations {
		want := want
		admissible := want.Lowering.Task == "I25" || want.Lowering.Task == "A05"
		got := collectionOperation(want.Identity)
		if !admissible {
			if got != nil {
				t.Fatalf("collectionOperation(%q) admitted task %q", want.Identity, want.Lowering.Task)
			}
			continue
		}
		admitted++
		if !strings.HasPrefix(want.Identity, namespace) {
			t.Fatalf("admitted %q violates namespace %q", want.Identity, namespace)
		}
		if got == nil {
			t.Fatalf("collectionOperation(%q) declined I25/A05 operation", want.Identity)
		}
		if !reflect.DeepEqual(*got, want) {
			t.Fatalf("collectionOperation(%q) metadata differs from inventory oracle", want.Identity)
		}
	}
	if admitted == 0 {
		t.Fatal("no I25/A05 operations in embedded inventory")
	}
	// Mutating nested fields must not affect later lookups: the positive
	// scan still serves deep defensive copies of the embedded inventory.
	first := ""
	var original catalogue.Operation
	for _, op := range inventory.Operations {
		if (op.Lowering.Task == "I25" || op.Lowering.Task == "A05") && len(op.Inputs) > 0 {
			first, original = op.Identity, op
			break
		}
	}
	if first == "" {
		t.Fatal("no I25/A05 operation with inputs for mutation isolation")
	}
	mutated := collectionOperation(first)
	if mutated == nil {
		t.Fatalf("collectionOperation(%q) declined during mutation check", first)
	}
	mutated.Result = "mutated::type"
	mutated.Inputs[0].Type = "mutated::nested"
	mutated.Inputs[0].Name = "mutated_nested"
	if len(mutated.Parameters) > 0 {
		mutated.Parameters[0].Constraint = "mutated"
	}
	again := collectionOperation(first)
	if again == nil {
		t.Fatalf("collectionOperation(%q) declined after mutation", first)
	}
	if !reflect.DeepEqual(*again, original) {
		t.Fatal("collectionOperation result aliases embedded inventory (nested mutation leaked)")
	}
	rejections := []string{
		"",
		"audit::f0",
		"main",
		"can.project.audit@1::f0",
		// Near-prefix impostors.
		"can.std.collections@1:empty_map",
		"can.std.collections@1",
		"can.std.collections@1::",
		"can.std.collections@10::empty_map",
		"can.std.collections@1X::empty_map",
		"can.std.collections@2::empty_map",
		"can.std.collections@1::empty_map ",
		" can.std.collections@1::empty_map",
		"CAN.STD.COLLECTIONS@1::EMPTY_MAP",
		"xcan.std.collections@1::empty_map",
		// Unknown names inside the namespace.
		"can.std.collections@1::does_not_exist",
		"can.std.collections@1::Empty_Map",
		// Real non-collection built-ins across several tasks.
		"can.std.text@1::from_int",
		"can.intrinsic.str@1::includes",
		"can.std.bytes@1::empty",
	}
	for _, identity := range rejections {
		if got := collectionOperation(identity); got != nil {
			t.Fatalf("collectionOperation(%q) should decline, got %+v", identity, got.Identity)
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
