# Evidence for the consolidated Can recommendation program

24 September 2026 · reviewed source revision `4c2db1e1162a45e88a58ef154e848790a1a44c60`

The [consolidated program](/Users/vince/Projects/can-lang/docs/syntax-taste/can-recommendation-program-2026-09-24.md) merges two separate reports, without treating overlap as two feature requests:

- [SaaS example review](/Users/vince/Projects/can-lang/docs/syntax-taste/saas-language-review-2026-09-24.md) and its [inventory, three consultations and checks](/Users/vince/Projects/can-lang/docs/syntax-taste/evidence/2026-09-24/saas-review/README.md).
- [Independent clean-room language review](/Users/vince/Projects/can-lang/docs/syntax-taste/clean-room-language-review-2026-09-24.md) and its [three source audits, probes, three consultations and checks](/Users/vince/Projects/can-lang/docs/syntax-taste/evidence/2026-09-24/clean-room-full-review/README.md).

The consolidation rechecked selected source facts: current stream cleanup in `examples/stream/src/main.can`; the default HTMX status policy in `runtime/platform/html.ts`; the sole qualified distribution target in `distribution/target.json`; and the earlier September 22 design dispositions. It introduces no new compiler/runtime behavior claim requiring a live provider or database run. The crosswalk and independent architectural critique were performed after the two reports were complete; they are synthesis checks, not fresh blind source reviews.

## Fresh Jev consultation procedure

The [TypeSafe skill](/Users/vince/Projects/can-lang/.agents/skills/typesafe-ai/SKILL.md) requires current official [documentation index](https://docs.typesafe.ai/llms.txt), [API](https://docs.typesafe.ai/api.md), and [Choice](https://docs.typesafe.ai/primitives/choice.md) guidance. The index was checked again for this turn; API/Choice contract information was available from the immediately preceding review and its saved integration script. Jev is a typed classifier. It received the factual context in the request and did not research the repository.

`consult.py` prepared all three requests before any was sent. It rewords **every explanatory state field, question instruction and option description** while keeping the observations, constraints and alternatives semantically aligned. `wording-audit.json` records the mechanical alignment checks; the coordinator compared the complete triples for factual equivalence, scope and wording variation. The three requests are distinct, use the same question/choice IDs, and contain no earlier answers. Rewording cannot guarantee removal of framing effects.

The exact request/response pairs are `request-1.json`/`response-1.json` through `request-3.json`/`response-3.json`, with timing/status metadata. The API returned `jev-1.13.0` for the `jev-latest` alias. Total usage was 4,658 input tokens and 801 output tokens. No credential is recorded.

The table shows the selected choice and its probability; raw files preserve all alternative probabilities and confidence values. These probabilities are model outputs, not empirical success rates.

| Decision question | Request 1 | Request 2 | Request 3 |
| --- | --- | --- | --- |
| Pattern intent | explicit binding 0.88 | explicit binding 0.72 | explicit binding 0.52 |
| Validated value boundary | owner control 0.86 | owner control 0.87 | owner control 0.85 |
| Fixture scenarios | explicit seams 0.89 | explicit seams 0.95 | explicit seams 0.92 |
| Generic errors | measure then trial finite rows 0.91 | measure then trial 0.70 | measure then trial 0.99 |
| Frontend path | typed external client 0.70 | server then Can browser 0.91 | server then Can browser 0.91 |

### Disagreement investigation

The frontend question is genuinely sensitive to framing and scope. Request 1 selected a typed existing-browser client; it is a practical way to recommend Can **as a server language** in a full application, with Can-to-client wire contracts. It does not satisfy the user's stated ambition that Can itself be the recommended *frontend* language too. Requests 2 and 3 selected a staged Can browser target after server qualification. The consolidated program chooses that staged path for the broader user goal, while retaining the external-client bridge as a narrower supported integration. The selection is an engineering judgment, not a 2-to-1 vote.

The previous clean-room consultations had split on owner-controlled authored values versus catalogue-only opacity. All three consolidation requests favored owner control when the question explicitly targeted independently reusable validated business values. This shift is evidence of context sensitivity, not proof that one design is correct. The program therefore requires an email/quantity/tenant-value prototype that covers decoding, equality, tests and updates before selecting syntax or changing the authoritative specification.

Earlier September 22 dispositions had retained ambiguous bare patterns and deferred symbolic scenario seams. New executable probes in the clean-room evidence changed the factual basis for considering those decisions. Jev's agreement does not establish a grammar; the program proposes a specific acceptance behavior and compares candidate forms. Generic error-set parameters stay behind a current-idiom benchmark even though all three consultations favored an eventual trial if needed.

## Verification boundary

No compiler, runtime, standard library or example implementation was edited for this consolidation. The earlier two reviews' test and probe logs remain the evidence for their behavior findings. The consolidation's own checks and their limits are saved in `verification.md` and `artifact-validation.json`. It did not run live database/provider/browser integrations or qualify a new Linux/Bun target. The resulting recommendation program is a concrete target and sequence, not a claim that the current build has passed those gates.
