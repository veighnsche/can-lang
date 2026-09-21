package check

import (
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// A direct recursive substitution retaining every corresponding parameter and
// placing at least one under a type constructor grows on every application.
// Neither concrete size nor recurrence of a source location proves that fact.
// In particular, field projection can traverse a finite nominal chain.
func (c *programChecker) provesRecursiveGrowth(symbol *resolve.Symbol, arguments []*types.Type, substitutions []syntax.TypeNode) bool {
	if c.current == nil || c.current.Symbol.ID != symbol.ID || len(substitutions) != len(symbol.Parameters) || !growingArguments(c.current.TypeArguments, arguments) {
		return false
	}
	strict := false
	for i, name := range symbol.Parameters {
		node := substitutions[i]
		if node == nil || !containsParameter(node, name) {
			return false
		}
		// Inferred evidence must describe the actual invariant solution, not a
		// contextual widening or a different solution from another constraint.
		actual, err := c.annotation(c.world.Files[symbol.Source], node, false)
		if err != nil || !types.Equal(actual, arguments[i]) {
			return false
		}
		strict = strict || !isParameter(node, name)
	}
	return strict
}

func isParameter(node syntax.TypeNode, name string) bool {
	n, ok := node.(*syntax.NamedType)
	return ok && n.Name.Package == "" && n.Name.Name == name && len(n.Arguments) == 0
}

func containsParameter(node syntax.TypeNode, name string) bool {
	if isParameter(node, name) {
		return true
	}
	switch n := node.(type) {
	case *syntax.NamedType:
		for _, arg := range n.Arguments {
			if containsParameter(arg, name) {
				return true
			}
		}
	case *syntax.ArrayType:
		return containsParameter(n.Element, name)
	case *syntax.CallableType:
		if containsParameter(n.Result, name) {
			return true
		}
		for _, input := range n.Inputs {
			if containsParameter(input, name) {
				return true
			}
		}
	}
	return false
}

// Recover only source-backed substitutions. Unsupported expressions are unknown,
// never a guess based on their concrete result type. Ordinary checking has
// already established exact inference; this evidence is solely for recursion.
func (c *programChecker) inferredSubstitutions(file *resolve.File, scope *resolve.Scope, symbol *resolve.Symbol, constraints []argumentConstraint) []syntax.TypeNode {
	if c.current == nil || c.current.Symbol.ID != symbol.ID {
		return nil
	}
	result := make([]syntax.TypeNode, len(symbol.Parameters))
	var match func(syntax.TypeNode, syntax.TypeNode)
	match = func(pattern, actual syntax.TypeNode) {
		if actual == nil {
			return
		}
		for i, name := range symbol.Parameters {
			if isParameter(pattern, name) {
				result[i] = actual
				return
			}
		}
		switch p := pattern.(type) {
		case *syntax.ArrayType:
			if a, ok := actual.(*syntax.ArrayType); ok {
				match(p.Element, a.Element)
			}
		case *syntax.NamedType:
			if a, ok := actual.(*syntax.NamedType); ok && p.Name.Package == a.Name.Package && p.Name.Name == a.Name.Name && len(p.Arguments) == len(a.Arguments) {
				for i := range p.Arguments {
					match(p.Arguments[i], a.Arguments[i])
				}
			}
		}
	}
	for _, constraint := range constraints {
		match(constraint.annotation, c.sourceExpressionType(file, scope, constraint.expression))
	}
	return result
}

func (c *programChecker) sourceExpressionType(file *resolve.File, scope *resolve.Scope, expression syntax.Expr) syntax.TypeNode {
	switch n := expression.(type) {
	case *syntax.GroupExpr:
		return c.sourceExpressionType(file, scope, n.Value)
	case *syntax.NameExpr:
		symbol, err := file.Lookup(scope, n.Name, resolve.ValueUse)
		if err == nil && strings.HasPrefix(symbol.ID, c.current.Symbol.ID+"/input/") {
			declaration := c.current.Symbol.Declaration.(*syntax.FunctionDecl)
			for _, input := range declaration.Inputs {
				if input.Name.Text == symbol.Name && input.Variadic {
					return &syntax.ArrayType{Element: symbol.Type}
				}
			}
			return symbol.Type
		}
	case *syntax.ArrayExpr:
		for _, element := range n.Elements {
			if node := c.sourceExpressionType(file, scope, element.Value); node != nil {
				if element.Spread {
					if _, ok := node.(*syntax.ArrayType); ok {
						return node
					}
				} else {
					return &syntax.ArrayType{Element: node}
				}
			}
		}
	}
	return nil
}
