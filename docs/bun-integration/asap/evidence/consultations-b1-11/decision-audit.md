# B1-11 document codecs decision audit

Three independently rewritten packets asked three identical questions over
the B1-11 plan, the codec pipeline bounds and the pinned Bun 1.4.2 format
probe ledger (unsafe integers, duplicates, nonfinite scalars, TOML Temporal
dates, YAML 1.2 core typing, merge keys, shared aliases, cyclic anchors,
multi-document arrays, TOML table roots, JSON5 extensions, JSONL framing
tolerance, prefix-success on trailing garbage, parseChunk shape). All prose
(state, instructions, criteria) differs across packets; question keys,
option keys, identifiers and measured facts are stable. Verified
mechanically before dispatch (0 identical prose fields of 13 comparisons;
20 fact tokens present in every state). Model: `jev-1.13.0` via
`jev-latest`, three HTTP 200 rounds, 3780 input / 407 output tokens total.

## Results

| Question | Round 1 | Round 2 | Round 3 |
|---|---|---|---|
| naming | codec_prefixed 0.73 (conf 0.61) | codec_prefixed 0.81 (conf 0.71) | codec_prefixed 0.59 (conf 0.38) |
| jsonl_strictness | native_whole 0.99 (conf 0.98) | native_whole 0.93 (conf 0.90) | native_whole 1.00 (conf 0.99) |
| toml_dates | project_instant 0.51 (conf 0.26) | project_instant 0.47 (conf 0.21) | reject_dates 0.69 (conf 0.53) |

Requests, responses and SHA metadata: `request-N.json`,
`response-N.json`, `response-N.metadata.json` in this directory.

## Agreement (accepted as advice)

- naming `codec_prefixed` 3/3: one codec domain keeps every decoder
  (`codec::decode_toml`, `codec::decode_yaml`, `codec::decode_json5`,
  JSONL siblings), mirroring the existing `decode_json` generic shape.
  `single_decode` never exceeds 0.02; `format_domains` peaks at 0.39.

## Disagreement (investigated, plan prevails)

- jsonl_strictness picks `native_whole` 3/3 at near-certainty while the
  plan selects `framed_exact`. The packets carried the thin-adapter rule
  but not the acceptance pair that decides the shape: a malformed suffix
  after a valid prefix must fail, and the unsafe-integer regression must
  hold for every format. `Bun.JSONL.parse` reports neither an error nor a
  consumed-byte count, so whole-input parsing cannot fail the suffix, and
  it rounds 9007199254740993 to 9007199254740992, so no native-values path
  can keep Can integers exact: the rounded value is itself a safe integer
  and would project silently corrupted. Even a `parseChunk` read-accounting
  loop only fixes validation, never exactness. The framed path still
  decodes every record through the native-backed exact JSON decoder;
  framing UTF-8 lines is not a document grammar (contract step 3 blesses
  exactly this shape). Fail-closed exactness beats multiline tolerance:
  cross-line records are rejected loudly and documented, never silently
  corrupted. `framed_exact` scores 0.00/0.04/0.00 only because the packets
  under-weighted the acceptance constraints; the contract's explicit
  preference plus the corruption argument settle it.
- toml_dates splits: `project_instant` 0.51 and 0.47 at confidence 0.26
  and 0.21 (coin flips) against `reject_dates` 0.69 in round 3. The schema
  leaf set stated in every packet admits only str, int, float and bool, so
  no Temporal value can project without inventing temporal schema support,
  and the acceptance list explicitly rejects unsupported native dates.
  `reject_dates` is forced by construction and by contract, not taste;
  `project_string` (peaking at 0.44) would additionally hide type
  mismatches by laundering dates into strings.
