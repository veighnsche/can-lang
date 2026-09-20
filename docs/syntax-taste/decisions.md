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

The standard-library capabilities described in `../ASTRA_STDLIB.md` belong in
Can's language-design scope, including capabilities not yet implemented. Review
and carry those requirements into the current design rather than discarding
them because the document uses older language assumptions. Its proposed syntax,
contracts and implementation strategies are not automatically approved: reconcile
them with these decisions and identify remaining design and implementation gaps.
This inclusion does not authorize implementation work.

Native AI judgments are foundational to Can's purpose and name, not an optional
library integration added after the language is designed. Probabilistic judgment
and branching are central design concerns, including how they relate to pattern
matching. Design effects, asynchronous execution, concurrency and assertion
support around concrete native-judgment use cases. This establishes direction,
not a particular judgment spelling or a change to ordinary pattern-match semantics.

Jev-backed AI primitives must have their own native Can grammar and syntax;
ordinary standard-library function wrappers alone do not satisfy this requirement.
Design their concrete forms as part of the language, reconciling the older
native-AI proposals with current decisions. The exact primitive inventory and
spellings still require design; this requirement does not adopt the old proposals
wholesale.

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

Callable domain-error annotation placement is provisionally selected below;
compatibility and inference rules remain undecided. Callables require no purity
or effect classification.

## Native Noul declaration direction

Noul, Choice and Score use shared project configuration for provider settings.
Question declarations describe state, instructions, criteria and handlers, not
HTTP request construction. The compiler/runtime owns authentication, transport
and response decoding; authors do not repeat fetch setup in judgment handlers.
Use a dedicated named declaration with indented settings (the selected first
option), rather than a configuration-record binding or factory function.
Use `connection` as the declaration keyword. The indented setting notation
below is selected; detailed transport behavior remains technical design work.

Wrappers are generic shared transport configuration, usable by ordinary HTTP
fetch operations and native AI judgments, not TypeSafe-only declarations.
Credentials are shared transport concerns; model selection is AI-specific
metadata, not a mandatory field of every wrapper. Support configurable remote
and local service endpoints rather than hard-coding the hosted TypeSafe service.
Native judgments still require a compatible request/response protocol or an
explicitly supported adapter; choosing a different endpoint alone does not
establish protocol compatibility. Detailed metadata typing and adapter
configuration remain to be designed.

### Shared connection configuration

Use indented named settings, including `endpoint`, `auth bearer env`,
`timeout_ms`, and a nested `metadata` section:

```text
connection default_wrapper
    endpoint "http://localhost:8080"
    auth bearer env "LOCAL_API_KEY"
    timeout_ms 30000
    metadata
        model "local-model"
```

`endpoint` selects the destination, including a compatible local service.
`auth bearer env` reads the named environment variable at runtime and supplies
Bearer authentication; it does not embed a secret in source. A connection
without authentication omits `auth`. Additional authentication schemes and
headers belong to transport configuration, not individual question handlers.
`metadata` carries operation-specific settings: the native AI adapter places
`model` in the request body. Generic fetch calls do not automatically acquire
an AI model field, and metadata is not implicitly converted to HTTP headers.
Native judgments still own their question/state encoding and response decoding.
This configuration does not allow embedded backend code or user-written externs.

`timeout_ms` expresses a timeout in milliseconds; 30000 is an example, not a
global default. Further technical design must specify timeout boundaries,
headers, configuration validation,
metadata typing and application, and supported authentication schemes. No retry
policy, configuration evaluation mechanism, or arbitrary metadata forwarding is
silently established by this example.

