package check

import (
	"os"
	"testing"
)

// Raw native obligations N stay distinct from declared authored obligations E
// per native declaration, keyed by exact error identity so one identity may
// belong to both sets with different provenance.
func TestNativeOriginSets(t *testing.T) {
	fetchSource, err := os.ReadFile("../../testdata/current/fetch/main.can")
	if err != nil {
		t.Fatal(err)
	}
	fetch, err := programFixture(t, map[string]string{"src/main.can": string(fetchSource)})
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]*NativeDeclaration{}
	for _, native := range fetch.Natives {
		byName[native.Symbol.Name] = native
	}
	in := func(set []string, identity string) bool {
		for _, member := range set {
			if member == identity {
				return true
			}
		}
		return false
	}
	codec := "can.std.codec@1::invalid_data"
	status := "can.std.http@1::status_error"
	loadJSON := byName["load_json"]
	if loadJSON == nil || loadJSON.Fetch == nil {
		t.Fatal("load_json native plan missing")
	}
	if len(loadJSON.Native) != 7 || !in(loadJSON.Native, codec) {
		t.Fatalf("load_json N is not the seven raw leaves: %v", loadJSON.Native)
	}
	if len(loadJSON.Emitted) != 7 {
		t.Fatalf("load_json E is not the seven declared errors: %v", loadJSON.Emitted)
	}
	for _, identity := range loadJSON.Native {
		if !in(loadJSON.Emitted, identity) {
			t.Fatalf("load_json N member %s missing from E: %v", identity, loadJSON.Emitted)
		}
	}
	envelope := byName["load_envelope"]
	if envelope == nil || envelope.Fetch == nil {
		t.Fatal("load_envelope native plan missing")
	}
	if len(envelope.Native) != 5 || in(envelope.Native, status) || in(envelope.Native, codec) {
		t.Fatalf("envelope N keeps status/codec obligations: %v", envelope.Native)
	}
	sendBytes := byName["send_bytes"]
	if sendBytes == nil {
		t.Fatal("send_bytes native plan missing")
	}
	if in(sendBytes.Native, codec) {
		t.Fatalf("bytes N keeps a codec obligation: %v", sendBytes.Native)
	}
	for _, native := range []*NativeDeclaration{loadJSON, envelope, sendBytes} {
		if len(native.Emitted) != 7 {
			t.Fatalf("fetch %s E is not its declared errors: %v", native.Symbol.Name, native.Emitted)
		}
		if len(native.Fetch.Native) != len(native.Native) || len(native.Fetch.Emitted) != len(native.Emitted) {
			t.Fatalf("IR fetch plan drops origin sets for %s", native.Symbol.Name)
		}
	}

	judgeSource, err := os.ReadFile("../../testdata/current/native/noul.can")
	if err != nil {
		t.Fatal(err)
	}
	judged, err := programFixture(t, map[string]string{"src/main.can": string(judgeSource)})
	if err != nil {
		t.Fatal(err)
	}
	var assess *NativeDeclaration
	var likelihood *NativeDeclaration
	for _, native := range judged.Natives {
		if native.Symbol.Name == "assess" {
			assess = native
		}
		if native.Symbol.Name == "likelihood" {
			likelihood = native
		}
	}
	if assess == nil || assess.Judge == nil {
		t.Fatal("assess native plan missing")
	}
	if len(assess.Native) != 7 || in(assess.Native, "can.std.ai@1::invalid_question") || in(assess.Native, "can.std.ai@1::invalid_answer") {
		t.Fatalf("judge N mixes AI validation into raw infrastructure: %v", assess.Native)
	}
	for _, identity := range []string{"can.std.ai@1::invalid_question", "can.std.ai@1::invalid_answer", codec, "can.std.io@1::write_failed"} {
		if !in(assess.Emitted, identity) {
			t.Fatalf("judge E drops %s: %v", identity, assess.Emitted)
		}
	}
	if !in(assess.Native, codec) {
		t.Fatalf("judge N drops its intrinsic codec obligation: %v", assess.Native)
	}
	if len(assess.Judge.Native) != len(assess.Native) || len(assess.Judge.Emitted) != len(assess.Emitted) {
		t.Fatal("IR judge plan drops origin sets")
	}
	generationSource, err := os.ReadFile("../../testdata/current/native/generation.can")
	if err != nil {
		t.Fatal(err)
	}
	generated, err := programFixture(t, map[string]string{"src/main.can": string(generationSource)})
	if err != nil {
		t.Fatal(err)
	}
	var describe *NativeDeclaration
	for _, native := range generated.Natives {
		if native.Symbol.Name == "describe" {
			describe = native
		}
	}
	if describe == nil {
		t.Fatal("describe native plan missing")
	}
	if len(describe.Native) != 0 {
		t.Fatalf("LLM holds raw obligations: %v", describe.Native)
	}
	if len(describe.Emitted) != 10 {
		t.Fatalf("LLM E is not its declared errors: %v", describe.Emitted)
	}
	if likelihood == nil {
		t.Fatal("likelihood native plan missing")
	}
	if len(likelihood.Native) != 0 {
		t.Fatalf("question holds raw obligations: %v", likelihood.Native)
	}
	if len(likelihood.Emitted) != 4 {
		t.Fatalf("question E is not its declared errors: %v", likelihood.Emitted)
	}
}
