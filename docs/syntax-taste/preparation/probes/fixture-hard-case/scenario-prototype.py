"""Disposable generated-output F2 spike over the private-clock current-Can case.

It deliberately does not add Can syntax or modify repository compiler/runtime files.
"""

from pathlib import Path
import json
import os
import re
import shutil
import subprocess

from run import REPO, WORK, HERE, AMBIENT_HELPER, app, make_project, run

PROTOTYPE = WORK / 'scenario-prototype'

def helper_source(suffix='!'):
    source = AMBIENT_HELPER.replace(
        '        unit: => ok 1000\n    match call clock::wall_millis()',
        '        unit: => ok 1000\n        fixed_time: => ok 1000\n    match call clock::wall_millis()',
    ).replace('            customer: => ok 1000', '            fixed_time: => ok 1000')
    return source.replace('label + "!"', f'label + "{suffix}"')

def app_source(two_calls=False):
    source = app('ambient', 'renamed')
    source = source.replace('    ok call helper::stamp(invoice)\n',
        '    match call helper::stamp(invoice)\n'
        '        when\n'
        '            sample: 7 => ok "1000!"\n'
        '        ok str value => ok value\n')
    if two_calls:
        source = source.replace('        renamed: => ok "1000!"', '        renamed: => ok "1000!1000!"')
        source = source.replace('    ok call helper::stamp(7)\n',
            '    str first = call helper::stamp(7)\n'
            '    str second = call helper::stamp(7)\n'
            '    ok first + second\n')
    return source

def manifest(two_calls=False):
    return {
        'owner': 'can.project.root/helper',
        'callee': 'can.project.root/helper::stamp',
        'scenario': 'fixed_time',
        'caller': 'can.project.root/app::read_customer',
        'target': 'can.project.root/helper::sample_time#0/when',
        'targetOperation': 'clock::wall_millis',
        'targetArguments': [],
        'successType': 'int',
        'row': 'fixed_time',
        'occurrences': 2 if two_calls else 1,
        'callSites': [f'can.project.root/app::read_customer#{index}' for index in range(2 if two_calls else 1)],
    }

def assert_manifest(link, helper, caller, generated_app):
    expected = manifest(len(link.get('callSites', [])) == 2)
    for field in ('owner', 'callee', 'scenario', 'caller', 'target', 'targetOperation',
                  'targetArguments', 'successType', 'row', 'occurrences'):
        if link.get(field) != expected[field]:
            raise ValueError(f'scenario link {field} mismatch')
    if link.get('callSites') != expected['callSites']:
        raise ValueError('scenario link call-site list mismatch')
    if 'package helper\n    provides [stamp]' not in helper:
        raise ValueError('scenario owner or callee is inaccessible')
    if 'fixed_time: => ok 1000' not in helper or 'match call clock::wall_millis()' not in helper:
        raise ValueError('helper-owned scenario row or target missing')
    if caller.count('call helper::stamp(7)') != len(link['callSites']):
        raise ValueError('caller link count no longer matches source')
    for site in link['callSites']:
        if generated_app.count(f'"{site}"') != 1:
            raise ValueError(f'caller call site missing or ambiguous: {site}')

def patch_runtime(generated, link):
    runtime = generated / 'runtime'
    if not runtime.is_symlink():
        raise RuntimeError('expected generated runtime symlink')
    runtime.unlink()
    shutil.copytree(REPO / 'runtime', runtime)
    context = runtime / 'assert/context.ts'
    source = context.read_text()
    anchor = 'const contexts = new WeakMap<object, View>();'
    insert = '''const scenarioByContext = new WeakMap<object, string>();
export function contextScenario(context: AssertionContext): string | undefined {
  return scenarioByContext.get(context);
}
export async function withScenario<T>(
  context: AssertionContext | undefined,
  name: string,
  run: () => Promise<T> | T,
): Promise<T> {
  if (context === undefined) return run();
  if (scenarioByContext.has(context)) throw new Error("ambiguous scenario link");
  scenarioByContext.set(context, name);
  try { return await run(); }
  finally { scenarioByContext.delete(context); }
}
'''
    if source.count(anchor) != 1:
        raise RuntimeError('context state anchor changed')
    source = source.replace(anchor, anchor + '\n' + insert)
    anchor = '  suspendFrame(parent.frame);\n  startFrame(view(child).frame);'
    replacement = '''  const linkedScenario = scenarioByContext.get(context);
  if (linkedScenario !== undefined) scenarioByContext.set(child, linkedScenario);
  suspendFrame(parent.frame);
  startFrame(view(child).frame);'''
    if source.count(anchor) != 1:
        raise RuntimeError('context propagation anchor changed')
    context.write_text(source.replace(anchor, replacement))
    fixtures = runtime / 'assert/fixtures.ts'
    source = fixtures.read_text()
    source = source.replace('  contextReport,\n', '  contextReport,\n  contextScenario,\n', 1)
    anchor = '  const selected = rows.filter((row) => row.selector === contextReport(context).root.name);'
    replacement = '''  const root = contextReport(context).root;
  let selected = rows.filter((row) =>
    (identity.startsWith(root.package + "::") && row.selector === root.name) ||
    (contextScenario(context) === "SCENARIO" && identity === "TARGET" && row.selector === "ROW")
  );
  if (contextScenario(context) === "SCENARIO" && identity === "TARGET" && selected.length === 1)
    selected = Array.from({ length: OCCURRENCES }, () => selected[0]);'''.replace('SCENARIO', link['scenario']).replace('TARGET', link['target']).replace('ROW', link['row']).replace('OCCURRENCES', str(link['occurrences']))
    if source.count(anchor) != 1:
        raise RuntimeError('fixture selection anchor changed')
    fixtures.write_text(source.replace(anchor, replacement))

