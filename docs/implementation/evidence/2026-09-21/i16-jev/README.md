# I16 generated spread shape consultation

The final three requests compare annotation-only shape projection, a provisional
checked graph followed by rebuilding, and splitting the expression checker into
pre-seal and post-seal phases. Facts, constraints and alternatives are equivalent;
context, instructions and option descriptions were rewritten for each request.
All final requests and raw responses are retained in this directory.

All three choose projection, with confidence 0.97, 0.98 and 0.8. The third assigns
0.07 probability to refactoring and 0.06 to rebuilding; there is no categorical
disagreement. The chosen implementation discovers names from declared shapes and
still requires the normal sealed-type checker to validate actual expressions,
arm contracts and error bounds. It never treats discovered shape as proof of
expression validity and never runs authored expressions or providers.

Tests exercise interleaved inline/spread fields, metadata order, duplicate names,
and incompatible arm contracts. Cyclic or missing shape evidence produces a
compile diagnostic rather than an incomplete or guessed record. Agreement is
advice, not proof or a guarantee of bias removal.

The initial attempt returned HTTP 529; the failed request and observed status are
preserved under failed-attempt. An intermediate round reused an explanatory
suffix and failed the wording audit. That entire round is preserved under
wording-audit and was superseded by the fully rewritten final three requests.
