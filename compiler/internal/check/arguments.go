package check

import (
	"fmt"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
	"strconv"
)

func literalSpreadLength(node syntax.Expr) (int, bool) {
	if group, ok := node.(*syntax.GroupExpr); ok {
		return literalSpreadLength(group.Value)
	}
	array, ok := node.(*syntax.ArrayExpr)
	if !ok {
		return 0, false
	}
	count := 0
	for _, arg := range array.Elements {
		if arg.Group != nil {
			return 0, false
		}
		if arg.Spread {
			n, known := literalSpreadLength(arg.Value)
			if !known {
				return 0, false
			}
			count += n
		} else {
			count++
		}
	}
	return count, true
}

// All authored arguments are checked and evaluated once, in source order. A
// spread entering fixed inputs needs a syntactically known literal length; only
// the final variadic array admits dynamic length. Splitting a literal spread
// across that boundary references one prepared immutable array, never reevaluates.
func (c *regionChecker) arguments(e *Expressions, callee ValueBinding, args []syntax.Argument, receiver *ir.Expression) ([]ir.Preparation, []*ir.Expression, error) {
	inputs := callee.Type.Inputs()
	descriptor := c.context.Callables[callee.Identity]
	var state []syntax.Expr
	var stateTypes []*types.Type
	if descriptor.Grouped {
		if descriptor.State < 0 || descriptor.State > len(inputs) || len(args) == 0 {
			return nil, nil, fmt.Errorf("native invocation requires its final state argument group")
		}
		last := args[len(args)-1]
		if last.Spread {
			return nil, nil, fmt.Errorf("state group cannot be spread")
		}
		if last.Group != nil {
			state = last.Group.Values
		} else if group, ok := last.Value.(*syntax.GroupExpr); ok {
			state = []syntax.Expr{group.Value}
		} else {
			return nil, nil, fmt.Errorf("native invocation requires its final state argument group")
		}
		if len(state) != descriptor.State {
			return nil, nil, fmt.Errorf("native state group arity mismatch")
		}
		args = args[:len(args)-1]
		stateTypes = inputs[len(inputs)-descriptor.State:]
		inputs = inputs[:len(inputs)-descriptor.State]
	}
	fixed := len(inputs)
	variadic := c.context.Variadic[callee.Identity]
	var tail *ir.Expression
	if variadic {
		if fixed == 0 || inputs[fixed-1].Kind() != types.Array {
			return nil, nil, fmt.Errorf("variadic contract requires final array input")
		}
		fixed--
		tail = &ir.Expression{Kind: ir.Array, Type: inputs[fixed]}
	}
	var prepared []ir.Preparation
	var values []*ir.Expression
	save := func(value *ir.Expression) *ir.Expression {
		local := ir.Local{Identity: c.identity("argument"), Type: value.Type}
		prepared = append(prepared, ir.Preparation{Local: local, Value: value})
		return &ir.Expression{Kind: ir.Binding, Span: value.Span, Type: value.Type, Text: local.Identity}
	}
	if receiver != nil {
		if fixed == 0 || !types.Assignable(receiver.Type, inputs[0]) {
			return nil, nil, fmt.Errorf("method receiver contract mismatch")
		}
		values = append(values, save(receiver))
	}
	integer, err := e.scalar("int")
	if err != nil {
		return nil, nil, err
	}
	index := func(array *ir.Expression, i int) *ir.Expression {
		return &ir.Expression{Kind: ir.Index, Span: array.Span, Type: array.Type.Element(), Inputs: []*ir.Expression{array, {Kind: ir.Literal, Span: array.Span, Type: integer, Text: strconv.Itoa(i)}}}
	}
	for _, arg := range args {
		if arg.Group != nil {
			return nil, nil, fmt.Errorf("state argument group requires its native declaration")
		}
		if !arg.Spread {
			var want *types.Type
			if len(values) < fixed {
				want = inputs[len(values)]
			} else if tail != nil {
				want = tail.Type.Element()
			} else {
				return nil, nil, fmt.Errorf("call arity mismatch")
			}
			value, err := e.Check(arg.Value, want)
			if err != nil {
				return nil, nil, err
			}
			value = save(value)
			if len(values) < fixed {
				values = append(values, value)
			} else {
				tail.Inputs = append(tail.Inputs, value)
				tail.Spread = append(tail.Spread, false)
			}
			continue
		}
		length, known := literalSpreadLength(arg.Value)
		if len(values) < fixed && !known {
			return nil, nil, fmt.Errorf("runtime-length spread cannot supply fixed inputs")
		}
		if !variadic && (!known || len(values)+length > fixed) {
			return nil, nil, fmt.Errorf("spread arity mismatch")
		}
		var element *types.Type
		if len(values) < fixed {
			element = inputs[len(values)]
		} else if tail != nil {
			element = tail.Type.Element()
		} else if known && length == 0 {
			element = integer
		}
		if element == nil {
			return nil, nil, fmt.Errorf("spread has no expected element contract")
		}
		expected, err := types.ArrayOfChecked(element)
		if err != nil {
			return nil, nil, err
		}
		value, err := e.Check(arg.Value, nil)
		if err != nil && known {
			value, err = e.Check(arg.Value, expected)
		}
		if err == nil && value.Type.Kind() != types.Array {
			err = fmt.Errorf("spread requires an array")
		}
		if err != nil {
			return nil, nil, err
		}
		value = save(value)
		consumed := 0
		if known {
			for consumed < length && len(values) < fixed {
				if !types.Assignable(value.Type.Element(), inputs[len(values)]) {
					return nil, nil, fmt.Errorf("spread element type does not fit fixed input")
				}
				values = append(values, index(value, consumed))
				consumed++
			}
		}
		if tail != nil && (!known || consumed < length) {
			if !types.Assignable(value.Type.Element(), tail.Type.Element()) {
				return nil, nil, fmt.Errorf("spread element type does not fit variadic input")
			}
			rest := value
			if consumed > 0 {
				rest = &ir.Expression{Kind: ir.Slice, Span: value.Span, Type: value.Type, Inputs: []*ir.Expression{value, {Kind: ir.Literal, Span: value.Span, Type: integer, Text: strconv.Itoa(consumed)}, nil}}
			}
			tail.Inputs = append(tail.Inputs, rest)
			tail.Spread = append(tail.Spread, true)
		}
	}
	if len(values) != fixed {
		return nil, nil, fmt.Errorf("call arity mismatch")
	}
	if tail != nil {
		values = append(values, tail)
	}
	for i, node := range state {
		value, err := e.Check(node, stateTypes[i])
		if err != nil {
			return nil, nil, err
		}
		values = append(values, save(value))
	}
	return prepared, values, nil
}

// fixedArgumentElements supplies syntax-level equalities for fixed-arity
// inference. Runtime preparation must still use arguments on the original list
// so every spread is checked as one homogeneous array and evaluated once.
func fixedArgumentElements(args []syntax.Argument) ([]syntax.Argument, error) {
	var result []syntax.Argument
	for _, arg := range args {
		if arg.Group != nil {
			return nil, fmt.Errorf("fixed inputs do not accept state groups")
		}
		if !arg.Spread {
			result = append(result, arg)
			continue
		}
		if _, known := literalSpreadLength(arg.Value); !known {
			return nil, fmt.Errorf("runtime-length spread cannot supply fixed inputs")
		}
		for _, element := range literalSpreadElements(arg.Value) {
			result = append(result, syntax.Argument{Span: element.ExprSpan(), Value: element})
		}
	}
	return result, nil
}
