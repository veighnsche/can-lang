package syntax

import (
	"strings"
	"testing"
)

// Q1: the formatter canonicalizes ordinary single-scrutinee true-first
// Boolean pairs to false-first; every other match mode keeps its order.
func TestBooleanArmCanonicalization(t *testing.T) {
	header := testHeader
	trueFirst := header + `fn int pick
    emits []
    given
        bool flag
    asserts
        sample: true => ok 1
    match flag
        true => ok 1
        false => ok 0
`
	out := formatTrivia(t, trueFirst)
	falseLine, trueLine := strings.Index(out, "\n        false => ok 0"), strings.Index(out, "\n        true => ok 1")
	if falseLine < 0 || trueLine < 0 || falseLine > trueLine {
		t.Fatalf("true-first pair not canonicalized:\n%s", out)
	}
	assertIdempotent(t, out)
}

func TestBooleanArmCanonicalizationValueMatch(t *testing.T) {
	text := testHeader + `fn int pick
    emits []
    given
        bool flag
    asserts
        sample: true => ok 1
    int selected = match flag
        true => 1
        false => 0
    ok selected
`
	out := formatTrivia(t, text)
	falseLine, trueLine := strings.Index(out, "false => 0"), strings.Index(out, "true => 1")
	if falseLine < 0 || trueLine < 0 || falseLine > trueLine {
		t.Fatalf("value-match pair not canonicalized:\n%s", out)
	}
	assertIdempotent(t, out)
}

func TestBooleanArmCanonicalizationNegatives(t *testing.T) {
	bodies := []string{
		// Already canonical.
		"    match flag\n        false => ok 0\n        true => ok 1\n",
		// Wildcard second arm: reordering would change selection.
		"    match flag\n        true => ok 1\n        _ => ok 0\n",
		// Multi-scrutinee: out of Q1 scope.
		"    match flag, flag\n        true, true => ok 1\n        true, false => ok 2\n        false, _ => ok 0\n",
		// Single complementary alternative: one arm, nothing to swap.
		"    match flag\n        true | false => ok 1\n",
	}
	for _, body := range bodies {
		text := testHeader + "fn int pick\n    emits []\n    given\n        bool flag\n    asserts\n        sample: true => ok 1\n" + body
		out := formatTrivia(t, text)
		if out != text {
			t.Fatalf("non-canonicalizable match moved:\n%s\n---\n%s", text, out)
		}
	}
}

func TestBooleanArmCanonicalizationCompletionUntouched(t *testing.T) {
	text := testHeader + `fn int guarded
    emits [codec::invalid_data]
    asserts
        sample:  => ok 1
    match call number()
        codec::invalid_data => ok 0
        ok int value => ok value
`
	out := formatTrivia(t, text)
	if out != text {
		t.Fatalf("completion match moved:\n%s", out)
	}
}

func TestBooleanArmCanonicalizationWithComments(t *testing.T) {
	text := testHeader + `fn int pick
    emits []
    given
        bool flag
    asserts
        sample: true => ok 1
    match flag
        // leading note
        true => ok 1 // trailing note
        false => ok 0
`
	out := formatTrivia(t, text)
	if !strings.Contains(out, "leading note") || !strings.Contains(out, "trailing note") {
		t.Fatalf("comments lost under canonicalization:\n%s", out)
	}
	falseLine, trueLine := strings.Index(out, "\n        false => ok 0"), strings.Index(out, "\n        true => ok 1")
	if falseLine < 0 || trueLine < 0 || falseLine > trueLine {
		t.Fatalf("commented pair not canonicalized:\n%s", out)
	}
	assertIdempotent(t, out)
}
