package types

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
)

// Specializer admits body-discovered annotations without reopening any graph
// used by checked expressions. A failed extension cannot invalidate prior types.
type Specializer struct {
	world *resolve.World
	types map[string]*Type
}

func NewSpecializer(world *resolve.World, initial *Model) (*Specializer, error) {
	if world == nil || initial == nil {
		return nil, fmt.Errorf("specialization requires resolved declarations and sealed initial types")
	}
	s := &Specializer{world: world, types: map[string]*Type{}}
	for _, typ := range initial.Types() {
		if !Equal(typ, typ) {
			return nil, fmt.Errorf("unsealed initial specialization evidence")
		}
		s.types[typ.id] = typ
	}
	return s, nil
}

// importChecked copies only the reachable argument graph, retaining cycles and
// shared substructure by identity. The destination owns every mutable node until
// Finish validates and seals it; source evidence remains untouched throughout.
func (b *Builder) importChecked(source *Type) (*Type, error) {
	if !Equal(source, source) {
		return nil, fmt.Errorf("specialization argument requires sealed evidence")
	}
	if prior := b.graph.nodes[source.id]; prior != nil {
		if prior.key != source.key {
			return nil, fmt.Errorf("concrete type identity collision")
		}
		return prior, nil
	}
	copy := *source
	copy.graph = b.graph
	b.graph.nodes[copy.id] = &copy
	clone := func(list []*Type) ([]*Type, error) {
		result := make([]*Type, len(list))
		for i, child := range list {
			var err error
			result[i], err = b.importChecked(child)
			if err != nil {
				return nil, err
			}
		}
		return result, nil
	}
	var err error
	if copy.arguments, err = clone(source.arguments); err != nil {
		return nil, err
	}
	if copy.alternatives, err = clone(source.alternatives); err != nil {
		return nil, err
	}
	if copy.inputs, err = clone(source.inputs); err != nil {
		return nil, err
	}
	if copy.errors, err = clone(source.errors); err != nil {
		return nil, err
	}
	copy.fields = make([]Field, len(source.fields))
	for i, field := range source.fields {
		typ, err := b.importChecked(field.Type)
		if err != nil {
			return nil, err
		}
		copy.fields[i] = Field{Name: field.Name, Type: typ}
	}
	if source.element != nil {
		copy.element, err = b.importChecked(source.element)
		if err != nil {
			return nil, err
		}
	}
	if source.result != nil {
		copy.result, err = b.importChecked(source.result)
		if err != nil {
			return nil, err
		}
	}
	return &copy, nil
}

func (s *Specializer) Resolve(file *resolve.File, node syntax.TypeNode, parameters map[string]*Type, allowVoid bool) (*Type, error) {
	b := NewBuilder(s.world)
	bound := map[string]*Type{}
	for name, typ := range parameters {
		if typ == nil || typ.Kind() == Void {
			return nil, fmt.Errorf("invalid generic data argument %s", name)
		}
		copy, err := b.importChecked(typ)
		if err != nil {
			return nil, err
		}
		bound[name] = copy
	}
	typ, err := b.Resolve(file, node, bound, allowVoid)
	if err != nil {
		return nil, err
	}
	model, err := b.Finish()
	if err != nil {
		return nil, err
	}
	for _, candidate := range model.Types() {
		if prior := s.types[candidate.id]; prior != nil && prior.key != candidate.key {
			return nil, fmt.Errorf("concrete type identity collision")
		}
	}
	for _, candidate := range model.Types() {
		if s.types[candidate.id] == nil {
			s.types[candidate.id] = candidate
		}
	}
	return s.types[typ.id], nil
}

func (s *Specializer) Model() *Model {
	result := &Model{}
	for _, typ := range s.types {
		result.types = append(result.types, typ)
	}
	sort.Slice(result.types, func(i, j int) bool { return result.types[i].id < result.types[j].id })
	return result
}

// SpecializationKey uses resolved identity and concrete arguments, never source
// aliases or printed type spellings. All callers share one cached body per key.
func SpecializationKey(declaration string, arguments []*Type) (string, error) {
	if declaration == "" {
		return "", fmt.Errorf("missing specialization declaration identity")
	}
	parts := []string{declaration}
	for _, arg := range arguments {
		if !Equal(arg, arg) || arg.kind == Void {
			return "", fmt.Errorf("specialization requires concrete data arguments")
		}
		if name := OpaqueParameterName(arg); name != "" {
			return "", fmt.Errorf("cannot specialize %s with opaque type parameter %s from an exported generic declaration: pass the value through a non-generic contract or an explicit callable input", declaration, name)
		}
		parts = append(parts, arg.id)
	}
	encoded, err := json.Marshal(parts)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(append([]byte("can-specialization-v1\x00"), encoded...))
	return declaration + "/instance/" + hex.EncodeToString(digest[:]), nil
}
