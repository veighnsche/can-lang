# Generated startup attribution result (final)

Status: final, 2026-09-28. Packet: G28e-G30b correction of G25-G30
startup attribution. Raw record:
`.performance/performance-push-20260928/generated-startup-attribution-final.json`
(1092771 bytes, sha256
`fb060ec5…`, 216 rows, status complete). Tool: final
`tools/performance/startup-attribution.py --measure` after G28h
qualification (191/191 tests, strict TS, 24/24 oracles); that producer is
driver `77f29d04`. Codex independently verified all 69 reported summaries,
216-row membership, bounds, counterbalance and 456 input hashes, but
withheld tool acceptance for three failure-branch defects (allocation
rollback gaps, pipe-capacity supervision stall, first-importer resolution
collapse), since narrowly repaired and separately qualified as driver
`3ebeff0c` (209/209 tests, identity continuity, no new sampling). A
fourth review withheld acceptance again for two custody gaps (replaced
marker destroyed on failed registration, metadata commands without
durable group custody), since narrowly repaired and separately qualified
as driver `f8f8bc51` (224/224 tests, identity continuity, no new
sampling). This record stays historical `77f29d04` producer evidence
accepted within the bounded builtin-only attribution scope after Codex's
224-test/zero-skip review, one actual validation graph, replacement/custody
checks and verified cleanup; the auxiliary lane is frozen. Compact evidence:
`generated-packet-5-independent-bounded-acceptance.json`. It is never relabelled as repaired-code
output. Every number below comes from the final record only. The original
`generated-startup-attribution.json` and the corrected
`generated-startup-attribution-corrected.json` stay on disk unchanged and are
explicitly provisional; their limits are listed at the end. No p95, no
speedup number, no causal or production-entry claim.

## Scope and conditions

This attributes adapter/facade-root startup over the actual generated runtime
fixture, not total production entry startup. Each batch is one fresh Bun child
with a fresh private fd-3 `{}` snapshot; the launcher never statically imports
Can runtime. Preparation (one supervised perfemit build, emission, JS
transforms, probe instrumentation, bundling) is recorded separately and is
not a startup sample; preparation and sampling bounds are recorded separately
in the final record.

Conditions are explicitly **non-isolated/busy-host**: OS and Bun caches
uncontrolled, no quiet waiting, no CPU-idle polls, no app/power control. Host
activity was observed before/after sampling via `ps` without command arguments
or environment (595/592 processes; WindowServer and Chrome active). No
competing task-owned builds or tests ran during sampling
(2026-09-28T17:59:13Z to 17:59:19Z; overall run 17:59:09Z to 17:59:20Z).
Six independent sequential driver trials ran with counterbalanced ordinary
profile order and the separately labelled diagnostic phase fixed last. Two
warmup batches per trial/profile are retained in the raw record and excluded
from medians. Complete prepared-graph/source/driver/resolved identities
frozen before trials exactly match the post-sampling inventory (no drift);
scratch retired after verification (42.5 MiB/298 files, under caps).

## Stage observations (ms)

Across-trial median / range / MAD of per-trial medians (exactly 7 accepted
batches each; membership enforced, no partial data). Parent wall is separate
from child stages. Diagnostics stages never measured: perfemit emits no
`diagnostics/source-index.json`, so module-graph profiles honestly record
`skipped_no_metadata` and the bundle records `not_applicable_bundled` (its
diagnostics module cannot be timed separately).

| Profile | import | initialize | parent wall |
|---|---|---|---|
| ordinary-modules | 16.942 [16.800, 17.648] 0.019 | 2.974 [2.931, 3.013] 0.035 | 30.011 [29.939, 31.412] 0.049 |
| ordinary-bundle | 12.369 [12.150, 12.666] 0.072 | 2.840 [2.780, 2.875] 0.031 | 25.034 [24.877, 25.911] 0.118 |
| minimal control | 0.178 [0.174, 0.185] 0.003 | 0.005 [0.003, 0.005] 0.000 | 7.256 [7.228, 7.421] 0.019 |
| diagnostic-modules | 17.523 [17.269, 19.416] 0.208 | 3.072 [2.966, 3.286] 0.071 | 30.760 [30.277, 34.745] 0.373 |

Per-trial medians (import / initialize / parent wall). Trial 5 ran hot on
the diagnostic profile and module import (busy-host noise, retained
honestly; nothing excluded).

