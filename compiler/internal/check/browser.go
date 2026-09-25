package check

import (
	"fmt"
	"math/big"
	"strings"
	"unicode/utf8"

	"github.com/veighnsche/can-lang/compiler/internal/catalogue"
	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// T22 bounded browser catalogue operations. Listener operations carry a
// named Can callback and are admitted with an explicitly constructed
// contract like HTTP route mounts; state operations specialize per
// concrete data type like collections.
const (
	browserMount           = "can.std.browser@1::mount"
	browserRoot            = "can.std.browser@1::root"
	browserOpenView        = "can.std.browser@1::open_view"
	browserDisposeView     = "can.std.browser@1::dispose_view"
	browserDisposeApp      = "can.std.browser@1::dispose_app"
	browserCreateElement   = "can.std.browser@1::create_element"
	browserCreateText      = "can.std.browser@1::create_text"
	browserSetText         = "can.std.browser@1::set_text"
	browserSetAttribute    = "can.std.browser@1::set_attribute"
	browserRemoveAttribute = "can.std.browser@1::remove_attribute"
	browserAppendChild     = "can.std.browser@1::append_child"
	browserRemoveNode      = "can.std.browser@1::remove_node"
	browserFocus           = "can.std.browser@1::focus"
	browserOnEvent         = "can.std.browser@1::on_event"
	browserSetTimeout      = "can.std.browser@1::set_timeout"
	browserCreateState     = "can.std.browser@1::create_state"
	browserReadState       = "can.std.browser@1::read_state"
	browserReplaceState    = "can.std.browser@1::replace_state"
	browserQueryParameter  = "can.std.browser@1::query_parameter"
	browserOnCancelKey     = "can.std.browser@1::on_cancel_key"
	browserOnCancelEvent   = "can.std.browser@1::on_cancel_event"
	browserInvalidQuery    = "can.std.browser@1::invalid_query"
)

// browserMaxDelayMs is the largest setTimeout delay the catalogue admits:
// the native timer range. Larger values fail instead of silently clamping.
const browserMaxDelayMs = 2147483647

// browserTags mirrors the shared author tag vocabulary in
// runtime/platform/html.ts. Script, style and other non-author tags are
// absent on both sides; runtime/test/browser-names.json pins the union.
var browserTags = map[string]bool{
	"main": true, "header": true, "footer": true, "nav": true, "section": true,
	"article": true, "aside": true, "h1": true, "h2": true, "h3": true,
	"h4": true, "h5": true, "h6": true, "p": true, "div": true, "span": true,
	"ul": true, "ol": true, "li": true, "a": true, "form": true, "label": true,
	"input": true, "textarea": true, "select": true, "option": true, "button": true,
	"table": true, "thead": true, "tbody": true, "tr": true, "th": true, "td": true,
	"dl": true, "dt": true, "dd": true, "strong": true, "em": true, "small": true,
	"br": true, "hr": true, "code": true, "pre": true, "blockquote": true,
	"img": true, "del": true,
}

// browserAttributes mirrors the live-DOM name rule: the html global
// attributes, the applicability keys, the URL attributes and aria-* names.
// Admission is name-level: per-tag applicability and value shapes are
// html-string-builder rules, not live-DOM safety rules.
var browserAttributes = map[string]bool{
	"id": true, "class": true, "title": true, "lang": true, "dir": true,
	"hidden": true, "tabindex": true, "role": true,
	"name": true, "value": true, "type": true, "placeholder": true,
	"autocomplete": true, "for": true, "method": true, "rel": true,
	"checked": true, "selected": true, "disabled": true, "required": true,
	"multiple": true, "rows": true, "cols": true, "scope": true,
	"colspan": true, "rowspan": true, "alt": true, "align": true, "start": true,
	"href": true, "action": true, "formaction": true, "src": true,
}

// browserURLAttributes carry navigable URLs and are scheme-checked.
var browserURLAttributes = map[string]bool{
	"href": true, "action": true, "formaction": true, "src": true,
}

// browserEvents is the bounded listener vocabulary shared with the runtime.
var browserEvents = map[string]bool{
	"click": true, "dblclick": true, "input": true, "change": true,
	"keydown": true, "keyup": true, "focus": true, "blur": true, "submit": true,
}

func browserListenerOperation(identity string) bool {
	return identity == browserOnEvent || identity == browserSetTimeout ||
		identity == browserOnCancelKey || identity == browserOnCancelEvent
}

// browserStateOperation resolves the three generic state operations for
// per-type specialization. The prefix guard keeps the inventory scan off
// every unrelated call path.
func browserStateOperation(identity string) *catalogue.Operation {
	if !strings.HasPrefix(identity, "can.std.browser@1::") {
		return nil
	}
	if identity != browserCreateState && identity != browserReadState && identity != browserReplaceState {
		return nil
	}
	for _, op := range catalogue.Builtin().Inventory().Operations {
		if op.Identity == identity && op.Lowering.Task == "T22" {
			operation := op
			return &operation
		}
	}
	return nil
}

func browserStaticOperation(identity string) bool {
	switch identity {
	case browserCreateElement, browserSetAttribute, browserRemoveAttribute, browserOnEvent, browserSetTimeout,
		browserQueryParameter, browserOnCancelKey, browserOnCancelEvent:
		return true
	}
	return false
}

// browserLower folds ASCII uppercase exactly like the runtime admission.
func browserLower(value string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'A' && r <= 'Z' {
			return r + ('a' - 'A')
		}
		return r
	}, value)
}

