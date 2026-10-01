package catalogue

import (
	"reflect"
	"testing"
)

// TestDatabaseObserverCatalogueContract pins the NT-I13 database row
// observation surface: 20 owner-first operations over the adapter-held
// K22/K23 services (open_namespace, receipt, close_namespace, seed,
// pin_connection, connection_token_for_test, unpin_connection,
// begin_read, fetch, end_read, compare_row, record_compile,
// compile_record, record_returning, payload_facts, record_final_rows,
// final_facts, compare_payload_to_final, compare_claimed_row,
// credit_verdict), 19 types (the db::cell variant with four leaves
// plus 14 inert fact records and the seed_row input wrapper), and the
// db_fault{layer, code} error. The Jev x3 consult
// (evidence/I13-jev, full_adapter_held_service 2-1 with investigated
// dissent) chose the I03/I04 doubles precedent over the I02-style
// pure/stateful split and N-owner dispatch: the observer holds no
// credit authority (creditVerdict is constant-false, QD1 holds
// credit), and the I13 card mandates the complete handle/effect
// surface with no Q-case deferral. Seventeen fallible ops emit
// test::stale_handle and db::db_fault; the three never-throwing
// pure comparisons (compare_row, compare_claimed_row,
// credit_verdict) emit test::stale_handle only.
func TestDatabaseObserverCatalogueContract(t *testing.T) {
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
	if err := c.CheckProjectPackage("db"); err == nil {
		t.Fatal("db package is not distribution-owned")
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
	leavesOf := func(name string) []string {
		t.Helper()
		typ, ok := c.Type(name)
		if !ok {
			t.Fatalf("%s missing", name)
		}
		return typ.Leaves
	}
	if leaves := leavesOf("db::cell"); !reflect.DeepEqual(leaves, []string{"db::number_cell", "db::text_cell", "db::bytes_cell", "db::null_cell"}) {
		t.Fatalf("db::cell leaves differ: %+v", leaves)
	}
	number := fieldsOf("db::number_cell")
	if number["lexeme"] != "str" {
		t.Fatalf("db::number_cell fields differ: %+v", number)
	}
	bytesCell := fieldsOf("db::bytes_cell")
	if bytesCell["data"] != "bytes::buffer" {
		t.Fatalf("db::bytes_cell fields differ: %+v", bytesCell)
	}
	receipt := fieldsOf("db::namespace_receipt")
	if receipt["namespace"] != "str" || receipt["owner"] != "str" || receipt["handle"] != "str" || receipt["handle_digest"] != "str" || receipt["tables"] != "int" {
		t.Fatalf("db::namespace_receipt fields differ: %+v", receipt)
	}
	connection := fieldsOf("db::connection_facts")
	if connection["connection"] != "str" || connection["pinned"] != "bool" || connection["conversation_open"] != "bool" {
		t.Fatalf("db::connection_facts fields differ: %+v", connection)
	}
	row := fieldsOf("db::row_facts")
	if row["seq"] != "int" || row["cells"] != "db::cell[]" || row["digest"] != "str" {
		t.Fatalf("db::row_facts fields differ: %+v", row)
	}
	read := fieldsOf("db::read_facts")
	if read["table"] != "str" || read["rows"] != "db::row_facts[]" || read["row_count"] != "int" || read["digest"] != "str" {
		t.Fatalf("db::read_facts fields differ: %+v", read)
	}
	compile := fieldsOf("db::compile_facts")
	if compile["compile"] != "str" || compile["fixture"] != "str" || compile["statement_digest"] != "str" || compile["schema"] != "str[]" {
		t.Fatalf("db::compile_facts fields differ: %+v", compile)
	}
	verdict := fieldsOf("db::returning_credit_verdict")
	if verdict["credit_c"] != "bool" || verdict["reason"] != "str" {
		t.Fatalf("db::returning_credit_verdict fields differ: %+v", verdict)
	}
	type descriptor struct {
		result string
		inputs []string
		emits  []string
	}
	owner := "test::owner"
	namespace := "str"
	stale := []string{"test::stale_handle"}
	both := []string{"test::stale_handle", "db::db_fault"}
	want := map[string]descriptor{
		"db::open_namespace":            {"db::namespace_receipt", []string{owner, namespace}, both},
		"db::receipt":                   {"db::namespace_receipt", []string{owner, namespace}, both},
		"db::close_namespace":           {"void", []string{owner, namespace}, both},
		"db::seed":                      {"db::seed_facts", []string{owner, namespace, "str", "db::seed_row[]"}, both},
		"db::pin_connection":            {"db::connection_facts", []string{owner, namespace, "str"}, both},
		"db::connection_token_for_test": {"str", []string{owner, namespace, "str"}, both},
		"db::unpin_connection":          {"void", []string{owner, namespace, "str", "str"}, both},
		"db::begin_read":                {"db::conversation_facts", []string{owner, namespace, "str", "str", "str"}, both},
		"db::fetch":                     {"db::read_facts", []string{owner, namespace, "str", "str", "str"}, both},
		"db::end_read":                  {"void", []string{owner, namespace, "str", "str"}, both},
		"db::compare_row":               {"db::row_comparison", []string{owner, "db::cell[]", "db::cell[]"}, stale},
		"db::record_compile":            {"db::compile_facts", []string{owner, "str", "str", "str[]"}, both},
		"db::compile_record":            {"db::compile_facts", []string{owner, "str"}, both},
		"db::record_returning":          {"db::returning_payload_facts", []string{owner, "str", "db::seed_row[]"}, both},
		"db::payload_facts":             {"db::returning_payload_facts", []string{owner, "str"}, both},
		"db::record_final_rows":         {"db::final_rows_facts", []string{owner, "str", "db::seed_row[]"}, both},
		"db::final_facts":               {"db::final_rows_facts", []string{owner, "str"}, both},
		"db::compare_payload_to_final":  {"db::returning_comparison", []string{owner, "db::returning_payload_facts", "db::final_rows_facts"}, both},
		"db::compare_claimed_row":       {"db::returning_comparison", []string{owner, "db::cell[]", "db::cell[]"}, stale},
		"db::credit_verdict":            {"db::returning_credit_verdict", []string{owner}, stale},
	}
	if len(want) != 20 {
		t.Fatalf("contract pins %d operations, want 20", len(want))
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
		if op.Lowering.Task != "NT-I13" || !reflect.DeepEqual(op.Refs, []string{"NT-I13"}) || op.Assertion != "supplied" {
			t.Fatalf("%s traceability differs: task=%q refs=%v assertion=%q", name, op.Lowering.Task, op.Refs, op.Assertion)
		}
		if len(op.Lowering.Native) == 0 || op.Lowering.Adapter == "" {
			t.Fatalf("%s names no native recipe", name)
		}
	}
}
