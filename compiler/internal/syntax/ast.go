package syntax

import "github.com/veighnsche/can-lang/compiler/internal/source"

// QualifiedName retains spelling and location without resolving a declaration.
// Resolution belongs to the project checker, never to the parser.
type QualifiedName struct {
	Span    source.Span
	Package string
	Name    string
}

type TypeNode interface {
	TypeSpan() source.Span
	typeNode()
}

type NamedType struct {
	Span      source.Span
	Name      QualifiedName
	Arguments []TypeNode
}

type ArrayType struct {
	Span    source.Span
	Element TypeNode
}

type CallableType struct {
	Span   source.Span
	Result TypeNode
	Inputs []TypeNode
	Errors ErrorBound
}

type ChoiceArmType struct {
	Span   source.Span
	Result TypeNode
	Errors ErrorBound
}

// ErrorBound records even an explicitly empty emits clause.
type ErrorBound struct {
	Span  source.Span
	Types []TypeNode
}

func (*NamedType) typeNode()                   {}
func (*ArrayType) typeNode()                   {}
func (*CallableType) typeNode()                {}
func (*ChoiceArmType) typeNode()               {}
func (n *NamedType) TypeSpan() source.Span     { return n.Span }
func (n *ArrayType) TypeSpan() source.Span     { return n.Span }
func (n *CallableType) TypeSpan() source.Span  { return n.Span }
func (n *ChoiceArmType) TypeSpan() source.Span { return n.Span }

type Expr interface {
	ExprSpan() source.Span
	exprNode()
}

type ExpressionLocation struct{ Span source.Span }

func (n ExpressionLocation) ExprSpan() source.Span { return n.Span }
func (ExpressionLocation) exprNode()               {}

type LiteralExpr struct {
	ExpressionLocation
	Token Token
}

// ScopeExpr is synthesized by the assertion checker, never parsed: it stands
// for one harness-supplied ingress value at an elided root argument position.
type ScopeExpr struct {
	ExpressionLocation
}
type NameExpr struct {
	ExpressionLocation
	Name QualifiedName
}
type GroupExpr struct {
	ExpressionLocation
	Value Expr
}
type UnaryExpr struct {
	ExpressionLocation
	Operator string
	Operand  Expr
}
type BinaryExpr struct {
	ExpressionLocation
	Operator    string
	Left, Right Expr
}

// ComparisonExpr preserves the chain instead of falsely nesting boolean results.
type ComparisonExpr struct {
	ExpressionLocation
	Operands  []Expr
	Operators []string
}
type ArrayExpr struct {
	ExpressionLocation
	Elements []Argument
}
type FieldExpr struct {
	ExpressionLocation
	Receiver Expr
	Field    Token
}
type IndexExpr struct {
	ExpressionLocation
	Receiver, Index Expr
}
type SliceExpr struct {
	ExpressionLocation
	Receiver, Start, End Expr
}
type ConstructorExpr struct {
	ExpressionLocation
	Name      QualifiedName
	Types     []TypeNode
	Arguments []Argument
}
type CallExpr struct {
	ExpressionLocation
	Invocation Invocation
	Methods    []MethodInvocation
}
type ReferenceExpr struct {
	ExpressionLocation
	Callee Expr
	Types  []TypeNode
	// Bindings holds explicit Q3 near-input pins: `with param = expr`
	// pairs in listed order. Empty keeps pure name-based capture.
	Bindings []WithBinding
}

