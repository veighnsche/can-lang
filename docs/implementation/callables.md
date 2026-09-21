# Named callable values

A named reference resolves an eligible declaration when the reference is checked.
Ordinary functions and fetch declarations have the ordinary reference contract;
question, judge and LLM declarations require an ordinary function wrapper.
Receiver methods capture their checked receiver, and `near` inputs capture the
exact same-name binding with the exact declared type. Direct calls still supply
all inputs. The remaining callable inputs and result are exact; assignment may
widen only the declared error bound.

Lowering evaluates the target and captures once, then allocates a native async
closure. Captures remain immutable aliases. Each creation produces a distinct
function even at the same lexical site. The closure reconstructs declaration
argument order, forwards the invoking assertion context and returns the existing
protected completion carrier. Merely creating a reference does not execute its
target. Field, index and grouped callable expressions evaluate before arguments;
a failed callee or argument prevents subsequent preparation and invocation.

A private WeakMap records the creation site, target and shallow frozen capture
list against the native function. This preserves instance evidence for I18;
receipt data is neither a Can projection nor a default diagnostic. I18 still
owns preorder/occurrence/participant-path fixture queue assignment. I16 owns
native declaration grammar and provider integration, while I46 owns concrete
generic references. Native eligibility tests supply checked native symbols and
check parsed reference bodies; they do not claim provider execution.

Validation and consultations: [I08 evidence](evidence/2026-09-21/i08-validation.md).
