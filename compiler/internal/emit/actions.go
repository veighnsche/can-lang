package emit

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/check"
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
// the HTML-only visible response mode: swap for fragment cases, document
// for full-page cases. JSON cases carry neither.
type emittedActionCase struct {
	Leaf     string `json:"leaf"`
	Status   int    `json:"status"`
	Swap     string `json:"swap,omitempty"`
	Document bool   `json:"document,omitempty"`
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

// emittedSwapCase is one declared non-2xx swap case the HTML renderer
// re-admits through an exact hx-status attribute.
type emittedSwapCase struct {
	Status int    `json:"status"`
	Swap   string `json:"swap"`
}

// emittedSwapPolicy is the renderer-visible projection of one HTML action:
// its method, its path segments with captures normalized to "{}", and its
// declared swap cases. The runtime matches htmx verb URLs structurally
// against this table; 2xx cases are omitted because the global noSwap
// policy already lets them through.
type emittedSwapPolicy struct {
	Method   string            `json:"method"`
	Segments []string          `json:"segments"`
	Cases    []emittedSwapCase `json:"cases"`
}

// swapPathSegments normalizes a checked action path for structural URL
// matching: whole-segment captures become "{}", static segments decode
// once, and an undecodable static stays raw (the checker already
// validated the spelling, so this only affects policy matching).
func swapPathSegments(path string) []string {
	parts := strings.Split(path, "/")[1:]
	segments := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.HasPrefix(part, ":") {
			segments = append(segments, "{}")
			continue
		}
		if decoded, err := url.PathUnescape(part); err == nil {
			segments = append(segments, decoded)
			continue
		}
		segments = append(segments, part)
	}
	return segments
}

// actionCaseMetadata freezes one checked case: fragment swaps keep the
// swap policy, document cases set the document mode, JSON cases carry
// neither. Both the action table and mount sites share this mapping.
func actionCaseMetadata(leaf string, status int, mode string) (emittedActionCase, error) {
	emitted := emittedActionCase{Leaf: leaf, Status: status}
	switch mode {
	case "":
	case "inner":
		emitted.Swap = mode
	case "document":
		emitted.Document = true
	default:
		return emittedActionCase{}, fmt.Errorf("unknown action case mode %s", mode)
	}
	return emitted, nil
}

// swapPolicies derives the HTML swap-exception table from the checked
// actions in declaration order. JSON actions and HTML actions without a
// declared non-2xx swap case contribute nothing; an empty result keeps
// the HTML factory call byte-identical.
func swapPolicies(actions []*check.ActionDeclaration) []emittedSwapPolicy {
	var policies []emittedSwapPolicy
	for _, action := range actions {
		if action.Body != "html" {
			continue
		}
		var cases []emittedSwapCase
		for _, kase := range action.Cases {
			// Only fragment cases reach the htmx guard table; document
			// cases render full pages and 2xx cases swap globally.
			if kase.Swap != "inner" || (kase.Status >= 200 && kase.Status <= 299) {
				continue
			}
			cases = append(cases, emittedSwapCase{Status: kase.Status, Swap: kase.Swap})
		}
		if len(cases) == 0 {
			continue
		}
		policies = append(policies, emittedSwapPolicy{Method: action.Method, Segments: swapPathSegments(action.Path), Cases: cases})
	}
	return policies
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
			emitted, err := actionCaseMetadata(kase.Leaf, kase.Status, kase.Swap)
			if err != nil {
				return err
			}
			entry.Cases = append(entry.Cases, emitted)
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
