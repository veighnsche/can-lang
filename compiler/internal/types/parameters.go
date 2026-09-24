package types

import (
	"fmt"
	"strings"
)

// SymbolicParameter manufactures one sealed opaque type variable for an
// exported generic declaration's symbolic body check. The declaration
// identity qualifies the name, so identical parameter names from different
// declarations never compare equal. The returned node is a leaf: it has no
// fields, alternatives, element, result, inputs or errors, and every
// representation-requiring operation rejects it by failing its kind gate.
func SymbolicParameter(declaration, name string) (*Type, error) {
	if declaration == "" || name == "" {
		return nil, fmt.Errorf("opaque type parameter requires its declaring symbol and name")
	}
	g := newGraph()
	t := g.intern(&Type{kind: Parameter, id: identity(Parameter, declaration+"<"+name+">", nil), declaration: declaration + "<" + name + ">", defined: true})
	if err := g.seal(); err != nil {
		return nil, err
	}
	return t, nil
}

// MentionsParameter reports whether t's reachable graph contains an opaque
// type variable, either directly or nested under an array, nominal,
// callable or choice-arm constructor. It never follows variant leaf caches,
// only structural children, and tolerates recursive graphs.
func MentionsParameter(t *Type) bool {
	return firstParameter(t, map[*Type]bool{}) != nil
}

// OpaqueParameterName renders the first opaque type variable in t's graph
// for diagnostics: the short `declaration<name>` spelling, or "" when t
// mentions no parameter.
func OpaqueParameterName(t *Type) string {
	p := firstParameter(t, map[*Type]bool{})
	if p == nil {
		return ""
	}
	name := p.declaration
	if i := strings.LastIndex(name, "::"); i >= 0 {
		name = name[i+2:]
	}
	return name
}

func firstParameter(t *Type, seen map[*Type]bool) *Type {
	if t == nil || seen[t] {
		return nil
	}
	seen[t] = true
	if t.kind == Parameter {
		return t
	}
	children := append(append(append([]*Type(nil), t.arguments...), t.alternatives...), t.inputs...)
	children = append(children, t.errors...)
	for _, field := range t.fields {
		children = append(children, field.Type)
	}
	if t.element != nil {
		children = append(children, t.element)
	}
	if t.result != nil {
		children = append(children, t.result)
	}
	for _, child := range children {
		if found := firstParameter(child, seen); found != nil {
			return found
		}
	}
	return nil
}
