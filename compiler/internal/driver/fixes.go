package driver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/veighnsche/can-lang/compiler/internal/project"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
)

// Fix is one validated source repair proposal: a title plus a single
// replaced byte range of the diagnosed file. Version is the overlay
// document version the diagnosis belongs to, or zero for saved disk text.
type Fix struct {
	Title   string
	File    string
	Start   int
	End     int
	NewText string
	Version int64
}

// SuggestedFixes collects the compiler-proposed repairs attached to one
// file's snapshot diagnostics, stamped with the diagnosis versions. It
// proposes only; callers must pass every fix through ValidateFix and
// present only fixes that validate.
func SuggestedFixes(snapshot *Snapshot, file string) []Fix {
	if snapshot == nil {
		return nil
	}
	var out []Fix
	for _, diagnostic := range snapshot.Diagnostics {
		if diagnostic.File != file {
			continue
		}
		for _, fix := range diagnostic.Fixes {
			fix.Version = snapshot.Versions[fix.File]
			out = append(out, fix)
		}
	}
	return out
}

// ValidateFix proves one repair against an isolated snapshot: the edit must
// still belong to the diagnosed buffer, preserve declared contracts,
// assertions and completion ownership, resolve the target issue, and
// introduce no error beyond unrelated unchanged findings. Anything else
// is rejected with the reason. Validation never
// builds, publishes, runs, dials out or mutates editor state.
func ValidateFix(directory string, snapshot *Snapshot, overlay *project.Overlay, fix Fix) error {
	if snapshot == nil || snapshot.Graph == nil {
		return fmt.Errorf("fix validation requires a diagnosis snapshot")
	}
	if fix.File == "" || fix.Start < 0 || fix.End < fix.Start {
		return fmt.Errorf("fix %q names an invalid range", fix.Title)
	}
	// Compiler-generated fixes are insert-only; anything else deletes
	// source the preservation check cannot attribute to an added arm.
	if fix.Start != fix.End {
		return fmt.Errorf("fix %q is not an insertion", fix.Title)
	}
	before, ok := fileText(snapshot.Graph, fix.File)
	if !ok {
		return fmt.Errorf("fix %q targets %s outside the diagnosed project", fix.Title, fix.File)
	}
	if err := checkFresh(overlay, snapshot, fix, before); err != nil {
		return err
	}
	if err := checkSnapshotInputsFresh(snapshot, overlay); err != nil {
		return err
	}
	if fix.End > len(before) {
		return fmt.Errorf("fix %q range exceeds the diagnosed buffer", fix.Title)
	}
	after := before[:fix.Start] + fix.NewText + before[fix.End:]
	if err := checkPreserved(fix.File, before, after); err != nil {
		return fmt.Errorf("fix %q rejected: %v", fix.Title, err)
	}
	isolated := project.NewOverlay()
	if overlay != nil {
		for path, entry := range overlay.Snapshot() {
			if err := isolated.Set(path, entry.Version, entry.Text); err != nil {
				return err
			}
		}
	}
	if err := isolated.Set(fix.File, fix.Version, after); err != nil {
		return err
	}
	var rechecked *Snapshot
	var err error
	if scratchSnapshot(snapshot) {
		rechecked, err = CheckScratchSnapshot(context.Background(), fix.File, after)
	} else {
		rechecked, err = CheckSnapshot(directory, fix.File, isolated)
	}
	if err != nil {
		return err
	}
	targetIndex := -1
	for i := range snapshot.Diagnostics {
		for _, proposed := range snapshot.Diagnostics[i].Fixes {
			if proposed.File == fix.File && proposed.Start == fix.Start && proposed.End == fix.End && proposed.NewText == fix.NewText {
				targetIndex = i
				break
			}
		}
	}
	if targetIndex < 0 {
		return fmt.Errorf("fix %q is not proposed by this diagnosis", fix.Title)
	}
	remaining := map[string]int{}
	for i, diagnostic := range snapshot.Diagnostics {
		if diagnostic.Severity == "error" || diagnostic.Severity == "" {
			if i == targetIndex {
				continue
			}
			mapped, ok := mapFixDiagnostic(diagnostic, before, after, fix)
			if !ok {
				return fmt.Errorf("fix %q overlaps an unrelated diagnostic", fix.Title)
			}
			remaining[fixIssueKey(mapped)]++
		}
	}
	for _, diagnostic := range rechecked.Diagnostics {
		if diagnostic.Severity != "error" && diagnostic.Severity != "" {
			continue
		}
		key := fixIssueKey(diagnostic)
		if remaining[key] == 0 {
			return fmt.Errorf("fix %q introduces %s: %s", fix.Title, diagnostic.Code, diagnostic.Message)
		}
		remaining[key]--
	}
	return nil
}

