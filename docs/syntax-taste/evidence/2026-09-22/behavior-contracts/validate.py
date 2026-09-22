"""Validate documentation/evidence integrity; this does not compile proposed Can."""
from pathlib import Path
import base64, collections, hashlib, json, re
ROOT=Path(__file__).resolve().parents[5]
HERE=Path(__file__).resolve().parent
contract=ROOT/'docs/implementation/language-behavior-contracts-2026-09-22.md'
ledger=ROOT/'docs/implementation/language-design-dispositions-2026-09-22.md'
checked_links=0
for path in [contract,ledger,HERE/'README.md',HERE/'disagreement-investigation.md']:
 text=path.read_text()
 for target in re.findall(r'\]\(([^)]+)\)',text):
  if '://' in target or target.startswith('#'): continue
  assert (path.parent/target.split('#')[0]).exists(),(path,target)
  checked_links+=1
s=contract.read_text()
ids=re.findall(r'^\| (BC\d+) \|',s,re.M)
assert ids==[f'BC{i:02}' for i in range(1,25)]
for i in range(1,10): assert f'## B{i}.' in s
rows=re.findall(r'^\| (LD\d+) \|.*?\| \*\*(implement|retain current design|defer with a reason)\*\* \|',ledger.read_text(),re.M)
assert [i for i,_ in rows]==[f'LD{i:02}' for i in range(1,50)]
counts=dict(collections.Counter(status for _,status in rows))
assert counts=={'retain current design':18,'defer with a reason':15,'implement':16},counts
requests=[]
responses=[]
for i in range(1,4):
 raw=(HERE/f'request-{i}.json').read_bytes()
 answer=(HERE/f'response-{i}.json').read_bytes().rstrip(b'\n')
 meta=json.loads((HERE/f'response-{i}.metadata.json').read_text())
 assert hashlib.sha256(raw).hexdigest()==meta['request_sha256']
 assert hashlib.sha256(answer).hexdigest()==meta['response_sha256']
 assert meta['http_status']==200
 requests.append(json.loads(raw));responses.append(json.loads(answer))
assert len({p['state'] for p in requests})==3
for key in requests[0]['questions']:
 assert len({p['questions'][key]['instructions'] for p in requests})==3
 for option in requests[0]['questions'][key]['criteria']:
  assert len({p['questions'][key]['criteria'][option] for p in requests})==3
 assert all(set(p['questions'][key]['criteria'])==set(requests[0]['questions'][key]['criteria']) for p in requests)
 assert all(key in r['answers'] for r in responses)
# Validate catalogue facts used to settle the payload, without changing allocation.
cat=json.loads((ROOT/'compiler/internal/catalogue/catalogue.json').read_text())
errors=cat['errors']
assert not any(e['id']==1106 for e in errors)
expected={'http::invalid_request':[('reason','str')], 'http::credentials_missing':[('variable','str')], 'http::transport_failed':[('phase','str')], 'http::timeout':[('timeout_ms','int')], 'http::body_limit':[('limit','int')], 'http::status_error':[('status','int'),('headers','http::header[]')], 'codec::invalid_data':[('path','str'),('reason','str')]}
for name,fields in expected.items():
 error=next(e for e in errors if e['name']==name)
 assert [(f['name'],f['type']) for f in error['fields']]==fields
platform=ROOT/'docs/syntax-taste/platform-testing-spec.md'
owner=platform.read_text().split('<a id="p41-attached-native-and-wrapper-assertions"></a>',1)[1].split('## P5.',1)[0]
examples=[json.loads(x) for x in re.findall(r'```json\n(.*?)\n```',owner,re.S)]
assert len(examples)==1
ex=examples[0]
assert set(ex)=={'schema','target','environment','exchange'}
assert ex['schema']=='can.native-fixture.v1'
assert json.loads(base64.b64decode(ex['exchange']['outcome']['response']['body_base64']))=={'id':7}
report={'kind':'documentation-integrity','local_links_checked':checked_links,'acceptance_cases':len(ids),'dispositions':counts,'consultations':len(requests),'choices_per_request':len(requests[0]['questions']),'catalogue_payloads_checked':len(expected),'proposed_id_1106_currently_available':True,'json_examples_checked':len(examples),'contract_index_sha256':hashlib.sha256(contract.read_bytes()).hexdigest(),'authoritative_sha256':{name:hashlib.sha256((ROOT/'docs/syntax-taste'/name).read_bytes()).hexdigest() for name in ['decisions.md','technical-spec.md','ai-io-spec.md','coordination-spec.md','platform-testing-spec.md']},'proposed_can_compiled':False,'compiler_runtime_tests_run':False}
(HERE/'validation.json').write_text(json.dumps(report,indent=2)+'\n')
print(json.dumps(report,indent=2))
