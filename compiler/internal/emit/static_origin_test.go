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
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

func staticOriginDouble(t *testing.T) (*ir.Region, string) {
	t.Helper()
	program := actionEmitProgram(t, map[string]string{"src/main.can": "package app\n    provides []\n    uses []\nfn int double\n    emits {}\n    given\n        int value\n    asserts\n        sample: 2 => ok 4\n    ok value * 2\nfn void main\n    emits {}\n    given\n        str[] arguments\n    asserts\n        empty: [] => ok\n    ok\n"})
	for _, fn := range program.Functions {
		if fn.Symbol.Name == "double" {
			return fn.Region, fn.Symbol.Source.ID
		}
	}
	t.Fatal("missing double region")
	return nil, ""
}

func TestStaticOriginLazySlotsAndFrozenShape(t *testing.T) {
	region, sourceID := staticOriginDouble(t)
	emitter := RegionEmitter{SourceID: sourceID}
	code, err := emitter.Function("testDouble", region)
	if err != nil {
		t.Fatal(err)
	}
	clean, mappings, err := extractMappings(code)
	if err != nil {
		t.Fatal(err)
	}
	if len(mappings) == 0 {
		t.Fatal("mapped function lost coverage")
	}
	if !strings.HasPrefix(clean, "async function testDouble(") {
		t.Fatalf("function token moved:\n%s", clean)
	}
	if !strings.Contains(clean, "var $canOrigin_testDouble_") || !strings.Contains(clean, "Readonly<{source: string; start: number; end: number; invocation: readonly string[]}> | undefined") {
		t.Fatalf("typed private slots missing:\n%s", clean)
	}
	if !strings.Contains(clean, "??=") || !strings.Contains(clean, "Object.freeze({source:") || !strings.Contains(clean, "invocation:Object.freeze([") {
		t.Fatalf("lazy frozen init missing:\n%s", clean)
	}
	if strings.Contains(clean, "let $canOrigin_testDouble_") {
		t.Fatalf("slot must be hoisted var, not let:\n%s", clean)
	}
	found := false
	for _, line := range strings.Split(clean, "\n") {
		if strings.HasPrefix(line, "let $canOrigin = (") && strings.Contains(line, "??=") {
			found = true
		}
	}
	if !found {
		t.Fatalf("entry lazy init not single-line:\n%s", clean)
	}
	// Two mapped functions in one module get noncolliding slots.
	other, err := emitter.Function("testOther", region)
	if err != nil {
		t.Fatal(err)
	}
	otherClean, _, err := extractMappings(other)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(otherClean, "var $canOrigin_testOther_") || strings.Contains(otherClean, "var $canOrigin_testDouble_") {
		t.Fatalf("second function slots collide:\n%s", otherClean)
	}
}

func TestStaticOriginExclusionsKeepLiterals(t *testing.T) {
	region, _ := staticOriginDouble(t)
	unmapped := RegionEmitter{}
	plain, err := unmapped.Function("testPlain", region)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(plain, "??=") || strings.Contains(plain, "var $canOrigin_") {
		t.Fatalf("unmapped function cached:\n%s", plain)
	}
	if !strings.Contains(plain, "{source:") {
		t.Fatalf("unmapped literal lost:\n%s", plain)
	}
	tail := selfTailRegion(t, selfTailCountdownSource, "countdown")
	emitter := RegionEmitter{SourceID: "tail.can"}
	lowered, err := emitter.Function("testTail", tail)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(lowered, "??=") || strings.Contains(lowered, "var $canOrigin_") {
		t.Fatalf("self-tail function cached:\n%s", lowered)
	}
	if !strings.Contains(lowered, `"step:"+`) {
		t.Fatalf("self-tail step lost:\n%s", lowered)
	}
}

func TestStaticOriginFailedEmissionResets(t *testing.T) {
	f := newRegionFixture(t)
	region, err := f.region(t, "    ok call first() + 1\n", "int", nil, ir.FunctionRegion)
	if err != nil {
		t.Fatal(err)
	}
	emitter := RegionEmitter{SourceID: "reset.can"}
	if _, err := emitter.Function("testFail", region); err == nil {
		t.Fatal("missing target admitted")
	}
	if emitter.originCacheActive || emitter.originCache != nil || emitter.originSlots != nil || emitter.originCachePrefix != "" {
		t.Fatal("failed emission leaked cache state")
	}
	good, err := f.region(t, "    ok 1\n", "int", nil, ir.FunctionRegion)
	if err != nil {
		t.Fatal(err)
	}
	code, err := emitter.Function("testGood", good)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(code, "var $canOrigin_testGood_0:") || strings.Contains(code, "var $canOrigin_testFail_") {
		t.Fatalf("reuse did not restart slots:\n%s", code)
	}
	if _, err := emitter.configure(good); err != nil {
		t.Fatal(err)
	}
	literal := emitter.origin(good.Span)
	if strings.Contains(literal, "??=") || !strings.Contains(literal, "{source:") {
		t.Fatalf("configure-only origin cached: %s", literal)
	}
	if emitter.originCacheActive || emitter.originCache != nil || emitter.originSlots != nil {
		t.Fatal("configure-only path activated cache")
	}
}

