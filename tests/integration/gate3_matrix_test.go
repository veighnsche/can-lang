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
//   - swap-config regression: the served global policy plus the
//     current absence of per-element error admission, parsed from
//     real served bytes;
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
	"errors"
	"io"
	"net"
	"net/http"
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

const (
	gate3MatrixPort  = 18561
	gate3RoutePort   = 18562
	gate3AdapterPort = 18571
	gate3RacePort    = 18572
	gate3ReplayPort  = 18573
	gate3RenderPort  = 18574
	gate3LifePort    = 18575
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
// staged vendored bytes, mirroring the current lineage/lock digest
// format: the manifest digest covers the raw manifest bytes and the
// source digest covers the length-prefixed path-tagged source tree.
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
	var length [8]byte
	path := "invoice_contract/invoice_contract.can"
	binary.BigEndian.PutUint64(length[:], uint64(len(path)))
	sourceSum.Write(length[:])
	sourceSum.Write([]byte(path))
	binary.BigEndian.PutUint64(length[:], uint64(len(source)))
	sourceSum.Write(length[:])
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

// gate3Raw exchanges one verbatim HTTP/1.1 request over TCP and splits
// the status code from the body. Paths go out exactly as written: no
// client normalization touches dot segments or encoded separators.
func gate3Raw(t *testing.T, addr, request string) (int, string, string) {
	t.Helper()
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(10 * time.Second))
	if _, err := io.WriteString(conn, request); err != nil {
		t.Fatal(err)
	}
	raw, err := io.ReadAll(conn)
	if err != nil {
		t.Fatal(err)
	}
	head, body, _ := strings.Cut(string(raw), "\r\n\r\n")
	line, _, _ := strings.Cut(head, "\r\n")
	parts := strings.SplitN(line, " ", 3)
	if len(parts) < 2 {
		t.Fatalf("malformed raw status line %q", line)
	}
	status, err := strconv.Atoi(parts[1])
	if err != nil {
		t.Fatalf("malformed raw status %q", line)
	}
	return status, body, head
}

