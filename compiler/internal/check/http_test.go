package check

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
)

func httpFixture(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("../../testdata/current/http/main.can")
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestHTTPFixtureAdmitsHandlersMountsAndSpecializations(t *testing.T) {
	p, err := programFixture(t, map[string]string{"src/main.can": httpFixture(t)})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.HTTPs) != 3 {
		t.Fatalf("expected three HTTP specializations, got %d", len(p.HTTPs))
	}
	seen := map[string]bool{}
	for key, special := range p.HTTPs {
		seen[special.Operation] = true
		if special.Contract == nil || special.Data == nil {
			t.Fatalf("missing HTTP contract for %s", key)
		}
		switch special.Operation {
		case httpRequestJSON, httpResponseJSON:
			if special.Schema.Root != special.Data.Identity() {
				t.Fatalf("missing JSON schema for %s", key)
			}
		case httpRequestForm:
			if special.Form.Root != special.Data.Identity() || len(special.Form.Fields) != 3 {
				t.Fatalf("missing form schema for %s: %+v", key, special.Form)
			}
		default:
			t.Fatalf("unexpected HTTP specialization %s", special.Operation)
		}
	}
	for _, op := range []string{httpRequestJSON, httpRequestForm, httpResponseJSON} {
		if !seen[op] {
			t.Fatalf("missing specialization for %s", op)
		}
	}
	scoped := 0
	var expression func(*ir.Expression)
	var invocation func(*ir.Invocation)
	var completion func(*ir.Completion)
	var block func(*ir.Block)
	expression = func(e *ir.Expression) {
		if e == nil {
			return
		}
		if e.Kind == ir.ScopeRequest {
			scoped++
		}
		for _, input := range e.Inputs {
			expression(input)
		}
		invocation(e.Invocation)
		if e.Match != nil {
			invocation(e.Match.Call)
			for i := range e.Match.Arms {
				completion(e.Match.Arms[i].Body)
				expression(e.Match.Arms[i].Value)
			}
		}
	}
	invocation = func(call *ir.Invocation) {
		if call == nil {
			return
		}
		for i := range call.Steps {
			step := &call.Steps[i]
			expression(step.Callee)
			expression(step.Native)
			for _, prepared := range step.Prepare {
				expression(prepared.Value)
			}
			for _, arg := range step.Arguments {
				expression(arg)
			}
		}
	}
	completion = func(c *ir.Completion) {
		if c == nil {
			return
		}
		expression(c.Value)
		invocation(c.Call)
		block(c.Block)
		if c.Match != nil {
			invocation(c.Match.Call)
			for i := range c.Match.Arms {
				completion(c.Match.Arms[i].Body)
				expression(c.Match.Arms[i].Value)
			}
		}
	}
	block = func(b *ir.Block) {
		if b == nil {
			return
		}
		for i := range b.Steps {
			expression(b.Steps[i].Value)
			invocation(b.Steps[i].Call)
		}
		completion(b.Terminal)
	}
	for _, assertion := range p.Assertions {
		if assertion.Actual != nil {
			block(assertion.Actual.Body)
		}
	}
	if scoped == 0 {
		t.Fatal("elided scope arguments left no harness scope in checked assertions")
	}
	routes := 0
	for _, fn := range p.Functions {
		if fn.Region == nil {
			continue
		}
		var match func(*ir.Match)
		var comp func(*ir.Completion)
		var blk func(*ir.Block)
		steps := func(call *ir.Invocation) {
			if call == nil {
				return
			}
			for i := range call.Steps {
				if routeOperation(call.Steps[i].Identity) {
					routes++
				}
			}
		}
		comp = func(c *ir.Completion) {
			if c == nil {
				return
			}
			steps(c.Call)
			blk(c.Block)
			match(c.Match)
		}
		match = func(m *ir.Match) {
			if m == nil {
				return
			}
			steps(m.Call)
			for i := range m.Arms {
				comp(m.Arms[i].Body)
			}
		}
		blk = func(b *ir.Block) {
			if b == nil {
				return
			}
			for i := range b.Steps {
				steps(b.Steps[i].Call)
			}
			comp(b.Terminal)
		}
		blk(fn.Region.Body)
	}
	if routes != 14 {
		t.Fatalf("expected fourteen mounted routes, got %d", routes)
	}
}

