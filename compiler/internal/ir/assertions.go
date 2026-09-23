package ir

// AssertionRoot is the immutable P3 identity, independent of source position or
// the order in which a runner chooses roots.
type AssertionRoot struct {
	Package     string `json:"package"`
	Declaration string `json:"declaration"`
	Name        string `json:"name"`
}
type Assertion struct {
	Root     AssertionRoot
	Actual   *Region
	Expected *Region
	Raw      *RawFixture
}

type FixtureTable struct {
	Identity string
	Rows     []FixtureRow
}
type FixtureRow struct {
	Selector  string
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
