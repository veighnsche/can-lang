# Replacement review method

Baseline: `8bbcf13b33c74d26bc7bb0f9b72fba0d5415d70b`, 2026-09-19.
The user requested Workstream A of the master TODO. Scope is research,
self-contained context, thirteen replacement judgments, and an evidence-based
audit disposition. No language redesign, acceptance-baseline promotion, compiler
repair, or ABI migration is authorized by a probability.

## Research and independent checking

Three independent investigations covered language core, authority/evidence, and
host/async boundaries. The coordinating auditor inspected their source excerpts,
reproductions and disputed statements. A second pass corrected claims about
callable invocation positions, recursion schemas, extern purity admission,
acceptance authority, exact source examples and materially different B05 host
contracts before submission. A newly observed named-argument evaluation-order
disagreement received a separate executable probe.

The dossier is a current factual reconciliation, not a rewritten historical
requirements document. Each specialist document labels implemented, proposed and
historical facts, and names which implementation supersedes contradictory prose.
Older specifications are retained. Their text is not silently promoted into
current fact, nor is observed implementation automatically a sound specification.

Artifacts:

- `shared-model.md`: the complete shared model supplied in every request.
- `language-core.md`, `authority-evidence.md`, `host-async.md`: research with
  actual source, compiler rules, customers, counterexamples and alternatives.
- `questions.json`: eight architecture and five surface judgments, with current
  examples, benefits, costs, obligations and proposed examples for each option.
- `requests/*.json`: **the complete serialized request bodies actually supplied**.
- `responses/*.body.json`: original returned HTTP body bytes.
- `responses/*.json`: timestamps, hashes, model, parsed responses, distributions,
  confidence and usage. No credential or authorization header is stored.
- `packet-manifest.json`: source hashes, exact included sections, request hashes,
  byte preflight and model limits.
- `baseline-checks.json`, `evidence-*.txt`, `logs/commands.json`, `logs/*.txt`:
  ordinary gate results and current compiler/target reproductions.

## Provider facts and limits

The TypeSafe skill was read and the live documentation was fetched directly by
HTTPS after the browser retrieval tool could not access it. Retrieval timestamps,
URLs, sizes and hashes are in `provider-docs.json`. The pinned model is
`jev-1.13.0`; the documented budgets are 64k tokens for the request and 32k for
state plus the longest question. These limits were checked before submission.
[Models](https://docs.typesafe.ai/models.md)

The HTTP endpoint accepts a shared state and typed questions. We use Choice with
complete option descriptions and preserve every returned option probability.
JEV is used as a System One classifier, not as a researching or generative LLM.
It returns no prose design rationale here; all explanations in the final audit
disposition belong to the coordinating auditor, not to JEV.
The transport uses the fixed HTTPS endpoint, refuses redirects, and has no retry
or truncation setting. Authentication is read from `TYPESAFE_API_KEY` only at
submission. [API](https://docs.typesafe.ai/api.md)

Confidence describes distribution concentration; it does not certify factual
correctness or design approval. No empirical threshold calibrated on Can
architecture choices exists. We therefore use **no automatic acceptance
threshold**, and review the result against code and tests.
[Confidence](https://docs.typesafe.ai/confidence.md)

The provider describes weaknesses in indirection and long irrelevant context.
This is a bounded preference judgment over researched alternatives, not a request
for JEV to perform a language audit, reason through an unseen repository, or
prove soundness. Relevant whole sections are selected for each request, instead
of sending all historical documents. [Model limitations](https://docs.typesafe.ai/model-jaggedness/jev-1.13.md)

## Context packing and question design

Each of the thirteen packets contains the same complete shared Can model plus
whole relevant dossier sections and one decision. One question per packet allows
different evidence selection; there are no identical-state questions needlessly
split across calls. No request depends on another's response or on cross-request
memory. Source paths are provenance, never instructions to fetch missing context.

The prepare step rejects serialized requests larger than **30,000 UTF-8 bytes**,
including JSON overhead. This deliberately pessimistic one-byte-per-token
engineering proxy leaves margin under the documented token limits, but is **not
an exact provider-tokenizer calculation**. The response's actual reported input
usage is saved and checked below 32,000; no API truncation option is supplied.
When the initial B05 packet exceeded the local byte budget, complete generic
boundary/trust sections already covered by the shared model were deliberately
excluded. Both actual B05 proposals and their disagreements remained in full.
No text was silently cut to fit.

Questions retain the thirteen requested subject areas but replace straw choices
with serious alternatives. For example modules includes strict-global repair,
and sequencing includes repair-and-specify without committing to a new core.
B05 includes customer-contract-first deferral without assuming a runtime winner.
Every actual request also includes an explicit `defer` alternative. All benefits,
costs, obligations and examples are visible to the reviewer. Alternative syntax
is explicitly illustrative; it is not presented as currently compilable Can.

Old request/response files are unchanged and their probabilities were not sent
to the replacement reviewer. Because both evidence and candidate sets changed,
new probabilities are not statistically comparable with the withdrawn run. One
run per question is recorded; there was no resampling to obtain a desired choice.
Option-order sensitivity and empirical calibration were not measured.

## Reproduction

Prepare and validate packets without network access:

```sh
python3 docs/audit-probes/workstream-a/review.py
```

The script fails if a recorded response's request would change. To run a new
review, preserve this version and create a new dated/versioned artifact set;
recheck live provider limits and the new source baseline first. For a fresh
artifact set, `--submit <question-id> ...` is the explicit network operation.
There is no silent overwrite or retry of a recorded response. Request/response
hashes and schema checks are reproducibility checks, not proof of model quality.

Audit dispositions in `README.md` are the coordinating auditor's judgments.
An accepted recommendation means a supported direction for later design work;
it is not user ratification of syntax, semantics, proof rules or ABI. Unresolved
questions remain unresolved even if JEV assigns a concentrated distribution.
