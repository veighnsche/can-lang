#!/usr/bin/env python3
"""Compiler, real assertion supervision, artifacts and LSP performance driver.

Preparation builds a fresh hash-locked distribution from a local pinned archive.
Run never builds Go or emits assertion artifacts. Compiler measurements explicitly
execute compiler internals. Sample batches contain --iterations operations; three
quick or seven standard batches are returned, with --warmups excluded batches.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import queue
import shutil
import signal
import subprocess
import sys
import threading
import time

SUITES = {"compiler", "assertions", "artifacts", "editor"}
SIZES = (10, 100, 1000)


class Blocked(Exception):
    pass


def command(argv, repo, env=None, timeout=900):
    own_group = (env if env is not None else os.environ).get("CAN_PERF_SUPERVISED") != "1"
    process = subprocess.Popen([str(a) for a in argv], cwd=repo, env=env,
                               stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                               text=True, start_new_session=own_group)
    try:
        stdout, stderr = process.communicate(timeout=timeout)
    except BaseException:
        if own_group:
            os.killpg(process.pid, signal.SIGKILL)
        else:
            process.kill()
        process.wait(timeout=10)
        raise
    if process.returncode:
        raise RuntimeError(f"command failed ({process.returncode}): {' '.join(map(str, argv))}\n{stderr[-5000:]}")
    return stdout


def fixture(directory, size):
    (directory / "src").mkdir(parents=True, exist_ok=True)
    (directory / "can.project.json").write_text('{"source_root":"src","error_registry":"can.errors.json"}')
    (directory / "can.errors.json").write_text('{"active":[],"retired":[]}')
    text = 'package audit\n    provides [main]\n    uses []\n\nfn void main\n    emits {}\n    given\n        str[] args\n    asserts\n        smoke: [] => ok\n    match chain\n'
    text += ''.join(f'        call f{i}() as int v{i}\n' for i in range(size))
    text += '        ok => ok\n'
    text += ''.join(f'\nfn int f{i}\n    emits {{}}\n    asserts\n        sample: => ok {i}\n    ok {i}\n' for i in range(size))
    (directory / "src/main.can").write_text(text)
    return text


def input_inventory(directory):
    return {str(path.relative_to(directory)): hashlib.sha256(path.read_bytes()).hexdigest()
            for path in sorted(directory.rglob('*'))
            if path.is_file() and path.suffix != '.md' and path.name != 'prepared.json'
            and not {'dist', '.git', '__pycache__'}.intersection(path.relative_to(directory).parts)}


def input_identity(hashes):
    return hashlib.sha256(json.dumps(hashes, sort_keys=True, separators=(',', ':')).encode()).hexdigest()


def prepare(args):
    suites = set(args.suites.split(','))
    if not suites or suites - SUITES:
        raise ValueError(f"unknown suites: {suites - SUITES}")
    if shutil.which("go") is None:
        raise Blocked("Go compiler prerequisite missing from PATH")
    archive = Path(os.environ.get("CAN_PERF_BUN_ARCHIVE", "/tmp/bun-darwin-aarch64.zip"))
    if not archive.is_file():
        raise Blocked("local pinned Bun archive missing; set CAN_PERF_BUN_ARCHIVE (no download is performed)")
    env = os.environ.copy()
    env.setdefault("GOCACHE", str(args.work_dir / "go-cache"))
    env.setdefault("GOTMPDIR", str(args.work_dir / "go-tmp"))
    Path(env["GOCACHE"]).mkdir(parents=True, exist_ok=True)
    Path(env["GOTMPDIR"]).mkdir(parents=True, exist_ok=True)
    distbuild = args.work_dir / "distbuild"
    command(["go", "build", "-o", distbuild, "./tools/distbuild"], args.repo, env)
    bundle = Path(command([distbuild, "--archive", archive, "--source", args.repo,
                           "--out", args.work_dir / "bundles", "--version", "perf-suites"], args.repo, env).strip())
    helper = bundle / "bin/perfmeasure"
    command(["go", "build", "-o", helper, "./compiler/perfmeasure"], args.repo, env)
    digest = hashlib.sha256((bundle / "manifest.json").read_bytes()).hexdigest()
    for size in SIZES:
        directory = args.work_dir / f"flat-{size}"
        fixture(directory, size)
        command([helper, "--mode", "prepare", "--project", directory, "--manifest", digest], args.repo, env)
    anchors = []
    for name, target in (("utilities", "bun"), ("invoice-compare", "browser")):
        original = args.repo / "examples" / name
        destination = args.work_dir / "anchors" / name
        shutil.copytree(original, destination, ignore=shutil.ignore_patterns("dist", "prepared.json", ".git"))
        input_hashes = input_inventory(destination)
        command([helper, "--mode", "prepare", "--project", destination, "--manifest", digest, "--target", target], args.repo, env)
        anchors.append({"name": name, "target": target, "directory": str(destination), "source_file_hashes": input_hashes, "input_content_sha256": input_identity(input_hashes)})
    metadata = {"anchors": anchors, "bundle": str(bundle), "helper": str(helper), "manifest": digest,
                "sizes": list(SIZES), "suites": sorted(suites)}
    (args.work_dir / "prepared-driver.json").write_text(json.dumps(metadata, indent=2))
    return {"suite": "prepare", "status": "complete", "cases": [],
            "artifacts": [str(args.work_dir / "prepared-driver.json")],
            "notes": ["Fresh qualified sidecar built from current source and local hash-pinned archive; no dependencies installed.",
                      "Flat call-chain fixtures contain 10, 100 and 1000 pure functions plus one real attached assertion per function and entry.",
                      "Maintained utilities and invoice-compare anchors retain their original source/manifests/vendor controls and carry stable input-content identities."]}


class LSP:
    """Actual Content-Length JSON-RPC transport, bounded by a reader queue."""
    def __init__(self, executable, repo, request_timeout=30):
        self.request_timeout = request_timeout
        self.process = subprocess.Popen([str(executable), "lsp", "--stdio"], cwd=repo,
                                        stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL)
        self.messages = queue.Queue()
        self.reader = threading.Thread(target=self.read, daemon=True)
        self.reader.start()

    def read(self):
        try:
            while True:
                headers = {}
                while True:
                    line = self.process.stdout.readline()
                    if not line:
                        raise EOFError("LSP closed its output")
                    if line in (b'\r\n', b'\n'):
                        break
                    key, value = line.decode().split(':', 1)
                    headers[key.lower()] = value.strip()
                length = int(headers['content-length'])
                chunks = bytearray()
                while len(chunks) < length:
                    data = self.process.stdout.read(length - len(chunks))
                    if not data:
                        raise EOFError("truncated LSP frame")
                    chunks.extend(data)
                self.messages.put(json.loads(chunks))
        except Exception as error:
            self.messages.put(error)

    def send(self, method, params, identity=None):
        msg = {"jsonrpc": "2.0", "method": method, "params": params}
        if identity is not None:
            msg['id'] = identity
        payload = json.dumps(msg).encode()
        self.process.stdin.write(f"Content-Length: {len(payload)}\r\n\r\n".encode() + payload)
        self.process.stdin.flush()

    def until(self, predicate, allow_errors=False):
        deadline = time.monotonic() + self.request_timeout
        while True:
            message = self.messages.get(timeout=max(0.001, deadline - time.monotonic()))
            if isinstance(message, Exception):
                raise message
            if 'error' in message and not allow_errors:
                raise RuntimeError(f"LSP error: {message['error']}")
            if predicate(message):
                return message
            if time.monotonic() >= deadline:
                raise TimeoutError("LSP response timeout")

    def close(self):
        try:
            if self.process.poll() is None:
                self.send('shutdown', {}, 999999)
                self.until(lambda m: m.get('id') == 999999)
                self.send('exit', {})
                self.process.wait(timeout=3)
        finally:
            if self.process.poll() is None:
                self.process.kill()
                self.process.wait(timeout=3)
            self.process.stdin.close()
            self.process.stdout.close()
            self.reader.join(timeout=3)
        if self.process.returncode != 0:
            raise RuntimeError(f'LSP process exited {self.process.returncode}; trial is invalid')


def expected_rename(uri, call_line, declaration_line):
    return {'changes': {uri: [
        {'range': {'start': {'line': call_line, 'character': 13}, 'end': {'line': call_line, 'character': 15}}, 'newText': 'f_zero'},
        {'range': {'start': {'line': declaration_line, 'character': 7}, 'end': {'line': declaration_line, 'character': 9}}, 'newText': 'f_zero'}]}}


def validate_rename(actual, expected):
    if set(actual) != {'changes'} or set(actual['changes']) != set(expected['changes']):
        raise RuntimeError('rename changed an unexpected document or response field')
    for uri in expected['changes']:
        normalize = lambda edits: sorted(json.dumps(edit, sort_keys=True) for edit in edits)
        if normalize(actual['changes'][uri]) != normalize(expected['changes'][uri]):
            raise RuntimeError(f'rename WorkspaceEdit differs from exact token oracle: {actual}')


def apply_workspace_edits(text, edits):
    lines = text.splitlines(keepends=True)
    offsets = [0]
    for line in lines:
        offsets.append(offsets[-1] + len(line))
    # This fixture is ASCII; UTF-16 columns are therefore Python indices.
    replacements = [(offsets[e['range']['start']['line']] + e['range']['start']['character'],
                     offsets[e['range']['end']['line']] + e['range']['end']['character'], e['newText']) for e in edits]
    for start, end, replacement in sorted(replacements, reverse=True):
        text = text[:start] + replacement + text[end:]
    return text


def expected_hover(name, call_line):
    return {'contents': {'kind': 'markdown', 'value': '```can\nfn int ' + name + '\nemits {}\n```\n\npackage audit'},
            'range': {'start': {'line': call_line, 'character': 13}, 'end': {'line': call_line, 'character': 13 + len(name)}}}


def validate_completion(items, size):
    if not isinstance(items, list):
        raise RuntimeError('completion did not return an item array')
    authored = [item for item in items if item.get('documentation') == 'package audit']
    expected = [{'label': f'f{i}', 'kind': 3, 'detail': '() -> int', 'documentation': 'package audit'} for i in range(size)]
    expected.append({'label': 'main', 'kind': 3, 'detail': '(args: str[]) -> void', 'documentation': 'package audit'})
    normalize = lambda values: sorted(json.dumps(value, sort_keys=True) for value in values)
    if normalize(authored) != normalize(expected):
        raise RuntimeError(f'completion callable set/signatures differ: {authored}')


def editor(args, metadata, directory, batches):
    server = LSP(Path(metadata['bundle']) / 'bin/canlc', args.repo, request_timeout=max(30, args.size // 4))
    samples = {name: [] for name in ('diagnostics', 'hover', 'definition', 'formatting', 'completion', 'rename')}
    warmups = {name: [] for name in samples}
    request_latencies = {name: [] for name in samples}
    text = (directory / 'src/main.can').read_text()
    uri = (directory / 'src/main.can').as_uri()
    # First call is a resolvable symbol in the emitted workload.
    call_line = next(i for i, line in enumerate(text.splitlines()) if 'call f0()' in line)
    position = {'line': call_line, 'character': 14}
    declaration_line = next(i for i, line in enumerate(text.splitlines()) if line == 'fn int f0')
    rename_expected = expected_rename(uri, call_line, declaration_line)
    version = 1
    identity = 10
    try:
        server.send('initialize', {'rootUri': directory.as_uri()}, 1)
        init = server.until(lambda m: m.get('id') == 1)
        capabilities = init.get('result', {}).get('capabilities', {})
        if not capabilities.get('hoverProvider'):
            raise RuntimeError('LSP does not advertise hover')
        if not capabilities.get('completionProvider') or capabilities.get('renameProvider') is not True:
            raise RuntimeError('production completion/rename capabilities missing')
        server.send('initialized', {})
        server.send('textDocument/didOpen', {'textDocument': {'uri': uri, 'languageId': 'can', 'version': version, 'text': text}})
        healthy = server.until(lambda m: m.get('method') == 'textDocument/publishDiagnostics' and m['params'].get('version') == version and m['params'].get('uri') == uri)
        if healthy['params']['diagnostics']:
            raise RuntimeError('initial fixture diagnostics are not empty')
        # Probe the actual missing method once, outside every latency sample.
        server.send('textDocument/prepareRename', {'textDocument': {'uri': uri}, 'position': position}, 2)
        refused = server.until(lambda m: m.get('id') == 2, allow_errors=True)
        if refused.get('error') != {'code': -32601, 'message': 'unknown method textDocument/prepareRename'}:
            raise RuntimeError('prepareRename unsupported-protocol oracle changed')
        for batch in range(args.warmups + batches):
            totals = {name: 0 for name in samples}
            for operation in range(args.iterations):
                version += 1
                is_invalid = (batch * args.iterations + operation) % 2 == 0
                invalid = text + '\n@\n' if is_invalid else text
                start = time.perf_counter_ns()
                server.send('textDocument/didChange', {'textDocument': {'uri': uri, 'version': version}, 'contentChanges': [{'text': invalid}]})
                result = server.until(lambda m: m.get('method') == 'textDocument/publishDiagnostics' and m['params'].get('version') == version and m['params'].get('uri') == uri)
                elapsed = time.perf_counter_ns() - start
                if bool(result['params']['diagnostics']) != is_invalid:
                    raise RuntimeError('versioned diagnostic invalid/valid oracle failed')
                totals['diagnostics'] += elapsed
                if batch >= args.warmups:
                    request_latencies['diagnostics'].append(elapsed)
                # Restore a checked snapshot before semantic requests, excluded from those timings.
                version += 1
                server.send('textDocument/didChange', {'textDocument': {'uri': uri, 'version': version}, 'contentChanges': [{'text': text}]})
                restored = server.until(lambda m: m.get('method') == 'textDocument/publishDiagnostics' and m['params'].get('version') == version and m['params'].get('uri') == uri)
                if restored['params']['diagnostics']:
                    raise RuntimeError('restored document still has diagnostics')
                for name in ('hover', 'definition', 'formatting', 'completion', 'rename'):
                    identity += 1
                    params = {'textDocument': {'uri': uri}}
                    if name == 'formatting':
                        params['options'] = {'tabSize': 4, 'insertSpaces': True}
                    else:
                        params['position'] = position
                    if name == 'rename':
                        params['newName'] = 'f_zero'
                    start = time.perf_counter_ns()
                    server.send('textDocument/' + name, params, identity)
                    response = server.until(lambda m: m.get('id') == identity)
                    elapsed = time.perf_counter_ns() - start
                    result = response.get('result')
                    if result is None or (name != 'formatting' and not result):
                        raise RuntimeError(f'{name} returned no semantic result')
                    if name == 'definition' and result != {'uri': uri, 'range': rename_expected['changes'][uri][1]['range']}:
                        raise RuntimeError('definition differs from exact f0 declaration location')
                    if name == 'hover' and result != expected_hover('f0', call_line):
                        raise RuntimeError('hover differs from exact declared f0 contract/range/provenance')
                    if name == 'completion':
                        validate_completion(result, args.size)
                    if name == 'rename':
                        validate_rename(result, rename_expected)
                        changed = apply_workspace_edits(text, result['changes'][uri])
                        expected_text = text.replace('call f0()', 'call f_zero()').replace('fn int f0\n', 'fn int f_zero\n')
                        if changed != expected_text:
                            raise RuntimeError('rename application modified unintended text')
                        version += 1
                        server.send('textDocument/didChange', {'textDocument': {'uri': uri, 'version': version}, 'contentChanges': [{'text': changed}]})
                        renamed = server.until(lambda m: m.get('method') == 'textDocument/publishDiagnostics' and m['params'].get('version') == version and m['params'].get('uri') == uri)
                        if renamed['params']['diagnostics']:
                            raise RuntimeError('applied semantic rename is not clean')
                        identity += 1
                        server.send('textDocument/hover', {'textDocument': {'uri': uri}, 'position': position}, identity)
                        renamed_hover = server.until(lambda m: m.get('id') == identity)
                        if renamed_hover.get('result') != expected_hover('f_zero', call_line):
                            raise RuntimeError('renamed symbol identity not reflected by hover')
                        version += 1
                        server.send('textDocument/didChange', {'textDocument': {'uri': uri, 'version': version}, 'contentChanges': [{'text': text}]})
                        restored = server.until(lambda m: m.get('method') == 'textDocument/publishDiagnostics' and m['params'].get('version') == version and m['params'].get('uri') == uri)
                        if restored['params']['diagnostics']:
                            raise RuntimeError('restoring original after rename failed')
                    totals[name] += elapsed
                    if batch >= args.warmups:
                        request_latencies[name].append(elapsed)
            target = warmups if batch < args.warmups else samples
            for name in samples:
                target[name].append(totals[name] / args.iterations)
    finally:
        server.close()
    return [case(args, name, samples[name], warmups[name],
                 'Client monotonic clock from JSON-RPC frame write through matching ' + ('versioned publishDiagnostics' if name == 'diagnostics' else name + ' response') + '; includes wire framing and production LSP work; initialization excluded',
                 {'request_latencies_ns': request_latencies[name], 'transport': 'stdio Content-Length JSON-RPC', 'request_timeout_seconds': server.request_timeout, 'tail_percentiles': 'unavailable; raw per-request latencies supplied'},
                 {'diagnostics': ['versioned invalid/valid diagnostics'], 'hover': ['exact declared contract, range and provenance'], 'definition': ['exact declaration URI and token range'], 'formatting': ['formatting response'], 'completion': ['exact authored callable set, kinds, arities and provenance'], 'rename': ['exact two-token WorkspaceEdit', 'applied text equals intended rename only', 'renamed overlay checks clean', 'renamed hover identity', 'original restored clean']}[name]) for name in samples]


def case(args, name, samples, warmups, scope, metrics, checks, workload="flat", parameters=None):
    return {'name': f'{args.suite}.{workload}.{name}', 'unit': 'ns/op', 'samples': samples,
            'warmup_samples': warmups, 'iterations_per_sample': args.iterations,
            'timing_scope': scope, 'parameters': parameters or {'size': args.size, 'shape': 'flat-call-chain', 'cache': 'warm filesystem; natural GC'},
            'correctness': {'passed': True, 'checks': checks}, 'metrics': metrics}


def run(args):
    if args.size not in SIZES:
        raise Blocked(f'size {args.size} not prepared; supported sizes are {SIZES}')
    meta_path = args.work_dir / 'prepared-driver.json'
    if not meta_path.is_file():
        raise Blocked('run requires successful compiler driver prepare')
    metadata = json.loads(meta_path.read_text())
    if args.suite not in metadata['suites']:
        raise Blocked('requested suite was not included in prepare')
    directory = args.work_dir / f'flat-{args.size}'
    batches = 3 if args.profile == 'quick' else 7
    artifacts = []
    if args.suite == 'editor':
        cases = editor(args, metadata, directory, batches)
    else:
        workloads = [{'name': 'flat', 'directory': str(directory), 'target': 'bun', 'jobs': 1}]
        if args.suite == 'compiler':
            workloads += [dict(anchor, phases='load,check,emit,pipeline') for anchor in metadata['anchors']]
        if args.suite == 'assertions':
            utilities = next(anchor for anchor in metadata['anchors'] if anchor['name'] == 'utilities')
            workloads += [dict(utilities, name='utilities.jobs-' + str(jobs), jobs=jobs) for jobs in (1, 4)]
        cases = []
        for workload in workloads:
            if workload['name'] != 'flat':
                current_inputs = input_inventory(Path(workload['directory']))
                if current_inputs != workload['source_file_hashes'] or input_identity(current_inputs) != workload['input_content_sha256']:
                    raise RuntimeError('maintained anchor inputs changed after prepare: ' + workload['name'])
            argv = [metadata['helper'], '--project', workload['directory'], '--manifest', metadata['manifest'], '--mode', args.suite,
                    '--target', workload['target'], '--iterations', args.iterations, '--warmups', args.warmups, '--batches', batches,
                    '--jobs', workload.get('jobs', 1)]
            if 'phases' in workload:
                argv += ['--phases', workload['phases']]
            raw = command(argv, args.repo, timeout=3600)
            evidence = args.output.with_name(args.output.stem + '.' + workload['name'] + '.raw.jsonl')
            evidence.write_text(raw)
            artifacts.append(str(evidence))
            groups = {}
            for line in raw.splitlines():
                row = json.loads(line)
                groups.setdefault(row['name'], []).append(row)
            if not groups:
                raise RuntimeError('helper produced no samples')
            for name, rows in groups.items():
                measured = [r for r in rows if not r['warmup']]
                metrics = dict(rows[0]['metrics'])
                metrics.update({k: [r[k] for r in measured] for k in ('alloc_bytes_per_op', 'allocations_per_op', 'gc_cycles')})
                metrics['allocation_scope'] = 'Go parent only; worker Bun allocations unavailable'
                metrics['process_tree_rss_bytes'] = 'unavailable'
                if args.suite == 'assertions':
                    metrics.pop('root_wall_ms', None)
                    metrics['root_wall_ms_by_sample'] = [r['metrics']['root_wall_ms'] for r in measured]
                    metrics['root_wall_scope'] = 'production supervisor elapsedMs per root (integer milliseconds); excludes whole-generation validation; raw observations only'
                checks = ['actual production pipeline succeeded']
                if args.suite == 'compiler' and name in ('emit', 'pipeline'):
                    checks.append('final batch emitted bytes and paths equal prepared checked output; oracle outside timing')
                if args.suite == 'assertions':
                    checks = ['all prepared roots delivered', 'every real assertion passed', 'production timeout and worker isolation retained']
                if args.suite == 'artifacts':
                    checks = {'source-maps': ['production encoder request receipt', 'production map semantic validation', 'exact prepared index/maps/modules outside timing'],
                              'publish-new': ['fresh owned empty store', 'current lease selects expected generation', 'all published artifact bytes equal prepared output outside timing'],
                              'validate': ['closed import inventory', 'native TypeScript validation receipt'],
                              'publish-reuse': ['owned publication integrity']}[name]
                parameters = None
                if workload['name'] != 'flat':
                    parameters = {'shape': workload['name'], 'fixed_workload': True, 'requested_scale': args.size,
                                  'size': metrics['source_files'], 'size_unit': 'source files', 'target': workload['target'],
                                  'cache': 'warm filesystem; natural GC', 'input_content_sha256': workload['input_content_sha256']}
                    metrics['source_file_hashes'] = workload['source_file_hashes']
                cases.append(case(args, name, [r['ns_per_op'] for r in measured], [r['ns_per_op'] for r in rows if r['warmup']],
                                  rows[0]['timing_scope'], metrics, checks, workload['name'], parameters))
    notes = ['Three quick or seven standard sample batches; iterations count operations per batch; warmups excluded.']
    if args.suite == 'compiler':
        notes += ['Seven flat scale diagnostics plus maintained utilities and invoice-compare anchors; anchors keep their fixed real source/dependency inventory instead of synthetically scaling the application.',
                  'Stages overlap and cannot be summed; check includes resolve/declaration checking. Runtime bytes are resident before timing. Browser anchor measures checked capability-gated module emission, not browser execution or bundling.']
    if args.suite == 'assertions':
        notes += ['Flat roots plus maintained utilities roots (URL success/domain failure, query ordering, timezone, regex, Unicode/base64 and mocked stdout entry branches) with one and four production workers.',
                  'Every prepared root executes in each operation; parallel results must all pass. This does not cover the complete effect/provider/resource assertion corpus.']
    if args.suite == 'artifacts':
        notes.append('Production source-map encoding/validation, inventory/TypeScript validation, first payload publication and generation reuse are separate boundaries. Browser bundling and distribution release packaging remain outside coverage.')
    if args.suite == 'editor':
        notes.append('Completion and safe rename use exact semantic oracles; renamed overlays are checked and restored outside request timing. prepareRename remains unavailable: current production initialize advertises renameProvider=true with no prepareProvider, and the handler responds -32601 unknown method. No prepareRename latency case is fabricated.')
    return {'suite': args.suite, 'status': 'complete', 'cases': cases, 'notes': notes, 'artifacts': artifacts}



def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('action', choices=['prepare', 'run'])
    parser.add_argument('--repo', required=True, type=Path)
    parser.add_argument('--work-dir', required=True, type=Path)
    parser.add_argument('--output', required=True, type=Path)
    parser.add_argument('--suites', default=','.join(sorted(SUITES)))
    parser.add_argument('--suite', choices=sorted(SUITES))
    parser.add_argument('--profile', choices=['quick', 'standard'], default='quick')
    parser.add_argument('--iterations', type=int, default=1)
    parser.add_argument('--warmups', type=int, default=1)
    parser.add_argument('--size', type=int, default=10)
    args = parser.parse_args()
    args.repo = args.repo.resolve()
    args.work_dir = args.work_dir.resolve()
    args.output = args.output.resolve()
    args.work_dir.mkdir(parents=True, exist_ok=True)
    args.output.parent.mkdir(parents=True, exist_ok=True)
    try:
        if args.iterations < 1 or args.warmups < 0:
            raise ValueError('iterations must be positive; warmups nonnegative')
        if args.action == 'run' and args.suite is None:
            raise ValueError('--suite required for run')
        result = prepare(args) if args.action == 'prepare' else run(args)
        code = 0
    except Blocked as error:
        result = {'suite': args.suite or 'prepare', 'status': 'blocked', 'cases': [], 'reason': str(error)}
        code = 2
    except Exception as error:
        result = {'suite': args.suite or 'prepare', 'status': 'failed', 'cases': [], 'reason': str(error)}
        code = 1
    result['schema_version'] = 1
    args.output.write_text(json.dumps(result, indent=2) + '\n')
    print(json.dumps({'status': result['status'], 'output': str(args.output)}))
    return code


if __name__ == '__main__':
    sys.exit(main())
