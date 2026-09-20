# Can implementation preparation

The five preparation stages are complete. **Implementation has not started and is not authorized by this packet.** The compiler, runtime, stdlib and current language decisions are unchanged.

Read in order:

1. [Research](research.md) — actual code inventory, native probes, upstream facts and limitations.
2. [Brainstorm](brainstorm.md) — compiler replacement, Bun packaging and library alternatives.
3. [Reconciliation](reconciliation.md) — engineering choices, three fresh Jev consultations and investigated disagreements.
4. [Implementation plan](plan.md) — executable milestones, safe dist tree, source maps and distribution.
5. [Actionable tasks](tasks.md) — 50 unchecked tasks with dependencies, code areas, contract references and positive/negative/integration tests.

Supporting ledgers: [native reuse](native-reuse.md), [full coverage and exclusions](coverage.md), [saved evidence](evidence/2026-09-21/).

The first executable milestone is **M1**: a manifest-backed current Can CLI program and real generated-code assertion running on a private bundled Bun, with output exclusively in dist and diagnostics mapped to Can. **M2** immediately adds the native Noul/judge vertical slice with grouped state and real local HTTP/raw-provider validation.

The plan chooses a new typed Go pipeline and a versioned sidecar bundle containing upstream Bun; users install no separate runtime. An embedded/extracted alternative was evaluated. The PostgreSQL parser binding remains a bounded implementation spike between two upstream integrations; no custom SQL parser is authorized.

Research observed Bun1.4.2 on macOS arm64 and a passing legacy Go suite. These do not prove new-language conformance. Real PostgreSQL/HTMX application tests, full codecs/ownership, source-map composition and signed offline distribution remain implementation/release gates. No new author-visible syntax question blocks beginning M0/M1. Stage6 requires a subsequent request to implement.
