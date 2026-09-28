# Numeric JSON evidence production packet (G37-G42)

Design: [generated-numeric-retention-plan.md](generated-numeric-retention-plan.md).
Authoritative mutable coordinator/goal/viewer/cleanup state:
../../.performance/performance-push-20260928/muse-run-10-owner.json.
Previous production checkpoint52ae03f0 accepted; auxiliary tool lane is frozen.
One SAME sole Muse Contributor/MAX coordinator; no additional agents needed.
Native get_goal then matching reuse/create_goal before work; report_progress tracks
evidence, update_goal complete only at reviewable ALL-writer handoff. No goal budget.
Unavailable goal tools are a blocker, never a prose substitute.

- [x] **G37 — confirm consumed evidence and functional before probe**
  - Prerequisite: native goal, design and three consultations read.
  - Owner: Muse coordinator; read-only source/probe, this checklist progress.
  - Action: confirm listed complete consumers. Fixed object:40 strings,10 booleans,
    10 nulls,4 numbers spelled9007199254740993,1e309,-0,1e2. Save compact native
    parsed values and holder token-entry count before edit; no timing/heap estimate.
  - Acceptance: numeric precision source retained; current unused entries proved;
    no auxiliary tool edits or measuring-device work. Evidence: met: exactInt
    number-guard, raw numeric-only lookup, AI parsed-only verified in source;
    before-probe 64 entries (60 unread), 4 numeric spellings verbatim, parsed
    exact. Record in `generated-packet-7-muse-evidence.json` (g37).

- [x] **G38 — omit unread nonnumeric token retention**
  - Prerequisite: G37 proof; owner: sole coordinator, runtime/codec/document.ts only.
  - Action: gate source insertion on native number type, update truthful comment;
    preserve unconditional rootHolder update, all parser/refusal behavior and consumers.
  - Acceptance: every numeric source survives; no string/bool/null Map entry or
    holder allocation solely for those values. No modes/API/parser changes. Evidence: met:
    `runtime/codec/document.ts` `589a0eb9` number-type gate + truthful comment;
    rootHolder/parser/refusal behavior unchanged (g38).

- [x] **G39 — pin numerical and consumer semantics**
  - Prerequisite: G38; owner: coordinator, new runtime/codec/document.test.ts;
    runtime/test/codec-json.test.ts or raw-provider.test.ts only if missing behavior
    needs a small addition. Do not rewrite existing suites.
  - Action: meaningful root/holder/array/empty-key/nonnumeric parsing and original
    numeric evidence checks; reuse exact integer, overflow,negative-zero,Unicode,
    duplicate/syntax precedence, JSONL and raw assertion number-spelling tests.
  - Acceptance: negative and edge cases fail for real semantic regressions;
    no implementation-mirroring counter tests. Evidence: met: new
    `runtime/codec/document.test.ts` 6/6 pass 39 expects (roots, holders,
    arrays, empty/repeated names, overflow/-0/spellings, precedence); no
    additions to existing suites needed (g39).

- [x] **G40 — correct stale browser expectation and qualify dependents**
  - Prerequisite: G39; owner: coordinator, runtime/test/browser-profile.test.ts
    expected owner inventory and ambient bindNativeCallback refusal only.
  - Action: account for actual existing export in both hosts, test browser failure
    with a callback. Run lint:fix:runtime,format:runtime,check:runtime then focused
    document/codec-json/codec-numbers/formats/raw-provider/browser-profile/questions/
    responses/ai-budget-adapters tests sequentially with bounded command timeouts.
  - Acceptance: no production owner edits, no skipped failures. Explain generated
    fixture dependency continuity; reuse accepted exact module/driver/source-map/
    strictTS/24 oracle evidence where this change is unreachable. If relevant new
    generated path exists, one bounded graph suffices; no broad repeat/tool suite.
    Evidence: met: browser-profile test-only fix (19-name surface, callback
    refusal), owners untouched; lint/format/check exit 0; 63 focused tests
    pass 0 fail; generated+runtime suites contain no JSON decode calls so
    accepted 52ae03f0 strictTS/24 evidence reused, no new graph (g40).

- [x] **G41 — record work removal and honest limits**
  - Prerequisite: G40; owner: coordinator, compact evidence and checklist.
  - Action: repeat SAME functional probe, preserve original numeric tokens and
    parsed values, record actual before/after entry counts plus source hashes.
  - Acceptance: no latency/heap/p95 claim, no sampling; fixtures/driver identities
    and qualified evidence continuity explicit. No retained execution graphs. Evidence: met:
    after-probe 4 entries (was 64), parsed/spellings identical; 60 unread
    entries removed, source-proven; no latency/heap claim (g41).

- [x] **G42 — release all writers and hand off**
  - Prerequisite: G37-G41; owner: coordinator, checklist and
    generated-packet-7-muse-evidence.json.
  - Action: actual commands/results/times/changes/limitations/cleanup and native
    goal lifecycle; explicit ALL-writer release, goal completion, idle TUI for review.
    Leave Codex tasks unticked; no commits or exhaustion claim. Evidence: met
    2026-09-28T20:03Z: changed document.ts, document.test.ts,
    browser-profile.test.ts, this checklist, packet-7 evidence; probe retired;
    no residue. Terminal handoff below. ALL writers released; TUI idle.
    Native goal completion is in the authoritative runtime monitor.

## Muse terminal handoff (G37-G42 numeric retention, 2026-09-28T20:03Z)

Sole Contributor/MAX/YOLO coordinator, same TUI, no extra executor.
Gated token retention on numeric type in `runtime/codec/document.ts`
(`589a0eb9`): fixed 64-member probe drops 64 to 4 retained entries with
identical parsed values and all 4 numeric spellings verbatim, including
overflow Infinity and -0. New `document.test.ts` 6/6; browser stale
inventory corrected test-only (19 names, callback refusal), owners
untouched. Gates green; 63 focused consumer tests pass; generated 24
cases proven not to execute JSON decoding so accepted strictTS/24
evidence stands with no new graph. No measurements, no tool edits, no
latency/heap claim. R7/Q7 left for Codex.

- [x] **R7 — independently review, verify applicable contracts and commit**
  - Owner: Codex after explicit writer release; actual diff/source/probe/consumer
    checks, reuse successful identities appropriately, promptly commit exact paths.
- [ ] **Q7 — finish remaining production decisions and final twelve-slice review**
  - Owner: Codex; no invented fixes or auxiliary expansion. Campaign remains active.

Muse long-command cadence: native bash initialyield120000ms, routine1-5min/default2,
retain handles/deadlines, await delivered completion; no empty polling/goal resets.
Register exact temporary cleanup immediately; retire groups before deletion,
protect active/foreign/replaced resources. Installed Bun/shared Go cache only,
no installs/copies/private caches/extra worktrees/bundles. Report cleanup failures.

## Independent acceptance (Codex, 2026-09-28)

Actual diff and complete consumer audit accepted.42 focused actual consumer tests/
600 assertions passed, zero failures/skips; runtime lint/format/typecheck passed.
Independent fixed-object replay of the old all-primitive reviver confirms64→4
retained entries, identical parsed values and every original numeric spelling.
This replay validates work removal; it does not recreate historical timing.
Owner implementations unchanged, stale browser expectation repaired with a real
callback refusal. Compiler/fixture/driver identities match52ae03f0; generated24
workloads do not decode JSON, so prior strictTS/actual oracle evidence is reused.
Changed document/browser test runtime hashes are disclosed, never called equal.
Probe absent; no independent temporary graph allocated. No latency/heap claim.
Evidence: generated-packet-7-independent-review.json.
