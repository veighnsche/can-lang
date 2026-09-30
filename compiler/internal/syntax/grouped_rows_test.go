package syntax

import (
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/source"
)

func TestGroupedCompletionHeadsKeepOneBodyAndForwarding(t *testing.T) {
	text := testHeader + `fn bool check
    emits {sql::query_failed, sql::connection_failed, sql::unsupported_value}
    asserts
        sample: => ok false
    match call lookup()
        sql::query_failed | sql::connection_failed => ok false
        sql::unsupported_value
        ok bool found => ok found
`
	file := parseFile(t, text)
	arms := file.Declarations[0].(*FunctionDecl).Body.Terminal.(*MatchBody).Match.Arms
	if len(arms) != 3 || arms[0].Outcome == nil || len(arms[0].AlternateOutcomes) != 1 || arms[0].Body == nil || !arms[1].Forward {
		t.Fatalf("grouped completion arms lost: %+v", arms)
	}
	formatted := Format(file)
	if !strings.Contains(formatted, "sql::query_failed | sql::connection_failed => ok false") {
		t.Fatalf("grouped handler lost in format:\n%s", formatted)
	}
	parseFile(t, formatted)
}

func TestGroupedCompletionHeadsInCoordination(t *testing.T) {
	text := coordinationFunction(`    match call race
        first()
        second()
        sql::query_failed | sql::connection_failed
        ok => ok
`)
	file := parseFile(t, text)
	coord := file.Declarations[0].(*FunctionDecl).Body.Steps[0].(*CoordinationStep).Coordination
	if len(coord.Arms) != 2 || len(coord.Arms[0].AlternateOutcomes) != 1 || !coord.Arms[0].Forward {
		t.Fatalf("coordination grouped forwarding lost: %+v", coord.Arms)
	}
	formatted := Format(file)
	if !strings.Contains(formatted, "sql::query_failed | sql::connection_failed\n") {
		t.Fatalf("coordination group lost in format:\n%s", formatted)
	}
}

func TestGroupedAssertionSelectorsRetainIndependentExpansion(t *testing.T) {
	text := testHeader + `fn void main
    emits {}
    asserts
        absent | configured | sample: "ASSET_ROOT" => ok link setup
    match call lookup()
        when
            absent | sample: use missing("ASSET_ROOT")
            scenario ready | served: "ASSET_ROOT" => ok "/tmp/assets"
        ok => ok
`
	file := parseFile(t, text)
	fn := file.Declarations[0].(*FunctionDecl)
	if len(fn.Assertions) != 1 || len(fn.Assertions[0].AlternateNames) != 2 || len(fn.Assertions[0].Links) != 1 {
		t.Fatalf("assertion group lost: %+v", fn.Assertions)
	}
	roots := ExpandAssertions(fn.Assertions)
	if len(roots) != 3 || roots[0].Name.Text != "absent" || roots[1].Name.Text != "configured" || roots[2].Name.Text != "sample" {
		t.Fatalf("assertion group did not expand by label: %+v", roots)
	}
	for _, root := range roots {
		if len(root.AlternateNames) != 0 || len(root.Links) != 1 || root.Expected == nil || root.Name.Span.Start == roots[0].Name.Span.Start && root.Name.Text != "absent" {
			t.Fatalf("expanded assertion lost payload or label span: %+v", root)
		}
	}
	when := fn.Body.Terminal.(*MatchBody).Match.When
	if len(when) != 2 || when[0].Use == nil || len(when[0].AlternateNames) != 1 {
		t.Fatalf("grouped template use lost: %+v", when)
	}
	rows := ExpandAssertions(when)
	if len(rows) != 4 || rows[2].Scenario == nil || rows[2].Scenario.Text != "ready" || rows[3].Scenario == nil || rows[3].Scenario.Text != "served" {
		t.Fatalf("grouped when labels lost selector or scenario: %+v", rows)
	}
	formatted := Format(file)
	for _, want := range []string{"absent | configured | sample:", "absent | sample: use", "scenario ready | served:"} {
		if !strings.Contains(formatted, want) {
			t.Fatalf("format lost %q:\n%s", want, formatted)
		}
	}
	parseFile(t, formatted)
}

