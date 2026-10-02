package catalogue

import (
	"reflect"
	"testing"
)

// TestWsCatalogueContract pins the NT-I05 websocket surface:
// 7 owner-first operations over the adapter-held K21 service
// (ws_connect, ws_send, ws_deliver_event, ws_poll_event,
// ws_close, ws_deliver_remote_close, ws_read_connection_facts)
// plus 8 inert fact records, reusing the http_peer package
// and http_peer::peer_fault{kind, reason}. The ws_ prefix
// keeps the family collision-free: the I04 peer family owns
// the bare connection nouns (close_connection,
// read_connection_facts). The admitted grant is the service
// owner (I04 precedent); destinations stay dynamic against
// the declared set. Five half-close-aware ops emit
// test::stale_handle, test::closed_handle and
// http_peer::peer_fault; ws_connect and
// ws_read_connection_facts never observe closed-handle and
// emit stale_handle plus peer_fault. Null closes and empty
// polls cross as option values.
func TestWsCatalogueContract(t *testing.T) {
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
	handle := fieldsOf("http_peer::ws_connection_handle")
	if handle["kind"] != "str" || handle["id"] != "str" || handle["owner"] != "str" || handle["destination"] != "str" {
		t.Fatalf("http_peer::ws_connection_handle fields differ: %+v", handle)
	}
	event := fieldsOf("http_peer::ws_event")
	if event["opcode"] != "str" || event["payload"] != "int[]" {
		t.Fatalf("http_peer::ws_event fields differ: %+v", event)
	}
	poll := fieldsOf("http_peer::ws_poll_result")
	if poll["event"] != "option::value<http_peer::ws_event>" || poll["pending_events"] != "int" || poll["consumed_total"] != "int" || poll["delivered_total"] != "int" {
		t.Fatalf("http_peer::ws_poll_result fields differ: %+v", poll)
	}
	facts := fieldsOf("http_peer::ws_connection_facts")
	if facts["state"] != "str" || facts["local_close"] != "option::value<http_peer::ws_close_facts>" || facts["remote_close"] != "option::value<http_peer::ws_close_facts>" || facts["dropped_bytes"] != "int" {
		t.Fatalf("http_peer::ws_connection_facts fields differ: %+v", facts)
	}
	type descriptor struct {
		result string
		inputs []string
		emits  []string
	}
	owner := "test::owner"
	connection := "str"
	full := []string{"test::stale_handle", "test::closed_handle", "http_peer::peer_fault"}
	noclose := []string{"test::stale_handle", "http_peer::peer_fault"}
	want := map[string]descriptor{
		"http_peer::ws_connect":               {"http_peer::ws_connection_handle", []string{owner, "str"}, noclose},
		"http_peer::ws_send":                  {"http_peer::ws_send_receipt", []string{owner, connection, "str", "int[]"}, full},
		"http_peer::ws_deliver_event":         {"http_peer::ws_deliver_receipt", []string{owner, connection, "str", "int[]"}, full},
		"http_peer::ws_poll_event":            {"http_peer::ws_poll_result", []string{owner, connection}, full},
		"http_peer::ws_close":                 {"http_peer::ws_close_receipt", []string{owner, connection, "int", "str"}, full},
		"http_peer::ws_deliver_remote_close":  {"http_peer::ws_close_receipt", []string{owner, connection, "int", "str"}, full},
		"http_peer::ws_read_connection_facts": {"http_peer::ws_connection_facts", []string{owner, connection}, noclose},
	}
	if len(want) != 7 {
		t.Fatalf("contract pins %d operations, want 7", len(want))
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
		if op.Lowering.Task != "NT-I05" || !reflect.DeepEqual(op.Refs, []string{"NT-I05"}) || op.Assertion != "supplied" {
			t.Fatalf("%s traceability differs: task=%q refs=%v assertion=%q", name, op.Lowering.Task, op.Refs, op.Assertion)
		}
		if len(op.Lowering.Native) == 0 || op.Lowering.Adapter == "" {
			t.Fatalf("%s names no native recipe", name)
		}
	}
}
