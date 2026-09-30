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

// collectionCorpus exercises every audited canonical map/set operation so
// assembly proof covers the full finite table from actual checked code.
const collectionCorpus = `package app
    provides []
    uses [collections]
fn int map_ops
    emits {collections::key_exists, collections::key_absent}
    asserts
        sample: => ok 201
    match chain
        call collections::empty_map<int,int>() as collections::map<int,int> m0
        call collections::insert(m0, 1, 10) as collections::map<int,int> m1
        call collections::insert(m1, 2, 20) as collections::map<int,int> m2
        call collections::replace(m2, 1, 100) as collections::map<int,int> m3
        call collections::remove(m3, 2) as collections::map<int,int> m4
        call collections::get(m4, 1) as int found
        call collections::entries(m4) as collections::entry<int,int>[] rows
        call collections::build_map<int,int>(rows) as collections::map<int,int> rebuilt
        call collections::get(rebuilt, 1) as int again
        collections::key_exists
        collections::key_absent
        ok => ok found + again + rows.length
fn int set_ops
    emits {}
    asserts
        sample: => ok 7
    collections::set<int> s0 = call collections::empty_set<int>()
    collections::set<int> s1 = call collections::add(s0, 1)
    collections::set<int> s2 = call collections::add(s1, 2)
    collections::set<int> s3 = call collections::build_set<int>([3, 4])
    collections::set<int> u = call collections::union(s2, s3)
    collections::set<int> i = call collections::intersection(s2, s3)
    collections::set<int> d = call collections::difference(u, s2)
    bool cu = call collections::contains(u, 4)
    bool ci = call collections::contains(i, 4)
    bool cd = call collections::contains(d, 4)
    match cu, ci, cd
        true, false, true => ok 7
        _, _, _ => ok 0
fn void main
    emits {}
    given
        str[] arguments
    asserts
        empty: [] => ok
    ok
`

func collectionProgram(t *testing.T) *check.Program {
	t.Helper()
	return actionEmitProgram(t, map[string]string{"src/main.can": collectionCorpus})
}

func collectionAssembly(t *testing.T, program *check.Program) *programAssembly {
	t.Helper()
	assembly, err := assembleProgramBindings(program)
	if err != nil {
		t.Fatal(err)
	}
	return assembly
}

func TestCollectionProofAssemblyExact(t *testing.T) {
	program := collectionProgram(t)
	assembly := collectionAssembly(t, program)
	want := map[string]string{
		"can.std.collections@1::empty_map": "empty", "can.std.collections@1::build_map": "build_map",
		"can.std.collections@1::get": "get", "can.std.collections@1::insert": "insert",
		"can.std.collections@1::replace": "replace", "can.std.collections@1::remove": "remove",
		"can.std.collections@1::entries": "entries", "can.std.collections@1::empty_set": "empty",
		"can.std.collections@1::build_set": "build_set", "can.std.collections@1::contains": "contains",
		"can.std.collections@1::add": "add", "can.std.collections@1::union": "union",
		"can.std.collections@1::intersection": "intersection", "can.std.collections@1::difference": "difference",
	}
	seen := map[string]bool{}
	for key, special := range program.Collections {
		method, ok := want[special.Operation]
		if !ok {
			t.Fatalf("unexpected collection operation %s", special.Operation)
		}
		seen[special.Operation] = true
		receiver := assembly.collectionNames[special.Collection.Identity()]
		if receiver == "" {
			t.Fatalf("missing receiver for %s", key)
		}
		if special.Operation == "can.std.collections@1::get" && special.Entry == nil {
			t.Fatal("map operation lost Entry pairing")
		}
		if special.Operation == "can.std.collections@1::contains" && special.Entry != nil {
			t.Fatal("set operation gained Entry pairing")
		}
		bound, ok := assembly.collectionAsyncProof[key]
		if !ok {
			t.Fatalf("canonical key %s missing from proof", key)
		}
		if bound != receiver+"."+method || bound != assembly.functions[key] {
			t.Fatalf("key %s proof %q disagrees with receiver %q method %q merged %q", key, bound, receiver, method, assembly.functions[key])
		}
		if _, clash := assembly.authoredProof[key]; clash {
			t.Fatalf("collection key %s leaked into authored proof", key)
		}
	}
	if len(seen) != 14 {
		t.Fatalf("only %d of 14 canonical operations covered: %v", len(seen), seen)
	}
	if len(assembly.collectionAsyncProof) != len(program.Collections) {
		t.Fatalf("proof size %d differs from collection count %d", len(assembly.collectionAsyncProof), len(program.Collections))
	}
	for key := range assembly.authoredProof {
		if _, clash := assembly.collectionAsyncProof[key]; clash {
			t.Fatalf("authored key %s leaked into collection proof", key)
		}
	}
}