func checkBrowserTag(tag string) error {
	if browserTags[browserLower(tag)] {
		return nil
	}
	return fmt.Errorf("browser tag %q is not admitted", tag)
}

func checkBrowserAttributeName(name string) error {
	lowered := browserLower(name)
	if browserAttributes[lowered] || browserAriaName(lowered) {
		return nil
	}
	return fmt.Errorf("browser attribute %q is not admitted", name)
}

// browserAriaName mirrors the runtime aria rule: aria- plus a lowercase
// letter followed by lowercase letters, digits or dashes.
func browserAriaName(lowered string) bool {
	rest, ok := strings.CutPrefix(lowered, "aria-")
	if !ok || len(rest) == 0 || rest[0] < 'a' || rest[0] > 'z' {
		return false
	}
	for i := 1; i < len(rest); i++ {
		c := rest[i]
		if c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' {
			continue
		}
		return false
	}
	return true
}

func checkBrowserEvent(kind string) error {
	if browserEvents[browserLower(kind)] {
		return nil
	}
	return fmt.Errorf("browser event %q is not admitted", kind)
}

// checkBrowserQueryKey enforces the compile-time literal key rule for
// browser::query_parameter: ASCII [a-z][a-z0-9_]*, at most 64 bytes.
// Dynamic keys are rejected before this runs; the runtime re-checks the
// literal plus the strict location.search budgets.
func checkBrowserQueryKey(key string) error {
	if len(key) == 0 || len(key) > 64 {
		return fmt.Errorf("browser query key %q must be 1-64 ASCII bytes", key)
	}
	if key[0] < 'a' || key[0] > 'z' {
		return fmt.Errorf("browser query key %q must match [a-z][a-z0-9_]*", key)
	}
	for i := 1; i < len(key); i++ {
		c := key[i]
		if c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_' {
			continue
		}
		return fmt.Errorf("browser query key %q must match [a-z][a-z0-9_]*", key)
	}
	return nil
}

// checkBrowserCancelKeyEvent admits only keydown/keyup for on_cancel_key.
func checkBrowserCancelKeyEvent(kind string) error {
	switch browserLower(kind) {
	case "keydown", "keyup":
		return nil
	}
	return fmt.Errorf("browser cancel key event %q is not admitted; expected keydown or keyup", kind)
}

