package emit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
)

// batchBun requires the pinned Bun for emitted-route execution. G80
// proves actual checked/emitted routes with zero skips.
func batchBun(t *testing.T) string {
	t.Helper()
	bun := os.Getenv("CAN_BUN")
	if bun == "" {
		t.Fatal("G80 requires CAN_BUN for emitted batch-route execution")
	}
	return bun
}

// batchCompanionText emits both proven value companions and returns
// their mapping-free TypeScript plus extracted source mappings.
func batchCompanionText(t *testing.T, assembly *programAssembly, fn *check.ProgramFunction, entry *MapBatchProof) (string, string, []ir.Mapping) {
	t.Helper()
	emitter := &RegionEmitter{Functions: assembly.functions, SourceID: fn.Symbol.Source.ID}
	absent, present, err := emitter.BatchCompanions(entry)
	if err != nil {
		t.Fatal(err)
	}
	cleanAbsent, mapsAbsent, err := extractMappings(absent)
	if err != nil {
		t.Fatal(err)
	}
	cleanPresent, mapsPresent, err := extractMappings(present)
	if err != nil {
		t.Fatal(err)
	}
	if len(mapsAbsent) == 0 || len(mapsPresent) == 0 {
		t.Fatal("emitted batch companions carry no source mappings")
	}
	return cleanAbsent, cleanPresent, append(mapsAbsent, mapsPresent...)
}

func TestMapBatchGalleryCallbackEligible(t *testing.T) {
	frequency, err := os.ReadFile(filepath.Join("..", "..", "..", "tools", "performance", "fixtures", "runtime", "src", "frequency.can"))
	if err != nil {
		t.Fatal(err)
	}
	program := mapLeafGalleryProgram(t)
	assembly := mapLeafAssembly(t, program)
	proof := assembly.checkedMapBatches()
	counter := mapLeafFunction(t, program, "gallery::count_one")
	entry, ok := proof[counter.Identity()]
	if !ok {
		t.Fatal("actual gallery count_one lacks a batch proof")
	}
	if entry.Target == "" || entry.Absent != entry.Target+MapBatchAbsentSuffix || entry.Present != entry.Target+MapBatchPresentSuffix {
		t.Errorf("bad companion derivation: %+v", entry)
	}
	if entry.Source != counter.Symbol.Source.ID || entry.Source == "" {
		t.Errorf("bad proof source: %+v", entry)
	}
	if entry.Region != counter.Region {
		t.Error("proof region is not the checked gallery region")
	}
	if entry.KeyKind != "str" {
		t.Errorf("KeyKind %q, want str", entry.KeyKind)
	}
	if entry.Receiver == "" {
		t.Error("proof lacks the exact factory receiver")
	}
	if len(entry.Calls) != 3 {
		t.Fatalf("want get/insert/replace call targets, have %v", entry.Calls)
	}
	for key, target := range entry.Calls {
		if !strings.HasPrefix(key, "can.std.collections@1::") {
			t.Errorf("non-canonical call key %s", key)
		}
		method := target[strings.LastIndex(target, ".")+1:]
		if method != "get" && method != "insert" && method != "replace" {
			t.Errorf("unexpected batch call target %s", target)
		}
		if head := target[:strings.LastIndex(target, ".")]; head != entry.Receiver {
			t.Errorf("batch call %s escapes receiver %s", target, entry.Receiver)
		}
	}
	if entry.AbsentValue == nil || entry.AbsentValue.Kind != ir.Literal || entry.AbsentValue.Text != "1" {
		t.Errorf("bad absent value: %+v", entry.AbsentValue)
	}
	if entry.PresentValue == nil || entry.PresentValue.Kind != ir.Binary || entry.PresentValue.Text != "+" {
		t.Errorf("bad present value: %+v", entry.PresentValue)
	}
	if entry.Previous == "" {
		t.Error("proof lacks the old-value binding")
	}
	getStart, getEnd := leafFixtureCallSpan(t, string(frequency), "count_one", "collections::get(totals, word)")
	if entry.GetSpan.Start != getStart || entry.GetSpan.End != getEnd {
		t.Errorf("get span %+v, want text span %d-%d", entry.GetSpan, getStart, getEnd)
	}
	if _, ok := proof[mapLeafFunction(t, program, "gallery::frequency").Identity()]; ok {
		t.Error("gallery frequency must not qualify as a batch")
	}
}

