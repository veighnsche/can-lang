from pathlib import Path
import datetime, json, os, sys, urllib.request
O=Path(__file__).resolve().parent
contexts=[
"Can serves AI coding agents, prioritizing reliable product behavior and repair, with total tokens per successful task secondary. No old source compatibility is needed. Planned server HTML action returns safe fragments and uses shallow keyed form rows; a planned Can browser target needs typed Fetch responses and immutable grid state. Current exact JSON request/response codecs exist, but the new action grammar is unimplemented. HTML `edit_outcome` includes `form::rows`, which is not JSON-admissible. Both clients must call one protected authorization/transaction operation. Native JS/Bun/Fetch should perform transport work.",
"Select a bounded client/server wire design for the agent-authored Can invoice grid. Correct status, draft retention, edit reconciliation and type linkage outweigh source brevity; measure full-task tokens later. The server-driven flow already plans an HTML form action with keyed raw fields and five rendered statuses. A separate main-thread Can frontend must send row records, receive typed data, and guard late saves. The form rows type cannot pass the existing JSON codec. Existing `http::request_json<T>` and `response_json<T>` provide a foundation. No production action feature exists, and no compatibility with earlier spellings is owed. Keep protected business checks shared.",
"The first Can browser target needs a typed grid save while the server HTMX flow needs retained raw HTML form submissions. Can is optimized for coding agents, not people, and token cost per successful change is a secondary metric. Planned HTML action bodies are fragments; its `form::rows<line_wire>` is intentionally form-only. Browser Fetch should decode finite 200/422/409/403/503 outcomes and never infer rollback from a failed response. JSON codecs already exist for transparent records/variants, with owner records crossing through explicit wire/factory conversion. Business authorization and revision checks belong to a common server operation; native platform transport is preferred. No new action implementation has shipped."
]
questions=[
"Which contract should the first Can browser grid use for typed submission and response while preserving the selected HTML form action?",
"Choose the implementable action boundary linking the browser grid to the protected invoice operation without pretending an HTML fragment decodes to a typed result.",
"What first wire/action design should engineering plan for two invoice clients, one HTML-form/HTMX and one Can-authored Fetch grid?"
]
options={
"separate_json_action":[
"Declare a second source action for JSON with a transparent row-record input, transparent finite result payload, codec/body limits and checked Fetch builder. Keep the HTML action and its raw rejection renderer. Bind both server handlers to the same protected operation. Browser mutation ID and expected revision are explicit JSON fields; server dedup/reconciliation or authorized read handles uncertain saves.",
"Use two linked but distinct action declarations: form-to-HTML for server rendering and JSON-to-JSON for the browser. JSON input/output are codec-admissible wire records/variants, and generated request/response adapters use native Fetch/Bun JSON support. Share only domain validation, actor authorization, revision write and mutation reconciliation in the server operation; retain structural form errors separately.",
"Add a bounded JSON action beside the keyed HTML action. The browser sends row objects with an operation identifier and revision; a checked response type maps finite statuses plus transport/codec failures. Both endpoints invoke one protected server function. Do not serialize form-only row handles or read an HTML fragment as a domain result; use native JSON and Fetch adapters."
],
"dual_format_action":[
"Expand one action declaration to have both HTML-form and JSON body modes and two response representations under a shared route symbol, with content negotiation. It can share identity but requires conditional codec, renderer, status, body-limit and client-adapter semantics in one declaration.",
"Create a single polymorphic action descriptor whose content type selects form/HTML or JSON/JSON at runtime, with separate typed branches and generated client helpers. This reduces declaration count but couples two decoding and rendering contracts to one route and status map.",
"Specify one source action with negotiated form and JSON variants, each with its own wire type and result representation. The compiler checks both branches and routes by content type; native adapters preserve distinct structural errors. This is a broader initial grammar and router contract."
],
"html_status_client":[
"Reuse only the HTML form action from browser code: submit URL-encoded rows, inspect status and possibly parse or mount returned fragments. Avoid JSON action machinery, but the browser receives no typed result/revision payload from that action and must obtain it elsewhere.",
"Have the Can browser grid call the existing HTML endpoint and treat response status/body as display markup. Preserve form decoder and HTMX behavior; any typed update or authoritative revision requires an extra read or parsing convention outside the action type.",
"Send browser edits through the keyed HTML action, using status codes and safe fragments as response. No separate JSON contract is added. The grid must reconcile typed state by another request or DOM extraction, so linkage between result payload and browser state is weaker."
]
}
assert len(set(contexts))==len(set(questions))==3
for k,v in options.items(): assert len(v)==len(set(v))==3
for i in range(3):
 p={"model":"jev-latest","state":{"context":contexts[i]},"questions":{"wire_action":{"type":"choice","instructions":questions[i],"criteria":{k:v[i] for k,v in options.items()}}}}
 (O/f"request-{i+1}.json").write_text(json.dumps(p,indent=2)+"\n")
(O/"wording-audit.json").write_text(json.dumps({"manual_review":"All three full requests independently rephrase context, question and every option. They preserve AI-agent audience, secondary whole-task token cost, no compatibility, current JSON codecs, planned HTML form-only rows, browser typed state, shared protected operation, native lowering and all three alternatives. Identifiers are retained literally. No earlier Jev answer is included.","mechanical_check":"Three distinct context, instruction and per-option strings.","limit":"Semantic comparison is an engineering check, not proof against wording bias."},indent=2)+"\n")
if '--send' in sys.argv:
 key=os.environ['TYPESAFE_API_KEY']
 for i in range(1,4):
  request=urllib.request.Request('https://api.typesafe.ai/v1/systemone',data=(O/f'request-{i}.json').read_bytes(),headers={'Content-Type':'application/json','Authorization':'Bearer '+key})
  started=datetime.datetime.now(datetime.timezone.utc).isoformat()
  with urllib.request.urlopen(request,timeout=60) as response: raw,status=response.read(),response.status
  (O/f'response-{i}.json').write_bytes(raw+b'\n')
  (O/f'response-{i}.metadata.json').write_text(json.dumps({'startedAt':started,'status':status,'endpoint':'v1/systemone'},indent=2)+'\n')
  parsed=json.loads(raw)
  print(json.dumps({'request':i,'model':parsed.get('model'),'answers':parsed.get('answers'),'usage':parsed.get('usage')}),flush=True)