func TestHTTPRouteMountRefusals(t *testing.T) {
	original := httpFixture(t)
	mount := `call http::route_get("/a", callable handle)`
	for _, tc := range []struct{ name, replacement string }{
		{"dynamic path", `call http::route_get(path, callable handle)`},
		{"empty path", `call http::route_get("", callable handle)`},
		{"relative path", `call http::route_get("a", callable handle)`},
		{"double slash", `call http::route_get("//x", callable handle)`},
		{"capture segment", `call http::route_get("/x/:id", callable handle)`},
		{"query suffix", `call http::route_get("/x?q=1", callable handle)`},
		{"fragment suffix", `call http::route_get("/x#h", callable handle)`},
		{"wildcard", `call http::route_get("/x/*", callable handle)`},
		{"backslash", `call http::route_get("/x\\y", callable handle)`},
		{"truncated escape", `call http::route_get("/%", callable handle)`},
		{"invalid escape", `call http::route_get("/%zz", callable handle)`},
		{"invalid encoding", `call http::route_get("/%ff", callable handle)`},
		{"encoded control", `call http::route_get("/%00", callable handle)`},
		{"reserved prefix", `call http::route_get("/__can/assets/htmx-4.0.0.min.js", callable handle)`},
		{"reserved root", `call http::route_get("/__can", callable handle)`},
		{"encoded reserved", `call http::route_get("/%5F%5Fcan/x", callable handle)`},
		{"call result callback", `call http::route_get("/a", call handle_path())`},
		{"local callback", `call http::route_get("/a", action)`},
		{"fallible handler", `call http::route_get("/a", callable inspect)`},
		{"route reference", `callable route_get`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := original
			if tc.name == "route reference" {
				source = strings.Replace(original, `str tag = "t"`, "str tag = \"t\"\n    callable http::route (str, callable http::server_response (http::request) emits []) emits [http::invalid_route] maker = callable http::route_get", 1)
			} else {
				source = strings.Replace(original, mount, tc.replacement, 1)
			}
			if source == original {
				t.Fatal("invalid refusal fixture")
			}
			if _, err := programFixture(t, map[string]string{"src/main.can": source}); err == nil {
				t.Fatalf("accepted invalid mount: %s", tc.replacement)
			}
		})
	}
}

func TestHTTPRoutePathCorpusMatchesRuntime(t *testing.T) {
	data, err := os.ReadFile("../../../runtime/test/http-paths.json")
	if err != nil {
		t.Fatal(err)
	}
	var corpus struct {
		Accept []string `json:"accept"`
		Reject []string `json:"reject"`
	}
	if err := json.Unmarshal(data, &corpus); err != nil {
		t.Fatal(err)
	}
	if len(corpus.Accept) == 0 || len(corpus.Reject) == 0 {
		t.Fatal("empty shared path corpus")
	}
	for _, path := range corpus.Accept {
		if err := checkRoutePath(path); err != nil {
			t.Fatalf("checker rejects runtime-accepted path %q: %v", path, err)
		}
	}
	for _, path := range corpus.Reject {
		if err := checkRoutePath(path); err == nil {
			t.Fatalf("checker accepts runtime-rejected path %q", path)
		}
	}
	// Lone surrogates and invalid bytes cannot cross the JSON corpus; the
	// checker rejects them exactly like the runtime isWellFormed gate.
	for _, path := range []string{string([]byte{0x2f, 0xed, 0xa0, 0x80}), string([]byte{0x2f, 0xed, 0xb0, 0x80, 0x78}), "/\xff", "/a\xff\xfe"} {
		if err := checkRoutePath(path); err == nil {
			t.Fatalf("checker accepts ill-formed path %q", path)
		}
	}
}

func TestHTTPHandlerContractRefusals(t *testing.T) {
	header := "package app\n    provides []\n    uses [http]\n"
	main := "fn void main\n    emits []\n    given\n        str[] args\n    asserts\n        sample: [] => ok\n    ok\n"
	for _, tc := range []struct{ name, handler string }{
		{"wrong input", "fn http::server_response handle\n    emits []\n    given\n        str req\n    asserts\n        sample: \"x\" => ok\n    ok call http::response_text(call http::status_ok(), call http::empty_server_headers(), req)\n"},
		{"wrong result", "fn str handle\n    emits []\n    given\n        http::request req\n    asserts\n        sample: => ok \"x\"\n    ok \"x\"\n"},
		{"fallible handler", "fn http::server_response handle\n    emits [http::invalid_request]\n    given\n        http::request req\n    asserts\n        sample: => http::invalid_request(\"x\")\n    http::invalid_request(\"x\")\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mount := "fn bool mounted\n    emits [http::invalid_route]\n    asserts\n        sample: => ok true\n    match call http::route_get(\"/a\", callable handle)\n        http::invalid_route\n        ok http::route route => ok true\n"
			source := header + tc.handler + mount + main
			if _, err := programFixture(t, map[string]string{"src/main.can": source}); err == nil {
				t.Fatalf("accepted %s", tc.name)
			}
		})
	}
}

