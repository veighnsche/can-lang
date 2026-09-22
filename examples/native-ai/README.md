# native-ai

The admission example for the AI pipeline: a review-triage CLI that
drafts a plan with an LLM, transforms it with ordinary Can, picks from
dynamic options, batches judgments, gates on an explicit confidence
threshold, joins coordinated database reads, and prints one typed
`triage_report` as JSON.

## Layout

- `src/records/` — plan, report, bundle, score, and SQL records.
- `src/oracles/` — the two provider connections, the `draft` LLM
  call, the dynamic `pick` choice, the `likely`/`rate` questions,
  the batching `review` judge, and ordinary same-package wrappers
  (questions answer only inside judge bodies, so cross-package
  callers go through the wrappers).
- `src/model/` — ordinary transformation (`plan_options`, `gate`)
  plus the two coordinated count readers.
- `src/app/` — `main` and the concurrent count join.

## Run

```
canlc assert examples/native-ai
canlc build examples/native-ai
```

`main` takes an account name plus review names and reads the
`TRIAGE_DB` credential from its fd-3 environment snapshot. Schema and
seed live in test support at
`tests/integration/testdata/applications/seed.sql`; there is no
runtime migration API.

Provider endpoints are placeholders (`http://127.0.0.1:1/…`) naming
the `openai_responses_v1` and `typesafe_systemone_v1` protocols; the
admission suite rewrites them to a local compatible stub and labels
raw-provider results separately from supplied boundary results. No
shipped verdict rests on a live provider call.

## Behavior

- Empty arguments print usage; otherwise the pipeline runs draft →
  options → review → gate → coordinated reads → JSON report.
- Scores at or above 0.7 ship, below hold; unknown counts print
  `unavailable` instead of a report.
- Native, SQL, codec, and IO failures propagate with their stable
  error names and a nonzero exit.
