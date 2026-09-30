"""Read-only source audit and documentation renderer; never executes the test suite."""
import collections
import hashlib
import json
import pathlib
import re
import subprocess

HERE = pathlib.Path(__file__).resolve().parent
ROOT = HERE.parents[3]
PARTS = ["core", "browser", "sql-history", "lifecycle", "external-callers"]
INDEX = json.loads((HERE / "source-index.json").read_text())
SCHEMA = json.loads((HERE / "schema.json").read_text())
GROUPS = {part: json.loads((HERE / (part + ".json")).read_text())["rows"] for part in PARTS}
ROWS = [row for rows in GROUPS.values() for row in rows]
OWNER = {row["id"]: part for part, rows in GROUPS.items() for row in rows}


def write_json(name, value):
    (HERE / name).write_text(json.dumps(value, indent=2, ensure_ascii=False) + "\n")


def link(path, line=None):
    # Absolute links support the desktop source editor and line navigation.
    return f"[{path}]({ROOT / path}{':' + str(line) if line else ''})"


def rowlink(row_id):
    return f"[{row_id}]({OWNER[row_id]}.md#{row_id.lower()})"


def sentence_list(items):
    return " ".join(items) if items else "None recorded."


errors = []
ids = [r["id"] for r in ROWS]
if len(ids) != len(set(ids)):
    errors.append("duplicate row IDs")
for r in ROWS:
    missing = set(SCHEMA["row_fields"]) - r.keys()
    if missing:
        errors.append(f"{r['id']}: missing {sorted(missing)}")
    p = ROOT / r["source"]
    if not p.is_file() or not 1 <= r["line"] <= len(p.read_text().splitlines()):
        errors.append(f"{r['id']}: invalid source anchor")
    for ref in r["related_sources"]:
        if not (ROOT / ref).exists():
            errors.append(f"{r['id']}: missing related source {ref}")
    unknown = set(r["capabilities"]) - SCHEMA["capabilities"].keys()
    if unknown:
        errors.append(f"{r['id']}: unknown capabilities {unknown}")
for kind, expected in [
    ("go_test", [x for x in INDEX["go_entrypoints"] if x["kind"] == "go_test"]),
    ("bun_test", INDEX["bun_declarations"]),
]:
    actual = collections.Counter((r["source"], r["line"]) for r in ROWS if r["kind"] == kind)
    wanted = {(r["source"], r["line"]) for r in expected}
    if set(actual) != wanted or any(n != 1 for n in actual.values()):
        errors.append(f"{kind}: missing={wanted - actual.keys()}, extra={actual.keys() - wanted}, duplicates={[k for k,n in actual.items() if n != 1]}")
for x in INDEX["go_entrypoints"]:
    if x["symbol"] == "TestMain" and not any(r["source"] == x["source"] and r["symbol"] == "TestMain" and r["line"] == x["line"] for r in ROWS):
        errors.append(f"unmapped TestMain: {x}")
for f in INDEX["files"]:
    if hashlib.sha256((ROOT / f["path"]).read_bytes()).hexdigest() != f["sha256"]:
        errors.append(f"source changed since inventory: {f['path']}")
tracked_now = set(subprocess.check_output(["git", "ls-files", "tests/"], cwd=ROOT, text=True).splitlines())
captured_paths = {f["path"] for f in INDEX["files"]}
if tracked_now != captured_paths:
    errors.append(f"tracked tests changed: added={tracked_now - captured_paths}, removed={captured_paths - tracked_now}")

