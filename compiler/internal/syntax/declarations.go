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
	p := newParser(lexed)
	p.recovering = true
	parsed := &File{Source: file, Comments: lexed.Comments, InvalidDeclarations: map[Declaration][]source.Span{}}
	result = ParseResult{File: parsed, Diagnostics: append([]Diagnostic(nil), lexed.Diagnostics...)}
	// Header failure makes package membership unprovable. Retain the source
	// and diagnostics, but do not invent a package or expose its declarations.
	if issue := p.attempt(func() { parsed.Header = p.packageHeader() }); issue != nil {
		result.Diagnostics = append(result.Diagnostics, *issue)
		parsed.Invalid = append(parsed.Invalid, source.Span{Start: file.BOMLength(), End: len(file.Text())})
		return result
	}
	for !p.at(EOF) {
		start := p.peek().Span.Start
		contextStart := len(p.incomplete)
		p.declarationName = Token{}
		p.invalidDeclaration = nil
		var declaration Declaration
		issue := p.attempt(func() { declaration = p.declaration() })
		if issue != nil {
			// Synchronize against physical top-level token positions. Layout
			// tokens have zero width and never count as declarations.
			p.pending = nil
			for !p.at(EOF) {
				token := p.peek()
				position, _ := file.Position(token.Span.Start)
				if token.Span.Start > start && token.Span.End > token.Span.Start && position.Column == 1 && token.Kind != Newline {
					break
				}
				p.take()
			}
			result.Diagnostics = append(result.Diagnostics, *issue)
			if p.declarationName.Text != "" {
				parsed.InvalidNames = append(parsed.InvalidNames, p.declarationName)
			}
			parsed.Invalid = append(parsed.Invalid, source.Span{Start: start, End: p.peek().Span.Start})
			continue
		}
		if len(p.invalidDeclaration) != 0 {
			parsed.InvalidDeclarations[declaration] = append([]source.Span(nil), p.invalidDeclaration...)
			parsed.Invalid = append(parsed.Invalid, p.invalidDeclaration...)
		}
		damaged := false
		for _, diagnostic := range lexed.Diagnostics {
			if diagnostic.Span.Start >= start && diagnostic.Span.Start < declaration.DeclSpan().End {
				damaged = true
				break
			}
		}
		if damaged {
			if fn, ok := declaration.(*FunctionDecl); ok {
				safe := true
				for _, diagnostic := range lexed.Diagnostics {
					if diagnostic.Span.Start >= start && diagnostic.Span.Start < declaration.DeclSpan().End {
						assertionFault := false
						for _, row := range fn.Assertions {
							if diagnostic.Span.Start >= row.Span.Start && diagnostic.Span.Start <= row.Span.End {
								fn.InvalidAssertions = append(fn.InvalidAssertions, row.Span)
								assertionFault = true
								break
							}
						}
						for _, invalid := range fn.InvalidAssertions {
							if diagnostic.Span.Start >= invalid.Start && diagnostic.Span.Start <= invalid.End {
								assertionFault = true
								break
							}
						}
						if assertionFault {
							continue
						}
						if diagnostic.Span.Start < fn.Body.Span.Start {
							safe = false
							break
						}
						fn.Body.Invalid = append(fn.Body.Invalid, diagnostic.Span)
					}
				}
				validRows := fn.Assertions[:0]
				for _, row := range fn.Assertions {
					valid := true
					for _, invalid := range fn.InvalidAssertions {
						if row.Span.Start >= invalid.Start && row.Span.Start < invalid.End {
							valid = false
							break
						}
					}
					if valid {
						validRows = append(validRows, row)
					}
				}
				fn.Assertions = validRows
				if safe {
					parsed.Declarations = append(parsed.Declarations, declaration)
				} else {
					parsed.Declarations = append(parsed.Declarations, declaration)
					parsed.InvalidDeclarations[declaration] = append(parsed.InvalidDeclarations[declaration], declaration.DeclSpan())
				}
			} else {
				parsed.Declarations = append(parsed.Declarations, declaration)
				parsed.InvalidDeclarations[declaration] = append(parsed.InvalidDeclarations[declaration], declaration.DeclSpan())
			}
			parsed.Invalid = append(parsed.Invalid, declaration.DeclSpan())
		} else {
			parsed.Declarations = append(parsed.Declarations, declaration)
		}
		for i := contextStart; i < len(p.incomplete); i++ {
			p.incomplete[i].Scope = declaration.DeclSpan()
		}
	}
	result.Diagnostics = append(result.Diagnostics, p.recovered...)
	parsed.Incomplete = p.incomplete
	return result
}

