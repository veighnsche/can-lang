# B1 module layout and file boundaries

This is the required implementation layout for the Bun milestone. It takes precedence over the earlier capability tables where they name an existing large file as an integration point. Those entries identify where behavior currently lives; they are **not instructions to append each new feature there**.

There is no universally perfect tree. This layout gives concrete ownership and reviewable boundaries for this repository. Implement needed modules as working slices, without creating empty placeholder files, one-function forwarding files or a general plugin framework.

## Architecture prerequisite

After integrating the required LF work or establishing the implementation checkout, record its base commit. Before adding B1 behavior, extract the emitter assembly responsibilities below and the SQL responsibilities when beginning B1-02. Preserve behavior with focused existing tests. Do not refactor unrelated LF work or touch the active implementer's checkout.

During this audit, `compiler/internal/emit/program.go` had 883 lines and mixed operation maps, imports and module assembly. It should become a small orchestration entry point, not collect thirteen more features. Line counts are observations of a moving checkout, not immutable baselines.

## Compiler structure

Keep these files in their existing Go packages unless a concrete dependency boundary warrants a package. File separation alone does not require extra exported APIs.

```text
compiler/internal/
  emit/
    program.go                  # entry points and ordered assembly only
    program_state.go            # emitted shared initialization/state module
    program_modules.go          # authored-module assembly
    program_entry.go            # executable and assertion entry artifacts
    runtime_bindings.go         # small composition function, duplicate-key checks
    runtime_core.go             # common runtime imports/bindings
    runtime_collections.go      # existing array/map/set operations
    runtime_ai.go               # existing native AI wiring
    runtime_files.go            # B1-01 filesystem/path/glob binding
    runtime_sql.go              # SQL runtime imports/configuration, B1-02/03
    runtime_process.go          # B1-04 process binding
    runtime_streams.go          # B1-05 stream binding
    runtime_http.go             # B1-06 HTTP/router/server binding
    runtime_websocket.go        # B1-07 WebSocket binding
    runtime_crypto.go           # B1-08 key/password/crypto binding
    runtime_cookies.go          # B1-09 cookie/CSRF binding
    runtime_s3.go               # B1-10 S3 binding
    runtime_formats.go          # B1-11 document codec binding
    runtime_markdown.go         # B1-12 Markdown binding
    runtime_utilities.go        # B1-13 URL/text/bytes/time binding
    sql.go                      # checked SQL call lowering, not SQL parsing
    streams.go                  # checked stream lowering, only if needed
    websocket.go                # selected native session lowering, if needed
  project/
    manifest.go                 # manifest envelope and delegation
    manifest_sql.go             # SQL descriptor/config decoding
  sql/
    dialect.go                  # dialect identity and backend selection
    parser.go                   # small common parse entry/type contracts
    postgres.go                 # current pinned libpg_query adaptation
    sqlite.go                   # selected SQLite grammar adapter
    mysql.go                    # selected MySQL grammar adapter
    descriptors.go              # compose checked descriptor, no dialect lexer
    parameters.go               # parameter spans/segments from parser evidence
    cardinality.go              # admitted statement/limit/cardinality rules
  check/
    sql.go                      # SQL checker orchestration
    sql_descriptors.go          # manifest-to-checked descriptor validation
    sql_specialization.go       # generic query specialization
    sql_sites.go                # pool/descriptor/call-site agreement
    streams.go                  # stream type/handler/ownership admission
    websocket.go                # domain-specific session admission, if selected
    codec_formats.go            # per-format schema admission
  ir/
    sql.go                      # inert checked SQL representation
    streams.go                  # inert checked stream representation, if needed
    websocket.go                # checked session representation, if selected
    resources.go                # shared capture/resource evidence only
  syntax/
    native.go                   # existing native form dispatch
    streams.go                  # selected stream/event grammar only
    websocket.go                # only if a separate WebSocket form is selected
    lifecycle_format.go         # formatting selected new lifecycle forms
```

Some proposed files may not be needed: library-only admission can use existing catalogue machinery, and the event comparison may select one form rather than separate stream/WebSocket grammar. Document the selected layout, remove inapplicable entries, and do not create empty modules to match a picture. Conversely, choosing fewer primitives does not authorize one large runtime or checker file.

### Binding interface and dependency direction

Use a small package-private binding contribution type containing operation-target mappings, native imports and any ordered initialization fragments. Feature files construct their own contribution. `runtime_bindings.go` combines contributions in explicit deterministic order, detects duplicate identities/conflicting imports, and supplies the result to program assembly. Preserve dependencies between domain setup, adapter creation and application initialization; do not sort initialization statements blindly.

This is a static composition helper, not a runtime registration system. Do not add reflection, init-time global registries or dynamic loading. Generated TypeScript still calls concrete native adapters.

