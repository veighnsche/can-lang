package emit

import (
	"context"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

const integerWorkerMain = `package app
    provides []
    uses []

fn int double
    emits {}
    given
        int value
    asserts
        sample: 2 => ok 4
    ok value * 2

fn int negate
    emits {}
    given
        int value
    asserts
        sample: 2 => ok -2
    ok -value

fn int add
    emits {}
    given
        int total
        int value
    asserts
        sample: 2, 3 => ok 5
    ok total + value

fn int combined
    emits {}
    given
        int first
        int second
        int third
    asserts
        sample: 1, 2, 3 => ok 9
    ok (first + second) * third

fn item identity<item>
    emits {}
    given
        item input
    asserts
        integer: 4 => ok 4
    ok input

fn int add_offset
    emits {}
    given
        near int offset
        int number
    asserts
        sample: 3, 4 => ok 7
    ok offset + number

fn int[] doubled
    emits {}
    given
        int[] values
    asserts
        sample: [1, 2, 3] => ok [2, 4, 6]
        empty: [] => ok []
    ok call values.map(callable double)

fn int sum
    emits {}
    given
        int[] values
    asserts
        sample: [1, 2, 3] => ok 6
        empty: [] => ok 0
    ok call values.fold(0, callable add)

fn int[] generic_pass
    emits {}
    given
        int[] values
    asserts
        sample: [1, 2] => ok [1, 2]
        empty: [] => ok []
    ok call values.map(callable identity<int>)

fn int[] captured_map
    emits {}
    given
        int[] values
        int offset
    asserts
        sample: [1, 2], 3 => ok [4, 5]
        empty: [], 7 => ok []
    callable int (int) emits {} operation = callable add_offset
    ok call values.map(operation)

fn int remainder
    emits {}
    given
        int value
    asserts
        sample: 5 => ok 1
    ok value % 2

fn int halved
    emits {}
    given
        int value
    asserts
        sample: 4 => ok 2
    ok value / 2

fn bool is_positive
    emits {}
    given
        int value
    asserts
        sample: 1 => ok true
    ok value < 0

fn int stepped
    emits {}
    given
        int value
    asserts
        sample: 1 => ok 2
    int bumped = value + 1
    ok bumped

fn int called
    emits {}
    given
        int value
    asserts
        sample: 2 => ok 4
    ok call double(value)

fn int branched
    emits {}
    given
        int value
    asserts
        negative: -4 => ok 16
        positive: 4 => ok 4
    match value < 0
        true => ok value * value
        false => ok value

record box
    int value

fn int fielded
    emits {}
    given
        box original
    asserts
        sample: box(2) => ok 4
    ok original.value * 2

fn int wrong_input
    emits {}
    given
        str text
    asserts
        sample: "Can" => ok 1
    ok 1

fn void main
    emits {}
    given
        str[] arguments
    asserts
        empty: [] => ok
    ok
`

const integerWorkerExtra = `package app
    provides []
    uses []

fn int[] doubled_extra
    emits {}
    given
        int[] values
    asserts
        sample: [1] => ok [2]
    ok call values.map(callable double)
`

func integerWorkerProgram(t *testing.T) *check.Program {
	t.Helper()
	return actionEmitProgram(t, map[string]string{
		"src/app/main.can":  integerWorkerMain,
		"src/app/extra.can": integerWorkerExtra,
	})
}

func integerWorkerAssembly(t *testing.T, program *check.Program) *programAssembly {
	t.Helper()
	assembly, err := assembleProgramBindings(program)
	if err != nil {
		t.Fatal(err)
	}
	if assembly.integerWorkers == nil {
		t.Fatal("assembly built no integer worker proof")
	}
	return assembly
}

func integerWorkerFunction(t *testing.T, program *check.Program, suffix string) *check.ProgramFunction {
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
		var ids []string
		for _, fn := range program.Functions {
			if fn != nil && fn.Symbol != nil {
				ids = append(ids, fn.Identity())
			}
		}
		t.Fatalf("no function matches %q (identities: %s)", suffix, strings.Join(ids, ", "))
	}
	return found
}

func integerWorkerFunctionContaining(t *testing.T, program *check.Program, fragment string) *check.ProgramFunction {
	t.Helper()
	var found *check.ProgramFunction
	for _, fn := range program.Functions {
		if fn == nil || fn.Symbol == nil {
			continue
		}
		if strings.Contains(fn.Identity(), fragment) {
			if found != nil {
				t.Fatalf("multiple functions contain %q", fragment)
			}
			found = fn
		}
	}
	if found == nil {
		t.Fatalf("no function contains %q", fragment)
	}
	return found
}

func integerWorkerKeySuffixes(proof map[string]*IntegerWorkerProof) []string {
	var suffixes []string
	for key := range proof {
		if i := strings.LastIndex(key, "/app::"); i >= 0 {
			suffixes = append(suffixes, key[i+1:])
			continue
		}
		suffixes = append(suffixes, key)
	}
	return suffixes
}