func TestMapBatchCorpusKeys(t *testing.T) {
	program := mapLeafProgram(t, map[string]string{"src/main.can": mapLeafCorpus})
	assembly := mapLeafAssembly(t, program)
	proof := assembly.checkedMapBatches()
	for identity, keyKind := range map[string]string{"app::count_str": "str", "app::count_int": "int", "app::count_bool": "bool"} {
		entry, ok := proof[mapLeafFunction(t, program, identity).Identity()]
		if !ok {
			t.Errorf("%s lacks a batch proof", identity)
			continue
		}
		if entry.KeyKind != keyKind || len(entry.Calls) != 3 || entry.AbsentValue == nil || entry.PresentValue == nil {
			t.Errorf("%s proof wrong: %+v", identity, entry)
		}
	}
	for _, identity := range []string{"app::uses_remove", "app::wrong_value", "app::has_statement"} {
		if _, ok := proof[mapLeafFunction(t, program, identity).Identity()]; ok {
			t.Errorf("%s must not batch-qualify", identity)
		}
	}
}

const mapBatchAdversarialCorpus = `package app
    provides []
    uses [collections]
fn collections::map<str,int> literal_key
    emits []
    given
        collections::map<str,int> totals
        str word
    asserts
        sample: call collections::empty_map<str,int>(), "a" => ok
    match call collections::get(totals, "a")
        collections::key_absent => match call collections::insert(totals, "a", 1)
            collections::key_exists => ok totals
            ok collections::map<str,int> next => ok next
        ok int previous => match call collections::replace(totals, "a", previous + 1)
            collections::key_absent => ok totals
            ok collections::map<str,int> next => ok next
fn collections::map<str,int> div_value
    emits []
    given
        collections::map<str,int> totals
        str word
    asserts
        sample: call collections::empty_map<str,int>(), "a" => ok
    match call collections::get(totals, word)
        collections::key_absent => match call collections::insert(totals, word, 1)
            collections::key_exists => ok totals
            ok collections::map<str,int> next => ok next
        ok int previous => match call collections::replace(totals, word, previous / 2)
            collections::key_absent => ok totals
            ok collections::map<str,int> next => ok next
fn collections::map<str,int> swapped_ops
    emits []
    given
        collections::map<str,int> totals
        str word
    asserts
        sample: call collections::empty_map<str,int>(), "a" => ok
    match call collections::get(totals, word)
        collections::key_absent => match call collections::replace(totals, word, 1)
            collections::key_absent => ok totals
            ok collections::map<str,int> next => ok next
        ok int previous => match call collections::insert(totals, word, previous + 1)
            collections::key_exists => ok totals
            ok collections::map<str,int> next => ok next
fn void main
    emits []
    given
        str[] arguments
    asserts
        empty: [] => ok
    ok
`

