package types

import (
	"errors"
	"fmt"
	"github.com/veighnsche/can-lang/compiler/internal/source"
)

// seal validates the whole finite graph before exposing any compatibility
// evidence. Back edges through records are legal; back edges through variants
// alone are not. Expansion to infinitely many distinct nodes is the builder's
// responsibility, separate from this finite-graph analysis.
type typeProblem struct {
	typ *Type
	err error
}

func (g *graph) seal() error {
	problems := g.validateAndSeal(nil, false)
	var failures []error
	for _, problem := range problems {
		failures = append(failures, problem.err)
	}
	return errors.Join(failures...)
}

// validateAndSeal runs one finite graph validation and propagates invalid
// prerequisites to their reverse dependencies. Recovery seals only nodes whose
// complete outgoing dependency closure is valid; rejected nodes remain unsealed.
func (g *graph) validateAndSeal(invalid map[*Type]error, recovering bool) []typeProblem {
	if g.sealed {
		return nil
	}
	if invalid == nil {
		invalid = map[*Type]error{}
	}
	nodes := g.ordered()
	for _, t := range nodes {
		if !t.defined {
			invalid[t] = fmt.Errorf("undefined concrete type %s", t.id)
		}
	}
	g.leaves = map[*Type][]*Type{}
	active := map[*Type]bool{}
	var flatten func(*Type) ([]*Type, error)
	flatten = func(t *Type) ([]*Type, error) {
		if t.kind != Variant {
			if t.kind == Record || t.kind == Error || t.kind == Opaque && t.standardFailure || t.kind == Parameter {
				return []*Type{t}, nil
			}
			return nil, fmt.Errorf("ineligible variant leaf %s", t.id)
		}
		if active[t] {
			return nil, fmt.Errorf("recursive variant-only alternatives: %s", t.id)
		}
		if invalid[t] != nil {
			return nil, &source.BlockedError{Dependency: t.declaration}
		}
		if leaves, ok := g.leaves[t]; ok {
			return leaves, nil
		}
		active[t] = true
		defer delete(active, t)
		seen := map[string]bool{}
		var leaves []*Type
		for _, alternative := range t.alternatives {
			nested, err := flatten(alternative)
			if err != nil {
				return nil, err
			}
			for _, leaf := range nested {
				if seen[leaf.id] {
					return nil, fmt.Errorf("duplicate variant leaf %s", leaf.id)
				}
				seen[leaf.id] = true
				leaves = append(leaves, leaf)
			}
		}
		g.leaves[t] = leaves
		return leaves, nil
	}
	for _, t := range nodes {
		if t.kind == Variant && invalid[t] == nil {
			if _, err := flatten(t); err != nil {
				invalid[t] = err
			}
		}
	}
	propagate := func() {
		for changed := true; changed; {
			changed = false
			for _, t := range nodes {
				if invalid[t] != nil {
					continue
				}
				for _, dependency := range typeDependencies(t) {
					if invalid[dependency] != nil {
						invalid[t] = &source.BlockedError{Dependency: dependency.declaration}
						changed = true
						break
					}
				}
			}
		}
	}
	propagate()
	inhabited := map[*Type]bool{}
	for _, t := range nodes {
		if invalid[t] != nil {
			continue
		}
		switch t.kind {
		case Primitive, Void, Array, Opaque, Callable, ChoiceArm, Parameter:
			inhabited[t] = true
		}
	}
	for changed := true; changed; {
		changed = false
		for _, t := range nodes {
			if inhabited[t] || invalid[t] != nil {
				continue
			}
			possible := t.kind == Record || t.kind == Error
			if possible {
				for _, field := range t.fields {
					possible = possible && inhabited[field.Type]
				}
			}
			if t.kind == Variant {
				for _, leaf := range g.leaves[t] {
					possible = possible || inhabited[leaf]
				}
			}
			if possible {
				inhabited[t] = true
				changed = true
			}
		}
	}
	for _, t := range nodes {
		if !inhabited[t] && invalid[t] == nil {
			invalid[t] = fmt.Errorf("type has no finite inhabitant: %s", t.id)
		}
	}
	propagate()
	var problems []typeProblem
	for _, t := range nodes {
		if err := invalid[t]; err != nil {
			problems = append(problems, typeProblem{t, err})
		}
	}
	if len(problems) > 0 && !recovering {
		return problems
	}
	if recovering && len(problems) > 0 {
		rejected := newGraph()
		for _, problem := range problems {
			delete(g.nodes, problem.typ.id)
			delete(g.leaves, problem.typ)
			problem.typ.graph = rejected
			rejected.nodes[problem.typ.id] = problem.typ
		}
	}
	g.sealed = true
	return problems
}

func typeDependencies(t *Type) []*Type {
	out := append([]*Type(nil), t.arguments...)
	out = append(out, t.alternatives...)
	out = append(out, t.inputs...)
	out = append(out, t.errors...)
	if t.element != nil {
		out = append(out, t.element)
	}
	if t.result != nil {
		out = append(out, t.result)
	}
	for _, field := range t.fields {
		out = append(out, field.Type)
	}
	return out
}

func (t *Type) Leaves() []*Type {
	if !t.graph.sealed || t.kind != Variant {
		return nil
	}
	return append([]*Type(nil), t.graph.leaves[t]...)
}

// EqualityEligible is a reachability property, not the inhabitation fixed point.
// A recursive eligible node may be revisited, but every outgoing field is still
// explored: a callable reached after a back edge must reject the whole type.
func EqualityEligible(t *Type) bool {
	if t == nil || !t.graph.sealed {
		return false
	}
	visited := map[*Type]bool{}
	var visit func(*Type) bool
	visit = func(t *Type) bool {
		if visited[t] {
			return true
		}
		visited[t] = true
		switch t.kind {
		case Primitive:
			return true
		case Array:
			return visit(t.element)
		case Record, Error:
			for _, f := range t.fields {
				if !visit(f.Type) {
					return false
				}
			}
			return true
		case Variant:
			for _, leaf := range t.graph.leaves[t] {
				if !visit(leaf) {
					return false
				}
			}
			return true
		case Parameter:
			// Nothing is known about an opaque variable, including
			// whether its future arguments support equality.
			return false
		default:
			return false
		}
	}
	return visit(t)
}
