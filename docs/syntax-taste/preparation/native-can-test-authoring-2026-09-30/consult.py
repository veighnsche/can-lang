"""Save and optionally send three independent, reworded design consultations."""
import datetime
import json
import os
from pathlib import Path
import sys
import urllib.error
import urllib.request

OUT = Path(__file__).resolve().parent / "jev"
OUT.mkdir(exist_ok=True)

state = {
    "requirements": [
        "Design, without implementation or measurements, how authors register and run compiler, subprocess, server, browser and native-runtime tests. The user requires arbitrary ordinary Can functions and removal of all authored host test scenarios, expectations and suite policy. Generic native compiler/platform operations remain allowed. The test execution backend is undecided and may differ from production TypeScript. Preserve meaningful coverage, bounded load, immediate resource ownership, cleanup on all outcomes and honest partial evidence; no old syntax, layouts or goldens need compatibility.",
        "The present deliverable is an authoring/execution specification illustrated by compilation rejection, child results, service lifetime, browser actions and native observations; no code migration or performance run is authorized. Tests must admit the full normal Can language. Foreign-language harness decisions, case choreography and comparisons must disappear, although native platform/compiler mechanics may remain. No test backend has been chosen; it need not emit TypeScript. Retained behavioral obligations, explicit budgets, early ownership and reliable reclamation are mandatory, as is truthful reporting of incomplete scope. Legacy spellings and representation snapshots have no compatibility entitlement.",
        "Work out the developer experience using five sample cases: a rejected program, an executed command, a managed server, a driven browser and a host-runtime fact. This is documentation work only, excluding implementation and benchmarking. All regular Can functions must be usable, with Can responsible for test sequences, expected answers and runner policy instead of host harnesses. Native execution infrastructure is permitted. The executor remains an open decision independent of the production TypeScript path. Maintain useful coverage and bounded resources, register ownership at acquisition, clean after failure too, and expose missing qualification. Historic syntax/ABI/output forms are not preservation requirements."
    ],
    "language": [
        "Current Can has typed functions, records, arrays, callable fields, closures with near captures, match chain, relay call, declared domain errors and standard failures. checks::require performs ordinary checks. Attached asserts with named when fixtures test ordinary helpers offline and intentionally prohibit unsupplied live I/O. Existing process::run returns separate byte streams/code/signal with cwd/env/stdin and caps/deadlines; it waits for termination. The language has no general finally and no runtime function-name reflection. A static ordinary function can return records containing same-signature case callables; a worker can reconstruct this registry and select an ID without serializing a closure.",
        "Available language machinery includes records and arrays, typed callable members, captured near inputs, ordinary declared-error functions, match chain and relay call, plus standard-fault propagation. checks::require already checks conditions. Named attached asserts/when rows execute helper logic but deny real effects without supplied fixtures. process::run can launch a bounded one-shot command with explicit input/environment/location and report stdout, stderr, exit code and signal; it does not expose a running child. There is neither general finally nor reflective lookup of a function string. Can code can assemble a uniform callback-record list and rebuild it inside each worker for identifier dispatch, avoiding callable transport.",
        "The existing surface provides normal functions with domain-error bounds, standard faults, records, arrays, callable record slots and lexical near captures; match chain and relay call compose them. Boolean requirements use checks::require. Offline asserts select named when inputs and reject unsupplied platform effects. A terminating command is already expressible through process::run, including bounded output/time, explicit cwd/env/stdin and distinct stdout/stderr/code/signal. It cannot act as a managed service handle. General finally and dynamic function-name lookup are absent. An authored registry function may instead build equal-signature callback records anew in every worker and dispatch by data ID, without encoding live closures."
    ],
    "platform_gaps": [
        "Missing contracts include owned workspaces, managed child start/readiness/events/signals/wait and descriptor passing, external browser automation/interception, and independent raw native/DB facts. A timer in a stuck case cannot enforce its own deadline. Native supervision must reclaim processes and owned storage when workers/controllers die. Case-specific readiness and cleanup policy stay Can-owned. Browser Can APIs operate in a page, not an automation driver; existing host scripts contain page.evaluate functions and interception callbacks. Native qualification contains assertions inside probes for JSON.parse source tokens, negative zero, JSON.rawJSON, thenable boxing and API removal. Relocating these scripts or embedding JavaScript source in Can would retain authored host test logic.",
        "The unresolved substrate comprises temporary-resource ownership, live child lifecycle operations including descriptors, browser-driver actions/events/interception, and native/database observation independent of the tested adapters. Non-yielding case code defeats its own timeout, so external enforcement and abandoned-resource recovery are required for worker or controller loss. Can must still decide scenario readiness and retention/cleanup policy. Current in-page browser operations do not drive another browser. Migrated Playwright callbacks presently evaluate JavaScript and control routes, while host qualification probes compare JSON source lexemes, signed zero, raw JSON, promise assimilation and missing APIs. Wrappers, moved scripts or foreign-code strings would not remove those harnesses.",
        "A complete design still needs parent-owned temporary storage, ongoing child handles with stream/readiness observations, fd setup, signals and waiting, external browser control with routes/events, and raw native or SQL evidence. Deadline enforcement and crash recovery must operate outside a worker that might hang; foreign and active resources must remain untouched. Can retains service-readiness choices and retention decisions. The browser catalogue serves page programs rather than external automation. Existing tests use JavaScript page evaluation and interception, and native test probes themselves assert token-source fidelity, -0, raw JSON, nonthenable boxes and API absence. Keeping that foreign scenario code behind a Can facade fails the requested boundary."
    ],
    "diagnostics_and_reports": [
        "A public semantic canlc check --json command is not present. The internal CheckSnapshot API already yields code, severity, message and source coordinates; parse is grammar-only. Expected rejection needs a paired valid fixture and checked phase/diagnostic/location, not merely nonzero exit. A proposed worker emits bounded structured events on a private channel separate from candidate stdout, identified by run/case/variant/attempt. Can reduces these into outcomes and coverage; native supervision only reports execution/cleanup facts. A caught ordinary check failure could otherwise allow final ok to erase an earlier failed expectation. Pure comparison helpers can be tested using attached asserts without committing live expectation events. Unexpected faults, malformed/missing reports and failed cleanup cannot supply a clean pass.",
        "Semantic diagnostics exist internally through CheckSnapshot with source locations, severity, code and wording, but there is no exposed canlc check --json; the parse command only establishes grammatical acceptance. To prove intended rejection, run an equivalent good control and inspect the bad program's particular stage, diagnostic and span rather than any failure status. Consider a bounded private worker-event stream carrying run, case, variant and attempt identity, isolated from child output. Can code would aggregate verdicts while the supervisor reports mechanical exit and cleanup facts. Simply returning ok after catching an expectation error risks hiding that mismatch; deterministic comparator tests can remain attached asserts with no live event commitment. Missing/bad evidence, unexpected faults or cleanup errors must prevent success.",
        "CheckSnapshot provides structured internal diagnostics, including codes/severity/text and positions. CLI users lack a semantic canlc check --json surface, and parse is only syntactic. A negative compiler example therefore needs a matching accepted control plus evidence for the intended diagnostic, stage and source span; a crash is not sufficient rejection. The candidate reporting shape uses identity-bound, size-limited events over a dedicated worker channel that candidate output cannot impersonate. Suite aggregation and verdicts belong in Can; termination and reclamation observations belong in native supervision. With uncaught-only handling, catching a failed expectation and later returning ok can mask it. Attached asserts can exercise pure comparator results separately. Absent or malformed records, unforeseen faults and unresolved cleanup cannot count as passed."
    ]
}

