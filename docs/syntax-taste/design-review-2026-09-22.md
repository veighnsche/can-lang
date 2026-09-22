# Can language design review — 2026-09-22

Status: design input, not decisions. Companion to
[redundancy-review-2026-09-22.md](redundancy-review-2026-09-22.md), which covers
surface-form duplication in detail; this document covers the rest of the design:
structural tradeoffs, thesis gaps, soundness risks, expressiveness ceilings,
and spec hygiene. Nothing here is approved until it goes through the normal
decision process. Per `AGENTS.md`, adopting any redesign requires three fresh
Jev consultations (fully reworded packets, saved requests/responses) before
committing, and there are zero external users: no back-compat obligation
constrains any finding below.

Method: read the four spec companions (`technical-spec.md`,
`coordination-spec.md`, `ai-io-spec.md`, `platform-testing-spec.md`, ~5,200
lines), `decisions.md`, and the admitted examples under `examples/`, `std/`,
and `compiler/testdata/current/`.

## Verdict

The design is coherent and genuinely ambitious: contract-first, AI-agent
audience, tests-as-declarations, native AI judgments, deterministic fixtures
for concurrent code. The thesis is consistently applied and the evidence
discipline (fifty implementation tasks, all linked to evidence) is excellent.
The findings below do not dispute the thesis; they dispute its coverage
(where the riskiest code is exempt from the central rules), its ceilings
(what users fundamentally cannot express or extend), and several spots where
the implementation or specification cost is disproportionate to the benefit.

## Strengths

- **Native reuse as law** (`AGENTS.md`) is the right implementation principle,
  and the Bun probes backing each mapping are real diligence.
- **Judge batching** (A7 phases: prepare, validate-all, single POST, ordered
  handlers) is a genuinely good answer to "how do probabilistic judgments
  compose."
- **The fixture identity story** (P3/P5) is the most thorough
  deterministic-concurrency test design in the spec corpus: lexical ordinals,
  participant paths, callable instances, canonical scheduler.
- **The zero-compat rule** is exactly right at this stage. Every finding below
  is cheap to act on because of it.

## F1 — The extensibility story is the biggest undecided tradeoff

Users cannot extend platform access at all: no `extern`, distribution-only
catalogue, every new capability needs a compiler release (SURFACE-063/064/067).
That is a defensible choice, but its consequence needs to be explicit: **the
language's expressiveness ceiling equals the distribution release cadence.**
A tag outside the HTML inventory, a platform API outside the 144 catalogue
operations, a new judgment protocol — each waits on a compiler change.

Options:

- (a) Accept distribution-only extension explicitly, with a documented
  capability-request process and a catalogue-evolution policy (how new ops are
  proposed, versioned, and retired).
- (b) Design a safe extension mechanism that is not embedded code: e.g.
  declarative adapter manifests with compiler-checked signatures — authors
  declare a typed binding to a named native API, the compiler validates the
  declaration and owns the lowering. Configuration, not code injection.

This decision gates all other platform-scope questions and dwarfs any single
redundancy. It should go through the Jev triple-consultation first.

## F2 — The riskiest code is exempt from the central thesis

**Natives have no mandatory assertions.** Every `fn` requires `asserts`, but
`judge`, `llm`, `fetch`, and question declarations explicitly "do not acquire
the function-only mandatory `asserts` grammar" (A3). "Tests and logic are one
artifact" stops where testing matters most: nondeterministic, network-touching
declarations. The A11 fixture matrix covers adapter conformance, not authored
question/LLM behavior (thresholds, criteria, fallback choices). At minimum,
decide what authored-native testing looks like — e.g. mandatory descriptor
assertions (prepared request shape for given inputs) even where live answers
cannot be asserted.

**`[_]` is a stringly-typed hole in contract-first.** Standard failures bypass
`emits`, and the handler receives an undifferentiated string: it cannot
distinguish `bounds` from `arithmetic` from `resource_state` without parsing
message text. Meanwhile `standard_failure` snapshots (with `kind`,
`occurrence_id`, `message` projections) exist only inside coordination
aggregates, never in `[_]`. So the failure taxonomy is both leaky (outside the
contract system) and inconsistently exposed (structured in one place,
stringly in another). Options: expose the snapshot in `[_]` too
(`[_] as standard_failure failure`), or admit that standard failures are
unrecoverable-by-kind and restrict `[_]` to logging/cleanup/fallback.

## F3 — The examples indict `match chain` and deep nesting

`examples/native-ai/src/app/main.can` nests `match` six or more levels deep
instead of using `match chain` — the construct built for exactly that shape.
When the admitted examples avoid the sequencing construct, its shape (shared
arms, `as` bindings) does not fit real heterogeneous code. Either reshape
chain until the examples want it, or accept the pyramid as load-bearing and
stop maintaining chain. Do not keep both a neglected abstraction and the
concrete pattern that replaced it.

