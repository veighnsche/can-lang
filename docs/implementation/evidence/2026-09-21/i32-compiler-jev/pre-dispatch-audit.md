# I32 compiler-mount consultations: pre-dispatch audit

All three requests ask the same two questions over the same facts,
constraints, and alternatives. Exact identifiers are preserved verbatim in
every request: `callable http::server_response (http::request) emits []`,
`http::route_get`, `http::route_post`, `http::make_router`,
`http::invalid_route`, `duplicate_route`, `ambiguous_route`,
`createRouter.get(path, callback)`,
`MountedCallback = (request: unknown, context?: AssertionContext) => Promise<Completion<unknown>>`,
`$canOwnCallable(site, target, captures, async (residual..., $canContext?) => Promise<Completion<Result>>, retained, context)`,
`ir/resources.go`, `ir/http.go`, `staticInputs`.

Facts held constant: P10 handler contract; catalogue signatures and error
sets; runtime draft mount/dispatch behavior (boxed completion, one fresh
native Response per dispatch); named-declaration-only mounting; first
staticInputs with no existing literal rule; existing callable lowering and
capture tracking; single-pipeline rule with the conditional ir/http.go
allowance.

Wording differences (intentionally rewritten, semantically equivalent):
- Request 1 frames the mount question around "representation" and the path
  question as "enforced across the Go checker and the Bun runtime".
- Request 2 frames mounting as "mounting design" and paths as "where validity
  is decided"; it paraphrases the wrapper as "produced $canOwnCallable
  wrapper straight to the router".
- Request 3 frames mounting as "registration binds" and paths as "how the
  static nature is guaranteed"; it paraphrases the wrapper as "resulting
  $canOwnCallable closure directly where a MountedCallback is expected".

No request adds, drops, or narrows an option: all three offer
ordinary_callable / dedicated_ir / synthetic_wrapper for mounting and
check_literal_runtime_recheck / runtime_only for paths, with matching
definitions. Option order is identical, so no order-bias check is needed
beyond noting it.
