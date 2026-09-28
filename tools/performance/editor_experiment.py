#!/usr/bin/env python3
"""Bounded, sequential editor comparison, reusing qualified local builds.

Session mode prepares and traces A, waits for the investigator's candidate
command on stdin, prepares B, then measures ABBAABBAABBA (six processes each).
The source checkout is never copied. Builds are fingerprinted before and after
preparation. All execution files retire through the standard scratch owner.
This editor-only contract is separate from the twelve-slice audit.
"""
import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import shutil
import statistics
import subprocess
import sys
import time

import isolation
from isolation import exclusive, execute, quiet_window
import schema
from storage import Scratch, archive_evidence, interruption_signals, reap

HERE = Path(__file__).resolve().parent
REPO = HERE.parents[1]
spec = importlib.util.spec_from_file_location('editor_driver', HERE / 'drivers/compiler.py')
driver = importlib.util.module_from_spec(spec)
spec.loader.exec_module(driver)
CONTRACT = 'editor-responsiveness-v1'
SCRATCH_LIMIT = 2 << 30
EVIDENCE_LIMIT = 10 << 20
# Split editor workloads; distinct from the historical editor.flat.diagnostics
# mixed case, which neighbors retain for regression only.
SCENARIOS = ('completion-unchanged', 'diagnostics-syntax', 'diagnostics-semantic',
             'edit-to-completion', 'completion-after-edit', 'recovery-to-completion')
BATCHES, WARMUPS = 7, 2
ORDER_100 = 'ABBAABBAABBA'
ORDER_SMALL = 'ABBA'
SIZE_BY_FIXTURE = {'flat-10': 10, 'flat-100': 100, 'invoice-compare': 100}
NEIGHBOR_CASES = {f'editor.flat.{name}' for name in
                  ('diagnostics', 'hover', 'definition', 'formatting', 'completion', 'rename')}
GOALS_MS = {'warm_completion': 100, 'edit_to_completion': 100}
# Bounded qualification must resolve through production driver.Resolve, which
# verifies the stamped manifest hash, required assets, pinned runtime and every
# manifest file hash. `version` returns before Resolve and proves nothing.
QUALIFICATION_COMMAND = 'runtime-check'


def save(path, value):
    path.write_text(json.dumps(value, indent=2, allow_nan=False) + '\n')


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def source_identity():
    paths = subprocess.check_output(['git', 'ls-files', '-z', '--cached', '--others', '--exclude-standard'], cwd=REPO).split(b'\0')
    inventory = {os.fsdecode(p): sha(REPO / os.fsdecode(p)) for p in paths if p and (REPO / os.fsdecode(p)).is_file()}
    return driver.input_identity(inventory)


def harness_identity():
    """Identity of the measurement harness itself; rows must match the manifest."""
    files = [Path(__file__), HERE / 'drivers/compiler.py', HERE / 'isolation.py', HERE / 'storage.py', HERE / 'schema.py']
    return hashlib.sha256(''.join(sha(p) for p in files).encode()).hexdigest()


def source_patch():
    """Tracked diff plus compact diffs for new authored files."""
    patch = subprocess.check_output(['git', 'diff', '--binary', 'HEAD'], cwd=REPO)
    others = subprocess.check_output(['git', 'ls-files', '--others', '--exclude-standard', '-z'], cwd=REPO).split(b'\0')
    for name in others:
        if not name:
            continue
        rel = os.fsdecode(name)
        if not (REPO / rel).is_file():
            continue
        diff = subprocess.run(['git', 'diff', '--no-index', '--binary', '/dev/null', rel],
                              cwd=REPO, capture_output=True)
        patch += diff.stdout
    return patch


def untracked_files():
    others = subprocess.check_output(['git', 'ls-files', '--others', '--exclude-standard', '-z'], cwd=REPO).split(b'\0')
    return sorted(os.fsdecode(p) for p in others if p and (REPO / os.fsdecode(p)).is_file())


def validate_args(args):
    """Resolve public paths and reject bad sampling before any work starts."""
    args.output = args.output.resolve()
    if args.trial:
        for field in ('project', 'executable', 'oracle'):
            value = getattr(args, field)
            if value is None:
                raise ValueError(f'--trial requires --{field.replace("_", "-")}')
            setattr(args, field, value.resolve())
        if args.trace is not None:
            args.trace = args.trace.resolve()
        if args.fixture not in SIZE_BY_FIXTURE:
            raise ValueError(f'unknown fixture {args.fixture!r}; want one of {sorted(SIZE_BY_FIXTURE)}')
        if args.size != SIZE_BY_FIXTURE[args.fixture]:
            raise ValueError(f'size {args.size} does not match fixture {args.fixture}')
        if not args.project.is_dir():
            raise ValueError(f'project is not a directory: {args.project}')
        if not args.executable.is_file():
            raise ValueError(f'executable is not a file: {args.executable}')
    else:
        args.archive = Path(args.archive).resolve()
        if not args.archive.is_file():
            raise ValueError(f'local pinned archive missing: {args.archive} (no download is performed)')
        if args.wait_seconds <= 0:
            raise ValueError('wait-seconds must be positive')


