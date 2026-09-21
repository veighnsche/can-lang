package emit

import (
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// RegionTypeDeclarations includes derived expression types as well as declared
// signatures. ArrayOfChecked may create a sealed node outside the initial model;
// collecting only the declaration graph would omit its TypeScript alias.
func RegionTypeDeclarations(regions ...*ir.Region) (string, error) {
	return NativeTypeDeclarations(regionTypes(regions...))
}
func regionTypes(regions ...*ir.Region) []*types.Type {
	var roots []*types.Type
	add := func(t *types.Type) {
		if t != nil {
			roots = append(roots, t)
		}
	}
	var expression func(*ir.Expression)
	var invocation func(*ir.Invocation)
	var match func(*ir.Match)
	var completion func(*ir.Completion)
	var block func(*ir.Block)
	var pattern func(*ir.Pattern)
	local := func(l *ir.Local) {
		if l != nil {
			add(l.Type)
		}
	}
	expression = func(e *ir.Expression) {
		if e == nil {
			return
		}
		add(e.Type)
		for _, input := range e.Inputs {
			expression(input)
		}
		invocation(e.Invocation)
		match(e.Match)
	}
	invocation = func(call *ir.Invocation) {
		if call == nil {
			return
		}
		add(call.Result)
		roots = append(roots, call.Errors...)
		for _, step := range call.Steps {
			for _, prepared := range step.Prepare {
				add(prepared.Local.Type)
				expression(prepared.Value)
			}
			add(step.Result)
			roots = append(roots, step.Errors...)
			expression(step.Native)
			for _, arg := range step.Arguments {
				expression(arg)
			}
		}
	}
	pattern = func(p *ir.Pattern) {
		if p == nil {
			return
		}
		add(p.Type)
		local(p.Binding)
		local(p.Rest)
		local(p.Narrow)
		for _, child := range p.Children {
			pattern(child)
		}
	}
	match = func(m *ir.Match) {
		if m == nil {
			return
		}
		add(m.ValueResult)
		invocation(m.Call)
		for _, value := range m.Values {
			expression(value)
		}
		for _, arm := range m.Arms {
			add(arm.Error)
			local(arm.Binding)
			expression(arm.Value)
			completion(arm.Body)
			for _, p := range arm.Patterns {
				pattern(p)
			}
		}
	}
	completion = func(c *ir.Completion) {
		if c == nil {
			return
		}
		expression(c.Value)
		invocation(c.Call)
		block(c.Block)
		match(c.Match)
	}
	block = func(b *ir.Block) {
		if b == nil {
			return
		}
		for _, step := range b.Steps {
			local(step.Local)
			expression(step.Value)
			invocation(step.Call)
		}
		completion(b.Terminal)
	}
	for _, region := range regions {
		if region == nil {
			continue
		}
		add(region.Result)
		roots = append(roots, region.Errors...)
		for _, input := range region.Inputs {
			add(input.Type)
		}
		block(region.Body)
	}
	return roots
}
