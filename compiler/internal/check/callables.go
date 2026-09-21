package check

import (
	"fmt"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// CallableDeclaration is declaration evidence, not a dynamically supplied value.
// Native front ends use the same eligibility gate: fetch has ordinary arguments;
// questions, judges and LLMs have no selected ordinary reference contract.
type CallableDeclaration struct {
	Grouped  bool
	State    int
	Kind     resolve.Kind
	Contract *types.Type
	Names    []string
	Near     []bool
	Receiver bool
}

func (d CallableDeclaration) validate() error {
	if d.Kind != resolve.Function && d.Kind != resolve.Fetch {
		return fmt.Errorf("%s declaration cannot be an ordinary callable reference; use a named function wrapper", d.Kind)
	}
	if !types.Equal(d.Contract, d.Contract) || d.Contract.Kind() != types.Callable || len(d.Names) != len(d.Contract.Inputs()) || len(d.Near) != len(d.Names) {
		return fmt.Errorf("incomplete callable declaration contract")
	}
	if d.Receiver && (len(d.Names) == 0 || d.Near[0]) {
		return fmt.Errorf("invalid receiver capture contract")
	}
	seen := map[string]bool{}
	for _, name := range d.Names {
		if name == "" || seen[name] {
			return fmt.Errorf("invalid callable input names")
		}
		seen[name] = true
	}
	return nil
}
func (c *regionChecker) reference(n *syntax.ReferenceExpr, scope bodyScope, expected *types.Type, hints ...callbackHint) (*ir.Expression, error) {
	e := c.expressions(scope)
	var binding ValueBinding
	var receiver *ir.Expression
	var err error
	switch callee := n.Callee.(type) {
	case *syntax.NameExpr:
		if c.context.IntrinsicIdentity != nil && c.context.IntrinsicIdentity(scope.symbols, callee.Name) == "can.prelude@1::append" {
			c.uses.Names[callee] = "can.prelude@1::append"
			return c.arrayReference(n, "append", nil, expected, hints)
		}
		if e.Reference == nil {
			return nil, fmt.Errorf("missing named reference resolver")
		}
		if len(n.Types) > 0 {
			if c.context.Specialize == nil {
				return nil, fmt.Errorf("missing generic reference resolver")
			}
			binding, err = c.context.Specialize(scope.symbols, callee.Name, n.Types)
		} else {
			handled := false
			if len(hints) > 0 && c.context.InferCallback != nil {
				binding, handled, err = c.context.InferCallback(scope.symbols, callee.Name, hints[0].inputs, hints[0].result, e)
			}
			if !handled && c.context.InferReference != nil {
				binding, handled, err = c.context.InferReference(scope.symbols, callee.Name, expected, e)
			}
			if !handled {
				binding, err = e.Reference(callee.Name)
			}
		}
		if err == nil {
			c.uses.Names[callee] = binding.Identity
		}
	case *syntax.FieldExpr:
		receiver, err = e.Check(callee.Receiver, nil)
		if err == nil && receiver.Type.Kind() == types.Array && (arrayMethod(callee.Field.Text) || callee.Field.Text == "slice") {
			return c.arrayReference(n, callee.Field.Text, receiver, expected, hints)
		}
		if err == nil {
			if c.context.Method == nil && c.context.ResolveMethod == nil {
				return nil, fmt.Errorf("missing receiver method resolver")
			}
			application := MethodApplication{Receiver: receiver.Type, Name: callee.Field, Types: n.Types, Expected: expected, Reference: true, Expressions: e}
			if len(hints) > 0 {
				application.CallbackInputs = hints[0].inputs
				application.CallbackResult = hints[0].result
			}
			binding, err = c.resolveMethod(application)
		}
	default:
		return nil, fmt.Errorf("callable reference requires a named declaration or receiver method")
	}
	if err != nil {
		return nil, err
	}
	if routeOperation(binding.Identity) {
		return nil, fmt.Errorf("route construction requires a direct call with a static path")
	}
	declaration, ok := c.context.Callables[binding.Identity]
	if !ok {
		return nil, fmt.Errorf("missing named callable declaration evidence")
	}
	if err = declaration.validate(); err != nil {
		return nil, err
	}
	if !types.Equal(binding.Type, declaration.Contract) || (receiver != nil) != declaration.Receiver {
		return nil, fmt.Errorf("callable target/receiver contract mismatch")
	}
	site, err := c.lexicalSite("callable", n.Span)
	if err != nil {
		return nil, err
	}
	out := &ir.Expression{Kind: ir.CallableValue, Span: n.Span, Callable: &ir.Callable{Site: site, Target: binding.Identity, Contract: binding.Type}}
	var inputs []*types.Type
	captures := []string{}
	for i, typ := range binding.Type.Inputs() {
		var captured *ir.Expression
		if declaration.Receiver && i == 0 {
			captured = receiver
		} else if declaration.Near[i] {
			captured, err = e.Check(&syntax.NameExpr{ExpressionLocation: syntax.ExpressionLocation{Span: n.Span}, Name: syntax.QualifiedName{Name: declaration.Names[i]}}, nil)
			if err != nil {
				return nil, fmt.Errorf("near %s: %w", declaration.Names[i], err)
			}
			captures = append(captures, captured.Text)
		}
		if captured != nil {
			if !types.Equal(captured.Type, typ) {
				return nil, fmt.Errorf("near or receiver capture %s requires exact declared type", declaration.Names[i])
			}
			out.Callable.Positions = append(out.Callable.Positions, i)
			out.Inputs = append(out.Inputs, captured)
		} else {
			inputs = append(inputs, typ)
		}
	}
	out.Type, err = types.CallableOfChecked(binding.Type.Result(), inputs, binding.Type.Errors())
	if err != nil {
		return nil, err
	}
	out.Callable.ResourceCaptures = ir.ResourceCaptureIndices(out.Inputs)
	c.uses.Captures[n] = captures
	return out, nil
}
