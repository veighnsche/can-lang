package ir

import (
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

type Initializer struct {
	Identity, QualifiedName, Source string
	Span                            source.Span
	Type                            *types.Type
	Value                           *Expression
}
