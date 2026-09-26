// Live qualification for the F05 authenticated webhook/companion pair.
// The test stages examples/webhook, asserts and builds it, seeds a
// disposable SQLite file, and serves the built entry over loopback
// HTTP. A Go provider fixture signs exact delivery bytes with
// HMAC-SHA-256; an authenticated Go carrier speaks protocol v1
// (version header, exact-byte signature, timestamp plus single-use
// nonce in the signed body); a stub downstream records delivery posts;
// a bun driver applies the schema and inspects rows. Crash legs kill
// the server with SIGKILL and restart it on the same file to prove
// one business effect per delivery under replay, and lease expiry
// redelivers rows the crashed worker never acked.
package integration

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

const webhookPort = 18494

type webhookRow struct {
	DeliveryID   string `json:"delivery_id"`
	Digest       string `json:"digest"`
	Event        string `json:"event"`
	Subscription string `json:"subscription"`
	Outcome      string `json:"outcome"`
	ReceivedMs   string `json:"received_ms"`
	State        string `json:"state"`
	Attempts     string `json:"attempts"`
	UpdatedMs    string `json:"updated_ms"`
	WorkerID     string `json:"worker_id"`
	LeaseUntilMs string `json:"lease_until_ms"`
	Version      string `json:"version"`
	ID           string `json:"id"`
	AtMs         string `json:"at_ms"`
}

type webhookDeadRow struct {
	DeliveryID   string `json:"delivery_id"`
	Event        string `json:"event"`
	Subscription string `json:"subscription"`
	Reason       string `json:"reason"`
	Attempts     string `json:"attempts"`
	DeadMs       string `json:"dead_ms"`
}

type webhookNonceRow struct {
	Nonce  string `json:"nonce"`
	SeenMs string `json:"seen_ms"`
}

type webhookStore struct {
	Ledger   []webhookRow      `json:"ledger"`
	Outbox   []webhookRow      `json:"outbox"`
	Attempts []webhookRow      `json:"attempts"`
	Dead     []webhookDeadRow  `json:"dead"`
	Nonces   []webhookNonceRow `json:"nonces"`
}

type webhookProvider struct {
	t      *testing.T
	base   string
	secret []byte
	client *http.Client
}

func (p *webhookProvider) sign(body string) string {
	p.t.Helper()
	tag := hmac.New(sha256.New, p.secret)
	if _, err := tag.Write([]byte(body)); err != nil {
		p.t.Fatal(err)
	}
	return hex.EncodeToString(tag.Sum(nil))
}

func (p *webhookProvider) deliverRaw(body, signature string, withHeader bool) (int, string, error) {
	request, err := http.NewRequest("POST", p.base+"/webhooks/provider", strings.NewReader(body))
	if err != nil {
		return 0, "", err
	}
	if withHeader {
		request.Header.Set("X-Provider-Signature", signature)
	}
	response, err := p.client.Do(request)
	if err != nil {
		return 0, "", err
	}
	defer response.Body.Close()
	payload, _ := io.ReadAll(response.Body)
	return response.StatusCode, string(payload), nil
}

func (p *webhookProvider) deliver(body, signature string, withHeader bool) (int, string) {
	p.t.Helper()
	status, payload, err := p.deliverRaw(body, signature, withHeader)
	if err != nil {
		p.t.Fatal(err)
	}
	return status, payload
}

func (p *webhookProvider) send(body string) (int, string) {
	p.t.Helper()
	return p.deliver(body, p.sign(body), true)
}

type webhookCarrier struct {
	t      *testing.T
	base   string
	secret []byte
	client *http.Client
}

type claimRequest struct {
	WorkerID    string `json:"worker_id"`
	LeaseMs     int64  `json:"lease_ms"`
	TimestampMs int64  `json:"timestamp_ms"`
	Nonce       string `json:"nonce"`
}

type heartbeatRequest struct {
	WorkerID    string `json:"worker_id"`
	DeliveryID  string `json:"delivery_id"`
	Version     int    `json:"version"`
	LeaseMs     int64  `json:"lease_ms"`
	TimestampMs int64  `json:"timestamp_ms"`
	Nonce       string `json:"nonce"`
}

type ackRequest struct {
	DeliveryID  string `json:"delivery_id"`
	Settled     bool   `json:"settled"`
	TimestampMs int64  `json:"timestamp_ms"`
	Nonce       string `json:"nonce"`
}

type deadRequest struct {
	DeliveryID  string `json:"delivery_id"`
	WorkerID    string `json:"worker_id"`
	Reason      string `json:"reason"`
	TimestampMs int64  `json:"timestamp_ms"`
	Nonce       string `json:"nonce"`
}

type claimItem struct {
	DeliveryID   string `json:"delivery_id"`
	Event        string `json:"event"`
	Subscription string `json:"subscription"`
	Attempts     int    `json:"attempts"`
	LeaseUntilMs int64  `json:"lease_until_ms"`
	Version      int    `json:"version"`
}

