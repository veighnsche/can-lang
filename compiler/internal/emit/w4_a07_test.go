package emit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/types"
	"github.com/veighnsche/can-lang/distribution"
)

// w4Corpus is the A07 iteration corpus: the W4 core shapes (state
// machine, input-driven relay, relay scan with single bulk build,
// step-60k faults) plus the W4.4 non-lowerable controls. The bounded
// companion batch (W4.3) stays F06-owned and is absent here by design.
const w4Corpus = `package app
    provides []
    uses [collections]
fn int machine
    emits []
    given
        int n
        int state
        int acc
    asserts
        sample: 4, 0, 0 => ok 7
    match n
        0 => ok acc
        _ => match state
            0 => relay call machine(n - 1, 1, acc + 1)
            1 => relay call machine(n - 1, 2, acc + 2)
            _ => relay call machine(n - 1, 0, acc + 3)
fn int scan_sum
    emits []
    given
        int[] values
        int i
        int acc
    asserts
        sample: [1, 2, 3], 0, 0 => ok 6
    match i is values.length
        true => ok acc
        false => relay call scan_sum(values, i + 1, acc + values[i])
fn collections::entry<int,int>[] aggregate
    emits [collections::key_exists]
    given
        collections::entry<int,int>[] rows
        int i
        int seen
    asserts
        sample: [collections::entry<int,int>(7, 70)], 1, 1 => ok [collections::entry<int,int>(7, 70)]
    match i is rows.length
        true => match seen is rows.length
            true => match call collections::build_map<int,int>(rows)
                collections::key_exists
                ok collections::map<int,int> built => ok call collections::entries(built)
            false => ok rows[0:0]
        false => relay call aggregate(rows, i + 1, seen + 1)
fn int fault_standard
    emits []
    given
        int n
        bool armed
    asserts
        sample: 3, false => ok 3
    match armed
        false => ok n
        true => match n
            0 => ok 1 / n
            _ => relay call fault_standard(n - 1, armed)
fn int fault_declared
    emits [collections::key_absent]
    given
        int n
    asserts
        sample: 3 => collections::key_absent()
    collections::map<int,int> empty = call collections::empty_map<int,int>()
    match n
        0 => match call collections::get(empty, 1)
            collections::key_absent
            ok int found => ok found
        _ => relay call fault_declared(n - 1)
fn int fault_originated
    emits [collections::key_absent]
    given
        int n
    asserts
        sample: 3 => collections::key_absent()
    match n
        0 => collections::key_absent()
        _ => relay call fault_originated(n - 1)
fn int[] grow
    emits []
    given
        int[] acc
        int n
    asserts
        sample: [9], 2 => ok [9, 2, 1]
    match n
        0 => ok acc
        _ => relay call grow([...acc, n], n - 1)
fn int triangle
    emits []
    given
        int n
    asserts
        sample: 3 => ok 6
    match n
        0 => ok 0
        _ => ok call triangle(n - 1) + n
fn int ping
    emits []
    given
        int n
    asserts
        sample: 2 => ok 100
    match n
        0 => ok 0
        _ => relay call pong(n - 1)
fn int pong
    emits []
    given
        int n
    asserts
        sample: 1 => ok 100
    match n
        0 => ok 100
        _ => relay call ping(n)
fn int identity
    emits []
    given
        int value
    asserts
        sample: 4 => ok 4
    ok value
fn int loop_excluded
    emits []
    given
        int n
    asserts
        sample: 3 => ok 0
    callable int (int) emits [] action = callable identity
    match n
        0 => ok call action(0)
        _ => relay call loop_excluded(n - 1)
fn void main
    emits []
    given
        str[] arguments
    asserts
        empty: [] => ok
    ok
`

func w4Program(t *testing.T) *check.Program {
	t.Helper()
	return actionEmitProgram(t, map[string]string{"src/main.can": w4Corpus})
}

