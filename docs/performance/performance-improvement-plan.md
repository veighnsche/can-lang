# Continuing performance improvement campaign

Status: active, authorized on 2026-09-28. The user clarified that the objective
is to exhaust performance fixes supported by evidence, rather than finish after
one verified editor improvement. There is no artificial total-time or step cap.
The completed [editor result](editor-responsiveness-result.md) is checkpoint one.

## Stop condition and operating contract

Continue through investigation, bounded design, Muse implementation, independent
review, correctness, qualified comparison, compact reporting and commits. A
completed patch returns to the remaining queue. Do not declare the campaign
complete until every identified candidate has a verified improvement or a
specific evidence-based disposition, and a final source/evidence pass finds no
ready supported fix. A missing prerequisite blocks its lane, not unrelated work.
Avoid speculative rewrites or invented work solely to occupy time.

Codex owns investigation, design, consultations, saved task lists, measurement supervision and
independent acceptance. Muse Spark 1.3 Contributor owns all implementation, implementation
tests, harness changes and review-driven corrections
from [the ordered checklist](performance-improvement-tasks.md), with maximum
reasoning effort as explicitly requested, authorized yolo and worktree off. The
first continuing run inherited the already configured `max`; subsequent dispatches
set `--reasoning-effort max` explicitly. No concurrent edits
to Muse-owned files. Preserve code, contracts and negative tests rather than
achieving apparent gains by reducing validation or workload.

The user waived the quiet-host waiting requirement for this campaign and prioritized
more fixes in generated TypeScript execution. Catalogue timing is optional and does
not block independent runtime work. Global harness defaults remain unchanged. Future
campaign timings must be explicitly labelled non-isolated/busy-host, with observed
host activity rather than fabricated quiet qualification. Measurements remain
sequential and allow no competing task-owned build/test work. Do not control user applications
or power settings. Use focused experiments rather than repeatedly rebuilding or
rerunning the twelve-slice audit. Keep the original seven batches, two excluded
warmups and six independent processes per variant for headline comparisons;
declare separate smaller functional/attribution exchanges honestly. No p95 claim.

Reuse the local pinned Bun archive and shared Go cache, bounded Go workers and
existing owned preparation. Register cleanup immediately; reclaim task-owned
execution files on success, failure and handled interruption. Keep execution
scratch below 2 GiB, aggregate trace below 32 MiB and each compact comparison
archive below 10 MiB. No installs, private caches, retained bundles, disposable
checkouts, shared-cache removal, Docker cleanup or swap cleanup.

## Evidence and current queue

The historical full-system run `.performance/20260927T235706.371974Z/evidence.zip`
contains 104 cases and the adjacent grading review. It is from an older source
revision; use it to choose investigations, not as a current candidate baseline or
an optimization-savings ranking. Grades are advisory, not proof of cause.

