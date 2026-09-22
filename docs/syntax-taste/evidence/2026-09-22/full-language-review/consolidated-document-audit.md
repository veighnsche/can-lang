# Consolidated document audit

Audited `docs/syntax-taste/full-language-review-2026-09-22.md` as read at 323 lines. No edits to that document and no tests run. Missing pending consultation/example artifacts are intentionally excluded. Verified current-state claims against checker and runtime/driver code.

## Fix 1 — P1: distinguish current assertion execution from a proposed publication gate

Report line 120 says a “mandatory build assertion” can prevent publication. Current implementation does not run assertions in `Build`:

- `compiler/main.go:41,64` dispatches `Assert` and `Build` separately.
- `compiler/internal/driver/commands.go:22–27` calls `publishProgram(..., false)` for a normal build.
- `commands.go:67–70` validates generated output then publishes it.
- `compiler/internal/driver/output_runtime.go:87–90,127` says this validation is parser-only and does not execute authored expressions.
- `compiler/internal/driver/assert.go:46–57` publishes assertion modules before invoking them.
- `runtime/assert/runner.ts:81–95,123–125` really does await roots/owner drain without a timeout and reports the suite only afterward.

Suggested replacement: “The assertion command can therefore wait indefinitely without a final report. Assertion-gated publication, if adopted, must also enforce this deadline before replacing previously published output.” Retain the proposed supervisor deadline and preservation policy, but label publication gating as proposed integration rather than current behavior. If the normative spec requires build-time execution, call this an implementation/spec gap separately.

## Fix 2 — P2: exact-specialization patterns should reach the motivating coordination consumers

Report line 152 limits the proposal to “ordinary completion matches,” while the motivating fixture consumes both aggregates under `race with error` and `concurrent with error` (`compiler/testdata/current/coordination/aggregate-composition.can:84–95`). If “ordinary” excludes coordination, those consumers still require the normalization wrappers being criticized.

Clarify: exact specialized patterns apply anywhere an explicit declared error bound is dispatched, including shared first-settlement race arms and per-participant all-settled arms. Preserve plain first-success race's one implicit `all_failed` aggregate arm as a separate rule. Checker reuse supports this conceptual scope: `compiler/internal/check/coordination.go:213–224` routes coordination arm checking through `completionArms`.

## Clarification 3 — P2: define handler output-bound derivation explicitly

Lines 95–99 correctly remove outputs of fully replaced parent handlers and retain outputs reached through `inherit`. But the text says contracts are computed from “explicit target and handler contracts,” while the only wrapper syntax sketch has no handler `emits` declaration. Specify whether each handler carries an explicit bound or its finite escapes are collected from its body using declared callees and explicit completions. The latter is a reasonable narrowly scoped rule and need not imply whole-function effect inference.

Suggested addition: “For wrapper handlers, the checker collects a conservative finite escape bound from declared callees and explicit completions; branch reachability means syntactic admitted paths, not arbitrary theorem proving. Delegation contributes the predecessor handler's bound.” Alternatively show the explicit handler-bound grammar. This closes an ambiguity; the current recommendation is not demonstrably inconsistent.

## Clarification 4 — P3: assertion hard-limit granularity

The current runner executes all roots sequentially in one process (`runtime/assert/runner.ts:122–125`). A process-wide timer alone is a suite limit, not a fresh per-assertion budget. State that the supervisor either launches isolated roots or receives root-start/checkpoint notifications and rearms a hard per-root timer. Killing a timed-out root in the current shared process also prevents later roots from running; a synthesized timeout report must mark them unrun rather than silently omit them. The draft does not falsely claim isolation already exists, but this implementation choice should be explicit before promising per-assertion reports.

## Checked and sound as written

- Confirmed fetch/judge-only normalization, no question wrappers, last override wins, omitted inherited handling, errors before final `ok`, and nesting excluded from acceptance are all preserved.
- Phase-aware keys avoid conflating decoder `codec::invalid_data` with authored handler emissions; standard failures remain outside the domain policy.
- Flattened policy plus explicit predecessor delegation is coherent: one table dispatch does not forbid a finite `inherit` chain. Handler-produced failures escape and do not reenter the policy.
- Public error derivation includes passthrough and effective/delegated handler bounds; no need to retain dead overridden parent emissions.
- Default normalization is distinguished from named aliases and finite error-set parameters.
- Parenthesized callable-array type syntax is marked proposed and has no hidden variance change. Choice-arm metadata distinctions are retained.
- The seven-case normalized detail variant has an intentionally closed infrastructure scope, unlike the rejected all-errors global variant.
- Native/raw fixture testing is correctly separated from supplied final-completion substitution. The eight-call count is explicitly a source projection, not compiled evidence.

The two concrete scope/current-state fixes above should be made before final delivery. The two clarifications can be added as design requirements without expanding implementation scope.
