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
	var target string
	if declaration.Array == nil {
		var err error
		target, err = e.target(declaration.Target)
		if err != nil {
			return LoweredExpression{}, err
		}
	}
	var out strings.Builder
	var captures []string
	savedTarget := e.temp()
	if declaration.Array == nil {
		fmt.Fprintf(&out, "const %s = %s;\n", savedTarget, target)
	}
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
	if e.Browser {
		parameters = append(parameters, "$canCtx: $canOwnerContext", "$canContext?: $canAssertionContext")
	} else {
		parameters = append(parameters, "$canContext?: $canAssertionContext")
	}
	var invoke string
	if declaration.Array != nil {
		var err error
		invoke, err = e.arrayInvocation(ir.InvocationStep{Array: declaration.Array, Site: declaration.Site, Span: node.Span, Result: node.Type.Result()}, arguments)
		if err != nil {
			return LoweredExpression{}, err
		}
	} else {
		if e.Browser {
			arguments = append(arguments, e.callContexts(declaration.Target)...)
		} else {
			arguments = append(arguments, "$canContext")
		}
		invoke = savedTarget + "(" + strings.Join(arguments, ", ") + ")"
	}
	name := e.temp()
	out.WriteString(e.mark(node.Span, "callable"))
	// A positively proven authored target is a native async function, so
	// the forwarding arrow returns its promise directly without a
	// redundant async adoption layer. The explicit Promise<Completion>
	// annotation, saved target/capture sequence and owner/assertion
	// context forwarding are unchanged; array, native, unclassified and
	// inconsistently rebound targets keep the async adapter.
	adapter := "async "
	if declaration.Array == nil && e.provenAuthoredBinding(declaration.Target, target) {
		adapter = ""
	}
	if e.Browser {
		fmt.Fprintf(&out, "const %s = $canOwnCallable(%s, %s, [%s], %s(%s): Promise<$canCompletion<%s>> => %s, [%s]);\n", name, quote(declaration.Site), quote(declaration.Target), strings.Join(captures, ", "), adapter, strings.Join(parameters, ", "), TypeName(node.Type.Result()), invoke, strings.Join(retained, ", "))
	} else {
		descriptor := ""
		if declaration.Array == nil {
			if companion, ok := e.provenIntegerWorker(declaration.Target, target); ok {
				if entry := e.integerWorkers[declaration.Target]; ok && entry != nil && entry.Region != nil && (len(node.Type.Inputs()) == 1 || len(node.Type.Inputs()) == 2) {
					slots := make([]string, len(declaration.Positions))
					for i, position := range declaration.Positions {
						slots[i] = strconv.Itoa(position)
					}
					span := entry.Region.Span
					descriptor = fmt.Sprintf(",{companion:%s,positions:[%s],arity:%d,origin:Object.freeze({source:%s,start:%d,end:%d,invocation:[%s]})}",
						companion, strings.Join(slots, ","), len(node.Type.Inputs()), quote(entry.Source), span.Start, span.End, quote(entry.Region.ID))
				}
			}
		}
		fmt.Fprintf(&out, "const %s = $canOwnCallable(%s, %s, [%s], %s(%s): Promise<$canCompletion<%s>> => %s, [%s],$canContext%s);\n", name, quote(declaration.Site), quote(declaration.Target), strings.Join(captures, ", "), adapter, strings.Join(parameters, ", "), TypeName(node.Type.Result()), invoke, strings.Join(retained, ", "), descriptor)
	}
	return LoweredExpression{Statements: out.String(), Value: name}, nil
}
