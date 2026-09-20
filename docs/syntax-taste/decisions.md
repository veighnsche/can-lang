# Can surface design — current decisions

This is the authoritative record of current design choices for Can's AI
coding-agent audience, including syntax and the Can-to-Bun architecture. It takes
precedence over conflicting rules in other design documents. These are design
requirements, not claims about implemented or compiler-validated behavior.

Decision: SURFACE-069.

Only choices explicitly confirmed in this record establish the current design.
Everything else, including earlier design documents and the existing
implementation, is historical material, not a default or fallback specification.
An unrecorded feature or rule remains undecided; it is not implicitly retained.
This authority rule does not authorize implementation changes or deletion.

Effects are deferred for later reconsideration; no effect-checking requirement
or effects mechanism is currently accepted. No effects or foreign-code mechanism
is inherited from earlier drafts or the existing implementation. The separately
recorded Can-to-Bun platform boundary remains in force.

## Maintaining this reference

- Record confirmed choices before asking the next conversational question.
- Bundle closely related syntax contexts into one clearly scoped question when
  they share a rule, rather than asking about each list form separately.
- Before asking for a syntax choice, explain any required companion syntax or
  consequences, including ignored-field notation. Do not treat agreement with
  a shown form as approval of syntax that was not shown or explained.
- Keep only current rules, useful examples, constraints and unresolved questions.
- Replace changed decisions in place; do not retain rejected alternatives or history.
- Preserve semantics established in this record when a sketch omits details. Suggestions and
  explanatory examples do not establish additional decisions.
- Consult Jev for technical design decisions with the relevant context supplied;
  user taste choices recorded here are not Jev validation.

Decision IDs identify the applicable choices; they are not a required sequence.

## Design direction

Native AI judgments are foundational to Can's purpose and name, not an optional
library integration added after the language is designed. Probabilistic judgment
and branching are central design concerns, including how they relate to pattern
matching. Design effects, asynchronous execution, concurrency and assertion
support around concrete native-judgment use cases. This establishes direction,
not a particular judgment spelling or a change to ordinary pattern-match semantics.

Decisions: SURFACE-074, SURFACE-083–084.

Can follows a primarily functional design paradigm. Functions are first-class
values and may be passed as inputs to other functions. Future design proposals
should take this functional direction as their baseline without treating
unselected syntax or semantics as confirmed.

Can is designed for AI coding agents. Human authoring convenience must not be
prioritized in language design; inconvenience for humans is acceptable.
Evaluate design trade-offs for AI coding agents rather than treating fewer
keystrokes or inline convenience for humans as goals.

All function declarations are named and top-level. Nested and anonymous
function declarations are not supported. Functions remain first-class values;
`callable <name>` creates references, and `near` inputs provide captures from the
reference-creation scope.

Callable error/effect contracts remain undecided.

## Naming

All user-defined names use lowercase `snake_case`, including functions, inputs,
locals, fields, assertions, records, choices and errors. Types and errors do not
use a different capitalization convention.

```text
fn int calculate_total
record order_item
choice payment_method
error 1003 invalid_quantity
```

These are declaration-head fragments illustrating naming only. User-defined
names that do not follow lowercase `snake_case` are compile-time errors.
Digits are allowed after the first letter in names, for example `vector2`.
Names must start with a letter; leading underscores in names are forbidden.
Letters in names are restricted to ASCII English `a`–`z`, consistent with the
lowercase naming rule. Unicode letters are not allowed in names.
Language keywords are reserved and forbidden as user-defined names.
Only single underscores between name parts are allowed. Consecutive underscores
and trailing underscores in names are compile-time errors.
The wildcard pattern is the standalone `_`; it is not a name and remains allowed.
Duplicate names in the same scope are compile-time errors, including collisions
across declaration kinds such as records, functions and variables. This is not a
project-wide uniqueness requirement. Inner scopes may shadow names from outer
scopes; a reference resolves to the nearest enclosing declaration of that name.
This does not permit duplicate declarations in the same scope. The precise
scope boundaries remain undecided.
Other detailed identifier rules
remain undecided. This naming choice does
not rename language keywords or the special `ok` completion marker, and does
not decide name-resolution rules when type and value names coincide.

## Packages

An executable starts at the function `main` in package `main`, rather than a
configurable entry-point name. It uses ordinary function declaration syntax,
including mandatory `emits` and `asserts`, rather than a special entry-point
form. Its success type is `void`, written `fn void main`; successful completion
uses bare `ok` and maps to process exit code zero. Command-line arguments are
supplied as an ordinary `str[]` input in `given`, so assertions can supply
them explicitly. The input's required name, if any, and which runtime argument
entries it contains remain to be specified.
The entry-point function `main` must appear explicitly in its defining file's
`provides` list.

There is no package-level `emits` list. The package's declared errors define
its own error set. This replaces the historical module-level error list, not
function-level `emits`, which remains required. Detailed relationships between
package-declared errors and forwarded errors from other packages remain open.

A file's package declaration uses `package <name>`:
Header order is mandatory: `package`, then its indented `provides` list, then
its indented `uses` list.

Packages are flat siblings, not a hierarchy. Package names must be unique within
the project. Each folder defines one package; its Can source files share the
same package name and may have different filenames. A package may span multiple
files in that folder. Each file has its own `uses` and `provides` lists, which
may differ between files. Package names
use the established naming rules. Files in the same package may access each
other's declarations directly, without imports, including declarations absent
from `provides`. The `provides` lists control access from other packages.
Each file's `provides` list may name only declarations defined in that file.
Imports in `uses` and their aliases apply only within the declaring file;
other files in the same package declare their own imports.
Underscores may group related
names by convention, such as `shop_payments`, without creating parent or child
packages. `::` separates a package name or alias from a declaration; it is not
part of a package name and does not express nested package paths.

