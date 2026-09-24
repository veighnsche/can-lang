"""Fresh Jev comparison for repeated form rows and first browser DOM architecture."""
from pathlib import Path
import datetime
import json
import os
import sys
import urllib.request

OUT = Path(__file__).resolve().parent
objective = [
    "Can is designed for AI coding agents, not human comfort. Correct behavior and dependable edits lead, with total tokens per successful coding task measured second. Old syntax needs no compatibility and generated TS should delegate equivalent work to native JS/Bun/browser APIs. Choose bounded contracts for the full server and Can-authored browser invoice grid, not implemented features.",
    "Evaluate two Can product-design decisions for AI-agent-authored applications. Prefer secure, observable semantics and repairable contracts; account for complete prompt/code/diagnostic/retry tokens only after correctness. No external users require preserving old source. Native JavaScript, Bun and browser operations should do equivalent work, with small adapters for immutability and Can rules. These options are planning candidates.",
    "The intended Can programmer is an AI coding agent even if syntax is harsh to people. Rank reliable product behavior and agent refactoring above measured whole-task token cost. There is no backwards-compatibility duty. Lower operations to native JS/Bun/DOM/Fetch instead of recreating their engines, adding only necessary language-contract adapters. Select a first implementable grid contract, not a claim of passing tests."
]
row_facts = [
    "The invoice edit requires repeated line rows, stable association of each ID with quantity and price, retained rejected text, and precise field errors after reorder or partial input. Current Can form decoding admits only shallow str, optional str and str[] with strict duplicate-scalar/unknown-field rules. Three earlier Jev trials favored a checked keyed-row experiment (.79/.63/.52), but the last nearly tied validated parallel arrays (.42). A length check on parallel id[], quantity[] and price[] cannot detect equal-length cross-row omissions: with three rows, omit row B's id, row C's quantity and row A's price; each column has two elements but zipping misassociates surviving values. A separate JSON form action is not ordinary HTML form decoding and may require browser code. No new row decoder has been built or agent-benchmarked.",
    "The server-driven invoice form and later Can grid need multiple editable lines without shifting an amount onto another item when controls are absent or reordered. Present form wire types are flat str/optional str/str[]; unknown fields and duplicate scalar names fail. Earlier three Jev requests ranked checked keyed row decoding above arrays, though one was close. Parallel id/quantity/price arrays can all retain equal count after different rows omit one control each, yielding type-valid but false pairings; a count comparison alone cannot identify the original row. JSON submissions are a separate codec/action mode rather than current form behavior. No candidate has production or comparative agent evidence.",
    "A required invoice workload submits repeated rows and must preserve row identity, raw invalid values and error location through moves and missing fields. Current Can parses only shallow string-oriented form members. Previous Jev advice for a keyed row trial was .79, .63 and .52, versus .42 for parallel arrays in one wording. Equal-length arrays are not sufficient: three rows may lose an ID in B, quantity in C and price in A, leaving two values per array that zip into incorrect records. A JSON body could carry row objects but is not an HTML form decoder and needs distinct client authoring. Neither keyed syntax nor agent benefit is measured.",
]
browser_facts = [
    "The broad goal requires a separately compiled Can main-thread browser grid: keyboard navigation, immutable draft transitions, exact totals, optimistic save/rollback, Fetch, safe rendering, offline in-view editing and view disposal. Current Can runs on Bun with safe server HTML/HTMX and forbids authored browser scripts; no browser target exists. Three prior Jev requests favored a bounded main-thread target. Native DOM APIs create/update nodes and register listeners; Fetch handles requests. Server-only SQL, process, secrets and file capabilities must be rejected transitively, and shared codecs must match Bun for int64/finite floats, variants and strict fields. An application-owned identified save must remain accountable after view listeners dispose. No framework/reactivity or worker requirement was stated.",
    "Can currently compiles Bun/server output and allows only pinned HTMX execution in browsers. The target product is an actual Can-authored editable invoice grid, so a typed bridge to another language would prove a narrower claim. Prior Jev advice consistently chose a main-thread browser slice. It needs DOM events, controlled text/structure rendering, Fetch, immutable state updates, keyboard/focus behavior and explicit disposal. Native browser primitives should implement these. SQL/process/secret modules cannot enter the bundle through direct calls, imports, callbacks or generic specialization; server and browser codecs must agree on exact numbers and data. A view owner may dispose listeners, while a tracked app-level save still needs settlement or reconciliation.",
    "For the full Can frontend gate, a browser-target build must run the invoice grid itself rather than serving only HTML/HTMX or generated foreign client calls. The repository has no Can browser execution now; previous three Jev choices favored main-thread scope. The first grid needs typed shared wire data, native DOM/event/Fetch access, pure draft calculations, optimistic response handling, offline in-view state and cleanup. Capability checking must exclude server secrets, filesystem, processes and SQL transitively and preserve codec parity, including int64. View disposal cannot silently abandon an identified mutation. There is no requirement for a general reactive framework, virtual DOM, worker execution or a new statement language.",
]
questions = {
    "line_row_contract": {
        "instructions": [
            "Which first HTML form row contract should the linked invoice action use to prevent cross-row misassociation under the stated partial-input case?",
            "Choose the bounded repeated-row wire representation for the server-driven invoice form given the concrete equal-length omission hazard.",
            "What row decoding rule should engineering specify for the invoice action so reorder and missing controls retain exact ownership?",
        ],
        "criteria": {
            "keyed_rows": [
                "Extend only the shallow form boundary with checked `rows<line_wire>` keyed field names such as lines[key].quantity; group by a stable row key, reject duplicate keys/fields, partial rows and unknown nested names, and retain per-key raw values/errors. Do not infer domain numbers or authorization. Compile to native FormData iteration plus validation adapter.",
                "Add a bounded row-group decoder to the action form contract, where each control carries its stable row identifier and typed direct field reference. Missing or duplicated members fail per row; reordering does not change associations; original text and key-specific problems remain available. Use native FormData parsing with strict Can checks, no general nested object grammar.",
                "Select a checked keyed row form extension: row IDs live in each wire field path, and the decoder collects one row record per key with exact member set, duplicate/unknown rejection and stable error paths. Preserve text values for invalid input; domain conversion stays explicit. Do not turn this into arbitrary nested forms or an ORM."
            ],
            "parallel_arrays": [
                "Keep shallow id[], quantity[] and price[] string arrays; verify equal lengths and unique IDs, then zip positions. Preserve raw arrays and surface errors. Accept that equal-length omissions in different rows can still pair unrelated values unless an additional association code is introduced; no new row decoder ships.",
                "Retain current repeated-array fields and application length/ID validation. Align columns by position and reject detected mismatch, while explicitly admitting that the supplied cross-row omission case can evade count checks without extra keyed data. Avoid new form syntax initially.",
                "Use the existing three parallel str[] columns with length and duplicate-ID checks, matching items by index. Record the unresolved case where missing controls leave all arrays equal in size but shift meanings. Rely on browser/protocol tests rather than a new keyed wire mechanism."
            ],
            "json_action": [
                "Move line edits to a separate JSON action with an ordinary array of line records and exact existing codec rules; abandon standard HTML form submission for these rows. A Can browser client must serialize the JSON payload, and the server still retains invalid field text by its own model.",
                "Use a distinct JSON endpoint for line rows, with array records preserving ID/quantity/price together under the shared codec. This requires authored browser behavior rather than the server-driven HTML form baseline and does not claim JSON is form decoding.",
                "Represent each line as a JSON record in a separate typed client action, so arrays of rows cannot misalign fields. Change the server-driven form workflow accordingly and maintain its own raw-input/error rendering; no keyed form extension is built."
            ],
        },
    },
    "browser_dom_architecture": {
        "instructions": [
            "Which first browser-target authoring/runtime architecture satisfies the Can-authored grid with the smallest justified machinery and native lowering?",
            "Select a bounded main-thread Can browser execution model for the required grid, considering capability safety, native DOM behavior and explicit lifetime.",
            "What browser programming surface should engineering plan first for the actual Can-authored invoice grid, avoiding an unproved general framework?",
        ],
        "criteria": {
            "explicit_dom": [
                "Compile ordinary Can named functions and immutable records for draft logic, with a small browser-only catalogue for native DOM element/text/property updates, listener registration, focus and Fetch. Use opaque view-scoped listener/resource ownership and an app-scoped identified-save owner; explicit state transitions live in Can, with a bounded state-cell adapter if needed. No virtual DOM, implicit effects or worker profile.",
                "Add a separate browser profile and direct typed wrappers over document.createElement, textContent, event listeners, focus and fetch. The application writes named transition/render functions over immutable draft records, while opaque view ownership releases listeners and app ownership tracks in-flight mutations. Keep any mutable state cell an explicit adapter rather than a general language mutation feature.",
                "Choose normal Can code plus a browser catalogue mapping to native DOM/event/Fetch APIs. State changes are explicit immutable values stored by a scoped opaque cell; views own event/timer handles and an application owner retains identified saves. Avoid a renderer engine, reactive dependency graph, new source grammar and worker support."
            ],
            "reducer_runner": [
                "Introduce a typed application runner over State and Event, with declared pure update, safe-node render and effect-command functions; the runtime owns dispatch, DOM reconciliation and command scheduling. This gives one controlled state loop but adds a general runner/reconciliation engine beyond native DOM wrappers.",
                "Build a source-level reducer/render/command contract and runtime that queues events, diffs safe views into DOM nodes and interprets Fetch/timer commands. Explicit state is maintained through the runner; it centralizes lifecycle but requires a new framework-like execution layer and effect vocabulary.",
                "Plan an Elm-like typed runner: Can functions map state plus event to next state and commands, render a safe tree, and a browser runtime reconciles DOM and handles effects. It can structure the grid, but introduces a new render engine and command scheduler rather than direct DOM operations."
            ],
            "foreign_bridge": [
                "Generate typed wire/client contracts for a separately authored JavaScript/TypeScript grid and keep Can on the server. This avoids implementing browser Can but satisfies only a narrower integration claim, not the requested Can-authored frontend gate.",
                "Use Can server code and an external browser language consuming generated types/actions. It may be practical for a product yet does not make the editable frontend Can-authored, so the broad recommendation would remain unqualified.",
                "Defer Can browser execution and expose typed action/codec bindings to a foreign client implementation. Keep the grid functional, but state plainly that the full Can-backend-and-frontend objective is unmet."
            ],
        },
    },
}
for triple in (objective, row_facts, browser_facts):
    assert len(triple) == 3 and len(set(triple)) == 3
