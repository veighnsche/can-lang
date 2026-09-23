package syntax

import "github.com/veighnsche/can-lang/compiler/internal/source"

type ParseResult struct {
	File        *File
	Diagnostics []Diagnostic
}

func (r ParseResult) OK() bool { return len(r.Diagnostics) == 0 }

// Parse is the current-language grammar entry point. Its only input is inert
// source text; it has no loader, runtime, environment or provider dependency.
func Parse(file *source.File) (result ParseResult) {
	lexed := Lex(file)
	if !lexed.OK() {
		return ParseResult{Diagnostics: lexed.Diagnostics}
	}
	p := newParser(lexed)
	defer func() {
		if caught := recover(); caught != nil {
			if failure, ok := caught.(parseFailure); ok {
				result = ParseResult{Diagnostics: []Diagnostic{failure.diagnostic}}
			} else {
				panic(caught)
			}
		}
	}()
	parsed := &File{Source: file, Comments: lexed.Comments, Header: p.packageHeader()}
	for !p.at(EOF) {
		parsed.Declarations = append(parsed.Declarations, p.declaration())
	}
	return ParseResult{File: parsed}
}

func (p *parser) nameList(end Kind) []Token {
	var names []Token
	if p.at(end) {
		return names
	}
	for {
		names = append(names, p.expect(Name))
		if !p.at(",") {
			break
		}
		p.take()
	}
	return names
}

func (p *parser) packageHeader() PackageHeader {
	start := p.expectWord("package").Span.Start
	name := p.expect(Name)
	p.expect(Newline)
	p.expect(Indent)
	p.expectWord("provides")
	p.expect("[")
	provides := p.nameList("]")
	p.expect("]")
	p.expect(Newline)
	p.expectWord("uses")
	p.expect("[")
	var uses []Import
	if !p.at("]") {
		for {
			name := p.expect(Name)
			var alias *Token
			if p.word("as") {
				p.take()
				t := p.expect(Name)
				alias = &t
			}
			uses = append(uses, Import{Span: p.span(name.Span.Start), Package: name, Alias: alias})
			if !p.at(",") {
				break
			}
			p.take()
		}
	}
	p.expect("]")
	p.expect(Newline)
	p.expect(Dedent)
	return PackageHeader{Span: p.span(start), Name: name, Provides: provides, Uses: uses}
}

func (p *parser) parameters() []Token {
	if !p.at("<") {
		return nil
	}
	p.take()
	names := []Token{p.expect(Name)}
	for p.at(",") {
		p.take()
		names = append(names, p.expect(Name))
	}
	p.closeAngle()
	return names
}

func (p *parser) field() Field {
	start := p.peek().Span.Start
	typeNode := p.parseType()
	name := p.expect(Name)
	return Field{Span: p.span(start), Type: typeNode, Name: name}
}

func (p *parser) declaration() Declaration {
	start := p.peek().Span.Start
	switch {
	case p.word("judge"):
		return p.judge()
	case p.word("choice_arm") && p.index+1 < len(p.tokens) && p.tokens[p.index+1].Kind != "<":
		return p.choiceArm()
	case p.word("noul") || p.word("choice") || p.word("score"):
		return p.question(start, nil)
	case p.word("connection"):
		return p.connection()
	case p.word("fetch"):
		return p.fetch()
	case p.word("llm"):
		return p.llm()
	case p.word("wrap"):
		return p.wrap()
	case p.word("fixture"):
		return p.fixture()
	case p.word("fn"):
		return p.function()
	case p.word("record"):
		p.take()
		name := p.expect(Name)
		parameters := p.parameters()
		if p.word("choice") || p.word("score") {
			if len(parameters) > 0 {
				p.fail("native generated records cannot declare type parameters")
			}
			return p.question(start, &name)
		}
		p.expect(Newline)
		var fields []Field
		if p.at(Indent) {
			p.take()
			for !p.at(Dedent) && !p.at(EOF) {
				fields = append(fields, p.field())
				p.expect(Newline)
			}
			p.expect(Dedent)
		}
		return &RecordDecl{DeclarationLocation: DeclarationLocation{p.span(start)}, Name: name, Parameters: parameters, Fields: fields}
	case p.word("variant"):
		p.take()
		name := p.expect(Name)
		parameters := p.parameters()
		p.expect(Newline)
		p.expect(Indent)
		alternatives := []TypeNode{p.parseType()}
		p.expect(Newline)
		for !p.at(Dedent) && !p.at(EOF) {
			alternatives = append(alternatives, p.parseType())
			p.expect(Newline)
		}
		p.expect(Dedent)
		return &VariantDecl{DeclarationLocation: DeclarationLocation{p.span(start)}, Name: name, Parameters: parameters, Alternatives: alternatives}
	case p.word("error"):
		p.take()
		id := p.expect(Integer)
		name := p.expect(Name)
		parameters := p.parameters()
		p.expect("(")
		var fields []Field
		if !p.at(")") {
			for {
				fields = append(fields, p.field())
				if !p.at(",") {
					break
				}
				p.take()
			}
		}
		p.expect(")")
		p.expect(Newline)
		return &ErrorDecl{DeclarationLocation: DeclarationLocation{p.span(start)}, ID: id, Name: name, Parameters: parameters, Fields: fields}
	default:
		binding := p.binding()
		return &ValueDecl{DeclarationLocation: DeclarationLocation{binding.Span}, Binding: binding}
	}
}

