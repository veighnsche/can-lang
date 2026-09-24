// Live qualification for the T15 server-rendered invoice form slice. The
// test stages examples/invoice, asserts and builds it twice with
// identical IDs, seeds a disposable SQLite file, and serves the built
// entry over loopback HTTP. TestInvoiceFormLive drives every documented
// form interaction over real HTTP and inspects real database rows:
// the edit page renders checked field names and escaped values, saves
// answer actual 200/422/409/403/503 fragments with truthful statuses,
// rejections never rewrite into success, denials disclose nothing, and
// an outage leg proves the 503 fragment plus recovery without wedged
// state. TestInvoiceBrowser qualifies the HTMX swap policy in a real
// browser: 200/422 swap into the status region with focus preserved
// and errors announced, while 409/403/503 never swap so the prior
// fragment and the form draft stay intact.
package integration

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/distribution"
)

const (
	invoiceHTTPPort    = 18551
	invoiceBrowserPort = 18552
)

type invoiceRow struct {
	ID        string `json:"id"`
	Tenant    string `json:"tenant"`
	Revision  string `json:"revision"`
	Customer  string `json:"customer"`
	UpdatedMs string `json:"updated_ms"`
}

type invoiceLine struct {
	InvoiceID string `json:"invoice_id"`
	LineKey   string `json:"line_key"`
	SKU       string `json:"sku"`
	Qty       string `json:"qty"`
	Position  string `json:"position"`
}

type invoiceReplay struct {
	Actor       string `json:"actor"`
	InvoiceID   string `json:"invoice_id"`
	OperationID string `json:"operation_id"`
	Digest      string `json:"digest"`
	Revision    string `json:"revision"`
	RecordedMs  string `json:"recorded_ms"`
}

type invoiceStore struct {
	Invoice []invoiceRow    `json:"invoice"`
	Lines   []invoiceLine   `json:"lines"`
	Replay  []invoiceReplay `json:"replay"`
}

func invoiceDriver(t *testing.T, ctx context.Context, bundle, home, driver string, args ...string) []byte {
	t.Helper()
	cmd := exec.CommandContext(ctx, filepath.Join(bundle, "runtime/bun"), append([]string{driver}, args...)...)
	cmd.Dir = home
	cmd.Env = []string{"PATH=/nonexistent", "HOME=" + home}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("invoice driver %v: %v %s", args, err, string(out))
	}
	return out
}

func inspectInvoice(t *testing.T, ctx context.Context, bundle, home, driver, db string) invoiceStore {
	t.Helper()
	out := invoiceDriver(t, ctx, bundle, home, driver, "inspect", db)
	var store invoiceStore
	if err := json.Unmarshal(out, &store); err != nil {
		t.Fatalf("invalid invoice inspect report %v %s", err, string(out))
	}
	return store
}