| Lane | Evidence / current hypothesis | Next acceptance obligation |
|---|---|---|
| Compiler and editor | First patch reduced flat-100 warm/edit medians to 15.607/31.230 ms. Final functional trace still has ten whole catalogue clones per snapshot (11.610 ms inclusive). Method dispatch, constructor metadata and array pin lookup still copy full inventory. Invoice warm/edit remain 196.638/393.102 ms, with qualified-A functional attribution showing 148 inventory copies costing 166.373 ms in a 201.135 ms request. | Remove demonstrated metadata over-copying through narrow owned lookups; prove exhaustive equivalence and compare current qualified binaries. Profile the maintained project before attributing its residual cost. |
| Generated execution | Historical doubled, generic/captured map, fold and frequency are approximately 8–20 times their named native controls, with small absolute workloads. | Inspect emitted/runtime adapters and native-equivalent contracts; preserve immutable values, completion boxing, failures, callback order, context, ownership and thenable safety. Design from attributable overhead before changing semantics. |
| Assertions | Historical 100-root serial supervisor costs 5.339 s; utilities jobs-1/jobs-4 cost 321/165 ms. Each root intentionally uses a fresh worker. | Separate root population/startup/validation costs and preserve fresh harness, isolation, ordered delivery, deadlines, cancellation and reaping. Do not treat a smaller assertion population or worker reuse as a gain. |
| Artifacts | Historical first publication 721 ms, reused publication 51.9 ms, native validation 126.7 ms. Source uses file synchronization and integrity walks. | Attribute redundant work while retaining durability, atomic selection, leases, tamper/symlink refusal and closed inventory. No removal of fsync or safety checks merely for speed. |
| Runtime and codecs | Helper costs are generally modest; array callbacks have multiple async/context/completion layers and immutable point-map updates intentionally copy histories. Native JSON parsing lacks equivalent validation. | Inspect shared costs together with generated output; preserve full schema, duplicate-member/number/Unicode rejection and resource/thenable contracts. No raw-parser equivalence claim. |
| Startup, server, I/O, journeys | Existing representative records and source paths remain available; no new cause identified yet. | Review controls, absolute costs, source changes and repeat work. Admit a fix only with a demonstrated cause and applicable semantics. |
| Browser | Historical callback timings below resolution do not measure input-to-paint. | Treat this as a measurement gap; do not invent a browser speedup from zero medians. Use existing deterministic browser behavior checks if a shared adapter changes. |

## Next selected compiler design

After the consultations and before Muse changed source, one bounded functional
invoice exchange reused qualified A. It counted 148 whole inventory copies in a
warm request, 166.373 ms of copy duration inside a 201.135 ms server request.
The snapshot still resolved twice and checked fully. This attribution supports
the selected metadata remedy on the maintained project; it is trace-enabled
diagnosis, not a quiet-host speed comparison. Raw trace and counters are under
`.performance/catalogue-20260928/raw/` and will retire into its evidence archive.

Three fresh Jev Choice consultations are saved under
`.performance/performance-push-20260928/catalogue-request-*.json` and corresponding
response files, with the full-request equivalence record. Every explanatory
context, question and option passage differs; numerical facts and exact code are
identical. All three selected narrow indexed catalogue access and exhaustive
metadata/ownership equivalence. The alternatives were one private inventory per
program checker with plumbing, explicit whole-inventory Go cloning, and obtaining
invoice attribution before code changes. Agreement is advice, not proof.

Code establishes that the embedded inventory is process-immutable, private and
validated with unique names and identities. Existing name lookup already copies
one operation. Extend only the lookup surface required by real consumers:
identity lookup, current pinned-name lookup, declaration/method metadata and the
error subset. Return deep defensive copies, never internal mutable maps or slices.
Build receiver-method indexes in original inventory order so first-match behavior
is retained. Explicit Operation target/revision rejection remains intact; a
current lookup derives pins from the same validated Catalogue instance.

Migrate collection/stream/browser/SQL helpers without changing their existing
admission filters or negative behavior, array paths without copying the catalogue
to obtain scalar pins, method resolution with original scalar and nominal
precedence, constructors with original constructibility/error handling, and
error-registry seeding with only the required error subset. Resolver construction
and the checker's complete contract gathering may still request an inventory.
No project snapshot cache, asynchronous checker rewrite or public mutable view.

If catalogue timing is resumed, before/after evidence will use a fresh comparison against checkpoint one's source,
with identical frozen harness and trace settings. The small lookup change has a
direct source/trace justification; whether it materially improves invoice-compare
must be established separately. After acceptance, move to the next queue entry.

