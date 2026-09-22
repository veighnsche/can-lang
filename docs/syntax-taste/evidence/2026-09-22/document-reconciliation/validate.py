"""Audit the reconciliation of selected documents, not compiler acceptance."""
from pathlib import Path
import hashlib,json,re,collections
HERE=Path(__file__).resolve().parent
ROOT=HERE.parents[4]
SD=ROOT/'docs/syntax-taste'
NAMES=['decisions.md','technical-spec.md','ai-io-spec.md','coordination-spec.md','platform-testing-spec.md']
mapping=json.loads((HERE/'migration-map.json').read_text())
old=(HERE/'behavior-contracts-before.md').read_text()
# Require all original B1-B7 prose/code after only recorded reference/editorial edits.
def normalize(s):
 s=re.sub(r'\[([^\]]+)\]\([^)]+\)',r'\1',s)
 for k,v in mapping.items():s=re.sub(r'\b'+k+r'\b',v['section'],s)
 s=s.replace('B9 specifies that revision','P15.1 specifies that revision')
 s=s.replace('For plain race, remove the Q6 rejection of distinct generic specializations in the collected leaf set.', 'For plain race, distinct generic specializations are permitted in the collected leaf set.')
 return re.sub(r'\s+',' ',s.replace('####','###')).strip()
for k,v in mapping.items():
 body=re.search(r'^## '+k+r'\. [^\n]+\n(.*?)(?=^## B\d+\.)',old,re.M|re.S)[1]
 assert normalize(body) in normalize((SD/v['file']).read_text()),('missing migrated clause',k)
# Resolve filesystem links and Markdown/explicit anchors in the canonical set.
def anchors(s):
 a=set(re.findall(r'<a id="([^"]+)"',s))
 for h in re.findall(r'^#{1,6} (.+)$',s,re.M):
  h=re.sub(r'\[([^\]]+)\]\([^)]+\)',r'\1',h).lower()
  a.add(re.sub(r'[^\w\- ]','',h).replace(' ','-'))
 return a
links=0;anchor_links=0;issues=[]
for p in [SD/n for n in NAMES]+[ROOT/'docs/implementation/language-behavior-contracts-2026-09-22.md']:
 for link in re.findall(r'\]\(([^)]+)\)',p.read_text()):
  if '://' in link:continue
  target,sep,anchor=link.partition('#');dest=p.parent/target if target else p
  if not dest.exists():issues.append([str(p.relative_to(ROOT)),link,'file']);continue
  links+=1
  if anchor and dest.suffix=='.md':
   anchor_links+=1
   if anchor not in anchors(dest.read_text()):issues.append([str(p.relative_to(ROOT)),link,'anchor'])
assert not issues,issues
texts={name:(SD/name).read_text() for name in NAMES}
for name,s in texts.items():
 for stale in ['[_] as str', 'generic arguments do not\nappear in an error pattern', 'Revision authority:', 'They do not acquire the function-only mandatory', 'Raw provider fixtures are compiler conformance data, not Can syntax', 'the five labels']:
  assert stale not in s,(name,stale)
# Public fetch/judge declaration examples no longer require raw seven-error lists.
raw={'http::invalid_request','http::credentials_missing','http::transport_failed','http::timeout','http::body_limit','http::status_error','codec::invalid_data'}
examples=0
for name,s in texts.items():
 for block in re.findall(r'```(?:can|text)?\n(.*?)\n```',s,re.S):
  owner=False
  for line in block.splitlines():
   if re.match(r'^(fetch|judge)\b',line):owner=True
   elif re.match(r'^\S',line) and not line.startswith('//'):owner=False
   if owner and re.match(r'\s+emits \[',line):
    items=set(re.search(r'\[(.*)\]',line)[1].split(', '))
    assert not items&raw,(name,line)
    examples+=1
ledger=(ROOT/'docs/implementation/language-design-dispositions-2026-09-22.md').read_text()
rows=re.findall(r'^\| (LD\d+) \|.*?\| \*\*(implement|retain current design|defer with a reason)\*\* \|',ledger,re.M)
assert [i for i,_ in rows]==[f'LD{i:02}' for i in range(1,50)]
counts=dict(collections.Counter(v for _,v in rows));assert counts=={'retain current design':18,'defer with a reason':15,'implement':16}
assert 'LD29' in texts['decisions.md'] and 'checks::require' in texts['decisions.md']
report={'kind':'authoritative-document-reconciliation','migrated_contracts':len(mapping),'original_contract_clauses_preserved':True,'canonical_files':{name:hashlib.sha256(s.encode()).hexdigest() for name,s in texts.items()},'local_links_checked':links,'anchor_links_checked':anchor_links,'normalized_native_signature_examples':examples,'dispositions':counts,'new_jev_consultations':0,'remaining_design_gate':None,'subsequent_ld29_consultations':3,'compiler_runtime_tests_run':False,'implementation_acceptance_passed':False}
(HERE/'validation.json').write_text(json.dumps(report,indent=2)+'\n');print(json.dumps(report,indent=2))
