"""Prepare three independently phrased consultations on selected behavior."""

from __future__ import annotations

import hashlib
import json
from pathlib import Path

OUT = Path(__file__).resolve().parent


def variants(first: str, second: str, third: str) -> tuple[str, str, str]:
    assert len({first, second, third}) == 3
    return first, second, third


state = {
    "mandate": variants(
        "Can is an AI-agent-authored language with zero external users and no compatibility obligation. The next upgrade must finish the earlier handler-free shared action/request-aware mount and supported browser target, repair two defects and accepted UI gaps, and admit bounded public generic helper composition. Compile to equivalent native JavaScript/Bun operations, using only contract and immutability adapters. You advise technical mechanisms from supplied evidence, not scope or research.",
        "The chosen Can scope is fixed: one server/browser wire action, explicit request-bearing server binding, maintained browser delivery, two conformance repairs, required UI behavior and safe public-generic forwarding. There are no outside users whose old spelling or ABI must survive. Prefer native JS/Bun execution with narrow Can-contract adapters. Can is written by coding agents; this classification may compare designs but cannot remove a selected outcome or inspect code itself.",
        "This is mechanism advice for Can's already selected upgrade, not a vote to defer it. Shared action metadata without a handler, request-aware mounting, general browser delivery, the recorded bug/UI repairs and public generic-to-generic composition are mandatory results. AI agents author Can; obsolete forms need no compatibility. Generated TypeScript should use native JavaScript/Bun behavior except adapters preserving exact Can semantics. Judge only the evidence given here.",
    ),
    "browser_runtime": variants(
        "The compiler emits browser TypeScript and a content-addressed module manifest, but the grid's test-only Bun bundler replaces node:crypto, node:util, node:fs and node:async_hooks. Its async context is synchronous-only and loses ownership across await; its file shim omits source maps. A supported empty app and invoice grid need one maintained JS delivery path, full reachable runtime/secret auditing, error/equality/codec/owner parity, CSP-safe same-origin assets and admitted coordination in named browsers.",
        "Today's browser entry and hash manifest are compiler artifacts, not served executable JavaScript. Gate 5 adds four Node semantic shims only in test tooling: crypto, util/proxy checks, filesystem diagnostics and AsyncLocalStorage. That context stub does not preserve asynchronous ownership, and diagnostic maps are discarded. The accepted target requires a reusable build for empty and grid apps, audited transitive modules and secrets, verified same-origin CSP publication, and observable parity for native failures, equality, codecs and coordination.",
        "Browser Can already executes in a grid harness, with emitted TS plus a digest manifest. Production bundling is absent; the harness aliases four Node builtins. Its stack-scoped async-hooks substitute is not faithful after suspension, and diagnostics use a baked index without module maps. The required product path must build arbitrary admitted apps including an empty one, audit every reachable runtime edge and secret-bearing output, publish the verified same-origin asset under CSP, and preserve Bun/browser contract results including owner-backed concurrency.",
    ),
    "entry_and_events": variants(
        "The accepted browser contract says main has no arguments. Current grid main takes str[] supplied by three handwritten test boot lines reading location.search; those lines must disappear. Invoice identity is untrusted bootstrap data, never permission. Existing browser::on_event passes an immutable four-field snapshot into an async Can callback. Native preventDefault must run before dispatch returns; awaiting the callback is too late. Raw Event objects cannot escape to Can, and disposed views cannot cancel later events.",
        "Earlier design chose an argument-free browser main, whereas shipped grid boot accepts str[] from a test-authored query-string launcher. A supported build must supply initial invoice selection through a checked bounded path and still reauthorize on the server. Event handlers run asynchronously over copied fields. Browser default cancellation is synchronous during the native listener, so a decision after an await cannot work; retain immutable snapshots and view-scoped disposal.",
        "There is a startup mismatch: selected main() has zero inputs, current main(args) depends on a harness snippet that extracts an invoice query value. The id is mere client-supplied routing data. Current on_event converts Event to strings and settles an asynchronous handler; preventDefault must occur during the original native dispatch. The browser surface must never expose a retained mutable Event and must honor disposal before cancellation.",
    ),
    "generic_boundary": variants(
        "DI-04 already checks public generic bodies once with opaque symbolic parameters, forbidding unprovided arithmetic/equality/codec operations; private generics remain concrete templates. Today specialize.go rejects a second generic call when its type argument contains the caller's opaque parameter, though public identity-to-identity forwarding is safe. Selected P03 requires that ordinary public helper extraction work across packages without false parametric proof or emitted opaque specialization. Mutual cycles and expanding type recursion need predictable sound checking.",
        "The current checker proves exported generic declarations symbolically, but permits opaque type arguments only for identical self-recursion. A public wrapper cannot call a separately valid public identity helper; a private template has no universal body proof. The upgrade promises bounded safe cross-package helper calls, while still diagnosing representation-dependent bodies at declaration and emitting only reached concrete instances. Cross-generic cycles must not commit an unvalidated callee or expand forever.",
        "Public generics have a DI-04 universal body check under declaration-qualified opaque types; local private templates check at concrete use sites. Current opaqueArgument rejection blocks even representation-independent F<T> calling validated G<T>. P03 selects such composition, not implicit traits or generic errors. The checker must separate symbolic proof edges from executable specializations, keep illegal operations failing at the callee, and bound mutual/expanding call graphs.",
    ),
}

