package check

import (
	"strings"
	"testing"
)

func TestJudgeRetainsCheckedPhases(t *testing.T) {
	text := nativeHeader + nativeClassifier + nativeQuestion + nativeJudge + programMain + "    ok\n"
	p, err := programFixture(t, map[string]string{"src/main.can": text})
	if err != nil {
		t.Fatal(err)
	}
	question, judge := p.Natives[0].Question, p.Natives[1].Judge
	if question == nil || judge == nil || question.Instructions == nil || question.Minimum.Text != "0.5" || len(question.Options) != 2 {
		t.Fatal("missing checked descriptor evidence")
	}
	if question.Options[0].Name != "true" || question.Options[1].Name != "false" || question.Options[0].Handler != p.Natives[0].Regions[0] || question.Options[1].Handler != p.Natives[0].Regions[1] {
		t.Fatal("lost option labels or handler association")
	}
	if len(judge.Registrations) != 2 || judge.Registrations[0].Question != question.Identity || judge.Registrations[1].Question != question.Identity {
		t.Fatal("repeated registrations were merged")
	}
	if len(judge.Inputs) != 1 || len(judge.Continuation.Inputs) != 3 || judge.Continuation.Inputs[1].Identity != judge.Registrations[0].Binding.Identity {
		t.Fatal("answer scope leaked out of continuation")
	}
	if len(judge.StateInputs) != 1 || judge.StateInputs[0].Identity != judge.Inputs[0].Identity {
		t.Fatal("state inputs lost")
	}
	for _, node := range judge.State.Nodes {
		if node.Identity == judge.State.Root && (len(node.Fields) != 1 || node.Fields[0].Name != "message") {
			t.Fatal("incorrect state object schema")
		}
	}
	// Both source option orders are legal; local selection uses labels, while
	// descriptor computation retains declaration order.
	reversed := strings.Replace(text, "true \"Yes\" => ok % >= 0.5\n        false \"No\" => ok false", "false \"No\" => ok false\n        true \"Yes\" => ok % >= 0.5", 1)
	p, err = programFixture(t, map[string]string{"src/main.can": reversed})
	if err != nil {
		t.Fatal(err)
	}
	if p.Natives[0].Question.Options[0].Name != "false" || p.Natives[0].Question.Options[1].Name != "true" {
		t.Fatal("reordered descriptor computations")
	}
}

func TestJudgeStateSchemaDisclosesOnlyState(t *testing.T) {
	for _, state := range []string{"", "    state\n        str message\n        int count\n"} {
		judge := "judge bool assess from classifier\n    emits [" + nativeHTTP + ", ai::invalid_question, ai::invalid_answer]\n    given\n        str private_description\n" + state + "    call question(private_description) as bool unused\n    ok => ok true\n"
		p, err := programFixture(t, map[string]string{"src/main.can": nativeHeader + nativeClassifier + nativeQuestion + judge + programMain + "    ok\n"})
		if err != nil {
			t.Fatal(err)
		}
		plan := p.Natives[1].Judge
		if len(plan.Registrations) != 1 || plan.Registrations[0].Binding == nil {
			t.Fatal("unused question registration eliminated")
		}
		want := 0
		if state != "" {
			want = 2
		}
		if len(plan.StateInputs) != want || len(plan.Inputs) != want+1 {
			t.Fatal("ordinary input disclosed as shared state")
		}
		found := false
		for _, node := range plan.State.Nodes {
			if node.Identity != plan.State.Root {
				continue
			}
			found = true
			if len(node.Fields) != want || want == 2 && (node.Fields[0].Name != "message" || node.Fields[1].Name != "count") {
				t.Fatal("state field order or empty object lost")
			}
		}
		if !found {
			t.Fatal("missing schema root")
		}
	}
}

func TestNamedArmInitializationDependencies(t *testing.T) {
	text := nativeHeader + `choice_arm float arm
    emits []
    describes description
    ok %

choice_arm<float> emits [] stored = arm
str description = "forward"
` + programMain + "    ok\n"
	p, err := programFixture(t, map[string]string{"src/main.can": text})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"::description", "::arm", "::stored"}
	if len(p.Initializers) != len(want) {
		t.Fatal("arm initializer missing")
	}
	for i, suffix := range want {
		if !strings.HasSuffix(p.Initializers[i].Identity, suffix) {
			t.Fatalf("initializer order: %s", p.Initializers[i].Identity)
		}
	}
	cyclic := strings.Replace(text, `str description = "forward"`, `str description = other
str other = description`, 1)
	if _, err := programFixture(t, map[string]string{"src/main.can": cyclic}); err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("arm description cycle: %v", err)
	}
}

