"""Lightweight editor-experiment regressions; no builds or benchmarks.

Focused() runs against a bounded fake LSP transport. Validation and summary
run against synthetic evidence directories. All temporary files live in
TemporaryDirectory and retire with the test.
"""
import argparse
import copy
import hashlib
import json
import os
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

import editor_experiment
from editor_experiment import BATCHES, CONTRACT, NEIGHBOR_CASES, SCENARIOS, WARMUPS


def scripted_completion(size):
    return [{'label': f'f{i}', 'kind': 3, 'detail': '() -> int', 'documentation': 'package audit'}
            for i in range(size)] + [{'label': 'main', 'kind': 3, 'detail': '(args: str[]) -> void',
                                      'documentation': 'package audit'}]


class FakeLSP:
    """Bounded fake Content-Length transport speaking the focused script."""

    completions = None
    last = None

    def __init__(self, executable, repo, request_timeout=30):
        self.queue = []
        self.closed = False
        self.calls = 0
        FakeLSP.last = self

    def send(self, method, params, identity=None):
        if method == 'initialize':
            self.queue.append({'id': identity, 'result': {}})
        elif method == 'initialized':
            pass
        elif method == 'textDocument/didOpen':
            td = params['textDocument']
            self.queue.append({'method': 'textDocument/publishDiagnostics',
                               'params': {'uri': td['uri'], 'version': td['version'], 'diagnostics': []}})
        elif method == 'textDocument/didChange':
            td = params['textDocument']
            text = params['contentChanges'][0]['text']
            bad = '\n@\n' in text
            diags = [{'message': 'syntax'}] if bad else []
            self.queue.append({'method': 'textDocument/publishDiagnostics',
                               'params': {'uri': td['uri'], 'version': td['version'], 'diagnostics': diags}})
        elif method == 'textDocument/completion':
            self.calls += 1
            script = FakeLSP.completions
            result = script(self.calls) if callable(script) else script
            self.queue.append({'id': identity, 'result': result})
        else:
            raise AssertionError(f'unexpected method {method}')

    def until(self, predicate, allow_errors=False):
        for i, message in enumerate(self.queue):
            if predicate(message):
                return self.queue.pop(i)
        raise AssertionError('no queued message matches predicate')

    def close(self):
        self.closed = True


FLAT_TEXT = ('package audit\n    provides [main]\n    uses []\n\nfn void main\n    emits {}\n    given\n'
             '        str[] args\n    asserts\n        smoke: [] => ok\n    match chain\n'
             '        call f0() as int v0\n        ok => ok\n\nfn int f0\n    emits {}\n    asserts\n'
             '        sample: => ok 0\n    ok 0\n')


def make_project(directory):
    (directory / 'src').mkdir(parents=True)
    (directory / 'src/main.can').write_text(FLAT_TEXT)
    return directory


class ValidateArgsTests(unittest.TestCase):
    def base(self, directory):
        project = make_project(directory / 'project')
        executable = directory / 'canlc'
        executable.write_text('fake')
        return argparse.Namespace(trial=True, output=directory / 'out.json', project=project,
                                  executable=executable, oracle=directory / 'oracle.json',
                                  fixture='flat-10', size=10, trace=None, neighbors=False)

    def test_trial_accepts_matching_fixture_and_resolves_paths(self):
        with tempfile.TemporaryDirectory() as tmp:
            args = self.base(Path(tmp))
            editor_experiment.validate_args(args)
            self.assertTrue(args.output.is_absolute())
            self.assertTrue(args.project.is_absolute())

    def test_trial_rejects_unknown_fixture(self):
        with tempfile.TemporaryDirectory() as tmp:
            args = self.base(Path(tmp))
            args.fixture = 'flat-1000'
            with self.assertRaisesRegex(ValueError, 'unknown fixture'):
                editor_experiment.validate_args(args)

    def test_trial_rejects_size_mismatch(self):
        with tempfile.TemporaryDirectory() as tmp:
            args = self.base(Path(tmp))
            args.size = 100
            with self.assertRaisesRegex(ValueError, 'does not match fixture'):
                editor_experiment.validate_args(args)

    def test_trial_requires_paths(self):
        with tempfile.TemporaryDirectory() as tmp:
            args = self.base(Path(tmp))
            args.executable = None
            with self.assertRaisesRegex(ValueError, '--executable'):
                editor_experiment.validate_args(args)

    def test_trial_rejects_missing_project(self):
        with tempfile.TemporaryDirectory() as tmp:
            args = self.base(Path(tmp))
            args.project = Path(tmp) / 'absent'
            with self.assertRaisesRegex(ValueError, 'not a directory'):
                editor_experiment.validate_args(args)

    def test_session_rejects_missing_archive(self):
        with tempfile.TemporaryDirectory() as tmp:
            args = argparse.Namespace(trial=False, output=Path(tmp) / 'run', archive=str(Path(tmp) / 'bun.zip'),
                                      wait_seconds=180)
            with self.assertRaisesRegex(ValueError, 'archive missing'):
                editor_experiment.validate_args(args)

    def test_session_rejects_nonpositive_wait(self):
        with tempfile.TemporaryDirectory() as tmp:
            archive = Path(tmp) / 'bun.zip'
            archive.write_bytes(b'fake')
            args = argparse.Namespace(trial=False, output=Path(tmp) / 'run', archive=str(archive), wait_seconds=0)
            with self.assertRaisesRegex(ValueError, 'wait-seconds'):
                editor_experiment.validate_args(args)


