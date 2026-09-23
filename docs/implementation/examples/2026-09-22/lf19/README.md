# LF19 guide examples — 2026-09-22

Executable positive and negative examples behind the
[capability admission guide](../../../capability-admission-guide-2026-09-22.md)
(AE33) and [finite language rules](../../../finite-language-rules-2026-09-22.md)
(AE47). Each `cases/<name>/` directory holds one project: `src/main.can`,
`expect.json`, and (manifest negatives and the SQL cases only) its own
`can.project.json`.

Run every case through a development bundle:

```sh
python3 run.py --bundle /path/to/can-<version>-<target> --out results.json
```

`expect.json` is `{"mode": "pass"}` (assert exits 0 with `passed: true`) or
`{"mode": "fail", "match": "<stderr substring>"}`.

## AE47 positives (assert passes)

| Case | Rule |
| --- | --- |
| `empty-literal-expected` | Expected typing of `[]` and `option::none()` |
| `generic-inference` | Exact generic equality inference, inferred and explicit |
| `leaf-inclusion` | Leaf and narrower-variant inclusion; variant matching |
| `callable-subset` | Narrower callable bound accepted at wider bound |
| `numeric-conversion` | Explicit `number` conversions with exact/inexact rows |
| `numeric-edges` | `-0.0`/`NaN` equality, truncating division, remainder signs |
| `string-edges` | Half-open slices, UTF-16 indexing, scalars, graphemes |

## AE47 negatives (assert fails with the recorded diagnostic)

| Case | Diagnostic |
| --- | --- |
| `mixed-arithmetic` | `operator + requires identical operand types` |
| `unconstrained-empty` | `empty array requires expected element type` |
| `nominal-mismatch` | `expression type does not fit expected type` |
| `container-covariance` | `expression type does not fit expected type` |
| `unresolved-generic` | generic-call conflict on `pick` |
| `manifest-badkey` | `unknown field "bogus"` |
| `manifest-duplicate` | `duplicate JSON key "source_root"` |
| `manifest-escape` | `parent traversal is forbidden` |

## AE33 capability cases

| Case | Role |
| --- | --- |
| `pure-capability` | Worked pure capability `text::from_int` (real-can) |
| `resource-capability` | Worked resource capability `sql::pool_open/query_one/pool_close` (real-can + supplied-completion) |
| `forged-pool` | Rejected forged handle (opaque `sql::pool`) |
| `unknown-operation` | `no eligible call text::bogus` |
| `authored-binding` | `unknown protocol` for a project-authored connection |
| `incomplete-proposal` | Reviewed artifact, not a program: signature-only proposal classified **incomplete** |
