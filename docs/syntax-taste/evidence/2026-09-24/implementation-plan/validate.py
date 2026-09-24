"""Validate the implementation plan's dependency and concurrency tables.

This checks a documentation graph. It does not execute Can or qualify a build.
Run: python3 docs/syntax-taste/evidence/2026-09-24/implementation-plan/validate.py
"""

from collections import Counter
from datetime import datetime, timezone
import hashlib
import json
from pathlib import Path
import re


HERE = Path(__file__).resolve().parent
TASTE = HERE.parents[2]
TASKS = TASTE / "post-upgrade-implementation-tasks-2026-09-24.md"
PLAN = TASTE / "post-upgrade-implementation-plan-2026-09-24.md"
DISPOSITIONS = TASTE / "post-upgrade-dispositions-2026-09-24.md"


def cells(line):
    return [cell.strip() for cell in line.strip().strip("|").split("|")]


def require(condition, message):
    if not condition:
        raise ValueError(message)


def main():
    task_text = TASKS.read_text()
    plan_text = PLAN.read_text()
    tasks = {}
    for line in task_text.splitlines():
        if not re.match(r"^\| UP\d\d \|", line):
            continue
        task_id, lane, deps, writes, title = cells(line)
        require(task_id not in tasks, f"Duplicate task: {task_id}")
        tasks[task_id] = {
            "lane": lane,
            "dependencies": re.findall(r"UP\d\d", deps),
            "write_sets": [part.strip() for part in writes.split(",")],
            "title": title,
        }
    expected_ids = [f"UP{n:02}" for n in range(1, 28)]
    require(list(tasks) == expected_ids, "Task IDs are not UP01–UP27 in order")
    for task_id, task in tasks.items():
        require(task["lane"] in {"I", "C", "R", "B"}, f"Bad lane: {task_id}")
        deps = task["dependencies"]
        require(len(deps) == len(set(deps)), f"Duplicate dependency: {task_id}")
        for dep in deps:
            require(dep in tasks, f"Unknown dependency: {task_id} -> {dep}")
            require(dep < task_id, f"Not topological: {task_id} -> {dep}")
    headings = re.findall(r"^### (UP\d\d) —", task_text, re.MULTILINE)
    require(headings == expected_ids, "Task detail headings do not match ledger")

    waves = []
    occurrences = Counter()
    assignment = {}
    for line in task_text.splitlines():
        if not re.match(r"^\| \d+ \|", line):
            continue
        wave, *columns = cells(line)
        wave = int(wave)
        require(wave == len(waves), "Wave numbers are not consecutive")
        active = {}
        writes = {}
        for lane, cell in zip(["I", "C", "R", "B"], columns, strict=True):
            if cell == "—":
                continue
            require(cell in tasks, f"Unknown wave task: {cell}")
            require(tasks[cell]["lane"] == lane, f"Wrong lane: {cell}")
            active[lane] = cell
            assignment[cell] = wave
            occurrences[cell] += 1
            for write_set in tasks[cell]["write_sets"]:
                require(write_set not in writes,
                        f"Wave {wave} write collision: {write_set}")
                writes[write_set] = cell
        require(not ("I" in active and len(active) > 1),
                f"Exclusive coordinator gate overlaps coding: wave {wave}")
        require(len(active) <= 3, f"Worker budget exceeded: wave {wave}")
        waves.append({"wave": wave, "tasks": active})
    require(occurrences == Counter(expected_ids), "Missing/duplicated wave task")
    for task_id, task in tasks.items():
        for dep in task["dependencies"]:
            require(assignment[dep] < assignment[task_id],
                    f"Wave violates dependency: {task_id} -> {dep}")

    fix_section = DISPOSITIONS.read_text().split("## Fix in this upgrade", 1)[1]
    fix_section = fix_section.split("## Retain or defer", 1)[0]
    fix_ids = set(re.findall(r"^\| \*\*([BUSP]\d\d)\*\*", fix_section, re.MULTILINE))
    coverage = {}
    for line in plan_text.splitlines():
        if not re.match(r"^\| [BUSP]\d\d \|", line):
            continue
        finding, _, task_cell = cells(line)
        require(finding not in coverage, f"Duplicate finding: {finding}")
        coverage[finding] = re.findall(r"UP\d\d", task_cell)
        require(coverage[finding], f"No tasks for {finding}")
        require(set(coverage[finding]) <= tasks.keys(), f"Unknown task in {finding}")
    require(len(fix_ids) == 11 and set(coverage) == fix_ids,
            "Coverage does not equal the 11 selected Fix findings")

    link_count = 0
    for document in [TASKS, PLAN, HERE / "README.md"]:
        for target in re.findall(r"(?<!!)\[[^]]+\]\(([^)]+)\)", document.read_text()):
            if target.startswith(("https://", "http://", "#", "mailto:")):
                continue
            target_path = target.split("#", 1)[0]
            resolved = (document.parent / target_path).resolve()
            require(resolved.exists() or resolved == HERE / "validation.json",
                    f"Missing link in {document.name}: {target}")
            link_count += 1
    result = {
        "checked_at_utc": datetime.now(timezone.utc).isoformat(),
        "kind": "documentation-plan-validation",
        "passed": True,
        "task_count": len(tasks),
        "fix_count": len(coverage),
        "wave_count": len(waves),
        "max_coding_workers": 3,
        "local_links_checked": link_count,
        "checks": ["ordered-acyclic-dependencies", "complete-task-details",
                   "each-task-scheduled-once", "dependencies-precede-wave",
                   "lane-match", "disjoint-declared-write-sets",
                   "exclusive-integration-gates", "all-selected-fixes-covered",
                   "local-link-targets-exist"],
        "inputs_sha256": {
            path.name: hashlib.sha256(path.read_bytes()).hexdigest()
            for path in [PLAN, TASKS, DISPOSITIONS, Path(__file__)]
        },
        "coverage": coverage,
        "waves": waves,
        "limit": "Validates declared scheduling and coverage, not implementation or runtime behavior.",
    }
    (HERE / "validation.json").write_text(json.dumps(result, indent=2) + "\n")
    print(json.dumps({key: result[key] for key in
                      ["passed", "task_count", "fix_count", "wave_count", "local_links_checked"]}))


if __name__ == "__main__":
    main()
