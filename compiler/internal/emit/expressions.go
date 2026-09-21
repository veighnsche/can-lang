package emit

import (
	"fmt"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

type LoweredExpression struct{ Statements, Value string }

// ExpressionEmitter preserves evaluation order with statements, so the owning
// region may later insert awaited, boxed call lowering at Call without introducing
// hidden async IIFEs or exposing payloads to promise assimilation.
type ExpressionEmitter struct {
	Bindings map[string]string
	Call     func(identity string, arguments []string) (LoweredExpression, error)
	serial   int
}

func PrimitiveImports(path string) string {
	return "import { intDivide as $canIntDivide, intRemainder as $canIntRemainder, intPower as $canIntPower, index as $canIndex, slice as $canSlice } from " + quote(path) + ";\n"
}
func (e *ExpressionEmitter) temporary() string {
	e.serial++
	return fmt.Sprintf("$canExpr%d", e.serial)
}
func (e *ExpressionEmitter) Lower(node *ir.Expression) (LoweredExpression, error) {
	if node == nil || !types.Equal(node.Type, node.Type) {
		return LoweredExpression{}, fmt.Errorf("emission requires a checked expression")
	}
	var statements strings.Builder
	bind := func(code string) LoweredExpression {
		name := e.temporary()
		fmt.Fprintf(&statements, "const %s = %s;\n", name, code)
		return LoweredExpression{statements.String(), name}
	}
	input := func(n *ir.Expression) (string, error) {
		lowered, err := e.Lower(n)
		if err != nil {
			return "", err
		}
		statements.WriteString(lowered.Statements)
		return lowered.Value, nil
	}
	switch node.Kind {
	case ir.Literal:
		value := node.Text
		switch node.Type.Declaration() {
		case "int":
			value += "n"
		case "str":
			value = quote(value)
		case "float":
			value = "Number(" + quote(value) + ")"
		}
		return bind(value), nil
	case ir.Binding:
		value, ok := e.Bindings[node.Text]
		if !ok {
			return LoweredExpression{}, fmt.Errorf("missing emitted binding %s", node.Text)
		}
		return bind(value), nil
	case ir.Unary:
		value, err := input(node.Inputs[0])
		if err != nil {
			return LoweredExpression{}, err
		}
		op := node.Text
		if op == "not" {
			op = "!"
		}
		return bind(op + "(" + value + ")"), nil
	case ir.Binary:
		left, err := input(node.Inputs[0])
		if err != nil {
			return LoweredExpression{}, err
		}
		if node.Text == "and" || node.Text == "or" {
			result := e.temporary()
			fmt.Fprintf(&statements, "let %s = %s;\n", result, left)
			right, err := e.Lower(node.Inputs[1])
			if err != nil {
				return LoweredExpression{}, err
			}
			condition := result
			if node.Text == "or" {
				condition = "!" + result
			}
			fmt.Fprintf(&statements, "if (%s) {\n%s%s = %s;\n}\n", condition, right.Statements, result, right.Value)
			return LoweredExpression{statements.String(), result}, nil
		}
		right, err := input(node.Inputs[1])
		if err != nil {
			return LoweredExpression{}, err
		}
		code := "(" + left + ") " + node.Text + " (" + right + ")"
		if node.Type.Declaration() == "int" {
			switch node.Text {
			case "/":
				code = "$canIntDivide(" + left + ", " + right + ")"
			case "%":
				code = "$canIntRemainder(" + left + ", " + right + ")"
			case "**":
				code = "$canIntPower(" + left + ", " + right + ")"
			}
		}
		return bind(code), nil
	case ir.Comparison:
		first, err := input(node.Inputs[0])
		if err != nil {
			return LoweredExpression{}, err
		}
		result := e.temporary()
		fmt.Fprintf(&statements, "let %s = true;\n", result)
		var pairs func(int, string) error
		pairs = func(i int, previous string) error {
			right, err := e.Lower(node.Inputs[i+1])
			if err != nil {
				return err
			}
			statements.WriteString(right.Statements)
			op := node.Operators[i]
			code := "(" + previous + ") " + op + " (" + right.Value + ")"
			if op == "is" || op == "is not" {
				switch node.Equality[i] {
				case ir.FloatIdentity:
					code = "Object.is(" + previous + ", " + right.Value + ")"
				case ir.Deep:
					code = "Bun.deepEquals(" + previous + ", " + right.Value + ", true)"
				default:
					code = "(" + previous + " === " + right.Value + ")"
				}
				if op == "is not" {
					code = "!(" + code + ")"
				}
			}
			fmt.Fprintf(&statements, "%s = %s;\n", result, code)
			if i+1 < len(node.Operators) {
				fmt.Fprintf(&statements, "if (%s) {\n", result)
				if err = pairs(i+1, right.Value); err != nil {
					return err
				}
				statements.WriteString("}\n")
			}
			return nil
		}
		if err = pairs(0, first); err != nil {
			return LoweredExpression{}, err
		}
		return LoweredExpression{statements.String(), result}, nil
	case ir.Index, ir.Slice, ir.Length, ir.Field, ir.Array, ir.Call, ir.Record, ir.Update:
		values := make([]string, len(node.Inputs))
		for i, n := range node.Inputs {
			if n == nil {
				values[i] = "undefined"
				continue
			}
			v, err := input(n)
			if err != nil {
				return LoweredExpression{}, err
			}
			values[i] = v
		}
		switch node.Kind {
		case ir.Record, ir.Update:
			entries := make([]string, len(node.Fields))
			offset := 0
			if node.Kind == ir.Update {
				offset = 1
			}
			for i, name := range node.Fields {
				entries[i] = "[" + quote(name) + ", " + values[i+offset] + "]"
			}
			if node.Kind == ir.Update {
				return bind("$canUpdate(" + values[0] + ", [" + strings.Join(entries, ", ") + "] )"), nil
			}
			return bind("$canRecord(" + quote(node.Text) + ", [" + strings.Join(entries, ", ") + "] )"), nil
		case ir.Index:
			return bind("$canIndex(" + strings.Join(values, ", ") + ")"), nil
		case ir.Slice:
			return bind("$canSlice(" + strings.Join(values, ", ") + ")"), nil
		case ir.Length:
			return bind("BigInt(" + values[0] + ".length)"), nil
		case ir.Field:
			return bind(values[0] + "[" + quote(node.Text) + "]"), nil
		case ir.Array:
			for i, spread := range node.Spread {
				if spread {
					values[i] = "..." + values[i]
				}
			}
			return bind("Object.freeze([" + strings.Join(values, ", ") + "])"), nil
		case ir.Call:
			if e.Call == nil {
				return LoweredExpression{}, fmt.Errorf("call lowering requires its owning region")
			}
			call, err := e.Call(node.Text, values)
			if err != nil {
				return LoweredExpression{}, err
			}
			statements.WriteString(call.Statements)
			return bind(call.Value), nil
		}
	}
	return LoweredExpression{}, fmt.Errorf("unknown expression IR %s", node.Kind)
}
