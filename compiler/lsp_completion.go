// canlc completion: scope-aware completion over an inert snapshot.
//
// The service walks the open file once, tracking function-local scopes
// with their spans plus the innermost cursor context (member, with-pin,
// callee, type, qualified package, declaration gaps), then offers the
// exact candidate set for that context: visible locals by binding
// identity and order, captured names from enclosing scopes, checked
// World symbols filtered by usage eligibility, and the selected-surface
// keywords valid there. Callable candidates carry their exact checked
// arity so caller repair shows signatures; near inputs surface both as
// value candidates and as with-pin names behind a resolved callee.
//
// Precision rules mirror the G02/G03 query boundaries: shadowed or
// same-spelled bindings in other scopes are never offered, unselected
// grammars contribute no keywords (A06 is inactive, so C-A adds none),
// member candidates resolve only behind annotation-known receivers,
// with-pins only behind a checked function callee, and anything
// unresolvable — an unloadable snapshot, an unknown receiver or
// callee — yields an empty list or a declined null rather than a
// guess. The walk reads the snapshot and returns freshly built items,
// so no mutable compiler state escapes.
package main

import (
	"path/filepath"
	"sort"

	"github.com/veighnsche/can-lang/compiler/internal/driver"
	"github.com/veighnsche/can-lang/compiler/internal/project"
	compileresolve "github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
)

// LSP CompletionItemKind numbers for the candidates below.
const (
	compText          = 1
	compMethod        = 2
	compFunction      = 3
	compField         = 5
	compVariable      = 6
	compClass         = 7
	compModule        = 9
	compProperty      = 10
	compKeyword       = 14
	compConstant      = 21
	compStruct        = 22
	compTypeParameter = 25
)

// compItem is one completion candidate: the insertable label, its LSP
// kind, a signature or type detail, and a provenance note. Details only
// ever state checked facts — exact arity, declared types, near marks —
// and stay empty where the source carries none.
type compItem struct {
	label string
	kind  int
	tail  string
	doc   string
}

func sortCompItems(items []compItem) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].label != items[j].label {
			return items[i].label < items[j].label
		}
		if items[i].kind != items[j].kind {
			return items[i].kind < items[j].kind
		}
		if items[i].tail != items[j].tail {
			return items[i].tail < items[j].tail
		}
		return items[i].doc < items[j].doc
	})
}

// Keyword sets per cursor context, drawn only from the selected
// surface. Unselected grammars (Q4/Q5/Q6) contribute nothing: A06 is
// inactive, so C-A adds no iteration keywords.
var (
	compGeneralKeywords = []string{"and", "call", "callable", "do", "false", "is", "match", "not", "ok", "or", "relay", "true"}
	compTypeKeywords    = []string{"bool", "float", "int", "str", "void"}
	compTopKeywords     = []string{"bool", "error", "float", "fn", "int", "record", "str", "variant", "void"}
	compHeaderKeywords  = []string{"as", "package", "provides", "uses"}
	compSignKeywords    = []string{"asserts", "emits", "given", "near"}
	compWithKeyword     = []string{"with"}
)

func keywordItems(words []string) []compItem {
	items := make([]compItem, 0, len(words))
	for _, word := range words {
		items = append(items, compItem{label: word, kind: compKeyword, doc: "keyword"})
	}
	return items
}

// ------------------------------------------------------------ walker ---

// compBinder is one function-local binding: inputs, receiver, steps,
// chain and pattern bindings with their declared type and near mark.
// from is the offset the binding becomes visible at: the name start
// for parameters and patterns, the step end for step and chain
// bindings, whose initializers never see the name they define.
type compBinder struct {
	name string
	span source.Span
	from int
	typ  syntax.TypeNode
	near bool
	fn   string
}

// compScope is one lexical frame with the source span it covers and
// its nesting depth.
type compScope struct {
	span    source.Span
	depth   int
	binders []*compBinder
}

type compWithPin struct {
	span   source.Span
	callee syntax.Expr
}

type compCallee struct {
	span      source.Span
	reference bool
}

type compMember struct {
	span     source.Span
	receiver syntax.Expr
	method   bool
}

type compQualified struct {
	span  source.Span
	name  syntax.QualifiedName
	usage compileresolve.Usage
}

type compRefExpr struct {
	span      source.Span
	calleeEnd int
	firstBind int
}

type compFunc struct {
	decl    source.Span
	body    source.Span
	asserts []source.Span
}

// compWalker mirrors the G03 reference walk for one file, recording
// scope spans, binder metadata, and the context markers behind the
// cursor classification. Traversal order matches refWalker so binding
// identity and visibility stay identical.
type compWalker struct {
	text string
	file *compileresolve.File

	scopes []*compScope
	frames []*compScope
	curFn  string

	types     []source.Span
	bounds    []source.Span
	withPins  []compWithPin
	callees   []compCallee
	members   []compMember
	qualified []compQualified
	refExprs  []compRefExpr
	ctors     []source.Span
	decls     []source.Span
	literals  []source.Span
	funcs     []compFunc
	records   []source.Span
	opaque    []source.Span
	declSpans []source.Span
	leads     []source.Span
	header    source.Span
}

func (w *compWalker) push(span source.Span) {
	frame := &compScope{span: span, depth: len(w.scopes)}
	w.scopes = append(w.scopes, frame)
	// Frames outlive the walk: queries resolve visibility from the
	// retained spans, since push and pop mirror AST nesting exactly.
	w.frames = append(w.frames, frame)
}

func (w *compWalker) pop() {
	w.scopes = w.scopes[:len(w.scopes)-1]
}

