package ir

import (
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// Coordination retains preparation separately from launched invocations. Spread
// entries remain dynamic until their immutable callable arrays are snapshotted.
type Coordination struct {
	Site          string
	Span          source.Span
	Mode          string // all, settled, any, race: the selected native Promise operation
	Result        *types.Type
	Entries       []Participant
	AggregateType *types.Type
	Aggregate     *OutcomeHandler
	Shared        *OutcomeHandler
	Errors        []*types.Type
}
type Participant struct {
	Span    source.Span
	Call    *Invocation
	Spread  *Expression
	Result  *types.Type
	Errors  []*types.Type
	Handler *OutcomeHandler
}
type OutcomeHandler struct {
	Region *Region
	Arms   []Arm
}
