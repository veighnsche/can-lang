package emit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// closedRecoveryFixture compiles the real gallery recovery fixture plus the
// real gallery main; extraSource is appended to recovery.can for larger
// corpora while the gallery text stays byte-identical.
func closedRecoveryFixture(t *testing.T, extraSource string) (*check.Program, string) {
	t.Helper()
	root := filepath.Join("..", "..", "..", "tools", "performance", "fixtures", "runtime", "src")
	data, err := os.ReadFile(filepath.Join(root, "recovery.can"))
	if err != nil {
		t.Fatal(err)
	}
	mainCan, err := os.ReadFile(filepath.Join(root, "main.can"))
	if err != nil {
		t.Fatal(err)
	}
	return actionEmitProgram(t, map[string]string{
		"src/recovery.can": string(data) + extraSource,
		"src/main.can":     string(mainCan),
	}), string(data) + extraSource
}

func closedRecoveryAssembly(t *testing.T, program *check.Program) *programAssembly {
	t.Helper()
	assembly, err := assembleProgramBindings(program)
	if err != nil {
		t.Fatal(err)
	}
	return assembly
}

// closedRecoveryIdentities mirrors coreIdentities for a checks-only model:
// the real failed identity plus a domain plan covering it.
func closedRecoveryIdentities(t *testing.T, program *check.Program) (string, map[string]string) {
	t.Helper()
	ids := map[string]string{}
	for _, typ := range program.Model.Types() {
		ids[typ.Declaration()] = typ.Identity()
	}
	var failures []*types.Type
	for _, typ := range program.Model.Types() {
		if typ.Declaration() == "can.std.checks@1::failed" {
			failures = append(failures, typ)
		}
	}
	if len(failures) == 0 {
		t.Fatal("model lacks can.std.checks@1::failed")
	}
	bound, err := program.Registry.Bound(failures)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(program.Registry.Plan(bound))
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded), ids
}

// closedRecoveryEligible adds two independently named eligible recoveries
// with distinct integer thresholds and reasons.
const closedRecoveryEligible = `
fn bool is_big
    emits []
    given
        int number
    asserts
        yes: 11 => ok true
        no: 10 => ok false
    match call checks::require(number > 10, "ten required")
        checks::failed => ok false
        ok => ok true
fn bool is_huge
    emits []
    given
        int number
    asserts
        yes: 101 => ok true
        no: 100 => ok false
    match call checks::require(number > 100, "hundred required")
        checks::failed => ok false
        ok => ok true
`

