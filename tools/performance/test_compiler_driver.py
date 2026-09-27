"""Lightweight compiler driver contract tests; heavy smoke runs use parent flock."""
import importlib.util
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import MagicMock, patch

DRIVER = Path(__file__).parent / 'drivers/compiler.py'


class CompilerDriverContract(unittest.TestCase):
    def invoke(self, *extra):
        with tempfile.TemporaryDirectory() as tmp:
            output = Path(tmp) / 'result.json'
            proc = subprocess.run([sys.executable, str(DRIVER), 'run', '--repo', str(DRIVER.parents[3]),
                                   '--work-dir', tmp, '--output', str(output), '--suite', 'compiler', *extra],
                                  capture_output=True, text=True, timeout=10)
            return proc.returncode, json.loads(output.read_text())

    def test_unprepared_run_is_blocked(self):
        code, report = self.invoke()
        self.assertEqual(code, 2)
        self.assertEqual(report['status'], 'blocked')
        self.assertIn('successful', report['reason'])
        self.assertEqual(report['schema_version'], 1)

    def test_unsupported_size_never_silently_substituted(self):
        code, report = self.invoke('--size', '11')
        self.assertEqual(code, 2)
        self.assertIn('11', report['reason'])
        self.assertEqual(report['cases'], [])

    def test_invalid_iteration_contract_fails(self):
        code, report = self.invoke('--iterations', '0')
        self.assertEqual(code, 1)
        self.assertEqual(report['status'], 'failed')
        self.assertIn('positive', report['reason'])

    def test_child_group_follows_supervisor_ownership(self):
        spec = importlib.util.spec_from_file_location('compiler_perf_driver_test', DRIVER)
        module = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(module)
        for supervised in (True, False):
            with self.subTest(supervised=supervised):
                child = MagicMock()
                child.pid = 12345
                child.communicate.side_effect = subprocess.TimeoutExpired(['helper'], 1)
                with patch.object(module.subprocess, 'Popen', return_value=child) as launch, patch.object(module.os, 'killpg') as kill_group:
                    with self.assertRaises(subprocess.TimeoutExpired):
                        module.command(['helper'], Path('/tmp'), env={'CAN_PERF_SUPERVISED': '1' if supervised else '0'}, timeout=1)
                    self.assertEqual(launch.call_args.kwargs['start_new_session'], not supervised)
                    if supervised:
                        child.kill.assert_called_once()
                        kill_group.assert_not_called()
                    else:
                        child.kill.assert_not_called()
                        kill_group.assert_called_once_with(12345, module.signal.SIGKILL)
                    child.wait.assert_called_once_with(timeout=10)

    def load_driver(self):
        spec = importlib.util.spec_from_file_location('compiler_perf_oracle_test', DRIVER)
        module = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(module)
        return module

    def test_completion_rejects_wrong_checked_arity_and_extra_authored_symbol(self):
        driver = self.load_driver()
        items = [{'label': 'f0', 'kind': 3, 'detail': '() -> int', 'documentation': 'package audit'},
                 {'label': 'main', 'kind': 3, 'detail': '(args: str[]) -> void', 'documentation': 'package audit'}]
        driver.validate_completion(items, 1)
        wrong = [dict(items[0], detail='() -> str'), items[1]]
        with self.assertRaises(RuntimeError):
            driver.validate_completion(wrong, 1)
        with self.assertRaises(RuntimeError):
            driver.validate_completion(items + [dict(items[0], label='f1')], 1)

    def test_rename_oracle_preserves_prefix_neighbor_and_rejects_extra_edit(self):
        driver = self.load_driver()
        text = '        call f0() as int v0\nfn int f0\nfn int f01\n'
        expected = driver.expected_rename('file:///fixture.can', 0, 1)
        driver.validate_rename(expected, expected)
        changed = driver.apply_workspace_edits(text, expected['changes']['file:///fixture.can'])
        self.assertEqual(changed, '        call f_zero() as int v0\nfn int f_zero\nfn int f01\n')
        extra = {'changes': {'file:///fixture.can': expected['changes']['file:///fixture.can'] + [dict(expected['changes']['file:///fixture.can'][0], newText='hijacked')]}}
        with self.assertRaises(RuntimeError):
            driver.validate_rename(extra, expected)

    def test_anchor_identity_excludes_locations_outputs_and_docs_but_detects_input_edits(self):
        driver = self.load_driver()
        with tempfile.TemporaryDirectory() as tmp:
            roots = [Path(tmp) / name for name in ('first', 'different-location')]
            for root in roots:
                (root / 'src').mkdir(parents=True)
                (root / 'src/main.can').write_text('package app\n')
                (root / 'can.project.json').write_text('{}')
                (root / 'dist').mkdir()
                (root / 'dist/output.ts').write_text(str(root))
                (root / 'README.md').write_text(str(root))
                (root / 'prepared.json').write_text(str(root))
            first = driver.input_inventory(roots[0])
            second = driver.input_inventory(roots[1])
            self.assertEqual(first, second)
            self.assertEqual(driver.input_identity(first), driver.input_identity(second))
            (roots[1] / 'src/main.can').write_text('package changed\n')
            self.assertNotEqual(driver.input_identity(first), driver.input_identity(driver.input_inventory(roots[1])))


if __name__ == '__main__':
    unittest.main()
