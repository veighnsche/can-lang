package check

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// ActionCapture is one checked path capture: a whole-segment {name} with a
// str or int type. Typed dispatch and URL building consume this metadata.
type ActionCapture struct {
	Name string
	Type *types.Type
}

// ActionBody is the checked POST wire contract: a json or form mode plus
// the record type crossing the wire and its derived schema.
type ActionBody struct {
	Mode   string
	Type   *types.Type
	Schema types.CodecSchema
	Form   types.FormSchema
}

// ActionCase maps one result leaf declaration to its wire status.
type ActionCase struct {
	Leaf   string
	Status int
}

// ActionDeclaration is the checked action contract: method, path template,
// captures, body, total handler, finite result and exhaustive case table.
// Actions carry no body; adapters consume the emitted metadata.
type ActionDeclaration struct {
	Symbol   *resolve.Symbol
	Method   string
	Path     string
	Captures []ActionCapture
	Body     *ActionBody
	Handler  string
	Result   *types.Type
	Cases    []ActionCase
}

// checkActions validates every action declaration after function signatures
// are gathered and the type graph is sealed. Files arrive in sorted order,
// so the checked table and duplicate-route diagnostics are deterministic.
func (c *programChecker) checkActions(files []*resolve.File) error {
	seen := map[string]string{}
	type priorShape struct {
		name     string
		method   string
		segments []string
	}
	var prior []priorShape
	for _, file := range files {
		for _, declaration := range file.Source.Syntax.Declarations {
			d, ok := declaration.(*syntax.ActionDecl)
			if !ok {
				continue
			}
			action, shape, err := c.checkAction(file, d)
			if err != nil {
				return err
			}
			if prev, dup := seen[shape]; dup {
				return source.Locate(file.Source.Path, d.Path.Span, fmt.Errorf("action %s duplicates the %s route of action %s", action.Symbol.ID, shape, prev))
			}
			segments := strings.Split(strings.SplitN(shape, " ", 2)[1], "/")
			for _, prev := range prior {
				if prev.method != action.Method {
					continue
				}
				if actionShapesAmbiguous(prev.segments, segments) {
					return source.Locate(file.Source.Path, d.Path.Span, fmt.Errorf("action %s ambiguously overlaps the %s route of action %s", action.Symbol.ID, shape, prev.name))
				}
			}
			seen[shape] = action.Symbol.ID
			prior = append(prior, priorShape{name: action.Symbol.ID, method: action.Method, segments: segments})
			c.program.Actions = append(c.program.Actions, action)
		}
	}
	return nil
}

// actionShapesAmbiguous reports whether two same-method route shapes can
// match one path without a static-priority winner. Equal shapes are
// duplicates, not ambiguous; the duplicate check runs first.
func actionShapesAmbiguous(first, second []string) bool {
	if len(first) != len(second) {
		return false
	}
	same := true
	for i := range first {
		if first[i] != "{}" && second[i] != "{}" {
			if first[i] != second[i] {
				return false
			}
			continue
		}
		// Captures share one shape position: names normalize away.
		if (first[i] == "{}") != (second[i] == "{}") {
			same = false
		}
	}
	if same {
		return false
	}
	covers := func(outer, inner []string) bool {
		outerStatics, innerStatics := 0, 0
		for i := range outer {
			if outer[i] != "{}" {
				outerStatics++
			}
			if inner[i] != "{}" {
				innerStatics++
			}
			if outer[i] == "{}" && inner[i] != "{}" {
				return false
			}
		}
		return outerStatics > innerStatics
	}
	return !covers(first, second) && !covers(second, first)
}

