package emit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/project"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

func exportedGenericProgram(t *testing.T) *check.Program {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"can.project.json": `{"source_root":"src","error_registry":"can.errors.json"}`,
		"can.errors.json":  `{"active":[],"retired":[]}`,
		"src/lib/lib.can":  "package lib\n    provides [doubled]\n    uses []\nfn item doubled<item>\n    emits []\n    given\n        item value\n        callable item (item, item) emits [] plus\n    asserts\n        triple: 3, callable int_plus => ok 6\n    ok call plus(value, value)\nfn int int_plus\n    emits []\n    given\n        int first\n        int second\n    asserts\n        sample: 3, 3 => ok 6\n    ok first + second\n",
		"src/app/main.can": "package app\n    provides []\n    uses [lib]\nfn int app_plus\n    emits []\n    given\n        int first\n        int second\n    asserts\n        sample: 1, 2 => ok 3\n    ok first + second\nfn int use_doubled\n    emits []\n    asserts\n        sample: => ok 8\n    ok call lib::doubled(4, callable app_plus)\nfn void main\n    emits []\n    given\n        str[] arguments\n    asserts\n        empty: [] => ok\n    ok\n",
	}
	for name, text := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	graph, err := project.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	program, err := check.CheckProgram(graph)
	if err != nil {
		t.Fatal(err)
	}
	return program
}

func TestExportedGenericCallableLowering(t *testing.T) {
	program := exportedGenericProgram(t)
	var instance, caller, plus *check.ProgramFunction
	for _, fn := range program.Functions {
		switch {
		case strings.HasSuffix(fn.Symbol.ID, "::doubled") && fn.Instance != "":
			instance = fn
		case strings.HasSuffix(fn.Symbol.ID, "::use_doubled"):
			caller = fn
		case strings.HasSuffix(fn.Symbol.ID, "::app_plus"):
			plus = fn
		}
	}
	if instance == nil || caller == nil || plus == nil {
		t.Fatal("missing checked functions")
	}
	// The generic instance threads its operation through the callable
	// input ($canArg1); it never inlines arithmetic for the variable.
	emitter := &RegionEmitter{Functions: map[string]string{}}
	body, err := emitter.Function("$doubled", instance.Region)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "$canArg1(") || strings.Contains(body, ") + (") {
		t.Fatalf("instance did not lower through its callable input:\n%s", body)
	}
	// The repaired caller supplies its own written operation; the
	// operation itself keeps native lowering.
	emitter.Functions[instance.Identity()] = "$doubled"
	emitter.Functions[plus.Symbol.ID] = "$appPlus"
	callerBody, err := emitter.Function("$useDoubled", caller.Region)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(callerBody, "$doubled(") || !strings.Contains(callerBody, "$appPlus") || !strings.Contains(callerBody, "4n") {
		t.Fatalf("caller did not pass its written callable:\n%s", callerBody)
	}
	plusBody, err := emitter.Function("$appPlus", plus.Region)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(plusBody, ") + (") {
		t.Fatalf("written operation lost native lowering:\n%s", plusBody)
	}
}

const exportedIdentitySource = `package app
    provides [identity]
    uses []
fn item identity<item>
    emits []
    given
        item value
    asserts
        number: 3 => ok 3
    ok value
fn void main
    emits []
    given
        str[] arguments
    asserts
        empty: [] => ok
    ok
`

func emittedBytes(t *testing.T, artifacts []ir.Artifact) string {
	t.Helper()
	var out strings.Builder
	nonempty := 0
	for _, module := range artifacts {
		if len(module.Bytes) > 0 {
			nonempty++
			out.Write(module.Bytes)
		}
	}
	if nonempty == 0 {
		t.Fatal("emission produced no modules")
	}
	return out.String()
}

