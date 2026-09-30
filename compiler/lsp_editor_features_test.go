package main

import (
	"fmt"
	"github.com/veighnsche/can-lang/compiler/internal/driver"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"path/filepath"
	"strings"
	"testing"
)

func eRequest(id int, method, uri, extra string) string {
	return fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":%q,"params":{"textDocument":{"uri":%s}%s}}`, id, method, jsonQuote(uri), extra)
}

func TestEditorCapabilitiesAndSemanticProducts(t *testing.T) {
	root := writeServerProject(t, map[string]string{"src/main.can": g03Main})
	uri := uriFromPath(filepath.Join(root, "src/main.can"))
	frames := runExchange(t, []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
		didOpen(uri, g03Main, 1),
		eRequest(2, "textDocument/documentSymbol", uri, ""),
		eRequest(3, "textDocument/semanticTokens/full", uri, ""),
		eRequest(4, "textDocument/foldingRange", uri, ""),
		`{"jsonrpc":"2.0","method":"exit"}`,
	})
	init := g01Response(t, frames, 1).(map[string]any)
	caps := init["capabilities"].(map[string]any)
	for _, key := range []string{"documentSymbolProvider", "workspaceSymbolProvider", "semanticTokensProvider", "foldingRangeProvider", "inlayHintProvider", "signatureHelpProvider", "codeActionProvider", "documentRangeFormattingProvider", "documentOnTypeFormattingProvider"} {
		if caps[key] == nil || caps[key] == false {
			t.Fatalf("missing capability %s: %v", key, caps)
		}
	}
	symbols, ok := g01Response(t, frames, 2).([]any)
	if !ok || len(symbols) < 3 {
		t.Fatalf("document symbols: %v", g01Response(t, frames, 2))
	}
	var record map[string]any
	for _, raw := range symbols {
		symbol := raw.(map[string]any)
		if symbol["name"] == "holder" {
			record = symbol
		}
	}
	if record == nil || len(record["children"].([]any)) != 2 {
		t.Fatalf("record field symbols: %v", record)
	}
	tokens := g01Response(t, frames, 3).(map[string]any)["data"].([]any)
	if len(tokens) < 10 || len(tokens)%5 != 0 {
		t.Fatalf("semantic token data: %v", tokens)
	}
	line, start := 0, 0
	for i := 0; i < len(tokens); i += 5 {
		dl := int(tokens[i].(float64))
		ds := int(tokens[i+1].(float64))
		if dl > 0 {
			line += dl
			start = ds
		} else {
			start += ds
		}
		if dl < 0 || ds < 0 || int(tokens[i+2].(float64)) <= 0 || line < 0 || start < 0 {
			t.Fatalf("invalid token delta at %d: %v", i, tokens[i:i+5])
		}
	}
	folds, ok := g01Response(t, frames, 4).([]any)
	if !ok || len(folds) < 3 {
		t.Fatalf("folds: %v", g01Response(t, frames, 4))
	}
}

func TestEditorLocalNavigationAndPrepareRename(t *testing.T) {
	root := writeServerProject(t, map[string]string{"src/main.can": g03Main})
	uri := uriFromPath(filepath.Join(root, "src/main.can"))
	line, column := positionOf(t, g03Main, "call action(doub|led)")
	position := fmt.Sprintf(`,"position":{"line":%d,"character":%d}`, line, column)
	frames := runExchange(t, []string{
		didOpen(uri, g03Main, 1),
		eRequest(1, "textDocument/definition", uri, position),
		eRequest(2, "textDocument/hover", uri, position),
		eRequest(3, "textDocument/prepareRename", uri, position),
		`{"jsonrpc":"2.0","method":"exit"}`,
	})
	if g01Response(t, frames, 1) == nil {
		t.Fatal("local definition declined")
	}
	hover := g01Response(t, frames, 2).(map[string]any)["contents"].(map[string]any)["value"].(string)
	if !strings.Contains(hover, "int doubled") {
		t.Fatalf("local hover: %s", hover)
	}
	prepare := g01Response(t, frames, 3).(map[string]any)
	if prepare["placeholder"] != "doubled" {
		t.Fatalf("prepare rename: %v", prepare)
	}
}

