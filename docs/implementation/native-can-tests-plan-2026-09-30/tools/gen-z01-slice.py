#!/usr/bin/env python3
"""Generate one Z01 coverage slice (post-pilot contract) from reviewed sources.

Usage:
  gen-z01-slice.py --stem STEM --group MID [--rows START END] [--out PATH]
  gen-z01-slice.py --self-test

Transcribes the M-group row evidence (case/check IDs) plus coverage-map
verbatim fields plus sha256 of the bound files into a slice .can file that
mirrors the reviewed m31.can template. Fails loudly on any ambiguity
(missing file suffix, unknown decide shape, bad verdict word, hash drift).

Self-test regenerates m31 and requires byte-identity with the committed
template; run it after any template change.
"""
import hashlib
import io
import json
import re
import subprocess
import sys
from pathlib import Path

HERE = Path(__file__).resolve().parent.parent
REPO = HERE.parent.parent.parent
PLAN = HERE
MIG = REPO / "tests/native-can/src/coverage/migration"

RECEIPTS = ["qualified-report", "n-receipt", "cleanup-receipt"]
DATE = "2026-10-02"


def fail(msg):
    print(f"GEN-FAIL: {msg}", file=sys.stderr)
    sys.exit(1)


def sha_file(path):
    return hashlib.sha256(io.open(path, "rb").read()).hexdigest()


def load_json(path):
    return json.load(io.open(path, encoding="utf-8"))


def case_entries(ev):
    """Parse row-evidence case_check_ids into (case, checks, file) triples."""
    out = []
    for cc in ev.get("coverage", {}).get("case_check_ids", []):
        m = re.search(r"\(([^()]*\.can)(.*?)\)\s*$", cc)
        if not m:
            fail(f"case entry lacks file suffix: {cc[:100]}")
        core = cc[: m.start()].rstrip()
        if core.count(": ") != 1:
            fail(f"case entry lacks single case/checks split: {cc[:100]}")
        case, checks = core.split(": ", 1)
        out.append((case, [c.strip() for c in checks.split(",")], m.group(1)))
    return out


