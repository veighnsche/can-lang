package check

import (
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/source"
)

const nativeSurfaceFixture = `package app
    provides []
    uses [native, option, test]
fn void demo
    emits {test::invalid_grant}
    given
        str kind
    asserts
        sample: "text" => ok
    match call test::grant_admit("ng1-0123456789abcdef0123456789abcdef")
        test::invalid_grant
        ok test::owner o => do
            native::limits lim = native::limits(1, 8, 8, 64, 65536, 4, 4)
            native::session s = call native::open(o, "bun", "observer", "run", lim)
            native::inert_literal lit = native::inert_literal("text", option::none(), option::none(), option::some("hi"), option::none(), option::none())
            native::value_handle h = call native::make(s, "text", lit)
            native::observe_bounds b = native::observe_bounds(64, 65536)
            native::value_handle[] hs = [h]
            native::inert_facts f = call native::observe(s, hs, "lexeme", b)
            native::deadline d = native::deadline("wall-utc", 5000)
            native::close_receipt r = call native::close(s, d)
            ok
` + programMain + "    ok\n"

func TestNativeSurfaceAdmitsKindLiterals(t *testing.T) {
	if _, err := programFixture(t, map[string]string{"src/main.can": nativeSurfaceFixture}); err != nil {
		t.Fatalf("valid native surface rejected: %v", err)
	}
}

func TestNativeStaticAdmissionRefusals(t *testing.T) {
	for _, tc := range []struct{ name, from, to, want string }{
		{"bad make kind", `call native::make(s, "text", lit)`, `call native::make(s, "ordered-entires", lit)`, `native make kind "ordered-entires" is not admitted`},
		{"dynamic make kind", `call native::make(s, "text", lit)`, `call native::make(s, kind, lit)`, `native make kind must be a static literal`},
		{"bad observe kind", `call native::observe(s, hs, "lexeme", b)`, `call native::observe(s, hs, "bytes", b)`, `native observe kind "bytes" is not admitted`},
		{"dynamic observe kind", `call native::observe(s, hs, "lexeme", b)`, `call native::observe(s, hs, kind, b)`, `native observe kind must be a static literal`},
		{"empty make kind", `call native::make(s, "text", lit)`, `call native::make(s, "", lit)`, `native make kind "" is not admitted`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			text := strings.Replace(nativeSurfaceFixture, tc.from, tc.to, 1)
			if text == nativeSurfaceFixture {
				t.Fatal("invalid refusal fixture")
			}
			_, err := programFixture(t, map[string]string{"src/main.can": text})
			if err == nil {
				t.Fatalf("accepted invalid native call: %s", tc.to)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("native diagnostic omits %q: %v", tc.want, err)
			}
		})
	}
}

func nativeElisionFixture() string {
	return "package app\n    provides []\n    uses [native]\nfn native::release_facts drop\n    emits {}\n    given\n        native::session s\n        native::gate g\n    asserts\n        drained: => ok native::release_facts(true, 0, true)\n    match call native::release(g)\n        when\n            drained: g => ok native::release_facts(true, 0, true)\n        ok native::release_facts got => ok got\n" + programMain + "    ok\n"
}

func TestNativeAssertionScopeElision(t *testing.T) {
	if _, err := programFixture(t, map[string]string{"src/main.can": nativeElisionFixture()}); err != nil {
		t.Fatalf("rejected elided session/gate rows: %v", err)
	}
	original := nativeElisionFixture()
	for _, tc := range []struct{ name, old, replacement string }{
		{"extra scope argument", "drained: => ok", "drained: \"x\" => ok"},
		{"supplied scope value", "drained: => ok", "drained: g => ok"},
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
}

func TestNativeDiagnosticSpans(t *testing.T) {
	for _, tc := range []struct{ name, from, to, want string }{
		{"make kind", `call native::make(s, "text", lit)`, `call native::make(s, "ordered-entires", lit)`, `"ordered-entires"`},
		{"observe kind", `call native::observe(s, hs, "lexeme", b)`, `call native::observe(s, hs, "bytes", b)`, `"bytes"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			text := strings.Replace(nativeSurfaceFixture, tc.from, tc.to, 1)
			if text == nativeSurfaceFixture {
				t.Fatal("invalid span fixture")
			}
			_, err := programFixture(t, map[string]string{"src/main.can": text})
			if err == nil {
				t.Fatalf("admitted invalid native literal: %s", tc.to)
			}
			located, ok := source.AsLocated(err)
			if !ok {
				t.Fatalf("native failure lost its span: %v", err)
			}
			if got := text[located.Span.Start:located.Span.End]; got != tc.want {
				t.Fatalf("native span covers %q, want %q", got, tc.want)
			}
		})
	}
}