func TestEditorIncompleteCallSignatureAndMemberCompletion(t *testing.T) {
	broken := strings.Replace(g03Main, "ok call action(doubled) + shared.tag", "ok call helper(", 1)
	root := writeServerProject(t, map[string]string{"src/main.can": broken})
	uri := uriFromPath(filepath.Join(root, "src/main.can"))
	line, column := positionOf(t, broken, "ok call helper(|")
	pos := fmt.Sprintf(`,"position":{"line":%d,"character":%d}`, line, column)
	frames := runExchange(t, []string{didOpen(uri, broken, 1), eRequest(1, "textDocument/signatureHelp", uri, pos), `{"jsonrpc":"2.0","method":"exit"}`})
	if g01Response(t, frames, 1) == nil {
		t.Fatalf("incomplete call signature declined: %v", frames)
	}
}

func TestEditorIncompleteConstructorAndMethodSignatures(t *testing.T) {
	for _, tc := range []struct{ name, text, needle, want string }{
		{"constructor", strings.Replace(g04Main, "ok call action(doubled) + shared.tag + call shared().describe(1)", "ok holder(", 1), "ok holder(|", "holder("},
		{"method", strings.Replace(g04Main, "ok call action(doubled) + shared.tag + call shared().describe(1)", "ok call shared().describe(", 1), ".describe(|", "describe("},
	} {
		t.Run(tc.name, func(t *testing.T) {
			uri := "untitled:can-incomplete-" + tc.name
			s := newLSPServer()
			s.open(uri, tc.text, 1)
			line, column := positionOf(t, tc.text, tc.needle)
			result, ok := s.signatureHelp(uri, featurePosition{Line: line, Character: column}).(map[string]any)
			if !ok {
				t.Fatalf("incomplete %s signature declined: %v", tc.name, result)
			}
			label := result["signatures"].([]any)[0].(map[string]any)["label"].(string)
			if !strings.HasPrefix(label, tc.want) {
				t.Fatalf("wrong incomplete signature %q, want %q", label, tc.want)
			}
		})
	}
}

func TestEditorUnresolvedProvideHasNoSemanticIdentity(t *testing.T) {
	text := strings.Replace(g03Main, "provides [run, shadowed]", "provides [run, shadowed, ghost]", 1)
	uri := "untitled:can-unresolved-export"
	s := newLSPServer()
	s.open(uri, text, 1)
	line, column := positionOf(t, text, "shadowed, gh|ost")
	if got := s.prepareRename(uri, featurePosition{Line: line, Character: column}); got != nil {
		t.Fatalf("unresolved provide was renameable: %v", got)
	}
	data := s.semanticTokens(uri).(map[string]any)["data"].([]int)
	position := 0
	start := 0
	for i := 0; i < len(data); i += 5 {
		position += data[i]
		if data[i] > 0 {
			start = data[i+1]
		} else {
			start += data[i+1]
		}
		if position == line && start == column-2 {
			t.Fatalf("unresolved provide has semantic role: %v", data[i:i+5])
		}
	}
}

