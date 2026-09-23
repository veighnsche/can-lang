package check

import (
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/project"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/types"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const coordinationDeclarations = `fn int number
    emits [codec::invalid_data]
    asserts
        sample: => ok 1
    ok 1
fn str text
    emits []
    asserts
        sample: => ok "text"
    ok "text"
fn int input
    emits []
    given
        int value
    asserts
        sample: 1 => ok 1
    ok value
`

func coordinationProgram(t *testing.T, body string) (*Program, error) {
	return programFixture(t, map[string]string{"src/main.can": strings.Replace(programHeader, "uses []", "uses [codec]", 1) + coordinationDeclarations + programMain + body + "    ok\n"})
}

func TestCoordinationAggregateComposition(t *testing.T) {
	data, err := os.ReadFile("../../testdata/current/coordination/aggregate-composition.can")
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	if _, err := programFixture(t, map[string]string{"src/main.can": source}); err != nil {
		t.Fatal(err)
	}
	bad := strings.Replace(source, "        normalized_a()\n        normalized_b()", "        original_a()\n        original_b()", 1)
	if _, err := programFixture(t, map[string]string{"src/main.can": bad}); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("distinct aggregate specializations admitted under one bare arm: %v", err)
	}
	bad = strings.Replace(source, "            combined_failure[] values = call all_failed.failures.map(callable widen_one)", "            combined_failure[] values = all_failed.failures", 1)
	if _, err := programFixture(t, map[string]string{"src/main.can": bad}); err == nil {
		t.Fatal("aggregate normalization admitted array covariance")
	}
	source += `fn int inspect_nested
    emits []
    given
        all_failed<combined_failure> value
    asserts
        empty: all_failed<combined_failure>([]) => ok 0
    match value
        all_failed => ok all_failed.failures.length
`
	if _, err := programFixture(t, map[string]string{"src/main.can": source}); err != nil {
		t.Fatalf("bare aggregate data pattern did not install its implicit alias: %v", err)
	}
	bad = strings.Replace(source, "    match value\n        all_failed => ok all_failed.failures.length", "    int all_failed = 0\n    ok all_failed", 1)
	if _, err := programFixture(t, map[string]string{"src/main.can": bad}); err == nil {
		t.Fatal("authored local shadowed reserved aggregate name")
	}
}

