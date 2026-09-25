package ir

import "github.com/veighnsche/can-lang/compiler/internal/types"

// ActionCapture is one frozen path capture of a symbol-based action call
// site: the whole-segment name with its str or int wire type.
type ActionCapture struct {
	Name string
	Type string
}

// ActionCase is one frozen result leaf runtime identity plus its wire
// status and the HTML-only visible swap policy ("inner", or "" for JSON
// cases). The mount site carries the full per-case table so the server
// adapter and the HTML guard share one checked policy.
type ActionCase struct {
	Leaf   string
	Status int
	Swap   string
}

// ActionSite is one checked action::mount, action::url, action::request or
// action::post call site: the resolved action plus the frozen contract
// the emitter splices into the invocation. The action symbol is static
// evidence only; it travels here, never as a lowered value argument.
// Request is the POST wire schema for post sites and nil otherwise;
// Response is the shared JSON wire schema of the result variant for
// JSON-mode mount/request/post sites and nil for HTML mounts and urls.
// Form is the form wire schema for HTML mounts. Rejected, RawEntry and
// Issue are the runtime identities the form adapter uses to build
// structural-422 values. Authored arguments stay the value operands
// (captures record, POST body, handler and renderer callables); the site
// joins the invocation only so fixture matching keeps comparing the
// authored values.
type ActionSite struct {
	Operation    string
	Action       string
	Method       string
	Path         string
	CapturesType string
	Captures     []ActionCapture
	InputMode    string
	InputType    string
	Limit        int
	RowsLimit    int
	Request      *types.CodecSchema
	Response     *types.CodecSchema
	Form         *types.FormSchema
	Returns      string
	Body         string
	Cases        []ActionCase
	Rejected     string
	RawEntry     string
	Issue        string
}
