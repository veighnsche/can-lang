# Oxc tooling for authored runtime TypeScript

The root development toolchain pins Oxlint 1.85.0, Oxfmt 0.70.0,
oxlint-tsgolint 7.0.2002, TypeScript 7.0.2, and the Bun 1.4.2 / Node 24
types. `npm run check:runtime` runs strict lint, Oxfmt's check mode,
and TypeScript's no-emit check on maintained `runtime/` and
`tools/runtime/` source. CI runs that command after `npm ci`.

Oxfmt uses its documented 100-column default, two-space indentation,
double quotes, semicolons, and trailing commas. Import sorting stays
disabled so formatting does not reorder imports with side effects.
The one-time baseline formatted 177 authored files. Generated
`runtime/catalogue.ts` and pinned vendor files are excluded. Future
agent edits use `npm run lint:fix:runtime` followed by
`npm run format:runtime`; the check command must then pass.

Oxlint retains its default correctness rules and adds type-aware checks
for floating or misused promises and incomplete switches. A switch with
an explicit default is accepted as exhaustive because the file adapters
intentionally map several native error kinds through that fallback.
Unused inline disable comments are errors. The test-only override for
`await-thenable` reflects Bun's documented `await expect(...).rejects`
pattern, which the pinned Bun types report as returning `void`.
`unbound-method` is test-only disabled because tests retain native
prototype methods to instrument their receivers. Production code keeps
both rules.

This focused configuration avoids enabling the entire `suspicious`
category. A trial produced 833 findings, dominated by type assertions
and generic-argument style warnings. Each additional rule should be
adopted with a clean baseline and behavior review. Three fresh Jev
consultations about format rollout and lint scope are saved in
`evidence/2026-09-23/oxc-tooling/`; their agreement is advisory.

References: [Oxc coding-agent guidance](https://oxc.rs/docs/guide/usage/coding-agents.html),
[Oxfmt configuration](https://oxc.rs/docs/guide/usage/formatter/config),
[Oxlint type-aware linting](https://oxc.rs/docs/guide/usage/linter/type-aware),
and [Bun's async assertion examples](https://bun.sh/docs/test/writing-tests).