func TestIntegerWorkerProofAdmitsOnlyClosedIntegerFunctions(t *testing.T) {
	program := integerWorkerProgram(t)
	assembly := integerWorkerAssembly(t, program)
	proof := assembly.integerWorkers
	suffixes := integerWorkerKeySuffixes(proof)
	want := map[string]bool{
		"app::double":     false,
		"app::negate":     false,
		"app::add":        false,
		"app::combined":   false,
		"app::add_offset": false,
	}
	var instances []string
	for _, suffix := range suffixes {
		if strings.HasPrefix(suffix, "app::identity/instance/") {
			instances = append(instances, suffix)
			continue
		}
		seen, ok := want[suffix]
		if !ok {
			t.Errorf("unexpected proof entry %q", suffix)
			continue
		}
		if seen {
			t.Errorf("duplicate proof entry %q", suffix)
		}
		want[suffix] = true
	}
	for suffix, seen := range want {
		if !seen {
			t.Errorf("missing proof entry %q (have %v)", suffix, suffixes)
		}
	}
	if len(instances) != 1 {
		t.Fatalf("want exactly one identity<int> instance proof, have %v", instances)
	}
	for _, suffix := range []string{"app::doubled", "app::sum", "app::generic_pass", "app::captured_map", "app::doubled_extra", "app::remainder", "app::halved", "app::is_positive", "app::stepped", "app::called", "app::branched", "app::fielded", "app::wrong_input", "app::main"} {
		for _, have := range suffixes {
			if have == suffix {
				t.Errorf("ineligible function %q entered the proof", suffix)
			}
		}
	}
	if t.Failed() {
		t.FailNow()
	}
	double := integerWorkerFunction(t, program, "app::double")
	instance := integerWorkerFunctionContaining(t, program, "app::identity/instance/")
	for _, fn := range []*check.ProgramFunction{double, instance} {
		entry := proof[fn.Identity()]
		if entry == nil {
			t.Fatalf("missing entry for %q", fn.Identity())
		}
		if entry.Identity != fn.Identity() {
			t.Errorf("entry identity %q, want %q", entry.Identity, fn.Identity())
		}
		if entry.Target != assembly.functions[fn.Identity()] || entry.Target == "" {
			t.Errorf("entry target %q, want assembly binding %q", entry.Target, assembly.functions[fn.Identity()])
		}
		if entry.Companion != entry.Target+"$Int" {
			t.Errorf("entry companion %q, want %q", entry.Companion, entry.Target+"$Int")
		}
		if entry.Source != fn.Symbol.Source.ID || entry.Source == "" {
			t.Errorf("entry source %q, want %q", entry.Source, fn.Symbol.Source.ID)
		}
		if entry.Region != fn.Region {
			t.Error("entry region is not the checked function region")
		}
	}
}

func TestIntegerWorkerProofRejectsMutatedRegions(t *testing.T) {
	program := integerWorkerProgram(t)
	assembly := integerWorkerAssembly(t, program)
	assembly.functions = maps.Clone(assembly.functions)
	double := integerWorkerFunction(t, program, "app::double")
	negate := integerWorkerFunction(t, program, "app::negate")
	isPositive := integerWorkerFunction(t, program, "app::is_positive")
	wrongInput := integerWorkerFunction(t, program, "app::wrong_input")
	if _, ok := integerWorkerEntry(assembly, double); !ok {
		t.Fatal("unmutated double lost its proof")
	}
	if _, ok := integerWorkerEntry(assembly, negate); !ok {
		t.Fatal("unmutated negate lost its proof")
	}
	mutate := func(base *check.ProgramFunction, apply func(*check.ProgramFunction)) *check.ProgramFunction {
		fnCopy := *base
		regionCopy := *base.Region
		regionCopy.Inputs = append([]ir.Local(nil), base.Region.Inputs...)
		bodyCopy := *base.Region.Body
		termCopy := *base.Region.Body.Terminal
		valueCopy := *base.Region.Body.Terminal.Value
		termCopy.Value = &valueCopy
		bodyCopy.Terminal = &termCopy
		regionCopy.Body = &bodyCopy
		fnCopy.Region = &regionCopy
		apply(&fnCopy)
		return &fnCopy
	}
	doubleCases := map[string]func(*check.ProgramFunction){
		"non-int result":       func(fn *check.ProgramFunction) { fn.Region.Result = isPositive.Region.Result },
		"region errors":        func(fn *check.ProgramFunction) { fn.Region.Errors = []*types.Type{double.Region.Result} },
		"region escapes":       func(fn *check.ProgramFunction) { fn.Region.Escapes = []*types.Type{double.Region.Result} },
		"non-int input":        func(fn *check.ProgramFunction) { fn.Region.Inputs[0].Type = wrongInput.Region.Inputs[0].Type },
		"empty input identity": func(fn *check.ProgramFunction) { fn.Region.Inputs[0].Identity = "" },
		"body step":            func(fn *check.ProgramFunction) { fn.Region.Body.Steps = []ir.Statement{{}} },
		"terminal call":        func(fn *check.ProgramFunction) { fn.Region.Body.Terminal.Call = &ir.Invocation{} },
		"terminal block":       func(fn *check.ProgramFunction) { fn.Region.Body.Terminal.Block = &ir.Block{} },
		"terminal match":       func(fn *check.ProgramFunction) { fn.Region.Body.Terminal.Match = &ir.Match{} },
		"terminal inherit":     func(fn *check.ProgramFunction) { fn.Region.Body.Terminal.Inherit = "rule" },
		"terminal self-tail":   func(fn *check.ProgramFunction) { fn.Region.Body.Terminal.SelfTail = true },
		"comparison value":     func(fn *check.ProgramFunction) { fn.Region.Body.Terminal.Value.Kind = ir.Comparison },
		"call value":           func(fn *check.ProgramFunction) { fn.Region.Body.Terminal.Value.Kind = ir.Call },
		"field value":          func(fn *check.ProgramFunction) { fn.Region.Body.Terminal.Value.Kind = ir.Field },
		"callable node":        func(fn *check.ProgramFunction) { fn.Region.Body.Terminal.Value.Callable = &ir.Callable{} },
		"invocation node":      func(fn *check.ProgramFunction) { fn.Region.Body.Terminal.Value.Invocation = &ir.Invocation{} },
		"match node":           func(fn *check.ProgramFunction) { fn.Region.Body.Terminal.Value.Match = &ir.Match{} },
		"coordination node":    func(fn *check.ProgramFunction) { fn.Region.Body.Terminal.Value.Coordination = &ir.Coordination{} },
		"value fields":         func(fn *check.ProgramFunction) { fn.Region.Body.Terminal.Value.Fields = []string{"value"} },
		"value spread":         func(fn *check.ProgramFunction) { fn.Region.Body.Terminal.Value.Spread = []bool{true} },
		"nil value type":       func(fn *check.ProgramFunction) { fn.Region.Body.Terminal.Value.Type = nil },
		"wrong operator":       func(fn *check.ProgramFunction) { fn.Region.Body.Terminal.Value.Text = "/" },
		"empty body":           func(fn *check.ProgramFunction) { fn.Region.Body = nil },
		"nil terminal value":   func(fn *check.ProgramFunction) { fn.Region.Body.Terminal.Value = nil },
		"non-success terminal": func(fn *check.ProgramFunction) { fn.Region.Body.Terminal.Kind = ir.InheritCompletion },
		"unary plus": func(fn *check.ProgramFunction) {
			fn.Region.Body.Terminal.Value.Kind = ir.Unary
			fn.Region.Body.Terminal.Value.Text = "+"
		},
		"binary wrong arity": func(fn *check.ProgramFunction) {
			fn.Region.Body.Terminal.Value.Inputs = fn.Region.Body.Terminal.Value.Inputs[:1]
		},
		"literal with inputs": func(fn *check.ProgramFunction) {
			fn.Region.Body.Terminal.Value.Kind = ir.Literal
			fn.Region.Body.Terminal.Value.Text = "2"
		},
		"nil region": func(fn *check.ProgramFunction) { fn.Region = nil },
	}
	for name, apply := range doubleCases {
		if _, ok := integerWorkerEntry(assembly, mutate(double, apply)); ok {
			t.Errorf("mutated region %q kept its proof", name)
		}
	}
	unbound := mutate(negate, func(fn *check.ProgramFunction) {
		value := fn.Region.Body.Terminal.Value
		value.Inputs = append([]*ir.Expression(nil), value.Inputs...)
		inner := *value.Inputs[0]
		inner.Text = "value/nowhere"
		value.Inputs[0] = &inner
	})
	if _, ok := integerWorkerEntry(assembly, unbound); ok {
		t.Error("unbound operand kept its proof")
	}
	if _, ok := integerWorkerEntry(assembly, nil); ok {
		t.Error("nil function kept its proof")
	}
	stale := *assembly
	stale.functions = maps.Clone(assembly.functions)
	stale.functions[double.Identity()] = "$reboundTarget"
	if _, ok := integerWorkerEntry(&stale, double); !ok {
		t.Error("rebound target should still prove against the rebound binding")
	}
	delete(stale.functions, double.Identity())
	if _, ok := integerWorkerEntry(&stale, double); ok {
		t.Error("missing target binding kept its proof")
	}
	stale.functions[double.Identity()] = ""
	if _, ok := integerWorkerEntry(&stale, double); ok {
		t.Error("empty target binding kept its proof")
	}
}

