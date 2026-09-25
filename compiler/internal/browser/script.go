// Structural JavaScript/TypeScript surface scan for the browser audit.
//
// The auditor must tell executed host operations apart from quoted data:
// a module containing the string "Bun." is harmless, while Bun.write(x)
// in code is a forbidden host operation. This file implements the small
// lexical layer that distinction rests on: it tokenizes module bytes into
// code tokens, skipping string literals, template static parts, comments
// and regular-expression literals, then matches exact structural shapes
// (static import/export edges, dynamic import(), require references, and
// Bun/process/eval/Function uses) with source coordinates.
//
// The scanner is intentionally not a full parser: it never builds a syntax
// tree and never evaluates or resolves anything. Anything it cannot lex
// (invalid UTF-8, unterminated literals, malformed import declarations)
// fails closed with an error. TypeScript-only syntax (type annotations,
// import type, generics) needs no special grammar because the matched
// shapes are purely lexical.
package browser

import (
	"fmt"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// Position is a source coordinate: 1-based line, 0-based column in UTF-16
// code units (matching the ir.Mapping generated space), and byte offset.
type Position struct {
	Offset int
	Line   int
	Column int
}

func (position Position) String() string {
	return fmt.Sprintf("%d:%d", position.Line, position.Column)
}

// Edge is one lexed static module edge: the raw specifier text plus whether
// the importing clause is type-only (import type / export type), which the
// transpiler erases deterministically before bundling.
type Edge struct {
	Specifier string
	TypeOnly  bool
	Pos       Position
}

// Finding is one lexed forbidden reference: a host-namespace use
// (Bun.write, process.env, a bare require/eval reference), a dynamic
// import(), or a Function construction.
type Finding struct {
	// Operation names the reference, e.g. "Bun.write", "process", "require",
	// "import()", "eval" or "Function".
	Operation string
	Pos       Position
}

// ModuleScan is the structural surface of one module: every static edge in
// source order and every forbidden reference in source order.
type ModuleScan struct {
	Edges    []Edge
	Findings []Finding
}

// ScanModule tokenizes src as a JavaScript/TypeScript module and returns
// its static edges and forbidden references. Lex failures fail closed.
func ScanModule(src []byte) (ModuleScan, error) {
	tokens, err := tokenizeScript(src)
	if err != nil {
		return ModuleScan{}, err
	}
	members := classMembers(tokens)
	scan := ModuleScan{}
	for index := range tokens {
		if edge, ok := staticEdge(tokens, index); ok {
			scan.Edges = append(scan.Edges, edge)
			continue
		}
		if finding, ok := forbiddenReference(tokens, index, members); ok {
			scan.Findings = append(scan.Findings, finding)
		}
	}
	return scan, nil
}

type scriptTokenKind uint8

const (
	tokIdent scriptTokenKind = iota
	tokNumber
	tokString
	tokPunct
)

type scriptToken struct {
	kind scriptTokenKind
	text string
	pos  Position
}

func (token scriptToken) isIdent(name string) bool {
	return token.kind == tokIdent && token.text == name
}

func (token scriptToken) isPunct(text string) bool {
	return token.kind == tokPunct && token.text == text
}

type scriptLexer struct {
	src       []byte
	offset    int
	line      int
	column    int
	tokens    []scriptToken
	previous  *scriptToken
	completed bool
}

func tokenizeScript(src []byte) ([]scriptToken, error) {
	if !utf8.Valid(src) {
		return nil, fmt.Errorf("module is not valid UTF-8")
	}
	if len(src) == 0 {
		return nil, nil
	}
	lexer := &scriptLexer{src: src, line: 1}
	if len(src) >= 2 && src[0] == '#' && src[1] == '!' {
		for lexer.offset < len(src) && src[lexer.offset] != '\n' {
			lexer.offset++
		}
	}
	if err := lexer.run(); err != nil {
		return nil, err
	}
	return lexer.tokens, nil
}

func (lexer *scriptLexer) run() error {
	for lexer.offset < len(lexer.src) {
		if err := lexer.step(); err != nil {
			return err
		}
	}
	return nil
}

func (lexer *scriptLexer) peek() byte {
	return lexer.src[lexer.offset]
}

func (lexer *scriptLexer) at(text string) bool {
	return strings.HasPrefix(string(lexer.src[lexer.offset:]), text)
}

// advance consumes one rune, tracking line and column. Columns count UTF-16
// code units so finding coordinates compare directly with mapping columns.
func (lexer *scriptLexer) advance() rune {
	runeValue, width := utf8.DecodeRune(lexer.src[lexer.offset:])
	lexer.offset += width
	switch runeValue {
	case '\n':
		lexer.line++
		lexer.column = 0
	case '\r':
		if lexer.offset < len(lexer.src) && lexer.src[lexer.offset] == '\n' {
			lexer.offset++
		}
		lexer.line++
		lexer.column = 0
	case '\u2028', '\u2029':
		lexer.line++
		lexer.column = 0
	default:
		lexer.column += utf16.RuneLen(runeValue)
	}
	return runeValue
}

func (lexer *scriptLexer) position() Position {
	return Position{Offset: lexer.offset, Line: lexer.line, Column: lexer.column}
}

func (lexer *scriptLexer) emit(kind scriptTokenKind, text string, pos Position) {
	token := scriptToken{kind: kind, text: text, pos: pos}
	lexer.tokens = append(lexer.tokens, token)
	lexer.previous = &lexer.tokens[len(lexer.tokens)-1]
}

func (lexer *scriptLexer) step() error {
	start := lexer.position()
	char := lexer.peek()
	switch {
	case char == ' ' || char == '\t' || char == '\n' || char == '\r' || char == '\v' || char == '\f':
		lexer.advance()
		return nil
	case char == '/' && lexer.at("//"):
		for lexer.offset < len(lexer.src) && lexer.src[lexer.offset] != '\n' {
			runeValue, width := utf8.DecodeRune(lexer.src[lexer.offset:])
			lexer.column += utf16.RuneLen(runeValue)
			lexer.offset += width
		}
		return nil
	case char == '/' && lexer.at("/*"):
		lexer.offset += 2
		lexer.column += 2
		for {
			if lexer.offset >= len(lexer.src) {
				return fmt.Errorf("unterminated block comment at %s", start)
			}
			if lexer.at("*/") {
				lexer.offset += 2
				lexer.column += 2
				return nil
			}
			lexer.advance()
		}
	case char == '"' || char == '\'':
		return lexer.string(start)
	case char == '`':
		return lexer.template()
	case char == '/':
		return lexer.slash(start)
	case char == '#' && lexer.offset == 0:
		return fmt.Errorf("unexpected '#' at %s", start)
	case char == '#':
		return lexer.private(start)
	case isDigit(char) || (char == '.' && lexer.offset+1 < len(lexer.src) && isDigit(lexer.src[lexer.offset+1])):
		return lexer.number(start)
	case isIdentStart(char):
		return lexer.ident(start)
	case char >= 0x80:
		return lexer.ident(start)
	default:
		return lexer.punct(start)
	}
}

func isDigit(char byte) bool { return char >= '0' && char <= '9' }

func isIdentStart(char byte) bool {
	return char == '$' || char == '_' || (char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z')
}

func isIdentPart(char byte) bool { return isIdentStart(char) || isDigit(char) }

func (lexer *scriptLexer) string(start Position) error {
	quote := lexer.peek()
	lexer.advance()
	var raw strings.Builder
	for {
		if lexer.offset >= len(lexer.src) {
			return fmt.Errorf("unterminated string literal at %s", start)
		}
		char := lexer.peek()
		if char == '\n' || char == '\r' {
			return fmt.Errorf("unterminated string literal at %s", start)
		}
		if char == '\\' {
			lexer.advance()
			if lexer.offset >= len(lexer.src) {
				return fmt.Errorf("unterminated string literal at %s", start)
			}
			escaped := lexer.peek()
			raw.WriteByte('\\')
			raw.WriteByte(escaped)
			lexer.advance()
			continue
		}
		lexer.advance()
		if char == quote {
			lexer.emit(tokString, raw.String(), start)
			return nil
		}
		raw.WriteByte(char)
	}
}

// template skips a template literal, tokenizing ${} interpolations as code.
// Static parts are data and never produce tokens.
func (lexer *scriptLexer) template() error {
	start := lexer.position()
	lexer.advance()
	for {
		if lexer.offset >= len(lexer.src) {
			return fmt.Errorf("unterminated template literal at %s", start)
		}
		char := lexer.peek()
		if char == '\\' {
			lexer.advance()
			if lexer.offset >= len(lexer.src) {
				return fmt.Errorf("unterminated template literal at %s", start)
			}
			lexer.advance()
			continue
		}
		if char == '`' {
			lexer.advance()
			return nil
		}
		if char == '$' && lexer.offset+1 < len(lexer.src) && lexer.src[lexer.offset+1] == '{' {
			lexer.advance()
			lexer.advance()
			if err := lexer.interpolation(); err != nil {
				return err
			}
			continue
		}
		lexer.advance()
	}
}

func (lexer *scriptLexer) interpolation() error {
	depth := 0
	for {
		if lexer.offset >= len(lexer.src) {
			return fmt.Errorf("unterminated template interpolation")
		}
		if lexer.peek() == '{' {
			lexer.emit(tokPunct, "{", lexer.position())
			lexer.advance()
			depth++
			continue
		}
		if lexer.peek() == '}' {
			pos := lexer.position()
			lexer.advance()
			if depth == 0 {
				return nil
			}
			depth--
			lexer.emit(tokPunct, "}", pos)
			continue
		}
		if err := lexer.step(); err != nil {
			return err
		}
	}
}

// slash lexes comments, /=, division, or a regular-expression literal. The
// regex-or-divide choice follows the standard previous-token heuristic:
// after an operand the slash divides, after an operator or keyword it opens
// a regex. A '}' predecessor reads as division, so a statement-position
// regex there is inspected as code rather than skipped: over-inspection
// fails safe, under-inspection would not.
func (lexer *scriptLexer) slash(start Position) error {
	if lexer.allowRegex() {
		return lexer.regex(start)
	}
	if lexer.at("/=") {
		lexer.emit(tokPunct, "/=", start)
		lexer.offset += 2
		lexer.column += 2
		return nil
	}
	lexer.emit(tokPunct, "/", start)
	lexer.offset++
	lexer.column++
	return nil
}

var regexAllowedPunct = map[string]bool{
	"(": true, ",": true, "=": true, ":": true, "[": true, "!": true,
	"&": true, "|": true, "?": true, "{": true, ";": true, "+": true,
	"-": true, "*": true, "%": true, "<": true, ">": true, "^": true,
	"~": true, "=>": true, "...": true, "&&": true, "||": true, "??": true,
	"**": true, "==": true, "!=": true, "===": true, "!==": true,
	"<=": true, ">=": true, "<<": true, ">>": true, ">>>": true,
	"+=": true, "-=": true, "*=": true, "/=": true, "%=": true,
	"&=": true, "|=": true, "^=": true, "<<=": true, ">>=": true,
	">>>=": true, "**=": true, "&&=": true, "||=": true, "??=": true,
}

var regexAllowedKeyword = map[string]bool{
	"return": true, "typeof": true, "instanceof": true, "in": true,
	"of": true, "new": true, "delete": true, "void": true, "throw": true,
	"case": true, "do": true, "else": true, "yield": true, "await": true,
}

func (lexer *scriptLexer) allowRegex() bool {
	if lexer.previous == nil {
		return true
	}
	prev := *lexer.previous
	if prev.kind == tokPunct {
		return regexAllowedPunct[prev.text]
	}
	if prev.kind == tokIdent {
		return regexAllowedKeyword[prev.text]
	}
	return false
}

func (lexer *scriptLexer) regex(start Position) error {
	lexer.advance()
	inClass := false
	for {
		if lexer.offset >= len(lexer.src) {
			return fmt.Errorf("unterminated regular expression at %s", start)
		}
		char := lexer.peek()
		if char == '\n' || char == '\r' {
			return fmt.Errorf("unterminated regular expression at %s", start)
		}
		if char == '\\' {
			lexer.advance()
			if lexer.offset >= len(lexer.src) {
				return fmt.Errorf("unterminated regular expression at %s", start)
			}
			lexer.advance()
			continue
		}
		lexer.advance()
		switch {
		case char == '[' && !inClass:
			inClass = true
		case char == ']' && inClass:
			inClass = false
		case char == '/' && !inClass:
			for lexer.offset < len(lexer.src) {
				flag := lexer.peek()
				if (flag < 'a' || flag > 'z') && (flag < 'A' || flag > 'Z') {
					break
				}
				lexer.advance()
			}
			return nil
		}
	}
}

func (lexer *scriptLexer) private(start Position) error {
	lexer.advance()
	if lexer.offset >= len(lexer.src) {
		return fmt.Errorf("unexpected '#' at %s", start)
	}
	char := lexer.peek()
	if !isIdentStart(char) && char < 0x80 {
		return fmt.Errorf("unexpected '#' at %s", start)
	}
	var name strings.Builder
	name.WriteByte('#')
	for lexer.offset < len(lexer.src) {
		char = lexer.peek()
		if char < 0x80 && !isIdentPart(char) {
			break
		}
		if char >= 0x80 {
			runeValue, width := utf8.DecodeRune(lexer.src[lexer.offset:])
			name.WriteRune(runeValue)
			lexer.offset += width
			lexer.column += utf16.RuneLen(runeValue)
			continue
		}
		name.WriteByte(char)
		lexer.offset++
		lexer.column++
	}
	lexer.emit(tokIdent, name.String(), start)
	return nil
}

func (lexer *scriptLexer) number(start Position) error {
	var text strings.Builder
	if lexer.at("0x") || lexer.at("0X") || lexer.at("0b") || lexer.at("0B") || lexer.at("0o") || lexer.at("0O") {
		text.WriteByte(lexer.peek())
		lexer.advance()
		text.WriteByte(lexer.peek())
		lexer.advance()
		for lexer.offset < len(lexer.src) {
			char := lexer.peek()
			if char == '_' || isDigit(char) || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F') {
				text.WriteByte(char)
				lexer.advance()
				continue
			}
			break
		}
		if lexer.offset < len(lexer.src) && lexer.peek() == 'n' {
			text.WriteByte('n')
			lexer.advance()
		}
		lexer.emit(tokNumber, text.String(), start)
		return nil
	}
	for lexer.offset < len(lexer.src) {
		char := lexer.peek()
		if char == '_' || isDigit(char) {
			text.WriteByte(char)
			lexer.advance()
			continue
		}
		break
	}
	if lexer.offset < len(lexer.src) && lexer.peek() == '.' &&
		lexer.offset+1 < len(lexer.src) && isDigit(lexer.src[lexer.offset+1]) {
		text.WriteByte('.')
		lexer.advance()
		for lexer.offset < len(lexer.src) {
			char := lexer.peek()
			if char == '_' || isDigit(char) {
				text.WriteByte(char)
				lexer.advance()
				continue
			}
			break
		}
	}
	if lexer.offset < len(lexer.src) && (lexer.peek() == 'e' || lexer.peek() == 'E') {
		saveOffset, saveColumn := lexer.offset, lexer.column
		mark := "e"
		lexer.advance()
		if lexer.offset < len(lexer.src) && (lexer.peek() == '+' || lexer.peek() == '-') {
			mark += string(lexer.peek())
			lexer.advance()
		}
		digits := 0
		for lexer.offset < len(lexer.src) && (isDigit(lexer.peek()) || lexer.peek() == '_') {
			mark += string(lexer.peek())
			lexer.advance()
			digits++
		}
		if digits == 0 {
			lexer.offset, lexer.column = saveOffset, saveColumn
		} else {
			text.WriteString(mark)
		}
	}
	if lexer.offset < len(lexer.src) && lexer.peek() == 'n' {
		text.WriteByte('n')
		lexer.advance()
	}
	lexer.emit(tokNumber, text.String(), start)
	return nil
}

func (lexer *scriptLexer) ident(start Position) error {
	var name strings.Builder
	for lexer.offset < len(lexer.src) {
		char := lexer.peek()
		if char < 0x80 && !isIdentPart(char) {
			break
		}
		if char >= 0x80 {
			runeValue, width := utf8.DecodeRune(lexer.src[lexer.offset:])
			name.WriteRune(runeValue)
			lexer.offset += width
			lexer.column += utf16.RuneLen(runeValue)
			continue
		}
		name.WriteByte(char)
		lexer.advance()
	}
	lexer.emit(tokIdent, name.String(), start)
	return nil
}

var punctuators = []string{
	">>>=", "...", "=>", "===", "!==", ">>>", "<<=", ">>=", "**=",
	"&&=", "||=", "??=", "==", "!=", "<=", ">=", "&&", "||", "??",
	"?.", "++", "--", "<<", ">>", "**", "+=", "-=", "*=", "/=",
	"%=", "&=", "|=", "^=", "(", ")", "{", "}", "[", "]", ";", ",",
	".", ":", "<", ">", "+", "-", "*", "%", "&", "|", "^", "!",
	"~", "=", "?", "@",
}

func (lexer *scriptLexer) punct(start Position) error {
	rest := string(lexer.src[lexer.offset:])
	for _, punct := range punctuators {
		if punct == "?." {
			if strings.HasPrefix(rest, "?.") && (len(rest) < 3 || !isDigit(rest[2])) {
				lexer.emit(tokPunct, punct, start)
				lexer.offset += 2
				lexer.column += 2
				return nil
			}
			continue
		}
		if strings.HasPrefix(rest, punct) {
			lexer.emit(tokPunct, punct, start)
			for range punct {
				lexer.column++
			}
			lexer.offset += len(punct)
			return nil
		}
	}
	return fmt.Errorf("unexpected character %q at %s", rest[:1], start)
}

// staticEdge matches import/export declarations at index: side-effect
// imports, named bindings resolved to their from-specifier, and re-export
// edges. Dynamic import() is a finding, not an edge. Property accesses
// (x.import) and object keys ({import: 1}) never match.
func staticEdge(tokens []scriptToken, index int) (Edge, bool) {
	token := tokens[index]
	if token.kind != tokIdent || (token.text != "import" && token.text != "export") {
		return Edge{}, false
	}
	if index > 0 {
		prev := tokens[index-1]
		if prev.kind == tokPunct && (prev.text == "." || prev.text == "?.") {
			return Edge{}, false
		}
	}
	next := func(offset int) *scriptToken {
		if index+offset < len(tokens) {
			return &tokens[index+offset]
		}
		return nil
	}
	first := next(1)
	if token.text == "import" {
		if first == nil {
			return Edge{}, false
		}
		if first.kind == tokString {
			return Edge{Specifier: first.text, Pos: first.pos}, true
		}
		if first.kind == tokPunct && (first.text == "(" || first.text == ".") {
			return Edge{}, false
		}
		if first.kind == tokPunct && first.text == ":" {
			return Edge{}, false
		}
		typeOnly := first.kind == tokIdent && first.text == "type"
		for offset := 2; ; offset++ {
			following := next(offset)
			if following == nil {
				return Edge{}, false
			}
			if following.isPunct(";") {
				return Edge{}, false
			}
			if following.kind == tokIdent && following.text == "from" {
				spec := next(offset + 1)
				if spec != nil && spec.kind == tokString {
					return Edge{Specifier: spec.text, TypeOnly: typeOnly, Pos: spec.pos}, true
				}
			}
		}
	}
	typeOnly := first != nil && first.kind == tokIdent && first.text == "type"
	for offset := 1; ; offset++ {
		following := next(offset)
		if following == nil {
			return Edge{}, false
		}
		if following.isPunct(";") {
			return Edge{}, false
		}
		if following.kind == tokIdent && following.text == "from" {
			spec := next(offset + 1)
			if spec != nil && spec.kind == tokString {
				return Edge{Specifier: spec.text, TypeOnly: typeOnly, Pos: spec.pos}, true
			}
			// A binding literally named `from`; the clause may continue.
			continue
		}
	}
}

// declarationPrev names identifier positions that declare or bind rather
// than reference: a require/eval/Bun/process in one of these positions is
// a local binding, never the host operation.
var declarationPrev = map[string]bool{
	"function": true, "class": true, "interface": true, "type": true,
	"import": true, "export": true, "from": true, "as": true,
	"satisfies": true, "const": true, "let": true, "var": true,
	"async": true, "get": true, "set": true, "static": true,
	"public": true, "private": true, "protected": true, "override": true,
	"declare": true, "abstract": true,
}

// hostGlobals are objects whose Bun/process/require/eval members reach the
// host namespace: globalThis.Bun is the same operation as bare Bun.
var hostGlobals = map[string]bool{
	"globalThis": true, "global": true, "self": true, "window": true,
}

// forbiddenReference matches host-namespace references at index: dynamic
// import(), require/eval references, Bun/process member access, calls and
// bare references, new Function, and host members off global objects.
// Members (x.require), declarations (async require()), and property keys
// ({Bun: 1}) never match. A bare typeof Bun existence sniff is not a host
// operation and never matches; typeof Bun.x still touches the host.
func forbiddenReference(tokens []scriptToken, index int, members map[int]bool) (Finding, bool) {
	token := tokens[index]
	if token.kind != tokIdent {
		return Finding{}, false
	}
	if members[index] {
		return Finding{}, false
	}
	var prev, next *scriptToken
	if index > 0 {
		prev = &tokens[index-1]
	}
	if index+1 < len(tokens) {
		next = &tokens[index+1]
	}
	if prev != nil && prev.kind == tokPunct && (prev.text == "." || prev.text == "?.") {
		return Finding{}, false
	}
	switch token.text {
	case "import":
		if next != nil && next.kind == tokPunct && next.text == "(" {
			return Finding{Operation: "import()", Pos: token.pos}, true
		}
		return Finding{}, false
	case "require", "eval":
		if prev != nil && prev.kind == tokIdent && declarationPrev[prev.text] {
			return Finding{}, false
		}
		if isKeyPosition(tokens, index, prev, next) {
			return Finding{}, false
		}
		return Finding{Operation: token.text, Pos: token.pos}, true
	case "Bun", "process":
		if prev != nil && prev.kind == tokIdent && declarationPrev[prev.text] {
			return Finding{}, false
		}
		if isKeyPosition(tokens, index, prev, next) {
			return Finding{}, false
		}
		if next != nil && next.kind == tokPunct && (next.text == "." || next.text == "?.") {
			property := "?"
			if index+2 < len(tokens) && tokens[index+2].kind == tokIdent {
				property = strings.TrimPrefix(tokens[index+2].text, "#")
			}
			return Finding{Operation: token.text + "." + property, Pos: token.pos}, true
		}
		if next != nil && next.kind == tokPunct && (next.text == "(" || next.text == "`") {
			return Finding{Operation: token.text + next.text, Pos: token.pos}, true
		}
		if prev != nil && prev.kind == tokIdent && prev.text == "typeof" {
			return Finding{}, false
		}
		return Finding{Operation: token.text, Pos: token.pos}, true
	case "Function":
		if prev != nil && prev.kind == tokIdent && prev.text == "new" {
			return Finding{Operation: "Function", Pos: token.pos}, true
		}
		return Finding{}, false
	default:
		if !hostGlobals[token.text] {
			return Finding{}, false
		}
		if next != nil && next.kind == tokPunct && (next.text == "." || next.text == "?.") &&
			index+2 < len(tokens) && tokens[index+2].kind == tokIdent {
			member := tokens[index+2].text
			if member == "Bun" || member == "process" || member == "require" || member == "eval" {
				return Finding{Operation: token.text + "." + member, Pos: token.pos}, true
			}
		}
		return Finding{}, false
	}
}

// memberPrev names the tokens that can precede a class member definition:
// the class opening brace, member separators, generator stars and member
// modifiers. Any other predecessor (notably '=' and '@') keeps the
// identifier a reference: initializers, computed keys and decorators
// execute.
var memberPrev = map[string]bool{
	"{": true, ";": true, "}": true, "*": true,
	"static": true, "async": true, "get": true, "set": true,
	"declare": true, "override": true, "abstract": true,
	"public": true, "private": true, "protected": true, "readonly": true,
}

// classMembers returns the token indexes that define class members. A
// bare require/eval/Bun/process in member-name position is a method or
// field definition, never a host reference, so the pinned vendor script's
// process() method and any similar member pass while genuine references
// in extends clauses, initializers, decorators and method bodies still
// fail. Unmodified object-literal methods are not recognized (telling an
// object literal from a block needs a full parser) and keep failing
// closed; none occur in compiler output, the pinned runtime, or the
// pinned vendor script.
func classMembers(tokens []scriptToken) map[int]bool {
	members := map[int]bool{}
	for index, token := range tokens {
		if !token.isIdent("class") {
			continue
		}
		if index > 0 {
			prev := tokens[index-1]
			if prev.kind == tokPunct && (prev.text == "." || prev.text == "?.") {
				continue
			}
		}
		if index+1 < len(tokens) {
			next := tokens[index+1]
			if next.kind == tokPunct && next.text == ":" {
				continue
			}
		}
		open := classOpen(tokens, index)
		if open < 0 {
			continue
		}
		end := matchingBrace(tokens, open)
		if end < 0 {
			continue
		}
		markMembers(tokens, members, open, end)
	}
	return members
}

// classOpen finds the body brace of the class at index: the first '{' at
// parenthesis/bracket depth zero past any name and extends clause.
func classOpen(tokens []scriptToken, index int) int {
	paren, bracket := 0, 0
	for scan := index + 1; scan < len(tokens); scan++ {
		token := tokens[scan]
		if token.kind != tokPunct {
			continue
		}
		switch token.text {
		case "(":
			paren++
		case ")":
			paren--
		case "[":
			bracket++
		case "]":
			bracket--
		case "{":
			if paren == 0 && bracket == 0 {
				return scan
			}
		case ";":
			return -1
		}
		if paren < 0 || bracket < 0 {
			return -1
		}
	}
	return -1
}

func matchingBrace(tokens []scriptToken, open int) int {
	depth := 0
	for scan := open; scan < len(tokens); scan++ {
		token := tokens[scan]
		if token.kind != tokPunct {
			continue
		}
		switch token.text {
		case "{":
			depth++
		case "}":
			depth--
			if depth == 0 {
				return scan
			}
		}
	}
	return -1
}

func markMembers(tokens []scriptToken, members map[int]bool, open, end int) {
	brace, bracket, paren := 0, 0, 0
	for scan := open + 1; scan < end; scan++ {
		token := tokens[scan]
		if token.kind == tokPunct {
			switch token.text {
			case "{":
				brace++
			case "}":
				brace--
			case "[":
				bracket++
			case "]":
				bracket--
			case "(":
				paren++
			case ")":
				paren--
			}
			continue
		}
		if token.kind != tokIdent || brace != 0 || bracket != 0 || paren != 0 {
			continue
		}
		switch token.text {
		case "require", "eval", "Bun", "process":
		default:
			continue
		}
		prev := tokens[scan-1]
		if prev.kind == tokPunct && memberPrev[prev.text] {
			members[scan] = true
			continue
		}
		if prev.kind == tokIdent && memberPrev[prev.text] {
			members[scan] = true
		}
	}
}

// isKeyPosition reports whether the identifier at index is a property key,
// label, or other non-reference: next is ':' outside a ternary or case
// label. A conditional colon (a ? b : c) still references its operands.
func isKeyPosition(tokens []scriptToken, index int, prev, next *scriptToken) bool {
	if next == nil || !next.isPunct(":") {
		return false
	}
	if prev != nil && prev.kind == tokIdent && prev.text == "case" {
		return false
	}
	depth := 0
	for scan := index - 1; scan >= 0; scan-- {
		candidate := tokens[scan]
		if candidate.kind != tokPunct {
			continue
		}
		switch candidate.text {
		case ":":
			depth++
		case "?":
			if depth == 0 {
				return false
			}
			depth--
		case "{", "}", ";", "(", ")", "[", "]", ",", "=>":
			return true
		}
	}
	return true
}
