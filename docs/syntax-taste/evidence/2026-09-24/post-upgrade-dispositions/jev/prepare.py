"""Prepare three independently worded advisory scope questions for Jev."""

from __future__ import annotations

import hashlib
import json
from pathlib import Path

OUT = Path(__file__).resolve().parent


def variants(a: str, b: str, c: str) -> tuple[str, str, str]:
    assert len({a, b, c}) == 3
    return a, b, c


state = {
    "task_and_constraints": variants(
        "Select next-upgrade scope for Can, a language for AI coding agents. Correct contracts, dependable edits and native JavaScript/Bun behavior lead; whole-task tokens are secondary. No external users require compatibility. This is an advisory classification, not research or a language decision. The two confirmed implementation defects will be fixed independently of your choices.",
        "Advise which optional Can findings deserve the coming upgrade. AI agents author the language, so reliable semantics and repair matter more than human taste; successful-task token use is a measured secondary aim. Old source and ABI need no preservation for outside users. Favor matching native JavaScript/Bun operations with limited contract adapters. The two concrete bugs are already in the fix scope; this classifier does not research or decide policy.",
        "Can has zero external users and is designed for AI programmers. Prioritize accurate public contracts, predictable changes and equivalent native JS/Bun lowering, with total tokens through a successful task considered second. Do not optimize human familiarity or preserve obsolete interfaces. Recommend scope only from the supplied evidence; engineering retains the decision. Both confirmed conformance bugs are mandatory repairs and are not among the choices below.",
    ),
    "accepted_platform_contracts": variants(
        "Earlier selected design specifies a shared handler-free action declaration imported by server and browser, and server mounting that receives the request plus explicit services. Current declarations require `handles`, handlers accept captures/body only, the invoice grid duplicates declarations with stubs, and a browser test rewrites its declared GET path to the live query route. A Can browser target and real grid exist, but emitted TypeScript needs a test bundler with four Node semantic shims. The browser design selected supported delivery, cancelable events and observable handler failures; general bundling and these event behaviors remain unfinished.",
        "The accepted action packet puts wire metadata in one handler-free package and binds request-aware server handlers separately, with startup dependencies available through explicit composition. Shipped `action` syntax instead mandates `handles`; the server uses manual JSON routes, the grid repeats action definitions, and a harness redirects the captured path to a query endpoint. Browser Can runs in the grid, yet its TypeScript is bundled by a grid-only test tool supplying four Node shims. Accepted browser delivery, `preventDefault` for admitted events and failure reporting are not complete.",
        "A prior contract already chose one shared action wire declaration with no declaration handler, plus explicit request-context server mounting and service binding. The present checker requires `handles` and capture/body-only handlers, while the invoice browser carries stub copies and its GET is rewritten to a differently mounted route. Can-authored browser execution is real; its emitted TS/asset manifest depends on a test bundler's four limited Node shims. The earlier browser contract also calls for maintained delivery, cancelable native events and visible callback faults; those remain open.",
    ),
    "core_and_runtime_evidence": variants(
        "Exported generic bodies receive symbolic declaration checks, but one exported generic cannot pass its opaque parameter to another verified generic identity; private templates can. Catalogue map/fold propagate callback-specific finite failures, authored wrappers cannot parameterize `emits`; finite error-set parameters were deferred. A self-tail `relay` countdown overflows around 10000 steps on local Bun 1.4.2, with no selected tail-call guarantee. Native race selection can finish around 2 ms while its owner returns around 83 ms after an 80 ms loser; HTTP latency follows from source but has not been measured live. Owners deliberately drain losers to preserve leases.",
        "DI-04 symbolic exported-generic checking shipped, but opaque type arguments at calls to a second generic still reject; private specialization differs. Public wrapper `emits` sets stay concrete even though built-in array callbacks retain their exact finite errors; the parameterized-error proposal was postponed. The saved terminal `relay` probe succeeds at 1000 and stack-overflows at 10000 on Bun 1.4.2, without a stack-safety promise. A root race measured early choice and roughly 80 ms later owner return; an HTTP response waits on the same scope, but no actual HTTP hedge trace was taken. Losing work retains resource leases by accepted policy.",
        "Current public generic declarations are checked against symbolic types, yet safe-looking generic-to-generic identity forwarding fails on an opaque argument; local private templates succeed. Higher-order catalogue operations keep callback failure bounds, while user generic libraries have fixed public `emits` or result data; DI-03 deferred new error parameters. Native emitted `relay` recursion has a recorded 1000-pass/10000-stack-failure, and no accepted loop or tail-call guarantee. The owner race probe observed selection near 2 ms but completion near 83 ms with an 80 ms loser. HTTP publication awaits owner drainage according to source, without a measured server request. Lease safety explains the retained drainage rule.",
    ),
    "other_boundaries": variants(
        "The first browser catalogue omits checked input, modifier, file, composition, persistence, history and client WebSocket operations; reload-durable drafts were expressly excluded. Typed node functions compose but the grid manually rebuilds fields and focus. `near` name-based capture, false-before-true Boolean order, local-elision validity and single-line delimiters are selected or deferred rules. Native Map/Set point updates copy collections and a small probe suggests quadratic distinct-key construction; bulk operations are deferred. The invoice/grid is the acceptance workload. No live AI-agent comparison has established a token benefit for new syntax.",
        "The accepted initial browser target is intentionally narrow: no native checkbox/modifier/file/IME snapshot, storage/history/client socket, or durable reload draft. Existing function-built nodes support UI, with repetitive grid state and focus code. Name-derived `near`, compile-time redundant-local rejection, the user-chosen Boolean arm order, and no multiline continuation remain current design. Immutable native Map/Set updates copy each time; distinct-key microtimings motivate but do not qualify a bulk API. Existing invoice/grid examples provide concrete tests. Registered agent cases have zero live attempts, so proposed syntax has no measured whole-task win.",
        "Outside the accepted browser event minimum, common input fields, persistent drafts, navigation and browser WebSocket remain unavailable; the first scope explicitly disclaimed reload survival. The Can grid uses typed named functions and substantial manual rendering/focus work. `near` lookup, local-elision errors and physical-line restrictions have prior retain/defer decisions; Boolean `false` then `true` was a direct user selection. Point updates publish copied native Maps/Sets and a local unique-key probe shows increasing cost, but no representative application workload was measured. The existing invoice and grid can test changes. AI agent comparisons were registered but not run live.",
    ),
}

