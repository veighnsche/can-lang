# Jev advice on the next-upgrade scope

24 September 2026. Three fresh `jev-latest` requests were prepared before any
response was sent. Every context field, instruction and option description has
three independently written versions; technical identifiers, facts, constraints,
question IDs and alternatives stay constant. The [wording audit](wording-audit.json)
checks 48 explanatory fields and request hashes. The coordinator also compared
their meanings before dispatch. All three requests were sent independently,
without earlier answers in later state. The [exact requests](request-1.json),
[second](request-2.json), [third](request-3.json), and [responses](response-1.json)
([second](response-2.json), [third](response-3.json)) are preserved with timing
metadata. The responses identify `jev-1.13.0`. Total reported usage: 6,423
input and 1,266 output tokens. [Full probabilities and confidence](summary.json)
are preserved separately. Jev is a classifier supplied with source facts; it
did not inspect the repository or run code.

The table gives each run's selected option and its probability, not empirical
correctness or confidence. Options are fully defined in the saved requests.

| Decision | Request 1 | Request 2 | Request 3 | Engineering disposition |
| --- | --- | --- | --- | --- |
| Action binding | selected split 1.00 | selected split .89 | selected split .86 | Complete the earlier accepted handler-free declaration and request-aware mount. |
| Browser delivery | grid only .72 | grid only .81 | grid only .43 | **Disagree:** supported general delivery was already selected and the user named it priority scope; the test grid alone cannot meet that contract. Fix U04, leaving the runtime layout to design preparation. |
| Generic calls | experiment .51 | experiment .60 | experiment .52 | Include a bounded validated public-generic composition outcome because the user selected it as priority; make soundness and helper/cycle probes a preimplementation gate. The classifier's caution affects that gate, not the disposition. |
| Error wrappers | finite parameters .42 | retain .75 | retain .61 | Defer new syntax; the split and no live agent comparison leave the DI-03 comparison requirement unmet. |
| Iteration | retain .51 | retain .60 | compare .56 | Retain the current recursion contract; defer a new iteration mechanism until dynamic-work and fixture-preservation evidence. |
| Race lifetime | measure .71 | measure .99 | measure .80 | Keep owner drainage and measure real HTTP hedge timing before reconsidering response/loser ownership. |
| Browser host expansion | experiment .84 | retain .94 | retain .70 | Defer additional fields/storage and a generic adapter architecture; complete the earlier accepted minimum event behavior first. |
| UI reuse | prototype .82 | prototype .62 | retain .67 | Defer new component syntax; a later library comparison may reopen it after accepted focus/notice behavior is fixed. |
| Bulk collections | measure .92 | measure .82 | measure .98 | Defer bulk API; test representative native one-build semantics and workload cost before selection. |
| Capture binding | retain .56 | retain .53 | trial .56 | Retain `near` and context records; defer syntax pending the registered rename/shadow comparison. |
| Multiline layout | trial .64 | retain .42 | trial .62 | Retain single-line grammar this round; defer continuation until agent-edit evidence after formatter support. |

The browser-delivery disagreement is a scope error for this decision: the
classifier preferred the smallest existing tested path even though the supplied
facts said that general delivery was already an accepted requirement. The
current browser asset is TypeScript plus a manifest; a grid-specific bundler
with limited Node shims supplies executable JavaScript. Keeping that as the
final target would leave U04 open. The weaker third response (.43, with .29
for complete shims and .28 for a maintained profile) reinforces that no
runtime *mechanism* has been established by the three results. The selected
outcome is delivery through a supported path; native browser operations and
complete reachable-dependency checks constrain either implementation.

The generic-call result reflects real soundness uncertainty. The current
checker validates exported generic bodies but cannot yet forward an opaque
type parameter to another verified generic. It would be unsound to trust a
private template signature or emit a concrete specialization for an unknown
type. The disposition therefore includes a declaration-proof/cycle experiment
before coding this feature; if that proof fails, the design returns to
resolution rather than weakening DI-04. This is not a vote to retain the
present helper-extraction failure forever.

Other disagreements concern optional mechanisms. Error parameters had a
split and no agent-task benefit measurement; richer browser host support split
between experiment and a narrow retained surface; UI reuse and multiline
layout also divided. Those options remain deferred. Repeated agreement on
race measurement or bulk workload testing is advice, not proof of optimal
design or a reason to promote the later mechanism automatically.

The live API and Choice shape were checked against TypeSafe's
[documentation index](https://docs.typesafe.ai/llms.txt),
[HTTP API](https://docs.typesafe.ai/api.md) and
[Choice guidance](https://docs.typesafe.ai/primitives/choice.md) before sending.
The latter two pages required a direct HTTP read because the web reader did
not load their Markdown. No earlier review response was included in these
consultations.
