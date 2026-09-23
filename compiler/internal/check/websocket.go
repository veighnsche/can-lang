package check

import (
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

const websocketSessionHandle = "can.std.ws@1::session"

// isWebSocketScopeRequest reports whether the type is the opaque websocket
// session handle. No Can expression constructs a session, so assertion
// rows omit session inputs while the harness splices its scope token; any
// session operation the row does not when-supply fails at the denied live
// boundary, exactly like stream handles.
func isWebSocketScopeRequest(typ *types.Type) bool {
	return typ != nil && typ.Kind() == types.Opaque && typ.Declaration() == websocketSessionHandle
}
