# Native declaration frontend

I16 adds current AST nodes for connections, fetch, LLM, judge, Noul, ordinary and
generated-record Choice/Score, and named Choice arms. Formatting preserves native
section structure. The parser requires explicit emits and connection identity,
keeps state grouping distinct from ordinary expression grouping, enforces section
closure, and admits contextual probability only in its owning handler region.
The `choice_arm<T>` stored-value type remains distinct from an arm declaration.

Resolution assigns separate kinds to connections, questions, judges, LLMs,
fetches and arms. Only questions may be registered; direct question calls and
references are rejected. Fetch has an ordinary callable reference contract;
judge and LLM calls require final state groups and cannot become ordinary callable
references. Both names of a generated record/question pair are registered before
signature checks, with ordinary collision and visibility rules. Exporting the
question cannot hide its generated result record.

Connection settings pass through the shared I15 closed policy. Compilation
accepts literal strings and checked integer constant expressions, never running
calls or reading authentication. Native declarations retain full authored error
bounds. Required intrinsic errors depend on declaration mode and authentication;
judge bounds also include all declared registered-question errors. Extra errors
remain in the public bound. Profile checks compare resolved connection identities,
not endpoint strings or equivalent settings.

Grouped calls separate the final state group before applying ordinary variadic
arity. Empty state still requires `()`, and a single field retains its parentheses.
Prepared arguments remain in written order. Judge registration arguments all use
the same input/state scope; answer bindings are added only after every registration
has been checked, for the continuation. Binding types must exactly match question
results, and void registrations cannot bind a result.

Native handlers use the existing checked completion-region machinery, preserving
return types, error propagation and lexical scope. Probability has a private
region input rather than a global binding. Dynamic Choice validates description
array types and selected-key scope. Score/Choice metadata is unavailable while
preparing request descriptors. Named arms have no free caller captures; their
constant descriptions and bodies are checked, and named-arm evidence is supplied
to immutable top-level initialization.

Generated record fields follow written inline/spread order, then metadata order.
A pre-seal projection reads declared data shapes from names, constructors, fields,
indexes, updates and declared call results. It does not execute expressions or
prove their validity. The ordinary sealed checker still checks each expression,
arm result and error contract. Missing or cyclic shape evidence and duplicate
expanded fields produce diagnostics instead of guessed fields.

This frontend does not claim provider execution. Judge runtime lowering is I17;
complete fetch modes, question adapters and LLM protocols are I26–I28. The staged
frontend fixture declares native forms but only builds them, with network denied
and credentials absent. Executable provider fixtures belong to those later tasks.
