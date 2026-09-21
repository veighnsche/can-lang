package ir

import (
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// LLM is a stateless request plan. State disclosure and output validation use
// ordinary codec schemas; the provider schema is a separate restricted profile.
type LLM struct {
	Identity, Source, Connection string
	Span                         source.Span
	Inputs, StateInputs          []Local
	Instructions                 *Expression
	State                        types.CodecSchema
	Result                       *types.Type
	Format                       *types.GenerationSchema
	Output                       *types.CodecSchema
}
