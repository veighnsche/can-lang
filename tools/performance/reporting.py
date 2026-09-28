"""Readable presentation of measured evidence and supplied comparison boards."""

import html
import json

from ranking import timing_resolution_issue

DEFAULT_SUITES = {
    "compiler": ("compiler", "source loading, checking, emission and scaling"),
    "assertions": ("compiler", "precompiled assertions and worker supervision"),
    "artifacts": ("compiler", "output processing, validation and footprint"),
    "generated": ("runtime", "actual emitted Can versus native equivalents"),
    "runtime": ("runtime", "collections, calls and ownership helpers"),
    "codecs": ("runtime", "validated data conversion and rejection"),
    "startup": ("runtime", "fresh process and module loading"),
    "browser": ("apps", "native browser controls and observable completion"),
    "server": ("apps", "bounded-client and arrival-rate load"),
    "io": ("apps", "controlled host I/O and adapter boundaries"),
    "editor": ("compiler", "language-server feedback after edits"),
    "journeys": ("apps", "complete emitted-Can application workflow"),
}


def escape(value):
    text = html.escape(str(value), quote=True).replace("\n", "<br>").replace("\r", "")
    for character in "\\`*_{}[]|":
        text = text.replace(character, "\\" + character)
    return text


def number(value):
    return "unavailable" if value is None else f"{value:.6g}"


def display_scale(unit, value):
    """Choose presentation units without altering the saved measurement contract."""
    if unit == "ns/op" and value is not None:
        if abs(value) >= 1_000_000:
            return 1_000_000, "ms/op"
        if abs(value) >= 1_000:
            return 1_000, "µs/op"
    return 1, unit


def scaled_number(value, scale):
    return number(None if value is None else value / scale)


def exclusion_details(board, measured_cases):
    """Summarize shared exclusions once while retaining partial case coverage."""
    exclusions = board.get("exclusions") or {}
    if not exclusions:
        return []
    grouped = {}
    for case, reason in exclusions.items():
        grouped.setdefault(reason, []).append(case)
    lines = []
    for reason, cases in grouped.items():
        if set(cases) == set(measured_cases) and len(grouped) == 1:
            if reason == board.get("reason"):
                lines.append(f"All {len(cases)} measured cases are excluded for the comparison reason above.")
            else:
                lines.append(f"- {escape(reason)} — all {len(cases)} measured cases.")
        elif len(cases) == 1:
            lines.append(f"- {escape(cases[0])}: {escape(reason)}")
        else:
            lines.append(f"- {escape(reason)} — {len(cases)} cases: " + "; ".join(escape(case) for case in cases) + ".")
    return lines


