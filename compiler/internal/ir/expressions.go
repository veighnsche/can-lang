// Package ir contains checked current-language operations, separate from source
// spelling and from native TypeScript formatting.
package ir

import (
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

type ExpressionKind string

const (
	ArmValue           ExpressionKind = "arm"
	CoordinationValue  ExpressionKind = "coordination"
	CallableValue      ExpressionKind = "callable"
	Literal            ExpressionKind = "literal"
	Binding            ExpressionKind = "binding"
	Unary              ExpressionKind = "unary"
	Binary             ExpressionKind = "binary"
	Comparison         ExpressionKind = "comparison"
	Index              ExpressionKind = "index"
	Slice              ExpressionKind = "slice"
	Length             ExpressionKind = "length"
	Field              ExpressionKind = "field"
	StandardProjection ExpressionKind = "standard_projection"
	Array              ExpressionKind = "array"
	Call               ExpressionKind = "call"
	InvocationValue    ExpressionKind = "invocation"
	MatchValue         ExpressionKind = "match_value"
	Record             ExpressionKind = "record"
	Update             ExpressionKind = "update"
)

type EqualityMode string

const (
	Strict        EqualityMode = "strict"
	FloatIdentity EqualityMode = "float_identity"
	Deep          EqualityMode = "deep"
)

type Expression struct {
	Coordination *Coordination
	Callable     *Callable
	Invocation   *Invocation
	Match        *Match
	Kind         ExpressionKind
	Span         source.Span
	Type         *types.Type
	// Text is decoded string data, exact numeric spelling, an operator, a field
	// name, or a resolved binding identity according to Kind; never emitted code.
	Text      string
	Inputs    []*Expression
	Operators []string
	Equality  []EqualityMode
	// Slice has fixed [receiver,start,end] slots; omitted bounds are nil.
	Spread []bool
	Fields []string
}
