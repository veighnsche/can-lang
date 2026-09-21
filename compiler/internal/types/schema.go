package types

import (
	"fmt"
	"sort"
	"strings"
)

// CodecSchema is a finite graph, not expanded JSON Schema or a serialized value
// tree. All referenced nodes come from checked sealed concrete contracts.
type CodecSchema struct {
	Root  string      `json:"root"`
	Nodes []CodecNode `json:"nodes"`
}
type CodecField struct {
	Name string `json:"name"`
	Type string `json:"type"`
}
type CodecNode struct {
	Identity string       `json:"identity"`
	Kind     Kind         `json:"kind"`
	Name     string       `json:"name"`
	Element  string       `json:"element,omitempty"`
	Fields   []CodecField `json:"fields,omitempty"`
	Leaves   []string     `json:"leaves,omitempty"`
}

func CanonicalName(t *Type) string {
	if t.kind == Array {
		return CanonicalName(t.element) + "[]"
	}
	name := t.canonical
	if name == "" {
		name = t.declaration
	}
	if len(t.arguments) > 0 {
		args := make([]string, len(t.arguments))
		for i, arg := range t.arguments {
			args[i] = CanonicalName(arg)
		}
		name += "<" + strings.Join(args, ",") + ">"
	}
	return name
}
func Schema(t *Type) (CodecSchema, error) {
	if !Equal(t, t) {
		return CodecSchema{}, fmt.Errorf("codec schema requires a sealed concrete type")
	}
	out := CodecSchema{Root: t.id}
	seen := map[string]bool{}
	var visit func(*Type) error
	visit = func(t *Type) error {
		if seen[t.id] {
			return nil
		}
		seen[t.id] = true
		node := CodecNode{Identity: t.id, Kind: t.kind, Name: CanonicalName(t)}
		switch t.kind {
		case Primitive:
			switch t.declaration {
			case "str", "int", "float", "bool":
			default:
				return fmt.Errorf("unsupported codec primitive %s", t.declaration)
			}
		case Array:
			node.Element = t.element.id
			if err := visit(t.element); err != nil {
				return err
			}
		case Record, Error:
			for _, field := range t.fields {
				node.Fields = append(node.Fields, CodecField{field.Name, field.Type.id})
				if err := visit(field.Type); err != nil {
					return err
				}
			}
		case Variant:
			tags := map[string]bool{}
			for _, leaf := range t.Leaves() {
				tag := CanonicalName(leaf)
				if tags[tag] {
					return fmt.Errorf("ambiguous codec variant tag %s", tag)
				}
				tags[tag] = true
				node.Leaves = append(node.Leaves, leaf.id)
				if err := visit(leaf); err != nil {
					return err
				}
			}
		default:
			return fmt.Errorf("type %s (%s) is not codec-admissible", CanonicalName(t), t.kind)
		}
		out.Nodes = append(out.Nodes, node)
		return nil
	}
	if err := visit(t); err != nil {
		return CodecSchema{}, err
	}
	sort.Slice(out.Nodes, func(i, j int) bool { return out.Nodes[i].Identity < out.Nodes[j].Identity })
	return out, nil
}
