# Acceptance-specification evidence

The [acceptance specification](../../../../implementation/language-change-acceptance-2026-09-22.md) covers every accepted disposition. These artifacts validate its completeness and baseline, not implementation acceptance.

- `eight-fetch-before.can.txt` freezes the entire current fetch fixture. The comparison stops before `fn str referenced`; keeping the remaining source preserves its helper dependencies and exposes the full-project cost boundary.
- `baseline.json` records the source path, SHA-256 and comparison marker.
- `acceptance-manifest.json` contains one record per accepted change and all seven required evidence dimensions. Every record is now evidence-required; none is marked passed.
- `validation.json` records exact coverage and recalculated baseline counts: eight declarations/calls, nine infrastructure-bound declarations, 63 entries and 56 forwarding arms. After targets remain unverified until implemented source compiles and runs.
- `validate.py` reproduces coverage/link/hash/count checks and regenerates the two JSON reports.

Run from any directory:

```sh
python3 /absolute/can-lang/docs/syntax-taste/evidence/2026-09-22/acceptance-contracts/validate.py
```

The [earlier full-review measurements](../full-language-review/measurements.json) remain source-only comparison evidence; its projected after sketch omits newly required native assertions. It is not a complete acceptance fixture. The [gap-verification evidence](../implementation-gaps/README.md) establishes current build, runner, location and resource behavior, not fixes for those gaps.

AE29 was initially design-gated. The subsequent [LD29 consultations](../ld29-checks/README.md) and C9.2 contract close that gate; implementation evidence remains required. No new difficult language-design decision or Jev consultation was made in this evidence-definition pass. No compiler/runtime implementation was changed or tested as if the proposed changes existed.
