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
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// bypassBun resolves the Bun binary for authored-bypass execution. These
// contracts never skip: Bun is installed and required.
func bypassBun(t *testing.T) string {
	t.Helper()
	if bun := os.Getenv("CAN_BUN"); bun != "" {
		return bun
	}
	bun, err := exec.LookPath("bun")
	if err != nil {
		t.Fatal("bun is required for authored bypass execution")
	}
	return bun
}

func bypassRun(t *testing.T, name, code string) string {
	t.Helper()
	file := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(file, []byte(code), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, bypassBun(t), "--no-install", "--no-env-file", file).CombinedOutput()
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	return string(out)
}

func TestBypassEligibilityPredicate(t *testing.T) {
	plain := func() *ir.InvocationStep {
		return &ir.InvocationStep{Identity: "function/pair", Site: "s1"}
	}
	proof := map[string]string{"function/pair": "$pair", "function/doubled<int>": "$doubledInt"}
	emitter := RegionEmitter{
		Functions:     map[string]string{"function/pair": "$pair"},
		authoredProof: proof,
	}
	if !emitter.eligibleAuthoredBypass(plain(), "$pair") {
		t.Fatal("exact identity+binding proof did not qualify")
	}
	generic := &ir.InvocationStep{Identity: "function/doubled<int>", Site: "s2"}
	if !emitter.eligibleAuthoredBypass(generic, "$doubledInt") {
		t.Fatal("exact generic instance proof did not qualify")
	}
	cases := map[string]struct {
		emitter RegionEmitter
		step    *ir.InvocationStep
		target  string
	}{
		"nil proof":            {RegionEmitter{}, plain(), "$pair"},
		"empty proof":          {RegionEmitter{authoredProof: map[string]string{}}, plain(), "$pair"},
		"stale binding":        {RegionEmitter{authoredProof: map[string]string{"function/pair": "$stale"}}, plain(), "$pair"},
		"unknown identity":     {emitter, &ir.InvocationStep{Identity: "function/other", Site: "s1"}, "$pair"},
		"same spelling source": {emitter, &ir.InvocationStep{Identity: "other/pair", Site: "s1"}, "$pair"},
		"cross-paired binding": {emitter, plain(), "$other"},
		"empty identity":       {emitter, &ir.InvocationStep{Site: "s1"}, "$pair"},
		"empty target":         {emitter, plain(), ""},
		"nil step":             {emitter, nil, "$pair"},
		"browser":              {RegionEmitter{Browser: true, authoredProof: proof}, plain(), "$pair"},
		"callee":               {emitter, &ir.InvocationStep{Identity: "function/pair", Site: "s1", Callee: &ir.Expression{}}, "$pair"},
		"native":               {emitter, &ir.InvocationStep{Identity: "function/pair", Site: "s1", Native: &ir.Expression{}}, "$pair"},
		"array":                {emitter, &ir.InvocationStep{Identity: "function/pair", Site: "s1", Array: &ir.ArrayOperation{}}, "$pair"},
		"asset":                {emitter, &ir.InvocationStep{Identity: "function/pair", Site: "s1", Asset: &ir.AssetResolution{}}, "$pair"},
		"fixtures":             {emitter, &ir.InvocationStep{Identity: "function/pair", Site: "s1", Fixtures: &ir.FixtureTable{}}, "$pair"},
		"sql":                  {emitter, &ir.InvocationStep{Identity: "function/pair", Site: "s1", SQL: &ir.SQLCallSite{}}, "$pair"},
		"form action":          {emitter, &ir.InvocationStep{Identity: "function/pair", Site: "s1", FormAction: &ir.FormActionSite{}}, "$pair"},
		"json fetch":           {emitter, &ir.InvocationStep{Identity: "function/pair", Site: "s1", JSONFetch: &ir.JSONFetchSite{}}, "$pair"},
		"action":               {emitter, &ir.InvocationStep{Identity: "function/pair", Site: "s1", Action: &ir.ActionSite{}}, "$pair"},
	}
	for name, tc := range cases {
		if tc.emitter.eligibleAuthoredBypass(tc.step, tc.target) {
			t.Fatalf("%s qualified for authored bypass", name)
		}
	}
	if len(proof) != 2 || proof["function/pair"] != "$pair" || proof["function/doubled<int>"] != "$doubledInt" {
		t.Fatalf("shared proof mutated: %v", proof)
	}
}

