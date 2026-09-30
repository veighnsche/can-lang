package emit

import (
	"context"
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
)

func mapLeafGalleryProgram(t *testing.T) *check.Program {
	t.Helper()
	path := filepath.Join("..", "..", "..", "tools", "performance", "fixtures", "runtime", "src", "frequency.can")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	mainCan, err := os.ReadFile(filepath.Join("..", "..", "..", "tools", "performance", "fixtures", "runtime", "src", "main.can"))
	if err != nil {
		t.Fatal(err)
	}
	return mapLeafProgram(t, map[string]string{
		"src/frequency.can": string(data),
		"src/main.can":      string(mainCan),
	})
}

func mapLeafProgram(t *testing.T, files map[string]string) *check.Program {
	t.Helper()
	return actionEmitProgram(t, files)
}

func mapLeafAssembly(t *testing.T, program *check.Program) *programAssembly {
	t.Helper()
	assembly, err := assembleProgramBindings(program)
	if err != nil {
		t.Fatal(err)
	}
	return assembly
}

func mapLeafFunction(t *testing.T, program *check.Program, suffix string) *check.ProgramFunction {
	t.Helper()
	var found *check.ProgramFunction
	for _, fn := range program.Functions {
		if fn == nil || fn.Symbol == nil {
			continue
		}
		if strings.HasSuffix(fn.Identity(), suffix) {
			if found != nil {
				t.Fatalf("multiple functions match %q", suffix)
			}
			found = fn
		}
	}
	if found == nil {
		t.Fatalf("no function %s", suffix)
	}
	return found
}

