package syntax

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/veighnsche/can-lang/compiler/internal/source"
)

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:_[a-z0-9]+)*$`)

type delimiter struct {
	kind   byte
	offset int
}
type lexer struct {
	file          *source.File
	text          string
	pos           int
	result        Result
	level         int
	leadingSpaces int
	lineStart     int
	significant   bool
	delimiters    []delimiter
}

// Lex emits significant-line layout tokens, preserving comment trivia separately.
// Invalid input fails closed at its first precise diagnostic; parsers must not
// consume a result with diagnostics. Every result ends in one EOF token.
func Lex(file *source.File) Result {
	l := &lexer{file: file, text: file.Text(), pos: file.BOMLength(), lineStart: file.BOMLength(), result: Result{File: file}}
	for l.pos < len(l.text) && len(l.result.Diagnostics) == 0 {
		l.next()
	}
	if len(l.result.Diagnostics) == 0 && len(l.delimiters) > 0 {
		d := l.delimiters[len(l.delimiters)-1]
		l.fail("CAN-LEX-DELIMITER", "unclosed delimiter", d.offset, d.offset+1)
	}
	if len(l.result.Diagnostics) == 0 {
		if l.significant {
			l.emit(Newline, l.pos, l.pos, "")
		}
		for l.level > 0 {
			l.emit(Dedent, l.pos, l.pos, "")
			l.level--
		}
	}
	l.emit(EOF, l.pos, l.pos, "")
	return l.result
}
func (l *lexer) emit(kind Kind, start, end int, value string) {
	l.result.Tokens = append(l.result.Tokens, Token{Kind: kind, Span: source.Span{Start: start, End: end}, Text: l.text[start:end], Value: value})
}
func (l *lexer) fail(code, message string, start, end int) {
	l.result.Diagnostics = append(l.result.Diagnostics, Diagnostic{Code: code, Message: message, Span: source.Span{Start: start, End: end}})
}
func (l *lexer) layout() bool {
	if l.significant {
		return true
	}
	if l.leadingSpaces%4 != 0 {
		l.fail("CAN-LEX-INDENT", "indentation must use four spaces per level", l.lineStart, l.pos)
		return false
	}
	level := l.leadingSpaces / 4
	if level > l.level+1 {
		l.fail("CAN-LEX-INDENT", "indentation cannot skip a level", l.lineStart, l.pos)
		return false
	}
	if len(l.result.Tokens) == 0 && level != 0 {
		l.fail("CAN-LEX-INDENT", "first significant line must be unindented", l.lineStart, l.pos)
		return false
	}
	for l.level < level {
		l.emit(Indent, l.pos, l.pos, "")
		l.level++
	}
	for l.level > level {
		l.emit(Dedent, l.pos, l.pos, "")
		l.level--
	}
	l.significant = true
	return true
}
func (l *lexer) newline() {
	start := l.pos
	if l.text[l.pos] == '\r' {
		l.pos++
	}
	l.pos++
	if len(l.delimiters) > 0 {
		l.fail("CAN-LEX-CONTINUATION", "parentheses and brackets must stay on one physical line", start, l.pos)
		return
	}
	if l.significant {
		l.emit(Newline, start, l.pos, "")
	}
	l.significant = false
	l.leadingSpaces = 0
	l.lineStart = l.pos
}
func (l *lexer) invalidWhitespace(pos int) bool {
	r, size := utf8.DecodeRuneInString(l.text[pos:])
	switch {
	case r == '\t':
		l.fail("CAN-LEX-TAB", "tabs are not permitted outside string literals", pos, pos+size)
	case r == '\r' && (pos+1 == len(l.text) || l.text[pos+1] != '\n'):
		l.fail("CAN-LEX-CR", "bare carriage return outside a string literal", pos, pos+1)
	case r == '\uFEFF':
		l.fail("CAN-LEX-BOM", "a byte order mark is allowed only at the beginning of the file", pos, pos+size)
	case unicode.IsSpace(r) && r != ' ' && r != '\n' && r != '\r':
		l.fail("CAN-LEX-WHITESPACE", "only ASCII spaces and source newlines are permitted outside literals", pos, pos+size)
	default:
		return false
	}
	return true
}
func (l *lexer) next() {
	ch := l.text[l.pos]
	if l.invalidWhitespace(l.pos) {
		return
	}
	if ch == ' ' {
		if !l.significant {
			l.leadingSpaces++
		}
		l.pos++
		return
	}
	if ch == '\n' || ch == '\r' {
		l.newline()
		return
	}
	if strings.HasPrefix(l.text[l.pos:], "//") {
		l.lineComment()
		return
	}
	if strings.HasPrefix(l.text[l.pos:], "/*") {
		l.blockComment()
		return
	}
	if !l.layout() {
		return
	}
	if ch == '"' || ch == 'r' && l.pos+1 < len(l.text) && l.text[l.pos+1] == '"' {
		l.stringLiteral()
		return
	}
	if ch >= '0' && ch <= '9' {
		l.number()
		return
	}
	r, _ := utf8.DecodeRuneInString(l.text[l.pos:])
	if wordRune(r) {
		l.word()
		return
	}
	l.punctuation()
}
func wordRune(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' }
func (l *lexer) word() {
	start := l.pos
	for l.pos < len(l.text) {
		r, size := utf8.DecodeRuneInString(l.text[l.pos:])
		if !wordRune(r) {
			break
		}
		l.pos += size
	}
	text := l.text[start:l.pos]
	if text == "_" {
		l.emit(Wildcard, start, l.pos, "")
		return
	}
	if !namePattern.MatchString(text) {
		l.fail("CAN-LEX-NAME", "identifier must match lowercase Can spelling", start, l.pos)
		return
	}
	kind := Name
	if hardKeywords[text] {
		kind = Keyword
	}
	l.emit(kind, start, l.pos, "")
}
func (l *lexer) lineComment() {
	start := l.pos
	kind := LineComment
	if strings.HasPrefix(l.text[l.pos:], "///") {
		kind = DocComment
	}
	for l.pos < len(l.text) && l.text[l.pos] != '\n' && !(l.text[l.pos] == '\r' && l.pos+1 < len(l.text) && l.text[l.pos+1] == '\n') {
		if l.invalidWhitespace(l.pos) {
			return
		}
		_, size := utf8.DecodeRuneInString(l.text[l.pos:])
		l.pos += size
	}
	l.result.Comments = append(l.result.Comments, Comment{Kind: kind, Span: source.Span{Start: start, End: l.pos}, Text: l.text[start:l.pos]})
}
func (l *lexer) blockComment() {
	start := l.pos
	l.pos += 2
	depth := 1
	for l.pos < len(l.text) {
		if l.invalidWhitespace(l.pos) {
			return
		}
		switch {
		case strings.HasPrefix(l.text[l.pos:], "/*"):
			depth++
			l.pos += 2
		case strings.HasPrefix(l.text[l.pos:], "*/"):
			depth--
			l.pos += 2
			if depth == 0 {
				l.result.Comments = append(l.result.Comments, Comment{Kind: BlockComment, Span: source.Span{Start: start, End: l.pos}, Text: l.text[start:l.pos]})
				return
			}
		case l.text[l.pos] == '\n' || l.text[l.pos] == '\r':
			l.newline()
			if len(l.result.Diagnostics) > 0 {
				return
			}
		default:
			_, size := utf8.DecodeRuneInString(l.text[l.pos:])
			l.pos += size
		}
	}
	l.fail("CAN-LEX-COMMENT", "unterminated block comment", start, l.pos)
}

func (l *lexer) stringLiteral() {
	start := l.pos
	raw := false
	if l.text[l.pos] == 'r' {
		raw = true
		l.pos++
	}
	triple := strings.HasPrefix(l.text[l.pos:], `"""`)
	width := 1
	if triple {
		width = 3
	}
	l.pos += width
	delimiter := strings.Repeat(`"`, width)
	var value strings.Builder
	multiline := false
	for l.pos < len(l.text) {
		if strings.HasPrefix(l.text[l.pos:], delimiter) {
			l.pos += width
			l.emit(String, start, l.pos, value.String())
			token := &l.result.Tokens[len(l.result.Tokens)-1]
			token.Raw = raw
			token.Multiline = multiline
			return
		}
		ch := l.text[l.pos]
		if ch == '\n' || ch == '\r' && l.pos+1 < len(l.text) && l.text[l.pos+1] == '\n' {
			newlineStart := l.pos
			newlineEnd := l.pos + 1
			if ch == '\r' {
				newlineEnd++
			}
			if !triple {
				l.fail("CAN-LEX-STRING", "ordinary strings cannot contain a source newline", newlineStart, newlineEnd)
				return
			}
			if len(l.delimiters) > 0 {
				l.fail("CAN-LEX-CONTINUATION", "a multiline literal cannot span a parenthesized or bracketed form", newlineStart, newlineEnd)
				return
			}
			value.WriteByte('\n')
			l.pos = newlineEnd
			multiline = true
			continue
		}
		if ch == '\\' && !raw && l.pos+1 < len(l.text) {
			next := l.text[l.pos+1]
			escaped, known := map[byte]byte{'n': '\n', 'r': '\r', 't': '\t', '0': 0, '"': '"', '\\': '\\'}[next]
			if known {
				value.WriteByte(escaped)
				l.pos += 2
				continue
			}
			// Unknown escapes retain the backslash and leave the following scalar or
			// normalized source newline to the ordinary literal-content path.
			value.WriteByte('\\')
			l.pos++
			continue
		}
		_, size := utf8.DecodeRuneInString(l.text[l.pos:])
		value.WriteString(l.text[l.pos : l.pos+size])
		l.pos += size
	}
	l.fail("CAN-LEX-STRING", "unterminated string literal", start, l.pos)
}
func asciiDigit(ch byte) bool { return ch >= '0' && ch <= '9' }
func (l *lexer) numericSuffix() bool {
	start := l.pos
	for l.pos < len(l.text) {
		r, size := utf8.DecodeRuneInString(l.text[l.pos:])
		if !wordRune(r) {
			break
		}
		l.pos += size
	}
	return l.pos > start
}
func (l *lexer) number() {
	start := l.pos
	if l.text[l.pos] == '0' && l.pos+1 < len(l.text) && strings.ContainsRune("xbo", rune(l.text[l.pos+1])) {
		prefix := l.text[l.pos+1]
		l.pos += 2
		digits := l.pos
		l.numericSuffix()
		valid := l.pos > digits
		for _, ch := range l.text[digits:l.pos] {
			switch prefix {
			case 'x':
				valid = valid && (ch >= '0' && ch <= '9' || ch >= 'a' && ch <= 'f' || ch >= 'A' && ch <= 'F')
			case 'b':
				valid = valid && (ch == '0' || ch == '1')
			case 'o':
				valid = valid && (ch >= '0' && ch <= '7')
			}
		}
		if !valid {
			l.fail("CAN-LEX-NUMBER", "invalid base-prefixed integer literal", start, l.pos)
			return
		}
		if l.pos < len(l.text) && l.text[l.pos] == '.' && !strings.HasPrefix(l.text[l.pos:], "..") {
			l.pos++
			l.numericSuffix()
			l.fail("CAN-LEX-NUMBER", "base-prefixed floating-point literals are not supported", start, l.pos)
			return
		}
		l.emit(Integer, start, l.pos, "")
		return
	}
	for l.pos < len(l.text) && asciiDigit(l.text[l.pos]) {
		l.pos++
	}
	integerEnd := l.pos
	isFloat := false
	if l.pos < len(l.text) && l.text[l.pos] == '.' && !strings.HasPrefix(l.text[l.pos:], "..") {
		isFloat = true
		l.pos++
		if l.pos == len(l.text) || !asciiDigit(l.text[l.pos]) {
			l.fail("CAN-LEX-NUMBER", "a decimal point requires a fractional digit", start, l.pos)
			return
		}
		for l.pos < len(l.text) && asciiDigit(l.text[l.pos]) {
			l.pos++
		}
	}
	if l.pos < len(l.text) && (l.text[l.pos] == 'e' || l.text[l.pos] == 'E') {
		isFloat = true
		l.pos++
		if l.pos < len(l.text) && (l.text[l.pos] == '+' || l.text[l.pos] == '-') {
			l.pos++
		}
		if l.pos == len(l.text) || !asciiDigit(l.text[l.pos]) {
			l.numericSuffix()
			l.fail("CAN-LEX-NUMBER", "an exponent requires decimal digits", start, l.pos)
			return
		}
		for l.pos < len(l.text) && asciiDigit(l.text[l.pos]) {
			l.pos++
		}
	}
	if l.numericSuffix() {
		l.fail("CAN-LEX-NUMBER", "numeric literals have no suffixes or digit separators", start, l.pos)
		return
	}
	if !isFloat {
		if integerEnd-start > 1 && l.text[start] == '0' {
			l.fail("CAN-LEX-NUMBER", "decimal integers cannot have leading zeros", start, l.pos)
			return
		}
		l.emit(Integer, start, l.pos, "")
		return
	}
	value, err := strconv.ParseFloat(l.text[start:l.pos], 64)
	if math.IsInf(value, 0) {
		l.fail("CAN-LEX-FLOAT-RANGE", "float literal overflows binary64", start, l.pos)
		return
	}
	if err != nil {
		l.fail("CAN-LEX-NUMBER", "invalid decimal float literal", start, l.pos)
		return
	}
	l.emit(Float, start, l.pos, "")
}
func (l *lexer) punctuation() {
	start := l.pos
	remaining := l.text[start:]
	for _, invalid := range []string{"**=", "<<=", "==", "!=", "&&", "||", "+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "->"} {
		if strings.HasPrefix(remaining, invalid) {
			l.pos += len(invalid)
			l.fail("CAN-LEX-PUNCTUATION", "unsupported punctuation "+invalid, start, l.pos)
			return
		}
	}
	if remaining[0] == '.' && len(remaining) > 1 && asciiDigit(remaining[1]) {
		l.pos++
		for l.pos < len(l.text) && asciiDigit(l.text[l.pos]) {
			l.pos++
		}
		l.fail("CAN-LEX-NUMBER", "a decimal float requires digits before the point", start, l.pos)
		return
	}
	for _, punct := range []string{"...", "..", "::", "=>", "**", "<<", ">>", "<=", ">="} {
		if strings.HasPrefix(remaining, punct) {
			l.pos += len(punct)
			l.emit(Kind(punct), start, l.pos, "")
			return
		}
	}
	ch := remaining[0]
	if !strings.ContainsRune("()[],:.=+-*/%&|^~<>", rune(ch)) {
		_, size := utf8.DecodeRuneInString(remaining)
		l.pos += size
		l.fail("CAN-LEX-PUNCTUATION", "character is not part of Can syntax", start, l.pos)
		return
	}
	l.pos++
	if ch == '(' || ch == '[' {
		l.delimiters = append(l.delimiters, delimiter{kind: ch, offset: start})
	}
	if ch == ')' || ch == ']' {
		if len(l.delimiters) == 0 {
			l.fail("CAN-LEX-DELIMITER", "unmatched closing delimiter", start, l.pos)
			return
		}
		top := l.delimiters[len(l.delimiters)-1]
		if ch == ')' && top.kind != '(' || ch == ']' && top.kind != '[' {
			l.fail("CAN-LEX-DELIMITER", "mismatched closing delimiter", start, l.pos)
			return
		}
		l.delimiters = l.delimiters[:len(l.delimiters)-1]
	}
	l.emit(Kind(string(ch)), start, l.pos, "")
}