func (c *programChecker) checkAction(file *resolve.File, d *syntax.ActionDecl) (*ActionDeclaration, string, error) {
	symbol := file.Package.Scope.Symbols[d.Name.Text]
	defFile := file.Source.Syntax.Source.Name()
	fail := func(span source.Span, err error) error {
		return source.Locate(defFile, span, fmt.Errorf("action %s: %w", d.Name.Text, err))
	}
	method := "GET"
	if d.Method.Text == "post" {
		method = "POST"
	}
	shape, captures, err := actionRouteShape(d.Path.Value)
	if err != nil {
		return nil, "", fail(d.Path.Span, err)
	}
	declared := map[string]syntax.Field{}
	for _, capture := range d.Captures {
		declared[capture.Name.Text] = capture
	}
	var checked []ActionCapture
	for _, name := range captures {
		field, ok := declared[name]
		if !ok {
			return nil, "", fail(d.Path.Span, fmt.Errorf("path capture {%s} has no captures row", name))
		}
		typ, err := c.annotation(file, field.Type, false)
		if err != nil {
			return nil, "", fail(field.Type.TypeSpan(), err)
		}
		if typ.Kind() != types.Primitive || (typ.Declaration() != "str" && typ.Declaration() != "int") {
			return nil, "", fail(field.Type.TypeSpan(), fmt.Errorf("capture %s must be str or int", name))
		}
		checked = append(checked, ActionCapture{Name: name, Type: typ})
	}
	for _, capture := range d.Captures {
		found := false
		for _, name := range captures {
			if name == capture.Name.Text {
				found = true
				break
			}
		}
		if !found {
			return nil, "", fail(capture.Span, fmt.Errorf("capture %s does not appear in the action path", capture.Name.Text))
		}
	}
	var body *ActionBody
	if d.Body != nil {
		typ, err := c.annotation(file, d.Body.Type, false)
		if err != nil {
			return nil, "", fail(d.Body.Type.TypeSpan(), err)
		}
		if typ.Kind() != types.Record {
			return nil, "", fail(d.Body.Type.TypeSpan(), fmt.Errorf("action body must be a record wire type, not %s", types.CanonicalName(typ)))
		}
		body = &ActionBody{Mode: d.Body.Mode.Text, Type: typ}
		switch body.Mode {
		case "json":
			schema, err := types.Schema(typ)
			if err != nil {
				return nil, "", fail(d.Body.Type.TypeSpan(), err)
			}
			body.Schema = schema
		case "form":
			form, err := types.Form(typ)
			if err != nil {
				return nil, "", fail(d.Body.Type.TypeSpan(), err)
			}
			body.Form = form
		}
	}
	result, err := c.annotation(file, d.Result, false)
	if err != nil {
		return nil, "", fail(d.Result.TypeSpan(), err)
	}
	if result.Kind() != types.Variant {
		return nil, "", fail(d.Result.TypeSpan(), fmt.Errorf("action result must be a finite variant, not %s", types.CanonicalName(result)))
	}
	leaves := result.Leaves()
	if len(leaves) == 0 {
		return nil, "", fail(d.Result.TypeSpan(), fmt.Errorf("action result %s has no finite leaves", types.CanonicalName(result)))
	}
	cases, err := c.checkActionCases(file, d, result, leaves, fail)
	if err != nil {
		return nil, "", err
	}
	handler, err := c.checkActionHandler(file, d, checked, body, result, fail)
	if err != nil {
		return nil, "", err
	}
	return &ActionDeclaration{Symbol: symbol, Method: method, Path: d.Path.Value, Captures: checked, Body: body, Handler: handler, Result: result, Cases: cases}, method + " " + shape, nil
}

func (c *programChecker) checkActionCases(file *resolve.File, d *syntax.ActionDecl, result *types.Type, leaves []*types.Type, fail func(source.Span, error) error) ([]ActionCase, error) {
	admitted := map[string]bool{}
	for _, leaf := range leaves {
		admitted[leaf.Identity()] = true
	}
	seen := map[string]bool{}
	var cases []ActionCase
	for _, kase := range d.Cases {
		spelling := kase.Leaf.Name
		if kase.Leaf.Package != "" {
			spelling = kase.Leaf.Package + "::" + kase.Leaf.Name
		}
		typ, err := c.annotation(file, &syntax.NamedType{Span: kase.Span, Name: kase.Leaf}, false)
		if err != nil {
			return nil, fail(kase.Leaf.Span, err)
		}
		if !admitted[typ.Identity()] {
			return nil, fail(kase.Leaf.Span, fmt.Errorf("case %s is not a leaf of result %s", spelling, types.CanonicalName(result)))
		}
		if seen[typ.Identity()] {
			return nil, fail(kase.Leaf.Span, fmt.Errorf("duplicate case for leaf %s", spelling))
		}
		seen[typ.Identity()] = true
		status, err := strconv.Atoi(kase.Status.Text)
		if err != nil || status < 200 || status > 599 {
			return nil, fail(kase.Status.Span, fmt.Errorf("action status must be 200-599"))
		}
		if status == 204 || status == 205 || status == 304 {
			return nil, fail(kase.Status.Span, fmt.Errorf("action status %d carries no representation; every action case renders a body", status))
		}
		cases = append(cases, ActionCase{Leaf: typ.Declaration(), Status: status})
	}
	var missing []string
	for _, leaf := range leaves {
		if !seen[leaf.Identity()] {
			missing = append(missing, types.CanonicalName(leaf))
		}
	}
	if len(missing) != 0 {
		sort.Strings(missing)
		return nil, fail(d.Cases[0].Span, fmt.Errorf("action cases omit result leaves %s", strings.Join(missing, ", ")))
	}
	return cases, nil
}