func TestMapAggregateConversionRejections(t *testing.T) {
	source := exactComposition(t)
	narrow := source + `variant narrow_failure
    codec::invalid_data
fn narrow_failure bad_one
    emits []
    given
        a_failure item
    asserts
        data: codec::invalid_data("a", "type") => ok codec::invalid_data("a", "type")
    ok item
`
	if _, err := programFixture(t, map[string]string{"src/main.can": narrow}); err == nil || !strings.Contains(err.Error(), "expression type does not fit expected type") {
		t.Fatalf("converter omitting a leaf admitted: %v", err)
	}
	badInput := strings.Replace(source, "call all_failed.failures.map(callable widen_one)", "call all_failed.failures.map(callable describe)", 1)
	if _, err := programFixture(t, map[string]string{"src/main.can": badInput}); err == nil || !strings.Contains(err.Error(), "array.map: callback inputs must equal element and accumulator types") {
		t.Fatalf("map callback with foreign input admitted: %v", err)
	}
	counting := source + `fn int count_leaves
    emits []
    given
        a_failure item
    asserts
        data: codec::invalid_data("a", "type") => ok 1
    ok 1
`
	badResult := strings.Replace(counting, "call all_failed.failures.map(callable widen_one)", "call all_failed.failures.map(callable count_leaves)", 1)
	if _, err := programFixture(t, map[string]string{"src/main.can": badResult}); err == nil || !strings.Contains(err.Error(), "expression type does not fit expected type") {
		t.Fatalf("map callback with foreign result admitted: %v", err)
	}
}
func TestCheckedConcurrentHandlers(t *testing.T) {
	for _, tc := range []struct{ header, body string }{
		{"concurrent", `        number()
            ok int value => ok value
        text()
            ok str value => ok value.length
        codec::invalid_data => ok []
`},
		{"concurrent with error", `        number()
            codec::invalid_data => ok 0
            ok int value => ok value
        text()
            ok str value => ok value.length
`},
	} {
		t.Run(tc.header, func(t *testing.T) {
			p, err := coordinationProgram(t, "    int[] results = match call "+tc.header+"\n"+tc.body)
			if err != nil {
				t.Fatal(err)
			}
			node := p.Entry.Region.Body.Steps[0].Value
			if node.Kind != ir.CoordinationValue || len(node.Coordination.Entries) != 2 {
				t.Fatal("missing checked participants")
			}
			for _, entry := range node.Coordination.Entries {
				if entry.Handler.Region.Kind != ir.HandlerRegion || entry.Handler.Region.Parent != p.Entry.Region.ID {
					t.Fatal("handler not local to coordination")
				}
				if entry.Handler.Arms[0].Body.RegionID != entry.Handler.Region.ID {
					t.Fatal("handler completion escaped to outer function")
				}
			}
		})
	}
}
func TestCheckedRaceAndSpread(t *testing.T) {
	body := `    callable int () emits [codec::invalid_data][] operations = [callable number]
    int result = match call race with error
        ...operations
        number()
        codec::invalid_data => ok 0
        ok int value => ok value
`
	if _, err := coordinationProgram(t, body); err != nil {
		t.Fatal(err)
	}
	bad := strings.Replace(body, "        number()", "        text()", 1)
	if _, err := coordinationProgram(t, bad); err == nil || !strings.Contains(err.Error(), "identical") {
		t.Fatalf("heterogeneous race accepted: %v", err)
	}
	bad = strings.Replace(body, "callable int () emits [codec::invalid_data][] operations = [callable number]", "callable int (int) emits [][] operations = [callable input]", 1)
	if _, err := coordinationProgram(t, bad); err == nil || !strings.Contains(err.Error(), "nullary") {
		t.Fatalf("non-nullary spread accepted: %v", err)
	}
}

func TestCoordinationSpreadUsesDeclaredCommonBound(t *testing.T) {
	prefix := strings.Replace(programHeader, "uses []", "uses [codec]", 1) + coordinationDeclarations + `fn int pure
    emits []
    asserts
        sample: => ok 1
    ok 1
` + programMain
	body := `    callable int () emits [codec::invalid_data][] operations = [callable pure]
    int result = match call race with error
        ...operations
        codec::invalid_data => ok 0
        ok int value => ok value
    ok
`
	if _, err := programFixture(t, map[string]string{"src/main.can": prefix + body}); err != nil {
		t.Fatal(err)
	}
	for name, bad := range map[string]string{
		"missing common bound arm": strings.Replace(body, "        codec::invalid_data => ok 0\n", "", 1),
		"different success":        strings.Replace(body, "[callable pure]", "[callable text]", 1),
		"remaining input":          strings.Replace(body, "[callable pure]", "[callable input]", 1),
		"error outside bound":      strings.Replace(strings.Replace(body, "emits [codec::invalid_data][]", "emits [][]", 1), "[callable pure]", "[callable number]", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := programFixture(t, map[string]string{"src/main.can": prefix + bad}); err == nil {
				t.Fatal("invalid callable spread admitted")
			}
		})
	}
}

func TestCoordinationHeterogeneousRecords(t *testing.T) {
	data, err := os.ReadFile("../../testdata/current/coordination/heterogeneous.can")
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	if _, err := programFixture(t, map[string]string{"src/main.can": source}); err != nil {
		t.Fatal(err)
	}
	bad := strings.Replace(source, "        normalized_left()\n        normalized_right()", "        make_left()\n        make_right()", 1)
	if _, err := programFixture(t, map[string]string{"src/main.can": bad}); err == nil || !strings.Contains(err.Error(), "identical") {
		t.Fatalf("heterogeneous record race admitted: %v", err)
	}
}
func TestCoordinationCoverageAndHandlerContracts(t *testing.T) {
	good := `    int[] results = match call concurrent
        number()
            ok int value => ok value
        codec::invalid_data => ok []
`
	for name, bad := range map[string]string{
		"missing shared domain arm": strings.Replace(good, "        codec::invalid_data => ok []\n", "", 1),
		"wrong whole fallback":      strings.Replace(good, "codec::invalid_data => ok []", "codec::invalid_data => ok 0", 1),
		"wrong element result":      strings.Replace(good, "ok int value => ok value", "ok int value => ok [value]", 1),
		"handler failure escapes":   strings.Replace(good, "ok int value => ok value", `ok int value => codec::invalid_data("", "type")`, 1),
		"no success arm":            strings.Replace(good, "            ok int value => ok value\n", "", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := coordinationProgram(t, bad); err == nil {
				t.Fatal("invalid coordination admitted")
			}
		})
	}
}

