# A08 authoring-policy comparisons (X-R08-1/2/3)

Task A08. Source: R08; X-R08-1/2/3; P02.2. Prerequisites A01, A02.
This directory holds the registered held-out comparison for the three
shipped authoring policies — Boolean arm order (Q1), final-local
warning (Q2), near bindings (Q3) — against their pre-change idioms,
without reopening Q1–Q3.

## Layout

- `registry.json` — the frozen case registration: prompts, held-out
  variants, hidden checks, current-idiom runs, trial settings. Frozen
  2026-09-26 before any attempt; attempts run: 0.
- `<case>/baseline/` — immutable repair-task input. Never edit after
  registration; agent attempts start from a copy.
- `<case>/reference/` — owner reference repair. Validates that the
  hidden checks pass; it is NOT agent output and must never be shown
  in a trial prompt.
- `candidates/` — empty agent-output slots (0 attempts ran here).
- `x-r08-comparison-2026-09-26.md` — the three comparison records,
  deterministic repair-cost measurements, and the H14/H11 handoff.

## Discipline

- Held-out variants exist ONLY as rename/data maps in the registry.
  No held-out program is authored anywhere in this repo.
- Baselines are byte-frozen; candidates stay writable copies.
- No model was trained or prompted on hidden cases: no agent trial
  ran in this environment (no model access), so whole-task tokens
  are unmeasured and no superiority is claimed either way.

## Reproduce the deterministic legs

Build an isolated bundle (never in the tree):

```
go run ./tools/distbuild --archive <pinned bun zip> \
  --out /tmp/a08dist --version <tag>
CANLC=/tmp/a08dist/<root>/bin/canlc
```

Then, from scratch copies (assert needs a real directory, and the
baselines stay frozen):

```
cp -r tests/authoring-policies/<case>/<variant> /tmp/w
$CANLC assert /tmp/w            # exit code + stderr per the record
go run ./compiler format <file> # canonicalization / fixpoint legs
```

Expected: boolean baseline exit 0, reference = `format` output;
locals baseline exit 0 + one stderr warning, reference silent;
near baseline exit 1 (`outcome mismatch`, silent), reference exit 0.