// checkBrowserCancelEvent admits only submit for on_cancel_event.
func checkBrowserCancelEvent(kind string) error {
	if browserLower(kind) == "submit" {
		return nil
	}
	return fmt.Errorf("browser cancel event %q is not admitted; expected submit", kind)
}

// checkBrowserURLValue mirrors the runtime URL rule for URL-valued
// attributes: same-origin relative references or https. Anything else,
// including protocol-relative and javascript: values, is rejected.
func checkBrowserURLValue(value string) error {
	fail := func() error { return fmt.Errorf("browser URL attribute value is not admitted") }
	if value == "" || !utf8.ValidString(value) {
		return fail()
	}
	for i := 0; i < len(value); i++ {
		c := value[i]
		if c <= 0x20 || c == 0x7f || c == '\\' {
			return fail()
		}
	}
	if strings.HasPrefix(value, "//") {
		return fail()
	}
	if strings.HasPrefix(value, "/") {
		return nil
	}
	if len(value) >= 8 && strings.EqualFold(value[:8], "https://") {
		return nil
	}
	return fail()
}

// admitBrowserListener builds the fixed listener contract without feeding
// the catalogue $callback placeholder to the ordinary type parser: the
// callback input is an explicitly constructed callable with an empty
// error bound, like HTTP route mounts.
func (c *programChecker) admitBrowserListener(p *Program, builtin *resolve.File, op catalogue.Operation) error {
	str, err := parseCatalogueType("str")
	if err != nil {
		return err
	}
	integer, err := parseCatalogueType("int")
	if err != nil {
		return err
	}
	view, err := parseCatalogueType("browser::view")
	if err != nil {
		return err
	}
	node, err := parseCatalogueType("browser::node")
	if err != nil {
		return err
	}
	event, err := parseCatalogueType("browser::event")
	if err != nil {
		return err
	}
	result, err := parseCatalogueType("void")
	if err != nil {
		return err
	}
	disposed, err := parseCatalogueType("browser::disposed")
	if err != nil {
		return err
	}
	rejected, err := parseCatalogueType("browser::rejected")
	if err != nil {
		return err
	}
	var signature *syntax.CallableType
	switch op.Identity {
	case browserOnEvent:
		callback := &syntax.CallableType{Result: result, Inputs: []syntax.TypeNode{event}}
		signature = &syntax.CallableType{Result: result, Inputs: []syntax.TypeNode{view, node, str, callback}}
	case browserSetTimeout:
		callback := &syntax.CallableType{Result: result}
		signature = &syntax.CallableType{Result: result, Inputs: []syntax.TypeNode{view, integer, callback}}
	case browserOnCancelKey:
		callback := &syntax.CallableType{Result: result, Inputs: []syntax.TypeNode{event}}
		signature = &syntax.CallableType{Result: result, Inputs: []syntax.TypeNode{view, node, str, str, callback}}
	case browserOnCancelEvent:
		callback := &syntax.CallableType{Result: result, Inputs: []syntax.TypeNode{event}}
		signature = &syntax.CallableType{Result: result, Inputs: []syntax.TypeNode{view, node, str, callback}}
	default:
		return fmt.Errorf("unknown browser listener %s", op.Identity)
	}
	signature.Errors.Types = []syntax.TypeNode{disposed, rejected}
	typ, err := c.gather(builtin, signature)
	if err != nil {
		return err
	}
	c.bindings[op.Identity] = typ
	p.Intrinsics[op.Identity] = typ
	return nil
}

