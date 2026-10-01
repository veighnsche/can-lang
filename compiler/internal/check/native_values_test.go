package check

import (
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/source"
)

const nativeSurfaceFixture = `package app
    provides []
    uses [native, option]
fn void demo
    emits {}
    given
        str kind
    asserts
        sample: "text" => ok
    native::limits lim = native::limits(1, 8, 8, 64, 65536, 4, 4)
    native::session s = call native::open("bun", "observer", "run", lim)
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
