# H12 staging — qualify paired deployment on native x86 (UP25)

Staging-only turn. UP25 machine access is **not granted** (see
`distribution/x86-window.md`), so no native leg has run: this document
pins the candidates, gives every validation leg an executable procedure
with exact commands, env inputs, and pass/fail criteria, and records
what was verified here without execution. No linux/amd64 binary or
container was executed or emulated in this worktree.

Staged at worktree commit `4d67bdb6` (post-IC1 main). Design advice:
[H12 Jev findings](jev/findings.md) (outer `unshare -n` denial,
invoice-grid Playwright vehicle, supervised-process service posture;
unanimous, treated as advice, not proof).

## 1. Candidate versions pinned

| Item | Pin | Source |
| --- | --- | --- |
| IC1 join revision | main `932924f7` (worktree base `4d67bdb6` superset) | `execution-2026-09-26/h10-ic1-record.md` |
| Catalogue | sha `87c05b44978ff20cea35c8e67a427c535a1aab20ab15e041a0cfb9006894b99f`, 37 pkgs / 105 types / 299 ops / 110 errs | `evidence/2026-09-26/h10/h10-manifest.md` |
| C-H generation | `can-output-generation-v1`, 64-hex buildID; mismatch schema v1 | `distribution/paired-deploy.md`, H06 `1cf506e7` |
| C-G companion | carrier protocol v1, chart `d02.chart/1` | H10 manifest (F05 `b9930b47`) |
| Linux target | `bun-1.4.2-linux-amd64-v1` | `distribution/target-linux-amd64.json` |
| Bun archive | `bun-linux-x64.zip`, 36646985 bytes, sha256 `36368faef7527875d5ffa52e53cd48021741f2a83eb6208a8dd64068d422a913` | target file + `distribution/linux/provenance.json` |
| Bun executable | rev `744846f844374847c902b5e7fd59b4342a51ef99`, sha256 `a83d263767d839e4d2649ca8e35d07159c7afc99afdc96d731ced29e056dda0c` | provenance |
| Go toolchain | go1.27.1 (`go1.27.1.linux-amd64.tar.gz`, sha256 `63d339f0…168445`) | provenance |
| Host floor | Debian 13+ amd64, glibc (observed ref: trixie 13.7, glibc 2.41) | provenance, `x86-window.md` |
| PostgreSQL | `postgres:17`, server `17.x` (ref `17.11 Debian 17.11-1.pgdg13+2`) | provenance |
| Browser leg | Playwright 1.55.1, Chromium 140.0.7339.186 | provisioning register, `tests/integration/browser` |
| Old-browser app | invoice-grid (blocking prompt owner) | C06 report; §6 rationale |

## 2. What staging verified here (no execution)

- **Pinned archive availability.** Fetched the pinned URL once to
  `/tmp` (outside the worktree) and verified size + sha256 against the
  target file: exact match (`36368fae…a913`, 36646985 bytes). The zip
  was never unpacked and no binary from it ran. Staging needs no local
  copy: the runbook verifies `CAN_BUN_ARCHIVE` again on the UP25 host.
- **Go linux/amd64 compile.** `GOOS=linux GOARCH=amd64 go build` +
  `go vet` pass for `./distribution/` and `./tools/distbuild/` (the
  installer behind build/smoke). The full `canlc` link cannot
  cross-compile from this Mac: `compiler/internal/driver` depends
  transitively on the cgo tree-sitter package
  `compiler/internal/sql/sqlite`, and no Linux C toolchain exists here.
  This is a host limitation, not a product defect: the UP25 box
  provides gcc (window-check enforces it) and compiles natively.
  No cross-built binary was executed.
- **Shell syntax.** `sh -n` passes on `distribution/linux/build.sh`,
  `smoke.sh`, `run.sh`, `postgres.sh`,
  `distribution/provision-local.sh`, and the new
  `distribution/h12/window-check.sh`. (No `shellcheck` binary on this
  host; `sh -n` is the recorded gate.)
- **Procedure completeness.** Every leg below names an executable
  command, its env inputs, and pass/fail criteria; §3–§5 are the
  checklist.

## 3. Window-entry checklist automation

