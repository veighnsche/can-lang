package check

import (
	"strings"
	"testing"
)

func TestMandatoryAssertionsAreTyped(t *testing.T) {
	valid := programHeader + programMain + "    ok\nfn int plus\n    emits []\n    given\n        int value\n    asserts\n        sample: 2 => ok 3\n    ok value + 1\n"
	if _, err := programFixture(t, map[string]string{"src/main.can": valid}); err != nil {
		t.Fatal(err)
	}
	for name, replacement := range map[string]string{
		"wrong input":            "sample: true => ok 3",
		"missing input":          "sample: => ok 3",
		"wrong expected":         "sample: 2 => ok false",
		"missing expected value": "sample: 2 => ok",
		"duplicate name":         "sample: 2 => ok 3\n        sample: 1 => ok 2",
	} {
		t.Run(name, func(t *testing.T) {
			source := strings.Replace(valid, "sample: 2 => ok 3", replacement, 1)
			if _, err := programFixture(t, map[string]string{"src/main.can": source}); err == nil {
				t.Fatal("invalid assertion accepted")
			}
		})
	}
}
func TestSuppliedCompletionsAreCheckedBeforeExecution(t *testing.T) {
	source := programHeader + programMain + `    match call plus(1)
        when
            empty: 1 => ok 7
        ok int result => match result
            7 => ok
            _ => ok
fn int plus
    emits []
    given
        int value
    asserts
        sample: 1 => ok 2
    ok value + 1
`
	if _, err := programFixture(t, map[string]string{"src/main.can": source}); err != nil {
		t.Fatal(err)
	}
	for _, row := range []string{"empty: true => ok 7", "empty: 1 => ok false", "empty: 1 => ok", "empty: => ok 7"} {
		if _, err := programFixture(t, map[string]string{"src/main.can": strings.Replace(source, "empty: 1 => ok 7", row, 1)}); err == nil {
			t.Fatalf("invalid fixture accepted: %s", row)
		}
	}
}

func TestNativeSliceFixturesUseCheckedInvocationContract(t *testing.T) {
	source := programHeader + `fn str sample
    emits []
    asserts
        selected: => ok "fake"
    match call "abc".slice(1, 3)
        when
            selected: 1, 3 => ok "fake"
        ok
` + programMain + "    ok\n"
	p, err := programFixture(t, map[string]string{"src/main.can": source})
	if err != nil {
		t.Fatal(err)
	}
	for _, fn := range p.Functions {
		if fn.Symbol.Name == "sample" {
			step := fn.Region.Body.Terminal.Match.Call.Steps[0]
			if step.Native == nil || step.Contract == nil || step.Fixtures == nil || !step.Receiver || len(step.Prepare) != 3 || len(step.Arguments) != 3 {
				t.Fatal("native fixture lost its checked native operation or argument contract")
			}
		}
	}
	for _, row := range []string{`selected: "wrong", 3 => ok "fake"`, `selected: 1 => ok "fake"`, `selected: 1, 3 => ok 7`, `selected: 1, 3 => ok`} {
		if _, err := programFixture(t, map[string]string{"src/main.can": strings.Replace(source, `selected: 1, 3 => ok "fake"`, row, 1)}); err == nil {
			t.Fatalf("malformed native fixture admitted: %s", row)
		}
	}
}

func TestUsingRawOnOrdinaryFunctionsRejected(t *testing.T) {
	attached := programHeader + "fn int plus\n    emits []\n    given\n        int value\n    asserts\n        sample: 2 => ok 3\n            using raw \"fixtures/plus.json\"\n    ok value + 1\n" + programMain + "    ok\n"
	if _, err := programFixture(t, withNativeRaw(map[string]string{"src/main.can": attached}, "plus")); err == nil || !strings.Contains(err.Error(), "using raw is allowed on fetch/judge/LLM targets only") {
		t.Fatalf("raw mode on ordinary function admitted: %v", err)
	}
	lexical := programHeader + "fn int sample\n    emits []\n    asserts\n        selected: => ok 7\n    match call plus(1)\n        when\n            selected: 1 => ok 7\n                using raw \"fixtures/plus.json\"\n        ok\nfn int plus\n    emits []\n    given\n        int value\n    asserts\n        sample: 1 => ok 2\n    ok value + 1\n" + programMain + "    ok\n"
	if _, err := programFixture(t, withNativeRaw(map[string]string{"src/main.can": lexical}, "plus")); err == nil || !strings.Contains(err.Error(), "using raw is allowed on fetch/judge/LLM targets only") {
		t.Fatalf("raw mode on ordinary lexical target admitted: %v", err)
	}
}
