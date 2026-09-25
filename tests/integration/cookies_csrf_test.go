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
)

func TestCurrentBundledCookiesCSRF(t *testing.T) {
	t.Parallel()
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged cookies execution")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	bundle, err := harnessBundle(t, ctx, archive)
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
	data, err := os.ReadFile(filepath.Join(sourceRoot, "compiler/testdata/current/cookies/session.can"))
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
		t.Fatalf("cookies assertions: %d %s %s", status, out, diag)
	}
	var report map[string]any
	if err = json.Unmarshal([]byte(out), &report); err != nil || report["passed"] != true || len(report["assertions"].([]any)) != 18 {
		t.Fatalf("invalid cookies report %v %s", err, out)
	}
	if !strings.Contains(out, "supplied-completion") || !strings.Contains(out, "real-can") {
		t.Fatalf("cookies report lacks scoped and real evidence: %s", out)
	}
	status, _, diag = run("run")
	if status != 0 || diag != "" {
		t.Fatalf("cookies execution: %d %s", status, diag)
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
		cmd := exec.CommandContext(ctx, filepath.Join(bundle, "runtime/bun"), tsc, "--noEmit", "--ignoreConfig", "--strict", "--skipLibCheck", "--target", "esnext", "--module", "esnext", "--moduleResolution", "bundler", "--allowImportingTsExtensions", "--typeRoots", filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(tsc))), "@types"), "--types", "bun,node", filepath.Join(build.Directory, "entry.ts"))
		if result, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("strict generated TypeScript: %v %s", err, result)
		}
	}
	names := map[string]string{}
	want := []string{"app::boot", "app::mint"}
	var module string
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
			for _, identity := range want {
				if strings.Contains(chunk[:line], "can.project.root/"+identity+"\"") {
					module = path
					names[identity] = chunk[:first]
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, identity := range want {
		if names[identity] == "" {
			t.Fatalf("missing generated session function %s", identity)
		}
	}
	runtimes, err := filepath.Glob(filepath.Join(build.Directory, "runtime", "r-*"))
	if err != nil || len(runtimes) != 1 {
		t.Fatal("missing runtime")
	}
	quote := func(s string) string { data, _ := json.Marshal(s); return string(data) }
	harness := fmt.Sprintf(`import {strict as assert} from "node:assert";
import {%s as boot, %s as mint} from %s;
import {$canInitialize, $canServer} from %s;
import {runOwnedRoot} from %s;
import {success} from %s;
$canInitialize();
const owned=await runOwnedRoot(async ()=>{
 const started=await boot(18374n);
 assert.equal(started.kind,"ok");
 const token=started.value;
 const granted=await fetch("http://127.0.0.1:18374/establish");
 assert.equal(granted.status,200);
 const fields=granted.headers.getSetCookie();
 assert.equal(fields.length,1);
 assert.ok(fields[0].startsWith("session=A;"));
 const minted=await mint("s3cret","A");
 assert.equal(minted.kind,"ok");
 const real=minted.value;
 const bare=await fetch("http://127.0.0.1:18374/login?token="+real);
 assert.equal(bare.status,422);assert.equal(await bare.text(),"bad");
 const welcome=await fetch("http://127.0.0.1:18374/login?token="+real,{headers:{cookie:"theme=dark; session=A"}});
 assert.equal(welcome.status,200);assert.equal(await welcome.text(),"welcome");
 const cross=await fetch("http://127.0.0.1:18374/login?token="+real,{headers:{cookie:"session=B"}});
 assert.equal(cross.status,422);assert.equal(await cross.text(),"bad");
 const bogus=await fetch("http://127.0.0.1:18374/login?token=bogus",{headers:{cookie:"session=A"}});
 assert.equal(bogus.status,422);assert.equal(await bogus.text(),"bad");
 assert.equal((await $canServer.stop(token)).kind,"ok");
 assert.equal((await $canServer.wait(token)).kind,"ok");
 return success(undefined);
});
assert.equal(owned.completion.kind,"ok");assert.equal(owned.cleanupFailed,false);
console.log("compiled cookies lifecycle passed");
`, names["app::boot"], names["app::mint"], quote(module), quote(filepath.Join(build.Directory, "program/state.ts")), quote(filepath.Join(runtimes[0], "owner.ts")), quote(filepath.Join(runtimes[0], "completion.ts")))

	harnessPath := filepath.Join(t.TempDir(), "session.ts")
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
		t.Fatalf("compiled cookies lifecycle: %v %s", err, result)
	}
}
