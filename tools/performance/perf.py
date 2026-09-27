#!/usr/bin/env python3
"""Can's repeatable performance suites. No installs and no production edits."""

import argparse
import datetime
import hashlib
import io
import json
import math
import os
from pathlib import Path
import platform
import shutil
import socket
import subprocess
import sys
import tarfile
import time

from isolation import exclusive, execute, quiet_window
from schema import summarize_trials, validate_result

HERE = Path(__file__).resolve().parent
REPO = HERE.parents[1]
SUITES = {
    "compiler": ("compiler", "source loading, checking, emission and scaling"),
    "assertions": ("compiler", "precompiled assertions and worker supervision"),
    "artifacts": ("compiler", "output processing, validation and footprint"),
    "generated": ("runtime", "actual emitted Can versus native equivalents"),
    "runtime": ("runtime", "collections, calls and ownership helpers"),
    "codecs": ("runtime", "validated data conversion and rejection"),
    "startup": ("runtime", "fresh process and module loading"),
    "browser": ("apps", "native browser controls and observable completion"),
    "server": ("apps", "bounded-client and arrival-rate load"),
    "io": ("apps", "controlled host I/O and adapter boundaries"),
    "editor": ("compiler", "language-server feedback after edits"),
    "journeys": ("apps", "complete emitted-Can application workflow"),
}


def write_json(path, value):
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary = path.with_suffix(path.suffix + ".tmp")
    temporary.write_text(json.dumps(value, indent=2, allow_nan=False) + "\n")
    temporary.replace(path)


def digest(path):
    hasher = hashlib.sha256()
    with path.open("rb") as handle:
        for block in iter(lambda: handle.read(1024 * 1024), b""):
            hasher.update(block)
    return hasher.hexdigest()


def tree_digest(root):
    hasher = hashlib.sha256()
    count, size = 0, 0
    for path in sorted(root.rglob("*")):
        relative = str(path.relative_to(root))
        if path.is_symlink():
            payload = ("link:" + os.readlink(path)).encode()
        elif path.is_file():
            payload = digest(path).encode()
            count += 1
            size += path.stat().st_size
        else:
            continue
        hasher.update(relative.encode() + b"\0" + payload + b"\0")
    return {"sha256": hasher.hexdigest(), "files": count, "bytes": size}


def git(*args, repo=REPO):
    return subprocess.check_output(["git", *args], cwd=repo, stderr=subprocess.PIPE)


def version(command):
    try:
        return subprocess.check_output(command, text=True, stderr=subprocess.STDOUT, timeout=15).strip()
    except (OSError, subprocess.SubprocessError):
        return None


def environment():
    tools = {}
    for name, args in {"go": ["version"], "bun": ["--version"], "node": ["--version"]}.items():
        location = shutil.which(name)
        tools[name] = {"version": version([name, *args]), "sha256": digest(Path(location).resolve()) if location else None}
    return {
        "system": platform.system(), "release": platform.release(), "machine": platform.machine(),
        "cpu_count": os.cpu_count(), "python": platform.python_version(),
        "host_id": hashlib.sha256(socket.gethostname().encode()).hexdigest(), "tools": tools,
        "power_and_thermal_control": "not enforced; keep machine settings consistent",
    }


def child_environment(work):
    # Only explicit local-tool settings enter workloads; no ambient API keys.
    allowed = {"PATH", "HOME", "USER", "LOGNAME", "SHELL", "TMPDIR", "TEMP", "TMP", "LANG", "LC_ALL", "TZ",
               "PLAYWRIGHT_BROWSERS_PATH", "CAN_BUN_ARCHIVE", "CAN_PERF_BUN_ARCHIVE", "CAN_FIREFOX_WS", "CAN_PERF_BROWSER_ENGINES"}
    result = {key: value for key, value in os.environ.items() if key in allowed}
    result["GOCACHE"] = str(work / "go-cache")
    # Reuse the installed module cache read-only in ordinary cached builds;
    # any required download is explicitly disabled.
    result["GOPROXY"] = "off"
    result["GOSUMDB"] = "off"
    result["GOTOOLCHAIN"] = "local"
    if "GOMODCACHE" in os.environ:
        result["GOMODCACHE"] = os.environ["GOMODCACHE"]
    result["NO_COLOR"] = "1"
    result["CAN_PERF_SUPERVISED"] = "1"
    return result