class FocusedTests(unittest.TestCase):
    def run_focused(self, directory, size=2, batches=2, warmups=1, fixture='flat-test'):
        project = make_project(directory / 'project')
        oracle = directory / 'oracle.json'
        args = argparse.Namespace(fixture=fixture, size=size)
        with patch.object(editor_experiment.driver, 'LSP', FakeLSP):
            return editor_experiment.focused(args, project, directory / 'canlc', oracle, batches, warmups), oracle

    def test_sample_counts_and_warmup_exclusion(self):
        FakeLSP.completions = scripted_completion(2)
        with tempfile.TemporaryDirectory() as tmp:
            row, oracle = self.run_focused(Path(tmp))
            self.assertEqual(row['contract'], CONTRACT)
            self.assertEqual(set(row['samples_ns']), set(SCENARIOS))
            for name in SCENARIOS:
                self.assertEqual(len(row['samples_ns'][name]), 2, name)
                self.assertEqual(len(row['warmups_ns'][name]), 1, name)
            self.assertTrue(oracle.is_file(), 'first run captures the oracle')
            self.assertIn('monotonic', row['timing'])
            self.assertIn('exact entire completion array', row['correctness'])

    def test_false_oracle_rejected_and_server_closed(self):
        with tempfile.TemporaryDirectory() as tmp:
            directory = Path(tmp)
            expected = scripted_completion(2)
            (directory / 'oracle.json').write_text(json.dumps(expected))
            tampered = copy.deepcopy(expected) + [{'label': 'extra', 'kind': 3}]
            calls = {'n': 0}

            def script(n):
                calls['n'] = n
                return expected if n < 3 else tampered

            FakeLSP.completions = script
            project = make_project(directory / 'project')
            args = argparse.Namespace(fixture='flat-test', size=2)
            with patch.object(editor_experiment.driver, 'LSP', FakeLSP):
                with self.assertRaisesRegex(RuntimeError, 'differs from baseline oracle'):
                    editor_experiment.focused(args, project, directory / 'canlc', directory / 'oracle.json', 2, 1)
            self.assertTrue(FakeLSP.last.closed, 'trial transport must close on oracle failure')

    def test_authored_signatures_validated_independently(self):
        with tempfile.TemporaryDirectory() as tmp:
            broken = [item for item in scripted_completion(2) if item['label'] != 'f1']
            FakeLSP.completions = broken
            project = make_project(Path(tmp) / 'project')
            args = argparse.Namespace(fixture='flat-test', size=2)
            with patch.object(editor_experiment.driver, 'LSP', FakeLSP):
                with self.assertRaisesRegex(RuntimeError, 'callable set'):
                    editor_experiment.focused(args, project, Path(tmp) / 'canlc', Path(tmp) / 'oracle.json', 1, 0)

    def test_syntax_error_oracle_detects_clean_impostor(self):
        with tempfile.TemporaryDirectory() as tmp:
            FakeLSP.completions = scripted_completion(2)
            real_send = FakeLSP.send

            def always_clean(self, method, params, identity=None):
                if method == 'textDocument/didChange':
                    params = copy.deepcopy(params)
                    params['contentChanges'][0]['text'] = FLAT_TEXT
                return real_send(self, method, params, identity)

            project = make_project(Path(tmp) / 'project')
            args = argparse.Namespace(fixture='flat-test', size=2)
            with patch.object(editor_experiment.driver, 'LSP', FakeLSP), patch.object(FakeLSP, 'send', always_clean):
                with self.assertRaisesRegex(RuntimeError, 'diagnostics oracle failed'):
                    editor_experiment.focused(args, project, Path(tmp) / 'canlc', Path(tmp) / 'oracle.json', 1, 0)

    def test_malformed_oracle_starts_no_child(self):
        for payload, pattern in (('{bad json', 'not valid JSON'),
                                 ('{}', 'non-empty completion array'),
                                 ('[]', 'non-empty completion array')):
            with self.subTest(payload=payload):
                FakeLSP.last = None
                FakeLSP.completions = scripted_completion(2)
                with tempfile.TemporaryDirectory() as tmp:
                    directory = Path(tmp)
                    project = make_project(directory / 'project')
                    (directory / 'oracle.json').write_text(payload)
                    args = argparse.Namespace(fixture='flat-test', size=2)
                    with patch.object(editor_experiment.driver, 'LSP', FakeLSP):
                        with self.assertRaisesRegex(RuntimeError, pattern):
                            editor_experiment.focused(args, project, directory / 'canlc',
                                                      directory / 'oracle.json', 1, 0)
                self.assertIsNone(FakeLSP.last, 'malformed oracle must not start a child')

    def test_oracle_validation_outside_clock(self):
        FakeLSP.completions = scripted_completion(2)
        ticks = {'now': 0}

        def fake_counter():
            ticks['now'] += 100
            return ticks['now']

        original_validate = editor_experiment.driver.validate_completion

        def advancing_validate(response, size):
            ticks['now'] += 50_000_000
            return original_validate(response, size)

        class AdvancingList(list):
            def __bool__(self):
                ticks['now'] += 50_000_000
                return len(self) > 0

        real_send = FakeLSP.send

        def send_with_advancing(self, method, params, identity=None):
            if method == 'textDocument/didChange':
                td = params['textDocument']
                text = params['contentChanges'][0]['text']
                bad = '\n@\n' in text
                diags = AdvancingList([{'message': 'syntax'}] if bad else [])
                self.queue.append({'method': 'textDocument/publishDiagnostics',
                                   'params': {'uri': td['uri'], 'version': td['version'], 'diagnostics': diags}})
                return
            return real_send(self, method, params, identity)

        with tempfile.TemporaryDirectory() as tmp:
            directory = Path(tmp)
            project = make_project(directory / 'project')
            oracle = directory / 'oracle.json'
            args = argparse.Namespace(fixture='flat-test', size=2)
            with patch.object(editor_experiment.time, 'perf_counter_ns', fake_counter), \
                    patch.object(editor_experiment.driver, 'LSP', FakeLSP), \
                    patch.object(FakeLSP, 'send', send_with_advancing), \
                    patch.object(editor_experiment.driver, 'validate_completion', advancing_validate):
                row = editor_experiment.focused(args, project, directory / 'canlc', oracle, 2, 1)
            self.assertTrue(oracle.is_file(), 'oracle still captured')
            self.assertIn('after clock stop', row['timing'])
            self.assertIn('both versioned diagnostics and completion', row['timing'])
            for name in SCENARIOS:
                for value in row['samples_ns'][name] + row['warmups_ns'][name]:
                    self.assertEqual(value, 100, f'{name} includes validation time: {value}')
            self.assertGreater(ticks['now'], 1_000_000, 'validations did not advance the fake clock')


