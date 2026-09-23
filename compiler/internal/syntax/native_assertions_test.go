package syntax

import (
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/source"
)

func TestNativeAttachedAssertions(t *testing.T) {
	text := `connection service
    timeout_ms 1000
    endpoint "https://example.invalid/"

fetch str load from service
    emits []
    asserts
        decoded: => ok "hi"
            using raw "fixtures/load.json"
    get "/receipt"

llm str summarize from service
    emits []
    state
        str email
    asserts
        drafted: "a@b.c" => ok "done"
            using raw "fixtures/summarize.json"
    asks email
`
	result := nativeParse(t, text)
	if !result.OK() {
		t.Fatal(result.Diagnostics)
	}
	fetch := result.File.Declarations[1].(*FetchDecl)
	if len(fetch.Assertions) != 1 || fetch.Assertions[0].Mode == nil || fetch.Assertions[0].Mode.Raw.Value != "fixtures/load.json" {
		t.Fatal("fetch attached raw assertion missing")
	}
	llm := result.File.Declarations[2].(*LLMDecl)
	if len(llm.Assertions) != 1 || llm.Assertions[0].Mode == nil || llm.Assertions[0].Mode.Raw.Value != "fixtures/summarize.json" {
		t.Fatal("llm attached raw assertion missing")
	}
	formatted := Format(result.File)
	for _, want := range []string{"asserts", "decoded:  => ok \"hi\"", "using raw \"fixtures/load.json\""} {
		found := false
		for _, line := range []string{formatted} {
			if len(line) > 0 && contains(line, want) {
				found = true
			}
		}
		if !found {
			t.Fatalf("formatted output omits %q", want)
		}
	}
}

func TestAssertionModeLinesRejectMalformed(t *testing.T) {
	for _, bad := range []string{
		"decoded: => ok \"hi\"\n            using failure native ok\n",
		"decoded: => ok \"hi\"\n            using raw \"a.json\"\n            using raw \"b.json\"\n",
		"decoded: => ok \"hi\"\n            raw \"a.json\"\n",
		"decoded: => ok \"hi\"\n            using raw 7\n",
	} {
		text := "connection service\n    timeout_ms 1000\n    endpoint \"https://example.invalid/\"\n\nfetch str load from service\n    emits []\n    asserts\n        " + bad + "    get \"/x\"\n"
		if result := nativeParse(t, text); result.OK() {
			t.Fatalf("malformed mode admitted: %q", bad)
		}
	}
}

func TestWhenRowAcceptsRawMode(t *testing.T) {
	text := nativePackage + `fn str main
    emits []
    given
        str[] args
    asserts
        sample: [] => ok "hi"
    match call load()
        when
            sample: => ok "hi"
                using raw "fixtures/load.json"
        ok str value => ok value
`
	file, err := source.New("when.can", text)
	if err != nil {
		t.Fatal(err)
	}
	result := Parse(file)
	if !result.OK() {
		t.Fatal(result.Diagnostics)
	}
	fn := result.File.Declarations[0].(*FunctionDecl)
	match := fn.Body.Terminal.(*MatchBody)
	if len(match.Match.When) != 1 || match.Match.When[0].Mode == nil {
		t.Fatal("when raw mode missing")
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
