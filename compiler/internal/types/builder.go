package types

import (
	"fmt"
	"sort"

	"github.com/veighnsche/can-lang/compiler/internal/catalogue"
	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
)

// Builder gathers concrete annotations and reachable declaration shapes. Finish
// validates the entire graph; no unsealed type is compatibility evidence.
type Builder struct {
	world       *resolve.World
	graph       *graph
	catalogue   map[string]*resolve.Symbol
	constraints []constraint
	pending     map[*Type]bool
	active      map[string]int
	failure     error
}
type constraint struct {
	argument *Type
	name     string
}
type Model struct{ types []*Type }

func (m *Model) Types() []*Type { return append([]*Type(nil), m.types...) }

func NewBuilder(world *resolve.World) *Builder {
	b := &Builder{world: world, graph: newGraph(), catalogue: map[string]*resolve.Symbol{}, active: map[string]int{}}
	for name, symbol := range world.Prelude.Symbols {
		b.catalogue[name] = symbol
	}
	for name, pkg := range world.Packages {
		if pkg.Source != nil {
			continue
		}
		for local, symbol := range pkg.Scope.Symbols {
			b.catalogue[name+"::"+local] = symbol
		}
	}
	return b
}

// Resolve checks a source annotation in its file-local import scope. Parameter
// bindings are concrete and keyed by the declaring template's parameter names.
// allowVoid is true only for success results, never data positions.
func (b *Builder) Resolve(file *resolve.File, node syntax.TypeNode, parameters map[string]*Type, allowVoid bool) (result *Type, err error) {
	if b.failure != nil {
		return nil, b.failure
	}
	if b.graph.sealed {
		return nil, fmt.Errorf("type builder is already finished")
	}
	defer func() {
		if err != nil {
			b.failure = err
		}
	}()
	switch n := node.(type) {
	case *syntax.NamedType:
		if n.Name.Package == "" {
			if bound := parameters[n.Name.Name]; bound != nil {
				if len(n.Arguments) != 0 {
					return nil, fmt.Errorf("type parameter cannot take arguments")
				}
				if err = b.graph.data(bound); err != nil {
					return nil, err
				}
				return bound, nil
			}
		}
		symbol, e := file.Lookup(nil, n.Name, resolve.TypeUse)
		if e != nil {
			return nil, e
		}
		arguments := make([]*Type, len(n.Arguments))
		for i, arg := range n.Arguments {
			arguments[i], err = b.Resolve(file, arg, parameters, false)
			if err != nil {
				return nil, err
			}
		}
		result, err = b.instantiate(symbol, arguments)
	case *syntax.ArrayType:
		element, e := b.Resolve(file, n.Element, parameters, false)
		if e != nil {
			return nil, e
		}
		result, err = b.graph.array(element)
	case *syntax.CallableType:
		result, err = b.functional(file, Callable, n.Result, n.Inputs, n.Errors, parameters)
	case *syntax.ChoiceArmType:
		result, err = b.functional(file, ChoiceArm, n.Result, nil, n.Errors, parameters)
	default:
		return nil, fmt.Errorf("unknown annotation node %T", node)
	}
	if err != nil {
		return nil, err
	}
	if !allowVoid {
		if err = b.graph.data(result); err != nil {
			return nil, err
		}
	}
	return result, nil
}
func (b *Builder) functional(file *resolve.File, kind Kind, result syntax.TypeNode, inputs []syntax.TypeNode, bound syntax.ErrorBound, parameters map[string]*Type) (*Type, error) {
	r, err := b.Resolve(file, result, parameters, true)
	if err != nil {
		return nil, err
	}
	xs := make([]*Type, len(inputs))
	es := make([]*Type, len(bound.Types))
	for i, x := range inputs {
		xs[i], err = b.Resolve(file, x, parameters, false)
		if err != nil {
			return nil, err
		}
	}
	for i, e := range bound.Types {
		es[i], err = b.Resolve(file, e, parameters, false)
		if err != nil {
			return nil, err
		}
	}
	return b.graph.function(kind, r, xs, es)
}
func (b *Builder) instantiate(symbol *resolve.Symbol, arguments []*Type) (*Type, error) {
	if len(arguments) != len(symbol.Parameters) {
		return nil, fmt.Errorf("type argument arity for %s: want %d, got %d", symbol.ID, len(symbol.Parameters), len(arguments))
	}
	if symbol.Kind == resolve.Primitive {
		return b.graph.scalar(symbol.Name), nil
	}
	kind := Kind(symbol.Kind)
	n, err := b.graph.nominal(kind, symbol.ID, arguments)
	if err != nil {
		return nil, err
	}
	n.canonical = symbol.Name
	if symbol.Package != nil {
		n.canonical = symbol.Package.Name + "::" + symbol.Name
	}
	// A same-specialization reference may point back to a node whose fields are
	// being built. Its full shape is checked when the graph is sealed.
	if n.defined || b.building(n) {
		return n, nil
	}
	// This is an explicit implementation resource guard, not evidence that a
	// particular finite program has mathematically infinite specialization.
	if b.active[symbol.ID] >= 256 || len(b.graph.nodes) > 16384 {
		return nil, fmt.Errorf("concrete type expansion exceeds implementation limit at %s", symbol.ID)
	}
	b.active[symbol.ID]++
	defer func() { b.active[symbol.ID]-- }()
	b.pending[n] = true
	defer delete(b.pending, n)
	env := map[string]*Type{}
	for i, name := range symbol.Parameters {
		env[name] = arguments[i]
	}
	if symbol.Source == nil {
		return b.catalogueShape(symbol, n, env)
	}
	file := b.world.Files[symbol.Source]
	var sourceFields []syntax.Field
	var sourceAlternatives []syntax.TypeNode
	switch d := symbol.Declaration.(type) {
	case *syntax.RecordDecl:
		sourceFields = d.Fields
	case *syntax.ErrorDecl:
		sourceFields = d.Fields
	case *syntax.VariantDecl:
		sourceAlternatives = d.Alternatives
	default:
		return nil, fmt.Errorf("declaration is not nominal data: %s", symbol.ID)
	}
	fields := make([]Field, len(sourceFields))
	alternatives := make([]*Type, len(sourceAlternatives))
	for i, f := range sourceFields {
		typ, e := b.Resolve(file, f.Type, env, false)
		if e != nil {
			return nil, fmt.Errorf("%s field %s: %w", symbol.ID, f.Name.Text, e)
		}
		fields[i] = Field{Name: f.Name.Text, Type: typ}
	}
	for i, a := range sourceAlternatives {
		alternatives[i], err = b.Resolve(file, a, env, false)
		if err != nil {
			return nil, err
		}
	}
	if err = b.graph.define(n, fields, alternatives); err != nil {
		return nil, err
	}
	return n, nil
}
func (b *Builder) building(n *Type) bool {
	if b.pending == nil {
		b.pending = map[*Type]bool{}
	}
	return b.pending[n]
}