// bypassPairBody emits one region calling pair(first(), captured, 2) so the
// defined branch, argument order and single evaluation are all observable.
func bypassPairBody(t *testing.T, emitter RegionEmitter) (string, *ir.Region) {
	t.Helper()
	fixture := newRegionFixture(t)
	region, err := fixture.region(t, "    ok call pair(call first(), 2)\n", "int", nil, ir.FunctionRegion)
	if err != nil {
		t.Fatal(err)
	}
	body, err := emitter.Function("$bypass", region)
	if err != nil {
		t.Fatal(err)
	}
	return body, region
}

func TestBypassEmittedBranchShape(t *testing.T) {
	functions := map[string]string{"function/pair": "$pair", "function/first": "$first"}
	legacyEmitter := RegionEmitter{Functions: functions, SourceID: "region.can"}
	legacy, _ := bypassPairBody(t, legacyEmitter)
	if strings.Contains(legacy, "=== undefined ?") {
		t.Fatalf("legacy lowering gained a bypass branch:\n%s", legacy)
	}
	proofEmitter := RegionEmitter{Functions: functions, SourceID: "region.can", authoredProof: map[string]string{"function/pair": "$pair"}}
	body, _ := bypassPairBody(t, proofEmitter)
	ternary := strings.Index(body, "($canContext === undefined ? ")
	if ternary < 0 {
		t.Fatalf("eligible call missed strict bypass branch:\n%s", body)
	}
	// The defined branch is the byte-identical original lowering: extract
	// the legacy invocation and require it verbatim after the branch.
	needle := "() => $canCallContext($canContext,"
	at := strings.LastIndex(legacy, needle)
	if at < 0 {
		t.Fatalf("legacy shape unrecognized:\n%s", legacy)
	}
	rest := legacy[at+len("() => "):]
	end := len(rest)
	for _, terminator := range []string{", ($canOrigin_", ", {source:"} {
		if at := strings.Index(rest, terminator); at >= 0 && at < end {
			end = at
		}
	}
	if end == len(rest) {
		t.Fatalf("legacy invocation unterminated:\n%s", legacy)
	}
	original := rest[:end]
	if !strings.Contains(original, "($canContext) => $pair(") || !strings.Contains(original, "$canCallableInstance($pair)") {
		t.Fatalf("legacy extraction missed defined route:\n%s", legacy)
	}
	if !strings.Contains(body, " : "+original+")") {
		t.Fatalf("defined branch differs from original lowering:\n%s", body)
	}
	direct := body[ternary+len("($canContext === undefined ? "):]
	direct = direct[:strings.Index(direct, " : ")]
	if !strings.HasPrefix(direct, "$pair(") || !strings.HasSuffix(direct, ", $canContext)") {
		t.Fatalf("direct branch is not the original call:\n%s", body)
	}
	if strings.Contains(direct, "$canCallContext") || strings.Contains(direct, "$canCallableInstance") {
		t.Fatalf("direct branch kept assertion work:\n%s", body)
	}
	if got := strings.Count(body, "$canCallableInstance($pair)"); got != 1 {
		t.Fatalf("pair receipt lookup evaluated %d times, want once in defined branch:\n%s", got, body)
	}
	if got := strings.Count(body, "$first("); got != 1 {
		t.Fatalf("argument call evaluated %d times, want exactly once:\n%s", got, body)
	}
	if !(strings.Index(body, "$first(") < ternary) {
		t.Fatalf("argument lowered after invocation:\n%s", body)
	}
	if !strings.Contains(body, "await $canInvoke(() => ($canContext === undefined ? ") {
		t.Fatalf("branch escaped the original invoke thunk:\n%s", body)
	}
	// Paired mapping markers surround the call with authored spans.
	if got := strings.Count(body, "\x00"); got < 4 || got%2 != 0 {
		t.Fatalf("mapping marks unpaired (%d):\n%s", got, body)
	}
	_, mappings, err := extractMappings(body)
	if err != nil {
		t.Fatal(err)
	}
	source := "package app\n    provides []\n    uses []\nfn int run\n    emits {}\n    asserts\n        test: => ok 1\n    ok call pair(call first(), 2)\n"
	seenCall := false
	for _, mapping := range mappings {
		if mapping.Operation != "call" || mapping.Source != "region.can" {
			continue
		}
		if mapping.Start < 0 || mapping.End > len(source) || mapping.Start >= mapping.End {
			t.Fatalf("call mapping outside source: %+v", mapping)
		}
		if strings.Contains(source[mapping.Start:mapping.End], "pair(call first()") {
			seenCall = true
		}
	}
	if !seenCall {
		t.Fatalf("no authored call span survived:\n%s", body)
	}
}

