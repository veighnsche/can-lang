package types

import (
	"strings"
	"testing"
)

func TestFormSchemaClosedShallowFields(t *testing.T) {
	g := newGraph()
	root := nominal(t, g, Record, "app::form")
	define(t, g, root, []Field{{"name", g.scalar("str")}, {"aliases", array(t, g, g.scalar("str"))}})
	seal(t, g)
	schema, err := Form(root)
	if err != nil {
		t.Fatal(err)
	}
	if schema.Root != root.Identity() || len(schema.Fields) != 2 || schema.Fields[0].Kind != "str" || schema.Fields[1].Kind != "array" {
		t.Fatal(schema)
	}
}
func TestFormSchemaRejectsNonTextFields(t *testing.T) {
	for _, kind := range []string{"int", "float", "bool", "nested", "array"} {
		t.Run(kind, func(t *testing.T) {
			g := newGraph()
			var field *Type
			switch kind {
			case "nested":
				field = nominal(t, g, Record, "app::nested")
				define(t, g, field, nil)
			case "array":
				field = array(t, g, g.scalar("int"))
			default:
				field = g.scalar(kind)
			}
			root := nominal(t, g, Record, "app::form")
			define(t, g, root, []Field{{"field", field}})
			seal(t, g)
			if _, err := Form(root); err == nil || !strings.Contains(err.Error(), "/field") {
				t.Fatalf("invalid form field: %v", err)
			}
		})
	}
	g := newGraph()
	root := g.scalar("str")
	seal(t, g)
	if _, err := Form(root); err == nil {
		t.Fatal("primitive form root accepted")
	}
}

func TestFormSchemaConcreteOptionIdentity(t *testing.T) {
	for _, scalar := range []string{"str", "int"} {
		t.Run(scalar, func(t *testing.T) {
			g := newGraph()
			data := g.scalar(scalar)
			none := nominal(t, g, Record, "can.std.option@1::none")
			define(t, g, none, nil)
			some := nominal(t, g, Record, "can.std.option@1::some", data)
			define(t, g, some, []Field{{"value", data}})
			option := nominal(t, g, Variant, "can.std.option@1::value", data)
			define(t, g, option, nil, none, some)
			root := nominal(t, g, Record, "app::form", data)
			define(t, g, root, []Field{{"note", option}})
			seal(t, g)
			schema, err := Form(root)
			if scalar != "str" {
				if err == nil {
					t.Fatal("non-string optional field accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(schema.Fields) != 1 || schema.Fields[0].Kind != "optional" || schema.Fields[0].Some != some.Identity() || schema.Fields[0].None != none.Identity() {
				t.Fatal(schema)
			}
		})
	}
}

func formTestOption(t *testing.T, g *graph, data *Type) *Type {
	t.Helper()
	none := nominal(t, g, Record, "can.std.option@1::none")
	define(t, g, none, nil)
	some := nominal(t, g, Record, "can.std.option@1::some", data)
	define(t, g, some, []Field{{"value", data}})
	option := nominal(t, g, Variant, "can.std.option@1::value", data)
	define(t, g, option, nil, none, some)
	return option
}

func TestFormSchemaKeyedRows(t *testing.T) {
	g := newGraph()
	str := g.scalar("str")
	row := nominal(t, g, Record, "app::line_wire")
	define(t, g, row, []Field{{"sku", str}, {"tags", array(t, g, str)}, {"note", formTestOption(t, g, str)}})
	rows := nominal(t, g, Record, "can.std.form@1::rows", row)
	define(t, g, rows, []Field{{"order", array(t, g, str)}})
	root := nominal(t, g, Record, "app::form")
	define(t, g, root, []Field{{"customer", str}, {"lines", rows}})
	seal(t, g)
	schema, err := Form(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(schema.Fields) != 2 || schema.Fields[1].Kind != "rows" || schema.Fields[1].Rows == nil {
		t.Fatalf("rows field missing: %+v", schema.Fields)
	}
	derived := schema.Fields[1].Rows
	if derived.Row != row.Identity() || derived.Order != "lines_order" || derived.Collection != rows.Identity() {
		t.Fatalf("row contract = %+v", derived)
	}
	if len(derived.Fields) != 3 || derived.Fields[0].Kind != "str" || derived.Fields[1].Kind != "array" || derived.Fields[2].Kind != "optional" {
		t.Fatalf("row fields = %+v", derived.Fields)
	}
	if derived.Fields[2].Some == "" || derived.Fields[2].None == "" {
		t.Fatalf("row option leaves = %+v", derived.Fields[2])
	}
}

func TestFormSchemaKeyedRowsRejects(t *testing.T) {
	cases := []struct {
		name  string
		build func(t *testing.T, g *graph) *Type
		want  string
	}{
		{"non-record row", func(t *testing.T, g *graph) *Type {
			rows := nominal(t, g, Record, "can.std.form@1::rows", g.scalar("str"))
			define(t, g, rows, nil)
			root := nominal(t, g, Record, "app::form")
			define(t, g, root, []Field{{"lines", rows}})
			return root
		}, "/lines"},
		{"non-text row field", func(t *testing.T, g *graph) *Type {
			row := nominal(t, g, Record, "app::line_wire")
			define(t, g, row, []Field{{"qty", g.scalar("int")}})
			rows := nominal(t, g, Record, "can.std.form@1::rows", row)
			define(t, g, rows, nil)
			root := nominal(t, g, Record, "app::form")
			define(t, g, root, []Field{{"lines", rows}})
			return root
		}, "/lines/qty"},
		{"nested rows", func(t *testing.T, g *graph) *Type {
			inner := nominal(t, g, Record, "app::inner")
			define(t, g, inner, []Field{{"sku", g.scalar("str")}})
			nested := nominal(t, g, Record, "can.std.form@1::rows", inner)
			define(t, g, nested, nil)
			row := nominal(t, g, Record, "app::line_wire")
			define(t, g, row, []Field{{"parts", nested}})
			rows := nominal(t, g, Record, "can.std.form@1::rows", row)
			define(t, g, rows, nil)
			root := nominal(t, g, Record, "app::form")
			define(t, g, root, []Field{{"lines", rows}})
			return root
		}, "/lines/parts"},
		{"order collision", func(t *testing.T, g *graph) *Type {
			row := nominal(t, g, Record, "app::line_wire")
			define(t, g, row, []Field{{"sku", g.scalar("str")}})
			rows := nominal(t, g, Record, "can.std.form@1::rows", row)
			define(t, g, rows, nil)
			root := nominal(t, g, Record, "app::form")
			define(t, g, root, []Field{{"lines", rows}, {"lines_order", g.scalar("str")}})
			return root
		}, "/lines_order"},
		{"non-string row option", func(t *testing.T, g *graph) *Type {
			row := nominal(t, g, Record, "app::line_wire")
			define(t, g, row, []Field{{"note", formTestOption(t, g, g.scalar("int"))}})
			rows := nominal(t, g, Record, "can.std.form@1::rows", row)
			define(t, g, rows, nil)
			root := nominal(t, g, Record, "app::form")
			define(t, g, root, []Field{{"lines", rows}})
			return root
		}, "/lines/note"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := newGraph()
			root := tc.build(t, g)
			seal(t, g)
			_, err := Form(root)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("rows accepted: %v", err)
			}
		})
	}
}
