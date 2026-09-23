package catalogue

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCompleteInventoryAndMirrors(t *testing.T) {
	c := Builtin()
	inv := c.Inventory()
	if len(inv.Packages) != 26 || len(inv.Types) != 41 || len(inv.Errors) != 72 || len(inv.Operations) != 173 || len(inv.NativeDeclarations) != 10 {
		t.Fatalf("inventory coverage changed: packages=%d types=%d errors=%d operations=%d modes=%d", len(inv.Packages), len(inv.Types), len(inv.Errors), len(inv.Operations), len(inv.NativeDeclarations))
	}
	if !reflect.DeepEqual(inv.StandardFailures, []string{"arithmetic", "bounds", "resource_state", "assertion", "native_exception", "cleanup"}) {
		t.Fatal("standard failure inventory differs from C9")
	}
	if SourceHash() != GeneratedSourceSHA256 || inv.Revision != GeneratedRevision || inv.TargetID != GeneratedTargetID {
		t.Fatal("Go mirror is stale")
	}
	if err := Generate("../../..", true); err != nil {
		t.Fatal(err)
	}
	for _, p := range strings.Fields("ai asset bytes checks cli clock codec collections crypto env files html htmx http io json llm log number option path process random sql stream text") {
		if err := c.CheckProjectPackage(p); err == nil {
			t.Errorf("allowed project catalogue owner %s", p)
		}
	}
	if err := c.CheckProjectPackage("billing"); err != nil {
		t.Fatal(err)
	}
	// Every descriptor must resolve with concrete admitted types and exact callback
	// shapes. Semantic fixture tests below independently assert nontrivial bounds.
	for _, op := range inv.Operations {
		t.Run(op.Name, func(t *testing.T) {
			args := map[string]string{}
			for _, p := range op.Parameters {
				switch p.Constraint {
				case "form", "sql_parameters", "sql_row":
					args[p.Name] = "http::header"
				case "failure_variant":
					args[p.Name] = "option::value<str>"
				default:
					args[p.Name] = "str"
				}
			}
			callbacks := map[string]CallbackContract{}
			for _, cb := range op.Callbacks {
				actual := CallbackContract{Result: substitute(cb.Result, args)}
				for _, inp := range cb.Inputs {
					actual.Inputs = append(actual.Inputs, substitute(inp, args))
				}
				callbacks[cb.Name] = actual
			}
			spec, err := c.Resolve(op.Name, inv.TargetID, inv.Revision, args, callbacks, nil)
			if err != nil {
				t.Fatal(err)
			}
			ids := map[int]bool{}
			for _, e := range spec.Emits {
				ids[e.ID] = true
			}
			for _, name := range op.Emits {
				d, _ := c.Error(name)
				if !ids[d.ID] {
					t.Fatalf("lost declared error %s", name)
				}
			}
			if len(spec.Emits) != len(op.Emits) {
				t.Fatalf("unexpected hidden domain bound: %+v", spec.Emits)
			}
		})
	}
}

