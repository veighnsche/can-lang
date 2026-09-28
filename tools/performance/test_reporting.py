"""Synthetic report checks; no drivers, builds or benchmarks are executed."""

import copy
import unittest

from reporting import DEFAULT_SUITES, markdown_report


def fixtures():
    manifest = {
        "status": "complete", "quality": "measurement", "requested_suites": list(DEFAULT_SUITES),
        "completed_suites": list(DEFAULT_SUITES), "wall_seconds": 50, "trials": 3,
        "iterations": 4, "warmups": 1, "size": 10,
        "source": {"revision": "abc123", "source_mode": "tracked-working-tree", "dirty_checkout": True},
        "steps": [{"phase": "prepare", "elapsed_seconds": 2}] + [
            {"phase": "trial", "suite": suite, "elapsed_seconds": 3} for suite in DEFAULT_SUITES],
    }
    summary = {suite + "/case": {"unit": "ms", "parameters": {"size": 10},
                "timing_scope": "single call", "distribution": {"n": 3, "median": 4,
                "min": 3, "max": 5, "median_absolute_deviation": 1}} for suite in DEFAULT_SUITES}
    rows = [{"case": key, "unit": "ms", "direction": "lower", "observed": 4,
             "reference": 2, "ratio": 2, "change_percent": 100, "trial_count": 3,
             "mad": 1, "reference_label": "budget"} for key in reversed(summary)]
    board = {"status": "ranked", "reason": "compatible", "eligible_cases": 12,
             "total_cases": 12, "exclusions": {}, "rows": rows, "per_case": {row["case"]: row for row in rows}}
    return manifest, summary, {"targets": board, "baseline": board}


