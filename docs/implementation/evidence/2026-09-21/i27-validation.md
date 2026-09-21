# I27 validation — mixed questions, named arms and generated records

The checked question plan now covers Noul, Choice and Score. Preparation evaluates
ordinary inputs and descriptor expressions once, preserving source order, then
returns a private descriptor and a typed deferred handler. Static record spreads
retain the evaluated arm values across transport. The judge prepares every
registration, submits one request, validates every response (including unused
registrations), and invokes handlers in registration order. Results become visible
to the continuation only after all required handlers have succeeded.

The native adapter validates 1–256 registrations, 2–255 Choice keys, 2–10 Score
levels, scalar descriptions and keys, UTF-8 key limits, duplicate keys and finite
thresholds before credentials or launch. The wire contains only the specified
model/state/questions profile. Answers require exact question IDs and distribution
keys, finite probabilities and confidence, bounded sum error, provider-selected
maximizing Choice keys (including ties), weighted Score agreement and exact legend
matching. Accepted probabilities and scores are preserved rather than normalized.

Ordinary minimum policies compare confidence with equality selecting success;
fallback has metadata but no selected key or probability. Dynamic description
arrays concatenate in source order and use one shared selected-key handler.
Generated Choice/Score records run all expanded field handlers in declaration
order, then append metadata in binder order. A failed field stops remaining fields,
later questions and continuation. Generated records retain normal nominal
construction, equality and shared codec support.

Capture-free named arms are frozen description/run values. Their descriptions
participate in the existing top-level initialization dependency graph, supporting
forward references and rejecting cycles. Generated backend runners do not add
anonymous arm syntax. Arm result/error compatibility and prohibited free captures
remain frontend checks.

The three fresh Jev consultations and pre-dispatch wording audit are in `i27-jev`.
All recommended the prepared-runner/dependency-graph approach; agreement is advice,
not validation. Live official API, Choice and Score documentation was read to
confirm wire names; Can's specification governs local tolerances and ordering.

Validation:

- Four new mixed-question runtime tests: 174 expectations covering wire shape,
  exact integers, hostile keys, counts/key-byte boundaries, missing fields,
  distributions, provider ties, score tolerance and legend errors.
- All 14 AI runtime tests pass: 345 expectations. Runtime adapters and tests pass
  strict TypeScript checking.
- Frontend checks cover distinct handler scopes, forbidden fallback key/% access,
  ordinary Score %, duplicate metadata, compatible static fields, free arm captures,
  forward arm descriptions and initialization cycles.
- Staged `questions.can` submits eight mixed registrations in one exact request.
  Observable output verifies registration/field order and runtime-swapped arm
  records. Dynamic options preserve `__proto__` and expose selected probability.
  Invalid later answers run zero handlers; duplicate dynamic keys launch nothing.
  Both first-handler and partial-record failures stop the remainder. Below-threshold
  Choice/Score use fallback; equality uses ordinary success.
- A two-stage staged variant derives the second request's state and option
  description from the first judge's completed result. The loopback server checks
  the second exact body. No retry occurs.
- The staged fixture also supplies three ordinary assertion roots, including
  generated-record codec roundtrip, and strict generated TypeScript. A separate
  raw-provider fixture executes the compiled mixed judge with all network denied.
  Supplied-completion consumer assertions and raw-provider evidence remain distinct.
- Full runtime suite: 185 tests, 19,549 expectations, zero failures.

The full compiler and integration gate passed with the pinned archive and CAN_TSC:
`go test ./compiler/... ./tests/integration -count=1` (integration: 84.254 s).
This includes the staged Noul regression, mixed question fixture, release/runtime
qualification and existing integration suites.
