package emit

import (
	"strings"
	"testing"
)

const browserControlsEmitFixture = `package app
    provides []
    uses [browser]
fn void on_field_input
    emits []
    given
        browser::event e
    asserts
        sample: browser::event("input", "field", "x", "", false, [], [], browser::modifiers(false, false, false, false), false, browser::selection(-1, -1, "none")) => ok
    ok
fn void main
    emits []
    asserts
        empty: => ok
    match call browser::mount("app")
        browser::missing_root => ok
        ok browser::app app => match call browser::root(app)
            browser::disposed => ok
            ok browser::node anchor => match call browser::open_view(app)
                browser::disposed => ok
                ok browser::view view => match call browser::create_element(view, "input")
                    browser::disposed => ok
                    browser::rejected => ok
                    ok browser::node field => match call browser::set_value(field, "x")
                        browser::disposed => ok
                        browser::rejected => ok
                        ok => match call browser::set_checked(field, true)
                            browser::disposed => ok
                            browser::rejected => ok
                            ok => match call browser::set_selected(field, [])
                                browser::disposed => ok
                                browser::rejected => ok
                                ok => match call browser::set_selection(field, 0, 0, "none")
                                    browser::disposed => ok
                                    browser::rejected => ok
                                    ok => match call browser::read_value(field)
                                        browser::disposed => ok
                                        browser::rejected => ok
                                        ok str live => match call browser::read_checked(field)
                                            browser::disposed => ok
                                            browser::rejected => ok
                                            ok bool flag => match call browser::read_selected(field)
                                                browser::disposed => ok
                                                browser::rejected => ok
                                                ok str[] picked => match call browser::read_selection(field)
                                                    browser::disposed => ok
                                                    ok browser::selection caret => match call browser::read_files(field)
                                                        browser::disposed => ok
                                                        browser::rejected => ok
                                                        ok browser::file[] found => match call browser::on_event(view, field, "input", callable on_field_input)
                                                            browser::disposed => ok
                                                            browser::rejected => ok
                                                            ok => ok
`

func TestBrowserControlsEmitBindings(t *testing.T) {
	program := browserUP11Program(t, map[string]string{"src/main.can": browserControlsEmitFixture})
	artifacts := browserUP11Artifacts(t, program)
	authored := browserUP11Authored(t, artifacts)
	state := stateText(t, artifacts)
	for _, want := range []string{
		"$canBrowser.setValue",
		"$canBrowser.setChecked",
		"$canBrowser.setSelected",
		"$canBrowser.setSelection",
		"$canBrowser.readValue",
		"$canBrowser.readChecked",
		"$canBrowser.readSelected",
		"$canBrowser.readSelection",
		"$canBrowser.readFiles",
	} {
		if !strings.Contains(authored, want) {
			t.Fatalf("browser controls emission lacks %q", want)
		}
	}
	for _, want := range []string{"modifiers:", "selection:", ",file:"} {
		if !strings.Contains(state, want) {
			t.Fatalf("browser factory contracts lack %q", want)
		}
	}
	for _, empty := range []string{`modifiers:""`, `selection:""`, `file:""`} {
		if strings.Contains(state, empty) {
			t.Fatalf("browser factory leaves %s unsealed", empty)
		}
	}
}
