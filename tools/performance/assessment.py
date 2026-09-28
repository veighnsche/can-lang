"""Validate advisory Score judgments bound to verified measurement evidence.

This module validates Score responses and maps advisory positions to display letters.
It neither contacts Jev nor assigns grades from timing thresholds.
The caller verifies raw evidence before attaching an assessment.
"""

import copy
import hashlib
import json
import statistics

from ranking import SUITE_UNITS, timing_resolution_issue
from schema import number

RUBRIC_VERSION = "can-jev-score-v1"
# Project presentation positions; these are not TypeSafe performance standards.
GRADE_POSITIONS = (("F", 0), ("D-", 2 / 3), ("D", 1), ("D+", 4 / 3),
                   ("C-", 5 / 3), ("C", 2), ("C+", 7 / 3), ("B-", 8 / 3),
                   ("B", 3), ("B+", 10 / 3), ("A-", 11 / 3), ("A", 4), ("A+", 5))
SCORE_INDICES = {str(index) for index in range(6)}


def evidence_fingerprint(data):
    """Hash source identity and case evidence, independent of presentation."""
    try:
        bound = {"source": data["manifest"]["source"], "cases": data["cases"]}
        encoded = json.dumps(bound, sort_keys=True, separators=(",", ":"), allow_nan=False).encode()
    except (KeyError, TypeError, ValueError) as exc:
        raise ValueError("Cannot fingerprint incomplete or nonfinite report evidence") from exc
    return hashlib.sha256(encoded).hexdigest()


def _measured_slices(data):
    if (not isinstance(data, dict) or data.get("kind") != "can.performance-report"
            or data.get("schema_version") != 1 or isinstance(data.get("schema_version"), bool)
            or not isinstance(data.get("manifest"), dict)):
        raise ValueError("Assessment requires a performance report")
    manifest = data["manifest"]
    requested = manifest.get("requested_suites")
    if (manifest.get("status") != "complete" or manifest.get("quality") != "measurement"
            or not isinstance(requested, list) or not requested
            or any(not isinstance(suite, str) or suite not in SUITE_UNITS for suite in requested)
            or len(set(requested)) != len(requested)
            or requested != manifest.get("completed_suites")
            or not isinstance(manifest.get("source"), dict)):
        raise ValueError("Assessment requires a complete measurement run with unique completed suites")
    cases = data.get("cases")
    if not isinstance(cases, dict) or not cases:
        raise ValueError("Assessment requires measured cases")
    slices = {suite: [] for suite in requested}
    for key, row in cases.items():
        if not isinstance(key, str) or "/" not in key or not isinstance(row, dict):
            raise ValueError("Invalid assessment case inventory")
        suite, name = key.split("/", 1)
        if suite not in slices or not name:
            raise ValueError(f"Case does not belong to a selected suite: {key}")
        stats = row.get("distribution")
        if row.get("unit") != SUITE_UNITS[suite] or not isinstance(stats, dict):
            raise ValueError(f"Invalid measurement unit or distribution: {key}")
        for field in ("median", "min", "max", "median_absolute_deviation"):
            if not _bounded_number(stats.get(field), 0, float("inf")):
                raise ValueError(f"Invalid finite measurement distribution: {key}/{field}")
        count = stats.get("n")
        if (not isinstance(count, int) or isinstance(count, bool) or count < 1
                or not stats["min"] <= stats["median"] <= stats["max"]):
            raise ValueError(f"Invalid measurement distribution: {key}")
        slices[suite].append(row)
    if any(not rows for rows in slices.values()):
        raise ValueError("Every selected suite must have measured cases for assessment")
    return slices


def _bounded_number(value, minimum, maximum):
    try:
        return number(value) and minimum <= value <= maximum
    except OverflowError:
        return False


def letter_grade(score):
    """Use nearest named display position, resolving midpoint ties downward."""
    if not _bounded_number(score, 0, 5):
        raise ValueError("Advisory Score must be finite and between 0 and 5")
    for (grade, lower), (_, upper) in zip(GRADE_POSITIONS, GRADE_POSITIONS[1:]):
        midpoint = (lower + upper) / 2
        if score <= midpoint:
            return grade
    return GRADE_POSITIONS[-1][0]