func TestRejectMalformedInventory(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Inventory)
	}{
		{"duplicate-error-id", func(i *Inventory) { i.Errors[1].ID = i.Errors[0].ID }},
		{"duplicate-name", func(i *Inventory) { i.Operations = append(i.Operations, i.Operations[0]) }},
		{"unallocated-error-id", func(i *Inventory) { i.Errors[1].ID = 1099 }},
		{"unknown-error-bound", func(i *Inventory) { i.Operations[0].Emits = []string{"number::not_allocated"} }},
		{"standard-in-domain-bound", func(i *Inventory) { i.Operations[0].Emits = []string{"standard_failure"} }},
		{"constructible-opaque", func(i *Inventory) {
			for n := range i.Types {
				if i.Types[n].Kind == "opaque" {
					i.Types[n].Constructible = true
					return
				}
			}
		}},
		{"opaque-fields", func(i *Inventory) {
			for n := range i.Types {
				if i.Types[n].Kind == "opaque" {
					i.Types[n].Fields = []Field{{Name: "native", Type: "str"}}
					return
				}
			}
		}},
		{"unknown-type", func(i *Inventory) { i.Operations[0].Result = "any" }},
		{"wrong-generic-arity", func(i *Inventory) { i.Operations[0].Result = "option::value" }},
		{"void-data", func(i *Inventory) { i.Types[0].Fields[0].Type = "void[]" }},
		{"protocol-registration", func(i *Inventory) { i.NativeDeclarations[0].Name = "user_protocol" }},
		{"missing-native-recipe", func(i *Inventory) { i.Operations[0].Lowering.Native = nil }},
		{"dropped-callback-errors", func(i *Inventory) {
			for n := range i.Operations {
				if len(i.Operations[n].CallbackErrors) > 0 {
					i.Operations[n].CallbackErrors = nil
					return
				}
			}
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			inv := Builtin().Inventory()
			tc.mutate(&inv)
			raw, err := json.Marshal(inv)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := load(raw); err == nil {
				t.Fatal("accepted invalid catalogue")
			}
		})
	}
	for _, raw := range [][]byte{append([]byte(`{"schemaVersion":1,`), source[1:]...), append([]byte(`{"kernels":[],`), source[1:]...), append([]byte(`{"protocols":[],`), source[1:]...)} {
		if _, err := load(raw); err == nil {
			t.Fatal("accepted duplicate/unknown JSON field")
		}
	}
}

