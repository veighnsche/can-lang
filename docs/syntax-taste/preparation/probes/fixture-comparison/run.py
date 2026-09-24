"""Execute the preregistered DI-06 current-Can mechanism cases."""

from pathlib import Path
import json
import os
import re
import shutil
import subprocess

REPO = Path(__file__).resolve().parents[5]
HERE = Path(__file__).resolve().parent
WORK = Path('/private/tmp/can-di06-fixture-comparison')
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

ambient_helper = '''package helper
    provides [read]
    uses [text]
fn str read
    emits []
    asserts
        unit: => ok "fixture"
    match call text::from_int(7)
        when
            unit: 7 => ok "fixture"
            customer: 7 => ok "fixture"
        ok str result => ok result
'''
template_helper = '''package helper
    provides [read]
    uses [text]
fixture converted for text::from_int
    cases
        7 => ok "fixture"
fn str read
    emits []
    asserts
        unit: => ok "fixture"
    match call text::from_int(7)
        when
            unit: use converted()
            customer: use converted()
        ok str result => ok result
'''
stub_helper = ambient_helper.replace('            customer: 7 => ok "fixture"\n', '')
inject_helper = '''package helper
    provides [read]
    uses []
fn str local_convert
    emits []
    given
        int value
    asserts
        unit: 7 => ok "fixtur"
    ok "fixtur"
fn str read
    emits []
    given
        callable str (int) emits [] convert
    asserts
        unit: callable local_convert => ok "fixture"
    str first = call convert(7)
    ok first + "e"
'''

def app(case: str, label: str) -> str:
    head = 'package app\n    provides []\n    uses [helper]\n'
    if case == 'injected-callable':
        body = '''fn str supplied_convert
    emits []
    given
        int value
    asserts
        unit: 7 => ok "fixtur"
    ok "fixtur"
fn str read_customer
    emits []
    asserts
        LABEL: => ok "fixture"
    ok call helper::read(callable supplied_convert)
'''
    elif case == 'whole-helper-stub':
        body = '''fn str read_customer
    emits []
    asserts
        LABEL: => ok "fixture"
    match call helper::read()
        when
            LABEL: => ok "fixture"
        ok str result => ok result
'''
    else:
        body = '''fn str read_customer
    emits []
    asserts
        LABEL: => ok "fixture"
    ok call helper::read()
'''
    return head + body.replace('LABEL', label) + MAIN

CASES = {
    'ambient-lexical': ambient_helper,
    'lexical-template': template_helper,
    'whole-helper-stub': stub_helper,
    'injected-callable': inject_helper,
}