func (c *programChecker) checkActionHandler(file *resolve.File, d *syntax.ActionDecl, captures []ActionCapture, body *ActionBody, result *types.Type, fail func(source.Span, error) error) (string, error) {
	handler, err := file.Lookup(nil, d.Handler, resolve.ReferenceUse)
	if err != nil {
		return "", fail(d.Handler.Span, err)
	}
	if handler.Kind != resolve.Function {
		return "", fail(d.Handler.Span, fmt.Errorf("action handler %s must be a function", handler.ID))
	}
	if len(handler.Parameters) != 0 {
		return "", fail(d.Handler.Span, fmt.Errorf("action handler %s must be non-generic", handler.ID))
	}
	if c.variadic[handler.ID] {
		return "", fail(d.Handler.Span, fmt.Errorf("action handler %s must not be variadic", handler.ID))
	}
	contract := c.bindings[handler.ID]
	if contract == nil {
		return "", fail(d.Handler.Span, fmt.Errorf("action handler %s has no checked signature", handler.ID))
	}
	descriptor, ok := c.callables[handler.ID]
	if !ok || len(descriptor.Names) != len(contract.Inputs()) {
		return "", fail(d.Handler.Span, fmt.Errorf("action handler %s has no checked inputs", handler.ID))
	}
	want := len(captures)
	if body != nil {
		want++
	}
	inputs := contract.Inputs()
	if len(inputs) != want {
		return "", fail(d.Handler.Span, fmt.Errorf("action handler %s takes %d inputs, want %d capture and body inputs", handler.ID, len(inputs), want))
	}
	for i, capture := range captures {
		if descriptor.Names[i] != capture.Name {
			return "", fail(d.Handler.Span, fmt.Errorf("action handler %s input %d must be capture %s", handler.ID, i, capture.Name))
		}
		if inputs[i].Identity() != capture.Type.Identity() {
			return "", fail(d.Handler.Span, fmt.Errorf("action handler %s capture %s must be %s", handler.ID, capture.Name, types.CanonicalName(capture.Type)))
		}
	}
	if body != nil {
		last := inputs[len(inputs)-1]
		if last.Identity() != body.Type.Identity() {
			return "", fail(d.Handler.Span, fmt.Errorf("action handler %s body input must be %s", handler.ID, types.CanonicalName(body.Type)))
		}
	}
	if contract.Result().Identity() != result.Identity() {
		return "", fail(d.Handler.Span, fmt.Errorf("action handler %s must return %s", handler.ID, types.CanonicalName(result)))
	}
	if len(contract.Errors()) != 0 {
		return "", fail(d.Handler.Span, fmt.Errorf("action handler %s must emit []; map every outcome into %s", handler.ID, types.CanonicalName(result)))
	}
	return handler.ID, nil
}

// actionRouteShape validates an action path and returns its duplicate-key
// shape plus capture names in segment order. Static segments follow the
// exact-path contract; a capture occupies one whole {name} segment, and
// same-shape routes collide even when capture names differ. The shape key
// carries decoded statics, so escaped and plain spellings of one path
// collide too.
func actionRouteShape(path string) (string, []string, error) {
	segments := strings.Split(path, "/")
	var captures []string
	seen := map[string]bool{}
	shape := make([]string, len(segments))
	copy(shape, segments)
	for i, segment := range segments {
		if i == 0 || (!strings.Contains(segment, "{") && !strings.Contains(segment, "}")) {
			continue
		}
		if !strings.HasPrefix(segment, "{") || !strings.HasSuffix(segment, "}") || len(segment) < 3 {
			return "", nil, fmt.Errorf("captures occupy one whole {name} segment")
		}
		name := segment[1 : len(segment)-1]
		if !actionCaptureName(name) {
			return "", nil, fmt.Errorf("invalid path capture {%s}", name)
		}
		if seen[name] {
			return "", nil, fmt.Errorf("duplicate path capture {%s}", name)
		}
		seen[name] = true
		captures = append(captures, name)
		shape[i] = "{}"
	}
	substituted := make([]string, len(segments))
	copy(substituted, segments)
	for i, segment := range shape {
		if segment == "{}" {
			substituted[i] = "capture"
		}
	}
	if err := checkRoutePath(strings.Join(substituted, "/")); err != nil {
		return "", nil, fmt.Errorf("invalid action route path")
	}
	// Collision shapes compare decoded statics: escaped and plain spellings
	// of one path are the same route. checkRoutePath already proved every
	// escape decodes, so unescaping cannot fail here.
	key := make([]string, len(shape))
	for i, segment := range shape {
		if segment == "{}" {
			key[i] = "{}"
			continue
		}
		decoded, err := url.PathUnescape(segment)
		if err != nil {
			return "", nil, fmt.Errorf("invalid action route path")
		}
		key[i] = decoded
	}
	return strings.Join(key, "/"), captures, nil
}

func actionCaptureName(name string) bool {
	if name == "" || name[0] < 'a' || name[0] > 'z' {
		return false
	}
	for i := 1; i < len(name); i++ {
		c := name[i]
		if c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_' {
			continue
		}
		return false
	}
	return true
}