questions: dict[str, dict[str, tuple[str, str, str] | dict[str, tuple[str, str, str]]]] = {}


def add(key: str, ask: tuple[str, str, str], **options: tuple[str, str, str]) -> None:
    assert all(len(v) == 3 and len(set(v)) == 3 for v in options.values())
    questions[key] = {"ask": ask, "options": options}


add(
    "action_binding",
    variants(
        "Which action architecture should the next upgrade treat as its required completion target?",
        "Choose the contract-completion path for sharing authenticated invoice operations across Bun and browser Can.",
        "What action binding direction best closes the already selected invoice client/server requirement?",
    ),
    selected_split=variants(
        "Implement the accepted handler-free wire declaration and separate request-aware server mount, with explicit service binding and one imported client contract.",
        "Finish the previously chosen shared metadata package, attach authenticated handlers at the server mount, and give them explicit startup dependencies.",
        "Honor the selected server-free declaration plus context-bearing mount so both targets import the same action and services stay server-owned.",
    ),
    alternate_projection=variants(
        "Reopen the design for a bound callable whose client projection is provably server-free and still shares one contract.",
        "Replace the selected split only after designing a context-capturing action with a checked browser-safe projection of its wire shape.",
        "Investigate one bound operation callable and derive an isolated client view, revising the prior declaration/mount contract explicitly.",
    ),
    manual_contracts=variants(
        "Retain separate manual JSON routes and duplicated client declarations as the supported pattern, with drift tests.",
        "Accept mirrored invoice actions and hand-mounted endpoints, relying on comparison tests for client/server agreement.",
        "Make current copied declarations, stub handlers and manual transport matching an intentional supported application convention.",
    ),
)

add(
    "browser_delivery",
    variants(
        "How should supported browser delivery be scoped for this upgrade?",
        "Which browser runtime commitment should accompany the accepted Can-authored grid target now?",
        "Select the next delivery boundary for emitted browser Can while preserving its existing execution evidence.",
    ),
    maintained_profile=variants(
        "Ship a maintained browser runtime/build profile and audit every reachable dependency, using native browser operations and small adapters.",
        "Provide a supported main-thread runtime and bundling command with complete transitive closure checks and browser-native semantics.",
        "Own the browser-target runtime profile, packaged output and reachable module audit as a tested distribution path.",
    ),
    complete_shims=variants(
        "Turn the shim layer into an official build path only after specifying and testing full admitted async, diagnostics and ownership behavior.",
        "Maintain the current bundler strategy as product tooling, qualifying each Node shim against all exposed browser operations.",
        "Support a documented semantic-shim bundle after exhaustive admitted-operation conformance and runtime-closure tests.",
    ),
    grid_only=variants(
        "Keep TypeScript plus manifest as intermediate output and qualify only the existing test-bundled grid integration.",
        "Retain the browser compiler artifact without promising a general delivery pipeline beyond the named grid harness.",
        "Scope browser support to the current integrated example and leave arbitrary application bundling to expert callers.",
    ),
)

