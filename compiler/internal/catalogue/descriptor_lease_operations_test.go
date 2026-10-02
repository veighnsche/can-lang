package catalogue

import (
	"reflect"
	"testing"
)

// TestDescriptorLeaseCatalogueContract pins the NT-I12 inherited
// generation-lease surface: 1 owner-first operation over the N-owner
// envelope (probe_lease), reusing the descriptor package, the
// I11-forwarded descriptor::lease_report record, and
// descriptor::descriptor_fault from NT-I11. No new types: the probe
// result shape already exists. No new Jev consult: the probe
// observes live external N-owner lease state through the injected
// dispatch, which is the I11 same-package envelope precedent, not
// the I13 observer-family held-service shape. The probe does real
// owner I/O, so it emits test::stale_handle and
// descriptor::descriptor_fault.
func TestDescriptorLeaseCatalogueContract(t *testing.T) {
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
	report := fieldsOf("descriptor::lease_report")
	if report["path"] != "str" || report["inherited"] != "bool" || report["held"] != "bool" || report["released"] != "bool" {
		t.Fatalf("descriptor::lease_report fields differ: %+v", report)
	}
	type descriptor struct {
		result string
		inputs []string
		emits  []string
	}
	owner := "test::owner"
	both := []string{"test::stale_handle", "descriptor::descriptor_fault"}
	want := map[string]descriptor{
		"descriptor::probe_lease": {"descriptor::lease_report", []string{owner, "str"}, both},
	}
	if len(want) != 1 {
		t.Fatalf("contract pins %d operations, want 1", len(want))
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
		if op.Lowering.Task != "NT-I12" || !reflect.DeepEqual(op.Refs, []string{"NT-I12"}) || op.Assertion != "supplied" {
			t.Fatalf("%s traceability differs: task=%q refs=%v assertion=%q", name, op.Lowering.Task, op.Refs, op.Assertion)
		}
		if len(op.Lowering.Native) == 0 || op.Lowering.Adapter == "" {
			t.Fatalf("%s names no native recipe", name)
		}
	}
}
