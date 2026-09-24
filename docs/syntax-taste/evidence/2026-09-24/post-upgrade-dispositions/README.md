# Next-upgrade disposition evidence

24 September 2026. The [scope ledger](../../../post-upgrade-dispositions-2026-09-24.md)
assigns one primary Fix, Retain or Defer disposition to each of the 39 IDs in
the [factual reconciliation](../../../post-upgrade-reconciliation-2026-09-24.md).
The [core](core-proposal.md), [platform](platform-proposal.md) and
[native/support](native-proposal.md) proposals were prepared in parallel,
then independently audited against the consolidated ledger. The audits found
no missing or repeated IDs. Their corrections to U03/U04, P02/P03/P05/P06 and
O03 were incorporated before validation.

The [Jev findings](jev/findings.md) link three exact independent requests,
responses, model/usage metadata, probabilities and a wording audit. All
explanatory prose was rewritten for each consultation while facts, constraints
and alternatives were preserved. The browser and generic results were
investigated against accepted contracts and the user's stated priority;
classifier agreement was not treated as proof.

Validation of this documentation change:

- All 39 reconciliation IDs occur exactly once in the disposition tables;
  there are no missing or additional IDs.
- 187 local Markdown link targets across the new ledger, reconciliation,
  docs index and disposition evidence files resolve.
- `git diff --check` passed.
- `GOCACHE=/tmp/can-disposition-go-cache go test ./tests/baseline/ -count=1`
  passed in 21.163 seconds.

No compiler, authored runtime, example or test source was changed for this
disposition. The baseline suite checks the existing implementation; it does
not demonstrate that any selected Fix is implemented. No new live HTTP hedge,
Linux host run, full browser matrix, external provider/database workflow or
agent comparison was performed.
