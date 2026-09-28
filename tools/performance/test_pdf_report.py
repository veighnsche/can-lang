"""Bounded PDF export tests with fake and installed Typst; no workloads run."""

import contextlib
import io
import json
import os
import shutil
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

import pdf_report


def evidence():
    return {"kind": "can.performance-report", "schema_version": 1,
            "manifest": {"status": "complete"}, "cases": {}, "rankings": {}}


class PdfReportTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        self.output = self.root / "result.pdf"
        self.compile_roots = []
        self.template = self.root / "report.typ"
        self.template.write_text('#json("report.json")', encoding="utf-8")
        self.module_path = patch.object(pdf_report, "__file__", str(self.root / "pdf_report.py"))
        self.module_path.start()
        self.addCleanup(self.module_path.stop)
        self.which = patch.object(pdf_report.shutil, "which", return_value="/fake/typst")
        self.which.start()
        self.addCleanup(self.which.stop)

    def compile(self, args, **kwargs):
        root = Path(args[args.index("--root") + 1])
        self.compile_roots.append(root)
        self.assertEqual(kwargs["cwd"], root)
        self.assertEqual(kwargs["timeout"], 30)
        self.assertTrue(kwargs["capture_output"])
        self.assertEqual(args[:6], ["/fake/typst", "compile", "--jobs", "1", "--ignore-system-fonts", "--root"])
        self.assertEqual(Path(args[-2]), root / "report.typ")
        self.assertEqual(Path(args[-1]), root / "report.pdf")
        self.assertEqual((root / "report.typ").read_text(), self.template.read_text())
        self.assertEqual(json.loads((root / "report.json").read_text()), evidence())
        Path(args[-1]).write_bytes(b"%PDF-1.7\nsynthetic")
        return subprocess.CompletedProcess(args, 0, "", "")

    def assert_clean(self):
        for root in self.compile_roots:
            self.assertFalse(root.exists())
        self.assertFalse(list(self.root.glob(".can-performance-pdf-*")))

    def test_success_and_compile_context(self):
        with patch.object(pdf_report.subprocess, "run", side_effect=self.compile):
            pdf_report.write_pdf(evidence(), self.output)
        self.assertEqual(self.output.read_bytes(), b"%PDF-1.7\nsynthetic")
        self.assert_clean()

    def test_output_expands_home_directory(self):
        with patch.dict(os.environ, {"HOME": str(self.root)}):
            with patch.object(pdf_report.subprocess, "run", side_effect=self.compile):
                pdf_report.write_pdf(evidence(), "~/result.pdf")
        self.assertTrue(self.output.is_file())
        self.assert_clean()

    def test_existing_file_and_dangling_symlink_preserved(self):
        self.output.write_bytes(b"original")
        with self.assertRaises(FileExistsError):
            pdf_report.write_pdf(evidence(), self.output)
        self.assertEqual(self.output.read_bytes(), b"original")
        self.output.unlink()
        self.output.symlink_to(self.root / "missing")
        with self.assertRaises(FileExistsError):
            pdf_report.write_pdf(evidence(), self.output)
        self.assertTrue(self.output.is_symlink())

    def test_missing_compiler(self):
        with patch.object(pdf_report.shutil, "which", return_value=None):
            with self.assertRaisesRegex(RuntimeError, "installed typst"):
                pdf_report.write_pdf(evidence(), self.output)
        self.assertFalse(self.output.exists())

    def test_invalid_input(self):
        for data in (None, {}, dict(evidence(), schema_version=2), dict(evidence(), manifest=[]),
                     dict(evidence(), cases={"x": []}), dict(evidence(), rankings={"x": []})):
            with self.subTest(data=data), self.assertRaises(ValueError):
                pdf_report.write_pdf(data, self.output)
        with self.assertRaisesRegex(ValueError, "extension"):
            pdf_report.write_pdf(evidence(), self.root / "result.txt")

    def test_compile_failures_cleanup(self):
        for failure in ("timeout", "nonzero", "launch", "missing", "invalid"):
            def fail(args, **kwargs):
                self.compile(args, **kwargs)
                if failure == "timeout":
                    raise subprocess.TimeoutExpired(args, 30)
                if failure == "launch":
                    raise OSError("compiler unavailable")
                if failure == "nonzero":
                    return subprocess.CompletedProcess(args, 1, "", "bad syntax" * 1000)
                compiled = Path(args[-1])
                if failure == "missing":
                    compiled.unlink()
                else:
                    compiled.write_bytes(b"not a PDF")
                return subprocess.CompletedProcess(args, 0, "", "")
            with self.subTest(failure=failure), patch.object(pdf_report.subprocess, "run", side_effect=fail):
                with self.assertRaises(RuntimeError) as caught:
                    pdf_report.write_pdf(evidence(), self.output)
                self.assertLess(len(str(caught.exception)), 2100)
                self.assertFalse(self.output.exists())
                self.assert_clean()

    def test_publish_race_and_copy_failure_cleanup(self):
        for failure in ("race", "copy"):
            original_link = os.link
            def race(source, target):
                Path(target).write_bytes(b"concurrent")
                original_link(source, target)
            mutation = (patch.object(pdf_report.os, "link", side_effect=race) if failure == "race" else
                        patch.object(pdf_report.shutil, "copyfileobj", side_effect=OSError("disk full")))
            with self.subTest(failure=failure), patch.object(pdf_report.subprocess, "run", side_effect=self.compile), mutation:
                with self.assertRaises(OSError):
                    pdf_report.write_pdf(evidence(), self.output)
            if failure == "race":
                self.assertEqual(self.output.read_bytes(), b"concurrent")
                self.output.unlink()
            else:
                self.assertFalse(self.output.exists())
            self.assert_clean()


