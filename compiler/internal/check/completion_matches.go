package check

import (
	"fmt"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

func (c *regionChecker) match(n syntax.Match, scope bodyScope, valueType *types.Type) (*ir.Match, error) {
	if len(n.Arms) == 0 {
		return nil, fmt.Errorf("match requires arms")
	}
	out := &ir.Match{Span: n.Span, ValueResult: valueType}
	if n.Kind == syntax.ValueMatch {
		return c.valueMatch(n, scope, valueType)
	}
	if valueType != nil {
		return nil, fmt.Errorf("call/chain match cannot initialize a value")
	}
	successScope := c.child(scope)
	var err error
	switch n.Kind {
	case syntax.CallMatch:
		out.Call, err = c.invocation(n.Call, scope)
	case syntax.ChainMatch:
		if len(n.Chain) == 0 {
			return nil, fmt.Errorf("chain requires calls")
		}
		void, err := c.context.Type(&syntax.NamedType{Name: syntax.QualifiedName{Name: "void"}}, true)
		if err != nil {
			return nil, err
		}
		out.Call = &ir.Invocation{Span: n.Span, Result: void}
		for _, entry := range n.Chain {
			call, err := c.invocation(entry.Call, successScope)
			if err != nil {
				return nil, err
			}
			if entry.Binding != nil {
				typ, err := c.context.Type(entry.Binding.Type, false)
				if err != nil {
					return nil, err
				}
				if !types.Assignable(call.Result, typ) {
					return nil, fmt.Errorf("chain binding type mismatch")
				}
				local, err := c.bind(successScope, entry.Binding.Name.Text, typ)
				if err != nil {
					return nil, err
				}
				call.Steps[len(call.Steps)-1].SuccessBinding = local.Identity
			} else if call.Result.Kind() != types.Void {
				return nil, fmt.Errorf("nonvoid chain result requires a binding")
			}
			out.Call.Steps = append(out.Call.Steps, call.Steps...)
			out.Call.Errors = unionErrors(out.Call.Errors, call.Errors)
		}
	default:
		return nil, fmt.Errorf("unknown match kind")
	}
	if err != nil {
		return nil, err
	}
	if len(n.When) != 0 {
		if n.Kind != syntax.CallMatch || len(out.Call.Steps) != 1 || out.Call.Steps[0].Contract == nil {
			return nil, fmt.Errorf("fixture table requires one resolved invocation; chain fixture scheduling belongs to the coordination harness")
		}
		table, err := c.fixtures(n.When, &out.Call.Steps[0], scope)
		if err != nil {
			return nil, err
		}
		out.Call.Steps[0].Fixtures = table
	}
	out.Arms, err = c.completionArms(n.Arms, out.Call, out.Call.Result, out.Call.Errors, scope, successScope, true)
	return out, err
}

func (c *regionChecker) completionArms(arms []syntax.MatchArm, call *ir.Invocation, result *types.Type, errors []*types.Type, scope, successScope bodyScope, requireSuccess bool) ([]ir.Arm, error) {
	bound, err := c.context.Registry.Bound(errors)
	if err != nil {
		return nil, err
	}
	var checked []ir.Arm
	seen := map[string]bool{}
	for _, arm := range arms {
		if arm.Outcome == nil || len(arm.Patterns) != 0 {
			return nil, fmt.Errorf("completion match requires exact outcome patterns")
		}
		a := ir.Arm{Span: arm.Span, Forward: arm.Forward}
		pattern := arm.Outcome
		armScope := c.child(scope)
		var bindingType *types.Type
		var bindingName string
		var key string
		switch {
		case pattern.Success:
			if pattern.StandardFailure || pattern.Error != nil {
				return nil, fmt.Errorf("invalid success pattern")
			}
			a.Outcome = "ok"
			key = "ok"
			bindingType = result
			armScope = c.child(successScope)
		case pattern.StandardFailure:
			if pattern.Error != nil || arm.Forward {
				return nil, fmt.Errorf("invalid standard pattern")
			}
			a.Outcome = "standard"
			key = "standard"
			// C9.1: a bound standard catch binds the opaque snapshot, never
			// a string. The former canonical text is failure.message.
			snapshot, err := c.context.Type(named("standard_failure"), false)
			if err != nil {
				return nil, c.locate(pattern.Span, err)
			}
			bindingType = snapshot
		default:
			name, ok := pattern.Error.(*syntax.NamedType)
			if !ok {
				return nil, c.locate(pattern.Span, fmt.Errorf("domain pattern must be an error head"))
			}
			var concrete ConcreteError
			if len(name.Arguments) == 0 {
				var declaration string
				if c.context.ErrorName != nil {
					declaration, err = c.context.ErrorName(name.Name)
				} else {
					var typ *types.Type
					typ, err = c.context.Type(name, false)
					if err == nil {
						declaration = typ.Declaration()
					}
				}
				if err != nil {
					return nil, c.locate(pattern.Span, err)
				}
				concrete, err = bound.ResolveBareArm(declaration)
				if err != nil {
					return nil, c.locate(pattern.Span, err)
				}
			} else {
				resolved, resolveErr := c.context.Type(name, false)
				if resolveErr != nil {
					return nil, c.locate(pattern.Span, resolveErr)
				}
				if _, concreteErr := c.context.Registry.Concrete(resolved); concreteErr != nil {
					return nil, c.locate(pattern.Span, concreteErr)
				}
				concrete, err = bound.ResolveExactArm(resolved.Identity())
				if err != nil {
					return nil, c.locate(pattern.Span, err)
				}
			}
			a.Outcome = "domain"
			a.Error = concrete.Type
			key = "domain:" + concrete.TypeIdentity
			bindingType = concrete.Type
			// An explicit alias binds the chosen payload under its own name;
			// otherwise the existing short error-name alias is introduced.
			bindingName = name.Name.Name
			if pattern.Alias != nil {
				bindingName = pattern.Alias.Text
			}
		}
		if seen[key] {
			return nil, c.locate(pattern.Span, fmt.Errorf("duplicate completion arm for %s", key))
		}
		// C5.1 order: every error arm, with the optional standard arm
		// anywhere among those failures, precedes exactly one final ok.
		// Individual failure order is unconstrained; only success-last is
		// enforced, at the failure head that breaks it.
		if a.Outcome != "ok" && seen["ok"] {
			return nil, c.locate(pattern.Span, fmt.Errorf("failure arms precede the final ok"))
		}
		seen[key] = true
		if pattern.Binding != nil {
			if a.Outcome == "domain" {
				return nil, c.locate(pattern.Span, fmt.Errorf("domain arm binds its declared error name"))
			}
			declared, err := c.context.Type(pattern.Binding.Type, false)
			if err != nil {
				return nil, err
			}
			if !types.Equal(declared, bindingType) {
				if a.Outcome == "standard" {
					return nil, c.locate(pattern.Span, fmt.Errorf("standard catch binds the standard_failure snapshot; str and other binder types are rejected"))
				}
				return nil, fmt.Errorf("completion arm binding type mismatch")
			}
			bindingName = pattern.Binding.Name.Text
		}
		if bindingName != "" && !arm.Forward {
			if a.Outcome == "domain" && pattern.Alias == nil {
				a.Binding, err = c.bindErrorAlias(armScope, bindingName, bindingType)
			} else {
				a.Binding, err = c.bind(armScope, bindingName, bindingType)
			}
			if err != nil {
				return nil, c.locate(pattern.Span, err)
			}
		}
		if arm.Forward {
			if arm.Body != nil || pattern.Binding != nil || pattern.Alias != nil {
				return nil, fmt.Errorf("forwarding arm cannot contain a body or binding")
			}
			if a.Outcome == "ok" {
				if !types.Assignable(result, c.region.Result) {
					return nil, fmt.Errorf("forwarded success does not fit current region")
				}
			} else {
				actual, err := c.context.Registry.Bound([]*types.Type{a.Error})
				if err != nil {
					return nil, err
				}
				if err = c.escaping(actual); err != nil {
					return nil, err
				}
			}
		} else {
			a.Body, err = c.completion(arm.Body, armScope)
			if err != nil && !c.deferAggregate(err) {
				return nil, err
			}
		}
		checked = append(checked, a)
	}
	var matchSpan source.Span
	if len(arms) != 0 {
		matchSpan = arms[0].Span
	}
	if requireSuccess && !seen["ok"] {
		return nil, c.locate(matchSpan, fmt.Errorf("completion match requires exactly one success arm"))
	}
	for _, entry := range bound.Entries() {
		if !seen["domain:"+entry.TypeIdentity] {
			missing := entry.Declaration.Name
			if len(entry.Arguments) != 0 {
				missing = entry.TypeIdentity
			}
			message := fmt.Sprintf("missing completion arm for %s", missing)
			if note := c.wrapperNote(call, entry.TypeIdentity, missing); note != "" {
				message += note
			}
			return nil, c.locate(matchSpan, fmt.Errorf("%s", message))
		}
	}
	return checked, nil
}

func (c *regionChecker) valueMatch(n syntax.Match, scope bodyScope, valueType *types.Type) (*ir.Match, error) {
	if n.Call != nil || len(n.Chain) != 0 || len(n.Values) == 0 {
		return nil, fmt.Errorf("invalid ordinary match")
	}
	out := &ir.Match{Span: n.Span, ValueResult: valueType}
	var columns []*types.Type
	for _, node := range n.Values {
		x, err := c.expressions(scope).Check(node, nil)
		if err != nil {
			return nil, err
		}
		out.Values = append(out.Values, x)
		columns = append(columns, x.Type)
	}
	var previous [][]*ir.Pattern
	for _, arm := range n.Arms {
		if arm.Outcome != nil || arm.Forward || len(arm.Patterns) != len(n.Values) {
			return nil, fmt.Errorf("ordinary match requires one data pattern per scrutinee")
		}
		armScope := c.child(scope)
		a := ir.Arm{Span: arm.Span}
		bindings := map[string]*ir.Local{}
		for i, node := range arm.Patterns {
			pattern, err := c.pattern(node, columns[i], bindings)
			if err != nil {
				return nil, err
			}
			a.Patterns = append(a.Patterns, pattern)
		}
		useful, err := patternsUseful(previous, a.Patterns, columns)
		if err != nil {
			return nil, err
		}
		if !useful {
			return nil, fmt.Errorf("match arm is fully covered by earlier arms")
		}
		for name, local := range bindings {
			if err := c.install(armScope, name, local); err != nil {
				return nil, err
			}
		}
		// Repeated occurrences of the same resolved binding share one inferred
		// arm-local narrowing. Parentheses preserve that binding identity. A pair
		// of disjoint nominal requirements cannot both hold for one immutable value.
		type narrowing struct {
			typ   *types.Type
			local *ir.Local
		}
		narrowed := map[string]narrowing{}
		for i, node := range n.Values {
			for {
				group, ok := node.(*syntax.GroupExpr)
				if !ok {
					break
				}
				node = group.Value
			}
			name, ok := node.(*syntax.NameExpr)
			if !ok || name.Name.Package != "" {
				continue
			}
			narrow := patternNarrow(a.Patterns[i])
			if narrow == nil || types.Equal(narrow, columns[i]) {
				continue
			}
			resolved := out.Values[i]
			if resolved.Kind != ir.Binding || resolved.Text == "" {
				return nil, fmt.Errorf("missing resolved scrutinee identity")
			}
			prior := narrowed[resolved.Text]
			if prior.typ != nil && !types.Equal(prior.typ, narrow) {
				return nil, fmt.Errorf("incompatible nominal patterns for repeated scrutinee")
			}
			prior.typ = narrow
			if _, explicit := bindings[name.Name.Name]; !explicit {
				if prior.local == nil {
					var err error
					prior.local, err = c.bind(armScope, name.Name.Name, narrow)
					if err != nil {
						return nil, err
					}
				}
				a.Patterns[i].Narrow = prior.local
			}
			narrowed[resolved.Text] = prior
		}
		if valueType != nil {
			switch body := arm.Body.(type) {
			case *syntax.ValueBody:
				var err error
				a.Value, err = c.expressions(armScope).Check(body.Value, valueType)
				if err != nil {
					return nil, err
				}
			case *syntax.MatchBody:
				if body.Match.Kind != syntax.ValueMatch {
					return nil, fmt.Errorf("completion match is not a value")
				}
				nested, err := c.valueMatch(body.Match, armScope, valueType)
				if err != nil {
					return nil, err
				}
				a.Value = &ir.Expression{Kind: ir.MatchValue, Type: valueType, Span: body.Span, Match: nested}
			default:
				return nil, fmt.Errorf("value match cannot contain completion arms")
			}
		} else {
			var err error
			a.Body, err = c.completion(arm.Body, armScope)
			if err != nil && !c.deferAggregate(err) {
				return nil, err
			}
		}
		out.Arms = append(out.Arms, a)
		previous = append(previous, a.Patterns)
	}
	any := make([]*ir.Pattern, len(columns))
	for i, t := range columns {
		any[i] = &ir.Pattern{Kind: "any", Type: t}
	}
	missing, err := patternsUseful(previous, any, columns)
	if err != nil {
		return nil, err
	}
	if missing {
		return nil, fmt.Errorf("ordinary match is not exhaustive")
	}
	return out, nil
}
