# tscheck — strict TypeScript over fresh emit

This project pins the TypeScript gate that typechecks freshly
emitted output from every maintained example. It replaces the
I44-retired committed-golden gate: nothing here checks in emit.

## Local run

From the repository root:

```sh
(cd tscheck && bun ci)
CAN_BUN_ARCHIVE=/path/to/bun.zip \
CAN_FRESH_EMIT_DIR="$PWD/tscheck/.fresh-emit" \
  go test ./tests/integration/ -run TestStdlibMaintained -count=1
(cd tscheck && bun ./node_modules/typescript/bin/tsc -p tsconfig.json)
```

`CAN_FRESH_EMIT_DIR` makes the staged stdlib test copy each
project's `dist/` under `.fresh-emit/<project>/`; `tsc` then
checks all 504 emitted files with the same strictness as the
staged per-fixture legs (`--strict`, bundler resolution, `bun`
and `node` types). The `.fresh-emit/` tree is gitignored.

## CI

`.github/workflows/tsc.yml` runs the same three steps on
`macos-15` and uploads the emit tree plus the tsc log as the
`fresh-emit-tsc-v1` artifact.