func TestMapBatchNearMissDeclines(t *testing.T) {
	program := mapLeafProgram(t, map[string]string{"src/main.can": mapLeafRecoveryCorpus})
	assembly := mapLeafAssembly(t, program)
	proof := assembly.checkedMapBatches()
	leaves := assembly.checkedMapLeaves()
	for _, identity := range []string{"app::count_recover", "app::identity_map"} {
		fn := mapLeafFunction(t, program, identity)
		if _, ok := proof[fn.Identity()]; ok {
			t.Errorf("%s must not batch-qualify", identity)
		}
		if _, ok := leaves[fn.Identity()]; !ok {
			t.Errorf("%s expected leaf-only", identity)
		}
	}
	adversarial := mapLeafProgram(t, map[string]string{"src/main.can": mapBatchAdversarialCorpus})
	adversarialAssembly := mapLeafAssembly(t, adversarial)
	adversarialProof := adversarialAssembly.checkedMapBatches()
	adversarialLeaves := adversarialAssembly.checkedMapLeaves()
	for _, identity := range []string{"app::literal_key", "app::div_value", "app::swapped_ops"} {
		if _, ok := adversarialProof[mapLeafFunction(t, adversarial, identity).Identity()]; ok {
			t.Errorf("%s must not batch-qualify", identity)
		}
	}
	for _, identity := range []string{"app::literal_key", "app::swapped_ops"} {
		if _, ok := adversarialLeaves[mapLeafFunction(t, adversarial, identity).Identity()]; !ok {
			t.Errorf("%s expected leaf-only", identity)
		}
	}
	gallery := mapLeafAssembly(t, mapLeafGalleryProgram(t))
	for identity := range gallery.checkedMapBatches() {
		if _, ok := gallery.checkedMapLeaves()[identity]; !ok {
			t.Errorf("batch-qualified %s lacks a leaf proof", identity)
		}
	}
}

func TestMapBatchProvenBinding(t *testing.T) {
	entry := &MapBatchProof{Identity: "app::count_str", Target: "$canFunction0", Absent: "$canFunction0$BatchAbsent", Present: "$canFunction0$BatchPresent"}
	emitter := &RegionEmitter{mapBatches: map[string]*MapBatchProof{"app::count_str": entry, "app::nil": nil}}
	got, ok := emitter.provenMapBatch("app::count_str", "$canFunction0")
	if !ok || got != entry {
		t.Error("matching binding rejected")
	}
	for name, args := range map[string][2]string{
		"unknown": {"app::missing", "$canFunction0"},
		"rebound": {"app::count_str", "$canOther"},
		"nil":     {"app::nil", "$canFunction0"},
		"empty":   {"", ""},
	} {
		if got, ok := emitter.provenMapBatch(args[0], args[1]); ok || got != nil {
			t.Errorf("%s binding qualified", name)
		}
	}
	partial := &RegionEmitter{mapBatches: map[string]*MapBatchProof{"app::count_str": {Identity: "app::count_str", Target: "$canFunction0", Absent: "$canFunction0$BatchAbsent"}}}
	if got, ok := partial.provenMapBatch("app::count_str", "$canFunction0"); ok || got != nil {
		t.Error("partial companions qualified")
	}
	if got, ok := (&RegionEmitter{}).provenMapBatch("app::count_str", "$canFunction0"); ok || got != nil {
		t.Error("proof-free emitter qualified")
	}
}

func TestMapBatchCompanionText(t *testing.T) {
	program := mapLeafGalleryProgram(t)
	assembly := mapLeafAssembly(t, program)
	counter := mapLeafFunction(t, program, "gallery::count_one")
	entry := assembly.checkedMapBatches()[counter.Identity()]
	if entry == nil {
		t.Fatal("actual gallery count_one lacks a batch proof")
	}
	emitter := &RegionEmitter{Functions: assembly.functions, SourceID: counter.Symbol.Source.ID}
	absent, present, err := emitter.BatchCompanions(entry)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(absent, "function "+entry.Absent+"($w0: string): bigint {") {
		t.Errorf("absent head:\n%s", absent)
	}
	if !strings.HasPrefix(present, "function "+entry.Present+"($w0: string, $w1: bigint): bigint {") {
		t.Errorf("present head:\n%s", present)
	}
	for _, want := range []string{"= 1n;", "($canExpr1) + ($canExpr2)", "} catch ($canError) { throw $canCaught($canError, "} {
		if !strings.Contains(absent+present, want) {
			t.Errorf("companions lack %q", want)
		}
	}
	cold := fmt.Sprintf("{source:%s,start:%d,end:%d,invocation:[%s]}", quote(entry.Source), entry.GetSpan.Start, entry.GetSpan.End, quote(entry.Region.ID))
	if strings.Count(absent+present, cold) != 2 {
		t.Errorf("companions lack doubled cold get-site origin %s", cold)
	}
	for _, banned := range []string{"$canOrigin", "Promise<", "async ", "$canInvokeSync", "$canMapMethodWorker"} {
		if strings.Contains(absent+present, banned) {
			t.Errorf("companions contain %q", banned)
		}
	}
	if _, _, err := emitter.BatchCompanions(nil); err == nil {
		t.Error("nil proof emitted")
	}
	closed := *entry
	closed.Present = "9bad"
	if _, _, err := emitter.BatchCompanions(&closed); err == nil {
		t.Error("invalid companion binding emitted")
	}
	oneInput := *entry
	single := *entry.Region
	single.Inputs = single.Inputs[:1]
	oneInput.Region = &single
	if _, _, err := emitter.BatchCompanions(&oneInput); err == nil {
		t.Error("single-input region emitted")
	}
	badKind := *entry
	badKind.KeyKind = "float"
	if _, _, err := emitter.BatchCompanions(&badKind); err == nil {
		t.Error("unknown key kind emitted")
	}
	if emitter.mapBatches != nil {
		t.Error("companion emission kept batch state")
	}
}