func (b *Builder) catalogueShape(symbol *resolve.Symbol, n *Type, env map[string]*Type) (*Type, error) {
	name := symbol.Name
	if symbol.Package != nil {
		name = symbol.Package.Name + "::" + name
	}
	var fields []catalogue.Field
	var leaves []string
	var parameters []catalogue.Parameter
	if d, ok := catalogue.Builtin().Type(name); ok {
		fields = d.Fields
		leaves = d.Leaves
		parameters = d.Parameters
		n.standardFailure = d.Identity == "can.prelude@1::standard_failure"
	} else if d, ok := catalogue.Builtin().Error(name); ok {
		fields = d.Fields
		parameters = d.Parameters
	} else {
		return nil, fmt.Errorf("unknown closed catalogue type %s", name)
	}
	for _, p := range parameters {
		b.constraints = append(b.constraints, constraint{env[p.Name], p.Constraint})
	}
	resolvedFields := make([]Field, len(fields))
	alternatives := make([]*Type, len(leaves))
	for i, f := range fields {
		t, err := b.descriptor(f.Type, env)
		if err != nil {
			return nil, err
		}
		resolvedFields[i] = Field{f.Name, t}
	}
	for i, leaf := range leaves {
		t, err := b.descriptor(leaf, env)
		if err != nil {
			return nil, err
		}
		alternatives[i] = t
	}
	if err := b.graph.define(n, resolvedFields, alternatives); err != nil {
		return nil, err
	}
	return n, nil
}
func (b *Builder) descriptor(text string, env map[string]*Type) (*Type, error) {
	d, err := catalogue.ParseDescriptor(text)
	if err != nil {
		return nil, err
	}
	var resolveDescriptor func(catalogue.Descriptor) (*Type, error)
	resolveDescriptor = func(d catalogue.Descriptor) (*Type, error) {
		if t := env[d.Name]; t != nil && len(d.Arguments) == 0 {
			return t, nil
		}
		args := make([]*Type, len(d.Arguments))
		for i, a := range d.Arguments {
			var err error
			args[i], err = resolveDescriptor(a)
			if err != nil {
				return nil, err
			}
		}
		if d.Name == "[]" {
			return b.graph.array(args[0])
		}
		symbol := b.catalogue[d.Name]
		if symbol == nil {
			return nil, fmt.Errorf("unknown catalogue descriptor %s", d.Name)
		}
		return b.instantiate(symbol, args)
	}
	return resolveDescriptor(d)
}

func (b *Builder) Finish() (model *Model, err error) {
	defer func() {
		if err != nil {
			b.failure = err
			b.graph.sealed = false
		}
	}()
	if b.failure != nil {
		return nil, b.failure
	}
	if err := b.graph.seal(); err != nil {
		return nil, err
	}
	for _, c := range b.constraints {
		accepted := false
		switch c.name {
		case "data":
			accepted = c.argument.kind != Void
		case "map_key":
			accepted = c.argument.kind == Primitive && (c.argument.declaration == "int" || c.argument.declaration == "bool" || c.argument.declaration == "str")
		case "failure_variant":
			accepted = c.argument.kind == Variant
		default:
			return nil, fmt.Errorf("unimplemented catalogue type constraint %q", c.name)
		}
		if !accepted {
			b.graph.sealed = false
			return nil, fmt.Errorf("type %s violates catalogue constraint %s", c.argument.id, c.name)
		}
	}
	return &Model{types: b.graph.ordered()}, nil
}

// SeedDeclarations includes every non-generic project nominal declaration.
// Generic bodies are instantiated at concrete uses by the specialization pass.
func (b *Builder) SeedDeclarations() error {
	var symbols []*resolve.Symbol
	for _, pkg := range b.world.Packages {
		if pkg.Source != nil {
			for _, s := range pkg.Scope.Symbols {
				if s.Kind == resolve.Record || s.Kind == resolve.Error || s.Kind == resolve.Variant {
					symbols = append(symbols, s)
				}
			}
		}
	}
	sort.Slice(symbols, func(i, j int) bool { return symbols[i].ID < symbols[j].ID })
	for _, s := range symbols {
		if len(s.Parameters) == 0 {
			if _, err := b.instantiate(s, nil); err != nil {
				b.failure = err
				return err
			}
		}
	}
	return nil
}
