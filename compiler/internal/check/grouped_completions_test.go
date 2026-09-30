package check

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/project"
	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
)

func TestGroupedCompletionChecksOneSharedBody(t *testing.T) {
	header := strings.Replace(programHeader, "uses []", "uses [codec, text]", 1)
	src := header + `fn int choose
    emits {codec::invalid_data, text::invalid_number}
    asserts
        sample: => ok 1
    ok 1
fn int fallback
    emits {}
    asserts
        sample: => ok 0
    ok 0
fn int handle
    emits {}
    asserts
        sample: => ok 1
    match call choose()
        codec::invalid_data | text::invalid_number => ok call fallback()
        ok int value => ok value
` + programMain + "    ok\n"
	program, err := programFixture(t, map[string]string{"src/main.can": src})
	if err != nil {
		t.Fatal(err)
	}
	for _, fn := range program.Functions {
		if fn.Symbol.Name != "handle" {
			continue
		}
		arms := fn.Region.Body.Terminal.Match.Arms
		if len(arms) != 3 || arms[0].Outcome != "domain" || arms[1].Outcome != "domain" || arms[0].Body == nil || arms[0].Body != arms[1].Body {
			t.Fatalf("group did not retain two exact heads and one checked body: %+v", arms)
		}
		return
	}
	t.Fatal("missing checked handler")
}

func TestGroupedCompletionPreservesCoverageAndForwarding(t *testing.T) {
	header := strings.Replace(programHeader, "uses []", "uses [codec, text]", 1)
	base := header + `fn int choose
    emits {codec::invalid_data, text::invalid_number}
    asserts
        sample: => ok 1
    ok 1
fn int handle
    emits {codec::invalid_data, text::invalid_number}
    asserts
        sample: => ok 1
    match call choose()
        codec::invalid_data | text::invalid_number
        ok int value => ok value
` + programMain + "    ok\n"
	program, err := programFixture(t, map[string]string{"src/main.can": base})
	if err != nil {
		t.Fatalf("grouped bare forwarding rejected: %v", err)
	}
	for _, fn := range program.Functions {
		if fn.Symbol.Name == "handle" {
			arms := fn.Region.Body.Terminal.Match.Arms
			if len(arms) != 3 || !arms[0].Forward || !arms[1].Forward || arms[0].Error.Identity() == arms[1].Error.Identity() {
				t.Fatalf("grouped forwarding lost exact error members: %+v", arms)
			}
		}
	}
	duplicate := strings.Replace(base, "        ok int value => ok value", "        text::invalid_number\n        ok int value => ok value", 1)
	if _, err := programFixture(t, map[string]string{"src/main.can": duplicate}); err == nil || !strings.Contains(err.Error(), "duplicate completion arm") {
		t.Fatalf("duplicate grouped member admitted: %v", err)
	}
}

// E01: every group member resolves against the exact bound. A later
// alternative outside the bound fails with its own head span, and a bound
// member covered by no head is still reported missing.
func TestGroupedCompletionValidatesEveryMember(t *testing.T) {
	header := strings.Replace(programHeader, "uses []", "uses [codec, text]", 1)
	choose := `fn int choose
    emits {codec::invalid_data}
    asserts
        sample: => ok 1
    ok 1
`
	impossible := header + choose + `fn int handle
    emits {}
    asserts
        sample: => ok 1
    match call choose()
        codec::invalid_data | text::invalid_number => ok 0
        ok int value => ok value
` + programMain + "    ok\n"
	err := mustFixtureError(t, impossible)
	if !strings.Contains(err.Error(), "is outside matched bound") {
		t.Fatalf("impossible later alternative admitted: %v", err)
	}
	located, ok := source.AsLocated(err)
	if !ok {
		t.Fatalf("member diagnostic lost its span: %v", err)
	}
	if head := impossible[located.Span.Start:located.Span.End]; !strings.Contains(head, "text::invalid_number") {
		t.Fatalf("member diagnostic points at %q instead of the failing alternative", head)
	}

	both := strings.Replace(choose, "emits {codec::invalid_data}", "emits {codec::invalid_data, text::invalid_number}", 1)
	missing := header + both + `fn int handle
    emits {}
    asserts
        sample: => ok 1
    match call choose()
        codec::invalid_data => ok 0
        ok int value => ok value
` + programMain + "    ok\n"
	if err := mustFixtureError(t, missing); !strings.Contains(err.Error(), "missing completion arm for text::invalid_number") {
		t.Fatalf("uncovered member admitted: %v", err)
	}

	within := header + both + `fn int handle
    emits {}
    asserts
        sample: => ok 1
    match call choose()
        codec::invalid_data | codec::invalid_data => ok 0
        ok int value => ok value
` + programMain + "    ok\n"
	if err := mustFixtureError(t, within); !strings.Contains(err.Error(), "duplicate completion arm") {
		t.Fatalf("duplicate within one group admitted: %v", err)
	}

	late := header + both + `fn int handle
    emits {}
    asserts
        sample: => ok 1
    match call choose()
        ok int value => ok value
        codec::invalid_data | text::invalid_number => ok 0
` + programMain + "    ok\n"
	if err := mustFixtureError(t, late); !strings.Contains(err.Error(), "failure arms precede the final ok") {
		t.Fatalf("group after success admitted: %v", err)
	}
}

