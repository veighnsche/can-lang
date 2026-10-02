package catalogue

import (
	"reflect"
	"testing"
)

// TestStoreCatalogueContract pins the NT-I17 object-store
// surface: 15 owner-first operations over the adapter-held K27
// service (open_prefix, receipt, close_prefix, open_session,
// session_token_for_test, close_session, put, settle_write,
// pending, get, delete, compare_bytes, list, seal_cleanup,
// shared_digest), 8 inert fact records, and the new store
// package with store::store_fault{layer, code}. The package
// verdict is new_store_package per evidence/I17-jev (unanimous
// 3-0; s3:: stays the live-client lane per K27's no-sharing
// scope). The token verdict is expose_token_op per the
// corrected round 2 (unanimous 3-0 at p>=0.98; round 1's
// compose advice was a framing artifact that omitted the
// identical db::connection_token_for_test precedent): the
// adapter passes sessionTokenForTest through verbatim and
// open_session returns bare SessionFacts. Thirteen fallible
// ops emit test::stale_handle and store::store_fault;
// compare_bytes and shared_digest never throw through the
// adapter (pure byte loop / pure digest over the foreign seed)
// and emit test::stale_handle only, following the I13
// compare_row/credit_verdict precedent. The K27 owner string
// travels as grant_owner to keep it distinct from the
// test::owner admission handle; put payloads and get bytes
// cross as bytes::buffer; the list continuation crosses as
// option::value<str>, none for the first page.
func TestStoreCatalogueContract(t *testing.T) {
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
	fault, ok := c.Error("store::store_fault")
	if !ok {
		t.Fatal("store::store_fault missing")
	}
	faultFields := map[string]string{}
	for _, field := range fault.Fields {
		faultFields[field.Name] = field.Type
	}
	if faultFields["layer"] != "str" || faultFields["code"] != "str" || len(faultFields) != 2 {
		t.Fatalf("store::store_fault fields differ: %+v", faultFields)
	}
	receipt := fieldsOf("store::prefix_receipt")
	if receipt["prefix"] != "str" || receipt["owner"] != "str" || receipt["handle"] != "str" || receipt["handle_digest"] != "str" || receipt["objects"] != "int" {
		t.Fatalf("store::prefix_receipt fields differ: %+v", receipt)
	}
	session := fieldsOf("store::session_facts")
	if session["session"] != "str" || session["prefix"] != "str" || session["owner"] != "str" || session["handle_digest"] != "str" || session["pinned"] != "bool" {
		t.Fatalf("store::session_facts fields differ: %+v", session)
	}
	page := fieldsOf("store::page_facts")
	if page["prefix"] != "str" || page["keys"] != "store::object_facts[]" || page["count"] != "int" || page["complete"] != "bool" || page["next_continuation"] != "option::value<str>" || page["generation"] != "int" || page["digest"] != "str" {
		t.Fatalf("store::page_facts fields differ: %+v", page)
	}
	stored := fieldsOf("store::stored_object")
	if stored["key"] != "str" || stored["bytes"] != "bytes::buffer" || stored["size"] != "int" || stored["digest"] != "str" {
		t.Fatalf("store::stored_object fields differ: %+v", stored)
	}
	type descriptor struct {
		result string
		inputs []string
		emits  []string
	}
	owner := "test::owner"
	grant := "str"
	prefix := "str"
	sessionName := "str"
	token := "str"
	key := "str"
	both := []string{"test::stale_handle", "store::store_fault"}
	stale := []string{"test::stale_handle"}
	want := map[string]descriptor{
		"store::open_prefix":           {"store::prefix_receipt", []string{owner, grant, prefix}, both},
		"store::receipt":               {"store::prefix_receipt", []string{owner, grant, prefix}, both},
		"store::close_prefix":          {"void", []string{owner, grant, prefix}, both},
		"store::open_session":          {"store::session_facts", []string{owner, grant, prefix, sessionName}, both},
		"store::session_token_for_test": {"str", []string{owner, grant, prefix, sessionName}, both},
		"store::close_session":         {"void", []string{owner, grant, prefix, sessionName, token}, both},
		"store::put":                  {"store::write_facts", []string{owner, grant, prefix, sessionName, token, key, "bytes::buffer"}, both},
		"store::settle_write":          {"store::object_facts", []string{owner, grant, prefix, sessionName, token, "str"}, both},
		"store::pending":               {"store::pending_facts", []string{owner, grant, prefix, sessionName, token}, both},
		"store::get":                  {"store::stored_object", []string{owner, grant, prefix, sessionName, token, key}, both},
		"store::delete":               {"void", []string{owner, grant, prefix, sessionName, token, key}, both},
		"store::compare_bytes":         {"bool", []string{owner, "bytes::buffer", "bytes::buffer"}, stale},
		"store::list":                 {"store::page_facts", []string{owner, grant, prefix, sessionName, token, "int", "option::value<str>"}, both},
		"store::seal_cleanup":          {"store::cleanup_receipt", []string{owner, grant, prefix}, both},
		"store::shared_digest":         {"str", []string{owner}, stale},
	}
	if len(want) != 15 {
		t.Fatalf("contract pins %d operations, want 15", len(want))
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
		if op.Lowering.Task != "NT-I17" || !reflect.DeepEqual(op.Refs, []string{"NT-I17"}) || op.Assertion != "supplied" {
			t.Fatalf("%s traceability differs: task=%q refs=%v assertion=%q", name, op.Lowering.Task, op.Refs, op.Assertion)
		}
		if len(op.Lowering.Native) == 0 || op.Lowering.Adapter == "" {
			t.Fatalf("%s names no native recipe", name)
		}
	}
}
