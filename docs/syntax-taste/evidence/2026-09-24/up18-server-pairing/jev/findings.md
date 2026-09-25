# Jev advice on UP18 server/browser pairing

26 September 2026. Three new `jev-latest` requests were prepared before any
response was sent. Each state paragraph, instruction, and option description
was rewritten in all three versions while preserving facts, constraints, and
alternatives. The [wording audit](wording-audit.json) records the 18 compared
explanatory fields and full-request hashes. Requests [1](request-1.json),
[2](request-2.json), and [3](request-3.json), their exact
[responses](response-1.json) ([2](response-2.json), [3](response-3.json)),
transport metadata, and [probability summary](summary.json) are saved. Each
call returned HTTP 200 and model `jev-1.13.0`; total reported usage was 5,174
input and 391 output tokens. No request contained another response. Jev did
not inspect source or run code; the supplied state summarized the ownership
boundaries, the browser/server generation layouts, the pairing duty, the
page/retention/race constraints, and the candidate mechanisms.

The table shows selected options and probabilities, not measured design
correctness. Exact option meanings are in each request.

| Decision | Request 1 | Request 2 | Request 3 | Engineering selection |
| --- | --- | --- | --- | --- |
| Page binding | inject .96 | inject .84 (defer .15) | inject .93 | **Inject document.** The HTTP server splices exactly one `<script type="module" src="<report-selected digest URL>">` before `</body>` of text/html documents for paired builds; fragments, non-HTML, HEAD, and unpaired builds keep exact bytes. |
| Retention | ledger .98 | ledger .99 | ledger .99 | **Durable ledger.** Replaced bytes persist in a dist-level content-addressed store with per-route replacement times in an atomically replaced ledger; serve-time expiry keeps routes alive within seven days and builds reap expired rows under the project lock. |
| Verify scope | reaudit .52 (conf .04) | hashes-only .77 | hashes-only .66 | **Full reaudit.** See disagreement analysis below. |

Page binding and retention agree unanimously with the proposed mechanisms and
are taken as advice supporting them, not as proof. The serve-bytes-only
option drew 0.15 in request 2 only; it was included as the do-less
alternative and stays rejected because the UP18 done-clause explicitly
requires pages to receive the report-selected URL.

## Verify-scope disagreement

Jev splits 2:1 toward hashes-and-lock-only, but request 1 is a coin flip
(confidence 0.04, reaudit 0.52 vs hashes-only 0.48), so the signal is weak
and wording-sensitive. Investigated against the task's own acceptance
criteria, hashes-and-lock-only cannot satisfy UP18: the done-clause requires
missing-map, missing-table, and secret-canary cases to fail before
publication. A hand-tampered manifest can stay self-consistent (recomputed
build ID matches) while dropping a map entry or shipping canary bytes with
correct file hashes; only reference checks (entry/table presence, every
script paired with its map, trailer binding) plus content scanning catch
those cases. The full reaudit reuses the existing post-bundle `AuditBundle`
rather than duplicating audit logic, and the pre-selection re-read closes
the verify-then-publish race the task names. The engineering selection
therefore keeps the full reaudit despite Jev's lean.
