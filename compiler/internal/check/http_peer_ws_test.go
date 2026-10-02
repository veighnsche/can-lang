package check

import (
	"strings"
	"testing"
)

// ws_send and ws_deliver_event pin their opcode to the K21
// words (text/binary/ping/pong). Destinations, connections,
// payloads, close codes and reasons stay dynamic.
func wsSendFixture(opcode string) string {
	stub := "http_peer::ws_send_receipt(\"text\", 3, 1, 3)"
	return "package app\n    provides []\n    uses [http_peer, test]\nfn http_peer::ws_send_receipt emit\n    emits {test::stale_handle, test::closed_handle, http_peer::peer_fault}\n    given\n        test::owner o\n        str connection\n        int[] payload\n    asserts\n        first: \"w1\", [1, 2, 3] => ok " + stub + "\n    match call http_peer::ws_send(o, connection, " + opcode + ", payload)\n        when\n            first: o, connection, " + opcode + ", payload => ok " + stub + "\n        test::stale_handle\n        test::closed_handle\n        http_peer::peer_fault\n        ok http_peer::ws_send_receipt got => ok got\n" + programMain + "    ok\n"
}

func wsDeliverFixture(opcode string) string {
	stub := "http_peer::ws_deliver_receipt(\"text\", 1, 3, 1)"
	return "package app\n    provides []\n    uses [http_peer, test]\nfn http_peer::ws_deliver_receipt inject\n    emits {test::stale_handle, test::closed_handle, http_peer::peer_fault}\n    given\n        test::owner o\n        str connection\n        int[] payload\n    asserts\n        first: \"w1\", [1, 2, 3] => ok " + stub + "\n    match call http_peer::ws_deliver_event(o, connection, " + opcode + ", payload)\n        when\n            first: o, connection, " + opcode + ", payload => ok " + stub + "\n        test::stale_handle\n        test::closed_handle\n        http_peer::peer_fault\n        ok http_peer::ws_deliver_receipt got => ok got\n" + programMain + "    ok\n"
}

func TestWsOpcodeLiterals(t *testing.T) {
	for _, opcode := range []string{"\"text\"", "\"binary\"", "\"ping\"", "\"pong\""} {
		if _, err := programFixture(t, map[string]string{"src/main.can": wsSendFixture(opcode)}); err != nil {
			t.Fatalf("rejected literal send opcode %s: %v", opcode, err)
		}
		if _, err := programFixture(t, map[string]string{"src/main.can": wsDeliverFixture(opcode)}); err != nil {
			t.Fatalf("rejected literal deliver opcode %s: %v", opcode, err)
		}
	}
}

func TestWsOpcodeRefusals(t *testing.T) {
	for _, tc := range []struct {
		name    string
		fixture func(string) string
		opcode  string
		want    string
	}{
		{"misspelled send opcode", wsSendFixture, "\"txt\"", "is not admitted"},
		{"dynamic send opcode", wsSendFixture, "connection", "must be a static literal"},
		{"misspelled deliver opcode", wsDeliverFixture, "\"close\"", "is not admitted"},
		{"dynamic deliver opcode", wsDeliverFixture, "connection", "must be a static literal"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := programFixture(t, map[string]string{"src/main.can": tc.fixture(tc.opcode)})
			if err == nil {
				t.Fatalf("accepted invalid ws opcode: %s", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("wrong refusal for %s: %v", tc.name, err)
			}
		})
	}
}

func TestWsDynamicInputsAdmit(t *testing.T) {
	connect := "package app\n    provides []\n    uses [http_peer, test]\nfn http_peer::ws_connection_handle dial\n    emits {test::stale_handle, http_peer::peer_fault}\n    given\n        test::owner o\n        str destination\n    asserts\n        first: \"ws-a\" => ok http_peer::ws_connection_handle(\"websocket\", \"w1\", \"grant-stub\", \"ws-a\")\n    match call http_peer::ws_connect(o, destination)\n        when\n            first: o, destination => ok http_peer::ws_connection_handle(\"websocket\", \"w1\", \"grant-stub\", \"ws-a\")\n        test::stale_handle\n        http_peer::peer_fault\n        ok http_peer::ws_connection_handle got => ok got\n" + programMain + "    ok\n"
	if _, err := programFixture(t, map[string]string{"src/main.can": connect}); err != nil {
		t.Fatalf("rejected dynamic ws destination: %v", err)
	}
	close := "package app\n    provides []\n    uses [http_peer, test]\nfn http_peer::ws_close_receipt shut\n    emits {test::stale_handle, test::closed_handle, http_peer::peer_fault}\n    given\n        test::owner o\n        str connection\n        int close_code\n        str reason\n    asserts\n        first: \"w1\", 1000, \"done\" => ok http_peer::ws_close_receipt(\"local\", 1000, false, 0, 0)\n    match call http_peer::ws_close(o, connection, close_code, reason)\n        when\n            first: o, connection, close_code, reason => ok http_peer::ws_close_receipt(\"local\", 1000, false, 0, 0)\n        test::stale_handle\n        test::closed_handle\n        http_peer::peer_fault\n        ok http_peer::ws_close_receipt got => ok got\n" + programMain + "    ok\n"
	if _, err := programFixture(t, map[string]string{"src/main.can": close}); err != nil {
		t.Fatalf("rejected dynamic ws close inputs: %v", err)
	}
}
