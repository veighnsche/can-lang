"""Offline command integration against tiny, complete synthetic evidence."""

import contextlib
import hashlib
import io
import json
from pathlib import Path
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import patch
import zipfile

import perf
from ranking import SUITE_UNITS, target_template
from schema import summarize_trials


class ReportCommandsTests(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory(prefix="can-report-test-")
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)
        for name in ("execute", "source_snapshot"):
            guard = patch.object(perf, name, side_effect=AssertionError("Offline report launched work"))
            guard.start()
            self.addCleanup(guard.stop)

    def make_run(self, name, *, scale=1, quality="measurement", status="complete", change=None):
        root = self.root / name
        root.mkdir()
        manifest = {
            "schema_version": 1, "status": status, "quality": quality,
            "requested_suites": list(perf.SUITES), "completed_suites": list(perf.SUITES),
            "profile": "quick", "trials": 1, "warmups": 0, "iterations": 1, "size": 10,
            "environment": {}, "isolation": {},
            "source": {"revision": name, "dependencies": None, "harness_overlays": {}},
            "steps": [],
        }
        trials = []
        for index, suite in enumerate(perf.SUITES, 1):
            case = {
                "name": "bounded", "unit": SUITE_UNITS[suite],
                "samples": [index * scale] * 3, "warmup_samples": [],
                "iterations_per_sample": 1, "timing_scope": "bounded synthetic operation",
                "parameters": {"size": 10},
                "correctness": {"passed": True, "checks": ["semantic result matches"]},
            }
            if suite == "server":
                case["parameters"]["request_count"] = 1
                case["metrics"] = {"trials": [
                    {"warmup": False, "scheduled_count": 1, "completed": 1,
                     "dropped_by_generator": 0, "errors": 0,
                     "requests": [{"id": 0, "correct": True}]} for _ in range(3)
                ]}
            result = {"schema_version": 1, "suite": suite, "status": "complete", "cases": [case]}
            trials.append({"suite": suite, "trial": 1, "result": result})
            perf.write_json(root / "raw" / f"{suite}-001.json", result)
        summary = summarize_trials(trials)
        data = perf.report_data(manifest, summary)
        perf.write_json(root / "manifest.json", manifest)
        perf.write_json(root / "summary.json", summary)
        perf.write_json(root / "report.json", data)
        (root / "report.md").write_text(perf.markdown_report(manifest, summary, data["rankings"]))
        if change:
            change(root)
        perf.archive_evidence(root)
        return root, manifest, summary

    def report(self, root, *, targets=None, baseline=None, format="json", output=None):
        capture = io.StringIO()
        with contextlib.redirect_stdout(capture):
            result = perf.report(SimpleNamespace(run=str(root), targets=targets, baseline=baseline,
                                                 format=format, output=output))
        self.assertEqual(result, 0)
        return capture.getvalue()

    def template(self, root, name="targets.json"):
        output = self.root / name
        with contextlib.redirect_stdout(io.StringIO()):
            self.assertEqual(perf.targets_command(SimpleNamespace(run=str(root), output=str(output))), 0)
        return output, json.loads(output.read_text())

    def populated_targets(self, root):
        path, targets = self.template(root)
        for row in targets["cases"].values():
            row.update(target=0.5, rationale="Explicit synthetic budget")
        perf.write_json(path, targets)
        return str(path)

    def test_stored_run_reports_survive_archive_and_offline_read(self):
        root, manifest, summary = self.make_run("stored")
        expected = perf.report_data(manifest, summary)
        with zipfile.ZipFile(root / "evidence.zip") as archive:
            self.assertIn("report.md", archive.namelist())
            self.assertIn("report.json", archive.namelist())
            markdown = archive.read("report.md").decode()
            original_json = archive.read("report.json").decode()
        self.assertFalse((root / "raw").exists())
        self.assertEqual(json.loads(self.report(root)), expected)
        self.assertEqual(self.report(root), original_json)
        self.assertEqual(self.report(root, format="markdown"), markdown)
        self.assertEqual(self.report(root / "evidence.zip"), original_json)

    def test_template_contains_every_contract_and_no_observed_target(self):
        root, _, summary = self.make_run("template")
        _, template = self.template(root)
        self.assertEqual(set(template["cases"]), set(summary))
        self.assertEqual(len(template["cases"]), 12)
        for key, row in template["cases"].items():
            self.assertIsNone(row["target"])
            self.assertEqual(row["rationale"], "")
            self.assertNotIn("distribution", row)
            self.assertNotIn("observed", row)
            for field in ("unit", "parameters", "timing_scope", "iterations_per_sample"):
                self.assertEqual(row[field], summary[key][field])

    def test_recompute_verified_raw_targets_and_baseline_sorted_top_ten(self):
        baseline, _, _ = self.make_run("baseline", scale=0.5)
        candidate, _, summary = self.make_run("candidate")
        targets = self.populated_targets(candidate)
        before = (candidate / "evidence.zip").read_bytes()
        data = json.loads(self.report(candidate, targets=targets, baseline=str(baseline)))
        expected = sorted(summary, key=lambda key: (-summary[key]["distribution"]["median"], key))
        target_board = data["rankings"]["targets"]
        self.assertEqual(target_board["eligible_cases"], 12)
        self.assertEqual([row["case"] for row in target_board["rows"]], expected)
        self.assertEqual(len(data["rankings"]["baseline"]["rows"]), 12)
        self.assertTrue(all(row["ratio"] == 2 for row in data["rankings"]["baseline"]["rows"]))
        markdown = self.report(candidate, targets=targets, baseline=str(baseline), format="markdown")
        target_section = markdown.split("## Top 10 target shortfalls\n", 1)[1].split("## Top 10 relative regressions", 1)[0]
        ranked_lines = [line for line in target_section.splitlines() if line.startswith("| ") and line.split(" | ")[0][2:].isdigit()]
        self.assertEqual(len(ranked_lines), 10)
        self.assertEqual([line.split(" | ")[1] for line in ranked_lines], expected[:10])
        self.assertEqual((candidate / "evidence.zip").read_bytes(), before)

    def test_forged_summary_and_raw_rejected_for_recompute_and_templates(self):
        def forge_summary(root):
            path = root / "summary.json"
            value = json.loads(path.read_text())
            value["compiler/bounded"]["distribution"]["median"] = 999
            perf.write_json(path, value)

        def forge_raw(root):
            path = root / "raw/compiler-001.json"
            value = json.loads(path.read_text())
            value["cases"][0]["correctness"]["passed"] = False
            perf.write_json(path, value)

        clean, _, _ = self.make_run("clean")
        targets = self.populated_targets(clean)
        for name, change in (("summary", forge_summary), ("raw", forge_raw)):
            with self.subTest(name=name):
                root, _, _ = self.make_run(name, change=change)
                with self.assertRaises(ValueError):
                    self.report(root, targets=targets)
                with self.assertRaises(ValueError):
                    self.template(root, name + "-targets.json")

    def test_incompatible_baseline_visible_as_unavailable(self):
        candidate, _, _ = self.make_run("candidate")
        def incompatible(root):
            path = root / "manifest.json"
            value = json.loads(path.read_text())
            value["environment"] = {"different_host": True}
            perf.write_json(path, value)
        baseline, _, _ = self.make_run("incompatible", change=incompatible)
        data = json.loads(self.report(candidate, baseline=str(baseline)))
        board = data["rankings"]["baseline"]
        self.assertEqual(board["status"], "unavailable")
        self.assertEqual(board["rows"], [])
        self.assertIn("context differs", board["reason"])
        markdown = self.report(candidate, baseline=str(baseline), format="markdown")
        self.assertIn("Comparison status: **unavailable**", markdown)
        self.assertIn("Baseline comparison context differs", markdown)

    def test_smoke_and_failed_runs_read_stored_but_reject_recompute_and_templates(self):
        clean, _, _ = self.make_run("clean")
        targets = self.populated_targets(clean)
        for name, options in (("smoke", {"quality": "smoke"}), ("failed", {"status": "failed"})):
            with self.subTest(name=name):
                root, _, _ = self.make_run(name, **options)
                self.assertIn("ineligible", self.report(root, format="markdown"))
                with self.assertRaises(ValueError):
                    self.report(root, targets=targets)
                with self.assertRaises(ValueError):
                    self.template(root, name + "-targets.json")

    def test_outputs_refuse_overwrite_and_preserve_original_archive(self):
        root, _, _ = self.make_run("immutable")
        archive = root / "evidence.zip"
        before = archive.read_bytes()
        targets = self.populated_targets(root)
        output = self.root / "new-report.json"
        self.report(root, targets=targets, output=str(output))
        self.assertEqual(json.loads(output.read_text())["rankings"]["targets"]["eligible_cases"], 12)
        original = output.read_bytes()
        with self.assertRaises(FileExistsError):
            self.report(root, targets=targets, output=str(output))
        self.assertEqual(output.read_bytes(), original)
        with self.assertRaises(FileExistsError):
            self.report(root, output=str(archive))
        with self.assertRaises(FileExistsError):
            self.template(root)
        self.assertEqual(archive.read_bytes(), before)

    def test_recomputed_report_records_effective_reference_provenance(self):
        baseline, baseline_manifest, baseline_summary = self.make_run("new-baseline", scale=0.5)
        def old_references(root):
            path = root / "manifest.json"
            value = json.loads(path.read_text())
            value["ranking_references"] = {
                "targets": {"name": "old targets", "sha256": "old hash"},
                "baseline": {"path": "old baseline", "source": {}, "verified_summary_sha256": "old hash"},
            }
            perf.write_json(path, value)
        candidate, _, _ = self.make_run("new-candidate", change=old_references)
        targets_path = self.populated_targets(candidate)
        targets = json.loads(Path(targets_path).read_text())
        before = (candidate / "evidence.zip").read_bytes()
        data = json.loads(self.report(candidate, targets=targets_path, baseline=str(baseline)))
        def canonical_hash(value):
            return hashlib.sha256(json.dumps(value, sort_keys=True, allow_nan=False).encode()).hexdigest()
        self.assertEqual(data["manifest"]["ranking_references"], {
            "targets": {"name": targets["name"], "sha256": canonical_hash(targets)},
            "baseline": {"path": str(baseline.resolve()), "source": baseline_manifest["source"],
                         "verified_summary_sha256": canonical_hash(baseline_summary)},
        })
        self.assertEqual((candidate / "evidence.zip").read_bytes(), before)
        targets_only = json.loads(self.report(candidate, targets=targets_path))
        self.assertIsNone(targets_only["manifest"]["ranking_references"]["baseline"])

    def test_embedded_targets_reused_only_when_recorded_hash_matches(self):
        baseline, _, _ = self.make_run("embedded-baseline", scale=0.5)
        def embed_targets(root, *, edit=False):
            path = root / "manifest.json"
            manifest = json.loads(path.read_text())
            summary = json.loads((root / "summary.json").read_text())
            targets = target_template(manifest, summary)
            for row in targets["cases"].values():
                row.update(target=0.5, rationale="Reviewed synthetic budget")
            original_hash = hashlib.sha256(json.dumps(targets, sort_keys=True, allow_nan=False).encode()).hexdigest()
            manifest["ranking_references"] = {
                "targets": {"name": targets["name"], "sha256": original_hash}, "baseline": None,
            }
            if edit:
                targets["cases"]["compiler/bounded"]["target"] = 0.25
            perf.write_json(path, manifest)
            perf.write_json(root / "raw/ranking-targets.json", targets)
        valid, _, _ = self.make_run("embedded-valid", change=embed_targets)
        before = (valid / "evidence.zip").read_bytes()
        data = json.loads(self.report(valid, baseline=str(baseline)))
        self.assertEqual(data["rankings"]["targets"]["eligible_cases"], 12)
        with zipfile.ZipFile(valid / "evidence.zip") as archive:
            original_manifest = json.loads(archive.read("manifest.json"))
        self.assertEqual(data["manifest"]["ranking_references"]["targets"],
                         original_manifest["ranking_references"]["targets"])
        self.assertEqual((valid / "evidence.zip").read_bytes(), before)
        edited, _, _ = self.make_run("embedded-edited", change=lambda root: embed_targets(root, edit=True))
        before = (edited / "evidence.zip").read_bytes()
        with patch.object(perf, "build_rankings", side_effect=AssertionError("Changed targets reached ranking")):
            with self.assertRaisesRegex(ValueError, "recorded reference hash"):
                self.report(edited, baseline=str(baseline))
        self.assertEqual((edited / "evidence.zip").read_bytes(), before)

    def test_invalid_run_targets_rejected_before_execution_or_output_allocation(self):
        reference, _, _ = self.make_run("preflight-reference")
        path, template = self.template(reference, "preflight-targets.json")
        for name, value in (("malformed", {}), ("nonfinite", template)):
            with self.subTest(name=name):
                if name == "nonfinite":
                    row = value["cases"]["compiler/bounded"]
                    row.update(target=float("inf"), rationale="Invalid nonfinite test value")
                path.write_text(json.dumps(value))
                output = self.root / ("must-not-allocate-" + name)
                args = SimpleNamespace(
                    suite=["all"], smoke=False, profile="quick", trials=1, iterations=1,
                    warmups=0, size=10, targets=str(path), baseline=None, output=str(output),
                    revision=None, working_tree=False, keep_work=False, quiet_seconds=30,
                    idle_percent=90, wait_seconds=1, timeout=1, prepare_timeout=1,
                )
                with patch.object(perf, "source_snapshot", side_effect=AssertionError("Preflight copied source")) as source:
                    with patch.object(perf, "execute", side_effect=AssertionError("Preflight launched workload")) as execute:
                        with self.assertRaises(ValueError):
                            perf.run(args)
                        source.assert_not_called()
                        execute.assert_not_called()
                self.assertFalse(output.exists())


if __name__ == "__main__":
    unittest.main()
