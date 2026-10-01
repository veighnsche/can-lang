package check

import (
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// isHttpPeerScopeRequest admits the test owner and the four http_peer
// opaque handles to assertion elision. No Can expression constructs an
// owner, listener, connection, dial or request, so assertion rows omit
// those inputs while the harness splices its scope token; any http_peer
// operation the row does not when-supply fails at the denied live
// boundary. The owner admission lands here (not P28) because I04 is the
// first task whose Can helpers take owner-first operations.
func isHttpPeerScopeRequest(typ *types.Type) bool {
	if typ == nil || typ.Kind() != types.Opaque {
		return false
	}
	switch typ.Declaration() {
	case "can.std.test@1::owner",
		"can.std.http_peer@1::listener",
		"can.std.http_peer@1::connection",
		"can.std.http_peer@1::dial",
		"can.std.http_peer@1::request":
		return true
	}
	return false
}