def source_snapshot(output, revision, working_tree):
    source = output / "source"
    source.mkdir()
    resolved = git("rev-parse", "--verify", (revision or "HEAD") + "^{commit}").decode().strip()
    archive = git("archive", resolved)
    with tarfile.open(fileobj=io.BytesIO(archive)) as handle:
        handle.extractall(source, filter="data")
    working_files = []
    if working_tree:
        for raw in git("ls-files", "-z").split(b"\0"):
            if not raw:
                continue
            name = os.fsdecode(raw)
            current, target = REPO / name, source / name
            if target.is_file() or target.is_symlink():
                target.unlink()
            if not current.exists() and not current.is_symlink():
                continue
            target.parent.mkdir(parents=True, exist_ok=True)
            if current.is_symlink():
                target.symlink_to(os.readlink(current))
            elif current.is_file():
                shutil.copy2(current, target)
            working_files.append(name)
    # The current harness is deliberately applied to the chosen source revision.
    # This supports comparing two revisions with one instrument and developing
    # the harness without committing it first. These overlays never touch
    # compiler/runtime production files.
    overlays = {}
    for name in ("tools/performance", "compiler/perfmeasure", "compiler/perfemit"):
        origin = REPO / name
        if origin.exists():
            target = source / name
            if target.exists():
                shutil.rmtree(target)
            shutil.copytree(origin, target, ignore=shutil.ignore_patterns("__pycache__", "*.pyc"))
            overlays[name] = tree_digest(target)
    source_identity = tree_digest(source)
    dependency_identity = None
    dependency_source = REPO / "node_modules"
    if dependency_source.exists():
        target = source / "node_modules"
        shutil.copytree(dependency_source.resolve(), target, symlinks=True)
        for link in target.rglob("*"):
            if link.is_symlink() and not link.resolve().is_relative_to(target.resolve()):
                raise ValueError("Dependency snapshot contains an external symlink; use a self-contained installation")
        dependency_identity = tree_digest(target)
    return source, {
        "revision": resolved, "source_mode": "tracked-working-tree" if working_tree else "committed",
        "source_identity": source_identity, "harness_overlays": overlays,
        "dependencies": dependency_identity,
        "dirty_checkout": bool(git("status", "--porcelain", "--untracked-files=no").strip()),
        "working_files": len(working_files),
        "untracked_production_files": "not included; only explicit performance harness overlays are copied",
    }


def command_for(source, family, mode, work, out, args, *, suite=None, suites=None):
    command = [sys.executable, str(source / "tools/performance/drivers" / (family + ".py")), mode,
               "--repo", str(source), "--work-dir", str(work), "--output", str(out), "--profile", args.profile]
    if suite:
        command += ["--suite", suite, "--iterations", str(args.iterations), "--warmups", str(args.warmups), "--size", str(args.size)]
    if suites:
        command += ["--suites", ",".join(suites)]
    return command


def failure_reason(path, fallback):
    try:
        result = json.loads(path.read_text())
    except (OSError, ValueError):
        return fallback
    reason = result.get("reason") if isinstance(result, dict) else None
    return reason if isinstance(reason, str) and reason else fallback


def markdown_report(manifest, summary):
    lines = ["# Can performance results", "", f"Status: **{manifest['status']}**. Evidence class: **{manifest['quality']}**.", "",
             f"Suites completed: {len(manifest['completed_suites'])}/{len(manifest['requested_suites'])}. All suites execute sequentially.", "",
             f"Total run wall time: {manifest.get('wall_seconds', 0):.2f}s, including preparation, quiet-host waits and validation.",
             f"Preparation driver wall time: {sum(x['elapsed_seconds'] for x in manifest['steps'] if x['phase'] == 'prepare'):.2f}s. "
             f"Suite driver wall time: {sum(x['elapsed_seconds'] for x in manifest['steps'] if x['phase'] == 'trial'):.2f}s.", "",
             "These orchestration durations are not summed workload latency or an overall performance score.", "",
             "Smoke and failed/incomplete runs are not accepted performance baselines.", "",
             "Independent process trials are summarized using each trial's batch median. These are not request-level tail percentiles.", "",
             "| Slice | Verified cases in this summary |", "|---|---:|"]
    for suite in manifest["requested_suites"]:
        count = sum(name.startswith(suite + "/") for name in summary)
        lines.append(f"| {suite} | {count if summary else 'not summarized'} |")
    lines += ["", "Case counts include scenario and implementation variants; they are not percentages of language or feature coverage.", "",
              "| Case | Unit | Process trials | Median | Min–max |", "|---|---|---:|---:|---:|"]
    for name, row in summary.items():
        stats = row["distribution"]
        lines.append(f"| {name} | {row['unit']} | {stats['n']} | {stats['median']:.6g} | {stats['min']:.6g}–{stats['max']:.6g} |")
    if manifest.get("reason"):
        lines += ["", "Failure or limitation: " + manifest["reason"]]
    lines += ["", "Each raw driver file contains exact timing scopes, parameters, correctness checks, notes and additional metrics.",
              "No automatic regression budget is calibrated by this run. Inspect workload-specific effects and repeatability.", ""]
    return "\n".join(lines)


