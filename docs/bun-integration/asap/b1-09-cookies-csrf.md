# B1-09 — Cookies and CSRF primitives

Status: prepared for ASAP execution; implementation and acceptance still pending. Dependencies: B1-06, B1-08. Surface: Library.

Expose typed cookie parsing/serialization and session-bound CSRF generation/verification. Use the existing HTTP headers and crypto contracts. A valid token for one session must fail for another. Do not expose the native thread-local default secret as the normal application contract.

## Where to implement

**Required layout:** follow [filetree and module boundaries](filetree.md). Its feature modules supersede the coarse existing-file locations below; do not append B1 implementation bodies to existing orchestration hubs.

Paths are repository-relative; **new** means a proposed file/directory, not existing code. The shared wiring below is part of this task, not optional cleanup.

| Path | State | Responsibility |
|---|---|---|
| `runtime/platform/cookies.ts` | new | Bun.Cookie/CookieMap conversion and response fields. |
| `runtime/platform/csrf.ts` | new | Explicit-secret, session-bound token operations. |
| `runtime/platform/http.ts` | existing | Repeated Set-Cookie integration. |
| `runtime/test/cookies.test.ts` | new | Attributes, repeats, expiry and injection. |
| `tests/integration/cookies_csrf_test.go` | new | Typed session flow and HTTP integration. |
| `compiler/internal/catalogue/catalogue.json` | existing | Canonical public types, signatures, exact error identities, native lowering and assertion policy. |
| `compiler/internal/emit/program.go` | existing | Operation identity -> runtime target binding, imports and adapter construction. |
| `runtime/modules.json` | existing | Register each runtime module and exact import edges. |
| `distribution/target.json` | existing | Add actually required qualified native APIs; do not change the pinned runtime casually. |

Generated mirrors are produced with `make catalogue`: `compiler/internal/catalogue/generated.go`, `runtime/catalogue.ts`, `std/catalogue/README.md`. Never hand-edit them. See [shared integration contract](integration-contract.md) for specs, module boundaries, fixture ownership and completion rules.

## Proposed API contract

This is interface notation, **not accepted Can syntax or a claim that these operations compile today**. Resolve final names against the current catalogue and express them in existing Can declaration syntax; only primitive gates may add grammar.

```text
cookie::parse(request_cookie_header) -> cookie_collection
cookie::make(name, value, attributes) -> cookie
cookie::serialize(cookie) -> set_cookie_field
attributes = path, domain?, secure, http_only, same_site, max_age?, expires?
csrf::generate(secret, session_id, expires_in) -> token
csrf::verify(secret, session_id, token, max_age) -> bool
```

## Implementation sequence

1. Specify duplicate request-cookie lookup behavior, malformed input, name/value encoding and cookie attributes. Enforce admitted SameSite combinations and cookie prefix rules if the API claims them.
2. Lower parsing/serialization to Bun.CookieMap/Bun.Cookie and immutable projections. Keep multiple Set-Cookie fields separate. Do not manually concatenate unvalidated header strings.
3. Require caller-supplied CSRF secret and nonempty session identifier. Verify using matching algorithm/encoding and explicit age policy; do not rely on a per-thread secret that changes at restart.
4. Distinguish invalid/expired/mismatched tokens returning false from invalid configuration. No secret or raw token in assertion failure diagnostics.
5. Integrate safe response construction and deletion using matching path/domain. Verify serialization against actual response headers.
6. Document application responsibility for deriving the session from authenticated state and checking relevant unsafe HTTP requests. Token cryptography alone is not session authentication.

## Acceptance evidence

- Success: parse/serialize attributes, repeated Set-Cookie, session A token accepted by A, invalidated by B or wrong secret.
- Rejected: CRLF/name injection, unsupported SameSite/prefix combinations, invalid duration and missing session identity.
- Runtime: expiry/max-age, deletion path/domain, malformed token false, concurrent sessions cannot share accepted tokens.

## Fixtures and local assertions

Cookie transformations execute for real. CSRF tests execute native verification; generation/time inputs use the assertion boundary where needed. A minimal loopback form submission must reject an A-token with B-session context. Do not assert whole random token strings.

## Native lowering sketch

This sketch describes the native operation, not a complete Can adapter. Real adapters must return the existing Completion representation, retain source origin and apply declared error boundaries.

```ts
const token = Bun.CSRF.generate(secret, {sessionId, expiresIn});
const valid = Bun.CSRF.verify(token, {secret, sessionId, maxAge});
const field = new Bun.Cookie(name, value, attributes).toString();
```

## Gates and limitations

The native probe verified cross-session rejection, not all expiration and cookie policy behavior. Session management remains application logic; this capability does not invent authentication or a session database.

## Source and proof

- [Pinned native observations](evidence/native-probe-results.json) and [reproducer](evidence/native-probe.ts). Only listed probe behavior was executed.
- [Official Bun documentation](https://bun.sh/docs/runtime/cookies)
- [Official Bun documentation](https://bun.sh/docs/runtime/csrf)

Do not check off this capability until its acceptance cases, local fixture behavior, generated TypeScript evidence and all required real-native runs are recorded. Use [execution queue](execution-queue.md) to continue immediately with the next unblocked capability.
