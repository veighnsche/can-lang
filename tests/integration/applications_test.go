// Staged and live admission coverage for the four I42 example
// applications. Staged legs assert, build (twice, byte-identical IDs),
// and serve from release layouts without a database; live legs seed a
// disposable PostgreSQL 17 and drive every documented interaction over
// loopback HTTP. Raw-provider results come only from the local
// compatible stub below and stay labeled in the test log.
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/distribution"
)

func stageApplication(t *testing.T, ctx context.Context, bundle, sourceRoot, name string) (root, home string) {
	t.Helper()
	var err error
	root, err = filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(filepath.Join(sourceRoot, "examples", name))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() == "dist" {
			continue
		}
		src := filepath.Join(sourceRoot, "examples", name, entry.Name())
		dst := filepath.Join(root, entry.Name())
		if entry.IsDir() {
			copyDir(t, src, dst)
			continue
		}
		data, err := os.ReadFile(src)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dst, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	_ = bundle
	_ = ctx
	home, err = filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return root, home
}

func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dst, 0700); err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		from := filepath.Join(src, entry.Name())
		to := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			copyDir(t, from, to)
			continue
		}
		data, err := os.ReadFile(from)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(to, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func canlcOffline(t *testing.T, ctx context.Context, bundle, home, command, root string) (int, string, string) {
	t.Helper()
	cmd := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-p", "(version 1)(allow default)(deny network*)", filepath.Join(bundle, "bin/canlc"), command, root)
	cmd.Dir = home
	cmd.Env = []string{"PATH=/nonexistent", "HOME=" + home}
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

func applicationBuild(t *testing.T, ctx context.Context, bundle, home, root string) (string, string) {
	t.Helper()
	status, out, diag := canlcOffline(t, ctx, bundle, home, "build", root)
	if status != 0 {
		t.Fatalf("build: %d %s %s", status, out, diag)
	}
	var report struct {
		BuildID   string `json:"buildID"`
		Directory string `json:"directory"`
	}
	if err := json.Unmarshal([]byte(out), &report); err != nil || report.BuildID == "" || report.Directory == "" {
		t.Fatalf("invalid build report %v %s", err, out)
	}
	return report.BuildID, report.Directory
}

func snapshotCredential(t *testing.T, home, id, value string) string {
	t.Helper()
	return snapshotMap(t, home, "snapshot-"+id, map[string]string{id: value})
}

func snapshotMap(t *testing.T, home, name string, values map[string]string) string {
	t.Helper()
	data, err := json.Marshal(values)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, name+".json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

// serveApplication runs a staged entry.ts main with a port argument and
// one fd-3 credential snapshot, waits for a 200 on readyPath, and
// returns a stopper asserting clean SIGTERM shutdown.
func serveApplication(t *testing.T, ctx context.Context, bundle, home, entry string, snapshot string, port int, readyPath string, extraEnv ...string) (string, func()) {
	t.Helper()
	file, err := os.Open(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	cmd := exec.CommandContext(ctx, filepath.Join(bundle, "runtime/bun"), entry, strconv.Itoa(port))
	cmd.Dir = home
	cmd.Env = append([]string{"PATH=/nonexistent", "HOME=" + home}, extraEnv...)
	cmd.ExtraFiles = []*os.File{file}
	var out, diag bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &diag
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	base := "http://127.0.0.1:" + strconv.Itoa(port)
	deadline := time.Now().Add(30 * time.Second)
	for {
		if status, _, _ := httpGet(base + readyPath); status == 200 {
			break
		}
		if time.Now().After(deadline) {
			cmd.Process.Kill()
			t.Fatalf("server never ready on %s: stdout=%q stderr=%q", base, out.String(), diag.String())
		}
		time.Sleep(200 * time.Millisecond)
	}
	stopped := false
	return base, func() {
		t.Helper()
		if stopped {
			return
		}
		stopped = true
		if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
			t.Fatalf("signal: %v", err)
		}
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("unclean shutdown: %v stdout=%q stderr=%q", err, out.String(), diag.String())
			}
		case <-time.After(20 * time.Second):
			cmd.Process.Kill()
			t.Fatalf("shutdown hung: stdout=%q stderr=%q", out.String(), diag.String())
		}
	}
}

func httpGet(url string) (int, string, http.Header) {
	client := &http.Client{Timeout: 10 * time.Second}
	response, err := client.Get(url)
	if err != nil {
		return 0, "", nil
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	return response.StatusCode, string(body), response.Header
}

func httpPostForm(url string, values url.Values) (int, string) {
	client := &http.Client{Timeout: 10 * time.Second}
	response, err := client.PostForm(url, values)
	if err != nil {
		return 0, ""
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	return response.StatusCode, string(body)
}

func applicationSeed(t *testing.T, ctx context.Context, bundle, sourceRoot, home, url, mode string) map[string]any {
	t.Helper()
	seed := filepath.Join(sourceRoot, "tests/integration/testdata/applications/seed.sql")
	driver := filepath.Join(sourceRoot, "tests/integration/testdata/applications/seed-driver.ts")
	cmd := exec.CommandContext(ctx, filepath.Join(bundle, "runtime/bun"), driver, mode, seed)
	cmd.Dir = home
	cmd.Env = []string{"PATH=/nonexistent", "HOME=" + home, "CAN_TEST_POSTGRES_URL=" + url}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("seed %s: %v %s", mode, err, strings.ReplaceAll(string(out), url, "<redacted-url>"))
	}
	var report map[string]any
	if err := json.Unmarshal(out, &report); err != nil {
		t.Fatalf("invalid seed %s report %v %s", mode, err, string(out))
	}
	if failure, ok := report["error"]; ok {
		t.Fatalf("seed %s: %v", mode, failure)
	}
	return report
}

func TestApplicationsStaged(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged application execution")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "applications-staged")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"account-search", "form-validation", "dashboard", "native-ai"} {
		root, home := stageApplication(t, ctx, bundle, sourceRoot, name)
		status, out, diag := canlcOffline(t, ctx, bundle, home, "assert", root)
		if status != 0 || diag != "" {
			t.Fatalf("%s assert: %d %s %s", name, status, out, diag)
		}
		var report struct {
			Passed     bool `json:"passed"`
			Assertions []struct {
				Evidence []string `json:"evidence"`
			} `json:"assertions"`
		}
		if err := json.Unmarshal([]byte(out), &report); err != nil || !report.Passed || len(report.Assertions) == 0 {
			t.Fatalf("%s invalid assert report %v %s", name, err, out)
		}
		real := 0
		for _, assertion := range report.Assertions {
			for _, evidence := range assertion.Evidence {
				if evidence == "real-can" {
					real++
					break
				}
			}
		}
		if real == 0 {
			t.Fatalf("%s asserts nothing real", name)
		}
		firstID, firstDir := applicationBuild(t, ctx, bundle, home, root)
		secondID, _ := applicationBuild(t, ctx, bundle, home, root)
		if firstID != secondID {
			t.Fatalf("%s rebuild drifted: %s vs %s", name, firstID, secondID)
		}
		entry := filepath.Join(firstDir, "entry.ts")
		snapshot := snapshotCredential(t, home, "UNUSED", "postgres://127.0.0.1:1/nope")
		file, err := os.Open(snapshot)
		if err != nil {
			t.Fatal(err)
		}
		usage := exec.CommandContext(ctx, filepath.Join(bundle, "runtime/bun"), entry)
		usage.Dir = home
		usage.Env = []string{"PATH=/nonexistent", "HOME=" + home}
		usage.ExtraFiles = []*os.File{file}
		usageOut, err := usage.CombinedOutput()
		file.Close()
		if err != nil || !strings.HasPrefix(string(usageOut), "usage: ") {
			t.Fatalf("%s usage: %v %q", name, err, string(usageOut))
		}
		t.Logf("%s staged: %d assertions (%d real-can), build %s", name, len(report.Assertions), real, firstID[:12])
	}
}

func TestApplicationsStagedForms(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged application execution")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "applications-forms")
	if err != nil {
		t.Fatal(err)
	}
	root, home := stageApplication(t, ctx, bundle, sourceRoot, "form-validation")
	_, dir := applicationBuild(t, ctx, bundle, home, root)
	snapshot := snapshotCredential(t, home, "UNUSED", "postgres://127.0.0.1:1/nope")
	base, stop := serveApplication(t, ctx, bundle, home, filepath.Join(dir, "entry.ts"), snapshot, 18531, "/signup")
	defer stop()
	checks := 0
	get := func(path string, want int, contains ...string) http.Header {
		t.Helper()
		checks++
		status, body, header := httpGet(base + path)
		if status != want {
			t.Fatalf("GET %s: %d, want %d (%q)", path, status, want, body)
		}
		for _, snippet := range contains {
			if !strings.Contains(body, snippet) {
				t.Fatalf("GET %s missing %q in %q", path, snippet, body)
			}
		}
		return header
	}
	post := func(path string, values url.Values, want int, contains ...string) {
		t.Helper()
		checks++
		status, body := httpPostForm(base+path, values)
		if status != want {
			t.Fatalf("POST %s: %d, want %d (%q)", path, status, want, body)
		}
		for _, snippet := range contains {
			if !strings.Contains(body, snippet) {
				t.Fatalf("POST %s missing %q in %q", path, snippet, body)
			}
		}
	}
	header := get("/signup", 200, `id="signup_form"`, `hx-post="/signup/validate"`, `hx-indicator`, `hx-disable`, "/__can/assets/htmx-4.0.0.min.js", "/__can/project/")
	if policy := header.Get("Content-Security-Policy"); !strings.Contains(policy, "script-src 'self'") {
		t.Fatalf("signup CSP missing: %q", policy)
	}
	post("/signup/validate", url.Values{"name": {"Ann"}, "tags": {"a", "b"}}, 200, "Ann tags 2")
	post("/signup/validate", url.Values{"name": {""}}, 422, "Name is required.")
	post("/signup/validate", url.Values{"name": {"<script>window.__evil=1</script>"}}, 200, "&lt;script&gt;")
	if status, body := httpPostForm(base+"/signup/validate", url.Values{"name": {"<b>Ann</b>"}}); status != 200 || strings.Contains(body, "<b>Ann</b>") {
		t.Fatalf("hostile name unescaped: %d %q", status, body)
	} else {
		checks++
	}
	get("/missing", 404)
	if client := (&http.Client{Timeout: 10 * time.Second}); true {
		checks++
		request, _ := http.NewRequest("POST", base+"/signup", nil)
		response, err := client.Do(request)
		if err != nil || response.StatusCode != 405 {
			t.Fatalf("POST /signup: %v %v", err, response)
		}
		response.Body.Close()
	}
	t.Logf("staged forms: %d checks", checks)
}

// stubProviders serves the local compatible endpoints for native-ai:
// openai_responses_v1 plans plus one batched systemone v1 judgment.
// Every response is fixture data; nothing leaves loopback. The mode
// pointer selects the served payloads: "ok" for the valid plan and
// judgment, "bad-plan" for a completed response with no output, and
// "bad-answers" for a judgment batch with an out-of-range answer.
func stubProviders(t *testing.T) (*httptest.Server, *struct {
	sync.Mutex
	paths  []string
	bodies []string
}, *string) {
	t.Helper()
	seen := &struct {
		sync.Mutex
		paths  []string
		bodies []string
	}{}
	mode := "ok"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen.Lock()
		defer seen.Unlock()
		data, _ := io.ReadAll(r.Body)
		seen.paths = append(seen.paths, r.URL.Path)
		seen.bodies = append(seen.bodies, string(data))
		if r.Method != "POST" || r.Header.Get("Authorization") != "Bearer test-only" || r.Header.Get("Content-Type") != "application/json" {
			t.Error("invalid provider request headers")
		}
		var request map[string]any
		if err := json.Unmarshal(data, &request); err != nil {
			t.Error(err)
		}
		var response string
		switch r.URL.Path {
		case "/responses":
			output := `{"keywords":["vip","new"],"ready":true}`
			if request["instructions"] != "Draft" {
				t.Errorf("unexpected instructions %v", request["instructions"])
			}
			encoded, _ := json.Marshal(output)
			response = `{"id":"response-1","object":"response","status":"completed","output":[{"type":"message","role":"assistant","status":"completed","content":[{"type":"output_text","text":` + string(encoded) + `}]}]}`
			if mode == "bad-plan" {
				response = `{"id":"r","object":"response","status":"completed","output":[]}`
			}
		case "/systemone":
			response = `{"model":"stub","answers":{"q0":{"type":"choice","choice":"vip","confidence":0.9,"probabilities":{"vip":0.9,"new":0.1}},"q1":{"type":"noul","noul":0.8},"q2":{"type":"score","score":1,"confidence":0.5,"probabilities":{"0":0.25,"1":0.5,"2":0.25},"legend":{"0":"Low","1":"Medium","2":"High"}}}}`
			if mode == "bad-answers" {
				response = `{"model":"stub","answers":{"q0":{"type":"choice","choice":"vip","confidence":0.9,"probabilities":{"vip":0.9,"new":0.1}},"q1":{"type":"noul","noul":1.5},"q2":{"type":"score","score":1,"confidence":0.5,"probabilities":{"0":0.25,"1":0.5,"2":0.25},"legend":{"0":"Low","1":"Medium","2":"High"}}}}`
			}
		default:
			t.Errorf("unexpected provider path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, response)
	}))
	return server, seen, &mode
}

