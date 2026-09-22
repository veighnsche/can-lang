# Prompt: adversarial pre-implementation review of B4 (issue #42)

Paste this into the reviewer with [docs/bytes-plan.md](bytes-plan.md)
(v3, items 1–2 + B4 row) attached plus the context files below.
B4 is not implemented yet — this review attacks the consumer design
before it lands in the standard library. B1–B3 (values, export
authority, generic encoder) are shipped and tested; do not re-litigate
them except where B4 depends on them.

---

Review the B4 design: `html__render__utf8(document: Html__Safe) ->
Bytes__Value` in `std/html/html.can`, authorized by an HTML-owned
`exports_utf8 Html__Safe via html__render__utf8@1` grant, rendering
exact serialization bytes with NUL preserved and no second escaping.
For each item, return: holds / broken with file:line evidence and a
concrete counterexample / unresolved with the exact missing source.
No B4 compiler gates have been run — there is no B4 code yet.

1. Grant placement. The grant, the `Html__Safe` brand, and the render
   function must share one canonical module. Is `std/html/html.can`
   exactly one module for this purpose, and does the grant text pin
   the right function revision? What breaks if the HTML module is
   ever split?
2. Promotion reach. `Html__Safe` admits `Html__Text` and
   `Html__Attributes` via `seals_from`, and `html__text__node`
   promotes without invoking the escaping encoder. Enumerate every
   value that can reach the exporter through permitted promotions,
   including NUL-bearing ones. Is each such flow intended, and does
   the design's "inspect the incoming seals_from paths" obligation
   close with evidence or with an assertion?
3. NUL end to end. Render accepts and preserves NUL (no filtering,
   rejection, or escaping). Trace a NUL-bearing `Html__Text` through
   `html__text__node` into `html__render__utf8` and name the exact
   output bytes. Then state the downstream consequence: who consumes
   NUL-bearing Bytes, and does any downstream contract assume
   NUL-free input? A preserved NUL the next stage cannot handle is a
   hole, not a feature.
4. No second escaping. `render("&amp;")` must yield the bytes of
   `"&amp;"`, not of `"&"`. Distinguish, with committed-row shapes,
   the escaping encoder's output alphabet from the renderer's byte
   alphabet, and show where a double-escaping regression would be
   caught. Is there an existing test that would catch render calling
   the escaper by mistake?
5. Unchanged contracts. The brand definition, `seals_from`, promotion
   identity, and `html.nul_byte` escaping behavior must be byte- and
   row-identical after B4. What is the complete proof (existing
   suite, goldens, registries), and which of those artifacts does B4
   regenerate vs leave untouched? Name any artifact B4 must touch
   that the plan omits.
6. Stdlib plumbing. `html.can` gains a function returning the
   compiler-owned `Bytes__Value`: required `provides` entry, header
   inventory consequences, `errors.json` regeneration (or a reasoned
   no-op), and `html.ts` regeneration. Does the compiler-owned
   record need a `uses` entry, and what emits the `Bytes__Value`
   definition into `html.ts`? Walk the exact regen commands and
   their expected diffs.
7. Shape compliance. Apply the exact exporter predicate (one
   parameter of the granted brand, `Bytes__Value` return, empty
   contract, single kernel match returning the result unchanged) to
   the proposed `html__render__utf8` text. Quote the function and
   check every clause — a stdlib function that fails certification
   is a design failure, not a test failure.
8. The acceptance claim. B4 either proves "`Html__Safe → Bytes` with
   no new wall" or finds the wall. If the implementation needs any
   machinery beyond B1–B3 (checker, evaluator, emitter, or grant
   changes), name it precisely. "No new wall" must survive contact
   with the real `html.can`: generics-free, effects-free, and
   revision-pinned.

Close with the single highest-risk hole in the B4 design and the
concrete fixture (source text + expected bytes or diagnostic) that
would expose it.

---

## Context files

- `docs/bytes-plan.md` (v3 — the design under review)
- `docs/bytes-workstream.md` (requirements, acceptance, non-goals)
- `docs/archive/a/a46-bytes-export.md` (shipped B2: grants, kernel, barrier)
- `docs/archive/a/a47-bytes-encode.md` (shipped B3: generic encoder)
- `std/html/html.can` (the module B4 changes)
- `std/html/html.ts` (the committed artifact B4 regenerates)
- `std/html/errors.json` (the registry B4 must justify touching or not)
- `docs/encoder-nul-policy.md` (NUL boundary the design relies on)
- `compiler/export.go` (the shape predicate and certifier B4 must satisfy)
