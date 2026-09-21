package types

import (
	"strings"
	"testing"
)

func nominal(t *testing.T, g *graph, k Kind, id string, args ...*Type) *Type {
	t.Helper()
	n, e := g.nominal(k, id, args)
	if e != nil {
		t.Fatal(e)
	}
	return n
}
func define(t *testing.T, g *graph, n *Type, fields []Field, alternatives ...*Type) {
	t.Helper()
	if e := g.define(n, fields, alternatives); e != nil {
		t.Fatal(e)
	}
}
func array(t *testing.T, g *graph, e *Type) *Type {
	t.Helper()
	n, err := g.array(e)
	if err != nil {
		t.Fatal(err)
	}
	return n
}
func function(t *testing.T, g *graph, k Kind, r *Type, inputs, errors []*Type) *Type {
	t.Helper()
	n, e := g.function(k, r, inputs, errors)
	if e != nil {
		t.Fatal(e)
	}
	return n
}
func seal(t *testing.T, g *graph) {
	t.Helper()
	if e := g.seal(); e != nil {
		t.Fatal(e)
	}
}

func TestFiniteInhabitation(t *testing.T) {
	for _, mode := range []string{"array", "option", "required", "mutual-required", "variant-exit"} {
		t.Run(mode, func(t *testing.T) {
			g := newGraph()
			node := nominal(t, g, Record, "project::node")
			other := nominal(t, g, Record, "project::other")
			define(t, g, other, []Field{{"node", node}})
			switch mode {
			case "array":
				define(t, g, node, []Field{{"children", array(t, g, other)}})
			case "option":
				none := nominal(t, g, Record, "option::none")
				define(t, g, none, nil)
				some := nominal(t, g, Record, "option::some", node)
				define(t, g, some, []Field{{"value", node}})
				option := nominal(t, g, Variant, "option::value", node)
				define(t, g, option, nil, none, some)
				define(t, g, node, []Field{{"next", option}})
			case "required":
				define(t, g, node, []Field{{"next", node}})
			case "mutual-required":
				define(t, g, node, []Field{{"next", other}})
			case "variant-exit":
				end := nominal(t, g, Record, "project::end")
				define(t, g, end, nil)
				choice := nominal(t, g, Variant, "project::choice")
				define(t, g, choice, nil, other, end)
				define(t, g, node, []Field{{"next", choice}})
			}
			err := g.seal()
			invalid := strings.Contains(mode, "required")
			if invalid {
				if err == nil || !strings.Contains(err.Error(), "finite inhabitant") {
					t.Fatalf("expected finite-inhabitant refusal, got %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestVariantLeafRules(t *testing.T) {
	for _, mode := range []string{"record", "error", "standard_failure", "primitive", "array", "opaque", "callable", "arm", "duplicate", "overlap", "cycle", "mutual-cycle"} {
		t.Run(mode, func(t *testing.T) {
			g := newGraph()
			v := nominal(t, g, Variant, "p::v")
			leaf := nominal(t, g, Record, "p::leaf")
			define(t, g, leaf, nil)
			var alternatives []*Type
			switch mode {
			case "record":
				alternatives = []*Type{leaf}
			case "error":
				e := nominal(t, g, Error, "p::error")
				define(t, g, e, nil)
				alternatives = []*Type{e}
			case "standard_failure", "opaque":
				o := nominal(t, g, Opaque, "distribution::snapshot")
				o.standardFailure = mode == "standard_failure"
				define(t, g, o, nil)
				alternatives = []*Type{o}
			case "primitive":
				alternatives = []*Type{g.scalar("int")}
			case "array":
				alternatives = []*Type{array(t, g, leaf)}
			case "callable", "arm":
				kind := Callable
				if mode == "arm" {
					kind = ChoiceArm
				}
				alternatives = []*Type{function(t, g, kind, g.scalar("void"), nil, nil)}
			case "duplicate":
				alternatives = []*Type{leaf, leaf}
			case "overlap":
				nested := nominal(t, g, Variant, "p::nested")
				define(t, g, nested, nil, leaf)
				alternatives = []*Type{leaf, nested}
			case "cycle":
				alternatives = []*Type{v, leaf}
			case "mutual-cycle":
				nested := nominal(t, g, Variant, "p::nested")
				define(t, g, nested, nil, v, leaf)
				alternatives = []*Type{nested}
			}
			define(t, g, v, nil, alternatives...)
			err := g.seal()
			valid := mode == "record" || mode == "error" || mode == "standard_failure"
			if valid && err != nil {
				t.Fatal(err)
			}
			if !valid && err == nil {
				t.Fatal("accepted invalid variant")
			}
		})
	}
}

func TestNominalAndVariantCompatibility(t *testing.T) {
	g := newGraph()
	i := g.scalar("int")
	s := g.scalar("str")
	a := nominal(t, g, Record, "root/a::item")
	b := nominal(t, g, Record, "root/b::item")
	define(t, g, a, []Field{{"value", i}})
	define(t, g, b, []Field{{"value", i}})
	narrow := nominal(t, g, Variant, "root::narrow")
	wide := nominal(t, g, Variant, "root::wide")
	define(t, g, narrow, nil, a)
	define(t, g, wide, nil, narrow, b)
	boxA := nominal(t, g, Record, "root::box", a)
	boxWide := nominal(t, g, Record, "root::box", wide)
	define(t, g, boxA, []Field{{"value", a}})
	define(t, g, boxWide, []Field{{"value", wide}})
	aa := array(t, g, a)
	aw := array(t, g, wide)
	seal(t, g)
	for _, pair := range [][2]*Type{{a, wide}, {narrow, wide}, {a, narrow}, {a, a}} {
		if !Assignable(pair[0], pair[1]) {
			t.Fatal("rejected variant inclusion")
		}
	}
	for _, pair := range [][2]*Type{{a, b}, {wide, narrow}, {boxA, boxWide}, {aa, aw}, {i, s}, {i, g.scalar("int")}} {
		if pair[0] == pair[1] {
			continue
		}
		if Assignable(pair[0], pair[1]) {
			t.Fatal("accepted nominal/invariant mismatch")
		}
	}
	if Equal(a, b) {
		t.Fatal("same-looking records lost owner identity")
	}
	if !EqualityEligible(wide) || !EqualityEligible(aa) {
		t.Fatal("ordinary data should support equality")
	}
}

func TestCallableCompatibility(t *testing.T) {
	g := newGraph()
	i := g.scalar("int")
	s := g.scalar("str")
	v := g.scalar("void")
	e := nominal(t, g, Error, "p::e")
	f := nominal(t, g, Error, "p::f")
	define(t, g, e, nil)
	define(t, g, f, nil)
	clean := function(t, g, Callable, i, []*Type{s}, nil)
	small := function(t, g, Callable, i, []*Type{s}, []*Type{e})
	large := function(t, g, Callable, i, []*Type{s}, []*Type{f, e})
	same := function(t, g, Callable, i, []*Type{s}, []*Type{e, f})
	wrongResult := function(t, g, Callable, s, []*Type{s}, nil)
	wrongInput := function(t, g, Callable, i, []*Type{i}, nil)
	arm := function(t, g, ChoiceArm, v, nil, []*Type{e})
	voidFn := function(t, g, Callable, v, nil, []*Type{e})
	if _, err := g.function(Callable, i, nil, []*Type{e, e}); err == nil {
		t.Fatal("duplicate bound")
	}
	if _, err := g.function(Callable, i, nil, []*Type{nil}); err == nil {
		t.Fatal("nil bound")
	}
	if _, err := g.function(Callable, i, []*Type{v}, nil); err == nil {
		t.Fatal("void input")
	}
	seal(t, g)
	if same != large || !Assignable(clean, large) || !Assignable(small, large) {
		t.Fatal("finite bound widening failed")
	}
	for _, pair := range [][2]*Type{{large, small}, {large, clean}, {wrongResult, large}, {wrongInput, large}, {arm, voidFn}} {
		if Assignable(pair[0], pair[1]) {
			t.Fatal("invalid callable compatibility")
		}
	}
}

func TestRecursiveEqualityRejectsReachableExcludedData(t *testing.T) {
	for _, excluded := range []Kind{Callable, ChoiceArm, Opaque} {
		t.Run(string(excluded), func(t *testing.T) {
			g := newGraph()
			a := nominal(t, g, Record, "p::a")
			b := nominal(t, g, Record, "p::b")
			var bad *Type
			if excluded == Opaque {
				bad = nominal(t, g, Opaque, "catalogue::resource")
				define(t, g, bad, nil)
			} else {
				bad = function(t, g, excluded, g.scalar("void"), nil, nil)
			}
			define(t, g, a, []Field{{"children", array(t, g, b)}})
			define(t, g, b, []Field{{"back", array(t, g, a)}, {"excluded", bad}})
			seal(t, g)
			if EqualityEligible(a) || EqualityEligible(b) {
				t.Fatal("back edge hid excluded data")
			}
		})
	}
	g := newGraph()
	a := nominal(t, g, Record, "p::a")
	define(t, g, a, []Field{{"children", array(t, g, a)}})
	seal(t, g)
	if !EqualityEligible(a) {
		t.Fatal("eligible recursive data rejected")
	}
}

func TestCopyUpdateContractsAndGraphEncapsulation(t *testing.T) {
	g := newGraph()
	i := g.scalar("int")
	s := g.scalar("str")
	record := nominal(t, g, Record, "p::record")
	resource := nominal(t, g, Opaque, "catalogue::resource")
	define(t, g, record, []Field{{"value", i}, {"then", s}})
	define(t, g, resource, nil)
	seal(t, g)
	fields := record.Fields()
	fields[0] = Field{"forged", s}
	if record.Fields()[0].Name != "value" {
		t.Fatal("mutable field alias")
	}
	got, err := CheckUpdate(record, []Replacement{{"value", i}})
	if err != nil || got != record {
		t.Fatal("update lost exact type")
	}
	for _, replacements := range [][]Replacement{nil, {{"missing", i}}, {{"value", s}}, {{"value", i}, {"value", i}}} {
		if _, err := CheckUpdate(record, replacements); err == nil {
			t.Fatal("accepted invalid update")
		}
	}
	if _, err := CheckUpdate(resource, []Replacement{{"value", i}}); err == nil {
		t.Fatal("opaque update accepted")
	}
}

func TestIdentitySizeAndGraphValidation(t *testing.T) {
	g := newGraph()
	i := g.scalar("int")
	nested := i
	for n := 0; n < 1000; n++ {
		nested = array(t, g, nested)
		if len(nested.Identity()) != 64 {
			t.Fatal("unbounded concrete identity")
		}
	}
	if _, err := g.array(g.scalar("void")); err == nil {
		t.Fatal("void element")
	}
	if _, err := g.array(newGraph().scalar("int")); err == nil {
		t.Fatal("mixed graph")
	}
	rec := nominal(t, g, Record, "p::rec")
	if err := g.seal(); err == nil {
		t.Fatal("undefined node accepted")
	}
	if err := g.define(rec, []Field{{"x", i}, {"x", i}}, nil); err == nil {
		t.Fatal("duplicate fields")
	}
	define(t, g, rec, nil)
	seal(t, g)
	if err := g.define(rec, nil, nil); err == nil {
		t.Fatal("redefinition")
	}
	other := newGraph()
	oi := other.scalar("int")
	seal(t, other)
	if !Equal(i, oi) {
		t.Fatal("same canonical type differs across independent graphs")
	}
}

func TestPhantomVariantArgumentsRemainInvariant(t *testing.T) {
	g := newGraph()
	leaf := nominal(t, g, Record, "p::leaf")
	define(t, g, leaf, nil)
	a := nominal(t, g, Variant, "p::phantom", g.scalar("int"))
	b := nominal(t, g, Variant, "p::phantom", g.scalar("str"))
	define(t, g, a, nil, leaf)
	define(t, g, b, nil, leaf)
	seal(t, g)
	if Assignable(a, b) || Assignable(b, a) {
		t.Fatal("identical leaves erased invariant phantom arguments")
	}
}