func TestBypassFallbackEmission(t *testing.T) {
	functions := map[string]string{"function/pair": "$pair", "function/first": "$first"}
	for name, emitter := range map[string]RegionEmitter{
		"nil proof":   {Functions: functions},
		"empty proof": {Functions: functions, authoredProof: map[string]string{}},
		"stale proof": {Functions: functions, authoredProof: map[string]string{"function/pair": "$stale"}},
	} {
		body, _ := bypassPairBody(t, emitter)
		if strings.Contains(body, "=== undefined ?") {
			t.Fatalf("%s emitted a bypass branch:\n%s", name, body)
		}
		if !strings.Contains(body, "$canCallContext($canContext,") {
			t.Fatalf("%s lost the original route:\n%s", name, body)
		}
	}
	// Browser emission never wraps calls in callContext; proof changes nothing there.
	browserBody, _ := bypassPairBody(t, RegionEmitter{Functions: functions, Browser: true, authoredProof: map[string]string{"function/pair": "$pair"}})
	if strings.Contains(browserBody, "=== undefined ?") || strings.Contains(browserBody, "$canCallContext(") {
		t.Fatalf("browser emission changed under proof:\n%s", browserBody)
	}
	if !strings.Contains(browserBody, "$canCtx") {
		t.Fatalf("browser emission lost explicit owner threading:\n%s", browserBody)
	}
	// Proof is per site: an unproven pair keeps its route while the proven
	// argument call bypasses.
	mixed, _ := bypassPairBody(t, RegionEmitter{Functions: functions, authoredProof: map[string]string{"function/first": "$first"}})
	if strings.Contains(mixed, "($canContext === undefined ? $pair(") {
		t.Fatalf("unproven pair emitted a bypass branch:\n%s", mixed)
	}
	if !strings.Contains(mixed, "($canContext === undefined ? $first(") {
		t.Fatalf("proven argument call missed its bypass branch:\n%s", mixed)
	}
	// Indirect callable values keep the original route even with proof text present.
	fixture := newRegionFixture(t)
	fixture.callables = map[string]check.CallableDeclaration{
		"function/increment": {
			Kind:     "function",
			Contract: fixture.ts["callable int (int) emits {}"],
			Names:    []string{"value"},
			Near:     []bool{false},
		},
	}
	region, err := fixture.region(t, "    callable int (int) emits {} action = callable increment\n    ok call action(4)\n", "int", nil, ir.FunctionRegion)
	if err != nil {
		t.Fatal(err)
	}
	emitter := RegionEmitter{
		Functions:     map[string]string{"function/increment": "$increment"},
		authoredProof: map[string]string{"function/increment": "$increment"},
	}
	body, err := emitter.Function("$indirect", region)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(body, "=== undefined ?") {
		t.Fatalf("indirect call emitted a bypass branch:\n%s", body)
	}
}

