"""Pure, contract-bound performance rankings; no inferred targets or combined score."""

import copy
import math

UNIT_DIRECTIONS = {
    "ns/op": "lower", "ms/launch": "lower", "ms/interaction": "lower",
    "ms/trial": "lower", "ms/op": "lower", "ms/journey": "lower", "ops/s": "higher",
}
SUITE_UNITS = {
    "compiler": "ns/op", "assertions": "ns/op", "artifacts": "ns/op",
    "generated": "ns/op", "runtime": "ns/op", "codecs": "ns/op",
    "startup": "ms/launch", "browser": "ms/interaction", "server": "ms/trial",
    "io": "ms/op", "editor": "ns/op", "journeys": "ms/journey",
}
CONTEXT_FIELDS = ("environment", "profile", "trials", "iterations", "warmups", "size", "isolation")


def comparison_context(manifest):
    source = manifest.get("source")
    if (any(key not in manifest for key in CONTEXT_FIELDS) or not isinstance(source, dict)
            or any(key not in source for key in ("dependencies", "harness_overlays"))):
        raise ValueError("Incomplete comparison context")
    return copy.deepcopy({**{key: manifest[key] for key in CONTEXT_FIELDS},
                          "source": {key: source[key] for key in ("dependencies", "harness_overlays")}})


def contract(row):
    return copy.deepcopy({"unit": row["unit"], "direction": UNIT_DIRECTIONS.get(row["unit"]),
                          **{key: row[key] for key in ("parameters", "timing_scope", "iterations_per_sample")}})


def target_template(manifest, summary):
    if not _eligible(manifest):
        raise ValueError("Target templates require a complete measurement run")
    return {"schema_version": 1, "kind": "can.performance-targets", "name": "reviewed case targets",
            "context": comparison_context(manifest),
            "cases": {key: {**contract(row), "target": None, "rationale": ""}
                      for key, row in sorted(summary.items())}}


def _eligible(manifest):
    requested = manifest.get("requested_suites")
    valid = (isinstance(requested, list) and bool(requested)
             and all(isinstance(suite, str) and suite for suite in requested)
             and len(set(requested)) == len(requested))
    if not valid:
        return False
    try:
        comparison_context(manifest)
    except ValueError:
        return False
    return (manifest.get("status") == "complete" and manifest.get("quality") == "measurement"
            and requested == manifest.get("completed_suites"))


def _board(summary, status, reason):
    return {"status": status, "reason": reason, "eligible_cases": 0, "total_cases": len(summary),
            "exclusions": {key: reason for key in sorted(summary)}, "rows": [], "per_case": {}}


def _number(value):
    return isinstance(value, (int, float)) and not isinstance(value, bool) and math.isfinite(value)


def validate_targets(targets):
    if (not isinstance(targets, dict) or targets.get("schema_version") != 1
            or targets.get("kind") != "can.performance-targets"
            or not isinstance(targets.get("name"), str) or not targets["name"].strip()
            or not isinstance(targets.get("context"), dict) or not isinstance(targets.get("cases"), dict)):
        raise ValueError("Invalid performance target document")
    context = targets["context"]
    comparison_context(context)
    for key, row in targets["cases"].items():
        if not isinstance(key, str) or not isinstance(row, dict) or "target" not in row:
            raise ValueError("Invalid target entry")
        value = row["target"]
        if value is not None and (not _number(value) or value <= 0):
            raise ValueError(f"Target must be finite and strictly positive: {key}")
        if value is not None and (not isinstance(row.get("rationale"), str) or not row["rationale"].strip()):
            raise ValueError(f"Populated target requires a rationale: {key}")


def _row(key, observed, reference, label):
    direction = UNIT_DIRECTIONS.get(observed["unit"])
    if direction is None:
        return None, "unknown metric unit"
    if observed.get("ranking_issues"):
        return None, "candidate evidence: " + "; ".join(observed["ranking_issues"])
    stats = observed["distribution"]
    value = stats["median"]
    if not _number(value) or value < 0:
        return None, "unusable observed median"
    if not _number(reference) or reference <= 0:
        return None, "unusable reference median"
    if direction == "higher" and value == 0:
        return None, "zero observed benefit cannot be inverted"
    ratio = value / reference if direction == "lower" else reference / value
    change_percent = (ratio - 1) * 100
    if not math.isfinite(ratio) or not math.isfinite(change_percent):
        return None, "nonfinite normalized ratio or percentage"
    return {"case": key, "unit": observed["unit"], "direction": direction, "observed": value,
            "reference": reference, "ratio": ratio, "change_percent": change_percent,
            "trial_count": stats["n"], "mad": stats["median_absolute_deviation"],
            "reference_label": label}, None


def _finish(board):
    board["eligible_cases"] = len(board["per_case"])
    board["rows"] = sorted((row for row in board["per_case"].values() if row["ratio"] > 1),
                           key=lambda row: (-row["ratio"], row["case"]))
    board["status"] = "partial" if board["exclusions"] and board["per_case"] else "ranked" if board["per_case"] else "unavailable"
    board["reason"] = "Some cases lack compatible usable references" if board["status"] == "partial" else "No eligible case references" if board["status"] == "unavailable" else "All cases have compatible references; only worsenings are listed"
    return board


def build_rankings(manifest, summary, targets=None, baseline=None):
    if targets is not None:
        validate_targets(targets)
    if not _eligible(manifest):
        return {kind: _board(summary, "ineligible", "Candidate is not a complete measurement run")
                for kind in ("targets", "baseline")}
    boards = {kind: _board(summary, "unavailable", "No " + kind + " reference supplied")
              for kind in ("targets", "baseline")}
    if targets is not None:
        if targets["context"] != comparison_context(manifest):
            boards["targets"] = _board(summary, "unavailable", "Target comparison context differs")
        else:
            board = _board(summary, "unavailable", "")
            board["exclusions"] = {key: "stale target case absent from candidate" for key in targets["cases"] if key not in summary}
            for key, observed in sorted(summary.items()):
                ref = targets["cases"].get(key)
                reason = "missing target" if ref is None else "unset target" if ref["target"] is None else None
                if reason is None and any(ref.get(field) != value for field, value in contract(observed).items()):
                    reason = "target workload contract differs"
                row = None
                if reason is None:
                    row, reason = _row(key, observed, ref["target"], targets["name"])
                if reason:
                    board["exclusions"][key] = reason
                else:
                    board["per_case"][key] = row
            boards["targets"] = _finish(board)
    if baseline is not None:
        before_manifest, before_summary = baseline
        reason = None
        if not _eligible(before_manifest):
            reason = "Baseline is not a complete measurement run"
        elif before_manifest["requested_suites"] != manifest["requested_suites"]:
            reason = "Baseline requested suite order differs"
        elif comparison_context(before_manifest) != comparison_context(manifest):
            reason = "Baseline comparison context differs"
        elif set(before_summary) != set(summary):
            reason = "Baseline case inventory differs"
        elif any(contract(before_summary[key]) != contract(row) for key, row in summary.items()):
            reason = "Baseline workload contracts differ"
        if reason:
            boards["baseline"] = _board(summary, "unavailable", reason)
        else:
            board = _board(summary, "unavailable", "")
            board["exclusions"] = {}
            for key, observed in sorted(summary.items()):
                reference = before_summary[key]
                issues = reference.get("ranking_issues")
                row, reason = (None, "baseline evidence: " + "; ".join(issues)) if issues else _row(key, observed, reference["distribution"]["median"], "matched prior run")
                if reason:
                    board["exclusions"][key] = reason
                else:
                    board["per_case"][key] = row
            boards["baseline"] = _finish(board)
    return boards