def focused(args, directory, executable, oracle_path, batches, warmups):
    """Split syntax/semantic clocks stop at response receipt; oracles run after."""
    anchor = args.fixture == 'invoice-compare'
    file = directory / ('src/model/model.can' if anchor else 'src/main.can')
    text, uri = file.read_text(), file.as_uri()
    if anchor:
        line = next(i for i, v in enumerate(text.splitlines()) if 'call make_row(' in v)
        column = text.splitlines()[line].index('make_row') + 1
        modified = text.replace('ok rows[index].key', 'ok rows[index].sku')
    else:
        line = next(i for i, v in enumerate(text.splitlines()) if 'call f0()' in v)
        column = 14
        modified = text.replace('\n    ok 0\n', '\n    ok 1\n')
    if modified == text:
        raise RuntimeError('valid-edit fixture did not change')
    if oracle_path.exists():
        try:
            expected = json.loads(oracle_path.read_text())
        except ValueError as error:
            raise RuntimeError(f'baseline oracle is not valid JSON: {error}') from error
        if not isinstance(expected, list) or not expected:
            raise RuntimeError('baseline oracle is not a non-empty completion array')
    else:
        expected = None
    server = driver.LSP(executable, REPO)
    version, identity = 1, 100
    samples, excluded = ({n: [] for n in SCENARIOS} for _ in range(2))

    def recv_diagnostics():
        return server.until(lambda m: m.get('method') == 'textDocument/publishDiagnostics' and m['params'].get('uri') == uri and m['params'].get('version') == version)

    def check_diagnostics(response, expected_error=False):
        if bool(response['params']['diagnostics']) != expected_error:
            raise RuntimeError('focused versioned diagnostics oracle failed')

    def diagnose(expected_error=False):
        check_diagnostics(recv_diagnostics(), expected_error)

    def change(value):
        nonlocal version
        version += 1
        server.send('textDocument/didChange', {'textDocument': {'uri': uri, 'version': version}, 'contentChanges': [{'text': value}]})

    def completion():
        nonlocal identity
        identity += 1
        server.send('textDocument/completion', {'textDocument': {'uri': uri}, 'position': {'line': line, 'character': column}}, identity)

    def recv_result():
        return server.until(lambda m: m.get('id') == identity).get('result')

    def check_result(response):
        nonlocal expected
        if not isinstance(response, list) or not response:
            raise RuntimeError('empty/declined completion is not a successful sample')
        if not anchor:
            driver.validate_completion(response, args.size)
        if expected is None:
            expected = response
            save(oracle_path, expected)
        if response != expected:
            raise RuntimeError('complete completion array differs from baseline oracle')

    def result():
        check_result(recv_result())

    def measure(target, name, timed, validate):
        started = time.perf_counter_ns()
        outcome = timed()
        elapsed = time.perf_counter_ns() - started
        validate(outcome)
        target[name].append(elapsed)

    try:
        server.send('initialize', {'rootUri': directory.as_uri()}, 1)
        server.until(lambda m: m.get('id') == 1)
        server.send('initialized', {})
        server.send('textDocument/didOpen', {'textDocument': {'uri': uri, 'languageId': 'can', 'version': version, 'text': text}})
        diagnose()
        completion()
        result()  # Exact full-array oracle before any sample.
        for batch in range(warmups + batches):
            target = excluded if batch < warmups else samples
            measure(target, 'completion-unchanged',
                    lambda: (completion(), recv_result())[1],
                    check_result)
            measure(target, 'diagnostics-syntax',
                    lambda: (change(text + '\n@\n'), recv_diagnostics())[1],
                    lambda response: check_diagnostics(response, True))
            measure(target, 'diagnostics-semantic',
                    lambda: (change(modified), recv_diagnostics())[1],
                    check_diagnostics)
            # The notification and query are written back-to-back. The clock
            # runs from the first frame write through receipt of both the
            # versioned diagnostics and the completion result; both oracles
            # are checked after the clock stops.
            measure(target, 'edit-to-completion',
                    lambda: (change(text), completion(), recv_diagnostics(), recv_result())[2:],
                    lambda outcome: (check_diagnostics(outcome[0]), check_result(outcome[1])))
            measure(target, 'completion-after-edit',
                    lambda: (completion(), recv_result())[1],
                    check_result)
            change(text + '\n@\n')
            diagnose(True)
            measure(target, 'recovery-to-completion',
                    lambda: (change(text), completion(), recv_diagnostics(), recv_result())[2:],
                    lambda outcome: (check_diagnostics(outcome[0]), check_result(outcome[1])))
    finally:
        server.close()
    return {'contract': CONTRACT, 'fixture': args.fixture, 'size': args.size,
            'input_content_sha256': driver.input_identity(driver.input_inventory(directory)),
            'oracle_sha256': sha(oracle_path), 'harness_sha256': harness_identity(),
            'samples_ns': samples, 'warmups_ns': excluded,
            'correctness': 'exact entire completion array; authored flat signatures/kinds/provenance; versioned syntax errors and clean valid edits',
            'timing': 'client monotonic clock from first frame write through matching response receipt; all semantic oracle validation after clock stop; edit clocks receive both versioned diagnostics and completion before stopping; initialization excluded'}


