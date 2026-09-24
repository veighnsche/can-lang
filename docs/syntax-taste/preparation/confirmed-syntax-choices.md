# Confirmed syntax choices for the planned design

**Status after the upgrade (24 September 2026):** The recorded user choices were subsequently implemented: explicit bind captures in T02 and dependency-qualified imports in T03. The original choice record below is preserved; its pre-implementation tense is historical. See the [current reconciliation](../post-upgrade-reconciliation-2026-09-24.md).

24 September 2026. These are the user's answers during design preparation.
They are binding inputs to P8 specification integration and the eventual
implementation list. They describe **planned** syntax; the current compiler
and existing `decisions.md` grammar have not yet been changed.

## Ordinary-data pattern capture — DI-01

The user selected **`bind name`**, including nested ordinary-data captures.
The question's complete scope was:

- Bare names such as `paid` are checked nominal case references in ordinary
  data matching; an unknown `decliend` is a compile error, not a binder.
- `bind item` intentionally captures a whole ordinary value.
- `[bind first, _, ...rest]` and `box(bind item)` show the same binding intent
  inside arrays and records. `_` remains discard/fallback and `...rest`
  remains its visibly marked array-tail capture form.
- Boolean/literal/range patterns and the separate error/completion `as alias`
  rules were not part of the question and are unchanged by this answer.

The [question packet](pattern-syntax-question.md) contains the alternatives
and examples; the [nested probe](pattern-nested-probe.md) and
[Jev findings](jev-pattern/findings.md) explain the technical basis. P8 must
specify exact grammar, expected-type checking, nested coverage, diagnostics,
generic specialization, field visibility and generated behavior. The answer
does not approve any unrelated pattern rewrite.

## Dependency-qualified source imports — DI-05a

The user selected **qualifying the direct dependency key and package in a
`uses` entry**, with an optional file-local alias:

```text
uses [billing::model as bill_model, crm::model as crm_model]
```

Here `billing` and `crm` are keys in the importing project's direct dependency
table; each contains a package called `model`. Inside a dependency, lookup is
relative to that dependency's own table. Canonical package identity is the
resolved project/package instance, not the local alias. Local package imports,
reserved catalogue names, direct/transitive visibility, private-package
checks, locks and native module mapping remain in scope. The user did not
select a manifest import-handle table. The [question packet](package-import-syntax-question.md)
and [technical alternatives](core-alternatives.md#di-05a--independent-package-identity)
record the comparison. P8 must settle lexical ambiguity with existing
`package::member` expressions, exact diagnostics and versioned instance
identity; the example is the selected source form.
