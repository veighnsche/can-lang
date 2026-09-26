// Package failureconventions is the Lane B (failure conventions) extraction
// and acceptance harness for the W3 reusable-infrastructure legs: X-R06-1
// result-data retry/trace conventions and X-R07-1 owner-value factory
// setup. It drives the real canlc CLI as a black box so every leg
// exercises the shipped check/emit/assert path.
//
// Bundle resolution: CONV_BUNDLE names a prebuilt development bundle
// (fast local loop; the manifest is re-verified on every use).
// Otherwise CAN_BUN_ARCHIVE builds through the shared cross-process
// bundle cache also used by tests/integration (identical root and key
// inputs, so entries are shared when both suites run). Without either,
// execution legs skip.
package failureconventions

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/distribution"
)

// assertReport decodes the can.assertion-report object canlc assert prints.
type assertReport struct {
	Passed     bool          `json:"passed"`
	Complete   bool          `json:"complete"`
	Assertions []assertEntry `json:"assertions"`
}

// assertEntry is one root result inside the report.
type assertEntry struct {
	Root struct {
		Package     string `json:"package"`
		Declaration string `json:"declaration"`
		Name        string `json:"name"`
	} `json:"root"`
	Passed     bool     `json:"passed"`
	Reason     string   `json:"reason"`
	Violations []string `json:"violations"`
	Evidence   []string `json:"evidence"`
	Frames     []struct {
		File      string `json:"file"`
		Line      int    `json:"line"`
		Column    int    `json:"column"`
		Operation string `json:"operation"`
	} `json:"frames"`
}

// assertOutcome is the full result of one canlc assert invocation: the
// exit status, the raw streams, and the decoded report when stdout is
// JSON (check failures print plain-text diagnostics instead).
type assertOutcome struct {
	Status int
	Stdout string
	Stderr string
	Report *assertReport
}

func (o assertOutcome) rootID(e assertEntry) string {
	return e.Root.Declaration + ":" + e.Root.Name
}

// resolveBundle returns a verified development bundle directory.
func resolveBundle(t *testing.T, ctx context.Context) string {
	t.Helper()
	if bundle := os.Getenv("CONV_BUNDLE"); bundle != "" {
		if err := verifyBundle(bundle); err != nil {
			t.Fatalf("CONV_BUNDLE %s failed verification: %v", bundle, err)
		}
		return bundle
	}
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CONV_BUNDLE or CAN_BUN_ARCHIVE for execution legs")
	}
	sourceRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	return cachedBundle(t, ctx, sourceRoot, archive)
}

var bundleMu sync.Mutex

// cachedBundle builds the toolchain bundle once per unique input key and
// shares it across processes. The root, key inputs, entry layout, and
// verification match tests/integration exactly so both suites share
// entries; a contaminated entry is detected and rebuilt.
func cachedBundle(t *testing.T, ctx context.Context, sourceRoot, archive string) string {
	t.Helper()
	key := bundleKey(t, sourceRoot, archive)
	root := bundleCacheRoot()
	entry := filepath.Join(root, "bundle-"+key)
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	if entryComplete(entry, key) && verifyBundle(entry) == nil {
		return entry
	}
	lock := entry + ".lock"
	deadline := time.Now().Add(30 * time.Minute)
	for {
		if err := os.Mkdir(lock, 0755); err == nil {
			defer os.RemoveAll(lock)
			if entryComplete(entry, key) && verifyBundle(entry) == nil {
				return entry
			}
			return buildBundleEntry(t, ctx, sourceRoot, archive, root, entry, key)
		}
		if waitForEntry(ctx, entry, key, deadline) {
			return entry
		}
		reapStaleLock(lock)
		if ctx.Err() != nil {
			t.Fatalf("bundle cache fill wait: %v", ctx.Err())
		}
	}
}

func bundleCacheRoot() string {
	if root := os.Getenv("CAN_TEST_CACHE"); root != "" {
		return root
	}
	return filepath.Join(os.TempDir(), "can-test-cache")
}

