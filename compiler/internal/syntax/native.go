package syntax

import "github.com/veighnsche/can-lang/compiler/internal/source"

// Native nodes keep configuration and executable expressions distinct. Parsing
// never opens a connection or evaluates a request/descriptor expression.
type ConnectionDecl struct {
	DeclarationLocation
	Name     Token
	Settings []ConnectionSetting
}
type ConnectionSetting struct {
	Span    source.Span
	Name    Token
	Value   Expr
	Entries []ConnectionSetting
}

type NativeHeader struct {
	Result     TypeNode
	Name       Token
	Connection QualifiedName
	Errors     ErrorBound
	Inputs     []Input
}
type NativeEntry struct {
	Span  source.Span
	Name  Token
	Value Expr
}
type FetchDecl struct {
	DeclarationLocation
	NativeHeader
	Assertions     []Assertion
	Method         Token
	Path           Expr
	Query, Headers []NativeEntry
	BodyEncoding   *Token
	Body           Expr
}
type LLMDecl struct {
	DeclarationLocation
	NativeHeader
	Assertions []Assertion
	State      []Field
	Asks       Expr
}

// FixtureDecl is a P3.1 fixture template: inert reusable rows for one exact
// executable target with optional typed parameters. It is test data, not a
// callable or an assertion root, and is erased from production output.
type FixtureDecl struct {
	DeclarationLocation
	Name   Token
	Target QualifiedName
	// Types holds the concrete type arguments for a generic target.
	Types []TypeNode
	Given []Field
	Cases []FixtureCase
}

// FixtureCase is one template row: target inputs in exact call grammar plus
// an expected completion, with an optional raw execution mode.
type FixtureCase struct {
	Span      source.Span
	Arguments []Argument
	Expected  Body
	Mode      *AssertionMode
}

// ScenarioDecl is a DI-06 exported test seam marker. It names a scenario
// whose rows are tagged at lexical when tables and activated by caller
// links. It carries no rows or target; checking validates tags and links,
// and the marker is erased from production output.
type ScenarioDecl struct {
	DeclarationLocation
	Name Token
}

// ActionDecl is a checked handler-free HTTP endpoint contract. It binds one
// POST or GET route to a captures record, one request input line, a finite
// returns variant, a response body mode and an exhaustive leaf-to-status
// case table. Handlers are bound later at action::mount; the declaration
// carries no executable reference. Checking validates the path, capture,
// input, limit, returns, body-mode and case contracts; emission records
// the canonical metadata the server and browser adapters consume.
type ActionDecl struct {
	DeclarationLocation
	Name     Token
	Method   Token
	Path     Token
	Captures TypeNode // captures record; nil when the path declares no captures
	Input    *ActionInput
	Returns  TypeNode
	Response Token // response body mode: json or html
	Cases    []ActionCase
}

// ActionInput is the one request line of an action: `input none` for a
// bodyless GET, `json <wire> limit <bytes>` for a JSON POST, or
// `form <wire> limit <bytes> [rows_limit <n>]` for an HTML POST.
type ActionInput struct {
	Span      source.Span
	Mode      Token
	Type      TypeNode // nil for input none
	Limit     *Token   // nil for input none
	RowsLimit *Token   // form rows bound; nil unless spelled
}

// ActionCase maps one returns-variant leaf to its wire status plus the
// HTML-only visible response mode: `swap inner` for fragments, `document`
// for full-page renders. Both are nil for JSON cases.
type ActionCase struct {
	Span     source.Span
	Leaf     QualifiedName
	Status   Token
	Swap     *Token // the swap keyword; nil unless the case swaps a fragment
	Document *Token // the document keyword; nil unless the case renders a document
}

// WrapDecl is an A3.2 operation wrapper: `from` names exactly one fetch,
// judge or wrapper base; the complete signature, grouped state, result and
// connection are inherited. Handles tables hold origin-specific policy arms.
type WrapDecl struct {
	DeclarationLocation
	Name       Token
	Base       QualifiedName
	Calculated Token
	Assertions []Assertion
	Native     []WrapArm
	Emitted    []WrapArm
	HasNative  bool
	HasEmitted bool
}

