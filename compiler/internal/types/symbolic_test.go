package types

import (
	"strings"
	"testing"
)

func TestUnusedGenericSymbolicVariantDuplicates(t *testing.T) {
	box := "record box<item>\n    item value\n\n"
	cases := []string{
		"variant invalid<item>\n    box<item>\n    box<item>\n",
		"variant invalid<item>\n    box<item[]>\n    box<item[]>\n",
		"variant inner<item>\n    box<item>\n\nvariant invalid<item>\n    inner<item>\n    box<item>\n",
		"variant invalid<item>\n    option::value<item>\n    option::value<item>\n",
		"variant invalid<item>\n    item\n    item\n",
	}
	for _, declaration := range cases {
		b, _ := buildSource(t, sourceHeader+box+declaration)
		if _, err := CheckDeclarations(b.world); err == nil || !strings.Contains(err.Error(), "duplicate variant leaf") {
			t.Fatalf("expected symbolic duplicate, got %v\n%s", err, declaration)
		}
	}
	aliasHeader := strings.Replace(sourceHeader, "uses [option, collections, bytes]", "uses [option, collections, bytes, app as first, app as second]", 1)
	b, _ := buildSource(t, aliasHeader+box+"variant invalid<item>\n    first::box<item>\n    second::box<item>\n")
	if _, err := CheckDeclarations(b.world); err == nil || !strings.Contains(err.Error(), "duplicate variant leaf") {
		t.Fatalf("alias hid declaration identity: %v", err)
	}
}

func TestSymbolicCallableArgumentsNormalizeBounds(t *testing.T) {
	header := strings.Replace(sourceHeader, "uses [option, collections, bytes]", "uses [option, collections, bytes, number, text]", 1)
	declaration := "record box<item>\n    item value\n\nvariant invalid<item>\n    box<callable item () emits [number::inexact, text::invalid_bool]>\n    box<callable item () emits [text::invalid_bool, number::inexact]>\n"
	b, _ := buildSource(t, header+declaration)
	if _, err := CheckDeclarations(b.world); err == nil || !strings.Contains(err.Error(), "duplicate variant leaf") {
		t.Fatalf("bound order hid symbolic identity: %v", err)
	}
}

func TestUnusedGenericSymbolicErrorBounds(t *testing.T) {
	registry := `{"active":["app::failed"],"retired":[]}`
	for _, bound := range []string{"failed<item>, failed<item>", "failed<item[]>, failed<item[]>"} {
		text := sourceHeader + "error failed<item>(item value)\nfn item work<item>\n    emits [" + bound + "]\n    given\n        item value\n    asserts\n        sample: 1 => ok 1\n    ok value\n"
		b, _ := buildRegisteredSource(t, text, registry)
		if _, err := CheckDeclarations(b.world); err == nil || !strings.Contains(err.Error(), "duplicate error in bound") {
			t.Fatalf("dependent bound admitted duplicate: %v", err)
		}
	}
	text := sourceHeader + "error failed<item>(item value)\nrecord holder<item>\n    callable item () emits [failed<item>, failed<item>] callback\n"
	b, _ := buildRegisteredSource(t, text, registry)
	if _, err := CheckDeclarations(b.world); err == nil || !strings.Contains(err.Error(), "duplicate error in bound") {
		t.Fatalf("nested callable bound admitted duplicate: %v", err)
	}
}

func TestIndependentCatalogueConstraintsInUnusedGenerics(t *testing.T) {
	for _, typ := range []string{"collections::map<float,item>", "collections::map<box<item>,item>", "collections::map<item[],item>", "all_failed<box<item>>"} {
		b, _ := buildSource(t, sourceHeader+"record box<item>\n    item value\n\nrecord holder<item>\n    "+typ+" value\n")
		if _, err := CheckDeclarations(b.world); err == nil || !strings.Contains(err.Error(), "catalogue constraint") {
			t.Fatalf("independent constraint hidden in %s: %v", typ, err)
		}
	}
	for _, typ := range []string{"collections::map<int,item>", "collections::map<item,str>"} {
		b, _ := buildSource(t, sourceHeader+"record holder<item>\n    "+typ+" value\n")
		if _, err := CheckDeclarations(b.world); err != nil {
			t.Fatalf("valid/dependent constraint rejected in %s: %v", typ, err)
		}
	}
}

func TestDistinctSymbolicParametersDeferPotentialOverlap(t *testing.T) {
	declaration := sourceHeader + "record box<item>\n    item value\n\nvariant possibly_disjoint<left,right>\n    box<left>\n    box<right>\n"
	b, _ := buildSource(t, declaration)
	if _, err := CheckDeclarations(b.world); err != nil {
		t.Fatal("different parameters are not definite duplicates", err)
	}
	b, file := buildSource(t, declaration)
	if _, err := b.Resolve(file, annotation(t, "possibly_disjoint<int,str>"), nil, false); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Finish(); err != nil {
		t.Fatal(err)
	}
	b, file = buildSource(t, declaration)
	if _, err := b.Resolve(file, annotation(t, "possibly_disjoint<int,int>"), nil, false); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Finish(); err == nil || !strings.Contains(err.Error(), "duplicate variant leaf") {
		t.Fatalf("concrete overlap escaped: %v", err)
	}
}

func TestUnusedGenericVariantOnlyCycle(t *testing.T) {
	b, _ := buildSource(t, sourceHeader+"variant cyclic<item>\n    cyclic<item>\n")
	if _, err := CheckDeclarations(b.world); err == nil || !strings.Contains(err.Error(), "variant-only") {
		t.Fatalf("generic variant cycle escaped: %v", err)
	}
}

func TestUnusedGenericRequiredCycles(t *testing.T) {
	for _, declaration := range []string{
		"record impossible<item>\n    impossible<item> next\n",
		"record first<item>\n    second<item> next\n\nrecord second<item>\n    first<item> next\n",
	} {
		b, _ := buildSource(t, sourceHeader+declaration)
		if _, err := CheckDeclarations(b.world); err == nil || !strings.Contains(err.Error(), "no possible finite inhabitant") {
			t.Fatalf("mandatory generic cycle admitted: %v", err)
		}
	}
	for _, declaration := range []string{
		"record possible<item>\n    item value\n",
		"record possible<item>\n    possible<item>[] children\n",
		"record possible<item>\n    option::value<possible<item>> next\n    item value\n",
	} {
		b, _ := buildSource(t, sourceHeader+declaration)
		if _, err := CheckDeclarations(b.world); err != nil {
			t.Fatalf("potentially inhabited template rejected: %v", err)
		}
	}
}
