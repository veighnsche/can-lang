package ir

import (
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// Fetch preserves descriptor evaluation order independently of native launch.
// Codec schemas and envelope identity are checked concrete compiler evidence.
type Fetch struct {
	Identity, Source, Connection, Method, BodyMode, ResultMode, Envelope string
	Span                                                                 source.Span
	Inputs                                                               []Local
	Result                                                               *types.Type
	Path, Body                                                           *Expression
	Query, Headers                                                       []FetchEntry
	BodySchema, ResultSchema                                             *types.CodecSchema
}
type FetchEntry struct {
	Name  string
	Value *Expression
}