func TestStaticOriginSubstitutedSourceInterning(t *testing.T) {
	emitter := RegionEmitter{SourceID: "use.can"}
	emitter.originCacheActive = true
	emitter.originCachePrefix = "$canOrigin_testSub_"
	emitter.originCache = map[originCacheKey]int{}
	emitter.region = &ir.Region{ID: "use::fn", Source: "use.can"}
	node := &ir.Expression{Source: "def.can", Span: source.Span{Start: 3, End: 9}}
	mark := emitter.markNode(node, "literal")
	if !strings.Contains(mark, `"def.can"`) || !strings.Contains(mark, "??=") {
		t.Fatalf("substituted source not interned: %q", mark)
	}
	clean, mappings, err := extractMappings(mark)
	if err != nil {
		t.Fatal(err)
	}
	if len(mappings) != 2 || mappings[0].Source != "def.can" || mappings[1].Source != "def.can" {
		t.Fatalf("definition mapping lost: %+v", mappings)
	}
	if !strings.Contains(clean, "$canOrigin = ($canOrigin_testSub_0 ??=") {
		t.Fatalf("assignment moved: %q", clean)
	}
	again := emitter.markNode(node, "literal")
	if !strings.Contains(again, "$canOrigin_testSub_0 ??=") {
		t.Fatalf("repeat did not reuse slot: %q", again)
	}
	emitter.region = &ir.Region{ID: "handler::1", Source: "use.can"}
	other := emitter.markNode(node, "literal")
	if !strings.Contains(other, "$canOrigin_testSub_1 ??=") {
		t.Fatalf("handler region not separated: %q", other)
	}
}

