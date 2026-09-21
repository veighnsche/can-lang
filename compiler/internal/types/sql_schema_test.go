package types

import (
	"strings"
	"testing"
)

func sqlOption(t *testing.T, g *graph, inner *Type) *Type {
	t.Helper()
	none := nominal(t, g, Record, "can.std.option@1::none")
	define(t, g, none, nil)
	some := nominal(t, g, Record, "can.std.option@1::some", inner)
	define(t, g, some, []Field{{"value", inner}})
	option := nominal(t, g, Variant, "can.std.option@1::value", inner)
	define(t, g, option, nil, none, some)
	return option
}

func TestSQLSchemaFlatCover(t *testing.T) {
	g := newGraph()
	blob := nominal(t, g, Opaque, "can.std.bytes@1::buffer")
	define(t, g, blob, nil)
	fields := []Field{
		{"flag", g.scalar("bool")},
		{"count", g.scalar("int")},
		{"ratio", g.scalar("float")},
		{"name", g.scalar("str")},
		{"blob", blob},
	}
	root := nominal(t, g, Record, "app::row")
	define(t, g, root, fields)
	seal(t, g)
	schema, err := SQLSchemaOf(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"bool", "int", "float", "str", "bytes"}
	if schema.Root != root.Identity() || len(schema.Fields) != len(want) {
		t.Fatal(schema)
	}
	for i, kind := range want {
		field := schema.Fields[i]
		if field.Name != fields[i].Name || field.Kind != kind || field.Inner != "" || field.Some != "" || field.None != "" {
			t.Fatal(schema.Fields)
		}
	}
}

func TestSQLSchemaOneOptionLayer(t *testing.T) {
	for _, scalar := range []string{"bool", "int", "float", "str"} {
		t.Run(scalar, func(t *testing.T) {
			g := newGraph()
			data := g.scalar(scalar)
			option := sqlOption(t, g, data)
			root := nominal(t, g, Record, "app::row")
			define(t, g, root, []Field{{"maybe", option}})
			seal(t, g)
			schema, err := SQLSchemaOf(root)
			if err != nil {
				t.Fatal(err)
			}
			field := schema.Fields[0]
			if field.Kind != "option" || field.Inner != scalar || field.Some == "" || field.None == "" || field.Some == field.None {
				t.Fatal(schema.Fields)
			}
		})
	}
	g := newGraph()
	blob := nominal(t, g, Opaque, "can.std.bytes@1::buffer")
	define(t, g, blob, nil)
	option := sqlOption(t, g, blob)
	root := nominal(t, g, Record, "app::row")
	define(t, g, root, []Field{{"maybe", option}})
	seal(t, g)
	schema, err := SQLSchemaOf(root)
	if err != nil {
		t.Fatal(err)
	}
	if schema.Fields[0].Kind != "option" || schema.Fields[0].Inner != "bytes" {
		t.Fatal(schema.Fields)
	}
}

func TestSQLSchemaRejects(t *testing.T) {
	cases := []string{"array", "record", "variant", "nested_option", "option_array", "missing_none"}
	for _, kind := range cases {
		t.Run(kind, func(t *testing.T) {
			g := newGraph()
			var field *Type
			switch kind {
			case "array":
				field = array(t, g, g.scalar("int"))
			case "record":
				field = nominal(t, g, Record, "app::nested")
				define(t, g, field, nil)
			case "variant":
				leaf := nominal(t, g, Record, "app::leaf")
				define(t, g, leaf, nil)
				field = nominal(t, g, Variant, "app::choice")
				define(t, g, field, nil, leaf)
			case "nested_option":
				inner := sqlOption(t, g, g.scalar("str"))
				none := nominal(t, g, Record, "can.std.option@1::none")
				some := nominal(t, g, Record, "can.std.option@1::some", inner)
				define(t, g, some, []Field{{"value", inner}})
				field = nominal(t, g, Variant, "can.std.option@1::value", inner)
				define(t, g, field, nil, none, some)
			case "option_array":
				field = sqlOption(t, g, array(t, g, g.scalar("str")))
			case "missing_none":
				data := g.scalar("str")
				some := nominal(t, g, Record, "can.std.option@1::some", data)
				define(t, g, some, []Field{{"value", data}})
				field = nominal(t, g, Variant, "can.std.option@1::value", data)
				define(t, g, field, nil, some)
			}
			root := nominal(t, g, Record, "app::row")
			define(t, g, root, []Field{{"field", field}})
			seal(t, g)
			if _, err := SQLSchemaOf(root); err == nil || !strings.Contains(err.Error(), "/field") {
				t.Fatalf("invalid sql field accepted: %v", err)
			}
		})
	}
	g := newGraph()
	primitive := g.scalar("int")
	seal(t, g)
	if _, err := SQLSchemaOf(primitive); err == nil {
		t.Fatal("primitive sql root accepted")
	}
}
