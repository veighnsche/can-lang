# H12 Jev consultations: paired-deploy staging decisions

26 September 2026. Advisory evidence for task H12 staging (paired-deploy
qualification on native x86, R13 with F-R13-01b/02/03, W6-deploy, C-H).
Three fresh live requests to `jev-latest` (all answered by `jev-1.13.0`)
cover the three hardest H12 staging decisions below. Every explanatory
state field, instruction, and option description was rewritten in full
across the three rounds (30 fields: 3 state fields, 3 instructions, and
9 option descriptions, each in 3 full phrasings); task IDs, requirement
IDs, file paths, kind strings, function names, test names, env names,
the 409 shape, `0400`/`0600` modes, the seven-day bound, and code
spellings stayed exact. No request contains any prior Jev answer. The
[TypeSafe skill](../../../../../../.agents/skills/typesafe-ai/SKILL.md)
was followed. Jev performed no research and returned typed choices,
distributions, and confidence values, not prose reasons.

## Why these three decisions

The H12 scope sentence fixes *what* (package/install/smoke/PG roundtrip/
credentials/health/retained assets/old browser/rollout/rollback on UP25)
and the window checklist fixes *where* (native Debian 13+ amd64/glibc,
no emulation). What remains genuinely open in staging is *how the
runbook earns each leg on a bare-metal box the project does not own*:
(1) **netdenial** — the T19 scripts assume `docker run --network none`,
yet the window prefers no Docker in the qualification path, so the
offline boundary needs a bare-metal mechanism with independent
verification; (2) **browserscenario** — C06 scoped the blocking prompt
to invoice-grid and left compare/header-only paths that tempt cheaper
vehicles which skip the prompt; (3) **servicehealth** — lifecycle legs
need a serving pair on borrowed hardware, where a real unit is invasive
and no service at all leaves F-R13-02 unexercised. Each question carries
at least one superficially cheaper alternative (container reuse, replay
or split claims, systemd or no service), which is what makes the choice
worth an outside judgment.

## Exact evidence

| Round | Submitted request | Saved response |
| --- | --- | --- |
| 1 | [request 1](request-1.json) | [response 1](response-1.json) |
| 2 | [request 2](request-2.json) | [response 2](response-2.json) |
| 3 | [request 3](request-3.json) | [response 3](response-3.json) |

[consult.py](consult.py) reproduces the requests byte-for-byte and sent
exactly three API calls with no retry; `--send` refuses to run when any
shared 8-gram violation is present (none were). The
[wording audit](wording-audit.json) lists the preserved fact sets, and
[summary.json](summary.json) records selections, probabilities, and
usage. Reported usage totals **6,531 input and 450 output tokens**.
Credentials were supplied through the authorization header and are absent
from saved evidence.

## Results

Selected probability is the model's probability for that candidate, not
the probability that the design is correct. Confidence is a separate
provider value. Order is round 1 / 2 / 3.

| Decision | Selected in every round | Probability | Confidence |
| --- | --- | --- | --- |
| Network denial | `outer_unshare` | 1.00 / 1.00 / 1.00 | 0.99 / 1.00 / 1.00 |
| Old-browser vehicle | `grid_playwright` | 0.96 / 0.95 / 1.00 | 0.95 / 0.93 / 0.99 |
| Service posture | `supervised_process` | 0.99 / 0.80 / 0.97 | 0.98 / 0.70 / 0.95 |

## Disagreement audit

No inter-round selection disagreement: all three rounds select the same
option on all three questions. One softness investigated: round 2 puts
0.19 probability on `no_service` (confidence 0.70) for the service
question. Round-2 phrasings carry the same facts as rounds 1/3 (env-only
secrets, health tied to the served generation, teardown leaving no init
state), and the round-2 `no_service` text states the same trap (lifecycle
left theoretical); the selection still lands on `supervised_process` at
0.80. Treated as wording noise, not a substantive split: the runbook
adopts supervised processes with explicit teardown, which is also the
only option consistent with the no-leftover-state constraint on borrowed
hardware. Agreement is advice, not proof: each adopted shape still has
to pass its live leg during the window.

## Adopted staging consequences

- Offline legs run under one runbook-owned `unshare -n` wrapper;
  qualify.py's live-interface verification is the independent check;
  hosts that forbid user namespaces leave offline legs BLOCKED.
- The old-browser scenario drives invoice-grid in pinned Playwright
  Chromium on the UP25 box; header replay covers refusal/retention as
  supporting legs, never as the prompt vehicle; no C06 observation is
  imported into the UP25 verdict.
- Lifecycle legs serve generations as runbook-supervised processes with
  env-only secrets, generation-stamped health, and full teardown; no
  systemd unit is installed on the user machine.
