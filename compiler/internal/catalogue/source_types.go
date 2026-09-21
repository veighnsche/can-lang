package catalogue

import (
	"fmt"

	sourcefile "github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
)

// Concrete arguments use the one current source grammar. The private inventory
// descriptor grammar remains separate because it includes uppercase template
// placeholders; it is never a fallback for malformed concrete source types.
func parseConcreteType(text string) (typeRef, error) {
	file, err := sourcefile.New("<catalogue-type>", text)
	if err != nil {
		return typeRef{}, err
	}
	node, diagnostics := syntax.ParseType(file)
	if len(diagnostics) > 0 {
		return typeRef{}, fmt.Errorf("%s", diagnostics[0].Format(file))
	}
	return typeFromSyntax(node), nil
}

func typeFromSyntax(node syntax.TypeNode) typeRef {
	switch n := node.(type) {
	case *syntax.NamedType:
		name := n.Name.Name
		if n.Name.Package != "" {
			name = n.Name.Package + "::" + name
		}
		r := typeRef{name: name}
		for _, a := range n.Arguments {
			r.arguments = append(r.arguments, typeFromSyntax(a))
		}
		return r
	case *syntax.ArrayType:
		return typeRef{name: "[]", arguments: []typeRef{typeFromSyntax(n.Element)}}
	case *syntax.CallableType:
		result := typeFromSyntax(n.Result)
		r := typeRef{name: "callable", result: &result}
		for _, input := range n.Inputs {
			r.inputs = append(r.inputs, typeFromSyntax(input))
		}
		for _, bound := range n.Errors.Types {
			r.emits = append(r.emits, typeFromSyntax(bound))
		}
		return r
	case *syntax.ChoiceArmType:
		result := typeFromSyntax(n.Result)
		r := typeRef{name: "choice_arm", result: &result}
		for _, bound := range n.Errors.Types {
			r.emits = append(r.emits, typeFromSyntax(bound))
		}
		return r
	default:
		panic("unhandled current type node")
	}
}
