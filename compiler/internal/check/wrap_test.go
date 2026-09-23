package check

import (
	"os"
	"strings"
	"testing"
)

const wrapHeader = "package app\n    provides []\n    uses [http, codec]\nrecord receipt\n    int count\nconnection service\n    endpoint \"http://localhost:1\"\n    timeout_ms 1000\n"
const wrapLoad = "fetch receipt load from service\n    emits [http::request_failed]\n    asserts\n        decoded: => ok receipt(7)\n            using raw \"fixtures/load.json\"\n    get \"/load\"\n"
const wrapChild = `wrap cached from load
    emits calculated
    asserts
        absent: => ok receipt(0)
            using failure native http::status_error(404, [])
        busy: => http::request_failed(http::status_error(429, []))
            using failure native http::status_error(429, [])
    handles native
        http::status_error as failed => match failed.status
            404 => ok receipt(0)
            _ => inherit
`

func wrapProgram(t *testing.T, source string) *Program {
	t.Helper()
	p, err := programFixture(t, withNativeRaw(map[string]string{"src/main.can": source}, "load"))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func wrapNative(t *testing.T, p *Program, name string) *NativeDeclaration {
	t.Helper()
	for _, native := range p.Natives {
		if native.Symbol.Name == name {
			return native
		}
	}
	t.Fatalf("native %s missing", name)
	return nil
}

func TestWrapperCalculatedBound(t *testing.T) {
	p := wrapProgram(t, wrapHeader+wrapLoad+wrapChild+programMain+"    ok\n")
	cached := wrapNative(t, p, "cached")
	if cached.Signature == nil || cached.Wrapper == nil {
		t.Fatal("wrapper publishes no calculated signature")
	}
	var bound []string
	for _, typ := range cached.Signature.Errors() {
		bound = append(bound, typ.Declaration())
	}
	if len(bound) != 1 || bound[0] != "can.std.http@1::request_failed" {
		t.Fatalf("calculated bound is %v", bound)
	}
	// Six raw keys (1100/1102/1103/1104/1105/1110, no auth): one local rule
	// plus five defaults, all normalizing to the same member.
	if len(cached.Wrapper.Rules) != 6 {
		t.Fatalf("effective rules cover %d keys", len(cached.Wrapper.Rules))
	}
	var rule *WrapperRule
	for i := range cached.Wrapper.Rules {
		if cached.Wrapper.Rules[i].Key.Origin == WrapperNative && cached.Wrapper.Rules[i].Key.Decl == "can.std.http@1::status_error" {
			rule = &cached.Wrapper.Rules[i]
		}
	}
	if rule == nil || rule.Region == nil || rule.Default || len(rule.Escapes) != 1 {
		t.Fatal("status rule lost its region or escape")
	}
	if len(rule.From) != 1 || len(rule.From[0].Chain) != 1 || rule.From[0].Chain[0] != "default" || rule.From[0].Handler != cached.Symbol.ID {
		t.Fatalf("status provenance is %+v", rule.From)
	}
	if len(cached.Wrapper.Local) != 1 {
		t.Fatalf("local keys are %v", cached.Wrapper.Local)
	}
}

func TestWrapperThreeGenerationOverride(t *testing.T) {
	grandchild := `wrap settled from cached
    emits calculated
    asserts
        gone: => ok receipt(0)
            using failure native http::status_error(404, [])
        slow: => http::request_failed(http::status_error(503, []))
            using failure native http::status_error(503, [])
    handles native
        http::status_error as failed => match failed.status
            404 => ok receipt(0)
            429 => inherit
            _ => http::request_failed(failed)
`
	p := wrapProgram(t, wrapHeader+wrapLoad+wrapChild+grandchild+programMain+"    ok\n")
	settled := wrapNative(t, p, "settled")
	cached := wrapNative(t, p, "cached")
	if len(settled.Signature.Errors()) != 1 {
		t.Fatal("grandchild bound lost its member")
	}
	var rule *WrapperRule
	for i := range settled.Wrapper.Rules {
		if settled.Wrapper.Rules[i].Key.Origin == WrapperNative && settled.Wrapper.Rules[i].Key.Decl == "can.std.http@1::status_error" {
			rule = &settled.Wrapper.Rules[i]
		}
	}
	if rule == nil || len(rule.From) != 1 {
		t.Fatalf("grandchild rule provenance is %+v", rule)
	}
	chain := rule.From[0].Chain
	if len(chain) != 2 || chain[0] != cached.Symbol.ID || chain[1] != "default" || rule.From[0].Handler != settled.Symbol.ID {
		t.Fatalf("override trace is %+v", rule.From)
	}
}

func TestWrapperObligationDiagnostic(t *testing.T) {
	match := wrapHeader + wrapLoad + wrapChild + `fn receipt fetch_cached
    emits []
    asserts
        sample: => ok receipt(0)
    match call cached()
        ok => ok receipt(0)
` + programMain + "    ok\n"
	_, err := programFixture(t, withNativeRaw(map[string]string{"src/main.can": match}, "load"))
	if err == nil {
		t.Fatal("caller without the calculated member admitted")
	}
	for _, want := range []string{"missing completion arm", "wrapper can.project.root/app::cached", "native key can.std.http@1::status_error", "via can.project.root/app::cached", "inherit default"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("obligation diagnostic omits %q: %v", want, err)
		}
	}
	relay := wrapHeader + wrapLoad + wrapChild + `fn receipt fetch_cached
    emits []
    asserts
        sample: => ok receipt(0)
    relay call cached()
` + programMain + "    ok\n"
	_, err = programFixture(t, withNativeRaw(map[string]string{"src/main.can": relay}, "load"))
	if err == nil {
		t.Fatal("relay without the calculated member admitted")
	}
	for _, want := range []string{"undeclared escaping domain error", "wrapper can.project.root/app::cached", "native key can.std.http@1::status_error"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("relay diagnostic omits %q: %v", want, err)
		}
	}
}