The syntax layer parses source only. The checker produces typed evidence. IR stores it. The emitter consumes it without reparsing/rechecking source. Runtime modules operate on native JS/Bun values under Can contracts. A SQL grammar adapter must not import the checker or emitter. SQL service clients belong in the runtime, not the compiler.

### Files that must not accumulate feature bodies

- `emit/program.go`: program orchestration only. No new per-feature operation map, factory body or native API implementation.
- `check/program.go`, `check/expressions.go`, `check/native.go`: small dispatch hooks only; feature checking belongs in its named module.
- `check/wrap.go`: no B1 feature logic. Reuse the finalized wrapper contract through its interface.
- `project/manifest.go`: envelope parsing and delegation; SQL details move to `manifest_sql.go`.
- `catalogue/catalogue.go`: generic catalogue validation; no per-capability operational switches. Split generic validation by actual responsibility if extension requires it.
- `runtime/owner.ts`: shared owner/resource protocol only; no SQL, socket, process or file-specific lifecycle implementations.
- `runtime/assert/provider.ts`: provider protocol and dispatch only; do not put thirteen fixture simulators in it.

## Runtime structure

Feature modules own native behavior. Existing monolithic runtime files should be moved into the named area when that capability is implemented; update imports/module declarations and remove obsolete paths instead of retaining compatibility re-export files.

```text
runtime/
  platform/
    files/
      read.ts                   # bounded text/bytes read and decoding
      write.ts                  # write/copy/move/exclusive-create behavior
      directory.ts              # stat/list/mkdir/remove semantics
      path.ts                   # path/URL conversion and resolution
      glob.ts                   # native enumeration and bounds
      errors.ts                 # verified filesystem error translation
    sql/
      pool.ts                   # client creation/lease/close orchestration
      config.ts                 # dialect-specific native option validation
      descriptor.ts             # inert template materialization
      values.ts                 # shared immutable binding/result projection
      postgres.ts               # PostgreSQL-specific mapping/errors
      sqlite.ts                 # SQLite-specific mapping/options/errors
      mysql.ts                  # MySQL-specific mapping/options/errors
      transaction.ts            # shared owned transaction orchestration
      errors.ts                 # common SQL failure construction/provenance
    process/
      run.ts                    # spawn and terminal-result orchestration
      pipes.ts                  # concurrent stdin/stdout/stderr and budgets
      shutdown.ts               # signals, grace, reaping, qualified tree policy
      errors.ts                 # native process error translation
    http.ts                     # existing typed request/response façade
    router.ts                   # route matching
    server.ts                   # server ownership/start/stop
    form.ts                     # bounded form handling
    multipart.ts                # only qualified native incremental path
    sse.ts                      # SSE framing and stream adaptation
    websocket/
      client.ts                 # native WebSocket client adapter
      server.ts                 # Bun upgrade/hooks and server session creation
      session.ts                # typed state/events and checked dispatch
      send.ts                   # side-specific backpressure/queue behavior
      errors.ts                 # session failure and close projection
    crypto/
      password.ts               # Bun.password and cost policy
      hash.ts                   # native digest/HMAC
      key.ts                    # opaque key storage/usages/import/export
      cipher.ts                 # native encryption/decryption
      signature.ts              # qualified native sign/verify
    cookies.ts                  # native cookie projection/serialization
    csrf.ts                     # explicit-session native token operations
    s3/
      client.ts                 # configuration and private credentials
      objects.ts                # read/write/stat/delete adapters
      listing.ts                # paging and continuation projection
      signing.ts                # native presigning and redaction
      upload.ts                 # qualified native streamed/multipart lifecycle
      errors.ts                 # native service error translation
    markdown/
      render.ts                 # ordinary-string native rendering
      safe.ts                   # qualified private safe-HTML bridge
      callbacks.ts              # native callback adaptation/staging
    url.ts                      # URL/query values
    datetime.ts                 # explicit instant/locale/zone conversions
  transport/
    stream/
      readable.ts               # owned reader and consumption
      writable.ts               # owned writer and backpressure
      lifecycle.ts              # terminal/cancel/cleanup protocol
      budget.ts                 # byte/item/queue limits, not another owner
  codec/
    projection.ts               # genuinely shared typed native-value projection
    formats.ts                  # small format dispatch only
    toml.ts                     # native TOML and its scalar policy
    yaml.ts                     # native YAML, aliases/dates/duplicate policy
    json5.ts                    # native JSON5 policy
    jsonl.ts                    # strict bounded framing/consumption
    json.ts                     # existing exact JSON behavior remains intact
  text.ts                       # existing basic text operations
  text_regex.ts                 # native RegExp, captures and offset policy
  bytes.ts                      # existing immutable bytes representation
  bytes_encoding.ts             # additional strict native encoding adapters
  assert/
    provider.ts                 # shared provider protocol only
    fixtures_files.ts           # per-capability fixture matching/adaptation
    fixtures_process.ts
    fixtures_streams.ts
    fixtures_sql.ts
    fixtures_http.ts
    fixtures_websocket.ts
    fixtures_s3.ts
```

