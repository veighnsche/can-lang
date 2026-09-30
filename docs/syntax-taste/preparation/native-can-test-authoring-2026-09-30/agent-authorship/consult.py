"""Advisory consultations about agent authorship; no test implementation."""
import datetime
import json
import os
from pathlib import Path
import sys
import urllib.error
import urllib.request

OUT = Path(__file__).resolve().parent
CONTEXTS = [
    "Can is designed for AI coding agents, not comfortable human hand-writing. Evaluate reliable generation, local reasoning, explicit contracts, precise diagnostics and safe mechanical edits; useful repetition is acceptable. This is a design-only review with no builds, agent trials or benchmarks authorized. Tests must support arbitrary normal Can functions; Can owns scenarios, expected values, policy and reports; authored foreign harnesses must disappear. Generic native mechanics are allowed. Coverage, bounded laptop load/storage and cleanup remain mandatory. There are no compatibility constraints; the execution backend is open. The current proposal uses ordinary case functions, a static callable registry, manual string required_checks plus string check IDs in expectation calls, constructible context with reporter/operation callables, mandatory offline roots supplying boundary results through match call/when/scenario links, and separate live workers. Current Can requires assertions on every concrete fn; when cannot decorate match chain; opaque context inputs are not generally elided; nested record callable equality needs care. These are current constraints, not immutable language goals. A declared check inventory catches early ok only if independent from executed calls; deriving obligations from observed events is unsafe. Shared typed Can check descriptors can give IDs one source while keeping a separate declared obligation plan, but need library validation and cannot alone prove an oracle correct. Plain strings with validation are simpler to implement; compiler-owned test declarations would add discovery/checking semantics while allowing ordinary bodies. The proposal also requires whole-suite offline verification before any selected live run. Existing canlc assert checks the whole source graph but permits selected root execution. A proposed focused development path could check all source, verify the selected case and a conservatively known dependency/fixture closure, fall back to all roots when uncertain, and label results partial. Full qualification would still require all roots/cases and environments; candidate/executor/suite digests bind evidence, and no full bundles persist by default. No performance or agent-repair advantage has been measured.",
    "The intended Can author is an AI agent; human typing ease is not a goal. Prefer dependable code generation, bounded reasoning context, explicit obligations, actionable failures and predictable edits, retaining repeated structure where it helps. Only documentation is authorized now, excluding compilation, performance runs and agent experiments. Arbitrary ordinary Can helpers and cases are required, and scenario order, oracles, suite decisions and reporting must be written in Can rather than host harnesses. Native platform mechanisms may remain. Preserve meaningful coverage, enforce laptop resource limits and reclaim owned resources. Backwards compatibility is unnecessary, and no backend has been selected. Today’s draft registers callable case records statically, duplicates textual check names in required_checks and expectation calls, passes constructible reporter/operation contexts, verifies supplied attached assertions through match call/when plus scenario links, then invokes live worker functions. Every concrete fn currently needs assertion rows. when is limited to a single call; arbitrary opaque input omission is unsupported; callable-containing records have equality pitfalls. These observed implementation rules are open to future language design. Early ok detection needs an independently declared inventory, never a list inferred only from executed observations. Typed shared Can descriptors could centralize identity while retaining that independent plan, with added library validation and no guarantee of oracle truth. Validated string metadata has less implementation cost; a new compiler test declaration could instead own discovery and checking without restricting function bodies. Another draft choice is running every offline root before each focused live selection. Existing canlc assert already checks all source while supporting root selectors. A narrower development contract could keep source checking, verify the selected case and a conservative transitive helper/fixture set, use the full set on uncertainty, and publish partial evidence. Complete qualification would still run all mandatory roots, cases and required environments. Evidence must bind suite/compiler/executor identities, and full execution bundles are not retained by default. There is no measured latency or agent repair evidence yet.",
    "Assess this testing proposal for AI coding agents, the primary writers of Can. Hand-authoring comfort is irrelevant; reliable synthesis, local contracts, clear diagnostics and safe source transformations matter, including beneficial explicit repetition. The current task authorizes design work only: do not execute builds, benchmarks or agent trials. The endpoint admits all normal Can functions, places action sequences, expectations, scheduling and verdicts in Can, and removes authored foreign test harnesses. Generic native operations are permitted; retained coverage, bounded resource use and cleanup are requirements. Old language behavior has no compatibility entitlement and the test backend remains undecided. The draft combines a package-level callable registry, string IDs repeated across required_checks and body expectations, injected constructible reporting/operation context, supplied mandatory offline assertions using individual match call/when sites and scenario links, plus separate live case workers. The checker mandates assertion rows for each concrete fn, rejects when on match chain and generally cannot elide opaque inputs; comparing records containing fresh callable values also needs care. Those facts describe the present compiler, not a permanent design boundary. To detect a case returning ok too soon, planned obligations must exist independently of emitted events. One option is typed Can descriptor references with central IDs and an explicit obligation plan, requiring library validation and still unable to establish that expected values are correct. Another keeps simpler string registration with validation; a third adds compiler-recognized test metadata/discovery while retaining ordinary bodies. The draft’s focused runs currently wait for complete-suite offline verification. Source-wide checking and selected assertion execution already coexist in canlc assert. A proposed development alternative would check all source, verify a selected case and conservatively known helper/fixture dependencies, fall back to all roots if that closure cannot be justified, and mark the result partial. Qualification still requires every mandatory root and planned case/environment. Provenance binds source/compiler/executor digests, without persistent full bundles by default. No experiments have demonstrated superior agent correctness or turnaround."
]
QUESTIONS = {
    "case_contract": {
        "instructions": [
            "Which direction should the next authoring revision explore first for agent-safe check identity and case registration, preserving independent obligations?",
            "Select the next registration/check-contract approach to investigate under the stated agent-authorship requirements.",
            "Choose the first design direction for reducing inconsistent case/check edits without losing declared coverage obligations."
        ],
        "criteria": {
            "typed_can_descriptors": [
                "Use shared typed check descriptors in ordinary Can, referenced by both an explicit obligation plan and expectation calls; retain Can registry policy and require validation of identity, ownership and missing evidence.",
                "Define check identities once as typed Can values and reuse them in the declared plan and case body, with Can validation for ownership, registration and evidence completeness.",
                "Investigate canonical Can descriptor values linking required checks to their expectation sites, keeping obligations explicit and using library validation for stale or missing references."
            ],
            "validated_strings": [
                "Retain manually written string IDs in registry and bodies, with runtime/preflight diagnostics for mismatches; prioritize a smaller implementation and tolerate synchronized edits.",
                "Keep the existing string-based records and add precise validation errors, accepting duplicate source spellings in exchange for fewer new library abstractions.",
                "Continue textual check names at each use and registration, relying on Can consistency checks while preserving the current low-complexity metadata format."
            ],
            "compiler_declarations": [
                "Introduce compiler-recognized test registration/check declarations with ordinary Can bodies, moving identity/discovery validation into language tooling and accepting new syntax/checker work.",
                "Explore a language-level declaration that owns case/check identity and discovery without limiting normal functions, with corresponding parser and compiler validation changes.",
                "Add dedicated compiler metadata for tests and checks while retaining full Can functions, taking on new language semantics to validate registration and references."
            ]
        }
    },
    "feedback_scope": {
        "instructions": [
            "Which verification scope should the next design use for a focused development run, with full qualification still separate?",
            "Choose the proposed execution-verification rule for an agent iterating on one case while preserving honest qualification claims.",
            "Which development feedback contract best fits these source facts, limits and coverage requirements?"
        ],
        "criteria": {
            "conservative_selected_closure": [
                "Check the whole suite source, verify selected roots and a justified conservative dependency/fixture closure, fall back to all roots if uncertain, and label evidence partial; keep full qualification mandatory separately.",
                "Permit scoped offline execution after complete source checking, including conservatively established helper/fixture dependencies and a full-root fallback, with explicit partial status and an unchanged full qualification gate.",
                "Design bounded selected-case verification with all-source validation and a conservative affected-root closure; use full verification when closure evidence is missing and never credit the result as full qualification."
            ],
            "always_all_roots": [
                "Keep whole-suite offline verification before each selected live run, avoiding dependency-closure machinery and retaining a simple complete verification receipt despite potentially broader work.",
                "Require every offline assertion before any focused live execution, preserving the draft’s straightforward receipt model and accepting the cost of unrelated roots.",
                "Retain an all-roots prerequisite for development subsets, favoring one easily understood verification boundary over a new dependency selection mechanism."
            ],
            "selected_case_roots_only": [
                "After whole-source checking, execute only the selected case's attached roots before live work; mark the run partial and leave transitive helper verification to the full gate.",
                "Use full source checking plus just the chosen case’s own assertions for development, explicitly omitting helper-root execution and deferring that coverage to qualification.",
                "Verify source globally but run only directly selected assertion roots before the live case, recording the narrower evidence and relying on later full qualification for helpers."
            ]
        }
    }
}

