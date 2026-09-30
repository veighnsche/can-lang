package acceptance

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/tools/native-test-owner/admission"
	"github.com/veighnsche/can-lang/tools/native-test-owner/host"
	"github.com/veighnsche/can-lang/tools/native-test-owner/journal"
	"github.com/veighnsche/can-lang/tools/native-test-owner/process"
	"github.com/veighnsche/can-lang/tools/native-test-owner/protocol/codec"
	"github.com/veighnsche/can-lang/tools/native-test-owner/receipt/cleanup"
	"github.com/veighnsche/can-lang/tools/native-test-owner/recovery"
)

// sleeperSecs marks stray-prone fixture sleepers so ps sweeps can prove
// none survive a test.
const sleeperSecs = "31"

func testWitness(t *testing.T) *Witness {
	t.Helper()
	w, err := NewWitness(t.TempDir())
	if err != nil {
		t.Fatalf("NewWitness: %v", err)
	}
	t.Cleanup(func() {
		w.Close()
		if n := w.Live(); n != 0 {
			t.Errorf("Live() = %d, want 0 (stray N processes)", n)
		}
	})
	return w
}

func testQual(t *testing.T) *host.Qualification {
	t.Helper()
	requireDarwin(t)
	q, err := host.Qualify(host.Darwin(), host.Quick(), host.QualifyOpts{TmpParent: t.TempDir()})
	if err != nil || !q.Valid() {
		t.Fatalf("quick did not qualify: %v %v", err, q.Failing())
	}
	return q
}

func testSelect(t *testing.T) *SelectedHost {
	t.Helper()
	sel, err := Select(host.Darwin(), testQual(t), t.TempDir())
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	return sel
}

func testSubject(t *testing.T) NSubject {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	abs, err := filepath.Abs(exe)
	if err != nil {
		t.Fatalf("Abs: %v", err)
	}
	id, err := AttestN("n-subject", "1.0.0-test", abs)
	if err != nil {
		t.Fatalf("AttestN: %v", err)
	}
	return NSubject{Identity: id, Args: NHelperArgs()}
}

func testRequest() admission.Request {
	return admission.Request{
		Demand:         admission.Demand{Live: true},
		Budget:         60 * time.Second,
		Body:           20 * time.Second,
		CleanupReserve: 5 * time.Second,
	}
}

func testRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("go.mod not found above %s", dir)
		}
		dir = parent
	}
}

func testConfig(t *testing.T, w *Witness, sel *SelectedHost, opID string, n NSubject) Config {
	t.Helper()
	return Config{
		OpID:        opID,
		N:           n,
		T:           w,
		Host:        sel,
		Root:        testRepoRoot(t),
		Request:     testRequest(),
		TmpParent:   t.TempDir(),
		ReceiptPath: filepath.Join(t.TempDir(), "receipt.json"),
	}
}

// ackFrameForEnv precomputes the child-side status bytes over the exact
// offered environment: sorted-key JSON snapshot, content-bound ack
// payload, one codec event frame. Shell fixtures embed it to stand in
// for subjects that hash fd 3 themselves.
func ackFrameForEnv(t *testing.T, opID string, env map[string]string) []byte {
	t.Helper()
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	ordered := make(map[string]string, len(env))
	for _, k := range keys {
		ordered[k] = env[k]
	}
	snap, err := json.Marshal(ordered)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	sum := sha256.Sum256(snap)
	payload := "ACK:" + opID + ":" + hex.EncodeToString(sum[:])
	var buf bytes.Buffer
	if err := codec.NewEncoder(&buf, 1<<20).Encode(codec.KindEvent, []byte(payload)); err != nil {
		t.Fatalf("encode: %v", err)
	}
	return buf.Bytes()
}