// WithBinding pins one near-input by callee parameter name. The name is a
// rename-reference to the callee parameter; the value checks against the
// declared input type in the creation scope.
type WithBinding struct {
	Span  source.Span
	Name  Token
	Value Expr
}
type UpdateExpr struct {
	ExpressionLocation
	Receiver Expr
	Fields   []Replacement
}
type Argument struct {
	Span   source.Span
	Spread bool
	Value  Expr
	// Group retains a syntactic final state group until declaration resolution.
	// It is not an expression or an anonymous tuple value.
	Group *ArgumentGroup
}
type ArgumentGroup struct {
	Span   source.Span
	Values []Expr
}
type Replacement struct {
	Span  source.Span
	Name  Token
	Value Expr
}
type Invocation struct {
	Span      source.Span
	Callee    Expr
	Types     []TypeNode
	Arguments []Argument
}
type MethodInvocation struct {
	Span      source.Span
	Name      Token
	Types     []TypeNode
	Arguments []Argument
}

type File struct {
	Source       *source.File
	Header       PackageHeader
	Declarations []Declaration
	Comments     []Comment
}
type PackageHeader struct {
	Span     source.Span
	Name     Token
	Provides []Token
	Uses     []Import
}
type Import struct {
	Span source.Span
	// Dependency names a direct dependency edge of the owning project for
	// cross-project imports (`dep::pkg`). It is nil for same-project and
	// catalogue imports, which stay unqualified.
	Dependency *Token
	Package    Token
	Alias      *Token
}
type Declaration interface {
	DeclSpan() source.Span
	declaration()
}
type DeclarationLocation struct{ Span source.Span }

func (d DeclarationLocation) DeclSpan() source.Span { return d.Span }
func (DeclarationLocation) declaration()            {}

type RecordDecl struct {
	DeclarationLocation
	Name Token
	// Owner marks an `owner record`: construction, with-updates and field
	// representation are confined to the declaring package. Exported owner
	// records project opaquely: foreign packages may pass, compare and
	// whole-leaf-match values but never observe fields.
	Owner      bool
	Parameters []Token
	Fields     []Field
}
type VariantDecl struct {
	DeclarationLocation
	Name         Token
	Parameters   []Token
	Alternatives []TypeNode
}
type ErrorDecl struct {
	DeclarationLocation
	Name       Token
	Parameters []Token
	Fields     []Field
}
type FunctionDecl struct {
	DeclarationLocation
	Result     TypeNode
	Name       Token
	Parameters []Token
	Receiver   *Field
	Errors     ErrorBound
	Inputs     []Input
	Assertions []Assertion
	Body       Block
}
type ValueDecl struct {
	DeclarationLocation
	Binding Binding
}
type Field struct {
	Span source.Span
	Type TypeNode
	Name Token
}
type Input struct {
	Field
	Near     bool
	Variadic bool
}
type Assertion struct {
	Span      source.Span
	Name      Token
	Receiver  Expr
	Arguments []Argument
	Expected  Body
	Mode      *AssertionMode
	// Use expands a fixture template in place; it carries no arguments,
	// expected completion or execution mode of its own.
	Use *AssertionUse
	// Scenario tags a when row with a package-local scenario marker.
	// The row activates only through a caller link, never by root name.
	Scenario *Token
	// Links wires an assertion root to exported scenarios. Only roots
	// carry links; when rows and template expansions must not.
	Links []QualifiedName
}

// AssertionUse is `use template(arguments)` at a lexical when row: the
// template cases expand in place under the row selector.
type AssertionUse struct {
	Span      source.Span
	Template  QualifiedName
	Arguments []Argument
}

// AssertionMode is the optional indented execution-mode line under an
// assertion row. Raw carries the source-relative fixture path token;
// Failure carries an attached wrapper policy injection.
type AssertionMode struct {
	Span    source.Span
	Raw     Token
	Failure *AssertionFailure
}

// AssertionFailure is `using failure native|emitted error_value`: a
// policy-only test that skips the wrapped operation and injects a fresh
// origin-tagged failure before the policy lookup.
type AssertionFailure struct {
	Span   source.Span
	Origin Token
	Value  Expr
}
type Binding struct {
	Span  source.Span
	Type  TypeNode
	Name  Token
	Value Expr
}
type Block struct {
	Span     source.Span
	Steps    []Step
	Terminal Body
}
type Step interface {
	StepSpan() source.Span
	step()
}
type BindingStep struct{ Binding }

