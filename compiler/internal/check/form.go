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
	formNamedCollection = "can.std.form@1::named_collection"
	formNamedField      = "can.std.form@1::named_field"
	formServeAction     = "can.std.http@1::serve_form_action"
	formRowsType        = "can.std.form@1::rows"
	formRowItemType     = "can.std.form@1::row_item"
	formRejectedType    = "can.std.form@1::rejected"
)

// FormSpecialization admits the generic keyed-row operations: the two
// static name builders plus the HTML action adapter binder. Data is the
// wire or row record; Result and Rejected serve the adapter only.
type FormSpecialization struct {
	Operation string
	Result    *types.Type
	Data      *types.Type
	Contract  *types.Type
	Rejected  *types.Type
}

func formGenericOperation(identity string) bool {
	switch identity {
	case formNamedCollection, formNamedField, formServeAction:
		return true
	}
	return false
}

// formSiteOperation recovers the keyed-row operation from a specialization
// key, or "" when the key is not a form specialization.
func formSiteOperation(key string) string {
	operation, _, ok := strings.Cut(key, "<")
	if !ok || !formGenericOperation(operation) {
		return ""
	}
	return operation
}

// formKey names one concrete form use. Like the HTTP specializations, the
// key is a plain identity join: gather runs before the graph seals, where
// digest keys cannot be computed yet.
func formKey(operation string, arguments []*types.Type) string {
	parts := make([]string, len(arguments))
	for i, arg := range arguments {
		parts[i] = arg.Identity()
	}
	return operation + "<" + strings.Join(parts, ",") + ">"
}

func formTypeArguments(operation string) int {
	if operation == formServeAction {
		return 2
	}
	return 1
}

func (c *programChecker) gatherForm(file *resolve.File, site syntax.Expr, callee syntax.Expr, args []syntax.TypeNode) error {
	name, ok := callee.(*syntax.NameExpr)
	if !ok {
		return nil
	}
	symbol, err := file.Lookup(nil, name.Name, resolve.CallUse)
	if err != nil || !formGenericOperation(symbol.ID) {
		return nil
	}
	want := formTypeArguments(symbol.ID)
	if len(args) != want {
		return locateGather(file, site.ExprSpan(), fmt.Errorf("form call requires %d explicit concrete type arguments", want))
	}
	arguments := make([]*types.Type, len(args))
	for i, arg := range args {
		arguments[i], err = c.gather(file, arg)
		if err != nil {
			return err
		}
		if name := types.OpaqueParameterName(arguments[i]); name != "" {
			return locateGather(file, site.ExprSpan(), fmt.Errorf("form %s cannot use opaque type parameter %s from an exported generic declaration", symbol.ID, name))
		}
	}
	key := formKey(symbol.ID, arguments)
	if c.forms == nil {
		c.forms = map[string]*FormSpecialization{}
	}
	if c.forms[key] != nil {
		return nil
	}
	special := &FormSpecialization{Operation: symbol.ID}
	if symbol.ID == formServeAction {
		special.Result, special.Data = arguments[0], arguments[1]
	} else {
		special.Data = arguments[0]
	}
	c.forms[key] = special
	if c.specializer != nil {
		if err = c.finishForm(key); err != nil {
			return locateGather(file, site.ExprSpan(), err)
		}
	}
	return nil
}

