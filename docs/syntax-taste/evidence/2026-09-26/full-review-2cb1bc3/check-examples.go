//go:build ignore

// Evidence probe script (committed output of the 2026-09-26 full review).
// Excluded from the module build: it imports compiler/internal packages,
// which breaks `go build/vet/test ./...` from the module root.
package main

import (
	"encoding/json"
	"github.com/veighnsche/can-lang/compiler/internal/browser"
	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/project"
	"os"
	"path/filepath"
	"sort"
)

func main() {
	roots, _ := filepath.Glob("examples/*/can.project.json")
	std, _ := filepath.Glob("std/*/current/can.project.json")
	roots = append(roots, std...)
	sort.Strings(roots)
	failed := false
	for _, manifest := range roots {
		dir, _ := filepath.Abs(filepath.Dir(manifest))
		target := check.TargetBun
		if filepath.Base(dir) == "invoice-grid" {
			target = check.TargetBrowser
		}
		row := map[string]any{"project": filepath.Dir(manifest), "target": target}
		graph, err := project.Load(dir)
		if err == nil {
			var program *check.Program
			program, err = check.CheckProgramForTarget(graph, target)
			if err == nil {
				row["functions"] = len(program.Functions)
				row["assertions"] = len(program.Assertions)
				if target == check.TargetBrowser {
					err = browser.CheckProgram(program)
				}
			}
		}
		row["passed"] = err == nil
		if err != nil {
			row["error"] = err.Error()
			failed = true
		}
		_ = json.NewEncoder(os.Stdout).Encode(row)
	}
	if failed {
		os.Exit(1)
	}
}
