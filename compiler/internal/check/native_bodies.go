package check

import (
	"fmt"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
	"strings"
)

func (c *programChecker) nativeContext(program *Program, native *NativeDeclaration, callables map[string]CallableDeclaration) (CompletionContext, error) {
	file := c.world.Files[native.Symbol.Source]
	scope := c.world.NativeScopes[native.Symbol.Declaration]
	if scope == nil {
		scope = resolve.NewScope(file.Scope)
	}
	bound, err := program.Registry.Bound(native.Signature.Errors())
	if err != nil {
		return CompletionContext{}, err
	}
	ctx := CompletionContext{Sites: indexLexicalSites(native.Symbol.ID, native.Symbol.Declaration), Identity: native.Symbol.ID, Kind: ir.FunctionRegion, File: file.Source.Syntax.Source, Scope: scope, Result: native.Signature.Result(), Errors: bound, Registry: program.Registry, Expressions: c.expressions(file, scope), Callables: callables, Variadic: c.variadic}
	if question, ok := native.Symbol.Declaration.(*syntax.QuestionDecl); ok {
		for _, binder := range question.Binders {
			id := native.Symbol.ID + "/metadata/" + binder.Name.Text
			ctx.Parameters = append(ctx.Parameters, ir.Local{Identity: id, Type: c.bindings[id]})
		}
	}

	ctx.IntrinsicIdentity = func(scope *resolve.Scope, name syntax.QualifiedName) string {
		symbol, err := file.Lookup(scope, name, resolve.CallUse)
		if err != nil {
			return ""
		}
		return symbol.ID
	}
	ctx.CatalogueType = c.catalogueType
	ctx.InferCallback = func(scope *resolve.Scope, name syntax.QualifiedName, inputs []*types.Type, result *types.Type, e *Expressions) (ValueBinding, bool, error) {
		return c.inferCallback(file, scope, name, inputs, result, e)
	}
	ctx.Type = func(node syntax.TypeNode, allowVoid bool) (*types.Type, error) {
		return c.annotation(file, node, allowVoid)
	}
	ctx.ErrorName = func(name syntax.QualifiedName) (string, error) {
		symbol, err := file.Lookup(nil, name, resolve.ErrorUse)
		if err != nil {
			return "", err
		}
		return symbol.ID, nil
	}
	ctx.PatternName = func(name syntax.QualifiedName) (string, error) {
		symbol, err := file.Lookup(nil, name, resolve.TypeUse)
		if err != nil {
			return "", err
		}
		return symbol.ID, nil
	}
	ctx.Specialize = func(scope *resolve.Scope, name syntax.QualifiedName, args []syntax.TypeNode) (ValueBinding, error) {
		return c.specialize(file, scope, name, args)
	}
	ctx.AggregateType = func(variant *types.Type) (*types.Type, error) { return c.aggregateType(file, variant) }
	ctx.ResolveMethod = func(application MethodApplication) (ValueBinding, error) { return c.method(file, application) }
	ctx.InferReference = func(scope *resolve.Scope, name syntax.QualifiedName, expected *types.Type, e *Expressions) (ValueBinding, bool, error) {
		return c.inferReference(file, scope, name, expected, e)
	}
	ctx.InferCall = func(scope *resolve.Scope, name syntax.QualifiedName, args []syntax.Argument, expected *types.Type, e *Expressions) (ValueBinding, bool, error) {
		return c.inferCall(file, scope, name, args, expected, e)
	}
	for _, name := range native.Descriptor.Names {
		id := native.Symbol.ID + "/input/" + name
		ctx.Parameters = append(ctx.Parameters, ir.Local{Identity: id, Type: c.bindings[id]})
	}
	return ctx, nil
}
func expressionRegion(ctx CompletionContext) *regionChecker {
	return &regionChecker{context: ctx, region: &ir.Region{ID: ctx.Identity}, locals: map[string]*types.Type{}, uses: LocalUses{Names: map[*syntax.NameExpr]string{}, Captures: map[*syntax.ReferenceExpr][]string{}}}
}
func (c *programChecker) checkNativeBodies(program *Program, callables map[string]CallableDeclaration) error {
	for _, native := range program.Natives {
		ctx, err := c.nativeContext(program, native, callables)
		if err != nil {
			return err
		}
		file := c.world.Files[native.Symbol.Source]
		// Put the phase restriction in the context inherited by nested calls,
		// matches and near captures, not only on the outer expression checker.
		preparation := ctx
		preparationExpressions := *ctx.Expressions
		preparation.Expressions = &preparationExpressions
		valueLookup := preparationExpressions.Value
		preparationExpressions.Value = func(name syntax.QualifiedName) (ValueBinding, error) {
			binding, err := valueLookup(name)
			if err == nil && strings.HasPrefix(binding.Identity, native.Symbol.ID+"/metadata/") {
				return ValueBinding{}, fmt.Errorf("answer metadata is unavailable during descriptor preparation")
			}
			return binding, err
		}
		checker := expressionRegion(preparation)
		expressions := checker.expressions(bodyScope{preparation.Scope})
		checkedDescriptors := map[syntax.Expr]*ir.Expression{}
		scalarExpression := func(expression syntax.Expr, kind string) error {
			value, err := expressions.Check(expression, c.annotations[file][kind])
			if err == nil {
				checkedDescriptors[expression] = value
			}
			return err
		}
		handler := func(body syntax.Body, result *types.Type, probability bool) error {
			h := ctx
			h.Kind = ir.HandlerRegion
			h.Parent = ctx.Identity
			h.Identity = fmt.Sprintf("%s/handler/%d", ctx.Identity, len(native.Regions))
			h.Result = result
			e := *ctx.Expressions
			h.Expressions = &e
			if probability {
				id := h.Identity + "/probability"
				e.Probability = &ValueBinding{Identity: id, Type: c.annotations[file]["float"]}
				h.Parameters = append(append([]ir.Local(nil), h.Parameters...), ir.Local{Identity: id, Type: e.Probability.Type})
			}
			region, err := CheckRegion(h, syntax.Block{Span: body.BodySpan(), Terminal: body})
			if err != nil {
				return err
			}
			native.Regions = append(native.Regions, region)
			return nil
		}
		switch d := native.Symbol.Declaration.(type) {
		case *syntax.JudgeDecl:
			judgeInputs := append([]ir.Local(nil), ctx.Parameters...)
			var pending []struct {
				field syntax.Field
				typ   *types.Type
			}
			for _, registration := range d.Registrations {
				call := registration.Call
				name, ok := call.Invocation.Callee.(*syntax.NameExpr)
				if !ok || len(call.Methods) != 0 || len(call.Invocation.Types) != 0 {
					return fmt.Errorf("judge registration requires a named concrete question")
				}
				question, e := file.Lookup(ctx.Scope, name.Name, resolve.QuestionUse)
				if e != nil {
					return e
				}
				var target *NativeDeclaration
				for _, candidate := range program.Natives {
					if candidate.Symbol == question {
						target = candidate
						break
					}
				}
				if target == nil || target.Connection != native.Connection {
					return fmt.Errorf("judge question must use the identical connection declaration")
				}
				questionBound, e := program.Registry.Bound(target.Signature.Errors())
				if e != nil {
					return e
				}
				if e = ctx.Errors.CheckEscaping(questionBound); e != nil {
					return e
				}
				prepare, args, e := checker.arguments(expressions, ValueBinding{Identity: question.ID, Type: target.Signature}, call.Invocation.Arguments, nil)
				if e != nil {
					return e
				}
				checked := ir.JudgeRegistration{Question: question.ID, Span: registration.Span, Prepare: prepare, Arguments: args}
				if target.Signature.Result().Kind() == types.Void {
					if registration.Binding != nil {
						return fmt.Errorf("void question registration cannot bind a value")
					}
				} else {
					if registration.Binding == nil {
						return fmt.Errorf("nonvoid question registration requires as type name")
					}
					typ, e := c.annotation(file, registration.Binding.Type, false)
					if e != nil {
						return e
					}
					if !types.Equal(typ, target.Signature.Result()) {
						return fmt.Errorf("question registration binding type mismatch")
					}
					pending = append(pending, struct {
						field syntax.Field
						typ   *types.Type
					}{*registration.Binding, typ})
					checked.Binding = &ir.Local{Identity: ctx.Identity + "/answer/" + registration.Binding.Name.Text, Type: typ}
				}
				native.Registrations = append(native.Registrations, checked)
			}
			// No answer binding exists while any registration arguments are checked.
			continuationScope := resolve.NewScope(ctx.Scope)
			for _, answer := range pending {
				id := ctx.Identity + "/answer/" + answer.field.Name.Text
				if err = continuationScope.Define(&resolve.Symbol{Name: answer.field.Name.Text, ID: id, Kind: resolve.Value, Type: answer.field.Type}); err != nil {
					return err
				}
				c.bindings[id] = answer.typ
				ctx.Parameters = append(ctx.Parameters, ir.Local{Identity: id, Type: answer.typ})
			}
			ctx.Scope = continuationScope
			ctx.Expressions = c.expressions(file, continuationScope)
			if err = handler(d.Continuation, ctx.Result, false); err != nil {
				return err
			}
			state, stateInputs, e := judgeState(native, judgeInputs)
			if e != nil {
				return e
			}
			native.Judge = &ir.Judge{Identity: native.Symbol.ID, Source: file.Source.ID, Connection: native.Connection, Span: d.Span, Inputs: judgeInputs, State: state, StateInputs: stateInputs, Registrations: native.Registrations, Continuation: native.Regions[0]}
		case *syntax.LLMDecl:
			if err = scalarExpression(d.Asks, "str"); err != nil {
				return err
			}
		case *syntax.FetchDecl:
			plan := &ir.Fetch{Identity: native.Symbol.ID, Source: file.Source.ID, Connection: native.Connection, Span: d.Span, Inputs: append([]ir.Local(nil), ctx.Parameters...), Result: native.Signature.Result(), Method: strings.ToUpper(d.Method.Text)}
			if err = checkFetchContentType(d, program.Connections[native.Connection]); err != nil {
				return err
			}
			if err = scalarExpression(d.Path, "str"); err != nil {
				return err
			}
			plan.Path = checkedDescriptors[d.Path]
			for sectionIndex, section := range [][]syntax.NativeEntry{d.Query, d.Headers} {
				seen := map[string]bool{}
				for _, entry := range section {
					if sectionIndex == 1 {
						name := strings.ToLower(strings.ReplaceAll(entry.Name.Text, "_", "-"))
						if !connectionHeaderName(entry.Name.Text) || seen[name] || name == "authorization" && program.Connections[native.Connection].BearerEnvironment != "" {
							return fmt.Errorf("invalid, conflicting or duplicate request header %s", name)
						}
						seen[name] = true
						if literal, ok := entry.Value.(*syntax.LiteralExpr); ok && literal.Token.Kind == syntax.String {
							for _, r := range literal.Token.Value {
								if r > 255 || r == 0 || r == '\r' || r == '\n' {
									return fmt.Errorf("invalid literal request header value")
								}
							}
						}
					}
					var expected *types.Type
					if _, ok := entry.Value.(*syntax.ArrayExpr); ok {
						expected = c.annotations[file]["str[]"]
					}
					value, e := expressions.Check(entry.Value, expected)
					if e != nil {
						return e
					}
					entryPlan := ir.FetchEntry{Name: entry.Name.Text, Value: value}
					if sectionIndex == 0 {
						plan.Query = append(plan.Query, entryPlan)
					} else {
						plan.Headers = append(plan.Headers, entryPlan)
					}
					if !scalar(value.Type, "str") && !(value.Type.Kind() == types.Array && scalar(value.Type.Element(), "str")) {
						return fmt.Errorf("query/header value requires str or str[]")
					}
				}
			}
			if d.Body != nil {
				body, e := expressions.Check(d.Body, nil)
				if e != nil {
					return e
				}
				plan.Body = body
				plan.BodyMode = d.BodyEncoding.Text
				switch d.BodyEncoding.Text {
				case "text":
					if !scalar(body.Type, "str") {
						return fmt.Errorf("text body requires str")
					}
				case "bytes":
					if body.Type.Declaration() != "can.std.bytes@1::buffer" {
						return fmt.Errorf("bytes body requires bytes::buffer")
					}
				case "json":
					if _, e = types.Schema(body.Type); e != nil {
						return e
					}
				}
			}
			if err = finishFetchPlan(plan); err != nil {
				return err
			}
			native.Fetch = plan
		case *syntax.QuestionDecl:
			if err = scalarExpression(d.Asks, "str"); err != nil {
				return err
			}
			if d.Minimum != nil {
				if err = scalarExpression(d.Minimum, "float"); err != nil {
					return err
				}
			}
			result, err := c.annotation(file, d.Result, true)
			if err != nil {
				return err
			}
			plan := &ir.Question{Identity: native.Symbol.ID, Source: file.Source.ID, Connection: native.Connection, Kind: d.Kind, Span: d.Span, Result: native.Signature.Result(), Record: d.RecordName != nil, Instructions: checkedDescriptors[d.Asks], Minimum: checkedDescriptors[d.Minimum]}
			plan.Inputs = append([]ir.Local(nil), ctx.Parameters[len(d.Binders):]...)
			if d.Kind == "noul" && plan.Minimum == nil {
				plan.Minimum = &ir.Expression{Kind: ir.Literal, Span: d.Span, Type: c.annotations[file]["float"], Text: "0.5"}
			}
			names := map[string]bool{}
			for i, binder := range d.Binders {
				plan.Metadata = append(plan.Metadata, ir.QuestionMetadata{Name: binder.Name.Text, Kind: binder.Kind.Text, Local: ctx.Parameters[i]})
				names[binder.Name.Text] = true
			}
			for _, option := range d.Options {
				preparedOption := ir.QuestionOption{Dynamic: d.Selected != nil}
				if option.Name != nil {
					preparedOption.Name = option.Name.Text
				}
				if option.Name != nil {
					if names[option.Name.Text] {
						return fmt.Errorf("duplicate option/metadata name %s", option.Name.Text)
					}
					names[option.Name.Text] = true
				}
				if option.Spread != nil {
					spread, e := expressions.Check(option.Spread, nil)
					if e != nil {
						return e
					}
					preparedOption.Spread = spread
					if d.Selected != nil {
						if spread.Type.Kind() != types.Array || spread.Type.Element().Declaration() != "can.prelude@1::choice_option" {
							return fmt.Errorf("dynamic Choice spread requires choice_option[]")
						}
					} else {
						if spread.Type.Kind() != types.Record {
							return fmt.Errorf("static Choice spread requires an ordinary arm record")
						}
						for _, field := range spread.Type.Fields() {
							if names[field.Name] {
								return fmt.Errorf("duplicate expanded option %s", field.Name)
							}
							names[field.Name] = true
							if field.Type.Kind() != types.ChoiceArm || !types.Equal(field.Type.Result(), result) {
								return fmt.Errorf("spread field requires a compatible choice_arm")
							}
							bound, e := program.Registry.Bound(field.Type.Errors())
							if e != nil {
								return e
							}
							if e = ctx.Errors.CheckEscaping(bound); e != nil {
								return e
							}
						}
					}
				}
				if option.Description != nil {
					if err = scalarExpression(option.Description, "str"); err != nil {
						return err
					}
				}
				preparedOption.Description = checkedDescriptors[option.Description]
				if option.Body != nil {
					if err = handler(option.Body, result, true); err != nil {
						return err
					}
					preparedOption.Handler = native.Regions[len(native.Regions)-1]
				}
				plan.Options = append(plan.Options, preparedOption)
			}
			if d.Fallback != nil {
				if err = handler(d.Fallback, result, false); err != nil {
					return err
				}
				plan.Fallback = native.Regions[len(native.Regions)-1]
			}
			if d.Selected != nil {
				typ, e := c.annotation(file, d.Selected.Type, false)
				if e != nil {
					return e
				}
				if !scalar(typ, "str") {
					return fmt.Errorf("dynamic Choice selected key must be str")
				}
				if ctx.Scope.Symbols[d.Selected.Name.Text] != nil {
					return fmt.Errorf("selected-key binder duplicates native input or metadata")
				}
				selectedScope := resolve.NewScope(ctx.Scope)
				id := ctx.Identity + "/selected/" + d.Selected.Name.Text
				if e = selectedScope.Define(&resolve.Symbol{Name: d.Selected.Name.Text, ID: id, Kind: resolve.Value, Type: d.Selected.Type}); e != nil {
					return e
				}
				c.bindings[id] = typ
				ctx.Scope = selectedScope
				ctx.Expressions = c.expressions(file, selectedScope)
				ctx.Parameters = append(ctx.Parameters, ir.Local{Identity: id, Type: typ})
				if err = handler(d.Shared, result, true); err != nil {
					return err
				}
				plan.Shared = native.Regions[len(native.Regions)-1]
			}
			if d.Shared != nil && d.Selected == nil {
				if err = handler(d.Shared, result, false); err != nil {
					return err
				}
				plan.Shared = native.Regions[len(native.Regions)-1]
			}
			native.Question = plan
		case *syntax.ChoiceArmDecl:
			if err = inertExpression(d.Description); err != nil {
				return fmt.Errorf("choice arm description must be a constant expression: %w", err)
			}
			if literal, ok := d.Description.(*syntax.LiteralExpr); ok && literal.Token.Kind == syntax.String && literal.Token.Value == "" {
				return fmt.Errorf("choice arm description must be nonempty")
			}
			if err = scalarExpression(d.Description, "str"); err != nil {
				return err
			}
			native.ArmDescription = checkedDescriptors[d.Description]
			ctx.Expressions.Probability = &ValueBinding{Identity: ctx.Identity + "/probability", Type: c.annotations[file]["float"]}
			ctx.Parameters = append(ctx.Parameters, ir.Local{Identity: ctx.Expressions.Probability.Identity, Type: ctx.Expressions.Probability.Type})
			region, e := CheckRegion(ctx, d.Body)
			if e != nil {
				return e
			}
			native.Regions = append(native.Regions, region)
		}
	}
	return nil
}