func (c *programChecker) finishForm(key string) error {
	special := c.forms[key]
	str, err := c.catalogueType("str", map[string]*types.Type{})
	if err != nil {
		return err
	}
	switch special.Operation {
	case formNamedCollection, formNamedField:
		if special.Data.Kind() != types.Record {
			return fmt.Errorf("form %s requires an ordinary record type, not %s", special.Operation, types.CanonicalName(special.Data))
		}
		want := "form::collection"
		if special.Operation == formNamedField {
			want = "form::field"
		}
		token, err := c.catalogueType(want, map[string]*types.Type{})
		if err != nil {
			return err
		}
		unknown, err := c.catalogueType("form::unknown_field", map[string]*types.Type{})
		if err != nil {
			return err
		}
		special.Contract, err = types.CallableOfChecked(token, []*types.Type{str}, []*types.Type{unknown})
		if err != nil {
			return err
		}
		if c.callables != nil {
			c.callables[key] = CallableDeclaration{Kind: resolve.Function, Contract: special.Contract, Names: []string{"name"}, Near: []bool{false}}
		}
	case formServeAction:
		if special.Result.Kind() != types.Variant || len(special.Result.Leaves()) == 0 {
			return fmt.Errorf("form %s requires a finite variant result type, not %s", special.Operation, types.CanonicalName(special.Result))
		}
		if special.Data.Kind() != types.Record {
			return fmt.Errorf("form %s requires an ordinary record wire type, not %s", special.Operation, types.CanonicalName(special.Data))
		}
		rejected, err := c.catalogueType("form::rejected<T>", map[string]*types.Type{"T": special.Data})
		if err != nil {
			return err
		}
		safe, err := c.catalogueType("html::safe", map[string]*types.Type{})
		if err != nil {
			return err
		}
		outcome, err := types.CallableOfChecked(safe, []*types.Type{special.Result}, nil)
		if err != nil {
			return err
		}
		structural, err := types.CallableOfChecked(safe, []*types.Type{rejected}, nil)
		if err != nil {
			return err
		}
		route, err := c.catalogueType("http::route", map[string]*types.Type{})
		if err != nil {
			return err
		}
		invalid, err := c.catalogueType("http::invalid_route", map[string]*types.Type{})
		if err != nil {
			return err
		}
		special.Contract, err = types.CallableOfChecked(route, []*types.Type{str, outcome, structural}, []*types.Type{invalid})
		if err != nil {
			return err
		}
		special.Rejected = rejected
		if c.callables != nil {
			c.callables[key] = CallableDeclaration{Kind: resolve.Function, Contract: special.Contract, Names: []string{"action", "outcome", "structural"}, Near: []bool{false, false, false}}
		}
	default:
		return fmt.Errorf("unknown form specialization %s", special.Operation)
	}
	c.program.Intrinsics[key] = special.Contract
	c.program.Forms = c.forms
	return nil
}

func (c *programChecker) specializeForm(file *resolve.File, scope *resolve.Scope, name syntax.QualifiedName, args []syntax.TypeNode) (ValueBinding, error) {
	symbol, err := file.Lookup(scope, name, resolve.CallUse)
	if err != nil {
		return ValueBinding{}, err
	}
	if !formGenericOperation(symbol.ID) || len(args) != formTypeArguments(symbol.ID) {
		return ValueBinding{}, fmt.Errorf("generic invocation requires specialization; form expects explicit types")
	}
	arguments := make([]*types.Type, len(args))
	for i, arg := range args {
		arguments[i], err = c.annotation(file, arg, false)
		if err != nil {
			return ValueBinding{}, err
		}
	}
	key := formKey(symbol.ID, arguments)
	special := c.forms[key]
	if special == nil {
		return ValueBinding{}, fmt.Errorf("missing checked form specialization")
	}
	return ValueBinding{Identity: key, Type: special.Contract}, nil
}

// fillFormRowItems resolves the row_item value identities the form schema
// cannot derive on its own: the item type is never spelled in source. Both
// form-derivation sites call this once the wire record is known.
func (c *programChecker) fillFormRowItems(wire *types.Type, schema *types.FormSchema) error {
	for _, f := range wire.Fields() {
		if f.Type.Kind() != types.Record || f.Type.Declaration() != formRowsType || len(f.Type.Arguments()) != 1 {
			continue
		}
		item, err := c.catalogueType("form::row_item<T>", map[string]*types.Type{"T": f.Type.Arguments()[0]})
		if err != nil {
			return err
		}
		for i := range schema.Fields {
			if schema.Fields[i].Name == f.Name && schema.Fields[i].Rows != nil {
				schema.Fields[i].Rows.Item = item.Identity()
			}
		}
	}
	return nil
}

// formSite validates one static form name against its specialization: name
// membership for the checked builders, action agreement for the adapter.
// The adapter site carries the frozen contract the emitter splices.
func (c *programChecker) formSite(file *resolve.File, operation, key, name string) (ir.FormActionSite, error) {
	special := c.forms[key]
	if special == nil {
		return ir.FormActionSite{}, fmt.Errorf("missing checked form specialization")
	}
	switch operation {
	case formNamedCollection, formNamedField:
		return ir.FormActionSite{}, checkFormName(operation, special.Data, name)
	case formServeAction:
		return c.serveFormSite(file, special, name)
	default:
		return ir.FormActionSite{}, fmt.Errorf("unknown form operation %s", operation)
	}
}

func checkFormName(operation string, wire *types.Type, name string) error {
	var field *types.Field
	for _, f := range wire.Fields() {
		if f.Name == name {
			owned := f
			field = &owned
			break
		}
	}
	if field == nil {
		return fmt.Errorf("wire record %s has no field %q", types.CanonicalName(wire), name)
	}
	rows := field.Type.Kind() == types.Record && field.Type.Declaration() == formRowsType
	if operation == formNamedCollection {
		if !rows {
			return fmt.Errorf("wire record %s field %q is not a keyed rows collection", types.CanonicalName(wire), name)
		}
		return nil
	}
	if rows {
		return fmt.Errorf("wire record %s field %q is a keyed rows collection; address it with form::named_collection", types.CanonicalName(wire), name)
	}
	if !formScalarField(field.Type) {
		return fmt.Errorf("wire record %s field %q is not a form scalar field", types.CanonicalName(wire), name)
	}
	return nil
}

