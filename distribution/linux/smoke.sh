#!/bin/sh
# Install and smoke the Linux distribution. Run with networking denied:
#
#   docker run --rm --network none \
#     -v "$PWD/out:/out:ro" -v "$PWD/work:/work" \
#     can-linux/t19 /src/distribution/linux/smoke.sh
#
# Asserts install layout, runtime identity, native qualification, and the
# process/file/crypto/SQL Can example matrix on the installed artifact.
set -eu

: "${SRC:=/src}"
: "${OUT:=/out}"
: "${WORK:=/work}"
: "${VERSION:?set VERSION to the installed release version}"
: "${BUN_ARCHIVE:?set BUN_ARCHIVE to the local pinned bun-linux-x64.zip (mounted read-only)}"

cd "$SRC"
ROOT="$WORK/install-root"
rm -rf "$ROOT"
mkdir -p "$ROOT" "$WORK/reports"

ARCHIVE="$OUT/can-$VERSION-bun-1.4.2-linux-amd64-v1.zip"
SHA="$ARCHIVE.sha256"

echo "--- install"
go run ./tools/distbuild install --archive "$ARCHIVE" --sha "$SHA" --root "$ROOT"
CANLC="$ROOT/current/bin/canlc"
test -x "$CANLC"
echo "installed: $(readlink "$ROOT/current")"

echo "--- runtime-check"
"$CANLC" runtime-check > "$WORK/reports/runtime-check.json"
python3 - "$WORK/reports/runtime-check.json" <<'PY'
import json, sys
report = json.load(open(sys.argv[1]))
assert report["kind"] == "can.development-runtime-check", report
assert report["bun"] == "1.4.2", report
assert report["revision"] == "744846f844374847c902b5e7fd59b4342a51ef99", report
assert report["platform"] == "linux", report
assert report["architecture"] == "x64", report
print("runtime-check ok:", report["bun"], report["platform"], report["architecture"])
PY

echo "--- native qualification"
python3 distribution/qualify.py \
  --archive "$BUN_ARCHIVE" \
  --report "$WORK/reports/native-capabilities-linux.json" \
  --target-file distribution/target-linux-amd64.json \
  --network-isolation "docker run --network none"
python3 - "$WORK/reports/native-capabilities-linux.json" <<'PY'
import json, sys
report = json.load(open(sys.argv[1]))
assert report["passed"], report["failures"]
assert report["targetId"] == "bun-1.4.2-linux-amd64-v1", report
assert all(report["capabilities"].values()), "missing native API"
assert all(report["behaviors"].values()), "failing behavior probe"
assert report["execution"]["negativeTestsPassed"], "rejection tests failed"
assert "glibc" in report["execution"], "glibc not recorded"
print(f"native ok: {len(report['capabilities'])} APIs, {len(report['behaviors'])} probes")
PY

stage_example() {
  rm -rf "$WORK/examples/$1"
  mkdir -p "$WORK/examples"
  cp -r "$SRC/examples/$1" "$WORK/examples/$1"
  printf '%s' "$WORK/examples/$1"
}

canlc_assert() {
  "$CANLC" assert "$1" > "$WORK/reports/assert-$2.json"
  python3 - "$WORK/reports/assert-$2.json" <<PY
import json, sys
report = json.load(open(sys.argv[1]))
assert report["passed"], report
assert len(report["assertions"]) > 0, report
print(f"assert $2 ok: {len(report['assertions'])} assertions")
PY
}

echo "--- process example"
PROC=$(stage_example process)
canlc_assert "$PROC" process
OUT_PROC=$("$CANLC" run "$PROC" -- ada)
test "$OUT_PROC" = "ok"
echo "process ok: 'ok' via real /bin/echo"

echo "--- files example"
FILES=$(stage_example files)
canlc_assert "$FILES" files
mkdir -p "$WORK/reports/dir1"
printf 'x' > "$WORK/reports/dir1/a.txt"
OUT_FILES=$("$CANLC" run "$FILES" -- "$WORK/reports/dir1")
test "$OUT_FILES" = "file a.txt"
echo "files ok: 'file a.txt'"

echo "--- crypto example"
CRYPTO=$(stage_example crypto)
canlc_assert "$CRYPTO" crypto
OUT_MATCH=$("$CANLC" run "$CRYPTO" -- "correct horse")
test "$OUT_MATCH" = "match"
OUT_MISS=$("$CANLC" run "$CRYPTO" -- "wrong")
test "$OUT_MISS" = "mismatch"
echo "crypto ok: match/mismatch via argon2id"

echo "--- sqlite example"
SQLITE=$(stage_example sqlite)
canlc_assert "$SQLITE" sqlite
DB="$WORK/reports/notes.sqlite"
rm -f "$DB"
"$ROOT/current/runtime/bun" tests/integration/testdata/sql/sqlite-driver.ts setup "$DB" tests/integration/testdata/sql/sqlite-seed.sql
"$ROOT/current/runtime/bun" distribution/linux/smoke/sqlite-seed.ts "$DB"
OUT_SQL=$("$CANLC" run "$SQLITE" -- "$DB")
test "$OUT_SQL" = "7: remember"
echo "sqlite ok: '7: remember' from file persistence"

echo "--- build determinism"
ID1=$("$CANLC" build "$SQLITE" | python3 -c 'import json,sys; print(json.load(sys.stdin)["buildID"])')
ID2=$("$CANLC" build "$SQLITE" | python3 -c 'import json,sys; print(json.load(sys.stdin)["buildID"])')
test "$ID1" = "$ID2"
echo "build ok: stable $ID1"

echo "smoke ok"
