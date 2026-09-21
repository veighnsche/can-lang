# Checked expression operations

Expression IR contains concrete type evidence, source spans, resolved binding
identities, ordered operands, and explicit primitive/equality operations. Text
fields hold decoded literals or semantic names, never injected TypeScript.

The emitter returns a statement sequence and a resulting temporary. Logical
branches and comparison chains retain conditional sub-blocks so skipped operands
cannot be hoisted into eager preparation. Each reached operand is assigned once;
comparison middle operands are retained for the next pair. Temporary names use
compiler-only `$` identifiers and stay unique across one emitter instance.

The owning region supplies call lowering after argument preparation. It may emit
awaited protected-completion extraction without wrapping the expression in a new
promise or async IIFE. A native execution test exercises that insertion point;
this is not a substitute for I10's complete region/boxing implementation.
