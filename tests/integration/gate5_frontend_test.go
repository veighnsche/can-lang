// T25 Gate 5 broad Can-authored frontend qualification. The staged live grid
// (the T24 Can-authored invoice grid, `canlc build --target browser`, bundled
// for real browsers) runs against the installed-or-staged invoice server on a
// same-origin test origin with a disposable database per browser, and the
// committed Playwright harness (tests/integration/browser/grid.mjs) drives it
// through boot, edits, rows, every save outcome, offline/slow/reconnect,
// truncated and garbage responses after commit, identical-id replay, conflict
// adopt-or-keep, leak/disposal/navigation and denied loads. The Go test owns
// the toolchain, builds, bundle, origin, fault injection and database
// cross-checks; the harness owns DOM, focus, announcements and network bytes.
//
// Known divergences pinned here rather than hidden (see the per-leg asserts
// and the report's limitations, mirroring the T17 known-limitation pattern):
//
//   - the grid fetches the declared captured load path GET /invoices/{id},
//     while the server serves /invoices/load?invoice_id= (its own comment:
//     "until the action adapter mounts JSON bodies and captured paths with
//     request context"). The origin rewrites exactly that one shape and the
//     test proves the rewrite is load-bearing (direct captured GET 404s) and
//     records every rewrite;
//   - row-level focus intents invoke focus() before the grid attaches, so
//     add/move focus never lands (remove-to-add lands; typing never disturbs
//     focus). The harness records the invocation proof as L-focus-row;
//   - the blocked-save guard notice is wiped by the immediate re-render
//     (L-notice; nothing is ever sent) and presses show no mid-flight
//     "saving..." indication (L-flight; the single-flight guard holds).
//
// Engine shims (tests/integration/browser/build-grid.mjs): node:async_hooks,
// node:util, node:fs (real source spans, zero module maps) and node:crypto
// (synchronous SHA-256, machine-checked against Go crypto/sha256 below). No
// other harness-authored client code exists: the boot entry is three fixed
// lines invoking the emitted $canBrowserMain, asserted byte-for-byte.
//
// Toolchain selection mirrors Gate 4: on linux/amd64 the test builds,
// releases and installs the distribution and drives the installed launcher
// and sidecar; elsewhere the same legs run against a staged bundle. Without
// CAN_BUN_ARCHIVE the tests skip by design, as do browser legs whose
// Playwright launcher cannot start (chromium is required; firefox and webkit
// are best-effort and the report names the exact qualified matrix).
package integration

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/distribution"
)

