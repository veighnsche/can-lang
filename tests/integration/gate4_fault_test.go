// T20 Gate 4 server-driven SaaS fault matrix. This file owns the fault
// legs the Gate 3 suites do not cover, all against real HTTP and
// database surfaces on one artifact toolchain:
//
//   - invoice client disconnect: a fully written save whose client
//     vanishes before reading the response still commits exactly one
//     ledger effect under its owned lease;
//   - webhook slow inbound delivery: a trickled request body is still
//     accepted exactly once;
//   - termination deadline: SIGTERM drains owned work and exits clean;
//     the ledger survives the restart and replays stay duplicates;
//   - stream read-plus-close: the corrected cleanup funnel runs
//     end to end on native files, and a missing input fails without
//     hanging;
//   - shutdown suite: the artifact's own runtime shutdown/owner/
//     server/transport/coordination suites run under the artifact
//     sidecar (pending-loser settlement, deadline bounds,
//     read-versus-close precedence, operation-specific abort).
//
// The test records the host bound (runtime-check identity), the
// abortable-operation set, the owned settling work observed per leg,
// and explicit limitations in the test log as JSON plus prose.
//
// Toolchain selection: on linux/amd64 with CAN_BUN_ARCHIVE set, the
// test builds, releases, and installs the distribution and drives the
// installed launcher and sidecar, which is the Gate 4 qualification
// path (run inside the pinned Debian image, e.g. via
// `go test ./tests/integration/ -run TestGate4FaultMatrix`). On other
// hosts the same legs run against a staged bundle so the assertions
// stay verified during development. Without CAN_BUN_ARCHIVE the test
// skips by design.
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/distribution"
)

const (
	gate4InvoicePort = 18611
	gate4WebhookPort = 18612
)