def make_bundle(root, files=None):
    """Minimal stamped bundle: manifest-listed assets plus manifest/launcher."""
    bundle = Path(root) / 'bundle'
    (bundle / 'bin').mkdir(parents=True)
    names = list(files) if files is not None else ['runtime/bun', 'tools/runtime/check.ts']
    manifest_files = {}
    for name in names:
        path = bundle / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_bytes(b'asset:' + name.encode())
        manifest_files[name] = hashlib.sha256(path.read_bytes()).hexdigest()
    manifest = {'schemaVersion': 1, 'kind': 'can.development-distribution',
                'version': 'editor-A', 'targetId': 'test-target', 'files': manifest_files}
    manifest_bytes = (json.dumps(manifest, sort_keys=True) + '\n').encode()
    (bundle / 'manifest.json').write_bytes(manifest_bytes)
    digest = hashlib.sha256(manifest_bytes).hexdigest()
    (bundle / 'bin' / 'canlc').write_bytes(b'fake-launcher-stamp:' + digest.encode())
    return bundle


def qualifying_run(command, name, timeout=30):
    return json.dumps({'kind': 'can.development-runtime-check', 'schemaVersion': 1})


class QualifyTests(unittest.TestCase):
    def test_accepts_valid_runtime_report(self):
        output = json.dumps({'kind': 'can.development-runtime-check', 'schemaVersion': 1, 'bun': '1.4.2'})
        report = editor_experiment.check_runtime_report(output)
        self.assertEqual(report['kind'], 'can.development-runtime-check')

    def test_rejects_non_json_output(self):
        with self.assertRaisesRegex(RuntimeError, 'not JSON'):
            editor_experiment.check_runtime_report('canlc editor-A\n')

    def test_rejects_wrong_kind_or_schema(self):
        for output in (json.dumps({'kind': 'other', 'schemaVersion': 1}),
                       json.dumps({'kind': 'can.development-runtime-check', 'schemaVersion': 2}),
                       json.dumps({})):
            with self.subTest(output=output), self.assertRaisesRegex(RuntimeError, 'report mismatch'):
                editor_experiment.check_runtime_report(output)

    def test_uses_resolving_tool_not_version(self):
        seen = {}

        def fake_run(command, name, timeout=30):
            seen['command'] = command
            seen['name'] = name
            return json.dumps({'kind': 'can.development-runtime-check', 'schemaVersion': 1})

        with tempfile.TemporaryDirectory() as tmp:
            bundle = make_bundle(Path(tmp))
            editor_experiment.qualify('A', bundle / 'bin' / 'canlc', fake_run)
        self.assertEqual(seen['command'][-1], 'runtime-check')
        self.assertNotIn('version', seen['command'])
        self.assertEqual(seen['name'], 'qualify-A')

    def test_failed_tool_exit_cannot_proceed_to_trial(self):
        def failing_run(command, name, timeout=30):
            raise RuntimeError(f'{name} failed: CAN-DIST-INTEGRITY: modified asset runtime/bun')

        with self.assertRaisesRegex(RuntimeError, 'qualify-A failed'):
            editor_experiment.qualify('A', Path('/tmp/fake/bin/canlc'), failing_run)

    def test_mismatched_report_cannot_proceed_to_trial(self):
        def bad_report_run(command, name, timeout=30):
            return 'canlc editor-A\n'

        with self.assertRaisesRegex(RuntimeError, 'not JSON'):
            editor_experiment.qualify('A', Path('/tmp/fake/bin/canlc'), bad_report_run)

    def test_accepts_closed_bundle_inventory(self):
        with tempfile.TemporaryDirectory() as tmp:
            bundle = make_bundle(Path(tmp))
            digest = editor_experiment.verify_closed_bundle_inventory(bundle / 'bin' / 'canlc')
            self.assertEqual(digest, hashlib.sha256((bundle / 'manifest.json').read_bytes()).hexdigest())
            report = editor_experiment.qualify('A', bundle / 'bin' / 'canlc', qualifying_run)
            self.assertEqual(report['kind'], 'can.development-runtime-check')

    def test_rejects_unexpected_file(self):
        with tempfile.TemporaryDirectory() as tmp:
            bundle = make_bundle(Path(tmp))
            (bundle / 'extra.txt').write_text('not in manifest')
            with self.assertRaisesRegex(RuntimeError, 'unknown file'):
                editor_experiment.qualify('A', bundle / 'bin' / 'canlc', qualifying_run)

    def test_rejects_symlinked_file(self):
        with tempfile.TemporaryDirectory() as tmp:
            bundle = make_bundle(Path(tmp))
            (bundle / 'link.txt').symlink_to(bundle / 'manifest.json')
            with self.assertRaisesRegex(RuntimeError, 'symlinked entry'):
                editor_experiment.qualify('A', bundle / 'bin' / 'canlc', qualifying_run)

    def test_rejects_symlinked_directory(self):
        with tempfile.TemporaryDirectory() as tmp:
            bundle = make_bundle(Path(tmp))
            (bundle / 'runtime-link').symlink_to(bundle / 'runtime', target_is_directory=True)
            with self.assertRaisesRegex(RuntimeError, 'symlinked entry'):
                editor_experiment.qualify('A', bundle / 'bin' / 'canlc', qualifying_run)

    def test_rejects_nonregular_entry(self):
        with tempfile.TemporaryDirectory() as tmp:
            bundle = make_bundle(Path(tmp))
            os.mkfifo(bundle / 'pipe')
            with self.assertRaisesRegex(RuntimeError, 'non-regular entry'):
                editor_experiment.qualify('A', bundle / 'bin' / 'canlc', qualifying_run)

    def test_rejects_launcher_stamp_mismatch(self):
        with tempfile.TemporaryDirectory() as tmp:
            bundle = make_bundle(Path(tmp))
            (bundle / 'bin' / 'canlc').write_bytes(b'unstamped-launcher')
            with self.assertRaisesRegex(RuntimeError, 'launcher stamp'):
                editor_experiment.qualify('A', bundle / 'bin' / 'canlc', qualifying_run)

    def test_rejects_non_launcher_path(self):
        with tempfile.TemporaryDirectory() as tmp:
            bundle = make_bundle(Path(tmp))
            with self.assertRaisesRegex(RuntimeError, 'not a bundle launcher'):
                editor_experiment.qualify('A', bundle / 'manifest.json', qualifying_run)


