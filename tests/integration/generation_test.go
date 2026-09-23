package integration

import (
	"bytes"
	"context"
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

func TestCurrentBundledGeneration(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	sourceRoot, _ := filepath.Abs("../..")
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "generation-integration")
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var bodies, urls, responses []string
	mode := "normal"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		data, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(data))
		urls = append(urls, r.URL.Path)
		if r.Method != "POST" || r.Header.Get("Authorization") != "Bearer test-only" || r.Header.Get("Content-Type") != "application/json" {
			t.Error("invalid generation request headers")
		}
		var request map[string]any
		if err := json.Unmarshal(data, &request); err != nil {
			t.Fatal(err)
		}
		response := `{"model":"fixture","answers":{"q0":{"type":"choice","choice":"first","confidence":1,"probabilities":{"first":1,"second":0}}}}`
		if r.URL.Path == "/responses" {
			output := "description"
			if request["instructions"] == "Plan" {
				output = `{"choices":[{"label":"First","count":9007199254740993.0}],"ready":true,"ratio":0.5}`
			}
			encoded, _ := json.Marshal(output)
			response = `{"id":"response-1","object":"response","status":"completed","output":[{"type":"message","role":"assistant","status":"completed","content":[{"type":"output_text","text":` + string(encoded) + `}]}]}`
			if mode == "refusal" {
				response = `{"id":"r","object":"response","status":"completed","output":[{"type":"message","role":"assistant","status":"completed","content":[{"type":"refusal","refusal":"private"}]}]}`
			}
			if mode == "truncated" {
				response = `{"id":"r","object":"response","status":"incomplete","incomplete_details":{"reason":"max_output_tokens"},"output":[]}`
			}
			if mode == "invalid" {
				response = `{"id":"r","object":"response","status":"completed","output":[]}`
			}
			if mode == "record" && request["instructions"] == "Plan" {
				response = strings.Replace(response, "9007199254740993.0", "1.5", 1)
			}
		}
		responses = append(responses, response)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, response)
	}))
	defer server.Close()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	write := func(name, text string) {
		t.Helper()
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("can.project.json", `{"source_root":"src","error_registry":"can.errors.json"}`)
	write("can.errors.json", `{"active":[],"retired":[]}`)
	data, err := os.ReadFile(filepath.Join(sourceRoot, "compiler/testdata/current/native/generation.can"))
	if err != nil {
		t.Fatal(err)
	}
	source := strings.ReplaceAll(string(data), "http://127.0.0.1:1", server.URL)
	stageRawFixtures(t, write, sourceRoot, "native", [2]string{"http://127.0.0.1:1", server.URL})
	write("src/main.can", source)
	credential := "test-only"
	run := func(command string) (int, string, string) {
		t.Helper()
		cmd := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-p", `(version 1)(allow default)(deny network*)(allow network-outbound (remote ip "localhost:*"))`, filepath.Join(bundle, "bin/canlc"), command, root)
		cmd.Dir = t.TempDir()
		cmd.Env = []string{"PATH=/nonexistent", "HOME=" + cmd.Dir, "CAN_I28_TOKEN=" + credential}
		var out, diag bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &diag
		if err := cmd.Run(); err != nil {
			if e, ok := err.(*exec.ExitError); ok {
				return e.ExitCode(), out.String(), diag.String()
			}
			t.Fatal(err)
		}
		return 0, out.String(), diag.String()
	}
	if code, out, diag := run("run"); code != 0 || out != "" || diag != "" {
		t.Fatalf("generation execution: %d %s %s", code, out, diag)
	}
	if code, out, diag := run("assert"); code != 0 || diag != "" || !strings.Contains(out, "supplied-completion") {
		t.Fatalf("consumer assertions: %d %s %s", code, out, diag)
	}
	mu.Lock()
	if len(bodies) != 3 {
		t.Fatalf("expected text, record, Choice requests: %d", len(bodies))
	}
	goodBodies := append([]string{}, bodies...)
	goodURLs := append([]string{}, urls...)
	goodResponses := append([]string{}, responses...)
	mu.Unlock()
	for i := 0; i < 2; i++ {
		var req map[string]any
		json.Unmarshal([]byte(goodBodies[i]), &req)
		if len(req) != 9 || req["model"] != "fixture-model" || req["max_output_tokens"] != float64(512) || req["store"] != false || req["stream"] != false || req["background"] != false || req["truncation"] != "disabled" {
			t.Fatalf("wrong request profile %s", goodBodies[i])
		}
		input := `{"count":9007199254740993}`
		asks := "Describe"
		if i == 1 {
			input = `{"description":"description"}`
			asks = "Plan"
		}
		if req["input"] != input || req["instructions"] != asks || strings.Contains(goodBodies[i], "private-not-disclosed") {
			t.Fatalf("state disclosure or asks mismatch %s", goodBodies[i])
		}
		format := req["text"].(map[string]any)["format"].(map[string]any)
		if i == 0 {
			if len(format) != 1 || format["type"] != "text" {
				t.Fatal(format)
			}
		} else {
			if format["type"] != "json_schema" || format["strict"] != true || !regexp.MustCompile(`^can_[0-9a-f]{32}$`).MatchString(format["name"].(string)) {
				t.Fatal(format)
			}
			schema := format["schema"].(map[string]any)
			if schema["additionalProperties"] != false || len(schema["required"].([]any)) != 3 {
				t.Fatal(schema)
			}
			item := schema["properties"].(map[string]any)["choices"].(map[string]any)["items"].(map[string]any)
			if item["additionalProperties"] != false || item["properties"].(map[string]any)["count"].(map[string]any)["type"] != "integer" {
				t.Fatal(item)
			}
		}
	}
	wantChoice := `{"model":"jev-latest","state":{"generated":{"choices":[{"label":"First","count":9007199254740993}],"ready":true,"ratio":0.5}},"questions":{"q0":{"type":"choice","instructions":"Choose","criteria":{"first":"First transformed","second":"Other"}}}}`
	if goodBodies[2] != wantChoice {
		t.Fatalf("transformed Choice request: %s", goodBodies[2])
	}
	for _, bad := range []struct {
		mode, id string
		count    int
	}{{"refusal", "1130", 1}, {"truncated", "1131", 1}, {"invalid", "1132", 1}, {"record", "1110", 2}} {
		mu.Lock()
		mode = bad.mode
		before := len(bodies)
		mu.Unlock()
		if code, out, diag := run("run"); code == 0 || out != "" || !strings.Contains(diag, bad.id) {
			t.Fatalf("%s: %d %s %s", bad.mode, code, out, diag)
		}
		mu.Lock()
		count := len(bodies) - before
		mu.Unlock()
		if count != bad.count {
			t.Fatalf("%s retried or advanced: %d", bad.mode, count)
		}
	}
	credential = ""
	write("src/main.can", strings.Replace(source, `asks "Describe"`, `asks " "`, 1))
	mu.Lock()
	before := len(bodies)
	mu.Unlock()
	if code, _, diag := run("run"); code == 0 || !strings.Contains(diag, "1100") {
		t.Fatalf("asks validation did not precede credential: %d %s", code, diag)
	}
	mu.Lock()
	after := len(bodies)
	mu.Unlock()
	if after != before {
		t.Fatal("invalid asks launched request")
	}
	write("src/main.can", strings.ReplaceAll(source, "    auth bearer env \"CAN_I28_TOKEN\"\n", ""))
	clearStagedFixtureEnvironments(t, root)
	stripStagedAuthorization(t, root)
	code, out, diag := run("build")
	if code != 0 {
		t.Fatalf("build: %d %s %s", code, out, diag)
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
			t.Fatalf("generation strict TypeScript: %v %s", err, output)
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
	fixtures := []string{}
	for i := range goodBodies {
		fixtures = append(fixtures, fmt.Sprintf(`{request:{method:"POST",url:%s,headers:[["accept","application/json"],["content-type","application/json"]],body:encode(%s)},response:{status:200,headers:[["content-type","application/json"]],body:encode(%s)}}`, quote(server.URL+goodURLs[i]), quote(goodBodies[i]), quote(goodResponses[i])))
	}
	harness := fmt.Sprintf(`import {strict as assert} from "node:assert";
import {%s as main} from %s;
import {$canInitialize as initialize} from %s;
import {runAssertion} from %s;
import {provideHTTP} from %s;
import {success} from %s;
initialize();const encode=(s:string)=>new TextEncoder().encode(s);
const report=await runAssertion({root:{package:"conformance",declaration:"compiled_generation",name:"raw_provider"},actual:async context=>{
provideHTTP(context,[%s]);return main([],context);
},expected:async()=>success(undefined)});
assert.equal(report.passed,true,JSON.stringify(report));assert.deepEqual(report.evidence,["raw-provider-fixture","real-can"]);
console.log(JSON.stringify(report));
`, mainImport[1], quote(filepath.Join(build.Directory, mainImport[2])), quote(filepath.Join(build.Directory, "program/state.ts")), quote(filepath.Join(runtimes[0], "assert/runner.ts")), quote(filepath.Join(runtimes[0], "assert/provider.ts")), quote(filepath.Join(runtimes[0], "completion.ts")), strings.Join(fixtures, ","))
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
