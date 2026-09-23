"""Prepare three reworded Jev requests; send only with --send."""

from pathlib import Path
import datetime
import json
import os
import sys
import urllib.request

OUT = Path(__file__).resolve().parent

facts = {
    "objective": (
        "Two independent 24 September reviews are being consolidated into one recommendation plan. The goal is a Can language credible for broad SaaS backend and frontend work. This consultation advises design experiments; it does not adopt syntax or authorize implementation.",
        "A SaaS-focused review and a fresh source-focused review need one ordered outcome. Can should eventually serve a wide range of server and client SaaS applications. The answers here guide a review, not a change to approved grammar or compiler behavior.",
        "Reconcile two current assessments of Can against the ambition of recommending it across SaaS services and user interfaces. Treat choices as advice about next evidence and contracts; no selected language rule changes merely because a response prefers it.",
    ),
    "baseline": (
        "Can currently has nominal immutable values, named functions, explicit public domain-error bounds, attached assertions, native AI forms with grouped state, and native JavaScript/Bun lowering with contract adapters. There are zero external users, so compatibility is not a product requirement. Large abstractions must beat the best present idiom in a real program.",
        "The existing language uses frozen nominal data, named callables, written outward failures, declaration-owned tests, distinct AI declarations and grouped AI inputs. Its target compiles to JS/Bun operations with safety glue. No outside adopter needs old spellings preserved; justify major additions by comparing representative Can code.",
        "Preserve the valuable core while reviewing its limits: immutable nominal records, named functions, visible error sets, attached executable examples, and native AI with grouped state. Emit equivalent JS/Bun operations. Migration does not constrain design because external users number zero. New general mechanisms need concrete cost and behavior evidence.",
    ),
    "pattern_evidence": (
        "Current executable probes show that a final bare pattern decliend intended as declined is treated as a binding/catch-all; adding refunded then remains accepted. The correctly spelled old match rejects the new case, and decliend() rejects immediately. A September 22 disposition had retained bare-name binding before this counterexample was established.",
        "The checker and emitted assertions accept a typo in the last union arm because unknown unqualified names bind the whole value. Extending the variant does not expose the error; the properly named three-arm version becomes non-exhaustive. Constructor spelling of the typo fails. Earlier policy kept the bare rule, but this new direct probe challenges that policy.",
        "A tested payment match with final decliend instead of declined compiles, executes and covers future refunded implicitly. Using declined would make refunded a missing arm; writing decliend() is diagnosed. The present parser/checker intentionally gives bare unknown names binder meaning. The earlier retain decision did not consider this demonstrated failure mode.",
    ),
    "library_evidence": (
        "Exported authored records can be constructed and copy-updated by consumers, so a validating email factory cannot guarantee every email is valid. Ordinary generic retry/audit/map_option wrappers cannot quantify over a caller's finite error set, unlike catalogue map/fold. Generic body operations may also be absent from a public signature. Independently written dependencies collide on short package names and numeric error IDs.",
        "Source libraries cannot hide ordinary record construction and with-updates while exporting the type. Catalogue combinators specialize callback errors, but an authored helper names concrete errors; operational templates are checked at each use without an expressed requirement. Two dependency projects using the same package label or application error number cannot be linked unchanged.",
        "Package composition exposes three gaps: ordinary exported validated records remain forgeable; user-authored higher-order code lacks the catalogue's failure-set genericity and template constraints are not visible in its signature; unrelated dependencies must coordinate package short names and diagnostic numbers. These are observed language/library boundaries rather than requests for unrestricted foreign JS.",
    ),
    "fixture_evidence": (
        "An emitted two-package probe changes a helper fixture's selection solely by renaming the caller's assertion label. The renamed call executes the real text::from_int(7), changing expected fixture from 'fixture' to '7'. Lexical queues and root identities remain isolated; the surprising dependency is the shared short label. Earlier policy deferred root-owned symbolic seams.",
        "Changing only the root row name in a consuming package leaves static checking green but stops a helper package's local when-row from activating. The helper then returns actual conversion text rather than injected data. This is not a concurrent queue collision. A previous disposition postponed explicit cross-call scenario links before the repro existed.",
        "The current fixture table filters by the root assertion's short name. In the current emitted probe, customer activates helper when-row and renamed does not, though the call inputs and helper source are unchanged. Checked arguments, lexical invocation paths and FIFO ownership work as designed. This new evidence reopens whether caller-owned scenario mapping is warranted.",
    ),
    "frontend_evidence": (
        "Can presently runs Bun services with safe server-rendered HTML and pinned HTMX; it deliberately has no browser Can execution. Routes, form fields and HTMX targets are separate string contracts. A broad Can-authored frontend would need local state, named events, async commands and component disposal. A controlled existing browser stack can instead consume typed wire contracts but would not make the client authored in Can.",
        "The qualified model serves HTML/HTMX from Bun, with no Can runtime inside a browser. String links between endpoints, form controls and swap targets survive independent renames. An invoice grid with drafts, keyboard behavior, optimistic saving and rollback needs client-side state. Keeping that code in another browser language can still make Can a strong server language, with an honest scope label.",
        "Server-driven forms, search and dashboards are supported; local interactive browser computation is excluded by the selected spec. URL membership, form names and DOM target identity are not linked by Can types. The stated all-frontend ambition requires a decision tested by an offline/optimistic editable-grid example: a Can client target or a typed bridge to an existing client stack, with different resulting product claims.",
    ),
    "delivery_evidence": (
        "Only Darwin ARM64 Bun is currently qualified. The combined recommendation should test tenant invoice editing, duplicate signed webhook processing with an outbox, failure/cleanup, actual browser behavior and bounded shutdown on a named Linux target. SQL parameters and runtime row decoding exist, but a declared result is not checked against an actual versioned schema. No design consultation substitutes for those execution gates.",
        "The release target pins macOS arm64; Linux service behavior is not yet release-qualified. A realistic proof would exercise authorized invoice edits, idempotent provider events, transaction/outbox outcomes, failed dependencies, displayed status and shutdown. Current SQL descriptors enforce shape/cardinality and codecs but do not validate a live schema projection at build time. These are operational gates, not mere syntax preferences.",
        "Can has not yet passed a supported Linux distribution gate. Before a broad SaaS recommendation, demonstrate a multi-tenant edit, a replayed webhook and uncertain commit, browser-visible errors, real database integration, and bounded service closure. SQL's runtime row checks contain mismatches while compile-time schema agreement is unproven. Feature selection should remain distinct from qualification evidence.",
    ),
}

