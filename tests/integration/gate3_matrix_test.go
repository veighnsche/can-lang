// T17 Gate 3 server product matrix. Sibling suites own the form/browser
// lanes (TestInvoiceFormLive, TestInvoiceBrowser) and the webhook lane
// (TestWebhookSliceLive); this file owns the rows those suites do not
// cover, all on real HTTP and database surfaces against staged builds:
//
//   - contract edits: one breaking route/field/case/error edit per leg
//     must fail canlc assert and build with a diagnostic naming the
//     stale statically linked use;
//   - route rebuild: a fresh path edit rebuilds the adapter, served
//     live to prove the new route answers;
//   - JSON matrix: POST-save and GET-load over real HTTP with row
//     evidence for every finite case plus transport failures;
//   - keyed-row reorder: swapped lines_order persists positions and
//     renders in the new order; a malformed order answers 422;
//   - swap-config regression: the served page swaps exactly 200-399
//     and 422, parsed from real served bytes;
//   - uncertain commits: SIGKILL mid-flight and after commit converge
//     to exactly one ledger effect under identical replay.
//
// The Gate 3 verdict combines this file's rows with the sibling rows
// executed in the same run; see the test logs for the per-row report.
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
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/distribution"
)

const (
	gate3MatrixPort = 18561
	gate3RoutePort  = 18562
)

// rewriteGate3File replaces exactly one occurrence of old with new in the
// staged file at rel. It fails the test unless the match is unique so an
// edit leg never silently applies to zero or several sites.
func rewriteGate3File(t *testing.T, root, rel, old, new string) {
	t.Helper()
	path := filepath.Join(root, rel)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(data), old) != 1 {
		t.Fatalf("%s: want exactly one %q site", rel, old)
	}
	if err := os.WriteFile(path, []byte(strings.Replace(string(data), old, new, 1)), 0600); err != nil {
		t.Fatal(err)
	}
}

type gate3Edit struct {
	name  string
	apply func(t *testing.T, root string)
	want  []string
}

func TestGate3ContractEdits(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged gate 3 execution")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "gate3-edits")
	if err != nil {
		t.Fatal(err)
	}
	edits := []gate3Edit{
		{
			name: "action-rename-diagnoses-stale-serve",
			apply: func(t *testing.T, root string) {
				t.Helper()
				rewriteGate3File(t, root, "src/web/web.can", "action save_invoice_form\n", "action save_invoice_form_v2\n")
				rewriteGate3File(t, root, "src/web/web.can", "save_invoice, save_invoice_form,", "save_invoice, save_invoice_form_v2,")
			},
			want: []string{`unknown form action "save_invoice_form"`, "invoice_routes"},
		},
		{
			name: "path-collision-diagnoses-duplicate-route",
			apply: func(t *testing.T, root string) {
				t.Helper()
				rewriteGate3File(t, root, "src/web/web.can", "action save_invoice_form\n    post \"/invoices/save-form\"", "action save_invoice_form\n    post \"/invoices/save\"")
			},
			want: []string{"duplicates the POST /invoices/save route", "save_invoice_form"},
		},
		{
			name: "field-rename-diagnoses-stale-use",
			apply: func(t *testing.T, root string) {
				t.Helper()
				old := "record invoice_form_wire\n    str session_token\n    str operation_id\n    str invoice_id\n    str revision\n    str customer\n"
				rewriteGate3File(t, root, "src/records/records.can", old, strings.Replace(old, "    str customer\n", "    str customer_name\n", 1))
			},
			want: []string{"unknown field customer", "check_form_save"},
		},
		{
			name: "case-drop-diagnoses-omitted-leaf",
			apply: func(t *testing.T, root string) {
				t.Helper()
				rewriteGate3File(t, root, "src/web/web.can", "    handles save_form_validated\n    result records::save_outcome\n    cases\n        records::saved => 200\n        records::rejected => 422\n        records::stale => 409\n        records::denied => 403\n        records::busy => 503\n", "    handles save_form_validated\n    result records::save_outcome\n    cases\n        records::saved => 200\n        records::rejected => 422\n        records::stale => 409\n        records::denied => 403\n")
			},
			want: []string{"action cases omit result leaves records::busy", "save_invoice_form"},
		},
		{
			name: "bogus-registry-entry-fails",
			apply: func(t *testing.T, root string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(root, "can.errors.json"), []byte("{\"active\": [\"web::bogus\"], \"retired\": []}"), 0600); err != nil {
					t.Fatal(err)
				}
			},
			want: []string{"error registry differs from source declarations"},
		},
		{
			name: "dangling-predecessor-chain-fails",
			apply: func(t *testing.T, root string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(root, "can.errors.json"), []byte("{\"active\": [], \"retired\": [], \"predecessors\": {\"web::x\": [\"web::y\"]}}"), 0600); err != nil {
					t.Fatal(err)
				}
			},
			want: []string{`predecessor chain for "web::x" lacks an active declaration`},
		},
	}
	for _, edit := range edits {
		t.Run(edit.name, func(t *testing.T) {
			root, home := stageApplication(t, ctx, bundle, sourceRoot, "invoice")
			edit.apply(t, root)
			for _, command := range []string{"assert", "build"} {
				status, out, diag := canlcOffline(t, ctx, bundle, home, command, root)
				if status == 0 {
					t.Fatalf("%s unexpectedly passed: %.500s", command, out)
				}
				combined := out + "\n" + diag
				for _, want := range edit.want {
					if !strings.Contains(combined, want) {
						t.Fatalf("%s lacks %q in %.1000s", command, want, combined)
					}
				}
			}
			t.Logf("gate3 edit %s: assert and build fail naming the stale use", edit.name)
		})
	}
}

