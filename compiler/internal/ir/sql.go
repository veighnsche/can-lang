package ir

import (
	"github.com/veighnsche/can-lang/compiler/internal/sql"
)

// SQLDescriptor is a fully checked manifest SQL descriptor: the immutable
// statement analysis plus the concrete parameter and row record identities
// it was checked against. Static descriptor names resolve to these values
// at compile time; they never become runtime lookup strings.
type SQLDescriptor struct {
	Owner      string
	ParamType  string
	RowType    string
	Parameters []string
	Checked    sql.Descriptor
}

// SQLCallSite is one checked query/execute call site: the owning project
// key plus the static manifest descriptor name. The emitter splices the
// checked descriptor value for this site; the name never becomes a
// runtime lookup string.
type SQLCallSite struct {
	Owner string
	Name  string
}
