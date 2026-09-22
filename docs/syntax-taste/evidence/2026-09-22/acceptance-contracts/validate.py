"""Check acceptance-spec coverage and frozen source counts, not implementation correctness."""
from pathlib import Path
import hashlib,json,re
HERE=Path(__file__).resolve().parent
ROOT=HERE.parents[4]
DOC=ROOT/'docs/implementation/language-change-acceptance-2026-09-22.md'
LEDGER=ROOT/'docs/implementation/language-design-dispositions-2026-09-22.md'
text=DOC.read_text()
accepted=set(re.findall(r'^\| (LD\d+) \|.*?\| \*\*implement\*\* \|',LEDGER.read_text(),re.M))
sections=re.split(r'^## (AE\d+) — ([^\n]+)\n',text,flags=re.M)
fields=['Before/after','Successful programs','Rejected programs','Runtime evidence','Fixture evidence','Native lowering','Completion gate']
records=[]
for i in range(1,len(sections),3):
 id,title,body=sections[i:i+3]
 # Last section ends before the final general evidence note.
 body=body.split('\n## Existing evidence versus required evidence')[0]
 evidence={}
 for j,name in enumerate(fields):
  marker='**'+name+'.**'
  assert body.count(marker)==1,(id,name)
  value=body.split(marker,1)[1]
  if j+1<len(fields): value=value.split('**'+fields[j+1]+'.**',1)[0]
  assert len(value.strip())>50,(id,name)
  evidence[name]=value.strip()
 records.append({'acceptance_id':id,'disposition_id':'LD'+id[2:],'title':title,'state':'evidence-required','required_evidence':evidence})
assert len(records)==16
assert {r['disposition_id'] for r in records}==accepted
links=0
for target in re.findall(r'\]\(([^)]+)\)',text):
 if '://' in target or target.startswith('#'): continue
 assert (DOC.parent/target.split('#')[0]).exists(),target
 links+=1
meta=json.loads((HERE/'baseline.json').read_text())
raw=(HERE/'eight-fetch-before.can.txt').read_bytes()
assert hashlib.sha256(raw).hexdigest()==meta['sha256']
source=raw.decode();selected=source.split('\n'+meta['comparison_end_marker']+'\n',1)[0]
errors=['http::invalid_request','http::credentials_missing','http::transport_failed','http::timeout','http::body_limit','http::status_error','codec::invalid_data']
lines=selected.splitlines();bounds=[l for l in lines if l.strip().startswith('emits [') and any(e in l for e in errors)]
fetches=re.findall(r'^fetch .*? (\w+) from ',selected,re.M)
metrics={'fetch_declarations':len(fetches),'main_fetch_calls':sum(len(re.findall(r'\bmatch call '+re.escape(f)+r'\(',selected)) for f in fetches),'infrastructure_bound_declarations':len(bounds),'infrastructure_bound_entries':sum(l.count(e) for l in bounds for e in errors),'infrastructure_forwarding_arms':sum(l.strip() in errors for l in lines)}
assert metrics=={'fetch_declarations':8,'main_fetch_calls':8,'infrastructure_bound_declarations':9,'infrastructure_bound_entries':63,'infrastructure_forwarding_arms':56},metrics
manifest={'kind':'language-change-acceptance-specification','implementation_acceptance_passed':False,'document':str(DOC.relative_to(ROOT)),'document_sha256':hashlib.sha256(DOC.read_bytes()).hexdigest(),'records':records}
(HERE/'acceptance-manifest.json').write_text(json.dumps(manifest,indent=2,ensure_ascii=False)+'\n')
result={'kind':'acceptance-document-validation','accepted_dispositions':sorted(accepted),'records':len(records),'dimensions_per_record':len(fields),'local_links_checked':links,'before_metrics':metrics,'after_metrics_status':'required targets; not measured from implemented syntax','design_gated':[],'implementation_tests_run':False}
(HERE/'validation.json').write_text(json.dumps(result,indent=2)+'\n')
print(json.dumps(result,indent=2))