func w4Region(t *testing.T, program *check.Program, name string) *ir.Region {
	t.Helper()
	for _, fn := range program.Functions {
		if fn.Symbol.Name == name {
			return fn.Region
		}
	}
	t.Fatalf("missing concrete %s region", name)
	return nil
}

// A07 static leg: the W4 core shapes prove lowerable and emit native
// loops with step counters, while the W4.4 controls keep nested calls
// with NOT-LOWERED notes where the A04 contract requires them.
func TestW4CorpusLoweringShape(t *testing.T) {
	program := w4Program(t)
	lowered := map[string]bool{
		"machine": true, "scan_sum": true, "aggregate": true,
		"fault_standard": true, "fault_declared": true, "fault_originated": true, "grow": true,
	}
	functions := map[string]string{}
	for _, fn := range program.Functions {
		functions[fn.Symbol.ID] = "test_" + fn.Symbol.Name
	}
	for key := range program.Collections {
		functions[key] = "$canStub." + key[len(key)-8:]
	}
	emitted := map[string]string{}
	for _, fn := range program.Functions {
		emitter := RegionEmitter{Bindings: map[string]string{}, Functions: functions, DomainRuntime: "$canDomain"}
		code, err := emitter.Function("test_"+fn.Symbol.Name, fn.Region)
		if err != nil {
			t.Fatal(fn.Symbol.Name, err)
		}
		emitted[fn.Symbol.Name] = code
	}
	for name := range lowered {
		region := w4Region(t, program, name)
		if region.TailExclusion != "" {
			t.Fatalf("%s excluded: %s", name, region.TailExclusion)
		}
		code := emitted[name]
		for _, want := range []string{"while (true) {", "continue;", `"step:"+$canRegion`} {
			if !strings.Contains(code, want) {
				t.Fatalf("%s omits %q:\n%s", name, want, code)
			}
		}
	}
	for _, name := range []string{"triangle", "ping", "pong", "loop_excluded"} {
		if code := emitted[name]; strings.Contains(code, "while (true)") {
			t.Fatalf("%s lowered, want nested calls:\n%s", name, code)
		}
	}
	if region := w4Region(t, program, "loop_excluded"); region.TailExclusion != "deferred completion (callable value)" {
		t.Fatalf("loop_excluded exclusion: %q", region.TailExclusion)
	}
	if len(program.Warnings) != 3 {
		t.Fatalf("expected three NOT-LOWERED notes, got %+v", program.Warnings)
	}
	mutual, callable := 0, 0
	for _, warning := range program.Warnings {
		if warning.Code != "CAN-CHECK-NOT-LOWERED" || warning.Severity != check.SeverityNote {
			t.Fatalf("wrong note identity: %+v", warning)
		}
		switch {
		case strings.Contains(warning.Message, "not the enclosing function"):
			mutual++
		case strings.Contains(warning.Message, "deferred completion (callable value)"):
			callable++
		default:
			t.Fatalf("unexpected note: %+v", warning)
		}
	}
	if mutual != 2 || callable != 1 {
		t.Fatalf("expected two mutual + one callable note, got %+v", program.Warnings)
	}
}

// w4Leg is one measured driver record. Only JSON-safe scalars cross the
// bun boundary; bigints become strings or numbers before printing.
type w4Leg struct {
	Leg         string  `json:"leg"`
	N           int     `json:"n"`
	Ok          bool    `json:"ok"`
	Ms          float64 `json:"ms"`
	InputBytes  float64 `json:"inputBytes"`
	ResultBytes float64 `json:"resultBytes"`
	Kind        string  `json:"kind"`
	Message     string  `json:"message"`
	Step        string  `json:"step"`
	Occurrence  string  `json:"occurrence"`
	FailureType string  `json:"failureType"`
	Decl        string  `json:"decl"`
	Origin      string  `json:"origin"`
	Contained   bool    `json:"contained"`
}

