package types

import (
	"fmt"
	"sort"

	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
)

// CheckDeclarations validates data declarations and explicit signature/value
// annotations without evaluating expressions. Expression semantics, generic body
// specialization and completion checking are subsequent compiler passes.
func CheckDeclarations(world *resolve.World) (*Model, error) {
	b := NewBuilder(world)
	if err := b.SeedDeclarations(); err != nil {
		return nil, err
	}
	var files []*resolve.File
	for _, f := range world.Files {
		files = append(files, f)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Source.ID < files[j].Source.ID })
	for _, file := range files {
		for _, declaration := range file.Source.Syntax.Declarations {
			params := map[string]bool{}
			var fields []syntax.Field
			var result syntax.TypeNode
			var bound *syntax.ErrorBound
			var alternatives []syntax.TypeNode
			var nominalName string
			switch d := declaration.(type) {
			case *syntax.ConnectionDecl:
				continue
			case *syntax.ChoiceArmDecl:
				result = d.Result
				bound = &d.Errors
			case *syntax.RecordDecl:
				nominalName = d.Name.Text
				for _, p := range d.Parameters {
					params[p.Text] = true
				}
				fields = d.Fields
			case *syntax.ErrorDecl:
				nominalName = d.Name.Text
				for _, p := range d.Parameters {
					params[p.Text] = true
				}
				fields = d.Fields
			case *syntax.VariantDecl:
				nominalName = d.Name.Text
				for _, p := range d.Parameters {
					params[p.Text] = true
				}
				alternatives = d.Alternatives
			case *syntax.ValueDecl:
				fields = []syntax.Field{{Type: d.Binding.Type, Name: d.Binding.Name}}
			case *syntax.FunctionDecl:
				for _, p := range d.Parameters {
					params[p.Text] = true
				}
				result = d.Result
				bound = &d.Errors
				if d.Receiver != nil {
					fields = append(fields, *d.Receiver)
				}
				for _, input := range d.Inputs {
					fields = append(fields, input.Field)
				}
			case *syntax.FixtureDecl:
				// The target contract resolves at fixture checking; only
				// the parameter annotations are templated here.
				fields = d.Given
			case *syntax.WrapDecl:
				// Inherited annotations resolve in the root's declaring
				// file. The bound is calculated from checked handlers in
				// a later pass, so nothing is templated for it here.
				symbol := file.Package.Scope.Symbols[d.Name.Text]
				if symbol == nil {
					return nil, fmt.Errorf("wrapper %q has no declared symbol", d.Name.Text)
				}
				_, root, header, err := world.WrapperOrigin(file, symbol)
				if err != nil {
					return nil, err
				}
				origin := world.Files[root.Source]
				if origin == nil {
					return nil, fmt.Errorf("wrapper root %s has no declaring file", root.ID)
				}
				if _, err := b.template(origin, header.Result, params, true); err != nil {
					return nil, err
				}
				wrapFields := []syntax.Field{}
				for _, input := range header.Inputs {
					wrapFields = append(wrapFields, input.Field)
				}
				if judge, ok := root.Declaration.(*syntax.JudgeDecl); ok {
					wrapFields = append(wrapFields, judge.State...)
				}
				for _, f := range wrapFields {
					if _, err := b.template(origin, f.Type, params, false); err != nil {
						return nil, fmt.Errorf("%s annotation %s: %w", file.Source.ID, f.Name.Text, err)
					}
				}
				continue
			default:
				header := syntax.NativeSignature(declaration)
				if header == nil {
					return nil, fmt.Errorf("unhandled declaration %T", declaration)
				}
				result = header.Result
				bound = &header.Errors
				for _, input := range header.Inputs {
					fields = append(fields, input.Field)
				}
				switch native := declaration.(type) {
				case *syntax.JudgeDecl:
					fields = append(fields, native.State...)
				case *syntax.LLMDecl:
					fields = append(fields, native.State...)
				}
			}
			for _, f := range fields {
				if _, err := b.template(file, f.Type, params, false); err != nil {
					return nil, fmt.Errorf("%s annotation %s: %w", file.Source.ID, f.Name.Text, err)
				}
			}
			if len(alternatives) != 0 {
				if err := b.checkSymbolicLeaves(file, alternatives, params); err != nil {
					return nil, err
				}
			}
			for _, a := range alternatives {
				_, err := b.template(file, a, params, false)
				if err != nil {
					return nil, err
				}
				if n, ok := a.(*syntax.NamedType); ok {
					if n.Name.Package == "" && params[n.Name.Name] {
						continue
					}
					s, err := file.Lookup(nil, n.Name, resolve.TypeUse)
					if err != nil {
						return nil, err
					}
					if s.Kind != resolve.Record && s.Kind != resolve.Error && s.Kind != resolve.Variant && s.ID != "can.prelude@1::standard_failure" {
						return nil, fmt.Errorf("ineligible variant alternative %s", s.ID)
					}
				} else {
					return nil, fmt.Errorf("variant alternatives require nominal leaves")
				}
			}
			if nominalName != "" && len(params) != 0 {
				if err := b.checkSymbolicInhabitation(file.Package.Scope.Symbols[nominalName], params); err != nil {
					return nil, err
				}
			}
			if result != nil {
				if _, err := b.template(file, result, params, true); err != nil {
					return nil, err
				}
			}
			if bound != nil {
				if err := b.templateBound(file, *bound, params); err != nil {
					return nil, err
				}
			}
		}
	}
	return b.Finish()
}