def markdown_report(manifest, summary, rankings=None, suites=None, assessment=None):
    suites = DEFAULT_SUITES if suites is None else suites
    requested = set(manifest.get("requested_suites", []))
    completed = set(manifest.get("completed_suites", []))
    steps = manifest.get("steps", [])
    status = manifest.get("status", "unknown")
    quality = manifest.get("quality", "unknown")
    full = requested == set(suites)
    lines = ["# Can performance results", "",
             f"Status: **{escape(status)}**. Evidence quality: **{escape(quality)}**.", ""]
    if manifest.get("reason"):
        lines += ["Failure or limitation: **" + escape(manifest["reason"]) + "**", ""]
    lines += [f"Coverage: **{'full-system suite selection' if full else 'subset suite selection'}**; "
              f"{len(completed & requested)}/{len(requested)} requested suites completed, "
              f"{len(suites)} available.", "",
              "Case counts describe measured scenarios and implementation variants, not percentages of language or feature coverage.", ""]
    if quality != "measurement" or status != "complete":
        lines += ["This run is ineligible for performance rankings or use as an accepted baseline (non-measurement or incomplete evidence).", ""]
    if quality == "exploratory":
        lines += ["Exploratory evidence uses weakened host-isolation settings and is INELIGIBLE for rankings and baselines.", ""]
    if assessment is not None and quality == "measurement" and status == "complete":
        lines += ["## Jev’s advisory report card", "",
                  "Generated automatically through TypeSafe Score with three fresh, independently worded requests. "
                  "A/A+ = very good/exceptional; B = good; C = usable with improvements; D = substantial concern; F = severe concern; U = ungraded. "
                  "This project supplies the rubric and letter mapping; these are not TypeSafe-certified performance standards.", "",
                  "| Slice | Grade | Wording range | Assessment basis |", "|---|---|---|---|"]
        for suite, row in assessment["slices"].items():
            spread = row["grade_min"] if row["grade_min"] == row["grade_max"] else row["grade_min"] + " to " + row["grade_max"]
            note = row["basis"] + (" " + row["ungraded_reason"] if row["grade"] == "U" else "")
            if row["excluded_cases"]:
                note += " Excluded rows: " + ", ".join(row["excluded_cases"]) + "."
            lines.append(f"| {escape(suite)} | **{row['grade']}** | {spread} | {escape(note)} |")
        lines += ["", "The median of three Score assessments sets the letter. Ranges describe wording sensitivity, not confidence intervals. "
                  "The companion grading archive preserves all scores, distributions and evidence binding. Grades describe the eligible tested subset, "
                  "not exhaustive coverage or recoverable savings.", ""]
    unresolved = [key for key, row in summary.items() if timing_resolution_issue(row)]
    if unresolved:
        count = f"{len(unresolved)} case is" if len(unresolved) == 1 else f"{len(unresolved)} cases are"
        lines += [f"Timing limit: **{count} below measurement resolution** because at least one trial median is zero. "
                  "These readings do not establish instantaneous execution. Affected cases are excluded from target and baseline ratios, "
                  "including when used as baseline references.", ""]
    boards = rankings or {}
    eligible = {}
    for kind in ("targets", "baseline"):
        board = boards.get(kind) or {}
        eligible[kind] = set(board.get("per_case", {})) if board.get("status") in ("ranked", "partial") and status == "complete" and quality == "measurement" else set()
    lines += ["## Suite overview", "", "| Suite | Scope | Status | Cases | Rankable target / baseline | Driver wall (s) |",
              "|---|---|---|---:|---:|---:|"]
    for suite, (_, description) in suites.items():
        cases = [key for key in summary if key.startswith(suite + "/")]
        ran = any(step.get("suite") == suite for step in steps)
        state = "not requested" if suite not in requested else "complete" if suite in completed else "incomplete" if ran else "not run"
        wall = sum(step.get("elapsed_seconds", 0) for step in steps if step.get("phase") == "trial" and step.get("suite") == suite)
        counts = [sum(key in eligible[kind] for key in cases) for kind in ("targets", "baseline")]
        lines.append(f"| {escape(suite)} | {escape(description)} | {state} | {len(cases)} | {counts[0]} / {counts[1]} | {number(wall) if ran else '—'} |")
    lines += ["", "## Measured results", "",
              "Each row measures its named workload. In timing units, /op means one workload execution, /launch one fresh launch, "
              "/interaction one browser interaction, /trial one complete server load trial, and /journey one complete application journey. "
              "A workload execution can process many items; it is not necessarily one element or language operation. "
              "ops/s is workload throughput. Exact workload sizes and timing boundaries are in the appendix.", "",
              "ns = nanoseconds; µs = microseconds; ms = milliseconds. Display conversions preserve the raw measurements. "
              "Cases appear in suite order, without ranking different workloads by their raw duration.", "",
              "Median, min–max and MAD summarize independent process trials, each represented by its batch median. "
              "MAD is the median absolute deviation of those trial medians. Independent n is the number of process trials."]
    for suite, (_, description) in suites.items():
        cases = [(key, row) for key, row in summary.items() if key.startswith(suite + "/")]
        if not cases:
            continue
        lines += ["", f"### {escape(suite)} — {escape(description)}", "",
                  "| Case | Median | Unit | Min–max | MAD | Independent n |",
                  "|---|---:|---|---|---:|---:|"]
        for key, row in cases:
            stats = row["distribution"]
            scale, unit = display_scale(row["unit"], stats["median"])
            median = scaled_number(stats["median"], scale)
            if timing_resolution_issue(row):
                median += " (below measurement resolution)"
            lines.append(f"| {escape(key)} | {median} | {escape(unit)} | {scaled_number(stats.get('min'), scale)}–{scaled_number(stats.get('max'), scale)} | {scaled_number(stats['median_absolute_deviation'], scale)} | {stats['n']} |")
    comparison_appendix = []
    for kind, title, empty in (("targets", "Top 10 target shortfalls", "No target exceedances"),
                               ("baseline", "Top 10 relative regressions", "No relative regressions")):
        board = boards.get(kind)
        lines += ["", "## " + title, ""]
        reference = (manifest.get("ranking_references") or {}).get(kind)
        if reference:
            if kind == "targets":
                label = "Target reference: " + escape(reference.get("name", "unnamed"))
            else:
                source_reference = reference.get("source") or {}
                label = "Baseline reference: " + "; ".join(
                    f"{field}: {escape(source_reference[field])}"
                    for field in ("revision", "source_mode") if source_reference.get(field) is not None
                )
                if not source_reference.get("revision") and not source_reference.get("source_mode"):
                    label += "source provenance unavailable"
            lines += [label + ".", ""]
            comparison_appendix += ["### " + kind.capitalize() + " reference", "", "<details>",
                                   "<summary>Comparison provenance and SHA256</summary>", "",
                                   "<pre>" + html.escape(json.dumps(reference, sort_keys=True, indent=2)) + "</pre>", "", "</details>", ""]
        if status != "complete" or quality != "measurement":
            lines += ["Ineligible evidence: only complete measurement runs can be ranked. No ranked positions available."]
            continue
        if not board:
            lines += ["Unavailable: no compatible references supplied; no ranked positions."]
            continue
        lines += [f"Comparison status: **{escape(board.get('status', 'unavailable'))}**. {escape(board.get('reason', ''))}", "",
                  f"Eligible cases: {board.get('eligible_cases', 0)}/{board.get('total_cases', len(summary))}."]
        if board.get("status") == "partial":
            lines += ["", "Partial reference coverage: conclusions apply only to eligible cases."]
        rows = board.get("rows", []) if board.get("status") in ("ranked", "partial") else []
        if rows:
            lines += ["", "| Position | Case | Current | Reference | Ratio | Ratio excess (%) | Independent n | MAD | Reference label |",
                      "|---:|---|---:|---:|---:|---:|---:|---:|---|"]
            for index, row in enumerate(rows[:10], 1):
                scale, display_unit = display_scale(row["unit"], row["observed"])
                unit = escape(display_unit)
                lines.append(f"| {index} | {escape(row['case'])} | {scaled_number(row['observed'], scale)} {unit} | {scaled_number(row['reference'], scale)} {unit} | {number(row['ratio'])}× | {number(row['change_percent'])} | {row['trial_count']} | {scaled_number(row['mad'], scale)} {unit} | {escape(row['reference_label'])} |")
        elif board.get("status") in ("ranked", "partial") and board.get("eligible_cases", 0) > 0:
            lines += ["", empty + " among eligible cases."]
        else:
            lines += ["", "No ranked positions available."]
        exclusions = exclusion_details(board, summary)
        if exclusions:
            comparison_appendix += ["### " + kind.capitalize() + " exclusions", "", *exclusions, ""]
    lines += ["", "Ratios are current/reference for lower-is-better measurements and reference/current for higher-is-better measurements. "
              "Only ratios above 1 appear. Ratios are descriptive, not evidence of statistical significance or business impact; "
              "overlapping cases are not independent costs.", "",
              "## Technical appendix", "", "<details>",
              "<summary>Timing boundaries, parameters, provenance and comparison exclusions</summary>", ""]
    source = manifest.get("source") or {}
    if source:
        fields = ("revision", "source_mode", "dirty_checkout")
        lines += ["Source provenance: " + "; ".join(f"{escape(k)}: {escape(source[k])}" for k in fields if k in source) + ".", "",
                  "<details>", "<summary>Full source provenance</summary>", "",
                  "<pre>" + html.escape(json.dumps(source, sort_keys=True, indent=2)) + "</pre>", "", "</details>", ""]
    settings = [(k, manifest[k]) for k in ("profile", "trials", "iterations", "warmups", "size") if k in manifest]
    if settings:
        lines += ["Sample settings: " + "; ".join(f"{k}: {escape(v)}" for k, v in settings) + ".", ""]
    if manifest.get("isolation"):
        lines += ["Host isolation: " + escape(json.dumps(manifest["isolation"], sort_keys=True)) + ".", ""]
    preparation = sum(step.get("elapsed_seconds", 0) for step in steps if step.get("phase") == "prepare")
    drivers = sum(step.get("elapsed_seconds", 0) for step in steps if step.get("phase") == "trial")
    lines += [f"Orchestration wall time: total run **{number(manifest.get('wall_seconds'))} s**; "
              f"preparation drivers **{number(preparation)} s**; suite drivers **{number(drivers)} s**.", "",
              "All suites execute sequentially. Driver durations include orchestration work and are not summed workload latency or an overall performance score.", ""]
    if unresolved:
        lines += ["Affected cases: " + "; ".join(escape(key) for key in sorted(unresolved)) + ".", ""]
    lines += comparison_appendix
    for suite in suites:
        cases = [(key, row) for key, row in summary.items() if key.startswith(suite + "/")]
        if not cases:
            continue
        lines += ["", f"### {escape(suite)} workload contracts ({len(cases)})", "",
                  "| Case | Raw unit | Parameters | Timing scope | Iterations per sample |",
                  "|---|---|---|---|---:|"]
        for key, row in cases:
            lines.append(f"| {escape(key)} | {escape(row['unit'])} | {escape(json.dumps(row['parameters'], sort_keys=True))} | {escape(row['timing_scope'])} | {escape(row.get('iterations_per_sample', 'unavailable'))} |")
        for key, row in cases:
            reasons = row.get("ranking_issues") or []
            if isinstance(reasons, str):
                reasons = [reasons]
            if reasons:
                lines += ["", f"Ranking limitations for {escape(key)}: " + "; ".join(escape(reason) for reason in reasons) + "."]
    lines += ["", "</details>"]
    return "\n".join(lines) + "\n"
