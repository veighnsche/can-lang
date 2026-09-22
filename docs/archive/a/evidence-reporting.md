# Row 3 (D1) — Evidence reporting and certificate safeguards

Status: shipped. Reporting distinguishes execution from structure;
every safeguard in the verdict's list now has a negative test or a
verified pre-existing check. No admission behavior changed except
one demonstrated defect, repaired with its reproducer.

## Reporting (D1, no behavior change)

- `errors.json` `hit_by_tests` records stub plus expectation
  references, not executions. Pinned by
  `TestCatalogNoPhantomHits`: a certified-but-unexecuted relay
  arm manufactures no hit, and a declared-but-unrealized error
  compiles with empty lists (conservative upper bound, both
  directions). Schema meaning unchanged — documented in
  `catalog.go`, not versioned, since nothing changed silently.
- The relay arm remains a `handled_by` site: handling is
  recorded, hitting is not. Consumers must not read hits as
  proof a kind was observed escaping.

## Safeguard inventory (all negative-tested)

| Safeguard | Status |
|---|---|
| Wrong-kind reconstruction | Pre-existing `TestDiagnoseInvalidRelayKind` |
| Changed field value | Pre-existing `TestDiagnoseInvalidRelayValue` (literal-instead-of-ref pins the invalid-binding branch) |
| Dropped field | Pre-existing `TestDiagnoseInvalidRelayDrops` |
| Unexpected field | Pre-existing `TestDiagnoseInvalidRelayUnexpected` |
| Nonlocal (foreign) call | NEW `TestDiagnoseForeignRelayUntaken` — field-perfect reconstruction over a foreign call stays under the execution law, never certifies |
| Shadowed / duplicate arms | NEW `TestDiagnoseShadowedRelay` — plus the D2 repair below |
| Kind absent from callee outcomes | Pre-existing machinery: exhaustiveness rejects it as a stale arm before the relay check runs (verified by code read) |
| Malformed arm bodies | Fall to the execution law by construction; generic untaken-arm tests cover the path |

## D2 repair (one demonstrated defect, then close)

Duplicate relay-shaped arms compiled clean: the second copy is
dead, but the certificate silenced it (`/tmp` probe, now a
committed test). `relayStatus` no longer certifies an arm
shadowed by an earlier same-kind pattern; the duplicate fails
with the same `CAN4107` family as non-relay duplicates
(`TestDiagnoseShadowedArm` untouched — zero existing-test
changes, zero new codes). Per the stopping condition, D2 closes
here: no other defect demonstrated, no new machinery invented.

## Still scheduled

Row 4 conditional repairs (source-decode rejection), row 5 NUL
policy, then a28. General site-level domain analysis stays
deferred as ordered.
