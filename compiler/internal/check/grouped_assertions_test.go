package check

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/project"
	"github.com/veighnsche/can-lang/compiler/internal/resolve"
)

func groupedRootsByDeclaration(t *testing.T, program *Program, declaration string) map[string]int {
	t.Helper()
	roots := map[string]int{}
	for i, assertion := range program.Assertions {
		if strings.HasSuffix(assertion.Root.Declaration, "::"+declaration) {
			if _, exists := roots[assertion.Root.Name]; exists {
				t.Fatalf("duplicate grouped root %s/%s", declaration, assertion.Root.Name)
			}
			roots[assertion.Root.Name] = i
		}
	}
	return roots
}

func TestGroupedAssertionRootsStayIndependent(t *testing.T) {
	src := programHeader + `fn int source
    emits {}
    asserts
        first | second: => ok 7
    ok 7
fn int consumer
    emits {}
    asserts
        first | second: => ok 7
    match call source()
        when
            first | second: => ok 7
        ok int value => ok value
` + programMain + "    ok\n"
	program, err := programFixture(t, map[string]string{"src/main.can": src})
	if err != nil {
		t.Fatal(err)
	}
	for _, declaration := range []string{"source", "consumer"} {
		roots := groupedRootsByDeclaration(t, program, declaration)
		for _, name := range []string{"first", "second"} {
			index, ok := roots[name]
			if !ok {
				t.Fatalf("missing independent assertion root %s/%s: %v", declaration, name, roots)
			}
			assertion := program.Assertions[index]
			if assertion.Actual == nil || assertion.Expected == nil {
				t.Fatalf("grouped root %s/%s lost its checked completions", declaration, name)
			}
		}
	}
	rows := templateFixtureSteps(t, program, "consumer").Fixtures.Rows
	if len(rows) != 2 || rows[0].Selector != "first" || rows[1].Selector != "second" {
		t.Fatalf("grouped when labels lost order/selectors: %+v", rows)
	}
	for name, bad := range map[string]string{
		"duplicate across group and ordinary row": strings.Replace(src, "        first | second: => ok 7\n    ok 7", "        first | second: => ok 7\n        second: => ok 7\n    ok 7", 1),
		"duplicate across different groups":       strings.Replace(src, "        first | second: => ok 7\n    ok 7", "        first | second: => ok 7\n        second | third: => ok 7\n    ok 7", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := programFixture(t, map[string]string{"src/main.can": bad}); err == nil || !strings.Contains(err.Error(), "assertion name") {
				t.Fatalf("duplicate grouped assertion name admitted: %v", err)
			}
		})
	}
}

func TestGroupedAssertionRootsShareArgsExpectedAndLinks(t *testing.T) {
	src := programHeader + `scenario setup
fn int load
    emits {}
    given
        str name
    asserts
        absent | configured: "x" => ok 7 link setup
    ok 7
` + programMain + "    ok\n"
	program, err := programFixture(t, map[string]string{"src/main.can": src})
	if err != nil {
		t.Fatal(err)
	}
	roots := groupedRootsByDeclaration(t, program, "load")
	if len(roots) != 2 {
		t.Fatalf("grouped roots = %v, want absent and configured", roots)
	}
	var links []string
	for _, name := range []string{"absent", "configured"} {
		index, ok := roots[name]
		if !ok {
			t.Fatalf("missing grouped root %s: %v", name, roots)
		}
		assertion := program.Assertions[index]
		if assertion.Actual == nil || assertion.Expected == nil {
			t.Fatalf("grouped root %s lost its checked completions", name)
		}
		if links == nil {
			links = assertion.Root.Links
		} else if strings.Join(links, ",") != strings.Join(assertion.Root.Links, ",") {
			t.Fatalf("grouped roots kept different links: %v vs %v", links, assertion.Root.Links)
		}
	}
	if len(links) != 1 || !strings.HasSuffix(links[0], "::setup") {
		t.Fatalf("grouped roots lost their shared link: %v", links)
	}
	mistyped := strings.Replace(src, `"x" => ok 7`, `7 => ok 7`, 1)
	if _, err := programFixture(t, map[string]string{"src/main.can": mistyped}); err == nil {
		t.Fatal("mistyped grouped assertion arguments admitted")
	}
	raw := strings.Replace(src, "link setup\n", "link setup\n            using raw \"fixtures/load.json\"\n", 1)
	if _, err := programFixture(t, withNativeRaw(map[string]string{"src/main.can": raw}, "load")); err == nil || !strings.Contains(err.Error(), "using raw is allowed on fetch/judge/LLM/wrapper targets only") {
		t.Fatalf("raw mode on grouped ordinary roots admitted: %v", err)
	}
	use := programHeader + templateTarget + `fixture doubled for double
    cases
        2 => ok 4
fn int bad
    emits {}
    asserts
        first | second: use doubled(2)
    ok 1
` + programMain + "    ok\n"
	if _, err := programFixture(t, map[string]string{"src/main.can": use}); err == nil || !strings.Contains(err.Error(), "lexical when tables only") {
		t.Fatalf("grouped template use on assertion roots admitted: %v", err)
	}
}