questions: dict[str, dict[str, object]] = {}


def add(key: str, ask: tuple[str, str, str], **options: tuple[str, str, str]) -> None:
    assert len(set(ask)) == 3
    assert all(len(set(words)) == 3 for words in options.values())
    questions[key] = {"ask": ask, "options": options}


add(
    "runtime_architecture",
    variants(
        "Which maintained browser runtime design best meets the accepted semantic closure?",
        "Select the browser delivery mechanism that can be qualified for both empty and invoice apps.",
        "What runtime architecture should the supported Can browser build own?",
    ),
    browser_profile=variants(
        "Build a browser-specific runtime profile: precompute concrete identities, use private Can-value branding and safe native errors, pass execution context explicitly across async branches, and embed verified diagnostic maps. Keep native Promise/DOM/Fetch operations.",
        "Choose host-specific browser modules and generated context threading rather than Node aliases; seal build-time identities, guard foreign values, package maps with the final asset, and leave coordination selection to native promises.",
        "Create a maintained main-thread browser runtime surface with explicit owner tokens, build-sealed identity data, branded Can values and bundled source metadata; adapt only missing host contracts around native JS/browser operations.",
    ),
    maintained_shims=variants(
        "Promote the existing Node-shim bundle only after replacing all four limited shims with faithful implementations for async context, proxy/error classification, hashing and source-map file access, then qualify every reachable operation.",
        "Keep the present alias-based bundler as the product path, but implement and test complete node:async_hooks, node:util, node:crypto and node:fs semantics for admitted Can browser execution.",
        "Support a documented Node-compatibility layer over the current grid build; upgrade each shim to cover real ownership across awaits, safe host-object checks, exact hashes and diagnostic mappings before release.",
    ),
)

add(
    "bootstrap_input",
    variants(
        "What narrow source supplies invoice selection to an argument-free Can browser main?",
        "Pick a checked boot input path that removes the handwritten grid launcher while keeping main() and server authorization.",
        "How should the supported asset pass an untrusted invoice id to no-argument browser startup?",
    ),
    query_parameter=variants(
        "Add a bounded browser::query_parameter literal-key read over native URLSearchParams(location.search); reject duplicate/oversized values, return immutable optional text, and recheck the captured resource on every server request.",
        "Expose only a checked read of one named current-URL query value, with explicit missing/invalid outcomes; the application passes it to boot and the server treats it solely as an identifier.",
        "Let main() call a restricted query-parameter catalogue operation backed by URLSearchParams, not a history/navigation object; cap and validate the value and never infer authority from it.",
    ),
    root_boot_data=variants(
        "Have the verified server page place one typed transparent boot record on the admitted root; a checked browser operation reads and decodes that inert data before boot, with no general DOM attribute or raw-script escape.",
        "Use a server-rendered, schema-checked bootstrap value attached to the known grid root and decode it through the shared wire codec, keeping the main signature empty and avoiding general location access.",
        "Carry invoice selection as a bounded encoded record in verified root markup; a narrow boot-data catalogue read yields the typed value, and server authorization still occurs on each action.",
    ),
)