// bundleKey hashes every input distribution.Build consumes. It must stay
// identical to the tests/integration key or cache entries diverge.
func bundleKey(t *testing.T, sourceRoot, archive string) string {
	t.Helper()
	hashes := map[string]string{}
	for _, dir := range []string{"compiler", "distribution", "runtime", "tools"} {
		err := filepath.WalkDir(filepath.Join(sourceRoot, dir), func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return err
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(sourceRoot, path)
			hashes[rel] = distribution.Hash(data)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"go.mod", "go.sum"} {
		data, err := os.ReadFile(filepath.Join(sourceRoot, name))
		if err != nil {
			t.Fatal(err)
		}
		hashes[name] = distribution.Hash(data)
	}
	archiveData, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	hashes["@archive"] = distribution.Hash(archiveData)
	hashes["@go"] = runtime.Version()
	hashes["@version"] = "harness-shared"
	names := make([]string, 0, len(hashes))
	for name := range hashes {
		names = append(names, name)
	}
	sort.Strings(names)
	var sb strings.Builder
	for _, name := range names {
		sb.WriteString(name)
		sb.WriteString("=")
		sb.WriteString(hashes[name])
		sb.WriteString("\n")
	}
	return distribution.Hash([]byte(sb.String()))
}

func buildBundleEntry(t *testing.T, ctx context.Context, sourceRoot, archive, root, entry, key string) string {
	t.Helper()
	stage, err := os.MkdirTemp(root, ".fill-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(stage)
	built, err := distribution.Build(ctx, sourceRoot, stage, archive, "harness-shared")
	if err != nil {
		t.Fatal(err)
	}
	bundleMu.Lock()
	defer bundleMu.Unlock()
	os.RemoveAll(entry)
	if err := os.Rename(built, entry); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(entry, "cache-complete"), []byte(key+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := verifyBundle(entry); err != nil {
		t.Fatal(err)
	}
	return entry
}

func entryComplete(entry, key string) bool {
	raw, err := os.ReadFile(filepath.Join(entry, "cache-complete"))
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(raw)) == key
}

func waitForEntry(ctx context.Context, entry, key string, deadline time.Time) bool {
	for time.Now().Before(deadline) {
		if entryComplete(entry, key) && verifyBundle(entry) == nil {
			return true
		}
		select {
		case <-ctx.Done():
			return entryComplete(entry, key) && verifyBundle(entry) == nil
		case <-time.After(500 * time.Millisecond):
		}
	}
	return false
}

func reapStaleLock(lock string) {
	info, err := os.Stat(lock)
	if err != nil {
		return
	}
	if time.Since(info.ModTime()) > 30*time.Minute {
		os.RemoveAll(lock)
	}
}

// verifyBundle re-hashes every manifest-listed file in a bundle.
func verifyBundle(entry string) error {
	raw, err := os.ReadFile(filepath.Join(entry, "manifest.json"))
	if err != nil {
		return err
	}
	var manifest distribution.Manifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return err
	}
	if manifest.SchemaVersion != 1 || manifest.Kind != "can.development-distribution" || len(manifest.Files) == 0 {
		return fmt.Errorf("unexpected bundle manifest in %s", entry)
	}
	for name, want := range manifest.Files {
		data, err := os.ReadFile(filepath.Join(entry, filepath.FromSlash(name)))
		if err != nil {
			return err
		}
		if distribution.Hash(data) != want {
			return fmt.Errorf("cached bundle file %s failed verification", name)
		}
	}
	return nil
}

// stageProject copies a fixture project into a resolved temp dir (canlc
// rejects symlinked ancestors) and returns its path. Fixtures are never
// asserted in place: assert writes dist/ into the project.
func stageProject(t *testing.T, name string) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	copyTree(t, name, root)
	return root
}

