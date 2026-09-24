package syntax

import (
	"testing"
)

func TestScenarioDeclarationParsing(t *testing.T) {
	result := nativeParse(t, "scenario checkout\n")
	if !result.OK() {
		t.Fatal(result.Diagnostics)
	}
	decl, ok := result.File.Declarations[0].(*ScenarioDecl)
	if !ok {
		t.Fatalf("scenario parsed as %T", result.File.Declarations[0])
	}
	if decl.Name.Text != "checkout" {
		t.Fatal("scenario lost its name")
	}
	formatted := Format(result.File)
	if !contains(formatted, "scenario checkout") {
		t.Fatalf("formatted output omits scenario marker:\n%s", formatted)
	}
}

func TestScenarioRowParsing(t *testing.T) {
	text := "fn str read\n    emits []\n    asserts\n        unit: => ok \"fixture\"\n    match call text::from_int(7)\n        when\n            scenario checkout: 7 => ok \"fixture\"\n            unit: 7 => ok \"fixture\"\n        ok str result => ok result\n"
	result := nativeParse(t, text)
	if !result.OK() {
		t.Fatal(result.Diagnostics)
	}
	fn := result.File.Declarations[0].(*FunctionDecl)
	match := fn.Body.Terminal.(*MatchBody).Match
	if len(match.When) != 2 {
		t.Fatalf("when rows missing: %+v", match.When)
	}
	tagged := match.When[0]
	if tagged.Scenario == nil || tagged.Scenario.Text != "checkout" || tagged.Name.Text != "checkout" {
		t.Fatalf("scenario tag lost: %+v", tagged)
	}
	if len(tagged.Arguments) != 1 || tagged.Expected == nil {
		t.Fatal("tagged row lost its inputs or completion")
	}
	plain := match.When[1]
	if plain.Scenario != nil || plain.Name.Text != "unit" {
		t.Fatalf("plain row gained a scenario tag: %+v", plain)
	}
	formatted := Format(result.File)
	for _, want := range []string{`scenario checkout: 7 => ok "fixture"`, `unit: 7 => ok "fixture"`} {
		if !contains(formatted, want) {
			t.Fatalf("formatted output omits %q:\n%s", want, formatted)
		}
	}
	reparsed := nativeParse(t, formatted[len(nativePackage):])
	if !reparsed.OK() {
		t.Fatalf("formatted scenario rows do not reparse: %v", reparsed.Diagnostics)
	}
}

func TestScenarioUseRowParsing(t *testing.T) {
	text := "fn str read\n    emits []\n    asserts\n        unit: => ok \"fixture\"\n    match call find_receipt(\"r-7\")\n        when\n            scenario checkout: use absent_receipt(\"r-7\")\n        ok str found => ok found\n"
	result := nativeParse(t, text)
	if !result.OK() {
		t.Fatal(result.Diagnostics)
	}
	fn := result.File.Declarations[0].(*FunctionDecl)
	row := fn.Body.Terminal.(*MatchBody).Match.When[0]
	if row.Scenario == nil || row.Scenario.Text != "checkout" || row.Use == nil {
		t.Fatalf("scenario use row lost its tag or template: %+v", row)
	}
	if row.Use.Template.Name != "absent_receipt" || len(row.Use.Arguments) != 1 {
		t.Fatal("scenario use row lost its template or arguments")
	}
	formatted := Format(result.File)
	if !contains(formatted, `scenario checkout: use absent_receipt("r-7")`) {
		t.Fatalf("formatted output omits scenario use row:\n%s", formatted)
	}
}

func TestScenarioSelectorKeepsPlainMeaning(t *testing.T) {
	text := "fn str read\n    emits []\n    asserts\n        scenario: => ok \"fixture\"\n    match call text::from_int(7)\n        when\n            scenario: 7 => ok \"fixture\"\n        ok str result => ok result\n"
	result := nativeParse(t, text)
	if !result.OK() {
		t.Fatal(result.Diagnostics)
	}
	fn := result.File.Declarations[0].(*FunctionDecl)
	if fn.Assertions[0].Scenario != nil || fn.Assertions[0].Name.Text != "scenario" {
		t.Fatalf("asserts row named scenario misread: %+v", fn.Assertions[0])
	}
	row := fn.Body.Terminal.(*MatchBody).Match.When[0]
	if row.Scenario != nil || row.Name.Text != "scenario" {
		t.Fatalf("when row named scenario misread as tag: %+v", row)
	}
}

