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

func TestCurrentBundledTLS(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged TLS execution")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "tls-integration")
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
	data, err := os.ReadFile(filepath.Join(sourceRoot, "compiler/testdata/current/http/tls.can"))
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
		t.Fatalf("TLS assertions: %d %s %s", status, out, diag)
	}
	var report map[string]any
	if err = json.Unmarshal([]byte(out), &report); err != nil || report["passed"] != true || len(report["assertions"].([]any)) != 5 {
		t.Fatalf("invalid TLS report %v %s", err, out)
	}
	if !strings.Contains(out, "supplied-completion") || !strings.Contains(out, "real-can") {
		t.Fatalf("TLS report lacks scoped and real evidence: %s", out)
	}
	status, _, diag = run("run")
	if status != 0 || diag != "" {
		t.Fatalf("TLS execution: %d %s", status, diag)
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
	var module, bootName string
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
			if strings.Contains(chunk[:line], `can.project.root/app::boot_tls`) {
				module = path
				bootName = chunk[:first]
			}
		}
		return nil
	})
	if err != nil || module == "" || bootName == "" {
		t.Fatalf("missing generated TLS boot function: %v", err)
	}
	runtimes, err := filepath.Glob(filepath.Join(build.Directory, "runtime", "r-*"))
	if err != nil || len(runtimes) != 1 {
		t.Fatal("missing runtime")
	}
	// Fresh local chain per run: nothing credential-shaped is committed.
	chainDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(chainDir, "san.cnf"), []byte("[req]\ndistinguished_name=dn\nreq_extensions=v3\n[dn]\n[v3]\nsubjectAltName=IP:127.0.0.1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	openssl := exec.CommandContext(ctx, "openssl", "req", "-x509", "-newkey", "rsa:2048", "-keyout", filepath.Join(chainDir, "key.pem"), "-out", filepath.Join(chainDir, "cert.pem"), "-days", "2", "-nodes", "-subj", "/CN=127.0.0.1", "-config", filepath.Join(chainDir, "san.cnf"), "-extensions", "v3")
	if result, err := openssl.CombinedOutput(); err != nil {
		t.Fatalf("openssl chain: %v %s", err, result)
	}
	certText, err := os.ReadFile(filepath.Join(chainDir, "cert.pem"))
	if err != nil {
		t.Fatal(err)
	}
	keyText, err := os.ReadFile(filepath.Join(chainDir, "key.pem"))
	if err != nil {
		t.Fatal(err)
	}
	quote := func(s string) string { data, _ := json.Marshal(s); return string(data) }
	harness := fmt.Sprintf(`import {strict as assert} from "node:assert";
import {%s as boot} from %s;
import {$canInitialize, $canServer} from %s;
import {runOwnedRoot} from %s;
import {success} from %s;
$canInitialize();
const cert=%s, key=%s;
const owned=await runOwnedRoot(async ()=>{
 const started=await boot(18372n,cert,key);
 assert.equal(started.kind,"ok");
 const token=started.value;
 const res=await fetch("https://127.0.0.1:18372/health",{tls:{ca:cert}});
 assert.equal(res.status,200);assert.equal(await res.text(),"healthy");
 await assert.rejects(fetch("https://127.0.0.1:18372/health"));
 assert.equal((await $canServer.stop(token)).kind,"ok");
 assert.equal((await $canServer.wait(token)).kind,"ok");
 return success(undefined);
});
assert.equal(owned.completion.kind,"ok");assert.equal(owned.cleanupFailed,false);
console.log("compiled TLS lifecycle passed");
`, bootName, quote(module), quote(filepath.Join(build.Directory, "program/state.ts")), quote(filepath.Join(runtimes[0], "owner.ts")), quote(filepath.Join(runtimes[0], "completion.ts")), quote(string(certText)), quote(string(keyText)))

	harnessPath := filepath.Join(t.TempDir(), "tls.ts")
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
		t.Fatalf("compiled TLS lifecycle: %v %s", err, result)
	}
}