func fixIssueKey(diagnostic Diagnostic) string {
	return fmt.Sprintf("%q|%q|%q|%d:%d-%d:%d|%q|%#v", diagnostic.Severity, diagnostic.Code, diagnostic.File, diagnostic.Line, diagnostic.Start, diagnostic.EndLine, diagnostic.End, diagnostic.Message, diagnostic.Related)
}

func mapFixDiagnostic(d Diagnostic, before, after string, fix Fix) (Diagnostic, bool) {
	d.Related = append([]RelatedDiagnostic(nil), d.Related...)
	oldFile, err := source.New(fix.File, before)
	if err != nil {
		return d, false
	}
	newFile, err := source.New(fix.File, after)
	if err != nil {
		return d, false
	}
	mapSpan := func(path string, line, start, endLine, end int) (int, int, int, int, bool) {
		if path != fix.File {
			return line, start, endLine, end, true
		}
		a, ae := oldFile.Offset(source.UTF16Position{Line: line, Character: start})
		if ae != nil {
			return 0, 0, 0, 0, false
		}
		b, be := oldFile.Offset(source.UTF16Position{Line: endLine, Character: end})
		if be != nil {
			return 0, 0, 0, 0, false
		}
		if a < fix.Start && fix.Start < b || a == b && a == fix.Start {
			return 0, 0, 0, 0, false
		}
		if a >= fix.Start {
			a += len(fix.NewText)
			b += len(fix.NewText)
		}
		first, fe := newFile.UTF16Position(a)
		last, le := newFile.UTF16Position(b)
		if fe != nil || le != nil {
			return 0, 0, 0, 0, false
		}
		return first.Line, first.Character, last.Line, last.Character, true
	}
	var ok bool
	d.Line, d.Start, d.EndLine, d.End, ok = mapSpan(d.File, d.Line, d.Start, d.EndLine, d.End)
	if !ok {
		return d, false
	}
	for i := range d.Related {
		related := &d.Related[i]
		related.Line, related.Start, related.EndLine, related.End, ok = mapSpan(related.File, related.Line, related.Start, related.EndLine, related.End)
		if !ok {
			return d, false
		}
	}
	return d, true
}

func checkSnapshotInputsFresh(snapshot *Snapshot, overlay *project.Overlay) error {
	if snapshot == nil {
		return fmt.Errorf("stale diagnosis snapshot")
	}
	if snapshot.Analysis != nil {
		for _, input := range snapshot.Analysis.Fingerprint.Docs() {
			if overlay != nil {
				if entry, ok := overlay.Get(input.Path); ok {
					if entry.Version != input.Version || entry.Text != input.Text {
						return fmt.Errorf("stale diagnosis: %s changed", input.Path)
					}
					continue
				}
			}
			if input.Version != 0 {
				return fmt.Errorf("stale diagnosis: missing overlay %s", input.Path)
			}
			if data, ok := snapshot.Graph.Inputs[input.Path]; ok && data == nil {
				if _, err := os.Stat(input.Path); os.IsNotExist(err) {
					continue
				}
			}
			current, err := os.ReadFile(input.Path)
			if err != nil || string(current) != input.Text {
				return fmt.Errorf("stale diagnosis: %s changed", input.Path)
			}
		}
	}
	for path, data := range snapshot.Graph.Inputs {
		if overlay != nil {
			if entry, ok := overlay.Get(path); ok {
				if entry.Text != string(data) {
					return fmt.Errorf("stale diagnosis: %s changed", path)
				}
				continue
			}
		}
		if data == nil {
			if _, err := os.Stat(path); err == nil {
				return fmt.Errorf("stale diagnosis: %s appeared", path)
			}
			continue
		}
		current, err := os.ReadFile(path)
		if err != nil || string(current) != string(data) {
			return fmt.Errorf("stale diagnosis: %s changed", path)
		}
	}
	return nil
}

