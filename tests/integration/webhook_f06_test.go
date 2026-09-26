// Live F06 qualification: two real companion processes against Can.
//
// TestCompanionPairLiveF06 stages examples/webhook, asserts and builds
// it, seeds one fresh SQLite file for the run, and serves the built
// entry over loopback HTTP. It then runs the actual companion entry
// (examples/webhook/companion/main.ts, F05's worker plus F06's tuning
// flags) as two worker processes against the live Can side — never a
// Go reimplementation of the protocol — and qualifies the full W6.3
// companion scope: two-worker leases/claims, bounded concurrency,
// backoff/idle sleeps, poison/dead-letter, crash/unacked redelivery,
// idempotent ack, carrier auth, and destination/credential enforcement.
// The closing W4.3 leg drains a bounded 200-row batch through both
// workers and measures wall time, throughput, and peak companion RSS
// while every batch step mirrors the C-A step/failure/occurrence shape
// from A07's handoff.
//
// At-least-once honesty: the crash leg proves redelivery can post a
// row twice downstream while the ledger still holds one effect. No
// leg claims exactly-once delivery, and no worker logic lives in Can.
package integration

import (
	"bufio"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

const f06Port = 18495

// f06Failure mirrors the companion's finite C-A step/failure shape
// (PROTOCOL.md section 5): identity is one of thirteen codes and
// occurrence carries the status or reason, never a secret or URL.
type f06Failure struct {
	Identity   string `json:"identity"`
	Occurrence string `json:"occurrence"`
}

type f06Step struct {
	Step       int        `json:"step"`
	DeliveryID string     `json:"deliveryId"`
	Outcome    string     `json:"outcome"`
	Attempts   int        `json:"attempts"`
	Version    int        `json:"version"`
	Failure    f06Failure `json:"failure"`
}

type f06Report struct {
	Protocol   string      `json:"protocol"`
	WorkerID   string      `json:"workerId"`
	Batch      int         `json:"batch"`
	Steps      []f06Step   `json:"steps"`
	BatchError *f06Failure `json:"batchError"`
}

var f06Outcomes = map[string]bool{
	"delivered": true, "failed": true, "dead": true, "error": true,
}

var f06Identities = map[string]bool{
	"ok": true, "downstream-status": true, "downstream-transport": true,
	"downstream-timeout": true, "credential-missing": true,
	"redirect-denied": true, "redirect-hops": true, "poison": true,
	"destination-denied": true, "no-destination": true, "lease-lost": true,
	"protocol-error": true, "store-unavailable": true,
}

// f06Receipt records one downstream arrival at handler entry, before
// any programmed delay, so the crash leg can count sends the killed
// companion never acked.
type f06Receipt struct {
	ID   string
	Auth string
}

// f06Stub is a programmable downstream: behavior switches on the
// delivery-id prefix, arrivals record at entry, and in-flight posts
// track the live concurrency peak.
type f06Stub struct {
	mu          sync.Mutex
	receipts    []f06Receipt
	inFlight    atomic.Int32
	maxInflight atomic.Int32
	gate        chan struct{}
	server      *httptest.Server
}

func newF06Stub(t *testing.T) *f06Stub {
	t.Helper()
	stub := &f06Stub{gate: make(chan struct{})}
	stub.server = httptest.NewServer(http.HandlerFunc(stub.serve))
	t.Cleanup(stub.server.Close)
	return stub
}

func (s *f06Stub) serve(w http.ResponseWriter, r *http.Request) {
	payload, _ := io.ReadAll(r.Body)
	var envelope struct {
		DeliveryID string `json:"delivery_id"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil || envelope.DeliveryID == "" {
		http.Error(w, "bad delivery", 400)
		return
	}
	current := s.inFlight.Add(1)
	s.mu.Lock()
	for {
		peak := s.maxInflight.Load()
		if current <= peak || s.maxInflight.CompareAndSwap(peak, current) {
			break
		}
	}
	s.receipts = append(s.receipts, f06Receipt{ID: envelope.DeliveryID, Auth: r.Header.Get("Authorization")})
	nth := 0
	for _, receipt := range s.receipts {
		if receipt.ID == envelope.DeliveryID {
			nth++
		}
	}
	s.mu.Unlock()
	defer s.inFlight.Add(-1)
	id := envelope.DeliveryID
	if strings.HasPrefix(id, "f06-poison-") || strings.HasPrefix(id, "f06-idle-") {
		http.Error(w, "downstream failed", 500)
		return
	}
	switch {
	case strings.HasPrefix(id, "f06-flaky-") && nth == 1:
		time.Sleep(3 * time.Second)
	case strings.HasPrefix(id, "f06-gate-"):
		<-s.gate
	case strings.HasPrefix(id, "f06-conc4-"):
		time.Sleep(100 * time.Millisecond)
	}
	w.WriteHeader(200)
}

func (s *f06Stub) count(deliveryID string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	total := 0
	for _, receipt := range s.receipts {
		if receipt.ID == deliveryID {
			total++
		}
	}
	return total
}

func (s *f06Stub) total() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.receipts)
}

func (s *f06Stub) authFor(deliveryID string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := len(s.receipts) - 1; i >= 0; i-- {
		if s.receipts[i].ID == deliveryID {
			return s.receipts[i].Auth
		}
	}
	return ""
}

func (s *f06Stub) peak() int {
	return int(s.maxInflight.Load())
}

func (s *f06Stub) resetPeak() {
	s.maxInflight.Store(0)
}

func (s *f06Stub) releaseGate() {
	select {
	case <-s.gate:
	default:
		close(s.gate)
	}
}

// f06Companion runs the real companion entry from the source tree
// (the staged root holds only the example, while the companion
// imports runtime modules by relative path, so it cannot run from
// the stage) with the pinned bundle bun.
type f06Companion struct {
	bun    string
	main   string
	base   string
	config string
	policy string
	home   string
}

type f06Run struct {
	Code   int
	Stdout string
	Stderr string
}

type f06Proc struct {
	cmd    *exec.Cmd
	stdout bytes.Buffer
	stderr bytes.Buffer
	done   chan error
}

func (c *f06Companion) start(ctx context.Context, workerID, secret string, extraEnv, args []string) *f06Proc {
	argv := append([]string{
		c.main, "--base", c.base, "--worker", workerID,
		"--config", c.config, "--policy", c.policy,
	}, args...)
	proc := &f06Proc{done: make(chan error, 1)}
	proc.cmd = exec.CommandContext(ctx, c.bun, argv...)
	proc.cmd.Dir = c.home
	proc.cmd.Env = append([]string{"PATH=/nonexistent", "HOME=" + c.home, "CARRIER_SECRET=" + secret}, extraEnv...)
	proc.cmd.Stdout = &proc.stdout
	proc.cmd.Stderr = &proc.stderr
	if err := proc.cmd.Start(); err != nil {
		proc.done <- err
		return proc
	}
	go func() { proc.done <- proc.cmd.Wait() }()
	return proc
}

func (p *f06Proc) wait() f06Run {
	run, _ := p.waitTimeout(0)
	return run
}

// waitTimeout bounds a companion wait: a hung carrier call has no
// client-side timeout by design (the Can side owns every verdict),
// so the harness kills and reports instead of burning the suite.
func (p *f06Proc) waitTimeout(limit time.Duration) (f06Run, bool) {
	var err error
	if limit <= 0 {
		err = <-p.done
	} else {
		select {
		case err = <-p.done:
		case <-time.After(limit):
			p.kill()
			<-p.done
			run := f06Run{Code: -1, Stdout: p.stdout.String(), Stderr: p.stderr.String()}
			return run, false
		}
	}
	run := f06Run{Stdout: p.stdout.String(), Stderr: p.stderr.String()}
	if err == nil {
		return run, true
	}
	var status *exec.ExitError
	if errors.As(err, &status) {
		run.Code = status.ExitCode()
		return run, true
	}
	run.Code = -1
	run.Stderr += " wait: " + err.Error()
	return run, true
}

func (p *f06Proc) pid() int {
	if p.cmd.Process == nil {
		return -1
	}
	return p.cmd.Process.Pid
}

func (p *f06Proc) kill() {
	if p.cmd.Process != nil {
		p.cmd.Process.Kill()
	}
}

func (p *f06Proc) term() {
	if p.cmd.Process != nil {
		p.cmd.Process.Signal(syscall.SIGTERM)
	}
}

// f06TimedLine is one looping-worker batch report with its arrival
// time, so the backoff leg can measure inter-batch sleeps.
type f06TimedLine struct {
	At     time.Time
	Line   string
	Report f06Report
}

// runLoop starts a looping companion whose stdout lines stream into
// timestamped reports. Call stop (SIGTERM + wait) to end it; the
// worker drains between batches and exits 0.
func (c *f06Companion) runLoop(ctx context.Context, t *testing.T, workerID, secret string, extraEnv, args []string) (stop func() f06Run, lines func() []f06TimedLine) {
	t.Helper()
	argv := append([]string{
		c.main, "--base", c.base, "--worker", workerID,
		"--config", c.config, "--policy", c.policy,
	}, args...)
	cmd := exec.CommandContext(ctx, c.bun, argv...)
	cmd.Dir = c.home
	cmd.Env = append([]string{"PATH=/nonexistent", "HOME=" + c.home, "CARRIER_SECRET=" + secret}, extraEnv...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var timed []f06TimedLine
	scanned := make(chan struct{})
	go func() {
		defer close(scanned)
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.TrimSpace(line) == "" {
				continue
			}
			var report f06Report
			if err := json.Unmarshal([]byte(line), &report); err != nil {
				continue
			}
			mu.Lock()
			timed = append(timed, f06TimedLine{At: time.Now(), Line: line, Report: report})
			mu.Unlock()
		}
	}()
	stop = func() f06Run {
		t.Helper()
		if cmd.Process != nil {
			cmd.Process.Signal(syscall.SIGTERM)
		}
		run := f06Run{Stderr: stderr.String()}
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		select {
		case err := <-done:
			if err != nil {
				var status *exec.ExitError
				if errors.As(err, &status) {
					run.Code = status.ExitCode()
				} else {
					run.Code = -1
				}
			}
		case <-time.After(30 * time.Second):
			cmd.Process.Kill()
			t.Fatal("looping companion would not drain on SIGTERM")
		}
		<-scanned
		return run
	}
	lines = func() []f06TimedLine {
		mu.Lock()
		defer mu.Unlock()
		return append([]f06TimedLine(nil), timed...)
	}
	return stop, lines
}

// runOnce runs one --once batch and parses its single report line.
// Batches legitimately take seconds (the flaky leg sleeps 3s on its
// first send); past two minutes the batch is wedged, so fail with
// the partial output instead of hanging the suite.
func (c *f06Companion) runOnce(ctx context.Context, t *testing.T, workerID, secret string, extraEnv, tuning []string) (f06Run, f06Report) {
	t.Helper()
	proc := c.start(ctx, workerID, secret, extraEnv, append([]string{"--once"}, tuning...))
	run, ok := proc.waitTimeout(2 * time.Minute)
	if !ok {
		t.Fatalf("companion --once wedged past 2m: stdout=%q stderr=%q", run.Stdout, run.Stderr)
	}
	if run.Code != 0 {
		t.Fatalf("companion --once exit %d: stdout=%q stderr=%q", run.Code, run.Stdout, run.Stderr)
	}
	reports := parseF06Reports(t, run.Stdout)
	if len(reports) != 1 {
		t.Fatalf("companion --once printed %d reports, want 1: %q", len(reports), run.Stdout)
	}
	return run, reports[0]
}

func parseF06Reports(t *testing.T, stdout string) []f06Report {
	t.Helper()
	var reports []f06Report
	for _, line := range strings.Split(stdout, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var report f06Report
		if err := json.Unmarshal([]byte(line), &report); err != nil {
			t.Fatalf("invalid batch report %v: %q", err, line)
		}
		reports = append(reports, report)
	}
	return reports
}

// requireF06Shape pins the C-A mirror on one report: protocol v1,
// the owning worker, 1-based contiguous claim-order steps, finite
// outcomes/identities, and non-empty occurrences on failures.
func requireF06Shape(t *testing.T, report f06Report, workerID string) {
	t.Helper()
	if report.Protocol != "1" {
		t.Fatalf("report protocol %q, want 1", report.Protocol)
	}
	if report.WorkerID != workerID {
		t.Fatalf("report worker %q, want %q", report.WorkerID, workerID)
	}
	if report.Steps == nil {
		t.Fatal("report holds no steps array")
	}
	for index, step := range report.Steps {
		if step.Step != index+1 {
			t.Fatalf("step index %d at position %d, want contiguous 1-based", step.Step, index)
		}
		if step.DeliveryID == "" {
			t.Fatal("step holds no delivery id")
		}
		if !f06Outcomes[step.Outcome] {
			t.Fatalf("step %s outcome %q outside the finite set", step.DeliveryID, step.Outcome)
		}
		if !f06Identities[step.Failure.Identity] {
			t.Fatalf("step %s identity %q outside the finite set", step.DeliveryID, step.Failure.Identity)
		}
		if step.Outcome == "delivered" && step.Failure.Identity != "ok" {
			t.Fatalf("delivered step %s carries identity %q", step.DeliveryID, step.Failure.Identity)
		}
		if step.Outcome != "delivered" && step.Failure.Occurrence == "" {
			t.Fatalf("non-delivered step %s holds no occurrence", step.DeliveryID)
		}
		if step.Version < 1 {
			t.Fatalf("step %s version %d, want >= 1", step.DeliveryID, step.Version)
		}
	}
	if report.BatchError != nil && !f06Identities[report.BatchError.Identity] {
		t.Fatalf("batch error identity %q outside the finite set", report.BatchError.Identity)
	}
}

// requireF06Clean asserts a report line carries no secret or URL:
// steps name the delivery, the finite outcome, and the reason code,
// and operators map subscriptions to URLs from their own config.
func requireF06Clean(t *testing.T, line string, secrets ...string) {
	t.Helper()
	for _, secret := range secrets {
		if secret != "" && strings.Contains(line, secret) {
			t.Fatal("batch report leaks a secret")
		}
	}
	if strings.Contains(line, "http://") || strings.Contains(line, "https://") {
		t.Fatalf("batch report leaks a URL: %q", line)
	}
}

// f06Seed posts n provider deliveries and returns their ids.
func f06Seed(t *testing.T, provider *webhookProvider, prefix, subscription string, n int) []string {
	t.Helper()
	ids := make([]string, 0, n)
	for i := 0; i < n; i++ {
		id := prefix + strconv.Itoa(i)
		body := `{"delivery_id":"` + id + `","event":"invoice.paid","subscription":"` + subscription + `"}`
		if status, payload := provider.send(body); status != 200 || payload != `{"status":"accepted"}` {
			t.Fatalf("seed %s: %d %s", id, status, payload)
		}
		ids = append(ids, id)
	}
	return ids
}

func f06Attempts(store webhookStore, deliveryID, outcome string) int {
	total := 0
	for _, row := range store.Attempts {
		if row.DeliveryID == deliveryID && row.Outcome == outcome {
			total++
		}
	}
	return total
}

func f06Dead(store webhookStore, deliveryID string) (webhookDeadRow, bool) {
	for _, row := range store.Dead {
		if row.DeliveryID == deliveryID {
			return row, true
		}
	}
	return webhookDeadRow{}, false
}

func f06LedgerCount(store webhookStore, deliveryID string) int {
	total := 0
	for _, row := range store.Ledger {
		if row.DeliveryID == deliveryID {
			total++
		}
	}
	return total
}

// sampleF06RSS sums the resident sets of live pids in KiB via ps,
// skipping exits and parse failures: short-lived companions may be
// gone between samples.
func sampleF06RSS(pids ...int) int {
	live := make([]string, 0, len(pids))
	for _, pid := range pids {
		if pid > 0 {
			live = append(live, strconv.Itoa(pid))
		}
	}
	if len(live) == 0 {
		return 0
	}
	out, err := exec.Command("ps", "-o", "rss=", "-p", strings.Join(live, ",")).Output()
	if err != nil {
		return 0
	}
	total := 0
	for _, field := range strings.Fields(string(out)) {
		value, err := strconv.Atoi(field)
		if err != nil {
			return 0
		}
		total += value
	}
	return total
}

func TestCompanionPairLiveF06(t *testing.T) {
	t.Parallel()
	acquireHeavy(t)
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged companion execution")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()
	bundle, err := harnessBundle(t, ctx, archive)
	if err != nil {
		t.Fatal(err)
	}
	root, home := stageApplication(t, ctx, bundle, sourceRoot, "webhook")
	if status, out, diag := canlcOffline(t, ctx, bundle, home, "assert", root); status != 0 || diag != "" {
		t.Fatalf("webhook assert: %d %s %s", status, out, diag)
	}
	firstID, firstDir := applicationBuild(t, ctx, bundle, home, root)
	driver := filepath.Join(sourceRoot, "tests/integration/testdata/webhook/driver.ts")
	// Distinct database per run: a fresh SQLite file in a fresh temp
	// home. The webhook Can side is SQLite-only by F05 design, so the
	// H02 PG/MySQL minting path does not apply; isolation here is one
	// disposable file per test run, never a shared table.
	db := filepath.Join(home, "hook-f06.sqlite")
	setup := webhookDriver(t, ctx, bundle, home, driver, "setup", db, filepath.Join(root, "schema.sql"))
	var setupReport struct {
		Tables int `json:"tables"`
	}
	if err := json.Unmarshal(setup, &setupReport); err != nil || setupReport.Tables != 5 {
		t.Fatalf("invalid webhook setup report %v %s", err, string(setup))
	}

	secret := "whsec-f06-live-0123456789abcdef"
	carrierSecret := "carrier-f06-live-0123456789abcdef"
	hookToken := "f06-hook-token-0123456789"
	snapshot := snapshotMap(t, home, "snapshot-f06", map[string]string{
		"WEBHOOK_SECRET": secret, "CARRIER_SECRET": carrierSecret,
	})
	entry := filepath.Join(firstDir, "entry.ts")
	base, _, stop := serveWebhook(t, ctx, bundle, home, entry, snapshot, f06Port, db)
	defer stop()
	provider := &webhookProvider{t: t, base: base, secret: []byte(secret), client: &http.Client{Timeout: 10 * time.Second}}

	stub := newF06Stub(t)
	stubURL, err := url.Parse(stub.server.URL)
	if err != nil {
		t.Fatal(err)
	}
	stubPort, err := strconv.Atoi(stubURL.Port())
	if err != nil {
		t.Fatal(err)
	}
	deliveries := map[string]any{
		"deliveries": map[string]string{
			"sub-ok":     stub.server.URL + "/ok",
			"sub-cred":   stub.server.URL + "/cred",
			"sub-poison": stub.server.URL + "/poison",
			"sub-denied": "http://example.com:9/hook",
		},
	}
	configPath := filepath.Join(home, "deliveries.json")
	rawDeliveries, err := json.Marshal(deliveries)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, rawDeliveries, 0600); err != nil {
		t.Fatal(err)
	}
	// First-match rules: the /cred binding first, then the plain
	// loopback rule. 127.0.0.1 faces both the loopback and the
	// private scopes, so both allow; redirects stay denied.
	policy := map[string]any{
		"version": "2026-09-26.f06",
		"rules": []map[string]any{
			{"scheme": "http", "host": "127.0.0.1", "port": stubPort, "pathPrefix": "/cred", "credential": "F06_HOOK_TOKEN"},
			{"scheme": "http", "host": "127.0.0.1", "port": stubPort},
		},
		"redirect": "deny", "maxRedirectHops": 0,
		"loopback": "allow", "privateNetworks": "allow",
	}
	policyPath := filepath.Join(home, "policy.json")
	rawPolicy, err := json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(policyPath, rawPolicy, 0600); err != nil {
		t.Fatal(err)
	}
	companion := &f06Companion{
		bun:  filepath.Join(bundle, "runtime/bun"),
		main: filepath.Join(sourceRoot, "examples/webhook/companion/main.ts"),
		base: base, config: configPath, policy: policyPath, home: home,
	}

	// Leg A — carrier auth plus CLI honesty: a wrong-secret companion
	// fails fast with exit 1 and touches nothing; a bad tuning flag
	// exits 2 before any I/O; the correct secret drains the row.
	f06Seed(t, provider, "f06-auth-", "sub-ok", 1)
	wrong := companion.start(ctx, "worker-A", "wrong-secret", nil, []string{"--once"})
	if run := wrong.wait(); run.Code != 1 || !strings.Contains(run.Stderr, "unauthorized") {
		t.Fatalf("wrong-secret companion: exit %d stderr=%q", run.Code, run.Stderr)
	}
	store := inspectWebhook(t, ctx, bundle, home, driver, db)
	requireOutbox(t, store, "f06-auth-0", "pending", "0", "", "0")
	badFlag := companion.start(ctx, "worker-A", carrierSecret, nil, []string{"--once", "--lease-ms", "5"})
	if run := badFlag.wait(); run.Code != 2 || !strings.Contains(run.Stderr, "usage:") {
		t.Fatalf("bad-flag companion: exit %d stderr=%q", run.Code, run.Stderr)
	}
	run, report := companion.runOnce(ctx, t, "worker-A", carrierSecret, nil, nil)
	requireF06Shape(t, report, "worker-A")
	requireF06Clean(t, run.Stdout, carrierSecret, hookToken)
	if len(report.Steps) != 1 || report.Steps[0].DeliveryID != "f06-auth-0" || report.Steps[0].Outcome != "delivered" {
		t.Fatalf("auth smoke steps %+v", report.Steps)
	}
	t.Logf("F06 leg A (auth): wrong secret exits 1 untouched, bad flag exits 2, smoke row delivered")

	// Leg B — two-worker leases/claims: both companions drain capped
	// concurrent batches over 48 rows. The 24-row caps force the split:
	// neither worker can take the whole outbox, so both must hold
	// disjoint leases whatever the scheduling.
	f06Seed(t, provider, "f06-pair-", "sub-ok", 48)
	procA := companion.start(ctx, "worker-A", carrierSecret, nil, []string{"--once", "--max-batch", "24"})
	procB := companion.start(ctx, "worker-B", carrierSecret, nil, []string{"--once", "--max-batch", "24"})
	runA, okA := procA.waitTimeout(2 * time.Minute)
	runB, okB := procB.waitTimeout(2 * time.Minute)
	if !okA || !okB {
		t.Fatalf("pair runs wedged: A ok=%v B ok=%v", okA, okB)
	}
	if runA.Code != 0 || runB.Code != 0 {
		t.Fatalf("pair runs: A=%d %q B=%d %q", runA.Code, runA.Stderr, runB.Code, runB.Stderr)
	}
	reportA := parseF06Reports(t, runA.Stdout)
	reportB := parseF06Reports(t, runB.Stdout)
	if len(reportA) != 1 || len(reportB) != 1 {
		t.Fatalf("pair reports %d/%d, want 1/1", len(reportA), len(reportB))
	}
	requireF06Shape(t, reportA[0], "worker-A")
	requireF06Shape(t, reportB[0], "worker-B")
	requireF06Clean(t, runA.Stdout, carrierSecret, hookToken)
	requireF06Clean(t, runB.Stdout, carrierSecret, hookToken)
	if len(reportA[0].Steps) == 0 || len(reportB[0].Steps) == 0 {
		t.Fatalf("pair split %d/%d, want both workers holding rows", len(reportA[0].Steps), len(reportB[0].Steps))
	}
	seen := map[string]string{}
	checkPair := func(report f06Report, worker string) {
		t.Helper()
		for _, step := range report.Steps {
			if step.Outcome != "delivered" {
				t.Fatalf("pair step %s outcome %q", step.DeliveryID, step.Outcome)
			}
			if owner, dup := seen[step.DeliveryID]; dup {
				t.Fatalf("pair step %s claimed by %s and %s", step.DeliveryID, owner, worker)
			}
			seen[step.DeliveryID] = worker
		}
	}
	checkPair(reportA[0], "worker-A")
	checkPair(reportB[0], "worker-B")
	if len(seen) != 48 {
		t.Fatalf("pair union %d rows, want 48", len(seen))
	}
	store = inspectWebhook(t, ctx, bundle, home, driver, db)
	for i := 0; i < 48; i++ {
		id := "f06-pair-" + strconv.Itoa(i)
		if stub.count(id) != 1 {
			t.Fatalf("pair stub saw %s %d times, want 1", id, stub.count(id))
		}
		if f06LedgerCount(store, id) != 1 || f06Attempts(store, id, "delivered") != 1 {
			t.Fatalf("pair store for %s holds %+v", id, store)
		}
	}
	// Idempotent ack tail: a drained outbox reports empty and records
	// nothing new; duplicate acks converge without new attempts.
	attemptsBefore := len(store.Attempts)
	_, drained := companion.runOnce(ctx, t, "worker-A", carrierSecret, nil, nil)
	if len(drained.Steps) != 0 || drained.BatchError != nil {
		t.Fatalf("drained batch %+v", drained)
	}
	store = inspectWebhook(t, ctx, bundle, home, driver, db)
	if len(store.Attempts) != attemptsBefore {
		t.Fatalf("drained run recorded %d attempts, want %d", len(store.Attempts), attemptsBefore)
	}
	t.Logf("F06 leg B (pair): A=%d B=%d disjoint delivered, stub 1 each, redrain empty",
		len(reportA[0].Steps), len(reportB[0].Steps))

	// Leg C — bounded concurrency: --concurrency 1 never overlaps two
	// sends; --concurrency 4 over delayed sends overlaps within the
	// bound. The peak is the live proof; no Can-side knob exists.
	stub.resetPeak()
	f06Seed(t, provider, "f06-c1-", "sub-ok", 8)
	_, serial := companion.runOnce(ctx, t, "worker-A", carrierSecret, nil, []string{"--concurrency", "1"})
	requireF06Shape(t, serial, "worker-A")
	if len(serial.Steps) != 8 {
		t.Fatalf("serial batch %d steps, want 8", len(serial.Steps))
	}
	for _, step := range serial.Steps {
		if step.Outcome != "delivered" {
			t.Fatalf("serial step %s outcome %q", step.DeliveryID, step.Outcome)
		}
	}
	if peak := stub.peak(); peak != 1 {
		t.Fatalf("serial peak in-flight %d, want 1", peak)
	}
	stub.resetPeak()
	f06Seed(t, provider, "f06-conc4-", "sub-ok", 12)
	_, parallel := companion.runOnce(ctx, t, "worker-A", carrierSecret, nil, []string{"--concurrency", "4"})
	requireF06Shape(t, parallel, "worker-A")
	if len(parallel.Steps) != 12 {
		t.Fatalf("parallel batch %d steps, want 12", len(parallel.Steps))
	}
	for _, step := range parallel.Steps {
		if step.Outcome != "delivered" {
			t.Fatalf("parallel step %s outcome %q", step.DeliveryID, step.Outcome)
		}
	}
	if peak := stub.peak(); peak < 2 || peak > 4 {
		t.Fatalf("parallel peak in-flight %d, want 2..4", peak)
	}
	t.Logf("F06 leg C (concurrency): serial peak 1, parallel peak %d within the bound of 4", stub.peak())

	// Leg D — poison/dead-letter: three rows against an always-500
	// downstream ride five failed attempts and then dead-letter
	// without a sixth downstream touch. Short leases pace the rounds;
	// the loop runs to dead with a cap, never a fixed round count.
	f06Seed(t, provider, "f06-poison-", "sub-poison", 3)
	poisonTuning := []string{"--lease-ms", "1000"}
	deadIDs := map[string]f06Step{}
	for round := 0; round < 12 && len(deadIDs) < 3; round++ {
		if round > 0 {
			time.Sleep(1300 * time.Millisecond)
		}
		run, report := companion.runOnce(ctx, t, "worker-A", carrierSecret, nil, poisonTuning)
		requireF06Shape(t, report, "worker-A")
		requireF06Clean(t, run.Stdout, carrierSecret, hookToken)
		summary := make([]string, 0, len(report.Steps))
		for _, step := range report.Steps {
			summary = append(summary, step.DeliveryID+"="+step.Outcome+"/"+step.Failure.Identity+"/"+step.Failure.Occurrence+"/a"+strconv.Itoa(step.Attempts)+"/v"+strconv.Itoa(step.Version))
		}
		t.Logf("F06 leg D round %d: steps=%d [%s] batchError=%v", round, len(report.Steps), strings.Join(summary, " "), report.BatchError)
		for _, step := range report.Steps {
			switch step.Outcome {
			case "failed":
				if step.Failure.Identity != "downstream-status" || step.Failure.Occurrence != "500" {
					t.Fatalf("poison step %+v, want downstream-status/500", step)
				}
			case "dead":
				if step.Failure.Identity != "poison" || step.Failure.Occurrence != "5" {
					t.Fatalf("poison step %+v, want poison/5", step)
				}
				deadIDs[step.DeliveryID] = step
			default:
				t.Fatalf("poison step %+v, want failed or dead", step)
			}
		}
	}
	if len(deadIDs) != 3 {
		t.Fatal("poison rows never reached dead within 12 rounds")
	}
	store = inspectWebhook(t, ctx, bundle, home, driver, db)
	for i := 0; i < 3; i++ {
		id := "f06-poison-" + strconv.Itoa(i)
		dead, ok := f06Dead(store, id)
		if !ok || dead.Reason != "poison" || dead.Attempts != "5" {
			t.Fatalf("poison dead row for %s: %+v %v", id, dead, ok)
		}
		// Six claims (five failures plus the dead-letter take) bump
		// the version to at least 6; a transient claim miss in any
		// round adds reclaim rounds, so later claims only raise it.
		// Unlike F05's single-claim raw leg, the companion reclaims
		// after every lease expiry.
		version := -1
		for _, row := range store.Outbox {
			if row.DeliveryID != id {
				continue
			}
			if row.State != "dead" || row.Attempts != "5" || row.WorkerID != "worker-A" {
				t.Fatalf("bad poison outbox row %+v", row)
			}
			version, _ = strconv.Atoi(row.Version)
		}
		if version < 6 {
			t.Fatalf("poison %s version %d, want >= 6", id, version)
		}
		if stub.count(id) != 5 {
			t.Fatalf("poison stub saw %s %d times, want 5", id, stub.count(id))
		}
		if f06Attempts(store, id, "failed") != 5 {
			t.Fatalf("poison attempts for %s hold %+v", id, store.Attempts)
		}
	}
	t.Logf("F06 leg D (poison): 3 rows dead after 5 attempts each, no sixth downstream touch")

	// Leg E — destination enforcement: the denied subscription dies
	// with the policy reason and the unmapped subscription with its
	// name; neither touches the downstream.
	f06Seed(t, provider, "f06-deny-", "sub-denied", 1)
	f06Seed(t, provider, "f06-unmap-", "sub-unmapped", 1)
	run, report = companion.runOnce(ctx, t, "worker-A", carrierSecret, nil, nil)
	requireF06Shape(t, report, "worker-A")
	if len(report.Steps) != 2 {
		t.Fatalf("destination batch %d steps, want 2", len(report.Steps))
	}
	for _, step := range report.Steps {
		if step.Outcome != "dead" {
			t.Fatalf("destination step %+v, want dead", step)
		}
	}
	byID := map[string]f06Step{}
	for _, step := range report.Steps {
		byID[step.DeliveryID] = step
	}
	if byID["f06-deny-0"].Failure.Identity != "destination-denied" || byID["f06-deny-0"].Failure.Occurrence != "no-matching-rule" {
		t.Fatalf("denied step %+v", byID["f06-deny-0"])
	}
	if byID["f06-unmap-0"].Failure.Identity != "no-destination" || byID["f06-unmap-0"].Failure.Occurrence != "sub-unmapped" {
		t.Fatalf("unmapped step %+v", byID["f06-unmap-0"])
	}
	store = inspectWebhook(t, ctx, bundle, home, driver, db)
	if dead, ok := f06Dead(store, "f06-deny-0"); !ok || dead.Reason != "destination_denied" {
		t.Fatalf("denied dead row %+v %v", dead, ok)
	}
	if dead, ok := f06Dead(store, "f06-unmap-0"); !ok || dead.Reason != "no_destination" {
		t.Fatalf("unmapped dead row %+v %v", dead, ok)
	}
	if stub.count("f06-deny-0") != 0 || stub.count("f06-unmap-0") != 0 {
		t.Fatal("denied rows touched the downstream")
	}
	t.Logf("F06 leg E (destination): denied + unmapped dead with reasons, downstream untouched")

	// Leg F — credential binding: without the bound variable the send
	// fails for retry and never posts; with it the stub sees the
	// Bearer credential and the row delivers.
	f06Seed(t, provider, "f06-cred-", "sub-cred", 1)
	credTuning := []string{"--lease-ms", "1000"}
	run, report = companion.runOnce(ctx, t, "worker-A", carrierSecret, nil, credTuning)
	requireF06Shape(t, report, "worker-A")
	if len(report.Steps) != 1 || report.Steps[0].Outcome != "failed" ||
		report.Steps[0].Failure.Identity != "credential-missing" ||
		report.Steps[0].Failure.Occurrence != "F06_HOOK_TOKEN" {
		t.Fatalf("credential-missing steps %+v", report.Steps)
	}
	if stub.count("f06-cred-0") != 0 {
		t.Fatal("credential-missing row posted downstream")
	}
	time.Sleep(1500 * time.Millisecond)
	run, report = companion.runOnce(ctx, t, "worker-A", carrierSecret, []string{"F06_HOOK_TOKEN=" + hookToken}, credTuning)
	requireF06Shape(t, report, "worker-A")
	requireF06Clean(t, run.Stdout, carrierSecret, hookToken)
	if len(report.Steps) != 1 || report.Steps[0].Outcome != "delivered" {
		t.Fatalf("credential steps %+v", report.Steps)
	}
	if got := stub.authFor("f06-cred-0"); got != "Bearer "+hookToken {
		t.Fatalf("credential auth %q, want the Bearer token", got)
	}
	t.Logf("F06 leg F (credential): missing fails for retry, bound token delivers with Bearer auth")

	// Leg G — downstream timeout (E04 caller bound): the first send
	// hangs past the 1.5s bound and fails as downstream-timeout; the
	// redelivery lands instantly. The contracted occurrence is the
	// timeout, never a URL or elapsed detail.
	f06Seed(t, provider, "f06-flaky-", "sub-ok", 1)
	flakyTuning := []string{"--lease-ms", "1000", "--timeout-ms", "1500"}
	_, report = companion.runOnce(ctx, t, "worker-A", carrierSecret, nil, flakyTuning)
	requireF06Shape(t, report, "worker-A")
	if len(report.Steps) != 1 || report.Steps[0].Outcome != "failed" ||
		report.Steps[0].Failure.Identity != "downstream-timeout" ||
		report.Steps[0].Failure.Occurrence != "timeout" {
		t.Fatalf("timeout steps %+v", report.Steps)
	}
	// The in-flight heartbeat extended the 1s lease; sleep past it.
	time.Sleep(2500 * time.Millisecond)
	_, report = companion.runOnce(ctx, t, "worker-A", carrierSecret, nil, flakyTuning)
	requireF06Shape(t, report, "worker-A")
	if len(report.Steps) != 1 || report.Steps[0].Outcome != "delivered" {
		t.Fatalf("flaky redelivery steps %+v", report.Steps)
	}
	store = inspectWebhook(t, ctx, bundle, home, driver, db)
	if stub.count("f06-flaky-0") != 2 {
		t.Fatalf("flaky stub saw %d posts, want 2", stub.count("f06-flaky-0"))
	}
	if f06Attempts(store, "f06-flaky-0", "failed") != 1 || f06Attempts(store, "f06-flaky-0", "delivered") != 1 {
		t.Fatalf("flaky attempts hold %+v", store.Attempts)
	}
	t.Logf("F06 leg G (timeout): first send times out at the bound, redelivery lands")

	// Leg H — crash/unacked redelivery: a companion SIGKILLed with
	// four gated sends in flight acks nothing; after the 2s leases
	// expire a fresh companion redelivers all eight. Four rows post
	// twice downstream while the ledger holds one effect each:
	// at-least-once with idempotent settlement, never exactly-once.
	f06Seed(t, provider, "f06-gate-", "sub-ok", 8)
	gateBase := stub.total()
	doomed := companion.start(ctx, "worker-A", carrierSecret, nil, []string{"--once", "--lease-ms", "2000", "--concurrency", "4"})
	gateDeadline := time.Now().Add(20 * time.Second)
	for stub.total()-gateBase < 4 {
		if time.Now().After(gateDeadline) {
			doomed.kill()
			t.Fatalf("gated companion stalled at %d arrivals", stub.total()-gateBase)
		}
		time.Sleep(50 * time.Millisecond)
	}
	// Exactly four lanes can be in flight: no send completes while
	// the gate holds, so no fifth arrival and no ack is possible.
	doomed.kill()
	doomed.wait()
	stub.releaseGate()
	time.Sleep(3 * time.Second)
	run, report = companion.runOnce(ctx, t, "worker-B", carrierSecret, nil, []string{"--lease-ms", "2000"})
	requireF06Shape(t, report, "worker-B")
	requireF06Clean(t, run.Stdout, carrierSecret, hookToken)
	if len(report.Steps) != 8 {
		t.Fatalf("redelivery batch %d steps, want 8", len(report.Steps))
	}
	for _, step := range report.Steps {
		if step.Outcome != "delivered" {
			t.Fatalf("redelivery step %+v, want delivered", step)
		}
	}
	store = inspectWebhook(t, ctx, bundle, home, driver, db)
	twice, once := 0, 0
	for i := 0; i < 8; i++ {
		id := "f06-gate-" + strconv.Itoa(i)
		switch stub.count(id) {
		case 2:
			twice++
		case 1:
			once++
		default:
			t.Fatalf("redelivery stub saw %s %d times, want 1 or 2", id, stub.count(id))
		}
		if f06LedgerCount(store, id) != 1 {
			t.Fatalf("redelivery ledger for %s holds %+v", id, store.Ledger)
		}
		if f06Attempts(store, id, "delivered") != 1 || f06Attempts(store, id, "failed") != 0 {
			t.Fatalf("redelivery attempts for %s hold %+v", id, store.Attempts)
		}
	}
	if twice != 4 || once != 4 {
		t.Fatalf("redelivery distribution twice=%d once=%d, want 4/4", twice, once)
	}
	t.Logf("F06 leg H (crash): SIGKILL mid-batch redelivers all 8; 4 rows posted twice, ledger 1 each")

	// Leg W4.3 — bounded batch mirror: 200 rows drain through both
	// companions in capped concurrent batches while the harness
	// samples peak RSS. Every step carries the C-A step/failure/
	// occurrence shape; wall time, throughput, and peak RSS are
	// measured and logged for the evidence record.
	const w43Rows = 200
	f06Seed(t, provider, "f06-w43-", "sub-ok", w43Rows)
	var peakRSS atomic.Int32
	drainStart := time.Now()
	w43Seen := map[string]int{}
	w43Batches := 0
	for round := 1; ; round++ {
		if round > 10 {
			t.Fatal("W4.3 outbox never drained")
		}
		pA := companion.start(ctx, "worker-A", carrierSecret, nil, []string{"--once"})
		pB := companion.start(ctx, "worker-B", carrierSecret, nil, []string{"--once"})
		stopSample := make(chan struct{})
		var sampled sync.WaitGroup
		sampled.Add(1)
		go func() {
			defer sampled.Done()
			for {
				select {
				case <-stopSample:
					return
				default:
					if rss := sampleF06RSS(pA.pid(), pB.pid()); rss > 0 {
						for {
							peak := peakRSS.Load()
							if int32(rss) <= peak || peakRSS.CompareAndSwap(peak, int32(rss)) {
								break
							}
						}
					}
					time.Sleep(100 * time.Millisecond)
				}
			}
		}()
		rA, okA := pA.waitTimeout(3 * time.Minute)
		rB, okB := pB.waitTimeout(3 * time.Minute)
		close(stopSample)
		sampled.Wait()
		if !okA || !okB {
			health, _, _ := httpGet(base + "/health")
			hist := map[int]int{}
			for i := 0; i < w43Rows; i++ {
				hist[stub.count("f06-w43-"+strconv.Itoa(i))]++
			}
			store := inspectWebhook(t, ctx, bundle, home, driver, db)
			claimed, pending, acked := 0, 0, 0
			for _, row := range store.Outbox {
				if !strings.HasPrefix(row.DeliveryID, "f06-w43-") {
					continue
				}
				if row.WorkerID != "" {
					claimed++
				} else {
					pending++
				}
			}
			for _, row := range store.Attempts {
				if strings.HasPrefix(row.DeliveryID, "f06-w43-") && row.Outcome == "delivered" {
					acked++
				}
			}
			probe := "unprobed"
			func() {
				defer func() {
					if recover() != nil {
						probe = "probe-fatal"
					}
				}()
				client := &http.Client{Timeout: 5 * time.Second}
				fields, _ := json.Marshal(ackRequest{DeliveryID: "f06-w43-0", Settled: true, TimestampMs: time.Now().UnixMilli(), Nonce: "f06probe000000000000000000000001"})
				mac := hmac.New(sha256.New, []byte(carrierSecret))
				mac.Write(fields)
				request, _ := http.NewRequest("POST", base+"/outbox/ack", bytes.NewReader(fields))
				request.Header.Set("Content-Type", "application/json")
				request.Header.Set("X-Carrier-Protocol", "1")
				request.Header.Set("X-Carrier-Signature", hex.EncodeToString(mac.Sum(nil)))
				response, err := client.Do(request)
				if err != nil {
					probe = "transport: " + err.Error()
					return
				}
				defer response.Body.Close()
				payload, _ := io.ReadAll(response.Body)
				probe = strconv.Itoa(response.StatusCode) + " " + string(payload)
			}()
			t.Fatalf("W4.3 round %d wedged: A ok=%v B ok=%v stub=%d drained=%d canHealth=%d receipts=%v claimed=%d pending=%d acked=%d ackProbe=%s",
				round, okA, okB, stub.total(), len(w43Seen), health, hist, claimed, pending, acked, probe)
		}
		if rA.Code != 0 || rB.Code != 0 {
			t.Fatalf("W4.3 round %d: A=%d %q B=%d %q", round, rA.Code, rA.Stderr, rB.Code, rB.Stderr)
		}
		repA := parseF06Reports(t, rA.Stdout)
		repB := parseF06Reports(t, rB.Stdout)
		if len(repA) != 1 || len(repB) != 1 {
			t.Fatalf("W4.3 round %d reports %d/%d, want 1/1", round, len(repA), len(repB))
		}
		requireF06Shape(t, repA[0], "worker-A")
		requireF06Shape(t, repB[0], "worker-B")
		requireF06Clean(t, rA.Stdout, carrierSecret, hookToken)
		requireF06Clean(t, rB.Stdout, carrierSecret, hookToken)
		if round == 1 && (len(repA[0].Steps) == 0 || len(repB[0].Steps) == 0) {
			t.Fatalf("W4.3 round 1 split %d/%d, want both workers draining", len(repA[0].Steps), len(repB[0].Steps))
		}
		for _, step := range append(append([]f06Step{}, repA[0].Steps...), repB[0].Steps...) {
			if step.Outcome != "delivered" {
				t.Fatalf("W4.3 step %+v, want delivered", step)
			}
			w43Seen[step.DeliveryID]++
		}
		w43Batches += 2
		t.Logf("F06 W4.3 round %d: A=%d B=%d drained=%d", round, len(repA[0].Steps), len(repB[0].Steps), len(w43Seen))
		if len(repA[0].Steps) == 0 && len(repB[0].Steps) == 0 {
			break
		}
	}
	wall := time.Since(drainStart)
	if len(w43Seen) != w43Rows {
		t.Fatalf("W4.3 union %d rows, want %d", len(w43Seen), w43Rows)
	}
	for id, times := range w43Seen {
		if times != 1 {
			t.Fatalf("W4.3 row %s in %d steps, want 1", id, times)
		}
	}
	store = inspectWebhook(t, ctx, bundle, home, driver, db)
	for i := 0; i < w43Rows; i++ {
		id := "f06-w43-" + strconv.Itoa(i)
		if stub.count(id) != 1 {
			t.Fatalf("W4.3 stub saw %s %d times, want 1", id, stub.count(id))
		}
		if f06LedgerCount(store, id) != 1 || f06Attempts(store, id, "delivered") != 1 {
			t.Fatalf("W4.3 store for %s incomplete", id)
		}
	}
	peakKiB := int(peakRSS.Load())
	if wall > 120*time.Second {
		t.Fatalf("W4.3 drain took %v, want under 120s", wall)
	}
	// Blowup guard, not a performance gate: the measured drain peaks
	// near 56 MiB for both companions, so 256 MiB allows 4.5x headroom
	// while catching retained-batch growth. Throughput is recorded,
	// never gated: loaded hosts legitimately drain slower.
	if peakKiB <= 0 || peakKiB >= 256*1024 {
		t.Fatalf("W4.3 peak companion RSS %d KiB, want 1..262143", peakKiB)
	}
	t.Logf("F06 W4.3 (batch mirror): %d rows in %d batches over %.1fs (%.0f rows/s), peak companion RSS %d KiB",
		w43Rows, w43Batches, wall.Seconds(), float64(w43Rows)/wall.Seconds(), peakKiB)

	// Leg I — backoff/idle sleeps plus SIGTERM drain: one looping
	// companion meets an always-500 downstream. The first batch fails
	// both rows, later batches idle empty, every gap proves a sleep
	// (no hot loop), and SIGTERM drains to exit 0. The exact ladder
	// stays pinned by the runWorker unit tests; this leg proves the
	// live sleeps exist and the worker keeps polling.
	f06Seed(t, provider, "f06-idle-", "sub-poison", 2)
	stopLoop, loopLines := companion.runLoop(ctx, t, "worker-A", carrierSecret, nil, nil)
	time.Sleep(6 * time.Second)
	loopRun := stopLoop()
	if loopRun.Code != 0 {
		t.Fatalf("looping companion exit %d stderr=%q", loopRun.Code, loopRun.Stderr)
	}
	if strings.Contains(loopRun.Stderr, "restart") {
		t.Fatalf("supervisor restarted during the idle leg: %q", loopRun.Stderr)
	}
	timed := loopLines()
	if len(timed) < 3 {
		t.Fatalf("looping companion printed %d batches, want >= 3", len(timed))
	}
	requireF06Shape(t, timed[0].Report, "worker-A")
	if len(timed[0].Report.Steps) != 2 {
		t.Fatalf("first loop batch %+v, want 2 failed steps", timed[0].Report.Steps)
	}
	for _, step := range timed[0].Report.Steps {
		if step.Outcome != "failed" || step.Failure.Identity != "downstream-status" || step.Failure.Occurrence != "500" {
			t.Fatalf("first loop step %+v, want failed downstream-status/500", step)
		}
	}
	for _, tl := range timed {
		requireF06Clean(t, tl.Line, carrierSecret, hookToken)
	}
	for i := 1; i < len(timed); i++ {
		requireF06Shape(t, timed[i].Report, "worker-A")
		if len(timed[i].Report.Steps) != 0 {
			t.Fatalf("idle batch %d holds steps %+v", i, timed[i].Report.Steps)
		}
		if gap := timed[i].At.Sub(timed[i-1].At); gap < 700*time.Millisecond || gap > 6*time.Second {
			t.Fatalf("inter-batch gap %v outside 0.7s..6s", gap)
		}
	}
	t.Logf("F06 leg I (backoff): %d batches, first failed 2 rows, idle gaps all slept, SIGTERM drained to 0", len(timed))

	t.Logf("F06 pair: build %s, stub receipts %d total", firstID[:12], stub.total())
}