class ReportingTests(unittest.TestCase):
    def test_measured_results_lead_and_technical_details_are_collapsed(self):
        manifest, summary, boards = fixtures()
        report = markdown_report(manifest, summary, boards)
        measured = report.index("## Measured results")
        comparisons = report.index("## Top 10 target shortfalls")
        appendix = report.index("## Technical appendix")
        self.assertLess(measured, comparisons)
        self.assertLess(comparisons, appendix)
        main = report[measured:comparisons]
        self.assertNotIn("<details>", main)
        for suite in DEFAULT_SUITES:
            self.assertIn(f"### {suite} —", main)
            self.assertIn(f"| {suite}/case | 4 | ms |", main)
        self.assertIn("/op means one workload execution", main)
        self.assertIn("not necessarily one element or language operation", main)
        self.assertIn("without ranking different workloads", main)
        self.assertGreater(report.index("Source provenance:"), appendix)
        self.assertGreater(report.index("Sample settings:"), appendix)
        self.assertGreater(report.index("| Case | Raw unit | Parameters | Timing scope |"), appendix)
        self.assertIn("<summary>Timing boundaries, parameters, provenance and comparison exclusions</summary>", report)

    def test_readable_units_apply_consistently_without_mutating_evidence(self):
        manifest, summary, boards = fixtures()
        for key, median in (("compiler/case", 2_500_000), ("runtime/case", 12_000),
                            ("generated/case", 250)):
            summary[key]["unit"] = "ns/op"
            summary[key]["distribution"].update(median=median, min=median / 2,
                                                 max=median * 2, median_absolute_deviation=median / 10)
        boards["targets"]["rows"][0].update(unit="ns/op", observed=2_500_000,
                                            reference=1_250_000, mad=250_000)
        before = copy.deepcopy((manifest, summary, boards))
        report = markdown_report(manifest, summary, boards)
        self.assertIn("| compiler/case | 2.5 | ms/op | 1.25–5 | 0.25 | 3 |", report)
        self.assertIn("| runtime/case | 12 | µs/op | 6–24 | 1.2 | 3 |", report)
        self.assertIn("| generated/case | 250 | ns/op | 125–500 | 25 | 3 |", report)
        self.assertIn("| 1 | journeys/case | 2.5 ms/op | 1.25 ms/op | 2× | 100 | 3 | 0.25 ms/op |", report)
        self.assertEqual(before, (manifest, summary, boards))

    def test_shared_missing_references_are_summarized_once_per_board(self):
        manifest, summary, _ = fixtures()
        boards = {kind: {"status": "unavailable", "reason": reason,
                        "eligible_cases": 0, "total_cases": len(summary), "rows": [],
                        "exclusions": {case: reason for case in summary}}
                  for kind, reason in (("targets", "No targets reference supplied"),
                                       ("baseline", "No baseline reference supplied"))}
        report = markdown_report(manifest, summary, boards)
        for reason in ("No targets reference supplied", "No baseline reference supplied"):
            self.assertEqual(report.count(reason), 1)
        self.assertEqual(report.count("All 12 measured cases are excluded for the comparison reason above."), 2)
        self.assertNotIn("- compiler/case:", report)
        self.assertLess(report.index("| browser/case | 4 | ms |"), report.index("No baseline reference supplied"))

    def test_partial_exclusions_group_reasons_and_retain_affected_cases(self):
        manifest, summary, boards = fixtures()
        boards["targets"] = {"status": "partial", "reason": "Some references unavailable",
                             "eligible_cases": 9, "total_cases": 12, "rows": [],
                             "exclusions": {"runtime/case": "missing target",
                                            "generated/case": "missing target",
                                            "browser/case": "target workload contract differs"}}
        report = markdown_report(manifest, summary, boards)
        self.assertEqual(report.count("missing target"), 1)
        self.assertIn("missing target — 2 cases: runtime/case; generated/case.", report)
        self.assertIn("browser/case: target workload contract differs", report)
        self.assertGreater(report.index("missing target"), report.index("## Technical appendix"))

    def test_stale_reference_cases_are_not_described_as_measured_cases(self):
        manifest, summary, boards = fixtures()
        boards["targets"] = {"status": "partial", "reason": "Some references unavailable",
                             "eligible_cases": 12, "total_cases": 12, "rows": [],
                             "exclusions": {f"runtime/stale-{index}": "stale target case absent from candidate"
                                            for index in range(12)}}
        report = markdown_report(manifest, summary, boards)
        self.assertNotIn("all 12 measured cases", report)
        self.assertIn("stale target case absent from candidate — 12 cases:", report)

    def test_zero_trial_timing_has_visible_limit_without_mutating_legacy_summary(self):
        manifest, summary, _ = fixtures()
        row = summary['browser/case']
        row['unit'] = 'ms/interaction'
        row['distribution']['min'] = 0
        report = markdown_report(manifest, summary)
        self.assertIn('Timing limit: **1 case is below measurement resolution**', report)
        self.assertIn('Affected cases: browser/case.', report)
        self.assertIn('do not establish instantaneous execution', report)
        self.assertIn('| browser/case | 4 (below measurement resolution) |', report)
        self.assertNotIn('ranking_issues', row)
        row['distribution'].pop('min')
        row['distribution']['median'] = 0
        self.assertIn('0 (below measurement resolution)', markdown_report(manifest, summary))

    def test_zero_count_and_unsupported_units_are_not_timing_limits(self):
        manifest, summary, _ = fixtures()
        for row in summary.values():
            row['unit'] = 'bytes'
            row['distribution'].update(median=0, min=0)
        self.assertNotIn('below measurement resolution', markdown_report(manifest, summary))

    def test_all_twelve_and_supplied_rank_order(self):
        manifest, summary, boards = fixtures()
        report = markdown_report(manifest, summary, boards)
        self.assertIn("full-system suite selection", report)
        self.assertIn("12/12 requested suites completed", report)
        for suite in DEFAULT_SUITES:
            self.assertIn(f"| {suite} |", report)
        target = report.split("## Top 10 target shortfalls")[1].split("## Top 10 relative regressions")[0]
        self.assertIn("| 1 | journeys/case |", target)
        self.assertIn("| 10 | artifacts/case |", target)
        self.assertNotIn("assertions/case", target)
        self.assertNotIn("compiler/case", target)
        self.assertIn("preparation drivers **2 s**; suite drivers **36 s**", report)
        self.assertIn("revision: abc123", report)
        self.assertIn("dirty\\_checkout: True", report)
        self.assertIn("Independent n", report)
        self.assertIn("single call", report)
        self.assertNotIn("p95", report)
        self.assertNotIn("percentile", report)

    def test_subset_and_suite_states(self):
        manifest, summary, boards = fixtures()
        manifest["requested_suites"] = ["runtime", "compiler", "startup"]
        manifest["completed_suites"] = ["runtime"]
        manifest["steps"] = [{"phase": "trial", "suite": "compiler", "elapsed_seconds": 1}]
        report = markdown_report(manifest, {"runtime/case": summary["runtime/case"]}, boards)
        self.assertIn("subset suite selection", report)
        self.assertIn("| complete | 1 |", report)
        self.assertIn("| incomplete | 0 |", report)
        self.assertIn("| not run | 0 |", report)
        self.assertIn("| not requested | 0 |", report)

    def test_escape_untrusted_case_and_prose(self):
        manifest, summary, boards = fixtures()
        manifest["reason"] = "bad | *text* <script>\nnext"
        summary = {"runtime/a|[link]": summary["runtime/case"]}
        summary["runtime/a|[link]"]["parameters"] = {"a": "<img>|\n"}
        summary["runtime/a|[link]"]["timing_scope"] = "x|y **z**"
        report = markdown_report(manifest, summary)
        self.assertIn("bad \\| \\*text\\* &lt;script&gt;<br>next", report)
        self.assertIn("runtime/a\\|\\[link\\]", report)
        self.assertIn("x\\|y \\*\\*z\\*\\*", report)
        self.assertNotIn("<img>", report)

    def test_smoke_and_failed_do_not_show_positions(self):
        for field, value in (("quality", "smoke"), ("status", "failed")):
            manifest, summary, boards = fixtures()
            manifest[field] = value
            manifest["reason"] = "driver stopped"
            report = markdown_report(manifest, summary, boards)
            self.assertIn("ineligible for performance rankings", report)
            self.assertIn("No ranked positions available", report)
            self.assertNotIn("| 1 | journeys/case |", report)
            self.assertLess(report.index("driver stopped"), report.index("Suite overview"))

    def test_missing_references_and_empty_ranked_board(self):
        manifest, summary, boards = fixtures()
        report = markdown_report(manifest, summary)
        self.assertIn("no compatible references supplied", report)
        self.assertNotIn("| Position |", report)
        boards["targets"]["rows"] = []
        report = markdown_report(manifest, summary, boards)
        self.assertIn("No target exceedances", report)
        self.assertIn("No relative regressions", report)

    def test_non_exceeding_eligible_cases_and_ranking_issues(self):
        manifest, summary, boards = fixtures()
        boards["targets"]["rows"] = []
        summary["server/case"]["ranking_issues"] = ["offered requests not delivered"]
        report = markdown_report(manifest, summary, boards)
        runtime = next(line for line in report.splitlines() if line.startswith("| runtime |"))
        self.assertIn("| 1 / 1 |", runtime)
        self.assertIn("offered requests not delivered", report)

    def test_exploratory_suppresses_arbitrary_supplied_rankings(self):
        manifest, summary, boards = fixtures()
        manifest["quality"] = "exploratory"
        report = markdown_report(manifest, summary, boards)
        self.assertIn("INELIGIBLE for rankings and baselines", report)
        self.assertNotIn("| Position |", report)
        self.assertNotIn("Eligible cases: 12/12", report)
        self.assertNotIn("Comparison status: **ranked**", report)
        self.assertNotIn("| 1 | journeys/case |", report)
        runtime = next(line for line in report.splitlines() if line.startswith("| runtime |"))
        self.assertIn("| 0 / 0 |", runtime)

    def test_full_provenance_is_collapsed_and_escaped(self):
        manifest, summary, boards = fixtures()
        manifest["source"]["dependencies"] = {"nested": "<unsafe>&value"}
        report = markdown_report(manifest, summary, boards)
        concise = next(line for line in report.splitlines() if line.startswith("Source provenance:"))
        self.assertNotIn("dependencies", concise)
        self.assertIn("abc123", concise)
        self.assertIn("<summary>Full source provenance</summary>", report)
        self.assertIn("&lt;unsafe&gt;&amp;value", report)
        self.assertNotIn("<unsafe>", report)

    def test_partial_empty_board_and_normalized_percent_heading(self):
        manifest, summary, boards = fixtures()
        boards["targets"] = dict(boards["targets"], status="partial", eligible_cases=11, rows=[])
        report = markdown_report(manifest, summary, boards)
        self.assertIn("No target exceedances among eligible cases", report)
        self.assertIn("Partial reference coverage", report)
        self.assertIn("Ratio excess (%)", report)
        self.assertNotIn("Worse (%)", report)

    def test_effective_reference_provenance_and_escaping(self):
        manifest, summary, boards = fixtures()
        manifest["ranking_references"] = {
            "targets": {"name": "budget | <unsafe>", "sha256": "target-hash"},
            "baseline": {"path": "/some/<path>", "source": {"revision": "rev*123", "source_mode": "committed"},
                         "verified_summary_sha256": "baseline-hash"},
        }
        report = markdown_report(manifest, summary, boards)
        self.assertIn(r"Target reference: budget \| &lt;unsafe&gt;", report)
        self.assertIn(r"Baseline reference: revision: rev\*123; source_mode: committed", report)
        self.assertIn("target-hash", report)
        self.assertIn("baseline-hash", report)
        self.assertIn("/some/&lt;path&gt;", report)
        baseline_label = next(line for line in report.splitlines() if line.startswith("Baseline reference:"))
        self.assertNotIn("/some/", baseline_label)
        self.assertEqual(report.count("<summary>Comparison provenance and SHA256</summary>"), 2)

    def test_exclusions_partial_and_unavailable(self):
        manifest, summary, boards = fixtures()
        boards["targets"] = {"status": "partial", "reason": "missing one reference",
                             "eligible_cases": 11, "total_cases": 12,
                             "exclusions": {"runtime/case": "no target"}, "rows": []}
        boards["baseline"] = {"status": "unavailable", "reason": "no baseline",
                              "eligible_cases": 0, "total_cases": 12, "exclusions": {}, "rows": []}
        report = markdown_report(manifest, summary, boards)
        runtime = next(line for line in report.splitlines() if line.startswith("| runtime |"))
        self.assertIn("| 0 / 0 |", runtime)
        self.assertIn("runtime/case: no target", report)
        self.assertIn("Eligible cases: 11/12", report)
        self.assertIn("no baseline", report)


if __name__ == "__main__":
    unittest.main()
