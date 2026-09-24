package check

import (
	"fmt"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// Assertions checks every row before execution. The actual side is a checked
// invocation of the declared function, not a copy of its expected completion.
// Receiver/variadic input rules therefore use the ordinary invocation checker.
func (c *programChecker) assertions(file *resolve.File, fn *ProgramFunction, context CompletionContext) ([]*ir.Assertion, error) {
	declaration := fn.Symbol.Declaration.(*syntax.FunctionDecl)
	if err := validateAssertionNames(declaration); err != nil {
		return nil, err
	}
	return c.assertionRows(file, fn, context, declaration.Assertions)
}

func (c *programChecker) assertionRows(file *resolve.File, fn *ProgramFunction, context CompletionContext, rows []syntax.Assertion) ([]*ir.Assertion, error) {
	declaration := fn.Symbol.Declaration.(*syntax.FunctionDecl)
	var result []*ir.Assertion
	for _, row := range rows {
		root := ir.AssertionRoot{Package: fn.Symbol.Package.ID, Declaration: fn.Symbol.ID, Name: row.Name.Text}
		if row.Use != nil {
			return nil, fmt.Errorf("assertion %s: fixture templates expand at lexical when tables only", row.Name.Text)
		}
		if row.Scenario != nil {
			return nil, fmt.Errorf("assertion %s: scenario rows belong to when tables, not assertion roots", row.Name.Text)
		}
		links, err := scenarioLinks(file, row)
		if err != nil {
			return nil, err
		}
		root.Links = links
		if row.Mode != nil && row.Mode.Failure != nil {
			return nil, fmt.Errorf("assertion %s: using failure is confined to attached wrapper assertions", row.Name.Text)
		}
		if row.Mode != nil {
			return nil, fmt.Errorf("assertion %s: using raw is allowed on fetch/judge/LLM/wrapper targets only", row.Name.Text)
		}
		contract := c.bindings[fn.Identity()]
		if contract != nil && contract.Kind() == types.Callable {
			if success, ok := row.Expected.(*syntax.SuccessBody); ok && success.Value != nil {
				if isScopeRequest(contract.Result()) {
					return nil, fmt.Errorf("assertion %s: ingress scope results cannot be named in expected completions", row.Name.Text)
				}
				if kind := contract.Result().Kind(); kind == types.Opaque || kind == types.Callable {
					return nil, fmt.Errorf("assertion %s: excluded opaque results use bare ok", row.Name.Text)
				}
			}
		}
		var callee syntax.Expr = &syntax.NameExpr{Name: syntax.QualifiedName{Name: declaration.Name.Text}, ExpressionLocation: syntax.ExpressionLocation{Span: row.Span}}
		receiver := 0
		if declaration.Receiver != nil {
			receiver = 1
			scope := contract != nil && contract.Kind() == types.Callable && len(contract.Inputs()) > 0 && isScopeRequest(contract.Inputs()[0])
			if row.Receiver == nil {
				if !scope {
					return nil, fmt.Errorf("method assertion requires a receiver")
				}
				callee = &syntax.FieldExpr{Receiver: &syntax.ScopeExpr{ExpressionLocation: syntax.ExpressionLocation{Span: row.Span}}, Field: declaration.Name, ExpressionLocation: syntax.ExpressionLocation{Span: row.Span}}
			} else {
				if scope {
					return nil, fmt.Errorf("assertion %s: ingress scope receivers are harness-supplied", row.Name.Text)
				}
				callee = &syntax.FieldExpr{Receiver: row.Receiver, Field: declaration.Name, ExpressionLocation: syntax.ExpressionLocation{Span: row.Span}}
			}
		} else if row.Receiver != nil {
			return nil, fmt.Errorf("ordinary function assertion cannot supply a receiver")
		}
		expanded := row.Arguments
		if contract != nil && contract.Kind() == types.Callable {
			inputs := contract.Inputs()
			var err error
			expanded, err = expandScopeArguments(len(inputs)-receiver, func(i int) bool {
				return isScopeRequest(inputs[i+receiver])
			}, c.variadic[fn.Identity()], row.Arguments, row.Span)
			if err != nil {
				return nil, fmt.Errorf("assertion %s arguments: %w", row.Name.Text, err)
			}
		}
		call := &syntax.CallExpr{ExpressionLocation: syntax.ExpressionLocation{Span: row.Span}, Invocation: syntax.Invocation{Span: row.Span, Callee: callee, Arguments: expanded}}
		actualContext := context
		actualContext.Identity = fn.Symbol.ID + "/assert/" + row.Name.Text + "/actual"
		actualContext.Sites = nil // This synthetic root call has its own syntax tree.
		actualContext.Scope = file.Scope
		actualContext.Parameters = nil
		actualContext.Expressions = c.expressions(file, file.Scope)
		actual, err := CheckRegion(actualContext, syntax.Block{Span: row.Span, Terminal: &syntax.RelayBody{BodyLocation: syntax.BodyLocation{Span: row.Span}, Call: call}})
		if err != nil {
			return nil, fmt.Errorf("assertion %s arguments: %w", row.Name.Text, err)
		}
		switch row.Expected.(type) {
		case *syntax.SuccessBody, *syntax.FailureBody:
		default:
			return nil, fmt.Errorf("assertion requires an explicit expected completion")
		}
		expectedContext := actualContext
		expectedContext.Identity = fn.Symbol.ID + "/assert/" + row.Name.Text + "/expected"
		expectedContext.BareOpaque = true
		expected, err := CheckRegion(expectedContext, syntax.Block{Span: row.Span, Terminal: row.Expected})
		if err != nil {
			return nil, fmt.Errorf("assertion %s expected completion: %w", row.Name.Text, err)
		}
		result = append(result, &ir.Assertion{Root: root, Actual: actual, Expected: expected})
	}
	return result, nil
}

// A basic table belongs to one checked invocation. The complete concurrent
// scheduler, participant paths and captured callable instances are I18's pass.
func (c *regionChecker) fixtures(rows []syntax.Assertion, step *ir.InvocationStep, scope bodyScope) (*ir.FixtureTable, error) {
	if step.Site == "" {
		return nil, fmt.Errorf("fixture requires a checked lexical call site")
	}
	table := &ir.FixtureTable{Identity: step.Site + "/when"}
	var receiver *ir.Expression
	if step.Receiver {
		receiver = step.Arguments[0]
	}
	for _, row := range rows {
		if len(row.Links) != 0 {
			return nil, fmt.Errorf("fixture %s: scenario links attach to assertion roots, not when rows", row.Name.Text)
		}
		scenario := ""
		if row.Scenario != nil {
			symbol, err := c.context.Scope.Lookup(row.Scenario.Text, resolve.ScenarioUse)
			if err != nil {
				return nil, fmt.Errorf("fixture %s: scenario %q is not declared in this package", row.Name.Text, row.Scenario.Text)
			}
			scenario = symbol.ID
		}
		if row.Use != nil {
			if err := c.expandUse(table, step, scope, row, scenario); err != nil {
				return nil, err
			}
			continue
		}
		if err := c.fixtureRow(table, step, scope, row, receiver, scenario); err != nil {
			return nil, err
		}
	}
	return table, nil
}

// expandUse expands `use template(arguments)` in place under the row
// selector after enforcing exact target identity. Rows arrive fully
// checked from the definition; expansion owns no queue, and rows keep
// the site's identity, fingerprints and FIFO order.
func (c *regionChecker) expandUse(table *ir.FixtureTable, step *ir.InvocationStep, scope bodyScope, row syntax.Assertion, scenario string) error {
	if c.context.Expand == nil {
		return fmt.Errorf("fixture %s: template expansion is unavailable here", row.Name.Text)
	}
	expanded, template, err := c.context.Expand(c.context.Scope, row)
	if err != nil {
		return fmt.Errorf("fixture %s: %w", row.Name.Text, err)
	}
	if template.Target.Identity != step.Identity {
		err := fmt.Errorf("fixture %s: use of %s targets %s, not the invoked %s (template defined at %s)", row.Name.Text, template.Symbol.ID, template.Target.Identity, step.Identity, template.File.Source.ID)
		err = source.LocateCode(c.context.File.Name(), row.Use.Span, "CAN-CHECK-FIXTURE-USE", err)
		return relateTemplate(template, err)
	}
	// Expanded completions re-home to the use region exactly as literal
	// rows would; their value expressions are region-independent.
	// Expansion owns no queue: rows take the use site's owner and the
	// row's scenario tag, never the template definition's package.
	for i := range expanded {
		if expanded[i].Expected != nil {
			expanded[i].Expected.RegionID = c.region.ID
		}
		expanded[i].Owner = c.context.Package
		expanded[i].Scenario = scenario
	}
	table.Rows = append(table.Rows, expanded...)
	return nil
}

func (c *regionChecker) fixtureRow(table *ir.FixtureTable, step *ir.InvocationStep, scope bodyScope, row syntax.Assertion, receiver *ir.Expression, scenario string) error {
	if row.Receiver != nil || row.Name.Text == "" {
		return fmt.Errorf("fixture receiver is implicit and selector must be named")
	}
	prepared, args, err := c.arguments(c.expressions(scope), ValueBinding{Identity: step.Identity, Type: step.Contract}, row.Arguments, receiver)
	if err != nil {
		return fmt.Errorf("fixture %s arguments: %w", row.Name.Text, err)
	}
	switch row.Expected.(type) {
	case *syntax.SuccessBody, *syntax.FailureBody:
	default:
		return fmt.Errorf("fixture requires an explicit completion")
	}
	savedContext, savedResult, savedEscapes := c.context, c.region.Result, c.region.Escapes
	bound, err := c.context.Registry.Bound(step.Errors)
	if err != nil {
		return err
	}
	c.context.Result = step.Result
	c.context.Errors = bound
	// Fixture expectations admit bare ok under the same opaque/callable
	// confinement as assertion expectations: supplied boundaries can
	// return tokens no Can expression can construct.
	c.context.BareOpaque = true
	c.region.Result = step.Result
	expected, err := c.completion(row.Expected, scope)
	c.context, c.region.Result, c.region.Escapes = savedContext, savedResult, savedEscapes
	if err != nil {
		return fmt.Errorf("fixture %s completion: %w", row.Name.Text, err)
	}
	var raw *ir.RawFixture
	if row.Mode != nil {
		if row.Mode.Failure != nil {
			return fmt.Errorf("fixture %s: using failure is confined to attached wrapper assertions", row.Name.Text)
		}
		if success, ok := row.Expected.(*syntax.SuccessBody); ok && success.Value == nil {
			if kind := step.Result.Kind(); kind == types.Opaque || kind == types.Callable {
				return fmt.Errorf("fixture %s: using raw cannot drive a supplied opaque mock", row.Name.Text)
			}
		}
		if c.context.Raw == nil {
			return fmt.Errorf("fixture %s: using raw cannot resolve its target", row.Name.Text)
		}
		native, ok := c.context.Raw.Natives[step.Identity]
		if !ok || native.Symbol.Kind != resolve.Fetch && native.Symbol.Kind != resolve.Judge && native.Symbol.Kind != resolve.LLM && native.Symbol.Kind != resolve.Wrapper {
			return fmt.Errorf("fixture %s: using raw is allowed on fetch/judge/LLM/wrapper targets only", row.Name.Text)
		}
		fixture, err := c.context.Raw.Load(row.Mode.Raw.Value)
		if err != nil {
			return fmt.Errorf("fixture %s: %w", row.Name.Text, err)
		}
		if err := c.context.Raw.CheckRawFixture(step.Identity, fixture); err != nil {
			return fmt.Errorf("fixture %s: %w", row.Name.Text, err)
		}
		// The wrapper target names the fixture, but the exchange
		// substitutes at the root operation transport boundary.
		operation := step.Identity
		if native.Symbol.Kind == resolve.Wrapper && native.Wrapper != nil {
			operation = native.Wrapper.Root
		}
		raw = convertRawFixture(operation, fixture)
	}
	table.Rows = append(table.Rows, ir.FixtureRow{Selector: row.Name.Text, Owner: c.context.Package, Scenario: scenario, Prepare: prepared, Arguments: args, Expected: expected, Raw: raw})
	return nil
}

// Native assertions attach real roots to fetch/judge/LLM declarations. Rows
// invoke the declaration through the ordinary call checker, so inputs and
// grouped state bind exactly as consumer calls do; questions and arms keep
// their judge-owned coverage and declare no attached rows.
func (c *programChecker) nativeAssertions(program *Program, callables map[string]CallableDeclaration) error {
	for _, native := range program.Natives {
		var rows []syntax.Assertion
		switch d := native.Symbol.Declaration.(type) {
		case *syntax.FetchDecl:
			rows = d.Assertions
		case *syntax.LLMDecl:
			rows = d.Assertions
		case *syntax.JudgeDecl:
			rows = d.Assertions
		case *syntax.WrapDecl:
			rows = d.Assertions
		default:
			continue
		}
		if len(rows) == 0 {
			return fmt.Errorf("%s requires mandatory assertions", native.Symbol.Name)
		}
		seen := map[string]bool{}
		for _, row := range rows {
			if row.Name.Text == "" || seen[row.Name.Text] {
				return fmt.Errorf("duplicate or missing assertion name in %s", native.Symbol.Name)
			}
			seen[row.Name.Text] = true
		}
		file := c.world.Files[native.Symbol.Source]
		context, err := c.nativeContext(program, native, callables)
		if err != nil {
			return err
		}
		covered := false
		selected := map[WrapperKey]bool{}
		wrapped := native.Symbol.Kind == resolve.Wrapper
		for _, row := range rows {
			root := ir.AssertionRoot{Package: native.Symbol.Package.ID, Declaration: native.Symbol.ID, Name: row.Name.Text}
			if row.Use != nil {
				return fmt.Errorf("assertion %s: fixture templates expand at lexical when tables only", row.Name.Text)
			}
			if row.Scenario != nil {
				return fmt.Errorf("assertion %s: scenario rows belong to when tables, not assertion roots", row.Name.Text)
			}
			links, err := scenarioLinks(file, row)
			if err != nil {
				return err
			}
			root.Links = links
			if row.Receiver != nil {
				return fmt.Errorf("assertion %s: native assertions cannot supply a receiver", row.Name.Text)
			}
			contract := native.Signature
			if contract != nil && contract.Kind() == types.Callable {
				if success, ok := row.Expected.(*syntax.SuccessBody); ok && success.Value != nil {
					if kind := contract.Result().Kind(); kind == types.Opaque || kind == types.Callable {
						return fmt.Errorf("assertion %s: excluded opaque results use bare ok", row.Name.Text)
					}
				}
			}
			expanded := row.Arguments
			if contract != nil && contract.Kind() == types.Callable {
				inputs := contract.Inputs()
				expanded, err = expandScopeArguments(len(inputs), func(i int) bool {
					return isScopeRequest(inputs[i])
				}, c.variadic[native.Symbol.ID], row.Arguments, row.Span)
				if err != nil {
					return fmt.Errorf("assertion %s arguments: %w", row.Name.Text, err)
				}
			}
			var callee syntax.Expr = &syntax.NameExpr{Name: syntax.QualifiedName{Name: native.Symbol.Name}, ExpressionLocation: syntax.ExpressionLocation{Span: row.Span}}
			call := &syntax.CallExpr{ExpressionLocation: syntax.ExpressionLocation{Span: row.Span}, Invocation: syntax.Invocation{Span: row.Span, Callee: callee, Arguments: expanded}}
			actualContext := context
			actualContext.Identity = native.Symbol.ID + "/assert/" + row.Name.Text + "/actual"
			actualContext.Sites = nil
			actualContext.Scope = file.Scope
			actualContext.Parameters = nil
			actualContext.Expressions = c.expressions(file, file.Scope)
			actual, err := CheckRegion(actualContext, syntax.Block{Span: row.Span, Terminal: &syntax.RelayBody{BodyLocation: syntax.BodyLocation{Span: row.Span}, Call: call}})
			if err != nil {
				return fmt.Errorf("assertion %s arguments: %w", row.Name.Text, err)
			}
			switch row.Expected.(type) {
			case *syntax.SuccessBody, *syntax.FailureBody:
			default:
				return fmt.Errorf("assertion requires an explicit expected completion")
			}
			expectedContext := actualContext
			expectedContext.Identity = native.Symbol.ID + "/assert/" + row.Name.Text + "/expected"
			expectedContext.BareOpaque = true
			expected, err := CheckRegion(expectedContext, syntax.Block{Span: row.Span, Terminal: row.Expected})
			if err != nil {
				return fmt.Errorf("assertion %s expected completion: %w", row.Name.Text, err)
			}
			var raw *ir.RawFixture
			var injected *ir.PolicyInjection
			if row.Mode != nil {
				if row.Mode.Failure != nil {
					if !wrapped {
						return fmt.Errorf("assertion %s: using failure is confined to attached wrapper assertions", row.Name.Text)
					}
					key, value, err := c.policyInjection(file, native, row)
					if err != nil {
						return fmt.Errorf("assertion %s: %w", row.Name.Text, err)
					}
					if native.Wrapper == nil {
						return fmt.Errorf("assertion %s: wrapper %s has no resolved policy", row.Name.Text, native.Symbol.Name)
					}
					value.Failed = native.Wrapper.Failed
					injected = value
					selected[key] = true
				} else {
					if context.Raw == nil {
						return fmt.Errorf("assertion %s: using raw cannot resolve its target", row.Name.Text)
					}
					fixture, err := context.Raw.Load(row.Mode.Raw.Value)
					if err != nil {
						return fmt.Errorf("assertion %s: %w", row.Name.Text, err)
					}
					if err := context.Raw.CheckRawFixture(native.Symbol.ID, fixture); err != nil {
						return fmt.Errorf("assertion %s: %w", row.Name.Text, err)
					}
					operation := native.Symbol.ID
					if wrapped && native.Wrapper != nil {
						operation = native.Wrapper.Root
					}
					raw = convertRawFixture(operation, fixture)
					if raw.Exchange != nil && raw.Exchange.Response != nil {
						covered = true
					}
					// A raw failure outcome runs the underlying operation
					// through normalization into exactly one native key.
					if wrapped && raw.Exchange != nil && raw.Exchange.Failure != nil {
						if leaf, ok := rawFailureLeaf(raw.Exchange.Failure.Kind); ok && native.Wrapper != nil {
							for _, local := range native.Wrapper.Local {
								if local.Origin == WrapperNative && local.Decl == leaf {
									selected[local] = true
								}
							}
						}
					}
				}
			}
			program.Assertions = append(program.Assertions, &ir.Assertion{Root: root, Actual: actual, Expected: expected, Raw: raw, Injected: injected})
		}
		if wrapped {
			// Wrapper transport evidence lives on the wrapped operation;
			// wrappers prove each locally declared key instead.
			if native.Wrapper == nil {
				return fmt.Errorf("wrapper %s has no resolved policy", native.Symbol.Name)
			}
			for _, key := range native.Wrapper.Local {
				if !selected[key] {
					return fmt.Errorf("wrapper %s lacks an attached assertion selecting local %s key %s", native.Symbol.Name, key.Origin, key.Identity)
				}
			}
			continue
		}
		if !covered {
			return fmt.Errorf("native declaration %s lacks request/decoder coverage: attach a raw case comparing its prepared request against a response", native.Symbol.Name)
		}
	}
	return nil
}

// rawFailureLeaf maps a fixture failure outcome to the exact native leaf it
// normalizes into at the producing boundary.
func rawFailureLeaf(kind string) (string, bool) {
	switch kind {
	case "transport":
		return "can.std.http@1::transport_failed", true
	case "timeout":
		return "can.std.http@1::timeout", true
	case "body_limit":
		return "can.std.http@1::body_limit", true
	default:
		return "", false
	}
}

func validateAssertionNames(declaration *syntax.FunctionDecl) error {
	if len(declaration.Assertions) == 0 {
		return fmt.Errorf("%s requires mandatory assertions", declaration.Name.Text)
	}
	seen := map[string]bool{}
	for _, row := range declaration.Assertions {
		if row.Name.Text == "" || seen[row.Name.Text] {
			return fmt.Errorf("duplicate or missing assertion name in %s", declaration.Name.Text)
		}
		seen[row.Name.Text] = true
	}
	return nil
}

// A generic assertion is its own application, independent of instances reached
// from executable bodies. Its expected success value can constrain the result.
func (c *programChecker) genericAssertions(files []*resolve.File) error {
	for _, file := range files {
		for _, node := range file.Source.Syntax.Declarations {
			d, ok := node.(*syntax.FunctionDecl)
			if !ok || len(d.Parameters) == 0 {
				continue
			}
			symbol := file.Package.Scope.Symbols[d.Name.Text]
			for _, row := range d.Assertions {
				elided := make([]bool, len(d.Inputs))
				for i, input := range d.Inputs {
					elided[i] = !input.Variadic && c.scopeElided(file, input.Type)
				}
				expanded, err := expandScopeArguments(len(d.Inputs), func(i int) bool { return elided[i] }, len(d.Inputs) > 0 && d.Inputs[len(d.Inputs)-1].Variadic, row.Arguments, row.Span)
				if err != nil {
					return fmt.Errorf("generic assertion %s: %w", row.Name.Text, err)
				}
				constraints, err := genericArguments(d.Inputs, expanded)
				if err != nil {
					return fmt.Errorf("generic assertion %s: %w", row.Name.Text, err)
				}
				if d.Receiver != nil {
					if row.Receiver == nil {
						if !c.scopeElided(file, d.Receiver.Type) {
							return fmt.Errorf("method assertion requires a receiver")
						}
						constraints = append(constraints, argumentConstraint{d.Receiver.Type, &syntax.ScopeExpr{ExpressionLocation: syntax.ExpressionLocation{Span: row.Span}}})
					} else {
						if c.scopeElided(file, d.Receiver.Type) {
							return fmt.Errorf("generic assertion %s: ingress scope receivers are harness-supplied", row.Name.Text)
						}
						constraints = append(constraints, argumentConstraint{d.Receiver.Type, row.Receiver})
					}
				}
				if success, ok := row.Expected.(*syntax.SuccessBody); ok && success.Value != nil {
					if c.scopeElided(file, d.Result) {
						return fmt.Errorf("generic assertion %s: ingress scope results cannot be named in expected completions", row.Name.Text)
					}
					constraints = append(constraints, argumentConstraint{d.Result, success.Value})
				}
				provisional := CompletionContext{Sites: indexLexicalSites(symbol.ID, d), Identity: symbol.ID + "/assert/inference", Package: file.Package.ID, Scope: file.Scope, Expressions: c.expressions(file, file.Scope), Callables: c.callables, Variadic: c.variadic}

				provisional.IntrinsicIdentity = func(scope *resolve.Scope, name syntax.QualifiedName) string {
					symbol, err := file.Lookup(scope, name, resolve.CallUse)
					if err != nil {
						return ""
					}
					return symbol.ID
				}
				provisional.CatalogueType = c.catalogueType
				provisional.InferCallback = func(scope *resolve.Scope, name syntax.QualifiedName, inputs []*types.Type, result *types.Type, e *Expressions) (ValueBinding, bool, error) {
					return c.inferCallback(file, scope, name, inputs, result, e)
				}
				provisional.Type = func(node syntax.TypeNode, allowVoid bool) (*types.Type, error) {
					return c.annotation(file, node, allowVoid)
				}
				provisional.Registry = c.program.Registry
				provisional.File = file.Source.Syntax.Source
				provisional.Specialize = func(scope *resolve.Scope, name syntax.QualifiedName, args []syntax.TypeNode) (ValueBinding, error) {
					return c.specialize(file, scope, name, args)
				}
				provisional.AggregateType = func(variant *types.Type) (*types.Type, error) { return c.aggregateType(file, variant) }
				provisional.ResolveMethod = func(application MethodApplication) (ValueBinding, error) { return c.method(file, application) }
				provisional.InferReference = func(scope *resolve.Scope, name syntax.QualifiedName, expected *types.Type, e *Expressions) (ValueBinding, bool, error) {
					return c.inferReference(file, scope, name, expected, e)
				}
				provisional.InferCall = func(scope *resolve.Scope, name syntax.QualifiedName, args []syntax.Argument, expected *types.Type, e *Expressions) (ValueBinding, bool, error) {
					return c.inferCall(file, scope, name, args, expected, e)
				}
				checker := &regionChecker{context: provisional, region: &ir.Region{ID: provisional.Identity}, locals: map[string]*types.Type{}, uses: LocalUses{Names: map[*syntax.NameExpr]string{}, Captures: map[*syntax.ReferenceExpr][]string{}}}
				arguments, err := c.inferArguments(file, symbol.Parameters, d.Result, nil, constraints, checker.expressions(bodyScope{file.Scope}))
				if err != nil {
					return fmt.Errorf("generic assertion %s in %s: %w", row.Name.Text, symbol.ID, err)
				}
				binding, err := c.instantiateFunction(symbol, d, arguments, applicationSite(file, row.Span.Start))
				if err != nil {
					return err
				}
				fn := c.instances[binding.Identity]
				context, err := c.functionContext(fn)
				if err != nil {
					return err
				}
				rows, err := c.assertionRows(file, fn, context, []syntax.Assertion{row})
				if err != nil {
					return err
				}
				c.program.Assertions = append(c.program.Assertions, rows...)
			}
		}
	}
	return nil
}