@unittest.skipUnless(shutil.which("typst"), "Typst is not installed")
class InstalledTypstTests(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory(prefix="can-pdf-template-test-")
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)

    def compile_report(self, quality):
        # Deliberately omit `synthetic`: real reports take the quality header
        # branch that synthetic previews cannot exercise.
        data = evidence()
        data["manifest"].update(
            quality=quality, requested_suites=["runtime", "browser"], completed_suites=["runtime", "browser"],
            reason='#panic("evidence must stay literal")',
        )
        data["cases"] = {"runtime/tiny": {
            "unit": "ns/op", "timing_scope": "one bounded operation", "parameters": {"size": 1},
            "distribution": {"median": 2, "min": 1, "max": 3,
                             "median_absolute_deviation": 1, "n": 3},
        }, "browser/unresolved": {
            "unit": "ms/interaction", "timing_scope": "one callback", "parameters": {"size": 1},
            "distribution": {"median": 0, "min": 0, "max": 0.1,
                             "median_absolute_deviation": 0, "n": 3},
        }}
        # Measurement renders this row; smoke follows the ineligible branch.
        data["rankings"] = {"targets": {
            "status": "partial", "eligible_cases": 1, "total_cases": 2,
            "per_case": {"runtime/tiny": {}}, "rows": [{
                "case": "runtime/tiny", "unit": "ns/op", "observed": 2,
                "reference": 1, "ratio": 2, "trial_count": 3, "mad": 1,
            }],
        }}
        output = self.root / f"{quality}.pdf"
        pdf_report.write_pdf(data, output)
        with output.open("rb") as result:
            self.assertEqual(result.read(5), b"%PDF-")
        self.assertGreater(output.stat().st_size, 1000)
        self.assertEqual(list(self.root.iterdir()), [output])

    def test_measurement_template_compiles(self):
        self.compile_report("measurement")

    def test_smoke_template_compiles(self):
        self.compile_report("smoke")

    def render(self, data, name):
        output = self.root / f"{name}.pdf"
        pdf_report.write_pdf(data, output)
        with output.open("rb") as result:
            self.assertEqual(result.read(5), b"%PDF-")
        self.assertFalse(list(self.root.glob(".can-performance-pdf-*")))
        try:
            from pypdf import PdfReader
        except ImportError:
            return None
        text = "\n".join(page.extract_text() or "" for page in PdfReader(output).pages)
        return " ".join(text.replace("\u200b", "").split())

    def generated_report(self):
        data = evidence()
        data["manifest"].update(quality="measurement", requested_suites=["generated"],
                                completed_suites=["generated"])
        return data

    def case(self, median, **changes):
        row = {"unit": "ns/op", "parameters": {"size": 1}, "timing_scope": "complete workload",
               "iterations_per_sample": 1,
               "distribution": {"median": median, "min": median, "max": median,
                                "median_absolute_deviation": 0, "n": 3}}
        row.update(changes)
        return row

    def add_pair(self, data, name, *, adapter=False, native_changes=None, can_changes=None):
        prefix = "generated/generated." + name
        data["cases"][prefix + ".can"] = self.case(4000, **(can_changes or {}))
        data["cases"][prefix + (".native-adapter" if adapter else ".native")] = self.case(
            1000, **(native_changes or {}))

    def test_matched_native_and_adapter_pairs_exclude_incompatible_contracts(self):
        data = self.generated_report()
        self.add_pair(data, "frequency")
        self.add_pair(data, "record-update", adapter=True)
        for name, changes in (
            ("wrong-parameters", {"parameters": {"size": 2}}),
            ("wrong-unit", {"unit": "ms/op"}),
            ("wrong-scope", {"timing_scope": "different timer boundary"}),
            ("wrong-iterations", {"iterations_per_sample": 2}),
            ("native-issue", {"ranking_issues": ["Incomplete correctness evidence"]}),
        ):
            self.add_pair(data, name, native_changes=changes)
        self.add_pair(data, "can-issue", can_changes={"ranking_issues": ["Incomplete delivery"]})
        for name, unit in (("not-time", "bytes"), ("throughput", "ops/s")):
            self.add_pair(data, name, native_changes={"unit": unit}, can_changes={"unit": unit})
        zero = {"median": 0, "min": 0, "max": 0, "median_absolute_deviation": 0, "n": 3}
        self.add_pair(data, "zero-native", native_changes={"distribution": zero})
        self.add_pair(data, "zero-can", can_changes={"distribution": zero})
        text = self.render(data, "matched-pairs")
        if text is not None:
            self.assertIn("Across 2 matched workloads", text)
            section = text.split("Generated code vs native", 1)[1].split("All measured results", 1)[0]
            self.assertIn("Word frequency", section)
            self.assertIn("Immutable record update", section)
            for excluded in ("wrong-parameters", "wrong-unit", "wrong-scope", "wrong-iterations",
                             "native-issue", "can-issue", "not-time", "throughput", "zero-native", "zero-can"):
                self.assertNotIn(excluded.replace("-", " "), section)
            self.assertEqual(section.count("4×"), 2)

    def test_missing_references_many_cases_do_not_create_rankings_or_exclusion_pages(self):
        data = self.generated_report()
        data["cases"] = {f"runtime/workload-{i}": self.case(1000 + i) for i in range(24)}
        data["manifest"].update(requested_suites=["runtime"], completed_suites=["runtime"])
        reason = "No reviewed target profile supplied"
        data["rankings"] = {"targets": {
            "status": "unavailable", "rows": [], "exclusions": dict.fromkeys(data["cases"], reason),
        }}
        text = self.render(data, "missing-references")
        if text is not None:
            self.assertIn("All 24 case results follow", text)
            self.assertIn("A defensible top ten", text)
            self.assertNotIn("Cross-slice priorities", text)
            self.assertNotIn("Comparison coverage", text)
            self.assertNotIn(reason, text)

    def test_partial_reference_exclusions_group_repeated_reasons(self):
        data = self.generated_report()
        data["cases"] = {f"runtime/workload-{i}": self.case(1000 + i) for i in range(12)}
        data["manifest"].update(requested_suites=["runtime"], completed_suites=["runtime"],
                                ranking_references={"targets": {"name": "Reviewed small profile"}})
        exclusions = {key: "Missing target" if i < 8 else "Contract differs"
                      for i, key in enumerate(data["cases"])}
        data["rankings"] = {"targets": {
            "status": "partial", "eligible_cases": 0, "total_cases": 12,
            "rows": [], "per_case": {}, "exclusions": exclusions,
        }}
        text = self.render(data, "partial-exclusions")
        if text is not None:
            self.assertIn("Partial coverage", text)
            section = text.split("Comparison coverage", 1)[1].split("Interpretation limits", 1)[0]
            self.assertIn("Missing target 8", section)
            self.assertIn("Contract differs 4", section)
            # Each reason occurs once in the count table and once above its case list.
            self.assertEqual(section.count("Missing target"), 2)
            self.assertEqual(section.count("Contract differs"), 2)
            self.assertIn("runtime/workload-0", section)
            self.assertIn("runtime/workload-11", section)

    def test_smoke_and_incomplete_reports_suppress_same_run_ratios(self):
        for quality, status in (("smoke", "complete"), ("measurement", "failed")):
            with self.subTest(quality=quality, status=status):
                data = self.generated_report()
                data["manifest"].update(quality=quality, status=status)
                self.add_pair(data, "frequency")
                data["rankings"] = {"targets": {
                    "status": "ranked", "eligible_cases": 1, "total_cases": 2,
                    "per_case": {"generated/generated.frequency.can": {}},
                    "rows": [{"case": "INELIGIBLE_RANK_SENTINEL", "unit": "ns/op", "observed": 4000,
                              "reference": 1000, "ratio": 4, "trial_count": 3, "mad": 0}],
                }}
                text = self.render(data, quality + "-" + status)
                if text is not None:
                    self.assertIn("Not eligible for performance rankings", text)
                    self.assertIn("Ineligible evidence. No ranked positions are shown", text)
                    self.assertNotIn("Generated code vs native", text)
                    self.assertNotIn("Across 1 matched workloads", text)
                    self.assertNotIn("INELIGIBLE_RANK_SENTINEL", text)


