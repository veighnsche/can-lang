"""Execute preregistered current-Can DI-06 private-effect cases."""

from pathlib import Path
import json
import os
import re
import shutil
import subprocess

REPO = Path(__file__).resolve().parents[5]
HERE = Path(__file__).resolve().parent
WORK = Path('/private/tmp/can-di06-fixture-hard-case')
MANIFEST = '{"source_root":"src","error_registry":"can.errors.json"}\n'
ERRORS = '{"active":[],"retired":[]}\n'
MAIN = '''fn void main
    emits []
    given
        str[] args
    asserts
        empty: [] => ok
    ok
'''

AMBIENT_HELPER = '''package helper
    provides [stamp]
    uses [clock, text]
fn int sample_time
    emits []
    asserts
        unit: => ok 1000
    match call clock::wall_millis()
        when
            unit: => ok 1000
            sample: => ok 1000
            customer: => ok 1000
        ok int millis => ok millis
fn str stamp
    emits []
    given
        int invoice
    asserts
        unit: 7 => ok "1000!"
    match call sample_time()
        ok int millis => match call text::from_int(millis)
            ok str label => ok label + "!"
'''

STUB_HELPER = AMBIENT_HELPER.replace('            customer: => ok 1000\n', '')

PARAM_HELPER = '''package helper
    provides [stamp]
    uses [text]
fn str stamp
    emits []
    given
        int invoice
        callable int () emits [] now
    asserts
        unit: 7, callable test_now => ok "1000!"
    match call now()
        ok int millis => match call text::from_int(millis)
            ok str label => ok label + "!"
fn int test_now
    emits []
    asserts
        unit: => ok 1000
    ok 1000
'''

ENTRY_HELPER = '''package helper
    provides [stamp, stamp_with_clock]
    uses [clock, text]
fn int real_now
    emits []
    asserts
        unit: => ok 1000
    match call clock::wall_millis()
        when
            unit: => ok 1000
            sample: => ok 1000
        ok int millis => ok millis
fn str stamp_with_clock
    emits []
    given
        int invoice
        callable int () emits [] now
    asserts
        unit: 7, callable test_now => ok "1000!"
    match call now()
        ok int millis => match call text::from_int(millis)
            ok str label => ok label + "!"
fn int test_now
    emits []
    asserts
        unit: => ok 1000
    ok 1000
fn str stamp
    emits []
    given
        int invoice
    asserts
        unit: 7 => ok "1000!"
    ok call stamp_with_clock(invoice, callable real_now)
'''

def app(case, label):
    uses = '[helper, clock]' if case == 'public-parameter' else '[helper]'
    text = 'package app\n    provides []\n    uses ' + uses + '\n'
    if case == 'public-parameter':
        text += '''fn int real_now
    emits []
    asserts
        unit: => ok 1000
    match call clock::wall_millis()
        when
            unit: => ok 1000
            sample: => ok 1000
        ok int millis => ok millis
fn int fake_now
    emits []
    asserts
        unit: => ok 1000
    ok 1000
'''
        call = 'helper::stamp(invoice, callable real_now)'
        caller = 'helper::stamp(7, callable fake_now)'
    elif case == 'test-entry':
        text += '''fn int fake_now
    emits []
    asserts
        unit: => ok 1000
    ok 1000
'''
        call = 'helper::stamp(invoice)'
        caller = 'helper::stamp_with_clock(7, callable fake_now)'
    else:
        call = 'helper::stamp(invoice)'
        caller = 'helper::stamp(7)'
    text += '''fn str preview
    emits []
    given
        int invoice
    asserts
        sample: 7 => ok "1000!"
    ok call CALL
fn str receipt
    emits []
    given
        int invoice
    asserts
        sample: 7 => ok "1000!"
    ok call CALL
fn str read_customer
    emits []
    asserts
        LABEL: => ok "1000!"
'''.replace('CALL', call).replace('LABEL', label)
    if case == 'whole-helper-stub':
        text += '''    match call helper::stamp(7)
        when
            LABEL: 7 => ok "1000!"
        ok str result => ok result
'''.replace('LABEL', label)
    else:
        text += '    ok call ' + caller + '\n'
    return text + MAIN

CASES = {
    'ambient': AMBIENT_HELPER,
    'whole-helper-stub': STUB_HELPER,
    'public-parameter': PARAM_HELPER,
    'test-entry': ENTRY_HELPER,
}

def run(command, **kwargs):
    return subprocess.run(command, cwd=REPO, text=True, capture_output=True, **kwargs)

def make_project(case, label, helper, caller):
    directory = WORK / case / label
    if directory.exists():
        shutil.rmtree(directory)
    (directory / 'src/helper').mkdir(parents=True)
    (directory / 'src/app').mkdir(parents=True)
    (directory / 'can.project.json').write_text(MANIFEST)
    (directory / 'can.errors.json').write_text(ERRORS)
    (directory / 'src/helper/main.can').write_text(helper)
    (directory / 'src/app/main.can').write_text(caller)
    return directory