func (s *BindingStep) StepSpan() source.Span { return s.Span }
func (*BindingStep) step()                   {}

type CallStep struct {
	Span source.Span
	Call *CallExpr
}

func (s *CallStep) StepSpan() source.Span { return s.Span }
func (*CallStep) step()                   {}

type Body interface {
	BodySpan() source.Span
	body()
}
type BodyLocation struct{ Span source.Span }

func (b BodyLocation) BodySpan() source.Span { return b.Span }
func (BodyLocation) body()                   {}

type ValueBody struct {
	BodyLocation
	Value Expr
}
type SuccessBody struct {
	BodyLocation
	Value Expr
}
type FailureBody struct {
	BodyLocation
	Error *ConstructorExpr
}
type RelayBody struct {
	BodyLocation
	Call *CallExpr
}

// InheritBody invokes the immediately preceding policy rule for the same
// wrapper key and original failure. It is terminal-only and valid only
// lexically inside a wrapper handler; checking enforces the scope.
type InheritBody struct {
	BodyLocation
}
type DoBody struct {
	BodyLocation
	Block Block
}
type MatchBody struct {
	BodyLocation
	Match Match
}
type MatchExpr struct {
	ExpressionLocation
	Match Match
}
type MatchKind string

const (
	ValueMatch MatchKind = "value"
	CallMatch  MatchKind = "call"
	ChainMatch MatchKind = "chain"
)

type Match struct {
	Span   source.Span
	Kind   MatchKind
	Values []Expr
	Call   *CallExpr
	Chain  []ChainEntry
	When   []Assertion
	Arms   []MatchArm
}
type ChainEntry struct {
	Span    source.Span
	Call    *CallExpr
	Binding *Field
}
type MatchArm struct {
	Span     source.Span
	Patterns []PatternNode
	Outcome  *OutcomePattern
	Body     Body
	Forward  bool
}
type OutcomePattern struct {
	Span            source.Span
	Success         bool
	StandardFailure bool
	Error           TypeNode
	Binding         *Field
	// Alias is the explicit untyped error-value name from `head as alias`.
	// Only completion error heads carry it; it always requires `=>`.
	Alias *Token
}
type PatternNode interface {
	PatternSpan() source.Span
	patternNode()
}
type PatternLocation struct{ Span source.Span }

func (p PatternLocation) PatternSpan() source.Span { return p.Span }
func (PatternLocation) patternNode()               {}

type WildcardPattern struct{ PatternLocation }
type LiteralPattern struct {
	PatternLocation
	Literal  Token
	Negative bool
}
type NamePattern struct {
	PatternLocation
	Name  QualifiedName
	Types []TypeNode
}
type ConstructorPattern struct {
	PatternLocation
	Name   QualifiedName
	Types  []TypeNode
	Fields []PatternNode
}
type ArrayPattern struct {
	PatternLocation
	Elements []PatternNode
	Rest     *Token
}
type RangePattern struct {
	PatternLocation
	Lower, Upper *LiteralPattern
}
type AlternativePattern struct {
	PatternLocation
	Alternatives []PatternNode
}

// BindPattern captures the matched value under an explicit name. Bare
// identifiers in patterns are nominal leaf tests, never captures.
type BindPattern struct {
	PatternLocation
	Name Token
}

type Coordination struct {
	Span         source.Span
	Mode         string
	WithError    bool
	Participants []Participant
	Arms         []MatchArm
}
type Participant struct {
	Span   source.Span
	Call   *CallExpr
	Spread Expr
	Arms   []MatchArm
}
type CoordinationExpr struct {
	ExpressionLocation
	Coordination Coordination
}
type CoordinationStep struct{ Coordination Coordination }

func (s *CoordinationStep) StepSpan() source.Span { return s.Coordination.Span }
func (*CoordinationStep) step()                   {}