func TestGroupedRowsPreserveComments(t *testing.T) {
	text := testHeader + `fn void main
    emits {}
    asserts
        a | b: => ok // root group
    match call lookup()
        when
            a | b: => ok // fixture group
        sql::query_failed | sql::connection_failed => ok // error group
        ok => ok
`
	file := parseFile(t, text)
	formatted, err := FormatTrivia(file)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"a | b:  => ok  // root group", "a | b:  => ok  // fixture group", "sql::query_failed | sql::connection_failed => ok  // error group"} {
		if !strings.Contains(formatted, want) {
			t.Fatalf("trivia format lost %q:\n%s", want, formatted)
		}
	}
	parseFile(t, formatted)
}

func TestGroupedCompletionHeadsRejectMixedOutcomesAndAliases(t *testing.T) {
	for _, head := range []string{
		"ok | sql::query_failed",
		"sql::query_failed | ok",
		"ok bool found | sql::query_failed",
		"sql::query_failed | ok bool found",
		"ok bool found | ok int other",
		"[_] | sql::query_failed",
		"sql::query_failed | [_]",
		"[_] as str message | sql::query_failed",
		"sql::query_failed | [_] as str message",
		"sql::query_failed as issue | sql::connection_failed",
		"sql::query_failed | sql::connection_failed as issue",
	} {
		text := testHeader + "fn void main\n    emits {}\n    asserts\n        sample: => ok\n    match call lookup()\n        " + head + " => ok\n        ok => ok\n"
		file, err := source.New("bad.can", text)
		if err != nil {
			t.Fatal(err)
		}
		if result := Parse(file); result.OK() {
			t.Fatalf("accepted mixed completion heads %q", head)
		}
	}
}

func TestGroupedAssertionsRejectDuplicateAndIncompleteLabels(t *testing.T) {
	for _, row := range []string{"a | a: => ok", "a | b | a: => ok", "scenario a | a: => ok", "a | : => ok", "a | => ok", "a | b | : => ok"} {
		text := testHeader + "fn void main\n    emits {}\n    asserts\n        " + row + "\n    ok\n"
		file, err := source.New("bad.can", text)
		if err != nil {
			t.Fatal(err)
		}
		if result := Parse(file); result.OK() {
			t.Fatalf("accepted malformed assertion group %q", row)
		}
	}
}

func TestGroupedCompletionHeadsAcceptGenericSpecializations(t *testing.T) {
	text := testHeader + `fn bool check
    emits {store_failed, load_failed, sql::query_failed, sql::connection_failed}
    asserts
        sample: => ok false
    match call lookup()
        store_failed<int> | load_failed<str> => ok true
        sql::query_failed<int> | sql::connection_failed => ok false
        ok bool found => ok found
`
	file := parseFile(t, text)
	arms := file.Declarations[0].(*FunctionDecl).Body.Terminal.(*MatchBody).Match.Arms
	if len(arms) != 3 || len(arms[0].AlternateOutcomes) != 1 || len(arms[1].AlternateOutcomes) != 1 {
		t.Fatalf("generic completion groups lost: %+v", arms)
	}
	first, ok := arms[0].Outcome.Error.(*NamedType)
	if !ok || len(first.Arguments) != 1 {
		t.Fatalf("generic head lost its specialization: %+v", arms[0].Outcome)
	}
	second, ok := arms[0].AlternateOutcomes[0].Error.(*NamedType)
	if !ok || len(second.Arguments) != 1 {
		t.Fatalf("generic alternative lost its specialization: %+v", arms[0].AlternateOutcomes[0])
	}
	plain, ok := arms[1].AlternateOutcomes[0].Error.(*NamedType)
	if !ok || len(plain.Arguments) != 0 {
		t.Fatalf("mixed generic/plain group lost its plain member: %+v", arms[1].AlternateOutcomes[0])
	}
	formatted := Format(file)
	for _, want := range []string{"store_failed<int> | load_failed<str> => ok true", "sql::query_failed<int> | sql::connection_failed => ok false"} {
		if !strings.Contains(formatted, want) {
			t.Fatalf("generic spelling lost in format %q:\n%s", want, formatted)
		}
	}
	parseFile(t, formatted)
}

