// Package syntax owns the current C2 lexer, parser and explicit syntax nodes.
// It does not import the predecessor compiler's line/brace scanners.
package syntax

import (
	"fmt"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/source"
)

type Kind string

const (
	Name     Kind = "name"
	Keyword  Kind = "keyword"
	Wildcard Kind = "_"
	Integer  Kind = "integer"
	Float    Kind = "float"
	String   Kind = "string"
	Newline  Kind = "newline"
	Indent   Kind = "indent"
	Dedent   Kind = "dedent"
	EOF      Kind = "eof"
)

// Punctuation kinds use their exact source spelling (for example Kind("=>")).
// Contextual words remain Name tokens until the owning grammar recognizes them.
type Token struct {
	Kind      Kind
	Span      source.Span
	Text      string // exact original source bytes, including CRLF inside a literal
	Value     string // decoded String content; empty for other token kinds
	Raw       bool
	Multiline bool
}

func (t Token) Is(kind Kind) bool { return t.Kind == kind }
func (t Token) IsWord(word string) bool {
	return (t.Kind == Name || t.Kind == Keyword) && t.Text == word
}

type CommentKind string

const (
	LineComment  CommentKind = "line"
	DocComment   CommentKind = "documentation"
	BlockComment CommentKind = "block"
)

type Comment struct {
	Kind CommentKind
	Span source.Span
	Text string
}
type Diagnostic struct {
	Code    string
	Message string
	Span    source.Span
}

func (d Diagnostic) Format(file *source.File) string {
	p, err := file.Position(d.Span.Start)
	if err != nil {
		return fmt.Sprintf("%s: %s: %s", file.Name(), d.Code, d.Message)
	}
	return fmt.Sprintf("%s:%d:%d: %s: %s", file.Name(), p.Line, p.Column, d.Code, d.Message)
}

type Result struct {
	File        *source.File
	Tokens      []Token
	Comments    []Comment
	Diagnostics []Diagnostic
}

func (r Result) OK() bool { return len(r.Diagnostics) == 0 }

var hardKeywords = wordSet("package provides uses as fn record variant error given near emits asserts call callable match chain do ok relay on with and or not is true false void int float bool str")
var contextualWords = wordSet("connection noul choice score judge choice_arm fetch llm from state asks minimum confidence describes endpoint auth bearer env timeout_ms metadata query headers body get post put patch delete head options concurrent race when fixture for use cases bind owner scenario link action handles captures result input none json form limit rows_limit returns status swap inner document")

func wordSet(words string) map[string]bool {
	out := map[string]bool{}
	for _, s := range strings.Fields(words) {
		out[s] = true
	}
	return out
}
func IsContextual(word string) bool  { return contextualWords[word] }
func IsHardKeyword(word string) bool { return hardKeywords[word] }