def run(args):
    started = time.monotonic()
    selected = list(SUITES) if not args.suite or "all" in args.suite else list(dict.fromkeys(args.suite))
    if "all" in (args.suite or []) and len(args.suite) > 1:
        raise ValueError("Select all or individual suites, not both")
    args.profile = args.profile or ("quick" if args.smoke else "standard")
    args.trials = args.trials if args.trials is not None else (1 if args.smoke else 6)
    args.iterations = args.iterations if args.iterations is not None else 1
    args.warmups = args.warmups if args.warmups is not None else (0 if args.smoke else 2)
    args.size = args.size if args.size is not None else (10 if args.smoke else 100)
    if args.smoke and args.trials != 1:
        raise ValueError("Smoke uses one process trial; use a measured run for repeated evidence")
    output = Path(args.output).expanduser().resolve() if args.output else REPO / ".performance" / datetime.datetime.now(datetime.timezone.utc).strftime("%Y%m%dT%H%M%S.%fZ")
    if output.exists():
        raise ValueError("Output directory already exists; evidence is never overwritten")
    if output.is_relative_to(HERE):
        raise ValueError("Output cannot be nested inside the harness directory")
    output.mkdir(parents=True)
    weakened = args.quiet_seconds < 30 or args.idle_percent < 90
    quality = "smoke" if args.smoke else ("exploratory" if weakened else "measurement")
    manifest = {
        "schema_version": 1, "status": "running", "quality": quality, "started_at": time.time(),
        "requested_suites": selected, "completed_suites": [], "profile": args.profile,
        "trials": args.trials, "iterations": args.iterations, "warmups": args.warmups, "size": args.size,
        "isolation": {"quiet_seconds": args.quiet_seconds, "idle_percent": args.idle_percent, "smoke_bypasses_quiet_gate": args.smoke},
        "steps": [], "environment": environment(), "source": None,
    }
    write_json(output / "manifest.json", manifest)
    evidence = output / "isolation.jsonl"
    trials = []
    summary = {}
    print(f"Evidence directory: {output}", flush=True)
    try:
        with exclusive(args.wait_seconds, evidence):
            source, provenance = source_snapshot(output, args.revision, args.working_tree)
            manifest["source"] = provenance
            write_json(output / "manifest.json", manifest)
            env = child_environment(output)
            families = list(dict.fromkeys(SUITES[s][0] for s in selected))
            for family in families:
                work = output / "work" / family
                work.mkdir(parents=True)
                target = output / "raw" / f"prepare-{family}.json"
                target.parent.mkdir(exist_ok=True)
                command = command_for(source, family, "prepare", work, target, args, suites=[s for s in selected if SUITES[s][0] == family])
                print(f"Preparing {family} workloads (outside measured intervals)", flush=True)
                code, elapsed = execute(command, source, env, target.with_suffix(".stdout.log"), target.with_suffix(".stderr.log"), args.prepare_timeout, evidence, False)
                manifest["steps"].append({"phase": "prepare", "family": family, "returncode": code, "elapsed_seconds": elapsed, "output": str(target.relative_to(output))})
                write_json(output / "manifest.json", manifest)
                if code or not target.exists():
                    raise RuntimeError(failure_reason(target, f"{family} preparation failed; inspect its saved logs"))
                validate_result(json.loads(target.read_text()), "prepare", preparation=True)
            for suite in selected:
                family = SUITES[suite][0]
                if not args.smoke:
                    quiet_window(args.quiet_seconds, args.idle_percent, args.wait_seconds, evidence)
                print(f"Running {suite}: {args.trials} independent process trial(s)", flush=True)
                for trial in range(args.trials):
                    target = output / "raw" / f"{suite}-{trial+1:03}.json"
                    command = command_for(source, family, "run", output / "work" / family, target, args, suite=suite)
                    code, elapsed = execute(command, source, env, target.with_suffix(".stdout.log"), target.with_suffix(".stderr.log"), args.timeout, evidence, not args.smoke)
                    manifest["steps"].append({"phase": "trial", "suite": suite, "trial": trial+1, "returncode": code, "elapsed_seconds": elapsed, "output": str(target.relative_to(output))})
                    write_json(output / "manifest.json", manifest)
                    if code or not target.exists():
                        raise RuntimeError(failure_reason(target, f"{suite} failed; inspect its saved logs"))
                    result = validate_result(json.loads(target.read_text()), suite)
                    validate_sampling(result, manifest)
                    trials.append({"suite": suite, "trial": trial+1, "result": result})
                manifest["completed_suites"].append(suite)
                write_json(output / "manifest.json", manifest)
            # Dependencies are copied, never linked to a changing installation.
            if provenance["dependencies"] is not None and tree_digest(source / "node_modules") != provenance["dependencies"]:
                raise RuntimeError("Dependency snapshot changed during the run")
            summary = summarize_trials(trials)
            manifest["status"] = "complete"
            write_json(output / "summary.json", summary)
    except (OSError, ValueError, RuntimeError, subprocess.SubprocessError, KeyboardInterrupt) as exc:
        manifest["status"] = "interrupted" if isinstance(exc, KeyboardInterrupt) else "failed"
        manifest["reason"] = str(exc) or "Interrupted"
        summary = {}
        # Partial raw files stay available, but never receive a completed summary.
        print(f"Run {manifest['status']}: {manifest['reason']}", file=sys.stderr, flush=True)
    finally:
        manifest["finished_at"] = time.time()
        manifest["wall_seconds"] = time.monotonic() - started
        write_json(output / "manifest.json", manifest)
        (output / "report.md").write_text(markdown_report(manifest, summary))
    print(f"{manifest['status']}: {len(manifest['completed_suites'])}/{len(selected)} suites; {quality} evidence", flush=True)
    return 0 if manifest["status"] == "complete" else 1


