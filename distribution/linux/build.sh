#!/bin/sh
# Build and release the Linux distribution inside the T19 container.
# The Bun archive is never downloaded: pass its local path as BUN_ARCHIVE.
# Outputs the release zip, sha256 record, inspection transcript, and a
# run report with resolved tool versions into OUT (default /out).
set -eu

: "${BUN_ARCHIVE:?set BUN_ARCHIVE to the local pinned bun-linux-x64.zip}"
: "${SRC:=/src}"
: "${OUT:=/out}"
: "${VERSION:?set VERSION to the release version, e.g. t19-1}"

cd "$SRC"
mkdir -p "$OUT"

echo "--- archive pin check"
python3 - "$BUN_ARCHIVE" distribution/target-linux-amd64.json <<'PY'
import hashlib, json, sys
archive = open(sys.argv[1], "rb").read()
target = json.load(open(sys.argv[2]))
assert len(archive) == target["upstream"]["size"], "archive size mismatch"
assert hashlib.sha256(archive).hexdigest() == target["upstream"]["sha256"], "archive digest mismatch"
print(f"archive ok: {len(archive)} bytes {hashlib.sha256(archive).hexdigest()[:16]}...")
PY

echo "--- tool versions"
go version
gcc --version | head -1
python3 --version
readelf --version | head -1
cat /etc/os-release | head -3
getconf GNU_LIBC_VERSION

echo "--- build and release"
go run ./tools/distbuild --archive "$BUN_ARCHIVE" --out /tmp/bundles --version "$VERSION" --release-out /tmp/releases
ls -l /tmp/releases
cp /tmp/releases/* "$OUT"/

echo "--- run report"
{
  echo "version=$VERSION"
  echo "sourceRevision=$(git rev-parse HEAD 2>/dev/null || echo unknown)"
  echo "go=$(go version)"
  echo "gcc=$(gcc -dumpversion)"
  echo "python=$(python3 --version 2>&1)"
  echo "os=$(cat /etc/os-release | grep ^PRETTY_NAME=)"
  echo "libc=$(getconf GNU_LIBC_VERSION)"
  echo "artifacts:"
  (cd "$OUT" && sha256sum -- *)
} | tee "$OUT/run-report.txt"
echo "build ok: $OUT"
