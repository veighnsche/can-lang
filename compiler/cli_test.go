package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersionFlag(t *testing.T) {
	for _, argv := range [][]string{{"--version"}, {"-version"}, {"version"}} {
		if code := run(argv); code != 0 {
			t.Fatalf("run(%q) = %d, want 0", argv, code)
		}
	}
}

func TestParseLSPArgs(t *testing.T) {
	if err := parseLSPArgs(nil); err != nil {
		t.Fatalf("bare lsp: err=%v", err)
	}
	if err := parseLSPArgs([]string{"--stdio"}); err != nil {
		t.Fatalf("--stdio: err=%v", err)
	}
	for _, argv := range [][]string{{"pos.can"}, {"--bogus", "x"}} {
		if err := parseLSPArgs(argv); err == nil {
			t.Fatalf("parseLSPArgs(%q) = nil error, want usage error", argv)
		}
	}
}

func TestRunUsageCodes(t *testing.T) {
	for _, argv := range [][]string{
		{},
		{"--bogus", "x"},
		{"unknown-command"},
		{"lsp", "pos.can"},
		{"assert"},
		{"assert", "--assert-timeout-ms"},
		{"assert", "--assert-timeout-ms", "5000"},
		{"assert", "--assert-timeout-ms", "0", "proj"},
		{"assert", "--assert-timeout-ms", "-5", "proj"},
		{"assert", "--assert-timeout-ms", "abc", "proj"},
		{"assert", "--assert-timeout-ms", "600001", "proj"},
		{"assert", "--assert-jobs"},
		{"assert", "--assert-jobs", "5000"},
		{"assert", "--assert-jobs", "0", "proj"},
		{"assert", "--assert-jobs", "abc", "proj"},
		{"assert", "--assert-jobs", "65", "proj"},
		{"assert", "proj", "only-package"},
		{"build"},
		{"build", "--assert-timeout-ms"},
		{"build", "--assert-timeout-ms", "5000"},
		{"build", "--assert-timeout-ms", "0", "proj"},
		{"build", "--assert-timeout-ms", "abc", "proj"},
		{"build", "--assert-timeout-ms", "600001", "proj"},
		{"build", "--assert-jobs"},
		{"build", "--assert-jobs", "0", "proj"},
		{"build", "--assert-jobs", "abc", "proj"},
		{"build", "--assert-jobs", "65", "proj"},
		{"build", "proj", "extra"},
		{"run"},
		{"run", "--assert-timeout-ms"},
		{"run", "--assert-timeout-ms", "5000"},
		{"run", "--assert-timeout-ms", "0", "proj"},
		{"run", "--assert-timeout-ms", "abc", "proj"},
		{"run", "--assert-timeout-ms", "600001", "proj"},
		{"run", "--assert-jobs"},
		{"run", "--assert-jobs", "0", "proj"},
		{"run", "--assert-jobs", "abc", "proj"},
		{"run", "--assert-jobs", "65", "proj"},
		{"run", "--target", "bun", "proj"},
		{"run", "proj", "extra"},
	} {
		if code := run(argv); code != 2 {
			t.Fatalf("run(%q) = %d, want usage exit 2", argv, code)
		}
	}
}

func TestRunAcceptsAssertionOptionsBeforeProject(t *testing.T) {
	if code := run([]string{"run", "--assert-timeout-ms", "600000", "--assert-jobs", "1", "project"}); code != 1 {
		t.Fatalf("run with assertion options = %d, want bundle-resolution failure after parsing", code)
	}
}

func TestRunTestCheckUsageCodes(t *testing.T) {
	for _, argv := range [][]string{
		{"test"},
		{"test", "proj"},
		{"test", "--candidate", "app", "proj"},
		{"test", "--reference", "spec", "proj"},
		{"test", "--candidate", "app", "--reference", "spec"},
		{"test", "--candidate", "app", "--reference", "spec", "one", "two"},
		{"test", "--candidate", "app", "--reference", "spec", "--schema", "2", "--list", "proj"},
		{"test", "--candidate", "app", "--reference", "spec", "--bogus", "proj"},
		{"test", "--candidate", "app", "--reference", "spec", "--list", "--owner-dir", "x", "proj"},
		{"test", "--candidate"},
		{"check"},
		{"check", "proj"},
		{"check", "--json"},
		{"check", "--json", "one", "two"},
		{"check", "--json", "--schema", "2", "proj"},
		{"check", "--bogus", "proj"},
	} {
		if code := run(argv); code != 2 {
			t.Fatalf("run(%q) = %d, want usage exit 2", argv, code)
		}
	}
}

func writeListFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write := func(name, body string) {
		target := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("can.project.json", "{\"error_registry\":\"can.errors.json\",\"source_root\":\"src\"}")
	write("can.errors.json", "{\"active\":[],\"retired\":[]}")
	write("src/app/main.can", "package app\n    provides [one]\n    uses []\nfn int one\n    emits {}\n    asserts\n        sample: => ok 1\n    ok 1\nfn void main\n    emits {}\n    given\n        str[] args\n    asserts\n        sample: [] => ok\n    ok\n")
	write("src/spec/vectors.can", "package spec\n    provides [two]\n    uses []\nfn int two\n    emits {}\n    asserts\n        sample: => ok 2\n    ok 2\n")
	return root
}

func snapshotTree(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
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

func TestRunTestListIsNonexecuting(t *testing.T) {
	root := writeListFixture(t)
	before := snapshotTree(t, root)
	var stdout, stderr bytes.Buffer
	code := runTest(&stdout, &stderr, []string{"--candidate", "app", "--reference", "spec", "--list", root})
	if code != 0 {
		t.Fatalf("list = %d, stderr %q", code, stderr.String())
	}
	var doc testListDoc
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &doc); err != nil {
		t.Fatalf("list output is not JSON: %v: %q", err, stdout.String())
	}
	if doc.Schema != "1" || doc.Kind != "can.test.list" || doc.Candidate != "app" || doc.Reference != "spec" || doc.Project != root {
		t.Fatalf("list envelope = %+v", doc)
	}
	if len(doc.Roots) != 2 {
		t.Fatalf("roots = %+v, want the 2 app roots only", doc.Roots)
	}
	for _, listed := range doc.Roots {
		if !strings.HasSuffix(listed.Package, "/app") {
			t.Fatalf("listed non-candidate root %+v", listed)
		}
	}
	after := snapshotTree(t, root)
	if len(after) != len(before) {
		t.Fatalf("list touched the tree: %d files before, %d after", len(before), len(after))
	}
	for name, body := range before {
		if after[name] != body {
			t.Fatalf("list rewrote %s", name)
		}
	}
}

func TestRunTestSelectionRefusals(t *testing.T) {
	root := writeListFixture(t)
	for _, argv := range [][]string{
		{"--candidate", "missing", "--reference", "spec", "--list", root},
		{"--candidate", "app", "--reference", "missing", "--list", root},
	} {
		var stdout, stderr bytes.Buffer
		if code := runTest(&stdout, &stderr, argv); code != 1 {
			t.Fatalf("runTest(%q) = %d, want 1", argv, code)
		}
		if stdout.Len() != 0 {
			t.Fatalf("runTest(%q) wrote stdout %q on refusal", argv, stdout.String())
		}
	}
}

func TestRunTestExecutionRefusals(t *testing.T) {
	root := writeListFixture(t)
	toolchain := filepath.Join(root, "canlc-ref")
	if err := os.WriteFile(toolchain, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	owner := t.TempDir()
	// Absent R refuses.
	var stdout, stderr bytes.Buffer
	argv := []string{"--candidate", "app", "--reference", "spec", "--reference-toolchain", filepath.Join(root, "nope"), "--owner-dir", owner, root}
	if code := runTest(&stdout, &stderr, argv); code != 1 {
		t.Fatalf("absent toolchain = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "reference toolchain absent") {
		t.Fatalf("stderr = %q, want toolchain-absent refusal", stderr.String())
	}
	// Absent N refuses.
	stderr.Reset()
	argv = []string{"--candidate", "app", "--reference", "spec", "--reference-toolchain", toolchain, "--owner-dir", filepath.Join(root, "nope"), root}
	if code := runTest(&stdout, &stderr, argv); code != 1 {
		t.Fatalf("absent owner = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "owner absent") {
		t.Fatalf("stderr = %q, want owner-absent refusal", stderr.String())
	}
	// Present R/N still refuses: live execution is P23-gated, never partial.
	stderr.Reset()
	argv = []string{"--candidate", "app", "--reference", "spec", "--reference-toolchain", toolchain, "--owner-dir", owner, root}
	if code := runTest(&stdout, &stderr, argv); code != 1 {
		t.Fatalf("gated execution = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "gated by P23") {
		t.Fatalf("stderr = %q, want P23-gate refusal", stderr.String())
	}
}

func TestRunCheckJSONPure(t *testing.T) {
	root := writeListFixture(t)
	before := snapshotTree(t, root)
	var stdout, stderr bytes.Buffer
	code := runCheck(&stdout, &stderr, []string{"--json", root})
	if code != 0 {
		t.Fatalf("check = %d, stderr %q, stdout %q", code, stderr.String(), stdout.String())
	}
	var doc struct {
		Schema string `json:"schema_version"`
		Kind   string `json:"kind"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &doc); err != nil {
		t.Fatalf("check output is not JSON: %v", err)
	}
	if doc.Schema != "1" || doc.Kind != "can.check" {
		t.Fatalf("check envelope = %+v", doc)
	}
	after := snapshotTree(t, root)
	if len(after) != len(before) {
		t.Fatalf("check touched the tree: %d files before, %d after", len(before), len(after))
	}
}
