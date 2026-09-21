package emit

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
	"github.com/veighnsche/can-lang/distribution"
)

func initialValues(t *testing.T, text string, ts map[string]*types.Type) []check.InitialValue {
	t.Helper()
	src, _ := source.New("app/main.can", "package app\n    provides []\n    uses [alpha]\n"+text)
	parsed := syntax.Parse(src)
	if !parsed.OK() {
		t.Fatal(parsed.Diagnostics)
	}
	context := expressionContext(ts)
	lookup := context.Value
	declared := map[string]*types.Type{}
	for _, d := range parsed.File.Declarations {
		v, ok := d.(*syntax.ValueDecl)
		if !ok {
			t.Fatal("expected value declaration")
		}
		declared[v.Binding.Name.Text] = ts[syntax.FormatType(v.Binding.Type)]
	}
	context.Value = func(name syntax.QualifiedName) (check.ValueBinding, error) {
		if name.Package == "" && declared[name.Name] != nil {
			return check.ValueBinding{Identity: "app::" + name.Name, Type: declared[name.Name]}, nil
		}
		return lookup(name)
	}
	var values []check.InitialValue
	for _, d := range parsed.File.Declarations {
		v := d.(*syntax.ValueDecl)
		values = append(values, check.InitialValue{Identity: "app::" + v.Binding.Name.Text, QualifiedName: "app::" + v.Binding.Name.Text, Source: src.Name(), Binding: v.Binding, Type: declared[v.Binding.Name.Text], Checker: context})
	}
	return values
}
func names(plan []ir.Initializer) string {
	var out []string
	for _, p := range plan {
		out = append(out, p.QualifiedName)
	}
	return strings.Join(out, ",")
}

func TestInitializationOrderingAndRefusals(t *testing.T) {
	ts := fixtureTypes(t)
	values := initialValues(t, "int a = z + 1\nint z = 2\nint m = 1\n", ts)
	plan, err := check.Initialization(values, nil)
	if err != nil {
		t.Fatal(err)
	}
	if names(plan) != "app::m,app::z,app::a" {
		t.Fatal("not ready-node qualified order", names(plan))
	}
	values[0], values[2] = values[2], values[0]
	again, err := check.Initialization(values, nil)
	if err != nil || names(again) != names(plan) {
		t.Fatal("input order affected initialization", err)
	}
	for _, text := range []string{
		"int a = b\nint b = a\n", "int a = a\n", "int a = call first()\n", "int a = (1 + call first())\n",
		"str a = call env::get(\"TOKEN\")\n", "bytes::buffer a = call bytes::from_utf8(\"x\")\n",
		"callable int () emits [] a = callable first\n", "int a = ints[0]\n",
		"bytes::buffer a = resource\n",
		"int a = match true\n    true => 1\n    false => 2\n",
		"bool a = false and (call first() is 1)\n",
	} {
		t.Run(text, func(t *testing.T) {
			if _, err := check.Initialization(initialValues(t, text, ts), nil); err == nil {
				t.Fatal("non-inert or cyclic initializer accepted")
			}
		})
	}
	// No evaluator runs in this pass: a checked primitive fault is deferred until
	// native startup, with its original source location.
	if _, err := check.Initialization(initialValues(t, "int a = 1 / 0\n", ts), nil); err != nil {
		t.Fatal(err)
	}
	if _, err := check.Initialization(initialValues(t, "alpha::item a = alpha::item(1, [2])\nalpha::item b = a with value=3\nint c = b.shared[0]\nstr s = \"abcd\"[1:3]\n", ts), nil); err != nil {
		t.Fatal(err)
	}
}

func TestInitializationNamedArmEvidence(t *testing.T) {
	ts := fixtureTypes(t)
	values := initialValues(t, "choice_arm<int> emits [] a = arm\n", ts)
	arm := check.ValueBinding{Identity: "app::arm", Type: ts["choice_arm<int> emits []"]}
	values[0].Checker.Value = func(_ syntax.QualifiedName) (check.ValueBinding, error) { return arm, nil }
	if _, err := check.Initialization(values, nil); err == nil {
		t.Fatal("unproven named arm admitted")
	}
	plan, err := check.Initialization(values, []check.ValueBinding{arm})
	if err != nil {
		t.Fatal(err)
	}
	code, err := Initialization(plan, map[string]string{arm.Identity: "$namedArm"})
	if err != nil || !strings.Contains(code.Code, "= $namedArm;") || strings.Contains(code.Code, "$namedArm(") {
		t.Fatal("storing a named arm must not invoke it", err, code.Code)
	}
	arm.Type = ts["callable int () emits []"]
	if _, err := check.Initialization(values, []check.ValueBinding{arm}); err == nil {
		t.Fatal("ordinary callable admitted as a named arm")
	}
}

