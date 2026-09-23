package integration

import (
	"bufio"
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

func TestCurrentBundledAssets(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged asset execution")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "assets-integration")
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := os.ReadFile(filepath.Join(sourceRoot, "compiler/testdata/current/assets/page.can"))
	if err != nil {
		t.Fatal(err)
	}
	writeProject := func(root string, source string, assets map[string]string) {
		t.Helper()
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
		entries := make([]string, 0, len(assets))
		for name, relative := range assets {
			entries = append(entries, fmt.Sprintf("%q:%q", name, relative))
		}
		write("can.project.json", `{"source_root":"src","error_registry":"can.errors.json","assets":{`+strings.Join(entries, ",")+`}}`)
		write("can.errors.json", `{"active":[],"retired":[]}`)
		write("src/main.can", source)
		write("assets/site.css", "body{color:black}\n")
	}
	writeProject(root, string(fixture), map[string]string{"site_css": "assets/site.css"})
	runAt := func(root, command string, args ...string) (int, string, string) {
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
	status, out, diag := runAt(root, "assert")
	if status != 0 || diag != "" {
		t.Fatalf("asset assertions: %d %s %s", status, out, diag)
	}
	var report map[string]any
	if err = json.Unmarshal([]byte(out), &report); err != nil || report["passed"] != true || len(report["assertions"].([]any)) != 12 {
		t.Fatalf("invalid asset report %v %s", err, out)
	}
	if !strings.Contains(out, "supplied-completion") || !strings.Contains(out, "real-can") {
		t.Fatalf("asset report lacks scoped and real evidence: %s", out)
	}
	status, _, diag = runAt(root, "run")
	if status != 0 || diag != "" {
		t.Fatalf("asset execution: %d %s", status, diag)
	}
	buildAt := func(root string) (string, string) {
		t.Helper()
		status, out, diag := runAt(root, "build")
		if status != 0 {
			t.Fatalf("build %s: %d %s %s", root, status, out, diag)
		}
		var build struct {
			BuildID   string `json:"buildID"`
			Directory string `json:"directory"`
		}
		if err := json.Unmarshal([]byte(out), &build); err != nil {
			t.Fatal(err)
		}
		return build.BuildID, build.Directory
	}
	firstID, firstDir := buildAt(root)
	secondID, _ := buildAt(root)
	if firstID != secondID {
		t.Fatalf("rebuild changed build identity %s %s", firstID, secondID)
	}
	moved, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	writeProject(moved, string(fixture), map[string]string{"site_css": "assets/site.css"})
	movedID, movedDir := buildAt(moved)
	if movedID != firstID {
		t.Fatalf("relocated build changed identity %s %s", movedID, firstID)
	}
	compareTrees := func(a, b string) {
		t.Helper()
		collect := func(dir string) map[string]string {
			t.Helper()
			files := map[string]string{}
			err := filepath.WalkDir(filepath.Join(dir, "assets"), func(path string, entry os.DirEntry, err error) error {
				if err != nil || entry.IsDir() {
					return err
				}
				data, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				relative, err := filepath.Rel(dir, path)
				if err != nil {
					return err
				}
				files[relative] = string(data)
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(files) != 2 {
				t.Fatalf("expected pinned script plus one project asset, got %v", files)
			}
			return files
		}
		left, right := collect(a), collect(b)
		if len(left) != len(right) {
			t.Fatalf("asset manifests differ %v %v", left, right)
		}
		for path, data := range left {
			if right[path] != data {
				t.Fatalf("asset bytes differ at %s", path)
			}
		}
	}
	compareTrees(firstDir, movedDir)
	if tsc := os.Getenv("CAN_TSC"); tsc != "" {
		cmd := exec.CommandContext(ctx, filepath.Join(bundle, "runtime/bun"), tsc, "--noEmit", "--strict", "--skipLibCheck", "--target", "esnext", "--module", "esnext", "--moduleResolution", "bundler", "--allowImportingTsExtensions", "--typeRoots", filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(tsc))), "@types"), "--types", "bun,node", filepath.Join(firstDir, "entry.ts"))
		if result, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("strict generated TypeScript: %v %s", err, result)
		}
	}
	var module, bootName string
	err = filepath.WalkDir(firstDir, func(path string, entry os.DirEntry, err error) error {
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
			if strings.Contains(chunk[:line], `can.project.root/app::boot`) {
				module = path
				bootName = chunk[:first]
			}
		}
		return nil
	})
	if err != nil || module == "" || bootName == "" {
		t.Fatalf("missing generated boot function: %v", err)
	}
	runtimes, err := filepath.Glob(filepath.Join(firstDir, "runtime", "r-*"))
	if err != nil || len(runtimes) != 1 {
		t.Fatal("missing runtime")
	}
	quote := func(s string) string { data, _ := json.Marshal(s); return string(data) }
	harness := fmt.Sprintf(`import {strict as assert} from "node:assert";
import {%s as boot} from %s;
import {$canInitialize,$canServer} from %s;
import {runOwnedRoot} from %s;
import {success} from %s;
$canInitialize();
const owned=await runOwnedRoot(async ()=>{
 const started=await boot(18483n);assert.equal(started.kind,"ok");
 const token=started.value,base="http://127.0.0.1:18483";
 const page=await fetch(base+"/");assert.equal(page.status,200);
 const html=await page.text();
 assert.ok(html.includes("/__can/assets/htmx-4.0.0.min.js"));
 assert.ok(html.includes("sha384-BvJpBiO8Kh31EqtJe5DRIeWrHWnCGkwytKs9NKFi86Hhw96dEqdEMzZDeK9iEGTc"));
 assert.ok(html.includes("htmx-config"));
 const policy=page.headers.get("content-security-policy")??"";
 assert.ok(policy.includes("script-src 'self'")&&!policy.includes("unsafe-eval"));
 assert.equal(page.headers.get("x-content-type-options"),"nosniff");
 const css=html.match(/\/__can\/project\/[0-9a-f]{64}\/site\.css/)?.[0];
 assert.ok(css);
 const script=await fetch(base+"/__can/assets/htmx-4.0.0.min.js");
 assert.equal(script.status,200);
 assert.equal(script.headers.get("content-type"),"text/javascript");
 assert.equal(script.headers.get("cache-control"),"public, max-age=31536000, immutable");
 assert.equal(script.headers.get("x-content-type-options"),"nosniff");
 assert.ok((script.headers.get("content-security-policy")??"").includes("script-src 'self'"));
 const etag=script.headers.get("etag")??"";assert.match(etag,/^"[0-9a-f]{64}"$/);
 const pinned=new Uint8Array(await Bun.file(%s).bytes());
 assert.deepEqual(new Uint8Array(await script.arrayBuffer()),pinned);
 const cached=await fetch(base+"/__can/assets/htmx-4.0.0.min.js",{headers:{"if-none-match":etag}});
 assert.equal(cached.status,304);assert.equal(await cached.text(),"");
 const head=await fetch(base+"/__can/assets/htmx-4.0.0.min.js",{method:"HEAD"});
 assert.equal(head.status,200);assert.equal(await head.text(),"");
 const posted=await fetch(base+"/__can/assets/htmx-4.0.0.min.js",{method:"POST"});
 assert.equal(posted.status,405);assert.equal(posted.headers.get("allow"),"GET, HEAD");
 const style=await fetch(base+css);
 assert.equal(style.status,200);
 assert.equal(style.headers.get("content-type"),"text/css; charset=utf-8");
 assert.equal(await style.text(),"body{color:black}\n");
 for(const bad of ["/__can/nope","/__can/assets/","/__can/assets/htmx-4.0.0.min.js.map","/__can"])assert.equal((await fetch(base+bad)).status,404);
 const traversal=await fetch(base+"/__can/project/e484d9171a9db30a39c8f16e3d709d4137f3211c659f8e6125816635033d593f/../x");
 assert.equal(traversal.status,404);
 const blank=await fetch(base+"/validate",{method:"POST",headers:{"content-type":"application/x-www-form-urlencoded"},body:"name="});
 assert.equal(blank.status,422);assert.ok((await blank.text()).includes("Name is required."));
 const hostile="Ann</p><script>evil()</script>";
 const named=await fetch(base+"/validate",{method:"POST",headers:{"content-type":"application/x-www-form-urlencoded"},body:"name="+encodeURIComponent(hostile)});
 assert.equal(named.status,200);
 const fragment=await named.text();
 assert.ok(fragment.includes("Ann")&&!fragment.includes("<script>"));
 const quiet=await fetch(base+"/quiet");assert.equal(quiet.status,204);assert.equal(await quiet.text(),"");
 const boom=await fetch(base+"/boom");assert.equal(boom.status,500);
 const dash=await fetch(base+"/dashboard");assert.equal(dash.status,200);assert.ok((await dash.text()).length>0);
 assert.equal((await fetch(base+"/unknown")).status,404);
 const wrong=await fetch(base+"/",{method:"POST"});
 assert.equal(wrong.status,405);assert.equal(wrong.headers.get("allow"),"GET");
 const file=%s+"/assets/"+css.split("/")[3]+"/site.css";
 const original=await Bun.file(file).bytes();
 await Bun.write(file,"tampered\n");
 assert.equal((await fetch(base+css)).status,404);
 await Bun.write(file,original);
 assert.equal((await fetch(base+css)).status,200);
 assert.equal((await $canServer.stop(token)).kind,"ok");
 assert.equal((await $canServer.wait(token)).kind,"ok");
 return success(undefined);
});
assert.equal(owned.completion.kind,"ok");assert.equal(owned.cleanupFailed,false);
console.log("compiled asset loopback passed");
`, bootName, quote(module), quote(filepath.Join(firstDir, "program/state.ts")), quote(filepath.Join(runtimes[0], "owner.ts")), quote(filepath.Join(runtimes[0], "completion.ts")), quote(filepath.Join(sourceRoot, "distribution/assets/htmx-4.0.0.min.js")), quote(firstDir))
	harnessPath := filepath.Join(t.TempDir(), "assets.ts")
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
		t.Fatalf("compiled asset loopback: %v %s", err, result)
	}
	negatives := []struct {
		name   string
		source string
		assets map[string]string
		want   string
		extra  map[string]string
	}{
		{name: "script asset", source: string(fixture), assets: map[string]string{"site_css": "assets/site.css", "app": "assets/app.js"}, want: "rejected .js content", extra: map[string]string{"assets/app.js": "alert(1)\n"}},
		{name: "path escape", source: string(fixture), assets: map[string]string{"site_css": "assets/site.css", "evil": "../secret.css"}, want: "forbidden"},
		{name: "reserved route", source: strings.Replace(string(fixture), `call http::route_get("/quiet", callable quiet)`, `call http::route_get("/__can/evil", callable quiet)`, 1), assets: map[string]string{"site_css": "assets/site.css"}, want: "invalid static route path"},
	}
	for _, negative := range negatives {
		dir, err := filepath.EvalSymlinks(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		writeProject(dir, negative.source, negative.assets)
		for name, text := range negative.extra {
			if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0600); err != nil {
				t.Fatal(err)
			}
		}
		if negative.name == "path escape" {
			if err := os.WriteFile(filepath.Join(filepath.Dir(dir), "secret.css"), []byte("body{}\n"), 0600); err != nil {
				t.Fatal(err)
			}
		}
		status, out, diag := runAt(dir, "build")
		if status == 0 || !strings.Contains(out+diag, negative.want) {
			t.Fatalf("%s admitted: %d %s %s", negative.name, status, out, diag)
		}
	}
}

