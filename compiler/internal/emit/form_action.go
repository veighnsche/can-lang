package emit

import (
	"encoding/json"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// emittedFormActionCase is one frozen result leaf runtime identity plus
// its wire status.
type emittedFormActionCase struct {
	Leaf   string `json:"leaf"`
	Status int    `json:"status"`
}

// emittedFormActionSite is the frozen adapter contract one
// serve_form_action call site splices into its invocation: identity,
// route, form schema, result cases and the rejected-value identities.
// The handler travels as a separate emitted binding.
type emittedFormActionSite struct {
	Action   string                  `json:"action"`
	Method   string                  `json:"method"`
	Path     string                  `json:"path"`
	Form     types.FormSchema        `json:"form"`
	Cases    []emittedFormActionCase `json:"cases"`
	Rejected string                  `json:"rejected"`
	RawEntry string                  `json:"rawEntry"`
	Issue    string                  `json:"issue"`
}

// formActionMetadata freezes one checked adapter site for emission.
func formActionMetadata(site *ir.FormActionSite) (string, error) {
	emitted := emittedFormActionSite{
		Action:   site.Action,
		Method:   site.Method,
		Path:     site.Path,
		Form:     site.Form,
		Cases:    []emittedFormActionCase{},
		Rejected: site.Rejected,
		RawEntry: site.RawEntry,
		Issue:    site.Issue,
	}
	for _, kase := range site.Cases {
		emitted.Cases = append(emitted.Cases, emittedFormActionCase{Leaf: kase.Leaf, Status: kase.Status})
	}
	encoded, err := json.Marshal(emitted)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}
