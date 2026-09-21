package ir

import (
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// Noul retains runtime descriptor computations separately from response
// handlers. Options remain in authored order for once-only preparation; their
// Boolean labels select handlers independently of that order.
type Noul struct {
	Identity, Source, Connection string
	Span                         source.Span
	Inputs                       []Local
	Instructions, Minimum        *Expression
	Options                      []NoulOption
}

type NoulOption struct {
	True        bool
	Description *Expression
	Handler     *Region
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
}

type JudgeRegistration struct {
	Question  string
	Span      source.Span
	Prepare   []Preparation
	Arguments []*Expression
	Binding   *Local
}
