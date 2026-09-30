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

// coreCorpus exercises text, bytes and checks operations so the checked
// program model carries every factory identity the tests need.
const coreCorpus = `package app
    provides []
    uses [text, bytes, codec, checks]
fn str text_flow
    emits {}
    asserts
        sample: => ok "hi"
    ok call "  HI  ".trim().to_lower_case()
fn int split_flow
    emits {text::empty_separator}
    asserts
        sample: => ok 2
    match chain
        call "a,b".split(",") as str[] parts
        text::empty_separator
        ok => ok parts.length
fn str byte_flow
    emits {codec::invalid_data}
    asserts
        sample: => ok "aGk="
    match chain
        call bytes::from_utf8("hi") as bytes::buffer raw
        codec::invalid_data
        ok => ok call bytes::encode_base64(raw)
fn bool check_flow
    emits {}
    asserts
        sample: => ok true
    match call checks::require(true, "ok")
        checks::failed => ok false
        ok => ok true
fn void main
    emits {}
    given
        str[] arguments
    asserts
        empty: [] => ok
    ok
`

// coreAuditPairs is the frozen 26-pair whitelist under test, mirroring the
// saved source audit without reading evidence files.
var coreAuditPairs = map[string]string{
	"can.intrinsic.str@1::ends_with":     "$canText.endsWith",
	"can.intrinsic.str@1::includes":      "$canText.includes",
	"can.intrinsic.str@1::replace_all":   "$canText.replaceAll",
	"can.intrinsic.str@1::slice":         "$canText.slice",
	"can.intrinsic.str@1::split":         "$canText.split",
	"can.intrinsic.str@1::starts_with":   "$canText.startsWith",
	"can.intrinsic.str@1::to_lower_case": "$canText.toLowerCase",
	"can.intrinsic.str@1::to_upper_case": "$canText.toUpperCase",
	"can.intrinsic.str@1::trim":          "$canText.trim",
	"can.std.bytes@1::decode_base64":     "$canBytes.decodeBase64",
	"can.std.bytes@1::decode_hex":        "$canBytes.decodeHex",
	"can.std.bytes@1::empty":             "$canBytes.empty",
	"can.std.bytes@1::encode_base64":     "$canBytes.encodeBase64",
	"can.std.bytes@1::encode_hex":        "$canBytes.encodeHex",
	"can.std.bytes@1::from_ints":         "$canBytes.fromInts",
	"can.std.bytes@1::from_utf8":         "$canBytes.fromUTF8",
	"can.std.bytes@1::to_ints":           "$canBytes.toInts",
	"can.std.bytes@1::to_utf8":           "$canBytes.toUTF8",
	"can.std.checks@1::require":          "$canChecks.require",
	"can.std.text@1::compile_regex":      "$canText.compileRegex",
	"can.std.text@1::from_scalars":       "$canText.fromScalars",
	"can.std.text@1::graphemes":          "$canText.graphemes",
	"can.std.text@1::join":               "$canText.join",
	"can.std.text@1::matches":            "$canText.findMatches",
	"can.std.text@1::normalize_nfc":      "$canText.normalizeNFC",
	"can.std.text@1::scalars":            "$canText.scalars",
}

func coreProgram(t *testing.T) *check.Program {
	t.Helper()
	return actionEmitProgram(t, map[string]string{"src/main.can": coreCorpus})
}

func coreAssembly(t *testing.T, program *check.Program) *programAssembly {
	t.Helper()
	assembly, err := assembleProgramBindings(program)
	if err != nil {
		t.Fatal(err)
	}
	return assembly
}

func TestCoreProofAssemblyExact(t *testing.T) {
	if len(coreAsyncTargets) != 26 {
		t.Fatalf("core table holds %d pairs, want 26", len(coreAsyncTargets))
	}
	for identity, want := range coreAuditPairs {
		if coreAsyncTargets[identity] != want {
			t.Fatalf("core table pair %s = %q, want %q", identity, coreAsyncTargets[identity], want)
		}
	}
	program := coreProgram(t)
	assembly := coreAssembly(t, program)
	if len(assembly.coreAsyncProof) != 26 {
		t.Fatalf("core proof holds %d pairs, want 26", len(assembly.coreAsyncProof))
	}
	for identity, want := range coreAuditPairs {
		bound, ok := assembly.coreAsyncProof[identity]
		if !ok {
			t.Fatalf("canonical %s missing from proof", identity)
		}
		if bound != want || bound != assembly.functions[identity] {
			t.Fatalf("%s proof %q disagrees with audited %q merged %q", identity, bound, want, assembly.functions[identity])
		}
		if _, clash := assembly.authoredProof[identity]; clash {
			t.Fatalf("core %s leaked into authored proof", identity)
		}
		if _, clash := assembly.collectionAsyncProof[identity]; clash {
			t.Fatalf("core %s leaked into collection proof", identity)
		}
	}
}