func TestStaticOriginTemplateSubstitutionEmission(t *testing.T) {
	// The fixture template lives in a separate definition file from its
	// consumer, so definition spans are distinguishable from use spans
	// through real checking and production emission (no manual markNode).
	helpers := "package app\n    provides []\n    uses []\nfn int double\n    emits {}\n    given\n        int value\n    asserts\n        sample: 2 => ok 4\n    ok value + value\nfixture doubled for double\n    given\n        int base\n    cases\n        base => ok base + base\n        3 => ok 6\n"
	mainText := "package app\n    provides []\n    uses []\nfn int consumer\n    emits {}\n    asserts\n        sample: => ok 4\n    match call double(2)\n        when\n            sample: use doubled(2)\n        ok int got => ok got\nfn void main\n    emits {}\n    given\n        str[] arguments\n    asserts\n        empty: [] => ok\n    ok\n"
	program := actionEmitProgram(t, map[string]string{"src/main.can": mainText, "src/helpers.can": helpers})
	var consumerRegion *ir.Region
	var consumerOut, useID, defID, consumerRegionID string
	foundDouble := false
	for _, fn := range program.Functions {
		switch fn.Symbol.Name {
		case "consumer":
			consumerRegion, consumerOut = fn.Region, fn.Symbol.Source.OutputPath
			useID, consumerRegionID = fn.Symbol.Source.ID, fn.Region.ID
		case "double":
			defID, foundDouble = fn.Symbol.Source.ID, true
		}
	}
	if consumerRegion == nil || !foundDouble {
		t.Fatal("consumer or definition missing from checked program")
	}
	if defID == "" || defID == useID {
		t.Fatalf("definition/use sources not distinct: %q %q", defID, useID)
	}
	call := consumerRegion.Body.Terminal.Match.Call
	if call == nil || len(call.Steps) != 1 || call.Steps[0].Fixtures == nil || len(call.Steps[0].Fixtures.Rows) != 2 {
		t.Fatal("consumer when table did not expand to two fixture rows")
	}
	// The expanded rows mix definition nodes (tagged with the definition
	// source) and substituted use-argument nodes (untagged, use file).
	seenDef, seenUse := false, false
	var walk func(*ir.Expression)
	walk = func(expr *ir.Expression) {
		if expr == nil {
			return
		}
		switch expr.Source {
		case "":
			seenUse = true
		case defID:
			seenDef = true
		default:
			t.Fatalf("fixture node attributed to %s", expr.Source)
		}
		for _, input := range expr.Inputs {
			walk(input)
		}
	}
	for _, row := range call.Steps[0].Fixtures.Rows {
		for _, prep := range row.Prepare {
			walk(prep.Value)
		}
		for _, arg := range row.Arguments {
			walk(arg)
		}
		if row.Expected != nil {
			walk(row.Expected.Value)
		}
	}
	if !seenDef || !seenUse {
		t.Fatalf("expansion missing definition (%t) or use (%t) nodes", seenDef, seenUse)
	}
	rows := call.Steps[0].Fixtures.Rows
	definition := rows[0].Expected.Value
	if definition.Kind != ir.Binary || definition.Source != defID {
		t.Fatalf("definition outcome lost its tag: %+v", definition)
	}
	if got := helpers[definition.Span.Start:definition.Span.End]; got != "base + base" {
		t.Fatalf("definition span slices %q, want the authored definition outcome", got)
	}
	useArgument := rows[0].Prepare[0].Value
	if useArgument.Source != "" || useArgument.Text != "2" {
		t.Fatalf("substituted use argument mistagged: %+v", useArgument)
	}
	if got := mainText[useArgument.Span.Start:useArgument.Span.End]; got != "2" {
		t.Fatalf("use span slices %q, want the authored use argument", got)
	}
	artifacts, err := ProgramModules(program, "runtime", httpDependencies(t))
	if err != nil {
		t.Fatal(err)
	}
	var consumerArtifact *ir.Artifact
	for i, artifact := range artifacts {
		if !artifact.Runtime && artifact.Path == consumerOut {
			consumerArtifact = &artifacts[i]
		}
	}
	if consumerArtifact == nil {
		t.Fatal("consumer module missing from production emission")
	}
	// Definition spans emit with the definition source under the active
	// (consumer) region; use spans keep the use source. Both keep paired
	// mapping tokens at their original sites.
	defCount, useCount, functionCount := 0, 0, 0
	paired := map[ir.Mapping]int{}
	sawDefinitionOutcome := false
	for _, mapping := range consumerArtifact.Mappings {
		// Function-entry marks carry a single leading token by design;
		// every other mark pairs tokens around its assignment.
		if mapping.Operation == "function" {
			functionCount++
		} else {
			key := ir.Mapping{Source: mapping.Source, Start: mapping.Start, End: mapping.End, Operation: mapping.Operation}
			paired[key]++
		}
		switch mapping.Source {
		case defID:
			defCount++
			if mapping.Start < 0 || mapping.End > len(helpers) || mapping.Start >= mapping.End {
				t.Fatalf("definition mapping outside its file: %+v", mapping)
			}
			if helpers[mapping.Start:mapping.End] == "base + base" {
				sawDefinitionOutcome = true
			}
		case useID:
			useCount++
			if mapping.Start < 0 || mapping.End > len(mainText) || mapping.Start >= mapping.End {
				t.Fatalf("use mapping outside its file: %+v", mapping)
			}
		default:
			t.Fatalf("consumer mapping attributed to %s", mapping.Source)
		}
	}
	if defCount == 0 || useCount == 0 {
		t.Fatalf("consumer emission missing definition (%d) or use (%d) mappings", defCount, useCount)
	}
	if !sawDefinitionOutcome {
		t.Fatal("definition outcome span missing from consumer mappings")
	}
	if functionCount != 2 {
		t.Fatalf("consumer module has %d function entries, want consumer plus main", functionCount)
	}
	for key, count := range paired {
		if count%2 != 0 {
			t.Fatalf("unpaired mapping token %+v appears %d times", key, count)
		}
	}
	want := fmt.Sprintf("Object.freeze({source:%s,start:%d,end:%d,invocation:Object.freeze([%s])})",
		quote(defID), definition.Span.Start, definition.Span.End, quote(consumerRegionID))
	body := string(consumerArtifact.Bytes)
	index := strings.Index(body, want)
	if index < 0 {
		t.Fatalf("definition origin missing from lazy slots:\n%s", body)
	}
	lineStart := strings.LastIndex(body[:index], "\n") + 1
	lineEnd := strings.Index(body[index:], "\n") + index
	if !strings.Contains(body[lineStart:lineEnd], "??=") {
		t.Fatalf("definition origin not lazily interned: %s", body[lineStart:lineEnd])
	}
}