// gate3BundleGrep reports whether any emitted .ts file under dir contains s.
func gate3BundleGrep(t *testing.T, dir, s string) bool {
	t.Helper()
	found := false
	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".ts") || found {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(data), s) {
			found = true
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return found
}

// TestGate3RouteRebuildLive proves a fresh route-path edit rebuilds the
// served adapter: the staged edit builds twice with identical IDs, the
// emitted route table carries the new path, and the served build answers
// the new path with a real 200. It also records a known limitation: the
// edit page's hx-post URL is a convention-linked literal, so the rebuilt
// page still posts to the old path and that post answers 404. A future
// checked page-URL linkage must update this leg.
func TestGate3RouteRebuildLive(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged gate 3 execution")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "gate3-route")
	if err != nil {
		t.Fatal(err)
	}
	root, home := stageApplication(t, ctx, bundle, sourceRoot, "invoice")
	rewriteGate3File(t, root, "src/web/web.can", "action save_invoice_form\n    post \"/invoices/save-form\"", "action save_invoice_form\n    post \"/invoices/save-form-v2\"")
	if status, out, diag := canlcOffline(t, ctx, bundle, home, "assert", root); status != 0 || diag != "" {
		t.Fatalf("edited assert: %d %s %s", status, out, diag)
	}
	firstID, firstDir := applicationBuild(t, ctx, bundle, home, root)
	secondID, _ := applicationBuild(t, ctx, bundle, home, root)
	if firstID != secondID {
		t.Fatalf("edited rebuild drifted: %s vs %s", firstID, secondID)
	}
	if !gate3BundleGrep(t, firstDir, "/invoices/save-form-v2") {
		t.Fatal("rebuilt bundle lacks the new route path")
	}
	driver := filepath.Join(sourceRoot, "tests/integration/testdata/invoice/driver.ts")
	db := seedInvoiceDB(t, ctx, bundle, home, driver, root)
	snapshot := snapshotCredential(t, home, "INVOICE_DB", db)
	base, stop := serveApplication(t, ctx, bundle, home, filepath.Join(firstDir, "entry.ts"), snapshot, gate3RoutePort, "/health")
	defer stop()

	status, page, _ := invoiceGet(t, base, "/invoices/form?invoice_id=inv-1", "tok-alice")
	if status != 200 {
		t.Fatalf("edited form page: %d %s", status, page)
	}
	if !strings.Contains(page, `hx-post="/invoices/save-form"`) {
		t.Fatalf("edited page lost its hx-post in %.500s", page)
	}
	t.Logf("KNOWN LIMITATION: rebuilt page still posts to /invoices/save-form after the route moved to /invoices/save-form-v2")

	values := url.Values{}
	values.Set("session_token", "tok-alice")
	values.Set("operation_id", "op-route-1")
	values.Set("invoice_id", "inv-1")
	values.Set("revision", "1")
	values.Set("customer", "Acme Route")
	values.Add("lines_order", "k1")
	values.Add("lines_order", "k2")
	values.Set("lines[k1][sku]", "sku-1")
	values.Set("lines[k1][qty]", "2")
	values.Set("lines[k2][sku]", "sku-2")
	values.Set("lines[k2][qty]", "1")
	if status, body, _ := invoicePostForm(t, base, "/invoices/save-form", values); status != 404 {
		t.Fatalf("stale page post: %d %s, want 404", status, body)
	}
	if status, body, _ := invoicePostForm(t, base, "/invoices/save-form-v2", values); status != 200 || !strings.Contains(body, "saved inv-1 revision 2") {
		t.Fatalf("rebuilt route post: %d %s, want 200 saved", status, body)
	}
	store := inspectInvoice(t, ctx, bundle, home, driver, db)
	requireInvoiceRevision(t, store, "inv-1", "2", "Acme Route")
	t.Logf("gate3 route rebuild: build %s, new path 200 with row evidence, stale page post 404", firstID[:12])
}