func (w *compWalker) bind(name syntax.Token, typ syntax.TypeNode, near bool, from int) {
	w.scopes[len(w.scopes)-1].binders = append(w.scopes[len(w.scopes)-1].binders, &compBinder{name: name.Text, span: name.Span, from: from, typ: typ, near: near, fn: w.curFn})
	w.decls = append(w.decls, name.Span)
}

// framesAt returns the frames enclosing the offset, innermost first.
func (w *compWalker) framesAt(offset int) []*compScope {
	var out []*compScope
	for _, frame := range w.frames {
		if coversEnd(frame.span, offset) {
			out = append(out, frame)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].depth > out[j].depth })
	return out
}

func (w *compWalker) lookup(name string, offset int) *compBinder {
	for _, frame := range w.framesAt(offset) {
		var best *compBinder
		for _, binder := range frame.binders {
			if binder.name != name || binder.from >= offset {
				continue
			}
			best = binder
		}
		if best != nil {
			return best
		}
	}
	return nil
}

func (w *compWalker) shadowsLocal(name syntax.QualifiedName, offset int) bool {
	return name.Package == "" && w.lookup(name.Name, offset) != nil
}

// occurrence records one name token with the usage its position
// carries. Only qualified spellings feed the query; unqualified names
// resolve through scope visibility and the World instead.
func (w *compWalker) occurrence(name syntax.QualifiedName, usage compileresolve.Usage) {
	w.qualified = append(w.qualified, compQualified{span: name.Span, name: name, usage: usage})
}

func (w *compWalker) walkFile(file *syntax.File) {
	w.header = file.Header.Span
	for _, declaration := range file.Declarations {
		w.declSpans = append(w.declSpans, declaration.DeclSpan())
		w.walkDecl(declaration)
	}
}

func (w *compWalker) walkDecl(declaration syntax.Declaration) {
	switch node := declaration.(type) {
	case *syntax.FunctionDecl:
		w.decls = append(w.decls, node.Name.Span)
		w.leads = append(w.leads, source.Span{Start: node.DeclSpan().Start, End: node.Name.Span.End})
		for _, parameter := range node.Parameters {
			w.decls = append(w.decls, parameter.Span)
		}
		w.walkType(node.Result)
		if node.Receiver != nil {
			w.walkType(node.Receiver.Type)
		}
		w.walkBound(node.Errors)
		for i := range node.Inputs {
			w.walkType(node.Inputs[i].Type)
		}
		fn := compFunc{decl: node.DeclSpan(), body: node.Body.Span}
		for i := range node.Assertions {
			fn.asserts = append(fn.asserts, node.Assertions[i].Span)
		}
		w.funcs = append(w.funcs, fn)
		outer := w.curFn
		w.curFn = node.Name.Text
		w.push(node.DeclSpan())
		if node.Receiver != nil {
			w.bind(node.Receiver.Name, node.Receiver.Type, false, node.Receiver.Name.Span.Start)
		}
		for i := range node.Inputs {
			w.bind(node.Inputs[i].Name, node.Inputs[i].Type, node.Inputs[i].Near, node.Inputs[i].Name.Span.Start)
		}
		w.walkBlock(node.Body)
		for i := range node.Assertions {
			w.walkAssertion(&node.Assertions[i])
		}
		w.pop()
		w.curFn = outer
	case *syntax.RecordDecl:
		w.decls = append(w.decls, node.Name.Span)
		w.leads = append(w.leads, source.Span{Start: node.DeclSpan().Start, End: node.Name.Span.End})
		for _, parameter := range node.Parameters {
			w.decls = append(w.decls, parameter.Span)
		}
		w.records = append(w.records, node.DeclSpan())
		for i := range node.Fields {
			w.decls = append(w.decls, node.Fields[i].Name.Span)
			w.walkType(node.Fields[i].Type)
		}
	case *syntax.VariantDecl:
		w.decls = append(w.decls, node.Name.Span)
		w.leads = append(w.leads, source.Span{Start: node.DeclSpan().Start, End: node.Name.Span.End})
		for _, parameter := range node.Parameters {
			w.decls = append(w.decls, parameter.Span)
		}
		w.records = append(w.records, node.DeclSpan())
		for _, alternative := range node.Alternatives {
			w.walkType(alternative)
		}
	case *syntax.ErrorDecl:
		w.decls = append(w.decls, node.Name.Span)
		w.leads = append(w.leads, source.Span{Start: node.DeclSpan().Start, End: node.Name.Span.End})
		for _, parameter := range node.Parameters {
			w.decls = append(w.decls, parameter.Span)
		}
		w.records = append(w.records, node.DeclSpan())
		for i := range node.Fields {
			w.decls = append(w.decls, node.Fields[i].Name.Span)
			w.walkType(node.Fields[i].Type)
		}
	case *syntax.ValueDecl:
		w.decls = append(w.decls, node.Binding.Name.Span)
		w.walkType(node.Binding.Type)
		w.walkExpr(node.Binding.Value)
	case *syntax.QuestionDecl:
		if node.RecordName != nil {
			w.decls = append(w.decls, node.RecordName.Span)
		}
		w.decls = append(w.decls, node.Name.Span)
		for _, option := range node.Options {
			if option.Name != nil {
				w.decls = append(w.decls, option.Name.Span)
			}
			w.walkExpr(option.Description)
			w.walkExpr(option.Spread)
			w.walkBody(option.Body)
		}
		w.walkExpr(node.Asks)
		w.walkExpr(node.Minimum)
		w.walkBody(node.Fallback)
		w.walkBody(node.Shared)
	default:
		w.opaque = append(w.opaque, node.DeclSpan())
	}
}