// closedRecoveryNegatives declines proof twelve different ways: a wrong
// ordered operator, a non-literal threshold, a non-ordered predicate, an
// effectful comparison operand, reordered comparison operands, swapped arm
// literals, an extra leading statement, a second input, a wrong result
// type, a payload-dependent arm body, a value match with a standard arm
// (a standard arm is checker-unexpressible on a require call, so the
// proof's standard gate stays defensive), a folded threshold, and a
// string input with an ordered comparison (input-type gate).
const closedRecoveryNegatives = `
fn bool is_small
    emits []
    given
        int number
    asserts
        sample: 3 => ok false
    match call checks::require(number < 0, "negative required")
        checks::failed => ok false
        ok => ok true
fn bool is_shifted
    emits []
    given
        int number
    asserts
        sample: 3 => ok false
    match call checks::require(number > number, "self required")
        checks::failed => ok false
        ok => ok true
fn bool is_zero
    emits []
    given
        int number
    asserts
        sample: 3 => ok false
    match call checks::require(number is 0, "zero required")
        checks::failed => ok false
        ok => ok true
fn bool is_effectful
    emits []
    given
        int number
    asserts
        sample: 3 => ok true
    match call checks::require(number + 0 > 0, "positive required")
        checks::failed => ok false
        ok => ok true
fn bool is_reordered
    emits []
    given
        int number
    asserts
        sample: 3 => ok true
    match call checks::require(0 < number, "positive required")
        checks::failed => ok false
        ok => ok true
fn bool is_swapped
    emits []
    given
        int number
    asserts
        sample: 3 => ok false
    match call checks::require(number > 0, "positive required")
        checks::failed => ok true
        ok => ok false
fn bool is_extra
    emits []
    given
        int number
    asserts
        sample: 3 => ok true
    bool warm = number is 0
    match call checks::require(number > 0, "positive required")
        checks::failed => ok false
        ok => ok true
fn bool is_two
    emits []
    given
        int number
        int extra
    asserts
        sample: 3, 0 => ok true
    match call checks::require(number > 0, "positive required")
        checks::failed => ok false
        ok => ok true
fn int is_int
    emits []
    given
        int number
    asserts
        sample: 3 => ok 1
    match call checks::require(number > 0, "positive required")
        checks::failed => ok 0
        ok => ok 1
fn bool is_payload
    emits []
    given
        int number
    asserts
        sample: 0 => ok true
    match call checks::require(number > 0, "positive required")
        checks::failed as got => ok got.reason is "positive required"
        ok => ok true
variant failure
    checks::failed
    standard_failure
fn bool is_value_match
    emits []
    given
        failure value
    asserts
        sample: checks::failed("positive required") => ok false
    match value
        checks::failed => ok false
        standard_failure => ok true
fn bool is_folded
    emits []
    given
        int number
    asserts
        sample: 3 => ok true
    match call checks::require(number > 0 + 0, "positive required")
        checks::failed => ok false
        ok => ok true
fn bool is_name
    emits []
    given
        str name
    asserts
        sample: "zed" => ok true
    match call checks::require(name > "m", "name required")
        checks::failed => ok false
        ok => ok true
`

// closedRecoveryEligibleIdentities pins the exact eligible trio.
var closedRecoveryEligibleIdentities = map[string]struct{ threshold, reason string }{
	"can.project.root/gallery::is_positive": {"0", "positive required"},
	"can.project.root/gallery::is_big":      {"10", "ten required"},
	"can.project.root/gallery::is_huge":     {"100", "hundred required"},
}

func TestClosedRecoveryProofExact(t *testing.T) {
	program, _ := closedRecoveryFixture(t, closedRecoveryEligible)
	assembly := closedRecoveryAssembly(t, program)
	proof := assembly.checkedClosedRecoveries()
	if len(proof) != len(closedRecoveryEligibleIdentities) {
		t.Fatalf("proven recoveries = %d, want %d", len(proof), len(closedRecoveryEligibleIdentities))
	}
	gallery, _ := closedRecoveryModules(t, program)
	names := closedRecoveryEmittedNames(t, string(gallery.Bytes))
	for identity, want := range closedRecoveryEligibleIdentities {
		entry := proof[identity]
		if entry == nil {
			t.Fatalf("missing proof for %s", identity)
		}
		if entry.Target != names[identity] {
			t.Fatalf("%s target = %q", identity, entry.Target)
		}
		if entry.Threshold != want.threshold || entry.Reason != want.reason {
			t.Fatalf("%s threshold/reason = %q/%q", identity, entry.Threshold, entry.Reason)
		}
		if entry.Input == "" || entry.Region == nil || entry.Predicate == nil {
			t.Fatalf("%s proof incomplete: %+v", identity, entry)
		}
		if len(entry.Predicate.Inputs) != 2 {
			t.Fatalf("%s predicate inputs = %d", identity, len(entry.Predicate.Inputs))
		}
	}
}

