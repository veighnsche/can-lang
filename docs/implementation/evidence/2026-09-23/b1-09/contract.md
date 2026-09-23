# B1-09 Cookies and CSRF contract confirmation

Target: Bun 1.4.2 `Bun.Cookie`, `Bun.CookieMap`, `Bun.CSRF`.
Every row below is an executed observation (probe, unit, loopback or
integration run) unless marked docs. Catalogue additions only: no new
grammar. Surface selection audited in
`docs/bun-integration/asap/evidence/consultations-b1-09/decision-audit.md`.

## Parse and lookup

| Aspect | Contract |
|---|---|
| Order | `cookie::parse` keeps every pair in wire order on `cookie::collection{pairs}`; duplicate names retain all entries |
| Lookup | `cookie::get` returns the first pair value for a name (`option::value<str>`), matching the native getter; absent names answer none |
| Case | names are case-sensitive throughout |
| Malformed | segments without `=` and empty names drop; empty values persist as empty strings; surrounding spaces trim; empty headers parse to no pairs |
| Encoding | values percent-decode; split happens at the first `=` so values may contain `=` |
| Quotes | double quotes neither protect inner semicolons nor strip; quoted values split and keep literal quote characters |

## Serialization

| Aspect | Contract |
|---|---|
| Handle | `cookie::make` validates the name and expiry, then wraps an opaque `cookie::cookie`; `cookie::serialize` renders one Set-Cookie field value and cannot fail |
| Defaults | path defaults carry explicitly: omitted domain, max-age and expiry stay absent; path, secure, http-only and same-site render as given |
| Native shape | rendering passes attributes to the native constructor: `Path`, `Domain`, `Max-Age`, `Expires`, `Secure`, `HttpOnly`, `SameSite` |
| SameSite | closed by type: `cookie::same_site` is strict, lax or none; unsupported spellings are unrepresentable |
| Policy | the adapter claims no prefix or combination policy: `__Host-`/`__Secure-` names and `SameSite=None` without `Secure` serialize natively; browser treatment is documented, not enforced |
| Values | values percent-encode (CRLF, semicolons, unicode); names reject CRLF, semicolons, spaces, `=` and emptiness with `cookie::invalid_cookie{name}`; paths and domains validate natively per field (`{path}`, `{domain}`) |
| Expiry | `expires_ms` outside the native date range fails `cookie::invalid_cookie{expires}`; `max_age` outside the safe-integer range fails `{max_age}`; negative max-age serializes as given |
| Deletion | `cookie::expire` builds an epoch-expiry tombstone with empty value scoped to the matching path and domain |

## Response integration

| Aspect | Contract |
|---|---|
| Separation | repeated Set-Cookie fields stay separate entries from `http::make_server_headers` through response snapshots to the native `Response`; no comma-joining anywhere on the path |
| Round-trip | emitted responses expose each field via `getSetCookie`; parsing the request `Cookie` header of a loopback exchange recovers the values |

## CSRF

| Aspect | Contract |
|---|---|
| Contract | `csrf::generate` and `csrf::verify` take explicit secret plus nonempty session id with fixed base64url/sha256; the native thread-local default is not exposed |
| Tokens | 86-character strings, distinct per generate; assertions never compare whole token strings |
| Failures | malformed, expired and mismatched tokens (wrong session or secret) answer false; only invalid configuration fails `csrf::invalid_config` |
| Config | empty secret fails `{secret}`, empty session fails `{session}`, out-of-range durations fail `{duration}`; no secret or token material enters diagnostics |
| Binding | session binding is strict both ways: unbound tokens fail bound verification and bound tokens fail unbound verification |
| Expiry | token expiry is absolute over `max_age_ms`; a lapsed token fails even under a generous max age |
| Responsibility | deriving the session from authenticated state and covering unsafe requests stays application logic (docs) |

## Error table

| Error | Fields | Meaning |
|---|---|---|
| cookie::invalid_cookie (1341) | reason | name, path, domain, expires, max_age |
| csrf::invalid_config (1342) | reason | secret, session, duration |
