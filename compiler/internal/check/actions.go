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

// ActionCapture is one checked path capture: a whole-segment :name with a
// str or int type from the captures record. Typed dispatch and URL building
// consume this metadata.
type ActionCapture struct {
	Name string
	Type *types.Type
}

// ActionInput is the checked request line: a none, json or form mode plus,
// for wire modes, the record type crossing the wire, its derived schema
// and the declared byte and row limits.
type ActionInput struct {
	Mode      string
	Type      *types.Type
	Schema    types.CodecSchema
	Form      types.FormSchema
	Limit     int
	RowsLimit int
}

// ActionCase maps one returns leaf declaration to its wire status plus the
// HTML-only visible swap policy ("inner", or "" for JSON cases).
type ActionCase struct {
	Leaf   string
	Status int
	Swap   string
}

// ActionDeclaration is the checked handler-free action contract: method,
// path template, captures record, request input, finite returns variant,
// response body mode and exhaustive case table. The canonical locked
// package identity plus declaration name (Symbol.ID) keys the metadata;
// adapters consume the emitted table, and handlers bind later at
// action::mount. ResponseSchema is the shared JSON wire schema of the
// returns variant for JSON-mode actions; it is nil for HTML actions,
// whose outcome rendering belongs to the form adapter.
type ActionDeclaration struct {
	Symbol         *resolve.Symbol
	Method         string
	Path           string
	CapturesType   *types.Type
	Captures       []ActionCapture
	Input          ActionInput
	Returns        *types.Type
	Body           string
	Cases          []ActionCase
	ResponseSchema *types.CodecSchema
}

// checkActions validates every action declaration after function signatures
// are gathered and the type graph is sealed. Files arrive in sorted order,
// so the checked table and duplicate-route diagnostics are deterministic.
// Declarations carry no handlers, so a shared contract package checks and
// exports its actions without importing anything executable.
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
	capturesType, checked, err := c.checkActionCaptures(file, d, captures, fail)
	if err != nil {
		return nil, "", err
	}
	input, err := c.checkActionInput(file, d, fail)
	if err != nil {
		return nil, "", err
	}
	if err := checkActionBodyAgreement(d, input.Mode, fail); err != nil {
		return nil, "", err
	}
	returns, err := c.annotation(file, d.Returns, false)
	if err != nil {
		return nil, "", fail(d.Returns.TypeSpan(), err)
	}
	if returns.Kind() != types.Variant {
		return nil, "", fail(d.Returns.TypeSpan(), fmt.Errorf("action returns must be a finite variant, not %s", types.CanonicalName(returns)))
	}
	leaves := returns.Leaves()
	if len(leaves) == 0 {
		return nil, "", fail(d.Returns.TypeSpan(), fmt.Errorf("action returns %s has no finite leaves", types.CanonicalName(returns)))
	}
	cases, err := c.checkActionCases(file, d, returns, leaves, fail)
	if err != nil {
		return nil, "", err
	}
	var response *types.CodecSchema
	if d.Response.Text == "json" {
		schema, err := types.Schema(returns)
		if err != nil {
			return nil, "", fail(d.Returns.TypeSpan(), fmt.Errorf("action returns %s is not a JSON response type: %w", types.CanonicalName(returns), err))
		}
		response = &schema
	}
	return &ActionDeclaration{Symbol: symbol, Method: method, Path: d.Path.Value, CapturesType: capturesType, Captures: checked, Input: input, Returns: returns, Body: d.Response.Text, Cases: cases, ResponseSchema: response}, method + " " + shape, nil
}

// checkActionCaptures binds every :name path capture to its captures-record
// field in path order. Each capture is one strict decoded segment matched
// by a required str or canonical int64 int field; record and path agree
// exactly, so neither an unbound field nor an untyped capture survives.
func (c *programChecker) checkActionCaptures(file *resolve.File, d *syntax.ActionDecl, captures []string, fail func(source.Span, error) error) (*types.Type, []ActionCapture, error) {
	if len(captures) == 0 {
		if d.Captures != nil {
			return nil, nil, fail(d.Captures.TypeSpan(), fmt.Errorf("action path declares no captures"))
		}
		return nil, nil, nil
	}
	if d.Captures == nil {
		return nil, nil, fail(d.Path.Span, fmt.Errorf("action requires a captures record"))
	}
	record, err := c.annotation(file, d.Captures, false)
	if err != nil {
		return nil, nil, fail(d.Captures.TypeSpan(), err)
	}
	if record.Kind() != types.Record {
		return nil, nil, fail(d.Captures.TypeSpan(), fmt.Errorf("action captures must be a record type, not %s", types.CanonicalName(record)))
	}
	fields := map[string]*types.Type{}
	for _, field := range record.Fields() {
		typ := field.Type
		fields[field.Name] = typ
	}
	var checked []ActionCapture
	for _, name := range captures {
		typ, ok := fields[name]
		if !ok {
			return nil, nil, fail(d.Captures.TypeSpan(), fmt.Errorf("path capture :%s has no captures field", name))
		}
		if typ.Kind() != types.Primitive || (typ.Declaration() != "str" && typ.Declaration() != "int") {
			return nil, nil, fail(d.Captures.TypeSpan(), fmt.Errorf("capture %s must be str or int", name))
		}
		checked = append(checked, ActionCapture{Name: name, Type: typ})
	}
	for _, field := range record.Fields() {
		found := false
		for _, name := range captures {
			if name == field.Name {
				found = true
				break
			}
		}
		if !found {
			return nil, nil, fail(d.Captures.TypeSpan(), fmt.Errorf("capture %s does not appear in the action path", field.Name))
		}
	}
	return record, checked, nil
}

