# Can technical specification

20 September 2026. Documentation specification; no implementation authorization.

<a id="c1"></a>
## C1. Authority, scope and reading map

[Current decisions](decisions.md) records the approved source forms and product boundaries. This document and its three incorporated companions complete their technical contracts. The seven user answers are constraints, not recommendations to revisit. Technical choices below select defaults where the decisions previously said “undecided”; they do not claim those choices were individually selected by the user. A conflict with a recorded user choice is a specification defect, not permission to implement a different language.

| Part | Normative contents |
| --- | --- |
| This document, C2–C10 | Grammar boundaries, scopes, types, values, native collection contracts, initialization, errors, finite compiler checks |
| [Coordination specification](coordination-spec.md) | Completion regions, callables, four native Promise mappings, typed aggregate failure, ownership and settlement traces |
| [AI and I/O specification](ai-io-spec.md) | Connections, native questions and generated records, judge phases, fetch, LLM, codecs and protocol fixtures |
| [Platform and testing specification](platform-testing-spec.md) | Assertion identity, deterministic fixtures, catalogue opacity, CLI, HTTP, HTML/HTMX, SQL and capability coverage |
| C11–C12 | Cross-document traces, finding/answer traceability, evidence boundary and readiness |

The [independent review](deep-design-review-2026-09-20.md) is preserved as historical evidence. Its proposals, earlier unresolved verdicts and seven option recommendations are not present policy. [ASTRA_STDLIB](../ASTRA_STDLIB.md) supplies capability intent; its effects, exact-decimal base type, externs, revision syntax, proof obligations and browser component architecture are superseded. No compatibility layer, old generated ABI or golden output is a requirement. No LLM tools, anonymous functions, general foreign-code escape, browser Can compiler, or implicit cancellation is introduced.

Catalogue signatures use `T`, `U`, `E` and arrows as mathematical notation. They are not new source syntax. Catalogue operations have fixed compiler-known type rules. Application declarations use only the selected Can grammar.

<a id="c2"></a>
## C2. Lexical and grammar contract

### Lines, names and literals

Source is valid UTF-8. Accept LF or CRLF, normalize each to a source newline before tokenization; reject bare CR outside strings. A BOM is allowed only at byte zero and ignored. Outside literal contents, horizontal whitespace is ASCII space only; indentation is exactly four spaces per level and tabs are errors. Blank/comment-only lines do not change indentation. An indented block requires at least one significant line. Comments act as token separators, preserve source line boundaries and may nest as specified in decisions. Trailing spaces are insignificant. Files end with or without one newline.

No implicit continuation, backslash continuation or semicolon exists. Parenthesized expressions, call arguments, constructors, array literals and method chains remain on one physical line. A multiline string is one literal token and may span lines outside the explicitly one-line assertion, call, constructor and array forms. Only grammar-defined blocks (`match`, `do`, declaration sections, handlers) span multiple significant lines. A typed initializer may be a multiline `match` or coordination expression because its block is grammar, not continuation.

Identifiers match `[a-z][a-z0-9]*(?:_[a-z0-9]+)*`. The standalone `_` is a wildcard. Hard reserved words are `package provides uses as fn record variant error given near emits asserts call callable match chain do ok relay on with and or not is true false void int float bool str`. The other declaration/section words (`connection noul choice score judge choice_arm fetch llm from state asks minimum confidence describes endpoint auth bearer env timeout_ms metadata query headers body get post put patch delete head options concurrent race when`) are contextual tokens recognized in the productions that own them. They remain usable as data names elsewhere, including `int score` and `int minimum`; they cannot displace required tokens in those productions. `choice_arm` is also contextual at the beginning of a type. This completes the previously unspecified keyword inventory without invalidating selected examples.

Unsigned decimal integers are `0` or a nonzero digit followed by digits; reject decimal leading zeros. Base-prefixed integers use lowercase `0x`, `0b`, `0o` and at least one digit of that base; hexadecimal digits may be either case. A float is decimal digits with `.` and at least one fractional digit, or decimal digits with an exponent, or both. Exponents are `e`/`E`, optional `+`/`-`, then at least one decimal digit. `1e3` is a float; `.5` and `1.` are rejected. No underscores. Two dots begin an integer range token rather than a decimal point; `1..5` is integer/range/integer. Sign is an expression operator, not part of a literal except inside an exponent. Reject float literals that overflow to infinity; finite underflow follows binary64 rounding. Infinity/NaN can result from computation and are not reserved literals.

The string escapes are exactly `\n`, `\r`, `\t`, `\0`, `\"`, `\\`. Every other backslash sequence retains both characters; there is no Unicode-escape syntax hidden behind that rule. Ordinary quoted strings cannot contain literal newlines. Triple quotes preserve indentation and boundary newlines; raw forms preserve every backslash. The shortest closing delimiter terminates the string. Source Unicode scalar values are encoded into native UTF-16 strings; subsequent native code-unit operations may produce unpaired surrogates.

### Declaration and expression grammar

The following EBNF fixes disputed compositions. `NL/INDENT/DEDENT` are lexical tokens; comma lists are nonempty unless marked `?`. `name` can be package-qualified where `qualified` is used. Companion documents expand the native section and coordination productions; they are part of this grammar, not alternative dialects.

```text
file          = package_header, { declaration } ;
package_header= 'package', name, NL, INDENT,
                'provides', '[', names?, ']', NL,
                'uses', '[', imports?, ']', NL, DEDENT ;
qualified     = name, [ '::', name ] ;
type          = primary_type, { '[]' } ;
primary_type  = qualified, [ '<', types, '>' ]
              | 'int' | 'float' | 'bool' | 'str' | 'void'
              | 'callable', type, '(', types?, ')', error_bound
              | 'choice_arm', '<', type, '>', error_bound ;
error_bound   = 'emits', '[', error_types?, ']' ;
parameters    = '<', names, '>' ;
function      = 'fn', type, name, parameters?, NL, INDENT,
                receiver?, error_bound, NL, given?, assertions,
                steps, DEDENT ;
receiver      = 'on', type, name, NL ;
given         = 'given', NL, INDENT, input, { input }, DEDENT ;
input         = 'near'?, type, '...'?, name, NL ;
record        = 'record', name, parameters?, NL,
                [ INDENT, field, { field }, DEDENT ] ;
field         = type, name, NL ;
variant       = 'variant', name, parameters?, NL, INDENT,
                type, NL, { type, NL }, DEDENT ;
error         = 'error', integer, name, parameters?,
                '(', fields?, ')', NL ;
binding       = type, name, '=', expression ;
call_expr     = 'call', invocation, { '.', name, type_args?, arguments } ;
invocation    = callee, type_args?, arguments ;
arguments     = '(', argument_list?, ')' ;
argument      = expression | '...', expression ;
reference     = 'callable', callee, type_args? ;
completion    = 'ok', expression? | error_construction ;
terminal      = completion | ordinary_match | call_match | chain_match
              | 'relay', call_expr ;
steps         = { binding, NL | empty_error_void_call, NL
                | unbound_coordination }, terminal, NL ;
arm_body      = expression | completion | terminal | 'do', NL,
                INDENT, steps, DEDENT ;
```

