package emit

import (
	"encoding/json"
	"fmt"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"strings"
)

func (e *RegionEmitter) LLM(name string, plan *ir.LLM, connection, model string, maxTokens int64) (string, error) {
	if plan == nil || !jsBinding.MatchString(name) {
		return "", fmt.Errorf("invalid LLM phase plan")
	}
	region := &ir.Region{ID: plan.Identity, Source: plan.Source, Span: plan.Span, Inputs: plan.Inputs, Result: plan.Result}
	args, err := e.configure(region)
	if err != nil {
		return "", err
	}
	var body strings.Builder
	var fields []string
	for _, node := range plan.State.Nodes {
		if node.Identity == plan.State.Root {
			for i, field := range node.Fields {
				fields = append(fields, "["+quote(field.Name)+","+e.expression.Bindings[plan.StateInputs[i].Identity]+"]")
			}
		}
	}
	state := e.temp()
	fmt.Fprintf(&body, "const %s=$canRecord(%s,[%s]);\n", state, quote(plan.State.Root), strings.Join(fields, ","))
	instructions, err := e.expression.Lower(plan.Instructions)
	if err != nil {
		return "", err
	}
	body.WriteString(instructions.Statements)
	schema, err := json.Marshal(plan.State)
	if err != nil {
		return "", err
	}
	format := "undefined"
	if plan.Format != nil {
		output, err := json.Marshal(plan.Output)
		if err != nil {
			return "", err
		}
		format = fmt.Sprintf("{name:%s,schema:%s,codec:%s}", quote(plan.Format.Name), quote(string(plan.Format.Schema)), output)
	}
	body.WriteString(e.mark(plan.Span, "llm_request"))
	fmt.Fprintf(&body, "return await $canResponses.generate<%s>(%s,%s,%d,%s,%s,%s,%s,$canOrigin,$canContext);\n", TypeName(plan.Result), connection, quote(model), maxTokens, instructions.Value, schema, state, format)
	return fmt.Sprintf("async function %s(%s):Promise<$canCompletion<%s>>{\nlet $canOrigin=%s;\ntry{\n%s}catch($canCause){return $canCaught($canCause,$canOrigin);}\n}\n", name, strings.Join(args, ","), TypeName(plan.Result), e.origin(plan.Span), body.String()), nil
}
