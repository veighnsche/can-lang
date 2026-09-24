package types

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/project"
	"github.com/veighnsche/can-lang/compiler/internal/resolve"
)

func TestSymbolicParameterIdentity(t *testing.T) {
	first, err := SymbolicParameter("can.project.root/lib::doubled", "item")
	if err != nil {
		t.Fatal(err)
	}
	again, err := SymbolicParameter("can.project.root/lib::doubled", "item")
	if err != nil {
		t.Fatal(err)
	}
	other, err := SymbolicParameter("can.project.root/lib::doubled", "other")
	if err != nil {
		t.Fatal(err)
	}
	foreign, err := SymbolicParameter("can.project.root/lib::halved", "item")
	if err != nil {
		t.Fatal(err)
	}
	if first.Kind() != Parameter || !Equal(first, first) {
		t.Fatal("opaque parameter is not sealed evidence")
	}
	if !Equal(first, again) {
		t.Fatal("same declaration parameter lost its identity")
	}
	if Equal(first, other) || Equal(first, foreign) {
		t.Fatal("distinct parameters compare equal")
	}
	if got := OpaqueParameterName(first); got != "doubled<item>" {
		t.Fatalf("parameter renders %q", got)
	}
	if EqualityEligible(first) {
		t.Fatal("opaque parameter claims equality eligibility")
	}
	if _, err := SymbolicParameter("", "item"); err == nil {
		t.Fatal("unnamed parameter declaration admitted")
	}
}

func TestMentionsParameter(t *testing.T) {
	param, err := SymbolicParameter("can.project.root/lib::doubled", "item")
	if err != nil {
		t.Fatal(err)
	}
	if MentionsParameter(nil) {
		t.Fatal("nil mentions a parameter")
	}
	if !MentionsParameter(param) {
		t.Fatal("bare parameter unnoticed")
	}
	array, err := ArrayOfChecked(param)
	if err != nil {
		t.Fatal(err)
	}
	if !Equal(array, array) || !MentionsParameter(array) {
		t.Fatal("parameter array lost its variable")
	}
	contract, err := CallableOfChecked(param, []*Type{param}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !MentionsParameter(contract) {
		t.Fatal("parameter callable lost its variable")
	}
	b, file := buildSource(t, sourceHeader+"record box<item>\n    item value\n    box<item>[] children\n")
	concrete, err := b.Resolve(file, annotation(t, "int"), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if MentionsParameter(concrete) {
		t.Fatal("concrete scalar mentions a parameter")
	}
	// A recursive nominal graph must terminate the search without
	// reporting a variable it does not contain.
	recursive, err := b.Resolve(file, annotation(t, "box<int>"), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if MentionsParameter(recursive) {
		t.Fatal("concrete recursive nominal mentions a parameter")
	}
	model, err := b.Finish()
	if err != nil {
		t.Fatal(err)
	}
	// Symbolic annotations resolve through the specializer, which admits
	// the opaque variable into a sealed graph of its own.
	world, symbolicFile := buildWorld(t, sourceHeader+"record box<item>\n    item value\n    box<item>[] children\n")
	specializer, err := NewSpecializer(world, model)
	if err != nil {
		t.Fatal(err)
	}
	symbolic, err := specializer.Resolve(symbolicFile, annotation(t, "box<item>"), map[string]*Type{"item": param}, false)
	if err != nil {
		t.Fatalf("parameter-sealed graph rejected: %v", err)
	}
	if !MentionsParameter(symbolic) {
		t.Fatal("symbolic nominal lost its variable")
	}
}

func buildWorld(t *testing.T, text string) (*resolve.World, *resolve.File) {
	t.Helper()
	root := t.TempDir()
	for name, data := range map[string]string{"can.project.json": `{"source_root":"src","error_registry":"can.errors.json"}`, "can.errors.json": `{"active":[],"retired":[]}`, "src/main.can": text} {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	graph, err := project.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	world, err := resolve.Build(graph)
	if err != nil {
		t.Fatal(err)
	}
	var file *resolve.File
	for _, candidate := range world.Files {
		file = candidate
	}
	return world, file
}

func TestSpecializationKeyRejectsOpaqueParameters(t *testing.T) {
	param, err := SymbolicParameter("can.project.root/lib::doubled", "item")
	if err != nil {
		t.Fatal(err)
	}
	array, err := ArrayOfChecked(param)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := SpecializationKey("can.project.root/lib::helper", []*Type{array}); err == nil {
		t.Fatal("opaque specialization key admitted")
	} else if !strings.Contains(err.Error(), "opaque type parameter") || !strings.Contains(err.Error(), "doubled<item>") {
		t.Fatalf("opaque key misdiagnosed: %v", err)
	}
}
