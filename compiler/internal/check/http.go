package check

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/veighnsche/can-lang/compiler/internal/catalogue"
	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

const (
	httpRequestJSON  = "can.std.http@1::request_json"
	httpRequestForm  = "can.std.http@1::request_form"
	httpResponseJSON = "can.std.http@1::response_json"
	httpRouteGet     = "can.std.http@1::route_get"
	httpRoutePost    = "can.std.http@1::route_post"
)

// HTTPSpecialization admits only the three generic I32 catalogue operations.
// Each key carries its concrete data type plus the JSON or form descriptor
// the emitter supplies to the shared runtime boundary.
type HTTPSpecialization struct {
	Operation string
	Data      *types.Type
	Contract  *types.Type
	Schema    types.CodecSchema
	Form      types.FormSchema
}

type httpParts struct {
	request  *types.Type
	integer  *types.Type
	status   *types.Type
	headers  *types.Type
	response *types.Type
	errors   []*types.Type
}

func httpGenericOperation(identity string) bool {
	return identity == httpRequestJSON || identity == httpRequestForm || identity == httpResponseJSON
}

func routeOperation(identity string) bool {
	return identity == httpRouteGet || identity == httpRoutePost
}

// isScopeRequest reports whether the type is an ingress-only harness scope
// value. The set is intentionally explicit: I32 admits http::request; later
// ingress types extend this predicate with their own admission task.
func isScopeRequest(typ *types.Type) bool {
	return typ != nil && typ.Kind() == types.Opaque && typ.Declaration() == "can.std.http@1::request"
}

// expandScopeArguments binds provided assertion-row arguments to non-scope
// inputs in order, splicing one synthetic harness scope value at every
// ingress-only position. Elision is total: no Can expression can be
// scope-typed outside an ingress scope, so a row either supplies exactly the
// assertable arguments or it is an arity error. Rows for scope-elided
// functions use fixed arguments or literal spreads; runtime-length spreads
// cannot be matched positionally against gapped inputs.
func expandScopeArguments(count int, elided func(int) bool, variadic bool, args []syntax.Argument, span source.Span) ([]syntax.Argument, error) {
	scoped := false
	for i := 0; i < count; i++ {
		if elided(i) {
			scoped = true
			break
		}
	}
	if !scoped {
		return args, nil
	}
	elements, err := fixedArgumentElements(args)
	if err != nil {
		return nil, err
	}
	out := make([]syntax.Argument, 0, count)
	at := 0
	assertable := 0
	for i := 0; i < count; i++ {
		if elided(i) {
			out = append(out, syntax.Argument{Span: span, Value: &syntax.ScopeExpr{ExpressionLocation: syntax.ExpressionLocation{Span: span}}})
			continue
		}
		assertable++
		if variadic && i == count-1 {
			out = append(out, elements[at:]...)
			at = len(elements)
			continue
		}
		if at >= len(elements) {
			return nil, fmt.Errorf("assertion arguments omit a required input")
		}
		out = append(out, elements[at])
		at++
	}
	if at != len(elements) {
		return nil, fmt.Errorf("assertion supplies %d arguments for %d assertable inputs", len(elements), assertable)
	}
	return out, nil
}

func (c *programChecker) scopeElided(file *resolve.File, node syntax.TypeNode) bool {
	if c.specializer == nil {
		return false
	}
	typ, err := c.specializer.Resolve(file, node, c.parameters(), false)
	if err != nil {
		return false
	}
	return isScopeRequest(typ)
}

func parseCatalogueType(text string) (syntax.TypeNode, error) {
	src, err := source.New("can:catalogue", text)
	if err != nil {
		return nil, err
	}
	node, ds := syntax.ParseType(src)
	if len(ds) != 0 {
		return nil, fmt.Errorf("invalid catalogue signature: %v", ds)
	}
	return node, nil
}