func TestBypassExecutesAbsentAndDefinedRoutes(t *testing.T) {
	functions := map[string]string{"function/pair": "$pair", "function/first": "$first"}
	emitter := RegionEmitter{
		Functions:     functions,
		authoredProof: map[string]string{"function/pair": "$pair", "function/first": "$first"},
	}
	body, region := bypassPairBody(t, emitter)
	if !strings.Contains(body, "($canContext === undefined ? $pair(") {
		t.Fatalf("harness region missed bypass branch:\n%s", body)
	}
	declarations, err := RegionTypeDeclarations(region)
	if err != nil {
		t.Fatal(err)
	}
	runtimeRoot, _ := filepath.Abs("../../../runtime")
	contextPath := filepath.Join(runtimeRoot, "assert", "context.ts")
	code := `import {strict as assert} from "node:assert";` + "\n" +
		CompletionImports(filepath.Join(runtimeRoot, "completion.ts")) +
		fmt.Sprintf("import {assertionContext as $canAssertionContext, contextIdentity as $canContextIdentity, closeContext as $canCloseContext} from %s;\n", quote(contextPath)) +
		declarations + body + `
const seen: unknown[] = [];
let pairCalls = 0, firstCalls = 0;
async function $first($canContext: unknown) {
  firstCalls++;
  seen.push($canContext);
  return $canSuccess(3n);
}
async function $pair(a: bigint, b: bigint, $canContext: unknown) {
  pairCalls++;
  seen.push($canContext);
  await new Promise((resolve) => setTimeout(resolve, 5));
  return $canSuccess(a + b);
}
const root = Object.freeze({package: "can.project.root/app", declaration: "can.project.root/app::subject", name: "sample"});
assert.equal($canValue(await $bypass()), 5n);
assert.equal(pairCalls, 1);
assert.equal(firstCalls, 1);
assert.deepEqual(seen, [undefined, undefined]);
const context = $canAssertionContext(root);
const parent = $canContextIdentity(context);
assert.equal($canValue(await $bypass(context)), 5n);
assert.equal(pairCalls, 2);
assert.equal(firstCalls, 2);
const child = seen[3] as object;
assert.equal(typeof child, "object");
assert.notEqual(child, context);
assert.notEqual($canContextIdentity(child), parent);
assert.equal($canContextIdentity(context), parent);
$canCloseContext(context);
const rejected = await $bypass(null as never);
assert.equal(rejected.kind, "standard");
console.log("bypass routes passed");
`
	out := bypassRun(t, "bypass-routes.ts", code)
	if !strings.Contains(out, "bypass routes passed") {
		t.Fatalf("route execution failed:\n%s", out)
	}
}

