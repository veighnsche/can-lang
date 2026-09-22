// Command modcheck is the can-lang module check over the maintained
// fixtures: every file under compiler/testdata/current must carry a
// parseable current-syntax package header, every uses entry must name a
// catalogue package or another fixture package in the tree (mirroring
// resolve.Build's package graph), and retired predecessor shapes must
// not reappear. Predecessor sources under sketches/ and std/ are owned
// by I43/I44 and are out of scope here.
//
// Run from anywhere inside the repo: go run ./tools/modcheck
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

var (
	pkgRe     = regexp.MustCompile(`(?s)package\s+(\w+)\s+provides\s*\[(.*?)\]\s+uses\s*\[(.*?)\]`)
	pinRe     = regexp.MustCompile(`@\d+`)
	modRe     = regexp.MustCompile(`mod\s+\w+\s+provides`)
	externRe  = regexp.MustCompile(`extern\s+fn`)
	externsRe = regexp.MustCompile(`(?m)^\s*externals\s*:`)
	testsRe   = regexp.MustCompile(`(?m)^\s*tests\s*$`)
)

// cataloguePackages loads the built-in catalogue package names from the
// generated JSON so uses resolution tracks the compiler's own graph.
func cataloguePackages(root string) (map[string]bool, error) {
	data, err := os.ReadFile(filepath.Join(root, "compiler", "internal", "catalogue", "catalogue.json"))
	if err != nil {
		return nil, err
	}
	var inventory struct {
		Packages []struct {
			Name string `json:"name"`
		} `json:"packages"`
	}
	if err := json.Unmarshal(data, &inventory); err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, p := range inventory.Packages {
		out[p.Name] = true
	}
	return out, nil
}

