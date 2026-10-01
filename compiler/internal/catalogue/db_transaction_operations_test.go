package catalogue

import (
	"reflect"
	"testing"
)

// TestDbTransactionCatalogueContract pins the NT-I14 fresh-pool
// transaction surface: 9 owner-first operations over the
// adapter-held K24 service (enter_callback, actor_facts,
// record_last_insert_id, last_insert_id_facts,
// compare_last_insert_id, record_settlement, settlement_facts,
// release, release_ack), 8 inert fact records, reusing the db
// package and db::db_fault{layer, code} from NT-I13. No new Jev
// consult: the I13 decision (evidence/I13-jev,
// full_adapter_held_service) settled the observer-family merge
// shape, and I14 applies it to the sibling module. Eight
// fallible ops emit test::stale_handle and db::db_fault; the
// never-throwing compare_last_insert_id emits test::stale_handle
// only. (Type/op name clashes resolve as in I13: ops keep the
// method names settlement_facts, release_ack and
// last_insert_id_facts; the fact types are
// db::settlement_record, db::release_record and
// db::last_insert_id_record.)
func TestDbTransactionCatalogueContract(t *testing.T) {
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
	entry := fieldsOf("db::callback_entry")
	if entry["facts"] != "db::callback_entry_facts" || entry["token"] != "str" {
		t.Fatalf("db::callback_entry fields differ: %+v", entry)
	}
	actor := fieldsOf("db::actor_identity_facts")
	if actor["actor"] != "str" || actor["connection"] != "str" || actor["entry_seq"] != "int" || actor["settled"] != "bool" || actor["released"] != "bool" {
		t.Fatalf("db::actor_identity_facts fields differ: %+v", actor)
	}
	lastID := fieldsOf("db::last_insert_id_record")
	if lastID["id"] != "db::cell" || lastID["digest"] != "str" {
		t.Fatalf("db::last_insert_id_record fields differ: %+v", lastID)
	}
	claim := fieldsOf("db::last_insert_id_claim")
	if claim["actor"] != "str" || claim["connection"] != "str" || claim["id"] != "db::cell" {
		t.Fatalf("db::last_insert_id_claim fields differ: %+v", claim)
	}
	settlement := fieldsOf("db::settlement_record")
	if settlement["outcome"] != "str" || settlement["engine"] != "str" || settlement["digest"] != "str" {
		t.Fatalf("db::settlement_record fields differ: %+v", settlement)
	}
	release := fieldsOf("db::release_record")
	if release["released"] != "bool" || release["ack_digest"] != "str" {
		t.Fatalf("db::release_record fields differ: %+v", release)
	}
	comparison := fieldsOf("db::identity_comparison")
	if comparison["match"] != "bool" || comparison["mismatches"] != "int[]" {
		t.Fatalf("db::identity_comparison fields differ: %+v", comparison)
	}
	type descriptor struct {
		result string
		inputs []string
		emits  []string
	}
	owner := "test::owner"
	stale := []string{"test::stale_handle"}
	both := []string{"test::stale_handle", "db::db_fault"}
	want := map[string]descriptor{
		"db::enter_callback":         {"db::callback_entry", []string{owner, "str"}, both},
		"db::actor_facts":            {"db::actor_identity_facts", []string{owner, "str"}, both},
		"db::record_last_insert_id":  {"db::last_insert_id_record", []string{owner, "str", "str", "db::cell"}, both},
		"db::last_insert_id_facts":   {"db::last_insert_id_record", []string{owner, "str"}, both},
		"db::compare_last_insert_id": {"db::identity_comparison", []string{owner, "db::last_insert_id_record", "db::last_insert_id_claim"}, stale},
		"db::record_settlement":      {"db::settlement_record", []string{owner, "str", "str", "str", "str"}, both},
		"db::settlement_facts":       {"db::settlement_record", []string{owner, "str"}, both},
		"db::release":                {"db::release_record", []string{owner, "str", "str"}, both},
		"db::release_ack":            {"db::release_record", []string{owner, "str"}, both},
	}
	if len(want) != 9 {
		t.Fatalf("contract pins %d operations, want 9", len(want))
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
		if op.Lowering.Task != "NT-I14" || !reflect.DeepEqual(op.Refs, []string{"NT-I14"}) || op.Assertion != "supplied" {
			t.Fatalf("%s traceability differs: task=%q refs=%v assertion=%q", name, op.Lowering.Task, op.Refs, op.Assertion)
		}
		if len(op.Lowering.Native) == 0 || op.Lowering.Adapter == "" {
			t.Fatalf("%s names no native recipe", name)
		}
	}
}
