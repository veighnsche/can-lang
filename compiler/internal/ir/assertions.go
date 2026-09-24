package ir

// AssertionRoot is the immutable P3 identity, independent of source position or
// the order in which a runner chooses roots.
type AssertionRoot struct {
	Package     string `json:"package"`
	Declaration string `json:"declaration"`
	Name        string `json:"name"`
	// Links carries the canonical scenario identities this root
	// linked. It is omitted when the root links nothing.
	Links []string `json:"links,omitempty"`
}
type Assertion struct {
	Root     AssertionRoot
	Actual   *Region
	Expected *Region
	Raw      *RawFixture
	Injected *PolicyInjection
}

// PolicyInjection is an attached wrapper `using failure` row: the actual
// invocation skips the wrapped operation and starts policy lookup from a
// fresh origin-tagged failure built from the checked value. Reported as
// policy-fixture evidence, never request/decoder evidence.
type PolicyInjection struct {
	Operation string
	Origin    string
	Identity  string
	Failed    string
	Value     *Expression
}

type FixtureTable struct {
	Identity string
	Rows     []FixtureRow
}
type FixtureRow struct {
	Selector string
	// Owner is the canonical package identity of the lexical when
	// table. Plain rows activate only for same-owner roots; this is
	// the DI-06 same-owner lexical selection rule.
	Owner string
	// Scenario is the canonical scenario identity for tagged rows
	// and empty for plain rows. Tagged rows activate only through
	// a caller link, never by root name.
	Scenario  string
	Prepare   []Preparation
	Arguments []*Expression
	Expected  *Completion
	Raw       *RawFixture
}

// RawFixture is a validated P4.1 native exchange fixture. The operation is
// the producing declaration identity; a nil exchange forbids transport.
type RawFixture struct {
	Operation   string
	Environment map[string]string
	Exchange    *RawExchange
}
type RawExchange struct {
	Method      string
	URL         string
	Headers     [][2]string
	BodyBase64  string
	BodyJSON    string
	HasBodyJSON bool
	Response    *RawResponse
	Failure     *RawFailure
}
type RawResponse struct {
	Status  int
	Headers [][2]string
	Body    string
}
type RawFailure struct {
	Kind  string
	Phase string
}
