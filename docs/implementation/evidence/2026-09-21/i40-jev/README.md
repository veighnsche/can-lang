# I40 mapping decision

Three fresh requests compare the same alternatives: explicit bundled upstream
encoding/tracing with checked origin fallback; Bun-only external map composition;
or catch-origin-only reports without map consumption. Every explanatory state,
question and option description was rewritten. Pairwise checks verified distinct
wording, and a semantic review preserved the same target observation, byte/UTF16
facts, privacy requirements, unimplemented status and proxy/accessor constraints.
Raw requests and responses are adjacent to this file.

All three consultations selected explicit tracing, with reported choice
probabilities 0.97, 1.00 and 1.00. This agreement is advice, not evidence of runtime
correctness. The implementation was chosen because the independent Bun probe
retained TS locations instead of following the external Can map, and then verified
by native composition tests and offline packaged publication/execution tests.

Subsequent testing also found that `.then(async …)` can omit outer callers. We
retain the callback's actual mapped location and document the limitation rather
than synthesize an unobserved stack. Direct awaits retain tested outer frames.
