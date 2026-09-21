package emit

import (
	"fmt"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
)

// Authored expressions cannot name the private success bindings connecting chain
// steps. Only compiler-created receiver expressions and their preparation aliases
// depend on those bindings. All written arguments remain prelaunch work.
func receiverDependent(value *ir.Expression, dependent map[string]bool) bool {
	if value == nil {
		return false
	}
	if value.Kind == ir.Binding && dependent[value.Text] {
		return true
	}
	for _, input := range value.Inputs {
		if receiverDependent(input, dependent) {
			return true
		}
	}
	return false
}

func (e *RegionEmitter) prepareParticipant(call *ir.Invocation) (LoweredExpression, []string, error) {
	if call == nil || len(call.Steps) == 0 {
		return LoweredExpression{}, nil, fmt.Errorf("missing checked participant invocation")
	}
	var out strings.Builder
	var captures []string
	dependent := map[string]bool{}
	prepared := *call
	prepared.Steps = append([]ir.InvocationStep(nil), call.Steps...)
	hoist := func(value *ir.Expression) (*ir.Expression, error) {
		if value == nil || receiverDependent(value, dependent) {
			return value, nil
		}
		lowered, err := e.expression.Lower(value)
		if err != nil {
			return nil, err
		}
		out.WriteString(lowered.Statements)
		name := e.temp()
		fmt.Fprintf(&out, "const %s = %s;\n", name, lowered.Value)
		identity := name + "/participant-preparation"
		e.expression.Bindings[identity] = name
		captures = append(captures, name)
		return &ir.Expression{Kind: ir.Binding, Type: value.Type, Span: value.Span, Text: identity}, nil
	}
	for i := range prepared.Steps {
		step := &prepared.Steps[i]
		if step.Fixtures != nil {
			return LoweredExpression{}, nil, fmt.Errorf("coordination participant fixture belongs inside a named wrapper")
		}
		var err error
		if step.Callee != nil {
			step.Callee, err = hoist(step.Callee)
			if err != nil {
				return LoweredExpression{}, nil, err
			}
		} else if step.Native == nil {
			target, err := e.target(step.Identity)
			if err != nil {
				return LoweredExpression{}, nil, err
			}
			name := e.temp()
			fmt.Fprintf(&out, "const %s = %s;\n", name, target)
			e.expression.Bindings[name] = name
			step.Callee = &ir.Expression{Kind: ir.Binding, Type: step.Contract, Text: name, Span: step.Span}
			captures = append(captures, name)
		}
		step.Prepare = nil
		for _, binding := range call.Steps[i].Prepare {
			if receiverDependent(binding.Value, dependent) {
				dependent[binding.Local.Identity] = true
				step.Prepare = append(step.Prepare, binding)
				continue
			}
			value, err := hoist(binding.Value)
			if err != nil {
				return LoweredExpression{}, nil, err
			}
			e.expression.Bindings[binding.Local.Identity] = e.expression.Bindings[value.Text]
		}
		step.Arguments = append([]*ir.Expression(nil), step.Arguments...)
		for index, argument := range step.Arguments {
			step.Arguments[index], err = hoist(argument)
			if err != nil {
				return LoweredExpression{}, nil, err
			}
		}
		if step.Native != nil {
			native := *step.Native
			native.Inputs = append([]*ir.Expression(nil), native.Inputs...)
			for index, input := range native.Inputs {
				native.Inputs[index], err = hoist(input)
				if err != nil {
					return LoweredExpression{}, nil, err
				}
			}
			step.Native = &native
		}
		dependent[step.SuccessBinding] = true
	}
	invocation, err := e.invocation(&prepared)
	if err != nil {
		return LoweredExpression{}, nil, err
	}
	origin := ""
	if e.SourceID != "" {
		origin = "let $canOrigin = " + e.origin(call.Span) + ";\n"
	}
	return LoweredExpression{Statements: out.String(), Value: "async () => {\n" + origin + invocation.Statements + "return " + invocation.Value + ";\n}"}, captures, nil
}
