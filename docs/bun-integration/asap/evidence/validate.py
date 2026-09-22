#!/usr/bin/env python3
"""Validate handoff completeness and saved observations, without running production tests."""
import pathlib,json,re,hashlib
p=pathlib.Path(__file__).resolve().parent;out=p.parent;root=p.parents[3]
read=lambda path:json.loads(path.read_text())
plans=read(out/'capabilities.json');parent=read(out.parent/'tasks.json');queue=read(out/'execution-queue.json')['tasks']
ids={t['id'] for t in parent['tasks'] if t['tier']==1}
assert len(plans)==13 and {t['id'] for t in plans}==ids
seen=set()
for t in queue:
 assert t['id'] not in seen
 assert set(t['depends'])<=seen,(t['id'],t['depends'])
 assert t['status']=='pending'
 seen.add(t['id'])
assert all(id+'-DONE' in seen for id in ids)
for t in plans:
 doc=out/t['file'];assert doc.is_file();text=doc.read_text()
 for section in ['Where to implement','Proposed API contract','Implementation sequence','Acceptance evidence','Fixtures and local assertions','Native lowering sketch','Gates and limitations']:
  assert section in text,(doc,section)
 for path,state,why in t['files']:
  if state=='existing':assert (root/path).exists(),path
  assert path in text and why in text
 for step in t['steps']:assert step in text and step in (out/'execution-queue.md').read_text()
 assert t['file'] in (out/'README.md').read_text()
for doc in out.rglob('*.md'):
 for target in re.findall(r'\]\(([^)]+)\)',doc.read_text()):
  if '://' in target or target.startswith('#'):continue
  target=target.split('#')[0]
  assert (doc.parent/target).exists(),(str(doc),target)
hash=lambda data:hashlib.sha256(data).hexdigest()
for i in range(1,4):
 d=p/'consultations';req=(d/f'request-{i}.json').read_bytes();res=(d/f'response-{i}.json').read_bytes().rstrip(b'\n');meta=read(d/f'response-{i}.metadata.json')
 assert hash(req)==meta['requestSHA256'];assert hash(res)==meta['responseSHA256']
 assert read(d/f'response-{i}.json')['answers'].keys()==read(d/f'request-{i}.json')['questions'].keys()
packets=[read(p/'consultations'/f'request-{i}.json') for i in range(1,4)]
for a in range(3):
 for b in range(a):
  assert packets[a]['state']!=packets[b]['state']
  for k,q in packets[a]['questions'].items():
   other=packets[b]['questions'][k];assert q['instructions']!=other['instructions'];assert q['criteria'].keys()==other['criteria'].keys()
   assert all(q['criteria'][x]!=other['criteria'][x] for x in q['criteria'])
r=read(p/'native-probe-results.json');s=read(p/'sqlite-lifecycle-results.json');target=read(root/'distribution/target.json')
assert r['version']==target['runtime']['version']
assert r['sqlite_sql']['value'][0]['id']==9007199254740992
assert r['sqlite_safe_option']['value'][0]['n']=={'bigint':'9007199254740993'}
assert r['sqlite_direct']['value']['n']=={'bigint':'9007199254740993'}
assert r['format_toml']['status']=='error'
for k in ['yaml','json5']:assert r['format_'+k]['value']['n']==9007199254740992
assert r['format_jsonl']['value'][0]['n']==9007199254740992
assert r['jsonl_partial']['value']['partial']==[{'ok':1}]
assert r['jsonl_partial']['value']['chunk']['error']['name']=='SyntaxError'
assert '<script>' in r['markdown']['value']['html'] and 'javascript:' in r['markdown']['value']['html']
assert r['cookie_csrf']['value']['valid'] and not r['cookie_csrf']['value']['crossSession']
assert r['http_websocket']['value']=={'body':'onetwo','echo':'echo'}
assert r['process']['value']=={'out':'$(echo unintended)','err':'err','exit':7}
assert r['stream']['value']['cancelled'] and not r['stream']['value']['locked']
assert r['files']['value']['text']=='hello' and r['files']['value']['glob']==['nested/b.txt']
assert r['password']['value']=={'valid':True,'wrong':False}
assert r['webcrypto']['value']=={'roundtrip':'probe','extractable':False}
assert s['commit']==s['reopened'] and len(s['commit'])==2
assert s['rollback']=={'originalError':True,'rows':[]}
meta=read(p/'probe-run-metadata.json')
for name,m in meta['runs'].items():
 assert hash((p/name).read_bytes())==m['scriptSHA256']
 assert hash((p/(name.removesuffix('-probe.ts')+'-results.json')).read_bytes())==m['resultSHA256'] if name=='sqlite-lifecycle-probe.ts' else hash((p/'native-probe-results.json').read_bytes())==m['resultSHA256']
result={'status':'passed','capabilities':len(plans),'implementationSteps':sum(len(x['steps']) for x in plans),'queueItems':len(queue),'freshConsultations':3,'nativeProbeRuns':len(meta['runs']),'productionImplementationTestsRun':False,'note':'Documentation completeness, local links, existing file paths, DAG, request hashes and saved native observations validated. Does not mark implementation complete.'}
(p/'validation.json').write_text(json.dumps(result,indent=2)+'\n');print(json.dumps(result,indent=2))
