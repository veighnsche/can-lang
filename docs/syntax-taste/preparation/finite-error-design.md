# DI-03/DI-04 finite error-set helper design

24 September 2026 · bounded design for an experiment, not adopted production
syntax. The [current-Can probe](generic-helper-probe.md) passed 28/28 assertions
with two separately authored error domains. Its fixed-bound helper forces each
caller to handle an impossible other-domain error; its generic result-as-data
helper needs adapters in both directions. Direct `match chain` keeps precise
errors but duplicates the shared sequence. The existing helper already shows
that ordinary callable parameters can make required operations explicit.

## Recommended trial contract

Add one finite **error-set parameter** to an authored helper. Its declaration
must name that parameter; the public callable inputs and outward `emits` bound
must state every use. The proposed grammar below is a concrete trial spelling,
not parseable current Can:

```can
fn int with_audit<errors E>
    emits [E, audit::audit_down]
    given
        callable int () emits [E] action
        callable int (int) emits [] transform
        callable bool (int) emits [audit::audit_down] audit
    match call action()
        E as failure => failure
        ok int original => do
            int changed = call transform(original)
            match call audit(changed)
                audit::audit_down as failure => audit::audit_down(failure.reason)
                ok bool accepted => do
                    match accepted
                        false => ok original
                        true => ok changed
```

`errors E` is a distinct generic parameter kind, not a value type or a general
effect row. E ranges over a **finite** set of fully specialized nominal error
identities and may be empty. `emits [E, audit::audit_down]` means normalized set
union, including deduplication if E itself contains `audit_down`. A declaration
must not omit E and rely on body or caller inference to invent a public
failure parameter. Concrete errors use canonical declaring-package identity
plus type arguments; authored/reporting numbers are not type identity.

At a use, infer E from the supplied `action` value's **declared static**
`emits` set, before any widening conversion. If a callable declares more
errors than its body currently reaches, those errors remain in E. An optional
explicit `with_audit<[invoice::missing, invoice::locked]>(...)` asserts the
**same normalized set**, rather than selecting an arbitrary superset. This
gives an agent a stable call-site boundary when desired and diagnoses a
callback-declaration change there. If more than one input could determine E,
the prototype must require their sets to agree or require a single explicit
E; it must not infer from the enclosing caller's broad bound. If no input
determines E, explicit application is required. Callable argument checking
retains normal input/output requirements. `transform` and `audit` are named
public operations; no hidden `+`, ordering, or implicit instance is introduced.

The grouped `E as failure => failure` arm only relays the exact completion it
received. It cannot read fields, construct an unknown E member, change its
identity, or swallow it. A concrete named error arm may precede it and handles
that member; the grouped arm covers the remaining E members. Duplicate named
arms or an uncovered member diagnose at the helper source. An E of `[]` makes
the generic forwarding arm unreachable at that specialization and emits no
branch. Any helper-origin error, such as `audit_down`, still needs a named
arm or another explicit forwarding path and must appear in the written bound.
The helper has no retry policy: relaying an error once does not say whether it
is safe to retry an operation.

This feature changes compile-time substitution and checking. It creates no
runtime row value: generated TypeScript calls the supplied functions and
forwards the existing Can completion object through the established adapter.
No new scheduler or exception transport is implied.

## Two independent callers

The following completion signatures illustrate distinct public contracts;
the shown `match` arms are the required caller handling, not implemented code:

```can
fn int invoice_action
    emits [invoice::missing, invoice::locked]
    ...
fn int payment_action
    emits [payment::declined]
    ...

fn int invoice_checked
    emits [invoice::missing, invoice::locked, audit::audit_down]
    match call helper::with_audit(callable invoice_action,
                                  callable double, callable audit_positive)
        invoice::missing as failure => invoice::missing(failure.key)
        invoice::locked as failure => invoice::locked(failure.key)
        audit::audit_down as failure => audit::audit_down(failure.reason)
        ok int value => ok value

fn int payment_checked
    emits [payment::declined, audit::audit_down]
    match call helper::with_audit(callable payment_action,
                                  callable double, callable audit_positive)
        payment::declined as failure => payment::declined(failure.code)
        audit::audit_down as failure => audit::audit_down(failure.reason)
        ok int value => ok value
```

