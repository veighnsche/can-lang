package emit

import (
	"context"
	"fmt"
	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/project"
	"github.com/veighnsche/can-lang/compiler/internal/types"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func callableProgram(t *testing.T) *check.Program {
	t.Helper()
	root := t.TempDir()
	data, err := os.ReadFile("../../testdata/current/callables/captures.can")
	if err != nil {
		t.Fatal(err)
	}
	for name, text := range map[string]string{"can.project.json": `{"source_root":"src","error_registry":"can.errors.json"}`, "can.errors.json": `{"active":[],"retired":[]}`, "src/main.can": string(data)} {
		p := filepath.Join(root, name)
		if err = os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(p, []byte(text), 0600); err != nil {
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
func TestNativeCallableInstancesAndCaptureTiming(t *testing.T) {
	bun := os.Getenv("CAN_BUN")
	if bun == "" {
		t.Skip("set CAN_BUN for native closure qualification")
	}
	program := callableProgram(t)
	var compute, receiver *ir.Region
	targets := map[string]string{}
	for _, fn := range program.Functions {
		switch fn.Symbol.Name {
		case "compute":
			compute = fn.Region
		case "receiver_capture":
			receiver = fn.Region
		case "combine":
			targets[fn.Symbol.ID] = "$combine"
		case "consume":
			targets[fn.Symbol.ID] = "$consume"
		case "make_box":
			targets[fn.Symbol.ID] = "$makeBox"
		case "read":
			targets[fn.Symbol.ID] = "$read"
		}
	}
	declarations, err := RegionTypeDeclarations(compute, receiver)
	if err != nil {
		t.Fatal(err)
	}
	emitter := RegionEmitter{Functions: targets}
	a, err := emitter.Function("$compute", compute)
	if err != nil {
		t.Fatal(err)
	}
	b, err := emitter.Function("$receiver", receiver)
	if err != nil {
		t.Fatal(err)
	}
	runtimeRoot, _ := filepath.Abs("../../../runtime")
	code := `import {strict as assert} from "node:assert";` + "\n" + CompletionImports(filepath.Join(runtimeRoot, "completion.ts")) + fmt.Sprintf("import {assertionContext} from %s;\n", quote(filepath.Join(runtimeRoot, "assert/context.ts"))) + fmt.Sprintf("import {ownCallable as $canOwnCallable, callableReceipt} from %s;\n", quote(filepath.Join(runtimeRoot, "callable.ts"))) + declarations + a + b + `
const root=(name:string)=>assertionContext({package:"p",declaration:"d",name});
const creator=root("creator"), caller=root("caller"), later=root("later");
const callbacks:any[]=[];const observed:any[]=[];
async function $combine(prefix:bigint,value:bigint,suffix:bigint,context?:$canAssertionContext){observed.push(context);return $canSuccess(prefix+value+suffix);}
async function $consume(action:any,value:bigint,context?:$canAssertionContext){assert.equal(context,creator);callbacks.push(action);return action(value,caller);}
assert.equal($canValue(await $compute(3n,5n,4n,creator)),12n);
assert.equal($canValue(await $compute(7n,11n,4n,creator)),22n);
assert.notEqual(callbacks[0],callbacks[1]);
const a=callableReceipt(callbacks[0])!, b=callableReceipt(callbacks[1])!;
assert.equal(a.site,b.site);assert.equal(a.target,b.target);
assert.deepEqual(a.captures,[3n,5n]);assert.deepEqual(b.captures,[7n,11n]);
assert(Object.isFrozen(a));assert(Object.isFrozen(a.captures));
assert.deepEqual(observed,[caller,caller]);
assert.equal($canValue(await callbacks[0](1n,later)),9n);
assert.equal($canValue(await callbacks[1](1n,later)),19n);
assert.deepEqual(observed,[caller,caller,later,later]);
const box=Object.freeze({value:7n});let creations=0;const receivers:any[]=[];
async function $makeBox(){creations++;return $canSuccess(box);}
async function $read(value:any,increment:bigint,context?:$canAssertionContext){assert.equal(context,creator);receivers.push(value);return $canSuccess(value.value+increment);}
assert.equal($canValue(await $receiver(3n,creator)),10n);
assert.equal($canValue(await $receiver(5n,creator)),12n);
assert.equal(creations,2);assert.equal(receivers[0],box);assert.equal(receivers[1],box);
console.log("native closure identities, immutable aliases, capture timing and caller context passed");
`
	file := filepath.Join(t.TempDir(), "closures.ts")
	if err = os.WriteFile(file, []byte(code), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, bun, "--no-install", "--no-env-file", file).CombinedOutput()
	if err != nil || !strings.Contains(string(out), "caller context passed") {
		t.Fatalf("%v\n%s", err, out)
	}
}

func TestDynamicCallablePreparationAndThenField(t *testing.T) {
	bun := os.Getenv("CAN_BUN")
	if bun == "" {
		t.Skip("set CAN_BUN for native dynamic call preparation")
	}
	fixture := newRegionFixture(t)
	factoryType, err := types.CallableOfChecked(fixture.ts["callable int (int) emits []"], nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	fixture.functions["factory"] = check.ValueBinding{Identity: "function/factory", Type: factoryType}
	dynamic, err := fixture.region(t, "    ok call (call factory())(call first())\n", "int", nil, ir.FunctionRegion)
	if err != nil {
		t.Fatal(err)
	}
	then, err := fixture.region(t, "    call make().then()\n    ok 1\n", "int", nil, ir.FunctionRegion)
	if err != nil {
		t.Fatal(err)
	}
	declarations, err := RegionTypeDeclarations(dynamic, then)
	if err != nil {
		t.Fatal(err)
	}
	emitter := RegionEmitter{Functions: map[string]string{"function/factory": "$factory", "function/first": "$argument", "function/make": "$make"}}
	a, err := emitter.Function("$dynamic", dynamic)
	if err != nil {
		t.Fatal(err)
	}
	b, err := emitter.Function("$then", then)
	if err != nil {
		t.Fatal(err)
	}
	runtimeRoot, _ := filepath.Abs("../../../runtime")
	code := `import {strict as assert} from "node:assert";` + "\n" + CompletionImports(filepath.Join(runtimeRoot, "completion.ts")) + declarations + a + b + `
let stage="ok";const events:string[]=[];
async function $factory(){events.push("callee");if(stage==="callee")throw new Error("callee failure");return $canSuccess(async(value:bigint)=>{events.push("invoke");return $canSuccess(value+1n);});}
async function $argument(){events.push("argument");if(stage==="argument")throw new Error("argument failure");return $canSuccess(4n);}
assert.equal($canValue(await $dynamic()),5n);assert.deepEqual(events,["callee","argument","invoke"]);
events.length=0;stage="callee";assert.equal((await $dynamic()).kind,"standard");assert.deepEqual(events,["callee"]);
events.length=0;stage="argument";assert.equal((await $dynamic()).kind,"standard");assert.deepEqual(events,["callee","argument"]);
let thenCalls=0;
async function $make(){return $canSuccess(Object.freeze({then:async()=>{thenCalls++;return $canSuccess(undefined);}}));}
assert.equal($canValue(await $then()),1n);assert.equal(thenCalls,1);
console.log("dynamic preparation and explicit then-field call passed");
`
	file := filepath.Join(t.TempDir(), "dynamic.ts")
	if err = os.WriteFile(file, []byte(code), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, bun, "--no-install", "--no-env-file", file).CombinedOutput()
	if err != nil || !strings.Contains(string(out), "then-field call passed") {
		t.Fatalf("%v\n%s", err, out)
	}
}
