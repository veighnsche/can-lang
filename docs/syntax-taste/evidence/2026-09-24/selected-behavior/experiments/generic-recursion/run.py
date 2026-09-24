#!/usr/bin/env python3
"""Reproduce the selected generic-recursion checker probe from current source."""

from __future__ import annotations

import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile


ROOT = Path(__file__).resolve().parents[7]
HERE = Path(__file__).resolve().parent
CASES = HERE / "cases"
RUNNER = ROOT / "compiler" / ".selected-generic-recursion-probe"
MAIN = """fn void main
    emits []
    given
        str[] arguments
    asserts
        empty: [] => ok
    ok
"""

IDENTITY = """fn item identity<item>
    emits []
    given
        item value
    asserts
        number: 3 => ok 3
    ok value
"""

SOURCES = {
    "stationary-self": """package app
    provides [first]
    uses []
fn item first<item>
    emits []
    given
        item value
        bool stop
    asserts
        base: 3, true => ok 3
    match stop
        false => ok call first<item>(value, true)
        true => ok value
""" + MAIN,
    "stationary-mutual": """package app
    provides [first, second]
    uses []
fn item first<item>
    emits []
    given
        item value
        bool stop
    asserts
        base: 3, true => ok 3
    match stop
        false => ok call second<item>(value, true)
        true => ok value
fn item second<item>
    emits []
    given
        item value
        bool stop
    asserts
        base: 3, true => ok 3
    match stop
        false => ok call first<item>(value, true)
        true => ok value
""" + MAIN,
    "expanding-mutual": """package app
    provides [a, b]
    uses []
fn void a<item>
    emits []
    given
        item value
        bool stop
    asserts
        base: 3, true => ok
    match stop
        false => relay call b<item[]>([value], true)
        true => ok
fn void b<item>
    emits []
    given
        item value
        bool stop
    asserts
        base: 3, true => ok
    match stop
        false => relay call a<item>(value, true)
        true => ok
""" + MAIN,
    "acyclic-nested-public": """package app
    provides [box, identity, nested]
    uses []
record box<item>
    item value
""" + IDENTITY + """fn box<item> nested<item>
    emits []
    given
        item value
    asserts
        number: 3 => ok box(3)
    ok call identity<box<item>>(box(value))
""" + MAIN,
    "acyclic-nested-private": """package app
    provides [box]
    uses []
record box<item>
    item value
""" + IDENTITY + """fn box<item> nested<item>
    emits []
    given
        item value
    asserts
        number: 3 => ok box(3)
    ok call identity<box<item>>(box(value))
""" + MAIN,
    "public-identity-control": """package app
    provides [identity]
    uses []
""" + IDENTITY + MAIN,
}

GO_RUNNER = r'''package main

import (
    "encoding/json"
    "os"
    "path/filepath"
    "sort"
    "strings"

    "github.com/veighnsche/can-lang/compiler/internal/check"
    "github.com/veighnsche/can-lang/compiler/internal/emit"
    "github.com/veighnsche/can-lang/compiler/internal/ir"
    "github.com/veighnsche/can-lang/compiler/internal/project"
    "github.com/veighnsche/can-lang/compiler/internal/types"
)

func main() {
    result := map[string]any{"checked": false}
    graph, err := project.Load(os.Args[1])
    if err == nil {
        var program *check.Program
        program, err = check.CheckProgram(graph)
        if err == nil {
            result["checked"] = true
            var concrete []string
            for _, fn := range program.Functions {
                if fn.Instance != "" && !strings.Contains(fn.Instance, "/symbolic") {
                    concrete = append(concrete, fn.Symbol.ID)
                }
            }
            sort.Strings(concrete)
            result["concrete_functions"] = concrete
            var symbolicTypes int
            for _, typ := range program.Model.Types() {
                if typ.Kind() == types.Parameter { symbolicTypes++ }
            }
            result["symbolic_parameter_model_nodes"] = symbolicTypes
            var dependencies []ir.Artifact
            walkErr := filepath.WalkDir(os.Args[2], func(path string, entry os.DirEntry, err error) error {
                if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".ts") { return err }
                relative, err := filepath.Rel(os.Args[2], path)
                if err != nil { return err }
                dependencies = append(dependencies, ir.Artifact{Path: filepath.ToSlash(filepath.Join("runtime", relative))})
                return nil
            })
            if walkErr != nil { result["emit_error"] = walkErr.Error() } else {
            var artifacts int
            modules, emitErr := emit.AssertionModules(program, "runtime", dependencies)
            if emitErr != nil {
                result["emit_error"] = emitErr.Error()
            } else {
                for _, module := range modules {
                    if len(module.Bytes) > 0 { artifacts++ }
                }
                result["emitted_modules"] = artifacts
            }
            }
        }
    }
    if err != nil { result["error"] = err.Error() }
    json.NewEncoder(os.Stdout).Encode(result)
}
'''


def run(command: list[str], *, cwd: Path = ROOT) -> dict[str, object]:
    process = subprocess.run(command, cwd=cwd, capture_output=True, text=True,
                             env={**os.environ, "GOCACHE": "/tmp/can-generic-recursion-go-cache"})
    return {"command": command, "status": process.returncode,
            "stdout": process.stdout, "stderr": process.stderr}


def main() -> None:
    CASES.mkdir(parents=True, exist_ok=True)
    for name, source in SOURCES.items():
        directory = CASES / name
        (directory / "src").mkdir(parents=True, exist_ok=True)
        (directory / "src" / "app.can").write_text(source)
        (directory / "can.project.json").write_text('{"source_root":"src","error_registry":"can.errors.json"}\n')
        (directory / "can.errors.json").write_text('{"active":[],"retired":[]}\n')

    if RUNNER.exists():
        raise RuntimeError(f"temporary probe runner already exists: {RUNNER}")
    results: dict[str, object] = {"compiler_revision": run(["git", "rev-parse", "HEAD"])}
    with tempfile.TemporaryDirectory(prefix="can-generic-recursion-") as temp:
        tmp = Path(temp)
        try:
            RUNNER.mkdir()
            (RUNNER / "main.go").write_text(GO_RUNNER)
            (HERE / "check-runner.go.txt").write_text(GO_RUNNER)
            results["build_parser"] = run(["go", "build", "-o", str(tmp / "canlc"), "./compiler"])
            results["build_checker"] = run(["go", "build", "-o", str(tmp / "check-probe"), str(RUNNER)])
            if results["build_parser"]["status"] or results["build_checker"]["status"]:
                raise RuntimeError("failed to build current compiler or checker probe")
            cases: dict[str, object] = {}
            for name in SOURCES:
                directory = CASES / name
                parsed = run([str(tmp / "canlc"), "parse", str(directory / "src" / "app.can")])
                checked = run([str(tmp / "check-probe"), str(directory), str(ROOT / "runtime")])
                cases[name] = {"parse": parsed, "check_and_emit": checked,
                               "result": json.loads(str(checked["stdout"])) if checked["status"] == 0 else None}
            results["cases"] = cases
        finally:
            shutil.rmtree(RUNNER, ignore_errors=True)
    (HERE / "results.json").write_text(json.dumps(results, indent=2, sort_keys=True) + "\n")


if __name__ == "__main__":
    main()
