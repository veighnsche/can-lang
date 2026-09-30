// Research helper: copied into an immediately owned compiler/ temporary
// directory by run_probe.py so Go's internal-package boundary remains intact.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/driver"
	"github.com/veighnsche/can-lang/compiler/internal/emit"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/project"
)

func main() {
	if len(os.Args) == 3 && os.Args[1] == "check-json" {
		report, code := driver.CheckProjectJSON(context.Background(), os.Args[2])
		os.Stdout.Write(report)
		os.Exit(code)
	}
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) != 4 {
		return fmt.Errorf("expected fixture, output and repository directories")
	}
	graph, err := project.Load(os.Args[1])
	if err != nil {
		return err
	}
	program, err := check.CheckAssertionProgram(graph)
	if err != nil {
		return err
	}
	var dependencies []ir.Artifact
	runtime := filepath.Join(os.Args[3], "runtime")
	err = filepath.WalkDir(runtime, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".ts") {
			return err
		}
		relative, err := filepath.Rel(runtime, path)
		if err == nil {
			dependencies = append(dependencies, ir.Artifact{Path: filepath.ToSlash(filepath.Join("runtime", relative))})
		}
		return err
	})
	if err != nil {
		return err
	}
	artifacts, err := emit.AssertionModules(program, "runtime", dependencies)
	if err != nil {
		return err
	}
	metadata := map[string]any{}
	bytes, files := 0, 0
	for _, artifact := range artifacts {
		if len(artifact.Bytes) == 0 {
			continue
		}
		path := filepath.Join(os.Args[2], artifact.Path)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return err
		}
		if err := os.WriteFile(path, artifact.Bytes, 0600); err != nil {
			return err
		}
		files++
		bytes += len(artifact.Bytes)
		for _, chunk := range strings.Split(string(artifact.Bytes), "export async function ")[1:] {
			open, boundary := strings.Index(chunk, "("), strings.Index(chunk, "try {")
			if open >= 0 && boundary >= 0 && strings.Contains(chunk[:boundary], "::editor\"") {
				metadata["module"], metadata["function"] = artifact.Path, chunk[:open]
			}
		}
	}
	for _, typ := range program.Model.Types() {
		if strings.HasSuffix(typ.Declaration(), "::service_fields") {
			metadata["record_identity"] = typ.Identity()
		}
	}
	if metadata["function"] == nil || metadata["record_identity"] == nil {
		return fmt.Errorf("emitted editor/record not found; re-scope probe")
	}
	metadata["generated_bytes"], metadata["generated_files"] = bytes, files
	return json.NewEncoder(os.Stdout).Encode(metadata)
}
