#!/usr/bin/env python3
"""Z01 ledger-partition check: the slice manifest covers every coverage row exactly once.

Usage: python3 tools/check-z01-partition.py
Exit 0 when the manifest is a valid partition; prints FAIL lines otherwise.

Independent anchor: coverage-map.json row_assignments (the 292-row ledger).
Checks (all mechanical, no judgment):
- every manifest entry names a known group with non-empty duplicate-free rows
- every manifest row belongs to its entry's group (tasks.json row_ids)
- the union over all entries equals the ledger set exactly (no missing,
  extra, or double-covered rows)
- per group, member stems partition the group rows exactly
- the manifest stem set equals the committed slice files exactly
  (tests/native-can/src/coverage/migration/*.can minus machinery.can)
"""
import io
import json
import sys
from pathlib import Path

HERE = Path(__file__).resolve().parent.parent
REPO = HERE.parent.parent.parent
PLAN = HERE
MIG = REPO / "tests/native-can/src/coverage/migration"

FAIL = []


def fail(msg):
    FAIL.append(msg)


def main():
    man = json.load(io.open(PLAN / "z01-slices.json", encoding="utf-8"))
    tasks = json.load(io.open(PLAN / "tasks.json", encoding="utf-8"))
    by_id = {t["id"]: t for t in tasks["tasks"]}
    ledger = [
        r["row_id"]
        for r in json.load(io.open(PLAN / "coverage-map.json", encoding="utf-8"))[
            "row_assignments"
        ]
    ]
    ledger_set = set(ledger)
    if len(ledger_set) != len(ledger):
        fail("coverage-map row_assignments contain duplicate row_ids")

    seen = {}
    for stem, entry in man.items():
        mid = entry.get("group")
        rows = entry.get("rows", [])
        if mid not in by_id:
            fail(f"{stem}: unknown group {mid!r}")
            continue
        if not rows:
            fail(f"{stem}: empty row list")
        if len(set(rows)) != len(rows):
            fail(f"{stem}: duplicate rows in manifest entry")
        group_rows = set(by_id[mid].get("row_ids", []))
        outside = [r for r in rows if r not in group_rows]
        if outside:
            fail(f"{stem}: rows outside {mid}: {outside}")
        for r in rows:
            seen.setdefault(r, []).append(stem)
    multi = {r: stems for r, stems in seen.items() if len(stems) > 1}
    if multi:
        fail(f"rows in >1 manifest entry: {sorted(multi)[:10]}")
    missing = sorted(ledger_set - set(seen))
    if missing:
        fail(f"ledger rows in no manifest entry ({len(missing)}): {missing[:10]}")
    extra = sorted(set(seen) - ledger_set)
    if extra:
        fail(f"manifest rows outside the ledger: {extra[:10]}")

    by_group = {}
    for stem, entry in man.items():
        by_group.setdefault(entry.get("group"), []).append(stem)
    for mid, stems in sorted(by_group.items()):
        if mid not in by_id:
            continue
        union = [r for s in stems for r in man[s]["rows"]]
        if sorted(union) != sorted(by_id[mid].get("row_ids", [])):
            fail(f"{mid}: member stems {stems} do not partition the group rows")

    tree_stems = sorted(
        p.stem for p in MIG.glob("*.can") if p.stem != "machinery"
    )
    if sorted(man) != tree_stems:
        only_man = sorted(set(man) - set(tree_stems))
        only_tree = sorted(set(tree_stems) - set(man))
        if only_man:
            fail(f"manifest stems without slice files: {only_man}")
        if only_tree:
            fail(f"slice files without manifest stems: {only_tree}")

    if FAIL:
        print("FAIL check-z01-partition:")
        for line in FAIL:
            print(f"  - {line}")
        return 1
    print(
        f"OK partition: {len(man)} stems cover {len(seen)} ledger rows exactly once"
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