def valid_neighbor(name, size=100):
    return {'name': name, 'unit': 'ns/op', 'samples': [1000000 + i for i in range(BATCHES)],
            'warmup_samples': [2000000, 2000001], 'iterations_per_sample': 1,
            'timing_scope': 'client frame write to response', 'parameters': {'size': size},
            'correctness': {'passed': True, 'checks': ['exact oracle']},
            'metrics': {'request_latencies_ns': [1000000] * BATCHES}}


def valid_isolation_lines(stems):
    """Labeled intervals mirroring session order: lock, two traces, trials."""
    lines = [json.dumps({'time': 1.0, 'event': 'lock-acquired', 'wait_seconds': 0.0})]
    for label in ('A', 'B'):
        name = f'trace-flat-100-00-{label}'
        lines.append(json.dumps({'time': 1.0, 'event': 'trial-start', 'trial': name}))
        lines.append(json.dumps({'time': 1.0, 'event': 'monitor-final', 'competing': []}))
        lines.append(json.dumps({'time': 1.0, 'event': 'trial-end', 'trial': name}))
    for stem in stems:
        lines.append(json.dumps({'time': 1.0, 'event': 'trial-start', 'trial': stem}))
        lines.append(json.dumps({'time': 1.0, 'event': 'quiet-sample', 'idle_percent': 95.0,
                                 'competing': [], 'continuous_seconds': 30.0}))
        lines.append(json.dumps({'time': 1.0, 'event': 'quiet-ready'}))
        lines.append(json.dumps({'time': 1.0, 'event': 'monitor-final', 'competing': []}))
        lines.append(json.dumps({'time': 1.0, 'event': 'trial-end', 'trial': stem}))
    lines.append(json.dumps({'time': 1.0, 'event': 'lock-released'}))
    return '\n'.join(lines) + '\n'


def interval_span(events, stem):
    """Start/end indices of one labeled trial interval."""
    start = next(i for i, event in enumerate(events)
                 if event.get('event') == 'trial-start' and event.get('trial') == stem)
    end = next(i for i, event in enumerate(events)
               if event.get('event') == 'trial-end' and event.get('trial') == stem)
    return start, end


def first_measured_stem():
    return next(iter(editor_experiment.expected_trials()))


def first_quiet_sample(events):
    start, end = interval_span(events, first_measured_stem())
    return next(events[i] for i in range(start, end) if events[i].get('event') == 'quiet-sample')


def first_monitor_final(events):
    start, end = interval_span(events, first_measured_stem())
    return next(events[i] for i in range(start, end) if events[i].get('event') == 'monitor-final')


def first_monitor_final_index(events):
    start, _ = interval_span(events, first_measured_stem())
    return next(i for i in range(start, len(events)) if events[i].get('event') == 'monitor-final')


def swap_first_intervals(events):
    stems = list(editor_experiment.expected_trials())[:2]
    first, second = interval_span(events, stems[0]), interval_span(events, stems[1])
    if first[1] + 1 != second[0]:
        raise AssertionError('measurement intervals must be adjacent in valid evidence')
    block = events[first[0]:second[1] + 1]
    cut = first[1] - first[0] + 1
    events[first[0]:second[1] + 1] = block[cut:] + block[:cut]


def second_measured_stem():
    return list(editor_experiment.expected_trials())[1]


HARNESS = 'h0'
FIXTURES = {'flat-10': 'fix10', 'flat-100': 'fix100', 'invoice-compare': 'fixinv'}
BUILDS = {'A': 'exeA', 'B': 'exeB'}


def valid_row(fixture, label, oracle_sha):
    row = {'contract': CONTRACT, 'fixture': fixture,
           'size': editor_experiment.SIZE_BY_FIXTURE[fixture], 'variant': label, 'quality': 'measurement',
           'input_content_sha256': FIXTURES[fixture], 'harness_sha256': HARNESS, 'executable_sha256': BUILDS[label],
           'executable_sha256_before': BUILDS[label],
           'oracle_sha256': oracle_sha, 'samples_ns': {n: [1000000 + i for i in range(BATCHES)] for n in SCENARIOS},
           'warmups_ns': {n: [2000000, 2000001] for n in SCENARIOS},
           'correctness': 'exact entire completion array', 'timing': 'client monotonic clock'}
    if fixture != 'invoice-compare':
        size = editor_experiment.SIZE_BY_FIXTURE[fixture]
        row['neighbors'] = [valid_neighbor(name, size) for name in sorted(NEIGHBOR_CASES)]
    return row


