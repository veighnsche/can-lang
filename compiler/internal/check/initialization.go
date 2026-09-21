package check

import (
	"fmt"
	"sort"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// InitialValue is supplied after declaration and annotation resolution. Its
// checker uses the declaring file's import scope and sees all top-level values,
// including forward declarations. No initializer is executed by this pass.
type InitialValue struct {
	Identity, QualifiedName, Source string
	Binding                         syntax.Binding
	Type                            *types.Type
	Checker                         *Expressions
}

// Initialization admits C8's closed AST set, checks concrete expressions, then
// orders the complete graph by dependencies and qualified name among ready nodes.
// namedArms contains only capture-free named declarations proven by the owning
// arm checker, never arbitrary local/callable/resource values.
func Initialization(values []InitialValue, namedArms []ValueBinding) ([]ir.Initializer, error) {
	byID := map[string]InitialValue{}
	names := map[string]bool{}
	arms := map[string]*types.Type{}
	for _, arm := range namedArms {
		if arm.Identity == "" || !types.Equal(arm.Type, arm.Type) || arm.Type.Kind() != types.ChoiceArm || arms[arm.Identity] != nil {
			return nil, fmt.Errorf("invalid capture-free named arm evidence")
		}
		arms[arm.Identity] = arm.Type
	}
	for _, v := range values {
		if v.Identity == "" || v.QualifiedName == "" || v.Source == "" || byID[v.Identity].Identity != "" || names[v.QualifiedName] || arms[v.Identity] != nil {
			return nil, fmt.Errorf("duplicate or missing initialization identity")
		}
		if !types.Equal(v.Type, v.Type) || v.Checker == nil {
			return nil, fmt.Errorf("initialization requires checked declaration types")
		}
		byID[v.Identity] = v
		names[v.QualifiedName] = true
	}
	// Sorting before checking also makes diagnostic selection deterministic.
	ordered := append([]InitialValue(nil), values...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].QualifiedName < ordered[j].QualifiedName })
	nodes := map[string]ir.Initializer{}
	dependencies := map[string]map[string]bool{}
	for _, v := range ordered {
		if err := inertExpression(v.Binding.Value); err != nil {
			return nil, fmt.Errorf("%s at byte %d: %w", v.Source, v.Binding.Value.ExprSpan().Start, err)
		}
		expr, err := v.Checker.Check(v.Binding.Value, v.Type)
		if err != nil {
			return nil, fmt.Errorf("%s initialization: %w", v.QualifiedName, err)
		}
		deps := map[string]bool{}
		var references func(*ir.Expression) error
		references = func(e *ir.Expression) error {
			if e == nil {
				return nil
			}
			if e.Kind == ir.Binding {
				if dep, ok := byID[e.Text]; ok {
					if !types.Equal(e.Type, dep.Type) {
						return fmt.Errorf("inconsistent top-level value type")
					}
					deps[e.Text] = true
				} else if arm := arms[e.Text]; arm == nil || !types.Equal(arm, e.Type) {
					return fmt.Errorf("initializer reference is not a top-level value or capture-free named arm")
				}
			}
			for _, child := range e.Inputs {
				if err := references(child); err != nil {
					return err
				}
			}
			return nil
		}
		if err = references(expr); err != nil {
			return nil, fmt.Errorf("%s: %w", v.QualifiedName, err)
		}
		dependencies[v.Identity] = deps
		nodes[v.Identity] = ir.Initializer{Identity: v.Identity, QualifiedName: v.QualifiedName, Source: v.Source, Span: v.Binding.Value.ExprSpan(), Type: v.Type, Value: expr}
	}
	result := make([]ir.Initializer, 0, len(values))
	emitted := map[string]bool{}
	for len(result) < len(values) {
		found := ""
		for _, v := range ordered {
			if emitted[v.Identity] {
				continue
			}
			ready := true
			for id := range dependencies[v.Identity] {
				if !emitted[id] {
					ready = false
					break
				}
			}
			if ready {
				found = v.Identity
				break
			}
		}
		if found == "" {
			var cycle []string
			for _, v := range ordered {
				if !emitted[v.Identity] {
					cycle = append(cycle, v.QualifiedName)
				}
			}
			return nil, fmt.Errorf("initialization dependency cycle prevents ordering: %v", cycle)
		}
		emitted[found] = true
		result = append(result, nodes[found])
	}
	return result, nil
}

func inertExpression(expr syntax.Expr) error {
	if expr == nil {
		return fmt.Errorf("missing initializer expression")
	}
	var children []syntax.Expr
	switch n := expr.(type) {
	case *syntax.LiteralExpr:
	case *syntax.NameExpr:
		if n.Name.Name == "%" {
			return fmt.Errorf("contextual AI value is not inert initialization")
		}
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
		children = []syntax.Expr{n.Receiver}
		if n.Start != nil {
			children = append(children, n.Start)
		}
		if n.End != nil {
			children = append(children, n.End)
		}
	case *syntax.UpdateExpr:
		children = []syntax.Expr{n.Receiver}
		for _, f := range n.Fields {
			children = append(children, f.Value)
		}
	case *syntax.ArrayExpr:
		for _, a := range n.Elements {
			if a.Group != nil {
				return fmt.Errorf("state groups are not inert initialization")
			}
			children = append(children, a.Value)
		}
	case *syntax.ConstructorExpr:
		for _, a := range n.Arguments {
			if a.Group != nil {
				return fmt.Errorf("state groups are not inert initialization")
			}
			children = append(children, a.Value)
		}
	default:
		return fmt.Errorf("expression %T is outside inert initialization", expr)
	}
	for _, child := range children {
		if err := inertExpression(child); err != nil {
			return err
		}
	}
	return nil
}