func (w *compWalker) walkType(node syntax.TypeNode) {
	w.walkTypeAt(node, compileresolve.TypeUse)
}

// walkTypeAt records one annotation with the usage its name position
// carries: error usage inside emits bounds, type usage elsewhere. Only
// qualified spellings consult the usage; the markers stay positional.
func (w *compWalker) walkTypeAt(node syntax.TypeNode, usage compileresolve.Usage) {
	if node == nil {
		return
	}
	w.types = append(w.types, node.TypeSpan())
	switch node := node.(type) {
	case *syntax.NamedType:
		for _, argument := range node.Arguments {
			w.walkType(argument)
		}
		w.occurrence(node.Name, usage)
	case *syntax.ArrayType:
		w.walkTypeAt(node.Element, usage)
	case *syntax.CallableType:
		w.walkType(node.Result)
		for _, input := range node.Inputs {
			w.walkType(input)
		}
		w.walkBound(node.Errors)
	case *syntax.ChoiceArmType:
		w.walkType(node.Result)
		w.walkBound(node.Errors)
	}
}

func (w *compWalker) walkBound(bound syntax.ErrorBound) {
	w.bounds = append(w.bounds, bound.Span)
	for _, typ := range bound.Types {
		w.walkTypeAt(typ, compileresolve.ErrorUse)
	}
}

func (w *compWalker) walkBlock(block syntax.Block) {
	w.push(block.Span)
	for _, step := range block.Steps {
		w.walkStep(step)
	}
	w.walkBody(block.Terminal)
	w.pop()
}

func (w *compWalker) walkStep(step syntax.Step) {
	switch node := step.(type) {
	case *syntax.BindingStep:
		w.walkType(node.Binding.Type)
		// The value walks before its own name binds: an initializer
		// never sees the binding it defines.
		w.walkExpr(node.Binding.Value)
		w.bind(node.Binding.Name, node.Binding.Type, false, node.Binding.Span.End)
	case *syntax.CallStep:
		w.walkExpr(node.Call)
	case *syntax.CoordinationStep:
		w.walkCoordination(&node.Coordination)
	}
}

func (w *compWalker) walkBody(body syntax.Body) {
	switch node := body.(type) {
	case *syntax.ValueBody:
		w.walkExpr(node.Value)
	case *syntax.SuccessBody:
		w.walkExpr(node.Value)
	case *syntax.FailureBody:
		if node.Error != nil {
			w.walkExpr(node.Error)
		}
	case *syntax.RelayBody:
		if node.Call != nil {
			w.walkExpr(node.Call)
		}
	case *syntax.DoBody:
		w.walkBlock(node.Block)
	case *syntax.MatchBody:
		w.walkMatch(&node.Match)
	}
}

func (w *compWalker) walkMatch(match *syntax.Match) {
	for _, value := range match.Values {
		w.walkExpr(value)
	}
	if match.Call != nil {
		w.walkExpr(match.Call)
	}
	w.push(match.Span)
	for i := range match.Chain {
		entry := &match.Chain[i]
		if entry.Call != nil {
			w.walkExpr(entry.Call)
		}
		if entry.Binding != nil {
			w.bind(entry.Binding.Name, entry.Binding.Type, false, entry.Span.End)
		}
	}
	for i := range match.When {
		w.walkAssertion(&match.When[i])
	}
	for i := range match.Arms {
		w.walkArm(&match.Arms[i])
	}
	w.pop()
}

func (w *compWalker) walkArm(arm *syntax.MatchArm) {
	w.push(arm.Span)
	for _, pattern := range arm.Patterns {
		w.walkPattern(pattern)
	}
	if arm.Outcome != nil {
		if arm.Outcome.Error != nil {
			w.walkTypeAt(arm.Outcome.Error, compileresolve.ErrorUse)
		}
		if arm.Outcome.Binding != nil {
			w.bind(arm.Outcome.Binding.Name, arm.Outcome.Binding.Type, false, arm.Outcome.Binding.Name.Span.Start)
		}
		if arm.Outcome.Alias != nil {
			w.bind(*arm.Outcome.Alias, nil, false, arm.Outcome.Alias.Span.Start)
		}
	}
	w.walkBody(arm.Body)
	w.pop()
}

func (w *compWalker) walkCoordination(coordination *syntax.Coordination) {
	for i := range coordination.Participants {
		participant := &coordination.Participants[i]
		if participant.Call != nil {
			w.walkExpr(participant.Call)
		}
		w.walkExpr(participant.Spread)
		for j := range participant.Arms {
			w.walkArm(&participant.Arms[j])
		}
	}
	for i := range coordination.Arms {
		w.walkArm(&coordination.Arms[i])
	}
}

func (w *compWalker) walkPattern(pattern syntax.PatternNode) {
	switch node := pattern.(type) {
	case *syntax.BindPattern:
		w.bind(node.Name, nil, false, node.Name.Span.Start)
	case *syntax.ConstructorPattern:
		for _, field := range node.Fields {
			w.walkPattern(field)
		}
		for _, typ := range node.Types {
			w.walkType(typ)
		}
		w.ctors = append(w.ctors, node.Name.Span)
		w.occurrence(node.Name, compileresolve.ConstructorUse)
	case *syntax.NamePattern:
		// Bare pattern names test nominal leaves, never values:
		// completing one offers type candidates.
		w.types = append(w.types, node.Name.Span)
		w.occurrence(node.Name, compileresolve.TypeUse)
		for _, typ := range node.Types {
			w.walkType(typ)
		}
	case *syntax.ArrayPattern:
		for _, element := range node.Elements {
			w.walkPattern(element)
		}
		if node.Rest != nil {
			w.bind(*node.Rest, nil, false, node.Rest.Span.Start)
		}
	case *syntax.AlternativePattern:
		for _, alternative := range node.Alternatives {
			w.walkPattern(alternative)
		}
	}
}