// checkFresh rejects edits that no longer belong to the diagnosed buffer:
// the overlay version must match and the current bytes must equal the
// diagnosed bytes.
func checkFresh(overlay *project.Overlay, snapshot *Snapshot, fix Fix, before string) error {
	if scratchSnapshot(snapshot) {
		if overlay == nil {
			return fmt.Errorf("fix %q is stale: scratch buffer is closed", fix.Title)
		}
		entry, ok := overlay.Get(fix.File)
		if !ok || entry.Version != fix.Version || entry.Text != before {
			return fmt.Errorf("fix %q is stale: scratch buffer changed", fix.Title)
		}
		return nil
	}
	want := snapshot.Versions[fix.File]
	if overlay == nil {
		if want != 0 {
			return fmt.Errorf("fix %q is stale: diagnosis belongs to overlay version %d", fix.Title, want)
		}
		current, err := os.ReadFile(fix.File)
		if err != nil || string(current) != before {
			return fmt.Errorf("fix %q is stale: %s changed since diagnosis", fix.Title, filepath.Base(fix.File))
		}
		return nil
	}
	entry, ok := overlay.Get(fix.File)
	if !ok {
		if want != 0 {
			return fmt.Errorf("fix %q is stale: diagnosis belongs to overlay version %d", fix.Title, want)
		}
		current, err := os.ReadFile(fix.File)
		if err != nil || string(current) != before {
			return fmt.Errorf("fix %q is stale: %s changed since diagnosis", fix.Title, filepath.Base(fix.File))
		}
		return nil
	}
	if entry.Version != want {
		return fmt.Errorf("fix %q is stale: diagnosis belongs to overlay version %d, editor holds %d", fix.Title, want, entry.Version)
	}
	if entry.Text != before {
		return fmt.Errorf("fix %q is stale: buffer changed since diagnosis", fix.Title)
	}
	return nil
}

func scratchSnapshot(snapshot *Snapshot) bool {
	return snapshot != nil && snapshot.Graph != nil && snapshot.Graph.Root != nil && snapshot.Graph.Root.ID == "can.scratch"
}

// checkPreserved rejects repairs that change anything but match arms:
// declared contracts (signatures, bounds, state, values), assertion rows
// (including fixture when rows) and relay/inherit completion ownership.
func checkPreserved(file, before, after string) error {
	oldDecls, err := parseDeclarations(file, before)
	if err != nil {
		return err
	}
	newDecls, err := parseDeclarations(file, after)
	if err != nil {
		return fmt.Errorf("fix breaks parsing: %v", err)
	}
	if len(oldDecls) != len(newDecls) {
		return fmt.Errorf("fix changes the declaration set")
	}
	for i := range oldDecls {
		if err := compareDeclaration(before, after, oldDecls[i], newDecls[i]); err != nil {
			return err
		}
	}
	return nil
}

func parseDeclarations(file, text string) ([]syntax.Declaration, error) {
	parsed, err := source.New(file, text)
	if err != nil {
		return nil, err
	}
	result := syntax.Parse(parsed)
	if len(result.Diagnostics) != 0 {
		return nil, fmt.Errorf("%s", result.Diagnostics[0].Message)
	}
	if result.File == nil {
		return nil, fmt.Errorf("fix leaves %s unparsed", file)
	}
	return result.File.Declarations, nil
}