func TestGroupedCompletionHeadsDeferDuplicateHeadsToChecking(t *testing.T) {
	text := testHeader + `fn bool check
    emits {sql::query_failed}
    asserts
        sample: => ok false
    match call lookup()
        sql::query_failed | sql::query_failed => ok false
        ok => ok true
`
	file := parseFile(t, text)
	arms := file.Declarations[0].(*FunctionDecl).Body.Terminal.(*MatchBody).Match.Arms
	if len(arms) != 2 || len(arms[0].AlternateOutcomes) != 1 {
		t.Fatalf("repeated group member lost: %+v", arms)
	}
}

func TestGroupedCompletionHeadsAcrossSurfaces(t *testing.T) {
	chain := testHeader + `fn bool check
    emits {sql::query_failed, sql::connection_failed}
    asserts
        sample: => ok false
    match chain
        call lookup("x") as int value
        call log(value)
        sql::query_failed | sql::connection_failed => ok false
        ok => ok true
`
	chainFile := parseFile(t, chain)
	chainMatch := chainFile.Declarations[0].(*FunctionDecl).Body.Terminal.(*MatchBody).Match
	if chainMatch.Kind != ChainMatch || len(chainMatch.Arms) != 2 || len(chainMatch.Arms[0].AlternateOutcomes) != 1 {
		t.Fatalf("chain completion group lost: %+v", chainMatch.Arms)
	}

	forward := testHeader + `fn bool check
    emits {sql::timeout, sql::cancelled}
    asserts
        sample: => ok false
    match call lookup()
        sql::timeout | sql::cancelled
        ok => ok true
`
	forwardFile := parseFile(t, forward)
	forwardArms := forwardFile.Declarations[0].(*FunctionDecl).Body.Terminal.(*MatchBody).Match.Arms
	if len(forwardArms) != 2 || len(forwardArms[0].AlternateOutcomes) != 1 || !forwardArms[0].Forward || forwardArms[0].Body != nil {
		t.Fatalf("bare grouped forwarding lost: %+v", forwardArms)
	}

	race := testHeader + `fn bool check
    emits {sql::query_failed, sql::connection_failed}
    asserts
        sample: => ok false
    match call race with error
        first()
        second()
        sql::query_failed | sql::connection_failed => ok false
        ok bool found => ok found
    ok
`
	raceFile := parseFile(t, race)
	raceCoord := raceFile.Declarations[0].(*FunctionDecl).Body.Steps[0].(*CoordinationStep).Coordination
	if len(raceCoord.Arms) != 2 || len(raceCoord.Arms[0].AlternateOutcomes) != 1 || raceCoord.Arms[0].Forward {
		t.Fatalf("race completion group lost: %+v", raceCoord.Arms)
	}

	concurrent := testHeader + `fn bool check
    emits {store_failed, load_failed}
    asserts
        sample: => ok false
    match call concurrent with error
        database::save(account)
            ok receipt saved => ok true
            store_failed | load_failed => ok false
        backup::save(account)
            ok receipt saved => ok true
            store_failed | load_failed => ok false
    ok true
`
	concurrentFile := parseFile(t, concurrent)
	participants := concurrentFile.Declarations[0].(*FunctionDecl).Body.Steps[0].(*CoordinationStep).Coordination.Participants
	if len(participants) != 2 {
		t.Fatalf("concurrent participants lost: %+v", participants)
	}
	for _, participant := range participants {
		if len(participant.Arms) != 2 || len(participant.Arms[1].AlternateOutcomes) != 1 || participant.Arms[1].Body == nil {
			t.Fatalf("participant completion group lost: %+v", participant.Arms)
		}
	}
	formatted := Format(concurrentFile)
	if !strings.Contains(formatted, "store_failed | load_failed => ok false") {
		t.Fatalf("participant group lost in format:\n%s", formatted)
	}
}

func TestGroupedCompletionHeadsRejectSeparatorsAndMalformedHeads(t *testing.T) {
	call := func(arm string) string {
		return testHeader + "fn void main\n    emits {}\n    asserts\n        sample: => ok\n    match call lookup()\n        " + arm + "\n        ok => ok\n"
	}
	cases := map[string]string{
		"empty member":                       call("sql::query_failed | => ok"),
		"trailing separator":                 call("sql::query_failed |"),
		"double separator":                   call("sql::query_failed | | sql::connection_failed => ok"),
		"leading separator":                  call("| sql::query_failed => ok"),
		"malformed head":                     call("sql::query_failed | 42 => ok"),
		"plain concurrent participant group": testHeader + "fn void main\n    emits {}\n    asserts\n        sample: => ok\n    match call concurrent\n        first()\n            store_failed | load_failed => ok\n    ok\n",
	}
	for name, text := range cases {
		file, err := source.New("bad.can", text)
		if err != nil {
			t.Fatal(err)
		}
		if result := Parse(file); result.OK() {
			t.Fatalf("accepted %s:\n%s", name, text)
		}
	}
}

