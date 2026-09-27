# Native x86 UP25 window (H05) — DEFERRED (not an outstanding ask)

Native x86 qualification (H12/UP25) is deferred: release preparation
and publication are not current objectives, and no x86 machine or
testing window is authorized at this stage. The legs below stay
explicitly unexecuted; nothing here is a request awaiting a user
answer. UP25 qualification cannot run on this MacBook Air: x86
emulation is forbidden here (no Docker `linux/amd64`, no
QEMU/Rosetta-Linux — none used, none present).

## Machine needed

- Debian 13+ amd64 with glibc (target `bun-1.4.2-linux-amd64-v1` per
  [target-linux-amd64.json](target-linux-amd64.json)).
- Pinned tooling to verify on arrival: bun 1.4.2 linux-x64
  (`CAN_BUN_ARCHIVE`), Go 1.27.1 toolchain, live PostgreSQL reachable
  for the roundtrip leg (`DATABASE_URL`, `CAN_LINUX_INSTALL_ROOT`).
- Reachability: an SSH (or Cloudflared/tailscale equivalent) path the
  operator can use non-interactively during the window, plus the OS/arch
  facts below. Several tailscale Linux nodes are visible from here, but
  H05 has probed none of them: the user must designate the machine.

## What a future window would need (reference only, not requested)

1. Machine designation: hostname/address + login user + which key or
   access path to use, and confirmation it is Debian 13+ amd64/glibc.
2. Exclusive window: single queue — no other jobs on the box
   during qualification; start time and duration (or "on demand").

## Verification checklist (run at window start, before any UP25 leg)

- `uname -m` reports `x86_64` on the remote host (native, not container
  arch spoofing); `/etc/os-release` shows Debian 13+; `getconf
  GNU_LIBC_VERSION` succeeds.
- `docker inspect` shows no `linux/amd64` containers involved in the
  qualification path (ideally no Docker at all); no `qemu-*` binaries in
  the path; host is bare metal or a native amd64 VM.
- `bun --version` is 1.4.2, `go version` is go1.27.1, archive sha256
  matches the pinned target file.
- No long full-tilt work stays on the Air: heavy legs run on the x86 box.
