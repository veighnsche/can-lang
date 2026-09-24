package check

import (
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/source"
)

// T02 (DI-01): bare ordinary-data pattern names must resolve to checked
// nominal leaves; explicit `bind name` captures at every depth. `_`,
// `...rest` and error `as` keep their scoped roles.
const patternBindHeader = programHeader + `record paid
record pending
record declined
variant payment
    paid
    pending
    declined
record receipt
    payment status
record circle
    int radius
record rectangle
    int width
    int height
variant shape
    circle
    rectangle
record hot
    int degrees
record cold
    int degrees
variant temperature
    hot
    cold
record left
    int value
record right
    str value
variant either
    left
    right
`

func patternBindProgram(t *testing.T, body string) (*Program, error) {
	t.Helper()
	return programFixture(t, map[string]string{"src/main.can": body})
}

func TestPatternBindCapturesAtEveryDepth(t *testing.T) {
	text := patternBindHeader + `fn bool accepted
    emits []
    given
        payment value
    asserts
        sample: paid() => ok true
        rejected: declined() => ok false
    match value
        paid => ok true
        bind rest => ok false
fn bool nested
    emits []
    given
        receipt value
    asserts
        sample: receipt(paid()) => ok true
        other: receipt(pending()) => ok false
    match value
        receipt(paid) => ok true
        receipt(bind status) => ok false
fn int area_units
    emits []
    given
        shape value
    asserts
        round: circle(3) => ok 3
        square: rectangle(3, 4) => ok 3
    match value
        circle(bind radius) => ok radius
        rectangle(bind width, _) => ok width
fn int first_or_zero
    emits []
    given
        int[] items
    asserts
        sample: [4, 2] => ok 4
        empty: [] => ok 0
    match items
        [] => ok 0
        [bind head, ...tail] => ok head + tail.length - tail.length
fn int classify
    emits []
    given
        int number
    asserts
        zero: 0 => ok 0
        other: 7 => ok 7
    match number
        0 => ok 0
        bind n => ok n
fn int either_first
    emits []
    given
        int[] items
    asserts
        sample: [0, 9] => ok 1
        empty: [] => ok 0
    match items
        [] => ok 0
        [0, ...tail] | [1, ...tail] => ok tail.length
        [bind head, ...tail] => ok head
fn int degrees_or_zero
    emits []
    given
        temperature value
    asserts
        sample: hot(21) => ok 21
        frost: cold(0) => ok 0
    match value
        hot(bind degrees) | cold(bind degrees) => ok degrees
` + programMain + "    ok\n"
	if _, err := patternBindProgram(t, text); err != nil {
		t.Fatalf("explicit bind rejected: %v", err)
	}
}

func TestPatternBareNominalHidesFields(t *testing.T) {
	text := patternBindHeader + `fn int area_units
    emits []
    given
        shape value
    asserts
        round: circle(3) => ok 9
        square: rectangle(3, 4) => ok 12
    match value
        circle => ok value.radius * value.radius
        rectangle => ok value.width * value.height
fn int radius_of
    emits []
    given
        circle value
    asserts
        sample: circle(3) => ok 3
    match value
        circle => ok value.radius
` + programMain + "    ok\n"
	if _, err := patternBindProgram(t, text); err != nil {
		t.Fatalf("bare nominal leaf rejected: %v", err)
	}
}