// gate3ServeInvoice runs a staged invoice entry with a port argument and
// one fd-3 credential snapshot, waits for /health, and returns the base
// URL with a SIGKILL crash handle and a clean SIGTERM stopper.
func gate3ServeInvoice(t *testing.T, ctx context.Context, bundle, home, entry, snapshot string, port int) (string, func(), func()) {
	t.Helper()
	file, err := os.Open(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
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
	base := "http://127.0.0.1:" + strconv.Itoa(port)
	deadline := time.Now().Add(30 * time.Second)
	for {
		if status, _, _ := httpGet(base + "/health"); status == 200 {
			break
		}
		if time.Now().After(deadline) {
			cmd.Process.Kill()
			t.Fatalf("gate3 server never ready on %s: stdout=%q stderr=%q", base, out.String(), diag.String())
		}
		time.Sleep(200 * time.Millisecond)
	}
	stopped := false
	crash := func() {
		t.Helper()
		if stopped {
			return
		}
		stopped = true
		cmd.Process.Kill()
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		select {
		case <-done:
		case <-time.After(20 * time.Second):
			t.Fatal("crashed gate3 server would not exit")
		}
	}
	stop := func() {
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
				t.Fatalf("unclean gate3 shutdown: %v stdout=%q stderr=%q", err, out.String(), diag.String())
			}
		case <-time.After(20 * time.Second):
			cmd.Process.Kill()
			t.Fatalf("gate3 shutdown hung: stdout=%q stderr=%q", out.String(), diag.String())
		}
	}
	return base, crash, stop
}

func gate3PostJSONRaw(base, path, session, media, body string) (int, string, http.Header, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	request, err := http.NewRequest("POST", base+path, strings.NewReader(body))
	if err != nil {
		return 0, "", nil, err
	}
	request.Header.Set("Content-Type", media)
	if session != "" {
		request.AddCookie(&http.Cookie{Name: "session", Value: session})
	}
	response, err := client.Do(request)
	if err != nil {
		return 0, "", nil, err
	}
	defer response.Body.Close()
	payload, _ := io.ReadAll(response.Body)
	return response.StatusCode, string(payload), response.Header, nil
}

func gate3PostJSON(t *testing.T, base, path, session, media, body string) (int, string, http.Header) {
	t.Helper()
	status, payload, headers, err := gate3PostJSONRaw(base, path, session, media, body)
	if err != nil {
		t.Fatal(err)
	}
	return status, payload, headers
}

