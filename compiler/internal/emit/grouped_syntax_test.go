package emit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
)

// Each grouped root below enters the same handler body. Its upstream call
// receives a different transitive fixture, while the handler's own grouped
// when row must remain separately selectable by both root identities.
//
// Expected execution sequences (no lane-A sequence was published, so the
// execution tests below record the contract directly):
//   - wrapped/first and wrapped/second return the exact fixture-supplied
//     error across bare forwarding; wrapped/sample passes success through.
//   - probe/first and probe/second each enter the one shared grouped handler
//     through a different upstream error, then consume their own row of the
//     shared nested marker() fixture table; probe/plain passes success.
//   - chain_probe/first and chain_probe/second recover through the grouped
//     chain handler; chain_probe/third passes success through.
//   - settled_group/sample recovers each participant through grouped
//     per-participant arms; race_group/sample recovers the winning failure
//     through the grouped shared arm. Both fail_first/sample and
//     fail_second/sample also prove bare-forwarded payloads.
//   - drain/rep visits one fixture table twice (outer call, then the
//     recursive inner call) consuming 7 then 8 in FIFO order for 78.
//   - revisit/both enters one grouped handler twice through different
//     errors (checks::failed, then codec::invalid_data) and consumes the
//     one nested marker() table in FIFO order for 78; separate per-error
//     tables would compute 77 instead.
//   - picky/flush must fail with "unused fixture" (one visit, two rows);
//     hungry/starved must fail with "missing fixture" (two visits, one row).
//   - fetch/customer and fetch/partner each activate their own scenario row
//     of one grouped scenario when row; fetch/guest and fetch/visitor share
//     one scenario link through grouped roots with independent queues.
//   - tpl_consumer/tpl_first and tpl_consumer/tpl_second each consume their
//     own independent expansion of one grouped template use.
const groupedSyntaxSource = `package app
    provides []
    uses [checks, codec, text]
scenario alpha
scenario beta
fn int source
    emits {checks::failed, codec::invalid_data}
    asserts
        sample: => ok 0
    ok 0
fn int wrapped
    emits {checks::failed, codec::invalid_data}
    asserts
        sample: => ok 0
        first: => checks::failed{"first payload"}
        second: => codec::invalid_data{"/second", "second payload"}
    match call source()
        when
            first: => checks::failed{"first payload"}
            second: => codec::invalid_data{"/second", "second payload"}
        checks::failed | codec::invalid_data
        ok int value => ok value
fn int marker
    emits {}
    asserts
        sample: => ok 9
    ok 9
fn int probe
    emits {}
    asserts
        first | second: => ok 7
        plain: => ok 0
    match call wrapped()
        checks::failed | codec::invalid_data => match call marker()
            when
                first | second: => ok 7
            ok int value => ok value
        ok int value => ok value
fn int chain_source
    emits {checks::failed, codec::invalid_data}
    given
        int choice
    asserts
        first: 0 => checks::failed{"chain first"}
        second: 1 => codec::invalid_data{"/chain", "chain second"}
    match choice
        0 => checks::failed{"chain first"}
        1 => codec::invalid_data{"/chain", "chain second"}
        _ => ok choice
fn int chain_probe
    emits {}
    given
        int choice
    asserts
        first: 0 => ok 7
        second: 1 => ok 7
        third: 2 => ok 2
    match chain
        call chain_source(choice) as int value
        checks::failed | codec::invalid_data => ok 7
        ok => ok value
fn int fail_first
    emits {checks::failed, codec::invalid_data}
    asserts
        sample: => checks::failed{"coord first"}
    match call source()
        when
            sample: => checks::failed{"coord first"}
        checks::failed | codec::invalid_data
        ok int value => ok value
fn int fail_second
    emits {checks::failed, codec::invalid_data}
    asserts
        sample: => codec::invalid_data{"/coord", "coord second"}
    match call source()
        when
            sample: => codec::invalid_data{"/coord", "coord second"}
        checks::failed | codec::invalid_data
        ok int value => ok value
fn int[] settled_group
    emits {}
    asserts
        sample: => ok [7, 7]
    int[] result = match call concurrent with error
        fail_first()
            checks::failed | codec::invalid_data => ok 7
            ok int value => ok value
        fail_second()
            checks::failed | codec::invalid_data => ok 7
            ok int value => ok value
    ok result
fn int race_group
    emits {}
    asserts
        sample: => ok 7
    int result = match call race with error
        fail_first()
        fail_second()
        checks::failed | codec::invalid_data => ok 7
        ok int value => ok value
    ok result
fn int double
    emits {}
    given
        int value
    asserts
        sample: 2 => ok 4
    ok value + value
fixture fixed for double
    cases
        2 => ok 99
fn int tpl_consumer
    emits {}
    asserts
        tpl_first: => ok 99
        tpl_second: => ok 99
    match call double(2)
        when
            tpl_first | tpl_second: use fixed()
        ok int got => ok got
fn int drain
    emits {}
    given
        int depth
    asserts
        rep: 1 => ok 78
    match call marker()
        when
            rep: => ok 7
            rep: => ok 8
        ok int value => match depth
            0 => ok value
            _ => do
                int rest = call drain(depth - 1)
                ok (value * 10) + rest
fn int revisit
    emits {}
    given
        int depth
    asserts
        both: 1 => ok 78
    match call wrapped()
        when
            both: => checks::failed{"revisit first"}
            both: => codec::invalid_data{"/revisit", "revisit second"}
        checks::failed | codec::invalid_data => match call marker()
            when
                both: => ok 7
                both: => ok 8
            ok int value => match depth
                0 => ok value
                _ => do
                    int rest = call revisit(depth - 1)
                    ok (value * 10) + rest
        ok int value => ok value
fn int hungry
    emits {}
    given
        int depth
    asserts
        starved: 1 => ok 0
    match call double(2)
        when
            starved: 2 => ok 99
        ok int value => match depth
            0 => ok value
            _ => do
                int rest = call hungry(depth - 1)
                ok (value * 10) + rest
fn int picky
    emits {}
    asserts
        flush: => ok 99
    match call double(2)
        when
            flush: 2 => ok 99
            flush: 2 => ok 100
        ok int value => ok value
fn str fetch
    emits {}
    asserts
        customer: => ok "served" link alpha
        partner: => ok "served" link beta
        guest | visitor: => ok "served" link alpha
    match call text::from_int(7)
        when
            scenario alpha | beta: 7 => ok "served"
        ok str result => ok result
fn void main
    emits {}
    given
        str[] arguments
    asserts
        empty: [] => ok
    ok
`

