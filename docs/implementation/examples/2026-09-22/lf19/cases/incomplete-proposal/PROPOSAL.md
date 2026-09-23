# Deliberately incomplete capability proposal (AE33 review-gate fixture)

Proposed operation: `text::word_count(str value) -> int`.

Supplied evidence: typed Can signature only (name, input type, result type).

Missing: native Bun recipe, error translation (total or emits?), immutability/lifetime
statement, fixture/conformance rows, substitution/raw-adapter/target breakdown,
catalogue identity, registry mirror update, retirement note.

Checklist verdict: **incomplete** — fails the admission review gate. A typed
signature alone never admits a capability; see the admission guide.
