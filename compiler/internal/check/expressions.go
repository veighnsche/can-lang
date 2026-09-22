// Package check checks current-language operations against resolved concrete
// evidence. It never executes authored expressions while deciding their types.
package check

import (
	"fmt"
	"strconv"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

type ValueBinding struct {
	Identity string
	Type     *types.Type
}
type Expressions struct {
	// File is the owning source file for structured failure spans. The
	// owning body pass sets it; a nil file leaves errors unlocated.
	File                *source.File
	DeferredCheck       func(syntax.Expr, *types.Type, *Expressions, error) (*ir.Expression, error)
	AggregateFieldCheck func(*syntax.FieldExpr, *types.Type) (*ir.Expression, bool, error)
	CoordinationCheck   func(*syntax.CoordinationExpr, *types.Type) (*ir.Expression, error)
	Probability         *ValueBinding
	ResolvedValue       func(*syntax.NameExpr, ValueBinding)
	Scalars             map[string]*types.Type
	ReferenceCheck      func(*syntax.ReferenceExpr, *types.Type) (*ir.Expression, error)
	CallCheck           func(*syntax.CallExpr, *types.Type) (*ir.Expression, error)
	MatchCheck          func(*syntax.MatchExpr, *types.Type) (*ir.Expression, error)
	// The owning body pass supplies its eligible-kind lexical/package resolution.
	// Call lookup is separate because an ineligible local must not hide a function.
	Value       func(syntax.QualifiedName) (ValueBinding, error)
	Reference   func(syntax.QualifiedName) (ValueBinding, error)
	Function    func(syntax.QualifiedName) (ValueBinding, error)
	Constructor func(*syntax.ConstructorExpr, *types.Type, *Expressions) (*types.Type, error)
}

func (c *Expressions) scalar(name string) (*types.Type, error) {
	t := c.Scalars[name]
	if t == nil || !types.Equal(t, t) || t.Declaration() != name || t.Kind() != types.Primitive {
		return nil, fmt.Errorf("missing checked scalar %s", name)
	}
	return t, nil
}
func scalar(t *types.Type, name string) bool {
	return t != nil && t.Kind() == types.Primitive && t.Declaration() == name
}
func numeric(t *types.Type) bool { return scalar(t, "int") || scalar(t, "float") }
func (c *Expressions) Check(node syntax.Expr, expected *types.Type) (*ir.Expression, error) {
	if node == nil {
		return nil, fmt.Errorf("missing expression")
	}
	out, err := c.expression(node, expected)
	if err != nil && c.DeferredCheck != nil {
		out, err = c.DeferredCheck(node, expected, c, err)
	}
	if err != nil {
		return nil, c.locate(node, fmt.Errorf("expression at byte %d: %w", node.ExprSpan().Start, err))
	}
	if expected != nil && !types.Assignable(out.Type, expected) {
		return nil, c.locate(node, fmt.Errorf("expression type does not fit expected type at byte %d", node.ExprSpan().Start))
	}
	return out, nil
}

// locate attaches the checked expression's file span to err, keeping the
// innermost span when nested checks already located the failure.
func (c *Expressions) locate(node syntax.Expr, err error) error {
	if c.File == nil {
		return err
	}
	return source.Locate(c.File.Name(), node.ExprSpan(), err)
}
func (c *Expressions) expression(node syntax.Expr, expected *types.Type) (*ir.Expression, error) {
	out := &ir.Expression{Span: node.ExprSpan()}
	var err error
	switch n := node.(type) {
	case *syntax.ReferenceExpr:
		if c.ReferenceCheck == nil {
			return nil, fmt.Errorf("callable reference requires its owning region")
		}
		return c.ReferenceCheck(n, expected)
	case *syntax.CoordinationExpr:
		if c.CoordinationCheck == nil {
			return nil, fmt.Errorf("coordination requires a checked region")
		}
		return c.CoordinationCheck(n, expected)
	case *syntax.MatchExpr:
		if c.MatchCheck == nil {
			return nil, fmt.Errorf("value match requires its owning region")
		}
		return c.MatchCheck(n, expected)
	case *syntax.ProbabilityExpr:
		if c.Probability == nil {
			return nil, fmt.Errorf("probability is not available in this region")
		}
		out.Kind = ir.Binding
		out.Text = c.Probability.Identity
		out.Type = c.Probability.Type
	case *syntax.ScopeExpr:
		if expected == nil || !isScopeRequest(expected) {
			return nil, fmt.Errorf("harness scope is only available at an elided ingress argument")
		}
		out.Kind = ir.ScopeRequest
		out.Type = expected
	case *syntax.LiteralExpr:
		out.Kind = ir.Literal
		out.Text = n.Token.Text
		switch n.Token.Kind {
		case syntax.Integer:
			out.Type, err = c.scalar("int")
		case syntax.Float:
			if _, e := strconv.ParseFloat(n.Token.Text, 64); e != nil {
				return nil, fmt.Errorf("nonfinite float literal")
			}
			out.Type, err = c.scalar("float")
		case syntax.String:
			out.Type, err = c.scalar("str")
			out.Text = n.Token.Value
		default:
			if n.Token.Text != "true" && n.Token.Text != "false" {
				return nil, fmt.Errorf("unknown literal")
			}
			out.Type, err = c.scalar("bool")
		}
	case *syntax.NameExpr:
		if c.Value == nil {
			return nil, fmt.Errorf("no value resolver")
		}
		binding, e := c.Value(n.Name)
		if e != nil {
			return nil, e
		}
		if binding.Identity == "" || !types.Equal(binding.Type, binding.Type) || binding.Type.Kind() == types.Void {
			return nil, fmt.Errorf("invalid checked value binding")
		}
		if c.ResolvedValue != nil {
			c.ResolvedValue(n, binding)
		}
		out.Kind = ir.Binding
		out.Text = binding.Identity
		out.Type = binding.Type
	case *syntax.GroupExpr:
		return c.Check(n.Value, expected)
	case *syntax.UnaryExpr:
		operand, e := c.Check(n.Operand, nil)
		if e != nil {
			return nil, e
		}
		valid := n.Operator == "-" && numeric(operand.Type) || n.Operator == "~" && scalar(operand.Type, "int") || n.Operator == "not" && scalar(operand.Type, "bool")
		if !valid {
			return nil, fmt.Errorf("invalid operand for %s", n.Operator)
		}
		out.Kind = ir.Unary
		out.Text = n.Operator
		out.Type = operand.Type
		out.Inputs = []*ir.Expression{operand}
	case *syntax.BinaryExpr:
		left, e := c.Check(n.Left, nil)
		if e != nil {
			return nil, e
		}
		right, e := c.Check(n.Right, nil)
		if e != nil {
			return nil, e
		}
		if !types.Equal(left.Type, right.Type) {
			return nil, fmt.Errorf("operator %s requires identical operand types", n.Operator)
		}
		valid := false
		switch n.Operator {
		case "+":
			valid = numeric(left.Type) || scalar(left.Type, "str")
		case "-", "*", "/", "%", "**":
			valid = numeric(left.Type)
		case "&", "|", "^", "<<", ">>":
			valid = scalar(left.Type, "int")
		case "and", "or":
			valid = scalar(left.Type, "bool")
		}
		if !valid {
			return nil, fmt.Errorf("operator %s is not defined for this type", n.Operator)
		}
		out.Kind = ir.Binary
		out.Text = n.Operator
		out.Type = left.Type
		out.Inputs = []*ir.Expression{left, right}
	case *syntax.ComparisonExpr:
		out.Kind = ir.Comparison
		out.Operators = append([]string(nil), n.Operators...)
		for _, operand := range n.Operands {
			x, e := c.Check(operand, nil)
			if e != nil {
				return nil, e
			}
			out.Inputs = append(out.Inputs, x)
		}
		for i, op := range n.Operators {
			a, b := out.Inputs[i].Type, out.Inputs[i+1].Type
			common := a
			if !types.Equal(a, b) {
				if a.Kind() == types.Variant && types.Assignable(b, a) {
					common = a
				} else if b.Kind() == types.Variant && types.Assignable(a, b) {
					common = b
				} else {
					return nil, fmt.Errorf("comparison requires identical types or a named variant admitting both operands")
				}
			}
			mode := ir.Strict
			if op == "is" || op == "is not" {
				if !types.EqualityEligible(common) {
					return nil, fmt.Errorf("type is not equality eligible")
				}
				if scalar(common, "float") {
					mode = ir.FloatIdentity
				} else if common.Kind() != types.Primitive {
					mode = ir.Deep
				}
			} else if !numeric(common) && !scalar(common, "str") {
				return nil, fmt.Errorf("type is not ordered")
			}
			out.Equality = append(out.Equality, mode)
		}
		out.Type, err = c.scalar("bool")
	case *syntax.IndexExpr:
		receiver, e := c.Check(n.Receiver, nil)
		if e != nil {
			return nil, e
		}
		integer, e := c.scalar("int")
		if e != nil {
			return nil, e
		}
		position, e := c.Check(n.Index, integer)
		if e != nil {
			return nil, e
		}
		if receiver.Type.Kind() == types.Array {
			out.Type = receiver.Type.Element()
		} else if scalar(receiver.Type, "str") {
			out.Type = receiver.Type
		} else {
			return nil, fmt.Errorf("indexing requires array or str")
		}
		out.Kind = ir.Index
		out.Inputs = []*ir.Expression{receiver, position}
	case *syntax.SliceExpr:
		receiver, e := c.Check(n.Receiver, nil)
		if e != nil {
			return nil, e
		}
		return c.slice(receiver, n.Start, n.End, node)
	case *syntax.FieldExpr:
		if c.AggregateFieldCheck != nil {
			value, handled, err := c.AggregateFieldCheck(n, expected)
			if handled {
				return value, err
			}
		}
		receiver, e := c.Check(n.Receiver, nil)
		if e != nil {
			return nil, e
		}
		out.Inputs = []*ir.Expression{receiver}
		out.Text = n.Field.Text
		if n.Field.Text == "length" && (receiver.Type.Kind() == types.Array || scalar(receiver.Type, "str") || receiver.Type.Declaration() == "can.std.bytes@1::buffer") {
			out.Kind = ir.Length
			out.Type, err = c.scalar("int")
		} else if receiver.Type.Kind() == types.Opaque && receiver.Type.Declaration() == "can.prelude@1::standard_failure" {
			out.Kind = ir.StandardProjection
			switch n.Field.Text {
			case "occurrence_id":
				out.Type, err = c.scalar("int")
			case "kind", "message":
				out.Type, err = c.scalar("str")
			default:
				return nil, fmt.Errorf("unknown standard_failure projection %s", n.Field.Text)
			}
		} else {
			if receiver.Type.Kind() != types.Record && receiver.Type.Kind() != types.Error {
				return nil, fmt.Errorf("field access requires an ordinary record/error or an explicit catalogue projection")
			}
			for _, f := range receiver.Type.Fields() {
				if f.Name == n.Field.Text {
					out.Type = f.Type
					break
				}
			}
			if out.Type == nil {
				return nil, fmt.Errorf("unknown field %s", n.Field.Text)
			}
			out.Kind = ir.Field
		}
	case *syntax.ArrayExpr:
		out.Kind = ir.Array
		var element *types.Type
		if expected != nil && expected.Kind() == types.Array {
			element = expected.Element()
			out.Type = expected
		}
		for _, argument := range n.Elements {
			if argument.Group != nil {
				return nil, fmt.Errorf("state group in array")
			}
			want := element
			if argument.Spread {
				want = nil
				// Context belongs to fresh literal construction, including grouped
				// nested spreads. Existing arrays retain their invariant type.
				literal := argument.Value
				for {
					group, ok := literal.(*syntax.GroupExpr)
					if !ok {
						break
					}
					literal = group.Value
				}
				if _, ok := literal.(*syntax.ArrayExpr); ok && element != nil {
					var err error
					want, err = types.ArrayOfChecked(element)
					if err != nil {
						return nil, err
					}
				}
			}
			x, e := c.Check(argument.Value, want)
			if e != nil {
				return nil, e
			}
			actual := x.Type
			if argument.Spread {
				if actual.Kind() != types.Array {
					return nil, fmt.Errorf("array spread requires array")
				}
				actual = actual.Element()
			}
			if element == nil {
				element = actual
			} else if expected == nil && !types.Equal(element, actual) || expected != nil && !types.Assignable(actual, element) {
				return nil, fmt.Errorf("array elements need one identical type or the expected named variant")
			}
			out.Inputs = append(out.Inputs, x)
			out.Spread = append(out.Spread, argument.Spread)
		}
		if element == nil {
			return nil, fmt.Errorf("empty array requires expected element type")
		}
		if out.Type == nil {
			out.Type, err = types.ArrayOfChecked(element)
		}
	case *syntax.CallExpr:
		return c.call(n, expected)
	case *syntax.ConstructorExpr:
		if c.Constructor == nil {
			return nil, fmt.Errorf("constructor requires nominal resolution")
		}
		typ, e := c.Constructor(n, expected, c)
		if e != nil {
			return nil, e
		}
		if !types.Equal(typ, typ) || (typ.Kind() != types.Record && typ.Kind() != types.Error) {
			return nil, fmt.Errorf("constructor requires an ordinary nominal record/error")
		}
		fields := typ.Fields()
		if len(fields) != len(n.Arguments) {
			return nil, fmt.Errorf("constructor arity mismatch")
		}
		out.Kind = ir.Record
		out.Type = typ
		out.Text = typ.Identity()
		for i, arg := range n.Arguments {
			if arg.Spread || arg.Group != nil {
				return nil, fmt.Errorf("constructor spread requires fixed-argument expansion")
			}
			value, e := c.Check(arg.Value, fields[i].Type)
			if e != nil {
				return nil, e
			}
			out.Inputs = append(out.Inputs, value)
			out.Fields = append(out.Fields, fields[i].Name)
		}
	case *syntax.UpdateExpr:
		receiver, e := c.Check(n.Receiver, nil)
		if e != nil {
			return nil, e
		}
		out.Kind = ir.Update
		out.Inputs = []*ir.Expression{receiver}
		fields := map[string]*types.Type{}
		for _, f := range receiver.Type.Fields() {
			fields[f.Name] = f.Type
		}
		var replacements []types.Replacement
		for _, field := range n.Fields {
			want := fields[field.Name.Text]
			if want == nil {
				return nil, fmt.Errorf("unknown update field %s", field.Name.Text)
			}
			value, e := c.Check(field.Value, want)
			if e != nil {
				return nil, e
			}
			out.Inputs = append(out.Inputs, value)
			out.Fields = append(out.Fields, field.Name.Text)
			replacements = append(replacements, types.Replacement{Name: field.Name.Text, Type: value.Type})
		}
		out.Type, err = types.CheckUpdate(receiver.Type, replacements)
	default:
		return nil, fmt.Errorf("expression %T requires its owning checker pass", node)
	}
	if err != nil {
		return nil, err
	}
	return out, nil
}
func (c *Expressions) slice(receiver *ir.Expression, start, end syntax.Expr, node syntax.Expr) (*ir.Expression, error) {
	if receiver.Type.Kind() != types.Array && !scalar(receiver.Type, "str") {
		return nil, fmt.Errorf("slicing requires array or str")
	}
	integer, err := c.scalar("int")
	if err != nil {
		return nil, err
	}
	out := &ir.Expression{Kind: ir.Slice, Type: receiver.Type, Span: node.ExprSpan(), Inputs: []*ir.Expression{receiver, nil, nil}}
	if start != nil {
		out.Inputs[1], err = c.Check(start, integer)
		if err != nil {
			return nil, err
		}
	}
	if end != nil {
		out.Inputs[2], err = c.Check(end, integer)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}
func (c *Expressions) call(n *syntax.CallExpr, expected *types.Type) (*ir.Expression, error) {
	if c.CallCheck != nil {
		return c.CallCheck(n, expected)
	}
	var out *ir.Expression
	invocation := n.Invocation
	if len(invocation.Types) != 0 {
		return nil, fmt.Errorf("generic call needs specialization checking")
	}
	if field, ok := invocation.Callee.(*syntax.FieldExpr); ok && field.Field.Text == "slice" {
		receiver, err := c.Check(field.Receiver, nil)
		if err != nil {
			return nil, err
		}
		out, err = c.sliceArguments(receiver, invocation.Arguments, n)
		if err != nil {
			return nil, err
		}
	} else {
		name, ok := invocation.Callee.(*syntax.NameExpr)
		if !ok || c.Function == nil {
			return nil, fmt.Errorf("call requires resolved callable contract")
		}
		binding, err := c.Function(name.Name)
		if err != nil {
			return nil, err
		}
		if binding.Type == nil || !types.Equal(binding.Type, binding.Type) || binding.Type.Kind() != types.Callable || binding.Identity == "" {
			return nil, fmt.Errorf("invalid callable binding")
		}
		if len(binding.Type.Errors()) != 0 {
			return nil, fmt.Errorf("domain-fallible call requires explicit completion handling")
		}
		inputs := binding.Type.Inputs()
		if len(inputs) != len(invocation.Arguments) {
			return nil, fmt.Errorf("call arity mismatch")
		}
		out = &ir.Expression{Kind: ir.Call, Text: binding.Identity, Type: binding.Type.Result(), Span: n.ExprSpan()}
		for i, a := range invocation.Arguments {
			if a.Spread || a.Group != nil {
				return nil, fmt.Errorf("call spread/state group requires owning call checker")
			}
			x, err := c.Check(a.Value, inputs[i])
			if err != nil {
				return nil, err
			}
			out.Inputs = append(out.Inputs, x)
		}
	}
	for _, method := range n.Methods {
		if method.Name.Text != "slice" || len(method.Types) != 0 {
			return nil, fmt.Errorf("method requires catalogue/method checking")
		}
		var err error
		out, err = c.sliceArguments(out, method.Arguments, n)
		if err != nil {
			return nil, err
		}
	}
	if out.Type.Kind() == types.Void {
		return nil, fmt.Errorf("void call is not a value expression")
	}
	return out, nil
}
func (c *Expressions) sliceArguments(receiver *ir.Expression, args []syntax.Argument, n syntax.Expr) (*ir.Expression, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("slice method requires two int arguments")
	}
	for _, a := range args {
		if a.Spread || a.Group != nil {
			return nil, fmt.Errorf("slice does not accept spread or state groups")
		}
	}
	return c.slice(receiver, args[0].Value, args[1].Value, n)
}