func TestCoreProofAssemblyNegatives(t *testing.T) {
	program := coreProgram(t)
	assembly := coreAssembly(t, program)
	build := func(mut func(functions map[string]string)) map[string]string {
		functions := map[string]string{}
		for key, target := range assembly.functions {
			functions[key] = target
		}
		mut(functions)
		probe := &programAssembly{program: program, functions: functions}
		return probe.checkedCoreProof()
	}
	full := build(func(map[string]string) {})
	if len(full) != 26 {
		t.Fatal("faithful copy lost proof entries")
	}
	trim, lower := "can.intrinsic.str@1::trim", "can.intrinsic.str@1::to_lower_case"
	cases := map[string]func(map[string]string){
		"rebound target": func(functions map[string]string) {
			functions[trim] = "$canText.trimmed"
		},
		"cross receiver": func(functions map[string]string) {
			functions[trim], functions[lower] = functions[lower], functions[trim]
		},
		"missing target": func(functions map[string]string) {
			delete(functions, trim)
		},
		"empty target": func(functions map[string]string) {
			functions[trim] = ""
		},
	}
	for name, mut := range cases {
		proof := build(mut)
		if _, kept := proof[trim]; kept {
			t.Fatalf("%s kept trim proof", name)
		}
		want := 25
		if name == "cross receiver" {
			if _, kept := proof[lower]; kept {
				t.Fatalf("%s kept swapped proof", name)
			}
			want = 24
		}
		if len(proof) != want {
			t.Fatalf("%s proof size %d, want %d", name, len(proof), want)
		}
	}
	// Non-whitelisted merged bindings never qualify.
	for _, other := range []string{"can.std.html@1::text", "can.std.io@1::stdin_text", "can.std.text@1::from_int"} {
		if _, ok := assembly.functions[other]; !ok {
			t.Fatalf("expected merged binding for %s", other)
		}
		if _, kept := full[other]; kept {
			t.Fatalf("non-core %s qualified", other)
		}
	}
	if proof := (&programAssembly{program: program, functions: map[string]string{}}).checkedCoreProof(); len(proof) != 0 {
		t.Fatalf("empty merge kept %d pairs", len(proof))
	}
}

