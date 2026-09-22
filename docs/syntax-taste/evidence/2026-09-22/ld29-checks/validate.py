"""Validate consultation evidence and contract references; not compiler conformance."""
from pathlib import Path
import hashlib,json,re
HERE=Path(__file__).resolve().parent;ROOT=HERE.parents[4]
requests=[];responses=[];usage=0
for i in range(1,4):
 r=(HERE/f'request-{i}.json').read_bytes();a=(HERE/f'response-{i}.json').read_bytes().rstrip(b'\n');m=json.loads((HERE/f'response-{i}.metadata.json').read_text())
 assert hashlib.sha256(r).hexdigest()==m['request_sha256']
 assert hashlib.sha256(a).hexdigest()==m['response_sha256'] and m['http_status']==200
 requests.append(json.loads(r));responses.append(json.loads(a))
 usage+=sum(responses[-1]['usage'][k] for k in ['input_tokens','output_tokens'])
assert len({r['state'] for r in requests})==3
for key in requests[0]['questions']:
 assert len({r['questions'][key]['instructions'] for r in requests})==3
 options=set(requests[0]['questions'][key]['criteria'])
 assert all(set(r['questions'][key]['criteria'])==options for r in requests)
 for opt in options:assert len({r['questions'][key]['criteria'][opt] for r in requests})==3
 assert all(r['answers'][key]['choice'] in options for r in responses)
assert all(r['answers']['channel']['choice']=='domain' and r['answers']['api']['choice']=='require' and r['answers']['domain_expectations']['choice']=='existing' for r in responses)
assert len({r['answers']['standard_expectations']['choice'] for r in responses})==2
cat=json.loads((ROOT/'compiler/internal/catalogue/catalogue.json').read_text())
assert not any(e['id']==1010 or e['name']=='checks::failed' for e in cat['errors'])
contract=ROOT/'docs/syntax-taste/technical-spec.md'
assert '### C9.2. Named runtime checks' in contract.read_text()
assert 'checks::require(bool condition, str reason) -> void emits [checks::failed]' in contract.read_text()
links=0
for p in [HERE/'README.md',HERE/'pre-dispatch-audit.md']:
 for link in re.findall(r'\]\(([^)]+)\)',p.read_text()):
  if '://' in link or link.startswith('#'):continue
  assert (p.parent/link.split('#')[0]).exists(),link
  links+=1
report={'kind':'ld29-consultation-evidence','requests':3,'questions_per_request':4,'all_prose_fields_distinct':True,'full_request_semantics_review':'pre-dispatch-audit.md','selected':{'api':'checks::require','channel':'domain','error':'checks::failed','assertions':'existing constructor expectations'},'unused_conditional_disagreement':'standard_expectations','reported_tokens':usage,'local_links_checked':links,'proposed_core_id_1010_available':True,'design_gate_closed':True,'compiler_runtime_tests_run':False,'implementation_acceptance_passed':False,'contract_sha256':hashlib.sha256(contract.read_bytes()).hexdigest()}
(HERE/'validation.json').write_text(json.dumps(report,indent=2)+'\n');print(json.dumps(report,indent=2))
