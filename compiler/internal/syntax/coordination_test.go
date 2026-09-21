package syntax

import (
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/source"
)

func coordinationFunction(body string) string {
	return testHeader + "fn void main\n    emits []\n    asserts\n        smoke: => ok\n" + body + "    ok\n"
}

func TestCoordinationGrammarAndRendering(t *testing.T) {
	for _, body := range []string{
		`    match call concurrent
        primary::lookup(key)
            ok profile value => ok
        ...operations
            ok profile value => ok
        unavailable => ok
        [_] => ok
`,
		`    bool[] results = match call concurrent with error
        database::save(account)
            ok receipt saved => ok true
            unavailable => ok false
        ...operations
            ok receipt saved => ok true
            unavailable => ok false
            [_] => ok false
`,
		`    profile result = match call race
        primary::lookup(key)
        backup::lookup(key)
        ok profile found => ok found
        all_failed => ok fallback
`,
		`    profile result = match call race with error
        ...operations
        ok profile found => ok found
        unavailable => ok fallback
        [_] => ok fallback
`,
	} {
		file := parseFile(t, coordinationFunction(body))
		if len(file.Declarations[0].(*FunctionDecl).Body.Steps) != 1 {
			t.Fatal("coordination did not remain a step")
		}
	}
}

func TestCoordinationRejectsWrongArmPlacement(t *testing.T) {
	for _, body := range []string{
		"    match call race\n        ok => ok\n",
		"    match call race\n        fetch()\n            ok => ok\n",
		"    match call concurrent\n        fetch()\n            failed => ok\n",
		"    match call concurrent\n        fetch()\n        ok => ok\n",
		"    match call concurrent with error\n        fetch()\n        failed => ok\n",
		"    match call race\n        call fetch()\n        ok => ok\n",
		"    match call race\n        fetch()\n        when\n            smoke: => ok\n        ok => ok\n",
	} {
		file, err := source.New("bad.can", coordinationFunction(body))
		if err != nil {
			t.Fatal(err)
		}
		if result := Parse(file); result.OK() {
			t.Fatalf("accepted %s", body)
		}
	}
	// Contextual coordination words remain ordinary function names when called.
	parseFile(t, coordinationFunction("    call concurrent()\n"))
}