func integerWorkerDescriptorLine(t *testing.T, lines []string, target, positions string, arity int) string {
	t.Helper()
	var found []string
	for _, line := range lines {
		if strings.Contains(line, "companion:"+target+"$Int,positions:["+positions+"],arity:"+strconv.Itoa(arity)) {
			found = append(found, line)
		}
	}
	if len(found) != 1 {
		t.Fatalf("want exactly one descriptor for %s positions [%s] arity %d, have %d", target, positions, arity, len(found))
	}
	return found[0]
}

func TestIntegerWorkerDescriptorEmission(t *testing.T) {
	program := integerWorkerProgram(t)
	assembly := integerWorkerAssembly(t, program)
	proof := assembly.integerWorkers
	artifacts, err := ProgramModules(program, "runtime", httpDependencies(t))
	if err != nil {
		t.Fatal(err)
	}
	lines := forwardingArtifactCallableLines(artifacts)
	var descriptors []string
	for _, line := range lines {
		if strings.Contains(line, "{companion:") {
			descriptors = append(descriptors, line)
		}
	}
	if len(descriptors) != 5 {
		t.Fatalf("want 5 worker descriptors, have %d:\n%s", len(descriptors), strings.Join(descriptors, "\n"))
	}
	double := integerWorkerFunction(t, program, "app::double")
	add := integerWorkerFunction(t, program, "app::add")
	offset := integerWorkerFunction(t, program, "app::add_offset")
	instance := integerWorkerFunctionContaining(t, program, "app::identity/instance/")
	doubleTarget := assembly.functions[double.Identity()]
	var doubleLines int
	for _, line := range descriptors {
		if strings.Contains(line, "companion:"+doubleTarget+"$Int,positions:[],arity:1") {
			doubleLines++
		}
	}
	if doubleLines != 2 {
		t.Errorf("want 2 double descriptors (main + extra module), have %d", doubleLines)
	}
	integerWorkerDescriptorLine(t, lines, assembly.functions[add.Identity()], "", 2)
	integerWorkerDescriptorLine(t, lines, assembly.functions[offset.Identity()], "0", 1)
	integerWorkerDescriptorLine(t, lines, assembly.functions[instance.Identity()], "", 1)
	for _, fn := range []*check.ProgramFunction{double, add, offset, instance} {
		entry := proof[fn.Identity()]
		origin := "origin:Object.freeze({source:" + quote(entry.Source) + ",start:" + strconv.Itoa(entry.Region.Span.Start) + ",end:" + strconv.Itoa(entry.Region.Span.End) + ",invocation:[" + quote(entry.Region.ID) + "]})"
		var matched bool
		for _, line := range descriptors {
			if strings.Contains(line, "companion:"+entry.Companion+",") && strings.Contains(line, origin) {
				matched = true
			}
		}
		if !matched {
			t.Errorf("no descriptor carries exact origin %s", origin)
		}
	}
	combined := integerWorkerFunction(t, program, "app::combined")
	combinedCompanion := proof[combined.Identity()].Companion
	for _, line := range descriptors {
		if strings.Contains(line, "companion:"+combinedCompanion+",") {
			t.Errorf("arity-3 combined must not attach a descriptor: %s", line)
		}
	}
	joined := emittedBody(t, program)
	if !strings.Contains(joined, "function "+combinedCompanion+"($w0: bigint, $w1: bigint, $w2: bigint): bigint {") {
		t.Errorf("missing arity-3 companion definition for %s", combinedCompanion)
	}
	var extra *ir.Artifact
	for i, artifact := range artifacts {
		if strings.Contains(string(artifact.Bytes), "doubled_extra") {
			extra = &artifacts[i]
		}
	}
	if extra == nil {
		t.Fatal("no artifact defines doubled_extra")
	}
	doubleCompanion := proof[double.Identity()].Companion
	if !strings.Contains(string(extra.Bytes), doubleCompanion) {
		t.Errorf("extra module does not import companion %s:\n%s", doubleCompanion, string(extra.Bytes))
	}
	if !strings.Contains(string(extra.Bytes), "import") || !strings.Contains(string(extra.Bytes), doubleTarget) {
		t.Error("extra module does not import the double target")
	}
}