func gate3GetWithBody(t *testing.T, base, path, session, body string) (int, string) {
	t.Helper()
	client := &http.Client{Timeout: 10 * time.Second}
	request, err := http.NewRequest("GET", base+path, strings.NewReader(body))
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
	return response.StatusCode, string(payload)
}

// gate3Case decodes one {"case":...,"value":{...}} outcome body and
// requires the named leaf.
func gate3Case(t *testing.T, note, body, leaf string) map[string]any {
	t.Helper()
	var outcome struct {
		Case  string         `json:"case"`
		Value map[string]any `json:"value"`
	}
	if err := json.Unmarshal([]byte(body), &outcome); err != nil {
		t.Fatalf("%s: invalid outcome %v %q", note, err, body)
	}
	if outcome.Case != leaf {
		t.Fatalf("%s: case %q, want %q in %q", note, outcome.Case, leaf, body)
	}
	if outcome.Value == nil {
		t.Fatalf("%s: case %q holds no value in %q", note, leaf, body)
	}
	return outcome.Value
}

func TestGate3ServerMatrix(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged gate 3 execution")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 14*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "gate3-matrix")
	if err != nil {
		t.Fatal(err)
	}
	root, home, driver, entry, assertions := stageInvoiceBundle(t, ctx, bundle, sourceRoot)
	db := seedInvoiceDB(t, ctx, bundle, home, driver, root)
	snapshot := snapshotCredential(t, home, "INVOICE_DB", db)
	base, crash, stop := gate3ServeInvoice(t, ctx, bundle, home, entry, snapshot, gate3MatrixPort)

	rev := 1
	committed := 0
	saveBody := func(op, id string, baseRev int, customer string) string {
		lines := `[{"key":"k1","sku":"sku-9","qty":4},{"key":"k2","sku":"sku-2","qty":1}]`
		return `{"session_token":"ignored","operation_id":"` + op + `","invoice_id":"` + id + `","revision":` + strconv.Itoa(baseRev) + `,"customer":"` + customer + `","lines":` + lines + `}`
	}
	post := func(note, session, media, body string, want int) (string, http.Header) {
		t.Helper()
		status, payload, headers := gate3PostJSON(t, base, "/invoices/save", session, media, body)
		if status != want {
			t.Fatalf("%s: %d %s, want %d", note, status, payload, want)
		}
		return payload, headers
	}

	// JSON save: a valid write commits and records one replay row.
	payload, headers := post("json save", "tok-alice", "application/json", saveBody("op-m1", "inv-1", rev, "Acme JSON"), 200)
	value := gate3Case(t, "json save", payload, "records::saved")
	if value["invoice_id"] != "inv-1" || value["revision"] != 2.0 {
		t.Fatalf("json save value %+v", value)
	}
	if content := headers.Get("Content-Type"); !strings.Contains(content, "application/json") {
		t.Fatalf("json save content type %q", content)
	}
	rev, committed = 2, 1
	store := inspectInvoice(t, ctx, bundle, home, driver, db)
	requireInvoiceRevision(t, store, "inv-1", "2", "Acme JSON")
	if len(store.Replay) != 1 || store.Replay[0].OperationID != "op-m1" || store.Replay[0].Revision != "2" || len(store.Replay[0].Digest) != 64 {
		t.Fatalf("json save replay %+v", store.Replay)
	}

	// Identical retry replays without a second effect; changed content
	// under the same operation conflicts with the current revision.
	payload, _ = post("json replay", "tok-alice", "application/json", saveBody("op-m1", "inv-1", 1, "Acme JSON"), 200)
	if value := gate3Case(t, "json replay", payload, "records::saved"); value["revision"] != 2.0 {
		t.Fatalf("json replay value %+v", value)
	}
	store = inspectInvoice(t, ctx, bundle, home, driver, db)
	requireInvoiceRevision(t, store, "inv-1", "2", "Acme JSON")
	if len(store.Replay) != 1 {
		t.Fatalf("json replay duplicated %+v", store.Replay)
	}
	payload, _ = post("json changed replay", "tok-alice", "application/json", saveBody("op-m1", "inv-1", 1, "Acme Changed"), 409)
	if value := gate3Case(t, "json changed replay", payload, "records::stale"); value["revision"] != 2.0 {
		t.Fatalf("json changed replay value %+v", value)
	}

	// A stale base answers 409; validation failures answer 422 and
	// write nothing.
	payload, _ = post("json stale", "tok-alice", "application/json", saveBody("op-m2", "inv-1", 1, "Stale"), 409)
	if value := gate3Case(t, "json stale", payload, "records::stale"); value["revision"] != 2.0 {
		t.Fatalf("json stale value %+v", value)
	}
	payload, _ = post("json empty customer", "tok-alice", "application/json", saveBody("op-m3", "inv-1", 2, ""), 422)
	gate3Case(t, "json empty customer", payload, "records::rejected")
	badQty := `{"session_token":"ignored","operation_id":"op-m4","invoice_id":"inv-1","revision":2,"customer":"Acme JSON","lines":[{"key":"k1","sku":"sku-9","qty":0}]}`
	payload, _ = post("json bad qty", "tok-alice", "application/json", badQty, 422)
	gate3Case(t, "json bad qty", payload, "records::rejected")
	store = inspectInvoice(t, ctx, bundle, home, driver, db)
	requireInvoiceRevision(t, store, "inv-1", "2", "Acme JSON")
	if len(store.Replay) != committed {
		t.Fatalf("rejections recorded %+v", store.Replay)
	}

	// Foreign, nonexistent, anonymous, revoked, and ghost writers share
	// one nondisclosing 403 carrying only the caller-sent id.
	for _, leg := range []struct {
		note, session, id string
	}{
		{"json foreign", "tok-alice", "inv-2"},
		{"json missing", "tok-alice", "inv-9"},
		{"json anonymous", "", "inv-1"},
		{"json revoked", "tok-revoked", "inv-1"},
		{"json ghost", "tok-ghost", "inv-1"},
	} {
		payload, _ := post(leg.note, leg.session, "application/json", saveBody("op-denied", leg.id, 1, "X"), 403)
		value := gate3Case(t, leg.note, payload, "records::denied")
		if value["invoice_id"] != leg.id || len(value) != 1 {
			t.Fatalf("%s value %+v discloses or mismatches", leg.note, value)
		}
		if strings.Contains(payload, "Globex") || strings.Contains(payload, "tenant-") || strings.Contains(payload, "Acme") {
			t.Fatalf("%s leaks stored content in %q", leg.note, payload)
		}
	}
	store = inspectInvoice(t, ctx, bundle, home, driver, db)
	requireInvoiceRevision(t, store, "inv-1", "2", "Acme JSON")
	requireInvoiceRevision(t, store, "inv-2", "1", "Globex")

	// Transport failures never reach the protected entry: bad media
	// and malformed bodies answer 400, oversize bodies answer 413.
	payload, _ = post("json bad media", "tok-alice", "text/plain", saveBody("op-m5", "inv-1", 2, "X"), 400)
	if payload != "Bad Request" {
		t.Fatalf("json bad media body %q", payload)
	}
	payload, _ = post("json malformed", "tok-alice", "application/json", "not json", 400)
	if payload != "Bad Request" {
		t.Fatalf("json malformed body %q", payload)
	}
	payload, _ = post("json oversize", "tok-alice", "application/json", saveBody("op-m6", "inv-1", 2, strings.Repeat("p", 9000)), 413)
	if payload != "Payload Too Large" {
		t.Fatalf("json oversize body %q", payload)
	}

	// The explicit utf-8 charset variant is accepted and commits.
	payload, _ = post("json charset", "tok-alice", "application/json; charset=utf-8", saveBody("op-m7", "inv-1", rev, "Acme Charset"), 200)
	if value := gate3Case(t, "json charset", payload, "records::saved"); value["revision"] != 3.0 {
		t.Fatalf("json charset value %+v", value)
	}
	rev, committed = 3, 2

	// JSON load: a member reads the current typed invoice with lines
	// in stored position order.
	status, body, _ := invoiceGet(t, base, "/invoices/load?invoice_id=inv-1", "tok-alice")
	if status != 200 {
		t.Fatalf("json load: %d %s", status, body)
	}
	value = gate3Case(t, "json load", body, "records::found")
	if value["invoice_id"] != "inv-1" || value["revision"] != 3.0 || value["customer"] != "Acme Charset" {
		t.Fatalf("json load value %+v", value)
	}
	lines, ok := value["lines"].([]any)
	if !ok || len(lines) != 2 || lines[0].(map[string]any)["key"] != "k1" || lines[1].(map[string]any)["key"] != "k2" {
		t.Fatalf("json load lines %+v", value["lines"])
	}
	for _, leg := range []struct {
		note, session, id string
	}{
		{"load foreign", "tok-alice", "inv-2"},
		{"load missing", "tok-alice", "inv-9"},
		{"load anonymous", "", "inv-1"},
	} {
		status, body, _ := invoiceGet(t, base, "/invoices/load?invoice_id="+leg.id, leg.session)
		if status != 403 {
			t.Fatalf("%s: %d %s, want 403", leg.note, status, body)
		}
		value := gate3Case(t, leg.note, body, "records::load_denied")
		if value["invoice_id"] != leg.id || len(value) != 1 {
			t.Fatalf("%s value %+v discloses or mismatches", leg.note, value)
		}
	}
	// A GET body never reaches the read: Bun 1.4.2 delivers GET
	// requests with a null body even when the wire carries bytes
	// (verified with a raw socket against the pinned runtime), so the
	// smuggled byte has zero effect on the outcome.
	plainStatus, plainBody, _ := invoiceGet(t, base, "/invoices/load?invoice_id=inv-1", "tok-alice")
	bodyStatus, withBody := gate3GetWithBody(t, base, "/invoices/load?invoice_id=inv-1", "tok-alice", "x")
	if bodyStatus != plainStatus || withBody != plainBody {
		t.Fatalf("load with body: %d %s, want identical %d %s", bodyStatus, withBody, plainStatus, plainBody)
	}
	if status, body, _ := invoiceGet(t, base, "/invoices/load", "tok-alice"); status != 400 {
		t.Fatalf("load without query: %d %s, want 400", status, body)
	}

	// A store outage turns writes into truthful 503s while reads keep
	// serving the current committed content: saves open a per-request
	// pool, which the outage breaks, whereas loads read through the
	// boot pool's open handle. The identical writes retry safely once
	// access returns.
	if err := os.Chmod(db, 0); err != nil {
		t.Fatal(err)
	}
	func() {
		defer func() {
			if err := os.Chmod(db, 0600); err != nil {
				t.Fatal(err)
			}
		}()
		payload, _ := post("json outage", "tok-alice", "application/json", saveBody("op-m8", "inv-1", rev, "Outage"), 503)
		gate3Case(t, "json outage", payload, "records::busy")
		status, body, _ := invoiceGet(t, base, "/invoices/load?invoice_id=inv-1", "tok-alice")
		if status != 200 {
			t.Fatalf("load outage: %d %s, want truthful 200", status, body)
		}
		if value := gate3Case(t, "load outage", body, "records::found"); value["revision"] != float64(rev) {
			t.Fatalf("load outage value %+v, want current revision %d", value, rev)
		}
	}()
	payload, _ = post("json recovery", "tok-alice", "application/json", saveBody("op-m8", "inv-1", rev, "Recovered"), 200)
	if value := gate3Case(t, "json recovery", payload, "records::saved"); value["revision"] != 4.0 {
		t.Fatalf("json recovery value %+v", value)
	}
	rev, committed = 4, 3

	// Keyed-row reorder: swapping lines_order persists swapped
	// positions and renders in the new order.
	reorder := url.Values{}
	reorder.Set("session_token", "tok-alice")
	reorder.Set("operation_id", "op-m9")
	reorder.Set("invoice_id", "inv-1")
	reorder.Set("revision", strconv.Itoa(rev))
	reorder.Set("customer", "Reordered")
	reorder.Add("lines_order", "k2")
	reorder.Add("lines_order", "k1")
	reorder.Set("lines[k1][sku]", "sku-9")
	reorder.Set("lines[k1][qty]", "4")
	reorder.Set("lines[k2][sku]", "sku-2")
	reorder.Set("lines[k2][qty]", "7")
	status, payload, _ = invoicePostForm(t, base, "/invoices/save-form", reorder)
	if status != 200 || !strings.Contains(payload, "saved inv-1 revision 5") {
		t.Fatalf("reorder: %d %s, want 200 saved rev 5", status, payload)
	}
	rev, committed = 5, 4
	store = inspectInvoice(t, ctx, bundle, home, driver, db)
	if len(store.Lines) != 2 || store.Lines[0].LineKey != "k2" || store.Lines[0].Position != "0" || store.Lines[1].LineKey != "k1" || store.Lines[1].Position != "1" || store.Lines[0].Qty != "7" {
		t.Fatalf("reordered lines %+v", store.Lines)
	}
	status, page, _ := invoiceGet(t, base, "/invoices/form?invoice_id=inv-1", "tok-alice")
	if status != 200 {
		t.Fatalf("reordered page: %d", status)
	}
	if strings.Index(page, `value="k2"`) < 0 || strings.Index(page, `value="k2"`) > strings.Index(page, `value="k1"`) {
		t.Fatalf("reordered page keeps old row order in %.800s", page)
	}

	// A malformed order naming an unknown key answers 422 and writes
	// nothing.
	badOrder := url.Values{}
	badOrder.Set("session_token", "tok-alice")
	badOrder.Set("operation_id", "op-m10")
	badOrder.Set("invoice_id", "inv-1")
	badOrder.Set("revision", strconv.Itoa(rev))
	badOrder.Set("customer", "Reordered")
	badOrder.Add("lines_order", "k9")
	badOrder.Set("lines[k1][sku]", "sku-9")
	badOrder.Set("lines[k1][qty]", "4")
	status, payload, _ = invoicePostForm(t, base, "/invoices/save-form", badOrder)
	if status != 422 || !strings.Contains(payload, `role="alert"`) {
		t.Fatalf("malformed order: %d %s, want 422 alert", status, payload)
	}
	store = inspectInvoice(t, ctx, bundle, home, driver, db)
	requireInvoiceRevision(t, store, "inv-1", "5", "Reordered")
	if len(store.Replay) != committed {
		t.Fatalf("malformed order recorded %+v", store.Replay)
	}

	// Swap-config regression on served bytes: exactly 200-399 and 422
	// swap into the status region; every other 4xx/5xx plus the quiet
	// 204/304 stay unswapped.
	status, page, _ = invoiceGet(t, base, "/invoices/form?invoice_id=inv-1", "tok-alice")
	if status != 200 {
		t.Fatalf("swap page: %d", status)
	}
	open := strings.Index(page, `<meta name="htmx-config" content="`)
	if open < 0 {
		t.Fatalf("swap page lacks htmx-config in %.800s", page)
	}
	rest := page[open+len(`<meta name="htmx-config" content="`):]
	encoded := rest[:strings.Index(rest, `">`)]
	var config struct {
		Mode   string `json:"mode"`
		NoSwap []int  `json:"noSwap"`
	}
	if err := json.Unmarshal([]byte(strings.ReplaceAll(encoded, "&quot;", `"`)), &config); err != nil {
		t.Fatalf("invalid served htmx-config %v %q", err, encoded)
	}
	want := []int{204, 304}
	for code := 400; code < 600; code++ {
		if code != 422 {
			want = append(want, code)
		}
	}
	if config.Mode != "same-origin" || len(config.NoSwap) != len(want) {
		t.Fatalf("served htmx-config %+v, want same-origin with %d noSwap codes", config, len(want))
	}
	for i, code := range want {
		if config.NoSwap[i] != code {
			t.Fatalf("served htmx-config noSwap[%d]=%d, want %d", i, config.NoSwap[i], code)
		}
	}

	// Uncertain commit: SIGKILL mid-flight, then the identical replay
	// converges to one ledger effect and one revision step however the
	// race landed.
	inflight := make(chan string, 1)
	go func() {
		status, payload, _, err := gate3PostJSONRaw(base, "/invoices/save", "tok-alice", "application/json", saveBody("op-m11", "inv-1", rev, "Uncertain"))
		if err != nil {
			inflight <- "transport: " + err.Error()
			return
		}
		inflight <- "response: " + strconv.Itoa(status) + " " + payload
	}()
	time.Sleep(100 * time.Millisecond)
	crash()
	base, crash, stop = gate3ServeInvoice(t, ctx, bundle, home, entry, snapshot, gate3MatrixPort)
	select {
	case outcome := <-inflight:
		t.Logf("in-flight outcome %s (either landing converges below)", outcome)
	case <-time.After(15 * time.Second):
		t.Fatal("in-flight save never returned")
	}
	for i := 0; i < 3; i++ {
		status, payload, _ := gate3PostJSON(t, base, "/invoices/save", "tok-alice", "application/json", saveBody("op-m11", "inv-1", rev, "Uncertain"))
		if status != 200 {
			t.Fatalf("converge replay %d: %d %s", i, status, payload)
		}
		if value := gate3Case(t, "converge replay", payload, "records::saved"); value["revision"] != float64(rev+1) {
			t.Fatalf("converge replay %d value %+v", i, value)
		}
	}
	rev, committed = rev+1, committed+1
	store = inspectInvoice(t, ctx, bundle, home, driver, db)
	requireInvoiceRevision(t, store, "inv-1", strconv.Itoa(rev), "Uncertain")
	effects := 0
	for _, row := range store.Replay {
		if row.OperationID == "op-m11" {
			effects++
		}
	}
	if effects != 1 || len(store.Replay) != committed {
		t.Fatalf("uncertain commit replayed %+v", store.Replay)
	}

	// Lost acknowledgement after commit: the write answered 200, the
	// server died, and the identical retry replays the stored revision
	// while a changed retry conflicts.
	payload, _ = post("pre-crash save", "tok-alice", "application/json", saveBody("op-m12", "inv-1", rev, "Ack Lost"), 200)
	if value := gate3Case(t, "pre-crash save", payload, "records::saved"); value["revision"] != float64(rev+1) {
		t.Fatalf("pre-crash save value %+v", value)
	}
	rev, committed = rev+1, committed+1
	crash()
	base, _, stop = gate3ServeInvoice(t, ctx, bundle, home, entry, snapshot, gate3MatrixPort)
	payload, _ = post("lost-ack replay", "tok-alice", "application/json", saveBody("op-m12", "inv-1", rev-1, "Ack Lost"), 200)
	if value := gate3Case(t, "lost-ack replay", payload, "records::saved"); value["revision"] != float64(rev) {
		t.Fatalf("lost-ack replay value %+v", value)
	}
	payload, _ = post("lost-ack changed", "tok-alice", "application/json", saveBody("op-m12", "inv-1", rev-1, "Ack Changed"), 409)
	gate3Case(t, "lost-ack changed", payload, "records::stale")
	store = inspectInvoice(t, ctx, bundle, home, driver, db)
	requireInvoiceRevision(t, store, "inv-1", strconv.Itoa(rev), "Ack Lost")
	if len(store.Replay) != committed {
		t.Fatalf("lost-ack ledger %+v, want %d rows", store.Replay, committed)
	}

	stop()
	t.Logf("gate3 matrix: %d can assertions, json save/load/reorder/swap/uncertain legs live with row evidence", assertions)
}
