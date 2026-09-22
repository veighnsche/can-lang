package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoPredecessorPaths is the I44 static retirement gate: the
// launcher package holds exactly the surviving production files,
// imports only the standard library and current internal packages,
// and defines or references none of the retired parser, checker,
// evaluator, prover, Z3, extern-grant, or baseline symbols. Behavior
// probes (legacy-syntax rejection, retired-command exit codes) cover
// the runtime side; this test pins the source side.
func TestNoPredecessorPaths(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	production := map[string]bool{
		"current_parse.go":   true,
		"current_project.go": true,
		"current_types.go":   true,
		"lsp.go":             true,
		"main.go":            true,
	}
	for _, file := range files {
		if !strings.HasSuffix(file, "_test.go") && !production[file] {
			t.Errorf("unexpected production file %s in retired launcher package", file)
		}
	}
	// Retired entry points, engines, and grants. Each name below died
	// with its owning file or command; none may be defined or
	// referenced by surviving production sources.
	legacy := []string{
		"legacyParsePaths", "legacyParse", "parseModuleText", "Module", "FnDecl",
		"checkProgram", "checkSem", "checkStatic", "runTestValue", "runLinkedPure",
		"expandGenerics", "VerifyProve", "VerifyContracts", "smtBinary", "CANLC_Z3",
		"WriteBaseline", "LoadBaseline", "FingerprintProgram", "RevisionBaseline",
		"CheckRevisionIdentity", "CheckPinnedRows", "PinnedRows",
		"runLint", "runExplain", "runNormalize", "runBaseline",
		"parseCompileArgs", "parseBaselineArgs", "diagnoseWith", "explainDocs",
		"Diag", "spanDiag", "proofDiag",
	}
	for _, file := range files {
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		node, err := parser.ParseFile(token.NewFileSet(), file, src, 0)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(file, "_test.go") {
			for _, decl := range node.Imports {
				path := strings.Trim(decl.Path.Value, `"`)
				if !strings.Contains(path, ".") {
					continue
				}
				if !strings.HasPrefix(path, "github.com/veighnsche/can-lang/compiler/internal/") {
					t.Errorf("%s imports non-internal %s", file, path)
				}
			}
		}
		idents := map[string]bool{}
		ast.Inspect(node, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.SelectorExpr:
				ast.Inspect(n.X, func(x ast.Node) bool {
					if id, ok := x.(*ast.Ident); ok {
						idents[id.Name] = true
					}
					return true
				})
				return false
			case *ast.Ident:
				idents[n.Name] = true
			}
			return true
		})
		for _, name := range legacy {
			if idents[name] {
				t.Errorf("%s references retired symbol %s", file, name)
			}
		}
	}
}