Run first on the UP25 host, before any leg:

```sh
cd <checkout>   # H12 candidate revision
sh distribution/h12/window-check.sh .
```

`window-check.sh` is inspection-only (prints PASS/BLOCKED/NOTE, never
secret values, nonzero exit on any block). It enforces the
`x86-window.md` checklist: `uname -m` = x86_64, Debian 13+,
glibc present, no qemu/docker-amd64/container surroundings, go1.27.1,
gcc, python3, `CAN_BUN_ARCHIVE` size+digest match, `unshare -n`
available, browser sources present, `DATABASE_URL` presence noted.
A BLOCKED verdict stops the window; nothing below runs until it is
READY (notes cleared before their affected legs).

## 4. Credential / env contract

Env names only, everywhere. Values live in process env on the UP25
host, never in the repo, register, runbook, or evidence.

| Name | Leg | Notes |
| --- | --- | --- |
| `CAN_BUN_ARCHIVE` | all build/smoke | absolute path to pinned `bun-linux-x64.zip` |
| `CAN_LINUX_INSTALL_ROOT` | PG roundtrip | install root from the smoke leg |
| `DATABASE_URL` | PG roundtrip, service, operator-DDL | live PG 17 reachable from UP25; value never printed |
| `CAN_FIREFOX_WS` | not used | Firefox stays off UP25; the old-browser leg is Chromium-only (§6) |

PG provisioning preference: (a) operator-provided `DATABASE_URL` to a
live PG 17 (least invasive, preferred); else (b) native install on the
box during the window (`apt` Debian PG 17 on an isolated database, user
approved). No Docker is required for any H12 leg. Secrets reach servers
only through spawned-process env; health/report artifacts carry
versions, digests, and counts — never credentials.

## 5. Per-leg runbook (all commands run on the UP25 host)

Conventions: `SRC` = H12 candidate checkout, `WORK` = fresh writable
dir (e.g. `/tmp/h12-work`), `VERSION=h12-1`. Every leg records revision,
pins, commands, env presence (names only), and pass/fail/skip.

### Leg 1 — package / install / network-denied smoke

```sh
export CAN_BUN_ARCHIVE=/path/to/bun-linux-x64.zip
# 1a. shell path (bare metal; no docker): build, then smoke offline
export SRC="$PWD" OUT="$WORK/out" WORK="$WORK" VERSION=h12-1 BUN_ARCHIVE="$CAN_BUN_ARCHIVE"
sh distribution/linux/build.sh
unshare -n sh distribution/linux/smoke.sh
# 1b. Go mirror of the same matrix (runs under the same outer namespace)
unshare -n env CAN_BUN_ARCHIVE="$CAN_BUN_ARCHIVE" \
  go test -count=1 -timeout 20m ./distribution/
unshare -n env CAN_BUN_ARCHIVE="$CAN_BUN_ARCHIVE" \
  go test -count=1 -timeout 50m -run TestLinuxInstalledArtifactSmoke -v ./tests/integration/
```

Pass: install selects through `current`; runtime-check reports Bun
1.4.2 / rev `744846f8…` / linux/x64 from the installed sidecar;
qualify.py passes APIs + behavior probes + nine rejection tests with
network `denied by Linux network namespace (unshare -n)`; process /
files / crypto / sqlite `canlc assert` + `run` outputs match
(`ok`, `file a.txt`, `match`/`mismatch`, `7: remember`); rebuild IDs
stable. Fail-closed: if `unshare -n` is unavailable the legs stay
BLOCKED (Jev: `outer_unshare`); never run them on the open host.

### Leg 2 — old-app typed mismatch + blocking refresh

Vehicle: `tests/integration/browser/drift.mjs` with `chromium` against
the x86-built invoice-grid pair (Jev: `grid_playwright`; §6).

```sh
# one-time browser provision inside the window (pinned, recorded)
(cd tests/integration/browser && bunx --bun playwright@1.55.1 install chromium)
# Go driver serves V1 -> harness boots page (READY) -> same port restarts
# on V2 (GO) -> stale action must 409 + prompt -> refresh resolves to V2
go test -count=1 -run 'TestC06ServedMatrix/drift' -v ./tests/integration/
```