func groupedSyntaxProgram(t *testing.T) *check.Program {
	t.Helper()
	return forwardingProgram(t, map[string]string{"src/app/main.can": groupedSyntaxSource})
}

func groupedFunction(t *testing.T, program *check.Program, name string) *check.ProgramFunction {
	t.Helper()
	for _, fn := range program.Functions {
		if fn.Symbol.Name == name {
			return fn
		}
	}
	t.Fatalf("missing %s", name)
	return nil
}

func groupedArms(t *testing.T, fn *check.ProgramFunction) []ir.Arm {
	t.Helper()
	if fn.Region == nil || fn.Region.Body == nil || fn.Region.Body.Terminal == nil || fn.Region.Body.Terminal.Match == nil {
		t.Fatalf("%s lost terminal match", fn.Symbol.Name)
	}
	return fn.Region.Body.Terminal.Match.Arms
}

func TestGroupedErrorBodiesAndForwardingEmission(t *testing.T) {
	program := groupedSyntaxProgram(t)
	probe := groupedArms(t, groupedFunction(t, program, "probe"))
	if len(probe) != 3 || probe[0].Outcome != "domain" || probe[1].Outcome != "domain" || probe[0].Body == nil || probe[0].Body != probe[1].Body {
		t.Fatalf("grouped handler was not checked once and shared: %+v", probe)
	}
	if probe[0].Error.Identity() == probe[1].Error.Identity() {
		t.Fatal("grouped handler lost its two distinct domain identities")
	}
	shared := probe[0].Body.Match
	if shared == nil || shared.Call == nil || len(shared.Call.Steps) != 1 || shared.Call.Steps[0].Fixtures == nil {
		t.Fatal("shared grouped handler lost its nested lexical fixture table")
	}
	rows := shared.Call.Steps[0].Fixtures.Rows
	if len(rows) != 2 || rows[0].Selector != "first" || rows[1].Selector != "second" {
		t.Fatalf("grouped when row did not expand to independent selectors: %+v", rows)
	}
	for _, arm := range probe[:2] {
		if arm.Body.Match.Call.Steps[0].Fixtures != shared.Call.Steps[0].Fixtures {
			t.Fatal("an error alternative received a different lexical fixture queue")
		}
	}

	wrapped := groupedArms(t, groupedFunction(t, program, "wrapped"))
	if len(wrapped) != 3 || !wrapped[0].Forward || !wrapped[1].Forward || wrapped[0].Error.Identity() == wrapped[1].Error.Identity() {
		t.Fatalf("grouped bare forwarding lost error coverage: %+v", wrapped)
	}
	chain := groupedArms(t, groupedFunction(t, program, "chain_probe"))
	if len(chain) != 3 || chain[0].Outcome != "domain" || chain[1].Outcome != "domain" || chain[0].Body != chain[1].Body {
		t.Fatalf("match chain did not share its grouped handler: %+v", chain)
	}

	assembly, err := assembleProgramBindings(program)
	if err != nil {
		t.Fatal(err)
	}
	fn := groupedFunction(t, program, "wrapped")
	emitted, err := (&RegionEmitter{
		Bindings: assembly.bindings, Functions: assembly.functions,
		DomainRuntime: "$canDomain", SourceID: fn.Symbol.Source.ID,
	}).Function("$wrapped", fn.Region)
	if err != nil {
		t.Fatal(err)
	}
	// A forwarded domain completion must be the very same object, retaining
	// its full payload and occurrence ID. Both branches return the one call's
	// completion; neither reconstructs a domain failure. Fixture-row
	// expectations above the dispatch legitimately construct errors, so scope
	// the absence check to the dispatch region.
	dispatch := emitted
	if i := strings.Index(emitted, "=== 'domain'"); i >= 0 {
		dispatch = emitted[i:]
	}
	if strings.Contains(dispatch, ".create(") {
		t.Fatalf("grouped forwarding created a fresh occurrence:\n%s", emitted)
	}
	forwarded := regexp.MustCompile(`return (\$can[A-Za-z0-9]+) as \$canCompletion<`).FindAllStringSubmatch(emitted, -1)
	if len(forwarded) < 2 || forwarded[0][1] != forwarded[1][1] {
		t.Fatalf("grouped forwarding did not return the same call completion twice:\n%s", emitted)
	}
}