Related expressiveness floor: no early-exit fold/scan and no break means
"fold until condition" must traverse fully or restructure into recursion-free
workarounds. `find`/`some`/`every` short-circuit but carry no accumulator;
`fold` carries an accumulator but cannot stop. A general language needs one
halting traversal. Candidates: a `fold_until` catalogue operation, or a
reducible result wrapper (`done`/`continue` variant) understood by one
traversal combinator.

## F4 — Resource safety is runtime-dynamic in a contract-first language

Ownership, leases, and scoped handles are enforced by a runtime registry; the
compiler "should diagnose obvious direct escapes as a quality improvement, but
program correctness does not depend on that optional diagnostic" (P6). Escaping
a transaction handle from its scope is a dynamic `resource_state` failure, not
a type error. For a language that statically proves arm coverage, error bounds,
and fixture shapes, resource safety being dynamic is an anomaly. The fix need
not be an ownership type system: a simple syntactic must-not-escape rule for
scoped handles (as a hard error, with the runtime check retained as defense
in depth) would close most of the gap at a fraction of Q6's complexity.

## F5 — Hanging constructs vs mandatory test execution

Empty `race with error` "remains pending forever" as an explicit consequence
of the native mapping (Q9). There is no termination proof (deliberate) and no
specified assertion timeout. An assertion that never settles blocks the suite
with no deadline and no diagnostic. Define assertion timeouts: a per-assertion
(or per-run) bound after which the harness reports the pending invocation
paths (P3 identities make this feasible) and fails. Without it, one empty
spread in one test hangs the build silently.

## F6 — Verbosity has no budget

The design optimizes for explicitness with no countervailing token budget, for
an audience (AI agents) where tokens are cost. Exhibits: the fifteen-entry
`emits` line in `examples/native-ai/src/app/main.can`, one-line rules forcing
very long assertion lines, `when` rows restating full argument lists at every
call site. Measure tokens-per-feature against the TypeScript equivalent for the
four admitted applications. If Can costs two to three times the tokens, that is
the price of the thesis — but it should be a measured number, not a vibe, and
it should inform which relief valves (named error sets, fixture hoisting —
see the redundancy review's R8/R9) pay for themselves first.

## F7 — The "no coercion / no inference" claims overstate reality

The language coerces and infers at chosen boundaries: variant-leaf injection
for array literals and `all_failed` payloads, expected-type propagation for
`[]`, generic inference with finite search. That is pragmatic and fine — but
the absolute language ("no implicit conversions," "no inference that hides
information") invites confusion about where the line is. State the actual
rule: **injection and inference happen only at construction and aggregate
boundaries; they never silently change an existing value's type.** The `[]`
needs-expected-type vs `1`-is-never-float line is principled; it is just not
"no inference," and the spec should say what it is.

## F8 — Contracts partly live in JSON, not Can

Error registries, SQL descriptors, the project manifest, and lock digests
(P2/P12) are JSON Schema validated by the compiler — a second, untested-by-
`asserts` surface alongside the language. Reasonable engineering, but a
"contract-first language" whose key contracts are not all in the language
deserves a sentence of acknowledgment, plus a decision on whether descriptors
ever become Can declarations (e.g. SQL descriptors as `fetch`-like checked
declarations, registries generated rather than hand-maintained).

## F9 — Specified-but-unowned: formatter and diagnostics UX

Strict layout rules (four-space indent, one-line calls/constructors/arrays/
assertions) with no formatter in the current specs — the R10 formatter
requirement was predecessor-only. Either the layout rules need a canonical
formatter (without one, authors fight layout by hand and agents burn tokens
on whitespace repair), or the rules should be relaxed to what the checker can
verify without a formatter.

Separately, diagnostics are where agent ergonomics actually live, yet the
specs define language semantics, not diagnostic quality. Recommend a
diagnostic-UX pass as its own work item: message wording, span precision, and
suggested fixes for the most common errors (missing arms, undeclared emits
entries, fixture mismatches). For the stated audience this matters more than
most grammar choices.

## Suggested order

1. F1 extensibility decision — structural, gates everything platform-related.
2. F2 natives/`[_]` thesis gaps and F5 assertion timeouts — correctness-adjacent.
3. F3 chain-vs-nesting and halting traversal, F4 static resource rule.
4. F6 verbosity measurement plus relief (with redundancy review R8/R9).
5. F7–F9 spec hygiene: coercion/inference wording, JSON-contract acknowledgment,
   formatter and diagnostics pass.

Each adopted step needs the `AGENTS.md` triple Jev consultation before the
decision is recorded.