// UP02: the public identity control must emit both assertion and production
// modules with complete concrete types and no symbolic placeholder.
func TestExportedGenericPublicIdentityEmitsModules(t *testing.T) {
	program := browserEmitProgram(t, map[string]string{"src/app.can": exportedIdentitySource})
	dependencies := httpDependencies(t)
	assertion, err := AssertionModules(program, "runtime", dependencies)
	if err != nil {
		t.Fatalf("assertion emission failed for public identity: %v", err)
	}
	production, err := ProgramModules(program, "runtime", dependencies)
	if err != nil {
		t.Fatalf("production emission failed for public identity: %v", err)
	}
	for name, body := range map[string]string{
		"assertion":  emittedBytes(t, assertion),
		"production": emittedBytes(t, production),
	} {
		if strings.Contains(strings.ToLower(body), "symbolic") {
			t.Fatalf("%s modules leak a symbolic placeholder", name)
		}
		// Concrete nested types survive isolation: the assertion's int
		// and main's str[] keep their native lowerings.
		for _, want := range []string{"bigint", "ReadonlyArray", "string"} {
			if !strings.Contains(body, want) {
				t.Fatalf("%s modules dropped concrete type lowering %q", name, want)
			}
		}
	}
}

// UP17: a generic call whose concrete target has no emitted function fails
// with the target identity plus call-site evidence, instead of silently
// rendering a symbolic identity.
func TestGenericMissingConcreteTargetFailsClosed(t *testing.T) {
	program := browserEmitProgram(t, map[string]string{
		"src/core/core.can": `package core
    provides [identity]
    uses []
fn item identity<item>
    emits []
    given
        item value
    asserts
        number: 3 => ok 3
    ok value
`,
		"src/helpers/helpers.can": `package helpers
    provides [box, nested]
    uses [core]
record box<item>
    item value
fn box<item> nested<item>
    emits []
    given
        item value
    asserts
        number: 3 => ok box(3)
    ok call core::identity<box<item>>(box(value))
`,
		"src/app/main.can": `package app
    provides []
    uses [helpers]
fn void main
    emits []
    given
        str[] arguments
    asserts
        empty: [] => ok
    helpers::box<int> first = call helpers::nested(3)
    ok
`,
	})
	var nested, boxed *check.ProgramFunction
	for _, fn := range program.Functions {
		switch {
		case fn.Symbol.Name == "nested" && len(fn.TypeArguments) == 1 && fn.TypeArguments[0].Declaration() == "int":
			nested = fn
		case fn.Symbol.Name == "identity" && len(fn.TypeArguments) == 1 && strings.HasSuffix(fn.TypeArguments[0].Declaration(), "::box"):
			boxed = fn
		}
	}
	if nested == nil || boxed == nil {
		t.Fatal("missing checked nested<int> or identity<box<int>> instance")
	}
	bound := map[string]string{}
	for _, fn := range program.Functions {
		bound[fn.Identity()] = "$emitted"
	}
	delete(bound, boxed.Identity())
	emitter := &RegionEmitter{Functions: bound}
	if _, err := emitter.Function("$nested", nested.Region); err == nil {
		t.Fatal("missing concrete target admitted")
	} else if !strings.Contains(err.Error(), "missing concrete target") || !strings.Contains(err.Error(), boxed.Instance) || !strings.Contains(err.Error(), nested.Region.ID) {
		t.Fatalf("missing target lacks identity and call-site evidence: %v", err)
	}
	// Positive control: with the concrete target bound, the call lowers
	// to the emitted function and no symbolic identity is rendered.
	bound[boxed.Identity()] = "$boxed"
	body, err := (&RegionEmitter{Functions: bound}).Function("$nested", nested.Region)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "$boxed(") || strings.Contains(strings.ToLower(body), "symbolic") {
		t.Fatalf("concrete call mislowered:\n%s", body)
	}
}

// UP02: an opaque type that actually reaches a runtime boundary is still a
// hard emission failure, diagnosed at the offending variable.
func TestNativeTypeDeclarationsRejectsOpaqueParameter(t *testing.T) {
	param, err := types.SymbolicParameter("can.project.root/lib::identity", "item")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NativeTypeDeclarations([]*types.Type{param}); err == nil {
		t.Fatal("opaque type parameter admitted at the emission boundary")
	} else if !strings.Contains(err.Error(), "opaque type parameter") || !strings.Contains(err.Error(), "identity<item>") {
		t.Fatalf("opaque emission misdiagnosed: %v", err)
	}
}
