package check

import (
	"fmt"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
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
	if len(n.When) != 0 {
		return nil, fmt.Errorf("when fixtures require assertion checking")
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
	bound, err := c.context.Registry.Bound(out.Call.Errors)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, arm := range n.Arms {
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
			bindingType = out.Call.Result
			armScope = c.child(successScope)
		case pattern.StandardFailure:
			if pattern.Error != nil || arm.Forward {
				return nil, fmt.Errorf("invalid standard pattern")
			}
			a.Outcome = "standard"
			key = "standard"
			bindingType = c.context.Expressions.Scalars["str"]
		default:
			name, ok := pattern.Error.(*syntax.NamedType)
			if !ok || len(name.Arguments) != 0 {
				return nil, fmt.Errorf("domain pattern must be a bare error name")
			}
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
				return nil, err
			}
			concrete, err := bound.ResolveBareArm(declaration)
			if err != nil {
				return nil, err
			}
			a.Outcome = "domain"
			a.Error = concrete.Type
			key = "domain:" + concrete.TypeIdentity
			bindingType = concrete.Type
			bindingName = name.Name.Name
		}
		if seen[key] {
			return nil, fmt.Errorf("duplicate completion arm")
		}
		seen[key] = true
		if pattern.Binding != nil {
			if a.Outcome == "domain" {
				return nil, fmt.Errorf("domain arm binds its declared error name")
			}
			declared, err := c.context.Type(pattern.Binding.Type, false)
			if err != nil {
				return nil, err
			}
			if !types.Equal(declared, bindingType) {
				return nil, fmt.Errorf("completion arm binding type mismatch")
			}
			bindingName = pattern.Binding.Name.Text
		}
		if bindingName != "" && !arm.Forward {
			a.Binding, err = c.bind(armScope, bindingName, bindingType)
			if err != nil {
				return nil, err
			}
		}
		if arm.Forward {
			if arm.Body != nil || pattern.Binding != nil {
				return nil, fmt.Errorf("forwarding arm cannot contain a body or binding")
			}
			if a.Outcome == "ok" {
				if !types.Assignable(out.Call.Result, c.region.Result) {
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
			if err != nil {
				return nil, err
			}
		}
		out.Arms = append(out.Arms, a)
	}
	if !seen["ok"] {
		return nil, fmt.Errorf("completion match requires exactly one success arm")
	}
	for _, entry := range bound.Entries() {
		if !seen["domain:"+entry.TypeIdentity] {
			return nil, fmt.Errorf("missing completion arm for %s", entry.Declaration.Name)
		}
	}
	return out, nil
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
			if err != nil {
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
