"""Synthetic report checks; no drivers, builds or benchmarks are executed."""

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