func TestBypassExecutesFailuresAndAdmission(t *testing.T) {
	fixture := newRegionFixture(t)
	fixture.functions["boom"] = check.ValueBinding{Identity: "function/boom", Type: fixture.ts["callable int () emits {}"]}
	match, err := fixture.region(t, "    match call lookup(-4)\n        missing => ok missing.code\n        ok\n", "int", []string{"missing"}, ir.HandlerRegion)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := fixture.region(t, "    ok call boom()\n", "int", nil, ir.FunctionRegion)
	if err != nil {
		t.Fatal(err)
	}
	proof := map[string]string{"function/lookup": "$lookup"}
	matchEmitter := RegionEmitter{Functions: map[string]string{"function/lookup": "$lookup"}, authoredProof: proof}
	matchBody, err := matchEmitter.Function("$failed", match)
	if err != nil {
		t.Fatal(err)
	}
	rawEmitter := RegionEmitter{Functions: map[string]string{"function/boom": "$boom"}}
	rawBody, err := rawEmitter.Function("$raw", raw)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(matchBody, "($canContext === undefined ? $lookup(") {
		t.Fatalf("proven failure route missed bypass:\n%s", matchBody)
	}
	if strings.Contains(rawBody, "=== undefined ?") {
		t.Fatalf("unproven sync target gained a bypass:\n%s", rawBody)
	}
	declarations, err := RegionTypeDeclarations(match, raw)
	if err != nil {
		t.Fatal(err)
	}
	runtimeRoot, _ := filepath.Abs("../../../runtime")
	path := func(name string) string { p, _ := filepath.Abs(filepath.Join(runtimeRoot, name+".ts")); return p }
	bound, _ := fixture.registry.Bound([]*types.Type{fixture.ts["missing"]})
	plan, _ := json.Marshal(fixture.registry.Plan(bound))
	code := `import {strict as assert} from "node:assert";` + "\n" +
		CompletionImports(path("completion")) +
		PatternImports(path("data"), path("failure")) +
		FailureImports(path("failure")) +
		DataImports(path("data")) +
		PrimitiveImports(path("primitive")) +
		fmt.Sprintf("import {createDomainRuntime} from %s;\n", quote(path("domain"))) +
		fmt.Sprintf("const $canDomain=createDomainRuntime(%s);\n", plan) +
		declarations + matchBody + rawBody + fmt.Sprintf(`
const origin = {source: "stub", start: 0, end: 0, invocation: []};
async function $lookup(n: bigint){return n<0n?$canFailure($canDomain.create(%s,$canRecord(%s,[["code",n]]),origin)):$canSuccess(n)}
assert.equal($canValue(await $failed()), -4n);
let mode = "throw";
function $boom() {
  if (mode === "throw") throw new Error("sync boom");
  return $canSuccess(7n);
}
const first = await $raw();
assert.equal(first.kind, "standard");
const second = await $raw();
assert.equal(second.kind, "standard");
assert.notEqual(first, second);
mode = "return";
assert.equal($canValue(await $raw()), 7n);
console.log("bypass failures passed");
`, quote(fixture.ts["missing"].Identity()), quote(fixture.ts["missing"].Identity()))
	out := bypassRun(t, "bypass-failures.ts", code)
	if !strings.Contains(out, "bypass failures passed") {
		t.Fatalf("failure execution failed:\n%s", out)
	}
}

