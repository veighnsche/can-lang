package check

import (
	"fmt"
	"sort"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// TemplateParam is one explicitly typed fixture parameter.
type TemplateParam struct {
	ID   string
	Name string
	Type *types.Type
}

// Template is a checked P3.1 fixture template: the exact target contract
// and definition-checked rows with parameter bindings. Expansion
// substitutes use-argument IR for parameter bindings, so case statics
// keep their defining-file meaning and use arguments keep their static
// use-site meaning; no expansion re-resolves names. Cases retains the
// source rows (including raw paths and spans) for fixture locks.
type Template struct {
	Symbol *resolve.Symbol
	File   *resolve.File
	Target ValueBinding
	Params []TemplateParam
	Rows   []ir.FixtureRow
	Cases  []syntax.FixtureCase
}

// checkTemplates resolves every fixture declaration after wrapper policies
// (wrapper targets need calculated contracts) and before native and
// function bodies (whose when tables expand templates).
func (c *programChecker) checkTemplates(program *Program, callables map[string]CallableDeclaration) error {
	var files []*resolve.File
	for _, file := range c.world.Files {
		files = append(files, file)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Source.ID < files[j].Source.ID })
	for _, file := range files {
		for _, declaration := range file.Source.Syntax.Declarations {
			d, ok := declaration.(*syntax.FixtureDecl)
			if !ok {
				continue
			}
			if err := c.checkTemplate(program, file, d, callables); err != nil {
				err = stampCode(err, "CAN-CHECK-FIXTURE-DEFINITION")
				err = source.LocateCode(file.Source.Syntax.Source.Name(), d.DeclSpan(), "CAN-CHECK-FIXTURE-DEFINITION", err)
				return fmt.Errorf("fixture %s: %w", d.Name.Text, err)
			}
		}
	}
	return nil
}

func (c *programChecker) checkTemplate(program *Program, file *resolve.File, d *syntax.FixtureDecl, callables map[string]CallableDeclaration) error {
	defFile := file.Source.Syntax.Source.Name()
	symbol := file.Package.Scope.Symbols[d.Name.Text]
	if symbol == nil || symbol.Kind != resolve.Fixture {
		return source.Locate(defFile, d.Name.Span, fmt.Errorf("fixture %q has no declared symbol", d.Name.Text))
	}
	target, err := c.fixtureTarget(program, file, d)
	if err != nil {
		return source.Locate(defFile, d.Target.Span, err)
	}
	seen := map[string]bool{}
	var params []TemplateParam
	for _, field := range d.Given {
		if seen[field.Name.Text] {
			return source.Locate(defFile, field.Name.Span, fmt.Errorf("duplicate fixture parameter %q", field.Name.Text))
		}
		seen[field.Name.Text] = true
		typ, err := c.annotation(file, field.Type, false)
		if err != nil {
			return source.Locate(defFile, field.Type.TypeSpan(), err)
		}
		params = append(params, TemplateParam{ID: symbol.ID + "/given/" + field.Name.Text, Name: field.Name.Text, Type: typ})
	}
	if len(d.Cases) == 0 {
		return source.Locate(defFile, d.Name.Span, fmt.Errorf("fixture cases must be nonempty"))
	}
	scope := resolve.NewScope(file.Scope)
	for _, param := range params {
		if err := scope.Define(&resolve.Symbol{ID: param.ID, Name: param.Name, Kind: resolve.Value}); err != nil {
			return err
		}
		c.bindings[param.ID] = param.Type
	}
	for i, kase := range d.Cases {
		if err := c.inertCase(kase); err != nil {
			return source.Locate(defFile, kase.Span, fmt.Errorf("case %d: %w", i+1, err))
		}
	}
	// Cases check through the ordinary fixture path against the target
	// contract, so definition enforces identical argument, completion
	// and raw rules. The definition table is retained; expansion
	// substitutes use-argument IR into its rows.
	region := &ir.Region{ID: symbol.ID + "/definition"}
	context := CompletionContext{Identity: region.ID, Kind: ir.FunctionRegion, Package: file.Package.ID, File: file.Source.Syntax.Source, Scope: scope, Result: target.Type.Result(), Registry: program.Registry, Expressions: c.expressions(file, scope), Callables: callables, Variadic: c.variadic, Raw: c.rawScope(file)}
	checker := &regionChecker{context: context, region: region, locals: map[string]*types.Type{}, uses: LocalUses{Names: map[*syntax.NameExpr]string{}, Captures: map[*syntax.ReferenceExpr][]string{}}}
	step := &ir.InvocationStep{Site: region.ID, Identity: target.Identity, Contract: target.Type, Result: target.Type.Result(), Errors: target.Type.Errors(), Span: d.DeclSpan()}
	rows := make([]syntax.Assertion, 0, len(d.Cases))
	for i, kase := range d.Cases {
		rows = append(rows, syntax.Assertion{Span: kase.Span, Name: syntax.Token{Kind: syntax.Name, Text: fmt.Sprintf("case %d", i+1), Span: kase.Span}, Arguments: kase.Arguments, Expected: kase.Expected, Mode: kase.Mode})
	}
	table, err := checker.fixtures(rows, step, bodyScope{scope})
	if err != nil {
		return err
	}
	c.templates[symbol.ID] = &Template{Symbol: symbol, File: file, Target: target, Params: params, Rows: table.Rows, Cases: d.Cases}
	return nil
}