# Every tracked source gets a role and a route into the semantic ledger. Family
# links are explicitly labelled context, never misrepresented as direct callers.
file_rows = []
for f in INDEX["files"]:
    path = f["path"]
    direct = [r["id"] for r in ROWS if r["source"] == path]
    related = [r["id"] for r in ROWS if any(path == ref or path.startswith(ref.rstrip("/") + "/") for ref in r["related_sources"])]
    family = "/".join(path.split("/")[:2])
    context = []
    if not direct and not related:
        if family != "tests/integration":
            context = [r["id"] for r in ROWS if r["source"].startswith(family + "/") and r["kind"] in ("go_test", "historical", "support")]
        else:
            fixture = path.split("/")[3] if "/testdata/" in path else ""
            families = {
                "applications": ["applications_test.go"],
                "browser-conformance": ["browser_controls_test.go"],
                "browser-controls": ["browser_controls_test.go"],
                "browser-controls-native": ["browser_controls_test.go"],
                "formats": ["formats_test.go"],
                "gallery-failing": ["applications_test.go"],
                "s3": ["s3_test.go"],
                "invoice": ["invoice_test.go", "invoice_contract_test.go", "invoice_grid_test.go"],
                "sql": ["sql_test.go", "sql_f02_test.go", "sql_f03_test.go", "sql_descriptors_test.go", "mysql_test.go", "sqlite_test.go"],
                "webhook": ["webhook_test.go", "webhook_f06_test.go"],
            }
            context = [r["id"] for r in ROWS if pathlib.Path(r["source"]).name in families.get(fixture, [])]
    suffix = pathlib.Path(path).suffix
    if suffix == ".go" or path.endswith(".test.ts"):
        role = "authored host tests/support"
        retirement = "Apply every linked row's gate; migrate active callers before deleting shared source."
    elif suffix in (".ts", ".mjs", ".sh"):
        role = "historical prototype subject" if path.startswith("tests/host-discrimination/") else "authored host scenario/probe/support"
        retirement = "Read linked row: Can owns any retained sequence and verdict. Historical prototype subjects need a relevance decision; generic mechanics may survive with no test policy."
        if not direct:
            errors.append(f"executable source lacks a dedicated row: {path}")
    elif suffix == ".can":
        role = "Can subject/fixture"
        retirement = "Retain or adapt source needed by retained obligations; already being Can does not replace its foreign scenario and oracle. Historical comparison inputs may be archived with their decision."
    elif suffix in (".sql", ".bin", ".html") or path.endswith(("can.project.json", "can.errors.json")):
        role = "static fixture/configuration"
        retirement = "Keep as explicit input/seed/expected data while a retained case uses it. Remove only after its consumers and coverage obligation are resolved."
    elif path.endswith(("package.json", "bun.lock")):
        role = "browser dependency provisioning metadata"
        retirement = "Retain or replace only for the selected generic browser infrastructure; update external host/conformance consumers before removal."
    else:
        role = "documentation/registry/decision evidence"
        retirement = "Preserve provenance where useful; update active commands/links. A dated document does not by itself retire executable checks linked to it."
    if not direct and not related and not context:
        errors.append(f"unmapped file: {path}")
    file_rows.append({**f, "role": role, "direct_rows": direct, "related_rows": related,
                      "family_context_rows": context, "retirement_condition": retirement})
write_json("file-map.json", {"status": "proposed dispositions; no files retired", "files": file_rows})

# Follow concrete runtime test invocations, including the two computed path
# forms. Mere comments about other unit suites are deliberately excluded.
delegates = collections.defaultdict(list)
for p in (ROOT / "tests/integration").glob("*.go"):
    for number, line in enumerate(p.read_text().splitlines(), 1):
        if line.strip().startswith("//"):
            continue
        refs = re.findall(r'"(runtime/(?:test/)?[^"\n]+\.test\.ts)"', line)
        if p.name == "gate3_matrix_test.go" and '"request-lifetime.test.ts"' in line:
            refs.append("runtime/test/request-lifetime.test.ts")
        if p.name == "gate4_fault_test.go" and "suiteFiles :=" in line:
            refs += ["runtime/test/" + name for name in re.findall(r'"([^"\n]+\.test\.ts)"', line)]
        for ref in refs:
            delegates[ref].append({"source": str(p.relative_to(ROOT)), "line": number})
delegated_rows = []
for n, (path, callers) in enumerate(sorted(delegates.items()), 1):
    p = ROOT / path
    text = p.read_text()
    cases = []
    for m in re.finditer(r'^\s*(?:test|it)(?:\.\w+)?\s*\(\s*(["\'`])([\s\S]*?)\1\s*,', text, re.M):
        token_offset = re.search(r"\b(?:test|it)\b", m.group(0)).start()
        line = text[:m.start() + token_offset].count("\n") + 1
        cases.append({"line": line, "title_or_template": m.group(2), "dynamic": m.group(1) == "`"})
    parents = [r["id"] for r in ROWS if path in r["related_sources"]]
    for c in callers:
        candidates = [r for r in ROWS if r["source"] == c["source"] and r["kind"] == "go_test" and r["line"] <= c["line"]]
        # A caller within a helper is associated by source, not asserted to be a
        # lexical child. Runtime call sites here have an unambiguous test owner.
        if candidates:
            parents.append(max(candidates, key=lambda r: r["line"])["id"])
    delegated_rows.append({"id": f"DELEGATE-{n:03}", "source": path,
        "sha256": hashlib.sha256(p.read_bytes()).hexdigest(), "callers": callers,
        "parent_rows": sorted(set(parents)), "declarations": cases,
        "replacement": "Can cases preserve the linked parent obligations and every retained declaration/variant below. Native probes may expose hostile values, resource events and raw observations; Can owns the expectations and scenario sequence.",
        "delete_when": "All retained subcases and dynamic dimensions have Can-owned evidence; independent consumers resolved. Running bun test from Can does not count. Exact carrier layouts and old pass-count spellings require current-contract relevance review.",
        "status": "indexed for parity; not migrated or executed"})
write_json("delegated-oracles.json", {"method": "Concrete invocation paths plus lexical test declarations; titles are a navigation/checklist index, not proof of semantic equivalence or dynamic runtime counts.", "suites": delegated_rows})