func TestLinkClauseParsing(t *testing.T) {
	text := "fn str read_customer\n    emits []\n    asserts\n        customer: => ok \"fixture\" link helper::checkout\n        pair: => ok \"both\" link helper::checkout, retry\n        bare: => ok link helper::checkout\n    ok call helper::read()\n"
	result := nativeParse(t, text)
	if !result.OK() {
		t.Fatal(result.Diagnostics)
	}
	fn := result.File.Declarations[0].(*FunctionDecl)
	if len(fn.Assertions) != 3 {
		t.Fatalf("asserts rows missing: %+v", fn.Assertions)
	}
	first := fn.Assertions[0]
	if len(first.Links) != 1 || first.Links[0].Package != "helper" || first.Links[0].Name != "checkout" {
		t.Fatalf("qualified link lost: %+v", first.Links)
	}
	second := fn.Assertions[1]
	if len(second.Links) != 2 || second.Links[0].Package != "helper" || second.Links[1].Package != "" || second.Links[1].Name != "retry" {
		t.Fatalf("link list lost: %+v", second.Links)
	}
	third := fn.Assertions[2]
	if len(third.Links) != 1 {
		t.Fatalf("bare ok link lost: %+v", third)
	}
	if success, ok := third.Expected.(*SuccessBody); !ok || success.Value != nil {
		t.Fatalf("bare ok link gained a value: %+v", third.Expected)
	}
	formatted := Format(result.File)
	for _, want := range []string{
		`customer:  => ok "fixture" link helper::checkout`,
		`pair:  => ok "both" link helper::checkout, retry`,
		`bare:  => ok link helper::checkout`,
	} {
		if !contains(formatted, want) {
			t.Fatalf("formatted output omits %q:\n%s", want, formatted)
		}
	}
	reparsed := nativeParse(t, formatted[len(nativePackage):])
	if !reparsed.OK() {
		t.Fatalf("formatted links do not reparse: %v", reparsed.Diagnostics)
	}
}

func TestLinkValueNamedLinkKeepsMeaning(t *testing.T) {
	source := "fn str read\n    emits []\n    given\n        str link\n    asserts\n        sample: \"x\" => ok link\n    ok link\n"
	result := nativeParse(t, source)
	if !result.OK() {
		t.Fatal(result.Diagnostics)
	}
	row := result.File.Declarations[0].(*FunctionDecl).Assertions[0]
	if len(row.Links) != 0 {
		t.Fatalf("value named link misread as clause: %+v", row.Links)
	}
	success, ok := row.Expected.(*SuccessBody)
	if !ok || success.Value == nil {
		t.Fatalf("value named link lost: %+v", row.Expected)
	}
}

func TestScenarioLinkRejects(t *testing.T) {
	cases := map[string]string{
		"missing name":   "scenario\n",
		"missing colon":  "fn str read\n    emits []\n    asserts\n        unit: => ok \"x\"\n    match call text::from_int(7)\n        when\n            scenario checkout 7 => ok \"x\"\n        ok str result => ok result\n",
		"missing target": "fn str read\n    emits []\n    asserts\n        customer: => ok \"x\" link\n    ok \"x\"\n",
		"dangling comma": "fn str read\n    emits []\n    asserts\n        customer: => ok \"x\" link helper::checkout,\n    ok \"x\"\n",
		"use with link":  "fn str read\n    emits []\n    asserts\n        unit: => ok \"x\"\n    match call find(1)\n        when\n            sample: use t(1) link helper::checkout\n        ok str r => ok r\n",
	}
	for name, text := range cases {
		t.Run(name, func(t *testing.T) {
			if nativeParse(t, text).OK() {
				t.Fatalf("invalid scenario form admitted: %s", name)
			}
		})
	}
}
