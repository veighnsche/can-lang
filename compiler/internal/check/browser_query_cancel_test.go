package check

import (
	"strings"
	"testing"
)

const browserQueryCancelFixture = `package app
    provides []
    uses [browser, option]
fn void on_field_key
    emits []
    given
        browser::event e
    asserts
        sample: browser::event("keydown", "input", "", "Enter") => ok
    ok
fn void on_submit
    emits []
    given
        browser::event e
    asserts
        sample: browser::event("submit", "form", "", "") => ok
    ok
fn void boot
    emits []
    given
        str selected
    asserts
        sample: "inv-1" => ok
    ok
fn void show_boot_notice
    emits []
    given
        str message
    asserts
        sample: "hi" => ok
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
`

func TestBrowserQueryParameterAdmitsSelectedShape(t *testing.T) {
	program, err := programFixture(t, map[string]string{"src/main.can": browserQueryCancelFixture})
	if err != nil {
		t.Fatalf("selected query_parameter shape rejected: %v", err)
	}
	if program.Intrinsics["can.std.browser@1::query_parameter"] == nil {
		t.Fatal("query_parameter intrinsic missing")
	}
}

func TestBrowserQueryParameterRefusals(t *testing.T) {
	for _, tc := range []struct{ name, from, to, want string }{
		{"uppercase key", `query_parameter("invoice")`, `query_parameter("Invoice")`, `browser query key "Invoice" must match`},
		{"empty key", `query_parameter("invoice")`, `query_parameter("")`, `must be 1-64`},
		{"digit start", `query_parameter("invoice")`, `query_parameter("1voice")`, `must match`},
		{"dash key", `query_parameter("invoice")`, `query_parameter("in-voice")`, `must match`},
		{"space key", `query_parameter("invoice")`, `query_parameter("in voice")`, `must match`},
		{"dynamic key", `query_parameter("invoice")`, `query_parameter(key)`, `must be a static literal`},
		{"missing invalid arm", "        browser::invalid_query => match call show_boot_notice(\"Invalid invoice link\")\n            ok => ok\n", "", `browser::invalid_query`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := strings.Replace(browserQueryCancelFixture, tc.from, tc.to, 1)
			if source == browserQueryCancelFixture {
				t.Fatal("invalid refusal fixture")
			}
			_, err := programFixture(t, map[string]string{"src/main.can": source})
			if err == nil {
				t.Fatalf("accepted invalid query_parameter: %s", tc.to)
			}
			if tc.want != "" && !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("diagnostic omits %q: %v", tc.want, err)
			}
		})
	}
	longKey := strings.Repeat("a", 65)
	source := strings.Replace(browserQueryCancelFixture, `query_parameter("invoice")`, `query_parameter("`+longKey+`")`, 1)
	if _, err := programFixture(t, map[string]string{"src/main.can": source}); err == nil {
		t.Fatal("accepted overlong query key")
	} else if !strings.Contains(err.Error(), "1-64") {
		t.Fatalf("overlong key diagnostic omits bound: %v", err)
	}
}

const browserCancelFixture = `package app
    provides []
    uses [browser]
fn void on_field_key
    emits []
    given
        browser::event e
    asserts
        sample: browser::event("keydown", "input", "", "Enter") => ok
    ok
fn void on_submit
    emits []
    given
        browser::event e
    asserts
        sample: browser::event("submit", "form", "", "") => ok
    ok
fn void demo
    emits [browser::missing_root, browser::disposed, browser::rejected]
    given
        str root
    asserts
        sample: "app" => ok
    match call browser::mount(root)
        browser::missing_root
        ok browser::app app => match call browser::root(app)
            browser::disposed
            ok browser::node anchor => match call browser::open_view(app)
                browser::disposed
                ok browser::view view => match call browser::create_element(view, "input")
                    browser::disposed
                    browser::rejected
                    ok browser::node input => match call browser::on_cancel_key(view, input, "keydown", "Enter", callable on_field_key)
                        browser::disposed
                        browser::rejected
                        ok => match call browser::on_cancel_event(view, anchor, "submit", callable on_submit)
                            browser::disposed
                            browser::rejected
                            ok => ok
fn void main
    emits []
    given
        str[] arguments
    asserts
        empty: [] => ok
    ok
`

func TestBrowserCancelAdmitsSelectedShapes(t *testing.T) {
	if _, err := programFixture(t, map[string]string{"src/main.can": browserCancelFixture}); err != nil {
		t.Fatalf("selected cancel shapes rejected: %v", err)
	}
}

func TestBrowserCancelRefusals(t *testing.T) {
	for _, tc := range []struct{ name, from, to, want string }{
		{"click cancel key", `"keydown", "Enter"`, `"click", "Enter"`, `browser cancel key event "click" is not admitted`},
		{"empty cancel key", `"keydown", "Enter"`, `"keydown", ""`, `nonempty exact key`},
		{"call result cancel callback", `callable on_field_key`, `call on_field_key(browser::event("keydown", "", "", ""))`, `must be a named reference`},
		{"click cancel event", `"submit", callable on_submit`, `"click", callable on_submit`, `browser cancel event "click" is not admitted`},
		{"keydown cancel event", `"submit", callable on_submit`, `"keydown", callable on_submit`, `browser cancel event "keydown" is not admitted`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := strings.Replace(browserCancelFixture, tc.from, tc.to, 1)
			if source == browserCancelFixture {
				t.Fatal("invalid refusal fixture")
			}
			_, err := programFixture(t, map[string]string{"src/main.can": source})
			if err == nil {
				t.Fatalf("accepted invalid cancel policy: %s", tc.to)
			}
			if tc.want != "" && !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("diagnostic omits %q: %v", tc.want, err)
			}
		})
	}
}