questions = {
    "pattern_policy": {
        "ask": (
            "Which language rule best addresses the demonstrated misspelled-case hazard?",
            "Choose the pattern-intent experiment that most directly restores future-case checking.",
            "How should nominal tests and bindings be distinguished after the current typo probe?",
        ),
        "options": {
            "explicit_binding": (
                "Require visible binding intent, leave bare nominal case tests, and reject unknown test names.",
                "Introduce a distinct binder form so an unresolved bare case cannot silently bind the subject.",
                "Give value captures their own syntax while familiar leaf names remain checked alternatives.",
            ),
            "constructors_only": (
                "Require explicit constructor syntax for every nominal pattern; retain bare names as binders.",
                "Make paid() style the nominal test form and use unadorned identifiers only for captures.",
                "Force leaf matching to spell a constructor and reserve simple names for whole-value binding.",
            ),
            "warn_only": (
                "Keep current acceptance and add a typo/unused-binder diagnostic that can be ignored.",
                "Retain ambiguous names but warn when a binding resembles a declared leaf.",
                "Use heuristic warnings instead of changing the present grammar or validity rule.",
            ),
            "retain": (
                "Leave the binder/case interpretation and teach authors to qualify or call constructors.",
                "Preserve the earlier retained pattern rule, relying on style guidance for case intent.",
                "Keep today's bare spelling and document the catch-all interpretation as expected.",
            ),
        },
    },
    "value_boundary": {
        "ask": (
            "Which experiment best establishes credible application-owned validated values?",
            "How should a reusable library express an email type whose construction it controls?",
            "Select the next representation-boundary direction for ordinary business values.",
        ),
        "options": {
            "owner_control": (
                "Prototype public nominal types with owner-controlled construction and explicit decoding, equality and test access.",
                "Compare package-private representation behind an exported immutable type, specifying factories and codecs together.",
                "Try authored abstract values whose creating package governs construction, updates and projection without resource privileges.",
            ),
            "transparent_validation": (
                "Keep records fully transparent and require validity checks at each operation that depends on the invariant.",
                "Treat public data as DTOs; validate again wherever an exported value is consumed.",
                "Retain unrestricted construction, with application operations explicitly checking the predicate repeatedly.",
            ),
            "catalogue_only": (
                "Reserve unforgeable types for distribution-owned catalogue entries and add each business value there.",
                "Use compiler-maintained types for invariants; ordinary source records remain publicly constructible.",
                "Make all protected representations platform-owned instead of allowing package authors to define them.",
            ),
            "defer": (
                "Collect more product cases before trying any authored value-ownership mechanism.",
                "Postpone representation control pending evidence beyond email, quantity and tenant identifier.",
                "Keep this question outside the next language milestone and seek additional applications first.",
            ),
        },
    },
    "fixture_scenarios": {
        "ask": (
            "What should be tested next to remove accidental fixture selection across package calls?",
            "Choose a response to the root-label rename changing helper behavior.",
            "How should a consumer deliberately arrange a dependency's test scenario?",
        ),
        "options": {
            "explicit_seams": (
                "Prototype named caller-to-helper scenario links while retaining typed arguments, lexical queues and concurrency identities.",
                "Evaluate explicit mapping from a root scenario to named dependency fixture seams with the current exact checks.",
                "Give cross-call fixture activation a visible named relationship rather than shared short-label coincidence.",
            ),
            "lexical_labels": (
                "Keep short root labels as the selector and document that packages must coordinate them.",
                "Retain current lexical tables and treat assertion-row naming as an intentional cross-package convention.",
                "Preserve inherited root-name activation, with guidance for consumers and helper authors.",
            ),
            "stub_boundary": (
                "Stub the entire helper call in consumers and test its real body only in the helper package.",
                "Use complete dependency substitution at the package edge instead of selecting interior fixtures.",
                "Make consumer tests replace helper results wholesale and leave internal behavior to owned unit tests.",
            ),
            "global_ordinals": (
                "Replace lexical identity with globally numbered invocations or function-name mocks.",
                "Use broad function/ordinal fixture replacement rather than explicit local-site ownership.",
                "Identify injected calls through global order or callee labels across packages.",
            ),
        },
    },
    "generic_errors": {
        "ask": (
            "How should the catalogue-versus-author error-set asymmetry enter the recommendation plan?",
            "Select the next evidence gate for a general fallible helper such as retry.",
            "What is the justified response to user code lacking finite callback-error polymorphism?",
        ),
        "options": {
            "measure_then_trial": (
                "Build equivalent real wrappers with fixed bounds and result data first; trial explicit finite error-row parameters if those materially fail.",
                "Measure current Can's best reusable retry/audit helper, then prototype a written finite set variable only if needed.",
                "Compare concrete errors and data-result adapters to a small explicit error-set generic using an application workload.",
            ),
            "result_permanent": (
                "Standardize nominal success/error values as the permanent generic helper boundary, adapting to emitted failures at callers.",
                "Retain fixed completion signatures and express all reusable variable errors through ordinary result data.",
                "Use result records/variants for higher-order libraries indefinitely, with explicit conversions to completion paths.",
            ),
            "add_intrinsics": (
                "Keep source functions fixed-bound and add each generic fallible operation to the compiler catalogue.",
                "Prefer more language-owned specialized helpers instead of authored error-set abstraction.",
                "Expand built-in combinators whenever a wrapper needs caller-specific failures.",
            ),
            "implicit_inference": (
                "Infer outward public errors from implementation bodies without explicit bounds.",
                "Let each function's body define its exported failure contract automatically.",
                "Remove written public error lists in favor of unannounced inferred sets.",
            ),
        },
    },
    "frontend_path": {
        "ask": (
            "Which product path most honestly reaches a recommendation for Can-authored backend and frontend SaaS?",
            "How should Can sequence the current SSR scope and the richer all-frontend ambition?",
            "Choose a scope strategy for recommending Can across services and interactive user interfaces.",
        ),
        "options": {
            "server_then_browser": (
                "Qualify server-driven SaaS first, then design and qualify Can browser state/events using the demanding editable-grid case before making a Can-only full-stack claim.",
                "Establish a production server/HTMX recommendation, then add a bounded Can client target proved by local drafts, optimistic updates and disposal.",
                "Complete core/service gates followed by native Can browser execution and wire-capability separation before claiming broad Can-authored frontend coverage.",
            ),
            "browser_first": (
                "Prioritize a full Can browser target before repairing core/library composition or qualifying server workflows.",
                "Lead with client runtime and component architecture while deferring the existing service-language contract gates.",
                "Build browser Can immediately as the first milestone, leaving package, fixture and service evidence for later.",
            ),
            "typed_external_client": (
                "Recommend Can servers plus a typed contract bridge to an existing browser language, explicitly limiting the Can-frontend claim.",
                "Use a supported TypeScript/browser client with generated wire types and position Can as the backend in the full application.",
                "Keep browser implementation in another stack, with checked HTTP contracts, and label Can a server-language recommendation.",
            ),
            "server_only_universal": (
                "Keep SSR/HTMX as the sole client mechanism and still advertise Can for every frontend style.",
                "Make the universal SaaS frontend claim without adding local browser execution or a separate client stack.",
                "Treat existing server-rendered interaction as sufficient for offline, canvas and optimistic clients alike.",
            ),
        },
    },
}

