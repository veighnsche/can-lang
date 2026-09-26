package check

import (
	"fmt"
	"slices"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// LocalUses comes from lexical/body resolution. Every visited NameExpr has a
// resolved identity (including function names), and every ReferenceExpr has an
// explicit capture entry, even when empty. Missing evidence is an error, not a
// claim of zero uses/captures. Identity accounting distinguishes shadowed names.
type LocalUses struct {
	Names    map[*syntax.NameExpr]string
	Captures map[*syntax.ReferenceExpr][]string
}

type LocalForwarding struct {
	File     *source.File
	Block    syntax.Block
	Binding  ValueBinding // the final typed binding's resolved identity/type
	Uses     LocalUses
	Checker  *Expressions // lexical scope of that initializer, before the new local
	Expected *types.Type  // terminal result type
}

type UnnecessaryLocal struct {
	Name        string
	Span        source.Span
	Replacement string
}

func (d *UnnecessaryLocal) Error() string {
	return fmt.Sprintf("unnecessary local %s at byte %d; replace with %s", d.Name, d.Span.Start, d.Replacement)
}

// CheckLocalForwarding checks exactly C8's four predicates. It diagnoses; it does
// not optimize or alter evaluation. The owning region pass calls this for each
// block with a value/success terminal after resolving its complete lexical scope.
func CheckLocalForwarding(context LocalForwarding) error {
	block := context.Block
	if len(block.Steps) == 0 {
		return nil
	}
	local, ok := block.Steps[len(block.Steps)-1].(*syntax.BindingStep)
	if !ok {
		return nil
	}
	var value syntax.Expr
	prefix := ""
	switch terminal := block.Terminal.(type) {
	case *syntax.SuccessBody:
		value = terminal.Value
		prefix = "ok "
	case *syntax.ValueBody:
		value = terminal.Value
	default:
		return nil
	}
	name, ok := value.(*syntax.NameExpr)
	if !ok {
		return nil
	}
	if name.Name.Package != "" || name.Name.Name != local.Name.Text {
		return nil
	}
	if context.Binding.Identity == "" || !types.Equal(context.Binding.Type, context.Binding.Type) || context.Checker == nil || context.File == nil || !types.Equal(context.Expected, context.Expected) {
		return fmt.Errorf("local forwarding requires complete checked context")
	}
	if !types.Assignable(context.Binding.Type, context.Expected) {
		return fmt.Errorf("local terminal does not fit expected type")
	}
	target, exists := context.Uses.Names[name]
	if !exists || target == "" {
		return fmt.Errorf("missing resolved terminal name")
	}
	if target != context.Binding.Identity {
		return nil
	}
	uses, captures, err := countLocalUses(block, context.Uses, context.Binding.Identity)
	if err != nil {
		return err
	}
	if uses != 1 || captures {
		return nil
	}
	if !simpleLocalSyntax(local.Value) {
		return nil
	}
	original, err := context.Checker.Check(local.Value, context.Binding.Type)
	if err != nil {
		return err
	}
	if !nonFaultingLocalIR(original) {
		return nil
	}
	// A failed substituted check means the local was semantically useful. Its
	// annotation or scope must be preserved, not diagnosed as another static error.
	substituted, err := context.Checker.Check(local.Value, context.Expected)
	if err != nil || !types.Equal(substituted.Type, context.Binding.Type) || !sameTyping(original, substituted) {
		return nil
	}
	replacement, err := context.File.Slice(local.Value.ExprSpan())
	if err != nil {
		return err
	}
	return &UnnecessaryLocal{Name: local.Name.Text, Span: local.Span, Replacement: prefix + replacement}
}
func simpleLocalSyntax(expr syntax.Expr) bool {
	if expr == nil {
		return false
	}
	var children []syntax.Expr
	switch n := expr.(type) {
	case *syntax.LiteralExpr:
	case *syntax.NameExpr:
		if n.Name.Name == "%" {
			return false
		}
	case *syntax.GroupExpr:
		children = []syntax.Expr{n.Value}
	case *syntax.UnaryExpr:
		children = []syntax.Expr{n.Operand}
	case *syntax.BinaryExpr:
		if slices.Contains([]string{"/", "%", "**", "<<", ">>"}, n.Operator) {
			return false
		}
		children = []syntax.Expr{n.Left, n.Right}
	case *syntax.ComparisonExpr:
		children = n.Operands
	case *syntax.FieldExpr:
		children = []syntax.Expr{n.Receiver}
	default:
		return false
	}
	for _, child := range children {
		if !simpleLocalSyntax(child) {
			return false
		}
	}
	return true
}
func nonFaultingLocalIR(expr *ir.Expression) bool {
	if expr == nil {
		return false
	}
	switch expr.Kind {
	case ir.Literal, ir.Binding, ir.Unary, ir.Binary, ir.Comparison, ir.Field, ir.Length, ir.StandardProjection:
	default:
		return false
	}
	for _, child := range expr.Inputs {
		if !nonFaultingLocalIR(child) {
			return false
		}
	}
	return true
}

// This is a finite equality check on typing evidence, not a proof of behavioral
// equivalence. Comparing every typed node also detects an expected-type change
// below a bool comparison whose final result type happens to stay bool.
func sameTyping(a, b *ir.Expression) bool {
	if a == nil || b == nil {
		return a == b
	}
	if a.Kind != b.Kind || a.Text != b.Text || !types.Equal(a.Type, b.Type) || len(a.Inputs) != len(b.Inputs) || !slices.Equal(a.Operators, b.Operators) || !slices.Equal(a.Equality, b.Equality) || !slices.Equal(a.Spread, b.Spread) || !slices.Equal(a.Fields, b.Fields) {
		return false
	}
	for i := range a.Inputs {
		if !sameTyping(a.Inputs[i], b.Inputs[i]) {
			return false
		}
	}
	return true
}

func countLocalUses(block syntax.Block, evidence LocalUses, target string) (int, bool, error) {
	count := 0
	captured := false
	var expr func(syntax.Expr) error
	var body func(syntax.Body) error
	var visitBlock func(syntax.Block) error
	var match func(syntax.Match) error
	var coordination func(syntax.Coordination) error
	args := func(arguments []syntax.Argument) error {
		for _, a := range arguments {
			if a.Value != nil {
				if err := expr(a.Value); err != nil {
					return err
				}
			}
			if a.Group != nil {
				for _, v := range a.Group.Values {
					if err := expr(v); err != nil {
						return err
					}
				}
			}
		}
		return nil
	}
	expr = func(value syntax.Expr) error {
		if value == nil {
			return nil
		}
		var children []syntax.Expr
		switch n := value.(type) {
		case *syntax.LiteralExpr, *syntax.ProbabilityExpr:
		case *syntax.NameExpr:
			id, ok := evidence.Names[n]
			if !ok || id == "" {
				return fmt.Errorf("missing resolved source use at byte %d", n.Span.Start)
			}
			if id == target {
				count++
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
			children = []syntax.Expr{n.Receiver, n.Start, n.End}
		case *syntax.ArrayExpr:
			return args(n.Elements)
		case *syntax.ConstructorExpr:
			return args(n.Arguments)
		case *syntax.UpdateExpr:
			children = []syntax.Expr{n.Receiver}
			for _, f := range n.Fields {
				children = append(children, f.Value)
			}
		case *syntax.CallExpr:
			children = []syntax.Expr{n.Invocation.Callee}
			if err := args(n.Invocation.Arguments); err != nil {
				return err
			}
			for _, m := range n.Methods {
				if err := args(m.Arguments); err != nil {
					return err
				}
			}
		case *syntax.ReferenceExpr:
			ids, ok := evidence.Captures[n]
			if !ok {
				return fmt.Errorf("missing implicit near-capture evidence")
			}
			for _, id := range ids {
				if id == "" {
					return fmt.Errorf("missing resolved implicit capture identity")
				}
				if id == target {
					captured = true
				}
			}
			children = []syntax.Expr{n.Callee}
			for _, pinned := range n.Bindings {
				children = append(children, pinned.Value)
			}
		case *syntax.MatchExpr:
			return match(n.Match)
		case *syntax.CoordinationExpr:
			return coordination(n.Coordination)
		default:
			return fmt.Errorf("unhandled source-use expression %T", value)
		}
		for _, v := range children {
			if err := expr(v); err != nil {
				return err
			}
		}
		return nil
	}
	arms := func(list []syntax.MatchArm) error {
		for _, arm := range list {
			if err := body(arm.Body); err != nil {
				return err
			}
		}
		return nil
	}
	match = func(m syntax.Match) error {
		for _, v := range m.Values {
			if err := expr(v); err != nil {
				return err
			}
		}
		if m.Call != nil {
			if err := expr(m.Call); err != nil {
				return err
			}
		}
		for _, entry := range m.Chain {
			if err := expr(entry.Call); err != nil {
				return err
			}
		}
		for _, a := range m.When {
			if err := expr(a.Receiver); err != nil {
				return err
			}
			if err := args(a.Arguments); err != nil {
				return err
			}
			if err := body(a.Expected); err != nil {
				return err
			}
		}
		return arms(m.Arms)
	}
	coordination = func(c syntax.Coordination) error {
		for _, p := range c.Participants {
			if p.Call != nil {
				if err := expr(p.Call); err != nil {
					return err
				}
			}
			if err := expr(p.Spread); err != nil {
				return err
			}
			if err := arms(p.Arms); err != nil {
				return err
			}
		}
		return arms(c.Arms)
	}
	body = func(value syntax.Body) error {
		switch n := value.(type) {
		case nil:
			return nil
		case *syntax.SuccessBody:
			return expr(n.Value)
		case *syntax.ValueBody:
			return expr(n.Value)
		case *syntax.FailureBody:
			return expr(n.Error)
		case *syntax.RelayBody:
			return expr(n.Call)
		case *syntax.DoBody:
			return visitBlock(n.Block)
		case *syntax.MatchBody:
			return match(n.Match)
		default:
			return fmt.Errorf("unhandled source-use body %T", value)
		}
	}
	visitBlock = func(b syntax.Block) error {
		for _, step := range b.Steps {
			switch n := step.(type) {
			case *syntax.BindingStep:
				if err := expr(n.Value); err != nil {
					return err
				}
			case *syntax.CallStep:
				if err := expr(n.Call); err != nil {
					return err
				}
			case *syntax.CoordinationStep:
				if err := coordination(n.Coordination); err != nil {
					return err
				}
			default:
				return fmt.Errorf("unhandled source-use step %T", step)
			}
		}
		return body(b.Terminal)
	}
	err := visitBlock(block)
	return count, captured, err
}
