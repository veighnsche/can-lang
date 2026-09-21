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
}
