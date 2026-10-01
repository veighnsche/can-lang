package check

import (
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// isDescriptorScopeRequest admits the descriptor launch handle to
// assertion elision. No Can expression constructs a launch, so assertion
// rows omit that input while the harness splices its scope token; any
// descriptor operation the row does not when-supply fails at the denied
// live boundary. The test owner admission stays with I04; this disjunct
// covers only the F1-owned launch handle.
func isDescriptorScopeRequest(typ *types.Type) bool {
	if typ == nil || typ.Kind() != types.Opaque {
		return false
	}
	return typ.Declaration() == "can.std.descriptor@1::launch"
}
