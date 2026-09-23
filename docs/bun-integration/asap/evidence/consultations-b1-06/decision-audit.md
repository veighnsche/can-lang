# B1-06 streaming-architecture decision audit

Three independently rewritten packets asked four identical questions over
the pinned HTTP/stream facts and Bun 1.4.2 probe ledger. All prose
(state, instructions, criteria) differs across packets; question keys,
option keys, identifiers and measured facts are stable. Verified
mechanically before dispatch (0 identical prose fields of 51
comparisons). Model: `jev-1.13.0` via `jev-latest`, three HTTP 200
rounds, 4085 input / 538 output tokens total.

## Results

| Question | Round 1 | Round 2 | Round 3 |
|---|---|---|---|
| body_mode | eager_only 0.44 (conf 0.17) | route_marked 0.76 (conf 0.64) | route_marked 0.83 (conf 0.73) |
| consumption | mode_exclusive 0.68 (conf 0.52) | buffered_repeatable 0.79 (conf 0.68) | buffered_repeatable 0.79 (conf 0.69) |
| response_stream | writer_pair 0.92 (conf 0.87) | writer_pair 0.95 (conf 0.94) | writer_pair 0.89 (conf 0.84) |
| sse_shape | raw_stream 0.92 (conf 0.88) | record_send 0.51 (conf 0.27) | record_send 0.90 (conf 0.85) |

Requests, responses and SHA metadata: `request-N.json`,
`response-N.json`, `response-N.metadata.json` in this directory.

## Agreement (accepted as advice)

- response_stream `writer_pair` 3/3: pending response plus a
  once-vended writer keeps production in caller code beside the B1-05
  pull surface instead of hiding the loop in a fold or restricting
  responses to prebuilt readers.
- consumption `buffered_repeatable` 2/3: buffered reads stay
  repeatable, the body reader opens once, buffered-after-live fails,
  reader-after-buffered replays retained bytes. Round 1 preferred
  `mode_exclusive` (reject reader-after-buffered); the retained choice
  loses no information because the bytes are already held, and it lets
  one handler body serve both route kinds.
- body_mode `route_marked` 2/3: unmarked routes keep drain-first
  ingress with the pre-dispatch fixed 413; marked routes defer every
  byte to one live reader. Round 1 split `eager_only` 0.44 /
  `route_marked` 0.38 at confidence 0.17; always-lazy never exceeds
  0.18.

## Disagreement (investigated)

- Round 1 `eager_only` on body_mode cannot satisfy H-HTTP-06-04: real
  connection termination and chunk order are unobservable from a
  prebuffered copy, and the B1-06 title promises incremental bodies.
  The packets carried this requirement, so the dissent is judged low
  confidence rather than smuggled evidence. Retained: `route_marked`.
- Round 1 `raw_stream` on sse_shape (0.92) contradicts B1-06.05
  acceptance: SSE encoding violations must be rejected with regression
  tests over malformed framing. A raw byte surface has no adapter
  frame to validate, so violations pass through uncaught. Rounds 2-3
  select `record_send`. Retained: `record_send` with validated framing
  and a comment operation for heartbeat lines.

## Retained selections

`route_marked`, `buffered_repeatable`, `writer_pair`, `record_send`.
These are engineering judgments; the agreement above is advice, and
acceptance evidence decides.
