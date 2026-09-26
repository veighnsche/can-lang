// UP21 staged shared-contract mutation suite. Each row of the invoice
// contract-edit table runs here against real staged builds: the shared
// contract is edited once in a disposable copy, synced into both
// consumer vendors with both locks rewritten, then both projects are
// rebuilt (grid --target browser, server paired with the grid manifest)
// and the affected flow executes through the served application with
// real HTTP, SQLite and Chromium DOM observation. Check-level stale
// diagnostics and the negative battery live in
// compiler/internal/driver/invoice_grid_test.go; this file proves the
// repaired live behavior, stale-path rejection/new-path success with
// handler-entry evidence, pre-entry 400/413/415, manifest-mismatch
// rejection and that importing the contract registers no route. The old
// load spelling answers 405 (the path stays recognized by the kept POST
// save) while genuinely unknown and unmounted paths answer 404.
//
// Run with a staged toolchain and the pinned harness. The rows run in
// parallel (distinct ports, workspaces and SQLite files per row), so the
// whole table fits one go invocation; on a thermally constrained host,
// cap fan-out with -parallel:
//
//	CAN_BUN_ARCHIVE=/private/tmp/bun-dl/bun-darwin-aarch64.zip \
//	  go test ./tests/integration/ -run 'TestInvoiceContract' -count=1 -parallel 4
package integration

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	contractLivePortRoute      = 18751
	contractLivePortCapture    = 18752
	contractLivePortField      = 18753
	contractLivePortLeaf       = 18754
	contractLivePortStatus     = 18755
	contractLivePortLimit      = 18756
	contractLivePortBodyMode   = 18757
	contractLivePortPristine   = 18758
	contractLivePortNoRegister = 18759
)

type contractLiveWS struct {
	shared string
	server string
	grid   string
	home   string
}

// contractLiveStage copies the shared contract and both consumers into
// disposable temp dirs. Committed fixtures stay read-only.
func contractLiveStage(t *testing.T, sourceRoot string) contractLiveWS {
	t.Helper()
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ws := contractLiveWS{
		shared: filepath.Join(base, "shared"),
		server: filepath.Join(base, "server"),
		grid:   filepath.Join(base, "grid"),
	}
	copyDir(t, filepath.Join(sourceRoot, "shared/invoice-contract"), ws.shared)
	copyDir(t, filepath.Join(sourceRoot, "examples/invoice"), ws.server)
	copyDir(t, filepath.Join(sourceRoot, "examples/invoice-grid"), ws.grid)
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ws.home = home
	// The stdlib digest below must reproduce the committed billing
	// digest on pristine trees, or edited locks cannot be trusted.
	const committed = "16b137f9cfebe39b9617e4fec00f0aaa985055c80310ee0399de7694365620e5"
	for _, root := range []string{ws.server, ws.grid} {
		if digest := contractLiveVendorDigest(t, root); digest != committed {
			t.Fatalf("digest recomputed %s, want committed %s", digest, committed)
		}
	}
	return ws
}