func TestCollectionProofAssemblyNegatives(t *testing.T) {
	program := collectionProgram(t)
	assembly := collectionAssembly(t, program)
	var mapKey, setKey string
	for key, special := range program.Collections {
		if special.Operation == "can.std.collections@1::get" {
			mapKey = key
		}
		if special.Operation == "can.std.collections@1::contains" {
			setKey = key
		}
	}
	if mapKey == "" || setKey == "" {
		t.Fatal("real program lacks map/set specializations")
	}
	build := func(mut func(collections map[string]*check.CollectionSpecialization, functions, names map[string]string)) map[string]string {
		collections := map[string]*check.CollectionSpecialization{}
		for key, special := range program.Collections {
			collections[key] = special
		}
		functions := map[string]string{}
		for key, target := range assembly.functions {
			functions[key] = target
		}
		names := map[string]string{}
		for id, receiver := range assembly.collectionNames {
			names[id] = receiver
		}
		mut(collections, functions, names)
		probe := &programAssembly{program: &check.Program{Collections: collections}, functions: functions, collectionNames: names}
		return probe.checkedCollectionProof()
	}
	full := build(func(map[string]*check.CollectionSpecialization, map[string]string, map[string]string) {})
	if len(full) != len(program.Collections) {
		t.Fatal("faithful copy lost proof entries")
	}
	original := *program.Collections[mapKey]
	cases := map[string]func(map[string]*check.CollectionSpecialization, map[string]string, map[string]string){
		"unknown operation": func(collections map[string]*check.CollectionSpecialization, _ map[string]string, _ map[string]string) {
			mut := original
			mut.Operation = "can.std.collections@1::bogus"
			collections[mapKey] = &mut
		},
		"map kind lost": func(collections map[string]*check.CollectionSpecialization, _ map[string]string, _ map[string]string) {
			mut := original
			mut.Entry = nil
			collections[mapKey] = &mut
		},
		"set kind gained": func(collections map[string]*check.CollectionSpecialization, _ map[string]string, _ map[string]string) {
			set := *program.Collections[setKey]
			set.Entry = original.Entry
			collections[setKey] = &set
		},
		"rebound target": func(_ map[string]*check.CollectionSpecialization, functions map[string]string, _ map[string]string) {
			functions[mapKey] = "$canCollection9.get"
		},
		"cross receiver": func(_ map[string]*check.CollectionSpecialization, functions map[string]string, _ map[string]string) {
			functions[mapKey] = strings.Replace(assembly.functions[mapKey], "$canCollection", "$canCollection9", 1)
		},
		"missing target": func(_ map[string]*check.CollectionSpecialization, functions map[string]string, _ map[string]string) {
			delete(functions, mapKey)
		},
		"missing receiver": func(_ map[string]*check.CollectionSpecialization, _ map[string]string, names map[string]string) {
			delete(names, original.Collection.Identity())
		},
		"nil specialization": func(collections map[string]*check.CollectionSpecialization, _ map[string]string, _ map[string]string) {
			collections[mapKey] = nil
		},
		"nil collection": func(collections map[string]*check.CollectionSpecialization, _ map[string]string, _ map[string]string) {
			mut := original
			mut.Collection = nil
			collections[mapKey] = &mut
		},
	}
	for name, mut := range cases {
		proof := build(mut)
		if _, kept := proof[mapKey]; kept && !strings.HasPrefix(name, "set ") {
			t.Fatalf("%s kept map proof", name)
		}
		if _, kept := proof[setKey]; name == "set kind gained" && kept {
			t.Fatalf("%s kept set proof", name)
		}
		want := len(program.Collections) - 1
		if name == "missing receiver" {
			// All map operations share one receiver; only set keys survive.
			want = 0
			for _, special := range program.Collections {
				if special.Entry == nil {
					want++
				}
			}
		}
		if len(proof) != want {
			t.Fatalf("%s proof size %d, want %d", name, len(proof), want)
		}
	}
	emptyKey := build(func(collections map[string]*check.CollectionSpecialization, _ map[string]string, _ map[string]string) {
		collections[""] = &original
	})
	if _, kept := emptyKey[""]; kept {
		t.Fatal("empty key qualified")
	}
}

