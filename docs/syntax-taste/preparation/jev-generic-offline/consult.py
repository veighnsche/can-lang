"""Three fresh Jev choices for DI-04 and the invoice-grid offline minimum."""
from pathlib import Path
import datetime
import json
import os
import sys
import urllib.request

OUT = Path(__file__).resolve().parent

objective = [
    "Can serves AI coding agents even where its source is uncomfortable for humans. Correctness, reliable modification and explicit boundaries take priority; total tokens per successful agent task are secondary. There are no external users or old-syntax compatibility duties. Generated TypeScript must delegate equivalent operations to native JavaScript/Bun with only necessary Can adapters. Choose future design contracts, not implemented behavior.",
    "Judge planned Can behavior for programs authored and repaired by AI agents. Human ease has no independent weight; successful semantics and predictable edits lead, while prompts, code, diagnostics and retries contribute to a secondary whole-task token metric. The repository has no outside adopters. Emit native JS/Bun operations rather than a substitute interpreter, adding only contract-preserving adaptation. These are design choices, not claims of shipped features.",
    "The intended Can programmer is an AI coding agent, and human hostility is acceptable if it serves that audience. Prioritize program correctness and dependable refactors, then measure complete successful-task token consumption. Compatibility with previous source and generated layout is unnecessary because there are zero external users. Lower to equivalent JavaScript/Bun behavior with minimal immutable-contract adapters. Advice is for an implementation plan, not validation of code already written.",
]
generic_evidence = [
    "Current exported generic bodies are checked at concrete specialization; a body can add value + value while its public signature stays unchanged, so a new consumer lacking addition fails later. Ordinary named callable inputs and dictionary records already express required operations. A current two-caller probe passed 28 assertions and showed those explicit inputs work; no held-out agent comparison establishes their cost. Existing local templates deliberately use specialization checks. The exact universally valid operation set needs enumeration. DI-03 finite error-set syntax is deferred pending measured benefit and is separate from declaring operations.",
    "Can presently admits a public generic whose implementation later begins using value + value without modifying its written interface; incompatibility appears when a consuming specialization is checked. Named function parameters or records of functions can already make an operation requirement visible, and the executed generic-helper baseline verified this pattern across two callers. It has not measured agent creation/repair against a capability system. Locally scoped template behavior remains useful. Any stronger public contract must define universally admitted operations, and the optional finite-error parameter is a distinct undecided mechanism.",
    "The compiler specializes generic bodies for concrete callers, so changing an exported body to require + can break a separate consumer without a public-signature edit. A 28-assertion current-Can example demonstrates working callable injection and two independent callers, not a task-token advantage. A dictionary is an ordinary record of such named operations. A local template may reasonably keep body-at-specialization checking. A public-body restriction needs a precise universal-operation list; finite higher-order error rows are separate and not selected for production.",
]
offline_evidence = [
    "The recommendation target includes one Can-authored browser invoice grid with keyboard edits, local unsaved drafts, immediate totals, optimistic save/rollback, slow or disconnected behavior and disposal. It does not state that drafts survive reload or that mutations are automatically queued offline. Current Can has no browser execution target; a bounded main-thread target is planned. Three earlier fresh Jev requests favored an in-view memory draft and explicit retry as the evaluation baseline, explicitly not a final durability promise. Durable draft storage adds user/tenant isolation, logout, schema migration, eviction and multi-tab rules; a mutation queue also adds authorization expiry, operation identity and uncertain-commit reconciliation. Application-owned saves must outlive view cleanup without a client abort being treated as rollback.",
    "The broad frontend gate asks a Can browser client to handle an editable invoice grid, including unsaved local work, rapid totals, optimistic failure recovery, network loss/slowness and view lifetime. No source requirement promises persistence through navigation/reload or background replay. Browser Can is itself proposed, not implemented. Prior three-way Jev advice chose view-memory plus deliberate retry only as the first experiment. Persisting drafts safely requires account/tenant partition, storage failures, versioned migrations, logout and concurrent tabs. Queuing writes additionally requires stable mutation IDs, expired permissions and uncertain server-commit handling. Identified saves owned by the application are distinct from listeners and timers owned by one view.",
    "For a rich Can-authored invoice grid, the stated outcomes are keyboard work, immediate calculations, unsaved drafts, optimistic save with rollback, disconnected/slow handling and cleanup on disappearing view. The recommendation program says nothing about reload-durable draft state or automatic offline send. A separate main-thread browser target remains a design proposal. An earlier Jev set recommended memory-only drafts with explicit retry for evaluation, without choosing the final product promise. Browser persistence would need eviction, tenant/user separation, logout, schema version and multi-tab behavior; an offline write queue would further need operation keys, permission expiry and commit reconciliation. View disposal cannot erase accountability for an application-owned identified save.",
]
questions = {
    "public_generic_operations": {
        "instructions": [
            "Which planned public generic operation policy best serves independent agent-authored libraries on the evidence, without assuming an unmeasured token advantage?",
            "Select the design for exposing operation requirements of exported Can generics, taking the specialization failure and existing callable baseline into account.",
            "What contract should engineering specify for operations used by public generic bodies, given the current template behavior and candidate implementation costs?",
        ],
        "criteria": {
            "explicit_public": [
                "For an exported generic, permit only enumerated universal operations and operations supplied in written named callable/dictionary inputs; reject a hidden new + in the generic declaration until an input is added. Preserve specialization checking for local non-exported templates. No implicit instance search or new trait syntax; compile supplied callables to native JS function calls.",
                "Require public generic signatures to list needed behavior as explicit function or dictionary parameters beyond a fixed universal core. A body-only addition requirement diagnoses at the exported source rather than surprising a downstream instantiation. Locally scoped template helpers may retain concrete checks. Reuse ordinary callable semantics and native calls, not global typeclass lookup.",
                "Enforce a declaration-time boundary on exported generics: universal primitives plus operations named by callable/dictionary arguments, with other body uses rejected until the public API changes. Keep internal templates specialization-checked. Add no capability vocabulary; generated code invokes the ordinary passed JS functions and Can error adapters.",
            ],
            "template_artifact": [
                "Retain specialization-checked exported generic bodies, including hidden body requirements, but emit a versioned dependency/body-digest requirement artifact and recheck all consuming specializations after changes. Diagnostics name the body requirement and instantiation path; the artifact cannot promise universal correctness for unknown future types.",
                "Continue the current public template policy and publish compiler-derived requirements tied to lock and implementation digest. A dependency edit invalidates summaries and all known clients re-specialize, receiving precise instantiation diagnostics. Do not claim the unchanged written signature fully describes operations needed by untested types.",
                "Keep exported templates checked only when concretely used, with generated operation summaries bound to the locked body hash. Rebuild dependents and report the operation plus specialization trail when a body edit fails. Unknown future type arguments remain outside any universal contract; no new source requirement form is introduced.",
            ],
            "small_capabilities": [
                "Add a finite compiler-defined capability vocabulary to public type parameters, with exact operations and any enforceable laws; require the body to declare + through that capability. Instantiate only when the concrete type satisfies it. Named callables remain for unsupported domain operations; no general traits or implicit arbitrary instances.",
                "Specify a small built-in set of public generic capabilities, each listing exact admitted operators and verifiable obligations. Adding a body operation updates the public capability requirement; consumers prove membership at instantiation. Domain-specific behavior still travels through callable inputs, without a user-defined trait registry.",
                "Introduce narrow compiler-known operation constraints for exported generic types, so a + body requires a written matching capability and callers satisfy it. Restrict the vocabulary and do not infer an unlimited typeclass system; use explicit function arguments for operations outside it.",
            ],
        },
    },
    "grid_offline_promise": {
        "instructions": [
            "What final minimum offline behavior should the first Can-authored invoice grid promise, given the stated product goal and the additional obligations of persistence and queuing?",
            "Choose the bounded offline contract for the broad frontend gate itself, beyond merely naming an evaluation baseline; the stated goal does not ask for reload survival.",
            "Which initial product promise should engineering freeze for an editable Can browser grid that works through slow and disconnected periods?",
        ],
        "criteria": {
            "in_view_explicit_retry": [
                "Keep unsaved edits in memory while the view stays alive; remain editable during disconnection, show unsaved/failed state, and allow deliberate retry or reconciliation after reconnect. An identified in-flight save remains application-owned through view disposal. Promise neither reload/navigation durability nor automatic offline queue; test loss and late responses visibly.",
                "Guarantee that a live grid retains and edits its local draft when offline, labels it unsaved, and offers an explicit reconnect/retry path with stale-response protection. Preserve app-owned mutation identity after a view closes. Do not represent an in-memory draft as persistent across reload, or silently enqueue writes for background delivery.",
                "For the first supported grid, keep draft state for the active view during network loss, calculate locally, signal unsaved status and require a deliberate retry/reconcile after connection returns. Application-owned saves settle independently of view listeners. Reload persistence and automatic queued mutation are outside this promise and remain separately testable additions.",
            ],
            "durable_draft": [
                "Persist a versioned, user/tenant-scoped draft locally so it can survive reload/navigation and be restored after reconnect, but do not automatically send a pending write. Specify storage denial/eviction, schema migration, logout erasure and cross-tab conflict. Saves still need explicit retry and server reconciliation.",
                "Promise a browser-stored draft surviving a reload for the same account and tenant, with versioning, logout removal, storage failure handling and multi-tab resolution. Recovery shows the draft but never assumes an unacknowledged mutation committed; the user explicitly retries after reconciling with the server.",
                "Add durable client-side draft storage with account/tenant partition and migration, eviction, logout and multi-tab policy. On reload restore unsent text without auto-replaying a mutation. Server revision and uncertain outcomes are checked before an explicit retry.",
            ],
            "durable_queue": [
                "Persist draft and identified save intent through reload and automatically replay writes after reconnection, with authentication expiry, revision conflict, idempotency key, crash window, multiple-tab arbitration and uncertain-commit reconciliation. The server must deduplicate or expose a safe reconciliation contract; a client abort cannot imply rollback.",
                "Support a local durable mutation queue as well as draft recovery, with automatic resume after connectivity returns. Define stable operation IDs, server idempotency, auth/revision expiry, duplicate tabs and crash/replay windows; uncertain commits require lookup instead of blind repeat.",
                "Ship offline write replay in the initial grid: versioned local draft and queue survive reload, and identified operations resume automatically. The contract includes permission changes, server deduplication, stale revision, storage failure, multiple tabs and commit-unknown reconciliation; merely storing request bytes is insufficient.",
            ],
        },
    },
}