def worker(args):
    validate_args(args)
    batches, warmups = (1, 0) if args.trace else (BATCHES, WARMUPS)
    if args.trace:
        os.environ['CAN_LSP_TRACE'] = str(args.trace)
    else:
        os.environ.pop('CAN_LSP_TRACE', None)
    exe_before = sha(args.executable)
    fixture_before = driver.input_identity(driver.input_inventory(args.project))
    harness_before = harness_identity()
    result = focused(args, args.project, args.executable, args.oracle, batches, warmups)
    if args.neighbors:
        namespace = argparse.Namespace(repo=REPO, size=args.size, warmups=warmups, iterations=1, suite='editor')
        result['neighbors'] = driver.editor(namespace, {'bundle': str(args.executable.parent.parent)}, args.project, batches)
    exe_after = sha(args.executable)
    fixture_after = driver.input_identity(driver.input_inventory(args.project))
    harness_after = harness_identity()
    if exe_before != exe_after:
        raise RuntimeError('executable changed during trial exchange')
    if fixture_before != fixture_after:
        raise RuntimeError('fixture inputs changed during trial exchange')
    if harness_before != harness_after:
        raise RuntimeError('harness changed during trial exchange')
    if result.get('input_content_sha256') != fixture_after:
        raise RuntimeError('worker fixture stamp mismatches observed inputs')
    if result.get('harness_sha256') != harness_after:
        raise RuntimeError('worker harness stamp mismatches observed harness')
    result['executable_sha256'] = exe_after
    result['executable_sha256_before'] = exe_before
    save(args.output, result)


def expected_trials():
    """Full ordered evidence: 12 size-100 plus 4+4 verifications."""
    expected = {}
    for index, label in enumerate(ORDER_100, 1):
        expected[f'trial-flat-100-{index:02d}-{label}'] = ('flat-100', label)
    for fixture in ('flat-10', 'invoice-compare'):
        for index, label in enumerate(ORDER_SMALL, 1):
            expected[f'trial-{fixture}-{index:02d}-{label}'] = (fixture, label)
    return expected


def check_runtime_report(output):
    """Validate the bounded qualification tool report; raises on mismatch."""
    try:
        report = json.loads(output)
    except ValueError:
        raise RuntimeError(f'qualification output is not JSON: {output[:200]!r}')
    if report.get('kind') != 'can.development-runtime-check' or report.get('schemaVersion') != 1:
        raise RuntimeError(f'qualification report mismatch: {output[:500]!r}')
    return report


def verify_closed_bundle_inventory(executable):
    """Narrow closed-inventory adapter mirroring distribution.VerifyBundle.

    Production driver.Resolve (exercised by `runtime-check`) verifies the
    stamped manifest hash, required assets, pinned runtime and every manifest
    file hash, but does not reject unknown files. This adapter closes exactly
    that gap: every entry under the bundle root must be manifest-listed (or
    the manifest/launcher pair itself); symlinks and non-regular entries
    anywhere are rejected, as is a launcher missing the manifest stamp.
    Raises on any violation; returns the manifest digest on success.
    """
    executable = Path(executable)
    if executable.name != 'canlc' or executable.parent.name != 'bin':
        raise RuntimeError(f'qualification path is not a bundle launcher: {executable}')
    root = executable.parent.parent
    if root.is_symlink() or not root.is_dir():
        raise RuntimeError(f'qualification root is not a directory: {root}')
    manifest_path = root / 'manifest.json'
    if manifest_path.is_symlink() or not manifest_path.is_file():
        raise RuntimeError('bundle manifest is missing or not a regular file')
    manifest_bytes = manifest_path.read_bytes()
    try:
        manifest = json.loads(manifest_bytes)
    except ValueError as error:
        raise RuntimeError(f'bundle manifest is not valid JSON: {error}')
    files = manifest.get('files')
    if (manifest.get('schemaVersion') != 1 or manifest.get('kind') != 'can.development-distribution'
            or not isinstance(files, dict)
            or any(not isinstance(name, str) or not isinstance(digest, str) for name, digest in files.items())):
        raise RuntimeError('bundle manifest has an unexpected closed-inventory shape')
    known = set(files) | {'manifest.json', 'bin/canlc'}
    stack = [root]
    while stack:
        directory = stack.pop()
        with os.scandir(directory) as entries:
            for entry in entries:
                if entry.is_symlink():
                    raise RuntimeError(f'bundle holds a symlinked entry: {entry.path}')
                if entry.is_dir(follow_symlinks=False):
                    stack.append(Path(entry.path))
                elif entry.is_file(follow_symlinks=False):
                    rel = Path(entry.path).relative_to(root).as_posix()
                    if rel not in known:
                        raise RuntimeError(f'bundle holds an unknown file: {rel}')
                else:
                    raise RuntimeError(f'bundle holds a non-regular entry: {entry.path}')
    digest = hashlib.sha256(manifest_bytes).hexdigest()
    if digest.encode() not in (root / 'bin' / 'canlc').read_bytes():
        raise RuntimeError('bundle launcher stamp does not match manifest')
    return digest