Use the primitive-first declaration head `noul <return_type> <name>`, rather than a `question`
prefix or an inner `kind` marker. Select its provider wrapper with
`noul <return_type> <name> from <wrapper_name>`, for example
`noul str needs_human from default_wrapper`. The explicit return type is the
type produced by the selected handler, not the probability type. The wrapper keeps provider/transport
setup separate from the question. The dedicated configuration declaration is
selected above; its final detailed grammar, relationship to project configuration,
and whether `from` may be omitted remain unresolved.
This does not introduce arbitrary embedded backend code or extern adapters.
Question declarations use `given` for ordinary parameters that may supply
question text or criterion descriptions. Shared evaluated content is declared
by the enclosing `judge` in `state`, as specified below. This replaces the
earlier interpretation of question-level `given` as state. The design must support descriptions of what counts as true and
false, and multiple named Noul questions in a single request over shared state.
Use `asks` for the question text. The `true` and `false` branches each carry
their criterion description followed by `=>` and an executable handler:

```text
noul bool needs_human from default_wrapper
    given
        str what_question
        str what_is_true
        str what_is_false
    asks what_question
    minimum 0.6
        true what_is_true => ok true
        false what_is_false => ok false
```

`minimum` is an optional cutoff on the Noul probability of yes. At or above
the cutoff selects the `true` handler; below it selects the `false` handler.
If `minimum` is omitted, the cutoff is 0.5. There is no third uncertainty branch
in this form; the cutoff is not a separate confidence score or a symmetric
certainty threshold. The descriptions are model criteria; the handlers execute
the selected Can branch.

Inside a Noul branch handler, standalone `%` exposes the returned probability
of true. It refers to the same underlying probability in both the `true` and
`false` handlers; the false handler does not substitute its complement.
This is a contextual expression, not a replacement for the existing binary
remainder operator. Its numeric scale is 0–1: a twenty-percent probability is
`0.2`, not `20`. It exposes the original probability without automatic scaling
or rounding. This supersedes the intervening 0–100 selection for `%` only;
the `minimum` cutoff remains unchanged on the same probability scale.
String formatting/conversion remains unresolved. The user's concatenation
sketch does not establish implicit numeric-to-string conversion.

The example records the shown explicit-cutoff layout. Exact branch indentation
when `minimum` is omitted and detailed typing/assertion rules remain to be
designed. Batched invocation is specified under `judge` below. Whether criterion descriptions are mandatory
also remains unresolved.

## Native Choice and full-distribution forms

`choice <return_type> <name> from <connection_name>` uses the primitive-first
AI declaration layout, with an explicit return type before the name. That type
describes the selected handler's result. Named criteria and handlers appear
beneath `asks`. Only the winning option's handler executes:

```text
choice str route_ticket from default_wrapper
    asks "Which team should handle this message?"
        billing "Payments, invoices, or refunds." => ok "billing"
        technical "Bugs or problems using the product." => ok "technical"
        other "Anything outside those categories." => ok "other"
```

Use `record <record_name> choice <field_type> <name> from <connection_name>` for the separate
native form processing all option probabilities. This replaces both the
initial standalone `distribution` and subsequent `choice distribution` spellings.
This form returns a compiler-generated record whose fields are the declared
option names. Every option's handler runs, with standalone `%` exposing that
option's original 0–1 probability. Its `ok` payload supplies that field's value;
the explicit field type applies to every generated field. This supersedes the
earlier handler-free, raw-probability-only form:

```text
record routing_result choice float routing_weights from default_wrapper
    given
        str question
    confidence as conf
    asks question
        billing "Payments, invoices, or refunds." => ok 100 * %
        technical "Bugs or problems using the product." => ok %
        other "Anything outside those categories." => ok 400 * %
```

The numeric example uses `float`, not `int`; no rounding or truncation rule is
introduced. A `record <record_name> choice str ...` declaration likewise produces string
fields. The user's string-field example requests automatic conversion of a
numeric handler payload to `str` in that context. Exact formatting and whether
that conversion applies in other language contexts remain unresolved; this is
not blanket authorization for implicit conversions throughout Can.