| Trial | ordinary-modules | ordinary-bundle | minimal | diagnostic |
|---|---|---|---|---|
| 0 | 16.80 / 2.94 / 29.94 | 12.45 / 2.88 / 25.23 | 0.17 / 0.00 / 7.26 | 17.27 / 2.97 / 30.28 |
| 1 | 16.93 / 2.95 / 30.01 | 12.15 / 2.78 / 24.88 | 0.18 / 0.00 / 7.25 | 17.62 / 3.09 / 30.92 |
| 2 | 16.92 / 3.00 / 30.02 | 12.30 / 2.81 / 25.02 | 0.17 / 0.00 / 7.24 | 17.42 / 3.05 / 30.60 |
| 3 | 16.96 / 2.93 / 29.98 | 12.36 / 2.84 / 24.96 | 0.18 / 0.00 / 7.23 | 17.36 / 3.04 / 30.50 |
| 4 | 16.95 / 3.00 / 30.18 | 12.37 / 2.84 / 25.05 | 0.18 / 0.00 / 7.42 | 18.05 / 3.26 / 31.98 |
| 5 | 17.65 / 3.01 / 31.41 | 12.67 / 2.87 / 25.91 | 0.18 / 0.00 / 7.41 | 19.42 / 3.29 / 34.75 |

One-time preparation observations (not samples): perfemit build 2689 ms,
emission 502 ms, JS transforms 200 ms, probe inventory+instrumentation
333 ms, bundle 24 ms. All 216 launches passed oracle, exact output, profile
status rules, finite-millisecond checks and (diagnostic) complete 114-event
coverage; zero rejections, zero partial acceptances.

## Diagnostic statement observations (ms, inclusive only)

Each value is one statement's inclusive cost (argument preparation plus
observer overhead), never an isolated factory time. The 57 instrumented rows
must not be summed into ordinary totals, and medians must not be subtracted
to manufacture exclusive costs. All 57 statements with full raw events are in
the final JSON record; the top 16 of 57 by across-trial median:

| # | Binding | median | range | MAD |
|---|---|---|---|---|
| 10 | $canDomain | 0.907 | [0.894, 0.970] | 0.009 |
| 38 | $canText | 0.855 | [0.847, 0.906] | 0.008 |
| 15 | $canAssets | 0.117 | [0.110, 0.136] | 0.006 |
| 16 | $canSQL | 0.082 | [0.078, 0.095] | 0.004 |
| 17 | $canSQLPools | 0.066 | [0.065, 0.078] | 0.001 |
| 51 | $canS3 | 0.050 | [0.049, 0.054] | 0.001 |
| 14 | $canHTML | 0.042 | [0.038, 0.047] | 0.002 |
| 53 | $canBrowser | 0.035 | [0.034, 0.042] | 0.002 |
| 25 | $canHTTPResponses | 0.021 | [0.020, 0.022] | 0.001 |
| 29 | $canServer | 0.020 | [0.019, 0.020] | 0.000 |
| 18 | $canTransactions | 0.019 | [0.018, 0.020] | 0.001 |
| 48 | $canWebSockets | 0.018 | [0.018, 0.020] | 0.001 |
| 24 | $canHTTPRequests | 0.018 | [0.017, 0.020] | 0.001 |
| 52 | $canMarkdown | 0.017 | [0.016, 0.019] | 0.001 |
| 11 | $canBytes | 0.015 | [0.014, 0.018] | 0.001 |
| 49 | $canCookies | 0.015 | [0.014, 0.018] | 0.001 |

All remaining 41 statements have across-trial medians at or below 0.015 ms.

## Explicit non-claims

No p95, no speedup number, no causal claim. The bundle import observation does
not prove a production emission rewrite safe (Bun tree shaking is part of that
prepared profile). File/module counts (282 transpiled modules, 59 state
imports) are context, not time per module. The dominant resolved stages are
module import/evaluation (~16.9 ms modules, ~12.4 ms bundle) and synchronous
initialization (~3.0 ms); diagnostics import/configuration is unresolved by
absence of emitted metadata, not by timing. Small noisy stages remain
unresolved where the data cannot support a remedy.

## Prior record limits (why they are provisional, not merged)

The original `generated-startup-attribution.json` is preserved byte-identical
for audit but must not be cited as evidence: its tool accepted
negative/NaN/infinite stage and event times, reports built from incomplete
batches, wrong whole-source identity, omitted-statement manifests, UTF-16
astral-span blindness, prefix-only foreign cleanup, preexisting-output
deletion on second-write failure, unsupervised trial-controller descendants,
and cleanup failures mislabelled as retired. It also lacks prepared-graph,
resolved-dependency and driver identity inventories, and its scratch was
retired, so those historical identities cannot be reconstructed. Correction:
the original note's prose said "trial 4 ran hot" while its own table showed
trial 3 hottest; that prose was wrong and is superseded here.

The corrected `generated-startup-attribution-corrected.json` is likewise
preserved byte-identical but must not be cited: its tool left preparation
unsupervised in the parent process, waited on descendant-held pipes instead
of retiring the worker group on controller exit, had no verified-retirement
failure type, accepted blank identity strings, recorded no emitter inputs,
linked-runtime/emitted inventories, executable/builtin identities, reachable
bare-specifier resolutions or separate preparation/sampling bounds, and
registered allocations without marker-failure rollback or keeper custody.
The final record above replaces both for every purpose; no prior row was
reused or merged.
