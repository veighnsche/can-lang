package check

import (
	"fmt"

	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
)

const (
	dbRecordSettlement = "can.std.db@1::record_settlement"
)

// dbVocabularies mirrors the K24 closed vocabularies
// (tools/runtime/test-services/db-observer/transactions.ts)
// exactly. The db-transaction adapter re-validates at runtime
// and fails unknown words as db_fault; the static mirror exists
// so a misspelled word fails at check time with a precise span.
// The NT-I13 db operations carry no closed-vocabulary scalar
// positions (all names are dynamic; schema tags travel inside
// arrays), so record_settlement is the only static db operation.
var dbVocabularies = map[string]map[string]bool{
	"outcome": {
		"committed":   true,
		"rolled-back": true,
		"unknown":     true,
	},
	"engine": {
		"postgres": true,
		"sqlite":   true,
		"mysql":    true,
	},
}

// dbStaticArity pins the fixed argument count (owner included) of
// every db operation carrying a closed vocabulary.
var dbStaticArity = map[string]int{
	dbRecordSettlement: 5,
}

// dbStaticWords pins, per static operation, the argument positions
// (0-based, owner included) that must be string literals from the
// named K24 vocabulary. Actors, tokens, compiles and ids stay
// dynamic.
var dbStaticWords = map[string]map[int]string{
	dbRecordSettlement: {3: "outcome", 4: "engine"},
}

func dbStaticOperation(identity string) bool {
	_, ok := dbStaticArity[identity]
	return ok
}

// checkDbCall enforces the static half of transaction-settlement
// admission: outcome and engine must be string literals exactly
// equal to K24 words (query-key pattern following checkLateCall).
// The owner input elides in rows through the existing test-owner
// disjunct; db:: has no opaque handles of its own, so no new scope
// disjunct is needed.
func (c *regionChecker) checkDbCall(identity string, args []syntax.Argument, span source.Span) error {
	arity, ok := dbStaticArity[identity]
	if !ok {
		return c.locate(span, fmt.Errorf("unknown db static operation %q", identity))
	}
	if len(args) != arity {
		return c.locate(span, fmt.Errorf("db call requires %d fixed arguments", arity))
	}
	for _, arg := range args {
		if arg.Spread || arg.Group != nil {
			return c.locate(span, fmt.Errorf("db call requires fixed ordinary arguments"))
		}
	}
	words, ok := dbStaticWords[identity]
	if !ok {
		return c.locate(span, fmt.Errorf("unknown db static operation %q", identity))
	}
	for index, vocab := range words {
		expression, ok := fetchUngroup(args[index].Value).(*syntax.LiteralExpr)
		if !ok || expression.Token.Kind != syntax.String {
			return c.locate(span, fmt.Errorf("db %s must be a static literal", vocab))
		}
		if !dbVocabularies[vocab][expression.Token.Value] {
			return c.locate(expression.Token.Span, fmt.Errorf("db %s %q is not admitted", vocab, expression.Token.Value))
		}
	}
	return nil
}
