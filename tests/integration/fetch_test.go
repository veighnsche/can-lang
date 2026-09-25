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

// restageFetchHeaders rewrites staged request headers for the
// header-default mutation: every request gains the inherited x-default,
// connection content-type applies except where an op removes it (body
// native defaults then apply), and only the load op drops x-omitted. The
// staged authorization pair is preserved: staged runs read the credential
// from each fixture environment, not the test process.
func restageFetchHeaders(t *testing.T, root string) {
	t.Helper()
	sets := map[string][][]any{
		"load_json.json":     {{"content-type", "text/plain"}, {"x-default", "inherited"}, {"x-probe", "yes"}},
		"load_envelope.json": {{"content-type", "text/plain"}, {"x-default", "inherited"}, {"x-omitted", "remove-me"}},
		"send_text.json":     {{"content-type", "text/plain; charset=utf-8"}, {"x-default", "inherited"}, {"x-omitted", "remove-me"}},
		"send_json.json":     {{"content-type", "application/json"}, {"x-default", "inherited"}, {"x-omitted", "remove-me"}},
		"send_bytes.json":    {{"content-type", "application/octet-stream"}, {"x-default", "inherited"}, {"x-omitted", "remove-me"}},
		"head_text.json":     {{"content-type", "text/plain"}, {"x-default", "inherited"}, {"x-omitted", "remove-me"}},
		"delete_text.json":   {{"content-type", "text/plain"}, {"x-default", "inherited"}, {"x-omitted", "remove-me"}},
		"options_text.json":  {{"content-type", "text/plain"}, {"x-default", "inherited"}, {"x-omitted", "remove-me"}},
	}
	for name, headers := range sets {
		path := filepath.Join(root, "src/fixtures", name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var fixture map[string]any
		if err := json.Unmarshal(data, &fixture); err != nil {
			t.Fatal(err)
		}
		exchange, ok := fixture["exchange"].(map[string]any)
		if !ok {
			t.Fatalf("fixture %s omits exchange", name)
		}
		request, ok := exchange["request"].(map[string]any)
		if !ok {
			t.Fatalf("fixture %s omits request", name)
		}
		var auth any
		if staged, ok := request["headers"].([]any); ok {
			for _, pair := range staged {
				cells, ok := pair.([]any)
				if ok && len(cells) == 2 && cells[0] == "authorization" {
					auth = pair
				}
			}
		}
		if auth == nil {
			t.Fatalf("fixture %s omits staged authorization", name)
		}
		pairs := []any{auth}
		for _, pair := range headers {
			pairs = append(pairs, pair)
		}
		request["headers"] = pairs
		rewritten, err := json.Marshal(fixture)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, rewritten, 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCurrentBundledFetch(t *testing.T) {
	t.Parallel()
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for named fetch execution")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	bundle, err := harnessBundle(t, ctx, archive)
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	requests := 0
	mode := "normal"
	credential := "test-only"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		requests++
		if r.Header.Get("Authorization") != "Bearer test-only" {
			t.Error("missing named-fetch credential")
		}
		if r.Header.Get("X-Default") != "inherited" {
			t.Error("missing normalized connection default")
		}
		body, _ := io.ReadAll(r.Body)
		switch r.URL.Path {
		case "/json":
			w.Header().Set("Content-Type", "application/json")
			if r.Method == "GET" {
				if r.URL.Query().Get("labels") != "a b" || len(r.URL.Query()["labels"]) != 2 || r.URL.Query()["labels"][1] != "+" || r.Header.Get("X-Probe") != "yes" || r.URL.Query().Has("omitted") || len(r.Header.Values("X-Omitted")) != 0 {
					t.Error("query/header mismatch")
				}
			}
			if r.Method == "PUT" && (string(body) != `{"count":9007199254740993}` || r.Header.Get("Content-Type") != "application/json") {
				t.Error("JSON request encoding mismatch")
			}
			if mode == "status" {
				w.WriteHeader(409)
			}
			if mode == "media" {
				w.Header().Set("Content-Type", "text/plain")
			}
			if mode == "charset" {
				w.Header().Set("Content-Type", "application/json; charset=latin1")
			}
			if mode == "malformed" {
				io.WriteString(w, `{"count":"wrong"}`)
			} else {
				io.WriteString(w, `{"count":9007199254740993}`)
			}
		case "/envelope":
			w.Header().Add("Set-Cookie", "a=1")
			w.Header().Add("Set-Cookie", "b=2")
			w.WriteHeader(418)
			w.Write([]byte{0, 255})
		case "/text":
			if r.Method != "POST" || string(body) != "hello" || r.Header.Get("Content-Type") != "text/plain; charset=utf-8" {
				t.Error("text request mismatch")
			}
			w.Write(body)
		case "/bytes":
			if r.Method != "PATCH" || len(body) != 0 || r.Header.Get("Content-Type") != "application/octet-stream" {
				t.Error("bytes request mismatch")
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(body)
		case "/head":
			if r.Method != "HEAD" {
				t.Error("HEAD mismatch")
			}
			w.Header().Set("Content-Type", "text/plain")
		case "/delete":
			if r.Method != "DELETE" {
				t.Error("DELETE mismatch")
			}
			io.WriteString(w, "delete")
		case "/options":
			if r.Method != "OPTIONS" {
				t.Error("OPTIONS mismatch")
			}
			io.WriteString(w, "options")
		default:
			t.Error("unexpected fetch path")
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
		profile := "(version 1)(allow default)(deny network*)"
		if command == "run" {
			profile += `(allow network-outbound (remote ip "localhost:*"))`
		}
		argv := []string{"-p", profile, filepath.Join(bundle, "bin/canlc"), command, root}
		argv = append(argv, args...)
		cmd := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", argv...)
		cmd.Dir = t.TempDir()
		cmd.Env = []string{"PATH=/nonexistent", "HOME=" + cmd.Dir, "CAN_I26_TOKEN=" + credential}
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

	// 20 rows: 8 native fetch roots plus the null-exchange pretransport row,
	// 8 selective-recovery consumer rows, and the extract, main and referenced
	// ordinary roots.
	for _, fixture := range []struct {
		name string
		rows int
	}{{"main", 20}} {
		data, err := os.ReadFile(filepath.Join(sourceRoot, "compiler/testdata/current/fetch", fixture.name+".can"))
		if err != nil {
			t.Fatal(err)
		}
		pristine := strings.Replace(string(data), "http://127.0.0.1:1/", server.URL+"/", 1)
		source := pristine
		source = strings.Replace(source, "    max_body_bytes 8192", "    max_body_bytes 8192\n    headers\n        content_type = \"text/plain\"\n        x_default = \"inherited\"\n        x_omitted = \"remove-me\"", 1)
		// Empty overrides remove defaults before body-specific native defaults apply.
		source = strings.Replace(source, "    post \"/text\"", "    post \"/text\"\n    headers\n        content_type = []", 1)
		source = strings.Replace(source, "    patch \"/bytes\"", "    patch \"/bytes\"\n    headers\n        content_type = []", 1)
		stageRawFixtures(t, write, sourceRoot, "fetch", [2]string{"http://127.0.0.1:1", server.URL})
		// Assertions run against the pristine source: exact request comparison
		// must see the same headers the committed fixtures record. Header
		// default/override behavior is a run-path concern below.
		write("src/main.can", pristine)
		status, out, diag := run("assert")
		if status != 0 || diag != "" {
			t.Fatalf("fetch assertions: %d %s %s", status, out, diag)
		}
		var report map[string]any
		if err = json.Unmarshal([]byte(out), &report); err != nil || report["passed"] != true || len(report["assertions"].([]any)) != fixture.rows {
			t.Fatalf("invalid fetch report %v %s", err, out)
		}
		// Run verifies first, so the header-default mutation must restage
		// matching fixtures: connection defaults apply to every request,
		// op-level removals apply before body native defaults, and the
		// load op removes the inherited x-omitted header.
		restageFetchHeaders(t, root)
		write("src/main.can", source)
		status, out, diag = run("run")
		if status != 0 || out != "" || diag != "" {
			t.Fatalf("fetch execution: %d %s %s", status, out, diag)
		}
		mu.Lock()
		count := requests
		mu.Unlock()
		if count != 8 {
			t.Fatalf("expected 8 runtime requests and no assertion requests, got %d", count)
		}
		for _, bad := range []struct{ mode, want, reason string }{{"status", `"error":"http::request_failed"`, ""}, {"media", `"error":"http::request_failed"`, "media_type"}, {"charset", `"error":"http::request_failed"`, "charset"}, {"malformed", `"error":"http::request_failed"`, "type"}} {
			mu.Lock()
			mode = bad.mode
			mu.Unlock()
			status, _, diag = run("run")
			if status == 0 || !strings.Contains(diag, bad.want) {
				t.Fatalf("fetch error %s: %d %s", bad.mode, status, diag)
			}
		}
		mu.Lock()
		before := requests
		mu.Unlock()
		credential = ""
		invalid := source
		// The computed value reaches preparation validation; no credential
		// is available. Verification runs the real preparation, which
		// rejects the header (outcome mismatch, not a skipped comparison),
		// so the gate fails before any launch or publication.
		invalid = strings.Replace(invalid, `x_probe = call [["yes"]].map(callable extract)`, `x_probe = "bad" + "Ā"`, 1)
		write("src/main.can", invalid)
		status, _, diag = run("run")
		if status == 0 || !strings.Contains(diag, "build verification failed") || !strings.Contains(diag, "outcome mismatch") {
			t.Fatalf("header validation must precede credentials: %d %s", status, diag)
		}
		mu.Lock()
		after := requests
		mu.Unlock()
		if after != before {
			t.Fatal("invalid header launched a request")
		}
		credential = "test-only"
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
}