func TestCollectionEligibilityPredicate(t *testing.T) {
	proof := map[string]string{"spec/get": "$canCollection0.get", "spec/add": "$canCollection1.add"}
	emitter := RegionEmitter{collectionProof: proof}
	plain := func() *ir.InvocationStep {
		return &ir.InvocationStep{Identity: "spec/get", Site: "s1"}
	}
	if !emitter.eligibleCollectionBypass(plain(), "$canCollection0.get") {
		t.Fatal("exact collection proof did not qualify")
	}
	if !emitter.provenCollectionBinding("spec/add", "$canCollection1.add") {
		t.Fatal("exact set proof did not qualify")
	}
	cases := map[string]struct {
		emitter RegionEmitter
		step    *ir.InvocationStep
		target  string
	}{
		"nil proof":              {RegionEmitter{}, plain(), "$canCollection0.get"},
		"empty proof":            {RegionEmitter{collectionProof: map[string]string{}}, plain(), "$canCollection0.get"},
		"stale target":           {emitter, plain(), "$canCollection9.get"},
		"unknown identity":       {emitter, &ir.InvocationStep{Identity: "spec/bogus", Site: "s1"}, "$canCollection0.get"},
		"cross specialization":   {emitter, &ir.InvocationStep{Identity: "spec/add", Site: "s1"}, "$canCollection0.get"},
		"cross receiver":         {emitter, plain(), "$canCollection1.get"},
		"bare method":            {emitter, plain(), "get"},
		"empty identity":         {emitter, &ir.InvocationStep{Site: "s1"}, "$canCollection0.get"},
		"empty target":           {emitter, plain(), ""},
		"nil step":               {emitter, nil, "$canCollection0.get"},
		"browser":                {RegionEmitter{Browser: true, collectionProof: proof}, plain(), "$canCollection0.get"},
		"authored proof ignored": {RegionEmitter{authoredProof: map[string]string{"spec/get": "$canCollection0.get"}}, plain(), "$canCollection0.get"},
		"callee":                 {emitter, &ir.InvocationStep{Identity: "spec/get", Site: "s1", Callee: &ir.Expression{}}, "$canCollection0.get"},
		"native":                 {emitter, &ir.InvocationStep{Identity: "spec/get", Site: "s1", Native: &ir.Expression{}}, "$canCollection0.get"},
		"array":                  {emitter, &ir.InvocationStep{Identity: "spec/get", Site: "s1", Array: &ir.ArrayOperation{}}, "$canCollection0.get"},
		"asset":                  {emitter, &ir.InvocationStep{Identity: "spec/get", Site: "s1", Asset: &ir.AssetResolution{}}, "$canCollection0.get"},
		"fixtures":               {emitter, &ir.InvocationStep{Identity: "spec/get", Site: "s1", Fixtures: &ir.FixtureTable{}}, "$canCollection0.get"},
		"sql":                    {emitter, &ir.InvocationStep{Identity: "spec/get", Site: "s1", SQL: &ir.SQLCallSite{}}, "$canCollection0.get"},
		"form action":            {emitter, &ir.InvocationStep{Identity: "spec/get", Site: "s1", FormAction: &ir.FormActionSite{}}, "$canCollection0.get"},
		"json fetch":             {emitter, &ir.InvocationStep{Identity: "spec/get", Site: "s1", JSONFetch: &ir.JSONFetchSite{}}, "$canCollection0.get"},
		"action":                 {emitter, &ir.InvocationStep{Identity: "spec/get", Site: "s1", Action: &ir.ActionSite{}}, "$canCollection0.get"},
	}
	for name, tc := range cases {
		if tc.emitter.eligibleCollectionBypass(tc.step, tc.target) {
			t.Fatalf("%s qualified for collection bypass", name)
		}
	}
	if len(proof) != 2 || proof["spec/get"] != "$canCollection0.get" {
		t.Fatalf("shared proof mutated: %v", proof)
	}
}