def run(command, **kwargs):
    return subprocess.run(command, cwd=REPO, text=True, capture_output=True, **kwargs)

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
            directory = WORK / case / label
            if directory.exists():
                shutil.rmtree(directory)
            (directory / 'src/helper').mkdir(parents=True)
            (directory / 'src/app').mkdir(parents=True)
            (directory / 'can.project.json').write_text(MANIFEST)
            (directory / 'can.errors.json').write_text(ERRORS)
            (directory / 'src/helper/main.can').write_text(helper)
            (directory / 'src/app/main.can').write_text(app(case, label))
            checked = run([str(probe), str(directory), str(REPO / 'runtime')])
            result = {'case': case, 'label': label, 'checkExit': checked.returncode}
            if checked.returncode or not checked.stdout:
                result['checkStderr'] = checked.stderr
                results.append(result)
                continue
            parsed = json.loads(checked.stdout)
            result['accepted'] = parsed['accepted']
            if not parsed['accepted'] or 'entry' not in parsed:
                result['diagnostic'] = parsed.get('error') or parsed.get('emitError')
                results.append(result)
                continue
            entry = Path(parsed['entry'])
            diagnostics = entry.parent / 'diagnostics'
            diagnostics.mkdir(exist_ok=True)
            (diagnostics / 'source-index.json').write_text(json.dumps({'schemaVersion': 1, 'kind': 'can.source-index', 'sources': [], 'modules': []}))
            envfile = directory / 'empty-environment.json'
            envfile.write_text('{}')
            root_count = len(re.findall(r'import \{ \$canCase as ', entry.read_text()))
            result['roots'] = []
            for index in range(root_count):
                ran = run(['/bin/sh', '-c', 'exec bun "$1" "$2" 3< "$3"', 'probe', str(entry), f'root={index}', str(envfile)], timeout=30)
                report = None
                if ran.stdout:
                    try:
                        report = json.loads(ran.stdout)
                    except json.JSONDecodeError:
                        pass
                result['roots'].append({'index': index, 'exit': ran.returncode,
                                        'report': report, 'stdout': None if report else ran.stdout,
                                        'stderr': ran.stderr})
            results.append(result)
    (HERE / 'results.json').write_text(json.dumps(results, indent=2) + '\n')
    for result in results:
        roots = result.get('roots', [])
        print(result['case'], result['label'], 'accepted=', result.get('accepted'),
              'roots=', [(r['report'].get('root', {}).get('name'), r['report'].get('status'))
                         if r['report'] else (r['index'], r['exit']) for r in roots],
              'diagnostic=', result.get('diagnostic'))

    controls = []
    for case, edit, old, new in [
        ('ambient-lexical', 'selector-repair', 'customer: 7 =>', 'renamed: 7 =>'),
        ('lexical-template', 'selector-repair', 'customer: use', 'renamed: use'),
        ('injected-callable', 'converter-change', '    ok "fixtur"', '    ok "wrong"'),
        ('injected-callable', 'helper-body-change', '    ok first + "e"', '    ok first + "x"'),
        ('whole-helper-stub', 'helper-body-change', '        ok str result => ok result', '        ok str result => ok "unexpected"'),
    ]:
        directory = WORK / 'controls' / case / edit
        if directory.exists():
            shutil.rmtree(directory)
        shutil.copytree(WORK / case / 'renamed', directory, ignore=shutil.ignore_patterns('generated'))
        changed = directory / ('src/app/main.can' if edit == 'converter-change' else 'src/helper/main.can')
        source = changed.read_text()
        if source.count(old) != 1:
            raise RuntimeError(f'{case}/{edit}: expected one edit target, found {source.count(old)}')
        changed.write_text(source.replace(old, new))
        checked = run([str(probe), str(directory), str(REPO / 'runtime')])
        control = {'case': case, 'edit': edit, 'changed': str(changed.relative_to(directory)),
                   'checkExit': checked.returncode}
        if checked.returncode or not checked.stdout:
            control['diagnostic'] = checked.stderr
        else:
            parsed = json.loads(checked.stdout)
            control['accepted'] = parsed['accepted']
            control['diagnostic'] = parsed.get('error') or parsed.get('emitError')
            if parsed.get('entry'):
                entry = Path(parsed['entry'])
                diagnostics = entry.parent / 'diagnostics'
                diagnostics.mkdir(exist_ok=True)
                (diagnostics / 'source-index.json').write_text(json.dumps({'schemaVersion': 1, 'kind': 'can.source-index', 'sources': [], 'modules': []}))
                envfile = directory / 'empty-environment.json'
                envfile.write_text('{}')
                root_count = len(re.findall(r'import \{ \$canCase as ', entry.read_text()))
                for index in range(root_count):
                    ran = run(['/bin/sh', '-c', 'exec bun "$1" "$2" 3< "$3"', 'probe', str(entry), f'root={index}', str(envfile)], timeout=30)
                    report = json.loads(ran.stdout) if ran.stdout else None
                    if report and report['root']['name'] == 'renamed':
                        control['callerExit'] = ran.returncode
                        control['callerReport'] = report
                        break
        controls.append(control)
    (HERE / 'controls.json').write_text(json.dumps(controls, indent=2) + '\n')
    for control in controls:
        report = control.get('callerReport', {})
        print('control', control['case'], control['edit'], 'accepted=', control.get('accepted'),
              'callerPassed=', report.get('passed'), 'evidence=', report.get('assertion', {}).get('evidence'),
              'diagnostic=', control.get('diagnostic'))

if __name__ == '__main__':
    main()