// checkBrowserCall enforces the static half of browser admission before
// ordinary argument checking: literal tags, attribute names, event kinds,
// URL values and delays are validated here with precise literal spans,
// and listener callbacks must be named references. Dynamic values flow
// through to the runtime admission checks; the runtime remains
// authoritative for normalization edge cases.
func (c *regionChecker) checkBrowserCall(identity string, args []syntax.Argument, span source.Span) error {
	fixed := func(count int) ([]syntax.Argument, error) {
		if len(args) != count {
			return nil, c.locate(span, fmt.Errorf("browser call requires %d fixed arguments", count))
		}
		for _, arg := range args {
			if arg.Spread || arg.Group != nil {
				return nil, c.locate(span, fmt.Errorf("browser call requires fixed ordinary arguments"))
			}
		}
		return args, nil
	}
	literal := func(arg syntax.Argument) *syntax.LiteralExpr {
		expression, ok := fetchUngroup(arg.Value).(*syntax.LiteralExpr)
		if !ok {
			return nil
		}
		return expression
	}
	switch identity {
	case browserCreateElement:
		fixedArgs, err := fixed(2)
		if err != nil {
			return err
		}
		if tag := literal(fixedArgs[1]); tag != nil && tag.Token.Kind == syntax.String {
			if err := checkBrowserTag(tag.Token.Value); err != nil {
				return c.locate(tag.Token.Span, err)
			}
		}
		return nil
	case browserSetAttribute:
		fixedArgs, err := fixed(3)
		if err != nil {
			return err
		}
		name := literal(fixedArgs[1])
		if name != nil && name.Token.Kind == syntax.String {
			if err := checkBrowserAttributeName(name.Token.Value); err != nil {
				return c.locate(name.Token.Span, err)
			}
			if value := literal(fixedArgs[2]); value != nil && value.Token.Kind == syntax.String && browserURLAttributes[browserLower(name.Token.Value)] {
				if err := checkBrowserURLValue(value.Token.Value); err != nil {
					return c.locate(value.Token.Span, err)
				}
			}
		}
		return nil
	case browserRemoveAttribute:
		fixedArgs, err := fixed(2)
		if err != nil {
			return err
		}
		if name := literal(fixedArgs[1]); name != nil && name.Token.Kind == syntax.String {
			if err := checkBrowserAttributeName(name.Token.Value); err != nil {
				return c.locate(name.Token.Span, err)
			}
		}
		return nil
	case browserOnEvent:
		fixedArgs, err := fixed(4)
		if err != nil {
			return err
		}
		if kind := literal(fixedArgs[2]); kind != nil && kind.Token.Kind == syntax.String {
			if err := checkBrowserEvent(kind.Token.Value); err != nil {
				return c.locate(kind.Token.Span, err)
			}
		}
		if _, ok := fetchUngroup(fixedArgs[3].Value).(*syntax.ReferenceExpr); !ok {
			return c.locate(span, fmt.Errorf("browser event callback must be a named reference"))
		}
		return nil
	case browserSetTimeout:
		fixedArgs, err := fixed(3)
		if err != nil {
			return err
		}
		if delay, literalSpan, ok := browserDelayLiteral(fixedArgs[1].Value); ok {
			if delay.Sign() < 0 || !delay.IsInt64() || delay.Int64() > browserMaxDelayMs {
				return c.locate(literalSpan, fmt.Errorf("browser delay must be 0-%d milliseconds", browserMaxDelayMs))
			}
		}
		if _, ok := fetchUngroup(fixedArgs[2].Value).(*syntax.ReferenceExpr); !ok {
			return c.locate(span, fmt.Errorf("browser timer callback must be a named reference"))
		}
		return nil
	case browserQueryParameter:
		fixedArgs, err := fixed(1)
		if err != nil {
			return err
		}
		key := literal(fixedArgs[0])
		if key == nil || key.Token.Kind != syntax.String {
			return c.locate(span, fmt.Errorf("browser query key must be a static literal"))
		}
		if err := checkBrowserQueryKey(key.Token.Value); err != nil {
			return c.locate(key.Token.Span, err)
		}
		return nil
	case browserOnCancelKey:
		fixedArgs, err := fixed(5)
		if err != nil {
			return err
		}
		if kind := literal(fixedArgs[2]); kind != nil && kind.Token.Kind == syntax.String {
			if err := checkBrowserCancelKeyEvent(kind.Token.Value); err != nil {
				return c.locate(kind.Token.Span, err)
			}
		}
		if key := literal(fixedArgs[3]); key != nil && key.Token.Kind == syntax.String {
			if key.Token.Value == "" {
				return c.locate(key.Token.Span, fmt.Errorf("browser cancel key must be a nonempty exact key"))
			}
		}
		if _, ok := fetchUngroup(fixedArgs[4].Value).(*syntax.ReferenceExpr); !ok {
			return c.locate(span, fmt.Errorf("browser cancel callback must be a named reference"))
		}
		return nil
	case browserOnCancelEvent:
		fixedArgs, err := fixed(4)
		if err != nil {
			return err
		}
		if kind := literal(fixedArgs[2]); kind != nil && kind.Token.Kind == syntax.String {
			if err := checkBrowserCancelEvent(kind.Token.Value); err != nil {
				return c.locate(kind.Token.Span, err)
			}
		}
		if _, ok := fetchUngroup(fixedArgs[3].Value).(*syntax.ReferenceExpr); !ok {
			return c.locate(span, fmt.Errorf("browser cancel callback must be a named reference"))
		}
		return nil
	default:
		return nil
	}
}