The invoice call specializes E to `{invoice::missing,
invoice::locked}` and the payment call to `{payment::declined}`. Neither caller
acquires the other domain's error. A callback with `emits []` specializes E to
empty and still exposes `audit_down`. A generic error such as
`store::not_found<invoice::id>` and the same declaration specialized to
`payment::id` are distinct members.

## Comparison against the strongest current idioms

| Approach | Exact outward failure set | Public required operations | Main cost/risk in the two-caller case |
| --- | --- | --- | --- |
| Direct `match chain` | Yes | Per caller | Repeats the common sequence. The current probe measured 27 source lines for two one-error chains including assertions. |
| Shared helper with fixed union | Helper advertises both domains; callers restore exactness by mapping or handling unreachable cases | Yes, via callable inputs | Each caller must handle errors its action cannot produce. The current probe confirms that omitting an unused arm is rejected. |
| Generic result-as-data | Yes for data payloads, with exact Can completion restored at boundary | Yes, via callable inputs | Completion/data conversions are needed both before and after the helper; failure representation can drift from named error identity. |
| Finite E with explicit operations | Yes by substitution of the action's declared set plus fixed helper errors | Yes, via callable inputs | Requires new kind, substitution, grouped forwarding, diagnostics, and agent evidence; no current executable proof of benefit. |

The result-as-data path is still a valuable control when callers deliberately
need a failure *value* for storage or aggregation. The finite-E design is for
transporting Can completions through a reusable higher-order operation without
changing them into application data. It does not justify replacing ordinary
data results or every fixed-bound helper.

## Acceptance and negative cases for a disposable prototype

1. **Two domains and native behavior.** Compile the invoice two-error callback,
   the payment one-error callback, and an empty-error callback against the same
   helper. Exercise success, each domain failure, and `audit_down`. Inspect the
   generated TS and runtime observations: exact member identity and payload
   survive forwarding; no row object or unrelated error is introduced.
2. **Growth and repair.** Add `invoice::expired` to the invoice callback's
   declared bound. The invoice call's inferred E and outward type change;
   `invoice_checked` reports its missing arm or bound. The payment call is
   unaffected. With an explicit old E assertion, the diagnostic occurs at
   the helper call and names expected and actual sets.
3. **No hidden widening.** A callback declared with a broad error union keeps
   that union even if one branch is currently unreachable. Passing it under
   an explicit smaller E fails. The enclosing caller's broad `emits` list must
   not silently choose E for an unrelated callback.
4. **Forwarding and narrowing.** An `E as failure` arm may forward but may not
   read `failure.reason`, remap the error, or construct an E member. A concrete
   named arm may inspect its member before the row remainder; duplicate or
   missing arms produce source-positioned diagnostics. Specialize once with
   `E=[]` and once with `E` containing `audit_down` to test empty and overlap.
5. **Operation contract.** Passing an `int -> int` function as the required
   `bool (int)` audit operation fails with a diagnostic naming `audit` and its
   expected callable type. Making `transform` fallible without adding its
   errors to the written helper contract fails at the helper declaration.
6. **Identity and composition.** Two independently authored dependencies with
   overlapping local error names or diagnostic numbers remain distinct by
   canonical declaration identity. Distinct specializations of one generic
   error remain separate members. Dependency-local aliases do not become
   nominal identity.
7. **No accidental retry.** A separate retry example must name retryable
   cases and final-attempt behavior; finite E by itself may only relay. A
   broad catch-all retry is rejected as a claimed consequence of this design.

Register creation, controlled error-addition/refactor, and diagnostic-repair
tasks before comparing this prototype with direct chains, fixed bounds, and
result-as-data. Use the [evaluation protocol](evaluation-protocol.md): five
independent attempts per candidate/model/task variant, semantic checks first,
then total tokens per successful task and wall time. The current 28 passing
assertions show the baseline cost, not a need to ship the new parameter. If
the finite-E prototype does not yield a reproduced contract or agent advantage
on held-out larger cases, retain the simpler current idioms.

## Jev advice and remaining status

The [follow-up consultation](jev-finite-error-followup/findings.md) strongly
favored explicit grouped forwarding in three reworded requests. It split on
mandatory explicit E versus callback-derived E (one versus two top choices,
all with material probability on both). Callback-derived E with an optional
exact assertion is the proposed experiment because it preserves the action's
declared finite set and diagnoses widening. This is a technical choice for the
trial, not measured adoption evidence or proof from Jev. No production grammar,
checker, runtime, or source has changed.