questions = {
    "registration": {
        "instructions": [
            "Which initial authoring/registration surface best meets these requirements without assuming an execution backend?",
            "Select the first test-definition and discovery model that fits the stated language and migration constraints while leaving execution technology open.",
            "Choose the registration approach to specify for the five examples, holding the unresolved backend choice independent."
        ],
        "criteria": {
            "can_library": [
                "Use ordinary Can case functions and an explicit Can registry of typed callback records and metadata; reconstruct it per worker. Add library/platform contracts, not test grammar or reflection.",
                "Define cases as existing Can functions, list their equal-signature callables and data in authored Can, and rebuild the list in workers. Supply the missing APIs without a test keyword or dynamic lookup.",
                "Adopt a Can library with an authored callback-record catalogue; worker-local construction resolves stable IDs to regular functions. Extend mechanics as needed while leaving the language grammar unchanged."
            ],
            "new_test_syntax": [
                "Introduce compiler-recognized test declarations and discovery while allowing fully ordinary Can bodies. Specify the new grammar, checker rules and registration emission before library examples.",
                "Create a dedicated test declaration form whose body accepts the complete language; have the compiler discover/register it, with corresponding syntax and checking work up front.",
                "Add test-specific source declarations with unrestricted Can implementations and compiler-generated case indexing, establishing the extra frontend rules first."
            ],
            "manifest_entrypoints": [
                "Register separate Can case entrypoints through a static external manifest of IDs, paths and metadata; actions and expectations stay Can-authored. Specify manifest validation and separate case builds.",
                "Use an external data inventory to name individual Can case programs and their requirements, while leaving all scenario logic in those programs and defining validation/build behavior.",
                "Choose a manifest listing IDs and Can executable entrypoints, with metadata only rather than scenario scripts; account for manifest checking and per-entry compilation."
            ]
        }
    },
    "observation": {
        "instructions": [
            "Which boundary should the examples adopt for external browser and otherwise unrepresentable native observations?",
            "Choose the proposed observation interface for driver interactions and host values ordinary Can cannot safely construct.",
            "How should browser/native evidence cross into Can without retaining foreign scenarios or verdicts?"
        ],
        "criteria": {
            "typed_mechanics": [
                "Use typed generic actions, raw observations and scoped fault operations, with Can selecting sequence and expected values. Browser-side programs are Can when needed. Mark unsupported hostile-value/driver operations as explicit gaps, without claiming the initial set covers all cases.",
                "Expose reviewed platform operations returning facts and controlled fault handles, leaving choreography and comparisons to Can; author any required page probe in Can too. Record unexpressible cases until more generic mechanics exist, rather than declaring comprehensive support.",
                "Specify data-returning native/browser bindings plus isolated perturbations, composed by normal Can test functions. Use compiled Can for page-side logic and leave missing low-level operations unresolved until designed; do not infer universal coverage from five examples."
            ],
            "foreign_source_eval": [
                "Allow Can to send JavaScript source strings for page evaluation and native probes; keep top-level expected comparisons in Can, accepting that lower-level scenario callbacks remain foreign-authored.",
                "Provide a raw JavaScript evaluation escape hatch for browser/native actions and let Can compare outputs, while explicitly retaining foreign code for the evaluated sequences.",
                "Embed arbitrary host-language programs in Can calls to construct observations, with final assertions in Can despite the embedded action logic remaining JavaScript."
            ],
            "special_case_adapters": [
                "Implement native functions per scenario family that execute a fixed browser/native test sequence and return a summary, leaving Can to register and assess the summary.",
                "Place each family-specific scenario procedure in a native adapter and return an observation report for Can's final verdict and scheduling.",
                "Use native helper procedures containing the predetermined browser or host-probe choreography; Can owns their invocation and report comparison."
            ]
        }
    },
    "expectation_failure": {
        "instructions": [
            "Which expectation-failure rule best prevents an apparently successful live case from hiding recorded mismatches?",
            "Select how the case outcome should account for an expectation error caught by ordinary Can code.",
            "What rule should connect failed expectations, subsequent recovery and the final live-test verdict?"
        ],
        "criteria": {
            "recorded_mismatch": [
                "Can expectation helpers append a mismatch event before emitting their error. Can report reduction keeps the case failed even if caught; pure comparator results remain separately testable with attached asserts. Native transport only retains bounded events.",
                "Have ordinary helpers record failed comparisons and then return their declared failure. The Can aggregator treats recorded mismatches as lasting failures, while offline comparator tests inspect values without event commitment; the native channel supplies storage/transport only.",
                "Commit each expectation defect as a Can-authored event, then emit the ordinary error. Later ok cannot erase the event in Can's outcome rules. Test comparator logic independently through asserts, with native code limited to bounded event mechanics."
            ],
            "uncaught_only": [
                "Classify only the final uncaught completion and cleanup; catching an expectation error permits success. Explain this convention and rely on authors not to suppress unintended mismatches.",
                "Let an ok function completion pass once cleanup succeeds even after a handled expectation failure; document author responsibility for not hiding accidental failures.",
                "Use terminal completion alone for behavior status, allowing recovered comparison errors to disappear from the verdict, and enforce correct recovery through review/convention."
            ]
        }
    },
    "compiler_rejection": {
        "instructions": [
            "Which diagnostic interface should the proposed rejection example target, without treating it as already implemented?",
            "Choose the planned public observation surface for the compiler-negative case, accounting for current internal diagnostic support.",
            "What compiler boundary should this authoring design specify for distinguishing intentional rejection from other failures?"
        ],
        "criteria": {
            "structured_check": [
                "Propose a general semantic check command emitting structured diagnostic observations from the existing frontend; Can owns expectations and paired controls. Define exit/protocol/span semantics and list the absent public API as work.",
                "Expose the compiler's existing semantic analysis through a new structured check CLI, with stage/code/location results consumed by Can. Specify transport statuses and coordinate conventions while clearly marking implementation pending.",
                "Design a general-purpose check/report command backed by current diagnostic machinery, keeping expected-error matching in the Can library. Record the required public contract, including status and positions, as a gap."
            ],
            "stderr_contract": [
                "Use an existing failing build/assert invocation and compare exit plus stderr in Can, defining wording parsing and phase identification as the initial contract; do not add a public check command yet.",
                "Initially drive current compiler commands and interpret their failure text and status in Can, specifying the diagnostic parser and stage checks instead of extending the compiler CLI.",
                "Base rejection cases on present build/assert stderr and exit behavior with a Can parser for wording and stage, postponing a structured semantic command."
            ]
        }
    }
}