// browserDelayLiteral reads a static integer delay through grouping and a
// leading sign. Anything else is dynamic and checked at runtime.
func browserDelayLiteral(expression syntax.Expr) (*big.Int, source.Span, bool) {
	switch node := expression.(type) {
	case *syntax.LiteralExpr:
		if node.Token.Kind != syntax.Integer {
			return nil, source.Span{}, false
		}
		value, ok := new(big.Int).SetString(node.Token.Text, 10)
		if !ok {
			return nil, source.Span{}, false
		}
		return value, node.Token.Span, true
	case *syntax.GroupExpr:
		return browserDelayLiteral(node.Value)
	case *syntax.UnaryExpr:
		value, _, ok := browserDelayLiteral(node.Operand)
		if !ok {
			return nil, source.Span{}, false
		}
		if node.Operator == "-" {
			return value.Neg(value), node.Span, true
		}
		if node.Operator == "+" {
			return value, node.Span, true
		}
	}
	return nil, source.Span{}, false
}

// BrowserStateSpecialization is one checked browser::state<T> operation
// instance. The emitter binds each key to its per-type browser state
// value, which carries the concrete state and snapshot identities the
// runtime brands tokens and records with.
type BrowserStateSpecialization struct {
	Operation string
	Data      *types.Type
	Contract  *types.Type
	State     *types.Type
	Snapshot  *types.Type
}

func (c *programChecker) browserStateSignature(op *catalogue.Operation) (*resolve.File, []string, *syntax.CallableType, error) {
	file := &resolve.File{Scope: c.world.Prelude, Imports: c.world.Packages}
	var original, names []string
	for _, p := range op.Parameters {
		original = append(original, p.Name)
		names = append(names, "catalogue_"+strings.ToLower(p.Name))
	}
	signature := &syntax.CallableType{}
	var err error
	signature.Result, err = catalogueTypeNode(op.Result, original)
	if err != nil {
		return nil, nil, nil, err
	}
	for _, input := range op.Inputs {
		node, e := catalogueTypeNode(input.Type, original)
		if e != nil {
			return nil, nil, nil, e
		}
		signature.Inputs = append(signature.Inputs, node)
	}
	for _, failure := range op.Emits {
		node, e := catalogueTypeNode(failure, original)
		if e != nil {
			return nil, nil, nil, e
		}
		signature.Errors.Types = append(signature.Errors.Types, node)
	}
	return file, names, signature, nil
}