func TestGroupedCompletionPreservesSingleArmBehavior(t *testing.T) {
	withValueMatch := testHeader + `fn int pick
    emits {sql::query_failed, sql::timeout}
    asserts
        sample: 1 => ok 1
    int result = match 1
        1 | 2 => 10
        _ => 20
    ok result
`
	valueFile := parseFile(t, withValueMatch)
	valueArms := valueFile.Declarations[0].(*FunctionDecl).Body.Steps[0].(*BindingStep).Binding.Value.(*MatchExpr).Match.Arms
	alternative, ok := valueArms[0].Patterns[0].(*AlternativePattern)
	if !ok || len(alternative.Alternatives) != 2 || valueArms[0].Outcome != nil || len(valueArms[0].AlternateOutcomes) != 0 {
		t.Fatalf("value-pattern alternative changed shape: %+v", valueArms[0])
	}

	withCompletionArms := testHeader + `fn int pick
    emits {sql::query_failed, sql::timeout}
    asserts
        sample: 1 => ok 1
    match call lookup()
        sql::query_failed as issue => ok 1
        ok bool found => ok 2
        [_] as str message => ok 3
        sql::timeout
        ok => ok 4
`
	armsFile := parseFile(t, withCompletionArms)
	arms := armsFile.Declarations[0].(*FunctionDecl).Body.Terminal.(*MatchBody).Match.Arms
	if len(arms) != 5 {
		t.Fatalf("single completion arms lost: %+v", arms)
	}
	for _, arm := range arms {
		if len(arm.AlternateOutcomes) != 0 {
			t.Fatalf("single arm gained alternates: %+v", arm)
		}
	}
	if arms[0].Outcome.Alias == nil || arms[1].Outcome.Binding == nil || !arms[2].Outcome.StandardFailure || !arms[3].Forward {
		t.Fatalf("single-arm payloads lost: %+v", arms)
	}

	groupedKey := testHeader + "wrap cached_load from load_json\n    emits calculated\n    asserts\n        absent: => ok receipt(0)\n    handles native\n        http::status_error | http::timeout => ok receipt(0)\n"
	file, err := source.New("bad.can", groupedKey)
	if err != nil {
		t.Fatal(err)
	}
	if result := Parse(file); result.OK() {
		t.Fatalf("wrapper policy accepted a grouped key:\n%s", groupedKey)
	}
}

func TestGroupedAssertionExpansionRetainsFieldsOrderAndSpans(t *testing.T) {
	text := testHeader + `fn void main
    emits {}
    asserts
        absent | configured: "ASSET_ROOT" => ok "/tmp/x" link setup
            using raw "fixtures/roots.json"
    ok
`
	file := parseFile(t, text)
	rows := file.Declarations[0].(*FunctionDecl).Assertions
	if len(rows) != 1 || len(rows[0].AlternateNames) != 1 {
		t.Fatalf("grouped assertion row lost: %+v", rows)
	}
	roots := ExpandAssertions(rows)
	if len(roots) != 2 || roots[0].Name.Text != "absent" || roots[1].Name.Text != "configured" {
		t.Fatalf("grouped assertion did not expand in source order: %+v", roots)
	}
	body := file.Source.Text()
	previous := -1
	for _, root := range roots {
		if len(root.AlternateNames) != 0 || root.Scenario != nil {
			t.Fatalf("expanded root kept group state: %+v", root)
		}
		if root.Span != rows[0].Span || root.Mode != rows[0].Mode || root.Expected != rows[0].Expected {
			t.Fatalf("expanded root lost the shared row payload: %+v", root)
		}
		if root.Mode == nil || root.Mode.Raw.Value != "fixtures/roots.json" {
			t.Fatalf("expanded root lost its execution mode: %+v", root.Mode)
		}
		if len(root.Links) != 1 || root.Links[0].Name != "setup" || len(root.Arguments) != 1 || root.Expected == nil {
			t.Fatalf("expanded root lost links, arguments or completion: %+v", root)
		}
		span := root.Name.Span
		if span.Start <= previous || body[span.Start:span.End] != root.Name.Text {
			t.Fatalf("expanded root lost its label location: %+v", root.Name)
		}
		previous = span.Start
	}
	if len(rows[0].AlternateNames) != 1 || rows[0].Name.Text != "absent" {
		t.Fatalf("expansion mutated the source row: %+v", rows[0])
	}
	formatted := Format(file)
	if strings.Count(formatted, "absent | configured:") != 1 {
		t.Fatalf("grouped row not printed once:\n%s", formatted)
	}
	for _, want := range []string{`link setup`, `using raw "fixtures/roots.json"`} {
		if !strings.Contains(formatted, want) {
			t.Fatalf("format lost %q:\n%s", want, formatted)
		}
	}
	parseFile(t, formatted)
}

