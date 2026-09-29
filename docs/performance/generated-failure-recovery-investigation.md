# Generated failure recovery: next source investigation

Status: source investigation, not an implementation design or a speedup claim.
The accepted batch-map packet is `b770b457`; the campaign and final twelve-slice
review remain active.

The retained packet-13 six-trial observation reports `generated.failure-recovery`
Can 208.354 µs versus native 20.792 µs for the named size-100 workload. The
previous retained record had Can 108.375 µs versus native 18.228 µs. This
endpoint changed without a failure-recovery production edit, and other controls
moved materially on the busy ordered host. The records establish a large current
gap but do not assign its cause or establish a before/after recovery gain.

`tools/performance/fixtures/runtime/src/recovery.can` defines `is_positive` as
a checked `checks::require(number > 0, "positive required")` completion match:
`checks::failed` becomes `ok false`, success becomes `ok true`. The standard
driver calls it once per mixed bigint in a 100-element batch. `runtime/checks.ts`
returns a native async branded Completion; on false it creates the named domain
failure and reason record. The generated call and completion match also cross
async boundaries. This is source-visible work repeated per element, not measured
component attribution. A valid optimization must preserve authored evaluation,
the exact checked failure arm, first-boundary origin and fresh occurrences when
observable, context/owner/fixture/cancellation behavior, and boxed thenable safety.

Candidate decisions to investigate: (1) a finite exact checked recovery proof
with a synchronous boolean companion only when every failure is consumed inside
the same closed function and the caller has no context/owner; (2) a narrower
native synchronous check adapter that retains the authored match and its
Completion handling; (3) defer source changes and perform one focused attribution
if the exact IR/effect proof cannot be sound. No benchmark-name specialization,
blanket boolean or all-Functions inference, public cache, auxiliary driver
extension or speculative wrapper series is authorized. Inspect the actual
checked IR, final bindings and emitted route before choosing; a false branch
that creates observable failure metadata or schedules assertions must decline.
For the difficult design choice, make three fresh fully rewritten equivalent
Jev requests with the same evidence/options and investigate disagreements;
agreement is advisory only. Only then save an ordered Muse-owned implementation
checklist if a supported remedy remains.

Current source SHA-256 at this investigation:

| Source | SHA-256 |
| --- | --- |
| `tools/performance/fixtures/runtime/src/recovery.can` | `9fdfb059579119f441f49fc5b7ed3398677ace265d031d83d741e6f47eb663ee` |
| `tools/performance/drivers/runtime-bench.ts` | `53995d555f7e914e938b47c4b5fa67f43869e13e2226bff9fccb5163b5757c23` |
| `runtime/checks.ts` | `7ba7b11a5972b67ece25a19d81f9864a27a049001673497edccdd11cd8582cec` |
| `runtime/collections/array.ts` | `a194ddb8100df03791625e252fc578060f958f191924da31fb7404fee19160fd` |
| `compiler/internal/emit/runtime_core.go` | `eb6a08c844da8eade754d4f735248ee7d23ce13bdfd03201441128d7d6c55967` |

The frozen startup-attribution auxiliary lane stays closed; catalogue, browser,
HTTP, docs/user-guide and local poster work in the dirty checkout is foreign to
this investigation and must be preserved.
