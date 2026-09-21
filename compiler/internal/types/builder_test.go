package types

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/catalogue"
	"github.com/veighnsche/can-lang/compiler/internal/project"
	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
)

func buildSource(t *testing.T, text string) (*Builder, *resolve.File) {
	t.Helper()
	return buildRegisteredSource(t, text, `{"active":[],"retired":[]}`)
}
func buildRegisteredSource(t *testing.T, text, registry string) (*Builder, *resolve.File) {
	t.Helper()
	root := t.TempDir()
	for name, data := range map[string]string{"can.project.json": `{"source_root":"src","error_registry":"can.errors.json"}`, "can.errors.json": registry, "src/main.can": text} {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	p, err := project.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	w, err := resolve.Build(p)
	if err != nil {
		t.Fatal(err)
	}
	var f *resolve.File
	for _, file := range w.Files {
		f = file
	}
	return NewBuilder(w), f
}
func annotation(t *testing.T, text string) syntax.TypeNode {
	t.Helper()
	f, err := source.New("annotation.can", text)
	if err != nil {
		t.Fatal(err)
	}
	n, ds := syntax.ParseType(f)
	if len(ds) != 0 {
		t.Fatal(ds)
	}
	return n
}

const sourceHeader = "package app\n    provides []\n    uses [option, collections, bytes]\n"

func TestSourceTypeGraphAndCatalogueOption(t *testing.T) {
	b, file := buildSource(t, sourceHeader+"record node\n    option::value<node> next\n    int value\n\nrecord box<item>\n    item value\n\nvariant failure\n    node\n    standard_failure\n")
	if err := b.SeedDeclarations(); err != nil {
		t.Fatal(err)
	}
	node, err := b.Resolve(file, annotation(t, "node"), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	box, err := b.Resolve(file, annotation(t, "box<node>"), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	failure, err := b.Resolve(file, annotation(t, "all_failed<failure>"), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	model, err := b.Finish()
	if err != nil {
		t.Fatal(err)
	}
	if len(model.Types()) < 8 || box.Fields()[0].Type != node || !EqualityEligible(node) {
		t.Fatal("source graph lost specialized recursive fields")
	}
	if EqualityEligible(failure) {
		t.Fatal("standard failure snapshot became equality eligible")
	}
	if len(node.Fields()[0].Type.Leaves()) != 2 {
		t.Fatal("option did not use closed nominal alternatives")
	}
}

func TestSourceConcreteTypeRefusals(t *testing.T) {
	cases := []struct{ declarations, annotation, diagnostic string }{
		{"record node\n    node next\n", "node", "finite inhabitant"},
		{"record box<item>\n    item value\n", "box", "arity"},
		{"record box<item>\n    item value\n", "box<void>", "void"},
		{"record leaf\n", "void[]", "void"},
		{"record leaf\n", "callable int (void) emits []", "void"},
		{"record leaf\n", "callable int () emits [leaf]", "nominal errors"},
		{"record leaf\n", "collections::set<float>", "map_key"},
		{"record leaf\n", "all_failed<leaf>", "failure_variant"},
		{"variant invalid\n    int\n", "invalid", "ineligible variant leaf"},
		{"record leaf\n\nvariant invalid\n    leaf\n    leaf\n", "invalid", "duplicate variant leaf"},
		{"variant invalid\n    invalid\n", "invalid", "variant-only"},
		{"record growing<item>\n    growing<item[]>[] values\n", "growing<int>", "expansion exceeds"},
	}
	for _, c := range cases {
		t.Run(c.annotation+"/"+c.diagnostic, func(t *testing.T) {
			b, file := buildSource(t, sourceHeader+c.declarations)
			typ, err := b.Resolve(file, annotation(t, c.annotation), nil, false)
			if err == nil {
				_, err = b.Finish()
			}
			if err == nil || !strings.Contains(err.Error(), c.diagnostic) {
				t.Fatalf("expected %s, got %v", c.diagnostic, err)
			}
			if typ != nil && Equal(typ, typ) {
				t.Fatal("failed graph remained compatibility evidence")
			}
			if _, err = b.Finish(); err == nil {
				t.Fatal("failed builder recovered silently")
			}
		})
	}
}

func TestGenericPermutationHasFiniteConcreteGraph(t *testing.T) {
	b, file := buildSource(t, sourceHeader+"record alternating<first, second>\n    alternating<second, first>[] children\n    first value\n")
	typ, err := b.Resolve(file, annotation(t, "alternating<int,str>"), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = b.Finish(); err != nil {
		t.Fatal(err)
	}
	next := typ.Fields()[0].Type.Element()
	if next == typ || next.Fields()[0].Type.Element() != typ {
		t.Fatal("finite permutation graph not interned")
	}
}

func TestDeclarationAnnotationsCheckUnusedTemplates(t *testing.T) {
	for _, declaration := range []string{
		"record unused<item>\n    void value\n",
		"record box<item>\n    item value\n\nrecord unused<item>\n    box value\n",
		"variant unused<item>\n    int\n    item\n",
		"record leaf\n\nvariant unused<item>\n    leaf\n    leaf\n    item\n",
		"record unused<item>\n    callable item (void) emits [] callback\n",
	} {
		b, _ := buildSource(t, sourceHeader+declaration)
		if _, err := CheckDeclarations(b.world); err == nil {
			t.Fatalf("accepted parameter-independent template error: %s", declaration)
		}
	}
	b, _ := buildSource(t, sourceHeader+"record unused<item>\n    callable item (item) emits [] callback\n")
	if _, err := CheckDeclarations(b.world); err != nil {
		t.Fatal(err)
	}
}

func TestGeneratedFieldMetadataCollisions(t *testing.T) {
	g := newGraph()
	i := g.scalar("int")
	f := g.scalar("float")
	seal(t, g)
	fields := []Field{{"selected", i}}
	for _, metadata := range [][]Field{{{"selected", f}}, {{"confidence", f}, {"confidence", f}}, {{"confidence", i}}} {
		if _, err := MergeGeneratedFields(fields, metadata); err == nil {
			t.Fatal("accepted field/metadata collision or wrong metadata type")
		}
	}
	out, err := MergeGeneratedFields(fields, []Field{{"confidence", f}, {"score", f}})
	if err != nil || out[0].Name != "selected" || out[1].Name != "confidence" || out[2].Name != "score" {
		t.Fatal("generated field order changed")
	}
	out[0].Name = "changed"
	if fields[0].Name != "selected" {
		t.Fatal("generated fields share mutable backing storage")
	}
}

func TestFiniteSpecializationCanChangeThenStabilize(t *testing.T) {
	b, file := buildSource(t, sourceHeader+"record stable<item>\n    stable<int[]>[] children\n")
	typ, err := b.Resolve(file, annotation(t, "stable<int>"), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = b.Finish(); err != nil {
		t.Fatal(err)
	}
	child := typ.Fields()[0].Type.Element()
	if child == typ || child.Fields()[0].Type.Element() != child {
		t.Fatal("transient growth was mistaken for infinite expansion")
	}
}

func TestLongFiniteNominalChain(t *testing.T) {
	var text strings.Builder
	text.WriteString(sourceHeader)
	for i := 0; i < 300; i++ {
		fmt.Fprintf(&text, "record node%d\n    node%d next\n", i, i+1)
	}
	text.WriteString("record node300\n")
	b, _ := buildSource(t, text.String())
	if _, err := CheckDeclarations(b.world); err != nil {
		t.Fatal(err)
	}
}

func TestClosedCatalogueTypeShapes(t *testing.T) {
	b, file := buildSource(t, sourceHeader+"record leaf\n\nvariant failure\n    leaf\n    standard_failure\n")
	failure, err := b.Resolve(file, annotation(t, "failure"), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	inventory := catalogue.Builtin().Inventory()
	check := func(name string, parameters []catalogue.Parameter) {
		args := make([]*Type, len(parameters))
		for i, p := range parameters {
			args[i] = b.graph.scalar("int")
			if p.Constraint == "failure_variant" {
				args[i] = failure
			}
		}
		if _, err := b.instantiate(b.catalogue[name], args); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	for _, d := range inventory.Types {
		check(d.Name, d.Parameters)
	}
	for _, d := range inventory.Errors {
		check(d.Name, d.Parameters)
	}
	if _, err = b.Finish(); err != nil {
		t.Fatal(err)
	}
}