func rejectPattern(t *testing.T, text, want string) error {
	t.Helper()
	_, err := patternBindProgram(t, text)
	if err == nil {
		t.Fatalf("bare pattern admitted, want %q", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("wrong pattern diagnostic: %v", err)
	}
	return err
}

func locatePattern(t *testing.T, text, spelling string) {
	t.Helper()
	err := rejectPattern(t, text, spelling)
	located, ok := source.AsLocated(err)
	if !ok || located.Code != "CAN-CHECK-UNKNOWN-PATTERN" {
		t.Fatalf("unknown-pattern diagnostic lost its span/code: %v", err)
	}
	if located.Span.End <= located.Span.Start || text[located.Span.Start:located.Span.End] != spelling {
		t.Fatalf("unknown-pattern span covers %q, want %q: %v", text[located.Span.Start:located.Span.End], spelling, err)
	}
	if !strings.Contains(err.Error(), "bind "+spelling) {
		t.Fatalf("unknown-pattern diagnostic hides the bind repair: %v", err)
	}
}

// The final-arm typo from the clean-room probe must diagnose instead of
// silently capturing the remainder as an `any` binding.
func TestPatternFinalArmTypoRejected(t *testing.T) {
	text := patternBindHeader + `fn bool accepted
    emits []
    given
        payment value
    asserts
        sample: paid() => ok true
        rejected: declined() => ok false
    match value
        paid => ok true
        pending => ok false
        decliend => ok false
` + programMain + "    ok\n"
	locatePattern(t, text, "decliend")
}

// A misspelled leaf nested inside a constructor field must diagnose at the
// nested position, not capture the field.
func TestPatternNestedTypoRejected(t *testing.T) {
	text := patternBindHeader + `fn bool nested
    emits []
    given
        receipt value
    asserts
        sample: receipt(paid()) => ok true
        other: receipt(pending()) => ok false
    match value
        receipt(paidd) => ok true
        _ => ok false
` + programMain + "    ok\n"
	locatePattern(t, text, "paidd")
}

// Renaming a leaf type must diagnose the stale arm; the corrected spelling
// keeps checking.
func TestPatternRefactoredLeafDiagnosesStaleArm(t *testing.T) {
	refactored := programHeader + `record paid
record pending
record refunded
variant payment
    paid
    pending
    refunded
fn bool accepted
    emits []
    given
        payment value
    asserts
        sample: paid() => ok true
        refund: refunded() => ok false
    match value
        paid => ok true
        pending => ok false
        declined => ok false
` + programMain + "    ok\n"
	locatePattern(t, refactored, "declined")
	fixed := strings.Replace(refactored, "        declined => ok false\n", "        refunded => ok false\n", 1)
	if _, err := patternBindProgram(t, fixed); err != nil {
		t.Fatalf("renamed leaf rejected: %v", err)
	}
}

func TestPatternBareNameOutsideVariantRejected(t *testing.T) {
	scalar := patternBindHeader + `fn int classify
    emits []
    given
        int number
    asserts
        zero: 0 => ok 0
        other: 7 => ok 7
    match number
        0 => ok 0
        other => ok 0
` + programMain + "    ok\n"
	locatePattern(t, scalar, "other")
	record := patternBindHeader + `fn int radius_of
    emits []
    given
        circle value
    asserts
        sample: circle(3) => ok 3
    match value
        round => ok 0
` + programMain + "    ok\n"
	locatePattern(t, record, "round")
}

func TestPatternBindDuplicatesAndAlternatives(t *testing.T) {
	duplicate := patternBindHeader + `fn int first
    emits []
    given
        int[] items
    asserts
        sample: [4, 2] => ok 4
    match items
        [bind head, bind head] => ok head
        _ => ok 0
` + programMain + "    ok\n"
	rejectPattern(t, duplicate, "duplicate pattern binding head")
	alternativeMerge := patternBindHeader + `fn int first
    emits []
    given
        int[] items
    asserts
        sample: [4, 2] => ok 4
    match items
        [bind head, bind head | bind head] => ok head
        _ => ok 0
` + programMain + "    ok\n"
	rejectPattern(t, alternativeMerge, "duplicate alternative binding")
	missingName := patternBindHeader + `fn int first
    emits []
    given
        int[] items
    asserts
        sample: [4, 2] => ok 4
    match items
        [bind head] | [] => ok head
        _ => ok 0
` + programMain + "    ok\n"
	rejectPattern(t, missingName, "alternatives must bind the same names")
	typeMismatch := patternBindHeader + `fn int either_value
    emits []
    given
        either choice
    asserts
        sample: left(4) => ok 1
    match choice
        left(bind value) | right(bind value) => ok 1
` + programMain + "    ok\n"
	rejectPattern(t, typeMismatch, "alternative binding types differ")
}
