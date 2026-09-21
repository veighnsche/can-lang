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

func TestCurrentBundledNoulJudge(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged Noul execution")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	sourceRoot, _ := filepath.Abs("../..")
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "judge-integration")
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	response := `{"model":"resolved-model","answers":{"q0":{"type":"noul","noul":0.5},"q1":{"type":"noul","noul":0.25},"q2":{"type":"noul","noul":1}}}`
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
		io.WriteString(w, response)
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
	data, err := os.ReadFile(filepath.Join(sourceRoot, "compiler/testdata/current/native/noul.can"))
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Replace(string(data), "http://127.0.0.1:1/systemone", server.URL+"/systemone", 1)
	write("src/main.can", text)
	run := func(command string) (int, string, string) {
		t.Helper()
		profile := `(version 1)(allow default)(deny network*)(allow network-outbound (remote ip "localhost:*"))`
		cmd := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-p", profile, filepath.Join(bundle, "bin/canlc"), command, root)
		cmd.Dir = t.TempDir()
		cmd.Env = []string{"PATH=/nonexistent", "HOME=" + cmd.Dir, "CAN_I17_TOKEN=test-only"}
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
	if code, out, diag := run("run"); code != 0 || out != "TFT" || diag != "" {
		t.Fatalf("judge execution: %d %q %s", code, out, diag)
	}
	if code, out, diag := run("assert"); code != 0 || diag != "" || !strings.Contains(out, "supplied-completion") {
		t.Fatalf("ordinary Noul consumer assertions: %d %s %s", code, out, diag)
	}
	mu.Lock()
	want := `{"model":"jev-latest","state":{"amount":9007199254740993,"signed_zero":-0},"questions":{"q0":{"type":"noul","instructions":"First","criteria":{"true":"Yes","false":"No"}},"q1":{"type":"noul","instructions":"Second","criteria":{"true":"Yes","false":"No"}},"q2":{"type":"noul","instructions":"Unused","criteria":{"true":"Yes","false":"No"}}}}`
	if requests != 1 || bodies[0] != want {
		t.Errorf("batch request count/body: %d %q", requests, bodies)
	}
	response = `{"model":"resolved-model","answers":{"q0":{"type":"noul","noul":0.5},"q1":{"type":"noul","noul":0.25},"q2":{"type":"noul","noul":2}}}`
	mu.Unlock()
	if code, out, diag := run("run"); code == 0 || out != "" || !strings.Contains(diag, "1121") {
		t.Fatalf("invalid later answer ran handlers: %d %q %s", code, out, diag)
	}
	write("src/main.can", strings.Replace(text, `likelihood("Unused")`, `likelihood("")`, 1))
	if code, out, diag := run("run"); code == 0 || out != "" || !strings.Contains(diag, "1120") {
		t.Fatalf("invalid descriptor launched: %d %q %s", code, out, diag)
	}
	mu.Lock()
	if requests != 2 {
		t.Fatalf("invalid descriptor issued request: %d", requests)
	}
	response = `{"model":"resolved-model","answers":{"q0":{"type":"noul","noul":0.5},"q1":{"type":"noul","noul":0.25},"q2":{"type":"noul","noul":1}}}`
	mu.Unlock()
	// The first handler fails after its observable T. Later handlers and the
	// continuation must not run, and the same request is never dispatched again.
	failing := strings.Replace(text, "ok int written => ok probability", "ok int written => do\n                int invalid = 1 / 0\n                ok probability", 1)
	write("src/main.can", failing)
	if code, out, diag := run("run"); code == 0 || out != "T" || diag == "" {
		t.Fatalf("handler failure did not stop judge: %d %q %s", code, out, diag)
	}
	mu.Lock()
	if requests != 3 {
		t.Fatalf("handler failure redispatched request: %d", requests)
	}
	mu.Unlock()

	// Conformance fixtures are private harness data, not Can syntax. Run the
	// compiled main and handlers through a raw HTTP fixture with all networking
	// denied. The same adapter must still encode and validate the complete batch.
	pure := strings.Replace(text, "    auth bearer env \"CAN_I17_TOKEN\"\n", "", 1)
	pure = strings.ReplaceAll(pure, "relay call report(%, \"T\")", "ok %")
	pure = strings.ReplaceAll(pure, "relay call report(%, \"F\")", "ok %")
	write("src/main.can", pure)
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
