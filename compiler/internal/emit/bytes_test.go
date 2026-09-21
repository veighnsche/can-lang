package emit

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
)

func TestBytesThroughCanCaptureAndTransportFixture(t *testing.T) {
	bun := os.Getenv("CAN_BUN")
	if bun == "" {
		t.Skip("set CAN_BUN for native bytes integration")
	}
	program := sourceProgram(t, "../../testdata/current/bytes/roundtrip.can")
	var captured, roundtrip *ir.Region
	targets := map[string]string{"can.std.bytes@1::from_utf8": "$encode", "can.std.bytes@1::to_utf8": "$transport"}
	for _, fn := range program.Functions {
		switch fn.Symbol.Name {
		case "captured":
			captured = fn.Region
			targets[fn.Symbol.ID] = "$captured"
		case "callable_roundtrip":
			roundtrip = fn.Region
		}
	}
	declarations, err := RegionTypeDeclarations(captured, roundtrip)
	if err != nil {
		t.Fatal(err)
	}
	emitter := RegionEmitter{Functions: targets}
	a, err := emitter.Function("$captured", captured)
	if err != nil {
		t.Fatal(err)
	}
	b, err := emitter.Function("$roundtrip", roundtrip)
	if err != nil {
		t.Fatal(err)
	}
	runtimeRoot, _ := filepath.Abs("../../../runtime")
	code := `import {strict as assert} from "node:assert";` + "\n" + CompletionImports(filepath.Join(runtimeRoot, "completion.ts")) + fmt.Sprintf("import {ownCallable as $canOwnCallable} from %s;\nimport {ownBytes,copyBytes} from %s;\n", quote(filepath.Join(runtimeRoot, "callable.ts")), quote(filepath.Join(runtimeRoot, "bytes.ts"))) + declarations + a + b + `
const origin={source:"test:transport",start:0,end:0,invocation:[]};
let original:any;let received=0;
async function $encode(text:string) {original=ownBytes(new TextEncoder().encode(text));return $canSuccess(original);}
async function $transport(bytes:any) {
 assert.equal(bytes,original);received++;
 const request=new Request("https://fixture.invalid/echo",{method:"POST",body:new Uint8Array(copyBytes(bytes,origin))});
 const sent=new Uint8Array(await request.arrayBuffer());
 assert.equal(new TextDecoder().decode(sent),"héllo 😀");
 const response=new Response(sent,{status:200});
 const reply=ownBytes(new Uint8Array(await response.arrayBuffer()));
 sent.fill(0);
 assert.equal(new TextDecoder().decode(copyBytes(original,origin)),"héllo 😀");
 return $canSuccess(new TextDecoder("utf-8",{fatal:true,ignoreBOM:true}).decode(copyBytes(reply,origin)));
}
assert.equal($canValue(await $roundtrip("héllo 😀")),"héllo 😀");assert.equal(received,1);
`
	file := filepath.Join(t.TempDir(), "bytes.ts")
	if err = os.WriteFile(file, []byte(code), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, bun, "--no-install", file)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("native bytes capture/transport: %v\n%s", err, output)
	}
}
