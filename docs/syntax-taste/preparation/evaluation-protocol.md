# Draft evaluation protocol for an AI-agent language

24 September 2026 · P2.1–P2.5 working protocol; P2.4 objective chosen by the user

Can's [confirmed identity](../decisions.md#design-direction) defines the audience:
AI coding agents. Human readability, familiarity and comfort are not design
goals. The [recommendation program](../can-recommendation-program-2026-09-24.md)
proposes a server-driven SaaS milestone and a broader Can-authored frontend
milestone. Existing decisions do not yet establish browser execution or Linux
qualification. This draft turns the agent audience into observable criteria;
it does not select language mechanisms.

## Outcomes to evaluate

| Candidate outcome | What would count as evidence |
| --- | --- |
| Correct generation | The agent can write a valid Can program from a task description that passes specified behavior and negative-case tests. Count all attempts and interventions. |
| Safe change | After a route, field, variant, error or package change, the agent preserves intended behavior; the checker diagnoses mistakes that should not silently pass. |
| Contract understanding | From public declarations and diagnostics, the agent can identify required operations, possible failures and capability/lifetime boundaries without relying on hidden compiler behavior. |
| Efficient repair | Given a failing build/test and its diagnostic, the agent finds the fault and makes a correct fix. Track failed edits, retries, tools used and total wall time. |
| Independent composition | Agent-authored libraries with overlapping local names and errors compose without source edits or privileged manual intervention. |
| Product result | Invoice edit, webhook recovery and browser grid show the specified visible and persistent behavior under failures, security boundary tests and supported deployment targets. |

The current program supplies specific adversarial cases: a misspelled final
variant arm; a renamed caller assertion selecting a different helper fixture;
two unrelated dependencies with overlapping short names/error IDs; route and
form-field renames; same-typed reordered constructor fields; cross-tenant invoice
access; duplicate delivery and uncertain commit; 503 fragment visibility; and
optimistic rollback and disposal. These are candidate workload seeds, not an
exhaustive evaluation set.

## Comparison method

1. Define each task's intended behavior, acceptance checks, source revision,
   available tools and allowed context before exposing candidate syntax.
2. Compare the strongest current-Can idiom with each proposed mechanism on the
   same semantic task. Give agents equivalent instructions, context and test
   access. Changes to the compiler/tooling baseline are recorded explicitly.
3. Include creation, rename/refactor, failure diagnosis and repair. Keep
   held-out cases so an example used to invent syntax is not the only test of it.
4. Measure pass/fail on semantic checks first. Record compile attempts, repaired
   mistakes, unobserved contract changes, required prompts/context, tool calls,
   wall time and final program size. Repeated trials are needed for model
   nondeterminism; preserve the exact model and effort settings.
5. For security, resource and platform claims, use executable negative cases
   and external observations. Agent preference or Jev advice cannot substitute
   for them.
6. Decide acceptance thresholds before running comparative trials. The user's
   frozen agent-first identity and secondary token objective govern the tradeoff;
   record measured results, uncertainty and the engineering decision.

