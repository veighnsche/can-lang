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