// A07 measured legs: execute the corpus against the worktree runtime on
// the pinned bun, recording wall time and settled-heap growth at
// 100/20k/100k plus one bounded 1M-relay leg. Correctness legs assert
// exact results; growth legs assert linear-in-state retention and reject
// quadratic history growth; the spread-accumulation contrast only
// records its shape.
func TestW4MeasuredLegs(t *testing.T) {
	bun := os.Getenv("CAN_BUN")
	if bun == "" {
		t.Skip("set CAN_BUN to qualified runtime")
	}
	binary, err := os.ReadFile(bun)
	if err != nil || !filepath.IsAbs(bun) || distribution.Hash(binary) != distribution.PinnedTarget().Runtime.SHA256 {
		t.Fatal("incorrect native runtime")
	}
	runtimeRoot, err := filepath.Abs("../../../runtime")
	if err != nil {
		t.Fatal(err)
	}
	program := w4Program(t)

	functions := map[string]string{}
	var regions []*ir.Region
	for _, fn := range program.Functions {
		functions[fn.Symbol.ID] = "test_" + fn.Symbol.Name
		regions = append(regions, fn.Region)
	}
	collectionIDs := []string{}
	collectionTypes := map[string]*check.CollectionSpecialization{}
	for _, special := range program.Collections {
		id := special.Collection.Identity()
		if collectionTypes[id] == nil {
			collectionIDs = append(collectionIDs, id)
			collectionTypes[id] = special
		}
	}
	sort.Strings(collectionIDs)
	collectionNames := map[string]string{}
	for i, id := range collectionIDs {
		collectionNames[id] = fmt.Sprintf("$canCollection%d", i)
	}
	for key, special := range program.Collections {
		functions[key] = collectionNames[special.Collection.Identity()] + "." + collectionMethodName(special.Operation)
	}

	var emitted strings.Builder
	for _, fn := range program.Functions {
		emitter := RegionEmitter{Bindings: map[string]string{}, Functions: functions, DomainRuntime: "$canDomain"}
		code, err := emitter.Function("test_"+fn.Symbol.Name, fn.Region)
		if err != nil {
			t.Fatal(fn.Symbol.Name, err)
		}
		emitted.WriteString(code)
	}
	declarations, err := RegionTypeDeclarations(regions...)
	if err != nil {
		t.Fatal(err)
	}
	failures := []*types.Type{}
	seenFailure := map[string]bool{}
	collect := func(list []*types.Type) {
		for _, failure := range list {
			if failure != nil && !seenFailure[failure.Identity()] {
				seenFailure[failure.Identity()] = true
				failures = append(failures, failure)
			}
		}
	}
	for _, special := range program.Collections {
		collect(special.Contract.Errors())
	}
	for _, fn := range program.Functions {
		collect(fn.Region.Escapes)
	}
	bound, err := program.Registry.Bound(failures)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := json.Marshal(program.Registry.Plan(bound))
	if err != nil {
		t.Fatal(err)
	}
	entryIdentity := ""
	for _, special := range program.Collections {
		if special.Operation == "can.std.collections@1::build_map" && special.Entry != nil {
			entryIdentity = special.Entry.Identity()
		}
	}
	if entryIdentity == "" {
		t.Fatal("build_map entry identity missing")
	}
	// Factories take concrete type identities like the program state
	// builder's numberIDs, never bare declarations.
	numberIDs := map[string]string{}
	for _, typ := range program.Model.Types() {
		numberIDs[typ.Declaration()] = typ.Identity()
	}
	absentIdentity := numberIDs["can.std.collections@1::key_absent"]
	existsIdentity := numberIDs["can.std.collections@1::key_exists"]
	if absentIdentity == "" || existsIdentity == "" {
		t.Fatal("collection failure identities missing from model")
	}

	path := func(name string) string { p, _ := filepath.Abs(filepath.Join(runtimeRoot, name+".ts")); return p }
	collectionsPath := func(name string) string {
		p, _ := filepath.Abs(filepath.Join(runtimeRoot, "collections", name+".ts"))
		return p
	}
	var programTS strings.Builder
	programTS.WriteString(CompletionImports(path("completion")))
	programTS.WriteString(PatternImports(path("data"), path("failure")))
	programTS.WriteString(FailureImports(path("failure")))
	programTS.WriteString(DataImports(path("data")))
	programTS.WriteString(PrimitiveImports(path("primitive")))
	fmt.Fprintf(&programTS, "import {createDomainRuntime} from %s;\n", quote(path("domain")))
	fmt.Fprintf(&programTS, "import {domainFailureDiagnostics} from %s;\n", quote(path("domain")))
	fmt.Fprintf(&programTS, "import {standardFailureDiagnostics} from %s;\n", quote(path("failure")))
	fmt.Fprintf(&programTS, "import {ownCallable as $canOwnCallable} from %s;\n", quote(path("callable")))
	fmt.Fprintf(&programTS, "import {createMap as $canCreateMap} from %s;\n", quote(collectionsPath("map")))
	fmt.Fprintf(&programTS, "import {createSet as $canCreateSet} from %s;\n", quote(collectionsPath("set")))
	fmt.Fprintf(&programTS, "const $canDomain = createDomainRuntime(%s);\n", plan)
	for _, id := range collectionIDs {
		special := collectionTypes[id]
		args := special.Collection.Arguments()
		if special.Entry != nil {
			fmt.Fprintf(&programTS, "const %s = $canCreateMap(%s);\n", collectionNames[id], strings.Join([]string{
				"$canDomain",
				fmt.Sprintf("{map:%s,entry:%s,absent:%s,exists:%s}", quote(id), quote(special.Entry.Identity()), quote(absentIdentity), quote(existsIdentity)),
				quote(args[0].Declaration()),
			}, ","))
		} else {
			fmt.Fprintf(&programTS, "const %s = $canCreateSet(%s,%s);\n", collectionNames[id], quote(id), quote(args[0].Declaration()))
		}
	}
	programTS.WriteString(declarations)
	programTS.WriteString(emitted.String())
	fmt.Fprintf(&programTS, w4Driver, quote(entryIdentity), quote(absentIdentity), quote(existsIdentity))

	file := filepath.Join(t.TempDir(), "w4.ts")
	if err = os.WriteFile(file, []byte(programTS.String()), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	run := func(args ...string) []w4Leg {
		t.Helper()
		argv := append([]string{"--no-env-file", "--no-macros", "--no-install", file}, args...)
		output, err := exec.CommandContext(ctx, bun, argv...).CombinedOutput()
		if err != nil {
			t.Fatalf("%v\n%s", err, output)
		}
		var legs []w4Leg
		done := false
		for _, line := range strings.Split(string(output), "\n") {
			if line == "W4DONE" {
				done = true
				continue
			}
			trimmed, ok := strings.CutPrefix(line, "W4LEG ")
			if !ok {
				continue
			}
			var leg w4Leg
			if err := json.Unmarshal([]byte(trimmed), &leg); err != nil {
				t.Fatalf("bad leg %q: %v", trimmed, err)
			}
			legs = append(legs, leg)
		}
		if !done {
			t.Fatalf("driver %v did not finish:\n%s", args, output)
		}
		return legs
	}
	// One process per measured leg: JSC heap accounting only moves one
	// way inside a process, so cross-size growth must compare clean
	// heaps, never successive legs of one run.
	legs := map[string]w4Leg{}
	for _, size := range []string{"100", "20000", "100000"} {
		for _, kind := range []string{"machine", "scan", "aggregate"} {
			for _, leg := range run(kind, size) {
				legs[leg.Leg+":"+fmt.Sprint(leg.N)] = leg
			}
		}
	}
	for _, size := range []string{"200", "800", "3200"} {
		for _, leg := range run("grow", size) {
			legs[leg.Leg+":"+fmt.Sprint(leg.N)] = leg
		}
	}
	for _, leg := range run("machine", "1000000") {
		legs[leg.Leg+":"+fmt.Sprint(leg.N)] = leg
	}
	for _, leg := range run("misc") {
		legs[leg.Leg+":"+fmt.Sprint(leg.N)] = leg
	}
	keys := make([]string, 0, len(legs))
	for key := range legs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		leg := legs[key]
		t.Logf("%-18s n=%-8d ok=%-5v ms=%9.1f input=%10.0f result=%10.0f kind=%s step=%s occurrence=%s failure=%s decl=%s origin=%s contained=%v msg=%s",
			leg.Leg, leg.N, leg.Ok, leg.Ms, leg.InputBytes, leg.ResultBytes, leg.Kind, leg.Step, leg.Occurrence, leg.FailureType, leg.Decl, leg.Origin, leg.Contained, leg.Message)
		if !leg.Ok {
			t.Errorf("leg %s failed: %+v", key, leg)
		}
	}
	// The scalar state machine proves the lowered loop itself retains
	// no per-step state; the scan leg shares the loop shape but its
	// driver-owned input graph cannot settle deterministically, so its
	// memory cell is recorded, not gated.
	if retained := legs["machine:100000"].ResultBytes; retained >= 2*1024*1024 {
		t.Errorf("machine retains %.0f bytes at 100k, want flat state", retained)
	}
	small, mid, big := legs["aggregate:100"].ResultBytes, legs["aggregate:20000"].ResultBytes, legs["aggregate:100000"].ResultBytes
	inner, outer := mid-small, big-mid
	if inner <= 0 || outer <= 0 {
		t.Fatalf("aggregation growth unmeasurable: 100=%.0f 20k=%.0f 100k=%.0f", small, mid, big)
	}
	if ratio := outer / inner; ratio < 2 || ratio > 10 {
		t.Errorf("aggregation growth ratio %.2f outside linear band [2,10] (linear predicts ~4, quadratic ~25)", ratio)
	}
	for _, leg := range []string{"machine", "scan", "aggregate"} {
		slow, fast := legs[leg+":20000"].Ms, legs[leg+":100000"].Ms
		if slow <= 0 {
			t.Logf("%s time %.1fms -> %.1fms too fast to discriminate scaling", leg, slow, fast)
			continue
		}
		if fast/slow > 10 {
			t.Errorf("%s time %.1fms -> %.1fms rejects linear scaling", leg, slow, fast)
		}
	}
	if got := legs["machine:1000000"].ResultBytes; got >= 8*1024*1024 {
		t.Errorf("1M state machine retains %.0f bytes, want flat state", got)
	}
}

