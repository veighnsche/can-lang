# B05 — W3 reusable infrastructure final qualification (2026-09-26)

Source: R06/R07; W3; C-B. Owner: lane B (sole owner of legs
W3.1–W3.4). Machine record: [b05-legs.json](b05-legs.json).

Verdict: **BLOCKED** — by a lane-A emitter defect, not a conventions
failure. Six of eleven W3 legs fail at build time with `source map
encoding failed: exit status 1` (no assertion report produced) on
every project containing a lowered self-tail relay. The five legs
that avoid the lowered path still pass on the final tree. Per the
B05 shared-file boundary (compiler/runtime fixes stay with A/E
owners), B hands off and does not patch; no fixture was reshaped to
dodge the defect, and no W3 pass is claimed.

Environment: macOS darwin-arm64, go1.27.1, pinned bun 1.4.2
archive (`/tmp/bun-darwin-aarch64.zip`, sha256 verified
`90987a3a…be1d12f` against `distribution/target.json`). Base
`b283d2ab`. Command:

```sh
CAN_BUN_ARCHIVE=/tmp/bun-darwin-aarch64.zip go test ./tests/failure-conventions/ -v -count=1
```

## Final conventions (unchanged)

B03 (Q5) and B04 (Q4) are recorded INACTIVE, so there is no
follow-up surface: the final conventions are exactly the B01/B02
conventions in `tests/failure-conventions/` (README rules R1–R9,
F1–F5; experiment records `x-r06-1.md`, `x-r07-1.md` remain inputs,
not duplicate owners). No convention, fixture, or harness file
changed for B05: the qualification runs the committed harness
unmodified against the final tree.

## Leg results (5 pass / 6 blocked, one root cause)

| Leg | Result | Evidence |
|---|---|---|
| W3.1 helper + oracle | blocked | 35-root helper leg and over-attempt leg fail at build; no-retry mutant leg passes (11 oracle roots fail `unused fixture`, other 24 green) |
| W3.2 add-error isolation | blocked | result-data leg fails at build; fixed-style comparison leg passes (18 roots, unrelated files identical) |
| W3.3 extraction + costs | blocked | extraction-green and repair-green legs fail at build; factory-privacy leg passes and the guided half of the repair leg passes (`missing completion arm for ids::retired`) |
| W3.4 fail-closed + no forge | blocked | tainted-row negative fails at build; forge rejection passes at check time |

All six failures share one signature: `canlc assert` exits 1 with
empty stdout and `source map encoding failed: exit status 1` on
stderr. Nothing about the conventions regressed: every leg that
reaches assertion execution behaves as at B01/B02 time; the six
blocked legs never get that far because the build dies in
source-map encoding.

## Blocker characterization (handoff to the A surface owner)

Trigger: **any lowered self-tail relay under `canlc assert`**
(`build` runs the same encoding). The A04 while-lowering emission
places two `binding` mapping marks for one relay argument at a
single generated coordinate; `tools/runtime/source-map-validation.ts`
requires strictly increasing same-line columns and throws `invalid
mapping segment`. Pinpointed on the `retry` project: mark
`binding:1707:1716` (the `operation` argument at the self-relay
site) is emitted twice at the same `(line, col 0)`.

Bisection (each verified against a pristine bundle):

- `retry/` vs `retry-negative/` differ only by the self-relay lines
  in `src/retry/retry.can` (diff-verified): relay present fails,
  relay absent passes.
- `retry-fixed/` self-relays carry `when` fixture tables, so A04
  declines lowering (`CAN-CHECK-NOT-LOWERED: fixture table`) and the
  suite passes on the old emission path.
- Minimal self-relay countdown (M1, below) fails; the relay-free
  control (M0, same scaffold) passes.
- `owner-setup/` fails via the self-relay in `fixture_id`
  (`relay call fixture_id(1 / 0)`); its cross-function relays are
  not implicated.

Regression window: the W3 legs passed at B01/B02 time
(`dbafd204`/`7834c66f`, before A04 `78668ec7`) and fail at
`b283d2ab`, which includes A04 self-tail lowering. A07 stayed green
because `compiler/internal/emit/w4_a07_test.go` emits via
`RegionEmitter` directly and never runs `encodeSourceMaps` — the
invalid lowered mappings are invisible on that path. Suspect:
relay-argument rebinding marks in the lowering template
(`compiler/internal/emit/regions.go`).

Minimal repro (stage under a real, non-symlinked directory;
`canlc assert <dir>` reproduces):

`can.project.json`: `{"source_root":"src","error_registry":"can.errors.json"}`
`can.errors.json`: `{"active": [], "retired": []}`
`src/app/main.can`: the standard empty main (as in M0).
`src/m/m.can`:

```can
package m
    provides [countdown]
    uses []
fn int countdown
    emits []
    given
        int n
    asserts
        zero: 0 => ok 0
        two: 2 => ok 0
    match n > 0
        false => ok 0
        true => relay call countdown(n - 1)
```

Full repro sources are also embedded in `b05-legs.json`
(`blocker.minimalRepro`).

## Costs and limitations (final, carried from B01/B02)

Setup: 9 lines per factory input shape, 1 arm line per constructor
error; conversion premium +12 lines per domain (set declaration
plus consumer wrapper); per-layer repair at parity across styles
(3-line capture arm + 1-line re-raise arm); constructor evolution
touches the factory and handler contract only, leaving the
extracted helper byte-identical. Limitations: member-to-variant
inference workarounds (R4/R8), single-error domains skip the
variant, loader stubs are pure/total (live conversion belongs to
lanes E/F), the trap line reads poorly (a catalogue-level
primitive, not grammar, could address it), one owner shape
exercised. Full tables: `x-r06-1.md`, `x-r07-1.md`.

## Handoffs

- A surface owner: emitter fix for the duplicate lowered-relay
  mapping marks, plus a regression leg that runs lowered relay
  through `canlc assert`/`build` (the `RegionEmitter`-direct path
  cannot catch this class). M1 above is the minimal case.
- H11 (IC2): no W3 pass to aggregate yet; W3.1–W3.4 stay open
  behind the emitter fix. Rerun after the fix is the single command
  at the top of this report — no new harness is needed.
- Q4/Q5 gates: no new evidence either way; both stay INACTIVE on
  their recorded trip conditions.