func rewriteEndpoints(t *testing.T, root, file, from, to string) {
	t.Helper()
	path := filepath.Join(root, file)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), from) {
		t.Fatalf("%s has no %s endpoint", file, from)
	}
	if err := os.WriteFile(path, []byte(strings.ReplaceAll(string(data), from, to)), 0600); err != nil {
		t.Fatal(err)
	}
}

func runEntry(t *testing.T, ctx context.Context, bundle, home, entry, snapshot string, extraEnv []string, args ...string) (int, string, string) {
	t.Helper()
	file, err := os.Open(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	cmd := exec.CommandContext(ctx, filepath.Join(bundle, "runtime/bun"), append([]string{entry}, args...)...)
	cmd.Dir = home
	cmd.Env = append([]string{"PATH=/nonexistent", "HOME=" + home}, extraEnv...)
	cmd.ExtraFiles = []*os.File{file}
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

func TestApplicationsNativeAIStubbed(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged application execution")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "applications-native-stubbed")
	if err != nil {
		t.Fatal(err)
	}
	server, seen, mode := stubProviders(t)
	defer server.Close()
	root, home := stageApplication(t, ctx, bundle, sourceRoot, "native-ai")
	rewriteEndpoints(t, root, "src/oracles/oracles.can", "http://127.0.0.1:1", server.URL)
	_, dir := applicationBuild(t, ctx, bundle, home, root)
	entry := filepath.Join(dir, "entry.ts")
	refused := snapshotMap(t, home, "triage-refused", map[string]string{"TRIAGE_DB": "postgres://127.0.0.1:1/nope", "CAN_I28_TOKEN": "test-only"})
	status, out, diag := runEntry(t, ctx, bundle, home, entry, refused, nil, "Ann")
	if status != 1 || out != "" || !strings.Contains(diag, `"error":"sql::connection_failed"`) {
		t.Fatalf("stubbed refused run: %d stdout=%q stderr=%s", status, out, diag)
	}
	seen.Lock()
	if len(seen.paths) != 2 || seen.paths[0] != "/responses" || seen.paths[1] != "/systemone" {
		paths := append([]string(nil), seen.paths...)
		seen.Unlock()
		t.Fatalf("raw-provider calls: %v", paths)
	}
	bodies := append([]string(nil), seen.bodies...)
	seen.Unlock()
	for _, want := range []string{`"instructions":"Draft"`, `"instructions":"Pick"`, `"instructions":"Likely"`, `"instructions":"Rate"`} {
		found := false
		for _, body := range bodies {
			if strings.Contains(body, want) {
				found = true
			}
		}
		if !found {
			t.Fatalf("stub never saw %s in %q", want, bodies)
		}
	}
	seen.Lock()
	*mode = "bad-plan"
	seen.Unlock()
	status, out, diag = runEntry(t, ctx, bundle, home, entry, refused, nil, "Ann")
	if status != 1 || out != "" || !strings.Contains(diag, `"error":"llm::invalid_response"`) {
		t.Fatalf("stubbed bad plan: %d stdout=%q stderr=%s", status, out, diag)
	}
	seen.Lock()
	if len(seen.paths) != 3 || seen.paths[2] != "/responses" {
		paths := append([]string(nil), seen.paths...)
		seen.Unlock()
		t.Fatalf("bad plan advanced past draft: %v", paths)
	}
	*mode = "bad-answers"
	seen.Unlock()
	status, out, diag = runEntry(t, ctx, bundle, home, entry, refused, nil, "Ann")
	if status != 1 || out != "" || !strings.Contains(diag, `"error":"ai::invalid_answer"`) {
		t.Fatalf("stubbed bad answers: %d stdout=%q stderr=%s", status, out, diag)
	}
	seen.Lock()
	paths := append([]string(nil), seen.paths...)
	seen.Unlock()
	if len(paths) != 5 || paths[3] != "/responses" || paths[4] != "/systemone" {
		t.Fatalf("bad answers mis-sequenced: %v", paths)
	}
	t.Logf("stubbed native-ai: plan plus one batched judgment; refusal, invalid plan, and invalid answers surface typed failures")
}

func TestApplicationsLive(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged application execution")
	}
	dbURL := os.Getenv("CAN_TEST_POSTGRES_URL")
	if dbURL == "" {
		t.Skip("set CAN_TEST_POSTGRES_URL to a disposable PostgreSQL for live application checks")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "applications-live")
	if err != nil {
		t.Fatal(err)
	}
	seedHome, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	setup := applicationSeed(t, ctx, bundle, sourceRoot, seedHome, dbURL, "setup")
	version, _ := setup["serverVersion"].(string)
	if setup["accounts"] != 7.0 || setup["dashboard"] != 3.0 || setup["triage"] != 2.0 {
		t.Fatalf("unexpected live seed %v", setup)
	}
	checks := 0

	t.Run("accounts", func(t *testing.T) {
		root, home := stageApplication(t, ctx, bundle, sourceRoot, "account-search")
		_, dir := applicationBuild(t, ctx, bundle, home, root)
		live := snapshotCredential(t, home, "ACCOUNTS_DB", dbURL)
		base, stop := serveApplication(t, ctx, bundle, home, filepath.Join(dir, "entry.ts"), live, 18532, "/accounts")
		defer stop()
		status, body, _ := httpGet(base + "/accounts")
		checks++
		if status != 200 || !strings.Contains(body, `hx-get="/accounts/search"`) || !strings.Contains(body, `hx-trigger="change"`) {
			t.Fatalf("accounts page: %d %q", status, body)
		}
		status, body, _ = httpGet(base + "/accounts/search?query=Ann")
		checks++
		if status != 200 || !strings.Contains(body, "Ann") || !strings.Contains(body, "<li>") {
			t.Fatalf("search Ann: %d %q", status, body)
		}
		status, body, _ = httpGet(base + "/accounts/search?query=%3Cem%3E")
		checks++
		if status != 200 || !strings.Contains(body, "&lt;em&gt;") || strings.Contains(body, "<em>Hax</em>") {
			t.Fatalf("search hostile: %d %q", status, body)
		}
		for _, probe := range []string{"/accounts/search?query=", "/accounts/search", "/accounts/search?query=a&query=b"} {
			status, body, _ = httpGet(base + probe)
			checks++
			if status != 422 || !strings.Contains(body, "Enter a search term.") {
				t.Fatalf("search %s: %d %q", probe, status, body)
			}
		}
		status, body = httpPostForm(base+"/accounts/validate", url.Values{"name": {"Ann"}})
		checks++
		if status != 200 || !strings.Contains(body, "Ann") {
			t.Fatalf("validate Ann: %d %q", status, body)
		}
		status, body = httpPostForm(base+"/accounts/validate", url.Values{"name": {""}})
		checks++
		if status != 422 || !strings.Contains(body, "Name is required.") {
			t.Fatalf("validate blank: %d %q", status, body)
		}
		status, body, _ = httpGet(base + "/dashboard/summary")
		checks++
		if status != 200 {
			t.Fatalf("dashboard summary: %d %q", status, body)
		}
		status, _, _ = httpGet(base + "/missing")
		checks++
		if status != 404 {
			t.Fatalf("unknown route: %d", status)
		}
		client := &http.Client{Timeout: 10 * time.Second}
		request, _ := http.NewRequest("POST", base+"/accounts/search?query=Ann", nil)
		response, err := client.Do(request)
		checks++
		if err != nil || response.StatusCode != 405 {
			t.Fatalf("wrong method: %v %v", err, response)
		}
		response.Body.Close()
		applicationSeed(t, ctx, bundle, sourceRoot, seedHome, dbURL, "teardown")
		status, body, _ = httpGet(base + "/accounts/search?query=Ann")
		checks++
		if status != 503 || !strings.Contains(body, "Temporarily unavailable.") {
			t.Fatalf("search dropped table: %d %q", status, body)
		}
		setup := applicationSeed(t, ctx, bundle, sourceRoot, seedHome, dbURL, "setup")
		if setup["accounts"] != 7.0 {
			t.Fatalf("reseed failed: %v", setup)
		}
	})

	t.Run("dashboard", func(t *testing.T) {
		root, home := stageApplication(t, ctx, bundle, sourceRoot, "dashboard")
		_, dir := applicationBuild(t, ctx, bundle, home, root)
		live := snapshotCredential(t, home, "DASHBOARD_DB", dbURL)
		base, stop := serveApplication(t, ctx, bundle, home, filepath.Join(dir, "entry.ts"), live, 18533, "/dashboard")
		defer stop()
		status, body, _ := httpGet(base + "/dashboard")
		checks++
		if status != 200 || !strings.Contains(body, `hx-get="/dashboard/data"`) || !strings.Contains(body, `hx-trigger="every 5000ms"`) {
			t.Fatalf("dashboard page: %d %q", status, body)
		}
		status, body, _ = httpGet(base + "/dashboard/data")
		checks++
		if status != 200 || !strings.Contains(body, "3 accounts 3 recent") {
			t.Fatalf("dashboard data: %d %q", status, body)
		}
		applicationSeed(t, ctx, bundle, sourceRoot, seedHome, dbURL, "teardown")
		status, body, _ = httpGet(base + "/dashboard/data")
		checks++
		if status != 503 || !strings.Contains(body, "Temporarily unavailable.") {
			t.Fatalf("dashboard dropped table: %d %q", status, body)
		}
		setup := applicationSeed(t, ctx, bundle, sourceRoot, seedHome, dbURL, "setup")
		if setup["dashboard"] != 3.0 {
			t.Fatalf("reseed failed: %v", setup)
		}
	})

	t.Run("triage", func(t *testing.T) {
		server, seen, _ := stubProviders(t)
		defer server.Close()
		root, home := stageApplication(t, ctx, bundle, sourceRoot, "native-ai")
		rewriteEndpoints(t, root, "src/oracles/oracles.can", "http://127.0.0.1:1", server.URL)
		_, dir := applicationBuild(t, ctx, bundle, home, root)
		live := snapshotMap(t, home, "triage-live", map[string]string{"TRIAGE_DB": dbURL, "CAN_I28_TOKEN": "test-only"})
		status, out, diag := runEntry(t, ctx, bundle, home, filepath.Join(dir, "entry.ts"), live, nil, "Ann")
		checks++
		if status != 0 || diag != "" {
			t.Fatalf("triage run: %d stdout=%q stderr=%s", status, out, diag)
		}
		var report struct {
			Keyword string  `json:"keyword"`
			Level   string  `json:"level"`
			Score   float64 `json:"score"`
			Total   int     `json:"total"`
			Recent  int     `json:"recent"`
		}
		if err := json.Unmarshal([]byte(out), &report); err != nil {
			t.Fatalf("triage report: %v %q", err, out)
		}
		if report.Keyword != "vip" || report.Level != "ship" || report.Score != 0.9 || report.Total != 2 || report.Recent != 2 {
			t.Fatalf("triage report mismatch: %+v", report)
		}
		seen.Lock()
		paths := append([]string(nil), seen.paths...)
		seen.Unlock()
		if len(paths) != 2 {
			t.Fatalf("raw-provider calls: %v", paths)
		}
		applicationSeed(t, ctx, bundle, sourceRoot, seedHome, dbURL, "teardown")
		status, out, diag = runEntry(t, ctx, bundle, home, filepath.Join(dir, "entry.ts"), live, nil, "Ann")
		checks++
		if status != 0 || out != "unavailable" || diag != "" {
			t.Fatalf("triage dropped table: %d stdout=%q stderr=%s", status, out, diag)
		}
		setup := applicationSeed(t, ctx, bundle, sourceRoot, seedHome, dbURL, "setup")
		if setup["triage"] != 2.0 {
			t.Fatalf("reseed failed: %v", setup)
		}
	})

	applicationSeed(t, ctx, bundle, sourceRoot, seedHome, dbURL, "teardown")
	t.Logf("live applications: %d checks against %s", checks, strings.SplitN(version, " on ", 2)[0])
}

func copyEvidenceFile(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, data, 0600); err != nil {
		t.Fatal(err)
	}
}