def execute(probe, directory, desired='renamed'):
    checked = run([str(probe), str(directory), str(REPO / 'runtime')])
    result = {'checkExit': checked.returncode}
    if checked.returncode or not checked.stdout:
        result['diagnostic'] = checked.stderr
        return result
    parsed = json.loads(checked.stdout)
    result['accepted'] = parsed['accepted']
    result['diagnostic'] = parsed.get('error') or parsed.get('emitError')
    if not parsed.get('entry'):
        return result
    entry = Path(parsed['entry'])
    diagnostics = entry.parent / 'diagnostics'
    diagnostics.mkdir(exist_ok=True)
    (diagnostics / 'source-index.json').write_text(json.dumps({'schemaVersion': 1, 'kind': 'can.source-index', 'sources': [], 'modules': []}))
    envfile = directory / 'empty-environment.json'
    envfile.write_text('{}')
    root_count = len(re.findall(r'import \{ \$canCase as ', entry.read_text()))
    result['rootCount'] = root_count
    result['roots'] = []
    for index in range(root_count):
        ran = run(['/bin/sh', '-c', 'exec bun "$1" "$2" 3< "$3"', 'probe', str(entry), f'root={index}', str(envfile)], timeout=30)
        try:
            report = json.loads(ran.stdout) if ran.stdout else None
        except json.JSONDecodeError:
            report = None
        item = {'index': index, 'exit': ran.returncode, 'report': report,
                'stdout': None if report else ran.stdout, 'stderr': ran.stderr}
        result['roots'].append(item)
        if report and report.get('root', {}).get('name') == desired:
            result['caller'] = {'exit': ran.returncode, 'report': report}
    return result

def main():
    WORK.mkdir(parents=True, exist_ok=True)
    historical_runner = REPO / 'docs/syntax-taste/evidence/2026-09-24/clean-room-full-review/probe-runner.go.txt'
    overlay = WORK / 'overlay.json'
    overlay.write_text(json.dumps({'Replace': {str(REPO / 'compiler/main.go'): str(historical_runner)}}))
    probe = WORK / 'probe'
    built = run(['go', 'build', '-overlay=' + str(overlay), '-o', str(probe), 'compiler/main.go'],
                env={**os.environ, 'GOCACHE': str(WORK / 'go-cache')})
    if built.returncode:
        raise RuntimeError(built.stderr)
    results = []
    for case, helper in CASES.items():
        for label in ('customer', 'renamed'):
            caller = app(case, label)
            directory = make_project(case, label, helper, caller)
            result = {'case': case, 'label': label,
                      'helperLines': len(helper.splitlines()), 'appLines': len(caller.splitlines()),
                      'appHelperCallSites': caller.count('helper::stamp(') + caller.count('helper::stamp_with_clock('),
                      'exportedNames': re.search(r'provides \[([^]]*)\]', helper).group(1).split(', ')}
            result.update(execute(probe, directory, label))
            results.append(result)
    (HERE / 'results.json').write_text(json.dumps(results, indent=2) + '\n')
    for result in results:
        caller = result.get('caller', {}).get('report', {})
        print(result['case'], result['label'], 'accepted', result.get('accepted'),
              'roots', result.get('rootCount'), 'caller', caller.get('status'),
              'evidence', caller.get('assertion', {}).get('evidence'),
              'diagnostic', result.get('diagnostic'))

    controls = []
    edits = [
        ('ambient', 'repair', 'helper', '            customer: => ok 1000', '            renamed: => ok 1000'),
        ('whole-helper-stub', 'suffix', 'helper', 'label + "!"', 'label + "?"'),
        ('public-parameter', 'fake', 'app', '    ok 1000\n', '    ok 2000\n'),
        ('public-parameter', 'suffix', 'helper', 'label + "!"', 'label + "?"'),
        ('test-entry', 'fake', 'app', '    ok 1000\n', '    ok 2000\n'),
        ('test-entry', 'suffix', 'helper', 'label + "!"', 'label + "?"'),
    ]
    for case, edit, owner, old, new in edits:
        directory = WORK / 'controls' / case / edit
        if directory.exists():
            shutil.rmtree(directory)
        shutil.copytree(WORK / case / 'renamed', directory, ignore=shutil.ignore_patterns('generated'))
        path = directory / f'src/{owner}/main.can'
        source = path.read_text()
        if source.count(old) != 1:
            raise RuntimeError(f'{case}/{edit} edit target count {source.count(old)}')
        path.write_text(source.replace(old, new))
        result = {'case': case, 'edit': edit, 'changed': f'src/{owner}/main.can'}
        result.update(execute(probe, directory))
        controls.append(result)
    (HERE / 'controls.json').write_text(json.dumps(controls, indent=2) + '\n')
    for result in controls:
        caller = result.get('caller', {}).get('report', {})
        print('control', result['case'], result['edit'], 'accepted', result.get('accepted'),
              'caller', caller.get('status'), 'evidence', caller.get('assertion', {}).get('evidence'),
              'diagnostic', result.get('diagnostic'))

if __name__ == '__main__':
    main()