func TestGroupedAssertionRootsExecuteWithSeparateFixtures(t *testing.T) {
	program := groupedSyntaxProgram(t)
	selected := map[string]int{}
	for i, assertion := range program.Assertions {
		if assertion.Root.Declaration == "can.project.root/app::probe" || assertion.Root.Declaration == "can.project.root/app::chain_probe" {
			key := assertion.Root.Declaration + "/" + assertion.Root.Name
			if _, exists := selected[key]; exists {
				t.Fatalf("duplicate grouped root %s", key)
			}
			selected[key] = i
		}
	}
	for _, key := range []string{
		"can.project.root/app::probe/first", "can.project.root/app::probe/second",
		"can.project.root/app::chain_probe/first", "can.project.root/app::chain_probe/second",
	} {
		if _, exists := selected[key]; !exists {
			t.Fatalf("grouped assertion root %s was not independently expanded", key)
		}
	}
	artifacts, err := AssertionModules(program, "runtime", httpDependencies(t))
	if err != nil {
		t.Fatal(err)
	}
	bun, err := exec.LookPath("bun")
	if err != nil {
		t.Skip("Bun is required for emitted assertion execution")
	}
	root := t.TempDir()
	runtimeRoot, err := filepath.Abs("../../../runtime")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(runtimeRoot, filepath.Join(root, "runtime")); err != nil {
		t.Fatal(err)
	}
	for _, artifact := range artifacts {
		if strings.HasPrefix(artifact.Path, "runtime/") {
			continue
		}
		liveWrite(t, root, artifact.Path, artifact.Bytes)
	}
	liveWrite(t, root, "diagnostics/source-index.json", []byte(`{"schemaVersion":1,"kind":"can.source-index","sources":[],"modules":[]}`))
	liveWrite(t, root, "environment.json", []byte("{}"))
	environment, err := os.Open(filepath.Join(root, "environment.json"))
	if err != nil {
		t.Fatal(err)
	}
	defer environment.Close()
	for _, key := range []string{
		"can.project.root/app::probe/first", "can.project.root/app::probe/second",
		"can.project.root/app::chain_probe/first", "can.project.root/app::chain_probe/second",
	} {
		if _, err := environment.Seek(0, 0); err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		cmd := exec.CommandContext(ctx, bun, "--no-install", "--no-env-file", filepath.Join(root, "entry.ts"), "root="+strconv.Itoa(selected[key]))
		cmd.Dir = root
		cmd.ExtraFiles = []*os.File{environment}
		output, runErr := cmd.Output()
		cancel()
		if runErr != nil {
			t.Fatalf("%s failed: %v\n%s", key, runErr, output)
		}
		var report struct {
			Kind   string `json:"kind"`
			Passed bool   `json:"passed"`
			Root   struct {
				Declaration string `json:"declaration"`
				Name        string `json:"name"`
			} `json:"root"`
		}
		if err := json.Unmarshal(output, &report); err != nil {
			t.Fatalf("%s emitted invalid report: %v\n%s", key, err, output)
		}
		if report.Kind != "can.assertion-root-report" || !report.Passed || report.Root.Declaration+"/"+report.Root.Name != key {
			t.Fatalf("%s lost independent report identity or fixtures: %+v", key, report)
		}
	}
}

