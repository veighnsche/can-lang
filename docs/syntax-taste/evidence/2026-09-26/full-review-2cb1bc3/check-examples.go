package main

import (
 "encoding/json"
 "os"
 "path/filepath"
 "sort"
 "github.com/veighnsche/can-lang/compiler/internal/project"
 "github.com/veighnsche/can-lang/compiler/internal/check"
 "github.com/veighnsche/can-lang/compiler/internal/browser"
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
  if filepath.Base(dir) == "invoice-grid" { target = check.TargetBrowser }
  row := map[string]any{"project":filepath.Dir(manifest), "target":target}
  graph, err := project.Load(dir)
  if err == nil {
   var program *check.Program
   program, err = check.CheckProgramForTarget(graph, target)
   if err == nil {
    row["functions"] = len(program.Functions)
    row["assertions"] = len(program.Assertions)
    if target == check.TargetBrowser { err = browser.CheckProgram(program) }
   }
  }
  row["passed"] = err == nil
  if err != nil { row["error"] = err.Error(); failed = true }
  _ = json.NewEncoder(os.Stdout).Encode(row)
 }
 if failed { os.Exit(1) }
}
