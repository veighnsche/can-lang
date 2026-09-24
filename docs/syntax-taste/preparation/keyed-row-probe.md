# DI-09b repeated-row registration and native FormData probe

24 September 2026. This is a bounded design experiment, not a compiler or
action-adapter implementation. The first invoice form must keep one row's ID,
quantity and price associated after reorder, partial submission and duplicate
input. Current Can admits only shallow `str`, optional `str` and `str[]`
form fields. [Earlier Jev](jev-product-contracts/README.md#di-09b-form-field-identity-and-row-evidence)
favored a keyed-row trial with one close comparison; three fresh
[Jev judgments](jev-grid-contract/findings.md) later considered a concrete
equal-length cross-row omission case. The probe below is registered before its
execution and uses native `FormData` to compare association rules.

## Registered cases

1. Three complete rows `a`, `b`, `c` in order; reorder to `c`, `a`, `b` and
   require the same fields to remain with each key and display order to follow
   the submitted order list.
2. Omit `b`'s ID from a parallel `id[]`, `c`'s quantity from `quantity[]`, and
   `a`'s price from `price[]`. All arrays still have length two; positional
   zipping must be shown to misassociate values despite length checks.
3. In keyed input, omit one field, duplicate a row field, duplicate an order
   key, submit an unknown row member and submit a row not listed in order.
   Each must be a located rejection that preserves known raw strings, with no
   domain record or protected write.
4. A complete keyed form passes regardless of the native iteration order of
   its fields; a row key or field rename diagnoses at the checked action/field
   reference boundary in the planned design, not in this native-only probe.

The intended key syntax is `lines[<key>].<member>`, where `<key>` is a stable
wire row token. A separate repeated `lines_order` field names each key once
to determine display order. This experiment tests the association mechanism,
not the final parser's exact token grammar, Can type checking or live browser
rendering. The saved [script](probes/keyed-rows/probe.ts) ran under Bun 1.4.2
and passed **12/12 checks**; its full [result record](probes/keyed-rows/results.json)
includes raw and decoded values. Complete rows decoded in submitted `a,b,c`
order, then in moved `c,a,b` order without changing any ID/quantity/price
association. The three parallel arrays in case 2 each had length two, so a
length check passed, but zipping paired `id-a` with `price-b`. The keyed
decoder rejected a missing `a.price`, duplicate `a.quantity`, duplicate
order key, unknown `discount` member and unlisted `b` row; the partial
rejection retained the known raw `quantity-a` text. This establishes the
association failure of bare parallel arrays and feasibility of native
`FormData` grouping for the sample. It does not prove the proposed Can parser,
action adapter, HTML controls, database write or agent-task outcome.