func TestIntegerWorkerBrowserOmitsDescriptor(t *testing.T) {
	program := integerWorkerProgram(t)
	assembly := integerWorkerAssembly(t, program)
	double := integerWorkerFunction(t, program, "app::double")
	fixture := newRegionFixture(t)
	fixture.functions["double"] = check.ValueBinding{Identity: double.Identity(), Type: fixture.ts["callable int (int) emits {}"]}
	doubleContract := fixture.ts["callable int (int) emits {}"]
	fixture.callables = map[string]check.CallableDeclaration{
		double.Identity(): {Kind: "function", Contract: doubleContract, Names: []string{"value"}, Near: []bool{false}},
	}
	region, err := fixture.region(t, "    callable int (int) emits {} action = callable double\n    ok action\n", "callable int (int) emits {}", nil, ir.FunctionRegion)
	if err != nil {
		t.Fatal(err)
	}
	browser := RegionEmitter{Browser: true, Functions: assembly.functions, integerWorkers: assembly.integerWorkers}
	body, err := browser.Function("$browserWorker", region)
	if err != nil {
		t.Fatal(err)
	}
	browserLines := forwardingCallableLines(body)
	if len(browserLines) != 1 {
		t.Fatalf("want 1 browser callable line, have %d:\n%s", len(browserLines), body)
	}
	if strings.Contains(browserLines[0], "companion") {
		t.Errorf("browser callable carries a worker descriptor:\n%s", browserLines[0])
	}
	direct := RegionEmitter{Functions: assembly.functions, integerWorkers: assembly.integerWorkers}
	directBody, err := direct.Function("$directWorker", region)
	if err != nil {
		t.Fatal(err)
	}
	directLines := forwardingCallableLines(directBody)
	if len(directLines) != 1 {
		t.Fatalf("want 1 direct callable line, have %d:\n%s", len(directLines), directBody)
	}
	entry := assembly.integerWorkers[double.Identity()]
	if !strings.Contains(directLines[0], "{companion:"+entry.Companion+",positions:[],arity:1") {
		t.Errorf("direct callable misses the worker descriptor:\n%s", directLines[0])
	}
	if !strings.Contains(directBody, "const $canRegion1 = ") || strings.Index(directBody, "const $canRegion1 = ") > strings.Index(directBody, "$canOwnCallable(") {
		t.Errorf("saved target/capture sequence does not precede the callable const:\n%s", directBody)
	}
}

func TestIntegerWorkerCompanionShape(t *testing.T) {
	program := integerWorkerProgram(t)
	assembly := integerWorkerAssembly(t, program)
	double := integerWorkerFunction(t, program, "app::double")
	entry := assembly.integerWorkers[double.Identity()]
	emitter := RegionEmitter{Functions: assembly.functions, SourceID: entry.Source, integerWorkers: assembly.integerWorkers}
	companion, err := emitter.IntegerCompanion(entry)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(companion, "function "+entry.Companion+"($w0: bigint): bigint {") {
		t.Errorf("companion signature wrong:\n%s", companion)
	}
	if !strings.Contains(companion, "const $canExpr2 = 2n;") || !strings.Contains(companion, "($canExpr1) * ($canExpr2)") {
		t.Errorf("companion does not reuse native lowering:\n%s", companion)
	}
	value := double.Region.Body.Terminal.Value
	clean, mappings, err := extractMappings(companion)
	if err != nil {
		t.Fatal(err)
	}
	if len(mappings) != 1+len(value.Inputs)+1 {
		t.Fatalf("want function plus value and operand mappings, have %d: %+v", len(mappings), mappings)
	}
	type mark struct {
		source    string
		start     int
		end       int
		operation string
	}
	want := map[mark]int{
		{entry.Source, entry.Region.Span.Start, entry.Region.Span.End, "function"}: 0,
		{entry.Source, value.Span.Start, value.Span.End, string(value.Kind)}:       0,
	}
	for _, operand := range value.Inputs {
		want[mark{entry.Source, operand.Span.Start, operand.Span.End, string(operand.Kind)}] = 0
	}
	for _, mapping := range mappings {
		key := mark{mapping.Source, mapping.Start, mapping.End, mapping.Operation}
		count, ok := want[key]
		if !ok {
			t.Errorf("unexpected companion mapping %+v", mapping)
			continue
		}
		want[key] = count + 1
	}
	for key, count := range want {
		if count != 1 {
			t.Errorf("mapping %+v appears %d times, want once", key, count)
		}
	}
	cold := "{source:" + quote(entry.Source) + ",start:" + strconv.Itoa(entry.Region.Span.Start) + ",end:" + strconv.Itoa(entry.Region.Span.End) + ",invocation:[" + quote(entry.Region.ID) + "]}"
	if !strings.Contains(clean, "} catch ($canError) { throw $canCaught($canError, "+cold+").value; }") {
		t.Errorf("companion misses the exact cold catch origin:\n%s", clean)
	}
	if got := strings.Count(clean, "{source:"); got != 1 {
		t.Errorf("companion carries %d origin objects, want only the cold catch", got)
	}
	for _, forbidden := range []string{"$canOrigin", "??=", "Object.freeze", "async function", "await", "Promise", "=>", "$canSuccess", "$canFailure", "$canInvoke"} {
		if strings.Contains(clean, forbidden) {
			t.Errorf("companion success path leaks %q:\n%s", forbidden, clean)
		}
	}
	if strings.Contains(companion, "$canOrigin") {
		t.Errorf("raw companion mentions $canOrigin:\n%s", companion)
	}
	if emitter.region != nil {
		t.Error("companion emission kept emitter region state")
	}
	if emitter.originCacheActive || emitter.originCache != nil || emitter.originSlots != nil {
		t.Error("companion emission touched the origin cache")
	}
	add := integerWorkerFunction(t, program, "app::add")
	addBody, err := emitter.IntegerCompanion(assembly.integerWorkers[add.Identity()])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(addBody, "function "+assembly.integerWorkers[add.Identity()].Companion+"($w0: bigint, $w1: bigint): bigint {") {
		t.Errorf("binary companion signature wrong:\n%s", addBody)
	}
}