func TestCheckedFirstSuccessRaceFallback(t *testing.T) {
	good := `    int result = match call race
        number()
        ok int value => ok value
        all_failed => ok 0
`
	p, err := coordinationProgram(t, good)
	if err != nil {
		t.Fatal(err)
	}
	node := p.Entry.Region.Body.Steps[0].Value.Coordination
	if node.Mode != "any" || node.Aggregate == nil || node.Shared == nil {
		t.Fatal("missing first-success handlers")
	}
	for name, bad := range map[string]string{
		"missing aggregate":   strings.Replace(good, "        all_failed => ok 0\n", "", 1),
		"participant error":   strings.Replace(good, "        all_failed => ok 0", "        codec::invalid_data => ok 0", 1),
		"standard arm":        good + "        [_] => ok 0\n",
		"duplicate aggregate": good + "        all_failed => ok 0\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := coordinationProgram(t, bad); err == nil {
				t.Fatal("invalid race arms accepted")
			}
		})
	}
}

func TestForwardedAggregateCoverage(t *testing.T) {
	declaration := `variant failure
    codec::invalid_data
    standard_failure
fn int aggregate
    emits [all_failed<failure>]
    asserts
        sample: => ok 1
    int result = match call race
        number()
        ok int value => ok value
        all_failed
    ok result
`
	source := strings.Replace(programHeader, "uses []", "uses [codec]", 1) + coordinationDeclarations + declaration + programMain + "    ok\n"
	if _, err := programFixture(t, map[string]string{"src/main.can": source}); err != nil {
		t.Fatal(err)
	}
	for name, bad := range map[string]string{
		"missing standard snapshot": strings.Replace(source, "    standard_failure\n", "", 1),
		"missing declared domain":   strings.Replace(source, "variant failure\n    codec::invalid_data\n", "variant failure\n", 1),
		"missing forwarded bound":   strings.Replace(source, "emits [all_failed<failure>]", "emits []", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := programFixture(t, map[string]string{"src/main.can": bad}); err == nil {
				t.Fatal("invalid aggregate forwarding admitted")
			}
		})
	}
}

