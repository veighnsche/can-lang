"""Three fresh Jev consultations after the DI-06 generated-output spike."""
from pathlib import Path
import datetime
import json
import os
import sys
import urllib.request

OUT = Path(__file__).resolve().parent
objective = [
    "Select the smallest honest cross-package fixture design for Can, a language authored by AI coding agents. Correct integrated behavior and safe refactoring outrank complete successful-task tokens, which are secondary and unmeasured here. Human convenience and backwards compatibility are not requirements; generated TS should use native JS/Bun operations with only Can assertion adapters. Jev gives advice, not implementation proof.",
    "Judge a planned Can testing contract for agent-written packages. Preserve helper-body integration, explicit scenario ownership, predictable caller renames and truthful evidence labels before optimizing prompts/code/retry tokens. There are no outside users demanding old syntax. Native JavaScript/Bun lowering remains preferred, with minimal fixture machinery only for Can's guarantees. The classifier cannot establish correctness by voting.",
    "Can targets AI agents rather than human readers. Pick an implementation-plan direction for cross-package controlled tests, prioritizing correct real helper execution and robust edits; measure whole-task tokens later. Old source compatibility is unnecessary. Emitted TS delegates ordinary operations to JS/Bun and adds only fixture-context adapters. This is design advice on supplied evidence, not a test result."
]
facts = [
    "Current helper stamp(int) privately calls sample_time() -> clock::wall_millis(), converts time to text and appends '!'. An ambient helper row keyed by a caller display label passes customer but fails when only that label becomes renamed, with no compiler diagnostic. A whole-stamp stub survives rename but keeps passing after the helper suffix changes, so it bypasses the body. Current Can public callable injection exercises real stamp and fake clock, but changes all three app calls and moves the clock dependency into app. A separate exported stamp_with_clock(int, callable now) leaves two normal calls unchanged and changes one assertion call; it executes the real body yet publishes the private clock dependency as test API. All current-Can forms were built and 58 roots exercised, with 57 expected passes and the ambient rename failure.",
    "In the executed hard case, public stamp(int) reads private clock::wall_millis() through sample_time(), then produces text plus '!'. The helper-owned lexical fixture silently stops activating after a caller root name changes from customer to renamed. A whole-helper supplied result cannot detect a suffix mutation. Injecting a now callback directly into stamp preserves real integration but edits three app call sites and makes the app own the clock adapter. Keeping stamp and adding exported stamp_with_clock changes only the test call and detects separate fake-time and helper-suffix changes, at the cost of exposing an internal clock seam. The pinned current-Can pilot checked eight variants and 58 roots; only the intended ambient-rename root failed.",
    "The library's unchanged functional API is stamp(int), with a private sample_time() clock read and a trailing '!'. A caller assertion label edit breaks an ambient helper fixture without static rejection; stubbing stamp entirely misses helper-body regressions. An ordinary callable parameter is executable and rename-stable but forces each of three consumer calls to supply a clock function. An additional exported stamp_with_clock preserves the normal signature/calls and runs the helper body for one test caller, while making private timing dependency public. Current-Can evidence includes eight valid projects and 58 assertion roots, with one deliberate ambient failure and both injection controls catching fake-time and suffix mutations."
]
spike = [
    "A disposable F2 sidecar names helper-owned fixed_time for the exact app::read_customer#0 -> helper::stamp(7) invocation and supplies ok 1000 at private sample_time -> clock::wall_millis(). It validates six mutated link fields before execution, patches generated app code and a copied runtime only in /private/tmp, and passes seven roots with renamed caller and two linked sequential calls using existing FIFO; a changed helper suffix makes the caller fail. Reports distinguish supplied-completion at the clock from real-can helper logic. This is not Can grammar/checker support: validation is hard-coded, and recursive/concurrent selection, arbitrary visibility/type changes and agent costs remain unproved.",
    "The post-comparison prototype uses a separate link manifest plus generated-output and temporary-runtime rewriting, not production compiler changes. It attaches a helper-exported fixed_time scenario to a precise caller stamp(7) site and replaces only the helper's private clock completion with ok 1000. A renamed caller passes, the helper's changed suffix fails, and two sequential links pass through existing FIFO; six malformed scenario/callee/site/target/type/argument fields are rejected by a prototype validator. Its seven-root results support a runtime path with real helper-body coverage and truthful supplied/real labels, but do not prove general source diagnostics, concurrency, recursion or task-token economy.",
    "F2 has a narrow generated-output simulation: an explicit sidecar link selects the helper-owned fixed_time row only while the named app call invokes stamp(7). The copied runtime supplies clock ok 1000 inside the real helper, leaves normal stamp calls untouched, and carries supplied-completion plus real-can evidence. Seven roots pass after caller rename; mutating the helper suffix fails; two sequential selected calls pass with two FIFO occurrences; six changed manifest fields reject before execution. The spike patches temporary emitted code and validates known fields by hand, so source syntax, arbitrary checked links, nested/concurrent/recursive isolation and comparative agent behavior are still future obligations."
]
instructions = [
    "Which final DI-06 planning contract best balances the demonstrated helper-body requirement, private dependency boundary and new mechanism cost after this spike?",
    "Select the cross-package test design engineering should specify now, using the current-Can controls and the limited F2 generated-output result.",
    "What should the implementation plan adopt for deliberate helper-owned behavior in a caller test, given the executed alternatives and remaining prototype limits?"
]
options = {
    "exported_test_entry": [
        "Retain same-owner lexical fixtures only and use an ordinary exported stamp_with_clock test entry for cross-package integrated coverage. It runs real helper logic and survives the caller rename while changing one test call; accept that the private dependency becomes public API. Whole-helper stubs remain labeled as bypassing the body. Do not add scenario grammar/runtime linking.",
        "Fix foreign-label activation, then make callers use a separately exported callable-taking helper entry where they must control a private effect. The normal stamp(int) remains unchanged, one assertion call changes, and the helper body stays under test. Treat the clock parameter as an intentional public test contract; omit any language-level exported fixture scenario.",
        "Choose current-language stamped test entry plus owner-local lexical rows. An exported stamp_with_clock exposes the clock seam to testing clients but permits actual body execution and mutation detection without a new scenario selector. Document API cost and distinct evidence for whole-function stubs; do not ship F2."
    ],
    "exported_scenario": [
        "Add a helper-owned named exported scenario that can bind a checked internal fixture plan to one exact qualified caller invocation without changing stamp(int) or ordinary call sites. The compiler checks owner/callee/site/target/argument/completion and rejects stale, missing, duplicate or unused links before runtime; selection is invocation-scoped with FIFO and honest supplied/real evidence. Implement recursive/concurrent isolation and compare against stamp_with_clock before claiming agent benefit.",
        "Specify a first-class exported helper scenario referenced at a caller's precise call site. Its selected private clock row executes inside real stamp logic while ordinary API stays untouched. Static checking must validate visibility, signatures, exact site, type, occurrence and link use, never falling through to a real clock if stale. Extend the demonstrated sequential FIFO to nested, recursive and parallel calls; retain current test-entry baseline for agent comparison.",
        "Adopt F2 named helper-owned scenarios as planned language support: a caller explicitly attaches the checked scenario to one stamp(7) invocation, with no display-label coupling or public clock parameter. Report both supplied internal completion and real helper execution; diagnose every broken link and preserve root/invocation queue isolation including concurrency. The sidecar spike shows only feasibility; production compiler and runtime checks plus agent comparison remain required."
    ],
    "public_parameter": [
        "Change stamp itself to accept a now callable and require ordinary consumers to pass the real adapter, with the test passing a fake. This uses current callable semantics and tests the body without new fixtures, but changes three app calls and moves private time dependency into public functional API. Repair ambient rows to same-owner selection separately.",
        "Use direct dependency injection in the public stamp signature for all callers. It makes the clock operation explicit and gives the test real helper-body coverage with no new scenario mechanism, while forcing both production callers and the assertion to supply a clock adapter. Stop ambient foreign-root activation independently.",
        "Retain only current language by adding a clock callable parameter to stamp(int), altering each of three consumer invocations. Fake and real callbacks both execute through the helper, but the library's internal clock boundary becomes an ordinary public requirement. Fix lexical fixture ownership and add no test entry or exported scenario."
    ]
}
for triple in (objective, facts, spike, instructions, *options.values()):
    assert len(triple) == 3 and len(set(triple)) == 3
for i in range(3):
    payload = {"model": "jev-latest", "state": {"objective": objective[i], "current_can": facts[i], "scenario_spike": spike[i]}, "questions": {"fixture_contract": {"type": "choice", "instructions": instructions[i], "criteria": {k: v[i] for k, v in options.items()}}}}
    (OUT / f"request-{i+1}.json").write_text(json.dumps(payload, indent=2, ensure_ascii=False) + "\n")
(OUT / "wording-audit.json").write_text(json.dumps({"manual_review": "Before sending, compared all complete payloads: same AI-agent priorities, no compatibility, native lowering, exact hard-case behavior and counts, equal current-Can alternatives, F2 post-spike evidence and limitations, unchanged three options. Every explanatory state, instruction and criterion has separately rewritten wording. No prior Jev answer or preferred conclusion was supplied.", "mechanical_check": "All explanatory triples have three pairwise distinct full strings.", "limit": "Equivalence and absence of framing bias are not mechanically proved."}, indent=2) + "\n")
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