for part, rows in GROUPS.items():
    out = [f"# Native Can migration ledger: {part}", "", "Status: **proposed replacements; implementation and parity evidence pending**.", "", "[Overview](../../native-can-tests-migration-ledger-2026-09-30.md) · [Machine-readable rows](" + part + ".json)", "", "Every deletion condition is conjunctive with the overview's common gate. It covers **all** protected facets, variants and delegated oracles, even where a row's short replacement sentence mentions only its first facet. Row IDs name coverage obligations, not one-to-one implementation files.", ""]
    for r in rows:
        out += [f"## {r['id']}", "", f"**{r['symbol']}** — {link(r['source'], r['line'])}", "", f"Kind: `{r['kind']}`. Proposed disposition: `{r['disposition']}`. Replacement obligation: `CAN-{r['id']}`. Evidence: pending.", "", "**Protects**", ""]
        out += ["- " + s for s in r["protects"]]
        for title, field in [("Current observations", "observes"), ("Variants", "variants"), ("Environment/selection gates", "gates"), ("Capabilities", "capabilities")]:
            out += ["", f"**{title}:** {sentence_list(r[field])}"]
        out += ["", "**Proposed Can replacement:** " + r["replacement"], "", "**Old harness may be deleted when:** " + r["delete_when"]]
        if r["related_sources"]:
            out += ["", "**Related source:** " + "; ".join(link(s) for s in r["related_sources"])]
        out += ["", "**Notes:** " + r["notes"], ""]
    (HERE / (part + ".md")).write_text("\n".join(out))

out = ["# Complete file disposition map", "", "[Overview](../../native-can-tests-migration-ledger-2026-09-30.md) · [JSON with hashes and conditions](file-map.json)", "", "All 224 tracked `/tests` files. Direct rows own declarations/content; related rows cite a dependency; family context is a navigation aid and does **not** assert a direct executable call. The JSON records those separately. All dispositions remain proposed.", "", "| File | Role | Ledger links |", "| --- | --- | --- |"]
for f in file_rows:
    refs = f["direct_rows"] or f["related_rows"] or f["family_context_rows"]
    label = "direct: " if f["direct_rows"] else "related: " if f["related_rows"] else "family context: "
    out.append(f"| {link(f['path'])} | {f['role']} | {label}{', '.join(rowlink(r) for r in refs)} |")
(HERE / "file-map.md").write_text("\n".join(out) + "\n")

out = ["# Delegated runtime oracle checklist", "", "[Overview](../../native-can-tests-migration-ledger-2026-09-30.md) · [JSON with caller lines and hashes](delegated-oracles.json)", "", "These runtime suites are actually invoked by `/tests` harnesses. Their parent ledger rows state the behavioral mapping. This subordinate declaration index prevents silently reducing a whole suite to its wrapper's headline. Templates may expand into several cases; read their source loops. Titles alone do not prove parity. Every retained observation and fault variant must be reviewed when implementing its Can case. No oracle has been migrated or run here.", ""]
for d in delegated_rows:
    out += [f"## {d['id']}", "", link(d["source"]), "", "Parents: " + ", ".join(rowlink(x) for x in d["parent_rows"]), "", "Replacement: " + d["replacement"], "", "Deletion gate: " + d["delete_when"], ""]
    out += ["- " + link(d["source"], c["line"]) + ": " + c["title_or_template"] for c in d["declarations"]]
    out.append("")
(HERE / "delegated-oracles.md").write_text("\n".join(out))

stats = {"rows": len(ROWS), "parts": {p: len(r) for p, r in GROUPS.items()},
    "kinds": dict(collections.Counter(r["kind"] for r in ROWS)),
    "files": len(file_rows), "delegated_suites": len(delegated_rows),
    "delegated_declarations": sum(len(d["declarations"]) for d in delegated_rows),
    "capability_rows": {k: sum(k in r["capabilities"] for r in ROWS) for k in SCHEMA["capabilities"]}}
write_json("validation.json", {"scope": "Documentation/source audit only; no suite execution, build, installation or benchmark.",
    "baseline_head": INDEX["head"], "status": "passed" if not errors else "failed", "errors": errors,
    "checks": ["Unique IDs", "Row fields and capability identifiers", "Existing source and related paths", "All 146 Go test declaration sites exactly once", "All 58 Bun declaration sites exactly once", "Both TestMain entrypoints", "All 224 captured test file hashes unchanged", "Current tracked test paths match inventory", "Dedicated rows for authored host executable files", "Every tracked test file has a role and ledger route"],
    "limits": "Source/declaration completeness is not runtime parity. Browser call-site and dynamic variant semantic completeness requires review; no assertion is made that current suites pass.", "counts": stats})
print(json.dumps({"errors": errors, "counts": stats}, indent=2))
if errors:
    raise SystemExit(1)
