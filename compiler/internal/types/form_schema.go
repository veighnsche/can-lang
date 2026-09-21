package types

import "fmt"

// FormSchema is the closed, shallow P10 projection; it does not broaden the
// shared JSON wire subset or accept arbitrary URLSearchParams records.
type FormSchema struct {
	Root   string      `json:"root"`
	Fields []FormField `json:"fields"`
}
type FormField struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	Some string `json:"some,omitempty"`
	None string `json:"none,omitempty"`
}

func Form(root *Type) (FormSchema, error) {
	if !Equal(root, root) || root.Kind() != Record {
		return FormSchema{}, fmt.Errorf("form schema at /: requires an ordinary record")
	}
	result := FormSchema{Root: root.Identity(), Fields: []FormField{}}
	for _, f := range root.Fields() {
		field := FormField{Name: f.Name}
		switch {
		case f.Type.Kind() == Primitive && f.Type.Declaration() == "str":
			field.Kind = "str"
		case f.Type.Kind() == Array && f.Type.Element().Kind() == Primitive && f.Type.Element().Declaration() == "str":
			field.Kind = "array"
		case f.Type.Kind() == Variant && f.Type.Declaration() == "can.std.option@1::value" && len(f.Type.Arguments()) == 1 && f.Type.Arguments()[0].Declaration() == "str":
			field.Kind = "optional"
			for _, leaf := range f.Type.Leaves() {
				switch leaf.Declaration() {
				case "can.std.option@1::some":
					field.Some = leaf.Identity()
				case "can.std.option@1::none":
					field.None = leaf.Identity()
				}
			}
			if field.Some == "" || field.None == "" {
				return FormSchema{}, fmt.Errorf("form schema at /%s: invalid option shape", f.Name)
			}
		default:
			return FormSchema{}, fmt.Errorf("form schema at /%s: requires str, str[] or option::value<str>", f.Name)
		}
		result.Fields = append(result.Fields, field)
	}
	return result, nil
}
