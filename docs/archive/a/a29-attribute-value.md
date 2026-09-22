# a29 — Attribute-value encoder plus `attribute__text` composition

Status: shipped. Resolves the item a27 left scheduled: the
attribute-value encoder and the fate of the specified
`invalid_attribute_value`. The witnessable value-side reject a27
allowed for was found (NUL) — and the error is nevertheless
dropped, for the corrected reason below.

## Rule

- Canonical serialization is `name='value'` (single-quoted).
  Double quotes are inexpressible in can string literals: a source
  `"...\"..."` denotes backslash followed by quote (verified by
  probe against both the evaluator and the TS emit), and no
  alternate quoting exists. So `name="value"` cannot be assembled
  in-language, and a `"`-detection arm cannot even be written.
  The catalogue mandates no quote style (ASTRA_STDLIB line 321
  gives the signature only); one canonical spelling per meaning,
  documented here.
- `html__attribute__value_from(orig, s, acc, n)` mirrors
  `html__text__escape_from` arm-for-arm plus the quote arm:
  `&` → `&amp;`, `<` → `&lt;`, `>` → `&gt;`,
  `'` → `&#39;` (which the tokenizer resolves back inside the
  single-quoted state), NUL → `html.nul_byte(value = orig)`.
  `"` passes through literally — safe, since it cannot terminate
  the single-quoted value. Kept separate under the keep-separate
  default: quoting context and quote arm differ.
- `html__attribute__text(name, raw)` keeps the catalogue's
  brand-typed signature. The name brand is evidence its holder
  passed the gate; the serialized spelling is reconstructed by
  the owner-local `html__attribute__spelling` from the closed
  admitted domain — not extracted from the brand. No
  brand-to-string flow exists, and none is needed for a finite
  domain. Singleton today: the only inhabitant is `"title"`.
- INVARIANT (correspondence obligation): the reconstruction must
  mirror `html__attribute__name`'s admitted set exactly. Adding a
  name without extending it forges output. The spelling rows pin
  every admitted name; extend them with the gate.

## The corrected derivation, twice over

This slice broke an exclusivity claim and corrected a
justification in the same sitting. First, the F1 frame (fused
raw-name input as "the only sound construction") confused
extracting an arbitrary branded string with reconstructing a
member of a known finite set. The outside review supplied the
missed construction; its load-bearing fact (singleton-`title`
domain) was verified in source before adoption. Jev's 0.83
stands for what it actually judged — F1 is sound — not for the
exclusivity the frame wrapped around it.

Second, the old coverage argument for dropping
`invalid_attribute_value` ("no witnessable reject, therefore the
arm is impossible") was wrong in both halves: NUL witnesses a
value rejection, and `emits` is an upper bound that admits
unrealized entries (a12 precedent; check.go:634), so keeping the
kind would compile clean. The honest decision: NUL witnesses the
shared `html.nul_byte` cause already named by this module, and
no distinct value-side contract exists to justify a second
spelling. Dropped on semantic grounds, catalogue amended, not
compiler-forced.

## Alternatives recorded, not shipped

- F1 (fused raw-name input) survives as a reasonable raw-input
  API and stands approved as a catalogue amendment — not built,
  because the amendment's premise (unimplementable signature)
  was falsified. Its conditions, if ever revived: validate and
  serialize the same spelling, keep the encoded `str` available
  before sealing, never let a raw-assembly helper mint.
- Double-quoted canonical form stays blocked on a language gap,
  not a design choice: string-literal escapes (or a quote
  primitive) are the prerequisite. No in-language route exists
  (literals, slicing, case transforms, and cross-module calls
  all fail to produce the scalar).
- The catalogue's `Html__AttributeText` (encoded data, not a
  complete attribute) gets no type until a slice needs it as an
  interface; the worker result stays unbranded per the
  `Html__Escaped` precedent.

## Proof costs

- Shapes proved in /tmp before touching the module: bare-param
  record passthrough, nested `match call` in `on` arms,
  chained concatenation. No new machinery in the shipped code.
- 19 new decision-table rows, all green at compile time (65/65
  with the pre-existing 46): 12 worker rows (empty, plain,
  three escapes, quote, mixed, astral, five NUL positions —
  one per escape arm plus first), 1 spelling round-trip row,
  6 composer rows (plain, empty, amp, quote, tags, NUL).
- NUL rows carry raw `0x00` bytes like the a26 rows (38 NULs
  total, 21 baseline + 17 added, counted exactly).
- Independent oracle: the generated `html.ts` runs under node
  against breakout inputs (`' onclick='x"y`, mixed markup,
  NUL) — quote neutralized, `"` literal-but-safe, NUL typed.
  Scratch probe retained at `/tmp/attr-probe.mjs`, not committed.
- `errors.json` regenerated with complete `hit_by_tests`;
  `go test -count=1 ./...`, modcheck, and gramcheck all green
  fresh (the first `go test` run was cache-stale and rerun
  forced).

## Still scheduled (not silently dropped)

- A second admitted name reopens the reconstruction design: a
  comparison chain may need same-brand equality, whose
  existence is unverified (cross-brand `==` is CAN6003; same
  brand untested). Recorded open question, not assumed
  machinery. The round-trip rows are the tripwire meanwhile.
- `invalid_attribute_value` is gone from this constructor's
  contract; if a future value-side reject needs a distinct
  name, that slice justifies it fresh.
- Boolean names, remaining attributes, fragments, elements
  follow in program order.