for key, variants in facts.items():
    assert len(variants) == 3 and len(set(variants)) == 3, key
for key, question in questions.items():
    assert len(set(question["ask"])) == 3, key
    for option, variants in question["options"].items():
        assert len(variants) == 3 and len(set(variants)) == 3, (key, option)

for index in range(3):
    payload = {
        "model": "jev-latest",
        "state": {key: variants[index] for key, variants in facts.items()},
        "questions": {
            key: {
                "type": "choice",
                "instructions": question["ask"][index],
                "criteria": {option: variants[index] for option, variants in question["options"].items()},
            }
            for key, question in questions.items()
        },
    }
    (OUT / f"request-{index + 1}.json").write_text(json.dumps(payload, indent=2) + "\n")

(OUT / "wording-audit.json").write_text(
    json.dumps(
        {
            "fact_fields": list(facts),
            "question_alternatives": {key: list(question["options"]) for key, question in questions.items()},
            "mechanical_check": "Every explanatory fact, instruction and option has three distinct phrasings. Stable technical identifiers and all alternative sets are unchanged.",
            "semantic_check": "Coordinator compared corresponding triples for equivalent observations, scope caveats and choices before dispatch. Rewording does not prove absence of framing bias.",
            "prior_answers_excluded": True,
        },
        indent=2,
    ) + "\n"
)

print("Prepared three request payloads; no live call unless --send is present.", flush=True)

if "--send" in sys.argv:
    api_key = os.environ["TYPESAFE_API_KEY"]
    for index in range(1, 4):
        request = urllib.request.Request(
            "https://api.typesafe.ai/v1/systemone",
            data=(OUT / f"request-{index}.json").read_bytes(),
            headers={"Content-Type": "application/json", "Authorization": "Bearer " + api_key},
        )
        started = datetime.datetime.now(datetime.timezone.utc).isoformat()
        with urllib.request.urlopen(request, timeout=60) as response:
            body, status = response.read(), response.status
        (OUT / f"response-{index}.json").write_bytes(body + b"\n")
        (OUT / f"response-{index}.metadata.json").write_text(
            json.dumps({"startedAt": started, "status": status, "modelEndpoint": "v1/systemone"}, indent=2) + "\n"
        )
        decoded = json.loads(body)
        print(
            json.dumps(
                {
                    "request": index,
                    "model": decoded.get("model"),
                    "choices": {key: answer.get("choice") for key, answer in decoded.get("answers", {}).items()},
                    "usage": decoded.get("usage"),
                }
            ),
            flush=True,
        )
