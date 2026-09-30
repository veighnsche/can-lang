"""New consultations after discovering mandatory assertions and shutdown ordering."""
import datetime
import json
import os
from pathlib import Path
import sys
import urllib.request

OUT = Path(__file__).resolve().parent / "jev" / "mandatory-followup"
OUT.mkdir(exist_ok=True)
contexts = [
    "Design-only Can-native test authoring must allow all normal functions, eliminate host-authored scenarios/oracles/policy, preserve coverage and bounded cleanup, and leave the executor backend open. New source evidence: every concrete fn requires attached asserts; omitting them is invalid. Live I/O is forbidden in offline assertions unless supplied. when tables attach only to match call, not match chain, and scenario links explicitly activate exported fixtures across packages. Generic opaque assertion arguments cannot be omitted like special request-scope inputs. A package-level array of callable records is legal and avoids an unnecessary registry function. Cases can receive a constructible data context with a worker-local reporting callable: live reporting holds a scoped channel, while the offline recorder performs no I/O and the actual comparisons/errors still execute. Per-case match-call observation helpers can supply only boundary results; linked scenario rows/templates avoid replacing the complete case with canned success. All real action steps and expected values remain Can functions. A substantive mandatory root should execute the scenario under supplied boundaries, with separate mutant rows where useful; it cannot count as live evidence. Separately, server SIGTERM calls expireScopes and closeServerSessions before awaiting closeResource. The closer invokes Bun stop(false) only after active resource leases drain, so waiting for listener refusal while a held handler owns a lease can deadlock. An external WebSocket session receives native close code 1001/reason shutdown before lease drainage. A Can case can observe that close while its held request remains pending, then release a peer and require response plus normal child exit. This witnesses the signal path, not listener closure. No test/build has verified the proposed full example.",
    "The task specifies an authoring contract, not implementation: use unrestricted Can functions for test decisions and choreography, remove foreign harness policy, retain useful coverage with resource bounds/recovery, and do not choose a backend yet. The checker now inspected mandates at least one assertion row on every concrete function. Assertion execution denies unsupplied effects. Lexical when applies to an individual match call; chain matches reject it. Exported scenario markers must be linked explicitly for cross-package rows, and arbitrary opaque inputs have no automatic omission rule. Static top-level typed callback-record arrays already work, making a function solely to return the registry unnecessary. A data context can inject an ordinary reporting callable, instantiated locally per worker; offline reporting uses a pure sink and still runs comparisons and their errors. Case-owned observation wrappers may attach supplied completions at individual boundaries and reuse scenario/template rows. They must not substitute one fake successful completion for an entire case. The mandatory assertion should meaningfully traverse the Can scenario with supplied inputs and useful bad-result controls, with evidence labelled offline rather than live. For the service example, SIGTERM first expires request scopes and closes server WebSocket sessions, then waits for resource close. The underlying stop(false) is delayed until outstanding leases finish, so a blocked handler prevents a pre-release connection-refusal witness. A separate external WebSocket can instead see close(1001,shutdown) while the HTTP request is still outstanding; releasing its peer then allows a response and clean process exit. That observation establishes signal-handler progress, not stopped TCP admission. These proposed combined sequences remain unexecuted.",
    "Prescribe how users write tests with ordinary full Can, keeping scenario order, expected answers and suite rules out of host code. This documentation round requires bounded ownership/cleanup and preserved obligations, excludes execution work, and leaves backend selection undecided. Source review exposed compulsory asserts for each concrete fn: the earlier assertion-free snippets were invalid. Offline roots need supplied effect boundaries; a when table belongs to match call and cannot decorate match chain. Package-crossing fixture use is explicit through scenario links, and ordinary opaque parameters cannot rely on request-scope elision. Current top-level values support an array of typed callable records, so registration need not add another function. Normal context data plus an injected reporting callable lets worker-local live channels coexist with an offline no-I/O recorder; Can expectations and their failure completions execute in both modes. Small case-specific Can observation helpers can carry linked templates/when rows for native calls, while substantive attached assertions run the actual case path and test altered observations. Mocking the whole case or crediting supplied traces as production evidence is excluded. Another source constraint concerns shutdown: the SIGTERM handler expires scopes and initiates WebSocket closure before closeResource; only after leases end does the registered closer call stop(false). A held HTTP handler can therefore prevent listener refusal until released. The external witness already available is a WebSocket close with 1001 and shutdown while the held HTTP operation is pending, followed by peer release, expected reply and clean exit. It demonstrates entry to signal-driven shutdown, not removal of the listener. The assembled fixture has not been tested."
]
q = {
    "mandatory_assertions": {
        "instructions": [
            "Which authoring resolution should this design adopt for compulsory offline assertions without weakening the ordinary-function requirement?",
            "Choose how to repair the case examples in light of mandatory assertion rows and effect-fixture restrictions.",
            "Select the approach that reconciles live test functions with the checker's existing assertion requirement."
        ],
        "criteria": {
            "explicit_boundary_fixtures": [
                "Keep current function/assertion rules. Use a static registry, constructible injected context and meaningful attached scenario assertions, with Can match-call helpers supplying individual observation/report boundaries and live execution separately qualified.",
                "Retain the language contract, register ordinary callbacks as a top-level value, and run their required roots with explicit context plus boundary-level fixtures in Can helpers. Distinguish those supplied checks from the actual live cases.",
                "Repair the examples using existing assertion semantics, a value registry and injected offline reporting; attach real scripted scenario roots whose operation results come from named Can fixtures, then run the same bodies live for qualification."
            ],
            "assertion_exemption": [
                "Add a compiler exception so registered test functions need no attached assertions; specify the new registration-sensitive checking rule and its effect on verification.",
                "Permit listed case functions to omit assertions through a new checker exemption, with an explicit language/verification contract for that special status.",
                "Change function validation for test entries so mandatory roots no longer apply there, documenting how the compiler recognizes exempt functions and qualifies them."
            ],
            "live_assertion_mode": [
                "Create an explicit assertion mode allowing real platform effects and use those roots as live cases; specify changed discovery, build and fixture semantics.",
                "Introduce live-effect assertion execution for test roots and define how it interacts with compilation, default verification and supplied fixtures.",
                "Expand assertion machinery with a live mode that executes services/processes directly, establishing new build-selection and effect rules."
            ]
        }
    },
    "shutdown_witness": {
        "instructions": [
            "Which causal witness should the representative server example specify without quietly changing production shutdown behavior?",
            "Choose a concrete ordering witness for the proposed SIGTERM/in-flight example under the observed implementation.",
            "How should this design demonstrate that shutdown was entered before it releases the held request?"
        ],
        "criteria": {
            "external_websocket": [
                "Use an explicitly opened external WebSocket and require close(1001,shutdown) while the held HTTP request is still pending, then release and check reply/exit. Record the extra generic client API and the limited claim; validate the full sequence later.",
                "Add the generic WebSocket client leg: establish it before signalling, observe the documented shutdown close before peer release, confirm the HTTP operation remains outstanding, then require successful reply and exit. Do not claim admission closure or completed validation.",
                "Specify an independent socket-close observation tied to the existing signal handler, with the held request still unfinished, and only then unblock it. Track the required client mechanics and later validation, limiting evidence to shutdown-path entry."
            ],
            "new_runtime_event": [
                "Add a generic runtime lifecycle event after signal processing and observe it before releasing the peer; document its instrumentation and weaker self-reported independence.",
                "Introduce a candidate-runtime event for entering shutdown, wait for that event and then release the request; specify new instrumentation and its dependence on the subject's report.",
                "Instrument the runtime with a shutdown-requested event and use it as the ordering barrier, acknowledging that the evidence comes from the implementation being tested."
            ],
            "change_stop_order": [
                "Change runtime shutdown to stop listening before lease drain, then observe connection refusal while the handler is held; treat this as a production semantic change requiring separate qualification.",
                "Move stop(false) ahead of waiting for leases and use independent fresh-connection refusal; explicitly include the added runtime behavior change and its validation burden.",
                "Redesign admission shutdown ordering to make early TCP refusal observable, and qualify that production change independently before using it as the fixture barrier."
            ]
        }
    }
}
for values in [contexts] + [x["instructions"] for x in q.values()] + [v for x in q.values() for v in x["criteria"].values()]:
    assert len(values) == 3 and len(set(values)) == 3