func TestWrapTestdataContracts(t *testing.T) {
	source, err := os.ReadFile("../../testdata/current/wrap/main.can")
	if err != nil {
		t.Fatal(err)
	}
	p, err := programFixture(t, testdataFixtures(t, map[string]string{"src/main.can": string(source)}, "wrap"))
	if err != nil {
		t.Fatal(err)
	}
	declarations := func(native *NativeDeclaration) []string {
		var out []string
		for _, typ := range native.Signature.Errors() {
			out = append(out, typ.Declaration())
		}
		return out
	}
	has := func(set []string, identity string) bool {
		for _, member := range set {
			if member == identity {
				return true
			}
		}
		return false
	}
	failed := "can.std.http@1::request_failed"
	settled := wrapNative(t, p, "settled")
	if bound := declarations(settled); len(bound) != 1 || bound[0] != failed {
		t.Fatalf("settled bound keeps more than the surviving native member: %v", bound)
	}
	var status *WrapperRule
	for i := range settled.Wrapper.Rules {
		rule := &settled.Wrapper.Rules[i]
		if rule.Key.Origin == WrapperNative && rule.Key.Decl == "can.std.http@1::status_error" {
			status = rule
		}
	}
	if status == nil || len(status.From) != 1 {
		t.Fatalf("settled status rule missing: %+v", status)
	}
	cached := wrapNative(t, p, "cached")
	chain := status.From[0].Chain
	if len(chain) != 2 || chain[0] != cached.Symbol.ID || chain[1] != "default" || status.From[0].Handler != settled.Symbol.ID {
		t.Fatalf("three-generation override trace is %+v", status.From)
	}
	guarded := wrapNative(t, p, "guarded")
	assess := wrapNative(t, p, "assess")
	bound := declarations(guarded)
	// The fully replaced invalid_answer disappears; authored passthrough and
	// the surviving normalized member stay; standard faults never enter.
	for _, want := range []string{failed, "can.std.ai@1::invalid_question", "can.std.codec@1::invalid_data", "can.std.io@1::write_failed"} {
		if !has(bound, want) {
			t.Fatalf("guarded bound %v omits %s", bound, want)
		}
	}
	if has(bound, "can.std.ai@1::invalid_answer") || len(bound) != 4 {
		t.Fatalf("guarded bound keeps the replaced answer key: %v", bound)
	}
	if len(guarded.Wrapper.Local) != 2 {
		t.Fatalf("guarded local keys are %v", guarded.Wrapper.Local)
	}
	var nativeCodec, emittedCodec bool
	for _, rule := range guarded.Wrapper.Rules {
		if rule.Key.Decl != "can.std.codec@1::invalid_data" {
			continue
		}
		if rule.Key.Origin == WrapperNative {
			nativeCodec = true
		}
		if rule.Key.Origin == WrapperEmitted {
			emittedCodec = true
		}
	}
	if !nativeCodec || !emittedCodec {
		t.Fatal("same-name native decoder and authored codec keys do not both dispatch")
	}
	if !guarded.Descriptor.Grouped || guarded.Descriptor.State != 2 || len(guarded.Descriptor.Names) != len(assess.Descriptor.Names) {
		t.Fatalf("judge wrapper drops grouped state: %+v", guarded.Descriptor)
	}
	for i, name := range assess.Descriptor.Names {
		if guarded.Descriptor.Names[i] != name {
			t.Fatalf("judge wrapper renames inherited input %s", name)
		}
	}
	if guarded.Connection != assess.Connection {
		t.Fatal("judge wrapper drops its inherited connection")
	}
	found := false
	for _, fn := range p.Functions {
		if fn.Symbol.Name == "via_reference" {
			found = true
		}
	}
	if !found {
		t.Fatal("fetch wrapper callable consumer did not check")
	}
}

