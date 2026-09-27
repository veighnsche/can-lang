package main

import (
	"encoding/json"
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
// The chained method call targets the module value: methods chain onto
// calls, and only a module value carries an annotation-known receiver,
// so the checker rejects the value-as-callee while resolve still
// supplies the World completion reads.
var g04Main = strings.Replace(g03Main,
	"    ok call action(doubled) + shared.tag",
	"    ok call action(doubled) + shared.tag + call shared().describe(1)",
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

// TestG04CompletionCalleeArity pins caller repair: a broken callee
// completes to the call-eligible names with exact checked arities —
// the intended function, the callback local, and the catalogue
// prelude — while non-callables and keywords stay out.
func TestG04CompletionCalleeArity(t *testing.T) {
	broken := strings.Replace(g04Main, "ok call action(doubled) + shared.tag + call shared().describe(1)", "ok call combin(doubled)", 1)
	items := g04Items(t, map[string]string{"src/main.can": broken}, "src/main.can", "ok call combi|n(doubled)", 2)
	g04WantLabels(t, items, []string{"action", "append", "combine", "helper", "run", "shadowed"})
	if item := g04Find(t, items, "combine"); item["kind"] != float64(compFunction) || item["detail"] != "(near prefix: int, value: int) -> int" || item["documentation"] != "package app" {
		t.Fatalf("combine candidate wrong: %v", item)
	}
	if item := g04Find(t, items, "action"); item["kind"] != float64(compVariable) || item["detail"] != "(int) -> int" || item["documentation"] != "local binding" {
		t.Fatalf("action candidate wrong: %v", item)
	}
	if item := g04Find(t, items, "append"); item["kind"] != float64(compFunction) || item["detail"] != "function" || item["documentation"] != "catalogue" {
		t.Fatalf("append candidate wrong: %v", item)
	}
	if keywords := g04Kind(t, items, compKeyword); len(keywords) != 0 {
		t.Fatalf("keywords in callee position: %v", keywords)
	}
}

// TestG04CompletionCallbackExtraction pins the callback scenario: a
// callable reference completes to reference-eligible functions only —
// no locals, since with-pin resolution declines them, and no keywords.
func TestG04CompletionCallbackExtraction(t *testing.T) {
	items := g04Items(t, map[string]string{"src/main.can": g04Main}, "src/main.can", "callable combi|ne with prefix = doubled", 2)
	g04WantLabels(t, items, []string{"append", "combine", "helper", "run", "shadowed"})
	for _, item := range g04Kind(t, items, compFunction) {
		if item["documentation"] != "package app" && item["documentation"] != "catalogue" {
			t.Fatalf("unexpected provenance: %v", item)
		}
	}
	if locals := g04Kind(t, items, compVariable); len(locals) != 0 {
		t.Fatalf("locals in callable-callee position: %v", locals)
	}
	if keywords := g04Kind(t, items, compKeyword); len(keywords) != 0 {
		t.Fatalf("keywords in callable-callee position: %v", keywords)
	}
}

// TestG04CompletionWithPins pins near precision: a with pin completes
// to the resolved callee's near parameters only, whatever partial
// name is typed; an unresolvable callee yields nothing for the
// checker to diagnose.
func TestG04CompletionWithPins(t *testing.T) {
	items := g04Items(t, map[string]string{"src/main.can": g04Main}, "src/main.can", "with pre|fix = doubled", 2)
	g04WantLabels(t, items, []string{"prefix"})
	if item := g04Find(t, items, "prefix"); item["kind"] != float64(compProperty) || item["detail"] != "near int" || item["documentation"] != "near parameter of combine" {
		t.Fatalf("prefix pin wrong: %v", item)
	}
	renamed := strings.Replace(g04Main, "with prefix = doubled", "with value = doubled", 1)
	items = g04Items(t, map[string]string{"src/main.can": renamed}, "src/main.can", "with val|ue = doubled", 2)
	g04WantLabels(t, items, []string{"prefix"})
	missing := strings.Replace(g04Main, "callable combine with", "callable missing with", 1)
	items = g04Items(t, map[string]string{"src/main.can": missing}, "src/main.can", "with pre|fix = doubled", 2)
	if len(items) != 0 {
		t.Fatalf("pins behind an unknown callee: %v", g04Labels(items))
	}
}

// TestG04CompletionMemberFields pins the shared-record scenario: a
// field use behind a module value completes to the record's declared
// fields, while unknown and function-local receivers yield nothing
// rather than a guessed member.
func TestG04CompletionMemberFields(t *testing.T) {
	items := g04Items(t, map[string]string{"src/main.can": g04Main}, "src/main.can", "shared.ta|g", 2)
	g04WantLabels(t, items, []string{"action", "tag"})
	if item := g04Find(t, items, "tag"); item["kind"] != float64(compField) || item["detail"] != "int" || item["documentation"] != "field of holder" {
		t.Fatalf("tag candidate wrong: %v", item)
	}
	if item := g04Find(t, items, "action"); item["kind"] != float64(compField) || item["detail"] != "callable int (int) emits []" || item["documentation"] != "field of holder" {
		t.Fatalf("action candidate wrong: %v", item)
	}
	missing := strings.Replace(g04Main, "shared.tag", "missing.tag", 1)
	if items := g04Items(t, map[string]string{"src/main.can": missing}, "src/main.can", "missing.ta|g", 2); len(items) != 0 {
		t.Fatalf("members behind an unknown receiver: %v", g04Labels(items))
	}
	local := strings.Replace(g04Main, "shared.tag", "doubled.tag", 1)
	if items := g04Items(t, map[string]string{"src/main.can": local}, "src/main.can", "doubled.ta|g", 2); len(items) != 0 {
		t.Fatalf("members behind a function-local receiver: %v", g04Labels(items))
	}
}

// TestG04CompletionMemberMethod pins method precision: a chained
// method name completes to the receiver record's methods with exact
// arity, not fields.
func TestG04CompletionMemberMethod(t *testing.T) {
	items := g04Items(t, map[string]string{"src/main.can": g04Main}, "src/main.can", ".des|cribe(1)", 2)
	g04WantLabels(t, items, []string{"describe"})
	if item := g04Find(t, items, "describe"); item["kind"] != float64(compMethod) || item["detail"] != "(extra: int) -> int" || item["documentation"] != "method of holder" {
		t.Fatalf("describe candidate wrong: %v", item)
	}
}

// TestG04CompletionConstructor pins constructor precision: a
// constructor name completes to constructible records and errors,
// never to functions, locals, or keywords.
func TestG04CompletionConstructor(t *testing.T) {
	items := g04Items(t, map[string]string{"src/main.can": g04Main}, "src/main.can", "hold|er(callable helper, 0)", 2)
	g04WantLabels(t, items, []string{"all_failed", "choice_option", "holder"})
	if item := g04Find(t, items, "holder"); item["kind"] != float64(compClass) || item["detail"] != "record" {
		t.Fatalf("holder candidate wrong: %v", item)
	}
}

// TestG04CompletionDeclSiteEmpty pins rename's territory: completing
// on a declared name yields nothing instead of competing candidates.
func TestG04CompletionDeclSiteEmpty(t *testing.T) {
	if items := g04Items(t, map[string]string{"src/main.can": g04Main}, "src/main.can", "int doub|led = seed", 2); len(items) != 0 {
		t.Fatalf("candidates on a binding name: %v", g04Labels(items))
	}
	if items := g04Items(t, map[string]string{"src/main.can": g04Main}, "src/main.can", "fn int run|\n", 2); len(items) != 0 {
		t.Fatalf("candidates on a function name: %v", g04Labels(items))
	}
}

// TestG04CompletionTypeContext pins annotation precision: inside a
// type, only type-eligible symbols and the type keywords surface — no
// locals, no functions, no statement keywords.
func TestG04CompletionTypeContext(t *testing.T) {
	items := g04Items(t, map[string]string{"src/main.can": g04Main}, "src/main.can", "in|t doubled = seed + seed", 2)
	g04WantLabels(t, items, []string{"all_failed", "bool", "choice_option", "float", "holder", "int", "standard_failure", "str", "void"})
	g04WantKindLabels(t, items, compKeyword, []string{"bool", "float", "int", "str", "void"})
	if locals := g04Kind(t, items, compVariable); len(locals) != 0 {
		t.Fatalf("locals in type position: %v", locals)
	}
}

// TestG04CompletionErrorBound pins emits precision: an empty bound
// completes to error-eligible names only, with no keywords.
func TestG04CompletionErrorBound(t *testing.T) {
	items := g04Items(t, map[string]string{"src/main.can": g04Main}, "src/main.can", "emits [|]", 2)
	g04WantLabels(t, items, []string{"all_failed"})
	if keywords := g04Kind(t, items, compKeyword); len(keywords) != 0 {
		t.Fatalf("keywords in emits position: %v", keywords)
	}
	items = g04Items(t, map[string]string{"src/main.can": g04Main}, "src/main.can", "fn int helper\n    emits [|]", 2)
	g04WantLabels(t, items, []string{"all_failed"})
}

// TestG04CompletionTopLevel pins declaration-start precision: between
// declarations only starter keywords and type names surface.
func TestG04CompletionTopLevel(t *testing.T) {
	items := g04Items(t, map[string]string{"src/main.can": g04Main}, "src/main.can", "ok value\n\n|fn int combine", 2)
	g04WantKindLabels(t, items, compKeyword, []string{"bool", "error", "float", "fn", "int", "record", "str", "variant", "void"})
	g04Find(t, items, "holder")
	if locals := g04Kind(t, items, compVariable); len(locals) != 0 {
		t.Fatalf("locals at top level: %v", locals)
	}
	if g04Count(items, "helper") != 0 {
		t.Fatalf("function at top level: %v", g04Labels(items))
	}
}

// TestG04CompletionHeader pins the manifest positions: header gaps
// offer the header keywords, the file's declared names, and visible
// import aliases, but no locals.
func TestG04CompletionHeader(t *testing.T) {
	items := g04Items(t, map[string]string{"src/main.can": g04Main}, "src/main.can", "provides [|run, shadowed]", 2)
	g04WantKindLabels(t, items, compKeyword, []string{"as", "package", "provides", "uses"})
	for _, present := range []string{"run", "helper", "holder", "shared"} {
		g04Find(t, items, present)
	}
	if locals := g04Kind(t, items, compVariable); len(locals) != 0 {
		t.Fatalf("locals in header: %v", locals)
	}
}

// TestG04CompletionSignature pins the signature gaps: between a
// function header and its body only section keywords surface.
func TestG04CompletionSignature(t *testing.T) {
	items := g04Items(t, map[string]string{"src/main.can": g04Main}, "src/main.can", "    given|", 2)
	g04WantLabels(t, items, []string{"asserts", "emits", "given", "near"})
}

// TestG04CompletionCallableWith pins the with keyword: after a bare
// callable callee only `with` follows.
func TestG04CompletionCallableWith(t *testing.T) {
	bare := strings.Replace(g04Main, "callable combine with prefix = doubled", "callable combine ", 1)
	items := g04Items(t, map[string]string{"src/main.can": bare}, "src/main.can", "= callable combine |\n", 2)
	g04WantLabels(t, items, []string{"with"})
}

const g04Browser = `package app
    provides [boot]
    uses [browser]

fn int boot
    emits []
    given
        int seed
    asserts
        sample: 1 => ok 1
    ok call browser::mount(seed)
`

// TestG04CompletionQualifiedBrowser pins the C-D catalogue path: a
// qualified name completes to the imported package's members under
// the position's usage with catalogue provenance and no arity guess;
// the package part completes visible import aliases; and an
// unimported package yields nothing.
func TestG04CompletionQualifiedBrowser(t *testing.T) {
	files := map[string]string{"src/main.can": g04Browser}
	items := g04Items(t, files, "src/main.can", "browser::mou|nt(seed)", 2)
	mount := g04Find(t, items, "mount")
	if mount["kind"] != float64(compFunction) || mount["detail"] != "function" || mount["documentation"] != "catalogue" {
		t.Fatalf("mount candidate wrong: %v", mount)
	}
	for _, raw := range items {
		item := g04Object(t, raw)
		if item["documentation"] != "catalogue" {
			t.Fatalf("non-catalogue member behind browser::: %v", item)
		}
		if item["kind"] != float64(compFunction) {
			t.Fatalf("non-function member in call position: %v", item)
		}
	}
	if len(items) == 0 {
		t.Fatalf("browser package completed empty")
	}
	items = g04Items(t, files, "src/main.can", "browser|::mount(seed)", 2)
	g04WantLabels(t, items, []string{"browser"})
	if item := g04Find(t, items, "browser"); item["kind"] != float64(compModule) || item["documentation"] != "catalogue package browser" {
		t.Fatalf("browser alias wrong: %v", item)
	}
	unimported := strings.Replace(g04Browser, "uses [browser]", "uses []", 1)
	if items := g04Items(t, map[string]string{"src/main.can": unimported}, "src/main.can", "browser::mou|nt(seed)", 2); len(items) != 0 {
		t.Fatalf("members of an unimported package: %v", g04Labels(items))
	}
}

// TestG04CompletionStringAndCommentEmpty pins non-code positions:
// inside a string literal or a comment no candidates surface.
func TestG04CompletionStringAndCommentEmpty(t *testing.T) {
	commented := "// leading note\n" + g04Main
	if items := g04Items(t, map[string]string{"src/main.can": commented}, "src/main.can", "// lead|ing note", 2); len(items) != 0 {
		t.Fatalf("candidates inside a comment: %v", g04Labels(items))
	}
	quoted := strings.Replace(g04Main, "    ok extra", `    ok "extra"`, 1)
	if items := g04Items(t, map[string]string{"src/main.can": quoted}, "src/main.can", `ok "extr|a"`, 2); len(items) != 0 {
		t.Fatalf("candidates inside a string: %v", g04Labels(items))
	}
}

// TestG04CompletionWarnedFile pins diagnostics parity: a file carrying
// a check-pipeline warning still completes, and still publishes
// exactly that warning.
func TestG04CompletionWarnedFile(t *testing.T) {
	root := writeServerProject(t, map[string]string{"src/main.can": g01FinalLocal})
	uri := uriFromPath(filepath.Join(root, "src/main.can"))
	line, character := positionOf(t, g01FinalLocal, "ok tot|al")
	frames := runExchange(t, []string{
		didOpen(uri, g01FinalLocal, 1),
		g04Completion(2, uri, line, character),
		`{"jsonrpc":"2.0","method":"exit"}`,
	})
	raw := g01Response(t, frames, 2)
	items, ok := raw.([]any)
	if !ok || len(items) == 0 {
		t.Fatalf("warned file declined completion: %v", raw)
	}
	g04Find(t, items, "total")
	last := lastPublishFor(frames, uri)
	if last == nil {
		t.Fatalf("no publish in %v", frames)
	}
	diags := diagnosticsOf(t, last)
	if len(diags) != 1 || diags[0]["severity"] != 2.0 {
		t.Fatalf("warned file misdiagnosed beside completion: %v", diags)
	}
}

// TestG04CompletionDeterministic pins statelessness: the same request
// twice yields byte-identical results, so no mutable server state
// leaks between queries.
func TestG04CompletionDeterministic(t *testing.T) {
	root := writeServerProject(t, map[string]string{"src/main.can": g04Main})
	uri := uriFromPath(filepath.Join(root, "src/main.can"))
	line, character := positionOf(t, g04Main, "ok call action(doub|led)")
	frames := runExchange(t, []string{
		didOpen(uri, g04Main, 1),
		g04Completion(2, uri, line, character),
		g04Completion(3, uri, line, character),
		`{"jsonrpc":"2.0","method":"exit"}`,
	})
	first, err := json.Marshal(g01Response(t, frames, 2))
	if err != nil {
		t.Fatal(err)
	}
	second, err := json.Marshal(g01Response(t, frames, 3))
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatalf("completion not deterministic:\n%s\n%s", first, second)
	}
}

// TestG04CompletionNoUnselectedKeywords pins the keyword obligation:
// every context offers only selected-surface keywords, and none of
// the unselected iteration or foreign words the inactive grammars
// would have contributed.
func TestG04CompletionNoUnselectedKeywords(t *testing.T) {
	unselected := []string{"while", "loop", "for", "repeat", "until", "each", "iterate", "break", "continue", "return", "yield", "async", "await", "where", "select", "function", "class", "struct", "enum", "interface", "import", "export", "let", "var", "const", "new", "this", "self", "super", "extends", "private", "public", "static"}
	files := map[string]string{"src/main.can": g04Main}
	sweep := map[string]string{
		"general":   "ok call action(doub|led)",
		"type":      "in|t doubled = seed + seed",
		"top":       "ok value\n\n|fn int combine",
		"signature": "    given|",
		"header":    "provides [|run, shadowed]",
		"callee":    "call act|ion(doubled)",
		"member":    "shared.ta|g",
		"withPin":   "with pre|fix = doubled",
	}
	id := 2
	for context, needle := range sweep {
		items := g04Items(t, files, "src/main.can", needle, id)
		id++
		for _, item := range g04Kind(t, items, compKeyword) {
			label, _ := item["label"].(string)
			for _, word := range unselected {
				if label == word {
					t.Fatalf("unselected keyword %q in %s context", word, context)
				}
			}
		}
	}
	items := g04Items(t, files, "src/main.can", "ok call action(doub|led)", id)
	g04WantKindLabels(t, items, compKeyword, []string{"and", "call", "callable", "do", "false", "is", "match", "not", "ok", "or", "relay", "true"})
}
