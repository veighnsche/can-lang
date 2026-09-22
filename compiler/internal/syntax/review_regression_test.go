package syntax

import (
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/source"
)

func TestCompletionPatternsAcceptExactHeads(t *testing.T) {
	for _, body := range []string{
		"    match call fetch()\n        ok int value => ok value\n        failure<int> => ok 0\n",
		"    match call fetch()\n        ok int value => ok value\n        remote::failure<int> as exact => ok exact\n",
		"    match call fetch()\n        ok int value => ok value\n        failure<int>\n",
		"    int result = match call race\n        fetch()\n        ok int value => ok value\n        all_failed<int> as first => ok 0\n    ok result\n",
		"    match call concurrent with error\n        fetch()\n            ok int value => ok\n            failure<int> as exact => ok\n    ok 0\n",
	} {
		text := testHeader + "fn int example\n    emits []\n    asserts\n        sample: => ok 0\n" + body
		// Exact heads, explicit aliases, and unaliased exact forwarding are
		// valid syntax; resolution against the matched bound happens in check.
		parseFile(t, text)
	}
}

func TestCompletionPatternsRejectAliasWithoutArrow(t *testing.T) {
	for _, body := range []string{
		"    match call fetch()\n        ok int value => ok value\n        failure<int> as exact\n",
		"    match call fetch()\n        ok int value => ok value\n        failure as exact\n",
	} {
		text := testHeader + "fn int example\n    emits []\n    asserts\n        sample: => ok 0\n" + body
		file, err := source.New("alias-pattern.can", text)
		if err != nil {
			t.Fatal(err)
		}
		result := Parse(file)
		if result.OK() {
			t.Fatalf("aliased head without arrow parsed: %q", body)
		}
	}
}

func TestNumericFieldDiagnosticFixturesRoundTrip(t *testing.T) {
	for _, value := range []string{"1", "1.5", "1e3", "-1", "1 /* separator */ .other"} {
		parseFile(t, testHeader+"fn int example\n    emits []\n    asserts\n        sample: => ok 0\n    ok "+value+" /* token separator */ .value\n")
	}
}