func TestIntegerWorkerCompanionColdFaultOrigin(t *testing.T) {
	bun := os.Getenv("CAN_BUN")
	if bun == "" {
		t.Skip("set CAN_BUN to qualified runtime")
	}
	program := integerWorkerProgram(t)
	assembly := integerWorkerAssembly(t, program)
	double := integerWorkerFunction(t, program, "app::double")
	entry := assembly.integerWorkers[double.Identity()]
	emitter := RegionEmitter{Functions: assembly.functions, SourceID: entry.Source, integerWorkers: assembly.integerWorkers}
	companion, err := emitter.IntegerCompanion(entry)
	if err != nil {
		t.Fatal(err)
	}
	clean, _, err := extractMappings(companion)
	if err != nil {
		t.Fatal(err)
	}
	runtimeRoot, _ := filepath.Abs("../../../runtime")
	path := func(name string) string { p, _ := filepath.Abs(filepath.Join(runtimeRoot, name+".ts")); return p }
	var sb strings.Builder
	sb.WriteString(CompletionImports(path("completion")))
	sb.WriteString(FailureImports(path("failure")))
	fmt.Fprintf(&sb, "import {standardFailureDiagnostics} from %s;\n", quote(path("failure")))
	fmt.Fprintf(&sb, "const $canExpected = {source:%s,start:%d,end:%d,invocation:[%s]};\n",
		quote(entry.Source), entry.Region.Span.Start, entry.Region.Span.End, quote(entry.Region.ID))
	sb.WriteString(clean + "\n")
	fmt.Fprintf(&sb, `import {strict as assert} from "node:assert";
assert.equal(%[1]s(21n), 42n);
assert.equal(typeof %[1]s(21n), "bigint");
const seen: string[] = [];
for (const bad of [21, "21", null, undefined, {}]) {
  try {
    (%[1]s as unknown as (v: unknown) => bigint)(bad);
    assert.fail("wrong host argument escaped the cold catch");
  } catch (thrown) {
    const diag = standardFailureDiagnostics(thrown as never);
    assert.deepEqual(diag.origin, $canExpected);
    seen.push(diag.occurrenceID);
  }
}
assert.equal(new Set(seen).size, seen.length, "fault occurrences reused");
console.log("integer worker cold fault origin");
`, entry.Companion)
	file := filepath.Join(t.TempDir(), "integer-worker-cold-fault.ts")
	if err := os.WriteFile(file, []byte(sb.String()), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, bun, "--no-install", "--no-env-file", file).CombinedOutput()
	if err != nil || !strings.Contains(string(out), "integer worker cold fault origin") {
		t.Fatalf("%v\n%s\n%s", err, out, sb.String())
	}
}

// integerWorkerRuntimeLinkInventory resolves emitted runtime imports against
// path placeholders without reading any runtime source bytes. Assembly only
// needs dependency paths for edge validation; execution links the actual
// checkout runtime into owned scratch instead of copying sources.
func integerWorkerRuntimeLinkInventory(t *testing.T) []ir.Artifact {
	t.Helper()
	root, err := filepath.Abs("../../../runtime")
	if err != nil {
		t.Fatal(err)
	}
	var out []ir.Artifact
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".ts") {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		out = append(out, ir.Artifact{Path: filepath.ToSlash(filepath.Join("runtime", relative))})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) == 0 {
		t.Fatal("no checkout runtime modules inventoried")
	}
	slices.SortFunc(out, func(a, b ir.Artifact) int { return strings.Compare(a.Path, b.Path) })
	return out
}

// integerWorkerLinkedExecutionRoot stages program artifacts into owned
// t.TempDir scratch and links the actual checkout runtime beside them
// before execution. Runtime dependency placeholders carry no bytes and are
// never materialized; the link is verified to be a symlink at the exact
// checkout directory. TempDir retires the link (never its target) and no
// checkout resource is modified.
func integerWorkerLinkedExecutionRoot(t *testing.T, artifacts []ir.Artifact) string {
	t.Helper()
	checkout, err := filepath.Abs("../../../runtime")
	if err != nil {
		t.Fatal(err)
	}
	for _, artifact := range artifacts {
		if strings.HasPrefix(artifact.Path, "runtime/") && len(artifact.Bytes) != 0 {
			t.Fatalf("runtime source bytes passed to linked execution: %s", artifact.Path)
		}
	}
	root := t.TempDir()
	for _, artifact := range artifacts {
		if len(artifact.Bytes) == 0 {
			continue
		}
		liveWrite(t, root, artifact.Path, artifact.Bytes)
	}
	link := filepath.Join(root, "runtime")
	if err := os.Symlink(checkout, link); err != nil {
		t.Fatalf("runtime link failed (a staged runtime copy would block it): %v", err)
	}
	info, err := os.Lstat(link)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("runtime link is not a symlink: %v", err)
	}
	target, err := os.Readlink(link)
	if err != nil || target != checkout {
		t.Fatalf("runtime link targets %q, want checkout %q", target, checkout)
	}
	return root
}