func compareDeclaration(before, after string, oldDecl, newDecl syntax.Declaration) error {
	oldFn, oldOK := oldDecl.(*syntax.FunctionDecl)
	newFn, newOK := newDecl.(*syntax.FunctionDecl)
	if oldOK != newOK {
		return fmt.Errorf("fix changes the declaration set")
	}
	if !oldOK {
		if spanText(before, oldDecl.DeclSpan()) != spanText(after, newDecl.DeclSpan()) {
			return fmt.Errorf("fix changes declared contracts")
		}
		return nil
	}
	if oldFn.Name.Text != newFn.Name.Text {
		return fmt.Errorf("fix changes declared contracts")
	}
	if spanText(before, oldFn.Result.TypeSpan()) != spanText(after, newFn.Result.TypeSpan()) {
		return fmt.Errorf("fix changes declared contracts")
	}
	if spanText(before, oldFn.Errors.Span) != spanText(after, newFn.Errors.Span) {
		return fmt.Errorf("fix changes declared contracts")
	}
	if len(oldFn.Inputs) != len(newFn.Inputs) {
		return fmt.Errorf("fix changes declared contracts")
	}
	for i := range oldFn.Inputs {
		if spanText(before, oldFn.Inputs[i].Span) != spanText(after, newFn.Inputs[i].Span) {
			return fmt.Errorf("fix changes declared contracts")
		}
	}
	oldRows := assertionTexts(before, oldFn)
	newRows := assertionTexts(after, newFn)
	if len(oldRows) != len(newRows) {
		return fmt.Errorf("fix touches assertions")
	}
	for i := range oldRows {
		if oldRows[i] != newRows[i] {
			return fmt.Errorf("fix touches assertions")
		}
	}
	oldRelay, oldInherit := ownershipBlock(&oldFn.Body)
	newRelay, newInherit := ownershipBlock(&newFn.Body)
	if oldRelay != newRelay || oldInherit != newInherit {
		return fmt.Errorf("fix changes completion ownership")
	}
	return nil
}

// assertionTexts collects every assertion row text in order: attached rows
// plus fixture when rows anywhere in the body.
func assertionTexts(text string, fn *syntax.FunctionDecl) []string {
	out := make([]string, 0, len(fn.Assertions))
	for i := range fn.Assertions {
		out = append(out, spanText(text, fn.Assertions[i].Span))
	}
	collectBlockRows(text, &fn.Body, &out)
	return out
}

func collectBodyRows(text string, body syntax.Body, out *[]string) {
	switch b := body.(type) {
	case *syntax.DoBody:
		collectBlockRows(text, &b.Block, out)
	case *syntax.MatchBody:
		collectMatchRows(text, &b.Match, out)
	}
}

func collectBlockRows(text string, block *syntax.Block, out *[]string) {
	for _, step := range block.Steps {
		switch s := step.(type) {
		case *syntax.BindingStep:
			collectExprRows(text, s.Value, out)
		case *syntax.CallStep:
			if s.Call != nil {
				collectExprRows(text, s.Call, out)
			}
		case *syntax.CoordinationStep:
			collectCoordinationRows(text, &s.Coordination, out)
		}
	}
	if block.Terminal != nil {
		collectBodyRows(text, block.Terminal, out)
	}
}

func collectMatchRows(text string, match *syntax.Match, out *[]string) {
	for i := range match.When {
		*out = append(*out, spanText(text, match.When[i].Span))
		if match.When[i].Expected != nil {
			collectBodyRows(text, match.When[i].Expected, out)
		}
	}
	for i := range match.Arms {
		if match.Arms[i].Body != nil {
			collectBodyRows(text, match.Arms[i].Body, out)
		}
	}
	for i := range match.Values {
		collectExprRows(text, match.Values[i], out)
	}
	if match.Call != nil {
		collectExprRows(text, match.Call, out)
	}
	for i := range match.Chain {
		if match.Chain[i].Call != nil {
			collectExprRows(text, match.Chain[i].Call, out)
		}
	}
}

func collectCoordinationRows(text string, coord *syntax.Coordination, out *[]string) {
	for i := range coord.Participants {
		participant := &coord.Participants[i]
		if participant.Call != nil {
			collectExprRows(text, participant.Call, out)
		}
		if participant.Spread != nil {
			collectExprRows(text, participant.Spread, out)
		}
		for a := range participant.Arms {
			if participant.Arms[a].Body != nil {
				collectBodyRows(text, participant.Arms[a].Body, out)
			}
		}
	}
	for i := range coord.Arms {
		if coord.Arms[i].Body != nil {
			collectBodyRows(text, coord.Arms[i].Body, out)
		}
	}
}

