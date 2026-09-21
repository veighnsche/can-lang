package types

import "fmt"

// seal validates the whole finite graph before exposing any compatibility
// evidence. Back edges through records are legal; back edges through variants
// alone are not. Expansion to infinitely many distinct nodes is the builder's
// responsibility, separate from this finite-graph analysis.
func (g *graph) seal() error {
	if g.sealed {
		return nil
	}
	nodes := g.ordered()
	for _, t := range nodes {
		if !t.defined {
			return fmt.Errorf("undefined concrete type %s", t.id)
		}
	}
	g.leaves = map[*Type][]*Type{}
	active := map[*Type]bool{}
	var flatten func(*Type) ([]*Type, error)
	flatten = func(t *Type) ([]*Type, error) {
		if t.kind != Variant {
			if t.kind == Record || t.kind == Error || t.kind == Opaque && t.standardFailure {
				return []*Type{t}, nil
			}
			return nil, fmt.Errorf("ineligible variant leaf %s", t.id)
		}
		if active[t] {
			return nil, fmt.Errorf("recursive variant-only alternatives: %s", t.id)
		}
		if leaves, ok := g.leaves[t]; ok {
			return leaves, nil
		}
		active[t] = true
		seen := map[string]bool{}
		var leaves []*Type
		for _, a := range t.alternatives {
			nested, err := flatten(a)
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
		delete(active, t)
		g.leaves[t] = leaves
		return leaves, nil
	}
	for _, t := range nodes {
		if t.kind == Variant {
			if _, err := flatten(t); err != nil {
				return err
			}
		}
	}
	// Least fixed point: never assume a required recursive field already has a
	// value. Empty arrays, primitives and callable/resource values are bases.
	inhabited := map[*Type]bool{}
	for _, t := range nodes {
		switch t.kind {
		case Primitive, Void, Array, Opaque, Callable, ChoiceArm:
			inhabited[t] = true
		}
	}
	for changed := true; changed; {
		changed = false
		for _, t := range nodes {
			if inhabited[t] {
				continue
			}
			possible := t.kind == Record || t.kind == Error
			if possible {
				for _, f := range t.fields {
					possible = possible && inhabited[f.Type]
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
		if !inhabited[t] {
			return fmt.Errorf("type has no finite inhabitant: %s", t.id)
		}
	}
	g.sealed = true
	return nil
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
		default:
			return false
		}
	}
	return visit(t)
}
