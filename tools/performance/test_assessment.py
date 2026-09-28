"""Bounded validation of advisory evidence binding and grade composition."""

import copy
import hashlib
import json
import math
import unittest

import assessment


def answer(score, confidence=0.8):
    lower, upper = math.floor(score), math.ceil(score)
    probabilities = {str(index): 0 for index in range(6)}
    probabilities[str(lower)] = 1 - (score - lower)
    if upper != lower:
        probabilities[str(upper)] = score - lower
    return {"type": "score", "score": score, "confidence": confidence,
            "legend": {str(index): "Level " + str(index) for index in range(6)},
            "probabilities": probabilities}


def payload(data):
    return {"kind": "can.performance-assessment", "schema_version": 1,
            "rubric_version": assessment.RUBRIC_VERSION,
            "evidence_fingerprint": assessment.evidence_fingerprint(data),
            "rubric": ["Descriptor " + str(index) for index in range(6)],
            "models": ["jev"] * 3,
            "slices": {
                "runtime": {"answers": [answer(4), answer(3), answer(5, 0.7)],
                            "basis": "A synthetic operation", "excluded_cases": []},
                "browser": {"answers": [], "basis": "Unresolved interaction timing",
                            "excluded_cases": ["browser/unresolved"],
                            "ungraded_reason": "Every timing is below measurement resolution"}}}


def report_data():
    return {"kind": "can.performance-report", "schema_version": 1,
            "manifest": {"status": "complete", "quality": "measurement",
                         "requested_suites": ["runtime", "browser"],
                         "completed_suites": ["runtime", "browser"],
                         "source": {"revision": "synthetic"}},
            "cases": {
                "runtime/tiny": {"unit": "ns/op", "distribution": {
                    "median": 2, "min": 1, "max": 3, "median_absolute_deviation": 1, "n": 3}},
                "browser/unresolved": {"unit": "ms/interaction", "distribution": {
                    "median": 0, "min": 0, "max": 0, "median_absolute_deviation": 0, "n": 3}}},
            "rankings": {}}


class EvidenceValidationTests(unittest.TestCase):
    def test_canonical_fingerprint_ignores_presentation_and_binds_source_and_cases(self):
        data = report_data()
        expected = hashlib.sha256(json.dumps(
            {"source": data["manifest"]["source"], "cases": data["cases"]},
            sort_keys=True, separators=(",", ":"), allow_nan=False).encode()).hexdigest()
        self.assertEqual(assessment.evidence_fingerprint(data), expected)
        display = copy.deepcopy(data)
        display["rankings"] = {"extra": "presentation"}
        display["manifest"]["started_at"] = 99
        display["cases"] = dict(reversed(list(display["cases"].items())))
        self.assertEqual(assessment.evidence_fingerprint(display), expected)
        display["manifest"]["source"]["revision"] = "changed"
        self.assertNotEqual(assessment.evidence_fingerprint(display), expected)
        display = copy.deepcopy(data)
        display["cases"]["runtime/tiny"]["distribution"]["median"] = 2.5
        self.assertNotEqual(assessment.evidence_fingerprint(display), expected)

    def test_nonfinite_fingerprint_rejected(self):
        data = report_data()
        data["cases"]["runtime/tiny"]["distribution"]["median"] = float("nan")
        with self.assertRaisesRegex(ValueError, "fingerprint"):
            assessment.evidence_fingerprint(data)

    def test_complete_measurement_inventory(self):
        self.assertEqual(set(assessment._measured_slices(report_data())), {"runtime", "browser"})
        changes = [
            lambda data: data["manifest"].update(status="failed"),
            lambda data: data["manifest"].update(quality="smoke"),
            lambda data: data["manifest"].update(requested_suites=["runtime", "runtime"]),
            lambda data: data["manifest"].update(completed_suites=["runtime"]),
            lambda data: data["manifest"].update(source=None),
            lambda data: data["manifest"].update(requested_suites=["runtime", "unknown"]),
            lambda data: data["cases"].pop("runtime/tiny"),
            lambda data: data["cases"].update({"unknown/tiny": data["cases"]["runtime/tiny"]}),
            lambda data: data["cases"].update({"runtime/": data["cases"]["runtime/tiny"]}),
            lambda data: data["cases"]["runtime/tiny"].update(unit="ms/op"),
        ]
        for index, change in enumerate(changes):
            with self.subTest(index=index):
                data = report_data()
                change(data)
                with self.assertRaises(ValueError):
                    assessment._measured_slices(data)