func TestGroupedAssertionExpansionIsIdempotentAndImmutable(t *testing.T) {
	text := testHeader + `fn void main
    emits {}
    asserts
        one: => ok
        two | three: => ok
    match call lookup()
        when
            scenario ready | served: "k" => ok "v"
            solo: "k" => ok "v"
        ok => ok
`
	file := parseFile(t, text)
	fn := file.Declarations[0].(*FunctionDecl)
	roots := ExpandAssertions(fn.Assertions)
	if len(roots) != 3 || roots[0].Name.Text != "one" || roots[1].Name.Text != "two" || roots[2].Name.Text != "three" {
		t.Fatalf("assertion roots miscounted: %+v", roots)
	}
	again := ExpandAssertions(roots)
	if len(again) != len(roots) {
		t.Fatalf("assertion expansion multiplied rows: %d into %d", len(roots), len(again))
	}
	for i := range again {
		if again[i].Name.Text != roots[i].Name.Text || len(again[i].AlternateNames) != 0 {
			t.Fatalf("assertion re-expansion changed identity: %+v", again[i])
		}
	}
	match := fn.Body.Terminal.(*MatchBody).Match
	rows := ExpandAssertions(match.When)
	if len(rows) != 3 || rows[0].Name.Text != "ready" || rows[1].Name.Text != "served" || rows[2].Name.Text != "solo" {
		t.Fatalf("fixture selectors miscounted: %+v", rows)
	}
	rowsAgain := ExpandAssertions(rows)
	if len(rowsAgain) != len(rows) {
		t.Fatalf("fixture expansion multiplied rows: %d into %d", len(rows), len(rowsAgain))
	}
	for i := range rowsAgain {
		if rowsAgain[i].Name.Text != rows[i].Name.Text || len(rowsAgain[i].AlternateNames) != 0 {
			t.Fatalf("fixture re-expansion changed identity: %+v", rowsAgain[i])
		}
	}
	for _, row := range rows[:2] {
		if row.Scenario == nil || row.Scenario.Text != row.Name.Text {
			t.Fatalf("scenario selector lost its independent tag: %+v", row)
		}
	}
	if rows[2].Scenario != nil {
		t.Fatalf("plain selector gained a scenario tag: %+v", rows[2])
	}
	if len(fn.Assertions) != 2 || len(fn.Assertions[1].AlternateNames) != 1 || len(match.When) != 2 || len(match.When[0].AlternateNames) != 1 {
		t.Fatalf("expansion mutated the source rows")
	}
}

func TestGroupedWhenRowsRetainModes(t *testing.T) {
	text := testHeader + `fn void main
    emits {}
    asserts
        smoke: => ok
    match call lookup()
        when
            flaky | slow: "k" => ok "v"
                using failure emitted io::timeout{"k"}
        ok => ok
`
	file := parseFile(t, text)
	when := file.Declarations[0].(*FunctionDecl).Body.Terminal.(*MatchBody).Match.When
	if len(when) != 1 || len(when[0].AlternateNames) != 1 {
		t.Fatalf("grouped when row lost: %+v", when)
	}
	rows := ExpandAssertions(when)
	if len(rows) != 2 || rows[0].Name.Text != "flaky" || rows[1].Name.Text != "slow" {
		t.Fatalf("grouped when row did not expand: %+v", rows)
	}
	for _, row := range rows {
		if row.Mode == nil || row.Mode.Failure == nil || row.Mode.Failure.Origin.Text != "emitted" {
			t.Fatalf("expanded fixture lost its failure mode: %+v", row.Mode)
		}
	}
	formatted := Format(file)
	for _, want := range []string{`flaky | slow: "k" => ok "v"`, `using failure emitted io::timeout{"k"}`} {
		if strings.Count(formatted, want) != 1 {
			t.Fatalf("format did not print %q exactly once:\n%s", want, formatted)
		}
	}
	parseFile(t, formatted)
}