// groupedReport is the subset of can.assertion-root-report the execution
// tests assert: identity, outcome, and the harness violation list that
// proves fixture queues were consumed exactly and independently.
type groupedReport struct {
	Kind   string `json:"kind"`
	Passed bool   `json:"passed"`
	Reason string `json:"reason"`
	Root   struct {
		Declaration string `json:"declaration"`
		Name        string `json:"name"`
	} `json:"root"`
	Assertion struct {
		Passed     bool     `json:"passed"`
		Reason     string   `json:"reason"`
		Violations []string `json:"violations"`
	} `json:"assertion"`
}

// groupedRunner stages one AssertionModules build in a t.TempDir-owned root
// with a runtime symlink (never a copy) and drives emitted roots and the
// occurrence probe under bounded timeouts. The environment file is opened
// once and closed by cleanup; no temporary directory, bundle, or cache is
// retained after the test.
type groupedRunner struct {
	bun  string
	root string
	env  *os.File
}

func groupedStage(t *testing.T, artifacts []ir.Artifact) *groupedRunner {
	t.Helper()
	bun, err := exec.LookPath("bun")
	if err != nil {
		t.Skip("Bun is required for emitted assertion execution")
	}
	root := t.TempDir()
	runtimeRoot, err := filepath.Abs("../../../runtime")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(runtimeRoot, filepath.Join(root, "runtime")); err != nil {
		t.Fatal(err)
	}
	for _, artifact := range artifacts {
		if strings.HasPrefix(artifact.Path, "runtime/") {
			continue
		}
		liveWrite(t, root, artifact.Path, artifact.Bytes)
	}
	liveWrite(t, root, "diagnostics/source-index.json", []byte(`{"schemaVersion":1,"kind":"can.source-index","sources":[],"modules":[]}`))
	liveWrite(t, root, "environment.json", []byte("{}"))
	liveWrite(t, root, "probe-occurrence.ts", []byte(groupedOccurrenceProbe))
	environment, err := os.Open(filepath.Join(root, "environment.json"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { environment.Close() })
	return &groupedRunner{bun: bun, root: root, env: environment}
}

func (r *groupedRunner) command(t *testing.T, args ...string) []byte {
	t.Helper()
	if _, err := r.env.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, r.bun, args...)
	cmd.Dir = r.root
	cmd.ExtraFiles = []*os.File{r.env}
	output, err := cmd.Output()
	if err != nil {
		// A failing root still delivers its report on stdout with a
		// nonzero status. Only fail here when no report was delivered.
		var probe groupedReport
		if json.Unmarshal(output, &probe) != nil || probe.Kind == "" {
			t.Fatalf("bun %v: %v\n%s", args, err, output)
		}
	}
	return output
}

