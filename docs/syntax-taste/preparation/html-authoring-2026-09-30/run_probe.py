"""Serial bounded authoring probe; all generated work is owned and reclaimed.

No application/distribution build, private cache, database, service or network.
Build one small emission instrument using the shared Go cache, reuse it, and
symlink to existing runtime source rather than copying dependencies.
"""
import hashlib
import html
import importlib.util
import json
import os
from contextlib import ExitStack
from pathlib import Path
import re
import signal
import subprocess
import sys
import tempfile

BASE = Path(__file__).resolve().parent
REPO = BASE.parents[3]
COMPILER = Path('/Users/vince/.cursor/extensions/can-lang.can-lang-0.2.0/bin/canlc')
spec = importlib.util.spec_from_file_location('tree_probe', BASE / 'lower_tree.py')
tree = importlib.util.module_from_spec(spec)
sys.modules[spec.name] = tree
spec.loader.exec_module(tree)
OWNER = 'can-html-authoring-probe-2026-09-30'
MARKER = '.html-probe-owner.json'
OWNED = []


def alive(pid, group=False):
    try:
        (os.killpg if group else os.kill)(pid, 0)
        return True
    except ProcessLookupError:
        return False
    except PermissionError:
        return True


def mark(path, groups=()):
    payload = dict(owner=OWNER, pid=os.getpid(), path=str(path), groups=list(groups))
    temporary = path / (MARKER + '.new')
    temporary.write_text(json.dumps(payload))
    temporary.replace(path / MARKER)


def recover():
    import shutil
    removed = 0
    for root, pattern in [(Path(tempfile.gettempdir()), 'can-html-probe-*'), (REPO / 'compiler', '.html-probe-*')]:
        for path in root.glob(pattern):
            if path.is_symlink() or not path.is_dir():
                continue
            try:
                info = json.loads((path / MARKER).read_text())
            except (OSError, ValueError):
                continue
            if not isinstance(info, dict) or info.get('owner') != OWNER or info.get('path') != str(path) or type(info.get('pid')) is not int or info['pid'] <= 0:
                continue
            if not isinstance(info.get('groups'), list) or any(type(pid) is not int or pid <= 0 for pid in info['groups']):
                continue
            if alive(info['pid']):
                continue
            if any(alive(pid, group=True) for pid in info.get('groups', [])):
                raise RuntimeError(f'abandoned owned probe still has a live process group: {path}')
            shutil.rmtree(path)
            removed += 1
    return removed


def fingerprint():
    paths = [p for root in [REPO / 'compiler', REPO / 'runtime'] for p in root.rglob('*') if p.is_file() and p.suffix in ('.go', '.ts') and not p.name.endswith('_test.go') and '/test/' not in str(p) and not any(part.startswith('.') for part in p.relative_to(REPO).parts)]
    hashes = {str(p.relative_to(REPO)): hashlib.sha256(p.read_bytes()).hexdigest() for p in sorted(paths)}
    return dict(files=len(hashes), sha256=hashlib.sha256(json.dumps(hashes, sort_keys=True).encode()).hexdigest(), html_sha256=hashes['runtime/platform/html.ts'])


def instruments():
    paths = [BASE / name for name in ['run_probe.py', 'emit_probe.go', 'lower_tree.py', 'render_probe.ts']]
    paths += [COMPILER, REPO / 'go.mod', REPO / 'go.sum']
    return {str(path.relative_to(REPO)) if path.is_relative_to(REPO) else str(path): hashlib.sha256(path.read_bytes()).hexdigest() for path in paths}


def run(command, **kwargs):
    timeout = kwargs.pop('timeout', 25)
    with subprocess.Popen(command, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, start_new_session=True, **kwargs) as process:
        try:
            for path in OWNED:
                mark(path, [process.pid])
            stdout, stderr = process.communicate(timeout=timeout)
        except BaseException:
            # Reclaim children too, before the owned workspace is removed.
            try:
                os.killpg(process.pid, signal.SIGKILL)
            except ProcessLookupError:
                pass
            process.communicate()
            raise
        finally:
            for path in OWNED:
                mark(path)
        return subprocess.CompletedProcess(command, process.returncode, stdout, stderr)


