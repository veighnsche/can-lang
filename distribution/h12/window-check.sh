#!/bin/sh
# H12 UP25 window-entry verification checklist (automation).
# Runs ON the native x86 host at window start, before any H12 leg.
# Inspection only: prints facts and PASS/BLOCKED per check, exits
# nonzero when any blocking check fails. Never prints secret values.
# Env inputs (names only): CAN_BUN_ARCHIVE, DATABASE_URL (optional here).
# Usage: sh distribution/h12/window-check.sh [SRC]
set -eu

SRC="${1:-$(pwd)}"
TARGET="$SRC/distribution/target-linux-amd64.json"
fail=0
warn=0

pass() { echo "PASS: $1"; }
block() { echo "BLOCKED: $1"; fail=1; }
note() { echo "NOTE: $1"; warn=$((warn + 1)); }

echo "--- host identity (must be native Debian 13+ amd64/glibc)"
ARCH="$(uname -m)"
if [ "$ARCH" = "x86_64" ]; then pass "uname -m = $ARCH"; else block "uname -m = $ARCH, want x86_64"; fi
if [ -r /etc/os-release ]; then
  # shellcheck disable=SC1091
  . /etc/os-release
  echo "os-release: ID=${ID:-?} VERSION_ID=${VERSION_ID:-?} PRETTY_NAME=${PRETTY_NAME:-?}"
  if [ "${ID:-}" = "debian" ]; then
    MAJOR="$(printf '%s' "${VERSION_ID:-0}" | cut -d. -f1)"
    if [ "$MAJOR" -ge 13 ] 2>/dev/null; then pass "Debian $VERSION_ID"; else block "Debian $VERSION_ID below 13"; fi
  else
    block "ID=${ID:-?}, want debian"
  fi
else
  block "/etc/os-release unreadable"
fi
if getconf GNU_LIBC_VERSION >/dev/null 2>&1; then
  pass "glibc: $(getconf GNU_LIBC_VERSION)"
else
  block "getconf GNU_LIBC_VERSION failed (glibc required)"
fi

echo "--- emulation absence (no linux/amd64 emulation anywhere)"
if command -v qemu-x86_64 >/dev/null 2>&1 || ls /usr/bin/qemu-* >/dev/null 2>&1; then
  block "qemu binaries present in path"
else
  pass "no qemu-* in path"
fi
if command -v docker >/dev/null 2>&1; then
  if docker ps --format '{{.Platform}}' 2>/dev/null | grep -q "linux/amd64"; then
    block "linux/amd64 containers running"
  else
    note "docker present but no linux/amd64 containers running (ideally no Docker at all)"
  fi
else
  pass "no docker on PATH"
fi
if [ -f /.dockerenv ]; then
  block "running inside a container (/.dockerenv present)"
else
  pass "not inside a container"
fi
if grep -qa "container=" /proc/1/environ 2>/dev/null; then
  block "PID 1 environ indicates a container"
else
  pass "PID 1 is not containerized"
fi

echo "--- pinned tooling"
if command -v bun >/dev/null 2>&1; then
  BUNV="$(bun --version 2>/dev/null || echo "?")"
  if [ "$BUNV" = "1.4.2" ]; then pass "bun $BUNV"; else block "bun $BUNV, want 1.4.2"; fi
else
  note "bun not on PATH (installed sidecar is used instead; record version at install)"
fi
if command -v go >/dev/null 2>&1; then
  GOV="$(go version 2>/dev/null || echo "?")"
  case "$GOV" in
    "go version go1.27.1 "*) pass "$GOV" ;;
    *) block "$GOV, want go1.27.1" ;;
  esac
else
  block "go toolchain missing (want go1.27.1)"
fi
if command -v gcc >/dev/null 2>&1; then pass "gcc: $(gcc -dumpversion)"; else block "gcc missing (cgo link of canlc needs it)"; fi
if command -v python3 >/dev/null 2>&1; then pass "python3: $(python3 --version 2>&1)"; else block "python3 missing (qualify.py needs it)"; fi

echo "--- pinned Bun archive"
if [ -z "${CAN_BUN_ARCHIVE:-}" ]; then
  block "CAN_BUN_ARCHIVE unset (pinned bun-linux-x64.zip path required)"
elif [ ! -f "$CAN_BUN_ARCHIVE" ]; then
  block "CAN_BUN_ARCHIVE=$CAN_BUN_ARCHIVE not a file"
else
  python3 - "$CAN_BUN_ARCHIVE" "$TARGET" <<'PY'
import hashlib, json, sys
archive = open(sys.argv[1], "rb").read()
target = json.load(open(sys.argv[2]))
assert len(archive) == target["upstream"]["size"], "archive size mismatch"
assert hashlib.sha256(archive).hexdigest() == target["upstream"]["sha256"], "archive digest mismatch"
print(f"PASS: archive {len(archive)} bytes sha256 {hashlib.sha256(archive).hexdigest()[:16]}... matches {target['targetId']}")
PY
fi

echo "--- offline mechanism"
if unshare -n true 2>/dev/null; then
  pass "unshare -n available (outer offline wrapper usable)"
else
  block "unshare -n unavailable (offline legs cannot run; no quiet fallback)"
fi

echo "--- browser provision (old-browser leg)"
if [ -d "$SRC/tests/integration/browser" ]; then
  if (cd "$SRC/tests/integration/browser" && bunx --bun playwright@1.55.1 --version 2>/dev/null) | grep -q "1.55.1"; then
    pass "playwright 1.55.1 resolvable"
  else
    note "playwright 1.55.1 not yet installed (runbook installs chromium during the window)"
  fi
else
  block "tests/integration/browser missing from checkout"
fi

echo "--- database reachability (names only, no values printed)"
if [ -z "${DATABASE_URL:-}" ]; then
  note "DATABASE_URL unset (PG roundtrip + operator-DDL legs need it before they run)"
else
  note "DATABASE_URL is set (value never printed; connectivity checked by the PG leg)"
fi

echo "--- verdict"
if [ "$fail" -ne 0 ]; then
  echo "WINDOW-ENTRY: BLOCKED ($fail blocking failures, $warn notes)"
  exit 1
fi
echo "WINDOW-ENTRY: READY ($warn notes to clear before the affected legs)"