// WrapArm is one policy rule: an exact error pattern with optional alias
// plus a terminal body over the inherited result type.
type WrapArm struct {
	Span    source.Span
	Pattern *OutcomePattern
	Body    Body
}

func (p *parser) connection() Declaration {
	start := p.expectWord("connection").Span.Start
	name := p.expect(Name)
	p.expect(Newline)
	p.expect(Indent)
	var settings []ConnectionSetting
	for !p.at(Dedent) && !p.at(EOF) {
		key := p.expect(Name)
		setting := ConnectionSetting{Name: key}
		switch key.Text {
		case "auth":
			p.expectWord("bearer")
			p.expectWord("env")
			setting.Value = p.expression(1)
			p.expect(Newline)
		case "headers", "metadata":
			p.expect(Newline)
			p.expect(Indent)
			for !p.at(Dedent) && !p.at(EOF) {
				field := p.expect(Name)
				if key.Text == "headers" {
					p.expect("=")
				}
				value := p.expression(1)
				p.expect(Newline)
				setting.Entries = append(setting.Entries, ConnectionSetting{Span: p.span(field.Span.Start), Name: field, Value: value})
			}
			p.expect(Dedent)
		default:
			setting.Value = p.expression(1)
			p.expect(Newline)
		}
		setting.Span = p.span(key.Span.Start)
		settings = append(settings, setting)
	}
	p.expect(Dedent)
	return &ConnectionDecl{DeclarationLocation: DeclarationLocation{p.span(start)}, Name: name, Settings: settings}
}
func (p *parser) nativeHeader(kind string) NativeHeader {
	p.expectWord(kind)
	result := p.parseType()
	name := p.expect(Name)
	p.expectWord("from")
	connection := p.qualified()
	p.expect(Newline)
	p.expect(Indent)
	bound := p.errorBound()
	p.expect(Newline)
	return NativeHeader{Result: result, Name: name, Connection: connection, Errors: bound, Inputs: p.nativeInputs(kind == "fetch")}
}
func (p *parser) nativeInputs(allowNear bool) []Input {
	if !p.word("given") {
		return nil
	}
	p.take()
	p.expect(Newline)
	p.expect(Indent)
	var inputs []Input
	for !p.at(Dedent) && !p.at(EOF) {
		start := p.peek().Span.Start
		near := p.word("near")
		if near {
			if !allowNear {
				p.fail("near is not allowed on this native declaration")
			}
			p.take()
		}
		t := p.parseType()
		variadic := p.at("...")
		if variadic {
			p.take()
		}
		name := p.expect(Name)
		if near && variadic {
			p.fail("a near input cannot be variadic")
		}
		inputs = append(inputs, Input{Field: Field{Span: p.span(start), Type: t, Name: name}, Near: near, Variadic: variadic})
		p.expect(Newline)
	}
	p.expect(Dedent)
	return inputs
}
func (p *parser) nativeState() []Field {
	if !p.word("state") {
		return nil
	}
	p.take()
	p.expect(Newline)
	p.expect(Indent)
	fields := []Field{p.field()}
	p.expect(Newline)
	for !p.at(Dedent) && !p.at(EOF) {
		fields = append(fields, p.field())
		p.expect(Newline)
	}
	p.expect(Dedent)
	return fields
}
func (p *parser) nativeEntries(section string) []NativeEntry {
	if !p.word(section) {
		return nil
	}
	p.take()
	p.expect(Newline)
	p.expect(Indent)
	var entries []NativeEntry
	for !p.at(Dedent) && !p.at(EOF) {
		name := p.expect(Name)
		p.expect("=")
		value := p.expression(1)
		p.expect(Newline)
		entries = append(entries, NativeEntry{Span: p.span(name.Span.Start), Name: name, Value: value})
	}
	p.expect(Dedent)
	return entries
}
func (p *parser) nativeAssertions() []Assertion {
	if !p.word("asserts") {
		return nil
	}
	p.take()
	p.expect(Newline)
	p.expect(Indent)
	assertions := []Assertion{p.assertion(false)}
	for !p.at(Dedent) && !p.at(EOF) {
		assertions = append(assertions, p.assertion(false))
	}
	p.expect(Dedent)
	return assertions
}
func (p *parser) fetch() Declaration {
	start := p.peek().Span.Start
	header := p.nativeHeader("fetch")
	assertions := p.nativeAssertions()
	method := p.expect(Name)
	switch method.Text {
	case "get", "head", "post", "put", "patch", "delete", "options":
	default:
		p.fail("unknown fetch method")
	}
	path := p.expression(1)
	p.expect(Newline)
	query := p.nativeEntries("query")
	headers := p.nativeEntries("headers")
	var encoding *Token
	var body Expr
	if p.word("body") {
		p.take()
		e := p.expect(Name)
		switch e.Text {
		case "json", "text", "bytes":
		default:
			p.fail("unknown fetch body encoding")
		}
		encoding = &e
		body = p.expression(1)
		p.expect(Newline)
		if method.Text == "get" || method.Text == "head" {
			p.fail("GET and HEAD cannot have a body")
		}
	}
	p.expect(Dedent)
	return &FetchDecl{DeclarationLocation: DeclarationLocation{p.span(start)}, NativeHeader: header, Assertions: assertions, Method: method, Path: path, Query: query, Headers: headers, BodyEncoding: encoding, Body: body}
}
func (p *parser) llm() Declaration {
	start := p.peek().Span.Start
	header := p.nativeHeader("llm")
	state := p.nativeState()
	assertions := p.nativeAssertions()
	p.expectWord("asks")
	asks := p.expression(1)
	p.expect(Newline)
	p.expect(Dedent)
	return &LLMDecl{DeclarationLocation: DeclarationLocation{p.span(start)}, NativeHeader: header, Assertions: assertions, State: state, Asks: asks}
}
func (p *parser) fixture() Declaration {
	start := p.expectWord("fixture").Span.Start
	name := p.expect(Name)
	p.expectWord("for")
	target := p.qualified()
	types := p.typeArguments()
	p.expect(Newline)
	p.expect(Indent)
	var given []Field
	if p.word("given") {
		p.take()
		p.expect(Newline)
		p.expect(Indent)
		for !p.at(Dedent) && !p.at(EOF) {
			given = append(given, p.field())
			p.expect(Newline)
		}
		p.expect(Dedent)
	}
	p.expectWord("cases")
	p.expect(Newline)
	p.expect(Indent)
	if p.at(Dedent) || p.at(EOF) {
		p.fail("fixture cases must be nonempty")
	}
	cases := []FixtureCase{p.fixtureCase()}
	for !p.at(Dedent) && !p.at(EOF) {
		cases = append(cases, p.fixtureCase())
	}
	p.expect(Dedent)
	p.expect(Dedent)
	return &FixtureDecl{DeclarationLocation: DeclarationLocation{p.span(start)}, Name: name, Target: target, Types: types, Given: given, Cases: cases}
}

