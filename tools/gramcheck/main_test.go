package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fullCorpus() string { return strings.Join(fixtureSamples, "\n") }

func writeAssets(t *testing.T, grammar string) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"package.json":                `{"scripts":{"test:grammar":"node --test"}}`,
		"language-configuration.json": `{"comments":{},"indentationRules":{}}`,
		"syntaxes/can.tmGrammar.json": grammar,
	}
	for name, body := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

const validGrammar = `{"scopeName":"source.can","repository":{"block-comment":{},"strings":{},"escapes":{},"headers":{},"declarations":{},"assertions":{},"numbers":{},"operators":{}}}`

func TestRepoGrammar(t *testing.T) {
	root, err := repoRoot()
	if err != nil {
		t.Fatal(err)
	}
	corpus, err := loadCorpus(root)
	if err != nil {
		t.Fatal(err)
	}
	if issues := check(filepath.Join(root, "editors", "vscode"), corpus); len(issues) != 0 {
		t.Fatal(issues)
	}
}

func TestInvalidJSONAndMissingRepository(t *testing.T) {
	dir := writeAssets(t, `{bad`)
	if issues := check(dir, fullCorpus()); len(issues) == 0 {
		t.Fatal("invalid JSON accepted")
	}
	dir = writeAssets(t, `{"scopeName":"source.can"}`)
	if issues := check(dir, fullCorpus()); len(issues) == 0 {
		t.Fatal("missing repository accepted")
	}
}

func TestFixtureDrift(t *testing.T) {
	dir := writeAssets(t, validGrammar)
	if issues := check(dir, "unrelated"); len(issues) == 0 {
		t.Fatal("missing compiler fixture samples accepted")
	}
	if issues := check(dir, fullCorpus()); len(issues) != 0 {
		t.Fatal(issues)
	}
}