// gate3Abandon writes one complete request and closes before reading,
// modelling a client that disconnects after the server holds the full
// bytes (lazy/abandoned ingress).
func gate3Abandon(t *testing.T, addr, request string) {
	t.Helper()
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(conn, request); err != nil {
		conn.Close()
		t.Fatal(err)
	}
	conn.Close()
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
	row := requireInvoice(t, store, "7", "2", "2", "Acme <em>&\" 'coop'\"")
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
	requireInvoice(t, store, "7", "2", "2", "Acme <em>&\" 'coop'\"")
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
	requireInvoice(t, store, "7", "2", "2", "Acme <em>&\" 'coop'\"")
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
	requireInvoice(t, store, "7", "2", "2", "Acme <em>&\" 'coop'\"")
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

	// Swap-config regression on served bytes: the global policy keeps
	// the quiet 204/304 and every 4xx/5xx class out of swaps, and the
	// served form carries per-element hx-status admission generated
	// from the checked HTML action case table, so declared error
	// fragments swap into their connected target. 2xx/3xx outside the
	// quiet pair swap without needing an exception.
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
	for _, admission := range []string{
		`hx-status:403="{&quot;swap&quot;:&quot;innerHTML&quot;}"`,
		`hx-status:409="{&quot;swap&quot;:&quot;innerHTML&quot;}"`,
		`hx-status:422="{&quot;swap&quot;:&quot;innerHTML&quot;}"`,
		`hx-status:503="{&quot;swap&quot;:&quot;innerHTML&quot;}"`,
	} {
		if !strings.Contains(page, admission) {
			t.Fatalf("swap page lacks %s in %.800s", admission, page)
		}
	}
	if strings.Contains(page, "hx-status:200") {
		t.Fatalf("swap page admits a redundant 2xx exception in %.800s", page)
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

// TestGate3AdapterMatrix proves every adapter rejection never enters a
// protected handler: each leg snapshots the observable entry count
// (invoice revision plus replay ledger length) and requires it
// unchanged afterwards. A success control proves the counter observes
// real entries. The disconnect leg proves abandoned (lazy) ingress
// still commits exactly one effect under its owned lease, and the
// drop-replay fault proves a truthful 503 with a recorded commit
// verdict plus a safe identical retry after restore.
func TestGate3AdapterMatrix(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged gate 3 execution")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 14*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "gate3-adapter")
	if err != nil {
		t.Fatal(err)
	}
	root, home, driver, entry, assertions := stageInvoiceBundle(t, ctx, bundle, sourceRoot)
	db := seedInvoiceDB(t, ctx, bundle, home, driver, root)
	base, stop := serveInvoice(t, ctx, bundle, home, entry, db, gate3AdapterPort, "")
	defer stop()
	origin := "http://127.0.0.1:" + strconv.Itoa(gate3AdapterPort)
	addr := "127.0.0.1:" + strconv.Itoa(gate3AdapterPort)
	const savePath = "/api/tenants/1/invoices/7"
	const formPath = "/tenants/1/invoices/7"
	saveBody := func(op string, baseRev int) string {
		return `{"operation_id":"` + op + `","revision":"` + strconv.Itoa(baseRev) + `","lines":[{"key":"k1","id":"a","quantity":"2","price":"5.00"}]}`
	}
	entries := func() protectedEntries {
		t.Helper()
		return snapshotEntries(inspectInvoice(t, ctx, bundle, home, driver, db), "7")
	}
	method := func(verb, path, session, media, body string) (int, string, http.Header) {
		t.Helper()
		client := &http.Client{Timeout: 10 * time.Second}
		var reader io.Reader
		if body != "" {
			reader = strings.NewReader(body)
		}
		request, err := http.NewRequest(verb, base+path, reader)
		if err != nil {
			t.Fatal(err)
		}
		if media != "" {
			request.Header.Set("Content-Type", media)
		}
		if verb == "POST" {
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

	// Missing routes answer 404 without handler entry.
	for _, leg := range []struct {
		note, verb, path string
	}{
		{"unknown GET", "GET", "/no-such-path"},
		{"unknown POST", "POST", savePath + "/extra/deep"},
		{"captured prefix without invoice", "GET", "/api/tenants/1/invoices"},
		{"trailing slash", "GET", savePath + "/"},
	} {
		before := entries()
		status, body, _ := method(leg.verb, leg.path, "tok-alice", "application/json", `{"operation_id":"op-404"}`)
		if status != 404 || body != "Not Found" {
			t.Fatalf("%s: %d %q, want 404 Not Found", leg.note, status, body)
		}
		requireNoEntry(t, leg.note, before, entries())
	}

	// Recognized paths with the wrong method answer 405 carrying the
	// sorted Allow set, without handler entry.
	for _, leg := range []struct {
		note, verb, path, allow string
	}{
		{"api delete", "DELETE", savePath, "GET, POST"},
		{"api put", "PUT", savePath, "GET, POST"},
		{"form get", "GET", formPath, "POST"},
		{"form delete", "DELETE", formPath, "POST"},
		{"page post", "POST", "/invoices/form?tenant_id=1&invoice_id=7", "GET"},
		{"grid put", "PUT", "/invoice-grid?tenant=1&invoice=7", "GET"},
		{"health post", "POST", "/health", "GET"},
	} {
		before := entries()
		status, body, headers := method(leg.verb, leg.path, "tok-alice", "application/json", `{"operation_id":"op-405"}`)
		if status != 405 || body != "Method Not Allowed" {
			t.Fatalf("%s: %d %q, want 405 Method Not Allowed", leg.note, status, body)
		}
		if headers.Get("Allow") != leg.allow {
			t.Fatalf("%s: Allow %q, want %q", leg.note, headers.Get("Allow"), leg.allow)
		}
		requireNoEntry(t, leg.note, before, entries())
	}

	// Malformed captures fail before handler entry. Each leg uses an
	// admitted method so capture validation, not method matching,
	// decides the outcome.
	for _, leg := range []struct{ verb, path string }{
		{"GET", "/api/tenants/01/invoices/7"},
		{"GET", "/api/tenants/x/invoices/7"},
		{"POST", "/tenants/1/invoices/007"},
		{"POST", "/tenants/1/invoices/7x"},
	} {
		before := entries()
		status, body, _ := method(leg.verb, leg.path, "tok-alice", "application/json", `{"operation_id":"op-cap"}`)
		if status != 400 || body != "Bad Request" {
			t.Fatalf("capture %s %s: %d %q, want 400 Bad Request", leg.verb, leg.path, status, body)
		}
		requireNoEntry(t, "capture "+leg.path, before, entries())
	}

	// Encoded separators and dot segments never route into a handler
	// as data: whatever the native layer admits, the observable
	// entries stay fixed and no success carries stored content.
	before := entries()
	status, body, _ := method("GET", "/api/tenants/1%2F2/invoices/7", "tok-alice", "", "")
	if status != 400 && status != 404 {
		t.Fatalf("encoded separator: %d %q, want 400 or 404", status, body)
	}
	requireNoEntry(t, "encoded separator", before, entries())
	// Dot segments resolve through native normalization, never into a
	// handler as data: climbing to a neighbor answers the neighbor's
	// denial, and a self-dot answers byte-identically to the direct
	// address. Both leave the observable entries fixed.
	before = entries()
	raw := "GET /api/tenants/1/invoices/7/../8 HTTP/1.1\r\nHost: " + addr + "\r\nCookie: session=tok-alice\r\nConnection: close\r\n\r\n"
	status, body, _ = gate3Raw(t, addr, raw)
	if status != 400 && status != 404 && status != 403 {
		t.Fatalf("dot climb: %d %q, want 400, 404 or denied 403", status, body)
	}
	if status == 403 {
		gate3Case(t, "dot climb", body, "invoice_contract::grid_load_forbidden")
	}
	requireNoEntry(t, "dot climb", before, entries())
	t.Logf("dot climb: %d without entry effect", status)
	directStatus, directBody, _ := invoiceGet(t, base, savePath, "tok-alice")
	self := "GET /api/tenants/./1/invoices/7 HTTP/1.1\r\nHost: " + addr + "\r\nCookie: session=tok-alice\r\nConnection: close\r\n\r\n"
	status, body, _ = gate3Raw(t, addr, self)
	if status != directStatus || body != directBody {
		t.Fatalf("dot self: %d %.300s, want byte-identical %d", status, body, directStatus)
	}
	requireNoEntry(t, "dot self", before, entries())

	// Media, syntax and budget failures stay pre-handler on both
	// channels; the contract limits are 8192 JSON bytes and 2048 form
	// bytes with 64 keyed rows.
	transport := []struct {
		note, path, media, body string
		status                 int
		text                   string
	}{
		{"json bad media", savePath, "text/plain", saveBody("op-t1", 1), 415, "Unsupported Media Type"},
		{"json malformed", savePath, "application/json", "not json", 400, "Bad Request"},
		{"json oversize", savePath, "application/json", `{"operation_id":"op-t2","revision":"1","lines":[{"key":"k1","id":"` + strings.Repeat("p", 9000) + `","quantity":"2","price":"5.00"}]}`, 413, "Payload Too Large"},
		{"form bad media", formPath, "application/json", saveBody("op-t3", 1), 415, "Unsupported Media Type"},
		{"form oversize", formPath, "application/x-www-form-urlencoded", "seats=2&details=" + strings.Repeat("d", 4096) + "&revision=1&lines_order=k1&lines[k1][id]=a&lines[k1][quantity]=2&lines[k1][price]=5.00", 413, "Payload Too Large"},
	}
	for _, leg := range transport {
		before := entries()
		status, body, _ := method("POST", leg.path, "tok-alice", leg.media, leg.body)
		if status != leg.status || body != leg.text {
			t.Fatalf("%s: %d %q, want %d %q", leg.note, status, body, leg.status, leg.text)
		}
		requireNoEntry(t, leg.note, before, entries())
	}
	// A missing content type is a media failure on both channels.
	for _, path := range []string{savePath, formPath} {
		before := entries()
		status, body, _ := method("POST", path, "tok-alice", "", saveBody("op-t4", 1))
		if status != 415 || body != "Unsupported Media Type" {
			t.Fatalf("missing media %s: %d %q, want 415", path, status, body)
		}
		requireNoEntry(t, "missing media "+path, before, entries())
	}
	// Sixty-five keyed rows exceed the declared rows_limit and render
	// the structural 422 fragment without handler entry.
	many := url.Values{}
	many.Set("seats", "2")
	many.Set("details", "many")
	many.Set("revision", "1")
	for i := 0; i < 65; i++ {
		many.Add("lines_order", "k"+strconv.Itoa(i))
	}
	before = entries()
	status, body, _ = postInvoiceForm(t, base, formPath, "tok-alice", origin, many)
	if status != 422 || !strings.Contains(body, "invoice form rejected") || !strings.Contains(body, `role="alert"`) || !strings.Contains(body, "row_limit") {
		t.Fatalf("rows over limit: %d %.600s, want 422 row_limit alert", status, body)
	}
	requireNoEntry(t, "rows over limit", before, entries())

	// Success control: one valid save advances revision and ledger by
	// exactly one, proving the counter observes real entries.
	before = entries()
	status, payload, _ := postInvoiceJSON(t, base, savePath, "tok-alice", origin, "application/json", saveBody("op-a1", 1))
	if status != 200 {
		t.Fatalf("adapter control save: %d %s", status, payload)
	}
	requireOneEntry(t, "adapter control save", "2", before, entries())

	// Disconnect: a fully written abandoned save still commits exactly
	// one effect under its owned lease; the identical replay returns
	// 200 and changed content conflicts.
	abandoned := saveBody("op-a2", 2)
	gate3Abandon(t, addr, "POST "+savePath+" HTTP/1.1\r\nHost: "+addr+"\r\nContent-Type: application/json\r\nContent-Length: "+strconv.Itoa(len(abandoned))+"\r\nOrigin: "+origin+"\r\nCookie: session=tok-alice\r\nConnection: close\r\n\r\n"+abandoned)
	deadline := time.Now().Add(20 * time.Second)
	for {
		store := inspectInvoice(t, ctx, bundle, home, driver, db)
		done := false
		for _, row := range store.Replay {
			if row.OperationID == "op-a2" {
				done = true
			}
		}
		if done {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("abandoned save never committed its replay row")
		}
		time.Sleep(250 * time.Millisecond)
	}
	store := inspectInvoice(t, ctx, bundle, home, driver, db)
	requireInvoice(t, store, "7", "3", "2", "Acme <em>&\" 'coop'\"")
	requireReplay(t, store, "op-a2", "3")
	before = entries()
	status, first, _ := postInvoiceJSON(t, base, savePath, "tok-alice", origin, "application/json", abandoned)
	if status != 200 {
		t.Fatalf("abandoned replay: %d %s", status, first)
	}
	status, second, _ := postInvoiceJSON(t, base, savePath, "tok-alice", origin, "application/json", abandoned)
	if status != 200 || second != first {
		t.Fatalf("abandoned reread: %d, want identical 200 bytes", status)
	}
	changed := `{"operation_id":"op-a2","revision":"2","lines":[{"key":"k1","id":"changed","quantity":"2","price":"5.00"}]}`
	status, payload, _ = postInvoiceJSON(t, base, savePath, "tok-alice", origin, "application/json", changed)
	if status != 409 {
		t.Fatalf("abandoned changed: %d %s, want 409", status, payload)
	}
	requireNoEntry(t, "abandoned replays", before, entries())

	// Drop-replay fault: the write fails truthfully at 503 without
	// committing, and the identical retry commits after restore. The
	// mid-fault revision read bypasses the dropped ledger table.
	before = entries()
	faultInvoice(t, ctx, bundle, home, driver, db, "drop-replay")
	status, payload, _ = postInvoiceJSON(t, base, savePath, "tok-alice", origin, "application/json", saveBody("op-a3", 3))
	if status != 503 {
		t.Fatalf("replay outage save: %d %s, want 503", status, payload)
	}
	gate3Case(t, "replay outage save", payload, "invoice_contract::grid_unavailable")
	if mid := invoiceRevision(t, ctx, bundle, home, driver, db, "7"); mid != before.revision {
		t.Fatalf("drop-replay fault moved revision %s to %s", before.revision, mid)
	}
	t.Logf("fault drop-replay: committed=false (revision %s held, ledger table dropped)", before.revision)
	setup := invoiceDriver(t, ctx, bundle, home, driver, "setup", db, filepath.Join(root, "schema.sql"))
	var setupReport struct {
		Tables int `json:"tables"`
	}
	if err := json.Unmarshal(setup, &setupReport); err != nil || setupReport.Tables != 5 {
		t.Fatalf("invalid invoice restore report %v %s", err, string(setup))
	}
	status, payload, _ = postInvoiceJSON(t, base, savePath, "tok-alice", origin, "application/json", saveBody("op-a3", 3))
	if status != 200 {
		t.Fatalf("replay recovery save: %d %s", status, payload)
	}
	value := gate3Case(t, "replay recovery save", payload, "invoice_contract::grid_saved")
	acknowledged, _ := value["acknowledged"].(map[string]any)
	if acknowledged["revision"] != "4" {
		t.Fatalf("replay recovery save value %+v", value)
	}
	// The restore recreates an empty ledger, so the retry's commit
	// verdict compares revisions, not ledger lengths.
	after := entries()
	if after.revision != "4" {
		t.Fatalf("drop-replay retry left revision %s, want 4", after.revision)
	}
	requireReplay(t, inspectInvoice(t, ctx, bundle, home, driver, db), "op-a3", "4")
	t.Logf("fault drop-replay retry: committed=true (revision %s -> %s on a restored ledger)", before.revision, after.revision)
	t.Logf("gate3 adapter: %d can assertions, 404/405/capture/separator/media/budget legs without entry, disconnect and replay-outage legs with row evidence", assertions)
}

// gate3RacePost fires one JSON save and reports its status with body.
type gate3RacePost struct {
	operation string
	status    int
	body      string
}

func gate3RaceSave(base, path, session, origin, body string) gate3RacePost {
	operation := ""
	if i := strings.Index(body, `"operation_id":"`); i >= 0 {
		rest := body[i+len(`"operation_id":"`):]
		operation, _, _ = strings.Cut(rest, `"`)
	}
	client := &http.Client{Timeout: 30 * time.Second}
	request, err := http.NewRequest("POST", base+path, strings.NewReader(body))
	if err != nil {
		return gate3RacePost{operation: operation, status: -1, body: err.Error()}
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", origin)
	if session != "" {
		request.AddCookie(&http.Cookie{Name: "session", Value: session})
	}
	response, err := client.Do(request)
	if err != nil {
		return gate3RacePost{operation: operation, status: -1, body: err.Error()}
	}
	defer response.Body.Close()
	payload, _ := io.ReadAll(response.Body)
	return gate3RacePost{operation: operation, status: response.StatusCode, body: string(payload)}
}

// TestGate3Concurrency proves a concurrent revision or membership change
// blocks stale/unauthorized commit: distinct operations racing one base
// revision commit exactly once, identical operations racing commit one
// shared effect with byte-identical results, and membership flaps admit
// only authorized commits while every landing keeps the revision and
// ledger counts coherent.
func TestGate3Concurrency(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged gate 3 execution")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 14*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "gate3-race")
	if err != nil {
		t.Fatal(err)
	}
	root, home, driver, entry, assertions := stageInvoiceBundle(t, ctx, bundle, sourceRoot)
	db := seedInvoiceDB(t, ctx, bundle, home, driver, root)
	base, stop := serveInvoice(t, ctx, bundle, home, entry, db, gate3RacePort, "")
	defer stop()
	origin := "http://127.0.0.1:" + strconv.Itoa(gate3RacePort)
	const savePath = "/api/tenants/1/invoices/7"
	saveBody := func(op string, baseRev int) string {
		return `{"operation_id":"` + op + `","revision":"` + strconv.Itoa(baseRev) + `","lines":[{"key":"k1","id":"a","quantity":"2","price":"5.00"}]}`
	}
	race := func(bodies []string) []gate3RacePost {
		t.Helper()
		start := make(chan struct{})
		results := make([]gate3RacePost, len(bodies))
		var group sync.WaitGroup
		for i, body := range bodies {
			group.Add(1)
			go func() {
				defer group.Done()
				<-start
				results[i] = gate3RaceSave(base, savePath, "tok-alice", origin, body)
			}()
		}
		close(start)
		group.Wait()
		return results
	}
	revision := func() string {
		t.Helper()
		store := inspectInvoice(t, ctx, bundle, home, driver, db)
		for _, row := range store.Invoice {
			if row.ID == "7" {
				return row.Rev
			}
		}
		t.Fatal("invoice 7 missing")
		return ""
	}

	// Distinct operations racing one base revision: exactly one save
	// commits, every loser conflicts, and the ledger grows by one.
	var bodies []string
	for i := 0; i < 8; i++ {
		bodies = append(bodies, saveBody("op-r"+strconv.Itoa(i), 1))
	}
	results := race(bodies)
	winners, conflicts := 0, 0
	var winner gate3RacePost
	for _, result := range results {
		switch result.status {
		case 200:
			winners++
			winner = result
		case 409:
			conflicts++
		default:
			t.Fatalf("revision race %s: %d %s, want 200 once and 409 else", result.operation, result.status, result.body)
		}
	}
	if winners != 1 || conflicts != 7 {
		t.Fatalf("revision race: %d winners %d conflicts, want 1 and 7", winners, conflicts)
	}
	value := gate3Case(t, "revision race winner", winner.body, "invoice_contract::grid_saved")
	acknowledged, _ := value["acknowledged"].(map[string]any)
	if value["operation_id"] != winner.operation || acknowledged["revision"] != "2" {
		t.Fatalf("revision race winner value %+v", value)
	}
	if revision() != "2" {
		t.Fatalf("revision race left revision %s, want 2", revision())
	}
	store := inspectInvoice(t, ctx, bundle, home, driver, db)
	requireReplay(t, store, winner.operation, "2")
	if len(store.Replay) != 1 {
		t.Fatalf("revision race ledger %+v, want one row", store.Replay)
	}
	t.Logf("revision race: %s committed revision 2, seven losers conflicted", winner.operation)

	// Identical operations racing: every landing converges to one
	// shared effect with byte-identical 200 results and one ledger
	// row. A loser that slips past the lookup before the winner
	// commits may answer 503 once; its identical retry replays.
	var same []string
	for i := 0; i < 6; i++ {
		same = append(same, saveBody("op-same", 2))
	}
	results = race(same)
	for _, result := range results {
		if result.status != 200 && result.status != 503 {
			t.Fatalf("identical race %s: %d %s, want 200 or one 503 retry", result.operation, result.status, result.body)
		}
	}
	var converged []string
	for range results {
		status, payload, _ := postInvoiceJSON(t, base, savePath, "tok-alice", origin, "application/json", saveBody("op-same", 2))
		if status != 200 {
			t.Fatalf("identical converge: %d %s, want 200", status, payload)
		}
		converged = append(converged, payload)
	}
	for _, payload := range converged[1:] {
		if payload != converged[0] {
			t.Fatal("identical race converged to differing bytes")
		}
	}
	value = gate3Case(t, "identical race", converged[0], "invoice_contract::grid_saved")
	acknowledged, _ = value["acknowledged"].(map[string]any)
	if acknowledged["revision"] != "3" {
		t.Fatalf("identical race value %+v", value)
	}
	if revision() != "3" {
		t.Fatalf("identical race left revision %s, want 3", revision())
	}
	store = inspectInvoice(t, ctx, bundle, home, driver, db)
	requireReplay(t, store, "op-same", "3")
	if len(store.Replay) != 2 {
		t.Fatalf("identical race ledger %+v, want two rows", store.Replay)
	}
	t.Logf("identical race: one shared effect at revision 3 with byte-identical results")

	// Deterministic membership gate: removing alice's row denies saves
	// and loads without disclosure; restoring it admits saves again.
	setMembership(t, ctx, bundle, home, driver, db, "alice", "1", false)
	status, payload, _ := postInvoiceJSON(t, base, savePath, "tok-alice", origin, "application/json", saveBody("op-u1", 3))
	if status != 403 {
		t.Fatalf("unmembered save: %d %s, want 403", status, payload)
	}
	value = gate3Case(t, "unmembered save", payload, "invoice_contract::grid_forbidden")
	if value["operation_id"] != "op-u1" || len(value) != 2 {
		t.Fatalf("unmembered save value %+v discloses or mismatches", value)
	}
	status, body, _ := invoiceGet(t, base, savePath, "tok-alice")
	if status != 403 {
		t.Fatalf("unmembered load: %d %s, want 403", status, body)
	}
	gate3Case(t, "unmembered load", body, "invoice_contract::grid_load_forbidden")
	if revision() != "3" {
		t.Fatalf("unmembered legs moved revision to %s", revision())
	}
	setMembership(t, ctx, bundle, home, driver, db, "alice", "1", true)
	status, payload, _ = postInvoiceJSON(t, base, savePath, "tok-alice", origin, "application/json", saveBody("op-u2", 3))
	if status != 200 {
		t.Fatalf("restored save: %d %s, want 200", status, payload)
	}
	value = gate3Case(t, "restored save", payload, "invoice_contract::grid_saved")
	acknowledged, _ = value["acknowledged"].(map[string]any)
	if acknowledged["revision"] != "4" {
		t.Fatalf("restored save value %+v", value)
	}

	// Membership flap racing saves: the flap runs while eight distinct
	// saves race one base revision. Every landing keeps the counts
	// coherent: each 200 advanced the revision by exactly one, every
	// 200 operation owns its ledger row, and denials disclose nothing.
	flapDone := make(chan error, 1)
	go func() {
		for i := 0; i < 4; i++ {
			for _, mode := range []string{"unmember", "member"} {
				if out, err := invoiceDriverRaw(ctx, bundle, home, driver, mode, db, "alice", "1"); err != nil {
					flapDone <- errors.New(mode + ": " + string(out))
					return
				}
			}
		}
		flapDone <- nil
	}()
	var flapBodies []string
	for i := 0; i < 8; i++ {
		flapBodies = append(flapBodies, saveBody("op-f"+strconv.Itoa(i), 4))
	}
	results = race(flapBodies)
	if err := <-flapDone; err != nil {
		t.Fatalf("membership flap: %v", err)
	}
	// Restore idempotently: the flap ends present, so remove-then-add
	// converges regardless of the landing.
	setMembership(t, ctx, bundle, home, driver, db, "alice", "1", false)
	setMembership(t, ctx, bundle, home, driver, db, "alice", "1", true)
	var committed []gate3RacePost
	for _, result := range results {
		switch result.status {
		case 200:
			committed = append(committed, result)
		case 403:
			value := gate3Case(t, "flap denial "+result.operation, result.body, "invoice_contract::grid_forbidden")
			if value["operation_id"] != result.operation || len(value) != 2 {
				t.Fatalf("flap denial value %+v discloses or mismatches", value)
			}
		case 409:
			gate3Case(t, "flap conflict "+result.operation, result.body, "invoice_contract::grid_conflict")
		default:
			t.Fatalf("flap race %s: %d %s, want 200, 403 or 409", result.operation, result.status, result.body)
		}
	}
	if len(committed) > 1 {
		t.Fatalf("flap race committed %d saves on one base revision", len(committed))
	}
	final := revision()
	if len(committed) == 1 {
		value := gate3Case(t, "flap winner", committed[0].body, "invoice_contract::grid_saved")
		acknowledged, _ := value["acknowledged"].(map[string]any)
		if acknowledged["revision"] != "5" || final != "5" {
			t.Fatalf("flap winner value %+v revision %s", value, final)
		}
		requireReplay(t, inspectInvoice(t, ctx, bundle, home, driver, db), committed[0].operation, "5")
	} else if final != "4" {
		t.Fatalf("flap race committed nothing but left revision %s", final)
	}
	store = inspectInvoice(t, ctx, bundle, home, driver, db)
	wantRows := 3 + len(committed)
	if len(store.Replay) != wantRows {
		t.Fatalf("flap race ledger holds %d rows, want %d", len(store.Replay), wantRows)
	}
	t.Logf("membership flap: %d commit(s), revision %s, %d ledger rows, every landing coherent", len(committed), final, wantRows)

	// Recovery control: membership present, a fresh operation on the
	// current revision commits.
	freshRev, _ := strconv.Atoi(final)
	status, payload, _ = postInvoiceJSON(t, base, savePath, "tok-alice", origin, "application/json", saveBody("op-fresh", freshRev))
	if status != 200 {
		t.Fatalf("post-flap save: %d %s, want 200", status, payload)
	}
	value = gate3Case(t, "post-flap save", payload, "invoice_contract::grid_saved")
	acknowledged, _ = value["acknowledged"].(map[string]any)
	if acknowledged["revision"] != strconv.Itoa(freshRev+1) {
		t.Fatalf("post-flap save value %+v", value)
	}
	t.Logf("gate3 concurrency: %d can assertions, revision/identical/membership races live with row evidence", assertions)
}

// TestGate3ReplayExpiry proves revoked actors cannot recover prior saved
// results, live replay replays without sliding retention, expired replay
// conflicts with the row gone, and expired operation IDs that return
// with a fresh base commit new revisions instead of reusing recorded
// ones. The server runs a five-minute retention window so the live legs
// place rows a full minute on either side of the server-clock cutoff;
// the exact at-boundary millisecond stays pinned by the
// save_replay_at_edge model assertion (strict less-than keeps it live).
func TestGate3ReplayExpiry(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged gate 3 execution")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 14*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "gate3-replay")
	if err != nil {
		t.Fatal(err)
	}
	root, home, driver, entry, assertions := stageInvoiceBundle(t, ctx, bundle, sourceRoot)
	db := seedInvoiceDB(t, ctx, bundle, home, driver, root)
	const window = int64(300000)
	base, stop := serveInvoice(t, ctx, bundle, home, entry, db, gate3ReplayPort, strconv.FormatInt(window, 10))
	defer stop()
	origin := "http://127.0.0.1:" + strconv.Itoa(gate3ReplayPort)
	const savePath = "/api/tenants/1/invoices/7"
	lines := `[{"key":"k1","id":"a","quantity":"2","price":"5.00"}]`
	saveBody := func(op string, baseRev int) string {
		return `{"operation_id":"` + op + `","revision":"` + strconv.Itoa(baseRev) + `","lines":` + lines + `}`
	}
	entries := func() protectedEntries {
		t.Helper()
		return snapshotEntries(inspectInvoice(t, ctx, bundle, home, driver, db), "7")
	}
	revision := func() string { t.Helper(); return entries().revision }

	// Baseline save whose result a revoked actor must not recover.
	status, saved, _ := postInvoiceJSON(t, base, savePath, "tok-alice", origin, "application/json", saveBody("op-v1", 1))
	if status != 200 {
		t.Fatalf("revoke setup save: %d %s", status, saved)
	}
	before := entries()
	ledger := requireReplay(t, inspectInvoice(t, ctx, bundle, home, driver, db), "op-v1", "2")
	revokeSession(t, ctx, bundle, home, driver, db, "tok-alice", strconv.FormatInt(time.Now().UnixMilli(), 10))

	// Revoked replay: identical and changed retries share one
	// nondisclosing 403 carrying only the caller-sent operation id,
	// and the load denials match. Nothing leaks the recorded result.
	for _, leg := range []struct {
		note, body string
	}{
		{"revoked identical", saveBody("op-v1", 1)},
		{"revoked changed", `{"operation_id":"op-v1","revision":"1","lines":[{"key":"k1","id":"changed","quantity":"2","price":"5.00"}]}`},
		{"revoked fresh", saveBody("op-v2", 2)},
	} {
		status, payload, _ := postInvoiceJSON(t, base, savePath, "tok-alice", origin, "application/json", leg.body)
		if status != 403 {
			t.Fatalf("%s: %d %s, want 403", leg.note, status, payload)
		}
		value := gate3Case(t, leg.note, payload, "invoice_contract::grid_forbidden")
		if value["operation_id"] != "op-v1" && value["operation_id"] != "op-v2" {
			t.Fatalf("%s value %+v mismatches the caller-sent id", leg.note, value)
		}
		if len(value) != 2 || strings.Contains(payload, "grid_saved") || strings.Contains(payload, "Acme") {
			t.Fatalf("%s value %+v discloses the recorded result", leg.note, value)
		}
	}
	status, body, _ := invoiceGet(t, base, savePath, "tok-alice")
	if status != 403 {
		t.Fatalf("revoked load: %d %s, want 403", status, body)
	}
	gate3Case(t, "revoked load", body, "invoice_contract::grid_load_forbidden")
	requireNoEntry(t, "revoked legs", before, entries())

	// Unrevoking restores the recorded result byte-identically with
	// its retention stamp unmoved.
	revokeSession(t, ctx, bundle, home, driver, db, "tok-alice", "0")
	status, replayed, _ := postInvoiceJSON(t, base, savePath, "tok-alice", origin, "application/json", saveBody("op-v1", 1))
	if status != 200 || replayed != saved {
		t.Fatalf("unrevoked replay: %d, want identical 200 bytes", status)
	}
	again := requireReplay(t, inspectInvoice(t, ctx, bundle, home, driver, db), "op-v1", "2")
	if again.Recorded != ledger.Recorded {
		t.Fatalf("unrevoked replay slid retention %s to %s", ledger.Recorded, again.Recorded)
	}
	requireNoEntry(t, "unrevoked replay", before, entries())

	// Live replay: a row a minute inside the cutoff replays
	// byte-identically without sliding its stamp or touching the
	// revision.
	status, live, _ := postInvoiceJSON(t, base, savePath, "tok-alice", origin, "application/json", saveBody("op-e1", 2))
	if status != 200 {
		t.Fatalf("expiry setup save: %d %s", status, live)
	}
	liveStamp := strconv.FormatInt(time.Now().UnixMilli()-window+60000, 10)
	touchReplay(t, ctx, bundle, home, driver, db, "alice", "1", "7", "op-e1", liveStamp)
	before = entries()
	status, replayed, _ = postInvoiceJSON(t, base, savePath, "tok-alice", origin, "application/json", saveBody("op-e1", 2))
	if status != 200 || replayed != live {
		t.Fatalf("live replay: %d, want identical 200 bytes", status)
	}
	requireNoEntry(t, "live replay", before, entries())
	if held := requireReplay(t, inspectInvoice(t, ctx, bundle, home, driver, db), "op-e1", "3"); held.Recorded != liveStamp {
		t.Fatalf("live replay slid retention %s to %s", liveStamp, held.Recorded)
	}

	// Expired replay: a row a minute past the cutoff conflicts on its
	// advanced revision and is gone afterwards; nothing reuses the
	// recorded revision.
	deadStamp := strconv.FormatInt(time.Now().UnixMilli()-window-60000, 10)
	touchReplay(t, ctx, bundle, home, driver, db, "alice", "1", "7", "op-e1", deadStamp)
	status, payload, _ := postInvoiceJSON(t, base, savePath, "tok-alice", origin, "application/json", saveBody("op-e1", 2))
	if status != 409 {
		t.Fatalf("expired replay: %d %s, want 409", status, payload)
	}
	gate3Case(t, "expired replay", payload, "invoice_contract::grid_conflict")
	if revision() != "3" {
		t.Fatalf("expired replay moved revision to %s", revision())
	}
	for _, row := range inspectInvoice(t, ctx, bundle, home, driver, db).Replay {
		if row.OperationID == "op-e1" {
			t.Fatalf("expired replay kept %+v", row)
		}
	}

	// The expired operation ID returning with a fresh base commits a
	// new revision instead of reusing its recorded one.
	status, payload, _ = postInvoiceJSON(t, base, savePath, "tok-alice", origin, "application/json", saveBody("op-e1", 3))
	if status != 200 {
		t.Fatalf("expired id fresh base: %d %s, want 200", status, payload)
	}
	value := gate3Case(t, "expired id fresh base", payload, "invoice_contract::grid_saved")
	acknowledged, _ := value["acknowledged"].(map[string]any)
	if acknowledged["revision"] != "4" {
		t.Fatalf("expired id fresh base value %+v, want new revision 4", value)
	}
	requireReplay(t, inspectInvoice(t, ctx, bundle, home, driver, db), "op-e1", "4")

	// Cleanup keeps the near-boundary live row while the next commit
	// expires an ancient one.
	liveStamp = strconv.FormatInt(time.Now().UnixMilli()-window+60000, 10)
	touchReplay(t, ctx, bundle, home, driver, db, "alice", "1", "7", "op-e1", liveStamp)
	status, payload, _ = postInvoiceJSON(t, base, savePath, "tok-alice", origin, "application/json", saveBody("op-e2", 4))
	if status != 200 {
		t.Fatalf("cleanup setup save: %d %s", status, payload)
	}
	deadStamp = strconv.FormatInt(time.Now().UnixMilli()-window-60000, 10)
	touchReplay(t, ctx, bundle, home, driver, db, "alice", "1", "7", "op-e2", deadStamp)
	status, payload, _ = postInvoiceJSON(t, base, savePath, "tok-alice", origin, "application/json", saveBody("op-e3", 5))
	if status != 200 {
		t.Fatalf("cleanup commit save: %d %s", status, payload)
	}
	if revision() != "6" {
		t.Fatalf("cleanup left revision %s, want 6", revision())
	}
	seen := map[string]bool{}
	for _, row := range inspectInvoice(t, ctx, bundle, home, driver, db).Replay {
		seen[row.OperationID] = true
	}
	if !seen["op-v1"] || !seen["op-e1"] || !seen["op-e3"] || seen["op-e2"] {
		t.Fatalf("cleanup ledger keeps the live rows and drops only the ancient one: %+v", seen)
	}
	t.Logf("gate3 replay: %d can assertions, revoked/live/expired replay and revision invariants live with row evidence", assertions)
}

// TestGate3RendererFault proves a post-commit renderer fault fails closed:
// a committed write whose stored row the HTML renderer cannot mint
// answers 500 with a fixed body carrying no invoice content, the rows
// stay intact and readable over JSON, and removing the row restores the
// page. A store outage likewise renders the 503 fragment into the
// admitted shape instead of failing the swap contract.
func TestGate3RendererFault(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged gate 3 execution")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 14*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "gate3-render")
	if err != nil {
		t.Fatal(err)
	}
	root, home, driver, entry, assertions := stageInvoiceBundle(t, ctx, bundle, sourceRoot)
	db := seedInvoiceDB(t, ctx, bundle, home, driver, root)
	base, stop := serveInvoice(t, ctx, bundle, home, entry, db, gate3RenderPort, "")
	defer stop()
	origin := "http://127.0.0.1:" + strconv.Itoa(gate3RenderPort)
	const formPath = "/tenants/1/invoices/7"
	const pagePath = "/invoices/form?tenant_id=1&invoice_id=7"
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
	entries := func() protectedEntries {
		t.Helper()
		return snapshotEntries(inspectInvoice(t, ctx, bundle, home, driver, db), "7")
	}

	// Commit one form save with row evidence first.
	status, body, _ := postInvoiceForm(t, base, formPath, "tok-alice", origin, formValues("4", "Rush order", "1"))
	if status != 200 || !strings.Contains(body, "saved revision 2") {
		t.Fatalf("render setup save: %d %s", status, body)
	}
	requireInvoice(t, inspectInvoice(t, ctx, bundle, home, driver, db), "7", "2", "4", "Rush order")

	// A stored row the renderer cannot mint fails the page into a
	// fixed 500 carrying no invoice content and no entry effect.
	injectLine(t, ctx, bundle, home, driver, db, "7", "k 1", "a", "2", "500", "2")
	before := entries()
	status, page, _ := invoiceGet(t, base, pagePath, "tok-alice")
	if status != 500 || page != "error" {
		t.Fatalf("renderer fault page: %d %q, want 500 error", status, page)
	}
	requireNoEntry(t, "renderer fault", before, entries())
	store := inspectInvoice(t, ctx, bundle, home, driver, db)
	row := requireInvoice(t, store, "7", "2", "4", "Rush order")
	if len(row.Lines) != 2 {
		t.Fatalf("renderer fault lost committed rows %+v", row.Lines)
	}

	// The store itself is healthy: the JSON load still answers the
	// current snapshot including the unmintable key.
	status, loaded, _ := invoiceGet(t, base, "/api/tenants/1/invoices/7", "tok-alice")
	if status != 200 {
		t.Fatalf("renderer fault load: %d %s, want 200", status, loaded)
	}
	value := gate3Case(t, "renderer fault load", loaded, "invoice_contract::grid_loaded")
	current, _ := value["current"].(map[string]any)
	if current["revision"] != "2" {
		t.Fatalf("renderer fault load value %+v", value)
	}

	// Removing the row restores the page with the committed content.
	deleteLine(t, ctx, bundle, home, driver, db, "7", "k 1")
	status, page, _ = invoiceGet(t, base, pagePath, "tok-alice")
	if status != 200 || !strings.Contains(page, "Rush order") || !strings.Contains(page, `value="2"`) {
		t.Fatalf("renderer recovery page: %d %.500s, want 200 with committed content", status, page)
	}

	// A store outage renders the 503 fragment with its alert region
	// and commits nothing; the page degrades to its 503 notice; the
	// identical save commits after restore.
	before = entries()
	faultInvoice(t, ctx, bundle, home, driver, db, "drop-lines")
	status, body, _ = postInvoiceForm(t, base, formPath, "tok-alice", origin, formValues("4", "Rush order", "2"))
	if status != 503 || !strings.Contains(body, "unavailable") || !strings.Contains(body, `role="alert"`) {
		t.Fatalf("form outage save: %d %s, want 503 unavailable alert", status, body)
	}
	status, page, _ = invoiceGet(t, base, pagePath, "tok-alice")
	if status != 503 || !strings.Contains(page, "Invoice unavailable") {
		t.Fatalf("form outage page: %d %.300s, want 503 notice", status, page)
	}
	// The mid-fault revision read bypasses the dropped lines table.
	if mid := invoiceRevision(t, ctx, bundle, home, driver, db, "7"); mid != before.revision {
		t.Fatalf("drop-lines form fault moved revision %s to %s", before.revision, mid)
	}
	t.Logf("fault drop-lines form: committed=false (revision %s held, lines table dropped)", before.revision)
	setup := invoiceDriver(t, ctx, bundle, home, driver, "setup", db, filepath.Join(root, "schema.sql"))
	var setupReport struct {
		Tables int `json:"tables"`
	}
	if err := json.Unmarshal(setup, &setupReport); err != nil || setupReport.Tables != 5 {
		t.Fatalf("invalid invoice restore report %v %s", err, string(setup))
	}
	status, body, _ = postInvoiceForm(t, base, formPath, "tok-alice", origin, formValues("4", "Rush order", "2"))
	if status != 200 || !strings.Contains(body, "saved revision 3") {
		t.Fatalf("form recovery save: %d %s, want 200 saved revision 3", status, body)
	}
	if !recordFaultOutcome(t, "drop-lines form retry", before, entries()) {
		t.Fatal("drop-lines form retry committed nothing")
	}
	t.Logf("gate3 render: %d can assertions, post-commit renderer 500, JSON health, page recovery and form 503 fragment live with row evidence", assertions)
}

// gate3CallSites counts emitted call sites of one identifier across the
// staged build directory, logging every matching line on mismatch.
func gate3CallSites(t *testing.T, dir, identifier string, want int) {
	t.Helper()
	var matches []string
	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".ts") {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, line := range strings.Split(string(data), "\n") {
			if strings.Contains(line, identifier+"(") && !strings.Contains(strings.TrimSpace(line), "import ") {
				matches = append(matches, path+": "+strings.TrimSpace(line))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != want {
		t.Fatalf("%s call sites: %d, want %d:\n%s", identifier, len(matches), want, strings.Join(matches, "\n"))
	}
	t.Logf("%s: %d call site", identifier, want)
}

// TestGate3Lifecycle proves one pool, startup refusal, clean
// shutdown/drainage and request-token revocation: the staged server
// bundle opens its pool at exactly one call site and closes it at
// exactly one, missing databases and occupied ports fail closed,
// SIGTERM drains and exits clean with the ledger intact across
// restart, an in-flight trickled request converges to exactly one
// effect however the shutdown landed, and the artifact's own
// request-lifetime suite passes under the staged sidecar.
func TestGate3Lifecycle(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged gate 3 execution")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 14*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "gate3-life")
	if err != nil {
		t.Fatal(err)
	}
	root, home, driver, entry, assertions := stageInvoiceBundle(t, ctx, bundle, sourceRoot)
	db := seedInvoiceDB(t, ctx, bundle, home, driver, root)
	origin := "http://127.0.0.1:" + strconv.Itoa(gate3LifePort)
	const savePath = "/api/tenants/1/invoices/7"
	saveBody := func(op string, baseRev int) string {
		return `{"operation_id":"` + op + `","revision":"` + strconv.Itoa(baseRev) + `","lines":[{"key":"k1","id":"a","quantity":"2","price":"5.00"}]}`
	}

	// One pool: exactly one open call site and one close call site in
	// the emitted server bundle.
	buildDir := filepath.Dir(entry)
	gate3CallSites(t, buildDir, "sqlite_open_file", 1)
	gate3CallSites(t, buildDir, "pool_close", 1)

	// Startup refusal: a database path in a missing directory fails
	// closed before serving.
	expectInvoiceStartupFailure(t, ctx, bundle, home, entry, map[string]string{
		"INVOICE_DB":    filepath.Join(home, "no-such-dir", "inv.sqlite"),
		"PUBLIC_ORIGIN": origin,
	}, gate3LifePort+10, "missing-db")
	t.Log("startup refusal missing-db: nonzero exit before serving")

	// Startup refusal: an occupied port fails closed. The first
	// server also anchors the shutdown legs below.
	snapshot := snapshotMap(t, home, "snapshot-gate3-life", map[string]string{"INVOICE_DB": db, "PUBLIC_ORIGIN": origin})
	base, crash, stop := gate3ServeInvoice(t, ctx, bundle, home, entry, snapshot, gate3LifePort)
	expectInvoiceStartupFailure(t, ctx, bundle, home, entry, map[string]string{
		"INVOICE_DB":    db,
		"PUBLIC_ORIGIN": origin,
	}, gate3LifePort, "occupied-port")
	t.Log("startup refusal occupied-port: nonzero exit before serving")

	// Clean shutdown: SIGTERM drains and exits zero; the ledger
	// survives the restart; the pre-shutdown save replays its
	// recorded bytes and a fresh save commits.
	status, saved, _ := postInvoiceJSON(t, base, savePath, "tok-alice", origin, "application/json", saveBody("op-l1", 1))
	if status != 200 {
		crash()
		t.Fatalf("shutdown setup save: %d %s", status, saved)
	}
	shutdownStart := time.Now()
	stop()
	t.Logf("SIGTERM drained in %dms with a clean exit", time.Since(shutdownStart).Milliseconds())
	base, crash, stop = gate3ServeInvoice(t, ctx, bundle, home, entry, snapshot, gate3LifePort)
	status, replayed, _ := postInvoiceJSON(t, base, savePath, "tok-alice", origin, "application/json", saveBody("op-l1", 1))
	if status != 200 || replayed != saved {
		crash()
		t.Fatalf("post-restart replay: %d, want identical 200 bytes", status)
	}
	status, payload, _ := postInvoiceJSON(t, base, savePath, "tok-alice", origin, "application/json", saveBody("op-l2", 2))
	if status != 200 {
		crash()
		t.Fatalf("post-restart save: %d %s", status, payload)
	}
	requireInvoice(t, inspectInvoice(t, ctx, bundle, home, driver, db), "7", "3", "2", "Acme <em>&\" 'coop'\"")

	// In-flight drain: SIGTERM lands mid-body of a trickled save. The
	// request either completes or aborts, the server still exits
	// clean, and the identical retry converges to exactly one effect.
	trickled := saveBody("op-l3", 3)
	outcome := make(chan gate3RacePost, 1)
	go func() {
		request, err := http.NewRequest("POST", base+savePath, &gate4Trickle{body: []byte(trickled), chunk: 16, delay: 150 * time.Millisecond})
		if err != nil {
			outcome <- gate3RacePost{operation: "op-l3", status: -1, body: err.Error()}
			return
		}
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Origin", origin)
		request.AddCookie(&http.Cookie{Name: "session", Value: "tok-alice"})
		request.ContentLength = int64(len(trickled))
		client := &http.Client{Timeout: 30 * time.Second}
		response, err := client.Do(request)
		if err != nil {
			outcome <- gate3RacePost{operation: "op-l3", status: -1, body: err.Error()}
			return
		}
		defer response.Body.Close()
		payload, _ := io.ReadAll(response.Body)
		outcome <- gate3RacePost{operation: "op-l3", status: response.StatusCode, body: string(payload)}
	}()
	time.Sleep(400 * time.Millisecond)
	drainStart := time.Now()
	stop()
	t.Logf("SIGTERM with in-flight trickle drained in %dms with a clean exit", time.Since(drainStart).Milliseconds())
	select {
	case landed := <-outcome:
		t.Logf("in-flight outcome %d %.200s (either landing converges below)", landed.status, landed.body)
	case <-time.After(25 * time.Second):
		t.Fatal("in-flight trickle never returned")
	}
	pre := 0
	for _, row := range inspectInvoice(t, ctx, bundle, home, driver, db).Replay {
		if row.OperationID == "op-l3" {
			pre++
		}
	}
	if pre > 1 {
		t.Fatalf("in-flight trickle committed %d effects", pre)
	}
	base, _, stop = gate3ServeInvoice(t, ctx, bundle, home, entry, snapshot, gate3LifePort)
	defer stop()
	status, first, _ := postInvoiceJSON(t, base, savePath, "tok-alice", origin, "application/json", trickled)
	if status != 200 {
		t.Fatalf("drain converge save: %d %s", status, first)
	}
	status, second, _ := postInvoiceJSON(t, base, savePath, "tok-alice", origin, "application/json", trickled)
	if status != 200 || second != first {
		t.Fatalf("drain converge reread: %d, want identical 200 bytes", status)
	}
	value := gate3Case(t, "drain converge", first, "invoice_contract::grid_saved")
	acknowledged, _ := value["acknowledged"].(map[string]any)
	if acknowledged["revision"] != "4" {
		t.Fatalf("drain converge value %+v", value)
	}
	effects := 0
	for _, row := range inspectInvoice(t, ctx, bundle, home, driver, db).Replay {
		if row.OperationID == "op-l3" {
			effects++
		}
	}
	if effects != 1 {
		t.Fatalf("drain converged to %d op-l3 effects, want 1", effects)
	}

	// Buffered/lazy token revocation: the artifact's own
	// request-lifetime suite passes under the staged sidecar.
	sidecar := filepath.Join(bundle, "runtime/bun")
	clean, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(ctx, sidecar,
		"--no-env-file", "--no-macros", "--no-install",
		"--config="+filepath.Join(bundle, "tools/runtime/bunfig.toml"),
		"test", filepath.Join(bundle, "runtime/test", "request-lifetime.test.ts"))
	cmd.Dir = clean
	cmd.Env = []string{"HOME=" + clean, "XDG_CONFIG_HOME=" + clean, "PATH=/usr/local/bin:/usr/bin:/bin"}
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("request-lifetime suite: %v\n%s", err, output)
	}
	t.Logf("request-lifetime suite: %d pass under the staged sidecar", gate4PassCount(t, string(output)))
	t.Logf("gate3 lifecycle: %d can assertions, one pool, startup refusal, shutdown/drainage and revocation live with row evidence", assertions)
}