The explicit record name identifies the generated type. `confidence as conf`
binds confidence for handlers and exposes it as the generated record field
`conf`; an option named `conf` then conflicts and must be rejected. That metadata
field carries the provider confidence rather than an option handler result.
Handler failure semantics and inclusion of the selected option remain undecided.
This does not mean every
handler in a `choice` declaration executes. The distinction is a Can surface
design: TypeSafe Choice already returns the full probability distribution as
well as its selected option and confidence. It does not require a new provider
primitive. In ordinary Choice handlers, `%` exposes the selected option
probability on the original 0–1 scale. Use `confidence as <name>` to bind the
separate returned confidence value. An optional `minimum <threshold> => <fallback>`
selects the fallback below the minimum confidence; otherwise the winning option
handler runs. This differs from Noul minimum, which thresholds probability of
true. Confidence scale/default/formatting details remain unresolved; the source
examples do not establish new string conversion or percentage-scaling rules.

## Batched judgments with `judge`

Use `judge <return_type> <name> from <connection_name>`. Its `given` section
contains ordinary inputs. Its `state` section declares the shared content for
all questions. Question text and criteria may be supplied through ordinary
question parameters rather than being restricted to literals.

Each listed `call question(arguments) as <type> <name>` registers a question
for one shared AI request. It is not a sequential network request. All answers
are received and their question handlers evaluated and results bound before
executing the judge's final `ok => ...` arm. This batching behavior is explicitly
approved and differs from ordinary sequential `call` execution outside a judge.

```text
judge str assess_urgency from default_wrapper
    given
        str question
    state
        str email_content
        int num_emails_from_author
        int num_emails_from_system
        timestamp received_time
    call needs_human(question, "True when human assistance is needed.", "False otherwise.") as bool human_needed
    call route_ticket() as str department
    call routing_weights(question) as routing_result weights
    ok => ok department
```

This adapted excerpt supplies the three required arguments to `needs_human`;
the user's omitted arguments were not approval for missing fixed inputs.
The shared state is available to the AI questions, separate from these arguments.
Invoke a judge with ordinary `given` arguments first, followed by one final
parenthesized group containing its `state` arguments, both in declaration order:
`call assess_email(question, (email_content, email_count))`. The inner group is
judge-specific argument syntax, not a general anonymous tuple value.
Serialization (including the
illustrative `timestamp` type), request failures, assertions, handler execution
order and compatibility of question/judge connections remain unresolved.
The example does not approve a general timestamp type or new conversions.

### Choice confidence and reusable arms

The selected confidence/fallback surface is illustrated by:

```text
choice str route_ticket from default_wrapper
    asks "Which team should handle this message?"
    confidence as conf
    minimum 0.6 => ok "Review needed."
        billing "Payments, invoices, or refunds." => ok "billing"
        technical "Bugs or problems using the product." => ok "technical"
        other "Anything outside those categories." => ok "other"
```

Reusable choice arms are named top-level declarations. Use
`choice_arm <return_type> <name>`, a `describes` line for its criterion text,
and an executable completion body. `%` is the current option probability.
Store those named arm values in ordinary record fields and spread the record
beneath `asks`:

```text
choice_arm float billing_arm
    describes "Payments and refunds."
    ok %

choice_arm float technical_arm
    describes "Product faults."
    ok %

/// Reusable department judgment arms.
record departments
    choice_arm billing
    choice_arm technical

departments all_depts = departments(billing_arm, technical_arm)

record routing_result2 choice float routing_weights2 from default_wrapper
    confidence as conf
    asks "Which team should handle this message?"
        ...all_depts
        other "Anything outside those categories." => ok %
```

The record field names supply the option names. Exact captures and detailed
handler/field compatibility remain unresolved. This does not add general
anonymous functions or inline executable arm-constructor expressions.

### Runtime-defined Choice options

Spread description-only option data beneath `asks`, then use one shared success
arm receiving the selected key:

```text
choice str select_department from default_wrapper
    given
        str question
        choice_option[] candidates
    asks question
        ...candidates
        ok str selected_key => ok selected_key
```

`choice_option` describes a stable key and criterion text; its exact record
contract and validation remain technical work. These candidates contain data,
not executable handlers. Their spread is distinguished by element type from
spreading reusable `choice_arm` values. Runtime-generated names do not create
statically named result fields. No `dynamic` modifier or separate `options`
section is introduced.

