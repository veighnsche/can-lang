package check

import (
	"fmt"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

const (
	fetchJSONGet  = "can.std.http@1::fetch_json_get"
	fetchJSONPost = "can.std.http@1::fetch_json_post"
)

// FetchSpecialization admits the two T23 browser JSON fetch operations.
// Result is the finite action result variant; Data is the POST wire record
// and nil for bodyless GET. The call-site contract derives per resolved
// action: the static name, the path captures in order, and the POST body.
type FetchSpecialization struct {
	Operation string
	Result    *types.Type
	Data      *types.Type
	Contract  *types.Type
}

func fetchGenericOperation(identity string) bool {
	return identity == fetchJSONGet || identity == fetchJSONPost
}

// fetchSiteOperation recovers the JSON fetch operation from a
// specialization key, or "" when the key is not a fetch specialization.
func fetchSiteOperation(key string) string {
	operation, _, ok := strings.Cut(key, "<")
	if !ok || !fetchGenericOperation(operation) {
		return ""
	}
	return operation
}

// fetchKey names one concrete fetch use. Like the form specializations,
// the key is a plain identity join: gather runs before the graph seals,
// where digest keys cannot be computed yet.
func fetchKey(operation string, arguments []*types.Type) string {
	parts := make([]string, len(arguments))
	for i, arg := range arguments {
		parts[i] = arg.Identity()
	}
	return operation + "<" + strings.Join(parts, ",") + ">"
}

func fetchTypeArguments(operation string) int {
	if operation == fetchJSONPost {
		return 2
	}
	return 1
}

func (c *programChecker) gatherFetch(file *resolve.File, site syntax.Expr, callee syntax.Expr, args []syntax.TypeNode) error {
	name, ok := callee.(*syntax.NameExpr)
	if !ok {
		return nil
	}
	symbol, err := file.Lookup(nil, name.Name, resolve.CallUse)
	if err != nil || !fetchGenericOperation(symbol.ID) {
		return nil
	}
	want := fetchTypeArguments(symbol.ID)
	if len(args) != want {
		return locateGather(file, site.ExprSpan(), fmt.Errorf("fetch call requires %d explicit concrete type arguments", want))
	}
	arguments := make([]*types.Type, len(args))
	for i, arg := range args {
		arguments[i], err = c.gather(file, arg)
		if err != nil {
			return err
		}
		if name := types.OpaqueParameterName(arguments[i]); name != "" {
			return locateGather(file, site.ExprSpan(), fmt.Errorf("fetch %s cannot use opaque type parameter %s from an exported generic declaration", symbol.ID, name))
		}
	}
	key := fetchKey(symbol.ID, arguments)
	if c.fetches == nil {
		c.fetches = map[string]*FetchSpecialization{}
	}
	if c.fetches[key] != nil {
		return nil
	}
	special := &FetchSpecialization{Operation: symbol.ID, Result: arguments[0]}
	if symbol.ID == fetchJSONPost {
		special.Data = arguments[1]
	}
	c.fetches[key] = special
	if c.specializer != nil {
		if err = c.finishFetch(key); err != nil {
			return locateGather(file, site.ExprSpan(), err)
		}
	}
	return nil
}

// finishFetch builds the shared fetch contract from the spelled type
// arguments: the finite result variant plus the fixed failure bound.
// Actions check after every signature is gathered, so per-action capture
// contracts derive later at each call site.
func (c *programChecker) finishFetch(key string) error {
	special := c.fetches[key]
	if special.Result.Kind() != types.Variant || len(special.Result.Leaves()) == 0 {
		return fmt.Errorf("fetch %s requires a finite variant result type, not %s", special.Operation, types.CanonicalName(special.Result))
	}
	str, err := c.catalogueType("str", map[string]*types.Type{})
	if err != nil {
		return err
	}
	inputs := []*types.Type{str}
	if special.Operation == fetchJSONPost {
		if special.Data.Kind() != types.Record {
			return fmt.Errorf("fetch %s requires an ordinary record wire type, not %s", special.Operation, types.CanonicalName(special.Data))
		}
		inputs = append(inputs, special.Data)
	}
	errors, err := c.fetchFailureBound(special.Operation)
	if err != nil {
		return err
	}
	special.Contract, err = types.CallableOfChecked(special.Result, inputs, errors)
	if err != nil {
		return err
	}
	if c.callables != nil {
		names := []string{"action"}
		if special.Operation == fetchJSONPost {
			names = append(names, "body")
		}
		c.callables[key] = CallableDeclaration{Kind: resolve.Function, Contract: special.Contract, Names: names, Near: make([]bool, len(names))}
	}
	c.program.Intrinsics[key] = special.Contract
	c.program.Fetches = c.fetches
	return nil
}

// fetchFailureBound is the fixed browser-callable failure vocabulary the
// fetch adapter maps transport, abort, codec and unexpected-status
// outcomes into. Abort arrives as a cancelled transport phase: a
// cancellation is a failure, never a domain case and never a rollback
// claim. POST alone carries the request wire limit; response oversize is
// a codec failure on both verbs.
func (c *programChecker) fetchFailureBound(operation string) ([]*types.Type, error) {
	names := []string{"http::transport_failed", "http::invalid_request"}
	if operation == fetchJSONPost {
		names = append(names, "http::body_limit")
	}
	names = append(names, "http::status_error", "codec::invalid_data")
	bound := make([]*types.Type, 0, len(names))
	for _, name := range names {
		failure, err := c.catalogueType(name, map[string]*types.Type{})
		if err != nil {
			return nil, err
		}
		bound = append(bound, failure)
	}
	return bound, nil
}

func (c *programChecker) specializeFetch(file *resolve.File, scope *resolve.Scope, name syntax.QualifiedName, args []syntax.TypeNode) (ValueBinding, error) {
	symbol, err := file.Lookup(scope, name, resolve.CallUse)
	if err != nil {
		return ValueBinding{}, err
	}
	if !fetchGenericOperation(symbol.ID) || len(args) != fetchTypeArguments(symbol.ID) {
		return ValueBinding{}, fmt.Errorf("generic invocation requires specialization; fetch expects explicit types")
	}
	arguments := make([]*types.Type, len(args))
	for i, arg := range args {
		arguments[i], err = c.annotation(file, arg, false)
		if err != nil {
			return ValueBinding{}, err
		}
	}
	key := fetchKey(symbol.ID, arguments)
	special := c.fetches[key]
	if special == nil {
		return ValueBinding{}, fmt.Errorf("missing checked fetch specialization")
	}
	return ValueBinding{Identity: key, Type: special.Contract}, nil
}

// fetchSite resolves one static fetch name against the checked JSON
// action table and derives the call-site contract: the action name, the
// path captures in order, and the POST wire body. A renamed action, a
// method or wire change, and a result change diagnose here; path, case
// and schema edits regenerate the spliced contract on the next build,
// identically for both emit profiles.
func (c *programChecker) fetchSite(file *resolve.File, operation, key, name string) (ir.JSONFetchSite, *types.Type, error) {
	special := c.fetches[key]
	if special == nil {
		return ir.JSONFetchSite{}, nil, fmt.Errorf("missing checked fetch specialization")
	}
	identity := name
	if head, tail, ok := strings.Cut(name, "::"); ok {
		target := file.Imports[head]
		if target == nil && head == file.Package.Name {
			target = file.Package
		}
		if target == nil {
			return ir.JSONFetchSite{}, nil, fmt.Errorf("fetch action %q names unknown package %q", name, head)
		}
		identity = target.ID + "::" + tail
	} else {
		identity = file.Package.ID + "::" + name
	}
	var action *ActionDeclaration
	for _, candidate := range c.program.Actions {
		if candidate.Symbol.ID == identity {
			action = candidate
			break
		}
	}
	if action == nil {
		return ir.JSONFetchSite{}, nil, fmt.Errorf("unknown JSON action %q", name)
	}
	if operation == fetchJSONGet {
		if action.Method != "GET" || action.Input.Mode != "none" {
			return ir.JSONFetchSite{}, nil, fmt.Errorf("fetch action %q is not a bodyless GET action", name)
		}
	} else {
		if action.Method != "POST" || action.Input.Mode != "json" {
			return ir.JSONFetchSite{}, nil, fmt.Errorf("fetch action %q is not a JSON POST action", name)
		}
		if action.Input.Type.Identity() != special.Data.Identity() {
			return ir.JSONFetchSite{}, nil, fmt.Errorf("fetch action %q wire is %s, not %s", name, types.CanonicalName(action.Input.Type), types.CanonicalName(special.Data))
		}
	}
	if action.Returns.Identity() != special.Result.Identity() {
		return ir.JSONFetchSite{}, nil, fmt.Errorf("fetch action %q result is %s, not %s", name, types.CanonicalName(action.Returns), types.CanonicalName(special.Result))
	}
	if action.ResponseSchema == nil {
		return ir.JSONFetchSite{}, nil, fmt.Errorf("fetch action %q carries no JSON response schema", name)
	}
	site := ir.JSONFetchSite{Action: identity, Method: action.Method, Path: action.Path, Response: *action.ResponseSchema}
	str, err := c.catalogueType("str", map[string]*types.Type{})
	if err != nil {
		return ir.JSONFetchSite{}, nil, err
	}
	inputs := []*types.Type{str}
	for _, capture := range action.Captures {
		site.Captures = append(site.Captures, ir.JSONFetchCapture{Name: capture.Name, Type: capture.Type.Declaration()})
		inputs = append(inputs, capture.Type)
	}
	if operation == fetchJSONPost {
		site.Request = &action.Input.Schema
		inputs = append(inputs, special.Data)
	}
	for _, kase := range action.Cases {
		leaf, err := c.actionCaseIdentity(action, kase.Leaf)
		if err != nil {
			return ir.JSONFetchSite{}, nil, err
		}
		site.Cases = append(site.Cases, ir.JSONFetchCase{Leaf: leaf, Status: kase.Status})
	}
	bound, err := c.fetchFailureBound(operation)
	if err != nil {
		return ir.JSONFetchSite{}, nil, err
	}
	contract, err := types.CallableOfChecked(special.Result, inputs, bound)
	if err != nil {
		return ir.JSONFetchSite{}, nil, err
	}
	return site, contract, nil
}

// resolveFetchSite enforces the static action name before ordinary
// argument checking: the name is a string literal resolved against the
// checked action table, and the returned per-action contract drives the
// capture and body checks. The literal stays in the lowered arguments so
// supplied fixtures keep matching.
func (c *regionChecker) resolveFetchSite(operation, key string, args []syntax.Argument, span source.Span) (*ir.JSONFetchSite, *types.Type, error) {
	for _, arg := range args {
		if arg.Spread || arg.Group != nil {
			return nil, nil, c.locate(span, fmt.Errorf("fetch operation requires fixed ordinary arguments"))
		}
	}
	if len(args) == 0 {
		return nil, nil, c.locate(span, fmt.Errorf("fetch operation requires a static action name"))
	}
	literal, ok := fetchUngroup(args[0].Value).(*syntax.LiteralExpr)
	if !ok || literal.Token.Kind != syntax.String {
		return nil, nil, c.locate(args[0].Value.ExprSpan(), fmt.Errorf("fetch action name must be a static string literal"))
	}
	if c.context.FetchSite == nil {
		return nil, nil, c.locate(span, fmt.Errorf("fetch operation requires its calling project"))
	}
	site, contract, err := c.context.FetchSite(operation, key, literal.Token.Value)
	if err != nil {
		return nil, nil, c.locate(literal.Token.Span, err)
	}
	return &site, contract, nil
}
