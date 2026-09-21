package check

import (
	"strings"
	"testing"
)

func TestNativeDescriptorMetadataPhase(t *testing.T) {
	helpers := `fn str describe
    emits []
    given
        float value
    asserts
        sample: 0.5 => ok "Description"
    ok "Description"

fn float threshold
    emits []
    given
        float value
    asserts
        sample: 0.5 => ok 0.5
    ok value

fn str captured
    emits []
    given
        near float certainty
    asserts
        sample: 0.5 => ok "Description"
    ok "Description"

fn str invoke
    emits []
    given
        callable str () emits [] action
    asserts
        sample: callable fixed => ok "Description"
    relay call action()

fn str fixed
    emits []
    asserts
        sample: => ok "Description"
    ok "Description"
`
	question := `choice float choose from classifier
    emits [ai::invalid_question, ai::invalid_answer]
    confidence as certainty
    asks "Choose"
    minimum 0.5 => ok -1.0
        yes "Yes" => ok certainty
        no "No" => ok %
`
	for name, replacement := range map[string][2]string{
		"nested asks":      {`asks "Choose"`, `asks call describe(certainty)`},
		"nested criterion": {`yes "Yes"`, `yes call describe(certainty)`},
		"nested minimum":   {`minimum 0.5`, `minimum call threshold(certainty)`},
		"near capture":     {`asks "Choose"`, `asks call invoke(callable captured)`},
	} {
		t.Run(name, func(t *testing.T) {
			text := nativeHeader + nativeClassifier + helpers + strings.Replace(question, replacement[0], replacement[1], 1) + programMain + "    ok\n"
			if _, err := programFixture(t, map[string]string{"src/main.can": text}); err == nil || !strings.Contains(err.Error(), "answer metadata is unavailable") {
				t.Fatalf("expected preparation phase rejection, got %v", err)
			}
		})
	}
	if _, err := programFixture(t, map[string]string{"src/main.can": nativeHeader + nativeClassifier + helpers + question + programMain + "    ok\n"}); err != nil {
		t.Fatal(err)
	}
}

func TestNativeProbabilityLocal(t *testing.T) {
	question := `noul float likelihood from classifier
    emits [ai::invalid_question, ai::invalid_answer]
    asks "Likely?"
        true "Yes" => do
            float probability = %
            ok probability
        false "No" => do
            float probability = %
            ok probability
`
	if _, err := programFixture(t, map[string]string{"src/main.can": nativeHeader + nativeClassifier + question + programMain + "    ok\n"}); err != nil {
		t.Fatal(err)
	}
}

func TestNativeVariadicGeneratedShape(t *testing.T) {
	declarations := `record arms
    choice_arm<float> emits [] left
    choice_arm<float> emits [] right

record weights choice float weighted from classifier
    emits [ai::invalid_question, ai::invalid_answer]
    given
        arms ...choices
    asks "Choose"
        ...choices[0]
`
	p, err := programFixture(t, map[string]string{"src/main.can": nativeHeader + nativeClassifier + declarations + programMain + "    ok\n"})
	if err != nil {
		t.Fatal(err)
	}
	fields := p.Natives[0].Signature.Result().Fields()
	if len(fields) != 2 || fields[0].Name != "left" || fields[1].Name != "right" {
		t.Fatalf("wrong packed variadic spread shape: %+v", fields)
	}
}

func TestNativeStateCodecAdmission(t *testing.T) {
	for _, kind := range []string{"judge", "llm"} {
		for _, typ := range []string{"bytes::buffer", "callable str () emits []", "choice_arm<str> emits []", "opaque_record", "opaque_record[]"} {
			t.Run(kind+"/"+typ, func(t *testing.T) {
				declaration := "llm str generate from generator\n    emits [" + nativeLLM + "]\n    state\n        " + typ + " input\n    asks \"Generate\"\n"
				if kind == "judge" {
					declaration = strings.Replace(nativeJudge, "str message", typ+" message", 1)
				}
				header := strings.Replace(nativeHeader, "uses [", "uses [bytes, ", 1)
				text := header + "record opaque_record\n    bytes::buffer content\n" + nativeGenerator + nativeClassifier + nativeQuestion + declaration + programMain + "    ok\n"
				if _, err := programFixture(t, map[string]string{"src/main.can": text}); err == nil || !strings.Contains(err.Error(), "not codec-admissible") {
					t.Fatalf("expected codec admission error, got %v", err)
				}
			})
		}
	}
}

func TestNativeExportedConnection(t *testing.T) {
	for _, declaration := range []string{
		"fetch str exposed from generator\n    emits [" + nativeHTTP + "]\n    get \"/\"\n",
		"llm str exposed from generator\n    emits [" + nativeLLM + "]\n    asks \"Generate\"\n",
		strings.Replace(nativeQuestion, "bool question", "bool exposed", 1),
		strings.Replace(nativeJudge, "bool assess", "bool exposed", 1) + nativeQuestion,
	} {
		header := strings.Replace(nativeHeader, "provides []", "provides [exposed]", 1)
		text := header + nativeGenerator + nativeClassifier + declaration + programMain + "    ok\n"
		if _, err := programFixture(t, map[string]string{"src/main.can": text}); err == nil || !strings.Contains(err.Error(), "private connection") {
			t.Fatalf("expected private connection error, got %v", err)
		}
		text = strings.Replace(text, "provides [exposed]", "provides [exposed, generator, classifier]", 1)
		if _, err := programFixture(t, map[string]string{"src/main.can": text}); err != nil {
			t.Fatal(err)
		}
	}
}
