package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/driver"
	"github.com/veighnsche/can-lang/compiler/internal/project"
	compileresolve "github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

var semanticLegend = []string{"namespace", "type", "class", "enum", "parameter", "variable", "property", "function", "method", "keyword", "string", "number", "comment", "operator"}
var semanticModifiers = []string{"declaration", "readonly", "defaultLibrary"}

func featureCapabilities() map[string]any {
	return map[string]any{
		"completionProvider": map[string]any{"triggerCharacters": []string{".", ":"}},
		"referencesProvider": true, "renameProvider": map[string]any{"prepareProvider": true},
		"signatureHelpProvider": map[string]any{"triggerCharacters": []string{"(", ","}, "retriggerCharacters": []string{","}},
		"codeActionProvider":    true, "documentSymbolProvider": true, "workspaceSymbolProvider": true,
		"semanticTokensProvider": map[string]any{"legend": map[string]any{"tokenTypes": semanticLegend, "tokenModifiers": semanticModifiers}, "full": true},
		"foldingRangeProvider":   true, "inlayHintProvider": true,
		"documentRangeFormattingProvider":  true,
		"documentOnTypeFormattingProvider": map[string]any{"firstTriggerCharacter": "\n"},
	}
}

type featurePosition struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}
type featureRange struct {
	Start featurePosition `json:"start"`
	End   featurePosition `json:"end"`
}
type featureParams struct {
	TextDocument docID           `json:"textDocument"`
	Position     featurePosition `json:"position"`
	Range        featureRange    `json:"range"`
	Context      struct {
		IncludeDeclaration bool                `json:"includeDeclaration"`
		Diagnostics        []featureDiagnostic `json:"diagnostics"`
	} `json:"context"`
	NewName string `json:"newName"`
	Query   string `json:"query"`
	Ch      string `json:"ch"`
}

type featureDiagnostic struct {
	Range   featureRange `json:"range"`
	Code    any          `json:"code"`
	Message string       `json:"message"`
}

func (s *lspServer) handleFeature(msg rpcMsg) (any, bool, error) {
	switch msg.Method {
	case "textDocument/prepareRename", "textDocument/signatureHelp", "textDocument/codeAction",
		"textDocument/documentSymbol", "workspace/symbol", "textDocument/semanticTokens/full",
		"textDocument/foldingRange", "textDocument/inlayHint", "textDocument/rangeFormatting",
		"textDocument/onTypeFormatting":
	default:
		return nil, false, nil
	}
	var p featureParams
	if err := json.Unmarshal(msg.Params, &p); err != nil {
		return nil, true, err
	}
	fields := []string{}
	if msg.Method == "workspace/symbol" {
		fields = []string{"query"}
	} else {
		fields = []string{"textDocument.uri"}
	}
	switch msg.Method {
	case "textDocument/prepareRename", "textDocument/signatureHelp", "textDocument/onTypeFormatting":
		fields = append(fields, "position.line", "position.character")
	case "textDocument/codeAction", "textDocument/inlayHint", "textDocument/rangeFormatting":
		fields = append(fields, "range.start.line", "range.start.character", "range.end.line", "range.end.character")
	}
	if msg.Method == "textDocument/onTypeFormatting" {
		fields = append(fields, "ch")
	}
	if err := requireLSPFields(msg.Params, fields...); err != nil {
		return nil, true, err
	}
	if msg.Method != "workspace/symbol" {
		if p.TextDocument.URI == "" || s.docs[p.TextDocument.URI] == nil {
			return nil, true, &lspResponseError{code: -32602, message: "unknown text document"}
		}
		switch msg.Method {
		case "textDocument/codeAction", "textDocument/rangeFormatting", "textDocument/onTypeFormatting", "textDocument/prepareRename":
			if err := s.editablePath(s.docs[p.TextDocument.URI].path); err != nil {
				return nil, true, &lspResponseError{code: -32803, message: err.Error()}
			}
		}
		file, err := source.New(p.TextDocument.URI, s.docs[p.TextDocument.URI].text)
		if err != nil {
			return nil, true, &lspResponseError{code: -32602, message: "invalid document text"}
		}
		validate := func(pos featurePosition) error {
			if pos.Line < 0 || pos.Character < 0 {
				return &lspResponseError{code: -32602, message: "invalid position"}
			}
			if _, err := file.Offset(source.UTF16Position{Line: pos.Line, Character: pos.Character}); err != nil {
				return &lspResponseError{code: -32602, message: "position outside document"}
			}
			return nil
		}
		switch msg.Method {
		case "textDocument/prepareRename", "textDocument/signatureHelp", "textDocument/onTypeFormatting":
			if err := validate(p.Position); err != nil {
				return nil, true, err
			}
		case "textDocument/codeAction", "textDocument/inlayHint", "textDocument/rangeFormatting":
			if err := validate(p.Range.Start); err != nil {
				return nil, true, err
			}
			if err := validate(p.Range.End); err != nil {
				return nil, true, err
			}
			start, _ := file.Offset(source.UTF16Position{Line: p.Range.Start.Line, Character: p.Range.Start.Character})
			end, _ := file.Offset(source.UTF16Position{Line: p.Range.End.Line, Character: p.Range.End.Character})
			if end < start {
				return nil, true, &lspResponseError{code: -32602, message: "range end precedes start"}
			}
		}
	}
	switch msg.Method {
	case "textDocument/prepareRename":
		return s.prepareRename(p.TextDocument.URI, p.Position), true, nil
	case "textDocument/signatureHelp":
		return s.signatureHelp(p.TextDocument.URI, p.Position), true, nil
	case "textDocument/codeAction":
		return s.codeActionsForContext(p.TextDocument.URI, p.Range, p.Context.Diagnostics), true, nil
	case "textDocument/documentSymbol":
		return s.documentSymbols(p.TextDocument.URI), true, nil
	case "workspace/symbol":
		return s.workspaceSymbols(p.Query), true, nil
	case "textDocument/semanticTokens/full":
		return s.semanticTokens(p.TextDocument.URI), true, nil
	case "textDocument/foldingRange":
		return s.foldingRanges(p.TextDocument.URI), true, nil
	case "textDocument/inlayHint":
		return s.inlayHints(p.TextDocument.URI, p.Range), true, nil
	case "textDocument/rangeFormatting":
		return s.rangeFormatting(p.TextDocument.URI, p.Range), true, nil
	case "textDocument/onTypeFormatting":
		return s.onTypeFormatting(p.TextDocument.URI, p.Position, p.Ch), true, nil
	}
	return nil, false, nil
}

