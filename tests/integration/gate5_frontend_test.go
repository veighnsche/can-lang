// UP23 served browser/runtime and invoice matrix. Three compiler-built
// browser programs (the Can-authored invoice grid, the query-selected
// browser-conformance fixture, and the minimal empty app) each pair with
// the real invoice server through --browser-manifest, and the committed
// Playwright harnesses drive the application-served pages and
// content-addressed assets (/invoice-grid, /invoices/form,
// /__can/assets/...) in the two required named browsers, Chromium and
// WebKit, with a disposable database per leg. The Go test owns the
// toolchain, builds, pairing, origin bytes and database cross-checks;
// the harnesses own DOM, focus, announcements and network bytes.
//
// No test-authored page, bundle, boot entry or engine shim stands on any
// claimed path: the server splices the exact report-selected paired
// script tag, serves every asset byte, and owns the CSP. The obsolete
// Gate 5 origin proxy (captured-load rewrite, after-commit response
// faults) and the static contract-edit legs are gone; UP21 supplies the
// replacement edits. After-commit truncation/garbage now travel through
// harness route interception that forwards first, so the server still
// commits before the browser reads corrupt bytes.
//
// Known divergences pinned here rather than hidden (see the per-leg
// asserts and the reports' limitations):
//
//   - no same-origin 302 is producible under WebKit interception, so the
//     invoice redirect leg records L-redirect-webkit while Chromium
//     qualifies the engine-independent guard branch end to end.
//
// The grid qualifies with no limitations: only Enter dispatches its
// save handler, the pending "saving..." indication paints mid-flight,
// and an ignored mid-flight press renders nothing, so the outcome
// render stays the single next render on exactly one grid tree.
package integration

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/distribution"
)

const (
	gate5PortGridChromium  = 18641
	gate5PortGridWebkit    = 18642
	gate5PortConfChromium  = 18643
	gate5PortConfWebkit    = 18644
	gate5PortEmptyChromium = 18645
	gate5PortEmptyWebkit   = 18646
	gate5PortInvChromium   = 18647
	gate5PortInvWebkit     = 18648

	gate5SeedDetails = `Acme <em>&" 'coop'"`
	gate5SeedSeats   = "2"

	gate5EmptyMain = `package app
    provides []
    uses []
fn void main
    emits []
    asserts
        empty: => ok
    ok
`
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
		bundle, err := harnessBundle(t, ctx, archive)
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
	bundle, err := harnessBundle(t, ctx, archive)
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

type gate5BuildReport struct {
	BuildID   string
	Directory string
	Roots     int
}

// gate5Build runs one toolchain build and verifies its assertion verdict:
// browser-entry projects cannot run the standalone assert entry check, so
// the build's own verified assertions carry the real-can bar.
func gate5Build(t *testing.T, ctx context.Context, canlc, home, root string, extra ...string) gate5BuildReport {
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
		Assert    *struct {
			Roots    int      `json:"roots"`
			Passed   int      `json:"passed"`
			Failed   int      `json:"failed"`
			Evidence []string `json:"evidence"`
		} `json:"assertions"`
	}
	if err := json.Unmarshal([]byte(out), &report); err != nil || report.BuildID == "" || report.Directory == "" {
		t.Fatalf("invalid build report %v %s", err, out)
	}
	if report.Assert == nil || report.Assert.Failed != 0 || report.Assert.Passed != report.Assert.Roots || report.Assert.Roots == 0 {
		t.Fatalf("build assertions unverified: %+v", report.Assert)
	}
	real := false
	for _, evidence := range report.Assert.Evidence {
		if evidence == "real-can" {
			real = true
			break
		}
	}
	if !real {
		t.Fatalf("build asserts nothing real: %+v", report.Assert.Evidence)
	}
	return gate5BuildReport{BuildID: report.BuildID, Directory: report.Directory, Roots: report.Assert.Roots}
}

// gate5Pairing is one verified server/browser pairing: the server build
// identity plus the browser section of its build report.
type gate5Pairing struct {
	BuildID        string
	Directory      string
	BrowserBuildID string
	Entry          string
	Table          string
	Files          map[string]string
}

