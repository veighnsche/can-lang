package catalogue

import (
	"reflect"
	"testing"
)

// TestCIngressCatalogueContract pins the NT-I02 C-ingress seam surface:
// 2 pure owner-first operations (parse_module, check_module) over module
// source text, parsed-edge/manifest wire records, and inert witness
// facts/counters/observation records. The stateful K05 witness keeps NO
// catalogue operations by scope decision (Jev x3, evidence/I02-jev):
// witness authority stays host-side so only QN2 harness code drives it,
// and candidate Can rows can never mint witness credit for artifacts
// they chose. Both ops emit test::stale_handle only; malformed inputs
// fail fast in the adapter.
func TestCIngressCatalogueContract(t *testing.T) {
	c := Builtin()
	target, revision := c.Inventory().TargetID, c.Inventory().Revision
	operation := func(name string) Operation {
		t.Helper()
		op, err := c.Operation(name, target, revision)
		if err != nil {
			t.Fatal(err)
		}
		return op
	}
	if err := c.CheckProjectPackage("c"); err == nil {
		t.Fatal("c package is not distribution-owned")
	}
	for _, name := range []string{"c::instrument", "c::invoke_executable", "c::invoke_case", "c::observe", "c::counters"} {
		if _, err := c.Operation(name, target, revision); err == nil {
			t.Fatalf("%s exists: witness authority leaked into the catalogue", name)
		}
	}
	fieldsOf := func(name string) map[string]string {
		t.Helper()
		typ, ok := c.Type(name)
		if !ok {
			t.Fatalf("%s missing", name)
		}
		fields := map[string]string{}
		for _, field := range typ.Fields {
			fields[field.Name] = field.Type
		}
		return fields
	}
	edge := fieldsOf("c::parsed_edge")
	if edge["specifier"] != "str" || edge["type_only"] != "bool" || edge["names"] != "c::parsed_name[]" || edge["side_effect"] != "bool" {
		t.Fatalf("c::parsed_edge fields differ: %+v", edge)
	}
	module := fieldsOf("c::parsed_module")
	if module["imports"] != "c::parsed_edge[]" || module["exports"] != "str[]" || module["dynamic_import"] != "bool" || module["body"] != "str" {
		t.Fatalf("c::parsed_module fields differ: %+v", module)
	}
	fixed := fieldsOf("c::fixed_edge")
	if fixed["runtime_module"] != "option::value<str>" || fixed["resolved"] != "option::value<str>" || fixed["type_only"] != "bool" || fixed["names"] != "c::name_binding[]" {
		t.Fatalf("c::fixed_edge fields differ: %+v", fixed)
	}
	manifest := fieldsOf("c::manifest")
	for _, name := range []string{"local_prefix", "program_roots", "runtime_modules", "executable", "assertion_root", "assertion_case"} {
		if _, ok := manifest[name]; !ok {
			t.Fatalf("c::manifest lacks %s: %+v", name, manifest)
		}
	}
	if manifest["program_roots"] != "str[]" || manifest["runtime_modules"] != "str[]" || manifest["executable"] != "c::manifest_entry" || manifest["assertion_case"] != "c::case_requirements" {
		t.Fatalf("c::manifest fields differ: %+v", manifest)
	}
	facts := fieldsOf("c::artifact_facts")
	for _, name := range []string{"artifact", "adapter_export", "adapter_binding", "artifact_digest", "token", "seq"} {
		if _, ok := facts[name]; !ok {
			t.Fatalf("c::artifact_facts lacks %s: %+v", name, facts)
		}
	}
	if facts["seq"] != "int" {
		t.Fatalf("c::artifact_facts seq differs: %+v", facts)
	}
	obs := fieldsOf("c::case_observation")
	if obs["passed"] != "bool" || obs["reason"] != "option::value<str>" || obs["completion"] != "str" {
		t.Fatalf("c::case_observation fields differ: %+v", obs)
	}
	type descriptor struct {
		result string
		inputs []string
		emits  []string
	}
	want := map[string]descriptor{
		"c::parse_module": {"c::parsed_module", []string{"test::owner", "str"}, []string{"test::stale_handle"}},
		"c::check_module": {"str[]", []string{"test::owner", "c::manifest", "str", "str", "str"}, []string{"test::stale_handle"}},
	}
	if len(want) != 2 {
		t.Fatalf("contract pins %d operations, want 2", len(want))
	}
	for name, descriptor := range want {
		op := operation(name)
		if op.Result != descriptor.result || len(op.Inputs) != len(descriptor.inputs) {
			t.Fatalf("%s descriptor differs: %+v", name, op)
		}
		for i, input := range descriptor.inputs {
			if op.Inputs[i].Type != input {
				t.Fatalf("%s input %d differs: %+v", name, i, op.Inputs)
			}
		}
		if op.Inputs[0].Type != "test::owner" {
			t.Fatalf("%s is not owner-first: %+v", name, op.Inputs)
		}
		if !reflect.DeepEqual(op.Emits, descriptor.emits) {
			t.Fatalf("%s emits differ: %+v", name, op.Emits)
		}
		if op.Lowering.Task != "NT-I02" || !reflect.DeepEqual(op.Refs, []string{"NT-I02"}) || op.Assertion != "supplied" {
			t.Fatalf("%s traceability differs: task=%q refs=%v assertion=%q", name, op.Lowering.Task, op.Refs, op.Assertion)
		}
		if len(op.Lowering.Native) == 0 || op.Lowering.Adapter == "" {
			t.Fatalf("%s names no native recipe", name)
		}
	}
}
