# Reconciliation after the independent phase

This is coordinator synthesis of source rechecking and explicit followups to the independent reviewers. It is not additional blind review. Initial core/native reports are retained so changes in judgment remain visible.

## Variant identity: behavior established, optimal contract unsettled

Jev preferred extensional leaf-set compatibility in requests 1 and 2, but retained current behavior in request 3. The source and probes still establish the same fact: the direct generic-variant conversion fails while bridge and leaf conversions succeed.

The core reviewer identified a coherent defense of the guard: it rejects accidental direct mixing of specializations, and an explicit annotation with a general variant can be understood as discarding the distinction. Assignability need not be a mathematical subtype relation. The weakness is that both adjacent conversions are implicit and the underlying leaf inhabits both specializations, so calling the distinction invariant nominal branding overstates it.

Disposition: medium-priority contract clarification, not type unsoundness. Compare extensional unions, a documented conservative direct guard, and genuinely nominal wrappers using the same probe suite plus ordinary option and generic-record examples. Extensional compatibility fits the current representation, but it is not uniquely compelled by the evidence.

## Abstract values: library scope versus uniform transparency

Jev selected package-owned abstraction once and catalogue-only protected values twice. Public construction remains an observed source-language limitation regardless of preference.

The strongest case for catalogue-only opacity is uniformity: ordinary records have simple construction, matching, equality, serialization, and attached assertions. Maintained opaque resources also carry runtime ownership rules, which should not casually become author-extensible.

The case for authored abstract immutable values is narrower: representation hiding can let a library establish a validation invariant without conferring native resource privileges. Transparent records require repeated validation or a factory convention that callers may ignore. Neither alternative replaces authorization or validation of untrusted input.

Disposition: high-priority scope experiment for reusable libraries, not a current compiler correctness bug. Compare email, positive-quantity, and tenant-identifier packages, including codec/assertion/equality implications. The final report does not claim the preferred remedy is already settled.

## Race diagnostics: consumed failures have a coherent meaning

Jev selected current timing twice and uniform losing-standard diagnostics once. The independent native reviewer initially preferred uniform reporting. Rechecking `runtime/coordination.ts:61-82`, `runtime/owner.ts:399-403`, and `runtime/test/coordination.test.ts:120-153,398-436` established that pre-winner failures are deliberately consumed by the first-success search. Tests require that behavior; it follows the selected contract.

A service can successfully fall back to another replica while an early `native_exception` receives no late-failure diagnostic. That is an observability concern. Conversely, reporting every failure of an intentionally redundant search can create noise after a successfully consumed outcome; if all participants fail, the aggregate retains the failures.

Disposition: observability tradeoff, not language correctness blocker. Preserve selection semantics. Consider an opt-in diagnostic policy only after a concrete service case justifies it. The initial report's stronger categorization is not the final conclusion.

## Scoped results: runtime safety versus earlier feedback

All consultations favored narrow diagnostics. This is not evidence that the current owner model is unsafe. `docs/syntax-taste/platform-testing-spec.md:555-563,1107-1109` explicitly permits carrying scoped handles without static lifetime meaning; subsequent use is guarded. `runtime/test/owner.test.ts:185-215` checks the lifetime behavior.

The native reviewer withdrew a broad result-type rejection proposal. A transaction callback can legitimately return an enclosing-owned pool or other live resource; a blanket ban on resource-valued results would reject those programs. Detecting the callback's own transaction through nested records/callables requires provenance or value-flow information; callable types alone do not encode capture lifetime.

Disposition: documented ergonomics limitation. Explore diagnostics for obvious escapes with both invalid and valid counterexamples. No general affine type system is inferred.

## Other calibrations

- Exact-path routing limits resource paths, not all CRUD; query IDs retain normal HTTP method semantics.
- SQL syntax/cardinality checking plus runtime row validation is real protection, but does not prove a projection against a database schema.
- HTML form wire strings are an appropriate representation; typed accumulated business validation can be a library layer.
- A closed native catalogue is an intentional trust boundary. No concrete SDK impossibility was established; an adapter proposal needs a demanding example.
- Server-rendered HTMX is the current selected frontend. Browser-local Can is a separate strategic scope decision, not a missing implementation of the existing specification.
- Generic error-row and capture proposals need comparisons with result data, fixed bounds, named wrappers, and explicit context records. Agreement by reviewers or Jev does not remove that requirement.