// admitRouteOperation builds the fixed mount contract without feeding the
// catalogue $callback placeholder to the ordinary type parser: the callback
// input is an explicitly constructed (http::request) -> http::server_response
// callable with an empty error bound, per P10.
func (c *programChecker) admitRouteOperation(p *Program, builtin *resolve.File, op catalogue.Operation) error {
	str, err := parseCatalogueType("str")
	if err != nil {
		return err
	}
	request, err := parseCatalogueType("http::request")
	if err != nil {
		return err
	}
	response, err := parseCatalogueType("http::server_response")
	if err != nil {
		return err
	}
	route, err := parseCatalogueType("http::route")
	if err != nil {
		return err
	}
	invalid, err := parseCatalogueType("http::invalid_route")
	if err != nil {
		return err
	}
	callback := &syntax.CallableType{Result: response, Inputs: []syntax.TypeNode{request}}
	signature := &syntax.CallableType{Result: route, Inputs: []syntax.TypeNode{str, callback}}
	signature.Errors.Types = []syntax.TypeNode{invalid}
	typ, err := c.gather(builtin, signature)
	if err != nil {
		return err
	}
	c.bindings[op.Identity] = typ
	p.Intrinsics[op.Identity] = typ
	return nil
}

func (c *programChecker) gatherHTTP(file *resolve.File, callee syntax.Expr, args []syntax.TypeNode) error {
	name, ok := callee.(*syntax.NameExpr)
	if !ok {
		return nil
	}
	symbol, err := file.Lookup(nil, name.Name, resolve.CallUse)
	if err != nil || !httpGenericOperation(symbol.ID) {
		return nil
	}
	if len(args) != 1 {
		return fmt.Errorf("HTTP call requires one explicit concrete type argument")
	}
	data, err := c.gather(file, args[0])
	if err != nil {
		return err
	}
	key := symbol.ID + "<" + data.Identity() + ">"
	if c.https == nil {
		c.https = map[string]*HTTPSpecialization{}
		c.httpParts = map[string]*httpParts{}
	}
	if c.https[key] != nil {
		return nil
	}
	// Catalogue dependencies resolve in the maintained package namespace, not the
	// author's imports: using HTTP does not require redundant utility imports.
	builtin := &resolve.File{Scope: c.world.Prelude, Imports: c.world.Packages}
	c.annotations[builtin] = map[string]*types.Type{}
	resolve := func(text string) (*types.Type, error) {
		node, err := parseCatalogueType(text)
		if err != nil {
			return nil, err
		}
		return c.gather(builtin, node)
	}
	parts := &httpParts{}
	if parts.request, err = resolve("http::request"); err != nil {
		return err
	}
	if parts.integer, err = resolve("int"); err != nil {
		return err
	}
	if parts.status, err = resolve("http::body_status"); err != nil {
		return err
	}
	if parts.headers, err = resolve("http::server_headers"); err != nil {
		return err
	}
	if parts.response, err = resolve("http::server_response"); err != nil {
		return err
	}
	for _, failure := range []string{"http::body_limit", "http::invalid_request", "codec::invalid_data"} {
		typ, err := resolve(failure)
		if err != nil {
			return err
		}
		parts.errors = append(parts.errors, typ)
	}
	// Residual signatures can be derived after graph sealing; retain ingredients.
	c.https[key] = &HTTPSpecialization{Operation: symbol.ID, Data: data}
	c.httpParts[key] = parts
	if c.specializer != nil {
		if err = c.finishHTTP(key); err != nil {
			return err
		}
	}
	return nil
}

func (c *programChecker) finishHTTP(key string) error {
	special := c.https[key]
	parts := c.httpParts[key]
	var err error
	switch special.Operation {
	case httpRequestJSON, httpResponseJSON:
		special.Schema, err = types.Schema(special.Data)
		if err != nil {
			return err
		}
	case httpRequestForm:
		special.Form, err = types.Form(special.Data)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown HTTP specialization %s", special.Operation)
	}
	names := []string{"request", "max_bytes"}
	inputs := []*types.Type{parts.request, parts.integer}
	result := special.Data
	errors := parts.errors
	if special.Operation == httpResponseJSON {
		names = []string{"status", "headers", "body"}
		inputs = []*types.Type{parts.status, parts.headers, special.Data}
		result = parts.response
		errors = parts.errors[2:]
	}
	special.Contract, err = types.CallableOfChecked(result, inputs, errors)
	if err != nil {
		return err
	}
	c.program.Intrinsics[key] = special.Contract
	c.program.HTTPs = c.https
	if c.callables != nil {
		c.callables[key] = CallableDeclaration{Kind: resolve.Function, Contract: special.Contract, Names: names, Near: make([]bool, len(names))}
	}
	return nil
}