def qualify(label, executable, run):
    """Run production Resolve plus the bounded tool, then closed inventory.

    Called during preparation, outside all trial clocks, before any trial.
    `runtime-check` exercises production driver.Resolve (stamped manifest
    hash, required assets, pinned runtime, every manifest file hash); the
    closed-inventory adapter then rejects unknown files, symlinks and
    non-regular entries, which Resolve does not cover. A nonzero tool exit,
    a mismatched report or an open inventory aborts the session so no trial
    can proceed on an unqualified bundle.
    """
    output = run([str(executable), QUALIFICATION_COMMAND], 'qualify-' + label, timeout=30)
    report = check_runtime_report(output)
    verify_closed_bundle_inventory(executable)
    return report


def check_samples(values, count, what):
    if not isinstance(values, list) or len(values) != count:
        raise RuntimeError(f'{what}: want exactly {count} samples, got {len(values) if isinstance(values, list) else type(values).__name__}')
    if any(not schema.number(v) or v < 0 for v in values):
        raise RuntimeError(f'{what}: samples must be finite nonnegative numbers')


def validate_neighbors(row, name):
    neighbors = row.get('neighbors')
    if row['fixture'] == 'invoice-compare':
        if 'neighbors' in row:
            raise RuntimeError(f'{name}: invoice-compare carries no neighbors')
        return
    if not isinstance(neighbors, list) or {c.get('name') for c in neighbors} != NEIGHBOR_CASES:
        raise RuntimeError(f'{name}: neighbors must be exactly the six historical editor.flat cases')
    try:
        schema.validate_result({'schema_version': 1, 'suite': 'editor', 'status': 'complete', 'cases': neighbors}, 'editor')
    except ValueError as error:
        raise RuntimeError(f'{name}: neighbor evidence invalid: {error}')
    for case in neighbors:
        if case.get('unit') != 'ns/op':
            raise RuntimeError(f"{name}: neighbor {case.get('name')} unit {case.get('unit')!r} must be 'ns/op' (the report converts ns to ms)")
        if case.get('iterations_per_sample') != 1:
            raise RuntimeError(f"{name}: neighbor {case.get('name')} iterations {case.get('iterations_per_sample')!r} must be 1")
    for case in neighbors:
        # Raw per-request latencies are retained, never a fabricated tail.
        latencies = case.get('metrics', {}).get('request_latencies_ns')
        if (not isinstance(latencies, list) or len(latencies) != len(case['samples']) * case['iterations_per_sample']
                or any(not schema.number(v) or v < 0 for v in latencies)):
            raise RuntimeError(f'{name}: neighbor {case["name"]} lost raw request latencies')
        if len(case['samples']) != BATCHES or len(case.get('warmup_samples', [])) != WARMUPS:
            raise RuntimeError(f'{name}: neighbor {case["name"]} must hold seven measured and two warmup samples')


def validate_step_order(manifest, expected):
    """Require exact ordered trial steps with no duplicates or reordering."""
    steps_list = manifest.get('steps', [])
    names = [s.get('name') for s in steps_list]
    if len(names) != len(set(names)):
        seen, dups = set(), set()
        for name in names:
            if name in seen:
                dups.add(name)
            seen.add(name)
        raise RuntimeError(f'duplicated manifest steps: {sorted(dups)}')
    trial_steps = [n for n in names if isinstance(n, str) and n.startswith('trial-')]
    expected_order = list(expected.keys())
    if trial_steps != expected_order:
        raise RuntimeError(f'trial step order mismatch: want {expected_order}, got {trial_steps}')
    return {s['name']: s for s in steps_list}


