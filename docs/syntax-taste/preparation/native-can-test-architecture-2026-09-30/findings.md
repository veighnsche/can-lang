# Architecture decision evidence

Status: documentation and source review only. No implementation, build, live
test, interpreter experiment, benchmark or bootstrap qualification was run.

The [architecture decision](../../native-can-test-architecture-2026-09-30.md)
adopts existing Can-to-TypeScript/Bun execution, a Can controller and fresh
Can worker OS processes under generic native supervision. A separately
qualified reference toolchain builds the authoritative suite; the candidate
compiler/runtime produces subjects.

## Independent source reviews

One reviewer checked the current driver, emitter and runtime; another checked
bootstrap, candidate trust and interpreter reconsideration. Their source-backed
findings informed the decision:

- The emitter and Go supervisor already support emitted entries and separate
  Bun processes. The existing protocol is for assertion roots, not live cases.
- `Assert` checks the complete source graph before selecting root execution.
  `Build`/`Run` verify and publish; they are not a nonpublishing focused suite path.
- Runtime manifests and generation leases support identity checks. Integrity
  does not establish semantic trust, and no qualified seed was identified here.
- Worker termination kills the direct process. Detached subprocesses and remote
  resources need outside-worker cleanup authority. Worker-local cleanup does
  not survive worker `SIGKILL`.
- Buffered output truncated after collection is not a hard memory limit.
  Temporary launch directories and Bun flags do not establish an OS sandbox.
  Cleanup errors/recovery require explicit new handling.
- New suite syntax/platform APIs must trigger reviewed reference refresh or
  blocked coverage. Candidate self-testing cannot silently promote the judge.
  Transitional independent host checks may establish bootstrap evidence before
  those harnesses are retired; they are not the final endpoint.
- A reviewed acceptance/digest record is required; no new cryptographic signing
  system was found or prescribed. Common compiler/runtime bugs remain possible.

Source anchors and the corresponding implementation gaps are in the decision
document. The focused final review is recorded in [review.md](review.md).
That review added explicit all-root bootstrap/planner ordering and separate
reference versus edited-suite trust gates. The consultations below advise the
three architecture choices; they do not constitute validation of those gates.

## Jev consultations

The repository requires three fresh consultations for difficult decisions.
All explanatory state, instructions and option descriptions were rewritten for
each request. Exact Can/API identifiers and alternative keys remained stable.
Semantic equivalence and full-field text differences were checked before sending;
no earlier answer was passed to a later request.

| Question | Preferred alternative | Selected probability in requests 1 / 2 / 3 |
| --- | --- | --- |
| Initial backend | Existing full Can execution | 0.98 / 0.99 / 1.00 |
| Case boundary | Fresh Bun OS process | 0.92 / 0.99 / 0.90 |
| Authoritative suite compiler | Qualified reference toolchain | 1.00 / 1.00 / 1.00 |

These are returned option probabilities, not measured correctness rates.
Requests used `jev-latest`; every response identifies `jev-1.13.0`.

- [Request 1](request-1.json), [response 1](response-1.json), [transport metadata](response-1.metadata.json).
- [Request 2](request-2.json), [response 2](response-2.json), [transport metadata](response-2.metadata.json).
- [Request 3](request-3.json), [response 3](response-3.json), [transport metadata](response-3.metadata.json).
- [Pre-send wording audit](wording-audit.json), [reproduction script](consult.py).

No winning alternative differed. The first response assigned 0.07 to threads;
the third assigned 0.09 to a reusable process pool. Those options could reduce
launch work, but require shared-state/reset and failure-boundary evidence that
does not exist here. Fresh processes are a deliberate initial isolation choice,
not an assertion that they are the fastest. There is likewise no measurement
showing that an interpreter improves startup or disk use.

A frozen suite artifact provides stable provenance but slows adoption of new
test-source changes. Compiling source with a qualified reference preserves
separate authority while allowing changes R can understand. Unsupported syntax
or APIs remain a refresh gate; the design does not hide that bootstrap cost.
Candidate-built suite controls can detect some self-miscompilation, but passing
them cannot rule it out; candidate self-build is therefore supplementary evidence.

An interpreter remains a credible full-language alternative when a broader
language need, meaningful semantic diversity or attributable execution cost
justifies it. The decision records entry and parity conditions rather than
using Jev agreement to foreclose the option.

The [TypeSafe API](https://docs.typesafe.ai/api) and
[Choice contract](https://docs.typesafe.ai/primitives/choice) were checked during
this ongoing design session. Credentials were read from the environment and
were not saved. Wording variation and classifier agreement cannot prove bias
removal, semantic correctness, containment or bootstrap qualification.