## Native Score declarations

The revised Score forms are approved, superseding the earlier deferral. Put
settings before `asks`; `asks` is the final section, containing ordered levels
and handlers. This placement decision applies to Score forms, not a silent
rewrite of other declaration layouts.

```text
score float assess_severity from default_wrapper
    confidence as conf
    score as value
    minimum 0.6 => ok -1.0
    asks "How severe is the problem described in the email?"
        minor "An inconvenience; normal work can continue."
        moderate "Work is disrupted, but a workaround exists."
        severe "Essential work is blocked with no workaround."
        ok => ok value

record severity_weights score float assess_severity_weights from default_wrapper
    confidence as conf
    score as value
    asks "How severe is the problem described in the email?"
        minor "An inconvenience; normal work can continue." => ok %
        moderate "Work is disrupted, but a workaround exists." => ok %
        severe "Essential work is blocked with no workaround." => ok %
```

Levels are numbered from zero in written order. `score as <name>` binds the
returned weighted numeric score, which may be fractional; `confidence as <name>`
binds the separate confidence. Ordinary Score runs the minimum-confidence
fallback below its explicit threshold, otherwise the final `ok` handler. It
never implicitly rounds the score or selects the highest-probability level.
The example's `-1.0` is an author-chosen fallback, not a built-in failure value.

`record <record_name> score <field_type> <name> from <connection_name>` runs
every level handler and generates a field for each named level. In those
handlers `%` is that level's probability on the 0–1 scale. The confidence and
score bindings also name generated metadata fields carrying their respective
provider values, separate from transformed level fields. Name collisions are
errors. These metadata fields are not forced to the level-handler field type.

Both forms receive shared state through `judge` and may have ordinary `given`
parameters. Default confidence policy, assertions, provider failures and other
unresolved judge contracts are not supplied implicitly by these examples.

## Named fetch declarations

Use a named fetch declaration with the expected response record type before
the name and `from` selecting shared connection configuration:

```text
fetch user_profile load_profile from account_service
    given
        str user_id
    get "/profile"
    query
        id = user_id
    headers
        accept = "application/json"
```

Invoke it using ordinary call syntax and completion handling:

```text
match call load_profile("42")
    ok user_profile profile => ok profile
    // Error arms depend on the fetch contract, which remains to be designed.
```

`user_profile` is a separately declared record describing the expected decoded
response body, validated before exposure as that type. The connection supplies
shared transport settings. This selects the named declaration option, not the
ordinary-library-call alternative or its proposed call-site `from` suffix.
Use ordinary `given` parameters and named `query` and `headers` sections, with
`name = expression` entries. Query encoding, header mapping/merging, other HTTP
methods, request bodies, response status/header access, text/bytes responses and
error contracts remain to be designed. No request-record or helper-expression
alternative is selected.

## Native LLM responses and tools

Can must support native LLM-generated responses in addition to the judgment
primitives. Responses must support both plain text and structured data, such as
JSON. The exact structured-output type/schema syntax and validation/error rules
remain to be designed; this does not introduce a general untyped JSON value.

For structured responses, use `llm <record_type> <name> from <connection_name>`.
The output shape is an ordinary separately declared record. There is no
`returns` section; the record type is in the declaration header. This selects
the first proposed form, superseding the briefly discussed bottom `returns`
layout, without adding nested record declarations.

```text
/// A generated summary of an email.
record email_summary
    str subject
    str summary

llm email_summary summarize_email from default_wrapper
    state
        str email_content
    asks "Summarize the email with a short subject and a factual summary."
```

Validate generated structured data against the declared record shape before
exposing it as that type. Invalid-output handling remains to be designed.
The previously requested plain-text capability is not removed by this
structured-response syntax selection.

The author defines the shape of a structured LLM response. This is a general
capability, not a dedicated generator for Choice questions or options. Ordinary
Can code can transform the typed response and pass selected values into a
Choice's `given` parameters. The response need not itself match a built-in
Choice schema. Generating question text, option sets and descriptions is one
motivating application, not a restriction on what an LLM can return.