for variants in state.values():
    assert len(set(variants)) == 3
for q in questions.values():
    assert len(set(q["instructions"])) == 3
    for variants in q["criteria"].values():
        assert len(set(variants)) == 3
requests = []
for i in range(3):
    payload = {"model": "jev-latest", "state": {k: v[i] for k, v in state.items()},
               "questions": {k: {"type": "choice", "instructions": q["instructions"][i],
                                 "criteria": {key: v[i] for key, v in q["criteria"].items()}}
                             for k, q in questions.items()}}
    requests.append(payload)
    (OUT / f"request-{i+1}.json").write_text(json.dumps(payload, indent=2) + "\n")

(OUT / "wording-audit.json").write_text(json.dumps({
    "status": "checked before sending",
    "method": "Manual semantic review of all four state fields, four instructions and ten option descriptions across three requests, plus complete-field text inequality checks. Each explanatory field is fully rephrased; exact Can/API identifiers and option keys are intentionally stable. No earlier response feeds a later request.",
    "equivalence": ["same five examples and full-Can/no-host-harness requirements", "same language support and explicit missing operations", "same backend uncertainty and budget/cleanup constraints", "same compiler API facts, paired controls and evidence failure conditions", "same three registration alternatives, three observation alternatives, two outcome alternatives and two compiler alternatives"],
    "limitations": "Equal facts and wording variation reduce one framing risk; they do not prove bias removal, independent reasoning or design correctness. Agreement is advisory."
}, indent=2) + "\n")