// E01: a bare generic head in alternate position gathers like the first
// head and resolves against the bound. Two specializations keep the
// exact-specialization obligation with its stable code.
func TestGroupedCompletionResolvesBareGenericAlternate(t *testing.T) {
	src := strings.Replace(programHeader, "uses []", "uses [codec, text]", 1) + `variant a_failure
    codec::invalid_data
    standard_failure
variant b_failure
    text::invalid_number
    standard_failure
fn int choose
    emits {all_failed<a_failure>, all_failed<b_failure>}
    asserts
        sample: => ok 1
    ok 1
fn int handle
    emits {}
    asserts
        sample: => ok 1
    match call choose()
        all_failed<a_failure> | all_failed => ok 0
        ok int value => ok value
` + programMain + "    ok\n"
	err := mustFixtureError(t, src)
	if !strings.Contains(err.Error(), "ambiguous error head") {
		t.Fatalf("ambiguous generic alternate admitted: %v", err)
	}
	located, ok := source.AsLocated(err)
	if !ok || located.Code != "CAN-CHECK-EXACT-SPECIALIZATION" {
		t.Fatalf("ambiguity diagnostic lost its code: %v", err)
	}
}

// E02: grouped handlers bind no error payload while single arms keep
// their existing implicit and explicit bindings.
func TestGroupedCompletionArmsCarryNoPayloadBinding(t *testing.T) {
	header := strings.Replace(programHeader, "uses []", "uses [codec, text]", 1)
	choose := `fn int choose
    emits {codec::invalid_data, text::invalid_number}
    asserts
        sample: => ok 1
    ok 1
`
	grouped := header + choose + `fn int handle
    emits {}
    asserts
        sample: => ok 1
    match call choose()
        codec::invalid_data | text::invalid_number => ok 0
        ok int value => ok value
` + programMain + "    ok\n"
	program, err := programFixture(t, map[string]string{"src/main.can": grouped})
	if err != nil {
		t.Fatal(err)
	}
	seen := false
	for _, fn := range program.Functions {
		if fn.Symbol.Name != "handle" {
			continue
		}
		seen = true
		arms := fn.Region.Body.Terminal.Match.Arms
		if len(arms) != 3 || arms[0].Binding != nil || arms[1].Binding != nil {
			t.Fatalf("grouped arms bound payloads: %+v", arms)
		}
	}
	if !seen {
		t.Fatal("missing checked handler")
	}

	single := header + choose + `fn int handle
    emits {}
    asserts
        sample: => ok 1
    match call choose()
        codec::invalid_data => ok 0
        text::invalid_number as merged => ok 0
        ok int value => ok value
` + programMain + "    ok\n"
	program, err = programFixture(t, map[string]string{"src/main.can": single})
	if err != nil {
		t.Fatal(err)
	}
	for _, fn := range program.Functions {
		if fn.Symbol.Name != "handle" {
			continue
		}
		arms := fn.Region.Body.Terminal.Match.Arms
		if len(arms) != 3 || arms[0].Binding == nil || arms[1].Binding == nil {
			t.Fatalf("single arm lost its payload binding: %+v", arms)
		}
		return
	}
	t.Fatal("missing checked handler")
}

// E02: grouped aliases stay rejected; single-arm aliases keep working.
func TestGroupedCompletionRejectsAliases(t *testing.T) {
	header := strings.Replace(programHeader, "uses []", "uses [codec, text]", 1)
	src := header + `fn int choose
    emits {codec::invalid_data, text::invalid_number}
    asserts
        sample: => ok 1
    ok 1
fn int handle
    emits {}
    asserts
        sample: => ok 1
    match call choose()
        codec::invalid_data | text::invalid_number as merged => ok 0
        ok int value => ok value
` + programMain + "    ok\n"
	mustFixtureError(t, src)
	file, err := source.New("src/main.can", src)
	if err != nil {
		t.Fatal(err)
	}
	for _, diagnostic := range syntax.Parse(file).Diagnostics {
		if strings.Contains(diagnostic.Message, "grouped completion heads") {
			return
		}
	}
	t.Fatal("grouped alias diagnostic missing")
}

// E02: every bare member must fit the enclosing bound. The escaping
// member carries its own head span.
func TestGroupedForwardingRequiresEveryMemberInBound(t *testing.T) {
	header := strings.Replace(programHeader, "uses []", "uses [codec, text]", 1)
	src := header + `fn int choose
    emits {codec::invalid_data, text::invalid_number}
    asserts
        sample: => ok 1
    ok 1
fn int handle
    emits {codec::invalid_data}
    asserts
        sample: => ok 1
    match call choose()
        codec::invalid_data | text::invalid_number
        ok int value => ok value
` + programMain + "    ok\n"
	err := mustFixtureError(t, src)
	located, ok := source.AsLocated(err)
	if !ok || located.Code != "CAN-CHECK-OUTWARD-ERROR" {
		t.Fatalf("escaping member lost its outward diagnostic: %v", err)
	}
	if head := src[located.Span.Start:located.Span.End]; !strings.Contains(head, "text::invalid_number") {
		t.Fatalf("outward diagnostic points at %q instead of the escaping member", head)
	}
}

