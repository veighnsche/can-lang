# Request failure reporting (R12, lane E)

Source: R12; F-R12-01/02/03; C-B (redaction), C-C (identity). Task E06
installs the O2 builtin_auto reporter; E09 owns W5 live failure-hook
acceptance. Nothing here changes the compiler, the catalogue, or the
C-B failure conventions (`tests/failure-conventions/README.md` R1–R9):
the hook owns the projection point R9 hands off.

## 1. Automatic hook

Every unexpected request failure produces one redacted server-side
record and a fixed client 500. No author wiring is needed.

Placement (`runtime/transport/request-report.ts` plus three call sites):

- `serveOuter` (`runtime/platform/server.ts`) is the catch-all: a failed
  completion from handler, adaptation, dispatch, or scope drain, or a
  value that escaped those layers unboxed, is reported before the fixed
  500 is served.
- Dispatch annotates the source: the server action table and
  router-carried actions note `action:<identity>` from the checked
  table; legacy exact routes note `route:<METHOD> <template>` from the
  validated mount. Both are authored metadata, never request bytes.
- The JSON-action adapter (`runtime/platform/action-json.ts`) reports
  handler and adaptation failures itself, because it converts them to
  successful 500 responses the boundary cannot distinguish from
  rejections. An undeclared response leaf has no occurrence of its own
  and is boxed from fixed text.

Report shape (one JSON object per line on stderr):

- `schemaVersion: 1`, `kind: "can.runtime-failure"`, `phase: "request"`.
- Domain channel: `identity` (`can.error.v2:` + declaration identity),
  `error` (declaration name), `typeIdentity`, `occurrence`,
  `payload: "<redacted>"`.
- Standard channel: `category` (finite kind), `occurrence`. There is no
  message field: native messages can carry paths and credentials.
- `correlation`: minted UUID for the native request (C-G vocabulary,
  lane F). No client-supplied value is honored.
- `source`: `http` (pre-dispatch fallback), `action:<identity>`, or
  `route:<METHOD> <template>`.
- `frames`: sanitized Can frames from the verified source maps only.

Redaction follows the entry policy exactly: domain identity, category
plus occurrence, no native messages, paths, query/header/body bytes, or
secrets. Nothing request-derived appears in any field.

Exactly-once discipline (`claimFailureReport` in
`runtime/failure.ts`) is shared with the main, late-owner, and browser
reporters. The late-owner, browser, and request reporters skip claimed
occurrences. The terminal main reporter always delivers and only marks
(existing contract: a repeated main failure still reports every time).

Delivery never breaks the boundary: reporting never throws or rejects,
so a broken sink yields no record but still a fixed 500. The default
sink writes the line to stderr (`console.error`; the module stays
host-free because the JSON-action adapter imports it on a
browser-reachable path). Tests override it with `setRequestReportSink`.

Two paths stay silent by construction:

- Drain auto-close failures are claimed first by the owner `emit`
  path: the record is the existing phase-`cleanup` diagnostic, and the
  request hook dedups. Those records carry occurrence and category but
  no request correlation (see §4).
- The pre-ingress `Bun.serve` error callback has no Can occurrence and
  no request to correlate; it serves a fixed 500 with no report.

## 2. Manual policy for expected failures

Expected failures are handled domain outcomes and client rejections.
They never reach the hook: client rejections (400/404/405/413/415 and
malformed JSON) stay silent automatically, and handled domain outcomes
are ordinary values. Authors deal with them explicitly:

- Match the outcome and map it to a finite declared response (action
  case table or explicit status). Client-visible bytes never carry
  failure internals (R9): no `kind`/`message`/`occurrence_id`
  observations, no payloads, no native text.
- Server-side logging uses `log::write_info`/`log::write_error` with
  fixed text. Safe to log: fixed strings, finite category names you
  wrote, occurrence IDs. Never log: `failure.message` (may carry
  native text for `native_exception`), error payload fields (may carry
  secrets), request bytes (bodies, headers, query), or paths.
- Keep captured error values convertible (R9): match each error into
  its set member; do not stringify failures into logs or responses at
  capture. The hook needs the occurrence intact for the unexpected
  remainder.
- Do not add catch-all wrappers to manufacture reports for unexpected
  failures: the hook covers handler, adaptation, dispatch, and drain
  failures, including a throwing wrapper (a handler fault masked by a
  faulting manual log still yields exactly one record). A wrapper that
  converts an unexpected failure into `ok` hides it from the hook —
  let unexpected failures propagate.

## 3. Record contract for operators

- One JSON line per unexpected failure on stderr; clients always see
  `500 Internal Server Error` with `text/plain` and `nosniff`.
- Join key within a request: `correlation`. Join key across reporters:
  `occurrence` (shared claim; see the drain exception in §1).
- `source` tells which dispatch point was active; `http` means the
  failure preceded dispatch identification (assets, closing scope).
- `frames` may be empty for synthetic runtime origins; that is normal
  and carries no information loss for triage beyond category/identity.

## 4. Handoff to E09/H

- Safe correlation/source reports: §1 shape, `runtime/test/request-report.test.ts`
  (17 legs: handler/adaptation/drain failures, throwing wrapper and
  throwing sink, main/late-owner/browser dedup, redaction of native
  messages/paths/query/header/body/payload secrets, fixed 500s,
  expected-rejection silence).
- Expected-failure guidance: §2 above.
- Known W5 qualification points (E09 owns acceptance):
  drain auto-close failures surface as phase-`cleanup` diagnostics
  without request correlation; the pre-ingress error path has no
  occurrence at all.
- Finding, out of E06 scope: `OwnerDiagnostic.message`
  (`runtime/owner-core.ts`) serializes the standard message, which for
  `native_exception` carries native text. The request hook needed no
  change (it never emits that field), but a follow-up should align the
  late/cleanup diagnostic with the entry redaction policy.
- Ship closure: `runtime/modules.json` inventories
  `transport/request-report.ts`, its `outbound/identity.ts` leaf, and
  the three importer edges. Six pre-existing uninventoried modules
  remain (E01 `transport/request-budget.ts`; F01 ledger/policy set) and
  still fail `TestRuntimeInventoryMatchesBodies` on main.