def lowered(source):
    common, _ = (BASE / 'components.can').read_text().split('fn html::safe editor\n', 1)
    original = (BASE / 'components.can').read_text().split('fn html::safe editor\n', 1)[1]
    header = 'fn html::safe editor\n' + original.split('            match chain\n', 1)[0] + '            match chain\n'
    body, mapping = tree.lower(tree.parse(source))
    offset = len((common + header).splitlines())
    for entry in mapping:
        entry['generated_line'] += offset
        for expression in entry['expressions']:
            expression['generated_start'] += 8
            expression['generated_end'] += 8
    body = ''.join('        ' + line + '\n' for line in body.splitlines())
    return common + header + body + '                html::invalid_structure\n                ok => ok document\n', mapping


FIELDS = ['id', 'slug', 'title', 'deliverable', 'description']
CASES = [
    dict(name='blank', fields=dict(zip(FIELDS, ['', '', '', '', ''])), csrf='token', notice=''),
    dict(name='saved', fields=dict(zip(FIELDS, ['21', 'logo', 'Logo', 'One logo', 'Design'])), csrf='token', notice='Saved.'),
    dict(name='hostile', fields=dict(zip(FIELDS, ['1', 'logo', '<script>', '" onfocus="', '</textarea>'])), csrf='token', notice='Review fields.'),
    dict(name='hostile-all', fields=dict(zip(FIELDS, ['1', '"<&', '<script>x</script>', '" onfocus="x', '</textarea><script>x</script>'])), csrf='<csrf&"\'>', notice='<img src=x onerror=x>&"\''),
]


def expected(sample, edit='original'):
    # Independent explicit markup oracle, not the runtime's serializer.
    esc = lambda value: html.escape(value, quote=True)
    hidden = lambda name, value: f'<input type="hidden" name="{name}" value="{esc(value)}">'
    button = lambda label: f'<button type="submit">{label}</button>'
    fields, csrf = sample['fields'], sample['csrf']
    controls = hidden('id', fields['id']) + hidden('csrf', csrf)
    for label, name, identity, maximum in [('URL name', 'slug', 'service_slug', '48'), ('Title', 'title', 'service_title', '120'), ('Deliverable', 'deliverable', 'service_deliverable', '500')]:
        controls += f'<div><label for="{identity}">{label}</label><input type="text" name="{name}" id="{identity}" value="{esc(fields[name])}" maxlength="{maximum}"></div>'
    controls += f'<div><label for="service_description">Description</label><textarea name="description" id="service_description" maxlength="4000">{esc(fields["description"])}</textarea></div>' + button('Save draft')
    form = '<form method="post" action="/service/edit">' + controls + '</form>'
    publication = '<p></p>' if not fields['id'] else '<form method="post" action="/service/state">' + hidden('id', fields['id']) + hidden('csrf', csrf) + hidden('target_state', 'published') + button('Publish service') + '</form>'
    intro = '<h1>Service draft</h1><p>' + esc(sample['notice']) + '</p>'
    if edit == 'group-intro':
        intro = '<section>' + intro + '</section>'
    body = intro + (publication + form if edit == 'move-publication' else form + publication)
    return '<!doctype html><html><head><title>Service draft</title><meta name="viewport" content="width=device-width, initial-scale=1"></head><body>' + body + '</body></html>'


