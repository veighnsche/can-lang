# E08 R15 remedy record: destructive discard_upload + between-awaits deadline

Task E08, 2026-09-26. Both E07 branches went negative
(`e07-x-r15-1-3.md`, commit `bef41a2d`), so this task implements the
destructive-rename branch and the between-awaits deadline posture —
no preservation/cancel semantics the experiment refuted. Every live
row below is an executed observation against the pinned natives. No
credentials appear here; env names only.

Environment: Bun 1.4.2, MinIO `RELEASE.2025-09-07T16-13-09Z` on
127.0.0.1:9000 (`CAN_TEST_S3_ENDPOINT`, bucket `can-b1-10` via
`CAN_TEST_S3_BUCKET`), per `distribution/provision-local.md`. Live
legs ran under isolated prefixes `s3e08/e08harness/<run>/` (new E08
legs) plus the rebased `s3e07/` and `s3t/` suites; no other lane
prefix was touched. Committed legs: `runtime/test/s3-e08-remedy.test.ts`
(5 legs: E1/E2/E3/D3/D4), rebased `runtime/test/s3.test.ts` (19 legs)
and `runtime/test/s3-e07-remedy.test.ts` (21 legs), reusing the E07
tap (`runtime/test/s3-e07-tap.ts`).

## Scope honesty: S3-protocol, not AWS-real

All qualification is against MinIO over the S3 wire protocol. The
E07 MinIO caveats carry over unchanged (prefix/max-uploads ignored by
`ListMultipartUploads`; `AbortMultipartUpload` 204s even for
never-existing keys, so abort status is never cleanup proof; one
unreproduced post-complete MPU listing in H3). The bucket was left
clean: 0 objects under `s3e08/`, `s3e07/`, `s3t/`, 0 pending uploads
(verified with `aws s3api` after the final run).

## Selected remedy: discard_upload (X-R15-1 negative)

`s3::cancel_upload` is REMOVED from the catalogue and replaced by
`s3::discard_upload` with the destructive contract. The adapter
(`runtime/platform/s3.ts`) keeps the only sink release the pinned API
offers — synchronous `end()` then `delete()` — and now reports the
first failed cleanup await instead of resolving:

- Healthy small/multipart discard resolves `void`; wire shapes are
  `[PUT put-object, DELETE delete-object]` and `[PUT upload-part,
  POST complete-mpu, DELETE delete-object]` (rebased C0 legs).
- The completion transiently overwrites any pre-existing key and is
  visible to concurrent readers (C4: 2/3 runs saw replacement bytes;
  no partial bytes ever; post-discard state missing in all runs).
- Failed completion (E07 F1/F4 injection) now returns
  `s3::service_error {code: InternalError, operation: discard_upload}`;
  the follow-up delete still runs (it carries the delete-coupled
  abort on the part-failure path) and the stranded MPU (2 parts,
  ~6 MB/run shape) is documented for the operator recipe below.
- Failed delete (E07 F2 injection) now returns the same
  `service_error`: the transient completion stays in place
  permanently and the caller learns it.
- Failed small PUT (E07 F5 injection) returns the failed end even
  though the follow-up delete settles the key absent.
- Part-failure (E07 F3 injection) still resolves cleanly: the
  no-op end plus the delete-coupled abort (+ bounded background
  burst) leave no orphan.
- Retained negative (E07 F7): when the delete-coupled abort faults
  but the delete lands, the native swallows the abort failure and
  discard resolves while an (empty-shell) MPU orphans. No in-band
  signal exists; detection needs the operator recipe. This is the
  honest limit of the adapter, kept visible in the leg.
- Terminal guards: `discard_upload` on a non-open handle returns
  `s3::upload_closed {operation: discard_upload, state:
  finished|discarded}`; write/finish after discard fail likewise
  with state `discarded` (D3 + rebased s3.test.ts legs). Scope drain
  discards abandoned open uploads destructively and surfaces a
  failed scrub through `cleanupFailed`.

`discard_upload` emits `[s3::upload_closed, s3::access_denied,
s3::service_error]`. No silent delete keeps the cancel name: E1
asserts `s3::cancel_upload` is absent from the catalogue (lookup
throws; neither name nor identity occurs), and a repo-wide sweep
leaves the string only in the four compiler-slice files below plus
intentional test/history references.

## Deadline posture: between-awaits bound (X-R15-3 negative)

`write_stream` keeps the honestly-documented between-awaits bound:
the deadline fires at the top of each pump iteration only. No
deadline operand promises an abort.