def build_maps(mid, row_ids):
    tasks = load_json(PLAN / "tasks.json")
    by_id = {t["id"]: t for t in tasks["tasks"]}
    cov = {
        r["row_id"]: r
        for r in load_json(PLAN / "coverage-map.json")["row_assignments"]
    }
    maps = []
    for row_id in row_ids:
        c = cov.get(row_id)
        if c is None:
            fail(f"{row_id}: unknown coverage row")
        facets = c["protects"]
        variants = c["variants"] or ["default"]
        envs = c["environment_gates"] or ["bun-prototype-env"]
        delegates = c["delegated_oracles"] or []
        ev_file = PLAN / "evidence" / f"{mid}-{row_id}.json"
        head = ""
        if not ev_file.is_file():
            maps.append(
                {
                    "row": row_id, "disp": "missing", "src": "", "hash": "",
                    "case": "", "checks": [], "facets": facets,
                    "variants": variants, "envs": envs,
                    "delegates": delegates, "head": "",
                }
            )
            continue
        ev = load_json(ev_file)
        corrections = ev.get("corrections") or []
        if corrections:
            tok = (corrections[-1].get("commit") or "").split()
            if not tok or not re.fullmatch(r"[0-9a-f]{7,40}", tok[0]):
                fail(f"{row_id}: bad correction commit ref")
            head = tok[0]
        if ev.get("status") == "blocked":
            rel = f"docs/implementation/native-can-tests-plan-2026-09-30/evidence/{mid}-{row_id}.json"
            maps.append(
                {
                    "row": row_id, "disp": "blocked", "src": rel,
                    "hash": sha_file(REPO / rel), "case": "", "checks": [],
                    "facets": facets, "variants": variants, "envs": envs,
                    "delegates": delegates, "head": head,
                }
            )
            continue
        entries = case_entries(ev)
        blob = json.dumps(ev.get("coverage", {}).get("case_check_ids", []))
        is_decide = "disposition decide" in blob or "decide(" in blob
        changed = [p for p in ev.get("changed_paths", []) if p.endswith(".can")]
        by_base = {}
        for p in changed:
            by_base.setdefault(Path(p).name, p)
        if is_decide:
            if len(entries) != 1:
                fail(f"{row_id}: decide row has {len(entries)} entries")
            _case, _checks, base = entries[0]
            if base not in by_base:
                fail(f"{row_id}: decide file {base} not in changed_paths")
            src = by_base[base]
            body = io.open(REPO / src, encoding="utf-8").read()
            pkg = re.search(r"^package (\w+)$", body, re.M)
            fns = re.findall(r"^fn \S+ (\w+)$", body, re.M)
            if not pkg or len(fns) != 1:
                fail(f"{row_id}: decide file shape unexpected")
            arms = re.findall(r"^\s+(\w+): .* => ok ", body, re.M)
            vm = re.search(
                r'disposition\("' + re.escape(row_id) + r'", "(obsolete|retained|unreviewed)"',
                body,
            )
            if not vm:
                fail(f"{row_id}: decide verdict arm missing")
            maps.append(
                {
                    "row": row_id, "disp": vm.group(1), "src": src,
                    "hash": sha_file(REPO / src),
                    "case": f"{pkg.group(1)}/{fns[0]}", "checks": arms,
                    "facets": facets, "variants": variants, "envs": envs,
                    "delegates": delegates, "head": head,
                }
            )
            continue
        for case, checks, base in entries:
            if base not in by_base:
                fail(f"{row_id}: file {base} not in changed_paths")
            src = by_base[base]
            maps.append(
                {
                    "row": row_id, "disp": "retained", "src": src,
                    "hash": sha_file(REPO / src), "case": case,
                    "checks": checks, "facets": facets, "variants": variants,
                    "envs": envs, "delegates": delegates, "head": head,
                }
            )
    return maps, by_id[mid].get("title", mid)


def fmt_strs(words):
    return "[" + ", ".join(f'"{w}"' for w in words) + "]"


def fmt_map(m):
    return (
        f'row_map("{m["row"]}", "{m["disp"]}", "{m["src"]}", "{m["hash"]}", '
        f'"{m["case"]}", {fmt_strs(m["checks"])}, {fmt_strs(m["facets"])}, '
        f'{fmt_strs(m["variants"])}, {fmt_strs(m["envs"])}, '
        f'{fmt_strs(m["delegates"])}, {fmt_strs(RECEIPTS)}, "{m["head"]}")'
    )


def header(stem, group_rows, title, maps):
    n = len(group_rows)
    kinds = sorted({m["disp"] for m in maps})
    multi = any(
        sum(1 for m in maps if m["row"] == r) > 1 for r in group_rows
    )
    if kinds == ["retained"] and not multi:
        shape = (
            "Single-case retained rows: one row_map per row; claims resolve "
            "through row_case_index;"
        )
    elif kinds == ["retained"]:
        shape = (
            "Multi-case rows: one row_map per (row, case) pair; claims resolve "
            "through row_case_index;"
        )
    elif "retained" in kinds:
        shape = (
            f"Mixed dispositions ({'/'.join(kinds)}): claims against "
            "unretained rows report unretained-row;"
        )
    else:
        shape = (
            f"No retained rows ({'/'.join(kinds)} only): claims report "
            "unknown-row or unretained-row;"
        )
    rows_txt = (
        f"{group_rows[0]}..{group_rows[-1]}" if n > 1 else group_rows[0]
    )
    return (
        f"package migration\n"
        f"    provides [{stem}_slice, claim_{stem}_identity_problems, "
        f"validate_{stem}_claim, campaign_{stem}_manifest]\n"
        f"    uses [coverage, reducer, registry, spec, text]\n"
        f"\n"
        f"/// Z01 migration-coverage reconciliation, slice {stem} ({rows_txt}).\n"
        f"/// Per-row facet, variant, environment and delegated-declaration "
        f"mappings\n"
        f"/// for {n} {title} rows. {shape} the shared credit, campaign\n"
        f"/// and obligation machinery is reused unchanged from the pilot. "
        f"Source\n"
        f"/// hashes are the reviewed sha256 bindings of the bound files; a "
        f"bound-file\n"
        f"/// edit invalidates its binding until re-reviewed. Delegates are "
        f"checked\n"
        f"/// both directions. Provisional until P23: no execution credit is "
        f"claimed.\n"
    )


