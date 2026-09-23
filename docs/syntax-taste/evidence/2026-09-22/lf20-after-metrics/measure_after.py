"""LF20/AE49 after-implementation measurements.

Reruns the frozen eight-fetch comparison against the implemented after
source, reruns the four application metrics with the identical method as
../full-language-review/measure.py, and adds full-project production versus
test/fixture cost splits. Writes after-metrics.json; never overwrites the
frozen before artifacts.
"""
from pathlib import Path
import hashlib
import json
import re

HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[4]
BEFORE = (HERE.parent / "acceptance-contracts" / "eight-fetch-before.can.txt").read_text()
AFTER_PATH = ROOT / "compiler/testdata/current/fetch/main.can"
AFTER = AFTER_PATH.read_text()
RAW7 = ["http::invalid_request", "http::credentials_missing", "http::transport_failed",
        "http::timeout", "http::body_limit", "http::status_error", "codec::invalid_data"]
MARKER = "fn str referenced"

# Frozen before check (same rule as acceptance-contracts/validate.py).
meta = json.loads((HERE.parent / "acceptance-contracts" / "baseline.json").read_text())
assert hashlib.sha256(BEFORE.encode()).hexdigest() == meta["sha256"]
before_region = BEFORE.split("\n" + MARKER + "\n", 1)[0].splitlines()
before_bounds = [l for l in before_region if l.strip().startswith("emits [") and any(e in l for e in RAW7)]
before = {
    "fetch_declarations": len(re.findall(r"^fetch .*? (\w+) from ", "\n".join(before_region), re.M)),
    "infrastructure_bound_declarations": len(before_bounds),
    "infrastructure_bound_entries": sum(l.count(e) for l in before_bounds for e in RAW7),
    "infrastructure_forwarding_arms": sum(l.strip() in RAW7 for l in before_region),
}
assert before == {"fetch_declarations": 8, "infrastructure_bound_declarations": 9,
                  "infrastructure_bound_entries": 63, "infrastructure_forwarding_arms": 56}, before

# After region: same end marker, normalized single-error bounds.
after_region = AFTER.split("\n" + MARKER + "\n", 1)[0].splitlines()
after_bounds = [l for l in after_region if l.strip().startswith("emits [") and "http::request_failed" in l]
after = {
    "scope": f"compiler/testdata/current/fetch/main.can lines 1-{len(after_region)}; ends before {MARKER!r}",
    "source_lines_in_region": len(after_region),
    "fetch_declarations": len(re.findall(r"^fetch .*? (\w+) from ", "\n".join(after_region), re.M)),
    "main_fetch_calls": sum(len(re.findall(r"\bmatch call " + re.escape(f) + r"\(", "\n".join(after_region)))
                            for f in re.findall(r"^fetch .*? (\w+) from ", "\n".join(after_region), re.M)),
    "infrastructure_bound_declarations": len(after_bounds),
    "infrastructure_bound_entries": sum(l.count("http::request_failed") for l in after_bounds),
    "infrastructure_forwarding_arms": sum(l.strip() == "http::request_failed" for l in after_region),
    "checks_failed_arms_in_region": sum(l.strip() == "checks::failed" for l in after_region),
}
assert after["fetch_declarations"] == 8 and after["main_fetch_calls"] == 8, after
assert after["infrastructure_bound_declarations"] == 9, after
assert after["infrastructure_bound_entries"] == 9, after
assert after["infrastructure_forwarding_arms"] == 8, after

# Detail distinctions recoverable from the normalized error (whole project).
details = sorted({e for e in RAW7 if re.search(r"request_failed\(" + re.escape(e) + r"\b", AFTER)})
assert len(details) == 7, details

# Production versus test/fixture cost split. A source line counts as
# test/fixture when it opens an asserts/when block, carries an assertion or
# substitution row (name: ... => ...), or attaches a raw fixture. Raw JSON
# fixture bytes are counted separately. Same classifier both versions.
ROW = re.compile(r"^\s+[\w\-\"]+(\([^)]*\))?\s*:.*=>")
HEADER = re.compile(r"^\s+(asserts|when)\b")


def split_cost(text):
    prod, test = 0, 0
    for line in text.splitlines():
        s = line.strip()
        if not s:
            continue
        if HEADER.match(line) or ROW.match(line) or "using raw" in s:
            test += 1
        else:
            prod += 1
    return {"production_lines": prod, "assert_fixture_lines": test}


before_cost = split_cost(BEFORE)
after_cost = split_cost(AFTER)
fixture_files = sorted((AFTER_PATH.parent / "fixtures").glob("*.json"))
fixture_bytes = sum(p.stat().st_size for p in fixture_files)

# Four applications, identical method to full-language-review/measure.py.
apps = []
for app in sorted((ROOT / "examples").iterdir()):
    files = sorted(app.glob("src/**/*.can"))
    if not files:
        continue
    lines = [line for p in files for line in p.read_text().splitlines()]
    emits = [line.strip() for line in lines if line.strip().startswith("emits [")]
    cost = {"production_lines": 0, "assert_fixture_lines": 0}
    for p in files:
        part = split_cost(p.read_text())
        cost["production_lines"] += part["production_lines"]
        cost["assert_fixture_lines"] += part["assert_fixture_lines"]
    apps.append({"application": app.name, "can_files": len(files), "source_lines": len(lines),
                 "maximum_line_characters": max(map(len, lines)),
                 "emits_declarations": len(emits),
                 "emits_characters_excluding_indent": sum(map(len, emits)),
                 "seven_infrastructure_error_mentions": sum(line.count(e) for line in lines for e in RAW7),
                 "normalized_request_failed_mentions": sum(l.count("http::request_failed") for l in lines),
                 **cost})

result = {
    "method": ("Frozen before hash + counts re-verified; after region uses the same end marker "
               "with normalized http::request_failed counts; app method identical to "
               "full-language-review/measure.py plus a production/assert-fixture line split "
               "(asserts/when headers, name: ... => ... rows, using-raw lines). Raw fixture "
               "JSON bytes counted separately. Not model tokens; no TypeScript comparison."),
    "before_metrics": before,
    "after_metrics": after,
    "detail_distinctions_recoverable": {"count": len(details), "details": details},
    "before_cost": {**before_cost, "raw_fixture_files": 0, "raw_fixture_bytes": 0},
    "after_cost": {**after_cost, "raw_fixture_files": len(fixture_files),
                   "raw_fixture_bytes": fixture_bytes,
                   "after_source": str(AFTER_PATH.relative_to(ROOT))},
    "applications": apps,
}
(HERE / "after-metrics.json").write_text(json.dumps(result, indent=2) + "\n")
print(json.dumps(result, indent=2))
