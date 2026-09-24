package ir

import "github.com/veighnsche/can-lang/compiler/internal/types"

// FormActionCase is one frozen result leaf identity plus its wire status.
type FormActionCase struct {
	Leaf   string
	Status int
}

// FormActionSite is one checked serve_form_action call site: the served
// action plus the frozen adapter contract the emitter splices into the
// invocation. Rejected, RawEntry and Issue are the runtime identities the
// adapter uses to build structural-422 values.
type FormActionSite struct {
	Action   string
	Method   string
	Path     string
	Form     types.FormSchema
	Cases    []FormActionCase
	Handler  string
	Rejected string
	RawEntry string
	Issue    string
}