func (w *compWalker) walkAssertion(assertion *syntax.Assertion) {
	w.decls = append(w.decls, assertion.Name.Span)
	if assertion.Scenario != nil {
		w.decls = append(w.decls, assertion.Scenario.Span)
	}
	if assertion.Use != nil {
		w.decls = append(w.decls, assertion.Use.Template.Span)
		for i := range assertion.Use.Arguments {
			w.walkArgument(&assertion.Use.Arguments[i])
		}
	}
	for _, link := range assertion.Links {
		w.decls = append(w.decls, link.Span)
	}
	w.walkExpr(assertion.Receiver)
	for i := range assertion.Arguments {
		w.walkArgument(&assertion.Arguments[i])
	}
	w.walkBody(assertion.Expected)
}

func (w *compWalker) walkArgument(argument *syntax.Argument) {
	w.walkExpr(argument.Value)
	if argument.Group != nil {
		for _, value := range argument.Group.Values {
			w.walkExpr(value)
		}
	}
}

func (w *compWalker) walkExpr(expr syntax.Expr) {
	switch node := expr.(type) {
	case *syntax.NameExpr:
		w.occurrence(node.Name, compileresolve.ValueUse)
	case *syntax.LiteralExpr:
		// Only data literals decline: keyword literals (true/false)
		// stay completable through the general keyword set.
		switch node.Token.Kind {
		case syntax.String, syntax.Integer, syntax.Float:
			w.literals = append(w.literals, node.Token.Span)
		}
	case *syntax.ConstructorExpr:
		for _, typ := range node.Types {
			w.walkType(typ)
		}
		for i := range node.Arguments {
			w.walkArgument(&node.Arguments[i])
		}
		w.ctors = append(w.ctors, node.Name.Span)
		w.occurrence(node.Name, compileresolve.ConstructorUse)
	case *syntax.CallExpr:
		w.walkCallee(node.Invocation.Callee, false)
		for _, typ := range node.Invocation.Types {
			w.walkType(typ)
		}
		for i := range node.Invocation.Arguments {
			w.walkArgument(&node.Invocation.Arguments[i])
		}
		for i := range node.Methods {
			method := &node.Methods[i]
			w.members = append(w.members, compMember{span: method.Name.Span, receiver: node.Invocation.Callee, method: true})
			for _, typ := range method.Types {
				w.walkType(typ)
			}
			for j := range method.Arguments {
				w.walkArgument(&method.Arguments[j])
			}
		}
	case *syntax.ReferenceExpr:
		w.walkReference(node)
	case *syntax.FieldExpr:
		w.members = append(w.members, compMember{span: node.Field.Span, receiver: node.Receiver})
		w.walkExpr(node.Receiver)
	case *syntax.GroupExpr:
		w.walkExpr(node.Value)
	case *syntax.UnaryExpr:
		w.walkExpr(node.Operand)
	case *syntax.BinaryExpr:
		w.walkExpr(node.Left)
		w.walkExpr(node.Right)
	case *syntax.ComparisonExpr:
		for _, operand := range node.Operands {
			w.walkExpr(operand)
		}
	case *syntax.ArrayExpr:
		for i := range node.Elements {
			w.walkArgument(&node.Elements[i])
		}
	case *syntax.IndexExpr:
		w.walkExpr(node.Receiver)
		w.walkExpr(node.Index)
	case *syntax.SliceExpr:
		w.walkExpr(node.Receiver)
		w.walkExpr(node.Start)
		w.walkExpr(node.End)
	case *syntax.UpdateExpr:
		for i := range node.Fields {
			w.members = append(w.members, compMember{span: node.Fields[i].Name.Span, receiver: node.Receiver})
			w.walkExpr(node.Fields[i].Value)
		}
		w.walkExpr(node.Receiver)
	case *syntax.MatchExpr:
		w.walkMatch(&node.Match)
	case *syntax.CoordinationExpr:
		w.walkCoordination(&node.Coordination)
	}
}

// walkCallee records a bare-name callee with the usage its call form
// carries: reference usage for callable references, call usage for
// calls. Complex callees walk as ordinary expressions.
func (w *compWalker) walkCallee(callee syntax.Expr, reference bool) {
	name, ok := callee.(*syntax.NameExpr)
	if !ok {
		w.walkExpr(callee)
		return
	}
	usage := compileresolve.CallUse
	if reference {
		usage = compileresolve.ReferenceUse
	}
	w.callees = append(w.callees, compCallee{span: name.Name.Span, reference: reference})
	w.occurrence(name.Name, usage)
}

func (w *compWalker) walkReference(node *syntax.ReferenceExpr) {
	marker := compRefExpr{span: node.ExprSpan(), firstBind: -1}
	if callee, ok := node.Callee.(*syntax.NameExpr); ok {
		marker.calleeEnd = callee.Name.Span.End
	} else {
		marker.calleeEnd = node.Callee.ExprSpan().End
	}
	if len(node.Bindings) > 0 {
		marker.firstBind = node.Bindings[0].Span.Start
	}
	w.refExprs = append(w.refExprs, marker)
	w.walkCallee(node.Callee, true)
	for _, typ := range node.Types {
		w.walkType(typ)
	}
	for i := range node.Bindings {
		binding := &node.Bindings[i]
		w.withPins = append(w.withPins, compWithPin{span: binding.Name.Span, callee: node.Callee})
		w.walkExpr(binding.Value)
	}
}