func (r *groupedRunner) runRoot(t *testing.T, index int) groupedReport {
	t.Helper()
	output := r.command(t, "--no-install", "--no-env-file", filepath.Join(r.root, "entry.ts"), "root="+strconv.Itoa(index))
	var report groupedReport
	if err := json.Unmarshal(output, &report); err != nil {
		t.Fatalf("root %d emitted invalid report: %v\n%s", index, err, output)
	}
	return report
}

// groupedAssertion locates one checked assertion root and its runner index.
func groupedAssertion(t *testing.T, program *check.Program, declaration, name string) (*ir.Assertion, int) {
	t.Helper()
	for i, assertion := range program.Assertions {
		if assertion.Root.Declaration == declaration && assertion.Root.Name == name {
			return assertion, i
		}
	}
	t.Fatalf("missing grouped assertion root %s/%s", declaration, name)
	return nil, 0
}

func groupedExpectPass(t *testing.T, key string, report groupedReport) {
	t.Helper()
	if report.Kind != "can.assertion-root-report" {
		t.Fatalf("%s delivered %q, want can.assertion-root-report", key, report.Kind)
	}
	if report.Root.Declaration+"/"+report.Root.Name != key {
		t.Fatalf("report identity %s/%s, want %s", report.Root.Declaration, report.Root.Name, key)
	}
	if !report.Passed || !report.Assertion.Passed {
		t.Fatalf("%s failed: reason %q violations %q", key, report.Assertion.Reason, report.Assertion.Violations)
	}
	if len(report.Assertion.Violations) != 0 {
		t.Fatalf("%s passed with queue violations %q", key, report.Assertion.Violations)
	}
}

func groupedExpectViolation(t *testing.T, key, want string, report groupedReport) {
	t.Helper()
	if report.Kind != "can.assertion-root-report" {
		t.Fatalf("%s delivered %q, want can.assertion-root-report", key, report.Kind)
	}
	if report.Root.Declaration+"/"+report.Root.Name != key {
		t.Fatalf("report identity %s/%s, want %s", report.Root.Declaration, report.Root.Name, key)
	}
	if report.Passed || report.Assertion.Passed {
		t.Fatalf("%s passed, want a %q failure", key, want)
	}
	for _, violation := range report.Assertion.Violations {
		if violation == want {
			return
		}
	}
	t.Fatalf("%s violations %q, want %q (reason %q)", key, report.Assertion.Violations, want, report.Assertion.Reason)
}

// TestGroupedErrorsExecuteEveryAlternativeAndSuccess executes both error
// alternatives plus ordinary success through grouped call matches, grouped
// bare forwarding, and grouped chain handlers. The wrapped/first and
// wrapped/second roots prove the bare-forwarded completion keeps the exact
// nominal error and payload at execution time; probe/plain,
// wrapped/sample, and chain_probe/third prove success still flows through
// the same grouped matches. Empty violation lists prove each root consumed
// its own fixture rows with no queue bleed between alternatives.
func TestGroupedErrorsExecuteEveryAlternativeAndSuccess(t *testing.T) {
	program := groupedSyntaxProgram(t)
	artifacts, err := AssertionModules(program, "runtime", httpDependencies(t))
	if err != nil {
		t.Fatal(err)
	}
	runner := groupedStage(t, artifacts)
	roots := [][2]string{
		{"can.project.root/app::wrapped", "first"},
		{"can.project.root/app::wrapped", "second"},
		{"can.project.root/app::wrapped", "sample"},
		{"can.project.root/app::fail_first", "sample"},
		{"can.project.root/app::fail_second", "sample"},
		{"can.project.root/app::probe", "first"},
		{"can.project.root/app::probe", "second"},
		{"can.project.root/app::probe", "plain"},
		{"can.project.root/app::chain_probe", "first"},
		{"can.project.root/app::chain_probe", "second"},
		{"can.project.root/app::chain_probe", "third"},
	}
	for _, root := range roots {
		_, index := groupedAssertion(t, program, root[0], root[1])
		groupedExpectPass(t, root[0]+"/"+root[1], runner.runRoot(t, index))
	}
}

