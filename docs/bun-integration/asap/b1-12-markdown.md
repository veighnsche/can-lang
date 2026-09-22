# B1-12 — Markdown rendering and structured processing

Status: prepared for ASAP execution; implementation and acceptance still pending. Dependencies: none. Surface: Library.

Expose native Markdown rendering and structured rendering through supported Bun callbacks. Default rendered HTML is an ordinary string. Safe HTML requires a separately qualified path using the existing Can HTML contract; the probe preserved both script markup and javascript: URLs in default native output.

## Where to implement

Paths are repository-relative; **new** means a proposed file/directory, not existing code. The shared wiring below is part of this task, not optional cleanup.

| Path | State | Responsibility |
|---|---|---|
| `runtime/platform/markdown.ts` | new | Native options, string rendering and callback bridge. |
| `runtime/platform/html.ts` | existing | Reuse safe nodes/URLs and extend missing safe Markdown elements deliberately. |
| `runtime/test/markdown.test.ts` | new | Native structure and hostile markup regressions. |
| `tests/integration/markdown_test.go` | new | String versus safe HTML typing and response integration. |
| `compiler/internal/catalogue/catalogue.json` | existing | Canonical public types, signatures, exact error identities, native lowering and assertion policy. |
| `compiler/internal/emit/program.go` | existing | Operation identity -> runtime target binding, imports and adapter construction. |
| `runtime/modules.json` | existing | Register each runtime module and exact import edges. |
| `distribution/target.json` | existing | Add actually required qualified native APIs; do not change the pinned runtime casually. |

Generated mirrors are produced with `make catalogue`: `compiler/internal/catalogue/generated.go`, `runtime/catalogue.ts`, `std/catalogue/README.md`. Never hand-edit them. See [shared integration contract](integration-contract.md) for specs, module boundaries, fixture ownership and completion rules.

## Proposed API contract

This is interface notation, **not accepted Can syntax or a claim that these operations compile today**. Resolve final names against the current catalogue and express them in existing Can declaration syntax; only primitive gates may add grammar.

```text
markdown::render_text_html(source, options, limits) -> str
markdown::render_safe(source, safe_options, limits) -> html::safe
markdown::render_with(source, typed_handlers, limits) -> rendered_result
structured output only where actual native callbacks expose required structure
```

## Implementation sequence

1. Qualify Bun.markdown.html and render callback inputs/order. Do not invent a token/AST API because callbacks exist; record whether callbacks receive source text, escaped text or already-rendered children.
2. Return native HTML as str with an explicit raw-HTML policy. Prevent implicit conversion of that string into html::safe or a trusted HTTP HTML response.
3. Prototype safe rendering through native callbacks using private trusted construction. Disable raw HTML blocks/spans; enforce URL scheme and attribute rules; preserve escaping exactly once.
4. Audit native callback completeness before claiming safe mode. If callbacks cannot expose enough structured information for the existing builder, ship string rendering and keep safe mode blocked with a reproducer; never regex-sanitize HTML or brand it safe.
5. Extend the safe HTML element allowlist only for required Markdown structures such as code/pre, with precise attribute policy. Reuse existing node/URL validation rather than duplicate a sanitizer.
6. Specify options, input/output budgets and callback failure bounds. Native synchronous parsing cannot be interrupted by a JavaScript timeout; do not promise a hard per-render deadline without an execution strategy.
7. Add ordinary display, safe HTTP response and structured-renderer examples, with attached assertions that execute native parsing.

## Acceptance evidence

- Success: headings, lists, code fences, tables, links and Unicode; safe rendering can enter response_html only through its typed result.
- Rejected: raw rendered str used as html::safe; forbidden URL scheme, invalid option, unsupported callback type or unhandled callback error.
- Runtime: script tags, event attributes, javascript/data URL policy, encoded schemes, quote injection, raw HTML, malformed nesting and double escaping.
- Native rendering callbacks must be exercised; no handwritten Markdown parser.

## Fixtures and local assertions

Markdown conversion executes for real in attached tests. Keep a hostile-input corpus and stable expected safe output. Test rendered browser behavior with the existing integration browser harness when safe mode is implemented, not only string snapshots.

## Native lowering sketch

This sketch describes the native operation, not a complete Can adapter. Real adapters must return the existing Completion representation, retain source origin and apply declared error boundaries.

```ts
const ordinaryHTML = Bun.markdown.html(source, nativeOptions);
// ordinaryHTML stays str.
// Separate safe path: Bun.markdown.render(source, verifiedCallbacks, safeNativeOptions).
```

## Gates and limitations

Disabling raw HTML alone does not prove links safe. Existing html.ts lacks some Markdown elements; a string-based callback API may not directly create typed nodes. That compatibility is an early prototype gate, not a license to trust native output.

## Source and proof

- [Pinned native observations](evidence/native-probe-results.json) and [reproducer](evidence/native-probe.ts). Only listed probe behavior was executed.
- [Official Bun documentation](https://bun.sh/docs/runtime/markdown)

Do not check off this capability until its acceptance cases, local fixture behavior, generated TypeScript evidence and all required real-native runs are recorded. Use [execution queue](execution-queue.md) to continue immediately with the next unblocked capability.

## Callback execution constraint

Native Markdown render callbacks are synchronous string transformations. Can handlers lowered as async Completion-returning functions cannot be passed directly into them. The initial safe renderer uses compiler-private synchronous native callbacks with validated fragment construction. Any public render_with API needs a proven staged callback/structure design compatible with Can handler errors and ownership; it must not stringify a Promise or bypass checked invocation. Keep that subpart visible if the gate fails. Do not redesign Can callables merely to fit this native API.