// attempt catches only canonical grammar failures, never implementation bugs.
func (p *parser) attempt(parse func()) (diagnostic *Diagnostic) {
	defer func() {
		if caught := recover(); caught != nil {
			if failure, ok := caught.(parseFailure); ok {
				diagnostic = &failure.diagnostic
			} else {
				panic(caught)
			}
		}
	}()
	parse()
	return nil
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
	p.rememberDeclarationName(name)
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
			p.rememberDeclarationName(name)
			var dependency *Token
			if p.at("::") {
				p.take()
				dep := name
				dependency = &dep
				name = p.expect(Name)
			}
			var alias *Token
			if p.word("as") {
				p.take()
				t := p.expect(Name)
				alias = &t
			}
			start := name.Span.Start
			if dependency != nil {
				start = dependency.Span.Start
			}
			uses = append(uses, Import{Span: p.span(start), Dependency: dependency, Package: name, Alias: alias})
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
	p.rememberDeclarationName(name)
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
	case p.word("scenario"):
		return p.scenario()
	case p.word("action"):
		return p.action()
	case p.word("fn"):
		return p.function()
	case p.word("record") || p.word("owner") && p.index+1 < len(p.tokens) && p.tokens[p.index+1].IsWord("record"):
		owned := p.word("owner")
		if owned {
			p.take()
		}
		p.take()
		name := p.expect(Name)
		p.rememberDeclarationName(name)
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
				if span, failed := p.recoverLine(func() { field := p.field(); p.expect(Newline); fields = append(fields, field) }); failed {
					p.invalidDeclaration = append(p.invalidDeclaration, span)
				}
			}
			p.expect(Dedent)
		}
		return &RecordDecl{DeclarationLocation: DeclarationLocation{p.span(start)}, Name: name, Owner: owned, Parameters: parameters, Fields: fields}
	case p.word("variant"):
		p.take()
		name := p.expect(Name)
		p.rememberDeclarationName(name)
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
		name := p.expect(Name)
		p.rememberDeclarationName(name)
		parameters := p.parameters()
		p.expect("{")
		var fields []Field
		if !p.at("}") {
			for {
				fields = append(fields, p.field())
				if !p.at(",") {
					break
				}
				p.take()
			}
		}
		p.expect("}")
		p.expect(Newline)
		return &ErrorDecl{DeclarationLocation: DeclarationLocation{p.span(start)}, Name: name, Parameters: parameters, Fields: fields}
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
	p.rememberDeclarationName(name)
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
			span, failed := p.recoverLine(func() {
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
				p.rememberDeclarationName(name)
				if near && variadic {
					p.fail("a near input cannot be variadic")
				}
				inputs = append(inputs, Input{Field: Field{Span: p.span(start), Type: typeNode, Name: name}, Near: near, Variadic: variadic})
				p.expect(Newline)
			})
			if failed {
				p.invalidDeclaration = append(p.invalidDeclaration, span)
			}
		}
		p.expect(Dedent)
	}
	p.expectWord("asserts")
	p.expect(Newline)
	p.expect(Indent)
	var assertions []Assertion
	var invalidAssertions []source.Span
	for !p.at(Dedent) && !p.at(EOF) {
		span, failed := p.recoverLine(func() { assertions = append(assertions, p.assertion(receiver != nil)) })
		if failed {
			invalidAssertions = append(invalidAssertions, span)
		}
	}
	p.expect(Dedent)
	body := p.block()
	return &FunctionDecl{InvalidAssertions: invalidAssertions, DeclarationLocation: DeclarationLocation{p.span(start)}, Result: result, Name: name, Parameters: parameters, Receiver: receiver, Errors: bound, Inputs: inputs, Assertions: assertions, Body: body}
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
	start := p.peek().Span.Start
	var scenario *Token
	var name Token
	if p.scenarioTag() {
		p.take()
		tag := p.expect(Name)
		scenario = &tag
		name = tag
		p.expect(":")
	} else {
		name = p.expect(Name)
		p.expect(":")
	}
	if p.word("use") {
		use := p.take().Span.Start
		template := p.qualified()
		p.expect("(")
		arguments := p.invocationArguments(")")
		p.expect(")")
		p.singleLine(use, p.span(use).End)
		p.expect(Newline)
		if p.at(Indent) {
			p.fail("use expansion carries no execution mode")
		}
		return Assertion{Span: p.span(start), Name: name, Scenario: scenario, Use: &AssertionUse{Span: p.span(use), Template: template, Arguments: arguments}}
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
	var links []QualifiedName
	if p.word("link") {
		p.take()
		for {
			links = append(links, p.qualified())
			if !p.at(",") {
				break
			}
			p.take()
		}
	}
	end := expected.BodySpan().End
	if len(links) != 0 {
		end = links[len(links)-1].Span.End
	}
	p.singleLine(start, end)
	p.expect(Newline)
	mode := p.assertionMode()
	return Assertion{Span: p.span(start), Name: name, Receiver: receiver, Arguments: arguments, Expected: expected, Mode: mode, Scenario: scenario, Links: links}
}