func TestCoreEligibilityPredicate(t *testing.T) {
	proof := map[string]string{
		"can.intrinsic.str@1::trim":  "$canText.trim",
		"can.std.bytes@1::from_utf8": "$canBytes.fromUTF8",
		"can.std.checks@1::require":  "$canChecks.require",
	}
	emitter := RegionEmitter{coreProof: proof}
	plain := func() *ir.InvocationStep {
		return &ir.InvocationStep{Identity: "can.intrinsic.str@1::trim", Site: "s1"}
	}
	if !emitter.eligibleCoreBypass(plain(), "$canText.trim") {
		t.Fatal("exact core proof did not qualify")
	}
	if !emitter.provenCoreBinding("can.std.bytes@1::from_utf8", "$canBytes.fromUTF8") {
		t.Fatal("exact bytes proof did not qualify")
	}
	if !emitter.provenCoreBinding("can.std.checks@1::require", "$canChecks.require") {
		t.Fatal("exact checks proof did not qualify")
	}
	cases := map[string]struct {
		emitter RegionEmitter
		step    *ir.InvocationStep
		target  string
	}{
		"nil proof":          {RegionEmitter{}, plain(), "$canText.trim"},
		"empty proof":        {RegionEmitter{coreProof: map[string]string{}}, plain(), "$canText.trim"},
		"stale target":       {emitter, plain(), "$canText.trimmed"},
		"unknown identity":   {emitter, &ir.InvocationStep{Identity: "can.std.text@1::from_int", Site: "s1"}, "$canNumbers.fromInt"},
		"cross method":       {emitter, plain(), "$canText.toLowerCase"},
		"cross receiver":     {emitter, &ir.InvocationStep{Identity: "can.std.bytes@1::from_utf8", Site: "s1"}, "$canText.fromUTF8"},
		"bare method":        {emitter, plain(), "trim"},
		"empty identity":     {emitter, &ir.InvocationStep{Site: "s1"}, "$canText.trim"},
		"empty target":       {emitter, plain(), ""},
		"nil step":           {emitter, nil, "$canText.trim"},
		"browser":            {RegionEmitter{Browser: true, coreProof: proof}, plain(), "$canText.trim"},
		"authored ignored":   {RegionEmitter{authoredProof: map[string]string{"can.intrinsic.str@1::trim": "$canText.trim"}}, plain(), "$canText.trim"},
		"collection ignored": {RegionEmitter{collectionProof: map[string]string{"can.intrinsic.str@1::trim": "$canText.trim"}}, plain(), "$canText.trim"},
		"callee":             {emitter, &ir.InvocationStep{Identity: "can.intrinsic.str@1::trim", Site: "s1", Callee: &ir.Expression{}}, "$canText.trim"},
		"native":             {emitter, &ir.InvocationStep{Identity: "can.intrinsic.str@1::trim", Site: "s1", Native: &ir.Expression{}}, "$canText.trim"},
		"array":              {emitter, &ir.InvocationStep{Identity: "can.intrinsic.str@1::trim", Site: "s1", Array: &ir.ArrayOperation{}}, "$canText.trim"},
		"asset":              {emitter, &ir.InvocationStep{Identity: "can.intrinsic.str@1::trim", Site: "s1", Asset: &ir.AssetResolution{}}, "$canText.trim"},
		"fixtures":           {emitter, &ir.InvocationStep{Identity: "can.intrinsic.str@1::trim", Site: "s1", Fixtures: &ir.FixtureTable{}}, "$canText.trim"},
		"sql":                {emitter, &ir.InvocationStep{Identity: "can.intrinsic.str@1::trim", Site: "s1", SQL: &ir.SQLCallSite{}}, "$canText.trim"},
		"form action":        {emitter, &ir.InvocationStep{Identity: "can.intrinsic.str@1::trim", Site: "s1", FormAction: &ir.FormActionSite{}}, "$canText.trim"},
		"json fetch":         {emitter, &ir.InvocationStep{Identity: "can.intrinsic.str@1::trim", Site: "s1", JSONFetch: &ir.JSONFetchSite{}}, "$canText.trim"},
		"action":             {emitter, &ir.InvocationStep{Identity: "can.intrinsic.str@1::trim", Site: "s1", Action: &ir.ActionSite{}}, "$canText.trim"},
	}
	for name, tc := range cases {
		if tc.emitter.eligibleCoreBypass(tc.step, tc.target) {
			t.Fatalf("%s qualified for core bypass", name)
		}
	}
	if len(proof) != 3 {
		t.Fatalf("shared proof mutated: %v", proof)
	}
}

// coreBun resolves the installed Bun binary. Core contracts never skip.
func coreBun(t *testing.T) string {
	t.Helper()
	if bun := os.Getenv("CAN_BUN"); bun != "" {
		return bun
	}
	bun, err := exec.LookPath("bun")
	if err != nil {
		t.Fatal("bun is required for core execution")
	}
	return bun
}

func coreRun(t *testing.T, name, code string) string {
	t.Helper()
	file := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(file, []byte(code), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, coreBun(t), "--no-install", "--no-env-file", file).CombinedOutput()
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	return string(out)
}

