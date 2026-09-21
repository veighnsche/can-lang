# Generated source maps and runtime locations

Every compiled program/assertion module has an adjacent `.ts.map`. The generation
also owns `diagnostics/source-index.json` (schema 1, `can.source-index`). Its stable
source IDs distinguish packages and dependencies; displayed filenames are relative
to each project's source root. Maps contain no original source bodies:
`sourcesContent` entries are `null`. The index contains only identities, relative
filenames, checked spans, operation labels and generated segment positions.
Generated TypeScript necessarily still contains the compiled program's literals.

Go carries checked IR byte spans through emission using private assembly tokens,
then removes those tokens and computes generated UTF-16 coordinates after imports
have been rendered. Authored strings cannot inject tokens. The fixed offline tool
`tools/runtime/source-maps.ts` uses bundled `@jridgewell/gen-mapping` to encode
segments. Go does not implement VLQ. The lock and four upstream licenses are in
`distribution/notices/source-maps.lock.json` and adjacent notice files. The helper
lives in the distribution's existing, manifest-owned `tools/runtime` tool tree.

Before publication, the driver requires the map/index/reference inventory, checks
source spans against the parsed Can files, and invokes the pinned native validator.
Upstream decoding and canonical re-encoding must agree with every compiler segment;
unknown source IDs, embedded source contents, malformed/noncanonical maps, out-of-
range generated positions and helper failure refuse publication. All these files
participate in the immutable generation digest and active execution lease.

Bun 1.4.2 reports TypeScript coordinates for direct `.ts` execution but does not
apply the external Can map in the qualification probe. The private diagnostic
runtime therefore traces those TS coordinates through bundled
`@jridgewell/trace-mapping`. Standard failures and assertion failures expose only
indexed Can locations and fixed operation labels. Domain failures and causes
without an eligible native stack use the checked operation origin, marked
`synthetic: true`. Runtime helper paths and native frame names are omitted; mapped
caller operations retain useful labels such as `index`, `binary`, or `call`.
Native messages, causes, application payloads and absolute machine paths are not
serialized. Diagnostic byte intervals are half-open; displayed line/column pairs
are one-based, with columns measured in UTF-16 units.

The target preserves nested direct-await frames and the throwing async callback's
location. It can omit callers across `.then(async …)`; no missing frame is invented.
The qualification tests cover both forms, Unicode before the fault, and synthetic
origin fallback. Stack inspection rejects proxies and stack accessors, and the
pinned native Error descriptor probe verifies that name/message getters are not
called. Compiler-private initialization/configuration defects remain nonzero and
sanitized rather than exposing an unhandled native stack.

Assertion suite initialization failures use the same mapped reporting boundary.
Sticky harness violations retain their occurrence tokens and locations even if
authored code catches the standard failure. Synthetic adapter failures keep their
original metadata and acquire a separate first checked call-site origin; later
callers do not replace it.
