#!/usr/bin/env python3
"""Application workloads. Compilation and browser capability discovery are preparation."""
import argparse
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import uuid

SUITES = {'browser', 'server', 'io', 'journeys'}

class Blocked(Exception):
    pass

def launch_args(args):
    args = [str(x) for x in args]
    if args[0] != 'bun':
        return args
    launcher = 'import os,sys; r,w=os.pipe(); os.write(w,b"{}"); os.close(w); os.dup2(r,3,inheritable=True); os.set_inheritable(3,True); os.close(r) if r != 3 else None; os.execvp(sys.argv[1],sys.argv[1:])'
    return [sys.executable, '-c', launcher, *args]

def command(args, timeout=120):
    proc = subprocess.run(launch_args(args), capture_output=True, text=True, timeout=timeout)
    if proc.returncode:
        raise RuntimeError(f'{args[0]} exited {proc.returncode}: {proc.stderr[-6000:]} {proc.stdout[-1000:]}')
    return proc.stdout

def read_child_result(path, suite=None):
    value = json.loads(path.read_text())
    if not isinstance(value, dict):
        raise ValueError('Child result must be an object')
    if suite is not None:
        if value.get('schema_version') != 1 or value.get('suite') != suite or value.get('status') not in {'complete', 'blocked', 'failed'} or not isinstance(value.get('cases'), list):
            raise ValueError('Malformed app child envelope')
        if value['status'] != 'complete' and not isinstance(value.get('reason'), str):
            raise ValueError('Failed/blocked child envelope requires a reason')
    elif not isinstance(value.get('available'), list) or not isinstance(value.get('unavailable'), dict):
        raise ValueError('Malformed browser capability result')
    return value

