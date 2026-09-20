# Round 1 Jev payload preflight review

## Scope and method

Reviewed `request-1.json` against both the canonical
`../jev-design-2026-09-20/request-1.json` and `problem-ledger.json`. Also
compared every natural-language value with the new `request-2.json` and
`request-3.json` to check independence of wording. No Jev or other API call was
made.

The mechanical comparison covers 54 natural-language fields. Topic order,
question order, criterion key order, choice IDs, and the `jev-latest` model are
unchanged. Numeric-token multisets match the canonical facts for every topic.

## Shared material

Round 1 retains the complete advisory-only purpose, inability to research or
explain reasoning, absence of implementation authority, and explicit uncertain
outcome. All six fixed constraints remain present:

- immutable Can for AI coding agents, top-level named authored functions,
  asynchronous implementation with sequential ordinary calls, and the four
  excluded facilities;
- equivalent JavaScript/Bun lowering, adapter-only contract enforcement, zero
  external users, no compatibility obligation, and compiler simplicity;
- the four exact Promise mappings, no loser cancellation, and one ordered bare
  `all_failed` aggregate;
- finite explicit `emits`, the separate standard-failure channel, available
  nominal data forms, and absent union/error-set syntax;
- `choice_arm<T> emits [errors]`, return-first named arms, `llm str`, structured
  LLM grouped state, named fetch encodings, `http::response<T>`, explicit float
  formatting, Bun SSR with upstream HTMX, and exclusion of LLM tools; and
- permission to change technical defaults and adapters without changing the
  approved syntax or product boundary, with no further user questions.

The evidence disclaimer still distinguishes documentation and bounded probes
from implementation proof, and probabilities from authority. `snapshot_scope`
correctly frames the payload as a replay from before the earlier consultation
and withholds earlier Jev results and later selections.

## Topic-by-topic coverage

| Topic | Fact and qualification coverage | Literal numeric and identifier check | Alternatives retained in original order |
| --- | --- | --- | --- |
| `integer_codec` | Preserves arbitrary-precision `int`, the shared codec, exact source-token access versus rounded `Number`, schema-valid integral spellings, every forbidden coercion, native general parsing, exact coefficient/exponent normalization, and pre-expansion bounds. | `bigint`, `JSON.parse`, `Number`, `BigInt`, `1.0`, `1e0`, `int`, `float`, and `1eHuge` are present. | `integer_lexemes`, `exact_integral_tokens`, `review_needed` |
| `generation_shapes` | Preserves general codec variants and recursive records, finite requirements, the historical narrow profile, provider capability and root restriction, independent validation, the demonstrated workflow, broader emitter costs, and all exclusions. | `case`, `value`, `bool`, `str`, `int`, `float`, `anyOf`, depth `8`, `1024` properties, and `65536` bytes are present. | `bounded_initial_subset`, `reuse_broader_type_graph`, `review_needed` |
| `aggregate_type` | Preserves finite `U`, ordered duplicate occurrences, bare-arm syntax, nominal/invariant typing, contextual `F`, private ignored payloads, named helpers, multiple covering variants, and wrapper normalization. | `U`, `standard_failure`, `all_failed`, `all_failed<F>`, `F[]`, `F`, and `emits` are present. | `lazy_expected_type`, `always_contextual_type`, `unique_cover_search`, `review_needed` |
| `resource_deadline` | Preserves indefinitely pending work, absence of loser cancellation, lease/sublease invariants, historical graceful close and close failure, root waiting, lack of eventual-exit assurance, no universal timeout/cancellation, hard-kill consequences, and the external-supervisor possibility. | `Promise.race([])` and `close/commit/rollback` are present; no numeric fact was added or removed. | `drain_without_host_policy`, `runtime_hard_stop`, `external_supervisor_boundary`, `review_needed` |
| `sequential_map` | Preserves effectful async callbacks, strict sequential order, early failure with no partial output, the historical predecessor graph, the complete local probe, TC39 qualification, pinned-runtime scope, payload shielding, boxed/indexed lowering, and traversal exclusions. | `Promise.all`, `Array.fromAsync`, `Bun 1.4.2`, `[1,2,3]`, `start1/end1/start2/end2/start3/end3`, maximum `1`, `[2,4,6]`, rejection at `2`, `[1,2]`, `[]`, and `TC39` are present. | `predecessor_promises`, `native_from_async`, `shared_handwritten_traversal`, `review_needed` |
| `fixture_queue` | Preserves named assertions, complete repeated `when` rows, no participant syntax, full root/call identities, deterministic transitive fixtures, expected-argument and no-live-call rules, historical shared FIFO, canonical path barriers, the one-schedule limitation, separate native timing tests, and assertion-local drainage. | `fn`, `when`, `FIFO`, `P3-P4`, and `Q2/Q10` are present. | `canonical_shared_fifo`, `participant_local_fifo`, `argument_matching`, `review_needed` |
| `generic_checking` | Preserves explicit parameters and every absent constraint form, ordinary generic operators/methods, invariance, two-phase historical checking of all reachable branches, recursion limits, public instantiation, assertion limits, the parametric alternative, unchanged syntax, and catalogue specialization. | `typeclass`, `trait`, `bound`, `overload`, `effect-parameter`, and `C4` are present. | `concrete_specialization`, `parametric_only`, `review_needed` |
| `promise_payloads` | Preserves callable record fields, legal `then`, native assimilation and its risks, mandatory native Promise reuse, historical coordination tags versus the missing universal shield, absence of layout compatibility, source/wire name preservation, and immutability. | `then`, `Promise`, `await`, `TypeScript`, and `C4-C7, Q2` are present. | `private_completion_boxes`, `mangled_record_fields`, `reserve_then_name`, `review_needed` |

Every question still instructs Jev to use all of `common` and the named topic,
use other topics only as background, remain within supplied evidence, and avoid
research. Candidate descriptions retain their original benefits, costs,
exclusions, and uncertainty condition; no option was rewritten to favor a later
engineering selection.

## Corrections and permitted metadata cleanup

Two round-1 wording defects were corrected:

1. The initial round-1 constraint ended with “The user does not want another
   syntax questionnaire.” This narrowed the canonical prohibition on any new
   questionnaire. It now says “Do not send the user additional questions.”
2. The initial round-1 sequential-map fact used the sentence “Local Bun 1.4.2
   provides native Array.fromAsync.” That exact sentence also appeared in round
   3. It now says “The observed local Bun 1.4.2 runtime includes native
   Array.fromAsync.” The observation and identifiers are unchanged.

The shared mechanical cleanup shortened two `sections` metadata values in all
new rounds: `Q6 finite inference, Q9, C4/C9` became `Q6, Q9, C4/C9`, and
`C4-C7, Q2; newly isolated runtime representation obligation` became
`C4-C7, Q2`. These fields now contain technical section identifiers only. This
removes descriptive prose without changing facts, options, or referenced
sections and is therefore permitted by the rewrite instructions.

No other omission, factual change, altered candidate, or conclusion-shaped
rewrite was found after correction.

## Wording verdict

Round 1 passes the full-prose variation requirement:

- versus the canonical request: 0 unchanged natural-language values and 0
  reused sentences; highest same-field similarity is 0.759;
- versus new round 2: 0 unchanged natural-language values and 0 reused
  sentences; highest same-field similarity is 0.673; and
- versus new round 3: 0 unchanged natural-language values and 0 reused
  sentences; highest same-field similarity is 0.684.

The final `request-1.json` SHA-256 is
`6fd8f83252c06a68f66cf2c8e1656dac5dea37f9409e8482d998db4d9633bb2d`.
