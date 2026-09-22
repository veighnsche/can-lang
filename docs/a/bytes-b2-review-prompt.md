# Prompt: adversarial pre-implementation review of B2 (issue #42)

Paste this into the reviewer with [docs/bytes-plan.md](bytes-plan.md)
(v3, item 1 + item 6 + B2 row) attached plus the context files below.
B2 is not implemented yet — this review attacks the design before code
lands. B1 (value admission) is already shipped and tested; do not
re-litigate it except where B2 depends on it.

---

Review the B2 design: an owner-local `exports_utf8 Brand via fn@rev`
grant authorizing one exact-shape function to disclose one brand as
UTF-8 Bytes, enforced by a checked certificate in both evaluator and
emitter, with the generic encoder staying strictly `str → Bytes`.
For each item, return: holds / broken with file:line evidence and a
concrete bypass or hole / unresolved with the exact missing source.
No compiler gates have been run on B2 — there is no B2 code yet.

1. Grant form. Is `exports_utf8 Brand via fn@rev` parseable
   unambiguously in the existing declaration grammar? Name the exact
   parse function and node shape it needs, and any grammar collision
   (new keyword? `via`/`exports_utf8` conflicts?).
2. Authority rule. "Same canonical `Module.ID`" — is `Module.ID`
   vs display `File` the right anchor, and is it sufficient? Construct
   the strongest same-basename/different-owner attack you can and show
   why the rule stops it or fails to.
3. Shape rule. "Exactly one parameter of exactly the granted brand;
   exactly `Bytes__Value`; `emits []`; exactly one call match over
   `bytes__utf8__export(parameter)` returning `r.value` unchanged."
   Enumerate every evasion: extra params with defaults (if the language
   has them), wrapper indirection, string intermediates, helper
   forwarding, `given` injection, multi-arm matches with an unreachable
   exfiltration arm. Which does the stated rule catch, and which needs
   a stronger check?
4. Certificate enforcement. "Both evaluator and emitter must reject a
   restricted export call without that certificate." Trace both paths:
   where does the evaluator dispatch the export kernel, and where does
   the emitter lower the granted call? Name the exact functions. Is
   there a third path (exhaustiveness, coverage, linkage trust,
   test-data sealing) that could admit an uncertified export?
5. The Secret composition. Under this design, write the full
   `Vault__Secret → encode → Bytes → decode → str` attempt and the
   exact diagnostic that must fire at each edge. Then try to break it:
   ungranted-function export, cross-module grant, same-file private
   brand, record/sequence smuggling, test/given seals. The design
   claims "no new representation-recovery path for an unrelated,
   unexported brand in checked CAN" — falsify it or confirm the
   boundary, including the two stated limits (deliberate owner grant,
   promotion into an exported brand).
6. `Bytes__Value` ownership. B2 introduces a compiler-owned record
   before any codec needs it. Is the shared declaration lookup
   (world construction, `newTycker`, `recordDecl`, `recordShapes`,
   return-union construction, catalog) correctly scoped for one
   record with zero kernels depending on it yet? What breaks if the
   lookup lands only partially?
7. B2/B3 split. B2 ships the restricted exporter; B3 ships the generic
   encoder. Is that order sound — can B2's acceptance (granted export
   rows, unrelated-brand denials, no string-returning exporter) be
   demonstrated without the generic kernel? Is the test-only declared
   Bytes-to-text sink a sound stand-in for the decoder, or does it
   prove less than claimed?
8. Scope bleed. Does any B2 requirement secretly need B3+ machinery
   (kernel descriptor table, `EmitsOf` entries, codec call lowering,
   error declarations)? If the descriptor table must exist for one
   kernel, say so — do not let B2 inherit an unscoped framework.

Close with the single highest-risk hole in the B2 design and the
concrete fixture that would expose it.

---

## Context files

- `docs/bytes-plan.md` (v3 — the design under review)
- `docs/a/a45-bytes-values.md` (shipped B1 foundation B2 builds on)
- `docs/bytes-workstream.md` (requirements, acceptance, non-goals)
- `compiler/types.go` (sealing, `seals_from`, nominal-type rule)
- `compiler/parse.go` (`Module.ID`, declaration parsing, ctor branch)
- `compiler/check.go` (world construction, emits, linkage trust)
- `compiler/eval.go` (call dispatch, exhaustiveness)
- `compiler/emit.go` (call lowering, result-union emission)
- `compiler/code.go` (`CAN6010`/`CAN6011` already reserved)
- `std/html/html.can` (future consumer; incoming `seals_from` paths)
