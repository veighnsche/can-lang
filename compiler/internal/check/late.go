package check

import (
	"fmt"

	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
)

const (
	lateSelect          = "can.std.late@1::select"
	lateEnroll          = "can.std.late@1::enroll"
	lateArmGate         = "can.std.late@1::arm_gate"
	lateEmit            = "can.std.late@1::emit"
	lateWitnessTerminal = "can.std.late@1::witness_terminal"
	lateObserve         = "can.std.late@1::observe"
	lateGrantLease      = "can.std.late@1::grant_lease"
	lateObserveLease    = "can.std.late@1::observe_lease"
	lateReleaseLease    = "can.std.late@1::release_lease"
	lateReconcile       = "can.std.late@1::reconcile"
	lateKillWorker      = "can.std.late@1::kill_worker"
	lateReadOutcome     = "can.std.late@1::read_outcome"
	lateReadCounters    = "can.std.late@1::read_counters"
)

// lateVocabularies mirrors the K06 closed vocabularies
// (tools/runtime/test-services/native-values/late.ts) exactly. The
// late adapter re-validates at runtime and fails unknown words as
// late_fault; the static mirror exists so a misspelled word fails at
// check time with a precise span.
var lateVocabularies = map[string]map[string]bool{
	"identity": {
		"late.alpha": true,
		"late.beta":  true,
	},
	"participant": {
		"worker":   true,
		"observer": true,
	},
	"kind": {
		"fault":    true,
		"use":      true,
		"terminal": true,
	},
	"terminal": {
		"completed": true,
		"rejected":  true,
		"failed":    true,
	},
}

// lateStaticArity pins the fixed argument count (owner included) of
// every late operation carrying a closed vocabulary.
var lateStaticArity = map[string]int{
	lateSelect:          2,
	lateEnroll:          2,
	lateArmGate:         2,
	lateEmit:            4,
	lateWitnessTerminal: 3,
	lateObserve:         4,
	lateGrantLease:      2,
	lateObserveLease:    3,
	lateReleaseLease:    3,
	lateReconcile:       2,
}

// lateStaticWords pins, per static operation, the argument positions
// (0-based, owner included) that must be string literals from the
// named K06 vocabulary. Sequence numbers, lease ids, and the pure
// reads (kill_worker, read_outcome, read_counters) carry no closed
// vocabulary and stay dynamic.
var lateStaticWords = map[string]map[int]string{
	lateSelect:          {1: "identity"},
	lateEnroll:          {1: "participant"},
	lateArmGate:         {1: "participant"},
	lateEmit:            {1: "participant", 2: "kind"},
	lateWitnessTerminal: {1: "participant", 2: "terminal"},
	lateObserve:         {1: "participant", 2: "participant"},
	lateGrantLease:      {1: "participant"},
	lateObserveLease:    {1: "participant"},
	lateReleaseLease:    {1: "participant"},
	lateReconcile:       {1: "participant"},
}

func lateStaticOperation(identity string) bool {
	_, ok := lateStaticArity[identity]
	return ok
}

// checkLateCall enforces the static half of late-occurrence
// admission: every closed-vocabulary argument must be a string
// literal exactly equal to a K06 word (query-key pattern following
// checkCCall). The owner input elides in rows through the existing
// test-owner disjunct; late:: has no opaque handles of its own, so
// no new scope disjunct is needed.
func (c *regionChecker) checkLateCall(identity string, args []syntax.Argument, span source.Span) error {
	arity, ok := lateStaticArity[identity]
	if !ok {
		return c.locate(span, fmt.Errorf("unknown late static operation %q", identity))
	}
	if len(args) != arity {
		return c.locate(span, fmt.Errorf("late call requires %d fixed arguments", arity))
	}
	for _, arg := range args {
		if arg.Spread || arg.Group != nil {
			return c.locate(span, fmt.Errorf("late call requires fixed ordinary arguments"))
		}
	}
	words, ok := lateStaticWords[identity]
	if !ok {
		return c.locate(span, fmt.Errorf("unknown late static operation %q", identity))
	}
	for index, vocab := range words {
		expression, ok := fetchUngroup(args[index].Value).(*syntax.LiteralExpr)
		if !ok || expression.Token.Kind != syntax.String {
			return c.locate(span, fmt.Errorf("late %s must be a static literal", vocab))
		}
		if !lateVocabularies[vocab][expression.Token.Value] {
			return c.locate(expression.Token.Span, fmt.Errorf("late %s %q is not admitted", vocab, expression.Token.Value))
		}
	}
	return nil
}