func TestClosedLookupAndOpacity(t *testing.T) {
	c := Builtin()
	for _, name := range []string{"bytes::buffer", "collections::map", "html::safe", "http::request", "sql::transaction", "standard_failure", "option::value"} {
		if c.CheckConstructor(name) == nil {
			t.Errorf("public constructor for %s", name)
		}
	}
	for _, name := range []string{"choice_option", "option::some", "http::header", "sql::commit", "http::invalid_request"} {
		if err := c.CheckConstructor(name); err != nil {
			t.Error(err)
		}
	}
	for _, name := range []string{"protocol::register", "kernel::register", "html::raw", "json::parse", "sql::unsafe", "array.append"} {
		if _, err := c.Operation(name, GeneratedTargetID, GeneratedRevision); err == nil {
			t.Errorf("admitted %s", name)
		}
	}
	if _, err := c.Operation("append", "linux", 1); err == nil {
		t.Fatal("accepted unsupported target")
	}
	if _, err := c.Operation("append", GeneratedTargetID, 2); err == nil {
		t.Fatal("accepted unknown revision")
	}
	changed := c.Inventory()
	changed.Operations[0].Emits = []string{"http::timeout"}
	op, err := c.Operation("text::from_int", GeneratedTargetID, 1)
	if err != nil || len(op.Emits) != 0 {
		t.Fatal("caller mutated closed catalogue")
	}
}
func errorNames(es []ErrorIdentity) []string {
	out := []string{}
	for _, e := range es {
		out = append(out, e.Name)
	}
	return out
}
func TestCallbackAndStaticBounds(t *testing.T) {
	c := Builtin()
	e, _ := c.ErrorIdentity("http::timeout", nil)
	spec, err := c.Resolve("array.map", GeneratedTargetID, 1, map[string]string{"T": "int", "U": "str"}, map[string]CallbackContract{"callback": {Inputs: []string{"int"}, Result: "str", Emits: []ErrorIdentity{e, e}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Operation.Receiver != "int[]" || spec.Operation.Result != "str[]" || !reflect.DeepEqual(errorNames(spec.Emits), []string{"http::timeout"}) {
		t.Fatalf("wrong specialization: %+v", spec)
	}
	projectError := ErrorIdentity{Name: "billing::declined", Identity: "project.billing::declined", ID: 1000000, TypeArguments: []string{}}
	spec, err = c.Resolve("array.find", GeneratedTargetID, 1, map[string]string{"T": "int"}, map[string]CallbackContract{"callback": {Inputs: []string{"int"}, Result: "bool", Emits: []ErrorIdentity{projectError}}}, nil)
	if err != nil || spec.Operation.Result != "option::value<int>" || !reflect.DeepEqual(errorNames(spec.Emits), []string{"billing::declined"}) {
		t.Fatalf("project callback bound: %+v %v", spec, err)
	}
	for _, bad := range []CallbackContract{{Inputs: []string{"str"}, Result: "str"}, {Inputs: []string{"int"}, Result: "float"}, {Inputs: []string{"int"}, Result: "str", Emits: []ErrorIdentity{{Name: "fake::error", Identity: "fake", ID: 1099}}}} {
		if _, err := c.Resolve("array.map", GeneratedTargetID, 1, map[string]string{"T": "int", "U": "str"}, map[string]CallbackContract{"callback": bad}, nil); err == nil {
			t.Fatal("accepted invalid callback")
		}
	}
	if _, err := c.Resolve("http::route_get", GeneratedTargetID, 1, nil, map[string]CallbackContract{"callback": {Inputs: []string{"http::request"}, Result: "http::server_response", Emits: []ErrorIdentity{e}}}, nil); err == nil {
		t.Fatal("accepted mounted callback domain errors")
	}
	for _, name := range []string{"sql::query_one", "sql::transaction_query_one"} {
		spec, err := c.Resolve(name, GeneratedTargetID, 1, map[string]string{"P": "http::header", "R": "http::header"}, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		hasConnection := contains(errorNames(spec.Emits), "sql::connection_failed")
		if hasConnection != (name == "sql::query_one") {
			t.Fatal("wrong transaction domain contract")
		}
	}
}
func TestNativeModeBounds(t *testing.T) {
	c := Builtin()
	cases := []struct {
		mode  string
		flags map[string]bool
		want  []string
	}{
		{"noul", nil, []string{"ai::invalid_question", "ai::invalid_answer"}},
		{"fetch_envelope", nil, []string{"http::request_failed"}},
		{"fetch_body", nil, []string{"http::request_failed"}},
		{"judge", nil, []string{"http::request_failed", "ai::invalid_question", "ai::invalid_answer"}},
		{"llm", map[string]bool{"authenticated": true}, []string{"http::invalid_request", "http::credentials_missing", "http::transport_failed", "http::timeout", "http::body_limit", "http::status_error", "codec::invalid_data", "llm::refused", "llm::truncated", "llm::invalid_response"}},
	}
	for _, tc := range cases {
		got, err := c.RequiredNativeBound(tc.mode, tc.flags, nil)
		if err != nil || !reflect.DeepEqual(errorNames(got), tc.want) {
			t.Fatalf("%s: %+v %v", tc.mode, got, err)
		}
	}
	q, _ := c.ErrorIdentity("text::empty_pattern", nil)
	got, err := c.RequiredNativeBound("judge", nil, map[string][]ErrorIdentity{"questions": {q}})
	if err != nil || !contains(errorNames(got), q.Name) {
		t.Fatal("lost declared question bound")
	}
	if _, err := c.RequiredNativeBound("plugin", nil, nil); err == nil {
		t.Fatal("registered protocol")
	}
	if _, err := c.RequiredNativeBound("fetch_body", map[string]bool{"pretend_success": true}, nil); err == nil {
		t.Fatal("unknown mode condition")
	}
}
func TestGenericConstraints(t *testing.T) {
	c := Builtin()
	for _, row := range []struct {
		value, constraint string
		want              bool
	}{
		{"int", "map_key", true}, {"float", "map_key", false}, {"http::header", "map_key", false},
		{"float", "sort_key", true}, {"void", "data", false}, {"void[]", "data", false},
		{"bytes::buffer", "data", true}, {"bytes::buffer", "wire", false}, {"standard_failure", "wire", false},
		{"http::response<int>", "wire", true}, {"http::response<bytes::buffer>", "wire", false},
		{"http::header", "form", true}, {"number::division", "form", false},
		{"option::value<str>", "failure_variant", true}, {"str", "failure_variant", false},
	} {
		if got := c.admits(row.value, row.constraint, nil); got != row.want {
			t.Errorf("%s as %s: %v", row.value, row.constraint, got)
		}
	}
	if _, err := c.Resolve("collections::empty_map", GeneratedTargetID, 1, map[string]string{"K": "float", "V": "str"}, nil, nil); err == nil {
		t.Fatal("accepted float key")
	}
	if _, err := c.Resolve("codec::encode_json", GeneratedTargetID, 1, map[string]string{"T": "html::safe"}, nil, func(string, string) bool { return true }); err == nil {
		t.Fatal("project proof bypassed catalogue opacity")
	}
}
func TestGenerationDetectsDrift(t *testing.T) {
	root := t.TempDir()
	if err := Generate(root, false); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "runtime/catalogue.ts")
	if err := os.WriteFile(path, []byte("stale"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := Generate(root, true); err == nil {
		t.Fatal("stale TS mirror admitted")
	}
}

func TestCallbackUnionRejectsConflictingIDs(t *testing.T) {
	c := Builtin()
	left := ErrorIdentity{Name: "billing::declined", Identity: "project.billing::declined", ID: 1000000, TypeArguments: []string{}}
	right := ErrorIdentity{Name: "shipping::missing", Identity: "project.shipping::missing", ID: 1000000, TypeArguments: []string{}}
	if _, err := c.union([]ErrorIdentity{left, right}); err == nil {
		t.Fatal("accepted duplicate ID across callback kinds")
	}
}

func TestRequestFailedContract(t *testing.T) {
	c := Builtin()
	failed, ok := c.Error("http::request_failed")
	if !ok || failed.ID != 1106 || failed.Identity != "can.std.http@1::request_failed" {
		t.Fatalf("http::request_failed allocation lost: %+v %v", failed, ok)
	}
	if len(failed.Fields) != 1 || failed.Fields[0].Name != "detail" || failed.Fields[0].Type != "http::failure_detail" {
		t.Fatalf("http::request_failed payload differs from A2.4: %+v", failed.Fields)
	}
	detail, ok := c.Type("http::failure_detail")
	if !ok || detail.Kind != "variant" || detail.Identity != "can.std.http@1::failure_detail" || len(detail.Parameters) != 0 {
		t.Fatalf("http::failure_detail declaration lost: %+v %v", detail, ok)
	}
	want := []string{"http::invalid_request", "http::credentials_missing", "http::transport_failed", "http::timeout", "http::body_limit", "http::status_error", "codec::invalid_data"}
	if !reflect.DeepEqual(detail.Leaves, want) {
		t.Fatalf("http::failure_detail leaves differ from A2.4: %v", detail.Leaves)
	}
	for _, leaf := range want {
		if _, ok := c.Error(leaf); !ok {
			t.Fatalf("detail leaf %s is not an allocated error", leaf)
		}
	}
}

func TestChecksRequireContract(t *testing.T) {
	c := Builtin()
	failed, ok := c.Error("checks::failed")
	if !ok || failed.ID != 1010 || failed.Identity != "can.std.checks@1::failed" {
		t.Fatalf("checks::failed allocation lost: %+v %v", failed, ok)
	}
	if len(failed.Fields) != 1 || failed.Fields[0].Name != "reason" || failed.Fields[0].Type != "str" {
		t.Fatalf("checks::failed payload differs from C9.2: %+v", failed.Fields)
	}
	op, err := c.Operation("checks::require", c.Inventory().TargetID, c.Inventory().Revision)
	if err != nil {
		t.Fatal(err)
	}
	if op.Identity != "can.std.checks@1::require" || op.Kind != "function" || op.Result != "void" {
		t.Fatalf("checks::require descriptor differs from C9.2: %+v", op)
	}
	if len(op.Inputs) != 2 || op.Inputs[0].Name != "condition" || op.Inputs[0].Type != "bool" || op.Inputs[1].Name != "reason" || op.Inputs[1].Type != "str" {
		t.Fatalf("checks::require inputs differ from C9.2: %+v", op.Inputs)
	}
	if len(op.Emits) != 1 || op.Emits[0] != "checks::failed" {
		t.Fatalf("checks::require bound differs from C9.2: %+v", op.Emits)
	}
	if op.Assertion != "real" || len(op.Refs) != 1 || op.Refs[0] != "C9.2" || op.Lowering.Task != "LF08" || len(op.Lowering.Native) == 0 {
		t.Fatalf("checks::require evidence contract differs from C9.2: %+v", op)
	}
}
