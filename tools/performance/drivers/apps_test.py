"""Fast driver contract checks; no benchmark load, compiler or browser launch."""
import json
import importlib.util
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

spec = importlib.util.spec_from_file_location('apps_driver', Path(__file__).with_name('apps.py'))
driver = importlib.util.module_from_spec(spec)
spec.loader.exec_module(driver)

class DriverContract(unittest.TestCase):
    def run_driver(self, manifest, suite, env=None, profile="quick", extra=()):
        with tempfile.TemporaryDirectory() as temp:
            work = Path(temp)
            (work / 'apps-manifest.json').write_text(json.dumps(manifest))
            output = work / 'output.json'
            command = [sys.executable, str(Path(__file__).with_name('apps.py')), 'run', '--repo', str(Path(__file__).resolve().parents[3]), '--work-dir', temp, '--output', str(output), '--suite', suite, '--profile', profile, *extra]
            proc = subprocess.run(command, env={**os.environ, **(env or {})}, capture_output=True, text=True, timeout=10)
            return proc.returncode, json.loads(output.read_text())

    def test_requested_missing_engine_blocks(self):
        code, output = self.run_driver({'suites': ['browser'], 'browser': {'available': ['chromium']}}, 'browser', {'CAN_PERF_BROWSER_ENGINES': 'firefox'})
        self.assertEqual(code, 2)
        self.assertEqual(output['status'], 'blocked')
        self.assertIn('firefox', output['reason'])
        self.assertEqual(output['cases'], [])

    def test_unprepared_suite_blocks(self):
        code, output = self.run_driver({'suites': ['io']}, 'journeys')
        self.assertEqual(code, 2)
        self.assertEqual(output['status'], 'blocked')
        self.assertIn('not prepared', output['reason'])

    def test_io_sampling_batches_preserve_operations(self):
        for profile, count in [('quick', 3), ('standard', 7)]:
            with self.subTest(profile=profile):
                code, output = self.run_driver({'suites': ['io']}, 'io', profile=profile, extra=['--iterations', '2', '--warmups', '1', '--size', '8'])
                self.assertEqual(code, 0, output)
                self.assertEqual(len(output['cases']), 6)
                self.assertEqual({case['name'] for case in output['cases']}, {f'{impl}.{scenario}' for impl in ['runtime', 'native-contract'] for scenario in ['bounded-stream-utf8', 'binary-write-read', 'bounded-read-rejection']})
                for case in output['cases']:
                    self.assertEqual(len(case['samples']), count)
                    self.assertEqual(len(case['warmup_samples']), 1)
                    self.assertEqual(case['iterations_per_sample'], 2)
                    raw = case['metrics']['operation_batches_ms']
                    self.assertEqual(len(raw), count)
                    for sample, operations in zip(case['samples'], raw):
                        self.assertEqual(len(operations), 2)
                        self.assertAlmostEqual(sample, sum(operations) / 2)

    def test_large_child_file_retains_raw_observations(self):
        with tempfile.TemporaryDirectory() as temp:
            payload = [{'id': i, 'value': 'x' * 128} for i in range(1500)]
            code = 'import json,sys; from pathlib import Path; data={"schema_version":1,"suite":"browser","status":"complete","cases":[{"metrics":{"raw": [{"id":i,"value":"x"*128} for i in range(1500)]}}],"artifacts":[]}; Path(sys.argv[1]).write_text(json.dumps(data))'
            result = driver.child_result([sys.executable, '-c', code], Path(temp), 'browser')
            self.assertEqual(result['cases'][0]['metrics']['raw'], payload)
            path = Path(result['artifacts'][-1])
            self.assertGreater(path.stat().st_size, 65536)
            again = driver.child_result([sys.executable, '-c', code], Path(temp), 'browser')
            self.assertNotEqual(path, Path(again['artifacts'][-1]))

    def test_failed_child_file_is_never_consumed(self):
        with tempfile.TemporaryDirectory() as temp:
            code = 'import json,sys; from pathlib import Path; Path(sys.argv[1]).write_text(json.dumps({"schema_version":1,"suite":"browser","status":"complete","cases":[]})); sys.exit(1)'
            with self.assertRaises(RuntimeError):
                driver.child_result([sys.executable, '-c', code], Path(temp), 'browser')

    def test_malformed_child_file_fails(self):
        with tempfile.TemporaryDirectory() as temp:
            code = 'import sys; from pathlib import Path; Path(sys.argv[1]).write_text("{truncated")'
            with self.assertRaises(json.JSONDecodeError):
                driver.child_result([sys.executable, '-c', code], Path(temp), 'browser')

if __name__ == '__main__':
    unittest.main()
