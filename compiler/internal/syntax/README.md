# Current C2 lexical boundary

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
I04: declaration grammar, required nonempty blocks, trailing commas, physical
one-line assertions and completed method chains. Multiline literal spans let
the parser enforce those restrictions without scanning strings again.

Right-angle operators use maximal tokens. While reading generic arguments,
I04 may split `>>` or `>=` into closing angles and the remaining token. Thus
three nested type closers remain lexable; no unsigned-right-shift token exists.
The expression grammar must reject an attempted unsigned shift. Obsolete
semantics are not inferred from the predecessor lexer or old goldens.

The old brace helpers are explicitly named `Legacy*` and kept only for existing
predecessor-parser/tool callers until their I04/I41/I44 replacement gates. They
are not lexical authority for the current language and have no compatibility
aliases. New lexical fixtures live in `compiler/testdata/current/lexer`.
