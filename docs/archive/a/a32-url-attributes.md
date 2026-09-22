# a32 — URL attributes (narrow absolute-https, fused input)

Status: shipped. The URL bundle: `html__attribute__href` and
`html__attribute__src` validating absolute-`https` URLs and
serializing `href='...'` / `src='...'`. Fused raw input, narrow
scope, no brands. New errors `html.invalid_url(value)` and
`html.disallowed_scheme(value)` — both catalogue-listed, so no
error amendment, but the signatures are amended (below).

## Rule

- Admitted: absolute `https` URLs under a restricted ASCII
  authority profile. Authority: first scalar alnum; rest
  alnum, dot, or hyphen; no leading dot/hyphen; no empty
  labels; trailing hyphen rejected; trailing dot admitted
  explicitly, never stripped. Excluded until their own
  contracts: userinfo, ports, IP literals, internationalized
  hosts, percent-encoded hosts. No normalization, no case
  folding, no stripping — the spelling is validated as given.
- Scheme detection is grammar-first: letter, then letters,
  digits, plus, minus, dot, then a colon, all at the start.
  Present scheme must be exactly lowercase `https`, else
  `disallowed_scheme` as an unadmitted spelling (uppercase
  HTTPS included). A colon that is not a scheme token —
  query/fragment position, leading non-letter — earns
  `invalid_url`. Relatives of every form (bare, root,
  protocol) are `invalid_url` as unsupported reference forms;
  base-dependent resolution waits with the base brand.
- Tail (path/query/fragment, undifferentiated — the encoding
  contract is identical): no ASCII whitespace, no backslash.
  Percent escapes pass through undecoded. NUL passes
  validation and fails downstream as `nul_byte`, keeping each
  failure in its domain and the shared worker's NUL arm
  reachable.
- Both composers reuse the shared single-quoted worker and
  seal from its actual result. Query `&` encodes — the reason
  reuse is correct, not merely convenient.

## Why fused, why narrow, why no brand

Three verified walls, not preferences. Branded-base plus
relatives is unbuildable (infinite host domain, no
extraction, no opaque fields, no maps). The full-shape
validator as first specified admitted `%2F`-hosts, `:99999`
ports, and `[bad]` IPv6 — executed counterexamples, not
theory. And `href(url: Html__Url)` cannot call a
string-taking worker — the output-side wall. So: narrow
absolute-only scope, fused raw input, and no `Html__Url` or
`Web__Origin` brands, because a brand no consumer can
serialize is speculative. The brand returns with an
observation primitive; relatives return with base resolution.
An `https` URL is not resource authorization — future
element contracts own `src`-context safety, and `https`
admission must never be read as script-source approval.

## Proof costs

- New shapes proved in /tmp before use: integer subtraction
  on lengths (`#s - 7`), match on int-typed record fields,
  bool params carrying verdicts (no bool literals anywhere —
  state travels as ints and strings).
- The coverage law shaped the code twice: the tail dispatch
  collapsed from three delimiter branches to one call when
  the audit showed two unreachable error arms, and the twin
  admission copies collapsed into one `admit` helper when the
  uppercase path proved structurally dead. Both simplifications
  removed code, not coverage.
- Inexpressible-source ledger, all witnessed by other means:
  LF/CR (classifier-style integer rows elsewhere cover the
  codes; runtime probe executes them), braces (DEL rows cover
  the above-122 arm), backslash (single literal backslash —
  backslashes are literal in can source).
- Decision-table rows cover every match arm including each
  recursion's Ok and error unwinds; 180/180 green with the
  pre-existing 95.
- Independent oracle: the generated `html.ts` runs under node
  over the judge's own counterexamples plus scheme policy,
  trailing-dot admission, and NUL layering — 13/13 green.
  Scratch probe at `/tmp/url-probe.mjs`, not committed.
- `errors.json` regenerated with complete `hit_by_tests`;
  `go test -count=1 ./...`, modcheck, and gramcheck all
  green fresh.

## Inputs

- Jev returned 0.62 on full-vs-narrow — amber, correctly
  carrying no decision weight. The outside review decided:
  narrow wins, with executed counterexamples for the
  validator, the scheme-grammar correction, the output-side
  wall, and the rider deferral — all adopted. The review's
  preference (preserve URL-taking consumers via explicit
  authorization) is recorded as the language-work path that
  reinstates the brand; fused input is the std-side path
  taken now, openly amending those two signatures.

## Still scheduled (not silently dropped)

- `http`/`mailto` and further schemes: each needs its own
  admission slice with justification.
- Ports, userinfo, IP literals, internationalized and
  percent-encoded hosts: each needs its reference contract.
- Base-dependent relatives, `Web__Origin`, and the
  `Html__Url` brand: return together with base resolution.
- The empty-fragment rider stays deferred per review: sound
  but consumerless.
- `classes` and the attributes-maker wait on Collections.
