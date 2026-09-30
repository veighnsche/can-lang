"""One-page research probe. Reuse source fixtures; reclaim all build output."""
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile

REPO = Path(sys.argv[1]).resolve()
OUT = Path(__file__).resolve().parent
BASE = OUT / 'inputs'
sys.dont_write_bytecode = True
spec = importlib.util.spec_from_file_location('lower_tree', OUT / 'lower_tree.py')
tree = importlib.util.module_from_spec(spec)
sys.modules[spec.name] = tree
spec.loader.exec_module(tree)
results = {'scope': 'Disposable one-page development check, not release or browser qualification', 'source_commit': subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=REPO, text=True).strip()}

def command(args, cwd=REPO, timeout=120):
    if Path(args[0]).name == 'bun':
        environment = Path(args[-1]).parent/'environment.json'
        environment.write_text('{}')
        with environment.open('rb') as pipe:
            assert pipe.fileno() == 3
            return subprocess.run([str(a) for a in args], cwd=cwd, capture_output=True, text=True, timeout=timeout, pass_fds=(3,))
    return subprocess.run([str(a) for a in args], cwd=cwd, capture_output=True, text=True, timeout=timeout)

def require(p, label):
    if p.returncode:
        results[label] = {'exit': p.returncode, 'stdout': p.stdout[-4000:], 'stderr': p.stderr[-6000:]}
        raise RuntimeError(label + ': ' + p.stderr[-2000:])
    return p.stdout

component = (OUT/'components-current.can').read_text()
notation = (OUT/'editor-current.tree').read_text()
controls = '[id_field, csrf_field, slug_field, title_field, deliverable_field, description_field, submit]'
results['shared_fixture_correction'] = 'Replay starts from the exact build-tested candidates. The earlier inherited prepare/post_form assertion failures and common corrections are retained in attempts and the report.'

def lowered(text):
    body, mapping = tree.lower(tree.parse(text))
    start = component.index('        call heading("Service draft")', component.index('fn html::safe editor'))
    end = component.index('        html::invalid_structure\n', start)
    return component[:start] + body + component[end:]

# Apply the independently authored layout edits after the identical scaffold corrections.
edit_start = component.index('fn html::safe editor')
edited_component = component[:edit_start]+component[edit_start:].replace('        call html::make_tag("form")', '        call html::make_tag("section") as html::tag section_tag\n        call html::element(section_tag, [], [heading, notice_node]) as html::node section\n        call html::make_tag("form")').replace('[heading, notice_node, form, publication]', '[section, publication, form]')
edited_tree = notation.replace('        element h1 []\n            text "Service draft"\n        element p []\n            text notice\n', '        element section []\n            element h1 []\n                text "Service draft"\n            element p []\n                text notice\n        node publication\n').removesuffix('        node publication\n')
variants = {'baseline': (BASE/'baseline.can').read_text(), 'components': component, 'tree': lowered(notation), 'components_edit': edited_component, 'tree_edit': lowered(edited_tree)}
(OUT/'component-edit.can').write_text(edited_component)
(OUT/'tree-edit.tree').write_text(edited_tree)
(OUT/'tree-fresh-lowered.can').write_text(variants['tree'])
results['inputs'] = {n: hashlib.sha256(s.encode()).hexdigest() for n,s in variants.items()}

cases = {'blank': 'service_fields("", "", "", "", ""), "token", ""', 'saved': 'service_fields("21", "logo", "Logo", "One logo", "Design"), "token", "Saved."', 'hostile': 'service_fields("1", "logo", "<script>", "\\\" onfocus=\\\"", "</textarea>"), "token", "Review fields."'}
def wrappers(source):
    for name, args in cases.items():
        source += f'\nfn html::safe render_{name}\n    emits {{html::invalid_structure, html::invalid_url}}\n    asserts\n        sample:  => ok\n    match call editor({args})\n        html::invalid_structure\n        html::invalid_url\n        ok html::safe page => ok page\n'
    return source