def child_result(args, work, suite=None, timeout=300):
    result_path = work / ('apps-child-' + uuid.uuid4().hex + '.json')
    if result_path.exists():
        raise RuntimeError('Child result path unexpectedly exists')
    # Only consume a newly created file after successful child exit. A failing
    # process cannot cause an old or partially written envelope to be reused.
    command([*args, result_path], timeout=timeout)
    if not result_path.is_file():
        raise RuntimeError('Successful child did not write its result file')
    value = read_child_result(result_path, suite)
    if suite is not None:
        value.setdefault('artifacts', []).append(str(result_path))
    return value

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('action', choices=['prepare', 'run'])
    parser.add_argument('--repo', type=Path, required=True)
    parser.add_argument('--work-dir', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--suites', default='')
    parser.add_argument('--suite', choices=sorted(SUITES))
    parser.add_argument('--profile', choices=['quick', 'standard'], required=True)
    parser.add_argument('--iterations', type=int, default=1)
    parser.add_argument('--warmups', type=int, default=1)
    parser.add_argument('--size', type=int, default=100)
    args = parser.parse_args()
    suite = 'prepare' if args.action == 'prepare' else args.suite
    output = {'schema_version': 1, 'suite': suite, 'status': 'failed', 'cases': [], 'notes': [], 'artifacts': []}
    exitcode = 0
    try:
        if not shutil.which('bun'):
            raise Blocked('Bun is required; no dependencies are installed by this driver.')
        if args.iterations < 1 or args.warmups < 0 or args.size < 1:
            raise ValueError('iterations/size must be positive and warmups nonnegative')
        args.work_dir.mkdir(parents=True, exist_ok=True)
        helper = args.repo / 'tools/performance/drivers'
        manifest_path = args.work_dir / 'apps-manifest.json'
        if args.action == 'prepare':
            suites = set(filter(None, args.suites.split(',')))
            if not suites or suites - SUITES:
                raise ValueError(f'Invalid apps suites: {sorted(suites)}')
            manifest = {'schema_version': 1, 'suites': sorted(suites), 'profile': args.profile, 'capabilities': {'browser': 'C02 input/keyboard snapshots, state reset, live caret and observable DOM completion', 'server': 'runtime GET and POST body/response boundary handlers, bounded-client and arrival-rate scenarios', 'io': 'bounded cached text/binary reads, byte-cap errors and binary write/read lifecycle', 'journeys': 'real emitted Can doubled and invoice record validation/map/fold application paths'}}
            if 'browser' in suites:
                if not (args.repo / 'node_modules/playwright').exists():
                    raise Blocked('Installed Playwright package missing.')
                command(['bun', helper / 'apps_build.ts', args.repo, args.work_dir])
                manifest['browser'] = child_result(['bun', helper / 'apps_playwright.ts', 'probe', args.work_dir, args.profile, '0', '0', '0'], args.work_dir)
                manifest_path.write_text(json.dumps(manifest, indent=2) + '\n')
                requested = os.environ.get('CAN_PERF_BROWSER_ENGINES', '').split(',') if os.environ.get('CAN_PERF_BROWSER_ENGINES') else manifest['browser']['available']
                if not requested or any(engine not in manifest['browser']['available'] for engine in requested):
                    raise Blocked(f'Requested browser coverage unavailable: {requested}; available: {manifest["browser"]["available"]}. See manifest for launch failures.')
            if 'journeys' in suites:
                if not shutil.which('go'):
                    raise Blocked('Go is needed for actual compiler emission.')
                emitter = args.work_dir / 'perfemit'
                command(['go', 'build', '-o', emitter, './compiler/perfemit'], timeout=300) if Path.cwd() == args.repo else build_emitter(args.repo, emitter)
                command([emitter, args.repo / 'tools/performance/fixtures/apps', args.work_dir / 'journey', args.repo / 'runtime'])
                generate_adapter(args.work_dir)
            manifest_path.write_text(json.dumps(manifest, indent=2) + '\n')
            output.update(status='complete', artifacts=[str(manifest_path)] + ([str(args.work_dir / 'browser.js')] if 'browser' in suites else []) + ([str(args.work_dir / 'journey'), str(args.work_dir / 'journey-adapter.ts')] if 'journeys' in suites else []), notes=['Preparation builds fixtures outside measured runs. Available engines are recorded explicitly; CAN_PERF_BROWSER_ENGINES requests exact engine coverage.'])
        else:
            if not args.suite:
                raise ValueError('--suite is required for run')
            if not manifest_path.exists():
                raise Blocked('Run requires successful apps prepare.')
            manifest = json.loads(manifest_path.read_text())
            if args.suite not in manifest['suites']:
                raise Blocked(f'{args.suite} was not prepared.')
            options = [args.work_dir, args.profile, args.iterations, args.warmups, args.size]
            if args.suite == 'browser':
                requested = os.environ.get('CAN_PERF_BROWSER_ENGINES', '').split(',') if os.environ.get('CAN_PERF_BROWSER_ENGINES') else manifest['browser']['available']
                if not requested or any(engine not in manifest['browser']['available'] for engine in requested):
                    raise Blocked(f'Requested browser engines unavailable: {requested}')
                output = child_result(['bun', helper / 'apps_playwright.ts', 'run', *options], args.work_dir, 'browser')
            elif args.suite == 'server':
                host = subprocess.Popen(launch_args(['bun', str(helper / 'apps_workloads.ts'), 'server-host', *map(str, options)]), stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
                try:
                    import select
                    if not select.select([host.stdout], [], [], 15)[0]:
                        raise RuntimeError('Server readiness exceeded 15 seconds')
                    ready = json.loads(host.stdout.readline())
                    output = child_result(['bun', helper / 'apps_workloads.ts', 'server', *options, ready['url']], args.work_dir, 'server')
                finally:
                    host.terminate()
                    try:
                        host.communicate(timeout=10)
                    except subprocess.TimeoutExpired:
                        host.kill()
                        host.communicate(timeout=5)
            else:
                output = child_result(['bun', helper / 'apps_workloads.ts', args.suite, *options, ''], args.work_dir, args.suite)
    except Blocked as exc:
        output.update(status='blocked', reason=str(exc))
        exitcode = 2
    except Exception as exc:
        output.update(status='failed', reason=str(exc))
        exitcode = 1
    if output['status'] != 'complete' and exitcode == 0:
        exitcode = 2 if output['status'] == 'blocked' else 1
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(output, indent=2, allow_nan=False) + '\n')
    return exitcode

def build_emitter(repo, emitter):
    proc = subprocess.run(['go', 'build', '-o', str(emitter), './compiler/perfemit'], cwd=repo, capture_output=True, text=True, timeout=300)
    if proc.returncode:
        raise RuntimeError(proc.stderr[-6000:])

def generate_adapter(work):
    import re
    names = ['doubled', 'make_invoice_line', 'summarize_invoice']
    bindings = {}
    for source in sorted((work / 'journey/packages').rglob('*.ts')):
        for name in names:
            match = re.search(r'export async function (\$canFunction\d+)\([^\n]*\nlet \$canOrigin = [^\n]*gallery::' + name + r'"', source.read_text())
            if match:
                if name in bindings:
                    raise RuntimeError('Ambiguous emitted app binding: ' + name)
                bindings[name] = (source, match.group(1))
    if set(bindings) != set(names):
        raise RuntimeError('Missing actual emitted app bindings: ' + str(set(names) - set(bindings)))
    generated = work / 'journey'
    imports = [f'import {{ {symbol} as {name} }} from {json.dumps(str(source))};' for name, (source, symbol) in bindings.items()]
    imports += [f'import {{ $canInitialize }} from {json.dumps(str(generated / "program/state.ts"))};', f'import {{ value }} from {json.dumps(str(generated / "runtime/completion.ts"))};', f'import {{ array }} from {json.dumps(str(generated / "runtime/data.ts"))};', 'await $canInitialize();']
    imports += ['export async function runJourney(input: readonly bigint[]) { return value(await doubled(array(input))) as readonly bigint[]; }', 'export async function runInvoice(rows: readonly {sku:string,quantity:string,unit_minor:string}[]) { const lines = []; for(const row of rows) lines.push(value(await make_invoice_line(row.sku, BigInt(row.quantity), BigInt(row.unit_minor)))); const input=array(lines); const summary=value(await summarize_invoice(input)); return {summary,input,lines}; }']
    (work / 'journey-adapter.ts').write_text('\n'.join(imports) + '\n')

if __name__ == '__main__':
    sys.exit(main())
