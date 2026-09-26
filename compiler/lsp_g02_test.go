package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// G02: checked-contract hover over the editor wire. Hover resolves the
// token at the cursor through the checked World — the same resolution
// go-to-definition uses — and renders the declared contract in the
// canonical spelling with package or catalogue provenance. Unresolved
// names, body-local bindings, and whitespace decline to null; warnings
// never block a hover and diagnostics stay identical to the CLI stream.

// Shared record carrying a callback field plus the module value and the
// functions that exercise it: hovering the record, the value, the fields,
// and the callback callee covers the acceptance flow's inspect-contract
// step on callback and shared-record scenarios.
const g02Main = `package app
    provides [run]
    uses []

record holder
    callable int (int) emits [] action
    int tag

holder shared = holder(callable helper, 0)

fn int helper
    emits []
    given
        int value
    asserts
        sample: 1 => ok 1
    ok value

fn int run
    emits []
    given
        int seed
    asserts
        sample: 1 => ok 1
    ok shared.tag + call shared.action(seed)
`

func g02Hover(id int, uri string, line, character int) string {
	return fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"textDocument/hover","params":{"textDocument":{"uri":%s},"position":{"line":%d,"character":%d}}}`,
		id, jsonQuote(uri), line, character)
}

func g02Result(t *testing.T, frames []map[string]any, id float64) map[string]any {
	t.Helper()
	result, ok := g01Response(t, frames, id).(map[string]any)
	if !ok {
		t.Fatalf("hover %v not an object: %v", id, frames)
	}
	return result
}

func g02Contents(t *testing.T, result map[string]any) string {
	t.Helper()
	contents, ok := result["contents"].(map[string]any)
	if !ok {
		t.Fatalf("hover contents not an object: %v", result)
	}
	if contents["kind"] != "markdown" {
		t.Fatalf("hover contents not markdown: %v", contents)
	}
	value, ok := contents["value"].(string)
	if !ok {
		t.Fatalf("hover value not a string: %v", contents)
	}
	return value
}

// g02Token extracts the source text covered by the hover range: ranges
// must land exactly on the hovered token on one line.
func g02Token(t *testing.T, fixture string, result map[string]any) string {
	t.Helper()
	rng, _ := result["range"].(map[string]any)
	start, _ := rng["start"].(map[string]any)
	end, _ := rng["end"].(map[string]any)
	if start["line"] != end["line"] {
		t.Fatalf("hover range spans lines: %v", rng)
	}
	lines := strings.Split(fixture, "\n")
	line := int(start["line"].(float64))
	return lines[line][int(start["character"].(float64)):int(end["character"].(float64))]
}

func g02Exchange(t *testing.T, fixture, needle string, id int) (string, []map[string]any) {
	t.Helper()
	root := writeServerProject(t, map[string]string{"src/main.can": fixture})
	uri := uriFromPath(filepath.Join(root, "src/main.can"))
	line, character := positionOf(t, fixture, needle)
	frames := runExchange(t, []string{
		didOpen(uri, fixture, 1),
		g02Hover(id, uri, line, character),
		`{"jsonrpc":"2.0","method":"exit"}`,
	})
	return uri, frames
}

// TestG02InitializeAdvertisesHover pins the capability: editors only
// offer hover when the server declares it, alongside the G01 surface.
func TestG02InitializeAdvertisesHover(t *testing.T) {
	frames := runExchange(t, []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
		`{"jsonrpc":"2.0","method":"exit"}`,
	})
	result, ok := g01Response(t, frames, 1).(map[string]any)
	if !ok {
		t.Fatalf("initialize result not an object: %v", frames)
	}
	capabilities, ok := result["capabilities"].(map[string]any)
	if !ok {
		t.Fatalf("no capabilities in %v", result)
	}
	if capabilities["hoverProvider"] != true {
		t.Fatalf("hover not advertised: %v", capabilities)
	}
	if capabilities["textDocumentSync"] != 1.0 || capabilities["definitionProvider"] != true || capabilities["documentFormattingProvider"] != true {
		t.Fatalf("existing capabilities regressed: %v", capabilities)
	}
}

// TestG02HoverCallbackCallee pins AU-LSP-hover on the callback scenario:
// the callable reference's callee hovers as its full function contract.
func TestG02HoverCallbackCallee(t *testing.T) {
	_, frames := g02Exchange(t, g02Main, "callable help|er", 2)
	result := g02Result(t, frames, 2)
	want := "```can\nfn int helper\nemits []\ngiven\n    int value\n```\n\npackage app"
	if got := g02Contents(t, result); got != want {
		t.Fatalf("hover contents:\n%s\nwant:\n%s", got, want)
	}
	if token := g02Token(t, g02Main, result); token != "helper" {
		t.Fatalf("hover range covers %q, want helper", token)
	}
}

// TestG02HoverDeclarationName pins self hovers: the declaration name
// itself reports the same contract a use site sees.
func TestG02HoverDeclarationName(t *testing.T) {
	_, frames := g02Exchange(t, g02Main, "fn int r|un", 2)
	result := g02Result(t, frames, 2)
	want := "```can\nfn int run\nemits []\ngiven\n    int seed\n```\n\npackage app"
	if got := g02Contents(t, result); got != want {
		t.Fatalf("hover contents:\n%s\nwant:\n%s", got, want)
	}
	if token := g02Token(t, g02Main, result); token != "run" {
		t.Fatalf("hover range covers %q, want run", token)
	}
}

// TestG02HoverSharedRecordField pins AU-LSP-hover on the shared-record
// scenario: a field behind a module value hovers as its declared line.
func TestG02HoverSharedRecordField(t *testing.T) {
	_, frames := g02Exchange(t, g02Main, "shared.ta|g", 2)
	result := g02Result(t, frames, 2)
	want := "```can\nint holder.tag\n```\n\npackage app"
	if got := g02Contents(t, result); got != want {
		t.Fatalf("hover contents:\n%s\nwant:\n%s", got, want)
	}
	if token := g02Token(t, g02Main, result); token != "tag" {
		t.Fatalf("hover range covers %q, want tag", token)
	}
}

// TestG02HoverCallbackField pins the callback contract behind the
// shared record: the callable-typed field hovers with its full type.
func TestG02HoverCallbackField(t *testing.T) {
	_, frames := g02Exchange(t, g02Main, "shared.act|ion", 2)
	result := g02Result(t, frames, 2)
	want := "```can\ncallable int (int) emits [] holder.action\n```\n\npackage app"
	if got := g02Contents(t, result); got != want {
		t.Fatalf("hover contents:\n%s\nwant:\n%s", got, want)
	}
	if token := g02Token(t, g02Main, result); token != "action" {
		t.Fatalf("hover range covers %q, want action", token)
	}
}

// TestG02HoverRecordName pins record inspection from a use site: the
// shared record hovers as its header plus every declared field.
func TestG02HoverRecordName(t *testing.T) {
	_, frames := g02Exchange(t, g02Main, "hold|er shared", 2)
	result := g02Result(t, frames, 2)
	want := "```can\nrecord holder\n    callable int (int) emits [] action\n    int tag\n```\n\npackage app"
	if got := g02Contents(t, result); got != want {
		t.Fatalf("hover contents:\n%s\nwant:\n%s", got, want)
	}
	if token := g02Token(t, g02Main, result); token != "holder" {
		t.Fatalf("hover range covers %q, want holder", token)
	}
}

// TestG02HoverModuleValue pins module-value hovers: the shared value
// reports its nominal record annotation.
func TestG02HoverModuleValue(t *testing.T) {
	_, frames := g02Exchange(t, g02Main, "ok shar|ed.tag", 2)
	result := g02Result(t, frames, 2)
	want := "```can\nholder shared\n```\n\npackage app"
	if got := g02Contents(t, result); got != want {
		t.Fatalf("hover contents:\n%s\nwant:\n%s", got, want)
	}
	if token := g02Token(t, g02Main, result); token != "shared" {
		t.Fatalf("hover range covers %q, want shared", token)
	}
}

// TestG02HoverPrimitive pins catalogue provenance: prelude types hover
// by kind even though no source file declares them.
func TestG02HoverPrimitive(t *testing.T) {
	_, frames := g02Exchange(t, g02Main, "in|t seed", 2)
	result := g02Result(t, frames, 2)
	want := "```can\nprimitive int\n```\n\ncatalogue"
	if got := g02Contents(t, result); got != want {
		t.Fatalf("hover contents:\n%s\nwant:\n%s", got, want)
	}
	if token := g02Token(t, g02Main, result); token != "int" {
		t.Fatalf("hover range covers %q, want int", token)
	}
}

// TestG02HoverUnresolvedReturnsNull pins the no-guess rule: a callee
// the checked World cannot resolve hovers to null, and the breakage
// still publishes its diagnostic; whitespace hovers to null too.
func TestG02HoverUnresolvedReturnsNull(t *testing.T) {
	broken := strings.Replace(g02Main, "ok shared.tag + call shared.action(seed)", "ok call missing_fn(seed)", 1)
	root := writeServerProject(t, map[string]string{"src/main.can": broken})
	uri := uriFromPath(filepath.Join(root, "src/main.can"))
	line, character := positionOf(t, broken, "call missing|_fn(seed)")
	frames := runExchange(t, []string{
		didOpen(uri, broken, 1),
		g02Hover(2, uri, line, character),
		g02Hover(3, uri, 3, 0),
		`{"jsonrpc":"2.0","method":"exit"}`,
	})
	if result := g01Response(t, frames, 2); result != nil {
		t.Fatalf("unresolved callee hovered: %v", result)
	}
	if result := g01Response(t, frames, 3); result != nil {
		t.Fatalf("whitespace hovered: %v", result)
	}
	last := lastPublishFor(frames, uri)
	if last == nil {
		t.Fatalf("no publish in %v", frames)
	}
	diags := diagnosticsOf(t, last)
	if len(diags) != 1 || !strings.Contains(diags[0]["message"].(string), "missing_fn") {
		t.Fatalf("breakage unpublished beside null hovers: %v", diags)
	}
}

// TestG02HoverDeclinesLocals pins the G03 handoff: body-local bindings
// decline exactly like go-to-definition until references own binding
// identity, rather than risk a shadowed name's type.
func TestG02HoverDeclinesLocals(t *testing.T) {
	_, frames := g02Exchange(t, g02Main, "action(se|ed)", 2)
	if result := g01Response(t, frames, 2); result != nil {
		t.Fatalf("local binding hovered: %v", result)
	}
}

// TestG02HoverWarnedFile pins diagnostics parity on the authoring flow:
// a file carrying a check-pipeline warning still hovers its contract,
// still publishes exactly that warning, and hover never touches disk.
func TestG02HoverWarnedFile(t *testing.T) {
	root := writeServerProject(t, map[string]string{"src/main.can": g01FinalLocal})
	uri := uriFromPath(filepath.Join(root, "src/main.can"))
	path := filepath.Join(root, "src/main.can")
	line, character := positionOf(t, g01FinalLocal, "fn int forward|ed")
	frames := runExchange(t, []string{
		didOpen(uri, g01FinalLocal, 1),
		g02Hover(2, uri, line, character),
		`{"jsonrpc":"2.0","method":"exit"}`,
	})
	result := g02Result(t, frames, 2)
	want := "```can\nfn int forwarded\nemits []\ngiven\n    int left\n    int right\n```\n\npackage app"
	if got := g02Contents(t, result); got != want {
		t.Fatalf("hover contents:\n%s\nwant:\n%s", got, want)
	}
	last := lastPublishFor(frames, uri)
	if last == nil {
		t.Fatalf("no publish in %v", frames)
	}
	diags := diagnosticsOf(t, last)
	if len(diags) != 1 || diags[0]["severity"] != 2.0 {
		t.Fatalf("warned file misdiagnosed beside hover: %v", diags)
	}
	if disk, err := os.ReadFile(path); err != nil || string(disk) != g01FinalLocal {
		t.Fatalf("hover reached disk: %v", err)
	}
}