try:
    with tempfile.TemporaryDirectory(prefix='html-authoring-', dir=OUT) as temp:
        work = Path(temp).resolve()
        print('Building one disposable development sidecar from the existing pinned archive.', flush=True)
        bundle = Path(require(command(['go','run','./tools/distbuild','--source',REPO,'--archive',REPO/'.local-deps/bun-darwin-aarch64.zip','--out',work/'bundle','--version','html-research'], timeout=180), 'sidecar').strip())
        cli = bundle/'bin/canlc'
        results['compiler'] = require(command([cli,'version']), 'version').strip()
        project = work/'page'
        (project/'src').mkdir(parents=True)
        (project/'can.project.json').write_text('{"source_root":"src","error_registry":"can.errors.json"}\n')
        (project/'can.errors.json').write_text('{"active":[],"retired":[]}\n')
        (project/'src/app').mkdir()
        (project/'src/app/main.can').write_text('package app\n    provides []\n    uses []\nfn void main\n    emits {}\n    given\n        str[] args\n    asserts\n        sample: [] => ok\n    ok\n')
        for name, src in variants.items():
            src = re.sub(r'^package \w+', 'package '+name, wrappers(src), count=1)
            (project/f'src/{name}').mkdir()
            (project/f'src/{name}/page.can').write_text(src)
        print('Checking and rendering the baseline, both candidates, and both independent edits.', flush=True)
        report = json.loads(require(command([cli,'build','--assert-jobs','2',project], timeout=180),'build'))
        results['build'] = {'kind':report.get('kind'), 'assertions':report.get('assertions'), 'exit':0}
        generation = Path(report['directory'])
        runtime = next((generation/'runtime').glob('r-*'))
        found = {}
        for file in (generation/'packages').rglob('*.ts'):
            for part in file.read_text().split('export async function ')[1:]:
                header = part.split('try {',1)[0]
                for name in variants:
                    for case in cases:
                        if f'can.project.root/{name}::render_{case}' in header:
                            found[name+'_'+case] = {'module':str(file), 'symbol':part.split('(',1)[0]}
        assert len(found)==len(variants)*len(cases), found
        driver = work/'render.ts'
        driver.write_text('import {renderSafe} from '+json.dumps(str(runtime/'platform/html.ts'))+';\nimport {$canInitialize} from '+json.dumps(str(generation/'program/state.ts'))+';\n$canInitialize();\nconst entries='+json.dumps(found)+';\nconst out={};\nfor(const [k,e] of Object.entries(entries)){const m=await import(e.module);const r=await m[e.symbol]();out[k]=r.kind==="ok"?{kind:r.kind,html:renderSafe(r.value)}:{kind:r.kind};}\nconsole.log(JSON.stringify(out));\n')
        rendered = json.loads(require(command([bundle/'runtime/bun','--no-env-file',driver]),'render'))
        (OUT/'rendered.json').write_text(json.dumps(rendered,indent=2)+'\n')
        results['equivalence'] = {case: all(rendered[n+'_'+case]==rendered['baseline_'+case] for n in ['components','tree']) for case in cases}
        assert all(results['equivalence'].values())
        results['edit_equivalence'] = {case:rendered['components_edit_'+case]==rendered['tree_edit_'+case] for case in cases}
        assert all(results['edit_equivalence'].values())
        for case in cases:
            raw=rendered['baseline_'+case]['html']
            expected=raw.replace('<body><h1>', '<body><section><h1>')
            expected=expected.replace('</p><form method="post" action="/service/edit">', '</p></section><form method="post" action="/service/edit">',1)
            edit_start=expected.index('<form method="post" action="/service/edit">')
            edit_end=expected.index('</form>',edit_start)+len('</form>')
            end=expected.index('</body>')
            expected=expected[:edit_start]+expected[edit_end:end]+expected[edit_start:edit_end]+expected[end:]
            assert rendered['tree_edit_'+case]['html']==expected
        results['independent_edits']={'components':'first attempt passes exact expected HTML in three states','tree':'first attempt passes exact expected HTML in three states','samples_per_approach':1,'candidate_edit_repairs':0,'shared_fixture_repairs_before_frozen_inputs':2,'not_a_statistical_reliability_estimate':True}
        # Static diagnostics: fresh copies are checked, never rewrite the input fixtures.
        probe = work/'diagnostic'
        (probe/'src').mkdir(parents=True)
        (probe/'can.project.json').write_text((project/'can.project.json').read_text())
        (probe/'can.errors.json').write_text((project/'can.errors.json').read_text())
        diagnostics = {}
        static_cases = {
            'component_wrong_text_type':component.replace('call paragraph(notice)','call paragraph('+controls+')'),
            'tree_wrong_text_type':lowered(notation.replace('text notice','text id_field')),
            'tree_wrong_splice_type':lowered(notation.replace('node id_field','nodes id_field')),
            'component_same_type_wrong_field':component.replace('call title_slot(), fields.title','call title_slot(), fields.deliverable'),
            'component_nested_publication':component.replace(controls, controls[:-1]+', publication]').replace('[heading, notice_node, form, publication]','[heading, notice_node, form]'),
            'tree_nested_publication':lowered(notation.replace('        node publication','            node publication')),
        }
        for name,source in static_cases.items():
            (probe/'src/page.can').write_text(source)
            p=command([cli,'assert','--assert-jobs','2',probe])
            diagnostics[name]={'exit':p.returncode,'diagnostics':p.stderr.replace(str(probe),'$PROJECT'),'report':json.loads(p.stdout) if p.stdout.strip().startswith('{') else p.stdout}
        results['static_diagnostics']=diagnostics
        tree_diagnostics={}
        for name,src in {'literal_nested_form':notation.replace('            node id_field','            element form []'), 'void_child':notation.replace('element form [method_attr, action_attr]','element input []'), 'duplicate_attribute':notation.replace('[method_attr, action_attr]','[method_attr, method_attr]'), 'unknown_tag':notation.replace('element h1 []','element script []')}.items():
            try: tree.parse(src); tree_diagnostics[name]={'accepted':True}
            except tree.TreeError as e: tree_diagnostics[name]=e.diagnostic
        results['tree_diagnostics']=tree_diagnostics
        # Execute already-compiled component helpers with the same opaque runtime values.
        module,symbols=None,{}
        for file in (generation/'packages').rglob('*.ts'):
            for part in file.read_text().split('export async function ')[1:]:
                header=part.split('try {',1)[0]
                for name in ['publish_control','post_form','prepare','editor']:
                    if f'can.project.root/components::{name}' in header:
                        module=str(file);symbols[name]=part.split('(',1)[0]
        # Only the publication helper needs string inputs; HTML constructors supply outer form.
        direct=work/'negative.ts'
        direct.write_text('import {strict as assert} from "node:assert";\nimport {$canInitialize,$canHTML as h} from '+json.dumps(str(generation/'program/state.ts'))+';\nimport {domainFailureDiagnostics} from '+json.dumps(str(runtime/'domain.ts'))+';\nconst m=await import('+json.dumps(module)+');$canInitialize();\nconst tag=(await h.makeTag("form")).value;const out={};\nfor(const id of ["","21"]){const p=await m['+json.dumps(symbols['publish_control'])+'](id,"token");assert.equal(p.kind,"ok");const r=await h.element(tag,[],[p.value]);out[id||"blank"]={kind:r.kind,...(r.kind==="domain"?{error:domainFailureDiagnostics(r.value).declaration.name}: {})};}\nassert.equal(out.blank.kind,"ok");assert.equal(out["21"].error,"html::invalid_structure");console.log(JSON.stringify(out));\n')
        results['nested_publication_runtime']=json.loads(require(command([bundle/'runtime/bun','--no-env-file',direct]),'negative_runtime'))
        results['assertions_do_not_prove_field_binding']='same-typed field mutation passed actual assertions; output binding requires semantic oracle'
        results['all_comparison_checks_passed']=True
finally:
    results['temporary_builds_reclaimed']=not work.exists() if 'work' in globals() else True
    (OUT/'probe-results.json').write_text(json.dumps(results,indent=2)+'\n')
print(json.dumps({k:results[k] for k in ['build','equivalence','edit_equivalence','all_comparison_checks_passed','temporary_builds_reclaimed']}))
