package main

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// G04: scope-aware completion over the editor wire. Completion resolves
// the cursor to its context over the checked snapshot and offers the
// exact candidate set for it: visible body-local bindings by identity
// and order with captured names from enclosing scopes, checked World
// symbols filtered by usage eligibility, near parameters behind a
// resolved callee, and the selected-surface keywords valid there.
// Callable candidates carry their exact checked arity; shadowed or
// same-spelled other-scope names never leak in; unselected grammars
// contribute no keywords; and unresolvable positions yield an empty
// list or a declined null rather than a guess.

// g04Main extends the G03 fixture with a fallible error, a holder
// method, and a chained method call, keeping every G03 needle intact.
var g04Main = strings.Replace(g03Main,
	"    ok call action(doubled) + shared.tag",
	"    ok call action(doubled) + shared.tag + call holder(callable helper, 0).describe(1)",
	1) + `
fn int describe
    on holder self
    emits []
    given
        int extra
    asserts
        sample: holder(callable helper, 0) => 1 => ok 1
    ok extra
`

func g04Completion(id int, uri string, line, character int) string {
	return fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"textDocument/completion","params":{"textDocument":{"uri":%s},"position":{"line":%d,"character":%d}}}`,
		id, jsonQuote(uri), line, character)
}

// g04Raw requests completion at a needle cursor and returns the raw
// wire result, which may be nil for a declined query.
func g04Raw(t *testing.T, files map[string]string, open, needle string, id int) any {
	t.Helper()
	root := writeServerProject(t, files)
	uri := uriFromPath(filepath.Join(root, open))
	line, character := positionOf(t, files[open], needle)
	frames := runExchange(t, []string{
		didOpen(uri, files[open], 1),
		g04Completion(id, uri, line, character),
		`{"jsonrpc":"2.0","method":"exit"}`,
	})
	return g01Response(t, frames, float64(id))
}

// g04Items requests completion and fails on a decline, returning the
// candidate array.
func g04Items(t *testing.T, files map[string]string, open, needle string, id int) []any {
	t.Helper()
	raw := g04Raw(t, files, open, needle, id)
	items, ok := raw.([]any)
	if !ok {
		t.Fatalf("completion at %q declined or malformed: %v", needle, raw)
	}
	return items
}

func g04Object(t *testing.T, raw any) map[string]any {
	t.Helper()
	item, ok := raw.(map[string]any)
	if !ok {
		t.Fatalf("completion item not an object: %v", raw)
	}
	return item
}

func g04Labels(items []any) []string {
	labels := make([]string, len(items))
	for i, raw := range items {
		label, _ := raw.(map[string]any)["label"].(string)
		labels[i] = label
	}
	sort.Strings(labels)
	return labels
}

func g04Kind(t *testing.T, items []any, kind int) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, raw := range items {
		item := g04Object(t, raw)
		if item["kind"] == float64(kind) {
			out = append(out, item)
		}
	}
	return out
}

func g04Count(items []any, label string) int {
	count := 0
	for _, raw := range items {
		if raw.(map[string]any)["label"] == label {
			count++
		}
	}
	return count
}

// g04Find returns the single item carrying a label, failing on zero or
// several: shadowing must surface each name exactly once.
func g04Find(t *testing.T, items []any, label string) map[string]any {
	t.Helper()
	var found map[string]any
	for _, raw := range items {
		item := g04Object(t, raw)
		if item["label"] == label {
			if found != nil {
				t.Fatalf("label %q offered twice in %v", label, g04Labels(items))
			}
			found = item
		}
	}
	if found == nil {
		t.Fatalf("label %q missing from %v", label, g04Labels(items))
	}
	return found
}

func g04WantLabels(t *testing.T, items []any, want []string) {
	t.Helper()
	got := g04Labels(items)
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("completion labels:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func g04WantKindLabels(t *testing.T, items []any, kind int, want []string) {
	t.Helper()
	var got []string
	for _, item := range g04Kind(t, items, kind) {
		label, _ := item["label"].(string)
		got = append(got, label)
	}
	sort.Strings(got)
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("kind %d labels:\n%s\nwant:\n%s", kind, strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// TestG04InitializeAdvertisesCompletion pins the capability: editors
// only offer completion when the server declares it, alongside the
// G01/G02/G03 surface.
func TestG04InitializeAdvertisesCompletion(t *testing.T) {
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
	provider, ok := capabilities["completionProvider"].(map[string]any)
	if !ok {
		t.Fatalf("completion not advertised: %v", capabilities)
	}
	triggers, ok := provider["triggerCharacters"].([]any)
	if !ok || len(triggers) == 0 {
		t.Fatalf("completion triggers missing: %v", provider)
	}
	if capabilities["textDocumentSync"] != 1.0 || capabilities["definitionProvider"] != true || capabilities["documentFormattingProvider"] != true || capabilities["hoverProvider"] != true || capabilities["referencesProvider"] != true {
		t.Fatalf("existing capabilities regressed: %v", capabilities)
	}
}

// TestG04CompletionLocalScope pins scope precision on a body use: the
// input, the step binding, and the callback local surface once each
// with declared types, while other functions' same-spelled inputs and
// the near input stay out.
func TestG04CompletionLocalScope(t *testing.T) {
	items := g04Items(t, map[string]string{"src/main.can": g04Main}, "src/main.can", "ok call action(doub|led)", 2)
	g04WantKindLabels(t, items, compVariable, []string{"action", "doubled", "seed"})
	if item := g04Find(t, items, "action"); item["detail"] != "callable int (int) emits []" || item["documentation"] != "local binding" {
		t.Fatalf("action candidate wrong: %v", item)
	}
	if item := g04Find(t, items, "doubled"); item["detail"] != "int" || item["documentation"] != "local binding" {
		t.Fatalf("doubled candidate wrong: %v", item)
	}
	if item := g04Find(t, items, "seed"); item["detail"] != "int" || item["documentation"] != "local binding" {
		t.Fatalf("seed candidate wrong: %v", item)
	}
	for _, absent := range []string{"prefix", "value", "total", "widened"} {
		if g04Count(items, absent) != 0 {
			t.Fatalf("other-scope %q leaked into %v", absent, g04Labels(items))
		}
	}
	for _, present := range []string{"helper", "combine", "holder", "shared", "append"} {
		g04Find(t, items, present)
	}
}

// TestG04CompletionOrderSensitive pins binder order: an initializer
// sees only bindings starting before it, never its own name.
func TestG04CompletionOrderSensitive(t *testing.T) {
	items := g04Items(t, map[string]string{"src/main.can": g04Main}, "src/main.can", "int doubled = see|d + seed", 2)
	g04WantKindLabels(t, items, compVariable, []string{"seed"})
}

// TestG04CompletionCapturedAndShadowed pins captures and shadowing:
// inside the rebinding arm the enclosing input and the inner binding
// surface once each, and the later step stays out; past the match the
// outer binding returns beside the completed step.
func TestG04CompletionCapturedAndShadowed(t *testing.T) {
	items := g04Items(t, map[string]string{"src/main.can": g04Main}, "src/main.can", "=> tot|al + seed", 2)
	g04WantKindLabels(t, items, compVariable, []string{"seed", "total"})
	if g04Count(items, "total") != 1 {
		t.Fatalf("shadowed total leaked beside the inner one: %v", g04Labels(items))
	}
	if g04Count(items, "widened") != 0 {
		t.Fatalf("later step widened visible inside its own match: %v", g04Labels(items))
	}
	items = g04Items(t, map[string]string{"src/main.can": g04Main}, "src/main.can", "ok wide|ned", 2)
	g04WantKindLabels(t, items, compVariable, []string{"seed", "total", "widened"})
	if g04Count(items, "total") != 1 {
		t.Fatalf("arm total leaked past the match: %v", g04Labels(items))
	}
}

// TestG04CompletionShadowedSymbolSuppressed pins unqualified shadowing
// against the World: a local shadowing a function name suppresses the
// unreachable symbol while unrelated symbols stay.
func TestG04CompletionShadowedSymbolSuppressed(t *testing.T) {
	shadowed := strings.Replace(g04Main, "int doubled = seed + seed", "int helper = seed + seed", 1)
	shadowed = strings.Replace(shadowed, "with prefix = doubled", "with prefix = helper", 1)
	shadowed = strings.Replace(shadowed, "ok call action(doubled)", "ok call action(helper)", 1)
	items := g04Items(t, map[string]string{"src/main.can": shadowed}, "src/main.can", "ok call action(helpe|r)", 2)
	item := g04Find(t, items, "helper")
	if item["kind"] != float64(compVariable) || item["documentation"] != "local binding" {
		t.Fatalf("shadowed helper is not the local: %v", item)
	}
	g04Find(t, items, "combine")
}

// TestG04CompletionUnparseableDeclinesNull pins the no-guess rule: a
// buffer that cannot load declines to null, as does an unknown file.
func TestG04CompletionUnparseableDeclinesNull(t *testing.T) {
	broken := strings.Replace(g04Main, "fn int helper", "fn int helper(", 1)
	if raw := g04Raw(t, map[string]string{"src/main.can": broken}, "src/main.can", "fn int helpe|r(", 2); raw != nil {
		t.Fatalf("unparseable buffer completed: %v", raw)
	}
	root := writeServerProject(t, map[string]string{"src/main.can": g04Main})
	uri := uriFromPath(filepath.Join(root, "src/main.can"))
	frames := runExchange(t, []string{
		g04Completion(2, uri, 0, 0),
		`{"jsonrpc":"2.0","method":"exit"}`,
	})
	if result := g01Response(t, frames, 2); result != nil {
		t.Fatalf("unopened file completed: %v", result)
	}
}
