package syntax

import (
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/source"
)

func TestNestedParenthesizedArgumentsParseOnce(t *testing.T) {
	text := strings.Repeat("call f((", 100) + "1" + strings.Repeat("))", 100)
	parsed := expressionFragment(t, text)
	if FormatExpression(parsed) != text {
		t.Fatal("nested argument grouping changed")
	}
	for _, text := range []string{"call f((a)+b)", "call f((a).field)", "call f((a)[0])", "call f((a),b)", "call f(())", "call f((a,b))"} {
		expressionFragment(t, text)
	}
}

func expressionFragment(t *testing.T, text string) Expr {
	t.Helper()
	file, err := source.New("expression.can", text)
	if err != nil {
		t.Fatal(err)
	}
	node, diagnostics := ParseExpression(file)
	if len(diagnostics) > 0 {
		t.Fatal(diagnostics[0].Format(file))
	}
	if err := file.Validate(node.ExprSpan()); err != nil {
		t.Fatal(err)
	}
	return node
}

func TestExpressionPrecedence(t *testing.T) {
	minus := expressionFragment(t, "-2 ** 2").(*UnaryExpr)
	if minus.Operator != "-" || minus.Operand.(*BinaryExpr).Operator != "**" {
		t.Fatal(minus)
	}
	power := expressionFragment(t, "2 ** -2 ** 3").(*BinaryExpr)
	if power.Operator != "**" || power.Right.(*UnaryExpr).Operand.(*BinaryExpr).Operator != "**" {
		t.Fatal(power)
	}
	or := expressionFragment(t, "not a < b <= c and d or e").(*BinaryExpr)
	and := or.Left.(*BinaryExpr)
	not := and.Left.(*UnaryExpr)
	chain := not.Operand.(*ComparisonExpr)
	if or.Operator != "or" || and.Operator != "and" || not.Operator != "not" || len(chain.Operands) != 3 || chain.Operators[1] != "<=" {
		t.Fatal(or)
	}
	bits := expressionFragment(t, "a | b ^ c & d << e + f * g").(*BinaryExpr)
	for _, op := range []string{"|", "^", "&", "<<", "+", "*"} {
		if bits.Operator != op {
			t.Fatalf("wanted %s, got %s", op, bits.Operator)
		}
		if op != "*" {
			bits = bits.Right.(*BinaryExpr)
		}
	}
	if expressionFragment(t, "a is not b").(*ComparisonExpr).Operators[0] != "is not" {
		t.Fatal("lost is not")
	}
}

func TestExplicitCallsAndConstructors(t *testing.T) {
	call := expressionFragment(t, "call panel.resize(5, 4).area()").(*CallExpr)
	if call.Invocation.Callee.(*FieldExpr).Field.Text != "resize" || len(call.Methods) != 1 || call.Methods[0].Name.Text != "area" {
		t.Fatal(call)
	}
	mapping := expressionFragment(t, "call [1, 2, 3].map(callable transform<int>)").(*CallExpr)
	ref := mapping.Invocation.Arguments[0].Value.(*ReferenceExpr)
	if FormatType(ref.Types[0]) != "int" {
		t.Fatal(ref)
	}
	ctor := expressionFragment(t, "option::some<outer<inner<int>>>(value)").(*ConstructorExpr)
	if ctor.Name.Package != "option" || FormatType(ctor.Types[0]) != "outer<inner<int>>" {
		t.Fatal(ctor)
	}
	comparison := expressionFragment(t, "left < middle > right").(*ComparisonExpr)
	if len(comparison.Operands) != 3 {
		t.Fatal(comparison)
	}
	field := expressionFragment(t, "call produce().value").(*FieldExpr)
	if _, ok := field.Receiver.(*CallExpr); !ok {
		t.Fatal(field)
	}
}

func TestArraysSlicesAndUpdates(t *testing.T) {
	array := expressionFragment(t, "[...scores, 40, ...bonus_scores]").(*ArrayExpr)
	if len(array.Elements) != 3 || !array.Elements[0].Spread || array.Elements[1].Spread || !array.Elements[2].Spread {
		t.Fatal(array)
	}
	for _, text := range []string{"scores[start:end]", "scores[:end]", "scores[start:]", "scores[:]"} {
		if _, ok := expressionFragment(t, text).(*SliceExpr); !ok {
			t.Fatal(text)
		}
	}
	update := expressionFragment(t, "shape with (width = shape.width + amount, height = 2)").(*UpdateExpr)
	if len(update.Fields) != 2 || update.Fields[0].Name.Text != "width" {
		t.Fatal(update)
	}
	if _, ok := expressionFragment(t, "(shape with width = 2) with height = 3").(*UpdateExpr); !ok {
		t.Fatal("grouped update")
	}
}

func TestExpressionsRejectMalformedForms(t *testing.T) {
	for _, text := range []string{
		"call f(1,)", "record_name(1,)", "[1,]", "call f(,1)", "call f(1\n)",
		"call f", "a >>> b", "a >>", "a[]", "a[::]", "(a, b)",
		"shape with x = 1, y = 2", "shape with (x = 1)", "shape with (x = 1, y = 2,)",
		"shape with x = 1 with y = 2", "shape with x =", "%", "ok 1",
		"call f((a,b),c)", "call f((a,))", "call f(...())",
	} {
		file, err := source.New("bad.can", text)
		if err != nil {
			t.Fatal(err)
		}
		node, diagnostics := ParseExpression(file)
		if node != nil || len(diagnostics) == 0 {
			t.Fatalf("accepted %q", text)
		}
		for _, d := range diagnostics {
			if err := file.Validate(d.Span); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func FuzzExpressionParser(f *testing.F) {
	for _, text := range []string{"-2 ** 2", "call f(1)", "a < b > c", "shape with x = 1", "[...values]"} {
		f.Add(text)
	}
	f.Fuzz(func(t *testing.T, text string) {
		file, err := source.New("fuzz.can", text)
		if err != nil {
			return
		}
		node, diagnostics := ParseExpression(file)
		if node != nil {
			if err := file.Validate(node.ExprSpan()); err != nil {
				t.Fatal(err)
			}
		}
		for _, d := range diagnostics {
			if err := file.Validate(d.Span); err != nil {
				t.Fatal(err)
			}
		}
	})
}
