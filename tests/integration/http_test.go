package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/distribution"
)

func TestCurrentBundledHTTP(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged HTTP execution")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "http-integration")
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	write := func(name, text string) {
		t.Helper()
		p := filepath.Join(root, name)
		if err = os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(p, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("can.project.json", `{"source_root":"src","error_registry":"can.errors.json"}`)
	write("can.errors.json", `{"active":[],"retired":[]}`)
	data, err := os.ReadFile(filepath.Join(sourceRoot, "compiler/testdata/current/http/main.can"))
	if err != nil {
		t.Fatal(err)
	}
	write("src/main.can", string(data))
	run := func(command string, args ...string) (int, string, string) {
		t.Helper()
		argv := []string{"-p", "(version 1)(allow default)(deny network*)", filepath.Join(bundle, "bin/canlc"), command, root}
		argv = append(argv, args...)
		cmd := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", argv...)
		cmd.Dir = t.TempDir()
		cmd.Env = []string{"PATH=/nonexistent", "HOME=" + cmd.Dir}
		var out, diag bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &diag
		if err := cmd.Run(); err != nil {
			var status *exec.ExitError
			if !errors.As(err, &status) {
				t.Fatal(err)
			}
			return status.ExitCode(), out.String(), diag.String()
		}
		return 0, out.String(), diag.String()
	}
	status, out, diag := run("assert")
	if status != 0 || diag != "" {
		t.Fatalf("HTTP assertions: %d %s %s", status, out, diag)
	}
	var report map[string]any
	if err = json.Unmarshal([]byte(out), &report); err != nil || report["passed"] != true || len(report["assertions"].([]any)) != 13 {
		t.Fatalf("invalid HTTP report %v %s", err, out)
	}
	if !strings.Contains(out, "supplied-completion") || !strings.Contains(out, "real-can") {
		t.Fatalf("HTTP report lacks scoped and real evidence: %s", out)
	}
	status, _, diag = run("run")
	if status != 0 || diag != "" {
		t.Fatalf("HTTP execution: %d %s", status, diag)
	}
	status, out, diag = run("build")
	if status != 0 {
		t.Fatalf("build: %d %s %s", status, out, diag)
	}
	var build struct {
		Directory string `json:"directory"`
	}
	if err := json.Unmarshal([]byte(out), &build); err != nil {
		t.Fatal(err)
	}
	if tsc := os.Getenv("CAN_TSC"); tsc != "" {
		cmd := exec.CommandContext(ctx, filepath.Join(bundle, "runtime/bun"), tsc, "--noEmit", "--strict", "--skipLibCheck", "--target", "esnext", "--module", "esnext", "--moduleResolution", "bundler", "--allowImportingTsExtensions", "--typeRoots", filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(tsc))), "@types"), "--types", "bun,node", filepath.Join(build.Directory, "entry.ts"))
		if result, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("strict generated TypeScript: %v %s", err, result)
		}
	}
	var module, mountedName string
	err = filepath.WalkDir(build.Directory, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".ts") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		chunks := strings.Split(string(data), "export async function ")
		for _, chunk := range chunks[1:] {
			first := strings.Index(chunk, "(")
			line := strings.Index(chunk, "try {")
			if first < 0 || line < 0 {
				continue
			}
			if strings.Contains(chunk[:line], `can.project.root/app::mounted`) {
				module = path
				mountedName = chunk[:first]
			}
		}
		return nil
	})
	if err != nil || module == "" || mountedName == "" {
		t.Fatalf("missing generated mount function: %v", err)
	}
	runtimes, err := filepath.Glob(filepath.Join(build.Directory, "runtime", "r-*"))
	if err != nil || len(runtimes) != 1 {
		t.Fatal("missing runtime")
	}
	quote := func(s string) string { data, _ := json.Marshal(s); return string(data) }
	harness := fmt.Sprintf(`import {strict as assert} from "node:assert";
import {%s as mounted} from %s;
import {$canInitialize} from %s;
import {snapshotRequest} from %s;
import {dispatch} from %s;
$canInitialize();
const built=await mounted();assert.equal(built.kind,"ok");
// PRIVATE TEST BRIDGE (not the I33 server API): loopback ingress only.
const server=Bun.serve({port:0,hostname:"127.0.0.1",fetch:async native=>{
 const snap=await snapshotRequest(native,65536);
 if(snap.kind!=="request")return new Response(snap.status===413?"Content Too Large":"Bad Request",{status:snap.status});
 const routed=await dispatch(built.value,snap.value);
 assert.equal(routed.kind,"ok");return routed.value;
}});
const base="http://127.0.0.1:"+server.port;
const text=(response)=>response.text();
{
 const ok=await fetch(base+"/a?q=v");assert.equal(ok.status,200);assert.equal(await text(ok),"GET");
 const missing=await fetch(base+"/a");assert.equal(missing.status,422);assert.equal(await text(missing),"missing");
 const repeated=await fetch(base+"/a?q=a&q=b");assert.equal(repeated.status,422);
 const plus=await fetch(base+"/a?q=a+b");assert.equal(plus.status,200);
 const encoded=await fetch(base+"/a?q="+encodeURIComponent("a b"));assert.equal(encoded.status,200);
 assert.equal(ok.headers.get("content-type"),"text/plain; charset=utf-8");
}
{
 const created=await fetch(base+"/b",{method:"POST",headers:{"content-type":"application/json"},body:JSON.stringify({name:"x",count:2})});
 assert.equal(created.status,200);assert.equal(await text(created),"x");
 assert.equal(created.headers.get("content-type"),"text/plain; charset=utf-8");
 const media=await fetch(base+"/b",{method:"POST",headers:{"content-type":"text/plain"},body:"x"});
 assert.equal(media.status,422);assert.equal(await text(media),"bad");
 const malformed=await fetch(base+"/b",{method:"POST",headers:{"content-type":"application/json"},body:"{oops"});
 assert.equal(malformed.status,422);
}
{
 const form=await fetch(base+"/s",{method:"POST",headers:{"content-type":"application/x-www-form-urlencoded"},body:"query=ann&tags=x&tags=y"});
 assert.equal(form.status,200);assert.equal(await text(form),"ann");
 const noted=await fetch(base+"/s",{method:"POST",headers:{"content-type":"application/x-www-form-urlencoded"},body:"query=ann&note=hi"});
 assert.equal(noted.status,200);assert.equal(await text(noted),"ann");
 const empty=await fetch(base+"/s",{method:"POST",headers:{"content-type":"application/x-www-form-urlencoded"},body:"query=&tags=x"});
 assert.equal(empty.status,200);assert.equal(await text(empty),"");
 const absent=await fetch(base+"/s",{method:"POST",headers:{"content-type":"application/x-www-form-urlencoded"},body:"tags=x"});
 assert.equal(absent.status,422);
 const broken=await fetch(base+"/s",{method:"POST",headers:{"content-type":"application/x-www-form-urlencoded"},body:"query=%%FF&tags=x"});
 assert.equal(broken.status,422);
}
{
 const generic=await fetch(base+"/g");assert.equal(generic.status,200);assert.equal(await text(generic),"t");
 const wrapped=await fetch(base+"/w?q=v",{method:"POST"});assert.equal(wrapped.status,200);assert.equal(await text(wrapped),"POST");
}
{
 const missing=await fetch(base+"/nope");assert.equal(missing.status,404);assert.equal(await text(missing),"Not Found");
 const method=await fetch(base+"/a",{method:"POST"});assert.equal(method.status,405);assert.equal(method.headers.get("allow"),"GET");
 const reversed=await fetch(base+"/b");assert.equal(reversed.status,405);assert.equal(reversed.headers.get("allow"),"POST");
 const head=await fetch(base+"/a",{method:"HEAD"});assert.equal(head.status,405);assert.equal(head.headers.get("allow"),"GET");
}
server.stop();
console.log("compiled HTTP dispatch passed");
`, mountedName, quote(module), quote(filepath.Join(build.Directory, "program/state.ts")), quote(filepath.Join(runtimes[0], "platform/http.ts")), quote(filepath.Join(runtimes[0], "platform/router.ts")))
	harnessPath := filepath.Join(t.TempDir(), "http.ts")
	if err := os.WriteFile(harnessPath, []byte(harness), 0600); err != nil {
		t.Fatal(err)
	}
	environmentPath := filepath.Join(t.TempDir(), "environment.json")
	os.WriteFile(environmentPath, []byte("{}"), 0600)
	environment, err := os.Open(environmentPath)
	if err != nil {
		t.Fatal(err)
	}
	defer environment.Close()
	cmd := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-p", `(version 1)(allow default)(deny network*)(allow network-outbound (remote ip "localhost:*"))(allow network-bind (local ip "localhost:*"))(allow network-inbound (local ip "localhost:*"))`, filepath.Join(bundle, "runtime/bun"), "--no-install", "--no-env-file", harnessPath)
	cmd.Env = []string{"PATH=/nonexistent", "HOME=" + t.TempDir()}
	cmd.ExtraFiles = []*os.File{environment}
	if result, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compiled HTTP dispatch: %v %s", err, result)
	}
}