func sourceFor(snapshot *driver.Snapshot, path string) (*syntax.File, *source.File) {
	if snapshot == nil || snapshot.Graph == nil {
		return nil, nil
	}
	for _, p := range snapshot.Graph.Projects {
		for _, src := range p.Sources {
			if src.Path == path && src.Syntax != nil {
				file, err := source.New(path, string(src.Bytes))
				if err == nil {
					return src.Syntax, file
				}
			}
		}
	}
	return nil, nil
}

func bytePosition(file *source.File, p featurePosition) (int, bool) {
	if file == nil {
		return 0, false
	}
	offset, err := file.Offset(source.UTF16Position{Line: p.Line, Character: p.Character})
	return offset, err == nil
}

func wireRange(file *source.File, span source.Span) (map[string]any, bool) {
	if file == nil || file.Validate(span) != nil {
		return nil, false
	}
	a, ae := file.UTF16Position(span.Start)
	b, be := file.UTF16Position(span.End)
	if ae != nil || be != nil {
		return nil, false
	}
	return map[string]any{"start": map[string]int{"line": a.Line, "character": a.Character},
		"end": map[string]int{"line": b.Line, "character": b.Character}}, true
}

func (s *lspServer) featureSnapshot(uri string) (*lspDoc, *driver.Snapshot, *syntax.File, *source.File) {
	doc := s.docs[uri]
	if doc == nil || doc.path == "" {
		return nil, nil, nil, nil
	}
	snapshot, err := s.snapshot(uri)
	if err != nil || snapshot == nil {
		return doc, nil, nil, nil
	}
	ast, file := sourceFor(snapshot, doc.path)
	if file == nil {
		file, _ = source.New(doc.path, doc.text)
	}
	return doc, snapshot, ast, file
}

func (s *lspServer) prepareRename(uri string, p featurePosition) any {
	doc, snapshot, _, file := s.featureSnapshot(uri)
	if doc == nil || snapshot == nil || file == nil {
		return nil
	}
	offset, ok := bytePosition(file, p)
	if !ok {
		return nil
	}
	data := s.featureData(snapshot)
	if data == nil {
		return nil
	}
	occ, ok := data.occurrences.At(doc.path, offset)
	if !ok {
		return nil
	}
	span, ok := renameTokenRange(doc.text, occ.NameSpan)
	if !ok {
		return nil
	}
	name := doc.text[span.Start:span.End]
	if !validRenameName(name) {
		return nil
	}
	rng, ok := wireRange(file, span)
	if !ok {
		return nil
	}
	return map[string]any{"range": rng, "placeholder": name}
}

func (s *lspServer) localOccurrence(snapshot *driver.Snapshot, path string, line, character int) (driver.Occurrence, *source.File, bool) {
	_, file := sourceFor(snapshot, path)
	offset, ok := bytePosition(file, featurePosition{Line: line, Character: character})
	if !ok {
		return driver.Occurrence{}, nil, false
	}
	occ, ok := s.featureData(snapshot).occurrences.At(path, offset)
	if !ok || (occ.Kind != driver.OccurrenceLocal && occ.Kind != driver.OccurrenceInput && occ.Kind != driver.OccurrencePattern && occ.Kind != driver.OccurrenceCapture && occ.Kind != driver.OccurrencePin && occ.Kind != driver.OccurrenceField && occ.Kind != driver.OccurrenceFunction) {
		return driver.Occurrence{}, nil, false
	}
	return occ, file, true
}