// gate5PairBuild builds the server project against one browser manifest and
// verifies the pairing report selects the manifest's browser build.
func gate5PairBuild(t *testing.T, ctx context.Context, canlc, home, root, manifest string) gate5Pairing {
	t.Helper()
	manifestRaw, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var manifestDecoded struct {
		BrowserBuildID string `json:"browserBuildId"`
	}
	if err := json.Unmarshal(manifestRaw, &manifestDecoded); err != nil || manifestDecoded.BrowserBuildID == "" {
		t.Fatalf("browser manifest lacks its build identity in %s", manifestRaw)
	}
	status, out, diag := gate5Canlc(t, ctx, canlc, home, "build", "--browser-manifest", manifest, root)
	if status != 0 {
		t.Fatalf("paired build: %d %s %s", status, out, diag)
	}
	var report struct {
		BuildID   string `json:"buildID"`
		Directory string `json:"directory"`
		Browser   *struct {
			BrowserBuildID string `json:"browserBuildId"`
			Entry          string `json:"entry"`
			Table          string `json:"table"`
			Files          []struct {
				Path  string `json:"path"`
				Route string `json:"route"`
			} `json:"files"`
		} `json:"browser"`
	}
	if err := json.Unmarshal([]byte(out), &report); err != nil || report.Browser == nil {
		t.Fatalf("paired report lacks its browser section: %v %s", err, out)
	}
	if report.Browser.BrowserBuildID != manifestDecoded.BrowserBuildID {
		t.Fatalf("paired browser %s, want %s", report.Browser.BrowserBuildID, manifestDecoded.BrowserBuildID)
	}
	if !strings.HasPrefix(report.Browser.Entry, "/__can/assets/") || !strings.HasSuffix(report.Browser.Entry, ".js") || strings.HasSuffix(report.Browser.Entry, ".js.map") {
		t.Fatalf("paired entry %q is not a digest script route", report.Browser.Entry)
	}
	if !strings.HasPrefix(report.Browser.Table, "/__can/assets/") || !strings.HasSuffix(report.Browser.Table, ".json") {
		t.Fatalf("paired table %q is not a digest route", report.Browser.Table)
	}
	files := map[string]string{}
	for _, file := range report.Browser.Files {
		if !strings.HasPrefix(file.Route, "/__can/assets/") {
			t.Fatalf("paired file %s route %q is not content addressed", file.Path, file.Route)
		}
		files[file.Path] = file.Route
	}
	return gate5Pairing{
		BuildID:        report.BuildID,
		Directory:      report.Directory,
		BrowserBuildID: report.Browser.BrowserBuildID,
		Entry:          report.Browser.Entry,
		Table:          report.Browser.Table,
		Files:          files,
	}
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

func mustSourceRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// gate5StageFixture copies the committed browser-conformance fixture to a
// disposable root the browser build can emit into.
func gate5StageFixture(t *testing.T, sourceRoot string) (root, home string) {
	t.Helper()
	var err error
	root, err = filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	from := filepath.Join(sourceRoot, "tests/integration/testdata/browser-conformance")
	for _, name := range []string{"can.project.json", "can.errors.json"} {
		data, err := os.ReadFile(filepath.Join(from, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	copyDir(t, filepath.Join(from, "src"), filepath.Join(root, "src"))
	home, err = filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return root, home
}

// gate5StageEmpty writes the minimal generated browser program: a
// dependency-free zero-argument main proving the degenerate program boots
// from served bytes.
func gate5StageEmpty(t *testing.T) (root, home string) {
	t.Helper()
	var err error
	root, err = filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	write := func(name, text string) {
		t.Helper()
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("can.project.json", `{"source_root":"src","error_registry":"can.errors.json"}`)
	write("can.errors.json", `{"active":[],"retired":[]}`)
	write("src/main.can", gate5EmptyMain)
	home, err = filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return root, home
}

func gate5BrowserProbe(t *testing.T, ctx context.Context, nodePath, browserDir, name string) string {
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
		t.Fatalf("required browser %s did not launch: %s", name, first)
	}
	version := strings.TrimSpace(string(result))
	if version == "" {
		t.Fatalf("required browser %s reported no version", name)
	}
	return version
}

type gate5Check struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
}

type gate5Limitation struct {
	ID     string `json:"id"`
	Detail string `json:"detail"`
}

type gate5Request struct {
	Method string `json:"method"`
	URL    string `json:"url"`
	Status int    `json:"status"`
}

type gate5LedgerEntry struct {
	Method       string `json:"method"`
	URL          string `json:"url"`
	Status       int    `json:"status"`
	RequestBody  string `json:"requestBody"`
	ResponseBody string `json:"responseBody"`
}

type gate5Occurrence struct {
	Kind   string `json:"kind"`
	Phase  string `json:"phase"`
	Effect string `json:"effect"`
	Reason string `json:"reason"`
	Header string `json:"header"`
}

// gate5Report decodes every UP23 harness report: absent sections stay empty.
type gate5Report struct {
	Browser       string             `json:"browser"`
	Version       string             `json:"version"`
	UserAgent     string             `json:"userAgent"`
	Base          string             `json:"base"`
	Passed        bool               `json:"passed"`
	Checks        []gate5Check       `json:"checks"`
	Limitations   []gate5Limitation  `json:"limitations"`
	Requests      []gate5Request     `json:"requests"`
	Ledger        []gate5LedgerEntry `json:"ledger"`
	Aborted       []string           `json:"aborted"`
	PageErrors    []string           `json:"pageerrors"`
	ConsoleErrors []string           `json:"consoleErrors"`
	Occurrences   []gate5Occurrence  `json:"occurrences"`
}

// gate5ReadReport parses one harness report and asserts the shared
// identities: the named engine, every check green, no page fault, and no
// request anywhere but loopback.
func gate5ReadReport(t *testing.T, suite, engine string, raw []byte, wantChecks int, wantUA bool) gate5Report {
	t.Helper()
	var report gate5Report
	if err := json.Unmarshal(raw, &report); err != nil || !report.Passed || len(report.Checks) != wantChecks {
		t.Fatalf("%s %s invalid report %v %s", suite, engine, err, raw)
	}
	for _, entry := range report.Checks {
		if !entry.Passed {
			t.Fatalf("%s %s check %s failed: %s", suite, engine, entry.Name, entry.Detail)
		}
	}
	if report.Browser != engine || report.Version == "" || (wantUA && report.UserAgent == "") {
		t.Fatalf("%s %s report identity %+v", suite, engine, report)
	}
	if len(report.Aborted) != 0 || len(report.PageErrors) != 0 {
		t.Fatalf("%s %s aborted=%v pageerrors=%v", suite, engine, report.Aborted, report.PageErrors)
	}
	for _, entry := range report.Requests {
		parsed, parseErr := url.Parse(entry.URL)
		if parseErr != nil || (parsed.Hostname() != "127.0.0.1" && parsed.Hostname() != "localhost") {
			t.Fatalf("%s %s left loopback: %s", suite, engine, entry.URL)
		}
	}
	return report
}

// gate5RunHarness executes one committed Playwright harness; argv is the full
// command line with the output directory already spliced in.
func gate5RunHarness(t *testing.T, ctx context.Context, nodePath, browserDir, suite, engine string, argv ...string) {
	t.Helper()
	cmd := exec.CommandContext(ctx, nodePath, argv...)
	cmd.Dir = browserDir
	cmd.Env = []string{"PATH=" + filepath.Dir(nodePath) + ":/usr/bin:/bin", "HOME=" + os.Getenv("HOME")}
	result, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s harness: %v %s", suite, engine, err, result)
	}
}

// gate5Evidence archives one leg's machine-readable report and screenshot
// for UP24/25 when CAN_BROWSER_EVIDENCE_DIR points at an evidence root.
func gate5Evidence(t *testing.T, outdir, name string) {
	t.Helper()
	evidence := os.Getenv("CAN_BROWSER_EVIDENCE_DIR")
	if evidence == "" {
		return
	}
	dest := filepath.Join(evidence, name)
	if err := os.MkdirAll(dest, 0700); err != nil {
		t.Fatal(err)
	}
	copyEvidenceFile(t, filepath.Join(outdir, "report.json"), filepath.Join(dest, "report.json"))
	copyEvidenceFile(t, filepath.Join(outdir, "screenshot.png"), filepath.Join(dest, "screenshot.png"))
}

// gate5Screenshot requires the leg's proof screenshot to exist and be
// non-empty.
func gate5Screenshot(t *testing.T, suite, engine, outdir string) {
	t.Helper()
	shot, err := os.Stat(filepath.Join(outdir, "screenshot.png"))
	if err != nil || shot.Size() == 0 {
		t.Fatalf("%s %s missing browser screenshot", suite, engine)
	}
}

// gate5ServedPairing inspects the application's own served bytes for one
// pairing: the page carries exactly the report-selected paired script tag,
// the script, map and diagnostic table routes serve verified bytes, the
// served CSP scopes scripts and connections to self, and non-HTML answers
// stay untouched.
func gate5ServedPairing(t *testing.T, suite, base, shellPath, session string, pairing gate5Pairing) {
	t.Helper()
	mapRoute := pairing.Files["browser/browser.js.map"]
	if mapRoute == "" || !strings.HasSuffix(mapRoute, ".js.map") {
		t.Fatalf("%s pairing lacks its map route in %+v", suite, pairing.Files)
	}
	status, shell, headers := invoiceGet(t, base, shellPath, session)
	if status != 200 {
		t.Fatalf("%s shell %s: %d, want 200", suite, shellPath, status)
	}
	csp := headers.Get("Content-Security-Policy")
	if !strings.Contains(csp, "script-src 'self'") || !strings.Contains(csp, "connect-src 'self'") {
		t.Fatalf("%s shell CSP %q scopes beyond self", suite, csp)
	}
	want := `<script type="module" src="` + pairing.Entry + `"></script>`
	if strings.Count(shell, want) != 1 {
		t.Fatalf("%s shell carries the paired entry %d times, want exactly once: %.500s", suite, strings.Count(shell, want), shell)
	}
	status, script, headers := invoiceGet(t, base, pairing.Entry, "")
	if status != 200 || !strings.Contains(headers.Get("Content-Type"), "text/javascript") {
		t.Fatalf("%s paired entry: %d %q", suite, status, headers.Get("Content-Type"))
	}
	if !strings.Contains(script, "$canBrowserMain") {
		t.Fatal("paired entry lacks the Can browser entry")
	}
	for _, banned := range gate5ServerModules {
		if strings.Contains(script, banned) {
			t.Fatalf("served script reaches server-only %s", banned)
		}
	}
	status, mapBody, _ := invoiceGet(t, base, mapRoute, "")
	if status != 200 {
		t.Fatalf("%s paired map: %d", suite, status)
	}
	var parsedMap struct {
		Version int      `json:"version"`
		Sources []string `json:"sources"`
	}
	if err := json.Unmarshal([]byte(mapBody), &parsedMap); err != nil || parsedMap.Version != 3 || len(parsedMap.Sources) == 0 {
		t.Fatalf("%s paired map invalid: %.200s", suite, mapBody)
	}
	status, tableBody, _ := invoiceGet(t, base, pairing.Table, "")
	if status != 200 {
		t.Fatalf("%s paired table: %d", suite, status)
	}
	var table struct {
		SchemaVersion int    `json:"schemaVersion"`
		Kind          string `json:"kind"`
	}
	if err := json.Unmarshal([]byte(tableBody), &table); err != nil || table.SchemaVersion != 1 || table.Kind != "can.diagnostic-table" {
		t.Fatalf("%s paired table invalid: %.200s", suite, tableBody)
	}
	status, probe, _ := invoiceGet(t, base, "/health", "")
	if status != 200 || probe != "ok" || strings.Contains(probe, "<script") {
		t.Fatalf("%s health: %d %q, want untouched bytes", suite, status, probe)
	}
	t.Logf("%s pairing: shell carries %s once, script %d bytes, map %d sources, sealed table, health untouched",
		suite, pairing.Entry, len(script), len(parsedMap.Sources))
}

// gate5SeedDB provisions one disposable seeded database for a browser leg.
func gate5SeedDB(t *testing.T, ctx context.Context, toolchain, home, driver, serverRoot string) string {
	t.Helper()
	db, err := filepath.EvalSymlinks(filepath.Join(home, "leg.sqlite"))
	if err != nil {
		resolved, resolveErr := filepath.EvalSymlinks(home)
		if resolveErr != nil {
			t.Fatal(resolveErr)
		}
		db = filepath.Join(resolved, "leg.sqlite")
	}
	setup := invoiceDriver(t, ctx, toolchain, home, driver, "setup", db, filepath.Join(serverRoot, "schema.sql"))
	var setupReport struct {
		Tables int `json:"tables"`
	}
	if err := json.Unmarshal(setup, &setupReport); err != nil || setupReport.Tables != 5 {
		t.Fatalf("invalid setup report %v %s", err, string(setup))
	}
	seed := invoiceDriver(t, ctx, toolchain, home, driver, "seed", db)
	var seedReport struct {
		Sessions int `json:"sessions"`
		Invoices int `json:"invoices"`
		Lines    int `json:"lines"`
	}
	if err := json.Unmarshal(seed, &seedReport); err != nil || seedReport.Sessions != 3 || seedReport.Invoices != 2 || seedReport.Lines != 2 {
		t.Fatalf("invalid seed report %v %s", err, string(seed))
	}
	return db
}

// gate5RequireSeedLines asserts the invoice keeps exactly its two hostile
// seeded lines, escaped at rest and unmodified.
func gate5RequireSeedLines(t *testing.T, suite, engine string, row invoiceRow) {
	t.Helper()
	if len(row.Lines) != 2 {
		t.Fatalf("%s %s lines %+v, want the two seeded lines", suite, engine, row.Lines)
	}
	first, second := row.Lines[0], row.Lines[1]
	if first.Key != "k1" || first.ID != "sku-1 <b>" || first.Quantity != "2" || first.Price != "1999" || first.Position != "0" {
		t.Fatalf("%s %s first line %+v", suite, engine, first)
	}
	if second.Key != "k2" || second.ID != "plain" || second.Quantity != "1" || second.Price != "500" || second.Position != "1" {
		t.Fatalf("%s %s second line %+v", suite, engine, second)
	}
}

// gate5RequireReplay asserts the replay ledger holds exactly the committed
// revisions with well-formed rows.
func gate5RequireReplay(t *testing.T, suite, engine string, store invoiceStore, wantRevs ...string) map[string]bool {
	t.Helper()
	if len(store.Replay) != len(wantRevs) {
		t.Fatalf("%s %s replay rows %+v, want %d", suite, engine, store.Replay, len(wantRevs))
	}
	ops := map[string]bool{}
	seen := map[string]bool{}
	for _, row := range store.Replay {
		if len(row.Digest) != 64 || row.Result == "" || row.Tenant != "1" || row.InvoiceID != "7" {
			t.Fatalf("%s %s replay row %+v", suite, engine, row)
		}
		if ops[row.OperationID] {
			t.Fatalf("%s %s duplicated replay for %s", suite, engine, row.OperationID)
		}
		ops[row.OperationID] = true
		seen[row.Revision] = true
	}
	for _, rev := range wantRevs {
		if !seen[rev] {
			t.Fatalf("%s %s replay lacks revision %s in %+v", suite, engine, rev, store.Replay)
		}
	}
	return ops
}

// gate5GridCrossCheck proves exactly-once effects for the grid run: every
// wire operation classified by its response cases, the committed ops
// matching the replay rows exactly, and the rejected, stale, denied,
// busy and corrupted attempts leaving no row.
func gate5GridCrossCheck(t *testing.T, engine string, report gate5Report, store invoiceStore) {
	t.Helper()
	const api = "/api/tenants/1/invoices/7"
	gets := map[string]int{}
	deniedLoads := 0
	opCases := map[string]map[string]bool{}
	opSavedReal := map[string]bool{}
	opSavedSynthetic := map[string]bool{}
	unreadable := 0
	for _, call := range report.Ledger {
		if !strings.HasPrefix(call.URL, report.Base) {
			t.Fatalf("grid %s ledger holds a non-origin call %s", engine, call.URL)
		}
		invoice := strings.TrimPrefix(call.URL, report.Base)
		if invoice != api && invoice != "/api/tenants/2/invoices/8" {
			t.Fatalf("grid %s ledger holds a non-invoice call %s", engine, call.URL)
		}
		if call.Method == "GET" {
			var outcome struct {
				Case string `json:"case"`
			}
			if err := json.Unmarshal([]byte(call.ResponseBody), &outcome); err != nil || outcome.Case == "" {
				t.Fatalf("grid %s ledger holds an unreadable load body %q", engine, call.ResponseBody)
			}
			if invoice != api {
				if outcome.Case != "invoice_contract::grid_load_forbidden" {
					t.Fatalf("grid %s foreign-invoice load case %s, want the denial", engine, outcome.Case)
				}
				deniedLoads++
				continue
			}
			gets[outcome.Case]++
			continue
		}
		if invoice != api {
			t.Fatalf("grid %s ledger holds a foreign-invoice %s", engine, call.URL)
		}
		if call.Method != "POST" {
			t.Fatalf("grid %s ledger holds a %s call", engine, call.Method)
		}
		var body struct {
			OperationID string `json:"operation_id"`
		}
		if err := json.Unmarshal([]byte(call.RequestBody), &body); err != nil || body.OperationID == "" {
			t.Fatalf("grid %s ledger holds an unreadable save body %q", engine, call.RequestBody)
		}
		if opCases[body.OperationID] == nil {
			opCases[body.OperationID] = map[string]bool{}
		}
		var outcome struct {
			Case string `json:"case"`
		}
		if err := json.Unmarshal([]byte(call.ResponseBody), &outcome); err != nil || outcome.Case == "" {
			unreadable++
			continue
		}
		opCases[body.OperationID][outcome.Case] = true
		if outcome.Case == "invoice_contract::grid_saved" {
			if strings.Contains(call.ResponseBody, `"revision":"99"`) {
				opSavedSynthetic[body.OperationID] = true
			} else {
				opSavedReal[body.OperationID] = true
			}
		}
	}
	if gets["invoice_contract::grid_loaded"] != 8 || gets["invoice_contract::grid_load_unavailable"] != 1 || deniedLoads != 1 || len(report.Ledger) != 32 {
		t.Fatalf("grid %s loads %+v with %d foreign denials over %d calls, want 8 loaded, 1 unavailable, 1 denial, 32 calls", engine, gets, deniedLoads, len(report.Ledger))
	}
	terminal := map[string]int{}
	for _, cases := range opCases {
		for _, leaf := range []string{"invoice_contract::grid_invalid", "invoice_contract::grid_conflict", "invoice_contract::grid_forbidden", "invoice_contract::grid_unavailable"} {
			if cases[leaf] {
				terminal[leaf]++
			}
		}
	}
	for _, leaf := range []string{"invoice_contract::grid_invalid", "invoice_contract::grid_conflict", "invoice_contract::grid_forbidden", "invoice_contract::grid_unavailable"} {
		if terminal[leaf] != 1 {
			t.Fatalf("grid %s terminal case %s seen %d times, want 1", engine, leaf, terminal[leaf])
		}
	}
	if unreadable != 2 {
		t.Fatalf("grid %s unreadable saves = %d, want the truncate and garbage attempts", engine, unreadable)
	}
	if len(opSavedSynthetic) != 1 {
		t.Fatalf("grid %s synthetic saves = %d, want the one rev-99 probe fulfill", engine, len(opSavedSynthetic))
	}
	replayOps := gate5RequireReplay(t, "grid", engine, store, "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12", "13", "14", "15", "16")
	if !replayOps["op-up23-race"] {
		t.Fatalf("grid %s replay lacks the racing commit", engine)
	}
	for op := range opSavedReal {
		if !replayOps[op] {
			t.Fatalf("grid %s saved op %s left no replay row", engine, op)
		}
	}
	for op, cases := range opCases {
		if opSavedReal[op] {
			continue
		}
		if replayOps[op] {
			t.Fatalf("grid %s unsaved op %s left a replay row (cases %v)", engine, op, cases)
		}
	}
	if len(opSavedReal) != 15 || len(replayOps) != 15 {
		t.Fatalf("grid %s saved ops = %d, replay rows = %d, want 15 and 15", engine, len(opSavedReal), len(replayOps))
	}
	row := requireInvoice(t, store, "7", "16", gate5SeedSeats, gate5SeedDetails)
	if len(row.Lines) != 1 {
		t.Fatalf("grid %s lines %+v, want the single adopted line", engine, row.Lines)
	}
	line := row.Lines[0]
	if line.Key != "k1" || line.ID != "sku-1" || line.Quantity != "2" || line.Price != "2800" || line.Position != "0" {
		t.Fatalf("grid %s final line %+v", engine, line)
	}
	requireInvoice(t, store, "8", "1", "1", "Globex")
}

// gate5OccurrenceTuple is the pinned guard-evidence shape per occurrence.
type gate5OccurrenceTuple struct {
	Kind   string
	Phase  string
	Effect string
	Reason string
	Header string
}

// gate5InvoiceOccurrences asserts the exact guard-occurrence sequence: the
// missing-target request/response pair, the control-header and OOB
// rejections, and the redirect rejection on Chromium.
func gate5InvoiceOccurrences(t *testing.T, engine string, report gate5Report) {
	t.Helper()
	want := []gate5OccurrenceTuple{
		{"action::missing_target", "request", "none", "target_absent", ""},
		{"action::missing_target", "response", "uncertain", "", ""},
		{"action::protocol", "response", "uncertain", "control_header", "redirect"},
		{"action::protocol", "swap", "uncertain", "task_shape", ""},
	}
	if engine == "chromium" {
		want = append(want, gate5OccurrenceTuple{"action::protocol", "response", "uncertain", "redirect", ""})
	}
	if len(report.Occurrences) != len(want) {
		t.Fatalf("invoice %s occurrences %+v, want %d", engine, report.Occurrences, len(want))
	}
	for i, tuple := range want {
		got := report.Occurrences[i]
		if got.Kind != tuple.Kind || got.Phase != tuple.Phase || got.Effect != tuple.Effect || got.Header != tuple.Header {
			t.Fatalf("invoice %s occurrence %d = %+v, want %+v", engine, i, got, tuple)
		}
		if i == 1 {
			if got.Reason != "target_absent" && got.Reason != "target_detached" {
				t.Fatalf("invoice %s occurrence %d reason %q, want target_absent or target_detached", engine, i, got.Reason)
			}
			continue
		}
		if got.Reason != tuple.Reason {
			t.Fatalf("invoice %s occurrence %d = %+v, want %+v", engine, i, got, tuple)
		}
	}
	if engine == "webkit" {
		if len(report.Limitations) != 1 || report.Limitations[0].ID != "L-redirect-webkit" {
			t.Fatalf("invoice webkit limitations %+v, want exactly L-redirect-webkit", report.Limitations)
		}
	} else if len(report.Limitations) != 0 {
		t.Fatalf("invoice chromium limitations %+v, want none", report.Limitations)
	}
}

// gate5Matrix shares one toolchain, staged projects and verified pairings
// across the four served suites.
type gate5Matrix struct {
	ctx        context.Context
	toolchain  string
	canlc      string
	nodePath   string
	sourceRoot string
	browserDir string
	driver     string
	serverRoot string
	serverHome string
	pairings   map[string]gate5Pairing
	versions   map[string]string
}

func (m *gate5Matrix) serve(t *testing.T, pairing gate5Pairing, port int) (base string, db string, stop func()) {
	t.Helper()
	home := t.TempDir()
	db = gate5SeedDB(t, m.ctx, m.toolchain, home, m.driver, m.serverRoot)
	base, stop = serveInvoice(t, m.ctx, m.toolchain, home, filepath.Join(pairing.Directory, "entry.ts"), db, port, "")
	return base, db, stop
}

func (m *gate5Matrix) gridLeg(t *testing.T, engine string, port int) {
	t.Helper()
	pairing := m.pairings["grid"]
	base, db, stop := m.serve(t, pairing, port)
	gate5ServedPairing(t, "grid", base, "/invoice-grid?tenant=1&invoice=7", "", pairing)
	outdir := t.TempDir()
	gate5RunHarness(t, m.ctx, m.nodePath, m.browserDir, "grid", engine,
		"grid.mjs", engine, base, outdir, db, pairing.Entry)
	stop()
	raw, err := os.ReadFile(filepath.Join(outdir, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	report := gate5ReadReport(t, "grid", engine, raw, 32, true)
	if len(report.Limitations) != 0 {
		t.Fatalf("grid %s limitations %+v, want none", engine, report.Limitations)
	}
	gate5Screenshot(t, "grid", engine, outdir)
	store := inspectInvoice(t, m.ctx, m.toolchain, t.TempDir(), m.driver, db)
	gate5GridCrossCheck(t, engine, report, store)
	gate5Evidence(t, outdir, "grid-"+engine)
	t.Logf("gate5 grid %s %s: 32 checks, %d loopback requests, %d invoice calls, rev 16 committed with 15 replay rows",
		engine, report.Version, len(report.Requests), len(report.Ledger))
}

func (m *gate5Matrix) conformanceLeg(t *testing.T, engine string, port int) {
	t.Helper()
	pairing := m.pairings["fixture"]
	base, db, stop := m.serve(t, pairing, port)
	gate5ServedPairing(t, "conformance", base, "/invoice-grid?scenario=equality", "", pairing)
	outdir := t.TempDir()
	gate5RunHarness(t, m.ctx, m.nodePath, m.browserDir, "conformance", engine,
		"conformance.mjs", engine, base, outdir, pairing.Entry)
	stop()
	raw, err := os.ReadFile(filepath.Join(outdir, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	report := gate5ReadReport(t, "conformance", engine, raw, 17, true)
	if len(report.Limitations) != 0 {
		t.Fatalf("conformance %s limitations %+v, want none", engine, report.Limitations)
	}
	for _, entry := range report.Requests {
		if strings.Contains(entry.URL, "/api/") {
			t.Fatalf("conformance %s fixture called an API: %s", engine, entry.URL)
		}
	}
	gate5Screenshot(t, "conformance", engine, outdir)
	store := inspectInvoice(t, m.ctx, m.toolchain, t.TempDir(), m.driver, db)
	row := requireInvoice(t, store, "7", "1", gate5SeedSeats, gate5SeedDetails)
	gate5RequireSeedLines(t, "conformance", engine, row)
	if len(store.Replay) != 0 {
		t.Fatalf("conformance %s replay rows %+v, want none", engine, store.Replay)
	}
	gate5Evidence(t, outdir, "conformance-"+engine)
	t.Logf("gate5 conformance %s %s: 17 checks, %d loopback requests, database untouched",
		engine, report.Version, len(report.Requests))
}

func (m *gate5Matrix) emptyLeg(t *testing.T, engine string, port int) {
	t.Helper()
	pairing := m.pairings["empty"]
	base, db, stop := m.serve(t, pairing, port)
	gate5ServedPairing(t, "empty", base, "/invoice-grid?tenant=1&invoice=7", "", pairing)
	mapRoute := pairing.Files["browser/browser.js.map"]
	outdir := t.TempDir()
	gate5RunHarness(t, m.ctx, m.nodePath, m.browserDir, "empty", engine,
		"empty.mjs", engine, base, outdir, pairing.Entry, mapRoute, pairing.Table)
	stop()
	raw, err := os.ReadFile(filepath.Join(outdir, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	report := gate5ReadReport(t, "empty", engine, raw, 4, true)
	if len(report.Limitations) != 0 {
		t.Fatalf("empty %s limitations %+v, want none", engine, report.Limitations)
	}
	for _, entry := range report.Requests {
		if strings.Contains(entry.URL, "/api/") || strings.Contains(entry.URL, "/tenants/") {
			t.Fatalf("empty %s called an API: %s", engine, entry.URL)
		}
	}
	gate5Screenshot(t, "empty", engine, outdir)
	store := inspectInvoice(t, m.ctx, m.toolchain, t.TempDir(), m.driver, db)
	row := requireInvoice(t, store, "7", "1", gate5SeedSeats, gate5SeedDetails)
	gate5RequireSeedLines(t, "empty", engine, row)
	if len(store.Replay) != 0 {
		t.Fatalf("empty %s replay rows %+v, want none", engine, store.Replay)
	}
	gate5Evidence(t, outdir, "empty-"+engine)
	t.Logf("gate5 empty %s %s: 4 checks, degenerate program booted from served bytes",
		engine, report.Version)
}

func (m *gate5Matrix) invoiceLeg(t *testing.T, engine string, port int) {
	t.Helper()
	pairing := m.pairings["grid"]
	base, db, stop := m.serve(t, pairing, port)
	status, form, _ := invoiceGet(t, base, "/invoices/form?tenant_id=1&invoice_id=7", "tok-alice")
	if status != 200 {
		t.Fatalf("invoice %s form page: %d, want 200", engine, status)
	}
	want := `<script type="module" src="` + pairing.Entry + `"></script>`
	if strings.Count(form, want) != 1 {
		t.Fatalf("invoice %s form carries the paired entry %d times, want exactly once", engine, strings.Count(form, want))
	}
	outdir := t.TempDir()
	gate5RunHarness(t, m.ctx, m.nodePath, m.browserDir, "invoice", engine,
		"invoice.mjs", base, outdir, db, engine, pairing.Entry)
	stop()
	raw, err := os.ReadFile(filepath.Join(outdir, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	report := gate5ReadReport(t, "invoice", engine, raw, 19, false)
	gate5InvoiceOccurrences(t, engine, report)
	gate5Screenshot(t, "invoice", engine, outdir)
	store := inspectInvoice(t, m.ctx, m.toolchain, t.TempDir(), m.driver, db)
	row := requireInvoice(t, store, "7", "5", gate5SeedSeats, "Guard Final")
	gate5RequireSeedLines(t, "invoice", engine, row)
	gate5RequireReplay(t, "invoice", engine, store, "2", "3", "4", "5")
	requireInvoice(t, store, "8", "1", "1", "Globex")
	gate5Evidence(t, outdir, "invoice-"+engine)
	t.Logf("gate5 invoice %s %s: 19 checks, %d guard occurrences, rev 5 committed with 4 replay rows",
		engine, report.Version, len(report.Occurrences))
}

func TestGate5ServedMatrix(t *testing.T) {
	t.Parallel()
	acquireHeavy(t)
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
	ctx, cancel := context.WithTimeout(context.Background(), 55*time.Minute)
	defer cancel()
	matrix := &gate5Matrix{
		ctx:        ctx,
		sourceRoot: sourceRoot,
		browserDir: browserDir,
		pairings:   map[string]gate5Pairing{},
		versions:   map[string]string{},
	}
	toolchain, canlc, _, installed := gate5Toolchain(t, ctx, sourceRoot, archive)
	matrix.toolchain = toolchain
	matrix.canlc = canlc
	matrix.nodePath = nodePath
	matrix.driver = filepath.Join(sourceRoot, "tests/integration/testdata/invoice/driver.ts")
	mode := "staged bundle"
	if installed {
		mode = "installed release"
	}
	home := t.TempDir()
	host := gate4RuntimeIdentity(t, ctx, canlc, home)
	archiveDigest := sha256.Sum256(mustReadFile(t, archive))
	t.Logf("gate5 host bound: %s on %s/%s bun %s rev %s (archive sha256 %s)",
		mode, host["platform"], host["architecture"], host["bun"], host["revision"][:12], hex.EncodeToString(archiveDigest[:])[:12])

	// Both named engines are required: probe before building so a
	// missing engine fails fast instead of after a long build.
	for _, engine := range []string{"chromium", "webkit"} {
		matrix.versions[engine] = gate5BrowserProbe(t, ctx, nodePath, browserDir, engine)
		t.Logf("gate5 required browser %s at %s", engine, matrix.versions[engine])
	}

	serverRoot, serverHome := stageApplication(t, ctx, toolchain, sourceRoot, "invoice")
	matrix.serverRoot = serverRoot
	matrix.serverHome = serverHome
	serverAssertions, serverReal := gate5Assert(t, ctx, canlc, serverHome, serverRoot, "invoice")
	gridRoot, gridHome := stageProject(t, sourceRoot, "examples/invoice-grid")
	fixtureRoot, fixtureHome := gate5StageFixture(t, sourceRoot)
	emptyRoot, emptyHome := gate5StageEmpty(t)

	browserBuilds := map[string]string{}
	browserRoots := map[string]int{}
	browserDirs := map[string]string{}
	for _, project := range []struct{ name, home, root string }{
		{"grid", gridHome, gridRoot},
		{"fixture", fixtureHome, fixtureRoot},
		{"empty", emptyHome, emptyRoot},
	} {
		first := gate5Build(t, ctx, canlc, project.home, project.root, "--target", "browser")
		second := gate5Build(t, ctx, canlc, project.home, project.root, "--target", "browser")
		if first.BuildID != second.BuildID {
			t.Fatalf("%s browser rebuild drifted: %s vs %s", project.name, first.BuildID, second.BuildID)
		}
		assertNoStrayEmit(t, project.root, second.Directory)
		gate5Asset(t, second.Directory)
		gate5ImportAudit(t, second.Directory)
		browserBuilds[project.name] = first.BuildID
		browserRoots[project.name] = first.Roots
		browserDirs[project.name] = second.Directory
		matrix.pairings[project.name] = gate5PairBuild(t, ctx, canlc, serverHome, serverRoot, filepath.Join(second.Directory, "browser", "manifest.json"))
		repeat := gate5PairBuild(t, ctx, canlc, serverHome, serverRoot, filepath.Join(second.Directory, "browser", "manifest.json"))
		if repeat.BuildID != matrix.pairings[project.name].BuildID || repeat.Entry != matrix.pairings[project.name].Entry {
			t.Fatalf("%s paired rebuild drifted: %s vs %s", project.name, repeat.BuildID, matrix.pairings[project.name].BuildID)
		}
	}
	assertNoStrayEmit(t, serverRoot, matrix.pairings["grid"].Directory)
	gate5NoDirectIO(t, browserDirs["grid"])
	t.Logf("gate5 builds: server %d assertions (%d real-can); grid %d roots browser %s paired %s; fixture %d roots browser %s paired %s; empty %d roots browser %s paired %s",
		serverAssertions, serverReal,
		browserRoots["grid"], browserBuilds["grid"][:12], matrix.pairings["grid"].BuildID[:12],
		browserRoots["fixture"], browserBuilds["fixture"][:12], matrix.pairings["fixture"].BuildID[:12],
		browserRoots["empty"], browserBuilds["empty"][:12], matrix.pairings["empty"].BuildID[:12])

	t.Run("grid", func(t *testing.T) {
		matrix.gridLeg(t, "chromium", gate5PortGridChromium)
		matrix.gridLeg(t, "webkit", gate5PortGridWebkit)
	})
	t.Run("conformance", func(t *testing.T) {
		matrix.conformanceLeg(t, "chromium", gate5PortConfChromium)
		matrix.conformanceLeg(t, "webkit", gate5PortConfWebkit)
	})
	t.Run("empty", func(t *testing.T) {
		matrix.emptyLeg(t, "chromium", gate5PortEmptyChromium)
		matrix.emptyLeg(t, "webkit", gate5PortEmptyWebkit)
	})
	t.Run("invoice", func(t *testing.T) {
		matrix.invoiceLeg(t, "chromium", gate5PortInvChromium)
		matrix.invoiceLeg(t, "webkit", gate5PortInvWebkit)
	})

	finalReport, err := json.MarshalIndent(map[string]any{
		"mode": mode, "host": host,
		"server":  map[string]any{"assertions": serverAssertions, "real_can": serverReal},
		"grid":    map[string]any{"roots": browserRoots["grid"], "browser": browserBuilds["grid"], "paired": matrix.pairings["grid"].BuildID, "entry": matrix.pairings["grid"].Entry},
		"fixture": map[string]any{"roots": browserRoots["fixture"], "browser": browserBuilds["fixture"], "paired": matrix.pairings["fixture"].BuildID, "entry": matrix.pairings["fixture"].Entry},
		"empty":   map[string]any{"roots": browserRoots["empty"], "browser": browserBuilds["empty"], "paired": matrix.pairings["empty"].BuildID, "entry": matrix.pairings["empty"].Entry},
		"matrix":  matrix.versions,
		"limitations": []string{
			"L-redirect-webkit: no same-origin 302 is producible under WebKit interception; the redirect guard branch is qualified on Chromium",
		},
	}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("gate5 report:\n%s", finalReport)
}

// TestGate5GridStatic pins the browser-independent Gate 5 contracts without
// a browser: the server project's argv entry rejects the browser target at
// the entry-shape gate, and the grid keeps its timer-free leak surface.
// The shim parity, load-path liveness and contract-edit legs are gone with
// the origin proxy; UP21 supplies the replacement edits.
func TestGate5GridStatic(t *testing.T) {
	t.Parallel()
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged gate 5 execution")
	}
	sourceRoot := mustSourceRoot(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	toolchain, canlc, _, _ := gate5Toolchain(t, ctx, sourceRoot, archive)

	serverRoot, serverHome := stageApplication(t, ctx, toolchain, sourceRoot, "invoice")
	status, out, diag := gate5Canlc(t, ctx, canlc, serverHome, "build", "--target", "browser", serverRoot)
	combined := out + "\n" + diag
	if status == 0 || !strings.Contains(combined, "browser entry must be") {
		t.Fatalf("server browser build: %d %.500s, want an entry-shape rejection", status, combined)
	}
	t.Logf("gate5 capability: server project rejected for --target browser: %s", strings.TrimSpace(combined))

	gridSources := filepath.Join(sourceRoot, "examples/invoice-grid/src")
	var timerUses []string
	err := filepath.WalkDir(gridSources, func(path string, entry os.DirEntry, err error) error {
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
	t.Log("gate5 static: capability closure rejects the server browser build, grid sets no timers")
}
