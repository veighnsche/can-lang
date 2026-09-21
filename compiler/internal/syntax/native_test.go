package syntax

import (
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"strings"
	"testing"
)

const nativePackage = "package app\n    provides []\n    uses []\n\n"

func nativeParse(t *testing.T, text string) ParseResult {
	t.Helper()
	file, err := source.New("native.can", nativePackage+text)
	if err != nil {
		t.Fatal(err)
	}
	return Parse(file)
}
func TestNativeConnectionFetchLLM(t *testing.T) {
	text := `connection service
    timeout_ms 1000
    endpoint "http://localhost:8080/api/"
    auth bearer env "TOKEN"
    headers
        x_test = "yes"
    metadata
        protocol "openai_responses_v1"
        model "example"

fetch str load from service
    emits []
    given
        str path
    get path
    query
        q = ["one", "two"]
    headers
        accept = "text/plain"

llm str summarize from service
    emits []
    given
        str instructions
    state
        str email
    asks instructions
`
	result := nativeParse(t, text)
	if !result.OK() {
		t.Fatal(result.Diagnostics)
	}
	if len(result.File.Declarations) != 3 {
		t.Fatal("missing declarations")
	}
	if len(result.File.Declarations[0].(*ConnectionDecl).Settings) != 5 {
		t.Fatal("missing settings")
	}
	if len(result.File.Declarations[1].(*FetchDecl).Query) != 1 {
		t.Fatal("missing query")
	}
	if len(result.File.Declarations[2].(*LLMDecl).State) != 1 {
		t.Fatal("missing state")
	}
	formatted := Format(result.File)
	file, err := source.New("formatted.can", formatted)
	if err != nil {
		t.Fatal(err)
	}
	round := Parse(file)
	if !round.OK() {
		t.Fatal(round.Diagnostics)
	}
	if Format(round.File) != formatted {
		t.Fatal("unstable native formatting")
	}
}
func TestNativeSectionRejection(t *testing.T) {
	for _, text := range []string{
		"fetch str load from service\n    get \"/\"\n",
		"fetch str load from service\n    emits []\n    get \"/\"\n    body text \"bad\"\n",
		"fetch str load from service\n    emits []\n    get \"/\"\n    headers\n        accept = \"text/plain\"\n    query\n        q = \"late\"\n",
		"llm str make from service\n    emits []\n    given\n        near str secret\n    asks secret\n",
		"llm str make from service\n    emits []\n    state\n        str ...items\n    asks \"x\"\n",
		"llm str make from service\n    emits []\n    asks \"x\"\n    ok \"body\"\n",
		"connection service\n    auth basic env \"TOKEN\"\n",
	} {
		if r := nativeParse(t, text); r.OK() {
			t.Fatalf("accepted invalid native sections: %s", strings.TrimSpace(text))
		}
	}
}

func TestQuestionJudgeArmGrammar(t *testing.T) {
	text := `noul bool decide from service
    emits []
    asks "Proceed?"
    minimum 0.6
        false "No" => ok false
        true "Yes" => ok % > 0.5

choice str route from service
    emits []
    asks "Where?"
    confidence as confidence_value
    minimum 0.6 => ok "fallback"
        sales "Sales" => ok "sales"
        support "Support" => ok "support"

record weights choice float weighted from service
    emits []
    confidence as certainty
    asks "Where?"
        sales "Sales" => ok %
        support "Support" => ok %

score float severity from service
    emits []
    score as value
    confidence as confidence_value
    minimum 0.5 => ok -1.0
    asks "How severe?"
        low "Low"
        high "High"
        ok => ok value

record levels score float level_weights from service
    emits []
    asks "Severity?"
        low "Low" => ok %
        high "High" => ok %

choice str dynamic from service
    emits []
    given
        choice_option[] options
    asks "Choose"
        ...options
        ok str selected => ok selected

judge str classify from service
    emits []
    state
        str text
    call dynamic([]) as str result
    ok => ok result

choice_arm float reusable
    emits []
    describes "Helpful"
    ok %
`
	parsed := nativeParse(t, text)
	if !parsed.OK() {
		t.Fatal(parsed.Diagnostics)
	}
	if len(parsed.File.Declarations) != 8 {
		t.Fatal("missing native declarations")
	}
	formatted := Format(parsed.File)
	file, err := source.New("roundtrip.can", formatted)
	if err != nil {
		t.Fatal(err)
	}
	again := Parse(file)
	if !again.OK() {
		t.Fatalf("%v\n%s", again.Diagnostics, formatted)
	}
	if Format(again.File) != formatted {
		t.Fatal("unstable question formatting")
	}
}

func TestNativeQuestionClosureAndProbability(t *testing.T) {
	for _, text := range []string{
		"noul bool q from c\n    emits []\n    asks \"x\"\n        true \"x\" => ok true\n        true \"y\" => ok false\n",
		"choice str q from c\n    emits []\n    asks \"x\"\n    confidence as c\n        x \"x\" => ok \"x\"\n",
		"record r choice str q from c\n    emits []\n    asks \"x\"\n    minimum 0.5 => ok \"x\"\n        x \"x\" => ok \"x\"\n",
		"score float q from c\n    emits []\n    asks \"x\"\n        low \"low\"\n",
		"score float q from c\n    emits []\n    asks \"x\"\n        low \"low\"\n        ok => ok %\n",
		"choice str q from c\n    emits []\n    asks \"x\"\n        x \"x\" => ok \"x\"\n        ok str selected => ok selected\n",
		"judge str q from c\n    emits []\n    ok => ok \"x\"\n",
		"judge str q from c\n    emits []\n    call question() as str answer\n        ok => ok answer\n    ok => ok answer\n",
		"llm str q from c\n    emits []\n    asks %\n",
	} {
		if result := nativeParse(t, text); result.OK() {
			t.Fatalf("accepted invalid native form: %s", text)
		}
	}
}
