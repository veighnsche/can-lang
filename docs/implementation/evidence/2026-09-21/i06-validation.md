# I06 validation

Completed 2026-09-21. Current nominal types live in `compiler/internal/types`,
independent of predecessor string-shaped checking and completion layouts.

| Requirement | Implementation and evidence |
| --- | --- |
| Nominal records/errors and invariant concrete arguments | Interned declaration/argument identities; wrong owners, existing arrays, generic containers and phantom generic variants reject implicit widening. |
| Finite disjoint variants | Transitive leaf flattening rejects duplicate/overlapping leaves, invalid leaf kinds and variant-only cycles. Catalogue `standard_failure` is the sole opaque exception. |
| Arrays/options and inhabited recursion | Least-fixed-point construction accepts array/option bases and finite variant exits; mandatory self/mutual record cycles fail. Real catalogue option shapes are used. |
| Concrete annotations | Source/file-aware builder checks declaration, signature and top-level annotations, arity, data-only void restrictions and concrete error bounds. Unused generic templates retain parameter-independent checks. |
| Callable/opaque equality rejection | Every reachable field is checked, including fields reached after recursive back edges. Ordinary recursive data remains eligible. No runtime equality replacement is introduced. |
| Immutable copy-update | Checked emitter rejects empty, duplicate, unknown, wrong-typed and opaque updates. Native execution proves receiver-once, replacement order, unchanged original and shared immutable subtrees. |
| Field/metadata collisions | Duplicate ordinary fields reject; the generated-field API checks one ordered collision domain for option/level and explicit float metadata fields. It reserves no implicit ordinary field names. |
| Canonical emitted nominal data | A private symbol stores the checked nominal identity in frozen native objects. Generated code from two packages with identical basenames/declaration names compares unequal using strict Bun.deepEquals. |
| Opaque catalogue identity | All closed catalogue type/error shapes instantiate through inventory descriptors and resolved nodes; data, map-key and failure-variant constraints are enforced. Ordinary construction/update cannot forge opaque values. |

Validation artifacts:

- [Final type/emission tests](i06-final-type-tests.txt), including actual generated
  execution with exact qualified runtime SHA-256/OS/architecture checks.
- [Full Go suite](i06-go-tests.txt), with `CAN_BUN` set so emitted execution ran.
- [Native data conformance](i06-native-data-tests.txt).
- [Staged offline integration](i06-offline-integration.txt): declaration inspection
  and packaged native data conformance run with the absolute pinned release Bun,
  network denied and unavailable PATH. Invalid recursive data refuses. Existing
  sidecar tamper/missing/architecture checks and source/bundle hash guards pass.
- [Versioned declaration report](i06-type-report-v1.json), produced inertly from
  the locked multi-package fixture; unit tests also prove relocation stability.
- [Three Jev consultations](i06-jev/README.md), retained as advice and independently
  checked by the executable graph tests.

Implementation resource guards bound active distinct instances of one declaration
and total graph size; the diagnostic does not claim to prove mathematical infinite
expansion. Finite permutations, transient growth followed by stabilization, and
long acyclic declaration chains pass. Full generic inference/reachable body
checking remains I46. Expression/region checking and complete module publication
remain I07+/I09/I10; `inspect-types` explicitly reports declaration types rather
than claiming a full program check. Native AI syntax consumes the shared generated
field rules in I16. The old files are explicitly `legacy_types.go` and
`legacy_result.go` pending scheduled predecessor retirement.
