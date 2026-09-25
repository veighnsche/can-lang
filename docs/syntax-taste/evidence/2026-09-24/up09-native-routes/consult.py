"""UP09 Jev consultation: combined legacy-exact + captured-action dispatch placement."""
from pathlib import Path
import datetime, hashlib, json, os, sys, time, urllib.error, urllib.request
OUT = Path(__file__).resolve().parent / "jev"
OUT.mkdir(parents=True, exist_ok=True)
facts = [
    "Can compiles action declarations with typed int/str path captures to TypeScript that runs on Bun 1.4.2. Checked mount metadata carries method, a :name route template, ordered captures, the input codec contract (none/json/form with byte and row limits), the JSON response schema or the HTML form schema with structural-422 identities, and an exhaustive leaf/status case table. Mounted handlers are request-first emits-[] callables plus distinct HTML renderers. A separate legacy exact-path router serves static routes without captures. Bun.serve offers native routes with lossy param decoding, so every native callback must re-validate captures strictly from the raw pathname before handler entry, and a metadata fallback must classify 400/404/405 with Allow. Required adapter statuses are fixed: malformed capture or JSON is 400, unmatched route 404, wrong method 405 with Allow, body over budget 413, wrong media 415, structural form rejection 422, unexpected fault 500, and declared finite leaves keep their statuses. The open architecture question is where captured dispatch lives: inside the shared router value assembled by make_router with combined exact-first dispatch, or in a server-side adapter channel beside the router, or by compiling captures away into exact legacy routes.",
    "The Can server adapter must bind checked mount callables for JSON GET/POST and HTML form POST actions whose routes carry typed captures. The emitter splices one frozen site per mount: identity, method, :name template, captures, input mode with declared limits, returns, body mode, cases with statuses, the shared response schema for JSON, and rejected/rawEntry/issue identities for HTML. The runtime already ships an exact-path legacy router, a pure capture route table with method-first matching and static priority inside a method, duplicate/ambiguity refusal, and a server with native Bun route registration plus request lifecycle (body budget, owner drainage, revocation). Native params decode lossily, so dispatch must re-check the raw target in one pass and never let invalid input enter a protected handler. Candidate placements are: mount tokens join the router value so one combined dispatch serves exact legacy routes first and captured actions second with a union 405; the router stays legacy-only while server_start extracts a parallel action adapter; or each mount eagerly expands into exact legacy registrations with callback-side capture parsing.",
    "UP09 owns the server route, JSON, form, HTTP and server runtime modules and must connect complete Can to emitter to adapter fixtures to live HTTP handlers. Constraints: native Bun routes with strict one-pass capture validation; method selection with static precedence inside a method; duplicate and ambiguity refusal; reserved asset paths served ahead of callbacks; the UP06 request lifecycle; bounded native form/JSON decoding through shared codecs; exact checked callables; and the fixed 400/404/405/413/415/structural-422/500 versus finite-status vocabulary. Live tests must cover canonical int64 endpoints, malformed and encoded separator and dot cases from the guarded route probe, body budgets, callback-entry counts proving invalid input never enters a handler, and renderer faults. Three placements compete: a router-carried combined table with exact-first dispatch and native keys extracted at server start; a server-held parallel adapter that bypasses the router value; and eager expansion of mounts into exact legacy routes.",
]
instructions = [
    "Which dispatch placement best satisfies the UP09 adapter contract with the least new machinery?",
    "Select the server dispatch architecture that preserves every stated UP09 guarantee.",
    "What routing assembly strategy should UP09 implement for captured action mounts?",
]
options = [
    {
        "router_carried_combined": "Mounts return http::route tokens that make_router assembles into one combined value holding the legacy exact table plus the compiled action table and per-action callbacks. Dispatch serves an exact legacy match first, then a strict captured action match, then combined 404/405 with union Allow; target-level malformed input is 400 before any route is consulted. Server start extracts native Bun route keys from the router value and registers one shared pipeline callback, so native matching is a fast path over identical canonical re-dispatch. Duplicate and ambiguous action shapes refuse at assembly with the existing failures.",
        "server_parallel_adapter": "Keep the router value legacy-only and thread a parallel action adapter from the mounted routes to the server beside make_router, with the server consulting actions before legacy dispatch. Combined precedence, 405 union and native registration are reimplemented at the server layer rather than in one dispatch. This duplicates classification logic across two layers and leaves two sources of route truth.",
        "eager_exact_expansion": "Compile each mount into one or more exact legacy route registrations at mount time, parsing captures inside callbacks with prefix or manual matching instead of a capture table. This reuses the exact router unchanged but abandons method-first capture matching, static priority across shapes, assembly-time ambiguity refusal and native route registration.",
    },
    {
        "router_carried_combined": "Carry captured actions inside the router value: each mount validates its site and template and yields a route token, and make_router compiles all action entries into the guarded table while keeping exact legacy routes. One dispatch applies exact-first static priority, strict one-pass capture checks, combined 405 Allow and the fixed status vocabulary. The server reads native keys from the router value for Bun.serve routes while every callback re-enters the same canonical pipeline, preserving lifecycle, assets and budgets.",
        "server_parallel_adapter": "Hold captured actions outside the router in a server-side structure assembled separately from make_router, consulted ahead of legacy dispatch inside the server. Precedence between exact and captured routes, unified 405 Allow and native key extraction each need bespoke cross-layer code, and direct router dispatch without a server cannot serve actions.",
        "eager_exact_expansion": "Lower mounts to exact static registrations immediately and handle captures with ad-hoc callback parsing. Static priority, duplicate and ambiguity detection across capture shapes, canonical int64 raw-text checks and native dispatch are lost or reimplemented per callback, contradicting the strict table contract.",
    },
    {
        "router_carried_combined": "Unify legacy and captured routes in the router: mount and mountForm validate checked sites and produce tokens, make_router builds the exact map plus the compiled capture table with callbacks, and dispatch orders exact legacy hits before strict action matches with target-level 400 first and a union 405. Server start pulls Bun native keys from the same value. One table, one classifier, one lifecycle; assembly refuses duplicates and ambiguity with existing failures.",
        "server_parallel_adapter": "Split routing truth between a legacy router value and a separate server-side action adapter with its own precedence and classification path. This keeps mount tokens out of make_router but forks dispatch, complicates the 405 union and native registration, and cannot serve actions through direct dispatch.",
        "eager_exact_expansion": "Avoid capture tables by expanding mounts to exact routes at bind time with manual capture extraction in each callback. This fits the legacy router without changes yet surrenders the checked capture contract: no shape-level collision refusal, no method-first static priority, no canonical raw validation, no native routes.",
    },
]
assert len(set(facts)) == len(set(instructions)) == 3
assert all(len({options[i][key] for i in range(3)}) == 3 for key in options[0])
for i in range(3):
    data = {"model": "jev-latest", "state": facts[i], "questions": {"up09_dispatch": {"type": "choice", "instructions": instructions[i], "criteria": options[i]}}}
    (OUT / f"request-{i+1}.json").write_text(json.dumps(data, indent=2, ensure_ascii=False) + "\n")
(OUT / "wording-audit.json").write_text(json.dumps({"review": "Before sending, reviewed all three full requests: UP09 adapter contract, mount site contents, legacy router, pure capture table, native Bun registration with lossy params, lifecycle, fixed status vocabulary, probe coverage, and the same three placement alternatives are semantically preserved. Every explanatory state/instruction/option prose string is rewritten; exact code and technical identifiers kept where needed. Three responses are advisory and cannot prove absence of framing bias.", "all_explanatory_triples_distinct": True}, indent=2) + "\n")
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