// checkActionInput validates the one request line: `input none` carries no
// wire type, while json and form lines name a record wire type with a byte
// limit. A form wire with keyed rows additionally requires rows_limit;
// rows_limit without rows is rejected.
func (c *programChecker) checkActionInput(file *resolve.File, d *syntax.ActionDecl, fail func(source.Span, error) error) (ActionInput, error) {
	mode := d.Input.Mode.Text
	if mode == "input" {
		return ActionInput{Mode: "none"}, nil
	}
	typ, err := c.annotation(file, d.Input.Type, false)
	if err != nil {
		return ActionInput{}, fail(d.Input.Type.TypeSpan(), err)
	}
	if typ.Kind() != types.Record {
		return ActionInput{}, fail(d.Input.Type.TypeSpan(), fmt.Errorf("action %s input must be a record wire type, not %s", mode, types.CanonicalName(typ)))
	}
	limit, err := strconv.Atoi(d.Input.Limit.Text)
	if err != nil || limit < 1 {
		return ActionInput{}, fail(d.Input.Limit.Span, fmt.Errorf("action limit must be a positive byte count"))
	}
	input := ActionInput{Mode: mode, Type: typ, Limit: limit}
	switch mode {
	case "json":
		schema, err := types.Schema(typ)
		if err != nil {
			return ActionInput{}, fail(d.Input.Type.TypeSpan(), err)
		}
		input.Schema = schema
	case "form":
		form, err := types.Form(typ)
		if err != nil {
			return ActionInput{}, fail(d.Input.Type.TypeSpan(), err)
		}
		if err := c.fillFormRowItems(typ, &form); err != nil {
			return ActionInput{}, fail(d.Input.Type.TypeSpan(), err)
		}
		input.Form = form
		rows := false
		for _, field := range form.Fields {
			if field.Rows != nil {
				rows = true
				break
			}
		}
		if rows && d.Input.RowsLimit == nil {
			return ActionInput{}, fail(d.Input.Type.TypeSpan(), fmt.Errorf("form input with keyed rows requires rows_limit"))
		}
		if !rows && d.Input.RowsLimit != nil {
			return ActionInput{}, fail(d.Input.RowsLimit.Span, fmt.Errorf("form input without keyed rows takes no rows_limit"))
		}
		if d.Input.RowsLimit != nil {
			bound, err := strconv.Atoi(d.Input.RowsLimit.Text)
			if err != nil || bound < 1 || bound > types.FormMaxRows {
				return ActionInput{}, fail(d.Input.RowsLimit.Span, fmt.Errorf("action rows_limit must be 1-%d", types.FormMaxRows))
			}
			input.RowsLimit = bound
		}
	default:
		return ActionInput{}, fail(d.Input.Mode.Span, fmt.Errorf("unknown action input mode %s", mode))
	}
	return input, nil
}

// checkActionBodyAgreement ties each input mode to its response body mode:
// bodyless GET and JSON POST answer JSON, form POST answers HTML. GET with
// a wire input is already rejected by the grammar.
func checkActionBodyAgreement(d *syntax.ActionDecl, mode string, fail func(source.Span, error) error) error {
	body := d.Response.Text
	switch mode {
	case "none":
		if body != "json" {
			return fail(d.Response.Span, fmt.Errorf("GET actions use body json"))
		}
	case "json":
		if body != "json" {
			return fail(d.Response.Span, fmt.Errorf("json input requires body json"))
		}
	case "form":
		if body != "html" {
			return fail(d.Response.Span, fmt.Errorf("form input requires body html"))
		}
	}
	return nil
}

func (c *programChecker) checkActionCases(file *resolve.File, d *syntax.ActionDecl, returns *types.Type, leaves []*types.Type, fail func(source.Span, error) error) ([]ActionCase, error) {
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
			return nil, fail(kase.Leaf.Span, fmt.Errorf("case %s is not a leaf of returns %s", spelling, types.CanonicalName(returns)))
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
		swap := ""
		if kase.Swap != nil {
			if d.Response.Text != "html" {
				return nil, fail(kase.Swap.Span, fmt.Errorf("swap applies to html actions only"))
			}
			swap = "inner"
		} else if d.Response.Text == "html" {
			return nil, fail(kase.Span, fmt.Errorf("html action cases require swap inner"))
		}
		cases = append(cases, ActionCase{Leaf: typ.Declaration(), Status: status, Swap: swap})
	}
	var missing []string
	for _, leaf := range leaves {
		if !seen[leaf.Identity()] {
			missing = append(missing, types.CanonicalName(leaf))
		}
	}
	if len(missing) != 0 {
		sort.Strings(missing)
		return nil, fail(d.Cases[0].Span, fmt.Errorf("action cases omit returns leaves %s", strings.Join(missing, ", ")))
	}
	return cases, nil
}

// actionRouteShape validates an action path and returns its duplicate-key
// shape plus capture names in segment order. Static segments follow the
// exact-path contract; a capture occupies one whole :name segment, and
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
		if i == 0 {
			continue
		}
		if strings.Contains(segment, "{") || strings.Contains(segment, "}") {
			return "", nil, fmt.Errorf("captures occupy one whole :name segment")
		}
		if !strings.Contains(segment, ":") {
			continue
		}
		if !strings.HasPrefix(segment, ":") || len(segment) < 2 {
			return "", nil, fmt.Errorf("captures occupy one whole :name segment")
		}
		name := segment[1:]
		if !actionCaptureName(name) {
			return "", nil, fmt.Errorf("invalid path capture :%s", name)
		}
		if seen[name] {
			return "", nil, fmt.Errorf("duplicate path capture :%s", name)
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
