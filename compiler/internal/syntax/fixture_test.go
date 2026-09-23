package syntax

import (
	"testing"
)

const fixtureSource = `fixture absent_receipt for find_receipt
    given
        str key
    cases
        key => missing_receipt(key)
        "r-9" => missing_receipt("r-9")
            using raw "fixtures/absent.json"
`

func TestFixtureDeclarationParsing(t *testing.T) {
	result := nativeParse(t, fixtureSource)
	if !result.OK() {
		t.Fatal(result.Diagnostics)
	}
	decl, ok := result.File.Declarations[0].(*FixtureDecl)
	if !ok {
		t.Fatalf("fixture parsed as %T", result.File.Declarations[0])
	}
	if decl.Name.Text != "absent_receipt" || decl.Target.Name != "find_receipt" || len(decl.Types) != 0 {
		t.Fatal("fixture header lost its name or target")
	}
	if len(decl.Given) != 1 || decl.Given[0].Name.Text != "key" {
		t.Fatal("fixture given lost its parameter")
	}
	if len(decl.Cases) != 2 {
		t.Fatal("fixture cases missing")
	}
	if len(decl.Cases[0].Arguments) != 1 || decl.Cases[0].Mode != nil {
		t.Fatal("first case lost its shape")
	}
	raw := decl.Cases[1].Mode
	if raw == nil || raw.Failure != nil || raw.Raw.Value != "fixtures/absent.json" {
		t.Fatal("case using raw mode missing")
	}
	formatted := Format(result.File)
	for _, want := range []string{
		"fixture absent_receipt for find_receipt",
		"str key",
		"cases",
		`key => missing_receipt(key)`,
		`using raw "fixtures/absent.json"`,
	} {
		if !contains(formatted, want) {
			t.Fatalf("formatted output omits %q:\n%s", want, formatted)
		}
	}
}

func TestFixtureGenericTargetParsing(t *testing.T) {
	result := nativeParse(t, "fixture first_box for take_box<int>\n    cases\n        1 => ok 1\n")
	if !result.OK() {
		t.Fatal(result.Diagnostics)
	}
	decl := result.File.Declarations[0].(*FixtureDecl)
	if decl.Target.Name != "take_box" || len(decl.Types) != 1 {
		t.Fatalf("generic target lost its specialization: %+v", decl.Target)
	}
}

func TestUseRowParsing(t *testing.T) {
	text := "fn receipt fetch_cached\n    emits [http::request_failed]\n    given\n        str requested\n    asserts\n        sample: \"r-7\" => ok receipt(0)\n    match call find_receipt(requested)\n        when\n            sample: use absent_receipt(\"r-7\")\n        missing_receipt => ok receipt(0)\n        ok receipt found => ok found\n"
	result := nativeParse(t, text)
	if !result.OK() {
		t.Fatal(result.Diagnostics)
	}
	fn := result.File.Declarations[0].(*FunctionDecl)
	match := fn.Body.Terminal.(*MatchBody).Match
	if len(match.When) != 1 || match.When[0].Use == nil {
		t.Fatalf("when use row missing: %+v", match.When)
	}
	use := match.When[0].Use
	if use.Template.Name != "absent_receipt" || len(use.Arguments) != 1 {
		t.Fatal("use row lost its template or arguments")
	}
	if match.When[0].Expected != nil || match.When[0].Mode != nil {
		t.Fatal("use row carries completion content")
	}
	formatted := Format(result.File)
	if !contains(formatted, `sample: use absent_receipt("r-7")`) {
		t.Fatalf("formatted output omits use row:\n%s", formatted)
	}
}

func TestFixtureDeclarationRejects(t *testing.T) {
	cases := map[string]string{
		"missing cases":     "fixture absent_receipt for find_receipt\n    given\n        str key\n",
		"empty cases":       "fixture absent_receipt for find_receipt\n    cases\n",
		"missing target":    "fixture absent_receipt\n    cases\n        1 => ok 1\n",
		"spread case input": "fixture absent_receipt for find_receipt\n    cases\n        ...key => ok 1\n",
		"use with mode":     "fn receipt fetch_cached\n    emits []\n    asserts\n        sample: => ok receipt(0)\n    match call find_receipt(\"r-7\")\n        when\n            sample: use absent_receipt(\"r-7\")\n                using raw \"fixtures/absent.json\"\n        ok receipt found => ok found\n",
	}
	for name, text := range cases {
		t.Run(name, func(t *testing.T) {
			if nativeParse(t, text).OK() {
				t.Fatalf("invalid fixture admitted: %s", name)
			}
		})
	}
}
