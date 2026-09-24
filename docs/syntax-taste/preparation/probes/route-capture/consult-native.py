"""Fresh Jev policy consultation using actual Bun.serve routes behavior."""
from pathlib import Path
import datetime, hashlib, json, os, sys, time, urllib.error, urllib.request
OUT = Path(__file__).resolve().parent / "jev-native"
OUT.mkdir(exist_ok=True)
facts = [
    "Can is an AI-agent language; correctness and reliable edits precede secondary measured task tokens. No external-user compatibility applies. Prefer equivalent native Bun/JS lowering with only contract adapters. Exact source actions are selected, typed int/str captures optional. A Bun 1.4.2 live `Bun.serve({routes})` experiment mounted GET `/invoices/new`, POST `/invoices/:slug`, and POST `/tenants/:tenant_id/invoices/:invoice_id`. POST `/invoices/new` invoked the dynamic POST, GET invoked the exact GET: route choice is method-first. PATCH to the invoice route fell to `fetch` and produced 404 unless adapted, although the desired contract is 405 with Allow. `%ZZ` produced a route param replacement character despite `Request.url` still containing `%ZZ`; `%2F` decoded to `/`. A raw encoded `..` invoice route invoked its own native callback with param `..`, while `Request.url` was normalized to `/invoices/7`. A small Can guard can validate native params and raw URL escapes before business code, and an unmatched fallback can synthesize 405/400 from route metadata. An alternative is a Can dispatcher in Bun's `fetch`, using URL.pathname but then the original encoded dot can reroute to another normalized action. Resolve the architecture and path/method semantics for the bounded trial.",
    "Judge a future agent-authored Can capture increment under native-first lowering and no legacy constraints. The exact action prototype already exists as a design; total successful-task tokens are secondary to sound routing. Live pinned Bun 1.4.2 `routes` tests found that verb selection precedes static specificity: POST `/invoices/new` runs `/invoices/:slug`, GET runs `/invoices/new`. A PATCH miss enters fallback with 404 by default; Can requires 405 plus Allow for a known shape. Bun decodes `%ZZ` into replacement text in params, although req.url retains bad escape bytes, and `%2F` becomes `/` in params. Notably, encoded `..` matched the `/tenants/:tenant_id/invoices/:invoice_id` callback with `tenant_id=..` before Request.url was normalized elsewhere. Thus guarded native callbacks could reject it; a separate `fetch` matcher sees only normalized `/invoices/7` and may invoke another action. Metadata can assist the native fallback with 400/405. Compare native guarded dispatch with a custom static-first matcher and a strict but deferred capture feature.",
    "The design target is Can for AI agents, with correctness first and total task tokens later, no old-source promise, and native Bun operations when equivalent. Captured `int`/`str` path segments follow the planned exact source action. Actual Bun 1.4.2 routing accepts per-method dynamic routes: exact GET `/invoices/new` and captured POST `/invoices/:slug` both serve the same pathname according to method. A PATCH invoice path reaches fallback (default 404), while the Can contract wants 405/Allow. Native parameter decoding is lossy on `%ZZ` and decodes `%2F`; however `Request.url` retains those escapes for a strict guard. For `/tenants/%2E%2E/invoices/7`, Bun called the original native invoice route with param `..`, even though Request.url showed `/invoices/7`. A callback guard can reject before protected code. A custom matcher inside `fetch` cannot reconstruct that original target and in the probe normalized it to a different action. Decide the smallest truthful routing semantics and necessary adapters, still testing 400/404/405 and ambiguity at assembly."
]
instructions = [
    "Which route-dispatch/lowering direction best preserves a bounded Can capture contract after this native evidence?",
    "Select the planned captured-action execution rule given Bun's observed method selection, lossy params and pre-normalization route choice.",
    "What implementation-plan route strategy is preferable for the optional typed-path trial with the required status and safety checks?"
]
options = [
    {
        "guarded_native_method_first": "Use Bun.serve native routes for method-first matching and static specificity within each method. Can statically rejects same-method duplicate/equally ranked overlaps. Every callback strictly validates original URL escape text and native params, typed int64/str, and rejects invalid input with 400 before the handler. The fallback consults compiled route metadata to return 405/Allow for a recognized path or 404 otherwise. A path can select a different action by method, which is explicit in action identity. Test dot aliases and never trust lossy params alone.",
        "custom_static_first": "Retain a Can route selector in Bun.serve fetch that selects a path by static specificity before method, then returns 405/Allow and validates captures. This gives a method-independent path identity but bypasses Bun's native route matcher. Request.url has already normalized escaped dot segments, so original-target rejection cannot be promised; the observed `/tenants/%2E%2E/invoices/7` can become another action.",
        "defer_captures": "Keep exact source actions and postpone typed captures until a stronger raw-target/405 integration is specified and validated. The invoice resource URL remains unavailable in this phase, but no uncertain routing contract is adopted."
    },
    {
        "guarded_native_method_first": "Compile each action into Bun's native method route table, letting verb choice precede literal-over-parameter preference. Reject ambiguous registrations at Can assembly. Add a strict callback guard for percent text, decoded parameter validity and int64 conversion, plus a fallback matcher solely for 404 versus 405 with Allow. Native params are hints, not trusted typed data. The source action's method is part of its identity, so differing verbs may name differing actions at one path.",
        "custom_static_first": "Implement a dedicated matcher behind Bun.serve fetch, ranking literal path shapes across verbs before choosing the method. It can enforce the proposed 400/405 rule from URL.pathname, but uses more custom dispatch and cannot see a raw encoded dot that native URL normalization has erased; another route may then run.",
        "defer_captures": "Do not qualify the captured route yet. Proceed with the independent exact action only and require additional platform access/evidence before allowing `/tenants/:tenant_id/invoices/:invoice_id` in source."
    },
    {
        "guarded_native_method_first": "Prefer Bun.serve routes, with an explicit method-first contract and static wins only among that verb's routes. Before action code, a small adapter inspects preserved raw escapes plus captured params, forbids unsafe strings and parses canonical signed int64, returning 400. A metadata-backed fallback supplies 405/Allow or 404; compilation rejects duplicate/ambiguous patterns. Verify all cases on the pinned target because Bun's param decoder can replace malformed bytes.",
        "custom_static_first": "Route every request through a compiler-generated Can decision table in Bun's fetch callback, selecting static over captures independently of HTTP verb and manufacturing statuses. This avoids native method-first semantics but duplicates Bun routing and cannot distinguish a request whose dot segment was normalized into another path before the callback.",
        "defer_captures": "Exclude dynamic paths from the implementation task list for now while retaining exact actions, pending proof of raw-target handling and error statuses. This sacrifices the navigable invoice resource path in the optional trial."
    }
]
assert len(set(facts)) == len(set(instructions)) == 3
assert all(len({options[i][key] for i in range(3)}) == 3 for key in options[0])
for i in range(3):
    data = {"model": "jev-latest", "state": facts[i], "questions": {"native_route_policy": {"type": "choice", "instructions": instructions[i], "criteria": options[i]}}}
    (OUT / f"request-{i+1}.json").write_text(json.dumps(data, indent=2, ensure_ascii=False) + "\n")
(OUT / "wording-audit.json").write_text(json.dumps({"review": "Before sending, reviewed all three full requests: exact action baseline and agent criteria, same three route registrations, method-first evidence, PATCH 404 gap, lossy malformed parameter decode, raw URL escape visibility, encoded-dot pre-normalization native callback, custom fetch alias, and three implementation alternatives are semantically preserved. Every explanatory state/instruction/option prose string is rewritten; exact technical identifiers remain. Three responses are advisory and cannot prove absence of framing bias.", "all_explanatory_triples_distinct": True}, indent=2) + "\n")
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