A directory named `internal` restricts package imports, following Go's
directory-based rule: a package in or below `internal` may be imported only
from within the directory tree rooted at that `internal` directory's parent.
Packages are not required to live under `internal`. This is an import-visibility
boundary based on filesystem location, not hierarchical package naming.

```text
package shop
```

Imports use `uses` with a single-line bracket list, indented one level under
the package declaration:

```text
package shop
    uses [math, text]
```

Imports are package-only. Individual declarations cannot be imported directly
into local scope. Access imported declarations through the package name or its
approved alias, using `::`, for example `call math::add(2, 3)`.

The entries above illustrate layout only. Trailing commas are forbidden, as
with other comma-separated forms. Function declarations remain top-level;
the indented import list does not nest functions inside the package header.

Public declarations are listed in a single-line `provides [...]` list in the
package header, rather than marked individually with `pub`:

```text
package shop
    provides [calculate_total, order_item]
    uses [math, text]
```

The `provides` list is indented one level, like `uses`, and has no trailing
comma. This selects the public-declaration list form; exact visibility and
ownership rules remain to be specified.

Both `provides` and `uses` remain explicit when empty; do not omit an empty
list:

```text
package shop
    provides []
    uses []
```

Package-qualified names use `::`, as in:

```text
call math::add(left, right)
```

This selects the qualification separator only. Field, property and method
access continue to use dots.

Import aliases use `as`, with the imported name first and its local alias
second:

```text
uses [math as numbers]
```

This alias lets the file refer to that package as `numbers`, for example
`call numbers::add(left, right)`. This selects alias syntax and direction;
collision and shadowing rules remain undecided.

These choices select package-header and name-qualification syntax.
Import targets, visibility details, other placement
requirements and name-resolution rules remain undecided; no other old
package-header rules are inherited.

## Generic type spelling

Generic type applications use angle brackets, for example `box<int>`.
Generic record, function, choice and error declarations place type parameters immediately
after the declared name, also in angle brackets:

```text
record box<item>
    item value
```

The corresponding function-header shape is `fn item identity<item>`.
Choice and error declaration-head shapes include `choice outcome<item>` and
`error 1004 rejected<item>`. Error fields may use those type parameters.
Type-parameter names follow the same `snake_case` rule as other user-defined
names.

When type arguments are supplied explicitly, they immediately follow the name
in calls, callable references, record construction and error construction:

```text
call identity<int>(3)
callable identity<int>
box<int>(3)
rejected<int>(42)
```

These choices select spelling and placement only, not whether explicit type
arguments are mandatory. Generic typing, constraints, inference, assertions
remain undecided.

## Integers

`int` is an arbitrary-precision integer type, not a fixed-width type such as
64-bit integers. Its range has no language-defined fixed bound; actual execution
remains subject to available resources. This selects integer size only, not
other arithmetic, literal, conversion or fault rules from the old implementation.

## Decimal numbers

Can does not include an exact-decimal type. The old implementation's `dec`
type and exact-decimal literal syntax are not retained in the current design.
No implementation removal is authorized by this design decision.

## Numeric literal spelling

Scientific notation is allowed, for example `1.5e3`. Underscore digit separators
are not allowed: write `1000000`, not `1_000_000`. This selects literal spelling
only; detailed exponent grammar and exponent-form typing remain to be specified.

Non-decimal integer literals are allowed with these prefixes:

```text
0xff
0b1010
0o17
```

These denote hexadecimal, binary and octal integers respectively. The rule
against underscore digit separators applies to these forms as well.

## Floating-point numbers

Can supports `float` using IEEE 754 binary64, matching JavaScript/Bun's native
`number` representation and precision. For example, `0.1 + 0.2` evaluates to
approximately `0.30000000000000004`, not an exact decimal `0.3`.

Ordinary decimal-point literals such as `0.1` and `3.14` denote `float`;
integer literals such as `3` denote `int`. Other literal forms, conversions
and language rules for infinity, `NaN` and other special values remain separate
unresolved decisions. This selection does not
adopt JavaScript's implicit coercions or change arbitrary-precision `int`.

Mixed `int`/`float` arithmetic is a compile-time error unless an explicit
conversion makes the operand types agree. There is no implicit integer-literal
exception: `3 + 0.5` is rejected, while `3.0 + 0.5` is valid float arithmetic.
Mixed `int`/`float` equality and ordering are also compile-time errors unless
an explicit conversion makes the operand types agree. For example, `3 is 3.0`
and `3 < 3.5` are rejected.

Numeric conversions use ordinary standard-library function calls with `call`,
not special cast syntax. Conversion function names, rounding and
conversion-failure behavior remain undecided.

## Arithmetic operator spelling

Subtraction uses `-`, multiplication uses `*`, division uses `/`, and remainder
uses `%`. This selects their spelling; operand typing and numeric behavior are
separate decisions.
Unary negation uses a leading `-`, as in `-amount`. Unary `+` is forbidden.
Exponentiation uses `**`, as in `base ** exponent`.
This does not decide sign spelling inside scientific-notation exponents.

## Bitwise operator spelling

Bitwise operations use symbolic operators: `&` for AND, `|` for OR, `^` for
XOR, `~` for NOT, `<<` for left shift and `>>` for right shift.
The `|` spelling retains its existing meaning for alternatives in patterns.
This selects spelling only; operand typing and shift behavior remain separate
decisions.

## Expression grouping

Parentheses may group expressions to control calculation order, for example
`(left + right) * factor`. This does not introduce anonymous tuples or change
the success marker syntax.

## Boolean operators