func collectExprRows(text string, expr syntax.Expr, out *[]string) {
	switch e := expr.(type) {
	case *syntax.GroupExpr:
		collectExprRows(text, e.Value, out)
	case *syntax.UnaryExpr:
		collectExprRows(text, e.Operand, out)
	case *syntax.BinaryExpr:
		collectExprRows(text, e.Left, out)
		collectExprRows(text, e.Right, out)
	case *syntax.ComparisonExpr:
		for _, operand := range e.Operands {
			collectExprRows(text, operand, out)
		}
	case *syntax.ArrayExpr:
		for _, element := range e.Elements {
			collectExprRows(text, element.Value, out)
		}
	case *syntax.FieldExpr:
		collectExprRows(text, e.Receiver, out)
	case *syntax.IndexExpr:
		collectExprRows(text, e.Receiver, out)
		collectExprRows(text, e.Index, out)
	case *syntax.SliceExpr:
		collectExprRows(text, e.Receiver, out)
		collectExprRows(text, e.Start, out)
		collectExprRows(text, e.End, out)
	case *syntax.ConstructorExpr:
		for _, argument := range e.Arguments {
			collectExprRows(text, argument.Value, out)
		}
	case *syntax.CallExpr:
		collectExprRows(text, e.Invocation.Callee, out)
		for _, argument := range e.Invocation.Arguments {
			collectExprRows(text, argument.Value, out)
		}
		for _, method := range e.Methods {
			for _, argument := range method.Arguments {
				collectExprRows(text, argument.Value, out)
			}
		}
	case *syntax.ReferenceExpr:
		collectExprRows(text, e.Callee, out)
		for _, pinned := range e.Bindings {
			collectExprRows(text, pinned.Value, out)
		}
	case *syntax.UpdateExpr:
		collectExprRows(text, e.Receiver, out)
		for _, field := range e.Fields {
			collectExprRows(text, field.Value, out)
		}
	case *syntax.MatchExpr:
		collectMatchRows(text, &e.Match, out)
	case *syntax.CoordinationExpr:
		collectCoordinationRows(text, &e.Coordination, out)
	}
}

// ownershipCounts counts relay and inherit terminals anywhere in a body.
func ownershipCounts(body syntax.Body) (relay, inherit int) {
	switch b := body.(type) {
	case *syntax.RelayBody:
		return 1, 0
	case *syntax.InheritBody:
		return 0, 1
	case *syntax.DoBody:
		return ownershipBlock(&b.Block)
	case *syntax.MatchBody:
		return ownershipMatch(&b.Match)
	}
	return 0, 0
}

func ownershipBlock(block *syntax.Block) (relay, inherit int) {
	for _, step := range block.Steps {
		switch s := step.(type) {
		case *syntax.BindingStep:
			r, h := ownershipExpr(s.Value)
			relay, inherit = relay+r, inherit+h
		case *syntax.CallStep:
			if s.Call != nil {
				r, h := ownershipExpr(s.Call)
				relay, inherit = relay+r, inherit+h
			}
		case *syntax.CoordinationStep:
			r, h := ownershipCoordination(&s.Coordination)
			relay, inherit = relay+r, inherit+h
		}
	}
	if block.Terminal != nil {
		r, h := ownershipCounts(block.Terminal)
		relay, inherit = relay+r, inherit+h
	}
	return relay, inherit
}

func ownershipMatch(match *syntax.Match) (relay, inherit int) {
	for i := range match.Arms {
		if match.Arms[i].Body != nil {
			r, h := ownershipCounts(match.Arms[i].Body)
			relay, inherit = relay+r, inherit+h
		}
	}
	for i := range match.When {
		if match.When[i].Expected != nil {
			r, h := ownershipCounts(match.When[i].Expected)
			relay, inherit = relay+r, inherit+h
		}
	}
	for i := range match.Values {
		r, h := ownershipExpr(match.Values[i])
		relay, inherit = relay+r, inherit+h
	}
	if match.Call != nil {
		r, h := ownershipExpr(match.Call)
		relay, inherit = relay+r, inherit+h
	}
	for i := range match.Chain {
		if match.Chain[i].Call != nil {
			r, h := ownershipExpr(match.Chain[i].Call)
			relay, inherit = relay+r, inherit+h
		}
	}
	return relay, inherit
}