func TestMapLeafGalleryCallbackEligible(t *testing.T) {
	path := filepath.Join("..", "..", "..", "tools", "performance", "fixtures", "runtime", "src", "frequency.can")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	mainCan, err := os.ReadFile(filepath.Join("..", "..", "..", "tools", "performance", "fixtures", "runtime", "src", "main.can"))
	if err != nil {
		t.Fatal(err)
	}
	program := mapLeafProgram(t, map[string]string{
		"src/frequency.can": string(data),
		"src/main.can":      string(mainCan),
	})
	assembly := mapLeafAssembly(t, program)
	proof := assembly.checkedMapLeaves()
	counter := mapLeafFunction(t, program, "gallery::count_one")
	entry, ok := proof[counter.Identity()]
	if !ok {
		t.Fatal("actual gallery count_one lacks a map-leaf proof")
	}
	if entry.Target == "" || entry.Companion != entry.Target+MapLeafCompanionSuffix {
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
	if len(entry.Calls) != 3 {
		t.Fatalf("want get/insert/replace call targets, have %v", entry.Calls)
	}
	receiver := ""
	for _, target := range entry.Calls {
		method := target[strings.LastIndex(target, ".")+1:]
		if method != "get" && method != "insert" && method != "replace" {
			t.Errorf("unexpected leaf call target %s", target)
		}
		head := target[:strings.LastIndex(target, ".")]
		if receiver == "" {
			receiver = head
		} else if receiver != head {
			t.Errorf("leaf calls span receivers %s and %s", receiver, head)
		}
	}
	if _, ok := proof[mapLeafFunction(t, program, "gallery::frequency").Identity()]; ok {
		t.Error("gallery frequency must not qualify as a leaf")
	}
}

const mapLeafCorpus = `package app
    provides []
    uses [collections]
fn collections::map<str,int> count_str
    emits {}
    given
        collections::map<str,int> totals
        str word
    asserts
        sample: call collections::empty_map<str,int>(), "a" => ok
    match call collections::get(totals, word)
        collections::key_absent => match call collections::insert(totals, word, 1)
            collections::key_exists => ok totals
            ok collections::map<str,int> next => ok next
        ok int previous => match call collections::replace(totals, word, previous + 1)
            collections::key_absent => ok totals
            ok collections::map<str,int> next => ok next
fn collections::map<int,int> count_int
    emits {}
    given
        collections::map<int,int> totals
        int word
    asserts
        sample: call collections::empty_map<int,int>(), 1 => ok
    match call collections::get(totals, word)
        collections::key_absent => match call collections::insert(totals, word, 1)
            collections::key_exists => ok totals
            ok collections::map<int,int> next => ok next
        ok int previous => match call collections::replace(totals, word, previous + 1)
            collections::key_absent => ok totals
            ok collections::map<int,int> next => ok next
fn collections::map<bool,int> count_bool
    emits {}
    given
        collections::map<bool,int> totals
        bool word
    asserts
        sample: call collections::empty_map<bool,int>(), true => ok
    match call collections::get(totals, word)
        collections::key_absent => match call collections::insert(totals, word, 1)
            collections::key_exists => ok totals
            ok collections::map<bool,int> next => ok next
        ok int previous => match call collections::replace(totals, word, previous + 1)
            collections::key_absent => ok totals
            ok collections::map<bool,int> next => ok next
fn collections::map<str,int> uses_remove
    emits {}
    given
        collections::map<str,int> totals
        str word
    asserts
        sample: call collections::empty_map<str,int>(), "a" => ok
    match call collections::remove(totals, word)
        collections::key_absent => ok totals
        ok collections::map<str,int> next => ok next
fn collections::map<str,str> wrong_value
    emits {}
    given
        collections::map<str,str> totals
        str word
    asserts
        sample: call collections::empty_map<str,str>(), "a" => ok
    match call collections::get(totals, word)
        collections::key_absent => ok totals
        ok str previous => ok totals
fn collections::map<str,int> has_statement
    emits {}
    given
        collections::map<str,int> totals
        str word
    asserts
        sample: call collections::empty_map<str,int>(), "a" => ok
    collections::map<str,int> held = totals
    match call collections::get(held, word)
        collections::key_absent => ok held
        ok int previous => ok held
fn int helper
    emits {}
    given
        int value
    asserts
        sample: 1 => ok 1
    ok value
fn collections::map<str,int> authored_call
    emits {}
    given
        collections::map<str,int> totals
        str word
    asserts
        sample: call collections::empty_map<str,int>(), "a" => ok
    match call collections::get(totals, word)
        collections::key_absent => ok totals
        ok int previous => match call collections::replace(totals, word, call helper(previous))
            collections::key_absent => ok totals
            ok collections::map<str,int> next => ok next
fn void main
    emits {}
    given
        str[] arguments
    asserts
        empty: [] => ok
    ok
`

func TestMapLeafProofAdmitsOnlyClosedLeaves(t *testing.T) {
	program := mapLeafProgram(t, map[string]string{"src/main.can": mapLeafCorpus})
	assembly := mapLeafAssembly(t, program)
	proof := assembly.checkedMapLeaves()
	for identity, keyKind := range map[string]string{"app::count_str": "str", "app::count_int": "int", "app::count_bool": "bool"} {
		entry, ok := proof[mapLeafFunction(t, program, identity).Identity()]
		if !ok {
			t.Errorf("%s lacks a map-leaf proof", identity)
			continue
		}
		if entry.KeyKind != keyKind || len(entry.Calls) != 3 {
			t.Errorf("%s proof wrong: %+v", identity, entry)
		}
	}
	for _, identity := range []string{"app::uses_remove", "app::wrong_value", "app::has_statement", "app::authored_call", "app::helper", "app::main"} {
		if _, ok := proof[mapLeafFunction(t, program, identity).Identity()]; ok {
			t.Errorf("%s must not qualify as a leaf", identity)
		}
	}
}

func TestMapLeafProvenBinding(t *testing.T) {
	entry := &MapLeafProof{Identity: "app::count_str", Target: "$canFunction0", Companion: "$canFunction0$Leaf"}
	emitter := &RegionEmitter{mapLeaves: map[string]*MapLeafProof{"app::count_str": entry, "app::nil": nil}}
	if companion, ok := emitter.provenMapLeaf("app::count_str", "$canFunction0"); !ok || companion != "$canFunction0$Leaf" {
		t.Errorf("matching binding rejected: %q %v", companion, ok)
	}
	for name, args := range map[string][2]string{
		"unknown identity": {"app::missing", "$canFunction0"},
		"nil entry":        {"app::nil", "$canFunction0"},
		"rebound target":   {"app::count_str", "$reboundTarget"},
		"empty identity":   {"", "$canFunction0"},
		"empty target":     {"app::count_str", ""},
	} {
		if companion, ok := emitter.provenMapLeaf(args[0], args[1]); ok || companion != "" {
			t.Errorf("%s qualified: %q", name, companion)
		}
	}
	empty := &RegionEmitter{mapLeaves: map[string]*MapLeafProof{"app::count_str": {Identity: "app::count_str", Target: "$canFunction0"}}}
	if companion, ok := empty.provenMapLeaf("app::count_str", "$canFunction0"); ok || companion != "" {
		t.Error("empty companion qualified")
	}
	if companion, ok := (&RegionEmitter{}).provenMapLeaf("app::count_str", "$canFunction0"); ok || companion != "" {
		t.Error("nil proof map qualified")
	}
}

func TestMapLeafCompanionEmission(t *testing.T) {
	path := filepath.Join("..", "..", "..", "tools", "performance", "fixtures", "runtime", "src", "frequency.can")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	mainCan, err := os.ReadFile(filepath.Join("..", "..", "..", "tools", "performance", "fixtures", "runtime", "src", "main.can"))
	if err != nil {
		t.Fatal(err)
	}
	program := mapLeafProgram(t, map[string]string{
		"src/frequency.can": string(data),
		"src/main.can":      string(mainCan),
	})
	assembly := mapLeafAssembly(t, program)
	proof := assembly.checkedMapLeaves()
	counter := mapLeafFunction(t, program, "gallery::count_one")
	entry := proof[counter.Identity()]
	if entry == nil {
		t.Fatal("actual gallery count_one lacks a map-leaf proof")
	}
	emitter := &RegionEmitter{Functions: assembly.functions, SourceID: counter.Symbol.Source.ID}
	companion, err := emitter.LeafCompanion(entry)
	if err != nil {
		t.Fatal(err)
	}
	head := "function " + entry.Companion + "($w0: "
	if !strings.HasPrefix(companion, head) {
		t.Errorf("companion head:\n%s", companion)
	}
	for _, want := range []string{", $w1: ", "): $canCompletion<", "$canInvokeSync(", "$canMapMethodWorker(", "} catch ($canCause) { return $canCaught($canCause, "} {
		if !strings.Contains(companion, want) {
			t.Errorf("companion lacks %q:\n%s", want, companion)
		}
	}
	if strings.Contains(companion, "Promise<") || strings.Contains(companion, "async ") {
		t.Errorf("companion must stay synchronous:\n%s", companion)
	}
	if emitter.leaf != nil {
		t.Error("calling emitter kept leaf state")
	}
	closed := *entry
	closed.Companion = "9bad"
	if _, err := emitter.LeafCompanion(&closed); err == nil {
		t.Error("invalid companion binding emitted")
	}
	if _, err := emitter.LeafCompanion(nil); err == nil {
		t.Error("nil proof emitted")
	}
	nilRegion := *entry
	nilRegion.Region = nil
	if _, err := emitter.LeafCompanion(&nilRegion); err == nil {
		t.Error("nil region emitted")
	}
	single := *counter.Region
	single.Inputs = single.Inputs[:1]
	oneInput := *entry
	oneInput.Region = &single
	if _, err := emitter.LeafCompanion(&oneInput); err == nil {
		t.Error("single-input region emitted")
	}
	if got := strings.Count(companion, "$canInvokeSync("); got != 3 {
		t.Errorf("want get/insert/replace sync boundaries, have %d", got)
	}
	if got := strings.Count(companion, ")!("); got != 3 {
		t.Errorf("want 3 strict-safe worker assertions, have %d", got)
	}
}

func TestMapLeafDescriptorEmission(t *testing.T) {
	program := mapLeafGalleryProgram(t)
	assembly := mapLeafAssembly(t, program)
	proof := assembly.checkedMapLeaves()
	counter := mapLeafFunction(t, program, "gallery::count_one")
	entry := proof[counter.Identity()]
	if entry == nil {
		t.Fatal("actual gallery count_one lacks a map-leaf proof")
	}
	artifacts, err := ProgramModules(program, "runtime", httpDependencies(t))
	if err != nil {
		t.Fatal(err)
	}
	var descriptors []string
	for _, line := range forwardingArtifactCallableLines(artifacts) {
		if strings.Contains(line, "$Leaf,keyKind:") {
			descriptors = append(descriptors, line)
		}
	}
	if len(descriptors) != 1 {
		t.Fatalf("want exactly one leaf descriptor, have %d:\n%s", len(descriptors), strings.Join(descriptors, "\n"))
	}
	line := descriptors[0]
	for _, want := range []string{
		",$canContext,undefined,{companion:" + entry.Companion,
		",keyKind:\"str\",origin:Object.freeze({source:" + quote(entry.Source),
		",invocation:[" + quote(entry.Region.ID) + "]})}",
	} {
		if !strings.Contains(line, want) {
			t.Errorf("descriptor lacks %q:\n%s", want, line)
		}
	}
	var companion, invokeImport, workerImport bool
	for _, artifact := range artifacts {
		if !strings.HasSuffix(artifact.Path, ".ts") {
			continue
		}
		text := string(artifact.Bytes)
		if strings.Contains(text, "function "+entry.Companion+"($w0: ") {
			companion = true
		}
		for _, artifactLine := range strings.Split(text, "\n") {
			if strings.Contains(artifactLine, "import ") && strings.Contains(artifactLine, "$canInvokeSync") {
				invokeImport = true
			}
			if strings.Contains(artifactLine, "import ") && strings.Contains(artifactLine, "$canMapMethodWorker") {
				workerImport = true
			}
		}
	}
	if !companion {
		t.Errorf("emitted modules lack exported companion %s", entry.Companion)
	}
	if !invokeImport || !workerImport {
		t.Error("emitted modules lack companion sync/worker imports")
	}
}

// leafWorkerCall splits one emitted $canMapMethodWorker(receiver)!(args)
// call into its receiver text and top-level argument count with the third
// argument text. The strict-safe assertion is required. Plain identifiers
// classify by call shape; member roots classify by member name.
func leafWorkerCall(t *testing.T, text string) (receiver string, args int, third string) {
	t.Helper()
	const head = "$canMapMethodWorker("
	start := strings.Index(text, head)
	if start < 0 {
		t.Fatalf("missing leaf worker call:\n%s", text)
	}
	depth := 0
	end := -1
	for i := start + len(head); i < len(text); i++ {
		switch text[i] {
		case '(':
			depth++
		case ')':
			if depth == 0 {
				end = i
			} else {
				depth--
			}
		}
		if end >= 0 {
			break
		}
	}
	if end < 0 {
		t.Fatalf("unterminated leaf receiver:\n%s", text)
	}
	receiver = text[start+len(head) : end]
	rest := text[end+1:]
	if !strings.HasPrefix(rest, "!(") {
		t.Fatalf("leaf worker call lacks strict assertion and argument list:\n%s", text)
	}
	rest = rest[1:]
	depth = 0
	var current strings.Builder
	var parts []string
	for i := 1; i < len(rest); i++ {
		switch rest[i] {
		case '(', '[', '{':
			depth++
			current.WriteByte(rest[i])
		case ')', ']', '}':
			if depth == 0 {
				parts = append(parts, current.String())
				i = len(rest)
			} else {
				depth--
				current.WriteByte(rest[i])
			}
		case ',':
			if depth == 0 {
				parts = append(parts, current.String())
				current.Reset()
			} else {
				current.WriteByte(rest[i])
			}
		default:
			current.WriteByte(rest[i])
		}
	}
	if len(parts) > 2 {
		third = strings.TrimSpace(parts[2])
	}
	return receiver, len(parts), third
}

// leafCompanionText emits one proven leaf companion and returns its
// mapping-free TypeScript plus extracted source mappings. The harness
// links the actual checkout runtime by direct absolute imports; no
// runtime sources are copied and no shared helper is touched.
func leafCompanionText(t *testing.T, assembly *programAssembly, fn *check.ProgramFunction, entry *MapLeafProof) (string, []ir.Mapping) {
	t.Helper()
	emitter := &RegionEmitter{Functions: assembly.functions, SourceID: fn.Symbol.Source.ID}
	raw, err := emitter.LeafCompanion(entry)
	if err != nil {
		t.Fatal(err)
	}
	clean, mappings, err := extractMappings(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(mappings) == 0 {
		t.Fatal("emitted companion carries no source mappings")
	}
	return clean, mappings
}

// leafWorkerKinds classifies each emitted worker call by member name or
// call shape, asserting the expected count and positional order.
func leafWorkerKinds(t *testing.T, clean string, want int, order []string) map[string]string {
	t.Helper()
	if got := strings.Count(clean, "$canMapMethodWorker("); got != want {
		t.Fatalf("want %d emitted worker calls, have %d", want, got)
	}
	methods := map[string]string{}
	kinds := []string{}
	rest := clean
	for i := 0; i < want; i++ {
		at := strings.Index(rest, "$canMapMethodWorker(")
		receiver, args, third := leafWorkerCall(t, rest[at:])
		kind := ""
		if dot := strings.LastIndex(receiver, "."); dot >= 0 {
			kind = receiver[dot+1:]
		} else {
			switch {
			case args == 2:
				kind = "get"
			case args == 3 && third == "1n":
				kind = "insert"
			case args == 3:
				kind = "replace"
			}
		}
		if kind != "get" && kind != "insert" && kind != "replace" {
			t.Fatalf("unclassified worker call %q args=%d third=%q", receiver, args, third)
		}
		kinds = append(kinds, kind)
		if prev, ok := methods[receiver]; ok && prev != kind {
			t.Fatalf("receiver %q names %s and %s", receiver, prev, kind)
		}
		methods[receiver] = kind
		rest = rest[at+len("$canMapMethodWorker("):]
	}
	if len(order) != len(kinds) {
		t.Fatalf("worker order %v, want %v", kinds, order)
	}
	for i, want := range order {
		if kinds[i] != want {
			t.Fatalf("worker order %v, want %v", kinds, order)
		}
	}
	return methods
}

// leafReceiverDecls binds each distinct worker receiver root to the real
// factory methods object, or to a caller-supplied host-injection
// replacement keyed by root identifier.
func leafReceiverDecls(methods map[string]string, overrides map[string]string) string {
	roots := map[string]bool{}
	ordered := []string{}
	for receiver, kind := range methods {
		if dot := strings.Index(receiver, "."); dot >= 0 {
			root := receiver[:dot]
			if !roots[root] {
				roots[root] = true
				rhs := "$methods"
				if custom, ok := overrides[root]; ok {
					rhs = custom
				}
				ordered = append(ordered, "const "+root+" = "+rhs+";")
			}
			continue
		}
		rhs := "$methods." + kind
		if custom, ok := overrides[receiver]; ok {
			rhs = custom
		}
		ordered = append(ordered, "const "+receiver+" = "+rhs+";")
	}
	sort.Strings(ordered)
	return strings.Join(ordered, "\n") + "\n"
}

// leafHarnessHead imports the real checkout runtime modules used by
// emitted companions and route scripts, including standard failure
// diagnostics for cold-boundary assertions.
func leafHarnessHead(runtimeRoot string) string {
	quoteSlash := func(path string) string { return quote(filepath.ToSlash(path)) }
	return `import {strict as assert} from "node:assert";` + "\n" +
		`import {createHash} from "node:crypto";` + "\n" +
		CompletionImports(filepath.Join(runtimeRoot, "completion.ts")) +
		PatternImports(filepath.Join(runtimeRoot, "data.ts"), filepath.Join(runtimeRoot, "failure.ts")) +
		fmt.Sprintf("import {invokeSync as $canInvokeSync} from %s;\n", quoteSlash(filepath.Join(runtimeRoot, "completion.ts"))) +
		fmt.Sprintf("import {ownCallable as $canOwnCallable} from %s;\n", quoteSlash(filepath.Join(runtimeRoot, "callable.ts"))) +
		fmt.Sprintf("import {array as $canArray} from %s;\n", quoteSlash(filepath.Join(runtimeRoot, "data.ts"))) +
		fmt.Sprintf("import {fold as $canFold} from %s;\n", quoteSlash(filepath.Join(runtimeRoot, "collections/array.ts"))) +
		fmt.Sprintf("import {createMap as $canCreateMap, mapMethodWorker as $canMapMethodWorker} from %s;\n", quoteSlash(filepath.Join(runtimeRoot, "collections/map.ts"))) +
		fmt.Sprintf("import {createDomainRuntime as $canCreateDomainRuntime} from %s;\n", quoteSlash(filepath.Join(runtimeRoot, "domain.ts"))) +
		fmt.Sprintf("import {catalogue as $catalogue} from %s;\n", quoteSlash(filepath.Join(runtimeRoot, "catalogue.ts"))) +
		fmt.Sprintf("import {standardFailureDiagnostics as $canStandardDiagnostics, standardFailureOccurrenceID as $canStandardOccurrence} from %s;\n", quoteSlash(filepath.Join(runtimeRoot, "failure.ts")))
}

// leafDomainSetup builds a real domain runtime plus one real map factory
// of the requested key kind; absent/exists identities follow catalogue order.
func leafDomainSetup(mapName, keyKind string) string {
	return `
const $declarations = $catalogue.errors.filter((e) => ["collections::key_absent", "collections::key_exists"].includes(e.name));
const $shapes = $declarations.map((e) => ({identity: createHash("sha256").update("can-concrete-type-v1\0" + JSON.stringify(["error", e.identity])).digest("hex"), kind: "error", declaration: e.identity, arguments: [], fields: [], leaves: [], inputs: [], errors: []}));
const $domain = $canCreateDomainRuntime({declarations: $declarations.map((e) => ({identity: e.identity, name: e.name, parameters: 0})), shapes: $shapes});
const $methods = $canCreateMap($domain, {map: "` + mapName + `", entry: "entry-test", absent: $shapes[0].identity, exists: $shapes[1].identity}, "` + keyKind + `");
`
}

// leafRunScript executes one assembled route script with the frozen Bun
// flags and requires its sentinel on stdout.
func leafRunScript(t *testing.T, bun, name, code, sentinel string) {
	t.Helper()
	file := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(file, []byte(code), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, bun, "--no-install", "--no-env-file", file).CombinedOutput()
	if err != nil || !strings.Contains(string(out), sentinel) {
		t.Fatalf("%v\n%s", err, out)
	}
}

func TestMapLeafEmittedRouteExecutes(t *testing.T) {
	bun := os.Getenv("CAN_BUN")
	if bun == "" {
		t.Skip("set CAN_BUN for leaf route execution")
	}
	program := mapLeafGalleryProgram(t)
	assembly := mapLeafAssembly(t, program)
	proof := assembly.checkedMapLeaves()
	counter := mapLeafFunction(t, program, "gallery::count_one")
	entry := proof[counter.Identity()]
	if entry == nil {
		t.Fatal("actual gallery count_one lacks a map-leaf proof")
	}
	clean, _ := leafCompanionText(t, assembly, counter, entry)
	t.Logf("companion:\n%s", clean)
	methods := leafWorkerKinds(t, clean, 3, []string{"get", "insert", "replace"})
	runtimeRoot, _ := filepath.Abs("../../../runtime")
	code := leafHarnessHead(runtimeRoot) + clean + "\n" +
		leafDomainSetup("map-str-int", "str") +
		leafReceiverDecls(methods, nil) + `
const $seed = $canValue(await $methods.empty());
const $initial = $canValue(await $methods.insert($seed, "z", 9n));
const $origin = {source: "leaf-route.can", start: 0, end: 1, invocation: ["app::run"]};
const $action = $canOwnCallable("p::leaf#0", "app::count_one", [], async () => { throw new Error("slow adapter reached"); }, [], undefined, undefined, {companion: ` + entry.Companion + `, keyKind: "str", origin: Object.freeze({source: "leaf-route.can", start: 0, end: 1, invocation: Object.freeze(["app::count_one"])})});
const $result = await $canFold($canArray(["a", "b", "a", "a"]), $initial, $action, {origin: $origin, site: "p::leaf#0"});
const $entries = $canValue(await $methods.entries($canValue($result)));
assert.deepEqual($entries.map((e) => [e.key, e.value]), [["z", 9n], ["a", 3n], ["b", 1n]]);
const $snapshot = $canValue(await $methods.entries($initial));
assert.deepEqual($snapshot.map((e) => [e.key, e.value]), [["z", 9n]]);
console.log("leaf route executable passed");
`
	leafRunScript(t, bun, "leaf-route.ts", code, "leaf route executable passed")
}

const mapLeafRecoveryCorpus = `package app
    provides []
    uses [collections]
fn collections::map<str,int> count_recover
    emits {}
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
fn collections::map<str,int> identity_map
    emits {}
    given
        collections::map<str,int> totals
        str word
    asserts
        sample: call collections::empty_map<str,int>(), "a" => ok
    ok totals
fn void main
    emits {}
    given
        str[] arguments
    asserts
        empty: [] => ok
    ok
`

func TestMapLeafProofAdmitsRecoveryAndIdentity(t *testing.T) {
	program := mapLeafProgram(t, map[string]string{"src/main.can": mapLeafRecoveryCorpus})
	assembly := mapLeafAssembly(t, program)
	proof := assembly.checkedMapLeaves()
	recover := mapLeafFunction(t, program, "app::count_recover")
	entry, ok := proof[recover.Identity()]
	if !ok {
		t.Fatal("standard-arm recovery leaf lacks a map-leaf proof")
	}
	if entry.KeyKind != "str" || len(entry.Calls) != 3 {
		t.Errorf("recovery proof wrong: %+v", entry)
	}
	identity := mapLeafFunction(t, program, "app::identity_map")
	identical, ok := proof[identity.Identity()]
	if !ok {
		t.Fatal("call-free identity leaf lacks a map-leaf proof")
	}
	if identical.KeyKind != "str" || len(identical.Calls) != 0 {
		t.Errorf("identity proof wrong: %+v", identical)
	}
}

func TestMapLeafEmittedRoutesAllKeys(t *testing.T) {
	bun := os.Getenv("CAN_BUN")
	if bun == "" {
		t.Skip("set CAN_BUN for all-key leaf route execution")
	}
	program := mapLeafProgram(t, map[string]string{"src/main.can": mapLeafCorpus})
	assembly := mapLeafAssembly(t, program)
	proof := assembly.checkedMapLeaves()
	runtimeRoot, _ := filepath.Abs("../../../runtime")
	cases := []struct {
		suffix   string
		keyKind  string
		mapName  string
		keys     string
		preload  string
		expected string
		snapshot string
	}{
		{"app::count_int", "int", "map-int-int", "[1n, 2n, 1n, 1n]", `["k", 7n, 9n]`, "[[7n, 9n], [1n, 3n], [2n, 1n]]", "[[7n, 9n]]"},
		{"app::count_bool", "bool", "map-bool-int", "[true, false, true, true]", `["k", false, 5n]`, "[[false, 6n], [true, 3n]]", "[[false, 5n]]"},
	}
	for _, tc := range cases {
		t.Run(tc.keyKind, func(t *testing.T) {
			fn := mapLeafFunction(t, program, tc.suffix)
			entry := proof[fn.Identity()]
			if entry == nil {
				t.Fatalf("%s lacks a map-leaf proof", tc.suffix)
			}
			clean, _ := leafCompanionText(t, assembly, fn, entry)
			methods := leafWorkerKinds(t, clean, 3, []string{"get", "insert", "replace"})
			code := leafHarnessHead(runtimeRoot) + clean + "\n" +
				leafDomainSetup(tc.mapName, tc.keyKind) +
				leafReceiverDecls(methods, nil) + `
const $seed = $canValue(await $methods.empty());
const $preload = ` + tc.preload + `;
const $initial = $canValue(await $methods.insert($seed, $preload[1], $preload[2]));
const $origin = {source: "leaf-route.can", start: 0, end: 1, invocation: ["app::run"]};
const $action = $canOwnCallable("p::leaf#0", "` + tc.suffix + `", [], async () => { throw new Error("slow adapter reached"); }, [], undefined, undefined, {companion: ` + entry.Companion + `, keyKind: "` + tc.keyKind + `", origin: Object.freeze({source: "leaf-route.can", start: 0, end: 1, invocation: Object.freeze(["` + tc.suffix + `"])})});
const $result = await $canFold($canArray(` + tc.keys + `), $initial, $action, {origin: $origin, site: "p::leaf#0"});
const $entries = $canValue(await $methods.entries($canValue($result)));
assert.deepEqual($entries.map((e) => [e.key, e.value]), ` + tc.expected + `);
const $snapshot = $canValue(await $methods.entries($initial));
assert.deepEqual($snapshot.map((e) => [e.key, e.value]), ` + tc.snapshot + `);
console.log("leaf ` + tc.keyKind + ` route passed");
`
			leafRunScript(t, bun, "leaf-route-"+tc.keyKind+".ts", code, "leaf "+tc.keyKind+" route passed")
		})
	}
}

// leafFixtureCallSpan locates one authored call text inside a single
// function scope of a fixture source, returning exact byte offsets.
func leafFixtureCallSpan(t *testing.T, source, fnMarker, callText string) (int, int) {
	t.Helper()
	fnAt := strings.Index(source, fnMarker)
	if fnAt < 0 {
		t.Fatalf("function marker %q not found", fnMarker)
	}
	rest := source[fnAt:]
	scope := rest
	if endRel := strings.Index(rest, "\nfn "); endRel >= 0 {
		scope = rest[:endRel]
	}
	at := strings.Index(scope, callText)
	if at < 0 {
		t.Fatalf("call text %q not found in %s scope", callText, fnMarker)
	}
	return fnAt + at, fnAt + at + len(callText)
}

// leafMappingPair requires the exact doubled mapping tokens for one
// authored span: same operation/source/start/end on consecutive emitted
// lines, the first carrying the origin assignment and the second the
// expected lowered statement.
func leafMappingPair(t *testing.T, lines []string, mappings []ir.Mapping, source, op string, start, end int, nextContains string) {
	t.Helper()
	var found []ir.Mapping
	for _, m := range mappings {
		if m.Operation == op && m.Source == source && m.Start == start && m.End == end {
			found = append(found, m)
		}
	}
	if len(found) != 2 {
		t.Fatalf("want exactly one %s mapping pair at %s:%d-%d, have %d", op, source, start, end, len(found))
	}
	if found[1].Line != found[0].Line+1 {
		t.Fatalf("mapping pair not on consecutive lines: %+v %+v", found[0], found[1])
	}
	first, second := lines[found[0].Line-1], lines[found[1].Line-1]
	if !strings.Contains(first, "$canOrigin = ") {
		t.Fatalf("mapping first line lacks origin assignment:\n%s", first)
	}
	if !strings.Contains(second, nextContains) {
		t.Fatalf("mapping second line lacks %q:\n%s", nextContains, second)
	}
}

func TestMapLeafEmittedMappingPairs(t *testing.T) {
	frequency, err := os.ReadFile(filepath.Join("..", "..", "..", "tools", "performance", "fixtures", "runtime", "src", "frequency.can"))
	if err != nil {
		t.Fatal(err)
	}
	program := mapLeafGalleryProgram(t)
	assembly := mapLeafAssembly(t, program)
	proof := assembly.checkedMapLeaves()
	counter := mapLeafFunction(t, program, "gallery::count_one")
	entry := proof[counter.Identity()]
	if entry == nil {
		t.Fatal("actual gallery count_one lacks a map-leaf proof")
	}
	clean, mappings := leafCompanionText(t, assembly, counter, entry)
	lines := strings.Split(clean, "\n")
	for _, m := range mappings {
		if m.Source != entry.Source {
			t.Fatalf("misplaced cross-module mapping: %+v", m)
		}
	}
	for _, call := range []string{"collections::get(totals, word)", "collections::insert(totals, word, 1)", "collections::replace(totals, word, previous + 1)"} {
		start, end := leafFixtureCallSpan(t, string(frequency), "count_one", call)
		leafMappingPair(t, lines, mappings, entry.Source, "call", start, end, "$canInvokeSync(")
	}
	recovery := mapLeafProgram(t, map[string]string{"src/main.can": mapLeafRecoveryCorpus})
	recoveryAssembly := mapLeafAssembly(t, recovery)
	recoveryProof := recoveryAssembly.checkedMapLeaves()
	recoverFn := mapLeafFunction(t, recovery, "app::count_recover")
	recoverEntry := recoveryProof[recoverFn.Identity()]
	if recoverEntry == nil {
		t.Fatal("recovery leaf lacks a map-leaf proof")
	}
	recoverClean, recoverMappings := leafCompanionText(t, recoveryAssembly, recoverFn, recoverEntry)
	recoverLines := strings.Split(recoverClean, "\n")
	for _, m := range recoverMappings {
		if m.Source != recoverEntry.Source {
			t.Fatalf("misplaced cross-module mapping: %+v", m)
		}
	}
	for _, call := range []string{"collections::get(totals, word)", "collections::insert(totals, word, 1)", "collections::replace(totals, word, previous + 1)"} {
		start, end := leafFixtureCallSpan(t, mapLeafRecoveryCorpus, "count_recover", call)
		leafMappingPair(t, recoverLines, recoverMappings, recoverEntry.Source, "call", start, end, "$canInvokeSync(")
	}
	armAt := strings.Index(mapLeafRecoveryCorpus, "[_] as standard_failure snap => ok totals")
	if armAt < 0 {
		t.Fatal("standard arm text not found")
	}
	bindAt := armAt + len("[_] as standard_failure snap => ok ")
	leafMappingPair(t, recoverLines, recoverMappings, recoverEntry.Source, "binding", bindAt, bindAt+len("totals"), "= $w0;")
}

// leafCallSiteOrigin renders the exact expected call-site origin literal
// for an emitted worker call span.
func leafCallSiteOrigin(source string, start, end int, invocation string) string {
	return fmt.Sprintf(`{source: %s, start: %d, end: %d, invocation: [%s]}`, quote(source), start, end, quote(invocation))
}

func TestMapLeafEmittedColdStandardBoundary(t *testing.T) {
	bun := os.Getenv("CAN_BUN")
	if bun == "" {
		t.Skip("set CAN_BUN for cold boundary execution")
	}
	frequency, err := os.ReadFile(filepath.Join("..", "..", "..", "tools", "performance", "fixtures", "runtime", "src", "frequency.can"))
	if err != nil {
		t.Fatal(err)
	}
	program := mapLeafGalleryProgram(t)
	assembly := mapLeafAssembly(t, program)
	proof := assembly.checkedMapLeaves()
	counter := mapLeafFunction(t, program, "gallery::count_one")
	entry := proof[counter.Identity()]
	if entry == nil {
		t.Fatal("actual gallery count_one lacks a map-leaf proof")
	}
	clean, _ := leafCompanionText(t, assembly, counter, entry)
	methods := leafWorkerKinds(t, clean, 3, []string{"get", "insert", "replace"})
	getStart, getEnd := leafFixtureCallSpan(t, string(frequency), "count_one", "collections::get(totals, word)")
	getSite := leafCallSiteOrigin(entry.Source, getStart, getEnd, entry.Region.ID)
	runtimeRoot, _ := filepath.Abs("../../../runtime")
	action := `const $origin = {source: "leaf-route.can", start: 0, end: 1, invocation: ["app::run"]};
const $action = $canOwnCallable("p::leaf#0", "app::count_one", [], async () => { throw new Error("slow adapter reached"); }, [], undefined, undefined, {companion: ` + entry.Companion + `, keyKind: "str", origin: Object.freeze({source: "leaf-route.can", start: 0, end: 1, invocation: Object.freeze(["app::count_one"])})});
const $getSite = ` + getSite + `;
`
	wrongHost := leafHarnessHead(runtimeRoot) + clean + "\n" +
		leafDomainSetup("map-str-int", "str") +
		leafReceiverDecls(methods, nil) + action + `
// HOST INJECTION (outside valid Can inputs): fold initial from a foreign factory.
const $foreign = $canCreateMap($domain, {map: "map-str-int-foreign", entry: "entry-test", absent: $shapes[0].identity, exists: $shapes[1].identity}, "str");
const $hostile = $canValue(await $foreign.empty());
const $r1 = await $canFold($canArray(["a", "b"]), $hostile, $action, {origin: $origin, site: "p::leaf#0"});
const $r2 = await $canFold($canArray(["a", "b"]), $hostile, $action, {origin: $origin, site: "p::leaf#0"});
for (const $r of [$r1, $r2]) {
  assert.equal($r.kind, "standard");
  const $d = $canStandardDiagnostics($r.value);
  assert.equal($d.message, "resource_state: invalid resource use");
  assert.deepEqual($d.origin, {source: "can:collections:map", start: 0, end: 0, invocation: []});
  assert.deepEqual($d.boundaryOrigin, $getSite);
}
const $o1 = $canStandardOccurrence($r1.value);
const $o2 = $canStandardOccurrence($r2.value);
assert($o1 !== $o2);
assert($o2 === $o1 + 1n);
console.log("leaf wrong-host boundary passed");
`
	leafRunScript(t, bun, "leaf-route-wrong-host.ts", wrongHost, "leaf wrong-host boundary passed")
	overrides := map[string]string{}
	for receiver, kind := range methods {
		if dot := strings.Index(receiver, "."); dot >= 0 {
			overrides[receiver[:dot]] = "{get: $badWorker, insert: $methods.insert, replace: $methods.replace}"
		} else if kind == "get" {
			overrides[receiver] = "$badWorker"
		}
	}
	missingWorker := leafHarnessHead(runtimeRoot) + clean + "\n" +
		leafDomainSetup("map-str-int", "str") +
		"const $badWorker = () => { throw new Error(\"injected worker must stay unregistered\"); };\n" +
		leafReceiverDecls(methods, overrides) + action + `
// HOST INJECTION (outside valid Can inputs): get receiver without a private worker.
const $seed = $canValue(await $methods.empty());
const $m1 = await $canFold($canArray(["a", "b"]), $seed, $action, {origin: $origin, site: "p::leaf#0"});
const $m2 = await $canFold($canArray(["a", "b"]), $seed, $action, {origin: $origin, site: "p::leaf#0"});
for (const $r of [$m1, $m2]) {
  assert.equal($r.kind, "standard");
  const $d = $canStandardDiagnostics($r.value);
  assert.equal($d.kind, "native_exception");
  assert.deepEqual($d.origin, $getSite);
  assert.equal($d.boundaryOrigin, undefined);
}
const $p1 = $canStandardOccurrence($m1.value);
const $p2 = $canStandardOccurrence($m2.value);
assert($p1 !== $p2);
assert($p2 === $p1 + 1n);
console.log("leaf missing-worker boundary passed");
`
	leafRunScript(t, bun, "leaf-route-missing-worker.ts", missingWorker, "leaf missing-worker boundary passed")
}

func TestMapLeafEmittedStandardRecovery(t *testing.T) {
	bun := os.Getenv("CAN_BUN")
	if bun == "" {
		t.Skip("set CAN_BUN for recovery execution")
	}
	program := mapLeafProgram(t, map[string]string{"src/main.can": mapLeafRecoveryCorpus})
	assembly := mapLeafAssembly(t, program)
	proof := assembly.checkedMapLeaves()
	recoverFn := mapLeafFunction(t, program, "app::count_recover")
	entry := proof[recoverFn.Identity()]
	if entry == nil {
		t.Fatal("recovery leaf lacks a map-leaf proof")
	}
	clean, _ := leafCompanionText(t, assembly, recoverFn, entry)
	methods := leafWorkerKinds(t, clean, 3, []string{"get", "insert", "replace"})
	runtimeRoot, _ := filepath.Abs("../../../runtime")
	code := leafHarnessHead(runtimeRoot) + clean + "\n" +
		leafDomainSetup("map-str-int", "str") +
		leafReceiverDecls(methods, nil) + `
const $seed = $canValue(await $methods.empty());
const $initial = $canValue(await $methods.insert($seed, "z", 9n));
const $origin = {source: "leaf-route.can", start: 0, end: 1, invocation: ["app::run"]};
const $action = $canOwnCallable("p::leaf#0", "app::count_recover", [], async () => { throw new Error("slow adapter reached"); }, [], undefined, undefined, {companion: ` + entry.Companion + `, keyKind: "str", origin: Object.freeze({source: "leaf-route.can", start: 0, end: 1, invocation: Object.freeze(["app::count_recover"])})});
const $result = await $canFold($canArray(["a", "b", "a", "a"]), $initial, $action, {origin: $origin, site: "p::leaf#0"});
const $entries = $canValue(await $methods.entries($canValue($result)));
assert.deepEqual($entries.map((e) => [e.key, e.value]), [["z", 9n], ["a", 3n], ["b", 1n]]);
// HOST INJECTION (outside valid Can inputs): every visit fails closed and
// the checked standard arm recovers the untouched input snapshot.
const $foreign = $canCreateMap($domain, {map: "map-str-int-foreign", entry: "entry-test", absent: $shapes[0].identity, exists: $shapes[1].identity}, "str");
const $hostile = $canValue(await $foreign.empty());
const $recovered = await $canFold($canArray(["a", "b"]), $hostile, $action, {origin: $origin, site: "p::leaf#0"});
assert.equal($recovered.kind, "ok");
assert.equal($canValue($recovered), $hostile);
console.log("leaf recovery passed");
`
	leafRunScript(t, bun, "leaf-route-recovery.ts", code, "leaf recovery passed")
}

func TestMapLeafEmittedIdentityHostile(t *testing.T) {
	bun := os.Getenv("CAN_BUN")
	if bun == "" {
		t.Skip("set CAN_BUN for identity execution")
	}
	program := mapLeafProgram(t, map[string]string{"src/main.can": mapLeafRecoveryCorpus})
	assembly := mapLeafAssembly(t, program)
	proof := assembly.checkedMapLeaves()
	identityFn := mapLeafFunction(t, program, "app::identity_map")
	entry := proof[identityFn.Identity()]
	if entry == nil {
		t.Fatal("identity leaf lacks a map-leaf proof")
	}
	clean, _ := leafCompanionText(t, assembly, identityFn, entry)
	methods := leafWorkerKinds(t, clean, 0, []string{})
	runtimeRoot, _ := filepath.Abs("../../../runtime")
	code := leafHarnessHead(runtimeRoot) + clean + "\n" +
		leafDomainSetup("map-str-int", "str") +
		leafReceiverDecls(methods, nil) + `
const $seed = $canValue(await $methods.empty());
const $initial = $canValue(await $methods.insert($seed, "z", 9n));
const $origin = {source: "leaf-route.can", start: 0, end: 1, invocation: ["app::run"]};
const $action = $canOwnCallable("p::leaf#0", "app::identity_map", [], async () => { throw new Error("slow adapter reached"); }, [], undefined, undefined, {companion: ` + entry.Companion + `, keyKind: "str", origin: Object.freeze({source: "leaf-route.can", start: 0, end: 1, invocation: Object.freeze(["app::identity_map"])})});
const $result = await $canFold($canArray(["a", "b"]), $initial, $action, {origin: $origin, site: "p::leaf#0"});
assert.equal($result.kind, "ok");
assert.equal($canValue($result), $initial);
// HOST INJECTION (outside valid Can inputs): hostile accumulator payload
// stays opaque through the emitted identity leaf and fold.
let $touches = 0;
const $hostile = { get then() { $touches++; return undefined; }, get probe() { $touches++; return 1; } };
const $held = await $canFold($canArray(["a", "b"]), $hostile, $action, {origin: $origin, site: "p::leaf#0"});
assert.equal($held.kind, "ok");
assert.equal($canValue($held), $hostile);
assert.equal($touches, 0);
console.log("leaf identity hostile passed");
`
	leafRunScript(t, bun, "leaf-route-identity.ts", code, "leaf identity hostile passed")
}
