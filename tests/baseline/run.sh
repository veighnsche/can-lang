#!/usr/bin/env bash
# T01 repeatable baseline harness.
#
#   tests/baseline/run.sh --smoke   fast baseline verification (default)
#   tests/baseline/run.sh --full    full core regression from baseline.json
#
# Every run uses a fresh disposable directory (mktemp -d); nothing is written
# into the source tree. The report is written to <dir>/baseline-report.json
# and its path is printed. Set KEEP_DIR=1 to keep the directory for inspection.
set -euo pipefail

MODE="smoke"
case "${1:-}" in
  --smoke) MODE="smoke" ;;
  --full) MODE="full" ;;
  -h|--help)
    sed -n '2,12p' "$0"
    exit 0
    ;;
  "") MODE="smoke" ;;
  *) echo "unknown argument: $1 (want --smoke or --full)" >&2; exit 2 ;;
esac

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"
DIR="$(mktemp -d "${TMPDIR:-/tmp}/can-baseline.XXXXXX")"
if [ "${KEEP_DIR:-0}" != "1" ]; then
  trap 'rm -rf "$DIR"' EXIT
fi

BUN_VER="$(bun --version)"
BUN_REV="$(bun --revision)"
GO_VER="$(go version)"
GIT_REV="$(git -C "$ROOT" rev-parse HEAD)"
REPORT="$DIR/baseline-report.json"

step() { printf '%s\n' "--- $*"; }

failures=0
record() { # record <name> <status> <detail>
  printf '%s\t%s\t%s\n' "$1" "$2" "$3" >> "$DIR/steps.tsv"
}

step "baseline $MODE in disposable $DIR"
step "bun $BUN_VER ($BUN_REV) / $GO_VER / git $GIT_REV"

step "gofmt check (repo must be clean)"
if [ -z "$(gofmt -l compiler/ tests/ tools/ distribution/ internal/)" ]; then
  record gofmt pass "no files listed"
else
  record gofmt fail "$(gofmt -l compiler/ tests/ tools/ distribution/ internal/ | tr '\n' ' ')"
  failures=$((failures + 1))
fi

step "build canlc into disposable dir"
if go build -ldflags "-X main.version=$GIT_REV" -o "$DIR/canlc" ./compiler; then
  record build pass "$DIR/canlc"
else
  record build fail "go build ./compiler"
  failures=$((failures + 1))
fi

step "canlc --version smoke"
if [ -x "$DIR/canlc" ] && "$DIR/canlc" --version | grep -q "canlc $GIT_REV"; then
  record canlc-version pass "$("$DIR/canlc" --version)"
else
  record canlc-version fail "unexpected version output"
  failures=$((failures + 1))
fi

if [ "$MODE" = "full" ]; then
  step "go vet ./..."
  if go vet ./...; then record vet pass "clean"; else record vet fail "see log"; failures=$((failures + 1)); fi

  step "cataloguegen --check"
  if go run ./compiler/internal/catalogue/cmd/cataloguegen --check; then record catalogue pass "clean"; else record catalogue fail "see log"; failures=$((failures + 1)); fi

  step "modcheck"
  if go run ./tools/modcheck; then record modcheck pass "clean"; else record modcheck fail "see log"; failures=$((failures + 1)); fi

  step "gramcheck"
  if go run ./tools/gramcheck; then record gramcheck pass "clean"; else record gramcheck fail "see log"; failures=$((failures + 1)); fi

  step "runtime checks (bun ci + check:runtime)"
  if bun ci && bun run check:runtime; then record runtime pass "clean"; else record runtime fail "see log"; failures=$((failures + 1)); fi

  step "go test -count=1 ./..."
  if go test -timeout=60m -count=1 ./...; then record go-test pass "clean"; else record go-test fail "see log"; failures=$((failures + 1)); fi

  step "bun test runtime/test/"
  if bun test runtime/test/; then record bun-test pass "clean"; else record bun-test fail "see log"; failures=$((failures + 1)); fi
fi

{
  printf '{\n'
  printf '  "task": "T01",\n'
  printf '  "mode": "%s",\n' "$MODE"
  printf '  "date": "%s",\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  printf '  "bunVersion": "%s",\n' "$BUN_VER"
  printf '  "bunRevision": "%s",\n' "$BUN_REV"
  printf '  "goVersion": "%s",\n' "$GO_VER"
  printf '  "gitRevision": "%s",\n' "$GIT_REV"
  printf '  "dir": "%s",\n' "$DIR"
  printf '  "failures": %d,\n' "$failures"
  printf '  "steps": [\n'
  first=1
  while IFS="$(printf '\t')" read -r name status detail; do
    if [ "$first" -eq 1 ]; then first=0; else printf ',\n'; fi
    detail_esc="$(printf '%s' "$detail" | sed 's/\\/\\\\/g; s/"/\\"/g')"
    printf '    {"name": "%s", "status": "%s", "detail": "%s"}' "$name" "$status" "$detail_esc"
  done < "$DIR/steps.tsv"
  printf '\n  ]\n}\n'
} > "$REPORT"

step "report: $REPORT (failures=$failures)"
if [ "$failures" -ne 0 ]; then exit 1; fi
