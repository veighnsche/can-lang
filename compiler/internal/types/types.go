// Package types owns concrete nominal data and callable contracts. Source
// spellings are resolved before they become graph identities.
package types

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
)

type Kind string

const (
	Primitive Kind = "primitive"
	Void      Kind = "void"
	Array     Kind = "array"
	Record    Kind = "record"
	Error     Kind = "error"
	Variant   Kind = "variant"
	Opaque    Kind = "opaque"
	Callable  Kind = "callable"
	ChoiceArm Kind = "choice_arm"
)

type Field struct {
	Name string
	Type *Type
}

// Type is an interned concrete node. Its shape cannot be changed by consumers.
// Recursive fields refer to the same node instead of duplicating a string tree.
type Type struct {
	kind            Kind
	id, declaration string
	key             string
	arguments       []*Type
	fields          []Field
	alternatives    []*Type
	element, result *Type
	inputs, errors  []*Type
	defined         bool
	standardFailure bool
	graph           *graph
}

func (t *Type) Kind() Kind          { return t.kind }
func (t *Type) Identity() string    { return t.id }
func (t *Type) Declaration() string { return t.declaration }
func (t *Type) Arguments() []*Type  { return append([]*Type(nil), t.arguments...) }
func (t *Type) Fields() []Field     { return append([]Field(nil), t.fields...) }
func (t *Type) Element() *Type      { return t.element }
func (t *Type) Result() *Type       { return t.result }
func (t *Type) Inputs() []*Type     { return append([]*Type(nil), t.inputs...) }
func (t *Type) Errors() []*Type     { return append([]*Type(nil), t.errors...) }

// JSON array framing separates declaration IDs, argument IDs and structural
// constructors without relying on source punctuation being absent from an ID.
func identity(kind Kind, name string, arguments []*Type) string {
	parts := []string{string(kind), name}
	for _, arg := range arguments {
		parts = append(parts, arg.id)
	}
	data, _ := json.Marshal(parts)
	return string(data)
}

type graph struct {
	nodes  map[string]*Type
	sealed bool
	leaves map[*Type][]*Type
}

func newGraph() *graph { return &graph{nodes: map[string]*Type{}, leaves: map[*Type][]*Type{}} }
func (g *graph) intern(t *Type) *Type {
	// Bound identity size even for deeply nested arrays/generic arguments.
	// Keep the framed key so a hash collision refuses instead of merging types.
	t.key = t.id
	sum := sha256.Sum256([]byte("can-concrete-type-v1\x00" + t.key))
	t.id = hex.EncodeToString(sum[:])
	if old := g.nodes[t.id]; old != nil {
		if old.key != t.key {
			panic("concrete type identity collision")
		}
		return old
	}
	if g.sealed {
		panic("cannot extend a sealed type graph")
	}
	t.graph = g
	g.nodes[t.id] = t
	return t
}
func (g *graph) scalar(name string) *Type {
	kind := Primitive
	switch name {
	case "int", "float", "bool", "str":
	case "void":
		kind = Void
	default:
		panic("unknown primitive")
	}
	return g.intern(&Type{kind: kind, id: identity(kind, name, nil), declaration: name, defined: true})
}
func (g *graph) data(t *Type) error {
	if t == nil || t.graph != g {
		return fmt.Errorf("type belongs to a different graph")
	}
	if t.kind == Void {
		return fmt.Errorf("void is not a data type")
	}
	return nil
}
func (g *graph) array(element *Type) (*Type, error) {
	if err := g.data(element); err != nil {
		return nil, err
	}
	return g.intern(&Type{kind: Array, id: identity(Array, "", []*Type{element}), element: element, defined: true}), nil
}
func (g *graph) nominal(kind Kind, declaration string, arguments []*Type) (*Type, error) {
	switch kind {
	case Record, Error, Variant, Opaque:
	default:
		return nil, fmt.Errorf("invalid nominal kind %s", kind)
	}
	if declaration == "" {
		return nil, fmt.Errorf("nominal declaration identity is empty")
	}
	for _, arg := range arguments {
		if err := g.data(arg); err != nil {
			return nil, err
		}
	}
	return g.intern(&Type{kind: kind, id: identity(kind, declaration, arguments), declaration: declaration, arguments: append([]*Type(nil), arguments...)}), nil
}
func (g *graph) define(t *Type, fields []Field, alternatives []*Type) error {
	if g.sealed || t == nil || t.graph != g || t.defined {
		return fmt.Errorf("type cannot be defined again")
	}
	if t.kind == Variant {
		if len(fields) != 0 || len(alternatives) == 0 {
			return fmt.Errorf("variant must have nonempty alternatives and no fields")
		}
	} else if len(alternatives) != 0 {
		return fmt.Errorf("only variants have alternatives")
	}
	if t.kind != Record && t.kind != Error && len(fields) != 0 {
		return fmt.Errorf("only ordinary records and errors have representation fields")
	}
	names := map[string]bool{}
	for _, field := range fields {
		if field.Name == "" || names[field.Name] {
			return fmt.Errorf("duplicate or empty field %q", field.Name)
		}
		names[field.Name] = true
		if err := g.data(field.Type); err != nil {
			return err
		}
	}
	for _, alternative := range alternatives {
		if err := g.data(alternative); err != nil {
			return err
		}
	}
	t.fields = append([]Field(nil), fields...)
	t.alternatives = append([]*Type(nil), alternatives...)
	t.defined = true
	return nil
}
func (g *graph) function(kind Kind, result *Type, inputs, errors []*Type) (*Type, error) {
	if kind != Callable && kind != ChoiceArm {
		return nil, fmt.Errorf("invalid functional type")
	}
	if result == nil || result.graph != g {
		return nil, fmt.Errorf("invalid result graph")
	}
	if kind == ChoiceArm && len(inputs) != 0 {
		return nil, fmt.Errorf("choice arm has no ordinary inputs")
	}
	for _, input := range inputs {
		if err := g.data(input); err != nil {
			return nil, err
		}
	}
	bound := append([]*Type(nil), errors...)
	for _, e := range bound {
		if err := g.data(e); err != nil {
			return nil, err
		}
	}
	sort.Slice(bound, func(i, j int) bool { return bound[i].id < bound[j].id })
	for i, e := range bound {
		if err := g.data(e); err != nil {
			return nil, err
		}
		if e.kind != Error {
			return nil, fmt.Errorf("emits requires nominal errors")
		}
		if i > 0 && e.id == bound[i-1].id {
			return nil, fmt.Errorf("duplicate error in bound")
		}
	}
	// A nested array frame distinguishes the ordered input list from the set of
	// errors, including the nullary and empty-bound cases.
	key, _ := json.Marshal([]any{kind, result.id, ids(inputs), ids(bound)})
	return g.intern(&Type{kind: kind, id: string(key), result: result, inputs: append([]*Type(nil), inputs...), errors: bound, defined: true}), nil
}
func ids(ts []*Type) []string {
	out := make([]string, len(ts))
	for i, t := range ts {
		out[i] = t.id
	}
	return out
}
func (g *graph) ordered() []*Type {
	out := make([]*Type, 0, len(g.nodes))
	for _, t := range g.nodes {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].id < out[j].id })
	return out
}