func TestQuestionHandlerScopesRemainSeparate(t *testing.T) {
	declaration := `choice str dynamic from classifier
    emits [ai::invalid_question, ai::invalid_answer]
    given
        choice_option[] candidates
    confidence as certainty
    asks "Choose"
    minimum 0.5 => ok "fallback"
        ...candidates
        ok str selected => ok selected

score float rating from classifier
    emits [ai::invalid_question, ai::invalid_answer]
    score as measured
    confidence as certainty
    asks "Rate"
        low "Low"
        high "High"
        ok => ok measured + certainty
`
	text := nativeHeader + nativeClassifier + declaration + programMain + "    ok\n"
	if _, err := programFixture(t, map[string]string{"src/main.can": text}); err != nil {
		t.Fatal(err)
	}
	for name, bad := range map[string]string{
		"fallback selected": strings.Replace(text, `minimum 0.5 => ok "fallback"`, `minimum 0.5 => ok selected`, 1),
		"fallback probability": strings.Replace(text, `minimum 0.5 => ok "fallback"`, `minimum 0.5 => do
        float illegal = %
        ok "fallback"`, 1),
		"score probability":       strings.Replace(text, "ok measured + certainty", "ok %", 1),
		"duplicate metadata name": strings.Replace(text, "score as measured", "score as certainty", 1),
		"duplicate metadata kind": strings.Replace(text, "score as measured", "confidence as measured", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := programFixture(t, map[string]string{"src/main.can": bad}); err == nil {
				t.Fatal("invalid metadata/probability scope accepted")
			}
		})
	}
}

func TestGeneratedArmSpreadInfersOrdinaryGenericCalls(t *testing.T) {
	declarations := `record arms
    choice_arm<float> emits [] left
    choice_arm<float> emits [] right

fn item identity<item>
    emits []
    given
        item value
    asserts
        sample: 1 => ok 1
    ok value

fn item first<item>
    emits []
    given
        item[] values
    asserts
        sample: [1] => ok 1
    ok values[0]

fn item keep<item>
    emits []
    given
        item[] ignored
        item value
    asserts
        sample: [], 1 => ok 1
    ok value

fn item variadic<item>
    emits []
    given
        item ...values
    asserts
        sample: 1 => ok 1
    ok values[0]

record box<item>
    item value

fn item unwrap<item>
    on box<item> self
    emits []
    asserts
        sample: box<int>(1) => => ok 1
    ok self.value

record weights choice float weighted from classifier
    emits [ai::invalid_question, ai::invalid_answer]
    given
        arms choices
        box<arms> wrapped
    asks "Choose"
        ...SPREAD
`
	for _, spread := range []string{"call identity(choices)", "call identity<arms>(choices)", "call identity(call identity(choices))", "call first([choices])", "call keep([], choices)", "call variadic(...[choices])", "call wrapped.unwrap()"} {
		t.Run(spread, func(t *testing.T) {
			text := nativeHeader + nativeClassifier + strings.Replace(declarations, "SPREAD", spread, 1) + programMain + "    ok\n"
			p, err := programFixture(t, map[string]string{"src/main.can": text})
			if err != nil {
				t.Fatal(err)
			}
			fields := p.Natives[0].Signature.Result().Fields()
			if len(fields) != 2 || fields[0].Name != "left" || fields[1].Name != "right" {
				t.Fatal("inferred spread lost generated field order")
			}
		})
	}
	for name, change := range map[string][2]string{
		"ambiguous": {"identity<item>", "identity<item,unused>"},
		"conflict":  {"...SPREAD", "...call keep([1], choices)"},
		"arity":     {"...SPREAD", "...call identity<arms,int>(choices)"},
		"cycle":     {"arms choices", "weights choices"},
	} {
		t.Run(name, func(t *testing.T) {
			text := strings.Replace(declarations, change[0], change[1], 1)
			text = strings.Replace(text, "SPREAD", "call identity(choices)", 1)
			if _, err := programFixture(t, map[string]string{"src/main.can": nativeHeader + nativeClassifier + text + programMain + "    ok\n"}); err == nil {
				t.Fatal("invalid generic shape admitted")
			}
		})
	}
}
