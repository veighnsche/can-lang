package syntax

import (
	"reflect"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/source"
)

func typeFragment(t *testing.T, text string) TypeNode {
	t.Helper()
	file, err := source.New("type.can", text)
	if err != nil {
		t.Fatal(err)
	}
	node, diagnostics := ParseType(file)
	if len(diagnostics) != 0 {
		t.Fatal(diagnostics[0].Format(file))
	}
	if err := file.Validate(node.TypeSpan()); err != nil {
		t.Fatal(err)
	}
	return node
}

func TestTypeGrammarRoundTrip(t *testing.T) {
	for _, text := range []string{
		"int", "str[][]", "option::value<item>", "outer<middle<inner<int>>>",
		"pair<int[], option::value<str>>[]",
		"callable int[] () emits []",
		"callable int () emits [][]",
		"callable callable int (str) emits [failed] (bool) emits [other<int>]",
		"choice_arm<option::value<str>> emits [failed][]",
		"callable choice_arm<int> emits [] (callable str () emits []) emits []",
	} {
		t.Run(text, func(t *testing.T) {
			node := typeFragment(t, text)
			formatted := FormatType(node)
			if formatted != text {
				t.Fatalf("rendered %q, expected %q", formatted, text)
			}
			if again := FormatType(typeFragment(t, formatted)); again != formatted {
				t.Fatal(again)
			}
		})
	}
}

func TestCallableArrayOwnership(t *testing.T) {
	resultArray := typeFragment(t, "callable int[] () emits []").(*CallableType)
	if _, ok := resultArray.Result.(*ArrayType); !ok {
		t.Fatal("result array lost")
	}
	callableArray := typeFragment(t, "callable int () emits [][]").(*ArrayType)
	callable := callableArray.Element.(*CallableType)
	if _, ok := callable.Result.(*NamedType); !ok {
		t.Fatal("array attached to result")
	}
	if callable.Errors.Span.Start != 16 || callable.Errors.Span.End != 24 {
		t.Fatalf("error bound span: %+v", callable.Errors.Span)
	}
}

func TestTypeGrammarRejectsMalformedFragments(t *testing.T) {
	for _, text := range []string{
		"", "[]", "(int)", "item<>", "item<int,>", "int<str>",
		"callable int ()", "callable int (str,) emits []", "callable int () emits [bad,]",
		"choice_arm<int>", "choice_arm<int, str> emits []", "item<int>=x",
		"int[3]", "item<int", "item<int>>", "int str", "int\nstr",
		"callable int (\nstr) emits []", strings.Repeat("item<", 257) + "int" + strings.Repeat(">", 257),
	} {
		t.Run(text[:minLength(len(text), 60)], func(t *testing.T) {
			file, err := source.New("bad.can", text)
			if err != nil {
				t.Fatal(err)
			}
			node, diagnostics := ParseType(file)
			if node != nil || len(diagnostics) != 1 {
				t.Fatalf("accepted malformed type: %q", text)
			}
			if err := file.Validate(diagnostics[0].Span); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func minLength(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestGenericClosersPreserveLexerResult(t *testing.T) {
	for _, text := range []string{"outer<inner<int>>", "outer<int>=value"} {
		file, err := source.New("closers.can", text)
		if err != nil {
			t.Fatal(err)
		}
		result := Lex(file)
		before := append([]Token(nil), result.Tokens...)
		p := newParser(result)
		node := p.parseType()
		if !reflect.DeepEqual(before, result.Tokens) {
			t.Fatal("parser mutated shared lexer tokens")
		}
		if strings.Contains(text, "=") {
			if !p.at("=") || p.peek().Span.Start != len("outer<int>") {
				t.Fatal(p.peek())
			}
		}
		if text[node.TypeSpan().End-1] != '>' {
			t.Fatal(node.TypeSpan())
		}
	}
}

func FuzzTypeParser(f *testing.F) {
	for _, text := range []string{"int", "outer<inner<str>>", "callable int[] () emits []", "choice_arm<int> emits []"} {
		f.Add(text)
	}
	f.Fuzz(func(t *testing.T, text string) {
		file, err := source.New("fuzz.can", text)
		if err != nil {
			return
		}
		node, diagnostics := ParseType(file)
		for _, d := range diagnostics {
			if err := file.Validate(d.Span); err != nil {
				t.Fatal(err)
			}
		}
		if node == nil {
			return
		}
		canonical := FormatType(node)
		if again := FormatType(typeFragment(t, canonical)); again != canonical {
			t.Fatal("unstable render")
		}
	})
}
