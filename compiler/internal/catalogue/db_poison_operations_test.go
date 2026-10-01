package catalogue

import (
	"reflect"
	"testing"
)

// TestDbPoisonCatalogueContract pins the NT-I15 poisoned
// transaction surface: 17 owner-first operations over the
// adapter-held K25 service (begin_poison_attempt,
// begin_control_attempt, attempt_record, record_poison_error,
// error_facts, record_callback_report, callback_facts,
// record_poison_settlement, poison_settlement_facts,
// record_fresh_read, fresh_facts, record_replay_read,
// replay_facts, compare_fresh_to_replay,
// sentinel_present_in_fresh, sentinel_present_in_replay,
// settlement_agrees_with_reads), 9 inert fact records, reusing
// the db package and db::db_fault{layer, code}. No new Jev
// consult: the I13 decision (evidence/I13-jev,
// full_adapter_held_service) settled the observer-family merge
// shape. All 17 ops emit test::stale_handle and db::db_fault:
// even the boolean/verdict readers throw on missing or
// cross-attempt evidence. (The union-typed sentinelPresentIn
// read splits into _fresh/_replay ops; the poison settlement
// fact type is db::poison_settlement_record to avoid the op/type
// clash; poison_digest is option::value<str>, none for the
// omit-poison control.)
func TestDbPoisonCatalogueContract(t *testing.T) {
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
	attempt := fieldsOf("db::poison_attempt_record")
	if attempt["kind"] != "str" || attempt["schema"] != "str[]" || attempt["sentinel_digest"] != "str" || attempt["poison_digest"] != "option::value<str>" {
		t.Fatalf("db::poison_attempt_record fields differ: %+v", attempt)
	}
	errorFacts := fieldsOf("db::poison_error_facts")
	if errorFacts["code"] != "str" || errorFacts["class"] != "str" || errorFacts["detail_digest"] != "str" {
		t.Fatalf("db::poison_error_facts fields differ: %+v", errorFacts)
	}
	callback := fieldsOf("db::poison_callback_facts")
	if callback["reported"] != "str" || callback["proves"] != "str" {
		t.Fatalf("db::poison_callback_facts fields differ: %+v", callback)
	}
	fresh := fieldsOf("db::fresh_sentinel_facts")
	if fresh["read"] != "str" || fresh["rows"] != "db::sentinel_row_facts[]" || fresh["row_count"] != "int" {
		t.Fatalf("db::fresh_sentinel_facts fields differ: %+v", fresh)
	}
	agreement := fieldsOf("db::settlement_agreement")
	if agreement["agree"] != "bool" || agreement["reason"] != "str" {
		t.Fatalf("db::settlement_agreement fields differ: %+v", agreement)
	}
	type descriptor struct {
		result string
		inputs []string
	}
	owner := "test::owner"
	want := map[string]descriptor{
		"db::begin_poison_attempt":         {"db::poison_attempt_record", []string{owner, "str[]", "db::seed_row", "str"}},
		"db::begin_control_attempt":        {"db::poison_attempt_record", []string{owner, "str[]", "db::seed_row"}},
		"db::attempt_record":               {"db::poison_attempt_record", []string{owner, "str"}},
		"db::record_poison_error":          {"db::poison_error_facts", []string{owner, "str", "str", "str"}},
		"db::error_facts":                  {"db::poison_error_facts", []string{owner, "str"}},
		"db::record_callback_report":       {"db::poison_callback_facts", []string{owner, "str", "str"}},
		"db::callback_facts":               {"db::poison_callback_facts", []string{owner, "str"}},
		"db::record_poison_settlement":     {"db::poison_settlement_record", []string{owner, "str", "str"}},
		"db::poison_settlement_facts":      {"db::poison_settlement_record", []string{owner, "str"}},
		"db::record_fresh_read":            {"db::fresh_sentinel_facts", []string{owner, "str", "db::seed_row[]"}},
		"db::fresh_facts":                  {"db::fresh_sentinel_facts", []string{owner, "str"}},
		"db::record_replay_read":           {"db::replay_sentinel_facts", []string{owner, "str", "db::seed_row[]"}},
		"db::replay_facts":                 {"db::replay_sentinel_facts", []string{owner, "str"}},
		"db::compare_fresh_to_replay":      {"db::sentinel_comparison", []string{owner, "db::fresh_sentinel_facts", "db::replay_sentinel_facts"}},
		"db::sentinel_present_in_fresh":    {"bool", []string{owner, "str", "db::fresh_sentinel_facts"}},
		"db::sentinel_present_in_replay":   {"bool", []string{owner, "str", "db::replay_sentinel_facts"}},
		"db::settlement_agrees_with_reads": {"db::settlement_agreement", []string{owner, "str"}},
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
		if op.Lowering.Task != "NT-I15" || !reflect.DeepEqual(op.Refs, []string{"NT-I15"}) || op.Assertion != "supplied" {
			t.Fatalf("%s traceability differs: task=%q refs=%v assertion=%q", name, op.Lowering.Task, op.Refs, op.Assertion)
		}
		if len(op.Lowering.Native) == 0 || op.Lowering.Adapter == "" {
			t.Fatalf("%s names no native recipe", name)
		}
	}
}
