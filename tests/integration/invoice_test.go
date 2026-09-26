// UP16 invoice slice over the shared invoice_contract package, extended
// by UP22 with the live server/database/lifecycle matrix. The compiled
// program serves every behavior itself against real SQLite: setup and
// seed prepare the operator-owned file, HTTP legs drive the three
// mounted contract actions, and inspect/touch-replay/revoke/member/
// line modes read back or mutate rows the test asserts on. Live legs
// cover the form page, both save channels, replay
// (same/changed/expired/revoked), validation, denials, the exact-Origin
// table, retention cleanup, startup refusal, outage and recovery, HTML
// fragment bytes, adapter failures, concurrent revision/membership,
// expiry/revision invariants, lost acknowledgement, post-commit
// renderer faults, pool/shutdown lifecycle and token revocation;
// boundary-exact expiry and unit retention rules are pinned by model
// assertions (save_replay_at_edge, save_no_window), and the live legs
// prove the same code paths against real SQLite state. Browser DOM
// guard verdicts arrive from UP23 through TestInvoiceBrowserGuardDOM.
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	invoiceLivePort     = 18551
	invoiceBrowserPort  = 18552
	invoicePairedPort   = 18553
	invoiceFragmentPort = 18576
)

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
	payload, _ := io.ReadAll(response.Body)
	return response.StatusCode, string(payload), response.Header
}