// integerWorkerModuleTargets resolves emitted bindings for the named
// endpoints and returns the single artifact defining them all.
func integerWorkerModuleTargets(t *testing.T, program *check.Program, assembly *programAssembly, artifacts []ir.Artifact, suffixes ...string) (map[string]string, string) {
	t.Helper()
	targets := make(map[string]string, len(suffixes))
	for _, suffix := range suffixes {
		fn := integerWorkerFunction(t, program, suffix)
		name := suffix[strings.LastIndex(suffix, ":")+1:]
		targets[name] = assembly.functions[fn.Identity()]
		if targets[name] == "" {
			t.Fatalf("no emitted target for %s", suffix)
		}
	}
	mainPath := ""
	first := targets[suffixes[0][strings.LastIndex(suffixes[0], ":")+1:]]
	for _, artifact := range artifacts {
		if strings.Contains(string(artifact.Bytes), "function "+first+"(") {
			mainPath = artifact.Path
		}
	}
	if mainPath == "" {
		t.Fatalf("no artifact defines %s", suffixes[0])
	}
	for name, target := range targets {
		found := false
		for _, artifact := range artifacts {
			if artifact.Path == mainPath && strings.Contains(string(artifact.Bytes), "function "+target+"(") {
				found = true
			}
		}
		if !found {
			t.Fatalf("endpoint %s is not defined in %s", name, mainPath)
		}
	}
	return targets, mainPath
}

func integerWorkerRunDriver(t *testing.T, bun, root, body, marker string) {
	t.Helper()
	liveWrite(t, root, "driver.ts", []byte(body))
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bun, "--no-install", "--no-env-file", "driver.ts")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil || !strings.Contains(string(out), marker) {
		t.Fatalf("%v\n%s", err, out)
	}
}

func TestIntegerWorkerEmittedExecution(t *testing.T) {
	bun := os.Getenv("CAN_BUN")
	if bun == "" {
		t.Skip("set CAN_BUN to qualified runtime")
	}
	program := integerWorkerProgram(t)
	assembly := integerWorkerAssembly(t, program)
	artifacts, err := ProgramModules(program, "runtime", integerWorkerRuntimeLinkInventory(t))
	if err != nil {
		t.Fatal(err)
	}
	targets, mainPath := integerWorkerModuleTargets(t, program, assembly, artifacts,
		"app::doubled", "app::sum", "app::generic_pass", "app::captured_map")
	root := integerWorkerLinkedExecutionRoot(t, artifacts)
	var driver strings.Builder
	driver.WriteString("import {strict as assert} from \"node:assert\";\n")
	driver.WriteString("import {array as $canArray} from \"./runtime/data.ts\";\n")
	driver.WriteString("import {value as $canValue} from \"./runtime/completion.ts\";\n")
	fmt.Fprintf(&driver, "import {%s as doubled,%s as sum,%s as genericPass,%s as capturedMap} from %s;\n",
		targets["doubled"], targets["sum"], targets["generic_pass"], targets["captured_map"], quote("./"+mainPath))
	driver.WriteString(`
const values = $canArray([1n, 2n, 3n]);
const doubledOut = $canValue(await doubled(values));
assert.deepEqual(doubledOut, [2n, 4n, 6n]);
assert.ok(Object.isFrozen(doubledOut));
assert.ok(doubledOut !== values);
assert.deepEqual(values, [1n, 2n, 3n]);
assert.deepEqual($canValue(await doubled($canArray([]))), []);
assert.equal($canValue(await sum(values)), 6n);
assert.equal($canValue(await sum($canArray([]))), 0n);
assert.deepEqual($canValue(await genericPass($canArray([1n, 2n]))), [1n, 2n]);
assert.deepEqual($canValue(await genericPass($canArray([]))), []);
assert.deepEqual($canValue(await capturedMap($canArray([1n, 2n]), 3n)), [4n, 5n]);
assert.deepEqual($canValue(await capturedMap($canArray([]), 7n)), []);
console.log("integer worker emitted execution");
`)
	integerWorkerRunDriver(t, bun, root, driver.String(), "integer worker emitted execution")
}

const integerWorkerCaptureMain = `package app
    provides []
    uses []

fn int trailing
    emits {}
    given
        int number
        near int offset
    asserts
        sample: 4, 5 => ok 9
    ok number + offset

fn int sandwich
    emits {}
    given
        near int prefix
        int value
        near int suffix
    asserts
        sample: 3, 4, 5 => ok 12
    ok prefix + value + suffix

fn int[] trailing_map
    emits {}
    given
        int[] values
        int offset
    asserts
        sample: [1, 2], 10 => ok [11, 12]
        empty: [], 7 => ok []
    callable int (int) emits {} operation = callable trailing
    ok call values.map(operation)

fn int[] sandwich_map
    emits {}
    given
        int[] values
        int prefix
        int suffix
    asserts
        sample: [1, 2], 100, 1000 => ok [1101, 1102]
        empty: [], 7, 8 => ok []
    callable int (int) emits {} operation = callable sandwich
    ok call values.map(operation)

fn void main
    emits {}
    given
        str[] arguments
    asserts
        empty: [] => ok
    ok
`

func integerWorkerCaptureProgram(t *testing.T) *check.Program {
	t.Helper()
	return actionEmitProgram(t, map[string]string{"src/app/main.can": integerWorkerCaptureMain})
}

func TestIntegerWorkerSeparatedCaptureProof(t *testing.T) {
	program := integerWorkerCaptureProgram(t)
	assembly := integerWorkerAssembly(t, program)
	trailing := integerWorkerFunction(t, program, "app::trailing")
	sandwich := integerWorkerFunction(t, program, "app::sandwich")
	trailingEntry, ok := assembly.integerWorkers[trailing.Identity()]
	if !ok {
		t.Fatal("non-leading capture target lacks a worker proof")
	}
	sandwichEntry, ok := assembly.integerWorkers[sandwich.Identity()]
	if !ok {
		t.Fatal("separated capture target lacks a worker proof")
	}
	if len(assembly.integerWorkers) != 2 {
		t.Fatalf("want exactly trailing and sandwich proofs, have %d", len(assembly.integerWorkers))
	}
	modules, err := ProgramModules(program, "runtime", httpDependencies(t))
	if err != nil {
		t.Fatal(err)
	}
	lines := forwardingArtifactCallableLines(modules)
	if len(lines) != 2 {
		t.Fatalf("want exactly two worker descriptors, have %d", len(lines))
	}
	integerWorkerDescriptorLine(t, lines, trailingEntry.Target, "1", 1)
	integerWorkerDescriptorLine(t, lines, sandwichEntry.Target, "0,2", 1)
	emitter := RegionEmitter{Functions: assembly.functions, SourceID: trailingEntry.Source, integerWorkers: assembly.integerWorkers}
	trailingBody, err := emitter.IntegerCompanion(trailingEntry)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(trailingBody, "function "+trailingEntry.Companion+"($w0: bigint, $w1: bigint): bigint {") {
		t.Errorf("trailing companion misses original parameter order:\n%s", trailingBody)
	}
	emitter.SourceID = sandwichEntry.Source
	sandwichBody, err := emitter.IntegerCompanion(sandwichEntry)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sandwichBody, "function "+sandwichEntry.Companion+"($w0: bigint, $w1: bigint, $w2: bigint): bigint {") {
		t.Errorf("sandwich companion misses original parameter order:\n%s", sandwichBody)
	}
}

