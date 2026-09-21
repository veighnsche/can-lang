package check

import (
	"strings"
	"testing"
)

const nativeHTTP = "http::invalid_request, http::transport_failed, http::timeout, http::body_limit, http::status_error, codec::invalid_data"
const nativeLLM = nativeHTTP + ", llm::refused, llm::truncated, llm::invalid_response"
const nativeHeader = "package app\n    provides []\n    uses [http, codec, ai, llm]\n"
const nativeGenerator = "connection generator\n    endpoint \"http://localhost:1\"\n    timeout_ms 1000\n    metadata\n        protocol \"openai_responses_v1\"\n        model \"test\"\n"
const nativeClassifier = "connection classifier\n    endpoint \"http://localhost:1\"\n    timeout_ms 1000\n    metadata\n        protocol \"typesafe_systemone_v1\"\n        model \"jev-latest\"\n"

func TestConnectionPolicyThroughProgram(t *testing.T) {
	connection := strings.Replace(nativeGenerator, "timeout_ms 1000", "auth bearer env \"THIS_TEST_DOES_NOT_READ_A_CREDENTIAL\"\n    timeout_ms 1000 * 30\n    max_body_bytes 1024", 1)
	p, err := programFixture(t, map[string]string{"src/main.can": programHeader + connection + programMain + "    ok\n"})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Connections) != 1 {
		t.Fatal("missing connection policy")
	}
	for _, policy := range p.Connections {
		if policy.TimeoutMilliseconds != 30000 || policy.MaxBodyBytes != 1024 {
			t.Fatal(policy)
		}
	}
	for _, bad := range []string{
		strings.Replace(connection, "timeout_ms 1000 * 30", "timeout_ms call evaluate()", 1),
		strings.Replace(connection, "protocol \"openai_responses_v1\"", "unrecognized \"anything\"", 1),
		strings.Replace(connection, "max_body_bytes 1024", "max_body_bytes 0", 1),
		strings.Replace(connection, "timeout_ms 1000 * 30", "timeout_ms 1 / 0", 1),
	} {
		if _, err := programFixture(t, map[string]string{"src/main.can": programHeader + bad + programMain + "    ok\n"}); err == nil {
			t.Fatal("accepted invalid connection")
		}
	}
}
func TestGroupedNativeInvocation(t *testing.T) {
	for _, f := range []struct{ name, inputs, state, args string }{
		{"empty", "", "", "()"},
		{"one", "", "    state\n        str input\n", "(\"input\")"},
		{"multiple", "    given\n        int setting\n", "    state\n        str input\n        int count\n", "1, (\"input\", 2)"},
		{"variadic", "    given\n        str ...labels\n", "    state\n        str input\n", "\"a\", \"b\", (\"input\")"},
	} {
		t.Run(f.name, func(t *testing.T) {
			native := "llm str generate from generator\n    emits [" + nativeLLM + "]\n" + f.inputs + f.state + "    asks \"Generate\"\n"
			wrapper := "fn str wrap\n    emits [" + nativeLLM + "]\n    asserts\n        sample: => ok \"x\"\n    relay call generate(" + f.args + ")\n"
			p, err := programFixture(t, map[string]string{"src/main.can": nativeHeader + nativeGenerator + native + wrapper + programMain + "    ok\n"})
			if err != nil {
				t.Fatal(err)
			}
			if len(p.Natives) != 1 || !p.Natives[0].Descriptor.Grouped {
				t.Fatal("missing grouping evidence")
			}
			bad := strings.Replace(wrapper, "generate("+f.args+")", "generate(\"not grouped\")", 1)
			if _, err := programFixture(t, map[string]string{"src/main.can": nativeHeader + nativeGenerator + native + bad + programMain + "    ok\n"}); err == nil {
				t.Fatal("accepted missing state group")
			}
		})
	}
}
func TestNativeIntrinsicBoundsAndProfile(t *testing.T) {
	native := "llm str generate from generator\n    emits [" + nativeLLM + "]\n    asks \"Generate\"\n"
	for name, bad := range map[string]string{
		"missing error":               strings.Replace(nativeGenerator+native, "http::timeout, ", "", 1),
		"wrong profile":               strings.Replace(nativeGenerator+native, "openai_responses_v1", "typesafe_systemone_v1", 1),
		"missing authenticated error": strings.Replace(nativeGenerator+native, "    timeout_ms", "    auth bearer env \"TOKEN\"\n    timeout_ms", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := programFixture(t, map[string]string{"src/main.can": nativeHeader + bad + programMain + "    ok\n"}); err == nil {
				t.Fatal("accepted invalid native contract")
			}
		})
	}
}

const nativeQuestion = `noul bool question from classifier
    emits [ai::invalid_question, ai::invalid_answer]
    given
        str description
    asks description
        true "Yes" => ok % >= 0.5
        false "No" => ok false
`
const nativeJudge = `judge bool assess from classifier
    emits [` + nativeHTTP + `, ai::invalid_question, ai::invalid_answer]
    state
        str message
    call question("First") as bool first
    call question("Second") as bool second
    ok => ok first and second
`

