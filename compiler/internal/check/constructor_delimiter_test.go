package check

import (
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"strings"
	"testing"
)

// After resolution, errors require {} and records require (). The valid
// program covers local, imported, catalogue, generic, and expected-variant
// constructions using only corpus-proven shapes; each negative mutates one
// delimiter and must fail with the delimiter diagnostic at a located span.
func TestConstructorDelimiterRequiresResolvedKind(t *testing.T) {
	leaf := `package leaf
    provides [fused, crate]
    uses []
error fused{str key}
record crate
    str label
`
	main := `package app
    provides []
    uses [codec, leaf]
error missing{str key}
record wrapper
    str label
record held<item>
    item value
variant outcome
    missing
    codec::invalid_data
fn int load
    emits {missing}
    asserts
        sample: => missing{"x"}
    missing{"x"}
fn wrapper fetch
    emits {}
    asserts
        sample: => ok wrapper("stored")
    ok wrapper("stored")
fn held<missing> store
    emits {}
    asserts
        sample: => ok held<missing>(missing{"x"})
    ok held<missing>(missing{"x"})
fn int guarded
    emits {}
    asserts
        sample: => ok 1
    match call load()
        missing as failure => do
            outcome cause = missing{failure.key}
            int one = 1
            ok one
        ok int value => ok value
fn int aggregate
    emits {all_failed<outcome>}
    asserts
        sample: => all_failed<outcome>{[missing{"x"}]}
    all_failed<outcome>{[missing{"x"}]}
fn int decode
    emits {codec::invalid_data}
    asserts
        sample: => codec::invalid_data{"a", "type"}
    codec::invalid_data{"a", "type"}
fn int convert
    emits {leaf::fused}
    asserts
        sample: => leaf::fused{"x"}
    leaf::fused{"x"}
fn leaf::crate carry
    emits {}
    asserts
        sample: => ok leaf::crate("x")
    ok leaf::crate("x")
fn void main
    emits {}
    given
        str[] arguments
    asserts
        empty: [] => ok
    ok
`
	files := func(text string) map[string]string {
		return map[string]string{"src/app/main.can": text, "src/leaf/leaf.can": leaf}
	}
	registry := `{"active":["app::missing","leaf::fused"],"retired":[]}`
	if _, err := programFixtureRegistry(t, files(main), registry); err != nil {
		t.Fatalf("valid delimiters rejected: %v", err)
	}
	cases := []struct {
		name string
		old  string
		next string
		want string
	}{
		{"local error parens", "\n    missing{\"x\"}\n", "\n    missing(\"x\")\n", "requires brace construction"},
		{"local record braces", "\n    ok wrapper(\"stored\")\n", "\n    ok wrapper{\"stored\"}\n", "requires parenthesis construction"},
		{"generic record braces", "\n    ok held<missing>(missing{\"x\"})\n", "\n    ok held<missing>{missing{\"x\"}}\n", "requires parenthesis construction"},
		{"stored error parens", "\n    ok held<missing>(missing{\"x\"})\n", "\n    ok held<missing>(missing(\"x\"))\n", "requires brace construction"},
		{"variant expectation parens", "outcome cause = missing{failure.key}", "outcome cause = missing(failure.key)", "requires brace construction"},
		{"generic error parens", "\n    all_failed<outcome>{[missing{\"x\"}]}\n", "\n    all_failed<outcome>([missing{\"x\"}])\n", "requires brace construction"},
		{"catalogue error parens", "\n    codec::invalid_data{\"a\", \"type\"}\n", "\n    codec::invalid_data(\"a\", \"type\")\n", "requires brace construction"},
		{"imported error parens", "\n    leaf::fused{\"x\"}\n", "\n    leaf::fused(\"x\")\n", "requires brace construction"},
		{"imported record braces", "\n    ok leaf::crate(\"x\")\n", "\n    ok leaf::crate{\"x\"}\n", "requires parenthesis construction"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(main, tc.old) {
				t.Fatalf("mutation needle missing: %q", tc.old)
			}
			bad := strings.Replace(main, tc.old, tc.next, 1)
			if bad == main {
				t.Fatalf("mutation matched nothing: %q", tc.old)
			}
			_, err := programFixtureRegistry(t, files(bad), registry)
			if err == nil {
				t.Fatalf("wrong delimiter admitted: %s", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("diagnostic %q lacks %q", err.Error(), tc.want)
			}
			if located, ok := source.AsLocated(err); !ok || located.File == "" || located.Span.End <= located.Span.Start {
				t.Fatalf("diagnostic lacks a structured source span: %q", err.Error())
			}
		})
	}
}