// template checks parameter-independent errors even when a generic declaration
// is not instantiated. It resolves every concrete subtree, but never substitutes
// a guessed int/any type for an unconstrained parameter.
func (b *Builder) template(file *resolve.File, node syntax.TypeNode, params map[string]bool, allowVoid bool) (bool, error) {
	concrete := true
	switch n := node.(type) {
	case *syntax.NamedType:
		if n.Name.Package == "" && params[n.Name.Name] {
			if len(n.Arguments) != 0 {
				return false, fmt.Errorf("type parameter cannot take arguments")
			}
			return false, nil
		}
		s, err := file.Lookup(nil, n.Name, resolve.TypeUse)
		if err != nil {
			return false, err
		}
		if len(n.Arguments) != len(s.Parameters) {
			return false, fmt.Errorf("type argument arity for %s", s.ID)
		}
		for _, a := range n.Arguments {
			known, err := b.template(file, a, params, false)
			if err != nil {
				return false, err
			}
			concrete = concrete && known
		}
		if err := b.templateCatalogueConstraints(file, s, n.Arguments, params); err != nil {
			return false, err
		}
	case *syntax.ArrayType:
		known, err := b.template(file, n.Element, params, false)
		if err != nil {
			return false, err
		}
		concrete = known
	case *syntax.CallableType:
		known, err := b.template(file, n.Result, params, true)
		if err != nil {
			return false, err
		}
		concrete = known
		for _, input := range n.Inputs {
			known, err = b.template(file, input, params, false)
			if err != nil {
				return false, err
			}
			concrete = concrete && known
		}
		if err = b.templateBound(file, n.Errors, params); err != nil {
			return false, err
		}
		for _, e := range n.Errors.Types {
			known, err = b.template(file, e, params, false)
			if err != nil {
				return false, err
			}
			concrete = concrete && known
		}
	case *syntax.ChoiceArmType:
		known, err := b.template(file, n.Result, params, true)
		if err != nil {
			return false, err
		}
		concrete = known
		if err = b.templateBound(file, n.Errors, params); err != nil {
			return false, err
		}
		for _, e := range n.Errors.Types {
			known, err = b.template(file, e, params, false)
			if err != nil {
				return false, err
			}
			concrete = concrete && known
		}
	default:
		return false, fmt.Errorf("unknown annotation %T", node)
	}
	if concrete {
		if _, err := b.Resolve(file, node, nil, allowVoid); err != nil {
			return false, err
		}
	}
	return concrete, nil
}
func (b *Builder) templateBound(file *resolve.File, bound syntax.ErrorBound, params map[string]bool) error {
	seen := map[string]bool{}
	env := parameterEnvironment(params)
	for _, node := range bound.Types {
		symbolic, err := b.symbolicType(file, node, env)
		if err != nil {
			return err
		}
		if seen[symbolic.key] {
			return fmt.Errorf("duplicate error in bound")
		}
		seen[symbolic.key] = true
		known, err := b.template(file, node, params, false)
		if err != nil {
			return err
		}
		if known {
			typ, err := b.Resolve(file, node, nil, false)
			if err != nil {
				return err
			}
			if typ.kind != Error {
				return fmt.Errorf("emits requires nominal errors")
			}
		} else if n, ok := node.(*syntax.NamedType); ok {
			if n.Name.Package == "" && params[n.Name.Name] {
				continue
			}
			s, err := file.Lookup(nil, n.Name, resolve.ErrorUse)
			if err != nil {
				return err
			}
			_ = s
		} else {
			return fmt.Errorf("emits requires nominal errors")
		}
	}
	return nil
}

// MergeGeneratedFields is shared with native declaration checking: option/level
// fields precede explicit metadata, and the two lists share one collision domain.
// There are no implicit confidence/score names reserved on ordinary records.
func MergeGeneratedFields(fields, metadata []Field) ([]Field, error) {
	out := append(append([]Field(nil), fields...), metadata...)
	seen := map[string]bool{}
	for i, f := range out {
		if f.Name == "" || seen[f.Name] {
			return nil, fmt.Errorf("duplicate generated field/metadata name %q", f.Name)
		}
		seen[f.Name] = true
		if f.Type == nil || !Equal(f.Type, f.Type) || f.Type.kind == Void {
			return nil, fmt.Errorf("generated field requires checked data")
		}
		if i >= len(fields) && (f.Type.kind != Primitive || f.Type.declaration != "float") {
			return nil, fmt.Errorf("generated metadata field must be float")
		}
	}
	return out, nil
}