func TestJudgePreparationAndHandlerRegions(t *testing.T) {
	text := nativeHeader + nativeClassifier + nativeQuestion + nativeJudge + programMain + "    ok\n"
	p, err := programFixture(t, map[string]string{"src/main.can": text})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Natives) != 2 || len(p.Natives[0].Regions) != 2 || len(p.Natives[1].Registrations) != 2 || len(p.Natives[1].Regions) != 1 {
		t.Fatal("missing native regions")
	}
	for name, bad := range map[string]string{
		"cross registration":     strings.Replace(text, "question(\"Second\")", "question(first)", 1),
		"invalid handler result": strings.Replace(text, "ok false", "ok 1", 1),
		"missing binding":        strings.Replace(text, " as bool first", "", 1),
		"wrong binding":          strings.Replace(text, "as bool first", "as str first", 1),
		"wrong connection":       strings.Replace(text, "judge bool assess from classifier", "judge bool assess from other", 1) + strings.Replace(nativeClassifier, "connection classifier", "connection other", 1),
		"missing question bound": strings.Replace(text, "emits [ai::invalid_question, ai::invalid_answer]", "emits [ai::invalid_question, ai::invalid_answer, http::credentials_missing]", 1),
		"direct question call":   strings.Replace(text, "    ok\n", "    call question(\"direct\")\n    ok\n", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := programFixture(t, map[string]string{"src/main.can": bad}); err == nil {
				t.Fatal("accepted invalid native semantics")
			}
		})
	}
}
func TestChoiceAndScoreStaticBodies(t *testing.T) {
	declarations := `choice str dynamic from classifier
    emits [ai::invalid_question, ai::invalid_answer]
    given
        choice_option[] candidates
    asks "Choose"
        ...candidates
        ok str selected => ok selected

score float score_question from classifier
    emits [ai::invalid_question, ai::invalid_answer]
    score as measured
    confidence as certainty
    asks "Severity"
        low "Low"
        high "High"
        ok => ok measured * certainty

record weights choice float weighted from classifier
    emits [ai::invalid_question, ai::invalid_answer]
    confidence as certainty
    asks "Choose"
        first "First" => ok % * certainty
        second "Second" => ok %
`
	text := nativeHeader + nativeClassifier + declarations + programMain + "    ok\n"
	p, err := programFixture(t, map[string]string{"src/main.can": text})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Natives) != 3 || len(p.Natives[0].Regions) != 1 || len(p.Natives[1].Regions) != 1 || len(p.Natives[2].Regions) != 2 {
		t.Fatal("native handler regions skipped")
	}
	for name, bad := range map[string]string{
		"wrong dynamic result":        strings.Replace(text, "ok selected", "ok 1", 1),
		"wrong dynamic array":         strings.Replace(text, "choice_option[] candidates", "str[] candidates", 1),
		"wrong selected type":         strings.Replace(text, "ok str selected", "ok int selected", 1),
		"selected input collision":    strings.Replace(text, "ok str selected => ok selected", "ok str candidates => ok candidates", 1),
		"score wrong result":          strings.Replace(text, "ok measured * certainty", "ok \"wrong\"", 1),
		"metadata during preparation": strings.Replace(text, "    asks \"Severity\"", "    minimum measured => ok -1.0\n    asks \"Severity\"", 1),
		"metadata collision":          strings.Replace(text, "confidence as certainty\n    asks \"Choose\"\n        first", "confidence as first\n    asks \"Choose\"\n        first", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := programFixture(t, map[string]string{"src/main.can": bad}); err == nil {
				t.Fatal("accepted invalid question body")
			}
		})
	}
}

func TestGeneratedArmSpreadShape(t *testing.T) {
	declarations := `record arms
    choice_arm<float> emits [] left
    choice_arm<float> emits [] right

record weights choice float weighted from classifier
    emits [ai::invalid_question, ai::invalid_answer]
    given
        arms choices
    confidence as certainty
    asks "Choose"
        before "First" => ok %
        ...choices
        after "Last" => ok %
`
	text := nativeHeader + nativeClassifier + declarations + programMain + "    ok\n"
	p, err := programFixture(t, map[string]string{"src/main.can": text})
	if err != nil {
		t.Fatal(err)
	}
	fields := p.Natives[0].Signature.Result().Fields()
	want := []string{"before", "left", "right", "after", "certainty"}
	if len(fields) != len(want) {
		t.Fatalf("generated fields: %+v", fields)
	}
	for i, field := range fields {
		if field.Name != want[i] {
			t.Fatalf("field %d: %s", i, field.Name)
		}
	}
	for name, bad := range map[string]string{
		"expanded collision": strings.Replace(text, "after \"Last\"", "left \"Last\"", 1),
		"incompatible arm":   strings.Replace(text, "choice_arm<float> emits [] left", "choice_arm<str> emits [] left", 1),
		"ordinary field":     strings.Replace(text, "choice_arm<float> emits [] left", "str left", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := programFixture(t, map[string]string{"src/main.can": bad}); err == nil {
				t.Fatal("accepted invalid generated arm spread")
			}
		})
	}
}

func TestNamedArmDeclarationEvidence(t *testing.T) {
	declarations := `choice_arm float reusable
    emits []
    describes "Reusable"
    ok %

choice_arm<float> emits [] stored = reusable
`
	text := nativeHeader + declarations + programMain + "    ok\n"
	p, err := programFixture(t, map[string]string{"src/main.can": text})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Initializers) != 1 || len(p.Natives) != 1 || len(p.Natives[0].Regions) != 1 {
		t.Fatal("named arm evidence missing")
	}
	for name, bad := range map[string]string{
		"nonconstant description": strings.Replace(text, "describes \"Reusable\"", "describes call describe()", 1),
		"empty description":       strings.Replace(text, "describes \"Reusable\"", "describes \"\"", 1),
		"free capture":            strings.Replace(text, "ok %", "ok captured", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := programFixture(t, map[string]string{"src/main.can": bad}); err == nil {
				t.Fatal("accepted invalid arm declaration")
			}
		})
	}
}