// linkClauseFollows reports whether a link clause starts at the cursor: the
// word link followed by a target name. A bare value literally named link
// keeps its meaning when anything else follows: no valid value expression
// continues with an adjacent bare name, and qualified link::name values
// still parse as expressions.
func (p *parser) linkClauseFollows() bool {
	if !p.word("link") || p.pending != nil {
		return false
	}
	next := p.index + 1
	if next >= len(p.tokens) {
		return false
	}
	return p.tokens[next].Kind == Name
}

// scenarioTag reports whether the row starts with a `scenario name :` tag.
// A bare selector literally named scenario keeps its plain meaning, since
// scenario is contextual: only a following Name and colon form a tag.
func (p *parser) scenarioTag() bool {
	if !p.word("scenario") || p.pending != nil {
		return false
	}
	first, second := p.index+1, p.index+2
	if second >= len(p.tokens) {
		return false
	}
	return p.tokens[first].Kind == Name && p.tokens[second].Kind == ":"
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
		if !p.at(Newline) && !p.linkClauseFollows() {
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
	block := Block{}
	for {
		begin := p.peek().Span.Start
		terminal := false
		if !p.isCoordination() && (p.word("ok") || p.word("relay") || p.word("match") || p.word("inherit") || p.at(Name) && p.startsConstructor()) {
			terminal = true
		}
		issue := p.attempt(func() {
			if terminal {
				block.Terminal = p.armBody(true)
				p.expect(Dedent)
				return
			}
			if p.at(Dedent) || p.at(EOF) {
				p.fail("a block requires a terminal completion")
			}
			if p.isCoordination() {
				block.Steps = append(block.Steps, &CoordinationStep{Coordination: p.coordination()})
				return
			}
			if p.word("call") {
				call := p.callExpression().(*CallExpr)
				p.expect(Newline)
				block.Steps = append(block.Steps, &CallStep{Span: call.Span, Call: call})
				return
			}
			block.Steps = append(block.Steps, &BindingStep{Binding: p.binding()})
		})
		if issue == nil {
			if terminal {
				break
			}
			continue
		}
		if !p.recovering {
			panic(parseFailure{*issue})
		}
		p.recovered = append(p.recovered, *issue)
		p.pending = nil
		// No statement crosses a physical line except explicit block syntax;
		// nested layout tokens are skipped only inside the failed statement.
		for !p.at(EOF) && !p.at(Dedent) && !p.at(Newline) {
			p.take()
		}
		if p.at(Newline) {
			p.take()
		}
		block.Invalid = append(block.Invalid, source.Span{Start: begin, End: p.peek().Span.Start})
		if p.at(Dedent) {
			p.take()
			break
		}
		if p.at(EOF) {
			break
		}
	}
	block.Span = source.Span{Start: start, End: p.last.Span.End}
	return block
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
	return p.at("(") || p.at("{")
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

// recoverLine resumes only at canonical layout boundaries. Failed fragments
// remain explicitly invalid and never become executable syntax.
func (p *parser) recoverLine(parse func()) (source.Span, bool) {
	start := p.peek().Span.Start
	if !p.recovering {
		parse()
		return source.Span{}, false
	}
	issue := p.attempt(parse)
	if issue == nil {
		return source.Span{}, false
	}
	p.recovered = append(p.recovered, *issue)
	p.pending = nil
	for !p.at(Newline) && !p.at(Dedent) && !p.at(EOF) {
		p.take()
	}
	end := p.peek().Span.Start
	if p.at(Newline) {
		end = p.take().Span.End
	}
	return source.Span{Start: start, End: end}, true
}
