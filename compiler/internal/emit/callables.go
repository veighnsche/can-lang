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
		leaf := ""
		batch := ""
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
			// A proven leaf attaches its own eighth descriptor after the
			// unchanged optional integer seventh: companion/keyKind/origin
			// only, with no captures or resources. Integer-only call
			// sites keep their exact shape.
			if companion, ok := e.provenMapLeaf(declaration.Target, target); ok {
				if entry := e.mapLeaves[declaration.Target]; ok && entry != nil && entry.Region != nil && len(node.Type.Inputs()) == 2 && len(declaration.Positions) == 0 {
					span := entry.Region.Span
					leaf = fmt.Sprintf(",{companion:%s,keyKind:%s,origin:Object.freeze({source:%s,start:%d,end:%d,invocation:[%s]})}",
						companion, quote(entry.KeyKind), quote(entry.Source), span.Start, span.End, quote(entry.Region.ID))
				}
			}
			// A proven batch attaches its own ninth descriptor after the
			// unchanged optional integer seventh and leaf eighth:
			// absent/present companions, keyKind, first call-site
			// origin and the exact factory get method. Non-batch call
			// sites keep their exact shape.
			if entry, ok := e.provenMapBatch(declaration.Target, target); ok {
				if entry.Region != nil && len(node.Type.Inputs()) == 2 && len(declaration.Positions) == 0 {
					span := entry.GetSpan
					batch = fmt.Sprintf(",{absent:%s,present:%s,keyKind:%s,origin:Object.freeze({source:%s,start:%d,end:%d,invocation:[%s]}),factory:%s.get}",
						entry.Absent, entry.Present, quote(entry.KeyKind), quote(entry.Source), span.Start, span.End, quote(entry.Region.ID), entry.Receiver)
				}
			}
		}
		if leaf != "" && descriptor == "" {
			descriptor = ",undefined"
		}
		if batch != "" {
			if descriptor == "" {
				descriptor = ",undefined"
			}
			if leaf == "" {
				leaf = ",undefined"
			}
		}
		fmt.Fprintf(&out, "const %s = $canOwnCallable(%s, %s, [%s], %s(%s): Promise<$canCompletion<%s>> => %s, [%s],$canContext%s%s%s);\n", name, quote(declaration.Site), quote(declaration.Target), strings.Join(captures, ", "), adapter, strings.Join(parameters, ", "), TypeName(node.Type.Result()), invoke, strings.Join(retained, ", "), descriptor, leaf, batch)
	}
	return LoweredExpression{Statements: out.String(), Value: name}, nil
}