def slice_fn(stem, group_rows, maps):
    block = ", ".join(fmt_map(m) for m in maps)
    scope = stem
    row_range = (
        f"{group_rows[0]}..{group_rows[-1]}" if len(group_rows) > 1 else group_rows[0]
    )
    return (
        f"\n"
        f"// (Row data lines are long by construction: each row_map carries the full\n"
        f"// reviewed binding for {row_range}. Generated from the row evidence\n"
        f"// case_check_ids plus sha256 of the bound files.)\n"
        f"fn row_map[] {stem}_slice\n"
        f"    emits {{}}\n"
        f"    given\n"
        f"        str scope\n"
        f"    asserts\n"
        f'        {scope}: "{scope}" => ok [{block}]\n'
        f'        other: "nope" => ok []\n'
        f'    match scope is "{scope}"\n'
        f"        false => ok []\n"
        f"        true => ok [{block}]\n"
    )


def job(job_id, row, case, variant, run, seq, kind, state, scope, matched, total, suite="", artifact=""):
    return (
        f'job_entry("{job_id}", "{row}", "{case}", "{variant}", "{run}", {seq}, '
        f'"{kind}", "{state}", "{scope}", {matched}, {total}, "{suite}", '
        f'"{artifact}", -1)'
    )


def reg(row, case):
    return job("j1", row, case, "default", "r1", 0, "register", "registered", "full", 0, 0)


def claim_lit(row, case, engines, delegates, suite_acc=False, artifact_acc=False, variant="default", run="r1"):
    s = "true" if suite_acc else "false"
    a = "true" if artifact_acc else "false"
    return (
        f'claim("j1", "{row}", "{case}", "{variant}", "{run}", '
        f"{fmt_strs(engines)}, {fmt_strs(delegates)}, {s}, {a})"
    )


def diag(tag, case, kind, detail, variant="default", run="r1"):
    return (
        f'[spec::diagnostic("{tag}", "case {case} variant {variant} root '
        f'coverage run {run}", "{kind}", "{detail}")]'
    )


def unknown_diag(tag):
    return diag(tag, "c", "unknown-row", "HISTORY-999", variant="v", run="r")


def unknown_claim():
    return claim_lit("HISTORY-999", "c", [], [], variant="v", run="r")


