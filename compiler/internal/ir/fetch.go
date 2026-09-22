package ir

import (
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// Fetch preserves descriptor evaluation order independently of native launch.
// Codec schemas and envelope identity are checked concrete compiler evidence.
// Native lists the raw intrinsic obligations N by exact error identity;
// Emitted lists the declared authored obligations E. The same identity may
// appear in both sets with different provenance.
type Fetch struct {
	Identity, Source, Connection, Method, BodyMode, ResultMode, Envelope string
	Span                                                                 source.Span
	Inputs                                                               []Local
	Result                                                               *types.Type
	Path, Body                                                           *Expression
	Query, Headers                                                       []FetchEntry
	BodySchema, ResultSchema                                             *types.CodecSchema
	Native, Emitted                                                      []string
}
type FetchEntry struct {
	Name  string
	Value *Expression
}
