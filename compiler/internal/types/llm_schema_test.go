package types

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestLLMSchemaClosedExpandedAndStable(t *testing.T) {
	g := newGraph()
	item := nominal(t, g, Record, "app::item")
	define(t, g, item, []Field{{"amount", g.scalar("int")}, {"label", g.scalar("str")}})
	root := nominal(t, g, Record, "app::root")
	define(t, g, root, []Field{{"items", array(t, g, item)}, {"again", item}, {"ratio", g.scalar("float")}, {"enabled", g.scalar("bool")}})
	seal(t, g)
	schema, err := LLMSchema(root)
	if err != nil {
		t.Fatal(err)
	}
	repeated, err := LLMSchema(root)
	if err != nil || schema.Name != repeated.Name || string(schema.Schema) != string(repeated.Schema) {
		t.Fatal("schema nondeterminism")
	}
	if len(schema.Name) != 36 || !strings.HasPrefix(schema.Name, "can_") {
		t.Fatal(schema.Name)
	}
	var value map[string]any
	if err := json.Unmarshal(schema.Schema, &value); err != nil {
		t.Fatal(err)
	}
	if value["type"] != "object" || value["additionalProperties"] != false || len(value["required"].([]any)) != 4 {
		t.Fatal(value)
	}
	props := value["properties"].(map[string]any)
	if props["items"].(map[string]any)["items"].(map[string]any)["additionalProperties"] != false {
		t.Fatal("nested object open")
	}
	if strings.Count(string(schema.Schema), `"amount"`) != 4 {
		t.Fatal("repeated shape not expanded")
	}
}
func TestLLMSchemaRejectsUnsupportedAndRecursive(t *testing.T) {
	for _, kind := range []Kind{Opaque, Callable, ChoiceArm, Error, Variant} {
		t.Run(string(kind), func(t *testing.T) {
			g := newGraph()
			var bad *Type
			switch kind {
			case Opaque, Error:
				bad = nominal(t, g, kind, "app::bad")
				define(t, g, bad, nil)
			case Void:
				bad = g.scalar("void")
			case Variant:
				bad = nominal(t, g, Record, "app::leaf")
				define(t, g, bad, nil)
				leaf := bad
				bad = nominal(t, g, Variant, "app::bad")
				define(t, g, bad, nil, leaf)
			default:
				bad = function(t, g, kind, g.scalar("int"), nil, nil)
			}
			root := nominal(t, g, Record, "app::root")
			define(t, g, root, []Field{{"nested", array(t, g, bad)}})
			seal(t, g)
			if _, err := LLMSchema(root); err == nil || !strings.Contains(err.Error(), "/nested/*") {
				t.Fatalf("unsupported path: %v", err)
			}
		})
	}
	g := newGraph()
	text := g.scalar("str")
	root := nominal(t, g, Record, "app::node")
	define(t, g, root, []Field{{"children", array(t, g, root)}})
	seal(t, g)
	if _, err := LLMSchema(root); err == nil || !strings.Contains(err.Error(), "recursive") {
		t.Fatal(err)
	}
	if _, err := LLMSchema(text); err == nil {
		t.Fatal("scalar root accepted")
	}
}
func TestLLMSchemaBudgets(t *testing.T) {
	for _, depth := range []int{8, 9} {
		g := newGraph()
		leaf := g.scalar("str")
		for i := 1; i < depth; i++ {
			leaf = array(t, g, leaf)
		}
		root := nominal(t, g, Record, "app::deep")
		define(t, g, root, []Field{{"nested", leaf}})
		seal(t, g)
		_, err := LLMSchema(root)
		if (err != nil) != (depth > 8) {
			t.Fatalf("depth %d: %v", depth, err)
		}
	}
	for _, count := range []int{1024, 1025} {
		g := newGraph()
		root := nominal(t, g, Record, "app::wide")
		var fields []Field
		for i := 0; i < count; i++ {
			fields = append(fields, Field{fmt.Sprintf("f%d", i), g.scalar("bool")})
		}
		define(t, g, root, fields)
		seal(t, g)
		_, err := LLMSchema(root)
		if (err != nil) != (count > 1024) {
			t.Fatalf("property count %d: %v", count, err)
		}
	}
	g := newGraph()
	root := nominal(t, g, Record, "app::large")
	define(t, g, root, []Field{{strings.Repeat("x", 33000), g.scalar("str")}})
	seal(t, g)
	if _, err := LLMSchema(root); err == nil || !strings.Contains(err.Error(), "65536") {
		t.Fatalf("byte budget: %v", err)
	}
}
