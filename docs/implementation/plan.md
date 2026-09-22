# Stage 4 — Implementation plan

Prepared after [reconciliation](reconciliation.md). Stage6 implementation subsequently ran to completion: [tasks](tasks.md) records all fifty tasks checked with evidence, and [coverage](coverage.md) maps contracts to owners and evidence. The earlier pause is over and the [file-level handoff](remaining-tasks.md) now records every plan closed. This plan targeted the whole approved redesign, not repairs to the predecessor compiler.

## Milestones and exit gates

| Milestone | Tasks | Observable exit |
|---|---|---|
| M0 — Reproducible target and catalogue | I01,I02,I47 | Exact private Bun runs offline on arm64; catalogue IDs/signatures are machine-checkable; no application syntax executed yet |
| M1 — First executable compiler spine | I03–I07,I09–I12,I40,I48,I49 | A manifest-backed CLI program parses/checks/emits into dist, runs with packaged Bun, passes an actual generated-code assertion, and reports a Can source location for a deliberate failure |
| M2 — Native AI vertical slice | I08,I13–I17,I20 | A named Noul question and judge with grouped state produce one observed POST to a local compatible server; real codec/transport validate a raw-provider fixture; handler/continuation return a typed result; no live credentials required |
| M3 — Language composition and complete AI | I18,I19,I21–I28,I46 | Callables/generic bodies/coordination/all_failed, initial collections, full judgment forms, named fetch and text/record LLM pass positive, negative and runtime conformance gates |
| M4 — Useful platform | I29–I38 | CLI utilities, safe HTML, HTTP/HTMX and PostgreSQL including ownership/transactions pass their native integration suites |
| M5 — Whole-design admission | I39,I41–I45,I50 | Current examples/catalogue replace old behavior; packaged release candidate passes P15 gates; editor/docs match; no legacy execution/proof/extern path or colocated generated TS remains |

Task dependencies, rather than numeric ranges in this summary, govern execution. Independent tasks may proceed after dependencies pass; there is no requirement to retain old public behavior during the replacement. M1 is deliberately executable before full native AI exists; M2 is the next product slice, not an optional late integration.

## Compiler and runtime boundaries

Source files carry byte offsets plus line/column maps; diagnostics and mappings must convert to the convention required by each consumer (LSP/source maps use UTF-16 columns). The parser produces explicit declarations, expressions, blocks and section boundaries. The resolver creates package-owned identities and eligible-kind tables. The type pass admits immutable data, generics and callable contracts. Completion checking assigns executable region owners and exact error bounds; AI checks produce ordinary typed handler regions and provider descriptors. Assertions are annotated by typed-AST preorder before emission.

The checked IR must distinguish preparing a participant from launching it, producing a value from terminal completion, and a provider boundary from an ordinary function. TS emission never reconstructs these distinctions by string inspection. One closed intrinsic registry connects signatures/error catalogue/opaque identities/native mappings. The private runtime is partitioned by contract (completion, owner, codec, transport, AI, assertion, platform), and all identity-bearing registries are imported once per program.

Production coordination delegates settlement to native Promise combinators. The assertion driver controls supplied boundary completions through native promises and P5 barriers; it does not replace production scheduling. All normal Can computation still runs emitted TS. Root execution awaits tracked work and resource drain; an external process supervisor imposes test wall-clock limits for intentionally never-settling cases.

## Separate dist output

Planned output layout under the project root:

```text
dist/
  .can-owner.json
  builds/<build-id>/
    manifest.json
    entry.ts
    packages/<package-id>/<source-relative-path>.ts
    packages/<package-id>/<source-relative-path>.ts.map
    runtime/<distribution-id>/*.ts
    assets/<digest>/<safe-name>
    diagnostics/source-index.json
    assertions/<root-id>.ts
  current.json
```

`package-id` is a deterministic encoding of validated canonical package identity, not a sanitized basename. Build ID binds source/dependency/catalogue/compiler/runtime/options hashes. Handle filesystem case folding and reserved path collisions explicitly. Source-relative paths are confined below their package output root. Every emitted import names an exact relative `.ts` file in this tree. `import type` keeps type-only edges separate; nominal identities do not depend on machine absolute paths. Runtime files and helper dependencies are compiler-owned, prebundled and hash-locked. No import points into authored `.ts`, ambient npm modules or an extern file. No runtime auto-install.

First build may claim a missing or empty real `dist`; a nonempty unowned directory is an actionable refusal, never an invitation to delete its contents. A previously owned tree carries a schema/version/project identity marker. Stage files only inside a newly created owned build directory, validate the complete import graph/maps/assets, then atomically replace `current.json` last. A failed compile cannot make incomplete output current. Run reads and validates the manifest rather than guessing the newest directory. Concurrent builds require a project build lock and publication protocol; no overlapping writers silently replace a live build.

