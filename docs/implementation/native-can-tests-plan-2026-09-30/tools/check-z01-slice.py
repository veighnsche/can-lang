#!/usr/bin/env python3
"""Conformance gate for one Z01 coverage slice (post-pilot contract).

Usage: python3 tools/check-z01-slice.py <slice-can-file> <evidence-json>
Exit 0 when the slice file + evidence conform; prints FAIL lines otherwise.

Checks (all mechanical, no judgment):
- package/provides/uses shape; provides disjoint across the migration package
- covered rows == the reviewed manifest entry exactly (no dropped row passes);
  campaign rows == manifest rows in order; (row, case) pairs unique
- dispositions consistent with row-evidence kind (port/decide/blocked/missing)
- sha256 bindings match disk (ports, decide files, capture evidence)
- case/check IDs identical (IDs and order) to row evidence; decide cases are
  package/fn with assert-label checks verified in the decide file; every
  assert-block line matches the arm shape (no silently dropped arm)
- decide file verdict agrees in assert + body and with the evidence prose
- bound source is listed in the row evidence changed_paths (port/decide rows)
- facets/variants/environments/delegates verbatim vs coverage-map.json
  (["default"] / ["bun-prototype-env"] fallback where empty); receipts triple
- campaign manifest IDs/rows match the evidence json; evidence case lines
  match the slice (tamper check between the two files)
- claim/validate fns contain the required resolution/gate/diagnostic arms,
  with all 8 diagnostic kinds present in the claim body (not just asserts)

Authority chain: z01-slices.json assigns rows to stems (reviewed);
check-z01-partition.py verifies the manifest partitions the 292-row ledger.
"""
import hashlib
import io
import json
import re
import sys

HERE = __import__("pathlib").Path(__file__).resolve().parent.parent
REPO = HERE.parent.parent.parent
PLAN = HERE
MIG = REPO / "tests/native-can/src/coverage/migration"

FAIL = []


def fail(msg):
    FAIL.append(msg)


def strs(s):
    return re.findall(r'"([^"]*)"', s)


# List items are quoted strings that may contain brackets (verbatim coverage
# text); match item-wise since no item ever contains a quote (the
# generated Can would not parse otherwise).
LIST = r"\[((?:\"[^\"]*\"(?:, )?)*)\]"
ROW_RE = re.compile(
    r'row_map\("([^"]+)", "([^"]+)", "([^"]*)", "([0-9a-f]{64}|)", "([^"]*)", '
    rf"{LIST}, {LIST}, {LIST}, {LIST}, {LIST}, {LIST}, "
    r'"([^"]*)"\)'
)


def load_slice(path):
    src = io.open(path, encoding="utf-8").read()
    stem = __import__("pathlib").Path(path).stem  # m31 / m30a / machinery
    return src, stem


def check_package(src, stem):
    m = re.search(r"^package migration$", src, re.M)
    if not m:
        fail("package line is not 'package migration'")
    m = re.search(r"^    provides \[(.*)\]$", src, re.M)
    if not m:
        fail("provides line missing")
        return []
    provides = [p.strip() for p in m.group(1).split(",")]
    if stem == "machinery":
        want = ["row_case_index", "row_retained"]
    else:
        want = [
            f"{stem}_slice",
            f"claim_{stem}_identity_problems",
            f"validate_{stem}_claim",
            f"campaign_{stem}_manifest",
        ]
    if provides != want:
        fail(f"provides {provides} != {want}")
    if "uses [coverage, reducer, registry, spec, text]" not in src:
        fail("uses line differs from the pilot contract")
    return provides


def check_provides_disjoint(stem, provides):
    for other in sorted(MIG.glob("*.can")):
        if other.stem == stem:
            continue
        src = io.open(other, encoding="utf-8").read()
        m = re.search(r"^    provides \[(.*)\]$", src, re.M)
        if not m:
            continue
        theirs = {p.strip() for p in m.group(1).split(",")}
        clash = set(provides) & theirs
        if clash:
            fail(f"provides clash with {other.name}: {sorted(clash)}")


KINDS_8 = (
    "unknown-row",
    "unretained-row",
    "unregistered-job",
    "incompatible-identity",
    "missing-engine",
    "undeclared-delegate",
    "extra-delegate",
    "unaccepted-reference",
)


