"""Separate form-identity judgment omitted from the row-representation question."""
import datetime, hashlib, json, os, sys, time, urllib.request
from pathlib import Path
OUT=Path(__file__).resolve().parent
contexts=[
"A form's control names, retained raw values and error paths are separately spelled today. Named helpers can centralize them but do not supply compiler-linked rename diagnostics. Candidates add field references derived from the shallow wire record, or an action-linked contract additionally declaring decoder, validator and response cases. Renaming a domain property is distinct from renaming its form-wire field. The row representation question does not decide these links. No comparative field-rename trial has run.",
"Today the same form field is named independently in controls, preserved input and validation errors. Shared render/validation helpers reduce duplication without proving that a compiler finds every broken rename. A proposal can derive checked references from the flat wire schema; a larger one links those references with the action's declared decoder, validation and output cases. Domain-field and wire-name changes are separate edits. Choosing row encoding does not choose this identity mechanism, and a controlled rename comparison is still missing.",
"Control markup, rejected-value retention and error addressing currently use independent field spellings. Factored helpers are the baseline but offer no guarantee of complete static rename checking. The alternatives are wire-record-derived field identities or a linked action/form declaration covering those identities plus decoding, validation and result cases. A domain member's rename need not rename the wire control. Row codecs and field linkage require distinct judgments; no measured rename experiment has yet distinguished the linking options."
]
instructions=[
"For DI-09b, which first prototype most cleanly tests compiler detection of wire-field/control/retained-value/error-path drift against the best validation/render library, before larger action-form coupling is justified?",
"What DI-09b increment should first be compared with shared validation/render helpers to isolate checked field-reference value across controls, retained inputs and errors without presupposing a whole linked-action mechanism?",
"Select the initial DI-09b field-identity experiment for measuring static repair of wire-name drift in controls, retained raw data and error paths relative to factored helpers, while larger action integration remains open."
]
options={
'A':[
"Strengthen only shared validation/render helpers and rename tests, postponing compiler field references until a concrete baseline failure is reproduced.",
"Improve the library's shared form render/validation functions and controlled rename checks first; defer static identities pending a demonstrated shortcoming.",
"First measure enhanced helper composition and rename tests, adding no compiler-linked field model until the baseline shows an actual failure."
],
'B':[
"Prototype checked field references derived from the wire record that connect controls, retained raw values and error paths, keeping validation explicit.",
"Test wire-schema-derived static identities across control names, preserved inputs and field errors, with ordinary explicit domain validation.",
"Add a trial of checked wire-field links joining control naming, raw-value retention and error addressing while leaving validation as explicit application logic."
],
'C':[
"Prototype the full action-linked form contract together: wire field references, declared decoder, validator and response cases.",
"Build field identity inside a joint action/form declaration that also fixes its wire decoder, domain validator and outcome variants.",
"Evaluate integrated action/form linkage from the outset, covering wire fields plus named decoding, validation and response-case contracts."
],
'D':[
"The evidence cannot yet prioritize these experiments; collect a missing discriminator first.",
"Leave the comparison order open until an absent observation distinguishes the candidates.",
"Request a focused additional measurement because the supplied facts do not support ordering these alternatives."
]}
requests=[]
for i in range(3):
 base=json.loads((OUT/f'request-{i+1}.json').read_text())
 state={k:base['state'][k] for k in ['constraints','evidence_status','forms']}
 state['field_identity']=contexts[i]
 keys=list(options);keys=keys[i:]+keys[:i]
 body={'model':'jev-latest','state':state,'questions':{'form_field_links':{'type':'choice','instructions':instructions[i],'criteria':{k:options[k][i] for k in keys}}}}
 requests.append(body)
 (OUT/f'field-links-request-{i+1}.json').write_text(json.dumps(body,indent=2)+'\n')
for field in requests[0]['state']:
 assert len({r['state'][field] for r in requests})==3
assert len(set(instructions))==3
assert all(len(set(v))==3 for v in options.values())
audit={'before_send':True,'checked_explanatory_paths':9,'identical_explanatory_paths':[],'semantic_review':'Reviewed all three state/question/option wordings for identical baseline, missing evidence, rename target and A/B/C/D alternatives. Context includes the already prepared independently reworded evidence for this separate judgment. No responses from earlier questions are included. Option order rotates. Field identity is separated from line-row encoding so independent judgments are not bundled.','limits':'Uniqueness does not prove semantic equivalence or remove shared framing bias.'}
(OUT/'field-links-wording-audit.json').write_text(json.dumps(audit,indent=2)+'\n')
if '--send' in sys.argv:
 for i in range(1,4):
  body=(OUT/f'field-links-request-{i}.json').read_bytes()
  started=datetime.datetime.now(datetime.timezone.utc).isoformat();clock=time.monotonic()
  req=urllib.request.Request('https://api.typesafe.ai/v1/systemone',data=body,headers={'Content-Type':'application/json','Authorization':'Bearer '+os.environ['TYPESAFE_API_KEY']})
  with urllib.request.urlopen(req,timeout=90) as r: raw,status=r.read(),r.status
  (OUT/f'field-links-response-{i}.json').write_bytes(raw)
  data=json.loads(raw)
  meta={'started_at':started,'elapsed_seconds':round(time.monotonic()-clock,3),'status':status,'endpoint':'https://api.typesafe.ai/v1/systemone','request_sha256':hashlib.sha256(body).hexdigest(),'response_sha256':hashlib.sha256(raw).hexdigest(),'requested_model':'jev-latest','returned_model':data.get('model'),'usage':data.get('usage'),'fresh_call':True}
  (OUT/f'field-links-response-{i}.metadata.json').write_text(json.dumps(meta,indent=2)+'\n')
  print(json.dumps({'request':i,**data}),flush=True)
