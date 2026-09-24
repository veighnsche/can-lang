"""Save three freshly phrased Jev requests and responses per support set."""

from pathlib import Path
import datetime
import json
import os
import sys
import time
import urllib.error
import urllib.request

OUT = Path(__file__).resolve().parent


def q(instructions, **criteria):
    return {"instructions": instructions, "criteria": criteria}


SETS = {
    "native-ai": {
        "state": [
            "Can targets AI coding agents. Correct behavior, explicit finite errors and reliable edits outrank the secondary measure of complete tokens per successful task. There are no external users or compatibility duties. Generated TypeScript should delegate equivalent operations to JavaScript/Bun with narrow contract adapters. Current native AI forms keep llm, judge, fetch and question kinds distinct, with grouped state and explicit connections. The example already composes typed LLM generation with a dynamic Choice in two requests. Ordinary llm wrappers forward errors but cannot separate authored and decoder failures of one nominal kind; fetch/judge wrap has handles native and handles emitted provenance policy. Judge batches prepare registrations, make one native request, validate every answer, then run handlers in source order; an early external write survives a later handler failure. Named functions currently adapt grouped judge/LLM inputs; direct references are disallowed and repetition alone has no measured cost. Compare alternatives on one generated-record flow with same-kind authored/decoder faults, two state groups, multi-judge fan-out, a second handler failure after a controlled ledger write, request count, disclosure, errors, effects and agent repair. This is an experiment choice, not syntax or adoption.",
            "For this agent-oriented language, success means sound contracts and dependable program edits; whole-task token use matters afterward. Old spellings need no preservation, and native JS/Bun should perform equivalent work behind only necessary adapters. Can already has separate llm, judge, fetch and question declarations plus grouped state tied to written connection identities. A working example makes a typed LLM result and then a dynamic Choice using two calls. Existing ordinary forwarding around llm cannot distinguish a decoder-origin error from an authored error sharing its nominal identity, whereas fetch/judge wrap can distinguish handles native from handles emitted. A judge runs one batch request and checks all answers before invoking registered handlers in order, so a first handler's provider effect is not rolled back when the next fails. A named function provides an ordinary callable for grouped judge/LLM state; direct references are excluded, and no agent benchmark establishes harm from adapters. Use a controlled flow testing provenance, grouped fan-out, a ledger write followed by failure, exact requests, state disclosure, outward bounds and edit/repair outcomes. Select bounded trials only.",
            "Assess preparation for Can, whose programs are written by agents: semantic safety and correct repairs are hard gates, with total successful-task tokens a secondary outcome. With zero outside users, migration compatibility is irrelevant; generated TS should call native JavaScript/Bun equivalents plus minimal Can-contract adaptation. Separate llm, judge, fetch and question forms currently preserve state groups and connection identities. Typed generation followed by runtime Choice already works in an example with two requests. Manual llm forwarding loses origin information when an authored and decoder fault have the same nominal kind; fetch/judge wrap already carries handles native/handles emitted provenance. A judge validates one full batched response before ordered handlers, but cannot reverse handler one's external effect if handler two fails. For grouped inputs, a named adapter is accepted as a callable; a direct judge/LLM reference is not, and adapter cost is unmeasured. The decisive case combines a generated record, same-kind faults, two groups and several judges, and a controlled first-write/second-failure sequence; inspect request count, disclosure, error bounds, ledger state and agent attempts. No response selects grammar or production behavior.",
        ],
        "questions": {
            "di17a_recovery": q(
                ["Which bounded recovery experiment should follow for same-kind authored and decoder llm failures, without assuming the new policy is needed?", "What is the best next comparison for origin-sensitive llm recovery under the stated controlled fault case?", "Choose a trial that resolves whether llm failure provenance warrants a native policy extension."],
                ordinary_helpers=["Keep named forwarding/recovery helpers and explicit branches; benchmark whether they correctly avoid wrong recovery and extra request replay.", "Measure shared ordinary adapters with written match arms, leaving origin handling manual and testing whether agents preserve request and failure behavior.", "Use reusable source-level forwarding plus explicit recovery paths as the control; count faulty retries and repair effort without changing native llm policy."],
                llm_wrap_trial=["Prototype a narrow llm wrap origin tag and declared handles native/handles emitted policy; test request, decode, validation and authored errors with finite outward bounds.", "Try extending the existing native wrap mechanism solely to llm calls, preserving private fault origin and public finite errors across request/decode/validation/authored cases.", "Build an llm-specific provenance adapter at the native boundary, with explicit eligible recovery arms and no recovery outside policy; compare against ordinary helpers."],
            ),
            "di17b_effects": q(
                ["How should the controlled two-handler batch test examine partial external effects before any language change?", "Select the most useful bounded handling trial when handler one writes and handler two fails after batch validation.", "Which workflow comparison best clarifies ordered judge handler effects without promising general rollback?"],
                phase_documentation=["Keep the one-request, validate-all, ordered-handler rule; make the partial-effect outcome explicit in docs, diagnostics and a ledger example.", "Clarify and test existing phase semantics, including the surviving first write, in a controlled example without altering batch execution.", "Specify the current sequence and failure propagation precisely, showing that an earlier effect remains after a later handler error."],
                transaction_trial=["Wrap compatible database writes in an explicit native transaction; inject rollback and uncertain commit, without claiming atomicity for provider calls.", "Compare a written transaction for ledger effects, testing failure and indeterminate commit while keeping external operations outside its rollback guarantee.", "Trial native database transaction use around eligible handler writes and probe rollback/commit uncertainty; do not infer arbitrary effect reversal."],
                outbox_trial=["Stage intentions in an outbox after validated answers, then dispatch idempotently; inject duplicate delivery and replay failures.", "Compare an explicit durable outbox with stated at-least-once and idempotency behavior after response validation, including duplicate dispatch.", "Test persisted effect plans and idempotent delivery after the batch, measuring replay and failure paths rather than changing judge's native request."],
            ),
            "di17c_callable": q(
                ["Which grouped-state callable trial is justified before changing direct judge/LLM reference rules?", "Choose the next experiment for multi-judge fan-out while retaining group and connection identity.", "How should direct native AI references be compared with the current callable adaptation?"],
                named_adapters=["Use one named function per grouped state contract, optionally with explicit context; test renames, captures, request counts and finite errors.", "Keep ordinary named adapters with written group/context fields and measure fan-out editing, disclosure and failure propagation.", "Run the controlled fan-out with named wrappers as callable values; verify group fields, connection rename, capture-once and outward bounds."],
                direct_reference_trial=["Prototype a narrowly typed grouped-state native reference that retains group and connection descriptors through capture, invocation and fixtures.", "Try a direct judge/LLM callable contract limited to grouped state, checking disclosure and explicit connection identity without flattening or inferred errors.", "Build a constrained native AI reference lowering to a closure on the existing call path with checked group/connection data, then compare agent edits."],
            ),
        },
    },
    "lifetime-race": {
        "state": [
            "Can serves AI agents; correctness, safety and repair success are gates, while all tokens in a successful task are a secondary metric. No external compatibility is required; native JS/Bun operations should carry equivalent behavior. Resource owners and leases enforce scope at runtime before native use. Returning an own scoped handle, including in a record or closure, later fails resource_state; returning an enclosing live pool can be valid. Four runtime escape shapes have been probed, but no sound useful static checker rule is established. First-success race uses native Promise.any: pre-winner failures are consumed, late standard failures are diagnosed, losers retain leases, and empty/pending dynamic behavior remains selected. A close deadline stops the caller's wait but neither cancels pending work nor frees its lease. Test direct/aggregate/captured own escapes versus live pool, and a real fallback with zero/one/many participants, early win, expected/accidental faults, late standard fault, pending loser and close deadline. Compare diagnostic usefulness, noise, root drain and agent repairs. Ask for bounded trials, not grammar or final semantics.",
            "The design target is dependable agent-written Can code; include complete tokens only after contract and safety checks pass. Legacy compatibility has no constituency, and generated TypeScript should use native JavaScript/Bun selection with adapters for ownership. A scoped transaction or reader belongs to an owner: an escaped own token is rejected by a pre-native runtime lease guard even if carried through an aggregate or capture, while an outer pool may remain usable. Current probes show four forms, without proving a worthwhile static analysis. First-success coordination delegates selection to Promise.any. A failure before a later success is consumed, a late standard failure is reported, a losing live operation keeps its lease, and a timeout on close does not settle or cancel it. Dynamic empty and permanently pending cases retain their contract. Exercise own-handle return in direct, field and closure forms plus outer pool; also exercise replica fallback counts, fault timing, pending loser and shutdown. Judge both fault signal/noise and repair effort; this is preparation only.",
            "For agent-first Can, explicit contracts and safe effects are mandatory, with whole successful-task tokens measured second. No old ABI or spelling needs migration. Compile equivalent coordination to native Promise.any and guard resource use at the owner boundary. A scope's own reader/transaction can escape syntactically but its later use receives resource_state before native work; an enclosing pool with a live owner is legitimate. A four-case runtime probe exists, yet no static escape diagnostic has been validated. In first-success races, earlier failed candidates do not determine the winner, late standard faults are diagnosed, and losing operations hold leases until settlement. Empty dynamic input and never-settling candidates have selected behavior; a close deadline only limits waiting and reports cleanup failure. Evaluate direct, record and captured own returns against a valid outer return, then a controlled fallback with participant counts, early success, fault categories, late fault and close deadline. Require unchanged winner/drain behavior and count agent diagnosis and noise. The consultation chooses experiments, not surface syntax.",
        ],
        "questions": {
            "di18a_escape": q(
                ["Which next diagnostic experiment is proportionate to scoped resource escapes while preserving valid outer-pool returns?", "Choose a bounded scope-escape comparison that improves agent repair without overstating static coverage.", "What trial should address expired own handles given the existing runtime safety boundary?"],
                runtime_context=["Improve resource_state with creation, close and use context while retaining the owner/lease guard; measure repairs for all escape forms.", "Keep pre-native runtime rejection and add provenance to the fault diagnostic, then test how agents fix direct, field and capture cases.", "Enrich runtime ownership errors with the handle's creation/closure/use trail and compare repair attempts, including valid outer pools."],
                narrow_checker=["Prototype a checker warning for obvious direct and known field returns of the scope's own token; avoid rejecting outer pools or claiming alias/capture completeness.", "Try a limited compile-time diagnostic for syntactically evident own-handle return and known aggregate fields, explicitly excluding arbitrary aliases.", "Compare a precise static check for direct/known-record own escapes with runtime feedback; test false positives on enclosing-owned resources."],
            ),
            "di18b_faults": q(
                ["Which observability trial best tests consumed first-success race failures without changing Promise.any selection?", "Select a bounded way to compare fault visibility on an actual replica fallback while retaining winner and lease rules.", "How should hidden pre-winner faults be evaluated before adding any race policy?"],
                application_observation=["Have service code observe each participant's classified failure before the original race settles; measure useful diagnoses and routine noise.", "Instrument every fallback candidate explicitly in the application and record faults while leaving native selection, pending work and leases intact.", "Compare written participant observers around Promise.any, with service-owned redaction and fault reporting, on the controlled fallback."],
                opt_in_policy=["Prototype optional consumed-fault diagnostics at the race adapter; specify domain/standard inclusion, identity, timing, order, redaction and sink failure handling.", "Try an opt-in race observer that reports consumed failures without canceling losers or changing winner, pending, lease or root-drain semantics.", "Compare a native-boundary diagnostic policy for swallowed pre-winner faults, with explicit fault classes and sink behavior, against application reporting."],
            ),
        },
    },
    "catalogue-collections": {
        "state": [
            "Can is for AI agents; correctness and explicit contracts are gates, complete tokens per successful task secondary. No external users require compatibility. Can code cannot embed JavaScript/TypeScript or import arbitrary Bun APIs; the distribution owns admitted catalogue operations, which lower to native JS/Bun plus necessary contract adapters. Typed HTTP, crypto and process operations exist, and no required payment/email/queue SDK has been shown impractical through them. A concrete invoice/webhook provider trial must cover authentication, idempotency, malformed payload, drift, timeout, redaction, immutable output, cleanup and deterministic raw fixtures. Map/Set values are opaque and immutable to callers, backed by native Map/Set; each insert/replace clones, implying quadratic copies for mostly unique folding, but throughput is unmeasured. Compare realistic duplicate-heavy and unique sizes, order/equality/duplicate rules, callback failure timing, aliases, failed builds, time/allocation and agent repair. Retain no generic npm import or mutable alias. These are bounded next trials, not new syntax or product decisions.",
            "Agent completion and sound contracts lead this comparison; tally all successful-task tokens afterward. Old generated layouts need no compatibility, and equivalent operations should run through JavaScript/Bun with minimal adapters. The language admits a controlled distribution catalogue only: source cannot freely import JS/Bun. Existing typed fetch, cryptography and process calls can implement provider protocols; no documented necessary SDK defeats that route yet. Use a named invoice/webhook API and inject auth, duplicate event/key, bad payload, version drift, timeouts, sensitive diagnostics, lifecycle, immutable result and fixture behavior. Immutable collections currently wrap native Map and Set, copying the container at each point update; a mostly distinct-key fold therefore has a quadratic copying implication, without measured performance data. A bulk test must fix key equality, first insertion order, duplicate replacement, callback order/errors, no mutable exposure and realistic volumes. Compare native bulk construction and a tightly scoped internal builder only if evidence warrants. This consultation chooses experiments.",
            "For an AI coding language, valid effects and reliable repair outrank token savings, which are assessed over an entire successful attempt. There is no external migration obligation. Generated TypeScript delegates to native facilities, with adapters to keep Can immutable and contract-bound. The source language has no arbitrary npm or JS escape; approved catalogue entries belong to the distribution. Typed HTTP, crypto and process support is already present, and an indispensable blocked SDK operation has not been identified. Test one provider operation for invoice/webhook flows with signature/auth, deduplication, malformed and changed responses, timeout, redaction, raw fixtures, ownership and immutable values. Can maps and sets hide native Map/Set; single-item updates clone before publication. Distinct-key construction may copy quadratically but has not been benchmarked. Trial duplicate-rich and mostly unique batches, midstream callback failure, insertion order, key equality, alias attempts and a failed build at real sizes; compare time, allocation and total agent cost. No answer admits general imports or public mutation.",
        ],
        "questions": {
            "di19_provider": q(
                ["Which bounded integration trial should follow the HTTP baseline before admitting an external SDK mechanism?", "Select the next provider-boundary experiment given no demonstrated required SDK blockage.", "How should the catalogue respond to a concrete invoice/webhook provider gap if one is reproduced?"],
                protocol_catalogue=["Build a named typed HTTP/crypto/process package; add only a specifically missing controlled catalogue primitive with fixtures and finite errors.", "Use the protocol client as control, then fill a proven low-level admitted-operation gap rather than widening source imports.", "Trial provider handling through native fetch/crypto/process and narrowly extend distribution primitives only when the operation cannot be expressed."],
                distribution_adapter=["Compare a distribution-owned provider adapter with reviewed signatures, codecs, fixture ownership, abort/lifetime, redaction and finite errors.", "If the protocol path is materially costly, test a narrowly admitted native SDK adapter in the distribution with deterministic contract checks.", "Prototype one maintained provider binding under catalogue authority, specifying errors, fixtures, ownership, cancellation and sanitized diagnostics."],
                binding_manifest=["Only after a named SDK feature defeats the first paths, trial a narrowly reviewed third-party binding manifest with the same admission contracts.", "Reserve a constrained external SDK manifest experiment for a reproduced operation impossible or materially impractical through HTTP and distribution code.", "Test a third-party manifest solely for a measured SDK-only capability, with explicit signatures, codecs, fixtures, ownership and error admission."],
            ),
            "di20_bulk": q(
                ["Which collection construction experiment best addresses the cloning implication while preserving immutability?", "Choose the next bounded bulk-map/set trial after realistic size and error behavior are measured.", "What construction path should be compared with immutable folds on actual workloads?"],
                retain_folds=["Keep immutable per-item insert/add folds while measured workloads remain small or no material performance/agent benefit appears.", "Use current cloning updates as the baseline and defer additions unless realistic throughput or completion trials expose a meaningful cost.", "Continue source folds if measured time/allocation and successful agent tasks do not justify another API."],
                bulk_operations=["Trial specific native Map/Set grouping or aggregation operations that build once and publish one immutable value with defined order, equality and callback failures.", "Compare targeted bulk functions using internal native mutation and a single immutable publication, specifying duplicates and callback timing.", "Test bounded bulk aggregation lowered to one native container build, with no exposed mutators and exact duplicate/order/error contracts."],
                scoped_builder=["Only if bulk operations cannot express the measured workload, trial a nonescaping internal builder invalid after immutable publication.", "Compare an owner-like temporary Map/Set builder when fixed bulk APIs fail; reject post-publication use and mutable aliases.", "Prototype a scoped native collection builder as a fallback, with failed-build cleanup and one immutable result, after targeted bulk operations are tested."],
            ),
        },
    },
    "authoring": {
        "state": [
            "Can is agent-first: correct semantics, explicit contracts and edit reliability are hard gates; full tokens per successful task are secondary. Human visual taste is not a goal and no outside users require compatibility. Positional nominal record construction follows declaration order; adjacent int total and int recent show a same-type reorder risk, but no controlled production misbinding has been observed. Strong domain types, named factories and with replacements are available; a factory merely forwarding two ints does not remove the ambiguity. Calls, constructors and assertions currently use one physical line. A review counted 154 example lines above 120 characters, longest 336, but no agent failure was measured. Precise spans and a comment-preserving formatter are already accepted; multiline syntax is deferred. The checker rejects some immediately forwarded locals such as meaningful permitted, with no quantified agent cost. Compare creation, same-type reorder/rename, long-form edit, parser/formatter repair and meaningful versus redundant local on the same controlled agent task. Keep nominal immutability, evaluation order and assertion behavior. This selects trials, not exact syntax or policy.",
            "For Can's coding agents, success is a sound build and dependable refactoring; measure total prompt/code/diagnostic/retry tokens only for successful work. Source aesthetics have no independent weight and legacy syntax needs no migration. A record constructor binds by position, making two neighboring int fields total/recent vulnerable to silent swaps, although the example is not a measured product failure. Domain-distinct types, factories and named with updates are present; two same-type factory parameters can preserve the hazard. One-line calls, record expressions and assertion rows are the grammar today. The 154 long example lines and 336-character maximum indicate layout pressure only. Diagnostic spans and canonical comment-preserving formatting are accepted work; continuation remains deferred. A local named permitted may be rejected as immediate forwarding, but no agent benchmark demonstrates the consequence. Use controlled field reorder/add/rename, lengthy expression edits, comment-preserving parse/format/parse, attached assertions and both useful/redundant locals. Demand valid values, exact spans, unchanged effects and repair measurements; do not select spelling.",
            "Assess an authoring experiment for an AI-agent language, with correctness and contract preservation first and whole successful-task tokens second. No compatibility or human-comfort requirement is imposed. Nominal immutable records are currently built by declaration-order arguments; same-typed total and recent illustrate possible reorder misbinding, not an established agent mistake. Strong types, constructor helpers and with updates provide current controls, though a pass-through factory with two ints remains positional. Calls/constructors/assertion rows are line-bound. Examples include 154 lines over 120 characters and one of 336, but width alone proves no editing failure. Precise diagnostics and comment-safe canonical formatting are already chosen; multiline continuation is deferred. Immediate forwarded locals, including a meaningful permitted step, can be invalidated by the checker, with no observed task penalty yet. Compare agents creating, renaming and repairing these constructs, including redundant aliases; validate nominal identity, evaluation count, assertion attachment, spans, first-pass success and total task costs. This is technical advice only.",
        ],
        "questions": {
            "di22_fields": q(
                ["Which next trial should resolve whether named-field construction reduces same-type reorder errors?", "Choose an experiment for positional constructor safety under controlled add, reorder and rename edits.", "How should same-typed record field refactoring be compared before a new form is adopted?"],
                positional_controls=["Retain positional construction with stronger domain types or genuinely safe factories; compare silent swaps and agent repairs against the baseline.", "Test existing positional calls with nominally distinct field values or checked helper contracts, avoiding pass-through same-int factory claims.", "Use current construction plus domain types/factories as the control and measure correctness and repair on field edits."],
                named_form_trial=["Prototype nominal named-field construction with duplicate, missing and unknown checks plus defined evaluation order; measure reorder safety.", "Compare a compile-time named constructor that binds by declared field identity and diagnoses omissions/repetition, preserving immutable output.", "Trial explicit field-name binding lowered to the existing frozen record operation, with exact diagnostics and no implicit defaults."],
                tooling_trial=["Try a checked constructor rewrite or suspicious reorder diagnostic; quantify missed edits and false positives.", "Compare IDE/compiler assistance for same-type positional swaps, measuring detection coverage and spurious warnings.", "Test source-tool rewrites or diff checks that flag likely argument reorder mistakes without changing grammar."],
            ),
            "di23a_layout": q(
                ["Which layout trial should follow accepted formatter and span work if long-form repair remains costly?", "Select a comparison for lengthy calls, constructors and assertions after tooling improvements.", "How should physical-line limits be evaluated on agent editing tasks?"],
                formatter_first=["Complete canonical comment-preserving formatting, precise spans and refactor helpers under current one-line grammar; measure repair outcomes.", "Keep existing line rules while improving formatter and error locations, then benchmark agents on long expressions and assertion rows.", "Use accepted diagnostic/format tooling as the control, preserving parse/format/parse and attached assertion semantics."],
                delimited_continuation=["Only for remaining measured failures, trial newline-as-whitespace inside explicit delimiters with specified comments, trailing separators and recovery.", "Prototype bounded continuation inside parentheses/brackets if tooling leaves a reproducible problem; retain block indentation and row attachment.", "Compare delimiter-only multiline forms after the baseline, defining terminators, nesting, formatter idempotence and parser diagnostics."],
            ),
            "di23b_local": q(
                ["Which validity-policy trial should test an immediately forwarded but meaningful local?", "Choose a bounded comparison for the checker rejecting permitted-like names.", "How should useful intermediate locals and accidental aliases be handled in the next agent trial?"],
                hard_rule_fix=["Retain rejection with a precise diagnostic and safe automatic inline fix; measure whether it prevents mistakes or costs repairs.", "Keep the current validity rule but improve the source span and checked fix, then test meaningful and redundant aliases.", "Use hard rejection plus an accurate autofix as control and count build retries and semantic mistakes."],
                advisory_lint=["Allow the local and report style advice, preserving an intentionally named step; check evaluation and agent completion.", "Trial a nonblocking lint for immediate forwarding while agents may retain meaningful intermediate names.", "Compare accepting ordinary locals with an advisory suggestion, ensuring one evaluation and unchanged error behavior."],
                semantic_only_rule=["Reserve hard errors for a demonstrable semantic ambiguity, with exact negative examples; make pure style cases advisory.", "Try a narrowly defined rejection only where the alias causes a real contract ambiguity, after specifying counterexamples.", "Test whether a precise semantic-danger rule can replace style-based invalidity without broad exemptions."],
            ),
        },
    },
}


