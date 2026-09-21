package check

import (
	"errors"
	"fmt"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

type aggregateInference struct {
	identity  string
	scope     *resolve.Scope
	typ       *types.Type
	errors    []*types.Type
	discovery bool
	deferred  bool
	observed  bool
}
type deferredAggregate struct{}

func (*deferredAggregate) Error() string {
	return "all_failed.failures requires one expected named failure variant"
}
func (c *regionChecker) deferAggregate(err error) bool {
	var deferred *deferredAggregate
	if c.aggregate == nil || !c.aggregate.discovery || !errors.As(err, &deferred) {
		return false
	}
	c.aggregate.deferred = true
	return true
}

// Discovery never invents a type. A deferred expression with an authored
// expected type may provisionally use that type so later arguments/arms are
// visited. For an untyped operand, inspect its same-scope subexpressions for
// concrete named-call constraints and retry only after a specialization exists.
// The provisional expressions are discarded by the final complete handler pass.
func (c *regionChecker) discoverAggregateConstraints(node syntax.Expr, expected *types.Type, checker *Expressions, original error) (*ir.Expression, error) {
	if !c.deferAggregate(original) {
		return nil, original
	}
	var children []syntax.Expr
	arguments := func(args []syntax.Argument) {
		for _, arg := range args {
			if arg.Group != nil {
				children = append(children, arg.Group.Values...)
			} else {
				children = append(children, arg.Value)
			}
		}
	}
	switch n := node.(type) {
	case *syntax.GroupExpr:
		children = []syntax.Expr{n.Value}
	case *syntax.UnaryExpr:
		children = []syntax.Expr{n.Operand}
	case *syntax.BinaryExpr:
		children = []syntax.Expr{n.Left, n.Right}
	case *syntax.ComparisonExpr:
		children = n.Operands
	case *syntax.FieldExpr:
		children = []syntax.Expr{n.Receiver}
	case *syntax.IndexExpr:
		children = []syntax.Expr{n.Receiver, n.Index}
	case *syntax.SliceExpr:
		children = []syntax.Expr{n.Receiver, n.Start, n.End}
	case *syntax.ArrayExpr:
		arguments(n.Elements)
	case *syntax.ConstructorExpr:
		arguments(n.Arguments)
	case *syntax.CallExpr:
		children = append(children, n.Invocation.Callee)
		arguments(n.Invocation.Arguments)
		for _, method := range n.Methods {
			arguments(method.Arguments)
		}
	case *syntax.UpdateExpr:
		children = append(children, n.Receiver)
		for _, field := range n.Fields {
			children = append(children, field.Value)
		}
	}
	for _, child := range children {
		if child != nil {
			// Only constraint discovery occurs here. The complete second pass
			// checks every error and all original expected-type relationships.
			_, _ = checker.Check(child, nil)
		}
	}
	if c.aggregate.typ != nil {
		return checker.expression(node, expected)
	}
	if expected != nil {
		return &ir.Expression{Span: node.ExprSpan(), Type: expected}, nil
	}
	return nil, original
}
func (c *programChecker) aggregateType(file *resolve.File, variant *types.Type) (*types.Type, error) {
	return c.specializer.Resolve(file, &syntax.NamedType{Name: syntax.QualifiedName{Name: "all_failed"}, Arguments: []syntax.TypeNode{named("aggregate_failure")}}, map[string]*types.Type{"aggregate_failure": variant}, false)
}
func (c *regionChecker) aggregateField(node *syntax.FieldExpr, expected *types.Type, scope bodyScope) (*ir.Expression, bool, error) {
	a := c.aggregate
	if a == nil {
		return nil, false, nil
	}
	receiverNode := node.Receiver
	for {
		group, ok := receiverNode.(*syntax.GroupExpr)
		if !ok {
			break
		}
		receiverNode = group.Value
	}
	receiver, ok := receiverNode.(*syntax.NameExpr)
	if !ok || receiver.Name.Package != "" || receiver.Name.Name != "all_failed" {
		return nil, false, nil
	}
	// A nested ordinary domain arm may shadow this aggregate alias. An alias
	// outside this handler must not capture the new race's all_failed field.
	for current := scope.symbols; current != nil; current = current.Parent {
		if symbol := current.Symbols[receiver.Name.Name]; symbol != nil && c.locals[symbol.ID] != nil {
			return nil, false, nil
		}
		if current == a.scope {
			break
		}
	}
	a.observed = true
	if node.Field.Text != "failures" {
		return nil, true, fmt.Errorf("unknown all_failed field %s", node.Field.Text)
	}
	if expected != nil {
		if expected.Kind() != types.Array || expected.Element().Kind() != types.Variant {
			return nil, true, fmt.Errorf("all_failed.failures requires an expected named variant array")
		}
		variant := expected.Element()
		if a.typ != nil && !types.Equal(a.typ.Arguments()[0], variant) {
			return nil, true, fmt.Errorf("conflicting expected all_failed failure variants")
		}
		if a.typ == nil {
			if c.context.AggregateType == nil {
				return nil, true, fmt.Errorf("missing aggregate specialization resolver")
			}
			typ, err := c.context.AggregateType(variant)
			if err != nil {
				return nil, true, err
			}
			if err = c.aggregateCoverage(typ, a.errors); err != nil {
				return nil, true, err
			}
			a.typ = typ
		}
	}
	if a.typ == nil {
		return nil, true, &deferredAggregate{}
	}
	c.uses.Names[receiver] = a.identity
	binding := &ir.Expression{Kind: ir.Binding, Span: receiver.Span, Type: a.typ, Text: a.identity}
	return &ir.Expression{Kind: ir.Field, Span: node.Span, Type: a.typ.Fields()[0].Type, Text: "failures", Inputs: []*ir.Expression{binding}}, true, nil
}
func (c *regionChecker) observedAggregateHandler(arm *syntax.MatchArm, failures []*types.Type, result *types.Type, scope bodyScope) (*ir.OutcomeHandler, *types.Type, error) {
	state := &aggregateInference{identity: c.identity("all_failed"), errors: failures, discovery: true}
	// Discovery may defer a contextless initializer while retaining its authored
	// binding type. That provisional IR is discarded. One final full check uses
	// the unique sealed specialization; no guessed type can reach emission.
	check := func() (*ir.OutcomeHandler, error) {
		region := &ir.Region{ID: c.identity("aggregate-handler"), Parent: c.region.ID, Source: c.region.Source, Span: arm.Span, Kind: ir.HandlerRegion, Result: result, Errors: append([]*types.Type(nil), c.region.Errors...)}
		context := c.context
		context.Identity = region.ID
		context.Parent = region.Parent
		context.Kind = ir.HandlerRegion
		context.Result = result
		child := &regionChecker{aggregate: state, context: context, region: region, locals: map[string]*types.Type{}, uses: c.uses}
		for id, typ := range c.locals {
			child.locals[id] = typ
		}
		handlerScope := child.child(scope)
		state.scope = handlerScope.symbols
		body, err := child.completion(arm.Body, handlerScope)
		if err != nil && !child.deferAggregate(err) {
			return nil, err
		}
		checked := ir.Arm{Span: arm.Span, Outcome: "ok", Body: body}
		if state.typ != nil {
			checked.Outcome = "domain"
			checked.Error = state.typ
			checked.Binding = &ir.Local{Identity: state.identity, Type: state.typ}
		}
		return &ir.OutcomeHandler{Region: region, Arms: []ir.Arm{checked}}, nil
	}
	handler, err := check()
	if err != nil {
		return nil, nil, err
	}
	if state.typ == nil {
		if state.observed || state.deferred {
			return nil, nil, &deferredAggregate{}
		}
		return handler, nil, nil
	}
	state.discovery = false
	state.deferred = false
	handler, err = check()
	if err != nil {
		return nil, nil, err
	}
	return handler, state.typ, nil
}
