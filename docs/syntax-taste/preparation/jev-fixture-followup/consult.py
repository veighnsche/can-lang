"""Three independently worded DI-06 Choice consultations; --send calls Jev."""

from pathlib import Path
import difflib
import json
import os
import sys
import time
import urllib.error
import urllib.request

OUT = Path(__file__).resolve().parent

STATE = {
    "goal": [
        "Can is authored by AI coding agents. Prioritize correct behavior, reliable refactors, explicit contracts and independent libraries. Whole successful-task tokens, including prompts, diagnostics and retries, are secondary. Human convenience and short source are not goals. There are no external users or compatibility obligations. Lower ordinary operations to native JavaScript/Bun. Decide DI-06 fixture ownership for an implementation plan, using the evidence rather than assuming a new construct is valuable.",
        "The intended programmer is an AI coding agent, so judge designs by correctness, safe edits, visible guarantees and separately composed packages. Measure total tokens through a completed coding task as a lesser objective; source brevity and human familiarity have no standing. Existing source spellings need no preservation because no external users exist. Generated TypeScript should use equivalent JS/Bun behavior. Select the DI-06 language contract that present observations justify for planning.",
        "For this agent-targeted language, successful execution, dependable changes, inspectable contracts and library composition dominate. Task-wide token use counts context, repair, tests and code but is a secondary measured outcome. Neither human taste nor line count decides. There is no installed-user compatibility constraint. Use native JS/Bun primitives where Can semantics allow. This judgment concerns the fixture rule to schedule, not implementation of it.",
    ],
    "current_mechanism": [
        "Attached assertion roots have full package/declaration/name identities. A lexical when row currently activates by the running root's short display name; its FIFO queue is isolated by full root/table/invocation path and does not search arguments. Renaming caller root customer to renamed, leaving helper unchanged, checked and emitted but changed its behavior: the helper's customer row stopped selecting and text::from_int(7) executed for real. Templates expand at the lexical use-site selector and keep the same weakness. The queue mechanism is not the observed bug.",
        "Can assertions already possess complete root identities and independent FIFO state for lexical tables and invocation paths; allocation never hunts for matching arguments. The fragile part is selection: a helper when row compares the current caller's short label. A current executable rename of customer to renamed in only app/main.can still passed static checking, yet the helper no longer supplied text for text::from_int(7). A reusable template repeats content at that selector and cannot transfer ownership. No queue collision was observed.",
        "A root is internally package/declaration/assertion-name qualified, and fixture occurrence queues distinguish root, lexical table and invocation. They reserve in order without argument-based allocation. Selection nevertheless keys a helper's row to the caller's unqualified root name. The executed customer-to-renamed caller edit did not change the helper, produced no checker diagnostic and caused the internal conversion fixture to fall through to real execution. Inert template expansion retains a local selector. The defect is ambient activation, separate from queue isolation.",
    ],
    "new_probe": [
        "At current compiler revision 02a549d2, eight projects checked/emitted and 28 assertion-root runs were recorded. Ambient helper row and template each passed customer and failed renamed. Caller lexical stub of helper::read() passed both but bypassed the helper body even after the helper was changed to return unexpected text. A normal typed callable argument to helper::read() passed both; the caller-authored converter ran inside the real helper and the helper's suffix executed. Mutating either converter output or helper suffix made the caller fail. That path carried real-can evidence, no fixture-supplied completion. It requires adding a callable parameter to the helper API and does not show feasibility for every noninjectable internal effect. There are no candidate F1/F2/F3 implementations or measured agent-task costs.",
        "The new bounded run compiled all eight prepared Can project variants, then completed 28 root invocations. Both foreign-label fixtures, direct and template, changed result after the caller label edit. Stubbing the whole helper preserved the expected caller completion but an adversarial helper-body change remained invisible. In contrast, giving the helper a typed conversion function as an ordinary input retained caller-label independence and exercised that function plus the helper's own string suffix; changing either side broke the assertion. Reports label this path real-can, rather than a supplied fixture completion. The helper signature gained an input, and no result covers dependencies that cannot reasonably be injected. Exported scenario and seam machinery and whole-task token comparisons have not been built.",
        "A reproducible present-Can comparison checked/emitted eight project forms and ran 28 assertion roots. A helper-owned customer row, including one produced from a template, stopped working after only the caller name became renamed. A caller stub remained green through a malicious helper implementation change because the real helper never ran. A passed-in typed callable yielded stable results after rename while executing its caller-provided conversion and helper suffix; perturbing either part yielded an outcome mismatch. The integration evidence is real-can, not supplied-completion. This ordinary dependency-injection idiom alters the public helper API and the probe cannot establish its suitability for native or hidden targets generally. No new-link candidate or agent-cost trial exists.",
    ],
    "requirements": [
        "Reject implicit foreign-root short-name activation. Preserve attached roots, exact fixture targets and arguments, per-root/table/invocation FIFO, recursion/callable/concurrent-participant isolation, and separate real-can/supplied-completion/raw-provider evidence labels. A deliberate requested scenario or seam that vanishes must diagnose instead of running real code. The recommendation program calls for deliberate cross-package scenario ownership and honest caller-path helper-body coverage; separate helper unit assertions cannot be counted as that integration path. Package identity and protected-value fixture admission are decided elsewhere. Tests should cover duplicate labels, two helper calls, nesting, recursion, concurrent participants, missing links and unused fixtures.",
        "Any design must remove selection by an unrelated caller display label while retaining named attached assertions, checked target/argument pairs, isolated FIFO state across full roots, tables and invocations, and reservation for recursion, callables and parallel participants. Evidence labels distinguish real Can execution from supplied completion and raw provider fixtures. Explicitly selected missing links are errors, never silent real-call fallback. The program wants cross-library simulation deliberately owned and coverage through the helper body when claimed; a separate helper unit root is not the same caller path. Canonical package owners and validated fixture values are other decisions. Include duplicate short names, repeated and nested invocations, concurrent work, vanished selections and unused rows in validation.",
        "The fixture contract must cease foreign-root name matching. Keep full attached root identity, exact target and argument checks, FIFO by root/table/invocation, and distinct reservation paths through recursion, higher-order calls and concurrent participants. Do not merge real-can, supplied-completion or raw-provider-fixture evidence. If a caller explicitly chooses a missing scenario or seam, fail visibly rather than fall back. Required product coverage means the caller can deliberately exercise helper internals under a controlled dependency; mere independent unit coverage does not qualify. Other design tracks determine package IDs and owner-created values. Acceptance also includes duplicate display names, two occurrences, nested calls, recursion, participants, signature drift and unused bindings.",
    ],
    "decision_policy": [
        "Choose the smallest coherent fixture contract supported by current evidence, while treating callable injection as integrated real-code testing rather than a fixture row. An alternative that cannot meet deliberate cross-package helper-body simulation in the target acceptance case must be described as limited. A new feature requires a reproducible baseline gap or measured improvement after current idioms; no candidate implementation or agent efficiency result exists. If the remaining noninjectable-effect case prevents selection, choose the bounded discriminating trial, not a speculative adoption.",
        "Recommend what planning can defensibly schedule now. Credit normal typed dependency injection for the caller-through-helper path it actually exercised, but do not claim it proves every effect can be exposed as an API input. New scenario/seam syntax carries parser, checker, emitter, runtime and tooling costs and needs evidence of an unmet case. Conversely, restricted lexical rows plus outer stubs alone cannot claim an integrated simulated helper path. If evidence cannot distinguish the candidate contracts, specify the additional controlled comparison.",
        "Base the next implementation-plan disposition on observed coverage and explicit failure conditions. Current callable arguments are real integration, with a public signature cost and an untested unsuitable-target limit. Published fixture links add language and runtime surface and have not been prototyped. Lexical restriction repairs accidental activation but isolated helper assertions are separate coverage. Select a complete warranted rule when possible; otherwise pick the exact remaining experiment. Neither Jev agreement nor confidence substitutes for executed contract tests.",
    ],
}