If the C06 driver needs darwin-only paths on Linux, the fallback is the
same READY/GO rendezvous driven by shell: serve V1 `entry.ts` with the
installed sidecar, run `node drift.mjs chromium <base> <out> <script> <v1> <v2>`,
restart on V2, assert `report.json` 4/4 with zero limitations. Either
way the verdict comes from the harness report, not from imported C06
evidence. Pass: exact-409 refusal observed, blocking prompt rendered,
refresh resolves onto V2, 4/4 checks, no limitations — in both rollout
and rollback directions (V1/V2 swapped).

### Leg 3 — retained old assets

```sh
# after the Leg-2 rollout: replaced digest URLs from V1 must still serve
curl -fsS "$BASE/<v1-replaced-digest-route>" | cmp - <v1-expected-bytes>
go test -count=1 -run 'TestAssetRetainedBound' ./compiler/internal/driver/
```

Pass: old digest route returns 200 with byte-identical content while
old logic against the new server keeps failing per Leg 2 (bytes
retained, behavior refused).

### Leg 4 — rollout / rollback

Rollout is Leg 2's V1→V2 direction; rollback rebuilds the prior source
and reselects through the same atomic path (stable build IDs reproduce
the byte-identical generation — no retained stale trees):

```sh
canlc build --target browser <grid-project>            # browser gen V1'
canlc build --browser-manifest <V1'/manifest> <grid-project>  # pairs, selects
# serve V1', run the drift rollback direction (V2 page -> V1' server)
go test -count=1 -run 'TestC06ServedMatrix/drift.*rollback' -v ./tests/integration/
```

Pass: rebuilt V1' buildID equals V1; selection atomic (prior current
preserved on any failure); rollback drift leg 4/4; retention covers
replaced routes in both directions.

### Leg 5 — CAS GC / read-only safety

```sh
go test -count=1 -run 'TestPairedHandshakeIdentity|TestStageSharesIdenticalBytes|TestBreakLinkTamperIsolatesGenerations|TestPruneCollectsUnreferencedStore|TestMetadataWritesAreUnique' -v ./compiler/internal/driver/
```

Pass: pairing identity chain holds; identical bytes share an inode
(read-only `0400`, no full-copy staging); unlink+recreate tamper
isolates generations; prune deletes non-current generations and sweeps
unreferenced store entries (never current); manifests/metadata are
unique `0600` writes.

### Leg 6 — live PG roundtrip

```sh
# WORK must still point at the Leg-1 dir: postgres.sh resolves canlc/bun via $WORK/install-root
export CAN_LINUX_INSTALL_ROOT="$WORK/install-root"   # from Leg 1 (used by Leg 7)
export DATABASE_URL='<operator-provided, never recorded>'
sh distribution/linux/postgres.sh
go test -count=1 -run TestLinuxPostgresRoundtrip -v ./tests/integration/
```

Pass: `serverVersion` starts with `17.`, roundtrip reads `7: remember`;
the Go leg and `postgres.sh` agree.

### Leg 7 — service / health / credentials / operator-DDL

Posture: runbook-supervised foreground processes, env-only secrets,
generation-stamped health, full teardown (Jev: `supervised_process`; no
systemd unit is installed on the user machine).

```sh
# serve the selected generation; secrets via spawn env only
CANLC="$CAN_LINUX_INSTALL_ROOT/current/bin/canlc"
GEN_DIR="$("$CANLC" build <grid-project> | python3 -c 'import json,sys; print(json.load(sys.stdin)["directory"])')"
GEN_ID="$(basename "$GEN_DIR")"
DATABASE_URL="$DATABASE_URL" "$CAN_LINUX_INSTALL_ROOT/current/runtime/bun" "$GEN_DIR/entry.ts" &
SRV=$!; trap "kill $SRV" EXIT
# health: served page stamps the expected generation + one action passes
curl -fsS http://127.0.0.1:<port>/invoice-grid | grep -q "data-can-generation=\"$GEN_ID\""
<one grid action via curl with can-generation: $GEN_ID>  # expect 200
# operator-DDL recipe (F04) against the live PG, then verify
bun -e 'import("./runtime/outbound/ledger-schema.ts").then(m => console.log(m.ledgerDDL(process.argv[2])))' -- postgres > "$WORK/ledger-pg.sql"
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f "$WORK/ledger-pg.sql"
psql "$DATABASE_URL" -c "SELECT version FROM ai_budget_state WHERE id = 1;"  # exactly one row: 0
kill $SRV; trap - EXIT
```

