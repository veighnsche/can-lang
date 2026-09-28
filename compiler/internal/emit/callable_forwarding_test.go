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
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// forwardingProgram builds a checked Bun program from inline Can sources.
// All sources are owned by the test via t.TempDir; no testdata files are used.
func forwardingProgram(t *testing.T, files map[string]string) *check.Program {
	t.Helper()
	root := t.TempDir()
	all := map[string]string{
		"can.project.json": `{"source_root":"src","error_registry":"can.errors.json"}`,
		"can.errors.json":  `{"active":[],"retired":[]}`,
	}
	for name, text := range files {
		all[name] = text
	}
	for name, text := range all {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0600); err != nil {
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

// forwardingCallableLines returns every emitted $canOwnCallable construction line.
func forwardingCallableLines(text string) []string {
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		if strings.Contains(line, "$canOwnCallable(") {
			lines = append(lines, line)
		}
	}
	return lines
}

func forwardingArtifactCallableLines(artifacts []ir.Artifact) []string {
	var lines []string
	for _, artifact := range artifacts {
		if !strings.HasSuffix(artifact.Path, ".ts") {
			continue
		}
		lines = append(lines, forwardingCallableLines(string(artifact.Bytes))...)
	}
	return lines
}

// forwardingIsDirect reports the G20 selected shape: the forwarding arrow omits
// async but retains the explicit Promise<Completion> annotation.
func forwardingIsDirect(line string) bool {
	return strings.Contains(line, "], (") &&
		strings.Contains(line, "): Promise<$canCompletion<") &&
		!strings.Contains(line, "], async (")
}

func forwardingIsAsync(line string) bool {
	return strings.Contains(line, "], async (") &&
		strings.Contains(line, "): Promise<$canCompletion<")
}

func TestForwardingProofPredicateSelectsExactBinding(t *testing.T) {
	naked := RegionEmitter{Functions: map[string]string{"id/a": "$a"}}
	if naked.provenAuthoredBinding("id/a", "$a") {
		t.Fatal("nil proof qualified for direct forwarding")
	}
	empty := RegionEmitter{
		Functions:     map[string]string{"id/a": "$a"},
		authoredProof: map[string]string{},
	}
	if empty.provenAuthoredBinding("id/a", "$a") {
		t.Fatal("empty proof qualified for direct forwarding")
	}
	emitter := RegionEmitter{
		Functions: map[string]string{
			"id/a":                    "$a",
			"can.std.op@1::x":         "$op",
			"can.intrinsic.op@1::y":   "$native",
			"can.prelude@1::append":   "$append",
			"can.project.root/app::f": "$f",
		},
		authoredProof: map[string]string{"id/a": "$a"},
	}
	if !emitter.provenAuthoredBinding("id/a", "$a") {
		t.Fatal("agreeing checked identity/binding did not qualify")
	}
	for name, tc := range map[string][2]string{
		"unknown identity":     {"id/b", "$a"},
		"rebound binding":      {"id/a", "$stale"},
		"catalogue identity":   {"can.std.op@1::x", "$op"},
		"intrinsic identity":   {"can.intrinsic.op@1::y", "$native"},
		"prelude identity":     {"can.prelude@1::append", "$append"},
		"unproven authored":    {"can.project.root/app::f", "$f"},
		"empty identity":       {"", "$a"},
		"empty binding":        {"id/a", ""},
		"both empty":           {"", ""},
		"cross-paired binding": {"id/a", "$op"},
		"catalogue cross-pair": {"can.std.op@1::x", "$a"},
	} {
		if emitter.provenAuthoredBinding(tc[0], tc[1]) {
			t.Fatalf("%s qualified for direct forwarding", name)
		}
	}
}

func TestForwardingAssemblyProofAndSameSpelling(t *testing.T) {
	generic := forwardingProgram(t, map[string]string{
		"src/lib/lib.can":  "package lib\n    provides [doubled]\n    uses []\nfn item doubled<item>\n    emits []\n    given\n        item value\n        callable item (item, item) emits [] plus\n    asserts\n        triple: 3, callable int_plus => ok 6\n    ok call plus(value, value)\nfn int int_plus\n    emits []\n    given\n        int first\n        int second\n    asserts\n        sample: 3, 3 => ok 6\n    ok first + second\n",
		"src/app/main.can": "package app\n    provides []\n    uses [lib]\nfn int app_plus\n    emits []\n    given\n        int first\n        int second\n    asserts\n        sample: 1, 2 => ok 3\n    ok first + second\nfn int use_doubled\n    emits []\n    asserts\n        sample: => ok 8\n    ok call lib::doubled(4, callable app_plus)\nfn void main\n    emits []\n    given\n        str[] arguments\n    asserts\n        empty: [] => ok\n    ok\n",
	})
	assembly, err := assembleProgramBindings(generic)
	if err != nil {
		t.Fatal(err)
	}
	if len(assembly.authoredProof) != len(generic.Functions) {
		t.Fatalf("proof holds %d entries for %d checked functions", len(assembly.authoredProof), len(generic.Functions))
	}
	seenInstance := false
	for _, fn := range generic.Functions {
		bound, ok := assembly.authoredProof[fn.Identity()]
		if !ok || bound == "" {
			t.Fatalf("missing proof for checked function %s", fn.Identity())
		}
		if bound != assembly.functions[fn.Identity()] {
			t.Fatalf("proof/binding mismatch for %s: proof %s vs resolved %s", fn.Identity(), bound, assembly.functions[fn.Identity()])
		}
		if fn.Instance != "" {
			seenInstance = true
		}
	}
	if !seenInstance {
		t.Fatal("no concrete generic instance in proof")
	}
	checkedCatalogue := false
	for id, bound := range assembly.functions {
		if strings.HasPrefix(id, "can.std.") || strings.HasPrefix(id, "can.intrinsic.") {
			if _, ok := assembly.authoredProof[id]; ok {
				t.Fatalf("catalogue operation %s entered proof", id)
			}
			emitter := RegionEmitter{Functions: assembly.functions, authoredProof: assembly.authoredProof}
			if emitter.provenAuthoredBinding(id, bound) {
				t.Fatalf("catalogue operation %s qualified for direct forwarding", id)
			}
			checkedCatalogue = true
			break
		}
	}
	if !checkedCatalogue {
		t.Fatal("no catalogue operation available for negative proof control")
	}

	spelled := forwardingProgram(t, map[string]string{
		"src/a/first.can":  "package first\n    provides [dup]\n    uses []\nfn int dup\n    emits []\n    given\n        int value\n    asserts\n        sample: 1 => ok 2\n    ok value + 1\n",
		"src/b/second.can": "package second\n    provides [dup]\n    uses []\nfn int dup\n    emits []\n    given\n        int value\n    asserts\n        sample: 2 => ok 4\n    ok value + 2\n",
		"src/app/main.can": "package app\n    provides []\n    uses [first, second]\nfn void main\n    emits []\n    given\n        str[] arguments\n    asserts\n        empty: [] => ok\n    match (call first::dup(1) + call second::dup(2))\n        6 => ok\n        _ => do\n            int invalid = 1 / 0\n            ok\n",
	})
	var first, second *check.ProgramFunction
	for _, fn := range spelled.Functions {
		switch {
		case strings.Contains(fn.Symbol.ID, "first") && strings.HasSuffix(fn.Symbol.ID, "::dup"):
			first = fn
		case strings.Contains(fn.Symbol.ID, "second") && strings.HasSuffix(fn.Symbol.ID, "::dup"):
			second = fn
		}
	}
	if first == nil || second == nil {
		t.Fatal("missing same-spelled dup functions")
	}
	if first.Identity() == second.Identity() {
		t.Fatal("same-spelled identities collide")
	}
	spelledAssembly, err := assembleProgramBindings(spelled)
	if err != nil {
		t.Fatal(err)
	}
	emitter := RegionEmitter{Functions: spelledAssembly.functions, authoredProof: spelledAssembly.authoredProof}
	boundFirst := spelledAssembly.functions[first.Identity()]
	boundSecond := spelledAssembly.functions[second.Identity()]
	if boundFirst == "" || boundSecond == "" {
		t.Fatal("same-spelled functions lack resolved bindings")
	}
	if boundFirst == boundSecond {
		t.Fatal("same-spelled bindings collide")
	}
	if !emitter.provenAuthoredBinding(first.Identity(), boundFirst) ||
		!emitter.provenAuthoredBinding(second.Identity(), boundSecond) {
		t.Fatal("same-spelled functions missed their actual binding")
	}
	if emitter.provenAuthoredBinding(first.Identity(), boundSecond) ||
		emitter.provenAuthoredBinding(second.Identity(), boundFirst) {
		t.Fatal("same-spelled cross-pairing qualified for direct forwarding")
	}
}

const forwardingAuthoredSource = `package app
    provides []
    uses []
fn int add
    emits []
    given
        int first
        int second
    asserts
        sample: 3, 4 => ok 7
    ok first + second
fn int combine
    emits []
    given
        near int prefix
        int value
        near int suffix
    asserts
        sample: 3, 4, 5 => ok 12
    ok prefix + value + suffix
fn item identity<item>
    emits []
    given
        item value
    asserts
        number: 3 => ok 3
    ok value
fn int use_ordinary
    emits []
    asserts
        sample: => ok 7
    callable int (int, int) emits [] action = callable add
    ok call action(3, 4)
fn int use_captured
    emits []
    given
        int prefix
        int suffix
    asserts
        sample: 3, 5 => ok 12
    callable int (int) emits [] action = callable combine
    ok call action(4)
fn int use_generic
    emits []
    asserts
        sample: => ok 7
    callable int (int) emits [] action = callable identity<int>
    ok call action(7)
fn void main
    emits []
    given
        str[] arguments
    asserts
        empty: [] => ok
    match (call use_ordinary() + call use_captured(3, 5) + call use_generic())
        26 => ok
        _ => do
            int invalid = 1 / 0
            ok
`

const forwardingArraySource = `package app
    provides []
    uses []
fn int twice
    emits []
    given
        int value
    asserts
        sample: 3 => ok 6
    ok value * 2
fn int[] referenced_map
    emits []
    given
        int[] items
    asserts
        sample: [1, 2] => ok [2, 4]
    callable int[] (callable int (int) emits []) emits [] action = callable items.map
    ok call action(callable twice)
fn void main
    emits []
    given
        str[] arguments
    asserts
        empty: [] => ok
    int[] got = call referenced_map([1, 2])
    match got.length is 2
        false => do
            int invalid = 1 / 0
            ok
        true => ok
`

const forwardingNativeSource = `package app
    provides []
    uses [codec, bytes]
fn int via_callable
    emits [codec::invalid_data]
    given
        str text
    asserts
        sample: "7" => ok 7
    callable int (bytes::buffer) emits [codec::invalid_data] decode = callable codec::decode_json<int>
    match chain
        call bytes::from_utf8(text) as bytes::buffer encoded
        call decode(encoded) as int decoded
        codec::invalid_data
        ok => ok decoded
fn void main
    emits [codec::invalid_data]
    given
        str[] arguments
    asserts
        empty: [] => ok
    match call via_callable("7")
        codec::invalid_data => do
            int invalid = 1 / 0
            ok
        ok int got => match got is 7
            false => do
                int invalid = 1 / 0
                ok
            true => ok
`

func TestForwardingProductionSelectionOrdinaryGenericCaptured(t *testing.T) {
	authored := forwardingProgram(t, map[string]string{"src/main.can": forwardingAuthoredSource})
	dependencies := httpDependencies(t)
	production, err := ProgramModules(authored, "runtime", dependencies)
	if err != nil {
		t.Fatal(err)
	}
	selected := forwardingArtifactCallableLines(production)
	if len(selected) != 3 {
		t.Fatalf("authored production holds %d callables, want ordinary+captured+generic:\n%s", len(selected), strings.Join(selected, "\n"))
	}
	for _, line := range selected {
		if !forwardingIsDirect(line) {
			t.Fatalf("authored callable missed direct forwarding:\n%s", line)
		}
		for _, want := range []string{
			"): Promise<$canCompletion<",
			"=> $canRegion",
			"$canContext",
			"],$canContext)",
		} {
			if !strings.Contains(line, want) {
				t.Fatalf("direct adapter lost %q:\n%s", want, line)
			}
		}
		if strings.Contains(line, "async (") {
			t.Fatalf("direct adapter kept async:\n%s", line)
		}
	}
	// Generic invoke thunks and native collection lowering survive selection.
	joined := ""
	for _, artifact := range production {
		joined += string(artifact.Bytes) + "\n"
	}
	if !strings.Contains(joined, "async function $canFunction") {
		t.Fatal("production lost native async authored functions")
	}

	arrayed := forwardingProgram(t, map[string]string{"src/main.can": forwardingArraySource})
	arrayProduction, err := ProgramModules(arrayed, "runtime", dependencies)
	if err != nil {
		t.Fatal(err)
	}
	arrayLines := forwardingArtifactCallableLines(arrayProduction)
	if len(arrayLines) != 2 {
		t.Fatalf("array production holds %d callables, want map+twice:\n%s", len(arrayLines), strings.Join(arrayLines, "\n"))
	}
	direct, fallback := 0, 0
	for _, line := range arrayLines {
		switch {
		case forwardingIsDirect(line):
			direct++
		case forwardingIsAsync(line):
			fallback++
		default:
			t.Fatalf("array callable has unknown adapter:\n%s", line)
		}
	}
	if direct != 1 || fallback != 1 {
		t.Fatalf("array production selected direct=%d fallback=%d, want 1+1:\n%s", direct, fallback, strings.Join(arrayLines, "\n"))
	}

	native := forwardingProgram(t, map[string]string{"src/main.can": forwardingNativeSource})
	nativeProduction, err := ProgramModules(native, "runtime", dependencies)
	if err != nil {
		t.Fatal(err)
	}
	nativeLines := forwardingArtifactCallableLines(nativeProduction)
	if len(nativeLines) != 1 {
		t.Fatalf("native production holds %d callables, want decode_json:\n%s", len(nativeLines), strings.Join(nativeLines, "\n"))
	}
	if !forwardingIsAsync(nativeLines[0]) {
		t.Fatalf("catalogue callable selected direct forwarding:\n%s", nativeLines[0])
	}

	// Intentionally inconsistent proof cannot qualify even for an authored target.
	assembly, err := assembleProgramBindings(authored)
	if err != nil {
		t.Fatal(err)
	}
	var captured *check.ProgramFunction
	for _, fn := range authored.Functions {
		if fn.Symbol.Name == "use_captured" {
			captured = fn
		}
	}
	if captured == nil {
		t.Fatal("missing use_captured region")
	}
	stale := map[string]string{}
	for id, bound := range assembly.authoredProof {
		stale[id] = bound
	}
	for _, fn := range authored.Functions {
		if fn.Symbol.Name == "combine" {
			stale[fn.Identity()] = "$staleBinding"
		}
	}
	staleBody, err := (&RegionEmitter{
		Bindings:      assembly.bindings,
		Functions:     assembly.functions,
		DomainRuntime: "$canDomain",
		authoredProof: stale,
	}).Function("$stale", captured.Region)
	if err != nil {
		t.Fatal(err)
	}
	staleLines := forwardingCallableLines(staleBody)
	if len(staleLines) != 1 || !forwardingIsAsync(staleLines[0]) {
		t.Fatalf("stale proof selected direct forwarding:\n%s", staleBody)
	}
	// An emitter without proof retains legacy async behavior.
	legacyBody, err := (&RegionEmitter{Functions: assembly.functions}).Function("$legacy", captured.Region)
	if err != nil {
		t.Fatal(err)
	}
	legacyLines := forwardingCallableLines(legacyBody)
	if len(legacyLines) != 1 || !forwardingIsAsync(legacyLines[0]) {
		t.Fatalf("proof-less emitter left legacy behavior:\n%s", legacyBody)
	}
}

func TestForwardingBrowserAndAssertionSelection(t *testing.T) {
	authored := forwardingProgram(t, map[string]string{"src/main.can": forwardingAuthoredSource})
	dependencies := httpDependencies(t)
	browserArtifacts, err := BrowserModules(authored, "runtime", dependencies)
	if err != nil {
		t.Fatal(err)
	}
	browserLines := forwardingArtifactCallableLines(browserArtifacts)
	if len(browserLines) != 3 {
		t.Fatalf("browser production holds %d callables, want 3:\n%s", len(browserLines), strings.Join(browserLines, "\n"))
	}
	for _, line := range browserLines {
		if !forwardingIsDirect(line) {
			t.Fatalf("browser callable missed direct forwarding:\n%s", line)
		}
		for _, want := range []string{
			"$canCtx: $canOwnerContext",
			"$canContext?: $canAssertionContext",
			"$canCtx",
			"): Promise<$canCompletion<",
		} {
			if !strings.Contains(line, want) {
				t.Fatalf("browser adapter lost %q:\n%s", want, line)
			}
		}
		if strings.Contains(line, "],$canContext)") {
			t.Fatalf("browser adapter threads Bun assertion context:\n%s", line)
		}
		if !strings.HasSuffix(strings.TrimSpace(line), "]);") {
			t.Fatalf("browser adapter lost five-argument owner shape:\n%s", line)
		}
	}

	asserted := forwardingProgram(t, map[string]string{"src/main.can": `package app
    provides []
    uses []
fn int identity_int
    emits []
    given
        int value
    asserts
        sample: 4 => ok 4
    ok value
fn int consume
    emits []
    given
        callable int (int) emits [] action
        int value
    asserts
        sample: callable identity_int, 4 => ok 4
    ok call action(value)
fn void main
    emits []
    given
        str[] arguments
    asserts
        empty: [] => ok
    match (call consume(callable identity_int, 4) + 0)
        4 => ok
        _ => do
            int invalid = 1 / 0
            ok
`})
	assertion, err := AssertionModules(asserted, "runtime", dependencies)
	if err != nil {
		t.Fatal(err)
	}
	assertionLines := forwardingArtifactCallableLines(assertion)
	if len(assertionLines) == 0 {
		t.Fatal("assertion emission holds no callables")
	}
	for _, line := range assertionLines {
		if !forwardingIsDirect(line) {
			t.Fatalf("assertion callable missed direct forwarding:\n%s", line)
		}
		if !strings.Contains(line, "$canContext") {
			t.Fatalf("assertion adapter lost context forwarding:\n%s", line)
		}
	}
	seenCase := false
	for _, artifact := range assertion {
		if strings.HasPrefix(artifact.Path, "assertions/") {
			seenCase = len(forwardingCallableLines(string(artifact.Bytes))) > 0 || seenCase
		}
	}
	if !seenCase {
		t.Fatal("assertion case modules hold no direct callable from the callable assert")
	}
}

func TestForwardingFallbackKeepsAsyncExecutable(t *testing.T) {
	bun := os.Getenv("CAN_BUN")
	if bun == "" {
		t.Skip("set CAN_BUN for fallback execution")
	}
	fixture := newRegionFixture(t)
	fixture.callables = map[string]check.CallableDeclaration{
		"function/increment": {
			Kind:     "function",
			Contract: fixture.ts["callable int (int) emits []"],
			Names:    []string{"value"},
			Near:     []bool{false},
		},
	}
	region, err := fixture.region(t, "    callable int (int) emits [] action = callable increment\n    ok action\n", "callable int (int) emits []", nil, ir.FunctionRegion)
	if err != nil {
		t.Fatal(err)
	}
	emitter := RegionEmitter{Functions: map[string]string{"function/increment": "$increment"}}
	body, err := emitter.Function("$fallback", region)
	if err != nil {
		t.Fatal(err)
	}
	lines := forwardingCallableLines(body)
	if len(lines) != 1 || !forwardingIsAsync(lines[0]) {
		t.Fatalf("unclassified callable missed async fallback:\n%s", body)
	}
	declarations, err := RegionTypeDeclarations(region)
	if err != nil {
		t.Fatal(err)
	}
	runtimeRoot, _ := filepath.Abs("../../../runtime")
	code := `import {strict as assert} from "node:assert";` + "\n" +
		CompletionImports(filepath.Join(runtimeRoot, "completion.ts")) +
		fmt.Sprintf("import {ownCallable as $canOwnCallable} from %s;\n", quote(filepath.Join(runtimeRoot, "callable.ts"))) +
		declarations + body + `
const origin = {source: "fallback.can", start: 0, end: 1, invocation: ["app::run"]};
function $increment(value: bigint) {
  if (value === 4n) return $canSuccess(5n);
  if (value === 5n) throw new Error("boom");
  return $canSuccess(0n);
}
const callback = $canValue(await $fallback());
const syncOk = await $canInvoke(() => callback(4n), origin);
assert.equal($canValue(syncOk), 5n);
const thrown = await $canInvoke(() => callback(5n), origin);
assert.equal(thrown.kind, "standard");
console.log("fallback executable passed");
`
	file := filepath.Join(t.TempDir(), "fallback.ts")
	if err = os.WriteFile(file, []byte(code), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, bun, "--no-install", "--no-env-file", file).CombinedOutput()
	if err != nil || !strings.Contains(string(out), "fallback executable passed") {
		t.Fatalf("%v\n%s", err, out)
	}
}

func TestForwardingSavedTargetCaptureOrderAndResidual(t *testing.T) {
	fixture := newRegionFixture(t)
	three, err := types.CallableOfChecked(fixture.ts["int"], []*types.Type{fixture.ts["int"], fixture.ts["int"], fixture.ts["int"]}, nil)
	if err != nil {
		t.Fatal(err)
	}
	fixture.functions["combine"] = check.ValueBinding{Identity: "function/combine", Type: three}
	fixture.functions["second"] = check.ValueBinding{Identity: "function/second", Type: fixture.ts["callable int () emits []"]}
	fixture.callables = map[string]check.CallableDeclaration{
		"function/combine": {
			Kind:     "function",
			Contract: three,
			Names:    []string{"prefix", "value", "suffix"},
			Near:     []bool{true, false, true},
		},
	}
	region, err := fixture.region(t, "    callable int (int) emits [] action = callable combine with prefix = call first(), suffix = call second()\n    ok call action(4)\n", "int", nil, ir.FunctionRegion)
	if err != nil {
		t.Fatal(err)
	}
	emitter := RegionEmitter{
		Functions: map[string]string{
			"function/combine": "$combine",
			"function/first":   "$first",
			"function/second":  "$second",
		},
		authoredProof: map[string]string{"function/combine": "$combine"},
	}
	body, err := emitter.Function("$ordered", region)
	if err != nil {
		t.Fatal(err)
	}
	lines := forwardingCallableLines(body)
	if len(lines) != 1 || !forwardingIsDirect(lines[0]) {
		t.Fatalf("captured callable missed direct forwarding:\n%s", body)
	}
	saved := strings.Index(body, "const $canRegion")
	firstCall := strings.Index(body, "$first(")
	secondCall := strings.Index(body, "$second(")
	owned := strings.Index(body, "$canOwnCallable(")
	if saved < 0 || firstCall < 0 || secondCall < 0 || owned < 0 {
		t.Fatalf("saved target/capture sequence missing:\n%s", body)
	}
	if !(saved < firstCall && firstCall < secondCall && secondCall < owned) {
		t.Fatalf("capture order broken (saved=%d first=%d second=%d owned=%d):\n%s", saved, firstCall, secondCall, owned, body)
	}
	if got := strings.Count(body, "$first("); got != 1 {
		t.Fatalf("first capture evaluated %d times, want exactly once:\n%s", got, body)
	}
	if got := strings.Count(body, "$second("); got != 1 {
		t.Fatalf("second capture evaluated %d times, want exactly once:\n%s", got, body)
	}
	for _, want := range []string{
		"$canCallableArg0: ",
		"$canContext?: $canAssertionContext",
		"): Promise<$canCompletion<",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("residual signature lost %q:\n%s", want, body)
		}
	}

	bun := os.Getenv("CAN_BUN")
	if bun == "" {
		t.Skip("set CAN_BUN for capture-order execution")
	}
	declarations, err := RegionTypeDeclarations(region)
	if err != nil {
		t.Fatal(err)
	}
	runtimeRoot, _ := filepath.Abs("../../../runtime")
	code := `import {strict as assert} from "node:assert";` + "\n" +
		CompletionImports(filepath.Join(runtimeRoot, "completion.ts")) +
		fmt.Sprintf("import {ownCallable as $canOwnCallable} from %s;\n", quote(filepath.Join(runtimeRoot, "callable.ts"))) +
		declarations + body + `
const calls: string[] = [];
let firstCount = 0, secondCount = 0;
let $combine = async (prefix: bigint, value: bigint, suffix: bigint) => {
  calls.push("target:" + prefix + "," + value + "," + suffix);
  return $canSuccess(prefix + value + suffix);
};
async function $first() {
  firstCount++;
  calls.push("first");
  $combine = (async () => { throw new Error("rebound"); }) as any;
  return $canSuccess(3n);
}
async function $second() {
  secondCount++;
  calls.push("second");
  $combine = (async () => { throw new Error("rebound2"); }) as any;
  return $canSuccess(5n);
}
assert.equal($canValue(await $ordered()), 12n);
assert.deepEqual(calls, ["first", "second", "target:3,4,5"]);
assert.equal(firstCount, 1);
assert.equal(secondCount, 1);
console.log("capture order passed");
`
	file := filepath.Join(t.TempDir(), "capture-order.ts")
	if err = os.WriteFile(file, []byte(code), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, bun, "--no-install", "--no-env-file", file).CombinedOutput()
	if err != nil || !strings.Contains(string(out), "capture order passed") {
		t.Fatalf("%v\n%s", err, out)
	}
}

