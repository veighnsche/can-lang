from pathlib import Path
import datetime,json,os,sys,urllib.request
O=Path(__file__).resolve().parent
contexts=[
"Can is for AI coding agents; correct and repairable behavior leads, complete-task tokens are secondary. No compatibility is needed. A planned main-thread Can grid sends typed JSON saves and keeps in-view drafts; uncertain commits require authorized read or same-ID replay. HTML form and JSON save actions share protected invoice authorization. The browser must first acquire a typed invoice snapshot and later re-read it. Existing action proposal has POST with body and finite status cases, but no read contract. Native Bun/Fetch and exact shared JSON codecs should do transport work.",
"Choose a first typed invoice-load and reconciliation boundary for a Can-authored browser grid. The target is agent-first Can, no old spellings, and native JS/Bun/Fetch lowering. Planned JSON POST save has revision, operation ID and five typed outcomes; a durable replay ledger handles identical attempts. The grid still needs authoritative initial state and a fresh authorized snapshot after uncertainty. Server HTML bootstrap is a distinct wire context; current action declaration syntax has no GET mode yet. Avoid granting access merely because a typed path exists. Whole-task tokens are measured only after correctness.",
"The proposed Can browser frontend needs to obtain invoice rows, revision and totals before editing and to reconcile uncertain saves without assuming rollback. A separate JSON save action is planned, while HTML form handling remains its own action. Shared strict codecs and transparent wire records exist; owner data requires factory/projection conversion. Authentication, tenant and resource checks must run on every read and write. Can is optimized for AI agents rather than human syntax and has no compatibility obligation. Compare bounded native-backed read designs; no production action implementation has shipped."
]
questions=[
"Which initial read/reconciliation contract should accompany the browser JSON save action?",
"Select the narrowest checked source/API design for first load and later authorized re-read of the grid.",
"What read boundary should engineering specify so the Can grid can start and resolve uncertain saves with typed data?"
]
options={
"get_json_action":[
"Add a bounded source GET JSON action with typed captures, no request body, exact codec-admissible snapshot result and finite 200/403/503 cases. A checked browser read builder uses native Fetch; server mount performs authorization for every read. Use this same action for initial load and explicit reconciliation. Keep operation-ID POST replay separate.",
"Declare a distinct GET action for an authenticated invoice snapshot, carrying the same typed path key but no input body. Its finite JSON response includes revision and row records, with generic denial/unavailable cases. Browser initial load and uncertainty reconciliation call it through a checked Fetch request. Native Bun routing and Fetch implement transport; no typed ID grants permission.",
"Extend the bounded action grammar with `get` and `input none` for JSON output. The browser gets one checked snapshot action for both startup and authorized reread, preserving strict status and codec behavior. Server rechecks actor, tenant and resource. The save remains POST with its own mutation replay contract."
],
"html_bootstrap_plus_read_post":[
"Embed an initial typed JSON snapshot in server-rendered safe HTML, then add a POST read-only JSON action for reconciliation using a small request record. This avoids GET grammar but requires safe bootstrap/CSP handling and two load paths with parity checks.",
"Start from JSON data serialized into an approved HTML data island and use a POST JSON query action for later reread. The browser decodes both with the same codec. It can work but introduces separate bootstrap and Fetch transfer mechanisms plus cache/CSRF details.",
"Use a server-generated typed bootstrap payload in the HTML page, while uncertainty resolution posts a JSON read request. Reuse POST-only action syntax; specify how safe HTML and browser asset find and verify the initial payload. This adds two mechanisms for the same snapshot."
],
"save_post_replay_only":[
"Make the browser obtain initial values from form DOM controls, then rely solely on identical-ID POST replay after uncertain saves. Do not add an authorized typed read action; typed initial state and independent reconciliation remain tied to DOM serialization and replay retention.",
"Read initial rows/revision out of server-rendered form controls and use the durable save ledger as the only uncertainty resolution path. No distinct read contract is added, but the grid has no general authoritative refresh after another writer changes the invoice.",
"Avoid new read source syntax: parse current page inputs as the starting draft and resubmit the same operation ID if a response is lost. This keeps one JSON action but weakens explicit fresh-read reconciliation and typed initial snapshot linkage."
]}
assert len(set(contexts))==len(set(questions))==3
for v in options.values(): assert len(set(v))==3
for i in range(3):
 p={'model':'jev-latest','state':{'context':contexts[i]},'questions':{'grid_read':{'type':'choice','instructions':questions[i],'criteria':{k:v[i] for k,v in options.items()}}}}
 (O/f'request-{i+1}.json').write_text(json.dumps(p,indent=2)+'\n')
(O/'wording-audit.json').write_text(json.dumps({'manual_review':'Each whole request freshly rewrites context, instruction and all option descriptions. Preserved facts: agent-first/secondary tokens/no compatibility, typed JSON POST save, initial load plus uncertain reconciliation, protected read authorization, exact codecs and native lowering. Alternatives are GET JSON action, HTML bootstrap with POST read, or DOM and replay only. No previous Jev answer is present.','mechanical_check':'All prose triples are distinct; technical identifiers and alternatives stay fixed.','limit':'This equivalence audit does not prove absence of framing bias.'},indent=2)+'\n')
if '--send' in sys.argv:
 key=os.environ['TYPESAFE_API_KEY']
 for i in range(1,4):
  req=urllib.request.Request('https://api.typesafe.ai/v1/systemone',data=(O/f'request-{i}.json').read_bytes(),headers={'Content-Type':'application/json','Authorization':'Bearer '+key})
  start=datetime.datetime.now(datetime.timezone.utc).isoformat()
  with urllib.request.urlopen(req,timeout=60) as res: raw,status=res.read(),res.status
  (O/f'response-{i}.json').write_bytes(raw+b'\n')
  (O/f'response-{i}.metadata.json').write_text(json.dumps({'startedAt':start,'status':status,'endpoint':'v1/systemone'},indent=2)+'\n')
  v=json.loads(raw)
  print(json.dumps({'request':i,'model':v.get('model'),'answers':v.get('answers'),'usage':v.get('usage')}),flush=True)
