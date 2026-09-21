package check

import (
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
)

func TestLexicalSitesUseSyntaxPreorder(t *testing.T) {
	text := `package app
    provides []
    uses []
fn int run
    emits []
    given
        int value
    asserts
        sample: 1 => ok 1
    int prepared = call outer(call inner()).step(call later())
    callable int (int) emits [] action = callable run
    match call outer(value)
        when
            sample: call expected() => ok call supplied()
        ok int result => ok result
`
	var prior []string
	for _, input := range []string{text, "\n" + strings.ReplaceAll(text, "=>", "  =>  ")} {
		file, err := source.New("sites.can", input)
		if err != nil {
			t.Fatal(err)
		}
		parsed := syntax.Parse(file)
		if len(parsed.Diagnostics) != 0 {
			t.Fatal(parsed.Diagnostics)
		}
		sites := indexLexicalSites("p::run", parsed.File.Declarations[0])
		ordered := make([]string, len(sites))
		for key, identity := range sites {
			ordinal, err := strconv.Atoi(strings.TrimPrefix(identity, "p::run#"))
			if err != nil || ordinal < 0 || ordinal >= len(ordered) || ordered[ordinal] != "" {
				t.Fatalf("invalid lexical ordinal: %s", identity)
			}
			ordered[ordinal] = key.kind
		}
		want := []string{"call", "call", "method", "call", "callable", "call", "call", "call"}
		if !reflect.DeepEqual(ordered, want) {
			t.Fatalf("wrong preorder: %v", ordered)
		}
		if prior != nil && !reflect.DeepEqual(ordered, prior) {
			t.Fatal("formatting changed lexical identities")
		}
		prior = ordered
	}
}

func TestFixtureSiteIgnoresCheckerLocalAllocation(t *testing.T) {
	text := programHeader + `fn int value
    emits []
    asserts
        sample: => ok 1
    ok 1
fn int consumer
    emits []
    asserts
        sample: => ok 2
    match call value()
        when
            sample: => ok 2
        ok
` + programMain + "    ok\n"
	var identities []string
	for _, input := range []string{text, strings.Replace(text, "    match call value()", "    int unused = 7\n    match call value()", 1)} {
		program, err := programFixture(t, map[string]string{"src/main.can": input})
		if err != nil {
			t.Fatal(err)
		}
		for _, function := range program.Functions {
			if function.Symbol.Name == "consumer" {
				step := function.Region.Body.Terminal.Match.Call.Steps[0]
				identities = append(identities, step.Site, step.Fixtures.Identity)
			}
		}
	}
	if len(identities) != 4 || identities[0] != identities[2] || identities[1] != identities[3] || strings.Contains(identities[0], "/local/") {
		t.Fatalf("checker locals changed fixture site: %v", identities)
	}
	if !strings.HasSuffix(identities[1], "#0/when") {
		t.Fatal(identities)
	}
}
