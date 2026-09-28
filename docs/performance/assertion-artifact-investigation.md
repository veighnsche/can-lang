# Assertion/artifact investigation (I2, read-only)

Source revision `318ed458`. No implementation, measurement, build, or
source edit was performed for this note. Timings are the saved historical
run `.performance/20260927T235706.371974Z` (report.md plus per-root
distributions recomputed read-only from `raw/assertions-001.json`); older
source revision, hypothesis input only.

## Saved timings against the responsible paths

Assertions (`compiler/perfmeasure/main.go` `supervised-roots` work, jobs as
noted; compilation/staging excluded, one `RunSupervised` per op):

| Case | Median | Roots/files/bytes (saved metrics) |
|---|---|---|
| flat supervised-roots, jobs=1 | 5338.88 ms/op | 101 roots, 343 files, 16,295,037 B |
| utilities jobs-1 | 321.065 ms/op | 8 roots, 157 files, 2,890,071 B |
| utilities jobs-4 | 165.339 ms/op | same workload, 4 workers |

Saved `root_wall_ms_by_sample` (supervisor `elapsedMs` per root, whole-
generation validation excluded): flat median 45 ms/root (42–96, n=707);
utilities jobs-1 median 33 ms/root (32–37, n=56); utilities jobs-4 median
57 ms/root (51–67, n=56).

Artifacts (flat-call-chain, size 100):

| Case | Median | Scope |
|---|---|---|
| publish-new | 721.479 ms/op | fresh store: payload writes, CAS storage, manifest, selection |
| publish-reuse | 51.9284 ms/op | identical generation: integrity checks + selection, no payload writes |
| validate | 126.689 ms/op | PrepareOutput hashing + native TS transpile validation |
| source-maps | 15.2741 ms/op | span assembly + maps encoding/validation |

## Supervisor/launcher walk (intended costs)

`RunSupervised` (`compiler/internal/driver/supervise.go`): one
`validateLease` per op (runtime-manifest check, output-manifest check,
symlink-ancestor refusal, `OpenRoot`, manifest same-file check, one full
`validateOutputTree` re-read + re-hash), then `runRootsOrdered` launches
each root in its own worker: `prepareEntry`
(`compiler/internal/driver/runtime.go`: `MkdirTemp`, four `0700` mkdirs,
environment JSON marshal, fd-3 pipe, fresh `bun` process with allowlisted
env and leased generation fd) → `begin`/wait → `deliveredRoot` (single
report parse, one `lastProgress` stderr scan; `workerDiagnostic` scans
stderr a second time only for undelivered roots — zero in passing suites)
with timeout-wins-at-equality, kill + reap confirmation, ordered entries,
and cancellation on first genuine failure.

Attribution from saved data: flat 101 roots × 45 ms ≈ 4545 ms of the 5339
ms total, leaving ~800 ms for the one suite-level tree validation over
343 files/16 MB plus orchestration. Per-root cost is one process launch +
one real harness by construction. The 101-root population and the fresh
process per root are intended isolation obligations (fresh harness,
external wall-time budget, reaping, ordered delivery, deadlines,
cancellation); neither is reducible as a "gain". jobs-1 → jobs-4 halves
the utilities total (1.94x) while per-root medians rise 33 → 57 ms:
parallel-launch contention, not duplicate work — and worker pooling is
explicitly out of scope regardless.

Duplicate-work audit of the supervisor: none found. No repeated
validation, no repeated launch, no repeated parse on any path; the only
double scan is conditional on delivery failure.

## Output store / validation / CAS walk (intended costs)

`PrepareOutput` (`artifacts.go`): per-artifact path/collision checks and
one SHA-256 per payload into the manifest. `ValidateOutput`
(`output_runtime.go`): copies every module source into a JSON request,
marshals it, launches the `output-check.ts` bun tool for native
transpilation + relationship checks, then re-hashes the whole request to
bind `requestSHA256` to the delivered report. The 126.7 ms `validate`
case runs `PrepareOutput` + `ValidateOutput` per op.

