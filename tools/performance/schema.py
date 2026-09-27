"""Validate evidence before it can enter a performance comparison."""

import math
import statistics

SCHEMA_VERSION = 1


def number(value):
    return isinstance(value, (float, int)) and not isinstance(value, bool) and math.isfinite(value)


def validate_result(result, suite, *, preparation=False):
    if not isinstance(result, dict) or result.get("schema_version") != SCHEMA_VERSION:
        raise ValueError("Unsupported driver evidence schema")
    if result.get("suite") != ("prepare" if preparation else suite):
        raise ValueError("Driver returned evidence for a different suite")
    if result.get("status") != "complete":
        raise ValueError(result.get("reason", "Driver did not complete"))
    cases = result.get("cases")
    if not isinstance(cases, list) or (not preparation and not cases):
        raise ValueError("A completed measured suite must contain cases")
    names = set()
    for case in cases:
        if not isinstance(case, dict):
            raise ValueError("Each measured case must be an object")
        name = case.get("name")
        if not isinstance(name, str) or not name or name in names:
            raise ValueError("Case names must be nonempty and unique within a suite")
        names.add(name)
        if not isinstance(case.get("unit"), str) or not case["unit"]:
            raise ValueError(f"Missing unit: {name}")
        samples = case.get("samples")
        if not isinstance(samples, list) or not samples or any(not number(x) or x < 0 for x in samples):
            raise ValueError(f"Invalid measured samples: {name}")
        warmups = case.get("warmup_samples", [])
        if not isinstance(warmups, list) or any(not number(x) or x < 0 for x in warmups):
            raise ValueError(f"Invalid warmup samples: {name}")
        count = case.get("iterations_per_sample")
        if not isinstance(count, int) or isinstance(count, bool) or count < 1:
            raise ValueError(f"Invalid operations per sample: {name}")
        if not isinstance(case.get("timing_scope"), str) or not case["timing_scope"]:
            raise ValueError(f"Missing timing boundary: {name}")
        if not isinstance(case.get("parameters"), dict):
            raise ValueError(f"Missing workload parameters: {name}")
        correctness = case.get("correctness", {})
        checks = correctness.get("checks") if isinstance(correctness, dict) else None
        if not isinstance(correctness, dict) or correctness.get("passed") is not True or not isinstance(checks, list) or not checks or any(not isinstance(x, str) or not x for x in checks):
            raise ValueError(f"No successful semantic checks: {name}")
    return result


def distribution(samples):
    median = statistics.median(samples)
    return {
        "n": len(samples),
        "median": median,
        "min": min(samples),
        "max": max(samples),
        "median_absolute_deviation": statistics.median(abs(x - median) for x in samples),
    }


def summarize_trials(trials):
    """Summarize independent process-trial medians, never invented request tails."""
    grouped = {}
    inventories = {}
    for trial in trials:
        suite = trial["suite"]
        result = validate_result(trial["result"], suite)
        names = {case["name"] for case in result["cases"]}
        if suite in inventories and inventories[suite] != names:
            raise ValueError(f"Case inventory changed between trials: {suite}")
        inventories[suite] = names
        for case in result["cases"]:
            key = f"{suite}/{case['name']}"
            signature = {k: case[k] for k in ("unit", "parameters", "timing_scope", "iterations_per_sample")}
            if key not in grouped:
                grouped[key] = {**signature, "trial_medians": [], "raw_sample_counts": []}
            row = grouped[key]
            if any(row[k] != value for k, value in signature.items()):
                raise ValueError(f"Workload contract changed between trials: {key}")
            row["trial_medians"].append(statistics.median(case["samples"]))
            row["raw_sample_counts"].append(len(case["samples"]))
    return {key: {**value, "distribution": distribution(value["trial_medians"])}
            for key, value in sorted(grouped.items())}