func TestMapBatchDescriptorEmission(t *testing.T) {
	program := mapLeafGalleryProgram(t)
	assembly := mapLeafAssembly(t, program)
	counter := mapLeafFunction(t, program, "gallery::count_one")
	entry := assembly.checkedMapBatches()[counter.Identity()]
	if entry == nil {
		t.Fatal("actual gallery count_one lacks a batch proof")
	}
	artifacts, err := ProgramModules(program, "runtime", httpDependencies(t))
	if err != nil {
		t.Fatal(err)
	}
	var descriptors []string
	for _, line := range forwardingArtifactCallableLines(artifacts) {
		if strings.Contains(line, "$BatchAbsent") {
			descriptors = append(descriptors, line)
		}
	}
	if len(descriptors) != 1 {
		t.Fatalf("want exactly one batch descriptor, have %d", len(descriptors))
	}
	line := descriptors[0]
	for _, want := range []string{
		",$canContext,undefined,{companion:" + entry.Target + "$Leaf",
		"},{absent:" + entry.Absent + ",present:" + entry.Present,
		",keyKind:\"str\",origin:Object.freeze({source:" + quote(entry.Source),
		fmt.Sprintf(",start:%d,end:%d,invocation:[%s]}),factory:%s.get}", entry.GetSpan.Start, entry.GetSpan.End, quote(entry.Region.ID), entry.Receiver),
	} {
		if !strings.Contains(line, want) {
			t.Errorf("descriptor lacks %q:\n%s", want, line)
		}
	}
	var absent, present bool
	for _, artifact := range artifacts {
		if !strings.HasSuffix(artifact.Path, ".ts") {
			continue
		}
		text := string(artifact.Bytes)
		if strings.Contains(text, "export function "+entry.Absent+"($w0: string): bigint") {
			absent = true
		}
		if strings.Contains(text, "export function "+entry.Present+"($w0: string, $w1: bigint): bigint") {
			present = true
		}
	}
	if !absent || !present {
		t.Error("emitted modules lack exported batch companions")
	}
	recovery := mapLeafProgram(t, map[string]string{"src/main.can": mapBatchRecoveryUseCorpus})
	recoveryArtifacts, err := ProgramModules(recovery, "runtime", httpDependencies(t))
	if err != nil {
		t.Fatal(err)
	}
	var recoveryBatch, recoveryLeaf int
	for _, line := range forwardingArtifactCallableLines(recoveryArtifacts) {
		if strings.Contains(line, "$BatchAbsent") {
			recoveryBatch++
		}
		if strings.Contains(line, "$Leaf,keyKind:") {
			recoveryLeaf++
		}
	}
	if recoveryBatch != 0 {
		t.Error("standard-recovery use emitted a batch descriptor")
	}
	if recoveryLeaf != 1 {
		t.Errorf("standard-recovery use keeps %d leaf descriptors, want 1", recoveryLeaf)
	}
}