// sweepSleepers fails the test when a tagged fixture sleeper survives.
// It matches only the exact sleeper image, never a shell whose command
// line merely mentions the tag.
func sweepSleepers(t *testing.T) {
	t.Helper()
	out, err := exec.Command("ps", "-ax", "-o", "command=").Output()
	if err != nil {
		t.Skipf("ps sweep unavailable: %v", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "/bin/sleep" && fields[1] == sleeperSecs {
			t.Errorf("stray fixture sleeper survives: %q", strings.TrimSpace(line))
		}
	}
}

func TestAttestN(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	abs, _ := filepath.Abs(exe)
	id, err := AttestN("n-subject", "1.0.0-test", abs)
	if err != nil {
		t.Fatalf("AttestN: %v", err)
	}
	if id.Digest == "" || id.Executable == "" {
		t.Fatalf("attestation is empty: %+v", id)
	}
	if err := id.Verify(); err != nil {
		t.Fatalf("Verify: %v", err)
	}
	tampered := id
	tampered.Digest = "sha256:" + strings.Repeat("0", 64)
	if err := tampered.Verify(); !errors.Is(err, ErrNAttest) {
		t.Fatalf("tampered digest Verify = %v, want ErrNAttest", err)
	}
	if _, err := AttestN("", "1.0", abs); !errors.Is(err, ErrNAttest) {
		t.Fatalf("empty name = %v, want ErrNAttest", err)
	}
	if _, err := AttestN("n", "", abs); !errors.Is(err, ErrNAttest) {
		t.Fatalf("empty version = %v, want ErrNAttest", err)
	}
	if _, err := AttestN("n", "1.0", "relative/path"); !errors.Is(err, ErrNAttest) {
		t.Fatalf("relative path = %v, want ErrNAttest", err)
	}
	if _, err := AttestN("n", "1.0", filepath.Join(t.TempDir(), "missing")); !errors.Is(err, ErrNAttest) {
		t.Fatalf("missing file = %v, want ErrNAttest", err)
	}
}

func TestSelectHost(t *testing.T) {
	requireDarwin(t)
	q := testQual(t)
	sel, err := Select(host.Darwin(), q, t.TempDir())
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if sel.AdapterName != "darwin-strict" || sel.Foreign {
		t.Fatalf("selection = %+v, want local darwin-strict", sel)
	}
	if sel.Envelope.Name != "quick" || sel.Facts.GOOS != "darwin" {
		t.Fatalf("selection scope = %+v, want quick/darwin", sel)
	}
	if _, err := Select(nil, q, t.TempDir()); !errors.Is(err, ErrHostNotQualified) {
		t.Fatalf("nil adapter = %v, want ErrHostNotQualified", err)
	}
	if _, err := Select(host.Darwin(), nil, t.TempDir()); !errors.Is(err, ErrHostNotQualified) {
		t.Fatalf("nil qual = %v, want ErrHostNotQualified", err)
	}
	if _, err := Select(host.Unavailable("x"), q, t.TempDir()); !errors.Is(err, ErrHostNotQualified) {
		t.Fatalf("wrong adapter = %v, want ErrHostNotQualified", err)
	}
	bad, _ := host.Qualify(host.Unavailable("no mechanism"), host.Quick(), host.QualifyOpts{TmpParent: t.TempDir()})
	if bad.Valid() {
		t.Fatalf("unavailable adapter qualified")
	}
	if _, err := Select(host.Unavailable("no mechanism"), bad, t.TempDir()); !errors.Is(err, ErrHostNotQualified) {
		t.Fatalf("unavailable qual = %v, want ErrHostNotQualified", err)
	}
	drifted := *q
	drifted.Facts.MemBytes = 1
	if _, err := Select(host.Darwin(), &drifted, t.TempDir()); !errors.Is(err, ErrHostDrift) {
		t.Fatalf("drifted facts = %v, want ErrHostDrift", err)
	}
}

func TestUnsupportedEnforcementNeverPasses(t *testing.T) {
	bad, err := host.Qualify(host.Unavailable("no mechanism here"), host.Quick(),
		host.QualifyOpts{TmpParent: t.TempDir()})
	if err != nil {
		t.Fatalf("battery did not run: %v", err)
	}
	if bad.Valid() {
		t.Fatalf("unsupported enforcement qualified")
	}
	if _, err := Select(host.Unavailable("no mechanism here"), bad, t.TempDir()); err == nil {
		t.Fatalf("unsupported enforcement selected")
	}
	if _, err := SelectOther("", host.HostFacts{}, host.Quick()); !errors.Is(err, ErrForeignHost) {
		t.Fatalf("nameless other host = %v, want ErrForeignHost", err)
	}
}

func TestSelectOtherRefusesLocally(t *testing.T) {
	w := testWitness(t)
	other, err := SelectOther("other-host", host.HostFacts{GOOS: "linux"}, host.Quick())
	if err != nil {
		t.Fatalf("SelectOther: %v", err)
	}
	cfg := testConfig(t, w, other, "foreign1", testSubject(t))
	m, err := Accept(cfg)
	if err != nil {
		t.Fatalf("Accept: %v", err)
	}
	if m.Complete {
		t.Fatalf("foreign host completed: %+v", m)
	}
	if !strings.Contains(m.Reason, "foreign-host") {
		t.Fatalf("Reason=%q, want foreign-host refusal", m.Reason)
	}
	// Explicit selection recorded, execution never moved: no spawn
	// intent exists and no child was retained.
	if _, ok := w.journal.Lookup("foreign1"); ok {
		t.Fatalf("foreign run left spawn intent: execution moved")
	}
	if n := w.Live(); n != 0 {
		t.Fatalf("Live() = %d, want 0", n)
	}
	// Its own receipt exists but is not green.
	if _, err := os.Stat(cfg.ReceiptPath); err != nil {
		t.Fatalf("refusal receipt missing: %v", err)
	}
	if err := Verify(cfg.ReceiptPath, m.Receipt.OperationID); err == nil {
		t.Fatalf("foreign refusal receipt verifies as success")
	}
}

func TestAcceptWitnessLost(t *testing.T) {
	w := testWitness(t)
	other, err := SelectOther("other-host", host.HostFacts{GOOS: "darwin"}, host.Quick())
	if err != nil {
		t.Fatalf("SelectOther: %v", err)
	}
	w.Kill()
	cfg := testConfig(t, w, other, "lost1", testSubject(t))
	m, err := Accept(cfg)
	if err != nil {
		t.Fatalf("Accept: %v", err)
	}
	if m.Complete {
		t.Fatalf("dead witness completed: %+v", m)
	}
	if m.Reason != "witness-lost" {
		t.Fatalf("Reason=%q, want witness-lost", m.Reason)
	}
	if _, err := os.Stat(cfg.ReceiptPath); !os.IsNotExist(err) {
		t.Fatalf("dead witness left a receipt: %v", err)
	}
}

func TestVerifyReceipt(t *testing.T) {
	w := testWitness(t)
	sum := sha256.Sum256([]byte("verify-probe"))
	opID := "verify1"
	if _, err := w.journal.Reserve(opID, "sha256:"+hex.EncodeToString(sum[:]),
		w.id, journal.PathIdentity{}, "", 0); err != nil {
		t.Fatalf("Reserve: %v", err)
	}
	scan := recovery.Scan(w.journal, w.id, recovery.ScanConfig{})
	rec, err := cleanup.Perform(w.journal, cleanup.Request{
		OperationID: opID, Self: w.id,
		Allow: scan.Allowed(opID), DenyReason: scan.ReasonFor(opID),
		Prior: cleanup.OutcomePass,
	}, nil)
	if err != nil || !rec.Complete {
		t.Fatalf("Perform = %+v err=%v, want complete", rec, err)
	}
	path := filepath.Join(t.TempDir(), "receipt.json")
	if err := cleanup.WriteFile(path, rec); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := Verify(path, opID); err != nil {
		t.Fatalf("Verify green: %v", err)
	}
	if err := Verify(path, "someone-else"); err == nil {
		t.Fatalf("wrong operation verified")
	}
	// Forged receipt: tampered bytes fail closed.
	raw, _ := os.ReadFile(path)
	raw[len(raw)/2] ^= 0xff
	forged := filepath.Join(t.TempDir(), "forged.json")
	if err := os.WriteFile(forged, raw, 0o600); err != nil {
		t.Fatalf("write forged: %v", err)
	}
	if err := Verify(forged, opID); err == nil {
		t.Fatalf("forged receipt verified as success")
	}
	// Absent receipt fails closed.
	if err := Verify(filepath.Join(t.TempDir(), "absent.json"), opID); err == nil {
		t.Fatalf("absent receipt verified as success")
	}
	// An incomplete refusal receipt is not a pass either.
	denied, _ := cleanup.Perform(w.journal, cleanup.Request{
		OperationID: opID, Self: w.id, Allow: false,
		DenyReason: "denied", Prior: cleanup.OutcomeFailed,
	}, nil)
	deniedPath := filepath.Join(t.TempDir(), "denied.json")
	if err := cleanup.WriteFile(deniedPath, denied); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := Verify(deniedPath, opID); err == nil {
		t.Fatalf("refusal receipt verified as success")
	}
}

func TestAcceptMisuse(t *testing.T) {
	w := testWitness(t)
	sel, err := SelectOther("other-host", host.HostFacts{}, host.Quick())
	if err != nil {
		t.Fatalf("SelectOther: %v", err)
	}
	base := testConfig(t, w, sel, "misuse1", testSubject(t))
	cases := []struct {
		name string
		mut  func(*Config)
	}{
		{"nil-witness", func(c *Config) { c.T = nil }},
		{"nil-host", func(c *Config) { c.Host = nil }},
		{"bad-op", func(c *Config) { c.OpID = "-bad" }},
		{"unattested", func(c *Config) { c.N.Identity = NIdentity{} }},
		{"relative-root", func(c *Config) { c.Root = "relative" }},
		{"relative-receipt", func(c *Config) { c.ReceiptPath = "relative.json" }},
		{"target-without-remover", func(c *Config) {
			c.CleanupTarget = filepath.Join(t.TempDir(), "x")
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := base
			tc.mut(&cfg)
			if _, err := Accept(cfg); !errors.Is(err, ErrConfig) {
				t.Fatalf("Accept = %v, want ErrConfig", err)
			}
		})
	}
}

func checkManifestShape(t *testing.T, m *Manifest) {
	t.Helper()
	if m.SchemaVersion != ManifestSchemaVersion {
		t.Fatalf("schema %d, want %d", m.SchemaVersion, ManifestSchemaVersion)
	}
	if len(m.Checks) != 7 {
		t.Fatalf("%d checks, want 7", len(m.Checks))
	}
	for i, want := range []string{"witness", "host-scope", "protocol", "enforcement", "journal", "recovery", "receipt"} {
		if m.Checks[i].Name != want {
			t.Fatalf("check %d = %q, want %q", i, m.Checks[i].Name, want)
		}
	}
	if m.N.Executable == "" || m.N.Digest == "" || m.N.Version == "" {
		t.Fatalf("N not bound: %+v", m.N)
	}
	if m.T.PID != os.Getpid() || m.T.StartToken == "" {
		t.Fatalf("T not bound: %+v", m.T)
	}
	if m.Host.Adapter == "" || m.Host.Envelope == "" {
		t.Fatalf("host not bound: %+v", m.Host)
	}
}

func TestAcceptHappy(t *testing.T) {
	requireDarwin(t)
	w := testWitness(t)
	sel := testSelect(t)
	cfg := testConfig(t, w, sel, "happy1", testSubject(t))
	m, err := Accept(cfg)
	if err != nil {
		t.Fatalf("Accept: %v", err)
	}
	checkManifestShape(t, m)
	if !m.Complete {
		t.Fatalf("Complete=false reason=%q checks=%+v release=%+v", m.Reason, m.Checks, m.Release)
	}
	for _, c := range m.Checks {
		if !c.Pass {
			t.Fatalf("check %q failed: %s", c.Name, c.Detail)
		}
	}
	rel := m.Release
	if !rel.Offered || !rel.WriterDone || rel.WriterErr != "" ||
		!rel.Accepted || rel.Malformed || !rel.EOF || !rel.Reaped {
		t.Fatalf("release facts = %+v, want all positive", rel)
	}
	if rel.ExitCode != 0 || rel.Signaled {
		t.Fatalf("exit = %+v, want code 0", rel)
	}
	if !rel.LeaseHeld || !rel.LeaseRelease || rel.Orphan || !rel.Clean {
		t.Fatalf("lease = %+v, want held+released clean, no orphan", rel)
	}
	if m.Receipt.Outcome != "pass" || !m.Receipt.Complete {
		t.Fatalf("receipt = %+v, want a complete pass", m.Receipt)
	}
	if err := Verify(cfg.ReceiptPath, m.Receipt.OperationID); err != nil {
		t.Fatalf("Verify: %v", err)
	}
	// Independent witnesses: the lease is gone, no child is retained,
	// and the spawn intent reached closed in T's journal.
	if held, err := process.ProbeLease(filepath.Join(w.dir, "witness.lock")); err != nil || held {
		t.Fatalf("ProbeLease: held=%v err=%v, want released", held, err)
	}
	if n := w.Live(); n != 0 {
		t.Fatalf("Live() = %d, want 0", n)
	}
	res, ok := w.journal.Lookup("happy1")
	if !ok || res.State != journal.StateClosed {
		t.Fatalf("spawn intent = %+v ok=%v, want closed", res, ok)
	}
}

func TestAcceptNDeath(t *testing.T) {
	requireDarwin(t)
	w := testWitness(t)
	sel := testSelect(t)
	n := testSubject(t)
	n.Env = map[string]string{EnvMode: ModeDie}
	cfg := testConfig(t, w, sel, "death1", n)
	m, err := Accept(cfg)
	if err != nil {
		t.Fatalf("Accept: %v", err)
	}
	if m.Complete {
		t.Fatalf("N death completed: %+v", m)
	}
	if !strings.Contains(m.Reason, "release:") {
		t.Fatalf("Reason=%q, want a release refusal", m.Reason)
	}
	if m.Release.ExitCode != 1 || m.Release.Accepted {
		t.Fatalf("release = %+v, want exit 1 without ack", m.Release)
	}
	if err := Verify(cfg.ReceiptPath, m.Receipt.OperationID); err == nil {
		t.Fatalf("N-death receipt verifies as success")
	}
	res, ok := w.journal.Lookup("death1")
	if !ok || res.State != journal.StateFailed {
		t.Fatalf("spawn intent = %+v ok=%v, want failed", res, ok)
	}
	if n := w.Live(); n != 0 {
		t.Fatalf("Live() = %d, want 0", n)
	}
}

// TestRefuseCarriesDenialWithTarget proves refusal receipts name the
// operative denial even when a cleanup target is reserved: the
// authority check passes on the reserved path, so the receipt carries
// the denial reason instead of misreporting a replaced path, and the
// remover is never called.
func TestRefuseCarriesDenialWithTarget(t *testing.T) {
	requireDarwin(t)
	w := testWitness(t)
	sel := testSelect(t)
	n := testSubject(t)
	n.Env = map[string]string{EnvMode: ModeDie}
	cfg := testConfig(t, w, sel, "refusetarget1", n)
	target := filepath.Join(t.TempDir(), "target.txt")
	if err := os.WriteFile(target, []byte("intact"), 0o600); err != nil {
		t.Fatalf("write target: %v", err)
	}
	cfg.CleanupTarget = target
	called := false
	cfg.Remove = func(path string) error { called = true; return nil }
	m, err := Accept(cfg)
	if err != nil {
		t.Fatalf("Accept: %v", err)
	}
	if m.Complete {
		t.Fatalf("N death completed: %+v", m)
	}
	if !strings.Contains(m.Reason, "release:") {
		t.Fatalf("Reason=%q, want a release refusal", m.Reason)
	}
	if !strings.Contains(m.Receipt.Reason, "release:") {
		t.Fatalf("receipt reason=%q, want the release denial, not a path refusal", m.Receipt.Reason)
	}
	if called {
		t.Fatalf("refusal called the remover")
	}
	if raw, err := os.ReadFile(target); err != nil || string(raw) != "intact" {
		t.Fatalf("refusal touched the target: %q err=%v", raw, err)
	}
	if err := Verify(cfg.ReceiptPath, m.Receipt.OperationID); err == nil {
		t.Fatalf("refusal receipt verifies as success")
	}
}

func TestAcceptVersionFactsStripped(t *testing.T) {
	requireDarwin(t)
	w := testWitness(t)
	sel := testSelect(t)
	cfg := testConfig(t, w, sel, "noversion1", testSubject(t))
	cfg.OmitVersionFacts = true
	m, err := Accept(cfg)
	if err != nil {
		t.Fatalf("Accept: %v", err)
	}
	if m.Complete {
		t.Fatalf("versionless N completed: %+v", m)
	}
	if m.Release.ExitCode != 3 {
		t.Fatalf("exit = %+v, want N's version refusal (3)", m.Release)
	}
	if err := Verify(cfg.ReceiptPath, m.Receipt.OperationID); err == nil {
		t.Fatalf("versionless receipt verifies as success")
	}
}

func TestAcceptOrphan(t *testing.T) {
	requireDarwin(t)
	w := testWitness(t)
	sel := testSelect(t)
	shID, err := AttestN("sh-fixture", "1.0-sh", "/bin/sh")
	if err != nil {
		t.Fatalf("AttestN sh: %v", err)
	}
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "grandchild.pid")
	cfg := testConfig(t, w, sel, "orphan1", NSubject{Identity: shID})
	// Well-behaved except for orphaning a grandchild that inherits fd 4
	// (stdio redirected away so EOF is clean). The trailing sleep holds
	// the status channel open past T's collection; without it a fast
	// direct child can exit before the witness drains the pipe.
	script := fmt.Sprintf(`/bin/cat <&3 >/dev/null; printf '%%b' '%s'; /bin/sleep 31 </dev/null >/dev/null 2>&1 & echo $! > "$1"; /bin/sleep 1`,
		process.OctalEscape(ackFrameForEnv(t, cfg.OpID, cfg.ChildEnv())))
	cfg.N.Args = []string{"-c", script, "sh", pidFile}
	m, err := Accept(cfg)
	if err != nil {
		t.Fatalf("Accept: %v", err)
	}
	// Reap the stray grandchild first so no /tmp scratch or sleeper
	// survives whatever the verdict asserts below.
	raw, rerr := os.ReadFile(pidFile)
	if rerr != nil {
		t.Fatalf("read grandchild pid: %v", rerr)
	}
	var gpid int
	if _, serr := fmt.Sscanf(strings.TrimSpace(string(raw)), "%d", &gpid); serr != nil || gpid <= 0 {
		t.Fatalf("parse grandchild pid %q: %v", raw, serr)
	}
	if gp, ferr := os.FindProcess(gpid); ferr != nil {
		t.Fatalf("FindProcess: %v", ferr)
	} else {
		_ = gp.Kill()
	}
	deadline := time.Now().Add(10 * time.Second)
	for {
		held, perr := process.ProbeLease(filepath.Join(w.dir, "witness.lock"))
		if perr != nil {
			t.Fatalf("ProbeLease: %v", perr)
		}
		if !held || time.Now().After(deadline) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	sweepSleepers(t)

	if m.Complete {
		t.Fatalf("orphan run completed: %+v", m)
	}
	if !m.Release.Orphan || m.Release.Reason != "orphaned descendant still holds the lease" {
		t.Fatalf("release = %+v, want the orphan refusal", m.Release)
	}
	if !m.Release.Accepted || !m.Release.EOF {
		t.Fatalf("release = %+v, want accepted+EOF (orphan is the sole cause)", m.Release)
	}
	if err := Verify(cfg.ReceiptPath, m.Receipt.OperationID); err == nil {
		t.Fatalf("orphan receipt verifies as success")
	}
}

func TestAcceptLingeringDescriptor(t *testing.T) {
	requireDarwin(t)
	w := testWitness(t)
	sel := testSelect(t)
	shID, err := AttestN("sh-fixture", "1.0-sh", "/bin/sh")
	if err != nil {
		t.Fatalf("AttestN sh: %v", err)
	}
	cfg := testConfig(t, w, sel, "linger1", NSubject{
		Identity: shID,
		Args:     []string{"-c", "exec /bin/sleep 31"},
	})
	cfg.AckTimeout = 2 * time.Second
	cfg.EofTimeout = 2 * time.Second
	cfg.WaitTimeout = 2 * time.Second
	m, err := Accept(cfg)
	if err != nil {
		t.Fatalf("Accept: %v", err)
	}
	sweepSleepers(t)
	if m.Complete {
		t.Fatalf("lingering descriptor completed: %+v", m)
	}
	if !m.Release.LeaseHeld || m.Release.Reaped {
		t.Fatalf("release = %+v, want held-but-unreaped at decision", m.Release)
	}
	if err := Verify(cfg.ReceiptPath, m.Receipt.OperationID); err == nil {
		t.Fatalf("lingering receipt verifies as success")
	}
	if held, err := process.ProbeLease(filepath.Join(w.dir, "witness.lock")); err != nil || held {
		t.Fatalf("ProbeLease: held=%v err=%v, want released after abort", held, err)
	}
}

func TestAcceptCleanupFailure(t *testing.T) {
	requireDarwin(t)
	w := testWitness(t)
	sel := testSelect(t)
	cfg := testConfig(t, w, sel, "cleanupfail1", testSubject(t))
	cfg.CleanupTarget = filepath.Join(t.TempDir(), "target")
	cfg.Remove = func(path string) error { return fmt.Errorf("injected removal failure") }
	m, err := Accept(cfg)
	if err != nil {
		t.Fatalf("Accept: %v", err)
	}
	if m.Complete {
		t.Fatalf("cleanup failure completed: %+v", m)
	}
	if !strings.Contains(m.Reason, "cleanup:") {
		t.Fatalf("Reason=%q, want a cleanup refusal", m.Reason)
	}
	if m.Release.Clean != true {
		t.Fatalf("release = %+v, want the body clean (cleanup is the sole cause)", m.Release)
	}
	if m.Receipt.Complete || m.Receipt.Reason != "remove-failed" {
		t.Fatalf("receipt = %+v, want the remove-failed refusal", m.Receipt)
	}
	if err := Verify(cfg.ReceiptPath, m.Receipt.OperationID); err == nil {
		t.Fatalf("cleanup-failure receipt verifies as success")
	}
}

// TestNAcceptanceArtifact emits the N acceptance manifest as JSON: N and
// T identities, the exact host profile scope, the behavior bindings,
// the independent release facts and the receipt facts. The integrator
// commits this record separately; the package writes nothing under
// docs/ itself. Run with -v to read the artifact.
func TestNAcceptanceArtifact(t *testing.T) {
	requireDarwin(t)
	w := testWitness(t)
	sel := testSelect(t)
	cfg := testConfig(t, w, sel, "artifact1", testSubject(t))
	m, err := Accept(cfg)
	if err != nil {
		t.Fatalf("Accept: %v", err)
	}
	checkManifestShape(t, m)
	if !m.Complete {
		t.Fatalf("Complete=false reason=%q checks=%+v", m.Reason, m.Checks)
	}
	if err := Verify(cfg.ReceiptPath, m.Receipt.OperationID); err != nil {
		t.Fatalf("artifact receipt does not verify: %v", err)
	}
	t.Logf("N_ACCEPTANCE_ARTIFACT:\n%s", m.JSON())
}