def build_evidence(root):
    raw = root / 'raw'
    raw.mkdir(parents=True)
    oracle_shas = {}
    for fixture in FIXTURES:
        path = raw / (fixture + '.oracle.json')
        path.write_text(json.dumps(scripted_completion(2)))
        oracle_shas[fixture] = editor_experiment.sha(path)
    expected = editor_experiment.expected_trials()
    for stem, (fixture, label) in expected.items():
        (raw / (stem + '.json')).write_text(json.dumps(valid_row(fixture, label, oracle_shas[fixture])))
    manifest = {'contract': CONTRACT, 'fixtures': dict(FIXTURES), 'harness_sha256': HARNESS,
                'isolation': {'minimum_idle_percent': 90, 'continuous_seconds': 30},
                'builds': {'A': {'executable_sha256': 'exeA'}, 'B': {'executable_sha256': 'exeB'}},
                'steps': [{'name': stem, 'returncode': 0, 'elapsed_seconds': 1.0} for stem in expected]}
    (root / 'manifest.json').write_text(json.dumps(manifest))
    (root / 'isolation.jsonl').write_text(valid_isolation_lines(list(expected.keys())))
    return raw, expected


def rewrite(raw, stem, mutate):
    path = raw / (stem + '.json')
    row = json.loads(path.read_text())
    mutate(row)
    path.write_text(json.dumps(row))


class SessionSetupTests(unittest.TestCase):
    def test_early_tmp_failure_retires_owned_work(self):
        import zipfile
        with tempfile.TemporaryDirectory() as tmp:
            parent = Path(tmp)
            archive = parent / 'bun.zip'
            archive.write_bytes(b'fake-archive')
            output = parent / 'run-early-fail'
            args = argparse.Namespace(trial=False, output=output, archive=str(archive), wait_seconds=1)
            real_mkdir = Path.mkdir

            def failing_mkdir(self, *mkdir_args, **mkdir_kwargs):
                if self.name == 'tmp':
                    raise RuntimeError('injected tmp creation failure')
                return real_mkdir(self, *mkdir_args, **mkdir_kwargs)

            with patch.object(Path, 'mkdir', failing_mkdir):
                with self.assertRaisesRegex(RuntimeError, 'injected tmp creation failure'):
                    editor_experiment.session(args)
            self.assertIsNone(editor_experiment.isolation.PROCESS_OBSERVER)
            self.assertFalse((output / 'work').exists(), 'owned work dir must retire')
            self.assertTrue((output / 'evidence.zip').is_file(), 'incomplete evidence must archive')
            with zipfile.ZipFile(output / 'evidence.zip') as bundle:
                names = bundle.namelist()
                self.assertIn('manifest.json', names)
                self.assertNotIn('summary.json', names)
                manifest = json.loads(bundle.read('manifest.json'))
            self.assertEqual(manifest['status'], 'incomplete')
            self.assertIn('injected tmp creation failure', manifest['error'])
            self.assertEqual(manifest.get('cleanup'), 'verified standard scratch retirement')
            marker = json.loads((output / '.performance-owner.json').read_text())
            self.assertEqual(marker['state'], 'cleaned')

    def test_manifest_construction_failure_archives_honest_evidence(self):
        import zipfile
        with tempfile.TemporaryDirectory() as tmp:
            parent = Path(tmp)
            archive = parent / 'bun.zip'
            archive.write_bytes(b'fake-archive')
            output = parent / 'run-manifest-none'
            args = argparse.Namespace(trial=False, output=output, archive=str(archive), wait_seconds=1)
            with patch.object(editor_experiment, 'harness_identity',
                              side_effect=RuntimeError('injected manifest failure')):
                with self.assertRaisesRegex(RuntimeError, 'injected manifest failure'):
                    editor_experiment.session(args)
            self.assertIsNone(editor_experiment.isolation.PROCESS_OBSERVER)
            self.assertFalse((output / 'work').exists(), 'owned work dir must retire')
            self.assertFalse((output / 'raw').exists(), 'raw dir must never have been created')
            self.assertTrue((output / 'evidence.zip').is_file(), 'incomplete evidence must archive')
            with zipfile.ZipFile(output / 'evidence.zip') as bundle:
                names = bundle.namelist()
                self.assertIn('manifest.json', names)
                self.assertNotIn('summary.json', names)
                manifest = json.loads(bundle.read('manifest.json'))
            self.assertEqual(manifest['status'], 'incomplete')
            self.assertIn('injected manifest failure', manifest['error'])
            self.assertEqual(manifest.get('cleanup'), 'verified standard scratch retirement')
            marker = json.loads((output / '.performance-owner.json').read_text())
            self.assertEqual(marker['state'], 'cleaned')