QUESTION = [
    "Which DI-06 contract should the implementation plan carry on this evidence, or which single discriminating trial is still necessary before choosing? Judge required caller-path integration and implementation scope together.",
    "Given the newly measured injection path and the still unmeasured API cost, what is the defensible next fixture-ownership disposition for Can? Choose one complete route or a narrowly justified experiment.",
    "How should DI-06 be resolved for ordered tasks now, accounting for foreign-name failure, real helper coverage, and costs of exported links? Choose the strongest supported disposition.",
]

OPTIONS = {
    "lexical_injection": [
        "Adopt F1 restricted to same-owner lexical rows: foreign caller labels never select helper rows. A caller stubs the whole helper only when helper internals need not run; for integrated simulated behavior the helper accepts ordinary typed dependencies, and separate helper assertions cover local rows. This meets the observed injected case without new cross-package fixture syntax, but introduces API parameters and cannot claim arbitrary internal effects are injectable. Real-can paths remain labeled real-can.",
        "Choose same-scope fixture selection and ordinary callable injection as the integration mechanism. Remove ambient cross-library row activation; preserve whole-helper stubs for caller-only tests and helper-local assertion rows for unit cases. The tested helper path stays real-can while receiving a caller-authored typed function. Accept extra helper inputs and document that noninjectable targets need a later proof before adding language links.",
        "Restrict when rows to their own test scope, with templates only reusing local content. Callers either supply the helper's complete result or explicitly pass a typed dependency through its API when the helper body must participate. The passing injected program is real execution. This avoids exporting fixture link constructs; it leaves an acknowledged limit when a dependency cannot appropriately enter the public signature.",
    ],
    "exported_scenarios": [
        "Adopt F2: a helper exports a named configuration of its own lexical rows, and a caller assertion checks an explicit callee-declaration/scenario link at one invocation. Nested scenarios are declared; invocation paths isolate recursion and participants. Missing, inaccessible, ambiguous, unused or incompatible links diagnose. The helper body runs under supplied internal completions without changing its functional API, at the cost of a published test contract and new compiler/runtime routing.",
        "Select helper-owned exported scenarios. An assertion in another package requests a particular helper scenario at a checked call site, independently of both assertion display names. The helper keeps private lexical targets behind the exported scenario name; nested, recursive and concurrent invocations receive distinct routes. Broken/unused links are static or run-time failures. This allows integrated supplied behavior without adding a conversion parameter but needs a new scenario API and plan handling.",
        "Define exported named helper fixture bundles with typed caller links to the helper declaration and selected invocation. Helper internals execute while their selected local rows supply results; the public functional helper signature need not grow. Diagnose moved/missing/ambiguous scenarios, mismatch and unused requirements; retain queue paths. This gives deliberate cross-package simulation but expands the language and makes scenario names an authored contract.",
    ],
    "exported_seams": [
        "Adopt F3: helper authors export typed symbolic seams at approved internal lexical calls. A caller assertion binds content to a named seam along a checked helper-call path. The helper owns target/signature and caller owns supplied behavior; missing or unused required bindings diagnose. Queue identity includes invocation path. This gives precise control without changing the functional API, while exposing internal structure and adding new checker/runtime link machinery.",
        "Choose helper-approved exported seam points. Caller roots provide typed fixture rows for specific declared seam identities, traversing a checked invocation path. Signature changes, absent seams and unused bindings fail instead of executing real calls. Each recursive or concurrent invocation keeps its own FIFO allocation. Functional APIs stay unchanged, but tests become coupled to more of the helper's internals and need new compiler routing.",
        "Create explicit typed fixture seams owned by the callee and selected by the caller for permitted lexical targets. A checked path and invocation identity resolve nested calls, recursion and participants. Reject missing, incompatible or unused selections. The caller can simulate exactly one internal dependency without a new helper parameter, trading precision for structural coupling and a sizeable language/runtime feature.",
    ],
    "discriminating_trial": [
        "Do not adopt F1/F2/F3 yet. First run an integration case where the helper dependency cannot reasonably be an ordinary typed input, compare a current injection/stub baseline with small exported-scenario and exported-seam prototypes, and test label rename, missing links, two occurrences, nested/recursive/concurrent paths, diagnostics, evidence labels and complete agent-task tokens. Then choose the smallest passing contract. This schedules a bounded design experiment rather than silently treating the current injected case as universal.",
        "Schedule a narrow remaining comparison before a semantic choice: one noninjectable internal target in the same helper-body acceptance path, with ordinary callable injection and outer stub controls plus limited scenario/seam link implementations. Exercise changed caller label, vanished binding, repeated and nested invocations and isolated queues; measure creation/refactor/repair tokens through success. The existing probe proves one happy injection case only.",
        "Reserve adoption pending a controlled hard case unsuitable for public dependency injection. Contrast current API-level test double and whole-helper stub against minimally implemented exported scenario and seam links, including rename stability, broken-link failure, repeated/nested/concurrent identity and evidence provenance. Count full successful-task cost. This option recognizes no replacement has been executed yet and targets the remaining coverage question.",
    ],
}

