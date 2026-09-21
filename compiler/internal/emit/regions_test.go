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

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/project"
	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
	"github.com/veighnsche/can-lang/distribution"
)

type regionFixture struct {
	ts        map[string]*types.Type
	registry  *check.ErrorRegistry
	scope     *resolve.Scope
	callables map[string]check.CallableDeclaration
	functions map[string]check.ValueBinding
	values    map[string]check.ValueBinding
}

func newRegionFixture(t *testing.T) *regionFixture {
	t.Helper()
	root := t.TempDir()
	text := `package app
    provides []
    uses [sql, bytes, collections]
error 1000000 missing(int code)
error 1000001 other()
error 1000002 wrapped<item>(item value)
record receipt
    callable void () emits [] then
record left
    int value
record right
    str value
variant either
    left
    right
record node
    node[] children
`
	for name, data := range map[string]string{"can.project.json": `{"source_root":"src","error_registry":"can.errors.json"}`, "can.errors.json": `{"active":[{"id":1000000,"kind":"app::missing"},{"id":1000001,"kind":"app::other"},{"id":1000002,"kind":"app::wrapped"}],"retired":[]}`, "src/main.can": text} {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(data), 0600); err != nil {
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
	registry, err := check.ErrorDeclarations(world)
	if err != nil {
		t.Fatal(err)
	}
	var file *resolve.File
	for _, f := range world.Files {
		file = f
	}
	builder := types.NewBuilder(world)
	if err = builder.SeedDeclarations(); err != nil {
		t.Fatal(err)
	}
	names := []string{"collections::map<int,sql::pool>", "sql::pool", "bytes::buffer", "callable int (sql::pool) emits []", "callable bool () emits []", "callable left () emits []", "callable left (left, int) emits [missing]", "callable int (left) emits [other]", "wrapped<int>", "wrapped<str>", "callable int () emits [wrapped<int>, wrapped<str>]", "int", "float", "str", "bool", "void", "missing", "other", "receipt", "left", "right", "either", "node", "int[]", "bool[]", "node[]", "callable int () emits []", "callable int (int) emits [missing]", "callable void () emits []", "callable receipt () emits []", "callable int (int, int[]) emits []", "callable int (int, int) emits []", "callable int (int) emits []", "callable int () emits [missing, other]"}
	fixture := &regionFixture{ts: map[string]*types.Type{}, registry: registry, scope: file.Scope, functions: map[string]check.ValueBinding{}, values: map[string]check.ValueBinding{}}
	for _, name := range names {
		src, _ := source.New("type.can", name)
		node, ds := syntax.ParseType(src)
		if len(ds) != 0 {
			t.Fatal(ds)
		}
		fixture.ts[name], err = builder.Resolve(file, node, nil, true)
		if err != nil {
			t.Fatal(err)
		}
	}
	if _, err = builder.Finish(); err != nil {
		t.Fatal(err)
	}
	for name, typ := range map[string]string{"truth": "callable bool () emits []", "build": "callable left () emits []", "bump": "callable left (left, int) emits [missing]", "read": "callable int (left) emits [other]", "sum": "callable int (int, int[]) emits []", "pair": "callable int (int, int) emits []", "ambiguous": "callable int () emits [wrapped<int>, wrapped<str>]", "first": "callable int () emits []", "lookup": "callable int (int) emits [missing]", "log": "callable void () emits []", "make": "callable receipt () emits []", "increment": "callable int (int) emits []", "both": "callable int () emits [missing, other]"} {
		fixture.functions[name] = check.ValueBinding{Identity: "function/" + name, Type: fixture.ts[typ]}
	}
	for name, typ := range map[string]string{"number": "int", "flag": "bool", "items": "int[]", "choice": "either", "tree": "node", "payload": "receipt"} {
		fixture.values[name] = check.ValueBinding{Identity: "value/" + name, Type: fixture.ts[typ]}
	}
	return fixture
}
func (f *regionFixture) region(t *testing.T, body, result string, errors []string, kind ir.RegionKind) (*ir.Region, error) {
	t.Helper()
	text := "package app\n    provides []\n    uses []\nfn " + result + " run\n    emits []\n    asserts\n        test: => ok 1\n" + body
	src, _ := source.New("region.can", text)
	parsed := syntax.Parse(src)
	if !parsed.OK() {
		return nil, fmt.Errorf("parse: %v", parsed.Diagnostics)
	}
	function := parsed.File.Declarations[0].(*syntax.FunctionDecl)
	lookupType := func(node syntax.TypeNode, allowVoid bool) (*types.Type, error) {
		typ := f.ts[syntax.FormatType(node)]
		if typ == nil || !allowVoid && typ.Kind() == types.Void {
			return nil, fmt.Errorf("unknown/ineligible annotation %s", syntax.FormatType(node))
		}
		return typ, nil
	}
	expr := &check.Expressions{Scalars: f.ts, Value: func(name syntax.QualifiedName) (check.ValueBinding, error) {
		if b, ok := f.values[name.Name]; ok && name.Package == "" {
			return b, nil
		}
		return check.ValueBinding{}, fmt.Errorf("unknown value %s", name.Name)
	}, Function: func(name syntax.QualifiedName) (check.ValueBinding, error) {
		if b, ok := f.functions[name.Name]; ok && name.Package == "" {
			return b, nil
		}
		return check.ValueBinding{}, fmt.Errorf("unknown function %s", name.Name)
	}, Constructor: func(node *syntax.ConstructorExpr, expected *types.Type, _ *check.Expressions) (*types.Type, error) {
		return lookupType(&syntax.NamedType{Name: node.Name, Arguments: node.Types}, false)
	}}
	expr.Reference = expr.Function
	var boundTypes []*types.Type
	for _, name := range errors {
		boundTypes = append(boundTypes, f.ts[name])
	}
	bound, err := f.registry.Bound(boundTypes)
	if err != nil {
		t.Fatal(err)
	}
	context := check.CompletionContext{Identity: "app::run", Kind: kind, File: src, Scope: f.scope, Result: f.ts[result], Registry: f.registry, Errors: bound, Expressions: expr, Type: lookupType, Callables: f.callables}
	context.Method = func(receiver *types.Type, name syntax.Token, args []syntax.TypeNode) (check.ValueBinding, error) {
		if types.Equal(receiver, f.ts["left"]) && len(args) == 0 && (name.Text == "bump" || name.Text == "read") {
			return f.functions[name.Text], nil
		}
		return check.ValueBinding{}, fmt.Errorf("unknown method")
	}
	context.Variadic = map[string]bool{"function/sum": true}
	context.ErrorName = func(name syntax.QualifiedName) (string, error) {
		for _, d := range f.registry.Declarations() {
			if d.Name == "app::"+name.Name && name.Package == "" {
				return d.Identity, nil
			}
		}
		return "", fmt.Errorf("unknown error")
	}
	context.PatternName = func(name syntax.QualifiedName) (string, error) {
		if typ := f.ts[name.Name]; typ != nil && name.Package == "" {
			return typ.Declaration(), nil
		}
		return context.ErrorName(name)
	}
	if kind == ir.HandlerRegion {
		context.Parent = "app::outer"
	}
	return check.CheckRegion(context, function.Body)
}
func TestCompletionRegionContracts(t *testing.T) {
	f := newRegionFixture(t)
	good := []struct {
		body, result string
		errors       []string
	}{
		{"    ok\n", "void", nil},

		{"    match call build().bump(1).read()\n        ok\n        missing => ok 0\n        other => ok 0\n", "int", nil},
		{"    ok call pair(...[1,2])\n", "int", nil},
		{"    ok call sum(1, ...items)\n", "int", nil},
		{"    ok call sum(...[1,2,3])\n", "int", nil},
		{"    ok (call items.slice(0, 1).slice(0, 1)).length\n", "int", nil},
		{"    match call items.slice(0, 1)\n        ok int[] found => ok found.length\n", "int", nil},
		{"    ok call first() + 1\n", "int", nil},
		{"    relay call lookup(1)\n", "int", []string{"missing"}},
		{"    match call lookup(1)\n        ok\n        missing => ok missing.code\n", "int", nil},
		{"    match chain\n        call lookup(1) as int found\n        call increment(found) as int next\n        call log()\n        ok => ok next\n        missing => ok missing.code\n", "int", nil},
		{"    match flag\n        true => do\n            call log()\n            relay call first()\n        false => ok 0\n", "int", nil},
		{"    int result = match number\n        -5..0 => 1\n        1 | 2 => 2\n        _ => 3\n    ok result + 1\n", "int", nil},
		{"    match items\n        [] => ok 0\n        [head, ...tail] => ok head + tail.length\n", "int", nil},
		{"    match choice\n        left => ok choice.value\n        right(value) => ok value.length\n", "int", nil},
		{"    match tree\n        node([]) => ok 0\n        node([head, ...tail]) => ok head.children.length + tail.length\n", "int", nil},
	}
	for i, tc := range good {
		for _, kind := range []ir.RegionKind{ir.FunctionRegion, ir.HandlerRegion} {
			t.Run(fmt.Sprintf("good%d/%s", i, kind), func(t *testing.T) {
				if _, err := f.region(t, tc.body, tc.result, tc.errors, kind); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
	for i, body := range []string{
		"    match flag, flag\n        true, true => ok 1\n        true, false => ok 2\n        false, _ => ok 3\n",
		"    match items\n        [] => ok 0\n        [0, ...tail] | [1, ...tail] => ok tail.length\n        [head, ...tail] => ok head\n",
		"    match number\n        0..5 => ok 1\n        4..10 => ok 2\n        _ => ok 3\n",
		"    match choice\n        left | right => ok 1\n",
	} {
		if _, err := f.region(t, body, "int", nil, ir.FunctionRegion); err != nil {
			t.Fatalf("product/alternative %d: %v", i, err)
		}
	}
	bad := []string{
		"    ok call build().bump(1).read()\n",
		"    match call build().bump(1).read()\n        ok\n        missing => ok 0\n",
		"    ok call pair(...items)\n",
		"    ok call sum(...items)\n",
		"    ok call pair(...[1,2,3])\n",
		"    match call ambiguous()\n        ok\n        wrapped => ok 0\n",
		"    match flag, flag\n        true, true => ok 1\n        false, _ => ok 2\n",
		"    match flag, flag\n        true, _ => ok 1\n        false, _ => ok 2\n        _, true => ok 3\n",
		"    match number\n        0..5 => ok 1\n        6..10 => ok 2\n        0..10 => ok 3\n        _ => ok 4\n",
		"    match items\n        [head] | [] => ok head\n        _ => ok 0\n",
		"    match choice\n        left(value) | right(value) => ok 1\n",

		"    ok\n",
		"    missing(1)\n",
		"    relay call lookup(1)\n",
		"    ok call lookup(1) + 1\n",
		"    call first()\n    ok 1\n",
		"    match call lookup(1)\n        ok => ok 1\n",
		"    match call first()\n        ok\n        missing => ok 1\n",
		"    match call lookup(1)\n        ok\n        missing\n",
		"    match call lookup(1)\n        ok\n        ok => ok 2\n        missing => ok 3\n",
		"    match call lookup(1)\n        ok str text => ok 1\n        missing => ok 2\n",
		"    match call lookup(1)\n        ok\n        missing => other()\n",
		"    match chain\n        call lookup(1) as int found\n        ok => ok found\n        missing => ok found\n",
		"    match chain\n        call first()\n        ok => ok 1\n",
		"    match flag\n        true => ok 1\n",
		"    match flag\n        _ => ok 1\n        true => ok 2\n",
		"    match number\n        0..10 => ok 1\n        1..5 => ok 2\n        _ => ok 3\n",
		"    match items\n        [head] => ok head\n",
		"    int result = match flag\n        true => ok 1\n        false => 2\n    ok result\n",
		"    int result = 1 + 2\n    ok result\n",
	}
	for i, body := range bad {
		t.Run(fmt.Sprintf("bad%d", i), func(t *testing.T) {
			if _, err := f.region(t, body, "int", nil, ir.FunctionRegion); err == nil {
				t.Fatal("invalid body accepted", body)
			}
		})
	}
	escaped, err := f.region(t, "    match call lookup(1)\n        ok\n        missing => other()\n", "int", []string{"other", "missing"}, ir.HandlerRegion)
	if err != nil || len(escaped.Escapes) != 1 || !types.Equal(escaped.Escapes[0], f.ts["other"]) {
		t.Fatal("handler escape set includes handled participant", err)
	}
	if _, err := f.region(t, "    ok 1\n", "void", nil, ir.FunctionRegion); err == nil {
		t.Fatal("void accepted a value")
	}
}
func TestEmittedCompletionRegions(t *testing.T) {
	bun := os.Getenv("CAN_BUN")
	if bun == "" {
		t.Skip("set CAN_BUN to qualified runtime")
	}
	binary, err := os.ReadFile(bun)
	if err != nil || !filepath.IsAbs(bun) || distribution.Hash(binary) != distribution.PinnedTarget().Runtime.SHA256 {
		t.Fatal("incorrect native runtime")
	}
	runtimeRoot, _ := filepath.Abs("../../../runtime")
	var bundle string
	if archive := os.Getenv("CAN_BUN_ARCHIVE"); archive != "" {
		sourceRoot, _ := filepath.Abs("../../..")
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		bundle, err = distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "completion-test")
		if err != nil {
			t.Fatal(err)
		}
		runtimeRoot = filepath.Join(bundle, "runtime")
		bun = filepath.Join(runtimeRoot, "bun")
	}
	f := newRegionFixture(t)
	path := func(name string) string { p, _ := filepath.Abs(filepath.Join(runtimeRoot, name+".ts")); return p }
	var program strings.Builder
	program.WriteString(CompletionImports(path("completion")))
	program.WriteString(PatternImports(path("data"), path("failure")))
	program.WriteString(DataImports(path("data")))
	program.WriteString(PrimitiveImports(path("primitive")))
	fmt.Fprintf(&program, "import {createDomainRuntime} from %s;\n", quote(path("domain")))
	bound, _ := f.registry.Bound([]*types.Type{f.ts["missing"], f.ts["other"]})
	plan, _ := json.Marshal(f.registry.Plan(bound))
	fmt.Fprintf(&program, "const $canDomain=createDomainRuntime(%s);\n", plan)
	var allTypes []*types.Type
	for _, typ := range f.ts {
		allTypes = append(allTypes, typ)
	}
	decls, err := NativeTypeDeclarations(allTypes)
	if err != nil {
		t.Fatal(err)
	}
	program.WriteString(decls)
	fmt.Fprintf(&program, `let thenCalls=0;const events:string[]=[];
const payload=$canRecord(%s,[["then",async()=>{thenCalls++;return $canSuccess(undefined)}]]);
const number=2n,flag=true,items=Object.freeze([4n,5n]);
const choice=$canRecord(%s,[["value",7n]]);
const tree=$canRecord(%s,[["children",Object.freeze([])]]);
const origin={source:"stub",start:0,end:0,invocation:[]};
async function first(){events.push("first");return $canSuccess(3n)}
async function truth(){events.push("truth");return $canSuccess(true)}
async function lookup(n:bigint){events.push("lookup");return n<0n?$canFailure($canDomain.create(%s,$canRecord(%s,[["code",n]]),origin)):$canSuccess(n)}
async function log(){events.push("log");return $canSuccess(undefined)}
async function increment(n:bigint){events.push("increment");return $canSuccess(n+1n)}
async function make(){return $canSuccess(payload)}
async function build(){return $canSuccess(choice)}
async function bump(receiver:typeof choice,n:bigint){events.push("bump");if(n<0n){const r=await lookup(n);if(r.kind==="ok")throw new Error("bad fixture");return r;}return $canSuccess(receiver)}
async function read(receiver:typeof choice){events.push("read");return $canSuccess(receiver.value as bigint)}
async function pair(a:bigint,b:bigint){return $canSuccess(a+b)}
async function sum(prefix:bigint,rest:readonly bigint[]){return $canSuccess(rest.reduce((a,b)=>a+b,prefix))}
const assert=(ok:boolean,label:string)=>{if(!ok)throw new Error(label)};
`, quote(f.ts["receipt"].Identity()), quote(f.ts["left"].Identity()), quote(f.ts["node"].Identity()), quote(f.ts["missing"].Identity()), quote(f.ts["missing"].Identity()))
	tests := []struct {
		body, result, want string
		errors             []string
	}{
		{"    match choice, choice\n        left, left => ok choice.value\n        _, _ => ok 0\n", "int", "7n", nil},
		{"    match (choice), choice\n        left, left => ok choice.value\n        _, _ => ok 0\n", "int", "7n", nil},

		{"    ok false and call truth()\n", "bool", "false", nil},
		{"    ok 3 < call first() < call first()\n", "bool", "false", nil},

		{"    match call build().bump(1).read()\n        ok\n        missing => ok 0\n        other => ok 0\n", "int", "7n", nil},
		{"    match call build().bump(-1).read()\n        ok\n        missing => ok missing.code\n        other => ok 0\n", "int", "-1n", nil},
		{"    ok call pair(...[call first(), call first()])\n", "int", "6n", nil},
		{"    ok call sum(1, ...items)\n", "int", "10n", nil},
		{"    ok call sum(...[1,2,3])\n", "int", "6n", nil},
		{"    ok call sum(1)\n", "int", "1n", nil},
		{"    ok (call items.slice(0, 1).slice(0, 1)).length\n", "int", "1n", nil},
		{"    match call items.slice(0, 1)\n        ok int[] found => ok found.length\n", "int", "1n", nil},
		{"    ok call first() + call first()\n", "int", "6n", nil},
		{"    relay call make()\n", "receipt", "payload", nil},
		{"    match call lookup(-4)\n        ok\n        missing => ok missing.code\n", "int", "-4n", nil},
		{"    match call lookup(1 / 0)\n        ok\n        missing => ok 0\n        [_] as str message => ok message.length\n", "int", "BigInt('arithmetic: integer division by zero'.length)", nil},
		{"    match chain\n        call lookup(8) as int found\n        call increment(found) as int next\n        call log()\n        ok => ok next\n        missing => ok 0\n", "int", "9n", nil},
		{"    match flag\n        true => do\n            call log()\n            match call first()\n                ok int found => ok found + 1\n        false => ok 0\n", "int", "4n", nil},
		{"    int result = match number\n        -5..0 => 1\n        1 | 2 => 2\n        _ => 3\n    ok result + 1\n", "int", "3n", nil},
		{"    match items\n        [] => ok 0\n        [head, ...tail] => ok head + tail.length\n", "int", "5n", nil},
		{"    match choice\n        left => ok choice.value\n        right(value) => ok value.length\n", "int", "7n", nil},
		{"    match tree\n        node([]) => ok 0\n        node([head, ...tail]) => ok head.children.length + tail.length\n", "int", "0n", nil},

		{"    match items\n        [] => ok 0\n        [0, ...tail] | [4, ...tail] => ok tail.length\n        [head, ...tail] => ok head\n", "int", "1n", nil},
		{"    match flag, flag\n        true, true => ok 1\n        true, false => ok 2\n        false, _ => ok 3\n", "int", "1n", nil},
		{"    match choice\n        left | right => ok 1\n", "int", "1n", nil},
		{"    call log()\n    ok\n", "void", "undefined", nil},
	}
	for i, tc := range tests {
		region, err := f.region(t, tc.body, tc.result, tc.errors, ir.HandlerRegion)
		if err != nil {
			t.Fatal(i, err)
		}
		bindings := map[string]string{}
		for name, b := range f.values {
			bindings[b.Identity] = name
		}
		functions := map[string]string{}
		for name, b := range f.functions {
			functions[b.Identity] = name
		}
		emitter := RegionEmitter{Bindings: bindings, Functions: functions, DomainRuntime: "$canDomain"}
		name := fmt.Sprintf("test%d", i)
		code, err := emitter.Function(name, region)
		if err != nil {
			t.Fatal(i, err)
		}
		program.WriteString(code)
		fmt.Fprintf(&program, "events.length=0;assert($canValue(await %s()) === %s, %s);\n", name, tc.want, quote(name))
		if strings.Contains(tc.body, "false and") {
			program.WriteString("assert(events.length===0,'logical rhs was not skipped');\n")
		}
		if strings.Contains(tc.body, "3 < call first()") {
			program.WriteString("assert(events.join(',')==='first','comparison did not short circuit');\n")
		}
		if strings.Contains(tc.body, "bump(-1)") {
			program.WriteString("assert(events.join(',')==='bump,lookup','later method launched after failure');\n")
		}
		if strings.Contains(tc.body, "pair(...[") {
			program.WriteString("assert(events.join(',')==='first,first','spread evaluated more than once');\n")
		}
		if strings.Contains(tc.body, "lookup(1 / 0)") {
			program.WriteString("assert(!events.includes('lookup'),'call launched after argument fault');\n")
		}

	}

	failureCases := []struct{ body, check string }{
		{"    match call lookup(1)\n        ok => missing(9)\n        missing => ok 0\n", "r.kind==='domain' && ($canErrorPayload(r) as {code:bigint}).code===9n"},
		{"    match call lookup(-1)\n        ok\n        missing => match call first()\n            ok => missing(7)\n", "r.kind==='domain' && ($canErrorPayload(r) as {code:bigint}).code===7n"},
		{"    match call lookup(1 / 0)\n        ok\n        missing => ok 0\n        [_] => ok 2 / 0\n", "r.kind==='standard'"},
		{"    match chain\n        call lookup(-1) as int found\n        call increment(found) as int next\n        ok => ok next\n        missing\n", "r.kind==='domain' && !events.includes('increment')"},
	}
	for i, tc := range failureCases {
		region, err := f.region(t, tc.body, "int", []string{"missing"}, ir.HandlerRegion)
		if err != nil {
			t.Fatal(err)
		}
		functions := map[string]string{}
		for name, b := range f.functions {
			functions[b.Identity] = name
		}
		emitter := RegionEmitter{Functions: functions, DomainRuntime: "$canDomain"}
		name := fmt.Sprintf("failed%d", i)
		code, err := emitter.Function(name, region)
		if err != nil {
			t.Fatal(err)
		}
		program.WriteString(code)
		fmt.Fprintf(&program, "events.length=0;{const r=await %s();assert(%s,%s)}\n", name, tc.check, quote(name))
	}
	program.WriteString("assert(thenCalls===0,'then data assimilated');console.log('completion regions passed');\n")
	file := filepath.Join(t.TempDir(), "regions.ts")
	if err = os.WriteFile(file, []byte(program.String()), 0600); err != nil {
		t.Fatal(err)
	}
	if output := os.Getenv("CAN_REGION_TEST_OUTPUT"); output != "" {
		if err = os.WriteFile(output, []byte(program.String()), 0600); err != nil {
			t.Fatal(err)
		}
	}
	command := exec.Command(bun, "--no-env-file", "--no-macros", "--no-install", file)
	if bundle != "" {
		command = exec.Command("/usr/bin/sandbox-exec", "-p", "(version 1)(allow default)(deny network*)", bun, "--no-env-file", "--no-macros", "--no-install", "--config="+filepath.Join(bundle, "tools/runtime/bunfig.toml"), file)
	}
	command.Dir = t.TempDir()
	command.Env = []string{"HOME=" + command.Dir, "PATH=/nonexistent"}
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("%v\n%s\n%s", err, output, program.String())
	}
	if strings.TrimSpace(string(output)) != "completion regions passed" {
		t.Fatal(string(output))
	}
}

func TestRegionTypesAndOwnership(t *testing.T) {
	f := newRegionFixture(t)
	array, err := types.ArrayOfChecked(f.ts["receipt"])
	if err != nil {
		t.Fatal(err)
	}
	f.ts["receipt[]"] = array
	region, err := f.region(t, "    ok [payload]\n", "receipt[]", nil, ir.HandlerRegion)
	if err != nil {
		t.Fatal(err)
	}
	declarations, err := RegionTypeDeclarations(region)
	if err != nil {
		t.Fatal(err)
	}
	for _, typ := range []*types.Type{array, f.ts["receipt"], f.ts["void"]} {
		if !strings.Contains(declarations, "type "+TypeName(typ)+" =") {
			t.Fatal("derived/reachable type omitted")
		}
	}
	region.Body.Terminal.RegionID = region.Parent
	emitter := RegionEmitter{Bindings: map[string]string{"value/payload": "payload"}}
	if _, err = emitter.Function("wrong", region); err == nil {
		t.Fatal("foreign region return accepted")
	}
}