class ScoreValidationTests(unittest.TestCase):
    def test_every_named_grade_and_midpoint_tie(self):
        positions = assessment.GRADE_POSITIONS
        for grade, score in positions:
            with self.subTest(grade=grade):
                self.assertEqual(assessment.letter_grade(score), grade)
        for (lower_grade, lower), (upper_grade, upper) in zip(positions, positions[1:]):
            midpoint = (lower + upper) / 2
            with self.subTest(midpoint=midpoint):
                self.assertEqual(assessment.letter_grade(midpoint), lower_grade)
                self.assertEqual(assessment.letter_grade(midpoint - 1e-8), lower_grade)
                self.assertEqual(assessment.letter_grade(midpoint + 1e-8), upper_grade)
                self.assertEqual(assessment.letter_grade(math.nextafter(midpoint, 0)), lower_grade)
                self.assertEqual(assessment.letter_grade(math.nextafter(midpoint, 5)), upper_grade)

    def test_invalid_scores_and_confidence(self):
        for invalid in (None, True, "4", -0.01, 5.01, float("inf"), float("nan"), 10 ** 1000):
            with self.subTest(invalid=str(invalid)[:20]):
                with self.assertRaises(ValueError):
                    assessment.letter_grade(invalid)
                value = answer(3)
                value["score"] = invalid
                with self.assertRaises(ValueError):
                    assessment.validate_score_answer(value)
        for invalid in (None, False, "0.8", -0.01, 1.01, float("nan"), float("inf")):
            value = answer(3)
            value["confidence"] = invalid
            with self.subTest(confidence=invalid), self.assertRaises(ValueError):
                assessment.validate_score_answer(value)

    def test_complete_answer_and_rounded_probabilities(self):
        value = answer(3.7)
        self.assertIs(assessment.validate_score_answer(value), value)
        rounded = answer(3.02)
        rounded["probabilities"] = {"0": 0, "1": 0, "2": 0.33, "3": 0.33, "4": 0.34, "5": 0}
        self.assertIs(assessment.validate_score_answer(rounded), rounded)
        rounded["score"] = 3.07
        assessment.validate_score_answer(rounded)
        rounded["score"] = 3.071
        with self.assertRaisesRegex(ValueError, "probability-weighted"):
            assessment.validate_score_answer(rounded)

    def test_malformed_answers(self):
        changes = [
            lambda value: value.update(type="choice"),
            lambda value: value.update(legend=[]),
            lambda value: value["legend"].pop("5"),
            lambda value: value["legend"].update({"6": "extra"}),
            lambda value: value["legend"].update({"1": " "}),
            lambda value: value.update(probabilities=[]),
            lambda value: value["probabilities"].pop("5"),
            lambda value: value["probabilities"].update({"6": 0}),
            lambda value: value["probabilities"].update({"0": True}),
            lambda value: value["probabilities"].update({"0": float("nan")}),
            lambda value: value["probabilities"].update({"0": -0.01}),
            lambda value: value["probabilities"].update({"3": 1.01}),
            lambda value: value["probabilities"].update({"3": 0.97}),
            lambda value: value.update(score=4),
        ]
        for index, change in enumerate(changes):
            value = answer(3)
            change(value)
            with self.subTest(index=index), self.assertRaises(ValueError):
                assessment.validate_score_answer(value)


