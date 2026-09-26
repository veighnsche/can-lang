# Paired builds: generation handshake, retention, rollout (C-H)

H-owned interface contract for paired server/browser deployment
(H06; R13/R02/R03; interface C-H). This document fixes the wire
vocabulary, the fail-closed rules, the retention/CAS/lock semantics,
the rollout/rollback + credential recipe, and the E/C/A handoff slices.
E, C, and A implement their slices against these spellings; unilateral
drift is a contract break — counter-proposals come back through H.

Design advice: [H06 Jev findings](../docs/syntax-taste/evidence/2026-09-26/h06/jev/findings.md)
(page-embedded acquisition, manifest-startup typed comparison,
rebuild-stable rollback; unanimous, treated as advice, not proof).

## 1. Identities

- The **server generation** is the server generation manifest `buildID`:
  a 64-hex `can-output-generation-v1` content hash, always equal to its
  `dist/builds/<id>` directory name (`compiler/internal/driver/artifacts.go`).
- The **browser build** is the `can-browser-bundle-v1` hash over the
  published browser files, recorded in `browser/manifest.json`.
- The pairing record (`can.browser-pairing`) is staged at
  `browser/pairing.json` **inside** the server generation and hash-bound
  by the server manifest. It therefore cannot quote the server `buildID`
  (that would hash itself); the binding is structural: the record names
  the exact browser build, and the server generation directory names
  itself. The build report carries both values together.

## 2. Handshake contract

### 2.1 Browser learns the generation from the page

Every `text/html` document served by a paired build carries its serving
generation in the paired script tag:

```html
<script type="module" src="/__can/assets/<digest>.js" data-can-generation="<server-generation>"></script>
```

The `data-can-generation` value is the serving generation's `buildID`
(64-hex). It is spliced by the same server pass that splices the paired
script (`pairDocument` in `runtime/platform/server.ts`; E slice).
Fragments without a body marker, non-HTML media, HEAD responses, and
unpaired builds keep exact bytes, as today. A page rendered by
generation N keeps saying N forever: an old page can never claim to be
new, with no extra request and no cache race.

### 2.2 Browser sends the generation on every action

Every action request — JSON and form actions plus captured reads
(C-E × C-H: captured reads ship through the same paired-build identity)
— carries the generation read from the live document:

```text
can-generation: <server-generation>
```

Missing slot (unpaired markup, foreign document) means the request
carries no header; that also fails closed (2.3). The send site is the
browser action client (`fetchJsonAction` in
`runtime/platform/action-json.ts`; E slice, A emission if the slot read
needs generated support).

### 2.3 Server pins its generation and answers typed mismatch

At startup the server reads the `buildID` from the `manifest.json` of
the generation it runs from, once, and compares it against the
`can-generation` value on every action request **before any action
logic runs**. The check is symmetric string inequality (generations are
unordered content hashes): rollback mismatches refuse exactly like
rollout mismatches.

Missing, malformed (not 64-hex), or unequal values fail closed with the
typed mismatch — never silent execution, never a log-and-serve:

- Status: `409`
- Content type: `application/json`
- Body (exact shape):

```json
{"schemaVersion":1,"kind":"can.generation-mismatch","serverGeneration":"<id>"}
```

The client maps this response — and only this response — to a distinct
`generation_mismatch` outcome lowered into the declared per-action
failure bound (same lowering family as `transport`/`timeout` in
`createJsonActionFetch`; never a domain case, never
`unexpected_status`). Both sides are E slices; a new failure identity
needs the A catalogue/emission slice.

### 2.4 App presents a blocking refresh

On `generation_mismatch` the app presents a blocking refresh prompt
(C slice, both apps). The old action is not retried against the new
server; the prompt resolves only through a refresh that loads the
current generation's document (and its new `data-can-generation`).
No silent reload: the user sees that the app moved under them.

### 2.5 Fail-closed rules (normative)

1. Old app logic against a new server is the defined `409`
   `can.generation-mismatch` error, never interop.
2. New app logic against an old server (post-rollback page) fails the
   same way: any inequality mismatches.
3. Asset routes (`/__can/assets/…`, `/__can/project/…`) are content
   addressed and carry no generation check; the handshake gates action
   logic, not bytes.
4. Old digest URLs stay servable per the retention rule (3.2) while
   old logic errors per 2.3: bytes are retained, behavior is refused.

## 3. Retained pairing semantics

### 3.1 Pairing and shared locks

- `verifyBrowserManifest` (H, `compiler/internal/driver/pairing.go`)
  verifies build identity, digest-bound routes, entry, table,
  script/source-map pairing, generation binding, the post-bundle audit,
  and the shared locked snapshot before anything stages or publishes.
- `bindSharedLock`: every locked instance shared by the browser and
  server graphs pins identical lineage and content. The two roots may
  differ; only their shared snapshot must agree.
- Publication re-reads every browser input immediately before atomic
  selection (`reread`); any change fails the build for a fresh retry.

