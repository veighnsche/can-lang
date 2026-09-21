package emit

import (
	"context"
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

func TestEmittedFailureEvaluationOrder(t *testing.T) {
	bun := os.Getenv("CAN_BUN")
	if bun == "" {
		t.Skip("set CAN_BUN to the qualified absolute runtime")
	}
	binary, err := os.ReadFile(bun)
	if err != nil || !filepath.IsAbs(bun) || distribution.Hash(binary) != distribution.PinnedTarget().Runtime.SHA256 {
		t.Fatal("wrong qualified runtime")
	}
	runtimeRoot, _ := filepath.Abs("../../../runtime")
	var bundle string
	if archive := os.Getenv("CAN_BUN_ARCHIVE"); archive != "" {
		sourceRoot, _ := filepath.Abs("../../..")
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		bundle, err = distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "completion-order-test")
		if err != nil {
			t.Fatal(err)
		}
		runtimeRoot = filepath.Join(bundle, "runtime")
		bun = filepath.Join(runtimeRoot, "bun")
	}
	for name, body := range map[string]string{
		"binary":      "    ok call first() + call increment(1)\n",
		"wildcard":    "    match (call first())\n        _ => ok 1\n",
		"array":       "    ok [call first(), call increment(1)].length\n",
		"comparison":  "    match (call first() is call increment(1))\n        true => ok 1\n        false => ok 0\n",
		"logical":     "    match (call truth() and (call increment(1) is 2))\n        true => ok 1\n        false => ok 0\n",
		"value-match": "    int selected = match (call first())\n        _ => 1\n    ok selected\n",
		"constructor": "    ok left(call first()).value + call increment(1)\n",
		"scrutinees":  "    match (call first()), (call increment(1))\n        _, _ => ok 1\n",
	} {
		t.Run(name, func(t *testing.T) {
			f := newRegionFixture(t)
			region, err := f.region(t, body, "int", nil, ir.FunctionRegion)
			if err != nil {
				t.Fatal(err)
			}
			functions := map[string]string{}
			for name, b := range f.functions {
				functions[b.Identity] = name
			}
			emitter := RegionEmitter{Functions: functions}
			code, err := emitter.Function("test", region)
			if err != nil {
				t.Fatal(err)
			}
			root := runtimeRoot
			var all []*types.Type
			for _, v := range f.ts {
				all = append(all, v)
			}
			decls, err := NativeTypeDeclarations(all)
			if err != nil {
				t.Fatal(err)
			}
			program := DataImports(filepath.Join(root, "data.ts")) + CompletionImports(filepath.Join(root, "completion.ts")) + PatternImports(filepath.Join(root, "data.ts"), filepath.Join(root, "failure.ts")) + PrimitiveImports(filepath.Join(root, "primitive.ts")) + decls + `
const events:string[]=[];
async function first(){events.push('first');throw new Error('first fault')}
async function truth(){events.push('first');throw new Error('first fault')}
async function increment(n:bigint){events.push('increment');return $canSuccess(n+1n)}
` + code + `
const result=await test();
console.log(JSON.stringify({kind:result.kind,events}));
if(result.kind!=='standard'||events.join(',')!=='first')throw new Error('failure order violated');
`
			file := filepath.Join(t.TempDir(), "review.ts")
			if err := os.WriteFile(file, []byte(program), 0600); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(bun, "--no-env-file", "--no-macros", "--no-install", file)
			if bundle != "" {
				cmd = exec.Command("/usr/bin/sandbox-exec", "-p", "(version 1)(allow default)(deny network*)", bun, "--no-env-file", "--no-macros", "--no-install", "--config="+filepath.Join(bundle, "tools/runtime/bunfig.toml"), file)
			}
			cmd.Dir = t.TempDir()
			cmd.Env = []string{"HOME=" + cmd.Dir, "PATH=/nonexistent"}
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Errorf("%s\n%s\n%s", err, output, code)
			} else {
				t.Log(strings.TrimSpace(string(output)))
			}
		})
	}
}
func TestRepeatedScrutineeNarrowing(t *testing.T) {
	f := newRegionFixture(t)
	for _, body := range []string{
		"    match choice, choice\n        left, left => ok choice.value\n        _, _ => ok 0\n",
		"    match (choice), choice\n        left, left => ok choice.value\n        _, _ => ok 0\n",
		"    match choice, choice\n        left, _ => ok choice.value\n        _, _ => ok 0\n",
		"    match choice, choice\n        _, left => ok choice.value\n        _, _ => ok 0\n",
	} {
		if _, err := f.region(t, body, "int", nil, ir.FunctionRegion); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := f.region(t, "    match choice, choice\n        left, right => ok 1\n        _, _ => ok 0\n", "int", nil, ir.FunctionRegion); err == nil {
		t.Fatal("incompatible repeated nominal narrowings accepted")
	}
}
