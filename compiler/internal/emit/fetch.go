package emit

import (
	"encoding/json"
	"fmt"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"strings"
)

func (e *RegionEmitter) Fetch(name string, plan *ir.Fetch, connection string) (string, error) {
	if plan == nil || !jsBinding.MatchString(name) {
		return "", fmt.Errorf("invalid fetch plan")
	}
	region := &ir.Region{ID: plan.Identity, Source: plan.Source, Span: plan.Span, Inputs: plan.Inputs, Result: plan.Result}
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
		saved := e.temp()
		fmt.Fprintf(&body, "const %s = %s;\n", saved, value.Value)
		return saved, nil
	}
	path, err := lower(plan.Path)
	if err != nil {
		return "", err
	}
	entries := func(list []ir.FetchEntry) (string, error) {
		var values []string
		for _, entry := range list {
			value, err := lower(entry.Value)
			if err != nil {
				return "", err
			}
			values = append(values, "{name:"+quote(entry.Name)+",value:"+value+"}")
		}
		return "[" + strings.Join(values, ",") + "]", nil
	}
	query, err := entries(plan.Query)
	if err != nil {
		return "", err
	}
	headers, err := entries(plan.Headers)
	if err != nil {
		return "", err
	}
	requestBody := "undefined"
	if plan.Body != nil {
		value, err := lower(plan.Body)
		if err != nil {
			return "", err
		}
		schema, err := json.Marshal(plan.BodySchema)
		if err != nil {
			return "", err
		}
		requestBody = "{mode:" + quote(plan.BodyMode) + ",value:" + value + ",schema:" + string(schema) + "}"
		if plan.BodySchema == nil {
			requestBody = "{mode:" + quote(plan.BodyMode) + ",value:" + value + "}"
		}
	}
	result := map[string]any{"mode": plan.ResultMode}
	if plan.ResultSchema != nil {
		result["schema"] = plan.ResultSchema
	}
	if plan.Envelope != "" {
		result["envelope"] = plan.Envelope
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	body.WriteString(e.mark(plan.Span, "fetch_request"))
	fmt.Fprintf(&body, "return await $canFetch.request<%s>(%s,{path:%s,method:%s,query:%s,headers:%s},%s,%s,$canOrigin,$canContext);\n", TypeName(plan.Result), connection, path, quote(plan.Method), query, headers, requestBody, encoded)
	return fmt.Sprintf("async function %s(%s): Promise<$canCompletion<%s>> {\nlet $canOrigin = %s;\ntry {\n%s} catch ($canCause) { return $canCaught($canCause, $canOrigin); }\n}\n", name, strings.Join(args, ", "), TypeName(plan.Result), e.origin(plan.Span), body.String()), nil
}