### 3.2 Seven-day digest retention

- Replaced digest URLs stay servable for at least seven days past
  replacement; a route exactly at the bound still serves
  (`assetRetained`, driver + `runtime/platform/assets.ts`).
- The `dist/assets.json` ledger binds each retained route to its
  replacement moment; `dist/assets/<digest><ext>` holds the bytes.
  Generations stay timestamp-free; only dist metadata carries
  wall-clock time.
- Publication is atomic across crashes (pending record + ledger replay;
  `selectCurrentWithAssets`, `Prune` recovers first).

### 3.3 CAS staging and collection

- Each distinct byte string is stored once under `dist/cas/<sha256>`;
  generations link to it (hardlink, read-only `0400` content) instead
  of copying. In-place mutation fails loudly; honest tampering replaces
  the path (unlink + recreate), which breaks the link.
- Manifests and metadata stay unique per-build `0600` writes and are
  never hardlink-shared.
- `Prune` deletes non-current generations (never current) and sweeps
  store entries no generation, pending publication, or ledger row
  references; unknown store files fail closed for operator inspection.
- No full-copy staging: identical bytes always share an inode (falling
  back to a private read-only copy only when linking is unavailable).
- Mirrors: read-only test mirrors may link; mutable workspaces always
  take full copies (`copyLinkTree` vs `copyDir` precedent in
  `tests/integration/stdlib_test.go`).

## 4. Rollout/rollback + credential recipe (operator)

Applies to paired deploys on any host, including the UP25 x86 run
(H12 qualifies; this recipe defines the steps).

1. **Build the browser**: `canlc build --target browser PROJECT`
   produces the browser generation and `browser/manifest.json`.
2. **Build the paired server**:
   `canlc build --browser-manifest <browser>/browser/manifest.json PROJECT`.
   Pairing verification, test assertion, atomic selection, and retention
   apply in one command; any failure preserves the prior current.
3. **Serve the selected generation**: run the server from the selected
   `dist/builds/<id>` tree. Health is the served app responding with
   its generation embedded per 2.1.
4. **Rollout** is step 2 with new inputs: selection is atomic, replaced
   assets stay servable for seven days, and open old pages fail closed
   into the blocking refresh (2.4).
5. **Rollback** rebuilds the prior source and selects the result through
   the same path: stable build IDs reproduce the byte-identical
   generation, so rollback needs no retained stale trees and no prune
   change. Retention covers replaced routes in both directions.
6. **Credentials are env-only**: values never appear in the repo, the
   register, or this recipe — env names only, with secrets in process
   env or `$CAN_PROVISION_DIR` (mode 700/600) per
   [provision-local.md](provision-local.md). The service unit passes
   the same env names through; provisioning records grants, never
   values.

## 5. Lane handoff slices

### E — transport/server mismatch (applies against 2.1–2.3)

- `runtime/platform/server.ts`: splice `data-can-generation` in
  `pairDocument`; pin the serving generation from the generation
  `manifest.json` at startup; compare before action dispatch; answer
  the exact `409` shape on missing/malformed/unequal.
- `runtime/platform/action-json.ts`: send `can-generation` from the
  live document slot in `fetchJsonAction` (JSON + captured reads; form
  and htmx paths per the C-E wire contract); decode the exact `409`
  shape into a distinct `generation_mismatch` outcome in
  `createJsonActionFetch`.
- Tests: old-page/new-server refusal, missing/malformed header refusal,
  rollback-direction refusal, new-page success, byte-exact `409` shape,
  asset routes unaffected.

### C — app prompt (applies against 2.4)

- Both apps: map `generation_mismatch` to the blocking refresh prompt;
  no retry of the refused action, no silent reload.
- Tests: prompt appears on mismatch, refresh loads the current
  generation, no action is double-applied across the refresh.

### A — generated emission (only if required)

- Only if the E/C slices need compiler support (e.g. a new failure
  identity in the catalogue, the document slot in emitted markup, or
  the slot read in the emitted browser client). No speculative
  emission: hand back through H if the runtime-only shape suffices.

## 6. Consumption by C06/H10/H12

- **Immutable candidate builds**: any `canlc build` output is a
  content-addressed, read-only generation; leases (`AcquireGeneration`)
  qualify a candidate without selecting it.
- **C-H build contract**: `go test ./compiler/internal/driver/ -run
  'TestPairedHandshakeIdentity|TestStageSharesIdenticalBytes|TestBreakLinkTamperIsolatesGenerations|TestPruneCollectsUnreferencedStore|TestMetadataWritesAreUnique'`
  — pairing identity chain, retention, CAS/GC, tamper legs (H, green
  before handoff).
- **C06** runs W1 plus generation-policy legs on final controls/actions
  once the E/C/A slices land; **H10** joins the owner/consumer slices;
  **H12** executes this recipe on native x86 (network-denied smoke,
  old-app mismatch/refresh, retained assets, rollout/rollback, CAS
  GC/read-only safety).