def strings(value, path=""):
    if isinstance(value, str):
        yield path, value
    elif isinstance(value, dict):
        for key, item in value.items():
            yield from strings(item, f"{path}.{key}" if path else key)

def make_payload(index):
    return {
        "model": "jev-latest",
        "state": {key: variants[index] for key, variants in STATE.items()},
        "questions": {
            "fixture_contract": {
                "type": "choice",
                "instructions": QUESTION[index],
                "criteria": {key: variants[index] for key, variants in OPTIONS.items()},
            }
        },
    }

def main():
    payloads = [make_payload(i) for i in range(3)]
    maps = [dict(strings(payload)) for payload in payloads]
    for a in range(3):
        for b in range(a + 1, 3):
            assert maps[a].keys() == maps[b].keys()
            repeated = [key for key in maps[a] if key != "model" and key != "questions.fixture_contract.type" and maps[a][key] == maps[b][key]]
            assert not repeated, (a, b, repeated)
    audit = {
        "fact_and_option_equivalence_review": "Manually checked: all three state variants include identity/goal, exact present selector and queue mechanics, rename/template/stub/injection observations and coverage labels, API-cost limit, no candidate or agent-cost evidence, shared guarantees and four alternatives. No prior Jev result appears in any request.",
        "prose_fields_per_request": len(maps[0]) - 2,
        "all_explanatory_strings_pairwise_distinct": True,
        "pairwise_similarity_by_field": {
            f"{a+1}-{b+1}": {key: round(difflib.SequenceMatcher(None, maps[a][key], maps[b][key]).ratio(), 3) for key in maps[a] if key not in ("model", "questions.fixture_contract.type")}
            for a in range(3) for b in range(a + 1, 3)
        },
    }
    OUT.mkdir(parents=True, exist_ok=True)
    for i, payload in enumerate(payloads, 1):
        (OUT / f"request-{i}.json").write_text(json.dumps(payload, indent=2) + "\n")
    (OUT / "wording-audit.json").write_text(json.dumps(audit, indent=2) + "\n")
    if "--send" not in sys.argv:
        return
    key = os.environ.get("TYPESAFE_API_KEY")
    if not key:
        raise RuntimeError("TYPESAFE_API_KEY is absent")
    for i in range(1, 4):
        data = (OUT / f"request-{i}.json").read_bytes()
        request = urllib.request.Request(
            "https://api.typesafe.ai/v1/systemone",
            data=data,
            headers={"Authorization": f"Bearer {key}", "Content-Type": "application/json"},
            method="POST",
        )
        for attempt in range(4):
            try:
                with urllib.request.urlopen(request, timeout=90) as response:
                    body = response.read()
                    metadata = {"status": response.status, "attempt": attempt + 1, "model_requested": "jev-latest"}
                (OUT / f"response-{i}.json").write_bytes(json.dumps(json.loads(body), indent=2).encode() + b"\n")
                (OUT / f"http-{i}.json").write_text(json.dumps(metadata, indent=2) + "\n")
                break
            except urllib.error.HTTPError as error:
                body = error.read().decode(errors="replace")
                (OUT / f"http-error-{i}-{attempt+1}.json").write_text(json.dumps({"status": error.code, "body": body}, indent=2) + "\n")
                if error.code not in (429, 529) or attempt == 3:
                    raise
                time.sleep(2 ** attempt)

if __name__ == "__main__":
    main()
