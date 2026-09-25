// UP16 Gate 3 server product matrix. Sibling suites own the form/JSON
// live lanes (TestInvoiceFormLive, TestInvoiceBrowser); this file owns
// the rows those suites do not cover, all on real HTTP and database
// surfaces against staged builds:
//
//   - contract edits: one breaking shared-lock/server/mount edit per
//     leg must fail canlc assert and build with a diagnostic naming
//     the stale use or digest;
//   - route rebuild: a shared-route edit plus its lock update
//     rebuilds the adapter, and the served edit page follows through
//     its checked action URL (the old convention-linked limitation
//     is gone);
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
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
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
			name: "shared-route-edit-breaks-lock",
			apply: func(t *testing.T, root string) {
				t.Helper()
				rewriteGate3File(t, root, "vendor/billing/src/invoice_contract/invoice_contract.can", `    post "/tenants/:tenant_id/invoices/:invoice_id"`, `    post "/t/:tenant_id/i/:invoice_id"`)
			},
			want: []string{`stale dependency digest for "can.project.lineage/billing"`},
		},
		{
			name: "mount-handler-mismatch-diagnoses",
			apply: func(t *testing.T, root string) {
				t.Helper()
				rewriteGate3File(t, root, "src/web/web.can", "call action::mount(contract::save_invoice_grid, callable save_grid)", "call action::mount(contract::save_invoice_grid, callable load_grid)")
			},
			want: []string{`action::mount handler "load_grid"`, `contract::save_invoice_grid`},
		},
		{
			name: "field-rename-diagnoses-stale-use",
			apply: func(t *testing.T, root string) {
				t.Helper()
				old := "record line_draft\n    str key\n    str id\n"
				rewriteGate3File(t, root, "src/records/records.can", old, strings.Replace(old, "    str id\n", "    str line_id\n", 1))
			},
			want: []string{"unknown field id", "check_line"},
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

// gate3WriteLock recomputes the staged billing dependency lock from the
// staged vendored bytes, mirroring the repository lineage/lock digest
// format: the manifest digest covers the raw manifest bytes and the
// source digest covers the path-tagged source tree.
func gate3WriteLock(t *testing.T, root string) {
	t.Helper()
	manifest, err := os.ReadFile(filepath.Join(root, "vendor/billing/can.project.json"))
	if err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile(filepath.Join(root, "vendor/billing/src/invoice_contract/invoice_contract.can"))
	if err != nil {
		t.Fatal(err)
	}
	registryBytes, err := os.ReadFile(filepath.Join(root, "vendor/billing/can.errors.json"))
	if err != nil {
		t.Fatal(err)
	}
	digest := func(data []byte) string {
		sum := sha256.Sum256(data)
		return hex.EncodeToString(sum[:])
	}
	sourceSum := sha256.New()
	sourceSum.Write([]byte("can-source-tree-v1\x00"))
	var count [binary.MaxVarintLen64]byte
	sourceSum.Write(count[:binary.PutUvarint(count[:], 1)])
	path := "invoice_contract/invoice_contract.can"
	sourceSum.Write([]byte(path + "\x00"))
	sourceSum.Write(count[:binary.PutUvarint(count[:], uint64(len(source)))])
	sourceSum.Write(source)
	var registry any
	if err := json.Unmarshal(registryBytes, &registry); err != nil {
		t.Fatal(err)
	}
	id := "can.project.lineage/billing"
	lock := map[string]any{
		"edges": map[string]any{
			"billing": map[string]any{"target": id, "path": "vendor/billing"},
		},
		"projects": map[string]any{
			id: map[string]any{
				"lineage":         "billing",
				"manifest_sha256": digest(manifest),
				"source_sha256":   hex.EncodeToString(sourceSum.Sum(nil)),
				"fixtures_sha256": digest([]byte("can-fixture-tree-v1\x00")),
				"error_registry":  registry,
				"edges":           map[string]any{},
			},
		},
	}
	lockBytes, err := json.Marshal(lock)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "can.lock.json"), lockBytes, 0600); err != nil {
		t.Fatal(err)
	}
}

