package main

import "testing"

func TestVersionFlag(t *testing.T) {
	for _, argv := range [][]string{{"--version"}, {"-version"}, {"version"}} {
		if code := run(argv); code != 0 {
			t.Fatalf("run(%q) = %d, want 0", argv, code)
		}
	}
}

func TestParseCompileArgs(t *testing.T) {
	out, format, baseline, args, err := parseCompileArgs([]string{
		"--format", "json", "--baseline", "base.json", "--out", "outdir", "a.can", "b.can",
	})
	if err != nil {
		t.Fatalf("parseCompileArgs: %v", err)
	}
	if out != "outdir" || format != "json" || baseline != "base.json" {
		t.Fatalf("flags = %q %q %q, want outdir json base.json", out, format, baseline)
	}
	if len(args) != 2 || args[0] != "a.can" || args[1] != "b.can" {
		t.Fatalf("args = %q, want [a.can b.can]", args)
	}
	// Single-dash spellings work too.
	out, _, _, args, err = parseCompileArgs([]string{"-out", "outdir", "a.can"})
	if err != nil || out != "outdir" || len(args) != 1 {
		t.Fatalf("single-dash: out=%q args=%q err=%v", out, args, err)
	}
	// Standard flag behavior: parsing stops at the first positional, so
	// a flag after a file is a file, leaving --out empty for the caller
	// to report as usage. (The old hand parser accepted any order; all
	// documented invocations already put flags first.)
	out, _, _, args, err = parseCompileArgs([]string{"a.can", "--out", "outdir"})
	if err != nil || out != "" {
		t.Fatalf("flag-after-file: out=%q args=%q err=%v, want out empty", out, args, err)
	}
	for _, argv := range [][]string{
		{"--bogus", "x"},
		{"--out"},    // missing value
		{"--format"}, // missing value
	} {
		if _, _, _, _, err := parseCompileArgs(argv); err == nil {
			t.Fatalf("parseCompileArgs(%q) = nil error, want usage error", argv)
		}
	}
}

func TestParseBaselineArgs(t *testing.T) {
	out, origin, args, err := parseBaselineArgs([]string{
		"--out", "base.json", "--origin", "r1", "a.can",
	})
	if err != nil || out != "base.json" || origin != "r1" || len(args) != 1 {
		t.Fatalf("out=%q origin=%q args=%q err=%v", out, origin, args, err)
	}
	if _, _, _, err := parseBaselineArgs([]string{"a.can"}); err != nil {
		t.Fatalf("missing --out must parse (caller reports usage), got %v", err)
	}
	if _, _, _, err := parseBaselineArgs([]string{"--bogus", "x"}); err == nil {
		t.Fatal("unknown flag must be a usage error")
	}
}

func TestParseLSPArgs(t *testing.T) {
	if err := parseLSPArgs(nil); err != nil {
		t.Fatalf("bare lsp: err=%v", err)
	}
	if err := parseLSPArgs([]string{"--stdio"}); err != nil {
		t.Fatalf("--stdio: err=%v", err)
	}
	for _, argv := range [][]string{{"pos.can"}, {"--bogus", "x"}, {"--baseline", "b.json"}, {"--baseline"}} {
		if err := parseLSPArgs(argv); err == nil {
			t.Fatalf("parseLSPArgs(%q) = nil error, want usage error", argv)
		}
	}
}

func TestRunUsageCodes(t *testing.T) {
	for _, argv := range [][]string{
		{},
		{"--bogus", "x"},
		{"baseline"},
		{"explain"},
		{"explain", "a", "b"},
		{"lsp", "pos.can"},
	} {
		if code := run(argv); code != 2 {
			t.Fatalf("run(%q) = %d, want usage exit 2", argv, code)
		}
	}
}
