"""Fresh three-way Jev consultation after live dot-segment rerouting evidence."""
from pathlib import Path
import datetime, hashlib, json, os, sys, time, urllib.error, urllib.request

OUT = Path(__file__).resolve().parent / "jev-dot"
OUT.mkdir(exist_ok=True)
facts = [
    "Can targets AI coding agents; correctness and reliable edits precede measured total task tokens. There are no external users or old-route compatibility duties. Native Bun/JavaScript lowering is preferred with adapters only where required. The chosen source action first has an exact path; typed captures are a separate optional increment. On Bun 1.4.2, a live curl `--path-as-is` request to `/tenants/%2E%2E/invoices/7` reached `Bun.serve` as `/invoices/7`. In a trial router also mounting POST `/invoices/:slug`, that normalized URL successfully invoked the latter action with slug `7`. Neither `Request.url` nor `URL.pathname` contains the original dot segment. The generated builder can prohibit `.`/`..`, and each handler must authorize its own operation; source action identity cannot claim to recover the original request target. Other malformed escapes remain available for strict 400 handling. Decide how to specify this platform fact before implementation.",
    "The future language is optimized for agent-written correct software, with complete-task token use a second metric and no migration obligations to outside adopters. Generated TS should call native Bun where equivalent, using small Can contract guards. Exact source actions are already selected; captured paths can be gated independently. A Bun 1.4.2 HTTP experiment sent `/tenants/%2E%2E/invoices/7` without curl path cleanup, yet the `fetch` callback saw `/invoices/7`. Another POST action at `/invoices/:slug` then matched and received `slug=7`. The callback has no raw request-target bytes through native `Request.url`, so it cannot tell this from a direct `/invoices/7` request. A route builder can reject dot-valued segments, while protected handlers still enforce authorization. Bad percent/UTF-8 is separately visible for 400. Select the truthful capture boundary.",
    "We are specifying an optional typed route feature for Can, an agent-first language. Sound behavior outranks any later agent-token savings; compatibility with old callers is unnecessary. Equivalent operations should lower to Bun/JS rather than a new HTTP stack when possible. The exact source-action design proceeds independently. In pinned Bun 1.4.2, the native server normalizes a raw percent-encoded dotdot segment before exposing the URL. Sending `/tenants/%2E%2E/invoices/7` produced the same callback URL as `/invoices/7`; a separate mounted POST `/invoices/:slug` executed. A Can matcher based on `Request.url` cannot reconstruct the sender's initial spelling. Builders can refuse `.` and `..`, and route handlers need resource/tenant checks regardless. Malformed percent sequences are still detectable. Choose the design promise for the capture trial given this observed alias."
]
instructions = [
    "Which honest implementation-plan disposition best fits the live Bun normalization result while preserving the native lowering preference?",
    "How should the planned captured action describe inbound dot-segment behavior after seeing the concrete cross-route normalization?",
    "Select the bounded route policy or gate that follows from the observed loss of the raw target in Bun's Request API."
]
options = [
    {
        "normalized_contract": "Define route identity over the normalized URL path that Bun supplies. Builder rejects dot segments and produces canonical resource links; inbound raw or escaped dot navigation may become a different normalized route, so do not promise a 400/404 for the original spelling. Every selected action authorizes independently; protocol tests document cross-route normalization. Keep strict 400 for malformed encoding still visible in Request.url.",
        "gate_raw_target": "Do not implement typed captures until the chosen Bun integration can read and reject the original request target before URL normalization. Continue exact source actions meanwhile. This prevents the observed alias at the cost of delaying capture routes and possibly requiring a platform feature unavailable in native Request.",
        "custom_http": "Build a lower-level HTTP request-target parser/server to inspect raw bytes, reject dot traversal before normalization, then run action matching. This can give a raw-target 400 promise, but replaces Bun.serve request parsing and adds substantial protocol/security work beyond a bounded capture increment."
    },
    {
        "normalized_contract": "Promise matching of Bun's post-normalization pathname only, openly stating that `%2E%2E` can reroute before Can sees it. The checked URL builder refuses dot captures and all handlers retain their own authorization. A direct raw spelling is not an endpoint-identity guarantee; continue rejecting visible malformed encoding as 400 and test the alias.",
        "gate_raw_target": "Hold the separate capture increment until raw request-target access exists in the selected native server path, allowing a pre-normalization dot rejection. Ship planning for exact-path actions first. The stronger guarantee remains unimplemented until that ingress evidence exists.",
        "custom_http": "Use a custom byte-level HTTP ingress instead of Bun.serve's Request URL so the raw dot escape can be rejected. This gains the stronger inbound route promise, while enlarging the HTTP implementation and deviating from preferred native Bun operations."
    },
    {
        "normalized_contract": "Scope captured route dispatch to the URL after Bun normalization. Canonical action builders cannot create dot segments; clients may still submit one that Bun maps elsewhere, and the original spelling cannot be diagnosed. Enforce permissions in the reached operation and retain 400 on bad input that remains observable. Qualify this behavior with HTTP tests.",
        "gate_raw_target": "Treat original-target rejection as essential: postpone typed routes while exact actions proceed, pending a proven Bun API or adapter that exposes the original target before dot normalization. No claim of capture support is made until then.",
        "custom_http": "Implement a raw HTTP target layer that parses requests before WHATWG URL normalization, blocks the dot alias and forwards admitted requests to actions. This supplies strict raw-path rejection but introduces a broad non-native server/parser responsibility."
    }
]
assert len(set(facts)) == len(set(instructions)) == 3
assert all(len({options[i][key] for i in range(3)}) == 3 for key in options[0])
for i in range(3):
    data = {"model": "jev-latest", "state": facts[i], "questions": {"dot_normalization": {"type": "choice", "instructions": instructions[i], "criteria": options[i]}}}
    (OUT / f"request-{i+1}.json").write_text(json.dumps(data, indent=2, ensure_ascii=False) + "\n")