func (s *lspServer) localDefinition(snapshot *driver.Snapshot, path string, line, character int) any {
	occ, _, ok := s.localOccurrence(snapshot, path, line, character)
	if !ok {
		return nil
	}
	for _, binding := range s.featureData(snapshot).occurrences.Binding(occ.ID) {
		if !binding.Declaration {
			continue
		}
		_, file := sourceFor(snapshot, binding.File)
		rng, ok := wireRange(file, binding.NameSpan)
		if !ok {
			return nil
		}
		return map[string]any{"uri": s.uri(binding.File), "range": rng}
	}
	return nil
}

func (s *lspServer) localHover(snapshot *driver.Snapshot, path string, line, character int) any {
	occ, file, ok := s.localOccurrence(snapshot, path, line, character)
	if !ok {
		return nil
	}
	rng, ok := wireRange(file, occ.NameSpan)
	if !ok {
		return nil
	}
	if occ.Type == "" {
		if occ.Kind != driver.OccurrenceField {
			return nil
		}
		for _, binding := range s.featureData(snapshot).occurrences.Binding(occ.ID) {
			if !binding.Declaration {
				continue
			}
			ast, _ := sourceFor(snapshot, binding.File)
			if ast == nil {
				continue
			}
			for _, decl := range ast.Declarations {
				record, ok := decl.(*syntax.RecordDecl)
				if !ok || !checkedNominal(snapshot, record) {
					continue
				}
				for _, field := range record.Fields {
					if field.Name.Span == binding.NameSpan {
						name, err := file.Slice(occ.NameSpan)
						if err == nil {
							return map[string]any{"contents": map[string]any{"kind": "markdown", "value": "`" + syntax.FormatType(field.Type) + " " + name + "`"}, "range": rng}
						}
					}
				}
			}
		}
		return nil
	}
	name, err := file.Slice(occ.NameSpan)
	if err != nil {
		return nil
	}
	return map[string]any{"contents": map[string]any{"kind": "markdown", "value": "`" + occ.Type + " " + name + "`"}, "range": rng}
}

func (s *lspServer) codeActions(uri string, selected featureRange) any {
	return s.codeActionsForContext(uri, selected, nil)
}