func invoiceGet(t *testing.T, base, path, session string) (int, string, http.Header) {
	t.Helper()
	client := &http.Client{Timeout: 10 * time.Second}
	request, err := http.NewRequest("GET", base+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if session != "" {
		request.AddCookie(&http.Cookie{Name: "session", Value: session})
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	return response.StatusCode, string(body), response.Header
}

func invoicePostForm(t *testing.T, base, path string, values url.Values) (int, string, http.Header) {
	t.Helper()
	client := &http.Client{Timeout: 10 * time.Second}
	response, err := client.PostForm(base+path, values)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	return response.StatusCode, string(body), response.Header
}

// stageInvoiceBundle stages examples/invoice without its regenerable
// dist output, asserts every Can root, and builds twice with identical
// IDs. It returns the staged root, home, driver, and entry path.
func stageInvoiceBundle(t *testing.T, ctx context.Context, bundle, sourceRoot string) (root, home, driver, entry string, assertions int) {
	t.Helper()
	root, home = stageApplication(t, ctx, bundle, sourceRoot, "invoice")
	status, out, diag := canlcOffline(t, ctx, bundle, home, "assert", root)
	if status != 0 || diag != "" {
		t.Fatalf("invoice assert: %d %s %s", status, out, diag)
	}
	var report struct {
		Passed     bool `json:"passed"`
		Assertions []struct {
			Evidence []string `json:"evidence"`
		} `json:"assertions"`
	}
	if err := json.Unmarshal([]byte(out), &report); err != nil || !report.Passed || len(report.Assertions) == 0 {
		t.Fatalf("invalid invoice assert report %v %s", err, out)
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
		t.Fatal("invoice asserts nothing real")
	}
	firstID, firstDir := applicationBuild(t, ctx, bundle, home, root)
	secondID, _ := applicationBuild(t, ctx, bundle, home, root)
	if firstID != secondID {
		t.Fatalf("invoice rebuild drifted: %s vs %s", firstID, secondID)
	}
	assertNoStrayEmit(t, root, firstDir)
	return root, home, filepath.Join(sourceRoot, "tests/integration/testdata/invoice/driver.ts"), filepath.Join(firstDir, "entry.ts"), len(report.Assertions)
}

func seedInvoiceDB(t *testing.T, ctx context.Context, bundle, home, driver, root string) string {
	t.Helper()
	db := filepath.Join(home, "inv.sqlite")
	setup := invoiceDriver(t, ctx, bundle, home, driver, "setup", db, filepath.Join(root, "schema.sql"))
	var setupReport struct {
		Tables int `json:"tables"`
	}
	if err := json.Unmarshal(setup, &setupReport); err != nil || setupReport.Tables != 5 {
		t.Fatalf("invalid invoice setup report %v %s", err, string(setup))
	}
	seed := invoiceDriver(t, ctx, bundle, home, driver, "seed", db)
	var seedReport struct {
		Sessions    int `json:"sessions"`
		Memberships int `json:"memberships"`
		Invoices    int `json:"invoices"`
		Lines       int `json:"lines"`
	}
	if err := json.Unmarshal(seed, &seedReport); err != nil || seedReport.Sessions != 3 || seedReport.Invoices != 2 || seedReport.Lines != 2 {
		t.Fatalf("invalid invoice seed report %v %s", err, string(seed))
	}
	return db
}

func requireInvoiceRevision(t *testing.T, store invoiceStore, id, revision, customer string) {
	t.Helper()
	for _, row := range store.Invoice {
		if row.ID != id {
			continue
		}
		if row.Revision != revision || row.Customer != customer {
			t.Fatalf("invoice %s = rev %s customer %q, want rev %s customer %q", id, row.Revision, row.Customer, revision, customer)
		}
		return
	}
	t.Fatalf("invoice %s missing from %+v", id, store.Invoice)
}

func TestInvoiceFormLive(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged invoice execution")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "invoice-form")
	if err != nil {
		t.Fatal(err)
	}
	root, home, driver, entry, assertions := stageInvoiceBundle(t, ctx, bundle, sourceRoot)
	db := seedInvoiceDB(t, ctx, bundle, home, driver, root)
	snapshot := snapshotCredential(t, home, "INVOICE_DB", db)
	base, stop := serveApplication(t, ctx, bundle, home, entry, snapshot, invoiceHTTPPort, "/health")
	defer stop()

	if status, body, _ := invoiceGet(t, base, "/health", ""); status != 200 || body != "ok" {
		t.Fatalf("health: %d %q", status, body)
	}

	// The edit page renders checked names, the checked save URL, HTMX
	// wiring, and escaped stored values.
	status, page, headers := invoiceGet(t, base, "/invoices/form?invoice_id=inv-1", "tok-alice")
	if status != 200 {
		t.Fatalf("form page: %d %s", status, page)
	}
	if content := headers.Get("Content-Type"); !strings.HasPrefix(content, "text/html") {
		t.Fatalf("form page content type %q", content)
	}
	for _, want := range []string{
		`id="invoice_form"`, `hx-post="/invoices/save-form"`, `hx-target="#invoice_status"`,
		`hx-swap="innerHTML"`, `id="invoice_status"`, `name="customer"`, `name="session_token"`,
		`name="operation_id"`, `name="revision"`, `name="lines_order"`, `name="lines[k1][sku]"`,
		`name="lines[k1][qty]"`, `name="lines[k2][sku]"`, `id="invoice_customer"`,
		`/__can/assets/htmx-4.0.0.min.js`, `htmx-config`, `Acme &lt;em&gt;`, `sku-1 &lt;b&gt;`,
	} {
		if !strings.Contains(page, want) {
			t.Fatalf("form page lacks %q in %.500s", want, page)
		}
	}
	for _, hostile := range []string{`<em>`, `<b>`, `<script>`} {
		if strings.Contains(page, hostile) {
			t.Fatalf("form page leaks raw %q", hostile)
		}
	}

	valid := func(operation, revision, customer string) url.Values {
		values := url.Values{}
		values.Set("session_token", "tok-alice")
		values.Set("operation_id", operation)
		values.Set("invoice_id", "inv-1")
		values.Set("revision", revision)
		values.Set("customer", customer)
		values.Add("lines_order", "k1")
		values.Add("lines_order", "k2")
		values.Set("lines[k1][sku]", "sku-9")
		values.Set("lines[k1][qty]", "4")
		values.Set("lines[k2][sku]", "sku-2")
		values.Set("lines[k2][qty]", "1")
		return values
	}
	save := func(note string, values url.Values, want int) (string, http.Header) {
		t.Helper()
		status, body, headers := invoicePostForm(t, base, "/invoices/save-form", values)
		if status != want {
			t.Fatalf("%s: %d %s, want %d", note, status, body, want)
		}
		return body, headers
	}

	// A valid save answers its 200 fragment and rewrites the stored
	// invoice, lines, and replay ledger in one transaction.
	body, headers := save("save", valid("op-live-1", "1", "Acme Live"), 200)
	if !strings.Contains(body, `saved inv-1 revision 2`) || !strings.Contains(body, `role="status"`) {
		t.Fatalf("save fragment %q", body)
	}
	if content := headers.Get("Content-Type"); !strings.HasPrefix(content, "text/html") {
		t.Fatalf("save content type %q", content)
	}
	if headers.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("save lacks nosniff")
	}
	store := inspectInvoice(t, ctx, bundle, home, driver, db)
	requireInvoiceRevision(t, store, "inv-1", "2", "Acme Live")
	if len(store.Lines) != 2 || store.Lines[0].SKU != "sku-9" || store.Lines[0].Qty != "4" || store.Lines[1].SKU != "sku-2" {
		t.Fatalf("saved lines %+v", store.Lines)
	}
	if len(store.Replay) != 1 || store.Replay[0].OperationID != "op-live-1" || store.Replay[0].Revision != "2" || len(store.Replay[0].Digest) != 64 {
		t.Fatalf("saved replay %+v", store.Replay)
	}

	// The identical retry replays the stored revision without a second
	// effect; the same operation with changed content conflicts.
	body, _ = save("replay", valid("op-live-1", "1", "Acme Live"), 200)
	if !strings.Contains(body, `saved inv-1 revision 2`) {
		t.Fatalf("replay fragment %q", body)
	}
	store = inspectInvoice(t, ctx, bundle, home, driver, db)
	requireInvoiceRevision(t, store, "inv-1", "2", "Acme Live")
	if len(store.Replay) != 1 {
		t.Fatalf("replay duplicated %+v", store.Replay)
	}
	body, _ = save("changed replay", valid("op-live-1", "1", "Acme Changed"), 409)
	if !strings.Contains(body, `stale inv-1 revision 2`) || !strings.Contains(body, `role="alert"`) {
		t.Fatalf("conflict fragment %q", body)
	}

	// A stale base revision answers a truthful 409 carrying the current
	// revision; the store keeps the committed content.
	body, _ = save("stale", valid("op-live-2", "1", "Stale Attempt"), 409)
	if !strings.Contains(body, `stale inv-1 revision 2`) {
		t.Fatalf("stale fragment %q", body)
	}
	store = inspectInvoice(t, ctx, bundle, home, driver, db)
	requireInvoiceRevision(t, store, "inv-1", "2", "Acme Live")

	// Failed validation answers 422 with an announcing fragment and
	// writes nothing; the stored draft is untouched.
	rejected := valid("op-live-3", "2", "")
	body, _ = save("rejected", rejected, 422)
	if !strings.Contains(body, `rejected: empty customer`) || !strings.Contains(body, `role="alert"`) {
		t.Fatalf("rejected fragment %q", body)
	}
	store = inspectInvoice(t, ctx, bundle, home, driver, db)
	requireInvoiceRevision(t, store, "inv-1", "2", "Acme Live")
	if len(store.Replay) != 1 {
		t.Fatalf("rejection recorded %+v", store.Replay)
	}

	// Structural violations answer 422 with the issue list plus the
	// retained known raw pairs, escaped; hostile text never renders raw.
	structural := valid("op-live-4", "2", "</li><script>bad()</script>")
	structural["lines_order"] = []string{"k1", "k1"}
	body, _ = save("structural", structural, 422)
	for _, want := range []string{`invoice form rejected`, `lines_order: form_repeated`, `role="alert"`, `&lt;/li&gt;&lt;script&gt;`} {
		if !strings.Contains(body, want) {
			t.Fatalf("structural fragment lacks %q in %q", want, body)
		}
	}
	if strings.Contains(body, "<script>") {
		t.Fatalf("structural fragment leaks raw script in %q", body)
	}

	// Unknown names are violations whose values never enter the retained
	// raw text: the issue names the field but the payload stays out.
	unknown := valid("op-live-5", "2", "Known")
	unknown.Set("evil", "<script>nope</script>")
	body, _ = save("unknown", unknown, 422)
	if !strings.Contains(body, `evil: type`) {
		t.Fatalf("unknown fragment %q", body)
	}
	if strings.Contains(body, "nope") || strings.Contains(body, "<script>") {
		t.Fatalf("unknown value leaked into %q", body)
	}

	// Foreign and nonexistent targets share one nondisclosing 403 that
	// echoes only the caller-sent id: no customer, tenant, or revision.
	foreign := valid("op-live-6", "1", "X")
	foreign.Set("invoice_id", "inv-2")
	body, _ = save("foreign", foreign, 403)
	if !strings.Contains(body, `denied inv-2`) || strings.Contains(body, "Globex") || strings.Contains(body, "tenant-b") {
		t.Fatalf("foreign fragment %q", body)
	}
	missing := valid("op-live-7", "1", "X")
	missing.Set("invoice_id", "inv-9")
	body, _ = save("missing", missing, 403)
	if !strings.Contains(body, `denied inv-9`) || !strings.Contains(body, `save-denied`) {
		t.Fatalf("missing fragment %q", body)
	}
	store = inspectInvoice(t, ctx, bundle, home, driver, db)
	requireInvoiceRevision(t, store, "inv-1", "2", "Acme Live")
	requireInvoiceRevision(t, store, "inv-2", "1", "Globex")

	// The edit page denies ghost, missing, and anonymous loads with the
	// same nondisclosing shape and no stored content.
	for _, leg := range []struct {
		note, session, id string
	}{
		{"ghost page", "tok-ghost", "inv-1"},
		{"missing page", "tok-alice", "inv-9"},
		{"anonymous page", "", "inv-1"},
	} {
		status, denied, _ := invoiceGet(t, base, "/invoices/form?invoice_id="+leg.id, leg.session)
		if status != 403 || !strings.Contains(denied, "denied "+leg.id) {
			t.Fatalf("%s: %d %q", leg.note, status, denied)
		}
		if strings.Contains(denied, "Acme") || strings.Contains(denied, "Globex") || strings.Contains(denied, "tenant-") {
			t.Fatalf("%s leaks stored content in %q", leg.note, denied)
		}
	}

	// A store outage turns saves into truthful 503 busy fragments; the
	// identical form retries safely once access returns.
	if err := os.Chmod(db, 0); err != nil {
		t.Fatal(err)
	}
	func() {
		defer func() {
			if err := os.Chmod(db, 0600); err != nil {
				t.Fatal(err)
			}
		}()
		body, _ := save("outage", valid("op-live-8", "2", "Outage Attempt"), 503)
		if !strings.Contains(body, `busy inv-1`) || !strings.Contains(body, `role="alert"`) {
			t.Fatalf("busy fragment %q", body)
		}
	}()
	body, _ = save("recovery", valid("op-live-8", "2", "Recovered"), 200)
	if !strings.Contains(body, `saved inv-1 revision 3`) {
		t.Fatalf("recovery fragment %q", body)
	}
	store = inspectInvoice(t, ctx, bundle, home, driver, db)
	requireInvoiceRevision(t, store, "inv-1", "3", "Recovered")

	t.Logf("invoice form: %d assertions, build twice identical, 200/422/409/403/503 fragments live with row evidence", assertions)
}

// TestInvoiceBrowser qualifies the invoice HTMX swap policy in a real
// browser: 200/422 responses swap into the status region with focus
// preserved and errors announced, while 409/403/503 responses never
// swap so the prior fragment and the form draft stay intact. A final
// target-absence leg removes the swap target and proves the page stays
// stable, then restores it and commits a marker save the test reads
// back from the database.
func TestInvoiceBrowser(t *testing.T) {
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
	ctx, cancel := context.WithTimeout(context.Background(), 14*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "invoice-browser")
	if err != nil {
		t.Fatal(err)
	}
	root, home, driver, entry, assertions := stageInvoiceBundle(t, ctx, bundle, sourceRoot)
	db, err := filepath.EvalSymlinks(filepath.Join(home, "inv.sqlite"))
	if err != nil {
		// The file does not exist until the driver creates it; resolve
		// the home directory instead so the browser leg receives a
		// symlink-free database path.
		resolvedHome, resolveErr := filepath.EvalSymlinks(home)
		if resolveErr != nil {
			t.Fatal(resolveErr)
		}
		db = filepath.Join(resolvedHome, "inv.sqlite")
	}
	setup := invoiceDriver(t, ctx, bundle, home, driver, "setup", db, filepath.Join(root, "schema.sql"))
	var setupReport struct {
		Tables int `json:"tables"`
	}
	if err := json.Unmarshal(setup, &setupReport); err != nil || setupReport.Tables != 5 {
		t.Fatalf("invalid invoice setup report %v %s", err, string(setup))
	}
	invoiceDriver(t, ctx, bundle, home, driver, "seed", db)
	snapshot := snapshotCredential(t, home, "INVOICE_DB", db)
	base, stop := serveApplication(t, ctx, bundle, home, entry, snapshot, invoiceBrowserPort, "/health")
	defer stop()

	outdir := t.TempDir()
	browser := exec.CommandContext(ctx, nodePath, "invoice.mjs", base, outdir, db)
	browser.Dir = browserDir
	browser.Env = []string{"PATH=" + filepath.Dir(nodePath) + ":/usr/bin:/bin", "HOME=" + os.Getenv("HOME")}
	result, err := browser.CombinedOutput()
	if err != nil {
		t.Fatalf("invoice harness: %v %s", err, result)
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
	if err := json.Unmarshal(raw, &report); err != nil || !report.Passed || len(report.Checks) != 13 {
		t.Fatalf("invoice invalid browser report %v %s", err, raw)
	}
	for _, entry := range report.Requests {
		if !strings.HasPrefix(entry.URL, base+"/") {
			t.Fatalf("invoice browser left loopback: %s", entry.URL)
		}
	}
	shot, err := os.Stat(filepath.Join(outdir, "screenshot.png"))
	if err != nil || shot.Size() == 0 {
		t.Fatal("invoice missing browser screenshot")
	}
	if evidence := os.Getenv("CAN_BROWSER_EVIDENCE_DIR"); evidence != "" {
		dest := filepath.Join(evidence, "invoice")
		if err := os.MkdirAll(dest, 0700); err != nil {
			t.Fatal(err)
		}
		copyEvidenceFile(t, filepath.Join(outdir, "report.json"), filepath.Join(dest, "report.json"))
		copyEvidenceFile(t, filepath.Join(outdir, "screenshot.png"), filepath.Join(dest, "screenshot.png"))
	}
	store := inspectInvoice(t, ctx, bundle, home, driver, db)
	requireInvoiceRevision(t, store, "inv-1", "3", "Target Absent Save")
	t.Logf("invoice browser %s %s: %d checks, %d loopback requests, %d can assertions",
		report.Browser, report.Version, len(report.Checks), len(report.Requests), assertions)
}
