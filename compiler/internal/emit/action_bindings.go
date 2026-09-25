package emit

import (
	"encoding/json"
	"fmt"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
)

// emittedActionMountSite is the frozen server contract one action::mount
// call site splices into its invocation: identity, route, captures,
// input codec, returns, body mode and the exhaustive case table with
// per-case HTML swap policy for the guard. JSON mounts carry the shared
// response schema; HTML mounts carry the form schema plus the
// structural-422 value identities. Handler and renderer callables travel
// as ordinary lowered arguments; the site carries metadata only.
type emittedActionMountSite struct {
	Action         string                 `json:"action"`
	Method         string                 `json:"method"`
	Path           string                 `json:"path"`
	CapturesType   string                 `json:"capturesType,omitempty"`
	Captures       []emittedActionCapture `json:"captures"`
	Input          emittedActionInput     `json:"input"`
	Returns        string                 `json:"returns"`
	Body           string                 `json:"body"`
	Cases          []emittedActionCase    `json:"cases"`
	ResponseSchema interface{}            `json:"responseSchema,omitempty"`
	Rejected       string                 `json:"rejected,omitempty"`
	RawEntry       string                 `json:"rawEntry,omitempty"`
	Issue          string                 `json:"issue,omitempty"`
}

// emittedActionURLSite is the frozen builder contract one action::url
// call site splices into its invocation: identity, route template and
// the ordered captures the canonical path renders.
type emittedActionURLSite struct {
	Action   string                 `json:"action"`
	Path     string                 `json:"path"`
	Captures []emittedActionCapture `json:"captures"`
}