In that application, options may be determined at runtime rather than limited
to source-declared lists. The author explicitly connects generation, any data
transformation, and the subsequent judgment; no automatic LLM-to-Choice pipeline
is implied. A judgment depending on generated inputs follows generation rather
than running as an independent question in the same request.

Validation of generated question/options and detailed dynamic-result typing
remain unresolved. Description-only option spread and a shared selected-key
handler are selected under runtime-defined Choice options. Runtime-generated
labels do not automatically become statically known record fields, nor does
this requirement authorize executing generated descriptions as Can code or
generating arbitrary executable option handlers.

LLM tool syntax and execution design are explicitly deferred. The requirement
to support author-supplied tools remains; none of the chooser tool-list options
is selected. Tool exposure, execution, result-return rules and termination/error
handling remain to be designed.
This requirement does not grant a model unrestricted access to Can functions
or establish automatic execution of every requested tool call.

## Naming

All user-defined names use lowercase `snake_case`, including functions, inputs,
locals, fields, assertions, records, variants and errors. Types and errors do not
use a different capitalization convention.

```text
fn int calculate_total
record order_item
variant payment_method
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
Generic record, function, variant and error declarations place type parameters immediately
after the declared name, also in angle brackets:

```text
record box<item>
    item value
```

The corresponding function-header shape is `fn item identity<item>`.
Variant and error declaration-head shapes include `variant outcome<item>` and
`error 1004 rejected<item>(item value)`. Error fields may use those type parameters.
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

The example above illustrates the base callable-type shape only; omission of
an error annotation does not establish an empty contract.

Provisional choice, explicitly open to revision after practical use: place a
callable's `emits [...]` annotation after its input types and before the binding
name:

```text
given
    callable receipt () emits [postgres_failed, redis_failed] operation
```

Here `receipt` is the success return type, and `operation` is the callable input
name. This selects annotation placement only. Whether empty lists must be
written, inference from referenced function declarations, callable compatibility
and generic error propagation remain unresolved. No purity or effect annotation
is introduced.

Invoke a function-valued input or other callable value with the same `call`
syntax used for named functions:

```text
call transform(5)
```

`callable` creates a function reference; `call` invokes a function. This invocation
syntax does not settle callable error compatibility or introduce implicit
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

### Dependency outcomes with `when`

Use a `when` table beneath a `match call` and before its completion arms to
supply dependency outcomes during named assertions. This replaces the old
call-site `given` table; `given` remains the function-input declaration section.

Each row uses assertion-style syntax:
`assertion_name: expected_call_arguments => supplied_completion`.
Arguments are positional and comma-separated; zero-input calls leave the space
between the colon and arrow empty. The row name identifies the assertion for
which that dependency outcome is supplied. In that assertion, the table supplies
the outcome instead of executing the dependency. Normal execution invokes the
real dependency.

The following excerpt assumes `clock::wall_now()` returns `int` and declares
`clock::unavailable`:

```text
/// Reads the current time in milliseconds.
fn int read_millis
    emits [clock::unavailable]
    asserts
        available: => ok 1726920000000
        unavailable: => clock::unavailable()
    match call clock::wall_now()
        when
            available: => ok 1726920000000
            unavailable: => clock::unavailable()
        ok int millis => ok millis
        clock::unavailable
```

This approves the keyword, call-site placement and row notation. It does not
restore the old boolean `when` chain guards or a `given / when / then`
business-logic grammar. Repeated-call sequences, missing-row rules, transitive
assertion context and integration with coordination blocks still need design;
the old implementation's policies are not inherited automatically.

### Assertions for AI and fetch consumers

Reuse ordinary `asserts` and approved call-site `when` tables when testing
consumers of AI and fetch declarations. Do not introduce a separate top-level
assertion declaration or declaration-level `when` fixture section through this
selection. The excerpt below assumes `user_profile` has one string field:

```text
// Inside a calling function:
asserts
    example: "42" => ok user_profile("Sam")