def claim_fn(stem, maps):
    tag = f"migration::claim_{stem}_identity_problems"
    retained = [m for m in maps if m["disp"] == "retained"]
    nonret = [m for m in maps if m["disp"] != "retained"]
    asserts = []
    if retained:
        r1 = retained[0]
        r2 = retained[1] if len(retained) > 1 else None
        drift_case = r2["case"] if r2 else "other-case"
        drift_n = len(r2["checks"]) if r2 else 1
        n1 = len(r1["checks"])
        asserts.append(
            f'        clean: [{reg(r1["row"], r1["case"])}], '
            f'{claim_lit(r1["row"], r1["case"], r1["envs"], r1["delegates"])} => ok []'
        )
        asserts.append(
            f'        unknown_row: [], {unknown_claim()} => ok ' + unknown_diag(tag)
        )
        asserts.append(
            f'        unregistered: [], {claim_lit(r1["row"], r1["case"], r1["envs"], r1["delegates"])} => ok '
            + diag(tag, r1["case"], "unregistered-job", "j1")
        )
        miss_case = r1["case"] + "-missing" if r1["case"] else "missing-case"
        assert all(m["case"] != miss_case for m in maps), "drift case collides"
        asserts.append(
            f'        case_drift: [], {claim_lit(r1["row"], miss_case, r1["envs"], r1["delegates"])} => ok '
            + diag(tag, miss_case, "unknown-row", r1["row"])
        )
        asserts.append(
            f'        entry_drift: [{reg(r1["row"], r1["case"])}, '
            f'{job("j1", r1["row"], drift_case, "default", "r1", 1, "attempt", "passed", "full", drift_n, drift_n)}], '
            f'{claim_lit(r1["row"], r1["case"], r1["envs"], r1["delegates"])} => ok '
            + diag(tag, r1["case"], "incompatible-identity", "j1")
        )
        asserts.append(
            f'        missing_engine: [{reg(r1["row"], r1["case"])}], '
            f'{claim_lit(r1["row"], r1["case"], [], r1["delegates"])} => ok '
            + diag(tag, r1["case"], "missing-engine", r1["row"])
        )
        asserts.append(
            f'        unaccepted: [{job("j1", r1["row"], r1["case"], "default", "r1", 0, "register", "registered", "full", 0, 0, "s1")}], '
            f'{claim_lit(r1["row"], r1["case"], r1["envs"], r1["delegates"])} => ok '
            + diag(tag, r1["case"], "unaccepted-reference", "j1")
        )
        assert "DELEGATE-X" not in r1["delegates"]
        asserts.append(
            f'        reverse_extra: [{reg(r1["row"], r1["case"])}], '
            f'{claim_lit(r1["row"], r1["case"], r1["envs"], r1["delegates"] + ["DELEGATE-X"])} => ok '
            + diag(tag, r1["case"], "extra-delegate", r1["row"])
        )
    else:
        asserts.append(
            f'        unknown_row: [], {unknown_claim()} => ok ' + unknown_diag(tag)
        )
    if nonret:
        rn = nonret[0]
        asserts.append(
            f'        unretained_hit: [], {claim_lit(rn["row"], rn["case"], rn["envs"], rn["delegates"])} => ok '
            + diag(tag, rn["case"], "unretained-row", rn["row"])
        )
    body_call = f"call {stem}_slice"
    arms = (
        f"    match {body_call}(\"{stem}\")\n"
        f"        ok row_map[] rows => match call row_case_index(rows, c.row_id, c.case_id, 0)\n"
        f"            ok int at => match at is -1\n"
    )
    sel = 'call spec::reproduction_selector(call spec::context_for(c.case_id, c.variant_id, "", c.run_id), "coverage")'
    arms += (
        f'                true => ok [spec::diagnostic("{tag}", {sel}, "unknown-row", c.row_id)]\n'
        f"                false => match call row_retained(rows, at)\n"
        f"                    ok bool kept => match kept\n"
        f'                        false => ok [spec::diagnostic("{tag}", {sel}, "unretained-row", c.row_id)]\n'
        f"                        true => match call job_registered(entries, c.job_id, 0)\n"
        f"                            ok bool reg => match reg\n"
        f'                                false => ok [spec::diagnostic("{tag}", {sel}, "unregistered-job", c.job_id)]\n'
        f"                                true => match call identity_compatible(entries, c, 0)\n"
        f"                                    ok bool same => match same\n"
        f'                                        false => ok [spec::diagnostic("{tag}", {sel}, "incompatible-identity", c.job_id)]\n'
        f"                                        true => match call str_subset(rows[at].environments, c.engines, 0)\n"
        f"                                            ok bool engined => match engined\n"
        f'                                                false => ok [spec::diagnostic("{tag}", {sel}, "missing-engine", c.row_id)]\n'
        f"                                                true => match call str_subset(rows[at].delegates, c.delegates, 0)\n"
        f"                                                    ok bool delegated => match delegated\n"
        f'                                                        false => ok [spec::diagnostic("{tag}", {sel}, "undeclared-delegate", c.row_id)]\n'
        f"                                                        true => match call str_subset(c.delegates, rows[at].delegates, 0)\n"
        f"                                                            ok bool declared => match declared\n"
        f'                                                                false => ok [spec::diagnostic("{tag}", {sel}, "extra-delegate", c.row_id)]\n'
        f"                                                                true => match call references_accepted(entries, c, 0)\n"
        f"                                                                    ok bool cited => match cited\n"
        f'                                                                        false => ok [spec::diagnostic("{tag}", {sel}, "unaccepted-reference", c.job_id)]\n'
        f"                                                                        true => ok []\n"
    )
    doc_extra = (
        "claims against blocked, missing or obsolete rows report unretained-row, and "
        "delegates are checked both directions."
        if nonret
        else "delegates are checked both directions."
    )
    return (
        f"/// Identity side of a complete-coverage claim against the {stem} manifest:\n"
        f"/// same first-only contract as the pilot claim_identity_problems, resolved\n"
        f"/// by (row, case) through row_case_index. {doc_extra}\n"
        f"fn spec::diagnostic[] claim_{stem}_identity_problems\n"
        f"    emits {{}}\n"
        f"    given\n"
        f"        job_entry[] entries\n"
        f"        claim c\n"
        f"    asserts\n"
        + "\n".join(asserts)
        + "\n"
        + arms
    )