// E02: a failed group must not leak partial coverage into later arms.
// Recovery keeps the body failure and checks the re-covering arm on its
// own merits instead of reporting a duplicate.
func TestGroupedRecoveryRestoresCoverage(t *testing.T) {
	header := strings.Replace(programHeader, "uses []", "uses [codec, text]", 1)
	src := header + `fn int choose
    emits {codec::invalid_data, text::invalid_number}
    asserts
        sample: => ok 1
    ok 1
fn int handle
    emits {}
    asserts
        sample: => ok 1
    match call choose()
        codec::invalid_data | text::invalid_number => ok call missing()
        codec::invalid_data => ok 0
        ok int value => ok value
` + programMain + "    ok\n"
	_, err := analyzeFixture(t, map[string]string{"src/main.can": src})
	if err == nil {
		t.Fatal("failed group body admitted")
	}
	if strings.Contains(err.Error(), "duplicate completion arm") {
		t.Fatalf("failed group leaked coverage: %v", err)
	}
	if !strings.Contains(err.Error(), `no eligible declaration for "missing"`) {
		t.Fatalf("recovery lost the body failure: %v", err)
	}
}

// E03: coordination surfaces share the grouping rules. Race-with-error
// handlers and settled participant handlers accept groups with one
// checked body; first-success race keeps single heads.
func TestGroupedCoordinationHandlers(t *testing.T) {
	header := strings.Replace(programHeader, "uses []", "uses [codec]", 1) + "error stale{str key}\n"
	declarations := coordinationDeclarations + `fn int other
    emits {stale}
    asserts
        sample: => ok 2
    ok 2
fn int choose
    emits {codec::invalid_data, stale}
    asserts
        sample: => ok 1
    ok 1
`
	race := header + declarations + programMain + `    int result = match call race with error
        number()
        other()
        codec::invalid_data | stale => ok 0
        ok int value => ok value
    ok
`
	registry := `{"active":["app::stale"],"retired":[]}`
	fixture := func(src string) (*Program, error) {
		return programFixtureRegistry(t, map[string]string{"src/main.can": src}, registry)
	}
	program, err := fixture(race)
	if err != nil {
		t.Fatalf("grouped race handler rejected: %v", err)
	}
	node := program.Entry.Region.Body.Steps[0].Value.Coordination
	if node.Mode != "race" || node.Shared == nil || len(node.Shared.Arms) != 3 {
		t.Fatalf("grouped race handler lost its shared arms: %+v", node.Shared)
	}
	if node.Shared.Arms[0].Body == nil || node.Shared.Arms[0].Body != node.Shared.Arms[1].Body {
		t.Fatal("grouped race handler checked its body twice")
	}

	settled := header + declarations + programMain + `    int[] results = match call concurrent with error
        choose()
            codec::invalid_data | stale => ok 0
            ok int value => ok value
        text()
            ok str value => ok value.length
    ok
`
	program, err = fixture(settled)
	if err != nil {
		t.Fatalf("grouped participant handler rejected: %v", err)
	}
	entries := program.Entry.Region.Body.Steps[0].Value.Coordination.Entries
	if len(entries) != 2 || len(entries[0].Handler.Arms) != 3 {
		t.Fatalf("grouped participant handler lost its arms: %+v", entries)
	}
	if entries[0].Handler.Arms[0].Body == nil || entries[0].Handler.Arms[0].Body != entries[0].Handler.Arms[1].Body {
		t.Fatal("grouped participant handler checked its body twice")
	}

	first := header + declarations + programMain + `    int result = match call race
        number()
        ok int value => ok value
        all_failed | codec::invalid_data => ok 0
    ok
`
	if _, err := fixture(first); err == nil || !strings.Contains(err.Error(), "without grouped alternatives") {
		t.Fatalf("grouped first-success arm admitted: %v", err)
	}
}

func mustFixtureError(t *testing.T, src string) error {
	t.Helper()
	_, err := programFixture(t, map[string]string{"src/main.can": src})
	if err == nil {
		t.Fatal("invalid program admitted")
	}
	return err
}

// analyzeFixture stages sources like programFixture but checks with
// recovery enabled, collecting per-arm diagnostics.
func analyzeFixture(t *testing.T, files map[string]string) (*Program, error) {
	t.Helper()
	root := t.TempDir()
	files["can.project.json"] = `{"source_root":"src","error_registry":"can.errors.json"}`
	files["can.errors.json"] = `{"active":[],"retired":[]}`
	for name, text := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	graph, err := project.Load(root)
	if err != nil {
		return nil, err
	}
	world, err := resolve.Build(graph)
	if err != nil {
		return nil, err
	}
	return AnalyzeProgram(graph, world)
}