func (p *parser) scenario() Declaration {
	start := p.expectWord("scenario").Span.Start
	name := p.expect(Name)
	p.expect(Newline)
	return &ScenarioDecl{DeclarationLocation: DeclarationLocation{p.span(start)}, Name: name}
}

func (p *parser) action() Declaration {
	start := p.expectWord("action").Span.Start
	name := p.expect(Name)
	p.expect(Newline)
	if !p.at(Indent) {
		p.fail("action requires a method and path")
	}
	p.take()
	method := p.expect(Name)
	switch method.Text {
	case "get", "post":
	default:
		p.fail("unknown action method")
	}
	path := p.expect(String)
	p.expect(Newline)
	var captures TypeNode
	if p.word("captures") {
		p.take()
		captures = p.parseType()
		p.expect(Newline)
	}
	input := p.actionInput(method)
	p.rejectStaleActionClause("returns")
	if !p.word("returns") {
		p.fail("action requires a returns clause")
	}
	p.take()
	returns := p.parseType()
	p.expect(Newline)
	p.rejectStaleActionClause("body")
	if !p.word("body") {
		p.fail("action requires a body clause")
	}
	p.take()
	response := p.expect(Name)
	switch response.Text {
	case "json", "html":
	default:
		p.fail("unknown action body mode")
	}
	p.expect(Newline)
	p.rejectStaleActionClause("cases")
	p.expectWord("cases")
	p.expect(Newline)
	p.expect(Indent)
	cases := []ActionCase{p.actionCase()}
	for !p.at(Dedent) && !p.at(EOF) {
		cases = append(cases, p.actionCase())
	}
	p.expect(Dedent)
	p.expect(Dedent)
	return &ActionDecl{DeclarationLocation: DeclarationLocation{p.span(start)}, Name: name, Method: method, Path: path, Captures: captures, Input: input, Returns: returns, Response: response, Cases: cases}
}

