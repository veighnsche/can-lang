package ir

import (
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// Question separates ordered descriptor preparation from deferred handlers.
// Prepared static spreads retain their original arm values across transport.
type Question struct {
	Identity, Source, Connection, Kind string
	Span                               source.Span
	Inputs                             []Local
	Result                             *types.Type
	Record                             bool
	Instructions, Minimum              *Expression
	Options                            []QuestionOption
	Metadata                           []QuestionMetadata
	Fallback, Shared                   *Region
}
type QuestionMetadata struct {
	Name, Kind string
	Local      Local
}
type QuestionOption struct {
	Name                string
	Description, Spread *Expression
	Handler             *Region
	Dynamic             bool
}

// Judge is an explicit phase plan. Registration preparations cannot see answer
// bindings. Only Continuation receives those bindings after every handler has
// succeeded. No liveness optimization may remove a registration.
type Judge struct {
	Identity, Source, Connection string
	Span                         source.Span
	Inputs                       []Local
	State                        types.CodecSchema
	StateInputs                  []Local
	Registrations                []JudgeRegistration
	Continuation                 *Region
	// Native lists the raw intrinsic obligations N by exact error identity;
	// Emitted lists the declared authored obligations E, including AI
	// validation. The same identity may appear in both sets.
	Native, Emitted []string
}

type JudgeRegistration struct {
	Question  string
	Span      source.Span
	Prepare   []Preparation
	Arguments []*Expression
	Binding   *Local
}
