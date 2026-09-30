package main

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// G05: validated safe rename over the editor wire. Rename resolves the
// token at the cursor to its binding identity through the G03 project
// index and rewrites every occurrence sharing that identity —
// declaration site included — as one atomic WorkspaceEdit. Shadowed
// or same-spelled bindings in other scopes are different identities
// and stay untouched; explicit with pins rename with the callee's
// near parameter while implicit fallback captures keep their own
// caller-scope identity. Every rename validates through an isolated overlay:
// existing independent errors may remain at mapped locations, no new error
// may appear, and renamed tokens must re-resolve to exactly one identity.
// Unsafe proposals return an explained protocol refusal.

// g05Fallback keeps one unlisted near-input on fallback lookup: the
// caller's own `prefix` binding satisfies combine's near parameter by
// name. Renaming either side must decline rather than silently break
// the coupling.
const g05Fallback = `package app
    provides [run]
    uses []

fn int combine
    emits {}
    given
        near int prefix
        int value
    asserts
        sample: 3, 4 => ok 7
    ok prefix + value

fn int run
    emits {}
    given
        int seed
    asserts
        sample: 1 => ok 3
    int prefix = seed + 1
    callable int (int) emits {} action = callable combine
    ok call action(seed)
`

func g05Rename(id int, uri string, line, character int, newName string) string {
	return fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"textDocument/rename","params":{"textDocument":{"uri":%s},"position":{"line":%d,"character":%d},"newName":%s}}`,
		id, jsonQuote(uri), line, character, jsonQuote(newName))
}

// g05Raw requests a rename at a needle cursor and returns the raw wire
// result or an explained protocol refusal.
func g05Raw(t *testing.T, files map[string]string, open, needle, newName string, id int) (map[string]string, any) {
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
		g05Rename(id, uris[open], line, character, newName),
		`{"jsonrpc":"2.0","method":"exit"}`,
	})
	for _, frame := range frames {
		if frame["id"] == float64(id) {
			if err, ok := frame["error"].(map[string]any); ok {
				return texts, &g05Refusal{code: err["code"], message: err["message"]}
			}
		}
	}
	return texts, g01Response(t, frames, float64(id))
}

// Versioned document edits are the only accepted atomic rename shape.
func g05DocumentChanges(t *testing.T, result map[string]any) map[string]any {
	t.Helper()
	documents, ok := result["documentChanges"].([]any)
	if !ok {
		t.Fatalf("rename has no versioned documentChanges: %v", result)
	}
	changes := map[string]any{}
	for _, raw := range documents {
		change, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("invalid document edit: %v", raw)
		}
		doc, ok := change["textDocument"].(map[string]any)
		if !ok {
			t.Fatalf("missing textDocument: %v", change)
		}
		uri, ok := doc["uri"].(string)
		if !ok || uri == "" {
			t.Fatalf("missing edit URI: %v", doc)
		}
		if _, hasVersion := doc["version"]; !hasVersion {
			t.Fatalf("edit lacks explicit version: %v", doc)
		}
		if _, duplicate := changes[uri]; duplicate {
			t.Fatalf("multiple edit batches for %s", uri)
		}
		edits, ok := change["edits"].([]any)
		if !ok {
			t.Fatalf("invalid edits: %v", change)
		}
		changes[uri] = edits
	}
	return changes
}

// g05Edits renders a rename result as sorted "file:line:token=>new"
// rows so tests pin the exact occurrence set across files.
func g05Edits(t *testing.T, texts map[string]string, raw any) []string {
	t.Helper()
	result, ok := raw.(map[string]any)
	if !ok {
		t.Fatalf("rename not an object: %v", raw)
	}
	changes := g05DocumentChanges(t, result)
	uris := make([]string, 0, len(changes))
	for uri := range changes {
		uris = append(uris, uri)
	}
	sort.Strings(uris)
	var out []string
	for _, uri := range uris {
		text, ok := texts[uri]
		if !ok {
			t.Fatalf("rename edits outside fixtures: %v", uri)
		}
		edits, ok := changes[uri].([]any)
		if !ok {
			t.Fatalf("edits not an array: %v", changes[uri])
		}
		name := uri[strings.LastIndex(uri, "/")+1:]
		lines := strings.Split(text, "\n")
		for _, rawEdit := range edits {
			edit, ok := rawEdit.(map[string]any)
			if !ok {
				t.Fatalf("edit not an object: %v", rawEdit)
			}
			rng, _ := edit["range"].(map[string]any)
			start, _ := rng["start"].(map[string]any)
			end, _ := rng["end"].(map[string]any)
			if start["line"] != end["line"] {
				t.Fatalf("edit spans lines: %v", rng)
			}
			line := int(start["line"].(float64))
			token := lines[line][int(start["character"].(float64)):int(end["character"].(float64))]
			newText, _ := edit["newText"].(string)
			out = append(out, fmt.Sprintf("%s:%d:%s=>%s", name, line, token, newText))
		}
	}
	return out
}

func g05Want(t *testing.T, got, want []string) {
	t.Helper()
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("rename edits:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

type g05Refusal struct{ code, message any }

func g05WantRefusal(t *testing.T, texts map[string]string, raw any, what string) {
	t.Helper()
	refusal, ok := raw.(*g05Refusal)
	if !ok {
		t.Fatalf("%s must return an explained rename refusal, got %v", what, raw)
	}
	message, _ := refusal.message.(string)
	if (refusal.code != -32803.0 && refusal.code != -32602.0) || message == "" {
		t.Fatalf("%s gave an unhelpful failure: %+v", what, refusal)
	}
}

func g05RefusalFrame(t *testing.T, frames []map[string]any, id float64) {
	t.Helper()
	for _, frame := range frames {
		if frame["id"] == id {
			err, ok := frame["error"].(map[string]any)
			if !ok {
				t.Fatalf("rename was not refused: %v", frame)
			}
			g05WantRefusal(t, nil, &g05Refusal{code: err["code"], message: err["message"]}, "unsafe rename")
			return
		}
	}
	t.Fatalf("no response to %v", id)
}

// g05Apply splices a rename result into the fixture texts, returning
// the edited buffers keyed by file name.
func g05Apply(t *testing.T, texts map[string]string, raw any) map[string]string {
	t.Helper()
	result, ok := raw.(map[string]any)
	if !ok {
		t.Fatalf("rename not an object: %v", raw)
	}
	changes := g05DocumentChanges(t, result)
	out := map[string]string{}
	for uri, text := range texts {
		name := uri[strings.LastIndex(uri, "/")+1:]
		out[name] = text
	}
	for uri, rawEdits := range changes {
		text, ok := texts[uri]
		if !ok {
			t.Fatalf("rename edits outside fixtures: %v", uri)
		}
		name := uri[strings.LastIndex(uri, "/")+1:]
		type span struct {
			start, end int
			newText    string
		}
		var spans []span
		lines := strings.Split(text, "\n")
		offsets := make([]int, len(lines)+1)
		for i, content := range lines {
			offsets[i+1] = offsets[i] + len(content) + 1
		}
		for _, rawEdit := range rawEdits.([]any) {
			edit := rawEdit.(map[string]any)
			rng := edit["range"].(map[string]any)
			start := rng["start"].(map[string]any)
			end := rng["end"].(map[string]any)
			line := int(start["line"].(float64))
			spans = append(spans, span{
				start:   offsets[line] + int(start["character"].(float64)),
				end:     offsets[line] + int(end["character"].(float64)),
				newText: edit["newText"].(string),
			})
		}
		sort.Slice(spans, func(i, j int) bool { return spans[i].start < spans[j].start })
		var rebuilt strings.Builder
		cursor := 0
		for _, s := range spans {
			rebuilt.WriteString(text[cursor:s.start])
			rebuilt.WriteString(s.newText)
			cursor = s.end
		}
		rebuilt.WriteString(text[cursor:])
		out[name] = rebuilt.String()
	}
	return out
}

// TestG05InitializeAdvertisesRename pins the capability: editors only
// offer rename when the server declares it, alongside the G01–G04
// surface.
func TestG05InitializeAdvertisesRename(t *testing.T) {
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
	if provider, ok := capabilities["renameProvider"].(map[string]any); !ok || provider["prepareProvider"] != true {
		t.Fatalf("rename not advertised: %v", capabilities)
	}
	if !g01FullSync(capabilities) || capabilities["definitionProvider"] != true || capabilities["documentFormattingProvider"] != true || capabilities["hoverProvider"] != true || capabilities["referencesProvider"] != true {
		t.Fatalf("existing capabilities regressed: %v", capabilities)
	}
	completion, ok := capabilities["completionProvider"].(map[string]any)
	if !ok {
		t.Fatalf("completion capability regressed: %v", capabilities)
	}
	triggers, ok := completion["triggerCharacters"].([]any)
	if !ok || len(triggers) != 2 || triggers[0] != "." || triggers[1] != ":" {
		t.Fatalf("completion triggers regressed: %v", completion)
	}
}

// TestG05RenameLocalVariable pins AU-LSP-rename on a body-local: the
// declaration, the nested with-value capture, and the call use.
func TestG05RenameLocalVariable(t *testing.T) {
	texts, raw := g05Raw(t, map[string]string{"src/main.can": g03Main}, "src/main.can", "call action(doub|led)", "quadrupled", 2)
	g05Want(t, g05Edits(t, texts, raw), []string{
		"main.can:33:doubled=>quadrupled",
		"main.can:34:doubled=>quadrupled",
		"main.can:35:doubled=>quadrupled",
	})
}

// TestG05RenameSameSpelledLocalsStayDistinct pins binding identity
// across scopes: each `value` input renames only its own function.
func TestG05RenameSameSpelledLocalsStayDistinct(t *testing.T) {
	texts, raw := g05Raw(t, map[string]string{"src/main.can": g03Main}, "src/main.can", "ok val|ue\n\nfn int combine", "amount", 2)
	g05Want(t, g05Edits(t, texts, raw), []string{
		"main.can:13:value=>amount",
		"main.can:16:value=>amount",
	})
	texts, raw = g05Raw(t, map[string]string{"src/main.can": g03Main}, "src/main.can", "ok prefix + val|ue", "other", 2)
	g05Want(t, g05Edits(t, texts, raw), []string{
		"main.can:22:value=>other",
		"main.can:25:value=>other",
	})
}

// TestG05RenameShadowedBinding pins scope identity under shadowing:
// the match rebind renames only its own uses, and the outer binding
// keeps only its own.
func TestG05RenameShadowedBinding(t *testing.T) {
	texts, raw := g05Raw(t, map[string]string{"src/main.can": g03Main}, "src/main.can", "=> tot|al + seed", "subtotal", 2)
	g05Want(t, g05Edits(t, texts, raw), []string{
		"main.can:46:total=>subtotal",
		"main.can:46:total=>subtotal",
	})
	texts, raw = g05Raw(t, map[string]string{"src/main.can": g03Main}, "src/main.can", "int tot|al = seed\n    int widened", "gross", 2)
	g05Want(t, g05Edits(t, texts, raw), []string{
		"main.can:43:total=>gross",
		"main.can:44:total=>gross",
	})
}

// TestG05RenameNestedCapture pins captures from an enclosing scope:
// the input renames at the declaration, the body, and the nested
// match arm.
func TestG05RenameNestedCapture(t *testing.T) {
	texts, raw := g05Raw(t, map[string]string{"src/main.can": g03Main}, "src/main.can", "int se|ed\n    asserts\n        sample: 1 => ok 2", "grain", 2)
	g05Want(t, g05Edits(t, texts, raw), []string{
		"main.can:40:seed=>grain",
		"main.can:43:seed=>grain",
		"main.can:45:seed=>grain",
		"main.can:46:seed=>grain",
	})
}

// TestG05RenameWithBindingToCalleeParameter pins AU-Q3-rename: the
// explicit with pin renames with the callee's near input —
// declaration, body use, and pin — and the applied edit checks clean.
func TestG05RenameWithBindingToCalleeParameter(t *testing.T) {
	texts, raw := g05Raw(t, map[string]string{"src/main.can": g03Main}, "src/main.can", "with pre|fix = doubled", "label", 2)
	g05Want(t, g05Edits(t, texts, raw), []string{
		"main.can:21:prefix=>label",
		"main.can:25:prefix=>label",
		"main.can:34:prefix=>label",
	})
	edited := g05Apply(t, texts, raw)
	applied := map[string]string{"src/main.can": edited["main.can"]}
	root := writeServerProject(t, applied)
	uri := uriFromPath(filepath.Join(root, "src/main.can"))
	frames := runExchange(t, []string{
		didOpen(uri, applied["src/main.can"], 1),
		`{"jsonrpc":"2.0","method":"exit"}`,
	})
	last := lastPublishFor(frames, uri)
	if last == nil {
		t.Fatalf("no publish in %v", frames)
	}
	// The fixture carries one pre-existing inert-initialization error
	// (the module-level callable reference); the applied rename must
	// add nothing beside it.
	diags := diagnosticsOf(t, last)
	if len(diags) != 1 || diags[0]["severity"] != 1.0 {
		t.Fatalf("applied with rename misdiagnosed: %v", diags)
	}
	if message, _ := diags[0]["message"].(string); !strings.Contains(message, "outside inert initialization") {
		t.Fatalf("applied with rename changed the diagnosis: %v", diags)
	}
}

// TestG05RenameCallbackCallee pins the callback scenario: the callable
// reference's callee renames with its declaration.
func TestG05RenameCallbackCallee(t *testing.T) {
	texts, raw := g05Raw(t, map[string]string{"src/main.can": g03Main}, "src/main.can", "callable help|er, 0", "assistant", 2)
	g05Want(t, g05Edits(t, texts, raw), []string{
		"main.can:8:helper=>assistant",
		"main.can:10:helper=>assistant",
	})
}

// TestG05RenameSharedRecordField pins the shared-record scenario: a
// field use behind a module value renames with its declaration.
func TestG05RenameSharedRecordField(t *testing.T) {
	texts, raw := g05Raw(t, map[string]string{"src/main.can": g03Main}, "src/main.can", "shared.ta|g", "label", 2)
	g05Want(t, g05Edits(t, texts, raw), []string{
		"main.can:6:tag=>label",
		"main.can:35:tag=>label",
	})
}

// TestG05RenameCrossFile pins project scope: a use in a second file
// joins the rename.
func TestG05RenameCrossFile(t *testing.T) {
	texts, raw := g05Raw(t, map[string]string{"src/main.can": g03Main, "src/second.can": g03Second}, "src/main.can", "callable help|er, 0", "assistant", 2)
	g05Want(t, g05Edits(t, texts, raw), []string{
		"main.can:8:helper=>assistant",
		"main.can:10:helper=>assistant",
		"second.can:10:helper=>assistant",
	})
}

// TestG05RenameCaptureDeclines pins the no-capture rule in both
// directions: a new spelling that would resolve a renamed use to
// another binding, or merge another binding's uses into the renamed
// group, declines — even where the re-check alone would pass.
func TestG05RenameCaptureDeclines(t *testing.T) {
	files := map[string]string{"src/main.can": g03Main}
	texts, raw := g05Raw(t, files, "src/main.can", "=> tot|al + seed", "seed", 2)
	g05WantRefusal(t, texts, raw, "inner-total-to-seed")
	texts, raw = g05Raw(t, files, "src/main.can", "int se|ed\n    asserts\n        sample: 1 => ok 2", "total", 2)
	g05WantRefusal(t, texts, raw, "outer-seed-to-total")
	// The renamed outer binding would steal the arm's later `seed`
	// uses through its new shadowing position.
	texts, raw = g05Raw(t, files, "src/main.can", "int tot|al = seed\n    int widened", "seed", 2)
	g05WantRefusal(t, texts, raw, "outer-total-to-stolen-seed")
}

// TestG05RenameAdmissibleSpellings pins renames the validator must not
// over-decline: a shadowed-outer spelling where no later use is
// stolen (the renamed binding shadows consistently, the initializer
// still sees the input), a contextual word, and a record-field
// spelling (members resolve qualified, locals unqualified).
func TestG05RenameAdmissibleSpellings(t *testing.T) {
	files := map[string]string{"src/main.can": g03Main}
	texts, raw := g05Raw(t, files, "src/main.can", "int doub|led = seed", "seed", 2)
	g05Want(t, g05Edits(t, texts, raw), []string{
		"main.can:33:doubled=>seed",
		"main.can:34:doubled=>seed",
		"main.can:35:doubled=>seed",
	})
	texts, raw = g05Raw(t, files, "src/main.can", "int doub|led = seed", "status", 2)
	g05Want(t, g05Edits(t, texts, raw), []string{
		"main.can:33:doubled=>status",
		"main.can:34:doubled=>status",
		"main.can:35:doubled=>status",
	})
	texts, raw = g05Raw(t, files, "src/main.can", "int doub|led = seed", "tag", 2)
	g05Want(t, g05Edits(t, texts, raw), []string{
		"main.can:33:doubled=>tag",
		"main.can:34:doubled=>tag",
		"main.can:35:doubled=>tag",
	})
}

// Exported rename updates the provides occurrence atomically with its declaration.
func TestG05RenameProvidedName(t *testing.T) {
	texts, raw := g05Raw(t, map[string]string{"src/main.can": g03Main}, "src/main.can", "fn int r|un", "start", 2)
	g05Want(t, g05Edits(t, texts, raw), []string{"main.can:1:run=>start", "main.can:27:run=>start"})
	edited := g05Apply(t, texts, raw)["main.can"]
	if !strings.Contains(edited, "provides [start, shadowed]") || !strings.Contains(edited, "fn int start") {
		t.Fatalf("export was left stale: %s", edited)
	}
}

// TestG05EditorWorkflow pins AU-LSP-rename: the full ordered authoring
// flow — format (G01), inspect the shared-record contract (G02),
// find the callback's references (G03), complete the caller (G04),
// rename the callback and change the shared record (G05) — then
// applies both renames and verifies the repaired project carries no
// new diagnosis.
func TestG05EditorWorkflow(t *testing.T) {
	// G01: the true-first buffer formats to one whole-document edit
	// with the arms canonicalized false-first.
	formatRoot := writeServerProject(t, map[string]string{"src/pick.can": g01TrueFirst})
	formatURI := uriFromPath(filepath.Join(formatRoot, "src/pick.can"))
	frames := runExchange(t, []string{
		didOpen(formatURI, g01TrueFirst, 1),
		g01Formatting(10, formatURI),
		`{"jsonrpc":"2.0","method":"exit"}`,
	})
	formatted, ok := g01Response(t, frames, 10).([]any)
	if !ok || len(formatted) != 1 {
		t.Fatalf("workflow format unexpected: %v", g01Response(t, frames, 10))
	}
	newText, _ := formatted[0].(map[string]any)["newText"].(string)
	if falseArm, trueArm := strings.Index(newText, "false => ok 0"), strings.Index(newText, "        true => ok 1"); falseArm < 0 || trueArm < falseArm {
		t.Fatalf("workflow format not canonical: %q", newText)
	}

	files := map[string]string{"src/main.can": g03Main}
	root := writeServerProject(t, files)
	mainURI := uriFromPath(filepath.Join(root, "src/main.can"))
	hoverLine, hoverChar := positionOf(t, g03Main, "shared.ta|g")
	refLine, refChar := positionOf(t, g03Main, "callable help|er, 0")
	compLine, compChar := positionOf(t, g03Main, "ok call action(doub|led)")
	renameLine, renameChar := positionOf(t, g03Main, "callable help|er, 0")
	frames = runExchange(t, []string{
		didOpen(mainURI, g03Main, 1),
		g02Hover(11, mainURI, hoverLine, hoverChar),
		g03References(12, mainURI, refLine, refChar, true),
		g04Completion(13, mainURI, compLine, compChar),
		g05Rename(14, mainURI, renameLine, renameChar, "assistant"),
		`{"jsonrpc":"2.0","method":"exit"}`,
	})
	// G02: the shared-record field hovers with a markdown contract.
	hover, ok := g01Response(t, frames, 11).(map[string]any)
	if !ok {
		t.Fatalf("workflow hover declined: %v", g01Response(t, frames, 11))
	}
	contents, ok := hover["contents"].(map[string]any)
	if !ok || contents["kind"] != "markdown" {
		t.Fatalf("workflow hover not markdown: %v", hover)
	}
	if value, _ := contents["value"].(string); value == "" {
		t.Fatalf("workflow hover empty: %v", hover)
	}
	// G03: the callback resolves to declaration plus reference.
	texts := map[string]string{mainURI: g03Main}
	locations, ok := g01Response(t, frames, 12).([]any)
	if !ok {
		t.Fatalf("workflow references declined")
	}
	g03Want(t, g03Tokens(t, texts, locations), []string{
		"main.can:8:helper",
		"main.can:10:helper",
	})
	// G04: the caller completes with its locals.
	items, ok := g01Response(t, frames, 13).([]any)
	if !ok {
		t.Fatalf("workflow completion declined")
	}
	for _, label := range []string{"action", "doubled", "seed"} {
		g04Find(t, items, label)
	}
	// G05: the callback renames as one atomic edit.
	renamed, ok := g01Response(t, frames, 14).(map[string]any)
	if !ok {
		t.Fatalf("workflow rename declined")
	}
	g05Want(t, g05Edits(t, texts, renamed), []string{
		"main.can:8:helper=>assistant",
		"main.can:10:helper=>assistant",
	})

	// Apply the callback rename, then change the shared record on the
	// repaired buffer; both applies must diagnose nothing new.
	edited := g05Apply(t, texts, renamed)["main.can"]
	fieldLine, fieldChar := positionOf(t, edited, "shared.ta|g")
	frames = runExchange(t, []string{
		didOpen(mainURI, g03Main, 1),
		didChange(mainURI, edited, 2),
		g05Rename(20, mainURI, fieldLine, fieldChar, "label"),
		`{"jsonrpc":"2.0","method":"exit"}`,
	})
	field := g01Response(t, frames, 20)
	g05Want(t, g05Edits(t, map[string]string{mainURI: edited}, field), []string{
		"main.can:6:tag=>label",
		"main.can:35:tag=>label",
	})
	repaired := g05Apply(t, map[string]string{mainURI: edited}, field)["main.can"]
	callLine, callChar := positionOf(t, repaired, "callable assist|ant, 0")
	frames = runExchange(t, []string{
		didOpen(mainURI, repaired, 1),
		g03References(30, mainURI, callLine, callChar, true),
		`{"jsonrpc":"2.0","method":"exit"}`,
	})
	g03Want(t, g03Tokens(t, map[string]string{mainURI: repaired}, g01Response(t, frames, 30).([]any)), []string{
		"main.can:8:assistant",
		"main.can:10:assistant",
	})
	last := lastPublishFor(frames, mainURI)
	if last == nil {
		t.Fatalf("no publish in %v", frames)
	}
	diags := diagnosticsOf(t, last)
	if len(diags) != 1 || diags[0]["severity"] != 1.0 {
		t.Fatalf("repaired project misdiagnosed: %v", diags)
	}
	if message, _ := diags[0]["message"].(string); !strings.Contains(message, "outside inert initialization") {
		t.Fatalf("repaired project changed the diagnosis: %v", diags)
	}
}

// TestG05RenameFallbackCouplingDeclines pins the AU-Q3-rename fallback
// half: the callee's near parameter and the caller's fallback binding
// are distinct identities, and renaming either side declines rather
// than silently breaking the name coupling.
func TestG05RenameFallbackCouplingDeclines(t *testing.T) {
	files := map[string]string{"src/main.can": g05Fallback}
	texts, raw := g05Raw(t, files, "src/main.can", "near int pre|fix", "label", 2)
	g05WantRefusal(t, texts, raw, "near-param-away-from-fallback")
	texts, raw = g05Raw(t, files, "src/main.can", "int pre|fix = seed + 1", "base", 2)
	g05WantRefusal(t, texts, raw, "fallback-away-from-near-param")
}

// TestG05RenameInvalidNameDeclines pins spelling validation: hard
// keywords, non-identifier spellings, and the wildcard decline.
func TestG05RenameInvalidNameDeclines(t *testing.T) {
	files := map[string]string{"src/main.can": g03Main}
	for _, name := range []string{"", "with", "call", "match", "With", "9lives", "has space", "pkg::x", "_", "trailing_"} {
		texts, raw := g05Raw(t, files, "src/main.can", "int doub|led = seed", name, 2)
		g05WantRefusal(t, texts, raw, fmt.Sprintf("invalid-name-%q", name))
	}
}

// TestG05RenameUnresolvedDeclines pins the no-guess rule: a callee the
// checked World cannot resolve, and whitespace, decline to null.
func TestG05RenameUnresolvedDeclines(t *testing.T) {
	broken := strings.Replace(g03Main, "ok call action(doubled) + shared.tag", "ok call missing_fn(doubled)", 1)
	root := writeServerProject(t, map[string]string{"src/main.can": broken})
	uri := uriFromPath(filepath.Join(root, "src/main.can"))
	line, character := positionOf(t, broken, "call missing|_fn(doubled)")
	frames := runExchange(t, []string{
		didOpen(uri, broken, 1),
		g05Rename(2, uri, line, character, "renamed"),
		g05Rename(3, uri, 3, 0, "renamed"),
		`{"jsonrpc":"2.0","method":"exit"}`,
	})
	g05RefusalFrame(t, frames, 2)
	g05RefusalFrame(t, frames, 3)
}

// TestG05RenameSameNameNoOp pins the identity rename: renaming to the
// current spelling succeeds with an empty edit.
func TestG05RenameSameNameNoOp(t *testing.T) {
	texts, raw := g05Raw(t, map[string]string{"src/main.can": g03Main}, "src/main.can", "int doub|led = seed", "doubled", 2)
	_ = texts
	result, ok := raw.(map[string]any)
	if !ok {
		t.Fatalf("same-name rename not an object: %v", raw)
	}
	changes := g05DocumentChanges(t, result)
	if len(changes) != 0 {
		t.Fatalf("same-name rename not an empty edit: %v", raw)
	}
}

// TestG05RenameWarnedFile pins diagnostics parity: a file carrying a
// check-pipeline warning still renames, and still publishes exactly
// that warning.
func TestG05RenameWarnedFile(t *testing.T) {
	root := writeServerProject(t, map[string]string{"src/main.can": g01FinalLocal})
	uri := uriFromPath(filepath.Join(root, "src/main.can"))
	line, character := positionOf(t, g01FinalLocal, "int tot|al = left + right")
	frames := runExchange(t, []string{
		didOpen(uri, g01FinalLocal, 1),
		g05Rename(2, uri, line, character, "sum"),
		`{"jsonrpc":"2.0","method":"exit"}`,
	})
	raw := g01Response(t, frames, 2)
	texts := map[string]string{uri: g01FinalLocal}
	g05Want(t, g05Edits(t, texts, raw), []string{
		"main.can:11:total=>sum",
		"main.can:12:total=>sum",
	})
	last := lastPublishFor(frames, uri)
	if last == nil {
		t.Fatalf("no publish in %v", frames)
	}
	diags := diagnosticsOf(t, last)
	if len(diags) != 1 || diags[0]["severity"] != 2.0 {
		t.Fatalf("warned file misdiagnosed beside rename: %v", diags)
	}
}

// TestG05RenameInvalidParams pins the wire guard: malformed rename
// params answer -32602.
func TestG05RenameInvalidParams(t *testing.T) {
	root := writeServerProject(t, map[string]string{"src/main.can": g03Main})
	uri := uriFromPath(filepath.Join(root, "src/main.can"))
	frames := runExchange(t, []string{
		didOpen(uri, g03Main, 1),
		`{"jsonrpc":"2.0","id":2,"method":"textDocument/rename","params":{"textDocument":{"uri":` + jsonQuote(uri) + `},"position":"nope","newName":"x"}}`,
		`{"jsonrpc":"2.0","method":"exit"}`,
	})
	var found map[string]any
	for _, f := range frames {
		if f["id"] == 2.0 {
			found = f
		}
	}
	if found == nil {
		t.Fatalf("no response for id 2 in %v", frames)
	}
	errObj, ok := found["error"].(map[string]any)
	if !ok || errObj["code"] != -32602.0 {
		t.Fatalf("malformed rename params not rejected: %v", found)
	}
}