for q in questions.values():
    assert len(q["instructions"]) == 3 and len(set(q["instructions"])) == 3
    for triple in q["criteria"].values():
        assert len(triple) == 3 and len(set(triple)) == 3
for i in range(3):
    payload = {"model": "jev-latest", "state": {"objective": objective[i], "row_evidence": row_facts[i], "browser_evidence": browser_facts[i]}, "questions": {k: {"type": "choice", "instructions": q["instructions"][i], "criteria": {label: texts[i] for label, texts in q["criteria"].items()}} for k, q in questions.items()}}
    (OUT / f"request-{i+1}.json").write_text(json.dumps(payload, indent=2, ensure_ascii=False) + "\n")
(OUT / "wording-audit.json").write_text(json.dumps({"manual_review": "Compared all three complete requests before sending. They preserve the agent-first/native/no-compatibility goals; current shallow form limits, previous Jev row advice and concrete equal-length omission counterexample; first Can browser scope, current absence, native and capability constraints; all three alternatives for each independent question. Explanatory state, instructions and criteria are freshly written; exact identifiers and option keys stay fixed. No prior response or preferred answer was supplied.", "mechanical_check": "All explanatory text triples contain three distinct full strings.", "limit": "Manual semantic equivalence is not a proof of no framing bias."}, indent=2) + "\n")
if "--send" in sys.argv:
    key = os.environ["TYPESAFE_API_KEY"]
    for i in range(1, 4):
        req = urllib.request.Request("https://api.typesafe.ai/v1/systemone", data=(OUT / f"request-{i}.json").read_bytes(), headers={"Content-Type": "application/json", "Authorization": "Bearer " + key})
        started = datetime.datetime.now(datetime.timezone.utc).isoformat()
        with urllib.request.urlopen(req, timeout=60) as response:
            raw, status = response.read(), response.status
        (OUT / f"response-{i}.json").write_bytes(raw + b"\n")
        (OUT / f"response-{i}.metadata.json").write_text(json.dumps({"startedAt": started, "status": status, "endpoint": "v1/systemone"}, indent=2) + "\n")
        parsed = json.loads(raw)
        print(json.dumps({"request": i, "model": parsed.get("model"), "answers": parsed.get("answers"), "usage": parsed.get("usage")}), flush=True)
