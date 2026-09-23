# B1-04 operation/error table

As-built contract for the `process` catalogue package. Native causes are
retained privately on domain failures; public payloads carry only the fields
below. Command lines are never composed through a shell.

## Operations

| Operation | Native basis | Public errors |
|---|---|---|
| process::run(executable, args, options) | Bun.spawn detached + group, piped stdio | files::not_found, files::denied, spawn_failed, timeout, output_limit, invalid_config, io_error |
| process::require_success(value) | pure result check | nonzero |
| process::which(name) | Bun.which | files::not_found, invalid_config |

## process::options fields

cwd (empty inherits), inherit_env, env (`NAME=value` entries), stdin
(empty closes immediately), stdout_limit/stderr_limit (negative rejected,
zero allows empty output only), deadline_ms (zero disables, clamped to
2^31-1 like clock sleeps), grace_ms (zero SIGKILLs at once).

## Error mapping

| Public error | Fields | Cause |
|---|---|---|
| files::not_found | path | missing executable, or missing explicit cwd (attributed best-effort); unresolvable which() name |
| files::denied | path, operation | EACCES/EPERM at spawn with operation `spawn` |
| process::spawn_failed (1310) | executable | any other spawn throw |
| process::timeout (1311) | deadline_ms | deadline fired; the group was terminated and the child reaped first |
| process::output_limit (1312) | stream, limit | stdout/stderr cap exceeded mid-drain; the group was terminated first |
| process::nonzero (1313) | code, signal | require_success on code != 0 or signaled (-1) results |
| process::invalid_config (1314) | field, reason | empty/NUL executable, NUL args, bad env entries, negative caps, absurd durations |
| process::io_error (1315) | operation | drain/write/kill/wait failures that surface natively |

## Lifecycle guarantees

- Every run registers an owned resource; scope drain terminates the group
  even if the caller abandons the call.
- Termination is group SIGTERM, grace wait, then group SIGKILL. ESRCH (gone)
  is success, as is EPERM: on macOS a group of only unreaped zombies
  reports EPERM because no member can take the signal, while live
  own-user members accept delivery. The child is always reaped before run
  settles.
- The run registers its process resource idempotent so scope drain can
  start the close while the run is settling; the run's own close then
  joins the in-flight close instead of failing.
- Cleanup covers the direct child plus descendants still in its process
  group. Session leaders that daemonize out of the group escape; the
  contract names that boundary instead of promising tree-documented
  confinement it cannot prove.
- Late stdin writes after early exit are silently dropped by the native
  sink; the observed exit result is still reported.