def validate_isolation_gates(output, expected_order):
    """Bind each accepted trial to the gate inside its own labeled interval.

    Reads the preserved raw isolation observations without modifying them.
    Every trial emits labeled trial-start/trial-end observations around its
    own quiet gate and monitored execution. Measurement intervals must appear
    in exact expected order, each holding exactly one quiet-ready (whose
    preceding quiet-sample in the same interval shows 30s continuous, 90%
    idle, no competing jobs) and exactly one clean monitor-final. Mismatched
    labels, missing/duplicate markers or finals, quiet gates inside
    non-measurement intervals, and gate/monitor events outside any labeled
    interval are all rejected. Successful process exit stays bound through
    manifest steps in validate_trials.
    """
    path = output / 'isolation.jsonl'
    if not path.is_file():
        raise RuntimeError('missing isolation observations; each accepted trial needs a quiet-ready gate')
    events = []
    for line in path.read_text().splitlines():
        if not line.strip():
            continue
        try:
            events.append(json.loads(line))
        except ValueError:
            raise RuntimeError('isolation observations are not valid JSON lines')
    bound = {'quiet-sample', 'quiet-ready', 'monitor', 'monitor-final'}
    intervals = []
    current, body = None, []
    for event in events:
        kind = event.get('event')
        if kind == 'trial-start':
            if current is not None:
                raise RuntimeError(f"{event.get('trial')}: trial-start while {current} interval is still open")
            current = event.get('trial')
            if not current:
                raise RuntimeError('trial-start without a trial label')
            body = []
        elif kind == 'trial-end':
            if current is None:
                raise RuntimeError(f"{event.get('trial')}: trial-end without a trial-start")
            if event.get('trial') != current:
                raise RuntimeError(f"trial-end label {event.get('trial')!r} mismatches open interval {current!r}")
            intervals.append((current, body))
            current, body = None, []
        elif current is not None:
            body.append(event)
        elif kind in bound:
            raise RuntimeError(f'stray {kind} outside any labeled trial interval')
    if current is not None:
        raise RuntimeError(f'{current}: trial interval never closed by trial-end')
    expected_set = set(expected_order)
    measured = [(label, body) for label, body in intervals if label in expected_set]
    if [label for label, _ in measured] != list(expected_order):
        raise RuntimeError(f'labeled trial order mismatch: want {list(expected_order)}, got {[label for label, _ in measured]}')
    for label, body in measured:
        ready_pos = [i for i, event in enumerate(body) if event.get('event') == 'quiet-ready']
        if len(ready_pos) != 1:
            raise RuntimeError(f'{label}: want exactly one quiet-ready in its labeled interval, got {len(ready_pos)}')
        sample = None
        for event in body[:ready_pos[0]]:
            if event.get('event') == 'quiet-sample':
                sample = event
        if sample is None:
            raise RuntimeError(f'{label}: missing quiet-sample before quiet-ready gate')
        if not isinstance(sample.get('idle_percent'), (int, float)) or sample['idle_percent'] < 90:
            raise RuntimeError(f'{label}: quiet gate idle {sample.get("idle_percent")} below 90%')
        if not isinstance(sample.get('continuous_seconds'), (int, float)) or sample['continuous_seconds'] < 30:
            raise RuntimeError(f'{label}: quiet gate continuous {sample.get("continuous_seconds")} below 30s')
        if sample.get('competing'):
            raise RuntimeError(f'{label}: quiet gate observed competing jobs {sample.get("competing")}')
        finals = [event for event in body if event.get('event') == 'monitor-final']
        if len(finals) != 1:
            raise RuntimeError(f'{label}: want exactly one monitor-final in its labeled interval, got {len(finals)}')
        monitor_positions = [i for i, event in enumerate(body)
                             if event.get('event') in ('monitor', 'monitor-final')]
        if any(i <= ready_pos[0] for i in monitor_positions):
            raise RuntimeError(f'{label}: monitoring must follow the quiet-ready gate')
        if body[monitor_positions[-1]].get('event') != 'monitor-final':
            raise RuntimeError(f'{label}: monitoring continued after monitor-final')
        if any(event.get('event') == 'quiet-sample' for event in body[ready_pos[0] + 1:]):
            raise RuntimeError(f'{label}: quiet sampling continued after quiet-ready')
        for observed in body:
            if observed.get('event') in ('monitor', 'monitor-final') and observed.get('competing'):
                raise RuntimeError(f'{label}: competing jobs observed during trial {observed.get("competing")}')
    for label, body in intervals:
        if label not in expected_set and any(event.get('event') == 'quiet-ready' for event in body):
            raise RuntimeError(f'{label}: quiet-ready gate inside a non-measurement interval')


def validate_neighbor_consistency(rows):
    """Require identical neighbor contracts across each fixture's A/B trials."""
    by_fixture = {}
    for row in rows:
        by_fixture.setdefault(row['fixture'], []).append(row)
    for fixture, fixture_rows in by_fixture.items():
        if fixture == 'invoice-compare':
            continue
        contracts = {}
        for row in fixture_rows:
            for case in row.get('neighbors', []):
                name = case.get('name')
                params = case.get('parameters')
                if not isinstance(params, dict) or params.get('size') != SIZE_BY_FIXTURE[fixture]:
                    size = params.get('size') if isinstance(params, dict) else params
                    raise RuntimeError(f'{fixture} neighbor {name} size {size!r} mismatches fixture size {SIZE_BY_FIXTURE[fixture]}')
                key = (case.get('unit'), json.dumps(params, sort_keys=True),
                       case.get('timing_scope'), case.get('iterations_per_sample'))
                contracts.setdefault(name, set()).add(key)
        for name, keys in contracts.items():
            if len(keys) != 1:
                raise RuntimeError(f'{fixture} neighbor {name} contracts differ across A/B trials')