func TestWrapperJudgeStateGroup(t *testing.T) {
	base := nativeHeader + nativeClassifier + nativeQuestion + nativeJudge + `wrap guarded from assess
    emits calculated
    asserts
        sample: ("x") => ok true
            using failure emitted ai::invalid_answer("q", "r")
    handles emitted
        ai::invalid_answer => ok true
`
	flat := base + `fn bool caller
    emits [http::request_failed, ai::invalid_question, ai::invalid_answer]
    asserts
        sample: => ok true
    relay call guarded("x")
` + programMain + "    ok\n"
	if _, err := programFixture(t, withNativeRaw(map[string]string{"src/main.can": flat}, "assess")); err == nil || !strings.Contains(err.Error(), "requires its final state argument group") {
		t.Fatalf("flattened judge-wrapper call admitted: %v", err)
	}
	reference := base + `fn bool caller
    emits []
    asserts
        sample: => ok true
    callable bool (str) emits [http::request_failed, ai::invalid_question, ai::invalid_answer] action = callable guarded
    relay call action("x")
` + programMain + "    ok\n"
	if _, err := programFixture(t, withNativeRaw(map[string]string{"src/main.can": reference}, "assess")); err == nil || !strings.Contains(err.Error(), "judge wrapper cannot be an ordinary callable reference") {
		t.Fatalf("judge-wrapper callable reference admitted: %v", err)
	}
}

