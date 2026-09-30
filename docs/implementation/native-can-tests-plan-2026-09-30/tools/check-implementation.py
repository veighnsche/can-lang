"""Implementation-mode structural checks; complements frozen validate_plan.py.

validate_plan.py asserts planning-time invariants (every status == 'planned',
no implementation executed). Once P00 flips statuses, that script fails by
design at its status assertion. This script re-runs every structural check
that stays valid during implementation:

- task record completeness, dependency resolution, acyclicity
- tasks.json aggregate equals the regenerated source concatenation
- ledger/coverage/delegate/retirement counts, mappings and source hashes
- integrated-gate task presence, planning baseline frozen, plan-doc links

Source-drift note: tracked-file hashes are checked for every file WITHOUT a
recorded retirement commit. No retirement fields exist yet; the first Z02
retirement must extend retirement-map.json records and this script together.
Until then, any tracked source change fails loudly here.

Exit 0 prints a JSON summary; any failure raises ValueError.
"""
from collections import Counter
from datetime import datetime, timezone
from hashlib import sha256
import json
from pathlib import Path
import re

HERE = Path(__file__).resolve().parent.parent
REPO = HERE.parents[2]
LEDGER = REPO / "docs/syntax-taste/preparation/native-can-tests-migration-ledger-2026-09-30"
STATUSES = {"planned", "active", "blocked", "complete"}


def read(path):
    return json.loads(path.read_text())


def require(condition, message):
    if not condition:
        raise ValueError("implementation check failed: " + message)


parts = [read(HERE / f) for f in ["foundation.json", "capabilities.json", "integration.json", "migration-tasks.json"]]
tasks = [t for part in parts for t in part["tasks"]]
ids = [t["id"] for t in tasks]
require(len(ids) == len(set(ids)), "duplicate task IDs")
by_id = {t["id"]: t for t in tasks}
required = ["title", "lane", "start_after", "accept_after", "files", "changes", "acceptance", "evidence", "commit", "status"]
for t in tasks:
    require(all(k in t for k in required), "missing task field: " + t["id"])
    require(t["status"] in STATUSES, "bad status: " + t["id"] + " = " + repr(t.get("status")))
    require(bool(t["files"]) and bool(t["acceptance"]) and bool(t["evidence"]), "incomplete task card: " + t["id"])
    for dep in t["start_after"] + t["accept_after"]:
        require(dep in by_id, "unknown prerequisite: " + t["id"] + " -> " + dep)
seen, visiting, order = set(), set(), []


def visit(id):
    if id in seen:
        return
    require(id not in visiting, "dependency cycle: " + id)
    visiting.add(id)
    for dep in by_id[id]["start_after"] + by_id[id]["accept_after"]:
        visit(dep)
    visiting.remove(id)
    seen.add(id)
    order.append(id)


for id in ids:
    visit(id)

aggregate = read(HERE / "tasks.json")
require(aggregate["tasks"] == tasks, "tasks.json aggregate differs from source task files")
require(aggregate["topological_completion_order"] == order, "tasks.json order differs from recomputed order")

rows = [r for f in ["core", "browser", "sql-history", "lifecycle", "external-callers"] for r in read(LEDGER / (f + ".json"))["rows"]]
row_ids = {r["id"] for r in rows}
require(len(rows) == len(row_ids) == 292, "source ledger mismatch")
migrations = [t for t in tasks if t["id"].startswith("M")]
assigned = [id for t in migrations for id in t["row_ids"]]
require(Counter(assigned) == Counter(row_ids), "migration rows omitted/duplicated")
require(len({t["files"][0] for t in migrations}) == 45, "migration ownership overlap")
coverage = read(HERE / "coverage-map.json")
cm = coverage["row_assignments"]
require(Counter(r["row_id"] for r in cm) == Counter(row_ids), "coverage map mismatch")
original = {r["id"]: r for r in rows}
for r in cm:
    require(r["task_id"] in by_id, "unknown migration owner")
    for field in ["source", "protects", "observes", "variants", "disposition"]:
        require(r[field] == original[r["row_id"]][field], "changed source obligation: " + r["row_id"] + "/" + field)
    require(bool(r.get("substeps")), "row lacks execution substeps: " + r["row_id"])
    for dep in r.get("start_after", []) + r.get("accept_after", []) + r.get("qualification_gates", []) + r.get("if_retained_accept_after", []):
        require(dep in by_id, "unknown row gate: " + r["row_id"] + "/" + dep)