def validate_sampling(result, metadata):
    batches = 3 if metadata["profile"] == "quick" else 7
    for case in result["cases"]:
        if len(case["samples"]) != batches or len(case.get("warmup_samples", [])) != metadata["warmups"]:
            raise ValueError(f"Unexpected measured or warmup batch count: {case['name']}")
        if case["iterations_per_sample"] != metadata["iterations"]:
            raise ValueError(f"Unexpected operations per batch: {case['name']}")


def verified_summary(root, meta):
    suites, count = meta.get("requested_suites"), meta.get("trials")
    if not isinstance(suites, list) or not suites or any(not isinstance(s, str) or s not in SUITES for s in suites) or len(set(suites)) != len(suites):
        raise ValueError("Invalid requested suite inventory")
    if not isinstance(count, int) or isinstance(count, bool) or count < 1:
        raise ValueError("Invalid independent trial count")
    trials = []
    for suite in suites:
        for trial in range(1, count + 1):
            path = root / "raw" / f"{suite}-{trial:03}.json"
            try:
                result = validate_result(json.loads(path.read_text()), suite)
                validate_sampling(result, meta)
            except (OSError, ValueError) as exc:
                raise ValueError(f"Invalid or missing raw evidence: {path.name}: {exc}") from exc
            trials.append({"suite": suite, "trial": trial, "result": result})
    summary = summarize_trials(trials)
    if summary != json.loads((root / "summary.json").read_text()):
        raise ValueError("Saved summary does not match verified raw trials")
    return summary