func TestStaticOriginBunSharingAndFreshness(t *testing.T) {
	bun := os.Getenv("CAN_BUN")
	if bun == "" {
		t.Skip("set CAN_BUN to qualified runtime")
	}
	f := newRegionFixture(t)
	okBody, stdBody := "    ok 1\n", "    ok 1 / 0\n"
	domBody := "    match call lookup(1)\n        missing => ok 0\n        ok => missing{9}\n"
	okRegion, err := f.region(t, okBody, "int", nil, ir.FunctionRegion)
	if err != nil {
		t.Fatal(err)
	}
	stdRegion, err := f.region(t, stdBody, "int", nil, ir.FunctionRegion)
	if err != nil {
		t.Fatal(err)
	}
	domRegion, err := f.region(t, domBody, "int", []string{"missing"}, ir.FunctionRegion)
	if err != nil {
		t.Fatal(err)
	}
	// Authored-text oracles: every asserted span must slice the exact
	// fixture source the checker parsed (see regions_test.go), so emitted
	// tuples are proven against authored text rather than pairwise
	// agreement between two repeated origins.
	authored := func(result, body string) string {
		return "package app\n    provides []\n    uses []\nfn " + result + " run\n    emits {}\n    asserts\n        test: => ok 1\n" + body
	}
	okText, stdText, domText := authored("int", okBody), authored("int", stdBody), authored("int", domBody)
	slice := func(text string, span source.Span) string { return text[span.Start:span.End] }
	if okRegion.ID != "app::run" || stdRegion.ID != "app::run" || domRegion.ID != "app::run" {
		t.Fatalf("fixture regions moved: %q %q %q", okRegion.ID, stdRegion.ID, domRegion.ID)
	}
	if got := slice(okText, okRegion.Span); got != "ok 1\n" {
		t.Fatalf("ok entry span slices %q, want the authored terminal", got)
	}
	// Slots intern in emission order: body marks first, the function entry
	// last. The ok function therefore owns exactly the literal slot and
	// the entry slot, in that order.
	okValue := okRegion.Body.Terminal.Value
	if okValue == nil || okValue.Kind != ir.Literal {
		t.Fatalf("ok body is not the literal: %+v", okValue)
	}
	if got := slice(okText, okValue.Span); got != "1" {
		t.Fatalf("ok literal span slices %q, want the authored literal", got)
	}
	stdValue := stdRegion.Body.Terminal.Value
	if stdValue == nil || stdValue.Kind != ir.Binary {
		t.Fatalf("std failure site is not the division: %+v", stdValue)
	}
	if got := slice(stdText, stdValue.Span); got != "1 / 0" {
		t.Fatalf("std failure span slices %q, want the authored division", got)
	}
	var domSpan source.Span
	foundDomain := false
	for _, arm := range domRegion.Body.Terminal.Match.Arms {
		if arm.Body.Kind == ir.DomainCompletion {
			domSpan, foundDomain = arm.Body.Span, true
		}
	}
	if !foundDomain {
		t.Fatal("domain failure site missing from match arms")
	}
	if got := slice(domText, domSpan); got != "missing{9}" {
		t.Fatalf("domain failure span slices %q, want the authored constructor", got)
	}
	tuple := func(regionID string, span source.Span) string {
		return fmt.Sprintf("{source:\"share.can\",start:%d,end:%d,invocation:[%s]}", span.Start, span.End, quote(regionID))
	}
	okTuple, stdTuple, domTuple := tuple(okRegion.ID, okRegion.Span), tuple(stdRegion.ID, stdValue.Span), tuple(domRegion.ID, domSpan)
	okLitTuple := tuple(okRegion.ID, okValue.Span)
	branchRegion, err := f.region(t, "    match flag\n        false => ok 0\n        true => ok 1\n", "int", nil, ir.FunctionRegion)
	if err != nil {
		t.Fatal(err)
	}
	emitter := RegionEmitter{
		Bindings:      map[string]string{"value/flag": "flag"},
		Functions:     map[string]string{"function/lookup": "$lookup"},
		DomainRuntime: "$canDomain",
		SourceID:      "share.can",
	}
	okCode, err := emitter.Function("testShareOk", okRegion)
	if err != nil {
		t.Fatal(err)
	}
	stdCode, err := emitter.Function("testShareStd", stdRegion)
	if err != nil {
		t.Fatal(err)
	}
	domCode, err := emitter.Function("testShareDom", domRegion)
	if err != nil {
		t.Fatal(err)
	}
	branchCode, err := emitter.Function("testShareBranch", branchRegion)
	if err != nil {
		t.Fatal(err)
	}
	for name, code := range map[string]string{"ok": okCode, "std": stdCode, "dom": domCode, "branch": branchCode} {
		clean, mappings, err := extractMappings(code)
		if err != nil {
			t.Fatal(name, err)
		}
		if len(mappings) == 0 {
			t.Fatalf("%s lost mapping coverage", name)
		}
		if !strings.Contains(clean, "??=") || !strings.Contains(clean, "var $canOrigin_testShare") {
			t.Fatalf("%s not cached:\n%s", name, clean)
		}
	}
	okClean, _, _ := extractMappings(okCode)
	stdClean, _, _ := extractMappings(stdCode)
	domClean, _, _ := extractMappings(domCode)
	branchClean, _, _ := extractMappings(branchCode)
	runtimeRoot, _ := filepath.Abs("../../../runtime")
	path := func(name string) string { p, _ := filepath.Abs(filepath.Join(runtimeRoot, name+".ts")); return p }
	var program strings.Builder
	program.WriteString(CompletionImports(path("completion")))
	program.WriteString(FailureImports(path("failure")))
	program.WriteString(DataImports(path("data")))
	program.WriteString(PrimitiveImports(path("primitive")))
	fmt.Fprintf(&program, "import {createDomainRuntime} from %s;\n", quote(path("domain")))
	fmt.Fprintf(&program, "import {standardFailureDiagnostics} from %s;\n", quote(path("failure")))
	fmt.Fprintf(&program, "import {captureStandard} from %s;\n", quote(path("failure")))
	fmt.Fprintf(&program, "import {domainFailureDiagnostics} from %s;\n", quote(path("domain")))
	fmt.Fprintf(&program, "const $canExpectedOk = %s;\nconst $canExpectedOkLit = %s;\nconst $canExpectedStd = %s;\nconst $canExpectedDom = %s;\n", okTuple, okLitTuple, stdTuple, domTuple)
	bound, err := f.registry.Bound([]*types.Type{f.ts["missing"]})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := json.Marshal(f.registry.Plan(bound))
	if err != nil {
		t.Fatal(err)
	}
	fmt.Fprintf(&program, "const $canDomain=createDomainRuntime(%s);\n", plan)
	declarations, err := RegionTypeDeclarations(okRegion, stdRegion, domRegion, branchRegion)
	if err != nil {
		t.Fatal(err)
	}
	program.WriteString(declarations)
	program.WriteString("const flag=true;\n")
	program.WriteString("async function $lookup(n:bigint){return $canSuccess(n);}\n")
	program.WriteString(okClean + "\n" + stdClean + "\n" + domClean + "\n" + branchClean + "\n")
	okCount := len(strings.Split(okClean, "var $canOrigin_testShareOk_")) - 1
	program.WriteString(`
import {strict as assert} from "node:assert";
const okVars = [`)
	first := true
	for i := 0; i < okCount; i++ {
		if !first {
			program.WriteString(",")
		}
		first = false
		fmt.Fprintf(&program, `() => $canOrigin_testShareOk_%d`, i)
	}
	program.WriteString(`];
const okValues = () => okVars.map((read) => read());
assert.ok(okVars.length > 0, "ok slots missing");
assert.ok(okValues().every((s) => s === undefined), "slots eagerly initialized");
const first = await testShareOk();
assert.equal($canValue(first), 1n);
assert.ok(okValues().every((s) => s !== undefined), "reachable slots stayed empty");
const seen = okValues()[0];
assert.ok(Object.isFrozen(seen) && Object.isFrozen(seen.invocation), "slot not deeply frozen");
assert.deepEqual(okValues(), [$canExpectedOkLit, $canExpectedOk]);
const entry = okValues()[1];
const second = await testShareOk();
assert.equal($canValue(second), 1n);
assert.ok(okValues()[0] === seen, "repeat call reallocated origin");
const pair = await Promise.all([testShareOk(), testShareOk()]);
assert.equal($canValue(pair[0]), 1n);
assert.equal($canValue(pair[1]), 1n);
assert.ok(okValues()[0] === seen, "concurrent calls reallocated origin");
// Standard faults keep fresh occurrences with exact spans.
const stdA = await testShareStd();
const stdB = await testShareStd();
assert.equal(stdA.kind, "standard");
assert.equal(stdB.kind, "standard");
const diagA = standardFailureDiagnostics(stdA.value);
const diagB = standardFailureDiagnostics(stdB.value);
assert.notEqual(diagA.occurrenceID, diagB.occurrenceID);
assert.deepEqual(diagA.origin, $canExpectedStd);
assert.deepEqual(diagB.origin, $canExpectedStd);
assert.ok(Object.isFrozen(diagA.origin) && Object.isFrozen(diagA.origin.invocation));
// Domain faults keep fresh occurrences with exact spans.
const domA = await testShareDom();
const domB = await testShareDom();
assert.equal(domA.kind, "domain");
assert.equal(domB.kind, "domain");
const ddiagA = domainFailureDiagnostics(domA.value);
const ddiagB = domainFailureDiagnostics(domB.value);
assert.notEqual(ddiagA.occurrenceID, ddiagB.occurrenceID);
assert.deepEqual(ddiagA.origin, $canExpectedDom);
assert.deepEqual(ddiagB.origin, $canExpectedDom);
// A synthetic can: failure keeps its original occurrence and location while
// the first mapped authored invoke boundary records the exact call-site tuple.
const synthOrigin = {source:"can:static-origin-synth", start:7, end:13, invocation:["synthetic::site"]};
const synth = captureStandard("probe", synthOrigin);
const synthCarrier = $canFailure(synth);
const synthID = standardFailureDiagnostics(synth).occurrenceID;
const synthOut = await $canInvoke(() => synthCarrier, entry);
assert.ok(synthOut.value === synth, "synthetic occurrence replaced");
const synthDiag = standardFailureDiagnostics(synthOut.value);
assert.equal(synthDiag.occurrenceID, synthID);
assert.deepEqual(synthDiag.origin, synthOrigin);
assert.deepEqual(synthDiag.boundaryOrigin, $canExpectedOk);
await $canInvoke(() => synthCarrier, {source:"share.can", start:0, end:1, invocation:["app::run"]});
assert.deepEqual(standardFailureDiagnostics(synth).boundaryOrigin, $canExpectedOk, "first boundary overwritten");
// One branch taken: entry plus taken arm initialize, the other arm stays lazy.
const branchVars = [`)
	branchCount := len(strings.Split(branchClean, "var $canOrigin_testShareBranch_")) - 1
	first = true
	for i := 0; i < branchCount; i++ {
		if !first {
			program.WriteString(",")
		}
		first = false
		fmt.Fprintf(&program, `() => $canOrigin_testShareBranch_%d`, i)
	}
	program.WriteString(`];
const branchValues = () => branchVars.map((read) => read());
assert.ok(branchVars.length >= 3, "branch slots missing");
assert.ok(branchValues().every((s) => s === undefined), "branch eagerly initialized");
assert.equal($canValue(await testShareBranch()), 1n);
const defined = branchValues().filter((s) => s !== undefined).length;
assert.ok(defined > 0 && defined < branchVars.length, "unreached branch not lazy");
console.log("static origins shared");
`)
	file := filepath.Join(t.TempDir(), "static-origin.ts")
	if err := os.WriteFile(file, []byte(program.String()), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, bun, "--no-install", "--no-env-file", file).CombinedOutput()
	if err != nil || !strings.Contains(string(out), "static origins shared") {
		t.Fatalf("%v\n%s\n%s", err, out, program.String())
	}
}