Use the words `and`, `or`, and `not` for boolean operators, rather than `&&`,
`||`, and `!`.

```text
left and right
left or right
not ready
```

This selects spelling only. Precedence, associativity and eager versus
short-circuit evaluation remain undecided.

## Comparison spelling

Chained comparisons are allowed, for example `lower < value < upper`.
Comparisons are not restricted to two operands. Parentheses remain available
for grouping expressions. The precise rules for comparison chains remain to
be specified.

Use `is` for equality and `is not` for inequality:

```text
left is right
left is not right
```

This selects equality and inequality spelling only; it does not introduce an identity test
or change comparison type rules.

Ordering comparisons use the symbols `<`, `<=`, `>`, and `>=`, rather than
word-based spellings:

```text
width < 6
width >= 6
```

## String literals

Ordinary string literals use double quotes only, such as `"hello"`.
Single quotes are not an alternative string delimiter. An unrecognized escape
preserves the backslash literally, along with the following character, rather
than causing a compile-time error. Ordinary strings support
`\t` for tab, `\r` for carriage return and `\0` for the null character, as well as
backslash escapes, including `\n` for a newline, `\"` for a double quote, and
`\\` for a backslash:

```text
"line one\nline two"
"She said \"hello\""
"C:\\files"
```

Can has no string interpolation. Multiline strings use triple double quotes:

```text
"""
line one
line two
"""
```

Triple-quoted strings interpret backslash escapes the same way as ordinary
strings. Indentation spaces inside multiline strings are preserved exactly;
shared indentation is not automatically stripped. A newline immediately after
the opening triple quotes or immediately before the closing triple quotes is
preserved. The remaining ordinary-string escape inventory remains undecided.
Assertions still occupy one source line; this
choice does not create an exception for multiline literals inside assertions.

Raw strings use a lowercase `r` prefix, for example `r"C:\files\notes"`.
Backslashes in raw strings are literal rather than escape introducers.
A double quote ends a raw double-quoted string and cannot occur within its
contents. There is no doubled-quote escape. To include a double quote,
concatenate an ordinary string containing an escaped quote with `+`, for example:

```text
r"before" + "\"" + r"after"
```

Raw multiline strings use `r"""..."""`. Backslashes are literal, and individual
double quotes may appear inside. Three consecutive double quotes end the
string. Like other multiline strings, indentation and boundary newlines are
preserved exactly.

Use `+` for string concatenation:

```text
"Hello, " + name
```

This selects concatenation syntax; it does not introduce implicit conversion
of non-string values to strings.

Strings use bracket indexing and slicing, like arrays:

```text
text[index]
text[start:end]
```

This selects syntax only. The indexing unit, result types, bounds, endpoint
inclusion, omitted endpoints and failure behavior remain undecided.

## Step separators

Separate steps occupy separate source lines. Semicolons are not allowed as
statement terminators or separators. Semicolons inside strings or comments
are ordinary content, not syntax separators.

## Comma-separated forms

Trailing commas are forbidden across comma-separated syntax, including call
arguments, record and error constructor arguments, array literals, declaration
lists such as `emits`, patterns and callable input-type lists. Use the same rule
consistently rather than different trailing-comma policies for each form.

## Comments

Line comments start with `//`:

```text
// Explain the intent
```

Block comments use `/* ... */` and may span multiple lines:

```text
/* Explain the intent
   across multiple lines. */
```

Block comments may nest. Each `*/` closes the innermost open `/*`, so an outer
block comment can enclose code that already contains block comments.
Documentation comments start with `///`, a special marker distinguishable by
tools from ordinary comments. Each documentation line uses this marker.
Documentation text supports Markdown, including lists, emphasis and code
examples, within the existing `///` comment syntax.
Documentation comments go immediately above and attach to the declaration
they describe.
Record fields, function inputs, local values, top-level values and method
receivers are exceptions: their documentation follows the declaration on the
same line, using `///`, for example
`int width /// Horizontal size.` or `int left /// First factor.`
Documentation is mandatory only on function and record declarations. Documentation
on all other declarations is optional; when present, it uses the placements
specified above.

### MCP reading levels

Can should provide an MCP server for AI coding agents with three file-reading
levels:

- High level: documentation and declaration signatures, including function
  receivers, inputs and declared errors.
- Middle level: the high-level view plus all assertions.
- Lowest level: the entire code verbatim.

This records a tooling requirement, not authorization to implement it during
the syntax review.

## Callable types

Decisions: SURFACE-077–078.

Callable types put the return type first: `callable <return type> (<input types>)`.
A function-valued input follows the usual type-before-name declaration shape:

```text
given
    callable int (int) transform
```

This selects the callable-type shape only. Callable error/effect contracts remain
undecided; their omission in this sketch does not establish an empty contract.

Invoke a function-valued input or other callable value with the same `call`
syntax used for named functions:

```text
call transform(5)
```

`callable` creates a function reference; `call` invokes a function. This invocation
syntax does not settle callable error/effect contracts or introduce implicit
error propagation.

## Named function references

Decision: SURFACE-076.

Use `callable <name>` to refer to an existing named function as a value. The explicit
`callable` marker is required; a bare function name is not a function-value reference.
Creating a reference does not invoke the function.

```text
call numbers.map(callable double)
```

This selects function-reference syntax, not the `map` library contract.

### Near inputs

Decision: SURFACE-082.

Inside `given`, `near <type> <name>` declares an input captured from the scope
where `callable <name>` creates the function reference. The surrounding immutable
binding must have exactly the declared input name and match its declared type.
Creating the callable is a compile-time error if any `near` input lacks an
in-scope immutable binding with that exact name and matching type. A differently
named binding does not satisfy the requirement.

