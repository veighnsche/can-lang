# UP15 Jev consultations — findings

25 September 2026 · three fresh SystemOne consultations on the UP15
browser-repair boundary. Requests, responses and metadata are saved in this
directory (`request-{1,2,3}.json`, `response-{1,2,3}.json`,
`response-{N}.metadata.json`, `send.py`, `wording-audit.py`). All prose was
rewritten between requests; the wording audit confirmed zero shared
instruction/criteria/state strings across the three while preserving the
same forensics (59 reachable modules, nine host-op sites), constraints
(frozen check/emit, sealed overlay table, check-time gate, green server
suites, sibling-owned canonical assets) and alternatives. Question IDs
differ per wording by design; the topic mapping is below.

## Questions and advice

| Topic | Wording 1 | Wording 2 | Wording 3 |
| --- | --- | --- | --- |
| Server-only set (TOML/YAML/JSON5, CSRF, Markdown, assets) | stub_and_gate 0.99 | ship_stubs_and_reject 0.95 | reject_with_stubs 0.78 |
| Cookie scope | gate_cookies 1.00 | cookies_server_only 0.96 | restrict_cookies 0.98 |
| Clock sleep / log sink | overlay_timer_sink 0.96 | alternates_for_platform 0.88 | profile_alternates 0.98 |
| Assert identity chain | split_hashing_out 0.77 | extract_hashing 0.97 | isolate_digests 0.96 |
| HTML escape | overlay_escape 0.57 | prove_and_replace 0.90 | single_proven_helper 0.84 |
| Overlay application | rewrite_staged_edges 0.61 | plugin_redirect 0.65 | explicit_tree_edit 0.93 |

Unanimous: stub alternates plus check-time gating for operations with no
browser-native equivalent; cookies server-only; clock/log through browser
alternates that leave server bytes untouched; split the node:crypto
hashing functions out of the shared identity module.

## Disagreement analysis

HTML escape drew one dissenting wording (overlay 0.57, confidence 0.14 —
near a coin flip). The dissenting pull is uniformity: keep every server
byte untouched, as with clock and log. That concern is answered by
mechanism, not vote: the native escaper maps exactly five ASCII code
points (`&<>"'` to `&amp;&lt;&gt;&quot;&#x27;`) and passes everything else
through untouched, verified empirically including astral characters. A
UTF-16 code-unit loop with the same five replacements cannot diverge on
any input, including lone surrogates and NUL; an exhaustive differential
test against the native function (all ASCII plus adversarial Unicode)
supplies the proof. A 400-line module fork to avoid a provably identical
ten-line helper would manufacture drift risk instead of removing it. The
majority advice (canonical edit with proof) is adopted with that
documented rationale.

Overlay application split 2–1 for staged rewriting, with both minority
signals weak (0.61 at confidence 0.21; plugin 0.65 at confidence 0.30).
The plugin pull is wording-driven: request 2 framed the staged tree as a
"verbatim copy", which sounds safer. Mechanism decides: the UP15 contract
requires builds with no alias machinery, and a resolve hook is
alias-shaped process memory, while rewritten staged files are bytes on
disk, diffable against the hashed request inventory. Both directions fail
closed (a missed rewrite surfaces as a post-bundle host operation or an
unresolved import), but only the rewrite keeps the maintained tool a pure
native `Bun.build` call and attributes failures to file plus specifier.
The majority advice with the strongest single confidence (explicit tree
edit 0.93) is adopted with that documented rationale.

## Treatment

Agreement is advice, not proof. The adopted posture is additionally
required by the UP15 task contract (empty app and fixtures must build
with no host imports; failures prevent publication) and by the UP10
residual-risk handoff, which assigned exactly this repair set to
downstream tasks. The sealed-inventory additions (seven overlay entries)
and twelve new capability-gate rejections are flagged to the coordinator
in the UP15 handoff, since the profile table and browser operation
surface are shared across lanes. Implementation proceeds on that basis;
server suites and differential tests verify every canonical edit.
