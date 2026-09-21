package types

import "testing"

func TestCodecFiniteRecursiveSchema(t *testing.T) {
	g := newGraph()
	node := nominal(t, g, Record, "app::node")
	children := array(t, g, node)
	define(t, g, node, []Field{{"value", g.scalar("int")}, {"children", children}})
	seal(t, g)
	schema, err := Schema(node)
	if err != nil {
		t.Fatal(err)
	}
	if len(schema.Nodes) != 3 || schema.Root != node.id {
		t.Fatalf("recursive graph expanded or lost: %+v", schema)
	}
}
func TestCodecRejectsReachableUnsupportedContracts(t *testing.T) {
	for _, kind := range []Kind{Opaque, Callable, ChoiceArm, Void} {
		t.Run(string(kind), func(t *testing.T) {
			g := newGraph()
			var unsupported *Type
			switch kind {
			case Opaque:
				unsupported = nominal(t, g, Opaque, "bytes::buffer")
				define(t, g, unsupported, nil)
			case Void:
				unsupported = g.scalar("void")
			default:
				unsupported = function(t, g, kind, g.scalar("int"), nil, nil)
			}
			if kind == Void {
				seal(t, g)
				if _, err := Schema(unsupported); err == nil {
					t.Fatal("void admitted")
				}
				return
			}
			wrapper := nominal(t, g, Record, "app::wrapper")
			define(t, g, wrapper, []Field{{"nested", array(t, g, unsupported)}})
			seal(t, g)
			if _, err := Schema(wrapper); err == nil {
				t.Fatal("unsupported nested contract admitted")
			}
		})
	}
}
func TestCodecCanonicalGenericLeafNames(t *testing.T) {
	g := newGraph()
	integer := g.scalar("int")
	some := nominal(t, g, Record, "can.std.option@1::some", array(t, g, integer))
	define(t, g, some, []Field{{"value", array(t, g, integer)}})
	seal(t, g)
	if got := CanonicalName(some); got != "can.std.option@1::some<int[]>" {
		t.Fatal(got)
	}
}

func TestCodecWireNamesFromResolvedDeclarations(t *testing.T) {
	b, file := buildSource(t, sourceHeader+"record leaf\n    int value\nvariant choice\n    leaf\n")
	if err := b.SeedDeclarations(); err != nil {
		t.Fatal(err)
	}
	choice, err := b.Resolve(file, annotation(t, "choice"), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	option, err := b.Resolve(file, annotation(t, "option::value<int>"), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = b.Finish(); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		typ  *Type
		want string
	}{{choice, "app::leaf"}, {option, "option::some<int>"}} {
		schema, err := Schema(tc.typ)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, node := range schema.Nodes {
			if node.Name == tc.want {
				found = true
			}
		}
		if !found {
			t.Fatalf("wire tag %s absent: %+v", tc.want, schema)
		}
	}
}
