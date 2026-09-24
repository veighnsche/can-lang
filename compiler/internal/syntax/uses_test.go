package syntax

import (
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/source"
)

func parseUses(t *testing.T, uses string) []Import {
	t.Helper()
	text := "package app\n    provides []\n    uses [" + uses + "]\n"
	file, err := source.New("uses.can", text)
	if err != nil {
		t.Fatal(err)
	}
	result := Parse(file)
	if !result.OK() {
		t.Fatalf("uses [%s]: %s", uses, result.Diagnostics[0].Format(file))
	}
	return result.File.Header.Uses
}

func TestQualifiedUsesParseAndFormat(t *testing.T) {
	uses := parseUses(t, "alpha, sample::utils, lib::model as first, other as alias")
	if len(uses) != 4 {
		t.Fatalf("parsed %d imports", len(uses))
	}
	if uses[0].Dependency != nil || uses[0].Package.Text != "alpha" || uses[0].Alias != nil {
		t.Fatalf("unqualified import: %+v", uses[0])
	}
	if uses[1].Dependency == nil || uses[1].Dependency.Text != "sample" || uses[1].Package.Text != "utils" || uses[1].Alias != nil {
		t.Fatalf("qualified import: %+v", uses[1])
	}
	if uses[2].Dependency == nil || uses[2].Dependency.Text != "lib" || uses[2].Package.Text != "model" || uses[2].Alias == nil || uses[2].Alias.Text != "first" {
		t.Fatalf("qualified aliased import: %+v", uses[2])
	}
	if uses[3].Dependency != nil || uses[3].Package.Text != "other" || uses[3].Alias == nil || uses[3].Alias.Text != "alias" {
		t.Fatalf("aliased import: %+v", uses[3])
	}
	// The dependency qualifier must not alias the package token.
	if uses[1].Dependency.Span == uses[1].Package.Span || uses[2].Dependency.Span == uses[2].Package.Span {
		t.Fatal("dependency qualifier shares the package span")
	}
	if uses[1].Span.Start != uses[1].Dependency.Span.Start {
		t.Fatal("qualified import span does not cover the dependency qualifier")
	}
	text := "package app\n    provides []\n    uses [alpha, sample::utils, lib::model as first, other as alias]\n"
	file, err := source.New("uses.can", text)
	if err != nil {
		t.Fatal(err)
	}
	result := Parse(file)
	if !result.OK() {
		t.Fatal(result.Diagnostics[0].Format(file))
	}
	if formatted := Format(result.File); formatted != text {
		t.Fatalf("rendered %q", formatted)
	}
}

func TestQualifiedUsesRejectsMalformedQualifiers(t *testing.T) {
	for _, uses := range []string{
		"sample::",
		"::utils",
		"a::b::c",
		"sample::utils as",
		"sample as ::utils",
	} {
		text := "package app\n    provides []\n    uses [" + uses + "]\n"
		file, err := source.New("uses.can", text)
		if err != nil {
			t.Fatal(err)
		}
		result := Parse(file)
		if result.OK() {
			t.Errorf("uses [%s] accepted", uses)
			continue
		}
		if err := file.Validate(result.Diagnostics[0].Span); err != nil {
			t.Errorf("uses [%s] has an unlocatable span: %v", uses, err)
		}
	}
}
