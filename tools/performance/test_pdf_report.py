"""Bounded PDF export tests using a fake compiler; no performance work runs."""

import contextlib
import io
import json
import os
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
                                              "--output", "~/export.pdf"), 0)
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
                                          "--targets", str(path), "--baseline", str(baseline)), 0)
        data, output = export.call_args.args
        self.assertEqual(output, "export.pdf")
        self.assertEqual(data["kind"], "can.performance-report")
        self.assertEqual(data["cases"], summary)
        self.assertEqual(data["rankings"]["targets"]["eligible_cases"], 12)
        self.assertEqual(data["rankings"]["baseline"]["eligible_cases"], 12)
        self.assertTrue(all(row["ratio"] == 2 for row in data["rankings"]["baseline"]["rows"]))
        self.assertEqual((run / "evidence.zip").read_bytes(), before)


if __name__ == "__main__":
    unittest.main()
