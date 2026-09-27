"""Verify that doing less server work cannot improve a performance ranking."""
import copy
import unittest

from schema import summarize_trials


def server_result(dropped=0):
    return {"schema_version": 1, "suite": "server", "status": "complete", "cases": [{
        "name": "scheduled", "unit": "ms/trial", "samples": [3, 3, 3],
        "iterations_per_sample": 1, "timing_scope": "all scheduled work",
        "parameters": {"request_count": 2},
        "correctness": {"passed": True, "checks": ["scheduled requests accounted for"]},
        "metrics": {"trials": [{"warmup": False, "scheduled_count": 2,
                               "completed": 2 - dropped, "dropped_by_generator": dropped, "errors": 0,
                               "requests": [{"id": i, "correct": True} for i in range(2 - dropped)]}
                              for _ in range(3)]},
    }]}


def issues(result):
    return summarize_trials([{"suite": "server", "result": result}])["server/scheduled"]["ranking_issues"]


class DeliveryRankingTests(unittest.TestCase):
    def test_full_delivery_eligible_but_generator_drops_excluded(self):
        self.assertEqual(issues(server_result()), [])
        self.assertIn("dropped 3", issues(server_result(1))[0])

    def test_missing_malformed_or_unaccounted_delivery_is_excluded(self):
        mutations = [
            lambda case: case.pop("metrics"),
            lambda case: case["metrics"].update(trials=[]),
            lambda case: case["metrics"]["trials"].pop(),
            lambda case: case["metrics"]["trials"][0].update(completed=True),
            lambda case: case["metrics"]["trials"][0].update(scheduled_count=3),
            lambda case: case["metrics"]["trials"][0].update(errors=1),
            lambda case: case["metrics"]["trials"][0]["requests"][0].update(correct=False),
            lambda case: case["metrics"]["trials"][0]["requests"][1].update(id=0),
            lambda case: case["metrics"]["trials"][0]["requests"][1].update(id=8),
            lambda case: case["metrics"]["trials"][0].pop("warmup"),
        ]
        for mutation in mutations:
            result = server_result()
            mutation(result["cases"][0])
            with self.subTest(mutation=mutation):
                self.assertTrue(issues(result))

    def test_warmup_drops_do_not_contaminate_measured_delivery(self):
        result = server_result()
        warmup = copy.deepcopy(server_result(1)["cases"][0]["metrics"]["trials"][0])
        warmup["warmup"] = True
        result["cases"][0]["metrics"]["trials"].insert(0, warmup)
        self.assertEqual(issues(result), [])

    def test_one_incomplete_process_trial_excludes_case_across_trials(self):
        summary = summarize_trials([{"suite": "server", "result": server_result()},
                                    {"suite": "server", "result": server_result(1)}])
        self.assertTrue(summary["server/scheduled"]["ranking_issues"])
        self.assertEqual(summary["server/scheduled"]["distribution"]["n"], 2)


if __name__ == "__main__":
    unittest.main()