func TestObservedAggregateInference(t *testing.T) {
	declaration := `variant failure
    codec::invalid_data
    standard_failure
variant alternate
    codec::invalid_data
    standard_failure
fn int summarize
    emits []
    given
        failure[] failures
    asserts
        sample: [] => ok 0
    ok failures.length
fn int other
    emits []
    given
        alternate[] failures
    asserts
        sample: [] => ok 0
    ok failures.length
`
	prefix := strings.Replace(programHeader, "uses []", "uses [codec]", 1) + coordinationDeclarations + declaration + programMain
	body := `    int result = match call race
        number()
        all_failed => do
            int count = all_failed.failures.length
            int summarized = call summarize(all_failed.failures)
            ok count + summarized
        ok int value => ok value
    ok
`
	p, err := programFixture(t, map[string]string{"src/main.can": prefix + body})
	if err != nil {
		t.Fatal(err)
	}
	node := p.Entry.Region.Body.Steps[0].Value.Coordination
	if node.AggregateType == nil || node.AggregateType.Arguments()[0].Kind() != types.Variant {
		t.Fatal("aggregate projection did not supply sealed variant")
	}
	ordered := strings.Replace(body, "        all_failed => do\n            int count = all_failed.failures.length\n            int summarized = call summarize(all_failed.failures)\n            ok count + summarized", "        all_failed => ok all_failed.failures.length + call summarize(all_failed.failures)", 1)
	if _, err := programFixture(t, map[string]string{"src/main.can": prefix + ordered}); err != nil {
		t.Fatalf("aggregate inference depended on operand order: %v", err)
	}
	invalidOperand := strings.Replace(ordered, " + call summarize(all_failed.failures)", " + call summarize(all_failed.failures) + \"wrong\"", 1)
	if _, err := programFixture(t, map[string]string{"src/main.can": prefix + invalidOperand}); err == nil {
		t.Fatal("discovery placeholder bypassed the full operand check")
	}
	outer := `fn int outer_failure
    emits [all_failed<alternate>]
    asserts
        empty: => all_failed<alternate>([])
    all_failed<alternate>([])
`
	nested := `    match call outer_failure()
        all_failed => do
            int result = match call race
                number()
                all_failed => ok call summarize(all_failed.failures)
                ok int value => ok value
            ok
        ok int value => ok
`
	if _, err := programFixture(t, map[string]string{"src/main.can": strings.Replace(prefix, programMain, outer+programMain, 1) + nested}); err != nil {
		t.Fatalf("outer aggregate alias captured nested race alias: %v", err)
	}
	for name, bad := range map[string]string{
		"ambiguous observation":         strings.Replace(body, "            int summarized = call summarize(all_failed.failures)", "            int summarized = 0", 1),
		"conflicting expected variants": strings.Replace(body, "            ok count + summarized", "            int conflicting = call other(all_failed.failures)\n            ok count + summarized + conflicting", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := programFixture(t, map[string]string{"src/main.can": prefix + bad}); err == nil {
				t.Fatal("invalid aggregate inference admitted")
			}
		})
	}
}

func exactComposition(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("../../testdata/current/coordination/aggregate-composition.can")
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestExactHeadRejections(t *testing.T) {
	source := exactComposition(t)
	cases := map[string]struct {
		edits [][2]string
		want  string
	}{
		"ambiguous bare": {
			edits: [][2]string{{"        all_failed<a_failure> as first => ok \"a\"\n        all_failed<b_failure> as second => ok \"b\"\n", "        all_failed => ok \"a\"\n"}},
			want:  "ambiguous",
		},
		"duplicate bare/exact": {
			edits: [][2]string{{"        all_failed<a_failure>\n", "        all_failed<a_failure>\n        all_failed\n"}},
			want:  "duplicate",
		},
		"missing specialization": {
			edits: [][2]string{{"        all_failed<b_failure> as second => ok \"b\"\n", ""}},
			want:  "missing completion arm",
		},
		"wrong arity": {
			edits: [][2]string{{"all_failed<a_failure> as first", "all_failed<a_failure, b_failure> as first"}},
			want:  "arity",
		},
		"absent key": {
			edits: [][2]string{{"all_failed<a_failure> as first", "nope<int> as first"}},
			want:  "no eligible declaration",
		},
		"wrong specialization": {
			edits: [][2]string{{"        all_failed<a_failure>\n", "        all_failed<b_failure>\n"}},
			want:  "outside matched bound",
		},
		"incomplete outer": {
			edits: [][2]string{
				{"variant outer_failure\n    all_failed<a_failure>\n    all_failed<b_failure>\n    standard_failure\n", "variant outer_failure\n    all_failed<a_failure>\n    standard_failure\n"},
				{", all_failed<b_failure>([codec::invalid_data(\"b\", \"type\")])", ""},
			},
			want: "does not cover",
		},
		"participant arm at outer race": {
			edits: [][2]string{{"        all_failed<outer_failure> as agg => all_failed<outer_failure>(agg.failures)\n", "        codec::invalid_data => ok 0\n"}},
			want:  "only handles success and all_failed",
		},
		"wrong-specification fixture": {
			edits: [][2]string{{"sub_a: 1 => all_failed<a_failure>([codec::invalid_data(\"a\", \"type\")])", "sub_a: 1 => all_failed<combined_failure>([codec::invalid_data(\"a\", \"type\")])"}},
			want:  "undeclared escaping domain error",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			bad := source
			for _, edit := range tc.edits {
				next := strings.Replace(bad, edit[0], edit[1], 1)
				if next == bad {
					t.Fatalf("mutation missed: %q", edit[0])
				}
				bad = next
			}
			_, err := programFixture(t, map[string]string{"src/main.can": bad})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("accepted or misdiagnosed: %v", err)
			}
		})
	}
}