func (s *lspServer) codeActionsForContext(uri string, selected featureRange, context []featureDiagnostic) any {
	doc, snapshot, _, file := s.featureSnapshot(uri)
	if doc == nil || snapshot == nil || file == nil {
		return []any{}
	}
	start, ok := bytePosition(file, selected.Start)
	if !ok {
		return []any{}
	}
	end, ok := bytePosition(file, selected.End)
	if !ok {
		return []any{}
	}
	out := []any{}
	for _, diagnostic := range snapshot.Diagnostics {
		if diagnostic.File != doc.path {
			continue
		}
		current := featureDiagnostic{Range: featureRange{Start: featurePosition{Line: diagnostic.Line, Character: diagnostic.Start}, End: featurePosition{Line: diagnostic.EndLine, Character: diagnostic.End}}, Code: diagnostic.Code, Message: diagnostic.Message}
		diagnosticStart, startOK := bytePosition(file, current.Range.Start)
		diagnosticEnd, endOK := bytePosition(file, current.Range.End)
		if !startOK || !endOK || diagnosticEnd < start || diagnosticStart > end {
			continue
		}
		if len(context) > 0 {
			matched := false
			for _, given := range context {
				code, ok := given.Code.(string)
				if ok && given.Range == current.Range && code == diagnostic.Code && given.Message == current.Message {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		for _, fix := range diagnostic.Fixes {
			fix.Version = snapshot.Versions[fix.File]
			if strings.HasPrefix(uri, "untitled:") {
				fix.Version = doc.version
			}
			if driver.ValidateFix(discoverRoot(doc.path), snapshot, s.overlay, fix) != nil {
				continue
			}
			rng, ok := wireRange(file, source.Span{Start: fix.Start, End: fix.End})
			if !ok {
				continue
			}
			out = append(out, map[string]any{"title": fix.Title, "kind": "quickfix", "diagnostics": []any{map[string]any{"range": current.Range, "code": current.Code, "message": current.Message}}, "edit": map[string]any{
				"documentChanges": []any{map[string]any{"textDocument": map[string]any{"uri": uri, "version": fix.Version},
					"edits": []any{map[string]any{"range": rng, "newText": fix.NewText}}}}}})
		}
	}
	return out
}

func declarationSymbol(decl syntax.Declaration) (string, int, source.Span, []any) {
	fields := []any{}
	addFields := func(list []syntax.Field) {
		for _, field := range list {
			fields = append(fields, struct {
				Name     string
				Kind     int
				Span     source.Span
				Children []any
			}{field.Name.Text, 8, field.Name.Span, nil})
		}
	}
	switch n := decl.(type) {
	case *syntax.FunctionDecl:
		return n.Name.Text, 12, n.Name.Span, nil
	case *syntax.RecordDecl:
		addFields(n.Fields)
		return n.Name.Text, 23, n.Name.Span, fields
	case *syntax.ErrorDecl:
		addFields(n.Fields)
		return n.Name.Text, 23, n.Name.Span, fields
	case *syntax.VariantDecl:
		return n.Name.Text, 10, n.Name.Span, nil
	case *syntax.ValueDecl:
		return n.Binding.Name.Text, 13, n.Binding.Name.Span, nil
	case *syntax.QuestionDecl:
		if n.RecordName != nil {
			return n.RecordName.Text, 23, n.RecordName.Span, nil
		}
		return n.Name.Text, 12, n.Name.Span, nil
	case *syntax.ConnectionDecl:
		return n.Name.Text, 13, n.Name.Span, nil
	case *syntax.FetchDecl:
		return n.Name.Text, 12, n.Name.Span, nil
	case *syntax.LLMDecl:
		return n.Name.Text, 12, n.Name.Span, nil
	case *syntax.JudgeDecl:
		return n.Name.Text, 12, n.Name.Span, nil
	case *syntax.WrapDecl:
		return n.Name.Text, 12, n.Name.Span, nil
	case *syntax.ActionDecl:
		return n.Name.Text, 12, n.Name.Span, nil
	case *syntax.FixtureDecl:
		return n.Name.Text, 12, n.Name.Span, nil
	case *syntax.ScenarioDecl:
		return n.Name.Text, 13, n.Name.Span, nil
	case *syntax.ChoiceArmDecl:
		return n.Name.Text, 12, n.Name.Span, nil
	}
	return "", 0, source.Span{}, nil
}

func (s *lspServer) documentSymbols(uri string) any {
	_, _, ast, file := s.featureSnapshot(uri)
	if ast == nil || file == nil {
		return []any{}
	}
	out := []any{}
	for _, decl := range ast.Declarations {
		name, kind, sel, children := declarationSymbol(decl)
		if name == "" {
			continue
		}
		whole, ok := wireRange(file, decl.DeclSpan())
		if !ok {
			continue
		}
		selection, ok := wireRange(file, sel)
		if !ok {
			continue
		}
		childOut := []any{}
		for _, child := range children {
			c := child.(struct {
				Name     string
				Kind     int
				Span     source.Span
				Children []any
			})
			r, ok := wireRange(file, c.Span)
			if ok {
				childOut = append(childOut, map[string]any{"name": c.Name, "kind": c.Kind, "range": r, "selectionRange": r})
			}
		}
		out = append(out, map[string]any{"name": name, "kind": kind, "range": whole, "selectionRange": selection, "children": childOut})
	}
	return out
}

func (s *lspServer) workspaceSymbols(query string) any {
	out := []any{}
	seen := map[string]bool{}
	roots := map[string]bool{}
	for _, root := range s.workspaces {
		roots[root] = true
	}
	for uri, doc := range s.docs {
		if doc.path != "" && !strings.HasPrefix(uri, "untitled:") {
			roots[s.root(uri)] = true
		}
	}
	for root := range roots {
		snap, err := s.snapshotRoot(root)
		if err != nil || snap == nil || snap.Graph == nil {
			continue
		}
		for _, p := range snap.Graph.Projects {
			for _, src := range p.Sources {
				if seen[src.Path] || src.Syntax == nil {
					continue
				}
				seen[src.Path] = true
				f, err := source.New(src.Path, string(src.Bytes))
				if err != nil {
					continue
				}
				for _, decl := range src.Syntax.Declarations {
					name, kind, sel, _ := declarationSymbol(decl)
					if name == "" || !strings.Contains(strings.ToLower(name), strings.ToLower(query)) {
						continue
					}
					rng, ok := wireRange(f, sel)
					if ok {
						out = append(out, map[string]any{"name": name, "kind": kind, "location": map[string]any{"uri": s.uri(src.Path), "range": rng}})
					}
				}
			}
		}
	}
	return out
}

func (s *lspServer) semanticTokens(uri string) any {
	doc, snap, ast, file := s.featureSnapshot(uri)
	if doc == nil || file == nil {
		return map[string]any{"data": []int{}}
	}
	lexed := syntax.Lex(file)
	indexed := map[int]driver.Occurrence{}
	if snap != nil && ast != nil {
		for _, occ := range s.featureData(snap).occurrences.Occurrences() {
			if occ.File == doc.path {
				indexed[occ.NameSpan.Start] = occ
			}
		}
	}
	type item struct{ line, start, length, kind, mods int }
	items := []item{}
	for _, token := range lexed.Tokens {
		if token.Span.End <= token.Span.Start {
			continue
		}
		kind := -1
		mods := 0
		// Lexical colors belong to TextMate. A generic semantic keyword token
		// would mask declaration-specific scopes such as error declarations.
		if occ, ok := indexed[token.Span.Start]; ok && occ.NameSpan.End == token.Span.End {
			switch occ.Kind {
			case driver.OccurrenceInput, driver.OccurrencePin:
				kind = 4
			case driver.OccurrenceLocal, driver.OccurrencePattern, driver.OccurrenceCapture, driver.OccurrenceValue:
				kind = 5
			case driver.OccurrenceField:
				kind = 6
			case driver.OccurrenceRecord, driver.OccurrenceError, driver.OccurrenceTypeParam:
				kind = 1
			case driver.OccurrenceVariant:
				kind = 3
			case driver.OccurrenceImport:
				kind = 0
			case driver.OccurrenceFunction, driver.OccurrenceNative, driver.OccurrenceGenerated:
				kind = 7
			default:
				kind = -1
			}
			if occ.Declaration {
				mods = 1
			}
		}
		if kind < 0 {
			continue
		}
		a, ae := file.UTF16Position(token.Span.Start)
		b, be := file.UTF16Position(token.Span.End)
		if ae != nil || be != nil || a.Line != b.Line {
			continue
		}
		items = append(items, item{a.Line, a.Character, b.Character - a.Character, kind, mods})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].line != items[j].line {
			return items[i].line < items[j].line
		}
		return items[i].start < items[j].start
	})
	data := []int{}
	priorLine, priorStart := 0, 0
	for _, it := range items {
		delta := it.start
		if it.line == priorLine {
			delta -= priorStart
		}
		data = append(data, it.line-priorLine, delta, it.length, it.kind, it.mods)
		priorLine, priorStart = it.line, it.start
	}
	return map[string]any{"data": data}
}

func (s *lspServer) foldingRanges(uri string) any {
	doc, snapshot, ast, file := s.featureSnapshot(uri)
	if ast == nil || file == nil {
		return []any{}
	}
	out := []any{}
	seen := map[string]bool{}
	add := func(span source.Span, kind string) {
		a, ae := file.UTF16Position(span.Start)
		b, be := file.UTF16Position(span.End)
		if ae == nil && be == nil && b.Line > a.Line {
			key := fmt.Sprintf("%d:%d:%s", a.Line, b.Line, kind)
			if seen[key] {
				return
			}
			seen[key] = true
			out = append(out, map[string]any{"startLine": a.Line, "endLine": b.Line, "kind": kind})
		}
	}
	for _, decl := range ast.Declarations {
		add(decl.DeclSpan(), "region")
	}
	for _, comment := range ast.Comments {
		add(comment.Span, "comment")
	}
	for _, token := range syntax.Lex(file).Tokens {
		if token.Kind == syntax.String && token.Multiline {
			add(token.Span, "region")
		}
	}
	if snapshot != nil && doc != nil {
		if walker := s.featureData(snapshot).walkers[doc.path]; walker != nil {
			for _, frame := range walker.frames {
				add(frame.span, "region")
			}
		}
	}
	return out
}

func (s *lspServer) inlayHints(uri string, selected featureRange) any {
	if !s.inlayEnabled {
		return []any{}
	}
	doc, snap, ast, file := s.featureSnapshot(uri)
	if doc == nil || snap == nil || snap.World == nil || snap.Program == nil || ast == nil || file == nil {
		return []any{}
	}
	start, ok := bytePosition(file, selected.Start)
	if !ok {
		return []any{}
	}
	end, ok := bytePosition(file, selected.End)
	if !ok {
		return []any{}
	}
	walker := s.featureData(snap).walkers[doc.path]
	if walker == nil {
		return []any{}
	}
	index := s.featureData(snap).occurrences
	out := []any{}
	for _, call := range walker.calls {
		name, ok := call.Callee.(*syntax.NameExpr)
		if !ok {
			continue
		}
		occ, ok := index.At(doc.path, name.Name.Span.Start)
		if !ok {
			continue
		}
		decl := functionForOccurrence(snap, index, occ)
		if decl == nil || !checkedFunction(snap, decl) {
			continue
		}
		positional := make([]syntax.Input, 0, len(decl.Inputs))
		for _, input := range decl.Inputs {
			if !input.Near {
				positional = append(positional, input)
			}
		}
		for i, arg := range call.Arguments {
			if i >= len(positional) || arg.Span.Start < start || arg.Span.Start > end {
				continue
			}
			input := positional[i]
			if nameArg, ok := arg.Value.(*syntax.NameExpr); ok && nameArg.Name.Name == input.Name.Text {
				continue
			}
			pos, err := file.UTF16Position(arg.Span.Start)
			if err != nil {
				continue
			}
			out = append(out, map[string]any{"position": map[string]int{"line": pos.Line, "character": pos.Character}, "label": input.Name.Text + ":", "kind": 2, "paddingRight": true})
		}
	}
	return out
}

func functionForOccurrence(snapshot *driver.Snapshot, index *driver.OccurrenceIndex, occ driver.Occurrence) *syntax.FunctionDecl {
	for _, binding := range index.Binding(occ.ID) {
		if !binding.Declaration {
			continue
		}
		ast, _ := sourceFor(snapshot, binding.File)
		if ast == nil {
			continue
		}
		for _, decl := range ast.Declarations {
			if fn, ok := decl.(*syntax.FunctionDecl); ok && fn.Name.Span == binding.NameSpan {
				return fn
			}
		}
	}
	return nil
}

func checkedFunction(snapshot *driver.Snapshot, decl *syntax.FunctionDecl) bool {
	if snapshot == nil || snapshot.Program == nil || decl == nil {
		return false
	}
	for _, fn := range snapshot.Program.Functions {
		if fn != nil && fn.Symbol != nil && fn.Symbol.Declaration == decl && fn.Region != nil {
			return true
		}
	}
	return false
}

func checkedFunctionSignature(snapshot *driver.Snapshot, decl *syntax.FunctionDecl) bool {
	if snapshot == nil || snapshot.Program == nil || decl == nil {
		return false
	}
	for _, fn := range snapshot.Program.Functions {
		if fn != nil && fn.Instance == "" && fn.Symbol != nil && fn.Symbol.Declaration == decl && fn.Signature != nil && fn.Signature.Kind() == types.Callable && types.Equal(fn.Signature, fn.Signature) {
			return true
		}
	}
	return false
}

func (s *lspServer) rangeFormatting(uri string, selected featureRange) any {
	doc, snap, _, file := s.featureSnapshot(uri)
	if doc == nil || snap == nil || file == nil || snapshotHasErrors(snap) {
		return nil
	}
	start, ok := bytePosition(file, selected.Start)
	if !ok {
		return nil
	}
	end, ok := bytePosition(file, selected.End)
	if !ok || end < start {
		return nil
	}
	formatted, err := formatSource(doc.path, doc.text)
	if err != nil {
		return nil
	}
	if formatted == doc.text {
		return []any{}
	}
	newText := ""
	if selected.Start.Character == 0 && selected.End.Character == 0 && selected.Start.Line < selected.End.Line {
		originalLines := strings.SplitAfter(doc.text, "\n")
		formattedLines := strings.SplitAfter(formatted, "\n")
		// Equal physical-line counts give a linear, unambiguous line mapping.
		// Other noncanonical lines can remain untouched by this isolated edit.
		if len(originalLines) != len(formattedLines) || selected.End.Line > len(originalLines) {
			return nil
		}
		newText = strings.Join(formattedLines[selected.Start.Line:selected.End.Line], "")
	} else {
		// A partial-line edit needs an exact unchanged prefix and suffix.
		if start > len(formatted) || len(doc.text)-end > len(formatted)-start ||
			doc.text[:start] != formatted[:start] || doc.text[end:] != formatted[len(formatted)-(len(doc.text)-end):] {
			return nil
		}
		newText = formatted[start : len(formatted)-(len(doc.text)-end)]
	}
	if newText == doc.text[start:end] {
		return []any{}
	}
	candidate := project.NewOverlay()
	for path, entry := range s.overlay.Snapshot() {
		if candidate.Set(path, entry.Version, entry.Text) != nil {
			return nil
		}
	}
	if candidate.Set(doc.path, doc.version, doc.text[:start]+newText+doc.text[end:]) != nil {
		return nil
	}
	var proposed *driver.Snapshot
	if strings.HasPrefix(uri, "untitled:") {
		proposed, err = driver.CheckScratchSnapshot(s.ctx, doc.path, doc.text[:start]+newText+doc.text[end:])
	} else {
		proposed, err = driver.CheckSnapshot(discoverRoot(doc.path), doc.path, candidate)
	}
	if err != nil || snapshotHasErrors(proposed) {
		return nil
	}
	rng, ok := wireRange(file, source.Span{Start: start, End: end})
	if !ok {
		return nil
	}
	return []any{map[string]any{"range": rng, "newText": newText}}
}

func (s *lspServer) onTypeFormatting(uri string, p featurePosition, ch string) any {
	if ch != "\n" || p.Line == 0 {
		return nil
	}
	return s.rangeFormatting(uri, featureRange{Start: featurePosition{Line: p.Line - 1, Character: 0}, End: featurePosition{Line: p.Line, Character: 0}})
}

// activeArgument counts only separators owned by the invocation's outer
// delimiters. Lexer tokens keep strings and nested constructors indivisible.
func activeArgument(tokens []syntax.Token, after, end, cursor int) int {
	open := -1
	for i, token := range tokens {
		if token.Span.Start < after || token.Span.Start >= end {
			continue
		}
		if token.Kind == "(" || token.Kind == "{" {
			open = i
			break
		}
	}
	if open < 0 || cursor < tokens[open].Span.End {
		return 0
	}
	stack := []syntax.Kind{tokens[open].Kind}
	active := 0
	for _, token := range tokens[open+1:] {
		if token.Span.End > cursor || token.Span.Start >= end || len(stack) == 0 {
			break
		}
		switch token.Kind {
		case "(", "[", "{":
			stack = append(stack, token.Kind)
		case ")", "]", "}":
			stack = stack[:len(stack)-1]
		case ",":
			if len(stack) == 1 {
				active++
			}
		}
	}
	return active
}

func (s *lspServer) signatureHelp(uri string, p featurePosition) any {
	doc, snap, ast, file := s.featureSnapshot(uri)
	if doc == nil || snap == nil || ast == nil || file == nil {
		return nil
	}
	offset, ok := bytePosition(file, p)
	if !ok {
		return nil
	}
	var callee syntax.Expr
	var methodName *syntax.Token
	var methodReceiver syntax.Expr
	var constructor *syntax.ConstructorExpr
	var reference *syntax.ReferenceExpr
	active := 0
	tokens := syntax.Lex(file).Tokens
	best := int(^uint(0) >> 1)
	walker := s.featureData(snap).walkers[doc.path]
	if walker == nil {
		return nil
	}
	for _, call := range walker.calls {
		if call.Span.Start > offset || offset > call.Span.End {
			continue
		}
		width := call.Span.End - call.Span.Start
		if width >= best {
			continue
		}
		best = width
		callee = call.Callee
		reference = nil
		active = activeArgument(tokens, call.Callee.ExprSpan().End, call.Span.End, offset)
	}
	for _, method := range walker.methods {
		if method.Span.Start > offset || offset > method.Span.End {
			continue
		}
		width := method.Span.End - method.Span.Start
		if width >= best {
			continue
		}
		best = width
		copy := method.Name
		methodName = &copy
		methodReceiver = nil
		constructor = nil
		callee = nil
		reference = nil
		active = activeArgument(tokens, method.Name.Span.End, method.Span.End, offset)
	}
	for _, ctor := range walker.constructors {
		if ctor.ExprSpan().Start > offset || offset > ctor.ExprSpan().End || offset <= ctor.Name.Span.End {
			continue
		}
		width := ctor.ExprSpan().End - ctor.ExprSpan().Start
		if width >= best {
			continue
		}
		best = width
		constructor = ctor
		methodName = nil
		methodReceiver = nil
		callee = nil
		reference = nil
		active = activeArgument(tokens, ctor.Name.Span.End, ctor.ExprSpan().End, offset)
	}
	for _, partial := range ast.Incomplete {
		if (partial.Kind != "call" && partial.Kind != "constructor" && partial.Kind != "method" && partial.Kind != "pin") || partial.Callee == nil || offset < partial.Open.End || offset > partial.Span.End {
			continue
		}
		width := partial.Span.End - partial.Span.Start
		if width >= best {
			continue
		}
		best = width
		callee, methodName, methodReceiver, constructor = partial.Callee, nil, nil, nil
		reference = nil
		switch partial.Kind {
		case "pin":
			binding := syntax.WithBinding{Span: source.Span{Start: partial.Name.Span.Start, End: partial.Span.End}, Name: syntax.Token{Kind: syntax.Name, Text: partial.Name.Name, Span: partial.Name.Span}}
			reference = &syntax.ReferenceExpr{ExpressionLocation: syntax.ExpressionLocation{Span: partial.Span}, Callee: partial.Callee, Bindings: []syntax.WithBinding{binding}}
		case "constructor":
			constructor = &syntax.ConstructorExpr{ExpressionLocation: syntax.ExpressionLocation{Span: partial.Span}, Name: partial.Name}
			callee = nil
		case "method":
			name := syntax.Token{Kind: syntax.Name, Text: partial.Name.Name, Span: partial.Name.Span}
			methodName = &name
			methodReceiver = partial.Receiver
			callee = nil
		}
		active = activeArgument(tokens, partial.Open.Start, partial.Span.End, offset)
	}
	for _, candidate := range walker.references {
		span := candidate.ExprSpan()
		if span.Start > offset || offset > span.End || span.End-span.Start >= best {
			continue
		}
		best = span.End - span.Start
		reference, callee = candidate, candidate.Callee
		methodName, methodReceiver, constructor = nil, nil, nil
		active = 0
	}
	var name syntax.QualifiedName
	if methodName != nil {
		name = syntax.QualifiedName{Name: methodName.Text, Span: methodName.Span}
	} else if constructor != nil {
		name = constructor.Name
	} else if named, ok := callee.(*syntax.NameExpr); ok {
		name = named.Name
	} else {
		return nil
	}
	var declaration *syntax.FunctionDecl
	var record *syntax.RecordDecl
	var errorDecl *syntax.ErrorDecl
	var callback *syntax.CallableType
	// A local binder wins over the package symbol even when its invalid call
	// does not enter the occurrence index. Only a checked callable local can
	// provide a signature; an int (or an unproved local type) must decline.
	var localBinder *compBinder
	if methodName == nil && constructor == nil && name.Package == "" {
		localBinder = walker.lookup(name.Name, name.Span.Start)
	}
	localBound := localBinder != nil
	if occ, ok := s.featureData(snap).occurrences.At(doc.path, name.Span.Start); ok {
		declaration = functionForOccurrence(snap, s.featureData(snap).occurrences, occ)
		if constructor != nil {
			record = recordForOccurrence(snap, s.featureData(snap).occurrences, occ)
			errorDecl = errorForOccurrence(snap, s.featureData(snap).occurrences, occ)
		}
		if occ.Type != "" {
			for _, frame := range walker.frames {
				for _, binder := range frame.binders {
					if localIDFor(doc.path, binder.span) == occ.ID {
						callback, _ = binder.typ.(*syntax.CallableType)
					}
				}
			}
		}
	}
	if localBound {
		declaration = nil
		if callback == nil {
			for _, binding := range s.featureData(snap).occurrences.Binding(localIDFor(doc.path, localBinder.span)) {
				if binding.Declaration && binding.Type != "" {
					callback, _ = localBinder.typ.(*syntax.CallableType)
					break
				}
			}
		}
		if callback == nil {
			return nil
		}
	}
	if !localBound && declaration == nil && record == nil && errorDecl == nil && snap.World != nil && methodName == nil {
		for _, project := range snap.Graph.Projects {
			for _, src := range project.Sources {
				if src.Path != doc.path {
					continue
				}
				resolved := snap.World.Files[src]
				if resolved == nil {
					continue
				}
				usage := compileresolve.CallUse
				if constructor != nil {
					usage = compileresolve.ConstructorUse
				}
				symbol, err := resolved.Lookup(nil, name, usage)
				if err == nil && symbol != nil {
					declaration, _ = symbol.Declaration.(*syntax.FunctionDecl)
					record, _ = symbol.Declaration.(*syntax.RecordDecl)
					errorDecl, _ = symbol.Declaration.(*syntax.ErrorDecl)
				}
			}
		}
	}
	if declaration == nil && methodName != nil && methodReceiver != nil && snap.World != nil {
		for _, project := range snap.Graph.Projects {
			for _, src := range project.Sources {
				if src.Path != doc.path {
					continue
				}
				resolved := snap.World.Files[src]
				if resolved == nil {
					continue
				}
				record, err := knownReceiverRecord(resolved, walker.shadowsLocal, walker.localRecord, methodReceiver)
				if err != nil {
					continue
				}
				method, err := resolved.Method(record, methodName.Text)
				if err == nil {
					declaration, _ = method.Declaration.(*syntax.FunctionDecl)
				}
			}
		}
	}
	params := []any{}
	labels := []string{}
	if declaration != nil && checkedFunctionSignature(snap, declaration) {
		for _, input := range declaration.Inputs {
			if input.Near && reference == nil {
				continue
			}
			if reference != nil && input.Near {
				for _, binding := range reference.Bindings {
					if binding.Name.Text == input.Name.Text && offset >= binding.Name.Span.Start && offset <= binding.Span.End {
						active = len(params)
					}
				}
			}
			label := syntax.FormatType(input.Type) + " " + input.Name.Text
			if input.Near {
				label = "near " + label
			}
			labels = append(labels, label)
			params = append(params, map[string]any{"label": label})
		}
	} else if constructor != nil && (record != nil || errorDecl != nil) {
		var nominal syntax.Declaration
		var fields []syntax.Field
		if record != nil {
			nominal, fields = record, record.Fields
		} else {
			nominal, fields = errorDecl, errorDecl.Fields
		}
		if !checkedNominal(snap, nominal) {
			return nil
		}
		for _, field := range fields {
			label := syntax.FormatType(field.Type) + " " + field.Name.Text
			labels = append(labels, label)
			params = append(params, map[string]any{"label": label})
		}
	} else if callback != nil {
		for _, input := range callback.Inputs {
			label := syntax.FormatType(input)
			labels = append(labels, label)
			params = append(params, map[string]any{"label": label})
		}
	} else {
		return nil
	}
	if active >= len(params) && len(params) > 0 {
		active = len(params) - 1
	}
	return map[string]any{"signatures": []any{map[string]any{"label": name.Name + "(" + strings.Join(labels, ", ") + ")", "parameters": params}}, "activeSignature": 0, "activeParameter": active}
}

func recordForOccurrence(snapshot *driver.Snapshot, index *driver.OccurrenceIndex, occ driver.Occurrence) *syntax.RecordDecl {
	for _, binding := range index.Binding(occ.ID) {
		if !binding.Declaration {
			continue
		}
		ast, _ := sourceFor(snapshot, binding.File)
		if ast == nil {
			continue
		}
		for _, decl := range ast.Declarations {
			if record, ok := decl.(*syntax.RecordDecl); ok && record.Name.Span == binding.NameSpan {
				return record
			}
		}
	}
	return nil
}

func errorForOccurrence(snapshot *driver.Snapshot, index *driver.OccurrenceIndex, occ driver.Occurrence) *syntax.ErrorDecl {
	for _, binding := range index.Binding(occ.ID) {
		if !binding.Declaration {
			continue
		}
		ast, _ := sourceFor(snapshot, binding.File)
		if ast == nil {
			continue
		}
		for _, decl := range ast.Declarations {
			if named, ok := decl.(*syntax.ErrorDecl); ok && named.Name.Span == binding.NameSpan {
				return named
			}
		}
	}
	return nil
}

func checkedNominal(snapshot *driver.Snapshot, nominal syntax.Declaration) bool {
	if snapshot == nil || snapshot.World == nil || snapshot.Analysis == nil || nominal == nil {
		return false
	}
	for _, p := range snapshot.World.Files {
		for _, symbol := range p.Package.Scope.Symbols {
			if symbol.Declaration == nominal {
				return snapshot.Analysis.Unit(driver.UnitID(symbol.ID)).Status == source.UnitValid
			}
		}
	}
	return false
}