```text
fn int multiply
    emits []
    given
        near int factor
        int number
    asserts
        example: 3, 4 => ok 12
    ok factor * number
```

With an immutable `int factor` available in the reference-creation scope,
`callable multiply` captures that binding. The resulting function value receives
`number` when invoked; `factor` is already supplied. Thus the proposed library
call shape `call numbers.map(callable multiply)` supplies each element as `number`.
This does not establish the `map` library contract.

Assertions supply both near and ordinary inputs explicitly in `given`
declaration order, as in `example: 3, 4 => ok 12` above.

Direct calls supply all inputs explicitly in `given` declaration order,
including inputs marked `near`. For example, `call multiply(3, 4)` supplies
both `factor` and `number`; it does not capture a surrounding `factor` binding.
`near` affects capture when creating a callable, not argument supply for a
direct call. Name lookup follows the approved nearest-scope shadowing rule.
This does not introduce a
`let` keyword.

## Top-level values

Top-level values use the same type-before-name declaration syntax as immutable
locals, with no `const` or `let` keyword:

```text
int max_retries = 3
```

Both top-level and local bindings are immutable; the difference is their scope.
Allowed initializer expressions, initialization order and related rules remain
undecided.

## Functions and local values

Decisions: SURFACE-001–006, SURFACE-012–016, SURFACE-023, SURFACE-071,
SURFACE-073.

Scope is indentation-based, with four spaces per indentation level.
Tabs used for indentation are a compile-time error.
A function starts with `fn <success type> <name>`.
For a record method, a single-line `on <record type> <receiver name>` declaration
appears immediately below the function header, at function-body indentation.
The type comes first; the name identifies the immutable receiver value inside
the function body. A method has exactly one record receiver and remains a
top-level function declaration. For example, this declaration-head fragment
introduces a receiver named `shape` of record type `rectangle`:

```text
fn int area
    on rectangle shape
```

Record methods use dot-call syntax, like array and string methods:

```text
call panel.area()
call panel.resize(5, 4)
```

The value before the dot supplies the receiver. Parentheses contain the ordinary
arguments, excluding the receiver.

`callable panel.area` takes a record method as a callable value and captures the
immutable `panel` value as its receiver. The receiver is not an argument when
that callable is invoked. Existing `near` capture rules remain unchanged.

Record methods support chaining on one source line. One leading `call` covers
the whole chain; each subsequent method receives the preceding method's returned
value. For methods declaring `emits []`, an example is:

```text
call panel.resize(5, 4).area()
```

Here `resize` returns a new rectangle and `area` operates on that returned value.
This does not settle completion handling for chains with nonempty error sets.

Method assertions separate the receiver value, ordinary inputs and expected
completion with two `=>` separators:

```text
example: rectangle(5, 4) => 2 => ok 40
```

Here the receiver value is `rectangle(5, 4)`, the ordinary input is `2`, and the
expected completion is `ok 40`. This illustrates the assertion layout for a
method with one ordinary input. When there are no ordinary inputs, the middle
is empty and both arrows remain:

```text
example: rectangle(5, 4) => => ok 20
```

Ordinary non-method assertions retain their existing layout.

After the optional `on` line, sections appear in this order: `emits`, `given`, `asserts`, then business
logic. The `asserts` section is mandatory for named function declarations.
Ordinary input declarations appear only inside `given`, using type before name.
Functions with no inputs omit the `given` section; do not write `given []`.
The `emits` and `asserts` requirements remain unchanged.
Business logic begins only after the `asserts` body, by dedenting back to
function-body indentation. There is no separate business-logic heading.

```text
fn int add
    emits []
    given
        int left
        int right
    asserts
        small_sum: 1, 2 => ok 3
        larger_sum: 2, 3 => ok 5
    ok left + right
```

`int result = expression` declares an immutable, explicitly typed local.
Unnecessary local bindings must be rejected at compile time when the binding
can be replaced by the direct expression without changing behavior. This is
a compiler error, not a style preference or warning. For example, reject:

```text
int result = left + right
ok result
```

Require the direct form instead:

```text
ok left + right
```

Bindings remain available where needed. The precise, mechanically checkable
boundary beyond this example remains to be designed, including preservation of
evaluation order, evaluation count, typing and `near` capture requirements.
This decision does not introduce a `let` keyword.

The success marker is lowercase `ok` everywhere: completions, assertion
expectations and success-match arms. For a function returning a value,
`ok expression` completes it successfully with the value of the entire following
expression. No completion
parentheses are required: write `ok left + right`. Its one whole payload must match the declared success type. Multiple named fields can
be returned together as a record. Anonymous tuples are outside the current design.

Use `fn void <name>` for a function whose successful completion carries no value.
Successful completion and assertion expectations for such a function use `ok`.
This does not establish `void` as a type usable for inputs, fields or locals.

`emits [...]` is an authored upper bound on function-specific domain errors.
Runtime failures such as fatal out-of-memory conditions are outside `emits`;
this does not establish a runtime recovery or supervision policy. The implementation may
produce only listed domain errors, and callers must handle or forward the declared
set. An empty set is written explicitly as `emits []`; it does not promise freedom
from primitive faults or runtime/resource failures. Platform error mapping
remains to be designed within the compiler-owned Bun boundary.

`given` declares inputs. There is no `given / when / then` business-logic grammar.

## Assertions

Decisions: SURFACE-007–011, SURFACE-017–018, SURFACE-042.

Every assertion is named and runs the function with positional inputs in `given`
declaration order, checking the expected whole completion:

```text
small_sum: 1, 2 => ok 3
```

Every complete assertion must occupy one source line, including its inputs and
expected completion. There are no grouping parentheses around the input list.
Record and error constructor parentheses remain. Success expectations use
`ok expression`, or bare `ok` for void; error expectations use the error
constructor and complete payload.
Inputs must satisfy the full function signature.

