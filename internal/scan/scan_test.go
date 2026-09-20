package scan

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLegacyBraceOutsideString(t *testing.T) {
	cases := []struct {
		line string
		want bool
	}{
		{"no braces", false},
		{"  { stray }", true},
		{"Ok(value = \"{\" + value + \"}\")", false},
		{`x = "{}"`, false},
		{`x = "a\"{b"`, false},
		{"// { stray }", true},
		{`// say "hi" { stray }`, true},
		{`"//" + x`, false},
		{`x = "unterminated {`, false},
		{"} closer", true},
		{"", false},
	}
	for _, c := range cases {
		if got := LegacyBraceOutsideString(c.line); got != c.want {
			t.Errorf("LegacyBraceOutsideString(%q) = %v, want %v", c.line, got, c.want)
		}
	}
}

func TestLegacyHasBraceOutsideString(t *testing.T) {
	if LegacyHasBraceOutsideString("a\nb\nc") {
		t.Fatal("clean body must not trip the ban")
	}
	if !LegacyHasBraceOutsideString("a\nb { x }\nc") {
		t.Fatal("braced line must trip the ban")
	}
}

func TestRepoRoot(t *testing.T) {
	root, err := RepoRoot()
	if err != nil {
		t.Fatalf("RepoRoot inside the repo: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("RepoRoot %q holds no go.mod: %v", root, err)
	}
}
