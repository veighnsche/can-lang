package driver

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/editortrace"
	"github.com/veighnsche/can-lang/compiler/internal/project"
	compileresolve "github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
)

// Diagnostic is one editor squiggle: a canonical file, zero-based lines,
// UTF-16 columns, the pipeline's own code and message, a severity, and
// optional secondary locations. Lexing, parsing, resolution, and checking
// supply structured spans. Spanless failures carry an empty File and are
// reported as project status instead of source squiggles.
type Diagnostic struct {
	File     string
	Line     int
	EndLine  int
	Start    int
	End      int
	Code     string
	Message  string
	Severity string
	Related  []RelatedDiagnostic
	// Fixes carries compiler-proposed repairs for this diagnostic. Entries
	// are candidates only; drivers must validate each through ValidateFix
	// and present only fixes that validate.
	Fixes []Fix
}

// RelatedDiagnostic is one secondary location for a diagnostic: the
// canonical file, zero-based line range, UTF-16 columns, and the note
// naming its relationship to the primary span.
type RelatedDiagnostic struct {
	File    string
	Line    int
	EndLine int
	Start   int
	End     int
	Message string
}

// Location is a go-to-definition target in editor coordinates.
type Location struct {
	File  string
	Line  int
	Start int
	End   int
}

// Snapshot is one inert diagnosis over disk plus overlays: the loaded
// graph, the resolved world when resolution succeeded, and the CLI-first
// diagnostics. Versions records the overlay document version behind each
// diagnosed file so fix validation can reject stale edits. It never
// builds, runs, asserts, dials out, queries, reads the environment,
// emits files, or mutates registries.
type Snapshot struct {
	Analysis    *Analysis
	Program     *check.Program
	Graph       *project.Graph
	World       *compileresolve.World
	Diagnostics []Diagnostic
	Versions    map[string]int64
}

// CheckSnapshot loads the named project with overlay substitution and runs
// the current resolve and check stages without requiring an entry point,
// so library files diagnose like programs. Every pipeline failure becomes
// a diagnostic; the returned error is only for programmer misuse.
func CheckSnapshot(directory, openFile string, overlay *project.Overlay) (*Snapshot, error) {
	return CheckSnapshotContext(context.Background(), directory, openFile, overlay)
}