func TestEditorLocalRecordInputFieldNavigationAndCompletion(t *testing.T) {
	const text = `package scratch
    provides []
    uses []
record config
    int value
fn int read
    emits {}
    given
        config settings
    asserts
        sample: config(1) => ok 1
    ok settings.value
`
	root := writeServerProject(t, map[string]string{"src/main.can": text})
	uri := uriFromPath(filepath.Join(root, "src/main.can"))
	line, column := positionOf(t, text, "settings.val|ue")
	frames := runExchange(t, []string{didOpen(uri, text, 1),
		eRequest(1, "textDocument/definition", uri, fmt.Sprintf(`,"position":{"line":%d,"character":%d}`, line, column)),
		eRequest(2, "textDocument/references", uri, fmt.Sprintf(`,"position":{"line":%d,"character":%d},"context":{"includeDeclaration":true}`, line, column)),
		eRequest(3, "textDocument/completion", uri, fmt.Sprintf(`,"position":{"line":%d,"character":%d}`, line, column)),
		eRequest(4, "textDocument/hover", uri, fmt.Sprintf(`,"position":{"line":%d,"character":%d}`, line, column)),
		`{"jsonrpc":"2.0","method":"exit"}`})
	if g01Response(t, frames, 1) == nil {
		t.Fatal("checked local receiver field definition declined")
	}
	refs, ok := g01Response(t, frames, 2).([]any)
	if !ok || len(refs) < 2 {
		t.Fatalf("checked local receiver field references: %v", refs)
	}
	items, ok := g01Response(t, frames, 3).([]any)
	if !ok {
		t.Fatalf("checked local receiver completion declined: %v", items)
	}
	found := false
	for _, item := range items {
		if item.(map[string]any)["label"] == "value" {
			found = true
		}
	}
	if !found {
		t.Fatalf("checked local receiver field absent from completion: %v", items)
	}
	hover := g01Response(t, frames, 4).(map[string]any)["contents"].(map[string]any)["value"].(string)
	if !strings.Contains(hover, "int value") {
		t.Fatalf("checked local field hover: %q", hover)
	}
}

func TestEditorIncompleteAndBoundLocalRecordMembers(t *testing.T) {
	const head = `package scratch
    provides []
    uses []
record config
    int value
fn int read
    emits {}
    given
        config settings
    asserts
        sample: config(1) => ok 1
`
	for _, tc := range []struct{ name, body, needle string }{
		{"incomplete_input", "    ok settings.\n", "settings.|"},
		{"checked_binding", "    config local = config(1)\n    ok local.value\n", "local.val|ue"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			text := head + tc.body
			uri := "untitled:can-local-member-" + tc.name
			s := newLSPServer()
			s.open(uri, text, 1)
			line, column := positionOf(t, text, tc.needle)
			items := s.completion(uri, line, column)
			found := false
			for _, raw := range items.([]any) {
				if raw.(map[string]any)["label"] == "value" {
					found = true
				}
			}
			if !found {
				t.Fatalf("record member absent from %s completion: %v", tc.name, items)
			}
			if tc.name == "checked_binding" {
				if got := s.definition(uri, line, column); got == nil {
					t.Fatal("checked binding member definition declined")
				}
				if got := s.hover(uri, line, column); got == nil {
					t.Fatal("checked binding member hover declined")
				}
			}
		})
	}
}

func TestEditorIncompleteMemberAndCalleeCompletionEdits(t *testing.T) {
	cases := []struct{ name, text, needle, want string }{
		{"member", strings.Replace(g03Main, "ok call action(doubled) + shared.tag", "ok shared.", 1), "ok shared.|", "tag"},
		{"callee", strings.Replace(g03Main, "ok call action(doubled) + shared.tag", "ok call combi(", 1), "ok call comb|i(", "combine"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := writeServerProject(t, map[string]string{"src/main.can": tc.text})
			uri := uriFromPath(filepath.Join(root, "src/main.can"))
			line, column := positionOf(t, tc.text, tc.needle)
			frames := runExchange(t, []string{didOpen(uri, tc.text, 1), eRequest(1, "textDocument/completion", uri, fmt.Sprintf(`,"position":{"line":%d,"character":%d}`, line, column)), `{"jsonrpc":"2.0","method":"exit"}`})
			items, ok := g01Response(t, frames, 1).([]any)
			if !ok {
				t.Fatalf("completion declined: %v", frames)
			}
			var found map[string]any
			for _, raw := range items {
				item := raw.(map[string]any)
				if item["label"] == tc.want {
					found = item
				}
			}
			if found == nil {
				t.Fatalf("missing %s candidate: %v", tc.want, items)
			}
			edit, ok := found["textEdit"].(map[string]any)
			if !ok || edit["newText"] != tc.want {
				t.Fatalf("missing precise edit: %v", found)
			}
		})
	}
}

