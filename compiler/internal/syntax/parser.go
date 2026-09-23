package syntax

import (
	"fmt"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/source"
)

// parser owns a private token stream: splitting a generic closing angle must
// not alter a lexer result retained by diagnostics or editor callers.
type parser struct {
	file             *source.File
	tokens           []Token
	pending          *Token
	index            int
	last             Token
	depth            int
	probabilityDepth int
}

type parseFailure struct{ diagnostic Diagnostic }

func newParser(result Result) *parser {
	return &parser{file: result.File, tokens: append([]Token(nil), result.Tokens...)}
}

func (p *parser) peek() Token {
	if p.pending != nil {
		return *p.pending
	}
	return p.tokens[p.index]
}
func (p *parser) at(kind Kind) bool     { return p.peek().Kind == kind }
func (p *parser) word(word string) bool { return p.peek().IsWord(word) }
func (p *parser) take() Token {
	t := p.peek()
	if p.pending != nil {
		p.pending = nil
	} else if t.Kind != EOF {
		p.index++
	}
	p.last = t
	return t
}
func (p *parser) fail(message string) {
	panic(parseFailure{Diagnostic{Code: "syntax", Message: message, Span: p.peek().Span}})
}
func (p *parser) expect(kind Kind) Token {
	if !p.at(kind) {
		p.fail(fmt.Sprintf("expected %s, found %q", kind, p.peek().Text))
	}
	return p.take()
}
func (p *parser) expectWord(word string) Token {
	if !p.word(word) {
		p.fail("expected " + word)
	}
	return p.take()
}
func (p *parser) span(start int) source.Span { return source.Span{Start: start, End: p.last.Span.End} }

// Only a generic grammar context may split a maximal right-angle operator.
// Each fragment still has its precise original-byte span.
func (p *parser) closeAngle() {
	t := p.peek()
	if t.Kind == ">>" || t.Kind == ">=" {
		p.take()
		p.last = Token{Kind: ">", Text: ">", Span: source.Span{Start: t.Span.Start, End: t.Span.Start + 1}}
		t.Kind = Kind(t.Text[1:])
		t.Text = t.Text[1:]
		t.Span.Start++
		p.pending = &t
		return
	}
	p.expect(">")
}

func (p *parser) qualified() QualifiedName {
	first := p.expect(Name)
	n := QualifiedName{Span: first.Span, Name: first.Text}
	if p.at("::") {
		p.take()
		last := p.expect(Name)
		n.Package, n.Name, n.Span.End = n.Name, last.Text, last.Span.End
	}
	return n
}

func (p *parser) typeList(end Kind) []TypeNode {
	var out []TypeNode
	if p.at(end) {
		return out
	}
	for {
		out = append(out, p.parseType())
		if !p.at(",") {
			return out
		}
		p.take()
		if p.at(end) {
			p.fail("trailing comma is not allowed")
		}
	}
}

func (p *parser) errorBound() ErrorBound {
	start := p.expectWord("emits").Span.Start
	if p.word("calculated") {
		p.fail("emits calculated is only admitted on wrap declarations")
	}
	p.expect("[")
	types := p.typeList("]")
	p.expect("]")
	return ErrorBound{Span: p.span(start), Types: types}
}

func (p *parser) parseType() TypeNode {
	// Bound recursion for malformed/untrusted editor input, independently of
	// runtime evaluation (which this package never performs).
	p.depth++
	defer func() { p.depth-- }()
	if p.depth > 256 {
		p.fail("type nesting exceeds 256 levels")
	}
	start := p.peek().Span.Start
	var node TypeNode
	switch {
	case p.word("callable"):
		p.take()
		result := p.parseType()
		p.expect("(")
		inputs := p.typeList(")")
		p.expect(")")
		bound := p.errorBound()
		node = &CallableType{Span: p.span(start), Result: result, Inputs: inputs, Errors: bound}
	case p.word("choice_arm"):
		p.take()
		p.expect("<")
		result := p.parseType()
		p.closeAngle()
		bound := p.errorBound()
		node = &ChoiceArmType{Span: p.span(start), Result: result, Errors: bound}
	case p.word("int") || p.word("float") || p.word("bool") || p.word("str") || p.word("void"):
		t := p.take()
		node = &NamedType{Span: t.Span, Name: QualifiedName{Span: t.Span, Name: t.Text}}
	default:
		name := p.qualified()
		var arguments []TypeNode
		if p.at("<") {
			p.take()
			arguments = append(arguments, p.parseType())
			for p.at(",") {
				p.take()
				arguments = append(arguments, p.parseType())
			}
			p.closeAngle()
		}
		node = &NamedType{Span: p.span(start), Name: name, Arguments: arguments}
	}
	for p.at("[") {
		p.take()
		p.expect("]")
		node = &ArrayType{Span: p.span(start), Element: node}
	}
	return node
}

// ParseType parses a standalone type fragment for tooling and grammar tests.
// Lexical errors are retained verbatim; no partial node escapes on failure.
func ParseType(file *source.File) (node TypeNode, diagnostics []Diagnostic) {
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
	node = p.parseType()
	if p.at(Newline) {
		p.take()
	}
	p.expect(EOF)
	return node, nil
}

// FormatType renders syntax alone, without looking up any name or evaluating
// any code. Parentheses and emits ownership follow C2's recursive type grammar.
func FormatType(node TypeNode) string {
	list := func(types []TypeNode) string {
		parts := make([]string, len(types))
		for i, t := range types {
			parts[i] = FormatType(t)
		}
		return strings.Join(parts, ", ")
	}
	switch n := node.(type) {
	case *NamedType:
		name := n.Name.Name
		if n.Name.Package != "" {
			name = n.Name.Package + "::" + name
		}
		if len(n.Arguments) > 0 {
			name += "<" + list(n.Arguments) + ">"
		}
		return name
	case *ArrayType:
		return FormatType(n.Element) + "[]"
	case *CallableType:
		return "callable " + FormatType(n.Result) + " (" + list(n.Inputs) + ") emits [" + list(n.Errors.Types) + "]"
	case *ChoiceArmType:
		return "choice_arm<" + FormatType(n.Result) + "> emits [" + list(n.Errors.Types) + "]"
	default:
		panic("unknown type node")
	}
}