Create fixture modules only for real feature-specific matching/dispatch logic. Shared fixture schemas/helpers stay shared; do not copy the provider engine into each file. Crypto/cookies/formats/Markdown pure computation executes natively in tests and does not require a fake runtime per feature.

`runtime/modules.json` remains the authoritative explicit import graph. Register every actual module/import. Avoid barrel `index.ts` files added solely to hide imports; the emitter can import the concrete factory or operation module.

## Capability ownership summary

| B1 | Go boundary | Runtime area | Tests grouped by responsibility |
|---|---|---|---|
| 01 files | `emit/runtime_files.go` | `platform/files/` | read, mutations, directory, glob, errors |
| 02 SQLite/shared SQL | `project/manifest_sql.go`, `sql/`, `check/sql_*.go`, `emit/runtime_sql.go` | `platform/sql/` | dialect corpus, bindings, SQLite lifecycle |
| 03 MySQL | `sql/mysql.go`, shared descriptor/site checks | `platform/sql/mysql.ts` plus shared modules | MySQL values, errors, transactions, real service |
| 04 processes | `emit/runtime_process.go` | `platform/process/` | arguments, pipes, deadlines, shutdown/tree |
| 05 streams | selected syntax/check/IR/emit stream files | `transport/stream/` | backpressure, terminal races, ownership, fixtures |
| 06 HTTP | `emit/runtime_http.go`, focused existing HTTP checker | HTTP/router/server/form/multipart/SSE modules | method/header, bodies, multipart, TLS, publication |
| 07 sockets | `emit/runtime_websocket.go`, selected session checker/IR | `platform/websocket/` | client, server, send/backpressure, session lifecycle |
| 08 crypto | `emit/runtime_crypto.go`, catalogue types | `platform/crypto/` | passwords, keys, vectors, encryption, signatures |
| 09 cookies/CSRF | `emit/runtime_cookies.go` | cookies/CSRF modules | cookie policy, token/session policy |
| 10 S3 | `emit/runtime_s3.go` | `platform/s3/` | objects, listing, signing, upload cleanup |
| 11 formats | `check/codec_formats.go`, `emit/runtime_formats.go` | `codec/` per format | separate format semantics, JSONL framing |
| 12 Markdown | `emit/runtime_markdown.go` | `platform/markdown/` | ordinary render, callback bridge, hostile safe HTML |
| 13 utilities | `emit/runtime_utilities.go` | URL/datetime/text-regex/byte-encoding modules | URL, encodings, regex, dates/timezones |

Adjacent Go `_test.go` files mirror the responsibility being tested. Split integration suites by these groups when they grow; `tests/integration/stdlib_test.go` must not become a bucket for all B1 cases. Keep shared helpers small and behavior-neutral. Do not hide feature behavior in giant fixture-generating test helpers.

## Size and review rules

- Aim for 150–300 lines of hand-written production code per cohesive module. A tiny module is fine when its boundary is real.
- A new hand-written Go/TypeScript production file above **400 physical lines** fails the B1 size check. Split by responsibility before continuing. Do not remove whitespace or compress code to evade the limit.
- Existing oversized files are not an excuse for further growth: a touched oversized file may not increase in physical lines. For emitter assembly, additionally perform the required responsibility extraction; a line-count pass alone is insufficient.
- Hand-written test files have a **600-line** ceiling for new growth; split by behavior, not arbitrary chunks.
- Generated catalogue mirrors and actual fixture data are exempt from production-size limits. Do not label authored implementation as generated to obtain an exemption.
- Prefer short orchestration functions, normally under 60–80 lines. Review functions over 100 lines for mixed responsibilities. No artificial splitting into `part1`, `part2`, `helpers2`, `misc` or generic `utils` files.
- Mechanical checks cannot prove cohesion, acyclic ownership or lack of duplicate logic. At each capability completion, include changed-file line counts and a brief responsibility review alongside tests.

Run the [layout size guard](evidence/check-layout.py) against the implementation base commit. It inspects changed/untracked Go/TypeScript in compiler/internal, runtime and tests/integration. Use a base after integrating unrelated work. It does not refactor code, update the baseline or excuse semantic violations. If a justified exception is genuinely needed, document the specific responsibility and reason and explicitly revise the contract; do not silently increase the threshold.

## Required completion evidence

Before a capability's DONE step: show its actual file layout, verify the size guard, verify native lowering and runtime module graph, and review that no feature bodies were added to the protected hubs. Include the extraction behavior tests when existing code moves. The architecture prerequisite and these checks are part of implementation, not optional cleanup after all thirteen features are finished.
