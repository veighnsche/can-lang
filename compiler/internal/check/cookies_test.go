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
	cookieParse      = "can.std.cookie@1::parse"
	cookieGet        = "can.std.cookie@1::get"
	cookieMake       = "can.std.cookie@1::make"
	cookieSerialize  = "can.std.cookie@1::serialize"
	cookieExpire     = "can.std.cookie@1::expire"
	csrfGenerate     = "can.std.csrf@1::generate"
	csrfVerify       = "can.std.csrf@1::verify"
	cookieHandle     = "can.std.cookie@1::cookie"
	cookieAttributes = "can.std.cookie@1::attributes"
)

func cookieFixture(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("../../testdata/current/cookies/session.can")
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func cookieDeclarations(t *testing.T, list []*types.Type) []string {
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

func TestCookiesFixtureAdmitsOperationContracts(t *testing.T) {
	p, err := programFixture(t, map[string]string{"src/main.can": cookieFixture(t)})
	if err != nil {
		t.Fatal(err)
	}
	contracts := map[string]struct {
		inputs []string
		result string
		errors []string
	}{
		cookieParse: {inputs: []string{"str"}, result: "can.std.cookie@1::collection"},
		cookieGet:   {inputs: []string{"can.std.cookie@1::collection", "str"}, result: "can.std.option@1::value"},
		cookieMake: {
			inputs: []string{"str", "str", cookieAttributes},
			result: cookieHandle,
			errors: []string{"can.std.cookie@1::invalid_cookie"},
		},
		cookieSerialize: {inputs: []string{cookieHandle}, result: "str"},
		cookieExpire: {
			inputs: []string{"str", "str", "can.std.option@1::value"},
			result: cookieHandle,
			errors: []string{"can.std.cookie@1::invalid_cookie"},
		},
		csrfGenerate: {
			inputs: []string{"str", "str", "int"},
			result: "str",
			errors: []string{"can.std.csrf@1::invalid_config"},
		},
		csrfVerify: {
			inputs: []string{"str", "str", "str", "int"},
			result: "bool",
			errors: []string{"can.std.csrf@1::invalid_config"},
		},
	}
	for identity, want := range contracts {
		contract := p.Intrinsics[identity]
		if contract == nil {
			t.Fatalf("missing cookies intrinsic %s", identity)
		}
		got := cookieDeclarations(t, contract.Inputs())
		if strings.Join(got, ",") != strings.Join(want.inputs, ",") {
			t.Fatalf("%s inputs = %v, want %v", identity, got, want.inputs)
		}
		if contract.Result() == nil {
			t.Fatalf("missing result for %s", identity)
		}
		if result := contract.Result().Declaration(); result != want.result {
			t.Fatalf("%s result = %s, want %s", identity, result, want.result)
		}
		errors := cookieDeclarations(t, contract.Errors())
		sort.Strings(errors)
		sorted := append([]string(nil), want.errors...)
		sort.Strings(sorted)
		if strings.Join(errors, ",") != strings.Join(sorted, ",") {
			t.Fatalf("%s errors = %v, want %v", identity, errors, want.errors)
		}
	}
	// The validated cookie crosses as an opaque handle while attributes
	// stay a constructible data record.
	seen := map[string]bool{}
	for _, typ := range p.Model.Types() {
		switch typ.Declaration() {
		case cookieHandle:
			seen["cookie"] = typ.Kind() == types.Opaque
		case cookieAttributes:
			seen["attributes"] = typ.Kind() == types.Record
		}
	}
	for name, ok := range seen {
		if !ok {
			t.Fatalf("cookie %s has the wrong kind", name)
		}
	}
	if len(seen) != 2 {
		t.Fatalf("cookie handle types missing: %v", seen)
	}
}

func TestCookiesSuppliedFixturesAttachToMint(t *testing.T) {
	p, err := programFixture(t, map[string]string{"src/main.can": cookieFixture(t)})
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
	// Only the supplied and scoped boundaries carry fixtures: the mint
	// stands in for random generation while the login transcript drives
	// headers and query. Real cookie and CSRF verification run natively.
	mint := 0
	for identity, counts := range rows {
		for _, count := range counts {
			switch {
			case strings.Contains(identity, "csrf") && strings.Contains(identity, "generate"):
				mint += count
			case count >= 1 && count <= 2:
			default:
				t.Fatalf("fixture table %q carries %d rows", identity, count)
			}
		}
	}
	if mint != 1 {
		t.Fatalf("mint fixture missing: %v", rows)
	}
}