def main():
    recovered = recover()
    source_stamp = fingerprint()
    instrument_stamp = instruments()
    notation = (BASE / 'editor.tree').read_text()
    canonical, mapping = lowered(notation)
    (BASE / 'tree-lowered.can').write_text(canonical)
    (BASE / 'tree-map.json').write_text(json.dumps({'source': 'editor.tree', 'target': 'tree-lowered.can', 'columns': 'one-based scalar, ASCII probe expressions', 'entries': mapping}, indent=2) + '\n')
    candidates = {'baseline': (BASE / 'baseline.can').read_text(), 'components': (BASE / 'components.can').read_text(), 'tree': canonical}
    for edit in ['group-intro', 'move-publication']:
        candidates['components-' + edit] = (BASE / 'pilot-components-checked' / (edit + '.can')).read_text()
        candidates['tree-' + edit] = lowered((BASE / 'pilot-tree-checked' / (edit + '.tree')).read_text())[0]
    candidates['components-nested-publication'] = candidates['components'].replace('parts.action_attr], parts.controls)', 'parts.action_attr], [...parts.controls, parts.publication])')
    nested = notation.replace('            nodes parts.controls\n        node parts.publication', '            nodes parts.controls\n            node parts.publication')
    candidates['tree-nested-publication'] = lowered(nested)[0]
    candidates['tree-viewport-in-body'] = lowered(notation.replace('        node parts.viewport\n', '').replace('    body\n', '    body\n        node parts.viewport\n'))[0]
    candidates['components-tag-typo'] = candidates['components'].replace('                call html::make_tag("form")', '                call html::make_tag("from")')
    before, paragraph = candidates['components'].split('fn html::node paragraph\n', 1)
    candidates['components-void-child'] = before + 'fn html::node paragraph\n' + paragraph.replace('call html::make_tag("p")', 'call html::make_tag("input")', 1)
    candidates['components-duplicate-attribute'] = candidates['components'].replace('[parts.method_attr, parts.action_attr]', '[parts.method_attr, parts.method_attr]')
    diagnostics = []
    negative_trees = {
        'tag-typo': notation.replace('element form ', 'element from '),
        'literal-nested-form': notation.replace('            nodes parts.controls', '            element form []\n                nodes parts.controls'),
        'void-child': notation.replace('        element p []\n            text notice', '        element input []\n            text notice'),
        'duplicate-attribute': notation.replace('parts.method_attr, parts.action_attr', 'parts.method_attr, parts.method_attr'),
        'empty-attribute-slot': notation.replace('parts.method_attr, parts.action_attr', 'parts.method_attr,, parts.action_attr'),
        'effectful-splice': notation.replace('node parts.publication', 'node call publish_control(fields.id, csrf)'),
    }
    for name, source in negative_trees.items():
        try:
            tree.parse(source)
            raise AssertionError(f'seeded notation defect admitted: {name}')
        except tree.TreeError as error:
            diagnostics.append(dict(case=name, phase='prototype tree check', **error.diagnostic))
    with ExitStack() as cleanup:
        work = Path(cleanup.enter_context(tempfile.TemporaryDirectory(prefix='can-html-probe-')))
        mark(work)
        helper_path = Path(cleanup.enter_context(tempfile.TemporaryDirectory(prefix='.html-probe-', dir=REPO / 'compiler')))
        mark(helper_path)
        OWNED.extend([work, helper_path])
        (helper_path / 'main.go').write_bytes((BASE / 'emit_probe.go').read_bytes())
        binary = work / 'emit-probe'
        built = run(['go', 'build', '-p=1', '-o', str(binary), str(helper_path / 'main.go')], cwd=REPO, env={**os.environ, 'GOMAXPROCS': '2'}, timeout=60)
        if built.returncode:
            raise RuntimeError(built.stderr)
        blocked = work / 'blocked-record-helper'
        (blocked / 'src').mkdir(parents=True)
        (blocked / 'src/main.can').write_bytes((BASE / 'record-helper-blocked.can').read_bytes())
        (blocked / 'can.project.json').write_text('{"source_root":"src","error_registry":"can.errors.json"}')
        (blocked / 'can.errors.json').write_text('{"active":[],"retired":[]}')
        rejected = run([str(binary), str(blocked), str(work / 'unused'), str(REPO)])
        assert rejected.returncode != 0 and 'nonvoid success requires a value' in rejected.stderr, rejected.stderr
        diagnostics.append(dict(case='record-helper-bare-ok', phase='full Can assertion program check', message=rejected.stderr.replace(str(work), '<owned>').strip()))
        (blocked / 'src/main.can').write_bytes((BASE / 'attribute-helper-blocked.can').read_bytes())
        rejected = run([str(binary), str(blocked), str(work / 'unused'), str(REPO)])
        assert rejected.returncode != 0 and 'domain-fallible call requires explicit completion handling' in rejected.stderr, rejected.stderr
        diagnostics.append(dict(case='attribute-helper-fallible-fixture', phase='full Can assertion program check', message=rejected.stderr.replace(str(work), '<owned>').strip()))
        specifications, static = [], []
        for name, source in candidates.items():
            project, output = work / name / 'project', work / name / 'emitted'
            (project / 'src').mkdir(parents=True); output.mkdir(parents=True)
            (project / 'src/main.can').write_text(source)
            (project / 'can.project.json').write_text('{"source_root":"src","error_registry":"can.errors.json"}')
            (project / 'can.errors.json').write_text('{"active":[],"retired":[]}')
            checked = run([str(COMPILER), 'inspect-types', str(project)])
            assert checked.returncode == 0, name + ': ' + checked.stderr
            emitted = run([str(binary), str(project), str(output), str(REPO)])
            assert emitted.returncode == 0, name + ': ' + emitted.stderr
            meta = json.loads(emitted.stdout)
            (output / 'runtime').symlink_to(REPO / 'runtime', target_is_directory=True)
            specifications.append({**meta, 'name': name, 'module': str(output / meta['module']), 'state': str(output / 'program/state.ts'), 'cases': CASES})
            static.append(dict(candidate=name, accepted=True, generated_bytes=meta['generated_bytes'], generated_files=meta['generated_files']))
        # Wrong typed splices are checked by Can. Keep a precise mapping back
        # to their row even though they are embedded in a generated array.
        for name, source in [('string-node', notation.replace('node parts.publication', 'node notice')), ('string-nodes', notation.replace('nodes parts.controls', 'nodes notice'))]:
            invalid, source_map = lowered(source)
            project = work / 'invalid'; (project / 'src').mkdir(parents=True, exist_ok=True)
            (project / 'src/main.can').write_text(invalid)
            (project / 'can.project.json').write_text('{"source_root":"src","error_registry":"can.errors.json"}')
            (project / 'can.errors.json').write_text('{"active":[],"retired":[]}')
            rejected = run([str(binary), str(project), str(work / 'unused'), str(REPO)])
            assert rejected.returncode != 0, name
            reported = run([str(binary), 'check-json', str(project)])
            report = json.loads(reported.stdout)
            assert reported.returncode == 1 and not report['result']['accepted'], report
            issues = [d for d in report['result']['diagnostics'] if d.get('severity') == 'error']
            issue = issues[0]
            location = issue.get('location') or {}
            # The structured driver's locations are zero-based UTF-16. This
            # probe's embedded expressions are ASCII and its map is one-based.
            line, column = location.get('start_line', -1) + 1, location.get('start_column', -1) + 1
            rows = [entry for entry in source_map if entry['generated_line'] == line]
            expressions = [e for entry in rows for e in entry['expressions'] if e['generated_start'] <= column < e['generated_end']]
            # A truthful diagnostic gate: rejection is established independently
            # of whether the existing checker exposes the exact embedded span.
            mapped = expressions[0] if expressions else None
            diagnostics.append(dict(case=name, phase='existing Can full checker / structured driver', precise_splice_mapping=bool(mapped), tree_line=mapped['tree_line'] if mapped else None, tree_column=mapped['tree_column'] + column - mapped['generated_start'] if mapped else None, tree_expression=mapped['tree_expression'] if mapped else None, diagnostic=issue, fail_fast_message=rejected.stderr.replace(str(work), '<owned>').strip()))
        specification_file = work / 'specifications.json'
        specification_file.write_text(json.dumps(specifications))
        environment_file = work / 'environment.json'
        environment_file.write_text('{}')
        # Meet the existing runtime launcher contract. The snapshot is empty;
        # no caller credentials/configuration are copied into this observation.
        rendered = run(['sh', '-c', 'exec bun --no-env-file --no-macros --no-install "$2" "$3" 3<"$1"', 'html-probe', str(environment_file), str(BASE / 'render_probe.ts'), str(specification_file)], timeout=25)
        assert rendered.returncode == 0, rendered.stderr
        observations = json.loads(rendered.stdout)
        compact = []
        for candidate in observations:
            for sample, actual in zip(CASES, candidate['observations'], strict=True):
                name = candidate['name']
                if name in ('components-tag-typo', 'components-void-child', 'components-duplicate-attribute'):
                    reason = {'components-tag-typo': 'tag', 'components-void-child': 'void_children', 'components-duplicate-attribute': 'duplicate_attribute'}[name]
                    assert actual['kind'] == 'domain' and actual['payload']['reason'] == reason, actual
                    compact.append(dict(candidate=name, case=sample['name'], kind='expected domain', reason=reason, origin=actual['origin']))
                elif 'nested-publication' in name and sample['fields']['id']:
                    assert actual['kind'] == 'domain' and actual['error'] == 'html::invalid_structure' and actual['payload']['reason'] == 'nested_element', actual
                    compact.append(dict(candidate=name, case=sample['name'], kind='expected domain', reason='nested_element', origin=actual['origin']))
                elif name == 'tree-viewport-in-body':
                    assert actual['kind'] == 'domain' and actual['payload']['reason'] == 'document_context', actual
                    compact.append(dict(candidate=name, case=sample['name'], kind='expected domain', reason='document_context', origin=actual['origin']))
                elif 'nested-publication' in name:
                    assert actual['kind'] == 'ok', actual
                    compact.append(dict(candidate=name, case=sample['name'], kind='ok; blank publication contains no form'))
                else:
                    edit = next((e for e in ['group-intro', 'move-publication'] if name.endswith(e)), 'original')
                    wanted = expected(sample, edit)
                    assert actual['kind'] == 'ok' and actual['html'] == wanted, (name, sample['name'], actual)
                    compact.append(dict(candidate=name, case=sample['name'], kind='ok', exact_independent_oracle=True, html_sha256=hashlib.sha256(wanted.encode()).hexdigest(), html_bytes=len(wanted.encode())))
        evidence = dict(static=static, diagnostics=diagnostics, observations=compact, execution_scope='direct execution of emitted editor functions on four data cases; attached assertions fully checked and emitted but not executed; no integrated toolchain qualification', ai_pilot='two independently dispatched Sol medium agents, one per surface, first attempts after correcting shared preparation; inherited opaque-argument form helper removed mechanically from executable component samples by root; target grouping/reorder edits not repaired; no timing/token comparison; initial pilots inherited the blocked record-helper template and are retained separately')
    OWNED.clear()
    assert not work.exists() and not helper_path.exists()
    assert fingerprint() == source_stamp, 'compiler/runtime source changed during probe; rerun before crediting evidence'
    assert instruments() == instrument_stamp, 'probe tool or dependency manifest changed during execution; rerun before crediting evidence'
    evidence['source_snapshot'] = dict(**source_stamp, can_head=run(['git', 'rev-parse', 'HEAD'], cwd=REPO).stdout.strip(), instruments_sha256=instrument_stamp, fixture_sha256={name: hashlib.sha256(source.encode()).hexdigest() for name, source in candidates.items()})
    evidence['abandoned_owned_directories_recovered'] = recovered
    evidence['cleanup'] = 'owned emission/helper directories removed; repository runtime symlinks removed without touching their target'
    (BASE / 'probe-results.json').write_text(json.dumps(evidence, indent=2) + '\n')
    print(json.dumps({'checked_candidates': len(static), 'rendered_cases': len(compact), 'diagnostic_controls': len(diagnostics), 'cleanup': evidence['cleanup']}))


if __name__ == '__main__':
    def terminate(signum, frame):
        raise KeyboardInterrupt('termination requested; reclaiming owned work')
    signal.signal(signal.SIGTERM, terminate)
    main()
