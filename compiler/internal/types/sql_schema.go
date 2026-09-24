package types

import "fmt"

// SQLSchema is the closed scalar/option projection shared by SQL descriptor
// checking and query specialization: bool, signed-64-bit int, finite float,
// scalar str, owned bytes, or one optional layer of these. It rejects
// arrays, JSON, timestamps, decimals, arbitrary variants, and nested
// options. Field order is the record declaration order.
type SQLSchema struct {
	Root   string     `json:"root"`
	Fields []SQLField `json:"fields"`
}
type SQLField struct {
	Name  string `json:"name"`
	Kind  string `json:"kind"`
	Inner string `json:"inner,omitempty"`
	Some  string `json:"some,omitempty"`
	None  string `json:"none,omitempty"`
}

const (
	sqlOptionValue = "can.std.option@1::value"
	sqlOptionSome  = "can.std.option@1::some"
	sqlOptionNone  = "can.std.option@1::none"
	sqlBytesBuffer = "can.std.bytes@1::buffer"
)

// SQLFieldOf projects one field type onto the SQL scalar cover. Range,
// finiteness, and Unicode validity are per-value runtime checks; this
// projection admits only the shapes the driver boundary can carry.
func SQLFieldOf(field *Type) (SQLField, error) {
	if field.Owner() {
		return SQLField{}, fmt.Errorf("owner record %s needs an explicit wire type plus the owner factory", CanonicalName(field))
	}
	flat := func(t *Type) (string, bool) {
		if t.Kind() == Primitive {
			switch t.Declaration() {
			case "bool", "int", "float", "str":
				return t.Declaration(), true
			}
			return "", false
		}
		if t.Kind() == Opaque && t.Declaration() == sqlBytesBuffer {
			return "bytes", true
		}
		return "", false
	}
	if kind, ok := flat(field); ok {
		return SQLField{Kind: kind}, nil
	}
	if field.Kind() == Variant && field.Declaration() == sqlOptionValue && len(field.Arguments()) == 1 {
		inner, ok := flat(field.Arguments()[0])
		if !ok {
			return SQLField{}, fmt.Errorf("option layer requires bool, int, float, str, or bytes")
		}
		out := SQLField{Kind: "option", Inner: inner}
		for _, leaf := range field.Leaves() {
			switch leaf.Declaration() {
			case sqlOptionSome:
				out.Some = leaf.Identity()
			case sqlOptionNone:
				out.None = leaf.Identity()
			}
		}
		if out.Some == "" || out.None == "" {
			return SQLField{}, fmt.Errorf("invalid option shape")
		}
		return out, nil
	}
	return SQLField{}, fmt.Errorf("requires bool, int, float, str, bytes, or one option layer")
}

// SQLSchemaOf projects an ordinary record onto the SQL scalar cover,
// preserving declaration order for parameter binding and row decoding.
func SQLSchemaOf(root *Type) (SQLSchema, error) {
	if !Equal(root, root) || root.Kind() != Record || len(root.Arguments()) != 0 {
		return SQLSchema{}, fmt.Errorf("sql schema at /: requires a concrete ordinary record")
	}
	if root.Owner() {
		return SQLSchema{}, fmt.Errorf("sql schema at /: owner record %s needs an explicit wire type plus the owner factory", CanonicalName(root))
	}
	result := SQLSchema{Root: root.Identity(), Fields: []SQLField{}}
	for _, f := range root.Fields() {
		field, err := SQLFieldOf(f.Type)
		if err != nil {
			return SQLSchema{}, fmt.Errorf("sql schema at /%s: %v", f.Name, err)
		}
		field.Name = f.Name
		result.Fields = append(result.Fields, field)
	}
	return result, nil
}
