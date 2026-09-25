package emit

import (
	"encoding/json"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
)

// emittedFetchSite is the frozen client contract one fetch_json_get or
// fetch_json_post call site splices into its invocation: the resolved
// operation identity, the route template with its captures, the request
// schema and declared byte budget for POST, the shared response schema,
// and the finite case table. Both emit profiles splice the identical
// contract from one checked program, so a contract edit regenerates both
// sides together.
type emittedFetchSite struct {
	Action   string                 `json:"action"`
	Method   string                 `json:"method"`
	Path     string                 `json:"path"`
	Captures []emittedActionCapture `json:"captures"`
	Request  interface{}            `json:"request,omitempty"`
	Response interface{}            `json:"response"`
	Cases    []emittedActionCase    `json:"cases"`
	Limit    int                    `json:"limit,omitempty"`
}

// fetchActionMetadata freezes one checked fetch site for emission.
func fetchActionMetadata(site *ir.JSONFetchSite) (string, error) {
	emitted := emittedFetchSite{
		Action:   site.Action,
		Method:   site.Method,
		Path:     site.Path,
		Captures: []emittedActionCapture{},
		Response: site.Response,
		Cases:    []emittedActionCase{},
		Limit:    site.Limit,
	}
	for _, capture := range site.Captures {
		emitted.Captures = append(emitted.Captures, emittedActionCapture{Name: capture.Name, Type: capture.Type})
	}
	if site.Request != nil {
		emitted.Request = site.Request
	}
	for _, kase := range site.Cases {
		emitted.Cases = append(emitted.Cases, emittedActionCase{Leaf: kase.Leaf, Status: kase.Status})
	}
	encoded, err := json.Marshal(emitted)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}
