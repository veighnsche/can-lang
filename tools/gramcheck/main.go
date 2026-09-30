// Command gramcheck validates the editor assets and pins tokenizer fixtures to
// maintained compiler examples. The actual ordered scopes are checked by
// `bun run test:grammar` in editors/vscode using TextMate and Oniguruma.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var requiredFiles = []string{
	"package.json", "language-configuration.json", "syntaxes/can.tmGrammar.json",
}
var fixtureSamples = []string{
	"package literal_examples", `str path = r"C:\files\notes"`,
	`str escapes = "line\nquote\"unknown\u1234"`, "float ratio = 1.25e-2",
	"fn int square", "    asserts", "        small: 3 => ok 9",
	"owner record", "call ",
}

func loadJSON(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var value map[string]any
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, err
	}
	if value == nil {
		return nil, fmt.Errorf("%s must be a JSON object", path)
	}
	return value, nil
}

func check(extensionDir, corpus string) []string {
	var issues []string
	values := map[string]map[string]any{}
	for _, file := range requiredFiles {
		value, err := loadJSON(filepath.Join(extensionDir, file))
		if err != nil {
			issues = append(issues, fmt.Sprintf("%s: %v", file, err))
			continue
		}
		values[file] = value
	}
	grammar := values["syntaxes/can.tmGrammar.json"]
	if grammar != nil {
		if grammar["scopeName"] != "source.can" {
			issues = append(issues, "grammar scopeName must be source.can")
		}
		repo, ok := grammar["repository"].(map[string]any)
		if !ok {
			issues = append(issues, "grammar repository missing")
		} else {
			for _, key := range []string{"block-comment", "strings", "escapes", "headers", "declarations", "assertions", "numbers", "operators"} {
				if _, ok := repo[key]; !ok {
					issues = append(issues, "grammar repository missing "+key)
				}
			}
		}
	}
	config := values["language-configuration.json"]
	if config != nil {
		if _, ok := config["comments"].(map[string]any); !ok {
			issues = append(issues, "language configuration comments missing")
		}
		if _, ok := config["indentationRules"].(map[string]any); !ok {
			issues = append(issues, "language configuration indentationRules missing")
		}
	}
	manifest := values["package.json"]
	if manifest != nil {
		scripts, ok := manifest["scripts"].(map[string]any)
		if !ok || scripts["test:grammar"] == nil {
			issues = append(issues, "extension test:grammar script missing")
		}
	}
	for _, sample := range fixtureSamples {
		if !strings.Contains(corpus, sample) {
			issues = append(issues, fmt.Sprintf("%q missing from maintained compiler fixtures", sample))
		}
	}
	sort.Strings(issues)
	return issues
}

func loadCorpus(root string) (string, error) {
	var b strings.Builder
	base := filepath.Join(root, "compiler", "testdata", "current")
	err := filepath.WalkDir(base, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".can") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		b.Write(data)
		b.WriteByte('\n')
		return nil
	})
	return b.String(), err
}

func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("repo root (go.mod) not found")
		}
		dir = parent
	}
}

func main() {
	root, err := repoRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	corpus, err := loadCorpus(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if issues := check(filepath.Join(root, "editors", "vscode"), corpus); len(issues) != 0 {
		for _, issue := range issues {
			fmt.Fprintln(os.Stderr, issue)
		}
		os.Exit(1)
	}
	fmt.Println("grammar asset and fixture checks passed; run editors/vscode test:grammar for ordered scopes")
}
