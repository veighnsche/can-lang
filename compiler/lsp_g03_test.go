package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

// G03: project references over the editor wire. Find-references resolves
// the token at the cursor to a binding identity through the project index
// — top-level symbols across files, shared-record fields, callback
// callees, explicit with pins to the callee's near parameter, and
// function-local bindings by scope identity — and returns every indexed
// occurrence sharing that identity. Shadowed or same-spelled bindings in
// other scopes stay distinct; unresolved tokens decline to null; and
// go-to-definition keeps declining locals rather than guessing.

const g03Main = `package app
    provides [run, shadowed]
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

fn int combine
    emits []
    given
        near int prefix
        int value
    asserts
        sample: 3, 4 => ok 7
    ok prefix + value

fn int run
    emits []
    given
        int seed
    asserts
        sample: 1 => ok 4
    int doubled = seed + seed
    callable int (int) emits [] action = callable combine with prefix = doubled
    ok call action(doubled) + shared.tag

fn int shadowed
    emits []
    given
        int seed
    asserts
        sample: 1 => ok 2
    int total = seed
    int widened = match total
        1 => seed + 1
        bind total => total + seed
    ok widened
`

const g03Second = `package app
    provides [describe]
    uses []

fn int describe
    emits []
    given
        int seed
    asserts
        sample: 1 => ok 2
    ok call helper(seed) + 1
`

