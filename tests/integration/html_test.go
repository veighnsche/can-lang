package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/veighnsche/can-lang/distribution"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCurrentBundledHTML(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "html-integration")
	if err != nil {
		t.Fatal(err)
	}
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
	data, err := os.ReadFile(filepath.Join(sourceRoot, "compiler/testdata/current/html/main.can"))
	if err != nil {
		t.Fatal(err)
	}
	write("src/main.can", string(data))
	run := func(command string) (int, string, string) {
		t.Helper()
		cmd := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-p", "(version 1)(allow default)(deny network*)", filepath.Join(bundle, "bin/canlc"), command, root)
		cmd.Dir = t.TempDir()
		cmd.Env = []string{"PATH=/nonexistent", "HOME=" + cmd.Dir}
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
	if code, out, diag := run("assert"); code != 0 || diag != "" || !strings.Contains(out, "real-can") {
		t.Fatalf("real HTML assertions: %d %s %s", code, out, diag)
	}
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
		cmd := exec.CommandContext(ctx, filepath.Join(bundle, "runtime/bun"), tsc, "--noEmit", "--ignoreConfig", "--strict", "--skipLibCheck", "--target", "esnext", "--module", "esnext", "--moduleResolution", "bundler", "--allowImportingTsExtensions", "--typeRoots", filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(tsc))), "@types"), "--types", "bun,node", filepath.Join(build.Directory, "entry.ts"))
		if result, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("strict generated TypeScript: %v %s", err, result)
		}
	}
	var module, renderName, fragmentName string
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
			if first < 0 {
				continue
			}
			line := strings.Index(chunk, "try {")
			if line < 0 {
				continue
			}
			if strings.Contains(chunk[:line], `can.project.root/app::render`) {
				module = path
				renderName = chunk[:first]
			}
			if strings.Contains(chunk[:line], `can.project.root/app::fragment`) {
				fragmentName = chunk[:first]
			}
		}
		return nil
	})
	if err != nil || module == "" || renderName == "" || fragmentName == "" {
		t.Fatalf("missing generated HTML functions: %v", err)
	}
	runtimes, err := filepath.Glob(filepath.Join(build.Directory, "runtime", "r-*"))
	if err != nil || len(runtimes) != 1 {
		t.Fatal("missing runtime")
	}
	hostile := `</title><script>alert("request")</script><img src=x onerror=alert(1)>&'"`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, r.URL.Query().Get("message"))
	}))
	defer server.Close()
	quote := func(s string) string { data, _ := json.Marshal(s); return string(data) }
	harness := fmt.Sprintf(`import {strict as assert} from "node:assert";
import {%s as render,%s as fragment} from %s;
import {$canInitialize,$canHTML} from %s;
import {renderSafe} from %s;
$canInitialize();
const target=new URL(%s);target.searchParams.set("message",%s);
const input=await (await fetch(target)).text();assert.equal(input,%s);
const result=await render(input,"section");assert.equal(result.kind,"ok");
const output=renderSafe(result.value),escaped=Bun.escapeHTML(input);
assert.ok(output.startsWith("<!doctype html><html><head><title>"+escaped+"</title>"));
assert.ok(output.endsWith('<body><section id="results" hx-get="/results?q=a&amp;x=b" hx-target="#results" hx-trigger="every 1000ms"><p title="'+escaped+'">'+escaped+'</p></section></body></html>'));
assert.equal((output.match(/<script /g)||[]).length,1);assert.ok(!output.includes('<img'));assert.ok(!output.includes('<script>alert'));
assert.ok(output.includes('/__can/assets/htmx-4.0.0.min.js'));
const text=await $canHTML.text(input);assert.equal(text.kind,"ok");const part=await fragment([text.value]);assert.equal(part.kind,"ok");assert.equal(renderSafe(part.value),escaped);
console.log("compiled request-derived HTML passed");
`, renderName, fragmentName, quote(module), quote(filepath.Join(build.Directory, "program/state.ts")), quote(filepath.Join(runtimes[0], "platform/html.ts")), quote(server.URL), quote(hostile), quote(hostile))
	harnessPath := filepath.Join(t.TempDir(), "html.ts")
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
	cmd := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-p", `(version 1)(allow default)(deny network*)(allow network-outbound (remote ip "localhost:*"))`, filepath.Join(bundle, "runtime/bun"), "--no-install", "--no-env-file", harnessPath)
	cmd.Env = []string{"PATH=/nonexistent", "HOME=" + t.TempDir()}
	cmd.ExtraFiles = []*os.File{environment}
	if result, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compiled native HTML: %v %s", err, result)
	}
}