if "--send" in sys.argv:
    key = os.environ["TYPESAFE_API_KEY"]
    for i in range(1, 4):
        request = urllib.request.Request("https://api.typesafe.ai/v1/systemone",
            data=(OUT / f"request-{i}.json").read_bytes(),
            headers={"Content-Type": "application/json", "Authorization": "Bearer " + key})
        started = datetime.datetime.now(datetime.timezone.utc).isoformat()
        try:
            with urllib.request.urlopen(request, timeout=55) as response:
                body, status = response.read(), response.status
        except urllib.error.HTTPError as error:
            body, status = error.read(), error.code
        except urllib.error.URLError as error:
            (OUT / f"transport-error-{i}.json").write_text(json.dumps({"started_utc": started, "error": str(error.reason), "response_received": False}, indent=2) + "\n")
            raise SystemExit("transport failed; no classifier response received")
        (OUT / f"response-{i}.json").write_bytes(body + b"\n")
        (OUT / f"response-{i}.metadata.json").write_text(json.dumps({"started_utc": started, "http_status": status, "endpoint": "https://api.typesafe.ai/v1/systemone"}, indent=2) + "\n")
        parsed = json.loads(body)
        print(json.dumps({"request": i, "status": status, "model": parsed.get("model"), "answers": parsed.get("answers"), "usage": parsed.get("usage")}), flush=True)
        if status != 200:
            raise SystemExit("consultation failed; saved response for review")