// rejectStaleActionClause rejects removed action clauses with a migration
// diagnostic and repeated clauses as duplicates. It runs wherever the next
// fixed clause is expected, so stale spellings fail with a span no matter
// where they appear; expect names the clause due next and is never stale.
func (p *parser) rejectStaleActionClause(expect string) {
	if p.word("handles") {
		p.fail("action handles was removed; bind handlers with action::mount")
	}
	if p.word("result") {
		p.fail("action result was removed; use returns")
	}
	clauses := []string{"captures", "input", "json", "form", "returns", "body"}
	for i, word := range clauses {
		if word == expect {
			clauses = clauses[:i]
			break
		}
	}
	for _, word := range clauses {
		if p.word(word) {
			p.fail("duplicate action clause")
		}
	}
}

func (p *parser) actionInput(method Token) *ActionInput {
	start := p.peek().Span.Start
	if p.word("input") {
		mode := p.take()
		if !p.word("none") {
			p.fail("a bodyless action input is spelled input none")
		}
		p.take()
		p.expect(Newline)
		if method.Text == "post" {
			p.fail("POST actions require a json or form input")
		}
		return &ActionInput{Span: p.span(start), Mode: mode}
	}
	if p.word("json") || p.word("form") {
		mode := p.take()
		wire := p.parseType()
		if !p.word("limit") {
			p.fail("action wire input requires a byte limit")
		}
		p.take()
		limit := p.expect(Integer)
		var rows *Token
		if p.word("rows_limit") {
			if mode.Text != "form" {
				p.fail("rows_limit applies to form input only")
			}
			p.take()
			bound := p.expect(Integer)
			rows = &bound
		}
		p.expect(Newline)
		if method.Text == "get" {
			p.fail("GET actions cannot take a wire input")
		}
		return &ActionInput{Span: p.span(start), Mode: mode, Type: wire, Limit: &limit, RowsLimit: rows}
	}
	if p.word("body") {
		p.fail("action body wire input was removed; use a json or form input line")
	}
	p.fail("action requires an input line")
	return nil
}

func (p *parser) actionCase() ActionCase {
	start := p.peek().Span.Start
	leaf := p.qualified()
	p.expectWord("status")
	status := p.expect(Integer)
	var swap, document *Token
	if p.word("swap") {
		policy := p.take()
		if !p.word("inner") {
			p.fail("unknown action swap policy")
		}
		p.take()
		swap = &policy
	} else if p.word("document") {
		mode := p.take()
		document = &mode
	}
	p.expect(Newline)
	return ActionCase{Span: p.span(start), Leaf: leaf, Status: status, Swap: swap, Document: document}
}

func (p *parser) fixtureCase() FixtureCase {
	start := p.peek().Span.Start
	arguments := p.invocationArguments("=>")
	for _, argument := range arguments {
		if argument.Spread {
			p.fail("fixture case inputs must be explicit positional values")
		}
	}
	p.expect("=>")
	expected := p.completion()
	p.singleLine(start, expected.BodySpan().End)
	p.expect(Newline)
	mode := p.assertionMode()
	return FixtureCase{Span: p.span(start), Arguments: arguments, Expected: expected, Mode: mode}
}

