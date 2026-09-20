# I03 source representation consultation

Three fresh requests compared immutable original source plus newline-aware
lexing against normalized text plus a reverse offset map. Each supplied the
same C2 facts, original-byte and UTF-16 requirements, alternatives, tradeoffs
and lack of pre-existing implementation evidence. All explanatory state,
question and option wording differs between requests; corresponding fields
were checked programmatically for differences and reviewed for semantic
agreement. Exact technical terms were retained.

All responses selected the original-source representation. Reported confidence
was 0.97, 0.86 and 0.97; probabilities for that choice were 0.99, 0.93 and 0.99.
There was no choice disagreement. The lower concentration in request 2 does
not invalidate either design and is not a proof of bias removal. Returned
model: jev-1.13.0. Raw requests and responses are saved here. TypeSafe's current
API and Choice documentation read in this same implementation session supplied
the API contract; no application source secrets were sent.

Implementation evidence, not these opinions, supports the decision: original
spans round-trip through the shared UTF-16 conversion, LF/CRLF token streams
agree logically, astral/combining text remains correctly located, and split
UTF-8/UTF-16/CRLF coordinates are refused. I40 and I41 still implement actual
source-map encoding and editor protocol integration.
