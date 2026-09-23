package main

import "testing"

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
		{"lint"},
		{"normalize"},
		{"lsp", "pos.can"},
		{"assert"},
		{"assert", "--assert-timeout-ms"},
		{"assert", "--assert-timeout-ms", "5000"},
		{"assert", "--assert-timeout-ms", "0", "proj"},
		{"assert", "--assert-timeout-ms", "-5", "proj"},
		{"assert", "--assert-timeout-ms", "abc", "proj"},
		{"assert", "--assert-timeout-ms", "600001", "proj"},
		{"assert", "proj", "only-package"},
		{"build"},
		{"build", "--assert-timeout-ms"},
		{"build", "--assert-timeout-ms", "5000"},
		{"build", "--assert-timeout-ms", "0", "proj"},
		{"build", "--assert-timeout-ms", "abc", "proj"},
		{"build", "--assert-timeout-ms", "600001", "proj"},
		{"build", "proj", "extra"},
	} {
		if code := run(argv); code != 2 {
			t.Fatalf("run(%q) = %d, want usage exit 2", argv, code)
		}
	}
}