func TestClosedRecoveryProofDeclines(t *testing.T) {
	program, _ := closedRecoveryFixture(t, closedRecoveryNegatives)
	assembly := closedRecoveryAssembly(t, program)
	proof := assembly.checkedClosedRecoveries()
	if len(proof) != 1 {
		t.Fatalf("proven recoveries = %d, want exactly the gallery entry: %v", len(proof), proof)
	}
	if _, ok := proof["can.project.root/gallery::is_positive"]; !ok {
		t.Fatalf("gallery proof missing: %v", proof)
	}
	for _, declined := range []string{"is_small", "is_shifted", "is_zero", "is_effectful", "is_reordered", "is_swapped", "is_extra", "is_two", "is_int", "is_payload", "is_value_match", "is_folded", "is_name"} {
		for identity := range proof {
			if strings.Contains(identity, declined) {
				t.Fatalf("sibling %s admitted a proof: %v", declined, proof)
			}
		}
	}
	// Any non-(failed,false)+(ok,true) arm set declines through the same
	// arm-set gate, including a wrong error identity: the failed-identity
	// check itself is pinned by the match-const assertion below.
	artifacts, err := AssertionModules(program, "runtime", httpDependencies(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, artifact := range artifacts {
		if !strings.HasPrefix(artifact.Path, "packages/") {
			continue
		}
		if got := strings.Count(string(artifact.Bytes), "$canAmbientOwnerPresent()"); got > 1 {
			t.Fatalf("%s admits %d recovery branches, want at most 1", artifact.Path, got)
		}
	}
}

// closedRecoveryModules returns the authored gallery module plus the main
// module for a recovery program.
func closedRecoveryModules(t *testing.T, program *check.Program) (gallery, main ir.Artifact) {
	t.Helper()
	artifacts, err := AssertionModules(program, "runtime", httpDependencies(t))
	if err != nil {
		t.Fatal(err)
	}
	var packages []ir.Artifact
	for _, artifact := range artifacts {
		if strings.HasPrefix(artifact.Path, "packages/") {
			packages = append(packages, artifact)
		}
	}
	if len(packages) != 2 {
		t.Fatalf("authored modules = %d, want 2", len(packages))
	}
	for _, artifact := range packages {
		body := string(artifact.Bytes)
		switch {
		case strings.Contains(body, "gallery::is_positive"):
			gallery = artifact
		case strings.Contains(body, "gallery::main"):
			main = artifact
		}
	}
	if gallery.Path == "" || main.Path == "" {
		t.Fatal("gallery or main module missing")
	}
	return gallery, main
}

// closedRecoveryEmittedNames maps each proven function identity in the
// gallery module to its emitted function name.
func closedRecoveryEmittedNames(t *testing.T, body string) map[string]string {
	t.Helper()
	names := map[string]string{}
	re := regexp.MustCompile(`export async function (\$canFunction\d+)\(`)
	matches := re.FindAllStringSubmatchIndex(body, -1)
	for i, match := range matches {
		name := body[match[2]:match[3]]
		end := len(body)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		segment := body[match[0]:end]
		identity := ""
		for candidate := range closedRecoveryEligibleIdentities {
			if strings.Contains(segment, `invocation:["`+candidate+`"]`) || strings.Contains(segment, `invocation:Object.freeze(["`+candidate+`"])`) {
				identity = candidate
			}
		}
		if identity == "" {
			t.Fatalf("emitted %s matches no eligible identity", name)
		}
		names[identity] = name
	}
	if len(names) != len(closedRecoveryEligibleIdentities) {
		t.Fatalf("emitted functions = %d, want %d", len(names), len(closedRecoveryEligibleIdentities))
	}
	return names
}

// closedRecoveryFunctionText extracts one gallery function from the emitted
// module, renamed to its stable harness name.
func closedRecoveryFunctionText(t *testing.T, body, identity, stable string) string {
	t.Helper()
	names := closedRecoveryEmittedNames(t, body)
	name, ok := names[identity]
	if !ok {
		t.Fatalf("no emitted function for %s", identity)
	}
	head := "export async function " + name
	start := strings.Index(body, head)
	if start < 0 {
		t.Fatalf("emitted %s not found", name)
	}
	rest := body[start+len(head):]
	end := len(body)
	if next := regexp.MustCompile(`export async function \$canFunction\d+\(`).FindStringIndex(rest); next != nil {
		end = start + len(head) + next[0]
	}
	text := body[start:end]
	if strings.Contains(text, "\nimport ") || strings.Contains(text, "\ntype $canType") {
		t.Fatalf("extracted %s carries module head content", identity)
	}
	text = strings.ReplaceAll(text, name, stable)
	stripped, _, err := extractMappings(text)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(stripped, `typeof $canArg0 === "bigint"`); got != 1 {
		t.Fatalf("%s admission guard occurs %d times, want once", identity, got)
	}
	return stripped
}

// closedRecoveryLegacyText lowers the same proven region without the
// recovery proof: identical bindings, branch impossible. The source ID is
// the logical module ID the authored-module emitter passes, never the
// region's on-disk path.
func closedRecoveryLegacyText(t *testing.T, program *check.Program, assembly *programAssembly, entry *ClosedRecoveryProof, stable string) string {
	t.Helper()
	var fn *check.ProgramFunction
	for _, candidate := range program.Functions {
		if candidate != nil && candidate.Symbol != nil && candidate.Identity() == entry.Identity {
			fn = candidate
		}
	}
	if fn == nil {
		t.Fatalf("function %s missing", entry.Identity)
	}
	emitter := RegionEmitter{
		Bindings: assembly.bindings, Functions: assembly.functions,
		DomainRuntime: "$canDomain", SourceID: fn.Symbol.Source.ID, Browser: assembly.browser,
		authoredProof: assembly.authoredProof, collectionProof: assembly.collectionAsyncProof,
		coreProof: assembly.coreAsyncProof, integerWorkers: assembly.integerWorkers,
		mapLeaves: assembly.mapLeaves, mapBatches: assembly.mapBatches,
	}
	legacy, err := emitter.Function(stable, entry.Region)
	if err != nil {
		t.Fatal(err)
	}
	stripped, _, err := extractMappings(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stripped, "$canAmbientOwnerPresent") || strings.Contains(stripped, "$canAllocateOccurrenceID") {
		t.Fatalf("legacy %s carries recovery identifiers", entry.Identity)
	}
	return stripped
}

// closedRecoveryStableNames assigns harness names to the eligible trio.
func closedRecoveryStableNames(identity string) (stable, legacy string) {
	switch identity {
	case "can.project.root/gallery::is_positive":
		return "$crFn", "$crLegacy"
	case "can.project.root/gallery::is_big":
		return "$crBig", "$crLegacyBig"
	case "can.project.root/gallery::is_huge":
		return "$crHuge", "$crLegacyHuge"
	}
	return "", ""
}

func TestClosedRecoveryBranchMarksAndSpans(t *testing.T) {
	program, _ := closedRecoveryFixture(t, closedRecoveryEligible)
	assembly := closedRecoveryAssembly(t, program)
	proof := assembly.checkedClosedRecoveries()
	gallery, main := closedRecoveryModules(t, program)
	body := string(gallery.Bytes)
	if got := strings.Count(body, "$canAmbientOwnerPresent()"); got != len(proof) {
		t.Fatalf("gallery branch admissions = %d, want %d", got, len(proof))
	}
	if strings.Contains(string(main.Bytes), "$canAmbientOwnerPresent") {
		t.Fatal("main module carries recovery identifiers")
	}
	if strings.Contains(string(main.Bytes), "$canAllocateOccurrenceID") {
		t.Fatal("main module carries the occurrence import")
	}
	names := closedRecoveryEmittedNames(t, body)
	for identity := range proof {
		name := names[identity]
		short := identity[strings.LastIndex(identity, ":")+1:]
		// The mapped origin prefix stays on the line immediately after
		// the signature (the frozen workload adapter's binding line),
		// the native branch follows it, and the prefix reuses its lazy
		// frozen slot instead of allocating per call.
		adapter := regexp.MustCompile(`export async function ` + regexp.QuoteMeta(name) + `\([^` + "\n" + `]*` + "\n" + `let \$canOrigin = [^` + "\n" + `]*gallery::` + short + `"`).FindString(body)
		if adapter == "" {
			t.Fatalf("%s prefix no longer identifies the binding", identity)
		}
		if !strings.Contains(adapter, "??=") {
			t.Fatalf("%s prefix is not a cached slot lookup", identity)
		}
		head := strings.Index(body, "export async function "+name+"(")
		if head < 0 {
			t.Fatalf("%s emitted head not found", identity)
		}
		segment := body[head:]
		prefixAt := strings.Index(segment, adapter)
		branchAt := strings.Index(segment, "if ($canContext === undefined")
		if prefixAt < 0 || branchAt < prefixAt {
			t.Fatalf("%s branch does not follow the mapped prefix", identity)
		}
	}
	for identity, entry := range proof {
		cold := fmt.Sprintf(`{source:"%s",start:%d,end:%d,invocation:["%s"]}`,
			entry.Source, entry.Region.Span.Start, entry.Region.Span.End, identity)
		if !strings.Contains(body, "} catch ($canError) { return $canCaught($canError, "+cold+"); }") {
			t.Fatalf("%s cold origin is not the exact function span", identity)
		}
		// Honest inputs cannot execute the cold catch: bigints evaluate
		// `>` without throwing and every other input declines before
		// evaluation. Its contract is therefore the static origin above.
		seenComparison, seenFunction := false, false
		for _, mapping := range gallery.Mappings {
			if mapping.Source != entry.Source {
				continue
			}
			if mapping.Operation == "comparison" && mapping.Start == entry.Predicate.Span.Start && mapping.End == entry.Predicate.Span.End {
				seenComparison = true
			}
			if mapping.Operation == "function" && mapping.Start == entry.Region.Span.Start && mapping.End == entry.Region.Span.End {
				seenFunction = true
			}
		}
		if !seenComparison || !seenFunction {
			t.Fatalf("%s branch mappings missing: %+v", identity, gallery.Mappings)
		}
	}
	_, ids := closedRecoveryIdentities(t, program)
	matched := regexp.MustCompile(`=== "([0-9a-f]{64})"`).FindAllStringSubmatch(body, -1)
	if len(matched) != len(proof) {
		t.Fatalf("match identities = %d, want %d", len(matched), len(proof))
	}
	for _, found := range matched {
		if len(found) != 2 || found[1] != ids["can.std.checks@1::failed"] {
			t.Fatalf("match identity %q is not the model failed identity", found)
		}
	}
}

func TestClosedRecoveryLegacySlowBodyIdentical(t *testing.T) {
	program, _ := closedRecoveryFixture(t, closedRecoveryEligible)
	assembly := closedRecoveryAssembly(t, program)
	proof := assembly.checkedClosedRecoveries()
	gallery, _ := closedRecoveryModules(t, program)
	for identity, entry := range proof {
		stable, legacyName := closedRecoveryStableNames(identity)
		optimized := closedRecoveryFunctionText(t, string(gallery.Bytes), identity, stable)
		branch := strings.Index(optimized, "try {\nif ($canContext === undefined")
		cold := strings.Index(optimized, "} catch ($canError) { return $canCaught($canError,")
		if branch < 0 || cold < branch {
			t.Fatalf("%s branch span not found", identity)
		}
		lineEnd := strings.Index(optimized[cold:], "\n")
		slow := optimized[:branch] + optimized[cold+lineEnd+1:]
		slow = strings.ReplaceAll(slow, stable, "$crBoth")
		slow = strings.Replace(slow, "export async function", "async function", 1)
		legacy := closedRecoveryLegacyText(t, program, assembly, entry, legacyName)
		legacy = strings.ReplaceAll(legacy, legacyName, "$crBoth")
		if slow != legacy {
			t.Fatalf("%s slow body differs from legacy lowering:\n--- slow ---\n%s\n--- legacy ---\n%s", identity, slow, legacy)
		}
	}
}

func TestClosedRecoveryBrowserExcludes(t *testing.T) {
	program, _ := closedRecoveryFixture(t, closedRecoveryEligible)
	assembly := closedRecoveryAssembly(t, program)
	proof := assembly.checkedClosedRecoveries()
	entry := proof["can.project.root/gallery::is_positive"]
	if entry == nil {
		t.Fatal("gallery proof missing")
	}
	var fn *check.ProgramFunction
	for _, candidate := range program.Functions {
		if candidate != nil && candidate.Symbol != nil && candidate.Identity() == entry.Identity {
			fn = candidate
		}
	}
	if fn == nil {
		t.Fatal("gallery function missing")
	}
	browser := RegionEmitter{
		Bindings: assembly.bindings, Functions: assembly.functions,
		DomainRuntime: "$canDomain", SourceID: fn.Symbol.Source.ID, Browser: true,
		authoredProof: assembly.authoredProof, collectionProof: assembly.collectionAsyncProof,
		coreProof: assembly.coreAsyncProof, integerWorkers: assembly.integerWorkers,
		mapLeaves: assembly.mapLeaves, mapBatches: assembly.mapBatches,
		closedRecoveries: assembly.closedRecoveries,
	}
	body, err := browser.Function("$crBrowser", entry.Region)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(body, "$canAmbientOwnerPresent") || strings.Contains(body, "$canAllocateOccurrenceID") || strings.Contains(body, "typeof $canArg0") {
		t.Fatalf("browser lowering carries the native branch:\n%s", body)
	}
	if !strings.Contains(body, "$canChecks.require") {
		t.Fatal("browser lowering lost the require fallback")
	}
}

func TestClosedRecoveryExecutedRoutes(t *testing.T) {
	program, fixture := closedRecoveryFixture(t, closedRecoveryEligible)
	assembly := closedRecoveryAssembly(t, program)
	proof := assembly.checkedClosedRecoveries()
	if len(proof) != len(closedRecoveryEligibleIdentities) {
		t.Fatalf("proven recoveries = %d", len(proof))
	}
	gallery, _ := closedRecoveryModules(t, program)
	plan, ids := closedRecoveryIdentities(t, program)
	runtimeRoot, _ := filepath.Abs("../../../runtime")
	path := func(name string) string {
		absolute, _ := filepath.Abs(filepath.Join(runtimeRoot, name+".ts"))
		return absolute
	}
	contextPath := filepath.Join(runtimeRoot, "assert", "context.ts")
	var script strings.Builder
	script.WriteString(`import {strict as assert} from "node:assert";` + "\n")
	script.WriteString(CompletionImports(path("completion")))
	fmt.Fprintf(&script, "import {createDomainRuntime} from %s;\n", quote(path("domain")))
	fmt.Fprintf(&script, "import {domainFailureDiagnostics} from %s;\n", quote(path("domain")))
	fmt.Fprintf(&script, "import {createChecks as $canCreateChecks} from %s;\n", quote(path("checks")))
	fmt.Fprintf(&script, "import {allocateOccurrenceID as $canAllocateOccurrenceID, standardFailureDiagnostics, standardFailureMessage} from %s;\n", quote(path("failure")))
	fmt.Fprintf(&script, "import {ambientOwnerPresent as $canAmbientOwnerPresent, runOwnedRoot as $canRunOwnedRoot} from %s;\n", quote(path("owner")))
	fmt.Fprintf(&script, "import {assertionContext as $canAssertionContext, contextIdentity as $canContextIdentity, closeContext as $canCloseContext} from %s;\n", quote(contextPath))
	fmt.Fprintf(&script, "const $canDomain = createDomainRuntime(%s);\n", plan)
	fmt.Fprintf(&script, "const $realChecks = $canCreateChecks($canDomain, {failed: %s});\n", quote(ids["can.std.checks@1::failed"]))
	script.WriteString("let requireCalls = 0;\nconst requireArgs = [];\n")
	script.WriteString("const $canChecks = { require: async (condition, reason, origin, context) => { requireCalls++; requireArgs.push([condition, reason, origin]); return $realChecks.require(condition, reason, origin, context); } };\n")
	type route struct {
		identity, stable, legacy, threshold, reason, requireSpan string
		trueInput, falseInput                                    string
	}
	var routes []route
	for identity, entry := range proof {
		stable, legacyName := closedRecoveryStableNames(identity)
		script.WriteString(closedRecoveryFunctionText(t, string(gallery.Bytes), identity, stable))
		script.WriteString(closedRecoveryLegacyText(t, program, assembly, entry, legacyName))
		threshold, err := strconv.Atoi(entry.Threshold)
		if err != nil {
			t.Fatal(err)
		}
		call := fmt.Sprintf("checks::require(number > %s, %q)", entry.Threshold, entry.Reason)
		at := strings.Index(fixture, call)
		if at < 0 {
			t.Fatalf("%s require call not found", identity)
		}
		routes = append(routes, route{
			identity: identity, stable: stable, legacy: legacyName,
			threshold: entry.Threshold, reason: entry.Reason,
			requireSpan: fmt.Sprintf(`{source: "%s", start: %d, end: %d, invocation: ["%s"]}`, entry.Source, at, at+len(call), identity),
			trueInput:   fmt.Sprintf("%dn", threshold+1),
			falseInput:  fmt.Sprintf("%dn", threshold),
		})
	}
	var cases strings.Builder
	for _, item := range routes {
		fmt.Fprintf(&cases, `{
  const nativeTrue = await counts(() => %s(%s));
  assert.deepEqual(shape(nativeTrue.result), {kind: "ok", value: true});
  assert.equal(nativeTrue.calls, 0);
  assert.equal(nativeTrue.ids, 0n);
  const nativeFalse = await counts(() => %s(%s));
  assert.deepEqual(shape(nativeFalse.result), {kind: "ok", value: false});
  assert.equal(nativeFalse.calls, 0);
  assert.equal(nativeFalse.ids, 1n);
  const legacyTrue = await counts(() => %s(%s));
  assert.deepEqual(shape(legacyTrue.result), shape(nativeTrue.result));
  const legacyFalse = await counts(() => %s(%s));
  assert.deepEqual(shape(legacyFalse.result), shape(nativeFalse.result));
}
`, item.stable, item.trueInput, item.stable, item.falseInput, item.legacy, item.trueInput, item.legacy, item.falseInput)
	}
	gallerySpan := ""
	for _, item := range routes {
		if item.identity == "can.project.root/gallery::is_positive" {
			gallerySpan = item.requireSpan
		}
	}
	fmt.Fprintf(&script, `
const shape = (completion) => {
  if (completion.kind === "ok") return {kind: "ok", value: completion.value};
  if (completion.kind === "domain") {
    const details = domainFailureDiagnostics(completion.value);
    return {kind: "domain", type: details.typeIdentity, payload: details.payload, origin: details.origin};
  }
  return {kind: "standard", message: standardFailureMessage(completion.value), boundary: standardFailureDiagnostics(completion.value).boundaryOrigin};
};
const counts = async (run) => {
  const calls = requireCalls, before = $canAllocateOccurrenceID();
  const result = await run();
  const after = $canAllocateOccurrenceID();
  return {result, calls: requireCalls - calls, ids: after - before - 1n};
};
const root = Object.freeze({package: "can.project.root/gallery", declaration: "can.project.root/gallery::is_positive", name: "sample"});
// Native routes for every eligible function: no require call, no reason
// record, no domain token; legacy agrees on honest inputs.
%s
// Defined-context declines keep the exact reason and call-site origin.
const requireOrigin = %s;
const contextTrue = $canAssertionContext(root);
const parentTrue = $canContextIdentity(contextTrue);
const slowTrue = await counts(() => $crFn(3n, contextTrue));
assert.deepEqual(shape(slowTrue.result), {kind: "ok", value: true});
assert.equal(slowTrue.calls, 1);
assert.equal(slowTrue.ids, 0n);
assert.equal($canContextIdentity(contextTrue), parentTrue);
$canCloseContext(contextTrue);
const contextFalse = $canAssertionContext(root);
const slowFalse = await counts(() => $crFn(0n, contextFalse));
assert.deepEqual(shape(slowFalse.result), {kind: "ok", value: false});
assert.equal(slowFalse.calls, 1);
assert.equal(slowFalse.ids, 1n);
$canCloseContext(contextFalse);
const contextLegacy = $canAssertionContext(root);
const slowLegacy = await counts(() => $crLegacy(0n, contextLegacy));
assert.deepEqual(shape(slowLegacy.result), shape(slowFalse.result));
$canCloseContext(contextLegacy);
// Ambient-owner decline with an undefined context: any present store,
// even a closed scope, declines. Cancellation introduces no third signal;
// the closed-scope query contract lives in the runtime owner test.
let owned;
const ownedRoot = await $canRunOwnedRoot(async () => { owned = await counts(() => $crFn(0n)); return $canSuccess(undefined); }, () => {});
assert.equal(ownedRoot.cleanupFailed, false);
assert.deepEqual(shape(owned.result), {kind: "ok", value: false});
assert.equal(owned.calls, 1);
assert.equal(owned.ids, 1n);
// Wrong-host number decline (deliberate host fault injection, not a valid
// Can value): Bun coerces the mixed comparison instead of throwing, so the
// legacy path consumes a domain failure exactly like an honest false while
// the counters prove the native branch declined.
const numberSlow = await counts(() => $crFn(0));
assert.deepEqual(shape(numberSlow.result), {kind: "ok", value: false});
assert.equal(numberSlow.calls, 1);
assert.equal(numberSlow.ids, 1n);
const numberLegacy = await counts(() => $crLegacy(0));
assert.deepEqual(shape(numberLegacy.result), shape(numberSlow.result));
// Proxy decline (deliberate host fault injection): typeof is object so the
// native branch declines; unboxing a BigInt-object proxy throws (valueOf
// rejects proxies) before require, the region catch preserves the standard
// failure through the forwarding else arm, and legacy agrees exactly.
const proxied = new Proxy(Object(0n), {});
const proxySlow = await counts(() => $crFn(proxied));
assert.equal(proxySlow.result.kind, "standard");
assert.equal(proxySlow.calls, 0);
assert.equal(proxySlow.ids, 1n);
const proxyLegacy = await counts(() => $crLegacy(proxied));
assert.deepEqual(shape(proxyLegacy.result), shape(proxySlow.result));
// Every consumed require kept its authored reason and call-site origin.
const reasons = new Set(["positive required", "ten required", "hundred required"]);
for (const [condition, reason, origin] of requireArgs) {
  assert.equal(typeof condition, "boolean");
  assert.equal(reasons.has(reason), true);
  assert.equal(origin.source, "can.project.root/gallery/recovery.can");
}
const galleryOrigins = requireArgs.filter(([, , origin]) => origin.invocation[0] === "can.project.root/gallery::is_positive").map(([, , origin]) => origin);
assert.ok(galleryOrigins.length > 0);
for (const origin of galleryOrigins) assert.deepEqual(origin, requireOrigin);
console.log("closed recovery routes passed");
`, cases.String(), gallerySpan)
	out := coreRun(t, "closed-recovery-routes.ts", script.String())
	if !strings.Contains(out, "closed recovery routes passed") {
		t.Fatalf("route execution failed:\n%s", out)
	}
}
