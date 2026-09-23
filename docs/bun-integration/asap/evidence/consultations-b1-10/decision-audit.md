# B1-10 S3 decision audit

Three independently rewritten packets asked three identical questions over
the B1-10 plan, the filetree bounds and the pinned Bun 1.4.2 S3 probe
ledger (`s3-native-results.json`: methods, roundtrip, write_inputs, range,
list, list_continuation, slice_edges, presign, presign_bounds, multipart,
multipart_partsize_floor, writer_after_end, missing, auth_config, keys).
All prose (state, instructions, criteria) differs across packets; question
keys, option keys, identifiers and measured facts are stable. Verified
mechanically before dispatch (0 identical prose fields of 39
comparisons; 20 fact tokens present in every state). Model:
`jev-1.13.0` via `jev-latest`, three HTTP 200 rounds, 4041 input /
393 output tokens total.

## Results

| Question | Round 1 | Round 2 | Round 3 |
|---|---|---|---|
| cancel | adapter_cancel 0.83 (conf 0.74) | drop_only 0.52 (conf 0.28) | drop_only 0.90 (conf 0.85) |
| signed_headers | method_negatives 0.99 (conf 0.99) | method_negatives 1.00 (conf 1.00) | method_negatives 0.97 (conf 0.95) |
| timestamps | opaque_instant 0.89 (conf 0.84) | opaque_instant 0.84 (conf 0.76) | opaque_instant 0.96 (conf 0.93) |

Requests, responses and SHA metadata: `request-N.json`,
`response-N.json`, `response-N.metadata.json` in this directory.

## Agreement (accepted as advice)

- signed_headers `method_negatives` 3/3 at near-certainty: the
  contract pins the two enforced failure modes (method mismatch and
  signature forgery, both failing closed) while required headers stay
  unrepresentable and are recorded as a target gap. `custom_signer`
  scores 0.00 in every round and `emulate_via_type` never exceeds 0.03.
- timestamps `opaque_instant` 3/3: stat and list entries expose
  last-modified time as `time::instant` handles converted with
  `time::instant_from_epoch_millis`. `iso_string` never exceeds 0.02;
  `epoch_millis` peaks at 0.15.
- Unanimous rejections: `delete_on_cancel` scores 0.00 in every round
  (cancellation must never delete the destination key), and
  `custom_signer` scores 0.00 (no adapter-side SigV4).

## Disagreement (investigated, plan prevails)

- cancel splits: `adapter_cancel` 0.83 in round 1, `drop_only` 0.52
  at confidence 0.28 (a coin flip) in round 2, `drop_only` 0.90 in
  round 3. The packets carried the thin-adapter rule and the shared
  wire outcome (the key never materializes either way) but not the
  B1-10.06 acceptance pair that decides the shape: cancelled uploads
  are an observable outcome, and no handle may be reused after
  terminal state. `drop_only` defines neither a cancelled outcome nor
  a terminal state, so it cannot satisfy the stated acceptance; it
  also leaves the probed silent-drop behavior (writes after end
  vanish without error) unguarded, where the Completion discipline
  requires a declared state error. Selected: `adapter_cancel` per
  plan — explicit cancel retires the handle, releases the sink
  without end, and rejects later use; orphaned server-side parts age
  out under bucket lifecycle rules, documented, not adapter-enforced.

## Selected record

Upload cancellation is an explicit adapter-side terminal transition
with no key deletion; presigned-URL contracts assert method and
signature negatives only, with required headers filed as a target
gap; object timestamps cross as opaque `time::instant` values. No
B1-12 carryover arises: the markdown probe row already exists in the
native ledger and S3 shares no surface with it.