- New D4 leg (tap-observed): a 60 ms/chunk reader against a 100 ms
  deadline returns `s3::service_error {code: timeout, operation:
  write_stream}` after 3 pulls with exactly `[PUT put-object
  forwarded, DELETE delete-object forwarded]` on the wire — the
  timed-out pump scrubs through the shared destructive cleanup —
  and the key reads absent afterwards.
- The unbounded remainder is unchanged from E07 and stays W5-blocked:
  hung `source.read` (H1, zero wire ops), hung writer/flush (H2),
  hung `end()`/completion (H3), hung `stat` (H4) never settle
  client-side; abandoning a writer pins the event loop from creation
  (H5/C5). The catalogue adapter text states the bound ("only
  between awaits, never a hung await").

## Operator orphan guidance (handoff to E09/H)

- Complete-failure ALWAYS orphans (2 parts, ~6 MB/run shape at the
  E07 sizes); abort-failure orphans (parted or shell, incl. the F7
  blind spot above); part-failure orphans ONLY if the delete is
  skipped (the delete-coupled abort otherwise cleans).
- Out-of-band recipe: list uploads UNFILTERED + client-side filter
  by key prefix (MinIO ignores the prefix parameter) + abort each +
  re-list to verify. Never trust the abort status on MinIO.
- Sizing note: orphan cost scales with parts uploaded before the
  failed completion, not with the discard call itself.

## Validation mapping

| Requirement | Legs |
|---|---|
| Name removal + explicit call sites | E1/E2 (catalogue), repo-wide sweep |
| Destructive overwrite, small + multipart | C0-small, C0-multipart (rebased) |
| Transient concurrent-reader windows | C4 (rebased, 2/3 visibility) |
| Cleanup injection: complete-fail | F1, F4 (rebased: service_error + orphan) |
| Cleanup injection: part-fail | F3 (rebased: resolves, no orphan) |
| Cleanup injection: delete-fail / put-fail / abort-fail | F2, F5 (service_error), F7 (blind spot) |
| Terminal guards | D3, s3.test.ts terminal legs (rebased) |
| Between-awaits deadline + timeout scrub | D4 (new), s3.test.ts timeout leg |
| Deadline limit documented, W5-blocked | H1–H5 (unchanged), E3 wording leg |

## Checks

- `bun test runtime/test/s3-e08-remedy.test.ts`: 5/5 pass (live MinIO).
- `bun test runtime/test/s3-e07-remedy.test.ts`: 21/21 pass (rebased).
- `bun test runtime/test/s3.test.ts`: 19/19 pass (rebased).
- `bun run check:runtime`: green (oxlint 0/0, oxfmt clean, tsc clean).
- `make catalogue-check`: green; `go test
  ./compiler/internal/catalogue/`: pass; `go build ./compiler/...`: pass.
- `go test ./compiler/internal/check/`: exactly the two S3 fixture
  tests fail (`TestS3FixtureAdmitsOperationContracts`,
  `TestS3SuppliedFixturesAttachToWireBoundaries`), both at
  `objects.can`'s removed `s3::cancel_upload` call — the handed-off
  compiler slice below. Full-package run confirmed no other failure.
- Pre-existing, out of lane: `runtime/test/catalogue.test.ts`
  expects 288 operations but the mirror carries 290; verified failing
  identically at `bef41a2d` before any E08 change (drift from an
  earlier catalogue admission). Not touched; flagged for the owner.

## Handoffs

- E09/H (W5): S3 legs run on final adapters — destructive/unknown
  evidence above; the between-awaits deadline bound FAILS
  bounded-behavior acceptance, so the W5 S3 deadline legs stay
  BLOCKED pending explicit user scope return (per F-R15-03/04).
- Coordinator (compiler slice, exact): (1)
  `compiler/internal/emit/runtime_s3.go:30` →
  `"can.std.s3@1::discard_upload": "$canS3.discardUpload"`; (2)
  `compiler/internal/check/self_tail.go:46` →
  `"can.std.s3@1::discard_upload"` keeping
  `"drain-owned value (s3 upload)"`; (3)
  `compiler/internal/check/s3_test.go:127` → discard_upload with
  errors `[upload_closed, access_denied, service_error]`; (4)
  `compiler/testdata/current/s3/objects.can` → rewrite `abort_upload`
  (line 429 + call site 727, currently failing check) onto
  `s3::discard_upload` with arms for the three emitted errors and an
  honest destructive doc comment. No invoice patch needed (no `s3::`
  use in `examples/invoice/`).
- AWS-real flags (unchanged from E07): post-complete listing ghosts,
  abort-nonexistent status, background-burst counts, WAN retry
  backoff, AWS transient-visibility window.