func (p *parser) wrap() Declaration {
	start := p.expectWord("wrap").Span.Start
	name := p.expect(Name)
	p.expectWord("from")
	base := p.qualified()
	p.expect(Newline)
	p.expect(Indent)
	p.expectWord("emits")
	if !p.word("calculated") {
		p.fail("wrap declarations require emits calculated")
	}
	calculated := p.take()
	p.expect(Newline)
	assertions := p.nativeAssertions()
	var native, emitted []WrapArm
	hasNative, hasEmitted := false, false
	for p.word("handles") {
		p.take()
		section := ""
		switch {
		case p.word("native"):
			section = "native"
		case p.word("emitted"):
			section = "emitted"
		default:
			p.fail("handles selects a native or emitted table")
		}
		p.take()
		p.expect(Newline)
		p.expect(Indent)
		arms := []WrapArm{p.wrapArm()}
		for !p.at(Dedent) && !p.at(EOF) {
			arms = append(arms, p.wrapArm())
		}
		p.expect(Dedent)
		switch section {
		case "native":
			if hasNative {
				p.fail("duplicate handles native section")
			}
			if hasEmitted {
				p.fail("handles native precedes handles emitted")
			}
			hasNative, native = true, arms
		default:
			if hasEmitted {
				p.fail("duplicate handles emitted section")
			}
			hasEmitted, emitted = true, arms
		}
	}
	if !hasNative && !hasEmitted {
		p.fail("wrap requires at least one handles section")
	}
	p.expect(Dedent)
	return &WrapDecl{DeclarationLocation: DeclarationLocation{p.span(start)}, Name: name, Base: base, Calculated: calculated, Assertions: assertions, Native: native, Emitted: emitted, HasNative: hasNative, HasEmitted: hasEmitted}
}
func (p *parser) wrapArm() WrapArm {
	start := p.peek().Span.Start
	pattern := p.outcomePattern()
	if pattern.Success || pattern.StandardFailure {
		p.fail("a policy arm matches one exact error, never ok or [_]")
	}
	p.expect("=>")
	body := p.armBody(true)
	return WrapArm{Span: p.span(start), Pattern: &pattern, Body: body}
}

type JudgeDecl struct {
	DeclarationLocation
	NativeHeader
	Assertions    []Assertion
	State         []Field
	Registrations []ChainEntry
	Continuation  Body
}
type ChoiceArmDecl struct {
	DeclarationLocation
	Result      TypeNode
	Name        Token
	Errors      ErrorBound
	Description Expr
	Body        Block
}
type ProbabilityExpr struct{ ExpressionLocation }
type NativeBinder struct {
	Kind Token
	Name Token
}
type QuestionOption struct {
	Span        source.Span
	Name        *Token
	Description Expr
	Spread      Expr
	Body        Body
}
type QuestionDecl struct {
	DeclarationLocation
	NativeHeader
	Kind          string
	RecordName    *Token
	Binders       []NativeBinder
	Asks, Minimum Expr
	Fallback      Body
	Options       []QuestionOption
	Selected      *Field
	Shared        Body
}