func TestForwardingReceiptsResourcesDelayedGatesAndLeases(t *testing.T) {
	bun := os.Getenv("CAN_BUN")
	if bun == "" {
		t.Skip("set CAN_BUN for receipt/resource/gate execution")
	}
	runtimeRoot, _ := filepath.Abs("../../../runtime")

	t.Run("receipts", func(t *testing.T) {
		program := forwardingProgram(t, map[string]string{"src/main.can": `package closures
    provides []
    uses []
fn int combine
    emits []
    given
        near int prefix
        int value
        near int suffix
    asserts
        sample: 3, 4, 5 => ok 12
    ok prefix + value + suffix
fn int consume
    emits []
    given
        callable int (int) emits [] action
        int value
    asserts
        sample: callable identity, 4 => ok 4
    ok call action(value)
fn int identity
    emits []
    given
        int value
    asserts
        sample: 4 => ok 4
    ok value
fn int compute
    emits []
    given
        int prefix
        int suffix
        int value
    asserts
        first: 3, 5, 4 => ok 12
        second: 7, 11, 4 => ok 22
    callable int (int) emits [] action = callable combine
    ok call consume(action, value)
fn void main
    emits []
    given
        str[] args
    asserts
        sample: [] => ok
    match (call compute(3, 5, 4) + call compute(7, 11, 4))
        34 => ok
        _ => do
            int invalid = 1 / 0
            ok
`})
		var compute *ir.Region
		targets := map[string]string{}
		proof := map[string]string{}
		for _, fn := range program.Functions {
			switch fn.Symbol.Name {
			case "compute":
				compute = fn.Region
			case "combine":
				targets[fn.Identity()] = "$combine"
				proof[fn.Identity()] = "$combine"
			case "consume":
				targets[fn.Identity()] = "$consume"
			}
		}
		if compute == nil {
			t.Fatal("missing compute region")
		}
		declarations, err := RegionTypeDeclarations(compute)
		if err != nil {
			t.Fatal(err)
		}
		body, err := (&RegionEmitter{Functions: targets, authoredProof: proof}).Function("$compute", compute)
		if err != nil {
			t.Fatal(err)
		}
		lines := forwardingCallableLines(body)
		if len(lines) != 1 || !forwardingIsDirect(lines[0]) {
			t.Fatalf("compute callable missed direct forwarding:\n%s", body)
		}
		code := `import {strict as assert} from "node:assert";` + "\n" +
			CompletionImports(filepath.Join(runtimeRoot, "completion.ts")) +
			fmt.Sprintf("import {assertionContext,contextReport} from %s;\n", quote(filepath.Join(runtimeRoot, "assert/context.ts"))) +
			fmt.Sprintf("import {ownCallable as $canOwnCallable, callableReceipt, callableEqual} from %s;\n", quote(filepath.Join(runtimeRoot, "callable.ts"))) +
			declarations + body + `
const root = (name: string) => assertionContext({package: "p", declaration: "d", name});
const creator = root("creator"), caller = root("caller");
const callbacks: any[] = [];
async function $combine(prefix: bigint, value: bigint, suffix: bigint, context?: any) {
  return $canSuccess(prefix + value + suffix);
}
async function $consume(action: any, value: bigint, context?: any) {
  assert.deepEqual(contextReport(context!).root, contextReport(creator).root);
  callbacks.push(action);
  return action(value, caller);
}
assert.equal($canValue(await $compute(3n, 5n, 4n, creator)), 12n);
assert.equal($canValue(await $compute(3n, 5n, 4n, creator)), 12n);
assert.equal($canValue(await $compute(7n, 11n, 4n, creator)), 22n);
assert.notEqual(callbacks[0], callbacks[1]);
assert.notEqual(callbacks[0], callbacks[2]);
const same = callableEqual(callbacks[0], callbacks[1], (a, b) => a === b);
const diff = callableEqual(callbacks[0], callbacks[2], (a, b) => a === b);
assert.equal(same, true);
assert.equal(diff, false);
for (const cb of callbacks) {
  const receipt = callableReceipt(cb)!;
  assert(Object.isFrozen(cb));
  assert(Object.isFrozen(receipt));
  assert(Object.isFrozen(receipt.captures));
}
assert.equal(callableReceipt(async () => {}) as any, undefined);
console.log("receipts passed");
`
		file := filepath.Join(t.TempDir(), "receipts.ts")
		if err = os.WriteFile(file, []byte(code), 0600); err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		out, err := exec.CommandContext(ctx, bun, "--no-install", "--no-env-file", file).CombinedOutput()
		if err != nil || !strings.Contains(string(out), "receipts passed") {
			t.Fatalf("%v\n%s", err, out)
		}
	})

	t.Run("delayed", func(t *testing.T) {
		fixture := newRegionFixture(t)
		fixture.callables = map[string]check.CallableDeclaration{
			"function/increment": {
				Kind:     "function",
				Contract: fixture.ts["callable int (int) emits []"],
				Names:    []string{"value"},
				Near:     []bool{false},
			},
		}
		region, err := fixture.region(t, "    callable int (int) emits [] action = callable increment\n    ok action\n", "callable int (int) emits []", nil, ir.FunctionRegion)
		if err != nil {
			t.Fatal(err)
		}
		emitter := RegionEmitter{
			Functions:     map[string]string{"function/increment": "$gated"},
			authoredProof: map[string]string{"function/increment": "$gated"},
		}
		body, err := emitter.Function("$gatedFactory", region)
		if err != nil {
			t.Fatal(err)
		}
		if lines := forwardingCallableLines(body); len(lines) != 1 || !forwardingIsDirect(lines[0]) {
			t.Fatalf("gated callable missed direct forwarding:\n%s", body)
		}
		declarations, err := RegionTypeDeclarations(region)
		if err != nil {
			t.Fatal(err)
		}
		code := `import {strict as assert} from "node:assert";` + "\n" +
			CompletionImports(filepath.Join(runtimeRoot, "completion.ts")) +
			fmt.Sprintf("import {ownCallable as $canOwnCallable} from %s;\n", quote(filepath.Join(runtimeRoot, "callable.ts"))) +
			declarations + body + `
const events: string[] = [];
let release!: (value: any) => void;
const gate = new Promise<any>((resolve) => { release = resolve; });
async function $gated(value: bigint) {
  events.push("target-enter");
  const out = await gate;
  events.push("target-exit");
  return out;
}
const action = $canValue(await $gatedFactory());
const pending = action(1n);
assert.ok(pending instanceof Promise);
events.push("after-invoke");
let settled = false;
pending.then(() => { settled = true; });
await Promise.resolve();
await Promise.resolve();
assert.equal(settled, false);
assert.deepEqual(events, ["target-enter", "after-invoke"]);
release($canSuccess(42n));
assert.equal($canValue(await pending), 42n);
assert.deepEqual(events, ["target-enter", "after-invoke", "target-exit"]);
console.log("delayed passed");
`
		file := filepath.Join(t.TempDir(), "delayed.ts")
		if err = os.WriteFile(file, []byte(code), 0600); err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		out, err := exec.CommandContext(ctx, bun, "--no-install", "--no-env-file", file).CombinedOutput()
		if err != nil || !strings.Contains(string(out), "delayed passed") {
			t.Fatalf("%v\n%s", err, out)
		}
	})

	t.Run("resources", func(t *testing.T) {
		fixture := newRegionFixture(t)
		fixture.values["pool"] = check.ValueBinding{Identity: "value/pool", Type: fixture.ts["sql::pool"]}
		fixture.functions["inspect"] = check.ValueBinding{Identity: "function/inspect", Type: fixture.ts["callable int (sql::pool) emits []"]}
		fixture.callables = map[string]check.CallableDeclaration{
			"function/inspect": {
				Kind:     "function",
				Contract: fixture.ts["callable int (sql::pool) emits []"],
				Names:    []string{"pool"},
				Near:     []bool{true},
			},
		}
		region, err := fixture.region(t, "    callable int () emits [] action = callable inspect\n    ok action\n", "callable int () emits []", nil, ir.FunctionRegion)
		if err != nil {
			t.Fatal(err)
		}
		capture := region.Body.Steps[0].Value
		if len(capture.Callable.ResourceCaptures) != 1 || capture.Callable.ResourceCaptures[0] != 0 {
			t.Fatalf("missing checked resource capture evidence: %+v", capture.Callable)
		}
		emitter := RegionEmitter{
			Functions:     map[string]string{"function/inspect": "$inspect"},
			Bindings:      map[string]string{"value/pool": "pool"},
			authoredProof: map[string]string{"function/inspect": "$inspect"},
		}
		body, err := emitter.Function("$run", region)
		if err != nil {
			t.Fatal(err)
		}
		lines := forwardingCallableLines(body)
		if len(lines) != 1 || !forwardingIsDirect(lines[0]) {
			t.Fatalf("resource callable missed direct forwarding:\n%s", body)
		}
		if !strings.Contains(lines[0], ", [0]") {
			t.Fatalf("direct adapter lost resource indices:\n%s", lines[0])
		}
		declarations, err := RegionTypeDeclarations(region)
		if err != nil {
			t.Fatal(err)
		}
		code := `import {strict as assert} from "node:assert";` + "\n" +
			CompletionImports(filepath.Join(runtimeRoot, "completion.ts")) +
			fmt.Sprintf("import {ownCallable as $canOwnCallable} from %s;\nimport {runOwnedRoot,registerResource,resourceStatus,closeResource,launchOwned} from %s;\n", quote(filepath.Join(runtimeRoot, "callable.ts")), quote(filepath.Join(runtimeRoot, "owner.ts"))) +
			declarations + body + `
let pool: any;
let observed = 0;
let closeCalls = 0;
async function $inspect(value: any) {
  assert.equal(value, pool);
  observed++;
  assert.equal(resourceStatus(pool).leases, 1);
  return $canSuccess(42n);
}
const result = await runOwnedRoot(async () => {
  pool = registerResource("sql.pool", {}, () => { closeCalls++; return $canSuccess(undefined); });
  const callback = $canValue(await $run());
  assert.equal(resourceStatus(pool).leases, 0);
  const group = launchOwned([{captures: [callback], run: () => callback()}]);
  const completion = await group.promises[0];
  group.publish([0]);
  assert.equal($canValue(completion), 42n);
  assert.equal(resourceStatus(pool).leases, 0);
  await closeResource(pool, "sql.pool");
  return $canSuccess(undefined);
});
assert.equal(result.cleanupFailed, false);
assert.equal(observed, 1);
assert.equal(closeCalls, 1);
console.log("resources passed");
`
		file := filepath.Join(t.TempDir(), "forward-resources.ts")
		if err = os.WriteFile(file, []byte(code), 0600); err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		out, err := exec.CommandContext(ctx, bun, "--no-install", "--no-env-file", file).CombinedOutput()
		if err != nil || !strings.Contains(string(out), "resources passed") {
			t.Fatalf("%v\n%s", err, out)
		}
	})
}

