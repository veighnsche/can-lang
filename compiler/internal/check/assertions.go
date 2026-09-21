package check

import (
	"fmt"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
)

// Assertions checks every row before execution. The actual side is a checked
// invocation of the declared function, not a copy of its expected completion.
// Receiver/variadic input rules therefore use the ordinary invocation checker.
func (c *programChecker) assertions(file *resolve.File, fn *ProgramFunction, context CompletionContext) ([]*ir.Assertion, error) {
	declaration := fn.Symbol.Declaration.(*syntax.FunctionDecl)
	if err := validateAssertionNames(declaration); err != nil {
		return nil, err
	}
	var result []*ir.Assertion
	for _, row := range declaration.Assertions {
		root := ir.AssertionRoot{Package: fn.Symbol.Package.ID, Declaration: fn.Symbol.ID, Name: row.Name.Text}
		var callee syntax.Expr = &syntax.NameExpr{Name: syntax.QualifiedName{Name: declaration.Name.Text}, ExpressionLocation: syntax.ExpressionLocation{Span: row.Span}}
		if declaration.Receiver != nil {
			if row.Receiver == nil {
				return nil, fmt.Errorf("method assertion requires a receiver")
			}
			callee = &syntax.FieldExpr{Receiver: row.Receiver, Field: declaration.Name, ExpressionLocation: syntax.ExpressionLocation{Span: row.Span}}
		} else if row.Receiver != nil {
			return nil, fmt.Errorf("ordinary function assertion cannot supply a receiver")
		}
		call := &syntax.CallExpr{ExpressionLocation: syntax.ExpressionLocation{Span: row.Span}, Invocation: syntax.Invocation{Span: row.Span, Callee: callee, Arguments: row.Arguments}}
		actualContext := context
		actualContext.Identity = fn.Symbol.ID + "/assert/" + row.Name.Text + "/actual"
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
	table := &ir.FixtureTable{Identity: c.identity("when")}
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