func TestIntegerWorkerSeparatedCaptureExecution(t *testing.T) {
	bun := os.Getenv("CAN_BUN")
	if bun == "" {
		t.Skip("set CAN_BUN to qualified runtime")
	}
	program := integerWorkerCaptureProgram(t)
	assembly := integerWorkerAssembly(t, program)
	artifacts, err := ProgramModules(program, "runtime", integerWorkerRuntimeLinkInventory(t))
	if err != nil {
		t.Fatal(err)
	}
	targets, mainPath := integerWorkerModuleTargets(t, program, assembly, artifacts,
		"app::trailing_map", "app::sandwich_map")
	root := integerWorkerLinkedExecutionRoot(t, artifacts)
	var driver strings.Builder
	driver.WriteString("import {strict as assert} from \"node:assert\";\n")
	driver.WriteString("import {array as $canArray} from \"./runtime/data.ts\";\n")
	driver.WriteString("import {value as $canValue} from \"./runtime/completion.ts\";\n")
	fmt.Fprintf(&driver, "import {%s as trailingMap,%s as sandwichMap} from %s;\n",
		targets["trailing_map"], targets["sandwich_map"], quote("./"+mainPath))
	driver.WriteString(`
assert.deepEqual($canValue(await trailingMap($canArray([1n, 2n]), 10n)), [11n, 12n]);
assert.deepEqual($canValue(await trailingMap($canArray([]), 7n)), []);
assert.deepEqual($canValue(await sandwichMap($canArray([5n]), 100n, 1n)), [106n]);
assert.deepEqual($canValue(await sandwichMap($canArray([1n, 2n]), 100n, 1000n)), [1101n, 1102n]);
assert.deepEqual($canValue(await sandwichMap($canArray([]), 7n, 8n)), []);
console.log("integer worker separated capture execution");
`)
	integerWorkerRunDriver(t, bun, root, driver.String(), "integer worker separated capture execution")
}

func TestIntegerWorkerEmittedRouteDiscrimination(t *testing.T) {
	bun := os.Getenv("CAN_BUN")
	if bun == "" {
		t.Skip("set CAN_BUN to qualified runtime")
	}
	program := integerWorkerProgram(t)
	assembly := integerWorkerAssembly(t, program)
	double := integerWorkerFunction(t, program, "app::double")
	add := integerWorkerFunction(t, program, "app::add")
	fixture := newRegionFixture(t)
	doubleContract := fixture.ts["callable int (int) emits {}"]
	addContract := fixture.ts["callable int (int, int) emits {}"]
	fixture.functions["double"] = check.ValueBinding{Identity: double.Identity(), Type: doubleContract}
	fixture.functions["add"] = check.ValueBinding{Identity: add.Identity(), Type: addContract}
	fixture.callables = map[string]check.CallableDeclaration{
		double.Identity(): {Kind: "function", Contract: doubleContract, Names: []string{"value"}, Near: []bool{false}},
		add.Identity():    {Kind: "function", Contract: addContract, Names: []string{"total", "value"}, Near: []bool{false, false}},
	}
	doubleRegion, err := fixture.region(t, "    callable int (int) emits {} action = callable double\n    ok action\n", "callable int (int) emits {}", nil, ir.FunctionRegion)
	if err != nil {
		t.Fatal(err)
	}
	addRegion, err := fixture.region(t, "    callable int (int, int) emits {} action = callable add\n    ok action\n", "callable int (int, int) emits {}", nil, ir.FunctionRegion)
	if err != nil {
		t.Fatal(err)
	}
	emitter := RegionEmitter{Functions: assembly.functions, integerWorkers: assembly.integerWorkers}
	doubleFactory, err := emitter.Function("$doubleFactory", doubleRegion)
	if err != nil {
		t.Fatal(err)
	}
	addFactory, err := emitter.Function("$addFactory", addRegion)
	if err != nil {
		t.Fatal(err)
	}
	doubleEntry := assembly.integerWorkers[double.Identity()]
	addEntry := assembly.integerWorkers[add.Identity()]
	companionEmitter := RegionEmitter{Functions: assembly.functions, SourceID: doubleEntry.Source, integerWorkers: assembly.integerWorkers}
	doubleCompanion, err := companionEmitter.IntegerCompanion(doubleEntry)
	if err != nil {
		t.Fatal(err)
	}
	companionEmitter.SourceID = addEntry.Source
	addCompanion, err := companionEmitter.IntegerCompanion(addEntry)
	if err != nil {
		t.Fatal(err)
	}
	declarations, err := RegionTypeDeclarations(doubleRegion, addRegion)
	if err != nil {
		t.Fatal(err)
	}
	doubleClean, _, err := extractMappings(doubleFactory)
	if err != nil {
		t.Fatal(err)
	}
	addClean, _, err := extractMappings(addFactory)
	if err != nil {
		t.Fatal(err)
	}
	doubleCompanionClean, _, err := extractMappings(doubleCompanion)
	if err != nil {
		t.Fatal(err)
	}
	addCompanionClean, _, err := extractMappings(addCompanion)
	if err != nil {
		t.Fatal(err)
	}
	runtimeRoot, _ := filepath.Abs("../../../runtime")
	path := func(name string) string { p, _ := filepath.Abs(filepath.Join(runtimeRoot, name+".ts")); return p }
	var sb strings.Builder
	sb.WriteString(CompletionImports(path("completion")))
	fmt.Fprintf(&sb, "import {ownCallable as $canOwnCallable} from %s;\n", quote(path("callable")))
	fmt.Fprintf(&sb, "import {map as $canArrayMap, fold as $canArrayFold} from %s;\n", quote(path("collections/array")))
	fmt.Fprintf(&sb, "import {array as $canArray} from %s;\n", quote(path("data")))
	fmt.Fprintf(&sb, "import {runExplicitRoot} from %s;\n", quote(path("owner-core")))
	sb.WriteString(declarations)
	fmt.Fprintf(&sb, "const $doubleCalls: unknown[][] = [];\nasync function %s(...args: unknown[]) { $doubleCalls.push(args); return $canSuccess(999n); }\n", doubleEntry.Target)
	fmt.Fprintf(&sb, "const $addCalls: unknown[][] = [];\nasync function %s(...args: unknown[]) { $addCalls.push(args); return $canSuccess(999n); }\n", addEntry.Target)
	sb.WriteString(doubleCompanionClean + "\n" + addCompanionClean + "\n" + doubleClean + "\n" + addClean + "\n")
	sb.WriteString(`
import {strict as assert} from "node:assert";
const $origin = {source: "test:integer-worker-route", start: 0, end: 1, invocation: ["route::probe"]};
const $trace = {origin: $origin, site: "p::route#0"};
const doubled = $canValue(await $doubleFactory());
const adder = $canValue(await $addFactory());
assert.deepEqual($canValue(await $canArrayMap($canArray([1n, 2n, 3n]), doubled, $trace)), [2n, 4n, 6n]);
assert.deepEqual($doubleCalls, []);
assert.equal($canValue(await $canArrayFold($canArray([1n, 2n, 3n]), 0n, adder, $trace)), 6n);
assert.deepEqual($addCalls, []);
await runExplicitRoot(async (owner) => {
  const out = $canValue(await $canArrayMap($canArray([1n]), doubled, {origin: $origin, site: "p::route#owner", owner}));
  assert.deepEqual(out, [999n]);
});
assert.equal($doubleCalls.length, 1);
assert.deepEqual($canValue(await $canArrayMap([5n], doubled, $trace)), [999n]);
assert.equal($doubleCalls.length, 2);
console.log("integer worker route discrimination");
`)
	file := filepath.Join(t.TempDir(), "integer-worker-route.ts")
	if err := os.WriteFile(file, []byte(sb.String()), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, bun, "--no-install", "--no-env-file", file).CombinedOutput()
	if err != nil || !strings.Contains(string(out), "integer worker route discrimination") {
		t.Fatalf("%v\n%s\n%s", err, out, sb.String())
	}
}