Type parsing consumes a callable result type up to its input `(`, then its mandatory `emits [...]`, then any suffixes belonging to the completed callable type. Thus `callable int[] () emits []` returns an array; `callable int () emits [][]` is an array of nullary integer callables. A nested callable result is distinguishable by its own input list and error bound. Prefer storing such complex types in record fields when legibility matters; there is no new type-alias or parenthesized-type syntax.

Only judges and LLM invocations have a final state group inside their argument list. The group is required at every arity: `call name(())`, `call name((value))`, `call name(arg, (left, right))`. It is recognized from the resolved declaration, never constructed as an anonymous tuple. Ordinary functions do not accept a grouped comma expression. An empty state group has no values. Neither state groups nor ordinary call arguments allow spread into fixed state positions; use explicitly listed values.

Generic declaration parameters are registered before checking the entire signature, including a return type written before the declared name. In explicit call/reference/constructor contexts, `<...>` immediately after the callee is type application when followed by the corresponding arguments or reference terminator. Elsewhere `<` is comparison. Whitespace does not make application optional; no implicit generic application exists on arbitrary expressions.

Operator precedence, strongest to weakest:

| Level | Tokens / associativity |
| --- | --- |
| 1 | Primary, constructor/reference/call, field/property, indexing/slicing, method chain; left |
| 2 | `**`; right |
| 3 | Unary `-`, `~`; right (`-2 ** 2` means `-(2 ** 2)`; `2 ** -2` is syntactically valid) |
| 4 | `* / %`; left |
| 5 | `+ -`; left |
| 6 | `<< >>`; left |
| 7 | `&`, then `^`, then `|`; left at each level |
| 8 | `< <= > >= is is not`; one comparison-chain production |
| 9 | Unary `not`; right |
| 10 | `and`; left, short circuit |
| 11 | `or`; left, short circuit |
| 12 | Record `with`; receiver is the expression to its left; replacement expressions extend to the top-level comma/end |

Prefix `%` is a primary expression only in an AI handler context supplying it. Infix `%` is numeric remainder. `% % 0.1` is unambiguous. Pattern alternatives use `|` only in pattern grammar. Completion `ok` consumes a whole expression; it is not an operator precedence level. Multiple `with` updates require expression grouping; this prevents an unbounded assignment-looking tail grammar.

<a id="c3"></a>
## C3. Declaration tables, scopes and packages

One project package is one folder, with a unique flat package name; source files in it form one declaration table. Package declarations, method names, generated record types and generated native declaration names are registered before bodies. Same-scope duplicate names across kinds are errors. Public access is the union of file-local `provides` lists, each naming only declarations that file defines. A generated record-native declaration defines both its record name and operation name; export either or both explicitly. Exported signatures must not mention inaccessible project types, errors or connections. A public callable whose named errors are private is rejected.

The prelude supplies `append`, `choice_option`, `all_failed` and `standard_failure`; their names are reserved against redeclaration/shadowing and require no import. Catalogue package members otherwise require `uses`. Each file's aliases/imports are a separate lexical frame above its package table. Imports may not collide with declarations in that package, each other or reserved catalogue package names. A local can shadow an imported alias, but `alias::name` lookup considers package aliases only. Unqualified names search eligible declarations from inner to outer scopes: a type position sees types; a constructor position sees record/error constructors; a `call` position sees callables or executable declarations; an ordinary value position sees value bindings and bare named arm values. Lookup skips ineligible declaration kinds. This makes input `dimensions dimensions` and constructor `dimensions(3, 4)` coherent without permitting same-scope duplicates. A value binding of callable type can shadow a same-named outer function in a call position. Package qualification bypasses local shadowing.

Scopes are: package; file imports; generic parameters; receiver plus all function/native input declarations; each executable body; each `do`; each match arm; each native handler; each chain's success continuation; each coordination handler, including per-entry and shared success/error handlers. Inputs/receiver occupy a single frame and cannot duplicate each other or a type parameter. Locals enter scope only after their initializer, to the end of that block. A nested match inherits visible values; its arm bindings do not escape. Sibling arms never share bindings. All chain success names are visible to later steps and its `ok` arm, not failure arms. Native registration and answer scopes are defined by A's phase rules.

A matched error name is an arm-local value binding in addition to remaining an eligible constructor in constructor position. Bare `missing` in a completion arm forwards; `missing()` in its body constructs. Records in variant patterns narrow the existing scrutinee binding; error alternatives also bind their declared error name to payload data. Matching a non-binding expression still supports destructuring patterns, but a bare record-alternative pattern creates no implicit name for that expression.

Methods are top-level named functions owned by the package declaring their nominal receiver record. A package cannot add methods to somebody else's record, primitive, array or catalogue type. Receiver method names must be unique in that package declaration table; no overloads. Exported method lookup requires its owning package in `uses`, except within that package. Catalogue methods have fixed compiler-owned resolution. `call panel.area()` supplies the receiver once; `callable panel.area` captures it once. Methods cannot be invoked as static functions with a hidden receiver argument.

Project/dependency resolution and reserved catalogue names are defined in P. Apply the `internal` boundary to canonical filesystem locations after resolving symlinks, relative to the defining dependency root. Do not let alias spelling or a symlink bypass the boundary. Import cycles between source packages are allowed for signatures and functions; top-level value dependency cycles are rejected separately. No network lookup is performed by name resolution.

<a id="c4"></a>
## C4. Types, generics, immutable data and callable compatibility

Primitive value types are `int`, `float`, `bool`, `str`. `void` is only a success type (including callable/arm results); it is not a field, input, element, local or generic data argument. There is no implicit null/undefined/any type. Records and errors are nominal by fully qualified declaration identity and concrete type arguments. Values of errors are admissible ordinary immutable data: `ok missing()` can return an error value if the success type admits it, whereas terminal `missing()` produces that domain completion. There is no first-class raw completion type.

