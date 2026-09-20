# I02 environment consultation

The original three requests incorrectly described a separated `--config path`
probe as evidence of ineffective explicit configuration. The integration tests
falsified that conclusion: on the pinned executable the correct form is
`--config=path`. An explicit file then prevents the global preload. Original
requests/responses are retained as superseded evidence, not justification.

The three fresh requests in `corrected/` include this correction and the observed
upstream directory-mismatch diagnostic from `--tsconfig-override=path`. Each
rewrites the context, question instructions, and every option description;
a preflight check rejects identical wording across each corresponding field.
Manual semantic comparison confirmed all three retain the same target,
requirements, facts, alternatives and uncertainty. Exact technical identifiers
remain unchanged. No consultation used another response as input.

All corrected responses selected `snapshot`, with reported confidences
0.99, 0.98 and 1.0. There was no selection disagreement to investigate. This is
advice, not proof of safety or removal of framing effects. Actual offline
integration checks justify the implementation: a private startup environment,
explicit `--config=path`, no automatic installs/env files/macros, a bundle-local
locked tsconfig found by native lookup, and an inherited pipe carrying caller
values for the trusted env/auth accessor. No override flag is used.

Current API and Choice documentation were fetched from docs.typesafe.ai/api.md
and docs.typesafe.ai/primitives/choice.md before consultation. Returned model:
`jev-1.13.0`; request alias: `jev-latest`. The only submitted state is design and
probe context, not source secrets or application environment values.
