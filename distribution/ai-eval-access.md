# Capped AI evaluation access (H04) — BLOCKED on user approval

No provider credentials are held and no evaluation spend is approved.
No paid evaluation runs until the user grants both items below.

## What the eval harness supports (verified in-repo)

- Protocols (checker-enforced, `compiler/internal/check/connections.go`):
  `openai_responses_v1` and `typesafe_systemone_v1`. Endpoint is a
  literal pinned URL, model is free text, `max_output_tokens` only on
  the OpenAI profile.
- Worked W6-AI feature: server-side support-ticket triage into finite
  structured categories (classification-shaped; fits SystemOne/Jev; the
  `draft` LLM step in `examples/native-ai` fits the Responses profile).
- Known live SystemOne endpoint: `https://api.typesafe.ai/v1/systemone`
  (Bearer auth; precedent `docs/syntax-taste/preparation-2026-09-26/
  blocker-resolution/consult.py`, observed model `jev-1.13.0`).
- Accounting availability: TypeSafe responses carry
  `usage.input_tokens`/`usage.output_tokens` (observed in the filed
  `jev-response-*.json` fixtures), so metered accounting is decodable.
  H07/H08 own the ledger and bound qualification (X-R14-1).

## Credential env names (assigned by H04)

- `TYPESAFE_API_KEY` — TypeSafe SystemOne leg (established precedent).
- `CAN_EVAL_OPENAI_API_KEY` — proposed for the Responses leg, pending
  H08 harness adoption (no in-repo OpenAI key name exists yet).

Values never appear in artifacts or evidence files.

## Serialization

H serializes live-model runs (single queue; no concurrent provider legs).

## Exact user ask (both required to unblock)

1. Credentials: export `TYPESAFE_API_KEY` (and, if the Responses leg is
   in scope, the OpenAI key under the adopted env name) in the eval
   operator shell. H04 never stores the values.
2. Spend approval: state an explicit evaluation spend cap (max USD, and
   whether per-provider or total) for the H08/X-R14-1 runs. This cap is
   distinct from the H07 tenant token budget. Suggested framing if the
   user wants a default to accept/reject: a small fixed cap (single-digit
   USD) with H reporting metered usage after each serialized run and
   stopping at the cap.