func TestWrapperRejects(t *testing.T) {
	fullHeader := "package app\n    provides []\n    uses [http, codec, ai, llm]\nrecord receipt\n    int count\nconnection service\n    endpoint \"http://localhost:1\"\n    timeout_ms 1000\n" + nativeClassifier + nativeGenerator
	cases := map[string]struct {
		source string
		want   string
	}{
		"question base": {
			fullHeader + nativeQuestion + "wrap cached from question\n    emits calculated\n    asserts\n        sample: \"x\", () => ok true\n            using failure native http::status_error(404, [])\n    handles native\n        http::status_error => ok true\n" + programMain + "    ok\n",
			"fetch, judge or wrapper base",
		},
		"llm base": {
			fullHeader + "llm str draft from generator\n    emits [http::request_failed]\n    asserts\n        sample: () => ok \"x\"\n            using raw \"fixtures/draft.json\"\n    asks \"Draft\"\nwrap polished from draft\n    emits calculated\n    asserts\n        sample: () => ok \"x\"\n            using failure native http::status_error(404, [])\n    handles native\n        http::status_error => ok \"x\"\n" + programMain + "    ok\n",
			"fetch, judge or wrapper base",
		},
		"impossible native key": {
			wrapHeader + wrapLoad + "wrap cached from load\n    emits calculated\n    asserts\n        sample: => ok receipt(0)\n            using failure native http::credentials_missing(\"TOKEN\")\n    handles native\n        http::credentials_missing => ok receipt(0)\n" + programMain + "    ok\n",
			"impossible native key",
		},
		"duplicate key": {
			wrapHeader + wrapLoad + "wrap cached from load\n    emits calculated\n    asserts\n        sample: => ok receipt(0)\n            using failure native http::status_error(404, [])\n    handles native\n        http::status_error => ok receipt(0)\n        http::status_error => ok receipt(0)\n" + programMain + "    ok\n",
			"duplicate policy key",
		},
		"function base": {
			wrapHeader + wrapLoad + "fn int helper\n    emits []\n    asserts\n        sample: => ok 1\n    ok 1\nwrap cached from helper\n    emits calculated\n    asserts\n        sample: => ok receipt(0)\n            using failure native http::status_error(404, [])\n    handles native\n        http::status_error => ok receipt(0)\n" + programMain + "    ok\n",
			"fetch, judge or wrapper base",
		},
		"self base": {
			wrapHeader + wrapLoad + "wrap cached from cached\n    emits calculated\n    asserts\n        sample: => ok receipt(0)\n            using failure native http::status_error(404, [])\n    handles native\n        http::status_error => ok receipt(0)\n" + programMain + "    ok\n",
			"cycle",
		},
		"mutual base": {
			wrapHeader + wrapLoad + "wrap first from second\n    emits calculated\n    asserts\n        sample: => ok receipt(0)\n            using failure native http::status_error(404, [])\n    handles native\n        http::status_error => ok receipt(0)\nwrap second from first\n    emits calculated\n    asserts\n        sample: => ok receipt(0)\n            using failure native http::status_error(404, [])\n    handles native\n        http::status_error => ok receipt(0)\n" + programMain + "    ok\n",
			"cycle",
		},
		"inherit outside handler": {
			wrapHeader + wrapLoad + "fn int helper\n    emits []\n    asserts\n        sample: => ok 1\n    inherit\n" + programMain + "    ok\n",
			"inherit is only admitted in a wrapper policy handler",
		},
		"bad transport phase": {
			wrapHeader + wrapLoad + "wrap cached from load\n    emits calculated\n    asserts\n        sample: => ok receipt(0)\n            using failure native http::transport_failed(\"bogus\")\n    handles native\n        http::transport_failed => ok receipt(0)\n" + programMain + "    ok\n",
			"unknown transport phase",
		},
		"cross-origin injection": {
			wrapHeader + wrapLoad + "wrap cached from load\n    emits calculated\n    asserts\n        sample: => ok receipt(0)\n            using failure emitted http::status_error(404, [])\n    handles native\n        http::status_error => ok receipt(0)\n" + programMain + "    ok\n",
			"not a domain obligation",
		},
		"consumer injection": {
			wrapHeader + wrapLoad + "fn int helper\n    emits []\n    asserts\n        sample: => ok 1\n            using failure native http::status_error(404, [])\n    ok 1\n" + programMain + "    ok\n",
			"confined to attached wrapper assertions",
		},
		"missing key coverage": {
			wrapHeader + wrapLoad + "wrap cached from load\n    emits calculated\n    asserts\n        sample: => ok receipt(0)\n            using failure native http::timeout(1000)\n    handles native\n        http::status_error => ok receipt(0)\n" + programMain + "    ok\n",
			"selecting local native key",
		},
		"self call cycle": {
			wrapHeader + wrapLoad + "wrap cached from load\n    emits calculated\n    asserts\n        sample: => ok receipt(0)\n            using failure native http::status_error(404, [])\n    handles native\n        http::status_error => relay call cached()\n" + programMain + "    ok\n",
			"calculated-bound dependency cycle",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := programFixture(t, withNativeRaw(map[string]string{"src/main.can": tc.source}, "load", "draft"))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q, got %v", tc.want, err)
			}
		})
	}
}
