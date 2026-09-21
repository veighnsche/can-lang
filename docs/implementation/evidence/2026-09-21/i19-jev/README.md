# I19 ownership publication consultation

Three fresh Choice requests compare publication of observed pre-success failures plus the winner, the winner alone, or every slot. All explanatory prose was rewritten; `wording-audit.txt` records the semantic-equivalence check. Raw request and response bodies are retained without credentials.

All selected `observed_before_winner`. Its probabilities were 0.99, 0.97 and 0.55; confidence was 0.98, 0.96 and 0.31. The third answer gave `winner_only` 0.42, so agreement is not treated as strong proof.

Investigation: Q7 describes errors preceding a winning success as observed/released failed participants; Q10 requires sanitized diagnostics for late standard failures. Existing owner publication reports every unselected standard failure, including already settled ones. Recording failed adapters before the first success adapter settles distinguishes already-consumed failures from later failures. Native Promise.any still selects the winner; the adapter does not recreate winner selection. Publishing all slots would suppress required late diagnostics, while winner-only publication also reports consumed pre-success failures. The controlled-promise regression tests both sides of this temporal distinction and checks that the selected result never changes.

Live API and Choice documentation were fetched on 2026-09-21 from https://docs.typesafe.ai/api.md and https://docs.typesafe.ai/primitives/choice.md after the web reader failed. Requests used the documented v1 endpoint and `jev-latest`; responses identify `jev-1.13.0`.

This consultation is design advice, not completion evidence. The separate [I19 validation report](../i19-validation.md) maps source lowering, static coverage, aggregate typing and all nine Q12 traces to executable checks.
