package emit

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
)

func TestCheckedResourceCaptureRetainsNativeLease(t *testing.T) {
	bun := os.Getenv("CAN_BUN")
	if bun == "" {
		t.Skip("set CAN_BUN for resource capture execution")
	}
	fixture := newRegionFixture(t)
	fixture.values["pool"] = check.ValueBinding{Identity: "value/pool", Type: fixture.ts["sql::pool"]}
	fixture.functions["inspect"] = check.ValueBinding{Identity: "function/inspect", Type: fixture.ts["callable int (sql::pool) emits []"]}
	fixture.callables = map[string]check.CallableDeclaration{"function/inspect": {Kind: "function", Contract: fixture.ts["callable int (sql::pool) emits []"], Names: []string{"pool"}, Near: []bool{true}}}
	region, err := fixture.region(t, "    callable int () emits [] action = callable inspect\n    ok action\n", "callable int () emits []", nil, ir.FunctionRegion)
	if err != nil {
		t.Fatal(err)
	}
	capture := region.Body.Steps[0].Value
	if len(capture.Callable.ResourceCaptures) != 1 || capture.Callable.ResourceCaptures[0] != 0 {
		t.Fatalf("missing checked resource capture evidence: %+v", capture.Callable)
	}
	if got := ir.ResourceCaptureIndices([]*ir.Expression{{Type: fixture.ts["bytes::buffer"]}, {Type: fixture.ts["receipt"]}}); len(got) != 1 || got[0] != 1 {
		t.Fatalf("opaque byte/callable classification: %v", got)
	}
	if got := ir.ResourceCaptureIndices([]*ir.Expression{{Type: fixture.ts["collections::map<int,sql::pool>"]}}); len(got) != 1 || got[0] != 0 {
		t.Fatalf("opaque map capture omitted: %v", got)
	}
	emitter := RegionEmitter{Functions: map[string]string{"function/inspect": "$inspect"}, Bindings: map[string]string{"value/pool": "pool"}}
	declarations, err := RegionTypeDeclarations(region)
	if err != nil {
		t.Fatal(err)
	}
	body, err := emitter.Function("$run", region)
	if err != nil {
		t.Fatal(err)
	}
	runtimeRoot, _ := filepath.Abs("../../../runtime")
	code := `import {strict as assert} from "node:assert";` + "\n" + CompletionImports(filepath.Join(runtimeRoot, "completion.ts")) + fmt.Sprintf("import {ownCallable as $canOwnCallable} from %s;\nimport {runOwnedRoot,registerResource,resourceStatus,closeResource,launchOwned} from %s;\n", quote(filepath.Join(runtimeRoot, "callable.ts")), quote(filepath.Join(runtimeRoot, "owner.ts"))) + declarations + body + `
let pool:any;let observed=0;let closeCalls=0;
async function $inspect(value:any){assert.equal(value,pool);observed++;assert.equal(resourceStatus(pool).leases,1);return $canSuccess(42n);}
const result=await runOwnedRoot(async()=>{
 pool=registerResource("sql.pool",{},()=>{closeCalls++;return $canSuccess(undefined);});
 const callback=$canValue(await $run());assert.equal(resourceStatus(pool).leases,0);
 const group=launchOwned([{captures:[callback],run:()=>callback()}]);
 const completion=await group.promises[0];group.publish([0]);
 assert.equal($canValue(completion),42n);assert.equal(resourceStatus(pool).leases,0);
 await closeResource(pool,"sql.pool");return $canSuccess(undefined);
});
assert.equal(result.cleanupFailed,false);assert.equal(observed,1);assert.equal(closeCalls,1);
`
	file := filepath.Join(t.TempDir(), "resources.ts")
	if err = os.WriteFile(file, []byte(code), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if output, err := exec.CommandContext(ctx, bun, "--no-install", file).CombinedOutput(); err != nil {
		t.Fatalf("native resource captures: %v\n%s", err, output)
	}
}
