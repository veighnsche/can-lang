package check

import (
	"fmt"

	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

const (
	nativeMake    = "can.std.native@1::make"
	nativeObserve = "can.std.native@1::observe"
)

// nativeMakeKinds mirrors NATIVE_MAKE_KINDS in
// tools/runtime/test-services/native-values/session.ts exactly. The service
// re-validates at runtime; the static mirror exists so a misspelled kind
// fails at check time with a precise span.
var nativeMakeKinds = map[string]bool{
	"primitive_bits":     true,
	"text":               true,
	"ordered_entries":    true,
	"hostile_descriptor": true,
}

// nativeObserveKinds mirrors NATIVE_OBSERVE_KINDS in
// tools/runtime/test-services/native-values/observe.ts exactly (7 members;
// the schema.ts catalogue note claiming bytes/events is recorded drift).
var nativeObserveKinds = map[string]bool{
	"scalar_tag": true,
	"ieee_bits":  true,
	"lexeme":     true,
	"descriptor": true,
	"entries":    true,
	"identity":   true,
	"counters":   true,
}

func nativeStaticOperation(identity string) bool {
	return identity == nativeMake || identity == nativeObserve
}

func checkNativeMakeKind(kind string) error {
	if nativeMakeKinds[kind] {
		return nil
	}
	return fmt.Errorf("native make kind %q is not admitted", kind)
}

func checkNativeObserveKind(kind string) error {
	if nativeObserveKinds[kind] {
		return nil
	}
	return fmt.Errorf("native observe kind %q is not admitted", kind)
}

// checkNativeCall enforces the static half of native-values admission: the
// two top-level kind arguments must be string literals exactly equal to a
// service table member (query-key pattern, not the browser flow-through:
// closedness is the contract here). Nested closed fields (deadline clock,
// hostile descriptor, tag/coherence) stay service-enforced at runtime; the
// checker has no record-literal inspection precedent and the service
// validates them before any effect.
// isNativeScopeRequest admits the five native opaque handles to assertion
// elision. No Can expression constructs a session, value handle, pending
// action, gate or fault, so assertion rows omit those inputs while the
// harness splices its scope token; any native operation the row does not
// when-supply fails at the denied live boundary. The test owner admission
// stays with I04; this disjunct covers only the N-owned handles.
func isNativeScopeRequest(typ *types.Type) bool {
	if typ == nil || typ.Kind() != types.Opaque {
		return false
	}
	switch typ.Declaration() {
	case "can.std.native@1::session",
		"can.std.native@1::value_handle",
		"can.std.native@1::pending_action",
		"can.std.native@1::gate",
		"can.std.native@1::fault":
		return true
	}
	return false
}

func (c *regionChecker) checkNativeCall(identity string, args []syntax.Argument, span source.Span) error {
	fixed := func(count int) ([]syntax.Argument, error) {
		if len(args) != count {
			return nil, c.locate(span, fmt.Errorf("native call requires %d fixed arguments", count))
		}
		for _, arg := range args {
			if arg.Spread || arg.Group != nil {
				return nil, c.locate(span, fmt.Errorf("native call requires fixed ordinary arguments"))
			}
		}
		return args, nil
	}
	literal := func(arg syntax.Argument) *syntax.LiteralExpr {
		expression, ok := fetchUngroup(arg.Value).(*syntax.LiteralExpr)
		if !ok {
			return nil
		}
		return expression
	}
	switch identity {
	case nativeMake:
		fixedArgs, err := fixed(3)
		if err != nil {
			return err
		}
		kind := literal(fixedArgs[1])
		if kind == nil || kind.Token.Kind != syntax.String {
			return c.locate(span, fmt.Errorf("native make kind must be a static literal"))
		}
		if err := checkNativeMakeKind(kind.Token.Value); err != nil {
			return c.locate(kind.Token.Span, err)
		}
		return nil
	case nativeObserve:
		fixedArgs, err := fixed(4)
		if err != nil {
			return err
		}
		kind := literal(fixedArgs[2])
		if kind == nil || kind.Token.Kind != syntax.String {
			return c.locate(span, fmt.Errorf("native observe kind must be a static literal"))
		}
		if err := checkNativeObserveKind(kind.Token.Value); err != nil {
			return c.locate(kind.Token.Span, err)
		}
		return nil
	}
	return nil
}
