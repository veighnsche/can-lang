"""Pure Jev request construction and response validation; no service calls."""

import copy
import math
import unittest

from assessment import attach_assessment
import jev_grading
from ranking import SUITE_UNITS


def case(suite, median=2_000_000, *, parameters=None, scope="one bounded workload"):
    return {"unit": SUITE_UNITS[suite], "parameters": parameters or {"size": 10},
            "timing_scope": scope, "iterations_per_sample": 2, "ranking_issues": [],
            "distribution": {"median": median, "min": median, "max": median,
                             "median_absolute_deviation": 0, "n": 3}}


def evidence():
    rows = {
        "compiler/pipeline": case("compiler"), "compiler/check": case("compiler", 1_000_000),
        "generated/sum.can": case("generated"), "generated/sum.native": case("generated", 1_000_000),
        "browser/callback": case("browser", 0),
        "server/local.bounded-client": case("server", 12), "server/local.arrival-rate": case("server", 50),
        "editor/project.completion": case("editor"), "editor/project.diagnostics": case("editor"),
        "io/runtime.read": case("io", 4), "io/native-contract.read": case("io", 2),
        "runtime/collection.can": case("runtime"), "runtime/collection.native": case("runtime"),
        "codecs/json.decode": case("codecs"), "codecs/json.native": case("codecs"),
    }
    suites = list(dict.fromkeys(key.split("/")[0] for key in rows))
    return {"kind": "can.performance-report", "schema_version": 1,
            "manifest": {"status": "complete", "quality": "measurement", "source": {"revision": "fixture"},
                         "requested_suites": suites, "completed_suites": suites},
            "cases": rows, "rankings": {}}


def score_answer(criteria, score=4, confidence=0.8):
    lower, upper = math.floor(score), math.ceil(score)
    probabilities = {str(index): 0 for index in range(6)}
    probabilities[str(lower)] = 1 - (score - lower)
    if upper != lower:
        probabilities[str(upper)] = score - lower
    return {"type": "score", "score": score, "confidence": confidence,
            "legend": {str(index): text for index, text in enumerate(criteria)}, "probabilities": probabilities}


def responses(requests):
    return [{"model": "jev-test", "answers": {
        suite: score_answer(question["criteria"])
        for suite, question in request["questions"].items()}} for request in requests]


