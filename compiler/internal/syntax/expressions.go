package syntax

import "github.com/veighnsche/can-lang/compiler/internal/source"

func (p *parser) location(start int) ExpressionLocation { return ExpressionLocation{p.span(start)} }

// Binding powers increase with precedence. Power is right associative and its
// right operand admits unary minus; logical not binds below comparisons.
func binaryPower(t Token) int {
	switch t.Text {
	case "or":
		return 2
	case "and":
		return 3
	case "<", "<=", ">", ">=", "is":
		return 5
	case "|":
		return 6
	case "^":
		return 7
	case "&":
		return 8
	case "<<", ">>":
		return 9
	case "+", "-":
		return 10
	case "*", "/", "%":
		return 11
	case "**":
		return 13
	}
	return 0
}

func (p *parser) expression(minimum int) Expr {
	p.depth++
	defer func() { p.depth-- }()
	if p.depth > 256 {
		p.fail("expression nesting exceeds 256 levels")
	}
	start := p.peek().Span.Start
	var left Expr
	if p.at("-") || p.at("~") || p.word("not") {
		op := p.take().Text
		power := 12
		if op == "not" {
			power = 4
		}
		operand := p.expression(power)
		left = &UnaryExpr{ExpressionLocation: p.location(start), Operator: op, Operand: operand}
	} else if p.word("call") {
		left = p.callExpression()
	} else if p.word("callable") {
		p.take()
		callee := p.callee()
		types := p.typeArguments()
		left = &ReferenceExpr{ExpressionLocation: p.location(start), Callee: callee, Types: types}
	} else {
		left = p.primary(true)
	}
	return p.expressionTail(left, minimum)
}

func (p *parser) expressionTail(left Expr, minimum int) Expr {
	start := left.ExprSpan().Start
	left = p.postfix(left)
	for {
		power := binaryPower(p.peek())
		if power == 0 || power < minimum {
			break
		}
		op := p.take().Text
		if op == "is" && p.word("not") {
			p.take()
			op = "is not"
		}
		rightPower := power + 1
		if op == "**" {
			rightPower = 12
		}
		right := p.expression(rightPower)
		if power == 5 {
			if chain, ok := left.(*ComparisonExpr); ok {
				chain.Operands = append(chain.Operands, right)
				chain.Operators = append(chain.Operators, op)
				chain.Span = p.span(start)
			} else {
				left = &ComparisonExpr{ExpressionLocation: p.location(start), Operands: []Expr{left, right}, Operators: []string{op}}
			}
		} else {
			left = &BinaryExpr{ExpressionLocation: p.location(start), Operator: op, Left: left, Right: right}
		}
	}
	if minimum <= 1 && p.word("with") {
		p.take()
		parenthesized := p.at("(")
		if parenthesized {
			p.take()
		}
		var fields []Replacement
		for {
			name := p.expect(Name)
			p.expect("=")
			value := p.expression(2)
			fields = append(fields, Replacement{Span: p.span(name.Span.Start), Name: name, Value: value})
			if !parenthesized || !p.at(",") {
				break
			}
			p.take()
		}
		if parenthesized {
			if len(fields) < 2 {
				p.fail("a parenthesized update requires at least two replacements")
			}
			p.expect(")")
		}
		left = &UpdateExpr{ExpressionLocation: p.location(start), Receiver: left, Fields: fields}
	}
	return left
}

func (p *parser) primary(constructors bool) Expr {
	start := p.peek().Span.Start
	switch {
	case p.at("%"):
		if p.probabilityDepth == 0 {
			p.fail("% requires a native probability handler")
		}
		p.take()
		return &ProbabilityExpr{ExpressionLocation: p.location(start)}
	case p.at(Integer) || p.at(Float) || p.at(String) || p.word("true") || p.word("false"):
		t := p.take()
		return &LiteralExpr{ExpressionLocation: p.location(start), Token: t}
	case p.at(Name):
		name := p.qualified()
		if constructors {
			types := p.constructorTypes()
			if p.at("(") {
				args := p.arguments()
				for _, argument := range args {
					if argument.Group != nil {
						p.fail("a constructor cannot receive a state group")
					}
				}
				return &ConstructorExpr{ExpressionLocation: p.location(start), Name: name, Types: types, Arguments: args}
			}
		}
		return &NameExpr{ExpressionLocation: p.location(start), Name: name}
	case p.at("("):
		p.take()
		value := p.expression(1)
		p.expect(")")
		return &GroupExpr{ExpressionLocation: p.location(start), Value: value}
	case p.at("["):
		p.take()
		elements := p.argumentList("]")
		p.expect("]")
		return &ArrayExpr{ExpressionLocation: p.location(start), Elements: elements}
	default:
		p.fail("expected an expression")
		return nil
	}
}

func (p *parser) typeArguments() []TypeNode {
	if !p.at("<") {
		return nil
	}
	p.take()
	out := []TypeNode{p.parseType()}
	for p.at(",") {
		p.take()
		out = append(out, p.parseType())
	}
	p.closeAngle()
	return out
}

