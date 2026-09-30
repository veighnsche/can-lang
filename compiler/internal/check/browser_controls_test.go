package check

import (
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
)

const browserControlsFixture = `package app
    provides []
    uses [browser]
fn str[] use_selected
    emits {}
    given
        str[] picked
    asserts
        sample: [] => ok []
    ok picked
fn browser::file[] use_files
    emits {}
    given
        browser::file[] picked
    asserts
        sample: [] => ok []
    ok picked
fn bool use_modifiers
    emits {}
    given
        browser::modifiers held
    asserts
        sample: browser::modifiers(false, true, false, false) => ok true
    ok held.ctrl
fn int use_selection
    emits {}
    given
        browser::selection caret
    asserts
        sample: browser::selection(-1, -1, "none") => ok -1
    ok caret.start
fn str use_file
    emits {}
    given
        browser::file found
    asserts
        sample: browser::file("a.csv", 12, "text/csv") => ok "a.csv"
    ok found.name
fn void on_field_input
    emits {}
    given
        browser::event e
    asserts
        sample: browser::event("input", "field", "x", "", true, ["a"], [browser::file("a.csv", 12, "text/csv")], browser::modifiers(false, true, false, true), false, browser::selection(0, 1, "forward")) => ok
    match call use_selected(e.selected)
        ok str[] kept => match call use_files(e.files)
            ok browser::file[] held => match call use_modifiers(e.modifiers)
                ok bool flag => match call use_selection(e.selection)
                    ok int at => match e.composing
                        false => match e.checked
                            false => ok
                            true => ok
                        true => match e.checked
                            false => ok
                            true => ok
fn int demo
    emits {browser::missing_root, browser::disposed, browser::rejected}
    given
        str root
    asserts
        sample: "app" => ok 1
    match call browser::mount(root)
        browser::missing_root
        ok browser::app app => match call browser::root(app)
            browser::disposed
            ok browser::node anchor => match call browser::open_view(app)
                browser::disposed
                ok browser::view view => match call browser::create_element(view, "input")
                    browser::disposed
                    browser::rejected
                    ok browser::node field => match call browser::set_value(field, "draft")
                        browser::disposed
                        browser::rejected
                        ok => match call browser::set_checked(field, true)
                            browser::disposed
                            browser::rejected
                            ok => match call browser::create_element(view, "select")
                                browser::disposed
                                browser::rejected
                                ok browser::node tags => match call browser::set_selected(tags, ["a", "b"])
                                    browser::disposed
                                    browser::rejected
                                    ok => match call browser::set_selection(field, 0, 1, "Forward")
                                        browser::disposed
                                        browser::rejected
                                        ok => match call browser::read_value(field)
                                            browser::disposed
                                            browser::rejected
                                            ok str live => match call browser::read_checked(field)
                                                browser::disposed
                                                browser::rejected
                                                ok bool flag => match call browser::read_selected(tags)
                                                    browser::disposed
                                                    browser::rejected
                                                    ok str[] picked => match call browser::read_selection(field)
                                                        browser::disposed
                                                        ok browser::selection caret => match call browser::read_files(field)
                                                            browser::disposed
                                                            browser::rejected
                                                            ok browser::file[] found => match call browser::append_child(anchor, field)
                                                                browser::disposed
                                                                browser::rejected
                                                                ok => match call browser::dispose_view(view)
                                                                    ok => match call browser::dispose_app(app)
                                                                        ok => ok 1
fn void main
    emits {}
    given
        str[] arguments
    asserts
        empty: [] => ok
    ok
`

func TestBrowserControlsSurfaceChecks(t *testing.T) {
	program, err := programFixture(t, map[string]string{"src/main.can": browserControlsFixture})
	if err != nil {
		t.Fatal(err)
	}
	identities := map[string]bool{}
	var walk func(*ir.Region)
	walk = func(region *ir.Region) {
		if region == nil {
			return
		}
		var block func(*ir.Block)
		var completion func(*ir.Completion)
		steps := func(call *ir.Invocation) {
			if call == nil {
				return
			}
			for i := range call.Steps {
				identities[call.Steps[i].Identity] = true
			}
		}
		completion = func(done *ir.Completion) {
			if done == nil {
				return
			}
			steps(done.Call)
			block(done.Block)
			if done.Match != nil {
				steps(done.Match.Call)
				for i := range done.Match.Arms {
					completion(done.Match.Arms[i].Body)
				}
			}
		}
		block = func(b *ir.Block) {
			if b == nil {
				return
			}
			for i := range b.Steps {
				steps(b.Steps[i].Call)
			}
			completion(b.Terminal)
		}
		block(region.Body)
	}
	for _, fn := range program.Functions {
		walk(fn.Region)
	}
	for _, identity := range []string{
		browserSetValue, browserSetChecked, browserSetSelected, browserSetSelection,
		browserReadValue, browserReadChecked, browserReadSelected, browserReadSelection, browserReadFiles,
	} {
		if !identities[identity] {
			t.Fatalf("checked IR omits %s", identity)
		}
	}
}

func TestBrowserControlsAdmissionRefusals(t *testing.T) {
	for _, tc := range []struct{ name, from, to, want string }{
		{"bad direction", `set_selection(field, 0, 1, "Forward")`, `set_selection(field, 0, 1, "sideways")`, `browser selection direction "sideways" is not admitted; expected forward, backward or none`},
		{"empty direction", `set_selection(field, 0, 1, "Forward")`, `set_selection(field, 0, 1, "")`, `browser selection direction "" is not admitted`},
		{"selection arity", `set_selection(field, 0, 1, "Forward")`, `set_selection(field, 0, 1)`, `browser call requires 4 fixed arguments`},
		{"rejected arm outside selection bound", "ok browser::selection caret => match call browser::read_files(field)", "browser::rejected\n                                                        ok browser::selection caret => match call browser::read_files(field)", `error arm is outside matched bound`},
		{"set value mistyped", `set_value(field, "draft")`, `set_value(field, 7)`, ``},
		{"set checked mistyped", `set_checked(field, true)`, `set_checked(field, "yes")`, ``},
		{"set selected mistyped", `set_selected(tags, ["a", "b"])`, `set_selected(tags, "a")`, ``},
		{"read value mistyped binding", `ok str live => match call browser::read_checked(field)`, `ok int live => match call browser::read_checked(field)`, ``},
		{"read checked mistyped binding", `ok bool flag => match call browser::read_selected(tags)`, `ok str flag => match call browser::read_selected(tags)`, ``},
		{"short event construction", `browser::event("input", "field", "x", "", true, ["a"], [browser::file("a.csv", 12, "text/csv")], browser::modifiers(false, true, false, true), false, browser::selection(0, 1, "forward"))`, `browser::event("input", "field", "x", "")`, ``},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := strings.Replace(browserControlsFixture, tc.from, tc.to, 1)
			if source == browserControlsFixture {
				t.Fatal("invalid refusal fixture")
			}
			_, err := programFixture(t, map[string]string{"src/main.can": source})
			if err == nil {
				t.Fatalf("accepted invalid browser controls call: %s", tc.to)
			}
			if tc.want != "" && !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("browser diagnostic omits %q: %v", tc.want, err)
			}
		})
	}
}