func TestEmittedInitializationBeforeMain(t *testing.T) {
	bun := os.Getenv("CAN_BUN")
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if bun == "" && archive == "" {
		t.Skip("set CAN_BUN or CAN_BUN_ARCHIVE for native startup integration")
	}
	runtimeDir, _ := filepath.Abs("../../../runtime")
	offline := false
	if archive != "" {
		sourceRoot, _ := filepath.Abs("../../..")
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		root, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "initialization-test")
		if err != nil {
			t.Fatal(err)
		}
		runtimeDir = filepath.Join(root, "runtime")
		bun = filepath.Join(runtimeDir, "bun")
		offline = true
	}
	pinned := distribution.PinnedTarget()
	binary, err := os.ReadFile(bun)
	if !filepath.IsAbs(bun) || err != nil || distribution.Hash(binary) != pinned.Runtime.SHA256 || runtime.GOOS != pinned.Runtime.Platform || runtime.GOARCH != pinned.Runtime.Architecture {
		t.Fatal("initialization needs qualified native runtime")
	}
	ts := fixtureTypes(t)
	for _, fault := range []bool{false, true} {
		t.Run(fmt.Sprint("fault=", fault), func(t *testing.T) {
			text := "int a = z + 1\nint z = 2\nalpha::item value = alpha::item(a, [z])\n"
			if fault {
				text = "int a = z + 1\nint z = 1 / 0\nalpha::item value = alpha::item(a, [z])\n"
			}
			plan, err := check.Initialization(initialValues(t, text, ts), nil)
			if err != nil {
				t.Fatal(err)
			}
			emitted, err := Initialization(plan, nil)
			if err != nil {
				t.Fatal(err)
			}
			imports := DataImports(filepath.Join(runtimeDir, "data.ts")) + PrimitiveImports(filepath.Join(runtimeDir, "primitive.ts")) + InitializationImports(filepath.Join(runtimeDir, "failure.ts"))
			module := imports + emitted.Code + "\nexport const value = " + emitted.Bindings["app::value"] + ";\nconsole.log('main reached');\n"
			dir := t.TempDir()
			path := filepath.Join(dir, "initialized.ts")
			if err = os.WriteFile(path, []byte(module), 0600); err != nil {
				t.Fatal(err)
			}
			runner := `import {strict as assert} from "node:assert";
import { standardFailureKind, standardFailureMessage, standardFailureDiagnostics } from ` + quote(filepath.Join(runtimeDir, "failure.ts")) + `;`
			if fault {
				var span source.Span
				for _, entry := range plan {
					if entry.Identity == "app::z" {
						span = entry.Span
					}
				}
				runner += fmt.Sprintf(`
try {await import(%s);assert.fail("startup unexpectedly succeeded")}catch(failure){
 assert.equal(standardFailureKind(failure),"arithmetic");assert.equal(standardFailureMessage(failure),"arithmetic: integer division by zero");
 const origin=standardFailureDiagnostics(failure).origin;
 assert.deepEqual(origin,{source:"app/main.can",start:%d,end:%d,invocation:["initialization:app::z"]});
 console.log("startup failure reported");}
`, quote(path), span.Start, span.End)
			} else {
				runner += `const first=await import(` + quote(path) + `);const second=await import(` + quote(path) + `);assert.equal(first.value,second.value);assert.equal(first.value.value,3n);assert.deepEqual(first.value.shared,[2n]);console.log("startup once verified");`
			}
			runnerPath := filepath.Join(dir, "run.ts")
			if err = os.WriteFile(runnerPath, []byte(runner), 0600); err != nil {
				t.Fatal(err)
			}
			args := []string{"--no-env-file", "--no-macros", "--no-install", runnerPath}
			command := exec.Command(bun, args...)
			if offline {
				args = append([]string{"-p", "(version 1)(allow default)(deny network*)", bun}, args...)
				command = exec.Command("/usr/bin/sandbox-exec", args...)
			}
			command.Dir = t.TempDir()
			command.Env = []string{"HOME=" + command.Dir, "XDG_CONFIG_HOME=" + command.Dir, "PATH=/nonexistent"}
			output, err := command.CombinedOutput()
			if err != nil {
				t.Fatalf("%v\n%s", err, output)
			}
			if fault && strings.Contains(string(output), "main reached") {
				t.Fatal("main ran after startup fault")
			}
			if !fault && strings.Count(string(output), "main reached") != 1 {
				t.Fatal("startup did not execute exactly once", string(output))
			}
			t.Log(strings.TrimSpace(string(output)))
		})
	}
}
