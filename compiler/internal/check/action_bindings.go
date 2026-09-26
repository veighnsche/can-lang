package check

import (
	"fmt"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/catalogue"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

const (
	actionMount   = "can.std.action@1::mount"
	actionURL     = "can.std.action@1::url"
	actionRequest = "can.std.action@1::request"
	actionPost    = "can.std.action@1::post"
)

// actionOperation reports whether the identity is one of the four
// symbol-based action consumers. Their call contracts derive per resolved
// action, so the catalogue descriptors carry the shared result and failure
// vocabulary only.
func actionOperation(identity string) bool {
	switch identity {
	case actionMount, actionURL, actionRequest, actionPost:
		return true
	}
	return false
}

// actionSiteOperation recovers the action consumer operation from a call
// identity, or "" when the call is not an action consumer.
func actionSiteOperation(identity string) string {
	if actionOperation(identity) {
		return identity
	}
	return ""
}

// actionOperandError carries a site-resolution failure plus the full
// argument index it belongs to: 0 names the action symbol, rest positions
// shift by one, and -1 names the call itself. The region layer maps the
// index to its use-site span.
type actionOperandError struct {
	Index int
	Err   error
}

func (e *actionOperandError) Error() string { return e.Err.Error() }
func (e *actionOperandError) Unwrap() error { return e.Err }

// admitActionOperation binds the placeholder contract for one action
// consumer. Every call site resolves its action symbol first and replaces
// this contract with the per-action derived one before any argument is
// checked, so the placeholder is never used for checking; it only lets
// the callee name resolve. Like the generic admission path it gathers a
// signature node, since derived constructors require the sealed graph.
func (c *programChecker) admitActionOperation(p *Program, builtin *resolve.File, op catalogue.Operation) error {
	str, err := parseCatalogueType("str")
	if err != nil {
		return err
	}
	typ, err := c.gather(builtin, &syntax.CallableType{Result: str})
	if err != nil {
		return err
	}
	c.bindings[op.Identity] = typ
	p.Intrinsics[op.Identity] = typ
	return nil
}

// actionSpelling renders one action operand for diagnostics.
func actionSpelling(name syntax.QualifiedName) string {
	if name.Package != "" {
		return name.Package + "::" + name.Name
	}
	return name.Name
}

// actionSite resolves one static action symbol against the checked action
// table and derives the call-site contract plus the frozen site the
// emitter splices. The returned contract covers the value operands only;
// the symbol is static evidence and is dropped from the lowered
// arguments. Use records the server route or client fetch state need on
// the checked program for the emitter.
func (c *programChecker) actionSite(file *resolve.File, scope *resolve.Scope, operation string, name syntax.QualifiedName, rest []syntax.Argument) (ir.ActionSite, *types.Type, error) {
	spelling := actionSpelling(name)
	symbol, err := file.Lookup(scope, name, resolve.ActionUse)
	if err != nil {
		return ir.ActionSite{}, nil, &actionOperandError{Index: 0, Err: fmt.Errorf("action %s cannot resolve action %q: %w", strings.TrimPrefix(operation, "can.std.action@1::"), spelling, err)}
	}
	var action *ActionDeclaration
	for _, candidate := range c.program.Actions {
		if candidate.Symbol.ID == symbol.ID {
			action = candidate
			break
		}
	}
	if action == nil {
		return ir.ActionSite{}, nil, &actionOperandError{Index: 0, Err: fmt.Errorf("unknown action %q", spelling)}
	}
	switch operation {
	case actionMount:
		return c.actionMountSite(file, scope, action, spelling, rest)
	case actionURL:
		return c.actionURLSite(action, spelling, rest)
	case actionRequest:
		return c.actionRequestSite(action, spelling, rest)
	case actionPost:
		return c.actionPostSite(action, spelling, rest)
	default:
		return ir.ActionSite{}, nil, fmt.Errorf("unknown action operation %s", operation)
	}
}

// actionSiteBase freezes the declaration projection every consumer site
// shares: identity, route, captures, input, returns, body mode and the
// exhaustive case table with runtime leaf identities. Operation
// derivations add their request/response codecs on top.
func (c *programChecker) actionSiteBase(operation string, action *ActionDeclaration) (ir.ActionSite, error) {
	site := ir.ActionSite{Operation: operation, Action: action.Symbol.ID, Method: action.Method, Path: action.Path, InputMode: action.Input.Mode, Returns: action.Returns.Declaration(), Body: action.Body, Limit: action.Input.Limit, RowsLimit: action.Input.RowsLimit}
	if action.CapturesType != nil {
		site.CapturesType = action.CapturesType.Declaration()
	}
	for _, capture := range action.Captures {
		site.Captures = append(site.Captures, ir.ActionCapture{Name: capture.Name, Type: capture.Type.Declaration()})
	}
	if action.Input.Type != nil {
		site.InputType = action.Input.Type.Declaration()
	}
	for _, kase := range action.Cases {
		leaf, err := c.actionCaseIdentity(action, kase.Leaf)
		if err != nil {
			return ir.ActionSite{}, err
		}
		site.Cases = append(site.Cases, ir.ActionCase{Leaf: leaf, Status: kase.Status, Swap: kase.Swap})
	}
	return site, nil
}

// actionShape describes one action for operand diagnostics.
func actionShape(action *ActionDeclaration) string {
	mode := "JSON GET"
	switch {
	case action.Method == "POST" && action.Input.Mode == "json":
		mode = "JSON POST"
	case action.Method == "POST" && action.Input.Mode == "form":
		mode = "HTML POST"
	case action.Method == "GET" && action.Body == "html":
		mode = "HTML GET"
	}
	return mode
}

// actionMountSite derives the server binding contract: one request-first
// handler for JSON actions, plus the normal outcome renderer and the
// structural-rejection renderer for HTML form actions, or the single
// document renderer for HTML GET reads. All bound callables are
// non-generic, non-variadic and emits []; the application maps fallible
// work into declared leaves. Named callable operands are validated
// against their exact required shape here so arity, request position,
// captures, wire, result and error mismatches diagnose at the operand;
// other callable-valued operands fall through to ordinary checking.
func (c *programChecker) actionMountSite(file *resolve.File, scope *resolve.Scope, action *ActionDeclaration, spelling string, rest []syntax.Argument) (ir.ActionSite, *types.Type, error) {
	site, err := c.actionSiteBase(actionMount, action)
	if err != nil {
		return ir.ActionSite{}, nil, err
	}
	html := action.Body == "html"
	document := html && action.Method == "GET"
	want := 1
	switch {
	case document:
		want = 2
	case html:
		want = 3
	}
	if len(rest) != want {
		if document {
			return ir.ActionSite{}, nil, &actionOperandError{Index: -1, Err: fmt.Errorf("action::mount for HTML GET action %q takes its handler plus the document renderer", spelling)}
		}
		if html {
			return ir.ActionSite{}, nil, &actionOperandError{Index: -1, Err: fmt.Errorf("action::mount for HTML action %q takes its handler plus the normal and structural renderers", spelling)}
		}
		return ir.ActionSite{}, nil, &actionOperandError{Index: -1, Err: fmt.Errorf("action::mount for JSON action %q takes its handler only", spelling)}
	}
	request, err := c.catalogueType("http::request", map[string]*types.Type{})
	if err != nil {
		return ir.ActionSite{}, nil, err
	}
	var inputs []*types.Type
	inputs = append(inputs, request)
	if action.CapturesType != nil {
		inputs = append(inputs, action.CapturesType)
	}
	if action.Input.Type != nil {
		inputs = append(inputs, action.Input.Type)
	}
	handler, err := types.CallableOfChecked(action.Returns, inputs, nil)
	if err != nil {
		return ir.ActionSite{}, nil, err
	}
	if err := c.checkMountCallable(file, scope, action, spelling, "handler", rest[0], handler, 1); err != nil {
		return ir.ActionSite{}, nil, err
	}
	contracts := []*types.Type{handler}
	if document {
		safe, err := c.catalogueType("html::safe", map[string]*types.Type{})
		if err != nil {
			return ir.ActionSite{}, nil, err
		}
		render, err := types.CallableOfChecked(safe, []*types.Type{action.Returns}, nil)
		if err != nil {
			return ir.ActionSite{}, nil, err
		}
		if err := c.checkMountCallable(file, scope, action, spelling, "document renderer", rest[1], render, 2); err != nil {
			return ir.ActionSite{}, nil, err
		}
		contracts = append(contracts, render)
	} else if html {
		safe, err := c.catalogueType("html::safe", map[string]*types.Type{})
		if err != nil {
			return ir.ActionSite{}, nil, err
		}
		outcome, err := types.CallableOfChecked(safe, []*types.Type{action.Returns}, nil)
		if err != nil {
			return ir.ActionSite{}, nil, err
		}
		if err := c.checkMountCallable(file, scope, action, spelling, "normal renderer", rest[1], outcome, 2); err != nil {
			return ir.ActionSite{}, nil, err
		}
		rejected, err := c.catalogueType("form::rejected<T>", map[string]*types.Type{"T": action.Input.Type})
		if err != nil {
			return ir.ActionSite{}, nil, err
		}
		structural, err := types.CallableOfChecked(safe, []*types.Type{rejected}, nil)
		if err != nil {
			return ir.ActionSite{}, nil, err
		}
		if err := c.checkMountCallable(file, scope, action, spelling, "structural renderer", rest[2], structural, 3); err != nil {
			return ir.ActionSite{}, nil, err
		}
		contracts = append(contracts, outcome, structural)
		rawEntry, err := c.catalogueType("form::raw_entry", map[string]*types.Type{})
		if err != nil {
			return ir.ActionSite{}, nil, err
		}
		issue, err := c.catalogueType("form::issue", map[string]*types.Type{})
		if err != nil {
			return ir.ActionSite{}, nil, err
		}
		form := action.Input.Form
		site.Form = &form
		site.Rejected = rejected.Identity()
		site.RawEntry = rawEntry.Identity()
		site.Issue = issue.Identity()
	} else if action.ResponseSchema != nil {
		response := *action.ResponseSchema
		site.Response = &response
		if action.Input.Mode == "json" {
			request := action.Input.Schema
			site.Request = &request
		}
	}
	route, err := c.catalogueType("http::route", map[string]*types.Type{})
	if err != nil {
		return ir.ActionSite{}, nil, err
	}
	invalid, err := c.catalogueType("http::invalid_route", map[string]*types.Type{})
	if err != nil {
		return ir.ActionSite{}, nil, err
	}
	contract, err := types.CallableOfChecked(route, contracts, []*types.Type{invalid})
	if err != nil {
		return ir.ActionSite{}, nil, err
	}
	c.program.ActionRoutes = true
	return site, contract, nil
}

// checkMountCallable validates one named mount operand against its exact
// required callable shape. Only `callable name` references to named
// declarations are pre-validated; any other callable-valued operand is
// left to ordinary assignability checking. Near inputs are absent from
// the effective shape: they are captured lexically when the callable is
// constructed and are checked at the reference by the ordinary path.
func (c *programChecker) checkMountCallable(file *resolve.File, scope *resolve.Scope, action *ActionDeclaration, spelling, role string, arg syntax.Argument, want *types.Type, index int) error {
	reference, ok := fetchUngroup(arg.Value).(*syntax.ReferenceExpr)
	if !ok {
		return nil
	}
	name, ok := fetchUngroup(reference.Callee).(*syntax.NameExpr)
	if !ok {
		return nil
	}
	fail := func(err error) error {
		return &actionOperandError{Index: index, Err: err}
	}
	target, err := file.Lookup(scope, name.Name, resolve.ReferenceUse)
	if err != nil {
		return fail(fmt.Errorf("action::mount %s for action %q: %w", role, spelling, err))
	}
	if len(reference.Types) != 0 || len(target.Parameters) != 0 {
		return fail(fmt.Errorf("action::mount %s %q must be non-generic", role, actionSpelling(name.Name)))
	}
	if _, ok := target.Declaration.(*syntax.FunctionDecl); !ok || target.Kind != resolve.Function {
		return fail(fmt.Errorf("action::mount %s %q must reference a named function", role, actionSpelling(name.Name)))
	}
	if c.variadic[target.ID] {
		return fail(fmt.Errorf("action::mount %s %q must be non-variadic", role, actionSpelling(name.Name)))
	}
	contract := c.bindings[target.ID]
	if contract == nil || contract.Kind() != types.Callable {
		return fail(fmt.Errorf("action::mount %s %q has no checked contract", role, actionSpelling(name.Name)))
	}
	descriptor, ok := c.callables[target.ID]
	if !ok {
		return fail(fmt.Errorf("action::mount %s %q has no callable declaration evidence", role, actionSpelling(name.Name)))
	}
	var effective []*types.Type
	for i, input := range contract.Inputs() {
		if i < len(descriptor.Near) && descriptor.Near[i] {
			continue
		}
		effective = append(effective, input)
	}
	if len(contract.Errors()) != 0 {
		names := make([]string, 0, len(contract.Errors()))
		for _, failure := range contract.Errors() {
			names = append(names, types.CanonicalName(failure))
		}
		return fail(fmt.Errorf("action::mount %s %q emits [%s], but action %q requires emits []", role, actionSpelling(name.Name), strings.Join(names, ", "), spelling))
	}
	if contract.Result().Identity() != want.Result().Identity() {
		return fail(fmt.Errorf("action::mount %s %q returns %s, but action %q requires %s", role, actionSpelling(name.Name), types.CanonicalName(contract.Result()), spelling, types.CanonicalName(want.Result())))
	}
	if len(effective) != len(want.Inputs()) {
		return fail(fmt.Errorf("action::mount %s %q takes %d inputs, but action %q requires %s", role, actionSpelling(name.Name), len(effective), spelling, callableShape(want)))
	}
	for i, input := range want.Inputs() {
		if effective[i].Identity() != input.Identity() {
			if i == 0 {
				return fail(fmt.Errorf("action::mount %s %q must take %s first", role, actionSpelling(name.Name), types.CanonicalName(input)))
			}
			return fail(fmt.Errorf("action::mount %s %q input %d must be %s, not %s", role, actionSpelling(name.Name), i+1, types.CanonicalName(input), types.CanonicalName(effective[i])))
		}
	}
	return nil
}

// callableShape renders one required mount operand shape for diagnostics.
func callableShape(want *types.Type) string {
	inputs := make([]string, 0, len(want.Inputs()))
	for _, input := range want.Inputs() {
		inputs = append(inputs, types.CanonicalName(input))
	}
	return "(" + strings.Join(inputs, ", ") + ") -> " + types.CanonicalName(want.Result())
}

// actionURLSite derives the canonical path builder contract: the action
// symbol plus its captures record, omitted only when the action declares
// no captures. The builder checks strict single-segment admission and
// fails as action::invalid_path.
func (c *programChecker) actionURLSite(action *ActionDeclaration, spelling string, rest []syntax.Argument) (ir.ActionSite, *types.Type, error) {
	site, err := c.actionSiteBase(actionURL, action)
	if err != nil {
		return ir.ActionSite{}, nil, err
	}
	var inputs []*types.Type
	if action.CapturesType != nil {
		if len(rest) != 1 {
			return ir.ActionSite{}, nil, &actionOperandError{Index: -1, Err: fmt.Errorf("action::url for action %q requires its captures record", spelling)}
		}
		inputs = append(inputs, action.CapturesType)
	} else if len(rest) != 0 {
		return ir.ActionSite{}, nil, &actionOperandError{Index: -1, Err: fmt.Errorf("action %q declares no captures", spelling)}
	}
	str, err := c.catalogueType("str", map[string]*types.Type{})
	if err != nil {
		return ir.ActionSite{}, nil, err
	}
	invalid, err := c.catalogueType("action::invalid_path", map[string]*types.Type{})
	if err != nil {
		return ir.ActionSite{}, nil, err
	}
	contract, err := types.CallableOfChecked(str, inputs, []*types.Type{invalid})
	if err != nil {
		return ir.ActionSite{}, nil, err
	}
	c.program.ActionRoutes = true
	return site, contract, nil
}

// actionRequestSite derives the bodyless JSON GET client contract: the
// action symbol plus its captures record. POST, form and HTML actions
// are rejected; a GET construction takes no body.
func (c *programChecker) actionRequestSite(action *ActionDeclaration, spelling string, rest []syntax.Argument) (ir.ActionSite, *types.Type, error) {
	if action.Method != "GET" || action.Input.Mode != "none" || action.Body != "json" {
		return ir.ActionSite{}, nil, &actionOperandError{Index: 0, Err: fmt.Errorf("action::request requires a bodyless JSON GET action, but %q is %s", spelling, actionShape(action))}
	}
	site, err := c.actionSiteBase(actionRequest, action)
	if err != nil {
		return ir.ActionSite{}, nil, err
	}
	var inputs []*types.Type
	if action.CapturesType != nil {
		if len(rest) != 1 {
			return ir.ActionSite{}, nil, &actionOperandError{Index: -1, Err: fmt.Errorf("action::request for GET action %q takes its captures record and no body", spelling)}
		}
		inputs = append(inputs, action.CapturesType)
	} else if len(rest) != 0 {
		return ir.ActionSite{}, nil, &actionOperandError{Index: -1, Err: fmt.Errorf("action::request for GET action %q takes no body", spelling)}
	}
	if action.ResponseSchema == nil {
		return ir.ActionSite{}, nil, &actionOperandError{Index: 0, Err: fmt.Errorf("action %q carries no JSON response schema", spelling)}
	}
	response := *action.ResponseSchema
	site.Response = &response
	bound, err := c.fetchFailureBound(fetchJSONGet)
	if err != nil {
		return ir.ActionSite{}, nil, err
	}
	contract, err := types.CallableOfChecked(action.Returns, inputs, bound)
	if err != nil {
		return ir.ActionSite{}, nil, err
	}
	c.program.ActionClient = true
	return site, contract, nil
}

// actionPostSite derives the JSON POST client contract: the action
// symbol, its captures record and the exact declared wire body. GET,
// form and HTML actions are rejected, so a JSON body can never address
// a form action and an HTML response is never decoded as JSON.
func (c *programChecker) actionPostSite(action *ActionDeclaration, spelling string, rest []syntax.Argument) (ir.ActionSite, *types.Type, error) {
	if action.Method != "POST" || action.Input.Mode != "json" {
		return ir.ActionSite{}, nil, &actionOperandError{Index: 0, Err: fmt.Errorf("action::post requires a JSON POST action, but %q is %s", spelling, actionShape(action))}
	}
	site, err := c.actionSiteBase(actionPost, action)
	if err != nil {
		return ir.ActionSite{}, nil, err
	}
	var inputs []*types.Type
	if action.CapturesType != nil {
		if len(rest) != 2 {
			return ir.ActionSite{}, nil, &actionOperandError{Index: -1, Err: fmt.Errorf("action::post for action %q requires its captures record and wire body", spelling)}
		}
		inputs = append(inputs, action.CapturesType, action.Input.Type)
	} else {
		if len(rest) != 1 {
			return ir.ActionSite{}, nil, &actionOperandError{Index: -1, Err: fmt.Errorf("action::post for action %q requires its wire body", spelling)}
		}
		inputs = append(inputs, action.Input.Type)
	}
	if action.ResponseSchema == nil {
		return ir.ActionSite{}, nil, &actionOperandError{Index: 0, Err: fmt.Errorf("action %q carries no JSON response schema", spelling)}
	}
	request := action.Input.Schema
	response := *action.ResponseSchema
	site.Request = &request
	site.Response = &response
	bound, err := c.fetchFailureBound(fetchJSONPost)
	if err != nil {
		return ir.ActionSite{}, nil, err
	}
	contract, err := types.CallableOfChecked(action.Returns, inputs, bound)
	if err != nil {
		return ir.ActionSite{}, nil, err
	}
	c.program.ActionClient = true
	return site, contract, nil
}

// resolveActionSite enforces the static action symbol before ordinary
// argument checking: the first operand is an action reference resolved
// against the checked action table, and the returned per-action contract
// drives the value-operand checks. The symbol is dropped from the value
// list; the frozen site joins the lowered step instead.
func (c *regionChecker) resolveActionSite(operation string, scope bodyScope, args []syntax.Argument, span source.Span) (*ir.ActionSite, *types.Type, []syntax.Argument, error) {
	for _, arg := range args {
		if arg.Spread || arg.Group != nil {
			return nil, nil, nil, c.locate(span, fmt.Errorf("action operation requires fixed ordinary arguments"))
		}
	}
	if len(args) == 0 {
		return nil, nil, nil, c.locate(span, fmt.Errorf("action operation requires an action symbol as its first operand"))
	}
	name, ok := fetchUngroup(args[0].Value).(*syntax.NameExpr)
	if !ok {
		return nil, nil, nil, c.locate(args[0].Value.ExprSpan(), fmt.Errorf("%s requires an action symbol as its first operand, not a string or computed name", strings.TrimPrefix(operation, "can.std.")))
	}
	if c.context.ActionSite == nil {
		return nil, nil, nil, c.locate(span, fmt.Errorf("action operation requires its calling project"))
	}
	site, contract, err := c.context.ActionSite(operation, scope.symbols, name.Name, args[1:])
	if err != nil {
		if operand, ok := err.(*actionOperandError); ok {
			if operand.Index >= 0 && operand.Index < len(args) {
				return nil, nil, nil, c.locate(args[operand.Index].Value.ExprSpan(), operand.Err)
			}
			return nil, nil, nil, c.locate(span, operand.Err)
		}
		return nil, nil, nil, c.locate(span, err)
	}
	return &site, contract, args[1:], nil
}