// ------------------------------------------------------ classification ---

type compCtxKind int

const (
	ctxEmpty compCtxKind = iota
	ctxQualified
	ctxType
	ctxErrorType
	ctxWithPin
	ctxMember
	ctxCallee
	ctxCtor
	ctxWithKw
	ctxHeader
	ctxSignature
	ctxTopLevel
	ctxGeneral
)

// compContext is the classified cursor: the context kind plus the
// payload its candidates need.
type compContext struct {
	kind      compCtxKind
	usage     compileresolve.Usage
	pkg       string
	pkgPos    bool
	reference bool
	callee    syntax.Expr
	receiver  syntax.Expr
	method    bool
}

// coversEnd reports whether the offset sits inside the span or exactly
// at its end, where completion cursors rest after a partial token.
func coversEnd(span source.Span, offset int) bool {
	return span.Start <= offset && offset <= span.End
}

func coversAny(spans []source.Span, offset int) bool {
	for _, span := range spans {
		if coversEnd(span, offset) {
			return true
		}
	}
	return false
}

// sameLineGap reports whether the source between two offsets holds
// only same-line whitespace.
func (w *compWalker) sameLineGap(from, to int) bool {
	if from < 0 || to > len(w.text) || from > to {
		return false
	}
	for i := from; i < to; i++ {
		if w.text[i] != ' ' && w.text[i] != '\t' {
			return false
		}
	}
	return true
}

// classify resolves the cursor to its innermost context. Declaration
// sites, literals, comments, and uncovered native declarations yield
// no candidates; strict name positions (qualified, type, with-pin,
// member, callee, constructor) take precedence over the looser
// declaration gaps and the general body context.
func (w *compWalker) classify(file *syntax.File, offset int) compContext {
	for _, comment := range file.Comments {
		if coversEnd(comment.Span, offset) {
			return compContext{kind: ctxEmpty}
		}
	}
	if coversAny(w.decls, offset) || coversAny(w.literals, offset) || coversAny(w.opaque, offset) {
		return compContext{kind: ctxEmpty}
	}
	if found, marker := smallestQualified(w.qualified, offset); found {
		return w.qualifiedContext(marker, offset)
	}
	// Bounds precede types: positions inside an emits clause —
	// members or gaps — take error candidates, never the general
	// type set.
	if coversAny(w.bounds, offset) {
		return compContext{kind: ctxErrorType}
	}
	if coversAny(w.types, offset) {
		return compContext{kind: ctxType}
	}
	for _, pin := range w.withPins {
		if coversEnd(pin.span, offset) {
			return compContext{kind: ctxWithPin, callee: pin.callee}
		}
	}
	for _, member := range w.members {
		if coversEnd(member.span, offset) {
			return compContext{kind: ctxMember, receiver: member.receiver, method: member.method}
		}
	}
	for _, callee := range w.callees {
		if coversEnd(callee.span, offset) {
			return compContext{kind: ctxCallee, reference: callee.reference}
		}
	}
	if coversAny(w.ctors, offset) {
		return compContext{kind: ctxCtor}
	}
	for _, ref := range w.refExprs {
		if coversEnd(ref.span, offset) && offset > ref.calleeEnd && (ref.firstBind < 0 || offset < ref.firstBind) {
			return compContext{kind: ctxWithKw}
		}
		// A bare callee's span ends at its name, but `with` still
		// follows across same-line whitespace.
		if ref.firstBind < 0 && offset > ref.calleeEnd && w.sameLineGap(ref.span.End, offset) {
			return compContext{kind: ctxWithKw}
		}
	}
	if coversEnd(w.header, offset) {
		return compContext{kind: ctxHeader}
	}
	// Declaration leads (the starter keywords before a declared
	// name) complete like top level: the name itself already declined
	// above, and annotation positions resolved earlier.
	if coversAny(w.leads, offset) {
		return compContext{kind: ctxTopLevel}
	}
	for _, fn := range w.funcs {
		if !coversEnd(fn.decl, offset) {
			continue
		}
		if coversEnd(fn.body, offset) {
			return compContext{kind: ctxGeneral}
		}
		for _, assert := range fn.asserts {
			if coversEnd(assert, offset) {
				return compContext{kind: ctxGeneral}
			}
		}
		return compContext{kind: ctxSignature}
	}
	if coversAny(w.records, offset) {
		return compContext{kind: ctxType}
	}
	for _, span := range w.declSpans {
		if coversEnd(span, offset) {
			return compContext{kind: ctxGeneral}
		}
	}
	return compContext{kind: ctxTopLevel}
}

func smallestQualified(markers []compQualified, offset int) (bool, compQualified) {
	var best compQualified
	found := false
	for _, marker := range markers {
		if marker.name.Package == "" || !coversEnd(marker.span, offset) {
			continue
		}
		if !found || marker.span.End-marker.span.Start < best.span.End-best.span.Start {
			best, found = marker, true
		}
	}
	return found, best
}

// qualifiedContext splits a qualified occurrence at its `::`: the
// package part completes visible import aliases, the name part the
// imported package's members under the position's usage.
func (w *compWalker) qualifiedContext(marker compQualified, offset int) compContext {
	pkgEnd := marker.span.Start + len(marker.name.Package)
	if pkgEnd+1 >= len(w.text) || w.text[pkgEnd:pkgEnd+2] != "::" {
		return compContext{kind: ctxEmpty}
	}
	if offset <= pkgEnd {
		return compContext{kind: ctxQualified, pkgPos: true}
	}
	return compContext{kind: ctxQualified, pkg: marker.name.Package, usage: marker.usage}
}

