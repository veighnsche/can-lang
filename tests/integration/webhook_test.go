// Live qualification for the T16 signed provider webhook slice. The
// test stages examples/webhook, asserts and builds it twice with
// identical IDs, seeds a disposable SQLite file, and serves the built
// entry over loopback HTTP. A Go provider fixture signs exact delivery
// bytes with HMAC-SHA-256; a stub downstream records carrier posts;
// a bun driver applies the schema and inspects rows. Crash legs kill
// the server with SIGKILL and restart it on the same file to prove
// one business effect per delivery under replay.
package integration

import (
	"bytes"
	"context"
	"crypto/hmac"
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
	ID           string `json:"id"`
	AtMs         string `json:"at_ms"`
}

type webhookStore struct {
	Ledger   []webhookRow `json:"ledger"`
	Outbox   []webhookRow `json:"outbox"`
	Attempts []webhookRow `json:"attempts"`
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
	client *http.Client
}

type pendingItem struct {
	DeliveryID   string `json:"delivery_id"`
	Event        string `json:"event"`
	Subscription string `json:"subscription"`
	Attempts     int    `json:"attempts"`
}

func (c *webhookCarrier) pending() []pendingItem {
	c.t.Helper()
	response, err := c.client.Get(c.base + "/outbox/pending")
	if err != nil {
		c.t.Fatal(err)
	}
	defer response.Body.Close()
	payload, _ := io.ReadAll(response.Body)
	if response.StatusCode != 200 {
		c.t.Fatalf("pending: %d %s", response.StatusCode, payload)
	}
	var page struct {
		Items []pendingItem `json:"items"`
	}
	if err := json.Unmarshal(payload, &page); err != nil {
		c.t.Fatalf("invalid pending page %v %s", err, payload)
	}
	if page.Items == nil {
		c.t.Fatal("pending page holds no items array")
	}
	return page.Items
}

