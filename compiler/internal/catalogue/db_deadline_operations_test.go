package catalogue

import (
	"reflect"
	"testing"
)

// TestDbDeadlineCatalogueContract pins the NT-I16 deadline
// settlement surface: 17 owner-first operations over the
// adapter-held K26 service (dispatch, dispatch_facts,
// observe_deadline, deadline_facts, record_driver_settlement,
// driver_facts, record_server_ack, server_facts,
// acquire_cancel_grant, request_cancel, cancel_facts,
// quiesce_engine, fence, fence_facts, lease_facts,
// deadline_release, deadline_release_ack), 11 inert fact
// records, reusing the db package and db::db_fault{layer,
// code}. No new Jev consult: the I13 decision (evidence/I13-jev,
// full_adapter_held_service) settled the observer-family merge
// shape. All 17 ops emit test::stale_handle and db::db_fault:
// even the fact readers throw on unknown work labels. (Method
// dispatch returns an anonymous {facts, token} wrapper, named
// db::dispatched_work here because db::dispatch would clash
// with the op identity; release/releaseAck become
// deadline_release/deadline_release_ack because db::release and
// db::release_ack already exist in the I14 family.)
func TestDbDeadlineCatalogueContract(t *testing.T) {
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
	dispatched := fieldsOf("db::dispatched_work")
	if dispatched["facts"] != "db::dispatch_record" || dispatched["token"] != "str" {
		t.Fatalf("db::dispatched_work fields differ: %+v", dispatched)
	}
	deadline := fieldsOf("db::deadline_record")
	if deadline["work"] != "str" || deadline["engine"] != "str" || deadline["deadline_ms"] != "int" || deadline["digest"] != "str" {
		t.Fatalf("db::deadline_record fields differ: %+v", deadline)
	}
	grant := fieldsOf("db::cancel_grant")
	if grant["facts"] != "db::cancel_grant_facts" || grant["grant"] != "str" {
		t.Fatalf("db::cancel_grant fields differ: %+v", grant)
	}
	lease := fieldsOf("db::lease_record")
	if lease["work"] != "str" || lease["engine"] != "str" || lease["namespace"] != "str" || lease["handle_digest"] != "str" || lease["retained"] != "bool" {
		t.Fatalf("db::lease_record fields differ: %+v", lease)
	}
	release := fieldsOf("db::deadline_release_record")
	if release["work"] != "str" || release["engine"] != "str" || release["released"] != "bool" || release["ack_digest"] != "str" {
		t.Fatalf("db::deadline_release_record fields differ: %+v", release)
	}
	type descriptor struct {
		result string
		inputs []string
	}
	owner := "test::owner"
	want := map[string]descriptor{
		"db::dispatch":                 {"db::dispatched_work", []string{owner, "str", "str", "str"}},
		"db::dispatch_facts":           {"db::dispatch_record", []string{owner, "str"}},
		"db::observe_deadline":         {"db::deadline_record", []string{owner, "str", "str", "int"}},
		"db::deadline_facts":           {"db::deadline_record", []string{owner, "str"}},
		"db::record_driver_settlement": {"db::driver_settlement_facts", []string{owner, "str", "str", "str"}},
		"db::driver_facts":             {"db::driver_settlement_facts", []string{owner, "str"}},
		"db::record_server_ack":        {"db::server_ack_facts", []string{owner, "str", "str", "str"}},
		"db::server_facts":             {"db::server_ack_facts", []string{owner, "str"}},
		"db::acquire_cancel_grant":     {"db::cancel_grant", []string{owner, "str"}},
		"db::request_cancel":           {"db::cancel_record", []string{owner, "str", "str", "str"}},
		"db::cancel_facts":             {"db::cancel_record", []string{owner, "str"}},
		"db::quiesce_engine":           {"str[]", []string{owner, "str"}},
		"db::fence":                    {"db::fence_record", []string{owner, "str", "str"}},
		"db::fence_facts":              {"db::fence_record", []string{owner, "str"}},
		"db::lease_facts":              {"db::lease_record", []string{owner, "str"}},
		"db::deadline_release":         {"db::deadline_release_record", []string{owner, "str", "str"}},
		"db::deadline_release_ack":     {"db::deadline_release_record", []string{owner, "str"}},
	}
	if len(want) != 17 {
		t.Fatalf("contract pins %d operations, want 17", len(want))
	}
	both := []string{"test::stale_handle", "db::db_fault"}
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
		if !reflect.DeepEqual(op.Emits, both) {
			t.Fatalf("%s emits differ: %+v", name, op.Emits)
		}
		if op.Lowering.Task != "NT-I16" || !reflect.DeepEqual(op.Refs, []string{"NT-I16"}) || op.Assertion != "supplied" {
			t.Fatalf("%s traceability differs: task=%q refs=%v assertion=%q", name, op.Lowering.Task, op.Refs, op.Assertion)
		}
		if len(op.Lowering.Native) == 0 || op.Lowering.Adapter == "" {
			t.Fatalf("%s names no native recipe", name)
		}
	}
}
