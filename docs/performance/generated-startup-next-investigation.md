# Startup attribution follow-up (read-only)

Status: G30b handoff, 2026-09-28. Attribution (final):
[result](generated-startup-result.md). Nothing below is implemented; pruning,
lazy initialization and invoke changes all need Codex's next reviewed packet
with fresh consultations where required.

## Dominant resolved stages

Adapter/facade-root startup over the actual runtime fixture (busy-host
across-trial medians from the final record): module import/evaluation
~16.9 ms dominates synchronous initialization ~3.0 ms. Inside
initialization, `$canDomain` (0.91 ms) and `$canText` (0.86 ms) dominate;
the next statement is `$canAssets` at 0.12 ms and 41 of 57 statements sit
at or below 0.015 ms. Diagnostics import/configuration is unresolved by
absence (perfemit emits no `diagnostics/source-index.json`), not by timing.

Reconciliation with production: the production entry additionally evaluates the
entry supervisor, the main module and the diagnostics module, then runs
configure-then-initialize inside `runEntry`. The assertion path adds per-case
modules. This packet's import stage therefore understates production entry and
assertion startup; improvements here transfer qualitatively, never as a
measured production number. File counts (282 transpiled modules, 59 state
imports) are context, not time per module; the bundle observation does not
prove any emission rewrite safe.

## Source usage and side effects

`$canText` (`runtime/text.ts:23`): the factory eagerly constructs
`new Intl.Segmenter("und", {granularity: "grapheme"})` at line 34. The
segmenter is used only by `graphemes()` (line 114); the fixture workload,
facade oracle and all 24 generated/native oracles never call it (Unicode
coverage uses `normalizeNFC`, which needs no segmenter). The earlier note
claimed fixed valid arguments cannot throw; that is qualified now: native
capability/constructor effects (missing ICU data, platform capability,
first-use lazy initialization inside the engine) and first-use failure
timing need source/contract investigation before any lazy rewrite, because
moving construction to first use could shift when a failure surfaces. The
`unpaired surrogate` rejection precedes segment use and must keep its
precedence. Smallest safe remedy candidate, still needs a designed packet.

`$canDomain` (`runtime/domain-core.ts:100`): construction validates every plan
declaration/shape, verifies one synchronous sha256 digest per concrete shape,
and runs an O(declarations x catalogue errors) linear scan
(`catalogue.errors.find`, line 117). Invalid plans throw `TypeError` during
startup, which `runEntry` reports as phase `initialization`. Any restructure
must preserve fail-fast invalid-plan errors and their initialization-phase
attribution, plus catalogue agreement checks. Larger design, not a minimal fix.
Caveat: AST callee labels collected inside nested function bodies (for
example the `$canIs*` validators inventoried under statement 10) are
syntactic inventory, not proof those factories executed during startup;
counts must not be read as timings.

Import stage (~16.9 ms total, per-module split unmeasured): runtime module
top levels are mostly cheap `WeakMap`/`WeakSet`/`TextEncoder` allocations plus
the required single fd-3 read (`runtime/environment.ts:4`). The heaviest
visible top-level unit is the generated 405 KiB `runtime/catalogue.ts`
(18,415 lines) with a recursive deep freeze, imported via `domain-core.ts`.
Deferring that freeze has almost no window (the catalogue is consumed during
domain construction), so it is investigated but low-promise. Splitting the
import total needs a further temporary module-attribution harness, not a
production change; no reachability/pruning design is supported yet because
binding side effects and assertion/owner/domain initialization are unmapped.

## Candidate inventory (concrete files and contracts)

1. Lazy grapheme segmenter: `runtime/text.ts` (`createText`, `graphemes`).
   Contracts: frozen factory shape, exact `graphemes` values, rejection
   precedence, no new failure mode, preserved failure timing. Needs a
   designed packet + tests, no consultations strictly required (narrow, but
   Codex decides).
2. Domain validation structure: `runtime/domain-core.ts`
   (`createDomainRuntimeWithDigest`), `runtime/domain.ts`, generated
   `runtime/catalogue.ts`. Contracts: invalid-plan `TypeError`s, digest
   verification, catalogue agreement, initialization-phase reporting.
   Needs design + three fresh consultations (ownership/startup decision).
3. Temporary per-module import attribution: new opt-in harness only, no
   production change. Prerequisite evidence before any import
   pruning/lazy-init design. Needs a reviewed packet.
4. Not supported: catalogue freeze removal, import pruning, factory lazy-init
   beyond (1), invoke/await changes. No implementation in this packet.

## Continuing dispositions

Per-site authored invoke proof and secondary numeric JSON retention stay open
as designed investigations. All twelve slices remain dispositioned: compiler
(correctness accepted, timing optional), editor (historical comparison kept),
generated (forwarding landed, invoke proof open), runtime (completion/snapshot
landed), codecs (numeric retention open), startup (attributed here, remedy
packets pending), assertions (fresh roots required, inherits startup work),
artifacts (phase attribution still needed), server (no isolated cause), I/O
(ownership copies required), journeys (follow shared candidates), browser (no
performance conclusion; behavior checks ride any shared runtime change). The
final independent exhaustion review is still pending; this packet declares no
exhaustion and no speedup.