func (c *webhookCarrier) ack(deliveryID string, settled bool) (int, string) {
	c.t.Helper()
	body, _ := json.Marshal(map[string]any{"delivery_id": deliveryID, "settled": settled})
	response, err := c.client.Post(c.base+"/outbox/ack", "application/json", bytes.NewReader(body))
	if err != nil {
		c.t.Fatal(err)
	}
	defer response.Body.Close()
	payload, _ := io.ReadAll(response.Body)
	return response.StatusCode, string(payload)
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
	if err := json.Unmarshal(setup, &setupReport); err != nil || setupReport.Tables != 3 {
		t.Fatalf("invalid webhook setup report %v %s", err, string(setup))
	}

	secret := "whsec-live-0123456789abcdef"
	snapshot := snapshotCredential(t, home, "WEBHOOK_SECRET", secret)
	entry := filepath.Join(firstDir, "entry.ts")
	base, crash, stop := serveWebhook(t, ctx, bundle, home, entry, snapshot, webhookPort, db)
	provider := &webhookProvider{t: t, base: base, secret: []byte(secret), client: &http.Client{Timeout: 10 * time.Second}}
	carrier := &webhookCarrier{t: t, base: base, client: &http.Client{Timeout: 10 * time.Second}}
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
	if len(store.Ledger) != 1 || len(store.Outbox) != 1 || len(store.Attempts) != 0 {
		t.Fatalf("accept store %+v", store)
	}
	if store.Outbox[0].State != "pending" || store.Outbox[0].Attempts != "0" {
		t.Fatalf("accept outbox %+v", store.Outbox[0])
	}
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

	items := carrier.pending()
	if len(items) != 1 || items[0].DeliveryID != "del-1" || items[0].Event != "invoice.paid" || items[0].Subscription != "sub-9" || items[0].Attempts != 0 {
		t.Fatalf("pending %+v", items)
	}
	stub.post(first)
	if status, payload := carrier.ack("del-1", true); status != 200 || payload != `{"status":"done","attempts":1}` {
		t.Fatalf("ack: %d %s", status, payload)
	}
	if items := carrier.pending(); len(items) != 0 {
		t.Fatalf("pending after ack %+v", items)
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

	second := `{"delivery_id":"del-2","event":"invoice.paid","subscription":"sub-9"}`
	check(200, second, "accept second")
	if status, payload := carrier.ack("del-2", false); status != 200 || payload != `{"status":"pending","attempts":1}` {
		t.Fatalf("failed ack: %d %s", status, payload)
	}
	items = carrier.pending()
	if len(items) != 1 || items[0].DeliveryID != "del-2" || items[0].Attempts != 1 {
		t.Fatalf("pending after failed ack %+v", items)
	}
	stub.post(second)
	if status, payload := carrier.ack("del-2", true); status != 200 || payload != `{"status":"done","attempts":2}` {
		t.Fatalf("retry ack: %d %s", status, payload)
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
	if len(store.Ledger) != 2 || len(store.Outbox) != 2 {
		t.Fatalf("crash store %+v", store)
	}
	requireDelivery(t, store, second, "invoice.paid", "sub-9")

	// Crash after external success but before outbox acknowledgement:
	// the carrier redelivers, the stub sees two receipts, and the
	// ledger still holds one effect with one recorded attempt.
	third := `{"delivery_id":"del-3","event":"invoice.paid","subscription":"sub-9"}`
	check(200, third, "accept third")
	items = carrier.pending()
	if len(items) != 1 || items[0].DeliveryID != "del-3" {
		t.Fatalf("pending third %+v", items)
	}
	stub.post(third)
	crash()
	base, crash, stop = serveWebhook(t, ctx, bundle, home, entry, snapshot, webhookPort, db)
	provider.base, carrier.base = base, base
	items = carrier.pending()
	if len(items) != 1 || items[0].DeliveryID != "del-3" {
		t.Fatalf("pending after ack-window crash %+v", items)
	}
	stub.post(third)
	if status, payload := carrier.ack("del-3", true); status != 200 || payload != `{"status":"done","attempts":1}` {
		t.Fatalf("ack after crash: %d %s", status, payload)
	}
	if got := stub.count("del-3"); got != 2 {
		t.Fatalf("stub saw del-3 %d times, want 2", got)
	}
	store = inspectWebhook(t, ctx, bundle, home, driver, db)
	deliveries := 0
	for _, row := range store.Ledger {
		if row.DeliveryID == "del-3" {
			deliveries++
		}
	}
	if deliveries != 1 {
		t.Fatalf("del-3 ledger rows %d, want 1", deliveries)
	}
	delivered := 0
	for _, row := range store.Attempts {
		if row.DeliveryID == "del-3" {
			delivered++
			if row.Outcome != "delivered" {
				t.Fatalf("del-3 attempt %+v", row)
			}
		}
	}
	if delivered != 1 {
		t.Fatalf("del-3 attempts %d, want 1", delivered)
	}

	// Crash during an in-flight delivery: the replay converges to one
	// ledger row and one outbox row however the race landed.
	fourth := `{"delivery_id":"del-4","event":"invoice.paid","subscription":"sub-9"}`
	inflight := make(chan string, 1)
	signature := provider.sign(fourth)
	go func() {
		status, payload, err := provider.deliverRaw(fourth, signature, true)
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
		status, payload := provider.send(fourth)
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
	if len(store.Ledger) != 4 {
		t.Fatalf("converged ledger %+v", store.Ledger)
	}
	outbox := 0
	for _, row := range store.Outbox {
		if row.DeliveryID == "del-4" {
			outbox++
		}
	}
	if outbox != 1 {
		t.Fatalf("del-4 outbox rows %d, want 1", outbox)
	}
	stub.post(fourth)
	if status, payload := carrier.ack("del-4", true); status != 200 {
		t.Fatalf("ack fourth: %d %s", status, payload)
	}

	stop()
	if got := stub.count("del-1"); got != 1 {
		t.Fatalf("stub saw del-1 %d times, want 1", got)
	}
	t.Logf("webhook slice: %d assertions (%d real-can), build %s, stub receipts del-1=%d del-2=%d del-3=%d del-4=%d",
		len(report.Assertions), real, firstID[:12], stub.count("del-1"), stub.count("del-2"), stub.count("del-3"), stub.count("del-4"))
}