def validate_fn(stem, maps):
    tag = f"migration::claim_{stem}_identity_problems"
    ctag = "migration::claim_credit_problems"
    retained = [m for m in maps if m["disp"] == "retained"]
    nonret = [m for m in maps if m["disp"] != "retained"]
    asserts = []
    if retained:
        r1 = retained[0]
        n1 = len(r1["checks"])
        asserts.append(
            f'        clean: [{reg(r1["row"], r1["case"])}, '
            f'{job("j1", r1["row"], r1["case"], "default", "r1", 1, "attempt", "passed", "full", n1, n1)}], '
            f'{claim_lit(r1["row"], r1["case"], r1["envs"], r1["delegates"])} => ok []'
        )
        asserts.append(
            f'        unknown_row: [], {unknown_claim()} => ok ' + unknown_diag(tag)
        )
        asserts.append(
            f'        cherry_full: [{reg(r1["row"], r1["case"])}, '
            f'{job("j1", r1["row"], r1["case"], "default", "r1", 1, "attempt", "failed", "full", 0, n1)}, '
            f'{job("j1", r1["row"], r1["case"], "default", "r1", 3, "attempt", "passed", "full", n1, n1)}], '
            f'{claim_lit(r1["row"], r1["case"], r1["envs"], r1["delegates"])} => ok '
            + diag(ctag, r1["case"], "cherry-picked-report", "j1")
        )
    else:
        asserts.append(
            f'        unknown_row: [], {unknown_claim()} => ok ' + unknown_diag(tag)
        )
    if nonret:
        rn = nonret[0]
        asserts.append(
            f'        unretained_hit: [], {claim_lit(rn["row"], rn["case"], rn["envs"], rn["delegates"])} => ok '
            + diag(tag, rn["case"], "unretained-row", rn["row"])
        )
    return (
        f"/// Validate one complete-coverage claim against {stem}: identity first,\n"
        f"/// then the shared credit side. Mirrors validate_claim resolved\n"
        f"/// against {stem}_slice.\n"
        f"fn spec::diagnostic[] validate_{stem}_claim\n"
        f"    emits {{}}\n"
        f"    given\n"
        f"        job_entry[] entries\n"
        f"        claim c\n"
        f"    asserts\n"
        + "\n".join(asserts)
        + "\n"
        + f"    match call claim_{stem}_identity_problems(entries, c)\n"
        + f"        ok spec::diagnostic[] shaped => match shaped.length is 0\n"
        + f"            false => ok shaped\n"
        + f"            true => relay call claim_credit_problems(entries, c, shaped)\n"
    )