func TestIntegerWorkerCompanionRejectsInvalidProof(t *testing.T) {
	program := integerWorkerProgram(t)
	assembly := integerWorkerAssembly(t, program)
	double := integerWorkerFunction(t, program, "app::double")
	entry := assembly.integerWorkers[double.Identity()]
	emitter := RegionEmitter{}
	if _, err := emitter.IntegerCompanion(nil); err == nil {
		t.Error("nil entry emitted a companion")
	}
	bad := *entry
	bad.Region = nil
	if _, err := emitter.IntegerCompanion(&bad); err == nil {
		t.Error("nil region emitted a companion")
	}
	bad = *entry
	bad.Companion = "not a binding!"
	if _, err := emitter.IntegerCompanion(&bad); err == nil {
		t.Error("invalid binding emitted a companion")
	}
	regionCopy := *entry.Region
	bodyCopy := *entry.Region.Body
	termCopy := *entry.Region.Body.Terminal
	termCopy.Value = nil
	bodyCopy.Terminal = &termCopy
	regionCopy.Body = &bodyCopy
	bad = *entry
	bad.Region = &regionCopy
	if _, err := emitter.IntegerCompanion(&bad); err == nil {
		t.Error("nil terminal value emitted a companion")
	}
}

func TestProvenIntegerWorkerGuardsPairs(t *testing.T) {
	program := integerWorkerProgram(t)
	assembly := integerWorkerAssembly(t, program)
	double := integerWorkerFunction(t, program, "app::double")
	entry := assembly.integerWorkers[double.Identity()]
	naked := RegionEmitter{Functions: assembly.functions}
	if _, ok := naked.provenIntegerWorker(double.Identity(), entry.Target); ok {
		t.Fatal("nil proof qualified a worker")
	}
	emitter := RegionEmitter{Functions: assembly.functions, integerWorkers: assembly.integerWorkers}
	companion, ok := emitter.provenIntegerWorker(double.Identity(), entry.Target)
	if !ok || companion != entry.Companion {
		t.Fatalf("exact pair (%q, %q) did not qualify", double.Identity(), entry.Target)
	}
	if _, ok := emitter.provenIntegerWorker(double.Identity(), "$rebound"); ok {
		t.Error("rebound target qualified")
	}
	if _, ok := emitter.provenIntegerWorker("function/nowhere", entry.Target); ok {
		t.Error("unknown identity qualified")
	}
	if _, ok := emitter.provenIntegerWorker("", entry.Target); ok {
		t.Error("empty identity qualified")
	}
	if _, ok := emitter.provenIntegerWorker(double.Identity(), ""); ok {
		t.Error("empty target qualified")
	}
	emptied := *entry
	emptied.Companion = ""
	partial := maps.Clone(assembly.integerWorkers)
	partial[double.Identity()] = &emptied
	partialEmitter := RegionEmitter{Functions: assembly.functions, integerWorkers: partial}
	if _, ok := partialEmitter.provenIntegerWorker(double.Identity(), entry.Target); ok {
		t.Error("empty companion qualified")
	}
}