def check_claim_shape(src, stem, has_retained, has_nonret):
    tag = f"migration::claim_{stem}_identity_problems"
    if tag not in src:
        fail("claim fn tag missing from slice")
        return
    need_calls = [
        f"call {stem}_slice(",
        "call row_case_index(",
        "call row_retained(",
        "call job_registered(",
        "call identity_compatible(",
        "call str_subset(rows[at].environments, c.engines",
        "call str_subset(rows[at].delegates, c.delegates",
        "call str_subset(c.delegates, rows[at].delegates",
        "call references_accepted(",
    ]
    for call in need_calls:
        if call not in src:
            fail(f"claim fn lacks {call}")
    try:
        claim_start = src.index(f"fn spec::diagnostic[] claim_{stem}_identity_problems")
        body_start = src.index(f"match call {stem}_slice(", claim_start)
        validate_start = src.index(
            "/// Validate one complete-coverage claim", body_start
        )
    except ValueError:
        fail("claim fn region markers missing or out of order")
        return
    asserts_region = src[claim_start:body_start]
    body_region = src[body_start:validate_start]
    for kind in KINDS_8:
        if f'"{kind}"' not in body_region:
            fail(f"claim body lacks diagnostic kind {kind}")
    # Assert-side presence is conditional by template design: the retained
    # block emits 6 of the 8 kinds, unretained_hit emits unretained-row, and
    # undeclared-delegate is body-only (no subset-direction assert exists).
    want_asserts = {"unknown-row"}
    if has_retained:
        want_asserts |= {
            "unregistered-job",
            "incompatible-identity",
            "missing-engine",
            "extra-delegate",
            "unaccepted-reference",
        }
    if has_nonret:
        want_asserts.add("unretained-row")
    for kind in sorted(want_asserts):
        if f'"{kind}"' not in asserts_region:
            fail(f"claim asserts lack diagnostic kind {kind}")
    if f"call claim_{stem}_identity_problems(entries, c)" not in src:
        fail("validate fn does not call the slice claim fn")
    if "call claim_credit_problems(entries, c, shaped)" not in src:
        fail("validate fn does not call shared claim_credit_problems")


def finish(can_path, n_rows=0, n_covered=0):
    if FAIL:
        print(f"FAIL {can_path}:")
        for line in FAIL:
            print(f"  - {line}")
        return 1
    print(f"OK {can_path}: {n_rows} row_maps, {n_covered} rows conform")
    return 0