// collectionBun resolves the installed Bun binary. Collection contracts
// never skip: Bun is installed and required.
func collectionBun(t *testing.T) string {
	t.Helper()
	if bun := os.Getenv("CAN_BUN"); bun != "" {
		return bun
	}
	bun, err := exec.LookPath("bun")
	if err != nil {
		t.Fatal("bun is required for collection execution")
	}
	return bun
}

func collectionRun(t *testing.T, name, code string) string {
	t.Helper()
	file := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(file, []byte(code), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, collectionBun(t), "--no-install", "--no-env-file", file).CombinedOutput()
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	return string(out)
}

// collectionDomainPlan builds a domain runtime plan covering the collection
// failure identities of a real checked program.
func collectionDomainPlan(t *testing.T, program *check.Program) (plan string, absent, exists string) {
	t.Helper()
	failures := []*types.Type{}
	seen := map[string]bool{}
	for _, special := range program.Collections {
		for _, failure := range special.Contract.Errors() {
			if failure != nil && !seen[failure.Identity()] {
				seen[failure.Identity()] = true
				failures = append(failures, failure)
			}
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
	numberIDs := map[string]string{}
	for _, typ := range program.Model.Types() {
		numberIDs[typ.Declaration()] = typ.Identity()
	}
	return string(encoded), numberIDs["can.std.collections@1::key_absent"], numberIDs["can.std.collections@1::key_exists"]
}

func TestCollectionFactoryPremise(t *testing.T) {
	program := collectionProgram(t)
	plan, absent, exists := collectionDomainPlan(t, program)
	if absent == "" || exists == "" {
		t.Fatal("collection failure identities missing")
	}
	runtimeRoot, _ := filepath.Abs("../../../runtime")
	path := func(name string) string { p, _ := filepath.Abs(filepath.Join(runtimeRoot, name+".ts")); return p }
	collectionsPath := func(name string) string {
		p, _ := filepath.Abs(filepath.Join(runtimeRoot, "collections", name+".ts"))
		return p
	}
	code := `import {strict as assert} from "node:assert";` + "\n" +
		CompletionImports(path("completion")) +
		fmt.Sprintf("import {createDomainRuntime} from %s;\n", quote(path("domain"))) +
		fmt.Sprintf("import {createMap as $canCreateMap} from %s;\n", quote(collectionsPath("map"))) +
		fmt.Sprintf("import {createSet as $canCreateSet} from %s;\n", quote(collectionsPath("set"))) +
		fmt.Sprintf("const $canDomain = createDomainRuntime(%s);\n", plan) +
		fmt.Sprintf("const mapFactory = $canCreateMap($canDomain, {map: %s, entry: %s, absent: %s, exists: %s}, %s);\n",
			quote("collections::map<int,int>"), quote("collections::entry<int,int>"), quote(absent), quote(exists), quote("int")) +
		fmt.Sprintf("const setFactory = $canCreateSet(%s, %s);\n", quote("collections::set<int>"), quote("int")) + `
const mapMethods = ["empty", "build_map", "get", "insert", "replace", "remove", "entries"];
const setMethods = ["empty", "build_set", "contains", "add", "union", "intersection", "difference"];
assert.equal(Object.isFrozen(mapFactory), true);
assert.equal(Object.isFrozen(setFactory), true);
for (const name of mapMethods) {
  assert.equal(typeof (mapFactory as any)[name], "function");
  assert.equal((mapFactory as any)[name].constructor.name, "AsyncFunction");
}
for (const name of setMethods) {
  assert.equal(typeof (setFactory as any)[name], "function");
  assert.equal((setFactory as any)[name].constructor.name, "AsyncFunction");
}
const empty = await (mapFactory as any).empty(undefined);
assert.equal(empty.kind, "ok");
const built = await (mapFactory as any).insert($canValue(empty), 1n, 2n, undefined);
assert.equal(built.kind, "ok");
const found = await (mapFactory as any).get($canValue(built), 1n, undefined);
assert.equal(found.kind, "ok");
assert.equal($canValue(found), 2n);
const missing = await (mapFactory as any).get($canValue(built), 9n, undefined);
assert.equal(missing.kind, "domain");
const setEmpty = await (setFactory as any).empty(undefined);
assert.equal(setEmpty.kind, "ok");
const added = await (setFactory as any).add($canValue(setEmpty), 5n, undefined);
assert.equal($canValue(await (setFactory as any).contains($canValue(added), 5n, undefined)), true);
console.log("factory premise passed");
`
	out := collectionRun(t, "collection-premise.ts", code)
	if !strings.Contains(out, "factory premise passed") {
		t.Fatalf("factory premise failed:\n%s", out)
	}
}

func TestCollectionExecutesMapSetRoutes(t *testing.T) {
	program := collectionProgram(t)
	assembly := collectionAssembly(t, program)
	keyOf := func(operation string) string {
		for key, special := range program.Collections {
			if special.Operation == operation {
				return key
			}
		}
		t.Fatalf("missing specialization for %s", operation)
		return ""
	}
	var mapT, setT *types.Type
	for _, special := range program.Collections {
		if special.Operation == "can.std.collections@1::get" {
			mapT = special.Collection
		}
		if special.Operation == "can.std.collections@1::contains" {
			setT = special.Collection
		}
	}
	fixture := newRegionFixture(t)
	mustCallable := func(result *types.Type, inputs ...*types.Type) *types.Type {
		contract, err := types.CallableOfChecked(result, inputs, nil)
		if err != nil {
			t.Fatal(err)
		}
		return contract
	}
	emptyMapKey, insertKey, getKey, removeKey := keyOf("can.std.collections@1::empty_map"), keyOf("can.std.collections@1::insert"), keyOf("can.std.collections@1::get"), keyOf("can.std.collections@1::remove")
	emptySetKey, addKey, containsKey := keyOf("can.std.collections@1::empty_set"), keyOf("can.std.collections@1::add"), keyOf("can.std.collections@1::contains")
	fixture.functions["cemptymap"] = check.ValueBinding{Identity: emptyMapKey, Type: mustCallable(mapT)}
	fixture.functions["cinsert"] = check.ValueBinding{Identity: insertKey, Type: mustCallable(mapT, mapT, fixture.ts["int"], fixture.ts["int"])}
	fixture.functions["cget"] = check.ValueBinding{Identity: getKey, Type: mustCallable(fixture.ts["int"], mapT, fixture.ts["int"])}
	fixture.functions["cremove"] = check.ValueBinding{Identity: removeKey, Type: mustCallable(mapT, mapT, fixture.ts["int"])}
	fixture.functions["cemptyset"] = check.ValueBinding{Identity: emptySetKey, Type: mustCallable(setT)}
	fixture.functions["cadd"] = check.ValueBinding{Identity: addKey, Type: mustCallable(setT, setT, fixture.ts["int"])}
	fixture.functions["ccontains"] = check.ValueBinding{Identity: containsKey, Type: mustCallable(fixture.ts["bool"], setT, fixture.ts["int"])}
	fixture.functions["cempty9"] = check.ValueBinding{Identity: "spec/empty9", Type: mustCallable(mapT)}
	fixture.functions["cget9"] = check.ValueBinding{Identity: "spec/get9", Type: mustCallable(fixture.ts["int"], mapT, fixture.ts["int"])}
	// hostile models an opaque boxed value: the checker sees int, the
	// harness passes a hostile object the factory stores uninspected.
	fixture.values["hostile"] = check.ValueBinding{Identity: "value/hostile", Type: fixture.ts["int"]}
	bodies := map[string]string{
		"$mapFlow": "    ok call cget(call cinsert(call cemptymap(), 1, 42), 1)\n",
		"$setFlow": "    ok call ccontains(call cadd(call cemptyset(), 5), 5)\n",
		"$dupKey":  "    ok call cget(call cinsert(call cinsert(call cemptymap(), 1, 42), 1, 43), 1)\n",
		"$missing": "    ok call cget(call cemptymap(), 9)\n",
		"$removed": "    ok call cget(call cremove(call cinsert(call cemptymap(), 1, 42), 1), 1)\n",
		"$hostile": "    ok call cget(call cinsert(call cemptymap(), 7, hostile), 7)\n",
		"$cold":    "    ok call cget9(call cemptymap(), 1)\n",
		"$defined": "    ok call cget(call cinsert(call cemptymap(), 3, 33), 3)\n",
	}
	functions := map[string]string{}
	for key, target := range assembly.functions {
		functions[key] = target
	}
	functions["spec/empty9"] = "$canCollection9.empty"
	functions["spec/get9"] = "$canCollection9.get"
	proof := map[string]string{}
	for key, target := range assembly.collectionAsyncProof {
		proof[key] = target
	}
	proof["spec/empty9"] = "$canCollection9.empty"
	proof["spec/get9"] = "$canCollection9.get"
	bindings := map[string]string{"value/hostile": "$hostileThen"}
	type emitted struct {
		body   string
		region *ir.Region
	}
	callers := map[string]emitted{}
	var regions []*ir.Region
	names := []string{"$mapFlow", "$setFlow", "$dupKey", "$missing", "$removed", "$hostile", "$cold", "$defined"}
	for _, name := range names {
		result := "int"
		if name == "$setFlow" {
			result = "bool"
		}
		region, err := fixture.region(t, bodies[name], result, nil, ir.FunctionRegion)
		if err != nil {
			t.Fatal(name, err)
		}
		emitter := RegionEmitter{Bindings: bindings, Functions: functions, SourceID: "region.can", collectionProof: proof}
		body, err := emitter.Function(name, region)
		if err != nil {
			t.Fatal(name, err)
		}
		callers[name] = emitted{body: body, region: region}
		regions = append(regions, region)
	}
	// Every caller takes the eligible member-call branch; receivers stay
	// attached and argument calls evaluate exactly once in order.
	for _, name := range names {
		body := callers[name].body
		if !strings.Contains(body, "$canContext === undefined ? $canCollection") {
			t.Fatalf("%s missed the collection bypass branch:\n%s", name, body)
		}
	}
	mapRecv := strings.TrimSuffix(assembly.functions[getKey], ".get")
	setRecv := strings.TrimSuffix(assembly.functions[containsKey], ".contains")
	mapBody := callers["$mapFlow"].body
	for _, want := range []string{
		"($canContext === undefined ? " + mapRecv + ".get(",
		"($canContext === undefined ? " + mapRecv + ".insert(",
		"($canContext === undefined ? " + mapRecv + ".empty(",
	} {
		if !strings.Contains(mapBody, want) {
			t.Fatalf("map flow missed %q:\n%s", want, mapBody)
		}
	}
	// One textual occurrence per branch; the ternary executes exactly one.
	if got := strings.Count(mapBody, "=== undefined ? "+mapRecv+".empty("); got != 1 {
		t.Fatalf("empty direct branch %d times, want once:\n%s", got, mapBody)
	}
	if got := strings.Count(mapBody, "=> "+mapRecv+".empty("); got != 1 {
		t.Fatalf("empty defined branch %d times, want once:\n%s", got, mapBody)
	}
	emptyAt, insertAt, getAt := strings.Index(mapBody, mapRecv+".empty("), strings.Index(mapBody, mapRecv+".insert("), strings.Index(mapBody, mapRecv+".get(")
	if !(emptyAt < insertAt && insertAt < getAt) {
		t.Fatalf("operand order broken:\n%s", mapBody)
	}
	if got := strings.Count(mapBody, "$canCallableInstance("+mapRecv+".get)"); got != 1 {
		t.Fatalf("get receipt evaluated %d times, want once in defined branch:\n%s", got, mapBody)
	}
	// The defined branch is byte-identical to the legacy wrapper.
	legacyRegion, err := fixture.region(t, bodies["$mapFlow"], "int", nil, ir.FunctionRegion)
	if err != nil {
		t.Fatal(err)
	}
	legacyEmitter := RegionEmitter{Bindings: bindings, Functions: functions, SourceID: "region.can"}
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
	if !strings.Contains(mapBody, " : "+rest[:end]+")") {
		t.Fatalf("defined branch differs from legacy lowering:\n%s", mapBody)
	}
	header := "package app\n    provides []\n    uses []\nfn int run\n    emits {}\n    asserts\n        test: => ok 1\n"
	coldSource := header + bodies["$cold"]
	coldStart := strings.Index(coldSource, "cget9(")
	coldCall := "cget9(call cemptymap(), 1)"
	declarations, err := RegionTypeDeclarations(regions...)
	if err != nil {
		t.Fatal(err)
	}
	plan, absent, exists := collectionDomainPlan(t, program)
	runtimeRoot, _ := filepath.Abs("../../../runtime")
	path := func(name string) string { p, _ := filepath.Abs(filepath.Join(runtimeRoot, name+".ts")); return p }
	collectionsPath := func(name string) string {
		p, _ := filepath.Abs(filepath.Join(runtimeRoot, "collections", name+".ts"))
		return p
	}
	var mapID, entryID, setID string
	for _, special := range program.Collections {
		if special.Operation == "can.std.collections@1::get" {
			mapID, entryID = special.Collection.Identity(), special.Entry.Identity()
		}
		if special.Operation == "can.std.collections@1::contains" {
			setID = special.Collection.Identity()
		}
	}
	contextPath := filepath.Join(runtimeRoot, "assert", "context.ts")
	var programTS strings.Builder
	programTS.WriteString(`import {strict as assert} from "node:assert";` + "\n")
	programTS.WriteString(CompletionImports(path("completion")))
	fmt.Fprintf(&programTS, "import {createDomainRuntime} from %s;\n", quote(path("domain")))
	fmt.Fprintf(&programTS, "import {domainFailureDiagnostics} from %s;\n", quote(path("domain")))
	fmt.Fprintf(&programTS, "import {standardFailureDiagnostics} from %s;\n", quote(path("failure")))
	fmt.Fprintf(&programTS, "import {createMap as $canCreateMap} from %s;\n", quote(collectionsPath("map")))
	fmt.Fprintf(&programTS, "import {createSet as $canCreateSet} from %s;\n", quote(collectionsPath("set")))
	fmt.Fprintf(&programTS, "import {assertionContext as $canAssertionContext, contextIdentity as $canContextIdentity, closeContext as $canCloseContext} from %s;\n", quote(contextPath))
	fmt.Fprintf(&programTS, "const $canDomain = createDomainRuntime(%s);\n", plan)
	fmt.Fprintf(&programTS, "const %s = $canCreateMap($canDomain, {map: %s, entry: %s, absent: %s, exists: %s}, %s);\n",
		mapRecv, quote(mapID), quote(entryID), quote(absent), quote(exists), quote("int"))
	fmt.Fprintf(&programTS, "const %s = $canCreateSet(%s, %s);\n", setRecv, quote(setID), quote("int"))
	programTS.WriteString("let $canCollection9: any;\n")
	programTS.WriteString(declarations)
	for _, name := range names {
		stripped, _, err := extractMappings(callers[name].body)
		if err != nil {
			t.Fatal(err)
		}
		programTS.WriteString(stripped)
	}
	fmt.Fprintf(&programTS, `
let thenCalls = 0;
const $hostileThen: any = {then() { thenCalls++; throw new Error("assimilated"); }};
assert.equal($canValue(await $mapFlow()), 42n);
assert.equal($canValue(await $setFlow()), true);
const dup = await $dupKey();
assert.equal(dup.kind, "domain");
const missA = await $missing(), missB = await $missing();
assert.equal(missA.kind, "domain");
assert.equal(missB.kind, "domain");
assert.notEqual(domainFailureDiagnostics(missA.value).occurrenceID, domainFailureDiagnostics(missB.value).occurrenceID);
const gone = await $removed();
assert.equal(gone.kind, "domain");
const boxed = await $hostile();
assert.strictEqual($canValue(boxed), $hostileThen);
assert.equal(thenCalls, 0);
const coldA = await $cold(), coldB = await $cold();
assert.equal(coldA.kind, "standard");
assert.equal(coldB.kind, "standard");
const coldSpan = {source: "region.can", start: %d, end: %d, invocation: ["app::run"]};
assert.deepEqual(standardFailureDiagnostics(coldA.value).origin, coldSpan);
assert.deepEqual(standardFailureDiagnostics(coldB.value).origin, coldSpan);
assert.notEqual(standardFailureDiagnostics(coldA.value).occurrenceID, standardFailureDiagnostics(coldB.value).occurrenceID);
const root = Object.freeze({package: "can.project.root/app", declaration: "can.project.root/app::subject", name: "sample"});
const context = $canAssertionContext(root);
const parent = $canContextIdentity(context);
assert.equal($canValue(await $defined(context)), 33n);
assert.equal($canContextIdentity(context), parent);
$canCloseContext(context);
console.log("collection routes passed");
`, coldStart, coldStart+len(coldCall))
	out := collectionRun(t, "collection-routes.ts", programTS.String())
	if !strings.Contains(out, "collection routes passed") {
		t.Fatalf("route execution failed:\n%s", out)
	}
}