Clean validates canonical project identity, ownership marker, containment and every ancestor against symlinks before deleting only manifest-owned regular outputs. Refuse project/source/root/home paths, symlinked dist, `..`, mismatched markers and unexpected files. Never follow links during recursive deletion. Active run manifests retain their build until the child exits; pruning skips active generations. Power loss between staging and publication leaves an identifiable orphan generation that a subsequent validated cleanup can reclaim. Keep unknown user files intact and explain the conflict.

The existing `.gitignore` already ignores `dist`; test nested projects and add explicit rules only if required. Remove tracked generated TS only as part of replacing its example/test owner. Authored private runtime TS remains source and is not mistaken for generated output. Rebuild from a clean owned dist must produce the same semantic manifest and generated content, allowing explicitly documented nondeterministic metadata outside content hashes. No stale source-deleted module may remain importable through the published graph.

Source maps preserve Can filenames and source content policy without embedding secrets/environment. Go emits mapping segments; a pinned upstream helper encodes them. Synthetic frames carry meaningful generated-operation labels and nearest Can span. Both compile errors and runtime standard failures name Can locations; internal adapter errors retain useful cause information without leaking bearer values. Bun TS-transpilation map composition and thrown-stack mapping are explicit tests, not assumed from `.map` existence.

## Distribution, installation and upgrades

Ship a versioned macOS arm64 archive/container with Go launcher, exact upstream Bun sidecar, closed stdlib/runtime/tool assets and notices. M0 uses an unmodified upstream runtime in the same intended layout; M5 qualifies the complete artifact, which ships unsigned until signing credentials are authorized (see the distribution release notes). Users need no global Bun, Node, npm, Go, C compiler or Z3. Building Can from source may require Go/C tools according to the SQL binding chosen; that is a developer prerequisite, not a user runtime dependency.

Use exact Bun1.4.2/revision/digest as the initial candidate from research; freeze actual archive and extracted executable hashes in the release manifest. Never silently follow `latest`. Resolve runtime from the real launcher installation root, verify platform/minimum OS and expected manifest, then execute with an argument vector, controlled cwd and explicit configuration isolation. Preserve application environment required by approved env/auth operations, while blocking ambient Bun preload/config/auto-install behavior from injecting backend code. Qualify the exact supported isolation flags against the pinned executable.

Installation stages a complete verified version directory, validates executable permissions/signatures/notices, then atomically selects it. Updates repeat the same process; running processes keep their old runtime. Do not call `bun upgrade`, overwrite a running runtime or auto-download dependencies while compiling. If an update fails, retain the prior verified installation. Cleanup acts only on installation-owned inactive versions. Sidecar moved/missing/tampered, wrong architecture, unsupported OS and permission failures have specific diagnostics; PATH substitution is forbidden.

Release gates inspect upstream executable signature/entitlements, sign the Go launcher appropriately, validate the chosen distribution container/notarization workflow, and exercise a quarantined download on a clean machine without disabling OS protections. Collect all exact upstream notices and required corresponding sources/relink materials. No certificates were requested or signing/upload performed in preparation. Reconsider embedding only if a concrete one-file deployment requirement appears; its separate extraction/cache/trust tests would then be new work.

## Validation and sequencing rules

Each task declares positive, negative and integration evidence. Static acceptance cannot substitute for runtime behavior; mocked provider results cannot establish live quality. CI reports P4's five evidence labels separately. Core/native/fixture conformance is mandatory and deterministic. Credentialed provider quality is an explicit optional/release-policy-controlled suite with recorded model versions, never silently performed by compile or editor operations.

Use Go tests for compiler passes and path/manifest policy; strict pinned TypeScript checking for emitted artifacts; packaged Bun for runtime semantics; a local mock HTTP service for raw wire behavior; disposable PostgreSQL for driver/transaction gates; browser automation for pinned HTMX flows. Pin every test dependency and disable acquisition during normal compile/run. Do not add a new algorithm merely to make CI independent of an upstream runtime.

Replacement tests are derived from current spec examples and edge contracts. Keep only legacy scenarios whose meaning survives, rewrite their source/goldens and explain changed expected behavior. Reject superseded spellings explicitly during migration; no deprecation/compatibility layer is required. The final gate checks the entire catalogue, C11/A12/P13 applications and every historical finding's closure. Green old tests alone never close a task.

## Stop conditions

An unsupported native feature, parser scanner mismatch, incorrect deadline/drain behavior, incomplete source mapping or unsafe output deletion blocks its dependent milestone. Record the failing minimal case and upstream version before considering another upstream version/library. A change to a Can contract requires a separate design correction; this plan does not preauthorize it. Implementation was authorized, paused, resumed under renewed authorization, and completed with all fifty tasks checked.
