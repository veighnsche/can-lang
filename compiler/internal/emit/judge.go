package emit

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
)

// NoulPreparation emits only preparation. No provider or handler is reachable
// from this function, and the backend descriptor also crosses await in a box.
func (e *RegionEmitter) NoulPreparation(name string, question *ir.Noul) (string, error) {
	if question == nil || len(question.Options) != 2 || !jsBinding.MatchString(name) {
		return "", fmt.Errorf("invalid Noul phase plan")
	}
	region := &ir.Region{ID: question.Identity, Source: question.Source, Span: question.Span, Inputs: question.Inputs}
	args, err := e.configure(region)
	if err != nil {
		return "", err
	}
	var body strings.Builder
	lower := func(expression *ir.Expression) (string, error) {
		value, err := e.expression.Lower(expression)
		if err != nil {
			return "", err
		}
		body.WriteString(value.Statements)
		return value.Value, nil
	}
	instructions, err := lower(question.Instructions)
	if err != nil {
		return "", err
	}
	var yes, no string
	for _, option := range question.Options {
		value, err := lower(option.Description)
		if err != nil {
			return "", err
		}
		if option.True {
			yes = value
		} else {
			no = value
		}
	}
	minimum, err := lower(question.Minimum)
	if err != nil {
		return "", err
	}
	if yes == "" || no == "" {
		return "", fmt.Errorf("incomplete Noul option labels")
	}
	fmt.Fprintf(&body, "return $canSuccess(Object.freeze({instructions:%s,trueDescription:%s,falseDescription:%s,minimum:%s}));\n", instructions, yes, no, minimum)
	return fmt.Sprintf("async function %s(%s): Promise<$canCompletion<$canNoulDescriptor>> {\nlet $canOrigin = %s;\ntry {\n%s} catch ($canCause) { return $canCaught($canCause, $canOrigin); }\n}\n", name, strings.Join(args, ", "), e.origin(region.Span), body.String()), nil
}

// Judge emits registration-order preparation followed by one validated request,
// all selected handlers, and finally the ordinary continuation region.
func (e *RegionEmitter) Judge(name string, judge *ir.Judge, questions map[string]*ir.Noul, names map[string]string, connection, model string) (string, error) {
	if judge == nil || judge.Continuation == nil || !jsBinding.MatchString(name) {
		return "", fmt.Errorf("invalid judge phase plan")
	}
	region := &ir.Region{ID: judge.Identity, Source: judge.Source, Span: judge.Span, Inputs: judge.Inputs, Result: judge.Continuation.Result}
	args, err := e.configure(region)
	if err != nil {
		return "", err
	}
	var body strings.Builder
	type prepared struct {
		question   *ir.Noul
		arguments  []string
		descriptor string
	}
	var fields []string
	for i, input := range judge.StateInputs {
		for _, node := range judge.State.Nodes {
			if node.Identity == judge.State.Root {
				fields = append(fields, "["+quote(node.Fields[i].Name)+","+e.expression.Bindings[input.Identity]+"]")
			}
		}
	}
	stateValue := e.temp()
	fmt.Fprintf(&body, "const %s = $canRecord(%s,[%s]);\n", stateValue, quote(judge.State.Root), strings.Join(fields, ","))
	var registrations []prepared
	for _, registration := range judge.Registrations {
		question := questions[registration.Question]
		if question == nil || names[registration.Question] == "" {
			return "", fmt.Errorf("judge question %s has no supported Noul lowering", registration.Question)
		}
		if question.Connection != judge.Connection {
			return "", fmt.Errorf("judge connection mismatch in checked plan")
		}
		for _, preparation := range registration.Prepare {
			value, err := e.expression.Lower(preparation.Value)
			if err != nil {
				return "", err
			}
			body.WriteString(value.Statements)
			local := e.temp()
			e.expression.Bindings[preparation.Local.Identity] = local
			fmt.Fprintf(&body, "const %s = %s;\n", local, value.Value)
		}
		var arguments []string
		for _, argument := range registration.Arguments {
			value, err := e.expression.Lower(argument)
			if err != nil {
				return "", err
			}
			body.WriteString(value.Statements)
			arguments = append(arguments, value.Value)
		}
		boxed, descriptor := e.temp(), e.temp()
		body.WriteString(e.mark(registration.Span, "question_preparation"))
		fmt.Fprintf(&body, "const %s = await %s(%s);\nif (%s.kind !== \"ok\") return %s;\nconst %s = %s.value;\n", boxed, names[registration.Question], strings.Join(append(append([]string(nil), arguments...), "$canContext"), ", "), boxed, boxed, descriptor, boxed)
		registrations = append(registrations, prepared{question, arguments, descriptor})
	}
	var descriptors []string
	for _, registration := range registrations {
		descriptors = append(descriptors, registration.descriptor)
	}
	schema, err := json.Marshal(judge.State)
	if err != nil {
		return "", err
	}
	answers := e.temp()
	body.WriteString(e.mark(judge.Span, "judge_request"))
	fmt.Fprintf(&body, "const %s = await $canAI.noul(%s,%s,%s,%s,[%s],$canOrigin,$canContext);\nif (%s.kind !== \"ok\") return %s;\n", answers, connection, quote(model), schema, stateValue, strings.Join(descriptors, ","), answers, answers)
	for i, prepared := range registrations {
		var yes, no string
		for _, option := range prepared.question.Options {
			if option.True {
				yes = names[option.Handler.ID]
			} else {
				no = names[option.Handler.ID]
			}
		}
		if yes == "" || no == "" {
			return "", fmt.Errorf("missing Noul handler emission")
		}
		probability := fmt.Sprintf("%s.value[%d]", answers, i)
		target := fmt.Sprintf("(%s >= %s.minimum ? %s : %s)", probability, prepared.descriptor, yes, no)
		handlerArgs := append(append([]string(nil), prepared.arguments...), probability, "$canContext")
		result := e.temp()
		fmt.Fprintf(&body, "const %s = await %s(%s);\nif (%s.kind !== \"ok\") return %s;\n", result, target, strings.Join(handlerArgs, ","), result, result)
		if binding := judge.Registrations[i].Binding; binding != nil {
			e.expression.Bindings[binding.Identity] = result + ".value"
		}
	}
	var continuationArgs []string
	for _, input := range judge.Continuation.Inputs {
		binding := e.expression.Bindings[input.Identity]
		if binding == "" {
			return "", fmt.Errorf("missing judge continuation input")
		}
		continuationArgs = append(continuationArgs, binding)
	}
	fmt.Fprintf(&body, "return await %s(%s);\n", names[judge.Continuation.ID], strings.Join(append(continuationArgs, "$canContext"), ","))
	return fmt.Sprintf("async function %s(%s): Promise<$canCompletion<%s>> {\nlet $canOrigin = %s;\ntry {\n%s} catch ($canCause) { return $canCaught($canCause, $canOrigin); }\n}\n", name, strings.Join(args, ", "), TypeName(region.Result), e.origin(region.Span), body.String()), nil
}