type claimPage struct {
	Status string      `json:"status"`
	Items  []claimItem `json:"items"`
}

func (c *webhookCarrier) sign(body string) string {
	c.t.Helper()
	tag := hmac.New(sha256.New, c.secret)
	if _, err := tag.Write([]byte(body)); err != nil {
		c.t.Fatal(err)
	}
	return hex.EncodeToString(tag.Sum(nil))
}

func (c *webhookCarrier) nonce() string {
	c.t.Helper()
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		c.t.Fatal(err)
	}
	return hex.EncodeToString(raw)
}

func (c *webhookCarrier) now() int64 {
	return time.Now().UnixMilli()
}

// postRaw sends one carrier call with explicit headers for the
// negative legs. Empty protocol omits the version header; nil
// signature omits the signature header.
func (c *webhookCarrier) postRaw(path, body, protocol string, signature *string) (int, string) {
	c.t.Helper()
	request, err := http.NewRequest("POST", c.base+path, strings.NewReader(body))
	if err != nil {
		c.t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	if protocol != "" {
		request.Header.Set("X-Carrier-Protocol", protocol)
	}
	if signature != nil {
		request.Header.Set("X-Carrier-Signature", *signature)
	}
	response, err := c.client.Do(request)
	if err != nil {
		c.t.Fatal(err)
	}
	defer response.Body.Close()
	payload, _ := io.ReadAll(response.Body)
	return response.StatusCode, string(payload)
}

// call signs one carrier body and posts it with the v1 envelope. It
// returns the status, payload, and exact bytes sent so the replay leg
// can resend them verbatim.
func (c *webhookCarrier) call(path string, fields any) (int, string, string) {
	c.t.Helper()
	body, err := json.Marshal(fields)
	if err != nil {
		c.t.Fatal(err)
	}
	signature := c.sign(string(body))
	status, payload := c.postRaw(path, string(body), "1", &signature)
	return status, payload, string(body)
}

func (c *webhookCarrier) claim(workerID string, leaseMs int64) (int, claimPage) {
	c.t.Helper()
	status, payload, _ := c.call("/outbox/claim", claimRequest{
		WorkerID: workerID, LeaseMs: leaseMs, TimestampMs: c.now(), Nonce: c.nonce(),
	})
	var page claimPage
	if err := json.Unmarshal([]byte(payload), &page); err != nil {
		c.t.Fatalf("invalid claim page %d %v %s", status, err, payload)
	}
	if page.Items == nil {
		c.t.Fatal("claim page holds no items array")
	}
	return status, page
}

func (c *webhookCarrier) heartbeat(workerID, deliveryID string, version int, leaseMs int64) (int, string) {
	c.t.Helper()
	status, payload, _ := c.call("/outbox/heartbeat", heartbeatRequest{
		WorkerID: workerID, DeliveryID: deliveryID, Version: version,
		LeaseMs: leaseMs, TimestampMs: c.now(), Nonce: c.nonce(),
	})
	return status, payload
}

func (c *webhookCarrier) ack(deliveryID string, settled bool) (int, string) {
	c.t.Helper()
	status, payload, _ := c.call("/outbox/ack", ackRequest{
		DeliveryID: deliveryID, Settled: settled, TimestampMs: c.now(), Nonce: c.nonce(),
	})
	return status, payload
}

func (c *webhookCarrier) deadLetter(deliveryID, workerID, reason string) (int, string) {
	c.t.Helper()
	status, payload, _ := c.call("/outbox/dead_letter", deadRequest{
		DeliveryID: deliveryID, WorkerID: workerID, Reason: reason,
		TimestampMs: c.now(), Nonce: c.nonce(),
	})
	return status, payload
}

type downstreamStub struct {
	t        *testing.T
	mu       sync.Mutex
	receipts []string
	server   *httptest.Server
}

func newDownstreamStub(t *testing.T) *downstreamStub {
	t.Helper()
	stub := &downstreamStub{t: t}
	stub.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload, _ := io.ReadAll(r.Body)
		var envelope struct {
			DeliveryID string `json:"delivery_id"`
		}
		if err := json.Unmarshal(payload, &envelope); err != nil || envelope.DeliveryID == "" {
			http.Error(w, "bad delivery", 400)
			return
		}
		stub.mu.Lock()
		stub.receipts = append(stub.receipts, envelope.DeliveryID)
		stub.mu.Unlock()
		w.WriteHeader(200)
	}))
	t.Cleanup(stub.server.Close)
	return stub
}

func (s *downstreamStub) post(body string) {
	s.t.Helper()
	response, err := http.Post(s.server.URL, "application/json", strings.NewReader(body))
	if err != nil {
		s.t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		s.t.Fatalf("stub refused %d", response.StatusCode)
	}
}

func (s *downstreamStub) count(deliveryID string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	total := 0
	for _, receipt := range s.receipts {
		if receipt == deliveryID {
			total++
		}
	}
	return total
}