def compare(args):
    roots = [Path(args.baseline).resolve(), Path(args.candidate).resolve()]
    manifests, summaries = [], []
    for root in roots:
        meta = json.loads((root / "manifest.json").read_text())
        if meta.get("status") != "complete" or meta.get("quality") != "measurement":
            raise ValueError("Only completed measured runs can be compared; smoke, exploratory and failed evidence is excluded")
        if set(meta.get("completed_suites", [])) != set(meta.get("requested_suites", [])):
            raise ValueError("Run did not complete every requested suite")
        manifests.append(meta)
        summaries.append(verified_summary(root, meta))
    a, b = manifests
    for field in ("environment", "requested_suites", "profile", "trials", "iterations", "warmups", "size", "isolation"):
        if a[field] != b[field]:
            raise ValueError(f"Incompatible comparison condition: {field}")
    for field in ("dependencies", "harness_overlays"):
        if a["source"][field] != b["source"][field]:
            raise ValueError(f"Comparison changed {field}; use the same instrument and dependency set")
    if set(summaries[0]) != set(summaries[1]):
        raise ValueError("Case inventories differ; compare the same selected suites")
    rows = {}
    for name, left in summaries[0].items():
        right = summaries[1][name]
        for field in ("unit", "parameters", "timing_scope", "iterations_per_sample"):
            if left[field] != right[field]:
                raise ValueError(f"Workload changed: {name}/{field}")
        before, after = left["distribution"]["median"], right["distribution"]["median"]
        rows[name] = {"unit": left["unit"], "baseline": left["distribution"], "candidate": right["distribution"],
                      "change_percent": 100 * (after / before - 1) if before else None,
                      "verdict": "unclassified: practical budgets and uncertainty gates are not calibrated"}
    result = {"schema_version": 1, "baseline": str(roots[0]), "candidate": str(roots[1]), "cases": rows,
              "warning": "Separate-run median changes are descriptive, not causal speedups or calibrated regression verdicts. Confirm changes in counterbalanced fresh runs."}
    if args.output:
        target = Path(args.output).resolve()
        if target.exists():
            raise ValueError("Comparison output already exists")
        write_json(target, result)
    else:
        print(json.dumps(result, indent=2, allow_nan=False))
    return 0


def positive(value):
    number = int(value)
    if number < 1:
        raise argparse.ArgumentTypeError("must be positive")
    return number


def nonnegative(value):
    number = int(value)
    if number < 0:
        raise argparse.ArgumentTypeError("must be nonnegative")
    return number


def finite_positive(value):
    number = float(value)
    if not math.isfinite(number) or number <= 0:
        raise argparse.ArgumentTypeError("must be finite and positive")
    return number


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest="command", required=True)
    commands.add_parser("list", help="List the twelve implemented suite boundaries")
    runner = commands.add_parser("run", help="Prepare immutable inputs and execute selected suites")
    runner.add_argument("--suite", action="append", choices=["all", *SUITES])
    runner.add_argument("--smoke", action="store_true", help="Functional validation only; skips quiet gate and cannot create a baseline")
    runner.add_argument("--profile", choices=["quick", "standard"])
    runner.add_argument("--trials", type=positive)
    runner.add_argument("--iterations", type=positive)
    runner.add_argument("--warmups", type=nonnegative)
    runner.add_argument("--size", type=positive, choices=[10, 100, 1000])
    source = runner.add_mutually_exclusive_group()
    source.add_argument("--revision")
    source.add_argument("--working-tree", action="store_true")
    runner.add_argument("--output")
    runner.add_argument("--quiet-seconds", type=finite_positive, default=30)
    runner.add_argument("--idle-percent", type=finite_positive, default=90)
    runner.add_argument("--wait-seconds", type=finite_positive, default=900)
    runner.add_argument("--timeout", type=finite_positive, default=900)
    runner.add_argument("--prepare-timeout", type=finite_positive, default=1800)
    comparison = commands.add_parser("compare", help="Compare compatible completed measurements without inventing a regression gate")
    comparison.add_argument("baseline")
    comparison.add_argument("candidate")
    comparison.add_argument("--output")
    args = parser.parse_args()
    try:
        if args.command == "list":
            for name, (_, description) in SUITES.items():
                print(f"{name:12} {description}")
            return 0
        if args.command == "compare":
            return compare(args)
        if args.idle_percent > 100:
            raise ValueError("Idle percentage cannot exceed 100")
        return run(args)
    except (OSError, ValueError, subprocess.SubprocessError) as exc:
        parser.error(str(exc))


if __name__ == "__main__":
    raise SystemExit(main())
