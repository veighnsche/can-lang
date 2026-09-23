package check

import (
	"strings"
	"testing"
)

func TestLLMOutputSchemaAdmission(t *testing.T) {
	for _, tc := range []struct{ name, records, result, path string }{
		{"generic", "record box<item>\n    item value\n", "box<int>", ""},
		{"recursive", "record node\n    node[] children\n", "node", "/children/*"},
		{"opaque", "record bad\n    bytes::buffer value\n", "bad", "/value"},
		{"array root", "", "str[]", "/"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			asserts := ""
			if tc.path == "" {
				asserts = "    asserts\n        sample: () => ok box<int>(1)\n            using raw \"fixtures/generate.json\"\n"
			}
			source := strings.Replace(nativeHeader, "ai, llm", "ai, llm, bytes", 1) + nativeGenerator + tc.records + "llm " + tc.result + " generate from generator\n    emits [" + nativeLLM + "]\n" + asserts + "    asks \"Generate\"\n" + programMain + "    ok\n"
			p, err := programFixture(t, withNativeRaw(map[string]string{"src/main.can": source}, "generate"))
			if tc.path != "" {
				if err == nil || !strings.Contains(err.Error(), tc.path) {
					t.Fatalf("expected rejection at %s: %v", tc.path, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if p.Natives[0].LLM == nil || p.Natives[0].LLM.Format == nil {
				t.Fatal("missing structured generation plan")
			}
		})
	}
}

func TestLLMRejectsUnsupportedProviderSettings(t *testing.T) {
	for _, setting := range []string{"stream \"true\"", "tools \"search\"", "previous_response_id \"prior\""} {
		source := nativeHeader + strings.Replace(nativeGenerator, "        model \"test\"", "        model \"test\"\n        "+setting, 1) + "llm str generate from generator\n    emits [" + nativeLLM + "]\n    asks \"Generate\"\n" + programMain + "    ok\n"
		if _, err := programFixture(t, map[string]string{"src/main.can": source}); err == nil {
			t.Fatalf("accepted provider setting %s", setting)
		}
	}
}