add(
    "generic_calls",
    variants(
        "Should safe public generic-to-generic forwarding enter this upgrade?",
        "What is the justified disposition of opaque generic helper calls after DI-04 shipped?",
        "Choose whether extracting a verified public generic helper deserves a compiler change now.",
    ),
    implement=variants(
        "Admit calls through validated parametric callee signatures with symbolic substitution and cycle checks; still reject representation-dependent bodies.",
        "Implement bounded opaque-argument forwarding between declaration-checked generic contracts, preserving illegal-operation diagnostics.",
        "Allow certified public generic signatures to compose symbolically without concrete specialization, with recursion safeguards and strict body checks.",
    ),
    experiment=variants(
        "Keep current rejection temporarily and run helper-extraction, nested-call and agent repair trials before extending the checker.",
        "Defer the new rule until controlled public/private composition examples show a material task benefit and safe cycles.",
        "Prototype symbolic forwarding and compare it with explicit callable inputs on held-out library changes before adoption.",
    ),
    retain=variants(
        "Preserve the exported-generic call restriction and require explicit operation/callable parameters for extracted helpers.",
        "Treat today's opaque call rejection as the permanent public boundary, even for identity forwarding.",
        "Keep generic composition via written dictionaries or concrete private templates without a new symbolic call rule.",
    ),
)

add(
    "failure_wrappers",
    variants(
        "What should this upgrade do about authored generic wrappers that lose callback-specific failure sets?",
        "Choose scope for public error-bound abstraction after the earlier DI-03 deferral.",
        "How should reusable callback-failure preservation be handled in the next work round?",
    ),
    implement=variants(
        "Add an explicit finite error-set parameter to authored generic contracts, retaining exact public error obligations.",
        "Introduce named finite failure-bound parameters so wrappers propagate caller-specific `emits` types.",
        "Ship a bounded authored error-set kind that composes through nested generic wrappers without a general effects system.",
    ),
    compare=variants(
        "Defer syntax while comparing fixed bounds, result data and a finite-parameter prototype on two unrelated callbacks and nested wrappers.",
        "Run the registered two-domain library and agent-edit trial before promoting the previously deferred parameter mechanism.",
        "Keep the existing public contract this round and measure result adapters versus finite error rows in a reusable wrapper experiment.",
    ),
    retain=variants(
        "Keep written concrete `emits` and nominal result-data APIs as the long-term authoring model.",
        "Treat fixed finite bounds or explicit success/failure data as sufficient for user libraries without a new error kind.",
        "Preserve current authored error contracts and add only individually justified built-in higher-order operations.",
    ),
)

add(
    "iteration",
    variants(
        "Should general growing iteration be included now despite the absence of a prior stack-safety promise?",
        "What disposition fits the recorded deep `relay` overflow and application loops?",
        "Choose next-upgrade scope for dynamic stack-safe repetition beyond array folds.",
    ),
    implement=variants(
        "Add one explicitly stack-safe route for immutable dynamic state, preserving evaluation, failures, fixture identity and diagnostics.",
        "Ship a bounded native-loop-lowered iteration construct or proven self-tail case with exact completion and assertion semantics.",
        "Guarantee ordinary growing work without stack growth through one carefully specified native-loop lowering path.",
    ),
    compare=variants(
        "Defer syntax and compare self-tail lowering with an immutable-state iterator on pagination and state-machine programs.",
        "Run focused recursion/loop prototypes and agent tasks first, including fixture and diagnostic identity checks.",
        "Keep current runtime this round while testing both native-loop candidates and a real dynamic-continuation workload.",
    ),
    retain=variants(
        "State native recursion limits as a supported scope boundary and direct array work to stack-safe folds.",
        "Document that `relay` forwards completion but does not optimize the stack, retaining current collection traversal idioms.",
        "Accept bounded native stack behavior for recursion and recommend existing array combinators where applicable.",
    ),
)