// groupedCoordination navigates to the coordination node of a function whose
// body binds one coordination match before its terminal.
func groupedCoordination(t *testing.T, program *check.Program, name string) *ir.Coordination {
	t.Helper()
	fn := groupedFunction(t, program, name)
	if fn.Region == nil || fn.Region.Body == nil || len(fn.Region.Body.Steps) == 0 ||
		fn.Region.Body.Steps[0].Value == nil || fn.Region.Body.Steps[0].Value.Coordination == nil {
		t.Fatalf("%s lost its coordination match", name)
	}
	return fn.Region.Body.Steps[0].Value.Coordination
}

func groupedSharedHandlerArms(t *testing.T, what string, arms []ir.Arm) {
	t.Helper()
	if len(arms) != 3 || arms[0].Outcome != "domain" || arms[1].Outcome != "domain" || arms[2].Outcome != "ok" {
		t.Fatalf("%s grouped handler lost coverage: %+v", what, arms)
	}
	if arms[0].Body == nil || arms[0].Body != arms[1].Body {
		t.Fatalf("%s grouped handler was not checked once and shared", what)
	}
	if arms[0].Error.Identity() == arms[1].Error.Identity() {
		t.Fatalf("%s grouped handler lost its two distinct domain identities", what)
	}
}

// TestGroupedCoordinationArmsExecute covers the coordination paths whose
// lowering differs from ordinary matches: per-participant grouped arms in
// concurrent with error, and the grouped shared arm in race with error.
// The static arm check is supporting only; the passing roots prove both
// alternatives recover at execution time with independent fixture queues.
func TestGroupedCoordinationArmsExecute(t *testing.T) {
	program := groupedSyntaxProgram(t)
	settled := groupedCoordination(t, program, "settled_group")
	if settled.Mode != "settled" || len(settled.Entries) != 2 {
		t.Fatalf("settled_group lost its two participants: %+v", settled)
	}
	for i, entry := range settled.Entries {
		if entry.Handler == nil {
			t.Fatalf("settled participant %d lost its handler", i)
		}
		groupedSharedHandlerArms(t, fmt.Sprintf("settled participant %d", i), entry.Handler.Arms)
	}
	race := groupedCoordination(t, program, "race_group")
	if race.Mode != "race" || race.Shared == nil {
		t.Fatalf("race_group lost its shared handler: %+v", race)
	}
	groupedSharedHandlerArms(t, "race shared", race.Shared.Arms)

	artifacts, err := AssertionModules(program, "runtime", httpDependencies(t))
	if err != nil {
		t.Fatal(err)
	}
	runner := groupedStage(t, artifacts)
	for _, root := range [][2]string{
		{"can.project.root/app::settled_group", "sample"},
		{"can.project.root/app::race_group", "sample"},
	} {
		_, index := groupedAssertion(t, program, root[0], root[1])
		groupedExpectPass(t, root[0]+"/"+root[1], runner.runRoot(t, index))
	}
}

// groupedOccurrenceProbe invokes one emitted assertion case outside the
// assertion runner so the test observes domain occurrence allocation,
// which the runner deliberately ignores (it compares nominal type and
// payload, never occurrence IDs). Sentinel allocations bracket the call:
// exactly one occurrence during the call is the fixture's own creation,
// and zero after its return proves bare forwarding reconstructed nothing.
const groupedOccurrenceProbe = `import { $canInitialize } from "./program/state.ts";
import { assertionContext } from "./runtime/assert/context.ts";
import { domainFailureDiagnostics } from "./runtime/domain.ts";
import { allocateOccurrenceID } from "./runtime/failure.ts";
const target = await import("./" + process.argv[2]);
$canInitialize();
const context = assertionContext(JSON.parse(process.argv[3]));
const before = allocateOccurrenceID();
const completion = await target.$canCase.actual(context);
const after = allocateOccurrenceID();
if (completion.kind !== "domain") {
  console.log(JSON.stringify({ kind: completion.kind }));
  process.exit(0);
}
const details = domainFailureDiagnostics(completion.value);
let payloadText = null;
try {
  payloadText = JSON.stringify(details.payload);
} catch {
  payloadText = null;
}
console.log(JSON.stringify({
  kind: "domain",
  typeIdentity: details.typeIdentity,
  allocatedDuringCall: (details.occurrenceID - before).toString(),
  allocatedAfterReturn: (after - details.occurrenceID).toString(),
  payloadText,
}));
`

