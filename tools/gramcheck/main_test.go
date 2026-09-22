package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/internal/scan"
)

func fullCorpus() string {
	var sb strings.Builder
	for _, c := range cases {
		for _, s := range c.samples {
			sb.WriteString(s)
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

func TestRepoGrammar(t *testing.T) {
	root, err := scan.RepoRoot()
	if err != nil {
		t.Skip("not in repo checkout")
	}
	corpus, err := loadCorpus(root)
	if err != nil {
		t.Fatal(err)
	}
	if errs := check(filepath.Join(root, "editors", "vscode"), corpus); len(errs) > 0 {
		t.Fatalf("repo grammar failed: %v", errs)
	}
}

func writeGrammar(t *testing.T, patterns string) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"package.json":                `{"name": "x"}`,
		"language-configuration.json": `{}`,
		"syntaxes/can.tmGrammar.json": `{"scopeName": "source.can", "patterns": [` + patterns + `]}`,
	}
	for name, body := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

const errorRule = `{"match": "\\berror\\b", "name": "keyword.declaration.error.can"}`
const keywordRule = `{"match": "\\b(fn|on|rev)\\b", "name": "keyword.control.can"}`

func TestMissingErrorRule(t *testing.T) {
	dir := writeGrammar(t, keywordRule)
	if errs := check(dir, fullCorpus()); !contains(errs, "exactly one error rule") {
		t.Fatalf("expected error-rule violation, got %v", errs)
	}
}

func TestRetiredKeywordRejected(t *testing.T) {
	dir := writeGrammar(t, errorRule+","+keywordRule)
	if errs := check(dir, fullCorpus()); !contains(errs, "retired rev must not be a keyword") {
		t.Fatalf("expected retired-keyword violation, got %v", errs)
	}
}

func TestRetiredScopeRejected(t *testing.T) {
	pin := `{"match": "@[0-9]+", "name": "constant.numeric.version.can"}`
	dir := writeGrammar(t, errorRule+","+pin)
	if errs := check(dir, fullCorpus()); !contains(errs, "retired scope constant.numeric.version.can must go") {
		t.Fatalf("expected retired-scope violation, got %v", errs)
	}
}

func TestSamplePinnedToFixtures(t *testing.T) {
	types := `{"match": "\\b(str)\\b", "name": "storage.type.primitive.can"}`
	dir := writeGrammar(t, errorRule+","+types)
	if errs := check(dir, "unrelated corpus"); !contains(errs, "not in the maintained fixtures") {
		t.Fatalf("expected fixture-pinning violation, got %v", errs)
	}
}

func TestBadJSON(t *testing.T) {
	dir := writeGrammar(t, keywordRule)
	bad := filepath.Join(dir, "syntaxes", "can.tmGrammar.json")
	if err := os.WriteFile(bad, []byte(`{not json`), 0o644); err != nil {
		t.Fatal(err)
	}
	if errs := check(dir, fullCorpus()); len(errs) == 0 {
		t.Fatal("expected JSON violation, got none")
	}
}

func contains(errs []string, sub string) bool {
	for _, e := range errs {
		if strings.Contains(e, sub) {
			return true
		}
	}
	return false
}
