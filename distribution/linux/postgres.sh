#!/bin/sh
# PostgreSQL leg of the T19 smoke. Runs with network access to the
# disposable database only (see README for the docker network setup).
# Verifies pooled Bun.SQL behavior through the installed sidecar.
set -eu

: "${SRC:=/src}"
: "${WORK:=/work}"
: "${DATABASE_URL:?set DATABASE_URL to the disposable postgres:17 database}"

CANLC="$WORK/install-root/current/bin/canlc"
BUN="$WORK/install-root/current/runtime/bun"
test -x "$CANLC"
test -x "$BUN"

cd "$SRC"
"$BUN" distribution/linux/smoke/pg-driver.ts > "$WORK/reports/postgres.json"
python3 - "$WORK/reports/postgres.json" <<'PY'
import json, sys
report = json.load(open(sys.argv[1]))
assert report["serverVersion"].startswith("17."), report
assert report["id"] == "7" and report["body"] == "remember", report
print(f"postgres ok: server {report['serverVersion']}, roundtrip {report['id']}:{report['body']}")
PY
echo "postgres smoke ok"