For a zero-input non-method function, an assertion has no input text between its colon
and arrow:

```text
default_value: => ok 3
```

## Calls and completion handling

Call arguments allow array spread, for example `call combine(...values)`.
The array's elements become separate arguments; `call combine(values)` instead
passes the array as one argument. Argument-count and typing rules for spread
remain to be designed.

Variadic parameters collect any number of arguments into an array. Their
declaration uses the array type followed by three dots and the parameter name:

```text
given
    int[] ...values
```

For example, `call sum(1, 2, 3)` supplies the collected array `[1, 2, 3]`;
`call sum(...scores)` spreads an existing array into those arguments.
Each function may declare at most one variadic parameter, and it must be the
last parameter. Empty argument slots are forbidden, including in the variadic portion.

Decisions: SURFACE-024–031.

Invoke functions with `call`, using positional arguments in input declaration
order and preserving written evaluation order. A call whose declared error set
is empty can bind its successful payload directly:

```text
int sum = call add(1, 2)
```

Zero-input calls retain empty parentheses: `call default_limit()`.

This executes once and binds an immutable value. It does not introduce general
completion storage or implicit error propagation.

Call argument lists must stay on one source line; multiline call-argument
layout is not supported. Trailing commas in call arguments are
forbidden: for a two-parameter function, write `call add(left, right)`, not
`call add(left, right,)`.

Every fixed parameter requires an argument. Missing fixed arguments and empty
argument slots are compile-time errors; there is no implicit undefined value
for an omitted argument. A variadic parameter collects the additional arguments.
Array spread in calls remains allowed; validation when its length is unknown
remains to be specified.

A standard runtime failure propagates up through callers until a `[_] => expression`
arm in `match call` or `match chain` catches it. This handler is optional in
both forms. Without it, standard failures propagate automatically through
callers until an enclosing handler catches them; no explicit forwarding arm
is required for these failures. `[_]` is the catch-all for standard failures
outside `emits`, not for domain errors. Declared domain errors retain their
explicit handling or forwarding requirements. The wildcard covers standard
runtime failures that can propagate to a Can handler; it does not guarantee
recovery from fatal process termination, such as fatal out-of-memory.
If no handler catches a standard failure, it is fatal. The full inventory of
standard failures remains undecided.
The standard-failure handler can inspect a string description of the caught
failure, including runtime exceptions originating in generated TypeScript or
JavaScript/Bun operations rather than declared Can domain errors. This value
is exposed as a string, not as a declared Can error type. Bind the description
with `[_] as str message => expression`, using `as <type> <name>`.
The unbound `[_] => expression` form remains available when the description is
not needed. The precise conversion of runtime thrown values to strings remains
undecided.

Direct successful-payload binding is a compile-time error when the callee's
declared `emits` set is nonempty, even if a particular invocation would succeed.
The declared errors must be handled or explicitly forwarded using the completion
handling forms below.

Use `match call` to handle a call's success and declared errors. `ok int number`
introduces an immutable arm-local payload binding. Its type must agree with the
callee's success type; it is not a runtime type test or cast.

```text
fn int lookup_or_zero
    emits []
    given
        str key
    asserts
        known_key: "answer" => ok 42
        missing_becomes_zero: "other" => ok 0
    match call lookup(key)
        ok int number => ok number
        missing() => ok 0
```

Here `lookup` has success type `int` and declared error set `[missing]`, returning
`ok 42` for `"answer"` and `missing()` for `"other"`. `missing` is a payload-free
declared error with its own mandatory numeric ID.

A bare `ok` arm forwards the matched success and its whole payload unchanged:

```text
match call lookup(key)
    ok
    missing() => ok 0
```

A bare named error arm forwards that exact error and its complete payload:

```text
match call lookup(key)
    ok
    missing
```

Every declared outcome requires an explicit arm. There is no omitted-success
forwarding or wildcard error forwarding. The enclosing function's success type
must be compatible, and its error contract must permit forwarded errors.
A standalone bare `ok` arm in completion matching forwards the matched success
and its whole payload. In a completion-expression position, bare `ok` instead
completes a void function successfully. Success payload patterns also omit
parentheses: `ok int number`.

Use terminal `relay call` to forward every outcome of a call:

```text
fn int find_number
    emits [missing]
    given
        str key
    asserts
        known_key: "answer" => ok 42
        unknown_key: "other" => missing()
    relay call lookup(key)
```

This invokes once and finishes with the exact completion and payload. The caller
must permit the entire declared error set and have a compatible success type.
Computation can precede the relay; there is no continuation after it.

## Sequential call chains

`match chain` sequences calls with success bindings and one set of explicit
completion arms covering the declared errors of all calls:

```text
match chain
    call find_user(user_id) as user found_user
    call load_account(found_user.account_id) as account found_account
    call check_balance(found_account) as account checked_account
    ok => ok checked_account
    user_not_found => access_denied()
    account_not_found => access_denied()
    insufficient_balance
```

Each binding uses `as <type> <name>`, with an explicit type before the name.
Calls returning `void` omit the `as` binding entirely; successful completion
continues to the next step.
Each successful call binds its returned value for subsequent steps. The first
error stops the chain; later steps do not run. All declared errors from the
calls must be covered explicitly, with no catch-all domain-error arm. An optional
`[_] => expression` arm catches standard runtime failures from any chain step;
without it those failures bubble up automatically. A bare named
error arm forwards that error unchanged; `error_name => expression` handles
that specific error using the existing error-arm rules. If every call succeeds,
the `ok => expression` arm runs. These arms replace `then` and shared `else`.
The example assumes the calls declare `user_not_found`, `account_not_found`
and `insufficient_balance`, respectively.
There is no `when` clause in a chain. Checks are calls that report failure
through declared errors, handled by the same explicit error arms.
Additional typing and composition rules remain undecided.