// -------------------------------------------------------- candidates ---

// localsAt collects the binders visible at the offset: frames inside
// out, nearest binder starting before the use within a frame. Each
// name surfaces once, so shadowed bindings never leak beside the
// visible one and later same-spelled bindings never capture the use.
func (w *compWalker) localsAt(offset int) []compItem {
	seen := map[string]bool{}
	var items []compItem
	for _, frame := range w.framesAt(offset) {
		byName := map[string]*compBinder{}
		for _, binder := range frame.binders {
			if binder.from >= offset {
				continue
			}
			byName[binder.name] = binder
		}
		for name, binder := range byName {
			if seen[name] {
				continue
			}
			seen[name] = true
			items = append(items, localItem(binder))
		}
	}
	return items
}

func localNames(locals []compItem) map[string]bool {
	names := map[string]bool{}
	for _, local := range locals {
		names[local.label] = true
	}
	return names
}

func localItem(binder *compBinder) compItem {
	typ := ""
	if binder.typ != nil {
		typ = syntax.FormatType(binder.typ)
	}
	tail, doc := typ, "local binding"
	if binder.near {
		tail = "near " + typ
		doc = "near input"
		if binder.fn != "" {
			doc = "near input of " + binder.fn
		}
	}
	return compItem{label: binder.name, kind: compVariable, tail: tail, doc: doc}
}

// visibleSymbols walks the file scope chain innermost out, keeping the
// first binding per name and applying the context's eligibility
// filter. Results are freshly collected per request.
func visibleSymbols(file *compileresolve.File, accept func(*compileresolve.Symbol) bool) []*compileresolve.Symbol {
	seen := map[string]bool{}
	var out []*compileresolve.Symbol
	for scope := file.Scope; scope != nil; scope = scope.Parent {
		for name, symbol := range scope.Symbols {
			if seen[name] {
				continue
			}
			seen[name] = true
			if accept == nil || accept(symbol) {
				out = append(out, symbol)
			}
		}
	}
	return out
}

// typeSymbolAccept keeps type-eligible symbols except the primitives
// the type keywords already spell: one candidate per spelling.
func typeSymbolAccept(symbol *compileresolve.Symbol) bool {
	if !symbol.Eligible(compileresolve.TypeUse) {
		return false
	}
	switch symbol.Name {
	case "int", "float", "bool", "str", "void":
		return false
	}
	return true
}

func packageProvenance(symbol *compileresolve.Symbol) string {
	if symbol == nil || symbol.Package == nil || symbol.Package.Source == nil {
		return "catalogue"
	}
	return "package " + symbol.Package.Name
}

// symbolItem renders one checked symbol. Callable declarations carry
// their exact arity; catalogue operations without a declaration state
// only kind and provenance rather than a guessed signature.
func symbolItem(symbol *compileresolve.Symbol) compItem {
	doc := packageProvenance(symbol)
	switch symbol.Kind {
	case compileresolve.Function, compileresolve.Fetch, compileresolve.Judge, compileresolve.LLM, compileresolve.Wrapper:
		kind := compFunction
		if symbol.Receiver != nil {
			kind = compMethod
		}
		return compItem{label: symbol.Name, kind: kind, tail: callableTail(symbol), doc: doc}
	case compileresolve.Value, compileresolve.ChoiceArm:
		return compItem{label: symbol.Name, kind: compConstant, tail: valueTail(symbol), doc: doc}
	case compileresolve.Record, compileresolve.Variant, compileresolve.Error, compileresolve.Opaque, compileresolve.Primitive:
		return compItem{label: symbol.Name, kind: compClass, tail: string(symbol.Kind), doc: doc}
	case compileresolve.TypeParameter:
		return compItem{label: symbol.Name, kind: compTypeParameter, doc: doc}
	default:
		return compItem{label: symbol.Name, kind: compStruct, tail: string(symbol.Kind), doc: doc}
	}
}

// callableTail renders the exact checked arity of a callable
// declaration: named inputs with near and variadic marks plus the
// result type. Anything without a signature-shaped declaration falls
// back to its kind label instead of a guessed arity.
func callableTail(symbol *compileresolve.Symbol) string {
	switch declaration := symbol.Declaration.(type) {
	case *syntax.FunctionDecl:
		if tail, ok := functionArity(declaration.Result, declaration.Inputs); ok {
			return tail
		}
	case *syntax.FetchDecl:
		if tail, ok := functionArity(declaration.Result, declaration.Inputs); ok {
			return tail
		}
	case *syntax.LLMDecl:
		if tail, ok := functionArity(declaration.Result, declaration.Inputs); ok {
			return tail
		}
	case *syntax.JudgeDecl:
		if tail, ok := functionArity(declaration.Result, declaration.Inputs); ok {
			return tail
		}
	}
	return string(symbol.Kind)
}

func functionArity(result syntax.TypeNode, inputs []syntax.Input) (string, bool) {
	if result == nil {
		return "", false
	}
	params := make([]string, len(inputs))
	for i := range inputs {
		if inputs[i].Type == nil {
			return "", false
		}
		param := ""
		if inputs[i].Near {
			param = "near "
		}
		param += inputs[i].Name.Text + ": " + syntax.FormatType(inputs[i].Type)
		if inputs[i].Variadic {
			param += "..."
		}
		params[i] = param
	}
	joined := ""
	for i, param := range params {
		if i > 0 {
			joined += ", "
		}
		joined += param
	}
	return "(" + joined + ") -> " + syntax.FormatType(result), true
}