func TestHTTPSpecializationRefusals(t *testing.T) {
	original := httpFixture(t)
	for _, tc := range []struct{ name, old, replacement string }{
		{"opaque JSON body", "call http::request_json<item>(req, 1024)", "call http::request_json<http::request>(req, 1024)"},
		{"missing JSON argument", "call http::request_json<item>(req, 1024)", "call http::request_json(req, 1024)"},
		{"extra JSON argument", "call http::request_json<item>(req, 1024)", "call http::request_json<item, str>(req, 1024)"},
		{"opaque JSON response", "call http::response_json<item>(call http::status_ok(), call http::empty_server_headers(), item(\"a\", 1))", "call http::response_json<http::route>(call http::status_ok(), call http::empty_server_headers(), first)"},
		{"non-text form field", "record search_form\n    str query\n    str[] tags\n    option::value<str> note", "record search_form\n    str query\n    str[] tags\n    int note"},
		{"nested form field", "record search_form\n    str query\n    str[] tags\n    option::value<str> note", "record search_form\n    str query\n    str[] tags\n    item note"},
		{"non-string option", "record search_form\n    str query\n    str[] tags\n    option::value<str> note", "record search_form\n    str query\n    str[] tags\n    option::value<int> note"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := strings.Replace(original, tc.old, tc.replacement, 1)
			if source == original {
				t.Fatal("invalid refusal fixture")
			}
			if _, err := programFixture(t, map[string]string{"src/main.can": source}); err == nil {
				t.Fatalf("accepted invalid specialization: %s", tc.replacement)
			}
		})
	}
}

func TestHTTPAssertionScopeRefusals(t *testing.T) {
	original := httpFixture(t)
	for _, tc := range []struct{ name, old, replacement string }{
		{"extra scope argument", "method: => ok", "method: \"x\" => ok"},
		{"supplied scope value", "method: => ok", "method: req => ok"},
		{"valued opaque expectation", "method: => ok", "method: => ok call http::response_text(call http::status_ok(), call http::empty_server_headers(), \"GET\")"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := strings.Replace(original, tc.old, tc.replacement, 1)
			if source == original {
				t.Fatal("invalid refusal fixture")
			}
			if _, err := programFixture(t, map[string]string{"src/main.can": source}); err == nil {
				t.Fatalf("accepted invalid scope row: %s", tc.name)
			}
		})
	}
	helper := "package app\n    provides []\n    uses [http]\nfn http::request echo\n    emits []\n    given\n        http::request req\n    asserts\n        sample: => ok req\n    ok req\nfn void main\n    emits []\n    given\n        str[] args\n    asserts\n        sample: [] => ok\n    ok\n"
	if _, err := programFixture(t, map[string]string{"src/main.can": helper}); err == nil {
		t.Fatal("accepted scope-typed expected completion")
	}
	bare := "package app\n    provides []\n    uses [http]\nfn http::server_response handle\n    emits []\n    given\n        http::request req\n    asserts\n        sample: => ok\n    ok call http::response_text(call http::status_ok(), call http::empty_server_headers(), \"ok\")\nfn void main\n    emits []\n    given\n        str[] args\n    asserts\n        sample: [] => ok\n    ok\n"
	if _, err := programFixture(t, map[string]string{"src/main.can": bare}); err != nil {
		t.Fatalf("rejected bare ok for opaque results: %v", err)
	}
	valued := "package app\n    provides []\n    uses []\nfn str name\n    emits []\n    asserts\n        sample: => ok\n    ok \"x\"\nfn void main\n    emits []\n    given\n        str[] args\n    asserts\n        sample: [] => ok\n    ok\n"
	if _, err := programFixture(t, map[string]string{"src/main.can": valued}); err == nil {
		t.Fatal("accepted bare ok for data results")
	}
	body := "package app\n    provides []\n    uses [http]\nfn http::server_response handle\n    emits []\n    given\n        http::request req\n    asserts\n        sample: => ok\n    match call http::request_method(req)\n        ok str method => ok\nfn void main\n    emits []\n    given\n        str[] args\n    asserts\n        sample: [] => ok\n    ok\n"
	if _, err := programFixture(t, map[string]string{"src/main.can": body}); err == nil {
		t.Fatal("accepted bare ok in a nonvoid body")
	}
}