fields = [CONTEXTS] + [q["instructions"] for q in QUESTIONS.values()] + [v for q in QUESTIONS.values() for v in q["criteria"].values()]
assert all(len(values) == 3 and len(set(values)) == 3 for values in fields)
for i in range(3):
    request = {"model": "jev-latest", "state": CONTEXTS[i], "questions": {name: {"type": "choice", "instructions": q["instructions"][i], "criteria": {key: value[i] for key, value in q["criteria"].items()}} for name, q in QUESTIONS.items()}}
    (OUT / f"request-{i+1}.json").write_text(json.dumps(request, indent=2) + "\n")
(OUT / "wording-audit.json").write_text(json.dumps({"status": "reviewed before sending", "method": "Manual semantic comparison plus full-field text inequality for all context, question and option prose across three requests. Stable exact technical identifiers and alternative keys retained. No prior responses supplied.", "equivalent_facts": ["AI authors and optimization criteria", "normal Can/no host harnesses/coverage/resource constraints", "design-only and open backend/no compatibility", "current assertion/fixture/equality rules and draft metadata", "independent planned obligations and descriptor limits", "current whole-source checking and root selection", "conservative dependency closure, full fallback, partial scope and complete qualification", "provenance/no persistent full bundles and lack of measured advantage"], "limits": "Wording changes and agreement cannot establish absence of bias or design correctness."}, indent=2) + "\n")

if "--send" in sys.argv:
    for i in range(1, 4):
        started = datetime.datetime.now(datetime.timezone.utc).isoformat()
        req = urllib.request.Request("https://api.typesafe.ai/v1/systemone", data=(OUT / f"request-{i}.json").read_bytes(), headers={"Content-Type": "application/json", "Authorization": "Bearer " + os.environ["TYPESAFE_API_KEY"]})
        try:
            with urllib.request.urlopen(req, timeout=55) as response:
                body, status = response.read(), response.status
        except urllib.error.URLError as error:
            (OUT / f"transport-error-{i}.json").write_text(json.dumps({"started_utc": started, "kind": type(error).__name__, "reason": str(error.reason)}, indent=2) + "\n")
            raise SystemExit("Consultation transport failed; compact metadata saved.")
        (OUT / f"response-{i}.json").write_bytes(body + b"\n")
        (OUT / f"response-{i}.metadata.json").write_text(json.dumps({"started_utc": started, "http_status": status}, indent=2) + "\n")
        parsed = json.loads(body)
        print(json.dumps({"request": i, "model": parsed.get("model"), "answers": parsed.get("answers"), "usage": parsed.get("usage")}))