func TestGroupedGenericAssertionRoots(t *testing.T) {
	src := programHeader + `fn item identity<item>
    emits {}
    given
        item value
    asserts
        first | second: 3 => ok 3
    ok value
` + programMain + `    int result = call identity<int>(3)
    match result
        3 => ok
        _ => ok
`
	program, err := programFixture(t, map[string]string{"src/main.can": src})
	if err != nil {
		t.Fatal(err)
	}
	roots := groupedRootsByDeclaration(t, program, "identity")
	for _, name := range []string{"first", "second"} {
		if _, ok := roots[name]; !ok {
			t.Fatalf("missing generic grouped root %s: %v", name, roots)
		}
	}
	duplicate := strings.Replace(src, "first | second: 3 => ok 3", "first | second: 3 => ok 3\n        second | third: 3 => ok 3", 1)
	if _, err := programFixture(t, map[string]string{"src/main.can": duplicate}); err == nil || !strings.Contains(err.Error(), "assertion name") {
		t.Fatalf("duplicate generic grouped assertion name admitted: %v", err)
	}
}

// recoveringProgramFixture stages sources like programFixture but checks
// with recovery enabled, so partial programs stay observable.
func recoveringProgramFixture(t *testing.T, files map[string]string) (*Program, error) {
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

func TestGroupedGenericDuplicatesSkippedUnderRecovery(t *testing.T) {
	src := programHeader + `fn item identity<item>
    emits {}
    given
        item value
    asserts
        first | second: 3 => ok 3
        second | third: 3 => ok 3
    ok value
` + programMain + `    int result = call identity<int>(3)
    match result
        3 => ok
        _ => ok
`
	program, err := recoveringProgramFixture(t, map[string]string{"src/main.can": src})
	if err == nil || !strings.Contains(err.Error(), "duplicate assertion name") {
		t.Fatalf("duplicate generic grouped name not reported under recovery: %v", err)
	}
	seen := map[string]bool{}
	for _, assertion := range program.Assertions {
		if !strings.HasSuffix(assertion.Root.Declaration, "::identity") {
			continue
		}
		if seen[assertion.Root.Name] {
			t.Fatalf("duplicate generic root %s kept under recovery", assertion.Root.Name)
		}
		seen[assertion.Root.Name] = true
	}
}

func TestGroupedNativeAssertionRoots(t *testing.T) {
	native := "llm str generate from generator\n    emits {" + nativeLLM + "}\n    asserts\n        first | second: () => ok \"x\"\n            using raw \"fixtures/generate.json\"\n    asks \"Generate\"\n"
	program, err := programFixture(t, withNativeRaw(map[string]string{"src/main.can": nativeHeader + nativeGenerator + native + programMain + "    ok\n"}, "generate"))
	if err != nil {
		t.Fatal(err)
	}
	roots := groupedRootsByDeclaration(t, program, "generate")
	for _, name := range []string{"first", "second"} {
		index, ok := roots[name]
		if !ok {
			t.Fatalf("missing native grouped root %s: %v", name, roots)
		}
		if program.Assertions[index].Raw == nil {
			t.Fatalf("native grouped root %s lost its raw mode", name)
		}
	}
	duplicate := strings.Replace(native, "        first | second: () => ok \"x\"\n", "        first | second: () => ok \"x\"\n        second: () => ok \"x\"\n", 1)
	bad := nativeHeader + nativeGenerator + duplicate + programMain + "    ok\n"
	if _, err := programFixture(t, withNativeRaw(map[string]string{"src/main.can": bad}, "generate")); err == nil || !strings.Contains(err.Error(), "duplicate assertion name") {
		t.Fatalf("duplicate native grouped assertion name admitted: %v", err)
	}
}

func TestGroupedWhenSelectorsKeepSourceOrderAndRepeatedRows(t *testing.T) {
	src := programHeader + `fn int double
    emits {}
    given
        int value
    asserts
        sample: 2 => ok 4
    ok value + value
fn int consumer
    emits {}
    asserts
        sample: => ok 4
    match call double(2)
        when
            first | second: 2 => ok 4
            third: 2 => ok 4
            first: 2 => ok 4
        ok int got => ok got
` + programMain + "    ok\n"
	program, err := programFixture(t, map[string]string{"src/main.can": src})
	if err != nil {
		t.Fatal(err)
	}
	rows := templateFixtureSteps(t, program, "consumer").Fixtures.Rows
	var selectors []string
	for _, row := range rows {
		selectors = append(selectors, row.Selector)
		if row.Expected == nil || len(row.Arguments) != 1 {
			t.Fatalf("selector %s lost its arguments or completion: %+v", row.Selector, row)
		}
	}
	want := []string{"first", "second", "third", "first"}
	if strings.Join(selectors, ",") != strings.Join(want, ",") {
		t.Fatalf("fixture selectors = %v, want %v in source order with the repeated row kept", selectors, want)
	}
}

func TestGroupedWhenScenarioLabelsResolveIndependently(t *testing.T) {
	src := programHeader + `scenario ready
scenario served
fn int load
    emits {}
    given
        str name
    asserts
        sample: "x" => ok 7
    ok 7
fn int consumer
    emits {}
    asserts
        sample: => ok 7
    match call load("x")
        when
            scenario ready | served: "x" => ok 7
        ok int got => ok got
` + programMain + "    ok\n"
	program, err := programFixture(t, map[string]string{"src/main.can": src})
	if err != nil {
		t.Fatal(err)
	}
	rows := templateFixtureSteps(t, program, "consumer").Fixtures.Rows
	if len(rows) != 2 {
		t.Fatalf("grouped scenario row expanded to %d rows: %+v", len(rows), rows)
	}
	for i, name := range []string{"ready", "served"} {
		if rows[i].Selector != name || !strings.HasSuffix(rows[i].Scenario, "::"+name) {
			t.Fatalf("scenario selector %d = %+v, want independent %s resolution", i, rows[i], name)
		}
	}
	if rows[0].Scenario == rows[1].Scenario {
		t.Fatalf("grouped scenario labels shared one scenario: %+v", rows)
	}
	missing := strings.Replace(src, "scenario ready | served:", "scenario ready | absent:", 1)
	if _, err := programFixture(t, map[string]string{"src/main.can": missing}); err == nil || !strings.Contains(err.Error(), `scenario "absent" is not declared`) {
		t.Fatalf("later grouped scenario label without a declaration admitted: %v", err)
	}
}

func TestGroupedWhenTemplateUseExpandsPerSelector(t *testing.T) {
	text := programHeader + templateTarget + `fixture doubled for double
    given
        int base
    cases
        base => ok base + base
        3 => ok 6
fn int consumer
    emits {}
    asserts
        sample: => ok 4
    match call double(2)
        when
            first | second: use doubled(2)
        ok int got => ok got
` + programMain + "    ok\n"
	program, err := programFixture(t, map[string]string{"src/main.can": text})
	if err != nil {
		t.Fatal(err)
	}
	rows := templateFixtureSteps(t, program, "consumer").Fixtures.Rows
	var selectors []string
	for _, row := range rows {
		selectors = append(selectors, row.Selector)
	}
	want := []string{"first", "first", "second", "second"}
	if strings.Join(selectors, ",") != strings.Join(want, ",") {
		t.Fatalf("grouped template use selectors = %v, want %v", selectors, want)
	}
}

func TestGroupedWhenInNestedHandlerExpands(t *testing.T) {
	src := programHeader + `fn int source
    emits {}
    asserts
        sample: => ok 0
    ok 0
fn int marker
    emits {}
    asserts
        sample: => ok 9
    ok 9
fn int probe
    emits {}
    asserts
        first | second: => ok 7
    match call source()
        ok => match call marker()
            when
                first | second: => ok 7
            ok int nested => ok nested
` + programMain + "    ok\n"
	program, err := programFixture(t, map[string]string{"src/main.can": src})
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, fn := range program.Functions {
		if fn.Symbol.Name != "probe" {
			continue
		}
		found = true
		arms := fn.Region.Body.Terminal.Match.Arms
		if len(arms) != 1 || arms[0].Body == nil || arms[0].Body.Match == nil {
			t.Fatalf("nested handler match lost: %+v", arms)
		}
		rows := arms[0].Body.Match.Call.Steps[0].Fixtures.Rows
		if len(rows) != 2 || rows[0].Selector != "first" || rows[1].Selector != "second" {
			t.Fatalf("nested grouped when labels lost order/selectors: %+v", rows)
		}
	}
	if !found {
		t.Fatal("probe function missing")
	}
}
