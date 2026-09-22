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
	cases := map[string]string{
		"rev/extern":  "rev 1\nextern fetch\n",
		"mod header":  "mod scalars\n    provides [thing]\n",
		"effects":     "package app\n    provides []\n    uses []\nfn int bump\n    effects [total.read]\n    ok 1\n",
		"given table": "package app\n    provides []\n    uses []\nfn int f\n    emits []\n    given\n        int x\n    asserts\n        sample: 1 => ok 1\n    ok x\n    given\n        sample: 1 => ok 1\n",
		"decreases":   "fn int loop\n    decreases n\n    ok 1\n",
	}
	for name, source := range cases {
		path := filepath.Join(t.TempDir(), "bad.can")
		if err := os.WriteFile(path, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
		var out, diagnostics bytes.Buffer
		if code := runCurrentParse(&out, &diagnostics, []string{"--render", path}); code != 1 || out.Len() != 0 || !strings.Contains(diagnostics.String(), "syntax:") {
			t.Fatalf("%s: %d %s %s", name, code, &out, &diagnostics)
		}
	}
}