func (p *parser) binding() Binding {
	field := p.field()
	p.expect("=")
	value := p.valueExpression()
	switch value.(type) {
	case *MatchExpr, *CoordinationExpr:
	default:
		p.expect(Newline)
	}
	return Binding{Span: p.span(field.Span.Start), Type: field.Type, Name: field.Name, Value: value}
}

func (p *parser) function() Declaration {
	start := p.expectWord("fn").Span.Start
	result := p.parseType()
	name := p.expect(Name)
	parameters := p.parameters()
	p.expect(Newline)
	p.expect(Indent)
	var receiver *Field
	if p.word("on") {
		p.take()
		field := p.field()
		receiver = &field
		p.expect(Newline)
	}
	bound := p.errorBound()
	p.expect(Newline)
	var inputs []Input
	if p.word("given") {
		p.take()
		p.expect(Newline)
		p.expect(Indent)
		for !p.at(Dedent) && !p.at(EOF) {
			start := p.peek().Span.Start
			near := p.word("near")
			if near {
				p.take()
			}
			typeNode := p.parseType()
			variadic := p.at("...")
			if variadic {
				p.take()
			}
			name := p.expect(Name)
			if near && variadic {
				p.fail("a near input cannot be variadic")
			}
			inputs = append(inputs, Input{Field: Field{Span: p.span(start), Type: typeNode, Name: name}, Near: near, Variadic: variadic})
			p.expect(Newline)
		}
		p.expect(Dedent)
	}
	p.expectWord("asserts")
	p.expect(Newline)
	p.expect(Indent)
	assertions := []Assertion{p.assertion(receiver != nil)}
	for !p.at(Dedent) && !p.at(EOF) {
		assertions = append(assertions, p.assertion(receiver != nil))
	}
	p.expect(Dedent)
	body := p.block()
	return &FunctionDecl{DeclarationLocation: DeclarationLocation{p.span(start)}, Result: result, Name: name, Parameters: parameters, Receiver: receiver, Errors: bound, Inputs: inputs, Assertions: assertions, Body: body}
}

func (p *parser) expressionList(end Kind) []Expr {
	var values []Expr
	if p.at(end) {
		return values
	}
	for {
		values = append(values, p.expression(1))
		if !p.at(",") {
			return values
		}
		p.take()
	}
}

func (p *parser) assertion(method bool) Assertion {
	name := p.expect(Name)
	p.expect(":")
	if p.word("use") {
		start := p.take().Span.Start
		template := p.qualified()
		p.expect("(")
		arguments := p.invocationArguments(")")
		p.expect(")")
		p.singleLine(start, p.span(start).End)
		p.expect(Newline)
		if p.at(Indent) {
			p.fail("use expansion carries no execution mode")
		}
		return Assertion{Span: p.span(name.Span.Start), Name: name, Use: &AssertionUse{Span: p.span(start), Template: template, Arguments: arguments}}
	}
	var receiver Expr
	if method {
		receiver = p.expression(1)
		p.expect("=>")
	}
	arguments := p.invocationArguments("=>")
	for _, argument := range arguments {
		if argument.Spread {
			p.fail("assertion inputs must be explicit positional values")
		}
	}
	p.expect("=>")
	expected := p.completion()
	p.singleLine(name.Span.Start, expected.BodySpan().End)
	p.expect(Newline)
	mode := p.assertionMode()
	return Assertion{Span: p.span(name.Span.Start), Name: name, Receiver: receiver, Arguments: arguments, Expected: expected, Mode: mode}
}