`Publish` = `Stage` + `SelectCurrent` (`output.go`). New generation:
`Recover`, per-file `hashBytes` + manifest-digest compare,
`stageContentFile` → `ensureCAS` (`output_cas.go`: digest re-verify,
present-entry re-read + re-hash with tamper replacement, temp write +
`Sync` + rename, hard link or private read-only copy), `validateOutputTree`
(full re-read + re-hash of the staged tree), `syncOutputTree`, rename,
directory syncs; then `SelectCurrent`: freshness + layout rechecks,
`generation()` (second full re-read + re-hash), atomic `current.json`
write + sync. Reused generation: `Stage` finds the final tree and runs
`generation()` (full walk + re-hash), then `SelectCurrent` runs
`generation()` again (second full walk + re-hash) plus freshness/layout
checks and the `current.json` write + sync — the 51.9 ms with zero
payload writes.

## Duplicate-work candidates and why none is currently supported

1. Payload bytes are hashed about four times on publish-new
   (PrepareOutput manifest, Stage per-file, ensureCAS verify,
   validateOutputTree, then once more via SelectCurrent). Each hash
   guards a different boundary (inventory binding, staging integrity,
   store tamper repair, tree tamper refusal, pre-selection TOCTOU).
2. The full tree is re-read + re-hashed twice per `Publish` (Stage and
   SelectCurrent). The second check is an explicit TOCTOU guard:
   `Publish` takes no lease, so files can change between the two
   checks; only the pre-selection validation closes that window while
   keeping atomic selection and tamper/symlink refusal.
3. `ensureCAS` re-reads + re-hashes present entries on every stage.
   This is the tamper-repair obligation (a replaced entry is rebuilt
   from just-hashed bytes); skipping it trusts the store blindly.
4. `ValidateOutput` copies all sources into one JSON request and
   re-hashes the request. The copy is structural to the out-of-process
   validator; the re-hash binds the delivered report to the exact
   request bytes (tamper evidence), not the validator's word alone.

Items 1–4 are concrete repeated traversals, but every repetition is
currently attached to a stated durability, atomicity, or integrity
obligation (fsync per payload, atomic selection, lease checks,
tamper/symlink refusal, closed inventory). The saved evidence does not
separate obligated bytes (writes + fsyncs + bun launches + transpile)
from check overhead, so removing or merging any check is not supported:
it would trade an unmeasured saving against a named guarantee. In
particular: no fsync removal, no validation skip, no CAS trust-me flag,
no worker pooling, no publication rewrite is proposed here.

## Bounded attribution plan (for Codex supervision, not this packet)

1. Split suite vs root: time `validateLease` alone on the staged
   flat generation, then 1-root vs N-root `RunSupervised` scaling at
   jobs=1. Predicts the ~800 ms suite remainder vs ~45 ms/root slope.
2. Split launch vs harness: time an empty-worker launch through the
   production `prepareEntry` boundary vs one real root. Predicts how
   much of the 45 ms is bun startup (unfixable by design) vs harness.
3. Split publish phases: time `Stage`-only, `SelectCurrent`-only, and
   `generation()` revalidation on the flat generation to size items
   1–2 above against write+fsync cost. Only if a check dominates its
   phase does a check-preserving redesign (e.g. a held lease across
   Stage+Select, keeping both guarantees with one walk) become worth
   a design consultation.
4. jobs-4 contention cause is already bounded by the saved medians
   (33 → 57 ms/root under parallelism); no new exchange needed unless
   a launch-shape change is proposed, which this note does not propose.

Conclusion: current evidence supports no safe fix in the
assertion/artifact lanes. The measured costs trace to intended root
population, fresh-process isolation, fsync durability, atomic selection,
and revalidation integrity. Items 1–3 above are the minimum evidence
before any candidate in these lanes may be promoted.
