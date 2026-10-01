#!/usr/bin/env python3
"""R-core observation corpus runner. Stages committed fixtures at a fixed
scratch root and runs every corpus case against one canlc binary.

Usage:
  observe.py run <canlc> <out.json>      # stage + observe, write records
  observe.py compare <canlc> <fixed.json>  # stage + re-run, byte-compare
"""
import json
import os
import shutil
import subprocess
import sys

HERE = os.path.dirname(os.path.abspath(__file__))


def load_corpus():
    with open(os.path.join(HERE, "corpus.json")) as f:
        return json.load(f)


def stage(corpus):
    root = corpus["stage_root"]
    shutil.rmtree(root, ignore_errors=True)
    staged = {}
    for case in corpus["cases"]:
        name = case["fixture"]
        if name not in staged:
            src = os.path.join(HERE, name)
            dst = os.path.join(root, name)
            shutil.copytree(src, dst)
            staged[name] = dst
    return staged


def run_case(canlc, staged, case):
    argv = [(canlc if t == "{bin}" else staged[case["fixture"]] if t == "{proj}" else t)
           for t in case["argv"]]
    p = subprocess.run(argv, capture_output=True, timeout=120)
    return {"name": case["name"], "stdout": p.stdout.decode(),
            "stderr": p.stderr.decode(), "exit": p.returncode}


def main():
    mode, canlc, path = sys.argv[1], sys.argv[2], sys.argv[3]
    if not os.path.isabs(canlc):
        raise SystemExit("refusing non-absolute binary path")
    corpus = load_corpus()
    staged = stage(corpus)
    if mode == "run":
        obs = [run_case(canlc, staged, c) for c in corpus["cases"]]
        with open(path, "w") as f:
            json.dump(obs, f, indent=2)
        for o in obs:
            print(f"{o['name']}: exit={o['exit']} out={len(o['stdout'])} err={len(o['stderr'])}")
    elif mode == "compare":
        fixed = {o["name"]: o for o in json.load(open(path))}
        fails = 0
        for case in corpus["cases"]:
            got = run_case(canlc, staged, case)
            want = fixed[case["name"]]
            ok = (got["stdout"] == want["stdout"] and
                  got["stderr"] == want["stderr"] and got["exit"] == want["exit"])
            print(f"{got['name']}: {'MATCH' if ok else 'MISMATCH'}")
            if not ok:
                fails += 1
        if fails:
            raise SystemExit(f"{fails} mismatches")
        print("all observations reproduce byte-for-byte")
    else:
        raise SystemExit("usage: observe.py run|compare <canlc> <file>")


main()
