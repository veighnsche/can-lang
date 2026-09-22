package check

import (
	"fmt"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

func (c *regionChecker) coordination(n syntax.Coordination, scope bodyScope, expected *types.Type) (*ir.Coordination, error) {
	if len(n.Participants) == 0 {
		return nil, fmt.Errorf("coordination requires participant entries")
	}
	if expected == nil {
		var err error
		expected, err = c.context.Type(named("void"), true)
		if err != nil {
			return nil, err
		}
	}
	site, err := c.lexicalSite("coordination", n.Span)
	if err != nil {
		return nil, err
	}
	out := &ir.Coordination{Site: site, Span: n.Span, Result: expected}
	switch n.Mode {
	case "concurrent":
		out.Mode = "all"
		if n.WithError {
			out.Mode = "settled"
		}
		if expected.Kind() != types.Void && expected.Kind() != types.Array {
			return nil, fmt.Errorf("concurrent binding requires an array result")
		}
	case "race":
		out.Mode = "any"
		if n.WithError {
			out.Mode = "race"
		}
	default:
		return nil, fmt.Errorf("unknown coordination mode")
	}
	var success *types.Type
	var errors []*types.Type
	for _, entry := range n.Participants {
		participant := ir.Participant{Span: entry.Span}
		if entry.Spread != nil {
			if entry.Call != nil {
				return nil, fmt.Errorf("participant cannot be both call and spread")
			}
			spread, err := c.expressions(scope).Check(entry.Spread, nil)
			if err != nil {
				return nil, err
			}
			if spread.Type.Kind() != types.Array || spread.Type.Element().Kind() != types.Callable || len(spread.Type.Element().Inputs()) != 0 {
				return nil, fmt.Errorf("coordination spread requires an array of nullary callables")
			}
			participant.Spread = spread
			participant.Result = spread.Type.Element().Result()
			participant.Errors = spread.Type.Element().Errors()
		} else {
			if entry.Call == nil {
				return nil, fmt.Errorf("missing participant invocation")
			}
			arms := entry.Arms
			if n.Mode == "race" {
				arms = n.Arms
			}
			var want *types.Type
			for _, arm := range arms {
				if arm.Outcome != nil && arm.Outcome.Success && arm.Outcome.Binding != nil {
					var err error
					want, err = c.context.Type(arm.Outcome.Binding.Type, false)
					if err != nil {
						return nil, err
					}
					break
				}
			}
			call, err := c.invocation(entry.Call, scope, want)
			if err != nil {
				return nil, err
			}
			participant.Call = call
			participant.Result = call.Result
			participant.Errors = call.Errors
		}
		if n.Mode == "race" {
			if len(entry.Arms) != 0 {
				return nil, fmt.Errorf("race uses shared completion arms")
			}
			if success != nil && !types.Equal(success, participant.Result) {
				return nil, fmt.Errorf("race participant success types must be identical")
			}
			success = participant.Result
		} else {
			result := expected
			if expected.Kind() == types.Array {
				result = expected.Element()
			}
			bound := participant.Errors
			if !n.WithError {
				bound = nil
				for _, arm := range entry.Arms {
					if arm.Outcome == nil || !arm.Outcome.Success {
						return nil, fmt.Errorf("plain concurrent participant handles only success")
					}
				}
			}
			handler, err := c.coordinationHandler(entry.Arms, participant.Result, bound, result, scope, true)
			if err != nil {
				return nil, err
			}
			participant.Handler = handler
			out.Errors = unionErrors(out.Errors, handler.Region.Escapes)
		}
		errors = unionErrors(errors, participant.Errors)
		out.Entries = append(out.Entries, participant)
	}
	if out.Mode == "any" {
		// Distinct generic specializations stay distinct in the collected
		// leaf set; only identical types deduplicate via unionErrors.
		var successArms []syntax.MatchArm
		var aggregate *syntax.MatchArm
		for index := range n.Arms {
			arm := n.Arms[index]
			if arm.Outcome == nil {
				return nil, fmt.Errorf("race requires completion patterns")
			}
			if arm.Outcome.Success {
				successArms = append(successArms, arm)
				continue
			}
			name, ok := arm.Outcome.Error.(*syntax.NamedType)
			if !ok || arm.Outcome.StandardFailure {
				return nil, c.locate(arm.Outcome.Span, fmt.Errorf("first-success race only handles success and all_failed"))
			}
			declaration, err := c.context.ErrorName(name.Name)
			if err != nil {
				return nil, c.locate(arm.Outcome.Span, err)
			}
			if declaration != "can.prelude@1::all_failed" {
				return nil, c.locate(arm.Outcome.Span, fmt.Errorf("first-success race only handles success and all_failed"))
			}
			if aggregate != nil {
				return nil, c.locate(arm.Outcome.Span, fmt.Errorf("race requires exactly one all_failed arm"))
			}
			aggregate = &n.Arms[index]
		}
		if aggregate == nil {
			return nil, fmt.Errorf("race requires all_failed arm")
		}
		out.Shared, err = c.coordinationHandler(successArms, success, nil, expected, scope, true)
		if err != nil {
			return nil, err
		}

		head, _ := aggregate.Outcome.Error.(*syntax.NamedType)
		if head != nil && len(head.Arguments) != 0 {
			resolved, resolveErr := c.context.Type(head, false)
			if resolveErr != nil {
				return nil, c.locate(aggregate.Outcome.Span, resolveErr)
			}
			if _, concreteErr := c.context.Registry.Concrete(resolved); concreteErr != nil {
				return nil, c.locate(aggregate.Outcome.Span, concreteErr)
			}
			if err = c.aggregateCoverage(resolved, errors); err != nil {
				return nil, c.locate(aggregate.Outcome.Span, err)
			}
			out.AggregateType = resolved
			// The explicit head selects its specialization; the shared arm
			// checker binds the alias (or short name), checks the body, and
			// validates forwarding against the enclosing contract. Expected
			// `.failures` uses agree through the fixed payload type.
			out.Aggregate, err = c.coordinationHandler([]syntax.MatchArm{*aggregate}, nil, []*types.Type{resolved}, expected, scope, false)
			if err != nil {
				return nil, err
			}
		} else if aggregate.Forward {
			concrete, err := c.context.Errors.ResolveBareArm("can.prelude@1::all_failed")
			if err != nil {
				return nil, c.locate(aggregate.Outcome.Span, fmt.Errorf("forwarded all_failed needs one enclosing named variant specialization: %w", err))
			}
			if err = c.aggregateCoverage(concrete.Type, errors); err != nil {
				return nil, c.locate(aggregate.Outcome.Span, err)
			}
			out.AggregateType = concrete.Type
			out.Aggregate, err = c.coordinationHandler([]syntax.MatchArm{*aggregate}, nil, []*types.Type{concrete.Type}, expected, scope, false)
			if err != nil {
				return nil, err
			}

		} else {
			handler, typ, err := c.observedAggregateHandler(aggregate, errors, expected, scope)
			if err != nil {
				return nil, err
			}
			out.Aggregate = handler
			out.AggregateType = typ
		}

		out.Errors = unionErrors(out.Errors, out.Shared.Region.Escapes, out.Aggregate.Region.Escapes)
		c.region.Escapes = unionErrors(c.region.Escapes, out.Errors)
		return out, nil
	}
	if out.Mode == "settled" {
		if len(n.Arms) != 0 {
			return nil, fmt.Errorf("concurrent with error requires per-participant arms")
		}
	} else {
		requireSuccess := out.Mode == "race"
		if !requireSuccess {
			for _, arm := range n.Arms {
				if arm.Outcome != nil && arm.Outcome.Success {
					return nil, fmt.Errorf("concurrent success arms belong beneath participants")
				}
			}
		}
		handler, err := c.coordinationHandler(n.Arms, success, errors, expected, scope, requireSuccess)
		if err != nil {
			return nil, err
		}
		out.Shared = handler
		out.Errors = unionErrors(out.Errors, handler.Region.Escapes)
	}
	c.region.Escapes = unionErrors(c.region.Escapes, out.Errors)
	return out, nil
}

func (c *regionChecker) coordinationHandler(arms []syntax.MatchArm, success *types.Type, errors []*types.Type, result *types.Type, scope bodyScope, requireSuccess bool) (*ir.OutcomeHandler, error) {
	region := &ir.Region{ID: c.identity("coordination-handler"), Parent: c.region.ID, Source: c.region.Source, Kind: ir.HandlerRegion, Result: result, Errors: append([]*types.Type(nil), c.region.Errors...)}
	context := c.context
	context.Identity = region.ID
	context.Parent = region.Parent
	context.Kind = ir.HandlerRegion
	context.Result = result
	child := &regionChecker{aggregate: c.aggregate, context: context, region: region, locals: map[string]*types.Type{}, uses: c.uses}
	for id, typ := range c.locals {
		child.locals[id] = typ
	}
	checked, err := child.completionArms(arms, success, errors, scope, scope, requireSuccess)
	if err != nil {
		return nil, err
	}
	return &ir.OutcomeHandler{Region: region, Arms: checked}, nil
}

func (c *regionChecker) aggregateCoverage(aggregate *types.Type, errors []*types.Type) error {
	if aggregate == nil || aggregate.Declaration() != "can.prelude@1::all_failed" || len(aggregate.Arguments()) != 1 || aggregate.Arguments()[0].Kind() != types.Variant {
		return fmt.Errorf("all_failed requires one named failure variant")
	}
	variant := aggregate.Arguments()[0]
	standard, err := c.context.Type(named("standard_failure"), false)
	if err != nil {
		return err
	}
	for _, leaf := range append(append([]*types.Type(nil), errors...), standard) {
		if !types.Assignable(leaf, variant) {
			return fmt.Errorf("all_failed failure variant does not cover %s", leaf.Declaration())
		}
	}
	return nil
}
