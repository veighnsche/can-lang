# X-R01-1 host discrimination (D01)

Lane-D isolated prototypes comparing the catalogue-vs-adapter-vs-companion
tiers for three widget/operation classes. Nothing here is admitted: no
prototype is referenced from `runtime/`, `compiler/`, or the catalogue,
and every prototype file is listed as `unreviewed` in
`REVIEW-MANIFEST.json`. Tier selection is recorded in `x-r01-1.md`;
implementation of any selected tier belongs to D02.

Run:

```sh
bun test tests/host-discrimination/
go test ./tests/host-discrimination/ -count=1
```