func names(raw string) []string {
	var out []string
	for _, p := range strings.Split(raw, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// useEntry splits a uses entry into its package and alias: `beta` is
// both, `beta as other` separates them.
func useEntry(entry string) (pkg, alias string) {
	fields := strings.Fields(entry)
	if len(fields) == 0 {
		return "", ""
	}
	pkg, alias = fields[0], fields[0]
	if len(fields) >= 3 && fields[1] == "as" {
		alias = fields[2]
	}
	return pkg, alias
}

// stripCode removes string literals (plain, triple, and r-prefixed) and
// line comments so retired-shape scans never fire on prose or data. It
// also reports d- and e-prefixed strings, which only exist outside
// literals by construction of the scan.
func stripCode(body string) (string, bool, bool) {
	var sb strings.Builder
	sb.Grow(len(body))
	sawDec, sawInterp := false, false
	i := 0
	for i < len(body) {
		if strings.HasPrefix(body[i:], "//") {
			for i < len(body) && body[i] != '\n' {
				i++
			}
			continue
		}
		rest := body[i:]
		raw := false
		if len(rest) > 1 && rest[0] == 'r' && rest[1] == '"' {
			raw = true
			i++
			rest = body[i:]
		} else if len(rest) > 1 && rest[1] == '"' && (rest[0] == 'd' || rest[0] == 'e') {
			if rest[0] == 'd' {
				sawDec = true
			} else {
				sawInterp = true
			}
			i++
			rest = body[i:]
		}
		if strings.HasPrefix(rest, `"""`) {
			end := strings.Index(rest[3:], `"""`)
			if end < 0 {
				return sb.String(), sawDec, sawInterp
			}
			sb.WriteByte(' ')
			i += 3 + end + 3
			continue
		}
		if strings.HasPrefix(rest, `"`) {
			i++
			for i < len(body) {
				if !raw && body[i] == '\\' {
					i += 2
					continue
				}
				if body[i] == '"' {
					i++
					break
				}
				if body[i] == '\n' {
					break
				}
				i++
			}
			sb.WriteByte(' ')
			continue
		}
		sb.WriteByte(body[i])
		i++
	}
	return sb.String(), sawDec, sawInterp
}

// check scans roots recursively and returns the file count plus every
// violation. Pure filesystem in, strings out — see main_test.go.
func check(roots []string, catalogue map[string]bool) (scanned int, errs []string) {
	var files []string
	for _, root := range roots {
		_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if !entry.IsDir() && strings.HasSuffix(path, ".can") {
				files = append(files, path)
			}
			return nil
		})
	}
	sort.Strings(files)
	anchor := roots[0]
	keyOf := func(path string) string {
		if rel, err := filepath.Rel(anchor, path); err == nil {
			return rel
		}
		return filepath.Base(path)
	}
	packages := map[string]string{}
	uses := map[string][][2]string{}
	for _, path := range files {
		base := keyOf(path)
		text, err := os.ReadFile(path)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", base, err))
			continue
		}
		body := string(text)
		m := pkgRe.FindStringSubmatch(body)
		if m == nil {
			errs = append(errs, fmt.Sprintf("%s: no parseable package header", base))
			continue
		}
		packages[m[1]] = base
		seen := map[string]bool{}
		for _, name := range names(m[2]) {
			if seen[name] {
				errs = append(errs, fmt.Sprintf("%s: duplicate provides entry %q", base, name))
			}
			seen[name] = true
		}
		seenPkg, seenAlias := map[string]bool{}, map[string]bool{}
		for _, entry := range names(m[3]) {
			pkg, alias := useEntry(entry)
			if pkg == "" {
				continue
			}
			if seenPkg[pkg] {
				errs = append(errs, fmt.Sprintf("%s: duplicate uses entry %q", base, pkg))
			}
			seenPkg[pkg] = true
			if seenAlias[alias] {
				errs = append(errs, fmt.Sprintf("%s: import alias %q collides with another name", base, alias))
			}
			seenAlias[alias] = true
			uses[base] = append(uses[base], [2]string{pkg, alias})
		}
		code, sawDec, sawInterp := stripCode(body)
		switch {
		case modRe.MatchString(code):
			errs = append(errs, fmt.Sprintf("%s: predecessor mod header is gone, use package", base))
		case externRe.MatchString(code):
			errs = append(errs, fmt.Sprintf("%s: inline extern fn is gone", base))
		case externsRe.MatchString(code):
			errs = append(errs, fmt.Sprintf("%s: externals section is gone", base))
		case testsRe.MatchString(code):
			errs = append(errs, fmt.Sprintf("%s: tests section is gone, use per-function asserts", base))
		case sawDec:
			errs = append(errs, fmt.Sprintf("%s: dec literal is gone", base))
		case sawInterp:
			errs = append(errs, fmt.Sprintf("%s: interpreted e-string is gone", base))
		case pinRe.MatchString(code):
			errs = append(errs, fmt.Sprintf("%s: @N revision pin is gone", base))
		}
		if scan.LegacyHasBraceOutsideString(body) {
			errs = append(errs, fmt.Sprintf("%s: curly braces are banned, use () records", base))
		}
	}
	for file, deps := range uses {
		for _, dep := range deps {
			if catalogue[dep[0]] {
				continue
			}
			if _, ok := packages[dep[0]]; ok {
				continue
			}
			errs = append(errs, fmt.Sprintf("%s: uses %s resolves nowhere", file, dep[0]))
		}
	}
	sort.Strings(errs)
	return len(files), errs
}

func main() {
	root, err := scan.RepoRoot()
	if err != nil {
		fmt.Println("MODULE CHECK FAILED")
		fmt.Println(" -", err)
		os.Exit(1)
	}
	catalogue, err := cataloguePackages(root)
	if err != nil {
		fmt.Println("MODULE CHECK FAILED")
		fmt.Println(" -", err)
		os.Exit(1)
	}
	scanned, errs := check([]string{filepath.Join(root, "compiler", "testdata", "current")}, catalogue)
	if len(errs) > 0 {
		fmt.Println("MODULE CHECK FAILED")
		for _, e := range errs {
			fmt.Println(" -", e)
		}
		os.Exit(1)
	}
	fmt.Printf("modules OK: %d current fixtures, uses resolve to catalogue or tree packages\n", scanned)
}