match call load_profile("42")
    when
        example: "42" => ok user_profile("Sam")
    ok user_profile profile => ok profile
```

Provider-output fixtures for testing AI declarations themselves remain a separate
design task; a supplied dependency result is not evidence of live model quality.

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
There is no boolean `when` guard in a chain. Checks are calls that report failure
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
parenthesized type-before-name payload fields:

```text
error 1002 below_minimum(int actual, int minimum)
```

Error payload fields are declared in parentheses after the error name (and any
generic parameters), using comma-separated `<type> <name>` entries on the
declaration line. This replaces the indented field-list form for errors only;
record declarations are unchanged. Trailing commas remain forbidden.

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

Construct its value as `finished()`. Fieldless records may serve as variant
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
Variant matching uses bare type names as specified below.

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

## Variants

Decisions: SURFACE-080–081.

Use `variant` to declare a type whose value is one of the listed types. This
replaces the data-type keyword `choice`; it does not change matching semantics.
The name `choice` is freed for native AI judgments, whose exact declaration
syntax is still being designed. Records
are declared independently and then listed as alternatives:

```text
record circle
    int radius

record rectangle
    int width
    int height

variant shape
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

Within a variant arm, the original matched binding remains the value and is
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

## Effects and asynchronous execution

Can authors do not annotate functions as asynchronous or synchronous. The
compiler owns that distinction. Ordinary Can code requires no promise handling
or authored `async` or `await`. Code is written as if calls
are synchronous: an ordinary call obtains its completion before dependent or
subsequent steps proceed, while native asynchronous waiting need not block the
runtime's other work. Calls are not implicitly launched concurrently.
All Can functions use asynchronous execution underneath: generated functions
use native async functions and ordinary calls await their completions.
Ordinary calls execute sequentially; asynchronous waiting does not mean
starting subsequent calls before the current call completes.

Can must also make the native Promise method capabilities available through
its Bun-backed standard library, including coordinating concurrent operations
as with `Promise.all`. This supersedes the blanket rejection of concurrency
and promise-handling APIs. Reuse native runtime implementations rather than
reimplementing them. The exact Can spelling, method inventory and mapping to
Can completion/error contracts remain to be designed. In particular, the
coordination form must allow operations to start before awaiting each result;
ordinary sequential calls alone cannot express that overlap. No concrete
promise type, task type or start syntax is selected by this decision.

Use the Can function name `concurrent` for native `Promise.all`. It coordinates
multiple calls, starting them without awaiting each one sequentially and then
awaiting the native aggregate operation. The compiler must preserve this
behavior despite the ordinary automatic-await rule. Result representation
remains undecided.

Use `race` in Can for first-success coordination, backed by native `Promise.any`.
This replaces the earlier mapping to `Promise.race`. The compiler starts the
participating calls without awaiting each one sequentially, then waits for the
first successful completion. A failed call does not end the race while another
participating call can still succeed. If every call fails, the race fails.
All-failed handling uses the `errors` group specified below. The underlying
aggregate representation and adapter between Can completions and native promise
settlement remain undecided.
This selects the `race` spelling and first-success behavior, not a general
promise type exposed to Can authors or automatic cancellation of remaining calls.

All four coordination headers require the `match call` prefix: `match call concurrent`,
`match call concurrent with error`, `match call race`, and `match call race with error`.
Bare coordination headers are not the selected syntax.

Both coordination forms use an indented block with one participating call per
line. These direct entries omit the `call` keyword:

```text
match call concurrent
    classify(review_a)
    classify(review_b)

match call race
    classify(review_a)
    classify(review_b)
```

This omission applies only to calls listed directly in these blocks; it does
not remove `call` from ordinary invocation syntax. Result binding and detailed
completion semantics remain to be designed; arm placement is specified below.
Operation entries are function invocations, not declarations or arbitrary
statements. Completion arms are also allowed as specified below.
In particular, record declarations cannot appear as entries. The
coordination construct supplies the invocation context, making `call` redundant
on each entry.