add(
    "generic_proof_order",
    variants(
        "How should validated public generic helper calls handle mutual cycles?",
        "Choose the proof rule for safe symbolic public-to-public calls, including cyclic declarations.",
        "Which checker boundary admits helper forwarding without circular proof or runaway specialization?",
    ),
    scc_certification=variants(
        "Validate a strongly connected component as one unit using provisional signatures; commit none until every body and edge checks, and reject constructor growth on an internal opaque edge while allowing bare formals or closed concrete types.",
        "Build resolved public-generic call components, type internal calls against provisional contracts, certify all members atomically, and forbid any cycle edge that wraps an opaque formal in a new type constructor.",
        "Prove mutually recursive exported generics together under declared signatures; publish the proof only when the whole component passes, with cyclic arguments limited to bare caller variables or opaque-free concrete types.",
    ),
    acyclic_only=variants(
        "Certify public generic declarations only in dependency order; allow opaque forwarding to a completed callee but reject every cross-generic opaque cycle, including stationary ones, as outside this bounded feature.",
        "Ship helper composition for a directed acyclic public-generic proof graph; fail mutually recursive symbolic calls rather than introduce component-level certification this round.",
        "Reuse validated callee signatures only after they are final, so identity/helper chains compile but any cycle among public opaque calls gets a clear diagnostic and no provisional proof.",
    ),
)

add(
    "event_cancellation",
    variants(
        "Where should the cancellation choice be made so native preventDefault runs in time?",
        "Select a bounded native-event cancellation contract compatible with asynchronous Can handlers.",
        "Which event API can request default prevention without exposing a live Event object?",
    ),
    registration_policy=variants(
        "A checked registration declares cancel-on-submit or cancel-on-specific-key; its native listener tests liveness/cancelability and calls preventDefault synchronously before invoking the ordinary async snapshot handler.",
        "Make cancellation a finite listener policy chosen when registering, with optional exact key filter; the adapter cancels during native dispatch, then sends immutable fields to the Can callback.",
        "Add narrowly typed canceling event registrations whose native code prevents the admitted default immediately on a matching live event, leaving subsequent asynchronous handler work separate.",
    ),
    synchronous_predicate=variants(
        "Introduce a specially checked synchronous no-suspend Can predicate over an immutable event snapshot; native dispatch runs it to completion and prevents the default only when it returns cancel, before an async handler starts.",
        "Add a restricted sync Can decision callback that cannot await or call effects; run it in the native listener before launching the normal handler and use its Boolean decision for preventDefault.",
        "Compile a provably synchronous pure event selector on copied fields into the dispatch stack, then call preventDefault from its result and separately schedule the existing asynchronous callback.",
    ),
)


def explanatory_values(request: dict[str, object]) -> dict[str, str]:
    found = {f"state.{key}": value for key, value in request["state"].items()}
    for key, question in request["questions"].items():
        found[f"questions.{key}.instructions"] = question["instructions"]
        for option, value in question["criteria"].items():
            found[f"questions.{key}.criteria.{option}"] = value
    return found


requests = []
for index in range(3):
    request = {
        "model": "jev-latest",
        "state": {key: words[index] for key, words in state.items()},
        "questions": {
            key: {
                "type": "choice",
                "instructions": content["ask"][index],
                "criteria": {
                    option: words[index] for option, words in content["options"].items()
                },
            }
            for key, content in questions.items()
        },
    }
    requests.append(request)
    (OUT / f"request-{index + 1}.json").write_text(json.dumps(request, indent=2) + "\n")

paths = [explanatory_values(request) for request in requests]
assert all(set(item) == set(paths[0]) for item in paths)
assert all(len({item[path] for item in paths}) == 3 for path in paths[0])
assert all(set(request["questions"]) == set(requests[0]["questions"]) for request in requests)
assert all(
    set(request["questions"][key]["criteria"]) == set(requests[0]["questions"][key]["criteria"])
    for request in requests
    for key in questions
)
audit = {
    "rewritten_explanatory_fields": len(paths[0]),
    "checked_fields": sorted(paths[0]),
    "request_sha256": [
        hashlib.sha256((OUT / f"request-{index + 1}.json").read_bytes()).hexdigest()
        for index in range(3)
    ],
    "semantic_invariants": {
        "state_fields": list(state),
        "question_ids": list(questions),
        "option_ids_by_question": {
            key: list(content["options"]) for key, content in questions.items()
        },
    },
    "meaning_check": "All state variants preserve each factual constraint; each question retains its choice set and technical outcome. The coordinator compared all prose before dispatch. Exact technical names are stable; earlier answers are not present in later requests.",
}
(OUT / "wording-audit.json").write_text(json.dumps(audit, indent=2) + "\n")
print(f"prepared {len(requests)} requests with {len(paths[0])} reworded fields")