class WorkerTests(unittest.TestCase):
    def base_args(self, directory):
        project = make_project(directory / 'project')
        executable = directory / 'canlc'
        executable.write_text('fake-binary')
        return argparse.Namespace(trial=True, output=directory / 'out.json', project=project,
                                  executable=executable, oracle=directory / 'oracle.json',
                                  fixture='flat-10', size=10, trace=None, neighbors=False)

    def test_worker_stamps_executable_before_and_after(self):
        FakeLSP.completions = scripted_completion(10)
        with tempfile.TemporaryDirectory() as tmp:
            directory = Path(tmp)
            args = self.base_args(directory)
            with patch.object(editor_experiment.driver, 'LSP', FakeLSP):
                editor_experiment.worker(args)
            row = json.loads(args.output.read_text())
            expected_exe = editor_experiment.sha(args.executable)
            self.assertEqual(row['executable_sha256'], expected_exe)
            self.assertEqual(row['executable_sha256_before'], expected_exe)
            self.assertEqual(row['input_content_sha256'],
                             editor_experiment.driver.input_identity(
                                 editor_experiment.driver.input_inventory(args.project)))

    def test_worker_rejects_executable_change(self):
        FakeLSP.completions = scripted_completion(10)
        with tempfile.TemporaryDirectory() as tmp:
            directory = Path(tmp)
            args = self.base_args(directory)
            real_focused = editor_experiment.focused

            def tampering_focused(fargs, fdir, fexe, foracle, batches, warmups):
                result = real_focused(fargs, fdir, fexe, foracle, batches, warmups)
                Path(fexe).write_text('tampered-binary')
                return result

            with patch.object(editor_experiment.driver, 'LSP', FakeLSP), \
                    patch.object(editor_experiment, 'focused', tampering_focused):
                with self.assertRaisesRegex(RuntimeError, 'executable changed'):
                    editor_experiment.worker(args)

    def test_worker_rejects_fixture_change(self):
        FakeLSP.completions = scripted_completion(10)
        with tempfile.TemporaryDirectory() as tmp:
            directory = Path(tmp)
            args = self.base_args(directory)
            real_focused = editor_experiment.focused

            def tampering_focused(fargs, fdir, fexe, foracle, batches, warmups):
                result = real_focused(fargs, fdir, fexe, foracle, batches, warmups)
                (Path(fdir) / 'src/main.can').write_text(FLAT_TEXT + '\n// tampered\n')
                return result

            with patch.object(editor_experiment.driver, 'LSP', FakeLSP), \
                    patch.object(editor_experiment, 'focused', tampering_focused):
                with self.assertRaisesRegex(RuntimeError, 'fixture inputs changed'):
                    editor_experiment.worker(args)

    def test_worker_propagates_lsp_close_failure(self):
        FakeLSP.completions = scripted_completion(10)
        with tempfile.TemporaryDirectory() as tmp:
            directory = Path(tmp)
            args = self.base_args(directory)

            def failing_close(self):
                raise RuntimeError('LSP process exited 2; trial is invalid')

            with patch.object(editor_experiment.driver, 'LSP', FakeLSP), \
                    patch.object(FakeLSP, 'close', failing_close):
                with self.assertRaisesRegex(RuntimeError, 'exited 2'):
                    editor_experiment.worker(args)
            self.assertFalse(args.output.exists(), 'failed worker must not write a trial row')


