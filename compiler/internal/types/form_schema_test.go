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