func ownershipCoordination(coord *syntax.Coordination) (relay, inherit int) {
	for i := range coord.Participants {
		participant := &coord.Participants[i]
		if participant.Call != nil {
			r, h := ownershipExpr(participant.Call)
			relay, inherit = relay+r, inherit+h
		}
		if participant.Spread != nil {
			r, h := ownershipExpr(participant.Spread)
			relay, inherit = relay+r, inherit+h
		}
		for a := range participant.Arms {
			if participant.Arms[a].Body != nil {
				r, h := ownershipCounts(participant.Arms[a].Body)
				relay, inherit = relay+r, inherit+h
			}
		}
	}
	for i := range coord.Arms {
		if coord.Arms[i].Body != nil {
			r, h := ownershipCounts(coord.Arms[i].Body)
			relay, inherit = relay+r, inherit+h
		}
	}
	return relay, inherit
}

func ownershipExpr(expr syntax.Expr) (relay, inherit int) {
	switch e := expr.(type) {
	case *syntax.GroupExpr:
		return ownershipExpr(e.Value)
	case *syntax.UnaryExpr:
		return ownershipExpr(e.Operand)
	case *syntax.BinaryExpr:
		r1, h1 := ownershipExpr(e.Left)
		r2, h2 := ownershipExpr(e.Right)
		return r1 + r2, h1 + h2
	case *syntax.ComparisonExpr:
		for _, operand := range e.Operands {
			r, h := ownershipExpr(operand)
			relay, inherit = relay+r, inherit+h
		}
	case *syntax.ArrayExpr:
		for _, element := range e.Elements {
			r, h := ownershipExpr(element.Value)
			relay, inherit = relay+r, inherit+h
		}
	case *syntax.FieldExpr:
		return ownershipExpr(e.Receiver)
	case *syntax.IndexExpr:
		r1, h1 := ownershipExpr(e.Receiver)
		r2, h2 := ownershipExpr(e.Index)
		return r1 + r2, h1 + h2
	case *syntax.SliceExpr:
		r1, h1 := ownershipExpr(e.Receiver)
		r2, h2 := ownershipExpr(e.Start)
		r3, h3 := ownershipExpr(e.End)
		return r1 + r2 + r3, h1 + h2 + h3
	case *syntax.ConstructorExpr:
		for _, argument := range e.Arguments {
			r, h := ownershipExpr(argument.Value)
			relay, inherit = relay+r, inherit+h
		}
	case *syntax.CallExpr:
		r, h := ownershipExpr(e.Invocation.Callee)
		relay, inherit = r, h
		for _, argument := range e.Invocation.Arguments {
			r, h := ownershipExpr(argument.Value)
			relay, inherit = relay+r, inherit+h
		}
		for _, method := range e.Methods {
			for _, argument := range method.Arguments {
				r, h := ownershipExpr(argument.Value)
				relay, inherit = relay+r, inherit+h
			}
		}
	case *syntax.ReferenceExpr:
		relay, inherit := ownershipExpr(e.Callee)
		for _, pinned := range e.Bindings {
			r, h := ownershipExpr(pinned.Value)
			relay, inherit = relay+r, inherit+h
		}
		return relay, inherit
	case *syntax.UpdateExpr:
		r, h := ownershipExpr(e.Receiver)
		relay, inherit = r, h
		for _, field := range e.Fields {
			r, h := ownershipExpr(field.Value)
			relay, inherit = relay+r, inherit+h
		}
	case *syntax.MatchExpr:
		return ownershipMatch(&e.Match)
	case *syntax.CoordinationExpr:
		return ownershipCoordination(&e.Coordination)
	}
	return relay, inherit
}

func spanText(text string, span source.Span) string {
	if span.Start < 0 || span.End < span.Start || span.End > len(text) {
		return ""
	}
	return text[span.Start:span.End]
}
