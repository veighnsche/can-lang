package check

import (
	"fmt"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"math/big"
)

// ConnectionDeclaration converts inert literal syntax into the shared I15 policy.
// It deliberately never evaluates a call, reads the environment, or opens I/O.
func ConnectionDeclaration(declaration *syntax.ConnectionDecl) (ConnectionPolicy, error) {
	var convert func(syntax.ConnectionSetting) (ConnectionSetting, error)
	convert = func(setting syntax.ConnectionSetting) (ConnectionSetting, error) {
		out := ConnectionSetting{Name: setting.Name.Text}
		for _, entry := range setting.Entries {
			converted, err := convert(entry)
			if err != nil {
				return out, err
			}
			out.Entries = append(out.Entries, converted)
		}
		if setting.Value == nil {
			return out, nil
		}
		literal, ok := setting.Value.(*syntax.LiteralExpr)
		if ok && literal.Token.Kind == syntax.String {
			value := literal.Token.Value
			out.Text = &value
			return out, nil
		}
		integer, err := connectionInteger(setting.Value)
		if err != nil {
			return out, fmt.Errorf("connection %s setting %s: %w", declaration.Name.Text, setting.Name.Text, err)
		}
		out.Integer = integer
		return out, nil
	}
	var settings []ConnectionSetting
	for _, setting := range declaration.Settings {
		converted, err := convert(setting)
		if err != nil {
			return ConnectionPolicy{}, err
		}
		settings = append(settings, converted)
	}
	return CheckConnection(settings)
}
func connectionInteger(expression syntax.Expr) (*big.Int, error) {
	switch n := expression.(type) {
	case *syntax.LiteralExpr:
		if n.Token.Kind == syntax.Integer {
			if value, ok := new(big.Int).SetString(n.Token.Text, 10); ok {
				return value, nil
			}
		}
	case *syntax.GroupExpr:
		return connectionInteger(n.Value)
	case *syntax.UnaryExpr:
		value, err := connectionInteger(n.Operand)
		if err != nil {
			return nil, err
		}
		if n.Operator == "-" {
			return value.Neg(value), nil
		}
		if n.Operator == "+" {
			return value, nil
		}
	case *syntax.BinaryExpr:
		left, err := connectionInteger(n.Left)
		if err != nil {
			return nil, err
		}
		right, err := connectionInteger(n.Right)
		if err != nil {
			return nil, err
		}
		switch n.Operator {
		case "+":
			return left.Add(left, right), nil
		case "-":
			return left.Sub(left, right), nil
		case "*":
			return left.Mul(left, right), nil
		case "/":
			if right.Sign() != 0 {
				return left.Quo(left, right), nil
			}
		case "%":
			if right.Sign() != 0 {
				return left.Rem(left, right), nil
			}
		}
	}
	return nil, fmt.Errorf("expected literal text or a valid integer constant")
}
