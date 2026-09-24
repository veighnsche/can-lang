package emit

import (
	"encoding/json"
	"fmt"
)

// emittedActionCapture is one frozen path capture: a whole-segment name
// with its str or int wire type from the captures record.
type emittedActionCapture struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// emittedActionInput is the frozen request line: the none, json or form
// mode plus, for wire modes, the wire record declaration, the declared
// byte and row limits and the derived codec schema the adapters decode
// with.
type emittedActionInput struct {
	Mode      string      `json:"mode"`
	Type      string      `json:"type,omitempty"`
	Limit     int         `json:"limit,omitempty"`
	RowsLimit int         `json:"rowsLimit,omitempty"`
	Schema    interface{} `json:"schema,omitempty"`
}

// emittedActionCase maps one returns leaf identity to its wire status plus
// the HTML-only visible swap policy.
type emittedActionCase struct {
	Leaf   string `json:"leaf"`
	Status int    `json:"status"`
	Swap   string `json:"swap,omitempty"`
}

// emittedAction is the frozen contract one handler-free action declaration
// contributes. The locked package identity plus declaration name keys the
// metadata; server and browser adapters consume the table, and handlers
// bind later at action::mount. ResponseSchema is the typed
// action::response metadata: the shared JSON wire schema of the returns
// variant. JSON-mode actions always carry it; HTML actions omit it.
type emittedAction struct {
	Identity       string                 `json:"identity"`
	Method         string                 `json:"method"`
	Path           string                 `json:"path"`
	CapturesType   string                 `json:"capturesType,omitempty"`
	Captures       []emittedActionCapture `json:"captures"`
	Input          emittedActionInput     `json:"input"`
	Returns        string                 `json:"returns"`
	Body           string                 `json:"body"`
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
			Input:    emittedActionInput{Mode: action.Input.Mode},
			Returns:  action.Returns.Declaration(),
			Body:     action.Body,
		}
		if action.CapturesType != nil {
			entry.CapturesType = action.CapturesType.Declaration()
		}
		for _, capture := range action.Captures {
			entry.Captures = append(entry.Captures, emittedActionCapture{Name: capture.Name, Type: capture.Type.Declaration()})
		}
		switch action.Input.Mode {
		case "none":
		case "json":
			entry.Input.Type = action.Input.Type.Declaration()
			entry.Input.Limit = action.Input.Limit
			entry.Input.Schema = action.Input.Schema
		case "form":
			entry.Input.Type = action.Input.Type.Declaration()
			entry.Input.Limit = action.Input.Limit
			entry.Input.RowsLimit = action.Input.RowsLimit
			entry.Input.Schema = action.Input.Form
		default:
			return fmt.Errorf("unknown action input mode %s", action.Input.Mode)
		}
		for _, kase := range action.Cases {
			entry.Cases = append(entry.Cases, emittedActionCase{Leaf: kase.Leaf, Status: kase.Status, Swap: kase.Swap})
		}
		if action.Body == "json" {
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