const mapBatchRecoveryUseCorpus = `package app
    provides []
    uses [collections]
fn collections::map<str,int> count_recover
    emits []
    given
        collections::map<str,int> totals
        str word
    asserts
        sample: call collections::empty_map<str,int>(), "a" => ok
    match call collections::get(totals, word)
        collections::key_absent => match call collections::insert(totals, word, 1)
            collections::key_exists => ok totals
            ok collections::map<str,int> next => ok next
        [_] as standard_failure snap => ok totals
        ok int previous => match call collections::replace(totals, word, previous + 1)
            collections::key_absent => ok totals
            ok collections::map<str,int> next => ok next
fn int frequency
    emits [collections::key_absent]
    given
        str[] words
        str word
    asserts
        sample: ["can", "code", "can"], "can" => ok 2
    collections::map<str,int> totals = call words.fold(call collections::empty_map<str,int>(), callable count_recover)
    match call collections::get(totals, word)
        collections::key_absent
        ok int found => ok found
fn void main
    emits []
    given
        str[] arguments
    asserts
        empty: [] => ok
    ok
`

func TestMapBatchEmittedMappingSpans(t *testing.T) {
	frequency, err := os.ReadFile(filepath.Join("..", "..", "..", "tools", "performance", "fixtures", "runtime", "src", "frequency.can"))
	if err != nil {
		t.Fatal(err)
	}
	program := mapLeafGalleryProgram(t)
	assembly := mapLeafAssembly(t, program)
	counter := mapLeafFunction(t, program, "gallery::count_one")
	entry := assembly.checkedMapBatches()[counter.Identity()]
	if entry == nil {
		t.Fatal("actual gallery count_one lacks a batch proof")
	}
	_, _, mappings := batchCompanionText(t, assembly, counter, entry)
	for _, m := range mappings {
		if m.Source != entry.Source {
			t.Fatalf("misplaced cross-module mapping: %+v", m)
		}
	}
	insertStart, insertEnd := leafFixtureCallSpan(t, string(frequency), "count_one", "collections::insert(totals, word, 1)")
	absentStart, absentEnd := insertEnd-2, insertEnd-1
	if string(frequency)[absentStart:absentEnd] != "1" || absentStart != insertStart+len("collections::insert(totals, word, ") {
		t.Fatalf("absent value span %d-%d misses the insert literal", absentStart, absentEnd)
	}
	valueStart, valueEnd := leafFixtureCallSpan(t, string(frequency), "count_one", "previous + 1")
	found := map[string]bool{}
	for _, m := range mappings {
		if m.Start == absentStart && m.End == absentEnd {
			found[m.Operation] = true
		}
		if m.Start == valueStart && m.End == valueEnd {
			found[m.Operation] = true
		}
	}
	for _, operation := range []string{"literal", "binary"} {
		if !found[operation] {
			t.Errorf("mappings lack %s value span", operation)
		}
	}
}

// batchSiteOrigin renders the exact frozen first call-site origin for an
// emitted batch descriptor.
func batchSiteOrigin(entry *MapBatchProof, start, end int) string {
	return fmt.Sprintf(`Object.freeze({source: %s, start: %d, end: %d, invocation: Object.freeze([%s])})`, quote(entry.Source), start, end, quote(entry.Region.ID))
}

