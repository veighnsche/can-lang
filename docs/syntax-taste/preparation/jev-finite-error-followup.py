"""Record three reworded Jev consultations on finite error-set details."""

from pathlib import Path
import datetime
import json
import os
import sys
import urllib.request

OUT = Path(__file__).resolve().parent / "jev-finite-error-followup"
OUT.mkdir(exist_ok=True)

states = [
    {
        "context": "Can targets AI coding agents. Semantic correctness, safe edits, and inspectable contracts take priority; total tokens across a successful task are secondary. There are no external users or compatibility obligations. Generated TypeScript should use native JS/Bun calls plus existing Can failure adapters. This is a proposed trial, not an implemented feature or a final grammar decision.",
        "evidence": "A separate-library Can probe passed 28/28 assertions. An A caller emits only domain::a_failure, and a B caller emits only domain::b_failure. Direct match chains preserve exact bounds but duplicate the flow. A shared helper with a fixed union makes each caller handle the other impossible error. Generic outcome data needs adapters into and out of Can completion. Transform and inspect requirements are already explicit callable arguments. Only singleton caller sets have been tested; no agent trials establish a benefit from a new feature.",
        "candidate": "A bounded prototype would declare `errors E` in a helper header, `callable int () emits [E] action` in its inputs, and `emits [E, audit_down]` outward. E is a finite set of fully specialized nominal errors, possibly empty; duplicates normalize away. The helper may forward an unknown E member but may inspect only a named error after narrowing. Error-set parameters erase in generated code.",
    },
    {
        "context": "Evaluate a narrow library contract for an agent-authored language, without deciding that the grammar should ship. A correct compiled program and reliable future modifications dominate; whole-task token use is measured second. Old syntax needs no support because Can has no outside adopters. Equivalent runtime work should become native JavaScript/Bun calls, preserving Can's finite failure behavior.",
        "evidence": "The reproducible current program has 28 passing checks with two disjoint one-error consumers of a shared integer workflow. Inlined sequencing repeats the algorithm. A single concrete error union in the helper exposes both kinds to each consumer, including an unreachable branch. A parameterized success/failure record transports the distinction only through explicit completion/data conversions. The public helper already reveals its transform and predicate as typed callable parameters. Larger error sets and AI-agent repairs have not been measured.",
        "candidate": "The finite-row experiment uses a written `errors E` parameter, takes an action whose declared callable bound is `emits [E]`, and declares its own possible failures as `emits [E, audit_down]`. E denotes a normalized finite collection of specialized nominal error identities, including `[]`. Forwarding may preserve any member without reading its fields; named narrowing is required to inspect one. Runtime specialization carries no row object.",
    },
    {
        "context": "Choose technical details for a disposable Can experiment serving AI coding agents, not a syntax adoption vote. Prioritize preserved behavior, explicit failure contracts and repairability over brevity; count all tokens used by a successful coding task as a secondary measure. No compatibility migration is required. The TS backend should delegate ordinary operations to JS/Bun and retain necessary completion adapters.",
        "evidence": "In a current isolated library example, 28 assertions succeed. One caller has domain::a_failure and another domain::b_failure. Duplicated match-chain logic retains their precise errors. Reusing one helper with both errors in its fixed signature forces an irrelevant other-error arm at each site. A generic result variant instead requires conversions on both sides of the helper. Its transform and inspect capabilities are visible as ordinary callable inputs. The test covers one error per domain and no controlled agent completion comparison.",
        "candidate": "For the proposed trial, the source helper binds finite nominal error set E explicitly as `errors E`; the callback type contains `emits [E]`, and the helper's written bound is `emits [E, audit_down]`. Set union removes duplicate identities and accepts an empty E. A body can relay an arbitrary E completion but can examine fields only when it names a concrete member. E has no runtime representation after compilation.",
    },
]

