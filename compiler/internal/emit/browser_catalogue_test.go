package emit

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
)

const browserCatalogueSource = `package app
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
        ok browser::app app => match call browser::root(app)
            browser::disposed
            ok browser::node anchor => match call browser::open_view(app)
                browser::disposed
                ok browser::view view => match call browser::create_element(view, "div")
                    browser::disposed
                    browser::rejected
                    ok browser::node box => match call browser::append_child(anchor, box)
                        browser::disposed
                        browser::rejected
                        ok => match call browser::on_event(view, box, "click", callable on_click)
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
    emits []
    given
        str[] arguments
    asserts
        empty: [] => ok
    ok
`

func browserCatalogueArtifacts(t *testing.T, browser bool) []ir.Artifact {
	t.Helper()
	program := browserEmitProgram(t, map[string]string{"src/main.can": browserCatalogueSource})
	var artifacts []ir.Artifact
	var err error
	if browser {
		artifacts, err = BrowserModules(program, "runtime", httpDependencies(t))
	} else {
		artifacts, err = ProgramModules(program, "runtime", httpDependencies(t))
	}
	if err != nil {
		t.Fatal(err)
	}
	return artifacts
}

func stateText(t *testing.T, artifacts []ir.Artifact) string {
	t.Helper()
	for _, artifact := range artifacts {
		if artifact.Path == "program/state.ts" {
			return string(artifact.Bytes)
		}
	}
	t.Fatal("missing state module")
	return ""
}

func TestBrowserCatalogueBindsBothProfiles(t *testing.T) {
	for _, profile := range []struct {
		name    string
		browser bool
	}{{"bun", false}, {"browser", true}} {
		t.Run(profile.name, func(t *testing.T) {
			artifacts := browserCatalogueArtifacts(t, profile.browser)
			text := stateText(t, artifacts)
			for _, want := range []string{
				"$canCreateBrowser",
				"$canCreateBrowserState",
				"$canBrowserKinds",
				"$canIsBrowser",
				"$canIsBrowserState",
				"$canBrowserState0",
				"platform/browser.ts",
			} {
				if !strings.Contains(text, want) {
					t.Fatalf("%s state lacks %q", profile.name, want)
				}
			}
			var authored []string
			for _, artifact := range artifacts {
				if !strings.HasSuffix(artifact.Path, ".ts") || artifact.Runtime || artifact.Path == "program/state.ts" {
					continue
				}
				if strings.HasPrefix(artifact.Path, "assertions/") {
					continue
				}
				authored = append(authored, string(artifact.Bytes))
			}
			if len(authored) == 0 {
				t.Fatal("no authored modules emitted")
			}
			joined := strings.Join(authored, "\n")
			for _, want := range []string{
				"$canBrowser.mount",
				"$canBrowser.root",
				"$canBrowser.openView",
				"$canBrowser.createElement",
				"$canBrowser.appendChild",
				"$canBrowser.onEvent",
				"$canBrowser.setTimeout",
				"$canBrowser.disposeView",
				"$canBrowser.disposeApp",
				".createState",
				".readState",
				".replaceState",
			} {
				if !strings.Contains(joined, want) {
					t.Fatalf("%s output lacks binding %q", profile.name, want)
				}
			}
			if !strings.Contains(joined, "$canBrowserState") {
				t.Fatalf("%s output omits state value imports", profile.name)
			}
			for _, want := range []string{"$canIsBrowser($canBrowserKinds[identity],value)", "$canIsBrowserState(identity,value)"} {
				if !strings.Contains(text, want) {
					t.Fatalf("%s domain predicate lacks %q", profile.name, want)
				}
			}
		})
	}
}

func TestBrowserCatalogueStateIdentitiesAreConcrete(t *testing.T) {
	program := browserEmitProgram(t, map[string]string{"src/main.can": browserCatalogueSource})
	if len(program.BrowserStates) != 3 {
		t.Fatalf("expected three browser state specializations, got %d", len(program.BrowserStates))
	}
	artifacts, err := ProgramModules(program, "runtime", httpDependencies(t))
	if err != nil {
		t.Fatal(err)
	}
	text := stateText(t, artifacts)
	for _, special := range program.BrowserStates {
		for _, identity := range []string{special.State.Identity(), special.Snapshot.Identity()} {
			if identity == "" || !strings.Contains(text, `"`+identity+`"`) {
				t.Fatalf("state module omits concrete identity %q", identity)
			}
		}
	}
	keys := make([]string, 0, len(program.BrowserStates))
	for key := range program.BrowserStates {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var authored strings.Builder
	for _, artifact := range artifacts {
		if !strings.HasSuffix(artifact.Path, ".ts") || artifact.Runtime || artifact.Path == "program/state.ts" {
			continue
		}
		if strings.HasPrefix(artifact.Path, "assertions/") {
			continue
		}
		authored.Write(artifact.Bytes)
	}
	methods := map[string]string{
		"can.std.browser@1::create_state":  "createState",
		"can.std.browser@1::read_state":    "readState",
		"can.std.browser@1::replace_state": "replaceState",
	}
	for i, key := range keys {
		want := fmt.Sprintf("$canBrowserState%d.%s", i, methods[program.BrowserStates[key].Operation])
		if !strings.Contains(authored.String(), want) {
			t.Fatalf("authored output lacks specialization binding %q", want)
		}
		if !strings.Contains(authored.String(), fmt.Sprintf("$canBrowserState%d", i)) {
			t.Fatalf("authored output omits state value import $canBrowserState%d", i)
		}
	}
}

func TestBrowserCatalogueOmitsEmptyStateValues(t *testing.T) {
	program := browserEmitProgram(t, map[string]string{"src/main.can": browserPureSource})
	for _, emit := range []struct {
		name string
		run  func() ([]ir.Artifact, error)
	}{
		{"bun", func() ([]ir.Artifact, error) { return ProgramModules(program, "runtime", httpDependencies(t)) }},
		{"browser", func() ([]ir.Artifact, error) { return BrowserModules(program, "runtime", httpDependencies(t)) }},
	} {
		t.Run(emit.name, func(t *testing.T) {
			artifacts, err := emit.run()
			if err != nil {
				t.Fatal(err)
			}
			text := stateText(t, artifacts)
			if !strings.Contains(text, "$canCreateBrowser") {
				t.Fatalf("%s state lacks the browser factory", emit.name)
			}
			if strings.Contains(text, "$canBrowserState") {
				t.Fatalf("%s state carries state values without specializations", emit.name)
			}
		})
	}
}
