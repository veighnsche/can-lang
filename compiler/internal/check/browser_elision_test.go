package check

import (
	"strings"
	"testing"
)

// T24 elides browser opaque handles (app, view, node and state) in
// assertion rows: no Can expression can name such a value, so without
// elision no function taking one could satisfy mandatory assertions and
// no event handler could reach its view or cell.
const browserElisionFixture = `package app
    provides []
    uses [browser]
fn void on_press
    emits []
    given
        browser::event e
        near browser::state<int> cell
        near browser::node status
    asserts
        sample: browser::event("click", "save", "", "") => ok
    match call browser::read_state(cell)
        browser::disposed => ok
        ok browser::snapshot<int> snap => match call browser::set_text(status, "hi")
            browser::disposed => ok
            ok => ok
fn void mount
    emits [browser::disposed, browser::rejected]
    given
        browser::app app
        browser::view view
        browser::state<int> cell
        browser::node status
    asserts
        sample: => ok
    match call browser::on_event(view, status, "click", callable on_press)
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

func TestBrowserElisionAdmitsHandleInputs(t *testing.T) {
	program, err := programFixture(t, map[string]string{"src/main.can": browserElisionFixture})
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]int{}
	for _, assertion := range program.Assertions {
		seen[assertion.Root.Declaration]++
	}
	if seen["can.project.root/app::on_press"] != 1 || seen["can.project.root/app::mount"] != 1 {
		t.Fatalf("expected one elided assertion per handle-taking function, got %+v", seen)
	}
}

func TestBrowserElisionRefusesNonScopeOpaque(t *testing.T) {
	fixture := `package app
    provides []
    uses [browser, http]
fn void handle
    emits []
    given
        http::server_response response
    asserts
        sample: => ok
    ok
fn void main
    emits []
    given
        str[] arguments
    asserts
        empty: [] => ok
    ok
`
	_, err := programFixture(t, map[string]string{"src/main.can": fixture})
	if err == nil {
		t.Fatal("expected omitted non-scope opaque input to fail")
	}
	if !strings.Contains(err.Error(), "can.project.root/app::handle") {
		t.Fatalf("expected diagnostic to name the assertion owner, got %v", err)
	}
}

func TestBrowserElisionRecordStaysSupplied(t *testing.T) {
	fixture := `package app
    provides []
    uses [browser]
fn void main
    emits []
    given
        browser::event e
    asserts
        sample: => ok
    ok
`
	_, err := programFixture(t, map[string]string{"src/main.can": fixture})
	if err == nil {
		t.Fatal("expected omitted constructible record input to fail")
	}
}
