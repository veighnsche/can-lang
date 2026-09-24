// Command gramcheck verifies the can-lang TextMate grammar against the
// current syntax and the maintained fixtures: every scope's regex must
// fire on a representative sample, and every sample must occur verbatim
// in compiler/testdata/current so invented goldens cannot drift from the
// language. Smoke test only — most rule precedence (first-match-wins in
// TextMate) is still kept by hand, except the string rules, whose order
// is pinned below: raw triple before triple before raw before plain, so
// an r prefix or triple delimiter never lexes as a shorter string start.
//
// Run from anywhere inside the repo: go run ./tools/gramcheck
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/veighnsche/can-lang/internal/scan"
)

var jsonFiles = []string{
	"package.json",
	"language-configuration.json",
	"syntaxes/can.tmGrammar.json",
}

type sampleCase struct {
	scope   string
	samples []string
}

var cases = []sampleCase{
	{"keyword.control.can", []string{"fn void main", "record", "variant failure", "given", "asserts", "call", "callable", "match value", "chain", "do", "ok", "relay", " on ", " with ", " and ", " or ", " not ", " is ", " as ", "near", "when", "concurrent", "race"}},
	{"keyword.declaration.error.can", []string{"error unavailable(str reason)"}},
	{"entity.name.namespace.can", []string{"package offline", "package lexical"}},
	{"storage.type.primitive.can", []string{"str path", "int value", "float ratio", "bool", "void"}},
	{"storage.type.generic.can", []string{"<int>", "<account_row>", "record box<item>"}},
	{"punctuation.definition.generic.can", []string{"<int>"}},
	{"constant.numeric.can", []string{"42", "1.5", "1.25e-2"}},
	{"constant.language.can", []string{"true", "false", "_ => ok"}},
	{"keyword.operator.can", []string{"=>", ">=", "+", "-", "*", "/", "%", "|"}},
	{"entity.name.package.can", []string{"http::must_not_run", "option::some"}},
	{"constant.other.scoped-name.can", []string{"http::must_not_run", "codec::invalid_data"}},
	{"entity.name.tag.can", []string{"explosive:", "sample:", "private_name:"}},
	{"variable.other.readwrite.can", []string{"value", "seed", "document"}},
	{"string.quoted.double.can", []string{`"😀"`, `"https://invalid.example/"`}},
	{"string.quoted.double.raw.can", []string{`r"C:`}},
	{"string.quoted.triple.can", []string{`"""`}},
	{"string.quoted.triple.raw.can", []string{`r"""`}},
	{"comment.line.double-slash.can", []string{
		"/// A multiline literal keeps its indentation and boundary newlines.",
		"// Project inspection is inert: this initializer must not execute.",
	}},
}

// requiredKeywords is every hard lexer keyword that is neither a header
// word (package/provides/uses/emits), the error declarator, a literal
// (true/false), nor a primitive type: the control rule must match each
// one as a bare word.
var requiredKeywords = []string{
	"fn", "record", "variant", "given", "near", "asserts",
	"call", "callable", "match", "chain", "do", "ok", "relay",
	"on", "with", "and", "or", "not", "is", "as",
	"when", "concurrent", "race",
}

// retiredWords must not match the control rule: predecessor spellings
// highlight as plain identifiers so dead syntax never looks alive.
var retiredWords = []string{
	"extern", "rev", "mod", "dec", "brand", "const",
	"forward", "fnref", "invoke", "seal", "tests", "case",
	"then", "proof", "decreases", "requires", "ensures",
}

// retiredScopes must not exist anywhere in the grammar: version pins,
// decimal literals, double-underscore identities, and the unreachable
// row marker all died with the predecessor syntax.
var retiredScopes = []string{
	"constant.numeric.version.can",
	"constant.numeric.decimal.can",
	"constant.numeric.integer.can",
	"entity.name.function.can",
	"entity.name.type.can",
	"constant.language.unreachable.can",
	"constant.numeric.error.can",
}

