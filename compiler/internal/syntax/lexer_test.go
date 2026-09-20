package syntax

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/veighnsche/can-lang/compiler/internal/source"
)

func lexText(t *testing.T, text string) Result {
	t.Helper()
	file, err := source.New("test.can", text)
	if err != nil {
		t.Fatal(err)
	}
	return Lex(file)
}
func valid(t *testing.T, text string) Result {
	t.Helper()
	result := lexText(t, text)
	if !result.OK() {
		t.Fatal(result.Diagnostics[0].Format(result.File))
	}
	return result
}
func significantTokens(result Result) []Token {
	out := []Token{}
	for _, token := range result.Tokens {
		if token.Kind != Newline && token.Kind != Indent && token.Kind != Dedent && token.Kind != EOF {
			out = append(out, token)
		}
	}
	return out
}
func TestIndentationAndCommentOnlyLines(t *testing.T) {
	input := "package demo\r\n    provides []\r\n    // comment-only line\r\n          /* arbitrary comment-only indentation */\r\n    uses []\r\n\r\nfn void main\r\n    emits []\r\n    asserts\r\n        () => ok\r\n    ok"
	result := valid(t, input)
	layouts := []Kind{}
	for _, token := range result.Tokens {
		if token.Kind == Newline || token.Kind == Indent || token.Kind == Dedent || token.Kind == EOF {
			layouts = append(layouts, token.Kind)
		}
	}
	expected := []Kind{Newline, Indent, Newline, Newline, Dedent, Newline, Indent, Newline, Newline, Indent, Newline, Dedent, Newline, Dedent, EOF}
	if !reflect.DeepEqual(layouts, expected) {
		t.Fatalf("layout: %v", layouts)
	}
	if len(result.Comments) != 2 {
		t.Fatal("comment trivia lost")
	}
	// An empty or comment-only indentation never produces an empty lexical block;
	// the parser will diagnose a missing required INDENT for a block-owning form.
	for _, text := range []string{"", "\n   // empty\n", "/* empty */\n"} {
		for _, token := range valid(t, text).Tokens {
			if token.Kind != EOF {
				t.Fatalf("significant empty-file token %+v", token)
			}
		}
	}
}
func TestCommentSeparationNestingAndDocumentation(t *testing.T) {
	result := valid(t, "/// docs\nrecord/* outer /* nested */ { ; } */item\n    str name // body\n/* line\n boundary */\nrecord next")
	tokens := significantTokens(result)
	words := []string{}
	for _, token := range tokens {
		words = append(words, token.Text)
	}
	if !reflect.DeepEqual(words, []string{"record", "item", "str", "name", "record", "next"}) {
		t.Fatal(words)
	}
	if len(result.Comments) != 4 || result.Comments[0].Kind != DocComment || result.Comments[1].Kind != BlockComment {
		t.Fatal(result.Comments)
	}
	if result.Comments[1].Text != "/* outer /* nested */ { ; } */" {
		t.Fatal("nested comment span changed")
	}
	result = valid(t, "a /* line\nboundary */b")
	names := significantTokens(result)
	first, _ := result.File.Position(names[0].Span.Start)
	second, _ := result.File.Position(names[1].Span.Start)
	if first.Line != 1 || second.Line != 2 {
		t.Fatal("comment merged source lines")
	}
}
func TestKeywordsAndContextualNames(t *testing.T) {
	for word := range hardKeywords {
		tokens := significantTokens(valid(t, word))
		if len(tokens) != 1 || tokens[0].Kind != Keyword {
			t.Fatalf("hard keyword %s", word)
		}
	}
	for word := range contextualWords {
		tokens := significantTokens(valid(t, word))
		if len(tokens) != 1 || tokens[0].Kind != Name || !IsContextual(word) {
			t.Fatalf("contextual name %s", word)
		}
	}
	tokens := significantTokens(valid(t, "int score\nint minimum\nchoice_arm then _"))
	if tokens[1].Kind != Name || tokens[3].Kind != Name || tokens[4].Kind != Name || tokens[5].Kind != Name || tokens[6].Kind != Wildcard {
		t.Fatal(tokens)
	}
}
func TestStringDecodingAndOriginalSpans(t *testing.T) {
	cases := []struct {
		literal, value string
		raw, multiline bool
	}{
		{`"line\n\r\t\0\"\\"`, "line\n\r\t\x00\"\\", false, false},
		{`"\q\u1234\x41\a"`, `\q\u1234\x41\a`, false, false},
		{`r"C:\files\notes\"`, `C:\files\notes\`, true, false},
		{"\"\"\"\r\n    é😀\r\n\"\"\"", "\n    é😀\n", false, true},
		{"r\"\"\"\n  \\\"\n\"\"\"", "\n  \\\"\n", true, true},
		{"\"\"\"one \" two \"\" end\"\"\"", "one \" two \"\" end", false, false},
		{"\"bare\rreturn\tinside\"", "bare\rreturn\tinside", false, false},
		{"\"\uFEFF\"", "\uFEFF", false, false},
	}
	for _, tc := range cases {
		t.Run(tc.literal, func(t *testing.T) {
			result := valid(t, tc.literal)
			tokens := significantTokens(result)
			if len(tokens) != 1 || tokens[0].Kind != String || tokens[0].Value != tc.value || tokens[0].Text != tc.literal || tokens[0].Raw != tc.raw || tokens[0].Multiline != tc.multiline {
				t.Fatalf("decoded %+v; want %q", tokens, tc.value)
			}
			if slice, err := result.File.Slice(tokens[0].Span); err != nil || slice != tc.literal {
				t.Fatalf("original span %q %v", slice, err)
			}
		})
	}
	result := valid(t, `r"before" + "\"" + r"after"`)
	if len(significantTokens(result)) != 5 {
		t.Fatal("raw delimiters treated as quote escapes")
	}
}
func TestNumericLiteralsRangesAndSigns(t *testing.T) {
	input := "0 42 0xff 0XFF 0b101 0o17"
	bad := lexText(t, input)
	if bad.OK() {
		t.Fatal("uppercase prefix accepted")
	}
	input = "0 42 0xfF 0b101 0o17 1.0 1e3 1.2E-3 00.1 00e2 0e9999999999 1e-9999999 1..5 -2 ** -2"
	tokens := significantTokens(valid(t, input))
	want := []Kind{Integer, Integer, Integer, Integer, Integer, Float, Float, Float, Float, Float, Float, Float, Integer, "..", Integer, "-", Integer, "**", "-", Integer}
	got := []Kind{}
	for _, token := range tokens {
		got = append(got, token.Kind)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatal(got)
	}
}
func TestLexicalFailuresHavePreciseSpans(t *testing.T) {
	cases := []struct{ text, code, fragment string }{
		{"    first", "CAN-LEX-INDENT", "    "}, {"a\n  b", "CAN-LEX-INDENT", "  "}, {"a\n        b", "CAN-LEX-INDENT", "        "},
		{"a\n    b\n      c", "CAN-LEX-INDENT", "      "}, {"a\t b", "CAN-LEX-TAB", "\t"}, {"// tab\t", "CAN-LEX-TAB", "\t"},
		{"a\rb", "CAN-LEX-CR", "\r"}, {"/* x\r */", "CAN-LEX-CR", "\r"}, {"a\u00a0b", "CAN-LEX-WHITESPACE", "\u00a0"}, {"a\uFEFFb", "CAN-LEX-BOM", "\uFEFF"},
		{"Bad", "CAN-LEX-NAME", "Bad"}, {"bad__name", "CAN-LEX-NAME", "bad__name"}, {"bad_", "CAN-LEX-NAME", "bad_"}, {"_bad", "CAN-LEX-NAME", "_bad"}, {"café", "CAN-LEX-NAME", "café"},
		{"01", "CAN-LEX-NUMBER", "01"}, {"0x", "CAN-LEX-NUMBER", "0x"}, {"0b102", "CAN-LEX-NUMBER", "0b102"}, {"0o8", "CAN-LEX-NUMBER", "0o8"}, {"0x1.2", "CAN-LEX-NUMBER", "0x1.2"},
		{"1_000", "CAN-LEX-NUMBER", "1_000"}, {"1e+", "CAN-LEX-NUMBER", "1e+"}, {".5", "CAN-LEX-NUMBER", ".5"}, {"1.", "CAN-LEX-NUMBER", "1."}, {"1e9999", "CAN-LEX-FLOAT-RANGE", "1e9999"},
		{"\"x\ny\"", "CAN-LEX-STRING", "\n"}, {"\"unclosed", "CAN-LEX-STRING", "\"unclosed"}, {"r\"\"\"unclosed", "CAN-LEX-STRING", "r\"\"\"unclosed"}, {"/* unclosed /* */", "CAN-LEX-COMMENT", "/* unclosed /* */"},
		{"call f(\n)", "CAN-LEX-CONTINUATION", "\n"}, {"[r\"\"\"x\ny\"\"\"]", "CAN-LEX-CONTINUATION", "\n"}, {"(x /*\n*/)", "CAN-LEX-CONTINUATION", "\n"},
		{"(]", "CAN-LEX-DELIMITER", "]"}, {"(", "CAN-LEX-DELIMITER", "("}, {")", "CAN-LEX-DELIMITER", ")"},
	}
	for _, punct := range []string{";", "{", "}", "'", "\\", "==", "!=", "&&", "||", "+=", "->", "@", "?"} {
		cases = append(cases, struct{ text, code, fragment string }{punct, "CAN-LEX-PUNCTUATION", punct})
	}
	for _, tc := range cases {
		t.Run(tc.text, func(t *testing.T) {
			result := lexText(t, tc.text)
			if len(result.Diagnostics) != 1 {
				t.Fatalf("expected one refusal, got %+v", result.Diagnostics)
			}
			d := result.Diagnostics[0]
			fragment, err := result.File.Slice(d.Span)
			if d.Code != tc.code || fragment != tc.fragment || err != nil {
				t.Fatalf("diagnostic %+v span %q error %v", d, fragment, err)
			}
		})
	}
}
func TestLFAndCRLFHaveIdenticalLogicalTokens(t *testing.T) {
	lf := "\uFEFFpackage demo\n    provides []\n    uses []\nstr text = \"\"\"\n  é😀\n\"\"\"\n"
	left := valid(t, lf)
	right := valid(t, strings.ReplaceAll(lf, "\n", "\r\n"))
	if len(left.Tokens) != len(right.Tokens) {
		t.Fatal("newline normalization changed token count")
	}
	for i, a := range left.Tokens {
		b := right.Tokens[i]
		if a.Kind != b.Kind || a.Value != b.Value {
			t.Fatalf("token %d differs", i)
		}
		ap, _ := left.File.UTF16Position(a.Span.Start)
		bp, _ := right.File.UTF16Position(b.Span.Start)
		if ap != bp {
			t.Fatalf("token %d moved: %+v vs %+v", i, ap, bp)
		}
	}
}
func TestCurrentFixturesAndSharedPositions(t *testing.T) {
	paths, err := filepath.Glob("../../testdata/current/lexer/*.can")
	if err != nil || len(paths) == 0 {
		t.Fatal("missing current fixtures")
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			result := valid(t, string(data))
			for _, token := range result.Tokens {
				for _, offset := range []int{token.Span.Start, token.Span.End} {
					editor, err := result.File.UTF16Position(offset)
					if err != nil {
						t.Fatal(err)
					}
					mapping, err := result.File.MapPosition(offset)
					if err != nil || mapping.Line != editor.Line+1 || mapping.Column != editor.Character {
						t.Fatal("map/editor disagreement")
					}
					back, err := result.File.Offset(editor)
					if err != nil || back != offset {
						t.Fatal("token span did not round-trip")
					}
				}
			}
		})
	}
}
func FuzzLexerTerminatesWithOriginalSpans(f *testing.F) {
	for _, text := range []string{"", "a\n    b", `r"x\"`, "/* nested /* */ */", "é😀", "1e9999", "\"\"\"x\ny\"\"\""} {
		f.Add(text)
	}
	f.Fuzz(func(t *testing.T, text string) {
		if len(text) > 8192 || !utf8.ValidString(text) {
			t.Skip()
		}
		file, err := source.New("fuzz.can", text)
		if err != nil {
			t.Fatal(err)
		}
		result := Lex(file)
		if len(result.Tokens) == 0 || result.Tokens[len(result.Tokens)-1].Kind != EOF {
			t.Fatal("missing EOF")
		}
		for _, token := range result.Tokens {
			if err := file.Validate(token.Span); err != nil {
				t.Fatalf("invalid token %+v: %v", token, err)
			}
		}
		for _, d := range result.Diagnostics {
			if err := file.Validate(d.Span); err != nil {
				t.Fatalf("invalid diagnostic %+v: %v", d, err)
			}
		}
	})
}

func TestNestedGenericClosersRemainLexable(t *testing.T) {
	result := valid(t, "fn option::value<option::value<option::value<int>>> nested")
	shifts, greater := 0, 0
	for _, token := range result.Tokens {
		if token.Kind == ">>" {
			shifts++
		}
		if token.Kind == ">" {
			greater++
		}
		if token.Kind == ">>>" {
			t.Fatal("unsigned shift token introduced")
		}
	}
	if shifts != 1 || greater != 1 {
		t.Fatal("nested type closers lost")
	}
	// I04's type parser splits right-angle operator tokens only while consuming
	// generic arguments. An expression parser never receives an unsigned-shift token.
}

func TestDiagnosticSharesOriginalUTF16Location(t *testing.T) {
	result := lexText(t, "str x = \"😀\";")
	if result.OK() {
		t.Fatal("semicolon accepted")
	}
	diagnostic := result.Diagnostics[0]
	if diagnostic.Span.Start != 14 || diagnostic.Format(result.File) != "test.can:1:12: CAN-LEX-PUNCTUATION: character is not part of Can syntax" {
		t.Fatalf("diagnostic %+v: %s", diagnostic, diagnostic.Format(result.File))
	}
	editor, err := result.File.UTF16Position(diagnostic.Span.Start)
	if err != nil {
		t.Fatal(err)
	}
	mapping, err := result.File.MapPosition(diagnostic.Span.Start)
	if err != nil {
		t.Fatal(err)
	}
	if editor != (source.UTF16Position{Line: 0, Character: 12}) || mapping != (source.MapPosition{Line: 1, Column: 12}) {
		t.Fatalf("inconsistent UTF-16: %+v %+v", editor, mapping)
	}
	result = lexText(t, "/*😀*/   x")
	if result.OK() {
		t.Fatal("invalid indentation accepted")
	}
	if err := result.File.Validate(result.Diagnostics[0].Span); err != nil {
		t.Fatal("comment prefix split a scalar", err)
	}
}
