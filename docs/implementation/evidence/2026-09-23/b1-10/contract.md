# B1-10 S3 object storage contract confirmation

Target: Bun 1.4.2 `S3Client`, `S3File`, `NetworkSink` writer.
Every row below is an executed observation (probe, unit, staged or
integration run) unless marked docs. Catalogue additions plus one
catalogue bound method (`s3::describe` on `s3::presigned`); no new
grammar. Surface selection audited in
`docs/bun-integration/asap/evidence/consultations-b1-10/decision-audit.md`.

## Client and keys

| Aspect | Contract |
|---|---|
| Construction | `s3::client_open` validates endpoint, region, bucket and credentials locally and binds a native client; no I/O, assertion real |
| Explicitness | endpoint, region, bucket, access key and secret key are all required; empty values fail `s3::invalid_config{endpoint,region,bucket,credentials}` |
| Endpoint | must parse as http(s); anything else fails `{endpoint}` |
| Secrecy | credentials never enter diagnostics, failures, snapshots or reports; only the operation name crosses on faults |
| Keys | slash, space and unicode are key data, never traversal; keys round-trip verbatim through write, read, stat and list; empty keys fail `{key}` |

## Reads

| Aspect | Contract |
|---|---|
| Bounded read | `s3::read_bytes` stats first and fails `s3::over_limit{limit,size}` without downloading when the object exceeds `max_bytes` (1..64MiB, else `{limit}`) |
| Range | `s3::read_range` downloads one exact window; length 0 answers empty without a wire call (native `slice(n,n)` yields one byte); negative offset or length fails `{offset,length}` |
| Stream | `s3::read_stream` stats eagerly (missing, denied and over-limit surface at open), then serves a B1-05 byte reader; cancelling the reader cancels the download |
| Copies | every read returns a fresh copy; no caller-visible aliasing of native buffers |

## Writes

| Aspect | Contract |
|---|---|
| Single | `s3::write_bytes` stores one object with an optional content type, then stats it; the result is immutable `s3::metadata` |
| Content type | advisory: stored verbatim for binary types, gains a `charset` suffix for text; a Blob-supplied type is ignored natively |
| Stream pump | `s3::write_stream` pumps a byte reader into a multipart upload under byte and deadline budgets; reader failure cancels the upload and propagates `stream::read_failed`/`stream::cancelled`; native writer rejections fail `s3::service_error` after cancelling |
| Non-bytes reader | pumping a text or event reader fails `{reader}` without creating an upload |

## Metadata and listing

| Aspect | Contract |
|---|---|
| Metadata | `s3::stat` returns immutable `s3::metadata{size,etag,content_type,last_modified}`; etags stay quoted verbatim with a `-N` suffix after multipart completion |
| Timestamps | `last_modified` is an opaque `time::instant` converted at the boundary; instants never leak epoch arithmetic into callers |
| Exists | `s3::exists` answers false only for absent keys; denied credentials and service faults still fail |
| Delete | `s3::delete` is idempotent; deleting a missing key succeeds |
| Paging | `s3::list` returns one page: immutable entries, grouped prefixes, truncated flag and an opaque `s3::continuation`; the adapter never collects a whole bucket |
| Limits | page limits pass to `maxKeys` verbatim (limit < 1 fails `{max_keys}`); the service clamps above 1000 (docs) |

## Presigned access

| Aspect | Contract |
|---|---|
| Minting | `s3::presign` signs locally with a closed `s3::method` (get, put, delete, head; unsupported methods are unrepresentable) and `expires_in` 1..604800 (`{expires}` outside; ceiling is SigV4 docs, floor is native) |
| Sensitivity | the URL stays inside the opaque `s3::presigned` handle; only the bound `describe` method reveals `s3::presigned_info{url,method,expires_at}`; reports never print raw URLs |
| Negatives | method swaps and forged signatures fail closed (403, tampered PUT stores nothing); signed GET on a missing key answers 404; signed DELETE round-trips 204; signed HEAD answers 200 with length |
| Gap | required headers are unrepresentable: the native options carry no header binding and a mismatched Content-Type still uploads; recorded as a target gap, never emulated |

## Multipart uploads

| Aspect | Contract |
|---|---|
| Handle | `s3::begin_upload` opens a local handle with optional content type and part size (5MiB..5GiB when given, else the native default; outside fails `{part_size}`) |
| Writes | `s3::upload_write` appends one chunk and reports accepted bytes; the adapter guards the native silent drop after end |
| Finish | `s3::upload_finish` completes (native `end` resolves the byte count, not an etag) and returns fresh `s3::metadata` |
| Cancel | `s3::cancel_upload` retires the handle and abandons the sink without close or end, so the key never materializes; cancellation never deletes the key and never closes (native `close` async-completes single-part and multipart uploads alike) |
| Terminal | finished and cancelled handles reject every further operation with `s3::upload_closed{operation,state}`; scope release cancels silently |
| Orphans | unreferenced server-side parts age out under bucket lifecycle rules, documented, not adapter-enforced (no native abort exists) |

## Streams and errors

| Aspect | Contract |
|---|---|
| Reuse | transfers ride B1-05 reader cells; no new stream core (`runtime/transport/stream.ts` in the plan is the existing B1-05 module) |
| Mapping | `NoSuchKey` fails `s3::missing_key{key}`; `UnknownError` fails `s3::access_denied{operation}`; refused endpoints and other native faults fail `s3::service_error{code,operation}`; deadline fire fails `{timeout}` |
| Ambiguity | a missing bucket also reads `NoSuchKey`, and non-auth faults may share `UnknownError`; both collapse per the qualified matrix and are documented, not distinguished |
| Unsafe | no unsafe analogue exists: all byte transfers copy; there is no zero-copy path to keep memory-affect-free |

## Typing note

The repo carries no `.d.ts` convention; the B1-10.01 typed
surface is structural TypeScript in `runtime/platform/s3.ts` that
mirrors the probed `S3Client`/`S3File`/`NetworkSink` shapes, the
same pattern as the WebSocket structural socket types.

## Error table

| Error | Fields | Meaning |
|---|---|---|
| s3::invalid_config (1343) | reason | endpoint, region, bucket, credentials, key, expires, part_size, limit, offset, length, max_bytes, deadline, max_keys, delimiter, content_type, reader |
| s3::missing_key (1344) | key | stat/read of an absent key (a missing bucket reads the same) |
| s3::access_denied (1345) | operation | native UnknownError on the named operation |
| s3::service_error (1346) | code, operation | native code (ConnectionRefused, timeout, io_error) on the named operation |
| s3::upload_closed (1347) | operation, state | use of a finished or cancelled upload handle |
| s3::over_limit (1348) | limit, size | object or transfer exceeds the declared byte budget |