func TestMapBatchEmittedRouteStr(t *testing.T) {
	bun := batchBun(t)
	frequency, err := os.ReadFile(filepath.Join("..", "..", "..", "tools", "performance", "fixtures", "runtime", "src", "frequency.can"))
	if err != nil {
		t.Fatal(err)
	}
	program := mapLeafGalleryProgram(t)
	assembly := mapLeafAssembly(t, program)
	counter := mapLeafFunction(t, program, "gallery::count_one")
	entry := assembly.checkedMapBatches()[counter.Identity()]
	if entry == nil {
		t.Fatal("actual gallery count_one lacks a batch proof")
	}
	absent, present, _ := batchCompanionText(t, assembly, counter, entry)
	getStart, getEnd := leafFixtureCallSpan(t, string(frequency), "count_one", "collections::get(totals, word)")
	runtimeRoot, _ := filepath.Abs("../../../runtime")
	code := leafHarnessHead(runtimeRoot) + absent + "\n" + present + "\n" +
		leafDomainSetup("map-str-int", "str") + `
const $seed = $canValue(await $methods.empty());
const $initial = $canValue(await $methods.insert($seed, "z", 9n));
const $origin = {source: "batch-route.can", start: 0, end: 1, invocation: ["app::run"]};
const $action = $canOwnCallable("p::batch#0", "app::count_one", [], async () => { throw new Error("slow adapter reached"); }, [], undefined, undefined, undefined, {absent: ` + entry.Absent + `, present: ` + entry.Present + `, keyKind: "str", origin: ` + batchSiteOrigin(entry, getStart, getEnd) + `, factory: $methods.get});
const $result = await $canFold($canArray(["a", "b", "a", "z"]), $initial, $action, {origin: $origin, site: "p::batch#0"});
const $entries = $canValue(await $methods.entries($canValue($result)));
assert.deepEqual($entries.map((e) => [e.key, e.value]), [["z", 10n], ["a", 2n], ["b", 1n]]);
const $snapshot = $canValue(await $methods.entries($initial));
assert.deepEqual($snapshot.map((e) => [e.key, e.value]), [["z", 9n]]);
const $empty = await $canFold($canArray([]), $initial, $action, {origin: $origin, site: "p::batch#0"});
assert.strictEqual($canValue($empty), $initial);
console.log("batch str route passed");
`
	leafRunScript(t, bun, "batch-route-str.ts", code, "batch str route passed")
}

func TestMapBatchEmittedRoutesIntBool(t *testing.T) {
	bun := batchBun(t)
	program := mapLeafProgram(t, map[string]string{"src/main.can": mapLeafCorpus})
	assembly := mapLeafAssembly(t, program)
	runtimeRoot, _ := filepath.Abs("../../../runtime")
	cases := []struct {
		suffix   string
		marker   string
		keyKind  string
		mapName  string
		keys     string
		preload  string
		expected string
		snapshot string
	}{
		{"app::count_int", "count_int", "int", "map-int-int", "[1n, 2n, 1n, 1n]", `["k", 7n, 9n]`, "[[7n, 9n], [1n, 3n], [2n, 1n]]", "[[7n, 9n]]"},
		{"app::count_bool", "count_bool", "bool", "map-bool-int", "[true, false, true, true]", `["k", false, 5n]`, "[[false, 6n], [true, 3n]]", "[[false, 5n]]"},
	}
	for _, tc := range cases {
		t.Run(tc.keyKind, func(t *testing.T) {
			fn := mapLeafFunction(t, program, tc.suffix)
			entry := assembly.checkedMapBatches()[fn.Identity()]
			if entry == nil {
				t.Fatalf("%s lacks a batch proof", tc.suffix)
			}
			absent, present, _ := batchCompanionText(t, assembly, fn, entry)
			getStart, getEnd := leafFixtureCallSpan(t, mapLeafCorpus, tc.marker, "collections::get(totals, word)")
			code := leafHarnessHead(runtimeRoot) + absent + "\n" + present + "\n" +
				leafDomainSetup(tc.mapName, tc.keyKind) + `
const $seed = $canValue(await $methods.empty());
const $preload = ` + tc.preload + `;
const $initial = $canValue(await $methods.insert($seed, $preload[1], $preload[2]));
const $origin = {source: "batch-route.can", start: 0, end: 1, invocation: ["app::run"]};
const $action = $canOwnCallable("p::batch#0", "` + tc.suffix + `", [], async () => { throw new Error("slow adapter reached"); }, [], undefined, undefined, undefined, {absent: ` + entry.Absent + `, present: ` + entry.Present + `, keyKind: "` + tc.keyKind + `", origin: ` + batchSiteOrigin(entry, getStart, getEnd) + `, factory: $methods.get});
const $result = await $canFold($canArray(` + tc.keys + `), $initial, $action, {origin: $origin, site: "p::batch#0"});
const $entries = $canValue(await $methods.entries($canValue($result)));
assert.deepEqual($entries.map((e) => [e.key, e.value]), ` + tc.expected + `);
const $snapshot = $canValue(await $methods.entries($initial));
assert.deepEqual($snapshot.map((e) => [e.key, e.value]), ` + tc.snapshot + `);
console.log("batch ` + tc.keyKind + ` route passed");
`
			leafRunScript(t, bun, "batch-route-"+tc.keyKind+".ts", code, "batch "+tc.keyKind+" route passed")
		})
	}
}

