package emit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/types"
	"github.com/veighnsche/can-lang/distribution"
)

func TestGeneratedCoordinationPreparesWholeChains(t *testing.T) {
	bun := os.Getenv("CAN_BUN")
	runtimeRoot, _ := filepath.Abs("../../../runtime")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	var bundle string
	if archive := os.Getenv("CAN_BUN_ARCHIVE"); archive != "" {
		sourceRoot, _ := filepath.Abs("../../..")
		var err error
		bundle, err = distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "coordination-chain-test")
		if err != nil {
			t.Fatal(err)
		}
		runtimeRoot = filepath.Join(bundle, "runtime")
		bun = filepath.Join(runtimeRoot, "bun")
	}
	if bun == "" {
		t.Skip("set CAN_BUN to qualified runtime")
	}
	binary, err := os.ReadFile(bun)
	if err != nil || !filepath.IsAbs(bun) || distribution.Hash(binary) != distribution.PinnedTarget().Runtime.SHA256 {
		t.Fatal("incorrect native runtime")
	}
	f := newRegionFixture(t)
	actions, err := types.ArrayOfChecked(f.ts["callable int () emits []"])
	if err != nil {
		t.Fatal(err)
	}
	spread := f.functions["first"]
	spread.Identity, spread.Type = "value/actions", actions
	f.values["actions"] = spread
	spread.Identity = "value/empty_actions"
	f.values["empty_actions"] = spread
	region, err := f.region(t, `    int[] results = match call concurrent
        build().bump(call first()).read()
            ok int value => ok value
        increment(call first())
            ok int value => ok value
        ...actions
            ok int value => ok value
        missing => ok [missing.code]
        other => ok []
        [_] => ok [99]
    ok results
`, "int[]", nil, ir.FunctionRegion)
	if err != nil {
		t.Fatal(err)
	}
	functions := map[string]string{}
	for name, value := range f.functions {
		functions[value.Identity] = name
	}
	emitter := RegionEmitter{Bindings: map[string]string{"value/actions": "actions"}, Functions: functions, DomainRuntime: "$canDomain", SourceID: "coordination.can"}
	code, err := emitter.Function("coordinate", region)
	if err != nil {
		t.Fatal(err)
	}
	code, _, err = extractMappings(code)
	if err != nil {
		t.Fatal(err)
	}
	// Lowering must leave checked invocation steps intact for another emission.
	chain := region.Body.Steps[0].Value.Coordination.Entries[0].Call
	if len(chain.Steps) != 3 || len(chain.Steps[1].Prepare) != 2 || chain.Steps[1].Callee != nil {
		t.Fatal("preparation mutated checked chain")
	}
	emptyRegion, err := f.region(t, `    int result = match call race with error
        ...empty_actions
        ok int value => ok value
    ok result
`, "int", nil, ir.FunctionRegion)
	if err != nil {
		t.Fatal(err)
	}
	emptyEmitter := RegionEmitter{Bindings: map[string]string{"value/empty_actions": "empty_actions"}, SourceID: "empty-race.can"}
	emptyCode, err := emptyEmitter.Function("emptyRace", emptyRegion)
	if err != nil {
		t.Fatal(err)
	}
	emptyCode, _, err = extractMappings(emptyCode)
	if err != nil {
		t.Fatal(err)
	}
	declarations, err := RegionTypeDeclarations(region, emptyRegion)
	if err != nil {
		t.Fatal(err)
	}
	path := func(name string) string { return filepath.Join(runtimeRoot, name+".ts") }
	var program strings.Builder
	program.WriteString(CompletionImports(path("completion")))
	program.WriteString(PatternImports(path("data"), path("failure")))
	program.WriteString(DataImports(path("data")))
	program.WriteString(PrimitiveImports(path("primitive")))
	fmt.Fprintf(&program, "import {settle as $canCoordinateSettle,handle as $canCoordinateHandle} from %s;\n", quote(path("coordination")))
	fmt.Fprintf(&program, "import {runOwnedRoot,type Participant as $canParticipant} from %s;\n", quote(path("owner")))
	fmt.Fprintf(&program, "import {createDomainRuntime} from %s;\n", quote(path("domain")))
	bound, _ := f.registry.Bound([]*types.Type{f.ts["missing"], f.ts["other"]})
	plan, _ := json.Marshal(f.registry.Plan(bound))
	fmt.Fprintf(&program, "const $canDomain=createDomainRuntime(%s);\n", plan)
	program.WriteString(declarations)
	fmt.Fprintf(&program, `
const events:string[]=[];
const assert=(condition:boolean,message:string)=>{if(!condition)throw Error(message)};
const tick=async()=>{for(let i=0;i<30;i++)await Promise.resolve()};
function deferred(){let resolve!:(value:$canCompletion<unknown>)=>void;const promise=new Promise<$canCompletion<unknown>>(r=>{resolve=r});return {promise,resolve}}
let argumentsReady=[deferred(),deferred()],built=deferred(),argumentIndex=0;
const record=$canRecord(%s,[['value',7n]]);
const origin={source:'stub',start:0,end:0,invocation:[]};
async function first(_context?:$canAssertionContext){const index=argumentIndex++;events.push('prepare'+index);const value=await argumentsReady[index].promise;events.push('prepared'+index);return value}
async function build(){events.push('launchA');return built.promise}
async function bump(receiver:unknown,n:bigint){events.push('bump');return n<0n?$canFailure($canDomain.create(%s,$canRecord(%s,[['code',n]]),origin)):$canSuccess(receiver)}
async function read(receiver:{value:bigint}){events.push('read');return $canSuccess(receiver.value)}
async function increment(n:bigint){events.push('launchB');return $canSuccess(n+1n)}
const actions=Object.freeze([async()=>{events.push('launchC');return $canSuccess(5n)},async()=>{events.push('launchD');return $canSuccess(6n)}]);
const empty_actions=Object.freeze([]);
`, quote(f.ts["left"].Identity()), quote(f.ts["missing"].Identity()), quote(f.ts["missing"].Identity()))
	program.WriteString(code)
	program.WriteString(emptyCode)
	program.WriteString(`
for(const scenario of ['success','preparation-failure','chain-failure']){
 events.length=0;argumentIndex=0;argumentsReady=[deferred(),deferred()];built=deferred();
 const root=runOwnedRoot(()=>coordinate());
 await tick();assert(events.join(',')==='prepare0','first preparation not awaited');
 argumentsReady[0].resolve($canSuccess(scenario==='chain-failure'?-1n:3n));
 await tick();assert(events.join(',')==='prepare0,prepared0,prepare1','participant started before all argument preparation');
 argumentsReady[1].resolve(scenario==='preparation-failure'?$canCaught(Error('argument'),origin):$canSuccess(3n));
 await tick();
 if(scenario==='preparation-failure'){
  const result=await root;assert(result.completion.kind==='standard','preparation failure entered participant handler');
  assert(events.join(',')==='prepare0,prepared0,prepare1,prepared1','preparation failure launched a participant');continue;
 }
 assert(events.join(',')==='prepare0,prepared0,prepare1,prepared1,launchA,launchB,launchC,launchD','launch did not follow complete written preparation and spread order');
 built.resolve($canSuccess(record));const result=await root;
 assert(result.completion.kind==='ok','coordination failed');
 const values=$canValue(result.completion) as bigint[];
 assert(values.join(',')===(scenario==='success'?'7,4,5,6':'-1'),'wrong chain result or shared fallback');
 assert(events.includes('bump')&&events.includes('read')===(scenario==='success'),'chain failure executed later method');
}
let emptyCompleted=false;
const emptyRoot=runOwnedRoot(()=>emptyRace()).then(value=>{emptyCompleted=true;return value});
// The timer observes the generated pending operation; it cannot settle or cancel it.
const emptyObservation=await Promise.race([emptyRoot.then(()=>"completed"),new Promise<string>(resolve=>setTimeout(()=>resolve("deadline"),10))]);
assert(emptyObservation==='deadline'&&!emptyCompleted,'generated empty race produced a completion');
console.log('coordination chains passed');
`)
	file := filepath.Join(t.TempDir(), "coordination.ts")
	if err = os.WriteFile(file, []byte(program.String()), 0600); err != nil {
		t.Fatal(err)
	}
	if output := os.Getenv("CAN_COORDINATION_TEST_OUTPUT"); output != "" {
		if err = os.WriteFile(output, []byte(program.String()), 0600); err != nil {
			t.Fatal(err)
		}
	}
	command := exec.CommandContext(ctx, bun, "--no-env-file", "--no-macros", "--no-install", file)
	if bundle != "" {
		command = exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-p", "(version 1)(allow default)(deny network*)", bun, "--no-env-file", "--no-macros", "--no-install", "--config="+filepath.Join(bundle, "tools/runtime/bunfig.toml"), file)
	}
	command.Dir = t.TempDir()
	command.Env = []string{"PATH=/nonexistent", "HOME=" + command.Dir}
	output, err := command.CombinedOutput()
	if err != nil || strings.TrimSpace(string(output)) != "coordination chains passed" {
		t.Fatalf("%v\n%s", err, output)
	}
}