// valueTail renders a module value's declared type, or the callable
// arity for callable-typed values.
func valueTail(symbol *compileresolve.Symbol) string {
	if symbol.Type == nil {
		return ""
	}
	if callable, ok := symbol.Type.(*syntax.CallableType); ok {
		inputs := make([]string, len(callable.Inputs))
		for i, input := range callable.Inputs {
			if input == nil {
				return syntax.FormatType(symbol.Type)
			}
			inputs[i] = syntax.FormatType(input)
		}
		if callable.Result == nil {
			return syntax.FormatType(symbol.Type)
		}
		joined := ""
		for i, input := range inputs {
			if i > 0 {
				joined += ", "
			}
			joined += input
		}
		return "(" + joined + ") -> " + syntax.FormatType(callable.Result)
	}
	return syntax.FormatType(symbol.Type)
}

// candidates builds the exact candidate set for a classified context.
func (w *compWalker) candidates(context compContext, offset int) []compItem {
	var items []compItem
	switch context.kind {
	case ctxEmpty:
		return []compItem{}
	case ctxQualified:
		if context.pkgPos {
			return packageAliasItems(w.file)
		}
		return w.packageMemberItems(context.pkg, context.usage)
	case ctxType:
		for _, symbol := range visibleSymbols(w.file, typeSymbolAccept) {
			items = append(items, symbolItem(symbol))
		}
		items = append(items, keywordItems(compTypeKeywords)...)
	case ctxErrorType:
		for _, symbol := range visibleSymbols(w.file, func(symbol *compileresolve.Symbol) bool { return symbol.Eligible(compileresolve.ErrorUse) }) {
			items = append(items, symbolItem(symbol))
		}
	case ctxWithPin:
		return w.withPinItems(context.callee)
	case ctxMember:
		return w.memberItems(context.receiver, context.method)
	case ctxCallee:
		usage := compileresolve.CallUse
		if context.reference {
			usage = compileresolve.ReferenceUse
		}
		shadowed := localNames(w.localsAt(offset))
		for _, symbol := range visibleSymbols(w.file, func(symbol *compileresolve.Symbol) bool { return symbol.Eligible(usage) && !shadowed[symbol.Name] }) {
			items = append(items, symbolItem(symbol))
		}
		if !context.reference {
			items = append(items, w.callableLocals(offset)...)
		}
	case ctxCtor:
		for _, symbol := range visibleSymbols(w.file, func(symbol *compileresolve.Symbol) bool { return symbol.Eligible(compileresolve.ConstructorUse) }) {
			items = append(items, symbolItem(symbol))
		}
	case ctxWithKw:
		return keywordItems(compWithKeyword)
	case ctxHeader:
		items = append(items, keywordItems(compHeaderKeywords)...)
		items = append(items, w.fileDeclItems()...)
		items = append(items, packageAliasItems(w.file)...)
	case ctxSignature:
		return keywordItems(compSignKeywords)
	case ctxTopLevel:
		for _, symbol := range visibleSymbols(w.file, typeSymbolAccept) {
			items = append(items, symbolItem(symbol))
		}
		items = append(items, keywordItems(compTopKeywords)...)
	case ctxGeneral:
		locals := w.localsAt(offset)
		items = append(items, locals...)
		// A visible local shadows the same-spelled symbol for
		// unqualified uses, so the unreachable symbol stays out.
		shadowed := localNames(locals)
		for _, symbol := range visibleSymbols(w.file, func(symbol *compileresolve.Symbol) bool { return !shadowed[symbol.Name] }) {
			items = append(items, symbolItem(symbol))
		}
		items = append(items, keywordItems(compGeneralKeywords)...)
	}
	sortCompItems(items)
	return items
}

// callableLocals offers function-local bindings with an explicit
// callable annotation as call candidates. The annotation is
// source-evident, never inferred.
func (w *compWalker) callableLocals(offset int) []compItem {
	var items []compItem
	for _, item := range w.localsAt(offset) {
		binder := w.lookup(item.label, offset)
		if binder == nil || binder.typ == nil {
			continue
		}
		callable, ok := binder.typ.(*syntax.CallableType)
		if !ok || callable.Result == nil {
			continue
		}
		inputs := make([]string, len(callable.Inputs))
		complete := true
		for i, input := range callable.Inputs {
			if input == nil {
				complete = false
				break
			}
			inputs[i] = syntax.FormatType(input)
		}
		if !complete {
			continue
		}
		joined := ""
		for i, input := range inputs {
			if i > 0 {
				joined += ", "
			}
			joined += input
		}
		item.tail = "(" + joined + ") -> " + syntax.FormatType(callable.Result)
		items = append(items, item)
	}
	return items
}

// withPinItems offers the resolved callee's near parameter names, one
// identity per parameter. An unresolvable callee yields nothing: the
// checker owns the error. A06 is inactive, so no iteration surface
// expands this set.
func (w *compWalker) withPinItems(callee syntax.Expr) []compItem {
	declaration, symbol, ok := calleeFunction(w.file, w.shadowsLocal, callee)
	if !ok {
		return []compItem{}
	}
	var items []compItem
	for i := range declaration.Inputs {
		input := &declaration.Inputs[i]
		if !input.Near {
			continue
		}
		tail := "near"
		if input.Type != nil {
			tail = "near " + syntax.FormatType(input.Type)
		}
		items = append(items, compItem{label: input.Name.Text, kind: compProperty, tail: tail, doc: "near parameter of " + symbol.Name})
	}
	sortCompItems(items)
	return items
}