The modifier `with error` follows the coordination name on the block header:

```text
match call concurrent with error
    classify(review_a)
    classify(review_b)

match call race with error
    classify(review_a)
    classify(review_b)
```

`concurrent with error` maps to native `Promise.allSettled`: wait for every
participating call and retain each success or failure. `race with error` maps
to native `Promise.race`: take the first completion, whether success or failure.
Unmodified `concurrent` still maps to `Promise.all`, and unmodified `race` still
maps to `Promise.any`. The modifier selects coordination behavior; it does not
select result-binding syntax, change `emits`, or settle the representation of
collected domain errors and standard failures.

### Coordination completion-arm layout

The following sketches use `...` for handler bodies, not executable Can syntax.

```text
match call concurrent
    primary::lookup(user_id)
        ok profile primary_profile => ...
    backup::lookup(user_id)
        ok profile backup_profile => ...
    primary_unavailable => ...
    backup_unavailable => ...

match call concurrent with error
    primary::lookup(user_id)
        ok profile primary_profile => ...
        primary_unavailable => ...
    backup::lookup(user_id)
        ok profile backup_profile => ...
        backup_unavailable => ...

match call race
    primary::lookup(user_id)
    backup::lookup(user_id)
    ok profile found_profile => ...
    errors
        primary_unavailable => ...
        backup_unavailable => ...

match call race with error
    primary::lookup(user_id)
    backup::lookup(user_id)
    ok profile found_profile => ...
    primary_unavailable => ...
    backup_unavailable => ...
```

- `match call concurrent`: success arms are beneath their calls; shared domain-error
  arms follow at the call-entry indentation. Start all calls. If all succeed,
  process success arms in written order after all results arrive. Otherwise,
  dispatch the first failure to the shared arms and do not run success arms.
- `match call concurrent with error`: success and domain-error arms are beneath
  each call. Wait for every outcome, then process each call's matching arm in
  written order. A participant's failure does not discard another's success.
- `match call race`: the first success selects the shared success arm. Only when
  every participant fails, enter `errors` and dispatch each collected failure
  to its matching arm in input order. The same error arm can run more than once
  when multiple participants produce that error kind. This replaces individual
  shared domain-error arms outside an `errors` group for this form only.
- `match call race with error`: the first completion, success or failure, selects
  exactly one shared arm.

Calls run concurrently; handlers process the outcome of native coordination.
Unfinished calls are not automatically cancelled. Handler bodies are scoped to
the coordination operation: multiple success arms cannot each return from the
enclosing function. Successful per-call handler values are collected as described
below. Propagation when a handler itself fails remains unresolved.

Standard failures remain distinct from declared domain errors and use the
existing optional `[_]` handler where applicable. Propagation of unhandled
standard failures within collected outcomes remains unresolved. Exact aggregate
representation and handling for expanded callable collections
still need design.

These layouts retain lowercase `ok` and bare error-name patterns. Capitalized
success markers and constructor-shaped error patterns in earlier sketches did
not revise those established rules.

### Binding collected handler results

Bind a coordination result using the ordinary typed binding before the block,
not an `as` suffix or a separate binding line inside the block:

```text
save_result[] results = match call concurrent with error
    postgres::save(account)
        ok receipt saved => ok save_result("postgres", true)
        postgres_failed => ok save_result("postgres", false)
    redis::save(account)
        ok receipt saved => ok save_result("redis", true)
        redis_failed => ok save_result("redis", false)
```

This excerpt assumes a declared `save_result` record containing a database name
and success flag, and the illustrated function/error contracts. Each selected
per-call handler supplies one array entry through its `ok` payload, in input
call order. These completions do not finish the enclosing function. Handler
outputs must fit the declared array element type. This collects transformed
handler results, not automatically preserved raw completions; handlers can
explicitly preserve error information in their output values.

The typed-binding placement is selected. Handler-failure propagation, shared
failure-arm results for plain `concurrent`, and the result of an all-failed
`race` remain unresolved; this example does not settle those cases.