def validate_trials(output):
    """Admit only a complete, compatible, untraced comparison. Raises otherwise."""
    raw = output / 'raw'
    manifest = json.loads((output / 'manifest.json').read_text())
    if manifest.get('contract') != CONTRACT:
        raise RuntimeError('manifest contract mismatch')
    if manifest.get('isolation') != {'minimum_idle_percent': 90, 'continuous_seconds': 30}:
        raise RuntimeError(f'manifest isolation gate mismatch: {manifest.get("isolation")!r}')
    expected = expected_trials()
    found = {p.stem for p in raw.glob('trial-*.json')}
    if found != set(expected):
        raise RuntimeError(f'incomplete trial evidence: missing {sorted(set(expected) - found)}, unexpected {sorted(found - set(expected))}')
    steps = validate_step_order(manifest, expected)
    validate_isolation_gates(output, list(expected.keys()))
    rows = []
    for stem in expected:
        fixture, label = expected[stem]
        row = json.loads((raw / (stem + '.json')).read_text())
        if row.get('contract') != CONTRACT:
            raise RuntimeError(f'{stem}: contract mismatch')
        if row.get('quality') != 'measurement':
            raise RuntimeError(f'{stem}: trace-labeled trial cannot enter an accepted comparison')
        if (row.get('fixture'), row.get('size'), row.get('variant')) != (fixture, SIZE_BY_FIXTURE[fixture], label):
            raise RuntimeError(f'{stem}: fixture/size/variant disagree with filename')
        if row.get('input_content_sha256') != manifest.get('fixtures', {}).get(fixture):
            raise RuntimeError(f'{stem}: fixture inputs differ from prepared identity')
        if row.get('harness_sha256') != manifest.get('harness_sha256'):
            raise RuntimeError(f'{stem}: harness context differs from manifest')
        if row.get('executable_sha256') != manifest.get('builds', {}).get(label, {}).get('executable_sha256'):
            raise RuntimeError(f'{stem}: executable is not the qualified {label} bundle')
        if row.get('executable_sha256_before') != manifest.get('builds', {}).get(label, {}).get('executable_sha256'):
            raise RuntimeError(f'{stem}: worker pre-exchange executable is not the qualified {label} bundle')
        oracle_path = raw / (fixture + '.oracle.json')
        oracle = json.loads(oracle_path.read_text())
        if not isinstance(oracle, list) or not oracle:
            raise RuntimeError(f'{stem}: oracle is not a complete completion array')
        if row.get('oracle_sha256') != sha(oracle_path):
            raise RuntimeError(f'{stem}: oracle link broken')
        if set(row.get('samples_ns', {})) != set(SCENARIOS) or set(row.get('warmups_ns', {})) != set(SCENARIOS):
            raise RuntimeError(f'{stem}: want exactly the six split scenarios in samples and warmups')
        for name in SCENARIOS:
            check_samples(row['samples_ns'][name], BATCHES, f'{stem} {name}')
            check_samples(row['warmups_ns'][name], WARMUPS, f'{stem} warmup {name}')
        validate_neighbors(row, stem)
        step = steps.get(stem)
        if step is None or step.get('returncode') != 0:
            raise RuntimeError(f'{stem}: no successful process exit recorded')
        if not row.get('correctness') or not row.get('timing'):
            raise RuntimeError(f'{stem}: correctness/timing boundary missing')
        rows.append(row)
    validate_neighbor_consistency(rows)
    for path in sorted(raw.glob('trace-*.json')):
        row = json.loads(path.read_text())
        if row.get('quality') != 'functional-trace' or row.get('contract') != CONTRACT:
            raise RuntimeError(f'{path.stem}: mislabeled trace trial')
    return rows


def render_editor_report(summary, rows):
    """Human-readable report from verified records; Markdown is sufficient."""
    timings = sorted({row.get('timing', '') for row in rows})
    correctness = sorted({row.get('correctness', '') for row in rows})
    fixtures = sorted({row.get('fixture', '') for row in rows})
    lines = ['# Editor-only responsiveness report', '',
             f"Contract: {summary['contract']} (editor-only: {summary['editor_only']})",
             f"Goals (ms): {json.dumps(summary['goals_ms'], sort_keys=True)}",
             f"Goals note: {summary['goals_note']}",
             f"Separation: {summary['separation']}",
             f"Timing boundary: {' | '.join(timings)}",
             f"Correctness: {' | '.join(correctness)}",
             f"Scope: {len(rows)} accepted trials across {', '.join(fixtures)}; "
             'seven measured plus two warmup batches per trial; split workloads plus historical neighbors',
             '', '## Comparisons (medians over trial medians; ms)', '',
             '| Fixture | Scenario | Workload | A median | A range | A MAD | A trials | '
             'B median | B range | B MAD | B trials | Saved | Ratio B/A |',
             '|---|---|---|---:|---|---:|---:|---:|---|---:|---:|---:|---:|']
    for record in summary['comparisons']:
        a, b = record['A'], record['B']
        lines.append(f"| {record['fixture']} | {record['scenario']} | {record['workload']} | "
                     f"{a['median_ms']} | {a['trial_range_ms'][0]}-{a['trial_range_ms'][1]} | {a['trial_mad_ms']} | {a['trials']} | "
                     f"{b['median_ms']} | {b['trial_range_ms'][0]}-{b['trial_range_ms'][1]} | {b['trial_mad_ms']} | {b['trials']} | "
                     f"{record['saved_ms']} | {record['ratio_B_over_A']} |")
    lines.append('')
    return '\n'.join(lines) + '\n'