add(
    "race_lifetime",
    variants(
        "Which scope is justified for race-result latency while resource owners retain losing leases?",
        "How should the next upgrade address the unmeasured HTTP hedge implication of owner drainage?",
        "Choose the immediate response-timing work without assuming cancellation reverses effects.",
    ),
    redesign=variants(
        "Implement response-independent supervised losers or explicit cooperative cancellation after specifying cleanup, shutdown and side-effect ownership.",
        "Change request race lifetime now with an explicit server-owned scope for unfinished losers and contract-safe response publication.",
        "Ship a new request-scope cancellation/ownership policy that releases responses early while preserving leases and late diagnostics.",
    ),
    measure=variants(
        "Retain current owner drainage while measuring a real HTTP hedge and comparing deadlines with supervised ownership before design selection.",
        "Keep existing safety semantics this round; gather server timing and cleanup evidence for each cancellation alternative.",
        "Run a live request race and operation-deadline comparison first, leaving loser ownership unchanged until results justify revision.",
    ),
    retain=variants(
        "Keep mandatory draining and documented latency as the intended permanent policy; require operation-local timeouts where needed.",
        "Accept response delay behind owned losers as part of the service contract and focus on deadline guidance.",
        "Preserve the present race and request completion rule without a new hedge experiment or cancellation surface.",
    ),
)

add(
    "browser_host_surface",
    variants(
        "How much richer browser input/host support belongs in this upgrade beyond the already accepted cancelable-event minimum?",
        "Choose the scope of additional browser capabilities after completing admitted event cancellation and error reporting.",
        "What next work on native input, persistence and host adapters has sufficient evidence for near-term inclusion?",
    ),
    add_now=variants(
        "Ship checked state, modifiers and IME/file input plus reload drafts, choosing a maintained typed extension architecture now.",
        "Expand the browser capability catalogue for rich inputs and durable drafts in this round, with lifecycle and codec tests.",
        "Implement native input detail and persistent draft features immediately behind typed host operations and product acceptance.",
    ),
    experiment=variants(
        "Complete the accepted event minimum, then prototype persistent drafts and rich inputs under catalogue expansion and reviewed adapters before selection.",
        "Keep new host APIs out of this implementation scope but compare two bounded extension paths on the same draft/IME app.",
        "Defer expanded browser capabilities while testing native-event detail and storage with both controlled admission mechanisms.",
    ),
    retain=variants(
        "Keep the first browser catalogue narrow and support only frontends whose native needs fit its admitted operations.",
        "Preserve the existing host API range as the product boundary after satisfying the already selected event requirements.",
        "Declare richer input and persistence outside Can's browser recommendation rather than opening new capabilities.",
    ),
)

add(
    "ui_reuse",
    variants(
        "Should the next upgrade add an authored UI reuse mechanism beyond functions and typed nodes?",
        "What is justified by the grid's manual rendering and focus management now?",
        "Choose the authoring-reuse disposition for editable fields and keyed rows.",
    ),
    implement_library=variants(
        "Ship a reusable typed field/keyed-row library after proving focus, IME, row reorder, late response and disposal behavior.",
        "Implement shared field and row-editor helpers in maintained Can packages, with product-level focus and lifecycle checks.",
        "Make a tested typed UI helper library part of this upgrade while leaving component grammar out.",
    ),
    prototype=variants(
        "Defer a supported abstraction while prototyping the same grid field and keyed editor as named helpers and measuring edit costs.",
        "Trial reusable functions on the existing grid before making an API promise or adding new component syntax.",
        "Study one field-addition and row-reorder workflow with a helper library, keeping the current public UI surface this round.",
    ),
    retain=variants(
        "Keep named functions and typed nodes as the stable authoring approach with clearer examples only.",
        "Accept manual render/state synchronization as the current UI model and avoid a new reusable abstraction.",
        "Preserve the imperative node toolkit as the full supported composition layer without a library experiment.",
    ),
)

add(
    "bulk_collections",
    variants(
        "What is the disposition of immutable Map/Set bulk creation given the observed copying cost?",
        "Choose whether unique-key insertion scaling justifies a new native-backed collection API now.",
        "How should this upgrade respond to repeated copy-on-insert behavior?",
    ),
    implement=variants(
        "Add bulk constructors or grouping with one native Map/Set build and one immutable publication, specifying duplicate/order/failure rules.",
        "Ship bounded batch collection operations that preserve immutable handles and callback error contracts.",
        "Provide native-backed one-pass Map/Set aggregation with exact collision and ownership semantics this round.",
    ),
    measure=variants(
        "Defer the API while benchmarking representative unique and duplicate-heavy workloads and agent tasks against current folds.",
        "Run a realistic batch construction trial and define collision/order/error contracts before promoting DI-20.",
        "Keep copy-on-point-update semantics and test native bulk candidates at registered data sizes first.",
    ),
    retain=variants(
        "Keep immutable point updates only and document their cost as the collection model.",
        "Accept quadratic repeated distinct-key construction in exchange for a minimal stable API.",
        "Treat current copied native Map/Set updates as sufficient without bulk-workload qualification.",
    ),
)