TypeSafe API and question shapes were checked against the official
[API reference](https://docs.typesafe.ai/api) and
[Choice documentation](https://docs.typesafe.ai/primitives/choice).

## Independent queue review during host waiting

The first catalogue comparison qualified both binaries and traced full checks,
but could not reach a quiet window before its 30-minute gate allowance expired.
Zero accepted timed trials began. Archive, owner/group retirement, scratch
cleanup and exact C1 source restoration were independently verified. The user subsequently waived quiet-host waiting for this campaign and asked to
prioritize implementation. The exact source transition remains saved for optional
later comparison; zero timed trials support a catalogue speedup. No partial merge.

The [independent twelve-slice review](performance-queue-review.md) identifies
completion descriptor allocation, duplicated map containment materialization,
unused nonnumeric JSON token retention and broad generated startup bindings as
further concrete investigations. Their latency contributions remain unmeasured. Generated execution is now the main
lane. Completion construction and duplicate map containment materialization are
source-proven removal candidates; the new generated execution design/checklist
will govern Muse implementation. Compiler/editor timing is not a prerequisite.
Broader generated async callback/invocation and startup work remains in the queue.

## Current generated checkpoint

Completion and map containment allocation changes are independently accepted and
committed as `7e0d5641`; 25 focused tests, runtime lint/format/type checks and 24
fresh actual generated/native cases passed, with owned cleanup verified. No timing
claim. Muse now follows generated-callback-tasks.md G7-G12 with explicit max effort.
That narrower packet preserves the assertion-enabled route, generic context/invoke
implementations and native algorithms while removing source-proven per-element
wrapping/admission and private collection lookup repetition. The campaign continues
through its next source/evidence handoff and independent review.

## Generated callback checkpoint

G7-G12 independently accepted: absent-context collection callbacks retain invoke
while skipping unused assertion dispatch/receipt work; privately authenticated
successes use direct extraction; map/set backing reuses one validated metadata
lookup. 53 applicable contracts, required runtime check and 24 fresh actual
generated/native validation cases passed. Source and emitted identities checked;
all owner/prompt/execution scratch retired. No speedup number is asserted.
[Next source investigation](generated-next-investigation.md) identifies static
origin reuse and startup import/init attribution; these require reviewed design
before Muse implementation. Continue the campaign.

## Generated origin checkpoint and forwarding packet

G13-G18 and acceptance corrections are independently accepted and committed as
`5b1889ee`. Mapped non-self-tail functions lazily retain finite deeply frozen
source/span/region origins; dynamic tail steps and other emission remain unchanged.
Fourteen applicable independent contracts and 24 fresh actual generated/native
cases passed with strict TS and matching module/driver identities. A final narrow
ten-test static-origin run pins exact authored/substituted spans and a real emitted
synthetic failure's first invocation boundary, preserving occurrence freshness.
All owner groups, prompts and execution scratch are retired. The prior invalid
evidence acquisition timestamp is preserved explicitly and corrected in compact
records; successful broad checks were not repeated. No latency gain is measured.

The [forwarding design](generated-forwarding-plan.md) and
[ordered checklist](generated-forwarding-tasks.md) govern the next packet. Checked
authored functions are native async; only exact identity/resolved-binding proof
permits their callable adapters to return the existing promise directly. Generic
invoke admission, catches, boxing, context/owner contracts and unclassified adapters
stay in place. Three fresh fully rewritten equivalent Jev consultations advise this
narrow choice, with varying recommendation strength; correctness remains an
independent obligation. Startup import/evaluation versus initialization attribution,
secondary numeric JSON retention and the twelve-slice exhaustion review remain open.


## Checked authored forwarding checkpoint

G19-G24 independently accepted and committed `4589122c`: only an exact checked
authored identity/resolved binding permits direct promise forwarding. Sixteen
independent tests, zero skips, 24 fresh generated/native oracles, strict TS and
matching source/module/driver identities passed; ownership and execution cleanup
verified. The checkpoint removes source-proven redundant async adoption, with no
measured speedup claim. [Forwarding acceptance](generated-forwarding-tasks.md)
records the review. Next: bounded attribution of actual generated startup loading
and initialization, separate from compilation; then the next supported remedy.
Per-site invoke proof, secondary numeric JSON evidence retention and final
independent twelve-slice exhaustion review remain active.
