package emit

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
)

// QuestionPreparation captures exactly the values used to build the request.
// The typed runner is invoked only after the whole response has been validated.
func (e *RegionEmitter) QuestionPreparation(name string, q *ir.Question, names map[string]string) (string, error) {
	if q == nil || !jsBinding.MatchString(name) {
		return "", fmt.Errorf("invalid question phase plan")
	}
	region := &ir.Region{ID: q.Identity, Source: q.Source, Span: q.Span, Inputs: q.Inputs, Result: q.Result}
	args, err := e.configure(region)
	if err != nil {
		return "", err
	}
	var body strings.Builder
	preparedExpressions := map[*ir.Expression]string{}
	lower := func(expr *ir.Expression) (string, error) {
		if value, ok := preparedExpressions[expr]; ok {
			return value, nil
		}
		v, err := e.expression.Lower(expr)
		if err != nil {
			return "", err
		}
		body.WriteString(v.Statements)
		preparedExpressions[expr] = v.Value
		return v.Value, nil
	}
	expressions := []*ir.Expression{q.Instructions}
	if q.Minimum != nil {
		expressions = append(expressions, q.Minimum)
	}
	for _, option := range q.Options {
		if option.Description != nil {
			expressions = append(expressions, option.Description)
		}
		if option.Spread != nil {
			expressions = append(expressions, option.Spread)
		}
	}
	sort.SliceStable(expressions, func(i, j int) bool { return expressions[i].Span.Start < expressions[j].Span.Start })
	for _, expression := range expressions {
		if _, err := lower(expression); err != nil {
			return "", err
		}
	}
	instructions, err := lower(q.Instructions)
	if err != nil {
		return "", err
	}
	type option struct {
		name, description, target string
		handler                   *ir.Region
	}
	var options []option
	var dynamic []string
	for _, o := range q.Options {
		if o.Spread != nil {
			value, err := lower(o.Spread)
			if err != nil {
				return "", err
			}
			if o.Dynamic {
				dynamic = append(dynamic, "..."+value+".map(option=>Object.freeze({key:option.key,description:option.description}))")
				continue
			}
			for _, field := range o.Spread.Type.Fields() {
				captured := e.temp()
				fmt.Fprintf(&body, "const %s=%s[%s];\n", captured, value, quote(field.Name))
				options = append(options, option{name: field.Name, description: captured + ".description", target: captured + ".run"})
			}
		} else {
			description, err := lower(o.Description)
			if err != nil {
				return "", err
			}
			options = append(options, option{name: o.Name, description: description, handler: o.Handler})
		}
	}
	minimum := ""
	if q.Minimum != nil {
		minimum, err = lower(q.Minimum)
		if err != nil {
			return "", err
		}
	}
	descriptor := e.temp()
	fields := []string{"kind:" + quote(q.Kind), "instructions:" + instructions}
	if minimum != "" {
		fields = append(fields, "minimum:"+minimum)
	}
	switch q.Kind {
	case "noul":
		for _, o := range options {
			key := "falseDescription"
			if o.name == "true" {
				key = "trueDescription"
			}
			fields = append(fields, key+":"+o.description)
		}
	case "choice":
		entries := dynamic
		for _, o := range options {
			entries = append(entries, "Object.freeze({key:"+quote(o.name)+",description:"+o.description+"})")
		}
		fields = append(fields, "options:Object.freeze(["+strings.Join(entries, ",")+"])")
	case "score":
		entries := []string{}
		for _, o := range options {
			entries = append(entries, o.description)
		}
		fields = append(fields, "levels:Object.freeze(["+strings.Join(entries, ",")+"])")
	default:
		return "", fmt.Errorf("unsupported checked question kind")
	}
	fmt.Fprintf(&body, "const %s=Object.freeze({%s});\n", descriptor, strings.Join(fields, ","))
	runner := e.temp()
	fmt.Fprintf(&body, "const %s=async ($canAnswer:$canAnswer):Promise<$canCompletion<%s>>=>{\nif($canAnswer.kind!==%s)throw new TypeError(\"checked question answer required\");\n", runner, TypeName(q.Result), quote(q.Kind))
	for _, metadata := range q.Metadata {
		e.expression.Bindings[metadata.Local.Identity] = "$canAnswer." + metadata.Kind
	}
	invoke := func(handler *ir.Region, probability, selected string) (string, error) {
		if handler == nil || names[handler.ID] == "" {
			return "", fmt.Errorf("missing checked question handler")
		}
		values := []string{}
		for _, input := range handler.Inputs {
			value := e.expression.Bindings[input.Identity]
			if strings.HasSuffix(input.Identity, "/probability") {
				value = probability
			}
			if strings.Contains(input.Identity, "/selected/") {
				value = selected
			}
			if value == "" {
				return "", fmt.Errorf("missing question handler input %s", input.Identity)
			}
			values = append(values, value)
		}
		return names[handler.ID] + "(" + strings.Join(append(values, "$canContext"), ",") + ")", nil
	}
	callOption := func(o option, probability string) (string, error) {
		if o.target != "" {
			return o.target + "(" + probability + ",$canContext)", nil
		}
		return invoke(o.handler, probability, "")
	}
	if q.Fallback != nil {
		call, err := invoke(q.Fallback, "", "")
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&body, "if($canAnswer.confidence<%s.minimum)return await %s;\n", descriptor, call)
	}
	if q.Record {
		var entries []string
		for i, o := range options {
			call, err := callOption(o, fmt.Sprintf("$canAnswer.probabilities[%d]", i))
			if err != nil {
				return "", err
			}
			result := e.temp()
			fmt.Fprintf(&body, "const %s=await %s;if(%s.kind!==\"ok\")return %s;\n", result, call, result, result)
			entries = append(entries, "["+quote(o.name)+","+result+".value]")
		}
		for _, metadata := range q.Metadata {
			entries = append(entries, "["+quote(metadata.Name)+",$canAnswer."+metadata.Kind+"]")
		}
		fmt.Fprintf(&body, "return $canSuccess($canRecord(%s,[%s]) as %s);\n", quote(q.Result.Identity()), strings.Join(entries, ","), TypeName(q.Result))
	} else if q.Shared != nil {
		probability, selected := "", ""
		if q.Kind == "choice" {
			selected = "$canAnswer.choice"
			probability = "$canAnswer.probabilities[" + descriptor + ".options.findIndex(option=>option.key===$canAnswer.choice)]"
		}
		call, err := invoke(q.Shared, probability, selected)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&body, "return await %s;\n", call)
	} else {
		for i, o := range options {
			probability := fmt.Sprintf("$canAnswer.probabilities[%d]", i)
			condition := "$canAnswer.choice===" + quote(o.name)
			if q.Kind == "noul" {
				probability = "$canAnswer.probability"
				condition = probability + ">=" + descriptor + ".minimum"
				if o.name == "false" {
					condition = "!(" + condition + ")"
				}
			}
			call, err := callOption(o, probability)
			if err != nil {
				return "", err
			}
			fmt.Fprintf(&body, "if(%s)return await %s;\n", condition, call)
		}
		body.WriteString("throw new TypeError(\"validated question selection missing\");\n")
	}
	fmt.Fprintf(&body, "};\nreturn $canSuccess(Object.freeze({descriptor:%s,run:%s}));\n", descriptor, runner)
	return fmt.Sprintf("async function %s(%s):Promise<$canCompletion<$canPreparedQuestion<%s>>>{\nlet $canOrigin=%s;\ntry{\n%s}catch($canCause){return $canCaught($canCause,$canOrigin);}\n}\n", name, strings.Join(args, ","), TypeName(q.Result), e.origin(q.Span), body.String()), nil
}

