package check

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

func (c *regionChecker) install(scope bodyScope, name string, local *ir.Local) error {
	if local.ErrorAlias && name == "all_failed" && local.Type.Declaration() == "can.prelude@1::all_failed" {
		if scope.symbols.Symbols[name] != nil {
			return fmt.Errorf("duplicate error alias")
		}
		scope.symbols.Symbols[name] = &resolve.Symbol{ID: local.Identity, Name: name, Kind: resolve.Value}
		c.locals[local.Identity] = local.Type
		return nil
	}
	if err := scope.symbols.Define(&resolve.Symbol{ID: local.Identity, Name: name, Kind: resolve.Value}); err != nil {
		return err
	}
	c.locals[local.Identity] = local.Type
	return nil
}
func (c *regionChecker) pattern(node syntax.PatternNode, expected *types.Type, bindings map[string]*ir.Local) (*ir.Pattern, error) {
	if node == nil || !types.Equal(expected, expected) || expected.Kind() == types.Void {
		return nil, fmt.Errorf("invalid pattern type")
	}
	out := &ir.Pattern{Type: expected}
	bind := func(name string, typ *types.Type) (*ir.Local, error) {
		if name == "" || bindings[name] != nil {
			return nil, fmt.Errorf("duplicate pattern binding %s", name)
		}
		local := &ir.Local{Identity: c.identity(name), Type: typ}
		bindings[name] = local
		return local, nil
	}
	nominal := func(name syntax.QualifiedName, args []syntax.TypeNode) (*types.Type, error) {
		t, err := c.context.Type(&syntax.NamedType{Name: name, Arguments: args}, false)
		if err != nil {
			return nil, err
		}
		if !types.Assignable(t, expected) || (t.Kind() != types.Record && t.Kind() != types.Error && !(t.Kind() == types.Opaque && t.Declaration() == "can.prelude@1::standard_failure")) {
			return nil, fmt.Errorf("nominal pattern is outside matched type")
		}
		return t, nil
	}
	switch n := node.(type) {
	case *syntax.WildcardPattern:
		out.Kind = "any"
	case *syntax.NamePattern:
		var leaf *types.Type
		if expected.Kind() == types.Variant || expected.Kind() == types.Error || expected.Kind() == types.Record {
			candidates := expected.Leaves()
			if expected.Kind() != types.Variant {
				candidates = []*types.Type{expected}
			}
			if len(n.Types) == 0 && c.context.PatternName != nil {
				declaration, err := c.context.PatternName(n.Name)
				if err == nil {
					var matches []*types.Type
					for _, candidate := range candidates {
						if candidate.Declaration() == declaration {
							matches = append(matches, candidate)
						}
					}
					if len(matches) > 1 {
						alternatives := displayAlternatives(matches)
						return nil, c.locateCode(n.PatternSpan(), "CAN-CHECK-EXACT-SPECIALIZATION", fmt.Errorf("ambiguous concrete pattern leaf %q; %w: %s", n.Name.Name, errExactSpecialization, strings.Join(alternatives, ", ")))
					}
					if len(matches) == 1 {
						leaf = matches[0]
					}
				}
			} else if t, err := nominal(n.Name, n.Types); err == nil {
				leaf = t
			}
		}
		if leaf != nil {
			out.Kind = "nominal"
			out.Type = leaf
			out.Text = leaf.Identity()
			for _, f := range leaf.Fields() {
				out.Fields = append(out.Fields, f.Name)
				out.Children = append(out.Children, &ir.Pattern{Kind: "any", Type: f.Type})
			}
			if leaf.Kind() == types.Error {
				var err error
				out.Binding, err = bind(n.Name.Name, leaf)
				if err != nil {
					return nil, err
				}
				out.Binding.ErrorAlias = true
			}
		} else {
			spelling := n.Name.Name
			if n.Name.Package != "" {
				spelling = n.Name.Package + "::" + spelling
			}
			if n.Name.Package != "" || len(n.Types) != 0 {
				return nil, c.locateCode(n.PatternSpan(), "CAN-CHECK-UNKNOWN-PATTERN", fmt.Errorf("pattern %q does not name an admitted leaf", spelling))
			}
			return nil, c.locateCode(n.PatternSpan(), "CAN-CHECK-UNKNOWN-PATTERN", fmt.Errorf("pattern %q does not name an admitted leaf; write bind %s to capture the matched value", spelling, spelling))
		}
	case *syntax.BindPattern:
		out.Kind = "any"
		var err error
		out.Binding, err = bind(n.Name.Text, expected)
		if err != nil {
			return nil, err
		}
	case *syntax.ConstructorPattern:
		t, err := nominal(n.Name, n.Types)
		if err != nil {
			return nil, err
		}
		if t.Kind() != types.Record && t.Kind() != types.Error {
			return nil, fmt.Errorf("opaque value cannot be destructured")
		}
		fields := t.Fields()
		if len(n.Fields) != len(fields) {
			return nil, fmt.Errorf("constructor pattern arity mismatch")
		}
		out.Kind = "nominal"
		out.Type = t
		out.Text = t.Identity()
		for i, f := range fields {
			child, err := c.pattern(n.Fields[i], f.Type, bindings)
			if err != nil {
				return nil, err
			}
			out.Children = append(out.Children, child)
			out.Fields = append(out.Fields, f.Name)
		}
	case *syntax.ArrayPattern:
		if expected.Kind() != types.Array {
			return nil, fmt.Errorf("array pattern requires an array")
		}
		out.Kind = "array"
		for _, field := range n.Elements {
			child, err := c.pattern(field, expected.Element(), bindings)
			if err != nil {
				return nil, err
			}
			out.Children = append(out.Children, child)
		}
		if n.Rest != nil {
			var err error
			out.Rest, err = bind(n.Rest.Text, expected)
			if err != nil {
				return nil, err
			}
		}
	case *syntax.LiteralPattern:
		out.Kind = "literal"
		out.Text = n.Literal.Text
		if n.Negative {
			out.Text = "-" + out.Text
		}
		switch n.Literal.Kind {
		case syntax.Integer:
			if !scalar(expected, "int") {
				return nil, fmt.Errorf("integer pattern requires int")
			}
			x, ok := new(big.Int).SetString(out.Text, 0)
			if !ok {
				return nil, fmt.Errorf("invalid integer pattern")
			}
			out.Text = x.String()
		case syntax.Float:
			if !scalar(expected, "float") {
				return nil, fmt.Errorf("float pattern requires float")
			}
			f, err := strconv.ParseFloat(out.Text, 64)
			if err != nil || math.IsInf(f, 0) {
				return nil, fmt.Errorf("invalid float pattern")
			}
			out.Text = strconv.FormatFloat(f, 'g', -1, 64)
		case syntax.String:
			if !scalar(expected, "str") || n.Negative {
				return nil, fmt.Errorf("string pattern requires str")
			}
			out.Text = n.Literal.Value
		default:
			if !scalar(expected, "bool") || (out.Text != "true" && out.Text != "false") {
				return nil, fmt.Errorf("boolean pattern requires bool")
			}
		}
	case *syntax.RangePattern:
		if !scalar(expected, "int") {
			return nil, fmt.Errorf("range pattern requires int")
		}
		lo, err := c.pattern(n.Lower, expected, map[string]*ir.Local{})
		if err != nil {
			return nil, err
		}
		hi, err := c.pattern(n.Upper, expected, map[string]*ir.Local{})
		if err != nil {
			return nil, err
		}
		if n.Lower.Literal.Kind != syntax.Integer || n.Upper.Literal.Kind != syntax.Integer {
			return nil, fmt.Errorf("range bounds require integer literals")
		}
		a, _ := new(big.Int).SetString(lo.Text, 10)
		b, _ := new(big.Int).SetString(hi.Text, 10)
		if a.Cmp(b) > 0 {
			return nil, fmt.Errorf("reversed integer range")
		}
		out.Kind = "range"
		out.Text = lo.Text
		out.Upper = hi.Text
	case *syntax.AlternativePattern:
		if len(n.Alternatives) < 2 {
			return nil, fmt.Errorf("alternative pattern requires alternatives")
		}
		out.Kind = "or"
		var common map[string]*ir.Local
		for i, alternative := range n.Alternatives {
			names := map[string]*ir.Local{}
			child, err := c.pattern(alternative, expected, names)
			if err != nil {
				return nil, err
			}
			if i == 0 {
				common = names
			} else {
				if len(names) != len(common) {
					return nil, fmt.Errorf("alternatives must bind the same names")
				}
				ids := map[string]*ir.Local{}
				for name, local := range names {
					want := common[name]
					if want == nil || !types.Equal(local.Type, want.Type) {
						return nil, fmt.Errorf("alternative binding types differ")
					}
					ids[local.Identity] = want
				}
				rewritePatternBindings(child, ids)
			}
			out.Children = append(out.Children, child)
		}
		for name, local := range common {
			if bindings[name] != nil {
				return nil, fmt.Errorf("duplicate alternative binding")
			}
			bindings[name] = local
		}
	default:
		return nil, fmt.Errorf("unsupported pattern %T", node)
	}
	return out, nil
}
func patternNarrow(p *ir.Pattern) *types.Type {
	if p.Kind == "nominal" {
		return p.Type
	}
	if p.Kind == "or" && len(p.Children) > 0 {
		first := patternNarrow(p.Children[0])
		if first == nil {
			return nil
		}
		for _, child := range p.Children[1:] {
			next := patternNarrow(child)
			if next == nil || !types.Equal(first, next) {
				return nil
			}
		}
		return first
	}
	return nil
}
func rewritePatternBindings(p *ir.Pattern, ids map[string]*ir.Local) {
	if p.Binding != nil {
		p.Binding = ids[p.Binding.Identity]
	}
	if p.Rest != nil {
		p.Rest = ids[p.Rest.Identity]
	}
	for _, child := range p.Children {
		rewritePatternBindings(child, ids)
	}
}