// memberItems offers the fields or methods behind an annotation-known
// receiver, sharing the G03 known-receiver set. Unknown receivers —
// locals, unresolvable names, non-nominal annotations — yield nothing
// rather than a guessed member.
func (w *compWalker) memberItems(receiver syntax.Expr, method bool) []compItem {
	record, err := knownReceiverRecord(w.file, w.shadowsLocal, receiver)
	if err != nil {
		return []compItem{}
	}
	decl, ok := record.Declaration.(*syntax.RecordDecl)
	if !ok {
		return []compItem{}
	}
	if method {
		return w.methodItems(record)
	}
	if record.Owner && record.Package != w.file.Package {
		return []compItem{}
	}
	var items []compItem
	for i := range decl.Fields {
		tail := ""
		if decl.Fields[i].Type != nil {
			tail = syntax.FormatType(decl.Fields[i].Type)
		}
		items = append(items, compItem{label: decl.Fields[i].Name.Text, kind: compField, tail: tail, doc: "field of " + record.Name})
	}
	sortCompItems(items)
	return items
}

// methodItems enumerates the package methods owned by a record,
// applying the same export rules as go-to-definition.
func (w *compWalker) methodItems(record *compileresolve.Symbol) []compItem {
	if record.Package == nil || record.Package.Source == nil {
		return []compItem{}
	}
	imported := record.Package == w.file.Package
	if !imported {
		for _, pkg := range w.file.Imports {
			if pkg == record.Package {
				imported = true
			}
		}
	}
	var items []compItem
	for _, symbol := range record.Package.Scope.Symbols {
		if symbol.Receiver != record {
			continue
		}
		if !imported || (!symbol.Public && record.Package != w.file.Package) {
			continue
		}
		items = append(items, compItem{label: symbol.Name, kind: compMethod, tail: callableTail(symbol), doc: "method of " + record.Name})
	}
	sortCompItems(items)
	return items
}

// packageMemberItems offers one imported package's members under the
// position's usage, with the privacy and owner rules of qualified
// lookup. Unimported packages yield nothing.
func (w *compWalker) packageMemberItems(alias string, usage compileresolve.Usage) []compItem {
	pkg := w.file.Imports[alias]
	if pkg == nil {
		return []compItem{}
	}
	var items []compItem
	for _, symbol := range pkg.Scope.Symbols {
		if !symbol.Eligible(usage) {
			continue
		}
		if pkg != w.file.Package && !symbol.Public {
			continue
		}
		if usage == compileresolve.ConstructorUse && symbol.Owner && pkg != w.file.Package {
			continue
		}
		items = append(items, symbolItem(symbol))
	}
	sortCompItems(items)
	return items
}

func packageAliasItems(file *compileresolve.File) []compItem {
	var items []compItem
	for alias, pkg := range file.Imports {
		doc := "package " + pkg.Name
		if pkg.Source == nil {
			doc = "catalogue package " + pkg.Name
		}
		items = append(items, compItem{label: alias, kind: compModule, doc: doc})
	}
	sortCompItems(items)
	return items
}

// fileDeclItems offers the names this file declares, for header
// provides positions.
func (w *compWalker) fileDeclItems() []compItem {
	var items []compItem
	for _, symbol := range w.file.Package.Scope.Symbols {
		if symbol.Source != w.file.Source {
			continue
		}
		items = append(items, symbolItem(symbol))
	}
	sortCompItems(items)
	return items
}

// ------------------------------------------------------------- entry ---

// completion resolves the cursor to its context over the checked
// snapshot and reports the exact candidate set for it. It declines
// wherever the snapshot cannot support a query — unloadable,
// unresolvable, or missing scope data — so unparseable buffers never
// receive guessed candidates. ok distinguishes a decline (null on the
// wire) from an empty candidate list.
func completion(snapshot *driver.Snapshot, file string, line, character int) ([]compItem, bool) {
	if snapshot == nil || snapshot.World == nil || snapshot.Graph == nil {
		return nil, false
	}
	canonical, err := filepath.EvalSymlinks(file)
	if err != nil {
		return nil, false
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
		return nil, false
	}
	resolved, ok := snapshot.World.Files[src]
	if !ok || resolved == nil {
		return nil, false
	}
	text, err := source.New(canonical, string(src.Bytes))
	if err != nil {
		return nil, false
	}
	offset, err := text.Offset(source.UTF16Position{Line: line, Character: character})
	if err != nil {
		return nil, false
	}
	walker := &compWalker{text: string(src.Bytes), file: resolved}
	walker.walkFile(src.Syntax)
	items := walker.candidates(walker.classify(src.Syntax, offset), offset)
	if items == nil {
		items = []compItem{}
	}
	return items, true
}

// completion answers textDocument/completion over the checked snapshot:
// the exact candidate set for the cursor context, or null where the
// snapshot cannot support the query. Like the other queries it reads
// an inert overlay snapshot and returns freshly built items.
func (s *lspServer) completion(uri string, line, character int) any {
	doc, ok := s.docs[uri]
	if !ok || doc.path == "" {
		return nil
	}
	snapshot, err := driver.CheckSnapshot(discoverRoot(doc.path), doc.path, s.overlay)
	if err != nil || snapshot.World == nil {
		return nil
	}
	items, ok := completion(snapshot, doc.path, line, character)
	if !ok {
		return nil
	}
	out := make([]any, 0, len(items))
	for _, item := range items {
		out = append(out, map[string]any{
			"label":         item.label,
			"kind":          item.kind,
			"detail":        item.tail,
			"documentation": item.doc,
		})
	}
	return out
}