(OUT / "wording-audit.json").write_text(json.dumps({"review": "Manually checked each complete request before sending: same agent-first criteria, native preference, selected exact action, optional captures, live Bun cross-route dot normalization, inability to recover raw target, builder guard, independent authorization and visible malformed escape handling. Three alternatives have identical obligations and tradeoffs. All explanatory prose in state, instruction and option descriptions is rewritten. Stable technical strings and keys retain exact meaning. Distinct wording is not proof against framing bias.", "all_explanatory_triples_distinct": True}, indent=2) + "\n")
if "--send" in sys.argv:
    key = os.environ["TYPESAFE_API_KEY"]
    for i in range(1, 4):
        body = (OUT / f"request-{i}.json").read_bytes()
        for attempt in range(1, 4):
            started = datetime.datetime.now(datetime.timezone.utc).isoformat()
            req = urllib.request.Request("https://api.typesafe.ai/v1/systemone", data=body, headers={"Content-Type": "application/json", "Authorization": "Bearer " + key})
            try:
                with urllib.request.urlopen(req, timeout=60) as response:
                    raw, status = response.read(), response.status
                (OUT / f"response-{i}.json").write_bytes(raw + b"\n")
                (OUT / f"response-{i}.metadata.json").write_text(json.dumps({"startedAt": started, "status": status, "attempt": attempt, "endpoint": "v1/systemone", "requestSha256": hashlib.sha256(body).hexdigest(), "responseSha256": hashlib.sha256(raw).hexdigest()}, indent=2) + "\n")
                data = json.loads(raw)
                print(json.dumps({"request": i, "model": data.get("model"), "answers": data.get("answers"), "usage": data.get("usage")}), flush=True)
                break
            except urllib.error.HTTPError as error:
                (OUT / f"response-{i}.attempt-{attempt}.error.json").write_bytes(error.read() + b"\n")
                if error.code not in (429, 529) or attempt == 3: raise
                time.sleep(2 ** attempt)
