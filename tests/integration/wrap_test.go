package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestCurrentBundledWrappers(t *testing.T) {
	t.Parallel()
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged wrapper execution")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	bundle, err := harnessBundle(t, ctx, archive)
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	mode := "missing"
	loads, batches := 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch r.URL.Path {
		case "/load":
			loads++
			w.Header().Set("Content-Type", "application/json")
			switch mode {
			case "busy":
				w.WriteHeader(429)
			case "broken":
				w.WriteHeader(503)
			case "ok":
				w.WriteHeader(200)
			default:
				w.WriteHeader(404)
			}
			io.WriteString(w, `{"count":7}`)
		case "/systemone":
			batches++
			data, _ := io.ReadAll(r.Body)
			if r.Method != "POST" || r.Header.Get("Authorization") == "" || r.Header.Get("Content-Type") != "application/json" || !strings.Contains(string(data), `"model":"jev-latest"`) {
				t.Error("incorrect judge protocol request")
			}
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, `{"model":"resolved-model","answers":{"q0":{"type":"noul","noul":0.5},"q1":{"type":"noul","noul":0.25},"q2":{"type":"noul","noul":1}}}`)
		default:
			t.Error("unexpected wrapper path")
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
		if err = os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(p, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("can.project.json", `{"source_root":"src","error_registry":"can.errors.json"}`)
	write("can.errors.json", `{"active":[],"retired":[]}`)
	run := func(command string, args ...string) (int, string, string) {
		t.Helper()
		profile := `(version 1)(allow default)(deny network*)(allow network-outbound (remote ip "localhost:*"))`
		argv := []string{"-p", profile, filepath.Join(bundle, "bin/canlc"), command, root}
		argv = append(argv, args...)
		cmd := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", argv...)
		cmd.Dir = t.TempDir()
		cmd.Env = []string{"PATH=/nonexistent", "HOME=" + cmd.Dir, "CAN_I17_TOKEN=test-only"}
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

	data, err := os.ReadFile(filepath.Join(sourceRoot, "compiler/testdata/current/wrap", "main.can"))
	if err != nil {
		t.Fatal(err)
	}
	source := strings.Replace(string(data), "http://127.0.0.1:1/", server.URL+"/", 1)
	source = strings.Replace(source, "http://127.0.0.1:1/systemone", server.URL+"/systemone", 1)
	stageRawFixtures(t, write, sourceRoot, "wrap", [2]string{"http://127.0.0.1:1", server.URL})
	write("src/main.can", source)

	// 13 rows: load decoded; cached absent/busy; settled gone/slow/stalled;
	// report direct; assess sample; guarded absorbed/missing; probe,
	// via_reference and main samples. Injection and raw rows run with no
	// live transport.
	status, out, diag := run("assert")
	if status != 0 || diag != "" {
		t.Fatalf("wrapper assertions: %d %s %s", status, out, diag)
	}
	var report map[string]any
	if err = json.Unmarshal([]byte(out), &report); err != nil || report["passed"] != true || len(report["assertions"].([]any)) != 13 {
		t.Fatalf("invalid wrapper report %v %s", err, out)
	}
	for _, label := range []string{"policy-fixture", "raw-provider-fixture", "real-can", "supplied-completion"} {
		if !strings.Contains(out, label) {
			t.Fatalf("wrapper report omits %s evidence: %s", label, out)
		}
	}
	if strings.Contains(out, "live-quality") {
		t.Fatalf("offline wrapper run claims live quality: %s", out)
	}
	mu.Lock()
	if loads != 0 || batches != 0 {
		t.Fatalf("assertions issued live requests: load=%d judge=%d", loads, batches)
	}
	mu.Unlock()

	// A 404 recovers through the grandchild while the judge batch succeeds;
	// each underlying operation runs exactly once.
	status, out, diag = run("run")
	if status != 0 || out != "TFT" || diag != "" {
		t.Fatalf("wrapper execution: %d %q %s", status, out, diag)
	}
	mu.Lock()
	if loads != 1 || batches != 1 {
		t.Fatalf("expected one load and one judge request, got %d and %d", loads, batches)
	}
	mu.Unlock()

	// 429 delegates grandchild -> child -> default; 503 renormalizes at the
	// grandchild. Both escape as one normalized infrastructure failure.
	for _, bad := range []struct{ mode, identity string }{{"busy", "can.std.http@1::request_failed"}, {"broken", "can.std.http@1::request_failed"}} {
		mu.Lock()
		mode = bad.mode
		mu.Unlock()
		status, _, diag = run("run")
		if status == 0 || !strings.Contains(diag, "can.error.v2:"+bad.identity) {
			t.Fatalf("wrapper error %s: %d %s", bad.mode, status, diag)
		}
	}
	// A successful load bypasses policy untouched: count 7 reaches the
	// receipt assertion instead of any handler.
	mu.Lock()
	mode = "ok"
	mu.Unlock()
	status, _, diag = run("run")
	if status == 0 || !strings.Contains(diag, "checks::failed") || !strings.Contains(diag, `"identity":"can.error.v2:can.std.checks@1::failed"`) {
		t.Fatalf("success bypass did not reach checks: %d %s", status, diag)
	}

	// Dropping the row that selects the emitted local key fails the build.
	mu.Lock()
	mode = "missing"
	mu.Unlock()
	dropped := strings.Replace(source, "        absorbed: (9007199254740993, -0.0) => ok 0.0\n            using failure emitted ai::invalid_answer(\"q2\", \"out of range\")\n", "", 1)
	if dropped == source {
		t.Fatal("coverage mutation did not match")
	}
	write("src/main.can", dropped)
	status, _, diag = run("assert")
	if status == 0 || !strings.Contains(diag, "selecting local") {
		t.Fatalf("missing local-key coverage passed: %d %s", status, diag)
	}
	write("src/main.can", source)
	if compiler := os.Getenv("CAN_TSC"); compiler != "" {
		nodeModules := filepath.Dir(filepath.Dir(filepath.Dir(compiler)))
		args := []string{compiler, "--noEmit", "--ignoreConfig", "--strict", "--skipLibCheck", "--target", "esnext", "--module", "esnext", "--moduleResolution", "bundler", "--allowImportingTsExtensions", "--typeRoots", filepath.Join(nodeModules, "@types"), "--types", "bun,node"}
		err := filepath.WalkDir(filepath.Join(root, "dist"), func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !entry.IsDir() && strings.HasSuffix(path, ".ts") {
				args = append(args, path)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		cmd := exec.CommandContext(ctx, os.Getenv("CAN_BUN"), args...)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("strict generated TypeScript: %v\n%s", err, output)
		}
	}
}
