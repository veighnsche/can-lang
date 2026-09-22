"""Check the implementation planning documents, not implementation acceptance."""
from pathlib import Path
import json,re,hashlib
HERE=Path(__file__).resolve().parent;ROOT=HERE.parents[4];D=ROOT/'docs/implementation'
obj=json.loads((HERE/'tasks.json').read_text());tasks=obj['tasks']
ids=[t['id'] for t in tasks];assert ids==[f'LF{i:02}' for i in range(1,22)]
seen=set();ancestors={};coverage={}
for t in tasks:
 assert t['status']=='pending' and t['evidence']==[]
 assert len(set(t['depends']))==len(t['depends'])
 assert set(t['depends'])<=seen,(t['id'],t['depends'])
 ancestors[t['id']]=set(t['depends'])
 for d in t['depends']:ancestors[t['id']]|=ancestors[d]
 for key in ['work','positive','negative','integration']:assert len(t[key])>70,(t['id'],key)
 for path in t['areas']:assert (ROOT/path).exists(),path
 for ae in t['acceptance']:coverage.setdefault(ae,[]).append(t['id'])
 seen.add(t['id'])
assert {'LF03','LF04','LF11','LF12','LF13','LF14'}<=ancestors['LF15']
assert {'LF05','LF09','LF10','LF11'}<=ancestors['LF12']
assert {'LF11','LF12'}<=ancestors['LF13']
assert set(ids[:-1])<=ancestors['LF21']
ledger=(D/'language-design-dispositions-2026-09-22.md').read_text()
accepted={'AE'+x[2:] for x in re.findall(r'^\| (LD\d+) \|.*?\| \*\*implement\*\* \|',ledger,re.M)}
assert set(coverage)==accepted and len(accepted)==16
md=(D/'language-fixes-tasks-2026-09-22.md').read_text()
assert re.findall(r'^- \[ \] \*\*(LF\d+) —',md,re.M)==ids
assert '- [x]' not in md
for t in tasks:
 block=md.split('**'+t['id']+' —',1)[1].split('\n- [ ] **',1)[0]
 assert ', '.join(t['depends']) in block if t['depends'] else 'none' in block
 for key in ['work','positive','negative','integration']:assert t[key] in block
 for ae in t['acceptance']:assert ae in block
plan=D/'language-fixes-plan-2026-09-22.md'
assert set(re.findall(r'^\| (AE\d+) \|',plan.read_text(),re.M))==accepted
links=0
paths=[plan,D/'language-fixes-tasks-2026-09-22.md',D/'README.md',HERE/'README.md']
for p in paths:
 for link in re.findall(r'\]\(([^)]+)\)',p.read_text()):
  if '://' in link or link.startswith('#'):continue
  target=(p.parent/link.split('#')[0]).resolve()
  if target==HERE/'validation.json':continue
  assert target.exists(),(p.name,link)
  links+=1
report={'kind':'language-fixes-plan-validation','ordered_tasks':len(tasks),'pending_tasks':len(tasks),'completed_tasks':0,'dependency_edges':sum(len(t['depends']) for t in tasks),'topological_order_valid':True,'all_tasks_reach_final_gate':True,'accepted_changes':len(accepted),'acceptance_task_coverage':coverage,'local_links_checked':links,'plan_sha256':hashlib.sha256(plan.read_bytes()).hexdigest(),'task_list_sha256':hashlib.sha256(md.encode()).hexdigest(),'implementation_tests_run':False,'implementation_acceptance_passed':False}
(HERE/'validation.json').write_text(json.dumps(report,indent=2)+'\n')
print(json.dumps({k:v for k,v in report.items() if k!='acceptance_task_coverage'},indent=2))
