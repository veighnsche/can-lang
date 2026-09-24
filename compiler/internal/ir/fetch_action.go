package ir

import "github.com/veighnsche/can-lang/compiler/internal/types"

// JSONFetchCapture is one frozen path capture of a browser JSON fetch
// site: the whole-segment name with its str or int wire type.
type JSONFetchCapture struct {
	Name string
	Type string
}

// JSONFetchCase is one frozen result leaf identity plus its wire status.
type JSONFetchCase struct {
	Leaf   string
	Status int
}

// JSONFetchSite is one checked fetch_json_get/fetch_json_post call site:
// the resolved JSON action plus the frozen client contract the emitter
// splices into the invocation. Request is nil for bodyless GET; Response
// is the shared JSON wire schema of the result variant. Authored
// arguments stay (action name, captures in path order, POST body); the
// site joins the invocation only so fixture matching keeps comparing the
// authored values.
type JSONFetchSite struct {
	Action   string
	Method   string
	Path     string
	Captures []JSONFetchCapture
	Request  *types.CodecSchema
	Response types.CodecSchema
	Cases    []JSONFetchCase
}