Nominal generics are invariant. Explicit type arguments must match declaration arity and be concrete after specialization. Inference solves type-parameter equalities from fixed arguments and an available expected result/reference/constructor type; repeated constraints must agree. It does not infer a union, widen int to float, infer an error set or search arbitrary instantiations. Ambiguity requires explicit type arguments. `[]` needs an expected element type; nonempty arrays infer one common identical element type unless an expected named variant admits the individual elements. Existing arrays do not implicitly change element type; an explicit `map` or expected construction performs widening.

A `variant` is a nonempty finite disjoint union of nominal record/error leaves. It may list other variants, flattened transitively, but no leaf may appear twice after concrete specialization. The prelude opaque `standard_failure` snapshot is one special admitted leaf for typed failure aggregates. Primitive, callable, arm, other opaque resource, `void`, array, and recursive variant-only alternatives are rejected. Recursive records are allowed only with a finite inhabitant, for example through `option::value<node>` or an array; a required cycle of records with no finite constructor path is rejected. Arrays/records may contain resources/callables unless a particular boundary requires equality or wire admissibility. Recursive type expansion must not generate infinitely many different instantiated types.

A value of a listed leaf is accepted in its named variant; a narrower variant can enter a wider named variant when its complete leaf set is included. No general structural record subtyping exists. Variants have no constructor/wrapper: pattern matching uses the actual leaf's nominal identity. The catalogue `option` package defines ordinary records `none` (fieldless), `some<item>` with field `item value`, and variant `value<item>` listing them. It introduces no special null syntax.

Type parameters are template parameters, with no new constraint syntax. Every concrete reachable specialization must typecheck its complete body, not just branches exercised by tests. Operations on a type parameter are permitted only when that specialization supplies the operation. Diagnostics identify both the generic source and requesting application. Reject expanding polymorphic recursion; same-specialization ordinary recursion is allowed. Compilation does not prove termination. An uninstantiated public generic retains a template checked for grammar, names and parameter-independent errors, and its consumer checks concrete instantiations. Assertion rows infer or explicitly construct their own concrete types; they check those instances and are not a proof for every specialization.

Callable types always spell an error bound, including `emits []`. After binding receiver and `near` inputs, a reference has the remaining ordered input types and declared result. Input and result types must equal the expected callable contract; use a named wrapper returning a declared variant when results must be normalized. Its domain error types may be a subset of the expected bound. No parameter contravariance or inferred purity is used. Captured immutable values are retained by native closures, not deep-copied. Every `near` value is resolved by exact name at reference creation; its type must equal the declared input type. Direct calls supply every `given` input, including `near`. Nullary collection spreads require no remaining ordinary inputs.

Generic error types may occur in `emits` after substitution, but there is no authored error-set type parameter. Compiler-known higher-order catalogue signatures derive a finite error bound from their actual callback contracts at each call site (C7). A user-written higher-order function declares a concrete finite bound, with callbacks whose errors fit it. This provides useful reusable fallible collection APIs without introducing an effect system or pretending that a generic data type denotes an arbitrary set of errors.

Copy-update evaluates the receiver once, then replacements left to right against the original environment, then creates a new record of the identical concrete type. Reject unknown/duplicate fields, wrong types, zero replacements, non-record receivers and catalogue opaque values. A replacement cannot refer to an earlier replacement as a newly rebound field. Unchanged immutable field references may be shared. Constructors and array expressions evaluate elements once, left to right. Source cannot mutate a field, array element or captured alias. Adapters cannot leak mutable native aliases.