// coreIdentities collects the factory error identities a real program model
// carries, plus a domain plan covering them.
func coreIdentities(t *testing.T, program *check.Program) (plan string, ids map[string]string) {
	t.Helper()
	ids = map[string]string{}
	failures := []*types.Type{}
	seen := map[string]bool{}
	for _, typ := range program.Model.Types() {
		ids[typ.Declaration()] = typ.Identity()
	}
	for _, declaration := range []string{
		"can.std.text@1::empty_separator", "can.std.codec@1::invalid_data", "can.std.checks@1::failed",
	} {
		found := false
		for _, typ := range program.Model.Types() {
			if typ.Declaration() == declaration {
				found = true
				if !seen[typ.Identity()] {
					seen[typ.Identity()] = true
					failures = append(failures, typ)
				}
			}
		}
		if !found {
			t.Fatalf("model lacks %s", declaration)
		}
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

func TestCoreFactoryPremise(t *testing.T) {
	program := coreProgram(t)
	assembly := coreAssembly(t, program)
	_ = assembly
	plan, ids := coreIdentities(t, program)
	runtimeRoot, _ := filepath.Abs("../../../runtime")
	path := func(name string) string { p, _ := filepath.Abs(filepath.Join(runtimeRoot, name+".ts")); return p }
	code := `import {strict as assert} from "node:assert";` + "\n" +
		CompletionImports(path("completion")) +
		fmt.Sprintf("import {createDomainRuntime} from %s;\n", quote(path("domain"))) +
		fmt.Sprintf("import {createText as $canCreateText} from %s;\n", quote(path("text"))) +
		fmt.Sprintf("import {createBytes as $canCreateBytes} from %s;\n", quote(path("bytes"))) +
		fmt.Sprintf("import {createChecks as $canCreateChecks} from %s;\n", quote(path("checks"))) +
		fmt.Sprintf("const $canDomain = createDomainRuntime(%s);\n", plan) +
		fmt.Sprintf("const text = $canCreateText($canDomain, {emptySeparator: %s, emptyPattern: %s, invalidUnicode: %s, invalidRegex: %s, invalidLimit: %s, match: %s});\n",
			quote(ids["can.std.text@1::empty_separator"]), quote(ids["can.std.text@1::empty_pattern"]), quote(ids["can.std.text@1::invalid_unicode"]),
			quote(ids["can.std.text@1::invalid_regex"]), quote(ids["can.std.text@1::invalid_limit"]), quote(ids["can.std.text@1::regex_match"])) +
		fmt.Sprintf("const bytes = $canCreateBytes($canDomain, %s);\n", quote(ids["can.std.codec@1::invalid_data"])) +
		fmt.Sprintf("const checks = $canCreateChecks($canDomain, {failed: %s});\n", quote(ids["can.std.checks@1::failed"])) + `
const textMethods = ["endsWith", "includes", "replaceAll", "slice", "split", "startsWith", "toLowerCase", "toUpperCase", "trim", "compileRegex", "fromScalars", "graphemes", "join", "findMatches", "normalizeNFC", "scalars"];
const bytesMethods = ["decodeBase64", "decodeHex", "empty", "encodeBase64", "encodeHex", "fromInts", "fromUTF8", "toInts", "toUTF8"];
assert.equal(Object.isFrozen(text), true);
assert.equal(Object.isFrozen(bytes), true);
assert.equal(Object.isFrozen(checks), true);
for (const name of textMethods) {
  assert.equal(Object.hasOwn(text, name), true);
  assert.equal(text[name].constructor.name, "AsyncFunction");
}
for (const name of bytesMethods) {
  assert.equal(Object.hasOwn(bytes, name), true);
  assert.equal(bytes[name].constructor.name, "AsyncFunction");
}
assert.equal(Object.hasOwn(checks, "require"), true);
assert.equal(checks.require.constructor.name, "AsyncFunction");
assert.equal($canValue(await text.trim("  HI  ", undefined)), "HI");
assert.equal($canValue(await text.toLowerCase("HI", undefined)), "hi");
assert.deepEqual($canValue(await text.split("a,b", ",", undefined)), ["a", "b"]);
assert.equal((await text.split("a,b", "", undefined)).kind, "domain");
assert.equal($canValue(await bytes.encodeBase64($canValue(await bytes.fromUTF8("hi", undefined)), undefined)), "aGk=");
assert.equal($canValue(await bytes.toUTF8($canValue(await bytes.fromUTF8("hi", undefined)), undefined)), "hi");
assert.equal((await bytes.decodeBase64("!!!", undefined)).kind, "domain");
const site = {source: "premise", start: 0, end: 1, invocation: []};
assert.equal((await checks.require(true, "ok", site, undefined)).kind, "ok");
assert.equal((await checks.require(false, "nope", site, undefined)).kind, "domain");
console.log("core premise passed");
`
	out := coreRun(t, "core-premise.ts", code)
	if !strings.Contains(out, "core premise passed") {
		t.Fatalf("factory premise failed:\n%s", out)
	}
}

func TestCoreExecutesEligibleRoutes(t *testing.T) {
	program := coreProgram(t)
	assembly := coreAssembly(t, program)
	fixture := newRegionFixture(t)
	mustCallable := func(result *types.Type, inputs ...*types.Type) *types.Type {
		contract, err := types.CallableOfChecked(result, inputs, nil)
		if err != nil {
			t.Fatal(err)
		}
		return contract
	}
	strT, boolT, voidT, bufferT := fixture.ts["str"], fixture.ts["bool"], fixture.ts["void"], fixture.ts["bytes::buffer"]
	fixture.functions["ftrim"] = check.ValueBinding{Identity: "can.intrinsic.str@1::trim", Type: mustCallable(strT, strT)}
	fixture.functions["flower"] = check.ValueBinding{Identity: "can.intrinsic.str@1::to_lower_case", Type: mustCallable(strT, strT)}
	fixture.functions["fnfc"] = check.ValueBinding{Identity: "can.std.text@1::normalize_nfc", Type: mustCallable(strT, strT)}
	fixture.functions["ffutf8"] = check.ValueBinding{Identity: "can.std.bytes@1::from_utf8", Type: mustCallable(bufferT, strT)}
	fixture.functions["ftutf8"] = check.ValueBinding{Identity: "can.std.bytes@1::to_utf8", Type: mustCallable(strT, bufferT)}
	fixture.functions["fenc64"] = check.ValueBinding{Identity: "can.std.bytes@1::encode_base64", Type: mustCallable(strT, bufferT)}
	fixture.functions["fdec64"] = check.ValueBinding{Identity: "can.std.bytes@1::decode_base64", Type: mustCallable(bufferT, strT)}
	fixture.functions["freq"] = check.ValueBinding{Identity: "can.std.checks@1::require", Type: mustCallable(voidT, boolT, strT)}
	// hostile models opaque boxed values: the checker sees bytes::buffer,
	// the harness passes hostile objects the factory refuses uninspected.
	fixture.values["hostile"] = check.ValueBinding{Identity: "value/hostile", Type: bufferT}
	fixture.values["hostileg"] = check.ValueBinding{Identity: "value/hostileg", Type: bufferT}
	bodies := map[string]string{
		"$textFlow": "    ok call flower(call fnfc(call ftrim(\"  HI  \")))\n",
		"$byteFlow": "    ok call fenc64(call ffutf8(\"hi\"))\n",
		"$badB64":   "    ok call fdec64(\"!!!\")\n",
		"$reqOk":    "    call freq(true, \"ok\")\n    ok\n",
		"$reqFail":  "    call freq(false, \"nope\")\n    ok\n",
		"$hostile":  "    ok call ftutf8(hostile)\n",
		"$hostileg": "    ok call ftutf8(hostileg)\n",
		"$defined":  "    ok call flower(call ftrim(\"  AB  \"))\n",
	}
	results := map[string]string{"$badB64": "bytes::buffer"}
	bindings := map[string]string{"value/hostile": "$hostileThen", "value/hostileg": "$hostileGet"}
	type emitted struct {
		body   string
		region *ir.Region
	}
	callers := map[string]emitted{}
	var regions []*ir.Region
	names := []string{"$textFlow", "$byteFlow", "$badB64", "$reqOk", "$reqFail", "$hostile", "$hostileg", "$defined"}
	for _, name := range names {
		result := results[name]
		if result == "" {
			result = map[string]string{"$reqOk": "void", "$reqFail": "void"}[name]
		}
		if result == "" {
			result = "str"
		}
		region, err := fixture.region(t, bodies[name], result, nil, ir.FunctionRegion)
		if err != nil {
			t.Fatal(name, err)
		}
		emitter := RegionEmitter{Bindings: bindings, Functions: assembly.functions, SourceID: "region.can", coreProof: assembly.coreAsyncProof}
		body, err := emitter.Function(name, region)
		if err != nil {
			t.Fatal(name, err)
		}
		callers[name] = emitted{body: body, region: region}
		regions = append(regions, region)
	}
	for _, name := range names {
		body := callers[name].body
		if !strings.Contains(body, "$canContext === undefined ? $can") {
			t.Fatalf("%s missed the core bypass branch:\n%s", name, body)
		}
	}
	textBody := callers["$textFlow"].body
	for _, want := range []string{
		"($canContext === undefined ? $canText.toLowerCase(",
		"($canContext === undefined ? $canText.normalizeNFC(",
		"($canContext === undefined ? $canText.trim(",
	} {
		if !strings.Contains(textBody, want) {
			t.Fatalf("text flow missed %q:\n%s", want, textBody)
		}
	}
	// One textual occurrence per branch; the ternary executes exactly one.
	if got := strings.Count(textBody, "=== undefined ? $canText.trim("); got != 1 {
		t.Fatalf("trim direct branch %d times, want once:\n%s", got, textBody)
	}
	if got := strings.Count(textBody, "=> $canText.trim("); got != 1 {
		t.Fatalf("trim defined branch %d times, want once:\n%s", got, textBody)
	}
	trimAt, nfcAt, lowerAt := strings.Index(textBody, "$canText.trim("), strings.Index(textBody, "$canText.normalizeNFC("), strings.Index(textBody, "$canText.toLowerCase(")
	if !(trimAt < nfcAt && nfcAt < lowerAt) {
		t.Fatalf("operand order broken:\n%s", textBody)
	}
	if got := strings.Count(textBody, "$canCallableInstance($canText.toLowerCase)"); got != 1 {
		t.Fatalf("lower receipt evaluated %d times, want once in defined branch:\n%s", got, textBody)
	}
	// The defined branch is byte-identical to the legacy wrapper.
	legacyRegion, err := fixture.region(t, bodies["$textFlow"], "str", nil, ir.FunctionRegion)
	if err != nil {
		t.Fatal(err)
	}
	legacyEmitter := RegionEmitter{Bindings: bindings, Functions: assembly.functions, SourceID: "region.can"}
	legacy, err := legacyEmitter.Function("$legacy", legacyRegion)
	if err != nil {
		t.Fatal(err)
	}
	needle := "() => $canCallContext($canContext,"
	at := strings.LastIndex(legacy, needle)
	rest := legacy[at+len("() => "):]
	end := len(rest)
	for _, terminator := range []string{", ($canOrigin_", ", {source:"} {
		if cut := strings.Index(rest, terminator); cut >= 0 && cut < end {
			end = cut
		}
	}
	if !strings.Contains(textBody, " : "+rest[:end]+")") {
		t.Fatalf("defined branch differs from legacy lowering:\n%s", textBody)
	}
	// require keeps its hidden exact call-site origin before the context
	// in the direct branch.
	header := "package app\n    provides []\n    uses []\nfn void run\n    emits {}\n    asserts\n        test: => ok 1\n"
	reqSource := header + bodies["$reqFail"]
	reqStart := strings.Index(reqSource, "freq(false")
	reqSpan := fmt.Sprintf(`{source: "region.can", start: %d, end: %d, invocation: ["app::run"]}`, reqStart, reqStart+len(`freq(false, "nope")`))
	reqBody := callers["$reqFail"].body
	direct := reqBody[strings.Index(reqBody, "($canContext === undefined ? ")+len("($canContext === undefined ? "):]
	direct = direct[:strings.Index(direct, " : ")]
	if !strings.HasPrefix(direct, "$canChecks.require(") || !strings.HasSuffix(direct, ", $canContext)") {
		t.Fatalf("require direct branch malformed:\n%s", reqBody)
	}
	// Cached or literal origins both embed the exact numeric span.
	spanTuple := fmt.Sprintf("start:%d,end:%d", reqStart, reqStart+len(`freq(false, "nope")`))
	if !strings.Contains(direct, spanTuple) {
		t.Fatalf("require direct branch lost hidden origin %s:\n%s", spanTuple, reqBody)
	}
	hostHeader := "package app\n    provides []\n    uses []\nfn str run\n    emits {}\n    asserts\n        test: => ok 1\n"
	hostSource := hostHeader + bodies["$hostile"]
	hostStart := strings.Index(hostSource, "ftutf8(")
	hostCall := "ftutf8(hostile)"
	declarations, err := RegionTypeDeclarations(regions...)
	if err != nil {
		t.Fatal(err)
	}
	plan, ids := coreIdentities(t, program)
	runtimeRoot, _ := filepath.Abs("../../../runtime")
	path := func(name string) string { p, _ := filepath.Abs(filepath.Join(runtimeRoot, name+".ts")); return p }
	contextPath := filepath.Join(runtimeRoot, "assert", "context.ts")
	var programTS strings.Builder
	programTS.WriteString(`import {strict as assert} from "node:assert";` + "\n")
	programTS.WriteString(CompletionImports(path("completion")))
	fmt.Fprintf(&programTS, "import {createDomainRuntime} from %s;\n", quote(path("domain")))
	fmt.Fprintf(&programTS, "import {domainFailureDiagnostics} from %s;\n", quote(path("domain")))
	fmt.Fprintf(&programTS, "import {standardFailureDiagnostics} from %s;\n", quote(path("failure")))
	fmt.Fprintf(&programTS, "import {createText as $canCreateText} from %s;\n", quote(path("text")))
	fmt.Fprintf(&programTS, "import {createBytes as $canCreateBytes} from %s;\n", quote(path("bytes")))
	fmt.Fprintf(&programTS, "import {createChecks as $canCreateChecks} from %s;\n", quote(path("checks")))
	fmt.Fprintf(&programTS, "import {assertionContext as $canAssertionContext, contextIdentity as $canContextIdentity, closeContext as $canCloseContext} from %s;\n", quote(contextPath))
	fmt.Fprintf(&programTS, "const $canDomain = createDomainRuntime(%s);\n", plan)
	fmt.Fprintf(&programTS, "const $canText = $canCreateText($canDomain, {emptySeparator: %s, emptyPattern: %s, invalidUnicode: %s, invalidRegex: %s, invalidLimit: %s, match: %s});\n",
		quote(ids["can.std.text@1::empty_separator"]), quote(ids["can.std.text@1::empty_pattern"]), quote(ids["can.std.text@1::invalid_unicode"]),
		quote(ids["can.std.text@1::invalid_regex"]), quote(ids["can.std.text@1::invalid_limit"]), quote(ids["can.std.text@1::regex_match"]))
	fmt.Fprintf(&programTS, "const $canBytes = $canCreateBytes($canDomain, %s);\n", quote(ids["can.std.codec@1::invalid_data"]))
	fmt.Fprintf(&programTS, "const $canChecks = $canCreateChecks($canDomain, {failed: %s});\n", quote(ids["can.std.checks@1::failed"]))
	programTS.WriteString(declarations)
	for _, name := range names {
		stripped, _, err := extractMappings(callers[name].body)
		if err != nil {
			t.Fatal(err)
		}
		programTS.WriteString(stripped)
	}
	fmt.Fprintf(&programTS, `
let thenCalls = 0, getterCalls = 0;
const $hostileThen = {then() { thenCalls++; throw new Error("assimilated"); }};
const $hostileGet = {};
Object.defineProperty($hostileGet, "then", {
  enumerable: true, configurable: true,
  get() { getterCalls++; throw new Error("assimilated getter"); },
});
assert.equal($canValue(await $textFlow()), "hi");
assert.equal($canValue(await $byteFlow()), "aGk=");
assert.equal((await $badB64()).kind, "domain");
assert.equal((await $reqOk()).kind, "ok");
const refused = await $reqFail();
assert.equal(refused.kind, "domain");
assert.equal(domainFailureDiagnostics(refused.value).payload.reason, "nope");
assert.deepEqual(domainFailureDiagnostics(refused.value).origin, %s);
const hostileSpan = {source: "region.can", start: %d, end: %d, invocation: ["app::run"]};
const h1 = await $hostile(), h2 = await $hostile(), h3 = await $hostileg();
assert.equal(h1.kind, "standard");
assert.equal(h2.kind, "standard");
assert.equal(h3.kind, "standard");
assert.deepEqual(standardFailureDiagnostics(h1.value).boundaryOrigin, hostileSpan);
assert.notEqual(standardFailureDiagnostics(h1.value).occurrenceID, standardFailureDiagnostics(h2.value).occurrenceID);
assert.equal(thenCalls, 0);
assert.equal(getterCalls, 0);
const root = Object.freeze({package: "can.project.root/app", declaration: "can.project.root/app::subject", name: "sample"});
const context = $canAssertionContext(root);
const parent = $canContextIdentity(context);
assert.equal($canValue(await $defined(context)), "ab");
assert.equal($canContextIdentity(context), parent);
$canCloseContext(context);
console.log("core routes passed");
`, reqSpan, hostStart, hostStart+len(hostCall))
	out := coreRun(t, "core-routes.ts", programTS.String())
	if !strings.Contains(out, "core routes passed") {
		t.Fatalf("route execution failed:\n%s", out)
	}
}
