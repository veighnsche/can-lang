package emit

import (
	"fmt"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/types"
	"slices"
	"strconv"
	"strings"
)

func (e *RegionEmitter) callable(node *ir.Expression) (LoweredExpression, error) {
	declaration := node.Callable
	if declaration == nil || declaration.Site == "" || !types.Equal(declaration.Contract, declaration.Contract) || declaration.Contract.Kind() != types.Callable || node.Type.Kind() != types.Callable || len(declaration.Positions) != len(node.Inputs) {
		return LoweredExpression{}, fmt.Errorf("incomplete checked callable")
	}
	if !slices.Equal(declaration.ResourceCaptures, ir.ResourceCaptureIndices(node.Inputs)) {
		return LoweredExpression{}, fmt.Errorf("invalid checked resource capture evidence")
	}
	retained := make([]string, len(declaration.ResourceCaptures))
	for i, index := range declaration.ResourceCaptures {
		retained[i] = strconv.Itoa(index)
	}
	target, err := e.target(declaration.Target)
	if err != nil {
		return LoweredExpression{}, err
	}
	var out strings.Builder
	out.WriteString(e.mark(node.Span, "callable"))
	var captures []string
	savedTarget := e.temp()
	fmt.Fprintf(&out, "const %s = %s;\n", savedTarget, target)
	full := declaration.Contract.Inputs()
	arguments := make([]string, len(full))
	previous := -1
	for i, position := range declaration.Positions {
		if position <= previous || position >= len(full) || !types.Equal(node.Inputs[i].Type, full[position]) {
			return LoweredExpression{}, fmt.Errorf("invalid callable capture slot")
		}
		previous = position
		captured, err := e.expression.Lower(node.Inputs[i])
		if err != nil {
			return LoweredExpression{}, err
		}
		out.WriteString(captured.Statements)
		arguments[position] = captured.Value
		captures = append(captures, captured.Value)
	}
	var parameters []string
	next := 0
	for i := range full {
		if arguments[i] != "" {
			continue
		}
		if next >= len(node.Type.Inputs()) || !types.Equal(node.Type.Inputs()[next], full[i]) {
			return LoweredExpression{}, fmt.Errorf("invalid residual callable contract")
		}
		name := fmt.Sprintf("$canCallableArg%d", next)
		next++
		parameters = append(parameters, name+": "+TypeName(full[i]))
		arguments[i] = name
	}
	if next != len(node.Type.Inputs()) || !types.Equal(node.Type.Result(), declaration.Contract.Result()) {
		return LoweredExpression{}, fmt.Errorf("invalid residual callable result/arity")
	}
	if len(node.Type.Errors()) != len(declaration.Contract.Errors()) {
		return LoweredExpression{}, fmt.Errorf("invalid callable bound")
	}
	for i, errType := range node.Type.Errors() {
		if !types.Equal(errType, declaration.Contract.Errors()[i]) {
			return LoweredExpression{}, fmt.Errorf("invalid callable bound")
		}
	}
	parameters = append(parameters, "$canContext?: $canAssertionContext")
	arguments = append(arguments, "$canContext")
	name := e.temp()
	out.WriteString(e.mark(node.Span, "callable"))
	fmt.Fprintf(&out, "const %s = $canOwnCallable(%s, %s, [%s], async (%s): Promise<$canCompletion<%s>> => %s(%s), [%s]);\n", name, quote(declaration.Site), quote(declaration.Target), strings.Join(captures, ", "), strings.Join(parameters, ", "), TypeName(node.Type.Result()), savedTarget, strings.Join(arguments, ", "), strings.Join(retained, ", "))
	return LoweredExpression{Statements: out.String(), Value: name}, nil
}
