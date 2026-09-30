package syntax

import (
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/source"
)

func sliceOf(t *testing.T, file *source.File, span source.Span) string {
	t.Helper()
	text, err := file.Slice(span)
	if err != nil {
		t.Fatal(err)
	}
	return text
}

func TestOperatorSpansAreMinimalTokens(t *testing.T) {
	file, err := source.New("operators.can", "a + b * c")
	if err != nil {
		t.Fatal(err)
	}
	node, diagnostics := ParseExpression(file)
	if len(diagnostics) > 0 {
		t.Fatal(diagnostics[0].Format(file))
	}
	plus := node.(*BinaryExpr)
	if plus.Operator != "+" || sliceOf(t, file, plus.OperatorSpan) != "+" {
		t.Fatalf("plus span = %+v", plus.OperatorSpan)
	}
	star := plus.Right.(*BinaryExpr)
	if star.Operator != "*" || sliceOf(t, file, star.OperatorSpan) != "*" {
		t.Fatalf("star span = %+v", star.OperatorSpan)
	}

	negFile, err := source.New("unary.can", "not ready")
	if err != nil {
		t.Fatal(err)
	}
	negNode, diagnostics := ParseExpression(negFile)
	if len(diagnostics) > 0 {
		t.Fatal(diagnostics[0].Format(negFile))
	}
	neg := negNode.(*UnaryExpr)
	if neg.Operator != "not" || sliceOf(t, negFile, neg.OperatorSpan) != "not" {
		t.Fatalf("not span = %+v", neg.OperatorSpan)
	}
}

func TestComparisonOperatorSpansIncludingIsNot(t *testing.T) {
	file, err := source.New("compare.can", "a < b <= c")
	if err != nil {
		t.Fatal(err)
	}
	node, diagnostics := ParseExpression(file)
	if len(diagnostics) > 0 {
		t.Fatal(diagnostics[0].Format(file))
	}
	chain := node.(*ComparisonExpr)
	if len(chain.Operators) != 2 || len(chain.OperatorSpans) != 2 {
		t.Fatalf("chain = %+v", chain)
	}
	if sliceOf(t, file, chain.OperatorSpans[0]) != "<" || sliceOf(t, file, chain.OperatorSpans[1]) != "<=" {
		t.Fatalf("comparison spans = %+v", chain.OperatorSpans)
	}

	isFile, err := source.New("isnot.can", "x is not y")
	if err != nil {
		t.Fatal(err)
	}
	isNode, diagnostics := ParseExpression(isFile)
	if len(diagnostics) > 0 {
		t.Fatal(diagnostics[0].Format(isFile))
	}
	isChain := isNode.(*ComparisonExpr)
	if isChain.Operators[0] != "is not" || sliceOf(t, isFile, isChain.OperatorSpans[0]) != "is not" {
		t.Fatalf("is-not span = %+v", isChain.OperatorSpans)
	}
}

func TestQualifierAndMemberSpans(t *testing.T) {
	file, err := source.New("qualified.can", "pkg::thing")
	if err != nil {
		t.Fatal(err)
	}
	node, diagnostics := ParseExpression(file)
	if len(diagnostics) > 0 {
		t.Fatal(diagnostics[0].Format(file))
	}
	name := node.(*NameExpr).Name
	if name.Package != "pkg" || name.Name != "thing" {
		t.Fatalf("qualified = %+v", name)
	}
	if sliceOf(t, file, name.Span) != "pkg::thing" {
		t.Fatalf("whole span = %+v", name.Span)
	}
	if sliceOf(t, file, name.QualifierSpan) != "pkg" {
		t.Fatalf("qualifier span = %+v", name.QualifierSpan)
	}
	if sliceOf(t, file, name.MemberSpan) != "thing" {
		t.Fatalf("member span = %+v", name.MemberSpan)
	}

	soloFile, err := source.New("solo.can", "solo")
	if err != nil {
		t.Fatal(err)
	}
	soloNode, diagnostics := ParseExpression(soloFile)
	if len(diagnostics) > 0 {
		t.Fatal(diagnostics[0].Format(soloFile))
	}
	solo := soloNode.(*NameExpr).Name
	if solo.Package != "" || solo.QualifierSpan != (source.Span{}) {
		t.Fatalf("unqualified qualifier = %+v", solo)
	}
	if solo.MemberSpan != solo.Span || sliceOf(t, soloFile, solo.MemberSpan) != "solo" {
		t.Fatalf("unqualified member = %+v", solo)
	}
}

func TestCalleeMemberAndArgumentSpans(t *testing.T) {
	file, err := source.New("call.can", "call helper(seed, 2)")
	if err != nil {
		t.Fatal(err)
	}
	node, diagnostics := ParseExpression(file)
	if len(diagnostics) > 0 {
		t.Fatal(diagnostics[0].Format(file))
	}
	call := node.(*CallExpr)
	if sliceOf(t, file, call.Invocation.Callee.ExprSpan()) != "helper" {
		t.Fatalf("callee span = %+v", call.Invocation.Callee.ExprSpan())
	}
	if len(call.Invocation.Arguments) != 2 {
		t.Fatalf("arguments = %+v", call.Invocation.Arguments)
	}
	if sliceOf(t, file, call.Invocation.Arguments[0].Span) != "seed" {
		t.Fatalf("arg0 = %+v", call.Invocation.Arguments[0].Span)
	}
	if sliceOf(t, file, call.Invocation.Arguments[1].Span) != "2" {
		t.Fatalf("arg1 = %+v", call.Invocation.Arguments[1].Span)
	}

	fieldFile, err := source.New("field.can", "config.display_name")
	if err != nil {
		t.Fatal(err)
	}
	fieldNode, diagnostics := ParseExpression(fieldFile)
	if len(diagnostics) > 0 {
		t.Fatal(diagnostics[0].Format(fieldFile))
	}
	field := fieldNode.(*FieldExpr)
	if sliceOf(t, fieldFile, field.Field.Span) != "display_name" {
		t.Fatalf("member span = %+v", field.Field.Span)
	}
	if sliceOf(t, fieldFile, field.Receiver.ExprSpan()) != "config" {
		t.Fatalf("receiver span = %+v", field.Receiver.ExprSpan())
	}
}