// fixtureTarget resolves the template target to its exact contract: an
// ordinary function (generic only with concrete header arguments),
// supported catalogue operation, fetch, judge, LLM or wrapper.
func (c *programChecker) fixtureTarget(program *Program, file *resolve.File, d *syntax.FixtureDecl) (ValueBinding, error) {
	symbol, err := file.Lookup(file.Scope, d.Target, resolve.CallUse)
	if err != nil {
		return ValueBinding{}, fmt.Errorf("target %s names no invokable fixture target: write an ordinary function, supported catalogue operation, fetch, judge, LLM or wrapper", d.Target.Name)
	}
	if symbol.Kind == resolve.Value {
		return ValueBinding{}, fmt.Errorf("target %s is a callable value, not a fixture target declaration", symbol.ID)
	}
	if len(symbol.Parameters) != 0 {
		if len(d.Types) != len(symbol.Parameters) {
			return ValueBinding{}, fmt.Errorf("generic target %s requires %d concrete type arguments", symbol.ID, len(symbol.Parameters))
		}
		return c.specialize(file, file.Scope, d.Target, d.Types)
	}
	if len(d.Types) != 0 {
		return ValueBinding{}, fmt.Errorf("target %s is not generic", symbol.ID)
	}
	switch symbol.Kind {
	case resolve.Fetch, resolve.Judge, resolve.LLM, resolve.Wrapper:
		for _, native := range program.Natives {
			if native.Symbol == symbol {
				if native.Signature == nil {
					return ValueBinding{}, fmt.Errorf("target %s has no checked contract", symbol.ID)
				}
				return ValueBinding{Identity: symbol.ID, Type: native.Signature}, nil
			}
		}
		return ValueBinding{}, fmt.Errorf("target %s is not a checked operation", symbol.ID)
	default:
		typ := c.bindings[symbol.ID]
		if typ == nil || typ.Kind() != types.Callable {
			return ValueBinding{}, fmt.Errorf("target %s has no checked callable contract", symbol.ID)
		}
		return ValueBinding{Identity: symbol.ID, Type: typ}, nil
	}
}

// inertCase enforces the inert initialization subset on case inputs and
// outcomes, with template parameters as additional immutable names. Only
// explicit success and domain completions are admitted.
func (c *programChecker) inertCase(kase syntax.FixtureCase) error {
	for _, arg := range kase.Arguments {
		if arg.Group != nil {
			for _, value := range arg.Group.Values {
				if err := inertExpression(value); err != nil {
					return err
				}
			}
			continue
		}
		if err := inertExpression(arg.Value); err != nil {
			return err
		}
	}
	switch body := kase.Expected.(type) {
	case *syntax.SuccessBody:
		if body.Value != nil {
			return inertExpression(body.Value)
		}
	case *syntax.FailureBody:
		return inertExpression(body.Error)
	default:
		return fmt.Errorf("fixture case requires an explicit success or domain completion")
	}
	return nil
}

// expandTemplateUse resolves `use template(arguments)` at a lexical when
// row: exact target lookup, static inert argument checking and
// substitution of checked argument IR into the definition rows under the
// row selector. Arguments check in the use file's static scope, so
// runtime locals are invisible; the splice preserves both static
// meanings without re-resolution.
func (c *programChecker) expandTemplateUse(file *resolve.File, scope *resolve.Scope, row syntax.Assertion) ([]ir.FixtureRow, *Template, error) {
	useFile := file.Source.Syntax.Source.Name()
	symbol, err := file.Lookup(scope, row.Use.Template, resolve.FixtureUse)
	if err != nil {
		return nil, nil, source.LocateCode(useFile, row.Use.Template.Span, "CAN-CHECK-FIXTURE-USE", err)
	}
	template := c.templates[symbol.ID]
	if template == nil {
		return nil, nil, fmt.Errorf("fixture %s has no checked template", symbol.ID)
	}
	// Use failures point at the offending use span and link the template
	// definition; no fix inserts or reorders arguments, which would guess
	// values the author must supply.
	fail := func(primary source.Span, err error) ([]ir.FixtureRow, *Template, error) {
		err = stampCode(err, "CAN-CHECK-FIXTURE-USE")
		err = source.LocateCode(useFile, primary, "CAN-CHECK-FIXTURE-USE", err)
		return nil, nil, relateTemplate(template, err)
	}
	if len(row.Use.Arguments) != len(template.Params) {
		names := make([]string, 0, len(template.Params))
		for _, param := range template.Params {
			names = append(names, param.Name)
		}
		return fail(row.Use.Span, fmt.Errorf("use of %s expects %d template arguments (%s), not %d", symbol.ID, len(template.Params), strings.Join(names, ", "), len(row.Use.Arguments)))
	}
	static := c.expressions(file, file.Scope)
	args := map[string]*ir.Expression{}
	for i, arg := range row.Use.Arguments {
		if arg.Spread || arg.Group != nil {
			return fail(arg.Span, fmt.Errorf("template arguments must be explicit positional values"))
		}
		if err := inertExpression(arg.Value); err != nil {
			return fail(arg.Value.ExprSpan(), fmt.Errorf("template argument %d is executable: %w", i+1, err))
		}
		checked, err := static.Check(arg.Value, template.Params[i].Type)
		if err != nil {
			return fail(arg.Value.ExprSpan(), fmt.Errorf("template argument %s expects %s: %w", template.Params[i].Name, template.Params[i].Type.Identity(), err))
		}
		args[template.Params[i].ID] = checked
	}
	rows := make([]ir.FixtureRow, 0, len(template.Rows))
	for i, def := range template.Rows {
		substituted, err := substituteRow(def, args, template.File.Source.ID)
		if err != nil {
			return fail(row.Use.Span, fmt.Errorf("case %d: %w", i+1, err))
		}
		substituted.Selector = row.Name.Text
		rows = append(rows, substituted)
	}
	return rows, template, nil
}

