package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Retrospective row 0: raw control bytes are expressible in string
// literals (only newline breaks line syntax), evaluate faithfully,
// and emit byte-exact. This fixture preserves the falsification of
// the old "controls are inexpressible" premise: control reject-arms
// ARE witnessable, so encoder totality cannot rest on necessity.
// Code points are explicit Go escapes: NUL, SOH, DEL.
func TestControlBytesRoundTrip(t *testing.T) {
	src := "mod probe\n" +
		"  provides [probe__go, Probe__Out]\n" +
		"  uses []\n" +
		"  emits []\n" +
		"\n" +
		"type Probe__Out rev 1 (\n" +
		"  value: str\n" +
		")\n" +
		"\n" +
		"fn probe__go() -> Probe__Out rev 1\n" +
		"  emits []\n" +
		"  tests\n" +
		"    go() => Ok(\"a\x00b\x01c\x7fd\")\n" +
		"  Ok(\"a\x00b\x01c\x7fd\")\n"
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "probe.can")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out")
	if err := compileEx(out, []string{srcPath}, true); err != nil {
		t.Fatalf("control literals must parse, evaluate, and emit: %v", err)
	}
	ts, err := os.ReadFile(filepath.Join(out, "probe.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(ts), "\x00") {
		t.Fatalf("emitted TS must not contain a raw NUL byte")
	}
	if !strings.Contains(string(ts), `\u0000`) {
		t.Fatalf("emitted TS must escape NUL as \\u0000, got:\n%s", ts)
	}
}

// Row 4: malformed UTF-8 is refused at parse on every route into
// parseModuleText (CLI files and editor texts share the choke
// point). Only 0x0A is excluded from the expressible set, by line
// structure — every other probe above remains valid UTF-8.
func TestMalformedSourceRefused(t *testing.T) {
	bad := "mod probe\n  provides [probe__go]\n  uses []\n  emits []\n\nfn probe__go() -> Probe__Out rev 1\n  emits []\n  tests\n    go() => Ok(\"a\xff\xfeb\")\n  Ok(\"a\xff\xfeb\")\n"
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "probe.can")
	if err := os.WriteFile(srcPath, []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, collected, err := legacyParsePaths([]string{srcPath})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, d := range collected {
		if d.Sev == "error" && d.Code == CodeParse && strings.Contains(d.Msg, "not valid UTF-8") {
			found = true
		}
	}
	if !found {
		t.Fatalf("CLI path must refuse malformed source, got %v", collected)
	}
	wdir := writeLSPDir(t, map[string]string{"probe.can": bad})
	diags := diagnose(wdir, "probe.can", bad)
	if !hasDiag(diags, "error", "not valid UTF-8") {
		t.Fatalf("editor path must refuse malformed source, got %v", diags)
	}
}
