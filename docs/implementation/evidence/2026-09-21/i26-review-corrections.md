# I26 independent-review corrections

Three independently reported defects were reproduced on the post-I27 compiler
before repair. These corrections retain the intended I26 contract.

## Descriptor expression type declarations

Native descriptor expressions were omitted from generated TypeScript type-root
collection. Valid nested-array mapping in a fetch query could build and execute
but failed strict TypeScript with TS2304 for a derived array alias. The staged
fetch fixture now exercises nested-array mapping in path, query, header and text
body expressions. Before repair it produced missing-alias errors in all four
locations. After repair it passes strict TypeScript, eight real loopback requests,
and four ordinary assertion roots.

The existing recursive expression walker now accepts explicit descriptor roots.
Fetch path/query/header/body, question descriptors and judge registration
preparations/arguments use that same walker, including nested invocation and
callable contracts. No parallel type inference or permissive fallback is added.

## Static header literals

Header validation previously inspected only a top-level string literal. Arrays,
groups and literal spreads could hide source-visible CR/LF, NUL or values outside
native ByteString range. The checker now recursively inspects those literal
containers. Five focused mutations cover scalar, array, grouped array, literal
spread and grouped spread values. Dynamic calls and bindings still use runtime
transport validation; the maintained positive fixture uses computed header data.
CR/LF and non-ByteString container mutations were accepted before repair and
rejected afterward.
The NUL regression uses Can's `\0` escape (not JSON's `\u0000` spelling).

## HEAD raw-provider fixtures

The fixture response constructor knew bodyless statuses but ignored HEAD. It now
rejects nonempty HEAD response bytes during registration and constructs accepted
HEAD responses with a null native body. Both registration and consumption pass
the request method into the shared constructor. Catching the registration failure
cannot clear the sticky `malformed fixture` violation or earn provider evidence.

The new runtime regression failed before repair because registration succeeded.
After repair, uppercase/lowercase HEAD rows reject impossible payloads, and a valid
empty HEAD fixture passes through the named text adapter with raw-provider
provenance. Existing bodyless-status and malformed-fixture tests still pass.

Validation completed:

- Focused compiler/header checks and staged fetch integration passed.
- Twelve named-fetch/AI transport tests passed (179 expectations).
- Runtime provider and regression test passed strict TypeScript.
- Full runtime suite passed: 186 tests, 19,559 expectations, zero failures.
- Full compiler and integration gate passed with pinned archive and CAN_TSC:
  `go test ./compiler/... ./tests/integration -count=1` (integration: 85.678 s).
  This includes staged named fetch, mixed questions, offline fixture execution,
  runtime qualification and the existing release gates.