// relateTemplate links a use-site failure to its template definition,
// across files when the template is imported. Without a resolved
// definition it returns the failure unchanged.
func relateTemplate(template *Template, err error) error {
	if template == nil || template.Symbol == nil || template.Symbol.Declaration == nil || template.File == nil || template.File.Source == nil || template.File.Source.Syntax == nil || template.File.Source.Syntax.Source == nil {
		return err
	}
	return source.Relate(template.File.Source.Syntax.Source.Name(), template.Symbol.Declaration.DeclSpan(), "template defined here", err)
}

// substituteRow replaces parameter bindings in one definition row with
// copies of the checked use-argument IR. Definition nodes are tagged with
// the definition source ID so their spans keep addressing the definition
// file after expansion; use-argument copies keep the use file. Raw
// exchanges carry no parameters and are reused by value.
func substituteRow(row ir.FixtureRow, args map[string]*ir.Expression, source string) (ir.FixtureRow, error) {
	out := row
	out.Prepare = make([]ir.Preparation, 0, len(row.Prepare))
	for _, prep := range row.Prepare {
		value, err := substituteIR(prep.Value, args, source)
		if err != nil {
			return ir.FixtureRow{}, err
		}
		out.Prepare = append(out.Prepare, ir.Preparation{Local: prep.Local, Value: value})
	}
	out.Arguments = make([]*ir.Expression, 0, len(row.Arguments))
	for _, arg := range row.Arguments {
		substituted, err := substituteIR(arg, args, source)
		if err != nil {
			return ir.FixtureRow{}, err
		}
		out.Arguments = append(out.Arguments, substituted)
	}
	if row.Expected != nil {
		expected, err := substituteCompletion(row.Expected, args, source)
		if err != nil {
			return ir.FixtureRow{}, err
		}
		out.Expected = expected
	}
	if row.Raw != nil {
		raw := *row.Raw
		out.Raw = &raw
	}
	return out, nil
}

func substituteCompletion(completion *ir.Completion, args map[string]*ir.Expression, source string) (*ir.Completion, error) {
	if completion.Call != nil || completion.Block != nil || completion.Match != nil || completion.Inherit != "" {
		return nil, fmt.Errorf("template completion is outside the inert subset")
	}
	out := *completion
	if completion.Value != nil {
		value, err := substituteIR(completion.Value, args, source)
		if err != nil {
			return nil, err
		}
		out.Value = value
	}
	return &out, nil
}

// substituteIR replaces parameter bindings with copies of the checked
// use arguments. Executable IR (invocations, matches, coordination,
// callables) fails closed: inert cases cannot produce it.
func substituteIR(expr *ir.Expression, args map[string]*ir.Expression, source string) (*ir.Expression, error) {
	if expr == nil {
		return nil, nil
	}
	if expr.Kind == ir.Binding {
		if arg, ok := args[expr.Text]; ok {
			return copyIR(arg)
		}
	}
	if expr.Coordination != nil || expr.Callable != nil || expr.Invocation != nil || expr.Match != nil {
		return nil, fmt.Errorf("template expression is outside the inert subset")
	}
	out := *expr
	if source != "" {
		out.Source = source
	}
	out.Inputs = make([]*ir.Expression, 0, len(expr.Inputs))
	for _, input := range expr.Inputs {
		substituted, err := substituteIR(input, args, source)
		if err != nil {
			return nil, err
		}
		out.Inputs = append(out.Inputs, substituted)
	}
	out.Operators = append([]string(nil), expr.Operators...)
	out.Equality = append([]ir.EqualityMode(nil), expr.Equality...)
	out.Spread = append([]bool(nil), expr.Spread...)
	out.Fields = append([]string(nil), expr.Fields...)
	return &out, nil
}

// copyIR deep-copies a checked use argument so each expanded row owns its
// expressions.
func copyIR(expr *ir.Expression) (*ir.Expression, error) {
	return substituteIR(expr, nil, "")
}
