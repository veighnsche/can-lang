package syntax

import (
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/source"
)

// T09 (Gate 1/2 core contract integration): the T02-T08 surface syntax
// converges in one file — instance-qualified aliased imports, an unnumbered
// error declaration, an owner record, a scenario declaration with a link
// row, a variant matched with explicit bind captures — and round-trips
// through the formatter byte-for-byte.
const coreIntegratedSource = `package app
    provides [email, make_email, checkout, read]
    uses [left::model as first, codec]

owner record email
    str address

error failed(str reason)

scenario checkout

fn email make_email
    emits []
    given
        str address
    asserts
        sample: "a@b" => ok email("a@b")
    ok email(address)

fn str read
    emits [first::failed]
    asserts
        customer:  => ok "fixture" link checkout
    match call first::risky(0)
        first::failed as failed => ok failed.reason
        ok int got => ok "done"

fn int area_units
    emits []
    given
        first::shape value
    asserts
        round: first::circle(3) => ok 3
    match value
        first::circle(bind radius) => ok radius
        first::rectangle(bind width, _) => ok width
`

func TestCoreIntegratedSurfaceRoundTrip(t *testing.T) {
	file, err := source.New("core.can", coreIntegratedSource)
	if err != nil {
		t.Fatal(err)
	}
	result := Parse(file)
	if !result.OK() {
		t.Fatalf("integrated surface rejected: %s", result.Diagnostics[0].Format(file))
	}
	formatted := Format(result.File)
	if formatted != coreIntegratedSource {
		t.Fatalf("integrated surface moved under Format:\n%s", formatted)
	}
	reparsed, err := source.New("core.can", formatted)
	if err != nil {
		t.Fatal(err)
	}
	again := Parse(reparsed)
	if !again.OK() {
		t.Fatalf("formatted surface does not reparse: %s", again.Diagnostics[0].Format(reparsed))
	}
	if second := Format(again.File); second != coreIntegratedSource {
		t.Fatalf("integrated surface unstable under Format:\n%s", second)
	}
	for _, want := range []string{
		"uses [left::model as first, codec]",
		"owner record email",
		"error failed(str reason)",
		"scenario checkout",
		"link checkout",
		"first::failed as failed",
		"first::circle(bind radius)",
	} {
		if !strings.Contains(formatted, want) {
			t.Fatalf("formatted surface omits %q:\n%s", want, formatted)
		}
	}
}