// gate4Canlc runs the toolchain launcher with the network denied on
// darwin and a restricted environment on linux, where sandbox-exec
// does not exist.
func gate4Canlc(t *testing.T, ctx context.Context, canlc, home string, args ...string) (int, string, string) {
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

// gate4Assert asserts every Can root in a staged project and reports
// the assertion count and the real-can evidence count.
func gate4Assert(t *testing.T, ctx context.Context, canlc, home, root, name string) (assertions, real int) {
	t.Helper()
	status, out, diag := gate4Canlc(t, ctx, canlc, home, "assert", root)
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

// gate4Build builds a staged project and reports its build ID and
// output directory.
func gate4Build(t *testing.T, ctx context.Context, canlc, home, root string) (string, string) {
	t.Helper()
	status, out, diag := gate4Canlc(t, ctx, canlc, home, "build", root)
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

// gate4Toolchain resolves the artifact under test. On linux/amd64 it
// performs the full build, release, and install cycle and returns the
// installed selection; elsewhere it returns a staged bundle running
// the same legs for development verification.
func gate4Toolchain(t *testing.T, ctx context.Context, sourceRoot, archive string) (root, canlc, sidecar string, installed bool) {
	t.Helper()
	if runtime.GOOS == "linux" && runtime.GOARCH == "amd64" {
		bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "gate4-faults")
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
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "gate4-faults")
	if err != nil {
		t.Fatal(err)
	}
	return bundle, filepath.Join(bundle, "bin/canlc"), filepath.Join(bundle, "runtime/bun"), false
}

// gate4RuntimeIdentity records the runtime-check identity of the
// toolchain and asserts the pinned Bun revision.
func gate4RuntimeIdentity(t *testing.T, ctx context.Context, canlc, home string) map[string]string {
	t.Helper()
	status, out, diag := gate4Canlc(t, ctx, canlc, home, "runtime-check")
	if status != 0 || diag != "" {
		t.Fatalf("runtime-check: %d %s %s", status, out, diag)
	}
	var identity struct {
		Bun          string `json:"bun"`
		Revision     string `json:"revision"`
		Platform     string `json:"platform"`
		Architecture string `json:"architecture"`
		Executable   string `json:"executable"`
	}
	if err := json.Unmarshal([]byte(out), &identity); err != nil {
		t.Fatal(err)
	}
	if identity.Bun != "1.4.2" || identity.Revision != "744846f844374847c902b5e7fd59b4342a51ef99" {
		t.Fatalf("wrong runtime identity %+v", identity)
	}
	wantPlatform, wantArch := "darwin", "arm64"
	if runtime.GOOS == "linux" {
		wantPlatform, wantArch = "linux", "x64"
	}
	if identity.Platform != wantPlatform || identity.Architecture != wantArch {
		t.Fatalf("wrong runtime platform %+v", identity)
	}
	return map[string]string{
		"bun":          identity.Bun,
		"revision":     identity.Revision,
		"platform":     identity.Platform,
		"architecture": identity.Architecture,
	}
}

// gate4PostAndAbandon writes one complete form POST over raw TCP and
// closes the connection without reading the response, modelling a
// client that disconnects after the server already holds the full
// request bytes.
func gate4PostAndAbandon(t *testing.T, addr, path, body string) {
	t.Helper()
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	request := "POST " + path + " HTTP/1.1\r\nHost: " + addr +
		"\r\nContent-Type: application/x-www-form-urlencoded\r\nContent-Length: " + strconv.Itoa(len(body)) +
		"\r\nConnection: close\r\n\r\n" + body
	if _, err := io.WriteString(conn, request); err != nil {
		conn.Close()
		t.Fatal(err)
	}
	conn.Close()
}

// gate4Trickle delivers one body in small delayed chunks while still
// reading the response, modelling a slow inbound dependency.
type gate4Trickle struct {
	body  []byte
	at    int
	chunk int
	delay time.Duration
}

func (g *gate4Trickle) Read(p []byte) (int, error) {
	if g.at >= len(g.body) {
		return 0, io.EOF
	}
	time.Sleep(g.delay)
	n := copy(p, g.body[g.at:min(g.at+g.chunk, len(g.body))])
	g.at += n
	return n, nil
}

func gate4PassCount(t *testing.T, output string) int {
	t.Helper()
	if strings.Contains(output, "fail") && !strings.Contains(output, "0 fail") {
		t.Fatalf("suite output reports failures:\n%s", output)
	}
	match := regexp.MustCompile(`(\d+) pass`).FindStringSubmatch(output)
	if match == nil {
		t.Fatalf("suite output lacks a pass count:\n%s", output)
	}
	n, err := strconv.Atoi(match[1])
	if err != nil || n == 0 {
		t.Fatalf("suite passed nothing:\n%s", output)
	}
	return n
}

func TestGate4FaultMatrix(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for gate 4 fault execution")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	toolchain, canlc, sidecar, installed := gate4Toolchain(t, ctx, sourceRoot, archive)
	mode := "staged bundle"
	if installed {
		mode = "installed release"
	}

	host := gate4RuntimeIdentity(t, ctx, canlc, t.TempDir())
	t.Logf("gate4 host bound: %s on %s/%s bun %s rev %s", mode, host["platform"], host["architecture"], host["bun"], host["revision"][:12])

	report := map[string]any{
		"mode":     mode,
		"host":     host,
		"compiler": "canlc " + mode,
		"legs":     map[string]any{},
		"abortableOperations": []string{
			"transport deadlines with native abort and owner-signal cancellation",
			"stream cancel (terminal, records its reason)",
			"websocket cancel/stop prompts",
		},
		"limitations": []string{
			"deadline-expiry shutdown_failed(\"deadline\") is not forced on a live server here; it is covered by the artifact shutdown/owner suites below",
			"an empty first-completion race stays pending by design; the harness owns its observation deadline (coordination suite)",
			"the disconnect leg closes after a complete body; a mid-body abort is a 400/invalid_body rejection, never a partial commit",
		},
	}
	legs := report["legs"].(map[string]any)

	// Invoice client disconnect: the abandoned save still commits
	// exactly one replayed effect under its owned lease.
	root, home := stageApplication(t, ctx, toolchain, sourceRoot, "invoice")
	assertions, real := gate4Assert(t, ctx, canlc, home, root, "invoice")
	firstID, firstDir := gate4Build(t, ctx, canlc, home, root)
	secondID, _ := gate4Build(t, ctx, canlc, home, root)
	if firstID != secondID {
		t.Fatalf("invoice rebuild drifted: %s vs %s", firstID, secondID)
	}
	assertNoStrayEmit(t, root, firstDir)
	driver := filepath.Join(sourceRoot, "tests/integration/testdata/invoice/driver.ts")
	db := seedInvoiceDB(t, ctx, toolchain, home, driver, root)
	snapshot := snapshotCredential(t, home, "INVOICE_DB", db)
	base, stop := serveApplication(t, ctx, toolchain, home, filepath.Join(firstDir, "entry.ts"), snapshot, gate4InvoicePort, "/health")

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
	gate4PostAndAbandon(t, "127.0.0.1:"+strconv.Itoa(gate4InvoicePort), "/invoices/save-form", valid("op-gate4-disc", "1", "Acme Disc").Encode())
	deadline := time.Now().Add(20 * time.Second)
	for {
		store := inspectInvoice(t, ctx, toolchain, home, driver, db)
		done := false
		for _, row := range store.Replay {
			if row.OperationID == "op-gate4-disc" {
				done = true
			}
		}
		if done {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("abandoned invoice save never committed its replay row")
		}
		time.Sleep(250 * time.Millisecond)
	}
	store := inspectInvoice(t, ctx, toolchain, home, driver, db)
	requireInvoiceRevision(t, store, "inv-1", "2", "Acme Disc")
	if len(store.Replay) != 1 || store.Replay[0].OperationID != "op-gate4-disc" || len(store.Replay[0].Digest) != 64 {
		t.Fatalf("abandoned save recorded %+v", store.Replay)
	}
	if status, body, _ := invoicePostForm(t, base, "/invoices/save-form", valid("op-gate4-disc", "1", "Acme Disc")); status != 200 || !strings.Contains(body, "saved inv-1 revision 2") {
		t.Fatalf("disconnect replay: %d %s", status, body)
	}
	if status, body, _ := invoicePostForm(t, base, "/invoices/save-form", valid("op-gate4-disc", "1", "Acme Changed")); status != 409 || !strings.Contains(body, "stale inv-1 revision 2") {
		t.Fatalf("disconnect conflict: %d %s", status, body)
	}
	store = inspectInvoice(t, ctx, toolchain, home, driver, db)
	if len(store.Replay) != 1 {
		t.Fatalf("disconnect replay duplicated %+v", store.Replay)
	}
	invoiceStop := time.Now()
	stop()
	legs["invoice_disconnect"] = map[string]any{
		"assertions": assertions, "real_can": real, "build": firstID,
		"replay_rows": 1, "revision": "2",
		"stop_drain_ms": time.Since(invoiceStop).Milliseconds(),
		"settled":       "abandoned handler committed one effect; identical replay replays, changed content conflicts",
	}
	t.Logf("gate4 invoice disconnect: abandoned save committed revision 2 with one replay row; replay 200, conflict 409")

	// Webhook slow delivery plus termination deadline: a trickled
	// body is accepted once, SIGTERM drains owned work and exits
	// clean, and the ledger survives the restart.
	wroot, whome := stageApplication(t, ctx, toolchain, sourceRoot, "webhook")
	wassertions, wreal := gate4Assert(t, ctx, canlc, whome, wroot, "webhook")
	wfirstID, wfirstDir := gate4Build(t, ctx, canlc, whome, wroot)
	wsecondID, _ := gate4Build(t, ctx, canlc, whome, wroot)
	if wfirstID != wsecondID {
		t.Fatalf("webhook rebuild drifted: %s vs %s", wfirstID, wsecondID)
	}
	assertNoStrayEmit(t, wroot, wfirstDir)
	wdriver := filepath.Join(sourceRoot, "tests/integration/testdata/webhook/driver.ts")
	wdb := filepath.Join(whome, "hook.sqlite")
	setup := webhookDriver(t, ctx, toolchain, whome, wdriver, "setup", wdb, filepath.Join(wroot, "schema.sql"))
	var setupReport struct {
		Tables int `json:"tables"`
	}
	if err := json.Unmarshal(setup, &setupReport); err != nil || setupReport.Tables != 3 {
		t.Fatalf("invalid webhook setup report %v %s", err, string(setup))
	}
	secret := "whsec-gate4-fault-matrix"
	wsnapshot := snapshotCredential(t, whome, "WEBHOOK_SECRET", secret)
	wentry := filepath.Join(wfirstDir, "entry.ts")
	wbase, wcrash, wstop := serveWebhook(t, ctx, toolchain, whome, wentry, wsnapshot, gate4WebhookPort, wdb)
	provider := &webhookProvider{t: t, base: wbase, secret: []byte(secret), client: &http.Client{Timeout: 30 * time.Second}}
	slow := `{"delivery_id":"del-g4-slow","event":"invoice.paid","subscription":"sub-9"}`
	request, err := http.NewRequest("POST", wbase+"/webhooks/provider", &gate4Trickle{body: []byte(slow), chunk: 8, delay: 100 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("X-Provider-Signature", provider.sign(slow))
	request.ContentLength = int64(len(slow))
	response, err := provider.client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode != 200 || string(payload) != `{"status":"accepted"}` {
		t.Fatalf("slow delivery: %d %s", response.StatusCode, payload)
	}
	wstore := inspectWebhook(t, ctx, toolchain, whome, wdriver, wdb)
	requireDelivery(t, wstore, slow, "invoice.paid", "sub-9")
	if len(wstore.Ledger) != 1 || len(wstore.Outbox) != 1 {
		t.Fatalf("slow delivery store %+v", wstore)
	}
	webhookStop := time.Now()
	wstop()
	webhookDrainMs := time.Since(webhookStop).Milliseconds()
	wbase, wcrash, wstop = serveWebhook(t, ctx, toolchain, whome, wentry, wsnapshot, gate4WebhookPort, wdb)
	provider.base = wbase
	defer wcrash()
	if status, payload := provider.send(slow); status != 200 || payload != `{"status":"duplicate"}` {
		t.Fatalf("replay after SIGTERM: %d %s", status, payload)
	}
	after := `{"delivery_id":"del-g4-after","event":"invoice.paid","subscription":"sub-9"}`
	if status, payload := provider.send(after); status != 200 || payload != `{"status":"accepted"}` {
		t.Fatalf("delivery after restart: %d %s", status, payload)
	}
	wstore = inspectWebhook(t, ctx, toolchain, whome, wdriver, wdb)
	if len(wstore.Ledger) != 2 || len(wstore.Outbox) != 2 {
		t.Fatalf("restart store %+v", wstore)
	}
	requireDelivery(t, wstore, after, "invoice.paid", "sub-9")
	legs["webhook_slow_and_terminate"] = map[string]any{
		"assertions": wassertions, "real_can": wreal, "build": wfirstID,
		"ledger_rows": 2, "outbox_rows": 2,
		"sigterm_drain_ms": webhookDrainMs,
		"settled":          "trickled body accepted once; SIGTERM drained owned work; restart replays duplicate and accepts new work",
	}
	t.Logf("gate4 webhook: trickled delivery accepted once; SIGTERM drained in %dms; restart keeps 2 ledger rows", webhookDrainMs)

	// Stream read-plus-close: the corrected funnel counts native
	// lines end to end, and a missing input fails fast.
	sroot, shome := stageApplication(t, ctx, toolchain, sourceRoot, "stream")
	sassertions, sreal := gate4Assert(t, ctx, canlc, shome, sroot, "stream")
	lines := filepath.Join(shome, "lines.txt")
	if err := os.WriteFile(lines, []byte("a\nb\nc\n"), 0600); err != nil {
		t.Fatal(err)
	}
	status, out, diag := gate4Canlc(t, ctx, canlc, shome, "run", sroot, "--", lines)
	if status != 0 || out != "3" || diag != "" {
		t.Fatalf("stream run: %d stdout=%q stderr=%s", status, out, diag)
	}
	status, out, diag = gate4Canlc(t, ctx, canlc, shome, "run", sroot, "--", filepath.Join(shome, "missing.txt"))
	if status == 0 {
		t.Fatalf("missing stream input succeeded with %q", out)
	}
	if out == "" && diag == "" {
		t.Fatal("missing stream input reported nothing")
	}
	legs["stream_cleanup"] = map[string]any{
		"assertions": sassertions, "real_can": sreal,
		"counted": "3", "missing_input_exit": status,
		"settled": "funnel closes exactly once; read failure keeps precedence over close failure per the shutdown suite",
	}
	t.Logf("gate4 stream: counted 3 lines; missing input exits %d with a diagnostic", status)

	// Shutdown suite: the artifact's own runtime suites run under
	// the artifact sidecar and must pass with zero failures. The
	// server file generates a TLS chain through the host openssl
	// binary; hosts without it skip that file with a recorded
	// limitation instead of failing the product.
	suiteFiles := []string{"shutdown.test.ts", "owner.test.ts", "server.test.ts", "transport-owned.test.ts", "transport-late.test.ts", "coordination.test.ts"}
	suiteEnv := func(clean string) []string {
		return []string{"HOME=" + clean, "XDG_CONFIG_HOME=" + clean, "PATH=/usr/local/bin:/usr/bin:/bin"}
	}
	probe := exec.CommandContext(ctx, "sh", "-c", "command -v openssl")
	probe.Env = suiteEnv(t.TempDir())
	opensslOK := probe.Run() == nil
	if !opensslOK {
		suiteFiles = []string{"shutdown.test.ts", "owner.test.ts", "transport-owned.test.ts", "transport-late.test.ts", "coordination.test.ts"}
		report["limitations"] = append(report["limitations"].([]string), "server.test.ts skipped: host has no openssl for its TLS-chain leg; live SIGTERM drain still proven by the invoice/webhook legs")
		t.Log("gate4 shutdown suite: no host openssl, skipping server.test.ts")
	}
	suiteCounts := map[string]int{}
	for _, file := range suiteFiles {
		clean, err := filepath.EvalSymlinks(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		cmd := exec.CommandContext(ctx, sidecar,
			"--no-env-file", "--no-macros", "--no-install",
			"--config="+filepath.Join(toolchain, "tools/runtime/bunfig.toml"),
			"test", filepath.Join(toolchain, "runtime/test", file))
		cmd.Dir = clean
		cmd.Env = suiteEnv(clean)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s: %v\n%s", file, err, output)
		}
		suiteCounts[file] = gate4PassCount(t, string(output))
	}
	legs["shutdown_suite"] = map[string]any{
		"files":   suiteCounts,
		"failed":  0,
		"settled": "pending losers keep leases until they settle; deadlines bound only the waiter; disconnect never aborts an owned handler; abort exists only where an operation offers it",
	}
	t.Logf("gate4 shutdown suite: %v, 0 fail", suiteCounts)

	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("gate4 report:\n%s", encoded)
}
