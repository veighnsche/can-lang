# I12 assertion context consultations

Three fresh requests rewrote the entire context, question and option prose while
preserving the facts and alternatives. Pairwise field checks preceded submission,
and the full requests and raw responses are retained. No credentials were sent
in request content or stored here.

All three selected an explicit private context parameter, at probabilities 0.78,
0.51 and 0.57. Native AsyncLocalStorage remained a substantial alternative in the
last two responses. Its API was checked against the [Bun reference](https://bun.sh/reference/node/async_hooks/AsyncLocalStorage)
and [Node documentation](https://nodejs.org/api/async_context.html), but was not
adopted. No claim of qualification for that API is made, and the target inventory
is unchanged.

Code inspection favored the explicit parameter: generated calls already use one
checked private ABI, so it can pass the root context through named functions and
adapters without ambient state. Each context retains sticky harness violations,
and native output adapters consult it before any live operation. The decision
avoids a new async-context dependency, but requires every future callback and
adapter lowering to preserve that private argument. Tests of nested generated
calls, independent asynchronous roots, and caught boundary refusals are the
implementation evidence. Model agreement is advice, not proof.
