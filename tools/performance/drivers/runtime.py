#!/usr/bin/env python3
"""Real Bun runtime workloads. Compilation and fixture creation occur in prepare.

Independent emitter for other families:
  go build -o WORK/perfemit ./compiler/perfemit
  WORK/perfemit PROJECT OUTPUT REPO/runtime
Run both commands from REPO. Generated bindings are identified from the checked
emitter's origin identity, never by hard-coded numeric function names.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
import time

SUITES = {"generated", "runtime", "codecs", "startup"}
STARTUP_NAMES = ("minimal", "runtime", "generated", "application-bundle", "application-modules")
GENERATED_BINDINGS = ("doubled", "frequency", "sum", "branch_arithmetic", "make_counter", "update_counter", "project_counter", "make_shape", "shape_area", "is_positive", "generic_pass", "captured_map", "normalize_text", "base64_text")


def command(args, cwd, timeout=180):
    if args[0] != "bun":
        return subprocess.run(args, cwd=cwd, capture_output=True, text=True,
                              timeout=timeout, check=True)
    # Match the production launcher boundary: a private environment snapshot on
    # fd 3. Empty fixture environment is sufficient and cannot leak credentials.
    try:
        saved = os.dup(3)
    except OSError:
        saved = None
    read_fd, write_fd = os.pipe()
    try:
        os.write(write_fd, b"{}")
        os.close(write_fd)
        write_fd = -1
        os.dup2(read_fd, 3, inheritable=True)
        return subprocess.run(args, cwd=cwd, capture_output=True, text=True,
                              timeout=timeout, check=True, pass_fds=(3,))
    finally:
        if write_fd >= 0:
            os.close(write_fd)
        if read_fd != 3:
            os.close(read_fd)
        os.close(3)
        if saved is not None:
            os.dup2(saved, 3)
            os.close(saved)


def adapter(generated, names=GENERATED_BINDINGS):
    exports = ['export {$canInitialize as initialize} from "./program/state.ts";']
    found = set()
    inventory = []
    for path in sorted((generated / "packages").rglob("*.ts")):
        source = path.read_text()
        for name in names:
            match = re.search(r'export async function (\$canFunction\d+)\([^\n]*\nlet \$canOrigin = [^\n]*gallery::' + name + r'"', source)
            if match:
                if name in found:
                    raise ValueError("ambiguous generated binding: " + name)
                found.add(name)
                inventory.append({"name": name, "binding": match[1], "module": path.relative_to(generated).as_posix(), "module_bytes": path.stat().st_size, "module_sha256": hashlib.sha256(path.read_bytes()).hexdigest()})
                exports.append(f'export {{{match[1]} as {name}}} from "./{path.relative_to(generated).as_posix()}";')
    if found != set(names):
        raise ValueError("emitter did not emit the requested workload bindings")
    (generated / "generated-adapter.ts").write_text("\n".join(exports) + "\n")
    (generated / "bindings.json").write_text(json.dumps(inventory, indent=2) + "\n")


def prepare(args):
    requested = set(args.suites.split(","))
    if not requested <= SUITES:
        raise ValueError("unknown runtime suites: " + str(requested - SUITES))
    if not shutil.which("bun") or not shutil.which("go"):
        raise FileNotFoundError("Bun and Go must already be installed")
    work = args.work_dir
    work.mkdir(parents=True, exist_ok=True)
    command(["go", "build", "-o", str(work / "perfemit"), "./compiler/perfemit"], args.repo)
    generated = work / "generated"
    command([str(work / "perfemit"), str(args.repo / "tools/performance/fixtures/runtime"),
             str(generated), str(args.repo / "runtime")], args.repo)
    adapter(generated)
    # Runtime modules resolve their existing dependencies through this link.
    modules = args.repo / "node_modules"
    if not modules.exists():
        raise FileNotFoundError("existing node_modules required; preparation never installs dependencies")
    target = work / "node_modules"
    if not target.exists():
        target.symlink_to(modules, target_is_directory=True)
    shutil.copy2(args.repo / "tools/performance/drivers/runtime-bench.ts", work / "runtime-bench.ts")
    (work / "startup-minimal.ts").write_text('console.log("can-perf-ready");\n')
    (work / "startup-runtime.ts").write_text('import {success, value} from "./generated/runtime/completion.ts";\nif(value(success(42n))!==42n) throw Error("completion");\nconsole.log("can-perf-ready");\n')
    (work / "startup-generated.ts").write_text('import {initialize,doubled} from "./generated/generated-adapter.ts";\nimport {array} from "./generated/runtime/data.ts";\nimport {value} from "./generated/runtime/completion.ts";\nawait initialize(); const result=value(await doubled(array([1n,2n])));\nif(result.length!==2 || result[1]!==4n || !Object.isFrozen(result)) throw Error("generated result");\nconsole.log("can-perf-ready");\n')
    application = '\n'.join([
        'import * as g from "./generated/generated-adapter.ts";',
        'import {array} from "./generated/runtime/data.ts";',
        'import {value} from "./generated/runtime/completion.ts";',
        'await g.initialize(); const source=array([1n,2n,3n]);',
        'if(value(await g.sum(source))!==6n) throw Error("fold");',
        'const shifted=value(await g.captured_map(source,4n)); if(shifted[2]!==7n || !Object.isFrozen(shifted)) throw Error("capture");',
        'const original=value(await g.make_counter(3n)); const updated=value(await g.update_counter(original,7n));',
        'if(original.value!==3n || updated.value!==10n || !Object.isFrozen(updated)) throw Error("record");',
        'if(value(await g.normalize_text(" E\\u0301 😀 "))!=="é 😀") throw Error("unicode");',
        'if(value(await g.base64_text("héllo"))!=="aMOpbGxv") throw Error("bytes");',
        'if(value(await g.is_positive(0n))!==false) throw Error("recovery");',
        'console.log("can-perf-ready");',
    ]) + '\n'
    (work / "startup-application-bundle.ts").write_text(application)
    command(["bun", "build", str(work / "startup-application-bundle.ts"), "--target=bun",
             "--format=esm", "--outfile", str(work / "startup-application-bundle.js")], work)
    transform = command(["bun", str(args.repo / "tools/performance/drivers/runtime-transpile.ts"),
                         str(generated), str(work / "generated-js")], work)
    (work / "transpilation.json").write_text(transform.stdout)
    modules_application = application.replace("./generated/", "./generated-js/").replace('.ts"', '.js"')
    (work / "startup-application-modules.js").write_text(modules_application)
    # Bundle TypeScript once so fresh-process timings never include compilation.
    for name in ("minimal", "runtime", "generated"):
        command(["bun", "build", str(work / f"startup-{name}.ts"), "--target=bun",
                 "--format=esm", "--outfile", str(work / f"startup-{name}.js")], work)
    # Correctness preflight is outside every measurement.
    if "startup" in requested:
        for name in STARTUP_NAMES:
            child = command(["bun", str(work / f"startup-{name}.js")], work, 60)
            if child.stdout.strip() != "can-perf-ready":
                raise ValueError("startup preparation correctness failed")
    for suite in sorted(requested - {"startup"}):
        command(["bun", str(work / "runtime-bench.ts"), suite, "quick", "1", "0", "4", "validate"], work)
    return {"schema_version": 1, "suite": "prepare", "status": "complete", "cases": [],
            "notes": ["Go compilation and production emission completed outside measurement."],
            "artifacts": [str(generated), str(work / "perfemit")]}


def startup(args):
    cases = []
    for name in STARTUP_NAMES:
        samples, warmups, launches = [], [], []
        for batch in range(args.warmups + (3 if args.profile == "quick" else 7)):
            durations = []
            for _ in range(args.iterations):
                start = time.perf_counter_ns()
                result = command(["bun", str(args.work_dir / f"startup-{name}.js")], args.work_dir, 60)
                elapsed = (time.perf_counter_ns() - start) / 1e6
                if result.stdout.strip() != "can-perf-ready":
                    raise ValueError("startup child output incorrect")
                durations.append(elapsed)
            destination = warmups if batch < args.warmups else samples
            destination.append(sum(durations) / len(durations))
            if batch >= args.warmups:
                launches.append(durations)
        cases.append({"name": f"startup.bun.{name}", "unit": "ms/launch", "samples": samples,
                      "warmup_samples": warmups, "iterations_per_sample": args.iterations,
                      "timing_scope": "Parent wall clock around empty launcher snapshot setup, subprocess creation, prepared JavaScript parsing/module loading/evaluation, child correctness assertions, output capture and child exit",
                      "parameters": {"fresh_process": True, "artifact": "prepared JS module graph" if name == "application-modules" else "prepared Bun ESM bundle", "cache": "OS and Bun caches uncontrolled; no cache flush"},
                      "correctness": {"passed": True, "checks": ["child-exit-zero", "exact-ready-output", "child-result-assertions"]},
                      "metrics": {"launch_ms": launches, "entry_bytes": (args.work_dir / f"startup-{name}.js").stat().st_size,
                                  **({"prepared_javascript_modules": json.loads((args.work_dir / "transpilation.json").read_text())["javascript_modules"]} if name == "application-modules" else {}), "peak_rss": "unavailable", "allocation_bytes": "unavailable"}})
    return {"schema_version": 1, "suite": "startup", "status": "complete", "cases": cases,
            "notes": ["Prepared bundle/process startup only; no compiler, TypeScript transpilation or binary FFI compilation inside launches."] , "artifacts": []}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    subs = parser.add_subparsers(dest="action", required=True)
    for action in ("prepare", "run"):
        p = subs.add_parser(action)
        for flag in ("repo", "work-dir", "output"):
            p.add_argument("--" + flag, type=Path, required=True)
        p.add_argument("--profile", choices=("quick", "standard"), required=True)
        if action == "prepare":
            p.add_argument("--suites", required=True)
        else:
            p.add_argument("--suite", choices=sorted(SUITES), required=True)
            p.add_argument("--iterations", type=int, required=True)
            p.add_argument("--warmups", type=int, required=True)
            p.add_argument("--size", type=int, required=True)
    args = parser.parse_args()
    suite = "prepare" if args.action == "prepare" else args.suite
    try:
        if args.action == "prepare":
            result = prepare(args)
        else:
            if args.iterations < 1 or args.warmups < 0 or args.size < 1:
                raise ValueError("iterations and size must be positive; warmups must be nonnegative")
            if not (args.work_dir / "runtime-bench.ts").exists():
                raise FileNotFoundError("prepare must complete before run")
            if args.suite == "startup":
                result = startup(args)
            else:
                completed = command(["bun", str(args.work_dir / "runtime-bench.ts"), args.suite,
                                     args.profile, str(args.iterations), str(args.warmups), str(args.size)], args.work_dir, 600)
                result = json.loads(completed.stdout)
        code = 0
    except Exception as error:
        reason = str(error)
        if isinstance(error, subprocess.CalledProcessError):
            reason += "\n" + error.stderr[-8000:]
        result = {"schema_version": 1, "suite": suite,
                  "status": "blocked" if isinstance(error, FileNotFoundError) else "failed",
                  "reason": reason, "cases": [], "notes": [], "artifacts": []}
        code = 1
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(result, indent=2) + "\n")
    return code


if __name__ == "__main__":
    sys.exit(main())