## Array patterns

Array patterns use brackets. `[]` matches an empty array, and
`[first, ...rest]` matches a nonempty array, binding its first element to `first`
and the remaining array to `rest`:

```text
match scores
    [] => ok 0
    [first, ...rest] => ok first
```

Multiple explicit positions may precede the remainder, for example
`[first, second, ...rest]`, which requires at least two elements and binds the
remaining array to `rest`.
An underscore `_` ignores one required element; empty positions are not used
in array patterns. For example, `[first, _, ...rest]` binds the first element,
ignores the second and collects the remaining elements. `first` and `rest` are
arm-local binding names, not reserved keywords. Literal positions and exact-length matching
without a remainder remain undecided.

This complete example demonstrates matching and ignoring the second element:

```text
/// Returns the values with the second element removed, if present.
fn int[] remove_second
    emits []
    given
        int[] values
    asserts
        four_values: [10, 20, 30, 40] => ok [10, 30, 40]
        two_values: [10, 20] => ok [10]
        one_value: [10] => ok [10]
        empty: [] => ok []
    match values
        [first, _, ...rest] => ok [first, ...rest]
        _ => ok values
```

The underscore requires a second element but does not bind or constrain its
value. The result expression constructs a new array; matching does not modify
the original. The standalone `_` arm is the existing whole-match fallback.

## Matching

Decisions: SURFACE-019–022, SURFACE-028, SURFACE-036–037, SURFACE-047, SURFACE-052,
SURFACE-070, SURFACE-072.

Can has no `if` expression. Boolean branching uses `match` with `true` and
`false` patterns.

Simple arms use `pattern => expression`, including completion expressions.
An ordinary-data `match` may produce an ordinary value, including as the
initializer of a typed local binding:

```text
int amount = match member
    true => 80
    false => 100
```

This permits value-producing matches without making intermediate bindings the
preferred style. It does not establish additional expression-placement rules
or general completion storage.

Multiple steps use `pattern => do`
followed by an indented body. `do` denotes multiple steps, not merely multiple
lines. Arms have no prefix keyword.

```text
match eligible
    true => do
        int result = left + right
        ok result
    false => ok 0
```

A nested match begins on its parent arm's line, immediately after `=>`. Its
arms are indented one level further. A nested match is one expression and does
not require `do`.

```text
fn int sign
    emits []
    given
        int number
    asserts
        negative: -7 => ok -1
        zero: 0 => ok 0
        positive: 7 => ok 1
    match number < 0
        true => ok -1
        false => match number is 0
            true => ok 0
            false => ok 1
```

Multiple scrutinees and their patterns are comma-separated without enclosing
parentheses:

```text
match is_admin, is_owner
    true, _ => ok true
    false, true => ok true
    false, false => ok false
```

Use `|` between alternative patterns sharing one arm body. The arm matches when
any alternative matches; its body executes once. Alternatives may contain
record constructors, literals and ignored fields, as shown by `has_zero_side`.
Binding consistency across alternatives remains to be specified.

Ordinary-data matching remains exhaustive; arms are ordered and the first match
wins. A wildcard may cover ordinary data. Completion matching has the stricter
explicit-outcome coverage described above.

Inclusive integer range patterns use two dots. `1..5` matches integers from
1 through 5, including both endpoints:

```text
match number
    1..5 => ok true
    _ => ok false
```

Other range forms and bound-expression rules remain undecided.

Match an error with its bare name. Within that arm, the error name refers to
the matched error and its payload fields are accessed by their declared names
using a dot. Positional error-payload binding patterns are not supported;
unused payload fields require no placeholder. This rule applies to error
matching, not ordinary record patterns or the ordinary-data wildcard.

```text
below_minimum => ok below_minimum.minimum
```

Here the declared field is `minimum`. If the error declares a field named
`floor`, access it as `below_minimum.floor`; field access does not rename fields.

## Errors

Decisions: SURFACE-032–037.

Declare an error with a mandatory stable numeric ID followed by its name, then
indented type-before-name payload fields:

```text
error 1002 below_minimum
    int actual
    int minimum
```

IDs identify error kinds, are unique throughout the codebase, and must be visible
in relevant error reports. Missing and duplicate IDs must be compile-time errors.
They are declaration metadata, not payload arguments or replacements for nominal
error identity. All occurrences of an error kind share its ID. Numbers in these
examples are illustrative, not allocated registry entries.

Construct errors with positional payload arguments in field declaration order,
both in business logic and assertion expectations.

```text
fn int require_minimum
    emits [below_minimum]
    given
        int value
        int minimum
    asserts
        accepted: 5, 3 => ok 5
        rejected: 2, 3 => below_minimum(2, 3)
    match value < minimum
        true => below_minimum(value, minimum)
        false => ok value

fn int clamp_minimum
    emits []
    given
        int value
        int minimum
    asserts
        unchanged: 5, 3 => ok 5
        raised: 2, 3 => ok 3
    match call require_minimum(value, minimum)
        ok
        below_minimum => ok below_minimum.minimum
```

## Records

Records may have no fields. A fieldless record has no indented body and uses
empty parentheses for construction:

```text
/// Indicates that processing has finished.
record finished
```

Construct its value as `finished()`. Fieldless records may serve as choice
alternatives carrying no additional data.

Decisions: SURFACE-038–039, SURFACE-041, SURFACE-043–044, SURFACE-050.

Declare nominal records with indented type-before-name fields. Construct them
positionally in field declaration order and access fields with a dot.
Records are passed as whole values through named, typed function inputs.

