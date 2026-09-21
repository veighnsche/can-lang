package syntax

import (
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/source"
)

func TestCompletionPatternsRejectTypeApplications(t *testing.T) {
	for _, body := range []string{
		"    match call fetch()\n        ok int value => ok value\n        failure<int> => ok 0\n",
		"    match call fetch()\n        ok int value => ok value\n        remote::failure<int> => ok 0\n",
		"    int result = match call race\n        fetch()\n        ok int value => ok value\n        all_failed<int> => ok 0\n    ok result\n",
		"    match call concurrent with error\n        fetch()\n            ok int value => ok\n            failure<int> => ok\n    ok 0\n",
	} {
		text := testHeader + "fn int example\n    emits []\n    asserts\n        sample: => ok 0\n" + body
		file, err := source.New("generic-pattern.can", text)
		if err != nil {
			t.Fatal(err)
		}
		result := Parse(file)
		if result.OK() || !strings.Contains(result.Diagnostics[0].Message, "do not accept generic arguments") {
			t.Fatalf("expected generic-pattern diagnostic: %v", result.Diagnostics)
		}
		// The bare spelling remains valid syntax. Concrete error identity is
		// supplied by the checked completion contract, not pattern arguments.
		parseFile(t, strings.ReplaceAll(text, "<int>", ""))
	}
}

func TestNumericFieldDiagnosticFixturesRoundTrip(t *testing.T) {
	for _, value := range []string{"1", "1.5", "1e3", "-1", "1 /* separator */ .other"} {
		parseFile(t, testHeader+"fn int example\n    emits []\n    asserts\n        sample: => ok 0\n    ok "+value+" /* token separator */ .value\n")
	}
}
