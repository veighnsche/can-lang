package ir

import "github.com/veighnsche/can-lang/compiler/internal/types"

// ArrayOperation is a checked invocation of the closed native array catalogue.
// Unlike a Native expression, its adapter returns an awaited Completion.
type ArrayOperation struct {
	Name       string
	None, Some *types.Type
}