const (
	gate5APIChromium    = 18641
	gate5OriginChromium = 18642
	gate5APIFirefox     = 18643
	gate5OriginFirefox  = 18644
	gate5APIWebkit      = 18645
	gate5OriginWebkit   = 18646
	gate5APIStatic      = 18647

	gate5HostPage = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Invoice grid</title>
</head>
<body>
<div id="invoice-grid"></div>
<script type="module" src="/grid/boot.js"></script>
</body>
</html>
`
	gate5CSP = "default-src 'none'; script-src 'self'; connect-src 'self'; base-uri 'none'"
)

var gate5ServerModules = []string{
	"platform/server.ts",
	"platform/env.ts",
	"platform/s3.ts",
	"platform/websocket.ts",
	"platform/sql/",
	"platform/files/",
	"platform/process/",
	"platform/crypto/",
}

// gate5NoDirectIO asserts the emitted program never imports the io module
// directly: platform/io.ts ships in the browser bundle only transitively via
// log.ts's pure expectedIOFailure helper, while its Bun-backed createIO stays
// uncalled (its default Bun reference never evaluates).
func gate5NoDirectIO(t *testing.T, dir string) {
	t.Helper()
	for _, sub := range []string{"program", "packages", "browser.ts"} {
		root := filepath.Join(dir, sub)
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(path, ".ts") {
				return err
			}
			content, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			if strings.Contains(string(content), "platform/io") {
				t.Fatalf("%s imports the io module directly", path)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

// gate5Canlc runs the toolchain launcher with the network denied on darwin
// and a restricted environment on linux, where sandbox-exec does not exist.
func gate5Canlc(t *testing.T, ctx context.Context, canlc, home string, args ...string) (int, string, string) {
	t.Helper()
	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		argv := append([]string{"-p", "(version 1)(allow default)(deny network*)", canlc}, args...)
		cmd = exec.CommandContext(ctx, "/usr/bin/sandbox-exec", argv...)
	} else {
		cmd = exec.CommandContext(ctx, canlc, args...)
	}
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

// gate5Toolchain resolves the artifact under test. On linux/amd64 it performs
// the full build, release and install cycle and returns the installed
// selection; elsewhere it returns a staged bundle for the same legs.
func gate5Toolchain(t *testing.T, ctx context.Context, sourceRoot, archive string) (root, canlc, sidecar string, installed bool) {
	t.Helper()
	if runtime.GOOS == "linux" && runtime.GOARCH == "amd64" {
		bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "gate5-grid")
		if err != nil {
			t.Fatal(err)
		}
		artifacts, err := distribution.Release(ctx, bundle, t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		installRoot := t.TempDir()
		selected, err := distribution.Install(ctx, artifacts.Archive, artifacts.SHA256, installRoot)
		if err != nil {
			t.Fatal(err)
		}
		selection, err := distribution.Selection(installRoot)
		if err != nil || selection != selected {
			t.Fatalf("wrong install selection %q: %v", selection, err)
		}
		root = filepath.Join(installRoot, "current")
		return root, filepath.Join(root, "bin/canlc"), filepath.Join(root, "runtime/bun"), true
	}
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "gate5-grid")
	if err != nil {
		t.Fatal(err)
	}
	return bundle, filepath.Join(bundle, "bin/canlc"), filepath.Join(bundle, "runtime/bun"), false
}

func gate5Assert(t *testing.T, ctx context.Context, canlc, home, root, name string) (assertions, real int) {
	t.Helper()
	status, out, diag := gate5Canlc(t, ctx, canlc, home, "assert", root)
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
		t.Fatalf("invalid %s assert report %v %s", name, err, out)
	}
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
	return len(report.Assertions), real
}

func gate5Build(t *testing.T, ctx context.Context, canlc, home, root string, extra ...string) (string, string) {
	t.Helper()
	args := append([]string{"build"}, extra...)
	args = append(args, root)
	status, out, diag := gate5Canlc(t, ctx, canlc, home, args...)
	if status != 0 {
		t.Fatalf("build %v: %d %s %s", extra, status, out, diag)
	}
	var report struct {
		BuildID   string `json:"buildID"`
		Directory string `json:"directory"`
		Entry     string `json:"entry"`
		Asset     string `json:"asset"`
	}
	if err := json.Unmarshal([]byte(out), &report); err != nil || report.BuildID == "" || report.Directory == "" {
		t.Fatalf("invalid build report %v %s", err, out)
	}
	return report.BuildID, report.Directory
}

// gate5Asset verifies the content-addressed browser asset manifest: identity,
// profile, entry, the content digest over the file map, and every file digest
// against the shipped bytes.
func gate5Asset(t *testing.T, dir string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, "browser/asset.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		SchemaVersion int               `json:"schemaVersion"`
		Kind          string            `json:"kind"`
		Profile       string            `json:"profile"`
		Entry         string            `json:"entry"`
		ContentSHA256 string            `json:"contentSHA256"`
		Files         map[string]string `json:"files"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("asset invalid: %v", err)
	}
	if manifest.SchemaVersion != 1 || manifest.Kind != "can.browser-asset" || manifest.Profile != "browser-main" || manifest.Entry != "browser.ts" {
		t.Fatalf("asset identity = %+v", manifest)
	}
	if len(manifest.Files) == 0 {
		t.Fatal("asset binds no files")
	}
	names := make([]string, 0, len(manifest.Files))
	for name := range manifest.Files {
		names = append(names, name)
	}
	sort.Strings(names)
	hash := sha256.New()
	for _, name := range names {
		hash.Write([]byte(name))
		hash.Write([]byte{0})
		hash.Write([]byte(manifest.Files[name]))
		hash.Write([]byte{0})
	}
	if hex.EncodeToString(hash.Sum(nil)) != manifest.ContentSHA256 {
		t.Fatal("asset content digest mismatch")
	}
	for name, digest := range manifest.Files {
		content, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(content)
		if hex.EncodeToString(sum[:]) != digest {
			t.Fatalf("asset digest mismatch for %s", name)
		}
	}
}

// gate5ImportAudit asserts the emitted import graph carries the browser entry
// and no edge from any module to a server-only runtime module.
func gate5ImportAudit(t *testing.T, dir string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var generation struct {
		Entry   string              `json:"entry"`
		Imports map[string][]string `json:"imports"`
	}
	if err := json.Unmarshal(raw, &generation); err != nil {
		t.Fatal(err)
	}
	if generation.Entry != "browser.ts" {
		t.Fatalf("generation entry = %q", generation.Entry)
	}
	for from, edges := range generation.Imports {
		for _, edge := range edges {
			for _, banned := range []string{"/platform/sql/", "/platform/files/", "/platform/process/", "/platform/env", "/platform/crypto/", "/platform/server", "/platform/io", "/platform/s3", "/platform/websocket"} {
				if strings.Contains(edge, banned) {
					t.Fatalf("emitted edge %s -> %s reaches a server module", from, edge)
				}
			}
		}
	}
}

// gate5Bundle runs the committed grid bundler under the toolchain bun and
// returns the bundle bytes. The generated boot entry must equal the fixed
// three-line template: the only harness-authored client code.
func gate5Bundle(t *testing.T, ctx context.Context, bun, browserTs, outdir string) []byte {
	t.Helper()
	cmd := exec.CommandContext(ctx, bun, filepath.Join("tests/integration/browser", "build-grid.mjs"), browserTs, outdir)
	cmd.Dir = mustSourceRoot(t)
	cmd.Env = []string{"PATH=/usr/bin:/bin", "HOME=" + t.TempDir()}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("grid bundle: %v %s", err, out)
	}
	bundlePath := strings.TrimSpace(string(out))
	content, err := os.ReadFile(bundlePath)
	if err != nil {
		t.Fatal(err)
	}
	boot, err := os.ReadFile(filepath.Join(outdir, "grid-boot.ts"))
	if err != nil {
		t.Fatal(err)
	}
	want := fmt.Sprintf("import { $canBrowserMain } from %s;\nconst invoice = new URLSearchParams(location.search).get(\"invoice\") ?? \"inv-1\";\nglobalThis.__canGridBoot = $canBrowserMain([invoice]);\n", quoteJS(browserTs))
	if string(boot) != want {
		t.Fatalf("boot entry drifted:\n%s\nwant:\n%s", boot, want)
	}
	return content
}

