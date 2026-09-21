package ir

import (
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

type RegionKind string

const (
	FunctionRegion RegionKind = "function"
	HandlerRegion  RegionKind = "handler"
)

// Region is a named function or a selected handler result. Nested terminal
// matches, chains, relays and do blocks retain ID rather than inventing returns
// to an outer function. Errors is the exact permitted escaping domain bound.
type Region struct {
	ID, Parent, Source string
	Kind               RegionKind
	Span               source.Span
	Result             *types.Type
	Errors             []*types.Type
	Escapes            []*types.Type
	Inputs             []Local
	Body               *Block
}
type Local struct {
	Identity string
	Type     *types.Type
	// ErrorAlias distinguishes a language-provided bare error pattern alias
	// from an authored binding, including reserved prelude error names.
	ErrorAlias bool
}
type Block struct {
	Span     source.Span
	Steps    []Statement
	Terminal *Completion
}
type Statement struct {
	Coordination *Coordination
	Local        *Local
	Value        *Expression
	Call         *Invocation
}
type Invocation struct {
	Span   source.Span
	Steps  []InvocationStep
	Result *types.Type
	Errors []*types.Type
}
type Preparation struct {
	Local Local
	Value *Expression
}
type InvocationStep struct {
	Callee    *Expression
	Contract  *types.Type
	Receiver  bool
	Fixtures  *FixtureTable
	Prepare   []Preparation
	Native    *Expression
	Identity  string
	Span      source.Span
	Arguments []*Expression
	Result    *types.Type
	Errors    []*types.Type
	// SuccessBinding is available only after this step succeeds. Method chains
	// use it as the next receiver; match-chain arms see it only on success.
	SuccessBinding string
}
type CompletionKind string

const (
	SuccessCompletion CompletionKind = "success"
	DomainCompletion  CompletionKind = "domain"
	RelayCompletion   CompletionKind = "relay"
	DoCompletion      CompletionKind = "do"
	MatchCompletion   CompletionKind = "match"
)

type Completion struct {
	Kind     CompletionKind
	RegionID string
	Span     source.Span
	Value    *Expression
	Call     *Invocation
	Block    *Block
	Match    *Match
}
type Match struct {
	Span   source.Span
	Values []*Expression
	Call   *Invocation // nil for ordinary data matching
	Arms   []Arm
	// ValueResult is set only for an ordinary value match, never a completion match.
	ValueResult *types.Type
}
type Arm struct {
	Span     source.Span
	Patterns []*Pattern
	Outcome  string // ok, domain, standard, or empty for ordinary patterns
	Error    *types.Type
	Binding  *Local
	Forward  bool
	Body     *Completion
	Value    *Expression
}

// Pattern contains typed structural tests and bindings, never generated code.
type Pattern struct {
	Kind     string
	Type     *types.Type
	Text     string
	Upper    string
	Fields   []string
	Children []*Pattern
	Binding  *Local
	Rest     *Local
	Narrow   *Local
}
