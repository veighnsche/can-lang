package check

import (
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/types"
	"os"
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
	bad = strings.Replace(source, "            combined_failure[] values = call widen_a(all_failed.failures)", "            combined_failure[] values = all_failed.failures", 1)
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
func TestCheckedConcurrentHandlers(t *testing.T) {
	for _, tc := range []struct{ header, body string }{
		{"concurrent", `        number()
            ok int value => ok value
        text()
            ok str value => ok value.length
        codec::invalid_data => ok []
`},
		{"concurrent with error", `        number()
            ok int value => ok value
            codec::invalid_data => ok 0
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
        ok int value => ok value
        codec::invalid_data => ok 0
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
        ok int value => ok value
        codec::invalid_data => ok 0
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
        ok int value => ok value
        all_failed => do
            int count = all_failed.failures.length
            int summarized = call summarize(all_failed.failures)
            ok count + summarized
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
        ok int value => ok
        all_failed => do
            int result = match call race
                number()
                ok int value => ok value
                all_failed => ok call summarize(all_failed.failures)
            ok
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