def summarize(output):
    rows = validate_trials(output)
    groups = {}
    for row in rows:
        label, fixture = row['variant'], row['fixture']
        for name, values in row['samples_ns'].items():
            groups.setdefault((fixture, name, label, 'split'), []).append(statistics.median(values) / 1e6)
        for case in row.get('neighbors', []):
            groups.setdefault((fixture, case['name'], label, 'neighbor'), []).append(statistics.median(case['samples']) / 1e6)
    comparisons = []
    for fixture, name, workload in sorted({(f, n, w) for f, n, _, w in groups}):
        record = {'fixture': fixture, 'scenario': name, 'workload': workload}
        for label in ('A', 'B'):
            values = groups[fixture, name, label, workload]
            center = statistics.median(values)
            record[label] = {'median_ms': center, 'trial_range_ms': [min(values), max(values)],
                             'trial_mad_ms': statistics.median(abs(v - center) for v in values), 'trials': len(values)}
        record['saved_ms'] = record['A']['median_ms'] - record['B']['median_ms']
        record['ratio_B_over_A'] = record['B']['median_ms'] / record['A']['median_ms']
        comparisons.append(record)
    summary = {'contract': CONTRACT, 'editor_only': True, 'goals_ms': GOALS_MS,
               'goals_note': 'provisional engineering goals; seven measured batches per trial support medians/ranges/MAD only, never p95',
               'separation': 'editor-only; do not mix with the full-system 20260927 run or historical mixed-diagnostic medians',
               'comparisons': comparisons}
    save(output / 'summary.json', summary)
    (output / 'report.md').write_text(render_editor_report(summary, rows))


