package emit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/browser"
	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/project"
)

func browserUP11Program(t *testing.T, files map[string]string) *check.Program {
	t.Helper()
	root := t.TempDir()
	all := map[string]string{
		"can.project.json": `{"source_root":"src","error_registry":"can.errors.json"}`,
		"can.errors.json":  `{"active":[],"retired":[]}`,
	}
	for name, text := range files {
		all[name] = text
	}
	for name, text := range all {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	graph, err := project.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	program, err := check.CheckBrowserProgram(graph)
	if err != nil {
		t.Fatal(err)
	}
	return program
}

func browserUP11Artifacts(t *testing.T, program *check.Program) []ir.Artifact {
	t.Helper()
	artifacts, err := BrowserModules(program, "runtime", httpDependencies(t))
	if err != nil {
		t.Fatal(err)
	}
	return artifacts
}

func browserUP11Entry(t *testing.T, artifacts []ir.Artifact) string {
	t.Helper()
	for _, artifact := range artifacts {
		if artifact.Path == browser.BrowserEntry {
			return string(artifact.Bytes)
		}
	}
	t.Fatalf("missing %s", browser.BrowserEntry)
	return ""
}

func browserUP11Authored(t *testing.T, artifacts []ir.Artifact) string {
	t.Helper()
	var bodies []string
	for _, artifact := range artifacts {
		if !strings.HasSuffix(artifact.Path, ".ts") || artifact.Runtime {
			continue
		}
		if artifact.Path == "program/state.ts" || artifact.Path == browser.BrowserEntry {
			continue
		}
		if strings.HasPrefix(artifact.Path, "assertions/") {
			continue
		}
		bodies = append(bodies, string(artifact.Bytes))
	}
	if len(bodies) == 0 {
		t.Fatal("no authored browser modules emitted")
	}
	return strings.Join(bodies, "\n")
}

const browserUP11EmptyMain = `package app
    provides []
    uses []
fn void main
    emits []
    asserts
        empty: => ok
    ok
`

func TestBrowserEntryIsOnceOnlyDOMReadyStartup(t *testing.T) {
	program := browserUP11Program(t, map[string]string{"src/main.can": browserUP11EmptyMain})
	artifacts := browserUP11Artifacts(t, program)
	entry := browserUP11Entry(t, artifacts)
	for _, want := range []string{
		`BROWSER_PROFILE = "browser-main"`,
		"$canBrowserMain",
		"$canRunBrowserEntry",
		"browser/entry.ts",
		"$canInitialize()",
		"$canMain($canCtx)",
		"$canStarted",
		"void $canBrowserMain();",
		`kind: "can.source-index"`,
	} {
		if !strings.Contains(entry, want) {
			t.Fatalf("browser entry lacks %q:\n%s", want, entry)
		}
	}
	for _, banned := range []string{
		"process.",
		"Bun.",
		"require(",
		"node:",
		"AsyncLocalStorage",
		"runOwnedRoot",
		"$canCallContext",
		"$canWithFixture",
		"...$canArgs",
		"Parameters<typeof $canMain>",
	} {
		if strings.Contains(entry, banned) {
			t.Fatalf("browser entry contains forbidden %q:\n%s", banned, entry)
		}
	}
	if strings.Contains(entry, "$canConfigureDiagnostics") {
		t.Fatalf("browser entry still configures Bun diagnostics:\n%s", entry)
	}
}

func TestBrowserThreadsExplicitOwnerContext(t *testing.T) {
	program := browserUP11Program(t, map[string]string{"src/main.can": `package app
    provides []
    uses []
fn int doubled
    emits []
    given
        int value
    asserts
        sample: 4 => ok 8
    ok value + value
fn void main
    emits []
    asserts
        empty: => ok
    match call doubled(21)
        ok int got => ok
`})
	artifacts := browserUP11Artifacts(t, program)
	authored := browserUP11Authored(t, artifacts)
	for _, want := range []string{
		"$canCtx: $canOwnerContext",
		"$canCtx, $canContext",
		"$canOwnerContext",
	} {
		if !strings.Contains(authored, want) {
			t.Fatalf("browser authored lacks %q:\n%s", want, authored)
		}
	}
	for _, banned := range []string{
		"$canCallContext",
		"$canWithFixture",
		"$canScopeRequest",
	} {
		if strings.Contains(authored, banned) {
			t.Fatalf("browser authored contains forbidden %q", banned)
		}
	}
	for _, artifact := range artifacts {
		if !strings.HasSuffix(artifact.Path, ".ts") || artifact.Runtime {
			continue
		}
		for _, edge := range artifact.Imports {
			if strings.Contains(edge, "/assert/") {
				t.Fatalf("%s reaches assertion module %q in browser production", artifact.Path, edge)
			}
		}
	}
}

func TestBrowserCoordinationUsesExplicitSettle(t *testing.T) {
	program := browserUP11Program(t, map[string]string{"src/main.can": `package app
    provides []
    uses []
fn int left
    emits []
    asserts
        sample: => ok 1
    ok 1
fn int right
    emits []
    asserts
        sample: => ok 2
    ok 2
fn void main
    emits []
    asserts
        empty: => ok
    int[] both = match call concurrent
        left()
            ok int value => ok value
        right()
            ok int value => ok value
    ok
`})
	artifacts := browserUP11Artifacts(t, program)
	authored := browserUP11Authored(t, artifacts)
	for _, want := range []string{
		"$canCoordinateSettleWithContext($canCtx,",
		"run:($canCtx: $canOwnerContext)=>",
	} {
		if !strings.Contains(authored, want) {
			t.Fatalf("browser coordination lacks %q:\n%s", want, authored)
		}
	}
	for _, banned := range []string{
		"$canCoordinateSettle(",
		"$canCallContext",
		"number[][]",
	} {
		if strings.Contains(authored, banned) {
			t.Fatalf("browser coordination contains forbidden %q", banned)
		}
	}
}

func TestBrowserBindsQueryCancelOperations(t *testing.T) {
	program := browserUP11Program(t, map[string]string{"src/main.can": `package app
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
`})
	artifacts := browserUP11Artifacts(t, program)
	authored := browserUP11Authored(t, artifacts)
	state := stateText(t, artifacts)
	for _, want := range []string{
		"$canBrowser.queryParameter",
		"invalidQuery:",
	} {
		if !strings.Contains(authored+state, want) {
			t.Fatalf("browser query binding lacks %q", want)
		}
	}
	program2 := browserUP11Program(t, map[string]string{"src/main.can": `package app
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
    asserts
        empty: => ok
    ok
`})
	artifacts2 := browserUP11Artifacts(t, program2)
	authored2 := browserUP11Authored(t, artifacts2)
	for _, want := range []string{
		"$canBrowser.onCancelKey",
		"$canBrowser.onCancelEvent",
		"$canOwnCallable(",
	} {
		if !strings.Contains(authored2, want) {
			t.Fatalf("browser cancel binding lacks %q", want)
		}
	}
	if strings.Contains(authored2, "$canOwnCallable($canContext") || strings.Contains(authored2, "],$canContext)") {
		t.Fatalf("browser callable still threads assertion context:\n%s", authored2)
	}
}

func TestBunEmissionKeepsAssertionContext(t *testing.T) {
	program := browserEmitProgram(t, map[string]string{"src/main.can": browserPureSource})
	artifacts, err := ProgramModules(program, "runtime", httpDependencies(t))
	if err != nil {
		t.Fatal(err)
	}
	var bodies []string
	for _, artifact := range artifacts {
		if strings.HasSuffix(artifact.Path, ".ts") && !artifact.Runtime && artifact.Path != "program/state.ts" && artifact.Path != "entry.ts" && !strings.HasPrefix(artifact.Path, "assertions/") {
			bodies = append(bodies, string(artifact.Bytes))
		}
	}
	joined := strings.Join(bodies, "\n")
	for _, want := range []string{
		"$canContext?: $canAssertionContext",
		"$canCallContext",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("bun emission lost %q", want)
		}
	}
	for _, banned := range []string{
		"$canCtx",
		"$canOwnerContext",
		"$canCoordinateSettleWithContext",
	} {
		if strings.Contains(joined, banned) {
			t.Fatalf("bun emission contains browser-only %q", banned)
		}
	}
}
