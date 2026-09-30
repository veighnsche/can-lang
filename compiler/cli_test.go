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
