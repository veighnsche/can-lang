package types

import (
	"fmt"
	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
)

// Shape inference runs while generated declarations are being built. It uses
// the ordinary equality solver, but its evidence is local to this builder and
// cannot be exported as sealed compatibility or expression-admission evidence.
func (b *Builder) inferNativeShape(file *resolve.File, scope *resolve.Scope, declaring *resolve.File, symbol *resolve.Symbol, fn *syntax.FunctionDecl, args []syntax.Argument, receiver *Type) (map[string]*Type, error) {
	solver, err := NewInference(symbol.Parameters)
	if err != nil {
		return nil, err
	}
	constrain := func(pattern *InferencePattern, actual *Type) error {
		return solver.constrain(pattern, actual, func(t *Type) bool { return t != nil && t.graph == b.graph && t.defined }, func(a, c *Type) bool { return a == c })
	}
	if fn.Receiver != nil {
		if receiver == nil {
			return nil, fmt.Errorf("generic spread method requires receiver")
		}
		pattern, err := Pattern(declaring, fn.Receiver.Type, symbol.Parameters)
		if err != nil {
			return nil, err
		}
		if err = constrain(pattern, receiver); err != nil {
			return nil, err
		}
	}
	type constraint struct {
		annotation syntax.TypeNode
		expression syntax.Expr
		pattern    *InferencePattern
	}
	var constraints []constraint
	fixed := len(fn.Inputs)
	variadic := fixed > 0 && fn.Inputs[fixed-1].Variadic
	if variadic {
		fixed--
	}
	position := 0
	add := func(value syntax.Expr) error {
		index := position
		if index >= fixed {
			if !variadic {
				return fmt.Errorf("generic spread call arity mismatch")
			}
			index = fixed
		}
		constraints = append(constraints, constraint{annotation: fn.Inputs[index].Type, expression: value})
		position++
		return nil
	}
	for _, arg := range args {
		if arg.Group != nil {
			return nil, fmt.Errorf("generic function does not accept a state group")
		}
		if !arg.Spread {
			if err = add(arg.Value); err != nil {
				return nil, err
			}
			continue
		}
		if elements, known := nativeSpreadElements(arg.Value); known {
			for _, element := range elements {
				if err = add(element); err != nil {
					return nil, err
				}
			}
		} else {
			if position < fixed || !variadic {
				return nil, fmt.Errorf("runtime-length spread cannot supply generic fixed inputs")
			}
			constraints = append(constraints, constraint{annotation: &syntax.ArrayType{Element: fn.Inputs[fixed].Type}, expression: arg.Value})
		}
	}
	if position < fixed {
		return nil, fmt.Errorf("generic spread call arity mismatch")
	}
	for index := range constraints {
		constraints[index].pattern, err = Pattern(declaring, constraints[index].annotation, symbol.Parameters)
		if err != nil {
			return nil, err
		}
	}
	done := make([]bool, len(constraints))
	remaining := len(done)
	for remaining > 0 {
		progress := false
		var unresolved error
		for index, entry := range constraints {
			if done[index] {
				continue
			}
			var want *Type
			if solver.Resolved(entry.pattern) {
				want, err = b.Resolve(declaring, entry.annotation, solver.Bindings(), false)
				if err != nil {
					return nil, err
				}
			}
			actual, err := b.nativeArgumentShape(file, scope, entry.expression, want)
			if err != nil {
				unresolved = err
				continue
			}
			if entry.pattern.HasParameters() {
				if err = constrain(entry.pattern, actual); err != nil {
					return nil, err
				}
			}
			done[index] = true
			remaining--
			progress = true
		}
		if !progress {
			if _, err = solver.Arguments(); err != nil {
				return nil, err
			}
			return nil, unresolved
		}
	}
	if _, err = solver.Arguments(); err != nil {
		return nil, err
	}
	return solver.Bindings(), nil
}

func nativeSpreadElements(node syntax.Expr) ([]syntax.Expr, bool) {
	if group, ok := node.(*syntax.GroupExpr); ok {
		return nativeSpreadElements(group.Value)
	}
	literal, ok := node.(*syntax.ArrayExpr)
	if !ok {
		return nil, false
	}
	var values []syntax.Expr
	for _, item := range literal.Elements {
		if item.Group != nil {
			return nil, false
		}
		if item.Spread {
			nested, known := nativeSpreadElements(item.Value)
			if !known {
				return nil, false
			}
			values = append(values, nested...)
		} else {
			values = append(values, item.Value)
		}
	}
	return values, true
}
func (b *Builder) nativeArgumentShape(file *resolve.File, scope *resolve.Scope, node syntax.Expr, want *Type) (*Type, error) {
	switch value := node.(type) {
	case *syntax.GroupExpr:
		return b.nativeArgumentShape(file, scope, value.Value, want)
	case *syntax.LiteralExpr:
		name := map[syntax.Kind]string{syntax.Integer: "int", syntax.Float: "float", syntax.String: "str"}[value.Token.Kind]
		if name == "" && (value.Token.Text == "true" || value.Token.Text == "false") {
			name = "bool"
		}
		if name != "" {
			return b.graph.scalar(name), nil
		}
	case *syntax.ArrayExpr:
		var element *Type
		if want != nil && want.kind == Array {
			element = want.element
		}
		for _, item := range value.Elements {
			if item.Group != nil {
				return nil, fmt.Errorf("state group is not array shape evidence")
			}
			expected := element
			if item.Spread && element != nil {
				var err error
				expected, err = b.graph.array(element)
				if err != nil {
					return nil, err
				}
			}
			actual, err := b.nativeArgumentShape(file, scope, item.Value, expected)
			if err != nil {
				return nil, err
			}
			if item.Spread {
				if actual.kind != Array {
					return nil, fmt.Errorf("spread shape is not an array")
				}
				actual = actual.element
			}
			if element == nil {
				element = actual
			} else if want == nil && element != actual {
				return nil, fmt.Errorf("conflicting array shape evidence")
			}
		}
		if element == nil {
			return nil, fmt.Errorf("empty array needs contextual shape")
		}
		return b.graph.array(element)
	}
	return b.nativeShape(file, scope, node)
}
