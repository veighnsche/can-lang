#!/usr/bin/env python3
"""Run every LF19 guide example through a development canlc bundle.

Usage:
  python3 run.py --bundle /path/to/can-<ver>-<target> --out results.json

Each cases/<name>/ directory holds src/main.can plus expect.json:
  {"mode": "pass"}              -- `canlc assert` exits 0 with passed=true
  {"mode": "fail", "match": s}  -- `canlc assert` exits != 0, s in stderr

A case may carry its own can.project.json (manifest negatives); otherwise the
default project manifest is generated. can.errors.json is always generated.
"""
import argparse
import json
import os
import shutil
import subprocess
import sys
import tempfile

DEFAULT_MANIFEST = {"source_root": "src", "error_registry": "can.errors.json"}
DEFAULT_ERRORS = {"active": [], "retired": []}


def run_case(canlc, case_dir, work_root):
    name = os.path.basename(case_dir)
    with open(os.path.join(case_dir, "expect.json")) as f:
        expect = json.load(f)
    proj = os.path.join(work_root, name)
    shutil.copytree(os.path.join(case_dir, "src"), os.path.join(proj, "src"))
    manifest_src = os.path.join(case_dir, "can.project.json")
    if os.path.exists(manifest_src):
        shutil.copy(manifest_src, os.path.join(proj, "can.project.json"))
    else:
        with open(os.path.join(proj, "can.project.json"), "w") as f:
            json.dump(DEFAULT_MANIFEST, f)
    with open(os.path.join(proj, "can.errors.json"), "w") as f:
        json.dump(DEFAULT_ERRORS, f)
    proc = subprocess.run(
        [canlc, "assert", proj],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        timeout=300,
    )
    report = None
    if proc.returncode == 0 and proc.stdout.strip():
        try:
            report = json.loads(proc.stdout)
        except json.JSONDecodeError:
            report = None
    ok = False
    detail = ""
    if expect["mode"] == "pass":
        ok = proc.returncode == 0 and bool(report) and report.get("passed") is True
        if report:
            roots = [(a["root"]["declaration"], a["root"]["name"], a.get("passed")) for a in report.get("assertions", [])]
            detail = f"{len(roots)} assertions: {roots} evidence={sorted({e for a in report.get('assertions', []) for e in a.get('evidence', [])})}"
        else:
            detail = f"exit={proc.returncode} stdout={proc.stdout[:300]!r} stderr={proc.stderr[:500]!r}"
    elif expect["mode"] == "fail":
        ok = proc.returncode != 0 and expect["match"] in proc.stderr
        detail = f"exit={proc.returncode} stderr={proc.stderr[:500]!r}"
    else:
        raise ValueError(f"unknown mode {expect['mode']}")
    return {"name": name, "ok": ok, "mode": expect["mode"], "detail": detail,
            "exit": proc.returncode, "stderr": proc.stderr[:2000]}


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--bundle", required=True)
    ap.add_argument("--out", required=True)
    args = ap.parse_args()
    canlc = os.path.join(args.bundle, "bin", "canlc")
    root = os.path.dirname(os.path.abspath(__file__))
    cases = sorted(d for d in os.listdir(os.path.join(root, "cases"))
                   if os.path.isdir(os.path.join(root, "cases", d)))
    work_root = tempfile.mkdtemp(prefix="lf19-guide-")
    work_root = os.path.realpath(work_root)
    results = []
    for name in cases:
        if name == "incomplete-proposal":
            continue  # reviewed artifact, not a program; verdict lives in the guide
        print(f"--- {name}", flush=True)
        try:
            results.append(run_case(canlc, os.path.join(root, "cases", name), work_root))
        except Exception as e:  # noqa: BLE001 - record harness failures verbatim
            results.append({"name": name, "ok": False, "mode": "harness",
                            "detail": f"{type(e).__name__}: {e}", "exit": -1, "stderr": ""})
        print(f"    ok={results[-1]['ok']} {results[-1]['detail'][:300]}", flush=True)
    failed = [r["name"] for r in results if not r["ok"]]
    passed = len(results) - len(failed)
    with open(args.out, "w") as f:
        json.dump({"bundle": args.bundle, "passed": passed, "total": len(results), "results": results}, f, indent=1)
    print(f"{passed}/{len(results)} cases behave as expected")
    if failed:
        print("FAILED:", ", ".join(failed))
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
