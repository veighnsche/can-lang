package emit

import (
	"regexp"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/check"
)

// browserUP13QueryFixture exercises query_parameter against the checked
// option result: the adapter brands some/none with the sealed concrete
// leaf identities below, and reports malformed/duplicate/oversized
// queries through the invalid_query arm.
const browserUP13QueryFixture = `package app
    provides []
    uses [browser, option]
fn void show_boot_notice
    emits []
    given
        str message
    asserts
        sample: "hi" => ok
    ok
fn void boot
    emits []
    given
        str selected
    asserts
        sample: "inv-1" => ok
    ok
fn void main
    emits []
    asserts
        empty: => ok
    match call browser::query_parameter("invoice")
        browser::invalid_query => match call show_boot_notice("Invalid invoice link")
            ok => ok
        ok option::value<str> selected => match selected
            option::none => match call show_boot_notice("Choose an invoice")
                ok => ok
            option::some => match call boot(selected.value)
                ok => ok
`

func browserUP13OptionIDs(t *testing.T, program *check.Program) (some, none string) {
	t.Helper()
	intrinsic := program.Intrinsics["can.std.browser@1::query_parameter"]
	if intrinsic == nil {
		t.Fatal("query_parameter intrinsic missing")
	}
	for _, leaf := range intrinsic.Result().Leaves() {
		switch leaf.Declaration() {
		case "can.std.option@1::some":
			some = leaf.Identity()
		case "can.std.option@1::none":
			none = leaf.Identity()
		}
	}
	if some == "" || none == "" || some == none {
		t.Fatalf("query option leaves not sealed: some %q none %q", some, none)
	}
	return some, none
}

func TestBrowserSealsQueryOptionIdentities(t *testing.T) {
	program := browserUP11Program(t, map[string]string{"src/main.can": browserUP13QueryFixture})
	some, none := browserUP13OptionIDs(t, program)
	artifacts := browserUP11Artifacts(t, program)
	state := stateText(t, artifacts)
	for _, want := range []string{
		`invalidQuery:"`,
		`some:"` + some + `"`,
		`none:"` + none + `"`,
	} {
		if !strings.Contains(state, want) {
			t.Fatalf("browser state lacks %s:\n%s", want, state)
		}
	}
	for _, empty := range []string{`invalidQuery:""`, `some:""`, `none:""`} {
		if strings.Contains(state, empty) {
			t.Fatalf("browser state carries empty identity %s", empty)
		}
	}
	authored := browserUP11Authored(t, artifacts)
	if !strings.Contains(authored, "$canBrowser.queryParameter(") {
		t.Fatalf("browser authored lacks the query call:\n%s", authored)
	}
	// The UP11 call interface: key plus the trailing explicit owner
	// context, with no spliced metadata. The adapter ignores the
	// trailing context and reads the ambient location.
	matched, err := regexp.MatchString(`\$canBrowser\.queryParameter\(\$can\w+, \$canCtx\)`, authored)
	if err != nil || !matched {
		t.Fatalf("browser query call breaks the (key, $canCtx) shape:\n%s", authored)
	}
}

func TestBrowserSealsQueryOptionIdentitiesBun(t *testing.T) {
	program := actionEmitProgram(t, map[string]string{
		"src/main.can": `package app
    provides []
    uses [browser, option]
fn void show_boot_notice
    emits []
    given
        str message
    asserts
        sample: "hi" => ok
    ok
fn void boot
    emits []
    given
        str selected
    asserts
        sample: "inv-1" => ok
    ok
fn void main
    emits []
    given
        str[] arguments
    asserts
        empty: [] => ok
    match call browser::query_parameter("invoice")
        browser::invalid_query => match call show_boot_notice("Invalid invoice link")
            ok => ok
        ok option::value<str> selected => match selected
            option::none => match call show_boot_notice("Choose an invoice")
                ok => ok
            option::some => match call boot(selected.value)
                ok => ok
`,
	})
	some, none := browserUP13OptionIDs(t, program)
	joined := emittedBody(t, program)
	for _, want := range []string{
		`$canBrowser.queryParameter(`,
		`some:"` + some + `"`,
		`none:"` + none + `"`,
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("bun emission lacks %s", want)
		}
	}
}