func TestForwardingFailuresHostilePayloadsAndBoundaries(t *testing.T) {
	bun := os.Getenv("CAN_BUN")
	if bun == "" {
		t.Skip("set CAN_BUN for failure/hostile execution")
	}
	fixture := newRegionFixture(t)
	fixture.callables = map[string]check.CallableDeclaration{
		"function/lookup": {
			Kind:     "function",
			Contract: fixture.ts["callable int (int) emits [missing]"],
			Names:    []string{"value"},
			Near:     []bool{false},
		},
	}
	region, err := fixture.region(t, "    callable int (int) emits [missing] action = callable lookup\n    ok action\n", "callable int (int) emits [missing]", nil, ir.FunctionRegion)
	if err != nil {
		t.Fatal(err)
	}
	emitter := RegionEmitter{
		Functions:     map[string]string{"function/lookup": "$lookup"},
		DomainRuntime: "$canDomain",
		authoredProof: map[string]string{"function/lookup": "$lookup"},
	}
	body, err := emitter.Function("$failingFactory", region)
	if err != nil {
		t.Fatal(err)
	}
	if lines := forwardingCallableLines(body); len(lines) != 1 || !forwardingIsDirect(lines[0]) {
		t.Fatalf("failing callable missed direct forwarding:\n%s", body)
	}
	declarations, err := RegionTypeDeclarations(region)
	if err != nil {
		t.Fatal(err)
	}
	bound, err := fixture.registry.Bound([]*types.Type{fixture.ts["missing"]})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := json.Marshal(fixture.registry.Plan(bound))
	if err != nil {
		t.Fatal(err)
	}
	runtimeRoot, _ := filepath.Abs("../../../runtime")
	path := func(name string) string {
		p, _ := filepath.Abs(filepath.Join(runtimeRoot, name+".ts"))
		return p
	}
	var program strings.Builder
	program.WriteString(`import {strict as assert} from "node:assert";` + "\n")
	program.WriteString(CompletionImports(path("completion")))
	program.WriteString(DataImports(path("data")))
	fmt.Fprintf(&program, "import {ownCallable as $canOwnCallable} from %s;\n", quote(path("callable")))
	fmt.Fprintf(&program, "import {createDomainRuntime} from %s;\n", quote(path("domain")))
	fmt.Fprintf(&program, "import {domainFailureDiagnostics} from %s;\n", quote(path("domain")))
	fmt.Fprintf(&program, "import {captureStandard, standardFailureDiagnostics} from %s;\n", quote(path("failure")))
	fmt.Fprintf(&program, "import {checkedCompletion as $canChecked, isCompletion as $canIsCompletion} from %s;\n", quote(path("completion")))
	fmt.Fprintf(&program, "const $canDomain = createDomainRuntime(%s);\n", plan)
	fmt.Fprintf(&program, "const $canMissing = %s;\n", quote(fixture.ts["missing"].Identity()))
	program.WriteString(declarations)
	program.WriteString(body)
	program.WriteString(`
const origin = {source: "fail.can", start: 0, end: 1, invocation: ["app::run"]};
let thenCalls = 0, getterCalls = 0;
const thenPayload: any = {then() { thenCalls++; throw new Error("assimilated"); }};
const getterPayload: any = {};
Object.defineProperty(getterPayload, "then", {
  enumerable: true, configurable: true,
  get() { getterCalls++; throw new Error("assimilated getter"); },
});
const synthOrigin = {source: "can:forward-synth", start: 7, end: 13, invocation: ["synthetic::site"]};
const synthFailure = captureStandard("probe", synthOrigin);
const synthCarrier = $canFailure(synthFailure);
const synthID = standardFailureDiagnostics(synthFailure).occurrenceID;
async function $lookup(value: bigint) {
  if (value === 1n) return $canSuccess(thenPayload);
  if (value === 2n) return $canSuccess(getterPayload);
  if (value === 3n || value === 4n) return $canCaught(new Error("boom" + value), origin);
  if (value === 5n || value === 6n) {
    const payload = $canRecord($canMissing, [["code", value]]);
    return $canFailure($canDomain.create($canMissing, payload, origin));
  }
  if (value === 7n) return synthCarrier;
  return $canSuccess(0n);
}
const action = $canValue(await $failingFactory());
// Hostile payloads stay boxed without assimilation.
const hostileThen = await $canInvoke(() => action(1n), origin);
assert.strictEqual($canValue(hostileThen), thenPayload);
const hostileGetter = await $canInvoke(() => action(2n), origin);
assert.strictEqual($canValue(hostileGetter), getterPayload);
assert.equal(thenCalls, 0);
assert.equal(getterCalls, 0);
// Fresh standard and domain occurrences.
const stdA = await $canInvoke(() => action(3n), origin);
const stdB = await $canInvoke(() => action(4n), origin);
assert.equal(stdA.kind, "standard");
assert.equal(stdB.kind, "standard");
assert.notEqual(standardFailureDiagnostics(stdA.value).occurrenceID, standardFailureDiagnostics(stdB.value).occurrenceID);
const domA = await $canInvoke(() => action(5n), origin);
const domB = await $canInvoke(() => action(6n), origin);
assert.equal(domA.kind, "domain");
assert.equal(domB.kind, "domain");
assert.notEqual(domainFailureDiagnostics(domA.value).occurrenceID, domainFailureDiagnostics(domB.value).occurrenceID);
// Synthetic failure keeps its original occurrence and location while the
// first mapped authored invoke boundary is recorded exactly once.
const synthOut = await $canInvoke(() => action(7n), origin);
assert.ok(synthOut.value === synthFailure, "synthetic occurrence replaced");
assert.equal(standardFailureDiagnostics(synthOut.value).occurrenceID, synthID);
assert.deepEqual(standardFailureDiagnostics(synthOut.value).origin, synthOrigin);
assert.deepEqual(standardFailureDiagnostics(synthFailure).boundaryOrigin, origin);
await $canInvoke(() => action(7n), {source: "fail.can", start: 9, end: 10, invocation: ["app::run"]});
assert.deepEqual(standardFailureDiagnostics(synthFailure).boundaryOrigin, origin, "first boundary overwritten");
// Carrier refusal never inspects forged getters or proxies.
let forgedGetters = 0;
const forged: any = {};
Object.defineProperty(forged, "kind", {enumerable: true, configurable: true, get() { forgedGetters++; return "ok"; }});
Object.defineProperty(forged, "value", {enumerable: true, configurable: true, get() { forgedGetters++; return 1; }});
assert.equal($canIsCompletion(forged), false);
assert.throws(() => $canChecked(forged));
assert.equal(forgedGetters, 0);
let traps = 0;
const genuine = $canSuccess(1);
const proxy = new Proxy(genuine, {
  get(t, p, r) { traps++; return Reflect.get(t, p, r); },
  getOwnPropertyDescriptor(t, p) { traps++; return Reflect.getOwnPropertyDescriptor(t, p); },
  ownKeys(t) { traps++; return Reflect.ownKeys(t); },
});
assert.equal($canIsCompletion(proxy), false);
assert.throws(() => $canChecked(proxy));
assert.equal(traps, 0);
console.log("failures passed");
`)
	file := filepath.Join(t.TempDir(), "failures.ts")
	if err = os.WriteFile(file, []byte(program.String()), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, bun, "--no-install", "--no-env-file", file).CombinedOutput()
	if err != nil || !strings.Contains(string(out), "failures passed") {
		t.Fatalf("%v\n%s", err, out)
	}
}

