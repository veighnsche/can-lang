// perfemit emits a checked Can project for performance fixtures. Compilation is
// a preparation step; benchmark drivers invoke the resulting JavaScript later.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/emit"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/project"
)

func run() error {
	if len(os.Args) < 3 || len(os.Args) > 4 {
		return fmt.Errorf("usage: perfemit PROJECT OUTPUT [RUNTIME_ROOT]")
	}
	runtimeRoot := "runtime"
	if len(os.Args) == 4 {
		runtimeRoot = os.Args[3]
	}
	graph, err := project.Load(os.Args[1])
	if err != nil {
		return err
	}
	program, err := check.CheckProgram(graph)
	if err != nil {
		return err
	}
	var dependencies []ir.Artifact
	err = filepath.Walk(runtimeRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".ts") {
			contents, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			relative, err := filepath.Rel(runtimeRoot, path)
			if err != nil {
				return err
			}
			dependencies = append(dependencies, ir.Artifact{Path: "runtime/" + filepath.ToSlash(relative), Bytes: contents})
		}
		return nil
	})
	if err != nil {
		return err
	}
	artifacts, err := emit.ProgramModules(program, "runtime", dependencies)
	if err != nil {
		return err
	}
	for _, artifact := range artifacts {
		path := filepath.Join(os.Args[2], artifact.Path)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(path, artifact.Bytes, 0644); err != nil {
			return err
		}
	}
	fmt.Printf("emitted %d artifacts\n", len(artifacts))
	return nil
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
