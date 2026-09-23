package check

import (
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

const (
	wsConnect   = "can.std.ws@1::connect"
	wsAccept    = "can.std.ws@1::accept"
	wsSendText  = "can.std.ws@1::send_text"
	wsSendBytes = "can.std.ws@1::send_bytes"
	wsClose     = "can.std.ws@1::close"
)

func wsFixture(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("../../testdata/current/ws/socket.can")
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func wsDeclarations(t *testing.T, list []*types.Type) []string {
	t.Helper()
	out := make([]string, 0, len(list))
	for _, typ := range list {
		if typ == nil {
			t.Fatal("missing checked type")
		}
		out = append(out, typ.Declaration())
	}
	return out
}

func TestWebSocketFixtureAdmitsOperationContracts(t *testing.T) {
	p, err := programFixture(t, map[string]string{"src/main.can": wsFixture(t)})
	if err != nil {
		t.Fatal(err)
	}
	contracts := map[string]struct {
		inputs []string
		result string
		errors []string
	}{
		wsConnect: {inputs: []string{"str", "", "int", "int", "int", "int", "bool"}, result: "can.std.ws@1::connection", errors: []string{"can.std.ws@1::connect_failed", "can.std.ws@1::invalid_url", "can.std.ws@1::invalid_protocol", "can.std.ws@1::limit_exceeded"}},
		wsAccept:  {inputs: []string{"can.std.http@1::request", "str", "int", "int", "int"}, result: "can.std.ws@1::connection", errors: []string{"can.std.ws@1::upgrade_failed", "can.std.ws@1::unsupported_protocol", "can.std.ws@1::invalid_protocol", "can.std.ws@1::limit_exceeded"}},
		wsSendText: {
			inputs: []string{"can.std.ws@1::session", "str"},
			result: "int",
			errors: []string{"can.std.ws@1::send_failed"},
		},
		wsSendBytes: {
			inputs: []string{"can.std.ws@1::session", "can.std.bytes@1::buffer"},
			result: "int",
			errors: []string{"can.std.ws@1::send_failed"},
		},
		wsClose: {
			inputs: []string{"can.std.ws@1::session", "int", "str"},
			result: "void",
			errors: []string{"can.std.ws@1::invalid_close"},
		},
	}
	for identity, want := range contracts {
		contract := p.Intrinsics[identity]
		if contract == nil {
			t.Fatalf("missing websocket intrinsic %s", identity)
		}
		got := wsDeclarations(t, contract.Inputs())
		if strings.Join(got, ",") != strings.Join(want.inputs, ",") {
			t.Fatalf("%s inputs = %v, want %v", identity, got, want.inputs)
		}
		if contract.Result() == nil {
			t.Fatalf("missing result for %s", identity)
		}
		result := contract.Result().Declaration()
		if want.result == "void" {
			if contract.Result().Kind() != types.Void {
				t.Fatalf("%s result kind = %v, want void", identity, contract.Result().Kind())
			}
		} else if result != want.result {
			t.Fatalf("%s result = %s, want %s", identity, result, want.result)
		}
		errors := wsDeclarations(t, contract.Errors())
		sort.Strings(errors)
		sorted := append([]string(nil), want.errors...)
		sort.Strings(sorted)
		if strings.Join(errors, ",") != strings.Join(sorted, ",") {
			t.Fatalf("%s errors = %v, want %v", identity, errors, want.errors)
		}
	}
	protocols := p.Intrinsics[wsConnect].Inputs()[1]
	if protocols.Kind() != types.Array || protocols.Element() == nil || protocols.Element().Declaration() != "str" {
		t.Fatalf("connect protocols = %v, want str[]", protocols)
	}
}

func TestWebSocketSuppliedFixturesAttachToRelay(t *testing.T) {
	p, err := programFixture(t, map[string]string{"src/main.can": wsFixture(t)})
	if err != nil {
		t.Fatal(err)
	}
	rows := map[string][]int{}
	var match func(*ir.Match)
	var completion func(*ir.Completion)
	var block func(*ir.Block)
	steps := func(call *ir.Invocation) {
		if call == nil {
			return
		}
		for i := range call.Steps {
			step := &call.Steps[i]
			if step.Fixtures == nil {
				continue
			}
			rows[step.Identity] = append(rows[step.Identity], len(step.Fixtures.Rows))
			if !strings.HasSuffix(step.Fixtures.Identity, "/when") {
				t.Fatalf("fixture table %q lacks a call-site identity", step.Fixtures.Identity)
			}
		}
	}
	completion = func(c *ir.Completion) {
		if c == nil {
			return
		}
		steps(c.Call)
		block(c.Block)
		match(c.Match)
	}
	block = func(b *ir.Block) {
		if b == nil {
			return
		}
		for i := range b.Steps {
			steps(b.Steps[i].Call)
		}
		completion(b.Terminal)
	}
	match = func(m *ir.Match) {
		if m == nil {
			return
		}
		steps(m.Call)
		for i := range m.Arms {
			completion(m.Arms[i].Body)
		}
	}
	for _, fn := range p.Functions {
		if fn.Region == nil {
			continue
		}
		block(fn.Region.Body)
	}
	// The relay stirrups successive reads from one FIFO table: text, then
	// close. Every other supplied site carries exactly one row.
	fifo := 0
	for identity, counts := range rows {
		for _, count := range counts {
			switch {
			case count == 2 && strings.Contains(identity, "read_many"):
				fifo++
			case count == 1:
			default:
				t.Fatalf("fixture table %q carries %d rows", identity, count)
			}
		}
	}
	if fifo != 1 {
		t.Fatalf("relay FIFO read table missing: %v", rows)
	}
}
