# P09.2 scenario walk (coordinator) — design traced end to end

## S1 second-app edit propagation

Change a wire field in the shared contract → both server and browser rebuild
from one contract or diagnose (U03 legs, extended to the second app's
consumption of shared libs in W1). Rename a `near` parameter in a shared
control → `with` bindings + fallback lookups fail loudly at check time with
CAN-CHECK-CAPTURE; LSP rename (G) updates `with` sites. No silent drift path
found: every consumer rechecks against the locked package.

## S2 cancel-while-replacing S3 key

Seeded key + etag → begin/write/cancel. Under the O1 branch (end(Error)
verified): sink aborts without publish; original bytes+etag preserved;
concurrent readers observe only old or absent-never-partial. Under O2
(honest destructive): caller acknowledgment in source names the window.
Terminal guards prevent finish-after-cancel. Failure-path gap closed by
isolated qual (a)/(b)/(e)/(f); no prep-runnable probe exists (no endpoint).

## S3 stalled SQL + SIGTERM

Stalled query under a caller deadline (post-X-R04-1): bounded return with
honest unknown-write outcome; transaction reports commit_unknown where the
decision already passed. SIGTERM during nonsettling work: sessions close,
waiting deadline expires with work still owned, supervisor SIGKILL bounds
the rest; shutdown escalation documented, not promised away. Disconnect
mid-handler: propagation per the request-policy spec (X-R04-3 decides the
signal's existence).

## S4 old browser vs new server

Paired builds keep old digest URLs servable 7 days (retention); the new
acceptance leg drives an already-open app against the newly deployed server
and asserts defined behavior (version handshake/reload prompt/compat error —
exact UX fixed in Lane H, behavior defined not improvised). Rollback =
versioned roots + `current` swap.

## S5 two-worker crash via companion

Can owns ledger/outbox/idempotent decide/ack; companion owns claims/delivery/
retries/poison/crash recovery. Crash mid-delivery → attempt row stays
unacked → companion redelivers; idempotent ack absorbs duplicates;
commit_unknown reconciles. Carrier endpoints authenticated (R16-03 fix);
destination policy enforced by the companion. Failure identity crosses the
protocol versioned + conformed.

## S6 100k-step fault at step 60k

Lowered loop preserves step-indexed diagnostics: injected fault reports step
60000 + declared failure; no native overflow. Aggregation preserves order +
failure identity; bulk builder reports the offending element. Non-lowerable
recursion keeps current behavior + note (no silent stack dependence).

## Disposal paths (cross-scenario)

Browser: view disposal aborts listeners/timers, detaches nodes, disposes
cells; late replies cannot clobber newer edits (version checks). Request:
scope drains before response; revocation after drainage. S3: cancel retires
the handle before cleanup; terminal guards hold. Shutdown: owned work never
timer-revoked. No post-disposal use path found in the selected contracts.
