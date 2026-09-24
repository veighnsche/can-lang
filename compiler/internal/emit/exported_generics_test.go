package emit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/project"
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
