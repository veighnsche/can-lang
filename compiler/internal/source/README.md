# Original source and shared coordinates

`New(name, text)` validates UTF-8 without reading a path or executing source.
The immutable string retains the original bytes, including CRLF and an optional
initial BOM. All spans are half-open byte intervals in that string.

The initial BOM is ignored for logical positions: byte 3 is the first column.
Offsets inside that ignored prefix, a UTF-8 scalar, or a CRLF pair are rejected.
Line indexing follows C2's LF/CRLF source-newline forms. A bare CR is left as
content for the lexer to admit only inside literals or diagnose elsewhere.

- `Position`: one-based line and Unicode scalar column for human diagnostics.
- `UTF16Position`: zero-based line and UTF-16 column for LSP.
- `MapPosition`: one-based line and zero-based UTF-16 column for source maps.
- `Offset`: the inverse LSP conversion, rejecting split surrogate pairs and
  positions beyond the actual line rather than silently clamping them.

The lexer, diagnostics and future editor/map consumers share this file object.
I40/I41 still own actual map encoding and editor protocol integration. These
coordinate primitives do not claim that those later gates are complete.