def payload_for(name, index):
    item = SETS[name]
    questions = {}
    for ident, question in item["questions"].items():
        questions[ident] = {
            "type": "choice",
            "instructions": question["instructions"][index],
            "criteria": {key: variants[index] for key, variants in question["criteria"].items()},
        }
    return {"model": "jev-latest", "state": item["state"][index], "questions": questions}


def validate():
    for name, item in SETS.items():
        assert len(item["state"]) == len(set(item["state"])) == 3, name
        for ident, question in item["questions"].items():
            assert len(question["instructions"]) == len(set(question["instructions"])) == 3, ident
            for key, variants in question["criteria"].items():
                assert len(variants) == len(set(variants)) == 3, (ident, key)
        paths = [json.dumps(payload_for(name, i), sort_keys=True) for i in range(3)]
        assert len(set(paths)) == 3, name


def main():
    validate()
    for name in SETS:
        folder = OUT / name
        folder.mkdir(exist_ok=True)
        for index in range(3):
            (folder / f"request-{index + 1}.json").write_text(
                json.dumps(payload_for(name, index), indent=2, ensure_ascii=False) + "\n"
            )
    if "--send" not in sys.argv:
        return
    key = os.environ["TYPESAFE_API_KEY"]
    for name in SETS:
        for index in range(3):
            folder = OUT / name
            path = folder / f"request-{index + 1}.json"
            body = path.read_bytes()
            metadata_path = folder / f"response-{index + 1}.metadata.json"
            previous = json.loads(metadata_path.read_text()) if metadata_path.exists() else None
            if previous and previous.get("status") == 200:
                continue
            attempts = previous.get("attempts", [previous]) if previous else []
            req = urllib.request.Request(
                "https://api.typesafe.ai/v1/systemone",
                data=body,
                headers={"Content-Type": "application/json", "Authorization": "Bearer " + key},
            )
            for retry in range(5):
                if retry:
                    time.sleep(2 ** retry)
                started = datetime.datetime.now(datetime.timezone.utc).isoformat()
                try:
                    with urllib.request.urlopen(req, timeout=60) as response:
                        result, status = response.read(), response.status
                except urllib.error.HTTPError as error:
                    result, status = error.read(), error.code
                attempt = {
                    "startedAt": started,
                    "completedAt": datetime.datetime.now(datetime.timezone.utc).isoformat(),
                    "status": status,
                    "endpoint": "https://api.typesafe.ai/v1/systemone",
                }
                attempts.append(attempt)
                (folder / f"response-{index + 1}.attempt-{len(attempts)}.json").write_bytes(result + b"\n")
                (folder / f"response-{index + 1}.json").write_bytes(result + b"\n")
                metadata_path.write_text(json.dumps({**attempt, "attempts": attempts}, indent=2) + "\n")
                if status == 200 or status not in (429, 529):
                    break
            print(json.dumps({"set": name, "request": index + 1, "status": status,
                              "answers": json.loads(result).get("answers")}), flush=True)
            if status != 200:
                raise RuntimeError(f"{name} request {index + 1}: HTTP {status}")


if __name__ == "__main__":
    main()
