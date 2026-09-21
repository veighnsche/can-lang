# Ownership and resource lifecycle

I20 implements the private Q10/P6 lifecycle core in `runtime/owner.ts`.
Native AsyncLocalStorage binds independent root and nested scope executions.
The qualification manifest requires its API and interleaved-context behavior;
there is no ambient-global or alternate-runtime fallback.

`launchOwned` acquires every participant's captured resource leases before any
participant starts. Each native promise is normalized and immediately observed.
Publishing selected indexes does not release losing work. Settlement releases
captures and leases; unselected standard occurrences emit the C9 diagnostic
projection once. Diagnostic delivery cannot change the selected completion.
The deduplication registry is weak, and reports exclude native causes, stacks,
and private payloads. C9's message may contain the native Error message.

Checked callable IR records which captures can contain opaque resources,
including nested callable/data captures. Emission validates that evidence.
Ordinary callable construction and invocation retain metadata without acquiring
whole-invocation leases. Participant launch consumes the metadata, and maintained
native operations validate resource provenance, kind, scope and state before
acquiring a lease. This avoids inventing an ordinary-call self-close deadlock.

Resources have private nominal provenance, identity, owning scope, state and
owner-keyed lease counts. Closing refuses new owners but permits an existing
owner's finishing subleases. Native close waits for leases and has one shared
promise installed before invoking an adapter, including reentrant adapters.
Only explicitly idempotent close operations share repeated calls. A resource
created by a participant has an initial lease until settlement, explicit close
by its creator, or its nested scoped callback's exit; independent captures and
native-operation leases remain intact.

Host callback wrappers bind their owning operation scope even when native event
dispatch has no inherited async context. Active callback work is retained until
settlement. Completed ambient tasks are not reused. A callback cannot enter a
closed operation, and a new owner cannot enter a closing operation. Scoped
callback settlement starts closing; retained owners drain before native cleanup.

CLI and assertion roots drain owners and callbacks before reverse resource
cleanup. Omitted explicit close remains a cleanup failure even if automatic
close succeeds. Automatic cleanup faults are standard cleanup failures. Caller
close deadlines race only the caller's wait; registered shutdown deadlines run
while the root drains, including for resources created by late participants.
Neither deadline releases leases or cancels native work. A permanently pending
owner can keep the process pending indefinitely; external supervisor termination
makes no assertion that cleanup or a runtime exit completed.

The coordination syntax/native aggregate algebra is I19. Provider-specific
resource constructors, declared close errors, server and SQL adapters are later
tasks; this core does not claim those integrations are implemented.