// Judge emits registration-order preparation followed by one validated request,
// all selected handlers, and finally the ordinary continuation region.
func (e *RegionEmitter) Judge(name string, judge *ir.Judge, questions map[string]*ir.Question, names map[string]string, connection, model string) (string, error) {
	if judge == nil || judge.Continuation == nil || !jsBinding.MatchString(name) {
		return "", fmt.Errorf("invalid judge phase plan")
	}
	region := &ir.Region{ID: judge.Identity, Source: judge.Source, Span: judge.Span, Inputs: judge.Inputs, Result: judge.Continuation.Result}
	args, err := e.configure(region)
	if err != nil {
		return "", err
	}
	var body strings.Builder
	type prepared struct{ descriptor string }
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
			return "", fmt.Errorf("judge question %s has no checked question lowering", registration.Question)
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
		registrations = append(registrations, prepared{descriptor})
	}
	var descriptors []string
	for _, registration := range registrations {
		descriptors = append(descriptors, registration.descriptor+".descriptor")
	}
	schema, err := json.Marshal(judge.State)
	if err != nil {
		return "", err
	}
	answers := e.temp()
	body.WriteString(e.mark(judge.Span, "judge_request"))
	fmt.Fprintf(&body, "const %s = await $canAI.ask(%s,%s,%s,%s,[%s],$canOrigin,$canContext);\nif (%s.kind !== \"ok\") return %s;\n", answers, connection, quote(model), schema, stateValue, strings.Join(descriptors, ","), answers, answers)
	for i, prepared := range registrations {
		result := e.temp()
		fmt.Fprintf(&body, "const %s=await %s.run(%s.value[%d]);\nif(%s.kind!==\"ok\")return %s;\n", result, prepared.descriptor, answers, i, result, result)
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
