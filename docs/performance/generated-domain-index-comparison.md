# Domain catalogue index: before/after startup comparison

Status: G35 observation, 2026-09-28. Design:
[generated-domain-index-plan.md](generated-domain-index-plan.md).
Baseline: `generated-domain-index-baseline.json` (`7cf4f8eb`, 1059611 B,
2026-09-28T19:27:34Z-19:27:45Z, pre-change runtime). After:
`generated-domain-index-after.json` (`993e8424`, 1059505 B,
2026-09-28T19:35:47Z-19:35:57Z, `runtime/domain-core.ts` `ad4a4704` with
private catalogue indexes). Frozen tool `f8f8bc51` used unchanged for
both; no reruns, no resampling.

## Method and membership

Both records are explicit busy-host, non-isolated observations with the
same method: 6 independent sequential trials, 2 excluded warmups and 7
accepted batches per trial/profile, 4 profiles, one fresh startup per
batch, ordinary counterbalance with diagnostic last, 300-second sampling
and 64 MiB/500-file caps. Both completed 216/216 launches with 0
rejections, exact 7+2 cell membership, frozen == after identities, and
verified scratch retirement (42.5 MiB/298 files). Host activity was
639/639 processes (baseline) and 639/640 (after), observed without
command arguments or environment. No competing task-owned commands ran
during either sampling window.

Expected source delta: the prepared bundle hash changed from `27ebf9cf`
(907741 B) to `5e8ca435` (908366 B) because the bundle embeds the edited
`domain-core.ts`; emitted state is byte-identical (`5f43d077`, 57
statements), and all 13 reachable bare specifiers stay builtin.

## Observed stage medians (ms; across-trial median [range] MAD)

| Profile / stage | Before | After |
|---|---|---|
| modules import | 16.846 [16.711, 16.972] 0.096 | 16.825 [16.682, 17.441] 0.131 |
| modules initialize | 2.926 [2.870, 2.995] 0.028 | 2.742 [2.695, 2.808] 0.014 |
| modules parent wall | 30.014 [29.654, 30.104] 0.054 | 29.665 [29.493, 30.788] 0.080 |
| bundle import | 12.144 [12.024, 12.256] 0.066 | 12.122 [11.991, 12.209] 0.073 |
| bundle initialize | 2.781 [2.766, 2.868] 0.012 | 2.620 [2.570, 2.632] 0.008 |
| bundle parent wall | 24.751 [24.610, 24.893] 0.054 | 24.527 [24.346, 24.777] 0.087 |
| minimal import | 0.176 [0.170, 0.185] 0.003 | 0.182 [0.178, 0.220] 0.002 |
| minimal initialize | 0.004 [0.004, 0.005] 0.000 | 0.005 [0.005, 0.005] 0.000 |
| diagnostic import | 17.250 [17.153, 17.418] 0.060 | 17.173 [16.961, 17.271] 0.035 |
| diagnostic initialize | 3.017 [2.963, 3.050] 0.018 | 2.833 [2.791, 2.871] 0.031 |
| $canDomain statement (inclusive) | 0.898 [0.888, 0.906] 0.005 | 0.749 [0.740, 0.757] 0.003 |

Every after trial median sits below every before trial median for module,
bundle and diagnostic initialization (e.g. modules after max 2.808 ms vs
before min 2.870 ms). Import stages and the minimal control overlap
between versions, as expected: the indexes are lazy, so module import
defines but does not build them.

## What this does and does not show

Source-proven work removal (separate from timing): each
`createDomainRuntimeWithDigest` factory no longer scans 110 catalogue
errors per declaration nor rebuilds two 215-entry shape Maps; the
browser path paid that twice (verify then create). Bounded retained
metadata is exactly one 110-entry error index plus the paired 215-entry
shape indexes.

First-startup observation: initialization medians are lower after the
change with disjoint ranges, consistent with removed per-factory
catalogue work during `$canInitialize` (the `$canDomain` statement
itself drops ~0.15 ms). This is an observation, not an isolated causal
proof: versions ran sequentially (before then after) on a busy host, so
host drift and cache warmth cannot be excluded. The stable minimal
control argues against gross drift but does not isolate causation.

Unmeasured: repeated-factory benefit. Single first-startup observations
do not measure the repeated-construction gain that motivated the change;
no repeated-factory benchmark was taken in this packet.

No p95, no speedup claim, no production-entry claim. The numbers above
are the complete comparison; no partials were repeated or resampled.
