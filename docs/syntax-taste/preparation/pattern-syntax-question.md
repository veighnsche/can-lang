# DI-01 syntax question: intentional ordinary-data capture

24 September 2026 · historical comparison packet. The user subsequently
selected `bind name`; the complete accepted scope is in
[confirmed syntax choices](confirmed-syntax-choices.md#ordinary-data-pattern-capture--di-01).

The [current-source probe](pattern-nested-probe.md) confirms that `decliend`
and nested `paidd` can silently become whole-value binders. Three fresh
[Jev consultations](jev-pattern/findings.md) disagreed at low confidence.
Engineering assessment favors a uniform rule: an ordinary-data bare name is
a checked nominal leaf test, never an implicit binder. An unknown name is a
compile error. A distinct visible form captures a value. This avoids a
nonvariant-to-variant field edit silently turning a binder into a case test,
or the reverse edit turning a case test into a binder. The selected variant
identity rule is now in [accepted technical decisions](accepted-technical-decisions.md#di-07--extensional-closed-variant-compatibility).
Constructor-only testing and type-contextual binding were
compared in [core alternatives](core-alternatives.md#di-01--pattern-intent-and-typo-diagnostics).

The examples below show three **spellings of the same semantic rule**. Each
uses bare `paid`, `pending`, `declined` as checked nominal cases; `_` is an
intentional discard/default. Every spelling rejects `decliend` and requires
an explicit capture for an arbitrary value, including nested fields and array
elements. The existing `...rest` is already a visibly marked array-tail
capture; these examples retain it. Boolean/literal/range patterns and error
completion `as alias` keep their separate current meanings.

| Candidate spelling | Whole value | Array and nested record | Agent-relevant consequence |
| --- | --- | --- | --- |
| `bind name` | `bind item => ok item` | `[bind first, _, ...rest]`; `box(bind item)` | A word marks every ordinary capture, with a direct “missing bind” diagnostic. It adds source tokens; total task-token effect needs measurement. |
| `as name` | `as item => ok item` | `[as first, _, ...rest]`; `box(as item)` | Reuses the existing `as` word concept from error payload aliases but in prefix position for ordinary patterns. Parser/context diagnostics must keep the two uses clear. |
| `$name` | `$item => ok item` | `[$first, _, ...rest]`; `box($item)` | A sigil is compact in source but needs a new lexer token and may tokenize differently for agent models. No task-token advantage is established. |

In each candidate, this corrected closed match is exhaustive for the original
three-leaf variant and becomes non-exhaustive after adding `refunded`:

```text
match payment
    paid => ok "paid"
    pending => ok "pending"
    declined => ok "declined"
```

An explicit fallback remains available:

```text
match payment
    paid => ok "paid"
    _ => ok "other"
```

The user can select another spelling. A selection of `bind`, `as` or `$`
applies to ordinary-data whole-value, nested field and ordinary array-element
captures; it does not silently change error/completion patterns or the
`...rest` tail form. If the user wants those companion forms changed too, that
must be stated and analyzed before the grammar is recorded.