def patch_app(generated, link):
    files = [path for path in (generated / 'packages').rglob('*.ts')
             if 'can.project.root/app::read_customer#0' in path.read_text()]
    if len(files) != 1:
        raise RuntimeError('generated app module missing or ambiguous')
    path = files[0]
    source = path.read_text()
    import_line = 'import { withScenario as $canWithScenario } from "../../runtime/assert/context.ts";\n'
    source = import_line + source
    for site in link['callSites']:
        escaped = re.escape(site)
        pattern = re.compile(r'\(\) => (\$canCallContext\(\$canContext,"' + escaped + r'",\(\$canContext\) => \$canFunction\d+\(\$canExpr\d+, \$canContext\),\$canCallableInstance\(\$canFunction\d+\)\))')
        source, count = pattern.subn(lambda match: f'() => $canWithScenario($canContext,"{link["scenario"]}",() => {match.group(1)})', source)
        if count != 1:
            raise RuntimeError(f'generated exact call site not patchable: {site}, count={count}')
    path.write_text(source)

def execute_generated(directory, root_name='renamed'):
    entry = directory / 'generated/entry.ts'
    envfile = directory / 'empty-environment.json'
    envfile.write_text('{}')
    roots = []
    count = len(re.findall(r'import \{ \$canCase as ', entry.read_text()))
    for index in range(count):
        ran = run(['/bin/sh', '-c', 'exec bun "$1" "$2" 3< "$3"', 'prototype',
                   str(entry), f'root={index}', str(envfile)], timeout=30)
        try:
            report = json.loads(ran.stdout) if ran.stdout else None
        except json.JSONDecodeError:
            report = None
        roots.append({'index': index, 'exit': ran.returncode, 'report': report,
                      'stdout': None if report else ran.stdout, 'stderr': ran.stderr})
    caller = next((item for item in roots if item['report'] and item['report']['root']['name'] == root_name), None)
    return {'roots': roots, 'caller': caller}

def main():
    probe = WORK / 'probe'
    if not probe.exists():
        raise RuntimeError('run.py must build the current-Can probe first')
    outcomes = []
    for name, suffix, two_calls in [('single', '!', False), ('suffix-change', '?', False),
                                     ('two-calls', '!', True)]:
        helper = helper_source(suffix)
        caller = app_source(two_calls)
        directory = make_project('scenario-prototype', name, helper, caller)
        checked = run([str(probe), str(directory), str(REPO / 'runtime')])
        result = {'name': name, 'checkExit': checked.returncode}
        parsed = json.loads(checked.stdout) if checked.stdout else {}
        result['accepted'] = parsed.get('accepted')
        result['diagnostic'] = parsed.get('error') or parsed.get('emitError') or checked.stderr
        if not parsed.get('entry'):
            outcomes.append(result)
            continue
        generated = directory / 'generated'
        (generated / 'diagnostics').mkdir(exist_ok=True)
        (generated / 'diagnostics/source-index.json').write_text(json.dumps({'schemaVersion': 1, 'kind': 'can.source-index', 'sources': [], 'modules': []}))
        link = manifest(two_calls)
        (directory / 'scenario-link.json').write_text(json.dumps(link, indent=2) + '\n')
        app_module = next(path for path in (generated / 'packages').rglob('*.ts')
                          if 'can.project.root/app::read_customer#0' in path.read_text())
        assert_manifest(link, helper, caller, app_module.read_text())
        patch_runtime(generated, link)
        patch_app(generated, link)
        result.update(execute_generated(directory))
        outcomes.append(result)
    negative = []
    helper, caller = helper_source(), app_source()
    valid = manifest()
    generated_app = '"can.project.root/app::read_customer#0"'
    for name, mutation in [
        ('missing-scenario', {'scenario': 'deleted'}),
        ('stale-target', {'target': 'can.project.root/helper::sample_time#8/when'}),
        ('wrong-callee', {'callee': 'can.project.root/helper::unknown'}),
        ('missing-call-site', {'callSites': ['can.project.root/app::read_customer#9']}),
        ('wrong-success-type', {'successType': 'str'}),
        ('wrong-arguments', {'targetArguments': [7]}),
    ]:
        candidate = {**valid, **mutation}
        try:
            assert_manifest(candidate, helper, caller, generated_app)
            negative.append({'name': name, 'rejected': False})
        except ValueError as error:
            negative.append({'name': name, 'rejected': True, 'diagnostic': str(error)})
    report = {'positive': outcomes, 'negativeManifestChecks': negative}
    (HERE / 'scenario-prototype-results.json').write_text(json.dumps(report, indent=2) + '\n')
    for item in outcomes:
        caller = item.get('caller') or {}
        record = caller.get('report') or {}
        print(item['name'], 'accepted', item['accepted'], 'roots', len(item.get('roots', [])),
              'caller passed', record.get('passed'), 'evidence', record.get('assertion', {}).get('evidence'),
              'diagnostic', item['diagnostic'])
    for item in negative:
        print('negative', item['name'], item['rejected'], item.get('diagnostic'))

if __name__ == '__main__':
    main()