def validate_score_answer(answer):
    """Validate all six probability levels, retaining the complete response."""
    if not isinstance(answer, dict) or answer.get("type") != "score":
        raise ValueError("Expected a TypeSafe Score answer")
    if not _bounded_number(answer.get("score"), 0, 5):
        raise ValueError("TypeSafe Score must be finite and between 0 and 5")
    if not _bounded_number(answer.get("confidence"), 0, 1):
        raise ValueError("TypeSafe Score confidence must be finite and between 0 and 1")
    legend, probabilities = answer.get("legend"), answer.get("probabilities")
    if (not isinstance(legend, dict) or set(legend) != SCORE_INDICES
            or any(not isinstance(text, str) or not text.strip() for text in legend.values())):
        raise ValueError("TypeSafe Score legend must describe exactly six indexed levels")
    if (not isinstance(probabilities, dict) or set(probabilities) != SCORE_INDICES
            or any(not _bounded_number(value, 0, 1) for value in probabilities.values())):
        raise ValueError("TypeSafe Score requires six finite probabilities between 0 and 1")
    if abs(sum(probabilities.values()) - 1) > 0.02 + 1e-12:
        raise ValueError("TypeSafe Score probabilities must sum to one within rounding tolerance")
    weighted = sum(index * probabilities[str(index)] for index in range(6))
    if abs(weighted - answer["score"]) > 0.06 + 1e-12:
        raise ValueError("TypeSafe Score differs from its probability-weighted levels")
    return answer


def measured_slices(data):
    """Expose validated case inventories to the programmatic assessment client."""
    return _measured_slices(data)


def attach_assessment(data, payload):
    """Return a copy with validated responses and calculated advisory grades."""
    measured = _measured_slices(data)
    if (not isinstance(payload, dict) or payload.get("kind") != "can.performance-assessment"
            or payload.get("schema_version") != 1 or isinstance(payload.get("schema_version"), bool)
            or payload.get("rubric_version") != RUBRIC_VERSION):
        raise ValueError("Unsupported performance assessment schema or rubric")
    if payload.get("evidence_fingerprint") != evidence_fingerprint(data):
        raise ValueError("Assessment evidence fingerprint does not match this report")
    rubric = payload.get("rubric")
    if (not isinstance(rubric, list) or len(rubric) != 6
            or any(not isinstance(text, str) or not text.strip() for text in rubric)):
        raise ValueError("Assessment rubric requires six ordered display descriptors")
    slices = payload.get("slices")
    if not isinstance(slices, dict) or set(slices) != set(measured):
        raise ValueError("Assessment slices must exactly match selected suites")
    assessment = copy.deepcopy(payload)
    for suite, row in slices.items():
        if not isinstance(row, dict) or not isinstance(row.get("basis"), str) or not row["basis"].strip():
            raise ValueError(f"Assessment requires an authored evidence basis: {suite}")
        case_ids = {key for key in data["cases"] if key.startswith(suite + "/")}
        unresolved = {key for key in case_ids if timing_resolution_issue(data["cases"][key])}
        excluded = row.get("excluded_cases")
        if (not isinstance(excluded, list) or any(not isinstance(key, str) or key not in case_ids for key in excluded)
                or len(set(excluded)) != len(excluded) or not unresolved.issubset(excluded)):
            raise ValueError(f"Assessment exclusions must identify this slice's unresolved cases: {suite}")
        answers = row.get("answers")
        if not isinstance(answers, list) or len(answers) not in (0, 3):
            raise ValueError(f"Assessment requires three complete Score answers or an ungraded reason: {suite}")
        if len(unresolved) == len(case_ids) and answers:
            raise ValueError(f"All timings are below measurement resolution; slice must be ungraded: {suite}")
        if set(excluded) == case_ids and answers:
            raise ValueError(f"All measured cases are excluded; slice must be ungraded: {suite}")
        target = assessment["slices"][suite]
        if not answers:
            if not isinstance(row.get("ungraded_reason"), str) or not row["ungraded_reason"].strip():
                raise ValueError(f"Ungraded slice requires a reason: {suite}")
            target.update(grade="U", score_median=None, score_min=None, score_max=None,
                          grade_min="U", grade_max="U", confidence_min=None, disagreement=False)
            continue
        for answer in answers:
            validate_score_answer(answer)
        scores = [answer["score"] for answer in answers]
        median, minimum, maximum = statistics.median(scores), min(scores), max(scores)
        target.update(grade=letter_grade(median), score_median=median, score_min=minimum, score_max=maximum,
                      grade_min=letter_grade(minimum), grade_max=letter_grade(maximum),
                      confidence_min=min(answer["confidence"] for answer in answers),
                      disagreement=minimum != maximum)
    result = copy.deepcopy(data)
    result["assessment"] = assessment
    return result
