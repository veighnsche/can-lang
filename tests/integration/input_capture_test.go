package integration

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// captureTreeDigest independently implements the P2/P15.1 length-framed tree
// format (prefix plus zero, UTF-8 byte path order, u64 big-endian lengths)
// so the lock the test writes cross-checks the compiler's digests instead of
// reusing them.
func captureTreeDigest(t *testing.T, prefix string, files map[string]string) string {
	t.Helper()
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	hash := sha256.New()
	hash.Write([]byte(prefix))
	hash.Write([]byte{0})
	var length [8]byte
	for _, path := range paths {
		binary.BigEndian.PutUint64(length[:], uint64(len(path)))
		hash.Write(length[:])
		hash.Write([]byte(path))
		binary.BigEndian.PutUint64(length[:], uint64(len(files[path])))
		hash.Write(length[:])
		hash.Write([]byte(files[path]))
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func TestCurrentBundledInputCapture(t *testing.T) {
	t.Parallel()
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for input-capture execution")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	bundle, err := harnessBundle(t, ctx, archive)
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	write := func(name, text string) {
		t.Helper()
		p := filepath.Join(root, filepath.FromSlash(name))
		if err = os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(p, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	run := func(args ...string) (int, string, string) {
		t.Helper()
		profile := `(version 1)(allow default)(deny network*)(allow network-outbound (remote ip "localhost:*"))`
		argv := []string{"-p", profile, filepath.Join(bundle, "bin/canlc"), "assert"}
		argv = append(argv, args...)
		argv = append(argv, root)
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

	// The vendor owns a fetch declaration plus a fixture template with raw
	// cases; the root consumes the template. Fixture bytes below are
	// captured, locked, staged and executed end to end.
	write("can.project.json", `{"source_root":"src","dependencies":{"vendor":"vendor"},"error_registry":"can.errors.json"}`)
	write("can.errors.json", `{"active":[],"retired":[]}`)
	write("src/main.can", "package app\n    provides []\n    uses [http, codec, vendor::helpers]\nfn helpers::receipt load_helper\n    emits [http::request_failed]\n    asserts\n        direct: => ok helpers::receipt(1)\n    match call helpers::load()\n        when\n            sample: use helpers::fetched(7)\n            direct: => ok helpers::receipt(1)\n        http::request_failed\n        ok helpers::receipt got => ok got\nfn helpers::receipt load_consumer\n    emits [http::request_failed]\n    asserts\n        sample: => ok helpers::receipt(7)\n    match call load_helper()\n        http::request_failed\n        ok helpers::receipt first => match call load_helper()\n            http::request_failed\n            ok helpers::receipt second => ok second\nfn void main\n    emits []\n    given\n        str[] args\n    asserts\n        empty: [] => ok\n    ok\n")
	write("vendor/can.project.json", `{"source_root":"src","error_registry":"can.errors.json"}`)
	write("vendor/can.errors.json", `{"active":[],"retired":[]}`)
	vendorSource := "package helpers\n    provides [fetched, receipt, load, service]\n    uses [http, codec]\nrecord receipt\n    int count\nconnection service\n    endpoint \"http://127.0.0.1:1/\"\n    timeout_ms 1000\nfetch receipt load from service\n    emits [http::request_failed]\n    asserts\n        decoded: => ok receipt(7)\n            using raw \"fixtures/load.json\"\n    get \"/load\"\nfixture fetched for load\n    given\n        int count\n    cases\n        => ok receipt(count)\n        => ok receipt(7)\n            using raw \"fixtures/fetched.json\"\n"
	write("vendor/src/lib.can", vendorSource)
	fixture := func() string {
		return `{"schema":"can.native-fixture.v1","target":"can.project.dependency/vendor/helpers::load","environment":{},"exchange":{"request":{"method":"GET","url":"http://127.0.0.1:1/load","headers":[],"body":{"bytes_base64":""}},"outcome":{"response":{"status":200,"headers":[["content-type","application/json"]],"body_base64":"eyJjb3VudCI6N30="}}}}`
	}
	write("vendor/src/fixtures/load.json", fixture())
	write("vendor/src/fixtures/fetched.json", fixture())
	manifestData, err := os.ReadFile(filepath.Join(root, "vendor/can.project.json"))
	if err != nil {
		t.Fatal(err)
	}
	manifestSum := sha256.Sum256(manifestData)
	sourceDigest := captureTreeDigest(t, "can-source-tree-v1", map[string]string{"lib.can": vendorSource})
	fixtureDigest := captureTreeDigest(t, "can-fixture-tree-v1", map[string]string{
		"src/fixtures/fetched.json": fixture(),
		"src/fixtures/load.json":    fixture(),
	})
	lock, err := json.Marshal(map[string]any{
		"edges": map[string]any{"vendor": map[string]any{
			"target": "can.project.dependency/vendor", "path": "vendor"}},
		"projects": map[string]any{"can.project.dependency/vendor": map[string]any{
			"lineage": "", "manifest_sha256": hex.EncodeToString(manifestSum[:]),
			"source_sha256": sourceDigest, "fixtures_sha256": fixtureDigest,
			"error_registry": map[string]any{"active": []any{}, "retired": []any{}},
			"edges":          map[string]any{}}}})
	if err != nil {
		t.Fatal(err)
	}
	write("can.lock.json", string(lock))
	locked, err := os.ReadFile(filepath.Join(root, "can.lock.json"))
	if err != nil {
		t.Fatal(err)
	}

	manifests := func() map[string]map[string]any {
		t.Helper()
		out := map[string]map[string]any{}
		entries, err := os.ReadDir(filepath.Join(root, "dist/builds"))
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			data, err := os.ReadFile(filepath.Join(root, "dist/builds", entry.Name(), "manifest.json"))
			if err != nil {
				t.Fatal(err)
			}
			var manifest map[string]any
			if err := json.Unmarshal(data, &manifest); err != nil {
				t.Fatal(err)
			}
			out[entry.Name()] = manifest
		}
		return out
	}
	inputs := func(manifest map[string]any) map[string]any {
		inputs, ok := manifest["inputs"].(map[string]any)
		if !ok {
			t.Fatal("staged manifest omits inputs")
		}
		return inputs
	}

	status, out, diag := run("--assert-timeout-ms", "5000")
	if status != 0 || diag != "" {
		t.Fatalf("captured assertions: %d %s %s", status, out, diag)
	}
	var report map[string]any
	if err := json.Unmarshal([]byte(out), &report); err != nil || report["passed"] != true || len(report["assertions"].([]any)) != 4 {
		t.Fatalf("invalid capture report %v %s", err, out)
	}
	if !strings.Contains(out, "raw-provider-fixture") {
		t.Fatalf("capture report omits raw evidence: %s", out)
	}
	if report["timeoutMs"] != float64(5000) {
		t.Fatalf("capture report omits timeout policy: %s", out)
	}
	first := manifests()
	if len(first) != 1 {
		t.Fatalf("expected one staged generation, got %d", len(first))
	}
	var optionsA, sourceA, depsA string
	for _, manifest := range first {
		optionsA = inputs(manifest)["options"].(string)
		sourceA = inputs(manifest)["source"].(string)
		depsA = inputs(manifest)["dependencies"].(string)
	}

	// The same inputs under a different timeout policy stage a different
	// options identity while source/dependency identities stay fixed.
	status, out, diag = run("--assert-timeout-ms", "6000")
	if status != 0 || diag != "" {
		t.Fatalf("retimed assertions: %d %s %s", status, out, diag)
	}
	second := manifests()
	if len(second) != 2 {
		t.Fatalf("expected two staged generations, got %d", len(second))
	}
	var optionsB string
	for id, manifest := range second {
		if _, ok := first[id]; ok {
			continue
		}
		optionsB = inputs(manifest)["options"].(string)
		if inputs(manifest)["source"].(string) != sourceA || inputs(manifest)["dependencies"].(string) != depsA {
			t.Fatal("timeout-only rerun moved source/dependency identity")
		}
	}
	if optionsA == "" || optionsB == "" || optionsA == optionsB {
		t.Fatal("timeout policy is not bound into staged options")
	}

	// Staged workers observe the captured fixture bytes: the raw response
	// body is embedded in the staged modules.
	marker := "eyJjb3VudCI6N30="
	found := false
	err = filepath.WalkDir(filepath.Join(root, "dist/builds"), func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".ts") {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(data), marker) {
			found = true
		}
		return nil
	})
	if err != nil || !found {
		t.Fatalf("staged modules omit captured fixture bytes: %v", err)
	}

	// Changing only the dependency's raw fixture bytes stales the lock; the
	// failed run leaves the lock file untouched.
	edited := strings.Replace(fixture(), "eyJjb3VudCI6N30=", "eyJjb3VudCI6OH0=", 1)
	write("vendor/src/fixtures/load.json", edited)
	status, _, diag = run("--assert-timeout-ms", "5000")
	if status == 0 || !strings.Contains(diag, "stale dependency digest") {
		t.Fatalf("fixture-only edit escaped the lock: %d %s", status, diag)
	}
	after, err := os.ReadFile(filepath.Join(root, "can.lock.json"))
	if err != nil || string(after) != string(locked) {
		t.Fatal("failed verification rewrote the lock")
	}
}
