#!/usr/bin/env python3
"""Mark task status consistently in tasks.json and its source task file.

Usage: mark-task-status.py <TASK_ID> <planned|active|blocked|complete>

Preserves the plan's JSON convention (indent=2, ensure_ascii, trailing newline)
so diffs stay minimal. Source mapping: P* -> foundation.json, K*/Q* ->
capabilities.json, I*/Z* -> integration.json, M* -> migration-tasks.json.
"""
import json
import sys
from pathlib import Path

PLAN = Path(__file__).resolve().parent.parent
STATUSES = {"planned", "active", "blocked", "complete"}


def source_for(task_id: str) -> str:
    if task_id.startswith("P"):
        return "foundation.json"
    if task_id.startswith(("K", "Q")):
        return "capabilities.json"
    if task_id.startswith(("I", "Z")):
        return "integration.json"
    if task_id.startswith("M"):
        return "migration-tasks.json"
    raise SystemExit(f"unknown task prefix: {task_id}")


def load(name: str):
    path = PLAN / name
    raw = path.read_text()
    return path, raw, json.loads(raw)


def save(path: Path, raw: str, data) -> None:
    out = json.dumps(data, indent=2, ensure_ascii=True) + "\n"
    if out == raw:
        return
    path.write_text(out)


def set_status(data: dict, task_id: str, status: str, fname: str) -> None:
    for rec in data["tasks"]:
        if rec.get("id") == task_id:
            rec["status"] = status
            return
    raise SystemExit(f"{task_id} not found in {fname}")


def main(argv: list) -> int:
    if len(argv) != 3:
        print(__doc__.strip().splitlines()[0])
        return 2
    task_id, status = argv[1], argv[2]
    if status not in STATUSES:
        raise SystemExit(f"bad status {status!r}; want one of {sorted(STATUSES)}")
    source = source_for(task_id)
    for fname in ("tasks.json", source):
        path, raw, data = load(fname)
        set_status(data, task_id, status, fname)
        save(path, raw, data)
    print(f"{task_id} -> {status} (tasks.json, {source})")
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv))
