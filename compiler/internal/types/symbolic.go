package types

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/veighnsche/can-lang/compiler/internal/catalogue"
	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
)

// symbolicType preserves declaration and parameter identities without inventing
// a concrete type argument. It proves definite template duplicates; it cannot
// prove that different parameters stay disjoint at every specialization.
type symbolicType struct {
	key       string
	kind      string
	symbol    *resolve.Symbol
	arguments []*symbolicType
}

func symbolic(kind, name string, args []*symbolicType) *symbolicType {
	parts := []string{kind, name}
	for _, arg := range args {
		parts = append(parts, arg.key)
	}
	framed, _ := json.Marshal(parts)
	sum := sha256.Sum256(framed)
	return &symbolicType{key: hex.EncodeToString(sum[:]), kind: kind, arguments: args}
}
func parameterEnvironment(params map[string]bool) map[string]*symbolicType {
	env := map[string]*symbolicType{}
	for name := range params {
		env[name] = symbolic("parameter", name, nil)
	}
	return env
}
func (b *Builder) symbolicType(file *resolve.File, node syntax.TypeNode, env map[string]*symbolicType) (*symbolicType, error) {
	switch n := node.(type) {
	case *syntax.NamedType:
		if n.Name.Package == "" && env[n.Name.Name] != nil {
			if len(n.Arguments) != 0 {
				return nil, fmt.Errorf("type parameter cannot take arguments")
			}
			return env[n.Name.Name], nil
		}
		s, err := file.Lookup(nil, n.Name, resolve.TypeUse)
		if err != nil {
			return nil, err
		}
		if len(s.Parameters) != len(n.Arguments) {
			return nil, fmt.Errorf("symbolic type argument arity for %s", s.ID)
		}
		args := make([]*symbolicType, len(n.Arguments))
		for i, arg := range n.Arguments {
			args[i], err = b.symbolicType(file, arg, env)
			if err != nil {
				return nil, err
			}
		}
		out := symbolic(string(s.Kind), s.ID, args)
		out.symbol = s
		return out, nil
	case *syntax.ArrayType:
		element, err := b.symbolicType(file, n.Element, env)
		if err != nil {
			return nil, err
		}
		return symbolic("array", "", []*symbolicType{element}), nil
	case *syntax.CallableType:
		return b.symbolicFunction(file, "callable", n.Result, n.Inputs, n.Errors, env)
	case *syntax.ChoiceArmType:
		return b.symbolicFunction(file, "choice_arm", n.Result, nil, n.Errors, env)
	default:
		return nil, fmt.Errorf("unsupported symbolic annotation %T", node)
	}
}
func (b *Builder) symbolicFunction(file *resolve.File, kind string, result syntax.TypeNode, inputs []syntax.TypeNode, bound syntax.ErrorBound, env map[string]*symbolicType) (*symbolicType, error) {
	r, err := b.symbolicType(file, result, env)
	if err != nil {
		return nil, err
	}
	xs := make([]*symbolicType, len(inputs))
	for i, x := range inputs {
		xs[i], err = b.symbolicType(file, x, env)
		if err != nil {
			return nil, err
		}
	}
	es := make([]*symbolicType, len(bound.Types))
	for i, e := range bound.Types {
		es[i], err = b.symbolicType(file, e, env)
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(es, func(i, j int) bool { return es[i].key < es[j].key })
	return symbolic(kind, "", []*symbolicType{r, symbolic("inputs", "", xs), symbolic("errors", "", es)}), nil
}
func (b *Builder) symbolicDescriptor(d catalogue.Descriptor, env map[string]*symbolicType) (*symbolicType, error) {
	if env[d.Name] != nil && len(d.Arguments) == 0 {
		return env[d.Name], nil
	}
	args := make([]*symbolicType, len(d.Arguments))
	for i, a := range d.Arguments {
		var err error
		args[i], err = b.symbolicDescriptor(a, env)
		if err != nil {
			return nil, err
		}
	}
	if d.Name == "[]" {
		return symbolic("array", "", args), nil
	}
	s := b.catalogue[d.Name]
	if s == nil || len(args) != len(s.Parameters) {
		return nil, fmt.Errorf("invalid symbolic catalogue descriptor")
	}
	out := symbolic(string(s.Kind), s.ID, args)
	out.symbol = s
	return out, nil
}
func catalogueName(s *resolve.Symbol) string {
	if s.Package != nil {
		return s.Package.Name + "::" + s.Name
	}
	return s.Name
}

func (b *Builder) checkSymbolicLeaves(file *resolve.File, nodes []syntax.TypeNode, params map[string]bool) error {
	env := parameterEnvironment(params)
	seen := map[string]bool{}
	active := map[string]bool{}
	visits := 0
	var flatten func(*symbolicType, int) error
	flatten = func(t *symbolicType, depth int) error {
		visits++
		if depth > 256 || visits > 16384 {
			return fmt.Errorf("symbolic variant expansion exceeds implementation limit")
		}
		if t.kind != "variant" {
			if t.kind != "parameter" && t.kind != "record" && t.kind != "error" && (t.symbol == nil || t.symbol.ID != "can.prelude@1::standard_failure") {
				return fmt.Errorf("ineligible symbolic variant leaf")
			}
			if seen[t.key] {
				return fmt.Errorf("duplicate variant leaf in generic template")
			}
			seen[t.key] = true
			return nil
		}
		if active[t.key] {
			return fmt.Errorf("recursive variant-only alternatives in generic template")
		}
		active[t.key] = true
		defer delete(active, t.key)
		substitutions := map[string]*symbolicType{}
		for i, name := range t.symbol.Parameters {
			substitutions[name] = t.arguments[i]
		}
		if t.symbol.Source != nil {
			declaration := t.symbol.Declaration.(*syntax.VariantDecl)
			ownFile := b.world.Files[t.symbol.Source]
			for _, node := range declaration.Alternatives {
				child, err := b.symbolicType(ownFile, node, substitutions)
				if err != nil {
					return err
				}
				if err = flatten(child, depth+1); err != nil {
					return err
				}
			}
		} else {
			declaration, ok := catalogue.Builtin().Type(catalogueName(t.symbol))
			if !ok {
				return fmt.Errorf("missing catalogue variant")
			}
			for _, text := range declaration.Leaves {
				descriptor, err := catalogue.ParseDescriptor(text)
				if err != nil {
					return err
				}
				child, err := b.symbolicDescriptor(descriptor, substitutions)
				if err != nil {
					return err
				}
				if err = flatten(child, depth+1); err != nil {
					return err
				}
			}
		}
		return nil
	}
	for _, node := range nodes {
		t, err := b.symbolicType(file, node, env)
		if err != nil {
			return err
		}
		if err = flatten(t, 0); err != nil {
			return err
		}
	}
	return nil
}

// Catalogue constraints are per argument. A dependent sibling never hides a
// known forbidden key, and box<T> is definitely a record even before T is known.
func (b *Builder) templateCatalogueConstraints(file *resolve.File, symbol *resolve.Symbol, args []syntax.TypeNode, params map[string]bool) error {
	if symbol.Source != nil {
		return nil
	}
	var parameters []catalogue.Parameter
	if d, ok := catalogue.Builtin().Type(catalogueName(symbol)); ok {
		parameters = d.Parameters
	} else if d, ok := catalogue.Builtin().Error(catalogueName(symbol)); ok {
		parameters = d.Parameters
	}
	for i, p := range parameters {
		node := args[i]
		if n, ok := node.(*syntax.NamedType); ok && n.Name.Package == "" && params[n.Name.Name] {
			continue
		}
		accepted := false
		switch p.Constraint {
		case "data":
			accepted = true // template already rejects void and invalid children.
		case "map_key", "failure_variant":
			if n, ok := node.(*syntax.NamedType); ok {
				s, err := file.Lookup(nil, n.Name, resolve.TypeUse)
				if err != nil {
					return err
				}
				if p.Constraint == "map_key" {
					accepted = s.Kind == resolve.Primitive && (s.Name == "int" || s.Name == "bool" || s.Name == "str")
				} else {
					accepted = s.Kind == resolve.Variant
				}
			}
		default:
			return fmt.Errorf("unsupported template catalogue constraint %s", p.Constraint)
		}
		if !accepted {
			return fmt.Errorf("template argument violates catalogue constraint %s", p.Constraint)
		}
	}
	return nil
}

// Parameters are potentially inhabited, not guessed concrete types. If the
// least fixed point still cannot inhabit a template with that maximum allowance,
// no substitution can rescue its mandatory record cycle.
func (b *Builder) checkSymbolicInhabitation(symbol *resolve.Symbol, params map[string]bool) error {
	env := parameterEnvironment(params)
	args := make([]*symbolicType, len(symbol.Parameters))
	for i, name := range symbol.Parameters {
		args[i] = env[name]
	}
	root := symbolic(string(symbol.Kind), symbol.ID, args)
	root.symbol = symbol
	type shape struct {
		node      *symbolicType
		children  []string
		inhabited bool
	}
	shapes := map[string]*shape{}
	active := map[string]int{}
	var build func(*symbolicType) error
	build = func(node *symbolicType) error {
		if shapes[node.key] != nil {
			return nil
		}
		if len(shapes) >= 16384 {
			return fmt.Errorf("symbolic type expansion exceeds implementation limit")
		}
		current := &shape{node: node}
		shapes[node.key] = current
		switch node.kind {
		case "parameter", "primitive", "array", "callable", "choice_arm", "opaque":
			current.inhabited = true
			return nil
		}
		s := node.symbol
		if active[s.ID] >= 256 {
			return fmt.Errorf("symbolic type expansion exceeds implementation limit")
		}
		active[s.ID]++
		defer func() { active[s.ID]-- }()
		substitutions := map[string]*symbolicType{}
		for i, name := range s.Parameters {
			substitutions[name] = node.arguments[i]
		}
		add := func(child *symbolicType) error {
			current.children = append(current.children, child.key)
			return build(child)
		}
		if s.Source != nil {
			var fields []syntax.Field
			var alternatives []syntax.TypeNode
			switch d := s.Declaration.(type) {
			case *syntax.RecordDecl:
				fields = d.Fields
			case *syntax.ErrorDecl:
				fields = d.Fields
			case *syntax.VariantDecl:
				alternatives = d.Alternatives
			default:
				return fmt.Errorf("invalid symbolic data declaration")
			}
			ownFile := b.world.Files[s.Source]
			for _, field := range fields {
				child, err := b.symbolicType(ownFile, field.Type, substitutions)
				if err != nil {
					return err
				}
				if err = add(child); err != nil {
					return err
				}
			}
			for _, node := range alternatives {
				child, err := b.symbolicType(ownFile, node, substitutions)
				if err != nil {
					return err
				}
				if err = add(child); err != nil {
					return err
				}
			}
		} else {
			var descriptors []string
			if d, ok := catalogue.Builtin().Type(catalogueName(s)); ok {
				for _, f := range d.Fields {
					descriptors = append(descriptors, f.Type)
				}
				descriptors = append(descriptors, d.Leaves...)
			} else if d, ok := catalogue.Builtin().Error(catalogueName(s)); ok {
				for _, f := range d.Fields {
					descriptors = append(descriptors, f.Type)
				}
			} else {
				return fmt.Errorf("unknown symbolic catalogue type")
			}
			for _, text := range descriptors {
				descriptor, err := catalogue.ParseDescriptor(text)
				if err != nil {
					return err
				}
				child, err := b.symbolicDescriptor(descriptor, substitutions)
				if err != nil {
					return err
				}
				if err = add(child); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := build(root); err != nil {
		return err
	}
	for changed := true; changed; {
		changed = false
		for _, current := range shapes {
			if current.inhabited {
				continue
			}
			possible := current.node.kind != "variant"
			for _, child := range current.children {
				if current.node.kind == "variant" {
					possible = possible || shapes[child].inhabited
				} else {
					possible = possible && shapes[child].inhabited
				}
			}
			if possible {
				current.inhabited = true
				changed = true
			}
		}
	}
	if !shapes[root.key].inhabited {
		return fmt.Errorf("generic type has no possible finite inhabitant: %s", symbol.ID)
	}
	return nil
}
