# Independent continuing queue review

Status: intermediate source/evidence review, 2026-09-28. The campaign remains
active. This review admits investigations, not speedup claims or exhaustion.
It supplements Muse's [generated notes](generated-collection-investigation.md)
and [assertion/artifact notes](assertion-artifact-investigation.md).

The historical twelve-slice run is revision `40eaf6a88`; its numbers are
priority clues. The current catalogue patch has independent correctness
acceptance, but its first comparison could not pass the unchanged quiet-host
gate. No accepted timed trial began. Its compact failed archive is
`.performance/catalogue-20260928/evidence.zip`; owner/group and execution
scratch are retired, and the eleven candidate source hashes match C1.

## Twelve-slice disposition

| Slice | Independently inspected evidence/source | Current disposition |
|---|---|---|
| Compiler | `catalogue.go`, checker consumers, resolver and program contract gathering; qualified functional trace | Narrow lookup candidate correct; performance retry pending. Three whole-inventory copies remain per checked snapshot, versus ten in A. Do not infer remaining timing from copy count. |
| Editor | Current qualified A/B trace and previous verified comparison | Same checked snapshots, resolver population and collection observations. Catalogue correctness accepted; timing is optional under latest user priority, no partial results merged and no unmeasured speed claim. |
| Generated | Array adapter, callable lowering, completion and assertion context, fixture/native controls | Shared runtime overhead candidates. Generated callbacks are async, so an adapter path limited to synchronous callbacks would not directly accelerate these fixtures. |
| Runtime | Completion construction, data containment, map/set adapters | Two concrete allocation candidates below; retain boxing, brands, defensive ownership and histories. Attribution and qualified comparison pending. |
| Codecs | Native duplicate scan/reviver, schema projection, numeric policy, JSONL, AI protocol readers and assertion raw-provider comparison | Numeric-only source-token retention is a candidate. Raw parse timing is not an equivalent full-codec control. |
| Startup | Runtime startup controls, authored module imports and state factory imports/initialization | Broad unconditional Bun imports/bindings are a candidate. Attribute loading versus initialization before a reachability rewrite. Browser already restricts reached authored functions. |
| Assertions | `RunSupervised`, `validateLease`, fresh `prepareEntry` boundary | Root count/fresh process, leases, deadlines, reaping and order remain required. Startup improvements may transfer; worker reuse or fewer roots are not fixes. Root median × population is rough scale context, not measured suite attribution. |
| Artifacts | `Publish`, `Stage`, `SelectCurrent`, `generation`, CAS and synchronization | Repeated hashes/tree walks guard distinct boundaries, including pre-selection freshness/tamper checks. Phase attribution is needed before any check-preserving redesign. No safety/durability removal. |
| Server | Workload arrival scheduler, bounded clients, HTTP request/body/response ownership | No isolated cause yet. About 398 ms arrival trials intentionally schedule 200 requests at 500/s. Bounded-client totals are 200-request batch durations, not per-request latency. |
| I/O | File streaming/cap/cancellation and named native-contract controls | Required stream-view and immutable byte ownership copies match controls. Historical text gap is only about 0.6 µs; no demonstrated isolated fix. |
| Journeys | Emitted invoice and doubled adapters, decode/map/fold/serialize boundaries | Follow shared generated/codec candidates. Valid versus invalid invoice paths are different workloads, not before/after evidence. |
| Browser | Historical timing scope and browser driver | Below-resolution handler durations omit event queue/layout/paint. No browser performance conclusion; shared runtime changes still require browser behavior checks. |

## Concrete next candidates

1. **Completion construction** — `runtime/completion.ts:24` allocates a
   descriptor-map object and two property-descriptor objects for every carrier.
   A native null-prototype object literal followed by freezing may give the same
   final own data descriptors while keeping the existing `WeakSet` brand,
   `kind`/`value` keys, payload identity and thenable protection. This does not
   propose replacing branding or changing invocation scheduling.
2. **Opaque map containment snapshot** — `runtime/collections/map.ts:28`
   materializes `Array.from(values.values())`; `registerOpaqueContents` in
   `runtime/data.ts:11` immediately spreads that array into a second frozen
   array. Passing the private map iterator to the same copy/freeze boundary
   could remove the intermediate array. Keep the stored independent snapshot,
   single-registration check and immutable map histories. Set storage contains
   primitive keys and does not use this metadata path; no set duplication claimed.