// serveWebhook runs a staged entry.ts with port and database arguments
// plus one fd-3 credential snapshot, waits for /health, and returns
// the base URL with a SIGKILL crash handle and a clean SIGTERM stopper.
func serveWebhook(t *testing.T, ctx context.Context, bundle, home, entry, snapshot string, port int, db string) (string, func(), func()) {
	t.Helper()
	requirePortFree(t, port)
	file, err := os.Open(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	cmd := exec.CommandContext(ctx, filepath.Join(bundle, "runtime/bun"), entry, strconv.Itoa(port), db)
	cmd.Dir = home
	cmd.Env = []string{"PATH=/nonexistent", "HOME=" + home}
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
		if status, _, _ := httpGet(base + "/health"); status == 200 {
			break
		}
		if time.Now().After(deadline) {
			cmd.Process.Kill()
			t.Fatalf("webhook never ready on %s: stdout=%q stderr=%q", base, out.String(), diag.String())
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
			t.Fatal("crashed server would not exit")
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
				t.Fatalf("unclean shutdown: %v stdout=%q stderr=%q", err, out.String(), diag.String())
			}
		case <-time.After(20 * time.Second):
			cmd.Process.Kill()
			t.Fatalf("shutdown hung: stdout=%q stderr=%q", out.String(), diag.String())
		}
	}
	return base, crash, stop
}

func webhookDriver(t *testing.T, ctx context.Context, bundle, home, driver string, args ...string) []byte {
	t.Helper()
	cmd := exec.CommandContext(ctx, filepath.Join(bundle, "runtime/bun"), append([]string{driver}, args...)...)
	cmd.Dir = home
	cmd.Env = []string{"PATH=/nonexistent", "HOME=" + home}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("driver %v: %v %s", args, err, string(out))
	}
	return out
}

func inspectWebhook(t *testing.T, ctx context.Context, bundle, home, driver, db string) webhookStore {
	t.Helper()
	out := webhookDriver(t, ctx, bundle, home, driver, "inspect", db)
	var store webhookStore
	if err := json.Unmarshal(out, &store); err != nil {
		t.Fatalf("invalid inspect report %v %s", err, string(out))
	}
	return store
}

func digestOf(body string) string {
	sum := sha256.Sum256([]byte(body))
	return hex.EncodeToString(sum[:])
}

func requireDelivery(t *testing.T, store webhookStore, body, event, subscription string) {
	t.Helper()
	var envelope struct {
		DeliveryID string `json:"delivery_id"`
	}
	if err := json.Unmarshal([]byte(body), &envelope); err != nil {
		t.Fatal(err)
	}
	if len(store.Ledger) == 0 {
		t.Fatalf("no ledger row for %s", envelope.DeliveryID)
	}
	found := false
	for _, row := range store.Ledger {
		if row.DeliveryID != envelope.DeliveryID {
			continue
		}
		found = true
		if row.Digest != digestOf(body) || row.Event != event || row.Subscription != subscription || row.Outcome != "accepted" {
			t.Fatalf("bad ledger row %+v for %q", row, body)
		}
	}
	if !found {
		t.Fatalf("no ledger row for %s in %+v", envelope.DeliveryID, store.Ledger)
	}
}

func requireOutbox(t *testing.T, store webhookStore, deliveryID, state, attempts, workerID, version string) {
	t.Helper()
	for _, row := range store.Outbox {
		if row.DeliveryID != deliveryID {
			continue
		}
		if row.State != state || row.Attempts != attempts || row.WorkerID != workerID || row.Version != version {
			t.Fatalf("bad outbox row %+v", row)
		}
		return
	}
	t.Fatalf("no outbox row for %s in %+v", deliveryID, store.Outbox)
}

