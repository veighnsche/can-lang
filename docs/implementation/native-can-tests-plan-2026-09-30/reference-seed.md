# Proposed reference seed (R-seed)

Current audit status (2026-10-01): [P04 materialization](evidence/P04.json) and
[P05 bounded R-seed controls](evidence/P05.json) are recorded complete. This is
only the bootstrap/literal-subset seed scope; R-core (P26), actual N with a
qualified host (P14), and suite acceptance (P23) remain unfinished. No current
artifact execution or requalification was performed by this planning audit.

The inventory, missing-artifact statements and proposed recipe below are the
original P03 selection-time snapshot, retained for provenance; they are not
current availability or execution instructions.

## Inventory at selection time

HEAD is `626bf03743dbfa660d4b4651c2b0b3cceaeae0a5`. The delta from the
planning baseline `4cf8f91d` is exactly five commits (P00 baseline, design
contracts, two AGENTS.md rule updates, P01 schemas) with zero diff under
`tests/`, `compiler/`, `runtime/`, `tools/` and `host/`.

| Role/input | Exact selection | State |
| --- | --- | --- |
| Candidate source | `/Users/vince/Projects/can-lang` at `626bf037` | Identified; not an executable identity |
| Compiler source | `compiler/` tree at HEAD | Source only; no bundle built |
| Catalogue input | `compiler/internal/catalogue/catalogue.json`, SHA-256 `bdc7874222d8eccc7dc651bc949d98871e9b4e87e313b44cf1ca0c51c2957659` | Identified; `cataloguegen --check` pending in P04 |
| Runtime inputs | `runtime/`, `tools/runtime/`, `distribution/assets/`, `distribution/notices/`: 336 files, 4,092,369 bytes incl. three ignored `.DS_Store` files | Matches prerequisites inventory exactly |
| Bun subject runtime | `.local-deps/pinned/bun-darwin-aarch64/bun`, SHA-256 `35d20dd0263e5c950194434b925454fdfa9ba6e4467da960410fa05b08a7a5b5` | Bytes match; not executed for this record |
| Bun archive | `.local-deps/bun-darwin-aarch64.zip`, SHA-256 `90987a3a16d7db556d886ac3d551e7b6d3edf0a1cf43acaed622e8676be1d12f` | Matches pin; `CAN_BUN_ARCHIVE` unset, explicit path used |
| Target manifest | `distribution/target.json`, SHA-256 `79a61045c07d345254767554ce0a5c1f31c48129b94a425183379a3e2e68f6df` | Matches prerequisites record |
| Conformance probe | `tests/conformance/native.ts`, SHA-256 `e1af4ae127651b10d08b4be6cf18d67b3bdda5403c8a51972053be1c0d380f35` | Matches prerequisites record |
| Go toolchain | `go1.27.1 darwin/arm64` | Version observed; behavior unqualified |
| Host | Darwin kernel `27.0.0`, `arm64` | Observed; containment/quotas unqualified |
| R bundle | None: `dist/development` empty, no `canlc` on PATH | **Missing**; materialization is P04 |
| N executable | None accepted | **Missing**; implementation/acceptance is P07–P14 |

Machine-readable form: [reference-seed.json](reference-seed.json).

## Proposed R-seed recipe

Build the seed compiler distribution from the inventoried source with the
project's ordinary reviewed bootstrap tools only:

1. Verify every input hash above; any changed or missing artifact fails
   identity and stops the recipe.
2. `go run ./compiler/internal/catalogue/cmd/cataloguegen --check` must pass;
   generated mirrors stay byte-identical.
3. `go build` the compiler from the inventoried source (P04 stages the exact
   command and output ownership).
4. Pair the built compiler with the inventoried catalogue, private runtime,
   pinned Bun and trusted bindings; stage the bundle under run/acceptance
   ownership, not a per-case copy.

Never call candidate `canlc run` to build or stage its judge: that path
builds with its own compiler and would let C compile R. The suite
authoring in P15 and later tasks is compiled by the accepted seed only
after P05.

## Acceptance rule

This proposal carries no acceptance. P04 seals the exact compiler/runtime/Bun
contents and input graph; P05 qualifies the seed for the bounded judge
features with fixed native observations and existing host checks as
transitional evidence. Mutation after selection invalidates. Changed or
missing artifacts fail identity at every later step.
