package check

import (
	"os"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

const (
	serverMakeConfig = "can.std.http@1::make_server_config"
	serverStart      = "can.std.http@1::server_start"
	serverStop       = "can.std.http@1::server_stop"
	serverWait       = "can.std.http@1::server_wait"
)

func serverFixture(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("../../testdata/current/http/server.can")
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func serverDeclarations(t *testing.T, list []*types.Type) []string {
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

func TestServerFixtureAdmitsConfigurationContracts(t *testing.T) {
	p, err := programFixture(t, map[string]string{"src/main.can": serverFixture(t)})
	if err != nil {
		t.Fatal(err)
	}
	contracts := map[string]struct {
		inputs []string
		result string
		errors []string
	}{
		serverMakeConfig: {inputs: []string{"str", "int", "int", "int"}, result: "can.std.http@1::server_config", errors: []string{"can.std.http@1::invalid_server_config"}},
		serverStart:      {inputs: []string{"can.std.http@1::server_config", "can.std.http@1::router"}, result: "can.std.http@1::server", errors: []string{"can.std.http@1::bind_failed"}},
		serverStop:       {inputs: []string{"can.std.http@1::server"}, result: "void", errors: []string{"can.std.http@1::shutdown_failed"}},
		serverWait:       {inputs: []string{"can.std.http@1::server"}, result: "void", errors: []string{"can.std.http@1::shutdown_failed"}},
	}
	for identity, want := range contracts {
		contract := p.Intrinsics[identity]
		if contract == nil {
			t.Fatalf("missing server intrinsic %s", identity)
		}
		got := serverDeclarations(t, contract.Inputs())
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
		errors := serverDeclarations(t, contract.Errors())
		if strings.Join(errors, ",") != strings.Join(want.errors, ",") {
			t.Fatalf("%s errors = %v, want %v", identity, errors, want.errors)
		}
	}
}

func TestServerSuppliedFixturesAttachToLifecycle(t *testing.T) {
	p, err := programFixture(t, map[string]string{"src/main.can": serverFixture(t)})
	if err != nil {
		t.Fatal(err)
	}
	tables := map[string]int{}
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
			tables[step.Identity]++
			if !strings.HasSuffix(step.Fixtures.Identity, "/when") {
				t.Fatalf("fixture table %q lacks a call-site identity", step.Fixtures.Identity)
			}
			if len(step.Fixtures.Rows) != 1 || step.Fixtures.Rows[0].Selector != "sample" {
				t.Fatalf("fixture table %q lost its sampled row", step.Fixtures.Identity)
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
	want := map[string]int{serverStart: 2, serverStop: 1, serverWait: 1}
	if len(tables) != len(want) {
		t.Fatalf("fixture tables = %v, want %v", tables, want)
	}
	for identity, count := range want {
		if tables[identity] != count {
			t.Fatalf("fixture tables = %v, want %v", tables, want)
		}
	}
}

func TestServerLifecycleRejectsMismatches(t *testing.T) {
	base := serverFixture(t)
	for name, replace := range map[string]struct{ old, new string }{
		"swapped start arguments": {"call http::server_start(config, built)", "call http::server_start(built, config)"},
		"config stop argument":    {"call http::server_stop(sturdy)", "call http::server_stop(config)"},
		"config wait argument":    {"call http::server_wait(sturdy)", "call http::server_wait(built)"},
		"fixture arity":           {"sample: config, built => ok", "sample: config => ok"},
		"opaque fixture value":    {"sample: config, built => ok", "sample: config, built => ok 7"},
		"fixture token mismatch":  {"sample: sturdy => ok", "sample: config => ok"},
		"missing bind emission":   {"http::ambiguous_route, http::bind_failed]", "http::ambiguous_route]"},
	} {
		t.Run(name, func(t *testing.T) {
			source := strings.Replace(base, replace.old, replace.new, 1)
			if source == base {
				t.Fatal("replacement anchor missing from server fixture")
			}
			if _, err := programFixture(t, map[string]string{"src/main.can": source}); err == nil {
				t.Fatalf("invalid server lifecycle accepted: %s", name)
			}
		})
	}
}
