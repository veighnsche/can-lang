package ir

import "github.com/veighnsche/can-lang/compiler/internal/types"

// Callable fixes a named target and its complete declaration contract. Expression
// Inputs contain captures in evaluation order; Positions place them back into
// declaration-order arguments. The remaining slots are the closure parameters.
type Callable struct {
	Site      string
	Target    string
	Contract  *types.Type
	Positions []int
}
