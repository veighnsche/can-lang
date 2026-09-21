# Current C2 syntax boundary

`Parse(*source.File)` is the current core grammar authority. It returns explicit
nodes or diagnostics, never a partially accepted file. It only reads the supplied
source object: parsing does not resolve imports, evaluate assertions/bodies, read
the environment, or call providers. The command `canlc parse [--render] FILE.can`
exposes this boundary, including in a staged distribution. `--render` formats the
syntax tree to stdout; it omits comments without modifying the input file.

The AST distinguishes nominal/array/callable/choice-arm types, declarations,
immutable bindings, expressions, patterns, completion bodies and coordination
participants. Comparison chains retain all operands and operators. Original-byte
spans and separate comment trivia remain available. Formatting retains authored
grouping, particularly the distinction between ordinary call arguments and
parenthesized state syntax; parse/render tests compare node kinds and payloads.

Core syntax includes ordered package headers, return-first functions with required
emits/asserts and optional receiver/given sections, generic records/errors/variants,
top-level values, calls/references/constructors, arrays/spreads/slices, copy updates,
ordinary and completion matches, call-site when tables, chain, relay and do.
Coordination syntax includes all four headers, direct/spread participants, typed
bindings and the selected per-participant/shared handler placement. An unbound
coordination remains a step and requires a subsequent terminal completion.

Semantic checks belong to later checker tasks: eligible-kind lookup, duplicate
names, completion coverage, type/effect compatibility, valid error identities,
range ordering, and whether a grouped argument belongs to a native declaration.
Empty/multiple state groups are argument-only nodes, never tuple expressions;
one-value groups retain their ordinary GroupExpr spelling until resolution.
Native AI/fetch declarations and handler-specific `%` syntax remain I16.

`ParseType` and `ParseExpression` provide inert fragment entry points. Recursive
type/expression/pattern/block parsing is bounded at 256 levels. Right-angle token
fragments are held in a parser-local pending token, so speculative type application
does not mutate lexer results or copy the entire token stream at every expression.
Arguments are parsed once; nested parenthesized calls do not trigger exponential
speculative reparsing.

The predecessor parser is explicitly named `compiler/legacy_parse.go`. It supports
the old checker/emitter/editor until their scheduled replacement and I44 retirement;
the current parser neither imports it nor translates its nodes into that model.

`Lex(*source.File)` produces tokens with original byte spans, decoded string
values, raw/multiline flags, separate comment trivia and precise diagnostics.
Parsers must refuse results with diagnostics; every result ends with EOF.
There is no dependency on the predecessor line/brace scanner.

Significant lines emit NEWLINE/INDENT/DEDENT. Each indentation increase is one
four-space level; comment-only and blank lines do not open or close a block.
Comments are token separators, may nest, and preserve physical newline
boundaries. Documentation comments remain distinguishable trivia for I04.
Strings consume LF/CRLF as one normalized newline while retaining their exact
original spelling in token text. Raw forms preserve backslashes; ordinary
forms decode only the six C2 escapes and preserve all unknown escape pairs.

Hard keywords are classified as keywords. Contextual words remain names; the
owning grammar decides whether score, minimum, choice_arm and similar words
are section/declaration tokens or data names. Identifier spelling is ASCII and
lowercase. Numeric tokens retain exact spelling; decimal integers reject
leading zeros, base prefixes are lowercase, and decimal float overflow rejects.
C2's float production accepts decimal digits, separately from the stricter
integer spelling; exponent-form literals are always floats. Finite underflow
is accepted. Signs remain operators. Integer ranges take precedence over a
fractional point. No numeric value is evaluated as a Can operation here.

Parentheses/brackets cannot cross a physical source newline, even through a
comment or multiline literal. Other context-sensitive restrictions belong to
the parser: declaration grammar, required nonempty blocks, trailing commas, physical
one-line assertions and completed method chains. Multiline literal spans let
the parser enforce those restrictions without scanning strings again.

Right-angle operators use maximal tokens. While reading generic arguments,
the parser may split `>>` or `>=` into closing angles and the remaining token. Thus
three nested type closers remain lexable; no unsigned-right-shift token exists.
The expression grammar must reject an attempted unsigned shift. Obsolete
semantics are not inferred from the predecessor lexer or old goldens.

The old brace helpers are explicitly named `Legacy*` and kept only for existing
predecessor-parser/tool callers until their I04/I41/I44 replacement gates. They
are not lexical authority for the current language and have no compatibility
aliases. New lexical fixtures live in `compiler/testdata/current/lexer`.