for i in range(3):
    request = {"model": "jev-latest", "state": contexts[i], "questions": {name: {"type": "choice", "instructions": x["instructions"][i], "criteria": {key: v[i] for key, v in x["criteria"].items()}} for name, x in q.items()}}
    (OUT / f"request-{i+1}.json").write_text(json.dumps(request, indent=2) + "\n")
(OUT / "wording-audit.json").write_text(json.dumps({"status": "reviewed before send", "equivalence": "All three state passages preserve every requirement, newly discovered checker constraint, injected-context/boundary-fixture alternative, shutdown ordering fact and limit on evidence. Both questions and all six option descriptions are fully reworded. Exact syntax/API/status identifiers stay unchanged. No prior answer is supplied.", "limits": "Manual semantic review and complete-field text differences do not prove unbiased advice or implementation feasibility."}, indent=2) + "\n")
if "--send" in sys.argv:
    for i in range(1, 4):
        req = urllib.request.Request("https://api.typesafe.ai/v1/systemone", data=(OUT / f"request-{i}.json").read_bytes(), headers={"Content-Type": "application/json", "Authorization": "Bearer " + os.environ["TYPESAFE_API_KEY"]})
        started = datetime.datetime.now(datetime.timezone.utc).isoformat()
        with urllib.request.urlopen(req, timeout=55) as response:
            body, status = response.read(), response.status
        (OUT / f"response-{i}.json").write_bytes(body + b"\n")
        (OUT / f"response-{i}.metadata.json").write_text(json.dumps({"started_utc": started, "http_status": status}, indent=2) + "\n")
        parsed = json.loads(body)
        print(json.dumps({"request": i, "model": parsed.get("model"), "answers": parsed.get("answers"), "usage": parsed.get("usage")}), flush=True)