```text
record dimensions
    int width
    int height

fn int area
    emits []
    given
        dimensions dimensions
    asserts
        rectangle: dimensions(3, 4) => ok 12
    ok dimensions.width * dimensions.height
```

Record construction must stay on one source line, such as
`dimensions(width, height)`. Multiline constructor layout is not supported,
and trailing commas are forbidden. Record declaration layout is unchanged.

Record patterns identify fields positionally in declaration order. Every field
has a pattern; `_` ignores a field. Ordinary first-match semantics apply.
Choice matching uses bare type names as specified below.

```text
fn bool has_zero_side
    emits []
    given
        dimensions dimensions
    asserts
        zero_width: dimensions(0, 4) => ok true
        zero_height: dimensions(3, 0) => ok true
        rectangle: dimensions(3, 4) => ok false
    match dimensions
        dimensions(0, _) | dimensions(_, 0) => ok true
        dimensions(_, _) => ok false
```

Use `with` to create a value of the same record type, replacing named fields and
preserving the others. The original value stays unchanged. One replacement has
no replacement-list parentheses; two or more require them.

```text
fn dimensions widen
    emits []
    given
        dimensions dimensions
        int amount
    asserts
        wider: dimensions(3, 4), 2 => ok dimensions(5, 4)
    ok dimensions with width = dimensions.width + amount

fn dimensions enlarge
    emits []
    given
        dimensions dimensions
        int amount
    asserts
        larger: dimensions(3, 4), 2 => ok dimensions(5, 6)
    ok dimensions with (width = dimensions.width + amount, height = dimensions.height + amount)
```

## Choices

Decisions: SURFACE-080–081.

Use `choice` to declare a type whose value is one of the listed types. Records
are declared independently and then listed as alternatives:

```text
record circle
    int radius

record rectangle
    int width
    int height

choice shape
    circle
    rectangle
```

A `shape` value is either a `circle` or a `rectangle`, not a combination of their
fields. The records remain independently usable; their values are accepted where
`shape` is expected without an extra wrapper.

Match record alternatives using bare type names, without field parameters:

```text
match shape
    circle => ok shape
    rectangle => match shape.width < 6
        true => ok shape with height = shape.height - 4
        false => ok shape
```

Within a choice arm, the original matched binding remains the value and is
narrowed to the matched record type. Access its fields through that binding,
such as `shape.width`, and return or copy-update it as `shape`. The record type
name does not become a value binding. Only error matching exposes the matched
value through the error name. Other permitted alternative types, nesting and
overlap rules remain undecided.

## Arrays and array operations

Compiler simplicity takes priority over earlier array syntax preferences.
This is a criterion for targeted tradeoffs, not a goal of turning Can into
TypeScript. Preserve Can's intended language model; recommend specific changes
only with concrete costs and benefits, and obtain approval before replacing
established syntax choices.
Review both approved and unresolved array choices against straightforward
compilation to native TypeScript/JavaScript operations. Challenge prior choices
when they introduce unnecessary complexity; approval is not evidence that a
design is technically sound. Present concrete tradeoffs and recommended revisions
rather than silently changing unrelated decisions. Prefer native forms where
they serve the intended behavior; this is not blanket adoption of every
JavaScript behavior. Pattern matching, immutability and error handling must be
included in that review rather than assumed to have no implementation cost.

Array literals support spread notation, for example `[...scores, 40, ...bonus_scores]`,
creating a new array without changing the source arrays. The compiler should
lower array spread to native TypeScript/JavaScript array spread executed by Bun,
rather than implement the copying operation in Can library code.

Decisions: SURFACE-045–046, SURFACE-048–049.

Use `T[]` for an immutable, variable-length ordered collection of elements of
type `T`: for example, `int[]`, `str[]` or `dimensions[]`. Values use bracket
literals such as `[7, 7]`; the empty value is `[]`. Preserve existing element-type
checking and expected-type requirements.

Array literals must stay on one source line. Multiline array literals are not
supported.

```text
fn int[] repeat_twice
    emits []
    given
        int value
    asserts
        repeated: 7 => ok [7, 7]
    ok [value, value]
```

Arrays support methods and properties, as specified under Array and string
methods below. Arrays remain immutable values.

Array indexing uses brackets:

```text
scores[index]
```

This selects indexing syntax only. Index bounds, negative indices and failure
behavior remain undecided.

Array slicing uses `scores[start:end]`. Endpoint inclusion, omitted endpoints,
negative bounds and failure behavior remain undecided.

Array length uses the property `scores.length`.

Invoke `append` as a regular function, with the array as its first argument:
`call append(scores, 40)`. It returns a new array without mutating the original,
adds one element and has an empty declared domain-error set.

```text
fn int[] add_score
    emits []
    given
        int[] scores
        int score
    asserts
        appended: [10, 20], 30 => ok [10, 20, 30]
        first: [], 30 => ok [30]
    ok call append(scores, score)
```

Compose array operations through ordinary calls, with `call` on each invocation:

```text
call append(call append(scores, 40), 20)
```

This adds `40`, then `20`, without mutating `scores`.

## Array and string methods

Arrays and strings support receiver methods, properties and method chaining.
Methods use the existing `call` invocation marker; properties do not:

```text
call numbers.map(callable multiply)
call numbers.slice(start, end)
numbers.length
"string".length
```

The compiler must translate approved methods and properties to the equivalent
TypeScript/JavaScript built-in prototype methods or instance properties used
by Bun, rather than reimplementing an equivalent standard operation in Can.
This includes lowering `.map` and `.slice` through their corresponding native
methods and `.length` through the native property. Compiler-owned adapters may
bridge Can's types, callable/completion conventions and declared errors; native
lowering must preserve those contracts rather than bypass them.

