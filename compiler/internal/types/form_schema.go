package types

import "fmt"

// Row collection limits for keyed forms. Row values stay small enough to
// redisplay verbatim in a 422 fragment; the row count bounds one submit.
const (
	FormMaxRows       = 64
	FormMaxRowBytes   = 2048
	FormMaxRowNameLen = 64
)

const (
	formRowsDeclaration = "can.std.form@1::rows"
	formOptionValue     = "can.std.option@1::value"
	formOptionSome      = "can.std.option@1::some"
	formOptionNone      = "can.std.option@1::none"
)

// FormSchema is the closed, shallow P10 projection; it does not broaden the
// shared JSON wire subset or accept arbitrary URLSearchParams records.
// Keyed-row collections add one bounded nesting level for grid-style forms.
type FormSchema struct {
	Root   string      `json:"root"`
	Fields []FormField `json:"fields"`
}
type FormField struct {
	Name string         `json:"name"`
	Kind string         `json:"kind"`
	Some string         `json:"some,omitempty"`
	None string         `json:"none,omitempty"`
	Rows *FormRowSchema `json:"rows,omitempty"`
}

// FormRowSchema is the derived contract for one form::rows<Row> collection:
// the row record identity, its exact order field name and the scalar shape
// of every row field. Adapters decode `<name>_order` plus
// `<name>[<key>][<field>]` entries against this schema. Collection is the
// decoded rows value identity; Item is the row_item value identity, filled
// by the checker once the row type is known.
type FormRowSchema struct {
	Row        string      `json:"row"`
	Order      string      `json:"order"`
	Fields     []FormField `json:"fields"`
	Collection string      `json:"collection"`
	Item       string      `json:"item,omitempty"`
}

func Form(root *Type) (FormSchema, error) {
	if !Equal(root, root) || root.Kind() != Record {
		return FormSchema{}, fmt.Errorf("form schema at /: requires an ordinary record")
	}
	if root.Owner() {
		return FormSchema{}, fmt.Errorf("form schema at /: owner record %s needs an explicit wire type plus the owner factory", CanonicalName(root))
	}
	result := FormSchema{Root: root.Identity(), Fields: []FormField{}}
	orders := map[string]string{}
	for _, f := range root.Fields() {
		field := FormField{Name: f.Name}
		if f.Type.Owner() {
			return FormSchema{}, fmt.Errorf("form schema at /%s: owner record %s needs an explicit wire type plus the owner factory", f.Name, CanonicalName(f.Type))
		}
		if f.Type.Kind() == Record && f.Type.Declaration() == formRowsDeclaration {
			rows, err := formRows(f.Name, f.Type)
			if err != nil {
				return FormSchema{}, err
			}
			field.Kind = "rows"
			field.Rows = rows
			orders[field.Rows.Order] = f.Name
		} else {
			scalar, err := formScalar("/"+f.Name, f.Type)
			if err != nil {
				return FormSchema{}, err
			}
			field.Kind = scalar.Kind
			field.Some = scalar.Some
			field.None = scalar.None
		}
		result.Fields = append(result.Fields, field)
	}
	for _, field := range result.Fields {
		if owner, ok := orders[field.Name]; ok {
			return FormSchema{}, fmt.Errorf("form schema at /%s: field collides with the %s order field of keyed rows %s", field.Name, field.Name, owner)
		}
	}
	return result, nil
}

// formRows derives the bounded contract for one keyed-row collection. The
// row type is an ordinary flat record: scalar fields only, no owner
// records and no nested collections.
func formRows(name string, typ *Type) (*FormRowSchema, error) {
	args := typ.Arguments()
	if len(args) != 1 {
		return nil, fmt.Errorf("form schema at /%s: keyed rows require one row record type", name)
	}
	row := args[0]
	if row.Kind() != Record {
		return nil, fmt.Errorf("form schema at /%s: keyed rows require an ordinary record row type, not %s", name, CanonicalName(row))
	}
	if row.Owner() {
		return nil, fmt.Errorf("form schema at /%s: owner record %s needs an explicit wire type plus the owner factory", name, CanonicalName(row))
	}
	schema := &FormRowSchema{Row: row.Identity(), Order: name + "_order", Fields: []FormField{}, Collection: typ.Identity()}
	for _, f := range row.Fields() {
		if f.Type.Owner() {
			return nil, fmt.Errorf("form schema at /%s/%s: owner record %s needs an explicit wire type plus the owner factory", name, f.Name, CanonicalName(f.Type))
		}
		field, err := formScalar("/"+name+"/"+f.Name, f.Type)
		if err != nil {
			return nil, err
		}
		field.Name = f.Name
		schema.Fields = append(schema.Fields, *field)
	}
	return schema, nil
}

// formScalar classifies one flat text field shared by wire roots and row
// records: str, str[] or option::value<str>.
func formScalar(path string, typ *Type) (*FormField, error) {
	field := FormField{}
	switch {
	case typ.Kind() == Primitive && typ.Declaration() == "str":
		field.Kind = "str"
	case typ.Kind() == Array && typ.Element().Kind() == Primitive && typ.Element().Declaration() == "str":
		field.Kind = "array"
	case typ.Kind() == Variant && typ.Declaration() == formOptionValue && len(typ.Arguments()) == 1 && typ.Arguments()[0].Declaration() == "str":
		field.Kind = "optional"
		for _, leaf := range typ.Leaves() {
			switch leaf.Declaration() {
			case formOptionSome:
				field.Some = leaf.Identity()
			case formOptionNone:
				field.None = leaf.Identity()
			}
		}
		if field.Some == "" || field.None == "" {
			return nil, fmt.Errorf("form schema at %s: invalid option shape", path)
		}
	default:
		return nil, fmt.Errorf("form schema at %s: requires str, str[] or option::value<str>", path)
	}
	return &field, nil
}
