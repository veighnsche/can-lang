# I49 validation

Completed 2026-09-21. Error allocation now connects the validated project graph to
canonical resolver identities and sealed concrete types. Exact domain bounds and
compiler-generated payload plans support the later completion-region pass.
Standard and domain occurrences retain private source/invocation origin and native
cause, while fresh failures remain distinct even when they reuse one native cause
or one domain payload.

| Acceptance requirement | Evidence |
| --- | --- |
| Allocation/source/lock agreement | Existing strict project loader checks manifests, registry ranges, source declarations and lock snapshots. The error registry additionally ties each active allocation to its nominal owner and source ID; duplicate, retired, reserved, missing and mismatched allocations reject. Declaration inspection invokes this check. |
| Exact authored emits | Bounds accept only sealed allocated nominal errors, reject duplicate concrete specializations and compare exact declaration/allocation/type identities. Undeclared generic specializations fail; ambiguous bare error arms require explicit normalization. Standard failures remain outside domain emits. |
| Catalogue payload contracts | Generated descriptors derive from the closed catalogue. Runtime plans verify canonical identities, allocation, field types and variant leaves. All 51 catalogue errors, project generic errors, nested options and their malformed payload controls execute from Go-generated checked plans on the pinned Bun. |
| Safe standard failures | Arithmetic, bounds, resource-state, assertion, cleanup and nonfinite-sort categories use C9 messages. Native descriptions inspect own data descriptors only after proxy rejection; getters, revoked proxies, native Error subclasses, circular objects, symbols and primitives follow fixed rules. Adapter defects fall back to `native failure` while retaining the original cause. |
| Private occurrence identity | Each origin receives a fresh frozen opaque token and shared unique bigint ID; forwarding preserves it. Generic errors retain one declaration allocation and distinct concrete argument identities. Private WeakMaps retain cause, payload and origin; public standard projections expose only kind, message and occurrence ID. |
| No automatic secret disclosure | Tests exclude credential/body/cause/stack extras from descriptions, never invoke accessors, and reject forged/proxy-wrapped values. Explicit thrown strings and own Error message data remain the C9-specified message; this is not an arbitrary-content redactor. |
| Native integration | Standard projection expressions parse, check and lower to private adapters and execute on the qualified runtime. Staged native standard/domain tests execute offline from a clean cwd with an unavailable PATH. |

Validation artifacts:

- [Error registry, bounds, plan and emitted projection tests](i49-error-tests.txt).
- [Full Go suite](i49-go-tests.txt), with qualified emitted execution enabled.
- [Bun suites](i49-runtime-tests.txt): 27 tests, 1,063 expectations.
- [Separate native capability report](i49-native-capabilities-v1.json): the pinned
  target now explicitly requires native Error-brand detection as well as proxy
  detection; missing APIs reject qualification without fallback.
- [Staged offline integration](i49-offline-integration.txt): sidecar and packaged
  catalogue checks, including source/bundle integrity and negative runtime cases.
- [Three Jev consultations](i49-jev/README.md), independently checked by tests.

The I10 completion pass will provide escaping-error sets and distinguish handler
regions; I49 does not claim to check arbitrary authored function bodies or to
execute a full current-language program. Opaque resource/callable payload admission
requires the corresponding maintained adapter's private brand hook; absent hooks
reject those values. No structural impersonation or fallback admission is used.
