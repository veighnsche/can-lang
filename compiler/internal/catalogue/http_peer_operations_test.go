package catalogue

import (
	"reflect"
	"testing"
)

// TestHttpPeerCatalogueContract pins the NT-I04 HTTP and controlled-peers
// surface: 22 operations mirroring the K20 peer/http test services 1:1
// over 4 opaque handles, 14 fact/receipt records, and 2 faults. Every op
// takes test::owner first and admits before effects; foreign owners fail
// as test::stale_handle, service failures as the slice fault. Handles are
// opaque worker-local bindings, never ids or bytes; facts carry data plus
// subsidiary handles (dial_result.connection), never live service state.
func TestHttpPeerCatalogueContract(t *testing.T) {
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
	for _, name := range []string{"http_peer::listener", "http_peer::connection", "http_peer::dial", "http_peer::request"} {
		handle, ok := c.Type(name)
		if !ok || handle.Kind != "opaque" || handle.Constructible || len(handle.Projections) != 0 {
			t.Fatalf("%s marker differs: %+v", name, handle)
		}
		if err := c.CheckConstructor(name); err == nil {
			t.Fatalf("%s admits a public constructor", name)
		}
	}
	if err := c.CheckProjectPackage("http_peer"); err == nil {
		t.Fatal("http_peer package is not distribution-owned")
	}
	for _, name := range []string{"http_peer::peer_fault", "http_peer::http_fault"} {
		fault, ok := c.Error(name)
		if !ok || len(fault.Fields) != 2 || fault.Fields[0].Name != "kind" || fault.Fields[0].Type != "str" || fault.Fields[1].Name != "reason" || fault.Fields[1].Type != "str" {
			t.Fatalf("%s marker differs: %+v", name, fault)
		}
	}
	dialResult, ok := c.Type("http_peer::dial_result")
	if !ok {
		t.Fatal("http_peer::dial_result missing")
	}
	fields := map[string]string{}
	for _, field := range dialResult.Fields {
		fields[field.Name] = field.Type
	}
	if fields["dial"] != "http_peer::dial" || fields["connection"] != "option::value<http_peer::connection>" || fields["fail_reason"] != "option::value<str>" {
		t.Fatalf("http_peer::dial_result fields differ: %+v", dialResult.Fields)
	}
	facts, ok := c.Type("http_peer::request_facts")
	if !ok {
		t.Fatal("http_peer::request_facts missing")
	}
	headered := false
	for _, field := range facts.Fields {
		if field.Name == "headers" && field.Type == "http::header[]" {
			headered = true
		}
	}
	if !headered {
		t.Fatal("http_peer::request_facts headers are not http::header[]")
	}
	type descriptor struct {
		result string
		inputs []string
		emits  []string
	}
	stale := "test::stale_handle"
	closed := "test::closed_handle"
	peerFault := "http_peer::peer_fault"
	httpFault := "http_peer::http_fault"
	want := map[string]descriptor{
		"http_peer::open_listener":         {"http_peer::listener", []string{"test::owner", "str"}, []string{stale, peerFault}},
		"http_peer::close_listener":        {"http_peer::listener_close_receipt", []string{"test::owner", "http_peer::listener"}, []string{stale, closed, peerFault}},
		"http_peer::dial_peer":             {"http_peer::dial_result", []string{"test::owner", "str"}, []string{stale, peerFault}},
		"http_peer::retry_dial":            {"http_peer::dial_result", []string{"test::owner", "http_peer::dial"}, []string{stale, peerFault}},
		"http_peer::accept":                {"http_peer::connection", []string{"test::owner", "http_peer::listener"}, []string{stale, closed, peerFault}},
		"http_peer::write":                 {"http_peer::write_receipt", []string{"test::owner", "http_peer::connection", "int[]"}, []string{stale, closed, peerFault}},
		"http_peer::read":                  {"http_peer::read_result", []string{"test::owner", "http_peer::connection", "int"}, []string{stale, closed, peerFault}},
		"http_peer::half_close":            {"http_peer::connection_facts", []string{"test::owner", "http_peer::connection", "str"}, []string{stale, closed, peerFault}},
		"http_peer::close_connection":      {"http_peer::connection_close_receipt", []string{"test::owner", "http_peer::connection"}, []string{stale, closed, peerFault}},
		"http_peer::read_listener_facts":   {"http_peer::listener_facts", []string{"test::owner", "http_peer::listener"}, []string{stale, peerFault}},
		"http_peer::read_connection_facts": {"http_peer::connection_facts", []string{"test::owner", "http_peer::connection"}, []string{stale, peerFault}},
		"http_peer::read_dial_facts":       {"http_peer::dial_facts", []string{"test::owner", "http_peer::dial"}, []string{stale, peerFault}},
		"http_peer::open_request":          {"http_peer::request", []string{"test::owner", "str", "str", "str", "str"}, []string{stale, httpFault}},
		"http_peer::add_header":            {"http_peer::header_receipt", []string{"test::owner", "http_peer::request", "str", "str"}, []string{stale, closed, httpFault}},
		"http_peer::send_body_chunk":       {"http_peer::body_chunk_receipt", []string{"test::owner", "http_peer::request", "int[]"}, []string{stale, closed, httpFault}},
		"http_peer::end_upload":            {"http_peer::request_facts", []string{"test::owner", "http_peer::request"}, []string{stale, closed, httpFault}},
		"http_peer::deliver_response":      {"http_peer::response_facts", []string{"test::owner", "http_peer::request", "int", "http::header[]", "int[]"}, []string{stale, closed, httpFault}},
		"http_peer::read_body_chunk":       {"http_peer::read_result", []string{"test::owner", "http_peer::request", "int"}, []string{stale, closed, httpFault}},
		"http_peer::reissue":               {"http_peer::request", []string{"test::owner", "http_peer::request"}, []string{stale, closed, httpFault}},
		"http_peer::close_request":         {"http_peer::request_close_receipt", []string{"test::owner", "http_peer::request"}, []string{stale, closed, httpFault}},
		"http_peer::read_request_facts":    {"http_peer::request_facts", []string{"test::owner", "http_peer::request"}, []string{stale, httpFault}},
		// read_response_facts returns absent-or-facts: the option wrapper,
		// not the bare record, since no response may be delivered yet.
		"http_peer::read_response_facts": {"option::value<http_peer::response_facts>", []string{"test::owner", "http_peer::request"}, []string{stale, httpFault}},
	}
	if len(want) != 22 {
		t.Fatalf("contract pins %d operations, want 22", len(want))
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
		if !reflect.DeepEqual(op.Emits, descriptor.emits) {
			t.Fatalf("%s bound differs: %+v", name, op.Emits)
		}
		if op.Lowering.Task != "NT-I04" || !reflect.DeepEqual(op.Refs, []string{"NT-I04"}) || op.Assertion != "real" {
			t.Fatalf("%s traceability differs: task=%q refs=%v assertion=%q", name, op.Lowering.Task, op.Refs, op.Assertion)
		}
		if len(op.Lowering.Native) == 0 || op.Lowering.Adapter == "" {
			t.Fatalf("%s names no native recipe", name)
		}
	}
}