source_index = read(LEDGER / "source-index.json")["files"]
require(len(source_index) == 224, "source file count mismatch")
for f in source_index:
    p = REPO / f["path"]
    require(p.is_file() and sha256(p.read_bytes()).hexdigest() == f["sha256"], "source drift: " + f["path"])
delegates = read(LEDGER / "delegated-oracles.json")["suites"]
require(len(delegates) == 30, "delegate count mismatch")
for d in delegates:
    require(sha256((REPO / d["source"]).read_bytes()).hexdigest() == d["sha256"], "delegate drift: " + d["source"])
assigned_delegates = coverage["delegated_oracle_assignments"]
require(Counter(d["id"] for d in assigned_delegates) == Counter(d["id"] for d in delegates), "delegate owner mismatch")
for d in assigned_delegates:
    require(d["owner_group"] in by_id, "missing delegate owner")
    original_delegate = next(x for x in delegates if x["id"] == d["id"])
    require(d["source_sha256"] == original_delegate["sha256"], "delegate hash omitted/changed")
    require(set(d["parent_rows"]) == set(original_delegate["parent_rows"]), "delegate parent mapping changed")
retirement = read(HERE / "retirement-map.json")
require(len(retirement["tracked_test_files"]) == 224, "retirement file count mismatch")
require(len(retirement["external_callers"]) == 16, "external caller count mismatch")
require({f["path"] for f in retirement["tracked_test_files"]} == {f["path"] for f in source_index}, "retirement paths differ")
for f in retirement["tracked_test_files"]:
    require(f["owner"] == "integrator", "shared-file retirement owner mismatch")

hard_gates = {g: "Q" + g for g in ["N1", "N2", "N3", "B1", "B2", "B3", "B4", "B5", "F1", "F2", "D1", "D2", "D3", "D4"]}
require(all(q in by_id for q in hard_gates.values()), "missing integrated hard gate")
baseline = read(HERE / "planning-baseline.json")
require(all(x == "unrun" for x in baseline["integrated_gates"].values()), "planning baseline gates changed")
require(baseline["head"] == "4cf8f91d97f45cf937c3464de54bc6275e70d49f", "planning baseline head changed")
links = []
for p in [HERE.parent / "native-can-tests-plan-2026-09-30.md"] + sorted(HERE.glob("*.md")):
    for target in re.findall(r"\[[^\]]*\]\(([^)]+)\)", p.read_text()):
        target = target.strip("<>")
        if "://" in target or target.startswith("#"):
            continue
        target = re.sub(r":\d+$", "", target.split("#")[0])
        if target:
            linked = (p.parent / target).resolve()
            require(linked.exists() or linked == HERE / "validation.json" or linked == HERE / "tasks.json", "broken plan link: " + str(p) + " -> " + target)
            links.append(target)

summary = {
    "checked_utc": datetime.now(timezone.utc).isoformat(),
    "scope": "implementation-mode structural checks only; no qualification",
    "task_count": len(tasks),
    "status_counts": dict(Counter(t["status"] for t in tasks)),
    "ledger_rows_exactly_once": len(rows),
    "tracked_test_files_unchanged": len(source_index),
    "delegated_oracle_suites_unchanged": len(delegates),
    "local_links_checked": len(links),
}
print(json.dumps(summary, indent=2))