class RequestBuilderTests(unittest.TestCase):
    def test_three_complete_prose_variations_preserve_exact_measurement_facts(self):
        data = evidence()
        before = copy.deepcopy(data)
        requests = jev_grading.build_requests(data)
        self.assertEqual(len(requests), 3)
        self.assertEqual(len({request["state"]["context"] for request in requests}), 3)
        selected = set(requests[0]["questions"])
        self.assertNotIn("browser", selected)
        for suite in selected:
            self.assertEqual(len({request["questions"][suite]["instructions"] for request in requests}), 3)
            self.assertEqual(len({request["state"]["slices"][suite]["scope"] for request in requests}), 3)
            for index in range(6):
                self.assertEqual(len({request["questions"][suite]["criteria"][index] for request in requests}), 3)
            expected = requests[0]["state"]["slices"][suite]
            for request in requests:
                self.assertEqual(set(request["questions"]), selected)
                self.assertEqual(request["questions"][suite]["type"], "score")
                self.assertEqual(request["state"]["slices"][suite]["cases"], expected["cases"])
                self.assertEqual(request["state"]["slices"][suite]["matched_pairs"], expected["matched_pairs"])
        self.assertEqual(data, before)

    def test_nonadditive_phase_clocks_and_scoped_qualifications(self):
        for request in jev_grading.build_requests(evidence()):
            context = request["state"]["context"].lower()
            compiler = request["state"]["slices"]["compiler"]["scope"].lower()
            self.assertIn("overlap", compiler)
            self.assertIn("pipeline", compiler)
            self.assertTrue(any(word in compiler for word in ("offset", "compensate", "mask")))
            self.assertTrue("average" in context or "combin" in context or "summing" in context)
            self.assertTrue("advi" in context or "opinion" in context)
            self.assertEqual(set(request["state"]["slices"]["compiler"]["cases"]),
                             {"compiler/pipeline", "compiler/check"})
            self.assertEqual(set(request["state"]["slices"]["editor"]["cases"]), {"editor/project.completion"})
            self.assertEqual(set(request["state"]["slices"]["server"]["cases"]), {"server/local.bounded-client"})
            self.assertEqual(set(request["state"]["slices"]["runtime"]["cases"]), {"runtime/collection.can"})
            self.assertEqual(set(request["state"]["slices"]["codecs"]["cases"]), {"codecs/json.decode"})

    def test_native_pairs_require_identical_declared_contracts(self):
        data = evidence()
        rows = data["cases"]
        pairs = jev_grading.pairs(rows)
        self.assertEqual({row["subject"] for row in pairs}, {"generated/sum.can", "io/runtime.read"})
        generated = next(row for row in pairs if row["subject"].startswith("generated/"))
        self.assertEqual(generated["time_ratio"], 2)
        self.assertEqual(generated["extra_ms"], 1)
        for field, changed in (("unit", "ms/op"), ("parameters", {"size": 11}),
                               ("timing_scope", "a different operation"), ("iterations_per_sample", 3)):
            with self.subTest(field=field):
                unmatched = copy.deepcopy(rows)
                unmatched["generated/sum.native"][field] = changed
                self.assertFalse(any(row["subject"] == "generated/sum.can" for row in jev_grading.pairs(unmatched)))
        invalid = copy.deepcopy(rows)
        invalid["generated/sum.native"]["distribution"]["median"] = 0
        self.assertFalse(any(row["subject"] == "generated/sum.can" for row in jev_grading.pairs(invalid)))

    def test_browser_callback_never_becomes_a_visual_response_grade(self):
        data = evidence()
        data["cases"]["browser/callback"] = case("browser", 0.05)
        requests = jev_grading.build_requests(data)
        self.assertTrue(all("browser" not in request["questions"] for request in requests))
        result = attach_assessment(data, jev_grading.make_payload(data, requests, responses(requests)))
        row = result["assessment"]["slices"]["browser"]
        self.assertEqual(row["grade"], "U")
        self.assertEqual(row["answers"], [])
        self.assertTrue(row["ungraded_reason"])

    def test_invalid_evidence_rejected_before_request_construction(self):
        for changes in ({"quality": "smoke"}, {"quality": "exploratory"}, {"status": "failed"},
                        {"completed_suites": ["compiler"]}):
            data = evidence()
            data["manifest"].update(changes)
            with self.subTest(changes=changes), self.assertRaises(ValueError):
                jev_grading.build_requests(data)


class PayloadTests(unittest.TestCase):
    def test_payload_attaches_advisory_answers_without_mutating_evidence(self):
        data = evidence()
        before = copy.deepcopy(data)
        requests = jev_grading.build_requests(data)
        replies = responses(requests)
        saved = jev_grading.make_payload(data, requests, replies)
        result = attach_assessment(data, saved)
        self.assertEqual(result["assessment"]["slices"]["generated"]["grade"], "A")
        self.assertEqual(result["assessment"]["slices"]["browser"]["grade"], "U")
        self.assertIn("server/local.arrival-rate", saved["slices"]["server"]["excluded_cases"])
        self.assertIn("editor/project.diagnostics", saved["slices"]["editor"]["excluded_cases"])
        self.assertEqual(saved["models"], ["jev-test"] * 3)
        self.assertEqual(data, before)

    def test_wrong_question_legend_model_and_invalid_answer_rejected(self):
        data = evidence()
        requests = jev_grading.build_requests(data)
        changes = [
            lambda replies: replies[0]["answers"].pop("compiler"),
            lambda replies: replies[0]["answers"].update(extra=score_answer(jev_grading.RUBRICS[0])),
            lambda replies: replies[0]["answers"]["compiler"]["legend"].update({"0": "other rubric"}),
            lambda replies: replies[0].update(model=""),
            lambda replies: replies[0].update(model=None),
            lambda replies: replies[0]["answers"]["compiler"].update(score=float("nan")),
            lambda replies: replies[0]["answers"]["compiler"]["probabilities"].update({"4": 0.1}),
            lambda replies: replies[0].update(answers=[]),
            lambda replies: replies[0]["answers"].update(compiler=None),
        ]
        for index, change in enumerate(changes):
            replies = responses(requests)
            change(replies)
            with self.subTest(index=index), self.assertRaises(ValueError):
                jev_grading.make_payload(data, requests, replies)
        with self.assertRaises(ValueError):
            jev_grading.make_payload(data, requests[:2], responses(requests))


if __name__ == "__main__":
    unittest.main()