func CheckSnapshotContext(ctx context.Context, directory, openFile string, overlay *project.Overlay) (*Snapshot, error) {
	if directory == "" {
		return nil, fmt.Errorf("diagnose project: empty directory")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	graph, loadErr := project.LoadWithOverlayContext(ctx, directory, overlay)
	return analyzeGraph(ctx, directory, graph, loadErr, overlay)
}

func analyzeGraph(ctx context.Context, directory string, graph *project.Graph, loadErr error, overlay *project.Overlay) (*Snapshot, error) {
	defer editortrace.Stage("snapshot")()
	snapshot := &Snapshot{Graph: graph, Versions: map[string]int64{}}
	if overlay != nil {
		snapshot.Versions = overlay.Versions()
	}
	builder := NewBuilder(directory)
	builder.SetGraph(graph)
	addError := func(err error) {
		for _, issue := range errorIssues(err) {
			builder.AddIssue(issue)
		}
	}
	addError(loadErr)
	if graph != nil {
		for path, data := range graph.Inputs {
			builder.AddSource(DocInput{Path: path, Version: snapshot.Versions[path], Text: string(data)})
		}
		if graph.Root != nil {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			snapshot.World = compileresolve.BuildRecoveringContext(ctx, graph)
			builder.SetWorld(snapshot.World)
			for _, err := range snapshot.World.Errors {
				addError(err)
			}
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			// The checker consumes this same resolved world; recovery never
			// replays the project or changes the source to expose later errors.
			program, err := check.AnalyzeProgramContext(ctx, graph, snapshot.World)
			snapshot.Program = program
			addError(err)
			if program != nil {
				for _, warning := range program.Warnings {
					severity := warning.Severity
					if severity == "" {
						severity = source.SeverityWarning
					}
					builder.AddIssue(source.Issue{File: warning.File, Span: warning.Span, Code: warning.Code, Severity: severity, Message: warning.Message})
				}
			}
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if snapshot.World != nil {
		for src, file := range snapshot.World.Files {
			status := source.UnitValid
			if len(src.Syntax.Invalid) > 0 {
				status = source.UnitInvalid
			}
			builder.SetUnit(UnitID(src.Path), UnitReport{Status: status})
			for _, symbol := range file.Package.Scope.Symbols {
				if symbol.Source != src {
					continue
				}
				report := UnitReport{Status: source.UnitValid}
				if invalid := snapshot.World.Invalid[symbol.Declaration]; invalid != nil {
					report.Status = source.UnitInvalid
					var blocked *source.BlockedError
					if source.IsBlocked(invalid) && errors.As(invalid, &blocked) {
						report.Status = source.UnitBlocked
						report.BlockedBy = []string{blocked.Dependency}
					}
				}
				builder.SetUnit(UnitID(symbol.ID), report)
			}
		}
	}
	snapshot.Analysis = builder.Seal()
	snapshot.Analysis.Program = snapshot.Program
	for _, issue := range snapshot.Analysis.Issues {
		snapshot.Diagnostics = append(snapshot.Diagnostics, issueDiagnostic(graph, issue))
	}
	return snapshot, nil
}

func errorIssues(err error) []source.Issue {
	if err == nil {
		return nil
	}
	if many, ok := err.(interface{ Unwrap() []error }); ok {
		var issues []source.Issue
		for _, child := range many.Unwrap() {
			issues = append(issues, errorIssues(child)...)
		}
		return issues
	}
	if _, located := err.(*source.LocatedError); !located {
		if wrapped, ok := err.(interface{ Unwrap() error }); ok {
			// Editor identity uses the producer's semantic message. CLI
			// wrapper context can contain shifted byte offsets or renamed
			// function identities; structured ranges already carry location.
			return errorIssues(wrapped.Unwrap())
		}
	}
	if source.IsBlocked(err) {
		return nil
	}
	if parsed, ok := err.(*project.SourceError); ok {
		var issues []source.Issue
		for _, diagnostic := range parsed.Diagnostics {
			issues = append(issues, source.Issue{File: parsed.Path, Span: diagnostic.Span, Code: diagnostic.Code, Severity: source.SeverityError, Message: diagnostic.Message})
		}
		if len(issues) > 0 {
			return issues
		}
	}
	issue := source.Issue{Code: "CAN-PROJECT", Severity: source.SeverityError, Message: err.Error()}
	if located, ok := source.AsLocated(err); ok {
		issue.File, issue.Span, issue.Code, issue.Related, issue.Fixes = located.File, located.Span, located.Code, located.Related, located.Fixes
		if issue.Code == "" {
			issue.Code = "CAN-CHECK"
		}
	}
	return []source.Issue{issue}
}

func issueDiagnostic(graph *project.Graph, issue source.Issue) Diagnostic {
	diagnostic := Diagnostic{File: issue.File, Code: issue.Code, Severity: issue.Severity, Message: issue.Message}
	if issue.File == "" {
		return diagnostic
	}
	converted := semanticDiagnostic(graph, "", &source.LocatedError{File: issue.File, Span: issue.Span, Code: issue.Code, Related: issue.Related, Fixes: issue.Fixes, Err: fmt.Errorf("%s", issue.Message)})
	converted.Severity = issue.Severity
	return converted
}

// semanticDiagnostic converts a structured resolver or checker failure to
// its primary editor position with related locations. Spanless failures
// become project status. Unavailable bytes or invalid offsets retain the
// original file in the message and carry an explicit unavailable marker.
func semanticDiagnostic(graph *project.Graph, openFile string, err error) Diagnostic {
	located, ok := source.AsLocated(err)
	if !ok {
		return anchored(graph, openFile, "", err.Error())
	}
	diagnostic := Diagnostic{File: located.File, Code: located.Code, Message: err.Error(), Severity: "error"}
	text, ok := fileText(graph, located.File)
	if !ok {
		diagnostic.Code = unavailableCode(located.Code)
		diagnostic.File = ""
		diagnostic.Message = located.File + ": " + diagnostic.Message
		return diagnostic
	}
	file, fileErr := source.New(located.File, text)
	if fileErr != nil {
		diagnostic.Code = unavailableCode(located.Code)
		diagnostic.File = ""
		diagnostic.Message = located.File + ": " + diagnostic.Message
		return diagnostic
	}
	start, startErr := file.UTF16Position(located.Span.Start)
	end, endErr := file.UTF16Position(located.Span.End)
	if startErr != nil || endErr != nil {
		diagnostic.Code = unavailableCode(located.Code)
		diagnostic.File = ""
		diagnostic.Message = located.File + ": " + diagnostic.Message
		return diagnostic
	}
	diagnostic.Line, diagnostic.Start = start.Line, start.Character
	diagnostic.EndLine, diagnostic.End = end.Line, end.Character
	for _, related := range located.Related {
		diagnostic.Related = append(diagnostic.Related, convertRelated(graph, related))
	}
	for _, fix := range located.Fixes {
		diagnostic.Fixes = append(diagnostic.Fixes, Fix{Title: fix.Title, File: fix.File, Start: fix.Start, End: fix.End, NewText: fix.Text})
	}
	return diagnostic
}

func convertRelated(graph *project.Graph, related source.RelatedSpan) RelatedDiagnostic {
	converted := RelatedDiagnostic{File: related.File, Message: related.Note}
	text, ok := fileText(graph, related.File)
	if !ok {
		converted.Message = related.File + ": " + converted.Message + " [position unavailable]"
		converted.File = ""
		return converted
	}
	file, fileErr := source.New(related.File, text)
	if fileErr != nil {
		converted.Message = related.File + ": " + converted.Message + " [position unavailable]"
		converted.File = ""
		return converted
	}
	start, startErr := file.UTF16Position(related.Span.Start)
	end, endErr := file.UTF16Position(related.Span.End)
	if startErr != nil || endErr != nil {
		converted.Message = related.File + ": " + converted.Message + " [position unavailable]"
		converted.File = ""
		return converted
	}
	converted.Line, converted.Start = start.Line, start.Character
	converted.EndLine, converted.End = end.Line, end.Character
	return converted
}

func unavailableCode(code string) string {
	if code == "" {
		return source.SpanUnavailable
	}
	return code + " " + source.SpanUnavailable
}

// anchored retains a spanless compiler failure as project status.
func anchored(graph *project.Graph, openFile, code, message string) Diagnostic {
	return Diagnostic{Code: code, Message: message, Severity: source.SeverityError}
}

func fileText(graph *project.Graph, file string) (string, bool) {
	if graph == nil {
		return "", false
	}
	if data, ok := graph.Inputs[file]; ok && data != nil {
		return string(data), true
	}
	for _, p := range graph.Projects {
		for _, s := range p.Sources {
			if s.Path == file {
				return string(s.Bytes), true
			}
		}
	}
	return "", false
}

// Definition resolves the identifier at an editor offset to its declaring
// span through file, package, prelude, and import scopes: nominal types,
// calls, constructors, module values, and generated declarations,
// including record and generated-record fields behind annotation-known
// receivers. Body-local names decline rather than guess, and catalogue or
// prelude symbols without a source file decline as unjumpable.
func Definition(snapshot *Snapshot, file string, line, character int) (Location, bool, error) {
	var none Location
	if snapshot == nil || snapshot.World == nil || snapshot.Graph == nil {
		return none, false, nil
	}
	canonical, err := filepath.EvalSymlinks(file)
	if err != nil {
		return none, false, nil
	}
	var src *project.Source
	for _, p := range snapshot.Graph.Projects {
		for _, s := range p.Sources {
			if s.Path == canonical {
				src = s
			}
		}
	}
	if src == nil || src.Syntax == nil {
		return none, false, nil
	}
	resolved, ok := snapshot.World.Files[src]
	if !ok {
		return none, false, nil
	}
	text, err := source.New(canonical, string(src.Bytes))
	if err != nil {
		return none, false, nil
	}
	offset, err := text.Offset(source.UTF16Position{Line: line, Character: character})
	if err != nil {
		return none, false, nil
	}
	reference := findReference(src.Syntax, offset)
	if reference.empty() {
		return none, false, nil
	}
	if reference.self {
		return convertSpan(text, canonical, reference.span)
	}
	if reference.member != "" {
		return memberLocation(resolved, reference)
	}
	if reference.shadowable {
		if shadowedByLocal(src.Syntax, offset, reference.name.Name) {
			return none, false, nil
		}
	}
	symbol, err := resolved.Lookup(nil, reference.name, reference.usage)
	if err != nil {
		return none, false, nil
	}
	return symbolLocation(symbol)
}

type reference struct {
	name       syntax.QualifiedName
	usage      compileresolve.Usage
	receiver   syntax.Expr
	member     string
	method     bool
	span       source.Span
	self       bool
	shadowable bool
}

func (r reference) empty() bool {
	return r.name.Name == "" && r.member == "" && !r.self
}

func contains(span source.Span, offset int) bool {
	return span.Start <= offset && offset < span.End
}

// findReference walks declarations for the innermost name at the offset,
// preferring type, call, and constructor positions over bare names.
func findReference(file *syntax.File, offset int) reference {
	for _, declaration := range file.Declarations {
		if found := findInDecl(declaration, offset); !found.empty() {
			return found
		}
	}
	return reference{}
}

func findInDecl(declaration syntax.Declaration, offset int) reference {
	switch node := declaration.(type) {
	case *syntax.FunctionDecl:
		if contains(node.Name.Span, offset) {
			return reference{span: node.Name.Span, self: true}
		}
		if found := findInType(node.Result, compileresolve.TypeUse, offset); !found.empty() {
			return found
		}
		if node.Receiver != nil {
			if found := findInType(node.Receiver.Type, compileresolve.TypeUse, offset); !found.empty() {
				return found
			}
		}
		if found := findInErrorBound(node.Errors, offset); !found.empty() {
			return found
		}
		for i := range node.Inputs {
			if found := findInType(node.Inputs[i].Type, compileresolve.TypeUse, offset); !found.empty() {
				return found
			}
		}
		if found := findInBlock(node.Body, offset); !found.empty() {
			return found
		}
		for _, assertion := range node.Assertions {
			if found := findInAssertion(&assertion, offset); !found.empty() {
				return found
			}
		}
	case *syntax.RecordDecl:
		if contains(node.Name.Span, offset) {
			return reference{span: node.Name.Span, self: true}
		}
		for i := range node.Fields {
			if contains(node.Fields[i].Name.Span, offset) {
				return reference{span: node.Fields[i].Name.Span, self: true}
			}
			if found := findInType(node.Fields[i].Type, compileresolve.TypeUse, offset); !found.empty() {
				return found
			}
		}
	case *syntax.VariantDecl:
		if contains(node.Name.Span, offset) {
			return reference{span: node.Name.Span, self: true}
		}
		for _, alternative := range node.Alternatives {
			if found := findInType(alternative, compileresolve.TypeUse, offset); !found.empty() {
				return found
			}
		}
	case *syntax.ErrorDecl:
		if contains(node.Name.Span, offset) {
			return reference{span: node.Name.Span, self: true}
		}
		for i := range node.Fields {
			if contains(node.Fields[i].Name.Span, offset) {
				return reference{span: node.Fields[i].Name.Span, self: true}
			}
			if found := findInType(node.Fields[i].Type, compileresolve.TypeUse, offset); !found.empty() {
				return found
			}
		}
	case *syntax.ValueDecl:
		if contains(node.Binding.Name.Span, offset) {
			return reference{span: node.Binding.Name.Span, self: true}
		}
		if found := findInType(node.Binding.Type, compileresolve.TypeUse, offset); !found.empty() {
			return found
		}
		if found := findInExpr(node.Binding.Value, offset); !found.empty() {
			return found
		}
	case *syntax.QuestionDecl:
		if node.RecordName != nil && contains(node.RecordName.Span, offset) {
			return reference{span: node.RecordName.Span, self: true}
		}
		for _, option := range node.Options {
			if option.Name != nil && contains(option.Name.Span, offset) {
				return reference{span: option.Name.Span, self: true}
			}
			if found := findInExpr(option.Description, offset); !found.empty() {
				return found
			}
			if found := findInExpr(option.Spread, offset); !found.empty() {
				return found
			}
			if found := findInBody(option.Body, offset); !found.empty() {
				return found
			}
		}
		if found := findInExpr(node.Asks, offset); !found.empty() {
			return found
		}
		if found := findInExpr(node.Minimum, offset); !found.empty() {
			return found
		}
		if found := findInBody(node.Fallback, offset); !found.empty() {
			return found
		}
		if found := findInBody(node.Shared, offset); !found.empty() {
			return found
		}
	}
	return reference{}
}

func findInType(node syntax.TypeNode, usage compileresolve.Usage, offset int) reference {
	switch node := node.(type) {
	case *syntax.NamedType:
		for _, argument := range node.Arguments {
			if found := findInType(argument, compileresolve.TypeUse, offset); !found.empty() {
				return found
			}
		}
		if contains(node.Name.Span, offset) {
			return reference{name: node.Name, usage: usage}
		}
	case *syntax.ArrayType:
		return findInType(node.Element, compileresolve.TypeUse, offset)
	case *syntax.CallableType:
		if found := findInType(node.Result, compileresolve.TypeUse, offset); !found.empty() {
			return found
		}
		for _, input := range node.Inputs {
			if found := findInType(input, compileresolve.TypeUse, offset); !found.empty() {
				return found
			}
		}
		return findInErrorBound(node.Errors, offset)
	case *syntax.ChoiceArmType:
		if found := findInType(node.Result, compileresolve.TypeUse, offset); !found.empty() {
			return found
		}
		return findInErrorBound(node.Errors, offset)
	}
	return reference{}
}

func findInErrorBound(bound syntax.ErrorBound, offset int) reference {
	for _, typ := range bound.Types {
		if found := findInType(typ, compileresolve.ErrorUse, offset); !found.empty() {
			return found
		}
	}
	return reference{}
}

func findInBlock(block syntax.Block, offset int) reference {
	for _, step := range block.Steps {
		if found := findInStep(step, offset); !found.empty() {
			return found
		}
	}
	return findInBody(block.Terminal, offset)
}

func findInStep(step syntax.Step, offset int) reference {
	switch node := step.(type) {
	case *syntax.BindingStep:
		if found := findInType(node.Binding.Type, compileresolve.TypeUse, offset); !found.empty() {
			return found
		}
		return findInExpr(node.Binding.Value, offset)
	case *syntax.CallStep:
		return findInExpr(node.Call, offset)
	case *syntax.CoordinationStep:
		return findInCoordination(&node.Coordination, offset)
	}
	return reference{}
}

func findInBody(body syntax.Body, offset int) reference {
	switch node := body.(type) {
	case *syntax.ValueBody:
		return findInExpr(node.Value, offset)
	case *syntax.SuccessBody:
		return findInExpr(node.Value, offset)
	case *syntax.FailureBody:
		if node.Error == nil {
			return reference{}
		}
		return findInExpr(node.Error, offset)
	case *syntax.RelayBody:
		if node.Call == nil {
			return reference{}
		}
		return findInExpr(node.Call, offset)
	case *syntax.DoBody:
		return findInBlock(node.Block, offset)
	case *syntax.MatchBody:
		return findInMatch(&node.Match, offset)
	}
	return reference{}
}

func findInMatch(match *syntax.Match, offset int) reference {
	for _, value := range match.Values {
		if found := findInExpr(value, offset); !found.empty() {
			return found
		}
	}
	if match.Call != nil {
		if found := findInExpr(match.Call, offset); !found.empty() {
			return found
		}
	}
	for i := range match.Chain {
		if found := findInExpr(match.Chain[i].Call, offset); !found.empty() {
			return found
		}
	}
	for i := range match.When {
		if found := findInAssertion(&match.When[i], offset); !found.empty() {
			return found
		}
	}
	for i := range match.Arms {
		arm := &match.Arms[i]
		for _, pattern := range arm.Patterns {
			if found := findInPattern(pattern, offset); !found.empty() {
				return found
			}
		}
		if found := findInCompletionHeads(arm, offset); !found.empty() {
			return found
		}
		if found := findInBody(arm.Body, offset); !found.empty() {
			return found
		}
	}
	return reference{}
}

func findInCompletionHeads(arm *syntax.MatchArm, offset int) reference {
	if arm.Outcome != nil && arm.Outcome.Error != nil {
		if found := findInType(arm.Outcome.Error, compileresolve.ErrorUse, offset); !found.empty() {
			return found
		}
	}
	for _, outcome := range arm.AlternateOutcomes {
		if outcome.Error != nil {
			if found := findInType(outcome.Error, compileresolve.ErrorUse, offset); !found.empty() {
				return found
			}
		}
	}
	return reference{}
}

func findInCoordination(coordination *syntax.Coordination, offset int) reference {
	for i := range coordination.Participants {
		participant := &coordination.Participants[i]
		if participant.Call != nil {
			if found := findInExpr(participant.Call, offset); !found.empty() {
				return found
			}
		}
		if found := findInExpr(participant.Spread, offset); !found.empty() {
			return found
		}
		for j := range participant.Arms {
			arm := &participant.Arms[j]
			if found := findInCompletionHeads(arm, offset); !found.empty() {
				return found
			}
			for _, pattern := range arm.Patterns {
				if found := findInPattern(pattern, offset); !found.empty() {
					return found
				}
			}
			if found := findInBody(arm.Body, offset); !found.empty() {
				return found
			}
		}
	}
	for i := range coordination.Arms {
		arm := &coordination.Arms[i]
		if found := findInCompletionHeads(arm, offset); !found.empty() {
			return found
		}
		for _, pattern := range arm.Patterns {
			if found := findInPattern(pattern, offset); !found.empty() {
				return found
			}
		}
		if found := findInBody(arm.Body, offset); !found.empty() {
			return found
		}
	}
	return reference{}
}

func findInPattern(pattern syntax.PatternNode, offset int) reference {
	switch node := pattern.(type) {
	case *syntax.ConstructorPattern:
		for _, field := range node.Fields {
			if found := findInPattern(field, offset); !found.empty() {
				return found
			}
		}
		for _, typ := range node.Types {
			if found := findInType(typ, compileresolve.TypeUse, offset); !found.empty() {
				return found
			}
		}
		if contains(node.Name.Span, offset) {
			return reference{name: node.Name, usage: compileresolve.ConstructorUse}
		}
	case *syntax.NamePattern:
		for _, typ := range node.Types {
			if found := findInType(typ, compileresolve.TypeUse, offset); !found.empty() {
				return found
			}
		}
	case *syntax.ArrayPattern:
		for _, element := range node.Elements {
			if found := findInPattern(element, offset); !found.empty() {
				return found
			}
		}
	case *syntax.AlternativePattern:
		for _, alternative := range node.Alternatives {
			if found := findInPattern(alternative, offset); !found.empty() {
				return found
			}
		}
	}
	return reference{}
}

func findInAssertion(assertion *syntax.Assertion, offset int) reference {
	if found := findInExpr(assertion.Receiver, offset); !found.empty() {
		return found
	}
	for i := range assertion.Arguments {
		if found := findInExpr(assertion.Arguments[i].Value, offset); !found.empty() {
			return found
		}
		for _, value := range groupValues(&assertion.Arguments[i]) {
			if found := findInExpr(value, offset); !found.empty() {
				return found
			}
		}
	}
	return findInBody(assertion.Expected, offset)
}

func groupValues(argument *syntax.Argument) []syntax.Expr {
	if argument.Group == nil {
		return nil
	}
	return argument.Group.Values
}

func findInExpr(expr syntax.Expr, offset int) reference {
	switch node := expr.(type) {
	case *syntax.NameExpr:
		if contains(node.Name.Span, offset) {
			return reference{name: node.Name, usage: compileresolve.ValueUse, shadowable: true}
		}
	case *syntax.ConstructorExpr:
		for _, typ := range node.Types {
			if found := findInType(typ, compileresolve.TypeUse, offset); !found.empty() {
				return found
			}
		}
		for i := range node.Arguments {
			if found := findInArgument(&node.Arguments[i], offset); !found.empty() {
				return found
			}
		}
		if contains(node.Name.Span, offset) {
			return reference{name: node.Name, usage: compileresolve.ConstructorUse}
		}
	case *syntax.CallExpr:
		if found := findInCallee(node.Invocation.Callee, offset); !found.empty() {
			return found
		}
		for _, typ := range node.Invocation.Types {
			if found := findInType(typ, compileresolve.TypeUse, offset); !found.empty() {
				return found
			}
		}
		for i := range node.Invocation.Arguments {
			if found := findInArgument(&node.Invocation.Arguments[i], offset); !found.empty() {
				return found
			}
		}
		for i := range node.Methods {
			method := &node.Methods[i]
			if contains(method.Name.Span, offset) {
				return reference{receiver: node.Invocation.Callee, member: method.Name.Text, method: true, span: method.Name.Span}
			}
			for _, typ := range method.Types {
				if found := findInType(typ, compileresolve.TypeUse, offset); !found.empty() {
					return found
				}
			}
			for j := range method.Arguments {
				if found := findInArgument(&method.Arguments[j], offset); !found.empty() {
					return found
				}
			}
		}
	case *syntax.ReferenceExpr:
		if name, ok := node.Callee.(*syntax.NameExpr); ok && contains(name.Name.Span, offset) {
			return reference{name: name.Name, usage: compileresolve.CallUse, shadowable: true}
		}
		if found := findInExpr(node.Callee, offset); !found.empty() {
			return found
		}
		for _, typ := range node.Types {
			if found := findInType(typ, compileresolve.TypeUse, offset); !found.empty() {
				return found
			}
		}
	case *syntax.FieldExpr:
		if contains(node.Field.Span, offset) {
			return reference{receiver: node.Receiver, member: node.Field.Text, span: node.Field.Span}
		}
		return findInExpr(node.Receiver, offset)
	case *syntax.GroupExpr:
		return findInExpr(node.Value, offset)
	case *syntax.UnaryExpr:
		return findInExpr(node.Operand, offset)
	case *syntax.BinaryExpr:
		if found := findInExpr(node.Left, offset); !found.empty() {
			return found
		}
		return findInExpr(node.Right, offset)
	case *syntax.ComparisonExpr:
		for _, operand := range node.Operands {
			if found := findInExpr(operand, offset); !found.empty() {
				return found
			}
		}
	case *syntax.ArrayExpr:
		for i := range node.Elements {
			if found := findInArgument(&node.Elements[i], offset); !found.empty() {
				return found
			}
		}
	case *syntax.IndexExpr:
		if found := findInExpr(node.Receiver, offset); !found.empty() {
			return found
		}
		return findInExpr(node.Index, offset)
	case *syntax.SliceExpr:
		if found := findInExpr(node.Receiver, offset); !found.empty() {
			return found
		}
		if found := findInExpr(node.Start, offset); !found.empty() {
			return found
		}
		return findInExpr(node.End, offset)
	case *syntax.UpdateExpr:
		if found := findInExpr(node.Receiver, offset); !found.empty() {
			return found
		}
		for i := range node.Fields {
			if found := findInExpr(node.Fields[i].Value, offset); !found.empty() {
				return found
			}
		}
	case *syntax.MatchExpr:
		return findInMatch(&node.Match, offset)
	case *syntax.CoordinationExpr:
		return findInCoordination(&node.Coordination, offset)
	}
	return reference{}
}

func findInCallee(callee syntax.Expr, offset int) reference {
	if name, ok := callee.(*syntax.NameExpr); ok && contains(name.Name.Span, offset) {
		return reference{name: name.Name, usage: compileresolve.CallUse, shadowable: true}
	}
	return findInExpr(callee, offset)
}

func findInArgument(argument *syntax.Argument, offset int) reference {
	if found := findInExpr(argument.Value, offset); !found.empty() {
		return found
	}
	for _, value := range groupValues(argument) {
		if found := findInExpr(value, offset); !found.empty() {
			return found
		}
	}
	return reference{}
}

// shadowedByLocal declines bare and callee names bound anywhere in the
// enclosing function body. The set is a deliberate superset: declining a
// shadowed name is always safe, while jumping past a local never is.
func shadowedByLocal(file *syntax.File, offset int, name string) bool {
	for _, declaration := range file.Declarations {
		fn, ok := declaration.(*syntax.FunctionDecl)
		if !ok || !contains(fn.DeclSpan(), offset) {
			continue
		}
		binders := map[string]bool{}
		for _, parameter := range fn.Parameters {
			binders[parameter.Text] = true
		}
		for _, input := range fn.Inputs {
			binders[input.Name.Text] = true
		}
		collectBinders(fn.Body, binders)
		for i := range fn.Assertions {
			collectAssertionBinders(&fn.Assertions[i], binders)
		}
		return binders[name]
	}
	return false
}

func collectBinders(block syntax.Block, binders map[string]bool) {
	for _, step := range block.Steps {
		switch node := step.(type) {
		case *syntax.BindingStep:
			binders[node.Binding.Name.Text] = true
			collectExprBinders(node.Binding.Value, binders)
		case *syntax.CallStep:
			collectExprBinders(node.Call, binders)
		case *syntax.CoordinationStep:
			collectCoordinationBinders(&node.Coordination, binders)
		}
	}
	collectBodyBinders(block.Terminal, binders)
}

func collectBodyBinders(body syntax.Body, binders map[string]bool) {
	switch node := body.(type) {
	case *syntax.ValueBody:
		collectExprBinders(node.Value, binders)
	case *syntax.SuccessBody:
		collectExprBinders(node.Value, binders)
	case *syntax.FailureBody:
		if node.Error != nil {
			collectExprBinders(node.Error, binders)
		}
	case *syntax.RelayBody:
		collectExprBinders(node.Call, binders)
	case *syntax.DoBody:
		collectBinders(node.Block, binders)
	case *syntax.MatchBody:
		collectMatchBinders(&node.Match, binders)
	}
}

func collectMatchBinders(match *syntax.Match, binders map[string]bool) {
	for _, value := range match.Values {
		collectExprBinders(value, binders)
	}
	if match.Call != nil {
		collectExprBinders(match.Call, binders)
	}
	for i := range match.Chain {
		if match.Chain[i].Binding != nil {
			binders[match.Chain[i].Binding.Name.Text] = true
		}
	}
	for i := range match.Arms {
		arm := &match.Arms[i]
		for _, pattern := range arm.Patterns {
			collectPatternBinders(pattern, binders)
		}
		if arm.Outcome != nil && arm.Outcome.Binding != nil {
			binders[arm.Outcome.Binding.Name.Text] = true
		}
		collectBodyBinders(arm.Body, binders)
	}
}

func collectCoordinationBinders(coordination *syntax.Coordination, binders map[string]bool) {
	for i := range coordination.Participants {
		collectExprBinders(coordination.Participants[i].Call, binders)
		collectExprBinders(coordination.Participants[i].Spread, binders)
		for j := range coordination.Participants[i].Arms {
			arm := &coordination.Participants[i].Arms[j]
			for _, pattern := range arm.Patterns {
				collectPatternBinders(pattern, binders)
			}
			collectBodyBinders(arm.Body, binders)
		}
	}
	for i := range coordination.Arms {
		arm := &coordination.Arms[i]
		for _, pattern := range arm.Patterns {
			collectPatternBinders(pattern, binders)
		}
		collectBodyBinders(arm.Body, binders)
	}
}

func collectPatternBinders(pattern syntax.PatternNode, binders map[string]bool) {
	switch node := pattern.(type) {
	case *syntax.BindPattern:
		binders[node.Name.Text] = true
	case *syntax.ConstructorPattern:
		for _, field := range node.Fields {
			collectPatternBinders(field, binders)
		}
	case *syntax.ArrayPattern:
		for _, element := range node.Elements {
			collectPatternBinders(element, binders)
		}
		if node.Rest != nil {
			binders[node.Rest.Text] = true
		}
	case *syntax.AlternativePattern:
		for _, alternative := range node.Alternatives {
			collectPatternBinders(alternative, binders)
		}
	}
}

func collectAssertionBinders(assertion *syntax.Assertion, binders map[string]bool) {
	// Assertion receivers and arguments reference harness values, never bind.
	collectBodyBinders(assertion.Expected, binders)
}

func collectExprBinders(expr syntax.Expr, binders map[string]bool) {
	switch node := expr.(type) {
	case *syntax.MatchExpr:
		collectMatchBinders(&node.Match, binders)
	case *syntax.CoordinationExpr:
		collectCoordinationBinders(&node.Coordination, binders)
	case *syntax.CallExpr:
		for i := range node.Invocation.Arguments {
			collectExprBinders(node.Invocation.Arguments[i].Value, binders)
		}
	}
}

func memberLocation(file *compileresolve.File, ref reference) (Location, bool, error) {
	var none Location
	record, err := receiverRecord(file, ref.receiver)
	if err != nil {
		return none, false, nil
	}
	if ref.method {
		method, err := file.Method(record, ref.member)
		if err != nil {
			return none, false, nil
		}
		return symbolLocation(method)
	}
	decl, ok := record.Declaration.(*syntax.RecordDecl)
	if !ok {
		return none, false, nil
	}
	for i := range decl.Fields {
		if decl.Fields[i].Name.Text == ref.member {
			target, err := source.New(record.Source.Path, string(record.Source.Bytes))
			if err != nil {
				return none, false, nil
			}
			return convertSpan(target, record.Source.Path, decl.Fields[i].Name.Span)
		}
	}
	return none, false, nil
}

// receiverRecord resolves the small set of receivers whose nominal type is
// known without checking: module values with named annotations and direct
// constructor calls. Anything else declines.
func receiverRecord(file *compileresolve.File, receiver syntax.Expr) (*compileresolve.Symbol, error) {
	switch node := receiver.(type) {
	case *syntax.NameExpr:
		symbol, err := file.Lookup(nil, node.Name, compileresolve.ValueUse)
		if err != nil {
			return nil, err
		}
		value, ok := symbol.Declaration.(*syntax.ValueDecl)
		if !ok {
			return nil, fmt.Errorf("not a module value")
		}
		named, ok := value.Binding.Type.(*syntax.NamedType)
		if !ok || named.Name.Package != "" {
			return nil, fmt.Errorf("receiver type is not a local nominal")
		}
		record, err := file.Lookup(nil, named.Name, compileresolve.TypeUse)
		if err != nil {
			return nil, err
		}
		if record.Kind != compileresolve.Record {
			return nil, fmt.Errorf("receiver type is not a record")
		}
		return record, nil
	case *syntax.ConstructorExpr:
		symbol, err := file.Lookup(nil, node.Name, compileresolve.ConstructorUse)
		if err != nil {
			return nil, err
		}
		if symbol.Kind != compileresolve.Record {
			return nil, fmt.Errorf("constructor is not a record")
		}
		return symbol, nil
	}
	return nil, fmt.Errorf("receiver type needs checking")
}

func symbolLocation(symbol *compileresolve.Symbol) (Location, bool, error) {
	var none Location
	if symbol.Source == nil || symbol.Declaration == nil {
		return none, false, nil
	}
	file, err := source.New(symbol.Source.Path, string(symbol.Source.Bytes))
	if err != nil {
		return none, false, nil
	}
	return convertSpan(file, symbol.Source.Path, declarationNameSpan(symbol))
}

func declarationNameSpan(symbol *compileresolve.Symbol) source.Span {
	switch decl := symbol.Declaration.(type) {
	case *syntax.RecordDecl:
		return decl.Name.Span
	case *syntax.VariantDecl:
		return decl.Name.Span
	case *syntax.ErrorDecl:
		return decl.Name.Span
	case *syntax.FunctionDecl:
		return decl.Name.Span
	case *syntax.ValueDecl:
		return decl.Binding.Name.Span
	case *syntax.QuestionDecl:
		if decl.RecordName != nil {
			return decl.RecordName.Span
		}
	}
	return symbol.Declaration.DeclSpan()
}

func convertSpan(file *source.File, path string, span source.Span) (Location, bool, error) {
	var none Location
	if file == nil {
		return none, false, nil
	}
	start, startErr := file.UTF16Position(span.Start)
	end, endErr := file.UTF16Position(span.End)
	if startErr != nil || endErr != nil || start.Line != end.Line {
		return none, false, nil
	}
	return Location{File: path, Line: start.Line, Start: start.Character, End: end.Character}, true, nil
}