### Runtime-sized collections of different calls

Current selected direction, explicitly open to future revision: reuse existing
`callable <name>` values and `near` captures to collect different operations
without executing them. Named top-level wrapper functions may capture each
operation's distinct immutable inputs and invoke its underlying database or
other function when called. Creating the callable captures the inputs; it does
not start the operation. This introduces neither anonymous functions nor new
argument-binding or deferred-call syntax.

A collection may contain different compatible callable implementations, for
example `[callable save_postgres, callable save_redis]`, and may be constructed
with a runtime-dependent number of entries. Expand the collection inside a
coordination block with `...operations`:

```text
match call concurrent
    ...operations

match call race
    ...operations
```

The same expansion is available in the `with error` forms. Each callable is
invoked using its own captured inputs, without awaiting entries sequentially
before coordination. This extends the call-only block rule to permit a spread
entry supplying calls from a collection; it does not allow arbitrary statements
or declarations in the block.

For `concurrent with error`, put per-operation handlers directly beneath the
spread entry. The same arms apply separately to each expanded call, and their
successful payloads form the bound array in collection order:

```text
bool[] results = match call concurrent with error
    ...operations
        ok receipt saved => ok true
        postgres_failed => ok false
        redis_failed => ok false
```

This excerpt assumes compatible callable inputs returning `receipt` with the
illustrated declared domain errors. No `each` marker or coordination-specific
`for` syntax is introduced. This placement does not move the shared outcome
arms of the race forms beneath spread entries.

The illustrated operations share a success type, such as a named `receipt`
record. Exact collection typing, compatibility of callable domain-error sets,
and aggregate error handling remain unresolved. This decision
does not approve arbitrary unrelated result types in one collection or choose
new callable error-contract syntax. Extra named wrappers are an accepted
tradeoff of this current direction.

Effects are allowed in every function. There are no purity annotations, purity
checks, effect lists or effect-propagation checks. The compiler conservatively
treats every function as potentially effectful; this does not mean every function
actually interacts with external state. This decision does not change `emits`
or the separately recorded Can-to-Bun platform boundary.
Decisions about `read`, `write`, `modify` and `delete`, including their keyword
status or effect meaning, are also deferred.

## Iteration and early completion

Initial iteration uses named callable collection operations, with no new loop
syntax. `call items.for_each(callable save)` processes items sequentially.
This is a Can library contract; do not blindly lower an async callback through
native JavaScript `forEach`, which does not await it. Existing transformation
syntax such as `call items.map(callable transform)` remains available. Detailed
fallible iteration and callback contracts remain technical design work.

Keep terminal completion only; add no `return` or `finish` early-exit keyword.
Structure the remainder using existing `match` and `do` forms:

```text
match ready
    true => ok value
    false => do
        call prepare()
        ok fallback
```

This selects no additional early-return syntax and does not change the already
established completion rules of existing match arms.

## Unresolved questions

- Whitespace rules beyond four-space indentation and the rejection of indentation tabs.
- General expression continuation outside assertions.
- Detailed dependency-table behavior and integration beyond the approved
  call-site `when` syntax.
- Callable domain-error compatibility, inference and generic error propagation;
  annotation placement is provisional, while the basic callable type and generic
  error declaration spelling are settled.
- Binding consistency across `|` alternatives; applicability of alternatives to
  completion dispatch.
- Typed binding declarations within ordinary record patterns.
- Error-ID allocation and historical stability enforcement, retired-ID reuse,
  dependency collisions, numeric ranges and relation to compiler diagnostic IDs.
- Detailed copy-update evaluation and validation rules.
- Array operation inventory and contracts beyond the selected `append`.

## Can-to-Bun boundary

### Initial distribution scope

Windows is not a supported platform. Start with macOS support. Linux is a possible additional target, not yet a
committed initial platform. The initial macOS architecture coverage remains
to be selected. Runtime packaging should be platform-specific rather than
shipping binaries for every operating system in each download.

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

The approved inventory, exact Can APIs, error mapping and test
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