def run(can_path, ev_path):
    src, stem = load_slice(can_path)
    man = json.load(io.open(PLAN / "z01-slices.json", encoding="utf-8"))
    entry = man.get(stem)
    if entry is None:
        fail(f"stem {stem} has no z01-slices.json entry")
        return finish(can_path)
    mid = entry["group"]
    expected = entry["rows"]
    tasks = json.load(io.open(PLAN / "tasks.json", encoding="utf-8"))
    by_id = {t["id"]: t for t in tasks["tasks"]}
    if mid not in by_id:
        fail(f"manifest group {mid} unknown to tasks.json")
        return finish(can_path)
    provides = check_package(src, stem)
    if provides:
        check_provides_disjoint(stem, provides)

    cov = {
        r["row_id"]: r
        for r in json.load(io.open(PLAN / "coverage-map.json", encoding="utf-8"))[
            "row_assignments"
        ]
    }
    maps = ROW_RE.findall(src)
    seen = set()
    rows = []
    for m in maps:
        if m not in seen:
            seen.add(m)
            rows.append(m)
    if len(maps) != 2 * len(rows):
        fail(f"assert/body mirror mismatch: {len(maps)} literals, {len(rows)} distinct")
    pairs = [(r[0], r[4]) for r in rows]
    if len(set(pairs)) != len(pairs):
        fail("duplicate (row, case) pairs")
    covered = {r[0] for r in rows}
    if stem != "machinery":
        check_claim_shape(
            src, stem, any(r[1] == "retained" for r in rows), any(r[1] != "retained" for r in rows)
        )
    m = re.search(
        r'campaign\("([^"]+)", "([^"]+)", \[([^\]]*)\]\)', src
    )
    if not m:
        fail("campaign manifest literal missing")
        return finish(can_path)
    manifest_id, campaign_id, camp_rows = m.group(1), m.group(2), strs(m.group(3))
    if len(camp_rows) != len(set(camp_rows)):
        fail("campaign manifest declares duplicate rows")
    if camp_rows != expected:
        fail(f"campaign rows != manifest {stem} rows (IDs or order)")
    if covered != set(expected):
        fail(
            f"mapped rows {sorted(covered)} != manifest {stem} rows "
            f"{sorted(set(expected))}"
        )

    for row in rows:
        (row_id, disp, source, h, case, checks, facets, variants, envs, delegates, receipts, head) = row
        c = cov.get(row_id)
        if c is None:
            fail(f"{row_id}: unknown coverage row")
            continue
        if strs(facets) != c["protects"]:
            fail(f"{row_id}: facets not verbatim protects")
        if strs(variants) != (c["variants"] or ["default"]):
            fail(f"{row_id}: variants not verbatim (+default rule)")
        if strs(envs) != (c["environment_gates"] or ["bun-prototype-env"]):
            fail(f"{row_id}: environments not verbatim (+env rule)")
        if strs(delegates) != (c["delegated_oracles"] or []):
            fail(f"{row_id}: delegates not verbatim")
        if strs(receipts) != ["qualified-report", "n-receipt", "cleanup-receipt"]:
            fail(f"{row_id}: receipts differ from the standard triple")
        ev_file = PLAN / "evidence" / f"{mid}-{row_id}.json"
        want_head = ""
        if ev_file.is_file():
            ev0 = json.load(io.open(ev_file, encoding="utf-8"))
            corrections = ev0.get("corrections") or []
            if corrections:
                toks = (corrections[-1].get("commit") or "").split()
                want_head = toks[0] if toks else ""
        if head != want_head:
            fail(f"{row_id}: correction_head {head!r} != latest correction commit {want_head!r}")
        ev_file = PLAN / "evidence" / f"{mid}-{row_id}.json"
        if not ev_file.is_file():
            if disp != "missing" or source or h or case or strs(checks):
                fail(f"{row_id}: missing-evidence row must map empty with disposition missing")
            continue
        ev = json.load(io.open(ev_file, encoding="utf-8"))
        ccs = ev.get("coverage", {}).get("case_check_ids", [])
        if ev.get("status") == "blocked":
            if disp != "blocked":
                fail(f"{row_id}: blocked evidence needs disposition blocked")
            want_src = f"docs/implementation/native-can-tests-plan-2026-09-30/evidence/{mid}-{row_id}.json"
            if source != want_src:
                fail(f"{row_id}: blocked source must be the capture evidence path")
            disk = hashlib.sha256(io.open(REPO / source, "rb").read()).hexdigest()
            if disk != h:
                fail(f"{row_id}: blocked capture hash mismatch")
            if case or strs(checks):
                fail(f"{row_id}: blocked row must carry no case/checks")
            continue
        disk_body = io.open(REPO / source, encoding="utf-8").read()
        is_decide = "provides [decide, disposition]" in disk_body and bool(
            re.search(r"^fn disposition decide$", disk_body, re.M)
        )
        if source not in (ev.get("changed_paths") or []):
            fail(f"{row_id}: bound source absent from row-evidence changed_paths")
        if is_decide:
            disk = hashlib.sha256(disk_body.encode("utf-8")).hexdigest()
            if disk != h:
                fail(f"{row_id}: decide file hash mismatch")
            verdicts = re.findall(
                r'disposition\("' + re.escape(row_id) + r'", "([a-z-]+)"',
                disk_body,
            )
            if not verdicts:
                fail(f"{row_id}: decide verdict missing from file")
                continue
            if len(set(verdicts)) != 1:
                fail(f"{row_id}: decide assert/body verdicts disagree {sorted(set(verdicts))}")
                continue
            verdict = verdicts[0]
            if verdict not in ("obsolete", "preserve", "retain-fixture"):
                fail(f"{row_id}: decide verdict outside vocabulary")
            elif disp != verdict:
                fail(f"{row_id}: decide disposition {disp} != file verdict {verdict}")
            prose = " ".join(ccs)
            cands = re.findall(
                re.escape(row_id) + r" -> ([a-z-]+)", prose
            ) + re.findall(r"\('" + re.escape(row_id) + r"','([a-z-]+)',", prose)
            if not cands:
                fail(f"{row_id}: no prose verdict for the row in the evidence entry")
            elif any(c != verdict for c in cands):
                fail(f"{row_id}: evidence prose verdict {cands} != file verdict {verdict}")
            pkg = re.search(r"^package (\w+)$", disk_body, re.M)
            fn = re.search(r"^fn \w+ (\w+)$", disk_body, re.M)
            if not pkg or not fn or case != f"{pkg.group(1)}/{fn.group(1)}":
                fail(f"{row_id}: decide case must be package/fn, got {case!r}")
            am = re.search(r"^    asserts\n(.*?)^    match ", disk_body, re.M | re.S)
            if not am:
                fail(f"{row_id}: decide file has no asserts block")
                continue
            for ln in am.group(1).split("\n"):
                if ln.strip() and not re.match(r"^\s+(\w+): .* => ok ", ln):
                    fail(f"{row_id}: decide assert line outside the arm shape: {ln[:80]!r}")
                    break
            arms = re.findall(r"^\s+(\w+): .* => ok ", disk_body, re.M)
            if strs(checks) != arms:
                fail(f"{row_id}: decide checks must be the assert arms {arms}")
            continue
        if disp != "retained":
            fail(f"{row_id}: behavior-port row needs disposition retained")
        disk = hashlib.sha256(io.open(REPO / source, "rb").read()).hexdigest()
        if disk != h:
            fail(f"{row_id}: port file hash mismatch")
        body = io.open(REPO / source, encoding="utf-8").read()
        if case not in body:
            fail(f"{row_id}: case {case} absent from port")
        for cid in strs(checks):
            if cid not in body:
                fail(f"{row_id}: check {cid} absent from port")
        ev_cases = {}
        for cc in ccs:
            head_txt, rest = cc.split(": ", 1)
            # Port entries carry a "(file, package ...)" suffix; strip it.
            if " (" in rest and rest.rstrip().endswith(")"):
                rest = rest[: rest.rindex(" (")]
            ev_cases[head_txt] = [x.strip() for x in rest.split(",")]
        if case not in ev_cases:
            fail(f"{row_id}: case {case} absent from row evidence")
        elif ev_cases[case] != strs(checks):
            fail(f"{row_id}: checks differ from row evidence (IDs or order)")

    ev = json.load(io.open(ev_path, encoding="utf-8"))
    for key in ("status", "campaign", "commands_with_deadlines_and_results", "coverage", "controls"):
        if key not in ev:
            fail(f"evidence lacks {key}")
    if ev.get("status") != "slice-accepted-source-only":
        fail("evidence status must be slice-accepted-source-only")
    if ev.get("campaign", {}).get("manifest_id") != manifest_id:
        fail("evidence manifest_id differs from the slice")
    if ev.get("campaign", {}).get("campaign_id") != campaign_id:
        fail("evidence campaign_id differs from the slice")
    if ev.get("campaign", {}).get("declared_rows") != camp_rows:
        fail("evidence declared_rows differ from the slice")
    if not re.fullmatch(r"z01-manifest/[a-z0-9-]+/2026-10-0[12]", manifest_id):
        fail(f"manifest_id shape unexpected: {manifest_id}")
    if not re.fullmatch(r"z01-campaign/[a-z0-9-]+/attempt-1", campaign_id):
        fail(f"campaign_id shape unexpected: {campaign_id}")
    want_lines = [
        f"{r} {c or '(no case)'}: {len(strs(ch))} checks"
        for (r, _d, _s, _h, c, ch, *_rest) in rows
    ]
    if ev.get("coverage", {}).get("case_check_ids") != want_lines:
        fail("evidence case_check_ids differ from the slice row_maps")
    return finish(can_path, len(rows), len(covered))


def main():
    if len(sys.argv) != 3:
        print("usage: check-z01-slice.py <slice.can> <z01-evidence.json>")
        sys.exit(2)
    try:
        return run(sys.argv[1], sys.argv[2])
    except Exception as e:
        print(f"FAIL {sys.argv[1]}:")
        print(f"  - gate crashed: {type(e).__name__}: {e}")
        return 1


if __name__ == "__main__":
    sys.exit(main())