func TestBypassEligibleCallBoundaryAndBoxedPayloads(t *testing.T) {
	fixture := newRegionFixture(t)
	fixture.functions["probe"] = check.ValueBinding{Identity: "function/probe", Type: fixture.ts["callable int (int) emits {}"]}
	functions := map[string]string{"function/probe": "$probe", "function/log": "$log"}
	proof := map[string]string{"function/probe": "$probe"}
	bodies := map[string]string{
		"$fresh":   "    ok call probe(7)\n",
		"$shareA":  "    ok call probe(9)\n",
		"$shareB":  "    call log()\n    ok call probe(9)\n",
		"$thenP":   "    ok call probe(1)\n",
		"$getterP": "    ok call probe(2)\n",
	}
	header := "package app\n    provides []\n    uses []\nfn int run\n    emits {}\n    asserts\n        test: => ok 1\n"
	spanOf := func(body, call string) (int, int) {
		source := header + body
		start := strings.Index(source, call)
		if start < 0 {
			t.Fatalf("call %q missing from region source", call)
		}
		return start, start + len(call)
	}
	type emitted struct {
		body       string
		region     *ir.Region
		start, end int
	}
	callers := map[string]emitted{}
	var regions []*ir.Region
	for name, text := range bodies {
		region, err := fixture.region(t, text, "int", nil, ir.FunctionRegion)
		if err != nil {
			t.Fatal(err)
		}
		emitter := RegionEmitter{Functions: functions, SourceID: "region.can", authoredProof: proof}
		body, err := emitter.Function(name, region)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(body, "($canContext === undefined ? $probe(") {
			t.Fatalf("%s missed the eligible bypass branch:\n%s", name, body)
		}
		call := text[strings.Index(text, "probe("):]
		call = call[:strings.Index(call, ")")+1]
		start, end := spanOf(text, call)
		callers[name] = emitted{body: body, region: region, start: start, end: end}
		regions = append(regions, region)
	}
	// The pinned call span is neither the function entry nor a literal span.
	_, mappings, err := extractMappings(callers["$fresh"].body)
	if err != nil {
		t.Fatal(err)
	}
	for _, mapping := range mappings {
		if mapping.Source != "region.can" {
			continue
		}
		if mapping.Operation == "call" {
			continue
		}
		if mapping.Start == callers["$fresh"].start && mapping.End == callers["$fresh"].end {
			t.Fatalf("call span collides with %s span: %+v", mapping.Operation, mapping)
		}
	}
	if callers["$shareA"].start == callers["$shareB"].start && callers["$shareA"].end == callers["$shareB"].end {
		t.Fatal("share callers need distinct spans for first-boundary-once")
	}
	declarations, err := RegionTypeDeclarations(regions...)
	if err != nil {
		t.Fatal(err)
	}
	runtimeRoot, _ := filepath.Abs("../../../runtime")
	path := func(name string) string { p, _ := filepath.Abs(filepath.Join(runtimeRoot, name+".ts")); return p }
	origin := func(start, end int) string {
		return fmt.Sprintf(`{source: "region.can", start: %d, end: %d, invocation: ["app::run"]}`, start, end)
	}
	var program strings.Builder
	program.WriteString(`import {strict as assert} from "node:assert";` + "\n")
	program.WriteString(CompletionImports(path("completion")))
	fmt.Fprintf(&program, "import {captureStandard, standardFailureDiagnostics} from %s;\n", quote(path("failure")))
	program.WriteString(declarations)
	for _, name := range []string{"$fresh", "$shareA", "$shareB", "$thenP", "$getterP"} {
		stripped, _, err := extractMappings(callers[name].body)
		if err != nil {
			t.Fatal(err)
		}
		program.WriteString(stripped)
	}
	fmt.Fprintf(&program, `
const synthOrigin = {source: "can:bypass-synth", start: 3, end: 5, invocation: ["synthetic::site"]};
let thenCalls = 0, getterCalls = 0;
const thenPayload: any = {then() { thenCalls++; throw new Error("assimilated"); }};
const getterPayload: any = {};
Object.defineProperty(getterPayload, "then", {
  enumerable: true, configurable: true,
  get() { getterCalls++; throw new Error("assimilated getter"); },
});
let fresh = 0;
const shared = $canFailure(captureStandard("shared", synthOrigin));
async function $log() { return $canSuccess(undefined); }
async function $probe(n: bigint) {
  if (n === 1n) return $canSuccess(thenPayload);
  if (n === 2n) return $canSuccess(getterPayload);
  if (n === 7n) { fresh++; return $canFailure(captureStandard("fresh" + fresh, synthOrigin)); }
  if (n === 9n) return shared;
  return $canSuccess(0n);
}
const freshSpan = %s, shareASpan = %s;
const first = await $fresh(), second = await $fresh();
assert.equal(first.kind, "standard");
assert.equal(second.kind, "standard");
const d1 = standardFailureDiagnostics(first.value), d2 = standardFailureDiagnostics(second.value);
assert.ok(first.value !== second.value);
assert.notEqual(d1.occurrenceID, d2.occurrenceID);
assert.deepEqual(d1.origin, synthOrigin);
assert.deepEqual(d2.origin, synthOrigin);
assert.deepEqual(d1.boundaryOrigin, freshSpan);
assert.deepEqual(d2.boundaryOrigin, freshSpan);
const s1 = await $shareA(), s2 = await $shareB();
assert.ok(s1.value === s2.value, "shared occurrence replaced");
assert.deepEqual(standardFailureDiagnostics(s2.value).boundaryOrigin, shareASpan, "first boundary overwritten");
const hostileThen = await $thenP();
assert.strictEqual($canValue(hostileThen), thenPayload);
const hostileGetter = await $getterP();
assert.strictEqual($canValue(hostileGetter), getterPayload);
assert.equal(thenCalls, 0);
assert.equal(getterCalls, 0);
console.log("bypass boundary passed");
`, origin(callers["$fresh"].start, callers["$fresh"].end), origin(callers["$shareA"].start, callers["$shareA"].end))
	out := bypassRun(t, "bypass-boundary.ts", program.String())
	if !strings.Contains(out, "bypass boundary passed") {
		t.Fatalf("boundary execution failed:\n%s", out)
	}
}
