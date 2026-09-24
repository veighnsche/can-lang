"""Exercise current Can pattern checking with isolated projects under /private/tmp.

Usage: python3 run.py /absolute/path/to/bundled/canlc
"""

import json
import shutil
import subprocess
import sys
from pathlib import Path


ROOT = Path("/private/tmp/can-pattern-nested-cases")
HEADER = "package app\n    provides []\n    uses []\n"
MAIN = "fn void main\n    emits []\n    given\n        str[] args\n    asserts\n        empty: [] => ok\n    ok\n"


def payment(extra_leaf=False):
    leaves = "record paid\nrecord pending\nrecord declined\n"
    variant = "variant payment\n    paid\n    pending\n    declined\n"
    if extra_leaf:
        leaves += "record refunded\n"
        variant += "    refunded\n"
    return leaves + variant


def root_pattern(last, extra_leaf=False):
    return (
        HEADER
        + payment(extra_leaf)
        + "fn bool accepted\n    emits []\n    given\n        payment value\n"
        + "    asserts\n        paid_case: paid() => ok true\n"
        + "        pending_case: pending() => ok false\n"
        + "        declined_case: declined() => ok false\n"
        + ("        refunded_case: refunded() => ok false\n" if extra_leaf else "")
        + "    match value\n        paid => ok true\n"
        + f"        pending => ok false\n        {last} => ok false\n"
        + MAIN
    )


def nested_array(element, head, assertion):
    return (
        HEADER
        + payment()
        + f"fn bool select\n    emits []\n    given\n        {element}[] values\n"
        + f"    asserts\n{assertion}"
        + f"    match values\n        [{head}, ...rest] => ok true\n"
        + "        _ => ok false\n"
        + MAIN
    )


def nested_record(field, body, fallback, assertion, pattern="paid"):
    return (
        HEADER
        + payment()
        + f"record box\n    {field} item\n"
        + "fn bool select\n    emits []\n    given\n        box value\n"
        + f"    asserts\n{assertion}"
        + f"    match value\n        box({pattern}) => ok {body}\n"
        + ("        _ => ok false\n" if fallback else "")
        + MAIN
    )


CASES = {
    "root-typo": root_pattern("decliend"),
    "root-typo-new-leaf": root_pattern("decliend", True),
    "root-correct-new-leaf": root_pattern("declined", True),
    "root-constructor-typo": root_pattern("decliend()"),
    "nested-array-leaf": nested_array(
        "payment", "paid",
        "        paid_case: [paid()] => ok true\n"
        "        pending_case: [pending()] => ok false\n"
        "        empty_case: [] => ok false\n",
    ),
    "nested-array-typo": nested_array(
        "payment", "paidd",
        "        paid_case: [paid()] => ok true\n"
        "        pending_case: [pending()] => ok false\n"
        "        empty_case: [] => ok false\n",
    ),
    "nested-array-typo-accepted": nested_array(
        "payment", "paidd",
        "        paid_case: [paid()] => ok true\n"
        "        pending_case: [pending()] => ok true\n"
        "        empty_case: [] => ok false\n",
    ),
    "nested-array-scalar-binder": nested_array(
        "int", "paid",
        "        number_case: [3] => ok true\n"
        "        empty_case: [] => ok false\n",
    ),
    "nested-record-variant": nested_record(
        "payment", "true", True,
        "        paid_case: box(paid()) => ok true\n"
        "        pending_case: box(pending()) => ok false\n",
    ),
    "nested-record-typo": nested_record(
        "payment", "true", False,
        "        paid_case: box(paid()) => ok true\n"
        "        pending_case: box(pending()) => ok false\n",
        "paidd",
    ),
    "nested-record-typo-covered-arm": nested_record(
        "payment", "true", True,
        "        paid_case: box(paid()) => ok true\n"
        "        pending_case: box(pending()) => ok false\n",
        "paidd",
    ),
    "nested-record-scalar-binder": nested_record(
        "int", "paid", False,
        "        three_case: box(3) => ok 3\n"
        "        four_case: box(4) => ok 4\n",
    ).replace("fn bool select", "fn int select"),
    "array-value-binder": (
        HEADER
        + "fn int head\n    emits []\n    given\n        int[] values\n"
        + "    asserts\n        pair: [4, 5] => ok 5\n"
        + "        empty: [] => ok 0\n"
        + "    match values\n        [first, ...rest] => ok first + rest.length\n"
        + "        _ => ok 0\n"
        + MAIN
    ),
}


def invoke(*args):
    process = subprocess.run(args, text=True, capture_output=True, check=False)
    return {
        "exit": process.returncode,
        "stdout": process.stdout.strip(),
        "stderr": process.stderr.strip(),
    }


def main():
    if len(sys.argv) != 2:
        raise SystemExit("usage: python3 run.py /absolute/path/to/bundled/canlc")
    canlc = Path(sys.argv[1]).resolve(strict=True)
    if ROOT.exists():
        shutil.rmtree(ROOT)
    ROOT.mkdir(parents=True)
    results = {}
    for name, source in CASES.items():
        project = ROOT / name
        (project / "src").mkdir(parents=True)
        (project / "can.project.json").write_text(
            '{"source_root":"src","error_registry":"can.errors.json"}\n'
        )
        (project / "can.errors.json").write_text('{"active":[],"retired":[]}\n')
        source_file = project / "src" / "main.can"
        source_file.write_text(source)
        results[name] = {
            "parse": invoke(str(canlc), "parse", str(source_file)),
            "build": invoke(str(canlc), "build", str(project)),
            "assert": invoke(str(canlc), "assert", str(project)),
        }
    path = ROOT / "results.json"
    path.write_text(json.dumps(results, indent=2) + "\n")
    print(path)
    for name, phases in results.items():
        print(name, {phase: details["exit"] for phase, details in phases.items()})


if __name__ == "__main__":
    main()