func TestMapBatchEmittedColdBoundary(t *testing.T) {
	bun := batchBun(t)
	frequency, err := os.ReadFile(filepath.Join("..", "..", "..", "tools", "performance", "fixtures", "runtime", "src", "frequency.can"))
	if err != nil {
		t.Fatal(err)
	}
	program := mapLeafGalleryProgram(t)
	assembly := mapLeafAssembly(t, program)
	counter := mapLeafFunction(t, program, "gallery::count_one")
	entry := assembly.checkedMapBatches()[counter.Identity()]
	if entry == nil {
		t.Fatal("actual gallery count_one lacks a batch proof")
	}
	_, present, _ := batchCompanionText(t, assembly, counter, entry)
	getStart, getEnd := leafFixtureCallSpan(t, string(frequency), "count_one", "collections::get(totals, word)")
	getSite := leafCallSiteOrigin(entry.Source, getStart, getEnd, entry.Region.ID)
	runtimeRoot, _ := filepath.Abs("../../../runtime")
	code := leafHarnessHead(runtimeRoot) + present + "\n" +
		leafDomainSetup("map-str-int", "str") + `
const $seed = $canValue(await $methods.empty());
const $initial = $canValue(await $methods.insert($seed, "z", 9n));
const $origin = {source: "batch-route.can", start: 0, end: 1, invocation: ["app::run"]};
// HOST INJECTION (outside valid Can inputs): a throwing absent where the
// emitted pure companion cannot fail.
const $action = $canOwnCallable("p::batch#0", "app::count_one", [], async () => { throw new Error("slow adapter reached"); }, [], undefined, undefined, undefined, {absent: ((_key) => { throw new Error("injected batch fault"); }), present: ` + entry.Present + `, keyKind: "str", origin: ` + batchSiteOrigin(entry, getStart, getEnd) + `, factory: $methods.get});
const $getSite = ` + getSite + `;
const $r1 = await $canFold($canArray(["a", "b"]), $initial, $action, {origin: $origin, site: "p::batch#0"});
const $r2 = await $canFold($canArray(["a", "b"]), $initial, $action, {origin: $origin, site: "p::batch#0"});
for (const $r of [$r1, $r2]) {
  assert.equal($r.kind, "standard");
  const $d = $canStandardDiagnostics($r.value);
  assert.equal($d.kind, "native_exception");
  assert.deepEqual($d.origin, $getSite);
  assert.equal($d.boundaryOrigin, undefined);
}
const $o1 = $canStandardOccurrence($r1.value);
const $o2 = $canStandardOccurrence($r2.value);
assert($o1 !== $o2);
const $snapshot = $canValue(await $methods.entries($initial));
assert.deepEqual($snapshot.map((e) => [e.key, e.value]), [["z", 9n]]);
console.log("batch cold boundary passed");
`
	leafRunScript(t, bun, "batch-route-cold.ts", code, "batch cold boundary passed")
}