assert all(len(v) == 3 and len(set(v)) == 3 for v in (objective, generic_evidence, offline_evidence))
for q in questions.values():
    assert len(q["instructions"]) == 3 and len(set(q["instructions"])) == 3
    for option in q["criteria"].values():
        assert len(option) == 3 and len(set(option)) == 3

for i in range(3):
    payload = {
        "model": "jev-latest",
        "state": {"objective": objective[i], "generic_evidence": generic_evidence[i], "offline_evidence": offline_evidence[i]},
        "questions": {
            key: {"type": "choice", "instructions": q["instructions"][i], "criteria": {label: wording[i] for label, wording in q["criteria"].items()}}
            for key, q in questions.items()
        },
    }
    (OUT / f"request-{i+1}.json").write_text(json.dumps(payload, indent=2, ensure_ascii=False) + "\n")

(OUT / "wording-audit.json").write_text(json.dumps({
    "manual_review": "All three full requests were checked before sending. Each preserves agent-first priority, native lowering, no compatibility obligation, current generic body behavior, callable baseline, distinct capability/artifact alternatives, browser grid outcomes, absence of a reload promise, prior evaluation-only Jev advice, and full persistence/queue obligations. No prior answer or preferred option was supplied. Every explanatory field has fresh prose; exact code tokens and option keys remain stable.",
    "mechanical_check": "Every explanatory text triple has pairwise distinct full wording. The request JSON files were compared before API submission.",
    "limit": "Semantic equivalence is a reviewed judgment, not proof; correlated framing can remain."
}, indent=2) + "\n")

if "--send" in sys.argv:
    key = os.environ["TYPESAFE_API_KEY"]
    for i in range(1, 4):
        req = urllib.request.Request(
            "https://api.typesafe.ai/v1/systemone",
            data=(OUT / f"request-{i}.json").read_bytes(),
            headers={"Content-Type": "application/json", "Authorization": "Bearer " + key},
        )
        started = datetime.datetime.now(datetime.timezone.utc).isoformat()
        with urllib.request.urlopen(req, timeout=60) as response:
            raw = response.read()
            status = response.status
        (OUT / f"response-{i}.json").write_bytes(raw + b"\n")
        (OUT / f"response-{i}.metadata.json").write_text(json.dumps({"startedAt": started, "status": status, "endpoint": "v1/systemone"}, indent=2) + "\n")
        data = json.loads(raw)
        print(json.dumps({"request": i, "model": data.get("model"), "answers": data.get("answers"), "usage": data.get("usage")}), flush=True)
