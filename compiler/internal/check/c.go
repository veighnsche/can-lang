package check

import (
	"fmt"

	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
)

const (
	cParseModule = "can.std.c@1::parse_module"
	cCheckModule = "can.std.c@1::check_module"
)

// cEntryKinds mirrors the K04 EntryKind union
// (tools/runtime/test-services/native-values/c-ingress.ts) exactly. The
// seam adapter re-validates at runtime and fails unknown kinds closed
// as a problem string; the static mirror exists so a misspelled kind
// fails at check time with a precise span.
var cEntryKinds = map[string]bool{
	"executable":     true,
	"assertion_root": true,
	"assertion_case": true,
}

func cStaticOperation(identity string) bool {
	return identity == cCheckModule
}

func checkCEntryKind(kind string) error {
	if cEntryKinds[kind] {
		return nil
	}
	return fmt.Errorf("c entry kind %q is not admitted", kind)
}

// checkCCall enforces the static half of C-ingress admission: the entry
// kind argument of check_module must be a string literal exactly equal
// to a K04 entry kind (query-key pattern following checkNativeCall, not
// the browser flow-through: the kind selects the manifest entry
// statically, and the K04 checker treats any other value as an
// assertion case). parse_module takes no closed vocabulary. The owner
// input elides in rows through the existing test-owner disjunct; c:: has
// no opaque handles of its own, so no new scope disjunct is needed.
func (c *regionChecker) checkCCall(identity string, args []syntax.Argument, span source.Span) error {
	fixed := func(count int) ([]syntax.Argument, error) {
		if len(args) != count {
			return nil, c.locate(span, fmt.Errorf("c call requires %d fixed arguments", count))
		}
		for _, arg := range args {
			if arg.Spread || arg.Group != nil {
				return nil, c.locate(span, fmt.Errorf("c call requires fixed ordinary arguments"))
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
	case cCheckModule:
		fixedArgs, err := fixed(5)
		if err != nil {
			return err
		}
		kind := literal(fixedArgs[2])
		if kind == nil || kind.Token.Kind != syntax.String {
			return c.locate(span, fmt.Errorf("c check kind must be a static literal"))
		}
		if err := checkCEntryKind(kind.Token.Value); err != nil {
			return c.locate(kind.Token.Span, err)
		}
		return nil
	default:
		return c.locate(span, fmt.Errorf("unknown c static operation %q", identity))
	}
}