type groupedProbeResult struct {
	Kind                 string  `json:"kind"`
	TypeIdentity         string  `json:"typeIdentity"`
	AllocatedDuringCall  string  `json:"allocatedDuringCall"`
	AllocatedAfterReturn string  `json:"allocatedAfterReturn"`
	PayloadText          *string `json:"payloadText"`
}

// groupedCasePath finds the emitted assertion-case module embedding one root.
func groupedCasePath(t *testing.T, artifacts []ir.Artifact, root ir.AssertionRoot) string {
	t.Helper()
	want := fmt.Sprintf(`"declaration":%q,"name":%q`, root.Declaration, root.Name)
	for _, artifact := range artifacts {
		if !strings.HasPrefix(artifact.Path, "assertions/") || !strings.HasSuffix(artifact.Path, ".ts") {
			continue
		}
		if strings.Contains(string(artifact.Bytes), want) {
			return artifact.Path
		}
	}
	t.Fatalf("emitted case module missing for %s/%s", root.Declaration, root.Name)
	return ""
}

// TestGroupedForwardingPreservesOccurrenceAtExecution proves bare forwarding
// returns the original protected completion at execution time, per error
// alternative. The absence of .create() in generated text (see the static
// emission test) cannot show this alone: the probe counts live occurrence
// allocations around each forwarded call and compares the returned
// occurrence's type identity and payload with the checked contract.
func TestGroupedForwardingPreservesOccurrenceAtExecution(t *testing.T) {
	program := groupedSyntaxProgram(t)
	wrapped := groupedArms(t, groupedFunction(t, program, "wrapped"))
	if len(wrapped) != 3 || wrapped[0].Outcome != "domain" || wrapped[1].Outcome != "domain" {
		t.Fatalf("grouped bare forwarding lost error coverage: %+v", wrapped)
	}
	expect := map[string]struct {
		identity string
		payloads []string
	}{
		"first":  {wrapped[0].Error.Identity(), []string{`"first payload"`}},
		"second": {wrapped[1].Error.Identity(), []string{`"/second"`, `"second payload"`}},
	}
	artifacts, err := AssertionModules(program, "runtime", httpDependencies(t))
	if err != nil {
		t.Fatal(err)
	}
	runner := groupedStage(t, artifacts)
	for _, name := range []string{"first", "second"} {
		assertion, _ := groupedAssertion(t, program, "can.project.root/app::wrapped", name)
		rootJSON, err := json.Marshal(assertion.Root)
		if err != nil {
			t.Fatal(err)
		}
		output := runner.command(t, "--no-install", "--no-env-file", filepath.Join(runner.root, "probe-occurrence.ts"), groupedCasePath(t, artifacts, assertion.Root), string(rootJSON))
		var result groupedProbeResult
		if err := json.Unmarshal(output, &result); err != nil {
			t.Fatalf("wrapped/%s probe emitted invalid result: %v\n%s", name, err, output)
		}
		if result.Kind != "domain" {
			t.Fatalf("wrapped/%s forwarded kind %q, want domain", name, result.Kind)
		}
		if result.TypeIdentity != expect[name].identity {
			t.Fatalf("wrapped/%s forwarded identity %q, want %q", name, result.TypeIdentity, expect[name].identity)
		}
		if result.AllocatedDuringCall != "1" || result.AllocatedAfterReturn != "1" {
			t.Fatalf("wrapped/%s allocated %s occurrence(s) during the call and %s after return, want exactly the one fixture occurrence",
				name, result.AllocatedDuringCall, result.AllocatedAfterReturn)
		}
		for _, payload := range expect[name].payloads {
			if result.PayloadText == nil || !strings.Contains(*result.PayloadText, payload) {
				t.Fatalf("wrapped/%s forwarded payload %v, want %s", name, result.PayloadText, payload)
			}
		}
	}
}