// actionMountMetadata freezes one checked mount site for emission.
func actionMountMetadata(site *ir.ActionSite) (string, error) {
	emitted := emittedActionMountSite{
		Action:   site.Action,
		Method:   site.Method,
		Path:     site.Path,
		Captures: []emittedActionCapture{},
		Input:    emittedActionInput{Mode: site.InputMode},
		Returns:  site.Returns,
		Body:     site.Body,
		Cases:    []emittedActionCase{},
	}
	if site.CapturesType != "" {
		emitted.CapturesType = site.CapturesType
	}
	for _, capture := range site.Captures {
		emitted.Captures = append(emitted.Captures, emittedActionCapture{Name: capture.Name, Type: capture.Type})
	}
	switch site.InputMode {
	case "none":
	case "json":
		emitted.Input.Type = site.InputType
		emitted.Input.Limit = site.Limit
		if site.Request == nil {
			return "", fmt.Errorf("JSON mount site %s has no checked request schema", site.Action)
		}
		emitted.Input.Schema = site.Request
	case "form":
		emitted.Input.Type = site.InputType
		emitted.Input.Limit = site.Limit
		emitted.Input.RowsLimit = site.RowsLimit
		if site.Form == nil {
			return "", fmt.Errorf("HTML mount site %s has no checked form schema", site.Action)
		}
		emitted.Input.Schema = site.Form
	default:
		return "", fmt.Errorf("unknown mount input mode %s", site.InputMode)
	}
	for _, kase := range site.Cases {
		emitted.Cases = append(emitted.Cases, emittedActionCase{Leaf: kase.Leaf, Status: kase.Status, Swap: kase.Swap})
	}
	if site.Body == "json" {
		if site.Response == nil {
			return "", fmt.Errorf("JSON mount site %s has no checked response schema", site.Action)
		}
		emitted.ResponseSchema = site.Response
	} else {
		emitted.Rejected = site.Rejected
		emitted.RawEntry = site.RawEntry
		emitted.Issue = site.Issue
	}
	encoded, err := json.Marshal(emitted)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

// actionURLMetadata freezes one checked URL builder site for emission.
func actionURLMetadata(site *ir.ActionSite) (string, error) {
	emitted := emittedActionURLSite{Action: site.Action, Path: site.Path, Captures: []emittedActionCapture{}}
	for _, capture := range site.Captures {
		emitted.Captures = append(emitted.Captures, emittedActionCapture{Name: capture.Name, Type: capture.Type})
	}
	encoded, err := json.Marshal(emitted)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

// actionClientMetadata freezes one checked request/post site for
// emission. The client projection reuses the JSON fetch site shape:
// identity, route, captures, the POST request schema, the shared
// response schema and the finite case table. No handler, renderer or
// form binding enters the client projection.
func actionClientMetadata(site *ir.ActionSite) (string, error) {
	fetch := &ir.JSONFetchSite{Action: site.Action, Method: site.Method, Path: site.Path, Request: site.Request, Response: *site.Response, Limit: site.Limit}
	for _, capture := range site.Captures {
		fetch.Captures = append(fetch.Captures, ir.JSONFetchCapture{Name: capture.Name, Type: capture.Type})
	}
	for _, kase := range site.Cases {
		fetch.Cases = append(fetch.Cases, ir.JSONFetchCase{Leaf: kase.Leaf, Status: kase.Status})
	}
	return fetchActionMetadata(fetch)
}

// actionMetadata freezes one checked action consumer site for emission,
// selecting the server, builder or client projection by operation.
func actionMetadata(site *ir.ActionSite) (string, error) {
	switch site.Operation {
	case "can.std.action@1::mount":
		return actionMountMetadata(site)
	case "can.std.action@1::url":
		return actionURLMetadata(site)
	case "can.std.action@1::request", "can.std.action@1::post":
		if site.Response == nil {
			return "", fmt.Errorf("client site %s has no checked response schema", site.Action)
		}
		return actionClientMetadata(site)
	default:
		return "", fmt.Errorf("unknown action operation %s", site.Operation)
	}
}

// actionMountTarget selects the server mount entry for one spliced site.
// JSON mounts bind the handler alone; HTML mounts bind the handler plus
// the distinct normal and structural renderers.
func actionMountTarget(site *ir.ActionSite) string {
	if site.Body == "html" {
		return "$canActionRoutes.mountForm"
	}
	return "$canActionRoutes.mount"
}

// actionBindings maps the four symbol-based action consumers to their
// adapter targets. Mount and URL building share the server route state
// object; the GET and POST clients share the fetch state object.
func (assembly *programAssembly) actionBindings() bindingContribution {
	return bindingContribution{domain: "action-bindings", functions: map[string]string{
		"can.std.action@1::mount":   "$canActionRoutes.mount",
		"can.std.action@1::url":     "$canActionRoutes.url",
		"can.std.action@1::request": "$canActionClient.request",
		"can.std.action@1::post":    "$canActionClient.post",
	}}
}

// actionStateImports lists the action adapter factories the shared state
// module needs. Only the selected state objects are imported: programs
// without action consumers keep their existing import closure.
func (assembly *programAssembly) actionStateImports(runtime string) []ModuleImport {
	imports := []ModuleImport{}
	if assembly.program.ActionRoutes {
		imports = append(imports, ModuleImport{Target: runtime + "/platform/action-routes.ts", Names: []ImportName{{"createActionRoutes", "$canCreateActionRoutes"}}})
	}
	if assembly.program.ActionClient {
		imports = append(imports, ModuleImport{Target: runtime + "/platform/action-client.ts", Names: []ImportName{{"createActionClient", "$canCreateActionClient"}}})
	}
	return imports
}

// declareActionState emits the selected action factory bindings.
func (builder *stateBuilder) declareActionState() {
	if builder.assembly.program.ActionRoutes {
		builder.out.WriteString("export let $canActionRoutes: ReturnType<typeof $canCreateActionRoutes>;\n")
	}
	if builder.assembly.program.ActionClient {
		builder.out.WriteString("export let $canActionClient: ReturnType<typeof $canCreateActionClient>;\n")
	}
}

// initializeActionState constructs the selected action adapters inside
// the shared initializer, after the domain runtime exists. The route
// binder maps mount and builder failures into invalid_route and
// invalid_path; the client maps native fetch outcomes into the fixed
// failure bound, exactly like the JSON fetch adapter.
func (builder *stateBuilder) initializeActionState() {
	if builder.assembly.program.ActionRoutes {
		fmt.Fprintf(&builder.out, "$canActionRoutes=$canCreateActionRoutes($canDomain,{invalidRoute:%s,invalidPath:%s});\n", quote(builder.numberIDs["can.std.http@1::invalid_route"]), quote(builder.numberIDs["can.std.action@1::invalid_path"]))
	}
	if builder.assembly.program.ActionClient {
		fmt.Fprintf(&builder.out, "$canActionClient=$canCreateActionClient($canDomain,{transport:%s,invalidRequest:%s,bodyLimit:%s,statusError:%s,invalidData:%s,header:%s});\n", quote(builder.numberIDs["can.std.http@1::transport_failed"]), quote(builder.numberIDs["can.std.http@1::invalid_request"]), quote(builder.numberIDs["can.std.http@1::body_limit"]), quote(builder.numberIDs["can.std.http@1::status_error"]), quote(builder.numberIDs["can.std.codec@1::invalid_data"]), quote(builder.numberIDs["can.std.http@1::header"]))
	}
}