func TestGroupedRowsFormatRoundTrip(t *testing.T) {
	text := testHeader + `fn bool check
    emits {sql::query_failed, sql::connection_failed, store_failed, load_failed, sql::timeout}
    asserts
        absent | configured: "k" => ok "v" link setup
            using raw "fixtures/roots.json"
    match call lookup()
        sql::query_failed | sql::connection_failed => do
            call log("retry")
            ok false
        store_failed<int> | load_failed<str> => ok true
        sql::timeout
        ok bool found => ok found
`
	file := parseFile(t, text)
	arms := file.Declarations[0].(*FunctionDecl).Body.Terminal.(*MatchBody).Match.Arms
	if len(arms) != 4 {
		t.Fatalf("grouped arms miscounted: %+v", arms)
	}
	shared, ok := arms[0].Body.(*DoBody)
	if !ok || len(arms[0].AlternateOutcomes) != 1 || len(shared.Block.Steps) != 1 {
		t.Fatalf("grouped handler lost its single body: %+v", arms[0])
	}
	if len(arms[1].AlternateOutcomes) != 1 || !arms[2].Forward || len(arms[3].AlternateOutcomes) != 0 {
		t.Fatalf("grouped arm shapes lost: %+v", arms)
	}
	formatted := Format(file)
	for _, want := range []string{
		"sql::query_failed | sql::connection_failed => do",
		"store_failed<int> | load_failed<str> => ok true",
		`absent | configured: "k" => ok "v" link setup`,
		`using raw "fixtures/roots.json"`,
	} {
		if strings.Count(formatted, want) != 1 {
			t.Fatalf("format did not print %q exactly once:\n%s", want, formatted)
		}
	}
	configuredLines := 0
	for _, line := range strings.Split(formatted, "\n") {
		if strings.Contains(line, "configured") {
			configuredLines++
		}
	}
	if configuredLines != 1 {
		t.Fatalf("expanded label leaked into source format:\n%s", formatted)
	}
	reparsed := parseFile(t, formatted)
	rearms := reparsed.Declarations[0].(*FunctionDecl).Body.Terminal.(*MatchBody).Match.Arms
	if len(rearms) != 4 || len(rearms[0].AlternateOutcomes) != 1 || len(rearms[1].AlternateOutcomes) != 1 {
		t.Fatalf("reparsed groups lost: %+v", rearms)
	}
}

func TestGroupedRowsPreserveSingleHeadAndLabel(t *testing.T) {
	text := testHeader + `fn str read
    emits {}
    asserts
        unit: => ok "fixture"
    match call text::from_int(7)
        when
            solo: 7 => ok "fixture"
            scenario checkout: 7 => ok "fixture"
            templated: use absent_receipt("r-7")
        sql::query_failed as issue => ok "none"
        ok str result => ok result
`
	file := parseFile(t, text)
	fn := file.Declarations[0].(*FunctionDecl)
	for _, row := range fn.Assertions {
		if len(row.AlternateNames) != 0 {
			t.Fatalf("single assertion gained alternates: %+v", row)
		}
	}
	match := fn.Body.Terminal.(*MatchBody).Match
	for _, row := range match.When {
		if len(row.AlternateNames) != 0 {
			t.Fatalf("single when row gained alternates: %+v", row)
		}
	}
	for _, arm := range match.Arms {
		if len(arm.AlternateOutcomes) != 0 {
			t.Fatalf("single completion head gained alternates: %+v", arm)
		}
	}
	if match.When[1].Scenario == nil || match.When[2].Use == nil {
		t.Fatalf("single scenario/use rows lost: %+v", match.When)
	}
	formatted := Format(file)
	if strings.Contains(formatted, "|") {
		t.Fatalf("single-head format gained a separator:\n%s", formatted)
	}
	solo := ExpandAssertions(match.When)
	if len(solo) != 3 {
		t.Fatalf("single-label expansion changed row count: %+v", solo)
	}
}