// postInvoiceJSON posts one JSON body with an explicit session cookie
// and Origin value. Empty session omits the cookie; origin "omit"
// omits the header so the missing-origin leg stays exact.
func postInvoiceJSON(t *testing.T, base, path, session, origin, media, body string) (int, string, http.Header) {
	t.Helper()
	client := &http.Client{Timeout: 10 * time.Second}
	request, err := http.NewRequest("POST", base+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", media)
	if origin != "omit" {
		request.Header.Set("Origin", origin)
	}
	if session != "" {
		request.AddCookie(&http.Cookie{Name: "session", Value: session})
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	payload, _ := io.ReadAll(response.Body)
	return response.StatusCode, string(payload), response.Header
}

// postInvoiceForm posts one form body with an explicit session cookie
// and Origin value. Empty session omits the cookie; origin "omit"
// omits the header.
func postInvoiceForm(t *testing.T, base, path, session, origin string, values url.Values) (int, string, http.Header) {
	t.Helper()
	client := &http.Client{Timeout: 10 * time.Second}
	request, err := http.NewRequest("POST", base+path, strings.NewReader(values.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if origin != "omit" {
		request.Header.Set("Origin", origin)
	}
	if session != "" {
		request.AddCookie(&http.Cookie{Name: "session", Value: session})
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	payload, _ := io.ReadAll(response.Body)
	return response.StatusCode, string(payload), response.Header
}

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

func invoiceDriver(t *testing.T, ctx context.Context, bundle, home, driver string, args ...string) []byte {
	t.Helper()
	out, err := invoiceDriverRaw(ctx, bundle, home, driver, args...)
	if err != nil {
		t.Fatalf("driver %v: %v %s", args, err, string(out))
	}
	return out
}

// invoiceDriverRaw runs one driver mode and reports its output, for
// goroutines that cannot fail the test directly.
func invoiceDriverRaw(ctx context.Context, bundle, home, driver string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, filepath.Join(bundle, "runtime/bun"), append([]string{driver}, args...)...)
	cmd.Dir = home
	cmd.Env = []string{"PATH=/nonexistent", "HOME=" + home}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, err
	}
	return out, nil
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

type invoiceLine struct {
	Key      string `json:"key"`
	ID       string `json:"id"`
	Quantity string `json:"quantity"`
	Price    string `json:"price_minor"`
	Position string `json:"position"`
}

type invoiceRow struct {
	Tenant  string        `json:"tenant"`
	ID      string        `json:"id"`
	Rev     string        `json:"revision"`
	Seats   string        `json:"seats"`
	Details string        `json:"details"`
	Lines   []invoiceLine `json:"lines"`
}

type invoiceReplayRow struct {
	Actor       string `json:"actor"`
	Tenant      string `json:"tenant"`
	InvoiceID   string `json:"invoice"`
	OperationID string `json:"operation"`
	Digest      string `json:"digest"`
	Result      string `json:"result"`
	Revision    string `json:"revision"`
	Recorded    string `json:"recorded"`
}

// invoiceFlatLine is the pre-UP16 top-level line projection kept for
// the gate 4/5 suites: the UP16 driver nests lines under their invoice
// and inspectInvoice flattens them back into store.Lines with the line
// identifier in the SKU slot. The gate suites still target the old
// string-key domain and routes, so they fail at runtime until UP22/23
// migrate them; this projection keeps the package compiling so every
// suite can run.
type invoiceFlatLine struct {
	InvoiceID string
	LineKey   string
	SKU       string
	Qty       string
	Position  string
}

type invoiceStore struct {
	Invoice []invoiceRow       `json:"invoices"`
	Replay  []invoiceReplayRow `json:"replays"`
	Lines   []invoiceFlatLine  `json:"-"`
}

func inspectInvoice(t *testing.T, ctx context.Context, bundle, home, driver, db string) invoiceStore {
	t.Helper()
	out := invoiceDriver(t, ctx, bundle, home, driver, "inspect", db)
	var store invoiceStore
	if err := json.Unmarshal(out, &store); err != nil {
		t.Fatalf("invalid invoice inspect report %v %s", err, string(out))
	}
	for _, row := range store.Invoice {
		for _, line := range row.Lines {
			store.Lines = append(store.Lines, invoiceFlatLine{
				InvoiceID: row.ID,
				LineKey:   line.Key,
				SKU:       line.ID,
				Qty:       line.Quantity,
				Position:  line.Position,
			})
		}
	}
	return store
}

// invoiceRevision reads one invoice revision without touching the
// replay ledger, so mid-fault legs can prove no commit while the
// ledger table itself is dropped.
func invoiceRevision(t *testing.T, ctx context.Context, bundle, home, driver, db, invoice string) string {
	t.Helper()
	out := invoiceDriver(t, ctx, bundle, home, driver, "revision", db, invoice)
	var report struct {
		Invoice  string `json:"invoice"`
		Revision string `json:"revision"`
	}
	if err := json.Unmarshal(out, &report); err != nil || report.Invoice != invoice || report.Revision == "" {
		t.Fatalf("invalid revision report %v %s", err, string(out))
	}
	return report.Revision
}

func touchReplay(t *testing.T, ctx context.Context, bundle, home, driver, db, actor, tenant, invoice, operation, recorded string) {
	t.Helper()
	out := invoiceDriver(t, ctx, bundle, home, driver, "touch-replay", db, actor, tenant, invoice, operation, recorded)
	var report struct {
		Recorded string `json:"recorded"`
	}
	if err := json.Unmarshal(out, &report); err != nil || report.Recorded != recorded {
		t.Fatalf("invalid touch-replay report %v %s", err, string(out))
	}
}

func faultInvoice(t *testing.T, ctx context.Context, bundle, home, driver, db, fault string) {
	t.Helper()
	out := invoiceDriver(t, ctx, bundle, home, driver, "fault", db, fault)
	var report struct {
		Fault string `json:"fault"`
	}
	if err := json.Unmarshal(out, &report); err != nil || report.Fault != fault {
		t.Fatalf("invalid fault report %v %s", err, string(out))
	}
}

// revokeSession sets one session row's revoked stamp (0 restores it)
// and requires the stored value back, for revoked-replay legs.
func revokeSession(t *testing.T, ctx context.Context, bundle, home, driver, db, token, stamp string) {
	t.Helper()
	out := invoiceDriver(t, ctx, bundle, home, driver, "revoke", db, token, stamp)
	var report struct {
		Token   string `json:"token"`
		Revoked string `json:"revoked"`
	}
	if err := json.Unmarshal(out, &report); err != nil || report.Token != token || report.Revoked != stamp {
		t.Fatalf("invalid revoke report %v %s", err, string(out))
	}
}

// setMembership adds (present true) or removes (present false) one
// membership row, for concurrent-membership legs.
func setMembership(t *testing.T, ctx context.Context, bundle, home, driver, db, actor, tenant string, present bool) {
	t.Helper()
	mode := "unmember"
	if present {
		mode = "member"
	}
	out := invoiceDriver(t, ctx, bundle, home, driver, mode, db, actor, tenant)
	var report struct {
		Actor  string `json:"actor"`
		Tenant string `json:"tenant"`
	}
	if err := json.Unmarshal(out, &report); err != nil || report.Actor != actor || report.Tenant != tenant {
		t.Fatalf("invalid %s report %v %s", mode, err, string(out))
	}
}

// injectLine stores one line row exactly as given, including keys the
// page renderer cannot mint, for renderer-fault legs.
func injectLine(t *testing.T, ctx context.Context, bundle, home, driver, db, invoice, key, id, quantity, price, position string) {
	t.Helper()
	out := invoiceDriver(t, ctx, bundle, home, driver, "inject-line", db, invoice, key, id, quantity, price, position)
	var report struct {
		Invoice string `json:"invoice"`
		Key     string `json:"key"`
	}
	if err := json.Unmarshal(out, &report); err != nil || report.Invoice != invoice || report.Key != key {
		t.Fatalf("invalid inject-line report %v %s", err, string(out))
	}
}

// deleteLine removes one stored line row, restoring the renderer-fault
// database to its servable shape.
func deleteLine(t *testing.T, ctx context.Context, bundle, home, driver, db, invoice, key string) {
	t.Helper()
	out := invoiceDriver(t, ctx, bundle, home, driver, "delete-line", db, invoice, key)
	var report struct {
		Invoice string `json:"invoice"`
		Key     string `json:"key"`
	}
	if err := json.Unmarshal(out, &report); err != nil || report.Invoice != invoice || report.Key != key {
		t.Fatalf("invalid delete-line report %v %s", err, string(out))
	}
}

// protectedEntries snapshots the observable protected-handler entry
// count: one invoice revision plus the replay ledger length. Adapter
// rejections must leave both unchanged; each committed save advances
// both by exactly one.
type protectedEntries struct {
	revision string
	replays  int
}

func snapshotEntries(store invoiceStore, id string) protectedEntries {
	revision := ""
	for _, row := range store.Invoice {
		if row.ID == id {
			revision = row.Rev
		}
	}
	return protectedEntries{revision: revision, replays: len(store.Replay)}
}

func requireNoEntry(t *testing.T, note string, before, after protectedEntries) {
	t.Helper()
	if before != after {
		t.Fatalf("%s: protected entries moved from %+v to %+v, want no handler entry", note, before, after)
	}
}

func requireOneEntry(t *testing.T, note, wantRev string, before, after protectedEntries) {
	t.Helper()
	if after.revision != wantRev || after.replays != before.replays+1 {
		t.Fatalf("%s: protected entries moved from %+v to %+v, want revision %s and one ledger row", note, before, after, wantRev)
	}
}

// recordFaultOutcome logs whether a fault leg committed a write, so
// every injected fault carries its commit verdict in the test log.
func recordFaultOutcome(t *testing.T, note string, before, after protectedEntries) bool {
	t.Helper()
	committed := before != after
	t.Logf("fault %s: committed=%v (entries %+v -> %+v)", note, committed, before, after)
	return committed
}

func requireInvoice(t *testing.T, store invoiceStore, id, revision, seats, details string) invoiceRow {
	t.Helper()
	for _, row := range store.Invoice {
		if row.ID != id {
			continue
		}
		if row.Rev != revision || row.Seats != seats || row.Details != details {
			t.Fatalf("invoice %s = rev %s seats %s details %q, want rev %s seats %s details %q", id, row.Rev, row.Seats, row.Details, revision, seats, details)
		}
		return row
	}
	t.Fatalf("invoice %s missing from %+v", id, store.Invoice)
	return invoiceRow{}
}

func requireReplay(t *testing.T, store invoiceStore, operation, revision string) invoiceReplayRow {
	t.Helper()
	for _, row := range store.Replay {
		if row.OperationID != operation {
			continue
		}
		if row.Revision != revision || len(row.Digest) != 64 || row.Result == "" {
			t.Fatalf("replay %s = %+v, want revision %s with digest and result", operation, row, revision)
		}
		return row
	}
	t.Fatalf("replay %s missing from %+v", operation, store.Replay)
	return invoiceReplayRow{}
}

// requireInvoiceRevision is the pre-UP16 invoice check kept for the
// gate 4/5 suites: the legacy customer slot maps onto the UP16 details
// text. It fails honestly against the new integer-key domain until
// UP22/23 migrate those suites.
func requireInvoiceRevision(t *testing.T, store invoiceStore, id, revision, customer string) {
	t.Helper()
	for _, row := range store.Invoice {
		if row.ID != id {
			continue
		}
		if row.Rev != revision || row.Details != customer {
			t.Fatalf("invoice %s = rev %s details %q, want rev %s details %q", id, row.Rev, row.Details, revision, customer)
		}
		return
	}
	t.Fatalf("invoice %s missing from %+v", id, store.Invoice)
}

// invoicePostForm is the pre-UP16 sessionless form post kept for the
// gate 4/5 suites. New suites use postInvoiceForm with an explicit
// session and origin.
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

// serveInvoice runs a staged invoice entry with INVOICE_DB,
// PUBLIC_ORIGIN and an optional retention-window override in its
// fd-3 snapshot, and returns the base URL with a clean stopper.
func serveInvoice(t *testing.T, ctx context.Context, bundle, home, entry, db string, port int, window string) (string, func()) {
	t.Helper()
	values := map[string]string{
		"INVOICE_DB":    db,
		"PUBLIC_ORIGIN": "http://127.0.0.1:" + strconv.Itoa(port),
	}
	if window != "" {
		values["INVOICE_REPLAY_WINDOW_MS"] = window
	}
	snapshot := snapshotMap(t, home, "snapshot-invoice", values)
	return serveApplication(t, ctx, bundle, home, entry, snapshot, port, "/health")
}

// expectInvoiceStartupFailure runs a staged invoice entry that must
// exit nonzero before serving: malformed origin or window, or a
// missing origin, fails closed instead of serving.
func expectInvoiceStartupFailure(t *testing.T, ctx context.Context, bundle, home, entry string, values map[string]string, port int, note string) {
	t.Helper()
	snapshot := snapshotMap(t, home, "snapshot-"+note, values)
	file, err := os.Open(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, filepath.Join(bundle, "runtime/bun"), entry, strconv.Itoa(port))
	cmd.Dir = home
	cmd.Env = []string{"PATH=/nonexistent", "HOME=" + home}
	cmd.ExtraFiles = []*os.File{file}
	var out, diag strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &diag
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatalf("%s: startup unexpectedly succeeded", note)
		}
	case <-time.After(20 * time.Second):
		cmd.Process.Kill()
		t.Fatalf("%s: startup neither served nor failed", note)
	}
}

func TestInvoiceFormLive(t *testing.T) {
	t.Parallel()
	acquireHeavy(t)
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged invoice execution")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()
	bundle, err := harnessBundle(t, ctx, archive)
	if err != nil {
		t.Fatal(err)
	}
	root, home, driver, entry, assertions := stageInvoiceBundle(t, ctx, bundle, sourceRoot)
	db := seedInvoiceDB(t, ctx, bundle, home, driver, root)
	base, stop := serveInvoice(t, ctx, bundle, home, entry, db, invoiceLivePort, "")
	defer stop()
	origin := "http://127.0.0.1:" + strconv.Itoa(invoiceLivePort)

	jsonLines := `[{"key":"k1","id":"a","quantity":"2","price":"5.00"},{"key":"k2","id":"b","quantity":"1","price":"1.99"}]`
	saveBody := func(op string, rev int) string {
		return `{"operation_id":"` + op + `","revision":"` + strconv.Itoa(rev) + `","lines":` + jsonLines + `}`
	}
	formValues := func(seats, details, revision string) url.Values {
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

	// Form page: checked fields, keyed rows in stored order, hostile
	// text escaped, post URL from the shared action, and no hidden
	// session, operation or invoice fields anywhere.
	status, page, _ := invoiceGet(t, base, "/invoices/form?tenant_id=1&invoice_id=7", "tok-alice")
	if status != 200 {
		t.Fatalf("form page: %d %s", status, page)
	}
	for _, want := range []string{
		`<title>Invoice 1/7</title>`,
		`id="invoice_form"`,
		`hx-post="/tenants/1/invoices/7"`,
		`name="seats"`, `value="2"`,
		`name="details"`,
		`name="revision"`, `value="1"`,
		`name="lines_order"`, `value="k1"`, `value="k2"`,
		`name="lines[k1][id]"`, `name="lines[k1][quantity]"`, `name="lines[k1][price]"`,
		`value="19.99"`, `value="5.00"`,
		`id="invoice_status"`, `No save yet.`,
	} {
		if !strings.Contains(page, want) {
			t.Fatalf("form page lacks %q in %.1500s", want, page)
		}
	}
	if strings.Count(page, `name="lines_order"`) != 2 {
		t.Fatalf("form page order inputs: %.800s", page)
	}
	if strings.Index(page, `value="k1"`) > strings.Index(page, `value="k2"`) {
		t.Fatalf("form page scrambles stored row order in %.800s", page)
	}
	if strings.Contains(page, "sku-1 <b>") || strings.Contains(page, `Acme <em>`) {
		t.Fatalf("form page leaks hostile text in %.1500s", page)
	}
	if !strings.Contains(page, "&lt;em&gt;") || !strings.Contains(page, "&lt;b&gt;") {
		t.Fatalf("form page lacks escaped hostile text in %.1500s", page)
	}
	for _, hidden := range []string{`name="session_token"`, `name="operation_id"`, `name="invoice_id"`} {
		if strings.Contains(page, hidden) {
			t.Fatalf("form page carries hidden %s in %.1500s", hidden, page)
		}
	}
	// Swap admission: the served form carries per-element hx-status
	// exceptions generated from the checked HTML action case table,
	// so declared 403/409/422/503 fragments swap into the status
	// region while 2xx needs no exception.
	for _, admission := range []string{
		`hx-status:403="{&quot;swap&quot;:&quot;innerHTML&quot;}"`,
		`hx-status:409="{&quot;swap&quot;:&quot;innerHTML&quot;}"`,
		`hx-status:422="{&quot;swap&quot;:&quot;innerHTML&quot;}"`,
		`hx-status:503="{&quot;swap&quot;:&quot;innerHTML&quot;}"`,
	} {
		if !strings.Contains(page, admission) {
			t.Fatalf("form page lacks %s in %.1500s", admission, page)
		}
	}
	if strings.Contains(page, "hx-status:200") {
		t.Fatalf("form page admits a redundant 2xx exception in %.1500s", page)
	}

	// Form page denials: ghost, missing and anonymous readers share 403.
	for _, leg := range []struct {
		note, query, session string
	}{
		{"page ghost", "tenant_id=1&invoice_id=7", "tok-ghost"},
		{"page missing", "tenant_id=1&invoice_id=9", "tok-alice"},
		{"page anonymous", "tenant_id=1&invoice_id=7", ""},
	} {
		status, body, _ := invoiceGet(t, base, "/invoices/form?"+leg.query, leg.session)
		if status != 403 || !strings.Contains(body, "denied") {
			t.Fatalf("%s: %d %s, want 403 denied", leg.note, status, body)
		}
	}
	if status, body, _ := invoiceGet(t, base, "/invoices/form?tenant_id=1x&invoice_id=7", "tok-alice"); status != 400 {
		t.Fatalf("page bad tenant: %d %s, want 400", status, body)
	}

	// Grid shell: a static public document with the application mount
	// point, served without parsing or authentication. Anonymous and
	// garbage-keyed reads answer the identical bytes: the browser
	// application validates its own keys and renders API denials
	// itself. The unpaired build carries no script tag at all.
	var shells []string
	for _, leg := range []struct {
		note, query, session string
	}{
		{"grid shell", "tenant=1&invoice=7", "tok-alice"},
		{"grid anonymous", "tenant=1&invoice=7", ""},
		{"grid garbage", "tenant=x&invoice=y", ""},
	} {
		status, shell, headers := invoiceGet(t, base, "/invoice-grid?"+leg.query, leg.session)
		if status != 200 {
			t.Fatalf("%s: %d, want 200", leg.note, status)
		}
		for _, want := range []string{
			`<title>Invoice grid</title>`,
			`id="invoice-grid"`,
			`Loading invoice grid`,
		} {
			if !strings.Contains(shell, want) {
				t.Fatalf("%s lacks %q in %.800s", leg.note, want, shell)
			}
		}
		if strings.Contains(shell, "hx-post") || strings.Contains(shell, "<script") {
			t.Fatalf("%s carries form or script content in %.800s", leg.note, shell)
		}
		if content := headers.Get("Content-Security-Policy"); !strings.Contains(content, "script-src 'self'") {
			t.Fatalf("%s CSP %q lacks same-origin scripts", leg.note, content)
		}
		shells = append(shells, shell)
	}
	if shells[0] != shells[1] || shells[0] != shells[2] {
		t.Fatal("grid shell varies by session or query keys")
	}

	// Form save through the shared protected write: revision 2 with
	// seats, details and replaced lines.
	status, body, _ := postInvoiceForm(t, base, "/tenants/1/invoices/7", "tok-alice", origin, formValues("4", "Rush order", "1"))
	if status != 200 || !strings.Contains(body, "saved revision 2") {
		t.Fatalf("form save: %d %s, want 200 saved revision 2", status, body)
	}
	if !strings.Contains(body, `role="status"`) {
		t.Fatalf("form save lacks polite status in %q", body)
	}
	store := inspectInvoice(t, ctx, bundle, home, driver, db)
	row := requireInvoice(t, store, "7", "2", "4", "Rush order")
	if len(row.Lines) != 2 || row.Lines[0].Key != "k1" || row.Lines[0].Price != "500" || row.Lines[1].Key != "k2" || row.Lines[1].Price != "199" {
		t.Fatalf("form save lines %+v", row.Lines)
	}
	if len(store.Replay) != 1 {
		t.Fatalf("form save replay %+v, want one minted row", store.Replay)
	}

	// Form validation, staleness and structure: 422/409 with the
	// draft retained in the region, and no store effect.
	status, body, _ = postInvoiceForm(t, base, "/tenants/1/invoices/7", "tok-alice", origin, formValues("many", "", "2"))
	if status != 422 || !strings.Contains(body, "invoice invalid") || !strings.Contains(body, "seats: bad seats") || !strings.Contains(body, `role="alert"`) {
		t.Fatalf("form invalid: %d %s, want 422 alert", status, body)
	}
	status, body, _ = postInvoiceForm(t, base, "/tenants/1/invoices/7", "tok-alice", origin, formValues("4", "", "1"))
	if status != 409 || !strings.Contains(body, "conflict: stale revision") {
		t.Fatalf("form stale: %d %s, want 409 conflict", status, body)
	}
	broken := formValues("4", "", "2")
	broken.Del("seats")
	status, body, _ = postInvoiceForm(t, base, "/tenants/1/invoices/7", "tok-alice", origin, broken)
	if status != 422 || !strings.Contains(body, "invoice form rejected") || !strings.Contains(body, `role="alert"`) {
		t.Fatalf("form structural: %d %s, want 422 rejected alert", status, body)
	}
	store = inspectInvoice(t, ctx, bundle, home, driver, db)
	requireInvoice(t, store, "7", "2", "4", "Rush order")
	if len(store.Replay) != 1 {
		t.Fatalf("form rejections recorded %+v", store.Replay)
	}

	// Form authorization and exact Origin: anonymous, revoked and
	// wrong-origin writers share 403 with no store effect. Every
	// passing leg sends Go's canonical "Origin" header, proving the
	// lowercase snapshot guarantee end to end.
	for _, leg := range []struct{ note, session, origin string }{
		{"form anonymous", "", origin},
		{"form revoked", "tok-revoked", origin},
		{"form ghost", "tok-ghost", origin},
		{"form missing origin", "tok-alice", "omit"},
		{"form null origin", "tok-alice", "null"},
		{"form foreign origin", "tok-alice", "https://evil.example"},
		{"form sibling origin", "tok-alice", origin + ".evil.example"},
		{"form slashed origin", "tok-alice", origin + "/"},
	} {
		status, body, _ := postInvoiceForm(t, base, "/tenants/1/invoices/7", leg.session, leg.origin, formValues("4", "", "2"))
		if status != 403 || !strings.Contains(body, "forbidden: request denied") {
			t.Fatalf("%s: %d %s, want 403 forbidden", leg.note, status, body)
		}
	}
	doubled := func() int {
		client := &http.Client{Timeout: 10 * time.Second}
		request, err := http.NewRequest("POST", base+"/tenants/1/invoices/7", strings.NewReader(formValues("4", "", "2").Encode()))
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.Header["Origin"] = []string{origin, origin}
		request.AddCookie(&http.Cookie{Name: "session", Value: "tok-alice"})
		response, err := client.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		return response.StatusCode
	}()
	if doubled != 403 {
		t.Fatalf("form doubled origin: %d, want 403", doubled)
	}
	store = inspectInvoice(t, ctx, bundle, home, driver, db)
	requireInvoice(t, store, "7", "2", "4", "Rush order")

	// JSON save: revision 3 with the acknowledged snapshot and one
	// ledger row carrying digest plus committed result.
	status, payload, headers := postInvoiceJSON(t, base, "/api/tenants/1/invoices/7", "tok-alice", origin, "application/json", saveBody("op-j1", 2))
	if status != 200 {
		t.Fatalf("json save: %d %s", status, payload)
	}
	if content := headers.Get("Content-Type"); !strings.Contains(content, "application/json") {
		t.Fatalf("json save content type %q", content)
	}
	value := gate3Case(t, "json save", payload, "invoice_contract::grid_saved")
	acknowledged, _ := value["acknowledged"].(map[string]any)
	if value["operation_id"] != "op-j1" || acknowledged["revision"] != "3" || acknowledged["total_minor_units"] != 1199.0 {
		t.Fatalf("json save value %+v", value)
	}
	lines, _ := acknowledged["lines"].([]any)
	if len(lines) != 2 || lines[0].(map[string]any)["price"] != "5.00" || lines[1].(map[string]any)["price"] != "1.99" {
		t.Fatalf("json save lines %+v", acknowledged["lines"])
	}
	store = inspectInvoice(t, ctx, bundle, home, driver, db)
	requireInvoice(t, store, "7", "3", "4", "Rush order")
	ledger := requireReplay(t, store, "op-j1", "3")
	if !strings.Contains(ledger.Result, `"revision":3`) {
		t.Fatalf("json save result %q lacks the committed revision", ledger.Result)
	}

	// Identical replay returns the identical bytes with no second
	// effect and no sliding retention stamp; changed content under
	// the same id conflicts.
	status, replayed, _ := postInvoiceJSON(t, base, "/api/tenants/1/invoices/7", "tok-alice", origin, "application/json", saveBody("op-j1", 2))
	if status != 200 || replayed != payload {
		t.Fatalf("json replay: %d %s, want identical 200 bytes", status, replayed)
	}
	store = inspectInvoice(t, ctx, bundle, home, driver, db)
	again := requireReplay(t, store, "op-j1", "3")
	if again.Recorded != ledger.Recorded {
		t.Fatalf("json replay slid retention %s to %s", ledger.Recorded, again.Recorded)
	}
	requireInvoice(t, store, "7", "3", "4", "Rush order")
	changed := `{"operation_id":"op-j1","revision":"2","lines":[{"key":"k1","id":"changed","quantity":"2","price":"5.00"}]}`
	status, payload, _ = postInvoiceJSON(t, base, "/api/tenants/1/invoices/7", "tok-alice", origin, "application/json", changed)
	if status != 409 {
		t.Fatalf("json changed replay: %d %s, want 409", status, payload)
	}
	value = gate3Case(t, "json changed replay", payload, "invoice_contract::grid_conflict")
	if value["message"] != "stale revision" {
		t.Fatalf("json changed replay value %+v", value)
	}

	// JSON validation and staleness: 422/409 with no store effect.
	badPrice := `{"operation_id":"op-j2","revision":"3","lines":[{"key":"k1","id":"a","quantity":"2","price":"five"}]}`
	status, payload, _ = postInvoiceJSON(t, base, "/api/tenants/1/invoices/7", "tok-alice", origin, "application/json", badPrice)
	if status != 422 {
		t.Fatalf("json invalid: %d %s, want 422", status, payload)
	}
	value = gate3Case(t, "json invalid", payload, "invoice_contract::grid_invalid")
	problems, _ := value["errors"].([]any)
	if len(problems) != 1 || problems[0].(map[string]any)["field"] != "price" || problems[0].(map[string]any)["message"] != "bad price" {
		t.Fatalf("json invalid value %+v", value)
	}
	status, payload, _ = postInvoiceJSON(t, base, "/api/tenants/1/invoices/7", "tok-alice", origin, "application/json", saveBody("op-j3", 2))
	if status != 409 {
		t.Fatalf("json stale: %d %s, want 409", status, payload)
	}
	gate3Case(t, "json stale", payload, "invoice_contract::grid_conflict")
	store = inspectInvoice(t, ctx, bundle, home, driver, db)
	requireInvoice(t, store, "7", "3", "4", "Rush order")
	if len(store.Replay) != 2 {
		t.Fatalf("json rejections recorded %+v", store.Replay)
	}

	// JSON denials: foreign, missing, anonymous, revoked and ghost
	// writers share one nondisclosing 403 carrying only the
	// caller-sent operation id.
	for _, leg := range []struct {
		note, session, tenant, invoice string
	}{
		{"json foreign", "tok-alice", "2", "8"},
		{"json missing", "tok-alice", "1", "9"},
		{"json anonymous", "", "1", "7"},
		{"json revoked", "tok-revoked", "1", "7"},
		{"json ghost", "tok-ghost", "1", "7"},
	} {
		status, payload, _ := postInvoiceJSON(t, base, "/api/tenants/"+leg.tenant+"/invoices/"+leg.invoice, leg.session, origin, "application/json", saveBody("op-denied", 3))
		if status != 403 {
			t.Fatalf("%s: %d %s, want 403", leg.note, status, payload)
		}
		value := gate3Case(t, leg.note, payload, "invoice_contract::grid_forbidden")
		if value["operation_id"] != "op-denied" || len(value) != 2 {
			t.Fatalf("%s value %+v discloses or mismatches", leg.note, value)
		}
		if strings.Contains(payload, "Globex") || strings.Contains(payload, "Rush") || strings.Contains(payload, "tenant") {
			t.Fatalf("%s leaks stored content in %q", leg.note, payload)
		}
	}
	for _, leg := range []struct{ note, origin string }{
		{"json missing origin", "omit"},
		{"json null origin", "null"},
		{"json foreign origin", "https://evil.example"},
	} {
		status, payload, _ := postInvoiceJSON(t, base, "/api/tenants/1/invoices/7", "tok-alice", leg.origin, "application/json", saveBody("op-denied", 3))
		if status != 403 {
			t.Fatalf("%s: %d %s, want 403", leg.note, status, payload)
		}
		gate3Case(t, leg.note, payload, "invoice_contract::grid_forbidden")
	}

	// JSON load: the current snapshot with ordered lines and exact
	// total, plus nondisclosing 403s and capture 404s.
	status, body, _ = invoiceGet(t, base, "/api/tenants/1/invoices/7", "tok-alice")
	if status != 200 {
		t.Fatalf("json load: %d %s", status, body)
	}
	value = gate3Case(t, "json load", body, "invoice_contract::grid_loaded")
	current, _ := value["current"].(map[string]any)
	if current["revision"] != "3" || current["total_minor_units"] != 1199.0 {
		t.Fatalf("json load value %+v", value)
	}
	lines, _ = current["lines"].([]any)
	if len(lines) != 2 || lines[0].(map[string]any)["key"] != "k1" || lines[1].(map[string]any)["key"] != "k2" {
		t.Fatalf("json load lines %+v", current["lines"])
	}
	for _, leg := range []struct {
		note, session, tenant, invoice string
	}{
		{"load foreign", "tok-alice", "2", "8"},
		{"load missing", "tok-alice", "1", "9"},
		{"load anonymous", "", "1", "7"},
	} {
		status, body, _ := invoiceGet(t, base, "/api/tenants/"+leg.tenant+"/invoices/"+leg.invoice, leg.session)
		if status != 403 {
			t.Fatalf("%s: %d %s, want 403", leg.note, status, body)
		}
		gate3Case(t, leg.note, body, "invoice_contract::grid_load_forbidden")
	}
	// Malformed integer captures on an otherwise matching shape are
	// bad requests: the route layer rejects non-canonical ints before
	// any handler runs, so they never reach a protected entry as data.
	for _, path := range []string{"/api/tenants/01/invoices/7", "/api/tenants/x/invoices/7"} {
		if status, body, _ := invoiceGet(t, base, path, "tok-alice"); status != 400 {
			t.Fatalf("load %s: %d %s, want 400", path, status, body)
		}
	}
	if status, body, _ := invoiceGet(t, base, "/api/tenants/1/invoices/7/extra", "tok-alice"); status != 404 {
		t.Fatalf("load extra segment: %d %s, want 404", status, body)
	}

	// Retention: an identical old request whose ledger row expired
	// conflicts on its advanced revision instead of causing a second
	// effect, and the expired row is gone afterwards.
	status, payload, _ = postInvoiceJSON(t, base, "/api/tenants/1/invoices/7", "tok-alice", origin, "application/json", saveBody("op-exp1", 3))
	if status != 200 {
		t.Fatalf("expiry setup save: %d %s", status, payload)
	}
	touchReplay(t, ctx, bundle, home, driver, db, "alice", "1", "7", "op-exp1", "1000")
	status, payload, _ = postInvoiceJSON(t, base, "/api/tenants/1/invoices/7", "tok-alice", origin, "application/json", saveBody("op-exp1", 3))
	if status != 409 {
		t.Fatalf("expired replay: %d %s, want 409", status, payload)
	}
	gate3Case(t, "expired replay", payload, "invoice_contract::grid_conflict")
	store = inspectInvoice(t, ctx, bundle, home, driver, db)
	requireInvoice(t, store, "7", "4", "4", "Rush order")
	for _, row := range store.Replay {
		if row.OperationID == "op-exp1" {
			t.Fatalf("expired replay kept %+v", row)
		}
	}

	// Cleanup piggybacks commits: an ancient row vanishes with the
	// next successful save while live rows stay.
	status, payload, _ = postInvoiceJSON(t, base, "/api/tenants/1/invoices/7", "tok-alice", origin, "application/json", saveBody("op-c1", 4))
	if status != 200 {
		t.Fatalf("cleanup setup save: %d %s", status, payload)
	}
	touchReplay(t, ctx, bundle, home, driver, db, "alice", "1", "7", "op-c1", "1000")
	status, payload, _ = postInvoiceJSON(t, base, "/api/tenants/1/invoices/7", "tok-alice", origin, "application/json", saveBody("op-c2", 5))
	if status != 200 {
		t.Fatalf("cleanup commit save: %d %s", status, payload)
	}
	store = inspectInvoice(t, ctx, bundle, home, driver, db)
	requireInvoice(t, store, "7", "6", "4", "Rush order")
	seen := map[string]bool{}
	for _, row := range store.Replay {
		seen[row.OperationID] = true
	}
	if seen["op-c1"] || !seen["op-c2"] {
		t.Fatalf("cleanup ledger %+v", store.Replay)
	}

	// A store fault turns reads and writes into truthful 503s; the
	// identical write retries safely once the table is restored.
	faultInvoice(t, ctx, bundle, home, driver, db, "drop-lines")
	status, payload, _ = postInvoiceJSON(t, base, "/api/tenants/1/invoices/7", "tok-alice", origin, "application/json", saveBody("op-m8", 6))
	if status != 503 {
		t.Fatalf("outage save: %d %s, want 503", status, payload)
	}
	gate3Case(t, "outage save", payload, "invoice_contract::grid_unavailable")
	status, body, _ = invoiceGet(t, base, "/api/tenants/1/invoices/7", "tok-alice")
	if status != 503 {
		t.Fatalf("outage load: %d %s, want 503", status, body)
	}
	gate3Case(t, "outage load", body, "invoice_contract::grid_load_unavailable")
	setup := invoiceDriver(t, ctx, bundle, home, driver, "setup", db, filepath.Join(root, "schema.sql"))
	var setupReport struct {
		Tables int `json:"tables"`
	}
	if err := json.Unmarshal(setup, &setupReport); err != nil || setupReport.Tables != 5 {
		t.Fatalf("invalid invoice restore report %v %s", err, string(setup))
	}
	status, payload, _ = postInvoiceJSON(t, base, "/api/tenants/1/invoices/7", "tok-alice", origin, "application/json", saveBody("op-m8", 6))
	if status != 200 {
		t.Fatalf("recovery save: %d %s", status, payload)
	}
	value = gate3Case(t, "recovery save", payload, "invoice_contract::grid_saved")
	acknowledged, _ = value["acknowledged"].(map[string]any)
	if acknowledged["revision"] != "7" {
		t.Fatalf("recovery save value %+v", value)
	}
	store = inspectInvoice(t, ctx, bundle, home, driver, db)
	requireInvoice(t, store, "7", "7", "4", "Rush order")
	requireReplay(t, store, "op-m8", "7")
	stop()
	t.Logf("invoice live: %d can assertions, form/json save/load/replay/retention/outage legs with row evidence", assertions)
}

// Serial by design: refusal legs bind invoiceLivePort+20 upward, which
// overlaps the gate3 adapter port range, so this test must not run
// alongside the parallel suite.
func TestInvoiceStartupRefusal(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged invoice execution")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()
	bundle, err := harnessBundle(t, ctx, archive)
	if err != nil {
		t.Fatal(err)
	}
	root, home, driver, entry, _ := stageInvoiceBundle(t, ctx, bundle, sourceRoot)
	db := seedInvoiceDB(t, ctx, bundle, home, driver, root)
	base := map[string]string{"INVOICE_DB": db}
	port := invoiceLivePort + 20
	for _, leg := range []struct {
		note   string
		values map[string]string
	}{
		{"bad-origin", map[string]string{"INVOICE_DB": db, "PUBLIC_ORIGIN": "http://shop.example"}},
		{"missing-origin", base},
		{"zero-window", map[string]string{"INVOICE_DB": db, "PUBLIC_ORIGIN": "https://shop.example", "INVOICE_REPLAY_WINDOW_MS": "0"}},
		{"negative-window", map[string]string{"INVOICE_DB": db, "PUBLIC_ORIGIN": "https://shop.example", "INVOICE_REPLAY_WINDOW_MS": "-5"}},
		{"malformed-window", map[string]string{"INVOICE_DB": db, "PUBLIC_ORIGIN": "https://shop.example", "INVOICE_REPLAY_WINDOW_MS": "week"}},
	} {
		port++
		leg.values["INVOICE_DB"] = db
		expectInvoiceStartupFailure(t, ctx, bundle, home, entry, leg.values, port, leg.note)
		t.Logf("startup refusal %s: nonzero exit before serving", leg.note)
	}
}

// canlcBuildArgs runs one staged canlc build with explicit flags, for
// browser-target and browser-manifest builds the fixed-arg helpers do
// not cover.
func canlcBuildArgs(t *testing.T, ctx context.Context, bundle, home string, args ...string) (int, string, string) {
	t.Helper()
	argv := append([]string{"-p", "(version 1)(allow default)(deny network*)", filepath.Join(bundle, "bin/canlc"), "build"}, args...)
	cmd := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", argv...)
	cmd.Dir = home
	cmd.Env = []string{"PATH=/nonexistent", "HOME=" + home}
	var out, diag bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &diag
	if err := cmd.Run(); err != nil {
		var status *exec.ExitError
		if !errors.As(err, &status) {
			t.Fatal(err)
		}
		return status.ExitCode(), out.String(), diag.String()
	}
	return 0, out.String(), diag.String()
}

// TestInvoiceGridPagePaired proves server publication wiring: the invoice
// server built with a browser manifest serves the grid shell with exactly
// the report-selected paired script tag, serves the script bytes, and
// leaves non-HTML answers untouched. The paired browser input is a
// dependency-free empty fixture while UP20 migrates the grid, so this
// proves wiring only, not grid behavior.
func TestInvoiceGridPagePaired(t *testing.T) {
	t.Parallel()
	acquireHeavy(t)
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged invoice execution")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()
	bundle, err := harnessBundle(t, ctx, archive)
	if err != nil {
		t.Fatal(err)
	}
	outside, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	browserRoot, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	write := func(name, text string) {
		t.Helper()
		p := filepath.Join(browserRoot, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("can.project.json", `{"source_root":"src","error_registry":"can.errors.json"}`)
	write("can.errors.json", `{"active":[],"retired":[]}`)
	// Zero-argument main per the UP11 browser-entry contract; reuse the
	// canonical degenerate program so the shape cannot drift again.
	write("src/main.can", gate5EmptyMain)
	status, out, diag := canlcBuildArgs(t, ctx, bundle, outside, "--target", "browser", browserRoot)
	if status != 0 {
		t.Fatalf("browser build: %d %s %s", status, out, diag)
	}
	var browserReport struct {
		BuildID   string `json:"buildID"`
		Directory string `json:"directory"`
	}
	if err := json.Unmarshal([]byte(out), &browserReport); err != nil || browserReport.BuildID == "" {
		t.Fatalf("invalid browser report %v %s", err, out)
	}
	manifest := filepath.Join(browserReport.Directory, "browser", "manifest.json")
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

	root, home := stageApplication(t, ctx, bundle, sourceRoot, "invoice")
	assertStatus, assertOut, assertDiag := canlcOffline(t, ctx, bundle, home, "assert", root)
	if assertStatus != 0 || assertDiag != "" {
		t.Fatalf("invoice assert: %d %s %s", assertStatus, assertOut, assertDiag)
	}
	status, out, diag = canlcBuildArgs(t, ctx, bundle, home, "--browser-manifest", manifest, root)
	if status != 0 {
		t.Fatalf("paired build: %d %s %s", status, out, diag)
	}
	var report struct {
		BuildID   string `json:"buildID"`
		Directory string `json:"directory"`
		Browser   *struct {
			BrowserBuildID string `json:"browserBuildId"`
			Entry          string `json:"entry"`
		} `json:"browser"`
	}
	if err := json.Unmarshal([]byte(out), &report); err != nil || report.Browser == nil {
		t.Fatalf("paired report lacks its browser section: %v %s", err, out)
	}
	if report.Browser.BrowserBuildID != manifestDecoded.BrowserBuildID {
		t.Fatalf("paired browser %s, want %s", report.Browser.BrowserBuildID, manifestDecoded.BrowserBuildID)
	}
	entry := report.Browser.Entry
	if !strings.HasPrefix(entry, "/__can/assets/") || !strings.HasSuffix(entry, ".js") || strings.HasSuffix(entry, ".js.map") {
		t.Fatalf("paired entry %q is not a digest script route", entry)
	}
	driver := filepath.Join(sourceRoot, "tests/integration/testdata/invoice/driver.ts")
	db := seedInvoiceDB(t, ctx, bundle, home, driver, root)
	base, stop := serveInvoice(t, ctx, bundle, home, filepath.Join(report.Directory, "entry.ts"), db, invoicePairedPort, "")
	defer stop()
	want := `<script type="module" src="` + entry + `"></script>`
	status, shell, _ := invoiceGet(t, base, "/invoice-grid?tenant=1&invoice=7", "")
	if status != 200 || strings.Count(shell, want) != 1 || !strings.Contains(shell, `id="invoice-grid"`) {
		t.Fatalf("paired grid shell: %d %.800s, want exactly the report entry once", status, shell)
	}
	status, script, headers := invoiceGet(t, base, entry, "")
	if status != 200 || script == "" || !strings.Contains(headers.Get("Content-Type"), "text/javascript") {
		t.Fatalf("paired entry: %d %q %.200s", status, headers.Get("Content-Type"), script)
	}
	status, page, _ := invoiceGet(t, base, "/invoices/form?tenant_id=1&invoice_id=7", "tok-alice")
	if status != 200 || strings.Count(page, want) != 1 {
		t.Fatalf("paired form page: %d, want the same report entry once", status)
	}
	status, probe, _ := invoiceGet(t, base, "/health", "")
	if status != 200 || probe != "ok" || strings.Contains(probe, "<script") {
		t.Fatalf("paired health: %d %q, want untouched bytes", status, probe)
	}
	t.Logf("invoice paired: browser %s, grid and form pages carry entry %s once, health untouched", manifestDecoded.BrowserBuildID[:12], entry)
}

func TestInvoiceBrowser(t *testing.T) {
	t.Parallel()
	acquireHeavy(t)
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged invoice execution")
	}
	nodeModules := filepath.Join("..", "..", "tests", "integration", "browser", "node_modules", "playwright")
	if _, err := os.Stat(nodeModules); err != nil {
		t.Skip("browser lane needs tests/integration/browser/node_modules/playwright")
	}
	sourceRoot, _ := filepath.Abs("../..")
	script := filepath.Join(sourceRoot, "tests", "integration", "browser", "invoice.mjs")
	harness, err := os.ReadFile(script)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(harness), "tenant_id") {
		t.Skip("invoice.mjs still targets the pre-UP16 page (string ids, hidden inputs); browser lane blocked on harness migration (UP19/UP23)")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()
	bundle, err := harnessBundle(t, ctx, archive)
	if err != nil {
		t.Fatal(err)
	}
	root, home, driver, entry, assertions := stageInvoiceBundle(t, ctx, bundle, sourceRoot)
	db := seedInvoiceDB(t, ctx, bundle, home, driver, root)
	base, stop := serveInvoice(t, ctx, bundle, home, entry, db, invoiceBrowserPort, "")
	defer stop()

	outdir := t.TempDir()
	cmd := exec.CommandContext(ctx, "node", script, base, outdir, db)
	cmd.Dir = filepath.Join(sourceRoot, "tests", "integration", "browser")
	out, err := cmd.CombinedOutput()
	t.Logf("invoice browser harness:\n%s", out)
	if err != nil {
		t.Fatalf("invoice browser: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(outdir, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	var report struct {
		Browser string `json:"browser"`
		Passed  bool   `json:"passed"`
		Checks  []struct {
			Name   string `json:"name"`
			Passed bool   `json:"passed"`
			Detail string `json:"detail"`
		} `json:"checks"`
	}
	if err := json.Unmarshal(raw, &report); err != nil || !report.Passed {
		t.Fatalf("invalid invoice browser report %v %s", err, string(raw))
	}
	for _, check := range report.Checks {
		if !check.Passed {
			t.Fatalf("browser check %s failed: %s", check.Name, check.Detail)
		}
	}
	store := inspectInvoice(t, ctx, bundle, home, driver, db)
	committed := false
	for _, row := range store.Invoice {
		if row.ID == "7" && row.Rev != "1" {
			committed = true
			t.Logf("invoice browser: %d can assertions, %d checks passed, live submit committed rev %s seats %s details %q", assertions, len(report.Checks), row.Rev, row.Seats, row.Details)
		}
	}
	if !committed {
		t.Fatalf("browser run committed nothing: %+v", store.Invoice)
	}
}

// TestInvoiceHTMLFragments proves the server side of the HTML guard
// contract: the served page carries the global noSwap policy plus the
// checked per-status swap admissions, and every admitted form-save
// status answers its exact fragment bytes with an HTML content type.
// DOM swap observation is UP23's leg (TestInvoiceBrowserGuardDOM).
func TestInvoiceHTMLFragments(t *testing.T) {
	t.Parallel()
	acquireHeavy(t)
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged invoice execution")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()
	bundle, err := harnessBundle(t, ctx, archive)
	if err != nil {
		t.Fatal(err)
	}
	root, home, driver, entry, assertions := stageInvoiceBundle(t, ctx, bundle, sourceRoot)
	db := seedInvoiceDB(t, ctx, bundle, home, driver, root)
	base, stop := serveInvoice(t, ctx, bundle, home, entry, db, invoiceFragmentPort, "")
	defer stop()
	origin := "http://127.0.0.1:" + strconv.Itoa(invoiceFragmentPort)
	const formPath = "/tenants/1/invoices/7"
	formValues := func(seats, details, revision string) url.Values {
		values := url.Values{}
		values.Set("seats", seats)
		values.Set("details", details)
		values.Set("revision", revision)
		values.Add("lines_order", "k1")
		values.Set("lines[k1][id]", "a")
		values.Set("lines[k1][quantity]", "2")
		values.Set("lines[k1][price]", "5.00")
		return values
	}
	html := func(note string, headers http.Header) {
		t.Helper()
		if content := headers.Get("Content-Type"); !strings.Contains(content, "text/html") {
			t.Fatalf("%s content type %q, want text/html", note, content)
		}
	}

	// Served policy: the global config keeps the quiet pair and every
	// 4xx/5xx class out of swaps, and the form carries per-element
	// admission for exactly the four declared error statuses.
	status, page, _ := invoiceGet(t, base, "/invoices/form?tenant_id=1&invoice_id=7", "tok-alice")
	if status != 200 {
		t.Fatalf("fragment policy page: %d", status)
	}
	open := strings.Index(page, `<meta name="htmx-config" content="`)
	if open < 0 {
		t.Fatalf("fragment policy page lacks htmx-config in %.800s", page)
	}
	rest := page[open+len(`<meta name="htmx-config" content="`):]
	encoded := rest[:strings.Index(rest, `">`)]
	var config struct {
		Mode   string `json:"mode"`
		NoSwap []any  `json:"noSwap"`
	}
	if err := json.Unmarshal([]byte(strings.ReplaceAll(encoded, "&quot;", `"`)), &config); err != nil {
		t.Fatalf("invalid served htmx-config %v %q", err, encoded)
	}
	want := []any{204.0, 304.0, "4xx", "5xx"}
	if config.Mode != "same-origin" || len(config.NoSwap) != len(want) {
		t.Fatalf("served htmx-config %+v, want same-origin with %d noSwap entries", config, len(want))
	}
	for i, code := range want {
		if config.NoSwap[i] != code {
			t.Fatalf("served htmx-config noSwap[%d]=%v, want %v", i, config.NoSwap[i], code)
		}
	}
	for _, admission := range []string{"403", "409", "422", "503"} {
		if !strings.Contains(page, `hx-status:`+admission+`="{&quot;swap&quot;:&quot;innerHTML&quot;}"`) {
			t.Fatalf("fragment policy page lacks hx-status:%s admission in %.1200s", admission, page)
		}
	}
	if strings.Contains(page, "hx-status:200") {
		t.Fatalf("fragment policy page admits a redundant 2xx exception in %.1200s", page)
	}

	// 200 saved: a polite status fragment naming the revision.
	status, body, headers := postInvoiceForm(t, base, formPath, "tok-alice", origin, formValues("4", "Rush order", "1"))
	if status != 200 || !strings.Contains(body, `<p class="save-saved" role="status">saved revision 2</p>`) {
		t.Fatalf("saved fragment: %d %q", status, body)
	}
	html("saved fragment", headers)

	// 422 invalid: an alert region with the heading, the fielded
	// problems and the recovery hint.
	status, body, headers = postInvoiceForm(t, base, formPath, "tok-alice", origin, formValues("many", "", "2"))
	if status != 422 {
		t.Fatalf("invalid fragment: %d %s, want 422", status, body)
	}
	for _, want := range []string{`<div role="alert">`, "invoice invalid", "seats: bad seats", "Fix the form and save again."} {
		if !strings.Contains(body, want) {
			t.Fatalf("invalid fragment lacks %q in %q", want, body)
		}
	}
	html("invalid fragment", headers)

	// 409 conflict: an alert region with the stale-revision message
	// and the reread hint.
	status, body, headers = postInvoiceForm(t, base, formPath, "tok-alice", origin, formValues("4", "", "1"))
	if status != 409 {
		t.Fatalf("conflict fragment: %d %s, want 409", status, body)
	}
	for _, want := range []string{`<div role="alert">`, `<p class="save-conflict">conflict: stale revision</p>`, "Reload the form to reread, then save again."} {
		if !strings.Contains(body, want) {
			t.Fatalf("conflict fragment lacks %q in %q", want, body)
		}
	}
	html("conflict fragment", headers)

	// 403 forbidden: an alert region echoing nothing of the denied
	// submission, so foreign and nonexistent targets disclose nothing.
	status, body, headers = postInvoiceForm(t, base, formPath, "tok-ghost", origin, formValues("4", "Rush <em>order</em>", "2"))
	if status != 403 {
		t.Fatalf("forbidden fragment: %d %s, want 403", status, body)
	}
	for _, want := range []string{`<div role="alert">`, `<p class="save-denied">forbidden: request denied</p>`, "Sign in with an authorized account and try again."} {
		if !strings.Contains(body, want) {
			t.Fatalf("forbidden fragment lacks %q in %q", want, body)
		}
	}
	if strings.Contains(body, "Rush") {
		t.Fatalf("forbidden fragment echoes the denied submission in %q", body)
	}
	html("forbidden fragment", headers)

	// Structural 422: the heading plus retained raw entries and the
	// issue list, with hostile input escaped.
	hostile := formValues("4", "</li><script>bad()</script>", "2")
	hostile.Del("seats")
	status, body, headers = postInvoiceForm(t, base, formPath, "tok-alice", origin, hostile)
	if status != 422 {
		t.Fatalf("rejected fragment: %d %s, want 422", status, body)
	}
	for _, want := range []string{`<div role="alert">`, "invoice form rejected", `class="form-raw"`, `class="form-issues"`, "&lt;/li&gt;&lt;script&gt;"} {
		if !strings.Contains(body, want) {
			t.Fatalf("rejected fragment lacks %q in %q", want, body)
		}
	}
	if strings.Contains(body, "<script>") {
		t.Fatalf("rejected fragment leaks hostile markup in %q", body)
	}
	html("rejected fragment", headers)

	// 503 unavailable: an alert region with the busy message and the
	// reread-before-retry hint; the identical save commits after
	// restore.
	faultInvoice(t, ctx, bundle, home, driver, db, "drop-lines")
	status, body, headers = postInvoiceForm(t, base, formPath, "tok-alice", origin, formValues("4", "", "2"))
	if status != 503 {
		t.Fatalf("unavailable fragment: %d %s, want 503", status, body)
	}
	for _, want := range []string{`<div role="alert">`, `<p class="save-busy">unavailable: store unavailable</p>`, "The store is unavailable; reread before retrying."} {
		if !strings.Contains(body, want) {
			t.Fatalf("unavailable fragment lacks %q in %q", want, body)
		}
	}
	html("unavailable fragment", headers)
	setup := invoiceDriver(t, ctx, bundle, home, driver, "setup", db, filepath.Join(root, "schema.sql"))
	var setupReport struct {
		Tables int `json:"tables"`
	}
	if err := json.Unmarshal(setup, &setupReport); err != nil || setupReport.Tables != 5 {
		t.Fatalf("invalid invoice restore report %v %s", err, string(setup))
	}
	status, body, _ = postInvoiceForm(t, base, formPath, "tok-alice", origin, formValues("4", "", "2"))
	if status != 200 || !strings.Contains(body, "saved revision 3") {
		t.Fatalf("fragment recovery save: %d %s", status, body)
	}
	t.Logf("invoice fragments: %d can assertions, 200/422/409/403/503 fragment bytes plus served swap policy with row evidence", assertions)
}

// TestInvoiceBrowserGuardDOM gates on UP23's browser DOM guard verdicts.
//
// UP23 owns these legs: each admitted status swaps its fragment into
// exactly its connected target, missing-target submission and
// in-flight target loss report the finite failure, and OOB/partial
// markup plus response-control headers mutate nothing. This test stays
// skipped until UP23 writes its machine-readable verdict file and
// points CAN_UP23_RESULTS at it; the schema is:
//
//	{"engine":"chromium 141","commit":"<server-commit>",
//	 "legs":{"swap-200":{"verdict":"pass","evidence":"..."}, ...}}
//
// Required legs: swap-200, swap-403, swap-409, swap-422, swap-503,
// missing-target-before, missing-target-during, oob-rejected,
// partial-rejected, control-headers-rejected, remount-stable. Every
// leg must read verdict "pass" with non-empty evidence.
func TestInvoiceBrowserGuardDOM(t *testing.T) {
	t.Parallel()
	reportPath := os.Getenv("CAN_UP23_RESULTS")
	if reportPath == "" {
		t.Skip("UP23 browser DOM guard results not supplied (set CAN_UP23_RESULTS to the UP23 verdict file)")
	}
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	var report struct {
		Engine string `json:"engine"`
		Commit string `json:"commit"`
		Legs   map[string]struct {
			Verdict  string `json:"verdict"`
			Evidence string `json:"evidence"`
		} `json:"legs"`
	}
	if err := json.Unmarshal(raw, &report); err != nil || report.Engine == "" || report.Commit == "" {
		t.Fatalf("invalid UP23 verdict file %v %.500s", err, string(raw))
	}
	required := []string{"swap-200", "swap-403", "swap-409", "swap-422", "swap-503", "missing-target-before", "missing-target-during", "oob-rejected", "partial-rejected", "control-headers-rejected", "remount-stable"}
	for _, leg := range required {
		t.Run(leg, func(t *testing.T) {
			result, ok := report.Legs[leg]
			if !ok {
				t.Fatalf("UP23 verdicts lack leg %q (engine %s commit %s)", leg, report.Engine, report.Commit)
			}
			if result.Verdict != "pass" || result.Evidence == "" {
				t.Fatalf("UP23 leg %q: verdict %q evidence %q", leg, result.Verdict, result.Evidence)
			}
			t.Logf("UP23 leg %s passed on %s: %s", leg, report.Engine, result.Evidence)
		})
	}
}
