# Required design consultation

Status: **complete; three fresh responses saved and reconciled**.

The repository's AGENTS.md requires three fresh Jev consultations for difficult
design choices. The [TypeSafe skill](../../../../.agents/skills/typesafe-ai/SKILL.md)
was applied. Live [API](https://docs.typesafe.ai/api),
[Choice](https://docs.typesafe.ai/primitives/choice), and
[consistency guidance](https://docs.typesafe.ai/cookbooks/consistency_choice_cookbook)
were read. Jev is a classifier supplied with evidence and alternatives, not asked
to research or generate a design. Its answers are advisory, not proof.

Saved inputs: [request 1](request-1.json), [request 2](request-2.json),
[request 3](request-3.json), [pre-send wording/semantic audit](wording-audit.json).
Every state paragraph, question and option description is rewritten; technical
identifiers, evidence, constraints and alternatives are equivalent. A separate
read-only agent rechecked equivalence and the open-source authorization wording.
The exact pre-audited payload hashes were checked again before transmission.

Responses: [1](response-1.json), [2](response-2.json), [3](response-3.json).
All identify `jev-1.13.0`; the requested alias was `jev-latest`.

| Decision | Response 1 | Response 2 | Response 3 |
| --- | --- | --- | --- |
| Browser launcher | `driver_launch` 0.90 | `driver_launch` 0.73 | `driver_launch` 0.97 |
| Compiled-Can qualification order | `gated_lane` 0.76 | `gated_lane` 0.81 | `gated_lane` 0.96 |

No selected-answer disagreement occurred. Agreement does not remove bias or
establish executable correctness. The responses supply no reasoning explanation;
the following reconciliation is our analysis of the stated evidence/alternatives.

For the browser decision, response 2 assigns 0.14 to direct launch and 0.13 to
an isolated host, with confidence 0.59. These are real alternatives: direct
launch gives the native owner immediate process authority but requires auditing
and maintaining the complete driver-equivalent startup recipe; a separately
isolated host can provide stronger host-effect boundaries but needs independent
provisioning and qualification. Retain pinned driver launch inside N's owned
service as the first implementation candidate because it reuses the installed
platform defaults missed by the failed probe. Still qualify acquisition before
effects, service-death cleanup and the independent host-UI observation boundary.
If those guarantees cannot be established, the selected host is unavailable and
the direct/isolated alternatives must be reconsidered explicitly. No launch is
admitted merely because Jev preferred it.

For ordering, responses 1 and 2 assign 0.24 and 0.18 to doing another compiled-Can
probe before planning. That alternative reduces uncertainty earlier, but the
probe itself needs the missing compiler artifact, typed entry and external owner.
Writing the dependency plan can expose and isolate exactly that work without
claiming it succeeds. Keep compiled ingress and independent invocation controls
in an early acceptance lane; block every dependent migration if they fail.
Preparatory source imports still do not qualify generated Can. There is no new
evidence warranting an interpreter decision before planning those gates.

The questions compare browser launch ownership/configuration approaches and the
placement of compiled-Can ingress qualification relative to implementation
planning. Payloads contain project design/probe summaries, limits and remaining
gates, with no credential value, personal profile contents or raw project source.
Authentication used the existing key only in its header, without printing or
saving it. The destination was `https://api.typesafe.ai/v1/systemone`.

## Transmission history and authorization

Three initial sandbox attempts failed DNS resolution; their bounded failures
remain as `request-1-failure.json` through `request-3-failure.json`. Automatic
approval review rejected the first escalated attempt before process creation
because it treated repository-derived context as requiring explicit disclosure
approval. No workaround was attempted.

The user then clarified: “it's open source... nothing is proprietary ... please
put that in the AGENTS.md”. [AGENTS.md](../../../../AGENTS.md) now records that
relevant project source/design context is non-confidential and authorized for
these Jev consultations, while excluding credentials and unrelated personal data.
The subsequent reviewed transmission succeeded for all three unchanged payloads.
The prior restriction is resolved; no response is fabricated and no remaining
consultation approval is required. This does not change any integrated runtime
qualification or harness-retirement status.
