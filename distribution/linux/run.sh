#!/bin/sh
# T19 host orchestrator: build the pinned image, release, install, smoke,
# and run the PostgreSQL leg. Run from the repository root:
#
#   BUN_ARCHIVE=/absolute/path/bun-linux-x64.zip ./distribution/linux/run.sh
#
# The archive is acquired explicitly and verified against provenance.json;
# nothing here downloads Bun. Leaves release artifacts in ./out/linux and
# reports in ./out/linux-work/reports.
set -eu

: "${BUN_ARCHIVE:?set BUN_ARCHIVE to the local pinned bun-linux-x64.zip}"
: "${VERSION:=t19-1}"
: "${IMAGE:=can-linux/t19}"

ROOT=$(git rev-parse --show-toplevel 2>/dev/null || pwd)
cd "$ROOT"
OUT="$ROOT/out/linux"
WORK="$ROOT/out/linux-work"
mkdir -p "$OUT" "$WORK"

echo "--- provenance check"
python3 - "$BUN_ARCHIVE" distribution/linux/provenance.json <<'PY'
import hashlib, json, sys
archive = open(sys.argv[1], "rb").read()
upstream = json.load(open(sys.argv[2]))["upstream"]
assert len(archive) == upstream["size"], "archive size mismatch"
assert hashlib.sha256(archive).hexdigest() == upstream["sha256"], "archive digest mismatch"
print(f"archive ok: {upstream['archive']} {len(archive)} bytes")
PY
echo "sourceRevision=$(git rev-parse HEAD 2>/dev/null || echo unknown)"

echo "--- image build"
docker build --platform linux/amd64 -f distribution/linux/Dockerfile -t "$IMAGE" .

echo "--- release"
docker run --rm --platform linux/amd64 \
  -v "$BUN_ARCHIVE:/archive/bun.zip:ro" -v "$OUT:/out" \
  -e BUN_ARCHIVE=/archive/bun.zip -e VERSION="$VERSION" \
  "$IMAGE" /src/distribution/linux/build.sh

echo "--- install and smoke (no network)"
docker run --rm --platform linux/amd64 --network none \
  -v "$BUN_ARCHIVE:/archive/bun.zip:ro" -v "$OUT:/out:ro" -v "$WORK:/work" \
  -e BUN_ARCHIVE=/archive/bun.zip -e VERSION="$VERSION" \
  "$IMAGE" /src/distribution/linux/smoke.sh

echo "--- postgres leg"
PROV_PG=$(python3 -c 'import json; print(json.load(open("distribution/linux/provenance.json"))["postgresQualificationImage"]["amd64ManifestDigest"])')
docker pull --platform linux/amd64 "docker.io/library/postgres:17@$PROV_PG" >/dev/null
docker network create can-t19-smoke >/dev/null 2>&1 || true
cleanup() { docker rm -f can-t19-pg >/dev/null 2>&1 || true; docker network rm can-t19-smoke >/dev/null 2>&1 || true; }
trap cleanup EXIT
docker run -d --rm --platform linux/amd64 --name can-t19-pg --network can-t19-smoke \
  -e POSTGRES_PASSWORD=smoke -e POSTGRES_DB=smoke \
  "docker.io/library/postgres:17@$PROV_PG" >/dev/null
for _ in $(seq 1 60); do
  docker run --rm --platform linux/amd64 --network can-t19-smoke \
    "docker.io/library/postgres:17@$PROV_PG" \
    pg_isready -h can-t19-pg -U postgres >/dev/null 2>&1 && break
  sleep 2
done
docker run --rm --platform linux/amd64 --network can-t19-smoke \
  -v "$WORK:/work" \
  -e DATABASE_URL=postgres://postgres:smoke@can-t19-pg:5432/smoke \
  "$IMAGE" /src/distribution/linux/postgres.sh

echo "--- done"
echo "artifacts: $OUT"
echo "reports:   $WORK/reports"
