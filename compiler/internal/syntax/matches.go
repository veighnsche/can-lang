package syntax

func (p *parser) match(terminal bool) Match {
	p.depth++
	defer func() { p.depth-- }()
	if p.depth > 256 {
		p.fail("match nesting exceeds 256 levels")
	}
	start := p.expectWord("match").Span.Start
	match := Match{Kind: ValueMatch}
	if p.word("call") {
		if !terminal {
			p.fail("match call cannot initialize an ordinary value")
		}
		match.Kind = CallMatch
		match.Call = p.callExpression().(*CallExpr)
	} else if p.word("chain") {
		if !terminal {
			p.fail("match chain cannot initialize an ordinary value")
		}
		p.take()
		match.Kind = ChainMatch
	} else {
		match.Values = p.expressionList(Newline)
		if len(match.Values) == 0 {
			p.fail("match requires a scrutinee")
		}
	}
	p.expect(Newline)
	p.expect(Indent)
	if match.Kind == ChainMatch {
		for p.word("call") {
			call := p.callExpression().(*CallExpr)
			var binding *Field
			if p.word("as") {
				p.take()
				field := p.field()
				binding = &field
			}
			match.Chain = append(match.Chain, ChainEntry{Span: p.span(call.Span.Start), Call: call, Binding: binding})
			p.expect(Newline)
		}
		if len(match.Chain) == 0 {
			p.fail("match chain requires at least one call")
		}
	}
	if match.Kind == CallMatch && p.word("when") {
		p.take()
		p.expect(Newline)
		p.expect(Indent)
		match.When = append(match.When, p.assertion(false))
		for !p.at(Dedent) && !p.at(EOF) {
			match.When = append(match.When, p.assertion(false))
		}
		p.expect(Dedent)
	}
	for !p.at(Dedent) && !p.at(EOF) {
		start := p.peek().Span.Start
		arm := MatchArm{}
		if match.Kind == ValueMatch {
			arm.Patterns = append(arm.Patterns, p.pattern())
			for p.at(",") {
				p.take()
				arm.Patterns = append(arm.Patterns, p.pattern())
			}
		} else {
			outcome := p.outcomePattern()
			arm.Outcome = &outcome
			if p.at(Newline) {
				if outcome.Binding != nil || outcome.Alias != nil || outcome.StandardFailure {
					p.fail("only bare ok and unaliased error arms may forward")
				}
				arm.Forward = true
				p.take()
				arm.Span = p.span(start)
				match.Arms = append(match.Arms, arm)
				continue
			}
		}
		p.expect("=>")
		arm.Body = p.armBody(terminal)
		arm.Span = p.span(start)
		match.Arms = append(match.Arms, arm)
	}
	if len(match.Arms) == 0 {
		p.fail("match requires at least one arm")
	}
	p.expect(Dedent)
	match.Span = p.span(start)
	return match
}

func (p *parser) outcomePattern() OutcomePattern {
	start := p.peek().Span.Start
	pattern := OutcomePattern{}
	switch {
	case p.word("ok"):
		p.take()
		pattern.Success = true
		if !p.at("=>") && !p.at(Newline) {
			field := p.field()
			pattern.Binding = &field
		}
	case p.at("["):
		p.take()
		p.expect(Wildcard)
		p.expect("]")
		pattern.StandardFailure = true
		if p.word("as") {
			p.take()
			field := p.field()
			pattern.Binding = &field
		}
	case p.at(Name):
		name := p.qualified()
		types := p.typeArguments()
		pattern.Error = &NamedType{Span: name.Span, Name: name, Arguments: types}
		if p.word("as") {
			p.take()
			alias := p.expect(Name)
			pattern.Alias = &alias
		}
	default:
		p.fail("expected an explicit success, named error, or [_] completion arm")
	}
	pattern.Span = p.span(start)
	return pattern
}

func (p *parser) pattern() PatternNode {
	p.depth++
	defer func() { p.depth-- }()
	if p.depth > 256 {
		p.fail("pattern nesting exceeds 256 levels")
	}
	start := p.peek().Span.Start
	first := p.singlePattern()
	if !p.at("|") {
		return first
	}
	alternatives := []PatternNode{first}
	for p.at("|") {
		p.take()
		alternatives = append(alternatives, p.singlePattern())
	}
	return &AlternativePattern{PatternLocation: PatternLocation{p.span(start)}, Alternatives: alternatives}
}

func (p *parser) literalPattern() *LiteralPattern {
	start := p.peek().Span.Start
	negative := p.at("-")
	if negative {
		p.take()
	}
	if !(p.at(Integer) || p.at(Float) || p.at(String) || p.word("true") || p.word("false")) {
		p.fail("expected a literal pattern")
	}
	if negative && !p.at(Integer) && !p.at(Float) {
		p.fail("only numeric literals may be negative")
	}
	token := p.take()
	return &LiteralPattern{PatternLocation: PatternLocation{p.span(start)}, Literal: token, Negative: negative}
}

func (p *parser) singlePattern() PatternNode {
	start := p.peek().Span.Start
	switch {
	case p.word("bind") && p.index+1 < len(p.tokens) && p.tokens[p.index+1].Kind == Name:
		// A complete nominal pattern is never followed by another name,
		// so bind <name> is unambiguous; bare bind stays nominal.
		p.take()
		name := p.expect(Name)
		return &BindPattern{PatternLocation: PatternLocation{p.span(start)}, Name: name}
	case p.at(Wildcard):
		p.take()
		return &WildcardPattern{PatternLocation{p.span(start)}}
	case p.at("["):
		p.take()
		var elements []PatternNode
		var rest *Token
		if !p.at("]") {
			for {
				if p.at("...") {
					p.take()
					name := p.expect(Name)
					rest = &name
					break
				}
				elements = append(elements, p.pattern())
				if !p.at(",") {
					break
				}
				p.take()
			}
		}
		p.expect("]")
		return &ArrayPattern{PatternLocation: PatternLocation{p.span(start)}, Elements: elements, Rest: rest}
	case p.at(Name):
		name := p.qualified()
		types := p.typeArguments()
		if !p.at("(") {
			return &NamePattern{PatternLocation: PatternLocation{p.span(start)}, Name: name, Types: types}
		}
		p.take()
		var fields []PatternNode
		if !p.at(")") {
			for {
				fields = append(fields, p.pattern())
				if !p.at(",") {
					break
				}
				p.take()
			}
		}
		p.expect(")")
		return &ConstructorPattern{PatternLocation: PatternLocation{p.span(start)}, Name: name, Types: types, Fields: fields}
	default:
		lower := p.literalPattern()
		if !p.at("..") {
			return lower
		}
		p.take()
		upper := p.literalPattern()
		if lower.Literal.Kind != Integer || upper.Literal.Kind != Integer {
			p.fail("range patterns require integer literal bounds")
		}
		return &RangePattern{PatternLocation: PatternLocation{p.span(start)}, Lower: lower, Upper: upper}
	}
}