The [official OpenAI evaluation guidance](https://developers.openai.com/api/docs/guides/evaluation-best-practices)
supports task-specific, representative tests with defined success criteria and
metrics. This protocol additionally includes source-language semantics and Can's
known failure cases.

## Token efficiency: adopted secondary objective

Source length alone is an incomplete proxy. A shorter Can program might require
longer instructions, more compiler diagnostics, more retries or more reasoning to
repair. The relevant unit is **the complete successful agent task**: initial
context, code produced, tools/results read, retries and final validation.

Track separately:

- Exact input tokens for the actual model request, including message/tool
  structure, whenever the [token counting API](https://developers.openai.com/api/docs/guides/token-counting)
  applies. Plain-text local tokenization can support code comparisons but does
  not count the whole request reliably.
- Reported output tokens, including reasoning and non-visible formatting tokens.
  Visible source tokens alone understate this count; see the [official token
  accounting guidance](https://developers.openai.com/api/docs/guides/token-counting).
- Cached and uncached input, cache writes and subagent retries when measuring
  cost for a workflow; see [agent usage guidance](https://developers.openai.com/api/docs/guides/agents-api/observability).
- Successful-task rate and wall time. A smaller source file is valuable only in
  the context of valid contracts and correct observed behavior.

The user chose token efficiency as a **secondary measured criterion** across
successful AI coding tasks, including prompts, code, diagnostics and retries.
Correctness, agent edit reliability and explicit contracts remain the primary
criteria. The exact model-specific accounting and weighting in comparisons are
still protocol questions; no fixed token budget or source-length target was selected.

## Scope and decisions still needed

The target for this preparation is the [recommendation program's](../can-recommendation-program-2026-09-24.md)
broad Can-authored frontend and backend recommendation. A qualified
server-driven SaaS target is an intermediate milestone. The user asked for a
complete implementation task list for that program and confirmed Can's
agent-first identity; no later user message narrowed the application target.
Browser execution remains a proposed design change requiring evidence.

## Representative workloads and acceptance rule

Use these task families, drawing concrete fixtures from the current examples
and evidence. Each family needs creation, controlled refactor and diagnostic
repair cases. Preserve a held-out variant with different names/data so a model
cannot pass solely by copying an example it was shown.

| Family | Required demonstrations |
| --- | --- |
| Closed contract changes | Typo in final variant arm; added variant leaf; variant specialization/bridge policy; same-typed field reorder; renamed capture. |
| Independent libraries | Two dependencies with overlapping package/error names; validated `email` and quantity; a generic helper used by callers with different finite errors. |
| Reusable tests | Caller label rename with cross-package helper scenario; lexical queues and argument checks preserved. |
| Server product | Tenant invoice edit with cross-tenant rejection, form field rename, retained invalid input and visible 503; signed webhook replay, outbox retry and uncertain commit. |
| Native execution | Owner escape and cleanup failures; race/abort/shutdown under a named Linux deployment. |
| Can-authored client | Editable invoice grid with local draft, keyboard use, optimistic save, rollback, slow/offline behavior and disposal, including wire/route renames. |
| Native AI and later items | Representative native judgment, wrapper/batch/callable, catalogue SDK, bulk-collection and authoring examples when their disposition is evaluated. |

Use the same task definition and tools across candidates. For the first
comparative language trials, use `gpt-6-sol` at medium effort as the primary
coding agent and `gpt-6-luna` at medium effort as the efficient agent. Use
`gpt-6-astra` at high effort on a case where both fail or where a candidate's
alleged advantage may instead be a capacity limit. Run five independent
attempts per candidate/model/task variant, with fresh workspace state and no
access to another attempt's transcript. A preliminary pilot may debug the
harness but cannot be counted in the five attempts. Record the model identifier
returned by the host or API, date, effort, prompt, tools, source revision,
compiler version and test fixtures for every attempt; aliases alone do not
establish an immutable model snapshot.

Before each comparison, freeze one creation case, one controlled rename/add-case
refactor and one diagnostic-repair case from the relevant workload family above.
Keep at least one held-out variant with changed names and data. The exact prompt,
files, hidden checks and candidate-specific syntax examples go in a dated
case registration before either alternative is run. Do not tune a candidate
prompt after seeing its failures without restarting the comparison as a new
revision. The current-idiom baseline gets the same task and test access.

The common success criterion is all positive, negative, refactor and observable
product checks passing with no hidden semantic widening or unauthorized effect.
Report the five-attempt success counts and uncertainty rather than claiming a
small difference proves superiority. A candidate cannot advance if it weakens
a required contract. A new language mechanism needs a reproduced baseline
failure or a material agent benefit that persists on held-out cases; if five
attempts do not distinguish it, retain the simpler current design and record
the unresolved hypothesis. Among candidates with equivalent contract and
success outcomes, compare median total tokens to successful completion and
wall time as secondary measures. Report failed-attempt counts separately so
conditional token medians cannot hide a low success rate. No arbitrary token
budget determines acceptance.

**Acceptance rule:** a proposed design must pass the stated compiler/runtime
contract cases and preserve the product outcome on the relevant scenario.
Compare agent completion, repair and secondary token cost against the strongest
current idiom. For a new language mechanism, require a reproducible current-idiom
failure or a material measured benefit that remains after compiler/tooling/library
alternatives are considered. No arbitrary line-count target or token budget
substitutes for this rule. Record uncertainty when agent trials do not clearly
separate candidates, and retain the current design until stronger evidence is
available.

This protocol settles what can be compared while source audits continue.
Topic-specific thresholds, exact task fixtures and model settings are recorded
before their respective trials, with the rationale and the user's established
identity and token preference.