func quoteJS(path string) string {
	var out strings.Builder
	out.WriteByte('"')
	for _, r := range path {
		switch r {
		case '"', '\\':
			out.WriteByte('\\')
			out.WriteRune(r)
		default:
			out.WriteRune(r)
		}
	}
	out.WriteByte('"')
	return out.String()
}

func mustSourceRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// gate5Origin is the same-origin test host: it serves the grid host page and
// browser bundle, reverse-proxies the invoice API, rewrites the one divergent
// load shape, and injects one-shot after-commit response faults.
type gate5Origin struct {
	api            string
	bundle         []byte
	mu             sync.Mutex
	fault          string
	faultsConsumed map[string]int
	faultUpstream  map[string]int
	rewrites       []string
	pageHits       int
	bundleHits     int
}

func newGate5Origin(api string, bundle []byte) *gate5Origin {
	if _, err := url.Parse(api); err != nil {
		panic(err)
	}
	return &gate5Origin{
		api:            api,
		bundle:         bundle,
		faultsConsumed: map[string]int{},
		faultUpstream:  map[string]int{},
	}
}

var gate5CapturedLoad = regexp.MustCompile(`^/invoices/([^/]+)$`)

func (o *gate5Origin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	switch {
	case r.URL.Path == "/grid" && r.Method == "GET":
		o.mu.Lock()
		o.pageHits++
		o.mu.Unlock()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Security-Policy", gate5CSP)
		_, _ = io.WriteString(w, gate5HostPage)
		return
	case r.URL.Path == "/grid/boot.js" && r.Method == "GET":
		o.mu.Lock()
		o.bundleHits++
		o.mu.Unlock()
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Header().Set("Content-Length", strconv.Itoa(len(o.bundle)))
		_, _ = w.Write(o.bundle)
		return
	case r.URL.Path == "/t25/fault" && r.Method == "POST":
		body, _ := io.ReadAll(io.LimitReader(r.Body, 32))
		mode := strings.TrimSpace(string(body))
		if mode != "truncate" && mode != "garbage" {
			http.Error(w, "unknown fault", http.StatusBadRequest)
			return
		}
		o.mu.Lock()
		o.fault = mode
		o.mu.Unlock()
		_, _ = io.WriteString(w, "armed:"+mode)
		return
	}
	if r.URL.Path != "/health" && !strings.HasPrefix(r.URL.Path, "/invoices/") {
		http.NotFound(w, r)
		return
	}
	upstream := r.URL.Path
	if r.Method == "GET" {
		if match := gate5CapturedLoad.FindStringSubmatch(r.URL.Path); match != nil {
			// Known divergence L-route: the grid fetches the declared
			// captured load path; the server serves the query spelling.
			// The rewrite is pinned and ledgered, never silent.
			upstream = "/invoices/load?invoice_id=" + url.QueryEscape(match[1])
			o.mu.Lock()
			o.rewrites = append(o.rewrites, r.Method+" "+r.URL.Path+" -> "+upstream)
			o.mu.Unlock()
		}
	}
	o.mu.Lock()
	fault := ""
	if r.Method == "POST" && r.URL.Path == "/invoices/save" {
		fault, o.fault = o.fault, ""
	}
	o.mu.Unlock()
	if fault == "" {
		(&httputil.ReverseProxy{
			Director: func(out *http.Request) {
				target, _ := url.Parse(o.api + upstream)
				out.URL = target
				out.Host = target.Host
			},
		}).ServeHTTP(w, r)
		return
	}
	// After-commit fault: forward first so the server commits, then corrupt
	// only the response the browser reads.
	forward, err := http.NewRequest(r.Method, o.api+upstream, r.Body)
	if err != nil {
		http.Error(w, "fault setup", http.StatusInternalServerError)
		return
	}
	forward.Header = r.Header.Clone()
	upstreamRes, err := http.DefaultClient.Do(forward)
	if err != nil {
		http.Error(w, "fault upstream", http.StatusBadGateway)
		return
	}
	defer upstreamRes.Body.Close()
	full, err := io.ReadAll(io.LimitReader(upstreamRes.Body, 1<<20))
	if err != nil {
		http.Error(w, "fault drain", http.StatusBadGateway)
		return
	}
	o.mu.Lock()
	o.faultsConsumed[fault]++
	o.faultUpstream[fault] = upstreamRes.StatusCode
	o.mu.Unlock()
	if upstreamRes.StatusCode != 200 {
		t := upstreamRes.StatusCode
		http.Error(w, "fault leg expected an upstream commit", t)
		return
	}
	if fault == "garbage" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = io.WriteString(w, `{"case":`)
		return
	}
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "truncate needs a hijackable conn", http.StatusInternalServerError)
		return
	}
	conn, writer, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, "truncate hijack", http.StatusInternalServerError)
		return
	}
	defer conn.Close()
	cut := full
	if len(cut) > 10 {
		cut = cut[:10]
	}
	_, _ = fmt.Fprintf(writer, "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: %d\r\nConnection: close\r\n\r\n", len(full))
	_, _ = writer.Write(cut)
	_ = writer.Flush()
}