Pass: health requires the current `data-can-generation` plus a 200
action (a wrong-generation or ghost process cannot satisfy it);
credentials appear in no artifact; the F04 script applies cleanly and
the seed/check probes behave per
`evidence/2026-09-26/f04/f04-operator-ddl-recipe.md`; teardown leaves
no server process, no unit files, no residue.

### Leg 8 — old-browser scenario (acceptance wrapper)

Leg 2 is the scenario; this leg is its acceptance record: the drift
rollout + rollback reports, screenshots, served-generation transcripts,
and the Leg-3 retention transcript, all executed on UP25 in this
window. Selection and rationale: §6. Live-probe policy: no
unavailable-live pass (a leg that cannot run is BLOCKED with cause,
never green), no emulated claim (anything not native-x86 is not H12
evidence).

## 6. Old-browser scenario selection + rationale

**Selected: invoice-grid driven by pinned Playwright Chromium on the
UP25 host (Leg 2), with header replay (Legs 2–3 curl steps) as
supporting — never substitute — evidence.**

- Grid owns the blocking prompt: C06 implemented `stale_state`
  (prompt survives edits, blocks saves/replays, resolves through
  observed agreement or refresh) plus phase-branching failure arms in
  `send_save`/`load_pressed`; grid asserts 394/394.
- Compare is correctly excluded as a vehicle: it makes no server
  calls by design, so no mismatch path exists there; running the
  scenario against it would assert nothing.
- The htmx form page is excluded as the prompt vehicle: it refuses
  with the exact 409 and never swaps/retries, but renders no prompt
  (follow-up, no Can browser code on that page). Its refusal-only
  behavior is already covered honestly by header replay.
- Chromium-only on UP25: the full three-browser matrix is C06's
  (owned elsewhere); H12 needs one genuine browser execution of the
  prompt on the native pair, and Chromium 140 via pinned Playwright
  1.55.1 is the lightest honest provision. Firefox/WebKit are not
  claimed on UP25.
- No split claim: C06's Mac-matrix prompt observations are not
  imported into the UP25 verdict — the prompt must render from
  x86-built bundles during the window (Jev unanimous; `split_claim`
  rejected at ≤0.02 in every round).

## 7. Staging completeness per H12 validation bullet

| H12 bullet | Staged procedure | Status |
| --- | --- | --- |
| Network-denied smoke | Leg 1 (shell + Go under outer `unshare -n`) | staged, BLOCKED on window |
| Old-app typed mismatch / blocking refresh | Leg 2 (drift.mjs chromium, both directions) | staged, BLOCKED on window |
| Retained old assets | Leg 3 (curl 200s + `TestAssetRetainedBound`) | staged, BLOCKED on window |
| Rollout / rollback | Leg 4 (rebuild-stable + drift rollback) | staged, BLOCKED on window |
| CAS GC / read-only safety | Leg 5 (driver suite natively) | staged, BLOCKED on window |
| Live PG roundtrip | Leg 6 (`postgres.sh` + Go leg) | staged, BLOCKED on window + `DATABASE_URL` |
| Service / health / credentials / operator-DDL | Leg 7 (supervised process + F04 recipe) | staged, BLOCKED on window + `DATABASE_URL` |
| Old-browser scenario | Leg 8 (Leg-2 vehicle + rationale §6) | staged, BLOCKED on window |
| Exclusive native x86 run | window-check.sh gate (§3) | staged, BLOCKED on grant |

Zero x86 execution or emulation happened in this worktree: the only
linux/amd64 artifacts touched were the pinned zip's size+digest
(fetch-verify only, kept outside the worktree) and unexecuted
cross-compile outputs (removed). `git status` shows no new binaries.