// A name followed by < is still a comparison unless a complete explicit
// constructor type application is followed by its arguments. Speculation is
// confined to syntax and restores token fragments as well as cursor state.
func (p *parser) constructorTypes() (out []TypeNode) {
	if !p.at("<") {
		return nil
	}
	saved := *p
	defer func() {
		if caught := recover(); caught != nil {
			if _, ok := caught.(parseFailure); !ok {
				panic(caught)
			}
			*p = saved
			out = nil
		}
	}()
	out = p.typeArguments()
	if !p.at("(") {
		*p = saved
		return nil
	}
	return out
}

func (p *parser) argumentList(end Kind) []Argument {
	var args []Argument
	if p.at(end) {
		return args
	}
	for {
		start := p.peek().Span.Start
		spread := p.at("...")
		if spread {
			p.take()
		}
		value := p.expression(1)
		args = append(args, Argument{Span: p.span(start), Spread: spread, Value: value})
		if !p.at(",") {
			return args
		}
		p.take()
		if p.at(end) {
			p.fail("trailing comma is not allowed")
		}
	}
}

func (p *parser) arguments() []Argument {
	p.expect("(")
	args := p.invocationArguments(")")
	p.expect(")")
	return args
}

func (p *parser) invocationArguments(end Kind) []Argument {
	var args []Argument
	if !p.at(end) {
		for {
			start := p.peek().Span.Start
			spread := p.at("...")
			if spread {
				p.take()
			}
			value, group := p.argumentValue()
			if group != nil {
				if spread {
					p.fail("a state group cannot be spread")
				}
				args = append(args, Argument{Span: p.span(start), Group: group})
				if !p.at(end) {
					p.fail("a state group must be the final argument")
				}
				break
			}
			args = append(args, Argument{Span: p.span(start), Spread: spread, Value: value})
			if !p.at(",") {
				break
			}
			p.take()
		}
	}
	return args
}

// Parenthesized single values retain their GroupExpr representation because
// ordinary grouping and one-value state syntax are indistinguishable before
// lookup. Empty/multiple-value groups have a separate argument-only node.
// Parse each argument exactly once. Speculatively reparsing a one-value group
// would multiply work for recursively nested calls with parenthesized inputs.
func (p *parser) argumentValue() (Expr, *ArgumentGroup) {
	if !p.at("(") {
		return p.expression(1), nil
	}
	start := p.take().Span.Start
	values := p.expressionList(")")
	p.expect(")")
	if len(values) == 1 {
		group := &GroupExpr{ExpressionLocation: p.location(start), Value: values[0]}
		return p.expressionTail(group, 1), nil
	}
	return nil, &ArgumentGroup{Span: p.span(start), Values: values}
}

func (p *parser) postfix(left Expr) Expr {
	start := left.ExprSpan().Start
	for {
		switch {
		case p.at("."):
			p.take()
			field := p.expect(Name)
			left = &FieldExpr{ExpressionLocation: p.location(start), Receiver: left, Field: field}
		case p.at("["):
			p.take()
			var first Expr
			if !p.at(":") {
				first = p.expression(1)
			}
			if p.at(":") {
				p.take()
				var end Expr
				if !p.at("]") {
					end = p.expression(1)
				}
				p.expect("]")
				left = &SliceExpr{ExpressionLocation: p.location(start), Receiver: left, Start: first, End: end}
			} else {
				p.expect("]")
				left = &IndexExpr{ExpressionLocation: p.location(start), Receiver: left, Index: first}
			}
		default:
			return left
		}
	}
}

func (p *parser) callee() Expr { return p.postfix(p.primary(false)) }

func (p *parser) callExpression() Expr {
	start := p.expectWord("call").Span.Start
	return p.invocationExpression(start)
}

func (p *parser) invocationExpression(start int) *CallExpr {
	callee := p.callee()
	types := p.typeArguments()
	args := p.arguments()
	invocation := Invocation{Span: p.span(callee.ExprSpan().Start), Callee: callee, Types: types, Arguments: args}
	var methods []MethodInvocation
	for p.at(".") {
		// A property after a call is still ordinary postfix syntax unless it is
		// followed by a method argument list (with optional type arguments).
		saved := *p
		p.take()
		name := p.expect(Name)
		types := p.constructorTypes()
		if !p.at("(") {
			*p = saved
			break
		}
		args := p.arguments()
		methods = append(methods, MethodInvocation{Span: p.span(name.Span.Start), Name: name, Types: types, Arguments: args})
	}
	p.singleLine(start, p.last.Span.End)
	return &CallExpr{ExpressionLocation: p.location(start), Invocation: invocation, Methods: methods}
}

// ParseExpression reads an ordinary expression fragment. Grammar-owned block
// expressions will be supplied by the file parser; this entry point never runs
// a provider, constructor, or function.
func ParseExpression(file *source.File) (node Expr, diagnostics []Diagnostic) {
	result := Lex(file)
	if !result.OK() {
		return nil, result.Diagnostics
	}
	p := newParser(result)
	defer func() {
		if failure := recover(); failure != nil {
			if parsed, ok := failure.(parseFailure); ok {
				node = nil
				diagnostics = []Diagnostic{parsed.diagnostic}
			} else {
				panic(failure)
			}
		}
	}()
	node = p.expression(1)
	if p.at(Newline) {
		p.take()
	}
	p.expect(EOF)
	return node, nil
}