func loadJSON(dir, name string) (map[string]any, error) {
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return nil, err
	}
	var v map[string]any
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("%s: %v", name, err)
	}
	return v, nil
}

func patternsOf(grammar map[string]any) []map[string]any {
	var out []map[string]any
	if list, ok := grammar["patterns"].([]any); ok {
		for _, p := range list {
			if m, ok := p.(map[string]any); ok {
				out = append(out, m)
			}
		}
	}
	return out
}

func strOf(v any) string {
	s, _ := v.(string)
	return s
}

// check verifies the grammar in dir against the fixture corpus and
// returns every violation found.
func check(dir, corpus string) []string {
	var errs []string
	for _, name := range jsonFiles {
		if _, err := loadJSON(dir, name); err != nil {
			errs = append(errs, fmt.Sprintf("%v", err))
		}
	}
	grammar, err := loadJSON(dir, "syntaxes/can.tmGrammar.json")
	if err != nil {
		return errs
	}
	patterns := patternsOf(grammar)

	var errorRules []map[string]any
	for _, p := range patterns {
		if strOf(p["name"]) == "keyword.declaration.error.can" {
			errorRules = append(errorRules, p)
		}
	}
	if len(errorRules) != 1 {
		errs = append(errs, "expected exactly one error rule")
	} else if !strings.Contains(strOf(errorRules[0]["match"]), "error") {
		errs = append(errs, "error rule must match the error keyword")
	}
	for _, p := range patterns {
		if strOf(p["name"]) == "keyword.control.can" &&
			strings.Contains(strOf(p["match"]), "error") {
			errs = append(errs, "generic keyword rule must not also match error")
		}
	}
	var keywordMatches []string
	for _, p := range patterns {
		if strOf(p["name"]) == "keyword.control.can" {
			keywordMatches = append(keywordMatches, strOf(p["match"]))
		}
	}
	matchesKeyword := func(word string) bool {
		for _, m := range keywordMatches {
			if ok, _ := regexp.MatchString(m, word); ok {
				return true
			}
		}
		return false
	}
	for _, word := range requiredKeywords {
		if !matchesKeyword(word) {
			errs = append(errs, fmt.Sprintf("%s must be a keyword", word))
		}
	}
	for _, word := range retiredWords {
		if matchesKeyword(word) {
			errs = append(errs, fmt.Sprintf("retired %s must not be a keyword", word))
		}
	}
	seen := map[string]bool{}
	var walk func(v any)
	walk = func(v any) {
		switch v := v.(type) {
		case map[string]any:
			for key, val := range v {
				if key == "name" {
					if s, ok := val.(string); ok {
						seen[s] = true
					}
				} else {
					walk(val)
				}
			}
		case []any:
			for _, e := range v {
				walk(e)
			}
		}
	}
	walk(grammar["patterns"])
	for _, scope := range retiredScopes {
		if seen[scope] {
			errs = append(errs, fmt.Sprintf("retired scope %s must go", scope))
		}
	}
	// String-rule order: raw triple, triple, raw, plain. Each begin
	// pattern is a prefix of a longer form's, so a later longer rule
	// would never fire.
	wantOrder := []struct {
		scope string
		begin string
	}{
		{"string.quoted.triple.raw.can", `r"""`},
		{"string.quoted.triple.can", `"""`},
		{"string.quoted.double.raw.can", `r"`},
		{"string.quoted.double.can", `"`},
	}
	positions := map[string]int{}
	for i, p := range patterns {
		if name := strOf(p["name"]); name != "" {
			if _, ok := positions[name]; !ok {
				positions[name] = i
			}
		}
	}
	for _, want := range wantOrder {
		pos, ok := positions[want.scope]
		if !ok {
			errs = append(errs, fmt.Sprintf("expected one %s rule", want.scope))
			continue
		}
		if strOf(patterns[pos]["begin"]) != want.begin {
			errs = append(errs, fmt.Sprintf("%s rule has the wrong delimiter", want.scope))
		}
	}
	for i := 1; i < len(wantOrder); i++ {
		prev, okPrev := positions[wantOrder[i-1].scope]
		cur, okCur := positions[wantOrder[i].scope]
		if okPrev && okCur && prev > cur {
			errs = append(errs, fmt.Sprintf("%s rule must precede the %s rule", wantOrder[i-1].scope, wantOrder[i].scope))
		}
	}

	scopes := map[string][]string{}
	for _, p := range patterns {
		match := strOf(p["match"])
		if match == "" {
			match = strOf(p["begin"])
		}
		if match == "" {
			continue
		}
		var names []string
		if n := strOf(p["name"]); n != "" {
			names = append(names, n)
		}
		if caps, ok := p["captures"].(map[string]any); ok {
			for _, c := range caps {
				if cm, ok := c.(map[string]any); ok {
					if n := strOf(cm["name"]); n != "" {
						names = append(names, n)
					}
				}
			}
		}
		for _, n := range names {
			scopes[n] = append(scopes[n], match)
		}
	}
	for _, c := range cases {
		rules := scopes[c.scope]
		if len(rules) == 0 {
			errs = append(errs, fmt.Sprintf("no rule for %s", c.scope))
			continue
		}
		for _, sample := range c.samples {
			if !strings.Contains(corpus, sample) {
				errs = append(errs, fmt.Sprintf("%q is not in the maintained fixtures", sample))
				continue
			}
			hit := false
			for _, m := range rules {
				if ok, _ := regexp.MatchString(m, sample); ok {
					hit = true
					break
				}
			}
			if !hit {
				errs = append(errs, fmt.Sprintf("%q matches nothing under %s", sample, c.scope))
			}
		}
	}

	var blocks []map[string]any
	for _, p := range patterns {
		if _, ok := p["begin"]; ok && strings.Contains(strOf(p["begin"]), "given") {
			blocks = append(blocks, p)
		}
	}
	if len(blocks) != 1 {
		errs = append(errs, "expected one given/asserts block rule")
	} else {
		if !strings.Contains(strOf(blocks[0]["begin"]), "asserts") {
			errs = append(errs, "section-block rule must cover asserts as well as given")
		}
		var inner []string
		if list, ok := blocks[0]["patterns"].([]any); ok {
			for _, q := range list {
				if qm, ok := q.(map[string]any); ok {
					if caps, ok := qm["captures"].(map[string]any); ok {
						for _, c := range caps {
							if cm, ok := c.(map[string]any); ok {
								inner = append(inner, strOf(cm["name"]))
							}
						}
					}
				}
			}
		}
		found := false
		for _, v := range inner {
			if v == "entity.name.tag.can" {
				found = true
			}
		}
		if !found {
			errs = append(errs, "section-block must scope assertion-case keys")
		}
	}
	sort.Strings(errs)
	return errs
}

func loadCorpus(root string) (string, error) {
	var sb strings.Builder
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
		sb.Write(data)
		sb.WriteByte('\n')
		return nil
	})
	if err != nil {
		return "", err
	}
	return sb.String(), nil
}

func main() {
	root, err := scan.RepoRoot()
	if err != nil {
		fmt.Println("GRAMMAR CHECK FAILED")
		fmt.Println(" -", err)
		os.Exit(1)
	}
	corpus, err := loadCorpus(root)
	if err != nil {
		fmt.Println("GRAMMAR CHECK FAILED")
		fmt.Println(" -", err)
		os.Exit(1)
	}
	if errs := check(filepath.Join(root, "editors", "vscode"), corpus); len(errs) > 0 {
		fmt.Println("GRAMMAR CHECK FAILED")
		for _, e := range errs {
			fmt.Println(" -", e)
		}
		os.Exit(1)
	}
	fmt.Println("grammar OK: current syntax highlighted, samples pinned to maintained fixtures")
}