func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		from := filepath.Join(src, entry.Name())
		to := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := os.MkdirAll(to, 0700); err != nil {
				t.Fatal(err)
			}
			copyTree(t, from, to)
			continue
		}
		data, err := os.ReadFile(from)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(to, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
}

// runAssert runs `canlc assert` on a staged project. On macOS it uses the
// same offline sandbox profile as tests/integration.
func runAssert(t *testing.T, ctx context.Context, bundle, project string, args ...string) assertOutcome {
	t.Helper()
	canlc := filepath.Join(bundle, "bin/canlc")
	argv := append([]string{"assert"}, args...)
	argv = append(argv, project)
	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		full := append([]string{"-p", "(version 1)(allow default)(deny network*)", canlc}, argv...)
		cmd = exec.CommandContext(ctx, "/usr/bin/sandbox-exec", full...)
	} else {
		cmd = exec.CommandContext(ctx, canlc, argv...)
	}
	home := t.TempDir()
	cmd.Dir = home
	cmd.Env = []string{"PATH=/nonexistent", "HOME=" + home}
	var out, diag bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &diag
	result := assertOutcome{}
	if err := cmd.Run(); err != nil {
		var status *exec.ExitError
		if !errors.As(err, &status) {
			t.Fatal(err)
		}
		result.Status = status.ExitCode()
	}
	result.Stdout = out.String()
	result.Stderr = diag.String()
	var report assertReport
	if err := json.Unmarshal(out.Bytes(), &report); err == nil && len(report.Assertions) > 0 {
		result.Report = &report
	}
	return result
}

// requireGreen asserts a full pass and returns the decoded report.
func requireGreen(t *testing.T, outcome assertOutcome) *assertReport {
	t.Helper()
	if outcome.Status != 0 || outcome.Report == nil || !outcome.Report.Passed {
		t.Fatalf("assert failed: status=%d stdout=%s stderr=%s", outcome.Status, outcome.Stdout, outcome.Stderr)
	}
	for _, entry := range outcome.Report.Assertions {
		if !entry.Passed {
			t.Fatalf("root %s failed: %s %v", outcome.rootID(entry), entry.Reason, entry.Violations)
		}
	}
	return outcome.Report
}

// requireRealCan asserts every executed root ran real Can code (oracle
// roots additionally carry supplied-completion evidence by design).
func requireRealCan(t *testing.T, report *assertReport) {
	t.Helper()
	for _, entry := range report.Assertions {
		real := false
		for _, evidence := range entry.Evidence {
			if evidence == "real-can" {
				real = true
			}
		}
		if !real {
			t.Fatalf("root %s ran no real Can: %v", entry.Root.Declaration+":"+entry.Root.Name, entry.Evidence)
		}
	}
}

// rootNames returns sorted declaration:name identities for count pins.
func rootNames(report *assertReport) []string {
	var names []string
	for _, entry := range report.Assertions {
		names = append(names, entry.Root.Declaration+":"+entry.Root.Name)
	}
	sort.Strings(names)
	return names
}

// replaceOnce rewrites one exact substring in a staged file; the edit
// fails unless the old text appears exactly once.
func replaceOnce(t *testing.T, dir, rel, old, new string) {
	t.Helper()
	path := filepath.Join(dir, rel)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(raw), old) != 1 {
		t.Fatalf("%s: want exactly one occurrence of %q", rel, old)
	}
	updated := strings.Replace(string(raw), old, new, 1)
	if err := os.WriteFile(path, []byte(updated), 0600); err != nil {
		t.Fatal(err)
	}
}

// snapshotFiles records every file's bytes under dir.
func snapshotFiles(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if strings.HasPrefix(rel, "dist/") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out[rel] = string(raw)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// requireUnchanged asserts the listed relative paths are byte-identical
// between two snapshots.
func requireUnchanged(t *testing.T, before, after map[string]string, rels ...string) {
	t.Helper()
	for _, rel := range rels {
		if before[rel] != after[rel] {
			t.Fatalf("%s changed by an unrelated edit", rel)
		}
	}
}
