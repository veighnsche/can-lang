# H12 staging record — 2026-09-26

Status: **BLOCKED-awaiting-UP25-grant**. Staging complete; zero native
legs executed. No machine was probed (no SSH/tailscale contact — the
user designates the machine), no linux/amd64 binary or container ran
here, and no emulation was used anywhere.

Staging evidence: `evidence/2026-09-26/h12/h12-staging.md` (pins,
per-leg runbook, env contract, scenario rationale),
`evidence/2026-09-26/h12/jev/` (three fresh consultations, unanimous),
`distribution/h12/window-check.sh` (window-entry automation, `sh -n`
clean). Worktree base `4d67bdb6`, post-IC1 (`932924f7`).

## Exact machine + window ask (echo of distribution/x86-window.md)

1. Designate the machine: hostname/address + login user + which key or
   access path to use, and confirm it is Debian 13+ amd64/glibc.
2. Grant an exclusive window: single queue — no other jobs on the box
   during qualification; state start time and duration (or "on demand").
3. Reachability for the PG legs: either a `DATABASE_URL` to a live PG 17
   reachable from the UP25 host, or approval to natively install PG 17
   on the box during the window (isolated database).

## What runs the minute access lands

1. `sh distribution/h12/window-check.sh .` — READY verdict gates
   everything; any BLOCKED check stops the window.
2. Leg 1: `build.sh`, then `smoke.sh` + Go suite under outer
   `unshare -n` (BLOCKED, not degraded, if namespaces are denied).
3. Legs 5, 3 (driver/CAS/retention suites natively), Leg 6 (PG
   roundtrip once `DATABASE_URL` exists).
4. Legs 2, 4, 8: Playwright Chromium provision, drift rollout +
   rollback against the x86-built invoice-grid pair, retention
   transcript; Leg 7: supervised-process service/health/credentials +
   F04 operator-DDL, then full teardown.
5. Evidence lands under `evidence/2026-09-26/h12/` with revision, pins,
   commands, and per-leg pass/fail; W6-deploy verdict and lifecycle
   recipe hand off to H11/H13/H14.

## Live-probe policy

No unavailable-live pass: a leg that cannot run is BLOCKED with its
cause recorded, never green. No emulated claim: only native-x86
observations count as H12 evidence; C06's off-target results stay
supporting context and are never imported into the UP25 verdict.
