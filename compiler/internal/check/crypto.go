package check

import (
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

const cryptoKeyHandle = "can.std.crypto@1::key"

// isCryptoKeyScopeRequest reports whether the type is the opaque crypto
// key handle. No Can expression constructs a key, so assertion rows omit
// key inputs while the harness splices its scope token; any key
// operation the row does not when-supply fails at the denied live
// boundary, exactly like pool handles.
func isCryptoKeyScopeRequest(typ *types.Type) bool {
	return typ != nil && typ.Kind() == types.Opaque && typ.Declaration() == cryptoKeyHandle
}