func (c *programChecker) specializeHTTP(file *resolve.File, scope *resolve.Scope, name syntax.QualifiedName, args []syntax.TypeNode) (ValueBinding, error) {
	symbol, err := file.Lookup(scope, name, resolve.CallUse)
	if err != nil {
		return ValueBinding{}, err
	}
	if !httpGenericOperation(symbol.ID) || len(args) != 1 {
		return ValueBinding{}, fmt.Errorf("generic invocation requires specialization; HTTP expects one explicit type")
	}
	data, err := c.annotation(file, args[0], false)
	if err != nil {
		return ValueBinding{}, err
	}
	key := symbol.ID + "<" + data.Identity() + ">"
	special := c.https[key]
	if special == nil {
		return ValueBinding{}, fmt.Errorf("missing checked HTTP specialization")
	}
	return ValueBinding{Identity: key, Type: special.Contract}, nil
}

// checkRouteMount enforces the catalogue staticInputs contract before ordinary
// argument checking: the path is a string literal whose normalization is
// validated here, and the callback is a named reference. Generic reference
// inference and explicit type arguments still flow through the ordinary
// callback checker; the runtime mount constructor re-validates and remains
// authoritative for normalization edge cases.
func checkRouteMount(args []syntax.Argument) error {
	if len(args) != 2 {
		return fmt.Errorf("route mount requires a static path and a named callback")
	}
	for _, arg := range args {
		if arg.Spread || arg.Group != nil {
			return fmt.Errorf("route mount requires fixed ordinary arguments")
		}
	}
	path, ok := fetchUngroup(args[0].Value).(*syntax.LiteralExpr)
	if !ok || path.Token.Kind != syntax.String {
		return fmt.Errorf("route path must be a static string literal")
	}
	if err := checkRoutePath(path.Token.Value); err != nil {
		return err
	}
	if _, ok := fetchUngroup(args[1].Value).(*syntax.ReferenceExpr); !ok {
		return fmt.Errorf("route callback must be a named reference")
	}
	return nil
}

// checkRoutePath mirrors the runtime mount pre-checks plus its
// decodeURIComponent/normalizedPath sequence. URL dot-segment normalization
// is deliberately left to the runtime: anything accepted here but rejected
// there still fails safely at mount time.
func checkRoutePath(path string) error {
	fail := func() error { return fmt.Errorf("invalid static route path") }
	if !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") {
		return fail()
	}
	for _, r := range path {
		if r <= 0x20 || r == 0x7f || r == '\\' || r == '?' || r == '#' || r == '*' {
			return fail()
		}
	}
	for _, segment := range strings.Split(path, "/")[1:] {
		if strings.HasPrefix(segment, ":") {
			return fail()
		}
	}
	decoded := make([]byte, 0, len(path))
	for i := 0; i < len(path); {
		if path[i] != '%' {
			decoded = append(decoded, path[i])
			i++
			continue
		}
		if i+2 >= len(path) {
			return fail()
		}
		hi, ok := hexValue(path[i+1])
		if !ok {
			return fail()
		}
		lo, ok := hexValue(path[i+2])
		if !ok {
			return fail()
		}
		decoded = append(decoded, hi<<4|lo)
		i += 3
	}
	if !utf8.Valid(decoded) || len(decoded) == 0 || decoded[0] != '/' {
		return fail()
	}
	for _, r := range string(decoded) {
		if r < 0x20 || r == 0x7f || r == '\\' {
			return fail()
		}
	}
	if string(decoded) == "/__can" || strings.HasPrefix(string(decoded), "/__can/") {
		return fail()
	}
	return nil
}

func hexValue(b byte) (byte, bool) {
	switch {
	case b >= '0' && b <= '9':
		return b - '0', true
	case b >= 'a' && b <= 'f':
		return b - 'a' + 10, true
	case b >= 'A' && b <= 'F':
		return b - 'A' + 10, true
	}
	return 0, false
}