func formScalarField(typ *types.Type) bool {
	switch {
	case typ.Kind() == types.Primitive && typ.Declaration() == "str":
		return true
	case typ.Kind() == types.Array && typ.Element().Kind() == types.Primitive && typ.Element().Declaration() == "str":
		return true
	case typ.Kind() == types.Variant && typ.Declaration() == "can.std.option@1::value" && len(typ.Arguments()) == 1 && typ.Arguments()[0].Declaration() == "str":
		return true
	}
	return false
}

func (c *programChecker) serveFormSite(file *resolve.File, special *FormSpecialization, name string) (ir.FormActionSite, error) {
	identity := name
	if head, tail, ok := strings.Cut(name, "::"); ok {
		target := file.Imports[head]
		if target == nil && head == file.Package.Name {
			target = file.Package
		}
		if target == nil {
			return ir.FormActionSite{}, fmt.Errorf("form action %q names unknown package %q", name, head)
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
		return ir.FormActionSite{}, fmt.Errorf("unknown form action %q", name)
	}
	if action.Body == nil || action.Body.Mode != "form" {
		return ir.FormActionSite{}, fmt.Errorf("form action %q carries no form body", name)
	}
	if action.Result.Identity() != special.Result.Identity() {
		return ir.FormActionSite{}, fmt.Errorf("form action %q result is %s, not %s", name, types.CanonicalName(action.Result), types.CanonicalName(special.Result))
	}
	if action.Body.Type.Identity() != special.Data.Identity() {
		return ir.FormActionSite{}, fmt.Errorf("form action %q wire is %s, not %s", name, types.CanonicalName(action.Body.Type), types.CanonicalName(special.Data))
	}
	rawEntry, err := c.catalogueType("form::raw_entry", map[string]*types.Type{})
	if err != nil {
		return ir.FormActionSite{}, err
	}
	issue, err := c.catalogueType("form::issue", map[string]*types.Type{})
	if err != nil {
		return ir.FormActionSite{}, err
	}
	site := ir.FormActionSite{Action: identity, Method: action.Method, Path: action.Path, Form: action.Body.Form, Handler: action.Handler, Rejected: special.Rejected.Identity(), RawEntry: rawEntry.Identity(), Issue: issue.Identity()}
	for _, kase := range action.Cases {
		leaf, err := c.actionCaseIdentity(action, kase.Leaf)
		if err != nil {
			return ir.FormActionSite{}, err
		}
		site.Cases = append(site.Cases, ir.FormActionCase{Leaf: leaf, Status: kase.Status})
	}
	return site, nil
}

// actionCaseIdentity resolves a case leaf declaration to its runtime
// identity through the action result leaves.
func (c *programChecker) actionCaseIdentity(action *ActionDeclaration, leaf string) (string, error) {
	for _, candidate := range action.Result.Leaves() {
		if candidate.Declaration() == leaf {
			return candidate.Identity(), nil
		}
	}
	return "", fmt.Errorf("form action cannot resolve result leaf %s", leaf)
}

// resolveFormSite enforces the catalogue staticInputs contract before
// ordinary argument checking: the action or field name is a string
// literal checked against the specialization. The literal stays in the
// lowered arguments so supplied fixtures keep matching.
func (c *regionChecker) resolveFormSite(operation, key string, args []syntax.Argument, span source.Span) (*ir.FormActionSite, error) {
	want := 1
	if operation == formServeAction {
		want = 3
	}
	if len(args) != want {
		return nil, c.locate(span, fmt.Errorf("form operation requires a static name and its declared inputs"))
	}
	for _, arg := range args {
		if arg.Spread || arg.Group != nil {
			return nil, c.locate(span, fmt.Errorf("form operation requires fixed ordinary arguments"))
		}
	}
	literal, ok := fetchUngroup(args[0].Value).(*syntax.LiteralExpr)
	if !ok || literal.Token.Kind != syntax.String {
		return nil, c.locate(args[0].Value.ExprSpan(), fmt.Errorf("form name must be a static string literal"))
	}
	if c.context.FormSite == nil {
		return nil, c.locate(span, fmt.Errorf("form operation requires its calling project"))
	}
	site, err := c.context.FormSite(operation, key, literal.Token.Value)
	if err != nil {
		return nil, c.locate(literal.Token.Span, err)
	}
	if operation != formServeAction {
		return nil, nil
	}
	return &site, nil
}
