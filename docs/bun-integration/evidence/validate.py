"""Validate independent roadmap order/coverage and saved consultation evidence."""
from pathlib import Path
import collections,hashlib,json,re
from urllib.parse import urlparse
E=Path(__file__).resolve().parent;D=E.parent
obj=json.loads((D/'tasks.json').read_text());ts=obj['tasks'];seen=set();sources=set();counts=collections.Counter()
files={1:'1-implement-asap.md',2:'2-implement-later.md',3:'3-implement-whenever.md'}
for t in ts:
 assert t['id'] not in seen and set(t['depends'])<=seen,(t['id'],t['depends'])
 seen.add(t['id']);counts[t['tier']]+=1
 assert t['status']=='pending'
 s=(D/files[t['tier']]).read_text()
 assert '- [ ] **'+t['id']+' — '+t['title']+'**' in s
 for field in ['work','reason','acceptance','native','surface']:assert t[field] in s,(t['id'],field)
 for u in t['sources']:assert urlparse(u).hostname=='bun.sh' and u in s;sources.add(u)
assert dict(counts)=={1:13,2:14,3:9}
assert any(t['tier']==1 and 'MySQL' in t['title'] for t in ts)
assert any(t['tier']==1 and 'SQLite' in t['title'] for t in ts)
assert sum('primitive' in t['surface'].lower() for t in ts)>=8
links=0
for p in [D/'README.md',E/'README.md']+[D/f for f in files.values()]:
 for u in re.findall(r'\]\(([^)]+)\)',p.read_text()):
  if '://' in u or u.startswith('#'):continue
  target=(p.parent/u.split('#')[0]).resolve()
  if target==E/'validation.json':continue
  assert target.exists(),(p,u)
  links+=1
rs=[];ans=[];total=0
for i in range(1,4):
 raw=(E/f'request-{i}.json').read_bytes();out=(E/f'response-{i}.json').read_bytes().rstrip(b'\n');m=json.loads((E/f'response-{i}.metadata.json').read_text())
 assert m['http_status']==200 and hashlib.sha256(raw).hexdigest()==m['request_sha256'] and hashlib.sha256(out).hexdigest()==m['response_sha256']
 rs.append(json.loads(raw));ans.append(json.loads(out));total+=sum(ans[-1]['usage'][k] for k in ['input_tokens','output_tokens'])
assert len({r['state'] for r in rs})==3
for k in rs[0]['questions']:
 assert len({r['questions'][k]['instructions'] for r in rs})==3
 keys=set(rs[0]['questions'][k]['criteria'])
 assert all(set(r['questions'][k]['criteria'])==keys for r in rs)
 for key in keys:assert len({r['questions'][k]['criteria'][key] for r in rs})==3
 assert all(a['answers'][k]['choice'] in keys for a in ans)
report={'kind':'bun-integration-roadmap-validation','tasks':len(ts),'tiers':dict(counts),'all_pending':True,'dependency_order_valid':True,'both_sql_backends_asap':True,'primitive_candidate_tasks':sum('primitive' in t['surface'].lower() for t in ts),'official_source_urls':len(sources),'local_links_checked':links,'fresh_jev_requests':len(rs),'questions_per_request':len(rs[0]['questions']),'reported_tokens':total,'files_sha256':{p.name:hashlib.sha256(p.read_bytes()).hexdigest() for p in [D/'README.md',D/'tasks.json']+[D/f for f in files.values()]},'native_api_qualification_run':False,'implementation_started':False,'existing_implementer_contacted':False}
(E/'validation.json').write_text(json.dumps(report,indent=2)+'\n');print(json.dumps(report,indent=2))