func TestCurrentBrowserAssets(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged browser execution")
	}
	nodePath, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is required for the browser harness")
	}
	sourceRoot, _ := filepath.Abs("../..")
	browserDir := filepath.Join(sourceRoot, "tests/integration/browser")
	if _, err := os.Stat(filepath.Join(browserDir, "node_modules/playwright/package.json")); err != nil {
		t.Skip("run bun ci in tests/integration/browser for the pinned harness")
	}
	probeCtx, probeCancel := context.WithTimeout(context.Background(), time.Minute)
	defer probeCancel()
	probe := exec.CommandContext(probeCtx, nodePath, "--input-type=module", "-e", `import("playwright").then(async ({chromium}) => { const browser = await chromium.launch(); console.log(browser.version()); await browser.close(); })`)
	probe.Dir = browserDir
	if result, err := probe.CombinedOutput(); err != nil {
		if strings.Contains(string(result), "Executable doesn't exist") {
			t.Skip("install the pinned playwright chromium for browser evidence")
		}
		t.Fatalf("browser probe: %v %s", err, result)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "browser-integration")
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
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	fixture, err := os.ReadFile(filepath.Join(sourceRoot, "compiler/testdata/current/assets/page.can"))
	if err != nil {
		t.Fatal(err)
	}
	write("can.project.json", `{"source_root":"src","error_registry":"can.errors.json","assets":{"site_css":"assets/site.css"}}`)
	write("can.errors.json", `{"active":[],"retired":[]}`)
	write("src/main.can", string(fixture))
	write("assets/site.css", "body{color:black}\n")
	buildCmd := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-p", "(version 1)(allow default)(deny network*)", filepath.Join(bundle, "bin/canlc"), "build", root)
	buildCmd.Dir = t.TempDir()
	buildCmd.Env = []string{"PATH=/nonexistent", "HOME=" + buildCmd.Dir}
	buildOut, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build: %v %s", err, buildOut)
	}
	var build struct {
		Directory string `json:"directory"`
	}
	if err := json.Unmarshal(buildOut, &build); err != nil {
		t.Fatal(err)
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
			if strings.Contains(chunk[:line], `can.project.root/app::boot`) {
				module = path
				bootName = chunk[:first]
			}
		}
		return nil
	})
	if err != nil || module == "" || bootName == "" {
		t.Fatalf("missing generated boot function: %v", err)
	}
	runtimes, err := filepath.Glob(filepath.Join(build.Directory, "runtime", "r-*"))
	if err != nil || len(runtimes) != 1 {
		t.Fatal("missing runtime")
	}
	quote := func(s string) string { data, _ := json.Marshal(s); return string(data) }
	serve := fmt.Sprintf(`import {strict as assert} from "node:assert";
import {%s as boot} from %s;
import {$canInitialize} from %s;
import {runOwnedRoot} from %s;
import {success} from %s;
$canInitialize();
await runOwnedRoot(async ()=>{
 const started=await boot(18484n);assert.equal(started.kind,"ok");
 console.log("ready 18484");
 await new Promise(()=>{});
 return success(undefined);
});
`, bootName, quote(module), quote(filepath.Join(build.Directory, "program/state.ts")), quote(filepath.Join(runtimes[0], "owner.ts")), quote(filepath.Join(runtimes[0], "completion.ts")))
	servePath := filepath.Join(t.TempDir(), "serve.ts")
	if err := os.WriteFile(servePath, []byte(serve), 0600); err != nil {
		t.Fatal(err)
	}
	environmentPath := filepath.Join(t.TempDir(), "environment.json")
	os.WriteFile(environmentPath, []byte("{}"), 0600)
	environment, err := os.Open(environmentPath)
	if err != nil {
		t.Fatal(err)
	}
	defer environment.Close()
	// The server binds only 127.0.0.1 through its Can configuration and is
	// killed after the browser run; unlike self-exiting harnesses it cannot
	// run beneath sandbox-exec without orphaning the child on kill.
	server := exec.CommandContext(ctx, filepath.Join(bundle, "runtime/bun"), "--no-install", "--no-env-file", servePath)
	server.Env = []string{"PATH=/nonexistent", "HOME=" + t.TempDir()}
	server.ExtraFiles = []*os.File{environment}
	stdout, err := server.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	server.Stderr = server.Stdout
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	ready := make(chan struct{})
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			if strings.HasPrefix(scanner.Text(), "ready ") {
				close(ready)
				return
			}
		}
	}()
	select {
	case <-ready:
	case <-time.After(time.Minute):
		server.Process.Kill()
		server.Wait()
		t.Fatal("staged server never became ready")
	}
	defer func() {
		server.Process.Kill()
		server.Wait()
	}()
	outdir := t.TempDir()
	browser := exec.CommandContext(ctx, nodePath, "assets.mjs", "http://127.0.0.1:18484", outdir)
	browser.Dir = browserDir
	browser.Env = []string{"PATH=" + filepath.Dir(nodePath) + ":/usr/bin:/bin", "HOME=" + os.Getenv("HOME")}
	result, err := browser.CombinedOutput()
	if err != nil {
		t.Fatalf("browser harness: %v %s", err, result)
	}
	var report struct {
		Browser string `json:"browser"`
		Version string `json:"version"`
		Passed  bool   `json:"passed"`
		Checks  []struct {
			Name   string `json:"name"`
			Passed bool   `json:"passed"`
		} `json:"checks"`
		Requests []struct {
			URL string `json:"url"`
		} `json:"requests"`
	}
	raw, err := os.ReadFile(filepath.Join(outdir, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &report); err != nil || !report.Passed || len(report.Checks) != 12 {
		t.Fatalf("invalid browser report %v %s", err, raw)
	}
	for _, entry := range report.Requests {
		if !strings.HasPrefix(entry.URL, "http://127.0.0.1:18484/") {
			t.Fatalf("browser left loopback: %s", entry.URL)
		}
	}
	shot, err := os.Stat(filepath.Join(outdir, "screenshot.png"))
	if err != nil || shot.Size() == 0 {
		t.Fatal("missing browser screenshot")
	}
	t.Logf("browser %s %s: %d checks, %d loopback requests", report.Browser, report.Version, len(report.Checks), len(report.Requests))
}