3. **JSON numeric evidence** — `runtime/codec/document.ts:38` retains source
   spelling for every primitive. Typed codec/JSONL `exactInt` needs only numbers.
   The independently inspected raw assertion-provider comparison also reads token
   spelling only in its number branch; other primitives use `Object.is`. AI
   protocol consumers use the parsed value. Restricting retention to numeric
   primitives could preserve exact integers and raw numeric comparison while
   removing unused strings/booleans/null entries. Keep root-holder tracking,
   duplicate detection, native syntax checks, budgets and all rejection paths.
4. **Generated startup graph** — Bun authored modules and shared state import
   broad base runtime bindings, cross-module declarations and factory groups.
   Historical generated/minimal launch medians (22.47/5.52 ms) do not separate
   module loading from initialization. Establish current import/initialization
   attribution before designing narrower emission; do not omit side effects or
   assertion/owner/domain initialization merely because a binding seems unused.

These are source-supported opportunities, with latency contribution still
unmeasured. Each packet needs a demonstrated removable cost, exact contracts and
ordered acceptance tasks; current latency attribution is only necessary where
source inspection cannot establish a supported remedy. Difficult ownership, startup or callback decisions
require three fresh fully reworded equivalent Jev consultations; agreement is
advice. Catalogue C1 correctness is accepted. The user now prioritizes generated
implementation independently of optional catalogue timing, and waives quiet-host
waiting only for this campaign. While Muse is alive, its files are exclusively owned
and no competing checks may run. Do not repeat the full-system audit.

## Remaining acceptance work

Continue the ordered generated runtime packets and independent semantic review,
required runtime lint/format/check, compact evidence and owned cleanup. Catalogue
timing no longer blocks them. Optional focused measurements retain controls and
sampling and are explicitly non-isolated/busy-host with observed activity under
the campaign-only waiver; historical strict quiet evidence remains separate. Assertion/artifact phase attribution remains a
separate prerequisite; intended guarantees do not by themselves prove that a
safe implementation improvement is impossible. An independent final queue pass
must find no ready supported fix before ending the campaign.

## Implementation priority update

Explicit user steering makes generated TypeScript execution the main lane and
waives quiet-host waiting only for this campaign. The unmeasured catalogue
comparison no longer blocks independent fixes. Completion descriptor construction
and duplicate map containment snapshots are the first source-supported packet;
follow generated-execution-plan.md and generated-execution-tasks.md. Broader
callback/invocation overhead must be reviewed on the actual generated async path,
followed by supported startup work. JSON token retention remains a secondary
candidate. All implementation and corrective edits belong to Muse.


## Queue reconciliation after forwarding acceptance

Intermediate review, not exhaustion: catalogue `1319ceda` correctness is accepted
and unmeasured; editor retains its separate historical isolated comparison.
Generated/runtime completion, snapshot, private callback/extraction/metadata,
static origin and proven authored forwarding candidates have verified fixes in
`7e0d5641`, `a7130e42`, `5b1889ee` and `4589122c`, with no current timing claim.
Startup loading/init attribution is the next [ordered packet](generated-startup-tasks.md);
per-site authored invoke remains a proof opportunity. Codecs retain the specific
numeric-only token candidate. Assertions require fresh roots and may inherit
supported startup improvements; artifacts still need phase attribution without
removing durability/integrity boundaries. Server has no isolated supported cause;
I/O keeps its demonstrated ownership copies; journeys follow shared generated/
codec candidates; browser retains behavior checks with no performance conclusion.
All twelve slices are dispositioned but the final independent queue review is
pending. Do not reinterpret old revision `40eaf6a88` as current measurements.

Domain catalogue reuse accepted52ae03f0: independent relevant contracts, runtime
check, strictTS/24 actual generated/native cases, exact emitted/driver identities
and both216-row69-summary comparisons verified; execution scratch retired.
Modules initialization observed2.926→2.742ms (~6.3% lower), domain statement
0.898→0.749ms; ordered busy-host observations limit attribution, repeated-factory
benefit unmeasured. One pre-existing stale browser export expectation is separately
dispositioned for narrow test correction. Next production packet:
[consumed numeric JSON evidence](generated-numeric-retention-tasks.md),
G37-G42. Auxiliary lane remains frozen; no new performance sampling requested.

Numeric JSON retention accepted12970524: independent42 tests/600 assertions,
runtime checks and64→4 source-reference replay passed; all numerical spellings and
parsed values preserved, no latency/heap claim. Stale browser export inventory
corrected test-only. Next source-supported generated packet is
[guarded absent-context authored calls](generated-context-bypass-tasks.md),
G43-G48, retaining generic invoke and all assertion-defined/fallback routes.
Final independent twelve-slice exhaustion review remains pending.
