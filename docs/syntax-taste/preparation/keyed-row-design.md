# DI-09b: checked keyed rows for HTML forms

24 September 2026. Planned second increment of the source `action` form
contract, not current compiler behavior. The [native FormData probe](keyed-row-probe.md)
exposes a cross-row misassociation that parallel array length checks cannot
detect and shows a bounded keyed decoder can reject partial/duplicate input
while retaining known raw text. Three fresh [Jev judgments](jev-grid-contract/findings.md)
favored this direction; they do not prove compiler or product behavior.

## Wire shape and checked names

The action's ordinary wire record remains string-oriented. It may contain at
most one bounded `form::rows<line_wire>` field in this first increment:

```can
record line_wire
    str id
    str quantity
    str price

record invoice_form
    str seats
    option::value<str> details
    str revision
    form::rows<line_wire> lines
```

`form::rows<T>` is an opaque **form-wire** collection, admitted only when `T`
is a public transparent record whose direct members are `str` or
`option::value<str>`; nested rows, numbers, owner records, bytes, resources
and arbitrary JSON values are rejected. It is not a general codec-admissible
record, and no implicit numeric/domain conversion occurs. The action
declaration caps total body bytes and separately sets a maximum row count;
omitting that bound is a compile error when the wire record has rows. The
concrete first invoice limit is 64 rows, with the existing 2048-byte body
limit applying to the entire form.

The `lines` field owns wire names of the exact form
`lines[<key>].<member>`, with `<key>` a 1–64 character ASCII token from
`[A-Za-z0-9_-]`. A repeated `lines_order` field lists every submitted key
exactly once and determines display order. The compiler reserves the
`lines_order` name for this rows field, rejects a colliding ordinary field,
and derives the prefix/member names from checked `form::field` references.
The row key is a client-stable wire identity for a draft row, not an
authorization credential or necessarily the database line ID. A separate
`id` member is validated against the protected invoice before mutation.

The author builds controls through checked row-name operations taking the
`lines` field reference, one checked direct `line_wire` member reference and
a validated key. A wire-member rename invalidates these references, the
decoder and field errors at compile time. Arbitrary strings can still be used
at a dynamic HTML boundary, but then only runtime admission is claimed; safe
HTML escaping alone does not prove a field belongs to the action. No source
grammar beyond the `form::rows<T>` catalogue type and the action's `rows_limit`
line is selected:

```can
action save_invoice
    post "/invoice/edit"
    query invoice_key
    form invoice_form limit 2048 rows_limit 64
    returns edit_outcome
    body html
    cases
        saved status 200 swap inner
        invalid status 422 swap inner
        conflict status 409 swap inner
        forbidden status 403 swap inner
        unavailable status 503 swap inner
```

The row extension is compatible with the later `captures invoice_key`
route increment; `captures` replaces `query` independently of row decoding.

## Decode, rejection and retained input

Use native `Request.formData()` or equivalent Bun form parsing once within
the declared body budget, then group by exact key/member. Reject invalid key
syntax, duplicate order keys, a missing/unlisted row, duplicate row member,
unknown row member, missing required member, unknown top-level field, scalar
duplicate, excess row count and excess bytes before calling the protected
handler. A missing optional member is `none`; a present empty string is
`some("")`. Iteration order of fields cannot change associations; only
`lines_order` controls output order. Each row retains its key and raw text.
Invalid numbers remain strings until explicit application validation and
owner factories run. The decoder cannot mint an owner record.

A structural 422 must not discard the submitted values. The adapter builds a
read-only `form::rejected<invoice_form>` containing an ordered, sanitized view
of **known** raw field names and all their submitted string values, plus
located `form::problem` entries. It is a partial wire observation, not an
`invoice_form` or `line_wire` instance; duplicate values are retained in the
observation and flagged, never arbitrarily selected for a protected write.
Unknown names are diagnosed but not echoed into rendered controls. The
server-only `action::mount` accepts a separate safe renderer for this
pre-handler rejection in addition to the valid-wire handler and normal
outcome renderer. Its response is actual 422 with the declared visible swap
policy. A renderer failure follows the action's generic infrastructure 500
policy. This refines the earlier illustrative `invalid(none, ...)` branch in
[the action packet](action-contract-design.md#common-proposed-semantics-for-a-and-b):
domain-invalid **fully decoded** forms still use the ordinary `invalid` case
and typed `invoice_form` raw values; structurally malformed forms use
`form::rejected` and never invoke the business handler. Both paths can show
retained text, but neither fabricates a typed domain value from bad input.

The exact renderer and catalogue signatures must preserve this three-callback
separation: handler `(request, key, invoice_form) -> edit_outcome`, outcome
renderer `edit_outcome -> html::safe`, and rejection renderer
`form::rejected<invoice_form> -> html::safe`, with declared finite rendering
errors. The source declaration fixes the action identity; the callbacks bind
at server mount. Browser code can import the wire field/row names and action
URL/post operations, but not the server handlers or raw request.

## Native lowering and acceptance

The adapter performs one native form parse and ordinary `Map` grouping,
publishing immutable checked row records only after all structural checks.
No Can-written parser, ORM, general nested form language or automatic domain
validator is generated. Preserve raw observation only in the rejected branch,
subject to body/row limits and safe rendering; no private owner fields or
unknown attacker names enter a typed result.

Production acceptance includes the [12 native probe cases](keyed-row-probe.md),
real browser forms with reordered rows, partial/duplicate/unknown controls,
max limits, missing/empty optional values, multiple identical display texts,
hostile field text, changed row IDs and two users editing different tenants.
Inspect the rendered 422 text/order, focus/announcement, actual HTTP status
and unchanged database row. A malformed row must never reach the protected
operation. Change a wire member and a separate domain member independently;
checked references diagnose the first while the owner validation handles the
second. Compare the same task with validated parallel arrays and a JSON
client action under the evaluation protocol before claiming any agent-token
advantage.
