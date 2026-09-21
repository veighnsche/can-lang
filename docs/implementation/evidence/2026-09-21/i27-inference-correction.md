# I27 correction — inferred generic generated-record spreads

Independent review found that ordinary Choice accepted `...call identity(choices)`
for a concrete arm record while generated-record Choice required explicit generic
types. Generated fields were projected before graph sealing, and the projection
only resolved explicitly instantiated function results.

The projection now solves ordinary finite argument and receiver constraints using
the existing transactional structural equality walk. Its private builder entry
accepts only complete concrete nodes interned in that builder's graph. Public
inference still requires sealed evidence. The shape pass does not admit an
expression: normal sealed checking still verifies calls, bodies, effects, arm
results and error bounds. Conflicting, ambiguous or cyclic evidence is rejected;
there is no search over candidate instantiations.

Arguments retain fixed/variadic and literal/runtime spread distinctions. Multiple
passes allow later concrete arguments to supply context to an earlier empty array.
Generic method receiver evidence uses the same solver. Generated field names and
order come from the resolved record shape, independently of runtime arm identity.

Validation:

- An isolated `63514c6` snapshot with the new regression rejected six valid inferred
  cases with `spread call requires explicit concrete type arguments`; the explicit
  case passed.
- Corrected compiler passes seven cases: inferred, explicit, nested calls, array
  input, contextual empty array, variadic literal spread, and inferred receiver.
- Ambiguous parameters, conflicting argument types, explicit arity mismatch and
  generated shape cycles still reject. A separate regression verifies public
  inference rejects unsealed evidence without mutating bindings.
- The maintained mixed fixture now passes inferred `identity` calls through both
  ordinary and generated-record arm spreads. Its staged test passes the exact
  mixed request, handler/field ordering, pre-handler whole-batch rejection,
  threshold fallback, partial-record failure, dependent second request, strict
  generated TypeScript and deny-network raw-provider execution (7.957 s).
- Three fresh consultations and their assessment are in `i27-inference-jev`.

I28 schema work was set aside while correcting this confirmed I27 issue; no I28
implementation or acceptance claim is included in this correction.

Full compiler and staged integration validation passed with the pinned archive
and CAN_TSC: `go test ./compiler/... ./tests/integration -count=1` (integration:
89.301 s). This includes the mixed-question pipeline, release/runtime checks and
all existing integration gates.