func TestStaticOriginProductionMappingsAndLiterals(t *testing.T) {
	program := actionEmitProgram(t, map[string]string{"src/main.can": "package app\n    provides []\n    uses []\nfn str echo\n    emits {}\n    given\n        str value\n    asserts\n        sample: \"$canOrigin\" => ok \"$canOrigin\"\n    ok value\nfn str tricky\n    emits {}\n    asserts\n        sample: => ok \"{source:fake\" \n    ok \"{source:fake\"\nfn void main\n    emits {}\n    given\n        str[] arguments\n    asserts\n        empty: [] => ok\n    ok\n"})
	for _, emit := range []struct {
		name string
		call func() ([]ir.Artifact, error)
	}{
		{"production", func() ([]ir.Artifact, error) { return ProgramModules(program, "runtime", httpDependencies(t)) }},
		{"assertion", func() ([]ir.Artifact, error) { return AssertionModules(program, "runtime", httpDependencies(t)) }},
	} {
		artifacts, err := emit.call()
		if err != nil {
			t.Fatal(emit.name, err)
		}
		seenFunction, seenSlots := false, false
		for _, artifact := range artifacts {
			if artifact.Runtime || !strings.HasSuffix(artifact.Path, ".ts") {
				continue
			}
			body := string(artifact.Bytes)
			if strings.Contains(body, "var $canOrigin_") {
				seenSlots = true
			}
			line, column := 0, -1
			for _, segment := range artifact.Mappings {
				if segment.Operation == "function" {
					seenFunction = true
				}
				if segment.Line < 1 || segment.Column < 0 || segment.Line < line || (segment.Line == line && segment.Column <= column) {
					t.Fatalf("%s %s stacks marks at %d:%d", emit.name, artifact.Path, segment.Line, segment.Column)
				}
				line, column = segment.Line, segment.Column
			}
		}
		if !seenFunction || !seenSlots {
			t.Fatalf("%s lost function mappings or lazy slots", emit.name)
		}
		joined := ""
		for _, artifact := range artifacts {
			joined += string(artifact.Bytes) + "\n"
		}
		// User string data survives origin interning byte-for-byte.
		// tricky's body literal ships in both modes; echo's assertion
		// literal ships only with assertion roots.
		if !strings.Contains(joined, `"{source:fake"`) {
			t.Fatalf("%s rewrote body literal", emit.name)
		}
		if emit.name == "assertion" && !strings.Contains(joined, `"$canOrigin"`) {
			t.Fatalf("assertion rewrote user literal")
		}
		// Assertion roots keep distinct per-function slots.
		if emit.name == "assertion" && (!strings.Contains(joined, "var $canOrigin_$canActual_") || !strings.Contains(joined, "var $canOrigin_$canExpected_")) {
			t.Fatalf("assertion slots not per-function:\n%s", joined)
		}
	}
}

