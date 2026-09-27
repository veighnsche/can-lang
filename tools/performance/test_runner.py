"""Regression checks for evidence admission, serial orchestration and cleanup."""

import argparse
import contextlib
import copy
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

import isolation
import perf
import schema


def evidence(suite="runtime"):
    return {"schema_version": 1, "suite": suite, "status": "complete", "cases": [{
        "name": "anchor", "unit": "ns/op", "samples": [2, 4, 6],
        "warmup_samples": [999999], "iterations_per_sample": 1,
        "timing_scope": "one operation", "parameters": {"size": 10},
        "correctness": {"passed": True, "checks": ["exact expected output"]},
    }]}


class EvidenceTests(unittest.TestCase):
    def test_malformed_failed_driver_output_keeps_a_bounded_reason(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "result.json"
            for payload in ("[]", "null", "not JSON", '{"reason": []}'):
                path.write_text(payload)
                self.assertEqual(perf.failure_reason(path, "driver failed"), "driver failed")

    def test_rejects_incorrect_empty_nonfinite_and_duplicate_evidence(self):
        mutations = [
            lambda r: r.update(status="blocked"),
            lambda r: r.update(cases=[]),
            lambda r: r.update(cases=[None]),
            lambda r: r["cases"][0].update(samples=[float("nan")]),
            lambda r: r["cases"][0].update(samples=[True]),
            lambda r: r["cases"][0].update(correctness={"passed": False, "checks": ["x"]}),
            lambda r: r["cases"][0].update(correctness=None),
            lambda r: r["cases"].append(copy.deepcopy(r["cases"][0])),
        ]
        for mutate in mutations:
            result = evidence()
            mutate(result)
            with self.subTest(result=result), self.assertRaises(ValueError):
                schema.validate_result(result, "runtime")

    def test_uses_independent_trial_medians_and_excludes_warmups(self):
        a, b = evidence(), evidence()
        b["cases"][0]["samples"] = [10] * 9
        result = schema.summarize_trials([
            {"suite": "runtime", "result": a}, {"suite": "runtime", "result": b},
        ])["runtime/anchor"]
        self.assertEqual(result["trial_medians"], [4, 10])
        self.assertEqual(result["distribution"]["median"], 7)
        self.assertEqual(result["distribution"]["n"], 2)
        self.assertNotIn("p99", result["distribution"])

    def test_rejects_case_or_contract_drift_between_trials(self):
        for field, value in [("name", "changed"), ("parameters", {"size": 100}), ("unit", "ms/op")]:
            changed = evidence()
            changed["cases"][0][field] = value
            with self.subTest(field=field), self.assertRaises(ValueError):
                schema.summarize_trials([
                    {"suite": "runtime", "result": evidence()},
                    {"suite": "runtime", "result": changed},
                ])

    def test_comparison_rejects_smoke_and_changed_instruments(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            a, b = root / "a", root / "b"
            metadata = {
                "status": "complete", "quality": "measurement", "requested_suites": ["runtime", "codecs"],
                "completed_suites": ["runtime", "codecs"], "environment": {}, "profile": "quick",
                "iterations": 1, "warmups": 1, "size": 10, "isolation": {},
                "trials": 1,
                "source": {"dependencies": "one", "harness_overlays": "same"},
            }
            summary = schema.summarize_trials([{"suite": suite, "result": evidence(suite)} for suite in metadata["requested_suites"]])
            for target in (a, b):
                perf.write_json(target / "manifest.json", metadata)
                perf.write_json(target / "summary.json", summary)
                perf.write_json(target / "raw/runtime-001.json", evidence())
                perf.write_json(target / "raw/codecs-001.json", evidence("codecs"))
            args = argparse.Namespace(baseline=str(a), candidate=str(b), output=str(root / "comparison.json"))
            self.assertEqual(perf.compare(args), 0)
            for mutate in [lambda x: x.update(quality="smoke"), lambda x: x["source"].update(harness_overlays="new"), lambda x: x.update(requested_suites=["codecs", "runtime"])]:
                changed = copy.deepcopy(metadata)
                mutate(changed)
                perf.write_json(b / "manifest.json", changed)
                with self.assertRaises(ValueError):
                    perf.compare(args)

    def test_comparison_requires_raw_evidence_and_recomputed_summary(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            metadata = {"requested_suites": ["runtime"], "trials": 1, "profile": "quick", "warmups": 1, "iterations": 1}
            perf.write_json(root / "summary.json", {})
            with self.assertRaisesRegex(ValueError, "missing raw"):
                perf.verified_summary(root, metadata)
            perf.write_json(root / "raw/runtime-001.json", evidence())
            with self.assertRaisesRegex(ValueError, "does not match"):
                perf.verified_summary(root, metadata)


class IsolationTests(unittest.TestCase):
    def test_quiet_gate_resets_after_busy_observation(self):
        clock = [0.0]
        readings = iter([100, 100, 20, 100, 100, 100])
        def idle():
            clock[0] += 1
            return next(readings)
        def sleep(seconds):
            clock[0] += seconds
        with tempfile.TemporaryDirectory() as directory, patch.object(isolation.time, "monotonic", side_effect=lambda: clock[0]), patch.object(isolation.time, "sleep", side_effect=sleep), patch.object(isolation, "competing", return_value=[]), patch.object(isolation, "idle_percent", side_effect=idle):
            path = Path(directory) / "events"
            isolation.quiet_window(3, 90, 30, path)
            events = [json.loads(line) for line in path.read_text().splitlines()]
            samples = [e for e in events if e["event"] == "quiet-sample"]
            self.assertEqual(samples[2]["continuous_seconds"], 0)
            self.assertEqual(samples[-1]["continuous_seconds"], 4)
            self.assertEqual(events[-1]["event"], "quiet-ready")

    def test_process_inspection_failure_does_not_accept_quiet_host(self):
        with patch.object(isolation.subprocess, "check_output", return_value=""):
            with self.assertRaisesRegex(RuntimeError, "inspection is unavailable"):
                isolation.competing()

    def test_ancestry_does_not_hide_sibling_compilers(self):
        rows = isolation.parse_processes("1 0 init\n10 1 agent\n11 10 runner\n12 11 driver\n13 12 browser\n14 10 go build ./...")
        ignored = isolation.related(rows, 11, 12)
        self.assertEqual(ignored, {1, 10, 11, 12, 13})
        self.assertEqual(isolation.kind(rows[14][1]), "test/build")
        self.assertIsNone(isolation.kind("python3 /tmp/tools/performance/perf.py run"))
        self.assertIsNone(isolation.kind("canlc lsp"))

    def test_lock_contention_times_out_without_entering(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            with patch.object(isolation, "LOCK", root / "lock"):
                with isolation.exclusive(1, root / "events.jsonl"):
                    with self.assertRaises(TimeoutError):
                        with isolation.exclusive(0.02, root / "events.jsonl"):
                            self.fail("second lock holder entered")

    def test_timeout_terminates_driver_process(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            command = [sys.executable, "-c", "import os,pathlib,time;pathlib.Path('pid').write_text(str(os.getpid()));time.sleep(30)"]
            with self.assertRaises(TimeoutError):
                isolation.execute(command, root, os.environ.copy(), root / "out", root / "err", .3, root / "events", False)
            pid = int((root / "pid").read_text())
            with self.assertRaises(ProcessLookupError):
                os.kill(pid, 0)


class SerialRunTests(unittest.TestCase):
    def run_fixture(self, root, fail=False):
        source = root / "source"
        source.mkdir()
        driver = source / "fixture.py"
        driver.write_text('''import json,pathlib,sys,time
mode,suite,out,fail=sys.argv[1:]
root=pathlib.Path(__file__).parent
active=root/'active'
with active.open('x') as h: h.write(suite)
with (root/'order').open('a') as h: h.write('start '+mode+' '+suite+'\\n')
time.sleep(.03)
case={"name":"anchor","unit":"ns/op","samples":[1,1,1],"iterations_per_sample":1,"timing_scope":"fixture","parameters":{},"correctness":{"passed":True,"checks":["fixture"]}}
result={"schema_version":1,"suite":"prepare" if mode=="prepare" else suite,"status":"complete","cases":[] if mode=="prepare" else [case]}
if fail=="yes" and mode=="run" and suite=="second": result.update(status="failed",reason="deliberate fixture failure")
pathlib.Path(out).write_text(json.dumps(result))
with (root/'order').open('a') as h: h.write('end '+mode+' '+suite+'\\n')
active.unlink()
sys.exit(1 if result['status']=='failed' else 0)
''')
        args = argparse.Namespace(suite=None, profile="quick", smoke=True, trials=None, iterations=None,
            warmups=None, size=None, output=str(root / "output"), quiet_seconds=30, idle_percent=90,
            wait_seconds=1, prepare_timeout=5, timeout=5, revision=None, working_tree=False)
        def command(snapshot, family, mode, work, out, options, *, suite=None, suites=None):
            return [sys.executable, str(driver), mode, suite or family, str(out), "yes" if fail else "no"]
        with contextlib.ExitStack() as stack:
            stack.enter_context(patch.object(perf, "SUITES", {"first": ("one", "fixture"), "second": ("two", "fixture")}))
            stack.enter_context(patch.object(perf, "source_snapshot", return_value=(source, {"dependencies": None})))
            stack.enter_context(patch.object(perf, "environment", return_value={}))
            stack.enter_context(patch.object(perf, "command_for", side_effect=command))
            stack.enter_context(patch.object(isolation, "LOCK", root / "lock"))
            result = perf.run(args)
        return result, (source / "order").read_text().splitlines(), root / "output"

    def test_all_suites_run_serially_and_combined_report_is_complete(self):
        with tempfile.TemporaryDirectory() as directory:
            status, order, output = self.run_fixture(Path(directory))
            self.assertEqual(status, 0)
            self.assertEqual(order, [f"{event} {mode} {name}" for mode, name in [("prepare", "one"), ("prepare", "two"), ("run", "first"), ("run", "second")] for event in ("start", "end")])
            manifest = json.loads((output / "manifest.json").read_text())
            self.assertEqual(manifest["completed_suites"], ["first", "second"])
            self.assertEqual(manifest["quality"], "smoke")
            self.assertEqual(len(json.loads((output / "summary.json").read_text())), 2)

    def test_failed_suite_retains_raw_evidence_but_no_completed_summary(self):
        with tempfile.TemporaryDirectory() as directory:
            status, order, output = self.run_fixture(Path(directory), fail=True)
            self.assertEqual(status, 1)
            self.assertFalse((output / "summary.json").exists())
            self.assertTrue((output / "raw/first-001.json").exists())
            manifest = json.loads((output / "manifest.json").read_text())
            self.assertEqual(manifest["completed_suites"], ["first"])
            self.assertEqual(manifest["status"], "failed")


if __name__ == "__main__":
    unittest.main()