// serveGate5Origin starts the origin on a loopback port and returns its base
// URL with a stopper.
func serveGate5Origin(t *testing.T, origin *gate5Origin, port int) (string, func()) {
	t.Helper()
	server := &http.Server{Addr: "127.0.0.1:" + strconv.Itoa(port), Handler: origin}
	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = server.Serve(listener) }()
	base := "http://127.0.0.1:" + strconv.Itoa(port)
	deadline := time.Now().Add(10 * time.Second)
	for {
		response, err := http.Get(base + "/grid")
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == 200 {
				break
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("gate5 origin never ready on %s", base)
		}
		time.Sleep(100 * time.Millisecond)
	}
	return base, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}
}

func gate5BrowserProbe(t *testing.T, ctx context.Context, nodePath, browserDir, name string) (string, bool) {
	t.Helper()
	probeCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	probe := exec.CommandContext(probeCtx, nodePath, "--input-type=module", "-e",
		`import("playwright").then(async (pw) => { const browser = await pw["`+name+`"].launch({ timeout: 90000 }); console.log(browser.version()); await browser.close(); })`)
	probe.Dir = browserDir
	result, err := probe.CombinedOutput()
	if err != nil {
		first := strings.TrimSpace(string(result))
		if i := strings.Index(first, "\n"); i >= 0 {
			first = first[:i]
		}
		if len(first) > 300 {
			first = first[:300]
		}
		t.Logf("gate5 %s unavailable: %s", name, first)
		return "", false
	}
	return strings.TrimSpace(string(result)), true
}

type gate5Report struct {
	Browser   string `json:"browser"`
	Version   string `json:"version"`
	UserAgent string `json:"userAgent"`
	Passed    bool   `json:"passed"`
	Checks    []struct {
		Name   string `json:"name"`
		Passed bool   `json:"passed"`
		Detail string `json:"detail"`
	} `json:"checks"`
	Limitations []struct {
		ID     string `json:"id"`
		Detail string `json:"detail"`
	} `json:"limitations"`
	Requests []struct {
		Method string `json:"method"`
		URL    string `json:"url"`
		Status int    `json:"status"`
	} `json:"requests"`
	Ledger []struct {
		Method       string `json:"method"`
		URL          string `json:"url"`
		Status       int    `json:"status"`
		RequestBody  string `json:"requestBody"`
		ResponseBody string `json:"responseBody"`
	} `json:"ledger"`
	Aborted       []string `json:"aborted"`
	PageErrors    []string `json:"pageerrors"`
	ConsoleErrors []string `json:"consoleErrors"`
}