// TestGate3RouteRebuildLive proves a shared-route edit plus its lock
// update rebuilds the served adapter: the staged edit builds twice with
// identical IDs, the emitted route table carries the new path, the
// served edit page follows through its checked action URL, and the new
// path answers 200 with row evidence while the old path answers 404.
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
	rewriteGate3File(t, root, "vendor/billing/src/invoice_contract/invoice_contract.can", `    post "/tenants/:tenant_id/invoices/:invoice_id"`, `    post "/t/:tenant_id/i/:invoice_id"`)
	gate3WriteLock(t, root)
	if status, out, diag := canlcOffline(t, ctx, bundle, home, "assert", root); status != 0 || diag != "" {
		t.Fatalf("edited assert: %d %s %s", status, out, diag)
	}
	firstID, firstDir := applicationBuild(t, ctx, bundle, home, root)
	secondID, _ := applicationBuild(t, ctx, bundle, home, root)
	if firstID != secondID {
		t.Fatalf("edited rebuild drifted: %s vs %s", firstID, secondID)
	}
	if !gate3BundleGrep(t, firstDir, "/t/:tenant_id/i/:invoice_id") {
		t.Fatal("rebuilt bundle lacks the new route path")
	}
	driver := filepath.Join(sourceRoot, "tests/integration/testdata/invoice/driver.ts")
	db := seedInvoiceDB(t, ctx, bundle, home, driver, root)
	origin := "http://127.0.0.1:" + strconv.Itoa(gate3RoutePort)
	snapshot := snapshotMap(t, home, "snapshot-gate3-route", map[string]string{"INVOICE_DB": db, "PUBLIC_ORIGIN": origin})
	base, stop := serveApplication(t, ctx, bundle, home, filepath.Join(firstDir, "entry.ts"), snapshot, gate3RoutePort, "/health")
	defer stop()

	status, page, _ := invoiceGet(t, base, "/invoices/form?tenant_id=1&invoice_id=7", "tok-alice")
	if status != 200 {
		t.Fatalf("edited form page: %d %s", status, page)
	}
	if !strings.Contains(page, `hx-post="/t/1/i/7"`) {
		t.Fatalf("edited page lost its checked hx-post in %.800s", page)
	}

	values := url.Values{}
	values.Set("seats", "2")
	values.Set("details", "Route")
	values.Set("revision", "1")
	values.Add("lines_order", "k1")
	values.Set("lines[k1][id]", "a")
	values.Set("lines[k1][quantity]", "2")
	values.Set("lines[k1][price]", "5.00")
	if status, body, _ := postInvoiceForm(t, base, "/tenants/1/invoices/7", "tok-alice", origin, values); status != 404 {
		t.Fatalf("stale route post: %d %s, want 404", status, body)
	}
	if status, body, _ := postInvoiceForm(t, base, "/t/1/i/7", "tok-alice", origin, values); status != 200 || !strings.Contains(body, "saved revision 2") {
		t.Fatalf("rebuilt route post: %d %s, want 200 saved", status, body)
	}
	store := inspectInvoice(t, ctx, bundle, home, driver, db)
	requireInvoice(t, store, "7", "2", "2", "Route")
	t.Logf("gate3 route rebuild: build %s, new path 200 with row evidence, page follows checked URL", firstID[:12])
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
	request.Header.Set("Origin", base)
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
	origin := "http://127.0.0.1:" + strconv.Itoa(gate3MatrixPort)
	snapshot := snapshotMap(t, home, "snapshot-gate3-matrix", map[string]string{"INVOICE_DB": db, "PUBLIC_ORIGIN": origin})
	base, crash, stop := gate3ServeInvoice(t, ctx, bundle, home, entry, snapshot, gate3MatrixPort)

	rev := 1
	committed := 0
	const savePath = "/api/tenants/1/invoices/7"
	saveBody := func(op string, baseRev int) string {
		lines := `[{"key":"k1","id":"sku-9","quantity":"4","price":"9.99"},{"key":"k2","id":"sku-2","quantity":"1","price":"2.00"}]`
		return `{"operation_id":"` + op + `","revision":"` + strconv.Itoa(baseRev) + `","lines":` + lines + `}`
	}
	post := func(note, session, media, body string, want int) (string, http.Header) {
		t.Helper()
		status, payload, headers := gate3PostJSON(t, base, savePath, session, media, body)
		if status != want {
			t.Fatalf("%s: %d %s, want %d", note, status, payload, want)
		}
		return payload, headers
	}

	// JSON save: a valid write commits and records one replay row.
	payload, headers := post("json save", "tok-alice", "application/json", saveBody("op-m1", rev), 200)
	value := gate3Case(t, "json save", payload, "invoice_contract::grid_saved")
	acknowledged, _ := value["acknowledged"].(map[string]any)
	if value["operation_id"] != "op-m1" || acknowledged["revision"] != "2" {
		t.Fatalf("json save value %+v", value)
	}
	if content := headers.Get("Content-Type"); !strings.Contains(content, "application/json") {
		t.Fatalf("json save content type %q", content)
	}
	rev, committed = 2, 1
	store := inspectInvoice(t, ctx, bundle, home, driver, db)
	row := requireInvoice(t, store, "7", "2", "2", "Acme <em>&\" ''coop''\"")
	if len(row.Lines) != 2 || row.Lines[0].Price != "999" {
		t.Fatalf("json save lines %+v", row.Lines)
	}
	ledger := requireReplay(t, store, "op-m1", "2")
	if ledger.Actor != "alice" || ledger.Tenant != "1" || ledger.InvoiceID != "7" {
		t.Fatalf("json save replay scope %+v", ledger)
	}

	// Identical retry replays without a second effect; changed content
	// under the same operation conflicts.
	payload, _ = post("json replay", "tok-alice", "application/json", saveBody("op-m1", 1), 200)
	value = gate3Case(t, "json replay", payload, "invoice_contract::grid_saved")
	acknowledged, _ = value["acknowledged"].(map[string]any)
	if acknowledged["revision"] != "2" {
		t.Fatalf("json replay value %+v", value)
	}
	store = inspectInvoice(t, ctx, bundle, home, driver, db)
	requireInvoice(t, store, "7", "2", "2", "Acme <em>&\" ''coop''\"")
	if len(store.Replay) != 1 {
		t.Fatalf("json replay duplicated %+v", store.Replay)
	}
	changed := `{"operation_id":"op-m1","revision":"1","lines":[{"key":"k1","id":"changed","quantity":"4","price":"9.99"}]}`
	payload, _ = post("json changed replay", "tok-alice", "application/json", changed, 409)
	if value := gate3Case(t, "json changed replay", payload, "invoice_contract::grid_conflict"); value["message"] != "stale revision" {
		t.Fatalf("json changed replay value %+v", value)
	}

	// A stale base answers 409; validation failures answer 422 and
	// write nothing.
	payload, _ = post("json stale", "tok-alice", "application/json", saveBody("op-m2", 1), 409)
	gate3Case(t, "json stale", payload, "invoice_contract::grid_conflict")
	badQty := `{"operation_id":"op-m4","revision":"2","lines":[{"key":"k1","id":"sku-9","quantity":"0","price":"9.99"}]}`
	payload, _ = post("json bad qty", "tok-alice", "application/json", badQty, 422)
	value = gate3Case(t, "json bad qty", payload, "invoice_contract::grid_invalid")
	problems, _ := value["errors"].([]any)
	if len(problems) != 1 || problems[0].(map[string]any)["field"] != "quantity" {
		t.Fatalf("json bad qty value %+v", value)
	}
	store = inspectInvoice(t, ctx, bundle, home, driver, db)
	requireInvoice(t, store, "7", "2", "2", "Acme <em>&\" ''coop''\"")
	if len(store.Replay) != committed {
		t.Fatalf("rejections recorded %+v", store.Replay)
	}

	// Foreign, nonexistent, anonymous, revoked, and ghost writers share
	// one nondisclosing 403 carrying only the caller-sent operation id.
	for _, leg := range []struct {
		note, session, tenant, invoice string
	}{
		{"json foreign", "tok-alice", "2", "8"},
		{"json missing", "tok-alice", "1", "9"},
		{"json anonymous", "", "1", "7"},
		{"json revoked", "tok-revoked", "1", "7"},
		{"json ghost", "tok-ghost", "1", "7"},
	} {
		status, payload, _ := gate3PostJSON(t, base, "/api/tenants/"+leg.tenant+"/invoices/"+leg.invoice, leg.session, "application/json", saveBody("op-denied", 1))
		if status != 403 {
			t.Fatalf("%s: %d %s, want 403", leg.note, status, payload)
		}
		value := gate3Case(t, leg.note, payload, "invoice_contract::grid_forbidden")
		if value["operation_id"] != "op-denied" || len(value) != 2 {
			t.Fatalf("%s value %+v discloses or mismatches", leg.note, value)
		}
		if strings.Contains(payload, "Globex") || strings.Contains(payload, "tenant") || strings.Contains(payload, "Acme") {
			t.Fatalf("%s leaks stored content in %q", leg.note, payload)
		}
	}
	store = inspectInvoice(t, ctx, bundle, home, driver, db)
	requireInvoice(t, store, "7", "2", "2", "Acme <em>&\" ''coop''\"")
	requireInvoice(t, store, "8", "1", "1", "Globex")

	// Transport failures never reach the protected entry: bad media
	// answers 415, malformed bodies answer 400, oversize bodies 413.
	payload, _ = post("json bad media", "tok-alice", "text/plain", saveBody("op-m5", 2), 415)
	if payload != "Unsupported Media Type" {
		t.Fatalf("json bad media body %q", payload)
	}
	payload, _ = post("json malformed", "tok-alice", "application/json", "not json", 400)
	if payload != "Bad Request" {
		t.Fatalf("json malformed body %q", payload)
	}
	payload, _ = post("json oversize", "tok-alice", "application/json", saveBody("op-m6", 2)[:100]+strings.Repeat("p", 9000)+`}]}`, 413)
	if payload != "Payload Too Large" {
		t.Fatalf("json oversize body %q", payload)
	}

	// The explicit utf-8 charset variant is accepted and commits.
	payload, _ = post("json charset", "tok-alice", "application/json; charset=utf-8", saveBody("op-m7", rev), 200)
	value = gate3Case(t, "json charset", payload, "invoice_contract::grid_saved")
	acknowledged, _ = value["acknowledged"].(map[string]any)
	if acknowledged["revision"] != "3" {
		t.Fatalf("json charset value %+v", value)
	}
	rev, committed = 3, 2

	// JSON load: a member reads the current snapshot with lines in
	// stored position order and the exact total.
	status, body, _ := invoiceGet(t, base, "/api/tenants/1/invoices/7", "tok-alice")
	if status != 200 {
		t.Fatalf("json load: %d %s", status, body)
	}
	value = gate3Case(t, "json load", body, "invoice_contract::grid_loaded")
	current, _ := value["current"].(map[string]any)
	if current["revision"] != "3" || current["total_minor_units"] != 4196.0 {
		t.Fatalf("json load value %+v", value)
	}
	lines, ok := current["lines"].([]any)
	if !ok || len(lines) != 2 || lines[0].(map[string]any)["key"] != "k1" || lines[1].(map[string]any)["key"] != "k2" {
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
	// A GET body never reaches the read: Bun 1.4.2 delivers GET
	// requests with a null body even when the wire carries bytes
	// (verified with a raw socket against the pinned runtime), so the
	// smuggled byte has zero effect on the outcome.
	plainStatus, plainBody, _ := invoiceGet(t, base, "/api/tenants/1/invoices/7", "tok-alice")
	bodyStatus, withBody := gate3GetWithBody(t, base, "/api/tenants/1/invoices/7", "tok-alice", "x")
	if bodyStatus != plainStatus || withBody != plainBody {
		t.Fatalf("load with body: %d %s, want identical %d %s", bodyStatus, withBody, plainStatus, plainBody)
	}
	if status, body, _ := invoiceGet(t, base, "/api/tenants/1/invoices", "tok-alice"); status != 404 {
		t.Fatalf("load without capture: %d %s, want 404", status, body)
	}

	// A store fault turns reads and writes into truthful 503s; the
	// identical write retries safely once the table is restored.
	faultInvoice(t, ctx, bundle, home, driver, db, "drop-lines")
	payload, _ = post("json outage", "tok-alice", "application/json", saveBody("op-m8", rev), 503)
	gate3Case(t, "json outage", payload, "invoice_contract::grid_unavailable")
	status, body, _ = invoiceGet(t, base, "/api/tenants/1/invoices/7", "tok-alice")
	if status != 503 {
		t.Fatalf("load outage: %d %s, want truthful 503", status, body)
	}
	gate3Case(t, "load outage", body, "invoice_contract::grid_load_unavailable")
	setup := invoiceDriver(t, ctx, bundle, home, driver, "setup", db, filepath.Join(root, "schema.sql"))
	var setupReport struct {
		Tables int `json:"tables"`
	}
	if err := json.Unmarshal(setup, &setupReport); err != nil || setupReport.Tables != 5 {
		t.Fatalf("invalid invoice restore report %v %s", err, string(setup))
	}
	payload, _ = post("json recovery", "tok-alice", "application/json", saveBody("op-m8", rev), 200)
	value = gate3Case(t, "json recovery", payload, "invoice_contract::grid_saved")
	acknowledged, _ = value["acknowledged"].(map[string]any)
	if acknowledged["revision"] != "4" {
		t.Fatalf("json recovery value %+v", value)
	}
	rev, committed = 4, 3

	// Keyed-row reorder: swapping lines_order persists swapped
	// positions and renders in the new order.
	reorder := url.Values{}
	reorder.Set("seats", "6")
	reorder.Set("details", "Reordered")
	reorder.Set("revision", strconv.Itoa(rev))
	reorder.Add("lines_order", "k2")
	reorder.Add("lines_order", "k1")
	reorder.Set("lines[k1][id]", "sku-9")
	reorder.Set("lines[k1][quantity]", "4")
	reorder.Set("lines[k1][price]", "9.99")
	reorder.Set("lines[k2][id]", "sku-2")
	reorder.Set("lines[k2][quantity]", "7")
	reorder.Set("lines[k2][price]", "2.00")
	status, payload, _ = postInvoiceForm(t, base, "/tenants/1/invoices/7", "tok-alice", origin, reorder)
	if status != 200 || !strings.Contains(payload, "saved revision 5") {
		t.Fatalf("reorder: %d %s, want 200 saved rev 5", status, payload)
	}
	rev, committed = 5, 4
	store = inspectInvoice(t, ctx, bundle, home, driver, db)
	row = requireInvoice(t, store, "7", "5", "6", "Reordered")
	if len(row.Lines) != 2 || row.Lines[0].Key != "k2" || row.Lines[0].Position != "0" || row.Lines[1].Key != "k1" || row.Lines[1].Position != "1" || row.Lines[0].Quantity != "7" {
		t.Fatalf("reordered lines %+v", row.Lines)
	}
	status, page, _ := invoiceGet(t, base, "/invoices/form?tenant_id=1&invoice_id=7", "tok-alice")
	if status != 200 {
		t.Fatalf("reordered page: %d", status)
	}
	if strings.Index(page, `value="k2"`) < 0 || strings.Index(page, `value="k2"`) > strings.Index(page, `value="k1"`) {
		t.Fatalf("reordered page keeps old row order in %.800s", page)
	}

	// A malformed order naming an unknown key answers 422 and writes
	// nothing.
	badOrder := url.Values{}
	badOrder.Set("seats", "6")
	badOrder.Set("details", "Reordered")
	badOrder.Set("revision", strconv.Itoa(rev))
	badOrder.Add("lines_order", "k9")
	badOrder.Set("lines[k1][id]", "sku-9")
	badOrder.Set("lines[k1][quantity]", "4")
	badOrder.Set("lines[k1][price]", "9.99")
	status, payload, _ = postInvoiceForm(t, base, "/tenants/1/invoices/7", "tok-alice", origin, badOrder)
	if status != 422 || !strings.Contains(payload, `role="alert"`) {
		t.Fatalf("malformed order: %d %s, want 422 alert", status, payload)
	}
	store = inspectInvoice(t, ctx, bundle, home, driver, db)
	requireInvoice(t, store, "7", "5", "6", "Reordered")
	if len(store.Replay) != committed {
		t.Fatalf("malformed order recorded %+v", store.Replay)
	}

	// Swap-config regression on served bytes: exactly 200-399 and 422
	// swap into the status region; every other 4xx/5xx plus the quiet
	// 204/304 stay unswapped.
	status, page, _ = invoiceGet(t, base, "/invoices/form?tenant_id=1&invoice_id=7", "tok-alice")
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
		status, payload, _, err := gate3PostJSONRaw(base, savePath, "tok-alice", "application/json", saveBody("op-m11", rev))
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
		status, payload, _ := gate3PostJSON(t, base, savePath, "tok-alice", "application/json", saveBody("op-m11", rev))
		if status != 200 {
			t.Fatalf("converge replay %d: %d %s", i, status, payload)
		}
		value := gate3Case(t, "converge replay", payload, "invoice_contract::grid_saved")
		acknowledged, _ := value["acknowledged"].(map[string]any)
		if acknowledged["revision"] != strconv.Itoa(rev+1) {
			t.Fatalf("converge replay %d value %+v", i, value)
		}
	}
	rev, committed = rev+1, committed+1
	store = inspectInvoice(t, ctx, bundle, home, driver, db)
	requireInvoice(t, store, "7", strconv.Itoa(rev), "6", "Reordered")
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
	payload, _ = post("pre-crash save", "tok-alice", "application/json", saveBody("op-m12", rev), 200)
	value = gate3Case(t, "pre-crash save", payload, "invoice_contract::grid_saved")
	acknowledged, _ = value["acknowledged"].(map[string]any)
	if acknowledged["revision"] != strconv.Itoa(rev+1) {
		t.Fatalf("pre-crash save value %+v", value)
	}
	rev, committed = rev+1, committed+1
	crash()
	base, _, stop = gate3ServeInvoice(t, ctx, bundle, home, entry, snapshot, gate3MatrixPort)
	payload, _ = post("lost-ack replay", "tok-alice", "application/json", saveBody("op-m12", rev-1), 200)
	value = gate3Case(t, "lost-ack replay", payload, "invoice_contract::grid_saved")
	acknowledged, _ = value["acknowledged"].(map[string]any)
	if acknowledged["revision"] != strconv.Itoa(rev) {
		t.Fatalf("lost-ack replay value %+v", value)
	}
	lostChanged := `{"operation_id":"op-m12","revision":"` + strconv.Itoa(rev-1) + `","lines":[{"key":"k1","id":"changed","quantity":"4","price":"9.99"}]}`
	payload, _ = post("lost-ack changed", "tok-alice", "application/json", lostChanged, 409)
	gate3Case(t, "lost-ack changed", payload, "invoice_contract::grid_conflict")
	store = inspectInvoice(t, ctx, bundle, home, driver, db)
	requireInvoice(t, store, "7", strconv.Itoa(rev), "6", "Reordered")
	if len(store.Replay) != committed {
		t.Fatalf("lost-ack ledger %+v, want %d rows", store.Replay, committed)
	}

	stop()
	t.Logf("gate3 matrix: %d can assertions, json save/load/reorder/swap/uncertain legs live with row evidence", assertions)
}