func (p *parser) judge() Declaration {
	start := p.peek().Span.Start
	header := p.nativeHeader("judge")
	state := p.nativeState()
	assertions := p.nativeAssertions()
	var registrations []ChainEntry
	for p.word("call") {
		call := p.callExpression().(*CallExpr)
		var binding *Field
		if p.word("as") {
			p.take()
			field := p.field()
			binding = &field
		}
		registrations = append(registrations, ChainEntry{Span: p.span(call.Span.Start), Call: call, Binding: binding})
		p.expect(Newline)
	}
	if len(registrations) == 0 {
		p.fail("judge requires at least one registration")
	}
	p.expectWord("ok")
	p.expect("=>")
	continuation := p.armBody(true)
	p.expect(Dedent)
	return &JudgeDecl{DeclarationLocation: DeclarationLocation{p.span(start)}, NativeHeader: header, Assertions: assertions, State: state, Registrations: registrations, Continuation: continuation}
}
func (p *parser) choiceArm() Declaration {
	start := p.expectWord("choice_arm").Span.Start
	result := p.parseType()
	name := p.expect(Name)
	p.expect(Newline)
	p.expect(Indent)
	bound := p.errorBound()
	p.expect(Newline)
	p.expectWord("describes")
	description := p.expression(1)
	p.expect(Newline)
	p.probabilityDepth++
	body := p.block()
	p.probabilityDepth--
	return &ChoiceArmDecl{DeclarationLocation: DeclarationLocation{p.span(start)}, Result: result, Name: name, Errors: bound, Description: description, Body: body}
}
func (p *parser) nativeHandler(probability bool) Body {
	p.expect("=>")
	if probability {
		p.probabilityDepth++
		defer func() { p.probabilityDepth-- }()
	}
	return p.armBody(true)
}
func (p *parser) question(start int, record *Token) Declaration {
	kind := p.peek().Text
	header := p.nativeHeader(kind)
	q := &QuestionDecl{NativeHeader: header, Kind: kind, RecordName: record}
	binder := func() {
		key := p.take()
		p.expectWord("as")
		name := p.expect(Name)
		p.expect(Newline)
		for _, b := range q.Binders {
			if b.Kind.Text == key.Text {
				p.fail("duplicate native metadata binder")
			}
		}
		q.Binders = append(q.Binders, NativeBinder{Kind: key, Name: name})
	}
	minimum := func(fallback bool) {
		p.expectWord("minimum")
		q.Minimum = p.expression(1)
		if fallback {
			q.Fallback = p.nativeHandler(false)
		} else {
			p.expect(Newline)
		}
	}
	if kind == "score" {
		for p.word("confidence") || p.word("score") {
			binder()
		}
		if p.word("minimum") {
			if record != nil {
				p.fail("record Score has no minimum")
			}
			minimum(true)
		}
	} else if kind == "choice" && p.word("confidence") {
		binder()
	}
	p.expectWord("asks")
	q.Asks = p.expression(1)
	p.expect(Newline)
	switch kind {
	case "noul":
		if p.word("minimum") {
			minimum(false)
		}
	case "choice":
		after := p.word("confidence")
		if after {
			binder()
		}
		if p.word("minimum") {
			if record != nil {
				p.fail("record Choice has no minimum")
			}
			minimum(true)
		} else if after {
			p.fail("confidence after asks requires minimum")
		}
	}
	p.expect(Indent)
	for !p.at(Dedent) && !p.at(EOF) {
		start := p.peek().Span.Start
		if p.word("ok") {
			p.take()
			if kind == "choice" {
				field := p.field()
				q.Selected = &field
			} else if kind != "score" || record != nil {
				p.fail("shared native handler is not allowed")
			}
			q.Shared = p.nativeHandler(kind == "choice")
			if !p.at(Dedent) {
				p.fail("shared handler must end the option block")
			}
			break
		}
		option := QuestionOption{}
		if p.at("...") {
			if kind != "choice" {
				p.fail("only Choice admits option spreads")
			}
			p.take()
			option.Spread = p.expression(1)
			p.expect(Newline)
		} else {
			var name Token
			if kind == "noul" {
				if !p.word("true") && !p.word("false") {
					p.fail("Noul requires true and false handlers")
				}
				name = p.take()
			} else {
				name = p.expect(Name)
			}
			option.Name = &name
			option.Description = p.expression(1)
			if kind == "score" && record == nil {
				p.expect(Newline)
			} else {
				option.Body = p.nativeHandler(true)
			}
		}
		option.Span = p.span(start)
		q.Options = append(q.Options, option)
	}
	p.expect(Dedent)
	p.expect(Dedent)
	if len(q.Options) == 0 {
		p.fail("native question requires options or levels")
	}
	if kind == "noul" {
		if len(q.Options) != 2 || q.Options[0].Name.Text == q.Options[1].Name.Text {
			p.fail("Noul requires exactly one true and one false handler")
		}
	}
	if kind == "score" && record == nil && q.Shared == nil {
		p.fail("ordinary Score requires its shared ok handler")
	}
	if kind == "choice" && q.Shared != nil {
		if record != nil {
			p.fail("record Choice cannot use dynamic descriptions")
		}
		for _, option := range q.Options {
			if option.Spread == nil {
				p.fail("dynamic Choice cannot mix inline handlers")
			}
		}
	}
	q.DeclarationLocation = DeclarationLocation{p.span(start)}
	return q
}

// NativeSignature exposes common syntax evidence without coercing a native
// declaration into a function or granting ordinary function-call eligibility.
func NativeSignature(declaration Declaration) *NativeHeader {
	switch d := declaration.(type) {
	case *FetchDecl:
		return &d.NativeHeader
	case *LLMDecl:
		return &d.NativeHeader
	case *JudgeDecl:
		return &d.NativeHeader
	case *QuestionDecl:
		return &d.NativeHeader
	default:
		return nil
	}
}