func (c *programChecker) instantiateBrowserState(op *catalogue.Operation, args []*types.Type) (ValueBinding, error) {
	if len(args) != len(op.Parameters) {
		return ValueBinding{}, fmt.Errorf("browser state type argument count mismatch")
	}
	parameters := map[string]*types.Type{}
	for i, p := range op.Parameters {
		parameters[p.Name] = args[i]
	}
	key, err := types.SpecializationKey(op.Identity, args)
	if err != nil {
		return ValueBinding{}, err
	}
	if old := c.program.BrowserStates[key]; old != nil {
		return ValueBinding{Identity: key, Type: old.Contract}, nil
	}
	result, err := c.catalogueType(op.Result, parameters)
	if err != nil {
		return ValueBinding{}, err
	}
	var inputs, errors []*types.Type
	for _, input := range op.Inputs {
		typ, e := c.catalogueType(input.Type, parameters)
		if e != nil {
			return ValueBinding{}, e
		}
		inputs = append(inputs, typ)
	}
	for _, failure := range op.Emits {
		typ, e := c.catalogueType(failure, parameters)
		if e != nil {
			return ValueBinding{}, e
		}
		errors = append(errors, typ)
	}
	contract, err := types.CallableOfChecked(result, inputs, errors)
	if err != nil {
		return ValueBinding{}, err
	}
	state, err := c.catalogueType("browser::state<T>", parameters)
	if err != nil {
		return ValueBinding{}, err
	}
	snapshot, err := c.catalogueType("browser::snapshot<T>", parameters)
	if err != nil {
		return ValueBinding{}, err
	}
	if c.program.BrowserStates == nil {
		c.program.BrowserStates = map[string]*BrowserStateSpecialization{}
	}
	c.program.BrowserStates[key] = &BrowserStateSpecialization{Operation: op.Identity, Data: args[0], Contract: contract, State: state, Snapshot: snapshot}
	c.program.Intrinsics[key] = contract
	c.bindings[key] = contract
	descriptor := CallableDeclaration{Kind: resolve.Function, Contract: contract}
	for _, input := range op.Inputs {
		descriptor.Names = append(descriptor.Names, input.Name)
		descriptor.Near = append(descriptor.Near, false)
	}
	c.callables[key] = descriptor
	return ValueBinding{Identity: key, Type: contract}, nil
}

func (c *programChecker) inferBrowserState(op *catalogue.Operation, args []syntax.Argument, expected *types.Type, e *Expressions) (ValueBinding, bool, error) {
	file, names, signature, err := c.browserStateSignature(op)
	if err != nil {
		return ValueBinding{}, true, err
	}
	var inputs []syntax.Input
	for _, node := range signature.Inputs {
		inputs = append(inputs, syntax.Input{Field: syntax.Field{Type: node}})
	}
	constraints, err := genericArguments(inputs, args)
	if err != nil {
		return ValueBinding{}, true, err
	}
	arguments, err := c.inferArguments(file, names, signature.Result, expected, constraints, e)
	if err != nil {
		return ValueBinding{}, true, err
	}
	binding, err := c.instantiateBrowserState(op, arguments)
	return binding, true, err
}

func (c *programChecker) referenceBrowserState(op *catalogue.Operation, expected *types.Type, inputs []*types.Type, result *types.Type) (ValueBinding, bool, error) {
	file, names, signature, err := c.browserStateSignature(op)
	if err != nil {
		return ValueBinding{}, true, err
	}
	solver, err := types.NewInference(names)
	if err != nil {
		return ValueBinding{}, true, err
	}
	constrain := func(node syntax.TypeNode, actual *types.Type) error {
		if actual == nil {
			return nil
		}
		pattern, err := types.Pattern(file, node, names)
		if err != nil {
			return err
		}
		return solver.Constrain(pattern, actual)
	}
	if expected != nil {
		err = constrain(signature, expected)
	} else {
		if len(inputs) != len(signature.Inputs) {
			return ValueBinding{}, true, fmt.Errorf("browser state callback arity mismatch")
		}
		for i, input := range inputs {
			if err = constrain(signature.Inputs[i], input); err != nil {
				break
			}
		}
		if err == nil {
			err = constrain(signature.Result, result)
		}
	}
	if err != nil {
		return ValueBinding{}, true, err
	}
	arguments, err := solver.Arguments()
	if err != nil {
		return ValueBinding{}, true, err
	}
	binding, err := c.instantiateBrowserState(op, arguments)
	return binding, true, err
}