func g03References(id int, uri string, line, character int, include bool) string {
	return fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"textDocument/references","params":{"textDocument":{"uri":%s},"position":{"line":%d,"character":%d},"context":{"includeDeclaration":%v}}}`,
		id, jsonQuote(uri), line, character, include)
}

// g03Tokens renders each referenced location as "file:line:token" so tests
// pin the exact occurrence set across files.
func g03Tokens(t *testing.T, texts map[string]string, locations []any) []string {
	t.Helper()
	var out []string
	for _, raw := range locations {
		location, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("reference not an object: %v", raw)
		}
		uri, _ := location["uri"].(string)
		text, ok := texts[uri]
		if !ok {
			t.Fatalf("reference outside fixtures: %v", location)
		}
		rng, _ := location["range"].(map[string]any)
		start, _ := rng["start"].(map[string]any)
		end, _ := rng["end"].(map[string]any)
		if start["line"] != end["line"] {
			t.Fatalf("reference spans lines: %v", rng)
		}
		line := int(start["line"].(float64))
		token := strings.Split(text, "\n")[line][int(start["character"].(float64)):int(end["character"].(float64))]
		name := uri[strings.LastIndex(uri, "/")+1:]
		out = append(out, fmt.Sprintf("%s:%d:%s", name, line, token))
	}
	return out
}

func g03Exchange(t *testing.T, files map[string]string, open, needle string, id int, include bool) (map[string]string, []any) {
	t.Helper()
	root := writeServerProject(t, files)
	uris := map[string]string{}
	texts := map[string]string{}
	for name, text := range files {
		uri := uriFromPath(filepath.Join(root, name))
		uris[name] = uri
		texts[uri] = text
	}
	line, character := positionOf(t, files[open], needle)
	frames := runExchange(t, []string{
		didOpen(uris[open], files[open], 1),
		g03References(id, uris[open], line, character, include),
		`{"jsonrpc":"2.0","method":"exit"}`,
	})
	raw := g01Response(t, frames, float64(id))
	if raw == nil {
		t.Fatalf("references declined at %q", needle)
	}
	locations, ok := raw.([]any)
	if !ok {
		t.Fatalf("references not an array: %v", raw)
	}
	return texts, locations
}

func g03Want(t *testing.T, got, want []string) {
	t.Helper()
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("references:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// TestG03InitializeAdvertisesReferences pins the capability: editors only
// offer find-references when the server declares it, alongside the
// G01/G02 surface.
func TestG03InitializeAdvertisesReferences(t *testing.T) {
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
	if capabilities["referencesProvider"] != true {
		t.Fatalf("references not advertised: %v", capabilities)
	}
	if capabilities["textDocumentSync"] != 1.0 || capabilities["definitionProvider"] != true || capabilities["documentFormattingProvider"] != true || capabilities["hoverProvider"] != true {
		t.Fatalf("existing capabilities regressed: %v", capabilities)
	}
}

// TestG03ReferencesLocalVariable pins AU-LSP-references on a body-local:
// the declaration, the nested with-value capture, and the call use.
func TestG03ReferencesLocalVariable(t *testing.T) {
	texts, locations := g03Exchange(t, map[string]string{"src/main.can": g03Main}, "src/main.can", "call action(doub|led)", 2, true)
	g03Want(t, g03Tokens(t, texts, locations), []string{
		"main.can:33:doubled",
		"main.can:34:doubled",
		"main.can:35:doubled",
	})
}

// TestG03ReferencesSameSpelledLocalsStayDistinct pins binding identity
// across scopes: the two `value` inputs resolve to their own function.
func TestG03ReferencesSameSpelledLocalsStayDistinct(t *testing.T) {
	texts, locations := g03Exchange(t, map[string]string{"src/main.can": g03Main}, "src/main.can", "ok val|ue\n\nfn int combine", 2, true)
	g03Want(t, g03Tokens(t, texts, locations), []string{
		"main.can:13:value",
		"main.can:16:value",
	})
	texts, locations = g03Exchange(t, map[string]string{"src/main.can": g03Main}, "src/main.can", "ok prefix + val|ue", 2, true)
	g03Want(t, g03Tokens(t, texts, locations), []string{
		"main.can:22:value",
		"main.can:25:value",
	})
}

// TestG03ReferencesShadowedBinding pins scope identity under shadowing:
// the do-block rebind captures only its own uses, and the outer binding
// keeps only its own.
func TestG03ReferencesShadowedBinding(t *testing.T) {
	texts, locations := g03Exchange(t, map[string]string{"src/main.can": g03Main}, "src/main.can", "=> tot|al + seed", 2, true)
	g03Want(t, g03Tokens(t, texts, locations), []string{
		"main.can:46:total",
		"main.can:46:total",
	})
	texts, locations = g03Exchange(t, map[string]string{"src/main.can": g03Main}, "src/main.can", "int tot|al = seed\n    int widened", 2, true)
	g03Want(t, g03Tokens(t, texts, locations), []string{
		"main.can:43:total",
		"main.can:44:total",
	})
}

// TestG03ReferencesNestedCapture pins captures from an enclosing scope:
// the input is referenced from the nested do-block as well as the body.
func TestG03ReferencesNestedCapture(t *testing.T) {
	texts, locations := g03Exchange(t, map[string]string{"src/main.can": g03Main}, "src/main.can", "int se|ed\n    asserts\n        sample: 1 => ok 2", 2, true)
	g03Want(t, g03Tokens(t, texts, locations), []string{
		"main.can:40:seed",
		"main.can:43:seed",
		"main.can:45:seed",
		"main.can:46:seed",
	})
}

// TestG03ReferencesWithBindingToCalleeParameter pins the Q3 provenance:
// the explicit with pin and the callee's near input are one identity.
func TestG03ReferencesWithBindingToCalleeParameter(t *testing.T) {
	texts, locations := g03Exchange(t, map[string]string{"src/main.can": g03Main}, "src/main.can", "with pre|fix = doubled", 2, true)
	g03Want(t, g03Tokens(t, texts, locations), []string{
		"main.can:21:prefix",
		"main.can:25:prefix",
		"main.can:34:prefix",
	})
	texts, locations = g03Exchange(t, map[string]string{"src/main.can": g03Main}, "src/main.can", "near int pre|fix", 2, true)
	g03Want(t, g03Tokens(t, texts, locations), []string{
		"main.can:21:prefix",
		"main.can:25:prefix",
		"main.can:34:prefix",
	})
}

// TestG03ReferencesCallbackCallee pins the callback scenario: the
// callable reference's callee shares identity with its declaration.
func TestG03ReferencesCallbackCallee(t *testing.T) {
	texts, locations := g03Exchange(t, map[string]string{"src/main.can": g03Main}, "src/main.can", "callable help|er, 0", 2, true)
	g03Want(t, g03Tokens(t, texts, locations), []string{
		"main.can:8:helper",
		"main.can:10:helper",
	})
}

// TestG03ReferencesSharedRecordField pins the shared-record scenario: a
// field use behind a module value shares identity with its declaration.
func TestG03ReferencesSharedRecordField(t *testing.T) {
	texts, locations := g03Exchange(t, map[string]string{"src/main.can": g03Main}, "src/main.can", "shared.ta|g", 2, true)
	g03Want(t, g03Tokens(t, texts, locations), []string{
		"main.can:6:tag",
		"main.can:35:tag",
	})
}

// TestG03ReferencesCrossFile pins project scope: a use in a second file
// joins the declaration's identity.
func TestG03ReferencesCrossFile(t *testing.T) {
	texts, locations := g03Exchange(t, map[string]string{"src/main.can": g03Main, "src/second.can": g03Second}, "src/main.can", "callable help|er, 0", 2, true)
	g03Want(t, g03Tokens(t, texts, locations), []string{
		"main.can:8:helper",
		"main.can:10:helper",
		"second.can:10:helper",
	})
}

// TestG03ReferencesExcludeDeclaration pins includeDeclaration=false: the
// binding site drops out while uses stay.
func TestG03ReferencesExcludeDeclaration(t *testing.T) {
	texts, locations := g03Exchange(t, map[string]string{"src/main.can": g03Main}, "src/main.can", "int doub|led = seed", 2, false)
	g03Want(t, g03Tokens(t, texts, locations), []string{
		"main.can:34:doubled",
		"main.can:35:doubled",
	})
}

// TestG03ReferencesUnresolvedReturnsNull pins the no-guess rule: a callee
// the checked World cannot resolve declines to null, and whitespace
// declines too.
func TestG03ReferencesUnresolvedReturnsNull(t *testing.T) {
	broken := strings.Replace(g03Main, "ok call action(doubled) + shared.tag", "ok call missing_fn(doubled)", 1)
	root := writeServerProject(t, map[string]string{"src/main.can": broken})
	uri := uriFromPath(filepath.Join(root, "src/main.can"))
	line, character := positionOf(t, broken, "call missing|_fn(doubled)")
	frames := runExchange(t, []string{
		didOpen(uri, broken, 1),
		g03References(2, uri, line, character, true),
		g03References(3, uri, 3, 0, true),
		`{"jsonrpc":"2.0","method":"exit"}`,
	})
	if result := g01Response(t, frames, 2); result != nil {
		t.Fatalf("unresolved callee referenced: %v", result)
	}
	if result := g01Response(t, frames, 3); result != nil {
		t.Fatalf("whitespace referenced: %v", result)
	}
}

// TestG03DefinitionStillDeclinesLocals pins the G03 boundary: references
// own binding identity while go-to-definition keeps declining locals.
func TestG03DefinitionStillDeclinesLocals(t *testing.T) {
	root := writeServerProject(t, map[string]string{"src/main.can": g03Main})
	uri := uriFromPath(filepath.Join(root, "src/main.can"))
	line, character := positionOf(t, g03Main, "call action(doub|led)")
	frames := runExchange(t, []string{
		didOpen(uri, g03Main, 1),
		definition(2, uri, line, character),
		`{"jsonrpc":"2.0","method":"exit"}`,
	})
	if result := g01Response(t, frames, 2); result != nil {
		t.Fatalf("definition guessed a local: %v", result)
	}
}

// TestG03ReferencesWarnedFile pins diagnostics parity: a file carrying a
// check-pipeline warning still references, and still publishes exactly
// that warning.
func TestG03ReferencesWarnedFile(t *testing.T) {
	root := writeServerProject(t, map[string]string{"src/main.can": g01FinalLocal})
	uri := uriFromPath(filepath.Join(root, "src/main.can"))
	line, character := positionOf(t, g01FinalLocal, "fn int forward|ed")
	frames := runExchange(t, []string{
		didOpen(uri, g01FinalLocal, 1),
		g03References(2, uri, line, character, true),
		`{"jsonrpc":"2.0","method":"exit"}`,
	})
	raw := g01Response(t, frames, 2)
	locations, ok := raw.([]any)
	if !ok || len(locations) == 0 {
		t.Fatalf("warned file declined references: %v", raw)
	}
	last := lastPublishFor(frames, uri)
	if last == nil {
		t.Fatalf("no publish in %v", frames)
	}
	diags := diagnosticsOf(t, last)
	if len(diags) != 1 || diags[0]["severity"] != 2.0 {
		t.Fatalf("warned file misdiagnosed beside references: %v", diags)
	}
}