def session(args):
    validate_args(args)
    output = args.output
    for record in reap(output.parent):
        print('Recovery:', json.dumps(record), flush=True)
    output.mkdir()  # Refuse replacement of evidence.
    scratch = Scratch(output)
    manifest = None
    raw = work = None
    evidence = output / 'isolation.jsonl'
    try:
        manifest = {'contract': CONTRACT, 'status': 'running', 'baseline_revision': subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=REPO, text=True).strip(),
                    'sampling': {'trials_per_variant': 6, 'batches': BATCHES, 'excluded_warmups': WARMUPS, 'iterations': 1},
                    'isolation': {'minimum_idle_percent': 90, 'continuous_seconds': 30}, 'steps': [], 'builds': {}, 'fixtures': {},
                    'harness_sha256': harness_identity(),
                    'preparation': {'fixtures': ['flat-10', 'flat-100', 'invoice-compare'], 'builds': ['A', 'B'],
                                    'note': 'narrow editor-only scope; no flat-1000 or utilities anchor'},
                    'trace_policy': 'trace trials reuse the qualified bundle and are functional-only; accepted trials run with CAN_LSP_TRACE unset',
                    'limits_bytes': {'scratch': SCRATCH_LIMIT, 'trace': 32 << 20, 'compressed_evidence': EVIDENCE_LIMIT}}
        save(output / 'manifest.json', manifest)
        raw, work = output / 'raw', output / 'work'
        raw.mkdir(); work.mkdir()
        environment = {k: v for k, v in os.environ.items() if k in {'PATH', 'HOME', 'USER', 'LOGNAME', 'SHELL', 'LANG', 'LC_ALL', 'TZ', 'TMPDIR'}}
        temporary = work / 'tmp'; temporary.mkdir()
        environment.update({'GOTMPDIR': str(temporary), 'TMPDIR': str(temporary), 'GOFLAGS': '-p=2', 'GOMAXPROCS': '2', 'GOPROXY': 'off', 'GOSUMDB': 'off', 'GOTOOLCHAIN': 'local', 'CAN_PERF_SUPERVISED': '1'})
        isolation.PROCESS_OBSERVER = scratch.child

        def budget():
            total = sum(p.stat().st_size for p in work.rglob('*') if p.is_file())
            if total > SCRATCH_LIMIT:
                raise RuntimeError('scratch exceeds initial 2 GiB cap')
            manifest['peak_observed_scratch_bytes'] = max(total, manifest.get('peak_observed_scratch_bytes', 0))
            if sum(p.stat().st_size for p in raw.glob('*.trace.jsonl')) > 32 << 20:
                raise RuntimeError('combined traces exceed 32 MiB cap')

        def run(command, name, timeout=900, monitor=False):
            budget()
            code, elapsed = execute(command, REPO, environment, raw / (name + '.stdout.log'), raw / (name + '.stderr.log'), timeout, evidence, monitor)
            manifest['steps'].append({'name': name, 'returncode': code, 'elapsed_seconds': elapsed})
            save(output / 'manifest.json', manifest)
            budget()
            if code:
                raise RuntimeError(f'{name} failed: {(raw / (name + ".stderr.log")).read_text()[-2000:]}')
            return (raw / (name + '.stdout.log')).read_text().strip()

        def build(label):
            # Estimated peak: two ~150 MiB bundles + bounded two-worker Go scratch,
            # no source/dependency copies and the already installed shared Go cache.
            if shutil.disk_usage(work).free < 3 << 30:
                raise RuntimeError('less than 3 GiB free; preparation cannot fit safely')
            before = source_identity()
            (raw / (label + '.patch')).write_bytes(source_patch())
            bundle = Path(run([str(work / 'distbuild'), '--archive', args.archive, '--source', str(REPO), '--out', str(work / 'bundles'), '--version', 'editor-' + label], 'build-' + label))
            if source_identity() != before:
                raise RuntimeError('source changed during qualified build')
            executable = bundle / 'bin/canlc'
            # Qualification exercises production driver.Resolve (stamped manifest
            # hash, required assets, pinned runtime, every manifest file hash) plus
            # the bounded sidecar tool. `catalogue-check` would also resolve but
            # currently fails on a stale 290-operation assertion (inventory is 299).
            qualify(label, executable, run)
            manifest['builds'][label] = {'source_content_sha256': before, 'executable_sha256': sha(executable),
                                         'manifest_sha256': sha(bundle / 'manifest.json'), 'untracked': untracked_files(),
                                         'qualification': QUALIFICATION_COMMAND + '+closed-inventory'}
            save(output / 'manifest.json', manifest)
            return executable

        def trial(label, executable, fixture, index, trace=False, neighbors=False):
            size = 10 if fixture == 'flat-10' else 100
            name = ('trace' if trace else 'trial') + f'-{fixture}-{index:02d}-{label}'
            destination = raw / (name + '.json')
            command = [sys.executable, str(__file__), '--trial', '--project', str(work / fixture), '--executable', str(executable), '--oracle', str(raw / (fixture + '.oracle.json')), '--fixture', fixture, '--size', str(size), '--output', str(destination)]
            if trace:
                command += ['--trace', str(raw / (name + '.trace.jsonl'))]
            if neighbors:
                command += ['--neighbors']
            isolation.log(evidence, 'trial-start', trial=name)
            if not trace:
                quiet_window(30, 90, args.wait_seconds, evidence)
            run(command, name, timeout=300, monitor=True)
            isolation.log(evidence, 'trial-end', trial=name)
            row = json.loads(destination.read_text())
            qualified = sha(executable)
            if row.get('executable_sha256') != qualified or row.get('executable_sha256_before') != qualified:
                raise RuntimeError(f'{name}: worker executable stamp mismatches qualified {label} bundle')
            row['variant'] = label
            row['quality'] = 'functional-trace' if trace else 'measurement'
            save(destination, row)

        with exclusive(args.wait_seconds, evidence):
            if isolation.competing():
                raise RuntimeError('competing build/test work; preparation deferred')
            manifest['host'] = {'platform': sys.platform, 'cpu_count': os.cpu_count(), 'go_version': subprocess.check_output(['go', 'version'], text=True).strip(), 'initial_free_bytes': shutil.disk_usage(work).free}
            manifest['tools'] = {'bun_archive': str(args.archive), 'bun_archive_sha256': sha(args.archive)}
            run(['go', 'build', '-o', str(work / 'distbuild'), './tools/distbuild'], 'build-preparer')
            for size in (10, 100):
                driver.fixture(work / f'flat-{size}', size)
            shutil.copytree(REPO / 'examples/invoice-compare', work / 'invoice-compare', ignore=shutil.ignore_patterns('dist', '.git', 'prepared.json'))
            for name in ('flat-10', 'flat-100', 'invoice-compare'):
                manifest['fixtures'][name] = driver.input_identity(driver.input_inventory(work / name))
            save(output / 'manifest.json', manifest)
            baseline = build('A')
            trial('A', baseline, 'flat-100', 0, trace=True)
            print('BASELINE_TRACE_READY. Send candidate on stdin after choosing and implementing the fix.', flush=True)
            if sys.stdin.readline().strip() != 'candidate':
                raise RuntimeError('investigation stopped before candidate')
            candidate = build('B')
            trial('B', candidate, 'flat-100', 0, trace=True)
            for index, label in enumerate('ABBAABBAABBA', 1):
                print(f'Measuring {index}/12: {label} flat-100', flush=True)
                trial(label, baseline if label == 'A' else candidate, 'flat-100', index, neighbors=True)
            for fixture in ('flat-10', 'invoice-compare'):
                for index, label in enumerate('ABBA', 1):
                    print(f'Verifying {fixture}: {label} {index}/4', flush=True)
                    trial(label, baseline if label == 'A' else candidate, fixture, index, neighbors=fixture == 'flat-10')
            summarize(output)
            manifest['status'] = 'complete'
    except BaseException as error:
        if manifest is None:
            manifest = {'contract': CONTRACT, 'status': 'incomplete', 'error': str(error)}
        else:
            manifest['status'], manifest['error'] = 'incomplete', str(error)
        raise
    finally:
        isolation.PROCESS_OBSERVER = None
        try:
            scratch.finish()
            if manifest is not None:
                manifest['cleanup'] = 'verified standard scratch retirement'
        except BaseException as error:
            if manifest is not None:
                manifest['cleanup'] = 'failed: ' + str(error)
            raise
        finally:
            if manifest is not None:
                save(output / 'manifest.json', manifest)
            archive_evidence(output)
            if (output / 'evidence.zip').stat().st_size > EVIDENCE_LIMIT:
                raise RuntimeError('compressed evidence exceeds 10 MiB cap')
            print('Evidence:', output / 'evidence.zip', flush=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--trial', action='store_true')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--archive', default='/tmp/bun-darwin-aarch64.zip')
    parser.add_argument('--wait-seconds', type=float, default=180)
    parser.add_argument('--project', type=Path)
    parser.add_argument('--executable', type=Path)
    parser.add_argument('--oracle', type=Path)
    parser.add_argument('--fixture', default='flat-100')
    parser.add_argument('--size', type=int, default=100)
    parser.add_argument('--trace', type=Path)
    parser.add_argument('--neighbors', action='store_true')
    args = parser.parse_args()
    with interruption_signals():
        worker(args) if args.trial else session(args)


if __name__ == '__main__':
    main()
