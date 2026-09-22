#!/bin/sh
# Measurement protocol for the I36 SQL binding comparison. Run from
# docs/archive/research/sql-binding/. Writes results/measure.json and measure.log.
# Requires: Go toolchain, clang (CGo candidate only), python3.
set -euo pipefail
cd "$(dirname "$0")"
mkdir -p results/bin
LOG=results/measure.log
: > "$LOG"
say() { echo "$1" | tee -a "$LOG"; }

SQL="SELECT id, display_name FROM accounts WHERE display_name ILIKE \$1 ORDER BY id LIMIT \$2"

say "=== revisions ==="
go list -m github.com/pganalyze/pg_query_go/v6 github.com/wasilibs/go-pgquery github.com/tetratelabs/wazero 2>&1 | tee -a "$LOG"
say "go: $(go version) / $(cc --version | head -n 1) / $(uname -m)"

say "=== warm build ==="
time -p go build -o results/bin/cgo ./cgo 2>&1 | tee -a "$LOG"
time -p go build -o results/bin/wasm ./wasm 2>&1 | tee -a "$LOG"
say "cgo bytes: $(stat -f %z results/bin/cgo)"
say "wasm bytes: $(stat -f %z results/bin/wasm)"

say "=== cold build (fresh GOCACHE) ==="
COLD1=$(mktemp -d); COLD2=$(mktemp -d)
GOCACHE=$COLD1 time -p go build -o "$COLD1/cgo" ./cgo 2>&1 | tee -a "$LOG"
GOCACHE=$COLD2 time -p go build -o "$COLD2/wasm" ./wasm 2>&1 | tee -a "$LOG"
say "cold cgo bytes: $(stat -f %z "$COLD1/cgo")"
say "cold wasm bytes: $(stat -f %z "$COLD2/wasm")"
rm -rf "$COLD1" "$COLD2"

say "=== CGO_ENABLED=0 (no C toolchain) ==="
if CGO_ENABLED=0 go build -o results/bin/wasm-nocgo ./wasm 2>&1 | tee -a "$LOG"; then
  say "wasm CGO_ENABLED=0: builds, bytes: $(stat -f %z results/bin/wasm-nocgo)"
else
  say "wasm CGO_ENABLED=0: FAILED"
fi
if CGO_ENABLED=0 go build -o /tmp/cgo-nocgo ./cgo 2>&1 | tee -a "$LOG"; then
  say "cgo CGO_ENABLED=0: unexpectedly builds"
else
  say "cgo CGO_ENABLED=0: refuses as expected"
fi

say "=== cold invocation (21 fresh processes each, wall ms) ==="
python3 - "$SQL" <<'PY' 2>&1 | tee -a "$LOG"
import json, statistics, subprocess, sys, time
sql = sys.argv[1]
out = {}
for name in ("cgo", "wasm"):
    walls, inners = [], []
    for _ in range(21):
        start = time.perf_counter()
        proc = subprocess.run(["results/bin/" + name, "--once", sql],
                              capture_output=True, text=True, check=True)
        walls.append((time.perf_counter() - start) * 1000)
        inners.append(json.loads(proc.stdout)["elapsed_ns"] / 1e6)
    out[name] = {"wall_min": min(walls), "wall_median": statistics.median(walls),
                 "inner_min": min(inners), "inner_median": statistics.median(inners)}
    print(name, json.dumps(out[name]))
json.dump(out, open("results/cold.json", "w"), indent=1)
PY

say "=== warmed benchmarks ==="
go test ./cgo/ -run XXX -bench . -benchtime 1s -count 2 2>&1 | tee -a "$LOG"
go test ./wasm/ -run XXX -bench . -benchtime 1s -count 2 2>&1 | tee -a "$LOG"

say "=== batch memory (50 corpus passes) ==="
/usr/bin/time -l results/bin/cgo --batch 50 --corpus corpus/corpus.json --out /tmp/batch-cgo.json > /tmp/mem-cgo.json 2> /tmp/mem-cgo.time; cat /tmp/mem-cgo.json | tee -a "$LOG"; grep "peak memory footprint" /tmp/mem-cgo.time | tee -a "$LOG"
/usr/bin/time -l results/bin/wasm --batch 50 --corpus corpus/corpus.json --out /tmp/batch-wasm.json > /tmp/mem-wasm.json 2> /tmp/mem-wasm.time; cat /tmp/mem-wasm.json | tee -a "$LOG"; grep "peak memory footprint" /tmp/mem-wasm.time | tee -a "$LOG"
say "=== batch memory under GC pressure (GOGC=20) ==="
GOGC=20 results/bin/cgo --batch 50 --corpus corpus/corpus.json --out /tmp/batch-cgo.json 2>&1 | tee -a "$LOG"
GOGC=20 results/bin/wasm --batch 50 --corpus corpus/corpus.json --out /tmp/batch-wasm.json 2>&1 | tee -a "$LOG"

say "=== module payload ==="
go list -m -f '{{.Dir}}' github.com/wasilibs/go-pgquery | tee -a "$LOG" | xargs -I{} ls -l {}/internal/wasm/libpg_query.so | tee -a "$LOG"

say "done"
