package emit

import (
	"encoding/json"
	"fmt"
)

// emittedActionCapture is one frozen path capture: a whole-segment name
// with its str or int wire type.
type emittedActionCapture struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// emittedActionBody is the frozen POST wire contract: the json or form
// mode, the wire record declaration and the derived codec schema the
// adapters decode with.
type emittedActionBody struct {
	Mode   string      `json:"mode"`
	Type   string      `json:"type"`
	Schema interface{} `json:"schema"`
}

// emittedActionCase maps one result leaf identity to its wire status.
type emittedActionCase struct {
	Leaf   string `json:"leaf"`
	Status int    `json:"status"`
}

// emittedAction is the frozen contract one action declaration contributes.
// Server and browser adapters consume the table; it carries no behavior.
// ResponseSchema is the typed action::response metadata: the shared JSON
// wire schema of the result variant. JSON-mode actions (POST with a json
// body, and bodyless GET) always carry it; form actions omit it.
type emittedAction struct {
	Identity       string                 `json:"identity"`
	Method         string                 `json:"method"`
	Path           string                 `json:"path"`
	Captures       []emittedActionCapture `json:"captures"`
	Body           *emittedActionBody     `json:"body,omitempty"`
	Handler        string                 `json:"handler"`
	Result         string                 `json:"result"`
	Cases          []emittedActionCase    `json:"cases"`
	ResponseSchema interface{}            `json:"responseSchema,omitempty"`
}

// emitActionConstants freezes the checked action table after the shared
// state initializer. Programs without actions emit no table.
func (builder *stateBuilder) emitActionConstants() error {
	actions := builder.assembly.program.Actions
	if len(actions) == 0 {
		return nil
	}
	emitted := make([]emittedAction, 0, len(actions))
	for _, action := range actions {
		entry := emittedAction{
			Identity: action.Symbol.ID,
			Method:   action.Method,
			Path:     action.Path,
			Captures: []emittedActionCapture{},
			Handler:  action.Handler,
			Result:   action.Result.Declaration(),
		}
		for _, capture := range action.Captures {
			entry.Captures = append(entry.Captures, emittedActionCapture{Name: capture.Name, Type: capture.Type.Declaration()})
		}
		if action.Body != nil {
			body := &emittedActionBody{Mode: action.Body.Mode, Type: action.Body.Type.Declaration()}
			switch action.Body.Mode {
			case "json":
				body.Schema = action.Body.Schema
			case "form":
				body.Schema = action.Body.Form
			default:
				return fmt.Errorf("unknown action body mode %s", action.Body.Mode)
			}
			entry.Body = body
		}
		for _, kase := range action.Cases {
			entry.Cases = append(entry.Cases, emittedActionCase{Leaf: kase.Leaf, Status: kase.Status})
		}
		if action.Body == nil || action.Body.Mode == "json" {
			if action.ResponseSchema == nil {
				return fmt.Errorf("JSON action %s has no checked response schema", action.Symbol.ID)
			}
			entry.ResponseSchema = action.ResponseSchema
		}
		emitted = append(emitted, entry)
	}
	encoded, err := json.Marshal(emitted)
	if err != nil {
		return err
	}
	fmt.Fprintf(&builder.out, "export const $canActions = Object.freeze(%s);\n", encoded)
	return nil
}
