package check

import (
	"fmt"
	"github.com/veighnsche/can-lang/compiler/internal/catalogue"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

type argumentConstraint struct {
	annotation syntax.TypeNode
	expression syntax.Expr
}

func (c *programChecker) inferCall(file *resolve.File, scope *resolve.Scope, name syntax.QualifiedName, args []syntax.Argument, expected *types.Type, e *Expressions) (ValueBinding, bool, error) {
	symbol, err := file.Lookup(scope, name, resolve.CallUse)
	if err != nil {
		return ValueBinding{}, false, nil
	}
	if op := collectionOperation(symbol.ID); op != nil {
		return c.inferCollection(op, args, expected, e)
	}
	declaration, ok := symbol.Declaration.(*syntax.FunctionDecl)
	if !ok || len(symbol.Parameters) == 0 {
		return ValueBinding{}, false, nil
	}
	if declaration.Receiver != nil {
		return ValueBinding{}, true, fmt.Errorf("generic method requires receiver application")
	}
	constraints, err := genericArguments(declaration.Inputs, args)
	if err != nil {
		return ValueBinding{}, true, err
	}
	arguments, err := c.inferArguments(c.world.Files[symbol.Source], symbol.Parameters, declaration.Result, expected, constraints, e)
	if err != nil {
		return ValueBinding{}, true, fmt.Errorf("generic call %s at byte %d: %w", symbol.ID, name.Span.Start, err)
	}
	substitutions := c.inferredSubstitutions(file, scope, symbol, constraints)
	binding, err := c.instantiateFunction(symbol, declaration, arguments, applicationSite(file, name.Span.Start), substitutions)
	return binding, true, err
}

func genericArguments(inputs []syntax.Input, args []syntax.Argument) ([]argumentConstraint, error) {
	fixed := len(inputs)
	variadic := fixed > 0 && inputs[fixed-1].Variadic
	if variadic {
		fixed--
	}
	position := 0
	var constraints []argumentConstraint
	add := func(expression syntax.Expr) error {
		index := position
		if index >= fixed {
			if !variadic {
				return fmt.Errorf("generic call arity mismatch")
			}
			index = fixed
		}
		constraints = append(constraints, argumentConstraint{inputs[index].Type, expression})
		position++
		return nil
	}
	for _, arg := range args {
		if arg.Group != nil {
			return nil, fmt.Errorf("generic function does not accept a state group")
		}
		if !arg.Spread {
			if err := add(arg.Value); err != nil {
				return nil, err
			}
			continue
		}
		_, known := literalSpreadLength(arg.Value)
		if !known {
			if position < fixed || !variadic {
				return nil, fmt.Errorf("runtime-length spread cannot supply generic fixed inputs")
			}
			constraints = append(constraints, argumentConstraint{&syntax.ArrayType{Element: inputs[fixed].Type}, arg.Value})
			continue
		}
		for _, element := range literalSpreadElements(arg.Value) {
			if err := add(element); err != nil {
				return nil, err
			}
		}
	}
	if position < fixed {
		return nil, fmt.Errorf("generic call arity mismatch")
	}
	return constraints, nil
}

func (c *programChecker) inferArguments(file *resolve.File, parameters []string, result syntax.TypeNode, expected *types.Type, constraints []argumentConstraint, e *Expressions, seeds ...typeConstraint) ([]*types.Type, error) {
	solver, err := types.NewInference(parameters)
	if err != nil {
		return nil, err
	}
	for _, seed := range seeds {
		pattern, err := types.Pattern(file, seed.annotation, parameters)
		if err != nil {
			return nil, err
		}
		if err = solver.Constrain(pattern, seed.actual); err != nil {
			return nil, err
		}
	}
	if expected != nil {
		pattern, err := types.Pattern(file, result, parameters)
		if err != nil {
			return nil, err
		}
		if pattern.HasParameters() {
			if err = solver.Constrain(pattern, expected); err != nil {
				return nil, err
			}
		}
	}
	patterns := make([]*types.InferencePattern, len(constraints))
	for i, constraint := range constraints {
		patterns[i], err = types.Pattern(file, constraint.annotation, parameters)
		if err != nil {
			return nil, err
		}
	}
	done := make([]bool, len(constraints))
	remaining := len(done)
	var unresolved error
	for remaining > 0 {
		progress := false
		for i, constraint := range constraints {
			if done[i] {
				continue
			}
			var want *types.Type
			if solver.Resolved(patterns[i]) {
				want, err = c.specializer.Resolve(file, constraint.annotation, solver.Bindings(), false)
				if err != nil {
					return nil, err
				}
			}
			value, err := e.Check(constraint.expression, want)
			if err != nil {
				unresolved = err
				continue
			}
			if patterns[i].HasParameters() {
				if err = solver.Constrain(patterns[i], value.Type); err != nil {
					return nil, err
				}
			}
			done[i] = true
			remaining--
			progress = true
		}
		if !progress {
			if _, err := solver.Arguments(); err != nil {
				return nil, err
			}
			return nil, unresolved
		}
	}
	return solver.Arguments()
}

func (c *programChecker) inferReference(file *resolve.File, scope *resolve.Scope, name syntax.QualifiedName, expected *types.Type, e *Expressions) (ValueBinding, bool, error) {
	symbol, err := file.Lookup(scope, name, resolve.CallUse)
	if err != nil {
		return ValueBinding{}, false, nil
	}
	if op := collectionOperation(symbol.ID); op != nil {
		return c.referenceCollection(op, expected, nil, nil)
	}
	d, ok := symbol.Declaration.(*syntax.FunctionDecl)
	if !ok || len(symbol.Parameters) == 0 {
		return ValueBinding{}, false, nil
	}
	if d.Receiver != nil {
		return ValueBinding{}, true, fmt.Errorf("generic method reference requires receiver application")
	}
	return c.inferDeclaredReference(symbol, d, expected, e, applicationSite(file, name.Span.Start))
}

func (c *programChecker) inferDeclaredReference(symbol *resolve.Symbol, d *syntax.FunctionDecl, expected *types.Type, e *Expressions, request string, seeds ...typeConstraint) (ValueBinding, bool, error) {
	solver, err := types.NewInference(symbol.Parameters)
	if err != nil {
		return ValueBinding{}, true, err
	}
	declaring := c.world.Files[symbol.Source]
	for _, seed := range seeds {
		pattern, err := types.Pattern(declaring, seed.annotation, symbol.Parameters)
		if err != nil {
			return ValueBinding{}, true, err
		}
		if err = solver.Constrain(pattern, seed.actual); err != nil {
			return ValueBinding{}, true, err
		}
	}
	residual := &syntax.CallableType{Result: d.Result, Errors: d.Errors}
	for _, input := range d.Inputs {
		if input.Near {
			continue
		}
		node := input.Type
		if input.Variadic {
			node = &syntax.ArrayType{Element: node}
		}
		residual.Inputs = append(residual.Inputs, node)
	}
	if expected != nil {
		pattern, err := types.Pattern(declaring, residual, symbol.Parameters)
		if err != nil {
			return ValueBinding{}, true, err
		}
		if err = solver.Constrain(pattern, expected); err != nil {
			return ValueBinding{}, true, err
		}
	}
	for _, input := range d.Inputs {
		if !input.Near {
			continue
		}
		value, err := e.Check(&syntax.NameExpr{ExpressionLocation: syntax.ExpressionLocation{Span: d.Span}, Name: syntax.QualifiedName{Name: input.Name.Text}}, nil)
		if err != nil {
			return ValueBinding{}, true, err
		}
		node := input.Type
		if input.Variadic {
			node = &syntax.ArrayType{Element: node}
		}
		pattern, err := types.Pattern(declaring, node, symbol.Parameters)
		if err != nil {
			return ValueBinding{}, true, err
		}
		if err = solver.Constrain(pattern, value.Type); err != nil {
			return ValueBinding{}, true, err
		}
	}
	args, err := solver.Arguments()
	if err != nil {
		return ValueBinding{}, true, fmt.Errorf("generic reference %s at byte %d: %w", symbol.ID, d.Span.Start, err)
	}
	binding, err := c.instantiateFunction(symbol, d, args, request)
	return binding, true, err
}

func (c *programChecker) inferConstructor(symbol *resolve.Symbol, node *syntax.ConstructorExpr, expected *types.Type, e *Expressions) (*types.Type, error) {
	var fields []syntax.Field
	file := c.world.Files[symbol.Source]
	parameterNames := symbol.Parameters
	qualified := syntax.QualifiedName{Name: symbol.Name}
	switch d := symbol.Declaration.(type) {
	case *syntax.RecordDecl:
		fields = d.Fields
	case *syntax.ErrorDecl:
		fields = d.Fields
	default:
		var metadata []catalogue.Field
		found := false
		inventory := catalogue.Builtin().Inventory()
		for _, declaration := range inventory.Types {
			if declaration.Identity == symbol.ID && declaration.Kind == "record" && declaration.Constructible {
				metadata = declaration.Fields
				found = true
				break
			}
		}
		for _, declaration := range inventory.Errors {
			if declaration.Identity == symbol.ID {
				metadata = declaration.Fields
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("generic constructor requires record or error declaration")
		}
		for _, field := range metadata {
			annotation, err := catalogueTypeNode(field.Type, symbol.Parameters)
			if err != nil {
				return nil, err
			}
			fields = append(fields, syntax.Field{Name: syntax.Token{Text: field.Name}, Type: annotation})
		}
		file = &resolve.File{Scope: c.world.Prelude, Imports: c.world.Packages}
		if symbol.Package != nil {
			qualified.Package = symbol.Package.Name
		}
		parameterNames = make([]string, len(symbol.Parameters))
		for i, name := range symbol.Parameters {
			parameterNames[i] = "catalogue_" + strings.ToLower(name)
		}
	}
	if len(fields) != len(node.Arguments) {
		return nil, fmt.Errorf("constructor arity mismatch")
	}
	constraints := make([]argumentConstraint, len(fields))
	for i, field := range fields {
		argument := node.Arguments[i]
		if argument.Spread || argument.Group != nil {
			return nil, fmt.Errorf("constructor requires fixed field arguments")
		}
		constraints[i] = argumentConstraint{field.Type, argument.Value}
	}
	template := &syntax.NamedType{Name: qualified}
	for _, parameter := range parameterNames {
		template.Arguments = append(template.Arguments, named(parameter))
	}
	// Only a unique matching concrete nominal expectation contributes equalities.
	// Fields can still determine the application when a wider variant is expected.
	if expected != nil && expected.Declaration() != symbol.ID {
		var candidate *types.Type
		for _, leaf := range expected.Leaves() {
			if leaf.Declaration() == symbol.ID {
				if candidate != nil {
					candidate = nil
					break
				}
				candidate = leaf
			}
		}
		expected = candidate
	}
	arguments, err := c.inferArguments(file, parameterNames, template, expected, constraints, e)
	if err != nil {
		return nil, fmt.Errorf("generic constructor %s at byte %d: %w", symbol.ID, node.Span.Start, err)
	}
	parameters := map[string]*types.Type{}
	for i, name := range parameterNames {
		parameters[name] = arguments[i]
	}
	return c.specializer.Resolve(file, template, parameters, false)
}

type typeConstraint struct {
	annotation syntax.TypeNode
	actual     *types.Type
}

// Only syntactically known spreads reach this helper. Checking the original
// array during ordinary argument lowering still enforces its homogeneous type
// and single evaluation; inference retains each element's expected context.
func literalSpreadElements(node syntax.Expr) []syntax.Expr {
	if group, ok := node.(*syntax.GroupExpr); ok {
		return literalSpreadElements(group.Value)
	}
	array := node.(*syntax.ArrayExpr)
	var elements []syntax.Expr
	for _, element := range array.Elements {
		if element.Spread {
			elements = append(elements, literalSpreadElements(element.Value)...)
		} else {
			elements = append(elements, element.Value)
		}
	}
	return elements
}
