package check

import (
	"fmt"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// codeMissingArm is the stable diagnostic for a completion match that omits
// a required success or bound arm.
const codeMissingArm = "CAN-CHECK-MISSING-ARM"

// armSite carries the syntax a missing-arm diagnostic may reference: the
// match span for the primary location plus the call expression for the
// invocation link. Coordination handlers supply arms without a call.
type armSite struct {
	match source.Span
	call  *syntax.CallExpr
}

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
	out.Arms, err = c.completionArmsCall(n.Arms, out.Call, out.Call.Result, out.Call.Errors, scope, successScope, true, armSite{match: n.Span, call: n.Call})
	return out, err
}

func (c *regionChecker) completionArms(arms []syntax.MatchArm, result *types.Type, errors []*types.Type, scope, successScope bodyScope, requireSuccess bool) ([]ir.Arm, error) {
	site := armSite{}
	if len(arms) != 0 {
		site.match = arms[0].Span
	}
	return c.completionArmsCall(arms, nil, result, errors, scope, successScope, requireSuccess, site)
}

func (c *regionChecker) completionArmsCall(arms []syntax.MatchArm, call *ir.Invocation, result *types.Type, errors []*types.Type, scope, successScope bodyScope, requireSuccess bool, site armSite) ([]ir.Arm, error) {
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
					if isExactSpecialization(err) {
						err = c.locateCode(pattern.Span, "CAN-CHECK-EXACT-SPECIALIZATION", err)
						return nil, source.Relate(c.context.File.Name(), site.match, "matched bound carries several specializations", err)
					}
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
				if escapeErr := c.escaping(actual); escapeErr != nil {
					return nil, c.outward(arm.Span, site.match, escapeErr)
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
	if requireSuccess && !seen["ok"] {
		return nil, c.missingArm(site, arms, call, "completion match requires exactly one success arm", "", nil, true)
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
			omitted := entry
			return nil, c.missingArm(site, arms, call, message, missing, &omitted, false)
		}
	}
	return checked, nil
}

// missingArm reports an omitted success or bound arm with its obligation:
// the primary match span, the invocation link, wrapper policy provenance
// links when a calculated member escapes, and one inserted forward arm
// when forwarding preserves the region contract. Anything else is omitted
// rather than guessed.
func (c *regionChecker) missingArm(site armSite, arms []syntax.MatchArm, call *ir.Invocation, message, missing string, entry *ConcreteError, success bool) error {
	err := fmt.Errorf("%s", message)
	if call != nil && len(call.Steps) == 1 && call.Steps[0].Identity != "" {
		err = fmt.Errorf("%w; %s requires an arm for each bound member", err, call.Steps[0].Identity)
	} else if call != nil {
		err = fmt.Errorf("%w; the invocation requires an arm for each bound member", err)
	} else {
		err = fmt.Errorf("%w; the coordination bound requires an arm for each member", err)
	}
	if site.match != (source.Span{}) && c.context.File != nil {
		err = source.LocateCode(c.context.File.Name(), site.match, codeMissingArm, err)
	}
	if site.call != nil && c.context.File != nil {
		err = source.Relate(c.context.File.Name(), site.call.ExprSpan(), "invocation requires an arm for each bound member", err)
	}
	err = c.relateWrapperArm(call, entry, err)
	err = c.suggestArmFix(arms, missing, entry, success, err)
	return err
}

// relateWrapperArm links the policy rule behind one escaping calculated
// member plus each inherit predecessor on its delegation chain. Rules live
// in checked regions, so the links stay exact across files.
func (c *regionChecker) relateWrapperArm(call *ir.Invocation, entry *ConcreteError, err error) error {
	if err == nil || entry == nil || call == nil || len(call.Steps) != 1 {
		return err
	}
	plan := c.context.Wrappers[call.Steps[0].Identity]
	if plan == nil {
		return err
	}
	provenance := plan.provenance(entry.TypeIdentity)
	if provenance == nil {
		return err
	}
	if rule := plan.rule(provenance.Key); rule != nil && rule.Region != nil {
		err = source.Relate(rule.Region.Source, rule.Region.Span, fmt.Sprintf("policy rule escapes for %s key %s", provenance.Key.Origin, provenance.Key.Decl), err)
	}
	for _, link := range provenance.Chain {
		if link == "default" {
			continue
		}
		for i := range plan.Rules {
			rule := &plan.Rules[i]
			if rule.Handler == link && rule.Region != nil {
				err = source.Relate(rule.Region.Source, rule.Region.Span, "inherit predecessor "+link, err)
				break
			}
		}
	}
	return err
}

// suggestArmFix proposes one inserted arm when the repair preserves the
// region contract: a bare forward for an escapable short error head, or a
// bare ok for a void region missing success. Generic heads, unforwardable
// errors and valued regions are omitted rather than guessed; drivers must
// still recheck every proposal against an isolated snapshot.
func (c *regionChecker) suggestArmFix(arms []syntax.MatchArm, missing string, entry *ConcreteError, success bool, err error) error {
	if err == nil || c.context.File == nil || len(arms) == 0 {
		return err
	}
	var line string
	if success {
		if c.region == nil || c.region.Result == nil || c.region.Result.Kind() != types.Void {
			return err
		}
		line = "ok => ok"
	} else {
		if entry == nil || len(entry.Arguments) != 0 || missing == "" {
			return err
		}
		forward, boundErr := c.context.Registry.Bound([]*types.Type{entry.Type})
		if boundErr != nil || c.context.Errors.CheckEscaping(forward) != nil {
			return err
		}
		line = missing
	}
	text := c.context.File.Text()
	indent, ok := armIndent(text, arms[0].Span.Start)
	if !ok {
		return err
	}
	// C5.1 keeps ok final: insert ahead of the success arm when one is
	// present, otherwise start a fresh line after the last arm.
	offset := -1
	for _, arm := range arms {
		if arm.Outcome != nil && arm.Outcome.Success {
			offset = arm.Span.Start
			for offset > 0 && text[offset-1] == ' ' {
				offset--
			}
			break
		}
	}
	if offset < 0 {
		offset = arms[len(arms)-1].Span.End
		if offset < len(text) && text[offset] == '\n' {
			offset++
		}
	}
	if offset < 0 || offset > len(text) {
		return err
	}
	return source.Suggest(source.Fix{
		Title: "Add missing " + line + " arm",
		File:  c.context.File.Name(),
		Start: offset,
		End:   offset,
		Text:  indent + line + "\n",
	}, err)
}

// armIndent returns the leading whitespace of the line containing offset
// when it is spaces only.
func armIndent(text string, offset int) (string, bool) {
	if offset < 0 || offset > len(text) {
		return "", false
	}
	start := offset
	for start > 0 && text[start-1] != '\n' {
		start--
	}
	indent := text[start:offset]
	if indent == "" {
		return "", false
	}
	for _, r := range indent {
		if r != ' ' {
			return "", false
		}
	}
	return indent, true
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