func TestEditorWorkspaceSymbolsWithoutOpenDocument(t *testing.T) {
	root := writeServerProject(t, map[string]string{"src/main.can": g03Main})
	frames := runExchange(t, []string{
		fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"rootUri":%s}}`, jsonQuote(uriFromPath(root))),
		`{"jsonrpc":"2.0","id":2,"method":"workspace/symbol","params":{"query":"helper"}}`,
		`{"jsonrpc":"2.0","method":"exit"}`,
	})
	items, ok := g01Response(t, frames, 2).([]any)
	if !ok || len(items) == 0 {
		t.Fatalf("workspace symbol query without open document: %v", g01Response(t, frames, 2))
	}
	if items[0].(map[string]any)["name"] != "helper" {
		t.Fatalf("wrong workspace symbol: %v", items)
	}
}

func TestEditorCodeActionHasVersionedAtomicEdit(t *testing.T) {
	const code = `package app
    provides []
    uses [codec]
fn int number
    emits {codec::invalid_data}
    asserts
        sample: => ok 1
    ok 1
fn int handle
    emits {codec::invalid_data}
    asserts
        sample: => ok 1
    match call number()
        ok int value => ok value
`
	root := writeServerProject(t, map[string]string{"src/main.can": code})
	uri := uriFromPath(filepath.Join(root, "src/main.can"))
	frames := runExchange(t, []string{didOpen(uri, code, 7), eRequest(1, "textDocument/codeAction", uri, `,"range":{"start":{"line":0,"character":0},"end":{"line":14,"character":0}},"context":{"diagnostics":[]}`), `{"jsonrpc":"2.0","method":"exit"}`})
	items, ok := g01Response(t, frames, 1).([]any)
	if !ok || len(items) == 0 {
		t.Fatalf("compiler action missing: %v", g01Response(t, frames, 1))
	}
	edit := items[0].(map[string]any)["edit"].(map[string]any)
	changes := edit["documentChanges"].([]any)
	if len(changes) != 1 || changes[0].(map[string]any)["textDocument"].(map[string]any)["version"] != float64(7) {
		t.Fatalf("unversioned action: %v", changes)
	}
	s := newLSPServer()
	s.open(uri, code, 7)
	snapshot, err := s.snapshot(uri)
	if err != nil {
		t.Fatal(err)
	}
	var target *driver.Diagnostic
	for i := range snapshot.Diagnostics {
		if len(snapshot.Diagnostics[i].Fixes) > 0 {
			target = &snapshot.Diagnostics[i]
			break
		}
	}
	if target == nil {
		t.Fatal("compiler did not attach a fix diagnostic")
	}
	rng := featureRange{Start: featurePosition{Line: target.Line, Character: target.Start}, End: featurePosition{Line: target.EndLine, Character: target.End}}
	identity := featureDiagnostic{Range: rng, Code: target.Code, Message: target.Message}
	if got := s.codeActionsForContext(uri, rng, []featureDiagnostic{identity}).([]any); len(got) == 0 {
		t.Fatal("exact diagnostic range did not discover its repair")
	}
	identity.Message += " stale"
	if got := s.codeActionsForContext(uri, rng, []featureDiagnostic{identity}).([]any); len(got) != 0 {
		t.Fatalf("stale diagnostic identity acquired a repair: %v", got)
	}
}

func TestEditorDuplicateURIsRejectSourceEdits(t *testing.T) {
	root := writeServerProject(t, map[string]string{"src/main.can": g03Main})
	path := filepath.Join(root, "src/main.can")
	uri := uriFromPath(path)
	alias := uriFromPath(filepath.Join(root, "src", "..", "src", "main.can"))
	if alias == uri {
		alias = strings.Replace(uri, "/src/main.can", "/src/../src/main.can", 1)
	}
	frames := runExchange(t, []string{
		didOpen(uri, g03Main, 1), didOpen(alias, g03Main, 2),
		eRequest(1, "textDocument/rangeFormatting", uri, `,"range":{"start":{"line":0,"character":0},"end":{"line":1,"character":0}},"options":{"tabSize":4,"insertSpaces":true}`),
		`{"jsonrpc":"2.0","method":"exit"}`,
	})
	for _, frame := range frames {
		if frame["id"] == float64(1) {
			errObj, ok := frame["error"].(map[string]any)
			if !ok || errObj["code"] != float64(-32803) {
				t.Fatalf("duplicate aliases did not reject edit: %v", frame)
			}
			return
		}
	}
	t.Fatalf("missing rangeFormatting response: %v", frames)
}

func TestEditorScratchCompletionHoverAndFormatting(t *testing.T) {
	const scratch = `package scratch
    provides []
    uses []
fn int helper
    emits {}
    given
        int value
    asserts
        sample: 1 => ok 1
    ok value
`
	uri := "untitled:can-editor-scratch"
	line, column := positionOf(t, scratch, "ok val|ue")
	pos := fmt.Sprintf(`,"position":{"line":%d,"character":%d}`, line, column)
	frames := runExchange(t, []string{
		didOpen(uri, scratch, 1),
		eRequest(1, "textDocument/completion", uri, pos),
		eRequest(2, "textDocument/hover", uri, pos),
		eRequest(3, "textDocument/formatting", uri, `,"options":{"tabSize":4,"insertSpaces":true}`),
		`{"jsonrpc":"2.0","method":"exit"}`,
	})
	if g01Response(t, frames, 1) == nil {
		t.Fatal("scratch completion declined")
	}
	if g01Response(t, frames, 2) == nil {
		t.Fatal("scratch local hover declined")
	}
	if g01Response(t, frames, 3) == nil {
		t.Fatal("scratch formatting declined")
	}
}

func TestEditorCheckedCallableSignatures(t *testing.T) {
	root := writeServerProject(t, map[string]string{"src/main.can": g04Main})
	uri := uriFromPath(filepath.Join(root, "src/main.can"))
	queries := []string{"holder(callable helper, |0)", ".describe(|1)"}
	requests := []string{didOpen(uri, g04Main, 1)}
	for i, needle := range queries {
		line, column := positionOf(t, g04Main, needle)
		requests = append(requests, eRequest(i+1, "textDocument/signatureHelp", uri, fmt.Sprintf(`,"position":{"line":%d,"character":%d}`, line, column)))
	}
	requests = append(requests, `{"jsonrpc":"2.0","method":"exit"}`)
	frames := runExchange(t, requests)
	for i, needle := range queries {
		response := g01Response(t, frames, float64(i+1))
		if response == nil {
			t.Fatalf("signature declined at %s", needle)
		}
		if i == 0 && response.(map[string]any)["activeParameter"] != float64(1) {
			t.Fatalf("constructor signature did not advance after comma: %v", response)
		}
	}
}

func TestEditorCheckedLocalCallbackSignature(t *testing.T) {
	root := writeServerProject(t, map[string]string{"src/main.can": g03Main})
	uri := uriFromPath(filepath.Join(root, "src/main.can"))
	line, column := positionOf(t, g03Main, "call action(|doubled)")
	frames := runExchange(t, []string{didOpen(uri, g03Main, 1), eRequest(1, "textDocument/signatureHelp", uri, fmt.Sprintf(`,"position":{"line":%d,"character":%d}`, line, column)), `{"jsonrpc":"2.0","method":"exit"}`})
	if g01Response(t, frames, 1) == nil {
		t.Fatal("checked local callback signature declined")
	}
}

func TestEditorWithPinSignatureShowsNearContract(t *testing.T) {
	uri := "untitled:can-pin-signature"
	s := newLSPServer()
	s.open(uri, g03Main, 1)
	line, column := positionOf(t, g03Main, "with prefix = doub|led")
	result, ok := s.signatureHelp(uri, featurePosition{Line: line, Character: column}).(map[string]any)
	if !ok {
		t.Fatal("checked with-pin signature declined")
	}
	signature := result["signatures"].([]any)[0].(map[string]any)
	if signature["label"] != "combine(near int prefix, int value)" || result["activeParameter"] != 0 {
		t.Fatalf("with-pin contract wrong: %v", result)
	}
	partial := strings.Replace(g03Main, "callable combine with prefix = doubled", "callable combine with prefix =", 1)
	partialURI := "untitled:can-partial-pin-signature"
	partialServer := newLSPServer()
	partialServer.open(partialURI, partial, 1)
	line, column = positionOf(t, partial, "with prefix =|")
	result, ok = partialServer.signatureHelp(partialURI, featurePosition{Line: line, Character: column}).(map[string]any)
	if !ok || result["activeParameter"] != 0 {
		t.Fatalf("incomplete with-pin value lost signature: %v", result)
	}
}

func TestEditorSignatureDeclinesNonCallableLocalShadow(t *testing.T) {
	const text = `package scratch
    provides []
    uses []
fn int helper
    emits {}
    asserts
        sample: => ok 1
    ok 1
fn int run
    emits {}
    asserts
        sample: => ok 1
    int helper = 1
    ok call helper()
`
	uri := "untitled:can-signature-shadow"
	s := newLSPServer()
	s.open(uri, text, 1)
	line, column := positionOf(t, text, "call helper(|)")
	if got := s.signatureHelp(uri, featurePosition{Line: line, Character: column}); got != nil {
		t.Fatalf("noncallable local borrowed package helper signature: %v", got)
	}
}

func TestEditorIncompleteBodyRetainsCheckedCallableInputSignature(t *testing.T) {
	text := strings.Replace(g03Main, "        int seed\n    asserts\n        sample: 1 => ok 4", "        int seed\n        callable int (int) emits {} callback\n    asserts\n        sample: 1 => ok 4", 1)
	text = strings.Replace(text, "ok call action(doubled) + shared.tag", "ok call callback(", 1)
	uri := "untitled:can-callable-input-recovery"
	s := newLSPServer()
	s.open(uri, text, 1)
	line, column := positionOf(t, text, "call callback(|")
	result, ok := s.signatureHelp(uri, featurePosition{Line: line, Character: column}).(map[string]any)
	if !ok {
		t.Fatalf("sealed callable input lost signature in incomplete body: %v", result)
	}
	label := result["signatures"].([]any)[0].(map[string]any)["label"].(string)
	if label != "callback(int)" {
		t.Fatalf("wrong checked callable input signature: %s", label)
	}
}

func TestRenameDiagnosticMappingDoesNotMutateSnapshot(t *testing.T) {
	d := driver.Diagnostic{File: "/other.can", Line: 0, Start: 0, EndLine: 0, End: 1, Related: []driver.RelatedDiagnostic{{File: "/a.can", Line: 0, Start: 4, EndLine: 0, End: 7}}}
	target := renameTarget{file: "/a.can", span: source.Span{Start: 0, End: 3}, renamed: source.Span{Start: 0, End: 8}}
	mapped, ok := mappedRenameDiagnostic(d, map[string]string{"/a.can": "foo bar"}, map[string]string{"/a.can": "longname bar"}, map[string][]renameTarget{"/a.can": {target}})
	if !ok || mapped.Related[0].Start != 9 || d.Related[0].Start != 4 {
		t.Fatalf("mapping mutated input: mapped=%+v original=%+v", mapped.Related, d.Related)
	}
}

func TestEditorRangeAndOnTypeFormattingStayInsideLine(t *testing.T) {
	const raw = `package scratch
    provides []
    uses []
fn int helper
    emits {}
    given
        int value
    asserts
        sample: 1 => ok 1
    ok  value
`
	uri := "untitled:can-format-scratch"
	s := newLSPServer()
	s.open(uri, raw, 1)
	selected := featureRange{Start: featurePosition{Line: 9, Character: 0}, End: featurePosition{Line: 10, Character: 0}}
	result, ok := s.rangeFormatting(uri, selected).([]any)
	if !ok || len(result) != 1 {
		t.Fatalf("safe line edit declined: %v", result)
	}
	edit := result[0].(map[string]any)
	if strings.Contains(edit["newText"].(string), "ok  value") || !strings.Contains(edit["newText"].(string), "ok value") {
		t.Fatalf("wrong canonical line: %v", edit)
	}
	other := featureRange{Start: featurePosition{Line: 0, Character: 0}, End: featurePosition{Line: 1, Character: 0}}
	if got, ok := s.rangeFormatting(uri, other).([]any); !ok || len(got) != 0 {
		t.Fatalf("unchanged selected line produced an edit: %v", got)
	}
	if got := s.onTypeFormatting(uri, featurePosition{Line: 10, Character: 0}, "\n"); got == nil {
		t.Fatal("on-type safe line edit declined")
	}
}

func TestEditorRangeFormattingPreservesAnotherUnformattedLine(t *testing.T) {
	const raw = `package scratch
    provides []
    uses []
fn int first
    emits {}
    asserts
        sample: => ok 1
    ok  1

fn int second
    emits {}
    asserts
        sample: => ok 2
    ok  2
`
	uri := "untitled:can-format-two-lines"
	s := newLSPServer()
	s.open(uri, raw, 1)
	result, ok := s.rangeFormatting(uri, featureRange{Start: featurePosition{Line: 7}, End: featurePosition{Line: 8}}).([]any)
	if !ok || len(result) != 1 {
		t.Fatalf("isolated line edit declined: %v", result)
	}
	edit := result[0].(map[string]any)
	if edit["newText"] != "    ok 1\n" {
		t.Fatalf("selected line was not canonicalized: %v", edit)
	}
	file, _ := source.New(uri, raw)
	start, _ := file.Offset(source.UTF16Position{Line: 7})
	end, _ := file.Offset(source.UTF16Position{Line: 8})
	candidate := raw[:start] + edit["newText"].(string) + raw[end:]
	if !strings.Contains(candidate, "    ok  2\n") || strings.Contains(candidate, "    ok  1\n") {
		t.Fatalf("edit changed another line or failed its own: %q", candidate)
	}
}

func TestEditorSignatureSeparatorsAndNearInputs(t *testing.T) {
	const simple = `package scratch
    provides []
    uses []
fn int add
    emits {}
    given
        int left
        int right
    asserts
        sample: 1, 2 => ok 3
    ok left + right
fn int run
    emits {}
    asserts
        sample: => ok 3
    ok call add(1,  2)
`
	simpleURI := "untitled:can-signature-separator"
	simpleServer := newLSPServer()
	simpleServer.open(simpleURI, simple, 1)
	commaLine, commaColumn := positionOf(t, simple, "call add(1, | 2)")
	commaResult, ok := simpleServer.signatureHelp(simpleURI, featurePosition{Line: commaLine, Character: commaColumn}).(map[string]any)
	if !ok || commaResult["activeParameter"] != 1 {
		t.Fatalf("call signature did not advance in whitespace after comma: %v", commaResult)
	}

	text := strings.Replace(g03Main, "    ok call action(doubled) + shared.tag", "    ok call action(doubled) + shared.tag + call combine(  doubled)", 1)
	uri := "untitled:can-signature-near"
	s := newLSPServer()
	s.open(uri, text, 1)
	line, column := positionOf(t, text, "call combine(|  doubled)")
	result, ok := s.signatureHelp(uri, featurePosition{Line: line, Character: column}).(map[string]any)
	if !ok {
		t.Fatalf("near declaration signature declined: %v", result)
	}
	signature := result["signatures"].([]any)[0].(map[string]any)
	if signature["label"] != "combine(int value)" || result["activeParameter"] != 0 {
		t.Fatalf("near input leaked into positional signature: %v", result)
	}
	hints := s.inlayHints(uri, featureRange{Start: featurePosition{Line: 0}, End: featurePosition{Line: len(strings.Split(text, "\n")) - 1}}).([]any)
	found := false
	for _, rawHint := range hints {
		if rawHint.(map[string]any)["label"] == "value:" {
			found = true
		}
		if rawHint.(map[string]any)["label"] == "prefix:" {
			t.Fatalf("near input received positional hint: %v", hints)
		}
	}
	if !found {
		t.Fatalf("positional hint missing: %v", hints)
	}

	lexFile, _ := source.New("nested.can", "call choose(pair(1, 2),  3)")
	tokens := syntax.Lex(lexFile).Tokens
	inner := strings.Index(lexFile.Text(), "2)")
	outer := strings.Index(lexFile.Text(), "  3")
	if got := activeArgument(tokens, len("call choose"), len(lexFile.Text()), inner); got != 0 {
		t.Fatalf("nested constructor comma advanced outer parameter: %d", got)
	}
	if got := activeArgument(tokens, len("call choose"), len(lexFile.Text()), outer+1); got != 1 {
		t.Fatalf("cursor after outer comma stayed on previous parameter: %d", got)
	}
	arrayFile, _ := source.New("array.can", "call choose([1, 2],  3)")
	arrayTokens := syntax.Lex(arrayFile).Tokens
	if got := activeArgument(arrayTokens, len("call choose"), len(arrayFile.Text()), strings.Index(arrayFile.Text(), "2]")); got != 0 {
		t.Fatalf("nested array comma advanced outer parameter: %d", got)
	}
}

func TestEditorMultilineStringFoldingAndLexicalColorOwnership(t *testing.T) {
	const text = `package scratch
    provides []
    uses []
error missing{str key}
str poem = r"""
    one
    two
    """
`
	uri := "untitled:can-fold-string"
	s := newLSPServer()
	s.open(uri, text, 1)
	folds := s.foldingRanges(uri).([]any)
	found := false
	for _, raw := range folds {
		fold := raw.(map[string]any)
		if fold["startLine"] == 4 && fold["endLine"] == 7 {
			found = true
		}
	}
	if !found {
		t.Fatalf("raw multiline string has no fold: %v", folds)
	}
	data := s.semanticTokens(uri).(map[string]any)["data"].([]int)
	line, character := 0, 0
	nameColored := false
	for i := 0; i < len(data); i += 5 {
		line += data[i]
		if data[i] != 0 {
			character = data[i+1]
		} else {
			character += data[i+1]
		}
		if line == 3 && character == 0 {
			t.Fatalf("generic semantic token masks error keyword scope: %v", data[i:i+5])
		}
		if line == 3 && character == 6 && data[i+3] == 1 {
			nameColored = true
		}
	}
	if !nameColored {
		t.Fatalf("resolved error name lacks semantic role: %v", data)
	}
}

func TestEditorParameterHintsRequireCheckedCallAndSetting(t *testing.T) {
	const text = `package scratch
    provides []
    uses []
fn int add
    emits {}
    given
        int left
        int right
    asserts
        sample: 1, 2 => ok 3
    ok left + right
fn int run
    emits {}
    asserts
        sample: => ok 3
    ok call add(1, 2)
`
	uri := "untitled:can-inlay-scratch"
	s := newLSPServer()
	s.open(uri, text, 1)
	rng := featureRange{Start: featurePosition{Line: 0, Character: 0}, End: featurePosition{Line: 16, Character: 0}}
	hints, ok := s.inlayHints(uri, rng).([]any)
	if !ok || len(hints) < 2 {
		t.Fatalf("checked parameter hints missing: %v", hints)
	}
	s.inlayEnabled = false
	if muted := s.inlayHints(uri, rng).([]any); len(muted) != 0 {
		t.Fatalf("disabled hints remained: %v", muted)
	}
}