// assertionMode parses the optional indented execution-mode line under an
// assertion row or fixture case.
func (p *parser) assertionMode() *AssertionMode {
	if !p.at(Indent) {
		return nil
	}
	p.take()
	start := p.peek().Span.Start
	if !p.word("using") {
		p.fail("assertion execution mode must start with using")
	}
	p.take()
	mode := &AssertionMode{Span: p.span(start)}
	switch {
	case p.word("raw"):
		p.take()
		path := p.expect(String)
		p.singleLine(start, path.Span.End)
		p.expect(Newline)
		mode.Raw = path
	case p.word("failure"):
		p.take()
		if !p.word("native") && !p.word("emitted") {
			p.fail("using failure selects a native or emitted origin")
		}
		origin := p.take()
		value := p.expression(1)
		p.singleLine(start, value.ExprSpan().End)
		p.expect(Newline)
		mode.Failure = &AssertionFailure{Span: p.span(origin.Span.Start), Origin: origin, Value: value}
	default:
		p.fail("unknown assertion execution mode")
	}
	p.expect(Dedent)
	if p.at(Indent) {
		p.fail("assertion accepts a single execution mode")
	}
	mode.Span = p.span(start)
	return mode
}

func (p *parser) singleLine(start, end int) {
	first, _ := p.file.Position(start)
	last, _ := p.file.Position(end)
	if first.Line != last.Line {
		p.fail("this form must occupy one physical source line")
	}
}

func (p *parser) valueExpression() Expr {
	if p.isCoordination() {
		start := p.peek().Span.Start
		coordination := p.coordination()
		return &CoordinationExpr{ExpressionLocation: p.location(start), Coordination: coordination}
	}
	if p.word("match") {
		start := p.peek().Span.Start
		match := p.match(false)
		return &MatchExpr{ExpressionLocation: p.location(start), Match: match}
	}
	return p.expression(1)
}

func (p *parser) completion() Body {
	start := p.peek().Span.Start
	if p.word("ok") {
		p.take()
		var value Expr
		if !p.at(Newline) {
			value = p.expression(1)
		}
		return &SuccessBody{BodyLocation: BodyLocation{p.span(start)}, Value: value}
	}
	value := p.expression(1)
	constructor, ok := value.(*ConstructorExpr)
	if !ok {
		p.fail("expected ok or an error constructor completion")
	}
	return &FailureBody{BodyLocation: BodyLocation{p.span(start)}, Error: constructor}
}

func (p *parser) block() Block {
	start := p.peek().Span.Start
	var steps []Step
	for {
		if p.isCoordination() {
			steps = append(steps, &CoordinationStep{Coordination: p.coordination()})
			continue
		}
		if p.word("call") {
			call := p.callExpression().(*CallExpr)
			p.expect(Newline)
			steps = append(steps, &CallStep{Span: call.Span, Call: call})
			continue
		}
		if p.word("ok") || p.word("relay") || p.word("match") || p.word("inherit") {
			break
		}
		// An explicit constructor at statement position completes the region;
		// all other admitted starts must be typed immutable bindings.
		if p.at(Name) && p.startsConstructor() {
			break
		}
		if p.at(Dedent) || p.at(EOF) {
			p.fail("a block requires a terminal completion")
		}
		steps = append(steps, &BindingStep{Binding: p.binding()})
	}
	terminal := p.armBody(true)
	p.expect(Dedent)
	return Block{Span: p.span(start), Steps: steps, Terminal: terminal}
}

func (p *parser) startsConstructor() (yes bool) {
	saved := *p
	defer func() {
		*p = saved
		if caught := recover(); caught != nil {
			if _, ok := caught.(parseFailure); !ok {
				panic(caught)
			}
			yes = false
		}
	}()
	p.qualified()
	p.constructorTypes()
	return p.at("(")
}

func (p *parser) armBody(terminal bool) Body {
	start := p.peek().Span.Start
	if p.isCoordination() {
		p.fail("an unbound coordination is a step; use do and an explicit terminal completion")
	}
	if p.word("match") {
		match := p.match(terminal)
		return &MatchBody{BodyLocation: BodyLocation{p.span(start)}, Match: match}
	}
	if p.word("do") {
		if !terminal {
			p.fail("do must complete an enclosing completion region")
		}
		p.take()
		p.expect(Newline)
		p.expect(Indent)
		block := p.block()
		if len(block.Steps) == 0 {
			p.fail("do requires two or more steps")
		}
		return &DoBody{BodyLocation: BodyLocation{p.span(start)}, Block: block}
	}
	if p.word("relay") {
		if !terminal {
			p.fail("relay is only admitted in a completion region")
		}
		p.take()
		call := p.callExpression().(*CallExpr)
		p.expect(Newline)
		return &RelayBody{BodyLocation: BodyLocation{p.span(start)}, Call: call}
	}
	if p.word("inherit") {
		if !terminal {
			p.fail("inherit is only admitted as a terminal completion")
		}
		p.take()
		p.expect(Newline)
		return &InheritBody{BodyLocation: BodyLocation{p.span(start)}}
	}
	if terminal {
		body := p.completion()
		p.expect(Newline)
		return body
	}
	value := p.expression(1)
	p.expect(Newline)
	return &ValueBody{BodyLocation: BodyLocation{p.span(start)}, Value: value}
}