func contractLiveRead(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func contractLiveWrite(t *testing.T, name, content string) {
	t.Helper()
	if err := os.WriteFile(name, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func contractLiveReplaceAll(t *testing.T, name, old, new string) int {
	t.Helper()
	content := contractLiveRead(t, name)
	count := strings.Count(content, old)
	if count == 0 {
		t.Fatalf("contract edit anchor %q missing from %s", old, name)
	}
	contractLiveWrite(t, name, strings.ReplaceAll(content, old, new))
	return count
}

func contractLiveReplaceOnce(t *testing.T, name, old, new string) {
	t.Helper()
	if count := contractLiveReplaceAll(t, name, old, new); count != 1 {
		t.Fatalf("contract edit anchor %q occurs %d times in %s, want 1", old, count, name)
	}
}

// contractLiveEditShared applies replacements to the shared copy once,
// syncs the edited file into both vendors, and rewrites both locks.
func contractLiveEditShared(t *testing.T, ws contractLiveWS, replacements [][2]string) string {
	t.Helper()
	shared := filepath.Join(ws.shared, "src/invoice_contract/invoice_contract.can")
	for _, pair := range replacements {
		contractLiveReplaceAll(t, shared, pair[0], pair[1])
	}
	edited, err := os.ReadFile(shared)
	if err != nil {
		t.Fatal(err)
	}
	for _, root := range []string{ws.server, ws.grid} {
		if err := os.WriteFile(filepath.Join(root, "vendor/billing/src/invoice_contract/invoice_contract.can"), edited, 0600); err != nil {
			t.Fatal(err)
		}
	}
	serverDigest := contractLiveRelock(t, ws.server)
	gridDigest := contractLiveRelock(t, ws.grid)
	if serverDigest != gridDigest {
		t.Fatalf("relock diverged: server %s grid %s", serverDigest, gridDigest)
	}
	return serverDigest
}

type contractLiveSource struct {
	Path  string
	Bytes []byte
}

// contractLiveDigest mirrors project.SourceDigest (P2's byte-exact,
// length-prefixed source tree format) with stdlib only: internal
// packages are not importable from tests/integration. Every stage
// validates it against the committed billing digest.
func contractLiveDigest(t *testing.T, files []contractLiveSource) string {
	t.Helper()
	ordered := append([]contractLiveSource(nil), files...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Path < ordered[j].Path })
	hash := sha256.New()
	hash.Write([]byte("can-source-tree-v1\x00"))
	var length [8]byte
	previous := ""
	for _, file := range ordered {
		if !strings.HasSuffix(file.Path, ".can") || file.Path == previous {
			t.Fatal("source digest requires unique .can paths")
		}
		previous = file.Path
		binary.BigEndian.PutUint64(length[:], uint64(len([]byte(file.Path))))
		hash.Write(length[:])
		hash.Write([]byte(file.Path))
		binary.BigEndian.PutUint64(length[:], uint64(len(file.Bytes)))
		hash.Write(length[:])
		hash.Write(file.Bytes)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func contractLiveVendorDigest(t *testing.T, root string) string {
	t.Helper()
	vendor := filepath.Join(root, "vendor/billing")
	manifestRaw, err := os.ReadFile(filepath.Join(vendor, "can.project.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		SourceRoot string `json:"source_root"`
	}
	if err := json.Unmarshal(manifestRaw, &manifest); err != nil || manifest.SourceRoot == "" {
		t.Fatalf("invalid vendor manifest %v %s", err, manifestRaw)
	}
	var sources []contractLiveSource
	walkRoot := filepath.Join(vendor, manifest.SourceRoot)
	if err := filepath.WalkDir(walkRoot, func(name string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(entry.Name(), ".can") {
			return err
		}
		rel, err := filepath.Rel(walkRoot, name)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		sources = append(sources, contractLiveSource{Path: path.Join(filepath.ToSlash(filepath.Dir(rel)), entry.Name()), Bytes: data})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return contractLiveDigest(t, sources)
}

func contractLiveRelock(t *testing.T, root string) string {
	t.Helper()
	digest := contractLiveVendorDigest(t, root)
	lockPath := filepath.Join(root, "can.lock.json")
	raw, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	var lock map[string]any
	if err := json.Unmarshal(raw, &lock); err != nil {
		t.Fatal(err)
	}
	projects, _ := lock["projects"].(map[string]any)
	entry, _ := projects["can.project.lineage/billing"].(map[string]any)
	if entry == nil {
		t.Fatal("lock lacks the billing instance")
	}
	entry["source_sha256"] = digest
	rewritten, err := json.Marshal(lock)
	if err != nil {
		t.Fatal(err)
	}
	contractLiveWrite(t, lockPath, string(rewritten))
	return digest
}

// contractLiveBuildGrid builds the staged grid for the browser target
// and returns its build identity, directory and manifest path. Builds
// execute every assertion, so a separate assert leg would only repeat
// the same supervised roots; the report counts are the evidence.
func contractLiveBuildGrid(t *testing.T, ctx context.Context, bundle string, ws contractLiveWS) (string, string, string) {
	t.Helper()
	status, out, diag := canlcBuildArgs(t, ctx, bundle, ws.home, "--target", "browser", ws.grid)
	if status != 0 {
		t.Fatalf("contract grid browser build: %d %s %s", status, out, diag)
	}
	var report struct {
		BuildID   string `json:"buildID"`
		Directory string `json:"directory"`
		Assert    struct {
			Roots  int `json:"roots"`
			Passed int `json:"passed"`
			Failed int `json:"failed"`
		} `json:"assertions"`
	}
	if err := json.Unmarshal([]byte(out), &report); err != nil || report.BuildID == "" || report.Directory == "" {
		t.Fatalf("invalid grid build report %v %s", err, out)
	}
	if report.Assert.Failed != 0 || report.Assert.Passed == 0 || report.Assert.Passed != report.Assert.Roots {
		t.Fatalf("grid build assertions %+v", report.Assert)
	}
	t.Logf("grid build %s executed %d assertions", report.BuildID[:12], report.Assert.Passed)
	manifest := filepath.Join(report.Directory, "browser", "manifest.json")
	raw, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		BrowserBuildID string `json:"browserBuildId"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil || decoded.BrowserBuildID == "" {
		t.Fatalf("browser manifest lacks its build identity in %s", raw)
	}
	return report.BuildID, report.Directory, manifest
}

// contractLiveBuildServer builds the staged server, paired with the
// grid manifest when one is given, and returns its build identity,
// directory, entry and paired script route.
func contractLiveBuildServer(t *testing.T, ctx context.Context, bundle string, ws contractLiveWS, manifest string) (string, string, string, string) {
	t.Helper()
	args := []string{ws.server}
	if manifest != "" {
		args = []string{"--browser-manifest", manifest, ws.server}
	}
	status, out, diag := canlcBuildArgs(t, ctx, bundle, ws.home, args...)
	if status != 0 {
		t.Fatalf("contract server build: %d %s %s", status, out, diag)
	}
	var report struct {
		BuildID   string `json:"buildID"`
		Directory string `json:"directory"`
		Browser   *struct {
			BrowserBuildID string `json:"browserBuildId"`
			Entry          string `json:"entry"`
		} `json:"browser"`
		Assert struct {
			Roots  int `json:"roots"`
			Passed int `json:"passed"`
			Failed int `json:"failed"`
		} `json:"assertions"`
	}
	if err := json.Unmarshal([]byte(out), &report); err != nil || report.BuildID == "" || report.Directory == "" {
		t.Fatalf("invalid server build report %v %s", err, out)
	}
	if report.Assert.Failed != 0 || report.Assert.Passed == 0 || report.Assert.Passed != report.Assert.Roots {
		t.Fatalf("server build assertions %+v", report.Assert)
	}
	t.Logf("server build %s executed %d assertions", report.BuildID[:12], report.Assert.Passed)
	paired := ""
	if manifest != "" {
		if report.Browser == nil || report.Browser.BrowserBuildID == "" || report.Browser.Entry == "" {
			t.Fatalf("paired report lacks its browser section: %s", out)
		}
		paired = report.Browser.Entry
		if !strings.HasPrefix(paired, "/__can/assets/") || !strings.HasSuffix(paired, ".js") {
			t.Fatalf("paired entry %q is not a digest script route", paired)
		}
	}
	return report.BuildID, report.Directory, filepath.Join(report.Directory, "entry.ts"), paired
}

type contractObserverCall struct {
	Method       string `json:"method"`
	URL          string `json:"url"`
	Status       int    `json:"status"`
	RequestBody  string `json:"requestBody"`
	ResponseBody string `json:"responseBody"`
}

type contractObserverReport struct {
	Passed   bool   `json:"passed"`
	Browser  string `json:"browser"`
	Version  string `json:"version"`
	Scenario string `json:"scenario"`
	Checks   []struct {
		Name   string `json:"name"`
		Passed bool   `json:"passed"`
		Detail string `json:"detail"`
	} `json:"checks"`
	Requests []struct {
		Method string `json:"method"`
		URL    string `json:"url"`
		Status int    `json:"status"`
	} `json:"requests"`
	Ledger []contractObserverCall `json:"ledger"`
	DOM    *struct {
		H1         string `json:"h1"`
		Status     string `json:"status"`
		StatusRole string `json:"statusRole"`
		Totals     string `json:"totals"`
		Rows       int    `json:"rows"`
	} `json:"dom"`
	CSP           string   `json:"csp"`
	Aborted       []string `json:"aborted"`
	PageErrors    []string `json:"pageerrors"`
	ConsoleErrors []string `json:"consoleErrors"`
}

// contractLiveBrowser runs the task-local observer against the served
// paired application and returns its parsed report.
func contractLiveBrowser(t *testing.T, ctx context.Context, sourceRoot, base, scenario, tag string) contractObserverReport {
	t.Helper()
	nodePath, err := exec.LookPath("node")
	if err != nil {
		t.Fatal("node is required for the contract observer")
	}
	browserDir := filepath.Join(sourceRoot, "tests/integration/browser")
	if _, err := os.Stat(filepath.Join(browserDir, "node_modules/playwright/package.json")); err != nil {
		t.Fatal("run bun install in tests/integration/browser for the pinned harness")
	}
	outdir := t.TempDir()
	harness := exec.CommandContext(ctx, nodePath, "invoice-contract.mjs", "chromium", base, outdir, scenario)
	harness.Dir = browserDir
	harness.Env = []string{"PATH=" + filepath.Dir(nodePath) + ":/usr/bin:/bin", "HOME=" + os.Getenv("HOME")}
	result, err := harness.CombinedOutput()
	raw, readErr := os.ReadFile(filepath.Join(outdir, "report.json"))
	if err != nil {
		t.Fatalf("contract observer %s: %v %s report=%s", scenario, err, result, raw)
	}
	if readErr != nil {
		t.Fatal(readErr)
	}
	var report contractObserverReport
	if err := json.Unmarshal(raw, &report); err != nil || !report.Passed {
		t.Fatalf("contract observer %s invalid report %v %s", scenario, err, raw)
	}
	for _, entry := range report.Checks {
		if !entry.Passed {
			t.Fatalf("contract observer %s check %s failed: %s", scenario, entry.Name, entry.Detail)
		}
	}
	if report.Browser != "chromium" || report.Version == "" {
		t.Fatalf("contract observer identity %+v", report)
	}
	if len(report.Aborted) != 0 || len(report.PageErrors) != 0 {
		t.Fatalf("contract observer aborted=%v pageerrors=%v", report.Aborted, report.PageErrors)
	}
	for _, entry := range report.Requests {
		if !strings.Contains(entry.URL, "127.0.0.1") && !strings.Contains(entry.URL, "localhost") {
			t.Fatalf("contract grid left loopback: %s", entry.URL)
		}
	}
	shot, err := os.Stat(filepath.Join(outdir, "screenshot.png"))
	if err != nil || shot.Size() == 0 {
		t.Fatal("contract observer left no screenshot")
	}
	if evidence := os.Getenv("CAN_BROWSER_EVIDENCE_DIR"); evidence != "" {
		dest := filepath.Join(evidence, "contract-"+tag+"-"+scenario)
		if err := os.MkdirAll(dest, 0700); err != nil {
			t.Fatal(err)
		}
		copyEvidenceFile(t, filepath.Join(outdir, "report.json"), filepath.Join(dest, "report.json"))
		copyEvidenceFile(t, filepath.Join(outdir, "screenshot.png"), filepath.Join(dest, "screenshot.png"))
	}
	return report
}

type contractLiveEvidence struct {
	Edit   string `json:"edit"`
	Phase  string `json:"phase"`
	Detail string `json:"detail"`
}

func logContractLive(t *testing.T, rows []contractLiveEvidence) {
	t.Helper()
	raw, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("contract live evidence:\n%s", raw)
}

// contractLiveBundle shares the suite-wide staged toolchain across the
// sequential contract tests; every test keeps its own projects, homes,
// databases and ports.
func contractLiveBundle(t *testing.T, ctx context.Context, sourceRoot, archive, name string) string {
	t.Helper()
	_, _ = sourceRoot, name
	bundle, err := harnessBundle(t, ctx, archive)
	if err != nil {
		t.Fatal(err)
	}
	return bundle
}

func contractDigest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func contractJSONLines() string {
	return `[{"key":"k1","id":"a","quantity":"2","price":"5.00"},{"key":"k2","id":"b","quantity":"1","price":"1.99"}]`
}

func contractFormValues(seats, details, revision string) url.Values {
	values := url.Values{}
	values.Set("seats", seats)
	values.Set("details", details)
	values.Set("revision", revision)
	values.Add("lines_order", "k1")
	values.Add("lines_order", "k2")
	values.Set("lines[k1][id]", "a")
	values.Set("lines[k1][quantity]", "2")
	values.Set("lines[k1][price]", "5.00")
	values.Set("lines[k2][id]", "b")
	values.Set("lines[k2][quantity]", "1")
	values.Set("lines[k2][price]", "1.99")
	return values
}

func TestInvoiceContractRoute(t *testing.T) {
	t.Parallel()
	acquireHeavy(t)
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged contract execution")
	}
	sourceRoot := mustSourceRoot(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	bundle := contractLiveBundle(t, ctx, sourceRoot, archive, "contract-route")
	ws := contractLiveStage(t, sourceRoot)
	var evidence []contractLiveEvidence
	digest := contractLiveEditShared(t, ws, [][2]string{
		{`get "/api/tenants/:tenant_id/invoices/:invoice_id"`, `get "/api/v2/tenants/:tenant_id/invoices/:invoice_id"`},
	})
	evidence = append(evidence, contractLiveEvidence{Edit: "route", Phase: "relock", Detail: digest[:12]})
	gridID, _, manifest := contractLiveBuildGrid(t, ctx, bundle, ws)
	serverID, _, entry, paired := contractLiveBuildServer(t, ctx, bundle, ws, manifest)
	evidence = append(evidence,
		contractLiveEvidence{Edit: "route", Phase: "grid-build", Detail: gridID},
		contractLiveEvidence{Edit: "route", Phase: "server-build", Detail: serverID + " entry " + paired},
	)
	driver := filepath.Join(sourceRoot, "tests/integration/testdata/invoice/driver.ts")
	db := seedInvoiceDB(t, ctx, bundle, ws.home, driver, ws.server)
	base, stop := serveInvoice(t, ctx, bundle, ws.home, entry, db, contractLivePortRoute, "")
	defer stop()
	origin := "http://127.0.0.1:" + strconv.Itoa(contractLivePortRoute)

	// Served bytes carry the edited load path in the paired entry.
	status, script, headers := invoiceGet(t, base, paired, "")
	if status != 200 || !strings.Contains(headers.Get("Content-Type"), "text/javascript") {
		t.Fatalf("paired entry: %d %q", status, headers.Get("Content-Type"))
	}
	if !strings.Contains(script, "/api/v2/tenants") || !strings.Contains(script, "$canBrowserMain") {
		t.Fatal("paired entry lacks the edited load path or browser entry")
	}
	evidence = append(evidence, contractLiveEvidence{Edit: "route", Phase: "bytes", Detail: paired + " sha256 " + contractDigest(script)[:12]})

	// The old GET spelling no longer enters the load handler: the
	// path is still recognized by the kept POST save, so it answers
	// 405, while a genuinely unknown path answers 404. The new path
	// enters the bound load handler and returns live database state.
	status, body, headers := invoiceGet(t, base, "/api/tenants/1/invoices/7", "tok-alice")
	if status != 405 || !strings.Contains(headers.Get("Allow"), "POST") {
		t.Fatalf("old load path: %d %q allow %q, want 405 with POST", status, body, headers.Get("Allow"))
	}
	status, body, _ = invoiceGet(t, base, "/api/v9/tenants/1/invoices/7", "tok-alice")
	if status != 404 || body != "Not Found" {
		t.Fatalf("unknown path: %d %q, want direct 404", status, body)
	}
	status, body, _ = invoiceGet(t, base, "/api/v2/tenants/1/invoices/7", "tok-alice")
	if status != 200 {
		t.Fatalf("new load path: %d %s", status, body)
	}
	loaded := gate3Case(t, "new load", body, "invoice_contract::grid_loaded")
	snapshot, _ := loaded["current"].(map[string]any)
	lines, _ := snapshot["lines"].([]any)
	if snapshot["revision"] != "1" || len(lines) != 2 {
		t.Fatalf("new load snapshot %+v, want live revision 1 with 2 lines", loaded)
	}
	if status, body, _ := invoiceGet(t, base, "/api/v2/tenants/1x/invoices/7", "tok-alice"); status != 400 {
		t.Fatalf("new load malformed capture: %d %s, want 400", status, body)
	}
	store := inspectInvoice(t, ctx, bundle, ws.home, driver, db)
	requireInvoice(t, store, "7", "1", "2", "Acme <em>&\" 'coop'\"")
	if len(store.Replay) != 0 {
		t.Fatalf("reads recorded %+v", store.Replay)
	}
	evidence = append(evidence, contractLiveEvidence{Edit: "route", Phase: "load", Detail: "old GET 405, unknown 404, new 200 with live snapshot, malformed 400"})

	// The untouched save path still enters its handler and commits.
	save := `{"operation_id":"op-c1","revision":"1","lines":` + contractJSONLines() + `}`
	status, payload, _ := postInvoiceJSON(t, base, "/api/tenants/1/invoices/7", "tok-alice", origin, "application/json", save)
	if status != 200 {
		t.Fatalf("save on the kept path: %d %s", status, payload)
	}
	gate3Case(t, "kept save", payload, "invoice_contract::grid_saved")
	store = inspectInvoice(t, ctx, bundle, ws.home, driver, db)
	requireInvoice(t, store, "7", "2", "2", "Acme <em>&\" 'coop'\"")
	requireReplay(t, store, "op-c1", "2")

	report := contractLiveBrowser(t, ctx, sourceRoot, base, "save", "route")
	loads, saves := 0, 0
	for _, call := range report.Ledger {
		switch call.Method {
		case "GET":
			loads++
			if !strings.Contains(call.URL, "/api/v2/tenants/1/invoices/7") {
				t.Fatalf("browser load used a stale path: %s", call.URL)
			}
		case "POST":
			saves++
			if call.Status != 200 || !strings.Contains(call.URL, "/api/tenants/1/invoices/7") {
				t.Fatalf("browser save %+v", call)
			}
			if !strings.Contains(call.RequestBody, `"quantity":"3"`) {
				t.Fatalf("browser save missed the folded edit in %s", call.RequestBody)
			}
		}
	}
	if loads == 0 {
		t.Fatal("browser sent no load through the edited path")
	}
	if saves != 1 || report.DOM.H1 != "Tenant 1 invoice 7 revision 3" || report.DOM.Status != "saved revision 3" {
		t.Fatalf("browser save dom %+v saves %d", report.DOM, saves)
	}
	store = inspectInvoice(t, ctx, bundle, ws.home, driver, db)
	requireInvoice(t, store, "7", "3", "2", "Acme <em>&\" 'coop'\"")
	if len(store.Replay) != 2 {
		t.Fatalf("browser save replay %+v, want 2 rows", store.Replay)
	}
	evidence = append(evidence, contractLiveEvidence{Edit: "route", Phase: "browser", Detail: report.DOM.H1 + " / " + report.DOM.Status})
	logContractLive(t, evidence)
}

func TestInvoiceContractCapture(t *testing.T) {
	t.Parallel()
	acquireHeavy(t)
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged contract execution")
	}
	sourceRoot := mustSourceRoot(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	bundle := contractLiveBundle(t, ctx, sourceRoot, archive, "contract-capture")
	ws := contractLiveStage(t, sourceRoot)
	var evidence []contractLiveEvidence
	digest := contractLiveEditShared(t, ws, [][2]string{{"invoice_id", "document_id"}})
	evidence = append(evidence, contractLiveEvidence{Edit: "capture", Phase: "relock", Detail: digest[:12]})
	repaired := 0
	for _, root := range []string{ws.server, ws.grid} {
		err := filepath.WalkDir(filepath.Join(root, "src"), func(name string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !strings.HasSuffix(name, ".can") {
				return err
			}
			content := contractLiveRead(t, name)
			if count := strings.Count(content, "key.invoice_id"); count > 0 {
				contractLiveWrite(t, name, strings.ReplaceAll(content, "key.invoice_id", "key.document_id"))
				repaired += count
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if repaired == 0 {
		t.Fatal("capture repair found no named uses")
	}
	gridID, _, manifest := contractLiveBuildGrid(t, ctx, bundle, ws)
	serverID, _, entry, paired := contractLiveBuildServer(t, ctx, bundle, ws, manifest)
	evidence = append(evidence,
		contractLiveEvidence{Edit: "capture", Phase: "grid-build", Detail: gridID},
		contractLiveEvidence{Edit: "capture", Phase: "server-build", Detail: serverID + " entry " + paired},
	)
	driver := filepath.Join(sourceRoot, "tests/integration/testdata/invoice/driver.ts")
	db := seedInvoiceDB(t, ctx, bundle, ws.home, driver, ws.server)
	base, stop := serveInvoice(t, ctx, bundle, ws.home, entry, db, contractLivePortCapture, "")
	defer stop()
	origin := "http://127.0.0.1:" + strconv.Itoa(contractLivePortCapture)

	status, script, _ := invoiceGet(t, base, paired, "")
	if status != 200 || !strings.Contains(script, "document_id") {
		t.Fatal("paired entry lacks the renamed capture")
	}
	status, body, _ := invoiceGet(t, base, "/api/tenants/1/invoices/7", "tok-alice")
	if status != 200 {
		t.Fatalf("renamed capture load: %d %s", status, body)
	}
	loaded := gate3Case(t, "renamed load", body, "invoice_contract::grid_loaded")
	if loaded["current"].(map[string]any)["revision"] != "1" {
		t.Fatalf("renamed load snapshot %+v", loaded)
	}
	if status, body, _ := invoiceGet(t, base, "/api/tenants/1x/invoices/7", "tok-alice"); status != 400 {
		t.Fatalf("renamed capture malformed: %d %s, want 400", status, body)
	}
	save := `{"operation_id":"op-c2","revision":"1","lines":` + contractJSONLines() + `}`
	status, payload, _ := postInvoiceJSON(t, base, "/api/tenants/1/invoices/7", "tok-alice", origin, "application/json", save)
	if status != 200 {
		t.Fatalf("renamed capture save: %d %s", status, payload)
	}
	gate3Case(t, "renamed save", payload, "invoice_contract::grid_saved")
	store := inspectInvoice(t, ctx, bundle, ws.home, driver, db)
	requireInvoice(t, store, "7", "2", "2", "Acme <em>&\" 'coop'\"")
	requireReplay(t, store, "op-c2", "2")
	evidence = append(evidence, contractLiveEvidence{Edit: "capture", Phase: "http", Detail: "load/save 200 through the renamed capture, malformed 400"})

	report := contractLiveBrowser(t, ctx, sourceRoot, base, "save", "capture")
	saves := 0
	for _, call := range report.Ledger {
		if call.Method == "POST" {
			saves++
			if call.Status != 200 {
				t.Fatalf("browser save %+v", call)
			}
		}
	}
	if saves != 1 || report.DOM.H1 != "Tenant 1 invoice 7 revision 3" || report.DOM.Status != "saved revision 3" {
		t.Fatalf("browser save dom %+v saves %d", report.DOM, saves)
	}
	store = inspectInvoice(t, ctx, bundle, ws.home, driver, db)
	requireInvoice(t, store, "7", "3", "2", "Acme <em>&\" 'coop'\"")
	evidence = append(evidence, contractLiveEvidence{Edit: "capture", Phase: "browser", Detail: report.DOM.H1 + " / " + report.DOM.Status})
	logContractLive(t, evidence)
}

func TestInvoiceContractField(t *testing.T) {
	t.Parallel()
	acquireHeavy(t)
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged contract execution")
	}
	sourceRoot := mustSourceRoot(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	bundle := contractLiveBundle(t, ctx, sourceRoot, archive, "contract-field")
	ws := contractLiveStage(t, sourceRoot)
	var evidence []contractLiveEvidence
	digest := contractLiveEditShared(t, ws, [][2]string{{
		"record grid_edit_input\n    str operation_id\n    str revision\n    grid_line_wire[] lines",
		"record grid_edit_input\n    str operation_id\n    str expected_revision\n    grid_line_wire[] lines",
	}})
	evidence = append(evidence, contractLiveEvidence{Edit: "field", Phase: "relock", Detail: digest[:12]})
	contractLiveReplaceOnce(t, filepath.Join(ws.server, "src/model/model.can"),
		"    match call text::to_int(body.revision)\n        text::invalid_number => ok records::flawed_save(\"bad revision\", \"revision\")\n        ok int revision => match revision >= 1\n            false => ok records::flawed_save(\"bad revision\", \"revision\")\n            true => match call json_parse_lines(body.lines, revision, 0, [])",
		"    match call text::to_int(body.expected_revision)\n        text::invalid_number => ok records::flawed_save(\"bad revision\", \"revision\")\n        ok int revision => match revision >= 1\n            false => ok records::flawed_save(\"bad revision\", \"revision\")\n            true => match call json_parse_lines(body.lines, revision, 0, [])")
	gridID, _, manifest := contractLiveBuildGrid(t, ctx, bundle, ws)
	serverID, _, entry, paired := contractLiveBuildServer(t, ctx, bundle, ws, manifest)
	evidence = append(evidence,
		contractLiveEvidence{Edit: "field", Phase: "grid-build", Detail: gridID},
		contractLiveEvidence{Edit: "field", Phase: "server-build", Detail: serverID + " entry " + paired},
	)
	driver := filepath.Join(sourceRoot, "tests/integration/testdata/invoice/driver.ts")
	db := seedInvoiceDB(t, ctx, bundle, ws.home, driver, ws.server)
	base, stop := serveInvoice(t, ctx, bundle, ws.home, entry, db, contractLivePortField, "")
	defer stop()
	origin := "http://127.0.0.1:" + strconv.Itoa(contractLivePortField)

	status, script, _ := invoiceGet(t, base, paired, "")
	if status != 200 || !strings.Contains(script, "expected_revision") {
		t.Fatal("paired entry lacks the renamed field")
	}
	// The server decodes the new field and the protected write uses
	// it; an old-field payload is rejected without mutation.
	save := `{"operation_id":"op-f1","expected_revision":"1","lines":` + contractJSONLines() + `}`
	status, payload, _ := postInvoiceJSON(t, base, "/api/tenants/1/invoices/7", "tok-alice", origin, "application/json", save)
	if status != 200 {
		t.Fatalf("new-field save: %d %s", status, payload)
	}
	value := gate3Case(t, "new-field save", payload, "invoice_contract::grid_saved")
	if acknowledged, _ := value["acknowledged"].(map[string]any); acknowledged["revision"] != "2" {
		t.Fatalf("new-field save value %+v", value)
	}
	stale := `{"operation_id":"op-old","revision":"1","lines":[]}`
	status, payload, _ = postInvoiceJSON(t, base, "/api/tenants/1/invoices/7", "tok-alice", origin, "application/json", stale)
	if status == 200 {
		t.Fatalf("old-field payload committed: %d %s", status, payload)
	}
	store := inspectInvoice(t, ctx, bundle, ws.home, driver, db)
	requireInvoice(t, store, "7", "2", "2", "Acme <em>&\" 'coop'\"")
	requireReplay(t, store, "op-f1", "2")
	for _, row := range store.Replay {
		if row.OperationID == "op-old" {
			t.Fatalf("old-field payload recorded %+v", row)
		}
	}
	evidence = append(evidence, contractLiveEvidence{Edit: "field", Phase: "http", Detail: "new field 200 rev 2, old field rejected without mutation"})

	report := contractLiveBrowser(t, ctx, sourceRoot, base, "save", "field")
	saves := 0
	for _, call := range report.Ledger {
		if call.Method != "POST" {
			continue
		}
		saves++
		if call.Status != 200 {
			t.Fatalf("browser save %+v", call)
		}
		if count := strings.Count(call.RequestBody, `"expected_revision"`); count != 1 {
			t.Fatalf("browser request carries the new field %d times in %s", count, call.RequestBody)
		}
		if strings.Contains(call.RequestBody, `"revision"`) {
			t.Fatalf("browser request still sends the old field in %s", call.RequestBody)
		}
	}
	if saves != 1 || report.DOM.Status != "saved revision 3" {
		t.Fatalf("browser save dom %+v saves %d", report.DOM, saves)
	}
	store = inspectInvoice(t, ctx, bundle, ws.home, driver, db)
	requireInvoice(t, store, "7", "3", "2", "Acme <em>&\" 'coop'\"")
	evidence = append(evidence, contractLiveEvidence{Edit: "field", Phase: "browser", Detail: report.DOM.H1 + " / " + report.DOM.Status})
	logContractLive(t, evidence)
}

func TestInvoiceContractLeaf(t *testing.T) {
	t.Parallel()
	acquireHeavy(t)
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged contract execution")
	}
	sourceRoot := mustSourceRoot(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	bundle := contractLiveBundle(t, ctx, sourceRoot, archive, "contract-leaf")
	ws := contractLiveStage(t, sourceRoot)
	var evidence []contractLiveEvidence
	digest := contractLiveEditShared(t, ws, [][2]string{{"grid_invalid", "grid_rejected"}})
	evidence = append(evidence, contractLiveEvidence{Edit: "leaf", Phase: "relock", Detail: digest[:12]})
	repaired := 0
	for _, root := range []string{ws.server, ws.grid} {
		err := filepath.WalkDir(filepath.Join(root, "src"), func(name string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !strings.HasSuffix(name, ".can") {
				return err
			}
			content := contractLiveRead(t, name)
			if count := strings.Count(content, "grid_invalid"); count > 0 {
				contractLiveWrite(t, name, strings.ReplaceAll(content, "grid_invalid", "grid_rejected"))
				repaired += count
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if repaired == 0 {
		t.Fatal("leaf repair found no named uses")
	}
	gridID, _, manifest := contractLiveBuildGrid(t, ctx, bundle, ws)
	serverID, _, entry, paired := contractLiveBuildServer(t, ctx, bundle, ws, manifest)
	evidence = append(evidence,
		contractLiveEvidence{Edit: "leaf", Phase: "grid-build", Detail: gridID},
		contractLiveEvidence{Edit: "leaf", Phase: "server-build", Detail: serverID + " entry " + paired},
	)
	driver := filepath.Join(sourceRoot, "tests/integration/testdata/invoice/driver.ts")
	db := seedInvoiceDB(t, ctx, bundle, ws.home, driver, ws.server)
	base, stop := serveInvoice(t, ctx, bundle, ws.home, entry, db, contractLivePortLeaf, "")
	defer stop()
	origin := "http://127.0.0.1:" + strconv.Itoa(contractLivePortLeaf)

	status, script, _ := invoiceGet(t, base, paired, "")
	if status != 200 || !strings.Contains(script, "grid_rejected") {
		t.Fatal("paired entry lacks the renamed leaf")
	}
	if strings.Contains(script, "grid_invalid") {
		t.Fatal("paired entry still interprets the old tag")
	}
	// A real invalid save returns 422 with the new case tag and the
	// browser renders its draft and errors.
	badPrice := `{"operation_id":"op-l1","revision":"1","lines":[{"key":"k1","id":"a","quantity":"2","price":"five"}]}`
	status, payload, _ := postInvoiceJSON(t, base, "/api/tenants/1/invoices/7", "tok-alice", origin, "application/json", badPrice)
	if status != 422 {
		t.Fatalf("renamed invalid save: %d %s, want 422", status, payload)
	}
	value := gate3Case(t, "renamed invalid", payload, "invoice_contract::grid_rejected")
	problems, _ := value["errors"].([]any)
	if len(problems) != 1 || problems[0].(map[string]any)["field"] != "price" {
		t.Fatalf("renamed invalid value %+v", value)
	}
	save := `{"operation_id":"op-l2","revision":"1","lines":` + contractJSONLines() + `}`
	status, payload, _ = postInvoiceJSON(t, base, "/api/tenants/1/invoices/7", "tok-alice", origin, "application/json", save)
	if status != 200 || strings.Contains(payload, "grid_invalid") {
		t.Fatalf("valid save after rename: %d %s", status, payload)
	}
	store := inspectInvoice(t, ctx, bundle, ws.home, driver, db)
	requireInvoice(t, store, "7", "2", "2", "Acme <em>&\" 'coop'\"")
	evidence = append(evidence, contractLiveEvidence{Edit: "leaf", Phase: "http", Detail: "422 grid_rejected, then 200"})

	invalid := contractLiveBrowser(t, ctx, sourceRoot, base, "invalid", "leaf")
	found := false
	for _, call := range invalid.Ledger {
		if call.Method == "POST" && call.Status == 422 {
			found = true
			if !strings.Contains(call.ResponseBody, "invoice_contract::grid_rejected") {
				t.Fatalf("browser 422 lacks the new tag in %s", call.ResponseBody)
			}
		}
	}
	if !found {
		t.Fatal("browser sent no invalid save")
	}
	// The grid blocks unparsable numbers client-side, so the
	// observer clears the id input instead: the draft still sends
	// and the server rejects it with a rendered empty-id error.
	if !strings.Contains(invalid.DOM.Status, "empty line id") {
		t.Fatalf("browser dom %+v lacks the rendered error", invalid.DOM)
	}
	// The valid HTTP save above already proves the 200 path; the
	// browser leg stays on the invalid render.
	store = inspectInvoice(t, ctx, bundle, ws.home, driver, db)
	requireInvoice(t, store, "7", "2", "2", "Acme <em>&\" 'coop'\"")
	requireReplay(t, store, "op-l2", "2")
	if len(store.Replay) != 1 {
		t.Fatalf("invalid legs recorded %+v", store.Replay)
	}
	evidence = append(evidence, contractLiveEvidence{Edit: "leaf", Phase: "browser", Detail: invalid.DOM.Status})
	logContractLive(t, evidence)
}

func TestInvoiceContractStatus(t *testing.T) {
	t.Parallel()
	acquireHeavy(t)
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged contract execution")
	}
	sourceRoot := mustSourceRoot(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	bundle := contractLiveBundle(t, ctx, sourceRoot, archive, "contract-status")
	ws := contractLiveStage(t, sourceRoot)
	var evidence []contractLiveEvidence
	shared := filepath.Join(ws.shared, "src/invoice_contract/invoice_contract.can")
	contractLiveReplaceOnce(t, shared, "grid_saved status 200", "grid_saved status 201")
	edited, err := os.ReadFile(shared)
	if err != nil {
		t.Fatal(err)
	}
	for _, root := range []string{ws.server, ws.grid} {
		if err := os.WriteFile(filepath.Join(root, "vendor/billing/src/invoice_contract/invoice_contract.can"), edited, 0600); err != nil {
			t.Fatal(err)
		}
	}
	serverDigest, gridDigest := contractLiveRelock(t, ws.server), contractLiveRelock(t, ws.grid)
	if serverDigest != gridDigest {
		t.Fatalf("relock diverged: %s vs %s", serverDigest, gridDigest)
	}
	evidence = append(evidence, contractLiveEvidence{Edit: "status", Phase: "relock", Detail: serverDigest[:12]})
	gridID, _, manifest := contractLiveBuildGrid(t, ctx, bundle, ws)
	serverID, _, entry, paired := contractLiveBuildServer(t, ctx, bundle, ws, manifest)
	evidence = append(evidence,
		contractLiveEvidence{Edit: "status", Phase: "grid-build", Detail: gridID},
		contractLiveEvidence{Edit: "status", Phase: "server-build", Detail: serverID + " entry " + paired},
	)
	driver := filepath.Join(sourceRoot, "tests/integration/testdata/invoice/driver.ts")
	db := seedInvoiceDB(t, ctx, bundle, ws.home, driver, ws.server)
	base, stop := serveInvoice(t, ctx, bundle, ws.home, entry, db, contractLivePortStatus, "")
	defer stop()
	origin := "http://127.0.0.1:" + strconv.Itoa(contractLivePortStatus)

	// Actual response status and checked client case table move
	// together: the live save answers 201 and the browser reconciles
	// it instead of reporting an unexpected status.
	save := `{"operation_id":"op-s1","revision":"1","lines":` + contractJSONLines() + `}`
	status, payload, _ := postInvoiceJSON(t, base, "/api/tenants/1/invoices/7", "tok-alice", origin, "application/json", save)
	if status != 201 {
		t.Fatalf("moved status save: %d %s, want 201", status, payload)
	}
	gate3Case(t, "moved status", payload, "invoice_contract::grid_saved")
	store := inspectInvoice(t, ctx, bundle, ws.home, driver, db)
	requireInvoice(t, store, "7", "2", "2", "Acme <em>&\" 'coop'\"")
	requireReplay(t, store, "op-s1", "2")

	report := contractLiveBrowser(t, ctx, sourceRoot, base, "save", "status")
	saves := 0
	for _, call := range report.Ledger {
		if call.Method == "POST" {
			saves++
			if call.Status != 201 {
				t.Fatalf("browser save status %+v, want 201", call)
			}
		}
	}
	if saves != 1 || report.DOM.Status != "saved revision 3" {
		t.Fatalf("browser save dom %+v saves %d", report.DOM, saves)
	}
	store = inspectInvoice(t, ctx, bundle, ws.home, driver, db)
	requireInvoice(t, store, "7", "3", "2", "Acme <em>&\" 'coop'\"")
	evidence = append(evidence, contractLiveEvidence{Edit: "status", Phase: "browser", Detail: report.DOM.H1 + " / " + report.DOM.Status})

	// The committed base keeps its selected statuses and JSON limit.
	pristine, err := os.ReadFile(filepath.Join(sourceRoot, "shared/invoice-contract/src/invoice_contract/invoice_contract.can"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"grid_saved status 200", "grid_invalid status 422", "grid_conflict status 409",
		"grid_forbidden status 403", "grid_unavailable status 503", "json grid_edit_input limit 8192",
	} {
		if !strings.Contains(string(pristine), want) {
			t.Fatalf("committed base lost %q", want)
		}
	}
	logContractLive(t, evidence)
}

func TestInvoiceContractLimit(t *testing.T) {
	t.Parallel()
	acquireHeavy(t)
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged contract execution")
	}
	sourceRoot := mustSourceRoot(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	bundle := contractLiveBundle(t, ctx, sourceRoot, archive, "contract-limit")
	ws := contractLiveStage(t, sourceRoot)
	var evidence []contractLiveEvidence
	digest := contractLiveEditShared(t, ws, [][2]string{{"json grid_edit_input limit 8192", "json grid_edit_input limit 128"}})
	evidence = append(evidence, contractLiveEvidence{Edit: "limit", Phase: "relock", Detail: digest[:12]})
	gridID, _, manifest := contractLiveBuildGrid(t, ctx, bundle, ws)
	serverID, _, entry, paired := contractLiveBuildServer(t, ctx, bundle, ws, manifest)
	evidence = append(evidence,
		contractLiveEvidence{Edit: "limit", Phase: "grid-build", Detail: gridID},
		contractLiveEvidence{Edit: "limit", Phase: "server-build", Detail: serverID + " entry " + paired},
	)
	driver := filepath.Join(sourceRoot, "tests/integration/testdata/invoice/driver.ts")
	db := seedInvoiceDB(t, ctx, bundle, ws.home, driver, ws.server)
	base, stop := serveInvoice(t, ctx, bundle, ws.home, entry, db, contractLivePortLimit, "")
	defer stop()
	origin := "http://127.0.0.1:" + strconv.Itoa(contractLivePortLimit)

	// Over-limit raw HTTP is 413 before handler entry: no revision
	// moves and no replay row is minted.
	big := `{"operation_id":"op-big","revision":"1","lines":` + contractJSONLines() + `}`
	if len(big) <= 128 {
		t.Fatalf("oversized fixture is %d bytes", len(big))
	}
	status, payload, _ := postInvoiceJSON(t, base, "/api/tenants/1/invoices/7", "tok-alice", origin, "application/json", big)
	if status != 413 {
		t.Fatalf("over-limit save: %d %s, want 413", status, payload)
	}
	store := inspectInvoice(t, ctx, bundle, ws.home, driver, db)
	requireInvoice(t, store, "7", "1", "2", "Acme <em>&\" 'coop'\"")
	if len(store.Replay) != 0 {
		t.Fatalf("over-limit save recorded %+v", store.Replay)
	}
	evidence = append(evidence, contractLiveEvidence{Edit: "limit", Phase: "http", Detail: "over-limit 413 pre-entry"})

	// The browser applies the same declared budget: the loaded
	// two-line draft exceeds 128 bytes, so the save fails in the
	// client with no request on the wire. This runs before the
	// under-limit save below shrinks the invoice to one line.
	report := contractLiveBrowser(t, ctx, sourceRoot, base, "budget", "limit")
	if report.DOM.Status != "save too large; remove lines and retry" {
		t.Fatalf("browser budget dom %+v", report.DOM)
	}
	for _, call := range report.Ledger {
		if call.Method == "POST" {
			t.Fatalf("browser sent an over-budget save: %+v", call)
		}
	}
	store = inspectInvoice(t, ctx, bundle, ws.home, driver, db)
	requireInvoice(t, store, "7", "1", "2", "Acme <em>&\" 'coop'\"")
	if len(store.Replay) != 0 {
		t.Fatalf("budget save recorded %+v", store.Replay)
	}
	evidence = append(evidence, contractLiveEvidence{Edit: "limit", Phase: "browser", Detail: report.DOM.Status + ", no POST"})

	small := `{"operation_id":"op-small","revision":"1","lines":[{"key":"k1","id":"a","quantity":"1","price":"1.00"}]}`
	if len(small) > 128 {
		t.Fatalf("undersized fixture is %d bytes", len(small))
	}
	status, payload, _ = postInvoiceJSON(t, base, "/api/tenants/1/invoices/7", "tok-alice", origin, "application/json", small)
	if status != 200 {
		t.Fatalf("under-limit save: %d %s", status, payload)
	}
	store = inspectInvoice(t, ctx, bundle, ws.home, driver, db)
	row := requireInvoice(t, store, "7", "2", "2", "Acme <em>&\" 'coop'\"")
	if len(row.Lines) != 1 || row.Lines[0].Key != "k1" || row.Lines[0].ID != "a" {
		t.Fatalf("under-limit save lines %+v, want the single shrunk line", row.Lines)
	}
	requireReplay(t, store, "op-small", "2")
	evidence = append(evidence, contractLiveEvidence{Edit: "limit", Phase: "http-under", Detail: "under-limit 200 rev 2"})
	logContractLive(t, evidence)
}

func TestInvoiceContractBodyMode(t *testing.T) {
	t.Parallel()
	acquireHeavy(t)
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged contract execution")
	}
	sourceRoot := mustSourceRoot(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	bundle := contractLiveBundle(t, ctx, sourceRoot, archive, "contract-bodymode")
	ws := contractLiveStage(t, sourceRoot)
	var evidence []contractLiveEvidence
	shared := filepath.Join(ws.shared, "src/invoice_contract/invoice_contract.can")
	contractLiveReplaceOnce(t, shared,
		"    json grid_edit_input limit 8192\n    returns grid_edit_outcome\n    body json\n    cases\n        grid_saved status 200\n        grid_invalid status 422\n        grid_conflict status 409\n        grid_forbidden status 403\n        grid_unavailable status 503",
		"    form invoice_form limit 2048 rows_limit 64\n    returns edit_outcome\n    body html\n    cases\n        saved status 200 swap inner\n        invalid status 422 swap inner\n        conflict status 409 swap inner\n        forbidden status 403 swap inner\n        unavailable status 503 swap inner")
	edited, err := os.ReadFile(shared)
	if err != nil {
		t.Fatal(err)
	}
	for _, root := range []string{ws.server, ws.grid} {
		if err := os.WriteFile(filepath.Join(root, "vendor/billing/src/invoice_contract/invoice_contract.can"), edited, 0600); err != nil {
			t.Fatal(err)
		}
	}
	serverDigest, gridDigest := contractLiveRelock(t, ws.server), contractLiveRelock(t, ws.grid)
	if serverDigest != gridDigest {
		t.Fatalf("relock diverged: %s vs %s", serverDigest, gridDigest)
	}
	evidence = append(evidence, contractLiveEvidence{Edit: "body-mode", Phase: "relock", Detail: serverDigest[:12]})

	// The JSON grid has no form client: its staged browser build
	// rejects the flipped action instead of treating HTML as JSON.
	status, out, diag := canlcBuildArgs(t, ctx, bundle, ws.home, "--target", "browser", ws.grid)
	combined := out + "\n" + diag
	if status == 0 || !strings.Contains(combined, "save_invoice_grid") || !strings.Contains(combined, "JSON") {
		t.Fatalf("flipped grid build: %d %.800s, want a JSON-kind rejection", status, combined)
	}
	evidence = append(evidence, contractLiveEvidence{Edit: "body-mode", Phase: "grid-stale", Detail: strings.TrimSpace(combined)})

	contractLiveReplaceOnce(t, filepath.Join(ws.server, "src/web/web.can"),
		"call action::mount(contract::save_invoice_grid, callable save_grid) as http::route save",
		"call action::mount(contract::save_invoice_grid, callable save_html, callable render_html, callable render_bad_form) as http::route save")
	serverID, _, entry, _ := contractLiveBuildServer(t, ctx, bundle, ws, "")
	evidence = append(evidence,
		contractLiveEvidence{Edit: "body-mode", Phase: "server-build", Detail: serverID + " html-only"},
	)
	driver := filepath.Join(sourceRoot, "tests/integration/testdata/invoice/driver.ts")
	db := seedInvoiceDB(t, ctx, bundle, ws.home, driver, ws.server)
	base, stop := serveInvoice(t, ctx, bundle, ws.home, entry, db, contractLivePortBodyMode, "")
	defer stop()
	origin := "http://127.0.0.1:" + strconv.Itoa(contractLivePortBodyMode)

	// The flipped route serves HTML form saves; JSON is no longer
	// admitted there, while the untouched JSON load still answers.
	status, body, headers := postInvoiceForm(t, base, "/api/tenants/1/invoices/7", "tok-alice", origin, contractFormValues("4", "Rush order", "1"))
	if status != 200 || !strings.Contains(headers.Get("Content-Type"), "text/html") || !strings.Contains(body, "saved revision 2") {
		t.Fatalf("flipped form save: %d %q %.300s", status, headers.Get("Content-Type"), body)
	}
	store := inspectInvoice(t, ctx, bundle, ws.home, driver, db)
	requireInvoice(t, store, "7", "2", "4", "Rush order")
	save := `{"operation_id":"op-h1","revision":"2","lines":[]}`
	status, payload, _ := postInvoiceJSON(t, base, "/api/tenants/1/invoices/7", "tok-alice", origin, "application/json", save)
	if status != 415 {
		t.Fatalf("json to the flipped route: %d %s, want 415", status, payload)
	}
	status, body, _ = invoiceGet(t, base, "/api/tenants/1/invoices/7", "tok-alice")
	if status != 200 {
		t.Fatalf("untouched load: %d %s", status, body)
	}
	gate3Case(t, "untouched load", body, "invoice_contract::grid_loaded")
	evidence = append(evidence, contractLiveEvidence{Edit: "body-mode", Phase: "http", Detail: "form 200 html, json 415, load 200"})

	// The committed base grid remains JSON and never treats HTML as
	// JSON.
	pristine := contractLiveStage(t, sourceRoot)
	_, _, pristineEntry, _ := contractLiveBuildServer(t, ctx, bundle, pristine, "")
	pristineDB := seedInvoiceDB(t, ctx, bundle, pristine.home, driver, pristine.server)
	pristineBase, stopPristine := serveInvoice(t, ctx, bundle, pristine.home, pristineEntry, pristineDB, contractLivePortPristine, "")
	defer stopPristine()
	pristineOrigin := "http://127.0.0.1:" + strconv.Itoa(contractLivePortPristine)
	status, payload, _ = postInvoiceJSON(t, pristineBase, "/api/tenants/1/invoices/7", "tok-alice", pristineOrigin, "text/html", save)
	if status != 415 {
		t.Fatalf("html to the json route: %d %s, want 415", status, payload)
	}
	status, payload, headers = postInvoiceJSON(t, pristineBase, "/api/tenants/1/invoices/7", "tok-alice", pristineOrigin, "application/json", `{"operation_id":"op-p1","revision":"1","lines":`+contractJSONLines()+`}`)
	if status != 200 || !strings.Contains(headers.Get("Content-Type"), "application/json") {
		t.Fatalf("pristine json save: %d %q %s", status, headers.Get("Content-Type"), payload)
	}
	gate3Case(t, "pristine json", payload, "invoice_contract::grid_saved")
	evidence = append(evidence, contractLiveEvidence{Edit: "body-mode", Phase: "base-json", Detail: "html 415, json 200 application/json"})
	logContractLive(t, evidence)
}

func TestInvoiceContractManifestMismatch(t *testing.T) {
	t.Parallel()
	acquireHeavy(t)
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged contract execution")
	}
	sourceRoot := mustSourceRoot(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	bundle := contractLiveBundle(t, ctx, sourceRoot, archive, "contract-manifest")
	var evidence []contractLiveEvidence

	edited := contractLiveStage(t, sourceRoot)
	contractLiveEditShared(t, edited, [][2]string{
		{`get "/api/tenants/:tenant_id/invoices/:invoice_id"`, `get "/api/v2/tenants/:tenant_id/invoices/:invoice_id"`},
	})
	editedGridID, _, editedManifest := contractLiveBuildGrid(t, ctx, bundle, edited)
	pristine := contractLiveStage(t, sourceRoot)
	pristineGridID, _, pristineManifest := contractLiveBuildGrid(t, ctx, bundle, pristine)
	if editedGridID == pristineGridID {
		t.Fatal("edited and pristine grid builds share an identity")
	}
	evidence = append(evidence,
		contractLiveEvidence{Edit: "manifest", Phase: "grid-build", Detail: "edited " + editedGridID[:12] + " pristine " + pristineGridID[:12]},
	)

	// An old client cannot ship alongside a new server after a
	// contract edit.
	status, out, diag := canlcBuildArgs(t, ctx, bundle, edited.home, "--browser-manifest", pristineManifest, edited.server)
	combined := strings.TrimSpace(out + "\n" + diag)
	if status == 0 {
		t.Fatalf("edited server paired with the pristine manifest: %s", out)
	}
	evidence = append(evidence, contractLiveEvidence{Edit: "manifest", Phase: "old-client-new-server", Detail: combined})

	// A tampered manifest is rejected rather than published.
	raw, err := os.ReadFile(editedManifest)
	if err != nil {
		t.Fatal(err)
	}
	tampered := raw
	for i := len(tampered) - 20; i < len(tampered); i++ {
		if tampered[i] >= '0' && tampered[i] <= '9' {
			tampered[i] = '0' + (tampered[i]-'0'+1)%10
			break
		}
	}
	tamperedPath := filepath.Join(edited.home, "tampered-manifest.json")
	contractLiveWrite(t, tamperedPath, string(tampered))
	status, out, diag = canlcBuildArgs(t, ctx, bundle, edited.home, "--browser-manifest", tamperedPath, edited.server)
	combined = strings.TrimSpace(out + "\n" + diag)
	if status == 0 {
		t.Fatalf("edited server paired with a tampered manifest: %s", out)
	}
	evidence = append(evidence, contractLiveEvidence{Edit: "manifest", Phase: "tampered", Detail: combined})

	// Positive control: the matched pair publishes.
	serverID, _, _, paired := contractLiveBuildServer(t, ctx, bundle, edited, editedManifest)
	evidence = append(evidence, contractLiveEvidence{Edit: "manifest", Phase: "matched", Detail: serverID[:12] + " entry " + paired})
	logContractLive(t, evidence)
}

func TestInvoiceContractNoRegistration(t *testing.T) {
	t.Parallel()
	acquireHeavy(t)
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged contract execution")
	}
	sourceRoot := mustSourceRoot(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	bundle := contractLiveBundle(t, ctx, sourceRoot, archive, "contract-noregister")
	ws := contractLiveStage(t, sourceRoot)
	var evidence []contractLiveEvidence

	// Importing the contract registers nothing: with the mounts
	// removed, the server still builds and serves, but every action
	// path is an unregistered 404.
	mounts := filepath.Join(ws.server, "src/web/web.can")
	contractLiveReplaceOnce(t, mounts,
		"        call action::mount(contract::load_invoice_grid, callable load_grid) as http::route load\n        call action::mount(contract::save_invoice_grid, callable save_grid) as http::route save\n        call action::mount(contract::save_invoice_html, callable save_html, callable render_html, callable render_bad_form) as http::route form\n",
		"")
	contractLiveReplaceOnce(t, mounts,
		"call http::make_router([load, save, form, form_page, grid_page, probe]) as http::router built",
		"call http::make_router([form_page, grid_page, probe]) as http::router built")
	serverID, _, entry, _ := contractLiveBuildServer(t, ctx, bundle, ws, "")
	evidence = append(evidence, contractLiveEvidence{Edit: "no-register", Phase: "server-build", Detail: serverID})
	driver := filepath.Join(sourceRoot, "tests/integration/testdata/invoice/driver.ts")
	db := seedInvoiceDB(t, ctx, bundle, ws.home, driver, ws.server)
	base, stop := serveInvoice(t, ctx, bundle, ws.home, entry, db, contractLivePortNoRegister, "")
	defer stop()
	origin := "http://127.0.0.1:" + strconv.Itoa(contractLivePortNoRegister)

	status, body, _ := invoiceGet(t, base, "/api/tenants/1/invoices/7", "tok-alice")
	if status != 404 || body != "Not Found" {
		t.Fatalf("unmounted load: %d %q, want 404", status, body)
	}
	save := `{"operation_id":"op-n1","revision":"1","lines":[]}`
	status, body, _ = postInvoiceJSON(t, base, "/api/tenants/1/invoices/7", "tok-alice", origin, "application/json", save)
	if status != 404 || body != "Not Found" {
		t.Fatalf("unmounted save: %d %q, want 404", status, body)
	}
	status, body, _ = postInvoiceForm(t, base, "/tenants/1/invoices/7", "tok-alice", origin, contractFormValues("2", "", "1"))
	if status != 404 || body != "Not Found" {
		t.Fatalf("unmounted form save: %d %q, want 404", status, body)
	}
	status, probe, _ := invoiceGet(t, base, "/health", "")
	if status != 200 || probe != "ok" {
		t.Fatalf("health: %d %q, want the server alive", status, probe)
	}
	status, page, _ := invoiceGet(t, base, "/invoice-grid?tenant=1&invoice=7", "")
	if status != 200 || !strings.Contains(page, `id="invoice-grid"`) {
		t.Fatalf("grid shell: %d %.200s", status, page)
	}
	store := inspectInvoice(t, ctx, bundle, ws.home, driver, db)
	requireInvoice(t, store, "7", "1", "2", "Acme <em>&\" 'coop'\"")
	if len(store.Replay) != 0 {
		t.Fatalf("unmounted posts recorded %+v", store.Replay)
	}
	evidence = append(evidence, contractLiveEvidence{Edit: "no-register", Phase: "http", Detail: "all actions 404, health and shell 200, no effects"})
	logContractLive(t, evidence)
}
