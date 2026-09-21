package check

import (
	"fmt"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// Assertions checks every row before execution. The actual side is a checked
// invocation of the declared function, not a copy of its expected completion.
// Receiver/variadic input rules therefore use the ordinary invocation checker.
func (c *programChecker) assertions(file *resolve.File, fn *ProgramFunction, context CompletionContext) ([]*ir.Assertion, error) {
	declaration := fn.Symbol.Declaration.(*syntax.FunctionDecl)
	if err := validateAssertionNames(declaration); err != nil {
		return nil, err
	}
	return c.assertionRows(file, fn, context, declaration.Assertions)
}

func (c *programChecker) assertionRows(file *resolve.File, fn *ProgramFunction, context CompletionContext, rows []syntax.Assertion) ([]*ir.Assertion, error) {
	declaration := fn.Symbol.Declaration.(*syntax.FunctionDecl)
	var result []*ir.Assertion
	for _, row := range rows {
		root := ir.AssertionRoot{Package: fn.Symbol.Package.ID, Declaration: fn.Symbol.ID, Name: row.Name.Text}
		contract := c.bindings[fn.Identity()]
		if contract != nil && contract.Kind() == types.Callable {
			if success, ok := row.Expected.(*syntax.SuccessBody); ok && success.Value != nil {
				if isScopeRequest(contract.Result()) {
					return nil, fmt.Errorf("assertion %s: ingress scope results cannot be named in expected completions", row.Name.Text)
				}
				if kind := contract.Result().Kind(); kind == types.Opaque || kind == types.Callable {
					return nil, fmt.Errorf("assertion %s: excluded opaque results use bare ok", row.Name.Text)
				}
			}
		}
		var callee syntax.Expr = &syntax.NameExpr{Name: syntax.QualifiedName{Name: declaration.Name.Text}, ExpressionLocation: syntax.ExpressionLocation{Span: row.Span}}
		receiver := 0
		if declaration.Receiver != nil {
			receiver = 1
			scope := contract != nil && contract.Kind() == types.Callable && len(contract.Inputs()) > 0 && isScopeRequest(contract.Inputs()[0])
			if row.Receiver == nil {
				if !scope {
					return nil, fmt.Errorf("method assertion requires a receiver")
				}
				callee = &syntax.FieldExpr{Receiver: &syntax.ScopeExpr{ExpressionLocation: syntax.ExpressionLocation{Span: row.Span}}, Field: declaration.Name, ExpressionLocation: syntax.ExpressionLocation{Span: row.Span}}
			} else {
				if scope {
					return nil, fmt.Errorf("assertion %s: ingress scope receivers are harness-supplied", row.Name.Text)
				}
				callee = &syntax.FieldExpr{Receiver: row.Receiver, Field: declaration.Name, ExpressionLocation: syntax.ExpressionLocation{Span: row.Span}}
			}
		} else if row.Receiver != nil {
			return nil, fmt.Errorf("ordinary function assertion cannot supply a receiver")
		}
		expanded := row.Arguments
		if contract != nil && contract.Kind() == types.Callable {
			inputs := contract.Inputs()
			var err error
			expanded, err = expandScopeArguments(len(inputs)-receiver, func(i int) bool {
				return isScopeRequest(inputs[i+receiver])
			}, c.variadic[fn.Identity()], row.Arguments, row.Span)
			if err != nil {
				return nil, fmt.Errorf("assertion %s arguments: %w", row.Name.Text, err)
			}
		}
		call := &syntax.CallExpr{ExpressionLocation: syntax.ExpressionLocation{Span: row.Span}, Invocation: syntax.Invocation{Span: row.Span, Callee: callee, Arguments: expanded}}
		actualContext := context
		actualContext.Identity = fn.Symbol.ID + "/assert/" + row.Name.Text + "/actual"
		actualContext.Sites = nil // This synthetic root call has its own syntax tree.
		actualContext.Scope = file.Scope
		actualContext.Parameters = nil
		actualContext.Expressions = c.expressions(file, file.Scope)
		actual, err := CheckRegion(actualContext, syntax.Block{Span: row.Span, Terminal: &syntax.RelayBody{BodyLocation: syntax.BodyLocation{Span: row.Span}, Call: call}})
		if err != nil {
			return nil, fmt.Errorf("assertion %s arguments: %w", row.Name.Text, err)
		}
		switch row.Expected.(type) {
		case *syntax.SuccessBody, *syntax.FailureBody:
		default:
			return nil, fmt.Errorf("assertion requires an explicit expected completion")
		}
		expectedContext := actualContext
		expectedContext.Identity = fn.Symbol.ID + "/assert/" + row.Name.Text + "/expected"
		expectedContext.BareOpaque = true
		expected, err := CheckRegion(expectedContext, syntax.Block{Span: row.Span, Terminal: row.Expected})
		if err != nil {
			return nil, fmt.Errorf("assertion %s expected completion: %w", row.Name.Text, err)
		}
		result = append(result, &ir.Assertion{Root: root, Actual: actual, Expected: expected})
	}
	return result, nil
}

// A basic table belongs to one checked invocation. The complete concurrent
// scheduler, participant paths and captured callable instances are I18's pass.
func (c *regionChecker) fixtures(rows []syntax.Assertion, step *ir.InvocationStep, scope bodyScope) (*ir.FixtureTable, error) {
	if step.Site == "" {
		return nil, fmt.Errorf("fixture requires a checked lexical call site")
	}
	table := &ir.FixtureTable{Identity: step.Site + "/when"}
	var receiver *ir.Expression
	if step.Receiver {
		receiver = step.Arguments[0]
	}
	for _, row := range rows {
		if row.Receiver != nil || row.Name.Text == "" {
			return nil, fmt.Errorf("fixture receiver is implicit and selector must be named")
		}
		prepared, args, err := c.arguments(c.expressions(scope), ValueBinding{Identity: step.Identity, Type: step.Contract}, row.Arguments, receiver)
		if err != nil {
			return nil, fmt.Errorf("fixture %s arguments: %w", row.Name.Text, err)
		}
		switch row.Expected.(type) {
		case *syntax.SuccessBody, *syntax.FailureBody:
		default:
			return nil, fmt.Errorf("fixture requires an explicit completion")
		}
		savedContext, savedResult, savedEscapes := c.context, c.region.Result, c.region.Escapes
		bound, err := c.context.Registry.Bound(step.Errors)
		if err != nil {
			return nil, err
		}
		c.context.Result = step.Result
		c.context.Errors = bound
		c.region.Result = step.Result
		expected, err := c.completion(row.Expected, scope)
		c.context, c.region.Result, c.region.Escapes = savedContext, savedResult, savedEscapes
		if err != nil {
			return nil, fmt.Errorf("fixture %s completion: %w", row.Name.Text, err)
		}
		table.Rows = append(table.Rows, ir.FixtureRow{Selector: row.Name.Text, Prepare: prepared, Arguments: args, Expected: expected})
	}
	return table, nil
}

func validateAssertionNames(declaration *syntax.FunctionDecl) error {
	if len(declaration.Assertions) == 0 {
		return fmt.Errorf("%s requires mandatory assertions", declaration.Name.Text)
	}
	seen := map[string]bool{}
	for _, row := range declaration.Assertions {
		if row.Name.Text == "" || seen[row.Name.Text] {
			return fmt.Errorf("duplicate or missing assertion name in %s", declaration.Name.Text)
		}
		seen[row.Name.Text] = true
	}
	return nil
}

// A generic assertion is its own application, independent of instances reached
// from executable bodies. Its expected success value can constrain the result.
func (c *programChecker) genericAssertions(files []*resolve.File) error {
	for _, file := range files {
		for _, node := range file.Source.Syntax.Declarations {
			d, ok := node.(*syntax.FunctionDecl)
			if !ok || len(d.Parameters) == 0 {
				continue
			}
			symbol := file.Package.Scope.Symbols[d.Name.Text]
			for _, row := range d.Assertions {
				elided := make([]bool, len(d.Inputs))
				for i, input := range d.Inputs {
					elided[i] = !input.Variadic && c.scopeElided(file, input.Type)
				}
				expanded, err := expandScopeArguments(len(d.Inputs), func(i int) bool { return elided[i] }, len(d.Inputs) > 0 && d.Inputs[len(d.Inputs)-1].Variadic, row.Arguments, row.Span)
				if err != nil {
					return fmt.Errorf("generic assertion %s: %w", row.Name.Text, err)
				}
				constraints, err := genericArguments(d.Inputs, expanded)
				if err != nil {
					return fmt.Errorf("generic assertion %s: %w", row.Name.Text, err)
				}
				if d.Receiver != nil {
					if row.Receiver == nil {
						if !c.scopeElided(file, d.Receiver.Type) {
							return fmt.Errorf("method assertion requires a receiver")
						}
						constraints = append(constraints, argumentConstraint{d.Receiver.Type, &syntax.ScopeExpr{ExpressionLocation: syntax.ExpressionLocation{Span: row.Span}}})
					} else {
						if c.scopeElided(file, d.Receiver.Type) {
							return fmt.Errorf("generic assertion %s: ingress scope receivers are harness-supplied", row.Name.Text)
						}
						constraints = append(constraints, argumentConstraint{d.Receiver.Type, row.Receiver})
					}
				}
				if success, ok := row.Expected.(*syntax.SuccessBody); ok && success.Value != nil {
					if c.scopeElided(file, d.Result) {
						return fmt.Errorf("generic assertion %s: ingress scope results cannot be named in expected completions", row.Name.Text)
					}
					constraints = append(constraints, argumentConstraint{d.Result, success.Value})
				}
				provisional := CompletionContext{Sites: indexLexicalSites(symbol.ID, d), Identity: symbol.ID + "/assert/inference", Scope: file.Scope, Expressions: c.expressions(file, file.Scope), Callables: c.callables, Variadic: c.variadic}

				provisional.IntrinsicIdentity = func(scope *resolve.Scope, name syntax.QualifiedName) string {
					symbol, err := file.Lookup(scope, name, resolve.CallUse)
					if err != nil {
						return ""
					}
					return symbol.ID
				}
				provisional.CatalogueType = c.catalogueType
				provisional.InferCallback = func(scope *resolve.Scope, name syntax.QualifiedName, inputs []*types.Type, result *types.Type, e *Expressions) (ValueBinding, bool, error) {
					return c.inferCallback(file, scope, name, inputs, result, e)
				}
				provisional.Type = func(node syntax.TypeNode, allowVoid bool) (*types.Type, error) {
					return c.annotation(file, node, allowVoid)
				}
				provisional.Registry = c.program.Registry
				provisional.File = file.Source.Syntax.Source
				provisional.Specialize = func(scope *resolve.Scope, name syntax.QualifiedName, args []syntax.TypeNode) (ValueBinding, error) {
					return c.specialize(file, scope, name, args)
				}
				provisional.AggregateType = func(variant *types.Type) (*types.Type, error) { return c.aggregateType(file, variant) }
				provisional.ResolveMethod = func(application MethodApplication) (ValueBinding, error) { return c.method(file, application) }
				provisional.InferReference = func(scope *resolve.Scope, name syntax.QualifiedName, expected *types.Type, e *Expressions) (ValueBinding, bool, error) {
					return c.inferReference(file, scope, name, expected, e)
				}
				provisional.InferCall = func(scope *resolve.Scope, name syntax.QualifiedName, args []syntax.Argument, expected *types.Type, e *Expressions) (ValueBinding, bool, error) {
					return c.inferCall(file, scope, name, args, expected, e)
				}
				checker := &regionChecker{context: provisional, region: &ir.Region{ID: provisional.Identity}, locals: map[string]*types.Type{}, uses: LocalUses{Names: map[*syntax.NameExpr]string{}, Captures: map[*syntax.ReferenceExpr][]string{}}}
				arguments, err := c.inferArguments(file, symbol.Parameters, d.Result, nil, constraints, checker.expressions(bodyScope{file.Scope}))
				if err != nil {
					return fmt.Errorf("generic assertion %s in %s: %w", row.Name.Text, symbol.ID, err)
				}
				binding, err := c.instantiateFunction(symbol, d, arguments, applicationSite(file, row.Span.Start))
				if err != nil {
					return err
				}
				fn := c.instances[binding.Identity]
				context, err := c.functionContext(fn)
				if err != nil {
					return err
				}
				rows, err := c.assertionRows(file, fn, context, []syntax.Assertion{row})
				if err != nil {
					return err
				}
				c.program.Assertions = append(c.program.Assertions, rows...)
			}
		}
	}
	return nil
}
