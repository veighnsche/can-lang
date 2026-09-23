package integration

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/distribution"
)

// restageQuestionsConfidence rewrites the staged assess response with the
// same confidence edits the live server serves, so verification replays the
// fallback answers the mutated source expects. It returns the previous
// staged bytes for restoration.
func restageQuestionsConfidence(t *testing.T, root, response string) []byte {
	t.Helper()
	path := filepath.Join(root, "src/fixtures/questions_assess.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var fixture map[string]any
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	outcome, ok := fixture["exchange"].(map[string]any)["outcome"].(map[string]any)
	if !ok {
		t.Fatal("assess fixture omits outcome")
	}
	answer, ok := outcome["response"].(map[string]any)
	if !ok {
		t.Fatal("assess fixture omits response")
	}
	answer["body_base64"] = base64.StdEncoding.EncodeToString([]byte(response))
	rewritten, err := json.Marshal(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, rewritten, 0600); err != nil {
		t.Fatal(err)
	}
	return data
}

func TestCurrentBundledMixedQuestions(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged mixed questions execution")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	sourceRoot, _ := filepath.Abs("../..")
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "questions-integration")
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	response := `{"model":"resolved-model","answers":{"q0":{"type":"noul","noul":0.5},"q1":{"type":"choice","choice":"second","confidence":0.5,"probabilities":{"first":0.5,"second":0.5}},"q2":{"type":"choice","choice":"__proto__","confidence":0.8,"probabilities":{"one":0.1,"__proto__":0.9}},"q3":{"type":"score","score":1,"confidence":0.5,"probabilities":{"0":0.25,"1":0.5,"2":0.25},"legend":{"0":"Low","1":"Medium","2":"High"}},"q4":{"type":"choice","choice":"second","confidence":0.5,"probabilities":{"first":0.5,"second":0.5}},"q5":{"type":"score","score":0.75,"confidence":0.5,"probabilities":{"0":0.25,"1":0.75},"legend":{"0":"Low","1":"High"}},"q6":{"type":"choice","choice":"left","confidence":0.5,"probabilities":{"left":0.5,"right":0.5}},"q7":{"type":"choice","choice":"left","confidence":0.5,"probabilities":{"before":0.25,"left":0.25,"right":0.25,"after":0.25}}}}`
	validResponse := response
	requests := 0
	var bodies []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		requests++
		data, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(data))
		if r.Method != "POST" || r.URL.Path != "/systemone" || r.Header.Get("Authorization") != "Bearer test-only" || r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("incorrect protocol request")
		}
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(string(data), `"previous":`) {
			want := `{"model":"jev-latest","state":{"previous":10},"questions":{"q0":{"type":"choice","instructions":"Dynamic","criteria":{"ten":"10","other":"Other"}}}}`
			if string(data) != want {
				t.Errorf("second-stage preparation lost prior result: %s", data)
			}
			io.WriteString(w, `{"model":"resolved","answers":{"q0":{"type":"choice","choice":"ten","confidence":1,"probabilities":{"ten":1,"other":0}}}}`)
		} else {
			io.WriteString(w, response)
		}
	}))
	defer server.Close()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	write := func(name, text string) {
		t.Helper()
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("can.project.json", `{"source_root":"src","error_registry":"can.errors.json"}`)
	write("can.errors.json", `{"active":[],"retired":[]}`)
	data, err := os.ReadFile(filepath.Join(sourceRoot, "compiler/testdata/current/native/questions.can"))
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Replace(string(data), "http://127.0.0.1:1/systemone", server.URL+"/systemone", 1)
	write("src/main.can", text)
	stageRawFixtures(t, write, sourceRoot, "native", [2]string{"http://127.0.0.1:1", server.URL})
	run := func(command string) (int, string, string) {
		t.Helper()
		profile := `(version 1)(allow default)(deny network*)(allow network-outbound (remote ip "localhost:*"))`
		cmd := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-p", profile, filepath.Join(bundle, "bin/canlc"), command, root)
		cmd.Dir = t.TempDir()
		cmd.Env = []string{"PATH=/nonexistent", "HOME=" + cmd.Dir, "CAN_I27_TOKEN=test-only"}
		var out, diag bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &diag
		err := cmd.Run()
		if err != nil {
			if e, ok := err.(*exec.ExitError); ok {
				return e.ExitCode(), out.String(), diag.String()
			}
			t.Fatal(err)
		}
		return 0, out.String(), diag.String()
	}
	if code, out, diag := run("run"); code != 0 || out != "NBSCDEFLGRLH" || diag != "" {
		t.Fatalf("judge execution: %d %q %s", code, out, diag)
	}
	if code, out, diag := run("assert"); code != 0 || diag != "" || !strings.Contains(out, "supplied-completion") {
		t.Fatalf("ordinary mixed-question consumer assertions: %d %s %s", code, out, diag)
	}
	mu.Lock()
	want := `{"model":"jev-latest","state":{"message":"evidence"},"questions":{"q0":{"type":"noul","instructions":"Noul","criteria":{"true":"Yes","false":"No"}},"q1":{"type":"choice","instructions":"Choice","criteria":{"first":"First","second":"Second"}},"q2":{"type":"choice","instructions":"Dynamic","criteria":{"one":"One","__proto__":"Other"}},"q3":{"type":"score","instructions":"Score","criteria":["Low","Medium","High"]},"q4":{"type":"choice","instructions":"Weights","criteria":{"first":"First","second":"Second"}},"q5":{"type":"score","instructions":"Levels","criteria":["Low","High"]},"q6":{"type":"choice","instructions":"Spread","criteria":{"left":"Left","right":"Right"}},"q7":{"type":"choice","instructions":"Spread weights","criteria":{"before":"Before","left":"Right","right":"Left","after":"After"}}}}`
	if requests != 1 || bodies[0] != want {
		t.Errorf("batch request count/body: %d %q", requests, bodies)
	}
	response = `{"model":"resolved-model","answers":{"q0":{"type":"noul","noul":0.5},"q1":{"type":"choice","choice":"second","confidence":0.5,"probabilities":{"first":0.5,"second":0.5}},"q2":{"type":"choice","choice":"__proto__","confidence":0.8,"probabilities":{"one":0.1,"__proto__":0.9}},"q3":{"type":"score","score":1,"confidence":0.5,"probabilities":{"0":0.25,"1":0.5,"2":0.25},"legend":{"0":"Low","1":"Medium","2":"High"}},"q4":{"type":"choice","choice":"second","confidence":0.5,"probabilities":{"first":0.5,"second":0.5}},"q5":{"type":"score","score":0.75,"confidence":0.5,"probabilities":{"0":0.25,"1":0.75},"legend":{"0":"Low","1":"High"}},"q6":{"type":"choice","choice":"left","confidence":0.5,"probabilities":{"left":0.5,"right":0.5}},"q7":{"type":"choice","choice":"left","confidence":0.5,"probabilities":{"before":0.25,"left":2,"right":0.25,"after":0.25}}}}`
	mu.Unlock()
	if code, out, diag := run("run"); code == 0 || out != "" || !strings.Contains(diag, "1121") {
		t.Fatalf("invalid later answer ran handlers: %d %q %s", code, out, diag)
	}
	// A duplicate criterion cannot verify, so the gate rejects before any
	// launch; descriptor validation itself is covered in questions.test.ts.
	write("src/main.can", strings.Replace(text, `choice_option("__proto__", "Other")`, `choice_option("one", "Other")`, 1))
	if code, out, diag := run("run"); code == 0 || out != "" || !strings.Contains(diag, "build verification failed") || !strings.Contains(diag, "unused fixture") {
		t.Fatalf("invalid descriptor launched: %d %q %s", code, out, diag)
	}
	mu.Lock()
	if requests != 2 {
		t.Fatalf("invalid descriptor issued request: %d", requests)
	}
	response = `{"model":"resolved-model","answers":{"q0":{"type":"noul","noul":0.5},"q1":{"type":"choice","choice":"second","confidence":0.5,"probabilities":{"first":0.5,"second":0.5}},"q2":{"type":"choice","choice":"__proto__","confidence":0.8,"probabilities":{"one":0.1,"__proto__":0.9}},"q3":{"type":"score","score":1,"confidence":0.5,"probabilities":{"0":0.25,"1":0.5,"2":0.25},"legend":{"0":"Low","1":"Medium","2":"High"}},"q4":{"type":"choice","choice":"second","confidence":0.5,"probabilities":{"first":0.5,"second":0.5}},"q5":{"type":"score","score":0.75,"confidence":0.5,"probabilities":{"0":0.25,"1":0.75},"legend":{"0":"Low","1":"High"}},"q6":{"type":"choice","choice":"left","confidence":0.5,"probabilities":{"left":0.5,"right":0.5}},"q7":{"type":"choice","choice":"left","confidence":0.5,"probabilities":{"before":0.25,"left":0.25,"right":0.25,"after":0.25}}}}`
	mu.Unlock()
	// The first handler carries a fault, so the program cannot verify: the
	// gate rejects before anything launches, and the same request is never
	// dispatched again.
	failing := strings.Replace(text, "ok int written => ok probability", "ok int written => do\n                int invalid = 1 / 0\n                ok probability", 1)
	write("src/main.can", failing)
	if code, out, diag := run("run"); code == 0 || out != "" || !strings.Contains(diag, "build verification failed") {
		t.Fatalf("handler fault launched judge: %d %q %s", code, out, diag)
	}
	mu.Lock()
	if requests != 2 {
		t.Fatalf("handler fault dispatched request: %d", requests)
	}
	mu.Unlock()

	// A fault inside a generated record cannot verify: the gate rejects
	// before anything launches, so no partial record escapes and no request
	// is dispatched.
	partial := strings.Replace(text, `second "Second" => relay call report(%, "D")`, `second "Second" => do
            int invalid = 1 / 0
            ok %`, 1)
	write("src/main.can", partial)
	if code, out, diag := run("run"); code == 0 || out != "" || !strings.Contains(diag, "build verification failed") {
		t.Fatalf("partial record launched: %d %q %s", code, out, diag)
	}
	mu.Lock()
	if requests != 2 {
		t.Fatalf("partial record dispatched request: %d", requests)
	}
	mu.Unlock()
	// Equality used the normal handlers above; strictly lower confidence uses
	// fallback with metadata, leaving generated records unaffected. Run
	// verifies first, so the staged fixture serves the same lowered answers
	// the live server returns, and is restored before the pristine followups.
	mu.Lock()
	response = strings.Replace(validResponse, `"q1":{"type":"choice","choice":"second","confidence":0.5`, `"q1":{"type":"choice","choice":"second","confidence":0.25`, 1)
	response = strings.Replace(response, `"q3":{"type":"score","score":1,"confidence":0.5`, `"q3":{"type":"score","score":1,"confidence":0.25`, 1)
	mu.Unlock()
	stagedAssess := restageQuestionsConfidence(t, root, response)
	write("src/main.can", strings.ReplaceAll(text, "10.0", "8.0"))
	if code, out, diag := run("run"); code != 0 || out != "NfgCDEFLGRLH" || diag != "" {
		t.Fatalf("confidence fallback: %d %q %s", code, out, diag)
	}
	mu.Lock()
	response = validResponse
	mu.Unlock()
	if err := os.WriteFile(filepath.Join(root, "src/fixtures/questions_assess.json"), stagedAssess, 0600); err != nil {
		t.Fatal(err)
	}
	// A later request may depend on the fully completed first judge.
	followup := `judge void complete from classifier
    emits [http::request_failed, ai::invalid_question, ai::invalid_answer]
    state
        float previous
    asserts
        sample: (0.5) => ok
            using raw "fixtures/complete.json"
    call dynamic([choice_option("ten", call text::from_float(previous))], [choice_option("other", "Other")]) as str selected
    ok => match selected is "ten"
        true => ok
        false => do
            int invalid = 1 / 0
            ok

`
	twoStage := strings.Replace(text, "uses [ai, http, codec, bytes, io]", "uses [ai, http, codec, bytes, io, text]", 1)
	twoStage = strings.Replace(twoStage, "fn void main", followup+"fn void main", 1)
	// Staged main supplies the nested followup completion through a when
	// row; the live run still dispatches the real second request.
	twoStage = strings.Replace(twoStage, "            true => ok\n", "            true => match call complete((measured))\n                when\n                    sample: (10.0) => ok\n                http::request_failed\n                ai::invalid_question\n                ai::invalid_answer\n                ok => ok\n", 1)
	// The followup exchange is fully request-compared: the staged request
	// carries the 0.5 assert argument and the staged credential, and the
	// staged answer selects ten.
	completeRequest := `{"model":"jev-latest","state":{"previous":0.5},"questions":{"q0":{"type":"choice","instructions":"Dynamic","criteria":{"ten":"0.5","other":"Other"}}}}`
	completeAnswer := `{"model":"resolved","answers":{"q0":{"type":"choice","choice":"ten","confidence":1,"probabilities":{"ten":1,"other":0}}}}`
	completeFixture := map[string]any{
		"schema": "can.native-fixture.v1", "target": "can.project.root/app::complete",
		"environment": map[string]any{"CAN_I27_TOKEN": "fixture-secret"},
		"exchange": map[string]any{
			"request": map[string]any{"method": "POST", "url": server.URL + "/systemone",
				"headers": []any{[]any{"authorization", "Bearer fixture-secret"}, []any{"content-type", "application/json"}, []any{"accept", "application/json"}},
				"body":    map[string]any{"json_utf8": completeRequest}},
			"outcome": map[string]any{"response": map[string]any{"status": 200,
				"headers":     []any{[]any{"content-type", "application/json"}},
				"body_base64": base64.StdEncoding.EncodeToString([]byte(completeAnswer))}},
		},
	}
	completeBytes, err := json.Marshal(completeFixture)
	if err != nil {
		t.Fatal(err)
	}
	write("src/fixtures/complete.json", string(completeBytes))
	write("src/main.can", twoStage)
	if code, out, diag := run("run"); code != 0 || out != "NBSCDEFLGRLH" || diag != "" {
		t.Fatalf("two-stage dependency: %d %q %s", code, out, diag)
	}
	mu.Lock()
	// Two verified launches plus one invalid-answer launch before the gate
	// rejections, one confidence-fallback launch, and two two-stage launches.
	if requests != 5 {
		t.Errorf("mixed phase request count: %d", requests)
	}
	mu.Unlock()

	// Conformance fixtures are private harness data, not Can syntax. Run the
	// compiled main and handlers through a raw HTTP fixture with all networking
	// denied. The same adapter must still encode and validate the complete batch.
	pure := strings.Replace(text, "    auth bearer env \"CAN_I27_TOKEN\"\n", "", 1)
	pure = regexp.MustCompile(`relay call report\(([^\n]+), "[A-Za-z]"\)`).ReplaceAllString(pure, "ok $1")
	write("src/main.can", pure)
	clearStagedFixtureEnvironments(t, root)
	stripStagedAuthorization(t, root)
	code, out, diag := run("build")
	if code != 0 {
		t.Fatalf("raw-provider build: %d %s %s", code, out, diag)
	}
	var build struct {
		Directory string `json:"directory"`
	}
	if err := json.Unmarshal([]byte(out), &build); err != nil {
		t.Fatal(err)
	}
	if tsc := os.Getenv("CAN_TSC"); tsc != "" {
		cmd := exec.CommandContext(ctx, filepath.Join(bundle, "runtime/bun"), tsc, "--noEmit", "--strict", "--skipLibCheck", "--target", "esnext", "--module", "esnext", "--moduleResolution", "bundler", "--allowImportingTsExtensions", "--typeRoots", filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(tsc))), "@types"), "--types", "bun,node", filepath.Join(build.Directory, "entry.ts"))
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("mixed question strict TypeScript: %v %s", err, output)
		}
	}
	entry, err := os.ReadFile(filepath.Join(build.Directory, "entry.ts"))
	if err != nil {
		t.Fatal(err)
	}
	mainImport := regexp.MustCompile(`import \{ (\$canFunction[0-9]+) as \$canMain \} from "([^"]+)";`).FindStringSubmatch(string(entry))
	if len(mainImport) != 3 {
		t.Fatal("missing compiled main import")
	}
	runtimes, err := filepath.Glob(filepath.Join(build.Directory, "runtime", "r-*"))
	if err != nil || len(runtimes) != 1 {
		t.Fatal("missing shared runtime")
	}
	quote := func(s string) string { data, _ := json.Marshal(s); return string(data) }
	harness := fmt.Sprintf(`import {strict as assert} from "node:assert";
import {%s as main} from %s;
import {$canInitialize as initialize} from %s;
import {runAssertion} from %s;
import {provideHTTP} from %s;
import {success} from %s;
initialize();
const encode=(s:string)=>new TextEncoder().encode(s);
const report=await runAssertion({root:{package:"conformance",declaration:"compiled_judge",name:"raw_provider"},actual:async context=>{
 provideHTTP(context,[{request:{method:"POST",url:%s,headers:[["accept","application/json"],["content-type","application/json"]],body:encode(%s)},response:{status:200,headers:[["content-type","application/json"]],body:encode(%s)}}]);
 return main([],context);
},expected:async()=>success(undefined)});
assert.equal(report.passed,true,JSON.stringify(report));
assert.deepEqual(report.evidence,["raw-provider-fixture","real-can"]);
console.log(JSON.stringify(report));
`, mainImport[1], quote(filepath.Join(build.Directory, mainImport[2])), quote(filepath.Join(build.Directory, "program/state.ts")), quote(filepath.Join(runtimes[0], "assert/runner.ts")), quote(filepath.Join(runtimes[0], "assert/provider.ts")), quote(filepath.Join(runtimes[0], "completion.ts")), quote(server.URL+"/systemone"), quote(want), quote(response))
	harnessPath := filepath.Join(t.TempDir(), "raw-provider.ts")
	if err := os.WriteFile(harnessPath, []byte(harness), 0600); err != nil {
		t.Fatal(err)
	}
	environmentPath := filepath.Join(t.TempDir(), "environment.json")
	if err := os.WriteFile(environmentPath, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	environment, err := os.Open(environmentPath)
	if err != nil {
		t.Fatal(err)
	}
	defer environment.Close()
	cmd := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-p", "(version 1)(allow default)(deny network*)", filepath.Join(bundle, "runtime/bun"), "--no-install", "--no-env-file", harnessPath)
	cmd.Env = []string{"PATH=/nonexistent", "HOME=" + t.TempDir()}
	cmd.ExtraFiles = []*os.File{environment}
	result, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled raw provider assertion: %v %s", err, result)
	}
	t.Log(strings.TrimSpace(string(result)))
}