func TestStaticOriginCoordinationHandlerSeparation(t *testing.T) {
	f := newRegionFixture(t)
	region, err := f.region(t, `    int[] results = match call concurrent
        build().bump(call first()).read()
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
	emitter := RegionEmitter{Functions: functions, DomainRuntime: "$canDomain", SourceID: "coord.can"}
	code, err := emitter.Function("testCoord", region)
	if err != nil {
		t.Fatal(err)
	}
	clean, mappings, err := extractMappings(code)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(clean, "var $canOrigin_testCoord_") || !strings.Contains(clean, "??=") {
		t.Fatalf("coordination not cached:\n%s", clean)
	}
	// Handler regions keep their own invocation identity inside one cache.
	if !strings.Contains(clean, "coordination-handler") {
		t.Fatalf("handler region IDs missing:\n%s", clean)
	}
	seenCall, seenCoord := false, false
	for _, segment := range mappings {
		if segment.Operation == "call" {
			seenCall = true
		}
		if segment.Operation == "coordination" {
			seenCoord = true
		}
	}
	if !seenCall || !seenCoord {
		t.Fatalf("coordination mappings lost: %+v", mappings)
	}
}

func TestStaticOriginBrowserExportAndOwnerABI(t *testing.T) {
	program := browserEmitProgram(t, map[string]string{"src/main.can": browserPureSource})
	artifacts, err := BrowserModules(program, "runtime", httpDependencies(t))
	if err != nil {
		t.Fatal(err)
	}
	seenExport, seenSlots, seenCtx := false, false, false
	for _, artifact := range artifacts {
		if !strings.HasSuffix(artifact.Path, ".ts") {
			continue
		}
		body := string(artifact.Bytes)
		if strings.Contains(body, "export async function $canFunction") {
			seenExport = true
		}
		if strings.Contains(body, "var $canOrigin_$canFunction") {
			seenSlots = true
		}
		if strings.Contains(body, "$canCtx: $canOwnerContext") {
			seenCtx = true
		}
		for _, token := range []string{"process.", "Bun.", "require(", "node:"} {
			if strings.Contains(body, token) && strings.Contains(artifact.Path, "browser") {
				t.Fatalf("%s leaks host token %q", artifact.Path, token)
			}
		}
		line, column := 0, -1
		for _, segment := range artifact.Mappings {
			if segment.Line < line || (segment.Line == line && segment.Column <= column) {
				t.Fatalf("%s stacks marks at %d:%d", artifact.Path, segment.Line, segment.Column)
			}
			line, column = segment.Line, segment.Column
		}
	}
	if !seenExport || !seenSlots || !seenCtx {
		t.Fatal("browser export, slots or owner ABI missing")
	}
}

func TestStaticOriginEmittedInvocationBoundary(t *testing.T) {
	bun := os.Getenv("CAN_BUN")
	if bun == "" {
		t.Skip("set CAN_BUN to qualified runtime")
	}
	f := newRegionFixture(t)
	body := "    match call lookup(1)\n        missing => ok 0\n        ok int got => ok got\n"
	region, err := f.region(t, body, "int", nil, ir.FunctionRegion)
	if err != nil {
		t.Fatal(err)
	}
	// Authored-text oracle: the checked invocation step addresses the exact
	// call-site span, distinct from the function entry and the prepared
	// literal. The emitted caller must supply that step tuple as the first
	// boundary for a synthetic can: failure.
	text := "package app\n    provides []\n    uses []\nfn int run\n    emits {}\n    asserts\n        test: => ok 1\n" + body
	slice := func(span source.Span) string { return text[span.Start:span.End] }
	if region.ID != "app::run" {
		t.Fatalf("fixture region moved: %q", region.ID)
	}
	if region.Body.Terminal.Match == nil || region.Body.Terminal.Match.Call == nil {
		t.Fatal("match call missing from region terminal")
	}
	call := region.Body.Terminal.Match.Call
	if len(call.Steps) != 1 {
		t.Fatalf("match call has %d steps, want 1", len(call.Steps))
	}
	step := call.Steps[0]
	if step.Identity != "function/lookup" {
		t.Fatalf("invocation identity %q, want function/lookup", step.Identity)
	}
	if got := slice(step.Span); got != "lookup(1)" {
		t.Fatalf("invocation step slices %q, want the authored call site", got)
	}
	if entryText := slice(region.Span); !strings.Contains(entryText, "match call lookup(1)") {
		t.Fatalf("entry span slices %q, want the authored match", entryText)
	}
	if step.Span == region.Span {
		t.Fatal("invocation step shares the function-entry span")
	}
	var litSpan source.Span
	foundLit := false
	for _, prep := range step.Prepare {
		if prep.Value != nil && prep.Value.Kind == ir.Literal && prep.Value.Text == "1" {
			litSpan, foundLit = prep.Value.Span, true
		}
	}
	if !foundLit {
		t.Fatal("prepared literal 1 missing from invocation step")
	}
	if got := slice(litSpan); got != "1" {
		t.Fatalf("literal span slices %q, want the authored argument", got)
	}
	if litSpan == step.Span {
		t.Fatal("literal shares the invocation step span")
	}
	tuple := func(span source.Span) string {
		return fmt.Sprintf("{source:\"invoke.can\",start:%d,end:%d,invocation:[%s]}", span.Start, span.End, quote(region.ID))
	}
	invokeTuple, entryTuple := tuple(step.Span), tuple(region.Span)
	if invokeTuple == entryTuple {
		t.Fatal("invocation tuple equals the function-entry tuple")
	}
	emitter := RegionEmitter{
		Functions: map[string]string{"function/lookup": "$lookup"},
		SourceID:  "invoke.can",
	}
	code, err := emitter.Function("testInvokeBoundary", region)
	if err != nil {
		t.Fatal(err)
	}
	clean, mappings, err := extractMappings(code)
	if err != nil {
		t.Fatal(err)
	}
	if len(mappings) == 0 {
		t.Fatal("mapped caller lost coverage")
	}
	if !strings.Contains(clean, "??=") || !strings.Contains(clean, "var $canOrigin_testInvokeBoundary_") {
		t.Fatalf("caller not cached:\n%s", clean)
	}
	if !strings.Contains(clean, "$canInvoke") {
		t.Fatalf("caller emits no invocation step:\n%s", clean)
	}
	seenCall := 0
	for _, mapping := range mappings {
		if mapping.Operation == "call" && mapping.Source == "invoke.can" && mapping.Start == step.Span.Start && mapping.End == step.Span.End {
			seenCall++
		}
	}
	if seenCall != 2 {
		t.Fatalf("call-site mapping tokens %d, want paired marks: %+v", seenCall, mappings)
	}
	wantStep := fmt.Sprintf("Object.freeze({source:%s,start:%d,end:%d,invocation:Object.freeze([%s])})",
		quote("invoke.can"), step.Span.Start, step.Span.End, quote(region.ID))
	index := strings.Index(clean, wantStep)
	if index < 0 {
		t.Fatalf("invocation origin missing from lazy slots:\n%s", clean)
	}
	lineStart := strings.LastIndex(clean[:index], "\n") + 1
	lineEnd := strings.Index(clean[index:], "\n") + index
	if lineEnd < lineStart || !strings.Contains(clean[lineStart:lineEnd], "??=") {
		t.Fatalf("invocation origin not lazily interned: %s", clean[lineStart:lineEnd])
	}
	runtimeRoot, _ := filepath.Abs("../../../runtime")
	path := func(name string) string { p, _ := filepath.Abs(filepath.Join(runtimeRoot, name+".ts")); return p }
	var program strings.Builder
	program.WriteString(CompletionImports(path("completion")))
	fmt.Fprintf(&program, "import {captureStandard} from %s;\n", quote(path("failure")))
	fmt.Fprintf(&program, "import {standardFailureDiagnostics} from %s;\n", quote(path("failure")))
	fmt.Fprintf(&program, "const $canExpectedInvoke = %s;\nconst $canExpectedEntry = %s;\n", invokeTuple, entryTuple)
	declarations, err := RegionTypeDeclarations(region)
	if err != nil {
		t.Fatal(err)
	}
	program.WriteString(declarations)
	program.WriteString(clean + "\n")
	program.WriteString(`
import {strict as assert} from "node:assert";
assert.notDeepEqual($canExpectedInvoke, $canExpectedEntry, "invocation tuple equals entry");
// The injected target returns the synthetic carrier; the emitted caller,
// not the test, supplies the first checked boundary through $canInvoke.
const synthOrigin = {source:"can:static-origin-synth", start:7, end:13, invocation:["synthetic::site"]};
const synth = captureStandard("probe", synthOrigin);
const synthCarrier = $canFailure(synth);
const synthID = standardFailureDiagnostics(synth).occurrenceID;
async function $lookup(n:bigint){return synthCarrier;}
const out = await testInvokeBoundary();
assert.equal(out.kind, "standard");
assert.ok(out.value === synth, "synthetic occurrence replaced");
const diag = standardFailureDiagnostics(out.value);
assert.equal(diag.occurrenceID, synthID);
assert.deepEqual(diag.origin, synthOrigin);
assert.deepEqual(diag.boundaryOrigin, $canExpectedInvoke);
assert.notDeepEqual(diag.boundaryOrigin, $canExpectedEntry, "boundary fell back to entry");
await $canInvoke(() => synthCarrier, {source:"invoke.can", start:0, end:1, invocation:["app::run"]});
assert.deepEqual(standardFailureDiagnostics(synth).boundaryOrigin, $canExpectedInvoke, "first boundary overwritten");
console.log("static invocation boundary ok");
`)
	file := filepath.Join(t.TempDir(), "static-invoke-boundary.ts")
	if err := os.WriteFile(file, []byte(program.String()), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, bun, "--no-install", "--no-env-file", file).CombinedOutput()
	if err != nil || !strings.Contains(string(out), "static invocation boundary ok") {
		t.Fatalf("%v\n%s\n%s", err, out, program.String())
	}
}