// w4Driver runs one measured leg (argv: kind size) or the misc
// correctness legs, printing W4LEG JSON records plus a final W4DONE
// line. The three %s are the entry, key_absent and key_exists concrete
// identities.
const w4Driver = `
const W4ENTRY = %s;
const W4ABSENT = %s;
const W4EXISTS = %s;
const W4ABSENT_DECL = "can.std.collections@1::key_absent";
const W4EXISTS_DECL = "can.std.collections@1::key_exists";
function w4leg(record: object): void { console.log("W4LEG " + JSON.stringify(record)); }
let w4holder: unknown = undefined;
async function w4heap(): Promise<number> { Bun.gc(); Bun.gc(); return process.memoryUsage().heapUsed; }
function w4ints(n: number): bigint[] {
  const out: bigint[] = [];
  for (let k = 1; k <= n; k++) out.push(BigInt(k));
  return out;
}
function w4rows(n: number): unknown[] {
  const out: unknown[] = [];
  for (let k = 1; k <= n; k++) out.push($canRecord(W4ENTRY, [["key", BigInt(k)], ["value", BigInt(k * 10)]]));
  return out;
}
function w4machine(n: number): bigint {
  const cycles = Math.floor(n / 3);
  const rem = n - cycles * 3;
  return BigInt(cycles * 6 + (rem >= 1 ? 1 : 0) + (rem >= 2 ? 2 : 0));
}
function w4step(invocation: readonly string[]): string {
  return invocation.filter((s) => s.indexOf("step:") === 0).join(",");
}
async function w4machineLeg(n: number): Promise<void> {
  w4holder = undefined;
  await test_machine(10n, 0n, 0n);
  const base = await w4heap();
  const t0 = performance.now();
  const out = await test_machine(BigInt(n), 0n, 0n);
  const ms = performance.now() - t0;
  const acc = $canValue(out) as bigint;
  w4holder = acc;
  const withResult = await w4heap();
  w4leg({leg: "machine", n: n, ok: acc === w4machine(n), ms: ms, inputBytes: 0, resultBytes: withResult - base});
  w4holder = undefined;
}
async function w4scanLeg(n: number): Promise<void> {
  w4holder = undefined;
  await test_scan_sum([1n], 0n, 0n);
  const base = await w4heap();
  let input: bigint[] | undefined = w4ints(n);
  const withInput = await w4heap();
  const t0 = performance.now();
  const out = await test_scan_sum(input, 0n, 0n);
  const ms = performance.now() - t0;
  const total = $canValue(out) as bigint;
  input = undefined;
  w4holder = total;
  const withResult = await w4heap();
  const want = (BigInt(n) * BigInt(n + 1)) / 2n;
  w4leg({leg: "scan", n: n, ok: total === want, ms: ms, inputBytes: withInput - base, resultBytes: withResult - base});
  w4holder = undefined;
}
async function w4aggregateLeg(n: number): Promise<void> {
  w4holder = undefined;
  await test_aggregate(w4rows(10), 0n, 0n);
  const base = await w4heap();
  let input: unknown[] | undefined = w4rows(n);
  const withInput = await w4heap();
  const t0 = performance.now();
  const out = await test_aggregate(input, 0n, 0n);
  const ms = performance.now() - t0;
  const rows = $canValue(out) as { key: bigint; value: bigint }[];
  input = undefined;
  w4holder = rows;
  const withResult = await w4heap();
  let orderOk = rows.length === n;
  for (let k = 0; orderOk && k < rows.length; k++) {
    if (rows[k].key !== BigInt(k + 1) || rows[k].value !== BigInt((k + 1) * 10)) orderOk = false;
  }
  w4leg({leg: "aggregate", n: n, ok: orderOk, ms: ms, inputBytes: withInput - base, resultBytes: withResult - base});
  w4holder = undefined;
}
async function w4growLeg(n: number): Promise<void> {
  w4holder = undefined;
  await test_grow([], 10n);
  await w4heap();
  const t0 = performance.now();
  const out = await test_grow([], BigInt(n));
  const ms = performance.now() - t0;
  const grown = $canValue(out) as bigint[];
  w4holder = grown;
  await w4heap();
  let ok = grown.length === n;
  if (ok && n > 0) ok = grown[0] === BigInt(n) && grown[n - 1] === 1n;
  w4leg({leg: "grow", n: n, ok: ok, ms: ms});
  w4holder = undefined;
}
async function w4faultStandardLeg(): Promise<void> {
  const boxed = await test_fault_standard(60000n, true);
  const completion = boxed as { kind: string; value: unknown };
  let kind = completion.kind;
  let message = "";
  let step = "";
  let occurrence = "0";
  if (completion.kind === "standard") {
    const details = standardFailureDiagnostics(completion.value as never);
    kind = details.kind;
    message = details.message;
    occurrence = String(details.occurrenceID);
    step = w4step(details.origin.invocation);
  }
  const contained = ($canValue(await test_machine(100n, 0n, 0n)) as bigint) === w4machine(100);
  w4leg({leg: "fault_standard", n: 60000, contained: contained,
    ok: kind === "arithmetic" && message === "arithmetic: integer division by zero" && step === "step:60000" && occurrence !== "0" && contained,
    kind: kind, message: message, step: step, occurrence: occurrence});
}
async function w4faultDeclaredLeg(): Promise<void> {
  const boxed = await test_fault_declared(60000n);
  const completion = boxed as { kind: string; value: unknown };
  let failureType = "";
  let decl = "";
  let origin = "";
  let occurrence = "0";
  if (completion.kind === "domain") {
    failureType = $canErrorType(boxed as never);
    const details = domainFailureDiagnostics(completion.value as never);
    occurrence = String(details.occurrenceID);
    origin = details.origin.source;
    decl = details.declaration.identity;
  }
  const contained = ($canValue(await test_scan_sum([1n, 2n], 0n, 0n)) as bigint) === 3n;
  w4leg({leg: "fault_declared", n: 60000, contained: contained,
    ok: completion.kind === "domain" && failureType === W4ABSENT && decl === W4ABSENT_DECL && origin === "can:collections:map" && occurrence !== "0" && contained,
    kind: completion.kind, occurrence: occurrence, failureType: failureType, decl: decl, origin: origin});
}
async function w4faultOriginatedLeg(): Promise<void> {
  const boxed = await test_fault_originated(60000n);
  const completion = boxed as { kind: string; value: unknown };
  let failureType = "";
  let decl = "";
  let step = "";
  let occurrence = "0";
  if (completion.kind === "domain") {
    failureType = $canErrorType(boxed as never);
    const details = domainFailureDiagnostics(completion.value as never);
    occurrence = String(details.occurrenceID);
    step = w4step(details.origin.invocation);
    decl = details.declaration.identity;
  }
  const contained = ($canValue(await test_machine(100n, 0n, 0n)) as bigint) === w4machine(100);
  w4leg({leg: "fault_originated", n: 60000, contained: contained,
    ok: completion.kind === "domain" && failureType === W4ABSENT && decl === W4ABSENT_DECL && step === "step:60000" && occurrence !== "0" && contained,
    kind: completion.kind, step: step, occurrence: occurrence, failureType: failureType, decl: decl});
}
async function w4bulkFailureLeg(): Promise<void> {
  const dup: unknown[] = [
    $canRecord(W4ENTRY, [["key", 1n], ["value", 1n]]),
    $canRecord(W4ENTRY, [["key", 1n], ["value", 2n]]),
  ];
  const boxed = await test_aggregate(dup, 2n, 2n);
  const completion = boxed as { kind: string; value: unknown };
  let failureType = "";
  let decl = "";
  if (completion.kind === "domain") {
    failureType = $canErrorType(boxed as never);
    decl = domainFailureDiagnostics(completion.value as never).declaration.identity;
  }
  w4leg({leg: "bulk_failure", n: 2, ok: completion.kind === "domain" && failureType === W4EXISTS && decl === W4EXISTS_DECL,
    kind: completion.kind, failureType: failureType, decl: decl});
}
async function w4emptyLegs(): Promise<void> {
  const rows = $canValue(await test_aggregate([], 0n, 0n)) as unknown[];
  const total = $canValue(await test_scan_sum([], 0n, 0n)) as bigint;
  const acc = $canValue(await test_machine(0n, 0n, 5n)) as bigint;
  w4leg({leg: "empty", n: 0, ok: rows.length === 0 && total === 0n && acc === 5n});
}
async function w4controlLegs(): Promise<void> {
  const triangle = $canValue(await test_triangle(100n)) as bigint;
  const ping = $canValue(await test_ping(20n)) as bigint;
  const looped = $canValue(await test_loop_excluded(50n)) as bigint;
  w4leg({leg: "controls", n: 0, ok: triangle === 5050n && ping === 100n && looped === 0n});
}
const w4args = process.argv.slice(2);
const w4kind = w4args[0] === undefined ? "misc" : w4args[0];
const w4size = w4args[1] === undefined ? 0 : Number(w4args[1]);
if (w4kind === "machine") await w4machineLeg(w4size);
else if (w4kind === "scan") await w4scanLeg(w4size);
else if (w4kind === "aggregate") await w4aggregateLeg(w4size);
else if (w4kind === "grow") await w4growLeg(w4size);
else {
  await w4faultStandardLeg();
  await w4faultDeclaredLeg();
  await w4faultOriginatedLeg();
  await w4bulkFailureLeg();
  await w4emptyLegs();
  await w4controlLegs();
}
console.log("W4DONE");
`