class PdfCommandTests(unittest.TestCase):
    def setUp(self):
        import perf
        self.perf = perf
        temporary = tempfile.TemporaryDirectory(prefix="can-pdf-command-test-")
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)
        for name in ("execute", "source_snapshot"):
            guard = patch.object(perf, name, side_effect=AssertionError("PDF export launched workloads"))
            guard.start()
            self.addCleanup(guard.stop)

    def command(self, *arguments):
        with patch.object(sys, "argv", ["perf.py", "report", *arguments]):
            with contextlib.redirect_stdout(io.StringIO()):
                return self.perf.main()

    def make_run(self, name, **kwargs):
        from test_report_commands import ReportCommandsTests
        return ReportCommandsTests.make_run(self, name, **kwargs)

    def test_cli_requires_output_before_reading_evidence(self):
        with patch.object(self.perf, "read_evidence", side_effect=AssertionError("Read before validation")):
            with contextlib.redirect_stderr(io.StringIO()) as errors:
                with self.assertRaises(SystemExit) as caught:
                    self.command(str(self.root / "missing"), "--format", "pdf")
        self.assertEqual(caught.exception.code, 2)
        self.assertIn("requires --output", errors.getvalue())

    def test_cli_passes_exact_stored_json_from_archive(self):
        injected = '#include "/private/secret"; #panic("injected")'
        def add_literal(root):
            path = root / "report.json"
            data = json.loads(path.read_text())
            data["manifest"]["reason"] = injected
            self.perf.write_json(path, data)
        run, manifest, summary = self.make_run("stored", change=add_literal)
        expected = self.perf.report_data(manifest, summary)
        expected["manifest"]["reason"] = injected
        with patch.object(self.perf, "read_evidence", wraps=self.perf.read_evidence) as read:
            with patch.object(pdf_report, "write_pdf") as export:
                self.assertEqual(self.command(str(run / "evidence.zip"), "--format", "pdf",
                                              "--output", "~/export.pdf", "--no-grades"), 0)
        read.assert_called_once_with((run / "evidence.zip").resolve(), "report.json")
        export.assert_called_once_with(expected, "~/export.pdf")

    def test_cli_recomputes_targets_and_baseline_without_workloads(self):
        from ranking import target_template
        baseline, _, _ = self.make_run("baseline", scale=0.5)
        run, manifest, summary = self.make_run("candidate")
        targets = target_template(manifest, summary)
        for row in targets["cases"].values():
            row.update(target=0.5, rationale="Explicit synthetic budget")
        path = self.root / "targets.json"
        self.perf.write_json(path, targets)
        before = (run / "evidence.zip").read_bytes()
        with patch.object(pdf_report, "write_pdf") as export:
            self.assertEqual(self.command(str(run), "--format", "pdf", "--output", "export.pdf",
                                          "--targets", str(path), "--baseline", str(baseline), "--no-grades"), 0)
        data, output = export.call_args.args
        self.assertEqual(output, "export.pdf")
        self.assertEqual(data["kind"], "can.performance-report")
        self.assertEqual(data["cases"], summary)
        self.assertEqual(data["rankings"]["targets"]["eligible_cases"], 12)
        self.assertEqual(data["rankings"]["baseline"]["eligible_cases"], 12)
        self.assertTrue(all(row["ratio"] == 2 for row in data["rankings"]["baseline"]["rows"]))
        self.assertEqual((run / "evidence.zip").read_bytes(), before)

    def test_default_measured_pdf_grades_verified_raw_evidence(self):
        import jev_service
        run, manifest, summary = self.make_run("automatic")
        output = self.root / "automatic.pdf"
        before = (run / "evidence.zip").read_bytes()
        enriched = dict(self.perf.report_data(manifest, summary), assessment={"advisory": True})
        with patch.object(self.perf.shutil, "which", return_value="/fake/typst"):
            with patch.object(jev_service, "run_grading", return_value=enriched) as grade:
                with patch.object(pdf_report, "write_pdf") as export:
                    self.command(str(run), "--format", "pdf", "--output", str(output))
        grade.assert_called_once_with(self.perf.report_data(manifest, summary), output.with_suffix(".grading.zip"))
        export.assert_called_once_with(enriched, str(output))
        self.assertEqual((run / "evidence.zip").read_bytes(), before)

    def test_default_pdf_raw_forgery_is_rejected_before_network(self):
        import jev_service
        def forge(root):
            path = root / "report.json"
            data = json.loads(path.read_text())
            data["cases"]["compiler/bounded"]["distribution"]["median"] = 999
            self.perf.write_json(path, data)
        run, _, _ = self.make_run("forged-pdf", change=forge)
        with patch.object(self.perf.shutil, "which", return_value="/fake/typst"):
            with patch.object(jev_service, "run_grading") as grade:
                with contextlib.redirect_stderr(io.StringIO()):
                    with self.assertRaises(SystemExit):
                        self.command(str(run), "--format", "pdf", "--output", str(self.root / "forged.pdf"))
        grade.assert_not_called()

    def test_default_pdf_preflight_prevents_paid_calls(self):
        import jev_service
        run, _, _ = self.make_run("preflight")
        output = self.root / "preflight.pdf"
        scenarios = [(output, b"existing PDF"), (output.with_suffix(".grading.zip"), b"existing audit")]
        for occupied, contents in scenarios:
            occupied.write_bytes(contents)
            with patch.object(self.perf.shutil, "which", return_value="/fake/typst"):
                with patch.object(jev_service, "run_grading") as grade:
                    with contextlib.redirect_stderr(io.StringIO()):
                        with self.assertRaises(SystemExit):
                            self.command(str(run), "--format", "pdf", "--output", str(output))
            grade.assert_not_called()
            self.assertEqual(occupied.read_bytes(), contents)
            occupied.unlink()
        with patch.object(self.perf.shutil, "which", return_value=None):
            with patch.object(jev_service, "run_grading") as grade:
                with contextlib.redirect_stderr(io.StringIO()):
                    with self.assertRaises(SystemExit):
                        self.command(str(run), "--format", "pdf", "--output", str(output))
        grade.assert_not_called()

    def test_smoke_and_failed_pdf_omit_remote_grading(self):
        import jev_service
        for name, options in (("smoke-pdf", {"quality": "smoke"}), ("failed-pdf", {"status": "failed"})):
            run, manifest, summary = self.make_run(name, **options)
            with patch.object(self.perf.shutil, "which", return_value="/fake/typst"):
                with patch.object(jev_service, "run_grading") as grade:
                    with patch.object(pdf_report, "write_pdf") as export:
                        self.command(str(run), "--format", "pdf", "--output", str(self.root / (name + ".pdf")))
            grade.assert_not_called()
            export.assert_called_once()
            self.assertNotIn("assessment", export.call_args.args[0])


if __name__ == "__main__":
    unittest.main()
