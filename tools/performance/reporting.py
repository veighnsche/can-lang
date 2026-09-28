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


def markdown_report(manifest, summary, rankings=None, suites=None):
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
              f"{len(suites)} available. All suites execute sequentially.", "",
              "Case counts describe measured scenarios and implementation variants, not percentages of language or feature coverage.", ""]
    if quality != "measurement" or status != "complete":
        lines += ["This run is ineligible for performance rankings or use as an accepted baseline (non-measurement or incomplete evidence).", ""]
    if quality == "exploratory":
        lines += ["Exploratory evidence uses weakened host-isolation settings and is INELIGIBLE for rankings and baselines.", ""]
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
    unresolved = [key for key, row in summary.items() if timing_resolution_issue(row)]
    if unresolved:
        count = f"{len(unresolved)} case is" if len(unresolved) == 1 else f"{len(unresolved)} cases are"
        lines += [f"Timing limit: **{count} below measurement resolution** because at least one trial median is zero. "
                  "These readings do not establish instantaneous execution. Affected cases are excluded from target and baseline ratios, "
                  "including when used as baseline references.", "",
                  "Affected cases: " + "; ".join(escape(key) for key in sorted(unresolved)) + ".", ""]
    preparation = sum(s.get("elapsed_seconds", 0) for s in steps if s.get("phase") == "prepare")
    drivers = sum(s.get("elapsed_seconds", 0) for s in steps if s.get("phase") == "trial")
    lines += [f"Orchestration wall time: total run **{number(manifest.get('wall_seconds'))} s**; "
              f"preparation drivers **{number(preparation)} s**; suite drivers **{number(drivers)} s**.", "",
              "These durations include orchestration work and are not summed workload latency or an overall performance score.", ""]
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
            lines += [label + ".", "", "<details>", "<summary>Comparison provenance and SHA256</summary>", "",
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
        rows = board.get("rows", []) if board.get("status") in ("ranked", "partial") and status == "complete" and quality == "measurement" else []
        if rows:
            lines += ["", "| Position | Case | Current | Reference | Ratio | Ratio excess (%) | Independent n | MAD | Reference label |",
                      "|---:|---|---:|---:|---:|---:|---:|---:|---|"]
            for index, row in enumerate(rows[:10], 1):
                unit = escape(row["unit"])
                lines.append(f"| {index} | {escape(row['case'])} | {number(row['observed'])} {unit} | {number(row['reference'])} {unit} | {number(row['ratio'])}× | {number(row['change_percent'])} | {row['trial_count']} | {number(row['mad'])} {unit} | {escape(row['reference_label'])} |")
        elif board.get("status") in ("ranked", "partial") and board.get("eligible_cases", 0) > 0 and status == "complete" and quality == "measurement":
            lines += ["", empty + " among eligible cases."]
        else:
            lines += ["", "No ranked positions available."]
        if board.get("exclusions"):
            lines += ["", "<details>", "<summary>Excluded cases</summary>", ""]
            lines += [f"- {escape(case)}: {escape(reason)}" for case, reason in board["exclusions"].items()]
            lines += ["", "</details>"]
    lines += ["", "Ratios are current/reference for lower-is-better measurements and reference/current for higher-is-better measurements. "
              "Only ratios above 1 appear. Ratios are descriptive, not evidence of statistical significance or business impact; "
              "overlapping cases are not independent costs.", "",
              "Independent process trials summarize each trial's batch median. MAD is the median absolute deviation of those trial medians; min–max is their observed range."]
    for suite in suites:
        cases = [(key, row) for key, row in summary.items() if key.startswith(suite + "/")]
        if not cases:
            continue
        lines += ["", "<details>", f"<summary>{escape(suite)} case details ({len(cases)})</summary>", "",
                  "| Case | Median | Unit | Min–max | MAD | Independent n | Parameters | Timing scope |",
                  "|---|---:|---|---|---:|---:|---|---|"]
        for key, row in cases:
            stats = row["distribution"]
            median = number(stats['median'])
            if timing_resolution_issue(row):
                median += " (below measurement resolution)"
            lines.append(f"| {escape(key)} | {median} | {escape(row['unit'])} | {number(stats.get('min'))}–{number(stats.get('max'))} | {number(stats['median_absolute_deviation'])} | {stats['n']} | {escape(json.dumps(row['parameters'], sort_keys=True))} | {escape(row['timing_scope'])} |")
        issues = [(key, row.get("ranking_issues", [])) for key, row in cases if row.get("ranking_issues")]
        for key, reasons in issues:
            if isinstance(reasons, str):
                reasons = [reasons]
            lines += ["", f"Ranking limitations for {escape(key)}: " + "; ".join(escape(reason) for reason in reasons) + "."]
        lines += ["", "</details>"]
    return "\n".join(lines) + "\n"
