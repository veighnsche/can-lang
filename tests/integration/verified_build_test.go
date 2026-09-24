package integration

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/distribution"
)

func TestCurrentBundledVerifiedBuild(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for verified-build execution")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "verified-build")
	if err != nil {
		t.Fatal(err)
	}
	frozen := filepath.Join(sourceRoot, "docs/syntax-taste/evidence/2026-09-22/implementation-gaps/projects")
	copyFrozen := func(name string) string {
		t.Helper()
		root, err := filepath.EvalSymlinks(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		err = filepath.WalkDir(filepath.Join(frozen, name), func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(filepath.Join(frozen, name), path)
			if err != nil {
				return err
			}
			target := filepath.Join(root, rel)
			if entry.IsDir() {
				return os.MkdirAll(target, 0700)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			return os.WriteFile(target, data, 0600)
		})
		if err != nil {
			t.Fatal(err)
		}
		return root
	}
	write := func(root, name, text string) {
		t.Helper()
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	runAt := func(root, command string, args ...string) (int, string, string) {
		t.Helper()
		profile := `(version 1)(allow default)(deny network*)(allow network-outbound (remote ip "localhost:*"))`
		argv := []string{"-p", profile, filepath.Join(bundle, "bin/canlc"), command}
		if command == "build" {
			argv = append(argv, args...)
			argv = append(argv, root)
		} else {
			argv = append(argv, root)
			argv = append(argv, args...)
		}
		if command == "run" {
			argv = append(argv, "--")
		}
		cmd := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", argv...)
		cmd.Dir = t.TempDir()
		cmd.Env = []string{"PATH=/nonexistent", "HOME=" + cmd.Dir}
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
	current := func(root string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(root, "dist/current.json"))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}

	// G2 passing project: a full passing build publishes one complete
	// verified generation with identities, totals and deadline policy.
	passing := copyFrozen("passing")
	code, out, diag := runAt(passing, "build")
	if code != 0 || diag != "" {
		t.Fatalf("passing build: %d %s %s", code, out, diag)
	}
	var report struct {
		SchemaVersion int    `json:"schemaVersion"`
		Kind          string `json:"kind"`
		BuildID       string `json:"buildID"`
		Directory     string `json:"directory"`
		Entry         string `json:"entry"`
		Inputs        struct {
			Source, Dependencies, Catalogue, Compiler, Runtime, Options string
		} `json:"inputs"`
		Assertions struct {
			Roots    int      `json:"roots"`
			Passed   int      `json:"passed"`
			Failed   int      `json:"failed"`
			Evidence []string `json:"evidence"`
		} `json:"assertions"`
		TimeoutMs  int    `json:"timeoutMs"`
		Validation string `json:"validation"`
	}
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("invalid build report: %s", out)
	}
	if report.SchemaVersion != 1 || report.Kind != "can.build" || len(report.BuildID) != 64 || report.Entry != "entry.ts" {
		t.Fatalf("bad report identity: %s", out)
	}
	for _, digest := range []string{report.Inputs.Source, report.Inputs.Dependencies, report.Inputs.Catalogue, report.Inputs.Compiler, report.Inputs.Runtime, report.Inputs.Options} {
		if len(digest) != 64 {
			t.Fatalf("report omits input identity: %s", out)
		}
	}
	if report.Assertions.Roots != 2 || report.Assertions.Passed != 2 || report.Assertions.Failed != 0 || len(report.Assertions.Evidence) == 0 {
		t.Fatalf("bad assertion totals: %s", out)
	}
	if report.TimeoutMs != 5000 || report.Validation != "verified" {
		t.Fatalf("bad policy/status: %s", out)
	}
	if !strings.Contains(current(passing), report.BuildID) {
		t.Fatal("published current does not select the reported generation")
	}
	// Production ships no assertion roots or runner: the published
	// generation has no assertions tree and its entry never imports the
	// assert runner. Shared context helpers stay inert without a context.
	published := filepath.Join(passing, "dist/builds", report.BuildID)
	if _, err := os.Stat(filepath.Join(published, "assertions")); !os.IsNotExist(err) {
		t.Fatal("production generation ships assertion roots")
	}
	productionEntry, err := os.ReadFile(filepath.Join(published, "entry.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(productionEntry), "assert/runner") {
		t.Fatal("production entry imports the assert runner")
	}
	// The published production artifact executes with the reported output.
	if code, out, diag := runAt(passing, "run"); code != 0 || out != "" || diag != "" {
		t.Fatalf("published run: %d %q %q", code, out, diag)
	}

	// G1 failing project: the same build exits nonzero and publishes
	// nothing for the wrong expected value.
	failing := copyFrozen("failing-assertion")
	code, out, diag = runAt(failing, "build")
	if code == 0 || !strings.Contains(diag, "build verification failed") || !strings.Contains(diag, "wrong") {
		t.Fatalf("wrong assertion published: %d %s %s", code, out, diag)
	}
	if _, err := os.Stat(filepath.Join(failing, "dist/current.json")); !os.IsNotExist(err) {
		t.Fatal("failed build selected production current")
	}

	// A later failing rebuild leaves the prior production selection
	// unchanged, and the leased old generation still runs.
	broken, err := os.ReadFile(filepath.Join(frozen, "failing-assertion/src/main.can"))
	if err != nil {
		t.Fatal(err)
	}
	before := current(passing)
	write(passing, "src/main.can", string(broken))
	code, _, diag = runAt(passing, "build")
	if code == 0 || !strings.Contains(diag, "build verification failed") {
		t.Fatalf("failing rebuild published: %d %s", code, diag)
	}
	if after := current(passing); after != before {
		t.Fatal("failing rebuild moved production current")
	}
	// The old complete generation stays selected and intact: its entry
	// and manifest survive with the recorded manifest identity.
	var pinnedCurrent struct {
		BuildID        string `json:"buildID"`
		ManifestSHA256 string `json:"manifestSHA256"`
	}
	if err := json.Unmarshal([]byte(before), &pinnedCurrent); err != nil {
		t.Fatal(err)
	}
	pinnedDir := filepath.Join(passing, "dist/builds", pinnedCurrent.BuildID)
	manifest, err := os.ReadFile(filepath.Join(pinnedDir, "manifest.json"))
	if err != nil {
		t.Fatalf("pinned manifest lost: %v", err)
	}
	if sum := sha256.Sum256(manifest); hex.EncodeToString(sum[:]) != pinnedCurrent.ManifestSHA256 {
		t.Fatal("pinned manifest identity changed")
	}
	if _, err := os.Stat(filepath.Join(pinnedDir, "entry.ts")); err != nil {
		t.Fatalf("pinned entry lost: %v", err)
	}
	// Restoring the sources runs end to end on the same selection.
	good, err := os.ReadFile(filepath.Join(frozen, "passing/src/main.can"))
	if err != nil {
		t.Fatal(err)
	}
	write(passing, "src/main.can", string(good))
	if code, out, diag := runAt(passing, "run"); code != 0 || out != "" || diag != "" {
		t.Fatalf("restored run: %d %q %q", code, out, diag)
	}
	if after := current(passing); after != before {
		t.Fatal("restored run moved production current")
	}

	// Assert never publishes: passing, selected and failing assert runs
	// all leave current alone, and selection reports partial scope.
	held := copyFrozen("passing")
	if code, _, diag := runAt(held, "build"); code != 0 {
		t.Fatalf("held build: %d %s", code, diag)
	}
	pinned := current(held)
	if code, _, diag := runAt(held, "assert"); code != 0 {
		t.Fatalf("held assert: %d %s", code, diag)
	}
	if after := current(held); after != pinned {
		t.Fatal("passing assert moved production current")
	}
	code, out, _ = runAt(held, "assert", "can.project.root/app", "can.project.root/app::main", "empty")
	if code != 0 {
		t.Fatalf("selected assert: %d %s", code, out)
	}
	var selected map[string]any
	if err := json.Unmarshal([]byte(out), &selected); err != nil || selected["scope"] != "partial" || selected["passed"] != true {
		t.Fatalf("selected run misreports scope: %s", out)
	}
	if after := current(held); after != pinned {
		t.Fatal("selected assert moved production current")
	}
	write(held, "src/main.can", string(broken))
	if code, _, _ := runAt(held, "assert"); code == 0 {
		t.Fatal("failing assert passed")
	}
	if after := current(held); after != pinned {
		t.Fatal("failing assert moved production current")
	}

	// A timed-out root prevents publication without a current selection.
	pending := copyFrozen("pending-assertion")
	code, _, diag = runAt(pending, "build", "--assert-timeout-ms", "1000")
	if code == 0 || !(strings.Contains(diag, "timed out") || strings.Contains(diag, "timeout")) {
		t.Fatalf("timed-out root published: %d %s", code, diag)
	}
	if _, err := os.Stat(filepath.Join(pending, "dist/current.json")); !os.IsNotExist(err) {
		t.Fatal("timed-out build selected production current")
	}

	// Missing required native coverage fails at check with no staging.
	uncovered := copyFrozen("passing")
	write(uncovered, "src/main.can", "package app\n    provides []\n    uses [http, codec]\nrecord receipt\n    int count\nconnection service\n    endpoint \"http://127.0.0.1:1/\"\n    timeout_ms 5000\n    max_body_bytes 8192\nfetch receipt load from service\n    emits [http::request_failed]\n    get \"/load\"\nfn void main\n    emits []\n    given\n        str[] args\n    asserts\n        empty: [] => ok\n    ok\n")
	if code, _, diag := runAt(uncovered, "build"); code == 0 {
		t.Fatal("uncovered native published")
	} else if !strings.Contains(strings.ToLower(diag), "assert") {
		t.Fatalf("coverage failure misattributed: %s", diag)
	}
	if _, err := os.Stat(filepath.Join(uncovered, "dist/current.json")); !os.IsNotExist(err) {
		t.Fatal("uncovered build selected production current")
	}

	// A sticky harness violation (unused fixture row) fails verification.
	sticky := copyFrozen("passing")
	write(sticky, "src/main.can", "package app\n    provides []\n    uses []\nfn int double\n    emits []\n    given\n        int value\n    asserts\n        sample: 2 => ok 4\n    ok value + value\nfixture doubled for double\n    given\n        int base\n    cases\n        base => ok base + base\n        3 => ok 6\nfn int consumer\n    emits []\n    asserts\n        sample: => ok 4\n    match call double(2)\n        when\n            sample: use doubled(2)\n        ok int got => ok got\nfn void main\n    emits []\n    given\n        str[] args\n    asserts\n        empty: [] => ok\n    ok\n")
	if code, _, diag := runAt(sticky, "build"); code == 0 || !strings.Contains(diag, "harness violation") {
		t.Fatalf("sticky violation published: %d %s", code, diag)
	}
	if _, err := os.Stat(filepath.Join(sticky, "dist/current.json")); !os.IsNotExist(err) {
		t.Fatal("sticky build selected production current")
	}

	// A locked local dependency contributes required roots: the full
	// graph verifies before publication, and a wrong vendor assertion
	// blocks the build with current unchanged.
	dep, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	vendorSource := "package helpers\n    provides [double]\n    uses []\nfn int double\n    emits []\n    given\n        int value\n    asserts\n        sample: 2 => ok 4\n        triple: 3 => ok 6\n    ok value + value\n"
	write(dep, "can.project.json", `{"source_root":"src","dependencies":{"vendor":"vendor"},"error_registry":"can.errors.json"}`)
	write(dep, "can.errors.json", `{"active":[],"retired":[]}`)
	write(dep, "src/main.can", "package app\n    provides []\n    uses [vendor::helpers]\nfn int consumer\n    emits []\n    asserts\n        sample: => ok 4\n    relay call helpers::double(2)\nfn void main\n    emits []\n    given\n        str[] args\n    asserts\n        empty: [] => ok\n    ok\n")
	write(dep, "vendor/can.project.json", `{"source_root":"src","error_registry":"can.errors.json"}`)
	write(dep, "vendor/can.errors.json", `{"active":[],"retired":[]}`)
	write(dep, "vendor/src/lib.can", vendorSource)
	lockVendor := func(source string) {
		t.Helper()
		manifestData, err := os.ReadFile(filepath.Join(dep, "vendor/can.project.json"))
		if err != nil {
			t.Fatal(err)
		}
		manifestSum := sha256.Sum256(manifestData)
		lock, err := json.Marshal(map[string]any{
			"edges": map[string]any{"vendor": map[string]any{
				"target": "can.project.dependency/vendor", "path": "vendor"}},
			"projects": map[string]any{"can.project.dependency/vendor": map[string]any{
				"lineage": "", "manifest_sha256": hex.EncodeToString(manifestSum[:]),
				"source_sha256":   captureTreeDigest(t, "can-source-tree-v1", map[string]string{"lib.can": source}),
				"fixtures_sha256": captureTreeDigest(t, "can-fixture-tree-v1", map[string]string{}),
				"error_registry":  map[string]any{"active": []any{}, "retired": []any{}},
				"edges":           map[string]any{}}}})
		if err != nil {
			t.Fatal(err)
		}
		write(dep, "can.lock.json", string(lock))
	}
	lockVendor(vendorSource)
	code, out, diag = runAt(dep, "build")
	if code != 0 || diag != "" {
		t.Fatalf("dependency build: %d %s %s", code, out, diag)
	}
	var depReport struct {
		BuildID    string `json:"buildID"`
		Assertions struct {
			Roots  int `json:"roots"`
			Passed int `json:"passed"`
			Failed int `json:"failed"`
		} `json:"assertions"`
	}
	if err := json.Unmarshal([]byte(out), &depReport); err != nil {
		t.Fatalf("invalid dependency report: %s", out)
	}
	if depReport.Assertions.Roots != 4 || depReport.Assertions.Passed != 4 || depReport.Assertions.Failed != 0 {
		t.Fatalf("dependency roots unverified: %s", out)
	}
	if !strings.Contains(current(dep), depReport.BuildID) {
		t.Fatal("dependency build did not select the reported generation")
	}
	if code, out, diag := runAt(dep, "run"); code != 0 || out != "" || diag != "" {
		t.Fatalf("dependency run: %d %q %q", code, out, diag)
	}
	depBefore := current(dep)
	brokenVendor := strings.Replace(vendorSource, "triple: 3 => ok 6", "triple: 3 => ok 7", 1)
	write(dep, "vendor/src/lib.can", brokenVendor)
	lockVendor(brokenVendor)
	if code, _, diag := runAt(dep, "build"); code == 0 || !strings.Contains(diag, "build verification failed") || !strings.Contains(diag, "double") {
		t.Fatalf("wrong vendor assertion published: %d %s", code, diag)
	}
	if after := current(dep); after != depBefore {
		t.Fatal("wrong vendor assertion moved production current")
	}

	// Build performs no live work: a fetch project builds with zero
	// requests, then the published artifact runs live exactly once.
	var mu sync.Mutex
	loads := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		loads++
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"count":7}`)
	}))
	defer server.Close()
	live := copyFrozen("passing")
	write(live, "src/main.can", "package app\n    provides []\n    uses [http, codec]\nrecord receipt\n    int count\nconnection service\n    endpoint \""+server.URL+"/\"\n    timeout_ms 5000\n    max_body_bytes 8192\nfetch receipt load from service\n    emits [http::request_failed]\n    asserts\n        decoded: => ok receipt(7)\n            using raw \"fixtures/load.json\"\n    get \"/load\"\nfn void main\n    emits [http::request_failed]\n    given\n        str[] args\n    asserts\n        sample: [] => ok\n    match call load()\n        when\n            sample: => ok receipt(7)\n        http::request_failed\n        ok receipt got => ok\n")
	write(live, "src/fixtures/load.json", `{"schema":"can.native-fixture.v1","target":"can.project.root/app::load","environment":{},"exchange":{"request":{"method":"GET","url":"`+server.URL+`/load","headers":[],"body":{"bytes_base64":""}},"outcome":{"response":{"status":200,"headers":[["content-type","application/json"]],"body_base64":"eyJjb3VudCI6N30="}}}}`)
	if code, _, diag := runAt(live, "build"); code != 0 {
		t.Fatalf("live build: %d %s", code, diag)
	}
	mu.Lock()
	if loads != 0 {
		t.Fatalf("build issued %d live requests", loads)
	}
	mu.Unlock()
	if code, _, diag := runAt(live, "run"); code != 0 {
		t.Fatalf("live run: %d %s", code, diag)
	}
	mu.Lock()
	if loads != 1 {
		t.Fatalf("expected one live load, got %d", loads)
	}
	mu.Unlock()
}
