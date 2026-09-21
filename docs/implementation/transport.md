# Connection transport policy

I15 supplies static connection policy and the private native HTTP transport shared
by fetch, judge and LLM adapters. Native declaration syntax and checker wiring are
I16; provider envelopes and named fetch mode decoding are their later tasks.

`CheckConnection` accepts literal setting evidence only. It checks required URL
and timer configuration, bounded sizes, the closed metadata/profile inventory,
header ownership, mapped duplicates and literal values. `CheckAI` checks the
selected profile, model and JSON header conflicts. Neither reads credentials nor
runs authored expressions. Default request and response bounds are 8,388,608 bytes
independently, with an explicit maximum of 67,108,864.

The maintained runtime caches native endpoint normalization per immutable
connection plan. Every reached request resolves its path with native URL, checks
same origin and forbidden URL components, and appends scalar query values using
URLSearchParams. Native Headers validates and normalizes scalar byte-string
values after request entries replace whole default lists. Header snapshots sort
unique names, retain native combined values and preserve getSetCookie order;
the pinned Headers iterator repeats cookie names, so names must be deduplicated.

Bearer lookup occurs once after preparation and outgoing byte admission. Only
the configured variable name can appear in a missing-credential payload; invalid
values never appear in transport messages. Requests use native fetch exactly once
with manual redirects and no ambient credentials/cookie persistence. There are
no implicit retries. Body-only rejected statuses dispose the stream before any
decode; envelope policy admits final 200–599 statuses.

A monotonic deadline begins immediately before fetch. It covers headers, delivered
native stream bytes, decoding and validation, ending before authored handlers.
Native abort accompanies expiry. Monotonic checks reject synchronous validation
that finishes too late even if the timer could not run. Deadline expiry takes
precedence over owner cancellation. A recognized owner cancellation otherwise
maps to the cancelled transport phase. Only the actual native fetch/stream calls
classify their failures; arbitrary decoder errors remain standard failures.

Streaming counts delivered bytes after native decompression and rejects the first
chunk exceeding the bound. Remaining streams are cancelled. Native fetch/read/
cancel promises remain owned through actual settlement, independently of caller
waits. Native continuations may finish under an existing owner during scope drain;
that does not admit a new coordination owner. Unexpected cleanup faults become
late standard diagnostics without replacing an already selected domain failure.

Decoder results stay in protected Completion boxes throughout promise boundaries,
including results with a then-named data field. HTTP failures are constructed
through the sealed domain runtime as catalogue IDs 1100–1105 with exact typed
payloads and immutable header records. Wire body modes, media-type adapters and
provider-specific response validation remain the corresponding later tasks.