func TestMapBatchEmittedFallback(t *testing.T) {
	bun := batchBun(t)
	frequency, err := os.ReadFile(filepath.Join("..", "..", "..", "tools", "performance", "fixtures", "runtime", "src", "frequency.can"))
	if err != nil {
		t.Fatal(err)
	}
	program := mapLeafGalleryProgram(t)
	assembly := mapLeafAssembly(t, program)
	counter := mapLeafFunction(t, program, "gallery::count_one")
	entry := assembly.checkedMapBatches()[counter.Identity()]
	if entry == nil {
		t.Fatal("actual gallery count_one lacks a batch proof")
	}
	leafEntry := assembly.checkedMapLeaves()[counter.Identity()]
	if leafEntry == nil {
		t.Fatal("actual gallery count_one lacks a map-leaf proof")
	}
	absent, present, _ := batchCompanionText(t, assembly, counter, entry)
	leafClean, _, err := extractMappings(func() string {
		emitter := &RegionEmitter{Functions: assembly.functions, SourceID: counter.Symbol.Source.ID}
		raw, err := emitter.LeafCompanion(leafEntry)
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}())
	if err != nil {
		t.Fatal(err)
	}
	methods := leafWorkerKinds(t, leafClean, 3, []string{"get", "insert", "replace"})
	getStart, getEnd := leafFixtureCallSpan(t, string(frequency), "count_one", "collections::get(totals, word)")
	getSite := leafCallSiteOrigin(entry.Source, getStart, getEnd, entry.Region.ID)
	runtimeRoot, _ := filepath.Abs("../../../runtime")
	code := leafHarnessHead(runtimeRoot) + leafClean + "\n" + absent + "\n" + present + "\n" +
		leafDomainSetup("map-str-int", "str") +
		leafReceiverDecls(methods, nil) + `
const $origin = {source: "batch-route.can", start: 0, end: 1, invocation: ["app::run"]};
let $absentCalls = 0, $presentCalls = 0;
const $absentRun = ($key) => { $absentCalls++; return ` + entry.Absent + `($key); };
const $presentRun = ($key, $previous) => { $presentCalls++; return ` + entry.Present + `($key, $previous); };
const $action = $canOwnCallable("p::batch#0", "app::count_one", [], async () => { throw new Error("slow adapter reached"); }, [], undefined, undefined, {companion: ` + leafEntry.Companion + `, keyKind: "str", origin: Object.freeze({source: "batch-route.can", start: 0, end: 1, invocation: Object.freeze(["app::count_one"])})}, {absent: $absentRun, present: $presentRun, keyKind: "str", origin: ` + batchSiteOrigin(entry, getStart, getEnd) + `, factory: $methods.get});
const $getSite = ` + getSite + `;
// HOST INJECTION (outside valid Can inputs): fold initial from a foreign factory.
const $foreign = $canCreateMap($domain, {map: "map-str-int-foreign", entry: "entry-test", absent: $shapes[0].identity, exists: $shapes[1].identity}, "str");
const $hostile = $canValue(await $foreign.empty());
const $r1 = await $canFold($canArray(["a", "b"]), $hostile, $action, {origin: $origin, site: "p::batch#0"});
const $r2 = await $canFold($canArray(["a", "b"]), $hostile, $action, {origin: $origin, site: "p::batch#0"});
for (const $r of [$r1, $r2]) {
  assert.equal($r.kind, "standard");
  const $d = $canStandardDiagnostics($r.value);
  assert.equal($d.message, "resource_state: invalid resource use");
  assert.deepEqual($d.origin, {source: "can:collections:map", start: 0, end: 0, invocation: []});
  assert.deepEqual($d.boundaryOrigin, $getSite);
}
assert($absentCalls === 0 && $presentCalls === 0);
const $o1 = $canStandardOccurrence($r1.value);
const $o2 = $canStandardOccurrence($r2.value);
assert($o1 !== $o2);
console.log("batch emitted fallback passed");
`
	leafRunScript(t, bun, "batch-route-fallback.ts", code, "batch emitted fallback passed")
}