questions = [
    {
        "specialization": {
            "instructions": "Which specialization rule gives the trial the most checkable public and call-site error contract? Choose only among the described rules, not overall feature adoption.",
            "criteria": {
                "callback_static_bound": "Infer E from the supplied action callable's declared finite emits set; a broader declared callable bound stays broad. Optional explicit E must agree with the callback bound, and callers see E union audit_down.",
                "mandatory_explicit_row": "Require each use to spell a finite E argument at the call site, then check the callback's declared errors are contained in that argument; the helper bound substitutes that written set union audit_down.",
                "caller_context_row": "Choose E from the enclosing caller's expected outward emits list, then admit the action when its declared errors fit that list; the callback alone need not determine E.",
            },
        },
        "forwarding": {
            "instructions": "How should an authored helper relay the unknown finite E errors while preserving explicit control flow and named-error checking in this limited prototype?",
            "criteria": {
                "grouped_forward_arm": "Permit one explicit `E as failure => failure` completion arm that only relays an unchanged member of E; concrete named arms remain necessary for inspection or conversion.",
                "automatic_call_relay": "When E is included in the function's outward bound, let an ordinary `call action()` propagate its failure without a match arm; success remains an ordinary expression.",
                "specialization_arms": "Require a helper body to enumerate concrete arms after each E specialization; source or generated requirements must cover every member separately rather than one row forwarding arm.",
            },
        },
    },
    {
        "specialization": {
            "instructions": "Pick the error-row instantiation discipline for the proposed helper prototype, emphasizing predictable diagnostics when an action's declaration changes.",
            "criteria": {
                "callback_static_bound": "Derive E from the action argument's written emits contract, retaining any overapproximation in that contract; an author may also provide a checked identical E explicitly. The outward bound adds audit_down.",
                "mandatory_explicit_row": "Every helper invocation names E as a concrete finite list; verify the action's written possible errors are a subset, even if the chosen E includes extra kinds. Add audit_down to the resulting outward bound.",
                "caller_context_row": "Use the surrounding function's allowed failures to infer E first; an action may supply any subset, so the chosen row is shaped by context rather than its own declaration.",
            },
        },
        "forwarding": {
            "instructions": "Select a source-body error forwarding form for a generic E where the helper cannot know E's member declarations before specialization.",
            "criteria": {
                "grouped_forward_arm": "An explicit match completion arm `E as failure => failure` passes through whichever E member occurred, with no field access until a separate named arm narrows it.",
                "automatic_call_relay": "If the helper's declared emits set covers E, action invocation forwards E implicitly and returns only its success value to the surrounding expression.",
                "specialization_arms": "The authored generic body has a separate concrete error arm for each specialized E member, produced or required during instantiation instead of a single generic arm.",
            },
        },
    },
    {
        "specialization": {
            "instructions": "For a trial with two independently supplied error sets, which rule most clearly identifies the resulting set and a failure after a callback edit?",
            "criteria": {
                "callback_static_bound": "Bind E to the callback value's declared finite error list at the invocation; do not narrow by unreachable runtime branches. If E is stated explicitly, require the same normalized set. The helper also lists audit_down.",
                "mandatory_explicit_row": "Have agents state E in type arguments on every helper call, accepting a callback whose written error list is contained in E; all stated E members remain possible to the enclosing caller, along with audit_down.",
                "caller_context_row": "Infer E from the enclosing function's declared permitted errors and check the callback against it; a callback change can be hidden when that outer list is already broad.",
            },
        },
        "forwarding": {
            "instructions": "Which limited helper-body construct should account for all possible E failures without making a general effect system?",
            "criteria": {
                "grouped_forward_arm": "Write `E as failure => failure` once in a completion match; that arm can only preserve the original typed error, while named arms handle any examined or remapped error.",
                "automatic_call_relay": "Let a call expression transparently pass E failures to the helper's caller whenever the written helper bound includes them, bypassing a source match arm.",
                "specialization_arms": "Expand E into member-specific completion arms after the generic helper is specialized, so the generic source needs no explicit grouped forwarding construct.",
            },
        },
    },
]

assert len(states) == len(questions) == 3
for field in states[0]:
    assert len({s[field] for s in states}) == 3
for q in questions[0]:
    assert len({qs[q]["instructions"] for qs in questions}) == 3
    for criterion in questions[0][q]["criteria"]:
        assert len({qs[q]["criteria"][criterion] for qs in questions}) == 3
        assert all(criterion in qs[q]["criteria"] for qs in questions)

for i in range(3):
    payload = {"model": "jev-latest", "state": states[i], "questions": {
        name: {"type": "choice", **body} for name, body in questions[i].items()
    }}
    (OUT / f"request-{i + 1}.json").write_text(json.dumps(payload, indent=2) + "\n")

(OUT / "wording-audit.json").write_text(json.dumps({
    "equivalence": "Manual review: all three requests state the same 28/28 singleton-domain probe, priority order, finite-row candidate, three specialization rules and three forwarding rules. The options keep the same semantic affordances, including broad-bound consequences, exact-set optional override for callback-derived E, and fixed audit_down. No prior Jev result is included in state.",
    "rewrite": "Every explanatory state field, question instruction and criterion description has a distinct full phrasing; code identifiers and the stable option keys are intentionally unchanged.",
    "limit": "The audit checks wording and intended semantic equivalence, not absence of framing effects. Distribution disagreements will be assessed against the contract requirements and probe limitations.",
}, indent=2) + "\n")

if "--send" in sys.argv:
    key = os.environ["TYPESAFE_API_KEY"]
    for i in range(1, 4):
        req = urllib.request.Request("https://api.typesafe.ai/v1/systemone",
            data=(OUT / f"request-{i}.json").read_bytes(),
            headers={"Content-Type": "application/json", "Authorization": "Bearer " + key})
        started = datetime.datetime.now(datetime.timezone.utc).isoformat()
        with urllib.request.urlopen(req, timeout=60) as response:
            body, status = response.read(), response.status
        (OUT / f"response-{i}.json").write_bytes(body + b"\n")
        (OUT / f"response-{i}.metadata.json").write_text(json.dumps({
            "startedAt": started, "status": status, "endpoint": "v1/systemone"
        }, indent=2) + "\n")
        answer = json.loads(body)
        print(json.dumps({"request": i, "answers": answer.get("answers"), "usage": answer.get("usage")}))