Enforce array immutability by rejecting mutation and exposing native copying
operations. Do not require blanket deep copying or freezing of every array;
compiler-owned adapters must still prevent mutation through exposed APIs.

All exposed operations preserve immutability. No in-place mutation, mutable
element/property assignment or mutation of an input through an alias is allowed.
Array transformations return new arrays. Mutating native operations must not
be exposed with their in-place behavior; use an equivalent non-mutating native
operation where one exists. An API lacking an immutable equivalent requires
separate design, not silent admission of mutation.

Method chains use one `call` prefix, with each returned value becoming the next
receiver. For operations whose declared error sets are empty, the shape is:

```text
call numbers.map(callable multiply).slice(start, end)
```

Each operation executes once, in chain order. Chaining does not introduce
implicit error propagation; fallible operations still require the established
completion handling. The full method/property inventory, callback contracts,
type conversions and fallible-chain syntax remain unresolved. These examples
do not expose arbitrary backend imports, custom
prototype extensions or every JavaScript API.

Method chains must stay on one source line. Dot-led multiline continuation is
not supported. Chaining itself remains supported, as shown above.

The already selected bracket indexing/slicing syntax and ordinary `append`
function remain available. Whether `append` also has a method spelling is not
settled by restoring native-style methods.

## Effects — deferred

Revisit whether Can needs effect tracking and, if so, its mechanism and syntax
later. No requirement to distinguish pure and externally interacting functions
is currently established. This deferral does not change `emits` or the
separately recorded Can-to-Bun platform boundary.
Decisions about `read`, `write`, `modify` and `delete`, including their keyword
status or effect meaning, are also deferred.

## Unresolved questions

- Whitespace rules beyond four-space indentation and the rejection of indentation tabs.
- General expression continuation outside assertions.
- Early-return rules.
- Integration and naming of dependency expectations alongside input `given`.
- Effects: deferred, including whether effect tracking is needed at all.
- Callable domain-error contract spelling and generic error propagation;
  the basic callable type and generic error declaration spelling are settled.
- Binding consistency across `|` alternatives; applicability of alternatives to
  completion dispatch.
- Typed binding declarations within ordinary record patterns.
- Error-ID allocation and historical stability enforcement, retired-ID reuse,
  dependency collisions, numeric ranges and relation to compiler diagnostic IDs.
- Detailed copy-update evaluation and validation rules.
- Array operation inventory and contracts beyond the selected `append`.

## Can-to-Bun boundary

Current boundary: SURFACE-066. Catalogue ownership and command-execution
constraints: SURFACE-063–064.

Can programs contain only Can executable code. Embedded Bun, JavaScript and
TypeScript code is forbidden. There is no application-facing `extern` facility,
raw backend-code block, executable source-string escape, arbitrary backend import
or unrestricted access to Bun APIs from Can source. Ordinary strings remain data;
they cannot be used to inject backend code.

Platform operations are exposed through a special Can standard library. The
compiler translates calls to these Can library operations into the corresponding
Bun standard-library/runtime APIs in generated TypeScript. Can authors use the
Can library, not embedded backend code.

Only approved Can standard-library operations can access the platform. Their Can contracts and
Bun translations are maintained in the Go compiler. Application authors do not
write adapter files or platform implementation code. The execution pipeline is:

```text
Can source -> Go compiler -> generated TypeScript using Bun APIs -> Bun execution
```

### Reuse native operations; do not reimplement them

When an approved Can operation has an equivalent JavaScript language operation,
built-in method/property or Bun API, the compiler must emit that native operation
in generated TypeScript. Do not recreate the existing implementation in Can
standard-library code or in generated helper code. This rule applies generally,
not only to array spread or array/string methods. For example, concatenation
must use an equivalent native operation rather than a recursive Can loop that
rebuilds the result element by element.

Compiler-owned adaptation may enforce Can's types, immutability and completion/
error contracts, but must delegate the underlying operation to the native
implementation. A native operation must not mutate a Can input. If no native
operation meets the approved Can contract, that gap requires an explicit design
decision rather than silently substituting different behavior or rebuilding an
upstream implementation. This rule does not expose arbitrary backend APIs to
Can programs.

Go generates the platform calls; it is not an intermediate runtime executing
those calls. The compiler owns result/error conversion and any required emitted helpers.
It does not import project-authored TypeScript adapters. Generated TypeScript is
compiler output, not executable source supplied by the Can program author.

Only the maintained Can distribution can extend the approved operation catalogue.
Projects cannot register their own backend implementations, and separately
supplied providers are not an extension route. The catalogue must not expose
general-purpose execution of arbitrary programs, commands, scripts or interpreters.

String operations such as lowercasing belong to Can's language or standard library,
not project-defined backend escapes. Platform-specific details remain behind
Can operations; no unrestricted Bun namespace is exposed to application code.

The approved inventory, exact Can APIs, effect granularity (individual operations
versus broader categories), asynchronous execution rules, error mapping and test
substitution remain unresolved. Illustrative file-operation names are not yet
approved library APIs.

## Required removal and implementation authorization

Decision: SURFACE-067.

All old implementation paths that enable user-defined externs must be deleted
when implementation is authorized. This includes extern declaration support,
external implementation resolution and adapter imports, and the associated
legacy adapters and supporting artifacts. Do not retain an alternate or
compatibility path that permits arbitrary external code. Approved platform
operations belong in the compiler-owned Can-to-Bun mappings described above.

This is a recorded future implementation requirement, not permission to modify
implementation code. The user explicitly withheld that authorization. Limit
current work to design/documentation; do not delete, replace or otherwise modify
implementation code until the user authorizes it. No removal is claimed complete.