func TestAmbiguousBareHeadCarriesSpan(t *testing.T) {
	text := exactComposition(t)
	bad := strings.Replace(text, "        all_failed<a_failure> as first => ok \"a\"\n        all_failed<b_failure> as second => ok \"b\"\n", "        all_failed => ok \"a\"\n", 1)
	_, err := programFixture(t, map[string]string{"src/main.can": bad})
	if err == nil {
		t.Fatal("ambiguous bare head admitted")
	}
	located, ok := source.AsLocated(err)
	if !ok || located.Span.Start == 0 || located.Span.End <= located.Span.Start {
		t.Fatalf("ambiguity lost its head span: %v", err)
	}
	if !strings.Contains(located.File, "main.can") {
		t.Fatalf("ambiguity misattributed: %v", err)
	}
	if located.Code != "CAN-CHECK-EXACT-SPECIALIZATION" {
		t.Fatalf("ambiguity lost its code: %v", err)
	}
	if len(located.Related) != 1 || !strings.Contains(located.Related[0].Note, "matched bound") {
		t.Fatalf("ambiguity lost its bound link: %+v", located.Related)
	}
	if len(located.Fixes) != 0 {
		t.Fatalf("ambiguity proposed fixes: %+v", located.Fixes)
	}
	for _, alternative := range []string{"all_failed<a_failure>", "all_failed<b_failure>"} {
		if !strings.Contains(err.Error(), alternative) {
			t.Fatalf("ambiguity hides %s: %v", alternative, err)
		}
	}
}

func TestDataMatchBareAmbiguityListsAlternatives(t *testing.T) {
	text := programHeader + `variant a_failure
    codec::invalid_data
    standard_failure
variant b_failure
    codec::invalid_data
    standard_failure
variant both
    all_failed<a_failure>
    all_failed<b_failure>
fn str classify
    emits []
    given
        both value
    asserts
        sample: all_failed<a_failure>([codec::invalid_data("a", "type")]) => ok "a"
    match value
        all_failed => ok "a"
        all_failed<b_failure> => ok "b"
` + programMain + "    ok\n"
	_, err := programFixture(t, map[string]string{"src/main.can": strings.Replace(text, "uses []", "uses [codec]", 1)})
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("data bare head admitted: %v", err)
	}
	if !strings.Contains(err.Error(), "all_failed") {
		t.Fatalf("ambiguity hides alternatives: %v", err)
	}
}

func programFixtureRegistry(t *testing.T, files map[string]string, registry string) (*Program, error) {
	t.Helper()
	files["can.project.json"] = `{"source_root":"src","error_registry":"can.errors.json"}`
	files["can.errors.json"] = registry
	root := t.TempDir()
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
	return CheckProgram(graph)
}

func TestGenericBodyExactHeadPerSpecialization(t *testing.T) {
	text := programHeader + `error 1000010 hold<item>(item value)
fn str first<item>
    emits [hold<item>]
    given
        item value
    asserts
        integer: 1 => hold<int>(1)
    hold<item>(value)
fn str describe<item>
    emits []
    given
        item value
    asserts
        integer: 1 => ok "held"
    match call first<item>(value)
        hold<item> as kept => ok "held"
        ok str value => ok "unexpected"
` + programMain + "    ok\n"
	registry := `{"active":[{"id":1000010,"kind":"app::hold"}],"retired":[]}`
	if _, err := programFixtureRegistry(t, map[string]string{"src/main.can": text}, registry); err != nil {
		t.Fatalf("generic exact head rejected: %v", err)
	}
}
