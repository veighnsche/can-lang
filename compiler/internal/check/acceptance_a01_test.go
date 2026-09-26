package check

import (
	"strings"
	"testing"
)

// AU-Q1: true-first ordinary Boolean matches check; coverage and
// exhaustiveness diagnostics for other shapes are unchanged.
func TestAUQ1BooleanArmOrder(t *testing.T) {
	pick := `fn int pick
    emits []
    given
        bool flag
    asserts
        sample: true => ok 1
`
	program, err := programFixture(t, map[string]string{"src/main.can": programHeader + pick +
		"    match flag\n        true => ok 1\n        false => ok 0\n" + programMain + "    ok\n"})
	if err != nil {
		t.Fatalf("true-first terminal match rejected: %v", err)
	}
	if program == nil {
		t.Fatal("missing checked program")
	}
	program, err = programFixture(t, map[string]string{"src/main.can": programHeader + pick +
		"    int selected = match flag\n        true => 1\n        false => 0\n    ok selected\n" + programMain + "    ok\n"})
	if err != nil {
		t.Fatalf("true-first value match rejected: %v", err)
	}
	if program == nil {
		t.Fatal("missing checked program")
	}
}

// AU-Q1 negatives: duplicate coverage and missing arms still fail; only
// the order rule was removed.
func TestAUQ1BooleanCoverageUnchanged(t *testing.T) {
	pick := `fn int pick
    emits []
    given
        bool flag
    asserts
        sample: true => ok 1
`
	_, err := programFixture(t, map[string]string{"src/main.can": programHeader + pick +
		"    match flag\n        true => ok 1\n        true => ok 1\n" + programMain + "    ok\n"})
	if err == nil || !strings.Contains(err.Error(), "fully covered by earlier arms") {
		t.Fatalf("duplicate Boolean arm admitted: %v", err)
	}
}

// AU-Q2-core: the C8 final-local shape checks with one advisory warning;
// the program, its type contract and its diagnostics are otherwise clean.
func TestAUQ2CoreFinalLocalWarning(t *testing.T) {
	text := programHeader + `fn int forwarded
    emits []
    given
        int left
        int right
    asserts
        sample: 1, 2 => ok 3
    int total = left + right
    ok total
` + programMain + "    ok\n"
	program, err := programFixture(t, map[string]string{"src/main.can": text})
	if err != nil {
		t.Fatalf("C8 shape rejected: %v", err)
	}
	if len(program.Warnings) != 1 {
		t.Fatalf("expected one warning, got %+v", program.Warnings)
	}
	warning := program.Warnings[0]
	if warning.Code != "CAN-CHECK-UNNECESSARY-LOCAL" {
		t.Fatalf("wrong warning code: %+v", warning)
	}
	if !strings.Contains(warning.Message, "accidental alias") || !strings.Contains(warning.Message, "total") {
		t.Fatalf("wrong warning message: %+v", warning)
	}
	if !strings.HasSuffix(warning.File, "src/main.can") || warning.Line < 1 || warning.Column < 1 {
		t.Fatalf("warning lost its position: %+v", warning)
	}
	formatted := warning.Format()
	if !strings.Contains(formatted, "CAN-CHECK-UNNECESSARY-LOCAL") || !strings.Contains(formatted, "accidental alias") {
		t.Fatalf("warning format lost content: %q", formatted)
	}
}

// AU-Q2-core: programs without the shape report no warnings.
func TestAUQ2CoreNoWarningWithoutShape(t *testing.T) {
	text := programHeader + `fn int kept
    emits []
    given
        int left
        int right
    asserts
        sample: 1, 2 => ok 6
    int total = left + right
    ok total + total
` + programMain + "    ok\n"
	program, err := programFixture(t, map[string]string{"src/main.can": text})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Warnings) != 0 {
		t.Fatalf("unexpected warnings: %+v", program.Warnings)
	}
}
