package emit

import (
	"strings"
	"testing"
)

func originProgram(t *testing.T, fixture string) string {
	t.Helper()
	program := sourceProgram(t, fixture)
	artifacts, err := ProgramModules(program, "runtime", httpDependencies(t))
	if err != nil {
		t.Fatal(err)
	}
	var bodies []string
	for _, artifact := range artifacts {
		bodies = append(bodies, string(artifact.Bytes))
	}
	return strings.Join(bodies, "\n")
}

// Boundary machinery receives the statically known producing operation, and
// authored descriptor/preparation evaluation lowers before the native launch,
// so argument failures stay outside the wrapper/normalization boundary.
func TestFetchEmissionTagsProducingOperation(t *testing.T) {
	joined := originProgram(t, "../../testdata/current/fetch/main.can")
	if got := strings.Count(joined, "$canFetch.request<"); got != 8 {
		t.Fatalf("expected 8 lowered fetch launches, got %d", got)
	}
	for _, name := range []string{"load_json", "load_envelope", "send_text", "send_json", "send_bytes", "head_text", "delete_text", "options_text"} {
		if !strings.Contains(joined, `$canOrigin,"can.project.root/fetches::`+name+`",$canContext`) {
			t.Fatalf("fetch %s launch misses its producing operation", name)
		}
	}
	for _, chunk := range strings.Split(joined, "async function $canNative") {
		at := strings.Index(chunk, "$canFetch.request<")
		if at < 0 {
			continue
		}
		if strings.Index(chunk[:at], "const ") < 0 {
			t.Fatal("fetch launch precedes its descriptor evaluation")
		}
	}
}

func TestJudgeEmissionTagsProducingOperation(t *testing.T) {
	joined := originProgram(t, "../../testdata/current/native/noul.can")
	if got := strings.Count(joined, "$canAI.ask("); got != 1 {
		t.Fatalf("expected 1 lowered judge launch, got %d", got)
	}
	if !strings.Contains(joined, `$canOrigin,"can.project.root/app::assess",$canContext`) {
		t.Fatal("judge launch misses its producing operation")
	}
	for _, chunk := range strings.Split(joined, "async function $canNative") {
		at := strings.Index(chunk, "$canAI.ask(")
		if at < 0 {
			continue
		}
		if strings.Index(chunk[:at], "$canRecord(") < 0 {
			t.Fatal("judge launch precedes its state preparation")
		}
	}
}

func TestLLMEmissionTagsProducingOperation(t *testing.T) {
	joined := originProgram(t, "../../testdata/current/native/generation.can")
	if got := strings.Count(joined, "$canResponses.generate<"); got != 2 {
		t.Fatalf("expected 2 lowered LLM launches, got %d", got)
	}
	for _, name := range []string{"describe", "generate"} {
		if !strings.Contains(joined, `$canOrigin,"can.project.root/generation::`+name+`",$canContext`) {
			t.Fatalf("LLM %s launch misses its producing operation", name)
		}
	}
}