class AssessmentCompositionTests(unittest.TestCase):
    def test_median_range_confidence_and_immutability(self):
        data = report_data()
        saved = payload(data)
        saved["slices"]["runtime"].update(grade="F", score_median=0, disagreement=False)
        data_before, saved_before = copy.deepcopy(data), copy.deepcopy(saved)
        result = assessment.attach_assessment(data, saved)
        row = result["assessment"]["slices"]["runtime"]
        self.assertEqual(row["grade"], "A")
        self.assertEqual(row["score_median"], 4)
        self.assertEqual((row["score_min"], row["score_max"]), (3, 5))
        self.assertEqual((row["grade_min"], row["grade_max"]), ("B", "A+"))
        self.assertEqual(row["confidence_min"], 0.7)
        self.assertTrue(row["disagreement"])
        self.assertEqual(row["answers"], saved_before["slices"]["runtime"]["answers"])
        ungraded = result["assessment"]["slices"]["browser"]
        self.assertEqual(ungraded["grade"], "U")
        self.assertIsNone(ungraded["score_median"])
        self.assertIsNone(ungraded["confidence_min"])
        self.assertFalse(ungraded["disagreement"])
        result["cases"]["runtime/tiny"]["distribution"]["median"] = 100
        result["assessment"]["slices"]["runtime"]["answers"][0]["score"] = 0
        self.assertEqual(data, data_before)
        self.assertEqual(saved, saved_before)

    def test_exact_agreement_and_within_letter_disagreement(self):
        data = report_data()
        for scores, disagreement in (([3.7] * 3, False), ([3.7, 3.71, 3.72], True)):
            saved = payload(data)
            saved["slices"]["runtime"]["answers"] = [answer(score) for score in scores]
            row = assessment.attach_assessment(data, saved)["assessment"]["slices"]["runtime"]
            self.assertEqual(row["grade"], "A-")
            self.assertEqual(row["disagreement"], disagreement)

    def test_malformed_assessments(self):
        data = report_data()
        changes = [
            lambda saved: saved.update(kind="unknown"),
            lambda saved: saved.update(schema_version=True),
            lambda saved: saved.update(rubric_version="old"),
            lambda saved: saved.update(evidence_fingerprint="stale"),
            lambda saved: saved.update(rubric=["short"]),
            lambda saved: saved["rubric"].__setitem__(0, ""),
            lambda saved: saved["slices"].pop("runtime"),
            lambda saved: saved["slices"].update(extra={}),
            lambda saved: saved["slices"]["runtime"].update(basis=None),
            lambda saved: saved["slices"]["runtime"].update(answers=[answer(3)]),
            lambda saved: saved["slices"]["runtime"].update(excluded_cases=["browser/unresolved"]),
            lambda saved: saved["slices"]["runtime"].update(excluded_cases=["runtime/tiny"]),
            lambda saved: saved["slices"]["browser"].update(excluded_cases=[]),
            lambda saved: saved["slices"]["browser"].update(excluded_cases=["browser/unresolved"] * 2),
            lambda saved: saved["slices"]["browser"].update(answers=[answer(5)] * 3),
            lambda saved: saved["slices"]["browser"].pop("ungraded_reason"),
        ]
        for index, change in enumerate(changes):
            saved = payload(data)
            change(saved)
            with self.subTest(index=index), self.assertRaises(ValueError):
                assessment.attach_assessment(data, saved)

    def test_distribution_must_be_finite_nonnegative_and_ordered(self):
        for field, invalid in (("median", float("inf")), ("min", -1), ("max", "3"),
                               ("median_absolute_deviation", True), ("n", False),
                               ("n", 0), ("n", 1.5), ("median", 4), ("min", None)):
            with self.subTest(field=field, invalid=invalid):
                data = report_data()
                data["cases"]["runtime/tiny"]["distribution"][field] = invalid
                with self.assertRaises(ValueError):
                    assessment._measured_slices(data)


if __name__ == "__main__":
    unittest.main()
