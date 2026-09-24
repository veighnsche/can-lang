package check

import (
	"os"
	"strings"
	"testing"
)

func TestChecksCatalogue(t *testing.T) {
	source, err := os.ReadFile("../../testdata/current/checks/main.can")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := programFixture(t, map[string]string{"src/main.can": string(source)}); err != nil {
		t.Fatal(err)
	}
}

func TestChecksRejects(t *testing.T) {
	source, err := os.ReadFile("../../testdata/current/checks/main.can")
	if err != nil {
		t.Fatal(err)
	}
	call := `checks::require(value > 0, "expected a positive value")`
	bad := map[string]string{
		"wrong condition type":              strings.Replace(string(source), call, `checks::require("yes", "expected a positive value")`, 1),
		"wrong reason type":                 strings.Replace(string(source), call, `checks::require(value > 0, 7)`, 1),
		"wrong arity":                       strings.Replace(string(source), call, `checks::require(value > 0)`, 1),
		"missing bound":                     strings.Replace(string(source), "emits [checks::failed]", "emits []", 1),
		"missing failure arm":               strings.Replace(string(source), "        checks::failed => ok false\n", "", 1),
		"missing domain arm with std catch": strings.Replace(string(source), "        checks::failed => ok false\n", "        [_] as standard_failure failure => ok false\n", 1),
		"unchecked errorful":                strings.Replace(string(source), "    relay call "+call, "    call "+call+"\n    ok", 1),
		"literal-true keeps bound":          strings.Replace(string(source), "    relay call "+call, "    call checks::require(true, \"expected a positive value\")\n    ok", 1),
		"unknown operation":                 strings.Replace(string(source), "checks::require(", "checks::ensure(", 1),
		"manufactured std expectation":      strings.Replace(string(source), `zero: 0 => checks::failed("expected a positive value")`, `zero: 0 => standard_failure("boom")`, 1),
	}
	for name, text := range bad {
		t.Run(name, func(t *testing.T) {
			if _, err := programFixture(t, map[string]string{"src/main.can": text}); err == nil {
				t.Fatalf("invalid checks program admitted: %s", name)
			}
		})
	}
}

func TestChecksArgumentBoundary(t *testing.T) {
	risky := "fn str risky\n    emits [codec::invalid_data]\n    given\n        int flag\n    asserts\n        sample: 0 => codec::invalid_data(\"field\", \"detail\")\n    codec::invalid_data(\"field\", \"detail\")\n"
	main := "fn void main\n    emits []\n    given\n        str[] arguments\n    asserts\n        empty: [] => ok\n    ok\n"
	nested := "package app\n    provides []\n    uses [checks, codec]\n" + risky + "fn void caller\n    emits [checks::failed, codec::invalid_data]\n    asserts\n        sample: => codec::invalid_data(\"field\", \"detail\")\n    match call checks::require(true, call risky(0))\n        codec::invalid_data\n        checks::failed\n        ok => ok\n" + main
	if _, err := programFixture(t, map[string]string{"src/main.can": nested}); err == nil {
		t.Fatal("nested fallible reason argument admitted")
	}
	sequenced := "package app\n    provides []\n    uses [checks, codec]\n" + risky + "fn void caller\n    emits [checks::failed, codec::invalid_data]\n    asserts\n        sample: => codec::invalid_data(\"field\", \"detail\")\n    match call risky(0)\n        codec::invalid_data\n        ok str reason => relay call checks::require(true, reason)\n" + main
	if _, err := programFixture(t, map[string]string{"src/main.can": sequenced}); err != nil {
		t.Fatalf("caller-handled reason rejected: %v", err)
	}
}

func TestChecksRejectsNumberedRedeclaration(t *testing.T) {
	source, err := os.ReadFile("../../testdata/current/checks/main.can")
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Replace(string(source), "    uses [checks]\n", "    uses [checks]\nerror 1010 failed(int code)\n", 1)
	if _, err := programFixtureRegistry(t, map[string]string{"src/main.can": text}, `{"active":["app::failed"],"retired":[]}`); err == nil {
		t.Fatal("numbered error declaration admitted")
	}
}