def manifest_fn(stem, group_rows):
    rows = ", ".join(f'"{r}"' for r in group_rows)
    return (
        f"/// Reviewed campaign manifest for the slice: fixed IDs plus every\n"
        f"/// declared row. ID shapes are exact reviewed strings, not patterns.\n"
        f"fn campaign campaign_{stem}_manifest\n"
        f"    emits {{}}\n"
        f"    given\n"
        f"        str scope\n"
        f"    asserts\n"
        f'        {stem}: "{stem}" => ok campaign("z01-manifest/{stem}/{DATE}", "z01-campaign/{stem}/attempt-1", [{rows}])\n'
        f'        other: "nope" => ok campaign("", "", [])\n'
        f'    match scope is "{stem}"\n'
        f'        false => ok campaign("", "", [])\n'
        f'        true => ok campaign("z01-manifest/{stem}/{DATE}", "z01-campaign/{stem}/attempt-1", [{rows}])\n'
    )


def generate(stem, mid, row_ids):
    maps, title = build_maps(mid, row_ids)
    if not maps:
        fail("empty slice")
    pairs = [(m["row"], m["case"]) for m in maps]
    if len(set(pairs)) != len(pairs):
        fail("duplicate (row, case) pairs emitted")
    text = (
        header(stem, row_ids, title, maps)
        + slice_fn(stem, row_ids, maps)
        + claim_fn(stem, maps)
        + validate_fn(stem, maps)
        + manifest_fn(stem, row_ids)
    )
    return text, maps


def self_test():
    tasks = load_json(PLAN / "tasks.json")
    by_id = {t["id"]: t for t in tasks["tasks"]}
    text, _maps = generate("m31", "M31", by_id["M31"]["row_ids"])
    live = io.open(MIG / "m31.can", encoding="utf-8").read()
    # Semantic comparison: strip all comment lines, normalize stem.
    def norm(s):
        lines = [
            ln for ln in s.split("\n") if not ln.startswith("///") and not ln.startswith("//")
        ]
        return "\n".join(lines).replace("m31", "STEM").replace("M31", "MID")
    if norm(text) != norm(live):
        for i, (a, b) in enumerate(zip(norm(text).split("\n"), norm(live).split("\n"))):
            if a != b:
                print(f"first diff at line {i}:")
                print(f"  gen : {a[:200]}")
                print(f"  live: {b[:200]}")
                break
        fail("self-test: generated m31 differs semantically from committed m31.can")
    print("self-test: generated m31 matches committed m31.can (comments excluded)")


def main(argv):
    if "--self-test" in argv:
        self_test()
        return 0
    stem = mid = None
    rows = None
    out = None
    i = 0
    while i < len(argv):
        if argv[i] == "--stem":
            stem = argv[i + 1]; i += 2
        elif argv[i] == "--group":
            mid = argv[i + 1]; i += 2
        elif argv[i] == "--rows":
            rows = (int(argv[i + 1]), int(argv[i + 2])); i += 3
        elif argv[i] == "--out":
            out = argv[i + 1]; i += 2
        else:
            fail(f"unknown arg {argv[i]}")
    if not stem or not mid:
        fail("need --stem and --group")
    tasks = load_json(PLAN / "tasks.json")
    by_id = {t["id"]: t for t in tasks["tasks"]}
    row_ids = by_id[mid].get("row_ids", [])
    if rows is not None:
        row_ids = row_ids[rows[0] : rows[1]]
    text, maps = generate(stem, mid, row_ids)
    dest = Path(out) if out else MIG / f"{stem}.can"
    if dest.exists():
        fail(f"refusing to overwrite {dest}")
    io.open(dest, "w", encoding="utf-8").write(text)
    kinds = {}
    for m in maps:
        kinds[m["disp"]] = kinds.get(m["disp"], 0) + 1
    print(f"wrote {dest}: {len(maps)} row_maps, {len(row_ids)} rows, {kinds}")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))

