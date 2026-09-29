# Error brace triad: source comparison

Date: 2026-09-29. These proposed snippets are design mockups; the current compiler does not parse them.

## 1. Ordinary declaration and emitted completion

Source: tests/failure-conventions/retry/src/profiles/profiles.can:9-31.

Current:

~~~can
error unavailable(str key)
fn profile load
    emits [unavailable, forbidden]
    ...
    unavailable(key)
~~~

Proposed:

~~~can
error unavailable{str key}
fn profile load
    emits {unavailable, forbidden}
    ...
    unavailable{key}
~~~

Assessment: improved visual grouping. The declaration, bound, and constructed value are recognizable as three error roles. The keyword and position still distinguish a field list, a type set, and positional values.

## 2. Error value stored in successful data

Source: tests/failure-conventions/retry/src/profiles/profiles.can:35-42.

Current:

~~~can
gone: "bob" => ok retry::rejected<load_failure>(unavailable("bob"))
~~~

Proposed:

~~~can
gone: "bob" => ok retry::rejected<load_failure>(unavailable{"bob"})
~~~

Assessment: improved distinction between the outer record constructor and inner error value. The explicit ok still decides success; the braces do not emit the inner error.

## 3. Empty error

Source: docs/syntax-taste/evidence/2026-09-26/full-review-2cb1bc3/core-probes/owner-test-input/src/ids/ids.can:6-15.

Current:

~~~can
error invalid()
emits [invalid]
invalid()
~~~

Proposed:

~~~can
error invalid{}
emits {invalid}
invalid{}
~~~

Assessment: the declaration and value pair is compact and consistent. The syntax requires no special nullary rule beyond an empty field or argument list.

## 4. Nested generic error data

Source: compiler/testdata/current/coordination/aggregate-composition.can:12-17.

Current:

~~~can
emits [all_failed<a_failure>]
all_failed<a_failure>([codec::invalid_data("a", "type")])
~~~

Proposed:

~~~can
emits {all_failed<a_failure>}
all_failed<a_failure>{[codec::invalid_data{"a", "type"}]}
~~~

Assessment: mixed. The nested punctuation is still dense, but braces identify both error constructors while brackets retain the payload array. Positional values inside braces remain the unfamiliar part.

## 5. Array of pure callables

Source: compiler/testdata/current/assertions/queues.can:107.

Current:

~~~can
callable int () emits [][] actions
~~~

Proposed:

~~~can
callable int () emits {}[] actions
~~~

Assessment: improved. The empty error set and array suffix become distinct at a glance. The type rule still applies [] to the completed callable.

## 6. Deeply nested successful error data

Source: tests/failure-conventions/retry/src/profiles/profiles.can:103-113.

Current:

~~~can
traced_down: "bob" => ok retry::rejected<retry::traced<retry::traced<load_failure>>>(retry::traced<retry::traced<load_failure>>("directory.lookup", retry::traced<load_failure>("profiles.load", unavailable("bob"))))
~~~

Proposed:

~~~can
traced_down: "bob" => ok retry::rejected<retry::traced<retry::traced<load_failure>>>(retry::traced<retry::traced<load_failure>>("directory.lookup", retry::traced<load_failure>("profiles.load", unavailable{"bob"})))
~~~

Assessment: reviewers disagreed. The extra delimiter kind can look noisy; it also makes the innermost error leaf visible and separates its close from the surrounding record closes. The generics and nested records remain the real source of this line's length. I judge the proposed line slightly easier to scan.

## 7. Forwarding, reconstruction, and standard failure

Sources: tests/failure-conventions/retry/src/profiles/profiles.can:66-70, 140-147.

Current:

~~~can
unavailable as failure => do
    load_failure leaf = unavailable(failure.key)
    ok retry::rejected(leaf)

emits []
match call retry::retry(callable blast, 0, 2)
    [_] as standard_failure failure => ok failure.kind
~~~

Proposed:

~~~can
unavailable as failure => do
    load_failure leaf = unavailable{failure.key}
    ok retry::rejected(leaf)

emits {}
match call retry::retry(callable blast, 0, 2)
    [_] as standard_failure failure => ok failure.kind
~~~

Assessment: reconstruction becomes visually distinct from the unchanged error pattern. The standard-failure catch is unchanged; emits {} still excludes only declared domain errors, as emits [] does today.

## Decision

The mockups improve the simple, stored-data, empty, and callable-array cases. The generic aggregate and deeply nested retry assertion remain dense; the proposed error leaves are easier to locate, but the expressions do not become shorter. One independent reviewer judged the delimiter variation as noise, while another judged it as useful structure. The first reviewer also treated unchanged bare match heads as a mismatch; I do not count this as a new mismatch because matching a type and constructing a value already use different forms today.

The cost is unfamiliar positional arguments in braces, partly explained by the matching declaration form. On this exact source comparison, I recommend adopting the three-form design as the proposed grammar. No further design research is necessary to make that recommendation. Implementation must still verify parser/formatter round trips and semantics for generic and nullary errors, stored versus emitted values, forwarding, callable-array precedence, and standard failures. This is a design recommendation, not a claim that the unimplemented syntax already passes those checks.