func TestWebhookSliceLive(t *testing.T) {
	t.Parallel()
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged webhook execution")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	bundle, err := harnessBundle(t, ctx, archive)
	if err != nil {
		t.Fatal(err)
	}
	root, home := stageApplication(t, ctx, bundle, sourceRoot, "webhook")
	status, out, diag := canlcOffline(t, ctx, bundle, home, "assert", root)
	if status != 0 || diag != "" {
		t.Fatalf("webhook assert: %d %s %s", status, out, diag)
	}
	var report struct {
		Passed     bool `json:"passed"`
		Assertions []struct {
			Evidence []string `json:"evidence"`
		} `json:"assertions"`
	}
	if err := json.Unmarshal([]byte(out), &report); err != nil || !report.Passed || len(report.Assertions) == 0 {
		t.Fatalf("invalid webhook assert report %v %s", err, out)
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
		t.Fatal("webhook asserts nothing real")
	}
	// One build: pristine webhook determinism is proven once in
	// TestStdlibMaintained; this suite owns live delivery.
	firstID, firstDir := applicationBuild(t, ctx, bundle, home, root)
	assertNoStrayEmit(t, root, firstDir)

	driver := filepath.Join(sourceRoot, "tests/integration/testdata/webhook/driver.ts")
	db := filepath.Join(home, "hook.sqlite")
	setup := webhookDriver(t, ctx, bundle, home, driver, "setup", db, filepath.Join(root, "schema.sql"))
	var setupReport struct {
		Tables int `json:"tables"`
	}
	if err := json.Unmarshal(setup, &setupReport); err != nil || setupReport.Tables != 5 {
		t.Fatalf("invalid webhook setup report %v %s", err, string(setup))
	}

	secret := "whsec-live-0123456789abcdef"
	carrierSecret := "carrier-live-0123456789abcdef"
	snapshot := snapshotMap(t, home, "snapshot-webhook", map[string]string{
		"WEBHOOK_SECRET": secret, "CARRIER_SECRET": carrierSecret,
	})
	entry := filepath.Join(firstDir, "entry.ts")
	base, crash, stop := serveWebhook(t, ctx, bundle, home, entry, snapshot, webhookPort, db)
	provider := &webhookProvider{t: t, base: base, secret: []byte(secret), client: &http.Client{Timeout: 10 * time.Second}}
	carrier := &webhookCarrier{t: t, base: base, secret: []byte(carrierSecret), client: &http.Client{Timeout: 10 * time.Second}}
	stub := newDownstreamStub(t)

	check := func(wantStatus int, body, note string) string {
		t.Helper()
		status, payload := provider.send(body)
		if status != wantStatus {
			t.Fatalf("%s: %d %s, want %d", note, status, payload, wantStatus)
		}
		return payload
	}

	first := `{"delivery_id":"del-1","event":"invoice.paid","subscription":"sub-9"}`
	if payload := check(200, first, "accept"); payload != `{"status":"accepted"}` {
		t.Fatalf("accept body %s", payload)
	}
	store := inspectWebhook(t, ctx, bundle, home, driver, db)
	requireDelivery(t, store, first, "invoice.paid", "sub-9")
	if len(store.Ledger) != 1 || len(store.Outbox) != 1 || len(store.Attempts) != 0 || len(store.Dead) != 0 {
		t.Fatalf("accept store %+v", store)
	}
	requireOutbox(t, store, "del-1", "pending", "0", "", "0")
	if payload := check(200, first, "replay"); payload != `{"status":"duplicate"}` {
		t.Fatalf("replay body %s", payload)
	}
	changed := `{"delivery_id":"del-1","event":"invoice.refunded","subscription":"sub-9"}`
	if payload := check(409, changed, "conflict"); payload != `{"status":"conflict"}` {
		t.Fatalf("conflict body %s", payload)
	}
	if status, payload := provider.deliver(first, strings.Repeat("00", 32), true); status != 401 || payload != `{"status":"unauthorized"}` {
		t.Fatalf("bad signature: %d %s", status, payload)
	}
	if status, payload := provider.deliver(first, "", false); status != 401 || payload != `{"status":"unauthorized"}` {
		t.Fatalf("missing signature: %d %s", status, payload)
	}
	if payload := check(400, "not json", "malformed"); payload != `{"status":"invalid_body"}` {
		t.Fatalf("malformed body %s", payload)
	}
	if payload := check(400, `{"delivery_id":"","event":"e","subscription":"s"}`, "empty id"); payload != `{"status":"invalid_body"}` {
		t.Fatalf("empty id body %s", payload)
	}
	oversize := `{"delivery_id":"big","event":"e","subscription":"s","pad":"` + strings.Repeat("p", 9000) + `"}`
	if payload := check(413, oversize, "oversize"); payload != `{"status":"body_too_large"}` {
		t.Fatalf("oversize body %s", payload)
	}
	store = inspectWebhook(t, ctx, bundle, home, driver, db)
	if len(store.Ledger) != 1 || len(store.Outbox) != 1 || len(store.Attempts) != 0 {
		t.Fatalf("rejections changed the store %+v", store)
	}

	// Carrier auth matrix: every unauthenticated, replayed, stale,
	// or malformed carrier request rejects before touching the store.
	claimBody := func(fields claimRequest) string {
		t.Helper()
		body, err := json.Marshal(fields)
		if err != nil {
			t.Fatal(err)
		}
		return string(body)
	}
	validClaim := claimBody(claimRequest{WorkerID: "worker-A", LeaseMs: 60000, TimestampMs: carrier.now(), Nonce: carrier.nonce()})
	if status, payload := carrier.postRaw("/outbox/claim", validClaim, "", nil); status != 400 || payload != `{"status":"unsupported_protocol","items":[]}` {
		t.Fatalf("missing protocol: %d %s", status, payload)
	}
	signature := carrier.sign(validClaim)
	if status, payload := carrier.postRaw("/outbox/claim", validClaim, "2", &signature); status != 400 || payload != `{"status":"unsupported_protocol","items":[]}` {
		t.Fatalf("wrong protocol: %d %s", status, payload)
	}
	if status, payload := carrier.postRaw("/outbox/claim", validClaim, "1", nil); status != 401 || payload != `{"status":"unauthorized","items":[]}` {
		t.Fatalf("missing carrier signature: %d %s", status, payload)
	}
	wrong := strings.Repeat("00", 32)
	if status, payload := carrier.postRaw("/outbox/claim", validClaim, "1", &wrong); status != 401 || payload != `{"status":"unauthorized","items":[]}` {
		t.Fatalf("wrong carrier signature: %d %s", status, payload)
	}
	staleBody := claimBody(claimRequest{WorkerID: "worker-A", LeaseMs: 60000, TimestampMs: carrier.now() - 600000, Nonce: carrier.nonce()})
	staleSig := carrier.sign(staleBody)
	if status, payload := carrier.postRaw("/outbox/claim", staleBody, "1", &staleSig); status != 401 || payload != `{"status":"stale","items":[]}` {
		t.Fatalf("stale carrier timestamp: %d %s", status, payload)
	}
	futureBody := claimBody(claimRequest{WorkerID: "worker-A", LeaseMs: 60000, TimestampMs: carrier.now() + 600000, Nonce: carrier.nonce()})
	futureSig := carrier.sign(futureBody)
	if status, payload := carrier.postRaw("/outbox/claim", futureBody, "1", &futureSig); status != 401 || payload != `{"status":"stale","items":[]}` {
		t.Fatalf("future carrier timestamp: %d %s", status, payload)
	}
	badShape := claimBody(claimRequest{WorkerID: "worker-A", LeaseMs: 999, TimestampMs: carrier.now(), Nonce: carrier.nonce()})
	badShapeSig := carrier.sign(badShape)
	if status, payload := carrier.postRaw("/outbox/claim", badShape, "1", &badShapeSig); status != 400 || payload != `{"status":"invalid_shape","items":[]}` {
		t.Fatalf("bad claim shape: %d %s", status, payload)
	}
	huge := claimBody(claimRequest{WorkerID: strings.Repeat("w", 2000), LeaseMs: 60000, TimestampMs: carrier.now(), Nonce: carrier.nonce()})
	hugeSig := carrier.sign(huge)
	if status, payload := carrier.postRaw("/outbox/claim", huge, "1", &hugeSig); status != 413 || payload != `{"status":"body_too_large","items":[]}` {
		t.Fatalf("oversize carrier body: %d %s", status, payload)
	}
	// Unknown ids roll back without consuming the nonce, so an
	// identical retry answers 404 again instead of 401: only
	// committed outcomes pin their nonce. The replay rejection below
	// rides a committed claim instead.
	replayFields := heartbeatRequest{WorkerID: "worker-A", DeliveryID: "del-9", Version: 0, LeaseMs: 60000, TimestampMs: carrier.now(), Nonce: carrier.nonce()}
	replayBody, err := json.Marshal(replayFields)
	if err != nil {
		t.Fatal(err)
	}
	replaySig := carrier.sign(string(replayBody))
	if status, payload := carrier.postRaw("/outbox/heartbeat", string(replayBody), "1", &replaySig); status != 404 || payload != `{"status":"unknown","lease_until_ms":0,"version":0,"attempts":0}` {
		t.Fatalf("heartbeat unknown: %d %s", status, payload)
	}
	if status, payload := carrier.postRaw("/outbox/heartbeat", string(replayBody), "1", &replaySig); status != 404 || payload != `{"status":"unknown","lease_until_ms":0,"version":0,"attempts":0}` {
		t.Fatalf("heartbeat unknown retry: %d %s", status, payload)
	}
	store = inspectWebhook(t, ctx, bundle, home, driver, db)
	requireOutbox(t, store, "del-1", "pending", "0", "", "0")

	// Claim, deliver, ack: worker-A takes del-1, the stub sees one
	// post, and the ack retires the row. Resending the committed
	// claim bytes verbatim answers 401 replay: the nonce pinned.
	claimFields := claimRequest{WorkerID: "worker-A", LeaseMs: 60000, TimestampMs: carrier.now(), Nonce: carrier.nonce()}
	firstClaim, err := json.Marshal(claimFields)
	if err != nil {
		t.Fatal(err)
	}
	firstClaimSig := carrier.sign(string(firstClaim))
	status, payload := carrier.postRaw("/outbox/claim", string(firstClaim), "1", &firstClaimSig)
	var page claimPage
	if err := json.Unmarshal([]byte(payload), &page); err != nil {
		t.Fatalf("invalid claim page %d %v %s", status, err, payload)
	}
	if status != 200 || page.Status != "claimed" || len(page.Items) != 1 {
		t.Fatalf("claim: %d %+v", status, page)
	}
	if status, payload := carrier.postRaw("/outbox/claim", string(firstClaim), "1", &firstClaimSig); status != 401 || payload != `{"status":"replay","items":[]}` {
		t.Fatalf("claim replay: %d %s", status, payload)
	}
	got := page.Items[0]
	if got.DeliveryID != "del-1" || got.Event != "invoice.paid" || got.Subscription != "sub-9" || got.Attempts != 0 || got.Version != 1 {
		t.Fatalf("claim item %+v", got)
	}
	if got.LeaseUntilMs <= carrier.now() || got.LeaseUntilMs > carrier.now()+60000 {
		t.Fatalf("claim lease %d not within the ask", got.LeaseUntilMs)
	}
	store = inspectWebhook(t, ctx, bundle, home, driver, db)
	requireOutbox(t, store, "del-1", "pending", "0", "worker-A", "1")
	if status, page := carrier.claim("worker-B", 60000); status != 200 || page.Status != "empty" || len(page.Items) != 0 {
		t.Fatalf("rival claim while held: %d %+v", status, page)
	}
	stub.post(first)
	if status, payload := carrier.ack("del-1", true); status != 200 || payload != `{"status":"done","attempts":1}` {
		t.Fatalf("ack: %d %s", status, payload)
	}
	if status, page := carrier.claim("worker-A", 60000); status != 200 || page.Status != "empty" {
		t.Fatalf("claim after ack: %d %+v", status, page)
	}
	if status, payload := carrier.ack("del-1", true); status != 200 || payload != `{"status":"done","attempts":1}` {
		t.Fatalf("duplicate ack: %d %s", status, payload)
	}
	store = inspectWebhook(t, ctx, bundle, home, driver, db)
	if len(store.Attempts) != 1 || store.Attempts[0].Outcome != "delivered" || store.Attempts[0].DeliveryID != "del-1" {
		t.Fatalf("duplicate ack recorded %+v", store.Attempts)
	}
	if status, payload := carrier.ack("del-9", true); status != 404 || payload != `{"status":"unknown","attempts":0}` {
		t.Fatalf("unknown ack: %d %s", status, payload)
	}

	// Failed ack plus lease expiry: the row stays leased to worker-A
	// until expiry, then worker-B reclaims it at version 2.
	second := `{"delivery_id":"del-2","event":"invoice.paid","subscription":"sub-9"}`
	check(200, second, "accept second")
	if status, page := carrier.claim("worker-A", 1000); status != 200 || page.Status != "claimed" {
		t.Fatalf("claim second: %d %+v", status, page)
	}
	if status, payload := carrier.ack("del-2", false); status != 200 || payload != `{"status":"pending","attempts":1}` {
		t.Fatalf("failed ack: %d %s", status, payload)
	}
	if status, page := carrier.claim("worker-B", 60000); status != 200 || page.Status != "empty" {
		t.Fatalf("reclaim while leased: %d %+v", status, page)
	}
	time.Sleep(1200 * time.Millisecond)
	status, page = carrier.claim("worker-B", 60000)
	if status != 200 || page.Status != "claimed" || len(page.Items) != 1 || page.Items[0].DeliveryID != "del-2" || page.Items[0].Attempts != 1 || page.Items[0].Version != 2 {
		t.Fatalf("reclaim after expiry: %d %+v", status, page)
	}
	stub.post(second)
	if status, payload := carrier.ack("del-2", true); status != 200 || payload != `{"status":"done","attempts":2}` {
		t.Fatalf("retry ack: %d %s", status, payload)
	}

	// Heartbeat: the holder extends at the current version; stale and
	// foreign holders lose; terminal rows replay their state.
	third := `{"delivery_id":"del-3","event":"invoice.paid","subscription":"sub-9"}`
	check(200, third, "accept third")
	if status, page := carrier.claim("worker-A", 60000); status != 200 || page.Status != "claimed" || page.Items[0].Version != 1 {
		t.Fatalf("claim third: %d %+v", status, page)
	}
	if status, payload := carrier.heartbeat("worker-A", "del-3", 1, 60000); status != 200 || !strings.HasPrefix(payload, `{"status":"extended","lease_until_ms":`) || !strings.HasSuffix(payload, `,"version":2,"attempts":0}`) {
		t.Fatalf("heartbeat extend: %d %s", status, payload)
	}
	if status, payload := carrier.heartbeat("worker-A", "del-3", 1, 60000); status != 409 || payload != `{"status":"lease_lost","lease_until_ms":0,"version":0,"attempts":0}` {
		t.Fatalf("stale heartbeat: %d %s", status, payload)
	}
	if status, payload := carrier.heartbeat("worker-B", "del-3", 2, 60000); status != 409 {
		t.Fatalf("foreign heartbeat: %d %s", status, payload)
	}
	stub.post(third)
	if status, payload := carrier.ack("del-3", true); status != 200 || payload != `{"status":"done","attempts":1}` {
		t.Fatalf("ack third: %d %s", status, payload)
	}
	if status, payload := carrier.heartbeat("worker-A", "del-3", 2, 60000); status != 200 || !strings.HasPrefix(payload, `{"status":"done",`) {
		t.Fatalf("heartbeat after done: %d %s", status, payload)
	}

	// Poison and dead letter: five failures mark the row, the holder
	// buries it, and every replay converges on dead.
	fourth := `{"delivery_id":"del-4","event":"invoice.paid","subscription":"sub-9"}`
	check(200, fourth, "accept fourth")
	if status, page := carrier.claim("worker-A", 60000); status != 200 || page.Status != "claimed" {
		t.Fatalf("claim fourth: %d %+v", status, page)
	}
	for attempt := 1; attempt <= 5; attempt++ {
		want := `{"status":"pending","attempts":` + strconv.Itoa(attempt) + `}`
		if status, payload := carrier.ack("del-4", false); status != 200 || payload != want {
			t.Fatalf("poison ack %d: %d %s", attempt, status, payload)
		}
	}
	if status, payload := carrier.deadLetter("del-4", "worker-A", "poison"); status != 200 || payload != `{"status":"dead","attempts":5}` {
		t.Fatalf("dead letter: %d %s", status, payload)
	}
	if status, payload := carrier.ack("del-4", false); status != 200 || payload != `{"status":"dead","attempts":5}` {
		t.Fatalf("ack after dead: %d %s", status, payload)
	}
	if status, payload := carrier.deadLetter("del-4", "worker-A", "poison"); status != 200 || payload != `{"status":"dead","attempts":5}` {
		t.Fatalf("dead replay: %d %s", status, payload)
	}
	if status, payload := carrier.deadLetter("del-1", "worker-A", "poison"); status != 409 || payload != `{"status":"already_done","attempts":1}` {
		t.Fatalf("dead on done: %d %s", status, payload)
	}
	if status, payload := carrier.deadLetter("del-9", "worker-A", "poison"); status != 404 || payload != `{"status":"unknown","attempts":0}` {
		t.Fatalf("dead unknown: %d %s", status, payload)
	}
	store = inspectWebhook(t, ctx, bundle, home, driver, db)
	requireOutbox(t, store, "del-4", "dead", "5", "worker-A", "1")
	if len(store.Dead) != 1 || store.Dead[0].DeliveryID != "del-4" || store.Dead[0].Reason != "poison" || store.Dead[0].Attempts != "5" {
		t.Fatalf("dead table %+v", store.Dead)
	}
	if len(store.Nonces) == 0 {
		t.Fatal("nonce table holds no consumed nonces")
	}

	// Crash between deliveries: the committed ledger and outbox
	// survive the kill and the replay stays a duplicate.
	crash()
	base, crash, stop = serveWebhook(t, ctx, bundle, home, entry, snapshot, webhookPort, db)
	provider.base, carrier.base = base, base
	if payload := check(200, second, "replay after crash"); payload != `{"status":"duplicate"}` {
		t.Fatalf("replay after crash body %s", payload)
	}
	store = inspectWebhook(t, ctx, bundle, home, driver, db)
	if len(store.Ledger) != 4 || len(store.Outbox) != 4 {
		t.Fatalf("crash store %+v", store)
	}
	requireDelivery(t, store, second, "invoice.paid", "sub-9")

	// Crash after external success but before outbox acknowledgement:
	// the short lease expires, the carrier reclaims, the stub sees
	// two receipts, and the ledger still holds one effect with one
	// recorded attempt.
	fifth := `{"delivery_id":"del-5","event":"invoice.paid","subscription":"sub-9"}`
	check(200, fifth, "accept fifth")
	if status, page := carrier.claim("worker-A", 1000); status != 200 || page.Status != "claimed" {
		t.Fatalf("claim fifth: %d %+v", status, page)
	}
	stub.post(fifth)
	crash()
	base, crash, stop = serveWebhook(t, ctx, bundle, home, entry, snapshot, webhookPort, db)
	provider.base, carrier.base = base, base
	if status, page := carrier.claim("worker-B", 60000); status != 200 || page.Status != "empty" {
		t.Fatalf("reclaim before expiry: %d %+v", status, page)
	}
	time.Sleep(1200 * time.Millisecond)
	if status, page := carrier.claim("worker-B", 60000); status != 200 || page.Status != "claimed" || page.Items[0].DeliveryID != "del-5" || page.Items[0].Version != 2 {
		t.Fatalf("redeliver after expiry: %d %+v", status, page)
	}
	stub.post(fifth)
	if status, payload := carrier.ack("del-5", true); status != 200 || payload != `{"status":"done","attempts":1}` {
		t.Fatalf("ack after crash: %d %s", status, payload)
	}
	if got := stub.count("del-5"); got != 2 {
		t.Fatalf("stub saw del-5 %d times, want 2", got)
	}
	store = inspectWebhook(t, ctx, bundle, home, driver, db)
	deliveries := 0
	for _, row := range store.Ledger {
		if row.DeliveryID == "del-5" {
			deliveries++
		}
	}
	if deliveries != 1 {
		t.Fatalf("del-5 ledger rows %d, want 1", deliveries)
	}
	delivered := 0
	for _, row := range store.Attempts {
		if row.DeliveryID == "del-5" {
			delivered++
			if row.Outcome != "delivered" {
				t.Fatalf("del-5 attempt %+v", row)
			}
		}
	}
	if delivered != 1 {
		t.Fatalf("del-5 attempts %d, want 1", delivered)
	}

	// Crash during an in-flight delivery: the replay converges to one
	// ledger row and one outbox row however the race landed.
	sixth := `{"delivery_id":"del-6","event":"invoice.paid","subscription":"sub-9"}`
	inflight := make(chan string, 1)
	signature = provider.sign(sixth)
	go func() {
		status, payload, err := provider.deliverRaw(sixth, signature, true)
		if err != nil {
			inflight <- "transport: " + err.Error()
			return
		}
		inflight <- "response: " + strconv.Itoa(status) + " " + payload
	}()
	time.Sleep(100 * time.Millisecond)
	crash()
	base, _, stop = serveWebhook(t, ctx, bundle, home, entry, snapshot, webhookPort, db)
	provider.base, carrier.base = base, base
	select {
	case outcome := <-inflight:
		t.Logf("in-flight outcome %s (either landing converges below)", outcome)
	case <-time.After(15 * time.Second):
		t.Fatal("in-flight delivery never returned")
	}
	seen := ""
	for i := 0; i < 3; i++ {
		status, payload := provider.send(sixth)
		if status != 200 {
			t.Fatalf("converge replay %d: %d %s", i, status, payload)
		}
		seen = payload
		if payload == `{"status":"duplicate"}` {
			break
		}
		if payload != `{"status":"accepted"}` {
			t.Fatalf("converge replay %d body %s", i, payload)
		}
	}
	if seen != `{"status":"duplicate"}` {
		t.Fatal("in-flight delivery never converged to duplicate")
	}
	store = inspectWebhook(t, ctx, bundle, home, driver, db)
	if len(store.Ledger) != 6 {
		t.Fatalf("converged ledger %+v", store.Ledger)
	}
	outbox := 0
	for _, row := range store.Outbox {
		if row.DeliveryID == "del-6" {
			outbox++
		}
	}
	if outbox != 1 {
		t.Fatalf("del-6 outbox rows %d, want 1", outbox)
	}
	if status, page := carrier.claim("worker-A", 60000); status != 200 || page.Status != "claimed" || page.Items[0].DeliveryID != "del-6" {
		t.Fatalf("claim sixth: %d %+v", status, page)
	}
	stub.post(sixth)
	if status, payload := carrier.ack("del-6", true); status != 200 {
		t.Fatalf("ack sixth: %d %s", status, payload)
	}

	// Batch pair: two simultaneously pending rows both claim in turn
	// instead of faulting the candidate read.
	seventh := `{"delivery_id":"del-A","event":"invoice.paid","subscription":"sub-9"}`
	eighth := `{"delivery_id":"del-B","event":"invoice.paid","subscription":"sub-9"}`
	check(200, seventh, "accept seventh")
	check(200, eighth, "accept eighth")
	if status, page := carrier.claim("worker-A", 60000); status != 200 || page.Status != "claimed" || page.Items[0].DeliveryID != "del-A" {
		t.Fatalf("claim seventh: %d %+v", status, page)
	}
	if status, page := carrier.claim("worker-A", 60000); status != 200 || page.Status != "claimed" || page.Items[0].DeliveryID != "del-B" {
		t.Fatalf("claim eighth: %d %+v", status, page)
	}
	if status, page := carrier.claim("worker-A", 60000); status != 200 || page.Status != "empty" {
		t.Fatalf("claim drained: %d %+v", status, page)
	}
	stub.post(seventh)
	stub.post(eighth)
	if status, payload := carrier.ack("del-A", true); status != 200 {
		t.Fatalf("ack seventh: %d %s", status, payload)
	}
	if status, payload := carrier.ack("del-B", true); status != 200 {
		t.Fatalf("ack eighth: %d %s", status, payload)
	}
	store = inspectWebhook(t, ctx, bundle, home, driver, db)
	if len(store.Ledger) != 8 || len(store.Outbox) != 8 {
		t.Fatalf("batch store %+v", store)
	}

	stop()
	if got := stub.count("del-1"); got != 1 {
		t.Fatalf("stub saw del-1 %d times, want 1", got)
	}
	t.Logf("webhook pair: %d assertions (%d real-can), build %s, stub receipts del-1=%d del-2=%d del-3=%d del-5=%d del-6=%d",
		len(report.Assertions), real, firstID[:12], stub.count("del-1"), stub.count("del-2"), stub.count("del-3"), stub.count("del-5"), stub.count("del-6"))
}