func TestForwardingMappingsAndStrictTypes(t *testing.T) {
	program := forwardingProgram(t, map[string]string{"src/main.can": forwardingAuthoredSource})
	var captured *check.ProgramFunction
	for _, fn := range program.Functions {
		if fn.Symbol.Name == "use_captured" {
			captured = fn
		}
	}
	if captured == nil {
		t.Fatal("missing use_captured region")
	}
	assembly, err := assembleProgramBindings(program)
	if err != nil {
		t.Fatal(err)
	}
	sourceID := captured.Symbol.Source.ID
	emitter := RegionEmitter{
		Bindings:      assembly.bindings,
		Functions:     assembly.functions,
		DomainRuntime: "$canDomain",
		SourceID:      sourceID,
		authoredProof: assembly.authoredProof,
	}
	body, err := emitter.Function("$mapped", captured.Region)
	if err != nil {
		t.Fatal(err)
	}
	clean, mappings, err := extractMappings(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(mappings) == 0 {
		t.Fatal("mapped callable lost coverage")
	}
	paired := map[ir.Mapping]int{}
	seenCallable := false
	for _, mapping := range mappings {
		if mapping.Operation == "callable" {
			seenCallable = true
		}
		if mapping.Operation == "function" {
			continue
		}
		key := ir.Mapping{Source: mapping.Source, Start: mapping.Start, End: mapping.End, Operation: mapping.Operation}
		paired[key]++
	}
	if !seenCallable {
		t.Fatal("callable construction lost its mapping")
	}
	for key, count := range paired {
		if count%2 != 0 {
			t.Fatalf("unpaired mapping token %+v appears %d times", key, count)
		}
	}
	lines := forwardingCallableLines(clean)
	if len(lines) != 1 || !forwardingIsDirect(lines[0]) {
		t.Fatalf("mapped callable missed direct forwarding:\n%s", clean)
	}
	for _, want := range []string{
		"$canCallableArg0: ",
		"$canContext?: $canAssertionContext",
		"): Promise<$canCompletion<",
	} {
		if !strings.Contains(clean, want) {
			t.Fatalf("strict adapter lost %q:\n%s", want, clean)
		}
	}
	if strings.Contains(lines[0], "any") {
		t.Fatalf("strict adapter leaks any:\n%s", lines[0])
	}

	tsc, _ := filepath.Abs("../../../node_modules/.bin/tsc")
	if _, err := os.Stat(tsc); err != nil {
		t.Logf("tsc unavailable, strict string checks only: %v", err)
		return
	}
	declarations, err := RegionTypeDeclarations(captured.Region)
	if err != nil {
		t.Fatal(err)
	}
	var combineBinding string
	for _, fn := range program.Functions {
		if fn.Symbol.Name == "combine" {
			combineBinding = assembly.functions[fn.Identity()]
		}
	}
	if combineBinding == "" {
		t.Fatal("missing combine binding for strict stub")
	}
	stub := fmt.Sprintf("declare const %s: (prefix: bigint, value: bigint, suffix: bigint, $canContext?: $canAssertionContext) => Promise<$canCompletion<bigint>>;\n", combineBinding)
	runtimeRoot, _ := filepath.Abs("../../../runtime")
	checked := CompletionImports(filepath.Join(runtimeRoot, "completion.ts")) +
		fmt.Sprintf("import {ownCallable as $canOwnCallable} from %s;\n", quote(filepath.Join(runtimeRoot, "callable.ts"))) +
		declarations + stub + clean +
		"export {};\n"
	dir := t.TempDir()
	file := filepath.Join(dir, "adapter.ts")
	if err = os.WriteFile(file, []byte(checked), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, tsc,
		"--ignoreConfig", "--strict", "--noEmit", "--skipLibCheck",
		"--target", "ESNext", "--module", "Preserve", "--moduleResolution", "bundler",
		"--allowImportingTsExtensions", "--types", "bun,node", file)
	// Inherit the package working directory so bun/node type roots resolve
	// from the checkout; the checked file path is absolute.
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("strict tsc rejected the direct adapter: %v\n%s", err, out)
	}
}