class ValidateTrialsTests(unittest.TestCase):
    def test_accepts_complete_comparison(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            build_evidence(root)
            rows = editor_experiment.validate_trials(root)
            self.assertEqual(len(rows), 20)

    def test_functional_trace_rows_are_checked_but_excluded(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            raw, _ = build_evidence(root)
            for label in ('A', 'B'):
                row = valid_row('flat-100', label, editor_experiment.sha(raw / 'flat-100.oracle.json'))
                row['quality'] = 'functional-trace'
                (raw / f'trace-flat-100-00-{label}.json').write_text(json.dumps(row))
            rows = editor_experiment.validate_trials(root)
            self.assertEqual(len(rows), 20)
            editor_experiment.summarize(root)
            report = json.loads((root / 'summary.json').read_text())
            for record in report['comparisons']:
                self.assertLessEqual(record['A']['trials'] + record['B']['trials'], 20)

    def test_rejections(self):
        cases = {
            'contract': lambda raw, root: rewrite(raw, 'trial-flat-100-01-A', lambda r: r.update(contract='other')),
            'missing-file': lambda raw, root: (raw / 'trial-flat-100-01-A.json').unlink(),
            'extra-file': lambda raw, root: (raw / 'trial-flat-100-13-A.json').write_text('{}'),
            'wrong-label': lambda raw, root: rewrite(raw, 'trial-flat-100-01-A', lambda r: r.update(variant='B')),
            'samples': lambda raw, root: rewrite(raw, 'trial-flat-100-01-A', lambda r: r['samples_ns']['completion-unchanged'].pop()),
            'warmups': lambda raw, root: rewrite(raw, 'trial-flat-100-01-A', lambda r: r['warmups_ns']['completion-unchanged'].append(1)),
            'oracle-link': lambda raw, root: rewrite(raw, 'trial-flat-100-01-A', lambda r: r.update(oracle_sha256='dead')),
            'empty-oracle': lambda raw, root: (raw / 'flat-100.oracle.json').write_text('[]'),
            'trace-quality': lambda raw, root: rewrite(raw, 'trial-flat-100-01-A', lambda r: r.update(quality='functional-trace')),
            'harness': lambda raw, root: rewrite(raw, 'trial-flat-100-01-A', lambda r: r.update(harness_sha256='other')),
            'fixture-inputs': lambda raw, root: rewrite(raw, 'trial-flat-100-01-A', lambda r: r.update(input_content_sha256='other')),
            'executable': lambda raw, root: rewrite(raw, 'trial-flat-100-01-A', lambda r: r.update(executable_sha256='other')),
            'executable-before': lambda raw, root: rewrite(raw, 'trial-flat-100-01-A', lambda r: r.update(executable_sha256_before='other')),
            'failed-exit': lambda raw, root: self.rewrite_manifest(root, lambda m: m['steps'][0].update(returncode=1)),
            'missing-step': lambda raw, root: self.rewrite_manifest(root, lambda m: m['steps'].pop(0)),
            'manifest-contract': lambda raw, root: self.rewrite_manifest(root, lambda m: m.update(contract='other')),
            'neighbor-absent': lambda raw, root: rewrite(raw, 'trial-flat-100-01-A', lambda r: r['neighbors'].pop(0)),
            'neighbor-schema': lambda raw, root: rewrite(raw, 'trial-flat-100-01-A', lambda r: r['neighbors'][0].pop('unit')),
            'neighbor-latencies': lambda raw, root: rewrite(raw, 'trial-flat-100-01-A', lambda r: r['neighbors'][0]['metrics'].clear()),
            'invoice-neighbors': lambda raw, root: rewrite(raw, 'trial-invoice-compare-01-A', lambda r: r.update(neighbors=[])),
            'mislabeled-trace': lambda raw, root: (raw / 'trace-flat-100-00-A.json').write_text(json.dumps(
                dict(valid_row('flat-100', 'A', editor_experiment.sha(raw / 'flat-100.oracle.json')), quality='measurement'))),
            'reordered-steps': lambda raw, root: self.rewrite_manifest(
                root, lambda m: m['steps'].__setitem__(slice(0, 2), [m['steps'][1], m['steps'][0]])),
            'duplicated-steps': lambda raw, root: self.rewrite_manifest(
                root, lambda m: m['steps'].append(dict(m['steps'][0]))),
            'missing-isolation': lambda raw, root: (root / 'isolation.jsonl').unlink(),
            'missing-gate': lambda raw, root: self.rewrite_isolation(
                root, lambda events: events.pop(next(
                    i for i, event in enumerate(events) if event.get('event') == 'quiet-ready'))),
            'low-idle': lambda raw, root: self.rewrite_isolation(
                root, lambda events: first_quiet_sample(events).update(idle_percent=50.0)),
            'short-window': lambda raw, root: self.rewrite_isolation(
                root, lambda events: first_quiet_sample(events).update(continuous_seconds=5.0)),
            'gate-competing': lambda raw, root: self.rewrite_isolation(
                root, lambda events: first_quiet_sample(events).update(competing=[{'pid': 1, 'kind': 'test/build'}])),
            'monitor-competing': lambda raw, root: self.rewrite_isolation(
                root, lambda events: first_monitor_final(events).update(competing=[{'pid': 2, 'kind': 'compiler'}])),
            'missing-monitor': lambda raw, root: self.rewrite_isolation(
                root, lambda events: events.pop(first_monitor_final_index(events))),
            'trial-start-missing': lambda raw, root: self.rewrite_isolation(
                root, lambda events: events.pop(interval_span(events, first_measured_stem())[0])),
            'trial-end-missing': lambda raw, root: self.rewrite_isolation(
                root, lambda events: events.pop(interval_span(events, first_measured_stem())[1])),
            'trial-end-mismatch': lambda raw, root: self.rewrite_isolation(
                root, lambda events: events[interval_span(events, first_measured_stem())[1]].update(
                    trial=second_measured_stem())),
            'trial-order-swapped': lambda raw, root: self.rewrite_isolation(root, swap_first_intervals),
            'nested-trial-start': lambda raw, root: self.rewrite_isolation(
                root, lambda events: events.insert(interval_span(events, first_measured_stem())[0] + 1,
                                                   {'time': 1.0, 'event': 'trial-start',
                                                    'trial': first_measured_stem()})),
            'duplicate-quiet-ready': lambda raw, root: self.rewrite_isolation(
                root, lambda events: events.insert(interval_span(events, first_measured_stem())[1],
                                                   {'time': 1.0, 'event': 'quiet-ready'})),
            'duplicate-monitor-final': lambda raw, root: self.rewrite_isolation(
                root, lambda events: events.insert(interval_span(events, first_measured_stem())[1],
                                                   {'time': 1.0, 'event': 'monitor-final', 'competing': []})),
            'monitor-before-gate': lambda raw, root: self.rewrite_isolation(
                root, lambda events: events.insert(interval_span(events, first_measured_stem())[0] + 1,
                                                   events.pop(first_monitor_final_index(events)))),
            'monitor-after-final': lambda raw, root: self.rewrite_isolation(
                root, lambda events: events.insert(interval_span(events, first_measured_stem())[1],
                                                   {'time': 1.0, 'event': 'monitor', 'competing': []})),
            'sample-after-ready': lambda raw, root: self.rewrite_isolation(
                root, lambda events: events.insert(interval_span(events, first_measured_stem())[1],
                                                   dict(first_quiet_sample(events)))),
            'stray-monitor-final': lambda raw, root: self.rewrite_isolation(
                root, lambda events: events.append({'time': 1.0, 'event': 'monitor-final', 'competing': []})),
            'stray-quiet-ready': lambda raw, root: self.rewrite_isolation(
                root, lambda events: events.insert(interval_span(events, first_measured_stem())[1] + 1,
                                                   {'time': 1.0, 'event': 'quiet-ready'})),
            'quiet-ready-in-trace': lambda raw, root: self.rewrite_isolation(
                root, lambda events: events.insert(interval_span(events, 'trace-flat-100-00-A')[1],
                                                   {'time': 1.0, 'event': 'quiet-ready'})),
            'isolation-threshold': lambda raw, root: self.rewrite_manifest(
                root, lambda m: m.update(isolation={'minimum_idle_percent': 50, 'continuous_seconds': 5})),
            'neighbor-size': lambda raw, root: rewrite(
                raw, 'trial-flat-10-01-A', lambda r: r['neighbors'][0]['parameters'].update(size=100)),
            'neighbor-contract-drift': lambda raw, root: rewrite(
                raw, 'trial-flat-100-02-B', lambda r: r['neighbors'][0].update(unit='ms/op')),
            'neighbor-unit-absolute': lambda raw, root: self.rewrite_fixture_neighbors(
                raw, 'flat-100', lambda case: case.update(unit='ms/op')),
            'neighbor-iterations-absolute': lambda raw, root: self.rewrite_fixture_neighbors(
                raw, 'flat-100', lambda case: case.update(iterations_per_sample=2)),
        }
        for name, mutate in cases.items():
            with self.subTest(name), tempfile.TemporaryDirectory() as tmp:
                root = Path(tmp)
                raw, _ = build_evidence(root)
                mutate(raw, root)
                with self.assertRaises(RuntimeError, msg=name):
                    editor_experiment.validate_trials(root)

    def rewrite_manifest(self, root, mutate):
        path = root / 'manifest.json'
        manifest = json.loads(path.read_text())
        mutate(manifest)
        path.write_text(json.dumps(manifest))

    def rewrite_isolation(self, root, mutate):
        path = root / 'isolation.jsonl'
        events = [json.loads(line) for line in path.read_text().splitlines() if line.strip()]
        mutate(events)
        path.write_text('\n'.join(json.dumps(e) for e in events) + '\n')

    def rewrite_fixture_neighbors(self, raw, fixture, mutate_case):
        for stem, (fix, _) in editor_experiment.expected_trials().items():
            if fix == fixture:
                rewrite(raw, stem, lambda row: [mutate_case(case) for case in row.get('neighbors', [])])

    def test_isolation_observations_preserved(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            build_evidence(root)
            before = (root / 'isolation.jsonl').read_text()
            editor_experiment.validate_trials(root)
            self.assertEqual((root / 'isolation.jsonl').read_text(), before)

    def test_expected_trial_shape(self):
        expected = editor_experiment.expected_trials()
        self.assertEqual(len(expected), 20)
        hundreds = [label for stem, (_, label) in expected.items() if stem.startswith('trial-flat-100')]
        self.assertEqual(''.join(hundreds), 'ABBAABBAABBA')
        for fixture in ('flat-10', 'invoice-compare'):
            labels = [label for stem, (fix, label) in sorted(expected.items()) if fix == fixture]
            self.assertEqual(''.join(labels), 'ABBA')

    def test_harness_identity_stable(self):
        first = editor_experiment.harness_identity()
        self.assertEqual(len(first), 64)
        int(first, 16)
        self.assertEqual(first, editor_experiment.harness_identity())


class SummarizeTests(unittest.TestCase):
    def test_statistics_goals_and_separation(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            raw, expected = build_evidence(root)
            # Crafted medians: A 100 ms, B 50 ms; warmup outliers must not leak.
            for stem, (fixture, label) in expected.items():
                if fixture != 'flat-100':
                    continue
                target = 100_000_000 if label == 'A' else 50_000_000

                def set_row(row, target=target):
                    row['samples_ns']['completion-unchanged'] = [target] * BATCHES
                    row['warmups_ns']['completion-unchanged'] = [9_000_000_000_000_000] * WARMUPS

                rewrite(raw, stem, set_row)
            # Spread medians 10..60 ms across the six A trials for range/MAD math.
            a_stems = sorted(stem for stem, (fix, label) in expected.items() if fix == 'flat-100' and label == 'A')
            for i, stem in enumerate(a_stems):
                target = (10 + 10 * i) * 1_000_000

                def set_spread(row, target=target):
                    row['samples_ns']['diagnostics-syntax'] = [target] * BATCHES

                rewrite(raw, stem, set_spread)
            editor_experiment.summarize(root)
            report = json.loads((root / 'summary.json').read_text())
            self.assertEqual(report['contract'], CONTRACT)
            self.assertTrue(report['editor_only'])
            self.assertEqual(report['goals_ms'], {'warm_completion': 100, 'edit_to_completion': 100})
            for record in report['comparisons']:
                self.assertNotIn('p95', json.dumps(record))
            self.assertIn('never p95', report['goals_note'])
            self.assertIn('full-system', report['separation'])
            by_key = {(c['fixture'], c['scenario']): c for c in report['comparisons']}
            warm = by_key[('flat-100', 'completion-unchanged')]
            self.assertEqual(warm['workload'], 'split')
            self.assertEqual(warm['A']['median_ms'], 100)
            self.assertEqual(warm['B']['median_ms'], 50)
            self.assertEqual(warm['saved_ms'], 50)
            self.assertEqual(warm['ratio_B_over_A'], 0.5)
            self.assertEqual(warm['A']['trials'], 6)
            spread = by_key[('flat-100', 'diagnostics-syntax')]
            self.assertEqual(spread['A']['median_ms'], 35)
            self.assertEqual(spread['A']['trial_range_ms'], [10, 60])
            self.assertEqual(spread['A']['trial_mad_ms'], 15)
            neighbor = by_key[('flat-100', 'editor.flat.completion')]
            self.assertEqual(neighbor['workload'], 'neighbor')

    def test_markdown_report_content(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            raw, expected = build_evidence(root)
            for stem, (fixture, label) in expected.items():
                if fixture != 'flat-100':
                    continue
                target = 100_000_000 if label == 'A' else 50_000_000

                def set_row(row, target=target):
                    row['samples_ns']['completion-unchanged'] = [target] * BATCHES

                rewrite(raw, stem, set_row)
            editor_experiment.summarize(root)
            text = (root / 'report.md').read_text()
            self.assertIn('# Editor-only responsiveness report', text)
            self.assertIn(CONTRACT, text)
            self.assertIn('warm_completion', text)
            self.assertIn('never p95', text)
            self.assertIn('full-system', text)
            self.assertIn('Timing boundary:', text)
            self.assertIn('client monotonic clock', text)
            self.assertIn('Scope: 20 accepted trials', text)
            self.assertIn('| Fixture | Scenario | Workload |', text)
            self.assertIn('| flat-100 | completion-unchanged | split | 100.0', text)
            self.assertIn('| 50.0 | 50.0-50.0 |', text)
            self.assertIn('| 50.0 | 0.5 |', text)
            table = [line for line in text.splitlines() if line.startswith('|')]
            # Markdown requires the delimiter row to match the header width.
            self.assertEqual({len(line.strip('|').split('|')) for line in table}, {13})


if __name__ == '__main__':
    unittest.main()
