package types

import (
	"fmt"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
)

// InferencePattern is resolved declaration structure with data-parameter holes.
// It is not a Type and can never serve as compatibility or emission evidence.
type InferencePattern struct {
	parameter   string
	kind        Kind
	declaration string
	arguments   []*InferencePattern
	element     *InferencePattern
	result      *InferencePattern
	inputs      []*InferencePattern
}

func Pattern(file *resolve.File, node syntax.TypeNode, parameters []string) (*InferencePattern, error) {
	parameterSet := map[string]bool{}
	for _, name := range parameters {
		if name == "" || parameterSet[name] {
			return nil, fmt.Errorf("invalid inference parameter %q", name)
		}
		parameterSet[name] = true
	}
	var build func(syntax.TypeNode) (*InferencePattern, error)
	build = func(node syntax.TypeNode) (*InferencePattern, error) {
		p := &InferencePattern{}
		switch n := node.(type) {
		case *syntax.NamedType:
			if n.Name.Package == "" && parameterSet[n.Name.Name] {
				if len(n.Arguments) != 0 {
					return nil, fmt.Errorf("type parameter cannot take arguments")
				}
				p.parameter = n.Name.Name
				return p, nil
			}
			symbol, err := file.Lookup(nil, n.Name, resolve.TypeUse)
			if err != nil {
				return nil, err
			}
			if len(n.Arguments) != len(symbol.Parameters) {
				return nil, fmt.Errorf("inference template arity for %s", symbol.ID)
			}
			p.kind, p.declaration = Kind(symbol.Kind), symbol.ID
			if symbol.Kind == resolve.Primitive {
				p.declaration = symbol.Name
				if symbol.Name == "void" {
					p.kind = Void
				}
			}
			for _, arg := range n.Arguments {
				child, err := build(arg)
				if err != nil {
					return nil, err
				}
				p.arguments = append(p.arguments, child)
			}
		case *syntax.ArrayType:
			p.kind = Array
			var err error
			p.element, err = build(n.Element)
			if err != nil {
				return nil, err
			}
		case *syntax.CallableType:
			p.kind = Callable
			var err error
			p.result, err = build(n.Result)
			if err != nil {
				return nil, err
			}
			for _, input := range n.Inputs {
				child, err := build(input)
				if err != nil {
					return nil, err
				}
				p.inputs = append(p.inputs, child)
			}
		case *syntax.ChoiceArmType:
			p.kind = ChoiceArm
			var err error
			p.result, err = build(n.Result)
			if err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("unsupported inference template %T", node)
		}
		return p, nil
	}
	return build(node)
}

// Inference solves only equalities against sealed concrete types. There is no
// candidate search, union construction, numeric conversion or type evaluation.
// Callable error compatibility remains a separate checked-contract obligation:
// deriving result/input parameters never invents an error bound.
type Inference struct {
	parameters []string
	bindings   map[string]*Type
}

func (i *Inference) Bindings() map[string]*Type {
	result := map[string]*Type{}
	for name, typ := range i.bindings {
		if typ != nil {
			result[name] = typ
		}
	}
	return result
}
func (p *InferencePattern) HasParameters() bool {
	if p == nil {
		return false
	}
	if p.parameter != "" {
		return true
	}
	for _, child := range append(append([]*InferencePattern{p.element, p.result}, p.arguments...), p.inputs...) {
		if child.HasParameters() {
			return true
		}
	}
	return false
}
func (i *Inference) Resolved(p *InferencePattern) bool {
	if p == nil {
		return true
	}
	if p.parameter != "" {
		return i.bindings[p.parameter] != nil
	}
	for _, child := range append(append([]*InferencePattern{p.element, p.result}, p.arguments...), p.inputs...) {
		if !i.Resolved(child) {
			return false
		}
	}
	return true
}

func NewInference(parameters []string) (*Inference, error) {
	i := &Inference{parameters: append([]string(nil), parameters...), bindings: map[string]*Type{}}
	for _, name := range parameters {
		if _, ok := i.bindings[name]; ok || name == "" {
			return nil, fmt.Errorf("invalid inference parameter %q", name)
		}
		i.bindings[name] = nil
	}
	return i, nil
}

// Constrain is transactional: a conflict does not leave earlier holes from the
// same structural constraint bound. The owning checker retains source locations
// for diagnostics and validates assignability after concrete instantiation.
func (i *Inference) Constrain(pattern *InferencePattern, actual *Type) error {
	bindings := map[string]*Type{}
	for name, value := range i.bindings {
		bindings[name] = value
	}
	var visit func(*InferencePattern, *Type) error
	visit = func(p *InferencePattern, t *Type) error {
		if p == nil || !Equal(t, t) {
			return fmt.Errorf("inference requires a pattern and sealed concrete evidence")
		}
		if p.parameter != "" {
			prior, exists := bindings[p.parameter]
			if !exists {
				return fmt.Errorf("unknown inference parameter %s", p.parameter)
			}
			if t.kind == Void {
				return fmt.Errorf("void is not a generic data argument")
			}
			if prior != nil && !Equal(prior, t) {
				return fmt.Errorf("conflicting inference for %s: %s and %s", p.parameter, CanonicalName(prior), CanonicalName(t))
			}
			bindings[p.parameter] = t
			return nil
		}
		if p.kind != t.kind {
			return fmt.Errorf("inference shape mismatch: want %s, got %s", p.kind, t.kind)
		}
		switch p.kind {
		case Array:
			return visit(p.element, t.element)
		case Callable, ChoiceArm:
			if len(p.inputs) != len(t.inputs) {
				return fmt.Errorf("inference callable arity mismatch")
			}
			if err := visit(p.result, t.result); err != nil {
				return err
			}
			for index, input := range p.inputs {
				if err := visit(input, t.inputs[index]); err != nil {
					return err
				}
			}
		default:
			if p.declaration != t.declaration || len(p.arguments) != len(t.arguments) {
				return fmt.Errorf("inference nominal identity mismatch")
			}
			for index, arg := range p.arguments {
				if err := visit(arg, t.arguments[index]); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := visit(pattern, actual); err != nil {
		return err
	}
	i.bindings = bindings
	return nil
}

func (i *Inference) Arguments() ([]*Type, error) {
	var missing []string
	args := make([]*Type, len(i.parameters))
	for index, name := range i.parameters {
		args[index] = i.bindings[name]
		if args[index] == nil {
			missing = append(missing, name)
		}
	}
	if len(missing) != 0 {
		return nil, fmt.Errorf("ambiguous type arguments: %s; supply explicit types", strings.Join(missing, ", "))
	}
	return args, nil
}