Every Can payload crossing native Promise resolution must remain inside a compiler-private non-thenable completion or element box. Boxes have a null prototype, fixed own data properties, no `then` property, and no application-controlled keys. Generated async functions fulfill with these protected completions; coordination adapters convert their categories under Q3. Unwrap a payload only in a synchronous generated context, and rebox it before returning it from another async callback or promise continuation. This also applies to callback results, fold accumulators, fulfilled race winners, and nested native adapters. An authored record may legally contain a callable field named `then`: native `await`, `Promise.resolve`, or an async callback return must never execute that data field. Record field names and wire encodings are unchanged. The reason is native [Promise thenable assimilation](https://tc39.es/ecma262/multipage/control-abstraction-objects.html#sec-promise-resolve-functions), not a new source completion type. Implementations may choose private box layouts, but cannot rely on application records lacking a callable `then`.

<a id="c5"></a>
## C5. Expressions, patterns and completion ownership

All ordinary operand and argument evaluations are left to right and awaited before moving to the next. `and`/`or` accept bool operands and skip the unnecessary right operand. A comparison chain evaluates each operand at most once and skips later operands once a comparison is false; each adjacent pair must typecheck. There is no truthiness. A call with nonempty `emits` cannot appear as an unchecked arithmetic operand, constructor argument, initializer or later chain receiver. Use `match call`, `match chain`, or terminal `relay call` with explicit coverage. A chain expressed under one ordinary `call` prefix is directly usable only if every step has an empty error bound; `match call` may instead handle the entire method chain's union of declared errors.

A `match` in an ordinary-value context has value arms of one expected type and no completion arms. A terminal ordinary match has completion arms for the enclosing region. `do` inherits that region and requires two or more steps. Completing a region is terminal; statements after completion are unreachable errors. Ordinary `match call` and `match chain` are terminal constructs; this specification does not extend their binding positions merely because coordination has an approved local-result binding. Completion values cannot be stored by writing an unadmitted wrapper expression. A side-effect-only ordinary call step must return `void` and have `emits []`; nonvoid results are not silently discarded. A terminal function returning void uses bare `ok`.

An unbound coordination is a void step: its handlers use bare `ok`, no `void[]` is created, subsequent steps are allowed, and an explicit enclosing terminal completion remains required. Coordination's selected typed-binding form introduces a local completion region; Q defines its `ok`/failure behavior. Native question handlers similarly complete a selected result/field region; A defines assembly and propagation. An inner ordinary match/call match inside either of these completes that handler, not the outer function. A domain failure leaving a handler crosses the enclosing native/coordination contract without being redispatched to that construct's participant-error arms. Ordinary handler bodies and earlier effects are not rolled back.

Ordinary matching is ordered and exhaustive. Patterns are literals; `_`; record constructors with one positional subpattern per field; bare variant leaf names; bare error names; integer literal inclusive ranges; arrays with required positional subpatterns and optional final `...name`; and `|` alternatives. A lower-case name in an array/record field pattern binds the statically known field type; typed declaration syntax is not allowed inside a pattern. In a bare-name pattern position, a resolvable leaf of the expected variant denotes that leaf; otherwise a non-keyword name binds the entire expected value. No existing value name is implicitly a constant-equality pattern. Arrays without a remainder require exact length. A rest binding receives a new immutable slice. Range bounds are integer literals optionally negated, with lower <= upper; no float, open or computed ranges. Alternatives must bind exactly the same names with the same types and compatible narrowing in all alternatives; bindings have their common admitted type.

The exhaustive-check algorithm covers bool's two values, every finite variant leaf, constructor products of covered patterns, literal/range unions for integers, and array length partitions (fixed lengths plus minimum-length rests). Infinite string/float/int spaces otherwise require `_` or a variable binding. Do not perform semantic theorem proving over guards; there are none. Reject arms proved fully covered by earlier arms. Overlapping nonempty regions remain first-match. Completion dispatch requires exactly one success arm and one arm for each declared domain error kind/instantiation, plus at most one optional `[_]`; no `_`, range, `|`, positional error payload or omitted-outcome wildcard is admitted there.

Source call spread flattens an array at the position written. For fixed parameters a statically known literal spread may supply known positions; a runtime-length spread is permitted only in the trailing variadic portion after every fixed argument has been supplied. Reject a runtime-length spread into fixed arity instead of inventing undefined or a late arity exception. A variadic `near` input is rejected: `near` represents one captured declared value, not a changeable number of invocation positions. Method receiver arguments do not participate in spread.

<a id="c6"></a>
## C6. Primitive semantics and native mappings

| Operation | Contract / native implementation |
| --- | --- |
| `int` arithmetic | Native bigint `+ - * / % **`; division truncates toward zero, remainder has dividend's sign. `-7 / 3 == -2`, `-7 % 3 == -1` in explanatory mathematics (`is` is source equality). Division/remainder by zero and negative integer exponent produce standard `arithmetic` failures. `0 ** 0` is 1. No fixed overflow bound. |
| `float` arithmetic | Native binary64 `+ - * / % **`, IEEE infinity, signed zero and NaN outcomes included. Both operands must be float, including exponent. No implicit literal conversion. Negative-base fractional powers yield native NaN. |
| Bitwise | `& | ^ ~ << >>` accept/return int using native bigint operations. Negative shift counts reverse direction as native bigint does. No unsigned-right-shift spelling; no 32-bit coercion of floats. |
| Equality | Same admitted static type, with variant inclusion to a common expected named variant. Same nominal type and recursively same data. Scalar float equality is native `Object.is`: NaN equals NaN and +0 differs from -0. Other primitive equality is native strict equality. Records/arrays/errors use `Bun.deepEquals(..., true)` over canonical compiler-owned data representations including nominal identity; no custom recursive equality algorithm. `is not` negates this. |
| Equality eligibility | Primitives, arrays, nominal records/errors/variants whose reachable fields are eligible. Exclude callables, arms, connection declarations, resource/opaque values unless the catalogue defines an explicit comparison. Reject equality on a container of excluded fields. Whole-completion assertion comparison uses these rules and exact error kind, ID, specialization and payload. |
| Ordering | int/float numeric `< <= > >=`; str native UTF-16 lexicographic order. Mixed numerics, bool and record ordering rejected. Every ordered comparison with NaN is false; use an explicit finite predicate where a total order is needed. |
| Array/string `.length` | Native property widened exactly to bigint `int`; arrays have native finite storage capacity, distinct from int's mathematical range. |
| Indexing | Index must be int; require `0 <= i < length`, then convert exactly to native index. Negative or out-of-range index is standard `bounds` failure. Array returns element; str returns one UTF-16 code-unit str, possibly an unpaired surrogate. |
| Slicing | Half-open, omitted start=0/end=length. Negative bounds count from end; clamp each independently to [0,length], start>=end gives empty. Normalize using bigint before exact native conversion; delegate copying to native `.slice`. Brackets and `.slice(start,end)` agree. No implicit integer-to-float source conversion occurs. |
| String concatenation | str+str only, native `+`. No interpolation or implicit formatting. |

These choices deliberately select native semantics where the earlier catalogue offered alternatives. They do not imply UTF-16 is a Unicode scalar, float is exact money, or reference identity equals immutable data equality. ECMAScript specifies [numeric operations](https://tc39.es/ecma262/multipage/ecmascript-data-types-and-values.html#sec-numeric-types); Bun provides [deep equality](https://bun.com/reference/bun/deepEquals). Read-only Bun 1.4.2 probes verified strict equality on fresh record-shaped objects, NaN equality and distinction of signed zeros. Conformance must also verify canonical nominal tags, recursive admitted data and arrays; a runtime upgrade is not authority to change Can equality.

Explicit conversion catalogue (all parameters positional):

| Signature | Declared errors / exact behavior |
| --- | --- |
| `text::from_int(int) -> str`; `from_float(float) -> str`; `from_bool(bool) -> str` | `[]`; native `String`, base ten for int, shortest native number spelling for float, including `NaN`, `Infinity`, `-Infinity`; -0 formats as `0`. No locale, percentage scaling or requested decimal precision. |
| `number::int_to_float(int) -> float` | `[number::inexact]`; native Number then finite check and BigInt round-trip equality. |
| `number::float_to_int(float) -> int` | `[number::inexact]`; finite and integral required, then native BigInt; this converts the actual binary64 value, not an unknown original decimal token. |
| `number::floor(float)`, `ceil`, `trunc`, `round(float) -> float` | `[]`; corresponding Math operation; `round` uses native ties toward +infinity. Nonfinite values remain nonfinite. Exact money rounding uses C8. |
| `number::is_finite(float)`, `is_nan(float) -> bool` | `[]`; native Number predicates. |
| `text::to_int(str) -> int` | `[text::invalid_number]`; full-match `-?(0|[1-9][0-9]*)`, native BigInt. No whitespace, base prefixes, + sign or exponent; `-0` yields 0. |
| `text::to_float(str) -> float` | `[text::invalid_number]`; full decimal source numeric grammar with optional leading minus; native Number, must be finite. |
| `text::to_bool(str) -> bool` | `[text::invalid_bool]`; exactly `true` or `false`. |
| `number::bool_to_int(bool) -> int`; `int_to_bool(int) -> bool` | First `[]` maps false/true to0/1; second `[number::invalid_bool]` admits only0/1. |

Use the existing `call` marker for every conversion. Domain error allocations: `1000 number::inexact(str reason)`, `1001 text::invalid_number(str input)`, `1002 text::invalid_bool(str input)`, `1003 number::invalid_bool(int value)`. The numbers in older illustrative snippets are not registry allocations; real application IDs use C9's range.

<a id="c7"></a>
## C7. Collection and text catalogue

Every method below awaits its callback; no callback is passed unchanged into a synchronous native predicate/comparator. For mathematical callback bound E, the catalogue invocation exposes exactly E plus listed intrinsic domain errors. This is compile-time specialization of a closed catalogue operation. There is no hidden domain conversion, automatic propagation or implicit concurrent traversal. Creating `callable name` captures once before traversal; callback invocation does not recapture locals.

| Array `T[]` operation | Inputs/result; visits; mapping |
| --- | --- |
| `.map` | `callable U(T) emits E -> U[]`; each index ascending once; stop on first domain/standard failure. Native `Array.fromAsync` iterates the source array's native `.keys()` iterator; its generated async mapper invokes the Can callback for that index and returns a private success box or rejects with the appropriate failure tag. A final native synchronous `.map` unwraps the collected boxes. Iterating indices prevents native input awaiting from assimilating data records; boxed mapper results protect outputs. No downstream promise graph or user effects are started eagerly. |
| `.filter` | `callable bool(T) emits E -> T[]`; same visits; sequential predicate observations then native `.filter` on the completed boolean vector. No elements are exposed on callback failure. |
| `.for_each` | `callable void(T) emits E -> void`; sequential ascending visits, stop on first failure. Native `.reduce` builds the await chain; never native `forEach(async ...)`. |
| `.fold` | initial `U`, `callable U(U,T) emits E -> U`; native `.reduce` with awaited accumulator, supplied initial even for empty input. |
| `.find`, `.some`, `.every` | `callable bool(T) emits E`; result `option::value<T>`, bool, bool. Ascending visits; stop at first true/true/false respectively. Empty results none/false/true. Native async iteration via a bounded await adapter supplies the missing await/short-circuit capability; no synchronous predicate sees a Promise. |
| `.sort_by` | `callable K(T) emits E -> T[]`, K=int/finite float/str/bool; extract one key per element sequentially, then native `.toSorted` over key/index decorations with a synchronous built-in ascending comparator. NaN/infinite keys are standard `arithmetic` failures. Bool keys order false before true. The generated comparator uses native pairwise comparisons to return JS Number -1/0/1 (never bigint subtraction); equal keys use original index order, and +0/-0 are equal sort keys. No authored comparator receives hidden synchronous status. |
| `.slice(int,int)`, `.concat(T[])`, `.to_reversed()` | `[]`; native `.slice`, `.concat`, `.toReversed`; no mutation. The spelling `to_reversed` is a Can catalogue name. |
| `append(T[],T) -> T[]` | Prelude function, `[]`; native array spread or equivalent copy. No append method. |

Local adapters for sequential callback traversal are expressly needed because native synchronous search predicates do not await. They traverse native arrays but do not recreate sorting, filtering, string or copying algorithms. Concurrent traversal is expressed with Q's named nullary callables; there is no accidental Promise.all hidden in ordinary map scheduling. Internal scheduling and comparison callbacks may be generated JS closures; the no-anonymous-functions rule governs authored Can source.

The selected target must provide native `Array.fromAsync`; absence is a target-conformance error, not permission to substitute concurrent mapping. Its [TC39 algorithm](https://tc39.es/proposal-array-from-async/) awaits each mapper result before advancing. Bun 1.4.2 probes observed ordered completion, one active callback, rejection stopping before the next callback, empty output, and protected records containing callable `then`. These bounded observations support the mapping; implementation still must verify typed completions, captures, and failure propagation. Filter decisions and sort-key extraction can reuse this native sequential mapping contract; short-circuit search retains its bounded await adapter.

String methods with `emits []`: `.includes(str)`, `.starts_with(str)`, `.ends_with(str)` return bool; `.to_lower_case()`/`.to_upper_case()` return str using native Unicode casing; `.trim()` uses native ECMAScript whitespace; `.slice(int,int)` follows C6. `.split(str) -> str[]` rejects empty separator with `text::empty_separator()`; otherwise native split with no limit and preserved empty pieces. `.replace_all(str,str) -> str` rejects an empty search with `text::empty_pattern()` and uses native literal replacement; replacement `$` sequences are literal data, requiring native function-replacement adaptation rather than native replacement-template interpretation. `text::join(str[],str) -> str` uses native join. No regexp or arbitrary prototype surface is implied. Allocations:1004 empty_separator,1005 empty_pattern.

Named Unicode operations supplement native code-unit syntax: `text::scalars(str)->int[]`, `text::from_scalars(int[])->str`, `text::graphemes(str)->str[]`, `text::normalize_nfc(str)->str`. The first three validate scalar well-formedness and emit `text::invalid_unicode(str reason)` (1006); NFC also rejects unpaired surrogates. Use native string iteration/codePointAt/fromCodePoint, Intl.Segmenter with locale `und` and grapheme granularity, and `.normalize('NFC')`, with native bulk conversion chunking where argument limits require it. Segmentation/casing/normalization behavior is tied to the packaged Bun Unicode/ICU revision. Code-unit indexing and scalar access are intentionally distinct. Full Unicode casefolding is not confused with lowercase; its advanced catalogue status is recorded in P.

Immutable map/set keys are initially int, bool or str. Opaque catalogue `collections::map<key,value>` and `collections::set<key>` use native Map/Set internally; iteration is insertion order. Equality uses native key equality (same as C6 for these key types). `empty_map<K,V>()`, `empty_set<K>()` return empty values. `get(map,key)->V` emits `collections::key_absent()`(1007); `insert(map,key,value)->map` emits `key_exists()`(1008), `replace`/`remove` emit key_absent. Return fresh native Map copies; aliases remain unchanged. `entries(map)->entry<K,V>[]` preserves insertion order, where ordinary catalogue record `collections::entry<K,V>` has ordered fields `K key`, `V value`. `contains(set,key)->bool`, `add(set,key)->set`, `union`, `intersection`, `difference` have empty bounds; add duplicate is a no-op value result, union retains left order followed by unseen right entries, intersection/difference retain left order. Use native immutable copies and supported Set operations. Map/set decoding does not bypass opacity; convert through ordinary entry arrays with explicit duplicate validation. Structural record-key hashing and arbitrary equality callbacks are outside this initial contract.

<a id="c8"></a>
## C8. Exact quantities, initialization and the local-binding rule

An exact amount is an ordinary domain record with integer minor units and a declared currency/scale; no dec type is restored. Primitive bigint arithmetic handles balances and counts. The catalogue `number::divmod(int,int)->number::division` returns an ordinary catalogue record with ordered fields `int quotient`, `int remainder` with truncating C6 semantics and domain `number::zero_divisor()` (1009). Applications wanting Euclidean nonnegative remainders use `number::euclidean_divmod`, a sign adjustment around native division/remainder, with the same bound. `number::round_ratio_half_even(int numerator,int denominator)->number::rounded` returns an ordinary catalogue record with ordered fields `int value`, `int remainder_numerator`, `int denominator` normalized to positive denominator, preserving `numerator/original_denominator = value + remainder_numerator/denominator`; zero denominator emits zero_divisor. Normalize to numerator n and positive denominator d, compute native truncating q and r, and compare 2*abs(r) with d. When greater, adjust q by one in n’s sign; when equal, adjust only if q is odd; when less, retain q. Recompute remainder_numerator = n - q*d. This finite adapter supplies exact financial rounding, not a second arbitrary-precision arithmetic implementation. No GCD or certified transcendental kernel is claimed to be native; extended catalogue status is explicit in P.

Top-level values are initialized exactly once before `main` from a restricted closed expression set: literals; other top-level immutable values; record/error/array constructors; grouping; field access; primitive operations; indexing/slicing; copy-update; and capture-free named arm values. No `call`, `callable`, match/coordination, environment read or resource acquisition is permitted there. This syntactic rule does not classify functions as pure. Compiler-owned query/codec descriptors are derived at intrinsic call sites or from inert manifest data, never by an executable top-level call. Initialization follows the topological graph of referenced values, deterministic qualified-name order among independent nodes. Cycles are errors; a primitive standard failure during initialization is reported with its source location before main begins. Functions and type names are forward-referenceable.

Connections are immutable configuration declarations, not ordinary user-constructible records. Their static setting validation and runtime credential lookup follow A; initializing a connection never sends a request. Package loading performs no database migration, network request or effectful application callback.

The unnecessary-local error is the following decidable rule, and nothing broader:

1. A typed local is immediately followed in the same block by the terminal line `ok name` (or by `name` as the final value of a value-producing match arm).
2. The binding has exactly that one direct source use and is never required by an implicit `near` capture in the scope.
3. Its initializer AST contains only literals, existing immutable value names, parentheses, non-faulting field reads and primitive unary/binary/comparison/short-circuit expressions over them. Exclude every call/reference, constructor, array literal, match, indexing/slicing, division/remainder/power/shift, resource access and contextual AI `%`.
4. Checking the substituted expression in the terminal's expected type yields the identical concrete type and variant conversion, without changing type inference.

When all four hold, reject with a diagnostic showing the direct replacement. `int result = left + right` then `ok result` is rejected. A binding that captures `factor`, retains a fetched observation, supplies an empty array's expected type, changes effect timing or has two uses is permitted. The compiler performs no arbitrary equivalence proof, global purity inference or cost model. This is a language diagnostic, not a claim that every other local is essential.

<a id="c9"></a>
## C9. Domain IDs and standard failures

The exact registry and dependency-lock formats, and their required agreement with source declarations, are specified in [P2](platform-testing-spec.md#p2-distribution-catalogue-identity-and-linkage).

Every domain error has a stable positive integer ID at most 2147483647, unique in the resolved application and dependency graph. Identity is its declaring package/declaration plus concrete generic arguments, not its integer alone. All instantiations of a generic kind share one ID. Catalogue IDs1–999999 are distribution-reserved; project/dependency declarations allocate1000000–2147483647. The distribution registry allocates100 to `all_failed`,1000–1099 to this core catalogue,1100–1199 to A and1200–1299 to P. Unallocated reserved IDs are not usable. A dependency lock records its published allocations; resolving duplicate IDs is a compile error, not renumbering. A project registry records retired IDs; do not reuse a retired ID for another kind. Renaming/replacing an obsolete unpublished design may explicitly update the registry; no compatibility obligation is implied. Compiler diagnostic codes use a separate string namespace and are not domain IDs.

Bare error-arm lookup must identify one concrete payload type for that kind in the matched bound. If composition introduces the same generic error kind with incompatible concrete payload types, normalize through named wrappers into a common named variant/error contract before matching. Reject ambiguity rather than infer an anonymous union. Q defines `all_failed<F>`'s expected named-variant inference and payload preservation; the arm spelling remains `all_failed`.

Standard failures are outside every `emits` bound. Propagatable kinds are `arithmetic`, `bounds`, `resource_state`, `assertion`, `native_exception` and `cleanup`. Runtime owns a unique occurrence identity, original native cause and source/invocation path. Use the prelude opaque `standard_failure` data projection only where Q/P explicitly expose it; it is not a constructible application error or an untyped JSON escape. Its stable observations are kind, message and occurrence ID. The original cause is retained internally and never coerced into a declared domain error without an explicit handler.

The standard-message templates for Can-created failures are fixed:

| Kind / cause | Handler message |
| --- | --- |
| arithmetic / zero integer divisor | `arithmetic: integer division by zero` (division), or `arithmetic: integer remainder by zero` (remainder) |
| arithmetic / negative integer exponent | `arithmetic: negative integer exponent` |
| arithmetic / nonfinite sort key | `arithmetic: nonfinite sort key` |
| bounds / invalid index | `bounds: index out of range` |
| resource_state | `resource_state: invalid resource use` |
| cleanup | `cleanup: automatic resource cleanup failed` |
| assertion | `assertion: ` plus P's finite failure class, e.g. `missing fixture`; detailed paths are separate runner diagnostics |

Arguments, source locations and underlying causes remain structured runtime diagnostics rather than variable additions to these templates. Native capacity/engine exceptions without a selected Can primitive guard use native_exception.

For `[_] as str message`, primitive Can failures use these templates. For a non-proxy native Error, use string-valued own data descriptors for name/message; missing name falls back to the recognized built-in prototype name or `Error`, and missing/non-data message to an empty string. Join nonempty message with `: `. Native `node:util` type predicates reject proxies before descriptor inspection; unknown objects/functions/proxies/symbols get fixed type labels. A thrown string uses that string; number/bigint/bool/null/undefined native primitives use native String. Never invoke a getter, proxy trap, arbitrary toString, constructor property or whole-object serialization while describing a failure. Native detection uses [util.types](https://nodejs.org/api/util.html#utiltypesisnativeerrorvalue), verified with Bun1.4.2; a diagnostic-adapter defect falls back to `native failure` without replacing the original occurrence. No stack trace, credential or response body is automatically appended to the handler string. Platform adapters translate their specified expected errors before this boundary; unknown programming exceptions remain standard failures.

An ordinary `match call` standard catch covers receiver/callee and argument evaluation, the invoked body, and each method-chain step; a `match chain` catch covers the corresponding evaluation of every reached step. Coordination preparation instead lies outside participant arms under Q2. No catch covers exceptions raised by its selected handler itself. Uncaught standard failures propagate automatically, with the explicit coordination observation qualification in Q. An uncaught root failure emits a diagnostic and nonzero exit. Fatal process termination, abort, stack exhaustion that prevents handler execution or out-of-memory is not promised recoverable. No catch-all domain-error wildcard follows from this standard channel.

<a id="c10"></a>
## C10. Complete primitive and callback traces

These are specification traces, not claims that the historical compiler accepts current syntax. A declaration fragment omitting a package header is identified as a fragment; semantic tables show every branch outcome.

A complete ordinary package establishes native values, expected types and assertion comparison:

```text
package main
    provides [main]
    uses []

/// Returns a fresh pair of equal integer values.
fn int[] pair
    emits []
    given
        int value
    asserts
        exact: 9007199254740993 => ok [9007199254740993, 9007199254740993]
    ok [value, value]

/// Exercises immutable native array copying.
fn void main
    emits []
    given
        str[] arguments
    asserts
        empty_args: [] => ok
    int[] values = call pair(9007199254740993)
    match values is [9007199254740993, 9007199254740993]
        true => ok
        false => match call pair(values[2])
            ok int[] unexpected => ok
            [_] as str message => ok
```

The exact assertion compares data, not allocation identity. `arguments` contains application arguments only (P). The false branch deliberately traces an indexing fault while evaluating the matched invocation's arguments: it is within that call's standard-failure catch boundary. No element at index2 is synthesized. The branch is not reached for the shown values; P's target fixtures separately exercise it. No extra declaration is allowed to reuse the same name as `pair` in this package.

Callback trace for `call [1,2,3].map(callable transform)` (expression fragment): transform input1 awaits to success10 before input2 begins. If input2 emits a declared error, input3 never begins, the map has no successful partial array, and its caller must handle transform's declared error set. For filter, predicate false at1 then true at2 retains the original second element; a Promise is never used as a truthy predicate. For some, true at2 stops before3. For sort_by, keys for all elements finish sequentially before a synchronous native sort begins; equal keys preserve original order. Empty map/filter/sort_by return[], empty fold returns its initial value, empty for_each returns void success.

Exact quantity trace: balance9007199254740993 minor units plus2 remains9007199254740995; no Number conversion appears. Half-even5/2 returns value2 and remainder1/2;7/2 returns4 and remainder-1/2; -5/2 returns-2 and remainder-1/2. Non-tie8/3 returns3 and remainder-1/3;5/-2 normalizes to-5/2 and returns-2 with remainder-1/2. These records can be encoded through A's integer-token codec without losing a unit. A client wanting decimal-string interoperability declares actual str wire fields and converts explicitly.

<a id="c11"></a>
## C11. End-to-end closure and traceability

The following integration sequence uses the exact component contracts; companion traces supply complete declarations and failure tables rather than requiring a new source form at a transition.

| Stage | Typed crossing and observable failures |
| --- | --- |
| Ticket input | Named fetch decodes an ordinary record containing str text and int count; A's full emits includes transport/status/codec failures. 9007199254740993 survives original-token decoding; malformed/duplicate data produces no nominal record. |
| Generation | `llm routing_plan ...` receives final grouped state; provider protocol and schema subset are checked, then A's shared codec validates generated data. Text-only uses `llm str` with the same grouping. Refusal/truncation/invalid JSON is a declared failure, never an empty pretend plan. |
| Transformation | An ordinary named function maps generated candidate records into `choice_option` data using sequential `.map`; generated descriptions are data, never executable code. |
| Judgment | A's judge phases register independent Noul/Choice/Score over one state/connection, send once, validate all answers, and then run handlers in source order. Answer-dependent questions move to a later judge call. Handler failure stops later handlers and prevents partial output. |
| Persistence | Q's nullary captured operations can invoke P's checked SQL operations; `concurrent with error` maps every input outcome in input order to one common application result. Native allSettled retains both database failures. |
| Replica read | Plain race waits for one success or yields one `all_failed<F>` preserving typed domain values and standard_failure occurrences. Duplicate failure kinds do not collapse; empties and pending participants have Q's explicit outcomes. |
| Response | P's immutable response contract emits typed JSON or an opaque HTML fragment. No app backend adapter, body-stream alias, implicit coercion or LLM tool is needed. |
| Browser continuation | Upstream HTMX submits/query-fetches from Bun routes and swaps safe rendered fragments; form validation, pending state and dashboard refresh are server-driven. No Can code or credentials compile into a browser target. |
| Test | P's root assertion/dynamic-path identity chooses fixtures before concurrency timing matters. Consumer substitution tests callers; raw protocol fixtures test codecs/native handlers; target conformance tests Bun/SQL/HTML adapters. None claims live model quality. |

### Complete consumer of generation and judgment

Combine the declarations in [A12.1](ai-io-spec.md#a121-generated-data-followed-by-runtime-choice) and the function below in one package with this header. This is a library package; its public function may be called from an application's main or HTTP handler. The two assertions replace only the specified native calls. Candidate mapping executes real Can code, and the refused path never reaches mapping or judgment.

```text
package ticket_routing
    provides [route_email]
    uses [ai, codec, http, llm]

// Include A12.1's documented record, connection, native and to_option declarations here.

/// Generates routing criteria, then judges the email against them.
fn str route_email
    emits [http::invalid_request, http::credentials_missing, http::transport_failed, http::timeout, http::body_limit, http::status_error, codec::invalid_data, llm::refused, llm::truncated, llm::invalid_response, ai::invalid_question, ai::invalid_answer]
    given
        str email
    asserts
        routed: "Refund please." => ok "billing"
        refused: "Cannot summarize." => llm::refused("provider_refusal")
    match call propose((email))
        when
            routed: ("Refund please.") => ok suggestion("Which team?", [candidate("billing", "Payments."), candidate("technical", "Product faults.")])
            refused: ("Cannot summarize.") => llm::refused("provider_refusal")
        ok suggestion proposed => match call route(proposed.question, call proposed.candidates.map(callable to_option), (email))
            when
                routed: "Which team?", [choice_option("billing", "Payments."), choice_option("technical", "Product faults.")], ("Refund please.") => ok "billing"
            ok
            http::invalid_request
            http::credentials_missing
            http::transport_failed
            http::timeout
            http::body_limit
            http::status_error
            codec::invalid_data
            ai::invalid_question
            ai::invalid_answer
        http::invalid_request
        http::credentials_missing
        http::transport_failed
        http::timeout
        http::body_limit
        http::status_error
        codec::invalid_data
        llm::refused
        llm::truncated
        llm::invalid_response
```

Native-call `when` rows use the invocation's final state group, matching the declared native input grammar; these groups are not ordinary assertion input tuples. A whole package formed as stated declares every referenced application name. The full emitted domain bound remains visible even though these two supplied completions exercise only success and refusal. Raw fixtures in A12 exercise actual native construction/decoding and the duplicate-option rejection; P's reports distinguish that evidence from this consumer test.


Finding resolution map (all fifteen are retained; “resolved” means a written contract, not implementation evidence):

| Review finding | Resolution / normative sections | Evidence and remaining boundary |
| --- | --- | --- |
| F01 region/completion contracts | C4–C5/C9; Q regions; A native emits/handlers | Finite upper bounds, terminal ownership, handler failures distinct from participant failures; implementation conformance required |
| F02 coordination algebra | Q four-mode/empty/aggregate tables; C4/C9 | Native Promise reuse, explicit typed all_failed, ordered handlers; no new cancellation promise |
| F03 async callbacks | C7/C10; P host callbacks | Native map/filter/sort adapters and sequential stopping traces; all Can functions remain async |
| F04 shared codecs/exact numbers | A codec; C4/C6/C8 | Native original JSON token recovery, duplicate guard, one admitted wire subset; driver/provider conformance separate |
| F05 judge phases/connections | A judge phases/connection identity | One independent batch, deterministic answer validation and handler order; true dependencies require later request |
| F06 callables/arms | C2–C4; Q callable collections; A reusable arms | Expected finite bounds, exact captures, capture-free typed arm values; no anonymous/error-set syntax |
| F07 fetch/LLM capabilities | A transport/fetch/generation | Body/envelope/text/bytes, concrete protocol profile, complete declared failures; no tools |
| F08 assertions/fixtures | P assertion identity and fixture formats; C6 equality | Repeated/transitive/concurrent fixtures and separate evidence levels; live observations remain external evidence |
| F09 early settlement ownership | Q ownership; P resource/shutdown | Live participants retain runtime owner, leases, cleanup disposition; no automatic loser cancellation |
| F10 application boundary | P CLI/HTTP/HTML/HTMX/SQL; C3 | Bun service and SSR interaction close initial scope; browser Can explicitly excluded by U7 |
| F11 provenance | C4/C7; P opaque catalogue types | Unforgeable native-backed safe/resource values; no general opaque declaration added |
| F12 native equivalence | C2/C5–C8; A numeric validation | Complete operator/bounds/conversion choices and fractional AI values; native conformance still to run in implementation |
| F13 grammar/scope | C2–C5; A native productions; Q block grammar | Contextual keyword inventory, eligible-kind lookup, generic header scope, approved distinct layouts |
| F14 useless locals | C8 | Finite immediate-forward rule; excludes captures/effectful operations and expected-type changes |
| F15 stale authority/examples | Reconciled decisions; C1 | Tools removed, bare error patterns, float literals, valid do and explicit native emits; audit preserved unchanged |

| User answer | Adopted contract and location |
| --- | --- |
| U1 ordinary native emits | decisions native-error section; A/Q plus C9: intrinsic domain bounds are explicit, standard failures separate |
| U2 parameterized stored arms | decisions arm section; A: `choice_arm<T> emits [...] field`; named return-first declarations retained |
| U3 llm str plus grouped state | decisions LLM; C2/A: final grouped state even at arity1, zero group `()`, no returns section |
| U4 one all_failed aggregate | decisions race; Q/C9: bare all_failed, one stable generic error kind/ID, preserved input-order failures |
| U5 named fetch body/envelope | decisions fetch; A: method/query/headers, body encoding and http::response<T> preserved |
| U6 explicit float formatting | decisions generated fields; C6/A: ordinary text::from_float call, no contextual coercion or automatic scaling |
| U7 Bun SSR plus HTMX | decisions boundary; P: live forms/search/dashboard using upstream HTMX, no browser Can/app JS adapters |

<a id="c12"></a>
## C12. Readiness and verification boundary

The specified initial Can/Bun contract is ready for implementation planning. There are no remaining required user syntax or policy decisions for the traced AI, coordination, CLI, HTTP, PostgreSQL and HTMX programs. This assessment follows reconciliation of every review finding and all seven answers, including independent checks of the companion contracts. This specification is the input to implementation planning, not an implementation plan. Source syntax, runtime/type contracts and catalogue operations are requirements; no old compiler result is used to validate them. The historical review remains unchanged, and this work changes documentation only.

Evidence consists of complete current-decision/catalogue reading, independent domain review, primary native/provider documentation, bounded read-only Bun probes, and [24 paraphrased Jev design consultations](jev-design-consultations-2026-09-20.md) across eight technical choices. The consultation record preserves exact requests, responses, uncertainty, and the reviewing agents' separate judgments; repeated agreement is not proof. Required implementation evidence is explicitly separated: a parser/checker for this grammar, native lowering conformance, raw provider fixtures, deterministic assertion execution, driver/schema checks and HTMX integration. The live Jev requests were direct advisory consultations, not generated Can executions or adapter validation; no live database or deployment was used. Defining a contract does not assert model calibration, network reliability, transaction certainty or automatic effect rollback.

The initial boundary is deliberate: exact int plus binary64 (no dec); immutable native-backed collections; finite catalogue-specific callback error specialization (no authored error-set syntax); nonrecursive structured generation in the selected Responses profile; PostgreSQL with bounded materialized rows; and server-driven HTMX interaction. Broader numeric algorithms, database dialects/migrations/streaming, dynamic routes and offline browser computation are not prerequisites for these closed traces. P14 records their separate disposition rather than silently inheriting the historical catalogue’s proposed mechanisms.

Readiness does not mean an implementation has passed its conformance gates. No compiler, generated TypeScript, package publication, deployment or live database/model operation is part of this change. Implementation planning should treat the specified grammar, adapters, catalogue contracts and trace outcomes as acceptance criteria; the historical compiler and old goldens carry no compatibility authority.
