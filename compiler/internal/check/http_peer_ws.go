package check

import (
	"fmt"

	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
)

const (
	wsSend         = "can.std.http_peer@1::ws_send"
	wsDeliverEvent = "can.std.http_peer@1::ws_deliver_event"
)

// wsVocabularies mirrors the K21 closed opcode vocabulary
// (tools/runtime/test-services/http-peer/websocket.ts) exactly.
// The ws adapter re-validates at runtime and fails unknown
// words as peer_fault; the static mirror exists so a
// misspelled opcode fails at check time with a precise span.
// Destinations, connections, payloads, close codes and
// reasons stay dynamic.
var wsVocabularies = map[string]map[string]bool{
	"opcode": {
		"text":   true,
		"binary": true,
		"ping":   true,
		"pong":   true,
	},
}

var wsStaticArity = map[string]int{
	wsSend:         4,
	wsDeliverEvent: 4,
}

var wsStaticWords = map[string]map[int]string{
	wsSend:         {2: "opcode"},
	wsDeliverEvent: {2: "opcode"},
}

func wsStaticOperation(identity string) bool {
	_, ok := wsStaticArity[identity]
	return ok
}

// checkWsCall enforces the static half of websocket admission:
// the opcode must be a string literal exactly equal to a K21
// word (query-key pattern following checkDbCall).
func (c *regionChecker) checkWsCall(identity string, args []syntax.Argument, span source.Span) error {
	arity, ok := wsStaticArity[identity]
	if !ok {
		return c.locate(span, fmt.Errorf("unknown ws static operation %q", identity))
	}
	if len(args) != arity {
		return c.locate(span, fmt.Errorf("ws call requires %d fixed arguments", arity))
	}
	for _, arg := range args {
		if arg.Spread || arg.Group != nil {
			return c.locate(span, fmt.Errorf("ws call requires fixed ordinary arguments"))
		}
	}
	words, ok := wsStaticWords[identity]
	if !ok {
		return c.locate(span, fmt.Errorf("unknown ws static operation %q", identity))
	}
	for index, vocab := range words {
		expression, ok := fetchUngroup(args[index].Value).(*syntax.LiteralExpr)
		if !ok || expression.Token.Kind != syntax.String {
			return c.locate(span, fmt.Errorf("ws %s must be a static literal", vocab))
		}
		if !wsVocabularies[vocab][expression.Token.Value] {
			return c.locate(expression.Token.Span, fmt.Errorf("ws %s %q is not admitted", vocab, expression.Token.Value))
		}
	}
	return nil
}
