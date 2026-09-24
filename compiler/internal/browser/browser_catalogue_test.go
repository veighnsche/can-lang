package browser

import (
	"strings"
	"testing"
)

// TestBrowserAdmitsBrowserCatalogue verifies the T22 bounded browser
// catalogue passes the transitive capability closure: DOM, event, timer
// and versioned-state operations ship in browser bundles, including
// through callable references and generic state specializations.
func TestBrowserAdmitsBrowserCatalogue(t *testing.T) {
	program := browserFixture(t, map[string]string{"src/main.can": `package app
    provides []
    uses [browser]
fn void on_click
    emits []
    given
        browser::event e
    asserts
        sample: browser::event("click", "", "", "") => ok
    ok
fn void on_tick
    emits []
    asserts
        sample: => ok
    ok
fn int demo
    emits [browser::missing_root, browser::disposed, browser::rejected, browser::stale_version]
    given
        str root
    asserts
        sample: "app" => ok 1
    match call browser::mount(root)
        browser::missing_root
        ok browser::app app => match call browser::open_view(app)
            browser::disposed
            ok browser::view view => match call browser::create_element(view, "div")
                browser::disposed
                browser::rejected
                ok browser::node box => match call browser::on_event(view, box, "click", callable on_click)
                    browser::disposed
                    browser::rejected
                    ok => match call browser::set_timeout(view, 30, callable on_tick)
                        browser::disposed
                        browser::rejected
                        ok => match call browser::create_state(view, 7)
                            browser::disposed
                            ok browser::state<int> cell => match call browser::read_state(cell)
                                browser::disposed
                                ok browser::snapshot<int> snap => match call browser::replace_state(cell, snap.version, 8)
                                    browser::disposed
                                    browser::stale_version
                                    ok int next => match call browser::dispose_view(view)
                                        ok => match call browser::dispose_app(app)
                                            ok => ok next
fn void main
    emits [browser::missing_root, browser::disposed, browser::rejected, browser::stale_version]
    given
        str[] arguments
    asserts
        empty: [] => ok
    match call demo("app")
        browser::missing_root
        browser::disposed
        browser::rejected
        browser::stale_version
        ok int done => ok
`})
	if err := CheckProgram(program); err != nil {
		t.Fatalf("browser catalogue rejected: %v", err)
	}
	if len(program.BrowserStates) != 3 {
		t.Fatalf("expected three browser state specializations, got %d", len(program.BrowserStates))
	}
	for key := range program.BrowserStates {
		if !strings.Contains(key, "/instance/") {
			t.Fatalf("state specialization key %q escapes capability tracking", key)
		}
		if _, denied := ForbiddenReason(key); denied {
			t.Fatalf("browser state specialization %q forbidden", key)
		}
	}
}

// TestBrowserCatalogueKeepsServerDenials verifies browser programs still
// fail closed when a server capability hides behind a browser-shaped
// helper path.
func TestBrowserCatalogueKeepsServerDenials(t *testing.T) {
	program := browserFixture(t, map[string]string{"src/main.can": `package app
    provides []
    uses [browser, env, http]
fn void on_click
    emits []
    given
        browser::event e
    asserts
        sample: browser::event("click", "", "", "") => ok
    ok
fn str helper
    emits [env::invalid_name, http::credentials_missing]
    asserts
        sample: => ok "fixture"
    match call env::required("HOME")
        when
            sample: "HOME" => ok "fixture"
        env::invalid_name
        http::credentials_missing
        ok str value => ok value
fn int demo
    emits [browser::missing_root, browser::disposed, browser::rejected, env::invalid_name, http::credentials_missing]
    given
        str root
    asserts
        sample: "app" => ok 1
    match call browser::mount(root)
        browser::missing_root
        ok browser::app app => match call browser::open_view(app)
            browser::disposed
            ok browser::view view => match call browser::create_element(view, "div")
                browser::disposed
                browser::rejected
                ok browser::node box => match call helper()
                    env::invalid_name
                    http::credentials_missing
                    ok str value => match call browser::dispose_view(view)
                        ok => match call browser::dispose_app(app)
                            ok => ok 1
fn void main
    emits [browser::missing_root, browser::disposed, browser::rejected, env::invalid_name, http::credentials_missing]
    given
        str[] arguments
    asserts
        empty: [] => ok
    match call demo("app")
        browser::missing_root
        browser::disposed
        browser::rejected
        env::invalid_name
        http::credentials_missing
        ok int done => ok
`})
	err := CheckProgram(program)
	if err == nil {
		t.Fatal("browser program with a server helper admitted")
	}
	if !strings.Contains(err.Error(), "can.std.env@1::required") {
		t.Fatalf("capability diagnostic loses the server operation: %v", err)
	}
}