func TestGate5GridMatrix(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged gate 5 execution")
	}
	nodePath, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is required for the browser harness")
	}
	sourceRoot := mustSourceRoot(t)
	browserDir := filepath.Join(sourceRoot, "tests/integration/browser")
	if _, err := os.Stat(filepath.Join(browserDir, "node_modules/playwright/package.json")); err != nil {
		t.Skip("run bun ci in tests/integration/browser for the pinned harness")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	toolchain, canlc, sidecar, installed := gate5Toolchain(t, ctx, sourceRoot, archive)
	mode := "staged bundle"
	if installed {
		mode = "installed release"
	}
	home := t.TempDir()
	host := gate4RuntimeIdentity(t, ctx, canlc, home)
	archiveDigest := sha256.Sum256(mustReadFile(t, archive))
	t.Logf("gate5 host bound: %s on %s/%s bun %s rev %s (archive sha256 %s)",
		mode, host["platform"], host["architecture"], host["bun"], host["revision"][:12], hex.EncodeToString(archiveDigest[:])[:12])

	serverRoot, serverHome := stageApplication(t, ctx, toolchain, sourceRoot, "invoice")
	serverAssertions, serverReal := gate5Assert(t, ctx, canlc, serverHome, serverRoot, "invoice")
	serverFirst, _ := gate5Build(t, ctx, canlc, serverHome, serverRoot)
	serverSecond, serverDir := gate5Build(t, ctx, canlc, serverHome, serverRoot)
	if serverFirst != serverSecond {
		t.Fatalf("server rebuild drifted: %s vs %s", serverFirst, serverSecond)
	}
	assertNoStrayEmit(t, serverRoot, serverDir)

	gridRoot, gridHome := stageProject(t, sourceRoot, "examples/invoice-grid")
	gridAssertions, gridReal := gate5Assert(t, ctx, canlc, gridHome, gridRoot, "grid")
	gridFirst, _ := gate5Build(t, ctx, canlc, gridHome, gridRoot, "--target", "browser")
	gridSecond, gridDir := gate5Build(t, ctx, canlc, gridHome, gridRoot, "--target", "browser")
	if gridFirst != gridSecond {
		t.Fatalf("grid rebuild drifted: %s vs %s", gridFirst, gridSecond)
	}
	assertNoStrayEmit(t, gridRoot, gridDir)
	gate5Asset(t, gridDir)
	gate5ImportAudit(t, gridDir)
	gate5NoDirectIO(t, gridDir)

	browserTs := filepath.Join(gridDir, "browser.ts")
	bundleDir := t.TempDir()
	firstBundle := gate5Bundle(t, ctx, sidecar, browserTs, bundleDir)
	secondBundle := gate5Bundle(t, ctx, sidecar, browserTs, bundleDir)
	if !bytes.Equal(firstBundle, secondBundle) {
		t.Fatal("grid bundle is not deterministic")
	}
	for _, want := range []string{"$canBrowserMain", "platform/browser.ts", "expectedIOFailure", "/invoices/save", "/invoices/{invoice_id}"} {
		if !bytes.Contains(firstBundle, []byte(want)) {
			t.Fatalf("bundle lacks %q", want)
		}
	}
	for _, banned := range gate5ServerModules {
		if bytes.Contains(firstBundle, []byte(banned)) {
			t.Fatalf("bundle reaches server-only %s", banned)
		}
	}
	t.Logf("gate5 builds: server %d assertions (%d real-can) build %s; grid %d assertions (%d real-can) browser build %s, bundle %d bytes deterministic",
		serverAssertions, serverReal, serverFirst[:12], gridAssertions, gridReal, gridFirst[:12], len(firstBundle))

	browsers := []struct {
		name        string
		api, origin int
		required    bool
	}{
		{"chromium", gate5APIChromium, gate5OriginChromium, true},
		{"firefox", gate5APIFirefox, gate5OriginFirefox, false},
		{"webkit", gate5APIWebkit, gate5OriginWebkit, false},
	}
	qualified := map[string]string{}
	unavailable := []string{}
	driver := filepath.Join(sourceRoot, "tests/integration/testdata/invoice/driver.ts")
	entry := filepath.Join(serverDir, "entry.ts")
	for _, candidate := range browsers {
		version, ok := gate5BrowserProbe(t, ctx, nodePath, browserDir, candidate.name)
		if !ok {
			if candidate.required {
				t.Fatalf("required browser %s did not launch", candidate.name)
			}
			unavailable = append(unavailable, candidate.name)
			continue
		}
		qualified[candidate.name] = version
		bhome := t.TempDir()
		db, err := filepath.EvalSymlinks(filepath.Join(bhome, "grid.sqlite"))
		if err != nil {
			resolved, resolveErr := filepath.EvalSymlinks(bhome)
			if resolveErr != nil {
				t.Fatal(resolveErr)
			}
			db = filepath.Join(resolved, "grid.sqlite")
		}
		setup := invoiceDriver(t, ctx, toolchain, bhome, driver, "setup", db, filepath.Join(serverRoot, "schema.sql"))
		var setupReport struct {
			Tables int `json:"tables"`
		}
		if err := json.Unmarshal(setup, &setupReport); err != nil || setupReport.Tables != 5 {
			t.Fatalf("invalid grid setup report %v %s", err, string(setup))
		}
		invoiceDriver(t, ctx, toolchain, bhome, driver, "seed", db)
		snapshot := snapshotCredential(t, bhome, "INVOICE_DB", db)
		_, stopAPI := serveApplication(t, ctx, toolchain, bhome, entry, snapshot, candidate.api, "/health")
		origin := newGate5Origin("http://127.0.0.1:"+strconv.Itoa(candidate.api), firstBundle)
		originBase, stopOrigin := serveGate5Origin(t, origin, candidate.origin)

		outdir := t.TempDir()
		harness := exec.CommandContext(ctx, nodePath, "grid.mjs", candidate.name, originBase, outdir, db)
		harness.Dir = browserDir
		harness.Env = []string{"PATH=" + filepath.Dir(nodePath) + ":/usr/bin:/bin", "HOME=" + os.Getenv("HOME")}
		result, err := harness.CombinedOutput()
		stopOrigin()
		stopAPI()
		if err != nil {
			t.Fatalf("%s harness: %v %s", candidate.name, err, result)
		}
		raw, err := os.ReadFile(filepath.Join(outdir, "report.json"))
		if err != nil {
			t.Fatal(err)
		}
		var report gate5Report
		if err := json.Unmarshal(raw, &report); err != nil || !report.Passed || len(report.Checks) != 27 {
			t.Fatalf("%s invalid grid report %v %s", candidate.name, err, raw)
		}
		for _, entry := range report.Checks {
			if !entry.Passed {
				t.Fatalf("%s check %s failed: %s", candidate.name, entry.Name, entry.Detail)
			}
		}
		if report.Browser != candidate.name || report.Version == "" || report.UserAgent == "" {
			t.Fatalf("%s report identity %+v", candidate.name, report)
		}
		if len(report.Aborted) != 0 || len(report.PageErrors) != 0 {
			t.Fatalf("%s aborted=%v pageerrors=%v", candidate.name, report.Aborted, report.PageErrors)
		}
		for _, entry := range report.Requests {
			parsed, parseErr := url.Parse(entry.URL)
			if parseErr != nil || (parsed.Hostname() != "127.0.0.1" && parsed.Hostname() != "localhost") {
				t.Fatalf("%s grid left loopback: %s", candidate.name, entry.URL)
			}
		}
		shot, err := os.Stat(filepath.Join(outdir, "screenshot.png"))
		if err != nil || shot.Size() == 0 {
			t.Fatalf("%s missing browser screenshot", candidate.name)
		}

		origin.mu.Lock()
		rewrites, consumed, upstream, pageHits, bundleHits := origin.rewrites, origin.faultsConsumed, origin.faultUpstream, origin.pageHits, origin.bundleHits
		origin.mu.Unlock()
		if pageHits == 0 || bundleHits == 0 {
			t.Fatalf("%s origin served page=%d bundle=%d", candidate.name, pageHits, bundleHits)
		}
		if len(rewrites) == 0 {
			t.Fatalf("%s origin rewrote no captured loads", candidate.name)
		}
		for _, rewrite := range rewrites {
			if !strings.HasPrefix(rewrite, "GET /invoices/") || !strings.Contains(rewrite, " -> /invoices/load?invoice_id=") {
				t.Fatalf("%s unexpected rewrite %q", candidate.name, rewrite)
			}
		}
		if consumed["truncate"] != 1 || consumed["garbage"] != 1 || upstream["truncate"] != 200 || upstream["garbage"] != 200 {
			t.Fatalf("%s faults consumed=%v upstream=%v", candidate.name, consumed, upstream)
		}

		// Database cross-check: final committed state plus exactly-once
		// replay evidence for every operation id the harness sent.
		store := inspectInvoice(t, ctx, toolchain, bhome, driver, db)
		requireInvoiceRevision(t, store, "inv-1", "9", "Racer")
		requireInvoiceRevision(t, store, "inv-2", "1", "Globex")
		if len(store.Lines) != 1 || store.Lines[0].LineKey != "k1" || store.Lines[0].SKU != "sku-1" || store.Lines[0].Qty != "2" || store.Lines[0].Position != "0" {
			t.Fatalf("%s lines %+v", candidate.name, store.Lines)
		}
		// Every wire op classified by its response cases: the 8 saved ops
		// must match the 8 replay rows exactly (attempt and replay share
		// one id and one effect), while the rejected, stale and denied
		// attempts must have crossed the wire and left no row.
		opCases := map[string]map[string]bool{}
		for _, call := range report.Ledger {
			if call.Method != "POST" || !strings.HasSuffix(call.URL, "/invoices/save") {
				continue
			}
			var body struct {
				OperationID string `json:"operation_id"`
			}
			if err := json.Unmarshal([]byte(call.RequestBody), &body); err != nil || body.OperationID == "" {
				t.Fatalf("%s ledger holds an unreadable save body %q", candidate.name, call.RequestBody)
			}
			if opCases[body.OperationID] == nil {
				opCases[body.OperationID] = map[string]bool{}
			}
			var outcome struct {
				Case string `json:"case"`
			}
			if err := json.Unmarshal([]byte(call.ResponseBody), &outcome); err == nil && outcome.Case != "" {
				opCases[body.OperationID][outcome.Case] = true
			}
		}
		if len(store.Replay) != 8 {
			t.Fatalf("%s replay rows %+v, want 8", candidate.name, store.Replay)
		}
		replayOps := map[string]bool{}
		for _, row := range store.Replay {
			if len(row.Digest) != 64 || row.InvoiceID != "inv-1" {
				t.Fatalf("%s replay row %+v", candidate.name, row)
			}
			if replayOps[row.OperationID] {
				t.Fatalf("%s duplicated replay for %s", candidate.name, row.OperationID)
			}
			replayOps[row.OperationID] = true
		}
		saved, terminal := 0, map[string]int{}
		for op, cases := range opCases {
			switch {
			case cases["records::saved"]:
				saved++
				if !replayOps[op] {
					t.Fatalf("%s saved op %s left no replay row", candidate.name, op)
				}
			default:
				for _, leaf := range []string{"records::rejected", "records::stale", "records::denied"} {
					if cases[leaf] {
						terminal[leaf]++
					}
				}
				if replayOps[op] {
					t.Fatalf("%s unsaved op %s left a replay row", candidate.name, op)
				}
			}
		}
		if saved != 8 || len(replayOps) != 8 {
			t.Fatalf("%s saved ops = %d, replay rows = %d, want 8 and 8", candidate.name, saved, len(replayOps))
		}
		for _, leaf := range []string{"records::rejected", "records::stale", "records::denied"} {
			if terminal[leaf] != 1 {
				t.Fatalf("%s terminal case %s seen %d times, want 1", candidate.name, leaf, terminal[leaf])
			}
		}
		if evidence := os.Getenv("CAN_BROWSER_EVIDENCE_DIR"); evidence != "" {
			dest := filepath.Join(evidence, "grid-"+candidate.name)
			if err := os.MkdirAll(dest, 0700); err != nil {
				t.Fatal(err)
			}
			copyEvidenceFile(t, filepath.Join(outdir, "report.json"), filepath.Join(dest, "report.json"))
			copyEvidenceFile(t, filepath.Join(outdir, "screenshot.png"), filepath.Join(dest, "screenshot.png"))
		}
		t.Logf("gate5 %s %s: 27 checks, %d loopback requests, %d invoice calls, rev 9 Racer committed with 8 replay rows",
			candidate.name, report.Version, len(report.Requests), len(report.Ledger))
	}
	finalReport, err := json.MarshalIndent(map[string]any{
		"mode": mode, "host": host,
		"server": map[string]any{"assertions": serverAssertions, "real_can": serverReal, "build": serverFirst},
		"grid":   map[string]any{"assertions": gridAssertions, "real_can": gridReal, "build": gridFirst, "bundle_bytes": len(firstBundle)},
		"matrix": qualified, "unavailable": unavailable,
		"limitations": []string{
			"L-route: origin rewrites GET /invoices/{id} to /invoices/load?invoice_id=; the server serves the query spelling until the action adapter mounts captured paths",
			"L-focus-row: add/move focus() runs pre-attach and never lands; remove-to-add lands and typing never disturbs focus",
			"L-notice: blocked-save guard notice is wiped by the re-render; nothing is ever sent",
			"L-flight: no mid-flight saving indication; the single-flight guard holds",
			"L-wire: grid wire-field renames pass grid-only assert (positional construction); wire agreement is owned by the cross-target driver test",
			"L-unlink: server action renames pass with the served spelling untouched; declarations are not linked to served routes",
			"engine shims: node:async_hooks/node:util/node:fs(node:fs serves the real source index, zero module maps)/node:crypto(SHA-256, parity-checked)",
		},
	}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("gate5 report:\n%s", finalReport)
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// TestGate5GridStatic pins the browser-independent Gate 5 contracts without a
// browser: the SHA-256 shim's digest parity against Go crypto/sha256, server
// capability rejection of the browser target, the grid's timer-free leak
// surface, the liveness of the load-path divergence (the captured GET 404s on
// the API directly, so the origin rewrite is load-bearing), and four
// contract-edit legs (grid route edits reach fetch bytes; grid draft renames
// diagnose stale uses; grid wire renames pass positionally, pinning L-wire;
// server action renames pass unlinked, pinning L-unlink).
func TestGate5GridStatic(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged gate 5 execution")
	}
	sourceRoot := mustSourceRoot(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	toolchain, canlc, sidecar, _ := gate5Toolchain(t, ctx, sourceRoot, archive)

	vectors := []string{
		"",
		"abc",
		"can-callable-instance-v1\x00" + `[[{"root":{"package":"p","declaration":"d","name":"n"},"segments":[]}],"site#3",7]`,
		strings.Repeat("x", 1000),
		"héllo wörld ✓",
	}
	runner := "import { createHash } from " + strconv.Quote(filepath.Join(sourceRoot, "tests/integration/browser/sha256-shim.mjs")) + ";\n" +
		"const vectors = " + mustMarshalJSON(t, vectors) + ";\n" +
		"for (const v of vectors) console.log(createHash(\"sha256\").update(v).digest(\"hex\"));\n"
	script := filepath.Join(t.TempDir(), "vectors.mjs")
	if err := os.WriteFile(script, []byte(runner), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(ctx, sidecar, script)
	cmd.Dir = t.TempDir()
	cmd.Env = []string{"PATH=/nonexistent", "HOME=" + t.TempDir()}
	vectorOut, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("shim vectors: %v %s", err, vectorOut)
	}
	digests := strings.Split(strings.TrimSpace(string(vectorOut)), "\n")
	if len(digests) != len(vectors) {
		t.Fatalf("shim vectors = %d, want %d: %s", len(digests), len(vectors), vectorOut)
	}
	for i, vector := range vectors {
		sum := sha256.Sum256([]byte(vector))
		if digests[i] != hex.EncodeToString(sum[:]) {
			t.Fatalf("shim vector %d: %s, want %s", i, digests[i], hex.EncodeToString(sum[:]))
		}
	}
	t.Logf("gate5 shim: %d SHA-256 vectors agree with Go crypto/sha256", len(vectors))

	serverRoot, serverHome := stageApplication(t, ctx, toolchain, sourceRoot, "invoice")
	status, out, diag := gate5Canlc(t, ctx, canlc, serverHome, "build", "--target", "browser", serverRoot)
	combined := out + "\n" + diag
	if status == 0 || !strings.Contains(combined, "browser capability closure") || !strings.Contains(combined, "--target browser") {
		t.Fatalf("server browser build: %d %.500s, want a capability-closure rejection", status, combined)
	}
	t.Logf("gate5 capability: server project rejected for --target browser: %s", strings.TrimSpace(combined))

	gridSources := filepath.Join(sourceRoot, "examples/invoice-grid/src")
	var timerUses []string
	err = filepath.WalkDir(gridSources, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".can") {
			return err
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for _, token := range []string{"set_timeout", "setTimeout"} {
			if strings.Contains(string(content), token) {
				timerUses = append(timerUses, path+":"+token)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(timerUses) != 0 {
		t.Fatalf("grid sets timers, widening the leak surface: %v", timerUses)
	}

	// The divergence is live: the API serves the query spelling and 404s the
	// captured spelling the grid fetches, so the origin rewrite does real work.
	_, _ = gate5Assert(t, ctx, canlc, serverHome, serverRoot, "invoice")
	firstID, firstDir := gate5Build(t, ctx, canlc, serverHome, serverRoot)
	secondID, _ := gate5Build(t, ctx, canlc, serverHome, serverRoot)
	if firstID != secondID {
		t.Fatalf("static server rebuild drifted: %s vs %s", firstID, secondID)
	}
	driver := filepath.Join(sourceRoot, "tests/integration/testdata/invoice/driver.ts")
	db := seedInvoiceDB(t, ctx, toolchain, serverHome, driver, serverRoot)
	snapshot := snapshotCredential(t, serverHome, "INVOICE_DB", db)
	apiBase, stopAPI := serveApplication(t, ctx, toolchain, serverHome, filepath.Join(firstDir, "entry.ts"), snapshot, gate5APIStatic, "/health")
	defer stopAPI()
	if status, body, _ := invoiceGet(t, apiBase, "/invoices/inv-1", "tok-alice"); status != 404 || body != "Not Found" {
		t.Fatalf("captured load on the API: %d %q, want direct 404", status, body)
	}
	if status, body, _ := invoiceGet(t, apiBase, "/invoices/load?invoice_id=inv-1", "tok-alice"); status != 200 || !strings.Contains(body, `"records::found"`) {
		t.Fatalf("query load on the API: %d %.200s, want the served spelling", status, body)
	}

	// Grid route edits reach fetch bytes without touching the server: the
	// declaration moves, the browser build passes, and the bundle carries the
	// new path and no trace of the old one.
	editRoot, editHome := stageProject(t, sourceRoot, "examples/invoice-grid")
	rewriteGate3File(t, editRoot, "src/web/web.can", "action save_invoice\n    post \"/invoices/save\"", "action save_invoice\n    post \"/invoices/save-v2\"")
	if status, out, diag := gate5Canlc(t, ctx, canlc, editHome, "assert", editRoot); status != 0 || diag != "" {
		t.Fatalf("edited grid assert: %d %s %s", status, out, diag)
	}
	_, editDir := gate5Build(t, ctx, canlc, editHome, editRoot, "--target", "browser")
	edited := gate5Bundle(t, ctx, sidecar, filepath.Join(editDir, "browser.ts"), t.TempDir())
	if !bytes.Contains(edited, []byte("/invoices/save-v2")) || bytes.Contains(edited, []byte("/invoices/save\"")) {
		t.Fatal("edited bundle does not carry exactly the new save path")
	}

	// Grid draft-field renames diagnose their stale uses inside the grid.
	wireRoot, wireHome := stageProject(t, sourceRoot, "examples/invoice-grid")
	rewriteGate3File(t, wireRoot, "src/records/records.can", "record draft\n    str invoice_id\n    int base_revision\n    str customer\n", "record draft\n    str invoice_id\n    int base_revision\n    str customer_name\n")
	status, out, diag = gate5Canlc(t, ctx, canlc, wireHome, "assert", wireRoot)
	combined = out + "\n" + diag
	if status == 0 || !strings.Contains(combined, "customer") {
		t.Fatalf("draft rename assert: %d %.800s, want a diagnostic naming customer", status, combined)
	}

	// Grid wire-field renames pass grid-only assert: every construction site
	// is positional, so no stale use names the field. This pins L-wire (the
	// cross-target driver test owns wire agreement; grid assert alone does not
	// catch wire drift) so a future checked wire linkage must update this leg.
	posRoot, posHome := stageProject(t, sourceRoot, "examples/invoice-grid")
	rewriteGate3File(t, posRoot, "src/records/records.can", "record invoice_json_wire\n    str session_token\n    str operation_id\n    str invoice_id\n    int revision\n    str customer\n", "record invoice_json_wire\n    str session_token\n    str operation_id\n    str invoice_id\n    int revision\n    str customer_name\n")
	if status, out, diag := gate5Canlc(t, ctx, canlc, posHome, "assert", posRoot); status != 0 || diag != "" {
		t.Fatalf("wire rename assert: %d %s %s, want the positional pass-through (L-wire)", status, out, diag)
	}

	// Server action renames pass unlinked: the declaration is the only
	// reference, the served spelling keeps answering, and the build stays
	// green. This pins L-unlink (declaration-to-serve gap) so the future
	// checked linkage must update this leg.
	linkRoot, linkHome := stageApplication(t, ctx, toolchain, sourceRoot, "invoice")
	rewriteGate3File(t, linkRoot, "src/web/web.can", "provides [save_invoice, save_invoice_form,", "provides [save_invoice_v2, save_invoice_form,")
	rewriteGate3File(t, linkRoot, "src/web/web.can", "action save_invoice\n", "action save_invoice_v2\n")
	if status, out, diag := gate5Canlc(t, ctx, canlc, linkHome, "assert", linkRoot); status != 0 || diag != "" {
		t.Fatalf("renamed server assert: %d %s %s", status, out, diag)
	}
	_, linkDir := gate5Build(t, ctx, canlc, linkHome, linkRoot)
	linkDB := filepath.Join(linkHome, "link.sqlite")
	setup := invoiceDriver(t, ctx, toolchain, linkHome, driver, "setup", linkDB, filepath.Join(linkRoot, "schema.sql"))
	var setupReport struct {
		Tables int `json:"tables"`
	}
	if err := json.Unmarshal(setup, &setupReport); err != nil || setupReport.Tables != 5 {
		t.Fatalf("invalid link setup report %v %s", err, string(setup))
	}
	invoiceDriver(t, ctx, toolchain, linkHome, driver, "seed", linkDB)
	linkSnapshot := snapshotCredential(t, linkHome, "INVOICE_DB", linkDB)
	linkBase, stopLink := serveApplication(t, ctx, toolchain, linkHome, filepath.Join(linkDir, "entry.ts"), linkSnapshot, gate5APIStatic+2, "/health")
	defer stopLink()
	client := &http.Client{Timeout: 10 * time.Second}
	request, err := http.NewRequest("POST", linkBase+"/invoices/save", strings.NewReader("not json"))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(&http.Cookie{Name: "session", Value: "tok-alice"})
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode != 400 || string(payload) != "Bad Request" {
		t.Fatalf("renamed server save route: %d %q, want the served spelling alive", response.StatusCode, payload)
	}
	t.Log("gate5 static: L-route live on the API, grid route edit rebundles, draft rename diagnoses, wire rename passes (L-wire), server rename passes unlinked (L-unlink)")
}

func mustMarshalJSON(t *testing.T, value any) string {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}