// TestGroupedScenarioAndTemplateRootsExecuteIndependently proves grouped
// scenario selectors and grouped template uses expand into independent
// per-selector rows at execution time. Each fetch root activates only its
// own scenario row (guest/visitor share one link through grouped roots
// with independent queues); each tpl_consumer root consumes only its own
// template expansion, whose planted value differs from the live operation
// so a collapsed expansion would fail. Named reports with empty violation
// lists prove the roots, queues, and contexts stayed independent.
func TestGroupedScenarioAndTemplateRootsExecuteIndependently(t *testing.T) {
	program := groupedSyntaxProgram(t)
	artifacts, err := AssertionModules(program, "runtime", httpDependencies(t))
	if err != nil {
		t.Fatal(err)
	}
	runner := groupedStage(t, artifacts)
	for _, root := range [][2]string{
		{"can.project.root/app::fetch", "customer"},
		{"can.project.root/app::fetch", "partner"},
		{"can.project.root/app::fetch", "guest"},
		{"can.project.root/app::fetch", "visitor"},
		{"can.project.root/app::tpl_consumer", "tpl_first"},
		{"can.project.root/app::tpl_consumer", "tpl_second"},
	} {
		_, index := groupedAssertion(t, program, root[0], root[1])
		groupedExpectPass(t, root[0]+"/"+root[1], runner.runRoot(t, index))
	}
}

// TestGroupedRepeatedSelectorFIFOAndQueueFailures proves repeated selectors
// on separate rows of one lexical table form a FIFO queue shared by
// recursive visits: drain/rep consumes 7 then 8 across its outer and inner
// calls (a LIFO or re-read order would compute 87, 77, or 88 instead of
// 78). The two failing roots prove queue accounting in both directions:
// picky/flush leaves a row unused, hungry/starved visits past its last
// row, and each must fail with exactly that violation.
func TestGroupedRepeatedSelectorFIFOAndQueueFailures(t *testing.T) {
	program := groupedSyntaxProgram(t)
	drain := groupedFunction(t, program, "drain")
	if drain.Region == nil || drain.Region.Body == nil || drain.Region.Body.Terminal == nil || drain.Region.Body.Terminal.Match == nil ||
		drain.Region.Body.Terminal.Match.Call == nil || len(drain.Region.Body.Terminal.Match.Call.Steps) != 1 ||
		drain.Region.Body.Terminal.Match.Call.Steps[0].Fixtures == nil {
		t.Fatalf("drain lost its fixture table")
	}
	rows := drain.Region.Body.Terminal.Match.Call.Steps[0].Fixtures.Rows
	if len(rows) != 2 || rows[0].Selector != "rep" || rows[1].Selector != "rep" {
		t.Fatalf("repeated selectors did not stay ordered rows: %+v", rows)
	}
	artifacts, err := AssertionModules(program, "runtime", httpDependencies(t))
	if err != nil {
		t.Fatal(err)
	}
	runner := groupedStage(t, artifacts)
	_, rep := groupedAssertion(t, program, "can.project.root/app::drain", "rep")
	groupedExpectPass(t, "can.project.root/app::drain/rep", runner.runRoot(t, rep))
	_, flush := groupedAssertion(t, program, "can.project.root/app::picky", "flush")
	groupedExpectViolation(t, "can.project.root/app::picky/flush", "unused fixture", runner.runRoot(t, flush))
	_, starved := groupedAssertion(t, program, "can.project.root/app::hungry", "starved")
	groupedExpectViolation(t, "can.project.root/app::hungry/starved", "missing fixture", runner.runRoot(t, starved))
}

// TestGroupedSharedHandlerReenteredThroughBothAlternatives proves the one
// grouped handler keeps a single nested fixture site at execution time. One
// root enters it twice through different errors; the shared nested table is
// consumed FIFO for 78. IR pointer sharing alone cannot show this: separate
// per-alternative tables would each yield 7 for 77 instead.
func TestGroupedSharedHandlerReenteredThroughBothAlternatives(t *testing.T) {
	program := groupedSyntaxProgram(t)
	revisit := groupedArms(t, groupedFunction(t, program, "revisit"))
	if len(revisit) != 3 || revisit[0].Body == nil || revisit[0].Body != revisit[1].Body {
		t.Fatalf("grouped handler was not checked once and shared: %+v", revisit)
	}
	artifacts, err := AssertionModules(program, "runtime", httpDependencies(t))
	if err != nil {
		t.Fatal(err)
	}
	runner := groupedStage(t, artifacts)
	_, both := groupedAssertion(t, program, "can.project.root/app::revisit", "both")
	groupedExpectPass(t, "can.project.root/app::revisit/both", runner.runRoot(t, both))
}