add(
    "capture_binding",
    variants(
        "Should explicit `near` capture binding syntax be reopened in this upgrade?",
        "How should same-typed shadow/rename risk in callable capture be disposed?",
        "Choose the next scope for parameter-name-based closure capture.",
    ),
    implement=variants(
        "Add explicit reference-site capture mappings while preserving immutable capture timing and type checks.",
        "Replace implicit same-name `near` lookup with checked caller-provided capture bindings.",
        "Ship source syntax that names the intended value at callable creation and diagnoses stale bindings.",
    ),
    trial=variants(
        "Retain `near` while comparing explicit context records and capture-map syntax on registered rename/shadow repair tasks.",
        "Test the existing context-record idiom against same-typed refactors before selecting a new capture form.",
        "Defer grammar changes and measure whether context records and diagnostics prevent the demonstrated rename hazard.",
    ),
    retain=variants(
        "Keep exact-name capture as the intended permanent rule and rely on explicit context records when identity matters.",
        "Preserve `near` parameter-name lookup without a formal refactor experiment.",
        "Accept the current closure-binding convention, documenting its shadow behavior and existing typed-record alternative.",
    ),
)

add(
    "multiline_layout",
    variants(
        "Should delimiter continuation enter this upgrade after the formatter shipped?",
        "What scope follows from long records and mandatory assertion rows under physical-line limits?",
        "Choose whether the deferred multiline grammar has enough evidence for implementation now.",
    ),
    implement=variants(
        "Permit continuation within delimiters while keeping indentation block semantics and exact formatting checks.",
        "Add a bounded multiline expression grammar with preserved block layout and formatter round trips.",
        "Ship newline admission only inside delimited expressions, with clear lexer and parser failure cases.",
    ),
    trial=variants(
        "Keep current grammar while measuring held-out agent edits and formatter output on wide declarations and assertion rows.",
        "Compare single-line formatting against continuation prototypes on registered creation/refactor/repair tasks first.",
        "Defer parser changes pending evidence that line pressure materially impairs AI coding after existing formatting help.",
    ),
    retain=variants(
        "Keep every delimited expression on one physical line as the stable language layout rule.",
        "Accept wide record and assertion rows under the present lexer and rely on formatting conventions.",
        "Treat no-newline delimiters as the long-term grammar without additional trials.",
    ),
)


def request(i: int) -> dict:
    return {
        "model": "jev-latest",
        "state": {key: values[i] for key, values in state.items()},
        "questions": {
            key: {
                "type": "choice",
                "instructions": value["ask"][i],
                "criteria": {option: variants_[i] for option, variants_ in value["options"].items()},
            }
            for key, value in questions.items()
        },
    }


def audit(payloads: list[dict]) -> dict:
    assert len(payloads) == 3
    assert all(p["model"] == "jev-latest" for p in payloads)
    assert all(set(p["state"]) == set(state) for p in payloads)
    assert all(set(p["questions"]) == set(questions) for p in payloads)
    differences = []
    for key in state:
        assert len({p["state"][key] for p in payloads}) == 3
        differences.append("state." + key)
    for key in questions:
        option_keys = set(questions[key]["options"])
        assert all(set(p["questions"][key]["criteria"]) == option_keys for p in payloads)
        assert len({p["questions"][key]["instructions"] for p in payloads}) == 3
        differences.append("questions." + key + ".instructions")
        for option in option_keys:
            assert len({p["questions"][key]["criteria"][option] for p in payloads}) == 3
            differences.append("questions." + key + ".criteria." + option)
    hashes = [hashlib.sha256(json.dumps(p, sort_keys=True).encode()).hexdigest() for p in payloads]
    assert len(set(hashes)) == 3
    return {
        "semantic_invariants": {
            "state_fields": list(state),
            "question_ids": list(questions),
            "option_ids_by_question": {key: list(q["options"]) for key, q in questions.items()},
            "model": "jev-latest",
        },
        "rewritten_explanatory_fields": len(differences),
        "checked_fields": differences,
        "request_sha256": hashes,
        "meaning_check": "All three state variants preserve facts/constraints; every question preserves its choice set, outcome meaning and decision boundary. The prose was manually compared before transmission. Exact identifiers stay stable.",
    }


if __name__ == "__main__":
    payloads = [request(i) for i in range(3)]
    result = audit(payloads)
    for i, payload in enumerate(payloads, 1):
        (OUT / f"request-{i}.json").write_text(json.dumps(payload, indent=2) + "\n")
    (OUT / "wording-audit.json").write_text(json.dumps(result, indent=2) + "\n")
    print(f"prepared 3 fresh requests, {len(questions)} choices, {len(result['checked_fields'])} rewritten fields")