// TestApplicationsBrowser drives the three web applications through the
// pinned Playwright harness: staged forms without a database, then live
// accounts and dashboard against disposable PostgreSQL. Each scenario
// proves its documented HTMX interaction in a real browser with only the
// packaged HTMX script executing and every request staying on loopback.
func TestApplicationsBrowser(t *testing.T) {
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
		t.Skip("run npm install in tests/integration/browser for the pinned harness")
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
	ctx, cancel := context.WithTimeout(context.Background(), 14*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "applications-browser")
	if err != nil {
		t.Fatal(err)
	}
	run := func(name, script, app, readyPath string, port, want int, snapshot string) {
		t.Helper()
		root, home := stageApplication(t, ctx, bundle, sourceRoot, app)
		_, dir := applicationBuild(t, ctx, bundle, home, root)
		base, stop := serveApplication(t, ctx, bundle, home, filepath.Join(dir, "entry.ts"), snapshot, port, readyPath)
		defer stop()
		outdir := t.TempDir()
		browser := exec.CommandContext(ctx, nodePath, script, base, outdir)
		browser.Dir = browserDir
		browser.Env = []string{"PATH=" + filepath.Dir(nodePath) + ":/usr/bin:/bin", "HOME=" + os.Getenv("HOME")}
		result, err := browser.CombinedOutput()
		if err != nil {
			t.Fatalf("%s harness: %v %s", name, err, result)
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
		if err := json.Unmarshal(raw, &report); err != nil || !report.Passed || len(report.Checks) != want {
			t.Fatalf("%s invalid browser report %v %s", name, err, raw)
		}
		for _, entry := range report.Requests {
			if !strings.HasPrefix(entry.URL, base+"/") {
				t.Fatalf("%s browser left loopback: %s", name, entry.URL)
			}
		}
		shot, err := os.Stat(filepath.Join(outdir, "screenshot.png"))
		if err != nil || shot.Size() == 0 {
			t.Fatalf("%s missing browser screenshot", name)
		}
		if evidence := os.Getenv("CAN_BROWSER_EVIDENCE_DIR"); evidence != "" {
			dest := filepath.Join(evidence, name)
			if err := os.MkdirAll(dest, 0700); err != nil {
				t.Fatal(err)
			}
			copyEvidenceFile(t, filepath.Join(outdir, "report.json"), filepath.Join(dest, "report.json"))
			copyEvidenceFile(t, filepath.Join(outdir, "screenshot.png"), filepath.Join(dest, "screenshot.png"))
		}
		t.Logf("%s browser %s %s: %d checks, %d loopback requests", name, report.Browser, report.Version, len(report.Checks), len(report.Requests))
	}
	stageHome := t.TempDir()
	run("forms", "forms.mjs", "form-validation", "/signup", 18541, 8, snapshotCredential(t, stageHome, "UNUSED", "postgres://127.0.0.1:1/nope"))
	dbURL := os.Getenv("CAN_TEST_POSTGRES_URL")
	if dbURL == "" {
		t.Log("CAN_TEST_POSTGRES_URL unset; live browser legs skipped")
		return
	}
	seedHome, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	applicationSeed(t, ctx, bundle, sourceRoot, seedHome, dbURL, "setup")
	accountsHome := t.TempDir()
	run("accounts", "accounts.mjs", "account-search", "/accounts", 18542, 9, snapshotCredential(t, accountsHome, "ACCOUNTS_DB", dbURL))
	dashboardHome := t.TempDir()
	run("dashboard", "dashboard.mjs", "dashboard", "/dashboard", 18543, 5, snapshotCredential(t, dashboardHome, "DASHBOARD_DB", dbURL))
	applicationSeed(t, ctx, bundle, sourceRoot, seedHome, dbURL, "teardown")
}
