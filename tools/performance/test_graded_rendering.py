"""Bounded local rendering of synthetic advisory grades; no Jev or workloads."""

import copy
from pathlib import Path
import shutil
import tempfile
import unittest

from assessment import attach_assessment
import jev_grading
import pdf_report
from reporting import markdown_report
from test_jev_grading import case, responses, score_answer

try:
    from pypdf import PdfReader
except ImportError:
    PdfReader = None


def graded_evidence():
    data = {"kind": "can.performance-report", "schema_version": 1,
            "manifest": {"status": "complete", "quality": "measurement",
                         "source": {"revision": "tiny-render-fixture"},
                         "requested_suites": ["runtime", "browser"], "completed_suites": ["runtime", "browser"],
                         "profile": "quick", "trials": 3, "iterations": 2, "warmups": 0, "size": 10,
                         "steps": [], "environment": {}, "isolation": {}},
            "cases": {"runtime/collections": case("runtime"), "browser/callback": case("browser", 0)},
            "rankings": {}}
    requests = jev_grading.build_requests(data)
    replies = responses(requests)
    for index, score in enumerate((10 / 3, 11 / 3, 5)):
        replies[index]["answers"]["runtime"] = score_answer(requests[index]["questions"]["runtime"]["criteria"], score)
    return attach_assessment(data, jev_grading.make_payload(data, requests, replies))


class GradedMarkdownTests(unittest.TestCase):
    def test_grades_ranges_ungraded_and_project_rubric_disclosed(self):
        data = graded_evidence()
        text = markdown_report(data["manifest"], data["cases"], data["rankings"], assessment=data["assessment"])
        self.assertIn("advisory report card", text)
        self.assertIn("| runtime | **A-** | B+ to A+ |", text)
        self.assertIn("| browser | **U** | U |", text)
        self.assertIn("not TypeSafe-certified performance standards", text)
        self.assertIn("median of three Score", text)
        self.assertIn("not confidence intervals", text)

    def test_nonmeasurement_or_incomplete_evidence_suppresses_grades(self):
        for changes in ({"quality": "smoke"}, {"quality": "exploratory"}, {"status": "failed"}):
            data = graded_evidence()
            data["manifest"].update(changes)
            with self.subTest(changes=changes):
                text = markdown_report(data["manifest"], data["cases"], data["rankings"], assessment=data["assessment"])
                self.assertNotIn("advisory report card", text)
                self.assertNotIn("| runtime | **A-**", text)
                self.assertIn("ineligible", text.lower())

    def test_author_text_remains_literal(self):
        data = graded_evidence()
        data["assessment"]["slices"]["runtime"]["basis"] = '#panic("literal") | <script>unsafe</script> [link](bad)'
        text = markdown_report(data["manifest"], data["cases"], data["rankings"], assessment=data["assessment"])
        self.assertIn("&lt;script&gt;", text)
        self.assertNotIn("<script>unsafe", text)
        self.assertIn("\\|", text)


@unittest.skipUnless(shutil.which("typst"), "Typst is not installed")
class GradedPdfTests(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory(prefix="can-graded-render-test-")
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)

    def compile_report(self, data, name):
        output = self.root / (name + ".pdf")
        before = copy.deepcopy(data)
        pdf_report.write_pdf(data, output)
        self.assertTrue(output.read_bytes().startswith(b"%PDF-"))
        self.assertEqual(data, before)
        self.assertFalse(list(self.root.glob(".can-performance-pdf-*")))
        return output

    def test_real_template_compiles_with_score_assessment(self):
        # This test runs even where optional PDF text extraction is unavailable.
        self.compile_report(graded_evidence(), "graded")

    @unittest.skipUnless(PdfReader, "pypdf is unavailable; use the bundled Python to check extracted text")
    def test_pdf_shows_grade_range_ungraded_and_advisory_rubric(self):
        output = self.compile_report(graded_evidence(), "graded-text")
        text = " ".join(" ".join(page.extract_text() or "" for page in PdfReader(output).pages).split())
        self.assertIn("performance report card", text)
        self.assertIn("A-", text)
        self.assertIn("B+ to A+", text)
        self.assertIn("Browser · U", text)
        self.assertRegex(text, r"not (?:a )?TypeSafe-certified")
        self.assertRegex(text, r"wording sensitivity, not (?:a )?confidence interval")
        self.assertIn("Callback timing does not measure visual responsiveness", text)

    @unittest.skipUnless(PdfReader, "pypdf is unavailable; use the bundled Python to check extracted text")
    def test_pdf_smoke_cannot_display_attached_grades(self):
        data = graded_evidence()
        data["manifest"]["quality"] = "smoke"
        output = self.compile_report(data, "smoke")
        text = " ".join(" ".join(page.extract_text() or "" for page in PdfReader(output).pages).split())
        self.assertNotIn("performance report card", text)
        self.assertNotIn("How the grades were produced", text)
        self.assertIn("SMOKE", text)


if __name__ == "__main__":
    unittest.main()
