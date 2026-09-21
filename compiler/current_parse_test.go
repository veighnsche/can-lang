package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCurrentParseNeverEvaluatesBodiesOrAssertions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "inert.can")
	program := `package inert
    provides [run]
    uses [network]
fn int run
    emits []
    asserts
        explosive: => ok (1 / 0)
    int value = call network::must_not_run()
    ok value
`
	if err := os.WriteFile(path, []byte(program), 0600); err != nil {
		t.Fatal(err)
	}
	var out, diagnostics bytes.Buffer
	if code := runCurrentParse(&out, &diagnostics, []string{"--render", path}); code != 0 {
		t.Fatalf("%d: %s", code, &diagnostics)
	}
	if !strings.Contains(out.String(), "network::must_not_run") || !strings.Contains(out.String(), "1 / 0") {
		t.Fatal(out.String())
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatal("parse wrote files")
	}
	before := out.String()
	out.Reset()
	if err := os.WriteFile(path, []byte(before), 0600); err != nil {
		t.Fatal(err)
	}
	if code := runCurrentParse(&out, &diagnostics, []string{"--render", path}); code != 0 || out.String() != before {
		t.Fatalf("render did not round-trip: %d %s", code, &diagnostics)
	}
}

func TestCurrentParseRejectsLegacySyntaxAndHasNoPartialOutput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.can")
	if err := os.WriteFile(path, []byte("rev 1\nextern fetch\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var out, diagnostics bytes.Buffer
	if code := runCurrentParse(&out, &diagnostics, []string{"--render", path}); code != 1 || out.Len() != 0 || !strings.Contains(diagnostics.String(), ":1:1: syntax:") {
		t.Fatalf("%d %s %s", code, &out, &diagnostics)
	}
}
